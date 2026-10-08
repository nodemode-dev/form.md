package collection

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/order"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	maxCSVBytes = 25 << 20
	maxCSVRows  = 100000
	maxCSVCols  = 200
	stashTTL    = 20 * time.Minute
)

type stashed struct {
	user    string
	headers []string
	rows    [][]string
	at      time.Time
	name    string
}

type stash struct {
	mu    sync.Mutex
	items map[string]*stashed
}

func newStash() *stash { return &stash{items: map[string]*stashed{}} }

func (s *stash) put(x *stashed) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.items {
		if time.Since(v.at) > stashTTL {
			delete(s.items, k)
		}
	}
	if len(s.items) >= 16 {
		var oldest string
		for k, v := range s.items {
			if oldest == "" || v.at.Before(s.items[oldest].at) {
				oldest = k
			}
		}
		delete(s.items, oldest)
	}
	k := ids.New()
	s.items[k] = x
	return k
}

func (s *stash) take(key, user string) *stashed {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.items[key]
	if !ok || x.user != user || time.Since(x.at) > stashTTL {
		return nil
	}
	delete(s.items, key)
	return x
}

func (s *stash) peek(key, user string) *stashed {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.items[key]
	if !ok || x.user != user {
		return nil
	}
	return x
}

func sniffDelimiter(head []byte) rune {
	line, _, _ := bytes.Cut(head, []byte("\n"))
	best, count := ',', bytes.Count(line, []byte(","))
	for _, d := range []rune{';', '\t', '|'} {
		if n := bytes.Count(line, []byte(string(d))); n > count {
			best, count = d, n
		}
	}
	return best
}

func readCSV(r io.Reader) ([]string, [][]string, error) {
	br := bufio.NewReaderSize(io.LimitReader(r, maxCSVBytes+1), 64<<10)
	head, _ := br.Peek(8192)
	head = bytes.TrimPrefix(head, []byte("\xef\xbb\xbf"))
	if len(head) > 0 && !utf8.Valid(head[:min(len(head), 4096)]) && !utf8.Valid(head[:min(len(head), 4093)]) {
		return nil, nil, httpx.Fail(http.StatusBadRequest, "bad_csv", "the file is not UTF-8 text")
	}
	if bom, _ := br.Peek(3); bytes.Equal(bom, []byte("\xef\xbb\xbf")) {
		_, _ = br.Discard(3)
	}
	cr := csv.NewReader(br)
	cr.Comma = sniffDelimiter(head)
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = false
	headers, err := cr.Read()
	if err != nil {
		return nil, nil, httpx.Fail(http.StatusBadRequest, "bad_csv", "the file has no header row")
	}
	if len(headers) > maxCSVCols {
		return nil, nil, httpx.Fail(http.StatusBadRequest, "bad_csv", "up to 200 columns")
	}
	var rows [][]string
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, httpx.Fail(http.StatusBadRequest, "bad_csv", "the file could not be read: "+err.Error())
		}
		if len(rows) >= maxCSVRows {
			return nil, nil, httpx.Fail(http.StatusBadRequest, "bad_csv", "up to 100,000 rows")
		}
		empty := true
		for _, f := range rec {
			if strings.TrimSpace(f) != "" {
				empty = false
				break
			}
		}
		if !empty {
			rows = append(rows, rec)
		}
	}
	seen := map[string]int{}
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if h == "" {
			h = "Column " + strconv.Itoa(i+1)
		}
		if n := seen[strings.ToLower(h)]; n > 0 {
			seen[strings.ToLower(h)] = n + 1
			h = h + " " + strconv.Itoa(n+1)
		} else {
			seen[strings.ToLower(h)] = 1
		}
		headers[i] = trimText(h, maxName)
	}
	return headers, rows, nil
}

type ColumnGuess struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Samples []string `json:"samples"`
}

type Preview struct {
	Token   string        `json:"token"`
	Name    string        `json:"name"`
	Rows    int           `json:"rows"`
	Columns []ColumnGuess `json:"columns"`
}

func truthyWord(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yes", "true", "x", "checked", "1", "done", "y":
		return true, true
	case "no", "false", "", "unchecked", "0", "n":
		return false, true
	}
	return false, false
}

