package policy

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nodemode-ai/form.md/internal/store"
)

type Access struct {
	WorkspaceID string
	Role        string
	BreakGlass  bool
	Personal    bool
	AgentAccess string
}

func (a Access) CanRead() bool  { return a.Role != "" }
func (a Access) CanWrite() bool { return !a.BreakGlass && (a.Role == "admin" || a.Role == "member") }
func (a Access) IsAdmin() bool  { return !a.BreakGlass && a.Role == "admin" }

var ErrHidden = errors.New("policy: not visible to this principal")

type Gate struct {
	st *store.Store
}

func NewGate(st *store.Store) *Gate { return &Gate{st: st} }

func AgentAllowed(p *Principal, agentAccess string) bool {
	if p == nil || p.Kind == KindSession {
		return true
	}
	switch agentAccess {
	case "closed":
		return false
	case "strict":
		return p.Kind == KindOAuth
	}
	return true
}

func (g *Gate) Workspace(ctx context.Context, p *Principal, workspaceID string) (Access, error) {
	if p == nil || workspaceID == "" || !p.Reaches(workspaceID) {
		return Access{}, ErrHidden
	}
	var role sql.NullString
	var personal int
	var agentAccess string
	err := g.st.R.QueryRowContext(ctx, `
		SELECT m.role, w.personal, w.agent_access
		FROM workspaces w
		LEFT JOIN workspace_members m ON m.workspace_id = w.id AND m.user_id = ?
		WHERE w.id = ?`, p.UserID, workspaceID).Scan(&role, &personal, &agentAccess)
	if errors.Is(err, sql.ErrNoRows) {
		return Access{}, ErrHidden
	}
	if err != nil {
		return Access{}, err
	}
	if !AgentAllowed(p, agentAccess) {
		return Access{}, ErrHidden
	}
	a := Access{WorkspaceID: workspaceID, Role: role.String, Personal: personal == 1, AgentAccess: agentAccess}
	if a.Role != "" {
		return a, nil
	}
	if p.IsOwner() && !a.Personal {
		var n int
		if err := g.st.R.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM break_glass
			WHERE user_id = ? AND workspace_id = ? AND revoked_at = 0 AND expires_at > ?`,
			p.UserID, workspaceID, store.NowMS()).Scan(&n); err != nil {
			return Access{}, err
		}
		if n > 0 {
			a.Role, a.BreakGlass = "viewer", true
			return a, nil
		}
	}
	return Access{}, ErrHidden
}

type Membership struct {
	WorkspaceID string
	Role        string
	AgentAccess string
	Personal    bool
}

func (g *Gate) Memberships(ctx context.Context, p *Principal) ([]Membership, int, error) {
	if p == nil {
		return nil, 0, nil
	}
	rows, err := g.st.R.QueryContext(ctx, `
		SELECT m.workspace_id, m.role, w.agent_access, w.personal
		FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.user_id = ?`, p.UserID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Membership
	withheld := 0
	for rows.Next() {
		var m Membership
		var personal int
		if err := rows.Scan(&m.WorkspaceID, &m.Role, &m.AgentAccess, &personal); err != nil {
			return nil, 0, err
		}
		m.Personal = personal == 1
		if !p.Reaches(m.WorkspaceID) || !AgentAllowed(p, m.AgentAccess) {
			withheld++
			continue
		}
		out = append(out, m)
	}
	return out, withheld, rows.Err()
}
