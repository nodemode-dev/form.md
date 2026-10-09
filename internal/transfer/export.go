package transfer

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/site"
)

const maxExportPages = 5000

type exNode struct {
	id, parent, title, icon, cover, kind, desc, sortKey string
	row, inline                                         bool
	updated                                             int64
	file                                                string
	kids                                                []*exNode
}

type exportSet struct {
	byID  map[string]*exNode
	roots []*exNode
	ws    string
	all   []*exNode
}

func (s *Service) collect(ctx context.Context, p *policy.Principal, rootID, workspaceID string, subpages bool) (*exportSet, error) {
	ws := workspaceID
	if rootID != "" {
		m, _, err := s.pages.ReadableMeta(ctx, p, rootID)
		if err != nil {
			return nil, err
		}
		ws = m.WorkspaceID
	}
	tree, err := s.pages.Tree(ctx, p, ws)
	if err != nil {
		return nil, err
	}
	set := &exportSet{byID: map[string]*exNode{}, ws: ws}
	byParent := map[string][]*exNode{}
	for _, m := range tree {
		n := &exNode{id: m.ID, parent: m.ParentID, title: m.Title, icon: m.Icon, cover: m.Cover, kind: m.Kind, desc: m.Description, sortKey: m.SortKey, updated: m.UpdatedAt}
		set.byID[m.ID] = n
		byParent[m.ParentID] = append(byParent[m.ParentID], n)
	}
	if rootID != "" {
		if _, ok := set.byID[rootID]; !ok {
			m, err := s.pages.Load(ctx, rootID)
			if err != nil {
				return nil, err
			}
			n := &exNode{id: m.ID, parent: m.ParentID, title: m.Title, icon: m.Icon, cover: m.Cover, kind: m.Kind, desc: m.Description, sortKey: m.SortKey, updated: m.UpdatedAt, row: m.Row}
			set.byID[m.ID] = n
		}
	}
	keep := map[string]*exNode{}
	var add func(n *exNode, depth int)
	add = func(n *exNode, depth int) {
		if len(keep) >= maxExportPages || depth > 64 || keep[n.id] != nil {
			return
		}
		keep[n.id] = n
		set.all = append(set.all, n)
		if n.kind == "collection" {
			s.addRows(ctx, p, n, keep, set, byParent, depth, add)
		}
		for _, k := range byParent[n.id] {
			if !subpages && k.kind != "collection" {
				continue
			}
			n.kids = append(n.kids, k)
			add(k, depth+1)
		}
	}
	if rootID != "" {
		r := set.byID[rootID]
		set.roots = []*exNode{r}
		add(r, 0)
	} else {
		for _, m := range tree {
			n := set.byID[m.ID]
			if m.ParentID == "" || m.Detached || set.byID[m.ParentID] == nil {
				set.roots = append(set.roots, n)
			}
		}
		for _, r := range set.roots {
			add(r, 1)
		}
	}
	set.byID = keep
	for _, n := range set.all {
		sort.SliceStable(n.kids, func(i, j int) bool { return n.kids[i].sortKey < n.kids[j].sortKey })
	}
	assignPaths(set.roots, "")
	return set, nil
}

func (s *Service) addRows(ctx context.Context, p *policy.Principal, n *exNode, keep map[string]*exNode, set *exportSet, byParent map[string][]*exNode, depth int, add func(*exNode, int)) {
	for off := 0; len(keep) < maxExportPages; off += 1000 {
		res, err := s.cols.Query(ctx, p, n.id, collection.QueryInput{Offset: off, Limit: 1000, NoGroups: true})
		if err != nil || len(res.Rows) == 0 {
			return
		}
		for _, r := range res.Rows {
			if keep[r.ID] != nil {
				continue
			}
			rn := &exNode{id: r.ID, parent: n.id, title: r.Title, icon: r.Icon, cover: r.Cover, kind: "doc", sortKey: r.SortKey, row: true, updated: r.UpdatedAt}
			keep[rn.id] = rn
			set.all = append(set.all, rn)
			n.kids = append(n.kids, rn)
			for _, k := range byParent[rn.id] {
				rn.kids = append(rn.kids, k)
				add(k, depth+2)
			}
		}
		if off+len(res.Rows) >= res.Total {
			return
		}
	}
}

