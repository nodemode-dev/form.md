package collection

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection/formula"
)

type Filter struct {
	Op    string          `json:"op,omitempty"`
	Items []Filter        `json:"items,omitempty"`
	Prop  string          `json:"prop,omitempty"`
	Cond  string          `json:"cond,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

type SortKey struct {
	Prop string `json:"prop"`
	Dir  string `json:"dir"`
}

type Grouping struct {
	Prop      string   `json:"prop"`
	By        string   `json:"by,omitempty"`
	Step      float64  `json:"step,omitempty"`
	Sort      string   `json:"sort,omitempty"`
	Order     []string `json:"order,omitempty"`
	Hidden    []string `json:"hidden,omitempty"`
	HideEmpty bool     `json:"hide_empty,omitempty"`
}

type ViewProp struct {
	ID     string `json:"id"`
	Hidden bool   `json:"hidden,omitempty"`
	Width  int    `json:"width,omitempty"`
}

type ViewConfig struct {
	Props    []ViewProp        `json:"props,omitempty"`
	Filter   *Filter           `json:"filter,omitempty"`
	Sorts    []SortKey         `json:"sorts,omitempty"`
	Group    *Grouping         `json:"group,omitempty"`
	SubGroup *Grouping         `json:"sub_group,omitempty"`
	Calcs    map[string]string `json:"calcs,omitempty"`
	Layout   map[string]any    `json:"layout,omitempty"`
	Form     *FormConfig       `json:"form,omitempty"`
}

type RangeInput struct {
	Prop string `json:"prop"`
	End  string `json:"end_prop,omitempty"`
	From string `json:"from"`
	To   string `json:"to"`
}

type QueryInput struct {
	ViewID     string      `json:"view_id"`
	Filter     *Filter     `json:"filter,omitempty"`
	Sorts      []SortKey   `json:"sorts,omitempty"`
	Search     string      `json:"search,omitempty"`
	Offset     int         `json:"offset,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	GroupKey   *string     `json:"group_key,omitempty"`
	LaneKey    *string     `json:"lane_key,omitempty"`
	GroupLimit int         `json:"group_limit,omitempty"`
	Range      *RangeInput `json:"range,omitempty"`
	IDs        []string    `json:"ids,omitempty"`
	NoGroups   bool        `json:"no_groups,omitempty"`
	Tree       string      `json:"tree,omitempty"`
	Expanded   []string    `json:"expanded,omitempty"`
	Cursor     string      `json:"cursor,omitempty"`
}

type RowOut struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Icon      string         `json:"icon"`
	Cover     string         `json:"cover,omitempty"`
	Thumb     string         `json:"thumb,omitempty"`
	Snippet   string         `json:"snippet,omitempty"`
	SortKey   string         `json:"sort_key"`
	Values    map[string]any `json:"values"`
	Number    int64          `json:"number"`
	Version   int64          `json:"version"`
	CreatedAt int64          `json:"created_at"`
	CreatedBy string         `json:"created_by"`
	UpdatedAt int64          `json:"updated_at"`
	UpdatedBy string         `json:"updated_by"`
	Depth     int            `json:"depth,omitempty"`
	Children  int            `json:"children,omitempty"`
}

type GroupOut struct {
	Key    string         `json:"key"`
	Label  string         `json:"label"`
	Color  string         `json:"color,omitempty"`
	Count  int            `json:"count"`
	Rows   []string       `json:"rows"`
	Calcs  map[string]any `json:"calcs,omitempty"`
	Hidden bool           `json:"hidden,omitempty"`
}

type LaneOut struct {
	Key    string     `json:"key"`
	Label  string     `json:"label"`
	Color  string     `json:"color,omitempty"`
	Count  int        `json:"count"`
	Groups []GroupOut `json:"groups"`
	Hidden bool       `json:"hidden,omitempty"`
}

type Ref struct {
	Title      string `json:"title"`
	Icon       string `json:"icon"`
	Collection string `json:"collection"`
}

