package server_test

import (
	"net/http"
	"testing"
)

func listTools(t *testing.T, e *env, token string) map[string]map[string]any {
	t.Helper()
	r := e.client().do("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}, withBearer(token))
	if r.Status != http.StatusOK {
		t.Fatalf("tools/list: %d %s", r.Status, r.Body)
	}
	res, _ := r.JSON(t)["result"].(map[string]any)
	list, _ := res["tools"].([]any)
	out := map[string]map[string]any{}
	for _, v := range list {
		tool, _ := v.(map[string]any)
		ann, _ := tool["annotations"].(map[string]any)
		out[tool["name"].(string)] = ann
	}
	return out
}

func TestWindowToolsAreNotAnnouncedReadOnly(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	write := listTools(t, e, owner.apiToken("write"))
	for _, name := range []string{"window_command_run", "extension_command"} {
		ann, ok := write[name]
		if !ok {
			t.Fatalf("%s missing for a write credential", name)
		}
		if ann["readOnlyHint"] != false || ann["destructiveHint"] != true {
			t.Fatalf("%s annotations: %v", name, ann)
		}
	}
	ro := owner.apiToken("read")
	read := listTools(t, e, ro)
	if _, ok := read["extension_command"]; ok {
		t.Fatal("extension_command listed to a read-only credential")
	}
	r := e.client().do("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "extension_command", "arguments": map[string]any{"id": "x", "command": "y"}}}, withBearer(ro))
	if out := r.JSON(t); out["error"] == nil {
		t.Fatalf("extension_command ran for a read-only credential: %s", r.Body)
	}
}
