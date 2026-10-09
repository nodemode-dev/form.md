package mcp

import (
	"context"
	"encoding/json"

	"github.com/nodemode-ai/form.md/internal/policy"
)

const Instructions = instructions

type Spec struct {
	Name        string
	Title       string
	Description string
	Schema      map[string]any
	Write       bool
}

type Session struct {
	Name  string
	Kind  string
	Owner string
	RunID string
}

func (s *Server) Specs(p *policy.Principal) []Spec {
	out := make([]Spec, 0, len(s.tools))
	for _, t := range s.tools {
		if t.Write && !p.CanWrite() {
			continue
		}
		d := t.describe()
		schema, _ := d["inputSchema"].(map[string]any)
		if t.Write {
			if props, ok := schema["properties"].(map[string]any); ok {
				clean := map[string]any{}
				for k, v := range props {
					if k != "run_id" && k != "idempotency_key" {
						clean[k] = v
					}
				}
				schema["properties"] = clean
			}
		}
		out = append(out, Spec{Name: t.Name, Title: t.Title, Description: t.Description, Schema: schema, Write: t.Write})
	}
	return out
}

func (s *Server) Invoke(ctx context.Context, p *policy.Principal, sess *Session, name string, args json.RawMessage) (string, bool) {
	t := s.byName[name]
	if t == nil || (t.Write && !p.CanWrite()) {
		return "Unknown tool: " + name, true
	}
	var m map[string]any
	if len(args) == 0 || string(args) == "null" {
		m = map[string]any{}
	} else if err := json.Unmarshal(args, &m); err != nil || m == nil {
		return "arguments must be a JSON object", true
	}
	delete(m, "idempotency_key")
	delete(m, "run_id")
	if t.Write && sess.RunID != "" {
		m["run_id"] = sess.RunID
	}
	raw, _ := json.Marshal(m)
	pp := *p
	pp.ClientName = sess.Name
	c := &call{p: &pp, modern: true, client: sess.Name, nonce: newNonce(), kind: sess.Kind, owner: sess.Owner}
	res := s.callTool(ctx, c, t, raw)
	isErr, _ := res["isError"].(bool)
	text := ""
	if content, ok := res["content"].([]map[string]any); ok && len(content) > 0 {
		text, _ = content[0]["text"].(string)
	}
	if sc, ok := res["structuredContent"].(map[string]any); ok {
		if id, ok := sc["run_id"].(string); ok && id != "" {
			sess.RunID = id
		}
	}
	return text, isErr
}
