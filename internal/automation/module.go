package automation

import (
	"net/http"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "automation" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.s.Auth
	route := func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, a.Require(h))
	}
	route("GET /api/collections/{id}/automations", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, design, err := s.List(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"automations": list, "can_design": design})
		return nil
	})
	route("POST /api/collections/{id}/automations", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Input
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		out, err := s.Create(r.Context(), p, r.PathValue("id"), a.BaseURL(r), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"automation": out})
		return nil
	})
	route("PATCH /api/automations/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Input
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		out, err := s.Update(r.Context(), p, r.PathValue("id"), a.BaseURL(r), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"automation": out})
		return nil
	})
	route("DELETE /api/automations/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Delete(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	route("GET /api/automations/{id}/runs", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Runs(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"runs": list})
		return nil
	})
	route("POST /api/automations/{id}/run", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			RowID string `json:"row_id"`
		}
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		out, err := s.RunNow(r.Context(), p, r.PathValue("id"), in.RowID)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"outcome": out})
		return nil
	})
	route("POST /api/buttons/run", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ButtonInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		out, err := s.RunButton(r.Context(), p, a.BaseURL(r), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"outcome": out})
		return nil
	})
	route("POST /api/rows/{id}/buttons/{prop}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		out, err := s.RunProperty(r.Context(), p, a.BaseURL(r), r.PathValue("id"), r.PathValue("prop"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"outcome": out})
		return nil
	})
	route("GET /api/workspaces/{id}/automation-key", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		key, err := s.Key(r.Context(), p, r.PathValue("id"), false)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"key": key})
		return nil
	})
	route("POST /api/workspaces/{id}/automation-key", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		key, err := s.Key(r.Context(), p, r.PathValue("id"), true)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"key": key})
		return nil
	})
	return nil
}
