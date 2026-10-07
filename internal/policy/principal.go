package policy

import "context"

type Kind string

const (
	KindSession Kind = "session"
	KindToken   Kind = "token"
	KindOAuth   Kind = "oauth"
)

type Principal struct {
	UserID        string
	Email         string
	Name          string
	Role          string
	Kind          Kind
	Scope         string
	AllWorkspaces bool
	Workspaces    map[string]bool
	CredentialID  string
	ClientName    string
	AgentName     string
	AgentRun      string
}

func (p *Principal) IsSession() bool { return p != nil && p.Kind == KindSession }
func (p *Principal) IsAgent() bool   { return p != nil && p.Kind != KindSession }
func (p *Principal) CanWrite() bool  { return p != nil && p.Scope != "read" }
func (p *Principal) IsOwner() bool   { return p != nil && p.Role == "owner" }
func (p *Principal) IsAdmin() bool   { return p != nil && (p.Role == "owner" || p.Role == "admin") }

func (p *Principal) Reaches(workspaceID string) bool {
	if p == nil {
		return false
	}
	if p.Kind == KindSession || p.AllWorkspaces {
		return true
	}
	return p.Workspaces[workspaceID]
}

type ctxKey struct{}

func With(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func From(ctx context.Context) *Principal {
	p, _ := ctx.Value(ctxKey{}).(*Principal)
	return p
}
