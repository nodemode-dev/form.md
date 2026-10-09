package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	accessTTL    = time.Hour
	refreshTTL   = 30 * 24 * time.Hour
	codeTTL      = time.Minute
	requestTTL   = 10 * time.Minute
	metadataTTL  = time.Hour
	accessPrefix = "fmo_"
	refreshPfx   = "fmr_"
)

type Service struct {
	st        *store.Store
	auth      *auth.Service
	gate      *policy.Gate
	audit     *audit.Log
	fetch     *http.Client
	allowHTTP bool
	log       *slog.Logger
	regLimit  *httpx.Limiter
}

func New(st *store.Store, a *auth.Service, gate *policy.Gate, al *audit.Log, fetch *http.Client, allowHTTP bool, log *slog.Logger) *Service {
	s := &Service{st: st, auth: a, gate: gate, audit: al, fetch: fetch, allowHTTP: allowHTTP, log: log, regLimit: httpx.NewLimiter(20, 10)}
	a.AddResolver(s.resolve)
	a.OnRevoke(s.revokeUser)
	return s
}

func ms(d time.Duration) int64 { return int64(d / time.Millisecond) }

func (s *Service) issuer(r *http.Request) string   { return s.auth.BaseURL(r) }
func (s *Service) resource(r *http.Request) string { return s.issuer(r) + "/mcp" }

func (s *Service) apiResource(r *http.Request) string { return s.issuer(r) + "/api" }

func (s *Service) okResource(r *http.Request, res string) bool {
	res = strings.TrimRight(res, "/")
	return strings.EqualFold(res, s.resource(r)) || strings.EqualFold(res, s.issuer(r)) || strings.EqualFold(res, s.apiResource(r))
}

func (s *Service) canonResource(r *http.Request, res string) string {
	if strings.EqualFold(strings.TrimRight(res, "/"), s.apiResource(r)) {
		return s.apiResource(r)
	}
	return s.resource(r)
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, MCP-Protocol-Version")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func oauthErr(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": desc})
}

func random(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

type client struct {
	ID        string
	Kind      string
	Name      string
	URI       string
	Redirects []string
}

func isLoopback(host string) bool {
	h := strings.Trim(host, "[]")
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func validRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" || u.Scheme == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.Host != ""
	case "http":
		return isLoopback(u.Hostname())
	case "javascript", "data", "file", "vbscript", "about", "blob":
		return false
	}
	return true
}

func redirectMatch(list []string, given string) bool {
	if slices.Contains(list, given) {
		return true
	}
	g, err := url.Parse(given)
	if err != nil || g.Scheme != "http" || !isLoopback(g.Hostname()) {
		return false
	}
	for _, r := range list {
		u, err := url.Parse(r)
		if err != nil || u.Scheme != "http" || !isLoopback(u.Hostname()) {
			continue
		}
		if u.Hostname() == g.Hostname() && u.Path == g.Path && u.RawQuery == g.RawQuery {
			return true
		}
	}
	return false
}

func (s *Service) isMetadataID(id string) bool {
	u, err := url.Parse(id)
	if err != nil || u.Host == "" || u.Path == "" || u.Path == "/" || u.Fragment != "" {
		return false
	}
	return u.Scheme == "https" || (s.allowHTTP && u.Scheme == "http")
}

