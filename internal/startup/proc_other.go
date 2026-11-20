//go:build !unix && !windows

package startup

func Alive(pid int) bool { return false }

func Created(pid int) float64 { return 0 }

func end(pid int) bool { return false }
