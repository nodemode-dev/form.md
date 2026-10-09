package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

const ssoCookie = "formmd_sso"

type providerConfig struct {
	ID, Name, Issuer, ClientID, ClientSecret string
	MultiTenantMicrosoft                     bool
}

func (s *Service) provider(id string) (providerConfig, bool) {
	get := s.settings.Get
	switch id {
	case "google":
		c := providerConfig{ID: id, Name: "Google", Issuer: "https://accounts.google.com", ClientID: get("sso.google.client_id"), ClientSecret: get("sso.google.client_secret")}
		return c, c.ClientID != "" && c.ClientSecret != ""
	case "microsoft":
		tenant := get("sso.microsoft.tenant")
		c := providerConfig{ID: id, Name: "Microsoft", Issuer: "https://login.microsoftonline.com/" + tenant + "/v2.0",
			ClientID: get("sso.microsoft.client_id"), ClientSecret: get("sso.microsoft.client_secret"),
			MultiTenantMicrosoft: tenant == "common" || tenant == "organizations" || tenant == "consumers"}
		return c, c.ClientID != "" && c.ClientSecret != ""
	case "oidc":
		c := providerConfig{ID: id, Name: get("sso.oidc.name"), Issuer: strings.TrimRight(get("sso.oidc.issuer"), "/"),
			ClientID: get("sso.oidc.client_id"), ClientSecret: get("sso.oidc.client_secret")}
		if c.Name == "" {
			c.Name = "Single sign-on"
		}
		return c, c.Issuer != "" && c.ClientID != ""
	}
	return providerConfig{}, false
}

func (s *Service) SSOProviders() []SSOProvider {
	out := []SSOProvider{}
	if s.samlConfigured() {
		out = append(out, SSOProvider{ID: "saml", Name: s.samlName()})
		if s.SAMLOnly() {
			return out
		}
	}
	for _, id := range []string{"google", "microsoft", "oidc"} {
		if c, ok := s.provider(id); ok {
			out = append(out, SSOProvider{ID: c.ID, Name: c.Name})
		}
	}
	return out
}

type discovery struct {
	Issuer        string `json:"issuer"`
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
	JWKS          string `json:"jwks_uri"`
	fetched       time.Time
	keys          map[string]crypto.PublicKey
}

var (
	discoMu    sync.Mutex
	discoCache = map[string]*discovery{}
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

func fetchJSON(ctx context.Context, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", u, res.Status)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(v)
}

func discover(ctx context.Context, issuer string, refresh bool) (*discovery, error) {
	discoMu.Lock()
	d := discoCache[issuer]
	discoMu.Unlock()
	if d != nil && !refresh && time.Since(d.fetched) < time.Hour {
		return d, nil
	}
	nd := &discovery{}
	if err := fetchJSON(ctx, issuer+"/.well-known/openid-configuration", nd); err != nil {
		return nil, err
	}
	var set struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := fetchJSON(ctx, nd.JWKS, &set); err != nil {
		return nil, err
	}
	nd.keys = map[string]crypto.PublicKey{}
	for _, k := range set.Keys {
		if pub, err := parseJWK(k); err == nil {
			nd.keys[k["kid"]] = pub
		}
	}
	nd.fetched = time.Now()
	discoMu.Lock()
	discoCache[issuer] = nd
	discoMu.Unlock()
	return nd, nil
}

func b64(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
}

func parseJWK(k map[string]string) (crypto.PublicKey, error) {
	switch k["kty"] {
	case "RSA":
		n, err := b64(k["n"])
		if err != nil {
			return nil, err
		}
		e, err := b64(k["e"])
		if err != nil {
			return nil, err
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}, nil
	case "EC":
		if k["crv"] != "P-256" {
			return nil, errors.New("unsupported curve")
		}
		x, err := b64(k["x"])
		if err != nil {
			return nil, err
		}
		y, err := b64(k["y"])
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}, nil
	}
	return nil, errors.New("unsupported key type")
}

type idClaims struct {
	Iss           string          `json:"iss"`
	Sub           string          `json:"sub"`
	Aud           json.RawMessage `json:"aud"`
	Exp           int64           `json:"exp"`
	Nonce         string          `json:"nonce"`
	Email         string          `json:"email"`
	EmailVerified any             `json:"email_verified"`
	Name          string          `json:"name"`
	Preferred     string          `json:"preferred_username"`
	Tid           string          `json:"tid"`
	EDOV          any             `json:"xms_edov"`
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	case float64:
		return t == 1
	}
	return false
}

