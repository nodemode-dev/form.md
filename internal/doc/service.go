package doc

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/reearth/ygo/crdt"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Service struct {
	st    *store.Store
	pages *page.Service
	auth  *auth.Service
	log   *slog.Logger

	mu       sync.Mutex
	rooms    map[string]*room
	conns    map[*conn]struct{}
	stopped  bool
	refHooks []func(ctx context.Context, pageID, actor string, mentioned []string, reminders []Reminder)
	audit    *audit.Log

	remote RemoteImporter
}

func New(st *store.Store, pages *page.Service, a *auth.Service, log *slog.Logger) *Service {
	return &Service{st: st, pages: pages, auth: a, log: log, rooms: map[string]*room{}, conns: map[*conn]struct{}{}}
}

func hidden(err error) bool {
	return errors.Is(err, policy.ErrHidden) || errors.Is(err, store.ErrNotFound)
}

func (s *Service) register(c *conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return false
	}
	s.conns[c] = struct{}{}
	return true
}

func (s *Service) unregister(c *conn) {
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
}

func (s *Service) acquire(ctx context.Context, id string) (*room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.rooms[id]
	if r == nil {
		var err error
		if r, err = s.loadRoom(ctx, id); err != nil {
			return nil, err
		}
		s.rooms[id] = r
	}
	r.mu.Lock()
	r.refs++
	if r.idleTimer != nil {
		r.idleTimer.Stop()
		r.idleTimer = nil
	}
	r.mu.Unlock()
	return r, nil
}

func (s *Service) release(r *room) {
	r.mu.Lock()
	r.refs--
	r.scheduleIdle()
	r.mu.Unlock()
}

func (s *Service) unload(r *room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.mu.Lock()
	busy := r.refs > 0
	r.mu.Unlock()
	if busy || s.rooms[r.id] != r {
		return
	}
	s.materialise(context.Background(), r)
	if _, err := s.saveVersion(context.Background(), r, "auto", true); err != nil {
		s.log.Warn("doc: version", "page", r.id, "err", err)
	}
	r.mu.Lock()
	if r.refs > 0 {
		r.mu.Unlock()
		return
	}
	if r.matTimer != nil {
		r.matTimer.Stop()
	}
	r.mu.Unlock()
	delete(s.rooms, r.id)
	r.doc.Destroy()
}

