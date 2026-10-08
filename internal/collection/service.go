package collection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/order"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Service struct {
	st        *store.Store
	pages     *page.Service
	audit     *audit.Log
	hub       *events.Hub
	log       *slog.Logger
	cache     *cache
	epoch     atomic.Int64
	formLimit *httpx.Limiter
	stash     *stash
	results   *resultCache
}

func New(st *store.Store, pages *page.Service, al *audit.Log, hub *events.Hub, log *slog.Logger) *Service {
	s := &Service{st: st, pages: pages, audit: al, hub: hub, log: log, cache: &cache{snaps: map[string]*snap{}},
		formLimit: httpx.NewLimiter(20, 10), stash: newStash(), results: newResultCache()}
	s.epoch.Store(time.Now().UnixMilli())
	pages.OnCreate(s.onCreate)
	pages.OnDuplicated(s.onDuplicated)
	pages.OnMoveWorkspace(s.onMove)
	pages.OnChange(s.onChange)
	return s
}

func (s *Service) bump() int64 { return s.epoch.Add(1) }

func (s *Service) location(ctx context.Context, p *policy.Principal) *time.Location {
	if p == nil {
		return time.Local
	}
	var tz string
	_ = s.st.R.QueryRowContext(ctx, "SELECT time_zone FROM users WHERE id = ?", p.UserID).Scan(&tz)
	if tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return time.Local
}

func (s *Service) mondayFirst(ctx context.Context, p *policy.Principal) bool {
	if p == nil {
		return true
	}
	var ws string
	_ = s.st.R.QueryRowContext(ctx, "SELECT week_start FROM users WHERE id = ?", p.UserID).Scan(&ws)
	return ws != "sunday"
}

