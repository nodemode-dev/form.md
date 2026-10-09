package inbox

import (
	"net/http"
	"strconv"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module(a *auth.Service) app.Module { return module{s, a} }

func (module) ID() string { return "inbox" }

type handler = func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error

func ok(w http.ResponseWriter, v any) error {
	httpx.JSON(w, http.StatusOK, v)
	return nil
}

func done(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	route := func(pattern string, h handler) { r.HandleFunc(pattern, a.Require(h)) }

	route("GET /api/inbox", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		items, unread, err := s.List(r.Context(), p, r.URL.Query().Get("filter"), before)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"items": items, "unread": unread})
	})
	route("GET /api/inbox/count", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		n, err := s.Unread(r.Context(), p)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"unread": n})
	})
	route("POST /api/inbox/mark", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in MarkInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Mark(r.Context(), p, in); err != nil {
			return err
		}
		return done(w)
	})
	route("GET /api/me/notifications", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		pr, err := s.Prefs(r.Context(), p)
		if err != nil {
			return err
		}
		return ok(w, pr)
	})
	route("PUT /api/me/notifications", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Email string `json:"email"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetPrefs(r.Context(), p, in.Email); err != nil {
			return err
		}
		return done(w)
	})
	route("POST /api/me/push", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in SubscribeInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Subscribe(r.Context(), p, in); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/me/push", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Endpoint string `json:"endpoint"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Unsubscribe(r.Context(), p, in.Endpoint); err != nil {
			return err
		}
		return done(w)
	})
	route("POST /api/me/push/test", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		n, err := s.TestPush(r.Context(), p)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"sent": n})
	})
	return nil
}
