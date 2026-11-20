package agentlookup_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nodemode-ai/form.md/internal/agentlookup"
	"github.com/nodemode-ai/form.md/internal/agentlookup/testguard"
)

func TestMain(m *testing.M) { os.Exit(testguard.Run(m)) }

const zsh = `#!/bin/sh
login=0; inter=0; script=""
while [ $# -gt 0 ]; do
  case "$1" in -l) login=1 ;; -i) inter=1 ;; -c) shift; script="$1" ;; esac
  shift
done
if [ "$login" = 1 ] && [ -f "$HOME/.zprofile-path" ]; then PATH="$(cat "$HOME/.zprofile-path"):$PATH"; fi
if [ "$inter" = 1 ] && [ -f "$HOME/.zshrc-path" ]; then PATH="$(cat "$HOME/.zshrc-path"):$PATH"; fi
export PATH
if [ -n "$script" ]; then exec /bin/sh -c "$script"; fi
exec /bin/sh
`

const nodeScript = `#!/usr/bin/env node
if [ "$1" = "--version" ]; then echo "1.2.3 (stand-in)"; exit 0; fi
echo "first PATH entry: ${PATH%%:*}"
`

const node = `#!/bin/sh
script="$1"; shift; exec /bin/sh "$script" "$@"
`

func put(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func madeUpHome(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-ins are shell scripts")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", put(t, filepath.Join(home, "shell", "zsh"), zsh))
	t.Setenv("PATH", "/usr/bin:/bin:/usr/sbin:/sbin")
	agentlookup.Forget()
	t.Cleanup(agentlookup.Forget)
	return home
}

func look(program string, vars ...string) agentlookup.Found {
	found := agentlookup.Find(program, agentlookup.Query{OverrideVars: vars, ShellTimeout: 10 * time.Second, Fresh: true})
	return agentlookup.Check(found, nil, 15*time.Second, true, "")
}

func TestEachProgramIsFoundInAKnownFolderAndRunsWithItsOwnFolderFirst(t *testing.T) {
	home := madeUpHome(t)
	bin := filepath.Join(home, ".nvm", "versions", "node", "v22.3.0", "bin")
	put(t, filepath.Join(bin, "node"), node)
	for program, name := range map[string]string{"claude": "claude", "codex": "codex", "cursor": "cursor-agent", "antigravity": "agy"} {
		want := put(t, filepath.Join(bin, name), nodeScript)
		found := look(program)
		if found.Path != want || found.How != "folder" || found.State != "runs" || found.Version != "1.2.3" {
			t.Errorf("%s: %+v, want %s from a known folder, running at 1.2.3", program, found, want)
		}
		env := agentlookup.ChildEnv(agentlookup.Child{Path: found.Path})
		res, err := agentlookup.Run([]string{found.Path}, env, 15*time.Second, "")
		if err != nil || strings.TrimSpace(res.Stdout) != "first PATH entry: "+bin {
			t.Errorf("%s started: %q %v, want its own folder first on PATH", program, res.Stdout, err)
		}
	}
}

func TestTheOrderIsOverrideThenPathThenFolderThenShell(t *testing.T) {
	home := madeUpHome(t)
	inShell := put(t, filepath.Join(home, "tools", "bin", "codex"), "#!/bin/sh\necho 4.0.0\n")
	if err := os.WriteFile(filepath.Join(home, ".zshrc-path"), []byte(filepath.Dir(inShell)), 0o644); err != nil {
		t.Fatal(err)
	}
	if found := look("codex"); found.Path != inShell || found.How != "shell" || found.State != "runs" {
		t.Fatalf("shell: %+v", found)
	}
	inFolder := put(t, filepath.Join(home, ".local", "bin", "codex"), "#!/bin/sh\necho 3.0.0\n")
	if found := look("codex"); found.Path != inFolder || found.How != "folder" {
		t.Fatalf("folder: %+v", found)
	}
	onPath := put(t, filepath.Join(home, "path", "codex"), "#!/bin/sh\necho 2.0.0\n")
	t.Setenv("PATH", filepath.Dir(onPath)+":/usr/bin:/bin")
	if found := look("codex"); found.Path != onPath || found.How != "path" {
		t.Fatalf("path: %+v", found)
	}
	chosen := put(t, filepath.Join(home, "chosen", "my-codex"), "#!/bin/sh\necho 1.0.0\n")
	t.Setenv("FORMMD_CODEX_BIN", chosen)
	if found := look("codex", "FORMMD_CODEX_BIN"); found.Path != chosen || found.How != "override" || found.Version != "1.0.0" {
		t.Fatalf("override: %+v", found)
	}
	setting := chosen
	found := agentlookup.Find("codex", agentlookup.Query{Setting: &setting, OverrideVars: []string{"FORMMD_CODEX_BIN"}})
	if found.Path != chosen || found.How != "setting" {
		t.Fatalf("setting: %+v", found)
	}
}

func TestABrokenProgramSaysItsOwnErrorLineNeverTheEnvHint(t *testing.T) {
	home := madeUpHome(t)
	broken := put(t, filepath.Join(home, ".local", "bin", "claude"), "#!/bin/sh\necho \"env: node: No such file or directory\" >&2\necho \"env: use -[v]S to pass options in shebang lines\" >&2\nexit 127\n")
	found := look("claude")
	status := agentlookup.Answer(found, "", "")
	if status.Installed || status.Finding != "broken" || status.Error != "env: node: No such file or directory" || strings.Contains(status.Said, "use -[v]S to pass options") {
		t.Fatalf("broken: %+v", status)
	}
	if want := "Claude Code was found at " + broken + " (a known install folder) but does not start: env: node: No such file or directory."; status.Said != want {
		t.Fatalf("said %q, want %q", status.Said, want)
	}
}

func TestAMissingProgramSaysWhereItLooked(t *testing.T) {
	home := madeUpHome(t)
	put(t, filepath.Join(home, ".local", "bin", "agent"), "#!/bin/sh\necho 9.9.9\n")
	found := look("cursor")
	status := agentlookup.Answer(found, agentlookup.Install("cursor", ""), "")
	if status.Installed || status.Finding != "missing" || status.Path != nil || status.Version != nil || len(status.Looked) != 3 {
		t.Fatalf("missing: %+v", status)
	}
	for _, part := range []string{"Cursor was not found. Looked in: PATH (/usr/bin:/bin:/usr/sbin:/sbin); ~/.local/bin, ~/.npm-global/bin, ", "; your shell (zsh, login and interactive). Install it with: curl https://cursor.com/install -fsS | bash."} {
		if !strings.Contains(status.Said, part) {
			t.Errorf("said %q, without %q", status.Said, part)
		}
	}
	real := put(t, filepath.Join(home, ".local", "share", "cursor-agent", "versions", "2026.11.01", "cursor-agent"), "#!/bin/sh\necho 2026.11.01\n")
	if err := os.Remove(filepath.Join(home, ".local", "bin", "agent")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, ".local", "bin", "agent")); err != nil {
		t.Fatal(err)
	}
	if found := look("cursor"); found.State != "runs" || found.How != "folder" {
		t.Fatalf("agent linked into Cursor's own folder: %+v", found)
	}
}
