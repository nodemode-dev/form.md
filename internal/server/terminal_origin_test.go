package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestTerminalAttachesOnlyFromTheServersOwnPage(t *testing.T) {
	t.Setenv("FORMMD_TERMINAL", "")
	e := newEnv(t, "")
	owner := e.setupOwner()
	r := owner.do("POST", "/api/terminal", map[string]any{"kind": "shell", "cols": 80, "rows": 24})
	if r.Status != http.StatusCreated {
		t.Fatalf("create terminal: %d %s", r.Status, r.Body)
	}
	var term struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(r.Body, &term); err != nil || term.ID == "" {
		t.Fatalf("terminal id: %v %s", err, r.Body)
	}
	t.Cleanup(func() { owner.do("DELETE", "/api/terminal/"+term.ID, nil) })
	url := "ws" + strings.TrimPrefix(e.ts.URL, "http") + "/api/terminal/" + term.ID + "/attach"
	attach := func(origin string) (*websocket.Conn, *http.Response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h := http.Header{"Cookie": {owner.cookieHeader()}, "Origin": {origin}}
		return websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: h})
	}
	c, _, err := attach(e.ts.URL)
	if err != nil {
		t.Fatalf("the server's own page could not attach: %v", err)
	}
	c.CloseNow()
	port := e.ts.URL[strings.LastIndex(e.ts.URL, ":")+1:]
	for _, origin := range []string{"http://127.0.0.1:1", "http://localhost:" + port, "http://[::1]:" + port, "https://md.example.test"} {
		c, resp, err := attach(origin)
		if err == nil {
			c.CloseNow()
			t.Errorf("%s attached to the terminal", origin)
			continue
		}
		if resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s: %v, want 403", origin, err)
		}
	}
}
