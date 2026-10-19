package page

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/icon"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/order"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Teamspace struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Access      string `json:"access"`
	SortKey     string `json:"sort_key"`
	Role        string `json:"role"`
	Members     int    `json:"members"`
	CanManage   bool   `json:"can_manage"`
}

type TeamMember struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

var teamAccess = map[string]bool{"open": true, "closed": true, "private": true}

func cleanName(raw string, max int) (string, error) {
	n := strings.TrimSpace(strings.ReplaceAll(raw, "\n", " "))
	if n == "" || utf8.RuneCountInString(n) > max {
		return "", httpx.Fail(http.StatusBadRequest, "bad_name", "a name is needed, up to 100 characters")
	}
	return n, nil
}

func (s *Service) Teamspaces(ctx context.Context, p *policy.Principal, workspaceID string) ([]Teamspace, error) {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if errors.Is(err, policy.ErrHidden) {
		if err := s.Reach(ctx, p, workspaceID); err != nil {
			return nil, err
		}
		return []Teamspace{}, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT t.id, t.name, t.icon, t.description, t.access, t.sort_key, COALESCE(m.role, ''),
			(SELECT COUNT(*) FROM teamspace_members x WHERE x.teamspace_id = t.id)
		FROM teamspaces t LEFT JOIN teamspace_members m ON m.teamspace_id = t.id AND m.user_id = ?
		WHERE t.workspace_id = ? ORDER BY t.sort_key`, p.UserID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Teamspace{}
	for rows.Next() {
		t := Teamspace{WorkspaceID: workspaceID}
		if err := rows.Scan(&t.ID, &t.Name, &t.Icon, &t.Description, &t.Access, &t.SortKey, &t.Role, &t.Members); err != nil {
			return nil, err
		}
		if t.Access == "private" && t.Role == "" {
			continue
		}
		t.CanManage = !a.BreakGlass && (t.Role == "owner" || a.IsAdmin())
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) teamspace(ctx context.Context, q store.Querier, id string) (*Teamspace, error) {
	var t Teamspace
	err := q.QueryRowContext(ctx, "SELECT id, workspace_id, name, icon, description, access, sort_key FROM teamspaces WHERE id = ?", id).
		Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Icon, &t.Description, &t.Access, &t.SortKey)
	if err != nil {
		return nil, store.NoRows(err)
	}
	return &t, nil
}

func (s *Service) teamRole(ctx context.Context, team, user string) string {
	var role string
	_ = s.st.R.QueryRowContext(ctx, "SELECT role FROM teamspace_members WHERE teamspace_id = ? AND user_id = ?", team, user).Scan(&role)
	return role
}

func (s *Service) teamWritable(ctx context.Context, p *policy.Principal, workspaceID, team string) error {
	t, err := s.teamspace(ctx, s.st.R, team)
	if err != nil || t.WorkspaceID != workspaceID {
		return httpx.Fail(http.StatusBadRequest, "bad_teamspace", "that teamspace is not in this workspace")
	}
	if t.Access == "open" {
		return nil
	}
	if s.teamRole(ctx, team, p.UserID) == "" {
		return policy.ErrHidden
	}
	return nil
}

func (s *Service) manageTeam(ctx context.Context, p *policy.Principal, id string) (*Teamspace, policy.Access, error) {
	t, err := s.teamspace(ctx, s.st.R, id)
	if err != nil {
		return nil, policy.Access{}, err
	}
	a, err := s.gate.Workspace(ctx, p, t.WorkspaceID)
	if err != nil {
		return nil, a, err
	}
	role := s.teamRole(ctx, id, p.UserID)
	if t.Access == "private" && role == "" && !a.IsAdmin() {
		return nil, a, policy.ErrHidden
	}
	if a.BreakGlass || (role != "owner" && !a.IsAdmin()) {
		return t, a, httpx.ErrForbidden
	}
	return t, a, nil
}

type TeamInput struct {
	Name        *string `json:"name"`
	Icon        *string `json:"icon"`
	Description *string `json:"description"`
	Access      *string `json:"access"`
}

func (s *Service) CreateTeamspace(ctx context.Context, p *policy.Principal, workspaceID string, in TeamInput) (*Teamspace, error) {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	if !a.CanWrite() || a.Personal {
		return nil, httpx.ErrForbidden
	}
	if in.Name == nil {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_name", "a name is needed, up to 100 characters")
	}
	name, err := cleanName(*in.Name, 100)
	if err != nil {
		return nil, err
	}
	access := "open"
	if in.Access != nil {
		access = *in.Access
	}
	if !teamAccess[access] {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_access", "access is open, closed or private")
	}
	ic, desc := "", ""
	if in.Icon != nil {
		ic = icon.Normalize(strings.TrimSpace(*in.Icon))
	}
	if !icon.Valid(ic) || strings.HasPrefix(ic, "f:") {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_icon", "an icon is a name from the icon set")
	}
	if in.Description != nil {
		desc = strings.TrimSpace(*in.Description)
	}
	id := ids.New()
	now := store.NowMS()
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		var last sql.NullString
		if err := tx.QueryRowContext(ctx, "SELECT MAX(sort_key) FROM teamspaces WHERE workspace_id = ?", workspaceID).Scan(&last); err != nil {
			return err
		}
		key := order.After(last.String)
		if _, err := tx.ExecContext(ctx, `INSERT INTO teamspaces (id, workspace_id, name, icon, description, access, sort_key, created_at, created_by, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceID, name, ic, desc, access, key, now, p.UserID, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO teamspace_members (teamspace_id, user_id, role, joined_at) VALUES (?, ?, 'owner', ?)", id, p.UserID, now); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "teamspace.create", WorkspaceID: workspaceID, Detail: name})
	})
	if err != nil {
		return nil, err
	}
	s.hub.Workspace(workspaceID, "teamspaces")
	t, err := s.teamspace(ctx, s.st.R, id)
	if err == nil {
		t.Role, t.Members, t.CanManage = "owner", 1, true
	}
	return t, err
}

