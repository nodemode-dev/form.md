package family

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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

type Tuning struct {
	VocabularyWeight int
	ToolWeight       int
	MaxTools         int
	Cache            time.Duration
	Timeout          time.Duration
}

type Service struct {
	auth    *auth.Service
	version string
	bases   map[string]string
	off     bool
	client  *http.Client
	tune    Tuning

	mu     sync.Mutex
	cache  map[string]cached
	flight map[string]chan struct{}
}

type cached struct {
	m  Member
	at time.Time
}

func New(a *auth.Service, version, spec string, t Tuning) *Service {
	s := &Service{auth: a, version: version, bases: map[string]string{}, cache: map[string]cached{}, flight: map[string]chan struct{}{}, tune: t,
		client: &http.Client{Timeout: t.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
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

func sameHost(base, raw string) bool {
	if !loopback(raw) {
		return false
	}
	b, err1 := url.Parse(base)
	u, err2 := url.Parse(strings.TrimSpace(raw))
	return err1 == nil && err2 == nil && b.Host == u.Host
}

var toolName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
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
	for {
		if c, ok := s.cache[p.ID]; ok && time.Since(c.at) < s.tune.Cache {
			s.mu.Unlock()
			return c.m
		}
		wait, busy := s.flight[p.ID]
		if !busy {
			break
		}
		s.mu.Unlock()
		select {
		case <-wait:
		case <-ctx.Done():
			m.Problem = "the lookup was cancelled"
			return m
		}
		s.mu.Lock()
	}
	done := make(chan struct{})
	s.flight[p.ID] = done
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.flight, p.ID)
		close(done)
		s.mu.Unlock()
	}()
	var health struct {
		Version string `json:"version"`
	}
	if err := s.get(ctx, base+p.Health, &health); err != nil {
		m.Problem = "not running at " + base
	} else {
		m.Reachable, m.Version = true, clip(health.Version, 32)
		var status struct {
			Endpoint string `json:"endpoint"`
		}
		if s.get(ctx, base+p.Status, &status) == nil && status.Endpoint != "" && sameHost(base, status.Endpoint) {
			m.MCP = status.Endpoint
		} else {
			m.MCP = base + "/mcp"
		}
		if tools, err := s.listTools(ctx, m.MCP); err == nil {
			kept := tools[:0]
			for _, t := range tools {
				if toolName.MatchString(t.Name) {
					kept = append(kept, t)
				}
			}
			m.tools, m.ToolCount = kept, len(kept)
		} else {
			m.Problem = "running, but its tool list could not be read"
		}
	}
	if ctx.Err() == nil {
		s.mu.Lock()
		s.cache[p.ID] = cached{m: m, at: time.Now()}
		s.mu.Unlock()
	}
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
	vw, tw := s.tune.VocabularyWeight, s.tune.ToolWeight
	me := Member{Product: self.ID, Purpose: self.Purpose, Objects: self.Objects, Version: s.version, Reachable: true}
	if selfBase != "" {
		me.Base, me.MCP = selfBase, selfBase+"/mcp"
	}
	out := []Route{{Member: me, Score: score(ws, self.Words) * vw, Tools: []Tool{}}}
	for _, m := range s.List(ctx) {
		var keys []string
		for _, p := range siblings {
			if p.ID == m.Product {
				keys = p.Words
			}
		}
		r := Route{Member: m, Score: score(ws, keys) * vw, Tools: []Tool{}}
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
			if i == s.tune.MaxTools {
				break
			}
			r.Tools = append(r.Tools, Tool{Name: h.t.Name, Description: clip(h.t.Description, 200)})
			r.Score += h.n * tw
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func (s *Service) Suggest(ctx context.Context, task string, minScore, maxTools int) []Route {
	ws := words(task)
	hit := false
	for _, p := range siblings {
		hit = hit || score(ws, p.Words) > 0
	}
	if !hit || s.off {
		return nil
	}
	routes := s.Route(ctx, task, "")
	mine := 0
	for _, r := range routes {
		if r.Product == self.ID {
			mine = r.Score
		}
	}
	var out []Route
	for _, r := range routes {
		if r.Product == self.ID || r.Score < minScore || r.Score <= mine {
			continue
		}
		if len(r.Tools) > maxTools {
			r.Tools = r.Tools[:maxTools]
		}
		out = append(out, r)
	}
	return out
}

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "family" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	r.HandleFunc("GET /.well-known/form-family.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		httpx.JSON(w, http.StatusOK, m.s.Descriptor(m.s.auth.BaseURL(req)))
	})
	r.HandleFunc("GET /api/family", m.s.auth.Require(func(w http.ResponseWriter, req *http.Request, _ *policy.Principal) error {
		httpx.JSON(w, http.StatusOK, map[string]any{"self": m.s.Descriptor(m.s.auth.BaseURL(req)), "members": m.s.List(req.Context())})
		return nil
	}))
	return nil
}
