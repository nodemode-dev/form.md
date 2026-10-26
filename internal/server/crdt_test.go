package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/reearth/ygo/crdt"
	"github.com/reearth/ygo/encoding"
	ysync "github.com/reearth/ygo/sync"
)

const root = "prosemirror"

type peer struct {
	t        *testing.T
	conn     *websocket.Conn
	doc      *crdt.Doc
	page     string
	out      chan []byte
	cancel   context.CancelFunc
	dmu      sync.Mutex
	mu       sync.Mutex
	write    *bool
	canWrite bool
	refused  bool
	closed   bool
	denied   string
}

func frameOf(kind uint64, id string, payload []byte) []byte {
	enc := encoding.NewEncoder()
	enc.WriteVarUint(kind)
	enc.WriteVarString(id)
	enc.WriteRaw(payload)
	return enc.Bytes()
}

func (e *env) openPeer(c *client, page string) *peer {
	e.t.Helper()
	conn, resp, err := dialSync(e, c, e.ts.URL)
	if err != nil {
		e.t.Fatalf("sync dial: %v (%v)", err, resp)
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &peer{t: e.t, conn: conn, doc: crdt.New(), page: page, out: make(chan []byte, 256), cancel: cancel}
	e.t.Cleanup(func() { cancel(); conn.CloseNow() })
	p.doc.OnUpdate(func(update []byte, origin any) {
		if origin == "remote" || len(update) <= 2 {
			return
		}
		p.out <- frameOf(0, page, ysync.EncodeUpdate(update))
	})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case b := <-p.out:
				if conn.Write(ctx, websocket.MessageBinary, b) != nil {
					return
				}
			}
		}
	}()
	go p.read(ctx)
	ctl, _ := json.Marshal(map[string]string{"t": "open", "id": page})
	enc := encoding.NewEncoder()
	enc.WriteVarUint(2)
	enc.WriteVarString(string(ctl))
	p.out <- enc.Bytes()
	p.waitFor("open reply", func() bool { return p.write != nil || p.denied != "" })
	p.mu.Lock()
	p.canWrite = p.write != nil && *p.write
	denied := p.denied
	p.mu.Unlock()
	if denied != "" {
		e.t.Fatalf("page %s denied: %s", page, p.denied)
	}
	p.out <- frameOf(0, page, ysync.EncodeSyncStep1(p.doc))
	return p
}

func (p *peer) read(ctx context.Context) {
	for {
		_, data, err := p.conn.Read(ctx)
		if err != nil {
			p.mu.Lock()
			p.closed = true
			p.mu.Unlock()
			return
		}
		dec := encoding.NewDecoder(data)
		kind, err := dec.ReadVarUint()
		if err != nil {
			continue
		}
		if kind == 2 {
			raw, _ := dec.ReadVarString()
			var m struct {
				T     string `json:"t"`
				Write bool   `json:"write"`
				Code  string `json:"code"`
			}
			if json.Unmarshal([]byte(raw), &m) != nil {
				continue
			}
			p.mu.Lock()
			switch m.T {
			case "opened":
				w := m.Write
				p.write = &w
			case "denied":
				p.denied = m.Code
			case "refused":
				p.refused = true
			}
			p.mu.Unlock()
			continue
		}
		if _, err := dec.ReadVarString(); err != nil || kind != 0 {
			continue
		}
		p.dmu.Lock()
		reply, err := ysync.ApplySyncMessage(p.doc, dec.RemainingBytes(), "remote")
		p.dmu.Unlock()
		if err == nil && reply != nil {
			p.out <- frameOf(0, p.page, reply)
		}
	}
}

func (p *peer) waitFor(what string, cond func() bool) {
	p.t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		ok := cond()
		p.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	p.t.Fatalf("timed out waiting for %s", what)
}

func (p *peer) xml() string {
	p.dmu.Lock()
	defer p.dmu.Unlock()
	return p.doc.GetXmlFragment(root).ToXML()
}

func (p *peer) firstText() *crdt.YXmlText {
	p.t.Helper()
	for _, n := range p.doc.GetXmlFragment(root).Children() {
		if el, ok := n.(*crdt.YXmlElement); ok {
			for _, c := range el.Children() {
				if tx, ok := c.(*crdt.YXmlText); ok {
					return tx
				}
			}
		}
	}
	p.t.Fatalf("no text node in %s", p.doc.GetXmlFragment(root).ToXML())
	return nil
}

func (p *peer) typeAt(index int, s string) {
	p.dmu.Lock()
	defer p.dmu.Unlock()
	tx := p.firstText()
	p.doc.Transact(func(txn *crdt.Transaction) { tx.Insert(txn, index, s, nil) })
}

func mcp(t *testing.T, e *env, token, tool string, args map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args}}
	r := e.client().do("POST", "/mcp", body, withBearer(token))
	if r.Status != http.StatusOK {
		t.Fatalf("mcp %s: status %d: %s", tool, r.Status, r.Body)
	}
	out := r.JSON(t)
	res, _ := out["result"].(map[string]any)
	if res == nil {
		t.Fatalf("mcp %s: %s", tool, r.Body)
	}
	if bad, _ := res["isError"].(bool); bad {
		t.Fatalf("mcp %s failed: %s", tool, r.Body)
	}
	return res
}

func serverMarkdown(owner *client, page string) string {
	return string(owner.do("GET", "/api/pages/"+page+"/markdown", nil).Body)
}

