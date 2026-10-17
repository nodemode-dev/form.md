package transfer

import (
	"bytes"
	"encoding/csv"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

type treeImporter struct {
	v      *vfs
	b      *Bundle
	notion bool
	titles bool
	docs   map[string]*Node
	owners map[string]*Node
	tables map[*Node]string
	roots  []*Node
	assets map[string]*Asset
	kids   map[*Node][]childAt
}

type childAt struct {
	key  string
	node *Node
}

var notionID = regexp.MustCompile(`\s+[0-9a-f]{32}$`)

func (t *treeImporter) titleOf(name string) string {
	name = strings.TrimSpace(stripExt(name))
	if t.notion {
		name = notionID.ReplaceAllString(name, "")
		name = strings.TrimSuffix(name, "_all")
		name = notionID.ReplaceAllString(name, "")
	}
	return strings.TrimSpace(name)
}

func docExt(e string) bool {
	switch e {
	case ".md", ".markdown", ".mdown", ".txt", ".html", ".htm":
		return true
	}
	return false
}

func importTree(v *vfs, b *Bundle, notion bool) {
	t := &treeImporter{v: v, b: b, notion: notion, docs: map[string]*Node{}, owners: map[string]*Node{}, tables: map[*Node]string{}, assets: map[string]*Asset{}, kids: map[*Node][]childAt{}}
	files := v.sorted()
	csvs := map[string]*vfile{}
	for _, f := range files {
		if extOf(f.path) == ".csv" {
			key := strings.ToLower(stripExt(f.path))
			if notion {
				if base, ok := strings.CutSuffix(key, "_all"); ok {
					csvs[base] = f
					continue
				}
				if _, have := csvs[key]; have {
					continue
				}
			}
			csvs[key] = f
		}
	}
	for _, f := range files {
		e := extOf(f.path)
		if !docExt(e) {
			continue
		}
		if notion && e == ".html" && hasMarkdownTwin(v, f.path) {
			continue
		}
		n := &Node{Title: t.titleOf(path.Base(f.path))}
		t.docs[strings.ToLower(f.path)] = n
		t.owners[strings.ToLower(stripExt(f.path))] = n
	}
	for key, f := range csvs {
		n := &Node{Title: t.titleOf(path.Base(f.path)), Table: &Table{}}
		t.tables[n] = f.path
		t.owners[key] = n
		t.docs[strings.ToLower(f.path)] = n
	}
	for _, f := range files {
		n, ok := t.docs[strings.ToLower(f.path)]
		if !ok {
			continue
		}
		if n.Table != nil && t.tables[n] != f.path {
			continue
		}
		t.place(f.path, n)
	}
	for n, p := range t.tables {
		t.fillTable(n, p)
	}
	for _, f := range files {
		n, ok := t.docs[strings.ToLower(f.path)]
		if !ok || n.Table != nil {
			continue
		}
		data, err := v.read(f)
		if err != nil {
			b.Warn("Could not read " + f.path)
			continue
		}
		t.convert(f.path, data, n)
	}
	for n, list := range t.kids {
		sort.SliceStable(list, func(i, j int) bool { return list[i].key < list[j].key })
		for _, c := range list {
			if n.Table != nil {
				continue
			}
			n.Children = append(n.Children, c.node)
		}
	}
	b.Roots = append(b.Roots, t.roots...)
}

func hasMarkdownTwin(v *vfs, p string) bool {
	return v.lookup(stripExt(p)+".md") != nil
}

func (t *treeImporter) parentOf(p string) *Node {
	dir := path.Dir(p)
	if dir == "." || dir == "" {
		return nil
	}
	if n, ok := t.owners[strings.ToLower(dir)]; ok {
		return n
	}
	folder := &Node{Title: t.titleOf(path.Base(dir))}
	t.owners[strings.ToLower(dir)] = folder
	t.place(dir, folder)
	return folder
}

func (t *treeImporter) place(p string, n *Node) {
	base := strings.ToLower(stripExt(path.Base(p)))
	if (base == "index" || base == "readme") && path.Dir(p) != "." {
		if owner := t.parentOf(p); owner != nil && owner.Table == nil && owner.Body == "" {
			t.docs[strings.ToLower(p)] = owner
			return
		}
	}
	parent := t.parentOf(p)
	if parent == nil {
		t.roots = append(t.roots, n)
		return
	}
	t.kids[parent] = append(t.kids[parent], childAt{key: strings.ToLower(path.Base(p)), node: n})
}

func (t *treeImporter) fillTable(n *Node, p string) {
	f := t.v.lookup(p)
	if f == nil {
		return
	}
	data, err := t.v.read(f)
	if err != nil {
		t.b.Warn("Could not read " + p)
		return
	}
	headers, rows := readCSVBytes(data)
	if len(headers) == 0 {
		t.b.Warn("The table " + p + " is empty")
		return
	}
	n.Table.Headers, n.Table.Rows = headers, rows
	n.Table.Bodies = make([]*Node, len(rows))
	pending := map[string][]int{}
	for i, r := range rows {
		if len(r) > 0 {
			k := strings.ToLower(strings.TrimSpace(r[0]))
			pending[k] = append(pending[k], i)
		}
	}
	for _, c := range t.kids[n] {
		k := strings.ToLower(strings.TrimSpace(c.node.Title))
		if list := pending[k]; len(list) > 0 {
			n.Table.Bodies[list[0]] = c.node
			pending[k] = list[1:]
			continue
		}
		row := make([]string, len(headers))
		row[0] = c.node.Title
		n.Table.Rows = append(n.Table.Rows, row)
		n.Table.Bodies = append(n.Table.Bodies, c.node)
	}
	if t.notion {
		for i := range headers {
			if i > 0 && relationColumn(rows, i) {
				n.Table.Types = padTypes(n.Table.Types, len(headers))
				n.Table.Types[i] = "text"
				for _, r := range n.Table.Rows {
					if i < len(r) {
						r[i] = stripNotionLinks(r[i])
					}
				}
			}
		}
	}
}

func padTypes(t []string, n int) []string {
	for len(t) < n {
		t = append(t, "")
	}
	return t
}

var notionLinkCell = regexp.MustCompile(`\s*\([^()]*\.(?:md|csv)\)`)

func relationColumn(rows [][]string, i int) bool {
	for _, r := range rows {
		if i < len(r) && notionLinkCell.MatchString(r[i]) {
			return true
		}
	}
	return false
}

func stripNotionLinks(s string) string {
	s = notionLinkCell.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func readCSVBytes(data []byte) ([]string, [][]string) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data[:min(len(data), 4096)]) {
		data = []byte(strings.ToValidUTF8(string(data), "?"))
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	if line, _, _ := bytes.Cut(data, []byte("\n")); bytes.Count(line, []byte(";")) > bytes.Count(line, []byte(",")) {
		r.Comma = ';'
	} else if bytes.Count(line, []byte("\t")) > bytes.Count(line, []byte(",")) {
		r.Comma = '\t'
	}
	all, _ := r.ReadAll()
	if len(all) == 0 {
		return nil, nil
	}
	var rows [][]string
	for _, rec := range all[1:] {
		empty := true
		for _, c := range rec {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if !empty {
			rows = append(rows, rec)
		}
	}
	return all[0], rows
}

func (t *treeImporter) convert(p string, data []byte, n *Node) {
	switch extOf(p) {
	case ".html", ".htm":
		root, err := html.Parse(bytes.NewReader(data))
		if err != nil {
			return
		}
		if title := findTitle(root); title != "" && (n.Title == "" || t.notion || t.titles) {
			n.Title = title
		}
		body := mainContent(root)
		dir := path.Dir(p)
		n.Body = htmlToMarkdown(body, htmlOpts{
			link: func(href string) string { return t.resolve(dir, href) },
			media: func(src, _ string) (string, bool) {
				d := t.resolve(dir, src)
				return d, !strings.HasPrefix(d, "page:") && isImageName(src)
			},
		}, t.b.Warn)
		n.Body = dropLeadingHeading(n.Body, n.Title)
	case ".txt":
		n.Body = textToMarkdown(string(data))
	default:
		src := normalizeText(data)
		src, fmTitle := frontMatter(src)
		if fmTitle != "" {
			n.Title = fmTitle
		}
		if h1, rest, ok := leadingH1(src); ok && (t.notion || strings.EqualFold(strings.TrimSpace(h1), strings.TrimSpace(n.Title)) || n.Title == "") {
			n.Title = h1
			src = rest
		}
		if t.notion {
			src = notionAsides(src)
			src = t.dropNotionProps(src, n)
		}
		n.Body = t.rewriteLinks(path.Dir(p), src)
	}
}

func normalizeText(data []byte) string {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	s := string(data)
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "?")
	}
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func frontMatter(s string) (string, string) {
	if !strings.HasPrefix(s, "---\n") {
		return s, ""
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return s, ""
	}
	head := s[4 : 4+end]
	rest := strings.TrimLeft(s[4+end+4:], "\n")
	title := ""
	for _, line := range strings.Split(head, "\n") {
		if v, ok := strings.CutPrefix(line, "title:"); ok {
			title = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return rest, title
}

func leadingH1(s string) (string, string, bool) {
	trimmed := strings.TrimLeft(s, "\n ")
	if !strings.HasPrefix(trimmed, "# ") {
		return "", s, false
	}
	line, rest, _ := strings.Cut(trimmed, "\n")
	return strings.TrimSpace(strings.TrimPrefix(line, "# ")), strings.TrimLeft(rest, "\n"), true
}

func dropLeadingHeading(md, title string) string {
	if h, rest, ok := leadingH1(md); ok && strings.EqualFold(h, title) {
		return rest
	}
	return md
}

var asideRE = regexp.MustCompile(`(?s)<aside>\s*(.*?)\s*</aside>`)

func notionAsides(s string) string {
	return asideRE.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSpace(asideRE.FindStringSubmatch(m)[1])
		inner = stripLeadingSymbol(inner)
		inner = strings.ReplaceAll(inner, "<br>", "\n")
		return prefixAll("[!NOTE]\n"+inner, "> ")
	})
}

func stripLeadingSymbol(s string) string {
	for {
		r, size := utf8.DecodeRuneInString(s)
		if size == 0 || !isPictograph(r) {
			break
		}
		s = s[size:]
	}
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func isPictograph(r rune) bool {
	return (r >= 0x1F000 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) || (r >= 0x2B00 && r <= 0x2BFF) || r == 0xFE0F || r == 0x200D || (r >= 0x1F1E6 && r <= 0x1F1FF)
}

func stripPictographs(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !isPictograph(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(strings.Join(strings.Fields(b.String()), " "))
}

func (t *treeImporter) dropNotionProps(src string, n *Node) string {
	lines := strings.Split(src, "\n")
	i := 0
	dropped := 0
	for i < len(lines) {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			if dropped > 0 {
				i++
				break
			}
			i++
			continue
		}
		k, _, ok := strings.Cut(l, ": ")
		if !ok || len(k) > 80 || strings.HasPrefix(k, "#") || strings.HasPrefix(k, "-") || strings.HasPrefix(k, ">") {
			break
		}
		dropped++
		i++
	}
	if dropped == 0 {
		return src
	}
	return strings.Join(lines[i:], "\n")
}

var mdLinkRE = regexp.MustCompile(`(!?\[[^\]\n]*\]\()(<[^>\n]+>|[^)\s]+)((?:\s+"[^"\n]*")?\))`)

func (t *treeImporter) rewriteLinks(dir, src string) string {
	return mdLinkRE.ReplaceAllStringFunc(src, func(m string) string {
		sub := mdLinkRE.FindStringSubmatch(m)
		dest := strings.Trim(sub[2], "<>")
		r := t.resolve(dir, dest)
		if r == "" || r == dest {
			if t.notion && !strings.HasPrefix(sub[1], "!") && deadPageLink(dest) {
				return strings.TrimSuffix(strings.TrimPrefix(sub[1], "["), "](")
			}
			return m
		}
		if t.notion && sub[1] == "!["+dest+"](" {
			alt, err := url.PathUnescape(dest)
			if err != nil {
				alt = dest
			}
			sub[1] = "![" + strings.TrimSpace(stripExt(path.Base(alt))) + "]("
		}
		return sub[1] + r + sub[3]
	})
}

func deadPageLink(dest string) bool {
	if strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:") || strings.HasPrefix(dest, "/") {
		return false
	}
	if i := strings.IndexAny(dest, "?"); i >= 0 {
		dest = dest[:i]
	}
	e := strings.ToLower(path.Ext(dest))
	return e == ".md" || e == ".csv"
}

func (t *treeImporter) resolve(dir, dest string) string {
	if dest == "" || strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:") || strings.HasPrefix(dest, "data:") || strings.HasPrefix(dest, "/p/") || strings.HasPrefix(dest, "/files/") {
		return dest
	}
	if r := t.resolvePath(dir, dest); r != "" {
		return r
	}
	if strings.HasPrefix(dest, "#") {
		return dest
	}
	if i := strings.IndexAny(dest, "?#"); i >= 0 {
		if r := t.resolvePath(dir, dest[:i]); r != "" {
			return r
		}
	}
	return dest
}

