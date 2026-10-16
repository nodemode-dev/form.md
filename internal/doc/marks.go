package doc

import (
	"encoding/json"
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/reearth/ygo/crdt"
)

type mark struct {
	K string    `json:"k"`
	C string    `json:"c"`
	S float64   `json:"s"`
	P []float64 `json:"p"`
	T string    `json:"t,omitempty"`
}

type marks struct {
	V     int     `json:"v"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Items []mark  `json:"items"`
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func align(el *crdt.YXmlElement) string {
	switch v := attrString(el, "align"); v {
	case "center", "right", "justify":
		return v
	}
	return ""
}

func num(f float64) string { return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64) }

func parseMarks(raw string) *marks {
	if raw == "" {
		return nil
	}
	var m marks
	if json.Unmarshal([]byte(raw), &m) != nil || m.V != 1 || m.W <= 0 || m.H <= 0 || len(m.Items) == 0 {
		return nil
	}
	return &m
}

func marksSVG(raw string) string {
	m := parseMarks(raw)
	if m == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<svg class="media__marks" aria-hidden="true" preserveAspectRatio="xMidYMid meet" viewBox="0 0 ` + num(m.W) + " " + num(m.H) + `">`)
	for _, it := range m.Items {
		if !hexColor.MatchString(it.C) || it.S <= 0 || it.S > 200 {
			continue
		}
		c, s := it.C, num(it.S)
		stroke := `stroke="` + c + `" stroke-width="` + s + `" fill="none" stroke-linecap="round" stroke-linejoin="round"`
		p := func(i int) float64 {
			if i < len(it.P) {
				return it.P[i]
			}
			return 0
		}
		switch it.K {
		case "arrow":
			x1, y1, x2, y2 := p(0), p(1), p(2), p(3)
			a := math.Atan2(y2-y1, x2-x1)
			l := math.Max(10, it.S*4)
			b.WriteString(`<line ` + stroke + ` x1="` + num(x1) + `" y1="` + num(y1) + `" x2="` + num(x2) + `" y2="` + num(y2) + `"></line>`)
			b.WriteString(`<polygon fill="` + c + `" stroke="` + c + `" stroke-width="1" stroke-linejoin="round" points="` +
				num(x2) + "," + num(y2) + " " + num(x2-l*math.Cos(a-0.45)) + "," + num(y2-l*math.Sin(a-0.45)) + " " + num(x2-l*math.Cos(a+0.45)) + "," + num(y2-l*math.Sin(a+0.45)) + `"></polygon>`)
		case "rect":
			x1, y1, x2, y2 := p(0), p(1), p(2), p(3)
			b.WriteString(`<rect ` + stroke + ` x="` + num(math.Min(x1, x2)) + `" y="` + num(math.Min(y1, y2)) + `" width="` + num(math.Abs(x2-x1)) + `" height="` + num(math.Abs(y2-y1)) + `" rx="` + s + `"></rect>`)
		case "ellipse":
			x1, y1, x2, y2 := p(0), p(1), p(2), p(3)
			b.WriteString(`<ellipse ` + stroke + ` cx="` + num((x1+x2)/2) + `" cy="` + num((y1+y2)/2) + `" rx="` + num(math.Abs(x2-x1)/2) + `" ry="` + num(math.Abs(y2-y1)/2) + `"></ellipse>`)
		case "pen":
			var d strings.Builder
			for i := 0; i+1 < len(it.P); i += 2 {
				if i == 0 {
					d.WriteString("M")
				} else {
					d.WriteString("L")
				}
				d.WriteString(num(it.P[i]) + " " + num(it.P[i+1]))
			}
			b.WriteString(`<path ` + stroke + ` d="` + d.String() + `"></path>`)
		case "text":
			outline := "#111111"
			if c == "#111111" {
				outline = "#ffffff"
			}
			b.WriteString(`<text x="` + num(p(0)) + `" y="` + num(p(1)) + `" fill="` + c + `" font-size="` + num(it.S*6) + `" font-family="Inter Variable, system-ui, sans-serif" font-weight="600" paint-order="stroke" stroke="` + outline + `" stroke-width="` + num(it.S*0.6) + `">` + html.EscapeString(it.T) + `</text>`)
		}
	}
	b.WriteString("</svg>")
	return b.String()
}
