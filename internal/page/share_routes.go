package page

import (
	"net/http"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

func (m module) mountSharing(route func(string, handler)) {
	s := m.s

	route("GET /api/workspaces/{id}/teamspaces", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Teamspaces(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"teamspaces": list})
	})
	route("POST /api/workspaces/{id}/teamspaces", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in TeamInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		t, err := s.CreateTeamspace(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"teamspace": t})
		return nil
	})
	route("PATCH /api/teamspaces/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in TeamInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		t, err := s.UpdateTeamspace(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"teamspace": t})
	})
	route("DELETE /api/teamspaces/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DeleteTeamspace(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("GET /api/teamspaces/{id}/members", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.TeamMembers(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"members": list})
	})
	route("PUT /api/teamspaces/{id}/members/{user}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Role string `json:"role"`
		}
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		if err := s.SetTeamMember(r.Context(), p, r.PathValue("id"), r.PathValue("user"), in.Role); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/teamspaces/{id}/members/{user}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.RemoveTeamMember(r.Context(), p, r.PathValue("id"), r.PathValue("user")); err != nil {
			return err
		}
		return done(w)
	})

	route("GET /api/workspaces/{id}/groups", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Groups(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"groups": list})
	})
	route("POST /api/workspaces/{id}/groups", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		g, err := s.CreateGroup(r.Context(), p, r.PathValue("id"), in.Name, "")
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"group": g})
		return nil
	})
	route("PATCH /api/groups/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		g, err := s.RenameGroup(r.Context(), p, r.PathValue("id"), in.Name)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"group": g})
	})
	route("DELETE /api/groups/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DeleteGroup(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("GET /api/groups/{id}/members", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.GroupMembers(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"members": list})
	})
	route("PUT /api/groups/{id}/members/{user}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.SetGroupMember(r.Context(), p, r.PathValue("id"), r.PathValue("user"), true); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/groups/{id}/members/{user}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.SetGroupMember(r.Context(), p, r.PathValue("id"), r.PathValue("user"), false); err != nil {
			return err
		}
		return done(w)
	})

	route("GET /api/pages/{id}/shares", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		sh, err := s.Sharing(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, sh)
	})
	route("PUT /api/pages/{id}/shares", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ShareInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		res, err := s.Share(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		return ok(w, res)
	})
	route("PATCH /api/shares/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Level     string `json:"level"`
			ExpiresAt int64  `json:"expires_at"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.UpdateShare(r.Context(), p, r.PathValue("id"), in.Level, in.ExpiresAt); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/shares/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Unshare(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("POST /api/pages/{id}/links", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in LinkInput
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		l, err := s.CreateLink(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"link": l})
		return nil
	})
	route("DELETE /api/links/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.RevokeLink(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("DELETE /api/guest-invites/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.CancelGuestInvite(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		return done(w)
	})
	route("GET /api/shared", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.SharedWithMe(r.Context(), p)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"pages": list})
	})
}
