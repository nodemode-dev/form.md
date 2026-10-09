package collection

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type SchemaOp struct {
	Op       string    `json:"op"`
	ID       string    `json:"id,omitempty"`
	After    string    `json:"after,omitempty"`
	Prop     *Property `json:"prop,omitempty"`
	TwoWay   *bool     `json:"two_way,omitempty"`
	PairName string    `json:"pair_name,omitempty"`
}

func insertAfter(schema Schema, p *Property, after string) Schema {
	if after == "" {
		return append(schema, p)
	}
	for i, x := range schema {
		if x.ID == after {
			out := append(Schema{}, schema[:i+1]...)
			out = append(out, p)
			return append(out, schema[i+1:]...)
		}
	}
	return append(schema, p)
}

func removeProp(schema Schema, id string) Schema {
	out := Schema{}
	for _, x := range schema {
		if x.ID != id {
			out = append(out, x)
		}
	}
	return out
}

type schemaTx struct {
	s       *Service
	ctx     context.Context
	tx      *sql.Tx
	ws      string
	self    string
	schemas map[string]Schema
	dirty   map[string]bool
	rows    map[string]bool
}

func (st *schemaTx) get(id string) (Schema, error) {
	if sc, ok := st.schemas[id]; ok {
		return sc, nil
	}
	var ws string
	if err := st.tx.QueryRowContext(st.ctx, "SELECT workspace_id FROM collections WHERE page_id = ?", id).Scan(&ws); err != nil || ws != st.ws {
		return nil, badSchema("a relation must point to a database in this workspace")
	}
	sc, err := st.s.loadSchema(st.ctx, st.tx, id)
	if err != nil {
		return nil, err
	}
	st.schemas[id] = sc
	return sc, nil
}

func (st *schemaTx) set(id string, sc Schema) {
	st.schemas[id] = sc
	st.dirty[id] = true
}

func (st *schemaTx) addPair(p *Property, name string) error {
	target, err := st.get(p.Collection)
	if err != nil {
		return err
	}
	if name == "" {
		name = "Related"
		if m, err := st.s.pages.Load(st.ctx, st.self); err == nil && m.Title != "" {
			name = m.Title
		}
	}
	base := name
	for i := 2; target.ByName(name) != nil; i++ {
		name = base + " " + formula.FormatNumber(float64(i))
	}
	pair := &Property{ID: shortID(), Name: name, Type: TRelation, Collection: st.self, Pair: p.ID}
	if err := pair.normalise(); err != nil {
		return err
	}
	pair.Collection, pair.Pair = st.self, p.ID
	p.Pair = pair.ID
	if p.Collection == st.self {
		st.set(st.self, append(st.schemas[st.self], pair))
		return nil
	}
	st.set(p.Collection, append(target, pair))
	return nil
}

func (st *schemaTx) dropPair(p *Property) {
	if p.Pair == "" {
		return
	}
	target, err := st.get(p.Collection)
	if err == nil {
		st.set(p.Collection, removeProp(target, p.Pair))
	}
	p.Pair = ""
}

