package ops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/crypt"
	"github.com/nodemode-ai/form.md/internal/store"
)

type EncryptReport struct {
	Files   int
	Skipped int
}

func Encrypt(ctx context.Context, dataDir, keyFile string) (*EncryptReport, error) {
	path := crypt.KeyPath(keyFile)
	if path == "" {
		return nil, errors.New("set FORMMD_KEY_FILE to the key (form-md keygen makes one, keep it outside the data directory)")
	}
	key, err := crypt.ReadKey(path)
	if err != nil {
		return nil, err
	}
	if crypt.Encrypted(dataDir) {
		if _, err := crypt.Load(dataDir, path, true); err != nil {
			return nil, err
		}
		unlock, err := Lock(dataDir)
		if err != nil {
			return nil, err
		}
		defer unlock()
		rep, err := sealAll(key, dataDir)
		if err == nil {
			err = matchOwner(dataDir)
		}
		return rep, err
	}
	if !store.HasData(dataDir) {
		return nil, errors.New("there is no database to encrypt; start the server with FORMMD_KEY_FILE set and it is encrypted from the first write")
	}
	unlock, err := Lock(dataDir)
	if err != nil {
		return nil, err
	}
	defer unlock()

	dbPath := filepath.Join(dataDir, store.FileName)
	encPath := dbPath + ".encrypting"
	os.Remove(encPath)
	plain, err := openDB(dbPath, nil, nil)
	if err != nil {
		return nil, err
	}
	if _, err := plain.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		plain.Close()
		return nil, err
	}
	if err := store.Register(key); err != nil {
		plain.Close()
		return nil, err
	}
	_, err = plain.ExecContext(ctx, "VACUUM INTO "+sqlString("file:"+encPath+"?vfs="+store.VFSName))
	plain.Close()
	if err != nil {
		os.Remove(encPath)
		return nil, fmt.Errorf("encrypting the database: %w", err)
	}
	enc, err := openDB(encPath, key, nil)
	if err != nil {
		return nil, err
	}
	var check string
	err = enc.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check)
	enc.Close()
	if err != nil || check != "ok" {
		os.Remove(encPath)
		return nil, fmt.Errorf("the encrypted copy failed its check: %v %s", err, check)
	}

	rep, err := sealAll(key, dataDir)
	if err != nil {
		return rep, err
	}
	if err := os.Rename(encPath, dbPath); err != nil {
		return rep, err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		os.Remove(dbPath + suffix)
	}
	if err := key.WriteMeta(dataDir); err != nil {
		return rep, err
	}
	return rep, matchOwner(dataDir)
}

func sealAll(key *crypt.Key, dataDir string) (*EncryptReport, error) {
	rep := &EncryptReport{}
	for _, dir := range carriedDirs {
		err := filepath.WalkDir(filepath.Join(dataDir, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
				return nil
			}
			sealed, err := sealFile(key, p)
			if sealed {
				rep.Files++
			} else {
				rep.Skipped++
			}
			return err
		})
		if err != nil {
			return rep, err
		}
	}
	if _, err := sealFile(key, filepath.Join(dataDir, "secret.key")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return rep, err
	}
	return rep, nil
}

func sealFile(key *crypt.Key, path string) (bool, error) {
	in, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer in.Close()
	if crypt.IsSealed(in) {
		return false, nil
	}
	st, err := in.Stat()
	if err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".seal-*")
	if err != nil {
		return false, err
	}
	w := key.Writer(tmp)
	_, err = io.Copy(w, in)
	if err == nil {
		err = w.Close()
	}
	if err == nil {
		err = tmp.Sync()
	}
	tmp.Close()
	if err == nil {
		err = os.Chmod(tmp.Name(), st.Mode().Perm())
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return false, err
	}
	return true, nil
}

func ResetOwner(ctx context.Context, dataDir, keyFile string) (string, string, error) {
	key, err := OpenKey(dataDir, keyFile)
	if err != nil {
		return "", "", err
	}
	db, err := openDB(filepath.Join(dataDir, store.FileName), key, nil)
	if err != nil {
		return "", "", err
	}
	defer matchOwner(dataDir)
	defer db.Close()
	var id, email string
	if err := db.QueryRowContext(ctx, "SELECT id, email FROM users WHERE role = 'owner'").Scan(&id, &email); err != nil {
		return "", "", fmt.Errorf("no owner account found: %w", err)
	}
	password := strings.ToLower(crypt.Generate()[:24])
	hash, err := auth.HashPassword(ctx, password)
	if err != nil {
		return "", "", err
	}
	now := store.NowMS()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = ?, disabled = 0, totp_enabled = 0, totp_secret = '', totp_pending = '', updated_at = ? WHERE id = ?`, hash, now, id); err != nil {
		return "", "", err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", id); err != nil {
		return "", "", err
	}
	return email, password, tx.Commit()
}
