//go:build unix

package ops

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

func Lock(dataDir string) (func(), error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dataDir, LockFile), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, err
	}
	f.Truncate(0)
	f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}
