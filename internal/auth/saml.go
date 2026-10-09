package auth

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	samlCookie  = "formmd_saml"
	nameIDEmail = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
)

var (
	samlSeenMu sync.Mutex
	samlSeen   = map[string]time.Time{}
)

func (s *Service) samlConfigured() bool {
	return strings.TrimSpace(s.settings.Get("sso.saml.metadata")) != ""
}

func (s *Service) SAMLOnly() bool {
	return s.samlConfigured() && s.settings.Bool("auth.saml_only")
}

func (s *Service) SAMLName() string { return s.samlName() }

func (s *Service) samlName() string {
	if n := strings.TrimSpace(s.settings.Get("sso.saml.name")); n != "" {
		return n
	}
	return "SAML single sign-on"
}

func (s *Service) samlSP(base string) (*saml2.SAMLServiceProvider, error) {
	var md types.EntityDescriptor
	if err := xml.Unmarshal([]byte(s.settings.Get("sso.saml.metadata")), &md); err != nil {
		return nil, err
	}
	if md.IDPSSODescriptor == nil {
		return nil, errors.New("saml: the metadata has no IDPSSODescriptor")
	}
	store := &dsig.MemoryX509CertificateStore{}
	for _, kd := range md.IDPSSODescriptor.KeyDescriptors {
		if kd.Use != "" && kd.Use != "signing" {
			continue
		}
		for _, xc := range kd.KeyInfo.X509Data.X509Certificates {
			raw, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(xc.Data), ""))
			if err != nil {
				return nil, err
			}
			cert, err := x509.ParseCertificate(raw)
			if err != nil {
				return nil, err
			}
			store.Roots = append(store.Roots, cert)
		}
	}
	if len(store.Roots) == 0 {
		return nil, errors.New("saml: the metadata has no signing certificate")
	}
	sso, binding := "", ""
	for _, ep := range md.IDPSSODescriptor.SingleSignOnServices {
		if ep.Binding == saml2.BindingHttpRedirect || (sso == "" && ep.Binding == saml2.BindingHttpPost) {
			sso, binding = ep.Location, ep.Binding
		}
	}
	if sso == "" {
		return nil, errors.New("saml: the metadata has no single sign-on address")
	}
	return &saml2.SAMLServiceProvider{
		IdentityProviderSSOURL:      sso,
		IdentityProviderSSOBinding:  binding,
		IdentityProviderIssuer:      md.EntityID,
		ServiceProviderIssuer:       base + "/api/auth/saml/metadata",
		AssertionConsumerServiceURL: base + "/api/auth/saml/acs",
		AudienceURI:                 base + "/api/auth/saml/metadata",
		IDPCertificateStore:         store,
		NameIdFormat:                nameIDEmail,
		AllowMissingAttributes:      true,
	}, nil
}

func (s *Service) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) error {
	base := s.baseURL(r)
	entity, acs := html.EscapeString(base+"/api/auth/saml/metadata"), html.EscapeString(base+"/api/auth/saml/acs")
	body := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="` + entity + `">` +
		`<md:SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">` +
		`<md:NameIDFormat>` + nameIDEmail + `</md:NameIDFormat>` +
		`<md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="` + acs + `" index="1" isDefault="true"/>` +
		`</md:SPSSODescriptor></md:EntityDescriptor>`
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	_, _ = w.Write([]byte(body))
	return nil
}

type samlState struct {
	Request string `json:"r"`
	Next    string `json:"x"`
	Exp     int64  `json:"e"`
}

func (s *Service) handleSAMLStart(w http.ResponseWriter, r *http.Request) error {
	if !s.samlConfigured() {
		return httpx.ErrNotFound
	}
	base := s.baseURL(r)
	if !strings.EqualFold(base, httpx.RequestOrigin(r, s.TrustProxy())) {
		http.Redirect(w, r, base+r.URL.RequestURI(), http.StatusFound)
		return nil
	}
	sp, err := s.samlSP(base)
	if err != nil {
		s.log.Warn("saml: configuration", "err", err)
		http.Redirect(w, r, "/login?sso_error=unavailable", http.StatusFound)
		return nil
	}
	doc, err := sp.BuildAuthRequestDocument()
	if err != nil {
		return err
	}
	reqID := doc.Root().SelectAttrValue("ID", "")
	st := samlState{Request: reqID, Next: safeNext(r.URL.Query().Get("next")), Exp: time.Now().Add(10 * time.Minute).Unix()}
	raw, _ := json.Marshal(st)
	sealed := s.box.Seal(string(raw))
	same := http.SameSiteLaxMode
	if s.secureCookie(r) {
		same = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{Name: samlCookie, Value: reqID, Path: "/api/auth/saml", MaxAge: 600, HttpOnly: true, SameSite: same, Secure: s.secureCookie(r)})
	if sp.IdentityProviderSSOBinding == saml2.BindingHttpPost {
		body, err := sp.BuildAuthBodyPostFromDocument(sealed, doc)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; form-action https: http:")
		_, _ = w.Write(body)
		return nil
	}
	u, err := sp.BuildAuthURLFromDocument(sealed, doc)
	if err != nil {
		return err
	}
	http.Redirect(w, r, u, http.StatusFound)
	return nil
}

func firstValue(info *saml2.AssertionInfo, names ...string) string {
	for _, n := range names {
		if n == "" {
			continue
		}
		if v := strings.TrimSpace(info.Values.Get(n)); v != "" {
			return v
		}
	}
	return ""
}

