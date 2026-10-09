package page

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Share struct {
	ID        string `json:"id"`
	PageID    string `json:"page_id"`
	Kind      string `json:"kind"`
	SubjectID string `json:"subject_id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	Level     string `json:"level"`
	ExpiresAt int64  `json:"expires_at"`
	Guest     bool   `json:"guest,omitempty"`
	Inherited bool   `json:"inherited,omitempty"`
	FromTitle string `json:"from_title,omitempty"`
}

type ShareLink struct {
	ID        string `json:"id"`
	PageID    string `json:"page_id"`
	Level     string `json:"level"`
	Password  bool   `json:"password"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
	Token     string `json:"token,omitempty"`
}

type GuestInvite struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Level     string `json:"level"`
	ExpiresAt int64  `json:"expires_at"`
}

type Sharing struct {
	General  string        `json:"general"`
	Team     *Teamspace    `json:"teamspace,omitempty"`
	Shares   []Share       `json:"shares"`
	Links    []ShareLink   `json:"links"`
	Invites  []GuestInvite `json:"invites"`
	CanShare bool          `json:"can_share"`
}

func (s *Service) Sharing(ctx context.Context, p *policy.Principal, id string) (*Sharing, error) {
	m, a, err := s.readable(ctx, p, id)
	if err != nil {
		return nil, err
	}
	out := &Sharing{General: "workspace", Shares: []Share{}, Links: []ShareLink{}, Invites: []GuestInvite{}, CanShare: a.level >= LevelFull && p.CanWrite()}
	if len(a.chain) > 0 && a.chain[0].teamspace != "" {
		if t, err := s.teamspace(ctx, s.st.R, a.chain[0].teamspace); err == nil {
			out.Team, out.General = t, "teamspace"
		}
	}
	for _, n := range a.chain {
		if n.visibility == "private" {
			out.General = "private"
		}
	}
	titles := map[string]string{}
	for _, n := range a.chain {
		var t string
		if s.st.R.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = ?", n.id).Scan(&t) == nil {
			titles[n.id] = t
		}
	}
	for _, n := range a.chain {
		rows, err := s.st.R.QueryContext(ctx, `SELECT ps.id, ps.subject_kind, ps.subject_id, ps.level, ps.expires_at,
				COALESCE(u.name, g.name, ''), COALESCE(u.email, ''),
				CASE WHEN ps.subject_kind = 'user' AND NOT EXISTS (SELECT 1 FROM workspace_members wm WHERE wm.workspace_id = ps.workspace_id AND wm.user_id = ps.subject_id) THEN 1 ELSE 0 END
			FROM page_shares ps LEFT JOIN users u ON ps.subject_kind = 'user' AND u.id = ps.subject_id
			LEFT JOIN groups g ON ps.subject_kind = 'group' AND g.id = ps.subject_id
			WHERE ps.page_id = ? AND (ps.expires_at = 0 OR ps.expires_at > ?) ORDER BY ps.created_at`, n.id, store.NowMS())
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			sh := Share{PageID: n.id, Inherited: n.id != m.ID}
			var guest int
			if err := rows.Scan(&sh.ID, &sh.Kind, &sh.SubjectID, &sh.Level, &sh.ExpiresAt, &sh.Name, &sh.Email, &guest); err != nil {
				rows.Close()
				return nil, err
			}
			sh.Guest = guest == 1
			if sh.Inherited {
				sh.FromTitle = titles[n.id]
			}
			out.Shares = append(out.Shares, sh)
		}
		rows.Close()
	}
	if !out.CanShare {
		return out, nil
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT id, level, password_hash != '', expires_at, created_at FROM share_links
		WHERE page_id = ? AND revoked_at = 0 AND (expires_at = 0 OR expires_at > ?) ORDER BY created_at`, id, store.NowMS())
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		l := ShareLink{PageID: id}
		if err := rows.Scan(&l.ID, &l.Level, &l.Password, &l.ExpiresAt, &l.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out.Links = append(out.Links, l)
	}
	rows.Close()
	rows, err = s.st.R.QueryContext(ctx, `SELECT id, email, share_level, expires_at FROM invites WHERE page_id = ? AND accepted_at = 0 AND expires_at > ?`, id, store.NowMS())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var g GuestInvite
		if err := rows.Scan(&g.ID, &g.Email, &g.Level, &g.ExpiresAt); err != nil {
			return nil, err
		}
		out.Invites = append(out.Invites, g)
	}
	return out, rows.Err()
}

type ShareInput struct {
	Kind      string `json:"kind"`
	SubjectID string `json:"subject_id"`
	Email     string `json:"email"`
	Level     string `json:"level"`
	ExpiresAt int64  `json:"expires_at"`
}

type ShareResult struct {
	Share     *Share `json:"share,omitempty"`
	InviteURL string `json:"invite_url,omitempty"`
	Emailed   bool   `json:"emailed"`
}

func (s *Service) SetAuth(a *auth.Service) { s.auth = a }

func (s *Service) Share(ctx context.Context, p *policy.Principal, id string, in ShareInput) (*ShareResult, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.check(ctx, s.st.R, p, m, LevelFull); err != nil {
		return nil, err
	}
	if m.TrashedAt > 0 {
		return nil, errTrashed
	}
	level, ok := ParseLevel(in.Level)
	if !ok {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_level", "the level is full, edit, content, create, comment or view")
	}
	if in.ExpiresAt != 0 && in.ExpiresAt < store.NowMS() {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_expiry", "the expiry is in the past")
	}
	kind, subject := in.Kind, in.SubjectID
	if kind == "" {
		kind = "user"
	}
	switch kind {
	case "group":
		g, err := s.Group(ctx, subject)
		if err != nil || g.WorkspaceID != m.WorkspaceID {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_group", "that group is not in this workspace")
		}
	case "user":
		if subject == "" {
			email, err := auth.NormalizeEmail(in.Email)
			if err != nil {
				return nil, err
			}
			if err := s.st.R.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ? AND disabled = 0", email).Scan(&subject); err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return nil, err
				}
				return s.inviteGuest(ctx, p, m, email, level)
			}
		} else {
			var n int
			if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = ?", subject).Scan(&n); err != nil || n == 0 {
				return nil, httpx.Fail(http.StatusBadRequest, "bad_user", "no such person")
			}
		}
	default:
		return nil, httpx.Fail(http.StatusBadRequest, "bad_kind", "share with a user or a group")
	}
	before, err := s.Audience(ctx, m)
	if err != nil {
		return nil, err
	}
	shareID := ids.New()
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO page_shares (id, page_id, workspace_id, subject_kind, subject_id, level, expires_at, created_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (page_id, subject_kind, subject_id) DO UPDATE SET level = excluded.level, expires_at = excluded.expires_at`,
			shareID, id, m.WorkspaceID, kind, subject, level.String(), in.ExpiresAt, p.UserID, store.NowMS()); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.share", WorkspaceID: m.WorkspaceID, PageID: id, Detail: kind + " " + subject + " " + level.String()})
	})
	if err != nil {
		return nil, err
	}
	s.afterShare(ctx, m, before, kind, subject)
	sharing, err := s.Sharing(ctx, p, id)
	if err != nil {
		return nil, err
	}
	for i := range sharing.Shares {
		sh := sharing.Shares[i]
		if sh.PageID == id && sh.Kind == kind && sh.SubjectID == subject {
			return &ShareResult{Share: &sh}, nil
		}
	}
	return &ShareResult{}, nil
}

