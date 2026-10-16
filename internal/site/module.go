package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/files"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module(a *auth.Service) app.Module { return module{s, a} }

func (module) ID() string { return "site" }

func (m module) Mount(r *httpx.Router, d *app.Deps) error {
	s, a := m.s, m.auth
	s.assets = loadAssets(d.Dist)
	if s.set.Get("sites.key") == "" {
		if err := s.set.Set(context.Background(), map[string]string{"sites.key": secrets.Token("", 32)}); err != nil {
			return err
		}
	}
	route := func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, a.Require(h))
	}
	route("GET /api/pages/{id}/site", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		st, err := s.Status(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, st)
		return nil
	})
	route("PUT /api/pages/{id}/site", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Input
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		in.RequestOrigin = httpx.RequestOrigin(r, a.TrustProxy())
		x, err := s.Publish(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"site": x})
		return nil
	})
	route("DELETE /api/pages/{id}/site", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Unpublish(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	route("GET /api/workspaces/{id}/sites", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, on, admin, err := s.List(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"sites": list, "publishing": on, "can_manage": admin})
		return nil
	})
	route("PUT /api/workspaces/{id}/publishing", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Enabled bool `json:"enabled"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetPublishing(r.Context(), p, r.PathValue("id"), in.Enabled); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})

	site := func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.siteScope(r, r.PathValue("slug"), "/s/"+r.PathValue("slug"))
		s.serve(w, r, sc, err, r.PathValue("page"))
	}
	r.HandleFunc("GET /s/{slug}", site)
	r.HandleFunc("GET /s/{slug}/{$}", site)
	r.HandleFunc("GET /s/{slug}/{page}", site)
	r.HandleFunc("GET /s/{slug}/files/{file}", func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.siteScope(r, r.PathValue("slug"), "/s/"+r.PathValue("slug"))
		s.file(w, r, sc, err, r.PathValue("file"))
	})
	r.HandleFunc("POST /s/{slug}/unlock", func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.siteScope(r, r.PathValue("slug"), "/s/"+r.PathValue("slug"))
		s.unlockPost(w, r, sc, err)
	})
	r.HandleFunc("GET /s/{slug}/fonts.css", func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.siteScope(r, r.PathValue("slug"), "/s/"+r.PathValue("slug"))
		s.fontsSheet(w, r, sc, err)
	})
	r.HandleFunc("GET /l/{token}/fonts.css", func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.linkScope(r, r.PathValue("token"))
		s.fontsSheet(w, r, sc, err)
	})
	r.HandleFunc("GET /s/{slug}/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.siteScope(r, r.PathValue("slug"), "/s/"+r.PathValue("slug"))
		s.sitemap(w, r, sc, err)
	})
	link := func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.linkScope(r, r.PathValue("token"))
		if err == nil && sc.link != nil && policy.From(r.Context()).IsSession() && s.unlocked(r, sc) {
			target := r.PathValue("page")
			if _, ok := sc.pages[target]; !ok {
				target = sc.root
			}
			if gerr := s.pages.GrantLink(r.Context(), policy.From(r.Context()), sc.link); gerr == nil {
				http.Redirect(w, r, "/p/"+target, http.StatusFound)
				return
			}
		}
		s.serve(w, r, sc, err, r.PathValue("page"))
	}
	r.HandleFunc("GET /l/{token}", link)
	r.HandleFunc("GET /l/{token}/{$}", link)
	r.HandleFunc("GET /l/{token}/{page}", link)
	r.HandleFunc("GET /l/{token}/files/{file}", func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.linkScope(r, r.PathValue("token"))
		s.file(w, r, sc, err, r.PathValue("file"))
	})
	r.HandleFunc("POST /l/{token}/unlock", func(w http.ResponseWriter, r *http.Request) {
		sc, err := s.linkScope(r, r.PathValue("token"))
		s.unlockPost(w, r, sc, err)
	})
	return nil
}

func (s *Service) origin(r *http.Request) string {
	if pub := strings.TrimRight(s.set.Get("instance.public_url"), "/"); pub != "" && hostOnly(r.Host) == s.instanceHost() {
		return pub
	}
	return httpx.RequestOrigin(r, s.auth.TrustProxy())
}

func (s *Service) siteScope(r *http.Request, slug, base string) (*scope, error) {
	x, err := s.one(r.Context(), "s.slug = ?", slug)
	if err != nil {
		return nil, err
	}
	return s.scopeFor(r, x, base)
}

func (s *Service) scopeFor(r *http.Request, x *Site, base string) (*scope, error) {
	if !s.live(r.Context(), x) {
		return nil, store.ErrNotFound
	}
	sc := &scope{site: x, ws: x.WorkspaceID, root: x.PageID, base: base, origin: s.origin(r), indexing: x.Indexing && !x.Password, theme: x.Theme, font: x.Font}
	if err := s.load(r.Context(), sc, x.Subpages); err != nil {
		return nil, err
	}
	if len(sc.pages) == 0 {
		return nil, store.ErrNotFound
	}
	return sc, nil
}

func (s *Service) linkScope(r *http.Request, token string) (*scope, error) {
	l, err := s.pages.ResolveLink(r.Context(), token)
	if err != nil {
		return nil, err
	}
	sc := &scope{link: l, ws: l.WorkspaceID, root: l.PageID, base: "/l/" + token, origin: s.origin(r), level: l.Level, theme: "auto", font: "sans"}
	if err := s.load(r.Context(), sc, true); err != nil {
		return nil, err
	}
	if len(sc.pages) == 0 {
		return nil, store.ErrNotFound
	}
	return sc, nil
}

func (s *Service) cookieName(sc *scope) string {
	if sc.site != nil {
		return "fm_site_" + sc.site.ID
	}
	return "fm_link_" + sc.link.LinkID
}

func (s *Service) wantToken(sc *scope) string {
	if sc.site != nil {
		return s.unlockToken("site", sc.site.ID, sc.site.passwordRev)
	}
	return s.unlockToken("link", sc.link.LinkID, 0)
}

func (sc *scope) locked() bool {
	if sc.site != nil {
		return sc.site.passwordHash != ""
	}
	return sc.link.Password != ""
}

func (s *Service) unlocked(r *http.Request, sc *scope) bool {
	if !sc.locked() {
		return true
	}
	c, err := r.Cookie(s.cookieName(sc))
	return err == nil && secrets.Equal(c.Value, s.wantToken(sc))
}

func (s *Service) serve(w http.ResponseWriter, r *http.Request, sc *scope, err error, id string) {
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			s.log.Warn("site: scope", "err", err)
		}
		writeHTML(w, http.StatusNotFound, s.missing(nil))
		return
	}
	if !s.unlocked(r, sc) {
		action := sc.base + "/unlock"
		if sc.base == "" {
			action = "/unlock"
		}
		writeHTML(w, http.StatusUnauthorized, s.gatePage(sc, action, r.URL.Path, r.URL.Query().Get("failed") == "1"))
		return
	}
	if id == "" {
		id = sc.root
	}
	if _, ok := sc.pages[id]; !ok {
		writeHTML(w, http.StatusNotFound, s.missing(sc))
		return
	}
	body, err := s.renderPage(r.Context(), sc, id)
	if err != nil {
		s.log.Warn("site: render", "page", id, "err", err)
		writeHTML(w, http.StatusInternalServerError, s.missing(sc))
		return
	}
	writeHTML(w, http.StatusOK, body)
	if s.onView != nil && r.Method == http.MethodGet {
		user := ""
		if p := policy.From(r.Context()); p.IsSession() {
			user = p.UserID
		}
		s.onView(context.WithoutCancel(r.Context()), id, sc.ws, user, s.visitor(r))
	}
}

func (s *Service) visitor(r *http.Request) string {
	sum := sha256.Sum256([]byte(time.Now().UTC().Format("2006-01-02") + "|" + s.set.Get("sites.key") + "|" + httpx.ClientIP(r, s.auth.TrustProxy()) + "|" + r.UserAgent()))
	return hex.EncodeToString(sum[:12])
}

func (s *Service) fileAllowed(ctx context.Context, sc *scope, f *files.File) bool {
	if f.WorkspaceID != sc.ws {
		return false
	}
	ref := "f:" + f.ID
	if sc.site != nil && sc.site.Image == ref {
		return true
	}
	if sc.usedFonts()[f.ID] {
		return true
	}
	if _, ok := sc.pages[f.PageID]; ok && f.PageID != "" {
		return true
	}
	for _, n := range sc.pages {
		if n.Cover == ref || n.Icon == ref {
			return true
		}
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT page_id FROM doc_refs WHERE kind = 'file' AND target = ?", f.ID)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var pg string
		if rows.Scan(&pg) == nil {
			if _, ok := sc.pages[pg]; ok {
				return true
			}
		}
	}
	for id := range sc.pages {
		if s.docs.LiveFile(id, f.ID) {
			return true
		}
	}
	return false
}

func (s *Service) fontsSheet(w http.ResponseWriter, r *http.Request, sc *scope, err error) {
	if err != nil || !s.unlocked(r, sc) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(s.fontsCSS(sc)))
}

func (s *Service) file(w http.ResponseWriter, r *http.Request, sc *scope, err error, id string) {
	if err != nil || !s.unlocked(r, sc) {
		http.NotFound(w, r)
		return
	}
	if err := s.files.ServePublic(w, r, id, func(f *files.File) bool { return s.fileAllowed(r.Context(), sc, f) }); err != nil {
		http.NotFound(w, r)
	}
}

func (s *Service) unlockPost(w http.ResponseWriter, r *http.Request, sc *scope, err error) {
	if err != nil {
		writeHTML(w, http.StatusNotFound, s.missing(nil))
		return
	}
	if o := r.Header.Get("Origin"); o != "" && !strings.EqualFold(o, httpx.RequestOrigin(r, s.auth.TrustProxy())) {
		http.Error(w, "cross-site request refused", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	_ = r.ParseForm()
	home := sc.base
	if home == "" {
		home = "/"
	}
	next := r.PostForm.Get("next")
	if next == "" || (next != home && !strings.HasPrefix(next, sc.base+"/")) || strings.HasPrefix(next, "//") {
		next = home
	}
	key := s.cookieName(sc) + "|" + httpx.ClientIP(r, s.auth.TrustProxy())
	if !s.unlock.Allow(key) {
		http.Error(w, "too many attempts, wait a minute", http.StatusTooManyRequests)
		return
	}
	hash := ""
	if sc.site != nil {
		hash = sc.site.passwordHash
	} else {
		hash = sc.link.Password
	}
	ok, verr := auth.VerifyPassword(r.Context(), hash, r.PostForm.Get("password"))
	if verr != nil || !ok {
		sep := "?"
		if strings.Contains(next, "?") {
			sep = "&"
		}
		http.Redirect(w, r, next+sep+"failed=1", http.StatusSeeOther)
		return
	}
	path := sc.base
	if path == "" {
		path = "/"
	}
	http.SetCookie(w, &http.Cookie{Name: s.cookieName(sc), Value: s.wantToken(sc), Path: path, MaxAge: 30 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: httpx.RequestScheme(r, s.auth.TrustProxy()) == "https"})
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Service) sitemap(w http.ResponseWriter, r *http.Request, sc *scope, err error) {
	if err != nil || !sc.indexing {
		http.NotFound(w, r)
		return
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for id, n := range sc.pages {
		b.WriteString(`<url><loc>` + esc(sc.origin+sc.href(id)) + `</loc><lastmod>` + time.UnixMilli(n.UpdatedAt).UTC().Format("2006-01-02") + `</lastmod></url>`)
	}
	b.WriteString(`</urlset>`)
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func (s *Service) Host(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := hostOnly(r.Host)
		if host == "" || host == s.instanceHost() || strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/favicon.svg" {
			next.ServeHTTP(w, r)
			return
		}
		id := s.siteForHost(r.Context(), host)
		if id == "" {
			next.ServeHTTP(w, r)
			return
		}
		x, err := s.one(r.Context(), "s.id = ?", id)
		var sc *scope
		if err == nil {
			sc, err = s.scopeFor(r, x, "")
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		switch {
		case r.Method == http.MethodPost && path == "unlock":
			s.unlockPost(w, r, sc, err)
		case r.Method != http.MethodGet && r.Method != http.MethodHead:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		case path == "robots.txt":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			if err == nil && sc.indexing {
				_, _ = w.Write([]byte("User-agent: *\nAllow: /\nSitemap: " + sc.origin + "/sitemap.xml\n"))
			} else {
				_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
			}
		case path == "sitemap.xml":
			s.sitemap(w, r, sc, err)
		case strings.HasPrefix(path, "files/"):
			s.file(w, r, sc, err, strings.TrimPrefix(path, "files/"))
		case strings.Contains(path, "/"):
			writeHTML(w, http.StatusNotFound, s.missing(sc))
		default:
			s.serve(w, r, sc, err, path)
		}
	})
}
