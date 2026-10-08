package doc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/reearth/ygo/crdt"
	"github.com/reearth/ygo/encoding"
	ysync "github.com/reearth/ygo/sync"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	kindSync           = 0
	kindAwareness      = 1
	kindControl        = 2
	kindQueryAwareness = 3

	outQueue     = 1024
	maxFrame     = 32 << 20
	pingEvery    = 20 * time.Second
	writeTimeout = 15 * time.Second
	recheckWrite = 3 * time.Second
	maxSubs      = 32

	closeRevoked  websocket.StatusCode = 4001
	closeSlow     websocket.StatusCode = 4002
	closeShutdown websocket.StatusCode = 4003
)

var tintHex = map[string]string{
	"violet": "#a78bfa", "blue": "#60a5fa", "cyan": "#22d3ee", "teal": "#2dd4bf", "green": "#4ade80",
	"lime": "#a3e635", "amber": "#fbbf24", "orange": "#fb923c", "rose": "#fb7185", "pink": "#f472b6",
}

func frame(kind uint64, id string, payload []byte) []byte {
	enc := encoding.NewEncoder()
	enc.WriteVarUint(kind)
	enc.WriteVarString(id)
	enc.WriteRaw(payload)
	return enc.Bytes()
}

func syncFrame(id string, payload []byte) []byte      { return frame(kindSync, id, payload) }
func awarenessFrame(id string, payload []byte) []byte { return frame(kindAwareness, id, payload) }

func controlFrame(v map[string]any) []byte {
	b, _ := json.Marshal(v)
	enc := encoding.NewEncoder()
	enc.WriteVarUint(kindControl)
	enc.WriteVarString(string(b))
	return enc.Bytes()
}

type conn struct {
	s      *Service
	ws     *websocket.Conn
	p      *policy.Principal
	ident  map[string]any
	out    chan []byte
	ctx    context.Context
	cancel context.CancelFunc
	closed atomic.Bool
	once   sync.Once

	mu    sync.Mutex
	rooms map[string]*room
}

func (s *Service) sameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return false
	}
	u, err := url.Parse(o)
	if err != nil {
		return false
	}
	want := httpx.RequestOrigin(r, s.auth.TrustProxy())
	return strings.EqualFold(u.Scheme+"://"+u.Host, want)
}

func (s *Service) serveSync(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	if !s.sameOrigin(r) {
		return httpx.Fail(http.StatusForbidden, "bad_origin", "the sync socket only accepts this site's own pages")
	}
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true, CompressionMode: websocket.CompressionContextTakeover})
	if err != nil {
		return nil
	}
	ws.SetReadLimit(maxFrame)
	ctx, cancel := context.WithCancel(context.Background())
	c := &conn{s: s, ws: ws, p: p, out: make(chan []byte, outQueue), ctx: ctx, cancel: cancel, rooms: map[string]*room{}}
	c.ident = s.identity(ctx, p)
	if !s.register(c) {
		_ = ws.Close(closeShutdown, "the server is stopping")
		cancel()
		return nil
	}
	go c.writer()
	c.reader()
	c.close(websocket.StatusNormalClosure, "")
	return nil
}

func (s *Service) identity(ctx context.Context, p *policy.Principal) map[string]any {
	var color, avatar string
	_ = s.st.R.QueryRowContext(ctx, "SELECT color, avatar FROM users WHERE id = ?", p.UserID).Scan(&color, &avatar)
	hex, ok := tintHex[color]
	if !ok {
		color, hex = "violet", tintHex["violet"]
	}
	id := map[string]any{"id": p.UserID, "name": p.Name, "tint": color, "color": hex}
	if avatar != "" {
		id["avatar"] = avatar
	}
	return id
}

func (c *conn) send(b []byte) {
	if c.closed.Load() {
		return
	}
	select {
	case c.out <- b:
	default:
		go c.close(closeSlow, "the connection fell behind")
	}
}

func (c *conn) close(code websocket.StatusCode, reason string) {
	c.once.Do(func() {
		c.closed.Store(true)
		c.cancel()
		go func() { _ = c.ws.Close(code, reason) }()
		c.mu.Lock()
		rooms := c.rooms
		c.rooms = map[string]*room{}
		c.mu.Unlock()
		for _, r := range rooms {
			r.mu.Lock()
			r.removeSub(c)
			r.mu.Unlock()
		}
		c.s.unregister(c)
	})
}

func (c *conn) writer() {
	t := time.NewTicker(pingEvery)
	defer t.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case b := <-c.out:
			ctx, cancel := context.WithTimeout(c.ctx, writeTimeout)
			err := c.ws.Write(ctx, websocket.MessageBinary, b)
			cancel()
			if err != nil {
				c.close(websocket.StatusGoingAway, "")
				return
			}
		case <-t.C:
			c.send(controlFrame(map[string]any{"t": "ping", "at": store.NowMS()}))
			go c.recheck()
		}
	}
}

