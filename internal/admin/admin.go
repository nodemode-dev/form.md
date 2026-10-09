package admin

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/i18n"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/mail"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
	"github.com/nodemode-ai/form.md/internal/workspace"
)

const breakGlassTTL = 2 * time.Hour

type Service struct {
	st       *store.Store
	auth     *auth.Service
	ws       *workspace.Service
	settings *settings.Service
	mail     *mail.Mailer
	mailAuth *mail.OAuth
	audit    *audit.Log
	log      *slog.Logger
	deleting []func(ctx context.Context, tx *sql.Tx, userID string) error
}

func New(st *store.Store, a *auth.Service, ws *workspace.Service, set *settings.Service, m *mail.Mailer, al *audit.Log, log *slog.Logger) *Service {
	return &Service{st: st, auth: a, ws: ws, settings: set, mail: m, audit: al, log: log}
}

func (s *Service) UseMailOAuth(o *mail.OAuth) { s.mailAuth = o }

func (s *Service) mailConnect(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	if !p.IsSession() {
		return httpx.Fail(http.StatusForbidden, "session_only", "connect a mail account from a signed-in browser")
	}
	to, err := s.mailAuth.Start(r.URL.Query().Get("mode"), s.auth.BaseURL(r)+"/api/admin/mail/callback")
	if err != nil {
		return httpx.Fail(http.StatusBadRequest, "mail_connect", err.Error())
	}
	http.Redirect(w, r, to, http.StatusFound)
	return nil
}

func (s *Service) mailCallback(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	q := r.URL.Query()
	msg := ""
	if e := q.Get("error"); e != "" {
		msg = "The provider answered: " + e + ". Nothing was changed."
	} else if account, err := s.mailAuth.Finish(r.Context(), q.Get("state"), q.Get("code")); err != nil {
		msg = "The mail account was not connected: " + err.Error()
	} else {
		msg = "Mail is now sent as " + account + ". You can close this tab and send a test mail from Instance settings."
		s.log.Info("mail account connected", "by", p.UserID, "account", account, "mode", s.settings.Get("mail.mode"))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	_, err := fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>form.md</title><body style="font:16px system-ui;background:#111;color:#eee;padding:48px"><p>%s</p></body>`, html.EscapeString(msg))
	return err
}

func (s *Service) OnUserDelete(fn func(ctx context.Context, tx *sql.Tx, userID string) error) {
	s.deleting = append(s.deleting, fn)
}

type userRow struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Color         string `json:"color"`
	Avatar        string `json:"avatar"`
	Role          string `json:"role"`
	Disabled      bool   `json:"disabled"`
	TOTPEnabled   bool   `json:"totp_enabled"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     int64  `json:"created_at"`
	LastSeenAt    int64  `json:"last_seen_at"`
	Workspaces    int    `json:"workspaces"`
}

func (s *Service) listUsers(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	rows, err := s.st.R.QueryContext(r.Context(), `
		SELECT u.id, u.email, u.name, u.color, u.avatar, u.role, u.disabled, u.totp_enabled, u.email_verified, u.created_at,
		       COALESCE((SELECT MAX(last_seen_at) FROM sessions WHERE user_id = u.id), 0),
		       (SELECT COUNT(*) FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id WHERE m.user_id = u.id AND w.personal = 0)
		FROM users u ORDER BY u.name COLLATE NOCASE`)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []userRow{}
	for rows.Next() {
		var u userRow
		var disabled, totp, verified int
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Color, &u.Avatar, &u.Role, &disabled, &totp, &verified, &u.CreatedAt, &u.LastSeenAt, &u.Workspaces); err != nil {
			return err
		}
		u.Disabled, u.TOTPEnabled, u.EmailVerified = disabled == 1, totp == 1, verified == 1
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"users": out})
	return nil
}

