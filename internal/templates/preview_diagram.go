package templates

import (
	"html"
	"math"
	"regexp"
	"strings"
)

var (
	mermaidBlock = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">(.*?)</code></pre>`)
	flowArrow    = regexp.MustCompile(`\s*(?:--\s+[^-|>]+?\s+-->|-->|---|-\.->|==>|-\.-|===)\s*(?:\|([^|]*)\|)?\s*`)
	flowNode     = regexp.MustCompile(`^([A-Za-z0-9_]+)\s*(\(\([^)]*\)\)|\([^)]*\)|\[\[[^\]]*\]\]|\[[^\]]*\]|\{[^}]*\}|>[^\]]*\])?`)
	diagramKinds = map[string]string{"sequencediagram": "Sequence diagram", "classdiagram": "Class diagram", "statediagram": "State diagram", "statediagram-v2": "State diagram", "erdiagram": "Entity relationship diagram", "journey": "User journey", "gantt": "Gantt chart", "pie": "Pie chart", "mindmap": "Mind map", "timeline": "Timeline", "quadrantchart": "Quadrant chart", "xychart-beta": "XY chart", "sankey-beta": "Sankey diagram", "gitgraph": "Git graph", "kanban": "Kanban", "radar-beta": "Radar chart", "treemap-beta": "Treemap", "block-beta": "Block diagram", "packet-beta": "Packet diagram", "architecture-beta": "Architecture diagram", "requirementdiagram": "Requirement diagram", "c4context": "C4 diagram"}
)

type fnode struct {
	id, label, shape string
	layer, slot      int
	x, y, w          float64
}

func diagramHTML(out string) string {
	return mermaidBlock.ReplaceAllStringFunc(out, func(m string) string {
		src := html.UnescapeString(mermaidBlock.FindStringSubmatch(m)[1])
		lines := strings.Split(strings.TrimSpace(src), "\n")
		head := strings.Fields(strings.TrimSpace(lines[0]))
		kind := ""
		if len(head) > 0 {
			kind = strings.ToLower(head[0])
		}
		if kind == "flowchart" || kind == "graph" {
			dir := "TD"
			if len(head) > 1 {
				dir = strings.ToUpper(head[1])
			}
			if svg := flowSVG(lines[1:], dir == "LR" || dir == "RL"); svg != "" {
				return `<div class="tplp-diagram">` + svg + `</div>`
			}
		}
		name := diagramKinds[kind]
		if name == "" {
			name = "Diagram"
		}
		var b strings.Builder
		b.WriteString(`<div class="tplp-diagram tplp-diagram--code"><span class="tplp-diagram__kind">` + esc(name) + `</span><pre>`)
		for i, l := range lines[1:] {
			if i == 7 {
				break
			}
			b.WriteString(esc(l) + "\n")
		}
		b.WriteString(`</pre></div>`)
		return b.String()
	})
}

