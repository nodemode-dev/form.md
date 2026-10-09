package doc

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/reearth/ygo/crdt"
)

type htmlWriter struct {
	b       strings.Builder
	title   func(id string) string
	heads   []tocEntry
	crumbs  []page.Crumb
	nested  bool
	self    string
	local   func(name string) *crdt.YXmlFragment
	foreign map[syncedRef]string
	pub     *Public
}

func (r *room) render(title func(id string) string, crumbs []page.Crumb, foreign map[syncedRef]string) string {
	w := &htmlWriter{title: title, crumbs: crumbs, self: r.id, local: r.doc.GetXmlFragment, foreign: foreign}
	frag := r.fragment()
	w.collectHeadings(children(frag))
	for _, c := range children(frag) {
		if el, ok := c.(*crdt.YXmlElement); ok {
			w.node(el)
		}
	}
	return w.b.String()
}

type tocEntry struct {
	id    string
	level int
	text  string
}

func RenderHTML(frag *crdt.YXmlFragment, title func(id string) string) string {
	return RenderPage(frag, title, nil)
}

func RenderPage(frag *crdt.YXmlFragment, title func(id string) string, crumbs []page.Crumb) string {
	w := &htmlWriter{title: title, crumbs: crumbs}
	w.collectHeadings(children(frag))
	for _, c := range children(frag) {
		if el, ok := c.(*crdt.YXmlElement); ok {
			w.node(el)
		}
	}
	return w.b.String()
}

func (w *htmlWriter) collectHeadings(nodes []any) {
	for _, n := range nodes {
		el, ok := n.(*crdt.YXmlElement)
		if !ok {
			continue
		}
		if el.NodeName == "heading" {
			sw := &mdWriter{pages: map[string]bool{}, files: map[string]bool{}, users: map[string]bool{}, title: w.title}
			sw.inline(el, false)
			w.heads = append(w.heads, tocEntry{id: attrString(el, "id"), level: attrInt(el, "level", 1), text: strings.TrimSpace(sw.text.String())})
			continue
		}
		if !textblocks[el.NodeName] && !atoms[el.NodeName] {
			w.collectHeadings(children(el))
		}
	}
}

func esc(s string) string { return html.EscapeString(s) }

func (w *htmlWriter) open(tag string, attrs ...string) {
	w.b.WriteString("<" + tag)
	for i := 0; i+1 < len(attrs); i += 2 {
		if attrs[i+1] == "" && attrs[i] != "alt" {
			continue
		}
		w.b.WriteString(" " + attrs[i] + `="` + esc(attrs[i+1]) + `"`)
	}
	w.b.WriteString(">")
}

func (w *htmlWriter) remoteMedia(el *crdt.YXmlElement, id string) {
	src := attrString(el, "src")
	host := src
	if u, err := url.Parse(src); err == nil && u.Host != "" {
		host = u.Host
	}
	if w.pub != nil {
		w.open("figure", "class", "media media--"+el.NodeName, "data-block", id)
		w.open("a", "class", "embed__fallback", "href", safeHref(src), "target", "_blank", "rel", "noopener noreferrer nofollow")
		w.b.WriteString(esc(host) + "</a>")
		w.caption(attrString(el, "caption"))
		w.b.WriteString("</figure>")
		return
	}
	w.open("figure", "class", "media media--"+el.NodeName+" is-remote", "data-block", id, "data-src", src)
	w.b.WriteString(`<div class="media__pending"><span class="media__pending-text">` + esc(host) + `</span></div>`)
	w.caption(attrString(el, "caption"))
	w.b.WriteString("</figure>")
}

func (w *htmlWriter) kids(el *crdt.YXmlElement) {
	for _, c := range children(el) {
		if k, ok := c.(*crdt.YXmlElement); ok {
			w.node(k)
		}
	}
}

