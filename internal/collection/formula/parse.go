package formula

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Error struct {
	Msg string
	Pos int
}

func (e *Error) Error() string { return e.Msg }

func errAt(pos int, format string, args ...any) *Error {
	return &Error{Msg: fmt.Sprintf(format, args...), Pos: pos}
}

type tokKind uint8

const (
	tEOF tokKind = iota
	tNum
	tStr
	tIdent
	tOp
)

type token struct {
	k   tokKind
	s   string
	n   float64
	pos int
}

func lex(src string) ([]token, error) {
	var out []token
	i := 0
	for i < len(src) {
		r, w := utf8.DecodeRuneInString(src[i:])
		switch {
		case unicode.IsSpace(r):
			i += w
		case r >= '0' && r <= '9' || (r == '.' && i+1 < len(src) && src[i+1] >= '0' && src[i+1] <= '9'):
			j := i
			for j < len(src) && (src[j] >= '0' && src[j] <= '9' || src[j] == '.' || src[j] == '_') {
				j++
			}
			if j < len(src) && (src[j] == 'e' || src[j] == 'E') {
				k := j + 1
				if k < len(src) && (src[k] == '+' || src[k] == '-') {
					k++
				}
				if k < len(src) && src[k] >= '0' && src[k] <= '9' {
					j = k
					for j < len(src) && src[j] >= '0' && src[j] <= '9' {
						j++
					}
				}
			}
			f, err := strconv.ParseFloat(strings.ReplaceAll(src[i:j], "_", ""), 64)
			if err != nil {
				return nil, errAt(i, "%q is not a number", src[i:j])
			}
			out = append(out, token{k: tNum, n: f, pos: i})
			i = j
		case r == '"' || r == '\'' || r == '“' || r == '‘':
			closer := r
			if r == '“' {
				closer = '”'
			} else if r == '‘' {
				closer = '’'
			}
			var b strings.Builder
			j := i + w
			closed := false
			for j < len(src) {
				c, cw := utf8.DecodeRuneInString(src[j:])
				if c == '\\' && j+cw < len(src) {
					n, nw := utf8.DecodeRuneInString(src[j+cw:])
					switch n {
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					default:
						b.WriteRune(n)
					}
					j += cw + nw
					continue
				}
				if c == closer || (closer != r && c == r) {
					j += cw
					closed = true
					break
				}
				b.WriteRune(c)
				j += cw
			}
			if !closed {
				return nil, errAt(i, "a text value is missing its closing quote")
			}
			out = append(out, token{k: tStr, s: b.String(), pos: i})
			i = j
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(src) {
				c, cw := utf8.DecodeRuneInString(src[j:])
				if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
					break
				}
				j += cw
			}
			out = append(out, token{k: tIdent, s: src[i:j], pos: i})
			i = j
		default:
			two := ""
			if i+1 < len(src) {
				two = src[i : i+2]
			}
			switch two {
			case "==", "!=", ">=", "<=", "&&", "||":
				out = append(out, token{k: tOp, s: two, pos: i})
				i += 2
				continue
			}
			switch r {
			case '+', '-', '*', '/', '%', '^', '>', '<', '!', '(', ')', '[', ']', ',', '.', '?', ':', '=':
				out = append(out, token{k: tOp, s: string(r), pos: i})
				i += w
			case '≠':
				out = append(out, token{k: tOp, s: "!=", pos: i})
				i += w
			case '≥':
				out = append(out, token{k: tOp, s: ">=", pos: i})
				i += w
			case '≤':
				out = append(out, token{k: tOp, s: "<=", pos: i})
				i += w
			default:
				return nil, errAt(i, "unexpected character %q", r)
			}
		}
	}
	out = append(out, token{k: tEOF, pos: len(src)})
	return out, nil
}

type node interface{ at() int }

type numLit struct {
	v   float64
	pos int
}
type strLit struct {
	v   string
	pos int
}
type boolLit struct {
	v   bool
	pos int
}
type listLit struct {
	items []node
	pos   int
}
type ident struct {
	name string
	pos  int
}
type callNode struct {
	name   string
	args   []node
	pos    int
	method bool
}
type unaryNode struct {
	op  string
	x   node
	pos int
}
type binaryNode struct {
	op   string
	l, r node
	pos  int
}
type condNode struct {
	c, a, b node
	pos     int
}

func (n *numLit) at() int     { return n.pos }
func (n *strLit) at() int     { return n.pos }
func (n *boolLit) at() int    { return n.pos }
func (n *listLit) at() int    { return n.pos }
func (n *ident) at() int      { return n.pos }
func (n *callNode) at() int   { return n.pos }
func (n *unaryNode) at() int  { return n.pos }
func (n *binaryNode) at() int { return n.pos }
func (n *condNode) at() int   { return n.pos }

type parser struct {
	toks  []token
	i     int
	depth int
}

func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) next() token {
	t := p.toks[p.i]
	if p.i < len(p.toks)-1 {
		p.i++
	}
	return t
}

func (p *parser) isOp(s string) bool {
	t := p.peek()
	return t.k == tOp && t.s == s
}

func (p *parser) isWord(s string) bool {
	t := p.peek()
	return t.k == tIdent && strings.EqualFold(t.s, s)
}

func (p *parser) expect(s string) error {
	if !p.isOp(s) {
		t := p.peek()
		if t.k == tEOF {
			return errAt(t.pos, "expected %q before the end", s)
		}
		return errAt(t.pos, "expected %q", s)
	}
	p.next()
	return nil
}