func verifyIDToken(ctx context.Context, cfg providerConfig, d *discovery, raw, nonce string) (*idClaims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed id token")
	}
	hb, err := b64(parts[0])
	if err != nil {
		return nil, err
	}
	var hdr struct{ Alg, Kid string }
	if err := json.Unmarshal(hb, &hdr); err != nil {
		return nil, err
	}
	key, ok := d.keys[hdr.Kid]
	if !ok {
		if d, err = discover(ctx, cfg.Issuer, true); err != nil {
			return nil, err
		}
		if key, ok = d.keys[hdr.Kid]; !ok {
			return nil, errors.New("unknown signing key")
		}
	}
	sig, err := b64(parts[2])
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	switch hdr.Alg {
	case "RS256":
		pub, ok := key.(*rsa.PublicKey)
		if !ok || rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig) != nil {
			return nil, errors.New("bad signature")
		}
	case "ES256":
		pub, ok := key.(*ecdsa.PublicKey)
		if !ok || len(sig) != 64 || !ecdsa.Verify(pub, sum[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
			return nil, errors.New("bad signature")
		}
	default:
		return nil, fmt.Errorf("unsupported algorithm %q", hdr.Alg)
	}
	pb, err := b64(parts[1])
	if err != nil {
		return nil, err
	}
	var c idClaims
	if err := json.Unmarshal(pb, &c); err != nil {
		return nil, err
	}
	wantIss := d.Issuer
	if cfg.MultiTenantMicrosoft {
		wantIss = "https://login.microsoftonline.com/" + c.Tid + "/v2.0"
	}
	if c.Iss != wantIss {
		return nil, errors.New("issuer mismatch")
	}
	var auds []string
	if err := json.Unmarshal(c.Aud, &auds); err != nil {
		var one string
		if json.Unmarshal(c.Aud, &one) == nil {
			auds = []string{one}
		}
	}
	audOK := false
	for _, a := range auds {
		audOK = audOK || a == cfg.ClientID
	}
	if !audOK {
		return nil, errors.New("audience mismatch")
	}
	if time.Now().Unix() > c.Exp+60 {
		return nil, errors.New("token expired")
	}
	if !secrets.Equal(c.Nonce, nonce) {
		return nil, errors.New("nonce mismatch")
	}
	return &c, nil
}

type ssoState struct {
	Provider string `json:"p"`
	State    string `json:"s"`
	Verifier string `json:"v"`
	Nonce    string `json:"n"`
	Next     string `json:"x"`
	Exp      int64  `json:"e"`
}

func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/"
	}
	return next
}

func (s *Service) baseURL(r *http.Request) string {
	if pub := s.settings.Get("instance.public_url"); pub != "" {
		return pub
	}
	return httpx.RequestOrigin(r, s.TrustProxy())
}

func (s *Service) BaseURL(r *http.Request) string { return strings.TrimRight(s.baseURL(r), "/") }

func (s *Service) OriginAllowed(r *http.Request) bool { return s.originOK(r) }

func (s *Service) MCPChallenge(r *http.Request, extra string) string {
	h := `Bearer resource_metadata="` + s.BaseURL(r) + `/.well-known/oauth-protected-resource/mcp"`
	if extra != "" {
		h += ", " + extra
	}
	return h
}

