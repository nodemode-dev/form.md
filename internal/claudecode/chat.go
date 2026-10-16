package claudecode

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const chatNote = Note + " You are answering in form.md's Claude Code chat, shown as Markdown in a side panel: write short paragraphs, " +
	"use a bulleted list when you changed several things, name the pages you touched by title, and keep tool calls few."

const maxMessages = 200

var hiddenTools = map[string]bool{"ToolSearch": true}

type Part struct {
	Kind   string `json:"kind"`
	Text   string `json:"text,omitempty"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Input  string `json:"input,omitempty"`
	Status string `json:"status,omitempty"`
}

type Meta struct {
	Seconds      float64  `json:"seconds"`
	Turns        *int     `json:"turns,omitempty"`
	CostUSD      *float64 `json:"cost_usd,omitempty"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
}

type Message struct {
	Role  string `json:"role"`
	Text  string `json:"text,omitempty"`
	Note  string `json:"note,omitempty"`
	Parts []Part `json:"parts,omitempty"`
	Error string `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
}

type Usage struct {
	Input   int     `json:"input"`
	Output  int     `json:"output"`
	CostUSD float64 `json:"cost_usd"`
	Context int     `json:"context"`
	Window  int     `json:"window"`
	Model   string  `json:"model"`
}

type ChatState struct {
	Busy     bool      `json:"busy"`
	Session  string    `json:"session"`
	Messages []Message `json:"messages"`
	Usage    Usage     `json:"usage"`
}

type chat struct {
	mu        sync.Mutex
	state     ChatState
	proc      *exec.Cmd
	cancelled bool
}

type Chats struct {
	mu    sync.Mutex
	users map[string]*chat
}

func newChats() *Chats { return &Chats{users: map[string]*chat{}} }

func (c *Chats) of(user string) *chat {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := c.users[user]
	if ch == nil {
		ch = &chat{state: ChatState{Messages: []Message{}}}
		c.users[user] = ch
	}
	return ch
}

func (c *chat) snapshot() ChatState {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.state
	out.Messages = append([]Message(nil), c.state.Messages...)
	return out
}

func toolName(n string) string {
	if strings.HasPrefix(n, "mcp__") {
		parts := strings.Split(n, "__")
		return parts[len(parts)-1]
	}
	return n
}

func brief(v any) string {
	var s string
	if str, ok := v.(string); ok {
		s = str
	} else {
		raw, _ := json.Marshal(v)
		s = string(raw)
	}
	if r := []rune(s); len(r) > 160 {
		return string(r[:157]) + "..."
	}
	return s
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func usageOf(u map[string]any) (int, int) {
	return num(u["input_tokens"]) + num(u["cache_creation_input_tokens"]) + num(u["cache_read_input_tokens"]), num(u["output_tokens"])
}

func chatArgv(path, text, session, mcp string) []string {
	argv := []string{path, "-p", text, "--output-format", "stream-json", "--verbose", "--include-partial-messages",
		"--mcp-config", mcp, "--allowedTools", "mcp__formmd", "--append-system-prompt", chatNote}
	if session != "" {
		argv = append(argv, "--resume", session)
	}
	return argv
}

type emit func(kind string, data map[string]any)

func (c *chat) run(argv []string, dir string, env []string, send emit) {
	reply := Message{Role: "assistant", Parts: []Part{}}
	started := time.Now()
	var tail []string
	var result map[string]any
	doneOut, callOut := 0, 0
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir, cmd.Env = dir, env
	stdout, err := cmd.StdoutPipe()
	var stderr *bufio.Scanner
	if err == nil {
		var se interface{ Read([]byte) (int, error) }
		se, err = cmd.StderrPipe()
		if err == nil {
			stderr = bufio.NewScanner(se)
		}
	}
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		reply.Error = "Claude Code could not run: " + err.Error()
	} else {
		c.mu.Lock()
		c.proc = cmd
		c.mu.Unlock()
		var tailMu sync.Mutex
		go func() {
			for stderr.Scan() {
				tailMu.Lock()
				tail = append(tail, stderr.Text())
				if len(tail) > 15 {
					tail = tail[len(tail)-15:]
				}
				tailMu.Unlock()
			}
		}()
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		for sc.Scan() {
			var ev map[string]any
			if json.Unmarshal(sc.Bytes(), &ev) != nil || ev["parent_tool_use_id"] != nil {
				continue
			}
			switch ev["type"] {
			case "system":
				if ev["subtype"] == "init" {
					c.mu.Lock()
					if s, _ := ev["session_id"].(string); s != "" {
						c.state.Session = s
					}
					if m, _ := ev["model"].(string); m != "" {
						c.state.Usage.Model = m
					}
					c.mu.Unlock()
				}
			case "stream_event":
				e, _ := ev["event"].(map[string]any)
				d, _ := e["delta"].(map[string]any)
				switch e["type"] {
				case "message_start", "message_delta":
					var u map[string]any
					if e["type"] == "message_start" {
						msg, _ := e["message"].(map[string]any)
						u, _ = msg["usage"].(map[string]any)
					} else {
						u, _ = e["usage"].(map[string]any)
					}
					if u == nil {
						break
					}
					ctx, out := usageOf(u)
					c.mu.Lock()
					if e["type"] == "message_start" {
						doneOut += callOut
						callOut = 0
						if ctx > 0 {
							c.state.Usage.Context = ctx
						}
					} else {
						callOut = out
					}
					us := c.state.Usage
					c.mu.Unlock()
					send("usage", map[string]any{"context": us.Context, "window": us.Window, "output": doneOut + callOut, "model": us.Model})
				case "content_block_delta":
					if d["type"] == "text_delta" {
						if t, _ := d["text"].(string); t != "" {
							if n := len(reply.Parts); n == 0 || reply.Parts[n-1].Kind != "text" {
								reply.Parts = append(reply.Parts, Part{Kind: "text"})
							}
							reply.Parts[len(reply.Parts)-1].Text += t
							send("delta", map[string]any{"text": t})
						}
					}
				}
			case "assistant":
				msg, _ := ev["message"].(map[string]any)
				blocks, _ := msg["content"].([]any)
				for _, raw := range blocks {
					b, _ := raw.(map[string]any)
					name, _ := b["name"].(string)
					if b["type"] != "tool_use" || hiddenTools[name] {
						continue
					}
					id, _ := b["id"].(string)
					p := Part{Kind: "tool", ID: id, Name: toolName(name), Input: brief(b["input"]), Status: "running"}
					reply.Parts = append(reply.Parts, p)
					send("tool", map[string]any{"id": p.ID, "name": p.Name, "input": p.Input})
				}
			case "user":
				msg, _ := ev["message"].(map[string]any)
				blocks, _ := msg["content"].([]any)
				for _, raw := range blocks {
					b, _ := raw.(map[string]any)
					if b["type"] != "tool_result" {
						continue
					}
					id, _ := b["tool_use_id"].(string)
					status := "done"
					if isErr, _ := b["is_error"].(bool); isErr {
						status = "error"
					}
					for i := range reply.Parts {
						if reply.Parts[i].Kind == "tool" && reply.Parts[i].ID == id {
							reply.Parts[i].Status = status
							send("toolDone", map[string]any{"id": id, "status": status})
						}
					}
				}
			case "result":
				result = ev
				total, _ := ev["usage"].(map[string]any)
				in, out := usageOf(total)
				c.mu.Lock()
				if s, _ := ev["session_id"].(string); s != "" {
					c.state.Session = s
				}
				c.state.Usage.Input += in
				c.state.Usage.Output += out
				if cost, ok := ev["total_cost_usd"].(float64); ok {
					c.state.Usage.CostUSD += cost
				}
				if mu, ok := ev["modelUsage"].(map[string]any); ok {
					if per, ok := mu[c.state.Usage.Model].(map[string]any); ok {
						if w := num(per["contextWindow"]); w > 0 {
							c.state.Usage.Window = w
						}
					}
				}
				c.mu.Unlock()
			}
		}
		_ = cmd.Wait()
		tailMu.Lock()
		last := ""
		if len(tail) > 0 {
			last = tail[len(tail)-1]
		}
		tailMu.Unlock()
		c.mu.Lock()
		cancelled := c.cancelled
		c.proc = nil
		c.mu.Unlock()
		switch {
		case result == nil && cancelled:
			reply.Error = "Stopped"
		case result == nil:
			reply.Error = "Claude Code stopped before answering"
			if last != "" {
				reply.Error += ": " + last
			}
		case result["is_error"] == true:
			if r, _ := result["result"].(string); r != "" {
				reply.Error = r
			} else {
				reply.Error = "Claude Code reported an error"
			}
		}
	}
	meta := &Meta{Seconds: float64(time.Since(started).Milliseconds()/100) / 10}
	if result != nil {
		if t, ok := result["num_turns"].(float64); ok {
			n := int(t)
			meta.Turns = &n
		}
		if cost, ok := result["total_cost_usd"].(float64); ok {
			meta.CostUSD = &cost
		}
		u, _ := result["usage"].(map[string]any)
		meta.InputTokens, meta.OutputTokens = usageOf(u)
	}
	reply.Meta = meta
	for i := range reply.Parts {
		if reply.Parts[i].Status == "running" {
			reply.Parts[i].Status = "done"
		}
	}
	c.mu.Lock()
	c.state.Messages = append(c.state.Messages, reply)
	if n := len(c.state.Messages); n > maxMessages {
		c.state.Messages = c.state.Messages[n-maxMessages:]
	}
	c.state.Busy = false
	c.cancelled = false
	us := c.state.Usage
	c.mu.Unlock()
	send("done", map[string]any{"error": reply.Error, "meta": meta, "usage": us})
}

func (c *chat) cancel() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.proc != nil && c.proc.Process != nil {
		c.cancelled = true
		_ = c.proc.Process.Signal(os.Interrupt)
		p := c.proc.Process
		go func() {
			time.Sleep(3 * time.Second)
			_ = p.Kill()
		}()
		return true
	}
	return false
}

func projectDir(folder string) string {
	root := os.Getenv("CLAUDE_CONFIG_DIR")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".claude")
	}
	var b strings.Builder
	for _, r := range folder {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return filepath.Join(root, "projects", b.String())
}

type Conversation struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Modified int64  `json:"modified"`
	Current  bool   `json:"current"`
}

func transcript(path string) []Message {
	out := []Message{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 32<<20)
	for sc.Scan() {
		var ev map[string]any
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		kind, _ := ev["type"].(string)
		if (kind != "user" && kind != "assistant") || ev["isSidechain"] == true || ev["isMeta"] == true {
			continue
		}
		msg, _ := ev["message"].(map[string]any)
		var texts []string
		var tools []Part
		switch content := msg["content"].(type) {
		case string:
			texts = append(texts, content)
		case []any:
			for _, raw := range content {
				b, _ := raw.(map[string]any)
				switch b["type"] {
				case "text":
					if t, _ := b["text"].(string); t != "" {
						texts = append(texts, t)
					}
				case "tool_use":
					name, _ := b["name"].(string)
					if !hiddenTools[name] {
						id, _ := b["id"].(string)
						tools = append(tools, Part{Kind: "tool", ID: id, Name: toolName(name), Input: brief(b["input"]), Status: "done"})
					}
				}
			}
		}
		var keep []string
		for _, t := range texts {
			if !strings.HasPrefix(t, "<") {
				keep = append(keep, t)
			}
		}
		text := strings.Join(keep, "\n\n")
		if kind == "user" && text != "" {
			out = append(out, Message{Role: "user", Text: text, Note: "from Claude Code"})
		} else if kind == "assistant" && (text != "" || len(tools) > 0) {
			parts := tools
			if text != "" {
				parts = append(parts, Part{Kind: "text", Text: text})
			}
			if n := len(out); n > 0 && out[n-1].Role == "assistant" {
				out[n-1].Parts = append(out[n-1].Parts, parts...)
			} else {
				out = append(out, Message{Role: "assistant", Parts: parts})
			}
		}
	}
	if len(out) > maxMessages {
		out = out[len(out)-maxMessages:]
	}
	return out
}

func conversations(folder, current string) []Conversation {
	dir := projectDir(folder)
	matches, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	type item struct {
		path string
		mod  time.Time
	}
	var items []item
	for _, m := range matches {
		if st, err := os.Stat(m); err == nil {
			items = append(items, item{m, st.ModTime()})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.After(items[j].mod) })
	out := []Conversation{}
	for i, it := range items {
		if i >= 20 {
			break
		}
		id := strings.TrimSuffix(filepath.Base(it.path), ".jsonl")
		title := ""
		for _, m := range transcript(it.path) {
			if m.Role == "user" {
				title = m.Text
				break
			}
		}
		if r := []rune(title); len(r) > 90 {
			title = string(r[:90])
		}
		out = append(out, Conversation{ID: id, Title: title, Modified: it.mod.Unix(), Current: id == current})
	}
	return out
}
