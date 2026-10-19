package transfer

import (
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type htmlOpts struct {
	link    func(href string) string
	media   func(src, alt string) (dest string, image bool)
	enMedia func(hash, mime string) (dest string, image bool)
}

type htmlConv struct {
	o    htmlOpts
	warn func(string)
}

var skipTags = map[string]bool{"script": true, "style": true, "head": true, "title": true, "noscript": true, "template": true, "svg": true, "iframe": true, "object": true, "en-crypt": true, "button": true, "select": true, "nav": true}

var blockTags = map[string]bool{
	"p": true, "div": true, "section": true, "article": true, "main": true, "header": true, "footer": true, "body": true, "html": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "ul": true, "ol": true, "li": true, "blockquote": true,
	"pre": true, "table": true, "hr": true, "figure": true, "details": true, "aside": true, "en-note": true, "center": true,
	"dl": true, "dt": true, "dd": true, "form": true, "fieldset": true, "address": true, "tbody": true, "thead": true, "tr": true,
}

func htmlToMarkdown(root *html.Node, o htmlOpts, warn func(string)) string {
	c := &htmlConv{o: o, warn: warn}
	return joinBlocks(c.container(root))
}

func joinBlocks(blocks []string) string {
	var b strings.Builder
	prevItem := ""
	for _, s := range blocks {
		s = strings.Trim(s, "\n")
		if strings.TrimSpace(s) == "" {
			continue
		}
		item := ""
		if isListLine(s) {
			item = "list"
			if strings.HasPrefix(s, "- [") {
				item = "task"
			}
		}
		if b.Len() > 0 {
			if item != "" && item == prevItem {
				b.WriteString("\n")
			} else {
				b.WriteString("\n\n")
			}
		}
		if item == "list" && prevItem == "task" && strings.HasPrefix(s, "- ") {
			lines := strings.Split(s, "\n")
			for i, l := range lines {
				if strings.HasPrefix(l, "- ") {
					lines[i] = "* " + l[2:]
				}
			}
			s = strings.Join(lines, "\n")
		}
		b.WriteString(s)
		prevItem = item
	}
	return b.String()
}

func isListLine(s string) bool {
	return strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") || (len(s) > 2 && s[0] >= '0' && s[0] <= '9' && strings.Contains(s[:min(len(s), 6)], ". "))
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func tag(n *html.Node) string {
	if n.Type != html.ElementNode {
		return ""
	}
	return strings.ToLower(n.Data)
}

func isBlock(n *html.Node) bool {
	t := tag(n)
	if t == "" {
		return false
	}
	if blockTags[t] {
		return true
	}
	if t == "img" || t == "en-media" {
		return false
	}
	return false
}

func hasBlockChild(n *html.Node) bool {
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if isBlock(k) || (tag(k) == "en-todo" && k == firstMeaningful(n)) {
			return true
		}
	}
	return false
}

func firstMeaningful(n *html.Node) *html.Node {
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.TextNode && strings.TrimSpace(k.Data) == "" {
			continue
		}
		if k.Type == html.CommentNode {
			continue
		}
		return k
	}
	return nil
}

func (c *htmlConv) container(n *html.Node) []string {
	var out []string
	var buf []*html.Node
	flush := func() {
		if len(buf) == 0 {
			return
		}
		out = append(out, c.paragraph(buf)...)
		buf = nil
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.CommentNode || k.Type == html.DoctypeNode {
			continue
		}
		t := tag(k)
		if skipTags[t] {
			continue
		}
		if isBlock(k) {
			flush()
			out = append(out, c.block(k)...)
			continue
		}
		buf = append(buf, k)
	}
	flush()
	return out
}

