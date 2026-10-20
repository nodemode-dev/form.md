package templates

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/collection/formula"
)

type pvDB struct {
	st     *Starter
	byID   map[string]*prop
	byName map[string]*prop
	titles map[string]int
	progs  map[string]*formula.Program
}

type pvTree struct {
	now  time.Time
	keys map[string]*pvDB
	dbs  map[*Starter]*pvDB
	memo map[string]formula.Value
	busy map[string]bool
}

func newTree(root *Starter, now time.Time) *pvTree {
	t := &pvTree{now: now, keys: map[string]*pvDB{}, dbs: map[*Starter]*pvDB{}, memo: map[string]formula.Value{}, busy: map[string]bool{}}
	var walk func(s *Starter)
	walk = func(s *Starter) {
		if s.Kind == "database" {
			d := t.add(s)
			if s.key != "" {
				t.keys[s.key] = d
			}
		}
		for i := range s.children {
			walk(&s.children[i])
		}
	}
	walk(root)
	return t
}

func (t *pvTree) add(s *Starter) *pvDB {
	d := &pvDB{st: s, byID: map[string]*prop{}, byName: map[string]*prop{}, titles: map[string]int{}, progs: map[string]*formula.Program{}}
	for i := range s.schema {
		p := &s.schema[i]
		d.byID[p.ID] = p
		d.byName[strings.ToLower(p.Name)] = p
	}
	for i, r := range s.rows {
		if _, ok := d.titles[r.Title]; !ok {
			d.titles[r.Title] = i
		}
	}
	for i := range s.schema {
		p := &s.schema[i]
		if p.Type == "formula" && p.Expr != "" {
			if prog, err := formula.Compile(p.Expr, d); err == nil {
				d.progs[p.ID] = prog
			}
		}
	}
	t.dbs[s] = d
	return d
}

func (d *pvDB) HasProp(name string) bool { return d.byName[strings.ToLower(name)] != nil }

func (t *pvTree) db(s *Starter) *pvDB {
	if d, ok := t.dbs[s]; ok {
		return d
	}
	return t.add(s)
}

func (t *pvTree) target(p *prop) *pvDB {
	k, ok := strings.CutPrefix(p.Collection, "@")
	if !ok {
		return nil
	}
	return t.keys[k]
}

type pvEnv struct {
	t  *pvTree
	d  *pvDB
	ri int
}

func (e pvEnv) Prop(name string) (formula.Value, error) {
	p := e.d.byName[strings.ToLower(name)]
	if p == nil {
		return formula.Empty, nil
	}
	return e.t.value(e.d, e.ri, p), nil
}

func (e pvEnv) PageProp(string, string) (formula.Value, error) { return formula.Empty, nil }
func (e pvEnv) Now() time.Time                                 { return e.t.now }
func (e pvEnv) Location() *time.Location                       { return time.UTC }
func (e pvEnv) RowID() string                                  { return strconv.Itoa(e.ri + 1) }
func (e pvEnv) PersonEmail(string) string                      { return "" }

