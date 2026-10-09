package static

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"html"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/httpx"
)

type asset struct {
	body      []byte
	gz        []byte
	ctype     string
	etag      string
	immutable bool
}

type Document struct {
	Attrs    map[string]string
	Slots    map[string]string
	HTML     map[string]string
	Lang     string
	Boot     any
	Redirect string
	Preload  []string
}

type Renderer func(r *http.Request) (Document, error)

type Handler struct {
	files    map[string]*asset
	index    *asset
	csp      string
	render   Renderer
	preloads map[string][]string
}

type manifestChunk struct {
	File           string   `json:"file"`
	Imports        []string `json:"imports"`
	IsEntry        bool     `json:"isEntry"`
	IsDynamicEntry bool     `json:"isDynamicEntry"`
}

func readPreloads(dist fs.FS) map[string][]string {
	out := map[string][]string{}
	raw, err := fs.ReadFile(dist, ".vite/manifest.json")
	if err != nil {
		return out
	}
	var m map[string]manifestChunk
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for key, c := range m {
		rest, ok := strings.CutPrefix(key, "src/extensions/")
		name, _, nested := strings.Cut(rest, "/")
		if !ok || !nested || !c.IsDynamicEntry {
			continue
		}
		seen := map[string]bool{}
		var walk func(k string)
		walk = func(k string) {
			x, ok := m[k]
			if !ok || x.IsEntry || seen[k] {
				return
			}
			seen[k] = true
			out[name] = append(out[name], "/"+x.File)
			for _, i := range x.Imports {
				walk(i)
			}
		}
		walk(key)
	}
	return out
}

type module struct{ render Renderer }

func Module(render Renderer) app.Module { return module{render} }

func (module) ID() string { return "static" }

func (m module) Mount(r *httpx.Router, d *app.Deps) error {
	h, err := New(d.Dist)
	if err != nil {
		return err
	}
	h.render = m.render
	if h.index == nil {
		d.Log.Warn("web client not built; serving API only", "hint", "run make build, or use the Vite dev server on http://127.0.0.1:5750")
	}
	r.Handle("/", h)
	return nil
}

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

var slotPattern = regexp.MustCompile(`<!--slot:([a-z-]+)-->`)

var htmlSlotPattern = regexp.MustCompile(`<!--html:([a-z-]+)-->`)

var fontPreload = regexp.MustCompile(`\s*<link rel="preload"[^>]*as="font"[^>]*>`)

var compressible = map[string]bool{
	".html": true, ".js": true, ".mjs": true, ".css": true, ".svg": true,
	".json": true, ".webmanifest": true, ".txt": true, ".map": true,
}

func New(dist fs.FS) (*Handler, error) {
	h := &Handler{files: map[string]*asset{}}
	err := fs.WalkDir(dist, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && p != "." && strings.HasPrefix(path.Base(p), ".") {
			return fs.SkipDir
		}
		if err != nil || d.IsDir() || strings.HasPrefix(path.Base(p), ".") {
			return err
		}
		body, err := fs.ReadFile(dist, p)
		if err != nil {
			return err
		}
		ext := path.Ext(p)
		a := &asset{
			body:      body,
			ctype:     contentType(ext),
			etag:      etag(body),
			immutable: strings.HasPrefix(p, "assets/"),
		}
		if compressible[ext] && len(body) > 512 {
			if gz := gzipBytes(body); len(gz) < len(body) {
				a.gz = gz
			}
		}
		h.files["/"+p] = a
		return nil
	})
	if err != nil {
		return nil, err
	}
	h.index = h.files["/index.html"]
	h.csp = buildCSP(h.index)
	h.preloads = readPreloads(dist)
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		httpx.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	p := path.Clean("/" + r.URL.Path)
	if p == "/offline.json" {
		h.serveOffline(w, r)
		return
	}
	if a, ok := h.files[p]; ok && p != "/index.html" {
		switch {
		case p == "/sandbox.html":
			w.Header().Set("Content-Security-Policy", sandboxCSP)
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		case p == "/sw.js":
			w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'")
		case strings.HasPrefix(p, "/assets/"):
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		h.serve(w, r, a, false)
		return
	}
	if path.Ext(p) != "" && p != "/index.html" {
		http.NotFound(w, r)
		return
	}
	if h.index == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("form-md: the web client is not built.\nRun `make build`, or use the Vite dev server at http://127.0.0.1:5750 while developing.\n"))
		return
	}
	if h.render == nil {
		h.serve(w, r, h.index, true)
		return
	}
	h.serveDocument(w, r)
}