func (c *htmlConv) paragraph(nodes []*html.Node) []string {
	var out []string
	var line strings.Builder
	task := ""
	emit := func() {
		s := strings.TrimSpace(line.String())
		s = strings.TrimSuffix(strings.TrimSuffix(s, "\\"), "\\\n")
		line.Reset()
		if s == "" && task == "" {
			return
		}
		if task != "" {
			out = append(out, task+s)
			task = ""
			return
		}
		out = append(out, escapeLead(s))
	}
	for i, k := range nodes {
		if tag(k) == "en-todo" && strings.TrimSpace(line.String()) == "" {
			if strings.EqualFold(attr(k, "checked"), "true") {
				task = "- [x] "
			} else {
				task = "- [ ] "
			}
			line.WriteString(c.inlineChildren(k))
			continue
		}
		if tag(k) == "img" || tag(k) == "en-media" {
			if md, block := c.mediaMD(k); md != "" {
				if block && strings.TrimSpace(line.String()) == "" && (i == len(nodes)-1 || onlySpaceAfter(nodes[i+1:])) {
					emit()
					out = append(out, md)
					continue
				}
				line.WriteString(md)
			}
			continue
		}
		line.WriteString(c.inline(k))
	}
	emit()
	return out
}

func onlySpaceAfter(nodes []*html.Node) bool {
	for _, k := range nodes {
		if k.Type == html.TextNode && strings.TrimSpace(k.Data) == "" {
			continue
		}
		if tag(k) == "br" {
			continue
		}
		return false
	}
	return true
}

func escapeLead(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '#', '>', '-', '+', '=', '|':
		return "\\" + s
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) && (s[i] == '.' || s[i] == ')') {
		return s[:i] + "\\" + s[i:]
	}
	return s
}

func headingLevel(t string) int {
	if len(t) == 2 && t[0] == 'h' && t[1] >= '1' && t[1] <= '6' {
		return int(t[1] - '0')
	}
	return 0
}

func (c *htmlConv) block(n *html.Node) []string {
	t := tag(n)
	if lvl := headingLevel(t); lvl > 0 {
		s := oneLine(c.inlineChildren(n))
		if s == "" {
			return nil
		}
		return []string{strings.Repeat("#", lvl) + " " + s}
	}
	switch t {
	case "hr":
		return []string{"---"}
	case "ul", "ol":
		return []string{c.list(n, t == "ol", attr(n, "start"))}
	case "li":
		return []string{c.items([]*html.Node{n}, false, "")}
	case "blockquote":
		inner := joinBlocks(c.container(n))
		if inner == "" {
			return nil
		}
		return []string{prefixAll(inner, "> ")}
	case "aside":
		inner := joinBlocks(c.container(n))
		if inner == "" {
			return nil
		}
		return []string{prefixAll("[!NOTE]\n"+inner, "> ")}
	case "pre":
		return []string{fence(textContent(n), codeLang(n))}
	case "table":
		return c.table(n)
	case "figure":
		var out []string
		caption := ""
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if tag(k) == "figcaption" {
				caption = oneLine(c.inlineChildren(k))
			}
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if tag(k) == "figcaption" {
				continue
			}
			if tag(k) == "img" {
				if md, _ := c.mediaMDCaption(k, caption); md != "" {
					out = append(out, md)
				}
				continue
			}
			if isBlock(k) {
				out = append(out, c.block(k)...)
			} else if s := strings.TrimSpace(c.inline(k)); s != "" {
				out = append(out, s)
			}
		}
		return out
	case "details":
		summary := ""
		var body []string
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if tag(k) == "summary" {
				summary = oneLine(c.inlineChildren(k))
				continue
			}
			if isBlock(k) {
				body = append(body, c.block(k)...)
			} else if s := strings.TrimSpace(c.inline(k)); s != "" {
				body = append(body, s)
			}
		}
		return []string{"<details>\n<summary>" + summary + "</summary>\n\n" + joinBlocks(body) + "\n\n</details>"}
	case "dl":
		var out []string
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			switch tag(k) {
			case "dt":
				if s := oneLine(c.inlineChildren(k)); s != "" {
					out = append(out, "**"+s+"**")
				}
			case "dd":
				out = append(out, c.container(k)...)
			}
		}
		return out
	case "tbody", "thead", "tr":
		return c.container(n)
	}
	if strings.EqualFold(attr(n, "data-type"), "callout") || strings.Contains(attr(n, "class"), "callout") || strings.Contains(attr(n, "class"), "information-macro") {
		inner := joinBlocks(c.container(n))
		if inner != "" {
			return []string{prefixAll("[!NOTE]\n"+inner, "> ")}
		}
	}
	if !hasBlockChild(n) {
		var nodes []*html.Node
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if k.Type != html.CommentNode && !skipTags[tag(k)] {
				nodes = append(nodes, k)
			}
		}
		return c.paragraph(nodes)
	}
	return c.container(n)
}