func strs(v any) []string {
	switch x := v.(type) {
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	case []string:
		return x
	case []any:
		var out []string
		for _, it := range x {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func num(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	}
	return 0, false
}

func (t *pvTree) day(s string) (time.Time, bool, bool) {
	if d, ok := dayOffset(s, t.now); ok {
		s = d
	}
	if tm, err := time.Parse("2006-01-02", s); err == nil {
		return tm, false, true
	}
	for _, f := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", time.RFC3339, "2006-01-02 15:04"} {
		if tm, err := time.Parse(f, s); err == nil {
			return tm.UTC(), true, true
		}
	}
	return time.Time{}, false, false
}

func (t *pvTree) dates(v any) (time.Time, time.Time, bool) {
	switch x := v.(type) {
	case string:
		a, _, ok := t.day(x)
		return a, time.Time{}, ok
	case map[string]any:
		s, _ := x["start"].(string)
		a, _, ok := t.day(s)
		if !ok {
			return time.Time{}, time.Time{}, false
		}
		var b time.Time
		if e, _ := x["end"].(string); e != "" {
			if bt, _, ok := t.day(e); ok && !bt.Before(a) {
				b = bt
			}
		}
		return a, b, true
	case map[string]string:
		return t.dates(map[string]any{"start": x["start"], "end": x["end"]})
	}
	return time.Time{}, time.Time{}, false
}

func (t *pvTree) value(d *pvDB, ri int, p *prop) formula.Value {
	r := &d.st.rows[ri]
	raw := r.Values[p.ID]
	switch p.Type {
	case "title":
		return formula.Text(r.Title)
	case "text", "url", "email", "phone":
		s, _ := raw.(string)
		if s == "" {
			return formula.Empty
		}
		return formula.Text(s)
	case "number":
		if n, ok := num(raw); ok {
			return formula.Num(n)
		}
		return formula.Empty
	case "select", "status":
		for _, id := range strs(raw) {
			if o, ok := findOpt(*p, id); ok {
				return formula.Text(o.Name)
			}
		}
		return formula.Empty
	case "multi_select":
		var out []formula.Value
		for _, id := range strs(raw) {
			if o, ok := findOpt(*p, id); ok {
				out = append(out, formula.Text(o.Name))
			}
		}
		return formula.List(out)
	case "date":
		a, b, ok := t.dates(raw)
		if !ok {
			return formula.Empty
		}
		if !b.IsZero() {
			return formula.Range(a, b, false)
		}
		return formula.Date(a, false)
	case "person", "created_by", "last_edited_by":
		if p.Type != "person" || raw != nil {
			return formula.List([]formula.Value{formula.Person("me", "You")})
		}
		return formula.List(nil)
	case "checkbox":
		b, _ := raw.(bool)
		return formula.Bool(b)
	case "relation":
		var out []formula.Value
		for _, title := range strs(raw) {
			out = append(out, formula.Page(title, title))
		}
		return formula.List(out)
	case "created_time", "last_edited_time":
		return formula.Date(t.now, false)
	case "unique_id":
		return formula.Num(float64(ri + 1))
	case "place":
		if m, ok := raw.(map[string]any); ok {
			if n, _ := m["name"].(string); n != "" {
				return formula.Text(n)
			}
		}
		return formula.Empty
	case "formula", "rollup":
		k := fmt.Sprintf("%p|%d|%s", d, ri, p.ID)
		if v, ok := t.memo[k]; ok {
			return v
		}
		if t.busy[k] {
			return formula.Empty
		}
		t.busy[k] = true
		v := formula.Empty
		if p.Type == "formula" {
			if prog := d.progs[p.ID]; prog != nil {
				if out, err := prog.Eval(pvEnv{t, d, ri}); err == nil {
					v = out
				}
			}
		} else {
			v = t.rollup(d, ri, p)
		}
		delete(t.busy, k)
		t.memo[k] = v
		return v
	}
	if s, ok := raw.(string); ok && s != "" {
		return formula.Text(s)
	}
	return formula.Empty
}

func (t *pvTree) rollup(d *pvDB, ri int, p *prop) formula.Value {
	rel := d.byID[p.Relation]
	if rel == nil || rel.Type != "relation" {
		return formula.Empty
	}
	td := t.target(rel)
	if td == nil {
		return formula.Empty
	}
	tp := td.byID[p.Target]
	if tp == nil {
		tp = td.byID["title"]
	}
	if tp == nil {
		return formula.Empty
	}
	var vals []formula.Value
	for _, title := range strs(d.st.rows[ri].Values[rel.ID]) {
		if j, ok := td.titles[title]; ok {
			vals = append(vals, t.value(td, j, tp))
		}
	}
	return aggregate(p.Fn, vals, len(vals))
}

func aggregate(fn string, vals []formula.Value, n int) formula.Value {
	flat := formula.Flatten(vals)
	var nums []float64
	nonEmpty, checked := 0, 0
	for _, v := range vals {
		if !v.IsEmpty() && !(v.K == formula.KBool && !v.B) {
			nonEmpty++
		}
	}
	for _, v := range flat {
		if v.K == formula.KNumber {
			nums = append(nums, v.N)
		}
		if v.K == formula.KBool && v.B {
			checked++
		}
	}
	frac := func(x int) formula.Value {
		if n == 0 {
			return formula.Num(0)
		}
		return formula.Num(float64(x) / float64(n))
	}
	switch fn {
	case "count", "count_all":
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
		return frac(n - nonEmpty)
	case "percent_not_empty":
		return frac(nonEmpty)
	case "checked":
		return formula.Num(float64(checked))
	case "unchecked":
		return formula.Num(float64(n - checked))
	case "percent_checked":
		return frac(checked)
	case "percent_unchecked":
		return frac(n - checked)
	}
	if len(nums) == 0 {
		if fn == "sum" {
			return formula.Num(0)
		}
		if fn == "" || fn == "show_original" {
			return formula.List(vals)
		}
		return formula.Empty
	}
	sort.Float64s(nums)
	sum := 0.0
	for _, x := range nums {
		sum += x
	}
	switch fn {
	case "sum":
		return formula.Num(sum)
	case "average":
		return formula.Num(sum / float64(len(nums)))
	case "median":
		return formula.Num(formula.Median(nums))
	case "min":
		return formula.Num(nums[0])
	case "max":
		return formula.Num(nums[len(nums)-1])
	case "range":
		return formula.Num(nums[len(nums)-1] - nums[0])
	}
	return formula.List(vals)
}

func (t *pvTree) options(d *pvDB, ri int, p *prop) []string {
	raw := d.st.rows[ri].Values[p.ID]
	switch p.Type {
	case "select", "status", "multi_select":
		var out []string
		for _, id := range strs(raw) {
			if _, ok := findOpt(*p, id); ok {
				out = append(out, id)
			}
		}
		return out
	case "relation":
		return strs(raw)
	case "person":
		if raw != nil {
			return []string{"me"}
		}
	}
	return nil
}

type pvFV struct {
	S    string
	N    float64
	num  bool
	B    bool
	List []string
}

func filterValue(raw json.RawMessage) pvFV {
	var out pvFV
	var val any
	if len(raw) == 0 || json.Unmarshal(raw, &val) != nil {
		return out
	}
	switch x := val.(type) {
	case string:
		out.S = x
		out.List = []string{x}
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			out.N, out.num = f, true
		}
	case float64:
		out.N, out.num = x, true
		out.S = strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		out.B = x
	case []any:
		out.List = strs(x)
	case map[string]any:
		if v, ok := x["value"].(string); ok {
			out.S = v
		}
	}
	return out
}

