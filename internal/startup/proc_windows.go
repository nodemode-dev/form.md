//go:build windows

package startup

import (
	"errors"
	"os/exec"
	"strconv"
	"syscall"
)

const queryLimited = 0x1000

func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	defer syscall.CloseHandle(handle)
	event, _ := syscall.WaitForSingleObject(handle, 0)
	return event == syscall.WAIT_TIMEOUT
}

func Created(pid int) float64 {
	if pid <= 0 {
		return 0
	}
	handle, err := syscall.OpenProcess(queryLimited, false, uint32(pid))
	if err != nil {
		return 0
	}
	defer syscall.CloseHandle(handle)
	var made, exited, kernel, user syscall.Filetime
	if syscall.GetProcessTimes(handle, &made, &exited, &kernel, &user) != nil {
		return 0
	}
	return float64(made.Nanoseconds()) / 1e9
}

func end(pid int) bool {
	_ = exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
	return gone(pid, endWait)
}
