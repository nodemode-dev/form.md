package templates

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/collection/formula"
)

const pvW = 592.0

type pvWidget struct {
	Title string `json:"title"`
	Wide  bool   `json:"wide"`
	Size  string `json:"size"`
	KPI   *struct {
		Date string `json:"date"`
		By   string `json:"by"`
	} `json:"kpi"`
	Chart collection.ChartSpec `json:"chart"`
}

type pvHeat struct {
	Prop   string `json:"prop"`
	Domain string `json:"domain"`
	Count  int    `json:"count"`
	Fn     string `json:"fn"`
	Value  string `json:"value"`
	Color  string `json:"color"`
}

type pvLayout struct {
	Chart   *collection.ChartSpec `json:"chart"`
	Widgets []pvWidget            `json:"widgets"`
	Heatmap *pvHeat               `json:"heatmap"`
	Date    string                `json:"date"`
	End     string                `json:"end"`
	Place   string                `json:"place"`
	Cover   string                `json:"cover"`
	Size    string                `json:"size"`
}

type pvView struct {
	Filter *collection.Filter   `json:"filter"`
	Sorts  []collection.SortKey `json:"sorts"`
	Group  *collection.Grouping `json:"group"`
	Layout pvLayout             `json:"layout"`
}

func parseView(cfg string) pvView {
	var v pvView
	_ = json.Unmarshal([]byte(cfg), &v)
	return v
}

var visualViews = map[string]bool{"chart": true, "dashboard": true, "heatmap": true, "calendar": true, "timeline": true, "map": true, "graph": true, "gallery": true}

func (t *pvTree) visualHTML(d *pvDB, v view) string {
	cfg := parseView(v.Config)
	switch v.Type {
	case "chart":
		spec := defaultChart(d)
		if cfg.Layout.Chart != nil && cfg.Layout.Chart.Kind != "" {
			spec = *cfg.Layout.Chart
		}
		return `<div class="tplp-viz">` + t.chartSVG(d, spec, cfg.Filter, pvW, 250) + `</div>`
	case "dashboard":
		return t.dashboardHTML(d, cfg)
	case "heatmap":
		return t.heatHTML(d, cfg)
	case "calendar":
		return t.calendarHTML(d, cfg)
	case "timeline":
		return t.timelineHTML(d, cfg)
	case "map":
		return t.mapHTML(d, cfg)
	case "graph":
		return t.graphHTML(d, cfg)
	case "gallery":
		return t.galleryHTML(d, cfg)
	}
	return ""
}

func defaultChart(d *pvDB) collection.ChartSpec {
	for _, types := range [][]string{{"status"}, {"select"}, {"multi_select"}, {"checkbox"}, {"person"}, {"date", "created_time"}} {
		for _, p := range d.st.schema {
			for _, ty := range types {
				if p.Type == ty {
					g := &collection.Grouping{Prop: p.ID}
					if ty == "date" || ty == "created_time" {
						g.By = "month"
					}
					return collection.ChartSpec{Kind: "bar", X: g, Fn: "count"}
				}
			}
		}
	}
	return collection.ChartSpec{Kind: "number", Fn: "count"}
}

func (t *pvTree) chartSVG(d *pvDB, spec collection.ChartSpec, filter *collection.Filter, W, H float64) string {
	if spec.Kind == "" {
		spec.Kind = "bar"
	}
	res := t.chart(d, spec, filter)
	return drawChart(res, spec, chartFmt(d, spec), chartCaption(d, spec), W, H)
}

