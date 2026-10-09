package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

type PageText func(ctx context.Context, p *policy.Principal, pageID string) (workspaceID, markdown string, err error)

type Stream struct {
	w     http.ResponseWriter
	fl    http.Flusher
	mu    sync.Mutex
	ended bool
}

func OpenStream(w http.ResponseWriter) *Stream {
	fl, _ := w.(http.Flusher)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if fl != nil {
		fl.Flush()
	}
	return &Stream{w: w, fl: fl}
}

func (st *Stream) Send(event string, v any) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.ended {
		return errors.New("stream ended")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := st.w.Write([]byte("event: " + event + "\ndata: " + string(b) + "\n\n")); err != nil {
		st.ended = true
		return err
	}
	if st.fl != nil {
		st.fl.Flush()
	}
	return nil
}

func (st *Stream) Fail(err error) {
	code, msg := "internal", "the AI request failed"
	var api *httpx.APIError
	if errors.As(err, &api) {
		code, msg = api.Code, api.Message
	}
	_ = st.Send("error", map[string]string{"code": code, "message": msg})
}

type module struct {
	s    *Service
	auth *auth.Service
	page PageText
}

func (s *Service) Module(a *auth.Service, page PageText) app.Module { return module{s, a, page} }

func (module) ID() string { return "ai" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	r.HandleFunc("GET /api/admin/ai", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, _ *policy.Principal) error {
		list, err := s.Providers(r.Context())
		if err != nil {
			return err
		}
		usage, err := s.Usage(r.Context(), 30)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"enabled": s.set.Bool("ai.enabled"), "default_model": s.set.Get("ai.default_model"), "transcribe_model": s.set.Get("ai.transcribe_model"),
			"limit_per_hour": s.set.Int("ai.user_limit_per_hour"), "providers": list, "usage": usage, "kinds": Kinds})
		return nil
	}))
	r.HandleFunc("PUT /api/admin/ai", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Enabled         *bool   `json:"enabled"`
			DefaultModel    *string `json:"default_model"`
			TranscribeModel *string `json:"transcribe_model"`
			LimitPerHour    *int    `json:"limit_per_hour"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		ch := map[string]string{}
		if in.Enabled != nil {
			ch["ai.enabled"] = boolStr(*in.Enabled)
		}
		if in.DefaultModel != nil {
			ch["ai.default_model"] = *in.DefaultModel
		}
		if in.TranscribeModel != nil {
			ch["ai.transcribe_model"] = *in.TranscribeModel
		}
		if in.LimitPerHour != nil {
			ch["ai.user_limit_per_hour"] = itoa(*in.LimitPerHour)
		}
		if err := s.set.Set(r.Context(), ch); err != nil {
			if fe, ok := err.(*settings.FieldError); ok {
				return httpx.Fail(http.StatusBadRequest, "bad_setting", fe.Problem).With("key", fe.Key)
			}
			return err
		}
		_ = s.audit.Record(r.Context(), nil, p, auditEntry("ai.settings"))
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("POST /api/admin/ai/providers", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ProviderInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		id, err := s.CreateProvider(r.Context(), p, in)
		if err != nil {
			return err
		}
		n, ferr := s.RefreshModels(r.Context(), id)
		out := map[string]any{"id": id, "models": n}
		if ferr != nil {
			out["warning"] = ferr.Error()
		}
		httpx.JSON(w, http.StatusCreated, out)
		return nil
	}))
	r.HandleFunc("PATCH /api/admin/ai/providers/{id}", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ProviderInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.UpdateProvider(r.Context(), p, r.PathValue("id"), in); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("DELETE /api/admin/ai/providers/{id}", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DeleteProvider(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("POST /api/admin/ai/providers/{id}/refresh", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, _ *policy.Principal) error {
		n, err := s.RefreshModels(r.Context(), r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]int{"models": n})
		return nil
	}))
	r.HandleFunc("PUT /api/admin/ai/providers/{id}/models", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ModelInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.SetModel(r.Context(), p, r.PathValue("id"), in); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("POST /api/admin/ai/test", a.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Model string `json:"model"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		res, err := s.Run(r.Context(), p, "", "test", in.Model, Request{System: "Reply with the single word OK.", Messages: []Message{Text("user", "Say OK.")}, MaxTokens: 16}, nil)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"reply": strings.TrimSpace(res.Text), "usage": res.Usage})
		return nil
	}))
	r.HandleFunc("GET /api/ai/models", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Choices(r.Context())
		if err != nil {
			return err
		}
		ws := r.URL.Query().Get("workspace")
		on := s.set.Bool("ai.enabled") && len(list) > 0
		if ws != "" {
			if _, err := s.gate.Workspace(r.Context(), p, ws); err != nil {
				return err
			}
			on = on && s.WorkspaceOn(r.Context(), ws)
		}
		def := s.set.Get("ai.default_model")
		httpx.JSON(w, http.StatusOK, map[string]any{"enabled": on, "models": list, "default": def})
		return nil
	}))
	r.HandleFunc("PUT /api/workspaces/{id}/ai", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Enabled bool `json:"enabled"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		acc, err := s.gate.Workspace(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		if !acc.IsAdmin() {
			return httpx.ErrForbidden
		}
		if _, err := s.st.W.ExecContext(r.Context(), "UPDATE workspaces SET ai_enabled = ? WHERE id = ?", store.Bool(in.Enabled), r.PathValue("id")); err != nil {
			return err
		}
		e := auditEntry("workspace.ai")
		e.WorkspaceID = r.PathValue("id")
		_ = s.audit.Record(r.Context(), nil, p, e)
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("POST /api/ai/complete", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Ask
			WorkspaceID string `json:"workspace_id"`
			PageID      string `json:"page_id"`
			Model       string `json:"model"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if !Valid(in.Action) {
			return httpx.Fail(http.StatusBadRequest, "bad_action", "unknown writing action")
		}
		page := ""
		ws := in.WorkspaceID
		if in.PageID != "" && m.page != nil {
			pws, md, err := m.page(r.Context(), p, in.PageID)
			if err != nil {
				return err
			}
			ws, page = pws, md
		}
		if err := s.allowed(r.Context(), p, ws); err != nil {
			return err
		}
		if _, _, _, err := s.resolve(r.Context(), in.Model, false); err != nil {
			return err
		}
		st := OpenStream(w)
		req := Request{System: WritingSystem, Messages: []Message{Text("user", WritingPrompt(in.Ask, page))}, MaxTokens: 2048}
		res, err := s.Run(r.Context(), p, ws, "write."+in.Action, in.Model, req, func(t string) error { return st.Send("delta", map[string]string{"t": t}) })
		if err != nil {
			st.Fail(err)
			return nil
		}
		_ = st.Send("done", map[string]any{"usage": res.Usage})
		return nil
	}))
	r.HandleFunc("POST /api/ai/transcribe", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		r.Body = http.MaxBytesReader(w, r.Body, 26<<20)
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			return httpx.Fail(http.StatusRequestEntityTooLarge, "too_large", "an audio segment can be up to 25 MB")
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			return httpx.Fail(http.StatusBadRequest, "no_file", "send the audio as the file field")
		}
		defer f.Close()
		mime := hdr.Header.Get("Content-Type")
		if !strings.HasPrefix(mime, "audio/") && !strings.HasPrefix(mime, "video/webm") {
			return httpx.Fail(http.StatusBadRequest, "not_audio", "that file is not audio")
		}
		ws := r.FormValue("workspace_id")
		if pid := r.FormValue("page_id"); pid != "" && m.page != nil {
			pws, _, err := m.page(r.Context(), p, pid)
			if err != nil {
				return err
			}
			ws = pws
		}
		text, err := s.Transcribe(r.Context(), p, ws, hdr.Filename, mime, f)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"text": strings.TrimSpace(text)})
		return nil
	}))
	return nil
}
