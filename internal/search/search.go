package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	indexVersion = "1"
	batchSize    = 100
	debounce     = 400 * time.Millisecond
	maxLimit     = 50
	candidates   = 200
	recent       = 5000
	rankCap      = 2000
)

type Service struct {
	st      *store.Store
	gate    *policy.Gate
	auth    *auth.Service
	pages   *page.Service
	log     *slog.Logger
	rowText func(ctx context.Context, id string) (string, bool)
	rowIDs  func(ctx context.Context, collectionID string) []string

	mu         sync.Mutex
	dirty      map[string]struct{}
	rebuilding bool
	wake       chan struct{}
	cancel     context.CancelFunc
	done       chan struct{}
}

func New(st *store.Store, gate *policy.Gate, a *auth.Service, pages *page.Service, log *slog.Logger) *Service {
	s := &Service{st: st, gate: gate, auth: a, pages: pages, log: log, dirty: map[string]struct{}{}, wake: make(chan struct{}, 1)}
	pages.OnChange(func(_ context.Context, ids []string) { s.Touch(ids...) })
	return s
}

func (s *Service) SetRows(text func(ctx context.Context, id string) (string, bool), ids func(ctx context.Context, collectionID string) []string) {
	s.rowText, s.rowIDs = text, ids
}

func (s *Service) RowsChanged(ctx context.Context, collectionID string, ids []string) {
	for _, id := range ids {
		if id == "*" {
			if s.rowIDs != nil {
				s.Touch(s.rowIDs(ctx, collectionID)...)
			}
			return
		}
	}
	s.Touch(ids...)
}

func (s *Service) Touch(ids ...string) {
	if len(ids) == 0 {
		return
	}
	s.mu.Lock()
	for _, id := range ids {
		if id != "" && id != "*" {
			s.dirty[id] = struct{}{}
		}
	}
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (module) ID() string { return "search" }

type module struct{ s *Service }

func (s *Service) Module() app.Module { return module{s} }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.run(ctx)
	r.HandleFunc("GET /api/search", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		q := r.URL.Query()
		in := Query{
			Q: q.Get("q"), WorkspaceID: q.Get("workspace_id"), In: q.Get("in"), Parent: q.Get("parent"),
			Date: q.Get("date"), Sort: q.Get("sort"),
		}
		for _, c := range strings.Split(q.Get("creator"), ",") {
			if c = strings.TrimSpace(c); c != "" {
				in.Creators = append(in.Creators, c)
			}
		}
		in.From, _ = strconv.ParseInt(q.Get("from"), 10, 64)
		in.To, _ = strconv.ParseInt(q.Get("to"), 10, 64)
		in.Limit, _ = strconv.Atoi(q.Get("limit"))
		in.Offset, _ = strconv.Atoi(q.Get("offset"))
		if cur := q.Get("cursor"); cur != "" {
			c, err := httpx.DecodeCursor(cur)
			if err != nil {
				return err
			}
			in.Offset = c.Offset
		}
		res, err := s.Search(r.Context(), p, in)
		if err != nil {
			return err
		}
		if res.More {
			res.NextCursor = httpx.EncodeCursor(httpx.Cursor{Offset: in.Offset + len(res.Hits)})
		}
		httpx.JSONZ(w, r, http.StatusOK, res)
		return nil
	}))
	return nil
}

func (m module) Stop() {
	if m.s.cancel != nil {
		m.s.cancel()
		<-m.s.done
	}
}

func (s *Service) run(ctx context.Context) {
	defer close(s.done)
	s.prepare(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(debounce):
		}
		for s.drain(ctx) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(15 * time.Millisecond):
			}
		}
	}
}

