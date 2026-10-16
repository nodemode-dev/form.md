package doc

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/reearth/ygo/crdt"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type FileInfo struct {
	Mime string
	Name string
	Size int64
}

type FileLookup func(id string) (FileInfo, bool)

type pmNode struct {
	name  string
	attrs map[string]any
	kids  []*pmNode
	runs  []pmRun
}

type pmRun struct {
	text  string
	marks map[string]any
	atom  *pmNode
}

var withID = map[string]bool{
	"paragraph": true, "heading": true, "blockquote": true, "callout": true, "toggle": true,
	"bullet_list": true, "ordered_list": true, "task_list": true, "list_item": true, "task_item": true,
	"code_block": true, "horizontal_rule": true, "table": true, "image": true, "video": true, "audio": true,
	"file": true, "bookmark": true, "embed": true, "toc": true, "columns": true, "database": true, "breadcrumb": true,
	"math_block": true, "synced": true, "ai_block": true, "meeting": true, "button": true, "board": true,
}

var md = goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.TaskList, extension.Linkify, mathExtension{}))

func newNode(name string) *pmNode {
	n := &pmNode{name: name, attrs: map[string]any{}}
	if withID[name] {
		n.attrs["id"] = NewBlockID()
	}
	return n
}

type converter struct {
	src   []byte
	files FileLookup
}

func ParseMarkdown(src string, files FileLookup) []*pmNode {
	b := []byte(strings.ReplaceAll(src, "\r\n", "\n"))
	root := md.Parser().Parse(text.NewReader(b))
	c := &converter{src: b, files: files}
	return c.blocks(root)
}

var (
	summaryRE = regexp.MustCompile(`(?is)<summary>(.*?)</summary>`)
	headingRE = regexp.MustCompile(`(?is)^\s*<h([1-3])>(.*?)</h[1-3]>\s*$`)
	tagRE     = regexp.MustCompile(`<[^>]*>`)
	alertRE   = regexp.MustCompile(`^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*`)
	youtubeRE = regexp.MustCompile(`^https://(?:www\.)?(?:youtube\.com/watch\?(?:.*&)?v=|youtu\.be/|youtube\.com/shorts/)([A-Za-z0-9_-]{6,20})`)
	vimeoRE   = regexp.MustCompile(`^https://(?:www\.)?vimeo\.com/(\d{4,12})`)
)

func (c *converter) rawLines(n gast.Node) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(c.src))
	}
	return b.String()
}

func (c *converter) blocks(parent gast.Node) []*pmNode {
	var out []*pmNode
	var stack []*pmNode
	emit := func(n *pmNode) {
		if n == nil {
			return
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			top.kids = append(top.kids, n)
			return
		}
		out = append(out, n)
	}
	skipping := false
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if hb, ok := n.(*gast.HTMLBlock); ok {
			raw := strings.TrimSpace(c.rawLines(hb))
			if hb.HasClosure() {
				raw += string(hb.ClosureLine.Value(c.src))
			}
			lower := strings.ToLower(raw)
			if skipping {
				skipping = lower != "<!-- synced end -->"
				continue
			}
			if strings.HasPrefix(lower, "<!-- synced:") && strings.HasSuffix(lower, " begin -->") {
				skipping = true
			}
			if strings.HasPrefix(lower, "<details") {
				t := newNode("toggle")
				title := newNode("toggle_title")
				summary := ""
				if m := summaryRE.FindStringSubmatch(raw); m != nil {
					summary = m[1]
				}
				if m := headingRE.FindStringSubmatch(summary); m != nil {
					lvl, _ := strconv.Atoi(m[1])
					t.attrs["level"] = lvl
					summary = m[2]
				}
				title.runs = c.inlineHTMLText(summary)
				t.kids = append(t.kids, title)
				stack = append(stack, t)
				if strings.Contains(lower, "</details>") {
					stack = stack[:len(stack)-1]
					emit(t)
				}
				continue
			}
			if strings.HasPrefix(lower, "</details") && len(stack) > 0 {
				t := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				emit(t)
				continue
			}
			emit(c.htmlBlock(raw))
			continue
		}
		if skipping {
			continue
		}
		emit(c.block(n))
	}
	for len(stack) > 0 {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		emit(t)
	}
	return out
}