func guessType(values []string, first bool) string {
	if first {
		return TTitle
	}
	n, nums, bools, dates, urls, emails := 0, 0, 0, 0, 0, 0
	distinct := map[string]int{}
	commas := 0
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		n++
		if _, ok := formula.Text(v).ToNumber(); ok {
			nums++
		}
		if _, ok := truthyWord(v); ok {
			bools++
		}
		if _, _, ok := formula.ParseDate(v, time.UTC); ok {
			dates++
		}
		if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
			urls++
		}
		if strings.Contains(v, "@") {
			if _, err := mail.ParseAddress(v); err == nil {
				emails++
			}
		}
		if strings.Contains(v, ",") {
			commas++
		}
		distinct[strings.ToLower(v)]++
	}
	switch {
	case n == 0:
		return TText
	case nums == n && bools < n:
		return TNumber
	case bools == n:
		return TCheckbox
	case dates == n:
		return TDate
	case urls == n:
		return TURL
	case emails == n:
		return TEmail
	}
	if commas > 0 && commas*2 >= n {
		tokens := map[string]int{}
		for _, v := range values {
			for _, part := range strings.Split(v, ",") {
				if t := strings.ToLower(strings.TrimSpace(part)); t != "" {
					tokens[t]++
				}
			}
		}
		if len(tokens) <= 60 {
			return TMulti
		}
	}
	if len(distinct) <= 20 && n >= 4 && len(distinct)*2 <= n {
		return TSelect
	}
	return TText
}

func (s *Service) PreviewCSV(ctx context.Context, p *policy.Principal, name string, r io.Reader) (*Preview, error) {
	headers, rows, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	out := &Preview{Rows: len(rows), Name: strings.TrimSuffix(strings.TrimSuffix(name, ".csv"), ".tsv")}
	for i, h := range headers {
		col := make([]string, 0, min(len(rows), 2000))
		for j, rec := range rows {
			if j >= 2000 {
				break
			}
			if i < len(rec) {
				col = append(col, rec[i])
			}
		}
		g := ColumnGuess{Name: h, Type: guessType(col, i == 0), Samples: []string{}}
		for _, v := range col {
			if strings.TrimSpace(v) != "" {
				g.Samples = append(g.Samples, trimText(v, 80))
				if len(g.Samples) == 3 {
					break
				}
			}
		}
		out.Columns = append(out.Columns, g)
	}
	out.Token = s.stash.put(&stashed{user: p.UserID, headers: headers, rows: rows, at: time.Now(), name: out.Name})
	return out, nil
}

type ImportInput struct {
	Token        string   `json:"token"`
	Types        []string `json:"types"`
	Title        string   `json:"title"`
	WorkspaceID  string   `json:"workspace_id"`
	ParentID     string   `json:"parent_id"`
	CollectionID string   `json:"collection_id"`
}

