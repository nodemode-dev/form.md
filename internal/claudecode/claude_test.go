package claudecode

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestMCPConfigIsAPrivateFileNotAnArgument(t *testing.T) {
	dir := t.TempDir()
	const token = "fmd_owner_token_0123456789"
	path, err := MCPConfigFile(dir, "user-1", "http://127.0.0.1:8750", token)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(path, token) || strings.Contains(path, "{") {
		t.Fatalf("the argument carries the config itself: %q", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Bearer "+token) || !strings.Contains(string(raw), "http://127.0.0.1:8750/mcp") {
		t.Fatalf("the file does not hold the MCP config: %s", raw)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("the config is readable by others: %v", mode)
		}
	}
	again, err := MCPConfigFile(dir, "user-1", "http://127.0.0.1:8750", "fmd_rotated")
	if err != nil {
		t.Fatal(err)
	}
	if again != path {
		t.Fatalf("a second call wrote another file: %q and %q", path, again)
	}
	if raw, _ := os.ReadFile(path); strings.Contains(string(raw), token) {
		t.Fatal("a revoked token stayed in the file")
	}
	entries, _ := os.ReadDir(dir + "/agent-config")
	if len(entries) != 1 {
		t.Fatalf("temporary files were left behind: %d entries", len(entries))
	}
}
