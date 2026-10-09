package settings

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Kind int

const (
	Text Kind = iota
	Bool
	Int
	Secret
	Enum
)

type Def struct {
	Key     string
	Kind    Kind
	Default string
	Env     string
	Min     int
	Max     int
	Options []string
}

var Defs = []Def{
	{Key: "instance.name", Kind: Text, Default: "form.md"},
	{Key: "instance.public_url", Kind: Text},
	{Key: "net.trust_proxy", Kind: Bool, Default: "false", Env: "FORMMD_TRUST_PROXY"},
	{Key: "auth.session_days", Kind: Int, Default: "90", Min: 1, Max: 365},
	{Key: "auth.signup", Kind: Enum, Default: "invite", Options: []string{"invite", "domain", "open"}},
	{Key: "auth.signup_domains", Kind: Text},
	{Key: "workspaces.member_create", Kind: Bool, Default: "true"},
	{Key: "sso.google.client_id", Kind: Text},
	{Key: "sso.google.client_secret", Kind: Secret},
	{Key: "sso.microsoft.client_id", Kind: Text},
	{Key: "sso.microsoft.client_secret", Kind: Secret},
	{Key: "sso.microsoft.tenant", Kind: Text, Default: "common"},
	{Key: "sso.oidc.name", Kind: Text},
	{Key: "sso.oidc.issuer", Kind: Text},
	{Key: "sso.oidc.client_id", Kind: Text},
	{Key: "sso.oidc.client_secret", Kind: Secret},
	{Key: "sso.saml.name", Kind: Text},
	{Key: "sso.saml.metadata", Kind: Text},
	{Key: "sso.saml.email_attr", Kind: Text},
	{Key: "sso.saml.name_attr", Kind: Text},
	{Key: "auth.saml_only", Kind: Bool, Default: "false"},
	{Key: "mail.mode", Kind: Enum, Default: "off", Options: []string{"off", "smtp", "gmail", "graph"}},
	{Key: "mail.from", Kind: Text},
	{Key: "mail.smtp.host", Kind: Text},
	{Key: "mail.smtp.port", Kind: Int, Default: "587", Min: 1, Max: 65535},
	{Key: "mail.smtp.user", Kind: Text},
	{Key: "mail.smtp.password", Kind: Secret},
	{Key: "mail.smtp.security", Kind: Enum, Default: "starttls", Options: []string{"starttls", "tls", "none"}},
	{Key: "mail.oauth.client_id", Kind: Text},
	{Key: "mail.oauth.client_secret", Kind: Secret},
	{Key: "mail.oauth.refresh_token", Kind: Secret},
	{Key: "mail.oauth.account", Kind: Text},
	{Key: "trash.days", Kind: Int, Default: "30", Env: "FORMMD_TRASH_DAYS", Min: 0, Max: 3650},
	{Key: "files.max_upload_mb", Kind: Int, Default: "50", Min: 1, Max: 2048},
	{Key: "audit.retention_days", Kind: Int, Default: "0", Min: 0, Max: 36500},
	{Key: "update.check", Kind: Bool, Default: "true", Env: "FORMMD_UPDATE_CHECK"},
	{Key: "docs.print_page_size", Kind: Enum, Default: "a4", Options: []string{"a4", "letter"}},
	{Key: "https.auto", Kind: Bool, Default: "false"},
	{Key: "https.domain", Kind: Text},
	{Key: "tunnel.mode", Kind: Enum, Default: "off", Options: []string{"off", "quick", "token"}},
	{Key: "tunnel.token", Kind: Secret},
	{Key: "tunnel.autostart", Kind: Bool, Default: "false"},
	{Key: "history.days", Kind: Int, Default: "30", Min: 0, Max: 3650},
	{Key: "push.vapid_private", Kind: Secret},
	{Key: "notify.email_delay_minutes", Kind: Int, Default: "10", Min: 1, Max: 1440},
	{Key: "sites.key", Kind: Secret},
	{Key: "ai.enabled", Kind: Bool, Default: "true"},
	{Key: "ai.default_model", Kind: Text},
	{Key: "ai.transcribe_model", Kind: Text},
	{Key: "ai.user_limit_per_hour", Kind: Int, Default: "200", Min: 1, Max: 100000},
}

