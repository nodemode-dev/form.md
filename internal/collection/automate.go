package collection

import (
	"context"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type RowView struct {
	ID           string            `json:"id"`
	CollectionID string            `json:"collection_id"`
	Title        string            `json:"title"`
	Values       map[string]any    `json:"values"`
	Text         map[string]string `json:"text"`
	Schema       Schema            `json:"-"`
}

func (s *Service) RowMatches(ctx context.Context, p *policy.Principal, rowID string, f *Filter) (bool, error) {
	colID, err := s.RowCollection(ctx, rowID)
	if err != nil || colID == "" {
		return false, store.ErrNotFound
	}
	if _, _, err := s.readable(ctx, p, colID); err != nil {
		return false, err
	}
	if f == nil {
		return true, nil
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn, r := ev.findRow(rowID)
	if r == nil {
		return false, store.ErrNotFound
	}
	return ev.match(sn, r, f, 0), nil
}

func (s *Service) RowsMatching(ctx context.Context, p *policy.Principal, collectionID string, f *Filter, limit int) ([]string, error) {
	if _, _, err := s.readable(ctx, p, collectionID); err != nil {
		return nil, err
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn := ev.snap(collectionID)
	if sn == nil {
		return nil, store.ErrNotFound
	}
	out := []string{}
	for _, r := range s.results.ordered(ev, sn, ViewConfig{Filter: f}, QueryInput{}) {
		out = append(out, r.ID)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Service) ReadRow(ctx context.Context, p *policy.Principal, rowID string) (*RowView, error) {
	colID, err := s.RowCollection(ctx, rowID)
	if err != nil || colID == "" {
		return nil, store.ErrNotFound
	}
	if _, _, err := s.readable(ctx, p, colID); err != nil {
		return nil, err
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn, r := ev.findRow(rowID)
	if r == nil {
		return nil, store.ErrNotFound
	}
	out := ev.rowOut(sn, r, map[string]Ref{})
	v := &RowView{ID: rowID, CollectionID: colID, Title: r.Title, Values: out.Values, Text: map[string]string{}, Schema: sn.schema.Clone()}
	for _, prop := range sn.schema {
		val := ev.value(sn, r, prop)
		if !val.IsEmpty() {
			v.Text[prop.ID] = val.String()
		}
	}
	return v, nil
}