func (s *Service) afterShare(ctx context.Context, m *Meta, before events.Audience, kind, subject string) {
	if _, err := s.republish(ctx, m.WorkspaceID, m.ID, before); err != nil {
		s.log.Warn("page: share republish", "err", err)
	}
	if kind == "user" {
		s.hub.User(subject, "shares")
	} else {
		s.hub.Workspace(m.WorkspaceID, "shares")
	}
	for _, fn := range s.sharers {
		fn(ctx, m, kind, subject)
	}
}

func (s *Service) OnShare(fn func(ctx context.Context, m *Meta, kind, subject string)) {
	s.sharers = append(s.sharers, fn)
}

func (s *Service) shareOf(ctx context.Context, shareID string) (*Meta, string, string, error) {
	var page, kind, subject string
	if err := s.st.R.QueryRowContext(ctx, "SELECT page_id, subject_kind, subject_id FROM page_shares WHERE id = ?", shareID).Scan(&page, &kind, &subject); err != nil {
		return nil, "", "", store.NoRows(err)
	}
	m, err := s.load(ctx, s.st.R, page)
	return m, kind, subject, err
}

func (s *Service) UpdateShare(ctx context.Context, p *policy.Principal, shareID string, level string, expiresAt int64) error {
	m, kind, subject, err := s.shareOf(ctx, shareID)
	if err != nil {
		return err
	}
	if _, err := s.check(ctx, s.st.R, p, m, LevelFull); err != nil {
		return err
	}
	l, ok := ParseLevel(level)
	if !ok {
		return httpx.Fail(http.StatusBadRequest, "bad_level", "the level is full, edit, content, create, comment or view")
	}
	before, err := s.Audience(ctx, m)
	if err != nil {
		return err
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE page_shares SET level = ?, expires_at = ? WHERE id = ?", l.String(), expiresAt, shareID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.share", WorkspaceID: m.WorkspaceID, PageID: m.ID, Detail: kind + " " + subject + " " + l.String()})
	})
	if err != nil {
		return err
	}
	s.afterShare(ctx, m, before, kind, subject)
	return nil
}