func TestTwoClientsAndAnAgentConvergeOnTheSameParagraph(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Shared")
	owner.putMarkdown(page, "hello world\n\nsecond block")
	token := owner.apiToken("write")

	a := e.openPeer(owner, page)
	b := e.openPeer(owner, page)
	for _, p := range []*peer{a, b} {
		p.waitFor("the page content", func() bool { return strings.Contains(p.xml(), "hello world") })
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); a.typeAt(5, "AAA") }()
	go func() { defer wg.Done(); b.typeAt(5, "BBB") }()
	go func() {
		defer wg.Done()
		mcp(t, e, token, "insert_blocks", map[string]any{"page_id": page, "markdown": "agent paragraph"})
	}()
	wg.Wait()

	for _, p := range []*peer{a, b} {
		p.waitFor("all three edits", func() bool {
			x := p.xml()
			return strings.Contains(x, "AAA") && strings.Contains(x, "BBB") && strings.Contains(x, "agent paragraph")
		})
	}
	var settled bool
	for i := 0; i < 400 && !settled; i++ {
		settled = a.xml() == b.xml()
		time.Sleep(10 * time.Millisecond)
	}
	if !settled {
		t.Fatalf("the two clients diverged:\nA: %s\nB: %s", a.xml(), b.xml())
	}
	md := ""
	for i := 0; i < 400; i++ {
		md = serverMarkdown(owner, page)
		if strings.Contains(md, "AAA") && strings.Contains(md, "BBB") && strings.Contains(md, "agent paragraph") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"AAA", "BBB", "world", "second block", "agent paragraph"} {
		if !strings.Contains(md, want) {
			t.Fatalf("the server lost %q: %q", want, md)
		}
	}
	first := strings.SplitN(md, "\n", 2)[0]
	if !strings.HasPrefix(first, "hello") || !strings.HasSuffix(first, "world") || strings.Count(first, "AAA") != 1 || strings.Count(first, "BBB") != 1 {
		t.Fatalf("the shared paragraph did not merge both inserts once each: %q", first)
	}
	for _, p := range []*peer{a, b} {
		x := p.xml()
		if !strings.Contains(x, first[:5]) || strings.Count(x, "AAA") != 1 || strings.Count(x, "BBB") != 1 {
			t.Fatalf("a client differs from the server: %s vs %q", x, md)
		}
	}
}

func TestAgentEditKeepsWhatPeopleTypedElsewhere(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Mixed")
	owner.putMarkdown(page, "typed here\n\nagent rewrites this")
	token := owner.apiToken("write")
	a := e.openPeer(owner, page)
	a.waitFor("content", func() bool { return strings.Contains(a.xml(), "agent rewrites this") })

	res := mcp(t, e, token, "get_page", map[string]any{"page_id": page})
	raw, _ := json.Marshal(res)
	var blocks []struct {
		ID       string `json:"id"`
		Markdown string `json:"markdown"`
	}
	var found func(v any)
	found = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if id, ok := x["id"].(string); ok {
				if md, ok := x["markdown"].(string); ok {
					blocks = append(blocks, struct {
						ID       string `json:"id"`
						Markdown string `json:"markdown"`
					}{id, md})
				}
			}
			for _, c := range x {
				found(c)
			}
		case []any:
			for _, c := range x {
				found(c)
			}
		case string:
			var inner any
			if strings.HasPrefix(x, "{") && json.Unmarshal([]byte(x), &inner) == nil {
				found(inner)
			}
		}
	}
	var top any
	_ = json.Unmarshal(raw, &top)
	found(top)
	target := ""
	for _, b := range blocks {
		if strings.Contains(b.Markdown, "agent rewrites this") {
			target = b.ID
		}
	}
	if target == "" {
		t.Fatalf("no block id for the second paragraph in %s", raw)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a.typeAt(0, "LIVE ") }()
	go func() {
		defer wg.Done()
		mcp(t, e, token, "replace_block", map[string]any{"page_id": page, "block_id": target, "markdown": "rewritten by the agent"})
	}()
	wg.Wait()

	a.waitFor("both edits", func() bool {
		x := a.xml()
		return strings.Contains(x, "LIVE typed here") && strings.Contains(x, "rewritten by the agent") && !strings.Contains(x, "agent rewrites this")
	})
	var md string
	for i := 0; i < 400; i++ {
		md = serverMarkdown(owner, page)
		if strings.Contains(md, "LIVE typed here") && strings.Contains(md, "rewritten by the agent") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(md, "LIVE typed here") || !strings.Contains(md, "rewritten by the agent") || strings.Contains(md, "agent rewrites this") {
		t.Fatalf("server state is wrong: %q", md)
	}
}

func TestViewerSocketCannotEditThePage(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.createPage(ws, "Locked")
	owner.putMarkdown(page, "original text")
	viewer := owner.joinAs(ws, "viewer@example.test", "viewer")

	v := e.openPeer(viewer, page)
	if v.canWrite {
		t.Fatal("the server told a viewer it may write")
	}
	v.waitFor("content", func() bool { return strings.Contains(v.xml(), "original text") })
	v.typeAt(0, "VANDAL ")
	v.waitFor("a refusal", func() bool { return v.refused })
	time.Sleep(200 * time.Millisecond)
	if md := serverMarkdown(owner, page); strings.Contains(md, "VANDAL") || !strings.Contains(md, "original text") {
		t.Fatalf("a viewer's socket edit reached the page: %q", md)
	}
	w := e.openPeer(owner, page)
	if !w.canWrite {
		t.Fatal("the owner was told it cannot write")
	}
	w.waitFor("content", func() bool { return strings.Contains(w.xml(), "original text") })
	if strings.Contains(w.xml(), "VANDAL") {
		t.Fatal("another client received the viewer's edit")
	}
}
