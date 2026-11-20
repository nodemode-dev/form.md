package claudecode

import (
	"encoding/json"
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
			return nil, nil, err
		}
	}
	servers, _ := current["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	return current, servers, nil
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
	current["mcpServers"] = servers
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return
	}
	_ = writePrivate(filepath.Dir(path), path, raw)
}
