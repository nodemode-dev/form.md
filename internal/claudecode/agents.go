package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/agentlookup"
)

const DockContract = "2.0.1"

const TabServer = "formmd_tab"

const TabTokenVar = "FORMMD_TAB_TOKEN"

const MCPURLVar = "FORMMD_MCP_URL"

type Program struct {
	Kind    string
	Title   string
	Install string
}

var Tabs = []string{"codex", "cursor", "antigravity"}

func ClaudeInstall() string { return agentlookup.Install("claude", "") }

func programOf(kind string) (Program, bool) {
	for _, k := range Tabs {
		if k == kind {
			return Program{Kind: kind, Title: agentlookup.Programs[kind].Title, Install: agentlookup.Install(kind, "")}, true
		}
	}
	return Program{}, false
}

type AgentInfo struct {
	Title string `json:"title"`
	agentlookup.Status
	Install  string `json:"install"`
	SignedIn *bool  `json:"signedIn"`
}

var agentCache struct {
	sync.Mutex
	at  time.Time
	out map[string]AgentInfo
}

func findProgram(engineURL, kind string) agentlookup.Found {
	found := located(engineURL, kind, false)
	if found.State != "runs" {
		found = located(engineURL, kind, true)
	}
	return found
}

func claudeSignedIn(env []string, path string) *bool {
	yes := true
	if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") != "" {
		return &yes
	}
	res, _ := agentlookup.Run([]string{path, "auth", "status"}, env, 15*time.Second, "")
	var data struct {
		LoggedIn *bool `json:"loggedIn"`
	}
	if json.Unmarshal([]byte(res.Stdout), &data) != nil {
		return nil
	}
	return data.LoggedIn
}

func AgentStatus(engineURL string, fresh bool) map[string]AgentInfo {
	agentCache.Lock()
	defer agentCache.Unlock()
	if !fresh && agentCache.out != nil && time.Since(agentCache.at) < time.Minute {
		return agentCache.out
	}
	out := map[string]AgentInfo{}
	for _, kind := range agentlookup.Order {
		found := located(engineURL, kind, fresh)
		install := agentlookup.Install(kind, "")
		a := AgentInfo{Title: found.Title, Status: agentlookup.Answer(found, install, Otherwise), Install: install}
		if kind == "claude" && a.Installed {
			a.SignedIn = claudeSignedIn(childEnv(engineURL, found.Path), found.Path)
		}
		out[kind] = a
	}
	agentCache.at, agentCache.out = time.Now(), out
	return out
}

func codexArgs(engineURL string) []string {
	return []string{
		"-c", "mcp_servers." + TabServer + `.url="` + engineURL + `/mcp"`,
		"-c", "mcp_servers." + TabServer + `.bearer_token_env_var="` + TabTokenVar + `"`,
	}
}

func cursorEntry() map[string]any {
	return map[string]any{"url": "${env:" + MCPURLVar + "}", "headers": map[string]any{"Authorization": "Bearer ${env:" + TabTokenVar + "}"}}
}

func CursorConfig(folder string) (string, error) {
	dir := filepath.Join(folder, ".cursor")
	path := filepath.Join(dir, "mcp.json")
	current := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &current); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	servers, _ := current["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	want, _ := json.Marshal(cursorEntry())
	if have, err := json.Marshal(servers[TabServer]); err == nil && string(have) == string(want) {
		return path, nil
	}
	servers[TabServer] = cursorEntry()
	current["mcpServers"] = servers
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return path, writePrivate(dir, path, raw)
}

func writePrivate(dir, path string, raw []byte) error {
	tmp, err := os.CreateTemp(dir, ".tab-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func TabConfigFile(dataDir, tabKey, engineURL, token string) (string, error) {
	dir := filepath.Join(dataDir, "agent-config")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "tab-"+tabKey+".json")
	return path, writePrivate(dir, path, []byte(MCPConfig(engineURL, token)))
}

func SweepTabConfigs(dataDir string) {
	found, _ := filepath.Glob(filepath.Join(dataDir, "agent-config", "tab-*.json"))
	for _, p := range found {
		_ = os.Remove(p)
	}
}