func (s *Service) createUser(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Admin    bool   `json:"admin"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	role := "member"
	if in.Admin {
		if !p.IsOwner() {
			return httpx.Fail(http.StatusForbidden, "owner_only", "only the owner makes instance admins")
		}
		role = "admin"
	}
	ctx := r.Context()
	var id string
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		var err error
		if id, err = s.auth.CreateAccount(ctx, tx, in.Email, in.Name, in.Password, role); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "user.create", Detail: id + " as " + role})
	})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
	return nil
}

func (s *Service) setAccess(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	userID, wsID := r.PathValue("id"), r.PathValue("ws")
	if userID == p.UserID {
		return httpx.Fail(http.StatusConflict, "self", "you cannot grant yourself access here")
	}
	var in struct {
		Role string `json:"role"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = ?", userID).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return httpx.ErrNotFound
		}
		before, err := s.ws.SetAccess(ctx, tx, wsID, userID, in.Role)
		if err != nil || before == in.Role {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "member.access", WorkspaceID: wsID,
			Changes: []audit.Change{{Field: "role:" + userID, Before: before, After: in.Role}}})
	})
	if err != nil {
		return err
	}
	s.ws.Changed(wsID, "members", userID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) target(ctx context.Context, p *policy.Principal, id string) (role string, err error) {
	if id == p.UserID {
		return "", httpx.Fail(http.StatusConflict, "self", "use your own account settings for this")
	}
	err = s.st.R.QueryRowContext(ctx, "SELECT role FROM users WHERE id = ?", id).Scan(&role)
	if err != nil {
		return "", store.NoRows(err)
	}
	if role == "owner" {
		return "", httpx.Fail(http.StatusForbidden, "owner_protected", "the owner account can only be changed by its owner")
	}
	return role, nil
}

func (s *Service) updateUser(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	id := r.PathValue("id")
	cur, err := s.target(r.Context(), p, id)
	if err != nil {
		return err
	}
	var in struct {
		Role     *string `json:"role"`
		Disabled *bool   `json:"disabled"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if cur == "admin" && !p.IsOwner() {
		return httpx.Fail(http.StatusForbidden, "owner_only", "only the owner can change another admin")
	}
	if in.Role != nil {
		if !p.IsOwner() {
			return httpx.Fail(http.StatusForbidden, "owner_only", "only the owner changes instance roles")
		}
		if *in.Role != "admin" && *in.Role != "member" {
			return httpx.Fail(http.StatusBadRequest, "bad_role", "role must be admin or member")
		}
		if _, err := s.st.W.ExecContext(r.Context(), "UPDATE users SET role = ?, updated_at = ? WHERE id = ?", *in.Role, store.NowMS(), id); err != nil {
			return err
		}
		if err := s.audit.Record(r.Context(), nil, p, audit.Entry{Action: "user.role", Changes: []audit.Change{{Field: "role:" + id, Before: cur, After: *in.Role}}}); err != nil {
			return err
		}
	}
	if in.Disabled != nil {
		if _, err := s.st.W.ExecContext(r.Context(), "UPDATE users SET disabled = ?, updated_at = ? WHERE id = ?", store.Bool(*in.Disabled), store.NowMS(), id); err != nil {
			return err
		}
		if *in.Disabled {
			if err := s.auth.RevokeCredentials(r.Context(), id, ""); err != nil {
				return err
			}
		}
		action := "user.reactivate"
		if *in.Disabled {
			action = "user.deactivate"
		}
		if err := s.audit.Record(r.Context(), nil, p, audit.Entry{Action: action, Detail: id}); err != nil {
			return err
		}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) resetTOTP(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	id := r.PathValue("id")
	if _, err := s.target(r.Context(), p, id); err != nil {
		return err
	}
	if err := s.auth.DisableTOTP(r, id); err != nil {
		return err
	}
	if err := s.audit.Record(r.Context(), nil, p, audit.Entry{Action: "user.reset_2fa", Detail: id}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) setPassword(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	id := r.PathValue("id")
	if _, err := s.target(r.Context(), p, id); err != nil {
		return err
	}
	var in struct {
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if !auth.ValidPassword(in.Password) {
		return httpx.Fail(http.StatusBadRequest, "weak_password", "use at least 8 characters")
	}
	if err := s.auth.SetPassword(r.Context(), id, in.Password); err != nil {
		return err
	}
	if err := s.audit.Record(r.Context(), nil, p, audit.Entry{Action: "user.set_password", Detail: id}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type planWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type promotion struct {
	planWorkspace
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

type Plan struct {
	UserID      string          `json:"user_id"`
	Name        string          `json:"name"`
	Email       string          `json:"email"`
	Purge       []planWorkspace `json:"purge"`
	Promote     []promotion     `json:"promote"`
	HandToOwner []planWorkspace `json:"hand_to_owner"`
}

func (s *Service) plan(ctx context.Context, q store.Querier, userID string) (*Plan, error) {
	pl := &Plan{UserID: userID, Purge: []planWorkspace{}, Promote: []promotion{}, HandToOwner: []planWorkspace{}}
	if err := q.QueryRowContext(ctx, "SELECT name, email FROM users WHERE id = ?", userID).Scan(&pl.Name, &pl.Email); err != nil {
		return nil, store.NoRows(err)
	}
	rows, err := q.QueryContext(ctx, `SELECT w.id, w.name, w.personal, m.role FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	type mem struct {
		ws       planWorkspace
		personal bool
		role     string
	}
	var mine []mem
	for rows.Next() {
		var m mem
		var personal int
		if err := rows.Scan(&m.ws.ID, &m.ws.Name, &personal, &m.role); err != nil {
			rows.Close()
			return nil, err
		}
		m.personal = personal == 1
		mine = append(mine, m)
	}
	rows.Close()
	for _, m := range mine {
		if m.personal {
			pl.Purge = append(pl.Purge, m.ws)
			continue
		}
		if m.role != "admin" {
			continue
		}
		var otherAdmins int
		if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = ? AND role = 'admin' AND user_id != ?", m.ws.ID, userID).Scan(&otherAdmins); err != nil {
			return nil, err
		}
		if otherAdmins > 0 {
			continue
		}
		var heir, heirName string
		err := q.QueryRowContext(ctx, `SELECT u.id, u.name FROM workspace_members m JOIN users u ON u.id = m.user_id
			WHERE m.workspace_id = ? AND m.user_id != ? AND u.disabled = 0
			ORDER BY CASE m.role WHEN 'member' THEN 0 ELSE 1 END, m.joined_at LIMIT 1`, m.ws.ID, userID).Scan(&heir, &heirName)
		if err == sql.ErrNoRows {
			pl.HandToOwner = append(pl.HandToOwner, m.ws)
			continue
		}
		if err != nil {
			return nil, err
		}
		pl.Promote = append(pl.Promote, promotion{planWorkspace: m.ws, UserID: heir, UserName: heirName})
	}
	return pl, nil
}

