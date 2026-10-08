package workspace

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/icon"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const MaxRules = 16000

type Service struct {
	st    *store.Store
	gate  *policy.Gate
	audit *audit.Log
	log   *slog.Logger
	hooks []func(ctx context.Context, tx *sql.Tx, workspaceID string) error

	mayCreate  func(p *policy.Principal) bool
	imageCheck func(ctx context.Context, ref, workspaceID string) bool
	notifier   Notifier
}

func (s *Service) SetImageCheck(fn func(ctx context.Context, ref, workspaceID string) bool) {
	s.imageCheck = fn
}

type Notifier interface {
	Workspace(workspaceID, what string)
	User(userID, what string)
}

func (s *Service) SetNotifier(n Notifier) {
	s.notifier = n
}

func (s *Service) Changed(workspaceID, what string, users ...string) {
	if s.notifier == nil {
		return
	}
	if workspaceID != "" {
		s.notifier.Workspace(workspaceID, what)
	}
	for _, u := range users {
		s.notifier.User(u, "workspaces")
	}
}

func (s *Service) MemberIDs(ctx context.Context, workspaceID string) []string {
	rows, err := s.st.R.QueryContext(ctx, "SELECT user_id FROM workspace_members WHERE workspace_id = ?", workspaceID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out
}

func New(st *store.Store, gate *policy.Gate, al *audit.Log, log *slog.Logger) *Service {
	return &Service{st: st, gate: gate, audit: al, log: log}
}

func (s *Service) OnDelete(fn func(ctx context.Context, tx *sql.Tx, workspaceID string) error) {
	s.hooks = append(s.hooks, fn)
}

type Workspace struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Icon           string `json:"icon"`
	Image          string `json:"image"`
	Role           string `json:"role"`
	Personal       bool   `json:"personal"`
	AutoJoin       bool   `json:"auto_join"`
	SidebarMode    string `json:"sidebar_mode"`
	AgentAccess    string `json:"agent_access"`
	HasRules       bool   `json:"has_rules"`
	ProposalWaits  bool   `json:"proposal_waiting"`
	MemberCount    int    `json:"member_count"`
	BreakGlass     bool   `json:"break_glass,omitempty"`
	OwnerID        string `json:"owner_id"`
	RulesUpdatedAt int64  `json:"rules_updated_at"`
}

const wsColumns = `w.id, w.name, w.icon, w.image, w.personal, w.auto_join, w.sidebar_mode, w.agent_access,
	w.rules != '', w.rules_proposal != '', COALESCE(w.owner_id, ''), w.rules_updated_at,
	(SELECT COUNT(*) FROM workspace_members m2 WHERE m2.workspace_id = w.id)`

func scanWorkspace(row interface{ Scan(...any) error }, extra ...any) (*Workspace, error) {
	w := &Workspace{}
	var personal, autoJoin, hasRules, proposal int
	dest := append([]any{&w.ID, &w.Name, &w.Icon, &w.Image, &personal, &autoJoin, &w.SidebarMode, &w.AgentAccess,
		&hasRules, &proposal, &w.OwnerID, &w.RulesUpdatedAt, &w.MemberCount}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, store.NoRows(err)
	}
	w.Personal, w.AutoJoin, w.HasRules, w.ProposalWaits = personal == 1, autoJoin == 1, hasRules == 1, proposal == 1
	return w, nil
}

