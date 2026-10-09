package site

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/files"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Site struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	PageID      string `json:"page_id"`
	PageTitle   string `json:"page_title"`
	Slug        string `json:"slug"`
	Hostname    string `json:"hostname"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Theme       string `json:"theme"`
	Font        string `json:"font"`
	Password    bool   `json:"password"`
	Indexing    bool   `json:"indexing"`
	Subpages    bool   `json:"subpages"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	Path        string `json:"path"`

	passwordHash string
	passwordRev  int64
}

type ViewHook func(ctx context.Context, pageID, workspaceID, userID, visitor string)

type Service struct {
	st     *store.Store
	gate   *policy.Gate
	pages  *page.Service
	docs   *doc.Service
	cols   *collection.Service
	files  *files.Service
	auth   *auth.Service
	set    *settings.Service
	audit  *audit.Log
	log    *slog.Logger
	assets *assets
	unlock *httpx.Limiter
	onView ViewHook

	mu    sync.Mutex
	hosts map[string]string
	fresh time.Time
}

func New(st *store.Store, gate *policy.Gate, pages *page.Service, docs *doc.Service, cols *collection.Service, fs *files.Service, a *auth.Service, set *settings.Service, al *audit.Log, log *slog.Logger) *Service {
	return &Service{st: st, gate: gate, pages: pages, docs: docs, cols: cols, files: fs, auth: a, set: set, audit: al, log: log, unlock: httpx.NewLimiter(10, 10)}
}

func (s *Service) OnView(fn ViewHook) { s.onView = fn }

const siteColumns = `s.id, s.workspace_id, s.page_id, p.title, s.slug, s.hostname, s.title, s.description, s.image, s.theme, s.font,
	s.password_hash, s.password_rev, s.indexing, s.subpages, s.created_by, s.created_at, s.updated_at`

func scanSite(row interface{ Scan(...any) error }) (*Site, error) {
	var x Site
	var idx, sub int
	if err := row.Scan(&x.ID, &x.WorkspaceID, &x.PageID, &x.PageTitle, &x.Slug, &x.Hostname, &x.Title, &x.Description, &x.Image, &x.Theme, &x.Font,
		&x.passwordHash, &x.passwordRev, &idx, &sub, &x.CreatedBy, &x.CreatedAt, &x.UpdatedAt); err != nil {
		return nil, err
	}
	x.Password, x.Indexing, x.Subpages = x.passwordHash != "", idx == 1, sub == 1
	x.Path = "/s/" + x.Slug
	return &x, nil
}

func (s *Service) one(ctx context.Context, where string, arg any) (*Site, error) {
	x, err := scanSite(s.st.R.QueryRowContext(ctx, `SELECT `+siteColumns+` FROM sites s JOIN pages p ON p.id = s.page_id WHERE `+where, arg))
	if err != nil {
		return nil, store.NoRows(err)
	}
	return x, nil
}

func (s *Service) live(ctx context.Context, x *Site) bool {
	var on int
	var trashed int64
	err := s.st.R.QueryRowContext(ctx, `SELECT w.publishing, p.trashed_at FROM workspaces w JOIN pages p ON p.id = ? WHERE w.id = ?`, x.PageID, x.WorkspaceID).Scan(&on, &trashed)
	return err == nil && on == 1 && trashed == 0
}

func (s *Service) Publishing(ctx context.Context, workspaceID string) bool {
	var on int
	return s.st.R.QueryRowContext(ctx, "SELECT publishing FROM workspaces WHERE id = ?", workspaceID).Scan(&on) == nil && on == 1
}

type Status struct {
	Site       *Site  `json:"site"`
	CanPublish bool   `json:"can_publish"`
	Allowed    bool   `json:"allowed"`
	CanHost    bool   `json:"can_host"`
	Suggested  string `json:"suggested_slug"`
}

