package testguard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/agentlookup"
)

var Names = []string{"claude", "codex", "cursor-agent", "agent", "agy"}

var cleared = []string{"NPM_CONFIG_PREFIX", "npm_config_prefix", "PNPM_HOME", "VOLTA_HOME", "BUN_INSTALL", "NVM_SYMLINK",
	"FORMMD_CLAUDE_BIN", "FORMMD_CODEX_BIN", "FORMMD_CURSOR_BIN", "FORMMD_ANTIGRAVITY_BIN"}

const Shell = `#!/bin/sh
script=""
while [ $# -gt 0 ]; do if [ "$1" = "-c" ]; then shift; script="$1"; fi; shift; done
if [ -n "$script" ]; then exec /bin/sh -c "$script"; fi
exec /bin/sh
`

type runner interface{ Run() int }

func Run(m runner) int {
	dir, err := os.MkdirTemp("", "formmd-agent-stand-ins-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "agent program guard:", err)
		return 1
	}
	defer os.RemoveAll(dir)
	bin, home, calls, shell := filepath.Join(dir, "bin"), filepath.Join(dir, "home"), filepath.Join(dir, "calls.log"), filepath.Join(dir, "sh")
	for _, d := range []string{bin, home} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, "agent program guard:", err)
			return 1
		}
	}
	for _, name := range Names {
		body := "#!/bin/sh\necho \"" + name + " $*\" >> '" + calls + "'\nif [ \"$1\" = \"--version\" ]; then echo \"0.0.0 (stand-in)\"; fi\nexit 0\n"
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "agent program guard:", err)
			return 1
		}
	}
	if err := os.WriteFile(shell, []byte(Shell), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "agent program guard:", err)
		return 1
	}
	for _, key := range cleared {
		os.Unsetenv(key)
	}
	os.Setenv("HOME", home)
	os.Setenv("SHELL", shell)
	os.Setenv("PATH", strings.Join([]string{bin, "/usr/bin", "/bin"}, string(os.PathListSeparator)))
	agentlookup.SystemFolders = func(string) []string { return nil }
	agentlookup.Forget()
	code := m.Run()
	started := []string{}
	if raw, err := os.ReadFile(calls); err == nil {
		started = strings.Split(strings.TrimSpace(string(raw)), "\n")
	}
	seen := map[string]bool{}
	for _, line := range started {
		seen[line] = true
	}
	lines := []string{}
	for line := range seen {
		lines = append(lines, line)
	}
	sort.Strings(lines)
	said := "none"
	if len(lines) > 0 {
		said = strings.Join(lines, ", ")
	}
	fmt.Printf("agent program stand-ins started %d time(s): %s\n", len(started), said)
	return code
}
