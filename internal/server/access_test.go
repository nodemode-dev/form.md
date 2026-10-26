package server_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestViewerCanReadButNotWrite(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Plan")
	owner.putMarkdown(page, "# Secret plan\n\nlaunch on friday")
	viewer := owner.joinAs(ws, "viewer@example.test", "viewer")

	if r := viewer.do("GET", "/api/pages/"+page+"/markdown", nil); r.Status != http.StatusOK || !strings.Contains(string(r.Body), "launch on friday") {
		t.Fatalf("a viewer cannot read: %d %s", r.Status, r.Body)
	}
	writes := []struct {
		method, path string
		body         any
	}{
		{"PUT", "/api/pages/" + page + "/markdown", map[string]string{"markdown": "defaced", "mode": "replace"}},
		{"PATCH", "/api/pages/" + page, map[string]string{"title": "Defaced"}},
		{"POST", "/api/pages", map[string]string{"workspace_id": ws, "title": "Mine"}},
		{"DELETE", "/api/pages/" + page, nil},
		{"POST", "/api/pages/" + page + "/links", map[string]string{"level": "view"}},
	}
	for _, w := range writes {
		r := viewer.do(w.method, w.path, w.body)
		if r.Status < 400 || r.Status >= 500 {
			t.Errorf("viewer %s %s: status %d, want a 4xx refusal: %s", w.method, w.path, r.Status, r.Body)
		}
	}
	tok := viewer.apiToken("write")
	anon := e.client()
	r := anon.do("PUT", "/api/pages/"+page+"/markdown", map[string]string{"markdown": "defaced", "mode": "replace"}, withBearer(tok))
	if r.Status < 400 || r.Status >= 500 {
		t.Errorf("a viewer's write-scope token wrote: %d %s", r.Status, r.Body)
	}
	if got := owner.do("GET", "/api/pages/"+page+"/markdown", nil); !strings.Contains(string(got.Body), "launch on friday") || strings.Contains(string(got.Body), "defaced") {
		t.Fatalf("the page changed under a viewer: %s", got.Body)
	}
	if meta := owner.mustJSON("GET", "/api/pages/"+page, nil, http.StatusOK); meta["title"] != "Plan" {
		if p, _ := meta["page"].(map[string]any); p == nil || p["title"] != "Plan" {
			t.Fatalf("title changed under a viewer: %v", meta)
		}
	}
}

func TestReadOnlyTokenCannotWrite(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Notes")
	read := owner.apiToken("read")
	anon := e.client()
	if r := anon.do("GET", "/api/pages/"+page+"/markdown", nil, withBearer(read)); r.Status != http.StatusOK {
		t.Fatalf("read token cannot read: %d %s", r.Status, r.Body)
	}
	r := anon.do("PUT", "/api/pages/"+page+"/markdown", map[string]string{"markdown": "x"}, withBearer(read))
	if r.Status != http.StatusForbidden {
		t.Fatalf("read token write: status %d, want 403: %s", r.Status, r.Body)
	}
}

func TestRevokedShareLinkStopsWorking(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Public note")
	owner.putMarkdown(page, "shared words here")
	res := owner.mustJSON("POST", "/api/pages/"+page+"/links", map[string]string{"level": "view"}, http.StatusCreated)
	link := res["link"].(map[string]any)
	token, id := link["token"].(string), link["id"].(string)

	reader := e.client()
	first := reader.do("GET", "/l/"+token, nil)
	if first.Status != http.StatusOK || !strings.Contains(string(first.Body), "shared words here") {
		t.Fatalf("a live link does not show the page: %d", first.Status)
	}
	if r := owner.do("DELETE", "/api/links/"+id, nil); r.Status >= 300 {
		t.Fatalf("revoke: %d %s", r.Status, r.Body)
	}
	after := e.client().do("GET", "/l/"+token, nil)
	if after.Status == http.StatusOK || strings.Contains(string(after.Body), "shared words here") {
		t.Fatalf("a revoked link still serves the page: %d", after.Status)
	}
	if after.Status != http.StatusNotFound {
		t.Fatalf("revoked link: status %d, want 404", after.Status)
	}
}

func TestShareLinkNeverGivesWriteAccess(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Open page")
	r := owner.do("POST", "/api/pages/"+page+"/links", map[string]string{"level": "edit"})
	if r.Status != http.StatusBadRequest {
		t.Fatalf("an edit link was accepted: %d %s", r.Status, r.Body)
	}
	link := owner.mustJSON("POST", "/api/pages/"+page+"/links", map[string]string{"level": "comment"}, http.StatusCreated)["link"].(map[string]any)
	visitor := e.client()
	if w := visitor.do("PUT", "/api/pages/"+page+"/markdown", map[string]string{"markdown": "x"}, withHeader("Origin", e.ts.URL)); w.Status < 400 {
		t.Fatalf("an anonymous visitor wrote: %d", w.Status)
	}
	if got := visitor.do("GET", "/l/"+link["token"].(string), nil); got.Status != http.StatusOK {
		t.Fatalf("comment link: %d", got.Status)
	}
}

