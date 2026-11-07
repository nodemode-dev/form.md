package formext

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

const (
	Product  = "formmd"
	AppName  = "form.md"
	FileBase = "/api/extensions/files"
)

type Service struct {
	Auth *auth.Service
	Root string
}

func New(a *auth.Service) *Service { return &Service{Auth: a} }

func AppVersion() string { return strings.TrimPrefix(app.Version, "v") }

func (s *Service) List() []*Extension { return Resolve(Product, AppVersion(), AppName, s.Root) }

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "formext" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.s.Auth
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if p := r.URL.Path; strings.HasPrefix(p, FileBase+"/") && path.Clean(p) != p {
				for k, v := range HeadersFor("", "") {
					w.Header().Set(k, v)
				}
				w.WriteHeader(http.StatusNotFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	r.HandleFunc("GET /api/extensions", a.Require(func(w http.ResponseWriter, r *http.Request, _ *policy.Principal) error {
		list := s.List()
		out := make([]Public, 0, len(list))
		for _, e := range list {
			out = append(out, e.Public(FileBase))
		}
		root := s.Root
		if root == "" {
			root = PortalRoot()
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"product": Product, "app": AppName, "appVersion": AppVersion(), "api": API, "root": root, "stamp": Stamp(Product, s.Root), "extensions": out})
		return nil
	}))
	r.HandleFunc("GET /api/extensions/stamp", a.Require(func(w http.ResponseWriter, r *http.Request, _ *policy.Principal) error {
		httpx.JSON(w, http.StatusOK, map[string]any{"stamp": Stamp(Product, s.Root)})
		return nil
	}))
	r.HandleFunc("GET /api/extensions/files/{id}/{version}/{path...}", func(w http.ResponseWriter, r *http.Request) {
		target, kind, ok := FileFor(s.List(), r.PathValue("id"), r.PathValue("version"), r.PathValue("path"))
		h := w.Header()
		for k, v := range HeadersFor("", "") {
			h.Set(k, v)
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f, err := os.Open(target)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.Del("Content-Security-Policy")
		for k, v := range HeadersFor(kind, "") {
			h.Set(k, v)
		}
		if strings.HasPrefix(kind, "text/html") {
			h.Set("X-Frame-Options", "SAMEORIGIN")
		}
		h.Set("Content-Type", kind)
		http.ServeContent(w, r, "", st.ModTime(), f)
	})
	return nil
}
