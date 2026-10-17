package auth

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

func (s *Service) handleTOTPSetup(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var enabled int
	if err := s.st.R.QueryRowContext(r.Context(), "SELECT totp_enabled FROM users WHERE id = ?", p.UserID).Scan(&enabled); err != nil {
		return err
	}
	if enabled == 1 {
		return httpx.Fail(http.StatusConflict, "totp_active", "two-factor sign-in is already on; turn it off first")
	}
	secret := NewTOTPSecret()
	if _, err := s.st.W.ExecContext(r.Context(), "UPDATE users SET totp_pending = ? WHERE id = ?", s.box.Seal(secret), p.UserID); err != nil {
		return err
	}
	uri := TOTPURL(s.settings.Get("instance.name"), p.Email, secret)
	svg, err := QRSVG(uri)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"secret": secret, "uri": uri, "qr_svg": svg})
	return nil
}

type codeInput struct {
	Code string `json:"code"`
}

func (s *Service) handleTOTPConfirm(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in codeInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	var pending string
	if err := s.st.R.QueryRowContext(r.Context(), "SELECT totp_pending FROM users WHERE id = ?", p.UserID).Scan(&pending); err != nil {
		return err
	}
	secret, err := s.box.Unseal(pending)
	if err != nil || secret == "" {
		return httpx.Fail(http.StatusConflict, "totp_not_started", "start two-factor setup first")
	}
	step, ok := CheckTOTP(secret, in.Code, 0, time.Now())
	if !ok {
		return httpx.Fail(http.StatusBadRequest, "bad_totp", "that code did not work")
	}
	codes := NewRecoveryCodes(10)
	err = s.st.Tx(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(r.Context(), `UPDATE users SET totp_secret = totp_pending, totp_pending = '', totp_enabled = 1,
			totp_last_step = ?, updated_at = ? WHERE id = ?`, step, store.NowMS(), p.UserID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(r.Context(), "DELETE FROM recovery_codes WHERE user_id = ?", p.UserID); err != nil {
			return err
		}
		for _, c := range codes {
			if _, err := tx.ExecContext(r.Context(), "INSERT INTO recovery_codes (user_id, code_hash) VALUES (?, ?)", p.UserID, secrets.Hash(c)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
	return nil
}

func (s *Service) handleTOTPDisable(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in struct {
		Code     string `json:"code"`
		Recovery string `json:"recovery"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	ok, err := s.checkSecondFactor(r.Context(), p.UserID, in.Code, in.Recovery)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.Fail(http.StatusBadRequest, "bad_totp", "that code did not work")
	}
	if err := s.DisableTOTP(r, p.UserID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) DisableTOTP(r *http.Request, userID string) error {
	return s.st.Tx(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(r.Context(), `UPDATE users SET totp_secret = '', totp_pending = '', totp_enabled = 0,
			totp_last_step = 0, updated_at = ? WHERE id = ?`, store.NowMS(), userID); err != nil {
			return err
		}
		_, err := tx.ExecContext(r.Context(), "DELETE FROM recovery_codes WHERE user_id = ?", userID)
		return err
	})
}

type Token struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Prefix        string   `json:"prefix"`
	Scope         string   `json:"scope"`
	AllWorkspaces bool     `json:"all_workspaces"`
	Workspaces    []string `json:"workspaces"`
	CreatedAt     int64    `json:"created_at"`
	LastUsedAt    int64    `json:"last_used_at"`
	LastUsedIP    string   `json:"last_used_ip"`
}

func (s *Service) handleListTokens(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	rows, err := s.st.R.QueryContext(r.Context(), `
		SELECT t.id, t.name, t.prefix, t.scope, t.all_workspaces, t.created_at, t.last_used_at, t.last_used_ip,
		       COALESCE((SELECT group_concat(workspace_id) FROM api_token_workspaces WHERE token_id = t.id), '')
		FROM api_tokens t WHERE t.user_id = ? ORDER BY t.created_at DESC`, p.UserID)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []Token{}
	for rows.Next() {
		var t Token
		var all int
		var ws string
		if err := rows.Scan(&t.ID, &t.Name, &t.Prefix, &t.Scope, &all, &t.CreatedAt, &t.LastUsedAt, &t.LastUsedIP, &ws); err != nil {
			return err
		}
		t.AllWorkspaces = all == 1
		t.Workspaces = []string{}
		if ws != "" {
			t.Workspaces = strings.Split(ws, ",")
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tokens": out})
	return nil
}

type tokenInput struct {
	Name       string    `json:"name"`
	Scope      string    `json:"scope"`
	Workspaces *[]string `json:"workspaces"`
}

func (s *Service) handleCreateToken(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in tokenInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return httpx.Fail(http.StatusBadRequest, "bad_name", "give the token a name of up to 80 characters")
	}
	if in.Scope != "read" && in.Scope != "write" {
		return httpx.Fail(http.StatusBadRequest, "bad_scope", "scope must be read or write")
	}
	var chosen []string
	if in.Workspaces != nil {
		mine, _, err := s.gate.Memberships(r.Context(), p)
		if err != nil {
			return err
		}
		member := map[string]bool{}
		for _, m := range mine {
			member[m.WorkspaceID] = true
		}
		for _, ws := range *in.Workspaces {
			if member[ws] {
				chosen = append(chosen, ws)
			}
		}
		if len(chosen) == 0 {
			return httpx.Fail(http.StatusBadRequest, "no_workspaces", "choose at least one workspace you belong to, or all workspaces")
		}
	}
	plain := secrets.Token(TokenPrefix, 32)
	id := ids.New()
	err := s.st.Tx(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO api_tokens (id, user_id, name, token_hash, prefix, scope, all_workspaces, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, p.UserID, name, secrets.Hash(plain), plain[:len(TokenPrefix)+6], in.Scope, store.Bool(in.Workspaces == nil), store.NowMS()); err != nil {
			return err
		}
		for _, ws := range chosen {
			if _, err := tx.ExecContext(r.Context(), "INSERT INTO api_token_workspaces (token_id, workspace_id) VALUES (?, ?)", id, ws); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "token": plain})
	return nil
}

func (s *Service) handleDeleteToken(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	res, err := s.st.W.ExecContext(r.Context(), "DELETE FROM api_tokens WHERE id = ? AND user_id = ?", r.PathValue("id"), p.UserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpx.ErrNotFound
	}
	s.CredentialRevoked(r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type sessionInfo struct {
	ID         string `json:"id"`
	Current    bool   `json:"current"`
	CreatedAt  int64  `json:"created_at"`
	LastSeenAt int64  `json:"last_seen_at"`
	UserAgent  string `json:"user_agent"`
	IP         string `json:"ip"`
}

func (s *Service) handleListSessions(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	rows, err := s.st.R.QueryContext(r.Context(), `SELECT id, created_at, last_seen_at, user_agent, ip FROM sessions
		WHERE user_id = ? AND expires_at > ? ORDER BY last_seen_at DESC`, p.UserID, store.NowMS())
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []sessionInfo{}
	for rows.Next() {
		var si sessionInfo
		if err := rows.Scan(&si.ID, &si.CreatedAt, &si.LastSeenAt, &si.UserAgent, &si.IP); err != nil {
			return err
		}
		si.Current = si.ID == p.CredentialID
		out = append(out, si)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sessions": out})
	return nil
}

func (s *Service) handleDeleteSession(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	id := r.PathValue("id")
	q, args := "DELETE FROM sessions WHERE user_id = ? AND id = ? RETURNING id", []any{p.UserID, id}
	if id == "others" {
		q, args = "DELETE FROM sessions WHERE user_id = ? AND id != ? RETURNING id", []any{p.UserID, p.CredentialID}
	}
	rows, err := s.st.W.QueryContext(r.Context(), q, args...)
	if err != nil {
		return err
	}
	var gone []string
	for rows.Next() {
		var sid string
		if rows.Scan(&sid) == nil {
			gone = append(gone, sid)
		}
	}
	err = rows.Err()
	rows.Close()
	s.CredentialRevoked(gone...)
	if err != nil {
		return err
	}
	if id != "others" && len(gone) == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
