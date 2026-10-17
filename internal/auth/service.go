package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	SessionCookie = "formmd_session"
	TokenPrefix   = "fmd_"
)

type Resolver func(ctx context.Context, bearer string) (*policy.Principal, error)

type Service struct {
	st       *store.Store
	box      *secrets.Box
	settings *settings.Service
	gate     *policy.Gate
	log      *slog.Logger

	loginLimit    *httpx.Limiter
	badTokenLimit *httpx.Limiter
	apiLimit      *httpx.Limiter
	setupMu       sync.Mutex

	mu        sync.RWMutex
	resolvers []Resolver
	onRevoke  []func(ctx context.Context, userID string)
	onCred    []func(credentialID string)
	accounts  AccountHooks
	mailer    Mailer
	meExtras  func(ctx context.Context, p *policy.Principal) (map[string]any, error)

	avatarCheck func(ctx context.Context, ref, userID string) bool
	guestAccept GuestAccept
}

func (s *Service) SetMeExtras(fn func(ctx context.Context, p *policy.Principal) (map[string]any, error)) {
	s.meExtras = fn
}

func (s *Service) SetAvatarCheck(fn func(ctx context.Context, ref, userID string) bool) {
	s.avatarCheck = fn
}

type AccountHooks interface {
	Provision(ctx context.Context, tx *sql.Tx, userID, name string, owner bool) error
	JoinInvite(ctx context.Context, tx *sql.Tx, userID, workspaceID, role string) error
}

func New(st *store.Store, box *secrets.Box, set *settings.Service, gate *policy.Gate, log *slog.Logger) *Service {
	return &Service{
		st:            st,
		box:           box,
		settings:      set,
		gate:          gate,
		log:           log,
		loginLimit:    httpx.NewLimiter(30, 10),
		badTokenLimit: httpx.NewLimiter(60, 20),
		apiLimit:      httpx.NewLimiter(apiPerMinute, apiBurst),
	}
}

const (
	apiPerMinute = 600
	apiBurst     = 120
)

func (s *Service) SetAccountHooks(h AccountHooks) { s.accounts = h }

func (s *Service) AddResolver(r Resolver) {
	s.mu.Lock()
	s.resolvers = append(s.resolvers, r)
	s.mu.Unlock()
}

func (s *Service) OnRevoke(fn func(ctx context.Context, userID string)) {
	s.mu.Lock()
	s.onRevoke = append(s.onRevoke, fn)
	s.mu.Unlock()
}

func (s *Service) OnCredentialRevoke(fn func(credentialID string)) {
	s.mu.Lock()
	s.onCred = append(s.onCred, fn)
	s.mu.Unlock()
}

func (s *Service) CredentialRevoked(ids ...string) {
	s.mu.RLock()
	hooks := append([]func(string){}, s.onCred...)
	s.mu.RUnlock()
	for _, id := range ids {
		if id == "" {
			continue
		}
		for _, fn := range hooks {
			fn(id)
		}
	}
}

func (s *Service) Settings() *settings.Service { return s.settings }
func (s *Service) Gate() *policy.Gate          { return s.gate }
func (s *Service) TrustProxy() bool            { return s.settings.Bool("net.trust_proxy") }

func (s *Service) ClientIP(r *http.Request) string { return httpx.ClientIP(r, s.TrustProxy()) }

func (s *Service) authFailure(r *http.Request, kind string) {
	s.log.Warn("form-md auth failure", "kind", kind, "ip", s.ClientIP(r))
}

func (s *Service) sessionTTL() time.Duration {
	return time.Duration(s.settings.Int("auth.session_days")) * 24 * time.Hour
}

func (s *Service) secureCookie(r *http.Request) bool {
	return httpx.RequestScheme(r, s.TrustProxy()) == "https"
}