func (s *Service) List(ctx context.Context, p *policy.Principal) ([]*Workspace, int, error) {
	rows, err := s.st.R.QueryContext(ctx, "SELECT "+wsColumns+`, m.role
		FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.user_id = ? ORDER BY w.personal DESC, w.name COLLATE NOCASE`, p.UserID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*Workspace{}
	withheld := 0
	for rows.Next() {
		var role string
		w, err := scanWorkspace(rows, &role)
		if err != nil {
			return nil, 0, err
		}
		if !p.Reaches(w.ID) || !policy.AgentAllowed(p, w.AgentAccess) {
			withheld++
			continue
		}
		w.Role = role
		if role != "admin" {
			w.ProposalWaits = false
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if p.IsOwner() {
		bg, err := s.st.R.QueryContext(ctx, "SELECT "+wsColumns+` FROM workspaces w
			WHERE w.id IN (SELECT workspace_id FROM break_glass WHERE user_id = ? AND revoked_at = 0 AND expires_at > ?)
			AND w.id NOT IN (SELECT workspace_id FROM workspace_members WHERE user_id = ?)`, p.UserID, store.NowMS(), p.UserID)
		if err != nil {
			return nil, 0, err
		}
		defer bg.Close()
		for bg.Next() {
			w, err := scanWorkspace(bg)
			if err != nil {
				return nil, 0, err
			}
			w.Role, w.BreakGlass, w.ProposalWaits = "viewer", true, false
			out = append(out, w)
		}
	}
	listed := make(map[string]bool, len(out))
	for _, w := range out {
		listed[w.ID] = true
	}
	guest, err := s.st.R.QueryContext(ctx, "SELECT "+wsColumns+` FROM workspaces w
		WHERE w.id IN (SELECT ps.workspace_id FROM page_shares ps JOIN pages pg ON pg.id = ps.page_id WHERE pg.trashed_at = 0
			AND (ps.expires_at = 0 OR ps.expires_at > ?) AND ps.subject_kind = 'user' AND ps.subject_id = ?)
		AND w.id NOT IN (SELECT workspace_id FROM workspace_members WHERE user_id = ?) ORDER BY w.name COLLATE NOCASE`, store.NowMS(), p.UserID, p.UserID)
	if err != nil {
		return nil, 0, err
	}
	defer guest.Close()
	for guest.Next() {
		w, err := scanWorkspace(guest)
		if err != nil {
			return nil, 0, err
		}
		if listed[w.ID] || !p.Reaches(w.ID) || !policy.AgentAllowed(p, w.AgentAccess) {
			continue
		}
		w.Role, w.ProposalWaits, w.MemberCount, w.OwnerID = "guest", false, 0, ""
		out = append(out, w)
	}
	return out, withheld, guest.Err()
}

func (s *Service) Get(ctx context.Context, p *policy.Principal, id string) (*Workspace, policy.Access, error) {
	a, err := s.gate.Workspace(ctx, p, id)
	if err != nil {
		return nil, a, err
	}
	w, err := scanWorkspace(s.st.R.QueryRowContext(ctx, "SELECT "+wsColumns+" FROM workspaces w WHERE w.id = ?", id))
	if err != nil {
		return nil, a, err
	}
	w.Role, w.BreakGlass = a.Role, a.BreakGlass
	if !a.IsAdmin() {
		w.ProposalWaits = false
	}
	return w, a, nil
}

func cleanName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return "", httpx.Fail(http.StatusBadRequest, "bad_name", "enter a name of up to 80 characters")
	}
	return name, nil
}

func cleanIcon(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if !icon.Valid(v) {
		return "", httpx.Fail(http.StatusBadRequest, "bad_icon", "an icon is a name from the icon set or an uploaded image")
	}
	return v, nil
}

func (s *Service) insert(ctx context.Context, q store.Querier, name, icon, ownerID string, personal bool) (string, error) {
	id := ids.New()
	now := store.NowMS()
	if _, err := q.ExecContext(ctx, `INSERT INTO workspaces (id, name, icon, owner_id, personal, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, id, name, icon, ownerID, store.Bool(personal), now, now); err != nil {
		return "", err
	}
	if _, err := q.ExecContext(ctx, "INSERT INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, 'admin', ?)", id, ownerID, now); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) Provision(ctx context.Context, tx *sql.Tx, userID, name string, owner bool) error {
	if _, err := s.insert(ctx, tx, "Personal", "", userID, true); err != nil {
		return err
	}
	if owner {
		if _, err := s.insert(ctx, tx, "Team", "", userID, false); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, joined_at)
		SELECT id, ?, 'member', ? FROM workspaces WHERE auto_join = 1 AND personal = 0`, userID, store.NowMS())
	return err
}

func (s *Service) JoinInvite(ctx context.Context, tx *sql.Tx, userID, workspaceID, role string) error {
	_, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)",
		workspaceID, userID, role, store.NowMS())
	return err
}

