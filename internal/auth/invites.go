package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

const inviteTTL = 14 * 24 * time.Hour

type Mailer interface {
	Enabled() bool
	Send(ctx context.Context, to, subject, body string) error
}

func (s *Service) SetMailer(m Mailer) { s.mailer = m }

type Invite struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	WorkspaceID string `json:"workspace_id"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   int64  `json:"created_at"`
	ExpiresAt   int64  `json:"expires_at"`
}

var validRoles = map[string]bool{"admin": true, "member": true, "viewer": true}

func (s *Service) workspaceAdmin(ctx context.Context, p *policy.Principal, wsID string) (string, error) {
	a, err := s.gate.Workspace(ctx, p, wsID)
	if err != nil {
		return "", err
	}
	if !a.IsAdmin() {
		return "", httpx.ErrForbidden
	}
	if a.Personal {
		return "", httpx.Fail(http.StatusConflict, "personal_space", "personal spaces cannot have other members")
	}
	var name string
	err = s.st.R.QueryRowContext(ctx, "SELECT name FROM workspaces WHERE id = ?", wsID).Scan(&name)
	return name, store.NoRows(err)
}

func (s *Service) handleCreateInvite(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	wsID := r.PathValue("id")
	wsName, err := s.workspaceAdmin(r.Context(), p, wsID)
	if err != nil {
		return err
	}
	var in struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if !validRoles[in.Role] {
		return httpx.Fail(http.StatusBadRequest, "bad_role", "role must be admin, member or viewer")
	}
	email := ""
	if strings.TrimSpace(in.Email) != "" {
		if email, err = NormalizeEmail(in.Email); err != nil {
			return err
		}
	}
	token := secrets.Token("", 24)
	now := time.Now()
	inv := Invite{ID: ids.New(), Email: email, Role: in.Role, WorkspaceID: wsID, CreatedBy: p.UserID, CreatedAt: now.UnixMilli(), ExpiresAt: now.Add(inviteTTL).UnixMilli()}
	if _, err := s.st.W.ExecContext(r.Context(), `INSERT INTO invites (id, token_hash, email, role, workspace_id, created_by, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, inv.ID, secrets.Hash(token), inv.Email, inv.Role, inv.WorkspaceID, inv.CreatedBy, inv.CreatedAt, inv.ExpiresAt); err != nil {
		return err
	}
	link := s.baseURL(r) + "/invite/" + token
	emailed := false
	if email != "" && s.mailer != nil && s.mailer.Enabled() {
		body := fmt.Sprintf("%s invited you to the %q workspace on %s.\n\nAccept the invitation:\n%s\n\nThe link works for 14 days.\n",
			p.Name, wsName, s.settings.Get("instance.name"), link)
		if err := s.mailer.Send(r.Context(), email, "You are invited to "+wsName, body); err != nil {
			s.log.Warn("invite email failed", "err", err)
		} else {
			emailed = true
		}
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"invite": inv, "url": link, "emailed": emailed})
	return nil
}

func (s *Service) handleListInvites(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	wsID := r.PathValue("id")
	if _, err := s.workspaceAdmin(r.Context(), p, wsID); err != nil {
		return err
	}
	rows, err := s.st.R.QueryContext(r.Context(), `SELECT id, email, role, workspace_id, COALESCE(created_by, ''), created_at, expires_at
		FROM invites WHERE workspace_id = ? AND accepted_at = 0 AND expires_at > ? ORDER BY created_at DESC`, wsID, store.NowMS())
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []Invite{}
	for rows.Next() {
		var i Invite
		if err := rows.Scan(&i.ID, &i.Email, &i.Role, &i.WorkspaceID, &i.CreatedBy, &i.CreatedAt, &i.ExpiresAt); err != nil {
			return err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"invites": out})
	return nil
}