func (t *pvTree) match(d *pvDB, ri int, f *collection.Filter, depth int) bool {
	if f == nil || depth > 4 {
		return true
	}
	if f.Prop == "" {
		if len(f.Items) == 0 {
			return true
		}
		or := f.Op == "or"
		for i := range f.Items {
			ok := t.match(d, ri, &f.Items[i], depth+1)
			if or && ok {
				return true
			}
			if !or && !ok {
				return false
			}
		}
		return !or
	}
	p := d.byID[f.Prop]
	if p == nil {
		return true
	}
	v := t.value(d, ri, p)
	fv := filterValue(f.Value)
	switch f.Cond {
	case "empty", "is_empty":
		if p.Type == "checkbox" {
			return !v.Truthy()
		}
		return v.IsEmpty()
	case "not_empty", "is_not_empty":
		if p.Type == "checkbox" {
			return v.Truthy()
		}
		return !v.IsEmpty()
	}
	switch p.Type {
	case "select", "status":
		have := t.options(d, ri, p)
		id := ""
		if len(have) > 0 {
			id = have[0]
		}
		in := false
		for _, w := range fv.List {
			if w == id {
				in = true
			}
			if f.Cond == "group_is" {
				if o, ok := findOpt(*p, id); ok && o.Group == w {
					in = true
				}
			}
		}
		if f.Cond == "is_not" || f.Cond == "none" {
			return !in
		}
		return in
	case "multi_select", "relation", "person":
		have := t.options(d, ri, p)
		hit := false
		for _, w := range fv.List {
			for _, h := range have {
				if h == w {
					hit = true
				}
			}
		}
		if f.Cond == "not_contains" || f.Cond == "is_not" || f.Cond == "none" {
			return !hit
		}
		return hit
	case "checkbox":
		want := fv.B || fv.S == "true"
		if f.Cond == "is_not" {
			return v.Truthy() != want
		}
		return v.Truthy() == want
	}
	if v.K == formula.KNumber && fv.num {
		switch f.Cond {
		case "eq", "is":
			return v.N == fv.N
		case "ne", "is_not":
			return v.N != fv.N
		case "gt":
			return v.N > fv.N
		case "lt":
			return v.N < fv.N
		case "gte":
			return v.N >= fv.N
		case "lte":
			return v.N <= fv.N
		}
		return true
	}
	if v.K == formula.KDate {
		if w, _, ok := t.day(fv.S); ok {
			switch f.Cond {
			case "before":
				return v.T.Before(w)
			case "after":
				return v.T.After(w)
			case "on_or_before":
				return !v.T.After(w)
			case "on_or_after":
				return !v.T.Before(w)
			case "is":
				return v.T.Equal(w)
			}
		}
		return true
	}
	hay := strings.ToLower(v.String())
	needle := strings.ToLower(fv.S)
	switch f.Cond {
	case "is":
		return hay == needle
	case "is_not":
		return hay != needle
	case "contains":
		return strings.Contains(hay, needle)
	case "not_contains":
		return !strings.Contains(hay, needle)
	case "starts_with":
		return strings.HasPrefix(hay, needle)
	case "ends_with":
		return strings.HasSuffix(hay, needle)
	}
	return true
}

