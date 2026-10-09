//go:build !unix

package ops

import (
	"os"
	"path/filepath"
)

func Lock(dataDir string) (func(), error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, LockFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrBusy
		}
		return nil, err
	}
	return func() {
		f.Close()
		os.Remove(path)
	}, nil
}