func (w *htmlWriter) node(el *crdt.YXmlElement) {
	id := attrString(el, "id")
	if (el.NodeName == "image" || el.NodeName == "video" || el.NodeName == "audio") && fileID(attrString(el, "src")) == "" {
		w.remoteMedia(el, id)
		return
	}
	switch el.NodeName {
	case "paragraph":
		w.open("p", "data-block", id)
		w.inline(el)
		w.b.WriteString("</p>")
	case "heading":
		lvl := attrInt(el, "level", 1)
		if lvl < 1 || lvl > 6 {
			lvl = 1
		}
		tag := "h" + strconv.Itoa(lvl)
		w.open(tag, "data-block", id)
		w.inline(el)
		w.b.WriteString("</" + tag + ">")
	case "blockquote":
		w.open("blockquote", "data-block", id)
		w.kids(el)
		w.b.WriteString("</blockquote>")
	case "ai_block":
		prompt := attrString(el, "prompt")
		w.open("div", "class", "ai-block", "data-block", id, "data-prompt", prompt)
		icon := ""
		if w.pub != nil {
			icon = w.pub.UI("sparkle")
		}
		w.b.WriteString(`<div class="ai-block__head" contenteditable="false"><span class="ai-block__icon">` + icon + `</span><span class="ai-block__prompt">` + esc(prompt) + `</span>`)
		if w.pub == nil {
			w.b.WriteString(`<button class="ai-block__run" type="button">Regenerate</button>`)
		}
		w.b.WriteString(`</div><div class="ai-block__body">`)
		w.kids(el)
		w.b.WriteString("</div></div>")
	case "callout":
		w.open("div", "class", "doc-callout", "data-block", id, "data-tone", attrString(el, "tone"), "data-icon", attrString(el, "icon"))
		if w.pub != nil {
			w.b.WriteString(w.pub.Icon(attrString(el, "icon"), "lightbulb", "callout__icon") + `<div class="callout__body">`)
		} else {
			w.b.WriteString(`<span class="callout__icon" contenteditable="false"></span><div class="callout__body">`)
		}
		w.kids(el)
		w.b.WriteString("</div></div>")
	case "toggle":
		lvl := attrInt(el, "level", 0)
		level := ""
		if lvl > 0 {
			level = strconv.Itoa(lvl)
		}
		w.open("div", "class", "toggle", "data-block", id, "data-level", level)
		chevron := ""
		if w.pub != nil {
			chevron = w.pub.UI("chevron-right")
		}
		w.b.WriteString(`<button class="toggle__btn" type="button" contenteditable="false" aria-expanded="false" aria-label="Show or hide">` + chevron + `</button><div class="toggle__content">`)
		w.kids(el)
		w.b.WriteString("</div></div>")
	case "toggle_title":
		w.b.WriteString(`<div class="toggle__title">`)
		w.inline(el)
		w.b.WriteString("</div>")
	case "bullet_list":
		w.open("ul", "data-block", id)
		w.kids(el)
		w.b.WriteString("</ul>")
	case "ordered_list":
		start := attrInt(el, "start", 1)
		s := ""
		if start != 1 {
			s = strconv.Itoa(start)
		}
		w.open("ol", "data-block", id, "start", s)
		w.kids(el)
		w.b.WriteString("</ol>")
	case "task_list":
		w.open("ul", "class", "tasks", "data-block", id)
		w.kids(el)
		w.b.WriteString("</ul>")
	case "list_item":
		w.open("li", "data-block", id)
		w.kids(el)
		w.b.WriteString("</li>")
	case "task_item":
		checked := "false"
		box := `<input type="checkbox">`
		if attrBool(el, "checked") {
			checked = "true"
			box = `<input type="checkbox" checked>`
		}
		if w.pub != nil {
			box = strings.Replace(box, ">", ` disabled aria-label="Done">`, 1)
		}
		w.open("li", "class", "task", "data-block", id, "data-checked", checked)
		w.b.WriteString(`<span class="task__check" contenteditable="false">` + box + `</span><div class="task__body">`)
		w.kids(el)
		w.b.WriteString("</div></li>")
	case "code_block":
		lang := attrString(el, "language")
		view := ""
		if lang == "mermaid" {
			if v := attrString(el, "view"); v == "code" || v == "preview" {
				view = v
			}
		}
		w.open("div", "class", "code", "data-block", id, "data-language", lang, "data-view", view)
		bar := ""
		if w.pub != nil {
			label := lang
			if label == "" {
				label = "Plain text"
			}
			bar = `<span class="code__lang">` + esc(label) + `</span>`
		}
		w.b.WriteString(`<div class="code__bar" contenteditable="false">` + bar + `</div><pre class="code__pre" spellcheck="false"><code>`)
		code := (&mdWriter{}).plain(el)
		w.b.WriteString(esc(code))
		if code == "" || strings.HasSuffix(code, "\n") {
			w.b.WriteString(`<br class="ProseMirror-trailingBreak">`)
		}
		w.b.WriteString("</code></pre>")
		if d := attrString(el, "diagram"); lang == "mermaid" && strings.HasPrefix(d, "data:image/svg+xml;base64,") && len(d) < 2<<20 {
			w.b.WriteString(`<div class="diagram" contenteditable="false">`)
			w.open("img", "class", "diagram__img", "src", d, "alt", "", "width", dim(el, "dw"), "height", dim(el, "dh"), "decoding", "async")
			w.b.WriteString("</div>")
		}
		w.b.WriteString("</div>")
	case "horizontal_rule":
		w.open("hr", "data-block", id)
	case "table":
		cols := 0
		for _, r := range children(el) {
			if row, ok := r.(*crdt.YXmlElement); ok {
				n := 0
				for _, c := range children(row) {
					if cell, ok := c.(*crdt.YXmlElement); ok {
						n += attrInt(cell, "colspan", 1)
					}
				}
				if n > cols {
					cols = n
				}
			}
		}
		w.b.WriteString(`<div class="tableWrapper">`)
		w.open("table", "data-block", id)
		w.b.WriteString("<colgroup>" + strings.Repeat("<col>", cols) + "</colgroup><tbody>")
		w.kids(el)
		w.b.WriteString("</tbody></table></div>")
	case "table_row":
		w.b.WriteString("<tr>")
		w.kids(el)
		w.b.WriteString("</tr>")
	case "table_cell", "table_header":
		tag := "td"
		if el.NodeName == "table_header" {
			tag = "th"
		}
		cs, rs := "", ""
		if n := attrInt(el, "colspan", 1); n != 1 {
			cs = strconv.Itoa(n)
		}
		if n := attrInt(el, "rowspan", 1); n != 1 {
			rs = strconv.Itoa(n)
		}
		w.open(tag, "colspan", cs, "rowspan", rs)
		w.kids(el)
		w.b.WriteString("</" + tag + ">")
	case "image":
		w.open("figure", "class", "media media--image", "data-block", id)
		w.b.WriteString(`<div class="media__frame">`)
		w.open("img", "src", w.src(attrString(el, "src")), "alt", attrString(el, "alt"), "width", dim(el, "w"), "height", dim(el, "h"), "loading", "lazy", "decoding", "async", "draggable", "false")
		w.b.WriteString("</div>")
		w.caption(attrString(el, "caption"))
		w.b.WriteString("</figure>")
	case "video":
		w.open("figure", "class", "media media--video", "data-block", id)
		w.b.WriteString(`<div class="media__frame">`)
		w.open("video", "src", w.src(attrString(el, "src")), "width", dim(el, "w"), "height", dim(el, "h"), "controls", "controls", "preload", "metadata")
		w.b.WriteString("</video></div>")
		w.caption(attrString(el, "caption"))
		w.b.WriteString("</figure>")
	case "audio":
		w.open("figure", "class", "media media--audio", "data-block", id)
		w.open("audio", "src", w.src(attrString(el, "src")), "controls", "controls", "preload", "metadata")
		w.b.WriteString("</audio>")
		w.caption(attrString(el, "caption"))
		w.b.WriteString("</figure>")
	case "file":
		src := w.src(attrString(el, "src"))
		mime := attrString(el, "mime")
		name := attrString(el, "name")
		w.open("div", "class", "file", "data-block", id, "data-mime", mime)
		clip := ""
		if w.pub != nil {
			clip = w.pub.UI("attachment")
		}
		w.b.WriteString(`<div class="file__card"><span class="file__icon">` + clip + `</span>`)
		w.open("a", "class", "file__name", "href", src, "target", "_blank", "rel", "noopener")
		w.b.WriteString(esc(name) + `</a><span class="file__size">` + esc(humanSize(int64(attrInt(el, "size", 0)))) + "</span></div>")
		if mime == "application/pdf" {
			w.open("iframe", "class", "file__pdf", "src", src, "title", name, "loading", "lazy")
			w.b.WriteString("</iframe>")
		}
		w.b.WriteString("</div>")
	case "bookmark":
		url := attrString(el, "url")
		if safeHref(url) == "" {
			url = ""
		}
		w.open("div", "class", "bookmark", "data-block", id)
		w.open("a", "class", "bookmark__link", "href", url, "target", "_blank", "rel", "noopener noreferrer nofollow")
		title := attrString(el, "title")
		if title == "" {
			title = url
		}
		w.b.WriteString(`<span class="bookmark__text"><span class="bookmark__title">` + esc(title) + `</span><span class="bookmark__desc">` + esc(attrString(el, "description")) + `</span><span class="bookmark__url">`)
		if ic := attrString(el, "icon"); fileID(ic) != "" {
			w.open("img", "class", "bookmark__icon", "src", w.src(ic), "alt", "", "width", "16", "height", "16")
		}
		w.b.WriteString(esc(url) + `</span></span>`)
		if img := attrString(el, "image"); fileID(img) != "" {
			w.open("img", "class", "bookmark__image", "src", w.src(img), "alt", "", "loading", "lazy")
		}
		w.b.WriteString("</a></div>")
	case "embed":
		url := attrString(el, "url")
		w.open("figure", "class", "embed", "data-block", id, "data-provider", attrString(el, "provider"))
		if src, height := EmbedSrc(url, attrString(el, "provider")); src != "" {
			if hh := attrInt(el, "height", 0); hh >= 80 && hh <= 2000 {
				height = hh
			}
			hs := ""
			if height > 0 {
				hs = strconv.Itoa(height)
			}
			w.open("iframe", "src", src, "title", url, "height", hs, "loading", "lazy", "allow", "encrypted-media; picture-in-picture; fullscreen; clipboard-write", "referrerpolicy", "strict-origin-when-cross-origin", "sandbox", "allow-scripts allow-same-origin allow-presentation allow-popups allow-popups-to-escape-sandbox allow-forms")
			w.b.WriteString("</iframe>")
		} else if safeHref(url) != "" {
			w.open("a", "class", "embed__fallback", "href", safeHref(url), "target", "_blank", "rel", "noopener noreferrer nofollow")
			w.b.WriteString(esc(url) + "</a>")
		}
		w.b.WriteString("</figure>")
	case "database":
		w.open("div", "class", "dbblock", "data-block", id, "data-collection", attrString(el, "collection"), "data-view", attrString(el, "view"), "contenteditable", "false")
		if w.pub != nil && w.pub.Table != nil {
			w.b.WriteString(w.pub.Table(attrString(el, "collection"), attrString(el, "view")))
		}
		w.b.WriteString("</div>")
	case "toc":
		w.open("nav", "class", "toc", "data-block", id, "contenteditable", "false")
		for _, h := range w.heads {
			w.open("a", "class", "toc__item", "data-level", strconv.Itoa(h.level), "href", "#"+h.id)
			w.b.WriteString(esc(h.text) + "</a>")
		}
		w.b.WriteString("</nav>")
	case "math_block":
		w.math(el, true, id)
	case "synced":
		w.synced(el, id)
	case "breadcrumb":
		w.open("nav", "class", "crumbline", "data-block", id, "contenteditable", "false")
		for _, c := range w.crumbs {
			label := c.Title
			if label == "" {
				label = "Untitled"
			}
			if w.pub != nil {
				w.open("a", "class", "crumbline__item", "href", w.pub.Page(c.ID))
				w.b.WriteString(w.pub.PageIcon(c.ID, "crumbline__icon") + `<span class="crumbline__label">` + esc(label) + "</span></a>")
				continue
			}
			w.open("a", "class", "crumbline__item", "href", "/p/"+c.ID)
			w.b.WriteString(`<span class="crumbline__icon"></span><span class="crumbline__label">` + esc(label) + "</span></a>")
		}
		w.b.WriteString("</nav>")
	case "columns":
		n := len(children(el))
		w.open("div", "class", "columns", "data-block", id, "data-count", strconv.Itoa(n))
		w.kids(el)
		w.b.WriteString("</div>")
	case "column":
		if wd := attrInt(el, "width", 0); wd > 0 && wd <= 100 {
			w.b.WriteString(`<div class="column" data-w="` + strconv.Itoa(wd) + `">`)
		} else {
			w.b.WriteString(`<div class="column">`)
		}
		w.kids(el)
		w.b.WriteString("</div>")
	default:
		w.kids(el)
	}
}