func (t *pvTree) rowsOf(d *pvDB, f *collection.Filter) []int {
	var out []int
	for i := range d.st.rows {
		if t.match(d, i, f, 0) {
			out = append(out, i)
		}
	}
	return out
}

type pvKey struct {
	key, label, color string
	rank              float64
}

type pvBucket struct {
	pvKey
	rows []int
}

func statusLabel(g string) string {
	switch g {
	case "todo":
		return "To do"
	case "doing":
		return "In progress"
	}
	return "Complete"
}

func weekStartOf(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

func (t *pvTree) keysOf(d *pvDB, ri int, g *collection.Grouping) []pvKey {
	p := d.byID[g.Prop]
	if p == nil {
		return []pvKey{{}}
	}
	none := pvKey{key: "", label: "No " + p.Name, rank: math.Inf(1)}
	switch p.Type {
	case "select", "status", "multi_select":
		ids := t.options(d, ri, p)
		if p.Type == "status" && g.By == "group" {
			if len(ids) == 0 {
				return []pvKey{none}
			}
			o, _ := findOpt(*p, ids[0])
			for i, gname := range []string{"todo", "doing", "done"} {
				if gname == o.Group {
					return []pvKey{{key: gname, label: statusLabel(gname), rank: float64(i)}}
				}
			}
			return []pvKey{none}
		}
		var out []pvKey
		for _, id := range ids {
			for i, o := range p.Options {
				if o.ID == id {
					out = append(out, pvKey{key: id, label: o.Name, color: o.Color, rank: float64(i)})
				}
			}
		}
		if len(out) == 0 {
			return []pvKey{none}
		}
		return out
	case "person", "relation", "created_by", "last_edited_by":
		ids := t.options(d, ri, p)
		if p.Type != "person" && p.Type != "relation" {
			ids = []string{"me"}
		}
		var out []pvKey
		for _, id := range ids {
			label := id
			if id == "me" && p.Type != "relation" {
				label = "You"
			}
			out = append(out, pvKey{key: id, label: label, rank: math.NaN()})
		}
		if len(out) == 0 {
			return []pvKey{none}
		}
		return out
	case "checkbox":
		if t.value(d, ri, p).Truthy() {
			return []pvKey{{key: "true", label: "Checked", rank: 0}}
		}
		return []pvKey{{key: "false", label: "Unchecked", rank: 1}}
	}
	v := t.value(d, ri, p)
	if v.IsEmpty() {
		return []pvKey{none}
	}
	switch v.K {
	case formula.KDate:
		tm := v.T
		switch g.By {
		case "year":
			return []pvKey{{key: tm.Format("2006"), label: tm.Format("2006"), rank: float64(tm.Year())}}
		case "month":
			s := time.Date(tm.Year(), tm.Month(), 1, 0, 0, 0, 0, time.UTC)
			return []pvKey{{key: s.Format("2006-01"), label: s.Format("Jan 2006"), rank: float64(s.Unix())}}
		case "weekday":
			wd := int(tm.Weekday())
			return []pvKey{{key: "d" + strconv.Itoa(wd), label: tm.Weekday().String()[:3], rank: float64((wd + 6) % 7)}}
		case "hour":
			return []pvKey{{key: fmt.Sprintf("h%02d", tm.Hour()), label: fmt.Sprintf("%02d:00", tm.Hour()), rank: float64(tm.Hour())}}
		case "week":
			s := weekStartOf(tm)
			return []pvKey{{key: s.Format("2006-01-02") + "w", label: s.Format("Jan 2"), rank: float64(s.Unix())}}
		case "relative":
			today := time.Date(t.now.Year(), t.now.Month(), t.now.Day(), 0, 0, 0, 0, time.UTC)
			dd := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, time.UTC)
			days := int(math.Round(dd.Sub(today).Hours() / 24))
			switch {
			case days == 0:
				return []pvKey{{key: "r0", label: "Today", rank: 0}}
			case days == 1:
				return []pvKey{{key: "r1", label: "Tomorrow", rank: 1}}
			case days == -1:
				return []pvKey{{key: "r-1", label: "Yesterday", rank: -1}}
			case days > 1 && days <= 7:
				return []pvKey{{key: "r7", label: "Next 7 days", rank: 7}}
			case days < -1 && days >= -7:
				return []pvKey{{key: "r-7", label: "Last 7 days", rank: -7}}
			case days > 7 && days <= 30:
				return []pvKey{{key: "r30", label: "Next 30 days", rank: 30}}
			case days < -7 && days >= -30:
				return []pvKey{{key: "r-30", label: "Last 30 days", rank: -30}}
			case days > 30:
				return []pvKey{{key: "rf", label: "Later", rank: 1e6}}
			}
			return []pvKey{{key: "rp", label: "Older", rank: -1e6}}
		}
		s := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, time.UTC)
		return []pvKey{{key: s.Format("2006-01-02"), label: s.Format("Jan 2"), rank: float64(s.Unix())}}
	case formula.KNumber:
		step := g.Step
		if step <= 0 {
			step = 10
		}
		lo := math.Floor(v.N/step) * step
		return []pvKey{{key: formula.FormatNumber(lo), label: formula.FormatNumber(lo) + " to " + formula.FormatNumber(lo+step), rank: lo}}
	case formula.KBool:
		if v.B {
			return []pvKey{{key: "true", label: "True", rank: 0}}
		}
		return []pvKey{{key: "false", label: "False", rank: 1}}
	case formula.KList:
		var out []pvKey
		for _, it := range v.L {
			s := it.String()
			if s != "" {
				out = append(out, pvKey{key: strings.ToLower(s), label: s, rank: math.NaN()})
			}
		}
		if len(out) == 0 {
			return []pvKey{none}
		}
		return out
	}
	s := v.String()
	return []pvKey{{key: strings.ToLower(s), label: s, rank: math.NaN()}}
}

