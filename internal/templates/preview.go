package templates

import (
	"bytes"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type PreviewItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Preview struct {
	ID          string        `json:"id"`
	Kind        string        `json:"kind"`
	Title       string        `json:"title"`
	Icon        string        `json:"icon"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	HTML        string        `json:"html"`
	Sections    []string      `json:"sections,omitempty"`
	Views       []PreviewItem `json:"views,omitempty"`
	Properties  []PreviewItem `json:"properties,omitempty"`
	Rows        int           `json:"rows,omitempty"`
	Featured    bool          `json:"featured,omitempty"`
	Pages       int           `json:"pages,omitempty"`
	Databases   int           `json:"databases,omitempty"`
	Tree        []PreviewNode `json:"tree,omitempty"`
}

var (
	previewMu    sync.Mutex
	previewCache = map[string]*Preview{}
)

func findStarter(id string) *Starter {
	for i := range starters {
		if starters[i].ID == id {
			return &starters[i]
		}
	}
	return nil
}

func previewFor(id string) *Preview {
	st := findStarter(id)
	if st == nil {
		return nil
	}
	now := time.Now().UTC()
	key := id + "|" + now.Format("2006-01-02")
	previewMu.Lock()
	defer previewMu.Unlock()
	if p, ok := previewCache[key]; ok {
		return p
	}
	for k := range previewCache {
		if strings.HasPrefix(k, id+"|") {
			delete(previewCache, k)
		}
	}
	p := &Preview{ID: st.ID, Kind: st.Kind, Title: st.Name, Icon: st.Icon, Description: st.Description, Category: st.Category, Featured: st.Featured}
	t := newTree(st, now)
	if len(st.children) > 0 {
		p.Tree, p.Pages, p.Databases, p.Rows = treeOf(st)
	}
	if st.Kind == "database" {
		p.HTML = databaseHTML(t, st, now)
		for _, v := range st.views {
			p.Views = append(p.Views, PreviewItem{Name: v.Name, Type: v.Type})
		}
		for _, sp := range st.schema {
			p.Properties = append(p.Properties, PreviewItem{Name: sp.Name, Type: sp.Type})
		}
		p.Rows = len(st.rows)
	} else {
		p.HTML, p.Sections = t.docHTML(st, st.markdown)
	}
	previewCache[key] = p
	return p
}

type previewBlocks struct{}

func (previewBlocks) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, renderPreviewBlock)
}

var (
	summaryTag = regexp.MustCompile(`(?is)<summary>(.*?)</summary>`)
	anyTag     = regexp.MustCompile(`<[^>]*>`)
	calloutRE  = regexp.MustCompile(`<blockquote>\s*<p>\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*`)
	h2RE       = regexp.MustCompile(`(?s)<h2[^>]*>(.*?)</h2>`)
	tocMarker  = `<div class="tplp-toc"></div>`
	tones      = map[string]string{"NOTE": "blue", "TIP": "green", "IMPORTANT": "violet", "WARNING": "amber", "CAUTION": "rose"}
)

func renderPreviewBlock(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	var raw strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		raw.Write(seg.Value(src))
	}
	if hb, ok := n.(*ast.HTMLBlock); ok && hb.HasClosure() {
		raw.Write(hb.ClosureLine.Value(src))
	}
	s := strings.TrimSpace(raw.String())
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "<details"):
		summary := ""
		if m := summaryTag.FindStringSubmatch(s); m != nil {
			summary = html.UnescapeString(anyTag.ReplaceAllString(m[1], ""))
		}
		_, _ = w.WriteString(`<details class="tplp-toggle"><summary>` + html.EscapeString(summary) + "</summary>\n")
		if strings.Contains(lower, "</details>") {
			_, _ = w.WriteString("</details>\n")
		}
	case strings.HasPrefix(lower, "</details"):
		_, _ = w.WriteString("</details>\n")
	case lower == "<!-- toc -->":
		_, _ = w.WriteString(tocMarker + "\n")
	}
	return ast.WalkContinue, nil
}

var previewMD = goldmark.New(
	goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.TaskList),
	goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(previewBlocks{}, 100))),
)

func markdownHTML(md string) (string, []string) {
	var buf bytes.Buffer
	if err := previewMD.Convert([]byte(md), &buf); err != nil {
		return "", nil
	}
	out := calloutRE.ReplaceAllStringFunc(buf.String(), func(m string) string {
		sub := calloutRE.FindStringSubmatch(m)
		return `<blockquote class="tplp-callout" data-tone="` + tones[sub[1]] + `"><p>`
	})
	var sections []string
	for _, m := range h2RE.FindAllStringSubmatch(out, -1) {
		sections = append(sections, html.UnescapeString(anyTag.ReplaceAllString(m[1], "")))
	}
	if strings.Contains(out, tocMarker) {
		var toc strings.Builder
		toc.WriteString(`<div class="tplp-toc">`)
		for _, s := range sections {
			toc.WriteString("<p>" + html.EscapeString(s) + "</p>")
		}
		toc.WriteString("</div>")
		out = strings.Replace(out, tocMarker, toc.String(), 1)
	}
	return out, sections
}

type viewConfig struct {
	Group *struct {
		Prop string `json:"prop"`
	} `json:"group"`
}

func esc(s string) string { return html.EscapeString(s) }

func optionChip(o opt) string {
	return `<span class="tplp-chip" data-c="` + esc(o.Color) + `">` + esc(o.Name) + `</span>`
}

func findOpt(sp prop, id string) (opt, bool) {
	for _, o := range sp.Options {
		if o.ID == id {
			return o, true
		}
	}
	return opt{}, false
}

func formatNumber(f float64, format string) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if format == "dollar" || format == "comma" {
		whole, frac, _ := strings.Cut(s, ".")
		neg := strings.HasPrefix(whole, "-")
		whole = strings.TrimPrefix(whole, "-")
		var b strings.Builder
		for i, r := range whole {
			if i > 0 && (len(whole)-i)%3 == 0 {
				b.WriteByte(',')
			}
			b.WriteRune(r)
		}
		s = b.String()
		if frac != "" {
			s += "." + frac
		}
		if format == "dollar" {
			s = "$" + s
		}
		if neg {
			s = "-" + s
		}
	}
	return s
}

func cellHTML(sp prop, v any, idx int, now time.Time) string {
	if sp.Type == "unique_id" {
		id := strconv.Itoa(idx + 1)
		if sp.Prefix != "" {
			id = sp.Prefix + "-" + id
		}
		return esc(id)
	}
	if v == nil {
		return ""
	}
	switch sp.Type {
	case "select", "status":
		if id, ok := v.(string); ok {
			if o, ok := findOpt(sp, id); ok {
				return optionChip(o)
			}
		}
	case "multi_select":
		if ids, ok := v.([]string); ok {
			var b strings.Builder
			for _, id := range ids {
				if o, ok := findOpt(sp, id); ok {
					b.WriteString(optionChip(o))
				}
			}
			return b.String()
		}
	case "date":
		if s, ok := v.(string); ok {
			if d, ok := dayOffset(s, now); ok {
				t, _ := time.Parse("2006-01-02", d)
				return esc(t.Format("Jan 2"))
			}
		}
	case "person":
		return `<span class="tplp-avatar"></span>`
	case "checkbox":
		if b, ok := v.(bool); ok && b {
			return `<span class="tplp-check" data-on="true"></span>`
		}
		return `<span class="tplp-check"></span>`
	case "number":
		switch n := v.(type) {
		case int:
			return esc(formatNumber(float64(n), sp.Format))
		case float64:
			return esc(formatNumber(n, sp.Format))
		}
	case "url":
		if s, ok := v.(string); ok {
			if u, err := url.Parse(s); err == nil && u.Host != "" {
				return `<span class="tplp-link">` + esc(u.Host) + `</span>`
			}
		}
	case "place":
		if m, ok := v.(map[string]any); ok {
			if name, ok := m["name"].(string); ok {
				return esc(name)
			}
		}
	}
	if s, ok := v.(string); ok {
		return esc(s)
	}
	return ""
}

func databaseHTML(t *pvTree, st *Starter, now time.Time) string {
	var b strings.Builder
	b.WriteString(`<div class="tplp-tabs">`)
	for i, v := range st.views {
		if i == 6 {
			b.WriteString(`<span>+` + strconv.Itoa(len(st.views)-6) + `</span>`)
			break
		}
		cls := ""
		if i == 0 {
			cls = ` class="is-on"`
		}
		b.WriteString(`<span` + cls + ` data-view="` + esc(v.Type) + `">` + esc(v.Name) + `</span>`)
	}
	b.WriteString(`</div>`)
	first := st.views[0]
	if visualViews[first.Type] {
		if out := t.visualHTML(t.db(st), first); out != "" {
			b.WriteString(out)
			return b.String()
		}
	}
	var cfg viewConfig
	_ = json.Unmarshal([]byte(first.Config), &cfg)
	var title prop
	var rest []prop
	for _, sp := range st.schema {
		if sp.Type == "title" {
			title = sp
		} else {
			rest = append(rest, sp)
		}
	}
	if first.Type == "board" && cfg.Group != nil {
		var group *prop
		for i := range st.schema {
			if st.schema[i].ID == cfg.Group.Prop {
				group = &st.schema[i]
			}
		}
		if group != nil && len(group.Options) > 0 {
			b.WriteString(boardHTML(st, *group, rest, now))
			return b.String()
		}
	}
	cols := rest
	if len(cols) > 4 {
		cols = cols[:4]
	}
	b.WriteString(`<table class="tplp-table"><thead><tr><th>` + esc(title.Name) + `</th>`)
	for _, sp := range cols {
		b.WriteString(`<th>` + esc(sp.Name) + `</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for i, r := range st.rows {
		if i == 12 {
			break
		}
		b.WriteString(`<tr><td class="tplp-title">` + esc(r.Title) + `</td>`)
		for _, sp := range cols {
			b.WriteString(`<td>` + t.cellOf(t.db(st), i, sp, now) + `</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func cardProps(props []prop, r row, idx int, now time.Time, limit int) string {
	var b strings.Builder
	n := 0
	for _, sp := range props {
		if n >= limit {
			break
		}
		c := cellHTML(sp, r.Values[sp.ID], idx, now)
		if c == "" || sp.Type == "checkbox" {
			continue
		}
		b.WriteString(`<div class="tplp-card__prop">` + c + `</div>`)
		n++
	}
	return b.String()
}

func boardHTML(st *Starter, group prop, rest []prop, now time.Time) string {
	var others []prop
	for _, sp := range rest {
		if sp.ID != group.ID {
			others = append(others, sp)
		}
	}
	var b strings.Builder
	b.WriteString(`<div class="tplp-board">`)
	for _, o := range group.Options {
		b.WriteString(`<div class="tplp-col"><div class="tplp-col__head">` + optionChip(o) + `</div>`)
		shown := 0
		for i, r := range st.rows {
			if v, _ := r.Values[group.ID].(string); v != o.ID {
				continue
			}
			if shown++; shown > 4 {
				continue
			}
			b.WriteString(`<div class="tplp-card"><div class="tplp-card__title">` + esc(r.Title) + `</div>`)
			b.WriteString(cardProps(others, r, i, now, 2))
			b.WriteString(`</div>`)
		}
		if shown > 4 {
			b.WriteString(`<div class="tplp-col__more">+` + strconv.Itoa(shown-4) + `</div>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}