func (c *client) outsider(email string) *client {
	c.e.t.Helper()
	other := c.mustJSON("POST", "/api/workspaces", map[string]string{"name": "Elsewhere"}, http.StatusCreated)
	ws2 := other["workspace"].(map[string]any)["id"].(string)
	return c.joinAs(ws2, email, "member")
}

func TestPageShareLevelDecidesWhetherAGuestCanWrite(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Shared with one person")
	owner.putMarkdown(page, "body text")
	guest := owner.outsider("guest@example.test")
	if r := guest.do("GET", "/api/pages/"+page+"/markdown", nil); r.Status != http.StatusNotFound {
		t.Fatalf("an unshared page is visible to an outsider: %d", r.Status)
	}
	sh := owner.mustJSON("PUT", "/api/pages/"+page+"/shares", map[string]string{"email": "guest@example.test", "level": "view"}, http.StatusOK)
	id := sh["share"].(map[string]any)["id"].(string)
	if r := guest.do("GET", "/api/pages/"+page+"/markdown", nil); r.Status != http.StatusOK {
		t.Fatalf("a viewing guest cannot read: %d %s", r.Status, r.Body)
	}
	if r := guest.do("PUT", "/api/pages/"+page+"/markdown", map[string]string{"markdown": "guest wrote", "mode": "replace"}); r.Status < 400 || r.Status >= 500 {
		t.Fatalf("a view-level guest wrote: %d %s", r.Status, r.Body)
	}
	owner.mustJSON("PATCH", "/api/shares/"+id, map[string]string{"level": "edit"}, http.StatusNoContent)
	if r := guest.do("PUT", "/api/pages/"+page+"/markdown", map[string]string{"markdown": "guest wrote", "mode": "replace"}); r.Status != http.StatusNoContent {
		t.Fatalf("an edit-level guest cannot write: %d %s", r.Status, r.Body)
	}
	if r := owner.do("DELETE", "/api/shares/"+id, nil); r.Status >= 300 {
		t.Fatalf("unshare: %d %s", r.Status, r.Body)
	}
	if r := guest.do("GET", "/api/pages/"+page+"/markdown", nil); r.Status != http.StatusNotFound {
		t.Fatalf("a revoked share still reads: %d", r.Status)
	}
	if r := guest.do("PUT", "/api/pages/"+page+"/markdown", map[string]string{"markdown": "again", "mode": "replace"}); r.Status < 400 {
		t.Fatalf("a revoked share still writes: %d", r.Status)
	}
}

func TestRevokedTokenStopsWorking(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	made := owner.mustJSON("POST", "/api/me/tokens", map[string]any{"name": "agent", "scope": "write"}, http.StatusCreated)
	tok, id := made["token"].(string), made["id"].(string)
	anon := e.client()
	if r := anon.do("GET", "/api/workspaces/"+ws+"/pages", nil, withBearer(tok)); r.Status != http.StatusOK {
		t.Fatalf("live token: %d %s", r.Status, r.Body)
	}
	if r := owner.do("DELETE", "/api/me/tokens/"+id, nil); r.Status != http.StatusNoContent {
		t.Fatalf("revoke: %d", r.Status)
	}
	if r := anon.do("GET", "/api/workspaces/"+ws+"/pages", nil, withBearer(tok)); r.Status != http.StatusUnauthorized {
		t.Fatalf("revoked token: %d, want 401", r.Status)
	}
	if r := anon.do("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}, withBearer(tok)); r.Status != http.StatusUnauthorized {
		t.Fatalf("revoked token on MCP: %d, want 401", r.Status)
	}
}

func TestSigningOutClosesTheEditingSocket(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	page := owner.createPage(owner.workspaceID(), "Open page")
	p := e.openPeer(owner, page)
	if r := owner.do("POST", "/api/auth/logout", nil, withHeader("Origin", e.ts.URL)); r.Status != http.StatusNoContent {
		t.Fatalf("logout: %d", r.Status)
	}
	p.waitFor("the socket to close", func() bool { return p.closed })
	if r := owner.do("GET", "/api/me", nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("a signed-out session still answers: %d", r.Status)
	}
}

func TestDesktopEntrySignsTheOwnerIn(t *testing.T) {
	e := newEnv(t, desktopKey)
	c := e.client()
	r := c.do("GET", "/desktop/enter?key="+desktopKey, nil)
	if r.Status != http.StatusSeeOther {
		t.Fatalf("enter: %d %s", r.Status, r.Body)
	}
	me := c.mustJSON("GET", "/api/me", nil, http.StatusOK)
	if u, _ := me["user"].(map[string]any); u == nil || u["role"] != "owner" {
		t.Fatalf("the desktop key did not sign in the owner: %v", me)
	}
	again := e.client()
	again.do("GET", "/desktop/enter?key="+desktopKey, nil)
	me2 := again.mustJSON("GET", "/api/me", nil, http.StatusOK)
	if me["user"].(map[string]any)["id"] != me2["user"].(map[string]any)["id"] {
		t.Fatal("a second window entry made a second owner")
	}
	stranger := e.client()
	if r := stranger.do("GET", "/api/me", nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("without the key /api/me answers %d", r.Status)
	}
}
