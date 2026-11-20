package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func antigravityFile(folder string) string {
	return filepath.Join(folder, ".agents", "mcp_config.json")
}

func antigravityEntry(engineURL, token string) map[string]any {
	return map[string]any{"serverUrl": engineURL + "/mcp", "headers": map[string]any{"Authorization": "Bearer " + token}}
}

func antigravityServers(path string) (map[string]any, map[string]any, error) {
	current := map[string]any{}
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	if err == nil {
		if err := json.Unmarshal(raw, &current); err != nil {
			return nil, nil, unreadable(path)
		}
	}
	servers, isMap := current["mcpServers"].(map[string]any)
	if _, held := current["mcpServers"]; held && !isMap {
		return nil, nil, unreadable(path)
	}
	if servers == nil {
		servers = map[string]any{}
	}
	return current, servers, nil
}

func unreadable(path string) error {
	return fmt.Errorf("%s is not an MCP servers file form.md can read, so it was left alone and the Antigravity tab was not opened", path)
}

func AntigravityFolder(dataDir, tab string) (string, error) {
	folder := filepath.Join(dataDir, "agent-tabs", "antigravity", tab)
	return folder, os.MkdirAll(folder, 0o700)
}

func AntigravityConfig(folder, engineURL, token string) (string, error) {
	path := antigravityFile(folder)
	current, servers, err := antigravityServers(path)
	if err != nil {
		return "", err
	}
	servers[TabServer] = antigravityEntry(engineURL, token)
	current["mcpServers"] = servers
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return path, writePrivate(dir, path, raw)
}

func AntigravityForget(folder, token string) {
	path := antigravityFile(folder)
	current, servers, err := antigravityServers(path)
	if err != nil {
		return
	}
	entry, _ := servers[TabServer].(map[string]any)
	headers, _ := entry["headers"].(map[string]any)
	if headers["Authorization"] != "Bearer "+token {
		return
	}
	delete(servers, TabServer)
	if len(servers) == 0 && len(current) == 1 {
		_ = os.Remove(path)
		_ = os.Remove(filepath.Dir(path))
		_ = os.Remove(folder)
		return
	}
	current["mcpServers"] = servers
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return
	}
	_ = writePrivate(filepath.Dir(path), path, raw)
}