type QueryResult struct {
	Total      int            `json:"total"`
	Rows       []RowOut       `json:"rows"`
	Order      []string       `json:"order"`
	Groups     []GroupOut     `json:"groups,omitempty"`
	Lanes      []LaneOut      `json:"lanes,omitempty"`
	Calcs      map[string]any `json:"calcs,omitempty"`
	Refs       map[string]Ref `json:"refs"`
	Epoch      int64          `json:"epoch"`
	Offset     int            `json:"offset"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

func (ev *evaluator) rowOut(sn *snap, r *Row, refs map[string]Ref) RowOut {
	out := RowOut{ID: r.ID, Title: r.Title, Icon: r.Icon, Cover: r.Cover, Thumb: r.Thumb, Snippet: r.Snippet, SortKey: r.SortKey,
		Number: r.Number, Version: r.Version, CreatedAt: r.CreatedAt, CreatedBy: r.CreatedBy, UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedBy,
		Values: make(map[string]any, len(sn.schema))}
	for _, p := range sn.schema {
		switch p.Type {
		case TTitle, TButton:
			continue
		case TFormula, TRollup:
			out.Values[p.ID] = ev.encodeCell(sn, r, p)
		case TCreatedTime:
			out.Values[p.ID] = r.CreatedAt
		case TEditedTime:
			out.Values[p.ID] = r.UpdatedAt
		case TCreatedBy:
			out.Values[p.ID] = r.CreatedBy
		case TEditedBy:
			out.Values[p.ID] = r.UpdatedBy
		case TUniqueID:
			out.Values[p.ID] = r.Number
		case TRelation:
			ids := strList(r.Vals[p.ID])
			if len(ids) == 0 {
				continue
			}
			target := ev.snap(p.Collection)
			kept := make([]string, 0, len(ids))
			for _, id := range ids {
				if target == nil {
					continue
				}
				if t, ok := target.rows[id]; ok {
					kept = append(kept, id)
					if refs != nil {
						refs[id] = Ref{Title: t.Title, Icon: t.Icon, Collection: target.id}
					}
				}
			}
			if len(kept) > 0 {
				out.Values[p.ID] = kept
			}
		default:
			if raw, ok := r.Vals[p.ID]; ok && !isNull(raw) {
				out.Values[p.ID] = raw
			}
		}
	}
	return out
}

func (ev *evaluator) rawOption(r *Row, p *Property) []string {
	switch p.Type {
	case TSelect, TStatus:
		if id := str(r.Vals[p.ID]); id != "" {
			return []string{id}
		}
		return nil
	case TMulti, TPerson, TRelation:
		return strList(r.Vals[p.ID])
	case TCreatedBy:
		return []string{r.CreatedBy}
	case TEditedBy:
		return []string{r.UpdatedBy}
	}
	return nil
}

func filterKind(p *Property, v formula.Value) string {
	switch p.Type {
	case TTitle, TText, TURL, TEmail, TPhone, TPlace:
		return "text"
	case TNumber, TUniqueID:
		return "number"
	case TSelect, TStatus:
		return "option"
	case TMulti, TPerson, TRelation, TCreatedBy, TEditedBy:
		return "list"
	case TFiles:
		return "files"
	case TDate, TCreatedTime, TEditedTime:
		return "date"
	case TCheckbox:
		return "checkbox"
	}
	switch v.K {
	case formula.KNumber:
		return "number"
	case formula.KDate:
		return "date"
	case formula.KBool:
		return "checkbox"
	case formula.KList:
		return "valuelist"
	}
	return "text"
}

type filterVal struct {
	S        string
	N        float64
	B        bool
	List     []string
	Date     string
	Relative string
	Within   string
	To       string
	isNum    bool
}

func decodeFilterVal(raw json.RawMessage) filterVal {
	var fv filterVal
	if isNull(raw) {
		return fv
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		fv.S = s
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			fv.N, fv.isNum = f, true
		}
		fv.List = []string{s}
		return fv
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		fv.N, fv.isNum, fv.S = f, true, formula.FormatNumber(f)
		return fv
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		fv.B = b
		return fv
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		fv.List = list
		return fv
	}
	var obj struct {
		Date     string `json:"date"`
		To       string `json:"to"`
		Relative string `json:"relative"`
		Within   string `json:"within"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		fv.Date, fv.To, fv.Relative, fv.Within = obj.Date, obj.To, obj.Relative, obj.Within
	}
	return fv
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func weekStart(t time.Time, monday bool) time.Time {
	d := dayStart(t)
	wd := int(d.Weekday())
	if monday {
		wd = (wd + 6) % 7
	}
	return d.AddDate(0, 0, -wd)
}

