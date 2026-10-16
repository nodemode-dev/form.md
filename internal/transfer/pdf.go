package transfer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/policy"
)

var pdfMD = goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.TaskList))

type seg struct {
	text   string
	bold   bool
	italic bool
	code   bool
	strike bool
	link   string
	check  int
	img    string
}

type pdfR struct {
	s      *Service
	p      *policy.Principal
	ctx    context.Context
	pdf    *fpdf.Fpdf
	src    []byte
	origin string
	size   float64
	left   float64
	right  float64
	imgs   int

	lastImg [4]float64
}

const (
	pageMargin = 20.0
	bodySize   = 10.5
)

func (r *pdfR) lh() float64 { return r.size * 0.52 }

func (r *pdfR) font(bold, italic, code, strike bool, size float64) {
	family := "go"
	style := ""
	if code {
		family = "mono"
		if bold {
			style = "B"
		}
	} else {
		if bold {
			style += "B"
		}
		if italic {
			style += "I"
		}
	}
	if strike {
		style += "S"
	}
	r.pdf.SetFont(family, style, size)
}

func (s *Service) exportPDF(ctx context.Context, p *policy.Principal, set *exportSet, origin string, w io.Writer) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("go", "B", gobold.TTF)
	pdf.AddUTF8FontFromBytes("go", "I", goitalic.TTF)
	pdf.AddUTF8FontFromBytes("go", "BI", gobolditalic.TTF)
	pdf.AddUTF8FontFromBytes("mono", "", gomono.TTF)
	pdf.AddUTF8FontFromBytes("mono", "B", gomonobold.TTF)
	pdf.SetMargins(pageMargin, pageMargin, pageMargin)
	pdf.SetAutoPageBreak(true, pageMargin)
	pdf.AliasNbPages("{nb}")
	if len(set.roots) > 0 {
		pdf.SetTitle(orUntitled(set.roots[0].title), true)
	}
	pdf.SetCreator("form.md", true)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("go", "", 8)
		pdf.SetTextColor(140, 140, 140)
		pdf.CellFormat(0, 5, strconv.Itoa(pdf.PageNo())+" / {nb}", "", 0, "C", false, 0, "")
		pdf.SetTextColor(30, 30, 30)
	})
	pw, _ := pdf.GetPageSize()
	r := &pdfR{s: s, p: p, ctx: ctx, pdf: pdf, origin: origin, size: bodySize, left: pageMargin, right: pw - pageMargin}
	depth := map[string]int{}
	for _, n := range set.all {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		d := 0
		if pd, ok := depth[n.parent]; ok {
			d = pd + 1
		}
		depth[n.id] = d
		r.page(n, min(d, 4))
	}
	if pdf.Err() {
		return pdf.Error()
	}
	return pdf.Output(w)
}

func (r *pdfR) page(n *exNode, level int) {
	pdf := r.pdf
	pdf.AddPage()
	pdf.SetTextColor(30, 30, 30)
	title := orUntitled(n.title)
	pdf.Bookmark(title, level, -1)
	pdf.SetFont("go", "B", 20)
	pdf.MultiCell(0, 9, title, "", "L", false)
	if n.desc != "" {
		pdf.SetFont("go", "", 11)
		pdf.SetTextColor(100, 100, 100)
		pdf.MultiCell(0, 6, n.desc, "", "L", false)
		pdf.SetTextColor(30, 30, 30)
	}
	pdf.Ln(3)
	if n.kind == "collection" {
		r.collection(n)
		return
	}
	md, err := r.s.docs.Markdown(r.ctx, r.p, n.id)
	if err != nil || strings.TrimSpace(md) == "" {
		return
	}
	r.src = []byte(md)
	root := pdfMD.Parser().Parse(text.NewReader(r.src))
	r.blocks(root, r.left)
}

