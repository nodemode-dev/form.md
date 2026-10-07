package audit

import (
	"context"
	"encoding/json"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Change struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

type Entry struct {
	Action      string
	WorkspaceID string
	PageID      string
	Detail      string
	Changes     []Change
	ActorKind   string
	ActorLabel  string
}

type Log struct {
	st *store.Store
}

func New(st *store.Store) *Log { return &Log{st: st} }

func (l *Log) Record(ctx context.Context, q store.Querier, p *policy.Principal, e Entry) error {
	if q == nil {
		q = l.st.W
	}
	kind, actorID, label, agent, run := e.ActorKind, "", e.ActorLabel, "", ""
	if p != nil {
		actorID, label = p.UserID, p.Name
		if kind == "" {
			kind = "human"
			if p.IsAgent() {
				kind = "agent"
			}
		}
		agent, run = p.AgentName, p.AgentRun
		if agent == "" && p.ClientName != "" {
			agent = p.ClientName
		}
	}
	if kind == "" {
		kind = "system"
	}
	changes := ""
	if len(e.Changes) > 0 {
		b, err := json.Marshal(e.Changes)
		if err != nil {
			return err
		}
		changes = string(b)
	}
	_, err := q.ExecContext(ctx, `INSERT INTO audit_log (at, actor_kind, actor_id, actor_label, agent_name, agent_run, action, workspace_id, page_id, detail, changes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		store.NowMS(), kind, actorID, label, agent, run, e.Action, e.WorkspaceID, e.PageID, e.Detail, changes)
	return err
}
