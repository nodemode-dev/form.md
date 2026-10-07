package static

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
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

type Handler struct {
	files map[string]*asset
	index *asset
	csp   string
}

type module struct{}

func Module() app.Module { return module{} }

func (module) ID() string { return "static" }

func (module) Mount(r *httpx.Router, d *app.Deps) error {
	h, err := New(d.Dist)
	if err != nil {
		return err
	}
	if h.index == nil {
		d.Log.Warn("web client not built; serving API only", "hint", "run make build, or use the Vite dev server on http://127.0.0.1:5750")
	}
	r.Handle("/", h)
	return nil
}

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

var compressible = map[string]bool{
	".html": true, ".js": true, ".mjs": true, ".css": true, ".svg": true,
	".json": true, ".webmanifest": true, ".txt": true, ".map": true,
}

func New(dist fs.FS) (*Handler, error) {
	h := &Handler{files: map[string]*asset{}}
	err := fs.WalkDir(dist, ".", func(p string, d fs.DirEntry, err error) error {
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
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		httpx.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	p := path.Clean("/" + r.URL.Path)
	if a, ok := h.files[p]; ok && p != "/index.html" {
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
	h.serve(w, r, h.index, true)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request, a *asset, document bool) {
	hdr := w.Header()
	hdr.Set("Content-Type", a.ctype)
	hdr.Set("ETag", a.etag)
	hdr.Set("Vary", "Accept-Encoding")
	if a.immutable {
		hdr.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
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