const maxDepth = 200

func parse(src string) (node, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	if p.peek().k == tEOF {
		return nil, errAt(0, "the formula is empty")
	}
	n, err := p.expr(0)
	if err != nil {
		return nil, err
	}
	if t := p.peek(); t.k != tEOF {
		return nil, errAt(t.pos, "unexpected %q", tokText(t))
	}
	return n, nil
}

func tokText(t token) string {
	switch t.k {
	case tNum:
		return FormatNumber(t.n)
	case tStr:
		return `"` + t.s + `"`
	}
	return t.s
}

func binaryPower(p *parser) (string, int, bool) {
	t := p.peek()
	if t.k == tIdent {
		switch strings.ToLower(t.s) {
		case "or":
			return "||", 2, true
		case "and":
			return "&&", 3, true
		}
		return "", 0, false
	}
	if t.k != tOp {
		return "", 0, false
	}
	switch t.s {
	case "?":
		return "?", 1, true
	case "||":
		return "||", 2, true
	case "&&":
		return "&&", 3, true
	case "==", "!=", "=":
		if t.s == "=" {
			return "==", 4, true
		}
		return t.s, 4, true
	case "<", ">", "<=", ">=":
		return t.s, 5, true
	case "+", "-":
		return t.s, 6, true
	case "*", "/", "%":
		return t.s, 7, true
	case "^":
		return "^", 8, true
	}
	return "", 0, false
}

func (p *parser) expr(min int) (node, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxDepth {
		return nil, errAt(p.peek().pos, "the formula nests too deeply")
	}
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		op, power, ok := binaryPower(p)
		if !ok || power < min {
			return left, nil
		}
		t := p.next()
		if op == "?" {
			a, err := p.expr(1)
			if err != nil {
				return nil, err
			}
			if err := p.expect(":"); err != nil {
				return nil, err
			}
			b, err := p.expr(1)
			if err != nil {
				return nil, err
			}
			left = &condNode{c: left, a: a, b: b, pos: t.pos}
			continue
		}
		nextMin := power + 1
		if op == "^" {
			nextMin = power
		}
		right, err := p.expr(nextMin)
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, l: left, r: right, pos: t.pos}
	}
}

func (p *parser) unary() (node, error) {
	t := p.peek()
	if t.k == tOp && (t.s == "-" || t.s == "!" || t.s == "+") {
		p.next()
		x, err := p.expr(9)
		if err != nil {
			return nil, err
		}
		if t.s == "+" {
			return x, nil
		}
		return &unaryNode{op: t.s, x: x, pos: t.pos}, nil
	}
	if t.k == tIdent && strings.EqualFold(t.s, "not") && !(p.toks[p.i+1].k == tOp && p.toks[p.i+1].s == "(") {
		p.next()
		x, err := p.expr(9)
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: "!", x: x, pos: t.pos}, nil
	}
	return p.postfix()
}

func (p *parser) args() ([]node, error) {
	if err := p.expect("("); err != nil {
		return nil, err
	}
	var out []node
	if p.isOp(")") {
		p.next()
		return out, nil
	}
	for {
		a, err := p.expr(1)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
		if p.isOp(",") {
			p.next()
			continue
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return out, nil
	}
}

func (p *parser) postfix() (node, error) {
	n, err := p.primary()
	if err != nil {
		return nil, err
	}
	for p.isOp(".") {
		dot := p.next()
		name := p.next()
		if name.k != tIdent {
			return nil, errAt(dot.pos, "expected a function name after the dot")
		}
		var args []node
		if p.isOp("(") {
			if args, err = p.args(); err != nil {
				return nil, err
			}
		}
		n = &callNode{name: strings.ToLower(name.s), args: append([]node{n}, args...), pos: name.pos, method: true}
	}
	return n, nil
}

func (p *parser) primary() (node, error) {
	t := p.next()
	switch t.k {
	case tNum:
		return &numLit{v: t.n, pos: t.pos}, nil
	case tStr:
		return &strLit{v: t.s, pos: t.pos}, nil
	case tIdent:
		low := strings.ToLower(t.s)
		switch low {
		case "true":
			return &boolLit{v: true, pos: t.pos}, nil
		case "false":
			return &boolLit{v: false, pos: t.pos}, nil
		}
		if p.isOp("(") {
			args, err := p.args()
			if err != nil {
				return nil, err
			}
			return &callNode{name: low, args: args, pos: t.pos}, nil
		}
		return &ident{name: t.s, pos: t.pos}, nil
	case tOp:
		switch t.s {
		case "(":
			n, err := p.expr(0)
			if err != nil {
				return nil, err
			}
			if err := p.expect(")"); err != nil {
				return nil, err
			}
			return n, nil
		case "[":
			l := &listLit{pos: t.pos}
			if p.isOp("]") {
				p.next()
				return l, nil
			}
			for {
				x, err := p.expr(1)
				if err != nil {
					return nil, err
				}
				l.items = append(l.items, x)
				if p.isOp(",") {
					p.next()
					continue
				}
				if err := p.expect("]"); err != nil {
					return nil, err
				}
				return l, nil
			}
		}
		return nil, errAt(t.pos, "unexpected %q", t.s)
	}
	return nil, errAt(t.pos, "the formula ends too early")
}
