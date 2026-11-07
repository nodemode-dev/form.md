package mcp

import (
	"context"
	"encoding/json"

	"github.com/nodemode-ai/form.md/internal/policy"
)

type Window interface {
	CatalogueFor(p *policy.Principal, filter string) any
	DescribeFor(p *policy.Principal, id string) (any, bool)
	RunCommand(ctx context.Context, p *policy.Principal, id string, params map[string]any) (json.RawMessage, error)
}

func (s *Server) registerWindow() {
	if s.Window == nil {
		return
	}
	s.add(&Tool{Name: "window_commands", Title: "List the open window's commands", Brief: true,
		Description: "The command catalogue of the form.md window this account has open: every menu, toolbar and keyboard action with its id, group, description, parameters and whether it changes anything. Use it to drive the window the way the person would (open a page, switch a view, toggle focus mode, run a canvas tool). Empty when no window is open.",
		Props:       map[string]any{"query": str("Words to filter by id, label, group or description")},
		Run: func(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
			var in struct {
				Query string `json:"query"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, "", err
			}
			return map[string]any{"commands": s.Window.CatalogueFor(x.p, in.Query)}, "", nil
		}})
	s.add(&Tool{Name: "window_command_describe", Title: "Describe a window command", Brief: true,
		Description: "One command of the open window in full: its parameters with types and limits, keys, whether it changes anything and whether undo can reverse it.",
		Props:       map[string]any{"id": str("Command id from window_commands")}, Required: []string{"id"},
		Run: func(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
			var in struct {
				ID string `json:"id"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, "", err
			}
			c, ok := s.Window.DescribeFor(x.p, in.ID)
			if !ok {
				return nil, "", fail("no command " + in.ID + " in the open window; call window_commands first")
			}
			return c, "", nil
		}})
	s.add(&Tool{Name: "window_command_run", Title: "Run a window command",
		Description: "Runs one command in the form.md window this account has open, exactly as if the person chose it, and returns its result. The person sees it happen. Commands that change the workspace need a write credential; risky ones (marked risky) cannot be undone.",
		Props:       map[string]any{"id": str("Command id from window_commands"), "params": obj(map[string]any{})}, Required: []string{"id"},
		Run: func(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
			var in struct {
				ID     string         `json:"id"`
				Params map[string]any `json:"params"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, "", err
			}
			out, err := s.Window.RunCommand(ctx, x.p, in.ID, in.Params)
			if err != nil {
				return nil, "", fail(err.Error())
			}
			var v any
			if json.Unmarshal(out, &v) != nil {
				v = string(out)
			}
			return v, "", nil
		}})
	s.add(&Tool{Name: "extensions_list", Title: "List the window's extensions", Brief: true,
		Description: "The Form extensions installed for form.md (from FormPortal, or a folder run from form.ext) as the open window sees them: id, name, version, source, state (ready, needs-permission or refused, with the reason) and the commands each contributes. Empty when none is installed.",
		Run: func(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
			out, err := s.Window.RunCommand(ctx, x.p, "extensions.list", map[string]any{})
			if err != nil {
				return nil, "", fail(err.Error())
			}
			var v any
			if json.Unmarshal(out, &v) != nil {
				v = string(out)
			}
			return v, "", nil
		}})
	s.add(&Tool{Name: "extension_command", Title: "Run an extension command",
		Description: "Runs a command an extension contributes, in the open form.md window, exactly as if the person chose it from the Extensions menu. The window asks the person to allow the extension first when this version has not been allowed yet; a refusal fails the call. Returns what the command reports when it finishes.",
		Props:       map[string]any{"id": str("Extension id from extensions_list"), "command": str("Command id from the extension's commands"), "args": obj(map[string]any{})}, Required: []string{"id", "command"},
		Run: func(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
			var in struct {
				ID      string         `json:"id"`
				Command string         `json:"command"`
				Args    map[string]any `json:"args"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, "", err
			}
			if in.Args == nil {
				in.Args = map[string]any{}
			}
			out, err := s.Window.RunCommand(ctx, x.p, "extensions.run", map[string]any{"id": in.ID, "command": in.Command, "args": in.Args})
			if err != nil {
				return nil, "", fail(err.Error())
			}
			var v any
			if json.Unmarshal(out, &v) != nil {
				v = string(out)
			}
			return v, "", nil
		}})
}