func (s *Service) UpdateSchema(ctx context.Context, p *policy.Principal, collectionID string, ops []SchemaOp) (Schema, error) {
	m, err := s.structural(ctx, p, collectionID)
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 || len(ops) > 100 {
		return nil, badSchema("send between 1 and 100 changes")
	}
	for _, op := range ops {
		if op.Prop != nil && op.Prop.Autofill != nil && p != nil {
			op.Prop.Autofill.By = p.UserID
		}
	}
	nc := s.normCtx(ctx, m.WorkspaceID)
	var result Schema
	var touchedCols []string
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		st := &schemaTx{s: s, ctx: ctx, tx: tx, ws: m.WorkspaceID, self: collectionID, schemas: map[string]Schema{}, dirty: map[string]bool{}, rows: map[string]bool{}}
		schema, err := s.loadSchema(ctx, tx, collectionID)
		if err != nil {
			return err
		}
		st.schemas[collectionID] = schema
		for _, op := range ops {
			schema = st.schemas[collectionID]
			switch op.Op {
			case "add":
				if op.Prop == nil {
					return badSchema("add needs a property")
				}
				np := *op.Prop
				np.ID = shortID()
				if np.Type == TTitle {
					return badSchema("a database has one title property")
				}
				if err := np.normalise(); err != nil {
					return err
				}
				if schema.ByName(np.Name) != nil {
					base := np.Name
					for i := 2; schema.ByName(np.Name) != nil; i++ {
						np.Name = base + " " + formula.FormatNumber(float64(i))
					}
				}
				if np.Type == TRelation {
					if np.Collection == "" || np.Collection == "self" {
						np.Collection = collectionID
					}
					if np.Collection != collectionID {
						if _, err := st.get(np.Collection); err != nil {
							return err
						}
					}
					np.Pair = ""
				}
				if len(schema) >= maxProps {
					return badSchema("a database can have up to 200 properties")
				}
				st.set(collectionID, insertAfter(schema, &np, op.After))
				if np.Type == TRelation && op.TwoWay != nil && *op.TwoWay {
					if err := st.addPair(&np, op.PairName); err != nil {
						return err
					}
				}
			case "update":
				cur := schema.ByID(op.ID)
				if cur == nil || op.Prop == nil {
					return badSchema("that property was not found")
				}
				next := *op.Prop
				next.ID = cur.ID
				if cur.Type == TTitle {
					name, err := cleanName(next.Name)
					if err != nil {
						return err
					}
					cur.Name, cur.Description = name, next.Description
					st.set(collectionID, schema)
					continue
				}
				if next.Type == TTitle {
					return badSchema("a database has one title property")
				}
				if next.Type == "" {
					next.Type = cur.Type
				}
				if (next.Type == TSelect || next.Type == TMulti || next.Type == TStatus) && next.Options == nil && (cur.Type == TSelect || cur.Type == TMulti || cur.Type == TStatus) {
					next.Options = append([]Option(nil), cur.Options...)
				}
				if err := next.normalise(); err != nil {
					return err
				}
				if other := schema.ByName(next.Name); other != nil && other.ID != cur.ID {
					return badSchema("another property already has that name")
				}
				if next.Type == TRelation {
					if next.Collection == "" || next.Collection == "self" {
						next.Collection = collectionID
					}
					if next.Collection != collectionID {
						if _, err := st.get(next.Collection); err != nil {
							return err
						}
					}
					if cur.Type == TRelation && cur.Collection == next.Collection {
						next.Pair = cur.Pair
					} else {
						next.Pair = ""
						if cur.Type == TRelation {
							st.dropPair(cur)
						}
					}
					if op.TwoWay != nil {
						if *op.TwoWay && next.Pair == "" {
							if err := st.addPair(&next, op.PairName); err != nil {
								return err
							}
						} else if !*op.TwoWay && next.Pair != "" {
							tmp := next
							st.dropPair(&tmp)
							next.Pair = ""
						}
					}
				} else if cur.Type == TRelation {
					st.dropPair(cur)
				}
				if next.Type != cur.Type || (next.Type == TRelation && next.Collection != cur.Collection) {
					if err := s.convert(ctx, tx, collectionID, cur, &next, nc, st); err != nil {
						return err
					}
				} else if len(cur.Options) > 0 {
					if err := s.pruneOptions(ctx, tx, collectionID, cur, &next); err != nil {
						return err
					}
				}
				schema = st.schemas[collectionID]
				for i, x := range schema {
					if x.ID == cur.ID {
						c := next
						schema[i] = &c
					}
				}
				st.set(collectionID, schema)
			case "delete":
				cur := schema.ByID(op.ID)
				if cur == nil {
					continue
				}
				if cur.Type == TTitle {
					return badSchema("the title property cannot be deleted")
				}
				if cur.Type == TRelation {
					st.dropPair(cur)
				}
				st.set(collectionID, removeProp(st.schemas[collectionID], cur.ID))
			case "move":
				cur := schema.ByID(op.ID)
				if cur == nil {
					continue
				}
				rest := removeProp(schema, cur.ID)
				if op.After == "" {
					st.set(collectionID, append(Schema{cur}, rest...))
				} else {
					st.set(collectionID, insertAfter(rest, cur, op.After))
				}
			default:
				return badSchema("unknown change " + op.Op)
			}
		}
		final := st.schemas[collectionID]
		for _, x := range final {
			if x.Type == TFormula && x.Expr != "" {
				if _, err := formula.Compile(x.Expr, final); err != nil {
					if fe, ok := err.(*formula.Error); ok {
						return httpx.Fail(http.StatusBadRequest, "bad_formula", x.Name+": "+fe.Msg).With("prop", x.ID).With("pos", fe.Pos)
					}
					return err
				}
			}
		}
		for id := range st.dirty {
			if err := s.saveSchema(ctx, tx, id, st.schemas[id]); err != nil {
				return err
			}
			if id != collectionID {
				touchedCols = append(touchedCols, id)
			}
		}
		result = final
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "collection.schema", WorkspaceID: m.WorkspaceID, PageID: collectionID})
	})
	if err != nil {
		return nil, err
	}
	s.finishWrite(ctx, collectionID, nil, true)
	for _, id := range touchedCols {
		s.cache.drop(id)
		s.emit(ctx, id, map[string]any{"what": "schema"})
	}
	return result, nil
}