func (t *treeImporter) resolvePath(dir, dest string) string {
	dec, err := url.PathUnescape(dest)
	if err != nil {
		dec = dest
	}
	full := cleanPath(path.Join(dir, dec))
	if n, ok := t.docs[strings.ToLower(full)]; ok {
		return t.b.PageRef(n)
	}
	if e := strings.ToLower(path.Ext(full)); e == ".csv" || e == ".md" {
		if n, ok := t.owners[strings.ToLower(strings.TrimSuffix(stripExt(full), "_all"))]; ok {
			return t.b.PageRef(n)
		}
	}
	f := t.v.lookup(full)
	if f == nil {
		return ""
	}
	a, ok := t.assets[f.path]
	if !ok {
		data, err := t.v.read(f)
		if err != nil {
			return ""
		}
		a = &Asset{Name: path.Base(f.path), Data: data}
		t.assets[f.path] = a
	}
	t.b.used(f.path)
	return t.b.AssetRef(a)
}

func isImageName(p string) bool {
	switch extOf(p) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".avif":
		return true
	}
	return false
}

func findTitle(root *html.Node) string {
	var title string
	var walk func(*html.Node) bool
	walk = func(n *html.Node) bool {
		if tag(n) == "title" {
			title = strings.TrimSpace(textContent(n))
			return true
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if walk(k) {
				return true
			}
		}
		return false
	}
	walk(root)
	return oneLine(title)
}

func mainContent(root *html.Node) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if id := attr(n, "id"); id == "main-content" || id == "content" {
			found = n
			return
		}
		if tag(n) == "main" || tag(n) == "article" {
			found = n
			return
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
	}
	walk(root)
	if found != nil {
		return found
	}
	var body *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if body != nil {
			return
		}
		if tag(n) == "body" {
			body = n
			return
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			find(k)
		}
	}
	find(root)
	if body != nil {
		return body
	}
	return root
}

func textToMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	paras := regexp.MustCompile(`\n\s*\n`).Split(strings.TrimSpace(s), -1)
	out := make([]string, 0, len(paras))
	for _, p := range paras {
		lines := strings.Split(p, "\n")
		for i, l := range lines {
			lines[i] = escapeLead(escapeText(strings.TrimRight(l, " \t")))
		}
		out = append(out, strings.Join(lines, "\\\n"))
	}
	return strings.Join(out, "\n\n")
}
