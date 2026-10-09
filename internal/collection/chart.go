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
}

type ChartInput struct {
	ViewID string    `json:"view_id"`
	Chart  ChartSpec `json:"chart"`
	Search string    `json:"search,omitempty"`
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
}

var chartKinds = map[string]bool{"bar": true, "hbar": true, "line": true, "donut": true, "number": true}

func badChart(msg string) error { return httpx.Fail(http.StatusBadRequest, "bad_chart", msg) }

func (s *Service) Chart(ctx context.Context, p *policy.Principal, collectionID string, in ChartInput) (*ChartResult, error) {
	spec := in.Chart
	if !chartKinds[spec.Kind] {
		return nil, badChart("a chart is bar, hbar, line, donut or number")
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
	all := s.results.ordered(ev, sn, cfg, QueryInput{Search: in.Search})
	res := &ChartResult{Kind: spec.Kind, X: []ChartPoint{}, Series: []ChartSeries{}, Rows: len(all), Epoch: s.epoch.Load()}
	var yp *Property
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
	if spec.Kind == "number" {
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
	if spec.Series != nil && spec.Kind != "donut" && sn.schema.ByID(spec.Series.Prop) != nil {
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
	limit := spec.Limit
	if limit <= 0 {
		limit = 100
	}
	limit = min(limit, 500)
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
			if spec.Cumulative && spec.Kind == "line" {
				run += vals[j]
				vals[j] = run
			}
		}
		res.Series = append(res.Series, ChartSeries{Key: l.Key, Label: l.Label, Color: l.Color, Values: vals})
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
