package transfer

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	Note     string        `xml:"_note,attr"`
	Complete string        `xml:"_complete,attr"`
	Children []opmlOutline `xml:"outline"`
}

type opmlDoc struct {
	Title string        `xml:"head>title"`
	Body  []opmlOutline `xml:"body>outline"`
}

func importOPML(name string, data []byte, b *Bundle) {
	var d opmlDoc
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity
	if err := dec.Decode(&d); err != nil {
		b.Warn("The outline in " + name + " could not be read")
		return
	}
	title := strings.TrimSpace(d.Title)
	items := d.Body
	if title == "" && len(items) == 1 && len(items[0].Children) > 0 {
		title = outlineText(items[0])
		items = items[0].Children
	}
	if title == "" {
		title = stripExt(name)
	}
	var lines []string
	var walk func(list []opmlOutline, depth int)
	walk = func(list []opmlOutline, depth int) {
		for _, o := range list {
			pad := strings.Repeat("  ", depth)
			label := inlineHTML(outlineText(o))
			if o.Complete == "true" {
				label = wrap(label, "~~", "~~")
			}
			item := pad + "- " + label
			if note := strings.TrimSpace(o.Note); note != "" {
				for _, l := range strings.Split(note, "\n") {
					if strings.TrimSpace(l) != "" {
						item += "\\\n" + pad + "  *" + escapeText(strings.TrimSpace(l)) + "*"
					}
				}
			}
			lines = append(lines, item)
			walk(o.Children, depth+1)
		}
	}
	walk(items, 0)
	b.Roots = append(b.Roots, &Node{Title: title, Body: strings.Join(lines, "\n")})
}

func outlineText(o opmlOutline) string {
	if o.Text != "" {
		return o.Text
	}
	return o.Title
}

func inlineHTML(s string) string {
	if !strings.ContainsAny(s, "<&") {
		return escapeText(s)
	}
	nodes, err := html.ParseFragment(strings.NewReader(s), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return escapeText(s)
	}
	c := &htmlConv{}
	var out strings.Builder
	for _, n := range nodes {
		out.WriteString(c.inline(n))
	}
	return oneLine(out.String())
}

