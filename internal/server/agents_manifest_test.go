package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/nodemode-ai/form.md/internal/formext"
)

const manifestPath = "agents/manifest.json"

type manifestDock struct {
	Present   bool     `json:"present"`
	Positions []string `json:"positions"`
	Tabs      []string `json:"tabs"`
}

var dock = manifestDock{Present: true, Positions: []string{"bottom"}, Tabs: []string{"chat", "terminal", "claude"}}

type manifestSkill struct {
	Name      string   `json:"name"`
	SubSkills []string `json:"sub_skills"`
}

type manifestMCP struct {
	Tools         []string `json:"tools"`
	ReadOnly      []string `json:"read_only"`
	ApprovalGated []string `json:"approval_gated"`
	Transports    []string `json:"transports"`
}

type agentManifest struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	App        string            `json:"app"`
	MCP        *manifestMCP      `json:"mcp"`
	Skills     []manifestSkill   `json:"skills"`
	Dock       manifestDock      `json:"dock"`
	Extensions map[string]string `json:"extensions"`
}

func listedTools(t *testing.T, e *env, token string) *manifestMCP {
	t.Helper()
	r := e.client().do("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}, withBearer(token))
	if r.Status != http.StatusOK {
		t.Fatalf("tools/list: %d %s", r.Status, r.Body)
	}
	var out struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Annotations struct {
					ReadOnly bool `json:"readOnlyHint"`
				} `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatal(err)
	}
	m := &manifestMCP{Tools: []string{}, ReadOnly: []string{}, ApprovalGated: []string{}, Transports: []string{"http", "stdio"}}
	for _, tool := range out.Result.Tools {
		m.Tools = append(m.Tools, tool.Name)
		if tool.Annotations.ReadOnly {
			m.ReadOnly = append(m.ReadOnly, tool.Name)
		}
	}
	return m
}

func shippedSkills(t *testing.T, root string) []manifestSkill {
	t.Helper()
	out := []manifestSkill{}
	dirs, err := os.ReadDir(filepath.Join(root, ".claude", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		s := manifestSkill{Name: d.Name(), SubSkills: []string{}}
		subs, err := os.ReadDir(filepath.Join(root, ".claude", "skills", d.Name(), "sub-skills"))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		for _, sub := range subs {
			if sub.IsDir() {
				s.SubSkills = append(s.SubSkills, sub.Name())
			}
		}
		out = append(out, s)
	}
	return out
}

func TestAgentManifest(t *testing.T) {
	root := filepath.Join("..", "..")
	e := newEnv(t, "")
	owner := e.setupOwner()
	m := agentManifest{
		Format:     "form-agent-manifest",
		Version:    1,
		App:        "form.md",
		MCP:        listedTools(t, e, owner.apiToken("write")),
		Skills:     shippedSkills(t, root),
		Dock:       dock,
		Extensions: map[string]string{"host_api": formext.API},
	}
	want, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, '\n')
	file := filepath.Join(root, manifestPath)
	if os.Getenv("FORMMD_AGENTS_MANIFEST") == "write" {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, want, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s: %d tools, %d read-only, %d skills", manifestPath, len(m.MCP.Tools), len(m.MCP.ReadOnly), len(m.Skills))
		return
	}
	have, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%s: %v; run ./form-md agents manifest", manifestPath, err)
	}
	if !bytes.Equal(have, want) {
		t.Fatalf("%s is not what the server lists; run ./form-md agents manifest\nwant:\n%s", manifestPath, want)
	}
}
