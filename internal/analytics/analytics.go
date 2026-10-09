package analytics

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	dayMS     = int64(24 * time.Hour / time.Millisecond)
	repeatGap = int64(10 * time.Minute / time.Millisecond)
	keepDays  = 730
)

type Service struct {
	st    *store.Store
	gate  *policy.Gate
	pages *page.Service
	audit *audit.Log
	log   *slog.Logger
}

func New(st *store.Store, gate *policy.Gate, pages *page.Service, al *audit.Log, log *slog.Logger) *Service {
	return &Service{st: st, gate: gate, pages: pages, audit: al, log: log}
}

func anon(prefix, v string) string {
	sum := sha256.Sum256([]byte(prefix + v))
	return prefix + hex.EncodeToString(sum[:10])
}

func (s *Service) records(ctx context.Context, userID, workspaceID string) bool {
	var mine, ws int = 1, 1
	_ = s.st.R.QueryRowContext(ctx, "SELECT record FROM analytics_prefs WHERE user_id = ?", userID).Scan(&mine)
	_ = s.st.R.QueryRowContext(ctx, "SELECT view_history FROM workspaces WHERE id = ?", workspaceID).Scan(&ws)
	return mine == 1 && ws == 1
}

func (s *Service) Record(ctx context.Context, pageID, workspaceID, userID, visitor string) {
	var viewer string
	switch {
	case userID != "" && s.records(ctx, userID, workspaceID):
		viewer = userID
	case userID != "":
		viewer = anon("h:", userID)
	case visitor != "":
		viewer = "v:" + visitor
	default:
		return
	}
	now := store.NowMS()
	_, err := s.st.W.ExecContext(ctx, `INSERT INTO page_views (page_id, workspace_id, day, viewer, views, first_at, last_at) VALUES (?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT (page_id, day, viewer) DO UPDATE SET views = page_views.views + CASE WHEN excluded.last_at - page_views.last_at >= ? THEN 1 ELSE 0 END, last_at = excluded.last_at`,
		pageID, workspaceID, now/dayMS, viewer, now, now, repeatGap)
	if err != nil {
		s.log.Warn("analytics: record", "page", pageID, "err", err)
	}
}

func (s *Service) FromVisit(ctx context.Context, p *policy.Principal, pageID, workspaceID string) {
	s.Record(ctx, pageID, workspaceID, p.UserID, "")
}

func (s *Service) Purge(ctx context.Context) {
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM page_views WHERE day < ?", store.NowMS()/dayMS-keepDays)
}

type Person struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	At    int64  `json:"at"`
	Views int    `json:"views,omitempty"`
}

type Day struct {
	Day     string `json:"day"`
	Views   int    `json:"views"`
	Viewers int    `json:"viewers"`
}

type PageReport struct {
	Days      int      `json:"days"`
	Series    []Day    `json:"series"`
	Views     int      `json:"views"`
	Viewers   int      `json:"viewers"`
	Anonymous int      `json:"anonymous"`
	CreatedAt int64    `json:"created_at"`
	CreatedBy *Person  `json:"created_by"`
	UpdatedAt int64    `json:"updated_at"`
	UpdatedBy *Person  `json:"updated_by"`
	Editors   []Person `json:"editors"`
	People    []Person `json:"people"`
	History   bool     `json:"history"`
}

func clampDays(d int) int {
	switch {
	case d <= 0:
		return 28
	case d > 365:
		return 365
	}
	return d
}

type viewRow struct {
	page   string
	day    int64
	viewer string
	views  int
	last   int64
}

func (s *Service) series(rows []viewRow, days int) ([]Day, int, int) {
	today := store.NowMS() / dayMS
	first := today - int64(days) + 1
	views := make([]int, days)
	seen := make([]map[string]bool, days)
	all := map[string]bool{}
	total := 0
	for _, r := range rows {
		i := int(r.day - first)
		if i < 0 || i >= days {
			continue
		}
		views[i] += r.views
		if seen[i] == nil {
			seen[i] = map[string]bool{}
		}
		seen[i][r.viewer] = true
		all[r.viewer] = true
		total += r.views
	}
	out := make([]Day, days)
	for i := range out {
		out[i] = Day{Day: time.UnixMilli((first + int64(i)) * dayMS).UTC().Format("2006-01-02"), Views: views[i], Viewers: len(seen[i])}
	}
	return out, total, len(all)
}