func (ev *evaluator) relativeDay(name string) (time.Time, bool) {
	today := dayStart(ev.now.In(ev.loc))
	switch name {
	case "today":
		return today, true
	case "tomorrow":
		return today.AddDate(0, 0, 1), true
	case "yesterday":
		return today.AddDate(0, 0, -1), true
	case "one_week_ago":
		return today.AddDate(0, 0, -7), true
	case "one_week_from_now":
		return today.AddDate(0, 0, 7), true
	case "one_month_ago":
		return today.AddDate(0, -1, 0), true
	case "one_month_from_now":
		return today.AddDate(0, 1, 0), true
	}
	return time.Time{}, false
}

func (ev *evaluator) within(name string) (time.Time, time.Time, bool) {
	today := dayStart(ev.now.In(ev.loc))
	end := today.AddDate(0, 0, 1)
	switch name {
	case "past_week":
		return today.AddDate(0, 0, -7), end, true
	case "next_week":
		return today, today.AddDate(0, 0, 8), true
	case "past_month":
		return today.AddDate(0, -1, 0), end, true
	case "next_month":
		return today, today.AddDate(0, 1, 1), true
	case "past_year":
		return today.AddDate(-1, 0, 0), end, true
	case "next_year":
		return today, today.AddDate(1, 0, 1), true
	case "this_week":
		s := weekStart(today, ev.s.mondayFirst(ev.ctx, ev.p))
		return s, s.AddDate(0, 0, 7), true
	case "this_month":
		s := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, ev.loc)
		return s, s.AddDate(0, 1, 0), true
	case "this_year":
		s := time.Date(today.Year(), 1, 1, 0, 0, 0, 0, ev.loc)
		return s, s.AddDate(1, 0, 0), true
	}
	return time.Time{}, time.Time{}, false
}

func (ev *evaluator) filterDay(fv filterVal) (time.Time, bool) {
	if fv.Relative != "" {
		return ev.relativeDay(fv.Relative)
	}
	s := fv.Date
	if s == "" {
		s = fv.S
	}
	t, _, ok := parseDay(s)
	if !ok {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, ev.loc), true
}

func (ev *evaluator) matchOne(sn *snap, r *Row, f *Filter) bool {
	p := sn.schema.ByID(f.Prop)
	if p == nil {
		return true
	}
	v := ev.value(sn, r, p)
	fv := decodeFilterVal(f.Value)
	switch f.Cond {
	case "empty", "is_empty":
		if p.Type == TCheckbox {
			return !v.Truthy()
		}
		return v.IsEmpty()
	case "not_empty", "is_not_empty":
		if p.Type == TCheckbox {
			return v.Truthy()
		}
		return !v.IsEmpty()
	}
	switch filterKind(p, v) {
	case "text":
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
	case "number":
		if v.K != formula.KNumber {
			return f.Cond == "ne"
		}
		if !fv.isNum {
			return true
		}
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
	case "option":
		have := ev.rawOption(r, p)
		id := ""
		if len(have) > 0 {
			id = have[0]
		}
		want := fv.List
		in := func() bool {
			for _, w := range want {
				if w == id {
					return true
				}
			}
			return false
		}
		switch f.Cond {
		case "is", "any":
			return in()
		case "is_not", "none":
			return !in()
		case "group_is":
			o := p.Option(id)
			for _, w := range want {
				if o != nil && o.Group == w {
					return true
				}
			}
			return false
		}
	case "list":
		have := ev.rawOption(r, p)
		want := fv.List
		any := false
		for _, w := range want {
			if w == "me" {
				w = ev.p.UserID
			}
			for _, h := range have {
				if h == w {
					any = true
				}
			}
		}
		switch f.Cond {
		case "contains", "is", "any":
			return any
		case "not_contains", "is_not", "none":
			return !any
		}
	case "files", "valuelist":
		hay := strings.ToLower(v.String())
		needle := strings.ToLower(fv.S)
		switch f.Cond {
		case "contains":
			return strings.Contains(hay, needle)
		case "not_contains":
			return !strings.Contains(hay, needle)
		}
	case "checkbox":
		want := fv.B
		if fv.S == "true" {
			want = true
		}
		switch f.Cond {
		case "is", "eq":
			return v.Truthy() == want
		case "is_not", "ne":
			return v.Truthy() != want
		}
	case "date":
		if v.K != formula.KDate {
			return false
		}
		start := dayStart(v.T.In(ev.loc))
		end := start
		if !v.End.IsZero() {
			end = dayStart(v.End.In(ev.loc))
		}
		if f.Cond == "within" {
			a, b, ok := ev.within(fv.Within)
			if !ok {
				return true
			}
			return !end.Before(a) && start.Before(b)
		}
		day, ok := ev.filterDay(fv)
		if !ok {
			return true
		}
		switch f.Cond {
		case "is":
			return !day.Before(start) && !day.After(end)
		case "is_not":
			return day.Before(start) || day.After(end)
		case "before":
			return start.Before(day)
		case "after":
			return end.After(day)
		case "on_or_before":
			return !start.After(day)
		case "on_or_after":
			return !end.Before(day)
		case "between":
			to, ok := ev.filterDay(filterVal{Date: fv.To})
			if !ok {
				return !end.Before(day)
			}
			return !end.Before(day) && !start.After(to)
		}
	}
	return true
}

