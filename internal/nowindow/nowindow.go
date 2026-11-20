// Package nowindow starts every program the server starts. On Windows a
// program started through it opens no console window, so the desktop app,
// which starts the server with no window, shows none for what the server
// starts either. FORMMD_ENGINE_WINDOWED=0 turns that off for one start, and
// a server started by the desktop app then opens a console window of its own
// with a copy of its output, so a tester can watch it. On macOS and Linux
// nothing changes.
package nowindow

import (
	"context"
	"os"
	"os/exec"
)

// Env is the family's switch: a window program unless it is 0.
const Env = "FORMMD_ENGINE_WINDOWED"

// Shown reports whether this start asked to see the console (Env is 0).
func Shown() bool { return os.Getenv(Env) == "0" }

// Command is exec.Command, with no console window on Windows.
func Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	if !Shown() {
		hide(cmd)
	}
	return cmd
}

// CommandContext is exec.CommandContext, with no console window on Windows.
func CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	if !Shown() {
		hide(cmd)
	}
	return cmd
}
