package site

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/page"
)

type assets struct {
	css    string
	js     string
	font   string
	glyphs map[string]string
	icons  map[string]string
}

func loadAssets(dist fs.FS) *assets {
	a := &assets{glyphs: map[string]string{}, icons: map[string]string{}}
	if dist == nil {
		return a
	}
	var man map[string]struct {
		File   string   `json:"file"`
		CSS    []string `json:"css"`
		Assets []string `json:"assets"`
	}
	if raw, err := fs.ReadFile(dist, ".vite/manifest.json"); err == nil && json.Unmarshal(raw, &man) == nil {
		if e, ok := man["src/site/main.ts"]; ok {
			a.js = "/" + e.File
			if len(e.CSS) > 0 {
				a.css = "/" + e.CSS[0]
			}
			for _, f := range e.Assets {
				if strings.Contains(f, "inter-latin-wght-normal") {
					a.font = "/" + f
				}
			}
		}
	}
	var table struct {
		Glyphs map[string]string `json:"glyphs"`
		Icons  map[string]string `json:"icons"`
	}
	if raw, err := fs.ReadFile(dist, ".vite/icons.json"); err == nil && json.Unmarshal(raw, &table) == nil {
		a.glyphs, a.icons = table.Glyphs, table.Icons
	}
	return a
}

const themeScript = `(function(){var d=document.documentElement;if(d.getAttribute("data-theme")==="auto")d.setAttribute("data-theme",matchMedia("(prefers-color-scheme: light)").matches?"light":"dark")})()`