func (ev *evaluator) match(sn *snap, r *Row, f *Filter, depth int) bool {
	if f == nil || depth > 4 {
		return true
	}
	if f.Prop == "" {
		if len(f.Items) == 0 {
			return true
		}
		or := f.Op == "or"
		for i := range f.Items {
			ok := ev.match(sn, r, &f.Items[i], depth+1)
			if or && ok {
				return true
			}
			if !or && !ok {
				return false
			}
		}
		return !or
	}
	return ev.matchOne(sn, r, f)
}

func (ev *evaluator) search(sn *snap, r *Row, q string) bool {
	if strings.Contains(strings.ToLower(r.Title), q) {
		return true
	}
	for _, p := range sn.schema {
		switch p.Type {
		case TText, TURL, TEmail, TPhone, TSelect, TMulti, TStatus, TFormula, TPlace:
			if strings.Contains(strings.ToLower(ev.value(sn, r, p).String()), q) {
				return true
			}
		}
	}
	return false
}

func (ev *evaluator) overlaps(sn *snap, r *Row, rg *RangeInput) bool {
	p := sn.schema.ByID(rg.Prop)
	if p == nil {
		return false
	}
	v := ev.value(sn, r, p)
	if v.K != formula.KDate {
		return false
	}
	start := v.T
	end := v.End
	if rg.End != "" {
		if ep := sn.schema.ByID(rg.End); ep != nil {
			if e := ev.value(sn, r, ep); e.K == formula.KDate {
				end = e.T
				if !e.End.IsZero() {
					end = e.End
				}
			}
		}
	}
	if end.IsZero() || end.Before(start) {
		end = start
	}
	from, _, ok1 := parseDay(rg.From)
	to, _, ok2 := parseDay(rg.To)
	if !ok1 || !ok2 {
		return true
	}
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, ev.loc)
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, ev.loc)
	return !end.Before(from) && !start.After(to)
}

func (ev *evaluator) sortRows(sn *snap, rows []*Row, sorts []SortKey) {
	type keyed struct {
		r    *Row
		keys []formula.Value
	}
	var props []*Property
	var desc []bool
	for _, s := range sorts {
		if p := sn.schema.ByID(s.Prop); p != nil {
			props = append(props, p)
			desc = append(desc, s.Dir == "desc")
		}
	}
	if len(props) == 0 {
		return
	}
	ks := make([]keyed, len(rows))
	for i, r := range rows {
		k := keyed{r: r, keys: make([]formula.Value, len(props))}
		for j, p := range props {
			v := ev.value(sn, r, p)
			if p.Type == TSelect || p.Type == TStatus {
				if o := ev.rawOption(r, p); len(o) > 0 {
					for idx, opt := range p.Options {
						if opt.ID == o[0] {
							v = formula.Num(float64(idx))
						}
					}
				}
			}
			k.keys[j] = v
		}
		ks[i] = k
	}
	sort.SliceStable(ks, func(i, j int) bool {
		for x := range props {
			a, b := ks[i].keys[x], ks[j].keys[x]
			if a.IsEmpty() != b.IsEmpty() {
				return b.IsEmpty()
			}
			c, ok := formula.Compare(a, b)
			if !ok {
				c = strings.Compare(a.String(), b.String())
			}
			if c != 0 {
				if desc[x] {
					return c > 0
				}
				return c < 0
			}
		}
		return false
	})
	for i := range ks {
		rows[i] = ks[i].r
	}
}

type groupKey struct {
	key, label, color string
	rank              float64
}