func (r *pdfR) collection(n *exNode) {
	view := r.s.firstView(r.ctx, r.p, n.id)
	if view == "" {
		return
	}
	var buf bytes.Buffer
	if _, err := r.s.cols.ExportCSV(r.ctx, r.p, view, &buf); err != nil {
		return
	}
	headers, rows := readCSVBytes(buf.Bytes())
	if len(headers) == 0 {
		return
	}
	cols := min(len(headers), 6)
	table := [][]string{headers[:cols]}
	for i, row := range rows {
		if i >= 300 {
			break
		}
		cells := make([]string, cols)
		copy(cells, row)
		table = append(table, cells)
	}
	r.table(table, r.left)
	if len(rows) > 300 || len(headers) > cols {
		r.pdf.SetFont("go", "I", 9)
		r.pdf.MultiCell(0, 5, "Showing "+strconv.Itoa(min(len(rows), 300))+" of "+strconv.Itoa(len(rows))+" rows and "+strconv.Itoa(cols)+" of "+strconv.Itoa(len(headers))+" columns; the Markdown export has the full table.", "", "L", false)
	}
}

func (r *pdfR) gap(mm float64) {
	r.pdf.SetY(r.pdf.GetY() + mm)
}

func (r *pdfR) blocks(parent ast.Node, left float64) {
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		r.block(n, left)
	}
}

var alertMarker = regexp.MustCompile(`^\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*`)
var summaryText = regexp.MustCompile(`(?is)<summary>(.*?)</summary>`)
var tagStrip = regexp.MustCompile(`<[^>]*>`)

func (r *pdfR) block(n ast.Node, left float64) {
	pdf := r.pdf
	pdf.SetLeftMargin(left)
	switch x := n.(type) {
	case *ast.Heading:
		sizes := map[int]float64{1: 16, 2: 14, 3: 12.5, 4: 11.5, 5: 11, 6: 11}
		r.gap(2)
		pdf.SetX(left)
		old := r.size
		r.size = sizes[x.Level]
		r.inline(x, true)
		r.size = old
		pdf.Ln(r.size*0.52 + 1.5)
	case *ast.Paragraph, *ast.TextBlock:
		pdf.SetX(left)
		r.inline(x, false)
		pdf.Ln(r.lh())
		r.gap(1.6)
	case *ast.ThematicBreak:
		y := pdf.GetY() + 2
		pdf.SetDrawColor(210, 210, 210)
		pdf.Line(left, y, r.right, y)
		pdf.SetY(y + 3)
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		var b strings.Builder
		lines := x.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			b.Write(seg.Value(r.src))
		}
		pdf.SetX(left)
		pdf.SetFont("mono", "", 8.5)
		pdf.SetFillColor(244, 244, 246)
		pdf.MultiCell(0, 4.3, strings.TrimRight(b.String(), "\n"), "", "L", true)
		r.gap(2)
	case *ast.Blockquote:
		startY, startPage := pdf.GetY(), pdf.PageNo()
		callout := false
		if p, ok := x.FirstChild().(*ast.Paragraph); ok && alertMarker.MatchString(r.plain(p)) {
			callout = true
		}
		r.quoteBody(x, left+5, callout)
		endY := pdf.GetY() - 1.6
		if pdf.PageNo() == startPage {
			if callout {
				pdf.SetDrawColor(150, 135, 240)
			} else {
				pdf.SetDrawColor(200, 200, 200)
			}
			pdf.SetLineWidth(0.8)
			pdf.Line(left+1, startY, left+1, endY)
			pdf.SetLineWidth(0.2)
		}
		pdf.SetLeftMargin(left)
	case *ast.List:
		r.list(x, left)
		r.gap(1)
	case *east.Table:
		var rows [][]string
		for row := x.FirstChild(); row != nil; row = row.NextSibling() {
			var cells []string
			for c := row.FirstChild(); c != nil; c = c.NextSibling() {
				cells = append(cells, r.plain(c))
			}
			rows = append(rows, cells)
		}
		r.table(rows, left)
	case *ast.HTMLBlock:
		var b strings.Builder
		lines := x.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			b.Write(seg.Value(r.src))
		}
		raw := b.String()
		if strings.HasPrefix(strings.TrimSpace(raw), "<!-- marks:") {
			r.marks(raw)
			return
		}
		if m := summaryText.FindStringSubmatch(raw); m != nil {
			pdf.SetX(left)
			r.font(true, false, false, false, r.size)
			pdf.Write(r.lh(), strings.TrimSpace(tagStrip.ReplaceAllString(m[1], "")))
			pdf.Ln(r.lh() + 1)
		}
	default:
		if n.HasChildren() {
			r.blocks(n, left)
		}
	}
}

