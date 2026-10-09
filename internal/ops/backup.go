package ops

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/crypt"
	"github.com/nodemode-ai/form.md/internal/store"
	_ "modernc.org/sqlite"
)

const LockFile = "form-md.lock"

const manifestName = "form-md-backup.json"

var ErrBusy = errors.New("the data directory is in use by a running server: stop it first")

type Manifest struct {
	Format    int    `json:"format"`
	Version   string `json:"version"`
	CreatedAt int64  `json:"created_at"`
	Encrypted bool   `json:"encrypted"`
	Live      bool   `json:"live"`
	Files     int    `json:"files"`
	Bytes     int64  `json:"bytes"`
}

var carried = []string{"secret.key", crypt.MetaFile}

var carriedDirs = []string{"files", "acme"}

func OpenKey(dataDir, keyFile string) (*crypt.Key, error) {
	return crypt.Load(dataDir, crypt.KeyPath(keyFile), store.HasData(dataDir))
}

func openDB(path string, key *crypt.Key, query url.Values) (*sql.DB, error) {
	if query == nil {
		query = url.Values{}
	}
	if key != nil {
		if err := store.Register(key); err != nil {
			return nil, err
		}
		query.Set("vfs", store.VFSName)
		query.Add("_pragma", "temp_store(MEMORY)")
	}
	query.Add("_pragma", "busy_timeout(10000)")
	db, err := sql.Open("sqlite", path+"?"+query.Encode())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func Backup(ctx context.Context, dataDir, keyFile, out string) (*Manifest, error) {
	key, err := OpenKey(dataDir, keyFile)
	if err != nil {
		return nil, err
	}
	src := filepath.Join(dataDir, store.FileName)
	if !store.HasData(dataDir) {
		return nil, fmt.Errorf("no database at %s", src)
	}
	live := RequestFlush(dataDir)
	out, err = filepath.Abs(out)
	if err != nil {
		return nil, err
	}
	snap, err := os.CreateTemp(filepath.Dir(out), ".form-md-snapshot-*.db")
	if err != nil {
		return nil, err
	}
	snapPath := snap.Name()
	snap.Close()
	os.Remove(snapPath)
	defer os.Remove(snapPath)

	db, err := openDB(src, key, nil)
	if err != nil {
		return nil, err
	}
	target := snapPath
	if key != nil {
		target = "file:" + snapPath + "?vfs=" + store.VFSName
	}
	_, err = db.ExecContext(ctx, "VACUUM INTO "+sqlString(target))
	db.Close()
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(out), ".form-md-backup-*")
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		tmp.Close()
		if !keep {
			os.Remove(tmp.Name())
		}
	}()
	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)
	m := &Manifest{Format: 1, Version: app.Version, CreatedAt: time.Now().Unix(), Encrypted: key != nil, Live: live}
	add := func(name, path string) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: st.Size(), ModTime: st.ModTime(), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		n, err := io.Copy(tw, f)
		m.Bytes += n
		return err
	}
	if err := add(store.FileName, snapPath); err != nil {
		return nil, err
	}
	for _, name := range carried {
		p := filepath.Join(dataDir, name)
		if _, err := os.Stat(p); err == nil {
			if err := add(name, p); err != nil {
				return nil, err
			}
		}
	}
	for _, dir := range carriedDirs {
		root := filepath.Join(dataDir, dir)
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
				return nil
			}
			rel, _ := filepath.Rel(dataDir, p)
			if dir == "files" {
				m.Files++
			}
			return add(filepath.ToSlash(rel), p)
		})
		if err != nil {
			return nil, err
		}
	}
	mb, _ := json.MarshalIndent(m, "", "  ")
	if err := tw.WriteHeader(&tar.Header{Name: manifestName, Mode: 0o600, Size: int64(len(mb)), ModTime: time.Now(), Typeflag: tar.TypeReg}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(mb); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	if err := tmp.Sync(); err != nil {
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp.Name(), out); err != nil {
		return nil, err
	}
	keep = true
	return m, nil
}

func Restore(ctx context.Context, dataDir, keyFile, archive string) (string, *Manifest, error) {
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return "", nil, err
	}
	if _, err := os.Stat(dataDir); err == nil {
		unlock, err := Lock(dataDir)
		if err != nil {
			return "", nil, err
		}
		defer unlock()
	}
	parent := filepath.Dir(dataDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", nil, err
	}
	stage, err := os.MkdirTemp(parent, ".form-md-restore-*")
	if err != nil {
		return "", nil, err
	}
	done := false
	defer func() {
		if !done {
			os.RemoveAll(stage)
		}
	}()
	m, err := unpack(archive, stage)
	if err != nil {
		return "", nil, err
	}
	var key *crypt.Key
	if m.Encrypted {
		path := crypt.KeyPath(keyFile)
		if path == "" {
			return "", nil, errors.New("this backup is encrypted: set FORMMD_KEY_FILE to its key")
		}
		if key, err = crypt.Load(stage, path, true); err != nil {
			return "", nil, err
		}
	}
	db, err := openDB(filepath.Join(stage, store.FileName), key, nil)
	if err != nil {
		return "", nil, err
	}
	var check string
	err = db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check)
	db.Close()
	if err != nil {
		return "", nil, fmt.Errorf("the backup database could not be read: %w", err)
	}
	if check != "ok" {
		return "", nil, fmt.Errorf("the backup database failed its integrity check: %s", check)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		os.Remove(filepath.Join(stage, store.FileName+suffix))
	}
	old := ""
	if _, err := os.Stat(dataDir); err == nil {
		old = dataDir + ".before-restore-" + time.Now().Format("20060102-150405")
		if err := os.Rename(dataDir, old); err != nil {
			return "", nil, err
		}
	}
	if err := os.Rename(stage, dataDir); err != nil {
		if old != "" {
			os.Rename(old, dataDir)
		}
		return "", nil, err
	}
	done = true
	return old, m, nil
}

func unpack(archive, dst string) (*Manifest, error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, errors.New("not a form-md backup (gzip)")
	}
	tr := tar.NewReader(gz)
	var m *Manifest
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.FromSlash(h.Name)
		if !filepath.IsLocal(name) {
			return nil, fmt.Errorf("unsafe path in backup: %q", h.Name)
		}
		if h.Name == manifestName {
			var mm Manifest
			if err := json.NewDecoder(io.LimitReader(tr, 1<<20)).Decode(&mm); err != nil {
				return nil, err
			}
			m = &mm
			continue
		}
		p := filepath.Join(dst, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return nil, err
		}
		out, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(out, tr)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, err
		}
	}
	if m == nil || m.Format != 1 {
		return nil, errors.New("not a form-md backup (no manifest)")
	}
	if _, err := os.Stat(filepath.Join(dst, store.FileName)); err != nil {
		return nil, errors.New("the backup has no database")
	}
	return m, nil
}
