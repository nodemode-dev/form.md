package collection

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type PublicTable struct {
	ID       string
	Title    string
	View     string
	Type     string
	Size     string
	Columns  []string
	GroupCol int
	Rows     []PublicRow
	Groups   []PublicGroup
	Total    int
}

type PublicRow struct {
	ID    string
	Title string
	Icon  string
	Cover string
	Cells []string
}

type PublicGroup struct {
	Label string
	Color string
	Count int
	Rows  []int
}

func publicCover(sn *snap, r *Row, source string) string {
	switch source {
	case "none":
		return ""
	case "cover":
		if strings.HasPrefix(r.Cover, "f:") {
			return r.Cover
		}
		return ""
	case "", "content":
		if strings.HasPrefix(r.Thumb, "f:") {
			return r.Thumb
		}
		if strings.HasPrefix(r.Cover, "f:") {
			return r.Cover
		}
		return ""
	}
	p := sn.schema.ByID(source)
	if p == nil || p.Type != TFiles {
		return ""
	}
	var files []FileVal
	if json.Unmarshal(r.Vals[p.ID], &files) != nil {
		return ""
	}
	for _, f := range files {
		name := strings.ToLower(f.Name)
		image := strings.HasPrefix(f.Mime, "image/") || (f.Mime == "" && (strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".gif") || strings.HasSuffix(name, ".webp")))
		if f.ID != "" && image {
			return "f:" + f.ID
		}
	}
	return ""
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
		if vp.IsHidden() {
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
	out := &PublicTable{ID: collectionID, Title: title, View: v.Name, Type: v.Type, Total: len(rows)}
	for _, p := range props {
		out.Columns = append(out.Columns, p.Name)
	}
	coverSource := "content"
	if c, ok := v.Config.Layout["cover"].(string); ok {
		coverSource = c
	}
	if sz, ok := v.Config.Layout["size"].(string); ok && (sz == "small" || sz == "large") {
		out.Size = sz
	}
	for i, r := range rows {
		if limit > 0 && i >= limit {
			break
		}
		pr := PublicRow{ID: r.ID, Title: r.Title, Icon: r.Icon, Cells: make([]string, len(props))}
		if v.Type == "gallery" || v.Type == "board" {
			pr.Cover = publicCover(sn, r, coverSource)
		}
		for j, p := range props {
			if p.Type == TTitle {
				pr.Cells[j] = r.Title
				continue
			}
			pr.Cells[j] = ev.displayCell(sn, r, p)
		}
		out.Rows = append(out.Rows, pr)
	}
	if g := v.Config.Group; v.Type == "board" && g != nil && sn.schema.ByID(g.Prop) != nil {
		index := make(map[string]int, len(out.Rows))
		for i, r := range out.Rows {
			index[r.ID] = i
		}
		for j, p := range props {
			if p.ID == g.Prop {
				out.GroupCol = j
			}
		}
		hiddenGroups := hiddenSet(g.Hidden)
		for _, b := range ev.bucketise(sn, rows[:len(out.Rows)], g) {
			if hiddenGroups[b.key] || (len(b.rows) == 0 && (g.HideEmpty || b.key == "")) {
				continue
			}
			pg := PublicGroup{Label: b.label, Color: b.color, Count: len(b.rows), Rows: make([]int, 0, len(b.rows))}
			for _, r := range b.rows {
				pg.Rows = append(pg.Rows, index[r.ID])
			}
			out.Groups = append(out.Groups, pg)
		}
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
