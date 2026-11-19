package claudecode

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const DockContract = "2.0.1"

const TabServer = "formmd_tab"

const TabTokenVar = "FORMMD_TAB_TOKEN"

const MCPURLVar = "FORMMD_MCP_URL"

type Program struct {
	Kind    string
	Bin     string
	Title   string
	Install string
}

var Programs = []Program{
	{Kind: "codex", Bin: "codex", Title: "Codex", Install: "npm install -g @openai/codex"},
	{Kind: "cursor", Bin: "cursor-agent", Title: "Cursor", Install: "curl https://cursor.com/install -fsS | bash"},
}

func ClaudeInstall() string {
	if runtime.GOOS == "windows" {
		return "irm https://claude.ai/install.ps1 | iex"
	}
	return "curl -fsSL https://claude.ai/install.sh | bash"
}

func programOf(kind string) (Program, bool) {
	for _, p := range Programs {
		if p.Kind == kind {
			return p, true
		}
	}
	return Program{}, false
}

type AgentInfo struct {
	Title     string  `json:"title"`
	Installed bool    `json:"installed"`
	Version   *string `json:"version"`
	Install   string  `json:"install"`
	SignedIn  *bool   `json:"signedIn"`
}

var agentCache struct {
	sync.Mutex
	at  time.Time
	out map[string]AgentInfo
}

func findProgram(env []string, bin string) string {
	if p, err := exec.LookPath(bin); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
		return p
	}
	if runtime.GOOS == "windows" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sh := loginShell()
	cmd := exec.CommandContext(ctx, sh[0], append(sh[1:], "-c", "command -v '"+strings.ReplaceAll(bin, "'", `'\''`)+"'")...)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if filepath.IsAbs(last) && runnable(last) {
		return last
	}
	return ""
}

func programVersion(env []string, path string) *string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	for _, word := range strings.Fields(string(out)) {
		if word[0] >= '0' && word[0] <= '9' {
			return &word
		}
	}
	return nil
}

func claudeSignedIn(env []string, path string) *bool {
	yes := true
	if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") != "" {
		return &yes
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "auth", "status")
	cmd.Env = env
	out, _ := cmd.Output()
	var data struct {
		LoggedIn *bool `json:"loggedIn"`
	}
	if json.Unmarshal(out, &data) != nil {
		return nil
	}
	return data.LoggedIn
}

func AgentStatus(env []string, fresh bool) map[string]AgentInfo {
	claude := Detect(env, fresh)
	agentCache.Lock()
	defer agentCache.Unlock()
	if !fresh && agentCache.out != nil && time.Since(agentCache.at) < time.Minute {
		return agentCache.out
	}
	out := map[string]AgentInfo{}
	info := AgentInfo{Title: "Claude Code", Installed: claude.Installed, Install: ClaudeInstall()}
	if claude.Installed {
		if claude.Version != "" {
			v := claude.Version
			info.Version = &v
		}
		info.SignedIn = claudeSignedIn(env, claude.Path)
	}
	out["claude"] = info
	for _, p := range Programs {
		a := AgentInfo{Title: p.Title, Install: p.Install}
		if path := findProgram(env, p.Bin); path != "" {
			a.Installed = true
			a.Version = programVersion(env, path)
		}
		out[p.Kind] = a
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
