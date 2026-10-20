package templates

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type pendingDoc struct {
	id       string
	markdown string
}

type buildState struct {
	refs   map[string]string
	titles map[string]string
	rows   map[string]map[string]string
	docs   []pendingDoc
}

var placeholder = regexp.MustCompile(`\{\{(db|page):([A-Za-z0-9_-]+)\}\}`)

func (b *buildState) fill(md string) string {
	md = strings.ReplaceAll(fill(md), "```mermaid\n", "```mermaid preview\n")
	return placeholder.ReplaceAllStringFunc(md, func(m string) string {
		parts := placeholder.FindStringSubmatch(m)
		id, ok := b.refs[parts[2]]
		if !ok {
			return ""
		}
		if parts[1] == "db" {
			return "<!-- database:" + id + " -->"
		}
		label := strings.NewReplacer("[", "", "]", "").Replace(b.titles[parts[2]])
		return "[" + label + "](/p/" + id + ")"
	})
}

func (s *Service) build(ctx context.Context, p *policy.Principal, st *Starter, in UseInput, title string) (*page.Meta, error) {
	b := &buildState{refs: map[string]string{}, titles: map[string]string{}, rows: map[string]map[string]string{}}
	root, err := s.create(ctx, p, b, st, in.WorkspaceID, in.ParentID, in.TeamspaceID, title)
	if err != nil {
		return nil, err
	}
	for _, d := range b.docs {
		if err := s.docs.Write(ctx, p, d.id, doc.WriteInput{Markdown: b.fill(d.markdown), Mode: "replace"}); err != nil {
			return nil, err
		}
	}
	return root, nil
}

func (s *Service) create(ctx context.Context, p *policy.Principal, b *buildState, st *Starter, workspaceID, parentID, teamspaceID, title string) (*page.Meta, error) {
	create := page.CreateInput{WorkspaceID: workspaceID, ParentID: parentID, TeamspaceID: teamspaceID, Title: title, Icon: st.Icon, Kind: "doc"}
	if st.Kind == "database" {
		schema, err := st.schemaWith(b.refs)
		if err != nil {
			return nil, err
		}
		create.Kind = "collection"
		extra, _ := json.Marshal(map[string]any{"view": st.views[0].Type, "schema": schema})
		create.Extra = extra
	}
	m, err := s.pages.Create(ctx, p, create)
	if err != nil {
		return nil, err
	}
	if st.key != "" {
		b.refs[st.key] = m.ID
		b.titles[st.key] = title
	}
	if st.Kind == "database" {
		if err := s.fillDatabase(ctx, p, b, st, m.ID); err != nil {
			return nil, err
		}
	} else if st.markdown != "" {
		b.docs = append(b.docs, pendingDoc{m.ID, st.markdown})
	}
	for i := range st.children {
		c := &st.children[i]
		if _, err := s.create(ctx, p, b, c, workspaceID, m.ID, "", c.Name); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (s *Service) fillDatabase(ctx context.Context, p *policy.Principal, b *buildState, st *Starter, id string) error {
	out, err := s.cols.Get(ctx, p, id)
	if err != nil {
		return err
	}
	for i, v := range st.views {
		cfg := json.RawMessage(v.Config)
		name := v.Name
		typ := v.Type
		if i == 0 && len(out.Views) > 0 {
			if _, err := s.cols.UpdateView(ctx, p, out.Views[0].ID, collection.ViewInput{Name: &name, Type: &typ, Config: &cfg}); err != nil {
				return err
			}
			continue
		}
		if _, err := s.cols.CreateView(ctx, p, id, collection.ViewInput{Name: &name, Type: &typ, Config: &cfg}); err != nil {
			return err
		}
	}
	made := map[string]string{}
	if st.key != "" {
		b.rows[st.key] = made
	}
	var self []prop
	for _, sp := range st.schema {
		if k, ok := strings.CutPrefix(sp.Collection, "@"); ok && sp.Type == "relation" && (k == "self" || (st.key != "" && k == st.key)) {
			sp.Collection = "@self"
			self = append(self, sp)
		}
	}
	isSelf := func(pid string) bool {
		for _, sp := range self {
			if sp.ID == pid {
				return true
			}
		}
		return false
	}
	local := map[string]map[string]string{"self": made}
	for _, r := range st.rows {
		first := map[string]any{}
		for k, v := range r.Values {
			if !isSelf(k) {
				first[k] = v
			}
		}
		row, err := s.cols.CreateRow(ctx, p, id, collection.RowInput{Title: r.Title, Values: s.values(p, st.schema, first, b.rows)})
		if err != nil {
			return err
		}
		made[r.Title] = row.ID
		if r.Markdown != "" {
			b.docs = append(b.docs, pendingDoc{row.ID, r.Markdown})
		}
	}
	if len(self) > 0 {
		for _, r := range st.rows {
			later := map[string]any{}
			for _, sp := range self {
				if v, ok := r.Values[sp.ID]; ok {
					later[sp.ID] = v
				}
			}
			if len(later) == 0 {
				continue
			}
			if _, err := s.cols.UpdateRow(ctx, p, made[r.Title], collection.RowPatch{Values: s.values(p, self, later, local)}); err != nil {
				return err
			}
		}
	}
	for _, r := range st.templates {
		t, err := s.cols.CreateTemplate(ctx, p, id, r.Title)
		if err != nil {
			return err
		}
		if vals := s.values(p, st.schema, r.Values, b.rows); len(vals) > 0 {
			if _, err := s.cols.UpdateRow(ctx, p, t.ID, collection.RowPatch{Values: vals}); err != nil {
				return err
			}
		}
		if r.Markdown != "" {
			b.docs = append(b.docs, pendingDoc{t.ID, r.Markdown})
		}
	}
	return nil
}

func dateValue(v any, now time.Time) any {
	switch x := v.(type) {
	case string:
		if d, ok := dayOffset(x, now); ok {
			return map[string]string{"start": d}
		}
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			if str, ok := val.(string); ok && (k == "start" || k == "end") {
				if d, ok := dayOffset(str, now); ok {
					out[k] = d
					continue
				}
			}
			out[k] = val
		}
		return out
	}
	return v
}

func relationValue(v any, sp prop, rows map[string]map[string]string) any {
	key, ok := strings.CutPrefix(sp.Collection, "@")
	if !ok {
		return v
	}
	made := rows[key]
	var titles []string
	switch x := v.(type) {
	case string:
		titles = []string{x}
	case []string:
		titles = x
	}
	ids := []string{}
	for _, t := range titles {
		if id, ok := made[t]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}