func (r *pdfR) quoteBody(q *ast.Blockquote, left float64, callout bool) {
	first := true
	for c := q.FirstChild(); c != nil; c = c.NextSibling() {
		if p, ok := c.(*ast.Paragraph); ok && first && callout {
			first = false
			segs := r.segments(p, false)
			segs = stripMarker(segs)
			r.pdf.SetLeftMargin(left)
			r.pdf.SetX(left)
			if len(segs) > 0 {
				r.emit(segs)
				r.pdf.Ln(r.lh())
				r.gap(1.6)
			}
			continue
		}
		first = false
		r.block(c, left)
	}
}

func stripMarker(segs []seg) []seg {
	joined := ""
	for _, s := range segs {
		if s.check != 0 || s.img != "" {
			break
		}
		joined += s.text
		if len(joined) > 24 {
			break
		}
	}
	m := alertMarker.FindString(joined)
	cut := len(m)
	for cut > 0 && len(segs) > 0 {
		if len(segs[0].text) <= cut {
			cut -= len(segs[0].text)
			segs = segs[1:]
		} else {
			segs[0].text = segs[0].text[cut:]
			cut = 0
		}
	}
	for len(segs) > 0 && segs[0].text == "\n" {
		segs = segs[1:]
	}
	return segs
}

func (r *pdfR) list(l *ast.List, left float64) {
	pdf := r.pdf
	num := l.Start
	if num == 0 {
		num = 1
	}
	for item := l.FirstChild(); item != nil; item = item.NextSibling() {
		pdf.SetLeftMargin(left)
		pdf.SetX(left)
		r.font(false, false, false, false, r.size)
		marker := "•"
		if l.IsOrdered() {
			marker = strconv.Itoa(num) + "."
			num++
		}
		check := 0
		if fc := item.FirstChild(); fc != nil {
			if cb, ok := fc.FirstChild().(*east.TaskCheckBox); ok {
				check = 1
				if cb.IsChecked {
					check = 2
				}
			}
		}
		y := pdf.GetY()
		if check > 0 {
			bx := left + 0.6
			by := y + r.lh()/2 - 1.6
			pdf.SetDrawColor(120, 120, 120)
			pdf.Rect(bx, by, 3.2, 3.2, "D")
			if check == 2 {
				pdf.Line(bx+0.6, by+1.7, bx+1.4, by+2.6)
				pdf.Line(bx+1.4, by+2.6, bx+2.7, by+0.6)
			}
			pdf.SetX(left + 6)
		} else {
			pdf.CellFormat(6, r.lh(), marker, "", 0, "L", false, 0, "")
		}
		child := item.FirstChild()
		inner := left + 6
		pdf.SetLeftMargin(inner)
		firstDone := false
		for c := child; c != nil; c = c.NextSibling() {
			if !firstDone {
				firstDone = true
				switch tb := c.(type) {
				case *ast.TextBlock, *ast.Paragraph:
					r.inline(tb, false)
					pdf.Ln(r.lh())
					if _, ok := tb.(*ast.Paragraph); ok {
						r.gap(1)
					}
					continue
				}
				pdf.Ln(r.lh())
			}
			r.block(c, inner)
		}
		if !firstDone {
			pdf.Ln(r.lh())
		}
		r.gap(0.6)
	}
	pdf.SetLeftMargin(left)
}