func (s *Service) UpdateTeamspace(ctx context.Context, p *policy.Principal, id string, in TeamInput) (*Teamspace, error) {
	t, _, err := s.manageTeam(ctx, p, id)
	if err != nil {
		return nil, err
	}
	sets, args := []string{}, []any{}
	if in.Name != nil {
		n, err := cleanName(*in.Name, 100)
		if err != nil {
			return nil, err
		}
		sets, args = append(sets, "name = ?"), append(args, n)
	}
	if in.Icon != nil {
		v := icon.Normalize(strings.TrimSpace(*in.Icon))
		if !icon.Valid(v) || strings.HasPrefix(v, "f:") {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_icon", "an icon is a name from the icon set")
		}
		sets, args = append(sets, "icon = ?"), append(args, v)
	}
	if in.Description != nil {
		sets, args = append(sets, "description = ?"), append(args, strings.TrimSpace(*in.Description))
	}
	if in.Access != nil {
		if !teamAccess[*in.Access] {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_access", "access is open, closed or private")
		}
		sets, args = append(sets, "access = ?"), append(args, *in.Access)
	}
	if len(sets) == 0 {
		return t, nil
	}
	sets, args = append(sets, "updated_at = ?"), append(args, store.NowMS(), id)
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE teamspaces SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "teamspace.update", WorkspaceID: t.WorkspaceID, Detail: t.Name})
	})
	if err != nil {
		return nil, err
	}
	s.hub.Workspace(t.WorkspaceID, "teamspaces")
	if in.Access != nil {
		s.hub.Workspace(t.WorkspaceID, "pages")
	}
	return s.teamspace(ctx, s.st.R, id)
}

func (s *Service) DeleteTeamspace(ctx context.Context, p *policy.Principal, id string) error {
	t, _, err := s.manageTeam(ctx, p, id)
	if err != nil {
		return err
	}
	var n int
	if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM pages WHERE teamspace_id = ? AND trashed_at = 0", id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return httpx.Fail(http.StatusConflict, "teamspace_not_empty", "move or trash the teamspace's pages first")
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM teamspaces WHERE id = ?", id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "teamspace.delete", WorkspaceID: t.WorkspaceID, Detail: t.Name})
	})
	if err != nil {
		return err
	}
	s.hub.Workspace(t.WorkspaceID, "teamspaces")
	return nil
}

