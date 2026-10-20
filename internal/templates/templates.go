package templates

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Service struct {
	st    *store.Store
	auth  *auth.Service
	gate  *policy.Gate
	pages *page.Service
	docs  *doc.Service
	cols  *collection.Service
}

func New(st *store.Store, a *auth.Service, gate *policy.Gate, pages *page.Service, docs *doc.Service, cols *collection.Service) *Service {
	return &Service{st: st, auth: a, gate: gate, pages: pages, docs: docs, cols: cols}
}

type WorkspaceTemplate struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Icon      string `json:"icon"`
	Kind      string `json:"kind"`
	UpdatedAt int64  `json:"updated_at"`
	CanEdit   bool   `json:"can_edit"`
}

type Gallery struct {
	Categories []Category          `json:"categories"`
	Starters   []Starter           `json:"starters"`
	Workspace  []WorkspaceTemplate `json:"workspace"`
}

func (s *Service) List(ctx context.Context, p *policy.Principal, workspaceID string) (*Gallery, error) {
	if _, err := s.gate.Workspace(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	out := &Gallery{Categories: categories, Starters: starters, Workspace: []WorkspaceTemplate{}}
	rows, err := s.st.R.QueryContext(ctx, "SELECT id FROM pages WHERE workspace_id = ? AND is_template = 1 AND is_row = 0 AND trashed_at = 0 ORDER BY updated_at DESC LIMIT 200", workspaceID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		m, canWrite, err := s.pages.ReadableMeta(ctx, p, id)
		if err != nil {
			continue
		}
		out.Workspace = append(out.Workspace, WorkspaceTemplate{ID: m.ID, Title: m.Title, Icon: m.Icon, Kind: m.Kind, UpdatedAt: m.UpdatedAt, CanEdit: canWrite})
	}
	return out, nil
}

type UseInput struct {
	WorkspaceID string `json:"workspace_id"`
	ParentID    string `json:"parent_id"`
	TeamspaceID string `json:"teamspace_id"`
	Starter     string `json:"starter"`
	Page        string `json:"page"`
	Title       string `json:"title"`
}

func dayOffset(v string, now time.Time) (string, bool) {
	if len(v) < 3 || (v[0] != '+' && v[0] != '-') || !strings.HasSuffix(v, "d") {
		return "", false
	}
	n, err := strconv.Atoi(strings.TrimSuffix(v, "d"))
	if err != nil {
		return "", false
	}
	return now.AddDate(0, 0, n).Format("2006-01-02"), true
}

func (s *Service) values(p *policy.Principal, schema []prop, in map[string]any) map[string]json.RawMessage {
	now := time.Now().UTC()
	out := map[string]json.RawMessage{}
	for _, sp := range schema {
		v, ok := in[sp.ID]
		if !ok || sp.Type == "title" {
			continue
		}
		switch sp.Type {
		case "person":
			if v == "@me" {
				v = []string{p.UserID}
			}
		case "date":
			if str, ok := v.(string); ok {
				if d, ok := dayOffset(str, now); ok {
					v = map[string]string{"start": d}
				}
			}
		}
		b, err := json.Marshal(v)
		if err == nil {
			out[sp.ID] = b
		}
	}
	return out
}

func fill(md string) string {
	return strings.ReplaceAll(md, "{{date}}", time.Now().UTC().Format("2006-01-02"))
}

func (s *Service) Use(ctx context.Context, p *policy.Principal, in UseInput) (*page.Meta, error) {
	if in.Page != "" {
		src, _, err := s.pages.ReadableMeta(ctx, p, in.Page)
		if err != nil {
			return nil, err
		}
		if !src.IsTemplate {
			return nil, httpx.Fail(http.StatusBadRequest, "not_template", "that page is not a template")
		}
		title := strings.TrimSpace(in.Title)
		if title == "" {
			title = src.Title
		}
		if strings.TrimSpace(title) == "" {
			title = "Untitled"
		}
		copied, err := s.pages.Duplicate(ctx, p, in.Page, page.DuplicateInput{Title: title})
		if err != nil {
			return nil, err
		}
		if in.ParentID != copied.ParentID || in.WorkspaceID != copied.WorkspaceID {
			parent := in.ParentID
			mv := page.MoveInput{WorkspaceID: in.WorkspaceID, ParentID: &parent}
			if in.TeamspaceID != "" && parent == "" {
				mv.TeamspaceID = &in.TeamspaceID
			}
			if moved, err := s.pages.Move(ctx, p, copied.ID, mv); err == nil {
				return moved, nil
			}
		}
		return s.pages.Load(ctx, copied.ID)
	}
	st := findStarter(in.Starter)
	if st == nil {
		return nil, httpx.Fail(http.StatusNotFound, "no_template", "no such template")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = st.Name
	}
	create := page.CreateInput{WorkspaceID: in.WorkspaceID, ParentID: in.ParentID, TeamspaceID: in.TeamspaceID, Title: title, Icon: st.Icon, Kind: "doc"}
	if st.Kind == "database" {
		create.Kind = "collection"
		extra, _ := json.Marshal(map[string]any{"view": st.views[0].Type, "schema": st.schemaJSON()})
		create.Extra = extra
	}
	m, err := s.pages.Create(ctx, p, create)
	if err != nil {
		return nil, err
	}
	if st.Kind == "doc" {
		if err := s.docs.Write(ctx, p, m.ID, doc.WriteInput{Markdown: fill(st.markdown), Mode: "replace"}); err != nil {
			return nil, err
		}
		return m, nil
	}
	out, err := s.cols.Get(ctx, p, m.ID)
	if err != nil {
		return nil, err
	}
	for i, v := range st.views {
		cfg := json.RawMessage(v.Config)
		name := v.Name
		typ := v.Type
		if i == 0 && len(out.Views) > 0 {
			if _, err := s.cols.UpdateView(ctx, p, out.Views[0].ID, collection.ViewInput{Name: &name, Type: &typ, Config: &cfg}); err != nil {
				return nil, err
			}
			continue
		}
		if _, err := s.cols.CreateView(ctx, p, m.ID, collection.ViewInput{Name: &name, Type: &typ, Config: &cfg}); err != nil {
			return nil, err
		}
	}
	for _, r := range st.rows {
		row, err := s.cols.CreateRow(ctx, p, m.ID, collection.RowInput{Title: r.Title, Values: s.values(p, st.schema, r.Values)})
		if err != nil {
			return nil, err
		}
		if r.Markdown != "" {
			if err := s.docs.Write(ctx, p, row.ID, doc.WriteInput{Markdown: fill(r.Markdown), Mode: "replace"}); err != nil {
				return nil, err
			}
		}
	}
	for _, r := range st.templates {
		t, err := s.cols.CreateTemplate(ctx, p, m.ID, r.Title)
		if err != nil {
			return nil, err
		}
		if vals := s.values(p, st.schema, r.Values); len(vals) > 0 {
			if _, err := s.cols.UpdateRow(ctx, p, t.ID, collection.RowPatch{Values: vals}); err != nil {
				return nil, err
			}
		}
		if r.Markdown != "" {
			if err := s.docs.Write(ctx, p, t.ID, doc.WriteInput{Markdown: fill(r.Markdown), Mode: "replace"}); err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "templates" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	r.HandleFunc("GET /api/templates", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		g, err := s.List(r.Context(), p, r.URL.Query().Get("workspace"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, g)
		return nil
	}))
	r.HandleFunc("GET /api/templates/{id}/preview", s.auth.Require(func(w http.ResponseWriter, r *http.Request, _ *policy.Principal) error {
		p := previewFor(r.PathValue("id"))
		if p == nil {
			return httpx.Fail(http.StatusNotFound, "no_template", "no such template")
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		httpx.JSON(w, http.StatusOK, p)
		return nil
	}))
	r.HandleFunc("POST /api/templates/use", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in UseInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		m, err := s.Use(r.Context(), p, in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"page": m})
		return nil
	}))
	return nil
}
