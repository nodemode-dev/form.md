package automation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/netguard"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Env struct {
	WorkspaceID  string
	CollectionID string
	AutomationID string
	Name         string
	RowID        string
	PageID       string
	ActorID      string
	ActorName    string
	CreatorID    string
	RunAs        *policy.Principal
	Base         string
	Trigger      string
}

type Outcome struct {
	Done    []string `json:"done"`
	Failed  []string `json:"failed"`
	Created []string `json:"created,omitempty"`
}

func (o *Outcome) status() string {
	switch {
	case len(o.Failed) == 0:
		return "ok"
	case len(o.Done) == 0:
		return "failed"
	}
	return "partial"
}

func (o *Outcome) detail() string {
	parts := append([]string{}, o.Done...)
	for _, f := range o.Failed {
		parts = append(parts, "failed: "+f)
	}
	return clip(strings.Join(parts, "; "), 1000)
}

var placeholder = regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

func (s *Service) url(env Env, id string) string {
	base := strings.TrimRight(env.Base, "/")
	if s.Public != nil {
		if pub := strings.TrimRight(s.Public(), "/"); pub != "" {
			base = pub
		}
	}
	return base + "/p/" + id
}

func (s *Service) render(tpl string, env Env, row *collection.RowView) string {
	if !strings.Contains(tpl, "{{") {
		return tpl
	}
	now := time.Now().UTC()
	return placeholder.ReplaceAllStringFunc(tpl, func(m string) string {
		key := strings.ToLower(strings.TrimSpace(placeholder.FindStringSubmatch(m)[1]))
		switch key {
		case "title", "page":
			if row != nil {
				return row.Title
			}
			return ""
		case "url", "link":
			if env.RowID != "" {
				return s.url(env, env.RowID)
			}
			if env.PageID != "" {
				return s.url(env, env.PageID)
			}
			return ""
		case "user", "me", "actor":
			return env.ActorName
		case "date", "today":
			return now.Format("2006-01-02")
		case "now", "time":
			return now.Format("Jan 2, 2006 15:04 UTC")
		case "automation", "button":
			return env.Name
		}
		if row != nil {
			for _, p := range row.Schema {
				if strings.ToLower(p.Name) == key {
					return row.Text[p.ID]
				}
			}
		}
		return m
	})
}