var unsafeName = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]+`)

func safeFile(title string) string {
	t := strings.TrimSpace(unsafeName.ReplaceAllString(stripPictographs(title), " "))
	t = strings.Trim(t, ". ")
	if r := []rune(t); len(r) > 80 {
		t = strings.TrimSpace(string(r[:80]))
	}
	if t == "" {
		t = "Untitled"
	}
	return t
}

func assignPaths(list []*exNode, dir string) {
	used := map[string]int{}
	for _, n := range list {
		base := safeFile(n.title)
		key := strings.ToLower(base)
		if c := used[key]; c > 0 {
			used[key] = c + 1
			base = base + " (" + itoa(c+1) + ")"
		} else {
			used[key] = 1
		}
		if dir == "" {
			n.file = base
		} else {
			n.file = dir + "/" + base
		}
		assignPaths(n.kids, n.file)
	}
}

func relLink(fromFile, toPath string) string {
	r, err := filepath.Rel(filepath.FromSlash(path.Dir(fromFile)), filepath.FromSlash(toPath))
	if err != nil {
		return toPath
	}
	parts := strings.Split(filepath.ToSlash(r), "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

type fileCopy struct {
	id   string
	path string
}

type exporter struct {
	s      *Service
	p      *policy.Principal
	set    *exportSet
	origin string
	files  map[string]*fileCopy
	order  []*fileCopy
}

func (e *exporter) fileRef(id string) string {
	if fc, ok := e.files[id]; ok {
		return fc.path
	}
	f, rc, err := e.s.files.Open(context.Background(), e.p, id)
	if err != nil {
		return ""
	}
	rc.Close()
	name := safeFile(stripExt(f.Name))
	fc := &fileCopy{id: id, path: "files/" + id[:min(len(id), 8)] + "-" + name + f.Ext}
	e.files[id] = fc
	e.order = append(e.order, fc)
	return fc.path
}

var pageLinkRE = regexp.MustCompile(`\]\(/p/([a-z0-9]+)((?:#[^)\s]*)?)([ )])`)
var fileLinkRE = regexp.MustCompile(`\]\(/files/([a-z0-9]+)([ )])`)

func (e *exporter) markdownFor(ctx context.Context, n *exNode) string {
	md, err := e.s.docs.Markdown(ctx, e.p, n.id)
	if err != nil {
		md = ""
	}
	from := n.file + ".md"
	md = pageLinkRE.ReplaceAllStringFunc(md, func(m string) string {
		sub := pageLinkRE.FindStringSubmatch(m)
		if t, ok := e.set.byID[sub[1]]; ok {
			ext := ".md"
			if t.kind == "collection" {
				ext = ".csv"
			}
			return "](<" + strings.ReplaceAll(relLink(from, t.file+ext), "%20", " ") + ">" + sub[3]
		}
		return "](" + e.origin + "/p/" + sub[1] + sub[2] + sub[3]
	})
	md = fileLinkRE.ReplaceAllStringFunc(md, func(m string) string {
		sub := fileLinkRE.FindStringSubmatch(m)
		if fp := e.fileRef(sub[1]); fp != "" {
			return "](<" + strings.ReplaceAll(relLink(from, fp), "%20", " ") + ">" + sub[2]
		}
		return m
	})
	title := n.title
	if title == "" {
		title = "Untitled"
	}
	head := "# " + title + "\n\n"
	if n.desc != "" {
		head += escapeText(n.desc) + "\n\n"
	}
	return head + md
}

func (e *exporter) writeFiles(zw *zip.Writer) {
	for _, fc := range e.order {
		_, rc, err := e.s.files.Open(context.Background(), e.p, fc.id)
		if err != nil {
			continue
		}
		w, err := zw.Create(fc.path)
		if err == nil {
			_, _ = io.Copy(w, rc)
		}
		rc.Close()
	}
}

func (s *Service) firstView(ctx context.Context, p *policy.Principal, id string) string {
	out, err := s.cols.Get(ctx, p, id)
	if err != nil {
		return ""
	}
	for _, v := range out.Views {
		if v.Type != "form" {
			return v.ID
		}
	}
	return ""
}

func (s *Service) exportMarkdown(ctx context.Context, p *policy.Principal, set *exportSet, origin string, w io.Writer) error {
	e := &exporter{s: s, p: p, set: set, origin: origin, files: map[string]*fileCopy{}}
	zw := zip.NewWriter(w)
	for _, n := range set.all {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if n.kind == "collection" {
			view := s.firstView(ctx, p, n.id)
			if view == "" {
				continue
			}
			var buf bytes.Buffer
			if _, err := s.cols.ExportCSV(ctx, p, view, &buf); err != nil {
				continue
			}
			fw, err := zw.Create(n.file + ".csv")
			if err != nil {
				return err
			}
			_, _ = fw.Write(buf.Bytes())
			continue
		}
		md := e.markdownFor(ctx, n)
		fw, err := zw.Create(n.file + ".md")
		if err != nil {
			return err
		}
		_, _ = io.WriteString(fw, md)
	}
	e.writeFiles(zw)
	return zw.Close()
}

func (s *Service) exportHTML(ctx context.Context, p *policy.Principal, set *exportSet, origin, theme string, w io.Writer, single bool) error {
	e := &exporter{s: s, p: p, set: set, origin: origin, files: map[string]*fileCopy{}}
	nodes := make([]site.ExportNode, 0, len(set.all))
	for _, n := range set.all {
		nodes = append(nodes, site.ExportNode{ID: n.id, Parent: n.parent, Title: n.title, Icon: n.icon, Cover: n.cover, Kind: n.kind, Description: n.desc, SortKey: n.sortKey, Row: n.row, Inline: n.inline, UpdatedAt: n.updated})
	}
	var current *exNode
	root := ""
	if len(set.roots) == 1 {
		root = set.roots[0].id
	}
	embedded := 0
	href := func(id string) string {
		t, ok := set.byID[id]
		if !ok || current == nil {
			return ""
		}
		if single {
			return origin + "/p/" + id
		}
		return relLink(current.file+".html", t.file+".html")
	}
	file := func(id string) string {
		if single {
			f, rc, err := s.files.Open(ctx, p, id)
			if err == nil {
				defer rc.Close()
				if f.Size < 8<<20 && embedded+int(f.Size) < 24<<20 {
					data, err := io.ReadAll(rc)
					if err == nil {
						embedded += len(data)
						return "data:" + f.Mime + ";base64," + base64.StdEncoding.EncodeToString(data)
					}
				}
			}
			return origin + "/files/" + id
		}
		fp := e.fileRef(id)
		if fp == "" || current == nil {
			return origin + "/files/" + id
		}
		return relLink(current.file+".html", fp)
	}
	x := s.site.NewExport(nodes, root, theme, href, file)
	if single {
		current = set.roots[0]
		out, err := x.Page(ctx, current.id)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, out)
		return err
	}
	zw := zip.NewWriter(w)
	for _, n := range set.all {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		current = n
		out, err := x.Page(ctx, n.id)
		if err != nil || out == "" {
			continue
		}
		fw, err := zw.Create(n.file + ".html")
		if err != nil {
			return err
		}
		_, _ = io.WriteString(fw, out)
	}
	if len(set.roots) > 1 {
		var b strings.Builder
		b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Export</title><style>body{font:16px/1.6 system-ui,sans-serif;max-width:720px;margin:48px auto;padding:0 16px;color:#222}a{color:#5b4bd5}li{margin:4px 0}</style></head><body><h1>Exported pages</h1><ul>`)
		for _, r := range set.roots {
			b.WriteString(`<li><a href="` + relLink("index.html", r.file+".html") + `">` + htmlEscape(orUntitled(r.title)) + `</a></li>`)
		}
		b.WriteString(`</ul></body></html>`)
		if fw, err := zw.Create("index.html"); err == nil {
			_, _ = io.WriteString(fw, b.String())
		}
	}
	e.writeFiles(zw)
	return zw.Close()
}

func orUntitled(t string) string {
	if strings.TrimSpace(t) == "" {
		return "Untitled"
	}
	return t
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func origin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func attachment(w http.ResponseWriter, name, ctype string) {
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Cache-Control", "no-store")
}

func mountExports(r *httpx.Router, s *Service, route func(string, func(http.ResponseWriter, *http.Request, *policy.Principal) error)) {
	route("GET /api/pages/{id}/export", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		q := r.URL.Query()
		format := q.Get("format")
		subpages := q.Get("subpages") == "1" || q.Get("subpages") == "true"
		set, err := s.collect(r.Context(), p, r.PathValue("id"), "", subpages)
		if err != nil {
			return err
		}
		name := safeFile(set.roots[0].title)
		stamp := time.Now().UTC().Format("2006-01-02")
		switch format {
		case "markdown", "md":
			attachment(w, name+" "+stamp+".zip", "application/zip")
			return s.exportMarkdown(r.Context(), p, set, origin(r), w)
		case "html":
			single := !subpages && set.roots[0].kind != "collection"
			if single {
				attachment(w, name+".html", "text/html; charset=utf-8")
			} else {
				attachment(w, name+" "+stamp+".zip", "application/zip")
			}
			return s.exportHTML(r.Context(), p, set, origin(r), q.Get("theme"), w, single)
		case "pdf":
			if !subpages {
				set.all = []*exNode{set.roots[0]}
			}
			var buf bytes.Buffer
			if err := s.exportPDF(r.Context(), p, set, origin(r), &buf); err != nil {
				return err
			}
			attachment(w, name+".pdf", "application/pdf")
			_, err := w.Write(buf.Bytes())
			return err
		}
		return httpx.Fail(http.StatusBadRequest, "bad_format", "format is markdown, html or pdf")
	})
	route("GET /api/workspaces/{id}/export", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		q := r.URL.Query()
		a, err := s.gate.Workspace(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		if !a.IsAdmin() && a.Role != "member" {
			return httpx.ErrForbidden
		}
		set, err := s.collect(r.Context(), p, "", r.PathValue("id"), true)
		if err != nil {
			return err
		}
		stamp := time.Now().UTC().Format("2006-01-02")
		switch q.Get("format") {
		case "markdown", "md", "":
			attachment(w, "workspace "+stamp+".zip", "application/zip")
			return s.exportMarkdown(r.Context(), p, set, origin(r), w)
		case "html":
			attachment(w, "workspace "+stamp+".zip", "application/zip")
			return s.exportHTML(r.Context(), p, set, origin(r), q.Get("theme"), w, false)
		}
		return httpx.Fail(http.StatusBadRequest, "bad_format", "format is markdown or html")
	})
}
