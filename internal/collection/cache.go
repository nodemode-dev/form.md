package collection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/store"
)

const maxCached = 64

type Row struct {
	ID        string
	Title     string
	Icon      string
	Cover     string
	Thumb     string
	Snippet   string
	SortKey   string
	Vals      map[string]json.RawMessage
	Number    int64
	Version   int64
	CreatedAt int64
	CreatedBy string
	UpdatedAt int64
	UpdatedBy string
	Template  bool
}

type snap struct {
	id          string
	workspaceID string

	mu       sync.RWMutex
	schema   Schema
	inline   bool
	rows     map[string]*Row
	order    []*Row
	dirty    bool
	programs map[string]*formula.Program
	progErr  map[string]string
	used     time.Time
}

func (s *snap) compile() {
	s.programs = map[string]*formula.Program{}
	s.progErr = map[string]string{}
	for _, p := range s.schema {
		if p.Type != TFormula {
			continue
		}
		if p.Expr == "" {
			continue
		}
		prog, err := formula.Compile(p.Expr, s.schema)
		if err != nil {
			s.progErr[p.ID] = err.Error()
			continue
		}
		s.programs[p.ID] = prog
	}
}

func (s *snap) ordered() []*Row {
	if s.dirty {
		sort.SliceStable(s.order, func(i, j int) bool {
			a, b := s.order[i], s.order[j]
			if a.SortKey != b.SortKey {
				return a.SortKey < b.SortKey
			}
			return a.ID < b.ID
		})
		s.dirty = false
	}
	return s.order
}

func (s *snap) put(r *Row) {
	if old, ok := s.rows[r.ID]; ok {
		for i, x := range s.order {
			if x == old {
				s.order[i] = r
				break
			}
		}
		if old.SortKey != r.SortKey {
			s.dirty = true
		}
	} else {
		s.order = append(s.order, r)
		s.dirty = true
	}
	s.rows[r.ID] = r
}

func (s *snap) drop(id string) {
	old, ok := s.rows[id]
	if !ok {
		return
	}
	delete(s.rows, id)
	for i, x := range s.order {
		if x == old {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
}

type cache struct {
	mu    sync.Mutex
	snaps map[string]*snap
	gen   uint64
}

func (c *cache) bump() {
	c.mu.Lock()
	c.gen++
	c.mu.Unlock()
}

const rowColumns = `p.id, p.title, p.icon, p.cover, p.thumb, p.snippet, p.sort_key, r.vals, r.number, r.version, p.created_at, p.created_by, p.updated_at, p.updated_by, p.is_template`

func scanRow(sc interface{ Scan(...any) error }) (*Row, error) {
	var r Row
	var vals string
	var tpl int
	if err := sc.Scan(&r.ID, &r.Title, &r.Icon, &r.Cover, &r.Thumb, &r.Snippet, &r.SortKey, &vals, &r.Number, &r.Version, &r.CreatedAt, &r.CreatedBy, &r.UpdatedAt, &r.UpdatedBy, &tpl); err != nil {
		return nil, err
	}
	r.Template = tpl == 1
	r.Vals = map[string]json.RawMessage{}
	if vals != "" && vals != "{}" {
		if err := json.Unmarshal([]byte(vals), &r.Vals); err != nil {
			r.Vals = map[string]json.RawMessage{}
		}
	}
	return &r, nil
}

func loadSnap(ctx context.Context, q store.Querier, id string) (*snap, error) {
	var raw string
	var inline int
	s := &snap{id: id, rows: map[string]*Row{}}
	err := q.QueryRowContext(ctx, `SELECT c.workspace_id, c.schema, c.inline FROM collections c WHERE c.page_id = ?`, id).Scan(&s.workspaceID, &raw, &inline)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.inline = inline == 1
	if err := json.Unmarshal([]byte(raw), &s.schema); err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `SELECT `+rowColumns+` FROM collection_rows r JOIN pages p ON p.id = r.page_id
		WHERE r.collection_id = ? AND p.trashed_at = 0 ORDER BY p.sort_key, p.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		s.rows[r.ID] = r
		s.order = append(s.order, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.compile()
	s.used = time.Now()
	return s, nil
}

func (c *cache) get(ctx context.Context, q store.Querier, id string) (*snap, error) {
	var s *snap
	for try := 0; ; try++ {
		c.mu.Lock()
		if have, ok := c.snaps[id]; ok {
			have.used = time.Now()
			c.mu.Unlock()
			return have, nil
		}
		gen := c.gen
		c.mu.Unlock()
		var err error
		if s, err = loadSnap(ctx, q, id); err != nil {
			return nil, err
		}
		c.mu.Lock()
		if c.gen == gen {
			break
		}
		c.mu.Unlock()
		if try == 2 {
			return s, nil
		}
	}
	defer c.mu.Unlock()
	if have, ok := c.snaps[id]; ok {
		return have, nil
	}
	if len(c.snaps) >= maxCached {
		var oldest *snap
		for _, x := range c.snaps {
			if oldest == nil || x.used.Before(oldest.used) {
				oldest = x
			}
		}
		if oldest != nil {
			delete(c.snaps, oldest.id)
		}
	}
	c.snaps[id] = s
	return s, nil
}

func (c *cache) peek(id string) *snap {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snaps[id]
}

func (c *cache) drop(id string) {
	c.mu.Lock()
	delete(c.snaps, id)
	c.gen++
	c.mu.Unlock()
}

func (c *cache) dropAll() {
	c.mu.Lock()
	c.snaps = map[string]*snap{}
	c.gen++
	c.mu.Unlock()
}

func (c *cache) refresh(ctx context.Context, q store.Querier, collectionID string, ids []string) error {
	c.bump()
	s := c.peek(collectionID)
	if s == nil || len(ids) == 0 {
		return nil
	}
	fresh := map[string]*Row{}
	const chunk = 400
	for start := 0; start < len(ids); start += chunk {
		part := ids[start:min(start+chunk, len(ids))]
		args := make([]any, 0, len(part)+1)
		args = append(args, collectionID)
		marks := make([]byte, 0, len(part)*2)
		for i, id := range part {
			if i > 0 {
				marks = append(marks, ',')
			}
			marks = append(marks, '?')
			args = append(args, id)
		}
		rows, err := q.QueryContext(ctx, `SELECT `+rowColumns+` FROM collection_rows r JOIN pages p ON p.id = r.page_id
			WHERE r.collection_id = ? AND p.trashed_at = 0 AND p.id IN (`+string(marks)+`)`, args...)
		if err != nil {
			c.drop(collectionID)
			return err
		}
		for rows.Next() {
			r, err := scanRow(rows)
			if err != nil {
				rows.Close()
				c.drop(collectionID)
				return err
			}
			fresh[r.ID] = r
		}
		rows.Close()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		if r, ok := fresh[id]; ok {
			s.put(r)
		} else {
			s.drop(id)
		}
	}
	s.ordered()
	return nil
}

func (c *cache) setSchema(collectionID string, schema Schema) {
	c.bump()
	s := c.peek(collectionID)
	if s == nil {
		return
	}
	s.mu.Lock()
	s.schema = schema
	s.compile()
	s.mu.Unlock()
}
