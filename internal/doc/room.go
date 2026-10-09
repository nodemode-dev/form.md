package doc

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/reearth/ygo/crdt"
	"github.com/reearth/ygo/encoding"
	ysync "github.com/reearth/ygo/sync"

	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	flushDelay      = 300 * time.Millisecond
	settleDelay     = 1500 * time.Millisecond
	settleMax       = 10 * time.Second
	idleUnload      = 60 * time.Second
	compactUpdates  = 400
	compactBytes    = 1 << 20
	maxDocBytes     = 64 << 20
	maxAwareState   = 8 << 10
	maxAwareClients = 64
)

type origin struct {
	c    *conn
	user string
}

type sub struct {
	c       *conn
	write   bool
	checked time.Time
	refused bool
	clients map[uint64]bool
}

type awareEntry struct {
	clock uint64
	state []byte
	owner *conn
}

type pendingUpdate struct {
	data []byte
	user string
	at   int64
}

type room struct {
	s  *Service
	id string

	mu    sync.Mutex
	doc   *crdt.Doc
	gen   int64
	subs  map[*conn]*sub
	aware map[uint64]*awareEntry
	refs  int
	size  int

	pending    []pendingUpdate
	nextSeq    int64
	sinceSnap  int
	bytesSince int
	flushTimer *time.Timer
	persistMu  sync.Mutex

	dirty      bool
	dirtySince time.Time
	lastEditor string
	matTimer   *time.Timer
	idleTimer  *time.Timer

	editors             map[string]bool
	changedSinceVersion bool
	lastVersion         int64
	lastVersionID       string
	expectDS            []byte
}

func deleteSetBytes(doc *crdt.Doc) []byte {
	sv, err := crdt.DecodeStateVectorV1(crdt.EncodeStateVectorV1(doc))
	if err != nil {
		return nil
	}
	u := crdt.EncodeStateAsUpdateV1(doc, sv)
	if len(u) == 0 || u[0] != 0 {
		return nil
	}
	return u[1:]
}