func (s *Service) ImportCSV(ctx context.Context, p *policy.Principal, in ImportInput) (string, int, error) {
	x := s.stash.peek(in.Token, p.UserID)
	if x == nil {
		return "", 0, httpx.Fail(http.StatusGone, "expired", "the uploaded file expired; upload it again")
	}
	if len(in.Types) != len(x.headers) {
		return "", 0, httpx.Fail(http.StatusBadRequest, "bad_types", "send one type per column")
	}
	var collectionID string
	var schema Schema
	cols := make([]*Property, len(x.headers))
	if in.CollectionID != "" {
		m, err := s.writable(ctx, p, in.CollectionID)
		if err != nil {
			return "", 0, err
		}
		collectionID = m.ID
		if schema, err = s.loadSchema(ctx, s.st.R, collectionID); err != nil {
			return "", 0, err
		}
		var add []SchemaOp
		for i, h := range x.headers {
			if in.Types[i] == "skip" {
				continue
			}
			if i == 0 && in.Types[i] == TTitle {
				continue
			}
			if schema.ByName(h) == nil {
				t := in.Types[i]
				if !types[t] || computed(t) || t == TTitle || t == TRelation {
					t = TText
				}
				add = append(add, SchemaOp{Op: "add", Prop: &Property{Name: h, Type: t}})
			}
		}
		if len(add) > 0 {
			if schema, err = s.UpdateSchema(ctx, p, collectionID, add); err != nil {
				return "", 0, err
			}
		}
		for i, h := range x.headers {
			if in.Types[i] == "skip" {
				continue
			}
			if i == 0 && in.Types[i] == TTitle {
				cols[i] = schema.ByID(TTitle)
				continue
			}
			cols[i] = schema.ByName(h)
		}
	} else {
		title := strings.TrimSpace(in.Title)
		if title == "" {
			title = x.name
		}
		var props Schema
		titleSet := false
		for i, h := range x.headers {
			t := in.Types[i]
			if t == "skip" {
				continue
			}
			if t == TTitle && !titleSet {
				titleSet = true
				props = append(props, &Property{ID: TTitle, Name: h, Type: TTitle})
				continue
			}
			if !types[t] || computed(t) || t == TTitle || t == TRelation {
				t = TText
			}
			props = append(props, &Property{Name: h, Type: t})
		}
		extra, _ := json.Marshal(createExtra{Schema: props})
		m, err := s.pages.Create(ctx, p, page.CreateInput{WorkspaceID: in.WorkspaceID, ParentID: in.ParentID, Title: title, Kind: "collection", Extra: extra})
		if err != nil {
			return "", 0, err
		}
		collectionID = m.ID
		if schema, err = s.loadSchema(ctx, s.st.R, collectionID); err != nil {
			return "", 0, err
		}
		pi := 0
		for i := range x.headers {
			if in.Types[i] == "skip" {
				continue
			}
			for pi < len(schema) && schema[pi].Type == TTitle && in.Types[i] != TTitle {
				pi++
			}
			if pi < len(schema) {
				cols[i] = schema[pi]
				pi++
			}
		}
	}
	if s.stash.take(in.Token, p.UserID) == nil {
		return "", 0, httpx.Fail(http.StatusGone, "expired", "the uploaded file expired; upload it again")
	}
	m, err := s.pages.Load(ctx, collectionID)
	if err != nil {
		return "", 0, err
	}
	nc := s.normCtx(ctx, m.WorkspaceID)
	nc.targets = nil
	members := map[string]string{}
	if rows, err := s.st.R.QueryContext(ctx, `SELECT u.id, u.name, u.email FROM users u JOIN workspace_members w ON w.user_id = u.id WHERE w.workspace_id = ?`, m.WorkspaceID); err == nil {
		for rows.Next() {
			var id, name, email string
			if rows.Scan(&id, &name, &email) == nil {
				members[strings.ToLower(name)] = id
				members[strings.ToLower(email)] = id
			}
		}
		rows.Close()
	}
	count := 0
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		var last sql.NullString
		if err := tx.QueryRowContext(ctx, "SELECT MAX(sort_key) FROM pages WHERE parent_id = ? AND is_row = 1", collectionID).Scan(&last); err != nil {
			return err
		}
		keys := order.Sequence(last.String, len(x.rows))
		now := store.NowMS()
		insPage, err := tx.PrepareContext(ctx, `INSERT INTO pages (id, workspace_id, parent_id, kind, sort_key, title, icon, owner_id, created_at, created_by, updated_at, updated_by, is_row)
			VALUES (?, ?, ?, 'doc', ?, ?, '', ?, ?, ?, ?, ?, 1)`)
		if err != nil {
			return err
		}
		defer insPage.Close()
		insRow, err := tx.PrepareContext(ctx, "INSERT INTO collection_rows (page_id, collection_id, vals, number) VALUES (?, ?, ?, ?)")
		if err != nil {
			return err
		}
		defer insRow.Close()
		var next int64
		if err := tx.QueryRowContext(ctx, "SELECT next_number FROM collections WHERE page_id = ?", collectionID).Scan(&next); err != nil {
			return err
		}
		for n, rec := range x.rows {
			vals := map[string]json.RawMessage{}
			title := ""
			for i, prop := range cols {
				if prop == nil || i >= len(rec) {
					continue
				}
				cell := strings.TrimSpace(rec[i])
				if cell == "" {
					continue
				}
				if prop.Type == TTitle {
					title = trimText(cell, page.MaxTitle)
					continue
				}
				var in json.RawMessage
				switch prop.Type {
				case TSelect, TStatus:
					in = mustJSON(map[string]string{"new": cell})
				case TMulti:
					var items []map[string]string
					for _, part := range strings.Split(cell, ",") {
						if t := strings.TrimSpace(part); t != "" {
							items = append(items, map[string]string{"new": t})
						}
					}
					in = mustJSON(items)
				case TDate:
					t, has, ok := formula.ParseDate(cell, time.UTC)
					if !ok {
						continue
					}
					in = mustJSON(DateVal{Start: dateString(t, has)})
				case TCheckbox:
					b, _ := truthyWord(cell)
					in = mustJSON(b)
				case TPerson:
					var list []string
					for _, part := range strings.Split(cell, ",") {
						if id, ok := members[strings.ToLower(strings.TrimSpace(part))]; ok {
							list = append(list, id)
						}
					}
					in = mustJSON(list)
				case TFiles:
					var files []FileVal
					for _, part := range strings.Split(cell, ",") {
						if u := strings.TrimSpace(part); strings.HasPrefix(u, "http") {
							files = append(files, FileVal{URL: u, Name: u})
						}
					}
					in = mustJSON(files)
				default:
					in = mustJSON(cell)
				}
				v, err := nc.normalize(prop, in)
				if err != nil || v == nil {
					continue
				}
				vals[prop.ID] = v
			}
			id := ids.New()
			if _, err := insPage.ExecContext(ctx, id, m.WorkspaceID, collectionID, keys[n], title, p.UserID, now, p.UserID, now, p.UserID); err != nil {
				return err
			}
			vb, _ := json.Marshal(vals)
			if _, err := insRow.ExecContext(ctx, id, collectionID, string(vb), next); err != nil {
				return err
			}
			next++
			count++
		}
		if _, err := tx.ExecContext(ctx, "UPDATE collections SET next_number = ? WHERE page_id = ?", next, collectionID); err != nil {
			return err
		}
		if nc.changed {
			if err := s.saveSchema(ctx, tx, collectionID, schema); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "collection.import_csv", WorkspaceID: m.WorkspaceID, PageID: collectionID, Detail: strconv.Itoa(count)})
	})
	if err != nil {
		return "", 0, err
	}
	s.cache.drop(collectionID)
	s.bump()
	s.emit(ctx, collectionID, map[string]any{"what": "schema"})
	s.emit(ctx, collectionID, map[string]any{"what": "rows", "ids": []string{"*"}})
	return collectionID, count, nil
}

