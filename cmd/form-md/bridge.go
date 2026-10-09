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
	"strings"
	"sync"
	"time"
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
