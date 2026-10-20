package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

var Colors = []string{"violet", "blue", "cyan", "teal", "green", "lime", "amber", "orange", "rose", "pink"}

type Prefs struct {
	Lang       string      `json:"lang"`
	Region     string      `json:"region"`
	TimeZone   string      `json:"time_zone"`
	HourCycle  string      `json:"hour_cycle"`
	WeekStart  string      `json:"week_start"`
	NotesMode  bool        `json:"notes_mode"`
	SystemFont bool        `json:"system_font"`
	Appearance *Appearance `json:"appearance,omitempty"`
}

type User struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Color         string `json:"color"`
	Avatar        string `json:"avatar"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"email_verified"`
	Disabled      bool   `json:"disabled"`
	TOTPEnabled   bool   `json:"totp_enabled"`
	Prefs         Prefs  `json:"prefs"`
	CreatedAt     int64  `json:"created_at"`
}

const userColumns = `id, email, name, color, avatar, role, email_verified, disabled, totp_enabled,
	lang, region, time_zone, hour_cycle, week_start, notes_mode, system_font, appearance, created_at`

func scanUser(row interface{ Scan(...any) error }) (*User, error) {
	u := &User{}
	var verified, disabled, totp, notes, sysFont int
	var look string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Color, &u.Avatar, &u.Role, &verified, &disabled, &totp,
		&u.Prefs.Lang, &u.Prefs.Region, &u.Prefs.TimeZone, &u.Prefs.HourCycle, &u.Prefs.WeekStart, &notes, &sysFont, &look, &u.CreatedAt)
	if err != nil {
		return nil, store.NoRows(err)
	}
	u.EmailVerified, u.Disabled, u.TOTPEnabled = verified == 1, disabled == 1, totp == 1
	u.Prefs.NotesMode, u.Prefs.SystemFont = notes == 1, sysFont == 1
	u.Prefs.Appearance = decodeAppearance(look)
	return u, nil
}

func (s *Service) User(ctx context.Context, id string) (*User, error) {
	return scanUser(s.st.R.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id = ?", id))
}

func (s *Service) UserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.st.R.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE email = ?", email))
}

func (s *Service) SetupRequired(ctx context.Context) (bool, error) {
	var n int
	err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&n)
	return n == 0, err
}

func NormalizeEmail(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	a, err := mail.ParseAddress(raw)
	if err != nil || a.Address != raw || !strings.Contains(raw, "@") || len(raw) > 254 {
		return "", httpx.Fail(http.StatusBadRequest, "bad_email", "enter a valid email address")
	}
	return strings.ToLower(raw), nil
}

func cleanName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return "", httpx.Fail(http.StatusBadRequest, "bad_name", "enter a name of up to 80 characters")
	}
	return name, nil
}

func randomColor() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(Colors))))
	return Colors[n.Int64()]
}

var errPassword = httpx.Fail(http.StatusBadRequest, "weak_password", "use at least 8 characters")

func (s *Service) createUser(ctx context.Context, tx *sql.Tx, email, name, password, role string, verified bool) (string, error) {
	var hash string
	if password == "" {
		hash = "!" + secrets.Token("", 24)
	} else {
		h, err := HashPassword(ctx, password)
		if err != nil {
			return "", err
		}
		hash = h
	}
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists); err != nil {
		return "", err
	}
	if exists > 0 {
		return "", httpx.Fail(http.StatusConflict, "email_taken", "an account with this email already exists")
	}
	id := ids.New()
	now := store.NowMS()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, email, name, color, password_hash, role, email_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, email, name, randomColor(), hash, role, store.Bool(verified), now, now); err != nil {
		return "", err
	}
	if s.accounts != nil {
		if err := s.accounts.Provision(ctx, tx, id, name, role == "owner"); err != nil {
			return "", err
		}
	}
	return id, nil
}

type credentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Service) CreateAccount(ctx context.Context, tx *sql.Tx, email, name, password, role string) (string, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		return "", err
	}
	if name, err = cleanName(name); err != nil {
		return "", err
	}
	if !ValidPassword(password) {
		return "", httpx.Fail(http.StatusBadRequest, "weak_password", "use at least 8 characters")
	}
	return s.createUser(ctx, tx, email, name, password, role, true)
}