func (s *Service) Status(ctx context.Context, p *policy.Principal, pageID string) (*Status, error) {
	if err := s.pages.CanRead(ctx, p, pageID); err != nil {
		return nil, err
	}
	out := &Status{}
	d, err := s.pages.Get(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	out.Allowed = s.Publishing(ctx, d.WorkspaceID)
	if _, err := s.pages.CanShare(ctx, p, pageID); err == nil && p.CanWrite() {
		out.CanPublish = true
	}
	if a, err := s.gate.Workspace(ctx, p, d.WorkspaceID); err == nil && a.IsAdmin() {
		out.CanHost = true
	}
	if x, err := s.one(ctx, "s.page_id = ?", pageID); err == nil {
		out.Site = x
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	out.Suggested = s.suggest(ctx, d.Title)
	return out, nil
}

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,58}[a-z0-9])$`)

var reservedSlugs = map[string]bool{"api": true, "files": true, "assets": true, "admin": true, "login": true, "unlock": true}

func slugify(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
		if b.Len() >= 40 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

func (s *Service) suggest(ctx context.Context, title string) string {
	base := slugify(title)
	if len(base) < 3 {
		base = "page"
	}
	for i := 0; i < 8; i++ {
		cand := base
		if i > 0 || reservedSlugs[base] {
			cand = base + "-" + strings.ToLower(secrets.Token("", 3))
		}
		if !slugPattern.MatchString(cand) {
			continue
		}
		var n int
		if s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM sites WHERE slug = ?", cand).Scan(&n) == nil && n == 0 {
			return cand
		}
	}
	return "page-" + strings.ToLower(ids.New())[:10]
}

type Input struct {
	Slug          string  `json:"slug"`
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	Image         *string `json:"image"`
	Theme         *string `json:"theme"`
	Font          *string `json:"font"`
	Password      *string `json:"password"`
	Indexing      *bool   `json:"indexing"`
	Subpages      *bool   `json:"subpages"`
	Hostname      *string `json:"hostname"`
	RequestOrigin string  `json:"-"`
}

func pick[T any](v *T, cur T) T {
	if v != nil {
		return *v
	}
	return cur
}

func (s *Service) instanceHost() string {
	pub := s.set.Get("instance.public_url")
	_, rest, ok := strings.Cut(pub, "://")
	if !ok {
		return ""
	}
	host, _, _ := strings.Cut(rest, "/")
	return hostOnly(host)
}

func hostOnly(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	return strings.TrimSuffix(h, ".")
}

var hostPattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

func (s *Service) checkHost(h, origin string) (string, error) {
	h = hostOnly(h)
	if h == "" {
		return "", nil
	}
	_, rest, _ := strings.Cut(origin, "://")
	switch {
	case !hostPattern.MatchString(h), net.ParseIP(h) != nil, h == "localhost", strings.HasSuffix(h, ".localhost"):
		return "", httpx.Fail(http.StatusBadRequest, "bad_hostname", "use a domain name such as docs.example.com")
	case h == s.instanceHost(), h == hostOnly(rest):
		return "", httpx.Fail(http.StatusBadRequest, "bad_hostname", "this is the address of the workspace itself")
	}
	return h, nil
}

func (s *Service) Publish(ctx context.Context, p *policy.Principal, pageID string, in Input) (*Site, error) {
	ws, err := s.pages.CanShare(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	if !p.CanWrite() {
		return nil, policy.ErrHidden
	}
	if !s.Publishing(ctx, ws) {
		return nil, httpx.Fail(http.StatusForbidden, "publishing_off", "publishing to the web is turned off for this workspace")
	}
	cur, err := s.one(ctx, "s.page_id = ?", pageID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if in.Slug == "" {
		if cur != nil {
			in.Slug = cur.Slug
		} else {
			var title string
			_ = s.st.R.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = ?", pageID).Scan(&title)
			in.Slug = s.suggest(ctx, title)
		}
	}
	if !slugPattern.MatchString(in.Slug) || reservedSlugs[in.Slug] {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_slug", "use 3 to 60 lowercase letters, digits and dashes")
	}
	base := Site{Theme: "auto", Font: "sans", Subpages: true}
	if cur != nil {
		base = *cur
	}
	v := Site{
		Title:       clip(strings.TrimSpace(pick(in.Title, base.Title)), 120),
		Description: clip(strings.TrimSpace(pick(in.Description, base.Description)), 300),
		Image:       pick(in.Image, base.Image),
		Theme:       pick(in.Theme, base.Theme),
		Font:        pick(in.Font, base.Font),
		Indexing:    pick(in.Indexing, base.Indexing),
		Subpages:    pick(in.Subpages, base.Subpages),
	}
	if v.Theme != "auto" && v.Theme != "light" && v.Theme != "dark" {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_theme", "theme is auto, light or dark")
	}
	if v.Font != "sans" && v.Font != "serif" && v.Font != "mono" {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_font", "font is sans, serif or mono")
	}
	if in.Image != nil && v.Image != "" && v.Image != base.Image {
		var n int
		fid, ok := strings.CutPrefix(v.Image, "f:")
		if !ok || s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE id = ? AND workspace_id = ? AND mime LIKE 'image/%'", fid, ws).Scan(&n) != nil || n == 0 {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_image", "the preview image must be a picture in this workspace")
		}
	}
	host := ""
	if cur != nil {
		host = cur.Hostname
	}
	if in.Hostname != nil && hostOnly(*in.Hostname) != host {
		a, err := s.gate.Workspace(ctx, p, ws)
		if err != nil || !a.IsAdmin() {
			return nil, httpx.Fail(http.StatusForbidden, "admin_only", "only workspace admins set a custom domain")
		}
		if host, err = s.checkHost(*in.Hostname, in.RequestOrigin); err != nil {
			return nil, err
		}
	}
	hash, rev := "", int64(0)
	if cur != nil {
		hash, rev = cur.passwordHash, cur.passwordRev
	}
	if in.Password != nil {
		rev++
		hash = ""
		if *in.Password != "" {
			if len(*in.Password) < 4 {
				return nil, httpx.Fail(http.StatusBadRequest, "short_password", "use at least 4 characters")
			}
			if hash, err = auth.HashPassword(ctx, *in.Password); err != nil {
				return nil, err
			}
		}
	}
	now := store.NowMS()
	id := ids.New()
	if cur != nil {
		id = cur.ID
	}
	action := "site.publish"
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		var err error
		if cur == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO sites (id, workspace_id, page_id, slug, hostname, title, description, image, theme, font, password_hash, password_rev, indexing, subpages, created_by, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, ws, pageID, in.Slug, host, v.Title, v.Description, v.Image, v.Theme, v.Font, hash, rev, b2i(v.Indexing), b2i(v.Subpages), p.UserID, now, now)
		} else {
			action = "site.update"
			_, err = tx.ExecContext(ctx, `UPDATE sites SET slug = ?, hostname = ?, title = ?, description = ?, image = ?, theme = ?, font = ?, password_hash = ?, password_rev = ?, indexing = ?, subpages = ?, updated_at = ? WHERE id = ?`,
				in.Slug, host, v.Title, v.Description, v.Image, v.Theme, v.Font, hash, rev, b2i(v.Indexing), b2i(v.Subpages), now, id)
		}
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				if strings.Contains(err.Error(), "hostname") {
					return httpx.Fail(http.StatusConflict, "hostname_taken", "another site already uses this domain")
				}
				return httpx.Fail(http.StatusConflict, "slug_taken", "this address is taken")
			}
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: action, WorkspaceID: ws, PageID: pageID, Detail: in.Slug})
	})
	if err != nil {
		return nil, err
	}
	s.forgetHosts()
	return s.one(ctx, "s.id = ?", id)
}

func (s *Service) Unpublish(ctx context.Context, p *policy.Principal, pageID string) error {
	x, err := s.one(ctx, "s.page_id = ?", pageID)
	if err != nil {
		return err
	}
	if _, err := s.pages.CanShare(ctx, p, pageID); err != nil {
		a, aerr := s.gate.Workspace(ctx, p, x.WorkspaceID)
		if aerr != nil || !a.IsAdmin() {
			return err
		}
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM sites WHERE id = ?", x.ID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "site.unpublish", WorkspaceID: x.WorkspaceID, PageID: pageID, Detail: x.Slug})
	})
	s.forgetHosts()
	return err
}

func (s *Service) List(ctx context.Context, p *policy.Principal, workspaceID string) ([]*Site, bool, bool, error) {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return nil, false, false, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT `+siteColumns+` FROM sites s JOIN pages p ON p.id = s.page_id WHERE s.workspace_id = ? AND p.trashed_at = 0 ORDER BY s.created_at DESC`, workspaceID)
	if err != nil {
		return nil, false, false, err
	}
	var all []*Site
	for rows.Next() {
		x, err := scanSite(rows)
		if err != nil {
			rows.Close()
			return nil, false, false, err
		}
		all = append(all, x)
	}
	rows.Close()
	out := []*Site{}
	for _, x := range all {
		if a.IsAdmin() || s.pages.CanRead(ctx, p, x.PageID) == nil {
			out = append(out, x)
		}
	}
	return out, s.Publishing(ctx, workspaceID), a.IsAdmin(), nil
}

