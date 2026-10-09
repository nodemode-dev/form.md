package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/crypt"
	"github.com/nodemode-ai/form.md/internal/cryptvfs"
	_ "modernc.org/sqlite"
)

const FileName = "form-md.db"

type Store struct {
	W    *sql.DB
	R    *sql.DB
	Path string
	Key  *crypt.Key
	vfs  string
}

const VFSName = "formmd-crypt"

var registered sync.Once

func Register(key *crypt.Key) error {
	var err error
	registered.Do(func() {
		err = cryptvfs.Register(VFSName, cryptvfs.Keys{DB: key.Sub("db", 64), WAL: key.Sub("wal", 64), Journal: key.Sub("journal", 64)})
	})
	return err
}

func HasData(dataDir string) bool {
	st, err := os.Stat(filepath.Join(dataDir, FileName))
	return err == nil && st.Size() > 0
}

func Open(ctx context.Context, dataDir string, key *crypt.Key) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("data directory: %w", err)
	}
	path := filepath.Join(dataDir, FileName)
	s := &Store{Path: path, Key: key}
	if key != nil {
		if err := Register(key); err != nil {
			return nil, err
		}
		s.vfs = VFSName
	}

	w, err := sql.Open("sqlite", s.dsn(true))
	if err != nil {
		return nil, err
	}
	w.SetMaxOpenConns(1)
	w.SetMaxIdleConns(1)
	w.SetConnMaxLifetime(0)
	if err := w.PingContext(ctx); err != nil {
		w.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	s.W = w
	if err := s.migrate(ctx); err != nil {
		w.Close()
		return nil, err
	}

	r, err := sql.Open("sqlite", s.dsn(false))
	if err != nil {
		w.Close()
		return nil, err
	}
	n := min(max(runtime.NumCPU(), 2), 8)
	r.SetMaxOpenConns(n)
	r.SetMaxIdleConns(n)
	if err := r.PingContext(ctx); err != nil {
		w.Close()
		r.Close()
		return nil, err
	}
	s.R = r
	return s, nil
}

func (s *Store) dsn(writer bool) string {
	q := url.Values{}
	if s.vfs != "" {
		q.Set("vfs", s.vfs)
		q.Add("_pragma", "temp_store(MEMORY)")
	}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "synchronous(NORMAL)")
	if writer {
		q.Add("_pragma", "journal_mode(WAL)")
		q.Set("_txlock", "immediate")
	} else {
		q.Add("_pragma", "query_only(1)")
	}
	return s.Path + "?" + q.Encode()
}

func (s *Store) VFS() string {
	return s.vfs
}

func (s *Store) Ping(ctx context.Context) error {
	var one int
	return s.R.QueryRowContext(ctx, "SELECT 1").Scan(&one)
}

func (s *Store) Close() error {
	var errs []error
	if s.R != nil {
		errs = append(errs, s.R.Close())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.W.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		errs = append(errs, fmt.Errorf("checkpoint: %w", err))
	}
	errs = append(errs, s.W.Close())
	return errors.Join(errs...)
}

func NowMS() int64 {
	return time.Now().UnixMilli()
}