func (r *pdfR) table(rows [][]string, left float64) {
	if len(rows) == 0 {
		return
	}
	pdf := r.pdf
	cols := 0
	for _, row := range rows {
		cols = max(cols, len(row))
	}
	if cols == 0 {
		return
	}
	width := (r.right - left) / float64(cols)
	_, ph := pdf.GetPageSize()
	lh := 4.6
	for i, row := range rows {
		head := i == 0
		r.font(head, false, false, false, 9)
		lines := make([][]string, cols)
		maxLines := 1
		for c := 0; c < cols; c++ {
			txt := ""
			if c < len(row) {
				txt = row[c]
			}
			lines[c] = pdf.SplitText(txt, width-2.4)
			if len(lines[c]) > 12 {
				lines[c] = append(lines[c][:11], "...")
			}
			maxLines = max(maxLines, len(lines[c]))
		}
		h := float64(maxLines)*lh + 2
		y := pdf.GetY()
		if y+h > ph-pageMargin {
			pdf.AddPage()
			y = pdf.GetY()
		}
		for c := 0; c < cols; c++ {
			x := left + float64(c)*width
			pdf.SetDrawColor(215, 215, 220)
			if head {
				pdf.SetFillColor(240, 240, 244)
				pdf.Rect(x, y, width, h, "FD")
			} else {
				pdf.Rect(x, y, width, h, "D")
			}
			for k, l := range lines[c] {
				pdf.SetXY(x+1.2, y+1+float64(k)*lh)
				pdf.CellFormat(width-2.4, lh, l, "", 0, "L", false, 0, "")
			}
		}
		pdf.SetXY(left, y+h)
	}
	r.gap(3)
}

func (r *pdfR) plain(n ast.Node) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(r.src))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteString(" ")
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func (r *pdfR) segments(n ast.Node, bold bool) []seg {
	var out []seg
	var walk func(n ast.Node, s seg)
	walk = func(n ast.Node, s seg) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch t := c.(type) {
			case *ast.Text:
				v := string(t.Segment.Value(r.src))
				x := s
				x.text = v
				out = append(out, x)
				if t.HardLineBreak() {
					out = append(out, seg{text: "\n"})
				} else if t.SoftLineBreak() {
					y := s
					y.text = " "
					out = append(out, y)
				}
			case *ast.String:
				x := s
				x.text = string(t.Value)
				out = append(out, x)
			case *ast.CodeSpan:
				x := s
				x.code = true
				x.text = r.plain(t)
				out = append(out, x)
			case *ast.Emphasis:
				x := s
				if t.Level >= 2 {
					x.bold = true
				} else {
					x.italic = true
				}
				walk(t, x)
			case *east.Strikethrough:
				x := s
				x.strike = true
				walk(t, x)
			case *ast.Link:
				x := s
				x.link = r.href(string(t.Destination))
				walk(t, x)
			case *ast.AutoLink:
				x := s
				u := string(t.URL(r.src))
				x.link, x.text = u, string(t.Label(r.src))
				out = append(out, x)
			case *ast.Image:
				out = append(out, seg{img: string(t.Destination), text: r.plain(t)})
			case *east.TaskCheckBox:
			case *ast.RawHTML:
			default:
				walk(c, s)
			}
		}
	}
	walk(n, seg{bold: bold})
	return out
}

func (r *pdfR) href(dest string) string {
	if strings.HasPrefix(dest, "/") {
		return r.origin + dest
	}
	return dest
}

func (r *pdfR) inline(n ast.Node, bold bool) {
	r.emit(r.segments(n, bold))
}

func (r *pdfR) emit(segs []seg) {
	pdf := r.pdf
	pdf.SetTextColor(30, 30, 30)
	for _, s := range segs {
		if s.img != "" {
			r.image(s.img, s.text)
			continue
		}
		if s.text == "\n" {
			pdf.Ln(r.lh())
			continue
		}
		size := r.size
		if s.code {
			size = r.size - 1
		}
		r.font(s.bold, s.italic, s.code, s.strike, size)
		if s.link != "" {
			pdf.SetTextColor(80, 70, 200)
			pdf.WriteLinkString(r.lh(), s.text, s.link)
			pdf.SetTextColor(30, 30, 30)
			continue
		}
		pdf.Write(r.lh(), s.text)
	}
}