func (s *Service) pruneOptions(ctx context.Context, tx *sql.Tx, collectionID string, cur, next *Property) error {
	keep := map[string]bool{}
	for _, o := range next.Options {
		keep[o.ID] = true
	}
	gone := false
	for _, o := range cur.Options {
		if !keep[o.ID] {
			gone = true
		}
	}
	if !gone {
		return nil
	}
	return s.eachRow(ctx, tx, collectionID, func(id string, vals map[string]json.RawMessage) (bool, error) {
		raw, ok := vals[cur.ID]
		if !ok {
			return false, nil
		}
		var kept []string
		for _, x := range strList(raw) {
			if keep[x] {
				kept = append(kept, x)
			}
		}
		if next.Type == TMulti {
			setList(vals, cur.ID, kept)
		} else if len(kept) == 0 {
			delete(vals, cur.ID)
		} else {
			vals[cur.ID] = mustJSON(kept[0])
		}
		return true, nil
	})
}

func (s *Service) eachRow(ctx context.Context, tx *sql.Tx, collectionID string, fn func(id string, vals map[string]json.RawMessage) (bool, error)) error {
	rows, err := tx.QueryContext(ctx, "SELECT page_id, vals FROM collection_rows WHERE collection_id = ?", collectionID)
	if err != nil {
		return err
	}
	type item struct {
		id   string
		vals map[string]json.RawMessage
	}
	var items []item
	for rows.Next() {
		var id, raw string
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		m := map[string]json.RawMessage{}
		_ = json.Unmarshal([]byte(raw), &m)
		items = append(items, item{id, m})
	}
	rows.Close()
	for _, it := range items {
		changed, err := fn(it.id, it.vals)
		if err != nil {
			return err
		}
		if changed {
			if err := writeVals(ctx, tx, it.id, it.vals); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) displayText(ctx context.Context, q store.Querier, p *Property, raw json.RawMessage) string {
	switch p.Type {
	case TText, TURL, TEmail, TPhone, TTitle, TNumber:
		return str(raw)
	case TSelect, TStatus:
		if o := p.Option(str(raw)); o != nil {
			return o.Name
		}
	case TMulti:
		var names []string
		for _, id := range strList(raw) {
			if o := p.Option(id); o != nil {
				names = append(names, o.Name)
			}
		}
		return strings.Join(names, ", ")
	case TDate:
		var d DateVal
		if json.Unmarshal(raw, &d) == nil {
			return d.Start
		}
	case TCheckbox:
		var b bool
		if json.Unmarshal(raw, &b) == nil && b {
			return "Yes"
		}
	case TPerson:
		var names []string
		for _, id := range strList(raw) {
			var n string
			if q.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", id).Scan(&n) == nil {
				names = append(names, n)
			}
		}
		return strings.Join(names, ", ")
	case TRelation:
		var names []string
		for _, id := range strList(raw) {
			var n string
			if q.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = ?", id).Scan(&n) == nil {
				names = append(names, n)
			}
		}
		return strings.Join(names, ", ")
	case TFiles:
		var files []FileVal
		_ = json.Unmarshal(raw, &files)
		var names []string
		for _, f := range files {
			names = append(names, f.Name)
		}
		return strings.Join(names, ", ")
	case TPlace:
		var pl PlaceVal
		if json.Unmarshal(raw, &pl) == nil {
			if pl.Name != "" {
				return pl.Name
			}
			return pl.Address
		}
	}
	return ""
}

