//go:build unix

package ops

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func ownerOf(path string) (int, int, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, 0, false
	}
	s, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return int(s.Uid), int(s.Gid), true
}

func chownTree(root string, uid, gid int) error {
	if os.Geteuid() != 0 {
		return nil
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if u, g, ok := ownerOf(p); ok && (u != uid || g != gid) {
			return os.Lchown(p, uid, gid)
		}
		return nil
	})
}

func matchOwner(dataDir string) error {
	uid, gid, ok := ownerOf(dataDir)
	if !ok {
		return nil
	}
	return chownTree(dataDir, uid, gid)
}

func isMount(path string) bool {
	a, err1 := os.Stat(path)
	b, err2 := os.Stat(filepath.Dir(path))
	if err1 != nil || err2 != nil {
		return false
	}
	sa, ok1 := a.Sys().(*syscall.Stat_t)
	sb, ok2 := b.Sys().(*syscall.Stat_t)
	return ok1 && ok2 && sa.Dev != sb.Dev
}
