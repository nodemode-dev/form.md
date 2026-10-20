package templates

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/collection"
)

type pvFmt struct {
	format    string
	precision int
	pct       bool
	whole     bool
}

func roundTo(n float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(n*p) / p
}

func commas(n float64, digits int) string {
	s := strconv.FormatFloat(roundTo(n, digits), 'f', -1, 64)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if frac != "" {
		b.WriteString("." + frac)
	}
	return b.String()
}

var currency = map[string]string{"dollar": "$", "euro": "€", "pound": "£", "yen": "¥", "rupee": "₹", "won": "₩", "yuan": "¥", "real": "R$", "franc": "CHF ", "ruble": "₽", "lira": "₺", "peso": "$", "rand": "R", "canadian_dollar": "CA$", "australian_dollar": "A$"}

func (f pvFmt) digits(n float64) int {
	if f.precision > 0 {
		return f.precision
	}
	if f.whole || n == math.Trunc(n) {
		return 0
	}
	if math.Abs(n) >= 100 {
		return 0
	}
	return 2
}

func (f pvFmt) full(n float64) string {
	if f.pct {
		return strconv.FormatFloat(roundTo(n*100, 1), 'f', -1, 64) + "%"
	}
	if f.format == "percent" {
		return strconv.FormatFloat(roundTo(n*100, max(f.precision, 1)), 'f', -1, 64) + "%"
	}
	if sym, ok := currency[f.format]; ok {
		if n < 0 {
			return "-" + sym + commas(-n, f.digits(n))
		}
		return sym + commas(n, f.digits(n))
	}
	return commas(n, f.digits(n))
}

func (f pvFmt) short(n float64) string {
	if f.pct || f.format == "percent" {
		return f.full(n)
	}
	a := math.Abs(n)
	unit := ""
	v := n
	switch {
	case a >= 1e9:
		v, unit = n/1e9, "B"
	case a >= 1e6:
		v, unit = n/1e6, "M"
	case a >= 1e4:
		v, unit = n/1e3, "K"
	case a >= 100:
		g := f
		g.precision = 0
		g.whole = true
		return g.full(n)
	default:
		return f.full(n)
	}
	s := strconv.FormatFloat(roundTo(v, 1), 'f', -1, 64) + unit
	if sym, ok := currency[f.format]; ok {
		if strings.HasPrefix(s, "-") {
			return "-" + sym + s[1:]
		}
		return sym + s
	}
	return s
}

func chartFmt(d *pvDB, spec collection.ChartSpec) pvFmt {
	f := pvFmt{pct: strings.HasPrefix(spec.Fn, "percent_")}
	if p := d.byID[spec.Prop]; p != nil && spec.Fn != "" && spec.Fn != "count" && !strings.HasPrefix(spec.Fn, "count") && !strings.Contains(spec.Fn, "checked") {
		f.format, f.precision = p.Format, p.Precision
	}
	return f
}

var fnNames = map[string]string{
	"count": "Count", "count_all": "Count", "count_values": "Values", "count_unique": "Unique values", "count_empty": "Empty", "count_not_empty": "Not empty",
	"percent_empty": "Percent empty", "percent_not_empty": "Percent not empty", "sum": "Sum", "average": "Average", "median": "Median", "min": "Min", "max": "Max",
	"range": "Range", "checked": "Checked", "unchecked": "Unchecked", "percent_checked": "Percent checked", "percent_unchecked": "Percent unchecked",
}

func chartCaption(d *pvDB, spec collection.ChartSpec) string {
	name := fnNames[spec.Fn]
	if name == "" {
		name = "Count"
	}
	if p := d.byID[spec.Prop]; p != nil && spec.Fn != "" && spec.Fn != "count" && spec.Fn != "count_all" {
		return name + " of " + p.Name
	}
	return name
}

func ff(x float64) string {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		x = 0
	}
	return strconv.FormatFloat(roundTo(x, 1), 'f', -1, 64)
}

func textW(s string, size float64) float64 { return float64(utf8.RuneCountInString(s)) * size * 0.56 }