func (s *Service) SetCreatePolicy(fn func(p *policy.Principal) bool) {
	s.mayCreate = fn
}

func (s *Service) MayCreate(p *policy.Principal) bool {
	return s.mayCreate == nil || s.mayCreate(p)
}

func (s *Service) Create(ctx context.Context, p *policy.Principal, name, icon string) (*Workspace, error) {
	if !p.IsSession() && !p.AllWorkspaces {
		return nil, httpx.Fail(http.StatusForbidden, "credential_narrowed", "a credential limited to chosen workspaces cannot create workspaces")
	}
	if !s.MayCreate(p) {
		return nil, httpx.Fail(http.StatusForbidden, "create_closed", "only admins create workspaces on this instance")
	}
	name, err := cleanName(name)
	if err != nil {
		return nil, err
	}
	if icon, err = cleanIcon(icon); err != nil {
		return nil, err
	}
	var id string
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if id, err = s.insert(ctx, tx, name, icon, p.UserID, false); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.create", WorkspaceID: id, Detail: name})
	})
	if err != nil {
		return nil, err
	}
	w, _, err := s.Get(ctx, p, id)
	return w, err
}

type Update struct {
	Name        *string `json:"name"`
	Icon        *string `json:"icon"`
	Image       *string `json:"image"`
	SidebarMode *string `json:"sidebar_mode"`
	AgentAccess *string `json:"agent_access"`
	AutoJoin    *bool   `json:"auto_join"`
}

func (s *Service) Update(ctx context.Context, p *policy.Principal, id string, in Update) (*Workspace, error) {
	cur, a, err := s.Get(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !a.IsAdmin() {
		return nil, httpx.ErrForbidden
	}
	sets, args := []string{}, []any{}
	var changes []audit.Change
	if in.Name != nil {
		name, err := cleanName(*in.Name)
		if err != nil {
			return nil, err
		}
		sets, args = append(sets, "name = ?"), append(args, name)
		changes = append(changes, audit.Change{Field: "name", Before: cur.Name, After: name})
	}
	if in.Icon != nil {
		icon, err := cleanIcon(*in.Icon)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(icon, "f:") && (s.imageCheck == nil || !s.imageCheck(ctx, icon, id)) {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_image", "upload the picture to this workspace first")
		}
		sets, args = append(sets, "icon = ?"), append(args, icon)
	}
	if in.Image != nil {
		v := strings.TrimSpace(*in.Image)
		if v != "" && (s.imageCheck == nil || !s.imageCheck(ctx, v, id)) {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_image", "upload the picture to this workspace first")
		}
		sets, args = append(sets, "image = ?"), append(args, v)
	}
	if in.SidebarMode != nil {
		if *in.SidebarMode != "split" && *in.SidebarMode != "tree" {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_sidebar_mode", "sidebar mode must be split or tree")
		}
		sets, args = append(sets, "sidebar_mode = ?"), append(args, *in.SidebarMode)
	}
	if in.AgentAccess != nil {
		if !p.IsSession() {
			return nil, httpx.ErrSessionOnly
		}
		v := *in.AgentAccess
		if v != "open" && v != "strict" && v != "closed" {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_agent_access", "agent access must be open, strict or closed")
		}
		sets, args = append(sets, "agent_access = ?"), append(args, v)
		changes = append(changes, audit.Change{Field: "agent_access", Before: cur.AgentAccess, After: v})
	}
	if in.AutoJoin != nil {
		if !p.IsSession() || !p.IsOwner() {
			return nil, httpx.Fail(http.StatusForbidden, "owner_only", "only the instance owner chooses auto-join workspaces")
		}
		if cur.Personal {
			return nil, httpx.Fail(http.StatusConflict, "personal_space", "personal spaces cannot be auto-joined")
		}
		sets, args = append(sets, "auto_join = ?"), append(args, store.Bool(*in.AutoJoin))
	}
	if len(sets) == 0 {
		return cur, nil
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, store.NowMS(), id)
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE workspaces SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.update", WorkspaceID: id, Changes: changes})
	})
	if err != nil {
		return nil, err
	}
	w, _, err := s.Get(ctx, p, id)
	return w, err
}