func (s *Service) convert(ctx context.Context, tx *sql.Tx, collectionID string, cur, next *Property, nc *normCtx, st *schemaTx) error {
	if computed(next.Type) {
		return s.eachRow(ctx, tx, collectionID, func(_ string, vals map[string]json.RawMessage) (bool, error) {
			if _, ok := vals[cur.ID]; !ok {
				return false, nil
			}
			delete(vals, cur.ID)
			return true, nil
		})
	}
	if (next.Type == TSelect || next.Type == TMulti || next.Type == TStatus) && (cur.Type == TSelect || cur.Type == TMulti || cur.Type == TStatus) && len(next.Options) == 0 {
		next.Options = append([]Option(nil), cur.Options...)
		for i := range next.Options {
			if next.Type == TStatus && next.Options[i].Group == "" {
				next.Options[i].Group = "todo"
			}
			if next.Type != TStatus {
				next.Options[i].Group = ""
			}
		}
	}
	people := map[string]string{}
	if next.Type == TPerson {
		rows, err := tx.QueryContext(ctx, `SELECT u.id, u.name, u.email FROM users u JOIN workspace_members m ON m.user_id = u.id
			JOIN collections c ON c.workspace_id = m.workspace_id WHERE c.page_id = ?`, collectionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id, name, email string
			if rows.Scan(&id, &name, &email) == nil {
				people[strings.ToLower(name)] = id
				people[strings.ToLower(email)] = id
			}
		}
		rows.Close()
	}
	err := s.eachRow(ctx, tx, collectionID, func(_ string, vals map[string]json.RawMessage) (bool, error) {
		raw, ok := vals[cur.ID]
		if !ok {
			return false, nil
		}
		delete(vals, cur.ID)
		if cur.Type == next.Type && cur.Type != TRelation {
			vals[cur.ID] = raw
			return true, nil
		}
		text := strings.TrimSpace(s.displayText(ctx, tx, cur, raw))
		if text == "" {
			return true, nil
		}
		var in json.RawMessage
		switch next.Type {
		case TText, TURL, TEmail, TPhone, TNumber:
			in = mustJSON(text)
		case TSelect, TStatus:
			first, _, _ := strings.Cut(text, ",")
			in = mustJSON(map[string]string{"new": strings.TrimSpace(first)})
		case TMulti:
			var items []map[string]string
			for _, part := range strings.Split(text, ",") {
				if t := strings.TrimSpace(part); t != "" {
					items = append(items, map[string]string{"new": t})
				}
			}
			in = mustJSON(items)
		case TDate:
			t, has, ok := formula.ParseDate(text, time.UTC)
			if !ok {
				return true, nil
			}
			in = mustJSON(DateVal{Start: dateString(t, has)})
		case TCheckbox:
			low := strings.ToLower(text)
			in = mustJSON(low == "yes" || low == "true" || low == "1" || low == "x" || low == "checked" || low == "done")
		case TPerson:
			var list []string
			for _, part := range strings.Split(text, ",") {
				if id, ok := people[strings.ToLower(strings.TrimSpace(part))]; ok {
					list = append(list, id)
				}
			}
			in = mustJSON(list)
		default:
			return true, nil
		}
		v, err := nc.normalize(next, in)
		if err != nil {
			return true, nil
		}
		if v != nil {
			vals[cur.ID] = v
		}
		return true, nil
	})
	return err
}