func (s *Service) loadRoom(ctx context.Context, id string) (*room, error) {
	r := &room{s: s, id: id, doc: crdt.New(), gen: 1, nextSeq: 1, subs: map[*conn]*sub{}, aware: map[uint64]*awareEntry{}, editors: map[string]bool{}, lastVersion: s.lastVersionAt(ctx, id)}
	var snap []byte
	var snapSeq int64
	err := s.st.R.QueryRowContext(ctx, "SELECT generation, snapshot, snapshot_seq, next_seq FROM doc_state WHERE page_id = ?", id).Scan(&r.gen, &snap, &snapSeq, &r.nextSeq)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if len(snap) > 0 {
		if err := crdt.ApplyUpdateV1(r.doc, snap, nil); err != nil {
			return nil, err
		}
		r.size = len(snap)
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT seq, data FROM doc_updates WHERE page_id = ? AND seq > ? ORDER BY seq", id, snapSeq)
	if err != nil {
		return nil, err
	}
	var updates [][]byte
	for rows.Next() {
		var seq int64
		var data []byte
		if err := rows.Scan(&seq, &data); err != nil {
			rows.Close()
			return nil, err
		}
		updates = append(updates, data)
		if seq >= r.nextSeq {
			r.nextSeq = seq + 1
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, u := range updates {
		if err := crdt.ApplyUpdateV1(r.doc, u, nil); err != nil {
			s.log.Warn("doc: skipping a stored update that does not apply", "page", id, "err", err)
			continue
		}
		r.size += len(u)
		r.bytesSince += len(u)
	}
	r.sinceSnap = len(updates)
	frag := r.doc.GetXmlFragment(Root)
	r.doc.OnUpdate(r.onUpdate)
	if frag.Len() == 0 {
		r.mu.Lock()
		r.doc.Transact(func(txn *crdt.Transaction) {
			writeNodes(txn, frag, 0, []*pmNode{newNode("paragraph")})
		}, &origin{})
		r.dirty = false
		r.changedSinceVersion = false
		if r.matTimer != nil {
			r.matTimer.Stop()
			r.matTimer = nil
		}
		r.mu.Unlock()
	}
	return r, nil
}

func (r *room) onUpdate(update []byte, o any) {
	if len(update) <= 2 {
		return
	}
	if r.expectDS != nil && update[0] == 0 && bytes.Equal(update[1:], r.expectDS) {
		return
	}
	or, _ := o.(*origin)
	frame := syncFrame(r.id, ysync.EncodeUpdate(update))
	for c := range r.subs {
		if or != nil && or.c == c {
			continue
		}
		c.send(frame)
	}
	user := ""
	if or != nil {
		user = or.user
	}
	r.pending = append(r.pending, pendingUpdate{data: update, user: user, at: store.NowMS()})
	r.changedSinceVersion = true
	if user != "" {
		r.editors[user] = true
	}
	r.bytesSince += len(update)
	r.size += len(update)
	if r.flushTimer == nil {
		r.flushTimer = time.AfterFunc(flushDelay, func() {
			r.mu.Lock()
			r.flushTimer = nil
			r.mu.Unlock()
			if err := r.flush(context.Background()); err != nil {
				r.s.log.Warn("doc: flush", "page", r.id, "err", err)
			}
		})
	}
	r.markDirty(user)
}

func (r *room) markDirty(user string) {
	now := time.Now()
	r.dirty = true
	if user != "" {
		r.lastEditor = user
	}
	if r.dirtySince.IsZero() {
		r.dirtySince = now
	}
	delay := settleDelay
	if now.Sub(r.dirtySince) >= settleMax {
		delay = 0
	}
	if r.matTimer != nil {
		r.matTimer.Stop()
	}
	r.matTimer = time.AfterFunc(delay, func() { r.s.materialise(context.Background(), r) })
}

func (r *room) flush(ctx context.Context) error {
	r.persistMu.Lock()
	defer r.persistMu.Unlock()
	r.mu.Lock()
	batch := r.pending
	r.pending = nil
	if len(batch) == 0 {
		r.mu.Unlock()
		return nil
	}
	first := r.nextSeq
	r.nextSeq += int64(len(batch))
	last := r.nextSeq - 1
	var snap []byte
	if r.sinceSnap+len(batch) >= compactUpdates || r.bytesSince >= compactBytes {
		snap = crdt.EncodeStateAsUpdateV1(r.doc, nil)
		r.sinceSnap, r.bytesSince, r.size = 0, 0, len(snap)
	} else {
		r.sinceSnap += len(batch)
	}
	gen := r.gen
	r.mu.Unlock()
	now := store.NowMS()
	err := r.s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO doc_state (page_id, generation, snapshot, snapshot_seq, next_seq, updated_at) VALUES (?, ?, X'', 0, ?, ?)
			ON CONFLICT (page_id) DO UPDATE SET next_seq = excluded.next_seq, updated_at = excluded.updated_at`, r.id, gen, last+1, now); err != nil {
			return err
		}
		for i, u := range batch {
			if _, err := tx.ExecContext(ctx, "INSERT INTO doc_updates (page_id, seq, data, user_id, created_at) VALUES (?, ?, ?, ?, ?)", r.id, first+int64(i), u.data, u.user, u.at); err != nil {
				return err
			}
		}
		if snap != nil {
			if _, err := tx.ExecContext(ctx, "UPDATE doc_state SET snapshot = ?, snapshot_seq = ? WHERE page_id = ?", snap, last, r.id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM doc_updates WHERE page_id = ? AND seq <= ?", r.id, last); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			r.s.log.Info("doc: page is gone, dropping its unsaved changes", "page", r.id)
			return nil
		}
		r.mu.Lock()
		r.pending = append(batch, r.pending...)
		r.nextSeq = first
		r.mu.Unlock()
		return err
	}
	return nil
}

func (r *room) fragment() *crdt.YXmlFragment {
	return r.doc.GetXmlFragment(Root)
}

func (r *room) step1Frame() []byte {
	return syncFrame(r.id, ysync.EncodeSyncStep1(r.doc))
}

type awareUpdate struct {
	id    uint64
	clock uint64
	state []byte
}

func decodeAwareness(b []byte) ([]awareUpdate, error) {
	dec := encoding.NewDecoder(b)
	n, err := dec.ReadVarUint()
	if err != nil {
		return nil, err
	}
	if n > maxAwareClients {
		return nil, errors.New("too many awareness entries")
	}
	out := make([]awareUpdate, 0, n)
	for i := uint64(0); i < n; i++ {
		id, err := dec.ReadVarUint()
		if err != nil {
			return nil, err
		}
		clock, err := dec.ReadVarUint()
		if err != nil {
			return nil, err
		}
		state, err := dec.ReadVarString()
		if err != nil {
			return nil, err
		}
		out = append(out, awareUpdate{id: id, clock: clock, state: []byte(state)})
	}
	return out, nil
}

func encodeAwareness(list []awareUpdate) []byte {
	enc := encoding.NewEncoder()
	enc.WriteVarUint(uint64(len(list)))
	for _, u := range list {
		enc.WriteVarUint(u.id)
		enc.WriteVarUint(u.clock)
		enc.WriteVarString(string(u.state))
	}
	return enc.Bytes()
}

func (r *room) applyAwareness(c *conn, sb *sub, payload []byte) {
	list, err := decodeAwareness(payload)
	if err != nil {
		return
	}
	var out []awareUpdate
	for _, u := range list {
		e := r.aware[u.id]
		if e != nil && e.owner != c {
			continue
		}
		if e != nil && u.clock < e.clock {
			continue
		}
		if string(u.state) == "null" || len(u.state) == 0 {
			if e == nil {
				continue
			}
			delete(r.aware, u.id)
			delete(sb.clients, u.id)
			out = append(out, awareUpdate{id: u.id, clock: u.clock, state: []byte("null")})
			continue
		}
		if len(u.state) > maxAwareState || (e == nil && len(r.aware) >= maxAwareClients) {
			continue
		}
		var state map[string]any
		if json.Unmarshal(u.state, &state) != nil || state == nil {
			continue
		}
		state["user"] = c.ident
		stamped, err := json.Marshal(state)
		if err != nil {
			continue
		}
		r.aware[u.id] = &awareEntry{clock: u.clock, state: stamped, owner: c}
		sb.clients[u.id] = true
		out = append(out, awareUpdate{id: u.id, clock: u.clock, state: stamped})
	}
	if len(out) == 0 {
		return
	}
	frame := awarenessFrame(r.id, encodeAwareness(out))
	for other := range r.subs {
		if other != c {
			other.send(frame)
		}
	}
}

func (r *room) awarenessSnapshot() []byte {
	var list []awareUpdate
	for id, e := range r.aware {
		list = append(list, awareUpdate{id: id, clock: e.clock, state: e.state})
	}
	if len(list) == 0 {
		return nil
	}
	return awarenessFrame(r.id, encodeAwareness(list))
}

func (r *room) removeSub(c *conn) {
	sb, ok := r.subs[c]
	if !ok {
		return
	}
	delete(r.subs, c)
	r.refs--
	var out []awareUpdate
	for id := range sb.clients {
		e := r.aware[id]
		if e == nil {
			continue
		}
		delete(r.aware, id)
		out = append(out, awareUpdate{id: id, clock: e.clock + 1, state: []byte("null")})
	}
	if len(out) > 0 {
		frame := awarenessFrame(r.id, encodeAwareness(out))
		for other := range r.subs {
			other.send(frame)
		}
	}
	r.scheduleIdle()
}

func (r *room) scheduleIdle() {
	if r.refs > 0 {
		return
	}
	if r.idleTimer != nil {
		r.idleTimer.Stop()
	}
	r.idleTimer = time.AfterFunc(idleUnload, func() { r.s.unload(r) })
}
