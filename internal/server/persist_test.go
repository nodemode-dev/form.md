package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEditsSurviveARestart(t *testing.T) {
	dir := t.TempDir()
	first := newEnvAt(t, "", dir)
	owner := first.setupOwner()
	page := owner.createPage(owner.workspaceID(), "Kept")
	owner.putMarkdown(page, "before the restart")
	token := owner.apiToken("write")

	p := first.openPeer(owner, page)
	p.waitFor("content", func() bool { return strings.Contains(p.xml(), "before the restart") })
	p.typeAt(0, "typed live ")
	mcp(t, first, token, "insert_blocks", map[string]any{"page_id": page, "markdown": "added by an agent"})
	for i := 0; i < 400; i++ {
		md := serverMarkdown(owner, page)
		if strings.Contains(md, "typed live") && strings.Contains(md, "added by an agent") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	first.stop()

	second := newEnvAt(t, "", dir)
	again := second.client()
	again.mustJSON("POST", "/api/auth/login", map[string]string{"email": "owner@example.test", "password": "correct-horse-1"}, http.StatusOK)
	md := string(again.do("GET", "/api/pages/"+page+"/markdown", nil).Body)
	for _, want := range []string{"typed live before the restart", "added by an agent"} {
		if !strings.Contains(md, want) {
			t.Fatalf("the restarted server lost %q: %q", want, md)
		}
	}
}