var (
	authMarker = []byte(`data-auth="pending"`)
	langMarker = []byte(`<html lang="en"`)
	bootMarker = []byte(`<script type="application/json" id="boot">null</script>`)
)

func RenderIndex(index []byte, doc Document) ([]byte, error) {
	boot, err := json.Marshal(doc.Boot)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(doc.Attrs))
	for k := range doc.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var attrs []string
	for _, k := range keys {
		attrs = append(attrs, `data-`+k+`="`+html.EscapeString(doc.Attrs[k])+`"`)
	}
	out := bytes.Replace(index, authMarker, []byte(strings.Join(attrs, " ")), 1)
	out = slotPattern.ReplaceAllFunc(out, func(m []byte) []byte {
		key := string(slotPattern.FindSubmatch(m)[1])
		return []byte(html.EscapeString(doc.Slots[key]))
	})
	out = htmlSlotPattern.ReplaceAllFunc(out, func(m []byte) []byte {
		return []byte(doc.HTML[string(htmlSlotPattern.FindSubmatch(m)[1])])
	})
	if doc.Lang != "" && doc.Lang != "en" {
		out = bytes.Replace(out, langMarker, []byte(`<html lang="`+doc.Lang+`"`), 1)
	}
	if doc.Attrs["font"] == "system" {
		out = fontPreload.ReplaceAll(out, nil)
	}
	return bytes.Replace(out, bootMarker, append(append([]byte(`<script type="application/json" id="boot">`), boot...), []byte(`</script>`)...), 1), nil
}

func (h *Handler) serveDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := h.render(r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if doc.Redirect != "" {
		http.Redirect(w, r, doc.Redirect, http.StatusFound)
		return
	}
	var links strings.Builder
	for _, name := range doc.Preload {
		for _, href := range h.preloads[name] {
			links.WriteString(`<link rel="modulepreload" crossorigin href="` + html.EscapeString(href) + `" />`)
		}
	}
	if links.Len() > 0 {
		if doc.HTML == nil {
			doc.HTML = map[string]string{}
		}
		doc.HTML["preload"] = links.String()
	}
	body, err := RenderIndex(h.index.body, doc)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a := &asset{body: body, ctype: h.index.ctype, etag: etag(body)}
	if acceptsGzip(r.Header.Get("Accept-Encoding")) {
		a.gz = gzipBytes(body)
	}
	h.serve(w, r, a, true)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request, a *asset, document bool) {
	hdr := w.Header()
	hdr.Set("Content-Type", a.ctype)
	hdr.Set("ETag", a.etag)
	hdr.Set("Vary", "Accept-Encoding")
	switch {
	case a.immutable:
		hdr.Set("Cache-Control", "public, max-age=31536000, immutable")
	case document:
		hdr.Set("Cache-Control", "no-store")
	default:
		hdr.Set("Cache-Control", "no-cache")
	}
	if document {
		hdr.Set("Content-Security-Policy", h.csp)
	}
	if match := r.Header.Get("If-None-Match"); match != "" && match == a.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	body := a.body
	if a.gz != nil && acceptsGzip(r.Header.Get("Accept-Encoding")) {
		hdr.Set("Content-Encoding", "gzip")
		body = a.gz
	}
	hdr.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

const sandboxCSP = "sandbox allow-scripts; default-src 'none'; script-src 'self'; style-src 'unsafe-inline'; img-src data:; font-src 'self'; frame-ancestors 'self'; base-uri 'none'; form-action 'none'"

func buildCSP(index *asset) string {
	scripts := []string{"'self'"}
	if index != nil {
		for _, m := range inlineScript.FindAllSubmatch(index.body, -1) {
			sum := sha256.Sum256(m[1])
			scripts = append(scripts, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
		}
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + strings.Join(scripts, " "),
		"style-src 'self'",
		"img-src 'self' data: blob:",
		"font-src 'self'",
		"connect-src 'self'",
		"frame-src 'self' https:",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

func contentType(ext string) string {
	switch ext {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".webmanifest":
		return "application/manifest+json"
	case ".woff2":
		return "font/woff2"
	}
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "application/octet-stream"
}

func etag(b []byte) string {
	sum := sha256.Sum256(b)
	return `"` + hex.EncodeToString(sum[:12]) + `"`
}

func gzipBytes(b []byte) []byte {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(name), "gzip") {
			return strings.ReplaceAll(strings.TrimSpace(params), " ", "") != "q=0"
		}
	}
	return false
}