func (t *pvTree) buckets(d *pvDB, rows []int, g *collection.Grouping) []*pvBucket {
	byKey := map[string]*pvBucket{}
	var list []*pvBucket
	p := d.byID[g.Prop]
	add := func(k pvKey) *pvBucket {
		b := &pvBucket{pvKey: k}
		byKey[k.key] = b
		list = append(list, b)
		return b
	}
	if p != nil && (p.Type == "select" || p.Type == "status" || p.Type == "multi_select") && g.By != "group" {
		for i, o := range p.Options {
			add(pvKey{key: o.ID, label: o.Name, color: o.Color, rank: float64(i)})
		}
	}
	if p != nil && p.Type == "status" && g.By == "group" {
		for i, gname := range []string{"todo", "doing", "done"} {
			add(pvKey{key: gname, label: statusLabel(gname), rank: float64(i)})
		}
	}
	if p != nil && p.Type == "checkbox" {
		add(pvKey{key: "true", label: "Checked", rank: 0})
		add(pvKey{key: "false", label: "Unchecked", rank: 1})
	}
	for _, ri := range rows {
		seen := map[string]bool{}
		for _, k := range t.keysOf(d, ri, g) {
			if seen[k.key] {
				continue
			}
			seen[k.key] = true
			b, ok := byKey[k.key]
			if !ok {
				b = add(k)
			}
			b.rows = append(b.rows, ri)
		}
	}
	desc := g.Sort == "desc"
	less := func(a, b *pvBucket) bool {
		if math.IsNaN(a.rank) || math.IsNaN(b.rank) {
			return strings.ToLower(a.label) < strings.ToLower(b.label)
		}
		return a.rank < b.rank
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.key == "" || b.key == "" {
			return b.key == "" && a.key != ""
		}
		if desc {
			return less(b, a)
		}
		return less(a, b)
	})
	if len(g.Order) > 0 && g.Sort == "manual" {
		pos := map[string]int{}
		for i, k := range g.Order {
			pos[k] = i
		}
		sort.SliceStable(list, func(i, j int) bool {
			pi, oki := pos[list[i].key]
			pj, okj := pos[list[j].key]
			if oki && okj {
				return pi < pj
			}
			return oki
		})
	}
	return list
}