func (s *Service) Unshare(ctx context.Context, p *policy.Principal, shareID string) error {
	m, kind, subject, err := s.shareOf(ctx, shareID)
	if err != nil {
		return err
	}
	if kind != "user" || subject != p.UserID {
		if _, err := s.check(ctx, s.st.R, p, m, LevelFull); err != nil {
			return err
		}
	}
	before, err := s.Audience(ctx, m)
	if err != nil {
		return err
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM page_shares WHERE id = ?", shareID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.unshare", WorkspaceID: m.WorkspaceID, PageID: m.ID, Detail: kind + " " + subject})
	})
	if err != nil {
		return err
	}
	s.afterShare(ctx, m, before, kind, subject)
	if kind == "user" {
		pages, _ := s.subtree(ctx, s.st.R, m.ID, true)
		ids := make([]string, 0, len(pages))
		for _, x := range pages {
			ids = append(ids, x.ID)
		}
		s.hub.UserRemoved(subject, m.WorkspaceID, ids)
	}
	return nil
}

const guestInviteTTL = 14 * 24 * time.Hour

func (s *Service) inviteGuest(ctx context.Context, p *policy.Principal, m *Meta, email string, level Level) (*ShareResult, error) {
	if s.auth == nil {
		return nil, httpx.Fail(http.StatusServiceUnavailable, "no_invites", "invitations are not available")
	}
	token := secrets.Token("", 24)
	now := time.Now()
	id := ids.New()
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO invites (id, token_hash, email, role, workspace_id, created_by, created_at, expires_at, page_id, share_level)
			VALUES (?, ?, ?, 'viewer', ?, ?, ?, ?, ?, ?)`, id, secrets.Hash(token), email, m.WorkspaceID, p.UserID, now.UnixMilli(), now.Add(guestInviteTTL).UnixMilli(), m.ID, level.String()); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.invite_guest", WorkspaceID: m.WorkspaceID, PageID: m.ID, Detail: email + " " + level.String()})
	})
	if err != nil {
		return nil, err
	}
	url := "/invite/" + token
	emailed := s.auth.SendInviteMail(ctx, email, p.Name, m.Title, url)
	return &ShareResult{InviteURL: url, Emailed: emailed}, nil
}

func (s *Service) CancelGuestInvite(ctx context.Context, p *policy.Principal, inviteID string) error {
	var page string
	if err := s.st.R.QueryRowContext(ctx, "SELECT COALESCE(page_id, '') FROM invites WHERE id = ?", inviteID).Scan(&page); err != nil || page == "" {
		return store.ErrNotFound
	}
	if _, err := s.CanShare(ctx, p, page); err != nil {
		return err
	}
	_, err := s.st.W.ExecContext(ctx, "DELETE FROM invites WHERE id = ? AND accepted_at = 0", inviteID)
	return err
}

func (s *Service) AcceptGuest(ctx context.Context, tx *sql.Tx, userID, pageID, level string) error {
	var ws string
	if err := tx.QueryRowContext(ctx, "SELECT workspace_id FROM pages WHERE id = ?", pageID).Scan(&ws); err != nil {
		return store.NoRows(err)
	}
	if _, ok := ParseLevel(level); !ok {
		level = "view"
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO page_shares (id, page_id, workspace_id, subject_kind, subject_id, level, created_by, created_at)
		VALUES (?, ?, ?, 'user', ?, ?, '', ?) ON CONFLICT (page_id, subject_kind, subject_id) DO UPDATE SET level = excluded.level`,
		ids.New(), pageID, ws, userID, level, store.NowMS())
	return err
}

type LinkInput struct {
	Level     string `json:"level"`
	ExpiresAt int64  `json:"expires_at"`
	Password  string `json:"password"`
}

