package doc

import (
	"bytes"
	"encoding/xml"
	"html"
	"io"
	"strings"

	"github.com/reearth/ygo/crdt"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var mathTags = map[string]bool{
	"math": true, "semantics": true, "annotation": true, "mrow": true, "mi": true, "mn": true, "mo": true,
	"ms": true, "mtext": true, "mspace": true, "msub": true, "msup": true, "msubsup": true, "mfrac": true,
	"msqrt": true, "mroot": true, "mstyle": true, "mtable": true, "mtr": true, "mtd": true, "mlabeledtr": true,
	"mover": true, "munder": true, "munderover": true, "mpadded": true, "mphantom": true, "menclose": true,
	"merror": true, "mmultiscripts": true, "mprescripts": true, "none": true,
}

var mathAttrs = map[string]bool{
	"mathvariant": true, "displaystyle": true, "scriptlevel": true, "stretchy": true, "fence": true,
	"separator": true, "lspace": true, "rspace": true, "minsize": true, "maxsize": true, "movablelimits": true,
	"accent": true, "accentunder": true, "columnalign": true, "rowalign": true, "columnspacing": true,
	"rowspacing": true, "columnlines": true, "rowlines": true, "frame": true, "framespacing": true,
	"linethickness": true, "width": true, "height": true, "depth": true, "voffset": true, "notation": true,
	"encoding": true, "display": true, "mathcolor": true, "mathbackground": true, "symmetric": true,
	"largeop": true, "form": true, "columnspan": true, "rowspan": true, "align": true, "bevelled": true,
}

func sanitizeMathML(src string) string {
	if !strings.HasPrefix(strings.TrimSpace(src), "<math") || len(src) > 200000 {
		return ""
	}
	dec := xml.NewDecoder(strings.NewReader(src))
	dec.Strict = true
	var out strings.Builder
	skip := 0
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			ok := skip == 0 && mathTags[name] && (depth > 0 || name == "math")
			if ok && name == "annotation" {
				ok = false
				for _, a := range t.Attr {
					if a.Name.Local == "encoding" && a.Value == "application/x-tex" {
						ok = true
					}
				}
			}
			if !ok {
				skip++
				continue
			}
			depth++
			out.WriteString("<" + name)
			if name == "math" {
				out.WriteString(` xmlns="http://www.w3.org/1998/Math/MathML"`)
			}
			for _, a := range t.Attr {
				if a.Name.Space != "" || !mathAttrs[a.Name.Local] || (a.Name.Local == "display" && name != "math") {
					continue
				}
				out.WriteString(" " + a.Name.Local + `="` + html.EscapeString(a.Value) + `"`)
			}
			out.WriteString(">")
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			depth--
			out.WriteString("</" + t.Name.Local + ">")
		case xml.CharData:
			if skip == 0 && depth > 0 {
				out.WriteString(html.EscapeString(string(t)))
			}
		}
	}
	if depth != 0 {
		return ""
	}
	return out.String()
}

var kindMathInline = gast.NewNodeKind("MathInline")
var kindMathBlock = gast.NewNodeKind("MathBlock")

type mathInline struct {
	gast.BaseInline
	tex string
}

func (n *mathInline) Kind() gast.NodeKind { return kindMathInline }

func (n *mathInline) Dump(src []byte, level int) { gast.DumpHelper(n, src, level, nil, nil) }

type mathBlock struct {
	gast.BaseBlock
	tex  []string
	done bool
}

func (n *mathBlock) Kind() gast.NodeKind { return kindMathBlock }

func (n *mathBlock) Dump(src []byte, level int) { gast.DumpHelper(n, src, level, nil, nil) }

func (n *mathBlock) IsRaw() bool { return true }

type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$'} }

func (mathInlineParser) Parse(_ gast.Node, block text.Reader, _ parser.Context) gast.Node {
	line, _ := block.PeekLine()
	if len(line) < 3 || line[0] != '$' {
		return nil
	}
	if line[1] == '$' {
		end := bytes.Index(line[2:], []byte("$$"))
		if end <= 0 {
			return nil
		}
		tex := strings.TrimSpace(string(line[2 : 2+end]))
		if tex == "" {
			return nil
		}
		block.Advance(end + 4)
		return &mathInline{tex: tex}
	}
	if line[1] == ' ' || line[1] == '\t' {
		return nil
	}
	for i := 2; i < len(line); i++ {
		if line[i] == '\n' || line[i] == '\r' {
			return nil
		}
		if line[i] != '$' || line[i-1] == '\\' || line[i-1] == ' ' || line[i-1] == '\t' {
			continue
		}
		if i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9' {
			return nil
		}
		block.Advance(i + 1)
		return &mathInline{tex: string(line[1:i])}
	}
	return nil
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

func (mathBlockParser) Open(_ gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !bytes.HasPrefix(line[pos:], []byte("$$")) {
		return nil, parser.NoChildren
	}
	rest := strings.TrimSpace(string(line[pos+2:]))
	if inner, ok := strings.CutSuffix(rest, "$$"); ok {
		if strings.Contains(inner, "$$") {
			return nil, parser.NoChildren
		}
		reader.AdvanceToEOL()
		n := &mathBlock{done: true}
		if t := strings.TrimSpace(inner); t != "" {
			n.tex = append(n.tex, t)
		}
		return n, parser.NoChildren
	}
	if strings.Contains(rest, "$$") {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()
	n := &mathBlock{}
	if rest != "" {
		n.tex = append(n.tex, rest)
	}
	return n, parser.NoChildren
}

func (mathBlockParser) Continue(node gast.Node, reader text.Reader, _ parser.Context) parser.State {
	n := node.(*mathBlock)
	if n.done {
		return parser.Close
	}
	line, _ := reader.PeekLine()
	s := strings.TrimRight(string(line), "\r\n")
	if t := strings.TrimSpace(s); strings.HasSuffix(t, "$$") {
		if inner := strings.TrimSpace(strings.TrimSuffix(t, "$$")); inner != "" {
			n.tex = append(n.tex, inner)
		}
		reader.AdvanceToEOL()
		return parser.Close
	}
	n.tex = append(n.tex, s)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (mathBlockParser) Close(gast.Node, text.Reader, parser.Context) {}

func (mathBlockParser) CanInterruptParagraph() bool { return true }

func (mathBlockParser) CanAcceptIndentedLine() bool { return false }

type mathExtension struct{}

func (mathExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(util.Prioritized(mathBlockParser{}, 150)),
		parser.WithInlineParsers(util.Prioritized(mathInlineParser{}, 150)),
	)
}

func mathNode(name, tex string) *pmNode {
	n := newNode(name)
	n.attrs["tex"] = tex
	n.attrs["html"] = ""
	return n
}

func (w *htmlWriter) math(el *crdt.YXmlElement, block bool, id string) {
	tex := attrString(el, "tex")
	cache := sanitizeMathML(attrString(el, "html"))
	if block {
		w.open("div", "class", "math-block", "data-block", id, "data-tex", tex, "contenteditable", "false")
	} else {
		w.open("span", "class", "math-inline", "data-tex", tex, "contenteditable", "false")
	}
	switch {
	case cache != "":
		w.b.WriteString(cache)
	case tex != "":
		w.b.WriteString(`<code class="math-src">` + esc(tex) + "</code>")
	}
	if block {
		w.b.WriteString("</div>")
	} else {
		w.b.WriteString("</span>")
	}
}