func (s *Service) handleSSOStart(w http.ResponseWriter, r *http.Request) error {
	cfg, ok := s.provider(r.PathValue("provider"))
	if !ok {
		return httpx.ErrNotFound
	}
	if s.SAMLOnly() {
		http.Redirect(w, r, "/login?sso_error=saml_only", http.StatusFound)
		return nil
	}
	base := s.baseURL(r)
	if !strings.EqualFold(base, httpx.RequestOrigin(r, s.TrustProxy())) {
		http.Redirect(w, r, base+r.URL.RequestURI(), http.StatusFound)
		return nil
	}
	d, err := discover(r.Context(), cfg.Issuer, false)
	if err != nil {
		s.log.Warn("sso discovery failed", "provider", cfg.ID, "err", err)
		http.Redirect(w, r, "/login?sso_error=unavailable", http.StatusFound)
		return nil
	}
	st := ssoState{Provider: cfg.ID, State: secrets.Token("", 24), Verifier: secrets.Token("", 48), Nonce: secrets.Token("", 24),
		Next: safeNext(r.URL.Query().Get("next")), Exp: time.Now().Add(10 * time.Minute).Unix()}
	raw, _ := json.Marshal(st)
	http.SetCookie(w, &http.Cookie{Name: ssoCookie, Value: s.box.Seal(string(raw)), Path: "/api/auth/sso", MaxAge: 600,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookie(r)})
	challenge := sha256.Sum256([]byte(st.Verifier))
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {base + "/api/auth/sso/" + cfg.ID + "/callback"},
		"scope":                 {"openid email profile"},
		"state":                 {st.State},
		"nonce":                 {st.Nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	http.Redirect(w, r, d.Authorization+"?"+q.Encode(), http.StatusFound)
	return nil
}

func (s *Service) ssoFail(w http.ResponseWriter, r *http.Request, code string, err error) error {
	if err != nil {
		s.log.Warn("sso sign-in failed", "code", code, "err", err)
	}
	http.SetCookie(w, &http.Cookie{Name: ssoCookie, Value: "", Path: "/api/auth/sso", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/login?sso_error="+url.QueryEscape(code), http.StatusFound)
	return nil
}

func (s *Service) handleSSOCallback(w http.ResponseWriter, r *http.Request) error {
	cfg, ok := s.provider(r.PathValue("provider"))
	if !ok {
		return httpx.ErrNotFound
	}
	if s.SAMLOnly() {
		return s.ssoFail(w, r, "saml_only", nil)
	}
	q := r.URL.Query()
	if q.Get("error") != "" {
		return s.ssoFail(w, r, "denied", nil)
	}
	c, err := r.Cookie(ssoCookie)
	if err != nil {
		return s.ssoFail(w, r, "expired", nil)
	}
	plain, err := s.box.Unseal(c.Value)
	if err != nil {
		return s.ssoFail(w, r, "expired", err)
	}
	var st ssoState
	if err := json.Unmarshal([]byte(plain), &st); err != nil || st.Provider != cfg.ID || time.Now().Unix() > st.Exp || !secrets.Equal(st.State, q.Get("state")) {
		return s.ssoFail(w, r, "expired", err)
	}
	d, err := discover(r.Context(), cfg.Issuer, false)
	if err != nil {
		return s.ssoFail(w, r, "unavailable", err)
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {q.Get("code")},
		"redirect_uri":  {s.baseURL(r) + "/api/auth/sso/" + cfg.ID + "/callback"},
		"client_id":     {cfg.ClientID},
		"code_verifier": {st.Verifier},
	}
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, d.Token, strings.NewReader(form.Encode()))
	if err != nil {
		return s.ssoFail(w, r, "unavailable", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return s.ssoFail(w, r, "unavailable", err)
	}
	defer res.Body.Close()
	var tok struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&tok); err != nil || tok.IDToken == "" {
		return s.ssoFail(w, r, "exchange", fmt.Errorf("token endpoint: %s %s %v", res.Status, tok.Error, err))
	}
	claims, err := verifyIDToken(r.Context(), cfg, d, tok.IDToken, st.Nonce)
	if err != nil {
		return s.ssoFail(w, r, "invalid", err)
	}
	userID, code, err := s.ssoAccount(r.Context(), cfg, claims)
	if err != nil {
		return s.ssoFail(w, r, "internal", err)
	}
	if code != "" {
		return s.ssoFail(w, r, code, nil)
	}
	http.SetCookie(w, &http.Cookie{Name: ssoCookie, Value: "", Path: "/api/auth/sso", MaxAge: -1, HttpOnly: true})
	if err := s.StartSession(r.Context(), w, r, userID); err != nil {
		return err
	}
	http.Redirect(w, r, st.Next, http.StatusFound)
	return nil
}

func (s *Service) ssoAccount(ctx context.Context, cfg providerConfig, c *idClaims) (string, string, error) {
	var userID string
	var disabled int
	err := s.st.R.QueryRowContext(ctx, `SELECT u.id, u.disabled FROM sso_identities i JOIN users u ON u.id = i.user_id
		WHERE i.provider = ? AND i.subject = ?`, cfg.ID, c.Sub).Scan(&userID, &disabled)
	if err == nil {
		if disabled == 1 {
			return "", "disabled", nil
		}
		return userID, "", nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}
	email := c.Email
	if email == "" && cfg.ID == "microsoft" {
		email = c.Preferred
	}
	email, err = NormalizeEmail(email)
	if err != nil {
		return "", "no_email", nil
	}
	verified := truthy(c.EmailVerified)
	if cfg.ID == "microsoft" {
		verified = truthy(c.EDOV)
	}
	var localVerified int
	err = s.st.R.QueryRowContext(ctx, "SELECT id, disabled, email_verified FROM users WHERE email = ?", email).Scan(&userID, &disabled, &localVerified)
	switch {
	case err == nil:
		if !verified || localVerified == 0 {
			return "", "unverified", nil
		}
		if disabled == 1 {
			return "", "disabled", nil
		}
	case errors.Is(err, sql.ErrNoRows):
		if !s.SignupAllowed(email) {
			return "", "signup_closed", nil
		}
		name := c.Name
		if strings.TrimSpace(name) == "" {
			name, _, _ = strings.Cut(email, "@")
		}
		name, _ = cleanName(name)
		err = s.st.Tx(ctx, func(tx *sql.Tx) error {
			userID, err = s.createUser(ctx, tx, email, name, "", "member", verified)
			return err
		})
		if err != nil {
			return "", "", err
		}
	default:
		return "", "", err
	}
	_, err = s.st.W.ExecContext(ctx, `INSERT INTO sso_identities (provider, subject, user_id, email, created_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(provider, subject) DO NOTHING`, cfg.ID, c.Sub, userID, email, store.NowMS())
	return userID, "", err
}