func (s *Service) deletionPlan(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	if _, err := s.target(r.Context(), p, r.PathValue("id")); err != nil {
		return err
	}
	pl, err := s.plan(r.Context(), s.st.R, r.PathValue("id"))
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, pl)
	return nil
}

func (s *Service) deleteUser(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	id := r.PathValue("id")
	if _, err := s.target(r.Context(), p, id); err != nil {
		return err
	}
	ctx := r.Context()
	if err := s.auth.RevokeCredentials(ctx, id, ""); err != nil {
		return err
	}
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		pl, err := s.plan(ctx, tx, id)
		if err != nil {
			return err
		}
		for _, fn := range s.deleting {
			if err := fn(ctx, tx, id); err != nil {
				return err
			}
		}
		for _, ws := range pl.Purge {
			if err := s.ws.Purge(ctx, tx, ws.ID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id); err != nil {
			return err
		}
		for _, pr := range pl.Promote {
			if _, err := tx.ExecContext(ctx, "UPDATE workspace_members SET role = 'admin' WHERE workspace_id = ? AND user_id = ?", pr.ID, pr.UserID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE workspaces SET owner_id = ? WHERE id = ?", pr.UserID, pr.ID); err != nil {
				return err
			}
		}
		for _, ws := range pl.HandToOwner {
			if _, err := tx.ExecContext(ctx, "INSERT OR REPLACE INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, 'admin', ?)", ws.ID, p.UserID, store.NowMS()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE workspaces SET owner_id = ? WHERE id = ?", p.UserID, ws.ID); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "user.delete", Detail: fmt.Sprintf("%s <%s>", pl.Name, pl.Email)})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) transferOwnership(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	if _, err := s.auth.VerifyLogin(ctx, r, p.Email, in.Password, "", ""); err != nil {
		if api, ok := err.(*httpx.APIError); ok && api.Code == "totp_required" {
			return httpx.Fail(http.StatusForbidden, "bad_password", "confirm with your password")
		}
		return httpx.Fail(http.StatusForbidden, "bad_password", "the password is wrong")
	}
	if _, err := s.target(ctx, p, in.UserID); err != nil {
		return err
	}
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		var disabled int
		if err := tx.QueryRowContext(ctx, "SELECT disabled FROM users WHERE id = ?", in.UserID).Scan(&disabled); err != nil {
			return store.NoRows(err)
		}
		if disabled == 1 {
			return httpx.Fail(http.StatusConflict, "disabled", "a deactivated account cannot become owner")
		}
		now := store.NowMS()
		if _, err := tx.ExecContext(ctx, "UPDATE users SET role = 'admin', updated_at = ? WHERE id = ?", now, p.UserID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE users SET role = 'owner', updated_at = ? WHERE id = ?", now, in.UserID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE break_glass SET revoked_at = ? WHERE user_id = ? AND revoked_at = 0", now, p.UserID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "instance.transfer_ownership", Detail: in.UserID})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) createBreakGlass(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		Reason      string `json:"reason"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	reason := strings.TrimSpace(in.Reason)
	if utf8.RuneCountInString(reason) < 10 || utf8.RuneCountInString(reason) > 500 {
		return httpx.Fail(http.StatusBadRequest, "reason_required", "give a reason of at least 10 characters")
	}
	ctx := r.Context()
	var name string
	var personal int
	if err := s.st.R.QueryRowContext(ctx, "SELECT name, personal FROM workspaces WHERE id = ?", in.WorkspaceID).Scan(&name, &personal); err != nil {
		return store.NoRows(err)
	}
	if personal == 1 {
		return httpx.Fail(http.StatusForbidden, "personal_space", "emergency access never applies to personal spaces")
	}
	now := time.Now()
	id := ids.New()
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO break_glass (id, user_id, workspace_id, reason, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`,
			id, p.UserID, in.WorkspaceID, reason, now.UnixMilli(), now.Add(breakGlassTTL).UnixMilli()); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "instance.break_glass", WorkspaceID: in.WorkspaceID, Detail: reason})
	})
	if err != nil {
		return err
	}
	s.ws.Changed("", "break_glass", p.UserID)
	if s.mail.Enabled() {
		rows, err := s.st.R.QueryContext(ctx, `SELECT u.email, u.lang FROM workspace_members m JOIN users u ON u.id = m.user_id
			WHERE m.workspace_id = ? AND m.role = 'admin' AND u.disabled = 0`, in.WorkspaceID)
		if err == nil {
			var to, langs []string
			for rows.Next() {
				var e, l string
				if rows.Scan(&e, &l) == nil {
					to, langs = append(to, e), append(langs, l)
				}
			}
			rows.Close()
			for i, e := range to {
				body := i18n.T(langs[i], "%s used emergency read access to the %q workspace on %s for two hours.\n\nReason given:\n%s\n", p.Name, name, s.settings.Get("instance.name"), reason)
				if err := s.mail.Send(ctx, e, i18n.T(langs[i], "Emergency access to %s", name), body); err != nil {
					s.log.Warn("break-glass notice failed", "err", err)
				}
			}
		}
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "expires_at": now.Add(breakGlassTTL).UnixMilli()})
	return nil
}

