package transfer

import (
	"bytes"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

func pdfText(data []byte) (pages []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errPDF
		}
	}()
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for i := 1; i <= r.NumPage() && i <= 2000; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		pages = append(pages, glyphsToText(p.Content().Text))
	}
	return pages, nil
}

type pdfLine struct {
	y    float64
	size float64
	text strings.Builder
}

func glyphsToText(glyphs []pdf.Text) string {
	var lines []*pdfLine
	var cur *pdfLine
	var prev pdf.Text
	for _, g := range glyphs {
		size := g.FontSize
		if size <= 0 {
			size = 10
		}
		if cur == nil || diff(g.Y, cur.y) > size*0.4 {
			cur = &pdfLine{y: g.Y, size: size}
			lines = append(lines, cur)
			prev = pdf.Text{}
		}
		s := g.S
		if s == "\n" {
			s = " "
		}
		if prev.S != "" && prev.W > 0 && s != " " && !strings.HasSuffix(cur.text.String(), " ") && g.X-(prev.X+prev.W) > size*0.18 {
			cur.text.WriteString(" ")
		}
		cur.text.WriteString(s)
		prev = g
	}
	var gaps []float64
	for i := 1; i < len(lines); i++ {
		if g := lines[i-1].y - lines[i].y; g > 0 {
			gaps = append(gaps, g)
		}
	}
	typical := 14.0
	if len(gaps) > 0 {
		sorted := append([]float64(nil), gaps...)
		sort.Float64s(sorted)
		typical = sorted[(len(sorted)-1)/3]
	}
	var paras []string
	var para []string
	for i, l := range lines {
		s := strings.Join(strings.Fields(l.text.String()), " ")
		if i > 0 {
			gap := lines[i-1].y - l.y
			if (gap > typical*1.6 || gap < 0) && len(para) > 0 {
				paras = append(paras, strings.Join(para, " "))
				para = nil
			}
		}
		if s == "" {
			continue
		}
		if n := len(para); n > 0 && strings.HasSuffix(para[n-1], "-") {
			para[n-1] = strings.TrimSuffix(para[n-1], "-") + s
		} else {
			para = append(para, s)
		}
	}
	if len(para) > 0 {
		paras = append(paras, strings.Join(para, " "))
	}
	return strings.Join(paras, "\n\n")
}

func diff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

type pdfError struct{}

func (pdfError) Error() string { return "the PDF could not be read" }

var errPDF = pdfError{}

func importPDF(name string, data []byte, b *Bundle) {
	pages, err := pdfText(data)
	if err != nil {
		b.Warn("The text of " + name + " could not be read: " + err.Error())
		return
	}
	var parts []string
	a := &Asset{Name: name, Data: data}
	parts = append(parts, "["+escapeText(name)+"]("+b.AssetRef(a)+")")
	empty := true
	for i, p := range pages {
		if strings.TrimSpace(p) == "" {
			continue
		}
		empty = false
		if len(pages) > 1 {
			parts = append(parts, "## Page "+itoa(i+1))
		}
		for _, para := range strings.Split(p, "\n\n") {
			parts = append(parts, escapeLead(escapeText(para)))
		}
	}
	if empty {
		b.Warn("No text was found in " + name + "; it may be scanned images. The file is attached to its page.")
	}
	b.Roots = append(b.Roots, &Node{Title: stripExt(name), Body: strings.Join(parts, "\n\n")})
}
