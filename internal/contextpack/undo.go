package contextpack

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/nodemode-ai/form.md/internal/policy"
)

const UndoKind = "knowledge"

type Undo struct {
	Delete  string `json:"delete,omitempty"`
	Restore *Item  `json:"restore,omitempty"`
}

func (it *Item) Snapshot() *Item {
	c := *it
	c.bag = nil
	return &c
}

func (s *Service) Undo(ctx context.Context, p *policy.Principal, _ string, raw json.RawMessage) error {
	var u Undo
	if err := json.Unmarshal(raw, &u); err != nil {
		return err
	}
	id := u.Delete
	if u.Restore != nil {
		id = u.Restore.ID
	}
	var ws string
	err := s.Store.R.QueryRowContext(ctx, "SELECT workspace_id FROM knowledge WHERE id = ?", id).Scan(&ws)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.reach(ctx, p, ws, true); err != nil {
		return err
	}
	if u.Delete != "" {
		_, err = s.Store.W.ExecContext(ctx, "DELETE FROM knowledge WHERE id = ?", u.Delete)
		return err
	}
	r := u.Restore
	_, err = s.Store.W.ExecContext(ctx, "UPDATE knowledge SET confidence = ?, confirmations = ?, contradictions = ?, confirmed_at = ?, retired_at = ?, superseded_by = ? WHERE id = ?",
		r.Confidence, r.Confirmations, r.Contradictions, r.ConfirmedAt, r.RetiredAt, r.SupersededBy, r.ID)
	return err
}
