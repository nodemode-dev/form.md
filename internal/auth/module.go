package auth

import (
	"net/http"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type module struct{ s *Service }

func (s *Service) Module() app.Module { return module{s} }

func (module) ID() string { return "auth" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	r.HandleFunc("GET /api/auth/policy", s.Public(s.handlePolicy))
	r.HandleFunc("POST /api/setup", s.Public(s.handleSetup))
	r.HandleFunc("GET /desktop/enter", s.Public(s.handleDesktopEnter))
	r.HandleFunc("POST /api/auth/login", s.Public(s.handleLogin))
	r.HandleFunc("POST /api/auth/logout", s.Public(s.handleLogout))
	r.HandleFunc("POST /api/auth/signup", s.Public(s.handleSignup))
	r.HandleFunc("GET /api/auth/sso/saml/start", s.Public(s.handleSAMLStart))
	r.HandleFunc("GET /api/auth/saml/metadata", s.Public(s.handleSAMLMetadata))
	r.HandleFunc("POST /api/auth/saml/acs", s.Public(s.handleSAMLACS))
	r.HandleFunc("GET /api/auth/sso/{provider}/start", s.Public(s.handleSSOStart))
	r.HandleFunc("GET /api/auth/sso/{provider}/callback", s.Public(s.handleSSOCallback))

	r.HandleFunc("GET /api/me", s.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		me, err := s.Me(r.Context(), p)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, me)
		return nil
	}))
	r.HandleFunc("PATCH /api/me", s.RequireSession(s.handleUpdateMe))
	r.HandleFunc("POST /api/me/password", s.RequireSession(s.handlePassword))
	r.HandleFunc("POST /api/me/email", s.RequireSession(s.handleEmail))
	r.HandleFunc("POST /api/me/totp/setup", s.RequireSession(s.handleTOTPSetup))
	r.HandleFunc("POST /api/me/totp/confirm", s.RequireSession(s.handleTOTPConfirm))
	r.HandleFunc("POST /api/me/totp/disable", s.RequireSession(s.handleTOTPDisable))
	r.HandleFunc("GET /api/me/tokens", s.RequireSession(s.handleListTokens))
	r.HandleFunc("POST /api/me/tokens", s.RequireSession(s.handleCreateToken))
	r.HandleFunc("DELETE /api/me/tokens/{id}", s.RequireSession(s.handleDeleteToken))
	r.HandleFunc("GET /api/me/sessions", s.RequireSession(s.handleListSessions))
	r.HandleFunc("DELETE /api/me/sessions/{id}", s.RequireSession(s.handleDeleteSession))

	r.HandleFunc("POST /api/workspaces/{id}/invites", s.RequireSession(s.handleCreateInvite))
	r.HandleFunc("GET /api/workspaces/{id}/invites", s.RequireSession(s.handleListInvites))
	r.HandleFunc("DELETE /api/workspaces/{id}/invites/{invite}", s.RequireSession(s.handleDeleteInvite))
	r.HandleFunc("GET /api/invites/{token}", s.Public(s.handleGetInvite))
	r.HandleFunc("POST /api/invites/{token}/accept", s.Public(s.handleAcceptInvite))
	return nil
}
