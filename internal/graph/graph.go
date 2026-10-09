package graph

import (
	"context"
	"net/http"
	"strings"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const maxNodes = 3000

type Node struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Kind  string `json:"kind"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

type Graph struct {
	Nodes     []Node `json:"nodes"`
	Edges     []Edge `json:"edges"`
	Truncated bool   `json:"truncated,omitempty"`
}

type Service struct {
	st    *store.Store
	auth  *auth.Service
	pages *page.Service
}

func New(st *store.Store, a *auth.Service, pages *page.Service) *Service {
	return &Service{st: st, auth: a, pages: pages}
}

func (s *Service) Build(ctx context.Context, p *policy.Principal, workspaceID string) (*Graph, error) {
	list, err := s.pages.Tree(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	g := &Graph{Nodes: []Node{}, Edges: []Edge{}}
	seen := map[string]bool{}
	for _, m := range list {
		if len(g.Nodes) >= maxNodes {
			g.Truncated = true
			break
		}
		seen[m.ID] = true
		g.Nodes = append(g.Nodes, Node{ID: m.ID, Title: m.Title, Icon: m.Icon, Kind: m.Kind})
	}
	for _, m := range list {
		if seen[m.ID] && m.ParentID != "" && seen[m.ParentID] {
			g.Edges = append(g.Edges, Edge{From: m.ParentID, To: m.ID, Kind: "child"})
		}
	}
	if len(g.Nodes) == 0 {
		return g, nil
	}
	ids := make([]any, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		ids = append(ids, n.ID)
	}
	for start := 0; start < len(ids); start += 500 {
		part := ids[start:min(start+500, len(ids))]
		marks := strings.TrimSuffix(strings.Repeat("?,", len(part)), ",")
		rows, err := s.st.R.QueryContext(ctx, "SELECT page_id, kind, target FROM doc_refs WHERE page_id IN ("+marks+")", part...)
		if err != nil {
			return nil, err
		}
		dup := map[string]bool{}
		for rows.Next() {
			var from, kind, to string
			if rows.Scan(&from, &kind, &to) != nil || from == to || !seen[to] {
				continue
			}
			k := from + "|" + to
			if dup[k] {
				continue
			}
			dup[k] = true
			edge := "link"
			if kind != "page" {
				edge = kind
			}
			g.Edges = append(g.Edges, Edge{From: from, To: to, Kind: edge})
		}
		rows.Close()
	}
	return g, nil
}

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "graph" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	r.HandleFunc("GET /api/workspaces/{id}/graph", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		g, err := s.Build(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, g)
		return nil
	}))
	return nil
}