func (c *converter) inlineHTMLText(s string) []pmRun {
	s = strings.TrimSpace(tagRE.ReplaceAllString(s, ""))
	if s == "" {
		return nil
	}
	return []pmRun{{text: unescapeHTML(s), marks: map[string]any{}}}
}

func unescapeHTML(s string) string {
	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&amp;", "&").Replace(s)
}

func isPageID(s string) bool {
	if len(s) != 26 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z') && !(r >= '2' && r <= '7') {
			return false
		}
	}
	return true
}

func (c *converter) htmlBlock(raw string) *pmNode {
	trimmed := strings.TrimSpace(raw)
	if ref, ok := strings.CutPrefix(trimmed, "<!-- synced:"); ok {
		if ref, ok = strings.CutSuffix(ref, " -->"); ok {
			ref = strings.TrimSuffix(ref, " begin")
			if pg, key, ok := strings.Cut(ref, "/"); ok && isPageID(pg) && SyncKey(key) {
				n := newNode("synced")
				n.attrs["page"] = pg
				n.attrs["key"] = key
				return n
			}
		}
	}
	if enc, ok := strings.CutPrefix(trimmed, "<!-- button:"); ok {
		if enc, ok = strings.CutSuffix(enc, " -->"); ok {
			if n := buttonNode(enc); n != nil {
				return n
			}
		}
	}
	if enc, ok := strings.CutPrefix(trimmed, "<!-- board:"); ok {
		if enc, ok = strings.CutSuffix(enc, " -->"); ok {
			if n := boardNode(enc); n != nil {
				return n
			}
		}
	}
	if id, ok := strings.CutPrefix(trimmed, "<!-- database:"); ok {
		if id, ok = strings.CutSuffix(id, " -->"); ok && isPageID(id) {
			n := newNode("database")
			n.attrs["collection"] = id
			n.attrs["view"] = ""
			return n
		}
	}
	switch strings.ToLower(trimmed) {
	case "<!-- toc -->":
		return newNode("toc")
	case "<!-- breadcrumb -->":
		return newNode("breadcrumb")
	case "<br>", "<br/>", "<br />":
		return newNode("paragraph")
	}
	p := newNode("paragraph")
	if raw != "" {
		p.runs = []pmRun{{text: raw, marks: map[string]any{}}}
	}
	return p
}

func (c *converter) block(n gast.Node) *pmNode {
	switch x := n.(type) {
	case *gast.Paragraph, *gast.TextBlock:
		return c.paragraph(x)
	case *gast.Heading:
		h := newNode("heading")
		h.attrs["level"] = x.Level
		h.runs = c.inline(x)
		return h
	case *gast.ThematicBreak:
		return newNode("horizontal_rule")
	case *gast.FencedCodeBlock:
		cb := newNode("code_block")
		cb.attrs["language"] = strings.ToLower(string(x.Language(c.src)))
		cb.runs = codeRuns(strings.TrimSuffix(c.rawLines(x), "\n"))
		return cb
	case *gast.CodeBlock:
		cb := newNode("code_block")
		cb.attrs["language"] = ""
		cb.runs = codeRuns(strings.TrimSuffix(c.rawLines(x), "\n"))
		return cb
	case *gast.Blockquote:
		return c.quote(x)
	case *mathBlock:
		return mathNode("math_block", strings.Join(x.tex, "\n"))
	case *gast.List:
		return c.list(x)
	case *east.Table:
		return c.table(x)
	}
	if n.HasChildren() {
		p := newNode("paragraph")
		p.runs = c.inline(n)
		return p
	}
	return nil
}

func codeRuns(s string) []pmRun {
	if s == "" {
		return nil
	}
	return []pmRun{{text: s, marks: map[string]any{}}}
}

