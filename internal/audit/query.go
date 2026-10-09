package audit

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/nodemode-ai/form.md/internal/store"
)

type Row struct {
	ID          int64           `json:"id"`
	At          int64           `json:"at"`
	ActorKind   string          `json:"actor_kind"`
	ActorID     string          `json:"actor_id"`
	ActorLabel  string          `json:"actor_label"`
	Agent       string          `json:"agent"`
	Action      string          `json:"action"`
	WorkspaceID string          `json:"workspace_id"`
	PageID      string          `json:"page_id"`
	PageTitle   string          `json:"page_title"`
	Detail      string          `json:"detail"`
	Changes     json.RawMessage `json:"changes,omitempty"`
}

type Filter struct {
	WorkspaceID string
	Actor       string
	Action      string
	PageID      string
	From        int64
	To          int64
	Before      int64
	Limit       int
}

func (f Filter) where() (string, []any) {
	conds := []string{"a.workspace_id = ?"}
	args := []any{f.WorkspaceID}
	if f.Actor != "" {
		conds = append(conds, "a.actor_id = ?")
		args = append(args, f.Actor)
	}
	switch {
	case strings.HasSuffix(f.Action, "."):
		conds = append(conds, "a.action LIKE ? ESCAPE '\\'")
		args = append(args, strings.ReplaceAll(strings.ReplaceAll(f.Action, "_", "\\_"), "%", "\\%")+"%")
	case f.Action != "":
		conds = append(conds, "a.action = ?")
		args = append(args, f.Action)
	}
	if f.PageID != "" {
		conds = append(conds, "a.page_id = ?")
		args = append(args, f.PageID)
	}
	if f.From > 0 {
		conds = append(conds, "a.at >= ?")
		args = append(args, f.From)
	}
	if f.To > 0 {
		conds = append(conds, "a.at < ?")
		args = append(args, f.To)
	}
	if f.Before > 0 {
		conds = append(conds, "a.id < ?")
		args = append(args, f.Before)
	}
	return strings.Join(conds, " AND "), args
}

func (l *Log) Query(ctx context.Context, f Filter) ([]Row, int64, error) {
	if f.Limit <= 0 || f.Limit > 100000 {
		f.Limit = 50
	}
	where, args := f.where()
	rows, err := l.st.R.QueryContext(ctx, `SELECT a.id, a.at, a.actor_kind, a.actor_id, a.actor_label, a.agent_name, a.action, a.workspace_id, a.page_id, COALESCE(p.title, ''), a.detail, a.changes
		FROM audit_log a LEFT JOIN pages p ON p.id = a.page_id WHERE `+where+` ORDER BY a.id DESC LIMIT ?`, append(args, f.Limit+1)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Row{}
	for rows.Next() {
		var r Row
		var changes string
		if err := rows.Scan(&r.ID, &r.At, &r.ActorKind, &r.ActorID, &r.ActorLabel, &r.Agent, &r.Action, &r.WorkspaceID, &r.PageID, &r.PageTitle, &r.Detail, &changes); err != nil {
			return nil, 0, err
		}
		if changes != "" {
			r.Changes = json.RawMessage(changes)
		}
		out = append(out, r)
	}
	var next int64
	if len(out) > f.Limit {
		out = out[:f.Limit]
		next = out[len(out)-1].ID
	}
	return out, next, rows.Err()
}

type Actor struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (l *Log) Facets(ctx context.Context, workspaceID string) ([]string, []Actor, error) {
	actions := []string{}
	rows, err := l.st.R.QueryContext(ctx, "SELECT DISTINCT action FROM audit_log WHERE workspace_id = ? ORDER BY action", workspaceID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var a string
		if rows.Scan(&a) == nil {
			actions = append(actions, a)
		}
	}
	rows.Close()
	actors := []Actor{}
	rows, err = l.st.R.QueryContext(ctx, `SELECT a.actor_id, COALESCE(u.name, MAX(a.actor_label)) FROM audit_log a LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.workspace_id = ? AND a.actor_id != '' GROUP BY a.actor_id ORDER BY 2`, workspaceID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Actor
		if rows.Scan(&a.ID, &a.Label) == nil {
			actors = append(actors, a)
		}
	}
	return actions, actors, rows.Err()
}

func (l *Log) Purge(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		return 0, nil
	}
	res, err := l.st.W.ExecContext(ctx, "DELETE FROM audit_log WHERE at < ?", store.NowMS()-int64(days)*86400000)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