func dim(el *crdt.YXmlElement, key string) string {
	if n := attrInt(el, key, 0); n > 0 {
		return strconv.Itoa(n)
	}
	return ""
}

func (w *htmlWriter) caption(s string) {
	w.b.WriteString(`<figcaption class="media__caption">` + esc(s) + "</figcaption>")
}

func humanSize(n int64) string {
	switch {
	case n <= 0:
		return ""
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.0f KB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
}

func (w *htmlWriter) inline(el *crdt.YXmlElement) {
	var open []string
	var openHref string
	closeTo := func(n int) {
		for len(open) > n {
			m := open[len(open)-1]
			open = open[:len(open)-1]
			w.b.WriteString(markClose(m))
		}
	}
	empty := true
	lastBreak := false
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
				var want []string
				for _, m := range markOrder {
					if _, ok := marks[m]; ok {
						want = append(want, m)
					}
				}
				href := safeHref(markAttr(marks["link"], "href"))
				keep := 0
				for keep < len(open) && keep < len(want) && open[keep] == want[keep] && (open[keep] != "link" || href == openHref) {
					keep++
				}
				closeTo(keep)
				for _, m := range want[keep:] {
					switch m {
					case "link":
						openHref = href
						w.open("a", "href", w.href(href), "rel", "noopener noreferrer nofollow")
					case "highlight":
						w.open("mark", "data-color", markAttr(marks[m], "value"))
					case "color":
						w.open("span", "class", "tc", "data-color", markAttr(marks[m], "value"))
					default:
						w.b.WriteString("<" + markTag(m) + ">")
					}
					open = append(open, m)
				}
				w.b.WriteString(esc(s))
				empty = false
				lastBreak = false
			}
		case *crdt.YXmlElement:
			closeTo(0)
			switch x.NodeName {
			case "hard_break":
				w.b.WriteString("<br>")
				lastBreak = true
			case "mention":
				w.mention(x)
				lastBreak = false
			case "math_inline":
				w.math(x, false, "")
				lastBreak = false
			}
			empty = false
		}
	}
	closeTo(0)
	if empty || lastBreak {
		w.b.WriteString(`<br class="ProseMirror-trailingBreak">`)
	}
}

