package ops

import (
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

var cgroupLimits = []string{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory/memory.limit_in_bytes"}

func SizeMemory() int64 {
	if os.Getenv("GOMEMLIMIT") != "" {
		return 0
	}
	for _, path := range cgroupLimits {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(raw))
		if v == "max" {
			return 0
		}
		limit, err := strconv.ParseInt(v, 10, 64)
		if err != nil || limit <= 0 || limit >= 1<<60 {
			return 0
		}
		soft := limit / 10 * 9
		debug.SetMemoryLimit(soft)
		return soft
	}
	return 0
}