func (c *converter) paragraph(n gast.Node) *pmNode {
	if n.ChildCount() == 1 {
		switch x := n.FirstChild().(type) {
		case *gast.Image:
			return c.media(string(x.Destination), plainText(x, c.src), string(x.Title))
		case *gast.Link:
			if id := fileID(string(x.Destination)); id != "" && c.files != nil {
				if info, ok := c.files(id); ok {
					f := newNode("file")
					f.attrs["src"] = "f:" + id
					f.attrs["name"] = info.Name
					f.attrs["size"] = info.Size
					f.attrs["mime"] = info.Mime
					return f
				}
			}
		case *gast.AutoLink:
			if x.AutoLinkType == gast.AutoLinkURL {
				return linkBlock(string(x.URL(c.src)))
			}
		}
	}
	p := newNode("paragraph")
	p.runs = c.inline(n)
	return p
}

func linkBlock(url string) *pmNode {
	if p := EmbedProvider(url); p != "" {
		e := newNode("embed")
		e.attrs["url"] = url
		e.attrs["provider"] = p
		return e
	}
	b := newNode("bookmark")
	b.attrs["url"] = url
	return b
}

func (c *converter) media(dest, alt, title string) *pmNode {
	kind := "image"
	src := dest
	if id := fileID(dest); id != "" {
		src = "f:" + id
		if c.files != nil {
			if info, ok := c.files(id); ok {
				switch {
				case strings.HasPrefix(info.Mime, "video/"):
					kind = "video"
				case strings.HasPrefix(info.Mime, "audio/"):
					kind = "audio"
				case !strings.HasPrefix(info.Mime, "image/"):
					f := newNode("file")
					f.attrs["src"] = src
					f.attrs["name"] = info.Name
					f.attrs["size"] = info.Size
					f.attrs["mime"] = info.Mime
					return f
				}
			}
		}
	}
	n := newNode(kind)
	n.attrs["src"] = src
	if kind == "image" {
		n.attrs["alt"] = alt
		n.attrs["caption"] = title
	} else if alt != kind {
		n.attrs["caption"] = alt
	}
	return n
}

func plainText(n gast.Node, src []byte) string {
	var b strings.Builder
	_ = gast.Walk(n, func(c gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		switch x := c.(type) {
		case *gast.Text:
			b.Write(x.Value(src))
		case *gast.String:
			b.Write(x.Value)
		}
		return gast.WalkContinue, nil
	})
	return b.String()
}

func (c *converter) quote(n *gast.Blockquote) *pmNode {
	kids := c.blocks(n)
	lead := ""
	if len(kids) > 0 && kids[0].name == "paragraph" {
		for _, r := range kids[0].runs {
			if r.atom != nil || len(lead) > 24 {
				break
			}
			lead += r.text
		}
	}
	if lead != "" {
		if m := alertRE.FindStringSubmatch(lead); m != nil {
			callout := newNode("callout")
			callout.attrs["tone"] = map[string]string{"NOTE": "", "TIP": "green", "IMPORTANT": "violet", "WARNING": "amber", "CAUTION": "rose"}[m[1]]
			first := kids[0]
			for cut := len(m[0]); cut > 0 && len(first.runs) > 0 && first.runs[0].atom == nil; {
				if len(first.runs[0].text) <= cut {
					cut -= len(first.runs[0].text)
					first.runs = first.runs[1:]
				} else {
					first.runs[0].text = first.runs[0].text[cut:]
					cut = 0
				}
			}
			for len(first.runs) > 0 && first.runs[0].atom == nil && strings.TrimSpace(first.runs[0].text) == "" {
				first.runs = first.runs[1:]
			}
			if len(first.runs) > 0 && first.runs[0].atom != nil && first.runs[0].atom.name == "hard_break" {
				first.runs = first.runs[1:]
			}
			if len(first.runs) == 0 {
				kids = kids[1:]
			}
			if len(kids) == 0 {
				kids = []*pmNode{newNode("paragraph")}
			}
			callout.kids = kids
			return callout
		}
	}
	q := newNode("blockquote")
	if len(kids) == 0 {
		kids = []*pmNode{newNode("paragraph")}
	}
	q.kids = kids
	return q
}