func (s *Service) handleDeleteInvite(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	wsID := r.PathValue("id")
	if _, err := s.workspaceAdmin(r.Context(), p, wsID); err != nil {
		return err
	}
	res, err := s.st.W.ExecContext(r.Context(), "DELETE FROM invites WHERE id = ? AND workspace_id = ?", r.PathValue("invite"), wsID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type pendingInvite struct {
	ID, Email, Role, WorkspaceID, WorkspaceName, WorkspaceIcon, Inviter string
	ExpiresAt, AcceptedAt                                               int64
}

func (s *Service) lookupInvite(ctx context.Context, token string) (*pendingInvite, error) {
	var i pendingInvite
	err := s.st.R.QueryRowContext(ctx, `
		SELECT i.id, i.email, i.role, i.workspace_id, w.name, w.icon, COALESCE(u.name, ''), i.expires_at, i.accepted_at
		FROM invites i JOIN workspaces w ON w.id = i.workspace_id LEFT JOIN users u ON u.id = i.created_by
		WHERE i.token_hash = ?`, secrets.Hash(token)).
		Scan(&i.ID, &i.Email, &i.Role, &i.WorkspaceID, &i.WorkspaceName, &i.WorkspaceIcon, &i.Inviter, &i.ExpiresAt, &i.AcceptedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpx.Fail(http.StatusNotFound, "invite_invalid", "this invitation does not exist or was already used")
	}
	if err != nil {
		return nil, err
	}
	if i.AcceptedAt != 0 {
		return nil, httpx.Fail(http.StatusGone, "invite_used", "this invitation was already used")
	}
	if i.ExpiresAt < store.NowMS() {
		return nil, httpx.Fail(http.StatusGone, "invite_expired", "this invitation has expired")
	}
	return &i, nil
}

func (s *Service) handleGetInvite(w http.ResponseWriter, r *http.Request) error {
	i, err := s.lookupInvite(r.Context(), r.PathValue("token"))
	if err != nil {
		return err
	}
	out := map[string]any{
		"workspace":      map[string]string{"name": i.WorkspaceName, "icon": i.WorkspaceIcon},
		"inviter":        i.Inviter,
		"role":           i.Role,
		"email":          i.Email,
		"account_exists": false,
	}
	if i.Email != "" {
		var n int
		if err := s.st.R.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users WHERE email = ?", i.Email).Scan(&n); err != nil {
			return err
		}
		out["account_exists"] = n > 0
	}
	if p := policy.From(r.Context()); p.IsSession() {
		out["signed_in_as"] = p.Email
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

type acceptInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
	Recovery string `json:"recovery"`
}

func (s *Service) handleAcceptInvite(w http.ResponseWriter, r *http.Request) error {
	if !s.loginLimit.Allow(s.ClientIP(r)) {
		return httpx.Fail(http.StatusTooManyRequests, "rate_limited", "too many attempts, wait a minute")
	}
	ctx := r.Context()
	inv, err := s.lookupInvite(ctx, r.PathValue("token"))
	if err != nil {
		return err
	}
	var in acceptInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	userID := ""
	newSession := true
	verified := false
	if p := policy.From(ctx); p.IsSession() {
		if inv.Email != "" && !strings.EqualFold(inv.Email, p.Email) {
			return httpx.Fail(http.StatusForbidden, "invite_other_email", "this invitation is for a different email address")
		}
		userID, newSession = p.UserID, false
	} else {
		email := inv.Email
		if email == "" {
			if email, err = NormalizeEmail(in.Email); err != nil {
				return err
			}
		}
		var exists int
		if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			if userID, err = s.VerifyLogin(ctx, r, email, in.Password, in.Code, in.Recovery); err != nil {
				return err
			}
		} else {
			name, err := cleanName(in.Name)
			if err != nil {
				return err
			}
			if !ValidPassword(in.Password) {
				return errPassword
			}
			verified = inv.Email != ""
			err = s.st.Tx(ctx, func(tx *sql.Tx) error {
				userID, err = s.createUser(ctx, tx, email, name, in.Password, "member", verified)
				return err
			})
			if err != nil {
				return err
			}
		}
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE invites SET accepted_at = ?, accepted_by = ? WHERE id = ? AND accepted_at = 0", store.NowMS(), userID, inv.ID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return httpx.Fail(http.StatusGone, "invite_used", "this invitation was already used")
		}
		if inv.Email != "" {
			if _, err := tx.ExecContext(ctx, "UPDATE users SET email_verified = 1 WHERE id = ? AND email = ?", userID, inv.Email); err != nil {
				return err
			}
		}
		if s.accounts == nil {
			return nil
		}
		return s.accounts.JoinInvite(ctx, tx, userID, inv.WorkspaceID, inv.Role)
	})
	if err != nil {
		return err
	}
	if newSession {
		if err := s.StartSession(ctx, w, r, userID); err != nil {
			return err
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"workspace_id": inv.WorkspaceID})
	return nil
}