type boardExport struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Lists []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Closed bool   `json:"closed"`
	} `json:"lists"`
	Cards []struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Desc         string   `json:"desc"`
		IDList       string   `json:"idList"`
		Closed       bool     `json:"closed"`
		Due          string   `json:"due"`
		DueComplete  bool     `json:"dueComplete"`
		Pos          float64  `json:"pos"`
		IDChecklists []string `json:"idChecklists"`
		Labels       []struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"labels"`
		ShortURL string `json:"shortUrl"`
	} `json:"cards"`
	Checklists []struct {
		ID         string `json:"id"`
		IDCard     string `json:"idCard"`
		Name       string `json:"name"`
		CheckItems []struct {
			Name  string  `json:"name"`
			State string  `json:"state"`
			Pos   float64 `json:"pos"`
		} `json:"checkItems"`
	} `json:"checklists"`
	Actions []struct {
		Type string `json:"type"`
		Date string `json:"date"`
		Data struct {
			Text string `json:"text"`
			Card struct {
				ID string `json:"id"`
			} `json:"card"`
		} `json:"data"`
		MemberCreator struct {
			FullName string `json:"fullName"`
		} `json:"memberCreator"`
	} `json:"actions"`
}

func looksLikeBoardJSON(data []byte) bool {
	head := data[:min(len(data), 1<<20)]
	return bytes.Contains(head, []byte(`"idList"`)) || (bytes.Contains(head, []byte(`"lists"`)) && bytes.Contains(head, []byte(`"cards"`)))
}

func importBoardJSON(name string, data []byte, b *Bundle) {
	var t boardExport
	if err := json.Unmarshal(data, &t); err != nil {
		b.Warn("The board export in " + name + " could not be read")
		return
	}
	lists := map[string]string{}
	order := map[string]int{}
	for i, l := range t.Lists {
		lists[l.ID] = l.Name
		order[l.ID] = i
	}
	checks := map[string][]string{}
	for _, cl := range t.Checklists {
		items := append([]struct {
			Name  string  `json:"name"`
			State string  `json:"state"`
			Pos   float64 `json:"pos"`
		}(nil), cl.CheckItems...)
		sort.SliceStable(items, func(i, j int) bool { return items[i].Pos < items[j].Pos })
		var lines []string
		if cl.Name != "" {
			lines = append(lines, "### "+escapeText(cl.Name))
		}
		for _, it := range items {
			box := "[ ]"
			if it.State == "complete" {
				box = "[x]"
			}
			lines = append(lines, "- "+box+" "+escapeText(it.Name))
		}
		checks[cl.IDCard] = append(checks[cl.IDCard], strings.Join(lines, "\n"))
	}
	comments := map[string][]string{}
	for _, a := range t.Actions {
		if a.Type == "commentCard" && strings.TrimSpace(a.Data.Text) != "" {
			who := a.MemberCreator.FullName
			when := ""
			if ts, err := time.Parse(time.RFC3339, a.Date); err == nil {
				when = ts.Format("January 2, 2006")
			}
			comments[a.Data.Card.ID] = append(comments[a.Data.Card.ID], "> **"+escapeText(who)+"** "+when+"\n> "+strings.ReplaceAll(strings.TrimSpace(a.Data.Text), "\n", "\n> "))
		}
	}
	cards := t.Cards
	sort.SliceStable(cards, func(i, j int) bool {
		if order[cards[i].IDList] != order[cards[j].IDList] {
			return order[cards[i].IDList] < order[cards[j].IDList]
		}
		return cards[i].Pos < cards[j].Pos
	})
	tbl := &Table{Headers: []string{"Name", "List", "Labels", "Due", "Done", "Archived", "Card link"}, Types: []string{"title", "select", "multi_select", "date", "checkbox", "checkbox", "url"}, Board: "List"}
	for _, c := range cards {
		var labels []string
		for _, l := range c.Labels {
			n := strings.TrimSpace(l.Name)
			if n == "" {
				n = l.Color
			}
			if n != "" {
				labels = append(labels, strings.ReplaceAll(n, ",", " "))
			}
		}
		due := ""
		if ts, err := time.Parse(time.RFC3339, c.Due); err == nil {
			due = ts.UTC().Format("2006-01-02 15:04")
		}
		tbl.Rows = append(tbl.Rows, []string{c.Name, lists[c.IDList], strings.Join(labels, ", "), due, yesNo(c.DueComplete), yesNo(c.Closed), c.ShortURL})
		var parts []string
		if d := strings.TrimSpace(c.Desc); d != "" {
			parts = append(parts, d)
		}
		parts = append(parts, checks[c.ID]...)
		if cs := comments[c.ID]; len(cs) > 0 {
			parts = append(parts, "## Comments")
			parts = append(parts, cs...)
		}
		var body *Node
		if len(parts) > 0 {
			body = &Node{Title: c.Name, Body: strings.Join(parts, "\n\n")}
		}
		tbl.Bodies = append(tbl.Bodies, body)
	}
	title := strings.TrimSpace(t.Name)
	if title == "" {
		title = stripExt(name)
	}
	b.Roots = append(b.Roots, &Node{Title: title, Table: tbl})
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

var projectHeaders = []string{"task id", "name", "section/column"}

func looksLikeProjectCSV(headers []string) bool {
	have := map[string]bool{}
	for _, h := range headers {
		have[strings.ToLower(strings.TrimSpace(h))] = true
	}
	for _, h := range projectHeaders {
		if !have[h] {
			return false
		}
	}
	return true
}

func importProjectCSV(name string, headers []string, rows [][]string, b *Bundle) {
	idx := map[string]int{}
	for i, h := range headers {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(r []string, key string) string {
		if i, ok := idx[key]; ok && i < len(r) {
			return strings.TrimSpace(r[i])
		}
		return ""
	}
	cols := []struct{ key, name, typ string }{
		{"name", "Name", "title"}, {"section/column", "Section", "select"}, {"assignee", "Assignee", "select"},
		{"start date", "Start date", "date"}, {"due date", "Due date", "date"}, {"tags", "Tags", "multi_select"},
		{"created at", "Created", "date"}, {"completed at", "Completed", "date"}, {"projects", "Projects", "multi_select"},
		{"parent task", "Parent task", "text"},
	}
	tbl := &Table{Board: "Section"}
	for _, c := range cols {
		if _, ok := idx[c.key]; ok || c.key == "name" {
			tbl.Headers = append(tbl.Headers, c.name)
			tbl.Types = append(tbl.Types, c.typ)
		}
	}
	tbl.Headers = append(tbl.Headers, "Done")
	tbl.Types = append(tbl.Types, "checkbox")
	for _, r := range rows {
		var row []string
		for _, c := range cols {
			if _, ok := idx[c.key]; ok || c.key == "name" {
				row = append(row, get(r, c.key))
			}
		}
		row = append(row, yesNo(get(r, "completed at") != ""))
		tbl.Rows = append(tbl.Rows, row)
		var body *Node
		if notes := get(r, "notes"); notes != "" {
			body = &Node{Title: get(r, "name"), Body: textToMarkdown(notes)}
		}
		tbl.Bodies = append(tbl.Bodies, body)
	}
	b.Roots = append(b.Roots, &Node{Title: stripExt(name), Table: tbl})
}

func importCSVFile(name string, data []byte, b *Bundle) {
	headers, rows := readCSVBytes(data)
	if len(headers) == 0 {
		b.Warn("The file " + name + " has no header row")
		return
	}
	if looksLikeProjectCSV(headers) {
		importProjectCSV(name, headers, rows, b)
		return
	}
	b.Roots = append(b.Roots, &Node{Title: stripExt(name), Table: &Table{Headers: headers, Rows: rows}})
}

func importHTMLFile(name string, data []byte, b *Bundle) {
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		b.Warn("The file " + name + " could not be read")
		return
	}
	n := &Node{Title: findTitle(root)}
	if n.Title == "" {
		n.Title = stripExt(name)
	}
	n.Body = htmlToMarkdown(mainContent(root), htmlOpts{media: func(src, _ string) (string, bool) {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			return src, true
		}
		return "", false
	}}, b.Warn)
	n.Body = dropLeadingHeading(n.Body, n.Title)
	b.Roots = append(b.Roots, n)
}

func importMarkdownFile(name string, data []byte, b *Bundle) {
	src := normalizeText(data)
	src, fm := frontMatter(src)
	n := &Node{Title: stripExt(name)}
	if fm != "" {
		n.Title = fm
	}
	if h1, rest, ok := leadingH1(src); ok {
		n.Title = h1
		src = rest
	}
	n.Body = src
	b.Roots = append(b.Roots, n)
}

func importText(name string, data []byte, b *Bundle) {
	b.Roots = append(b.Roots, &Node{Title: stripExt(name), Body: textToMarkdown(normalizeText(data))})
}

var spaceTitle = regexp.MustCompile(`^[^:]{1,80} : `)

func looksLikeSpaceExport(v *vfs) bool {
	for _, f := range v.files {
		if path.Base(f.path) == "index.html" {
			data, err := v.read(f)
			if err == nil && bytes.Contains(data, []byte("main-content")) && bytes.Contains(data, []byte("Available Pages")) {
				return true
			}
		}
	}
	return false
}

func importSpaceExport(v *vfs, b *Bundle) {
	var index *vfile
	for _, f := range v.sorted() {
		if path.Base(f.path) == "index.html" && (index == nil || len(f.path) < len(index.path)) {
			index = f
		}
	}
	t := &treeImporter{v: v, b: b, titles: true, docs: map[string]*Node{}, owners: map[string]*Node{}, tables: map[*Node]string{}, assets: map[string]*Asset{}, kids: map[*Node][]childAt{}}
	files := v.sorted()
	for _, f := range files {
		if e := extOf(f.path); (e == ".html" || e == ".htm") && f != index {
			t.docs[strings.ToLower(f.path)] = &Node{Title: stripExt(path.Base(f.path))}
		}
	}
	placed := map[*Node]bool{}
	var roots []*Node
	if index != nil {
		data, _ := v.read(index)
		root, err := html.Parse(bytes.NewReader(data))
		if err == nil {
			dir := path.Dir(index.path)
			var walkList func(ul *html.Node, parent *Node)
			walkList = func(ul *html.Node, parent *Node) {
				for li := ul.FirstChild; li != nil; li = li.NextSibling {
					if tag(li) != "li" {
						continue
					}
					var cur *Node
					for k := li.FirstChild; k != nil; k = k.NextSibling {
						if tag(k) == "a" && cur == nil {
							if n, ok := t.docs[strings.ToLower(cleanPath(path.Join(dir, attr(k, "href"))))]; ok && !placed[n] {
								cur = n
								placed[n] = true
								if parent == nil {
									roots = append(roots, n)
								} else {
									parent.Children = append(parent.Children, n)
								}
							}
						}
						if tag(k) == "ul" {
							p := cur
							if p == nil {
								p = parent
							}
							walkList(k, p)
						}
					}
				}
			}
			var find func(*html.Node)
			find = func(n *html.Node) {
				if tag(n) == "ul" {
					walkList(n, nil)
					return
				}
				for k := n.FirstChild; k != nil; k = k.NextSibling {
					find(k)
				}
			}
			find(mainContent(root))
		}
	}
	for _, f := range files {
		n, ok := t.docs[strings.ToLower(f.path)]
		if !ok {
			continue
		}
		data, err := v.read(f)
		if err != nil {
			continue
		}
		t.convert(f.path, data, n)
		n.Title = spaceTitle.ReplaceAllString(n.Title, "")
		n.Body = dropLeadingHeading(n.Body, n.Title)
		if !placed[n] {
			roots = append(roots, n)
		}
	}
	b.Roots = append(b.Roots, roots...)
}
