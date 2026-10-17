package family

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type product struct {
	ID      string
	Base    string
	Health  string
	Status  string
	Purpose string
	Objects []string
	Connect string
	Words   []string
}

var siblings = []product{
	{
		ID: "formlit", Base: "http://127.0.0.1:8390", Health: "/api/health", Status: "/api/mcp/status",
		Purpose: "Photo library and raw developer: import, cull, rate, keyword, develop, mask, retouch, export, print and publish photos.",
		Objects: []string{"photo", "folder", "collection", "keyword", "person"},
		Connect: "./formlit mcp (stdio), or the formlit plugin with the app open; the first call asks in formlit's window to allow outside agents",
		Words: []string{"photo", "photos", "raw", "catalog", "catalogue", "cull", "culling", "rating", "rate", "flag", "keyword", "keywords", "develop", "exposure",
			"white", "balance", "preset", "presets", "lightroom", "import", "hdr", "panorama", "denoise", "print", "proofing", "gallery", "faces", "people", "camera", "shoot", "wedding"},
	},
	{
		ID: "formvis", Base: "http://127.0.0.1:8190", Health: "/version", Status: "/api/mcp/status",
		Purpose: "Layered image editor: layers, selections, masks, adjustments, filters, text, shapes, generative fill and expand, background removal, timeline and export.",
		Objects: []string{"project", "document", "layer"},
		Connect: "./formvis mcp (stdio); tools run in the open formvis window, which asks once to allow outside agents",
		Words: []string{"image", "picture", "layer", "layers", "photoshop", "retouch", "selection", "select", "mask", "background", "remove", "generative", "fill", "expand", "upscale",
			"text", "shape", "filter", "adjustment", "composite", "collage", "poster", "thumbnail", "banner", "crop", "resize", "canvas", "psd", "edit", "edited"},
	},
}

var self = product{
	ID:      "form.md",
	Purpose: "Workspace for documents, wikis, notes and databases that people and agents edit together in real time.",
	Objects: []string{"page", "database", "row", "file", "comment"},
	Words: []string{"document", "doc", "page", "pages", "note", "notes", "wiki", "write", "writing", "draft", "database", "table", "row", "rows", "task", "tasks", "project",
		"plan", "meeting", "brief", "spec", "knowledge", "comment", "board", "kanban", "calendar", "publish", "site"},
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Member struct {
	Product   string   `json:"product"`
	Purpose   string   `json:"purpose"`
	Objects   []string `json:"objects"`
	Base      string   `json:"base,omitempty"`
	MCP       string   `json:"mcp,omitempty"`
	Version   string   `json:"version,omitempty"`
	Reachable bool     `json:"reachable"`
	Problem   string   `json:"problem,omitempty"`
	Connect   string   `json:"connect,omitempty"`
	ToolCount int      `json:"tool_count,omitempty"`
	tools     []Tool
}

type Route struct {
	Member
	Score int    `json:"score"`
	Tools []Tool `json:"tools"`
}

type Service struct {
	auth    *auth.Service
	version string
	bases   map[string]string
	off     bool
	client  *http.Client

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	m  Member
	at time.Time
}

func New(a *auth.Service, version, spec string) *Service {
	s := &Service{auth: a, version: version, bases: map[string]string{}, cache: map[string]cached{},
		client: &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	spec = strings.TrimSpace(spec)
	if spec == "off" {
		s.off = true
	}
	for _, part := range strings.Split(spec, ",") {
		id, base, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && loopback(base) {
			s.bases[strings.TrimSpace(id)] = strings.TrimRight(strings.TrimSpace(base), "/")
		}
	}
	return s
}

func loopback(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return false
	}
	h := u.Hostname()
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func (s *Service) Descriptor(base string) map[string]any {
	family := []string{}
	for _, p := range siblings {
		family = append(family, p.ID)
	}
	return map[string]any{
		"product": self.ID, "version": s.version, "purpose": self.Purpose, "objects": self.Objects,
		"rest": base + "/api", "openapi": base + "/api/openapi.json", "mcp": base + "/mcp",
		"open": base + "/p/{id}", "auth": []string{"bearer token (Settings, API tokens)", "OAuth 2.1 (/.well-known/oauth-authorization-server)"},
		"family": family,
	}
}

func (s *Service) get(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return httpx.Fail(http.StatusBadGateway, "family_status", res.Status)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
}

func (s *Service) listTools(ctx context.Context, endpoint string) ([]Tool, error) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-11-25")
	req.Header.Set("X-Formlit-Agent", "form.md")
	req.Header.Set("X-Formvis-Client", "form.md")
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var out struct {
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&out); err != nil {
		return nil, err
	}
	return out.Result.Tools, nil
}

func (s *Service) probe(ctx context.Context, p product) Member {
	base := p.Base
	if b, ok := s.bases[p.ID]; ok {
		base = b
	}
	m := Member{Product: p.ID, Purpose: p.Purpose, Objects: p.Objects, Base: base, Connect: p.Connect}
	if s.off {
		m.Problem = "family lookups are off (FORMMD_FAMILY=off)"
		return m
	}
	s.mu.Lock()
	if c, ok := s.cache[p.ID]; ok && time.Since(c.at) < 30*time.Second {
		s.mu.Unlock()
		return c.m
	}
	s.mu.Unlock()
	var health struct {
		Version string `json:"version"`
	}
	if err := s.get(ctx, base+p.Health, &health); err != nil {
		m.Problem = "not running at " + base
	} else {
		m.Reachable, m.Version = true, health.Version
		var status struct {
			Endpoint string `json:"endpoint"`
		}
		if s.get(ctx, base+p.Status, &status) == nil && status.Endpoint != "" && loopback(status.Endpoint) {
			m.MCP = status.Endpoint
		} else {
			m.MCP = base + "/mcp"
		}
		if tools, err := s.listTools(ctx, m.MCP); err == nil {
			m.tools, m.ToolCount = tools, len(tools)
		} else {
			m.Problem = "running, but its tool list could not be read"
		}
	}
	s.mu.Lock()
	s.cache[p.ID] = cached{m: m, at: time.Now()}
	s.mu.Unlock()
	return m
}

func (s *Service) List(ctx context.Context) []Member {
	out := make([]Member, len(siblings))
	var wg sync.WaitGroup
	for i, p := range siblings {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = s.probe(ctx, p)
		}()
	}
	wg.Wait()
	return out
}

