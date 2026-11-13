package collection

import (
	"context"
	"net/http"
	"sort"

	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type ChartSpec struct {
	Kind       string    `json:"kind"`
	X          *Grouping `json:"x,omitempty"`
	Series     *Grouping `json:"series,omitempty"`
	Fn         string    `json:"fn,omitempty"`
	Prop       string    `json:"prop,omitempty"`
	Sort       string    `json:"sort,omitempty"`
	Cumulative bool      `json:"cumulative,omitempty"`
	HideEmpty  bool      `json:"hide_empty,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Filter     *Filter   `json:"filter,omitempty"`
	Title      string    `json:"title,omitempty"`
	Labels     bool      `json:"labels,omitempty"`
	Legend     bool      `json:"legend,omitempty"`
	Smooth     bool      `json:"smooth,omitempty"`
	Stacked    bool      `json:"stacked,omitempty"`
	Target     float64   `json:"target,omitempty"`
	Window     int       `json:"window,omitempty"`
	Scale      string    `json:"scale,omitempty"`
	Size       string    `json:"size,omitempty"`
}

type ChartPick struct {
	X      string `json:"x"`
	Series string `json:"series,omitempty"`
}

type ChartInput struct {
	ViewID    string     `json:"view_id"`
	Chart     ChartSpec  `json:"chart"`
	Search    string     `json:"search,omitempty"`
	Pick      *ChartPick `json:"pick,omitempty"`
	DateRange *DateRange `json:"date_range,omitempty"`
}

type ChartDot struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Group string  `json:"group"`
	Value float64 `json:"value"`
}

type ChartPoint struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Color string `json:"color,omitempty"`
}

type ChartSeries struct {
	Key    string    `json:"key"`
	Label  string    `json:"label"`
	Color  string    `json:"color,omitempty"`
	Values []float64 `json:"values"`
}

type ChartResult struct {
	Kind      string        `json:"kind"`
	X         []ChartPoint  `json:"x"`
	Series    []ChartSeries `json:"series"`
	Value     float64       `json:"value"`
	Rows      int           `json:"rows"`
	Truncated bool          `json:"truncated,omitempty"`
	Epoch     int64         `json:"epoch"`
	Points    []ChartDot    `json:"points,omitempty"`
	Picked    []string      `json:"picked,omitempty"`
}

var chartKinds = map[string]bool{"bar": true, "hbar": true, "line": true, "donut": true, "number": true, "area": true, "radial": true, "gauge": true, "histogram": true, "dots": true, "compare": true, "heatmap": true, "calendar": true}

const maxDots = 2000

func singleSeries(kind string) bool {
	return kind == "donut" || kind == "radial" || kind == "histogram" || kind == "calendar" || kind == "compare"
}

func badChart(msg string) error { return httpx.Fail(http.StatusBadRequest, "bad_chart", msg) }

func (s *Service) Chart(ctx context.Context, p *policy.Principal, collectionID string, in ChartInput) (*ChartResult, error) {
	spec := in.Chart
	if !chartKinds[spec.Kind] {
		return nil, badChart("unknown chart kind " + spec.Kind)
	}
	if spec.Fn != "" && spec.Fn != "count" && !calcFns[spec.Fn] {
		return nil, badChart("unknown measure " + spec.Fn)
	}
	if !filterDepth(spec.Filter, 1) {
		return nil, badChart("chart filters nest up to two groups deep")
	}
	if _, _, err := s.readable(ctx, p, collectionID); err != nil {
		return nil, err
	}
	var cfg ViewConfig
	if in.ViewID != "" {
		v, err := s.view(ctx, in.ViewID)
		if err != nil {
			return nil, err
		}
		if v.CollectionID != collectionID {
			return nil, store.ErrNotFound
		}
		cfg.Filter = v.Config.Filter
	}
	if spec.Filter != nil {
		if cfg.Filter == nil {
			cfg.Filter = spec.Filter
		} else {
			cfg.Filter = &Filter{Op: "and", Items: []Filter{*cfg.Filter, *spec.Filter}}
		}
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn := ev.snap(collectionID)
	if sn == nil {
		return nil, store.ErrNotFound
	}
	narrow, err := in.DateRange.compile(sn.schema)
	if err != nil {
		return nil, err
	}
	all := s.results.ordered(ev, sn, cfg, QueryInput{Search: in.Search, narrow: narrow})
	res := &ChartResult{Kind: spec.Kind, X: []ChartPoint{}, Series: []ChartSeries{}, Rows: len(all), Epoch: s.epoch.Load()}
	var yp *Property
	if spec.Kind == "dots" {
		return s.chartDots(ev, sn, all, spec, res)
	}
	if spec.Fn != "" && spec.Fn != "count" && spec.Fn != "count_all" {
		if yp = sn.schema.ByID(spec.Prop); yp == nil {
			return nil, badChart("pick a property to measure")
		}
	}
	measure := func(rows []*Row) float64 {
		if yp == nil {
			return float64(len(rows))
		}
		return ev.measure(sn, rows, yp, spec.Fn)
	}
	if spec.Kind == "number" || spec.Kind == "gauge" {
		res.Value = measure(all)
		return res, nil
	}
	if spec.X == nil || sn.schema.ByID(spec.X.Prop) == nil {
		return nil, badChart("pick a property for the x axis")
	}
	hidden := hiddenSet(spec.X.Hidden)
	var xs []*bucket
	for _, b := range ev.bucketise(sn, all, spec.X) {
		if hidden[b.key] || (len(b.rows) == 0 && (b.key == "" || spec.HideEmpty)) {
			continue
		}
		xs = append(xs, b)
	}
	type lane struct {
		ChartSeries
		member map[*Row]bool
	}
	var lanes []*lane
	if spec.Series != nil && !singleSeries(spec.Kind) && sn.schema.ByID(spec.Series.Prop) != nil {
		sh := hiddenSet(spec.Series.Hidden)
		for _, b := range ev.bucketise(sn, all, spec.Series) {
			if sh[b.key] || len(b.rows) == 0 {
				continue
			}
			l := &lane{ChartSeries: ChartSeries{Key: b.key, Label: b.label, Color: b.color}, member: map[*Row]bool{}}
			for _, r := range b.rows {
				l.member[r] = true
			}
			lanes = append(lanes, l)
			if len(lanes) == 24 {
				res.Truncated = true
				break
			}
		}
	}
	if len(lanes) == 0 {
		lanes = []*lane{{ChartSeries: ChartSeries{Key: "", Label: ""}}}
	}
	if in.Pick != nil {
		res.Picked = []string{}
		for _, b := range xs {
			if b.key != in.Pick.X {
				continue
			}
			for _, r := range b.rows {
				if in.Pick.Series != "" {
					member := false
					for _, l := range lanes {
						if l.Key == in.Pick.Series && l.member[r] {
							member = true
						}
					}
					if !member {
						continue
					}
				}
				res.Picked = append(res.Picked, r.ID)
				if len(res.Picked) == 500 {
					res.Truncated = true
					return res, nil
				}
			}
		}
		return res, nil
	}
	totals := make([]float64, len(xs))
	for _, l := range lanes {
		l.Values = make([]float64, len(xs))
		for i, b := range xs {
			rows := b.rows
			if l.member != nil {
				rows = nil
				for _, r := range b.rows {
					if l.member[r] {
						rows = append(rows, r)
					}
				}
			}
			l.Values[i] = measure(rows)
			totals[i] += l.Values[i]
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
	limit, ceiling := spec.Limit, 500
	if spec.Kind == "calendar" {
		ceiling = 1100
	}
	if limit <= 0 {
		limit = 100
		if spec.Kind == "calendar" {
			limit = ceiling
		}
	}
	limit = min(limit, ceiling)
	if len(order) > limit {
		order = order[:limit]
		res.Truncated = true
	}
	for _, i := range order {
		b := xs[i]
		res.X = append(res.X, ChartPoint{Key: b.key, Label: b.label, Color: b.color})
	}
	for _, l := range lanes {
		vals := make([]float64, len(order))
		run := 0.0
		for j, i := range order {
			vals[j] = l.Values[i]
			if spec.Cumulative && (spec.Kind == "line" || spec.Kind == "area") {
				run += vals[j]
				vals[j] = run
			}
		}
		res.Series = append(res.Series, ChartSeries{Key: l.Key, Label: l.Label, Color: l.Color, Values: vals})
	}
	return res, nil
}

func (s *Service) chartDots(ev *evaluator, sn *snap, all []*Row, spec ChartSpec, res *ChartResult) (*ChartResult, error) {
	vp := sn.schema.ByID(spec.Prop)
	if vp == nil {
		return nil, badChart("pick a number property for the dots")
	}
	group := map[*Row]string{}
	if spec.X != nil && sn.schema.ByID(spec.X.Prop) != nil {
		hidden := hiddenSet(spec.X.Hidden)
		for _, b := range ev.bucketise(sn, all, spec.X) {
			if hidden[b.key] || len(b.rows) == 0 {
				continue
			}
			res.X = append(res.X, ChartPoint{Key: b.key, Label: b.label, Color: b.color})
			for _, r := range b.rows {
				if _, ok := group[r]; !ok {
					group[r] = b.key
				}
			}
		}
	}
	res.Points = []ChartDot{}
	for _, r := range all {
		v := ev.value(sn, r, vp)
		if v.K != formula.KNumber {
			continue
		}
		g, ok := group[r]
		if spec.X != nil && !ok && len(res.X) > 0 {
			continue
		}
		res.Points = append(res.Points, ChartDot{ID: r.ID, Title: r.Title, Group: g, Value: v.N})
		if len(res.Points) == maxDots {
			res.Truncated = true
			break
		}
	}
	return res, nil
}

func (ev *evaluator) measure(sn *snap, rows []*Row, p *Property, fn string) float64 {
	vals := make([]formula.Value, len(rows))
	boolish := fn == "checked" || fn == "unchecked" || fn == "percent_checked" || fn == "percent_unchecked"
	for i, r := range rows {
		v := ev.value(sn, r, p)
		if boolish || (p.Type == TCheckbox && !v.Truthy()) {
			v = formula.Bool(v.Truthy())
		}
		vals[i] = v
	}
	out := aggregateRollup(fn, vals, len(rows))
	if out.K != formula.KNumber {
		return 0
	}
	return out.N
}