func markTag(m string) string {
	switch m {
	case "strong":
		return "strong"
	case "em":
		return "em"
	case "strike":
		return "s"
	case "underline":
		return "u"
	case "code":
		return "code"
	}
	return "span"
}

func markClose(m string) string {
	switch m {
	case "link":
		return "</a>"
	case "highlight":
		return "</mark>"
	case "color":
		return "</span>"
	}
	return "</" + markTag(m) + ">"
}

func (w *htmlWriter) mention(el *crdt.YXmlElement) {
	kind, ref, label := attrString(el, "kind"), attrString(el, "ref"), attrString(el, "label")
	switch kind {
	case "page":
		if w.pub != nil {
			w.pubMention(ref)
			return
		}
		if w.title != nil {
			if t := w.title(ref); t != "" {
				label = t
			}
		}
		if label == "" {
			label = "Untitled"
		}
		w.open("a", "class", "mention", "data-kind", "page", "data-ref", ref, "href", "/p/"+ref, "contenteditable", "false")
		w.b.WriteString(`<span class="mention__icon"></span><span class="mention__label">` + esc(label) + "</span></a>")
	case "user":
		if w.pub != nil {
			ref = ""
		}
		w.open("span", "class", "mention", "data-kind", "user", "data-ref", ref, "contenteditable", "false")
		w.b.WriteString("@" + esc(label) + "</span>")
	case "date":
		if who := attrString(el, "remind"); who != "" && w.pub == nil {
			w.open("span", "class", "mention", "data-kind", "date", "data-ref", ref, "data-remind", who, "data-at", attrString(el, "at"), "contenteditable", "false")
		} else {
			w.open("span", "class", "mention", "data-kind", "date", "data-ref", ref, "contenteditable", "false")
		}
		w.b.WriteString(esc(w.dateText(ref)) + "</span>")
	}
}
