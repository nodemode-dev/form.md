package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type cellKey struct{ row, prop string }

type person struct {
	Name  string
	Email string
}

type evaluator struct {
	s        *Service
	ctx      context.Context
	p        *policy.Principal
	loc      *time.Location
	now      time.Time
	snaps    map[string]*snap
	order    []*snap
	memo     map[cellKey]formula.Value
	errs     map[cellKey]string
	visiting map[cellKey]bool
	people   map[string]person
	readable map[string]bool
	rowCol   map[string]string
}

func (s *Service) evaluator(ctx context.Context, p *policy.Principal) *evaluator {
	return &evaluator{
		s: s, ctx: ctx, p: p, loc: s.location(ctx, p), now: time.Now(),
		snaps: map[string]*snap{}, memo: map[cellKey]formula.Value{}, errs: map[cellKey]string{},
		visiting: map[cellKey]bool{}, readable: map[string]bool{}, rowCol: map[string]string{},
	}
}

func (ev *evaluator) close() {
	for _, s := range ev.order {
		s.mu.RUnlock()
	}
	ev.order = nil
	ev.snaps = map[string]*snap{}
}

func (ev *evaluator) use(sn *snap) *snap {
	if have, ok := ev.snaps[sn.id]; ok {
		return have
	}
	sn.mu.RLock()
	ev.snaps[sn.id] = sn
	ev.order = append(ev.order, sn)
	return sn
}

func (ev *evaluator) snap(collectionID string) *snap {
	if sn, ok := ev.snaps[collectionID]; ok {
		return sn
	}
	if collectionID == "" {
		return nil
	}
	ok, seen := ev.readable[collectionID]
	if !seen {
		ok = ev.s.pages.CanRead(ev.ctx, ev.p, collectionID) == nil
		ev.readable[collectionID] = ok
	}
	if !ok {
		return nil
	}
	sn, err := ev.s.cache.get(ev.ctx, ev.s.st.R, collectionID)
	if err != nil {
		return nil
	}
	return ev.use(sn)
}

func (ev *evaluator) person(id string) person {
	if ev.people == nil {
		ev.people = map[string]person{}
		rows, err := ev.s.st.R.QueryContext(ev.ctx, "SELECT id, name, email FROM users")
		if err == nil {
			for rows.Next() {
				var uid string
				var pr person
				if rows.Scan(&uid, &pr.Name, &pr.Email) == nil {
					ev.people[uid] = pr
				}
			}
			rows.Close()
		}
	}
	return ev.people[id]
}

func (ev *evaluator) findRow(id string) (*snap, *Row) {
	for _, sn := range ev.snaps {
		if r, ok := sn.rows[id]; ok {
			return sn, r
		}
	}
	col, ok := ev.rowCol[id]
	if !ok {
		_ = ev.s.st.R.QueryRowContext(ev.ctx, "SELECT collection_id FROM collection_rows WHERE page_id = ?", id).Scan(&col)
		ev.rowCol[id] = col
	}
	if sn := ev.snap(col); sn != nil {
		if r, ok := sn.rows[id]; ok {
			return sn, r
		}
	}
	return nil, nil
}

type rowEnv struct {
	ev  *evaluator
	sn  *snap
	row *Row
}

func (e rowEnv) Prop(name string) (formula.Value, error) {
	p := e.sn.schema.ByName(name)
	if p == nil {
		return formula.Empty, fmt.Errorf("there is no property called %q", name)
	}
	return e.ev.value(e.sn, e.row, p), nil
}

func (e rowEnv) PageProp(pageID, name string) (formula.Value, error) {
	sn, row := e.ev.findRow(pageID)
	if row == nil {
		return formula.Empty, nil
	}
	p := sn.schema.ByName(name)
	if p == nil {
		return formula.Empty, fmt.Errorf("that page has no property called %q", name)
	}
	return e.ev.value(sn, row, p), nil
}

func (e rowEnv) Now() time.Time               { return e.ev.now }
func (e rowEnv) Location() *time.Location     { return e.ev.loc }
func (e rowEnv) RowID() string                { return e.row.ID }
func (e rowEnv) PersonEmail(id string) string { return e.ev.person(id).Email }

func (ev *evaluator) optionNames(p *Property, ids []string) []formula.Value {
	out := make([]formula.Value, 0, len(ids))
	for _, id := range ids {
		if o := p.Option(id); o != nil {
			out = append(out, formula.Text(o.Name))
		}
	}
	return out
}