func (s *Service) client(ctx context.Context, id string) (*client, error) {
	if id == "" {
		return nil, errors.New("client_id is missing")
	}
	var c client
	var redirects string
	var fetched int64
	err := s.st.R.QueryRowContext(ctx, "SELECT id, kind, name, uri, redirect_uris, fetched_at FROM oauth_clients WHERE id = ?", id).
		Scan(&c.ID, &c.Kind, &c.Name, &c.URI, &redirects, &fetched)
	if err == nil {
		_ = json.Unmarshal([]byte(redirects), &c.Redirects)
		if c.Kind == "registered" || store.NowMS()-fetched < ms(metadataTTL) {
			return &c, nil
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if !s.isMetadataID(id) {
		return nil, errors.New("this client is not registered with form.md")
	}
	return s.fetchMetadata(ctx, id)
}

func (s *Service) fetchMetadata(ctx context.Context, id string) (*client, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.fetch.Do(req)
	if err != nil {
		return nil, errors.New("the client's metadata document could not be fetched")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("the client's metadata document answered " + resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, err
	}
	var doc struct {
		ClientID     string   `json:"client_id"`
		ClientName   string   `json:"client_name"`
		ClientURI    string   `json:"client_uri"`
		RedirectURIs []string `json:"redirect_uris"`
		AuthMethod   string   `json:"token_endpoint_auth_method"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, errors.New("the client's metadata document is not valid JSON")
	}
	if doc.ClientID != id {
		return nil, errors.New("the client's metadata document names a different client_id")
	}
	if strings.TrimSpace(doc.ClientName) == "" || len(doc.RedirectURIs) == 0 {
		return nil, errors.New("the client's metadata document needs client_name and redirect_uris")
	}
	if doc.AuthMethod != "" && doc.AuthMethod != "none" {
		return nil, errors.New("only public clients (token_endpoint_auth_method none) are supported")
	}
	for _, r := range doc.RedirectURIs {
		if !validRedirect(r) {
			return nil, errors.New("the client's metadata document lists an unusable redirect URI")
		}
	}
	name := trim(doc.ClientName, 100)
	rb, _ := json.Marshal(doc.RedirectURIs)
	now := store.NowMS()
	_, err = s.st.W.ExecContext(ctx, `INSERT INTO oauth_clients (id, kind, name, uri, redirect_uris, fetched_at, created_at) VALUES (?, 'metadata', ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET name = excluded.name, uri = excluded.uri, redirect_uris = excluded.redirect_uris, fetched_at = excluded.fetched_at`,
		id, name, trim(doc.ClientURI, 300), string(rb), now, now)
	if err != nil {
		return nil, err
	}
	return &client{ID: id, Kind: "metadata", Name: name, URI: doc.ClientURI, Redirects: doc.RedirectURIs}, nil
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func (s *Service) protectedResource(w http.ResponseWriter, r *http.Request) {
	cors(w)
	res := s.resource(r)
	if strings.HasSuffix(r.URL.Path, "/api") {
		res = s.apiResource(r)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 res,
		"authorization_servers":    []string{s.issuer(r)},
		"scopes_supported":         []string{"read", "write"},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "form.md",
	})
}

func (s *Service) serverMetadata(w http.ResponseWriter, r *http.Request) {
	cors(w)
	iss := s.issuer(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                         iss,
		"authorization_endpoint":                         iss + "/oauth/authorize",
		"token_endpoint":                                 iss + "/oauth/token",
		"registration_endpoint":                          iss + "/oauth/register",
		"revocation_endpoint":                            iss + "/oauth/revoke",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"scopes_supported":                               []string{"read", "write"},
		"client_id_metadata_document_supported":          true,
		"authorization_response_iss_parameter_supported": true,
	})
}

func (s *Service) register(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if !s.regLimit.Allow(s.auth.ClientIP(r)) {
		oauthErr(w, http.StatusTooManyRequests, "slow_down", "too many registrations from this address")
		return
	}
	var in struct {
		RedirectURIs []string `json:"redirect_uris"`
		ClientName   string   `json:"client_name"`
		ClientURI    string   `json:"client_uri"`
		AppType      string   `json:"application_type"`
		AuthMethod   string   `json:"token_endpoint_auth_method"`
		GrantTypes   []string `json:"grant_types"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 32<<10)).Decode(&in); err != nil {
		oauthErr(w, http.StatusBadRequest, "invalid_client_metadata", "the registration is not valid JSON")
		return
	}
	if len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > 10 {
		oauthErr(w, http.StatusBadRequest, "invalid_redirect_uri", "give between one and ten redirect URIs")
		return
	}
	for _, u := range in.RedirectURIs {
		if !validRedirect(u) {
			oauthErr(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect URIs must be https, http on a loopback address, or an app scheme")
			return
		}
	}
	if in.AuthMethod != "" && in.AuthMethod != "none" {
		oauthErr(w, http.StatusBadRequest, "invalid_client_metadata", "only public clients (token_endpoint_auth_method none) are supported")
		return
	}
	appType := in.AppType
	if appType != "web" {
		appType = "native"
	}
	name := trim(in.ClientName, 100)
	if name == "" {
		name = "Unnamed agent"
	}
	id := "fmc_" + ids.New()
	rb, _ := json.Marshal(in.RedirectURIs)
	now := store.NowMS()
	if _, err := s.st.W.ExecContext(r.Context(), "INSERT INTO oauth_clients (id, kind, name, uri, redirect_uris, application_type, created_at) VALUES (?, 'registered', ?, ?, ?, ?, ?)",
		id, name, trim(in.ClientURI, 300), string(rb), appType, now); err != nil {
		oauthErr(w, http.StatusInternalServerError, "server_error", "could not register the client")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id": id, "client_id_issued_at": now / 1000, "client_name": name, "redirect_uris": in.RedirectURIs,
		"grant_types": []string{"authorization_code", "refresh_token"}, "response_types": []string{"code"},
		"token_endpoint_auth_method": "none", "application_type": appType,
	})
}

func errorPage(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Connection refused</title>`+
		`<style>body{margin:0;font:16px/1.5 system-ui,sans-serif;background:#111318;color:#e6e8ee;display:grid;place-items:center;min-height:100vh}main{max-width:32rem;padding:2rem}h1{font-size:1.25rem}p{color:#a9afbd}</style></head>`+
		`<body><main><h1>This app cannot connect</h1><p>`+html.EscapeString(msg)+`</p></main></body></html>`)
}

func withParams(base string, params url.Values) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for k, v := range params {
		q[k] = v
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func normScope(raw string) (string, bool) {
	scope := "read"
	if strings.TrimSpace(raw) == "" {
		return "write", true
	}
	for _, f := range strings.Fields(raw) {
		switch f {
		case "write":
			scope = "write"
		case "read", "offline_access":
		default:
			return "", false
		}
	}
	return scope, true
}

func (s *Service) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	c, err := s.client(r.Context(), q.Get("client_id"))
	if err != nil {
		errorPage(w, err.Error())
		return
	}
	redirect := q.Get("redirect_uri")
	if redirect == "" && len(c.Redirects) == 1 {
		redirect = c.Redirects[0]
	}
	if !redirectMatch(c.Redirects, redirect) {
		errorPage(w, "The redirect address "+redirect+" is not registered for "+c.Name+".")
		return
	}
	iss := s.issuer(r)
	fail := func(code, desc string) {
		http.Redirect(w, r, withParams(redirect, url.Values{"error": {code}, "error_description": {desc}, "state": {q.Get("state")}, "iss": {iss}}), http.StatusFound)
	}
	if q.Get("response_type") != "code" {
		fail("unsupported_response_type", "only the authorization code flow is supported")
		return
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		fail("invalid_request", "PKCE with code_challenge_method S256 is required")
		return
	}
	resource := q.Get("resource")
	if resource == "" {
		resource = s.resource(r)
	}
	if !s.okResource(r, resource) {
		fail("invalid_target", "this server issues tokens for "+s.resource(r)+" only")
		return
	}
	scope, ok := normScope(q.Get("scope"))
	if !ok {
		fail("invalid_scope", "scopes are read and write")
		return
	}
	id := random(18)
	_, err = s.st.W.ExecContext(r.Context(), "INSERT INTO oauth_requests (id, client_id, redirect_uri, state, challenge, scope, resource, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, c.ID, redirect, q.Get("state"), q.Get("code_challenge"), scope, s.canonResource(r, resource), store.NowMS()+ms(requestTTL))
	if err != nil {
		errorPage(w, "The request could not be stored; try again.")
		return
	}
	next := "/oauth/consent?request=" + url.QueryEscape(id)
	if p := policy.From(r.Context()); p == nil || !p.IsSession() {
		http.Redirect(w, r, "/login?next="+url.QueryEscape(next), http.StatusFound)
		return
	}
	http.Redirect(w, r, next, http.StatusFound)
}

type pending struct {
	ID        string
	ClientID  string
	Redirect  string
	State     string
	Challenge string
	Scope     string
	Resource  string
}

func (s *Service) request(ctx context.Context, id string) (*pending, error) {
	var p pending
	var exp int64
	err := s.st.R.QueryRowContext(ctx, "SELECT id, client_id, redirect_uri, state, challenge, scope, resource, expires_at FROM oauth_requests WHERE id = ?", id).
		Scan(&p.ID, &p.ClientID, &p.Redirect, &p.State, &p.Challenge, &p.Scope, &p.Resource, &exp)
	if err != nil || exp < store.NowMS() {
		return nil, httpx.Fail(http.StatusGone, "request_expired", "this connection request expired; start again from the app")
	}
	return &p, nil
}

type Workspace struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Personal bool   `json:"personal"`
}

type View struct {
	Request    string
	Client     string
	Host       string
	Scope      string
	Workspaces []Workspace
}

func (s *Service) ConsentView(ctx context.Context, p *policy.Principal, id string) (*View, error) {
	req, err := s.request(ctx, id)
	if err != nil {
		return nil, err
	}
	c, err := s.client(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}
	host := req.Redirect
	if u, err := url.Parse(req.Redirect); err == nil && u.Host != "" {
		host = u.Host
	} else if err == nil {
		host = u.Scheme + ":"
	}
	v := &View{Request: id, Client: c.Name, Host: host, Scope: req.Scope}
	list, _, err := s.gate.Memberships(ctx, p)
	if err != nil {
		return nil, err
	}
	for _, m := range list {
		var name string
		_ = s.st.R.QueryRowContext(ctx, "SELECT name FROM workspaces WHERE id = ?", m.WorkspaceID).Scan(&name)
		if !policy.AgentAllowed(&policy.Principal{Kind: policy.KindOAuth}, m.AgentAccess) {
			continue
		}
		v.Workspaces = append(v.Workspaces, Workspace{ID: m.WorkspaceID, Name: name, Personal: m.Personal})
	}
	slices.SortStableFunc(v.Workspaces, func(a, b Workspace) int {
		if a.Personal != b.Personal {
			if a.Personal {
				return 1
			}
			return -1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return v, nil
}

func (s *Service) consent(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in struct {
		Request    string   `json:"request"`
		Approve    bool     `json:"approve"`
		Scope      string   `json:"scope"`
		Workspaces []string `json:"workspaces"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	req, err := s.request(ctx, in.Request)
	if err != nil {
		return err
	}
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM oauth_requests WHERE id = ?", req.ID)
	iss := s.issuer(r)
	if !in.Approve {
		httpx.JSON(w, http.StatusOK, map[string]string{"redirect": withParams(req.Redirect, url.Values{"error": {"access_denied"}, "error_description": {"the person declined"}, "state": {req.State}, "iss": {iss}})})
		return nil
	}
	c, err := s.client(ctx, req.ClientID)
	if err != nil {
		return httpx.Fail(http.StatusBadRequest, "bad_client", err.Error())
	}
	scope := "read"
	if req.Scope == "write" && in.Scope != "read" {
		scope = "write"
	}
	view, err := s.ConsentView(ctx, p, req.ID)
	if err != nil {
		view = &View{}
		list, _, lerr := s.gate.Memberships(ctx, p)
		if lerr != nil {
			return lerr
		}
		for _, m := range list {
			if policy.AgentAllowed(&policy.Principal{Kind: policy.KindOAuth}, m.AgentAccess) {
				view.Workspaces = append(view.Workspaces, Workspace{ID: m.WorkspaceID})
			}
		}
	}
	allowed := map[string]bool{}
	for _, ws := range view.Workspaces {
		allowed[ws.ID] = true
	}
	var chosen []string
	for _, id := range in.Workspaces {
		if allowed[id] && !slices.Contains(chosen, id) {
			chosen = append(chosen, id)
		}
	}
	if len(chosen) == 0 {
		return httpx.Fail(http.StatusBadRequest, "no_workspaces", "choose at least one workspace")
	}
	grant := ids.New()
	code := random(32)
	now := store.NowMS()
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO oauth_grants (id, user_id, client_id, client_name, scope, all_workspaces, resource, created_at) VALUES (?, ?, ?, ?, ?, 0, ?, ?)`,
			grant, p.UserID, c.ID, c.Name, scope, req.Resource, now); err != nil {
			return err
		}
		for _, ws := range chosen {
			if _, err := tx.ExecContext(ctx, "INSERT INTO oauth_grant_workspaces (grant_id, workspace_id) VALUES (?, ?)", grant, ws); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO oauth_codes (code_hash, grant_id, client_id, redirect_uri, challenge, resource, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			secrets.Hash(code), grant, c.ID, req.Redirect, req.Challenge, req.Resource, now+ms(codeTTL)); err != nil {
			return err
		}
		for _, ws := range chosen {
			if err := s.audit.Record(ctx, tx, p, audit.Entry{Action: "oauth.grant", WorkspaceID: ws, Detail: c.Name + " (" + scope + ")"}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"redirect": withParams(req.Redirect, url.Values{"code": {code}, "state": {req.State}, "iss": {iss}})})
	return nil
}

func challengeOf(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *Service) issue(ctx context.Context, w http.ResponseWriter, grant, scope string) {
	access := secrets.Token(accessPrefix, 32)
	refresh := secrets.Token(refreshPfx, 32)
	now := store.NowMS()
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO oauth_tokens (token_hash, grant_id, kind, expires_at, created_at) VALUES (?, ?, 'access', ?, ?)", secrets.Hash(access), grant, now+ms(accessTTL), now); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO oauth_tokens (token_hash, grant_id, kind, expires_at, created_at) VALUES (?, ?, 'refresh', ?, ?)", secrets.Hash(refresh), grant, now+ms(refreshTTL), now)
		return err
	})
	if err != nil {
		oauthErr(w, http.StatusInternalServerError, "server_error", "could not issue tokens")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": int(accessTTL / time.Second), "refresh_token": refresh, "scope": scope})
}

func (s *Service) revokeGrant(ctx context.Context, grant string) {
	now := store.NowMS()
	_, _ = s.st.W.ExecContext(ctx, "UPDATE oauth_grants SET revoked_at = ? WHERE id = ? AND revoked_at = 0", now, grant)
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM oauth_tokens WHERE grant_id = ?", grant)
}

func (s *Service) token(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		oauthErr(w, http.StatusBadRequest, "invalid_request", "the token request must be form-encoded")
		return
	}
	f := r.PostForm
	switch f.Get("grant_type") {
	case "authorization_code":
		var grant, clientID, redirect, challenge, resource string
		var exp, used int64
		err := s.st.R.QueryRowContext(ctx, "SELECT grant_id, client_id, redirect_uri, challenge, resource, expires_at, used_at FROM oauth_codes WHERE code_hash = ?", secrets.Hash(f.Get("code"))).
			Scan(&grant, &clientID, &redirect, &challenge, &resource, &exp, &used)
		if err != nil {
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the code is not valid")
			return
		}
		res, _ := s.st.W.ExecContext(ctx, "UPDATE oauth_codes SET used_at = ? WHERE code_hash = ? AND used_at = 0", store.NowMS(), secrets.Hash(f.Get("code")))
		if n, _ := res.RowsAffected(); n == 0 || used > 0 {
			s.revokeGrant(ctx, grant)
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the code was already used; the grant is revoked")
			return
		}
		switch {
		case exp < store.NowMS():
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the code expired")
		case f.Get("client_id") != clientID:
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the code was issued to another client")
		case f.Get("redirect_uri") != "" && f.Get("redirect_uri") != redirect:
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match the authorization request")
		case f.Get("code_verifier") == "" || challengeOf(f.Get("code_verifier")) != challenge:
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "code_verifier does not match the code_challenge")
		case f.Get("resource") != "" && !s.okResource(r, f.Get("resource")):
			oauthErr(w, http.StatusBadRequest, "invalid_target", "this server issues tokens for "+s.resource(r)+" only")
		default:
			var scope string
			if err := s.st.R.QueryRowContext(ctx, "SELECT scope FROM oauth_grants WHERE id = ? AND revoked_at = 0", grant).Scan(&scope); err != nil {
				oauthErr(w, http.StatusBadRequest, "invalid_grant", "the grant was revoked")
				return
			}
			s.issue(ctx, w, grant, scope)
			return
		}
	case "refresh_token":
		hash := secrets.Hash(f.Get("refresh_token"))
		var grant string
		var exp, used int64
		err := s.st.R.QueryRowContext(ctx, "SELECT grant_id, expires_at, used_at FROM oauth_tokens WHERE token_hash = ? AND kind = 'refresh'", hash).Scan(&grant, &exp, &used)
		if err != nil {
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the refresh token is not valid")
			return
		}
		var clientID, scope string
		var revoked int64
		if err := s.st.R.QueryRowContext(ctx, "SELECT client_id, scope, revoked_at FROM oauth_grants WHERE id = ?", grant).Scan(&clientID, &scope, &revoked); err != nil || revoked > 0 {
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the grant was revoked")
			return
		}
		if f.Get("client_id") != "" && f.Get("client_id") != clientID {
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the refresh token was issued to another client")
			return
		}
		res, _ := s.st.W.ExecContext(ctx, "UPDATE oauth_tokens SET used_at = ? WHERE token_hash = ? AND used_at = 0", store.NowMS(), hash)
		if n, _ := res.RowsAffected(); n == 0 || used > 0 {
			s.revokeGrant(ctx, grant)
			s.log.Warn("oauth: refresh token reused; grant revoked", "grant", grant)
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the refresh token was already used; the grant is revoked")
			return
		}
		if exp < store.NowMS() {
			oauthErr(w, http.StatusBadRequest, "invalid_grant", "the refresh token expired")
			return
		}
		if f.Get("resource") != "" && !s.okResource(r, f.Get("resource")) {
			oauthErr(w, http.StatusBadRequest, "invalid_target", "this server issues tokens for "+s.resource(r)+" only")
			return
		}
		s.issue(ctx, w, grant, scope)
	default:
		oauthErr(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type is authorization_code or refresh_token")
	}
}

func (s *Service) revoke(w http.ResponseWriter, r *http.Request) {
	cors(w)
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		oauthErr(w, http.StatusBadRequest, "invalid_request", "form-encoded token expected")
		return
	}
	hash := secrets.Hash(r.PostForm.Get("token"))
	var grant, kind string
	if err := s.st.R.QueryRowContext(r.Context(), "SELECT grant_id, kind FROM oauth_tokens WHERE token_hash = ?", hash).Scan(&grant, &kind); err == nil {
		if kind == "refresh" {
			s.revokeGrant(r.Context(), grant)
		} else {
			_, _ = s.st.W.ExecContext(r.Context(), "DELETE FROM oauth_tokens WHERE token_hash = ?", hash)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Service) resolve(ctx context.Context, bearer string) (*policy.Principal, error) {
	if !strings.HasPrefix(bearer, accessPrefix) {
		return nil, nil
	}
	var grant, user, clientName, scope, resource string
	var exp, all, lastUsed int64
	err := s.st.R.QueryRowContext(ctx, `SELECT t.grant_id, g.user_id, g.client_name, g.scope, g.all_workspaces, t.expires_at, g.last_used_at, g.resource
		FROM oauth_tokens t JOIN oauth_grants g ON g.id = t.grant_id
		WHERE t.token_hash = ? AND t.kind = 'access' AND g.revoked_at = 0`, secrets.Hash(bearer)).Scan(&grant, &user, &clientName, &scope, &all, &exp, &lastUsed, &resource)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && exp < store.NowMS()) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p, err := s.auth.PrincipalFor(ctx, user)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Kind, p.Scope, p.AllWorkspaces, p.CredentialID, p.ClientName = policy.KindOAuth, scope, all == 1, grant, clientName
	if strings.HasSuffix(resource, "/api") {
		p.Audience = "api"
	}
	if !p.AllWorkspaces {
		p.Workspaces = map[string]bool{}
		rows, err := s.st.R.QueryContext(ctx, "SELECT workspace_id FROM oauth_grant_workspaces WHERE grant_id = ?", grant)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var ws string
			if rows.Scan(&ws) == nil {
				p.Workspaces[ws] = true
			}
		}
		rows.Close()
	}
	if now := store.NowMS(); now-lastUsed > ms(time.Minute) {
		_, _ = s.st.W.ExecContext(ctx, "UPDATE oauth_grants SET last_used_at = ? WHERE id = ?", now, grant)
	}
	return p, nil
}

func (s *Service) revokeUser(ctx context.Context, userID string) {
	rows, err := s.st.R.QueryContext(ctx, "SELECT id FROM oauth_grants WHERE user_id = ? AND revoked_at = 0", userID)
	if err != nil {
		return
	}
	var list []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			list = append(list, id)
		}
	}
	rows.Close()
	for _, id := range list {
		s.revokeGrant(ctx, id)
	}
}

type Grant struct {
	ID         string      `json:"id"`
	Client     string      `json:"client"`
	ClientID   string      `json:"client_id"`
	Scope      string      `json:"scope"`
	Workspaces []Workspace `json:"workspaces"`
	CreatedAt  int64       `json:"created_at"`
	LastUsedAt int64       `json:"last_used_at"`
}

func (s *Service) Grants(ctx context.Context, p *policy.Principal) ([]Grant, error) {
	rows, err := s.st.R.QueryContext(ctx, "SELECT id, client_name, client_id, scope, created_at, last_used_at FROM oauth_grants WHERE user_id = ? AND revoked_at = 0 ORDER BY created_at DESC", p.UserID)
	if err != nil {
		return nil, err
	}
	out := []Grant{}
	for rows.Next() {
		var g Grant
		if err := rows.Scan(&g.ID, &g.Client, &g.ClientID, &g.Scope, &g.CreatedAt, &g.LastUsedAt); err != nil {
			rows.Close()
			return nil, err
		}
		g.Workspaces = []Workspace{}
		out = append(out, g)
	}
	rows.Close()
	for i := range out {
		wr, err := s.st.R.QueryContext(ctx, "SELECT w.id, w.name, w.personal FROM oauth_grant_workspaces gw JOIN workspaces w ON w.id = gw.workspace_id WHERE gw.grant_id = ? ORDER BY w.name", out[i].ID)
		if err != nil {
			return nil, err
		}
		for wr.Next() {
			var ws Workspace
			var personal int
			if wr.Scan(&ws.ID, &ws.Name, &personal) == nil {
				ws.Personal = personal == 1
				out[i].Workspaces = append(out[i].Workspaces, ws)
			}
		}
		wr.Close()
	}
	return out, nil
}

func (s *Service) Purge(ctx context.Context) {
	now := store.NowMS()
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM oauth_requests WHERE expires_at < ?", now)
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM oauth_codes WHERE expires_at < ?", now-ms(time.Hour))
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM oauth_tokens WHERE expires_at < ?", now)
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM oauth_grants WHERE revoked_at > 0 AND revoked_at < ?", now-ms(30*24*time.Hour))
}

type module struct{ s *Service }

func (s *Service) Module() app.Module { return module{s} }

func (module) ID() string { return "oauth" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	preflight := func(w http.ResponseWriter, _ *http.Request) {
		cors(w)
		w.WriteHeader(http.StatusNoContent)
	}
	r.HandleFunc("GET /.well-known/oauth-protected-resource", s.protectedResource)
	r.HandleFunc("GET /.well-known/oauth-protected-resource/mcp", s.protectedResource)
	r.HandleFunc("GET /.well-known/oauth-protected-resource/api", s.protectedResource)
	r.HandleFunc("GET /.well-known/oauth-authorization-server", s.serverMetadata)
	for _, p := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp", "/.well-known/oauth-protected-resource/api", "/.well-known/oauth-authorization-server", "/oauth/token", "/oauth/register", "/oauth/revoke"} {
		r.HandleFunc("OPTIONS "+p, preflight)
	}
	r.HandleFunc("POST /oauth/register", s.register)
	r.HandleFunc("GET /oauth/authorize", s.authorize)
	r.HandleFunc("POST /oauth/token", s.token)
	r.HandleFunc("POST /oauth/revoke", s.revoke)
	r.HandleFunc("POST /api/oauth/consent", s.auth.RequireSession(s.consent))
	r.HandleFunc("GET /api/me/grants", s.auth.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Grants(r.Context(), p)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"grants": list})
		return nil
	}))
	r.HandleFunc("DELETE /api/me/grants/{id}", s.auth.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var user string
		if err := s.st.R.QueryRowContext(r.Context(), "SELECT user_id FROM oauth_grants WHERE id = ? AND revoked_at = 0", r.PathValue("id")).Scan(&user); err != nil || user != p.UserID {
			return httpx.ErrNotFound
		}
		s.revokeGrant(r.Context(), r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	return nil
}