func (s *Service) handleSetup(w http.ResponseWriter, r *http.Request) error {
	var in credentials
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	need, err := s.SetupRequired(r.Context())
	if err != nil {
		return err
	}
	if !need {
		return httpx.Fail(http.StatusConflict, "already_set_up", "this instance already has an owner")
	}
	name, err := cleanName(in.Name)
	if err != nil {
		return err
	}
	email, err := NormalizeEmail(in.Email)
	if err != nil {
		return err
	}
	if !ValidPassword(in.Password) {
		return errPassword
	}
	var id string
	err = s.st.Tx(r.Context(), func(tx *sql.Tx) error {
		id, err = s.createUser(r.Context(), tx, email, name, in.Password, "owner", true)
		return err
	})
	if err != nil {
		return err
	}
	if err := s.StartSession(r.Context(), w, r, id); err != nil {
		return err
	}
	return s.writeMe(w, r, id)
}

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
	Recovery string `json:"recovery"`
}

var errBadCredentials = httpx.Fail(http.StatusUnauthorized, "bad_credentials", "email or password is wrong")

func (s *Service) checkSecondFactor(ctx context.Context, userID, code, recovery string) (bool, error) {
	if code != "" {
		var secret string
		var last int64
		if err := s.st.R.QueryRowContext(ctx, "SELECT totp_secret, totp_last_step FROM users WHERE id = ?", userID).Scan(&secret, &last); err != nil {
			return false, err
		}
		plain, err := s.box.Unseal(secret)
		if err != nil {
			return false, err
		}
		step, ok := CheckTOTP(plain, code, last, time.Now())
		if !ok {
			return false, nil
		}
		res, err := s.st.W.ExecContext(ctx, "UPDATE users SET totp_last_step = ? WHERE id = ? AND totp_last_step < ?", step, userID, step)
		if err != nil {
			return false, err
		}
		n, _ := res.RowsAffected()
		return n == 1, nil
	}
	if recovery != "" {
		res, err := s.st.W.ExecContext(ctx, "UPDATE recovery_codes SET used_at = ? WHERE user_id = ? AND code_hash = ? AND used_at = 0",
			store.NowMS(), userID, secrets.Hash(NormalizeRecovery(recovery)))
		if err != nil {
			return false, err
		}
		n, _ := res.RowsAffected()
		return n == 1, nil
	}
	return false, nil
}

func (s *Service) VerifyLogin(ctx context.Context, r *http.Request, email, password, code, recovery string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var id, hash, role string
	var disabled, totp int
	err := s.st.R.QueryRowContext(ctx, "SELECT id, password_hash, disabled, totp_enabled, role FROM users WHERE email = ?", email).
		Scan(&id, &hash, &disabled, &totp, &role)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if s.SAMLOnly() && role != "owner" {
		return "", errSAMLOnly
	}
	if strings.HasPrefix(hash, "!") {
		hash = ""
	}
	ok, err := VerifyPassword(ctx, hash, password)
	if err != nil {
		return "", err
	}
	if !ok || id == "" {
		s.authFailure(r, "password")
		return "", errBadCredentials
	}
	if disabled == 1 {
		return "", httpx.Fail(http.StatusForbidden, "account_disabled", "this account is deactivated")
	}
	if totp == 1 {
		if code == "" && recovery == "" {
			return "", httpx.Fail(http.StatusUnauthorized, "totp_required", "enter the code from your authenticator app")
		}
		ok, err := s.checkSecondFactor(ctx, id, code, recovery)
		if err != nil {
			return "", err
		}
		if !ok {
			s.authFailure(r, "totp")
			return "", httpx.Fail(http.StatusUnauthorized, "bad_totp", "that code did not work")
		}
	}
	return id, nil
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if !s.loginLimit.Allow(s.ClientIP(r)) {
		return httpx.Fail(http.StatusTooManyRequests, "rate_limited", "too many attempts, wait a minute")
	}
	var in loginInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	id, err := s.VerifyLogin(r.Context(), r, in.Email, in.Password, in.Code, in.Recovery)
	if err != nil {
		return err
	}
	if err := s.StartSession(r.Context(), w, r, id); err != nil {
		return err
	}
	return s.writeMe(w, r, id)
}

