package startup

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	ShellEnv = "FORMMD_SHELL_PID"
	LockName = "engine.lock"
	slack    = 3.0
	endWait  = 5 * time.Second
	every    = 2 * time.Second
)

const (
	free = iota
	busy
	leftover
)

var started = time.Now()

type Record struct {
	PID       int     `json:"pid"`
	Started   int64   `json:"started"`
	Born      float64 `json:"born"`
	Shell     int     `json:"shell"`
	ShellBorn float64 `json:"shellBorn"`
}

type Busy struct{ Text string }

func (b *Busy) Error() string { return b.Text }

func ShellPID() int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(ShellEnv)))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func moved(now, then float64) bool {
	return now > 0 && then > 0 && math.Abs(now-then) > slack
}

func shellGone(rec Record) bool {
	return !Alive(rec.Shell) || moved(Created(rec.Shell), rec.ShellBorn)
}

func holder(path string) (Record, int) {
	var rec Record
	raw, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(raw, &rec) != nil || rec.PID <= 0 || rec.PID == os.Getpid() || !Alive(rec.PID) {
		return rec, free
	}
	if rec.Born <= 0 {
		return rec, busy
	}
	if now := Created(rec.PID); now <= 0 || moved(now, rec.Born) {
		return rec, free
	}
	if rec.Shell > 0 && shellGone(rec) {
		return rec, leftover
	}
	return rec, busy
}

func gone(pid int, limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !Alive(pid)
}

func Claim(dataDir string, shell int) (func(), int, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, 0, err
	}
	path := filepath.Join(dataDir, LockName)
	rec, state := holder(path)
	if state == busy {
		return nil, 0, &Busy{fmt.Sprintf("form.md is already running on %s (engine process %d).", dataDir, rec.PID)}
	}
	replaced := 0
	if state == leftover {
		if !end(rec.PID) {
			return nil, 0, &Busy{fmt.Sprintf("form.md could not stop an engine that was left running on %s (engine process %d).", dataDir, rec.PID)}
		}
		replaced = rec.PID
	}
	own := Record{PID: os.Getpid(), Started: started.Unix(), Born: Created(os.Getpid()), Shell: shell, ShellBorn: Created(shell)}
	raw, err := json.Marshal(own)
	if err != nil {
		return nil, 0, err
	}
	fresh := path + "." + strconv.Itoa(os.Getpid())
	if err := os.WriteFile(fresh, raw, 0o600); err != nil {
		return nil, 0, err
	}
	if err := os.Rename(fresh, path); err != nil {
		os.Remove(fresh)
		return nil, 0, err
	}
	release := func() {
		var now Record
		if raw, err := os.ReadFile(path); err == nil && json.Unmarshal(raw, &now) == nil && now.PID == os.Getpid() {
			os.Remove(path)
		}
	}
	return release, replaced, nil
}

func Watch(ctx context.Context, pid int, leave func()) {
	born := Created(pid)
	go func() {
		tick := time.NewTicker(every)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			if !Alive(pid) || moved(Created(pid), born) {
				leave()
				return
			}
		}
	}()
}
