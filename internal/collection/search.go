package collection

import (
	"context"
	"strings"
)

var searchable = map[string]bool{
	TText: true, TNumber: true, TSelect: true, TMulti: true, TStatus: true, TDate: true,
	TPerson: true, TFiles: true, TURL: true, TEmail: true, TPhone: true, TPlace: true,
}

func (s *Service) OnRows(fn func(ctx context.Context, collectionID string, ids []string)) {
	s.rowWatchers = append(s.rowWatchers, fn)
}

func (s *Service) RowText(ctx context.Context, rowID string) (string, bool) {
	colID, err := s.RowCollection(ctx, rowID)
	if err != nil {
		return "", false
	}
	ev := s.evaluator(ctx, nil)
	defer ev.close()
	ev.readable[colID] = true
	sn := ev.snap(colID)
	if sn == nil {
		return "", false
	}
	r := sn.rows[rowID]
	if r == nil {
		return "", false
	}
	var parts []string
	for _, p := range sn.schema {
		if !searchable[p.Type] {
			continue
		}
		if t := strings.TrimSpace(ev.displayCell(sn, r, p)); t != "" {
			parts = append(parts, p.Name+": "+t)
		}
	}
	return strings.Join(parts, "\n"), true
}

func (s *Service) SchemaOf(ctx context.Context, collectionID string) (Schema, error) {
	return s.loadSchema(ctx, s.st.R, collectionID)
}

func (s *Service) RowIDs(ctx context.Context, collectionID string) []string {
	rows, err := s.st.R.QueryContext(ctx, "SELECT page_id FROM collection_rows WHERE collection_id = ?", collectionID)
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
