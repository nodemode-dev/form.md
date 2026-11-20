//go:build !windows

package nowindow

import "os/exec"

func hide(cmd *exec.Cmd) {}

// OpenConsole does nothing outside Windows: the output is where it was.
func OpenConsole() {}
