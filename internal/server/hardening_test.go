package server_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/nodemode-ai/form.md/internal/config"
)

func direct(e *env, method, path, cookie string, edit func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = "127.0.0.1:50000"
	req.Host = "127.0.0.1:8750"
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if edit != nil {
		edit(req)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func errorCode(rec *httptest.ResponseRecorder) string {
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Error.Code
}

func TestAutomaticHTTPSTurnsTheTerminalOff(t *testing.T) {
	t.Setenv("FORMMD_TERMINAL", "")
	dir := t.TempDir()
	first := newEnvAt(t, "", dir)
	owner := first.setupOwner()
	cookie := owner.cookieHeader()
	if err := first.srv.Settings.Set(context.Background(), map[string]string{"https.auto": "true", "https.domain": "md.example.test"}); err != nil {
		t.Fatal(err)
	}
	first.stop()

	second := newEnvWith(t, config.Config{Addr: "127.0.0.1:8750", DataDir: dir, Family: "off"})
	if got := second.srv.Deps.Config.Addr; got != ":443" {
		t.Fatalf("automatic HTTPS left the address at %q", got)
	}
	rec := direct(second, "GET", "/api/terminal", cookie, nil)
	if rec.Code != http.StatusForbidden || errorCode(rec) != "terminal_off" {
		t.Fatalf("a server moved to :443 still opens terminals: %d %s", rec.Code, rec.Body)
	}
}

func TestAutomaticHTTPSIsIgnoredByTheDesktopApp(t *testing.T) {
	dir := t.TempDir()
	first := newEnvAt(t, desktopKey, dir)
	if err := first.srv.Settings.Set(context.Background(), map[string]string{"https.auto": "true", "https.domain": "md.example.test"}); err != nil {
		t.Fatal(err)
	}
	first.stop()

	second := newEnvWith(t, config.Config{Addr: "127.0.0.1:8750", DataDir: dir, DesktopKey: desktopKey, Family: "off"})
	if cfg := second.srv.Deps.Config; cfg.Addr != "127.0.0.1:8750" || len(cfg.ACMEDomains) != 0 {
		t.Fatalf("the desktop app took automatic HTTPS: addr %q, domains %v", cfg.Addr, cfg.ACMEDomains)
	}
}

func TestTerminalAnswersOnlyDirectRequests(t *testing.T) {
	t.Setenv("FORMMD_TERMINAL", "")
	e := newEnv(t, "")
	cookie := e.setupOwner().cookieHeader()
	if rec := direct(e, "GET", "/api/terminal", cookie, nil); rec.Code != http.StatusOK {
		t.Fatalf("a direct request from this computer: %d %s", rec.Code, rec.Body)
	}
	cases := map[string]func(*http.Request){
		"forwarded for":    func(r *http.Request) { r.Header.Set("X-Forwarded-For", "203.0.113.9") },
		"forwarded":        func(r *http.Request) { r.Header.Set("Forwarded", "for=203.0.113.9") },
		"real ip":          func(r *http.Request) { r.Header.Set("X-Real-IP", "203.0.113.9") },
		"tunnel":           func(r *http.Request) { r.TLS = &tls.ConnectionState{}; r.RemoteAddr = "203.0.113.9:0" },
		"public host":      func(r *http.Request) { r.Host = "md.example.test" },
		"public peer":      func(r *http.Request) { r.RemoteAddr = "203.0.113.9:50000" },
		"lan peer":         func(r *http.Request) { r.RemoteAddr = "192.168.1.20:50000" },
		"proxy with https": func(r *http.Request) { r.TLS = &tls.ConnectionState{} },
	}
	for name, edit := range cases {
		rec := direct(e, "GET", "/api/terminal", cookie, edit)
		if rec.Code != http.StatusForbidden || errorCode(rec) != "terminal_off" {
			t.Errorf("%s: %d %s, want 403 terminal_off", name, rec.Code, rec.Body)
		}
	}
	status := direct(e, "GET", "/api/claude", cookie, cases["forwarded for"])
	if status.Code != http.StatusOK || strings.Contains(status.Body.String(), `"enabled":true`) {
		t.Errorf("the status route offers the terminal behind a proxy: %d %s", status.Code, status.Body)
	}
	t.Setenv("FORMMD_TERMINAL", "on")
	if rec := direct(e, "GET", "/api/terminal", cookie, cases["forwarded for"]); rec.Code != http.StatusOK {
		t.Fatalf("FORMMD_TERMINAL=on behind a proxy: %d %s", rec.Code, rec.Body)
	}
}

func TestDesktopAnswersOnlyItsOwnAddress(t *testing.T) {
	e := newEnv(t, desktopKey)
	for _, host := range []string{"rebind.example.test:8750", "rebind.example.test", "192.168.1.20:8750"} {
		rec := direct(e, "GET", "/api/me", "", func(r *http.Request) { r.Host = host })
		if rec.Code != http.StatusMisdirectedRequest {
			t.Errorf("host %q: %d, want 421", host, rec.Code)
		}
	}
	for _, host := range []string{"127.0.0.1:8750", "localhost:8750", "[::1]:8750"} {
		rec := direct(e, "GET", "/api/me", "", func(r *http.Request) { r.Host = host })
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("host %q: %d, want 401 (answered, not signed in)", host, rec.Code)
		}
	}
	if rec := direct(e, "GET", "/api/me", "", nil); rec.Code == http.StatusMisdirectedRequest {
		t.Fatal("the desktop app refused its own address")
	}
	tunnelled := direct(e, "GET", "/api/me", "", func(r *http.Request) {
		r.Host = "quick-words.trycloudflare.com"
		*r = *r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey, &net.UnixAddr{Name: "/data/tunnel.sock", Net: "unix"}))
	})
	if tunnelled.Code == http.StatusMisdirectedRequest {
		t.Error("the desktop app refused a request from its own tunnel socket")
	}
	if r := newEnv(t, "").client().do("GET", "/api/me", nil, func(r *http.Request) { r.Host = "md.example.test" }); r.Status == http.StatusMisdirectedRequest {
		t.Fatal("a self-hosted server refused a public host name")
	}
}

