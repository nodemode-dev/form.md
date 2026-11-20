//go:build windows

package nowindow

import (
	"os"
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW: a console program started with it gets
// no console window. It is what tauri-plugin-shell gives the server itself.
const createNoWindow = 0x08000000

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	freeConsole      = kernel32.NewProc("FreeConsole")
	allocConsole     = kernel32.NewProc("AllocConsole")
)

func hide(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}

// OpenConsole is called first when the server starts. With Env at 0 and no
// console window to write to (the desktop app started it with none), it
// opens one and copies standard output and standard error there as well as
// where they went, so the desktop app still reads both pipes. Started from a
// terminal, the server has a window already and nothing changes.
func OpenConsole() {
	if !Shown() {
		return
	}
	if w, _, _ := getConsoleWindow.Call(); w != 0 {
		return
	}
	_, _, _ = freeConsole.Call()
	if ok, _, _ := allocConsole.Call(); ok == 0 {
		return
	}
	con, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	os.Stdout = both(os.Stdout, con)
	os.Stderr = both(os.Stderr, con)
}

// both gives a file whose writes reach first and con. A write that first
// cannot take (no pipe given) still reaches con.
func both(first, con *os.File) *os.File {
	r, w, err := os.Pipe()
	if err != nil {
		return first
	}
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				if first != nil {
					_, _ = first.Write(buf[:n])
				}
				_, _ = con.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	return w
}