func (s *Service) prepare(ctx context.Context) {
	_ = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM search_fts WHERE rowid IN (SELECT id FROM search_parts WHERE page_id NOT IN (SELECT id FROM pages))"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM search_parts WHERE page_id NOT IN (SELECT id FROM pages)")
		return err
	})
	var v string
	_ = s.st.R.QueryRowContext(ctx, "SELECT value FROM search_meta WHERE key = 'version'").Scan(&v)
	if v == indexVersion {
		return
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT id FROM pages WHERE trashed_at = 0")
	if err != nil {
		s.log.Warn("search: rebuild", "err", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	s.mu.Lock()
	s.rebuilding = true
	s.mu.Unlock()
	s.log.Info("search: building the index", "pages", len(ids))
	if len(ids) == 0 {
		s.finishRebuild(ctx)
		return
	}
	s.Touch(ids...)
}

func (s *Service) finishRebuild(ctx context.Context) {
	s.mu.Lock()
	was := s.rebuilding && len(s.dirty) == 0
	if was {
		s.rebuilding = false
	}
	s.mu.Unlock()
	if !was {
		return
	}
	if _, err := s.st.W.ExecContext(ctx, "INSERT INTO search_meta (key, value) VALUES ('version', ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value", indexVersion); err != nil {
		s.log.Warn("search: version", "err", err)
		return
	}
	s.log.Info("search: index ready")
}

func (s *Service) drain(ctx context.Context) bool {
	s.mu.Lock()
	batch := make([]string, 0, batchSize)
	for id := range s.dirty {
		batch = append(batch, id)
		delete(s.dirty, id)
		if len(batch) == batchSize {
			break
		}
	}
	more := len(s.dirty) > 0
	s.mu.Unlock()
	if len(batch) == 0 {
		s.finishRebuild(ctx)
		return false
	}
	docs := make(map[string]*source, len(batch))
	for _, id := range batch {
		src, err := s.load(ctx, id)
		if err != nil {
			s.log.Warn("search: read", "page", id, "err", err)
			continue
		}
		docs[id] = src
	}
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		for _, id := range batch {
			if src, ok := docs[id]; ok {
				if err := write(ctx, tx, id, src); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		s.log.Warn("search: write", "err", err)
	}
	if !more {
		s.finishRebuild(ctx)
	}
	return more
}

type source struct {
	gone  bool
	ws    string
	title string
	extra string
	parts []passage
	props string
}

func (s *Service) load(ctx context.Context, id string) (*source, error) {
	var ws, title, desc, tags string
	var trashed int64
	var row int
	err := s.st.R.QueryRowContext(ctx, "SELECT workspace_id, title, description, tags, trashed_at, is_row FROM pages WHERE id = ?", id).Scan(&ws, &title, &desc, &tags, &trashed, &row)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && trashed > 0) {
		return &source{gone: true}, nil
	}
	if err != nil {
		return nil, err
	}
	src := &source{ws: ws, title: title}
	var list []string
	_ = json.Unmarshal([]byte(tags), &list)
	src.extra = strings.TrimSpace(desc + "\n" + strings.Join(list, " "))
	var md sql.NullString
	if err := s.st.R.QueryRowContext(ctx, "SELECT markdown FROM doc_state WHERE page_id = ?", id).Scan(&md); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	src.parts = passages(md.String)
	if row == 1 && s.rowText != nil {
		src.props, _ = s.rowText(ctx, id)
	}
	return src, nil
}

func write(ctx context.Context, tx *sql.Tx, id string, src *source) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM search_fts WHERE rowid IN (SELECT id FROM search_parts WHERE page_id = ?)", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM search_parts WHERE page_id = ?", id); err != nil {
		return err
	}
	if src.gone {
		return nil
	}
	add := func(part int, kind, title, heading, body string) error {
		res, err := tx.ExecContext(ctx, "INSERT INTO search_parts (page_id, part, kind, heading) VALUES (?, ?, ?, ?)", id, part, kind, heading)
		if err != nil {
			return err
		}
		rid, err := res.LastInsertId()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO search_fts (rowid, title, heading, body, ws) VALUES (?, ?, ?, ?, ?)", rid, title, heading, body, src.ws)
		return err
	}
	if err := add(0, "title", src.title, "", src.extra); err != nil {
		return err
	}
	n := 1
	if src.props != "" {
		if err := add(n, "props", "", "", src.props); err != nil {
			return err
		}
		n++
	}
	for _, p := range src.parts {
		if err := add(n, "body", "", p.heading, p.text); err != nil {
			return err
		}
		n++
	}
	return nil
}

type Query struct {
	Q           string
	WorkspaceID string
	In          string
	Parent      string
	Date        string
	Sort        string
	Creators    []string
	From        int64
	To          int64
	Limit       int
	Offset      int
}

type Hit struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	TitleHL   string       `json:"title_hl"`
	Icon      string       `json:"icon"`
	Kind      string       `json:"kind"`
	Row       bool         `json:"row"`
	Path      []page.Crumb `json:"path"`
	Match     string       `json:"match"`
	Heading   string       `json:"heading"`
	Snippet   string       `json:"snippet"`
	CreatedAt int64        `json:"created_at"`
	CreatedBy string       `json:"created_by"`
	UpdatedAt int64        `json:"updated_at"`
	parentID  string
	part      int64
}