func (t *pvTree) dashboardHTML(d *pvDB, cfg pvView) string {
	ws := cfg.Layout.Widgets
	if len(ws) == 0 {
		ws = []pvWidget{{Title: "Total", Chart: collection.ChartSpec{Kind: "number", Fn: "count"}}}
		if def := defaultChart(d); def.Kind != "number" {
			ws = append(ws, pvWidget{Chart: def})
		}
	}
	if len(ws) > 6 {
		ws = ws[:6]
	}
	const gap, pad = 10.0, 10.0
	col := (pvW - 3*gap) / 4
	var b strings.Builder
	b.WriteString(`<div class="tplp-dash">`)
	for _, w := range ws {
		span := 2
		switch {
		case w.Size == "s":
			span = 1
		case w.Size == "l" || (w.Size == "" && w.Wide):
			span = 4
		}
		width := col*float64(span) + gap*float64(span-1) - 2*pad
		h := 150.0
		if w.Chart.Kind == "number" || w.Chart.Kind == "gauge" {
			h = 84
		}
		title := w.Title
		if title == "" {
			title = chartCaption(d, w.Chart)
		}
		b.WriteString(`<div class="tplp-w" data-span="` + strconv.Itoa(span) + `"><div class="tplp-w__title">` + esc(title) + `</div>`)
		if w.KPI != nil && w.Chart.Kind == "number" {
			b.WriteString(t.kpiSVG(d, w, cfg.Filter, width, h))
		} else {
			b.WriteString(t.chartSVG(d, w.Chart, cfg.Filter, width, h))
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (t *pvTree) kpiSVG(d *pvDB, w pvWidget, filter *collection.Filter, W, H float64) string {
	spec := w.Chart
	f := chartFmt(d, spec)
	value := t.chart(d, spec, filter).value
	by := w.KPI.By
	if by == "" {
		by = "month"
	}
	spark := collection.ChartSpec{Kind: "line", X: &collection.Grouping{Prop: w.KPI.Date, By: by}, Fn: spec.Fn, Prop: spec.Prop, Filter: spec.Filter}
	res := t.chart(d, spark, filter)
	b := svgOpen(W, H, "kpi")
	size := math.Min(math.Min(30, H*0.36), (W-6)/(float64(len([]rune(f.full(value))))*0.6))
	b.text(2, size+2, "", "pv-big", f.full(value), size)
	if len(res.series) > 0 && len(res.series[0].vals) > 1 {
		vals := res.series[0].vals
		n := len(vals)
		last, prev := vals[n-1], vals[n-2]
		if prev != 0 {
			ch := roundTo((last-prev)/math.Abs(prev)*100, 1)
			sign, cls := "", "pv-delta"
			if ch > 0 {
				sign = "+"
			} else if ch < 0 {
				cls += " pv-delta--down"
			}
			b.text(2, size+20, "", cls, sign+strconv.FormatFloat(ch, 'f', -1, 64)+"% vs last "+by, 11)
		}
		lo, hi := vals[0], vals[0]
		for _, v := range vals {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
		if hi == lo {
			hi = lo + 1
		}
		x0 := math.Max(W*0.52, textW(f.full(value), size)+16)
		if W-x0 < 60 {
			return b.done()
		}
		pts := make([][2]float64, n)
		for i, v := range vals {
			pts[i] = [2]float64{x0 + (W-x0-4)*float64(i)/float64(n-1), H - 8 - (v-lo)/(hi-lo)*(H-24)}
		}
		b.WriteString(`<path d="` + smoothPath(pts) + `" class="pv-line" data-p="0"/>`)
	}
	return b.done()
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (t *pvTree) firstOf(d *pvDB, want string, types ...string) *prop {
	if p := d.byID[want]; p != nil {
		for _, ty := range types {
			if p.Type == ty {
				return p
			}
		}
	}
	for _, ty := range types {
		for i := range d.st.schema {
			if d.st.schema[i].Type == ty {
				return &d.st.schema[i]
			}
		}
	}
	return nil
}

func (t *pvTree) heatHTML(d *pvDB, cfg pvView) string {
	h := pvHeat{}
	if cfg.Layout.Heatmap != nil {
		h = *cfg.Layout.Heatmap
	}
	p := t.firstOf(d, h.Prop, "date", "created_time", "last_edited_time")
	if p == nil {
		return ""
	}
	byDay := map[string][]int{}
	for _, ri := range t.rowsOf(d, cfg.Filter) {
		v := t.value(d, ri, p)
		if v.K == formula.KDate {
			k := dayKeyOf(v.T)
			byDay[k] = append(byDay[k], ri)
		}
	}
	fn := h.Fn
	if fn == "" {
		fn = "count"
	}
	value := func(k string) float64 {
		rows := byDay[k]
		if len(rows) == 0 {
			return 0
		}
		return t.measure(d, rows, fn, h.Value)
	}
	var from, to time.Time
	now := t.now
	switch h.Domain {
	case "", "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(1, 0, 0)
	default:
		n := h.Count
		if h.Domain != "month" || n <= 0 {
			n = 6
		}
		n = min(n, 12)
		to = monthStart(now).AddDate(0, 1, 0)
		from = to.AddDate(0, -n, 0)
	}
	color := ""
	if h.Color != "" {
		color = ` data-heat="` + esc(h.Color) + `"`
	}
	total := 0.0
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		total += value(dayKeyOf(d))
	}
	f := pvFmt{}
	if vp := d.byID[h.Value]; vp != nil && fn != "count" {
		f.format, f.precision = vp.Format, vp.Precision
	}
	return `<div class="tplp-viz tplp-heat"` + color + `>` + calendarSVG(from, to, value, pvW, 220, "heatmap") + `<div class="tplp-heat__foot"><span>` + esc(f.full(total)) + ` in this period</span><span class="tplp-heat__key"><i data-l="0"></i><i data-l="1"></i><i data-l="2"></i><i data-l="3"></i><i data-l="4"></i><i data-l="5"></i></span></div></div>`
}

func (t *pvTree) span(d *pvDB, ri int, p, end *prop) (time.Time, time.Time, bool) {
	v := t.value(d, ri, p)
	if v.K != formula.KDate {
		return time.Time{}, time.Time{}, false
	}
	a, b := v.T, v.End
	if end != nil {
		if e := t.value(d, ri, end); e.K == formula.KDate {
			b = e.T
			if !e.End.IsZero() {
				b = e.End
			}
		}
	}
	if b.IsZero() || b.Before(a) {
		b = a
	}
	return a, b, true
}

func tone(d *pvDB, ri int) string {
	for _, p := range d.st.schema {
		if p.Type == "status" || p.Type == "select" {
			for _, id := range strs(d.st.rows[ri].Values[p.ID]) {
				if o, ok := findOpt(p, id); ok && o.Color != "" && o.Color != "default" {
					return o.Color
				}
			}
		}
	}
	return ""
}

func (t *pvTree) calendarHTML(d *pvDB, cfg pvView) string {
	p := t.firstOf(d, cfg.Layout.Date, "date", "created_time")
	if p == nil {
		return ""
	}
	first := monthStart(t.now)
	start := first.AddDate(0, 0, -weekRow(first))
	last := first.AddDate(0, 1, -1)
	weeks := int(math.Ceil(float64(int(last.Sub(start).Hours()/24)+1) / 7))
	days := map[string][]int{}
	for _, ri := range t.rowsOf(d, cfg.Filter) {
		a, b, ok := t.span(d, ri, p, nil)
		if !ok {
			continue
		}
		for x := a; !x.After(b) && x.Sub(a).Hours() < 24*42; x = x.AddDate(0, 0, 1) {
			days[dayKeyOf(x)] = append(days[dayKeyOf(x)], ri)
		}
	}
	var s strings.Builder
	s.WriteString(`<div class="tplp-cal"><div class="tplp-cal__month">` + esc(first.Format("January 2006")) + `</div><div class="tplp-cal__grid">`)
	for _, n := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"} {
		s.WriteString(`<span class="tplp-cal__wd">` + n + `</span>`)
	}
	today := dayKeyOf(t.now)
	for i := 0; i < weeks*7; i++ {
		day := start.AddDate(0, 0, i)
		k := dayKeyOf(day)
		attrs := ""
		if day.Month() != first.Month() {
			attrs += ` data-out="true"`
		}
		if k == today {
			attrs += ` data-today="true"`
		}
		s.WriteString(`<div class="tplp-cal__day"` + attrs + `><span class="tplp-cal__n">` + strconv.Itoa(day.Day()) + `</span>`)
		rows := days[k]
		for j, ri := range rows {
			if j == 2 && len(rows) > 3 {
				s.WriteString(`<span class="tplp-cal__more">+` + strconv.Itoa(len(rows)-2) + `</span>`)
				break
			}
			c := tone(d, ri)
			attr := ""
			if c != "" {
				attr = ` data-color="` + esc(c) + `"`
			}
			s.WriteString(`<span class="tplp-cal__ev"` + attr + `>` + esc(d.st.rows[ri].Title) + `</span>`)
		}
		s.WriteString(`</div>`)
	}
	s.WriteString(`</div></div>`)
	return s.String()
}

func (t *pvTree) timelineHTML(d *pvDB, cfg pvView) string {
	p := t.firstOf(d, cfg.Layout.Date, "date", "created_time")
	if p == nil {
		return ""
	}
	end := d.byID[cfg.Layout.End]
	if end != nil && end.Type != "date" {
		end = nil
	}
	type bar struct {
		ri   int
		a, b time.Time
	}
	var bars []bar
	for _, ri := range t.rowsOf(d, cfg.Filter) {
		if a, b, ok := t.span(d, ri, p, end); ok {
			bars = append(bars, bar{ri, a, b})
		}
	}
	if len(bars) == 0 {
		return ""
	}
	sort.SliceStable(bars, func(i, j int) bool { return bars[i].a.Before(bars[j].a) })
	if len(bars) > 11 {
		bars = bars[:11]
	}
	lo, hi := bars[0].a, bars[0].b
	for _, x := range bars {
		if x.a.Before(lo) {
			lo = x.a
		}
		if x.b.After(hi) {
			hi = x.b
		}
	}
	lo = lo.AddDate(0, 0, -2)
	hi = hi.AddDate(0, 0, 3)
	if hi.Sub(lo).Hours() < 24*21 {
		hi = lo.AddDate(0, 0, 21)
	}
	const left, top, rowH = 168.0, 26.0, 26.0
	H := top + rowH*float64(len(bars)) + 6
	b := svgOpen(pvW, H, "timeline")
	span := hi.Sub(lo).Hours()
	x := func(tm time.Time) float64 { return left + tm.Sub(lo).Hours()/span*(pvW-left-4) }
	days := span / 24
	var ticks []time.Time
	switch {
	case days <= 45:
		for tm := lo.AddDate(0, 0, (7-weekRow(lo))%7); tm.Before(hi); tm = tm.AddDate(0, 0, 7) {
			ticks = append(ticks, tm)
		}
	case days <= 200:
		for tm := monthStart(lo).AddDate(0, 1, 0); tm.Before(hi); tm = tm.AddDate(0, 0, 14) {
			ticks = append(ticks, tm)
		}
	default:
		for tm := monthStart(lo).AddDate(0, 1, 0); tm.Before(hi); tm = tm.AddDate(0, 1, 0) {
			ticks = append(ticks, tm)
		}
	}
	for _, tm := range ticks {
		b.line(x(tm), top-6, x(tm), H, "pv-grid")
		b.text(x(tm)+3, 12, "", "pv-tick", tm.Format("Jan 2"), 0)
	}
	b.line(left, top-6, pvW, top-6, "pv-base")
	for i, br := range bars {
		y := top + rowH*float64(i)
		if i%2 == 1 {
			b.rect(0, y, pvW, rowH, 0, ` class="pv-band"`)
		}
		b.text(4, y+rowH/2+4, "", "pv-leg", clipText(d.st.rows[br.ri].Title, left-12, 12), 12)
		x0, x1 := x(br.a), x(br.b.AddDate(0, 0, 1))
		w := math.Max(8, x1-x0)
		b.rect(x0, y+5, w, rowH-10, 4, ` class="pv-m pv-tlbar"`+paint(tone(d, br.ri), 0))
		if label := clipText(d.st.rows[br.ri].Title, w-10, 11); label != "" && w > 60 {
			b.text(x0+6, y+rowH/2+4, "", "pv-in pv-in--left", label, 11)
		}
	}
	if !t.now.Before(lo) && t.now.Before(hi) {
		tx := x(t.now)
		b.line(tx, top-6, tx, H, "pv-today")
	}
	return `<div class="tplp-viz">` + b.done() + `</div>`
}

func (t *pvTree) mapHTML(d *pvDB, cfg pvView) string {
	p := t.firstOf(d, cfg.Layout.Place, "place")
	if p == nil {
		return ""
	}
	type pin struct {
		x, y  float64
		label string
	}
	var pins []pin
	for _, ri := range t.rowsOf(d, cfg.Filter) {
		m, ok := d.st.rows[ri].Values[p.ID].(map[string]any)
		if !ok {
			continue
		}
		lat, ok1 := num(m["lat"])
		lng, ok2 := num(m["lng"])
		if !ok1 || !ok2 {
			continue
		}
		lat = math.Max(-60, math.Min(84, lat))
		pins = append(pins, pin{(lng + 180) * 2, (84 - lat) * 2, d.st.rows[ri].Title})
	}
	minX, minY, maxX, maxY := 0.0, 0.0, worldW, worldH
	if len(pins) > 0 {
		minX, minY, maxX, maxY = pins[0].x, pins[0].y, pins[0].x, pins[0].y
		for _, q := range pins {
			minX, maxX = math.Min(minX, q.x), math.Max(maxX, q.x)
			minY, maxY = math.Min(minY, q.y), math.Max(maxY, q.y)
		}
	}
	const H = 270.0
	aspect := pvW / H
	w := math.Max(maxX-minX, 80) * 1.3
	h := math.Max(maxY-minY, 36) * 1.35
	if w/h < aspect {
		w = h * aspect
	} else {
		h = w / aspect
	}
	if w > worldW {
		w, h = worldW, worldW/aspect
	}
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	vx := math.Max(0, math.Min(worldW-w, cx-w/2))
	vy := math.Max(-20, math.Min(worldH-h+20, cy-h/2))
	k := w / pvW
	var b strings.Builder
	b.WriteString(`<div class="tplp-viz tplp-map"><svg class="pv" viewBox="` + ff(vx) + ` ` + ff(vy) + ` ` + ff(w) + ` ` + ff(h) + `" data-kind="map" aria-hidden="true"><path class="pv-land" stroke-width="` + ff(0.8*k) + `" d="` + worldPath + `"/>`)
	var placed [][2]float64
	r := 5 * k
	for _, q := range pins {
		b.WriteString(`<circle cx="` + ff(q.x) + `" cy="` + ff(q.y) + `" r="` + ff(r) + `" class="pv-pin" stroke-width="` + ff(2*k) + `"/>`)
	}
	for _, q := range pins {
		free := len(placed) < 8
		for _, p := range placed {
			if math.Abs(p[0]-q.x) < 110*k && math.Abs(p[1]-q.y) < 16*k {
				free = false
			}
		}
		if free {
			placed = append(placed, [2]float64{q.x, q.y})
			label := clipText(q.label, 120, 11)
			x, anchor := q.x+r+3*k, ""
			if x+textW(label, 11)*k > vx+w {
				x, anchor = q.x-r-3*k, ` text-anchor="end"`
			}
			b.WriteString(`<text x="` + ff(x) + `" y="` + ff(q.y+4*k) + `"` + anchor + ` font-size="` + ff(11*k) + `" class="pv-pinlabel" stroke-width="` + ff(3*k) + `">` + esc(label) + `</text>`)
		}
	}
	b.WriteString(`</svg></div>`)
	return b.String()
}

func (t *pvTree) graphHTML(d *pvDB, cfg pvView) string {
	type node struct {
		label  string
		ext    bool
		color  string
		deg    int
		x, y   float64
		dx, dy float64
	}
	var nodes []*node
	index := map[string]int{}
	rows := t.rowsOf(d, cfg.Filter)
	if len(rows) > 40 {
		rows = rows[:40]
	}
	for _, ri := range rows {
		index["r:"+d.st.rows[ri].Title] = len(nodes)
		nodes = append(nodes, &node{label: d.st.rows[ri].Title, color: tone(d, ri)})
	}
	type edge struct{ a, b int }
	var edges []edge
	seen := map[[2]int]bool{}
	for _, ri := range rows {
		a := index["r:"+d.st.rows[ri].Title]
		for _, p := range d.st.schema {
			if p.Type != "relation" {
				continue
			}
			for _, title := range strs(d.st.rows[ri].Values[p.ID]) {
				key := "x:" + p.Collection + ":" + title
				j, ok := index[key]
				if !ok {
					if len(nodes) >= 70 {
						continue
					}
					j = len(nodes)
					index[key] = j
					nodes = append(nodes, &node{label: title, ext: true})
				}
				if j == a || seen[[2]int{a, j}] || seen[[2]int{j, a}] {
					continue
				}
				seen[[2]int{a, j}] = true
				edges = append(edges, edge{a, j})
				nodes[a].deg++
				nodes[j].deg++
			}
		}
	}
	if len(nodes) == 0 {
		return ""
	}
	n := len(nodes)
	for i, nd := range nodes {
		ang := float64(i) * 2.399963
		rad := 10 * math.Sqrt(float64(i)+0.5)
		nd.x, nd.y = rad*math.Cos(ang), rad*math.Sin(ang)
	}
	kk := 26.0
	temp := 30.0
	for it := 0; it < 260; it++ {
		for _, nd := range nodes {
			nd.dx, nd.dy = -nd.x*0.02, -nd.y*0.02
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				a, c := nodes[i], nodes[j]
				dx, dy := a.x-c.x, a.y-c.y
				dist := math.Max(0.5, math.Hypot(dx, dy))
				f := kk * kk / dist
				a.dx += dx / dist * f
				a.dy += dy / dist * f
				c.dx -= dx / dist * f
				c.dy -= dy / dist * f
			}
		}
		for _, e := range edges {
			a, c := nodes[e.a], nodes[e.b]
			dx, dy := a.x-c.x, a.y-c.y
			dist := math.Max(0.5, math.Hypot(dx, dy))
			f := dist * dist / kk
			a.dx -= dx / dist * f
			a.dy -= dy / dist * f
			c.dx += dx / dist * f
			c.dy += dy / dist * f
		}
		for _, nd := range nodes {
			l := math.Max(0.01, math.Hypot(nd.dx, nd.dy))
			m := math.Min(l, temp)
			nd.x += nd.dx / l * m
			nd.y += nd.dy / l * m
		}
		temp = math.Max(1, temp*0.97)
	}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, nd := range nodes {
		minX, maxX = math.Min(minX, nd.x), math.Max(maxX, nd.x)
		minY, maxY = math.Min(minY, nd.y), math.Max(maxY, nd.y)
	}
	const H, padX, padY = 280.0, 70.0, 22.0
	sx := (pvW - 2*padX) / math.Max(1, maxX-minX)
	sy := (H - 2*padY) / math.Max(1, maxY-minY)
	sx = math.Min(sx, sy*2.2)
	ox := (pvW - (maxX-minX)*sx) / 2
	oy := (H - (maxY-minY)*sy) / 2
	pos := func(nd *node) (float64, float64) { return ox + (nd.x-minX)*sx, oy + (nd.y-minY)*sy }
	b := svgOpen(pvW, H, "graph")
	for _, e := range edges {
		x1, y1 := pos(nodes[e.a])
		x2, y2 := pos(nodes[e.b])
		b.line(x1, y1, x2, y2, "pv-edge")
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return nodes[order[i]].deg > nodes[order[j]].deg })
	labelled := map[int]bool{}
	var spots [][2]float64
	for _, j := range order {
		x, y := pos(nodes[j])
		free := len(spots) < 12
		for _, p := range spots {
			if math.Abs(p[0]-x) < 80 && math.Abs(p[1]-y) < 16 {
				free = false
			}
		}
		if free {
			spots = append(spots, [2]float64{x, y})
			labelled[j] = true
		}
	}
	for i, nd := range nodes {
		x, y := pos(nd)
		r := 4 + math.Sqrt(float64(nd.deg))*1.8
		pal := 0
		if nd.ext {
			pal = 1
		}
		b.WriteString(`<circle cx="` + ff(x) + `" cy="` + ff(y) + `" r="` + ff(r) + `" class="pv-m pv-node"` + paint(nd.color, pal) + `/>`)
		if labelled[i] {
			b.text(x, y-r-4, "middle", "pv-nlabel", clipText(nd.label, 110, 10.5), 10.5)
		}
	}
	return `<div class="tplp-viz">` + b.done() + `</div>`
}

func (t *pvTree) cellOf(d *pvDB, ri int, sp prop, now time.Time) string {
	switch sp.Type {
	case "relation":
		var b strings.Builder
		titles := strs(d.st.rows[ri].Values[sp.ID])
		for i, title := range titles {
			if i == 2 {
				b.WriteString(`<span class="tplp-more">+` + strconv.Itoa(len(titles)-2) + `</span>`)
				break
			}
			b.WriteString(`<span class="tplp-chip" data-c="gray">` + esc(clipText(title, 110, 13)) + `</span>`)
		}
		return b.String()
	case "formula", "rollup":
		v := t.value(d, ri, &sp)
		switch v.K {
		case formula.KNumber:
			return esc(pvFmt{format: sp.Format, precision: sp.Precision}.full(v.N))
		case formula.KBool:
			if v.B {
				return `<span class="tplp-check" data-on="true"></span>`
			}
			return `<span class="tplp-check"></span>`
		case formula.KDate:
			return esc(v.T.Format("Jan 2"))
		case formula.KEmpty:
			return ""
		}
		return esc(clipText(v.String(), 160, 13))
	}
	return cellHTML(sp, d.st.rows[ri].Values[sp.ID], ri, now)
}

func excerpt(md string, n int) []string {
	var out []string
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#>-*0123456789. []")
		line = strings.TrimSpace(strings.NewReplacer("**", "", "__", "", "`", "").Replace(line))
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "|") || strings.HasPrefix(line, "<") || strings.HasPrefix(line, "{{") {
			continue
		}
		out = append(out, line)
		if len(out) == n {
			break
		}
	}
	return out
}