func (t *pvTree) measure(d *pvDB, rows []int, fn, propID string) float64 {
	if fn == "" || fn == "count" || fn == "count_all" {
		return float64(len(rows))
	}
	p := d.byID[propID]
	if p == nil {
		return float64(len(rows))
	}
	boolish := strings.Contains(fn, "checked")
	vals := make([]formula.Value, len(rows))
	for i, ri := range rows {
		v := t.value(d, ri, p)
		if boolish || (p.Type == "checkbox" && !v.Truthy()) {
			v = formula.Bool(v.Truthy())
		}
		vals[i] = v
	}
	out := aggregate(fn, vals, len(rows))
	if out.K != formula.KNumber {
		return 0
	}
	return out.N
}

type pvPoint struct{ key, label, color string }

type pvSeries struct {
	key, label, color string
	vals              []float64
}

type pvDot struct {
	title, group string
	v            float64
}

type pvResult struct {
	x      []pvPoint
	series []pvSeries
	value  float64
	dots   []pvDot
}

func single(kind string) bool {
	return kind == "donut" || kind == "radial" || kind == "histogram" || kind == "calendar" || kind == "compare"
}

func hiddenKeys(list []string) map[string]bool {
	m := map[string]bool{}
	for _, k := range list {
		m[k] = true
	}
	return m
}