func (ev *evaluator) displayCell(sn *snap, r *Row, p *Property) string {
	switch p.Type {
	case TCheckbox:
		if ev.value(sn, r, p).Truthy() {
			return "Yes"
		}
		return "No"
	case TFiles:
		var files []FileVal
		_ = json.Unmarshal(r.Vals[p.ID], &files)
		parts := make([]string, 0, len(files))
		for _, f := range files {
			if f.URL != "" {
				parts = append(parts, f.URL)
			} else {
				parts = append(parts, f.Name)
			}
		}
		return strings.Join(parts, ", ")
	case TDate:
		var d DateVal
		if json.Unmarshal(r.Vals[p.ID], &d) != nil {
			return ""
		}
		if d.End != "" {
			return d.Start + " - " + d.End
		}
		return d.Start
	case TUniqueID:
		if p.Prefix != "" {
			return p.Prefix + "-" + strconv.FormatInt(r.Number, 10)
		}
		return strconv.FormatInt(r.Number, 10)
	}
	v := ev.value(sn, r, p)
	if v.K == formula.KDate {
		if v.HasTime {
			s := v.T.Format("2006-01-02 15:04")
			if !v.End.IsZero() && !v.End.Equal(v.T) {
				s += " - " + v.End.Format("2006-01-02 15:04")
			}
			return s
		}
		s := v.T.Format("2006-01-02")
		if !v.End.IsZero() && !v.End.Equal(v.T) {
			s += " - " + v.End.Format("2006-01-02")
		}
		return s
	}
	return v.String()
}

func (s *Service) ExportCSV(ctx context.Context, p *policy.Principal, viewID string, w io.Writer) (string, error) {
	v, err := s.view(ctx, viewID)
	if err != nil {
		return "", err
	}
	m, _, err := s.readable(ctx, p, v.CollectionID)
	if err != nil {
		return "", err
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn := ev.snap(v.CollectionID)
	if sn == nil {
		return "", store.ErrNotFound
	}
	rows := s.results.ordered(ev, sn, v.Config, QueryInput{})
	hidden := map[string]bool{}
	order := []*Property{}
	placed := map[string]bool{}
	for _, vp := range v.Config.Props {
		if vp.Hidden {
			hidden[vp.ID] = true
		}
		if p := sn.schema.ByID(vp.ID); p != nil && !placed[p.ID] {
			order = append(order, p)
			placed[p.ID] = true
		}
	}
	for _, p := range sn.schema {
		if !placed[p.ID] {
			order = append(order, p)
		}
	}
	var props []*Property
	for _, p := range order {
		if p.Type == TTitle {
			props = append([]*Property{p}, props...)
			continue
		}
		if !hidden[p.ID] && p.Type != TButton {
			props = append(props, p)
		}
	}
	_, _ = w.Write([]byte("\xef\xbb\xbf"))
	cw := csv.NewWriter(w)
	head := make([]string, len(props))
	for i, p := range props {
		head[i] = p.Name
	}
	if err := cw.Write(head); err != nil {
		return "", err
	}
	rec := make([]string, len(props))
	for _, r := range rows {
		for i, p := range props {
			if p.Type == TTitle {
				rec[i] = r.Title
				continue
			}
			rec[i] = ev.displayCell(sn, r, p)
		}
		if err := cw.Write(rec); err != nil {
			return "", err
		}
	}
	cw.Flush()
	name := m.Title
	if name == "" {
		name = "Untitled"
	}
	return name + " - " + v.Name, cw.Error()
}