func flowSVG(lines []string, lr bool) string {
	var nodes []*fnode
	byID := map[string]*fnode{}
	type fedge struct {
		a, b  *fnode
		label string
	}
	var edges []fedge
	node := func(seg string) *fnode {
		m := flowNode.FindStringSubmatch(strings.TrimSpace(seg))
		if m == nil {
			return nil
		}
		n := byID[m[1]]
		if n == nil {
			n = &fnode{id: m[1], label: m[1], shape: "rect"}
			byID[m[1]] = n
			nodes = append(nodes, n)
		}
		if def := m[2]; def != "" {
			switch {
			case strings.HasPrefix(def, "(("):
				n.shape, n.label = "circle", def[2:len(def)-2]
			case strings.HasPrefix(def, "("):
				n.shape, n.label = "round", def[1:len(def)-1]
			case strings.HasPrefix(def, "[["):
				n.label = def[2 : len(def)-2]
			case strings.HasPrefix(def, "{"):
				n.shape, n.label = "diamond", def[1:len(def)-1]
			default:
				n.label = def[1 : len(def)-1]
			}
			n.label = strings.Trim(strings.TrimSpace(n.label), `"`)
		}
		return n
	}
	for _, raw := range lines {
		l := strings.TrimSpace(raw)
		if l == "" || strings.HasPrefix(l, "%%") || strings.HasPrefix(l, "classDef") || strings.HasPrefix(l, "class ") || strings.HasPrefix(l, "style") || strings.HasPrefix(l, "subgraph") || l == "end" || strings.HasPrefix(l, "click") || strings.HasPrefix(l, "linkStyle") {
			continue
		}
		idx := flowArrow.FindAllStringSubmatchIndex(l, -1)
		prev := node(l[:firstOr(idx, len(l))])
		for i, m := range idx {
			end := len(l)
			if i+1 < len(idx) {
				end = idx[i+1][0]
			}
			next := node(l[m[1]:end])
			label := ""
			if m[2] >= 0 {
				label = strings.TrimSpace(l[m[2]:m[3]])
			} else if s := strings.TrimSpace(l[m[0]:m[1]]); strings.HasPrefix(s, "-- ") {
				label = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "--"), "-->"))
			}
			if prev != nil && next != nil {
				edges = append(edges, fedge{prev, next, label})
			}
			prev = next
		}
	}
	if len(nodes) < 2 || len(nodes) > 40 {
		return ""
	}
	out := map[*fnode][]*fnode{}
	for _, e := range edges {
		out[e.a] = append(out[e.a], e.b)
	}
	state := map[*fnode]int{}
	back := map[[2]*fnode]bool{}
	var dfs func(n *fnode)
	dfs = func(n *fnode) {
		state[n] = 1
		for _, c := range out[n] {
			if state[c] == 1 {
				back[[2]*fnode{n, c}] = true
			} else if state[c] == 0 {
				dfs(c)
			}
		}
		state[n] = 2
	}
	for _, n := range nodes {
		if state[n] == 0 {
			dfs(n)
		}
	}
	for changed, guard := true, 0; changed && guard < len(nodes)+2; guard++ {
		changed = false
		for _, e := range edges {
			if !back[[2]*fnode{e.a, e.b}] && e.b.layer < e.a.layer+1 {
				e.b.layer = e.a.layer + 1
				changed = true
			}
		}
	}
	layers := 0
	count := map[int]int{}
	for _, n := range nodes {
		n.slot = count[n.layer]
		count[n.layer]++
		layers = max(layers, n.layer+1)
	}
	maxSlots := 0
	for _, c := range count {
		maxSlots = max(maxSlots, c)
	}
	const W = pvW
	var H float64
	gapMain := 0.0
	for _, n := range nodes {
		n.w = math.Min(150, math.Max(54, textW(n.label, 11.5)+22))
	}
	if lr {
		gapMain = W / float64(layers)
		H = math.Max(120, float64(maxSlots)*52+16)
		for _, n := range nodes {
			n.w = math.Min(n.w, gapMain-18)
			n.x = gapMain * (float64(n.layer) + 0.5)
			n.y = H / float64(count[n.layer]) * (float64(n.slot) + 0.5)
		}
	} else {
		H = float64(layers)*64 + 12
		for _, n := range nodes {
			span := W / float64(count[n.layer])
			n.w = math.Min(n.w, span-14)
			n.x = span * (float64(n.slot) + 0.5)
			n.y = 64*float64(n.layer) + 38
		}
	}
	b := svgOpen(W, H, "flowchart")
	half := func(n *fnode) (float64, float64) {
		if n.shape == "diamond" {
			return n.w/2 + 6, 20
		}
		if n.shape == "circle" {
			return 21, 21
		}
		return n.w / 2, 16
	}
	for _, e := range edges {
		x1, y1, x2, y2 := e.a.x, e.a.y, e.b.x, e.b.y
		dx, dy := x2-x1, y2-y1
		d := math.Max(1, math.Hypot(dx, dy))
		ux, uy := dx/d, dy/d
		clip := func(n *fnode, sx, sy float64) float64 {
			hw, hh := half(n)
			tx, ty := math.Inf(1), math.Inf(1)
			if sx != 0 {
				tx = hw / math.Abs(sx)
			}
			if sy != 0 {
				ty = hh / math.Abs(sy)
			}
			return math.Min(tx, ty)
		}
		a := clip(e.a, ux, uy)
		c := clip(e.b, ux, uy) + 2
		sx, sy, ex, ey := x1+ux*a, y1+uy*a, x2-ux*c, y2-uy*c
		cls := "pv-flowedge"
		if back[[2]*fnode{e.a, e.b}] {
			cls += " pv-flowedge--back"
		}
		b.line(sx, sy, ex, ey, cls)
		px, py := -uy, ux
		b.WriteString(`<path class="pv-flowhead" d="M` + ff(ex) + `,` + ff(ey) + `L` + ff(ex-ux*7+px*3.5) + `,` + ff(ey-uy*7+py*3.5) + `L` + ff(ex-ux*7-px*3.5) + `,` + ff(ey-uy*7-py*3.5) + `Z"/>`)
		if e.label != "" {
			mx, my := (sx+ex)/2, (sy+ey)/2
			b.text(mx, my-4, "middle", "pv-flowlabel", clipText(e.label, 120, 10), 10)
		}
	}
	for _, n := range nodes {
		hw, hh := half(n)
		switch n.shape {
		case "diamond":
			b.WriteString(`<path class="pv-flownode" d="M` + ff(n.x) + `,` + ff(n.y-hh) + `L` + ff(n.x+hw) + `,` + ff(n.y) + `L` + ff(n.x) + `,` + ff(n.y+hh) + `L` + ff(n.x-hw) + `,` + ff(n.y) + `Z"/>`)
		case "circle":
			b.WriteString(`<circle class="pv-flownode" cx="` + ff(n.x) + `" cy="` + ff(n.y) + `" r="21"/>`)
		default:
			rx := 6.0
			if n.shape == "round" {
				rx = 16
			}
			b.rect(n.x-hw, n.y-hh, hw*2, hh*2, rx, ` class="pv-flownode"`)
		}
		room := hw*2 - 10
		if n.shape == "circle" {
			room = 40
		}
		b.text(n.x, n.y+4, "middle", "pv-flowtext", clipText(n.label, room, 11), 11)
	}
	return b.done()
}

func firstOr(idx [][]int, n int) int {
	if len(idx) == 0 {
		return n
	}
	return idx[0][0]
}