var siteCSP = func() string {
	sum := sha256.Sum256([]byte(themeScript))
	return strings.Join([]string{
		"default-src 'none'",
		"script-src 'self' 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'",
		"style-src 'self'",
		"img-src 'self' data:",
		"media-src 'self'",
		"font-src 'self'",
		"frame-src https:",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}()

type node struct {
	ID          string
	Parent      string
	Title       string
	Icon        string
	Cover       string
	Kind        string
	Description string
	Snippet     string
	SortKey     string
	Row         bool
	Inline      bool
	UpdatedAt   int64
}

type scope struct {
	site     *Site
	link     *page.LinkAccess
	ws       string
	root     string
	base     string
	origin   string
	level    string
	indexing bool
	theme    string
	font     string
	pages    map[string]*node
	kids     map[string][]*node
}

func (s *Service) load(ctx context.Context, sc *scope, subpages bool) error {
	rows, err := s.st.R.QueryContext(ctx, `WITH RECURSIVE tree(id, depth) AS (
			SELECT id, 0 FROM pages WHERE id = ? AND trashed_at = 0
			UNION ALL
			SELECT p.id, t.depth + 1 FROM pages p JOIN tree t ON p.parent_id = t.id LEFT JOIN collections c ON c.page_id = p.id
			WHERE p.trashed_at = 0 AND p.visibility = 'workspace' AND t.depth < 64 AND (? OR p.is_row = 1 OR c.inline = 1)
		)
		SELECT p.id, COALESCE(p.parent_id, ''), p.title, p.icon, p.cover, p.kind, p.description, p.snippet, p.sort_key, p.is_row, COALESCE(c.inline, 0), p.updated_at
		FROM tree t JOIN pages p ON p.id = t.id LEFT JOIN collections c ON c.page_id = p.id LIMIT 5000`, sc.root, subpages)
	if err != nil {
		return err
	}
	defer rows.Close()
	sc.pages, sc.kids = map[string]*node{}, map[string][]*node{}
	for rows.Next() {
		var n node
		var row, inline int
		if err := rows.Scan(&n.ID, &n.Parent, &n.Title, &n.Icon, &n.Cover, &n.Kind, &n.Description, &n.Snippet, &n.SortKey, &row, &inline, &n.UpdatedAt); err != nil {
			return err
		}
		n.Row, n.Inline = row == 1, inline == 1
		sc.pages[n.ID] = &n
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, n := range sc.pages {
		if n.ID != sc.root {
			sc.kids[n.Parent] = append(sc.kids[n.Parent], n)
		}
	}
	for _, list := range sc.kids {
		sort.Slice(list, func(i, j int) bool { return list[i].SortKey < list[j].SortKey })
	}
	return nil
}

func (sc *scope) href(id string) string {
	if _, ok := sc.pages[id]; !ok {
		return ""
	}
	if id == sc.root {
		if sc.base == "" {
			return "/"
		}
		return sc.base
	}
	return sc.base + "/" + id
}

func (sc *scope) fileHref(id string) string { return sc.base + "/files/" + id }

func esc(v string) string { return html.EscapeString(v) }

func (s *Service) glyph(name string) string {
	if g, ok := s.assets.glyphs[name]; ok {
		return g
	}
	return s.assets.glyphs["page"]
}

func (s *Service) iconSpan(sc *scope, ref, fallback, class string) string {
	cls := ""
	if class != "" {
		cls = ` class="` + esc(class) + `"`
	}
	if id, ok := strings.CutPrefix(ref, "f:"); ok {
		return `<span` + cls + ` data-kind="picture"><img src="` + esc(sc.fileHref(id)) + `" alt="" decoding="async"></span>`
	}
	if rest, ok := strings.CutPrefix(ref, "i:"); ok {
		name, tint, _ := strings.Cut(rest, ":")
		color := ""
		if tint != "" {
			color = ` data-color="` + esc(tint) + `"`
		}
		return `<span` + cls + ` data-kind="glyph"` + color + `>` + s.glyph(name) + `</span>`
	}
	return `<span` + cls + ` data-kind="default">` + s.glyph(fallback) + `</span>`
}

func (s *Service) pageIcon(sc *scope, id, class string) string {
	n := sc.pages[id]
	if n == nil {
		return s.iconSpan(sc, "", "page", class)
	}
	fb := "page"
	if n.Kind == "collection" {
		fb = "table"
	}
	return s.iconSpan(sc, n.Icon, fb, class)
}

func formatDate(ref string) string {
	if t, err := time.Parse("2006-01-02T15:04", ref); err == nil {
		return t.Format("Jan 2, 2006, 3:04 PM")
	}
	if t, err := time.Parse("2006-01-02", ref); err == nil {
		return t.Format("Jan 2, 2006")
	}
	return ref
}

func (s *Service) table(ctx context.Context, sc *scope, collectionID, viewID string, heading bool) string {
	if _, ok := sc.pages[collectionID]; !ok {
		return ""
	}
	t, err := s.cols.PublicTable(ctx, collectionID, viewID, 500)
	if err != nil {
		return ""
	}
	return s.tableHTML(sc, t, heading)
}

func (s *Service) tableHTML(sc *scope, t *collection.PublicTable, heading bool) string {
	var b strings.Builder
	b.WriteString(`<section class="site-table">`)
	if heading {
		title := t.Title
		if title == "" {
			title = "Untitled"
		}
		b.WriteString(`<div class="site-table__title">` + esc(title) + `</div>`)
	}
	b.WriteString(`<div class="site-table__scroll"><table><thead><tr>`)
	for _, c := range t.Columns {
		b.WriteString(`<th scope="col">` + esc(c) + `</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, r := range t.Rows {
		b.WriteString(`<tr>`)
		for i, c := range r.Cells {
			if i == 0 {
				label := c
				if label == "" {
					label = "Untitled"
				}
				if u := sc.href(r.ID); u != "" {
					b.WriteString(`<td><a href="` + esc(u) + `">` + s.pageIcon(sc, r.ID, "site-icon") + `<span>` + esc(label) + `</span></a></td>`)
				} else {
					b.WriteString(`<td>` + esc(label) + `</td>`)
				}
				continue
			}
			b.WriteString(`<td>` + esc(c) + `</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	if t.Total > len(t.Rows) {
		b.WriteString(`<p class="site-table__more">Showing ` + strconv.Itoa(len(t.Rows)) + ` of ` + strconv.Itoa(t.Total) + ` rows.</p>`)
	}
	b.WriteString(`</section>`)
	return b.String()
}

func (s *Service) public(ctx context.Context, sc *scope) *doc.Public {
	return &doc.Public{
		Page: sc.href,
		Title: func(id string) (string, bool) {
			n, ok := sc.pages[id]
			if !ok {
				return "", false
			}
			return n.Title, true
		},
		PageIcon: func(id, class string) string { return s.pageIcon(sc, id, class) },
		File:     sc.fileHref,
		Icon:     func(ref, fallback, class string) string { return s.iconSpan(sc, ref, fallback, class) },
		UI:       func(name string) string { return s.assets.icons[name] },
		Table:    func(collectionID, viewID string) string { return s.table(ctx, sc, collectionID, viewID, true) },
		Date:     formatDate,
	}
}

func (s *Service) head(b *strings.Builder, sc *scope, title, desc, canonical, image string) {
	theme := sc.theme
	if theme == "" {
		theme = "auto"
	}
	b.WriteString(`<!doctype html><html lang="en" data-theme="` + esc(theme) + `" data-font="` + esc(sc.font) + `"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1"><meta name="color-scheme" content="dark light">`)
	b.WriteString(`<title>` + esc(title) + `</title>`)
	if desc != "" {
		b.WriteString(`<meta name="description" content="` + esc(desc) + `">`)
	}
	if !sc.indexing {
		b.WriteString(`<meta name="robots" content="noindex, nofollow">`)
	}
	if canonical != "" {
		b.WriteString(`<link rel="canonical" href="` + esc(canonical) + `">`)
		b.WriteString(`<meta property="og:url" content="` + esc(canonical) + `">`)
	}
	b.WriteString(`<meta property="og:type" content="article"><meta property="og:title" content="` + esc(title) + `">`)
	if desc != "" {
		b.WriteString(`<meta property="og:description" content="` + esc(desc) + `">`)
	}
	if image != "" {
		b.WriteString(`<meta property="og:image" content="` + esc(image) + `"><meta name="twitter:card" content="summary_large_image">`)
	} else {
		b.WriteString(`<meta name="twitter:card" content="summary">`)
	}
	b.WriteString(`<script>` + themeScript + `</script>`)
	if s.assets.font != "" {
		b.WriteString(`<link rel="preload" href="` + esc(s.assets.font) + `" as="font" type="font/woff2" crossorigin>`)
	}
	if s.assets.css != "" {
		b.WriteString(`<link rel="stylesheet" href="` + esc(s.assets.css) + `">`)
	}
	if s.assets.js != "" {
		b.WriteString(`<script type="module" src="` + esc(s.assets.js) + `"></script>`)
	}
	b.WriteString(`<link rel="icon" href="/favicon.svg" type="image/svg+xml"></head>`)
}

func (s *Service) bar(b *strings.Builder, sc *scope, id string) {
	var chain []*node
	for n := sc.pages[id]; n != nil && len(chain) < 64; n = sc.pages[n.Parent] {
		chain = append([]*node{n}, chain...)
		if n.ID == sc.root {
			break
		}
	}
	b.WriteString(`<header class="site-bar"><nav class="site-crumbs" aria-label="Breadcrumb">`)
	for i, n := range chain {
		if i > 0 {
			b.WriteString(`<span class="site-sep" aria-hidden="true">/</span>`)
		}
		label := n.Title
		if label == "" {
			label = "Untitled"
		}
		if i == len(chain)-1 {
			b.WriteString(`<span class="site-crumb" aria-current="page">` + s.pageIcon(sc, n.ID, "site-icon") + `<span>` + esc(label) + `</span></span>`)
			continue
		}
		b.WriteString(`<a class="site-crumb" href="` + esc(sc.href(n.ID)) + `">` + s.pageIcon(sc, n.ID, "site-icon") + `<span>` + esc(label) + `</span></a>`)
	}
	b.WriteString(`</nav><div class="site-bar__end">`)
	if sc.link != nil {
		label := "Sign in"
		if sc.level == "comment" {
			label = "Sign in to comment"
		}
		b.WriteString(`<a href="/login?next=` + esc(url.QueryEscape(sc.base+"/"+strings.TrimPrefix(id, sc.root))) + `">` + label + `</a>`)
	}
	b.WriteString(`</div></header>`)
}

func (s *Service) renderPage(ctx context.Context, sc *scope, id string) (string, error) {
	n := sc.pages[id]
	var body string
	var err error
	if n.Kind == "doc" {
		body, err = s.docs.RenderPublic(ctx, id, s.public(ctx, sc))
		if err != nil {
			return "", err
		}
	}
	title := n.Title
	if title == "" {
		title = "Untitled"
	}
	pageTitle := title
	desc := n.Description
	if desc == "" {
		desc = n.Snippet
	}
	image := ""
	if sc.site != nil {
		if id == sc.root {
			if sc.site.Title != "" {
				pageTitle = sc.site.Title
			}
			if sc.site.Description != "" {
				desc = sc.site.Description
			}
		} else if root := sc.pages[sc.root]; root != nil {
			name := sc.site.Title
			if name == "" {
				name = root.Title
			}
			if name != "" {
				pageTitle = title + " · " + name
			}
		}
		if f, ok := strings.CutPrefix(sc.site.Image, "f:"); ok {
			image = sc.origin + sc.fileHref(f)
		}
	}
	if image == "" {
		if f, ok := strings.CutPrefix(n.Cover, "f:"); ok {
			image = sc.origin + sc.fileHref(f)
		}
	}
	canonical := ""
	if sc.site != nil {
		canonical = sc.origin + sc.href(id)
	}
	var b strings.Builder
	s.head(&b, sc, pageTitle, clip(desc, 200), canonical, image)
	cls := "site"
	if n.Cover != "" {
		cls += " has-cover"
	}
	b.WriteString(`<body class="` + cls + `">`)
	s.bar(&b, sc, id)
	if g, ok := strings.CutPrefix(n.Cover, "g:"); ok {
		b.WriteString(`<div class="site-cover" data-cover="` + esc(g) + `"></div>`)
	} else if f, ok := strings.CutPrefix(n.Cover, "f:"); ok {
		b.WriteString(`<div class="site-cover"><img src="` + esc(sc.fileHref(f)) + `" alt=""></div>`)
	}
	wide := ""
	if n.Kind == "collection" {
		wide = " data-wide"
	}
	b.WriteString(`<main class="site-main"` + wide + `><header class="site-head">`)
	if n.Icon != "" {
		b.WriteString(s.pageIcon(sc, id, "site-icon"))
	}
	b.WriteString(`<h1 class="site-title">` + esc(title) + `</h1>`)
	if n.Description != "" {
		b.WriteString(`<p class="site-desc">` + esc(n.Description) + `</p>`)
	}
	b.WriteString(`</header>`)
	if n.Row {
		if props, err := s.cols.PublicProps(ctx, id); err == nil && len(props) > 0 {
			b.WriteString(`<dl class="site-props">`)
			for _, p := range props {
				b.WriteString(`<dt>` + esc(p.Name) + `</dt><dd>` + esc(p.Value) + `</dd>`)
			}
			b.WriteString(`</dl>`)
		}
	}
	if n.Kind == "collection" {
		b.WriteString(s.table(ctx, sc, id, "", false))
	} else {
		b.WriteString(`<article class="doc-body">` + body + `</article>`)
	}
	var subs []*node
	for _, k := range sc.kids[id] {
		if !k.Row && !k.Inline {
			subs = append(subs, k)
		}
	}
	if len(subs) > 0 {
		b.WriteString(`<nav class="site-children" aria-label="Pages"><div class="site-children__head">Pages</div>`)
		for _, k := range subs {
			label := k.Title
			if label == "" {
				label = "Untitled"
			}
			b.WriteString(`<a class="site-child" href="` + esc(sc.href(k.ID)) + `">` + s.pageIcon(sc, k.ID, "site-icon") + `<span>` + esc(label) + `</span></a>`)
		}
		b.WriteString(`</nav>`)
	}
	b.WriteString(`</main><footer class="site-foot">Updated ` + esc(time.UnixMilli(n.UpdatedAt).UTC().Format("Jan 2, 2006")) + ` · Published with form.md</footer></body></html>`)
	return b.String(), nil
}

func (s *Service) missing(sc *scope) string {
	var b strings.Builder
	if sc == nil {
		sc = &scope{theme: "auto", font: "sans"}
	}
	s.head(&b, sc, "Page not found", "", "", "")
	b.WriteString(`<body class="site"><main class="site-missing"><h1 class="site-title">Nothing here</h1><p class="site-desc">This page is not published, or the link has ended.</p></main></body></html>`)
	return b.String()
}

func (s *Service) gatePage(sc *scope, action, next string, failed bool) string {
	var b strings.Builder
	s.head(&b, sc, "Password required", "", "", "")
	b.WriteString(`<body class="site"><main class="site-gate"><form class="site-gate__card" method="post" action="` + esc(action) + `">`)
	b.WriteString(`<h1 class="site-gate__title">This page is protected</h1><p>Enter the password you were given to read it.</p>`)
	if failed {
		b.WriteString(`<p class="site-gate__error" role="alert">That password is not right.</p>`)
	}
	b.WriteString(`<input type="password" name="password" autocomplete="current-password" aria-label="Password" required autofocus>`)
	b.WriteString(`<input type="hidden" name="next" value="` + esc(next) + `"><button type="submit">Continue</button></form></main></body></html>`)
	return b.String()
}

func (s *Service) unlockToken(kind, id string, rev int64) string {
	key := s.set.Get("sites.key")
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(kind + ":" + id + ":" + strconv.FormatInt(rev, 10)))
	return hex.EncodeToString(m.Sum(nil))
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", siteCSP)
	h.Set("Cache-Control", "no-store")
	h.Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