func (s *Service) SetPublishing(ctx context.Context, p *policy.Principal, workspaceID string, on bool) error {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.IsAdmin() {
		return httpx.Fail(http.StatusForbidden, "admin_only", "only workspace admins change this")
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE workspaces SET publishing = ? WHERE id = ?", b2i(on), workspaceID); err != nil {
			return err
		}
		detail := "off"
		if on {
			detail = "on"
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.publishing", WorkspaceID: workspaceID, Detail: detail})
	})
	s.forgetHosts()
	return err
}

func (s *Service) forgetHosts() {
	s.mu.Lock()
	s.hosts = nil
	s.mu.Unlock()
}

func (s *Service) siteForHost(ctx context.Context, host string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hosts == nil || time.Since(s.fresh) > time.Minute {
		s.hosts = map[string]string{}
		s.fresh = time.Now()
		rows, err := s.st.R.QueryContext(ctx, "SELECT hostname, id FROM sites WHERE hostname != ''")
		if err == nil {
			for rows.Next() {
				var h, id string
				if rows.Scan(&h, &id) == nil {
					s.hosts[h] = id
				}
			}
			rows.Close()
		}
	}
	return s.hosts[host]
}

func clip(v string, n int) string {
	r := []rune(v)
	if len(r) > n {
		return string(r[:n])
	}
	return v
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