func (s *Service) listBreakGlass(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	rows, err := s.st.R.QueryContext(r.Context(), `SELECT b.id, b.workspace_id, w.name, b.reason, b.created_at, b.expires_at
		FROM break_glass b JOIN workspaces w ON w.id = b.workspace_id
		WHERE b.user_id = ? AND b.revoked_at = 0 AND b.expires_at > ? ORDER BY b.created_at DESC`, p.UserID, store.NowMS())
	if err != nil {
		return err
	}
	defer rows.Close()
	type grant struct {
		ID            string `json:"id"`
		WorkspaceID   string `json:"workspace_id"`
		WorkspaceName string `json:"workspace_name"`
		Reason        string `json:"reason"`
		CreatedAt     int64  `json:"created_at"`
		ExpiresAt     int64  `json:"expires_at"`
	}
	out := []grant{}
	for rows.Next() {
		var g grant
		if err := rows.Scan(&g.ID, &g.WorkspaceID, &g.WorkspaceName, &g.Reason, &g.CreatedAt, &g.ExpiresAt); err != nil {
			return err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"grants": out})
	return nil
}

func (s *Service) revokeBreakGlass(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	res, err := s.st.W.ExecContext(r.Context(), "UPDATE break_glass SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at = 0", store.NowMS(), r.PathValue("id"), p.UserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpx.ErrNotFound
	}
	s.ws.Changed("", "break_glass", p.UserID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) stranded(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	rows, err := s.st.R.QueryContext(r.Context(), `SELECT w.id, w.name, w.created_at,
		(SELECT COUNT(*) FROM workspace_members m WHERE m.workspace_id = w.id)
		FROM workspaces w WHERE w.personal = 0
		AND NOT EXISTS (SELECT 1 FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = w.id AND m.role = 'admin' AND u.disabled = 0)
		ORDER BY w.name COLLATE NOCASE`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type st struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		CreatedAt int64  `json:"created_at"`
		Members   int    `json:"members"`
	}
	out := []st{}
	for rows.Next() {
		var x st
		if err := rows.Scan(&x.ID, &x.Name, &x.CreatedAt, &x.Members); err != nil {
			return err
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"workspaces": out})
	return nil
}

func (s *Service) adopt(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	id := r.PathValue("id")
	ctx := r.Context()
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		var personal int
		if err := tx.QueryRowContext(ctx, "SELECT personal FROM workspaces WHERE id = ?", id).Scan(&personal); err != nil {
			return store.NoRows(err)
		}
		if personal == 1 {
			return httpx.Fail(http.StatusForbidden, "personal_space", "personal spaces cannot be adopted")
		}
		if _, err := tx.ExecContext(ctx, "INSERT OR REPLACE INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, 'admin', ?)", id, p.UserID, store.NowMS()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE workspaces SET owner_id = ? WHERE id = ?", p.UserID, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.adopt", WorkspaceID: id})
	})
	if err != nil {
		return err
	}
	s.ws.Changed(id, "members", p.UserID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) deleteStranded(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	id := r.PathValue("id")
	ctx := r.Context()
	members := s.ws.MemberIDs(ctx, id)
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		var personal, admins int
		if err := tx.QueryRowContext(ctx, `SELECT w.personal, (SELECT COUNT(*) FROM workspace_members m JOIN users u ON u.id = m.user_id
			WHERE m.workspace_id = w.id AND m.role = 'admin' AND u.disabled = 0) FROM workspaces w WHERE w.id = ?`, id).Scan(&personal, &admins); err != nil {
			return store.NoRows(err)
		}
		if personal == 1 || admins > 0 {
			return httpx.Fail(http.StatusConflict, "not_stranded", "this workspace still has an admin")
		}
		if err := s.ws.Purge(ctx, tx, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.delete_stranded", WorkspaceID: id})
	})
	if err != nil {
		return err
	}
	s.ws.Changed("", "deleted", members...)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Service) access(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	rows, err := s.st.R.QueryContext(r.Context(), `SELECT w.id, w.name, w.icon, w.auto_join, w.agent_access, COALESCE(m.user_id, ''), COALESCE(m.role, '')
		FROM workspaces w LEFT JOIN workspace_members m ON m.workspace_id = w.id
		WHERE w.personal = 0 ORDER BY w.name COLLATE NOCASE`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type member struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	type ws struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Icon        string   `json:"icon"`
		AutoJoin    bool     `json:"auto_join"`
		AgentAccess string   `json:"agent_access"`
		Members     []member `json:"members"`
	}
	out := []*ws{}
	index := map[string]*ws{}
	for rows.Next() {
		var id, name, icon, agent, uid, role string
		var auto int
		if err := rows.Scan(&id, &name, &icon, &auto, &agent, &uid, &role); err != nil {
			return err
		}
		x := index[id]
		if x == nil {
			x = &ws{ID: id, Name: name, Icon: icon, AutoJoin: auto == 1, AgentAccess: agent, Members: []member{}}
			index[id] = x
			out = append(out, x)
		}
		if uid != "" {
			x.Members = append(x.Members, member{uid, role})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"workspaces": out})
	return nil
}

