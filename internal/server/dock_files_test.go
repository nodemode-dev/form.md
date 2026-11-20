package server_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const dockFolder = "web/src/dock"
const dockModule = "2.2.1"

var dockImport = regexp.MustCompile(`(?m)(?:^|\s)(?:import|export)\s[^;]*?from\s+['"]([^'"]+)['"]|(?:^|\s)import\s+['"]([^'"]+)['"]|import\(\s*['"]([^'"]+)['"]\s*\)`)

func checkDockFolder(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, dockFolder)
	raw, err := os.ReadFile(filepath.Join(dir, "MANIFEST.json"))
	if err != nil {
		t.Fatalf("%s/MANIFEST.json: %v", dockFolder, err)
	}
	var m struct {
		Module  string            `json:"module"`
		Version string            `json:"version"`
		Files   map[string]string `json:"files"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("%s/MANIFEST.json: %v", dockFolder, err)
	}
	if m.Module != "dock" || m.Version != dockModule {
		t.Errorf("%s/MANIFEST.json is module %q version %q, want dock %s", dockFolder, m.Module, m.Version, dockModule)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if name == "MANIFEST.json" {
			continue
		}
		seen[name] = true
		want, listed := m.Files[name]
		if e.IsDir() || !listed {
			t.Errorf("%s/%s is not in MANIFEST.json: the folder is the family's and takes no extra file", dockFolder, name)
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s/%s: %v", dockFolder, name, err)
			continue
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != want {
			t.Errorf("%s/%s does not match MANIFEST.json: copy it again from the family module, never edit it here", dockFolder, name)
		}
		if !strings.HasSuffix(name, ".ts") {
			continue
		}
		for _, hit := range dockImport.FindAllStringSubmatch(string(body), -1) {
			spec := hit[1] + hit[2] + hit[3]
			inFolder := strings.HasPrefix(spec, "./") && !strings.Contains(spec[2:], "/")
			if !inFolder && !strings.HasPrefix(spec, "@xterm/") {
				t.Errorf("%s/%s imports %q: the folder may import only its own files and the terminal emulator", dockFolder, name, spec)
			}
		}
	}
	for name := range m.Files {
		if !seen[name] {
			t.Errorf("%s/%s is listed in MANIFEST.json and missing", dockFolder, name)
		}
	}
}

func TestDockFolder(t *testing.T) {
	checkDockFolder(t, filepath.Join("..", ".."))
}
