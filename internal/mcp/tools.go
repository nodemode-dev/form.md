package mcp

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Tool struct {
	Name        string
	Title       string
	Description string
	Props       map[string]any
	Required    []string
	Output      map[string]any
	Write       bool
	Destructive bool
	Idempotent  bool
	Run         func(ctx context.Context, x *exec, args json.RawMessage) (any, string, error)
}

type exec struct {
	s      *Server
	c      *call
	p      *policy.Principal
	run    string
	name   string
	tool   string
	agent  *doc.Agent
	opened bool
	want   string
	kind   string
	owner  string
}

func (t *Tool) describe() map[string]any {
	props := map[string]any{}
	for k, v := range t.Props {
		props[k] = v
	}
	if t.Write {
		props["run_id"] = map[string]any{"type": "string", "description": "Group this write with earlier writes of the same run (from a previous result). Omit to start a new run."}
		props["idempotency_key"] = map[string]any{"type": "string", "description": "Any unique string; a retry with the same key returns the first result instead of writing twice (kept for 24 hours)."}
	}
	in := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(t.Required) > 0 {
		in["required"] = t.Required
	}
	ann := map[string]any{"title": t.Title, "readOnlyHint": !t.Write, "openWorldHint": false}
	if t.Write {
		ann["destructiveHint"] = t.Destructive
		ann["idempotentHint"] = t.Idempotent
	}
	d := map[string]any{"name": t.Name, "title": t.Title, "description": t.Description, "inputSchema": in, "annotations": ann}
	if t.Output != nil {
		d["outputSchema"] = t.Output
	}
	return d
}

func (s *Server) add(t *Tool) {
	s.tools = append(s.tools, t)
	s.byName[t.Name] = t
}

func (x *exec) openRun(ctx context.Context) (string, error) {
	if x.opened {
		return x.run, nil
	}
	kind := x.kind
	if kind == "" {
		kind = "mcp"
	}
	id, err := x.s.Runs.Open(ctx, x.p, x.want, kind, x.name, x.owner, x.tool)
	if err != nil {
		return "", err
	}
	x.run, x.opened = id, true
	x.p.AgentRun = id
	return id, nil
}

func (x *exec) record(ctx context.Context, kind, pageID, summary string, undo any) error {
	if _, err := x.openRun(ctx); err != nil {
		return err
	}
	return x.s.Runs.Record(ctx, x.run, kind, pageID, summary, undo)
}

func (x *exec) frame(label, text string) string {
	return "[untrusted " + label + " begin nonce=" + x.c.nonce + "]\n" + text + "\n[untrusted " + label + " end nonce=" + x.c.nonce + "]"
}

func argsHash(raw json.RawMessage) string {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	delete(m, "idempotency_key")
	b, _ := json.Marshal(m)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func toolResult(structured any, text string, isErr bool) map[string]any {
	out := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}, "isError": isErr}
	if structured != nil {
		out["structuredContent"] = structured
	}
	return out
}

func (s *Server) callTool(ctx context.Context, c *call, t *Tool, raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage("{}")
	}
	var meta struct {
		RunID string `json:"run_id"`
		Key   string `json:"idempotency_key"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return toolResult(nil, "arguments must be a JSON object", true)
	}
	key := c.p.CredentialID
	if key == "" {
		key = c.p.UserID
	}
	if !s.limit.Allow("mcp:" + key) {
		return toolResult(nil, "too many tool calls; wait a minute and try again", true)
	}
	pp := *c.p
	x := &exec{s: s, c: c, p: &pp, tool: t.Name, want: meta.RunID, kind: c.kind, owner: c.owner}
	x.name = s.displayName(ctx, c)
	pp.AgentName = x.name
	x.agent = &doc.Agent{Key: key, Name: x.name}
	hash := ""
	if t.Write && meta.Key != "" {
		if len(meta.Key) > 200 {
			return toolResult(nil, "idempotency_key can be up to 200 characters", true)
		}
		hash = argsHash(raw)
		var tool, h, stored string
		err := s.Store.R.QueryRowContext(ctx, "SELECT tool, args_hash, result FROM mcp_idempotency WHERE credential = ? AND key = ?", key, meta.Key).Scan(&tool, &h, &stored)
		if err == nil {
			if tool != t.Name || h != hash {
				return toolResult(nil, "that idempotency_key was already used with different arguments", true)
			}
			var prev map[string]any
			if json.Unmarshal([]byte(stored), &prev) == nil {
				return prev
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return toolResult(nil, errText(err), true)
		}
	}
	if t.Write && meta.RunID != "" {
		if _, err := x.openRun(ctx); err != nil {
			return toolResult(nil, errText(err), true)
		}
	}
	structured, summary, err := t.Run(ctx, x, raw)
	if err != nil {
		s.Log.Debug("mcp: tool failed", "tool", t.Name, "err", err)
		return toolResult(nil, errText(err), true)
	}
	if t.Write && x.run != "" {
		if m, ok := structured.(map[string]any); ok {
			m["run_id"] = x.run
		}
		summary += "\nrun_id: " + x.run
	}
	body, _ := json.MarshalIndent(structured, "", "  ")
	text := summary
	if structured != nil {
		text = strings.TrimSpace(summary + "\n" + x.frame("result", string(body)))
	}
	res := toolResult(structured, text, false)
	if hash != "" {
		if b, err := json.Marshal(res); err == nil {
			_, _ = s.Store.W.ExecContext(ctx, "INSERT OR IGNORE INTO mcp_idempotency (credential, key, tool, args_hash, result, created_at) VALUES (?, ?, ?, ?, ?, ?)", key, meta.Key, t.Name, hash, string(b), store.NowMS())
		}
	}
	return res
}

func decode(raw json.RawMessage, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	if err := dec.Decode(v); err != nil {
		return fail("arguments do not match the tool's input schema: " + err.Error())
	}
	return nil
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func intp(desc string) map[string]any { return map[string]any{"type": "integer", "description": desc} }

func boolp(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }

func strs(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

func enum(desc string, values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values, "description": desc}
}

func obj(props map[string]any, required ...string) map[string]any {
	o := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		o["required"] = required
	}
	return o
}
