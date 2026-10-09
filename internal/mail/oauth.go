package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/settings"
)

type Provider struct {
	AuthURL  string
	TokenURL string
	Scope    string
	SendURL  string
	Extra    url.Values
}

var Providers = map[string]*Provider{
	"gmail": {
		AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		Scope:    "openid email https://www.googleapis.com/auth/gmail.send",
		SendURL:  "https://gmail.googleapis.com/gmail/v1/users/me/messages/send",
		Extra:    url.Values{"access_type": {"offline"}, "prompt": {"consent"}},
	},
	"graph": {
		AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		Scope:    "openid email offline_access https://graph.microsoft.com/Mail.Send",
		SendURL:  "https://graph.microsoft.com/v1.0/me/sendMail",
	},
}

type OAuth struct {
	set    *settings.Service
	client *http.Client

	mu      sync.Mutex
	access  map[string]string
	expires map[string]time.Time
	pending map[string]pendingConnect
}

type pendingConnect struct {
	mode, verifier, redirect string
	at                       time.Time
}

func NewOAuth(set *settings.Service) *OAuth {
	return &OAuth{set: set, client: &http.Client{Timeout: 30 * time.Second}, access: map[string]string{}, expires: map[string]time.Time{}, pending: map[string]pendingConnect{}}
}

func (o *OAuth) Attach(m *Mailer) {
	for mode := range Providers {
		m.Register(mode, sender{o, mode})
	}
}

type sender struct {
	o    *OAuth
	mode string
}

func (s sender) Send(ctx context.Context, from, to string, msg []byte) error {
	return s.o.send(ctx, s.mode, msg)
}

func random(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (o *OAuth) Start(mode, redirect string) (string, error) {
	p := Providers[mode]
	if p == nil {
		return "", errors.New("unknown mail mode")
	}
	id := o.set.Get("mail.oauth.client_id")
	if id == "" || o.set.Get("mail.oauth.client_secret") == "" {
		return "", errors.New("save the client ID and secret first")
	}
	state, verifier := random(24), random(32)
	sum := sha256.Sum256([]byte(verifier))
	o.mu.Lock()
	for k, v := range o.pending {
		if time.Since(v.at) > 15*time.Minute {
			delete(o.pending, k)
		}
	}
	o.pending[state] = pendingConnect{mode: mode, verifier: verifier, redirect: redirect, at: time.Now()}
	o.mu.Unlock()
	q := url.Values{
		"client_id":             {id},
		"redirect_uri":          {redirect},
		"response_type":         {"code"},
		"scope":                 {p.Scope},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	for k, v := range p.Extra {
		q[k] = v
	}
	return p.AuthURL + "?" + q.Encode(), nil
}

type tokenReply struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
	Expires int    `json:"expires_in"`
	IDToken string `json:"id_token"`
	Error   string `json:"error"`
	Detail  string `json:"error_description"`
}

func (o *OAuth) token(ctx context.Context, p *Provider, form url.Values) (*tokenReply, error) {
	form.Set("client_id", o.set.Get("mail.oauth.client_id"))
	form.Set("client_secret", o.set.Get("mail.oauth.client_secret"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var t tokenReply
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t); err != nil {
		return nil, fmt.Errorf("token endpoint answered %d", resp.StatusCode)
	}
	if t.Error != "" || resp.StatusCode != http.StatusOK || t.Access == "" {
		return nil, fmt.Errorf("token endpoint: %s %s", t.Error, t.Detail)
	}
	return &t, nil
}

func (o *OAuth) Finish(ctx context.Context, state, code string) (string, error) {
	o.mu.Lock()
	pc, ok := o.pending[state]
	delete(o.pending, state)
	o.mu.Unlock()
	if !ok || time.Since(pc.at) > 15*time.Minute {
		return "", errors.New("this sign-in expired; start again from Instance settings")
	}
	p := Providers[pc.mode]
	t, err := o.token(ctx, p, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {pc.redirect}, "code_verifier": {pc.verifier}})
	if err != nil {
		return "", err
	}
	if t.Refresh == "" {
		return "", errors.New("the provider gave no refresh token; remove the app's access in your account and connect again")
	}
	account := claimEmail(t.IDToken)
	if err := o.set.Set(ctx, map[string]string{"mail.mode": pc.mode, "mail.oauth.refresh_token": t.Refresh, "mail.oauth.account": account}); err != nil {
		return "", err
	}
	o.mu.Lock()
	o.access[pc.mode], o.expires[pc.mode] = t.Access, time.Now().Add(time.Duration(t.Expires)*time.Second)
	o.mu.Unlock()
	return account, nil
}

func claimEmail(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var c struct {
		Email string `json:"email"`
		UPN   string `json:"preferred_username"`
	}
	if json.Unmarshal(raw, &c) != nil {
		return ""
	}
	if c.Email != "" {
		return c.Email
	}
	return c.UPN
}

func (o *OAuth) accessToken(ctx context.Context, mode string) (string, error) {
	o.mu.Lock()
	tok, exp := o.access[mode], o.expires[mode]
	o.mu.Unlock()
	if tok != "" && time.Until(exp) > time.Minute {
		return tok, nil
	}
	refresh := o.set.Get("mail.oauth.refresh_token")
	if refresh == "" {
		return "", errors.New("mail account not connected")
	}
	t, err := o.token(ctx, Providers[mode], url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}})
	if err != nil {
		return "", err
	}
	if t.Refresh != "" && t.Refresh != refresh {
		if err := o.set.Set(ctx, map[string]string{"mail.oauth.refresh_token": t.Refresh}); err != nil {
			return "", err
		}
	}
	o.mu.Lock()
	o.access[mode], o.expires[mode] = t.Access, time.Now().Add(time.Duration(t.Expires)*time.Second)
	o.mu.Unlock()
	return t.Access, nil
}

func (o *OAuth) send(ctx context.Context, mode string, msg []byte) error {
	p := Providers[mode]
	tok, err := o.accessToken(ctx, mode)
	if err != nil {
		return err
	}
	var body io.Reader
	ctype := "text/plain"
	if mode == "gmail" {
		b, _ := json.Marshal(map[string]string{"raw": base64.URLEncoding.EncodeToString(msg)})
		body, ctype = bytes.NewReader(b), "application/json"
	} else {
		body = strings.NewReader(base64.StdEncoding.EncodeToString(msg))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.SendURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", ctype)
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if resp.StatusCode == http.StatusUnauthorized {
			o.mu.Lock()
			delete(o.access, mode)
			o.mu.Unlock()
		}
		return fmt.Errorf("%s answered %d: %s", mode, resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}
