package server_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nodemode-ai/form.md/internal/agentlookup"
)

func dockStandIns(t *testing.T) (string, string) {
	t.Helper()
	bin, out := t.TempDir(), t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(bin, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	claude := write("claude", `case "$1" in --version) echo "9.9.9 (stand-in)"; exit 0;; auth) echo '{"loggedIn": true}'; exit 0;; esac
prev=""; for a in "$@"; do if [ "$prev" = "--mcp-config" ]; then printf '%s' "$a" > '`+out+`/claude.cfg'; fi; prev="$a"; done
echo started; read line`)
	for _, name := range []string{"codex", "cursor-agent", "agy"} {
		write(name, `case "$1" in --version) echo "1.2.3"; exit 0;; esac
printf '%s\n' "$*" > '`+out+`/`+name+`.args'
printf '%s' "$FORMMD_TAB_TOKEN" > '`+out+`/`+name+`.token'
echo "`+name+` stand-in started"; read line`)
	}
	t.Setenv("FORMMD_TERMINAL", "")
	t.Setenv("FORMMD_CLAUDE_BIN", claude)
	t.Setenv("FORMMD_AGENT_DIR", "")
	t.Setenv("HOME", t.TempDir())
	agentlookup.Forget()
	t.Setenv("PATH", bin+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")
	return bin, out
}

func waitFile(t *testing.T, path string) string {
	t.Helper()
	for i := 0; i < 100; i++ {
		if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
			return string(raw)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s was never written", path)
	return ""
}

func TestAgentTabsGetTheirOwnTokenWhichEndsWithTheTab(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the embedded terminal needs macOS or Linux")
	}
	_, out := dockStandIns(t)
	e := newEnv(t, "")
	owner := e.setupOwner()
	dock := owner.mustJSON("GET", "/api/dock", nil, http.StatusOK)
	if dock["contract"] != "2.0.1" || dock["consent"] != "credential" || len(dock["kinds"].([]any)) != 6 {
		t.Fatalf("availability: %v", dock)
	}
	status := owner.mustJSON("GET", "/api/dock/agents", nil, http.StatusOK)
	for _, kind := range []string{"claude", "codex", "cursor", "antigravity"} {
		if a, _ := status[kind].(map[string]any); a == nil || a["installed"] != true || a["version"] == nil || a["install"] == "" || a["finding"] != "runs" || a["said"] == "" {
			t.Errorf("status of %s: %v", kind, status[kind])
		}
	}
	folder, _ := owner.mustJSON("GET", "/api/claude", nil, http.StatusOK)["folder"].(string)
	shared := filepath.Join(folder, ".agents", "mcp_config.json")
	if err := os.MkdirAll(filepath.Dir(shared), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shared, []byte(`{"mcpServers":{"mine":{"serverUrl":"https://example.test/mcp"}},"other":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, kind := range []string{"shell", "claude", "codex", "cursor", "antigravity"} {
		tab := owner.mustJSON("POST", "/api/terminal", map[string]any{"kind": kind, "cols": 80, "rows": 24}, http.StatusCreated)
		if tab["kind"] != kind {
			t.Fatalf("asked for %s, got %v", kind, tab)
		}
		ids[kind] = tab["id"].(string)
		time.Sleep(3 * time.Millisecond)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			owner.do("DELETE", "/api/terminal/"+id, nil)
		}
	})
	var listed struct {
		Terminals []struct {
			Kind string `json:"kind"`
		} `json:"terminals"`
	}
	if err := json.Unmarshal(owner.do("GET", "/api/terminal", nil).Body, &listed); err != nil {
		t.Fatal(err)
	}
	order := []string{}
	for _, tab := range listed.Terminals {
		order = append(order, tab.Kind)
	}
	if strings.Join(order, " ") != "shell claude codex cursor antigravity" {
		t.Errorf("the list is not oldest first: %v", order)
	}
	cfg := waitFile(t, filepath.Join(out, "claude.cfg"))
	info, err := os.Stat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("the Claude Code tab's config has mode %o, want 600", info.Mode().Perm())
	}
	var held struct {
		Servers map[string]struct {
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	raw, _ := os.ReadFile(cfg)
	if err := json.Unmarshal(raw, &held); err != nil {
		t.Fatal(err)
	}
	tokens := map[string]string{
		"claude": strings.TrimPrefix(held.Servers["form-md"].Headers["Authorization"], "Bearer "),
		"codex":  waitFile(t, filepath.Join(out, "codex.token")),
		"cursor": waitFile(t, filepath.Join(out, "cursor-agent.token")),
	}
	tokens["antigravity"] = waitFile(t, filepath.Join(out, "agy.token"))
	var entries struct {
		Servers map[string]struct {
			URL     string            `json:"serverUrl"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
		Other int `json:"other"`
	}
	own, _ := filepath.Glob(filepath.Join(filepath.Dir(folder), "agent-tabs", "antigravity", "*", ".agents", "mcp_config.json"))
	if len(own) != 1 {
		t.Fatalf("the Antigravity tab's own folder file: %v", own)
	}
	raw, _ = os.ReadFile(own[0])
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	if tab := entries.Servers["formmd_tab"]; !strings.HasSuffix(tab.URL, "/mcp") || tab.Headers["Authorization"] != "Bearer "+tokens["antigravity"] {
		t.Errorf("the Antigravity tab's entry: %s", raw)
	}
	if kept, _ := os.ReadFile(shared); string(kept) != `{"mcpServers":{"mine":{"serverUrl":"https://example.test/mcp"}},"other":1}` {
		t.Errorf("the shared folder's file was changed: %s", kept)
	}
	banner := ""
	for i := 0; i < 100 && !strings.Contains(banner, "agy stand-in started"); i++ {
		time.Sleep(50 * time.Millisecond)
		banner, _ = owner.do("GET", "/api/terminal/"+ids["antigravity"]+"/read", nil).JSON(t)["text"].(string)
	}
	if !strings.Contains(banner, "agy stand-in started") {
		t.Errorf("the Antigravity tab shows %q, want the stand-in's banner", banner)
	}
	list := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
	for kind, token := range tokens {
		if !strings.HasPrefix(token, "fmd_tab_") {
			t.Fatalf("%s tab token: %q", kind, token)
		}
		if r := e.client().do("POST", "/mcp", list, withBearer(token)); r.Status != http.StatusOK {
			t.Errorf("%s tab token while the tab is open: %d %s", kind, r.Status, r.Body)
		}
		if r := e.client().do("GET", "/api/terminal", nil, withBearer(token)); r.Status != http.StatusForbidden {
			t.Errorf("%s tab token on a window route: %d, want 403", kind, r.Status)
		}
	}
	if tokens["claude"] == tokens["codex"] || tokens["codex"] == tokens["cursor"] {
		t.Errorf("two tabs share a token")
	}
	for _, name := range []string{"codex", "cursor-agent"} {
		if args := waitFile(t, filepath.Join(out, name+".args")); strings.Contains(args, "fmd_") {
			t.Errorf("%s was started with a token on its command line: %s", name, args)
		}
	}
	owner.mustJSON("POST", "/api/terminal/"+ids["shell"]+"/input", map[string]any{"data": "echo n=$(env | grep -c FORMMD_TAB_TOKEN)\r"}, http.StatusOK)
	seen := ""
	for i := 0; i < 100 && !strings.Contains(seen, "n=0"); i++ {
		time.Sleep(50 * time.Millisecond)
		seen, _ = owner.do("GET", "/api/terminal/"+ids["shell"]+"/read", nil).JSON(t)["text"].(string)
	}
	if !strings.Contains(seen, "n=0") {
		t.Errorf("a plain shell tab was given a tab token: %q", seen)
	}
	for kind, token := range tokens {
		owner.mustJSON("DELETE", "/api/terminal/"+ids[kind], nil, http.StatusOK)
		if r := e.client().do("POST", "/mcp", list, withBearer(token)); r.Status != http.StatusUnauthorized {
			t.Errorf("%s tab token after the tab closed: %d, want 401", kind, r.Status)
		}
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Errorf("the closed Claude Code tab's config is still on disk")
	}
	raw, _ = os.ReadFile(shared)
	if string(raw) != `{"mcpServers":{"mine":{"serverUrl":"https://example.test/mcp"}},"other":1}` {
		t.Errorf("after the Antigravity tab closed the shared folder's file holds: %s", raw)
	}
	if _, err := os.Stat(filepath.Dir(filepath.Dir(own[0]))); !os.IsNotExist(err) {
		t.Errorf("the closed Antigravity tab's own folder is still on disk")
	}
}

func TestDockRoutesAnswerOnlyTheSignedInWindow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the embedded terminal needs macOS or Linux")
	}
	dockStandIns(t)
	e := newEnv(t, "")
	owner := e.setupOwner()
	token := owner.apiToken("write")
	calls := []struct {
		method, path string
		body         any
	}{
		{"GET", "/api/dock", nil},
		{"GET", "/api/dock/agents", nil},
		{"POST", "/api/terminal", map[string]any{"kind": "codex"}},
		{"POST", "/api/terminal", map[string]any{"kind": "cursor"}},
	}
	for _, c := range calls {
		if r := e.client().do(c.method, c.path, c.body); r.Status != http.StatusUnauthorized {
			t.Errorf("%s %s with no credential: %d, want 401", c.method, c.path, r.Status)
		}
		if r := e.client().do(c.method, c.path, c.body, withBearer(token)); r.Status != http.StatusForbidden || r.Code(t) != "session_only" {
			t.Errorf("%s %s with an agent's token: %d %s, want 403 session_only", c.method, c.path, r.Status, r.Body)
		}
	}
	if list := owner.mustJSON("GET", "/api/terminal", nil, http.StatusOK)["terminals"].([]any); len(list) != 0 {
		t.Errorf("a refused call opened a tab: %v", list)
	}
}

func TestAMissingAgentProgramIsRefusedWithItsInstallLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the embedded terminal needs macOS or Linux")
	}
	bin, _ := dockStandIns(t)
	if err := os.WriteFile(filepath.Join(bin, "nosh"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", filepath.Join(bin, "nosh"))
	t.Setenv("PATH", t.TempDir())
	e := newEnv(t, "")
	owner := e.setupOwner()
	r := owner.do("POST", "/api/terminal", map[string]any{"kind": "codex"})
	if r.Status != http.StatusBadRequest || r.Code(t) != "agent_missing" || !strings.Contains(string(r.Body), "npm install -g @openai/codex") {
		t.Fatalf("a missing program: %d %s", r.Status, r.Body)
	}
	if left := owner.do("GET", "/api/me/tokens", nil); left.Status != http.StatusOK || strings.Contains(string(left.Body), "fmd_tab_") {
		t.Errorf("a refused tab left a token behind: %d %s", left.Status, left.Body)
	}
}
