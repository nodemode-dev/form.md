package unfurl

import (
	"context"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/files"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/netguard"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
)

const (
	maxHead    = 1 << 20
	cacheSize  = 256
	cacheTTL   = time.Hour
	imageLimit = 5 << 20
	iconLimit  = 256 << 10
)

type Preview struct {
	URL         string `json:"url"`
	Kind        string `json:"kind"`
	Mime        string `json:"mime,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Site        string `json:"site"`
	Image       string `json:"-"`
	Icon        string `json:"-"`
	Frameable   bool   `json:"frameable"`
}

type entry struct {
	at time.Time
	pv Preview
}

type Service struct {
	client *http.Client
	pages  *page.Service
	files  *files.Service
	log    *slog.Logger
	slots  chan struct{}
	mu     sync.Mutex
	cache  map[string]entry
}

func New(client *http.Client, pages *page.Service, fsvc *files.Service, log *slog.Logger) *Service {
	return &Service{client: client, pages: pages, files: fsvc, log: log, slots: make(chan struct{}, 8), cache: map[string]entry{}}
}

func (s *Service) cached(key string) (Preview, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.cache[key]
	if !ok || time.Since(e.at) > cacheTTL {
		return Preview{}, false
	}
	return e.pv, true
}

func (s *Service) remember(key string, pv Preview) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cache) >= cacheSize {
		var oldest string
		var at time.Time
		for k, e := range s.cache {
			if oldest == "" || e.at.Before(at) {
				oldest, at = k, e.at
			}
		}
		delete(s.cache, oldest)
	}
	s.cache[key] = entry{at: time.Now(), pv: pv}
}

func (s *Service) Preview(ctx context.Context, raw string) (Preview, error) {
	u, err := netguard.CheckURL(raw)
	if err != nil {
		return Preview{}, err
	}
	key := u.String()
	if pv, ok := s.cached(key); ok {
		return pv, nil
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-ctx.Done():
		return Preview{}, ctx.Err()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, key, nil)
	if err != nil {
		return Preview{}, netguard.ErrURL
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; form.md/"+app.Version+"; link preview)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5")
	req.Header.Set("Accept-Language", "en;q=0.9, *;q=0.5")
	resp, err := s.client.Do(req)
	if err != nil {
		return Preview{}, err
	}
	defer resp.Body.Close()
	final := resp.Request.URL
	pv := Preview{URL: final.String(), Site: strings.TrimPrefix(final.Hostname(), "www."), Frameable: frameable(resp.Header)}
	ctype, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	pv.Mime = ctype
	switch {
	case resp.StatusCode >= 400:
		pv.Kind = "page"
		pv.Frameable = false
	case ctype == "text/html" || ctype == "application/xhtml+xml" || ctype == "":
		pv.Kind = "page"
		head(io.LimitReader(resp.Body, maxHead), final, &pv)
	case strings.HasPrefix(ctype, "image/"):
		pv.Kind = "image"
	case strings.HasPrefix(ctype, "video/"):
		pv.Kind = "video"
	case strings.HasPrefix(ctype, "audio/"):
		pv.Kind = "audio"
	default:
		pv.Kind = "file"
	}
	if pv.Title == "" && pv.Kind != "page" {
		if name, err := url.PathUnescape(path.Base(final.Path)); err == nil && name != "/" && name != "." {
			pv.Title = clean(name, 300)
		}
	}
	if pv.Icon == "" && pv.Kind == "page" {
		pv.Icon = final.ResolveReference(&url.URL{Path: "/favicon.ico"}).String()
	}
	s.remember(key, pv)
	return pv, nil
}

func frameable(h http.Header) bool {
	switch strings.ToLower(strings.TrimSpace(h.Get("X-Frame-Options"))) {
	case "deny", "sameorigin":
		return false
	}
	for _, csp := range h.Values("Content-Security-Policy") {
		for _, d := range strings.Split(csp, ";") {
			f := strings.Fields(strings.ToLower(d))
			if len(f) == 0 || f[0] != "frame-ancestors" {
				continue
			}
			for _, v := range f[1:] {
				if v == "*" || v == "https:" {
					return true
				}
			}
			return false
		}
	}
	return true
}

func clean(s string, max int) string {
	s = strings.Join(strings.Fields(strings.ToValidUTF8(s, "")), " ")
	if utf8.RuneCountInString(s) > max {
		s = strings.TrimSpace(string([]rune(s)[:max-1])) + "…"
	}
	return s
}

type icon struct {
	href string
	size int
	rank int
}

func head(r io.Reader, base *url.URL, pv *Preview) {
	z := html.NewTokenizer(r)
	meta := map[string]string{}
	var title strings.Builder
	inTitle, seenTitle := false, false
	var best icon
	abs := func(ref string) string {
		u, err := base.Parse(strings.TrimSpace(ref))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return ""
		}
		return u.String()
	}
loop:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			break loop
		case html.TextToken:
			if inTitle && title.Len() < 2000 {
				title.Write(z.Text())
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "title":
				inTitle = false
			case "head":
				break loop
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, more := z.TagName()
			attrs := map[string]string{}
			for more {
				var k, v []byte
				k, v, more = z.TagAttr()
				attrs[string(k)] = string(v)
			}
			switch string(name) {
			case "body":
				break loop
			case "title":
				if !seenTitle && tt == html.StartTagToken {
					inTitle, seenTitle = true, true
				}
			case "meta":
				key := strings.ToLower(attrs["property"])
				if key == "" {
					key = strings.ToLower(attrs["name"])
				}
				if key != "" && meta[key] == "" {
					meta[key] = attrs["content"]
				}
			case "link":
				rel := " " + strings.ToLower(attrs["rel"]) + " "
				if !strings.Contains(rel, " icon ") && !strings.Contains(rel, " apple-touch-icon ") {
					continue
				}
				c := icon{href: abs(attrs["href"]), rank: 1}
				if strings.Contains(rel, "apple-touch-icon") {
					c.rank, c.size = 2, 180
				}
				if w, _, ok := strings.Cut(strings.ToLower(attrs["sizes"]), "x"); ok {
					c.size, _ = strconv.Atoi(w)
				}
				if c.href != "" && (best.href == "" || better(c, best)) {
					best = c
				}
			}
		}
	}
	first := func(keys ...string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(meta[k]); v != "" {
				return v
			}
		}
		return ""
	}
	pv.Title = clean(first("og:title", "twitter:title"), 300)
	if pv.Title == "" {
		pv.Title = clean(title.String(), 300)
	}
	pv.Description = clean(first("og:description", "twitter:description", "description"), 500)
	if site := clean(first("og:site_name", "application-name"), 120); site != "" {
		pv.Site = site
	}
	if img := first("og:image:secure_url", "og:image", "og:image:url", "twitter:image", "twitter:image:src"); img != "" {
		pv.Image = abs(img)
	}
	pv.Icon = best.href
}

func better(a, b icon) bool {
	score := func(c icon) int {
		if c.size > 256 {
			return c.rank*1000 - c.size
		}
		return c.rank*1000 + c.size
	}
	return score(a) > score(b)
}

type result struct {
	Preview
	Image string `json:"image"`
	Icon  string `json:"icon"`
	W     int    `json:"w,omitempty"`
	H     int    `json:"h,omitempty"`
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	q := r.URL.Query()
	pageID := q.Get("page_id")
	if _, err := s.pages.CanWrite(r.Context(), p, pageID); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	pv, err := s.Preview(ctx, q.Get("url"))
	if err != nil {
		code, msg := netguard.Reason(err)
		status := http.StatusBadGateway
		if code == "bad_url" || code == "private_address" {
			status = http.StatusBadRequest
		}
		return httpx.Fail(status, code, msg)
	}
	out := result{Preview: pv}
	var wg sync.WaitGroup
	pull := func(raw string, limit int64, ref *string, sized bool) {
		defer wg.Done()
		if raw == "" {
			return
		}
		f, err := s.files.Import(ctx, p, raw, files.ImportOptions{Purpose: "attachment", PageID: pageID, Limit: limit, ImagesOnly: true})
		if err != nil {
			s.log.Debug("unfurl: image skipped", "url", raw, "err", err)
			return
		}
		*ref = f.Ref
		if sized {
			out.W, out.H = f.W, f.H
		}
	}
	wg.Add(2)
	go pull(pv.Image, imageLimit, &out.Image, true)
	go pull(pv.Icon, iconLimit, &out.Icon, false)
	wg.Wait()
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module(a *auth.Service) app.Module { return module{s, a} }

func (module) ID() string { return "unfurl" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	r.HandleFunc("GET /api/unfurl", m.auth.Require(m.s.handle))
	return nil
}