func idList(v any) []string {
	switch x := v.(type) {
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	case []string:
		return x
	case json.RawMessage:
		var list []string
		if json.Unmarshal(x, &list) == nil {
			return list
		}
		var one string
		if json.Unmarshal(x, &one) == nil && one != "" {
			return []string{one}
		}
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func number(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case json.RawMessage:
		f, _ := strconv.ParseFloat(strings.TrimSpace(string(x)), 64)
		return f
	}
	return 0
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case json.RawMessage:
		return strings.TrimSpace(string(x)) == "true"
	}
	return false
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func (s *Service) values(env Env, schema collection.Schema, in map[string]json.RawMessage, trigger, current *collection.RowView) (map[string]json.RawMessage, *string) {
	out := map[string]json.RawMessage{}
	var title *string
	now := time.Now().UTC()
	for id, raw := range in {
		p := schema.ByID(id)
		if p == nil {
			continue
		}
		var token string
		_ = json.Unmarshal(raw, &token)
		var inc struct {
			Inc *float64 `json:"inc"`
		}
		_ = json.Unmarshal(raw, &inc)
		var val json.RawMessage
		switch {
		case token == "@clear":
			val = json.RawMessage("null")
		case token == "@now" && p.Type == collection.TDate:
			val = mustJSON(map[string]string{"start": now.Format("2006-01-02T15:04")})
		case token == "@today" && p.Type == collection.TDate:
			val = mustJSON(map[string]string{"start": now.Format("2006-01-02")})
		case (token == "@actor" || token == "@me") && p.Type == collection.TPerson:
			if env.ActorID == "" {
				continue
			}
			val = mustJSON([]string{env.ActorID})
		case token == "@row" && p.Type == collection.TRelation:
			if env.RowID == "" {
				continue
			}
			val = mustJSON([]string{env.RowID})
		case token == "@toggle" && p.Type == collection.TCheckbox:
			was := false
			if current != nil {
				was = truthy(current.Values[id])
			}
			val = mustJSON(!was)
		case inc.Inc != nil && p.Type == collection.TNumber:
			base := 0.0
			if current != nil {
				base = number(current.Values[id])
			}
			val = mustJSON(base + *inc.Inc)
		case token != "" && (p.Type == collection.TTitle || p.Type == collection.TText || p.Type == collection.TURL || p.Type == collection.TEmail || p.Type == collection.TPhone):
			val = mustJSON(s.render(token, env, trigger))
		default:
			val = raw
		}
		if p.Type == collection.TTitle {
			var t string
			_ = json.Unmarshal(val, &t)
			title = &t
			continue
		}
		out[id] = val
	}
	return out, title
}

func (s *Service) people(ctx context.Context, env Env, tokens []string, row *collection.RowView) []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, t := range tokens {
		switch {
		case t == "@actor" || t == "@me":
			add(env.ActorID)
		case t == "@creator":
			add(env.CreatorID)
		case strings.HasPrefix(t, "prop:"):
			if row != nil {
				for _, id := range idList(row.Values[strings.TrimPrefix(t, "prop:")]) {
					add(id)
				}
			}
		default:
			add(t)
		}
	}
	var members []string
	for _, id := range out {
		var one int
		if s.Store.R.QueryRowContext(ctx, "SELECT 1 FROM workspace_members WHERE workspace_id = ? AND user_id = ?", env.WorkspaceID, id).Scan(&one) == nil {
			members = append(members, id)
		}
	}
	return members
}

func (s *Service) execute(ctx context.Context, env Env, actions []Action) Outcome {
	var o Outcome
	var trigger *collection.RowView
	if env.RowID != "" {
		if r, err := s.Cols.ReadRow(ctx, env.RunAs, env.RowID); err == nil {
			trigger = r
		}
	}
	for i, a := range actions {
		label := fmt.Sprintf("%d %s", i+1, a.Kind)
		msg, err := s.one(ctx, env, a, trigger, &o)
		if err != nil {
			o.Failed = append(o.Failed, label+": "+clip(reason(err), 200))
			continue
		}
		if msg != "" {
			o.Done = append(o.Done, msg)
		}
		if a.Kind == "edit" && a.Target == "trigger" && env.RowID != "" {
			if r, err := s.Cols.ReadRow(ctx, env.RunAs, env.RowID); err == nil {
				trigger = r
			}
		}
	}
	return o
}

