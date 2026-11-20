//go:build unix

package startup

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const ticksPerSecond = 100

func stat(pid int) []string {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return nil
	}
	i := bytes.LastIndexByte(raw, ')')
	if i < 0 {
		return nil
	}
	return strings.Fields(string(raw[i+1:]))
}

func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return false
	}
	if fields := stat(pid); len(fields) > 0 && fields[0] == "Z" {
		return false
	}
	return true
}

func bootTime() float64 {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if rest, ok := strings.CutPrefix(line, "btime "); ok {
			at, _ := strconv.ParseFloat(strings.TrimSpace(rest), 64)
			return at
		}
	}
	return 0
}

func elapsed(text string) (float64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	days := 0.0
	if d, rest, ok := strings.Cut(text, "-"); ok {
		n, err := strconv.ParseFloat(d, 64)
		if err != nil {
			return 0, false
		}
		days, text = n, rest
	}
	total := 0.0
	for _, part := range strings.Split(text, ":") {
		n, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, false
		}
		total = total*60 + n
	}
	return days*86400 + total, true
}

func Created(pid int) float64 {
	if pid <= 0 {
		return 0
	}
	if fields := stat(pid); len(fields) > 19 {
		ticks, err := strconv.ParseFloat(fields[19], 64)
		if boot := bootTime(); err == nil && boot > 0 {
			return boot + ticks/ticksPerSecond
		}
	}
	out, err := exec.Command("ps", "-o", "etime=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0
	}
	if ago, ok := elapsed(string(out)); ok {
		return float64(time.Now().UnixNano())/1e9 - ago
	}
	return 0
}

func end(pid int) bool {
	_ = syscall.Kill(pid, syscall.SIGTERM)
	if gone(pid, endWait) {
		return true
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	return gone(pid, 2*time.Second)
}
