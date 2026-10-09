package comment

import (
	"net/http"

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

func (module) ID() string { return "comment" }

type handler = func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error

func ok(w http.ResponseWriter, v any) error {
	httpx.JSON(w, http.StatusOK, v)
	return nil
}

func done(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type bodyInput struct {
	Body string `json:"body"`
}

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	route := func(pattern string, h handler) { r.HandleFunc(pattern, a.Require(h)) }

	route("GET /api/pages/{id}/comments", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		l, err := s.List(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, l)
	})
	route("POST /api/pages/{id}/comments", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in CreateInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		t, err := s.Create(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"thread": t})
		return nil
	})
	route("POST /api/threads/{id}/comments", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in bodyInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		c, err := s.Reply(r.Context(), p, r.PathValue("id"), in.Body)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"comment": c})
		return nil
	})
	route("PUT /api/threads/{id}/resolved", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Resolved bool `json:"resolved"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Resolve(r.Context(), p, r.PathValue("id"), in.Resolved); err != nil {
			return err
		}
		return done(w)
	})
	route("PATCH /api/comments/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in bodyInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Edit(r.Context(), p, r.PathValue("id"), in.Body); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/comments/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Delete(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("PUT /api/comments/{id}/reactions/{icon}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.React(r.Context(), p, r.PathValue("id"), r.PathValue("icon"), true); err != nil {
			return err
		}
		return done(w)
	})
	route("POST /api/pages/{id}/suggestions", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in SuggestInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		x, err := s.Suggest(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"suggestion": x})
		return nil
	})
	route("PATCH /api/suggestions/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in SuggestInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Revise(r.Context(), p, r.PathValue("id"), in); err != nil {
			return err
		}
		return done(w)
	})
	route("PUT /api/suggestions/{id}/decision", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Accept bool `json:"accept"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Decide(r.Context(), p, r.PathValue("id"), in.Accept); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/comments/{id}/reactions/{icon}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.React(r.Context(), p, r.PathValue("id"), r.PathValue("icon"), false); err != nil {
			return err
		}
		return done(w)
	})
	pageReact := func(on bool) func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		return func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
			list, err := s.ReactPage(r.Context(), p, r.PathValue("id"), r.PathValue("icon"), on)
			if err != nil {
				return err
			}
			if list == nil {
				list = []Reaction{}
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"reactions": list})
			return nil
		}
	}
	route("PUT /api/pages/{id}/reactions/{icon}", pageReact(true))
	route("DELETE /api/pages/{id}/reactions/{icon}", pageReact(false))
	return nil
}
