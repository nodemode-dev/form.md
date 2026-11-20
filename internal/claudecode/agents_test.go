package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func standIn(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAgentStatusNamesEachProgramWithItsVersionAndInstallLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-ins are shell scripts")
	}
	bin, empty := t.TempDir(), t.TempDir()
	claude := standIn(t, bin, "claude", `case "$1" in --version) echo "9.9.9 (stand-in)";; auth) echo '{"loggedIn": true}';; esac`)
	standIn(t, bin, "codex", `echo "codex-cli 0.99.0"`)
	standIn(t, bin, "cursor-agent", `echo "2026.11.01-standin"`)
	standIn(t, bin, "agy", `echo "agy 0.7.1"`)
	t.Setenv("FORMMD_CLAUDE_BIN", claude)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")
	found := AgentStatus("http://127.0.0.1:8750", true)
	for kind, version := range map[string]string{"claude": "9.9.9", "codex": "0.99.0", "cursor": "2026.11.01-standin", "antigravity": "0.7.1"} {
		a := found[kind]
		if !a.Installed || a.Version == nil || *a.Version != version {
			t.Errorf("%s: %+v, want installed at version %s", kind, a, version)
		}
		if a.Install == "" || strings.Contains(a.Install, ". ") {
			t.Errorf("%s: the install line is not a command: %q", kind, a.Install)
		}
	}
	if s := found["claude"].SignedIn; s == nil || !*s {
		t.Errorf("claude signed in: %v, want true", s)
	}
	if found["codex"].SignedIn != nil || found["cursor"].SignedIn != nil {
		t.Errorf("codex and cursor: signed in must be unknown")
	}
	t.Setenv("PATH", empty)
	missing := AgentStatus("http://127.0.0.1:8750", true)
	for _, kind := range []string{"codex", "cursor", "antigravity"} {
		a := missing[kind]
		if a.Installed || a.Version != nil || a.Install == "" || a.Finding != "missing" || !strings.Contains(a.Said, "was not found. Looked in: PATH (") {
			t.Errorf("%s without the program: %+v", kind, a)
		}
	}
	raw, _ := json.Marshal(missing["codex"])
	if !strings.Contains(string(raw), `"version":null`) || !strings.Contains(string(raw), `"signedIn":null`) {
		t.Errorf("unknown values are not null: %s", raw)
	}
}

func TestCodexIsToldTheVariableNeverTheToken(t *testing.T) {
	args := strings.Join(codexArgs("http://127.0.0.1:8750"), " ")
	if !strings.Contains(args, `mcp_servers.formmd_tab.url="http://127.0.0.1:8750/mcp"`) || !strings.Contains(args, `mcp_servers.formmd_tab.bearer_token_env_var="FORMMD_TAB_TOKEN"`) {
		t.Fatalf("codex arguments: %s", args)
	}
	if strings.Contains(args, "fmd_") {
		t.Fatalf("a token is on the command line: %s", args)
	}
}

func TestCursorConfigKeepsOtherServersAndHoldsNoToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".cursor", "mcp.json"), []byte(`{"mcpServers":{"mine":{"url":"https://example.test/mcp"}},"other":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := CursorConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Servers map[string]struct {
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
		Other int `json:"other"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Servers["mine"].URL != "https://example.test/mcp" || got.Other != 1 {
		t.Errorf("the person's own entries were not kept: %s", raw)
	}
	tab := got.Servers[TabServer]
	if tab.URL != "${env:FORMMD_MCP_URL}" || tab.Headers["Authorization"] != "Bearer ${env:FORMMD_TAB_TOKEN}" {
		t.Errorf("the tab entry: %+v", tab)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("mode %o, want 600", info.Mode().Perm())
		}
	}
}