func (c *conn) reader() {
	for {
		typ, data, err := c.ws.Read(c.ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageBinary || len(data) == 0 {
			continue
		}
		c.handle(data)
	}
}

type control struct {
	T   string `json:"t"`
	ID  string `json:"id"`
	Gen int64  `json:"gen"`
}

func (c *conn) handle(data []byte) {
	dec := encoding.NewDecoder(data)
	kind, err := dec.ReadVarUint()
	if err != nil {
		return
	}
	if kind == kindControl {
		raw, err := dec.ReadVarString()
		if err != nil {
			return
		}
		var m control
		if json.Unmarshal([]byte(raw), &m) != nil {
			return
		}
		switch m.T {
		case "open":
			c.open(m.ID)
		case "close":
			c.leave(m.ID)
		case "ping":
			c.send(controlFrame(map[string]any{"t": "pong"}))
		}
		return
	}
	id, err := dec.ReadVarString()
	if err != nil {
		return
	}
	payload := dec.RemainingBytes()
	c.mu.Lock()
	r := c.rooms[id]
	c.mu.Unlock()
	if r == nil {
		return
	}
	switch kind {
	case kindSync:
		c.syncMessage(r, payload)
	case kindAwareness:
		r.mu.Lock()
		if sb := r.subs[c]; sb != nil {
			r.applyAwareness(c, sb, payload)
		}
		r.mu.Unlock()
	case kindQueryAwareness:
		r.mu.Lock()
		if snap := r.awarenessSnapshot(); snap != nil {
			c.send(snap)
		}
		r.mu.Unlock()
	}
}

func refusal(err error) string {
	var api *httpx.APIError
	if errors.As(err, &api) {
		return api.Code
	}
	if hidden(err) {
		return "not_found"
	}
	return "error"
}

func (c *conn) open(id string) {
	if id == "" {
		return
	}
	c.mu.Lock()
	existing := c.rooms[id]
	n := len(c.rooms)
	c.mu.Unlock()
	if existing == nil && n >= maxSubs {
		c.send(controlFrame(map[string]any{"t": "denied", "id": id, "code": "too_many"}))
		return
	}
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()
	if err := c.s.pages.CanRead(ctx, c.p, id); err != nil {
		c.send(controlFrame(map[string]any{"t": "denied", "id": id, "code": refusal(err)}))
		return
	}
	_, werr := c.s.pages.CanWrite(ctx, c.p, id)
	write := werr == nil
	r := existing
	if r == nil {
		var err error
		r, err = c.s.acquire(ctx, id)
		if err != nil {
			c.s.log.Warn("doc: load", "page", id, "err", err)
			c.send(controlFrame(map[string]any{"t": "denied", "id": id, "code": "error"}))
			return
		}
	}
	r.mu.Lock()
	sb := r.subs[c]
	if sb == nil {
		sb = &sub{c: c, clients: map[uint64]bool{}}
		r.subs[c] = sb
	}
	sb.write, sb.checked, sb.refused = write, time.Now(), false
	c.send(controlFrame(map[string]any{"t": "opened", "id": id, "gen": r.gen, "write": write}))
	c.send(r.step1Frame())
	if snap := r.awarenessSnapshot(); snap != nil {
		c.send(snap)
	}
	r.mu.Unlock()
	if existing == nil {
		c.mu.Lock()
		if c.closed.Load() {
			c.mu.Unlock()
			r.mu.Lock()
			r.removeSub(c)
			r.mu.Unlock()
			return
		}
		c.rooms[id] = r
		c.mu.Unlock()
	}
}

func (c *conn) leave(id string) {
	c.mu.Lock()
	r := c.rooms[id]
	delete(c.rooms, id)
	c.mu.Unlock()
	if r == nil {
		return
	}
	r.mu.Lock()
	r.removeSub(c)
	r.mu.Unlock()
}

func (c *conn) mayWrite(r *room) bool {
	r.mu.Lock()
	sb := r.subs[c]
	if sb == nil {
		r.mu.Unlock()
		return false
	}
	fresh := time.Since(sb.checked) < recheckWrite
	ok := sb.write
	r.mu.Unlock()
	if fresh {
		return ok
	}
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	_, err := c.s.pages.CanWrite(ctx, c.p, r.id)
	cancel()
	r.mu.Lock()
	if sb := r.subs[c]; sb != nil {
		sb.write, sb.checked = err == nil, time.Now()
	}
	r.mu.Unlock()
	return err == nil
}

func (c *conn) refuse(r *room, code string) {
	r.mu.Lock()
	sb := r.subs[c]
	first := sb != nil && !sb.refused
	if sb != nil {
		sb.refused = true
	}
	r.mu.Unlock()
	if first {
		c.send(controlFrame(map[string]any{"t": "refused", "id": r.id, "code": code}))
	}
}

func (c *conn) syncMessage(r *room, payload []byte) {
	t, body, err := ysync.ReadSyncMessage(payload)
	if err != nil {
		return
	}
	switch t {
	case ysync.MsgSyncStep1:
		r.mu.Lock()
		reply, err := ysync.EncodeSyncStep2(r.doc, payload)
		if err == nil {
			c.send(syncFrame(r.id, reply))
		}
		r.mu.Unlock()
	case ysync.MsgSyncStep2, ysync.MsgUpdate:
		if len(body) <= 2 {
			return
		}
		if !c.mayWrite(r) {
			c.refuse(r, "forbidden")
			return
		}
		r.mu.Lock()
		if r.size+len(body) > maxDocBytes {
			r.mu.Unlock()
			c.refuse(r, "doc_too_large")
			return
		}
		err := crdt.ApplyUpdateV1(r.doc, body, &origin{c: c, user: c.p.UserID})
		r.mu.Unlock()
		if err != nil {
			c.s.log.Warn("doc: rejected an update", "page", r.id, "user", c.p.UserID, "err", err)
			c.send(controlFrame(map[string]any{"t": "error", "id": r.id, "code": "bad_update"}))
		}
	}
}

func (c *conn) recheck() {
	c.mu.Lock()
	ids := make([]string, 0, len(c.rooms))
	for id := range c.rooms {
		ids = append(ids, id)
	}
	c.mu.Unlock()
	for _, id := range ids {
		ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		err := c.s.pages.CanRead(ctx, c.p, id)
		cancel()
		if err != nil && hidden(err) {
			c.leave(id)
			c.send(controlFrame(map[string]any{"t": "closed", "id": id, "code": "not_found"}))
		}
	}
}