var byKey = func() map[string]Def {
	m := make(map[string]Def, len(Defs))
	for _, d := range Defs {
		m[d.Key] = d
	}
	return m
}()

const SecretSet = "\x00set"

func IsSecret(key string) bool { return byKey[key].Kind == Secret }

type Service struct {
	st    *store.Store
	box   *secrets.Box
	mu    sync.RWMutex
	value map[string]string
	subs  []func(keys []string)
}

func New(ctx context.Context, st *store.Store, box *secrets.Box) (*Service, error) {
	s := &Service{st: st, box: box, value: map[string]string{}}
	rows, err := st.R.QueryContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		s.value[k] = v
	}
	return s, rows.Err()
}

func (s *Service) raw(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.value[key]
	return v, ok
}

func (s *Service) Get(key string) string {
	d, known := byKey[key]
	if v, ok := s.raw(key); ok {
		if d.Kind == Secret {
			plain, err := s.box.Unseal(v)
			if err != nil {
				return ""
			}
			return plain
		}
		return v
	}
	if known && d.Env != "" {
		if v := strings.TrimSpace(os.Getenv(d.Env)); v != "" {
			return v
		}
	}
	return d.Default
}

func (s *Service) Bool(key string) bool {
	v, _ := strconv.ParseBool(s.Get(key))
	return v
}

func (s *Service) Int(key string) int {
	v, err := strconv.Atoi(s.Get(key))
	if err != nil {
		v, _ = strconv.Atoi(byKey[key].Default)
	}
	return v
}

func (s *Service) Has(key string) bool {
	return s.Get(key) != ""
}

func (s *Service) Snapshot(prefixes ...string) map[string]string {
	out := map[string]string{}
	for _, d := range Defs {
		if !matches(d.Key, prefixes) {
			continue
		}
		if d.Kind == Secret {
			if s.Get(d.Key) != "" {
				out[d.Key] = SecretSet
			} else {
				out[d.Key] = ""
			}
			continue
		}
		out[d.Key] = s.Get(d.Key)
	}
	return out
}

func matches(key string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

type FieldError struct {
	Key     string
	Problem string
}

func (e *FieldError) Error() string { return e.Key + ": " + e.Problem }

func normalize(d Def, v string) (string, error) {
	v = strings.TrimSpace(v)
	switch d.Kind {
	case Bool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return "", &FieldError{d.Key, "expected true or false"}
		}
		return strconv.FormatBool(b), nil
	case Int:
		n, err := strconv.Atoi(v)
		if err != nil || n < d.Min || n > d.Max {
			return "", &FieldError{d.Key, fmt.Sprintf("expected a whole number from %d to %d", d.Min, d.Max)}
		}
		return strconv.Itoa(n), nil
	case Enum:
		for _, o := range d.Options {
			if v == o {
				return v, nil
			}
		}
		return "", &FieldError{d.Key, "expected one of " + strings.Join(d.Options, ", ")}
	}
	if d.Key == "instance.public_url" && v != "" {
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", &FieldError{d.Key, "expected an absolute http or https address"}
		}
		return strings.TrimRight(u.Scheme+"://"+u.Host+u.Path, "/"), nil
	}
	return v, nil
}

func (s *Service) Set(ctx context.Context, changes map[string]string) error {
	stored := map[string]string{}
	for key, v := range changes {
		d, ok := byKey[key]
		if !ok {
			return &FieldError{key, "unknown setting"}
		}
		if d.Kind == Secret {
			if v == SecretSet {
				continue
			}
			stored[key] = s.box.Seal(strings.TrimSpace(v))
			continue
		}
		n, err := normalize(d, v)
		if err != nil {
			return err
		}
		stored[key] = n
	}
	if len(stored) == 0 {
		return nil
	}
	now := store.NowMS()
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		for k, v := range stored {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
				 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, k, v, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(stored))
	s.mu.Lock()
	for k, v := range stored {
		s.value[k] = v
		keys = append(keys, k)
	}
	subs := append([]func([]string){}, s.subs...)
	s.mu.Unlock()
	for _, fn := range subs {
		fn(keys)
	}
	return nil
}

func (s *Service) OnChange(fn func(keys []string)) {
	s.mu.Lock()
	s.subs = append(s.subs, fn)
	s.mu.Unlock()
}