func (s *Service) Delete(ctx context.Context, p *policy.Principal, id string) error {
	cur, a, err := s.Get(ctx, p, id)
	if err != nil {
		return err
	}
	if !a.IsAdmin() || !p.IsSession() {
		return httpx.ErrForbidden
	}
	if cur.Personal {
		return httpx.Fail(http.StatusConflict, "personal_space", "a personal space is removed only with its account")
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if err := s.Purge(ctx, tx, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.delete", WorkspaceID: id, Detail: cur.Name})
	})
}

func (s *Service) Purge(ctx context.Context, tx *sql.Tx, id string) error {
	for _, fn := range s.hooks {
		if err := fn(ctx, tx, id); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, "DELETE FROM workspaces WHERE id = ?", id)
	return err
}

type Member struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Color    string `json:"color"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
	JoinedAt int64  `json:"joined_at"`
	Disabled bool   `json:"disabled"`
}

func (s *Service) Members(ctx context.Context, p *policy.Principal, id string) ([]Member, error) {
	if _, err := s.gate.Workspace(ctx, p, id); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT u.id, u.name, u.email, u.color, u.avatar, m.role, m.joined_at, u.disabled
		FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = ? ORDER BY u.name COLLATE NOCASE`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		var disabled int
		if err := rows.Scan(&m.UserID, &m.Name, &m.Email, &m.Color, &m.Avatar, &m.Role, &m.JoinedAt, &disabled); err != nil {
			return nil, err
		}
		m.Disabled = disabled == 1
		if !p.IsSession() {
			m.Email = ""
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) adminCount(ctx context.Context, q store.Querier, id string) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = ? AND role = 'admin'", id).Scan(&n)
	return n, err
}

var errLastAdmin = httpx.Fail(http.StatusConflict, "last_admin", "a workspace needs at least one admin")

func (s *Service) requireAdmin(ctx context.Context, p *policy.Principal, id string) (*Workspace, error) {
	if !p.IsSession() {
		return nil, httpx.ErrSessionOnly
	}
	w, a, err := s.Get(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !a.IsAdmin() {
		return nil, httpx.ErrForbidden
	}
	return w, nil
}

func (s *Service) AddMember(ctx context.Context, p *policy.Principal, id, email, role string) error {
	w, err := s.requireAdmin(ctx, p, id)
	if err != nil {
		return err
	}
	if w.Personal {
		return httpx.Fail(http.StatusConflict, "personal_space", "personal spaces cannot have other members")
	}
	if role != "admin" && role != "member" && role != "viewer" {
		return httpx.Fail(http.StatusBadRequest, "bad_role", "role must be admin, member or viewer")
	}
	var userID string
	err = s.st.R.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ? AND disabled = 0", strings.ToLower(strings.TrimSpace(email))).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.Fail(http.StatusNotFound, "no_such_user", "no active account uses that email; send an invitation instead")
	}
	if err != nil {
		return err
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)", id, userID, role, store.NowMS())
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return httpx.Fail(http.StatusConflict, "already_member", "that person is already a member")
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "member.add", WorkspaceID: id, Detail: userID + " as " + role})
	})
	if err == nil {
		s.Changed(id, "members", userID)
	}
	return err
}