func (s *Service) remember(id string, until time.Time) bool {
	samlSeenMu.Lock()
	defer samlSeenMu.Unlock()
	now := time.Now()
	for k, t := range samlSeen {
		if now.After(t) {
			delete(samlSeen, k)
		}
	}
	if _, ok := samlSeen[id]; ok {
		return false
	}
	samlSeen[id] = until
	return true
}

func (s *Service) handleSAMLACS(w http.ResponseWriter, r *http.Request) error {
	if !s.samlConfigured() {
		return httpx.ErrNotFound
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		return s.ssoFail(w, r, "invalid", err)
	}
	sp, err := s.samlSP(s.baseURL(r))
	if err != nil {
		return s.ssoFail(w, r, "unavailable", err)
	}
	info, err := sp.RetrieveAssertionInfo(r.PostForm.Get("SAMLResponse"))
	if err != nil {
		return s.ssoFail(w, r, "invalid", err)
	}
	if info.WarningInfo == nil || info.WarningInfo.InvalidTime || info.WarningInfo.NotInAudience {
		return s.ssoFail(w, r, "invalid", errors.New("saml: assertion outside its validity window or audience"))
	}
	next := "/"
	inResponseTo := ""
	if len(info.Assertions) > 0 {
		a := info.Assertions[0]
		if a.Subject != nil && a.Subject.SubjectConfirmation != nil && a.Subject.SubjectConfirmation.SubjectConfirmationData != nil {
			inResponseTo = a.Subject.SubjectConfirmation.SubjectConfirmationData.InResponseTo
		}
		until := time.Now().Add(10 * time.Minute)
		if a.Conditions != nil && a.Conditions.NotOnOrAfter != "" {
			if t, err := time.Parse(time.RFC3339, a.Conditions.NotOnOrAfter); err == nil {
				until = t
			}
		}
		if a.ID == "" || !s.remember(a.ID, until) {
			return s.ssoFail(w, r, "invalid", errors.New("saml: assertion replayed"))
		}
	}
	if relay := r.PostForm.Get("RelayState"); relay != "" {
		plain, err := s.box.Unseal(relay)
		var st samlState
		if err != nil || json.Unmarshal([]byte(plain), &st) != nil || time.Now().Unix() > st.Exp || (inResponseTo != "" && inResponseTo != st.Request) {
			return s.ssoFail(w, r, "expired", err)
		}
		if c, err := r.Cookie(samlCookie); err == nil && c.Value != st.Request {
			return s.ssoFail(w, r, "expired", errors.New("saml: the response belongs to another sign-in"))
		}
		next = st.Next
	} else if inResponseTo != "" {
		return s.ssoFail(w, r, "expired", errors.New("saml: response without its state"))
	}
	email := firstValue(info, s.settings.Get("sso.saml.email_attr"), "email", "mail", "Email", "emailAddress",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress", "urn:oid:0.9.2342.19200300.100.1.3")
	if email == "" && strings.Contains(info.NameID, "@") {
		email = info.NameID
	}
	name := firstValue(info, s.settings.Get("sso.saml.name_attr"), "displayName", "name", "cn",
		"http://schemas.microsoft.com/identity/claims/displayname", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name", "urn:oid:2.16.840.1.113730.3.1.241")
	if name == "" {
		name = strings.TrimSpace(firstValue(info, "givenName", "firstName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname") + " " +
			firstValue(info, "sn", "lastName", "surname", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"))
	}
	userID, code, err := s.samlAccount(r.Context(), info.NameID, email, name)
	if err != nil {
		return s.ssoFail(w, r, "internal", err)
	}
	if code != "" {
		return s.ssoFail(w, r, code, nil)
	}
	http.SetCookie(w, &http.Cookie{Name: samlCookie, Value: "", Path: "/api/auth/saml", MaxAge: -1, HttpOnly: true})
	if err := s.StartSession(r.Context(), w, r, userID); err != nil {
		return err
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
	return nil
}

func (s *Service) samlAccount(ctx context.Context, subject, rawEmail, name string) (string, string, error) {
	var userID string
	var disabled int
	err := s.st.R.QueryRowContext(ctx, `SELECT u.id, u.disabled FROM sso_identities i JOIN users u ON u.id = i.user_id WHERE i.provider = 'saml' AND i.subject = ?`, subject).Scan(&userID, &disabled)
	if err == nil {
		if disabled == 1 {
			return "", "disabled", nil
		}
		return userID, "", nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return "", "no_email", nil
	}
	err = s.st.R.QueryRowContext(ctx, "SELECT id, disabled FROM users WHERE email = ?", email).Scan(&userID, &disabled)
	switch {
	case err == nil:
		if disabled == 1 {
			return "", "disabled", nil
		}
	case errors.Is(err, sql.ErrNoRows):
		if !s.SignupAllowed(email) {
			return "", "signup_closed", nil
		}
		if strings.TrimSpace(name) == "" {
			name, _, _ = strings.Cut(email, "@")
		}
		name, _ = cleanName(name)
		err = s.st.Tx(ctx, func(tx *sql.Tx) error {
			userID, err = s.createUser(ctx, tx, email, name, "", "member", true)
			return err
		})
		if err != nil {
			return "", "", err
		}
	default:
		return "", "", err
	}
	_, err = s.st.W.ExecContext(ctx, `INSERT INTO sso_identities (provider, subject, user_id, email, created_at) VALUES ('saml', ?, ?, ?, ?)
		ON CONFLICT(provider, subject) DO NOTHING`, subject, userID, email, store.NowMS())
	return userID, "", err
}

var errSAMLOnly = httpx.Fail(http.StatusForbidden, "saml_required", "this instance signs people in through SAML single sign-on")
