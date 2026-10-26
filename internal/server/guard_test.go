package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func desktopEnter(e *env, remote, key string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/desktop/enter?key="+key, nil)
	req.RemoteAddr = remote
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func TestDesktopEnterNeedsTheKey(t *testing.T) {
	e := newEnv(t, desktopKey)
	for _, key := range []string{"", "wrong", desktopKey[:len(desktopKey)-1], desktopKey + "x"} {
		rec := desktopEnter(e, "127.0.0.1:50000", key, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("key %q: status %d, want 404", key, rec.Code)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Fatalf("key %q: a refused entry set a cookie", key)
		}
	}
	rec := desktopEnter(e, "127.0.0.1:50000", desktopKey, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/home?start=1" {
		t.Fatalf("right key: status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Fatal("right key signed nobody in")
	}
}

func TestDesktopEnterOnlyFromLoopback(t *testing.T) {
	e := newEnv(t, desktopKey)
	cases := []struct {
		name, remote string
		hdr          map[string]string
	}{
		{"lan peer", "192.168.1.20:50000", nil},
		{"public peer", "203.0.113.9:50000", nil},
		{"forwarded loopback", "203.0.113.9:50000", map[string]string{"X-Forwarded-For": "127.0.0.1", "X-Real-IP": "127.0.0.1"}},
		{"loopback host header", "203.0.113.9:50000", map[string]string{"Host": "127.0.0.1:8750"}},
	}
	for _, c := range cases {
		rec := desktopEnter(e, c.remote, desktopKey, c.hdr)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", c.name, rec.Code)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s: a refused entry set a cookie", c.name)
		}
	}
	for _, remote := range []string{"127.0.0.1:50000", "[::1]:50000"} {
		if rec := desktopEnter(e, remote, desktopKey, nil); rec.Code != http.StatusSeeOther {
			t.Errorf("loopback peer %s: status %d, want 303", remote, rec.Code)
		}
	}
}

func TestDesktopEnterIsOffWithoutAKey(t *testing.T) {
	e := newEnv(t, "")
	for _, key := range []string{"", desktopKey, "anything"} {
		if rec := desktopEnter(e, "127.0.0.1:50000", key, nil); rec.Code != http.StatusNotFound {
			t.Errorf("no desktop key configured, key %q: status %d, want 404", key, rec.Code)
		}
	}
}

func TestForeignOriginCannotChangeAnythingWithoutABearer(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	own := e.ts.URL
	foreign := []string{"https://evil.example", "http://localhost.evil.com", "http://127.0.0.1:1", "null", own + ".evil.example", own + "0"}
	for _, origin := range foreign {
		for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
			path := "/api/pages"
			switch method {
			case "PUT":
				path = "/api/favorites/none"
			case "PATCH":
				path = "/api/me"
			case "DELETE":
				path = "/api/pages/none"
			}
			r := owner.do(method, path, map[string]string{"workspace_id": ws, "title": "x"}, withHeader("Origin", origin))
			if r.Status != http.StatusForbidden || r.Code(t) != "bad_origin" {
				t.Errorf("%s %s from %q: status %d %s, want 403 bad_origin", method, path, origin, r.Status, r.Body)
			}
		}
	}
	list := owner.mustJSON("GET", "/api/workspaces/"+ws+"/pages", nil, http.StatusOK)
	if got, _ := list["pages"].([]any); len(got) != 0 {
		t.Fatalf("a refused request created %d pages", len(got))
	}
}

func TestOwnOriginAndNoOriginAreAllowedToWrite(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	for _, hdr := range []opt{withHeader("Origin", e.ts.URL), func(*http.Request) {}} {
		r := owner.do("POST", "/api/pages", map[string]string{"workspace_id": ws, "title": "ok"}, hdr)
		if r.Status != http.StatusCreated {
			t.Fatalf("status %d: %s", r.Status, r.Body)
		}
	}
	if r := owner.do("GET", "/api/me", nil, withHeader("Origin", "https://evil.example")); r.Status != http.StatusOK {
		t.Fatalf("a read from another origin: status %d, want 200 (reads carry no side effects)", r.Status)
	}
}

func TestBearerRequestsAreNotOriginGated(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	tok := owner.apiToken("write")
	anon := e.client()
	r := anon.do("POST", "/api/pages", map[string]string{"workspace_id": ws, "title": "by token"}, withBearer(tok), withHeader("Origin", "https://tool.example"))
	if r.Status != http.StatusCreated {
		t.Fatalf("token write from a tool: status %d: %s", r.Status, r.Body)
	}
	r = anon.do("POST", "/api/pages", map[string]string{"workspace_id": ws, "title": "no cred"}, withHeader("Origin", "https://tool.example"))
	if r.Status != http.StatusForbidden {
		t.Fatalf("no credential from a foreign origin: status %d, want 403", r.Status)
	}
	r = anon.do("POST", "/api/pages", map[string]string{"workspace_id": ws, "title": "bad token"}, withBearer("fmd_nope"), withHeader("Origin", "https://tool.example"))
	if r.Status != http.StatusUnauthorized {
		t.Fatalf("bad token: status %d, want 401", r.Status)
	}
}

func dialSync(e *env, c *client, origin string) (*websocket.Conn, *http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h := http.Header{}
	if origin != "" {
		h.Set("Origin", origin)
	}
	if c != nil {
		h.Set("Cookie", c.cookieHeader())
	}
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(e.ts.URL, "http")+"/sync", &websocket.DialOptions{HTTPHeader: h})
}

func TestSyncSocketRefusesAnotherOrigin(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	for _, origin := range []string{"https://evil.example", "http://localhost.evil.com", "http://127.0.0.1:1", "null", ""} {
		conn, resp, err := dialSync(e, owner, origin)
		if err == nil {
			conn.CloseNow()
			t.Errorf("origin %q: the socket opened", origin)
			continue
		}
		if resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("origin %q: want a 403 refusal, got %v (%v)", origin, resp, err)
		}
	}
	conn, resp, err := dialSync(e, owner, e.ts.URL)
	if err != nil {
		t.Fatalf("own origin refused: %v (%v)", err, resp)
	}
	conn.CloseNow()
}

func TestSyncSocketNeedsASession(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	conn, resp, err := dialSync(e, nil, e.ts.URL)
	if err == nil {
		conn.CloseNow()
		t.Fatal("an anonymous socket opened")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %v", resp)
	}
	tok := owner.apiToken("write")
	hdr := http.Header{"Authorization": {"Bearer " + tok}, "Origin": {e.ts.URL}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, resp, err = websocket.Dial(ctx, "ws"+strings.TrimPrefix(e.ts.URL, "http")+"/sync", &websocket.DialOptions{HTTPHeader: hdr})
	if err == nil {
		conn.CloseNow()
		t.Fatal("an API token opened the editing socket")
	}
	if resp == nil || resp.StatusCode < 400 {
		t.Fatalf("token on the socket: got %v", resp)
	}
}
