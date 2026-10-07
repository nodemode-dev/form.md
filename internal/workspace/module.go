package workspace

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

func (module) ID() string { return "workspace" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	r.HandleFunc("GET /api/workspaces", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, withheld, err := s.List(r.Context(), p)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"workspaces": list, "withheld": withheld})
		return nil
	}))
	r.HandleFunc("POST /api/workspaces", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Name string `json:"name"`
			Icon string `json:"icon"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		ws, err := s.Create(r.Context(), p, in.Name, in.Icon)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"workspace": ws})
		return nil
	}))
	r.HandleFunc("GET /api/workspaces/{id}", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		ws, _, err := s.Get(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"workspace": ws})
		return nil
	}))
	r.HandleFunc("PATCH /api/workspaces/{id}", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Update
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		ws, err := s.Update(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"workspace": ws})
		return nil
	}))
	r.HandleFunc("DELETE /api/workspaces/{id}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Delete(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("GET /api/workspaces/{id}/members", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Members(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"members": list})
		return nil
	}))
	r.HandleFunc("POST /api/workspaces/{id}/members", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.AddMember(r.Context(), p, r.PathValue("id"), in.Email, in.Role); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("PATCH /api/workspaces/{id}/members/{user}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Role string `json:"role"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetRole(r.Context(), p, r.PathValue("id"), r.PathValue("user"), in.Role); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("DELETE /api/workspaces/{id}/members/{user}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.RemoveMember(r.Context(), p, r.PathValue("id"), r.PathValue("user")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("GET /api/workspaces/{id}/rules", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		rules, err := s.GetRules(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, rules)
		return nil
	}))
	r.HandleFunc("PUT /api/workspaces/{id}/rules", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Rules string `json:"rules"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetRules(r.Context(), p, r.PathValue("id"), in.Rules); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("DELETE /api/workspaces/{id}/rules/proposal", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DismissProposal(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("GET /api/workspaces/{id}/emergency", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.EmergencyLog(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"grants": list})
		return nil
	}))
	r.HandleFunc("DELETE /api/workspaces/{id}/emergency/{grant}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.EndEmergency(r.Context(), p, r.PathValue("id"), r.PathValue("grant")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	return nil
}