func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) error {
	s.EndSession(r.Context(), w, r)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) SignupAllowed(email string) bool {
	switch s.settings.Get("auth.signup") {
	case "open":
		return true
	case "domain":
		_, domain, ok := strings.Cut(email, "@")
		if !ok {
			return false
		}
		for _, d := range strings.FieldsFunc(s.settings.Get("auth.signup_domains"), func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
			if strings.EqualFold(strings.TrimPrefix(d, "@"), domain) {
				return true
			}
		}
	}
	return false
}

func (s *Service) handleSignup(w http.ResponseWriter, r *http.Request) error {
	if !s.loginLimit.Allow(s.ClientIP(r)) {
		return httpx.Fail(http.StatusTooManyRequests, "rate_limited", "too many attempts, wait a minute")
	}
	if s.SAMLOnly() {
		return errSAMLOnly
	}
	var in credentials
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	name, err := cleanName(in.Name)
	if err != nil {
		return err
	}
	email, err := NormalizeEmail(in.Email)
	if err != nil {
		return err
	}
	if !s.SignupAllowed(email) {
		return httpx.Fail(http.StatusForbidden, "signup_closed", "sign-up is by invitation on this instance")
	}
	if !ValidPassword(in.Password) {
		return errPassword
	}
	var id string
	err = s.st.Tx(r.Context(), func(tx *sql.Tx) error {
		id, err = s.createUser(r.Context(), tx, email, name, in.Password, "member", false)
		return err
	})
	if err != nil {
		return err
	}
	if err := s.StartSession(r.Context(), w, r, id); err != nil {
		return err
	}
	return s.writeMe(w, r, id)
}

type SSOProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Service) handlePolicy(w http.ResponseWriter, r *http.Request) error {
	need, err := s.SetupRequired(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"setup_required": need,
		"signup":         map[bool]string{true: "open", false: "invite"}[s.settings.Get("auth.signup") != "invite"],
		"sso":            s.SSOProviders(),
		"saml_only":      s.SAMLOnly(),
		"instance_name":  s.settings.Get("instance.name"),
		"desktop":        s.Desktop(),
	})
	return nil
}

func (s *Service) Me(ctx context.Context, p *policy.Principal) (map[string]any, error) {
	u, err := s.User(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"user":       u,
		"credential": map[string]any{"kind": p.Kind, "scope": p.Scope},
		"instance":   map[string]any{"name": s.settings.Get("instance.name"), "desktop": s.Desktop()},
	}
	if s.meExtras != nil {
		extra, err := s.meExtras(ctx, p)
		if err != nil {
			return nil, err
		}
		for k, v := range extra {
			out[k] = v
		}
	}
	return out, nil
}

func (s *Service) writeMe(w http.ResponseWriter, r *http.Request, userID string) error {
	p, err := s.principalForUser(r.Context(), userID)
	if err != nil {
		return err
	}
	p.Kind, p.Scope, p.AllWorkspaces = policy.KindSession, "write", true
	me, err := s.Me(r.Context(), p)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, me)
	return nil
}

type meUpdate struct {
	Name   *string `json:"name"`
	Color  *string `json:"color"`
	Avatar *string `json:"avatar"`
	Prefs  *Prefs  `json:"prefs"`
}

var validHourCycle = map[string]bool{"": true, "h12": true, "h23": true}
var validWeekStart = map[string]bool{"": true, "monday": true, "sunday": true, "saturday": true}

