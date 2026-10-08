package page

import (
	"context"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

func (s *Service) SetContent(ctx context.Context, id, snippet, thumb, userID string) error {
	res, err := s.st.W.ExecContext(ctx, "UPDATE pages SET snippet = ?, thumb = ?, updated_at = ?, updated_by = CASE WHEN ? = '' THEN updated_by ELSE ? END WHERE id = ?",
		snippet, thumb, store.NowMS(), userID, userID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrNotFound
	}
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return err
	}
	if m.TrashedAt == 0 {
		s.emit(ctx, m)
	}
	return nil
}

func (s *Service) Readable(ctx context.Context, p *policy.Principal, ids []string) ([]*Meta, error) {
	out := []*Meta{}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		m, _, err := s.readable(ctx, p, id)
		if hidden(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if m.TrashedAt == 0 {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Service) Titles(ctx context.Context, p *policy.Principal, ids []string) map[string]string {
	out := map[string]string{}
	list, err := s.Readable(ctx, p, ids)
	if err != nil {
		return out
	}
	for _, m := range list {
		out[m.ID] = m.Title
		if m.Title == "" {
			out[m.ID] = "Untitled"
		}
	}
	return out
}