func (ev *evaluator) groupKeys(sn *snap, r *Row, g *Grouping) []groupKey {
	p := sn.schema.ByID(g.Prop)
	if p == nil {
		return []groupKey{{key: "", label: ""}}
	}
	none := groupKey{key: "", label: "No " + p.Name, rank: math.Inf(1)}
	switch p.Type {
	case TSelect, TStatus, TMulti:
		ids := ev.rawOption(r, p)
		if p.Type == TStatus && g.By == "group" {
			if len(ids) == 0 {
				return []groupKey{none}
			}
			o := p.Option(ids[0])
			if o == nil {
				return []groupKey{none}
			}
			for i, gname := range StatusGroups {
				if gname == o.Group {
					return []groupKey{{key: gname, label: statusGroupLabel(gname), rank: float64(i)}}
				}
			}
			return []groupKey{none}
		}
		var out []groupKey
		for _, id := range ids {
			for i, o := range p.Options {
				if o.ID == id {
					out = append(out, groupKey{key: id, label: o.Name, color: o.Color, rank: float64(i)})
				}
			}
		}
		if len(out) == 0 {
			return []groupKey{none}
		}
		return out
	case TPerson, TRelation, TCreatedBy, TEditedBy:
		ids := ev.rawOption(r, p)
		var out []groupKey
		for _, id := range ids {
			label := ""
			if p.Type == TRelation {
				if t := ev.snap(p.Collection); t != nil {
					if tr, ok := t.rows[id]; ok {
						label = tr.Title
					}
				}
			} else {
				label = ev.person(id).Name
			}
			if label == "" && p.Type == TRelation {
				continue
			}
			out = append(out, groupKey{key: id, label: label, rank: math.NaN()})
		}
		if len(out) == 0 {
			return []groupKey{none}
		}
		return out
	case TCheckbox:
		if ev.value(sn, r, p).Truthy() {
			return []groupKey{{key: "true", label: "Checked", rank: 0}}
		}
		return []groupKey{{key: "false", label: "Unchecked", rank: 1}}
	}
	v := ev.value(sn, r, p)
	if v.IsEmpty() {
		return []groupKey{none}
	}
	switch v.K {
	case formula.KDate:
		t := v.T.In(ev.loc)
		switch g.By {
		case "year":
			return []groupKey{{key: t.Format("2006"), label: t.Format("2006"), rank: float64(t.Year())}}
		case "month":
			s := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, ev.loc)
			return []groupKey{{key: s.Format("2006-01"), label: s.Format("January 2006"), rank: float64(s.Unix())}}
		case "weekday":
			wd := int(t.Weekday())
			rank := wd
			if ev.s.mondayFirst(ev.ctx, ev.p) {
				rank = (wd + 6) % 7
			}
			return []groupKey{{key: "d" + strconv.Itoa(wd), label: t.Weekday().String(), rank: float64(rank)}}
		case "hour":
			return []groupKey{{key: fmt.Sprintf("h%02d", t.Hour()), label: fmt.Sprintf("%02d:00", t.Hour()), rank: float64(t.Hour())}}
		case "week":
			s := weekStart(t, ev.s.mondayFirst(ev.ctx, ev.p))
			return []groupKey{{key: s.Format("2006-01-02") + "w", label: "Week of " + s.Format("Jan 2, 2006"), rank: float64(s.Unix())}}
		case "relative":
			today := dayStart(ev.now.In(ev.loc))
			d := dayStart(t)
			days := int(math.Round(d.Sub(today).Hours() / 24))
			switch {
			case days == 0:
				return []groupKey{{key: "r0", label: "Today", rank: 0}}
			case days == 1:
				return []groupKey{{key: "r1", label: "Tomorrow", rank: 1}}
			case days == -1:
				return []groupKey{{key: "r-1", label: "Yesterday", rank: -1}}
			case days > 1 && days <= 7:
				return []groupKey{{key: "r7", label: "Next 7 days", rank: 7}}
			case days < -1 && days >= -7:
				return []groupKey{{key: "r-7", label: "Last 7 days", rank: -7}}
			case days > 7 && days <= 30:
				return []groupKey{{key: "r30", label: "Next 30 days", rank: 30}}
			case days < -7 && days >= -30:
				return []groupKey{{key: "r-30", label: "Last 30 days", rank: -30}}
			case days > 30:
				return []groupKey{{key: "rf", label: "Later", rank: 1e6}}
			}
			return []groupKey{{key: "rp", label: "Older", rank: -1e6}}
		}
		d := dayStart(t)
		return []groupKey{{key: d.Format("2006-01-02"), label: d.Format("Jan 2, 2006"), rank: float64(d.Unix())}}
	case formula.KNumber:
		step := g.Step
		if step <= 0 {
			step = 10
		}
		lo := math.Floor(v.N/step) * step
		label := fmt.Sprintf("%s to %s", formula.FormatNumber(lo), formula.FormatNumber(lo+step))
		return []groupKey{{key: formula.FormatNumber(lo), label: label, rank: lo}}
	case formula.KBool:
		if v.B {
			return []groupKey{{key: "true", label: "True", rank: 0}}
		}
		return []groupKey{{key: "false", label: "False", rank: 1}}
	}
	s := v.String()
	return []groupKey{{key: strings.ToLower(s), label: s, rank: math.NaN()}}
}