func (s *Service) CreateLink(ctx context.Context, p *policy.Principal, id string, in LinkInput) (*ShareLink, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.check(ctx, s.st.R, p, m, LevelFull); err != nil {
		return nil, err
	}
	if in.Level == "" {
		in.Level = "view"
	}
	if in.Level != "view" && in.Level != "comment" {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_level", "a link gives view or comment access")
	}
	if in.ExpiresAt != 0 && in.ExpiresAt < store.NowMS() {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_expiry", "the expiry is in the past")
	}
	hash := ""
	if in.Password != "" {
		if hash, err = auth.HashPassword(ctx, in.Password); err != nil {
			return nil, err
		}
	}
	token := secrets.Token("", 18)
	l := &ShareLink{ID: ids.New(), PageID: id, Level: in.Level, Password: hash != "", ExpiresAt: in.ExpiresAt, CreatedAt: store.NowMS(), Token: token}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO share_links (id, page_id, workspace_id, token_hash, level, password_hash, expires_at, created_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, l.ID, id, m.WorkspaceID, secrets.Hash(token), l.Level, hash, l.ExpiresAt, p.UserID, l.CreatedAt); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.link", WorkspaceID: m.WorkspaceID, PageID: id, Detail: l.Level})
	})
	return l, err
}

func (s *Service) RevokeLink(ctx context.Context, p *policy.Principal, linkID string) error {
	var page string
	if err := s.st.R.QueryRowContext(ctx, "SELECT page_id FROM share_links WHERE id = ?", linkID).Scan(&page); err != nil {
		return store.NoRows(err)
	}
	if _, err := s.CanShare(ctx, p, page); err != nil {
		return err
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE share_links SET revoked_at = ? WHERE id = ?", store.NowMS(), linkID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM page_shares WHERE created_by = ?", "link:"+linkID)
		return err
	})
}

type LinkAccess struct {
	LinkID      string
	PageID      string
	WorkspaceID string
	Level       string
	Password    string
	ExpiresAt   int64
}

func (s *Service) ResolveLink(ctx context.Context, token string) (*LinkAccess, error) {
	var l LinkAccess
	err := s.st.R.QueryRowContext(ctx, `SELECT l.id, l.page_id, l.workspace_id, l.level, l.password_hash, l.expires_at FROM share_links l JOIN pages p ON p.id = l.page_id
		WHERE l.token_hash = ? AND l.revoked_at = 0 AND (l.expires_at = 0 OR l.expires_at > ?) AND p.trashed_at = 0`, secrets.Hash(token), store.NowMS()).
		Scan(&l.LinkID, &l.PageID, &l.WorkspaceID, &l.Level, &l.Password, &l.ExpiresAt)
	if err != nil {
		return nil, store.NoRows(err)
	}
	return &l, nil
}

func (s *Service) GrantLink(ctx context.Context, p *policy.Principal, l *LinkAccess) error {
	if !p.IsSession() {
		return policy.ErrHidden
	}
	want, ok := ParseLevel(l.Level)
	if !ok {
		want = LevelView
	}
	if have, _, err := s.Level(ctx, p, l.PageID); err == nil && have >= want {
		return nil
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO page_shares (id, page_id, workspace_id, subject_kind, subject_id, level, expires_at, created_by, created_at)
			VALUES (?, ?, ?, 'user', ?, ?, ?, ?, ?) ON CONFLICT (page_id, subject_kind, subject_id) DO NOTHING`,
			ids.New(), l.PageID, l.WorkspaceID, p.UserID, want.String(), l.ExpiresAt, "link:"+l.LinkID, store.NowMS()); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.link_open", WorkspaceID: l.WorkspaceID, PageID: l.PageID, Detail: l.Level})
	})
}

type SharedPage struct {
	*Meta
	WorkspaceName string `json:"workspace_name"`
	Level         string `json:"level"`
}

func (s *Service) SharedWithMe(ctx context.Context, p *policy.Principal) ([]SharedPage, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT DISTINCT ps.page_id FROM page_shares ps
		WHERE (ps.expires_at = 0 OR ps.expires_at > ?) AND ((ps.subject_kind = 'user' AND ps.subject_id = ?)
			OR (ps.subject_kind = 'group' AND ps.subject_id IN (SELECT group_id FROM group_members WHERE user_id = ?)))`, store.NowMS(), p.UserID, p.UserID)
	if err != nil {
		return nil, err
	}
	var pageIDs []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			pageIDs = append(pageIDs, id)
		}
	}
	rows.Close()
	out := []SharedPage{}
	for _, id := range pageIDs {
		m, l, err := s.ReadableLevel(ctx, p, id)
		if err != nil || m.TrashedAt > 0 {
			continue
		}
		var name string
		_ = s.st.R.QueryRowContext(ctx, "SELECT name FROM workspaces WHERE id = ?", m.WorkspaceID).Scan(&name)
		out = append(out, SharedPage{Meta: m, WorkspaceName: name, Level: l.String()})
	}
	return out, nil
}