func (s *Service) person(ctx context.Context, id string, at int64) *Person {
	if id == "" {
		return nil
	}
	p := &Person{ID: id, At: at}
	_ = s.st.R.QueryRowContext(ctx, "SELECT name, email FROM users WHERE id = ?", id).Scan(&p.Name, &p.Email)
	if p.Name == "" && p.Email == "" {
		return nil
	}
	return p
}

func (s *Service) Page(ctx context.Context, p *policy.Principal, pageID string, days int) (*PageReport, error) {
	ws, err := s.pages.CanEdit(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	days = clampDays(days)
	first := store.NowMS()/dayMS - int64(days) + 1
	q, err := s.st.R.QueryContext(ctx, "SELECT day, viewer, views, last_at FROM page_views WHERE page_id = ? AND day >= ?", pageID, first)
	if err != nil {
		return nil, err
	}
	var rows []viewRow
	for q.Next() {
		var r viewRow
		if err := q.Scan(&r.day, &r.viewer, &r.views, &r.last); err != nil {
			q.Close()
			return nil, err
		}
		rows = append(rows, r)
	}
	q.Close()
	out := &PageReport{Days: days, Editors: []Person{}, People: []Person{}}
	out.Series, out.Views, out.Viewers = s.series(rows, days)
	var vh int
	_ = s.st.R.QueryRowContext(ctx, "SELECT view_history FROM workspaces WHERE id = ?", ws).Scan(&vh)
	out.History = vh == 1
	people := map[string]*Person{}
	for _, r := range rows {
		if strings.HasPrefix(r.viewer, "v:") {
			out.Anonymous += r.views
		}
		if strings.Contains(r.viewer, ":") {
			continue
		}
		pp := people[r.viewer]
		if pp == nil {
			if pp = s.person(ctx, r.viewer, 0); pp == nil {
				continue
			}
			people[r.viewer] = pp
		}
		pp.Views += r.views
		if r.last > pp.At {
			pp.At = r.last
		}
	}
	if out.History {
		for _, pp := range people {
			out.People = append(out.People, *pp)
		}
		sort.Slice(out.People, func(i, j int) bool { return out.People[i].At > out.People[j].At })
	}
	var createdBy, updatedBy string
	_ = s.st.R.QueryRowContext(ctx, "SELECT created_at, created_by, updated_at, updated_by FROM pages WHERE id = ?", pageID).Scan(&out.CreatedAt, &createdBy, &out.UpdatedAt, &updatedBy)
	out.CreatedBy = s.person(ctx, createdBy, out.CreatedAt)
	out.UpdatedBy = s.person(ctx, updatedBy, out.UpdatedAt)
	editors := map[string]int64{}
	if updatedBy != "" {
		editors[updatedBy] = out.UpdatedAt
	}
	vq, err := s.st.R.QueryContext(ctx, "SELECT created_at, editors FROM page_versions WHERE page_id = ? AND created_at >= ?", pageID, first*dayMS)
	if err == nil {
		for vq.Next() {
			var at int64
			var ed string
			if vq.Scan(&at, &ed) != nil {
				continue
			}
			var ids []string
			_ = json.Unmarshal([]byte(ed), &ids)
			for _, id := range ids {
				if at > editors[id] {
					editors[id] = at
				}
			}
		}
		vq.Close()
	}
	for id, at := range editors {
		if pp := s.person(ctx, id, at); pp != nil {
			out.Editors = append(out.Editors, *pp)
		}
	}
	sort.Slice(out.Editors, func(i, j int) bool { return out.Editors[i].At > out.Editors[j].At })
	return out, nil
}

type TopPage struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Icon    string `json:"icon"`
	Kind    string `json:"kind"`
	Views   int    `json:"views"`
	Viewers int    `json:"viewers"`
}

type WorkspaceReport struct {
	Days    int       `json:"days"`
	Series  []Day     `json:"series"`
	Views   int       `json:"views"`
	Viewers int       `json:"viewers"`
	Pages   int       `json:"pages"`
	Top     []TopPage `json:"top"`
	History bool      `json:"history"`
}

