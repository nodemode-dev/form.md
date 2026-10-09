package agent

import (
	"context"
	"net/http"

	"github.com/nodemode-ai/form.md/internal/ai"
	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "agent" }

func (m module) Mount(r *httpx.Router, d *app.Deps) error {
	s := m.s
	a := s.Auth
	r.HandleFunc("GET /api/agent", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		ws := r.URL.Query().Get("workspace")
		chats, err := s.Chats(r.Context(), p, ws, 40)
		if err != nil {
			return err
		}
		mem := s.memories(r.Context(), p.UserID, ws)
		if mem == nil {
			mem = []Memory{}
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"prefs": s.prefs(r.Context(), p, ws), "memory": mem, "chats": chats})
		return nil
	}))
	r.HandleFunc("PUT /api/agent/prefs", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			WorkspaceID string `json:"workspace_id"`
			Prefs
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetPrefs(r.Context(), p, in.WorkspaceID, in.Prefs); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("DELETE /api/agent/memory/{id}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Forget(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("POST /api/agent/chats", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			WorkspaceID string `json:"workspace_id"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		c, err := s.NewChat(r.Context(), p, in.WorkspaceID, "", "")
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"chat": c})
		return nil
	}))
	r.HandleFunc("GET /api/agent/chats/{id}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		c, err := s.chat(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		msgs, err := s.Messages(r.Context(), c.ID)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"chat": c, "messages": msgs})
		return nil
	}))
	r.HandleFunc("DELETE /api/agent/chats/{id}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DeleteChat(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("POST /api/agent/chats/{id}/messages", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Text   string `json:"text"`
			PageID string `json:"page_id"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if in.Text == "" {
			return httpx.Fail(http.StatusBadRequest, "empty", "write a message")
		}
		c, err := s.chat(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		if c.AgentID != "" || c.userID != p.UserID {
			return httpx.ErrForbidden
		}
		if err := s.AI.Allowed(r.Context(), p, c.WorkspaceID); err != nil {
			return err
		}
		st := ai.OpenStream(w)
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		_, err = s.Run(ctx, p, &Turn{Chat: c, Text: in.Text, PageID: in.PageID}, func(event string, data any) error {
			if e := st.Send(event, data); e != nil {
				cancel()
				return e
			}
			return nil
		})
		if err != nil && ctx.Err() == nil {
			st.Fail(err)
		}
		return nil
	}))
	r.HandleFunc("GET /api/workspaces/{id}/agents", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Agents(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"agents": list})
		return nil
	}))
	r.HandleFunc("POST /api/workspaces/{id}/agents", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in AgentInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		ag, err := s.CreateAgent(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"agent": ag})
		return nil
	}))
	r.HandleFunc("PATCH /api/agents/{id}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in AgentInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		ag, err := s.UpdateAgent(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"agent": ag})
		return nil
	}))
	r.HandleFunc("DELETE /api/agents/{id}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DeleteAgent(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("POST /api/agents/{id}/run", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		c, err := s.Start(r.Context(), p, r.PathValue("id"), "")
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusAccepted, map[string]any{"chat": c})
		return nil
	}))
	r.HandleFunc("GET /api/agents/{id}/runs", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.AgentRuns(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"runs": list})
		return nil
	}))
	return nil
}