func statusGroupLabel(g string) string {
	switch g {
	case "todo":
		return "To do"
	case "doing":
		return "In progress"
	}
	return "Complete"
}

type bucket struct {
	groupKey
	rows []*Row
}

func (ev *evaluator) bucketise(sn *snap, rows []*Row, g *Grouping) []*bucket {
	byKey := map[string]*bucket{}
	var list []*bucket
	p := sn.schema.ByID(g.Prop)
	if p != nil && (p.Type == TSelect || p.Type == TStatus || p.Type == TMulti) && g.By != "group" {
		for i, o := range p.Options {
			b := &bucket{groupKey: groupKey{key: o.ID, label: o.Name, color: o.Color, rank: float64(i)}}
			byKey[o.ID] = b
			list = append(list, b)
		}
	}
	if p != nil && p.Type == TStatus && g.By == "group" {
		for i, gname := range StatusGroups {
			b := &bucket{groupKey: groupKey{key: gname, label: statusGroupLabel(gname), rank: float64(i)}}
			byKey[gname] = b
			list = append(list, b)
		}
	}
	if p != nil && p.Type == TCheckbox {
		for i, k := range []string{"true", "false"} {
			label := "Checked"
			if k == "false" {
				label = "Unchecked"
			}
			b := &bucket{groupKey: groupKey{key: k, label: label, rank: float64(i)}}
			byKey[k] = b
			list = append(list, b)
		}
	}
	for _, r := range rows {
		for _, k := range ev.groupKeys(sn, r, g) {
			b, ok := byKey[k.key]
			if !ok {
				b = &bucket{groupKey: k}
				byKey[k.key] = b
				list = append(list, b)
			}
			b.rows = append(b.rows, r)
		}
	}
	if _, ok := byKey[""]; !ok && p != nil {
		b := &bucket{groupKey: groupKey{key: "", label: "No " + p.Name, rank: math.Inf(1)}}
		list = append(list, b)
	}
	desc := g.Sort == "desc"
	less := func(a, b *bucket) bool {
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
			switch {
			case oki && okj:
				return pi < pj
			case oki:
				return true
			}
			return false
		})
	}
	return list
}

var calcFns = map[string]bool{
	"count_all": true, "count_values": true, "count_unique": true, "count_empty": true, "count_not_empty": true, "percent_empty": true,
	"percent_not_empty": true, "sum": true, "average": true, "median": true, "min": true, "max": true, "range": true, "earliest": true,
	"latest": true, "date_range": true, "checked": true, "unchecked": true, "percent_checked": true, "percent_unchecked": true,
}

func (ev *evaluator) calcs(sn *snap, rows []*Row, calcs map[string]string) map[string]any {
	if len(calcs) == 0 {
		return nil
	}
	out := map[string]any{}
	for propID, fn := range calcs {
		p := sn.schema.ByID(propID)
		if p == nil || !calcFns[fn] {
			continue
		}
		vals := make([]formula.Value, len(rows))
		for i, r := range rows {
			v := ev.value(sn, r, p)
			if p.Type == TCheckbox && !v.Truthy() {
				v = formula.Bool(false)
			}
			vals[i] = v
		}
		if fn == "checked" || fn == "unchecked" || fn == "percent_checked" || fn == "percent_unchecked" {
			for i := range vals {
				vals[i] = formula.Bool(vals[i].Truthy())
			}
		}
		out[propID] = encodeValue(aggregateRollup(fn, vals, len(rows)))
	}
	return out
}

func hiddenSet(list []string) map[string]bool {
	m := map[string]bool{}
	for _, k := range list {
		m[k] = true
	}
	return m
}
