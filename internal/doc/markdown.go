package doc

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/reearth/ygo/crdt"
)

type Summary struct {
	Markdown  string
	Text      string
	Snippet   string
	Thumb     string
	Pages     []string
	Files     []string
	Users     []string
	Reminders []Reminder
}

type Reminder struct {
	Ref  string
	User string
	At   int64
}

const snippetRunes = 240

type mdWriter struct {
	text   strings.Builder
	thumb  string
	pages  map[string]bool
	files  map[string]bool
	users  map[string]bool
	remind []Reminder
	title  func(id string) string
	self   string
	doc    *crdt.Doc
}

func Summarise(frag *crdt.YXmlFragment, title func(id string) string) Summary {
	return summarise(&mdWriter{pages: map[string]bool{}, files: map[string]bool{}, users: map[string]bool{}, title: title}, frag)
}

func (r *room) summary(title func(id string) string) Summary {
	return summarise(&mdWriter{pages: map[string]bool{}, files: map[string]bool{}, users: map[string]bool{}, title: title, self: r.id, doc: r.doc}, r.fragment())
}

func summarise(w *mdWriter, frag *crdt.YXmlFragment) Summary {
	lines := w.blocks(children(frag), true)
	md := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if md != "" {
		md += "\n"
	}
	text := strings.TrimSpace(w.text.String())
	return Summary{
		Markdown:  md,
		Text:      text,
		Snippet:   snippet(text),
		Thumb:     w.thumb,
		Pages:     keys(w.pages),
		Files:     keys(w.files),
		Users:     keys(w.users),
		Reminders: w.remind,
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func snippet(text string) string {
	s := strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(s) <= snippetRunes {
		return s
	}
	r := []rune(s)[:snippetRunes]
	cut := len(r)
	for i := len(r) - 1; i > snippetRunes*3/4; i-- {
		if unicode.IsSpace(r[i]) {
			cut = i
			break
		}
	}
	return strings.TrimRightFunc(string(r[:cut]), func(c rune) bool { return unicode.IsSpace(c) || unicode.IsPunct(c) }) + "..."
}

func (w *mdWriter) addText(s string) {
	if s == "" {
		return
	}
	w.text.WriteString(s)
}

func (w *mdWriter) blockBreak() {
	w.text.WriteString("\n")
}

func (w *mdWriter) noteFile(ref string) {
	if id := fileID(ref); id != "" {
		w.files[id] = true
	}
}

func (w *mdWriter) blocks(nodes []any, loose bool) []string {
	var out []string
	for i, n := range nodes {
		el, ok := n.(*crdt.YXmlElement)
		if !ok {
			continue
		}
		lines := w.block(el)
		if len(lines) == 0 {
			continue
		}
		if i > 0 && len(out) > 0 && loose {
			out = append(out, "")
		}
		out = append(out, lines...)
	}
	return out
}

func prefixLines(lines []string, first, rest string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		p := rest
		if i == 0 {
			p = first
		}
		if l == "" && strings.TrimSpace(p) == "" {
			out[i] = ""
			continue
		}
		if l == "" {
			out[i] = strings.TrimRight(p, " ")
			continue
		}
		out[i] = p + l
	}
	return out
}

func (w *mdWriter) block(el *crdt.YXmlElement) []string {
	switch el.NodeName {
	case "paragraph":
		s := w.inline(el, false)
		w.blockBreak()
		if s == "" {
			return []string{"<br>"}
		}
		return strings.Split(s, "\n")
	case "heading":
		level := attrInt(el, "level", 1)
		if level < 1 || level > 6 {
			level = 1
		}
		s := strings.ReplaceAll(w.inline(el, false), "\\\n", " ")
		w.blockBreak()
		return []string{strings.Repeat("#", level) + " " + s}
	case "code_block":
		code := w.plain(el)
		w.addText(code)
		w.blockBreak()
		fence := "```"
		for strings.Contains(code, fence) {
			fence += "`"
		}
		lines := []string{fence + attrString(el, "language")}
		if code != "" {
			lines = append(lines, strings.Split(code, "\n")...)
		}
		return append(lines, fence)
	case "horizontal_rule":
		return []string{"---"}
	case "blockquote":
		return prefixLines(w.blocks(children(el), true), "> ", "> ")
	case "callout":
		kind := map[string]string{"green": "TIP", "violet": "IMPORTANT", "amber": "WARNING", "orange": "WARNING", "rose": "CAUTION", "pink": "CAUTION"}[attrString(el, "tone")]
		if kind == "" {
			kind = "NOTE"
		}
		inner := append([]string{"[!" + kind + "]"}, w.blocks(children(el), true)...)
		return prefixLines(inner, "> ", "> ")
	case "toggle":
		kids := children(el)
		summary := ""
		var body []any
		for _, k := range kids {
			if ke, ok := k.(*crdt.YXmlElement); ok && ke.NodeName == "toggle_title" {
				summary = strings.ReplaceAll(w.inline(ke, false), "\\\n", " ")
				w.blockBreak()
				continue
			}
			body = append(body, k)
		}
		if lvl := attrInt(el, "level", 0); lvl > 0 && lvl <= 3 {
			summary = "<h" + strconv.Itoa(lvl) + ">" + summary + "</h" + strconv.Itoa(lvl) + ">"
		}
		lines := []string{"<details>", "<summary>" + summary + "</summary>", ""}
		lines = append(lines, w.blocks(body, true)...)
		return append(lines, "", "</details>")
	case "bullet_list", "ordered_list", "task_list":
		return w.list(el)
	case "table":
		return w.table(el)
	case "image":
		src := attrString(el, "src")
		w.noteFile(src)
		if w.thumb == "" && fileID(src) != "" {
			w.thumb = "f:" + fileID(src)
		}
		alt := attrString(el, "alt")
		caption := attrString(el, "caption")
		w.addText(caption)
		w.blockBreak()
		line := "![" + escapeInline(alt) + "](" + linkDest(fileURL(src))
		if caption != "" {
			line += ` "` + strings.ReplaceAll(caption, `"`, `\"`) + `"`
		}
		if enc := marksEncode(attrString(el, "marks")); enc != "" {
			return []string{line + ")", "", "<!-- marks:" + enc + " -->"}
		}
		return []string{line + ")"}
	case "video", "audio":
		src := attrString(el, "src")
		w.noteFile(src)
		caption := attrString(el, "caption")
		w.addText(caption)
		w.blockBreak()
		label := caption
		if label == "" {
			label = el.NodeName
		}
		return []string{"![" + escapeInline(label) + "](" + linkDest(fileURL(src)) + ")"}
	case "file":
		src := attrString(el, "src")
		w.noteFile(src)
		name := attrString(el, "name")
		if name == "" {
			name = "file"
		}
		w.addText(name)
		w.blockBreak()
		return []string{"[" + escapeInline(name) + "](" + linkDest(fileURL(src)) + ")"}
	case "bookmark":
		url := attrString(el, "url")
		w.noteFile(attrString(el, "image"))
		w.addText(attrString(el, "title"))
		w.blockBreak()
		return []string{"<" + url + ">"}
	case "embed":
		return []string{"<" + attrString(el, "url") + ">"}
	case "board":
		items := attrString(el, "items")
		for _, it := range parseBoard(items) {
			w.noteFile(it.Src)
			w.addText(it.Text)
		}
		w.blockBreak()
		return []string{"<!-- board:" + boardEncode(attrInt(el, "height", 600), items) + " -->"}
	case "toc":
		return []string{"<!-- toc -->"}
	case "button":
		label := attrString(el, "label")
		w.addText(label)
		return []string{"<!-- button:" + buttonEncode(label, attrString(el, "actions")) + " -->"}
	case "breadcrumb":
		return []string{"<!-- breadcrumb -->"}
	case "synced":
		pg, key := attrString(el, "page"), attrString(el, "key")
		if !SyncKey(key) {
			return nil
		}
		if pg != w.self {
			w.pages[pg] = true
		}
		if pg != w.self || w.doc == nil || w.doc.GetXmlFragment(SyncRoot(key)).Len() == 0 {
			return []string{"<!-- synced:" + pg + "/" + key + " -->"}
		}
		inner := (&mdWriter{pages: w.pages, files: w.files, users: w.users, title: w.title}).blocks(children(w.doc.GetXmlFragment(SyncRoot(key))), true)
		w.text.WriteString(" " + strings.Join(inner, " ") + " ")
		return append(append([]string{"<!-- synced:" + pg + "/" + key + " begin -->", ""}, inner...), "", "<!-- synced end -->")
	case "math_block":
		tex := attrString(el, "tex")
		w.addText(tex)
		return append(append([]string{"$$"}, strings.Split(tex, "\n")...), "$$")
	case "database":
		id := attrString(el, "collection")
		if id == "" {
			return nil
		}
		w.pages[id] = true
		return []string{"<!-- database:" + id + " -->"}
	case "columns":
		var out []string
		for _, c := range children(el) {
			col, ok := c.(*crdt.YXmlElement)
			if !ok {
				continue
			}
			lines := w.blocks(children(col), true)
			if len(lines) == 0 {
				continue
			}
			if len(out) > 0 {
				out = append(out, "")
			}
			out = append(out, lines...)
		}
		return out
	}
	return w.blocks(children(el), true)
}

func (w *mdWriter) list(el *crdt.YXmlElement) []string {
	var out []string
	n := attrInt(el, "start", 1)
	loose := false
	for _, it := range children(el) {
		item, ok := it.(*crdt.YXmlElement)
		if ok && len(children(item)) > 1 {
			loose = true
		}
	}
	for _, it := range children(el) {
		item, ok := it.(*crdt.YXmlElement)
		if !ok {
			continue
		}
		marker := "- "
		switch el.NodeName {
		case "ordered_list":
			marker = strconv.Itoa(n) + ". "
			n++
		case "task_list":
			if attrBool(item, "checked") {
				marker = "- [x] "
			} else {
				marker = "- [ ] "
			}
		}
		indent := strings.Repeat(" ", len(marker))
		if el.NodeName == "task_list" {
			indent = "  "
		}
		lines := w.blocks(children(item), loose)
		if len(lines) == 0 {
			lines = []string{""}
		}
		if lines[0] == "<br>" {
			lines[0] = ""
		}
		if loose && len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, prefixLines(lines, marker, indent)...)
	}
	return out
}

