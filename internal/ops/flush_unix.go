//go:build unix

package ops

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const flushedFile = "form-md.flushed"

func OnFlushRequest(ctx context.Context, dataDir string, flush func(context.Context)) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		defer signal.Stop(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				fctx, cancel := context.WithTimeout(ctx, 20*time.Second)
				flush(fctx)
				cancel()
				os.WriteFile(filepath.Join(dataDir, flushedFile), []byte(strconv.FormatInt(time.Now().UnixNano(), 10)), 0o600)
			}
		}
	}()
}

func RequestFlush(dataDir string) bool {
	raw, err := os.ReadFile(filepath.Join(dataDir, LockFile))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return false
	}
	if unlock, err := Lock(dataDir); err == nil {
		unlock()
		return false
	}
	asked := time.Now().UnixNano()
	if syscall.Kill(pid, syscall.SIGUSR1) != nil {
		return false
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(filepath.Join(dataDir, flushedFile)); err == nil {
			if at, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil && at >= asked {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
