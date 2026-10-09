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
	paths := cgroupLimits
	if raw, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if rest, ok := strings.CutPrefix(line, "0::"); ok && rest != "/" && rest != "" {
				paths = append([]string{"/sys/fs/cgroup" + rest + "/memory.max"}, paths...)
			}
		}
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(raw))
		limit, err := strconv.ParseInt(v, 10, 64)
		if err != nil || limit <= 0 || limit >= 1<<60 {
			continue
		}
		soft := limit / 10 * 9
		debug.SetMemoryLimit(soft)
		return soft
	}
	return 0
}
