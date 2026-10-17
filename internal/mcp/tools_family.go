package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nodemode-ai/form.md/internal/family"
)

type Family interface {
	List(ctx context.Context) []family.Member
	Route(ctx context.Context, task, selfBase string) []family.Route
}

func (s *Server) registerFamily() {
	if s.Family == nil {
		return
	}
	s.add(&Tool{Name: "family_list", Title: "List the other form family apps", Brief: true,
		Description: "The other form family apps that can run beside form.md on this computer (formlit for photos, formvis for layered image editing): what each is for, whether it is running, its version, its MCP endpoint, how many tools it has and how an agent connects to it. form.md does not call their tools; connect to them directly.",
		Props:       map[string]any{},
		Run: func(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
			list := s.Family.List(ctx)
			up := 0
			for _, m := range list {
				if m.Reachable {
					up++
				}
			}
			return map[string]any{"members": list}, fmt.Sprintf("%d of %d family apps are running", up, len(list)), nil
		}})
	s.add(&Tool{Name: "family_route", Title: "Find the form family app for a task", Brief: true,
		Description: "Describe a task in plain words and get the form family apps ranked by fit (form.md itself included), each with its MCP endpoint, how to connect and, for apps that are running, the tools of theirs that match the task. Use it before doing work that is about photos or picture editing rather than documents and databases.",
		Props:       map[string]any{"task": str("What needs doing, in plain words")}, Required: []string{"task"},
		Run: func(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
			var in struct {
				Task string `json:"task"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, "", err
			}
			if in.Task == "" {
				return nil, "", fail("describe the task")
			}
			routes := s.Family.Route(ctx, in.Task, s.Auth.BaseURL(x.c.r))
			best := routes[0]
			return map[string]any{"routes": routes}, fmt.Sprintf("Best fit: %s", best.Product), nil
		}})
}
