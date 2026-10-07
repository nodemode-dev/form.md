package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	ringSize  = 2048
	queueSize = 128
	keepalive = 20 * time.Second
)

type Event struct {
	Seq          int64
	Type         string
	Workspace    string
	PrivateOwner string
	Except       string
	User         string
	Data         map[string]any
}

type subscriber struct {
	p      *policy.Principal
	ch     chan Event
	closed bool
}

type Hub struct {
	st      *store.Store
	log     *slog.Logger
	version string

	mu   sync.Mutex
	seq  int64
	ring []Event
	subs map[*subscriber]struct{}
}

func New(st *store.Store, log *slog.Logger, version string) *Hub {
	return &Hub{st: st, log: log, version: version, subs: map[*subscriber]struct{}{}}
}

func (h *Hub) Page(workspaceID, privateOwner string, page any) {
	h.publish(Event{Type: "page", Workspace: workspaceID, PrivateOwner: privateOwner, Data: map[string]any{"workspace_id": workspaceID, "page": page}})
}

func (h *Hub) Removed(workspaceID string, ids []string, except string) {
	if len(ids) == 0 {
		return
	}
	h.publish(Event{Type: "page_removed", Workspace: workspaceID, Except: except, Data: map[string]any{"workspace_id": workspaceID, "ids": ids}})
}

func (h *Hub) Workspace(workspaceID, what string) {
	h.publish(Event{Type: "workspace", Workspace: workspaceID, Data: map[string]any{"workspace_id": workspaceID, "what": what}})
}

func (h *Hub) User(userID, what string) {
	h.publish(Event{Type: "user", User: userID, Data: map[string]any{"what": what}})
}

func (h *Hub) CloseUser(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		if s.p.UserID == userID && !s.closed {
			s.closed = true
			close(s.ch)
		}
	}
}

func (h *Hub) publish(e Event) {
	h.mu.Lock()
	h.seq++
	e.Seq = h.seq
	if len(h.ring) == ringSize {
		copy(h.ring, h.ring[1:])
		h.ring = h.ring[:ringSize-1]
	}
	h.ring = append(h.ring, e)
	targets := make([]*subscriber, 0, len(h.subs))
	for s := range h.subs {
		targets = append(targets, s)
	}
	h.mu.Unlock()
	if len(targets) == 0 {
		return
	}
	allowed := h.audience(e)
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range targets {
		if s.closed || !allowed(s.p) {
			continue
		}
		select {
		case s.ch <- e:
		default:
			s.closed = true
			close(s.ch)
		}
	}
}

func (h *Hub) audience(e Event) func(p *policy.Principal) bool {
	if e.User != "" {
		return func(p *policy.Principal) bool { return p.UserID == e.User }
	}
	if e.Workspace == "" {
		return func(*policy.Principal) bool { return true }
	}
	ctx := context.Background()
	readers := map[string]bool{}
	var agentAccess string
	if err := h.st.R.QueryRowContext(ctx, "SELECT agent_access FROM workspaces WHERE id = ?", e.Workspace).Scan(&agentAccess); err != nil {
		agentAccess = "open"
	}
	rows, err := h.st.R.QueryContext(ctx, `SELECT user_id FROM workspace_members WHERE workspace_id = ?
		UNION SELECT user_id FROM break_glass WHERE workspace_id = ? AND revoked_at = 0 AND expires_at > ?`,
		e.Workspace, e.Workspace, store.NowMS())
	if err != nil {
		h.log.Warn("events: audience", "err", err)
		return func(*policy.Principal) bool { return false }
	}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			readers[id] = true
		}
	}
	rows.Close()
	return func(p *policy.Principal) bool {
		if p.UserID == e.Except || !readers[p.UserID] || !p.Reaches(e.Workspace) || !policy.AgentAllowed(p, agentAccess) {
			return false
		}
		return e.PrivateOwner == "" || e.PrivateOwner == p.UserID
	}
}

func (h *Hub) subscribe(p *policy.Principal, after int64, hasAfter bool) (*subscriber, []Event, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := &subscriber{p: p, ch: make(chan Event, queueSize)}
	h.subs[s] = struct{}{}
	if !hasAfter {
		return s, nil, false
	}
	if after > h.seq {
		return s, nil, true
	}
	if after == h.seq {
		return s, nil, false
	}
	if len(h.ring) == 0 || h.ring[0].Seq > after+1 {
		return s, nil, true
	}
	var replay []Event
	for _, e := range h.ring {
		if e.Seq > after {
			replay = append(replay, e)
		}
	}
	return s, replay, false
}

func (h *Hub) unsubscribe(s *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subs, s)
	if !s.closed {
		s.closed = true
		close(s.ch)
	}
}

func (h *Hub) current() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seq
}

func (h *Hub) Seq() int64 { return h.current() }

func write(w http.ResponseWriter, rc *http.ResponseController, seq int64, data map[string]any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if seq > 0 {
		if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", seq, b); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
		return err
	}
	return rc.Flush()
}

func envelope(e Event) map[string]any {
	out := map[string]any{"type": e.Type, "seq": e.Seq}
	for k, v := range e.Data {
		out[k] = v
	}
	return out
}

type module struct {
	h    *Hub
	auth *auth.Service
}

func (h *Hub) Module(a *auth.Service) app.Module { return module{h, a} }

func (module) ID() string { return "events" }

func (m module) Stop() { m.h.CloseAll() }

func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		if !s.closed {
			s.closed = true
			close(s.ch)
		}
	}
}

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	r.HandleFunc("GET /api/events", m.auth.Require(m.h.Handler))
	return nil
}

func (h *Hub) Handler(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var after int64
	hasAfter := false
	raw := r.Header.Get("Last-Event-ID")
	if raw == "" {
		raw = r.URL.Query().Get("after")
	}
	if raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v < 0 {
			return httpx.Fail(http.StatusBadRequest, "bad_cursor", "the event cursor is not a number")
		}
		after, hasAfter = v, true
	}
	hdr := w.Header()
	hdr.Set("Content-Type", "text/event-stream")
	hdr.Set("Cache-Control", "no-store")
	hdr.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)

	s, replay, reset := h.subscribe(p, after, hasAfter)
	defer h.unsubscribe(s)

	if err := write(w, rc, 0, map[string]any{"type": "hello", "version": h.version, "seq": h.current(), "reset": reset}); err != nil {
		return nil
	}
	for _, e := range replay {
		if h.audience(e)(p) {
			if err := write(w, rc, e.Seq, envelope(e)); err != nil {
				return nil
			}
		}
	}
	tick := time.NewTicker(keepalive)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return nil
		case e, ok := <-s.ch:
			if !ok {
				return nil
			}
			if err := write(w, rc, e.Seq, envelope(e)); err != nil {
				return nil
			}
		case <-tick.C:
			if err := write(w, rc, 0, map[string]any{"type": "ping", "seq": h.current()}); err != nil {
				return nil
			}
		}
	}
}
