package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/i18n"
	"github.com/nodemode-ai/form.md/internal/server"
	"github.com/nodemode-ai/form.md/internal/store"
)

const desktopKey = "0123456789abcdef0123456789abcdef-desk"

var i18nOnce sync.Once

type env struct {
	t       *testing.T
	srv     *server.Server
	handler http.Handler
	ts      *httptest.Server
	dir     string
	stop    func()
}

func newEnv(t *testing.T, key string) *env {
	t.Helper()
	return newEnvAt(t, key, t.TempDir())
}

func newEnvAt(t *testing.T, key, dir string) *env {
	t.Helper()
	i18nOnce.Do(func() {
		if err := i18n.LoadFS(os.DirFS("../../web/src/i18n")); err != nil {
			t.Fatalf("catalogs: %v", err)
		}
	})
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, err := store.Open(ctx, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, err := st.Bootstrap(ctx, app.Version)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Addr: "127.0.0.1:0", DataDir: dir, DesktopKey: key, Family: "off"}
	dist := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>t</title>")}}
	srv, err := server.Build(ctx, cfg, log, st, dist, os.DirFS("../../.claude/skills/contextual-compression"), "", id)
	if err != nil {
		t.Fatal(err)
	}
	r := httpx.NewRouter(log)
	r.Use(srv.Deps.Middleware...)
	for _, m := range srv.Modules {
		if _, ok := m.(app.ExtraListener); ok {
			continue
		}
		if err := m.Mount(r, srv.Deps); err != nil {
			t.Fatalf("mount %s: %v", m.ID(), err)
		}
	}
	h := r.Handler()
	ts := httptest.NewServer(h)
	e := &env{t: t, srv: srv, handler: h, ts: ts, dir: dir}
	var once sync.Once
	e.stop = func() {
		once.Do(func() {
			ts.Close()
			for _, m := range srv.Modules {
				if s, ok := m.(app.Stopper); ok {
					s.Stop()
				}
			}
			dctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for _, m := range srv.Modules {
				if d, ok := m.(app.Drainer); ok {
					d.Drain(dctx)
				}
			}
			_ = st.Close()
		})
	}
	t.Cleanup(e.stop)
	return e
}

type client struct {
	e    *env
	http *http.Client
}

func (e *env) client() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type reply struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r reply) JSON(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("not JSON (%d): %s", r.Status, r.Body)
	}
	return m
}

func (r reply) Code(t *testing.T) string {
	t.Helper()
	e, _ := r.JSON(t)["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

type opt func(*http.Request)

func withHeader(k, v string) opt { return func(r *http.Request) { r.Header.Set(k, v) } }
func withBearer(tok string) opt  { return withHeader("Authorization", "Bearer "+tok) }

func (c *client) do(method, path string, body any, opts ...opt) reply {
	c.e.t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rd = strings.NewReader(b)
	case []byte:
		rd = bytes.NewReader(b)
	default:
		j, err := json.Marshal(b)
		if err != nil {
			c.e.t.Fatal(err)
		}
		rd = bytes.NewReader(j)
	}
	req, err := http.NewRequest(method, c.e.ts.URL+path, rd)
	if err != nil {
		c.e.t.Fatal(err)
	}
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, o := range opts {
		o(req)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return reply{Status: resp.StatusCode, Body: b, Header: resp.Header}
}

func (c *client) mustJSON(method, path string, body any, want int, opts ...opt) map[string]any {
	c.e.t.Helper()
	r := c.do(method, path, body, opts...)
	if r.Status != want {
		c.e.t.Fatalf("%s %s: status %d, want %d: %s", method, path, r.Status, want, r.Body)
	}
	if len(r.Body) == 0 {
		return nil
	}
	return r.JSON(c.e.t)
}

func (c *client) cookieHeader() string {
	u, err := url.Parse(c.e.ts.URL)
	if err != nil {
		c.e.t.Fatal(err)
	}
	var parts []string
	for _, ck := range c.http.Jar.Cookies(u) {
		parts = append(parts, ck.Name+"="+ck.Value)
	}
	return strings.Join(parts, "; ")
}

func (e *env) setupOwner() *client {
	c := e.client()
	c.mustJSON("POST", "/api/setup", map[string]string{"name": "Olive Owner", "email": "owner@example.test", "password": "correct-horse-1"}, http.StatusOK)
	return c
}

func (c *client) workspaceID() string {
	c.e.t.Helper()
	me := c.mustJSON("GET", "/api/me", nil, http.StatusOK)
	list, _ := me["workspaces"].([]any)
	var first string
	for _, w := range list {
		m := w.(map[string]any)
		if first == "" {
			first = m["id"].(string)
		}
		if p, _ := m["personal"].(bool); !p {
			return m["id"].(string)
		}
	}
	if first == "" {
		c.e.t.Fatalf("no workspace in /api/me: %v", me)
	}
	return first
}

func (c *client) createPage(ws, title string) string {
	c.e.t.Helper()
	res := c.mustJSON("POST", "/api/pages", map[string]string{"workspace_id": ws, "title": title}, http.StatusCreated)
	return res["page"].(map[string]any)["id"].(string)
}

func (c *client) putMarkdown(page, md string) {
	c.e.t.Helper()
	c.mustJSON("PUT", "/api/pages/"+page+"/markdown", map[string]string{"markdown": md, "mode": "replace"}, http.StatusNoContent)
}

func (c *client) apiToken(scope string) string {
	c.e.t.Helper()
	res := c.mustJSON("POST", "/api/me/tokens", map[string]any{"name": "t-" + scope, "scope": scope}, http.StatusCreated)
	return res["token"].(string)
}

func (c *client) joinAs(ws, email, role string) *client {
	c.e.t.Helper()
	inv := c.mustJSON("POST", "/api/workspaces/"+ws+"/invites", map[string]string{"email": email, "role": role}, http.StatusCreated)
	link := inv["url"].(string)
	token := link[strings.LastIndex(link, "/")+1:]
	other := c.e.client()
	other.mustJSON("POST", "/api/invites/"+token+"/accept", map[string]string{"name": "Guest Person", "email": email, "password": "another-pass-9"}, http.StatusOK)
	return other
}
