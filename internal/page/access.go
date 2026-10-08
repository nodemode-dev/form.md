package page

import (
	"context"
	"errors"
	"strings"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Level int8

const (
	LevelNone Level = iota
	LevelView
	LevelComment
	LevelCreate
	LevelContent
	LevelEdit
	LevelFull
)

var levelNames = [...]string{"", "view", "comment", "create", "content", "edit", "full"}

func (l Level) String() string {
	if l < 0 || int(l) >= len(levelNames) {
		return ""
	}
	return levelNames[l]
}

func ParseLevel(s string) (Level, bool) {
	for i, n := range levelNames {
		if i > 0 && n == s {
			return Level(i), true
		}
	}
	return LevelNone, false
}

func roleLevel(role string, breakGlass bool) Level {
	if breakGlass {
		return LevelView
	}
	switch role {
	case "admin":
		return LevelFull
	case "member":
		return LevelEdit
	case "viewer":
		return LevelView
	}
	return LevelNone
}

type teamView struct {
	access string
	member bool
	owner  bool
}

type viewer struct {
	me     string
	ws     policy.Access
	member bool
	teams  map[string]teamView
	shares map[string]Level
}

type chainNode struct {
	id         string
	parent     string
	visibility string
	owner      string
	teamspace  string
}

func (s *Service) viewer(ctx context.Context, q store.Querier, p *policy.Principal, workspaceID string) (*viewer, error) {
	if p == nil || !p.Reaches(workspaceID) {
		return nil, policy.ErrHidden
	}
	v := &viewer{me: p.UserID, teams: map[string]teamView{}, shares: map[string]Level{}}
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	switch {
	case err == nil:
		v.ws, v.member = a, true
	case errors.Is(err, policy.ErrHidden):
		var agentAccess string
		if err := q.QueryRowContext(ctx, "SELECT agent_access FROM workspaces WHERE id = ?", workspaceID).Scan(&agentAccess); err != nil {
			return nil, store.NoRows(err)
		}
		if !policy.AgentAllowed(p, agentAccess) {
			return nil, policy.ErrHidden
		}
		v.ws = policy.Access{WorkspaceID: workspaceID, AgentAccess: agentAccess}
	default:
		return nil, err
	}
	if v.member {
		rows, err := q.QueryContext(ctx, `SELECT t.id, t.access, COALESCE(m.role, '') FROM teamspaces t
			LEFT JOIN teamspace_members m ON m.teamspace_id = t.id AND m.user_id = ? WHERE t.workspace_id = ?`, p.UserID, workspaceID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, access, role string
			if err := rows.Scan(&id, &access, &role); err != nil {
				rows.Close()
				return nil, err
			}
			v.teams[id] = teamView{access: access, member: role != "", owner: role == "owner"}
		}
		rows.Close()
	}
	rows, err := q.QueryContext(ctx, `SELECT page_id, level FROM page_shares
		WHERE workspace_id = ? AND (expires_at = 0 OR expires_at > ?) AND (
			(subject_kind = 'user' AND subject_id = ?) OR
			(subject_kind = 'group' AND subject_id IN (SELECT g.id FROM groups g JOIN group_members gm ON gm.group_id = g.id WHERE g.workspace_id = ? AND gm.user_id = ?)))`,
		workspaceID, store.NowMS(), p.UserID, workspaceID, p.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var page, level string
		if err := rows.Scan(&page, &level); err != nil {
			return nil, err
		}
		if l, ok := ParseLevel(level); ok && l > v.shares[page] {
			v.shares[page] = l
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !v.member && len(v.shares) == 0 {
		return nil, policy.ErrHidden
	}
	return v, nil
}

func (v *viewer) root(n chainNode) Level {
	if !v.member {
		return LevelNone
	}
	base := roleLevel(v.ws.Role, v.ws.BreakGlass)
	if n.teamspace == "" {
		return base
	}
	t, ok := v.teams[n.teamspace]
	if !ok {
		return base
	}
	switch {
	case t.owner && !v.ws.BreakGlass:
		return LevelFull
	case t.member:
		return base
	case t.access == "open":
		return base
	case t.access == "closed" && v.ws.IsAdmin():
		return base
	}
	return LevelNone
}

func (v *viewer) step(inherited Level, n chainNode, isRoot bool) Level {
	l := inherited
	if isRoot {
		l = v.root(n)
	}
	if n.visibility == "private" {
		if n.owner == v.me && v.member {
			l = LevelFull
		} else {
			l = LevelNone
		}
	}
	if g := v.shares[n.id]; g > l {
		l = g
	}
	return l
}

func (v *viewer) chainLevel(chain []chainNode) Level {
	l := LevelNone
	for i, n := range chain {
		l = v.step(l, n, i == 0)
	}
	return l
}

func (v *viewer) levels(all []*Meta) map[string]Level {
	byID := make(map[string]*Meta, len(all))
	for _, m := range all {
		byID[m.ID] = m
	}
	memo := make(map[string]Level, len(all))
	var visit func(m *Meta, depth int) Level
	visit = func(m *Meta, depth int) Level {
		if l, ok := memo[m.ID]; ok {
			return l
		}
		n := chainNode{id: m.ID, parent: m.ParentID, visibility: m.Visibility, owner: m.OwnerID, teamspace: m.TeamspaceID}
		parent, ok := byID[m.ParentID]
		var l Level
		if ok && depth < 512 {
			l = v.step(visit(parent, depth+1), n, false)
		} else {
			l = v.step(LevelNone, n, true)
		}
		memo[m.ID] = l
		return l
	}
	for _, m := range all {
		visit(m, 0)
	}
	return memo
}

func (s *Service) chain(ctx context.Context, q store.Querier, id string) ([]chainNode, error) {
	rows, err := q.QueryContext(ctx, `WITH RECURSIVE chain(id, parent_id, visibility, owner_id, teamspace_id, depth) AS (
			SELECT id, parent_id, visibility, owner_id, teamspace_id, 0 FROM pages WHERE id = ?
			UNION ALL
			SELECT p.id, p.parent_id, p.visibility, p.owner_id, p.teamspace_id, c.depth + 1 FROM pages p JOIN chain c ON p.id = c.parent_id WHERE c.depth < 512
		) SELECT id, COALESCE(parent_id, ''), visibility, COALESCE(owner_id, ''), COALESCE(teamspace_id, '') FROM chain ORDER BY depth DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []chainNode
	for rows.Next() {
		var n chainNode
		if err := rows.Scan(&n.id, &n.parent, &n.visibility, &n.owner, &n.teamspace); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Service) levelOf(ctx context.Context, q store.Querier, p *policy.Principal, m *Meta) (Level, *viewer, []chainNode, error) {
	v, err := s.viewer(ctx, q, p, m.WorkspaceID)
	if err != nil {
		return LevelNone, nil, nil, err
	}
	chain, err := s.chain(ctx, q, m.ID)
	if err != nil {
		return LevelNone, nil, nil, err
	}
	return v.chainLevel(chain), v, chain, nil
}

func (s *Service) Level(ctx context.Context, p *policy.Principal, id string) (Level, string, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return LevelNone, "", err
	}
	l, _, _, err := s.levelOf(ctx, s.st.R, p, m)
	if err != nil {
		return LevelNone, "", err
	}
	if l < LevelView {
		return LevelNone, "", policy.ErrHidden
	}
	return l, m.WorkspaceID, nil
}

func (s *Service) Reach(ctx context.Context, p *policy.Principal, workspaceID string) error {
	_, err := s.viewer(ctx, s.st.R, p, workspaceID)
	return err
}

func (s *Service) CanComment(ctx context.Context, p *policy.Principal, id string) (string, error) {
	return s.require(ctx, p, id, LevelComment)
}

func (s *Service) CanEdit(ctx context.Context, p *policy.Principal, id string) (string, error) {
	return s.require(ctx, p, id, LevelEdit)
}

func (s *Service) CanShare(ctx context.Context, p *policy.Principal, id string) (string, error) {
	return s.require(ctx, p, id, LevelFull)
}

func (s *Service) require(ctx context.Context, p *policy.Principal, id string, need Level) (string, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return "", err
	}
	if _, err := s.check(ctx, s.st.R, p, m, need); err != nil {
		return "", err
	}
	if m.TrashedAt > 0 && need > LevelView {
		return "", errTrashed
	}
	return m.WorkspaceID, nil
}

func (s *Service) Readers(ctx context.Context, m *Meta) (scope string, users []string, err error) {
	chain, err := s.chain(ctx, s.st.R, m.ID)
	if err != nil {
		return "", nil, err
	}
	scope = "workspace"
	if len(chain) > 0 && chain[0].teamspace != "" {
		var access string
		if err := s.st.R.QueryRowContext(ctx, "SELECT access FROM teamspaces WHERE id = ?", chain[0].teamspace).Scan(&access); err == nil && access != "open" {
			scope = "team:" + chain[0].teamspace
		}
	}
	ids := make([]any, 0, len(chain))
	marks := make([]string, 0, len(chain))
	for _, n := range chain {
		if n.visibility == "private" {
			scope = "user:" + n.owner
		}
		ids = append(ids, n.id)
		marks = append(marks, "?")
	}
	if len(ids) == 0 {
		return scope, nil, nil
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT subject_id FROM page_shares WHERE subject_kind = 'user' AND page_id IN (`+strings.Join(marks, ",")+`)
		UNION SELECT gm.user_id FROM page_shares ps JOIN group_members gm ON gm.group_id = ps.subject_id WHERE ps.subject_kind = 'group' AND ps.page_id IN (`+strings.Join(marks, ",")+`)`, append(ids, ids...)...)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u string
		if rows.Scan(&u) == nil {
			users = append(users, u)
		}
	}
	return scope, users, rows.Err()
}