type Result struct {
	Hits       []*Hit  `json:"hits"`
	More       bool    `json:"more"`
	TookMS     float64 `json:"took_ms"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

func ftsQuery(q string) string {
	var terms []string
	for _, f := range strings.Fields(q) {
		f = strings.ReplaceAll(f, `"`, "")
		if strings.IndexFunc(f, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0 {
			continue
		}
		terms = append(terms, `"`+f+`"`)
	}
	if len(terms) == 0 {
		return ""
	}
	terms[len(terms)-1] += "*"
	return strings.Join(terms, " ")
}

func (s *Service) Search(ctx context.Context, p *policy.Principal, in Query) (*Result, error) {
	start := time.Now()
	visible, err := s.pages.Filter(ctx, p, in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 || limit > maxLimit {
		limit = 20
	}
	offset := max(in.Offset, 0)
	res := &Result{Hits: []*Hit{}}
	expr := ftsQuery(in.Q)
	if expr == "" {
		res.TookMS = ms(start)
		return res, nil
	}
	templates, err := s.templates(ctx, in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	scope := `{ws} : "` + in.WorkspaceID + `" AND `
	var cte, cond strings.Builder
	var cteArgs, condArgs []any
	if in.Parent != "" {
		cte.WriteString("WITH RECURSIVE sub(id) AS (SELECT ? UNION SELECT c.id FROM pages c JOIN sub ON c.parent_id = sub.id) ")
		cteArgs = append(cteArgs, in.Parent)
		cond.WriteString(" AND sp.page_id IN (SELECT id FROM sub) AND sp.page_id != ?")
		condArgs = append(condArgs, in.Parent)
	}
	if len(in.Creators) > 0 {
		cond.WriteString(" AND p.created_by IN (" + strings.TrimSuffix(strings.Repeat("?,", len(in.Creators)), ",") + ")")
		for _, c := range in.Creators {
			condArgs = append(condArgs, c)
		}
	}
	col := "p.updated_at"
	if in.Date == "created" {
		col = "p.created_at"
	}
	if in.From > 0 {
		cond.WriteString(" AND " + col + " >= ?")
		condArgs = append(condArgs, in.From)
	}
	if in.To > 0 {
		cond.WriteString(" AND " + col + " < ?")
		condArgs = append(condArgs, in.To)
	}
	pool := candidates
	if len(condArgs) > 0 {
		pool = 20 * candidates
	}
	seen := map[string]bool{}
	skipped := 0
	run := func(inner string, innerArgs []any, order string) (int, error) {
		q := cte.String() + `SELECT sp.id, sp.page_id, sp.kind, sp.heading, p.workspace_id, p.title, p.icon, p.kind, p.is_row, COALESCE(p.parent_id, ''), p.created_at, COALESCE(p.created_by, ''), p.updated_at
			FROM (` + inner + `) f JOIN search_parts sp ON sp.id = f.rowid JOIN pages p ON p.id = sp.page_id
			WHERE p.trashed_at = 0` + cond.String() + order
		args := append(append(append([]any{}, cteArgs...), innerArgs...), condArgs...)
		rows, err := s.st.R.QueryContext(ctx, q, args...)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		read := 0
		for rows.Next() {
			read++
			h := &Hit{}
			var row int
			var ws string
			if err := rows.Scan(&h.part, &h.ID, &h.Match, &h.Heading, &ws, &h.Title, &h.Icon, &h.Kind, &row, &h.parentID, &h.CreatedAt, &h.CreatedBy, &h.UpdatedAt); err != nil {
				return read, err
			}
			if ws != in.WorkspaceID || seen[h.ID] || !visible(h.ID) || templates[h.ID] {
				continue
			}
			seen[h.ID] = true
			if skipped < offset {
				skipped++
				continue
			}
			if len(res.Hits) == limit {
				res.More = true
				return read, nil
			}
			h.Row = row == 1
			res.Hits = append(res.Hits, h)
		}
		return read, rows.Err()
	}
	ranked := func(m string, cap int) (string, []any) {
		floor := s.floor(ctx, m, cap)
		return "SELECT rowid, rank FROM search_fts WHERE search_fts MATCH ? AND rowid >= ? ORDER BY rank LIMIT " + strconv.Itoa(pool), []any{m, floor}
	}
	tier := func(m string, cap int, order string) error {
		inner, args := ranked(m, cap)
		read, err := run(inner, args, order)
		if err != nil || res.More || (read < pool && args[1].(int64) == 0) {
			return err
		}
		inner, args = ranked(scope+m, cap)
		_, err = run(inner, args, order)
		return err
	}
	switch in.Sort {
	case "edited", "created":
		order := " ORDER BY p.updated_at DESC"
		if in.Sort == "created" {
			order = " ORDER BY p.created_at DESC"
		}
		m := "(" + expr + ")"
		if in.In == "title" {
			m = "{title} : (" + expr + ")"
		}
		newest := "SELECT rowid FROM search_fts WHERE search_fts MATCH ? ORDER BY rowid DESC LIMIT " + strconv.Itoa(recent)
		read, rerr := run(newest, []any{m}, order)
		if rerr == nil && !res.More && read == recent {
			_, rerr = run(newest, []any{scope + m}, order)
		}
		err = rerr
	default:
		err = tier("{title} : ("+expr+")", 0, " ORDER BY f.rank")
		if err == nil && in.In != "title" && !res.More {
			err = tier("{heading body} : ("+expr+")", rankCap, " ORDER BY f.rank")
		}
	}
	if err != nil {
		return nil, err
	}
	if err := s.highlight(ctx, in.Q, res.Hits); err != nil {
		return nil, err
	}
	for _, h := range res.Hits {
		h.Path = s.path(ctx, h.parentID)
	}
	res.TookMS = ms(start)
	s.log.Debug("search", "ms", res.TookMS, "hits", len(res.Hits))
	return res, nil
}

func (s *Service) floor(ctx context.Context, match string, cap int) int64 {
	if cap <= 0 {
		return 0
	}
	var id int64
	if err := s.st.R.QueryRowContext(ctx, "SELECT rowid FROM search_fts WHERE search_fts MATCH ? ORDER BY rowid DESC LIMIT 1 OFFSET ?", match, cap-1).Scan(&id); err != nil {
		return 0
	}
	return id
}

func (s *Service) highlight(ctx context.Context, q string, hits []*Hit) error {
	if len(hits) == 0 {
		return nil
	}
	terms := queryTerms(q)
	by := make(map[int64]*Hit, len(hits))
	args := []any{}
	for _, h := range hits {
		by[h.part] = h
		args = append(args, h.part)
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT rowid, body FROM search_fts WHERE rowid IN (`+strings.TrimSuffix(strings.Repeat("?,", len(hits)), ",")+`)`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var body string
		if err := rows.Scan(&id, &body); err != nil {
			return err
		}
		h := by[id]
		if h == nil {
			continue
		}
		if t, ok := mark(h.Title, terms); ok {
			h.TitleHL = t
		}
		if snip, ok := snippet(body, terms); ok || h.Match != "title" {
			h.Snippet = snip
		}
	}
	return rows.Err()
}

func ms(t time.Time) float64 {
	return float64(time.Since(t).Microseconds()) / 1000
}

func (s *Service) templates(ctx context.Context, workspaceID string) (map[string]bool, error) {
	rows, err := s.st.R.QueryContext(ctx, `WITH RECURSIVE tpl(id) AS (
		SELECT id FROM pages WHERE workspace_id = ? AND is_template = 1
		UNION SELECT c.id FROM pages c JOIN tpl ON c.parent_id = tpl.id)
		SELECT id FROM tpl`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (s *Service) path(ctx context.Context, parentID string) []page.Crumb {
	out := []page.Crumb{}
	if parentID == "" {
		return out
	}
	rows, err := s.st.R.QueryContext(ctx, `WITH RECURSIVE up(id, parent_id, title, icon, depth) AS (
		SELECT id, parent_id, title, icon, 0 FROM pages WHERE id = ?
		UNION ALL SELECT c.id, c.parent_id, c.title, c.icon, up.depth + 1 FROM pages c JOIN up ON c.id = up.parent_id WHERE up.depth < 64)
		SELECT id, title, icon FROM up ORDER BY depth DESC`, parentID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var c page.Crumb
		if rows.Scan(&c.ID, &c.Title, &c.Icon) == nil {
			out = append(out, c)
		}
	}
	return out
}
