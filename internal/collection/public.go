package collection

import (
	"context"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type PublicTable struct {
	ID      string
	Title   string
	View    string
	Columns []string
	Rows    []PublicRow
	Total   int
}

type PublicRow struct {
	ID    string
	Title string
	Icon  string
	Cells []string
}

type PublicProp struct {
	Name  string
	Value string
}

func (s *Service) publicEvaluator(ctx context.Context, collectionID string) *evaluator {
	ev := s.evaluator(ctx, nil)
	ev.p = &policy.Principal{}
	ev.readable[collectionID] = true
	return ev
}

func (s *Service) PublicTable(ctx context.Context, collectionID, viewID string, limit int) (*PublicTable, error) {
	views, err := s.views(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	var v *View
	for i := range views {
		if views[i].ID == viewID || (v == nil && views[i].Type != "form") {
			v = &views[i]
			if views[i].ID == viewID {
				break
			}
		}
	}
	if v == nil {
		return nil, store.ErrNotFound
	}
	var title string
	_ = s.st.R.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = ?", collectionID).Scan(&title)
	ev := s.publicEvaluator(ctx, collectionID)
	defer ev.close()
	sn := ev.snap(collectionID)
	if sn == nil {
		return nil, store.ErrNotFound
	}
	rows := s.results.ordered(ev, sn, v.Config, QueryInput{})
	hidden := map[string]bool{}
	order := []*Property{}
	placed := map[string]bool{}
	for _, vp := range v.Config.Props {
		if vp.Hidden {
			hidden[vp.ID] = true
		}
		if p := sn.schema.ByID(vp.ID); p != nil && !placed[p.ID] {
			order = append(order, p)
			placed[p.ID] = true
		}
	}
	for _, p := range sn.schema {
		if !placed[p.ID] {
			order = append(order, p)
		}
	}
	var props []*Property
	for _, p := range order {
		if p.Type == TTitle {
			props = append([]*Property{p}, props...)
			continue
		}
		if !hidden[p.ID] && p.Type != TButton {
			props = append(props, p)
		}
	}
	out := &PublicTable{ID: collectionID, Title: title, View: v.Name, Total: len(rows)}
	for _, p := range props {
		out.Columns = append(out.Columns, p.Name)
	}
	for i, r := range rows {
		if limit > 0 && i >= limit {
			break
		}
		pr := PublicRow{ID: r.ID, Title: r.Title, Icon: r.Icon, Cells: make([]string, len(props))}
		for j, p := range props {
			if p.Type == TTitle {
				pr.Cells[j] = r.Title
				continue
			}
			pr.Cells[j] = ev.displayCell(sn, r, p)
		}
		out.Rows = append(out.Rows, pr)
	}
	return out, nil
}

func (s *Service) PublicProps(ctx context.Context, rowID string) ([]PublicProp, error) {
	col, err := s.RowCollection(ctx, rowID)
	if err != nil {
		return nil, err
	}
	ev := s.publicEvaluator(ctx, col)
	defer ev.close()
	sn := ev.snap(col)
	if sn == nil {
		return nil, store.ErrNotFound
	}
	r := sn.rows[rowID]
	if r == nil {
		return nil, store.ErrNotFound
	}
	var out []PublicProp
	for _, p := range sn.schema {
		if p.Type == TTitle || p.Type == TButton {
			continue
		}
		if v := ev.displayCell(sn, r, p); v != "" {
			out = append(out, PublicProp{Name: p.Name, Value: v})
		}
	}
	return out, nil
}
