package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnAntigravityFileThatIsNotJSONGivesASentenceAndIsLeftAlone(t *testing.T) {
	for _, body := range []string{"not json at all", "[1, 2]", `{"mcpServers": 7}`} {
		folder := t.TempDir()
		path := filepath.Join(folder, ".agents", "mcp_config.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := AntigravityConfig(folder, "http://127.0.0.1:1", "fmd_tab_x")
		if err == nil || !strings.Contains(err.Error(), "is not an MCP servers file form.md can read") || !strings.Contains(err.Error(), path) {
			t.Errorf("%q: %v", body, err)
		}
		AntigravityForget(folder, "fmd_tab_x")
		if kept, _ := os.ReadFile(path); string(kept) != body {
			t.Errorf("%q was changed to %q", body, kept)
		}
	}
}
