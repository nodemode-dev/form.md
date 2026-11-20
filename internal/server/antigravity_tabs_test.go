package server_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func antigravityToken(t *testing.T, path string) string {
	t.Helper()
	var held struct {
		Servers map[string]struct {
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if err := json.Unmarshal(raw, &held); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return strings.TrimPrefix(held.Servers["formmd_tab"].Headers["Authorization"], "Bearer ")
}

func TestEachAntigravityTabHasItsOwnFolderEntryAndToken(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the embedded terminal needs macOS or Linux")
	}
	dockStandIns(t)
	e := newEnv(t, "")
	owner := e.setupOwner()
	folder, _ := owner.mustJSON("GET", "/api/claude", nil, http.StatusOK)["folder"].(string)
	files := filepath.Join(filepath.Dir(folder), "agent-tabs", "antigravity", "*", ".agents", "mcp_config.json")
	open := func() string {
		tab := owner.mustJSON("POST", "/api/terminal", map[string]any{"kind": "antigravity", "cols": 80, "rows": 24}, http.StatusCreated)
		return tab["id"].(string)
	}
	first := open()
	one, _ := filepath.Glob(files)
	if len(one) != 1 {
		t.Fatalf("one tab, files: %v", one)
	}
	second := open()
	t.Cleanup(func() {
		owner.do("DELETE", "/api/terminal/"+first, nil)
		owner.do("DELETE", "/api/terminal/"+second, nil)
	})
	two, _ := filepath.Glob(files)
	if len(two) != 2 {
		t.Fatalf("two tabs, files: %v", two)
	}
	other := two[0]
	if other == one[0] {
		other = two[1]
	}
	tokenOne, tokenOther := antigravityToken(t, one[0]), antigravityToken(t, other)
	if !strings.HasPrefix(tokenOne, "fmd_tab_") || !strings.HasPrefix(tokenOther, "fmd_tab_") || tokenOne == tokenOther {
		t.Fatalf("the two tabs' tokens: %q and %q", tokenOne, tokenOther)
	}
	if _, err := os.Stat(filepath.Join(folder, ".agents")); !os.IsNotExist(err) {
		t.Errorf("an Antigravity tab wrote into the shared agent folder")
	}
	list := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
	for _, token := range []string{tokenOne, tokenOther} {
		if r := e.client().do("POST", "/mcp", list, withBearer(token)); r.Status != http.StatusOK {
			t.Errorf("a tab's token while both are open: %d %s", r.Status, r.Body)
		}
	}
	before, _ := os.ReadFile(other)
	owner.mustJSON("DELETE", "/api/terminal/"+first, nil, http.StatusOK)
	if _, err := os.Stat(filepath.Dir(filepath.Dir(one[0]))); !os.IsNotExist(err) {
		t.Errorf("the closed tab's folder is still on disk")
	}
	if after, _ := os.ReadFile(other); string(after) != string(before) || len(after) == 0 {
		t.Errorf("closing one tab changed the other's file: %s", after)
	}
	if r := e.client().do("POST", "/mcp", list, withBearer(tokenOne)); r.Status != http.StatusUnauthorized {
		t.Errorf("the closed tab's token: %d, want 401", r.Status)
	}
	if r := e.client().do("POST", "/mcp", list, withBearer(tokenOther)); r.Status != http.StatusOK {
		t.Errorf("the open tab's token after the other closed: %d %s", r.Status, r.Body)
	}
	owner.mustJSON("DELETE", "/api/terminal/"+second, nil, http.StatusOK)
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(folder), "agent-tabs", "antigravity", "*")); len(left) != 0 {
		t.Errorf("after both tabs closed their folders remain: %v", left)
	}
}

func TestAnAntigravityTabWhoseFolderCannotBeMadeIsRefusedWithTheReason(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the embedded terminal needs macOS or Linux")
	}
	dockStandIns(t)
	e := newEnv(t, "")
	owner := e.setupOwner()
	folder, _ := owner.mustJSON("GET", "/api/claude", nil, http.StatusOK)["folder"].(string)
	blocked := filepath.Join(filepath.Dir(folder), "agent-tabs", "antigravity")
	if err := os.MkdirAll(filepath.Dir(blocked), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocked, []byte("not a folder"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := owner.do("POST", "/api/terminal", map[string]any{"kind": "antigravity", "cols": 80, "rows": 24})
	if r.Status != http.StatusBadRequest || !strings.Contains(string(r.Body), "terminal_failed") {
		t.Errorf("a tab folder that cannot be made: %d %s", r.Status, r.Body)
	}
}