func (ev *evaluator) value(sn *snap, row *Row, p *Property) formula.Value {
	raw := row.Vals[p.ID]
	switch p.Type {
	case TTitle:
		return formula.Text(row.Title)
	case TText, TURL, TEmail, TPhone:
		return formula.Text(str(raw))
	case TNumber:
		var f float64
		if isNull(raw) || json.Unmarshal(raw, &f) != nil {
			return formula.Empty
		}
		return formula.Num(f)
	case TSelect, TStatus:
		id := str(raw)
		if o := p.Option(id); o != nil {
			return formula.Text(o.Name)
		}
		return formula.Empty
	case TMulti:
		return formula.List(ev.optionNames(p, strList(raw)))
	case TDate:
		var d DateVal
		if isNull(raw) || json.Unmarshal(raw, &d) != nil {
			return formula.Empty
		}
		a, b, has, ok := d.Times(ev.loc)
		if !ok {
			return formula.Empty
		}
		if !b.IsZero() {
			return formula.Range(a, b, has)
		}
		return formula.Date(a, has)
	case TPerson:
		ids := strList(raw)
		out := make([]formula.Value, 0, len(ids))
		for _, id := range ids {
			out = append(out, formula.Person(id, ev.person(id).Name))
		}
		return formula.List(out)
	case TFiles:
		var files []FileVal
		_ = json.Unmarshal(raw, &files)
		out := make([]formula.Value, 0, len(files))
		for _, f := range files {
			out = append(out, formula.Text(f.Name))
		}
		return formula.List(out)
	case TCheckbox:
		var b bool
		_ = json.Unmarshal(raw, &b)
		return formula.Bool(b)
	case TRelation:
		target := ev.snap(p.Collection)
		ids := strList(raw)
		out := make([]formula.Value, 0, len(ids))
		if target == nil {
			return formula.List(out)
		}
		for _, id := range ids {
			if r, ok := target.rows[id]; ok {
				out = append(out, formula.Page(id, r.Title))
			}
		}
		return formula.List(out)
	case TCreatedTime:
		return formula.Date(time.UnixMilli(row.CreatedAt).In(ev.loc), true)
	case TEditedTime:
		return formula.Date(time.UnixMilli(row.UpdatedAt).In(ev.loc), true)
	case TCreatedBy:
		return formula.Person(row.CreatedBy, ev.person(row.CreatedBy).Name)
	case TEditedBy:
		return formula.Person(row.UpdatedBy, ev.person(row.UpdatedBy).Name)
	case TUniqueID:
		return formula.Num(float64(row.Number))
	case TPlace:
		var pl PlaceVal
		if isNull(raw) || json.Unmarshal(raw, &pl) != nil {
			return formula.Empty
		}
		if pl.Name != "" {
			return formula.Text(pl.Name)
		}
		if pl.Address != "" {
			return formula.Text(pl.Address)
		}
		return formula.Text(fmt.Sprintf("%.5f, %.5f", pl.Lat, pl.Lng))
	case TFormula, TRollup:
		k := cellKey{row.ID, p.ID}
		if v, ok := ev.memo[k]; ok {
			return v
		}
		if ev.visiting[k] {
			ev.errs[k] = "this refers back to itself"
			return formula.Empty
		}
		ev.visiting[k] = true
		var v formula.Value
		if p.Type == TFormula {
			v = ev.formula(sn, row, p, k)
		} else {
			v = ev.rollup(sn, row, p, k)
		}
		delete(ev.visiting, k)
		ev.memo[k] = v
		return v
	}
	return formula.Empty
}

func (ev *evaluator) formula(sn *snap, row *Row, p *Property, k cellKey) formula.Value {
	if msg, bad := sn.progErr[p.ID]; bad {
		ev.errs[k] = msg
		return formula.Empty
	}
	prog := sn.programs[p.ID]
	if prog == nil {
		return formula.Empty
	}
	v, err := prog.Eval(rowEnv{ev: ev, sn: sn, row: row})
	if err != nil {
		ev.errs[k] = err.Error()
		return formula.Empty
	}
	return v
}

func (ev *evaluator) rollup(sn *snap, row *Row, p *Property, k cellKey) formula.Value {
	rel := sn.schema.ByID(p.Relation)
	if rel == nil || rel.Type != TRelation {
		ev.errs[k] = "choose a relation to roll up"
		return formula.Empty
	}
	target := ev.snap(rel.Collection)
	if target == nil {
		return formula.Empty
	}
	tp := target.schema.ByID(p.Target)
	if tp == nil {
		tp = target.schema.ByID(TTitle)
	}
	var related []*Row
	for _, id := range strList(row.Vals[rel.ID]) {
		if r, ok := target.rows[id]; ok {
			related = append(related, r)
		}
	}
	vals := make([]formula.Value, len(related))
	for i, r := range related {
		vals[i] = ev.value(target, r, tp)
	}
	return aggregateRollup(p.Fn, vals, len(related))
}

