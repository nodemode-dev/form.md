//go:build unix

package startup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func standIn(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	return cmd
}

func finish(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}

func write(t *testing.T, dir string, rec Record) {
	t.Helper()
	raw, _ := json.Marshal(rec)
	if err := os.WriteFile(filepath.Join(dir, LockName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func held(t *testing.T, dir string) Record {
	t.Helper()
	var rec Record
	raw, err := os.ReadFile(filepath.Join(dir, LockName))
	if err != nil || json.Unmarshal(raw, &rec) != nil {
		t.Fatalf("the lock file: %s %v", raw, err)
	}
	return rec
}

func TestAProcessIsAliveUntilItEndsAndItsCreationTimeIsRead(t *testing.T) {
	cmd := standIn(t)
	if !Alive(cmd.Process.Pid) || !Alive(os.Getpid()) || Alive(0) {
		t.Fatalf("alive answers are wrong")
	}
	now := float64(time.Now().UnixNano()) / 1e9
	if at := Created(cmd.Process.Pid); at < now-10 || at > now+10 {
		t.Errorf("the stand-in was created at %v, now is %v", at, now)
	}
	_ = cmd.Process.Kill()
	time.Sleep(100 * time.Millisecond)
	if Alive(cmd.Process.Pid) {
		t.Errorf("a process that ended and was not yet collected counts as alive")
	}
}

func TestAMissingOrDamagedLockIsTaken(t *testing.T) {
	for _, body := range []string{"", "not json at all", "[1, 2]", `{"pid": "x"}`, `{"pid": 999999999, "born": 5}`} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, LockName), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		release, replaced, err := Claim(dir, 0)
		if err != nil || replaced != 0 {
			t.Fatalf("%q: %v, replaced %d", body, err, replaced)
		}
		if rec := held(t, dir); rec.PID != os.Getpid() || rec.Born <= 0 || rec.Shell != 0 {
			t.Errorf("%q: the lock holds %+v", body, rec)
		}
		release()
		if _, err := os.Stat(filepath.Join(dir, LockName)); !os.IsNotExist(err) {
			t.Errorf("%q: the lock was not given up", body)
		}
	}
	dir := filepath.Join(t.TempDir(), "new", "folder")
	if _, _, err := Claim(dir, 0); err != nil {
		t.Errorf("a folder that is not there yet: %v", err)
	}
}

func TestALiveEngineWhoseShellLivesIsRefusedWithTheSentence(t *testing.T) {
	engine, shell := standIn(t), standIn(t)
	for _, rec := range []Record{
		{PID: engine.Process.Pid, Born: Created(engine.Process.Pid), Shell: shell.Process.Pid, ShellBorn: Created(shell.Process.Pid)},
		{PID: engine.Process.Pid, Born: Created(engine.Process.Pid)},
		{PID: engine.Process.Pid},
	} {
		dir := t.TempDir()
		write(t, dir, rec)
		_, _, err := Claim(dir, os.Getpid())
		var busy *Busy
		if !errors.As(err, &busy) || !strings.Contains(err.Error(), "form.md is already running on "+dir) || !strings.Contains(err.Error(), "(engine process ") {
			t.Errorf("%+v: %v", rec, err)
		}
		if !Alive(engine.Process.Pid) || held(t, dir).PID != engine.Process.Pid {
			t.Errorf("%+v: the holder was touched", rec)
		}
	}
}

func TestAnEngineWhoseShellIsGoneIsEndedAndReplaced(t *testing.T) {
	engine, shell := standIn(t), standIn(t)
	dir := t.TempDir()
	write(t, dir, Record{PID: engine.Process.Pid, Born: Created(engine.Process.Pid), Shell: shell.Process.Pid, ShellBorn: Created(shell.Process.Pid)})
	finish(shell)
	_, replaced, err := Claim(dir, 0)
	if err != nil || replaced != engine.Process.Pid {
		t.Fatalf("claim: %v, replaced %d", err, replaced)
	}
	state, _ := engine.Process.Wait()
	if status, ok := state.Sys().(syscall.WaitStatus); !ok || status.Signal() != syscall.SIGTERM {
		t.Errorf("the leftover engine ended with %v, want SIGTERM", state)
	}
	if held(t, dir).PID != os.Getpid() {
		t.Errorf("the lock was not taken")
	}
}

func TestAStoppedEngineWhoseShellIsGoneIsKilledAfterTheWait(t *testing.T) {
	engine, shell := standIn(t), standIn(t)
	dir := t.TempDir()
	write(t, dir, Record{PID: engine.Process.Pid, Born: Created(engine.Process.Pid), Shell: shell.Process.Pid, ShellBorn: Created(shell.Process.Pid)})
	finish(shell)
	_ = engine.Process.Signal(syscall.SIGSTOP)
	began := time.Now()
	_, replaced, err := Claim(dir, 0)
	if err != nil || replaced != engine.Process.Pid {
		t.Fatalf("claim: %v, replaced %d", err, replaced)
	}
	if took := time.Since(began); took < endWait {
		t.Errorf("the stopped engine was killed after %v, before the wait of %v", took, endWait)
	}
	state, _ := engine.Process.Wait()
	if status, ok := state.Sys().(syscall.WaitStatus); !ok || status.Signal() != syscall.SIGKILL {
		t.Errorf("the stopped engine ended with %v, want SIGKILL", state)
	}
}

func TestALockNamingAnUnrelatedProcessIsTakenAndTheProcessIsLeftAlone(t *testing.T) {
	other := standIn(t)
	dir := t.TempDir()
	write(t, dir, Record{PID: other.Process.Pid, Born: Created(other.Process.Pid) - 600, Shell: 999999999})
	_, replaced, err := Claim(dir, 0)
	if err != nil || replaced != 0 || held(t, dir).PID != os.Getpid() {
		t.Fatalf("claim: %v, replaced %d", err, replaced)
	}
	if !Alive(other.Process.Pid) {
		t.Errorf("the unrelated process was ended")
	}
}

func TestTheWatchCallsLeaveOnceTheShellIsGone(t *testing.T) {
	shell := standIn(t)
	left := make(chan time.Time, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Watch(ctx, shell.Process.Pid, func() { left <- time.Now() })
	select {
	case <-left:
		t.Fatalf("leave was called while the shell lives")
	case <-time.After(every + 500*time.Millisecond):
	}
	ended := time.Now()
	finish(shell)
	select {
	case at := <-left:
		if at.Sub(ended) > every+time.Second {
			t.Errorf("leave came %v after the shell ended", at.Sub(ended))
		}
	case <-time.After(every + 2*time.Second):
		t.Fatalf("leave was never called")
	}
}

func TestTheShellProcessIdComesFromTheVariable(t *testing.T) {
	for value, want := range map[string]int{"": 0, "x": 0, "-4": 0, " 4321 ": 4321} {
		t.Setenv(ShellEnv, value)
		if got := ShellPID(); got != want {
			t.Errorf("%q: %d, want %d", value, got, want)
		}
	}
}