func TestDesktopHasNoFirstRunSetup(t *testing.T) {
	e := newEnv(t, desktopKey)
	c := e.client()
	r := c.do("POST", "/api/setup", map[string]string{"name": "Mallory", "email": "mallory@example.test", "password": "correct-horse-1"})
	if r.Status != http.StatusNotFound {
		t.Fatalf("desktop setup answered %d %s", r.Status, r.Body)
	}
	if r := c.do("GET", "/desktop/enter?key="+desktopKey, nil); r.Status != http.StatusSeeOther {
		t.Fatalf("enter after a refused setup: %d", r.Status)
	}
	me := c.mustJSON("GET", "/api/me", nil, http.StatusOK)
	if u, _ := me["user"].(map[string]any); u == nil || u["email"] == "mallory@example.test" {
		t.Fatalf("the window signed into the stranger's account: %v", me)
	}
}

type keyCatcher struct {
	mu   sync.Mutex
	seen []string
}

func (k *keyCatcher) server(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k.mu.Lock()
		k.seen = append(k.seen, r.Header.Get("Authorization")+r.Header.Get("X-Api-Key"))
		k.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(ts.Close)
	return ts
}

func (k *keyCatcher) got(secret string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, s := range k.seen {
		if strings.Contains(s, secret) {
			return true
		}
	}
	return false
}

func TestChangingTheAIAddressDropsTheSavedKey(t *testing.T) {
	const secret = "sk-owner-secret-0123456789"
	var home, other keyCatcher
	homeTS, otherTS := home.server(t), other.server(t)
	e := newEnv(t, "")
	owner := e.setupOwner()
	created := owner.mustJSON("POST", "/api/admin/ai/providers", map[string]string{"kind": "compatible", "name": "Local", "base_url": homeTS.URL, "api_key": secret}, http.StatusCreated)
	id, _ := created["id"].(string)
	if !home.got(secret) {
		t.Fatal("the provider's own server never saw its key; the test cannot tell anything")
	}
	if r := owner.do("PATCH", "/api/admin/ai/providers/"+id, map[string]string{"name": "Renamed", "base_url": homeTS.URL}); r.Status != http.StatusNoContent {
		t.Fatalf("rename: %d %s", r.Status, r.Body)
	}
	if r := owner.do("POST", "/api/admin/ai/providers/"+id+"/refresh", nil); r.Status >= 500 {
		t.Fatalf("refresh at the same address: %d %s", r.Status, r.Body)
	}
	if r := owner.do("PATCH", "/api/admin/ai/providers/"+id, map[string]string{"base_url": otherTS.URL}); r.Status != http.StatusNoContent {
		t.Fatalf("move: %d %s", r.Status, r.Body)
	}
	owner.do("POST", "/api/admin/ai/providers/"+id+"/refresh", nil)
	if other.got(secret) {
		t.Fatal("moving the provider to another address sent it the saved key")
	}
	if r := owner.do("PATCH", "/api/admin/ai/providers/"+id, map[string]string{"base_url": otherTS.URL, "api_key": "sk-new-key-for-other"}); r.Status != http.StatusNoContent {
		t.Fatalf("move with a new key: %d %s", r.Status, r.Body)
	}
	owner.do("POST", "/api/admin/ai/providers/"+id+"/refresh", nil)
	if !other.got("sk-new-key-for-other") {
		t.Fatal("a key given with the new address was not kept")
	}
}
