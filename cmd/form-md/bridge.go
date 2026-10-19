package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/claudecode"
)

const (
	bridgeURLEnv   = "FORMMD_MCP_URL"
	bridgeTokenEnv = "FORMMD_MCP_TOKEN"
	bridgeMax      = 4 << 20
)

type bridgeOut struct {
	mu sync.Mutex
	w  *bufio.Writer
}

func (o *bridgeOut) line(b []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.w.Write(b)
	o.w.WriteByte('\n')
	o.w.Flush()
}

func (o *bridgeOut) fail(id json.RawMessage, code int, msg string) {
	if len(id) == 0 || string(id) == "null" {
		return
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": msg}})
	o.line(b)
}

func headerValue(v string) string {
	for _, r := range v {
		if r < 0x20 || r > 0x7e {
			return "=?base64?" + base64.StdEncoding.EncodeToString([]byte(v)) + "?="
		}
	}
	return v
}

var (
	startOnce sync.Once
	started   bool
)

func autostart(endpoint string) bool {
	startOnce.Do(func() { started = launchServer(endpoint) })
	return started
}

func launchServer(endpoint string) bool {
	if v := strings.ToLower(os.Getenv("FORMMD_MCP_AUTOSTART")); v == "0" || v == "off" || v == "false" {
		return false
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" || !claudecode.Local(u.Host) {
		return false
	}
	data := os.Getenv("FORMMD_DATA")
	if data == "" {
		if st, err := os.Stat("data"); err != nil || !st.IsDir() {
			return false
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "form-md-server.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return false
	}
	defer logFile.Close()
	fmt.Fprintf(os.Stderr, "form-md mcp: starting the server (form-md serve); its log: %s\n", logFile.Name())
	cmd := exec.Command(exe, "serve")
	cmd.Env = append(os.Environ(), "FORMMD_ADDR="+u.Host)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	detach(cmd)
	if cmd.Start() != nil {
		return false
	}
	go func() { _ = cmd.Wait() }()
	probe := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 120; i++ {
		time.Sleep(500 * time.Millisecond)
		res, err := probe.Get(u.Scheme + "://" + u.Host + "/healthz")
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return true
			}
		}
	}
	return false
}

func bridge(args []string) int {
	fl := flag.NewFlagSet("mcp", flag.ContinueOnError)
	base := fl.String("url", os.Getenv(bridgeURLEnv), "form.md address, for example https://notes.example.com (default "+bridgeURLEnv+" or http://127.0.0.1:8750)")
	token := fl.String("token", os.Getenv(bridgeTokenEnv), "personal API token (default "+bridgeTokenEnv+")")
	if err := fl.Parse(args); err != nil {
		return 2
	}
	if *base == "" {
		*base = "http://127.0.0.1:8750"
	}
	u, err := url.Parse(strings.TrimRight(*base, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		fmt.Fprintf(os.Stderr, "form-md mcp: %q is not an http or https address\n", *base)
		return 2
	}
	if !strings.HasSuffix(u.Path, "/mcp") {
		u.Path += "/mcp"
	}
	endpoint := u.String()
	if *token == "" {
		fmt.Fprintln(os.Stderr, "form-md mcp: no token; set "+bridgeTokenEnv+" or pass --token, or every call will be refused")
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	out := &bridgeOut{w: bufio.NewWriter(os.Stdout)}
	in := bufio.NewReaderSize(os.Stdin, 64<<10)
	var wg sync.WaitGroup
	slots := make(chan struct{}, 4)
	for {
		raw, err := in.ReadBytes('\n')
		if msg := bytes.TrimSpace(raw); len(msg) > 0 {
			if len(msg) > bridgeMax {
				fmt.Fprintln(os.Stderr, "form-md mcp: dropped a message over 4 MB")
			} else {
				slots <- struct{}{}
				wg.Add(1)
				go func(msg []byte) {
					defer func() { <-slots; wg.Done() }()
					forward(client, endpoint, *token, msg, out)
				}(bytes.Clone(msg))
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintln(os.Stderr, "form-md mcp: read:", err)
			}
			break
		}
	}
	wg.Wait()
	return 0
}

func forward(client *http.Client, endpoint, token string, msg []byte, out *bridgeOut) {
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name string                     `json:"name"`
			URI  string                     `json:"uri"`
			Meta map[string]json.RawMessage `json:"_meta"`
		} `json:"params"`
	}
	if err := json.Unmarshal(msg, &req); err != nil {
		b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": nil, "error": map[string]any{"code": -32700, "message": "not a JSON-RPC message"}})
		out.line(b)
		return
	}
	hr, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(msg))
	if err != nil {
		out.fail(req.ID, -32603, err.Error())
		return
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Accept", "application/json, text/event-stream")
	hr.Header.Set("User-Agent", "form-md-mcp-bridge")
	if token != "" {
		hr.Header.Set("Authorization", "Bearer "+token)
	}
	if raw, ok := req.Params.Meta["io.modelcontextprotocol/protocolVersion"]; ok {
		var v string
		_ = json.Unmarshal(raw, &v)
		hr.Header.Set("MCP-Protocol-Version", v)
		hr.Header.Set("Mcp-Method", req.Method)
		switch req.Method {
		case "tools/call", "prompts/get":
			hr.Header.Set("Mcp-Name", headerValue(req.Params.Name))
		case "resources/read":
			hr.Header.Set("Mcp-Name", headerValue(req.Params.URI))
		}
	}
	resp, err := client.Do(hr)
	if err != nil && autostart(endpoint) {
		if body, e := hr.GetBody(); e == nil {
			hr.Body = body
			resp, err = client.Do(hr)
		}
	}
	if err != nil {
		out.fail(req.ID, -32603, "form.md is not reachable at "+endpoint+": "+err.Error())
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		out.fail(req.ID, -32603, "the answer from form.md was cut off")
		return
	}
	if resp.StatusCode == http.StatusAccepted || len(bytes.TrimSpace(body)) == 0 {
		if resp.StatusCode >= 400 {
			out.fail(req.ID, -32603, fmt.Sprintf("form.md answered %d", resp.StatusCode))
		}
		return
	}
	var compact bytes.Buffer
	if json.Compact(&compact, body) != nil {
		msg := fmt.Sprintf("form.md answered %d with something that is not JSON", resp.StatusCode)
		if resp.StatusCode == http.StatusUnauthorized {
			msg = "form.md refused the token (401); set " + bridgeTokenEnv + " to a personal API token from Account settings"
		}
		out.fail(req.ID, -32603, msg)
		return
	}
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Fprintln(os.Stderr, "form-md mcp: form.md refused the token (401); set "+bridgeTokenEnv+" to a personal API token from Account settings")
	}
	out.line(compact.Bytes())
}