func (r *pdfR) image(dest, alt string) {
	pdf := r.pdf
	id, ok := strings.CutPrefix(dest, "/files/")
	if !ok || r.imgs >= 400 {
		if alt != "" {
			r.font(false, true, false, false, r.size)
			pdf.Write(r.lh(), "["+alt+"]")
		}
		return
	}
	f, rc, err := r.s.files.Open(r.ctx, r.p, id)
	if err != nil {
		return
	}
	data, err := io.ReadAll(io.LimitReader(rc, 30<<20))
	rc.Close()
	if err != nil {
		return
	}
	kind := ""
	switch http.DetectContentType(data) {
	case "image/png":
		kind = "PNG"
	case "image/jpeg":
		kind = "JPG"
	case "image/gif":
		kind = "GIF"
	}
	if kind == "" {
		r.font(false, true, false, false, r.size)
		pdf.Write(r.lh(), "["+orUntitled(f.Name)+"]")
		return
	}
	name := "img" + id
	info := pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: kind}, bytes.NewReader(data))
	if info == nil || pdf.Err() {
		pdf.ClearError()
		return
	}
	r.imgs++
	iw, ih := info.Extent()
	if iw <= 0 || ih <= 0 {
		return
	}
	left, _, _, _ := pdf.GetMargins()
	avail := r.right - left
	w := min(avail, iw)
	h := w * ih / iw
	if h > 140 {
		h = 140
		w = h * iw / ih
	}
	if pdf.GetX() > left+0.5 {
		pdf.Ln(r.lh())
	}
	_, ph := pdf.GetPageSize()
	y := pdf.GetY()
	if y+h > ph-pageMargin {
		pdf.AddPage()
		y = pdf.GetY()
	}
	pdf.ImageOptions(name, left, y, w, h, false, fpdf.ImageOptions{ImageType: kind}, 0, "")
	r.lastImg = [4]float64{left, y, w, h}
	pdf.SetXY(left, y+h+1)
}

func (r *pdfR) marks(raw string) {
	mw, mh, items, ok := doc.MarksComment(raw)
	box := r.lastImg
	r.lastImg = [4]float64{}
	if !ok || box[2] <= 0 || mw <= 0 || mh <= 0 {
		return
	}
	pdf := r.pdf
	sx, sy := box[2]/mw, box[3]/mh
	at := func(x, y float64) (float64, float64) { return box[0] + x*sx, box[1] + y*sy }
	x0, y0 := pdf.GetX(), pdf.GetY()
	for _, m := range items {
		var cr, cg, cb int
		if _, err := fmt.Sscanf(m.C, "#%02x%02x%02x", &cr, &cg, &cb); err != nil {
			continue
		}
		pdf.SetDrawColor(cr, cg, cb)
		pdf.SetFillColor(cr, cg, cb)
		pdf.SetLineWidth(math.Max(0.2, m.S*sx))
		pdf.SetLineCapStyle("round")
		pdf.SetLineJoinStyle("round")
		p := func(i int) float64 {
			if i < len(m.P) {
				return m.P[i]
			}
			return 0
		}
		switch m.K {
		case "arrow":
			x1, y1 := at(p(0), p(1))
			x2, y2 := at(p(2), p(3))
			pdf.Line(x1, y1, x2, y2)
			a := math.Atan2(y2-y1, x2-x1)
			l := math.Max(10, m.S*4) * sx
			pdf.Polygon([]fpdf.PointType{{X: x2, Y: y2}, {X: x2 - l*math.Cos(a-0.45), Y: y2 - l*math.Sin(a-0.45)}, {X: x2 - l*math.Cos(a+0.45), Y: y2 - l*math.Sin(a+0.45)}}, "F")
		case "rect":
			x1, y1 := at(math.Min(p(0), p(2)), math.Min(p(1), p(3)))
			pdf.Rect(x1, y1, math.Abs(p(2)-p(0))*sx, math.Abs(p(3)-p(1))*sy, "D")
		case "ellipse":
			cx, cy := at((p(0)+p(2))/2, (p(1)+p(3))/2)
			pdf.Ellipse(cx, cy, math.Abs(p(2)-p(0))/2*sx, math.Abs(p(3)-p(1))/2*sy, 0, "D")
		case "pen":
			for i := 0; i+3 < len(m.P); i += 2 {
				x1, y1 := at(m.P[i], m.P[i+1])
				x2, y2 := at(m.P[i+2], m.P[i+3])
				pdf.Line(x1, y1, x2, y2)
			}
		case "text":
			x, y := at(p(0), p(1))
			r.font(true, false, false, false, math.Max(4, m.S*6*sy*72/25.4))
			pdf.SetTextColor(cr, cg, cb)
			pdf.Text(x, y, m.T)
			pdf.SetTextColor(30, 30, 30)
		}
	}
	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetXY(x0, y0)
}
