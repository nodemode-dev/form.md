package auditapi

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/settings"
)

type module struct {
	log  *audit.Log
	gate *policy.Gate
	auth *auth.Service
	set  *settings.Service
}

func Module(l *audit.Log, g *policy.Gate, a *auth.Service, set *settings.Service) app.Module {
	return module{l, g, a, set}
}

func (module) ID() string { return "auditapi" }

func (m module) admin(ctx context.Context, p *policy.Principal, workspaceID string) error {
	a, err := m.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.IsAdmin() {
		return httpx.Fail(http.StatusForbidden, "admin_only", "only workspace admins read the audit log")
	}
	return nil
}

func filter(r *http.Request) audit.Filter {
	q := r.URL.Query()
	num := func(k string) int64 {
		n, _ := strconv.ParseInt(q.Get(k), 10, 64)
		return n
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	return audit.Filter{WorkspaceID: r.PathValue("id"), Actor: q.Get("actor"), Action: q.Get("action"), PageID: q.Get("page"), From: num("from"), To: num("to"), Before: num("before"), Limit: limit}
}

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	route := func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, m.auth.Require(h))
	}
	route("GET /api/workspaces/{id}/audit", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := m.admin(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		f := filter(r)
		if f.Limit <= 0 || f.Limit > 200 {
			f.Limit = 50
		}
		if cur := r.URL.Query().Get("cursor"); cur != "" {
			c, err := httpx.DecodeCursor(cur)
			if err != nil {
				return err
			}
			f.Before = c.At
		}
		rows, next, err := m.log.Query(r.Context(), f)
		if err != nil {
			return err
		}
		out := map[string]any{"entries": rows, "next": next, "next_cursor": "", "retention_days": m.set.Int("audit.retention_days")}
		if next > 0 {
			out["next_cursor"] = httpx.EncodeCursor(httpx.Cursor{At: next})
		}
		if f.Before == 0 {
			actions, actors, err := m.log.Facets(r.Context(), f.WorkspaceID)
			if err != nil {
				return err
			}
			out["actions"], out["actors"] = actions, actors
		}
		httpx.JSON(w, http.StatusOK, out)
		return nil
	})
	route("GET /api/workspaces/{id}/audit.csv", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := m.admin(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		f := filter(r)
		f.Before, f.Limit = 0, 100000
		rows, _, err := m.log.Query(r.Context(), f)
		if err != nil {
			return err
		}
		h := w.Header()
		h.Set("Content-Type", "text/csv; charset=utf-8")
		h.Set("Content-Disposition", `attachment; filename="audit-log-`+time.Now().UTC().Format("2006-01-02")+`.csv"`)
		h.Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("\xef\xbb\xbf"))
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"id", "time (UTC)", "actor kind", "actor id", "actor", "agent", "action", "page id", "page", "detail", "changes"})
		for _, e := range rows {
			_ = cw.Write([]string{strconv.FormatInt(e.ID, 10), time.UnixMilli(e.At).UTC().Format(time.RFC3339), e.ActorKind, e.ActorID, safe(e.ActorLabel), safe(e.Agent), e.Action, e.PageID, safe(e.PageTitle), safe(e.Detail), safe(string(e.Changes))})
		}
		cw.Flush()
		return cw.Error()
	})
	return nil
}

func safe(v string) string {
	if v != "" && (v[0] == '=' || v[0] == '+' || v[0] == '-' || v[0] == '@' || v[0] == '\t' || v[0] == '\r') {
		return "'" + v
	}
	return v
}