func prefixAll(s, p string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l == "" {
			lines[i] = strings.TrimRight(p, " ")
		} else {
			lines[i] = p + l
		}
	}
	return strings.Join(lines, "\n")
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\\\n", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func codeLang(n *html.Node) string {
	find := func(cls string) string {
		for _, f := range strings.Fields(cls) {
			for _, p := range []string{"language-", "lang-", "brush:"} {
				if v, ok := strings.CutPrefix(f, p); ok {
					return strings.Trim(v, ";")
				}
			}
		}
		return ""
	}
	if l := find(attr(n, "class")); l != "" {
		return l
	}
	if l := attr(n, "data-language"); l != "" {
		return l
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if tag(k) == "code" {
			return find(attr(k, "class"))
		}
	}
	return ""
}

func fence(code, lang string) string {
	code = strings.TrimRight(code, "\n")
	f := "```"
	for strings.Contains(code, f) {
		f += "`"
	}
	return f + strings.ToLower(lang) + "\n" + code + "\n" + f
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		if tag(n) == "br" {
			b.WriteString("\n")
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
		if t := tag(n); t == "div" || t == "p" {
			if !strings.HasSuffix(b.String(), "\n") {
				b.WriteString("\n")
			}
		}
	}
	walk(n)
	return b.String()
}

func (c *htmlConv) list(n *html.Node, ordered bool, start string) string {
	var kids []*html.Node
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		kids = append(kids, k)
	}
	return c.items(kids, ordered, start)
}

func (c *htmlConv) items(kids []*html.Node, ordered bool, start string) string {
	var items []string
	num := 1
	if ordered && start != "" {
		if v := atoiSmall(start); v > 0 {
			num = v
		}
	}
	for _, k := range kids {
		t := tag(k)
		if t == "ul" || t == "ol" {
			if len(items) > 0 {
				nested := c.list(k, t == "ol", attr(k, "start"))
				items[len(items)-1] += "\n" + indent(nested, "  ")
			} else {
				items = append(items, c.list(k, t == "ol", attr(k, "start")))
			}
			continue
		}
		if t != "li" {
			if s := strings.TrimSpace(c.inline(k)); s != "" && k.Type != html.TextNode {
				items = append(items, "- "+s)
			}
			continue
		}
		marker := "- "
		if ordered {
			marker = itoa(num) + ". "
			num++
		}
		var head []*html.Node
		var rest []string
		checked := ""
		for x := k.FirstChild; x != nil; x = x.NextSibling {
			if tag(x) == "input" && strings.EqualFold(attr(x, "type"), "checkbox") {
				if hasAttr(x, "checked") {
					checked = "[x] "
				} else {
					checked = "[ ] "
				}
				continue
			}
			if tag(x) == "en-todo" {
				if strings.EqualFold(attr(x, "checked"), "true") {
					checked = "[x] "
				} else {
					checked = "[ ] "
				}
				for y := x.FirstChild; y != nil; y = y.NextSibling {
					head = append(head, y)
				}
				continue
			}
			if isBlock(x) {
				if tag(x) == "p" || tag(x) == "div" {
					if len(rest) == 0 && len(head) == 0 && !hasBlockChild(x) {
						for y := x.FirstChild; y != nil; y = y.NextSibling {
							head = append(head, y)
						}
						continue
					}
				}
				if tag(x) == "ul" || tag(x) == "ol" {
					rest = append(rest, c.list(x, tag(x) == "ol", attr(x, "start")))
					continue
				}
				rest = append(rest, c.block(x)...)
				continue
			}
			if len(rest) > 0 {
				if s := strings.TrimSpace(c.inline(x)); s != "" {
					rest = append(rest, s)
				}
				continue
			}
			head = append(head, x)
		}
		first := ""
		for _, h := range head {
			if tag(h) == "img" || tag(h) == "en-media" {
				md, _ := c.mediaMD(h)
				first += md
				continue
			}
			first += c.inline(h)
		}
		first = oneLineKeepBreaks(first)
		if checked != "" && !ordered {
			marker = "- " + checked
		}
		item := marker + first
		pad := strings.Repeat(" ", len(marker))
		if checked != "" && !ordered {
			pad = "  "
		}
		for _, r := range rest {
			item += "\n" + indent(r, pad)
		}
		items = append(items, item)
	}
	return strings.Join(items, "\n")
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return true
		}
	}
	return false
}