func (s *Service) SetRole(ctx context.Context, p *policy.Principal, id, userID, role string) error {
	if _, err := s.requireAdmin(ctx, p, id); err != nil {
		return err
	}
	if role != "admin" && role != "member" && role != "viewer" {
		return httpx.Fail(http.StatusBadRequest, "bad_role", "role must be admin, member or viewer")
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		var cur string
		if err := tx.QueryRowContext(ctx, "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", id, userID).Scan(&cur); err != nil {
			return store.NoRows(err)
		}
		if cur == "admin" && role != "admin" {
			n, err := s.adminCount(ctx, tx, id)
			if err != nil {
				return err
			}
			if n <= 1 {
				return errLastAdmin
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?", role, id, userID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "member.role", WorkspaceID: id,
			Changes: []audit.Change{{Field: "role:" + userID, Before: cur, After: role}}})
	})
}

func (s *Service) RemoveMember(ctx context.Context, p *policy.Principal, id, userID string) error {
	self := userID == p.UserID
	var w *Workspace
	var err error
	if self {
		if !p.IsSession() {
			return httpx.ErrSessionOnly
		}
		w, _, err = s.Get(ctx, p, id)
	} else {
		w, err = s.requireAdmin(ctx, p, id)
	}
	if err != nil {
		return err
	}
	if w.Personal {
		return httpx.Fail(http.StatusConflict, "personal_space", "you cannot leave your personal space")
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		var cur string
		if err := tx.QueryRowContext(ctx, "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", id, userID).Scan(&cur); err != nil {
			return store.NoRows(err)
		}
		if cur == "admin" {
			n, err := s.adminCount(ctx, tx, id)
			if err != nil {
				return err
			}
			if n <= 1 {
				return errLastAdmin
			}
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?", id, userID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "member.remove", WorkspaceID: id, Detail: userID})
	})
}

func (s *Service) SetAccess(ctx context.Context, tx *sql.Tx, id, userID, role string) (string, error) {
	var personal int
	if err := tx.QueryRowContext(ctx, "SELECT personal FROM workspaces WHERE id = ?", id).Scan(&personal); err != nil {
		return "", store.NoRows(err)
	}
	if personal == 1 {
		return "", httpx.Fail(http.StatusConflict, "personal_space", "personal spaces cannot have other members")
	}
	if role != "" && role != "admin" && role != "member" && role != "viewer" {
		return "", httpx.Fail(http.StatusBadRequest, "bad_role", "role must be admin, member, viewer or empty")
	}
	var before string
	err := tx.QueryRowContext(ctx, "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", id, userID).Scan(&before)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if before == role {
		return before, nil
	}
	if before == "admin" {
		n, err := s.adminCount(ctx, tx, id)
		if err != nil {
			return "", err
		}
		if n <= 1 {
			return "", errLastAdmin
		}
	}
	switch {
	case role == "":
		_, err = tx.ExecContext(ctx, "DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?", id, userID)
	case before == "":
		_, err = tx.ExecContext(ctx, "INSERT INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)", id, userID, role, store.NowMS())
	default:
		_, err = tx.ExecContext(ctx, "UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?", role, id, userID)
	}
	return before, err
}

type Rules struct {
	Rules          string `json:"rules"`
	UpdatedAt      int64  `json:"updated_at"`
	Proposal       string `json:"proposal,omitempty"`
	ProposalBy     string `json:"proposal_by,omitempty"`
	ProposalByName string `json:"proposal_by_name,omitempty"`
	ProposalAt     int64  `json:"proposal_at,omitempty"`
	CanEdit        bool   `json:"can_edit"`
}

func (s *Service) GetRules(ctx context.Context, p *policy.Principal, id string) (*Rules, error) {
	a, err := s.gate.Workspace(ctx, p, id)
	if err != nil {
		return nil, err
	}
	r := &Rules{}
	var by sql.NullString
	err = s.st.R.QueryRowContext(ctx, `SELECT w.rules, w.rules_updated_at, w.rules_proposal, w.rules_proposal_by, COALESCE(u.name, ''), w.rules_proposal_at
		FROM workspaces w LEFT JOIN users u ON u.id = w.rules_proposal_by WHERE w.id = ?`, id).
		Scan(&r.Rules, &r.UpdatedAt, &r.Proposal, &by, &r.ProposalByName, &r.ProposalAt)
	if err != nil {
		return nil, store.NoRows(err)
	}
	r.ProposalBy = by.String
	r.CanEdit = a.IsAdmin() && p.IsSession()
	if !a.IsAdmin() {
		r.Proposal, r.ProposalBy, r.ProposalByName, r.ProposalAt = "", "", "", 0
	}
	return r, nil
}