func (s *Service) handleUpdateMe(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in meUpdate
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	sets, args := []string{}, []any{}
	if in.Name != nil {
		name, err := cleanName(*in.Name)
		if err != nil {
			return err
		}
		sets, args = append(sets, "name = ?"), append(args, name)
	}
	if in.Color != nil {
		ok := false
		for _, c := range Colors {
			ok = ok || c == *in.Color
		}
		if !ok {
			return httpx.Fail(http.StatusBadRequest, "bad_color", "unknown colour")
		}
		sets, args = append(sets, "color = ?"), append(args, *in.Color)
	}
	if in.Avatar != nil {
		v := strings.TrimSpace(*in.Avatar)
		if v != "" && (s.avatarCheck == nil || !s.avatarCheck(r.Context(), v, p.UserID)) {
			return httpx.Fail(http.StatusBadRequest, "bad_avatar", "upload the picture first")
		}
		sets, args = append(sets, "avatar = ?"), append(args, v)
	}
	if pr := in.Prefs; pr != nil {
		if !validHourCycle[pr.HourCycle] || !validWeekStart[pr.WeekStart] || len(pr.Lang) > 16 || len(pr.Region) > 16 || len(pr.TimeZone) > 64 {
			return httpx.Fail(http.StatusBadRequest, "bad_prefs", "unsupported preference value")
		}
		if pr.TimeZone != "" {
			if _, err := time.LoadLocation(pr.TimeZone); err != nil {
				return httpx.Fail(http.StatusBadRequest, "bad_time_zone", "unknown time zone")
			}
		}
		sets = append(sets, "lang = ?", "region = ?", "time_zone = ?", "hour_cycle = ?", "week_start = ?", "notes_mode = ?", "system_font = ?")
		args = append(args, pr.Lang, pr.Region, pr.TimeZone, pr.HourCycle, pr.WeekStart, store.Bool(pr.NotesMode), store.Bool(pr.SystemFont))
		if pr.Appearance != nil {
			look, err := pr.Appearance.encode()
			if err != nil {
				return err
			}
			sets, args = append(sets, "appearance = ?"), append(args, look)
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at = ?")
		args = append(args, store.NowMS(), p.UserID)
		if _, err := s.st.W.ExecContext(r.Context(), "UPDATE users SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
			return err
		}
	}
	return s.writeMe(w, r, p.UserID)
}

type passwordChange struct {
	Current string `json:"current"`
	Next    string `json:"next"`
}

func (s *Service) checkCurrentPassword(ctx context.Context, userID, password string) error {
	var hash string
	if err := s.st.R.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = ?", userID).Scan(&hash); err != nil {
		return err
	}
	if strings.HasPrefix(hash, "!") {
		hash = ""
	}
	ok, err := VerifyPassword(ctx, hash, password)
	if err != nil && hash != "" {
		return err
	}
	if !ok {
		return httpx.Fail(http.StatusForbidden, "bad_password", "the current password is wrong")
	}
	return nil
}

func (s *Service) handlePassword(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in passwordChange
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if err := s.checkCurrentPassword(r.Context(), p.UserID, in.Current); err != nil {
		return err
	}
	if !ValidPassword(in.Next) {
		return errPassword
	}
	if err := s.SetPassword(r.Context(), p.UserID, in.Next); err != nil {
		return err
	}
	if err := s.StartSession(r.Context(), w, r, p.UserID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) SetPassword(ctx context.Context, userID, password string) error {
	hash, err := HashPassword(ctx, password)
	if err != nil {
		return err
	}
	if _, err := s.st.W.ExecContext(ctx, "UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?", hash, store.NowMS(), userID); err != nil {
		return err
	}
	return s.RevokeCredentials(ctx, userID, "")
}

type emailChange struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

func (s *Service) handleEmail(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in emailChange
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if err := s.checkCurrentPassword(r.Context(), p.UserID, in.Password); err != nil {
		return err
	}
	email, err := NormalizeEmail(in.Email)
	if err != nil {
		return err
	}
	var taken int
	if err := s.st.R.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users WHERE email = ? AND id != ?", email, p.UserID).Scan(&taken); err != nil {
		return err
	}
	if taken > 0 {
		return httpx.Fail(http.StatusConflict, "email_taken", "an account with this email already exists")
	}
	if _, err := s.st.W.ExecContext(r.Context(), "UPDATE users SET email = ?, email_verified = 0, updated_at = ? WHERE id = ?", email, store.NowMS(), p.UserID); err != nil {
		return err
	}
	return s.writeMe(w, r, p.UserID)
}