type View struct {
	ID           string     `json:"id"`
	CollectionID string     `json:"collection_id"`
	SortKey      string     `json:"sort_key"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Config       ViewConfig `json:"config"`
	CreatedAt    int64      `json:"created_at"`
	UpdatedAt    int64      `json:"updated_at"`
	FormToken    string     `json:"form_token,omitempty"`
}

type Out struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Title       string `json:"title"`
	Icon        string `json:"icon"`
	Inline      bool   `json:"inline"`
	Schema      Schema `json:"schema"`
	Views       []View `json:"views"`
	CanWrite    bool   `json:"can_write"`
	Rows        int    `json:"rows"`
	Epoch       int64  `json:"epoch"`
}

var viewTypes = map[string]bool{"table": true, "board": true, "list": true, "gallery": true, "calendar": true, "timeline": true, "form": true, "chart": true, "map": true, "feed": true, "dashboard": true}

func defaultViewName(t string) string {
	switch t {
	case "table":
		return "Table"
	case "board":
		return "Board"
	case "list":
		return "List"
	case "gallery":
		return "Gallery"
	case "calendar":
		return "Calendar"
	case "timeline":
		return "Timeline"
	case "form":
		return "Form"
	case "chart":
		return "Chart"
	case "map":
		return "Map"
	case "feed":
		return "Feed"
	}
	return "Dashboard"
}

type createExtra struct {
	Inline bool   `json:"inline"`
	View   string `json:"view"`
	Schema Schema `json:"schema"`
}

func (s *Service) onCreate(ctx context.Context, tx *sql.Tx, id, workspaceID, kind string, extra json.RawMessage) error {
	if kind != "collection" {
		return nil
	}
	var ex createExtra
	if len(extra) > 0 {
		if err := json.Unmarshal(extra, &ex); err != nil {
			return httpx.Fail(http.StatusBadRequest, "bad_extra", "the database options are not valid")
		}
	}
	schema := DefaultSchema()
	if len(ex.Schema) > 0 {
		cleaned, err := s.cleanSchema(ctx, tx, workspaceID, id, ex.Schema)
		if err != nil {
			return err
		}
		schema = cleaned
	}
	if !viewTypes[ex.View] || ex.View == "dashboard" {
		ex.View = "table"
	}
	return s.insertCollection(ctx, tx, id, workspaceID, schema, ex.Inline, ex.View)
}

func (s *Service) insertCollection(ctx context.Context, tx *sql.Tx, id, workspaceID string, schema Schema, inline bool, view string) error {
	raw, _ := json.Marshal(schema)
	now := store.NowMS()
	if _, err := tx.ExecContext(ctx, `INSERT INTO collections (page_id, workspace_id, schema, inline, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, workspaceID, string(raw), store.Bool(inline), now, now); err != nil {
		return err
	}
	cfg := ViewConfig{}
	if view == "board" {
		for _, p := range schema {
			if p.Type == TStatus || p.Type == TSelect {
				cfg.Group = &Grouping{Prop: p.ID}
				break
			}
		}
	}
	cb, _ := json.Marshal(cfg)
	_, err := tx.ExecContext(ctx, `INSERT INTO collection_views (id, collection_id, sort_key, name, type, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ids.New(), id, order.After(""), defaultViewName(view), view, string(cb), now, now)
	return err
}

func (s *Service) cleanSchema(ctx context.Context, q store.Querier, workspaceID, selfID string, in Schema) (Schema, error) {
	out := Schema{}
	seen := map[string]bool{}
	hasTitle := false
	for _, p := range in {
		c := *p
		if c.Type == TTitle {
			if hasTitle {
				continue
			}
			hasTitle = true
			c.ID = TTitle
		} else if c.ID == "" || seen[c.ID] || c.ID == TTitle {
			c.ID = shortID()
		}
		if err := c.normalise(); err != nil {
			return nil, err
		}
		if c.Type == TRelation {
			if c.Collection == "" || c.Collection == "self" {
				c.Collection = selfID
			}
			if c.Collection != selfID {
				var ws string
				if err := q.QueryRowContext(ctx, "SELECT workspace_id FROM collections WHERE page_id = ?", c.Collection).Scan(&ws); err != nil || ws != workspaceID {
					return nil, badSchema("a relation must point to a database in this workspace")
				}
			}
			c.Pair = ""
		}
		seen[c.ID] = true
		out = append(out, &c)
	}
	if !hasTitle {
		out = append(Schema{{ID: TTitle, Name: "Name", Type: TTitle}}, out...)
	}
	if len(out) > maxProps {
		return nil, badSchema("a database can have up to 200 properties")
	}
	return out, nil
}

func (s *Service) onDuplicated(ctx context.Context, tx *sql.Tx, mapping map[string]string) error {
	from := make([]string, 0, len(mapping))
	for k := range mapping {
		from = append(from, k)
	}
	sort.Strings(from)
	for _, old := range from {
		nid := mapping[old]
		var ws, raw string
		var inline int
		err := tx.QueryRowContext(ctx, "SELECT workspace_id, schema, inline FROM collections WHERE page_id = ?", old).Scan(&ws, &raw, &inline)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		var schema Schema
		if err := json.Unmarshal([]byte(raw), &schema); err != nil {
			return err
		}
		for _, p := range schema {
			if p.Type != TRelation {
				continue
			}
			if t, ok := mapping[p.Collection]; ok {
				p.Collection = t
			} else {
				p.Pair = ""
			}
		}
		sb, _ := json.Marshal(schema)
		now := store.NowMS()
		if _, err := tx.ExecContext(ctx, `INSERT INTO collections (page_id, workspace_id, schema, inline, next_number, created_at, updated_at)
			SELECT ?, workspace_id, ?, inline, next_number, ?, ? FROM collections WHERE page_id = ?`, nid, string(sb), now, now, old); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, "SELECT sort_key, name, type, config FROM collection_views WHERE collection_id = ? ORDER BY sort_key", old)
		if err != nil {
			return err
		}
		type vrow struct{ key, name, typ, cfg string }
		var views []vrow
		for rows.Next() {
			var v vrow
			if err := rows.Scan(&v.key, &v.name, &v.typ, &v.cfg); err != nil {
				rows.Close()
				return err
			}
			views = append(views, v)
		}
		rows.Close()
		for _, v := range views {
			if _, err := tx.ExecContext(ctx, `INSERT INTO collection_views (id, collection_id, sort_key, name, type, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				ids.New(), nid, v.key, v.name, v.typ, v.cfg, now, now); err != nil {
				return err
			}
		}
	}
	for _, old := range from {
		nid := mapping[old]
		var colID, vals string
		var number int64
		err := tx.QueryRowContext(ctx, "SELECT collection_id, vals, number FROM collection_rows WHERE page_id = ?", old).Scan(&colID, &vals, &number)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		target := colID
		if t, ok := mapping[colID]; ok {
			target = t
		}
		var raw string
		if err := tx.QueryRowContext(ctx, "SELECT schema FROM collections WHERE page_id = ?", target).Scan(&raw); err != nil {
			return err
		}
		var schema Schema
		_ = json.Unmarshal([]byte(raw), &schema)
		m := map[string]json.RawMessage{}
		_ = json.Unmarshal([]byte(vals), &m)
		for _, p := range schema {
			if p.Type != TRelation {
				continue
			}
			list := strList(m[p.ID])
			if len(list) == 0 {
				continue
			}
			out := make([]string, 0, len(list))
			for _, id := range list {
				if t, ok := mapping[id]; ok {
					out = append(out, t)
				} else if _, inside := mapping[p.Collection]; !inside && p.Pair == "" {
					out = append(out, id)
				}
			}
			if len(out) == 0 {
				delete(m, p.ID)
			} else {
				m[p.ID] = mustJSON(out)
			}
		}
		if target == colID {
			if err := tx.QueryRowContext(ctx, "UPDATE collections SET next_number = next_number + 1 WHERE page_id = ? RETURNING next_number - 1", target).Scan(&number); err != nil {
				return err
			}
		}
		vb, _ := json.Marshal(m)
		if _, err := tx.ExecContext(ctx, "INSERT INTO collection_rows (page_id, collection_id, vals, number) VALUES (?, ?, ?, ?)", nid, target, string(vb), number); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) onMove(ctx context.Context, tx *sql.Tx, list []string, workspaceID string) error {
	for _, id := range list {
		if _, err := tx.ExecContext(ctx, "UPDATE collections SET workspace_id = ? WHERE page_id = ?", workspaceID, id); err != nil {
			return err
		}
	}
	return nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

func anyArgs(list []string, lead ...any) []any {
	out := append([]any{}, lead...)
	for _, x := range list {
		out = append(out, x)
	}
	return out
}

func (s *Service) onChange(ctx context.Context, list []string) {
	byCol := map[string][]string{}
	const chunk = 400
	for start := 0; start < len(list); start += chunk {
		part := list[start:min(start+chunk, len(list))]
		rows, err := s.st.R.QueryContext(ctx, "SELECT page_id, collection_id FROM collection_rows WHERE page_id IN ("+placeholders(len(part))+")", anyArgs(part)...)
		if err != nil {
			s.log.Warn("collection: change lookup", "err", err)
			return
		}
		for rows.Next() {
			var pid, cid string
			if rows.Scan(&pid, &cid) == nil {
				byCol[cid] = append(byCol[cid], pid)
			}
		}
		rows.Close()
		for _, id := range part {
			if sn := s.cache.peek(id); sn != nil {
				var trashed int64
				_ = s.st.R.QueryRowContext(ctx, "SELECT trashed_at FROM pages WHERE id = ?", id).Scan(&trashed)
				if trashed > 0 {
					s.cache.drop(id)
				}
			}
		}
	}
	for col, rowIDs := range byCol {
		if err := s.cache.refresh(ctx, s.st.R, col, rowIDs); err != nil {
			s.log.Warn("collection: refresh", "err", err)
		}
		s.emitRows(ctx, col, rowIDs)
	}
}

func (s *Service) emit(ctx context.Context, collectionID string, data map[string]any) {
	m, err := s.pages.Load(ctx, collectionID)
	if err != nil {
		return
	}
	owner, err := s.pages.PrivateOwner(ctx, collectionID)
	if err != nil {
		return
	}
	data["collection_id"] = collectionID
	data["epoch"] = s.epoch.Load()
	s.hub.Collection(m.WorkspaceID, owner, data)
}

func (s *Service) emitRows(ctx context.Context, collectionID string, rowIDs []string) {
	s.bump()
	if len(rowIDs) > 200 {
		rowIDs = []string{"*"}
	}
	s.emit(ctx, collectionID, map[string]any{"what": "rows", "ids": rowIDs})
	for _, dep := range s.dependents(ctx, collectionID) {
		s.emit(ctx, dep, map[string]any{"what": "rows", "ids": []string{"*"}})
	}
}

func (s *Service) dependents(ctx context.Context, collectionID string) []string {
	rows, err := s.st.R.QueryContext(ctx, `SELECT c.page_id FROM collections c JOIN collections t ON t.workspace_id = c.workspace_id
		WHERE t.page_id = ? AND c.page_id != ? AND instr(c.schema, ?) > 0`, collectionID, collectionID, `"collection":"`+collectionID+`"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out
}

func (s *Service) readable(ctx context.Context, p *policy.Principal, id string) (*page.Meta, bool, error) {
	m, canWrite, err := s.pages.ReadableMeta(ctx, p, id)
	if err != nil {
		return nil, false, err
	}
	if m.Kind != "collection" {
		return nil, false, store.ErrNotFound
	}
	if err := s.ensure(ctx, m); err != nil {
		return nil, false, err
	}
	return m, canWrite, nil
}

func (s *Service) writable(ctx context.Context, p *policy.Principal, id string) (*page.Meta, error) {
	m, err := s.pages.Writable(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if m.Kind != "collection" {
		return nil, store.ErrNotFound
	}
	if err := s.ensure(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) ensure(ctx context.Context, m *page.Meta) error {
	var n int
	if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM collections WHERE page_id = ?", m.ID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM collections WHERE page_id = ?", m.ID).Scan(&n); err != nil || n > 0 {
			return err
		}
		return s.insertCollection(ctx, tx, m.ID, m.WorkspaceID, DefaultSchema(), false, "table")
	})
}

func (s *Service) views(ctx context.Context, collectionID string) ([]View, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT v.id, v.collection_id, v.sort_key, v.name, v.type, v.config, v.created_at, v.updated_at, COALESCE(f.token, '')
		FROM collection_views v LEFT JOIN form_links f ON f.view_id = v.id WHERE v.collection_id = ? ORDER BY v.sort_key, v.id`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []View{}
	for rows.Next() {
		var v View
		var cfg string
		if err := rows.Scan(&v.ID, &v.CollectionID, &v.SortKey, &v.Name, &v.Type, &cfg, &v.CreatedAt, &v.UpdatedAt, &v.FormToken); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(cfg), &v.Config)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) view(ctx context.Context, id string) (*View, error) {
	var v View
	var cfg string
	err := s.st.R.QueryRowContext(ctx, `SELECT v.id, v.collection_id, v.sort_key, v.name, v.type, v.config, v.created_at, v.updated_at, COALESCE(f.token, '')
		FROM collection_views v LEFT JOIN form_links f ON f.view_id = v.id WHERE v.id = ?`, id).
		Scan(&v.ID, &v.CollectionID, &v.SortKey, &v.Name, &v.Type, &cfg, &v.CreatedAt, &v.UpdatedAt, &v.FormToken)
	if err != nil {
		return nil, store.NoRows(err)
	}
	_ = json.Unmarshal([]byte(cfg), &v.Config)
	return &v, nil
}

func (s *Service) Get(ctx context.Context, p *policy.Principal, id string) (*Out, error) {
	m, canWrite, err := s.readable(ctx, p, id)
	if err != nil {
		return nil, err
	}
	sn, err := s.cache.get(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	views, err := s.views(ctx, id)
	if err != nil {
		return nil, err
	}
	sn.mu.RLock()
	schema := sn.schema.Clone()
	inline := sn.inline
	n := len(sn.rows)
	sn.mu.RUnlock()
	if !canWrite {
		for i := range views {
			views[i].FormToken = ""
		}
	}
	return &Out{ID: id, WorkspaceID: m.WorkspaceID, Title: m.Title, Icon: m.Icon, Inline: inline, Schema: schema, Views: views, CanWrite: canWrite, Rows: n, Epoch: s.epoch.Load()}, nil
}

func (s *Service) Query(ctx context.Context, p *policy.Principal, collectionID string, in QueryInput) (*QueryResult, error) {
	if _, _, err := s.readable(ctx, p, collectionID); err != nil {
		return nil, err
	}
	var cfg ViewConfig
	if in.ViewID != "" {
		v, err := s.view(ctx, in.ViewID)
		if err != nil {
			return nil, err
		}
		if v.CollectionID != collectionID {
			return nil, store.ErrNotFound
		}
		cfg = v.Config
	}
	if in.Filter != nil {
		cfg.Filter = in.Filter
	}
	if in.Sorts != nil {
		cfg.Sorts = in.Sorts
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn := ev.snap(collectionID)
	if sn == nil {
		return nil, store.ErrNotFound
	}
	res := &QueryResult{Rows: []RowOut{}, Order: []string{}, Refs: map[string]Ref{}, Epoch: s.epoch.Load()}
	if len(in.IDs) > 0 {
		for _, id := range in.IDs {
			if r, ok := sn.rows[id]; ok {
				res.Rows = append(res.Rows, ev.rowOut(sn, r, res.Refs))
				res.Order = append(res.Order, id)
			}
		}
		res.Total = len(res.Rows)
		return res, nil
	}
	all := s.results.ordered(ev, sn, cfg, in)
	res.Total = len(all)
	if cfg.Calcs != nil {
		res.Calcs = ev.calcs(sn, all, cfg.Calcs)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 100
	}
	limit = min(limit, 1000)
	groupLimit := in.GroupLimit
	if groupLimit <= 0 {
		groupLimit = 50
	}
	groupLimit = min(groupLimit, 500)
	emitted := map[string]bool{}
	add := func(r *Row) {
		if !emitted[r.ID] {
			emitted[r.ID] = true
			res.Rows = append(res.Rows, ev.rowOut(sn, r, res.Refs))
		}
	}
	makeGroups := func(rows []*Row, g *Grouping, laneKey string) []GroupOut {
		hidden := hiddenSet(g.Hidden)
		var out []GroupOut
		for _, b := range ev.bucketise(sn, rows, g) {
			if g.HideEmpty && len(b.rows) == 0 {
				continue
			}
			gout := GroupOut{Key: b.key, Label: b.label, Color: b.color, Count: len(b.rows), Rows: []string{}, Hidden: hidden[b.key]}
			off, lim := 0, groupLimit
			if in.GroupKey != nil && *in.GroupKey == b.key && (in.LaneKey == nil || *in.LaneKey == laneKey) {
				off, lim = in.Offset, limit
			} else if in.GroupKey != nil {
				continue
			}
			if !gout.Hidden || in.GroupKey != nil {
				for i := off; i < len(b.rows) && i < off+lim; i++ {
					add(b.rows[i])
					gout.Rows = append(gout.Rows, b.rows[i].ID)
				}
			}
			if cfg.Calcs != nil {
				gout.Calcs = ev.calcs(sn, b.rows, cfg.Calcs)
			}
			out = append(out, gout)
		}
		return out
	}
	switch {
	case cfg.Group != nil && cfg.Group.Prop != "" && !in.NoGroups && cfg.SubGroup != nil && cfg.SubGroup.Prop != "":
		hidden := hiddenSet(cfg.SubGroup.Hidden)
		for _, lane := range ev.bucketise(sn, all, cfg.SubGroup) {
			if cfg.SubGroup.HideEmpty && len(lane.rows) == 0 {
				continue
			}
			if in.LaneKey != nil && *in.LaneKey != lane.key {
				continue
			}
			lo := LaneOut{Key: lane.key, Label: lane.label, Color: lane.color, Count: len(lane.rows), Hidden: hidden[lane.key]}
			lo.Groups = makeGroups(lane.rows, cfg.Group, lane.key)
			res.Lanes = append(res.Lanes, lo)
		}
	case cfg.Group != nil && cfg.Group.Prop != "" && !in.NoGroups:
		res.Groups = makeGroups(all, cfg.Group, "")
	default:
		off := max(0, in.Offset)
		res.Offset = off
		for i := off; i < len(all) && i < off+limit; i++ {
			add(all[i])
			res.Order = append(res.Order, all[i].ID)
		}
	}
	return res, nil
}

type SearchHit struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
}

func (s *Service) SearchRows(ctx context.Context, p *policy.Principal, collectionID, q string, limit int) ([]SearchHit, error) {
	if _, _, err := s.readable(ctx, p, collectionID); err != nil {
		return nil, err
	}
	sn, err := s.cache.get(ctx, s.st.R, collectionID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q = strings.ToLower(strings.TrimSpace(q))
	sn.mu.RLock()
	defer sn.mu.RUnlock()
	out := []SearchHit{}
	for _, r := range sn.ordered() {
		if q != "" && !strings.Contains(strings.ToLower(r.Title), q) {
			continue
		}
		out = append(out, SearchHit{ID: r.ID, Title: r.Title, Icon: r.Icon})
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

type rowWrite struct {
	sn     *snap
	schema Schema
	nc     *normCtx
}

func (s *Service) normCtx(ctx context.Context, workspaceID string) *normCtx {
	members := map[string]bool{}
	rows, err := s.st.R.QueryContext(ctx, "SELECT user_id FROM workspace_members WHERE workspace_id = ?", workspaceID)
	if err == nil {
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				members[id] = true
			}
		}
		rows.Close()
	}
	return &normCtx{
		members: members,
		fileOK: func(id string) bool {
			var n int
			_ = s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE id = ? AND workspace_id = ?", id, workspaceID).Scan(&n)
			return n > 0
		},
		targets: func(collectionID string, list []string) []string {
			if len(list) == 0 {
				return list
			}
			ok := map[string]bool{}
			for start := 0; start < len(list); start += 400 {
				part := list[start:min(start+400, len(list))]
				rows, err := s.st.R.QueryContext(ctx, `SELECT r.page_id FROM collection_rows r JOIN pages p ON p.id = r.page_id
					WHERE r.collection_id = ? AND p.trashed_at = 0 AND r.page_id IN (`+placeholders(len(part))+`)`, anyArgs(part, collectionID)...)
				if err != nil {
					return nil
				}
				for rows.Next() {
					var id string
					if rows.Scan(&id) == nil {
						ok[id] = true
					}
				}
				rows.Close()
			}
			out := make([]string, 0, len(list))
			for _, id := range list {
				if ok[id] {
					out = append(out, id)
				}
			}
			return out
		},
	}
}

type RowInput struct {
	Title    string                     `json:"title"`
	Icon     string                     `json:"icon"`
	Values   map[string]json.RawMessage `json:"values"`
	BeforeID string                     `json:"before_id"`
	AfterID  string                     `json:"after_id"`
	Top      bool                       `json:"top"`
}

type RowPatch struct {
	Title    *string                    `json:"title"`
	Values   map[string]json.RawMessage `json:"values"`
	BeforeID string                     `json:"before_id"`
	AfterID  string                     `json:"after_id"`
	Move     bool                       `json:"move"`
}

func (s *Service) rowKey(ctx context.Context, q store.Querier, collectionID, beforeID, afterID string, top bool) (string, error) {
	keyOf := func(id string) (string, error) {
		var k string
		err := q.QueryRowContext(ctx, "SELECT sort_key FROM pages WHERE id = ? AND parent_id = ? AND is_row = 1", id, collectionID).Scan(&k)
		if err != nil {
			return "", httpx.Fail(http.StatusBadRequest, "bad_sibling", "the neighbouring row was not found")
		}
		return k, nil
	}
	near := func(key string, next bool) (string, error) {
		op, dir := ">", "ASC"
		if !next {
			op, dir = "<", "DESC"
		}
		var k sql.NullString
		err := q.QueryRowContext(ctx, `SELECT sort_key FROM pages WHERE parent_id = ? AND is_row = 1 AND sort_key `+op+` ? ORDER BY sort_key `+dir+` LIMIT 1`, collectionID, key).Scan(&k)
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return k.String, err
	}
	switch {
	case afterID != "":
		a, err := keyOf(afterID)
		if err != nil {
			return "", err
		}
		b, err := near(a, true)
		if err != nil {
			return "", err
		}
		return order.Between(a, b)
	case beforeID != "":
		b, err := keyOf(beforeID)
		if err != nil {
			return "", err
		}
		a, err := near(b, false)
		if err != nil {
			return "", err
		}
		return order.Between(a, b)
	case top:
		var first sql.NullString
		if err := q.QueryRowContext(ctx, "SELECT MIN(sort_key) FROM pages WHERE parent_id = ? AND is_row = 1", collectionID).Scan(&first); err != nil {
			return "", err
		}
		return order.Between("", first.String)
	}
	var last sql.NullString
	if err := q.QueryRowContext(ctx, "SELECT MAX(sort_key) FROM pages WHERE parent_id = ? AND is_row = 1", collectionID).Scan(&last); err != nil {
		return "", err
	}
	return order.After(last.String), nil
}

func (s *Service) loadSchema(ctx context.Context, q store.Querier, collectionID string) (Schema, error) {
	var raw string
	if err := q.QueryRowContext(ctx, "SELECT schema FROM collections WHERE page_id = ?", collectionID).Scan(&raw); err != nil {
		return nil, store.NoRows(err)
	}
	var schema Schema
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func (s *Service) saveSchema(ctx context.Context, tx *sql.Tx, collectionID string, schema Schema) error {
	b, _ := json.Marshal(schema)
	_, err := tx.ExecContext(ctx, "UPDATE collections SET schema = ?, updated_at = ? WHERE page_id = ?", string(b), store.NowMS(), collectionID)
	return err
}

type relChange struct {
	prop    *Property
	added   []string
	removed []string
}

func diff(before, after []string) (added, removed []string) {
	b := map[string]bool{}
	for _, x := range before {
		b[x] = true
	}
	a := map[string]bool{}
	for _, x := range after {
		a[x] = true
		if !b[x] {
			added = append(added, x)
		}
	}
	for _, x := range before {
		if !a[x] {
			removed = append(removed, x)
		}
	}
	return
}

func readVals(ctx context.Context, q store.Querier, rowID string) (map[string]json.RawMessage, string, error) {
	var raw, col string
	if err := q.QueryRowContext(ctx, "SELECT vals, collection_id FROM collection_rows WHERE page_id = ?", rowID).Scan(&raw, &col); err != nil {
		return nil, "", store.NoRows(err)
	}
	m := map[string]json.RawMessage{}
	_ = json.Unmarshal([]byte(raw), &m)
	return m, col, nil
}

func writeVals(ctx context.Context, tx *sql.Tx, rowID string, m map[string]json.RawMessage) error {
	b, _ := json.Marshal(m)
	_, err := tx.ExecContext(ctx, "UPDATE collection_rows SET vals = ?, version = version + 1 WHERE page_id = ?", string(b), rowID)
	return err
}

func setList(m map[string]json.RawMessage, key string, list []string) {
	if len(list) == 0 {
		delete(m, key)
		return
	}
	m[key] = mustJSON(list)
}

func (s *Service) syncPairs(ctx context.Context, tx *sql.Tx, rowID string, changes []relChange, touched map[string]map[string]bool) error {
	mark := func(col, id string) {
		if touched[col] == nil {
			touched[col] = map[string]bool{}
		}
		touched[col][id] = true
	}
	for _, ch := range changes {
		if ch.prop.Pair == "" {
			continue
		}
		targetSchema, err := s.loadSchema(ctx, tx, ch.prop.Collection)
		if err != nil {
			continue
		}
		pair := targetSchema.ByID(ch.prop.Pair)
		if pair == nil || pair.Type != TRelation {
			continue
		}
		for _, t := range ch.added {
			vals, col, err := readVals(ctx, tx, t)
			if err != nil {
				continue
			}
			list := strList(vals[pair.ID])
			if pair.Limit == 1 {
				for _, prev := range list {
					if prev == rowID {
						continue
					}
					pv, pcol, err := readVals(ctx, tx, prev)
					if err != nil {
						continue
					}
					var kept []string
					for _, x := range strList(pv[ch.prop.ID]) {
						if x != t {
							kept = append(kept, x)
						}
					}
					setList(pv, ch.prop.ID, kept)
					if err := writeVals(ctx, tx, prev, pv); err != nil {
						return err
					}
					mark(pcol, prev)
				}
				list = nil
			}
			found := false
			for _, x := range list {
				if x == rowID {
					found = true
				}
			}
			if !found {
				list = append(list, rowID)
			}
			setList(vals, pair.ID, list)
			if err := writeVals(ctx, tx, t, vals); err != nil {
				return err
			}
			mark(col, t)
		}
		for _, t := range ch.removed {
			vals, col, err := readVals(ctx, tx, t)
			if err != nil {
				continue
			}
			var kept []string
			for _, x := range strList(vals[pair.ID]) {
				if x != rowID {
					kept = append(kept, x)
				}
			}
			setList(vals, pair.ID, kept)
			if err := writeVals(ctx, tx, t, vals); err != nil {
				return err
			}
			mark(col, t)
		}
	}
	return nil
}

func (s *Service) applyValues(ctx context.Context, tx *sql.Tx, schema Schema, nc *normCtx, vals map[string]json.RawMessage, in map[string]json.RawMessage) ([]relChange, *string, error) {
	var changes []relChange
	var title *string
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		raw := in[k]
		p := schema.ByID(k)
		if p == nil {
			p = schema.ByName(k)
		}
		if p == nil {
			return nil, nil, httpx.Fail(http.StatusBadRequest, "bad_property", "there is no property "+k)
		}
		if p.Type == TTitle {
			t := strings.TrimSpace(strings.ReplaceAll(str(raw), "\n", " "))
			t = trimText(t, page.MaxTitle)
			title = &t
			continue
		}
		before := strList(vals[p.ID])
		v, err := nc.normalize(p, raw)
		if err != nil {
			return nil, nil, err
		}
		if v == nil {
			delete(vals, p.ID)
		} else {
			vals[p.ID] = v
		}
		if p.Type == TRelation {
			added, removed := diff(before, strList(vals[p.ID]))
			if len(added)+len(removed) > 0 {
				changes = append(changes, relChange{prop: p, added: added, removed: removed})
			}
		}
	}
	return changes, title, nil
}

func (s *Service) finishWrite(ctx context.Context, collectionID string, touched map[string]map[string]bool, schemaChanged bool) {
	if schemaChanged {
		s.cache.drop(collectionID)
		s.bump()
		s.emit(ctx, collectionID, map[string]any{"what": "schema"})
	}
	for col, set := range touched {
		list := make([]string, 0, len(set))
		for id := range set {
			list = append(list, id)
		}
		sort.Strings(list)
		if err := s.cache.refresh(ctx, s.st.R, col, list); err != nil {
			s.log.Warn("collection: refresh", "err", err)
		}
		s.emitRows(ctx, col, list)
	}
}

func (s *Service) CreateRow(ctx context.Context, p *policy.Principal, collectionID string, in RowInput) (*RowOut, error) {
	m, err := s.writable(ctx, p, collectionID)
	if err != nil {
		return nil, err
	}
	id, err := s.createRow(ctx, p, m, in, p.UserID)
	if err != nil {
		return nil, err
	}
	return s.rowOut(ctx, p, collectionID, id)
}

func (s *Service) createRow(ctx context.Context, p *policy.Principal, m *page.Meta, in RowInput, author string) (string, error) {
	collectionID := m.ID
	nc := s.normCtx(ctx, m.WorkspaceID)
	id := ids.New()
	touched := map[string]map[string]bool{collectionID: {id: true}}
	schemaChanged := false
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		schema, err := s.loadSchema(ctx, tx, collectionID)
		if err != nil {
			return err
		}
		vals := map[string]json.RawMessage{}
		changes, title, err := s.applyValues(ctx, tx, schema, nc, vals, in.Values)
		if err != nil {
			return err
		}
		name := strings.TrimSpace(in.Title)
		if title != nil {
			name = *title
		}
		key, err := s.rowKey(ctx, tx, collectionID, in.BeforeID, in.AfterID, in.Top)
		if err != nil {
			return err
		}
		now := store.NowMS()
		if _, err := tx.ExecContext(ctx, `INSERT INTO pages (id, workspace_id, parent_id, kind, sort_key, title, icon, owner_id, created_at, created_by, updated_at, updated_by, is_row)
			VALUES (?, ?, ?, 'doc', ?, ?, '', ?, ?, ?, ?, ?, 1)`, id, m.WorkspaceID, collectionID, key, trimText(name, page.MaxTitle), nullable(author), now, author, now, author); err != nil {
			return err
		}
		var number int64
		if err := tx.QueryRowContext(ctx, "UPDATE collections SET next_number = next_number + 1 WHERE page_id = ? RETURNING next_number - 1", collectionID).Scan(&number); err != nil {
			return err
		}
		vb, _ := json.Marshal(vals)
		if _, err := tx.ExecContext(ctx, "INSERT INTO collection_rows (page_id, collection_id, vals, number) VALUES (?, ?, ?, ?)", id, collectionID, string(vb), number); err != nil {
			return err
		}
		if err := s.syncPairs(ctx, tx, id, changes, touched); err != nil {
			return err
		}
		if nc.changed {
			schemaChanged = true
			if err := s.saveSchema(ctx, tx, collectionID, schema); err != nil {
				return err
			}
		}
		entry := audit.Entry{Action: "row.create", WorkspaceID: m.WorkspaceID, PageID: id, Detail: collectionID}
		if p == nil {
			entry.ActorKind, entry.ActorLabel = "form", "Public form"
		}
		return s.audit.Record(ctx, tx, p, entry)
	})
	if err != nil {
		return "", err
	}
	s.finishWrite(ctx, collectionID, touched, schemaChanged)
	return id, nil
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Service) rowOut(ctx context.Context, p *policy.Principal, collectionID, id string) (*RowOut, error) {
	res, err := s.Query(ctx, p, collectionID, QueryInput{IDs: []string{id}})
	if err != nil {
		return nil, err
	}
	if len(res.Rows) == 0 {
		return nil, store.ErrNotFound
	}
	return &res.Rows[0], nil
}

func (s *Service) RowCollection(ctx context.Context, rowID string) (string, error) {
	var col string
	err := s.st.R.QueryRowContext(ctx, "SELECT collection_id FROM collection_rows WHERE page_id = ?", rowID).Scan(&col)
	return col, store.NoRows(err)
}

func (s *Service) UpdateRow(ctx context.Context, p *policy.Principal, rowID string, in RowPatch) (*RowOut, error) {
	rm, err := s.pages.Writable(ctx, p, rowID)
	if err != nil {
		return nil, err
	}
	collectionID, err := s.RowCollection(ctx, rowID)
	if err != nil {
		return nil, err
	}
	nc := s.normCtx(ctx, rm.WorkspaceID)
	touched := map[string]map[string]bool{collectionID: {rowID: true}}
	schemaChanged := false
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		schema, err := s.loadSchema(ctx, tx, collectionID)
		if err != nil {
			return err
		}
		vals, _, err := readVals(ctx, tx, rowID)
		if err != nil {
			return err
		}
		changes, title, err := s.applyValues(ctx, tx, schema, nc, vals, in.Values)
		if err != nil {
			return err
		}
		if in.Title != nil {
			t := trimText(strings.TrimSpace(strings.ReplaceAll(*in.Title, "\n", " ")), page.MaxTitle)
			title = &t
		}
		now := store.NowMS()
		sets := []string{"updated_at = ?", "updated_by = ?"}
		args := []any{now, p.UserID}
		if title != nil {
			sets = append(sets, "title = ?")
			args = append(args, *title)
		}
		if in.Move || in.BeforeID != "" || in.AfterID != "" {
			key, err := s.rowKey(ctx, tx, collectionID, in.BeforeID, in.AfterID, false)
			if err != nil {
				return err
			}
			sets = append(sets, "sort_key = ?")
			args = append(args, key)
		}
		args = append(args, rowID)
		if _, err := tx.ExecContext(ctx, "UPDATE pages SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
			return err
		}
		if len(in.Values) > 0 {
			if err := writeVals(ctx, tx, rowID, vals); err != nil {
				return err
			}
		}
		if err := s.syncPairs(ctx, tx, rowID, changes, touched); err != nil {
			return err
		}
		if nc.changed {
			schemaChanged = true
			if err := s.saveSchema(ctx, tx, collectionID, schema); err != nil {
				return err
			}
		}
		var fields []audit.Change
		for k := range in.Values {
			fields = append(fields, audit.Change{Field: k, After: string(vals[k])})
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "row.update", WorkspaceID: rm.WorkspaceID, PageID: rowID, Changes: fields})
	})
	if err != nil {
		return nil, err
	}
	s.finishWrite(ctx, collectionID, touched, schemaChanged)
	if in.Title != nil {
		if m, err := s.pages.Load(ctx, rowID); err == nil {
			owner, _ := s.pages.PrivateOwner(ctx, rowID)
			s.hub.Page(m.WorkspaceID, owner, m)
		}
	}
	return s.rowOut(ctx, p, collectionID, rowID)
}

func (s *Service) DeleteRows(ctx context.Context, p *policy.Principal, collectionID string, list []string) (int, error) {
	if _, err := s.writable(ctx, p, collectionID); err != nil {
		return 0, err
	}
	if len(list) > 5000 {
		return 0, httpx.Fail(http.StatusBadRequest, "too_many", "up to 5,000 rows at a time")
	}
	n := 0
	for _, id := range list {
		col, err := s.RowCollection(ctx, id)
		if err != nil || col != collectionID {
			continue
		}
		if _, err := s.pages.TrashPage(ctx, p, id); err != nil {
			if errors.Is(err, store.ErrNotFound) || errors.Is(err, policy.ErrHidden) {
				continue
			}
			var api *httpx.APIError
			if errors.As(err, &api) && api.Code == "page_trashed" {
				continue
			}
			return n, err
		}
		n++
	}
	return n, nil
}