func (c *converter) list(n *gast.List) *pmNode {
	task := false
	for it := n.FirstChild(); it != nil; it = it.NextSibling() {
		if first := it.FirstChild(); first != nil {
			if _, ok := first.FirstChild().(*east.TaskCheckBox); ok {
				task = true
			}
		}
	}
	var l *pmNode
	switch {
	case task:
		l = newNode("task_list")
	case n.IsOrdered():
		l = newNode("ordered_list")
		l.attrs["start"] = n.Start
	default:
		l = newNode("bullet_list")
	}
	for it := n.FirstChild(); it != nil; it = it.NextSibling() {
		item := newNode("list_item")
		if task {
			item = newNode("task_item")
			checked := false
			if first := it.FirstChild(); first != nil {
				if cb, ok := first.FirstChild().(*east.TaskCheckBox); ok {
					checked = cb.IsChecked
				}
			}
			item.attrs["checked"] = checked
		}
		kids := c.blocks(it)
		if len(kids) == 0 || kids[0].name != "paragraph" {
			kids = append([]*pmNode{newNode("paragraph")}, kids...)
		}
		item.kids = kids
		l.kids = append(l.kids, item)
	}
	return l
}

func (c *converter) table(n *east.Table) *pmNode {
	t := newNode("table")
	for r := n.FirstChild(); r != nil; r = r.NextSibling() {
		row := newNode("table_row")
		_, header := r.(*east.TableHeader)
		for cell := r.FirstChild(); cell != nil; cell = cell.NextSibling() {
			name := "table_cell"
			if header {
				name = "table_header"
			}
			tc := newNode(name)
			p := newNode("paragraph")
			p.runs = c.inline(cell)
			tc.kids = []*pmNode{p}
			row.kids = append(row.kids, tc)
		}
		if len(row.kids) > 0 {
			t.kids = append(t.kids, row)
		}
	}
	return t
}

func copyMarks(m map[string]any) map[string]any {
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (c *converter) inline(parent gast.Node) []pmRun {
	var runs []pmRun
	underline := false
	var walk func(n gast.Node, marks map[string]any)
	add := func(s string, marks map[string]any) {
		if s == "" {
			return
		}
		m := copyMarks(marks)
		if underline {
			m["underline"] = map[string]any{}
		}
		runs = append(runs, pmRun{text: s, marks: m})
	}
	walk = func(n gast.Node, marks map[string]any) {
		for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
			switch x := ch.(type) {
			case *gast.Text:
				add(string(x.Value(c.src)), marks)
				if x.HardLineBreak() {
					runs = append(runs, pmRun{atom: &pmNode{name: "hard_break", attrs: map[string]any{}}})
				} else if x.SoftLineBreak() {
					add(" ", marks)
				}
			case *gast.String:
				add(string(x.Value), marks)
			case *gast.CodeSpan:
				m := copyMarks(marks)
				m["code"] = map[string]any{}
				add(plainText(x, c.src), m)
			case *mathInline:
				runs = append(runs, pmRun{atom: mathNode("math_inline", x.tex)})
			case *gast.Emphasis:
				m := copyMarks(marks)
				if x.Level >= 2 {
					m["strong"] = map[string]any{}
				} else {
					m["em"] = map[string]any{}
				}
				walk(x, m)
			case *east.Strikethrough:
				m := copyMarks(marks)
				m["strike"] = map[string]any{}
				walk(x, m)
			case *gast.Link:
				dest := string(x.Destination)
				if id, ok := strings.CutPrefix(dest, "/p/"); ok && id != "" && !strings.ContainsAny(id, "/?#") {
					runs = append(runs, pmRun{atom: &pmNode{name: "mention", attrs: map[string]any{"kind": "page", "ref": id, "label": plainText(x, c.src)}}})
					continue
				}
				m := copyMarks(marks)
				m["link"] = map[string]any{"href": safeHref(dest)}
				walk(x, m)
			case *gast.AutoLink:
				url := string(x.URL(c.src))
				m := copyMarks(marks)
				m["link"] = map[string]any{"href": safeHref(url)}
				add(string(x.Label(c.src)), m)
			case *gast.Image:
				add(plainText(x, c.src), marks)
			case *gast.RawHTML:
				var raw strings.Builder
				for i := 0; i < x.Segments.Len(); i++ {
					seg := x.Segments.At(i)
					raw.Write(seg.Value(c.src))
				}
				switch strings.ToLower(raw.String()) {
				case "<u>":
					underline = true
				case "</u>":
					underline = false
				case "<br>", "<br/>", "<br />":
					runs = append(runs, pmRun{atom: &pmNode{name: "hard_break", attrs: map[string]any{}}})
				default:
					add(raw.String(), marks)
				}
			case *east.TaskCheckBox:
			default:
				walk(ch, marks)
			}
		}
	}
	walk(parent, map[string]any{})
	for len(runs) > 0 && runs[0].atom == nil && strings.TrimSpace(runs[0].text) == "" {
		runs = runs[1:]
	}
	if len(runs) > 0 && runs[0].atom == nil {
		runs[0].text = strings.TrimLeft(runs[0].text, " ")
	}
	return runs
}