func clipText(s string, room, size float64) string {
	if room <= size {
		return ""
	}
	if textW(s, size) <= room {
		return s
	}
	n := int(room/(size*0.56)) - 1
	if n < 1 {
		return ""
	}
	r := []rune(s)
	if n >= len(r) {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}

func paint(color string, pal int) string {
	if color != "" && color != "default" {
		return ` data-color="` + esc(color) + `"`
	}
	return ` data-p="` + strconv.Itoa(pal%10) + `"`
}

type svgb struct {
	strings.Builder
}

func (b *svgb) text(x, y float64, anchor, cls, s string, size float64) {
	if s == "" {
		return
	}
	b.WriteString(`<text x="` + ff(x) + `" y="` + ff(y) + `"`)
	if anchor != "" && anchor != "start" {
		b.WriteString(` text-anchor="` + anchor + `"`)
	}
	if size > 0 {
		b.WriteString(` font-size="` + ff(size) + `"`)
	}
	b.WriteString(` class="` + cls + `">` + esc(s) + `</text>`)
}

func (b *svgb) rect(x, y, w, h, rx float64, attrs string) {
	if w <= 0 || h <= 0 {
		return
	}
	b.WriteString(`<rect x="` + ff(x) + `" y="` + ff(y) + `" width="` + ff(w) + `" height="` + ff(h) + `"`)
	if rx > 0 {
		b.WriteString(` rx="` + ff(rx) + `"`)
	}
	b.WriteString(attrs + `/>`)
}

func (b *svgb) line(x1, y1, x2, y2 float64, cls string) {
	b.WriteString(`<line x1="` + ff(x1) + `" y1="` + ff(y1) + `" x2="` + ff(x2) + `" y2="` + ff(y2) + `" class="` + cls + `"/>`)
}

func svgOpen(w, h float64, kind string) *svgb {
	b := &svgb{}
	b.WriteString(`<svg class="pv" viewBox="0 0 ` + ff(w) + ` ` + ff(h) + `" data-kind="` + esc(kind) + `" aria-hidden="true">`)
	return b
}

func (b *svgb) done() string {
	b.WriteString(`</svg>`)
	return b.String()
}

func niceTicks(lo, hi float64, n int) []float64 {
	if hi <= lo {
		hi = lo + 1
	}
	span := hi - lo
	raw := span / float64(n)
	step := math.Pow(10, math.Floor(math.Log10(raw)))
	switch e := raw / step; {
	case e >= 7.5:
		step *= 10
	case e >= 3.5:
		step *= 5
	case e >= 1.5:
		step *= 2
	}
	start := math.Floor(lo/step) * step
	end := math.Ceil(hi/step) * step
	var out []float64
	for v := start; v <= end+step/2 && len(out) < 12; v += step {
		out = append(out, roundTo(v, 9))
	}
	return out
}

func linearPath(pts [][2]float64) string {
	var b strings.Builder
	for i, p := range pts {
		if i == 0 {
			b.WriteString("M" + ff(p[0]) + "," + ff(p[1]))
		} else {
			b.WriteString("L" + ff(p[0]) + "," + ff(p[1]))
		}
	}
	return b.String()
}

func smoothPath(pts [][2]float64) string {
	n := len(pts)
	if n < 3 {
		return linearPath(pts)
	}
	dx := make([]float64, n-1)
	sl := make([]float64, n-1)
	for i := 0; i < n-1; i++ {
		dx[i] = pts[i+1][0] - pts[i][0]
		if dx[i] == 0 {
			dx[i] = 1e-6
		}
		sl[i] = (pts[i+1][1] - pts[i][1]) / dx[i]
	}
	m := make([]float64, n)
	m[0], m[n-1] = sl[0], sl[n-2]
	for i := 1; i < n-1; i++ {
		if sl[i-1]*sl[i] <= 0 {
			m[i] = 0
		} else {
			m[i] = 3 * (dx[i-1] + dx[i]) / ((2*dx[i]+dx[i-1])/sl[i-1] + (dx[i]+2*dx[i-1])/sl[i])
		}
	}
	var b strings.Builder
	b.WriteString("M" + ff(pts[0][0]) + "," + ff(pts[0][1]))
	for i := 0; i < n-1; i++ {
		h := dx[i] / 3
		b.WriteString("C" + ff(pts[i][0]+h) + "," + ff(pts[i][1]+m[i]*h) + " " + ff(pts[i+1][0]-h) + "," + ff(pts[i+1][1]-m[i+1]*h) + " " + ff(pts[i+1][0]) + "," + ff(pts[i+1][1]))
	}
	return b.String()
}

func arcPath(cx, cy, r, ir, a0, a1 float64) string {
	if a1-a0 >= math.Pi*2-1e-6 {
		a1 = a0 + math.Pi*2 - 1e-4
	}
	large := "0"
	if a1-a0 > math.Pi {
		large = "1"
	}
	p := func(rr, a float64) string { return ff(cx+rr*math.Sin(a)) + "," + ff(cy-rr*math.Cos(a)) }
	return "M" + p(r, a0) + "A" + ff(r) + "," + ff(r) + " 0 " + large + " 1 " + p(r, a1) + "L" + p(ir, a1) + "A" + ff(ir) + "," + ff(ir) + " 0 " + large + " 0 " + p(ir, a0) + "Z"
}

type swatch struct {
	label, color string
	pal          int
}

func (b *svgb) legend(items []swatch, w float64) float64 {
	if len(items) < 2 {
		return 0
	}
	x := 2.0
	for _, s := range items {
		label := clipText(s.label, 140, 11)
		if label == "" {
			label = "-"
		}
		iw := 16 + textW(label, 11) + 14
		if x+iw > w {
			break
		}
		b.rect(x, 3, 10, 10, 2, ` class="pv-m"`+paint(s.color, s.pal))
		b.text(x+15, 12, "", "pv-tick", label, 0)
		x += iw
	}
	return 24
}

type pvAxes struct {
	left, top, pw, ph, band, d0, d1 float64
	horiz                           bool
}

func (a pvAxes) at(v float64) float64 {
	span := a.d1 - a.d0
	if span == 0 {
		span = 1
	}
	if a.horiz {
		return a.left + (v-a.d0)/span*a.pw
	}
	return a.top + a.ph - (v-a.d0)/span*a.ph
}

func (a pvAxes) px(i int) float64 { return a.left + a.band*(float64(i)+0.5) }

func (b *svgb) axes(xs []pvPoint, lo, hi, top, W, H float64, f pvFmt, horiz bool, rightPad float64) pvAxes {
	ticks := niceTicks(lo, hi, map[bool]int{true: 4, false: 4}[horiz])
	a := pvAxes{top: top, horiz: horiz, d0: ticks[0], d1: ticks[len(ticks)-1]}
	maxChars := 1.0
	for _, x := range xs {
		maxChars = math.Max(maxChars, float64(utf8.RuneCountInString(x.label)))
	}
	if horiz {
		a.left = math.Min(150, math.Max(40, maxChars*6.2+14))
	} else {
		for _, t := range ticks {
			a.left = math.Max(a.left, textW(f.short(t), 11))
		}
		a.left += 12
	}
	right := 10 + rightPad
	bottom := 22.0
	a.pw = math.Max(10, W-a.left-right)
	a.ph = math.Max(10, H-top-bottom)
	n := float64(max(1, len(xs)))
	if horiz {
		a.band = a.ph / n
	} else {
		a.band = a.pw / n
	}
	for _, t := range ticks {
		p := a.at(t)
		cls := "pv-grid"
		if t == 0 {
			cls = "pv-base"
		}
		if horiz {
			b.line(p, top, p, top+a.ph, cls)
			b.text(p, top+a.ph+15, "middle", "pv-tick", f.short(t), 0)
		} else {
			b.line(a.left, p, a.left+a.pw, p, cls)
			b.text(a.left-7, p+4, "end", "pv-tick", f.short(t), 0)
		}
	}
	every := 1
	if horiz {
		every = max(1, int(math.Ceil(14/a.band)))
	} else {
		every = max(1, int(math.Ceil((math.Min(maxChars, 12)*6.2+8)/a.band)))
	}
	for i, x := range xs {
		if i%every != 0 {
			continue
		}
		label := x.label
		if label == "" {
			label = "-"
		}
		if horiz {
			b.text(a.left-7, top+a.band*(float64(i)+0.5)+4, "end", "pv-tick", clipText(label, a.left-10, 11), 0)
		} else {
			b.text(a.px(i), top+a.ph+15, "middle", "pv-tick", clipText(label, a.band*float64(every)-4, 11), 0)
		}
	}
	return a
}

func seriesVal(s pvSeries, i int) float64 {
	if i < len(s.vals) {
		return s.vals[i]
	}
	return 0
}

func drawChart(res *pvResult, spec collection.ChartSpec, f pvFmt, caption string, W, H float64) string {
	if spec.Kind == "calendar" {
		return calendarKindSVG(res, W, H)
	}
	b := svgOpen(W, H, spec.Kind)
	empty := func() string {
		b.text(W/2, H/2+4, "middle", "pv-cap", "No data yet", 13)
		return b.done()
	}
	switch spec.Kind {
	case "number":
		val := f.full(res.value)
		size := math.Min(math.Min(52, math.Max(22, H*0.36)), (W-12)/(float64(utf8.RuneCountInString(val))*0.6))
		b.text(W/2, H/2+size*0.3, "middle", "pv-big", val, size)
		b.text(W/2, H/2+size*0.3+20, "middle", "pv-cap", clipText(caption, W-10, 11), 11)
		return b.done()
	case "gauge":
		drawGauge(b, res.value, spec.Target, f, caption, W, H)
		return b.done()
	case "dots":
		if len(res.dots) == 0 {
			return empty()
		}
		drawDots(b, res, f, W, H)
		return b.done()
	}
	if len(res.x) == 0 || len(res.series) == 0 {
		return empty()
	}
	switch spec.Kind {
	case "donut", "radial":
		drawRound(b, res, spec, f, caption, W, H)
	case "heatmap":
		drawMatrix(b, res, spec, f, W, H)
	case "histogram":
		drawHistogram(b, res, f, W, H)
	case "compare":
		if !drawCompare(b, res, spec, f, W, H) {
			return empty()
		}
	case "line", "area":
		drawLines(b, res, spec, f, W, H)
	default:
		drawBars(b, res, spec, f, W, H)
	}
	return b.done()
}

func legendItems(res *pvResult) []swatch {
	var out []swatch
	for i, s := range res.series {
		out = append(out, swatch{s.label, s.color, i})
	}
	return out
}

func drawBars(b *svgb, res *pvResult, spec collection.ChartSpec, f pvFmt, W, H float64) {
	xs, series := res.x, res.series
	multi := len(series) > 1
	horiz := spec.Kind == "hbar"
	stacked := spec.Stacked && multi
	lo, hi := 0.0, 0.0
	for i := range xs {
		if stacked {
			pos, neg := 0.0, 0.0
			for _, s := range series {
				if v := seriesVal(s, i); v >= 0 {
					pos += v
				} else {
					neg += v
				}
			}
			hi, lo = math.Max(hi, pos), math.Min(lo, neg)
		} else {
			for _, s := range series {
				hi, lo = math.Max(hi, seriesVal(s, i)), math.Min(lo, seriesVal(s, i))
			}
		}
	}
	lh := 0.0
	if multi {
		lh = b.legend(legendItems(res), W)
	}
	top := lh + 8
	if spec.Labels && !horiz {
		top += 12
	}
	right := 0.0
	if spec.Labels && horiz {
		right = 40
	}
	a := b.axes(xs, lo, hi, top, W, H, f, horiz, right)
	inner := a.band * 0.72
	if len(xs) == 1 {
		inner = a.band * 0.4
	}
	slots := len(series)
	if stacked {
		slots = 1
	}
	capW := 56.0
	if horiz {
		capW = 30
	}
	sub := math.Min(inner/float64(slots), capW)
	group := sub * float64(slots)
	for xi, x := range xs {
		pos, neg := 0.0, 0.0
		edge := math.Inf(-1)
		if !horiz {
			edge = math.Inf(1)
		}
		total := 0.0
		for si, s := range series {
			v := seriesVal(s, xi)
			total += v
			start := a.band*float64(xi) + (a.band-group)/2
			if !stacked {
				start += sub * float64(si)
			}
			var lo, hi float64
			if stacked {
				if v >= 0 {
					lo, hi = pos, pos+v
					pos = hi
				} else {
					lo, hi = neg, neg+v
					neg = hi
				}
			} else {
				lo, hi = 0, v
			}
			p0, p1 := a.at(lo), a.at(hi)
			gap := 0.0
			if slots > 1 {
				gap = 2
			}
			var col swatch
			if multi {
				col = swatch{s.label, s.color, si}
			} else {
				pal := 0
				if x.color != "" {
					pal = xi
				}
				col = swatch{x.label, x.color, pal}
			}
			attrs := ` class="pv-m"` + paint(col.color, col.pal)
			if horiz {
				w := math.Abs(p1 - p0)
				b.rect(math.Min(p0, p1), a.top+start+gap/2, w, sub-gap, math.Min(3, w/2), attrs)
				edge = math.Max(edge, math.Max(p0, p1))
				if spec.Labels && !stacked {
					b.text(math.Max(p0, p1)+5, a.top+start+sub/2+4, "", "pv-val", f.short(v), 0)
				}
			} else {
				h := math.Abs(p1 - p0)
				b.rect(a.left+start+gap/2, math.Min(p0, p1), sub-gap, h, math.Min(3, (sub-gap)/2), attrs)
				edge = math.Min(edge, math.Min(p0, p1))
				if spec.Labels && !stacked && v != 0 {
					b.text(a.left+start+sub/2, math.Min(p0, p1)-5, "middle", "pv-val", f.short(v), 0)
				}
			}
		}
		if spec.Labels && stacked && total != 0 {
			if horiz {
				b.text(edge+5, a.top+a.band*(float64(xi)+0.5)+4, "", "pv-val", f.short(total), 0)
			} else {
				b.text(a.px(xi), edge-5, "middle", "pv-val", f.short(total), 0)
			}
		}
	}
}

func drawLines(b *svgb, res *pvResult, spec collection.ChartSpec, f pvFmt, W, H float64) {
	xs, series := res.x, res.series
	multi := len(series) > 1
	stacked := spec.Kind == "area" && spec.Stacked && multi
	base := make([]float64, len(xs))
	type band struct{ lo, hi []float64 }
	bands := make([]band, len(series))
	lo, hi := 0.0, 0.0
	for si, s := range series {
		bands[si] = band{make([]float64, len(xs)), make([]float64, len(xs))}
		for i := range xs {
			b0 := 0.0
			if stacked {
				b0 = base[i]
			}
			t := b0 + seriesVal(s, i)
			if stacked {
				base[i] = t
			}
			bands[si].lo[i], bands[si].hi[i] = b0, t
			lo, hi = math.Min(lo, math.Min(b0, t)), math.Max(hi, math.Max(b0, t))
		}
	}
	lh := 0.0
	if multi {
		lh = b.legend(legendItems(res), W)
	}
	top := lh + 10
	if spec.Labels {
		top += 10
	}
	a := b.axes(xs, lo, hi, top, W, H, f, false, 0)
	path := linearPath
	if spec.Smooth {
		path = smoothPath
	}
	for si, s := range series {
		up := make([][2]float64, len(xs))
		down := make([][2]float64, len(xs))
		for i := range xs {
			up[i] = [2]float64{a.px(i), a.at(bands[si].hi[i])}
			down[len(xs)-1-i] = [2]float64{a.px(i), a.at(bands[si].lo[i])}
		}
		col := paint("", si)
		if multi {
			col = paint(s.color, si)
		}
		if spec.Kind == "area" || !multi {
			back := strings.Replace(path(down), "M", "L", 1)
			cls := "pv-area"
			if spec.Kind == "area" {
				cls = "pv-area pv-area--solid"
			}
			b.WriteString(`<path d="` + path(up) + back + `Z" class="` + cls + `"` + col + `/>`)
		}
		b.WriteString(`<path d="` + path(up) + `" class="pv-line"` + col + `/>`)
		if len(xs) <= 40 {
			for i, p := range up {
				b.WriteString(`<circle cx="` + ff(p[0]) + `" cy="` + ff(p[1]) + `" r="3" class="pv-dot"` + col + `/>`)
				if spec.Labels && !multi {
					b.text(p[0], p[1]-8, "middle", "pv-val", f.short(seriesVal(s, i)), 0)
				}
			}
		}
	}
}

func drawRound(b *svgb, res *pvResult, spec collection.ChartSpec, f pvFmt, caption string, W, H float64) {
	s := res.series[0]
	xs := res.x
	if len(xs) > 12 {
		xs = xs[:12]
	}
	vals := make([]float64, len(xs))
	total := 0.0
	maxV := 0.0
	for i := range xs {
		vals[i] = math.Max(0, seriesVal(s, i))
		total += vals[i]
		maxV = math.Max(maxV, vals[i])
	}
	r := (H - 12) / 2
	legendW := math.Min(230, W*0.42)
	showLegend := W-(2*r+36) >= 110
	if !showLegend {
		legendW = 0
	}
	cx := math.Max(r+8, (W-legendW)/2)
	if !showLegend {
		cx = W / 2
	}
	cy := H / 2
	if spec.Kind == "donut" {
		ir := r * 0.62
		a := 0.0
		for i, v := range vals {
			if v <= 0 || total == 0 {
				continue
			}
			a1 := a + v/total*math.Pi*2
			b.WriteString(`<path d="` + arcPath(cx, cy, r, ir, a, a1) + `" class="pv-m pv-slice"` + paint(xs[i].color, i) + `/>`)
			if spec.Labels && a1-a > 0.3 {
				mid := (a + a1) / 2
				rr := (r + ir) / 2
				b.text(cx+rr*math.Sin(mid), cy-rr*math.Cos(mid)+4, "middle", "pv-in", strconv.Itoa(int(math.Round(v/total*100)))+"%", 0)
			}
			a = a1
		}
		tot := f.short(total)
		size := math.Max(12, math.Min(28, math.Min(ir/2.2, ir*1.7/(float64(utf8.RuneCountInString(tot))*0.58))))
		b.text(cx, cy+2, "middle", "pv-total", tot, size)
		b.text(cx, cy+18, "middle", "pv-cap", clipText(caption, ir*1.6, 10.5), 10.5)
	} else {
		ring := math.Max(4, math.Min(18, (r*0.72)/float64(len(xs))-3))
		for i, v := range vals {
			rr := r - float64(i)*(ring+3)
			b.WriteString(`<path d="` + arcPath(cx, cy, rr, rr-ring, 0, math.Pi*1.5) + `" class="pv-track"/>`)
			if v > 0 && maxV > 0 {
				b.WriteString(`<path d="` + arcPath(cx, cy, rr, rr-ring, 0, math.Pi*1.5*v/maxV) + `" class="pv-m"` + paint(xs[i].color, i) + `/>`)
			}
		}
		b.text(cx, cy+6, "middle", "pv-total", f.short(total), math.Max(14, math.Min(22, r*0.2)))
	}
	if !showLegend {
		return
	}
	lx := cx + r + 28
	rows := math.Min(float64(len(xs)), math.Floor((H-8)/20))
	ly := H/2 - rows*20/2 + 12
	for i := 0; i < int(rows); i++ {
		x := xs[i]
		label := x.label
		if label == "" {
			label = "-"
		}
		b.rect(lx, ly-9, 10, 10, 2, ` class="pv-m"`+paint(x.color, i))
		room := W - lx - 18
		val := f.short(vals[i])
		b.text(lx+16, ly, "", "pv-leg", clipText(label, room-textW(val, 11)-10, 11.5), 11.5)
		b.text(W-2, ly, "end", "pv-tick", val, 0)
		ly += 20
	}
}

func drawGauge(b *svgb, value, target float64, f pvFmt, caption string, W, H float64) {
	top := 0.0
	if target > 0 {
		top = math.Max(target, value)
		if value > target {
			top *= 1.1
		}
	} else {
		t := niceTicks(0, math.Max(1, value), 4)
		top = t[len(t)-1]
	}
	R := math.Max(30, math.Min(W/2-12, H-36))
	cx := W / 2
	cy := math.Min(H-20, H/2+R/2)
	thick := math.Max(10, R*0.2)
	a0 := -math.Pi / 2
	share := math.Max(0, math.Min(1, value/top))
	b.WriteString(`<path d="` + arcPath(cx, cy, R, R-thick, a0, a0+math.Pi) + `" class="pv-track"/>`)
	if share > 0 {
		state := ""
		if target > 0 && value >= target {
			state = ` data-state="met"`
		}
		b.WriteString(`<path d="` + arcPath(cx, cy, R, R-thick, a0, a0+math.Pi*share) + `" class="pv-m pv-gauge" data-p="0"` + state + `/>`)
	}
	if target > 0 {
		ta := a0 + math.Pi*math.Min(1, target/top)
		b.line(cx+(R+4)*math.Sin(ta), cy-(R+4)*math.Cos(ta), cx+(R-thick-4)*math.Sin(ta), cy-(R-thick-4)*math.Cos(ta), "pv-target")
	}
	val := f.full(value)
	if textW(val, 12) > (R-thick)*1.3 {
		val = f.short(value)
	}
	size := math.Max(9, math.Min(math.Min(34, R/3), (R-thick)*1.3/(float64(utf8.RuneCountInString(val))*0.58)))
	b.text(cx, cy-4, "middle", "pv-big", val, size)
	sub := caption
	if target > 0 {
		sub = strconv.Itoa(int(math.Round(value/target*100))) + "% of " + f.short(target)
	}
	b.text(cx, math.Min(H-2, cy+14), "middle", "pv-cap", clipText(sub, W-8, 10.5), 10.5)
}

func heatLevels(vals []float64) func(float64) int {
	m := 0.0
	for _, v := range vals {
		m = math.Max(m, v)
	}
	return func(n float64) int {
		if n <= 0 || m == 0 {
			return 0
		}
		return max(1, min(5, int(math.Ceil(n/m*5))))
	}
}

func drawMatrix(b *svgb, res *pvResult, spec collection.ChartSpec, f pvFmt, W, H float64) {
	rows := res.series
	xs := res.x
	left := 8.0
	if len(rows) > 1 || rows[0].label != "" {
		for _, s := range rows {
			left = math.Max(left, textW(s.label, 11)+12)
		}
		left = math.Min(130, left)
	}
	top, bottom := 6.0, 22.0
	cw := math.Max(4, (W-left-6)/float64(max(1, len(xs))))
	ch := math.Max(8, math.Min(34, (H-top-bottom)/float64(max(1, len(rows)))))
	var all []float64
	for _, s := range rows {
		all = append(all, s.vals...)
	}
	level := heatLevels(all)
	for ri, s := range rows {
		for xi := range xs {
			v := seriesVal(s, xi)
			b.rect(left+float64(xi)*cw+1, top+float64(ri)*ch+1, cw-2, ch-2, math.Min(3, cw/4), ` class="pv-cell" data-l="`+strconv.Itoa(level(v))+`"`)
			if spec.Labels && v > 0 && cw >= 26 && ch >= 16 {
				b.text(left+float64(xi)*cw+cw/2, top+float64(ri)*ch+ch/2+4, "middle", "pv-in", f.short(v), 10)
			}
		}
		if left > 8 {
			b.text(left-6, top+float64(ri)*ch+ch/2+4, "end", "pv-tick", clipText(s.label, left-10, 11), 0)
		}
	}
	maxChars := 1.0
	for _, x := range xs {
		maxChars = math.Max(maxChars, float64(utf8.RuneCountInString(x.label)))
	}
	every := max(1, int(math.Ceil((math.Min(12, maxChars)*6.2+6)/cw)))
	for xi, x := range xs {
		if xi%every == 0 {
			b.text(left+float64(xi)*cw+cw/2, top+float64(len(rows))*ch+15, "middle", "pv-tick", clipText(x.label, cw*float64(every)-4, 11), 0)
		}
	}
}

func drawHistogram(b *svgb, res *pvResult, f pvFmt, W, H float64) {
	s := res.series[0]
	xs := res.x
	hi, total := 0.0, 0.0
	for i := range xs {
		v := seriesVal(s, i)
		hi = math.Max(hi, v)
		total += math.Max(0, v)
	}
	a := b.axes(xs, 0, hi, 10, W, H, f, false, 30)
	run := 0.0
	var pts [][2]float64
	for i := range xs {
		v := math.Max(0, seriesVal(s, i))
		y := a.at(v)
		b.rect(a.left+a.band*float64(i)+0.5, y, a.band-1, a.at(0)-y, 0, ` class="pv-m" data-p="0"`)
		run += v
		share := 0.0
		if total > 0 {
			share = run / total
		}
		pts = append(pts, [2]float64{a.px(i), a.top + a.ph - share*a.ph})
	}
	for _, p := range []float64{0, 50, 100} {
		b.text(a.left+a.pw+6, a.top+a.ph-p/100*a.ph+4, "", "pv-tick", strconv.Itoa(int(p))+"%", 0)
	}
	b.WriteString(`<path d="` + linearPath(pts) + `" class="pv-line pv-line--alt" data-p="3"/>`)
}

func drawCompare(b *svgb, res *pvResult, spec collection.ChartSpec, f pvFmt, W, H float64) bool {
	s := res.series[0]
	n := len(res.x)
	N := n / 2
	if spec.Window > 0 && spec.Window < N {
		N = spec.Window
	}
	if N < 1 {
		return false
	}
	cur := make([]float64, N)
	prev := make([]float64, N)
	sc, sp := 0.0, 0.0
	lo, hi := 0.0, 0.0
	for i := 0; i < N; i++ {
		cur[i] = seriesVal(s, n-N+i)
		prev[i] = seriesVal(s, n-2*N+i)
		sc += cur[i]
		sp += prev[i]
		lo, hi = math.Min(lo, math.Min(cur[i], prev[i])), math.Max(hi, math.Max(cur[i], prev[i]))
	}
	label := "This period · " + f.short(sc)
	if sp != 0 {
		ch := roundTo((sc-sp)/math.Abs(sp)*100, 1)
		sign := ""
		if ch > 0 {
			sign = "+"
		}
		label += " (" + sign + strconv.FormatFloat(ch, 'f', -1, 64) + "%)"
	}
	b.rect(2, 3, 10, 10, 2, ` class="pv-m" data-p="0"`)
	b.text(17, 12, "", "pv-tick", label, 0)
	px := 17 + textW(label, 11) + 16
	b.rect(px, 3, 10, 10, 2, ` class="pv-m pv-prev" data-p="1"`)
	b.text(px+15, 12, "", "pv-tick", "Previous period · "+f.short(sp), 0)
	shown := res.x[n-N:]
	a := b.axes(shown, lo, hi, 34, W, H, f, false, 0)
	path := linearPath
	if spec.Smooth {
		path = smoothPath
	}
	line := func(vals []float64) string {
		pts := make([][2]float64, len(vals))
		for i, v := range vals {
			pts[i] = [2]float64{a.px(i), a.at(v)}
		}
		return path(pts)
	}
	b.WriteString(`<path d="` + line(prev) + `" class="pv-line pv-line--prev" data-p="1"/>`)
	b.WriteString(`<path d="` + line(cur) + `" class="pv-line" data-p="0"/>`)
	return true
}

func drawDots(b *svgb, res *pvResult, f pvFmt, W, H float64) {
	groups := res.x
	if len(groups) == 0 {
		groups = []pvPoint{{}}
	}
	idx := map[string]int{}
	for i, g := range groups {
		idx[g.key] = i
	}
	lo, hi := 0.0, 0.0
	for _, p := range res.dots {
		lo, hi = math.Min(lo, p.v), math.Max(hi, p.v)
	}
	diverging := lo < 0 && hi > 0
	ticks := niceTicks(lo, hi, 5)
	d0, d1 := ticks[0], ticks[len(ticks)-1]
	left := 14.0
	if len(res.x) > 0 {
		for _, g := range groups {
			left = math.Max(left, textW(g.label, 11)+14)
		}
		left = math.Min(140, math.Max(40, left))
	}
	top, bottom, right := 8.0, 22.0, 14.0
	pw, ph := W-left-right, H-top-bottom
	band := ph / float64(len(groups))
	sx := func(v float64) float64 { return left + (v-d0)/(d1-d0)*pw }
	for _, t := range ticks {
		cls := "pv-grid"
		if t == 0 && diverging {
			cls = "pv-base"
		}
		b.line(sx(t), top, sx(t), top+ph, cls)
		b.text(sx(t), top+ph+15, "middle", "pv-tick", f.short(t), 0)
	}
	if len(res.x) > 0 {
		for i, g := range groups {
			b.text(left-7, top+band*(float64(i)+0.5)+4, "end", "pv-tick", clipText(g.label, left-10, 11), 0)
		}
	}
	r := 3.6
	if len(res.dots) > 300 {
		r = 2.4
	}
	pts := append([]pvDot(nil), res.dots...)
	sort.SliceStable(pts, func(i, j int) bool { return pts[i].v < pts[j].v })
	lanes := map[int][][2]float64{}
	for _, p := range pts {
		gi := idx[p.group]
		mid := top + band*(float64(gi)+0.5)
		x := sx(p.v)
		y := mid
		for k := 0; k < 60; k++ {
			off := 0.0
			if k > 0 {
				off = math.Ceil(float64(k)/2) * r * 0.9
				if k%2 == 0 {
					off = -off
				}
			}
			if math.Abs(off) > band/2-r {
				break
			}
			ok := true
			for _, q := range lanes[gi] {
				if (q[0]-x)*(q[0]-x)+(q[1]-mid-off)*(q[1]-mid-off) < 4*r*r {
					ok = false
					break
				}
			}
			if ok {
				y = mid + off
				break
			}
		}
		lanes[gi] = append(lanes[gi], [2]float64{x, y})
		attr := paint("", gi)
		if diverging {
			attr = ` data-sign="pos"`
			if p.v < 0 {
				attr = ` data-sign="neg"`
			}
		}
		b.WriteString(`<circle cx="` + ff(x) + `" cy="` + ff(y) + `" r="` + ff(r) + `" class="pv-m pv-bee"` + attr + `/>`)
	}
}

func dayKeyOf(t time.Time) string { return t.Format("2006-01-02") }

func weekRow(d time.Time) int { return (int(d.Weekday()) + 6) % 7 }

func calStep(from, to time.Time, W, H float64) float64 {
	weeks := math.Ceil(float64(weekRow(from)+int(to.Sub(from).Hours()/24+0.5)) / 7)
	return math.Max(5, math.Min(20, math.Min((W-34)/weeks, (H-20)/7)))
}

func calendarSVG(from, to time.Time, value func(string) float64, W, H float64, kind string) string {
	h := math.Ceil(20 + 7*calStep(from, to, W, H))
	b := svgOpen(W, h, kind)
	calendarCells(b, from, to, value, W, h, true)
	return b.done()
}

func calendarCells(b *svgb, from, to time.Time, value func(string) float64, W, H float64, months bool) {
	var all []float64
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		all = append(all, value(dayKeyOf(d)))
	}
	level := heatLevels(all)
	row := weekRow
	left, top := 30.0, 16.0
	step := calStep(from, to, W, H)
	size := step - math.Max(1.5, step*0.14)
	last := -1
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		off := int(d.Sub(from).Hours()/24+0.5) + row(from)
		wi := off / 7
		x := left + float64(wi)*step
		y := top + float64(row(d))*step
		b.rect(x, y, size, size, math.Min(2.5, size/4), ` class="pv-cell" data-l="`+strconv.Itoa(level(value(dayKeyOf(d))))+`"`)
		if months && int(d.Month()) != last && row(d) <= 3 {
			last = int(d.Month())
			b.text(x, top-5, "", "pv-tick", d.Format("Jan"), 10)
		}
	}
	for i, n := range []string{"Mon", "Wed", "Fri"} {
		b.text(left-5, top+float64(i*2)*step+size*0.8, "end", "pv-tick", n, 9.5)
	}
}

func calendarKindSVG(res *pvResult, W, H float64) string {
	vals := map[string]float64{}
	var last time.Time
	if len(res.series) > 0 {
		for i, x := range res.x {
			vals[x.key] = seriesVal(res.series[0], i)
			if t, err := time.Parse("2006-01-02", x.key); err == nil && t.After(last) {
				last = t
			}
		}
	}
	if last.IsZero() {
		last = time.Now().UTC()
	}
	from := time.Date(last.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	return calendarSVG(from, from.AddDate(1, 0, 0), func(k string) float64 { return vals[k] }, W, H, "calendar")
}