func oneLineKeepBreaks(s string) string {
	parts := strings.Split(s, "\\\n")
	for i, p := range parts {
		parts[i] = strings.Join(strings.Fields(p), " ")
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func indent(s, pad string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func (c *htmlConv) table(n *html.Node) []string {
	var rows [][]string
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		for k := x.FirstChild; k != nil; k = k.NextSibling {
			switch tag(k) {
			case "tr":
				var cells []string
				for cell := k.FirstChild; cell != nil; cell = cell.NextSibling {
					if t := tag(cell); t == "td" || t == "th" {
						s := c.cellText(cell)
						cells = append(cells, s)
						if span := attr(cell, "colspan"); span != "" {
							for i := 1; i < atoiSmall(span) && i < 20; i++ {
								cells = append(cells, "")
							}
						}
					}
				}
				rows = append(rows, cells)
			case "thead", "tbody", "tfoot":
				walk(k)
			case "table":
			}
		}
	}
	walk(n)
	if len(rows) == 0 {
		return nil
	}
	width := 0
	for _, r := range rows {
		width = max(width, len(r))
	}
	if width == 0 {
		return nil
	}
	var b strings.Builder
	line := func(r []string) {
		b.WriteString("|")
		for i := 0; i < width; i++ {
			v := ""
			if i < len(r) {
				v = r[i]
			}
			b.WriteString(" " + v + " |")
		}
		b.WriteString("\n")
	}
	line(rows[0])
	b.WriteString("|")
	for i := 0; i < width; i++ {
		b.WriteString(" --- |")
	}
	b.WriteString("\n")
	for _, r := range rows[1:] {
		line(r)
	}
	return []string{strings.TrimRight(b.String(), "\n")}
}

func atoiSmall(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
		if n > 1000 {
			break
		}
	}
	return n
}

func (c *htmlConv) cellText(n *html.Node) string {
	var parts []string
	for _, blk := range c.container(n) {
		parts = append(parts, oneLine(blk))
	}
	s := strings.Join(parts, " ")
	return strings.ReplaceAll(s, "|", "\\|")
}

func (c *htmlConv) mediaMD(n *html.Node) (string, bool) {
	return c.mediaMDCaption(n, "")
}

func (c *htmlConv) mediaMDCaption(n *html.Node, caption string) (string, bool) {
	alt := strings.TrimSpace(attr(n, "alt"))
	var dest string
	image := true
	if tag(n) == "en-media" {
		if c.o.enMedia == nil {
			return "", false
		}
		dest, image = c.o.enMedia(attr(n, "hash"), attr(n, "type"))
	} else {
		src := attr(n, "src")
		if src == "" {
			return "", false
		}
		if c.o.media != nil {
			dest, image = c.o.media(src, alt)
		} else if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			dest = src
		}
	}
	if dest == "" {
		return "", false
	}
	alt = escapeText(alt)
	if !image {
		if alt == "" {
			alt = "file"
		}
		return "[" + alt + "](" + dest + ")", true
	}
	if caption != "" {
		return "![" + alt + "](" + dest + " \"" + strings.ReplaceAll(caption, "\"", "'") + "\")", true
	}
	return "![" + alt + "](" + dest + ")", true
}