func aggregateRollup(fn string, vals []formula.Value, n int) formula.Value {
	flat := formula.Flatten(vals)
	nonEmpty := 0
	for _, v := range vals {
		if !v.IsEmpty() && !(v.K == formula.KBool && !v.B) {
			nonEmpty++
		}
	}
	pct := func(x int) formula.Value {
		if n == 0 {
			return formula.Num(0)
		}
		return formula.Num(float64(x) / float64(n))
	}
	nums := func() []float64 {
		var out []float64
		for _, v := range flat {
			if v.K == formula.KNumber {
				out = append(out, v.N)
			}
		}
		return out
	}
	dates := func() (time.Time, time.Time, bool) {
		var lo, hi time.Time
		has := false
		for _, v := range flat {
			if v.K != formula.KDate {
				continue
			}
			end := v.End
			if end.IsZero() {
				end = v.T
			}
			if lo.IsZero() || v.T.Before(lo) {
				lo = v.T
			}
			if hi.IsZero() || end.After(hi) {
				hi = end
			}
			has = has || v.HasTime
		}
		return lo, hi, has
	}
	checked := 0
	for _, v := range flat {
		if v.K == formula.KBool && v.B {
			checked++
		}
	}
	switch fn {
	case "show_unique":
		var out []formula.Value
		for _, v := range flat {
			dup := false
			for _, o := range out {
				if formula.Equal(o, v) {
					dup = true
					break
				}
			}
			if !dup && !v.IsEmpty() {
				out = append(out, v)
			}
		}
		return formula.List(out)
	case "count_all":
		return formula.Num(float64(n))
	case "count_values":
		c := 0
		for _, v := range flat {
			if !v.IsEmpty() {
				c++
			}
		}
		return formula.Num(float64(c))
	case "count_unique":
		seen := map[string]bool{}
		for _, v := range flat {
			if !v.IsEmpty() {
				seen[strings.ToLower(v.String())] = true
			}
		}
		return formula.Num(float64(len(seen)))
	case "count_empty":
		return formula.Num(float64(n - nonEmpty))
	case "count_not_empty":
		return formula.Num(float64(nonEmpty))
	case "percent_empty":
		return pct(n - nonEmpty)
	case "percent_not_empty":
		return pct(nonEmpty)
	case "sum":
		s := 0.0
		for _, x := range nums() {
			s += x
		}
		return formula.Num(s)
	case "average":
		if xs := nums(); len(xs) > 0 {
			return formula.Num(formula.Mean(xs))
		}
		return formula.Empty
	case "median":
		if xs := nums(); len(xs) > 0 {
			return formula.Num(formula.Median(xs))
		}
		return formula.Empty
	case "min", "max", "range":
		xs := nums()
		if len(xs) == 0 {
			return formula.Empty
		}
		lo, hi := xs[0], xs[0]
		for _, x := range xs {
			lo, hi = min(lo, x), max(hi, x)
		}
		switch fn {
		case "min":
			return formula.Num(lo)
		case "max":
			return formula.Num(hi)
		}
		return formula.Num(hi - lo)
	case "earliest", "latest", "date_range":
		lo, hi, has := dates()
		if lo.IsZero() {
			return formula.Empty
		}
		switch fn {
		case "earliest":
			return formula.Date(lo, has)
		case "latest":
			return formula.Date(hi, has)
		}
		return formula.Range(lo, hi, has)
	case "checked":
		return formula.Num(float64(checked))
	case "unchecked":
		return formula.Num(float64(len(flat) - checked))
	case "percent_checked":
		if len(flat) == 0 {
			return formula.Num(0)
		}
		return formula.Num(float64(checked) / float64(len(flat)))
	case "percent_unchecked":
		if len(flat) == 0 {
			return formula.Num(0)
		}
		return formula.Num(float64(len(flat)-checked) / float64(len(flat)))
	}
	return formula.List(flat)
}

func (ev *evaluator) encodeCell(sn *snap, row *Row, p *Property) any {
	v := ev.value(sn, row, p)
	if msg, bad := ev.errs[cellKey{row.ID, p.ID}]; bad {
		return map[string]any{"type": "error", "value": msg}
	}
	return encodeValue(v)
}