func (s *Service) StartSession(ctx context.Context, w http.ResponseWriter, r *http.Request, userID string) error {
	token := secrets.Token("", 32)
	now := time.Now()
	exp := now.Add(s.sessionTTL())
	ua := r.UserAgent()
	if len(ua) > 300 {
		ua = ua[:300]
	}
	if _, err := s.st.W.ExecContext(ctx, `
		INSERT INTO sessions (id, token_hash, user_id, created_at, expires_at, last_seen_at, user_agent, ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		secrets.Token("", 12), secrets.Hash(token), userID, now.UnixMilli(), exp.UnixMilli(), now.UnixMilli(), ua, s.ClientIP(r)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: token, Path: "/", Expires: exp,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookie(r),
	})
	return nil
}

func (s *Service) clearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookie(r),
	})
}

func (s *Service) EndSession(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		var id string
		if s.st.W.QueryRowContext(ctx, "DELETE FROM sessions WHERE token_hash = ? RETURNING id", secrets.Hash(c.Value)).Scan(&id) == nil {
			s.CredentialRevoked(id)
		}
	}
	s.clearCookie(w, r)
}

func (s *Service) RevokeCredentials(ctx context.Context, userID, keepSessionID string) error {
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ? AND id != ?", userID, keepSessionID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM api_tokens WHERE user_id = ?", userID)
		return err
	})
	if err != nil {
		return err
	}
	s.mu.RLock()
	hooks := append([]func(context.Context, string){}, s.onRevoke...)
	s.mu.RUnlock()
	for _, fn := range hooks {
		fn(ctx, userID)
	}
	return nil
}

func (s *Service) SweepSessions(ctx context.Context) {
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", store.NowMS())
}

func (s *Service) PrincipalFor(ctx context.Context, userID string) (*policy.Principal, error) {
	return s.principalForUser(ctx, userID)
}

func (s *Service) principalForUser(ctx context.Context, userID string) (*policy.Principal, error) {
	p := &policy.Principal{UserID: userID}
	var disabled int
	err := s.st.R.QueryRowContext(ctx, "SELECT email, name, role, disabled FROM users WHERE id = ?", userID).
		Scan(&p.Email, &p.Name, &p.Role, &disabled)
	if err != nil {
		return nil, store.NoRows(err)
	}
	if disabled == 1 {
		return nil, store.ErrNotFound
	}
	return p, nil
}

func (s *Service) fromSession(ctx context.Context, r *http.Request) (*policy.Principal, error) {
	c, err := r.Cookie(SessionCookie)
	if err != nil || c.Value == "" {
		return nil, nil
	}
	var id, userID string
	var expires, lastSeen int64
	err = s.st.R.QueryRowContext(ctx, "SELECT id, user_id, expires_at, last_seen_at FROM sessions WHERE token_hash = ?", secrets.Hash(c.Value)).
		Scan(&id, &userID, &expires, &lastSeen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := store.NowMS()
	if expires < now {
		_, _ = s.st.W.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
		return nil, nil
	}
	p, err := s.principalForUser(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if now-lastSeen > int64(time.Minute/time.Millisecond) {
		_, _ = s.st.W.ExecContext(ctx, "UPDATE sessions SET last_seen_at = ? WHERE id = ?", now, id)
	}
	p.Kind, p.Scope, p.AllWorkspaces, p.CredentialID = policy.KindSession, "write", true, id
	return p, nil
}

func (s *Service) fromAPIToken(ctx context.Context, r *http.Request, token string) (*policy.Principal, error) {
	var id, userID, scope string
	var all int
	err := s.st.R.QueryRowContext(ctx, "SELECT id, user_id, scope, all_workspaces FROM api_tokens WHERE token_hash = ?", secrets.Hash(token)).
		Scan(&id, &userID, &scope, &all)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p, err := s.principalForUser(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Kind, p.Scope, p.AllWorkspaces, p.CredentialID = policy.KindToken, scope, all == 1, id
	if !p.AllWorkspaces {
		p.Workspaces = map[string]bool{}
		rows, err := s.st.R.QueryContext(ctx, "SELECT workspace_id FROM api_token_workspaces WHERE token_id = ?", id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var ws string
			if err := rows.Scan(&ws); err != nil {
				rows.Close()
				return nil, err
			}
			p.Workspaces[ws] = true
		}
		rows.Close()
	}
	_, _ = s.st.W.ExecContext(ctx, "UPDATE api_tokens SET last_used_at = ?, last_used_ip = ? WHERE id = ?", store.NowMS(), s.ClientIP(r), id)
	return p, nil
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (s *Service) resolve(ctx context.Context, r *http.Request) (*policy.Principal, *httpx.APIError, error) {
	if tok := bearer(r); tok != "" {
		ip := s.ClientIP(r)
		if s.badTokenLimit.Exhausted(ip) {
			return nil, httpx.Fail(http.StatusTooManyRequests, "rate_limited", "too many rejected credentials, slow down"), nil
		}
		var p *policy.Principal
		var err error
		if strings.HasPrefix(tok, TokenPrefix) {
			p, err = s.fromAPIToken(ctx, r, tok)
		} else {
			s.mu.RLock()
			resolvers := append([]Resolver{}, s.resolvers...)
			s.mu.RUnlock()
			for _, res := range resolvers {
				if p, err = res(ctx, tok); p != nil || err != nil {
					break
				}
			}
		}
		if err != nil {
			return nil, nil, err
		}
		if p == nil {
			s.badTokenLimit.Spend(ip)
			s.authFailure(r, "token")
			return nil, httpx.Fail(http.StatusUnauthorized, "bad_token", "invalid or expired credential"), nil
		}
		return p, nil, nil
	}
	p, err := s.fromSession(ctx, r)
	return p, nil, err
}

func (s *Service) originOK(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if strings.EqualFold(origin, httpx.RequestOrigin(r, s.TrustProxy())) {
		return true
	}
	if pub := s.settings.Get("instance.public_url"); pub != "" {
		scheme, rest, _ := strings.Cut(pub, "://")
		host, _, _ := strings.Cut(rest, "/")
		if strings.EqualFold(origin, scheme+"://"+host) {
			return true
		}
	}
	return false
}

var originExempt = map[string]bool{"/api/auth/saml/acs": true, "/oauth/token": true, "/oauth/register": true, "/oauth/revoke": true}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpx.IsUnsafe(r.Method) && bearer(r) == "" && !originExempt[r.URL.Path] && !s.originOK(r) {
			httpx.Problem(w, r, s.log, httpx.Fail(http.StatusForbidden, "bad_origin", "cross-site request refused"))
			return
		}
		p, apiErr, err := s.resolve(r.Context(), r)
		if err != nil {
			httpx.Problem(w, r, s.log, err)
			return
		}
		if apiErr != nil {
			if apiErr.Status == http.StatusUnauthorized {
				if r.URL.Path == "/mcp" {
					w.Header().Set("WWW-Authenticate", s.MCPChallenge(r, `error="invalid_token"`))
				} else {
					w.Header().Set("WWW-Authenticate", `Bearer realm="form-md"`)
				}
			}
			httpx.Problem(w, r, s.log, apiErr)
			return
		}
		if p != nil {
			if p.Kind == policy.KindOAuth && (r.URL.Path == "/mcp") == (p.Audience == "api") {
				msg := "this token was issued for the MCP endpoint only"
				if p.Audience == "api" {
					msg = "this token was issued for the REST API only"
				}
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="`+msg+`"`)
				httpx.Problem(w, r, s.log, httpx.Fail(http.StatusUnauthorized, "wrong_audience", msg))
				return
			}
			if !p.IsSession() && r.URL.Path != "/mcp" {
				ok, left, wait := s.apiLimit.Take(p.CredentialID)
				w.Header().Set("RateLimit-Limit", strconv.Itoa(apiBurst))
				w.Header().Set("RateLimit-Policy", strconv.Itoa(apiBurst)+";w="+strconv.Itoa(apiBurst*60/apiPerMinute))
				w.Header().Set("RateLimit-Remaining", strconv.Itoa(left))
				if !ok {
					secs := int(wait.Seconds()) + 1
					w.Header().Set("Retry-After", strconv.Itoa(secs))
					w.Header().Set("RateLimit-Reset", strconv.Itoa(secs))
					httpx.Problem(w, r, s.log, httpx.Fail(http.StatusTooManyRequests, "rate_limited", "too many requests with this credential; wait and try again").With("retry_after", secs))
					return
				}
			}
			if httpx.IsUnsafe(r.Method) && !p.CanWrite() && !strings.HasPrefix(r.URL.Path, "/mcp") {
				httpx.Problem(w, r, s.log, httpx.ErrReadOnly)
				return
			}
			r = r.WithContext(policy.With(r.Context(), p))
		}
		next.ServeHTTP(w, r)
	})
}