func (c *htmlConv) inlineChildren(n *html.Node) string {
	var b strings.Builder
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		b.WriteString(c.inline(k))
	}
	return b.String()
}

func styleHas(n *html.Node, key, val string) bool {
	st := strings.ToLower(strings.ReplaceAll(attr(n, "style"), " ", ""))
	return strings.Contains(st, key+":"+val)
}

func wrap(s, open, close string) string {
	core := strings.TrimSpace(s)
	if core == "" {
		return s
	}
	lead := s[:len(s)-len(strings.TrimLeftFunc(s, unicode.IsSpace))]
	trail := s[len(strings.TrimRightFunc(s, unicode.IsSpace)):]
	return lead + open + core + close + trail
}

func (c *htmlConv) inline(n *html.Node) string {
	switch n.Type {
	case html.TextNode:
		return escapeText(collapse(n.Data))
	case html.ElementNode:
	default:
		return ""
	}
	t := tag(n)
	if skipTags[t] {
		return ""
	}
	switch n.DataAtom {
	case atom.Br:
		return "\\\n"
	case atom.Img:
		md, _ := c.mediaMD(n)
		return md
	case atom.Input:
		return ""
	}
	if t == "en-media" {
		md, _ := c.mediaMD(n)
		return md
	}
	if t == "en-todo" {
		if strings.EqualFold(attr(n, "checked"), "true") {
			return "[x] " + c.inlineChildren(n)
		}
		return "[ ] " + c.inlineChildren(n)
	}
	inner := c.inlineChildren(n)
	switch t {
	case "strong", "b":
		return wrap(inner, "**", "**")
	case "em", "i", "cite", "dfn":
		return wrap(inner, "*", "*")
	case "s", "del", "strike":
		return wrap(inner, "~~", "~~")
	case "u", "ins":
		return wrap(inner, "<u>", "</u>")
	case "code", "kbd", "tt", "samp":
		raw := collapse(textContent(n))
		if strings.TrimSpace(raw) == "" {
			return raw
		}
		f := "`"
		if strings.Contains(raw, "`") {
			f = "``"
		}
		return wrap(raw, f, f)
	case "a":
		href := strings.TrimSpace(attr(n, "href"))
		if href == "" || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return inner
		}
		if c.o.link != nil {
			href = c.o.link(href)
		}
		if href == "" {
			return inner
		}
		label := strings.TrimSpace(inner)
		if label == "" {
			label = escapeText(href)
		}
		if strings.ContainsAny(href, " ()<>") {
			href = "<" + strings.NewReplacer("<", "%3C", ">", "%3E").Replace(href) + ">"
		}
		return "[" + label + "](" + href + ")"
	case "span", "font":
		out := inner
		if styleHas(n, "font-weight", "bold") || styleHas(n, "font-weight", "700") || styleHas(n, "font-weight", "600") {
			out = wrap(out, "**", "**")
		}
		if styleHas(n, "font-style", "italic") {
			out = wrap(out, "*", "*")
		}
		if styleHas(n, "text-decoration", "line-through") {
			out = wrap(out, "~~", "~~")
		}
		if styleHas(n, "text-decoration", "underline") {
			out = wrap(out, "<u>", "</u>")
		}
		return out
	}
	if isBlock(n) {
		return " " + inner + " "
	}
	return inner
}

func collapse(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) && r != ' ' {
			if !space {
				b.WriteByte(' ')
			}
			space = true
			continue
		}
		if r == ' ' {
			r = ' '
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}

var textEscaper = strings.NewReplacer("\\", "\\\\", "*", "\\*", "_", "\\_", "`", "\\`", "[", "\\[", "]", "\\]", "<", "\\<", "~", "\\~", "$", "\\$")

func escapeText(s string) string { return textEscaper.Replace(s) }
