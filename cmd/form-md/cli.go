package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func cliUsage(w io.Writer) {
	fmt.Fprintln(w, `form-md cli: the command catalogue of the form.md window you have open

  form-md cli list [text]           every command, or those matching text
  form-md cli describe ID           one command with its parameters
  form-md cli run ID [key=value...] run a command in the open window; values are JSON when they parse, text otherwise

Uses `+bridgeURLEnv+` (default http://127.0.0.1:8750) and `+bridgeTokenEnv+` (a personal API token from Account settings).`)
}

func cliCall(method, path string, body any) (map[string]any, error) {
	base := strings.TrimRight(os.Getenv(bridgeURLEnv), "/")
	if base == "" {
		base = "http://127.0.0.1:8750"
	}
	base = strings.TrimSuffix(base, "/mcp")
	token := os.Getenv(bridgeTokenEnv)
	if token == "" {
		return nil, fmt.Errorf("set %s to a personal API token (Account settings, API tokens)", bridgeTokenEnv)
	}
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, base+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var out map[string]any
	raw, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	if res.StatusCode >= 300 {
		if e, ok := out["error"].(map[string]any); ok {
			return nil, fmt.Errorf("%v", e["message"])
		}
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return out, nil
}

func cli(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		cliUsage(os.Stdout)
		return 0
	}
	fail := func(err error) int {
		fmt.Fprintln(os.Stderr, "form-md cli:", err)
		return 1
	}
	show := func(v any) {
		raw, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(raw))
	}
	switch args[0] {
	case "list":
		q := ""
		if len(args) > 1 {
			q = "?q=" + url.QueryEscape(strings.Join(args[1:], " "))
		}
		out, err := cliCall("GET", "/api/commands"+q, nil)
		if err != nil {
			return fail(err)
		}
		list, _ := out["commands"].([]any)
		if len(list) == 0 {
			fmt.Println("no commands: open form.md in a browser or the desktop app, signed in as the token's owner")
			return 0
		}
		for _, raw := range list {
			c, _ := raw.(map[string]any)
			mark := " "
			if c["mutates"] == true {
				mark = "*"
			}
			fmt.Printf("%s %-36s %-14s %v\n", mark, c["id"], c["group"], c["label"])
		}
		fmt.Println("\n* changes the workspace")
	case "describe":
		if len(args) < 2 {
			cliUsage(os.Stderr)
			return 2
		}
		out, err := cliCall("GET", "/api/commands/"+url.PathEscape(args[1]), nil)
		if err != nil {
			return fail(err)
		}
		show(out)
	case "run":
		if len(args) < 2 {
			cliUsage(os.Stderr)
			return 2
		}
		params := map[string]any{}
		for _, kv := range args[2:] {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				return fail(fmt.Errorf("%q is not key=value", kv))
			}
			var parsed any
			if json.Unmarshal([]byte(v), &parsed) == nil {
				params[k] = parsed
			} else if n, err := strconv.ParseFloat(v, 64); err == nil {
				params[k] = n
			} else {
				params[k] = v
			}
		}
		out, err := cliCall("POST", "/api/commands/run", map[string]any{"id": args[1], "params": params})
		if err != nil {
			return fail(err)
		}
		show(out)
	default:
		cliUsage(os.Stderr)
		return 2
	}
	return 0
}