var stop = map[string]bool{"the": true, "and": true, "for": true, "from": true, "this": true, "that": true, "with": true, "into": true, "onto": true, "all": true, "any": true,
	"are": true, "was": true, "can": true, "you": true, "your": true, "our": true, "its": true, "use": true, "using": true, "make": true, "get": true, "set": true, "add": true,
	"new": true, "open": true, "one": true, "some": true, "each": true, "such": true, "them": true, "then": true, "than": true, "out": true, "per": true, "via": true}

func words(task string) []string {
	f := strings.FieldsFunc(strings.ToLower(task), func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') })
	out := f[:0]
	for _, w := range f {
		if len(w) > 2 && !stop[w] {
			out = append(out, w)
		}
	}
	return out
}

func score(task []string, keys []string) int {
	n := 0
	for _, w := range task {
		for _, k := range keys {
			if w == k || (len(k) > 4 && strings.HasPrefix(w, k)) {
				n++
				break
			}
		}
	}
	return n
}

func (s *Service) Route(ctx context.Context, task string, selfBase string) []Route {
	ws := words(task)
	out := []Route{{Member: Member{Product: self.ID, Purpose: self.Purpose, Objects: self.Objects, Base: selfBase, MCP: selfBase + "/mcp", Version: s.version, Reachable: true}, Score: score(ws, self.Words) * 3}}
	for _, m := range s.List(ctx) {
		var keys []string
		for _, p := range siblings {
			if p.ID == m.Product {
				keys = p.Words
			}
		}
		r := Route{Member: m, Score: score(ws, keys) * 3, Tools: []Tool{}}
		type hit struct {
			t Tool
			n int
		}
		var hits []hit
		for _, t := range m.tools {
			n := score(ws, words(strings.ReplaceAll(t.Name, "_", " ")+" "+t.Description))
			if n > 0 {
				hits = append(hits, hit{t, n})
			}
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].n > hits[j].n })
		for i, h := range hits {
			if i == 5 {
				break
			}
			d := h.t.Description
			if len(d) > 200 {
				d = d[:200]
			}
			r.Tools = append(r.Tools, Tool{Name: h.t.Name, Description: d})
			r.Score += h.n
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "family" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	r.HandleFunc("GET /.well-known/form-family.json", func(w http.ResponseWriter, req *http.Request) {
		httpx.JSON(w, http.StatusOK, m.s.Descriptor(httpx.RequestOrigin(req, m.s.auth.TrustProxy())))
	})
	r.HandleFunc("GET /api/family", m.s.auth.Require(func(w http.ResponseWriter, req *http.Request, _ *policy.Principal) error {
		httpx.JSON(w, http.StatusOK, map[string]any{"self": m.s.Descriptor(httpx.RequestOrigin(req, m.s.auth.TrustProxy())), "members": m.s.List(req.Context())})
		return nil
	}))
	return nil
}