func (s *Service) Workspace(ctx context.Context, p *policy.Principal, workspaceID string, days int) (*WorkspaceReport, error) {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	if !a.IsAdmin() {
		return nil, httpx.Fail(http.StatusForbidden, "admin_only", "only workspace admins see workspace analytics")
	}
	days = clampDays(days)
	first := store.NowMS()/dayMS - int64(days) + 1
	q, err := s.st.R.QueryContext(ctx, "SELECT v.page_id, v.day, v.viewer, v.views, v.last_at FROM page_views v JOIN pages p ON p.id = v.page_id WHERE v.workspace_id = ? AND v.day >= ? AND p.trashed_at = 0", workspaceID, first)
	if err != nil {
		return nil, err
	}
	var rows []viewRow
	for q.Next() {
		var r viewRow
		if err := q.Scan(&r.page, &r.day, &r.viewer, &r.views, &r.last); err != nil {
			q.Close()
			return nil, err
		}
		rows = append(rows, r)
	}
	q.Close()
	out := &WorkspaceReport{Days: days, Top: []TopPage{}}
	out.Series, out.Views, out.Viewers = s.series(rows, days)
	var vh int
	_ = s.st.R.QueryRowContext(ctx, "SELECT view_history FROM workspaces WHERE id = ?", workspaceID).Scan(&vh)
	out.History = vh == 1
	byPage := map[string]*TopPage{}
	viewers := map[string]map[string]bool{}
	for _, r := range rows {
		t := byPage[r.page]
		if t == nil {
			t = &TopPage{ID: r.page}
			byPage[r.page] = t
			viewers[r.page] = map[string]bool{}
		}
		t.Views += r.views
		viewers[r.page][r.viewer] = true
	}
	list := make([]*TopPage, 0, len(byPage))
	for id, t := range byPage {
		t.Viewers = len(viewers[id])
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Views != list[j].Views {
			return list[i].Views > list[j].Views
		}
		return list[i].ID < list[j].ID
	})
	for _, t := range list {
		if s.pages.CanRead(ctx, p, t.ID) != nil {
			continue
		}
		out.Pages++
		if len(out.Top) < 20 {
			_ = s.st.R.QueryRowContext(ctx, "SELECT title, icon, kind FROM pages WHERE id = ?", t.ID).Scan(&t.Title, &t.Icon, &t.Kind)
			out.Top = append(out.Top, *t)
		}
	}
	return out, nil
}

func (s *Service) Prefs(ctx context.Context, p *policy.Principal) bool {
	rec := 1
	_ = s.st.R.QueryRowContext(ctx, "SELECT record FROM analytics_prefs WHERE user_id = ?", p.UserID).Scan(&rec)
	return rec == 1
}

func (s *Service) SetPrefs(ctx context.Context, p *policy.Principal, record bool) error {
	if !p.IsSession() {
		return policy.ErrHidden
	}
	_, err := s.st.W.ExecContext(ctx, `INSERT INTO analytics_prefs (user_id, record, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET record = excluded.record, updated_at = excluded.updated_at`, p.UserID, store.Bool(record), store.NowMS())
	return err
}

func (s *Service) SetHistory(ctx context.Context, p *policy.Principal, workspaceID string, on bool) error {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.IsAdmin() {
		return httpx.Fail(http.StatusForbidden, "admin_only", "only workspace admins change this")
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE workspaces SET view_history = ? WHERE id = ?", store.Bool(on), workspaceID); err != nil {
			return err
		}
		detail := "off"
		if on {
			detail = "on"
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.view_history", WorkspaceID: workspaceID, Detail: detail})
	})
}

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module(a *auth.Service) app.Module { return module{s, a} }

func (module) ID() string { return "analytics" }

func days(r *http.Request) int {
	n, _ := strconv.Atoi(r.URL.Query().Get("days"))
	return n
}

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	route := func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, a.Require(h))
	}
	route("GET /api/pages/{id}/analytics", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		rep, err := s.Page(r.Context(), p, r.PathValue("id"), days(r))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, rep)
		return nil
	})
	route("GET /api/workspaces/{id}/analytics", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		rep, err := s.Workspace(r.Context(), p, r.PathValue("id"), days(r))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, rep)
		return nil
	})
	route("PUT /api/workspaces/{id}/view-history", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Enabled bool `json:"enabled"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetHistory(r.Context(), p, r.PathValue("id"), in.Enabled); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	route("GET /api/me/analytics", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		httpx.JSON(w, http.StatusOK, map[string]any{"record": s.Prefs(r.Context(), p)})
		return nil
	})
	route("PUT /api/me/analytics", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Record bool `json:"record"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetPrefs(r.Context(), p, in.Record); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	return nil
}