func (s *Service) TeamMembers(ctx context.Context, p *policy.Principal, id string) ([]TeamMember, error) {
	t, err := s.teamspace(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	a, err := s.gate.Workspace(ctx, p, t.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if t.Access == "private" && s.teamRole(ctx, id, p.UserID) == "" && !a.IsAdmin() {
		return nil, policy.ErrHidden
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT u.id, u.name, u.email, m.role FROM teamspace_members m JOIN users u ON u.id = m.user_id
		WHERE m.teamspace_id = ? ORDER BY m.role DESC, u.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TeamMember{}
	for rows.Next() {
		var m TeamMember
		if err := rows.Scan(&m.UserID, &m.Name, &m.Email, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) SetTeamMember(ctx context.Context, p *policy.Principal, id, user, role string) error {
	if role != "owner" && role != "member" {
		return httpx.Fail(http.StatusBadRequest, "bad_role", "a teamspace role is owner or member")
	}
	t, err := s.teamspace(ctx, s.st.R, id)
	if err != nil {
		return err
	}
	a, err := s.gate.Workspace(ctx, p, t.WorkspaceID)
	if err != nil {
		return err
	}
	mine := s.teamRole(ctx, id, p.UserID)
	self := user == p.UserID
	switch {
	case a.BreakGlass:
		return httpx.ErrForbidden
	case mine == "owner" || a.IsAdmin():
	case self && t.Access == "open" && role == "member" && mine == "":
	default:
		if t.Access == "private" && mine == "" {
			return policy.ErrHidden
		}
		return httpx.ErrForbidden
	}
	var member int
	if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = ? AND user_id = ?", t.WorkspaceID, user).Scan(&member); err != nil {
		return err
	}
	if member == 0 {
		return httpx.Fail(http.StatusBadRequest, "not_a_member", "only members of the workspace can join a teamspace")
	}
	if prev := s.teamRole(ctx, id, user); prev == "owner" && role != "owner" {
		if err := s.lastOwner(ctx, id); err != nil {
			return err
		}
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO teamspace_members (teamspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)
			ON CONFLICT (teamspace_id, user_id) DO UPDATE SET role = excluded.role`, id, user, role, store.NowMS()); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "teamspace.member", WorkspaceID: t.WorkspaceID, Detail: t.Name + ": " + user + " " + role})
	})
	if err != nil {
		return err
	}
	s.hub.Workspace(t.WorkspaceID, "teamspaces")
	s.hub.User(user, "teamspaces")
	return nil
}

func (s *Service) lastOwner(ctx context.Context, team string) error {
	var owners int
	if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM teamspace_members WHERE teamspace_id = ? AND role = 'owner'", team).Scan(&owners); err != nil {
		return err
	}
	if owners <= 1 {
		return httpx.Fail(http.StatusConflict, "last_owner", "a teamspace needs at least one owner")
	}
	return nil
}

func (s *Service) RemoveTeamMember(ctx context.Context, p *policy.Principal, id, user string) error {
	t, err := s.teamspace(ctx, s.st.R, id)
	if err != nil {
		return err
	}
	a, err := s.gate.Workspace(ctx, p, t.WorkspaceID)
	if err != nil {
		return err
	}
	mine := s.teamRole(ctx, id, p.UserID)
	if user != p.UserID && mine != "owner" && !a.IsAdmin() {
		return httpx.ErrForbidden
	}
	if s.teamRole(ctx, id, user) == "owner" {
		if err := s.lastOwner(ctx, id); err != nil {
			return err
		}
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM teamspace_members WHERE teamspace_id = ? AND user_id = ?", id, user); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "teamspace.leave", WorkspaceID: t.WorkspaceID, Detail: t.Name + ": " + user})
	})
	if err != nil {
		return err
	}
	s.hub.Workspace(t.WorkspaceID, "teamspaces")
	s.hub.User(user, "teamspaces")
	return nil
}

type Group struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	ExternalID  string `json:"external_id,omitempty"`
	Members     int    `json:"members"`
}

func (s *Service) Groups(ctx context.Context, p *policy.Principal, workspaceID string) ([]Group, error) {
	if _, err := s.gate.Workspace(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	return s.groupList(ctx, workspaceID)
}

func (s *Service) groupList(ctx context.Context, workspaceID string) ([]Group, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT g.id, g.name, g.external_id, (SELECT COUNT(*) FROM group_members m WHERE m.group_id = g.id)
		FROM groups g WHERE g.workspace_id = ? ORDER BY g.name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Group{}
	for rows.Next() {
		g := Group{WorkspaceID: workspaceID}
		if err := rows.Scan(&g.ID, &g.Name, &g.ExternalID, &g.Members); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Service) groupAdmin(ctx context.Context, p *policy.Principal, workspaceID string) error {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.IsAdmin() || a.Personal {
		return httpx.ErrForbidden
	}
	return nil
}

func (s *Service) Group(ctx context.Context, id string) (*Group, error) {
	var g Group
	err := s.st.R.QueryRowContext(ctx, `SELECT g.id, g.workspace_id, g.name, g.external_id, (SELECT COUNT(*) FROM group_members m WHERE m.group_id = g.id)
		FROM groups g WHERE g.id = ?`, id).Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.ExternalID, &g.Members)
	if err != nil {
		return nil, store.NoRows(err)
	}
	return &g, nil
}

func (s *Service) CreateGroup(ctx context.Context, p *policy.Principal, workspaceID, name, externalID string) (*Group, error) {
	if p != nil {
		if err := s.groupAdmin(ctx, p, workspaceID); err != nil {
			return nil, err
		}
	}
	n, err := cleanName(name, 100)
	if err != nil {
		return nil, err
	}
	id := ids.New()
	now := store.NowMS()
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO groups (id, workspace_id, name, external_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			id, workspaceID, n, externalID, now, now); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return httpx.Fail(http.StatusConflict, "group_exists", "a group with that name exists")
			}
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "group.create", WorkspaceID: workspaceID, Detail: n})
	})
	if err != nil {
		return nil, err
	}
	s.hub.Workspace(workspaceID, "groups")
	return s.Group(ctx, id)
}

func (s *Service) RenameGroup(ctx context.Context, p *policy.Principal, id, name string) (*Group, error) {
	g, err := s.Group(ctx, id)
	if err != nil {
		return nil, err
	}
	if p != nil {
		if err := s.groupAdmin(ctx, p, g.WorkspaceID); err != nil {
			return nil, err
		}
	}
	n, err := cleanName(name, 100)
	if err != nil {
		return nil, err
	}
	if _, err := s.st.W.ExecContext(ctx, "UPDATE groups SET name = ?, updated_at = ? WHERE id = ?", n, store.NowMS(), id); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, httpx.Fail(http.StatusConflict, "group_exists", "a group with that name exists")
		}
		return nil, err
	}
	s.hub.Workspace(g.WorkspaceID, "groups")
	return s.Group(ctx, id)
}

func (s *Service) DeleteGroup(ctx context.Context, p *policy.Principal, id string) error {
	g, err := s.Group(ctx, id)
	if err != nil {
		return err
	}
	if p != nil {
		if err := s.groupAdmin(ctx, p, g.WorkspaceID); err != nil {
			return err
		}
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM page_shares WHERE subject_kind = 'group' AND subject_id = ?", id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM groups WHERE id = ?", id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "group.delete", WorkspaceID: g.WorkspaceID, Detail: g.Name})
	})
	if err != nil {
		return err
	}
	s.hub.Workspace(g.WorkspaceID, "groups")
	s.hub.Workspace(g.WorkspaceID, "pages")
	return nil
}

func (s *Service) GroupMembers(ctx context.Context, p *policy.Principal, id string) ([]TeamMember, error) {
	g, err := s.Group(ctx, id)
	if err != nil {
		return nil, err
	}
	if p != nil {
		if _, err := s.gate.Workspace(ctx, p, g.WorkspaceID); err != nil {
			return nil, err
		}
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT u.id, u.name, u.email FROM group_members m JOIN users u ON u.id = m.user_id
		WHERE m.group_id = ? ORDER BY u.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TeamMember{}
	for rows.Next() {
		var m TeamMember
		if err := rows.Scan(&m.UserID, &m.Name, &m.Email); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) SetGroupMember(ctx context.Context, p *policy.Principal, id, user string, add bool) error {
	g, err := s.Group(ctx, id)
	if err != nil {
		return err
	}
	if p != nil {
		if err := s.groupAdmin(ctx, p, g.WorkspaceID); err != nil {
			return err
		}
	}
	if add {
		var member int
		if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = ? AND user_id = ?", g.WorkspaceID, user).Scan(&member); err != nil {
			return err
		}
		if member == 0 {
			return httpx.Fail(http.StatusBadRequest, "not_a_member", "only members of the workspace can be in its groups")
		}
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		q := "DELETE FROM group_members WHERE group_id = ? AND user_id = ?"
		if add {
			q = "INSERT INTO group_members (group_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING"
		}
		if _, err := tx.ExecContext(ctx, q, id, user); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE groups SET updated_at = ? WHERE id = ?", store.NowMS(), id); err != nil {
			return err
		}
		action := "group.remove"
		if add {
			action = "group.add"
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: action, WorkspaceID: g.WorkspaceID, Detail: g.Name + ": " + user})
	})
	if err != nil {
		return err
	}
	s.hub.Workspace(g.WorkspaceID, "groups")
	s.hub.User(user, "shares")
	return nil
}