func (s *Service) one(ctx context.Context, env Env, a Action, trigger *collection.RowView, o *Outcome) (string, error) {
	p := env.RunAs
	switch a.Kind {
	case "edit":
		var targets []string
		colID := a.Collection
		if a.Target == "all" {
			list, err := s.Cols.RowsMatching(ctx, p, a.Collection, a.Filter, maxEditRows)
			if err != nil {
				return "", err
			}
			targets = list
		} else {
			if env.RowID == "" || trigger == nil {
				return "", failure("there is no page to edit")
			}
			targets, colID = []string{env.RowID}, trigger.CollectionID
		}
		schema, err := s.Cols.SchemaOf(ctx, colID)
		if err != nil {
			return "", err
		}
		n := 0
		for _, id := range targets {
			cur := trigger
			if id != env.RowID || cur == nil {
				if cur, err = s.Cols.ReadRow(ctx, p, id); err != nil {
					continue
				}
			}
			vals, title := s.values(env, schema, a.Values, trigger, cur)
			if _, err := s.Cols.UpdateRow(ctx, p, id, collection.RowPatch{Values: vals, Title: title}); err != nil {
				if len(targets) == 1 {
					return "", err
				}
				continue
			}
			n++
		}
		return fmt.Sprintf("edited %d page(s)", n), nil
	case "add":
		schema, err := s.Cols.SchemaOf(ctx, a.Collection)
		if err != nil {
			return "", err
		}
		vals, title := s.values(env, schema, a.Values, trigger, nil)
		in := collection.RowInput{Values: vals, Title: s.render(a.Title, env, trigger)}
		if title != nil {
			in.Title = *title
		}
		row, err := s.Cols.CreateRow(ctx, p, a.Collection, in)
		if err != nil {
			return "", err
		}
		o.Created = append(o.Created, row.ID)
		return "added " + row.Title, nil
	case "notify":
		target := env.RowID
		if target == "" {
			target = env.PageID
		}
		if target == "" {
			target = env.CollectionID
		}
		users := s.people(ctx, env, a.People, trigger)
		n := s.Inbox.FromAutomation(ctx, env.Name, target, s.render(a.Message, env, trigger), users)
		return fmt.Sprintf("notified %d", n), nil
	case "mail":
		users := s.people(ctx, env, a.People, trigger)
		subject := s.render(a.Subject, env, trigger)
		body := s.render(a.Message, env, trigger)
		link := ""
		if env.RowID != "" {
			link = s.url(env, env.RowID)
		} else if env.PageID != "" {
			link = s.url(env, env.PageID)
		}
		if link != "" {
			body = strings.TrimSpace(body) + "\n\n" + link
		}
		body += "\n\nSent by " + env.Name + " from form.md."
		sent := 0
		var last error
		for _, u := range users {
			var email string
			if s.Store.R.QueryRowContext(ctx, "SELECT email FROM users WHERE id = ?", u).Scan(&email) != nil || email == "" {
				continue
			}
			if err := s.Mail.Send(ctx, email, subject, body); err != nil {
				last = err
				continue
			}
			sent++
		}
		if sent == 0 && last != nil {
			return "", last
		}
		return fmt.Sprintf("mailed %d", sent), nil
	case "webhook":
		key, err := s.key(ctx, env.WorkspaceID)
		if err != nil {
			return "", err
		}
		payload := map[string]any{
			"type":         "automation.run",
			"at":           store.NowMS(),
			"workspace_id": env.WorkspaceID,
			"trigger":      env.Trigger,
			"source":       map[string]string{"id": env.AutomationID, "name": env.Name},
			"actor":        map[string]string{"id": env.ActorID, "name": env.ActorName},
		}
		if env.CollectionID != "" {
			payload["collection_id"] = env.CollectionID
		}
		if env.PageID != "" {
			payload["page_id"] = env.PageID
		}
		if trigger != nil {
			props := map[string]string{}
			for _, sp := range trigger.Schema {
				if t, ok := trigger.Text[sp.ID]; ok {
					props[sp.Name] = t
				}
			}
			payload["row"] = map[string]any{"id": trigger.ID, "title": trigger.Title, "url": s.url(env, trigger.ID), "properties": props}
		}
		body, _ := json.Marshal(payload)
		code, err := s.Hooks.Send(ctx, a.URL, "automation.run", key, body)
		if err != nil {
			return "", err
		}
		if code < 200 || code >= 300 {
			return "", failure(fmt.Sprintf("the address answered HTTP %d", code))
		}
		return fmt.Sprintf("posted to %s (HTTP %d)", hostOf(a.URL), code), nil
	}
	return "", nil
}

type failure string

func (f failure) Error() string { return string(f) }

func reason(err error) string {
	var f failure
	if errors.As(err, &f) {
		return string(f)
	}
	var he *httpx.APIError
	if errors.As(err, &he) {
		return he.Message
	}
	var ne net.Error
	if errors.Is(err, netguard.ErrBlocked) || errors.Is(err, netguard.ErrURL) || errors.As(err, &ne) {
		_, msg := netguard.Reason(err)
		return msg
	}
	return err.Error()
}

func hostOf(raw string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return s
}