func (t *pvTree) chart(d *pvDB, spec collection.ChartSpec, view *collection.Filter) *pvResult {
	f := view
	if spec.Filter != nil {
		if f == nil {
			f = spec.Filter
		} else {
			f = &collection.Filter{Op: "and", Items: []collection.Filter{*f, *spec.Filter}}
		}
	}
	all := t.rowsOf(d, f)
	res := &pvResult{}
	if spec.Kind == "dots" {
		group := map[int]string{}
		if spec.X != nil && d.byID[spec.X.Prop] != nil {
			hidden := hiddenKeys(spec.X.Hidden)
			for _, b := range t.buckets(d, all, spec.X) {
				if hidden[b.key] || len(b.rows) == 0 {
					continue
				}
				res.x = append(res.x, pvPoint{b.key, b.label, b.color})
				for _, ri := range b.rows {
					if _, ok := group[ri]; !ok {
						group[ri] = b.key
					}
				}
			}
		}
		vp := d.byID[spec.Prop]
		if vp == nil {
			return res
		}
		for _, ri := range all {
			v := t.value(d, ri, vp)
			if v.K != formula.KNumber {
				continue
			}
			g, ok := group[ri]
			if len(res.x) > 0 && !ok {
				continue
			}
			res.dots = append(res.dots, pvDot{d.st.rows[ri].Title, g, v.N})
		}
		return res
	}
	if spec.Kind == "number" || spec.Kind == "gauge" {
		res.value = t.measure(d, all, spec.Fn, spec.Prop)
		return res
	}
	if spec.X == nil || d.byID[spec.X.Prop] == nil {
		return res
	}
	hidden := hiddenKeys(spec.X.Hidden)
	var xs []*pvBucket
	for _, b := range t.buckets(d, all, spec.X) {
		if hidden[b.key] || (len(b.rows) == 0 && (b.key == "" || spec.HideEmpty || spec.X.HideEmpty)) {
			continue
		}
		xs = append(xs, b)
	}
	type lane struct {
		pvSeries
		member map[int]bool
	}
	var lanes []*lane
	if spec.Series != nil && !single(spec.Kind) && d.byID[spec.Series.Prop] != nil {
		sh := hiddenKeys(spec.Series.Hidden)
		for _, b := range t.buckets(d, all, spec.Series) {
			if sh[b.key] || len(b.rows) == 0 {
				continue
			}
			l := &lane{pvSeries: pvSeries{key: b.key, label: b.label, color: b.color}, member: map[int]bool{}}
			for _, ri := range b.rows {
				l.member[ri] = true
			}
			lanes = append(lanes, l)
			if len(lanes) == 12 {
				break
			}
		}
	}
	if len(lanes) == 0 {
		lanes = []*lane{{}}
	}
	totals := make([]float64, len(xs))
	for _, l := range lanes {
		l.vals = make([]float64, len(xs))
		for i, b := range xs {
			rows := b.rows
			if l.member != nil {
				rows = nil
				for _, ri := range b.rows {
					if l.member[ri] {
						rows = append(rows, ri)
					}
				}
			}
			l.vals[i] = t.measure(d, rows, spec.Fn, spec.Prop)
			totals[i] += l.vals[i]
		}
	}
	order := make([]int, len(xs))
	for i := range order {
		order[i] = i
	}
	switch spec.Sort {
	case "value_desc":
		sort.SliceStable(order, func(a, b int) bool { return totals[order[a]] > totals[order[b]] })
	case "value_asc":
		sort.SliceStable(order, func(a, b int) bool { return totals[order[a]] < totals[order[b]] })
	}
	limit := spec.Limit
	if limit <= 0 {
		limit = 100
		if spec.Kind == "calendar" {
			limit = 1100
		}
	}
	if len(order) > limit {
		order = order[:limit]
	}
	for _, i := range order {
		b := xs[i]
		res.x = append(res.x, pvPoint{b.key, b.label, b.color})
	}
	for _, l := range lanes {
		vals := make([]float64, len(order))
		run := 0.0
		for j, i := range order {
			vals[j] = l.vals[i]
			if spec.Cumulative && (spec.Kind == "line" || spec.Kind == "area") {
				run += vals[j]
				vals[j] = run
			}
		}
		res.series = append(res.series, pvSeries{key: l.key, label: l.label, color: l.color, vals: vals})
	}
	return res
}