func (s *Service) SetRules(ctx context.Context, p *policy.Principal, id, text string) error {
	if _, err := s.requireAdmin(ctx, p, id); err != nil {
		return err
	}
	if utf8.RuneCountInString(text) > MaxRules {
		return httpx.Fail(http.StatusBadRequest, "rules_too_long", "rules can be up to 16,000 characters")
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE workspaces SET rules = ?, rules_updated_at = ?, rules_proposal = '',
			rules_proposal_by = NULL, rules_proposal_at = 0 WHERE id = ?`, text, store.NowMS(), id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.rules", WorkspaceID: id})
	})
}

func (s *Service) DismissProposal(ctx context.Context, p *policy.Principal, id string) error {
	if _, err := s.requireAdmin(ctx, p, id); err != nil {
		return err
	}
	_, err := s.st.W.ExecContext(ctx, "UPDATE workspaces SET rules_proposal = '', rules_proposal_by = NULL, rules_proposal_at = 0 WHERE id = ?", id)
	return err
}

func (s *Service) ProposeRules(ctx context.Context, p *policy.Principal, id, text string) error {
	a, err := s.gate.Workspace(ctx, p, id)
	if err != nil {
		return err
	}
	if a.BreakGlass || a.Role != "admin" {
		return httpx.Fail(http.StatusForbidden, "admins_only", "only an admin of this workspace may propose rules")
	}
	if utf8.RuneCountInString(text) > MaxRules {
		return httpx.Fail(http.StatusBadRequest, "rules_too_long", "rules can be up to 16,000 characters")
	}
	if strings.TrimSpace(text) == "" {
		_, err = s.st.W.ExecContext(ctx, `UPDATE workspaces SET rules_proposal = '', rules_proposal_by = NULL, rules_proposal_at = 0
			WHERE id = ? AND rules_proposal_by = ?`, id, p.UserID)
		return err
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE workspaces SET rules_proposal = ?, rules_proposal_by = ?, rules_proposal_at = ? WHERE id = ?",
			text, p.UserID, store.NowMS(), id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.rules_proposal", WorkspaceID: id})
	})
}

type Emergency struct {
	ID        string `json:"id"`
	UserName  string `json:"user_name"`
	Reason    string `json:"reason"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
	RevokedAt int64  `json:"revoked_at"`
}

func (s *Service) EmergencyLog(ctx context.Context, p *policy.Principal, id string) ([]Emergency, error) {
	if _, err := s.requireAdmin(ctx, p, id); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT b.id, u.name, b.reason, b.created_at, b.expires_at, b.revoked_at
		FROM break_glass b JOIN users u ON u.id = b.user_id WHERE b.workspace_id = ? ORDER BY b.created_at DESC LIMIT 200`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Emergency{}
	for rows.Next() {
		var e Emergency
		if err := rows.Scan(&e.ID, &e.UserName, &e.Reason, &e.CreatedAt, &e.ExpiresAt, &e.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Service) EndEmergency(ctx context.Context, p *policy.Principal, id, grantID string) error {
	if _, err := s.requireAdmin(ctx, p, id); err != nil {
		return err
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		now := store.NowMS()
		res, err := tx.ExecContext(ctx, "UPDATE break_glass SET revoked_at = ? WHERE id = ? AND workspace_id = ? AND revoked_at = 0 AND expires_at > ?", now, grantID, id, now)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return httpx.ErrNotFound
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "workspace.break_glass_end", WorkspaceID: id, Detail: grantID})
	})
}
