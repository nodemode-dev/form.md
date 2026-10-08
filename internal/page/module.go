package page

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

func (module) ID() string { return "page" }

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

	route("GET /api/workspaces/{id}/pages", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		pages, err := s.Tree(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"pages": pages})
	})
	route("GET /api/workspaces/{id}/trash", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		items, err := s.Trash(r.Context(), p, r.PathValue("id"), r.URL.Query().Get("q"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"items": items})
	})
	route("DELETE /api/workspaces/{id}/trash", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		n, err := s.EmptyTrash(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"deleted": n})
	})
	route("GET /api/workspaces/{id}/tags", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		tags, err := s.Tags(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"tags": tags})
	})
	route("PUT /api/workspaces/{id}/tags/{tag}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Color string `json:"color"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetTagColor(r.Context(), p, r.PathValue("id"), r.PathValue("tag"), in.Color); err != nil {
			return err
		}
		return done(w)
	})

	route("POST /api/pages", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in CreateInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		m, err := s.Create(r.Context(), p, in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"page": m})
		return nil
	})
	route("GET /api/pages/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		d, err := s.Get(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"page": d})
	})
	route("PATCH /api/pages/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in UpdateInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		m, err := s.Update(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"page": m})
	})
	route("POST /api/pages/{id}/move", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in MoveInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		m, err := s.Move(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"page": m})
	})
	route("POST /api/pages/{id}/duplicate", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in DuplicateInput
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		m, err := s.Duplicate(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"page": m})
		return nil
	})
	route("DELETE /api/pages/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		ids, err := s.TrashPage(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"trashed": ids})
	})
	route("POST /api/pages/{id}/restore", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		m, err := s.Restore(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"page": m})
	})
	route("DELETE /api/pages/{id}/forever", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DeleteForever(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("POST /api/pages/{id}/visit", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Visit(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("GET /api/recent", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		list, err := s.Recent(r.Context(), p, limit)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"pages": list})
	})
	route("GET /api/favorites", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Favorites(r.Context(), p)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"pages": list})
	})
	route("PUT /api/favorites/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in FavoriteInput
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		if err := s.SetFavorite(r.Context(), p, r.PathValue("id"), in); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/favorites/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.RemoveFavorite(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	m.mountSharing(route)
	return nil
}