func (s *Service) materialise(ctx context.Context, r *room) {
	if err := r.flush(ctx); err != nil {
		s.log.Warn("doc: flush", "page", r.id, "err", err)
		return
	}
	r.mu.Lock()
	if !r.dirty {
		r.mu.Unlock()
		return
	}
	r.dirty = false
	r.dirtySince = time.Time{}
	sum := r.summary(nil)
	user := r.lastEditor
	r.mu.Unlock()
	now := store.NowMS()
	var added []string
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO doc_state (page_id, snapshot, markdown, text_size, materialised_at, updated_at) VALUES (?, X'', ?, ?, ?, ?)
			ON CONFLICT (page_id) DO UPDATE SET markdown = excluded.markdown, text_size = excluded.text_size, materialised_at = excluded.materialised_at`,
			r.id, sum.Markdown, len(sum.Text), now, now); err != nil {
			return err
		}
		before, err := mentionedUsers(ctx, tx, r.id)
		if err != nil {
			return err
		}
		added = added[:0]
		for _, u := range sum.Users {
			if !before[u] {
				added = append(added, u)
			}
		}
		return writeRefs(ctx, tx, r.id, sum)
	})
	if err != nil {
		if !strings.Contains(err.Error(), "FOREIGN KEY") {
			s.log.Warn("doc: materialise", "page", r.id, "err", err)
		}
		return
	}
	if err := s.pages.SetContent(ctx, r.id, sum.Snippet, sum.Thumb, user); err != nil && !hidden(err) {
		s.log.Warn("doc: page content", "page", r.id, "err", err)
	}
	for _, fn := range s.refHooks {
		fn(ctx, r.id, user, added, sum.Reminders)
	}
	if _, err := s.saveVersion(ctx, r, "auto", false); err != nil {
		s.log.Warn("doc: version", "page", r.id, "err", err)
	}
}

func (s *Service) OnRefs(fn func(ctx context.Context, pageID, actor string, mentioned []string, reminders []Reminder)) {
	s.refHooks = append(s.refHooks, fn)
}

func mentionedUsers(ctx context.Context, tx *sql.Tx, id string) (map[string]bool, error) {
	rows, err := tx.QueryContext(ctx, "SELECT target FROM doc_refs WHERE page_id = ? AND kind = 'user'", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out[t] = true
	}
	return out, rows.Err()
}

func writeRefs(ctx context.Context, tx *sql.Tx, id string, sum Summary) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM doc_refs WHERE page_id = ?", id); err != nil {
		return err
	}
	add := func(kind string, targets []string) error {
		for _, t := range targets {
			if t == "" || (kind == "page" && t == id) {
				continue
			}
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO doc_refs (page_id, kind, target) VALUES (?, ?, ?)", id, kind, t); err != nil {
				return err
			}
		}
		return nil
	}
	if err := add("page", sum.Pages); err != nil {
		return err
	}
	if err := add("file", sum.Files); err != nil {
		return err
	}
	return add("user", sum.Users)
}

func (s *Service) CloseUser(userID string) {
	s.mu.Lock()
	var list []*conn
	for c := range s.conns {
		if c.p.UserID == userID {
			list = append(list, c)
		}
	}
	s.mu.Unlock()
	for _, c := range list {
		c.close(closeRevoked, "signed out")
	}
}

func (s *Service) CloseCredential(id string) {
	s.mu.Lock()
	var list []*conn
	for c := range s.conns {
		if c.p.CredentialID == id {
			list = append(list, c)
		}
	}
	s.mu.Unlock()
	for _, c := range list {
		c.close(closeRevoked, "signed out")
	}
}

func (s *Service) Stop() {
	s.mu.Lock()
	s.stopped = true
	list := make([]*conn, 0, len(s.conns))
	for c := range s.conns {
		list = append(list, c)
	}
	s.mu.Unlock()
	for _, c := range list {
		c.close(closeShutdown, "the server is restarting")
	}
}

func (s *Service) Drain(ctx context.Context) {
	s.mu.Lock()
	rooms := make([]*room, 0, len(s.rooms))
	for _, r := range s.rooms {
		rooms = append(rooms, r)
	}
	s.mu.Unlock()
	for _, r := range rooms {
		r.mu.Lock()
		if r.flushTimer != nil {
			r.flushTimer.Stop()
			r.flushTimer = nil
		}
		if r.matTimer != nil {
			r.matTimer.Stop()
		}
		r.mu.Unlock()
		s.materialise(ctx, r)
	}
}

func (s *Service) Copy(ctx context.Context, tx *sql.Tx, from, to string) error {
	s.mu.Lock()
	r := s.rooms[from]
	s.mu.Unlock()
	var state []byte
	if r != nil {
		r.mu.Lock()
		state = crdt.EncodeStateAsUpdateV1(r.doc, nil)
		r.mu.Unlock()
	} else {
		d := crdt.New()
		var snap []byte
		var snapSeq int64
		err := tx.QueryRowContext(ctx, "SELECT snapshot, snapshot_seq FROM doc_state WHERE page_id = ?", from).Scan(&snap, &snapSeq)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(snap) > 0 {
			if err := crdt.ApplyUpdateV1(d, snap, nil); err != nil {
				return err
			}
		}
		rows, err := tx.QueryContext(ctx, "SELECT data FROM doc_updates WHERE page_id = ? AND seq > ? ORDER BY seq", from, snapSeq)
		if err != nil {
			return err
		}
		var updates [][]byte
		for rows.Next() {
			var u []byte
			if err := rows.Scan(&u); err != nil {
				rows.Close()
				return err
			}
			updates = append(updates, u)
		}
		rows.Close()
		for _, u := range updates {
			_ = crdt.ApplyUpdateV1(d, u, nil)
		}
		state = crdt.EncodeStateAsUpdateV1(d, nil)
		d.Destroy()
	}
	now := store.NowMS()
	res, err := tx.ExecContext(ctx, `INSERT INTO doc_state (page_id, generation, snapshot, snapshot_seq, next_seq, markdown, text_size, materialised_at, updated_at)
		SELECT ?, 1, ?, 0, 1, markdown, text_size, materialised_at, ? FROM doc_state WHERE page_id = ?`, to, state, now, from)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 && r != nil {
		if _, err := tx.ExecContext(ctx, `INSERT INTO doc_state (page_id, generation, snapshot, snapshot_seq, next_seq, markdown, text_size, materialised_at, updated_at)
			VALUES (?, 1, ?, 0, 1, '', 0, 0, ?)`, to, state, now); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO doc_refs (page_id, kind, target) SELECT ?, kind, target FROM doc_refs WHERE page_id = ? AND NOT (kind = 'page' AND target = ?)", to, from, to)
	return err
}

func (s *Service) Remap(ctx context.Context, tx *sql.Tx, mapping map[string]string) error {
	copied := make(map[string]bool, len(mapping))
	for _, to := range mapping {
		copied[to] = true
	}
	hosts := map[string]bool{}
	for _, to := range mapping {
		var parent sql.NullString
		err := tx.QueryRowContext(ctx, "SELECT parent_id FROM pages WHERE id = ? AND kind = 'collection'", to).Scan(&parent)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if parent.Valid && copied[parent.String] {
			hosts[parent.String] = true
		}
	}
	for to := range hosts {
		var snap []byte
		err := tx.QueryRowContext(ctx, "SELECT snapshot FROM doc_state WHERE page_id = ?", to).Scan(&snap)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && len(snap) == 0) {
			continue
		}
		if err != nil {
			return err
		}
		d := crdt.New()
		if err := crdt.ApplyUpdateV1(d, snap, nil); err != nil {
			d.Destroy()
			return err
		}
		frag := d.GetXmlFragment(Root)
		var hits []*crdt.YXmlElement
		var walk func(n any)
		walk = func(n any) {
			for _, c := range children(n) {
				if el, ok := c.(*crdt.YXmlElement); ok {
					if _, moved := mapping[attrString(el, "collection")]; moved && el.NodeName == "database" {
						hits = append(hits, el)
					}
					walk(el)
				}
			}
		}
		walk(frag)
		if len(hits) == 0 {
			d.Destroy()
			continue
		}
		d.Transact(func(txn *crdt.Transaction) {
			for _, el := range hits {
				el.SetAttributeValue(txn, "collection", mapping[attrString(el, "collection")])
			}
		}, &origin{})
		sum := Summarise(frag, nil)
		state := crdt.EncodeStateAsUpdateV1(d, nil)
		d.Destroy()
		if _, err := tx.ExecContext(ctx, "UPDATE doc_state SET snapshot = ?, markdown = ?, text_size = ? WHERE page_id = ?", state, sum.Markdown, len(sum.Text), to); err != nil {
			return err
		}
		if err := writeRefs(ctx, tx, to, sum); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) withRoom(ctx context.Context, id string, fn func(r *room)) error {
	r, err := s.acquire(ctx, id)
	if err != nil {
		return err
	}
	defer s.release(r)
	r.mu.Lock()
	defer r.mu.Unlock()
	fn(r)
	return nil
}

func (s *Service) Render(ctx context.Context, p *policy.Principal, id string) (string, error) {
	var out string
	var refs []string
	synced := map[syncedRef]bool{}
	err := s.withRoom(ctx, id, func(r *room) {
		refs = r.summary(nil).Pages
		syncedRefs(children(r.fragment()), synced)
	})
	if err != nil {
		return "", err
	}
	titles := s.pages.Titles(ctx, p, refs)
	var crumbs []page.Crumb
	if d, err := s.pages.Get(ctx, p, id); err == nil {
		crumbs = append(append(crumbs, d.Path...), page.Crumb{ID: d.ID, Title: d.Title, Icon: d.Icon})
	}
	foreign := s.renderSynced(ctx, p, id, synced)
	err = s.withRoom(ctx, id, func(r *room) {
		out = r.render(func(id string) string { return titles[id] }, crumbs, foreign)
	})
	return out, err
}

func (s *Service) LiveFile(pageID, fileID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.rooms[pageID]
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.summary(nil).Files {
		if f == fileID {
			return true
		}
	}
	return false
}

func (s *Service) Markdown(ctx context.Context, p *policy.Principal, id string) (string, error) {
	if err := s.pages.CanRead(ctx, p, id); err != nil {
		return "", err
	}
	var refs []string
	if err := s.withRoom(ctx, id, func(r *room) { refs = r.summary(nil).Pages }); err != nil {
		return "", err
	}
	titles := s.pages.Titles(ctx, p, refs)
	var md string
	err := s.withRoom(ctx, id, func(r *room) {
		md = r.summary(func(id string) string { return titles[id] }).Markdown
	})
	return md, err
}

type WriteInput struct {
	Markdown string `json:"markdown"`
	Mode     string `json:"mode"`
	After    string `json:"after"`
}

func (s *Service) fileLookup(ctx context.Context, workspaceID string) FileLookup {
	return func(id string) (FileInfo, bool) {
		var f FileInfo
		err := s.st.R.QueryRowContext(ctx, "SELECT mime, name, size FROM files WHERE id = ? AND workspace_id = ?", id, workspaceID).Scan(&f.Mime, &f.Name, &f.Size)
		return f, err == nil
	}
}

func (s *Service) Write(ctx context.Context, p *policy.Principal, id string, in WriteInput) error {
	workspaceID, err := s.pages.CanWrite(ctx, p, id)
	if err != nil {
		return err
	}
	if len(in.Markdown) > 4<<20 {
		return httpx.Fail(http.StatusRequestEntityTooLarge, "too_large", "Markdown can be up to 4 MB per write")
	}
	switch in.Mode {
	case "", "append", "prepend", "replace", "after":
	default:
		return httpx.Fail(http.StatusBadRequest, "bad_mode", "mode is append, prepend, replace or after")
	}
	nodes := ParseMarkdown(in.Markdown, s.fileLookup(ctx, workspaceID))
	var missing, busy bool
	err = s.withRoom(ctx, id, func(r *room) {
		if in.Mode == "replace" && !p.IsSession() && r.personIn("") {
			busy = true
			return
		}
		frag := r.fragment()
		at := frag.Len()
		switch in.Mode {
		case "prepend":
			at = 0
		case "after":
			at = -1
			for i, c := range frag.Children() {
				if el, ok := c.(*crdt.YXmlElement); ok && attrString(el, "id") == in.After {
					at = i + 1
				}
			}
			if at < 0 {
				missing = true
				return
			}
		}
		r.doc.Transact(func(txn *crdt.Transaction) {
			if in.Mode == "replace" {
				if n := frag.Len(); n > 0 {
					frag.Delete(txn, 0, n)
				}
				at = 0
			}
			writeNodes(txn, frag, at, nodes)
		}, &origin{user: p.UserID})
	})
	if err != nil {
		return err
	}
	if busy {
		return personTyping("")
	}
	if missing {
		return httpx.Fail(http.StatusNotFound, "block_not_found", "no top-level block has that id")
	}
	s.importRemote(p, id)
	return nil
}

func (s *Service) Backlinks(ctx context.Context, p *policy.Principal, id string) ([]*page.Meta, error) {
	if err := s.pages.CanRead(ctx, p, id); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT page_id FROM doc_refs WHERE kind = 'page' AND target = ?", id)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var x string
		if rows.Scan(&x) == nil {
			ids = append(ids, x)
		}
	}
	rows.Close()
	return s.pages.Readable(ctx, p, ids)
}

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module() app.Module { return module{s, s.auth} }

func (module) ID() string { return "doc" }

func (m module) Stop()                     { m.s.Stop() }
func (m module) Drain(ctx context.Context) { m.s.Drain(ctx) }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	r.HandleFunc("GET /sync", a.RequireSession(s.serveSync))
	m.mountVersions(func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, a.Require(h))
	})
	r.HandleFunc("GET /api/pages/{id}/markdown", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		md, err := s.Markdown(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(md))
		return nil
	}))
	r.HandleFunc("PUT /api/pages/{id}/markdown", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in WriteInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.Write(r.Context(), p, r.PathValue("id"), in); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	r.HandleFunc("GET /api/pages/{id}/backlinks", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Backlinks(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"pages": list})
		return nil
	}))
	return nil
}
