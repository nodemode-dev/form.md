//go:build !unix

package ops

func ownerOf(path string) (int, int, bool) { return 0, 0, false }

func chownTree(root string, uid, gid int) error { return nil }

func matchOwner(dataDir string) error { return nil }

func isMount(path string) bool { return false }