func (t *pvTree) galleryHTML(d *pvDB, cfg pvView) string {
	cols := 3
	switch cfg.Layout.Size {
	case "small":
		cols = 4
	case "large":
		cols = 2
	}
	var rest []prop
	for _, sp := range d.st.schema {
		if sp.Type != "title" {
			rest = append(rest, sp)
		}
	}
	rows := t.rowsOf(d, cfg.Filter)
	if len(rows) > cols*2 {
		rows = rows[:cols*2]
	}
	cover := cfg.Layout.Cover
	var b strings.Builder
	b.WriteString(`<div class="tplp-cards" data-cols="` + strconv.Itoa(cols) + `">`)
	for _, ri := range rows {
		r := d.st.rows[ri]
		b.WriteString(`<div class="tplp-card">`)
		if cover != "none" {
			lines := excerpt(r.Markdown, 4)
			c := tone(d, ri)
			attr := ""
			if c != "" {
				attr = ` data-color="` + esc(c) + `"`
			}
			if len(lines) > 0 {
				b.WriteString(`<div class="tplp-card__cover tplp-card__cover--text"` + attr + `>`)
				for _, l := range lines {
					b.WriteString(`<span>` + esc(clipText(l, 260, 10)) + `</span>`)
				}
				b.WriteString(`</div>`)
			} else {
				b.WriteString(`<div class="tplp-card__cover"` + attr + `></div>`)
			}
		}
		b.WriteString(`<div class="tplp-card__title">` + esc(r.Title) + `</div>`)
		b.WriteString(cardProps(rest, r, ri, t.now, 2))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}