func (s *Service) getSettings(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	httpx.JSON(w, http.StatusOK, map[string]any{"settings": s.settings.Snapshot(), "secret_set": settings.SecretSet, "mail_enabled": s.mail.Enabled()})
	return nil
}

func (s *Service) putSettings(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in map[string]string
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if err := s.settings.Set(r.Context(), in); err != nil {
		if fe, ok := err.(*settings.FieldError); ok {
			return httpx.Fail(http.StatusBadRequest, "bad_setting", fe.Problem).With("key", fe.Key)
		}
		return err
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	if err := s.audit.Record(r.Context(), nil, p, audit.Entry{Action: "instance.settings", Detail: strings.Join(keys, ", ")}); err != nil {
		return err
	}
	return s.getSettings(w, r, p)
}

func (s *Service) testMail(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in struct {
		To string `json:"to"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	to := in.To
	if to == "" {
		to = p.Email
	}
	lang := i18n.UserLang(r.Context(), s.st.R, p.UserID)
	body := i18n.T(lang, "This is a test message from %s. Mail sending works.\n", s.settings.Get("instance.name"))
	if err := s.mail.Send(r.Context(), to, i18n.T(lang, "Test message"), body); err != nil {
		return httpx.Fail(http.StatusBadGateway, "mail_failed", err.Error())
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type module struct{ s *Service }

func (s *Service) Module() app.Module { return module{s} }

func (module) ID() string { return "admin" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.s.auth
	r.HandleFunc("GET /api/admin/users", a.RequireAdmin(s.listUsers))
	r.HandleFunc("POST /api/admin/users", a.RequireAdmin(s.createUser))
	r.HandleFunc("PATCH /api/admin/users/{id}", a.RequireAdmin(s.updateUser))
	r.HandleFunc("PUT /api/admin/users/{id}/workspaces/{ws}", a.RequireAdmin(s.setAccess))
	r.HandleFunc("POST /api/admin/users/{id}/reset-2fa", a.RequireOwner(s.resetTOTP))
	r.HandleFunc("POST /api/admin/users/{id}/password", a.RequireOwner(s.setPassword))
	r.HandleFunc("GET /api/admin/users/{id}/deletion-plan", a.RequireOwner(s.deletionPlan))
	r.HandleFunc("DELETE /api/admin/users/{id}", a.RequireOwner(s.deleteUser))
	r.HandleFunc("POST /api/admin/ownership", a.RequireOwner(s.transferOwnership))
	r.HandleFunc("POST /api/admin/break-glass", a.RequireOwner(s.createBreakGlass))
	r.HandleFunc("GET /api/admin/break-glass", a.RequireOwner(s.listBreakGlass))
	r.HandleFunc("DELETE /api/admin/break-glass/{id}", a.RequireOwner(s.revokeBreakGlass))
	r.HandleFunc("GET /api/admin/stranded", a.RequireOwner(s.stranded))
	r.HandleFunc("POST /api/admin/stranded/{id}/adopt", a.RequireOwner(s.adopt))
	r.HandleFunc("DELETE /api/admin/stranded/{id}", a.RequireOwner(s.deleteStranded))
	r.HandleFunc("GET /api/admin/access", a.RequireAdmin(s.access))
	r.HandleFunc("GET /api/admin/settings", a.RequireAdmin(s.getSettings))
	r.HandleFunc("PUT /api/admin/settings", a.RequireAdmin(s.putSettings))
	r.HandleFunc("POST /api/admin/mail/test", a.RequireAdmin(s.testMail))
	if s.mailAuth != nil {
		r.HandleFunc("GET /api/admin/mail/connect", a.RequireAdmin(s.mailConnect))
		r.HandleFunc("GET /api/admin/mail/callback", a.RequireAdmin(s.mailCallback))
	}
	return nil
}