type Handler func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error

func (s *Service) wrap(check func(p *policy.Principal) error, h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := policy.From(r.Context())
		if p == nil {
			httpx.Problem(w, r, s.log, httpx.ErrUnauthorized)
			return
		}
		if check != nil {
			if err := check(p); err != nil {
				httpx.Problem(w, r, s.log, err)
				return
			}
		}
		if err := h(w, r, p); err != nil {
			if errors.Is(err, policy.ErrHidden) || errors.Is(err, store.ErrNotFound) {
				err = httpx.ErrNotFound
			}
			httpx.Problem(w, r, s.log, err)
		}
	}
}

func (s *Service) Require(h Handler) http.HandlerFunc { return s.wrap(nil, h) }

func (s *Service) RequireSession(h Handler) http.HandlerFunc {
	return s.wrap(func(p *policy.Principal) error {
		if !p.IsSession() {
			return httpx.ErrSessionOnly
		}
		return nil
	}, h)
}

func (s *Service) RequireAdmin(h Handler) http.HandlerFunc {
	return s.wrap(func(p *policy.Principal) error {
		if !p.IsSession() {
			return httpx.ErrSessionOnly
		}
		if !p.IsAdmin() {
			return httpx.ErrForbidden
		}
		return nil
	}, h)
}

func (s *Service) RequireOwner(h Handler) http.HandlerFunc {
	return s.wrap(func(p *policy.Principal) error {
		if !p.IsSession() {
			return httpx.ErrSessionOnly
		}
		if !p.IsOwner() {
			return httpx.ErrForbidden
		}
		return nil
	}, h)
}

func (s *Service) Public(h func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			httpx.Problem(w, r, s.log, err)
		}
	}
}

func (s *Service) Log() *slog.Logger { return s.log }