func safeHref(h string) string {
	l := strings.ToLower(strings.TrimSpace(h))
	for _, p := range []string{"http://", "https://", "mailto:", "/p/", "/files/", "#"} {
		if strings.HasPrefix(l, p) {
			return strings.TrimSpace(h)
		}
	}
	if !strings.Contains(l, ":") {
		return strings.TrimSpace(h)
	}
	return ""
}

type container interface {
	InsertElement(txn *crdt.Transaction, index int, elem *crdt.YXmlElement)
	InsertText(txn *crdt.Transaction, index int, txt *crdt.YXmlText)
}

func writeNodes(txn *crdt.Transaction, parent container, index int, nodes []*pmNode) int {
	for _, n := range nodes {
		el := crdt.NewYXmlElement(n.name)
		parent.InsertElement(txn, index, el)
		index++
		writeNode(txn, el, n)
	}
	return index
}

func writeNode(txn *crdt.Transaction, el *crdt.YXmlElement, n *pmNode) {
	for k, v := range n.attrs {
		if v == nil {
			continue
		}
		switch x := v.(type) {
		case string:
			if x == "" && k != "id" {
				continue
			}
		case int:
			v = float64(x)
		case int64:
			v = float64(x)
		}
		el.SetAttributeValue(txn, k, v)
	}
	if len(n.kids) > 0 {
		writeNodes(txn, el, 0, n.kids)
		return
	}
	idx := 0
	var group []pmRun
	flush := func() {
		if len(group) == 0 {
			return
		}
		var all strings.Builder
		for _, r := range group {
			all.WriteString(r.text)
		}
		t := crdt.NewYXmlText()
		el.InsertText(txn, idx, t)
		idx++
		t.Insert(txn, 0, all.String(), nil)
		pos := 0
		for _, r := range group {
			n := utf16Len(r.text)
			if len(r.marks) > 0 {
				t.Format(txn, pos, n, crdt.Attributes(copyMarks(r.marks)))
			}
			pos += n
		}
		group = nil
	}
	for _, r := range n.runs {
		if r.atom != nil {
			flush()
			atom := crdt.NewYXmlElement(r.atom.name)
			el.InsertElement(txn, idx, atom)
			idx++
			for k, v := range r.atom.attrs {
				if s, ok := v.(string); ok && s == "" {
					continue
				}
				atom.SetAttributeValue(txn, k, v)
			}
			continue
		}
		if r.text != "" {
			group = append(group, r)
		}
	}
	flush()
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}