func (w *mdWriter) table(el *crdt.YXmlElement) []string {
	var rows [][]string
	width := 0
	for _, r := range children(el) {
		row, ok := r.(*crdt.YXmlElement)
		if !ok {
			continue
		}
		var cells []string
		for _, c := range children(row) {
			cell, ok := c.(*crdt.YXmlElement)
			if !ok {
				continue
			}
			var parts []string
			for _, p := range children(cell) {
				if pe, ok := p.(*crdt.YXmlElement); ok {
					parts = append(parts, strings.ReplaceAll(strings.ReplaceAll(w.inline(pe, true), "\\\n", "<br>"), "\n", " "))
					w.addText(" ")
				}
			}
			cells = append(cells, strings.Join(parts, "<br>"))
		}
		w.blockBreak()
		if len(cells) > width {
			width = len(cells)
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 || width == 0 {
		return nil
	}
	line := func(cells []string) string {
		for len(cells) < width {
			cells = append(cells, "")
		}
		return "| " + strings.Join(cells, " | ") + " |"
	}
	out := []string{line(rows[0])}
	sep := make([]string, width)
	for i := range sep {
		sep[i] = "---"
	}
	out = append(out, line(sep))
	for _, r := range rows[1:] {
		out = append(out, line(r))
	}
	return out
}

func (w *mdWriter) plain(el *crdt.YXmlElement) string {
	var b strings.Builder
	for _, c := range el.Children() {
		if t, ok := c.(*crdt.YXmlText); ok {
			for _, d := range t.ToDelta() {
				if s, ok := d.Insert.(string); ok {
					b.WriteString(s)
				}
			}
		}
	}
	return b.String()
}

type run struct {
	text  string
	marks map[string]any
	raw   bool
}

func (w *mdWriter) inline(el *crdt.YXmlElement, inTable bool) string {
	var runs []run
	for _, c := range el.Children() {
		switch x := c.(type) {
		case *crdt.YXmlText:
			for _, d := range x.ToDelta() {
				s, ok := d.Insert.(string)
				if !ok || s == "" {
					continue
				}
				marks := map[string]any{}
				for k, v := range d.Attributes {
					if v != nil {
						marks[markName(k)] = v
					}
				}
				runs = append(runs, run{text: s, marks: marks})
				w.addText(s)
			}
		case *crdt.YXmlElement:
			switch x.NodeName {
			case "hard_break":
				runs = append(runs, run{text: "\n"})
				w.addText(" ")
			case "mention":
				runs = append(runs, w.mention(x)...)
			case "math_inline":
				tex := attrString(x, "tex")
				w.addText(tex)
				runs = append(runs, run{text: "$" + tex + "$", raw: true})
			}
		}
	}
	return renderRuns(runs, inTable)
}

func (w *mdWriter) mention(el *crdt.YXmlElement) []run {
	kind, ref, label := attrString(el, "kind"), attrString(el, "ref"), attrString(el, "label")
	switch kind {
	case "page":
		w.pages[ref] = true
		if w.title != nil {
			if t := w.title(ref); t != "" {
				label = t
			}
		}
		if label == "" {
			label = "Untitled"
		}
		w.addText(label)
		return []run{{text: label, marks: map[string]any{"link": map[string]any{"href": "/p/" + ref}}}}
	case "user":
		w.users[ref] = true
		w.addText("@" + label)
		return []run{{text: "@" + label}}
	case "date":
		if who := attrString(el, "remind"); who != "" && len(w.remind) < 200 {
			at, _ := strconv.ParseInt(attrString(el, "at"), 10, 64)
			w.remind = append(w.remind, Reminder{Ref: ref, User: who, At: at})
		}
		w.addText(ref)
		return []run{{text: ref}}
	}
	return nil
}

func renderRuns(runs []run, inTable bool) string {
	var b strings.Builder
	var open []string
	var openLink, codeFence string
	closeTo := func(n int) {
		for len(open) > n {
			m := open[len(open)-1]
			open = open[:len(open)-1]
			if m == "code" {
				b.WriteString(codeFence)
				continue
			}
			b.WriteString(closeDelim(m, openLink))
		}
	}
	atLineStart := true
	for _, r := range runs {
		if r.text == "\n" {
			closeTo(0)
			b.WriteString("\\\n")
			atLineStart = true
			continue
		}
		if r.raw {
			closeTo(0)
			b.WriteString(r.text)
			atLineStart = false
			continue
		}
		var want []string
		for _, m := range markOrder {
			if _, ok := r.marks[m]; ok {
				if m == "highlight" || m == "color" {
					continue
				}
				want = append(want, m)
			}
		}
		href := markAttr(r.marks["link"], "href")
		keep := 0
		for keep < len(open) && keep < len(want) && open[keep] == want[keep] && (open[keep] != "link" || href == openLink) {
			keep++
		}
		lead, core, trail := splitSpace(r.text)
		if keep < len(open) {
			closeTo(keep)
		}
		b.WriteString(lead)
		if core == "" {
			b.WriteString(trail)
			continue
		}
		for _, m := range want[keep:] {
			if m == "link" {
				openLink = href
			}
			if m == "code" {
				codeFence = fenceFor(core)
				b.WriteString(codeFence)
			} else {
				b.WriteString(openDelim(m))
			}
			open = append(open, m)
		}
		if len(open) > 0 && open[len(open)-1] == "code" {
			b.WriteString(codePad(core))
		} else {
			b.WriteString(escapeText(core, atLineStart, inTable))
		}
		atLineStart = false
		if trail != "" {
			closeTo(0)
			b.WriteString(trail)
		}
	}
	closeTo(0)
	return b.String()
}

func splitSpace(s string) (lead, core, trail string) {
	core = strings.TrimLeftFunc(s, unicode.IsSpace)
	lead = s[:len(s)-len(core)]
	trimmed := strings.TrimRightFunc(core, unicode.IsSpace)
	trail = core[len(trimmed):]
	return lead, trimmed, trail
}

func openDelim(m string) string {
	switch m {
	case "link":
		return "["
	case "strong":
		return "**"
	case "em":
		return "*"
	case "strike":
		return "~~"
	case "underline":
		return "<u>"
	case "code":
		return "`"
	}
	return ""
}

func closeDelim(m, href string) string {
	switch m {
	case "link":
		return "](" + linkDest(href) + ")"
	case "underline":
		return "</u>"
	}
	return openDelim(m)
}

func linkDest(href string) string {
	if strings.ContainsAny(href, " ()<>") {
		return "<" + strings.NewReplacer("<", "%3C", ">", "%3E").Replace(href) + ">"
	}
	return href
}

func fenceFor(s string) string {
	longest, cur := 0, 0
	for _, c := range s {
		if c == '`' {
			cur++
			if cur > longest {
				longest = cur
			}
		} else {
			cur = 0
		}
	}
	return strings.Repeat("`", longest+1)
}

func codePad(s string) string {
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return " " + s + " "
	}
	return s
}

var inlineEscaper = strings.NewReplacer(`\`, `\\`, "*", `\*`, "_", `\_`, "`", "\\`", "[", `\[`, "]", `\]`, "<", `\<`, "~", `\~`)

func escapeInline(s string) string {
	return inlineEscaper.Replace(s)
}

func escapeText(s string, lineStart, inTable bool) string {
	out := inlineEscaper.Replace(s)
	if inTable {
		out = strings.ReplaceAll(out, "|", `\|`)
	}
	if lineStart && out != "" {
		switch out[0] {
		case '#', '>', '-', '+', '=':
			out = `\` + out
		default:
			i := 0
			for i < len(out) && out[i] >= '0' && out[i] <= '9' {
				i++
			}
			if i > 0 && i < len(out) && (out[i] == '.' || out[i] == ')') {
				out = out[:i] + `\` + out[i:]
			}
		}
	}
	return out
}
