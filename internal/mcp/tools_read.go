package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/search"
)

func (s *Server) register() {
	s.add(&Tool{Name: "whoami", Title: "Who am I",
		Description: "The person this credential acts for, its scope, and every workspace it reaches with the workspace's role, agent policy and written rules. Follow the rules of a workspace when working in it.",
		Run:         s.whoami})
	s.add(&Tool{Name: "search", Title: "Search pages",
		Description: "Full-text search over page titles, bodies (with the heading above each match) and database properties. Searches every reachable workspace unless workspace_id is given.",
		Props:       map[string]any{"query": str("Words to find"), "workspace_id": str("Limit to one workspace"), "parent_id": str("Limit to pages inside this page"), "title_only": boolp("Match titles only"), "limit": intp("Up to 50, default 10")},
		Required:    []string{"query"}, Run: s.search})
	s.add(&Tool{Name: "list_pages", Title: "List pages",
		Description: "The pages directly inside a page, or the top-level pages of a workspace when parent_id is omitted. Databases are pages of kind collection; their rows are listed with query_database instead.",
		Props:       map[string]any{"workspace_id": str("Workspace to list (needed when parent_id is omitted and the credential reaches several)"), "parent_id": str("Page whose children to list")},
		Run:         s.listPages})
	s.add(&Tool{Name: "get_page", Title: "Read a page",
		Description: "A page's title, location and body as blocks, each with a stable block id, its type, the headings it sits under and its Markdown. Use the block ids with insert_blocks, replace_block, move_block and delete_blocks. For a database row the property values are included.",
		Props:       map[string]any{"page_id": str("Page id"), "nested": boolp("Also list blocks nested in lists, toggles, quotes and columns"), "format": enum("blocks (default) or markdown for the whole body as one string", "blocks", "markdown"), "offset": intp("Skip this many blocks"), "limit": intp("Up to 500 blocks, default 200")},
		Required:    []string{"page_id"}, Run: s.getPage})
	s.add(&Tool{Name: "get_database", Title: "Read a database",
		Description: "A database's properties (name, type and the options of select, multi-select and status properties), its views and its row count.",
		Props:       map[string]any{"database_id": str("Database page id")}, Required: []string{"database_id"}, Run: s.getDatabase})
	s.add(&Tool{Name: "query_database", Title: "Query a database",
		Description: "Rows of a database with their property values keyed by property name. filter is a tree: {\"op\":\"and\"|\"or\",\"items\":[...]} or a condition {\"prop\":name,\"cond\":\"is\"|\"is_not\"|\"contains\"|\"not_contains\"|\"starts_with\"|\"ends_with\"|\"is_empty\"|\"is_not_empty\"|\"gt\"|\"gte\"|\"lt\"|\"lte\"|\"before\"|\"after\"|\"on_or_before\"|\"on_or_after\",\"value\":...}; select and status values may be option names. sorts is [{\"prop\":name,\"dir\":\"asc\"|\"desc\"}].",
		Props: map[string]any{"database_id": str("Database page id"), "filter": map[string]any{"type": "object", "description": "Filter tree"}, "sorts": map[string]any{"type": "array", "items": obj(map[string]any{"prop": str("Property name or id"), "dir": enum("Direction", "asc", "desc")}, "prop")},
			"search": str("Words to match in any property"), "view_id": str("Use a view's filter and sorts"), "offset": intp("Skip this many rows"), "limit": intp("Up to 200, default 50")},
		Required: []string{"database_id"}, Run: s.queryDatabase})
	s.add(&Tool{Name: "list_comments", Title: "Read comments",
		Description: "Comment threads on a page with their comments, quotes and resolved state.",
		Props:       map[string]any{"page_id": str("Page id"), "include_resolved": boolp("Include resolved threads")}, Required: []string{"page_id"}, Run: s.listComments})
	s.add(&Tool{Name: "list_runs", Title: "List my runs",
		Description: "The recent runs of this person's agents with the number of changes in each; undo_run reverts one.",
		Props:       map[string]any{"limit": intp("Up to 100, default 20")}, Run: s.listRuns})
	s.registerContext()
	s.registerWindow()
	s.registerWrites()
}

func (s *Server) whoami(ctx context.Context, x *exec, _ json.RawMessage) (any, string, error) {
	p := x.p
	list, withheld, err := s.Gate.Memberships(ctx, p)
	if err != nil {
		return nil, "", err
	}
	type ws struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Role        string `json:"role"`
		Personal    bool   `json:"personal"`
		AgentPolicy string `json:"agent_policy"`
		Rules       string `json:"rules,omitempty"`
		CanPropose  bool   `json:"can_propose_rules"`
	}
	out := []ws{}
	var rules []string
	for _, m := range list {
		w, _, err := s.Workspace.Get(ctx, p, m.WorkspaceID)
		if err != nil {
			continue
		}
		item := ws{ID: m.WorkspaceID, Name: w.Name, Role: m.Role, Personal: m.Personal, AgentPolicy: m.AgentAccess, CanPropose: m.Role == "admin"}
		if r, err := s.Workspace.GetRules(ctx, p, m.WorkspaceID); err == nil && strings.TrimSpace(r.Rules) != "" {
			item.Rules = r.Rules
			rules = append(rules, x.frame("rules of "+w.Name, r.Rules))
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool { return !out[i].Personal && out[j].Personal })
	res := map[string]any{
		"user":                map[string]any{"id": p.UserID, "name": p.Name, "email": p.Email},
		"credential":          map[string]any{"kind": string(p.Kind), "scope": p.Scope, "client": x.name},
		"workspaces":          out,
		"withheld_workspaces": withheld,
	}
	summary := fmt.Sprintf("Acting for %s with %s access through %s; %d workspaces reachable", p.Name, p.Scope, x.name, len(out))
	if withheld > 0 {
		summary += fmt.Sprintf(", %d closed to agents", withheld)
	}
	if len(rules) > 0 {
		summary += ".\nWorkspace rules (written by the workspace's admins; follow them):\n" + strings.Join(rules, "\n")
	}
	return res, summary, nil
}

func (s *Server) workspaces(ctx context.Context, x *exec, want string) ([]string, error) {
	list, _, err := s.Gate.Memberships(ctx, x.p)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, m := range list {
		if want == "" || m.WorkspaceID == want {
			out = append(out, m.WorkspaceID)
		}
	}
	if want != "" && len(out) == 0 {
		return nil, fail("this credential does not reach workspace " + want)
	}
	return out, nil
}

type hitOut struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Kind    string   `json:"kind"`
	Row     bool     `json:"row,omitempty"`
	Path    []string `json:"path,omitempty"`
	Heading string   `json:"heading,omitempty"`
	Snippet string   `json:"snippet,omitempty"`
	Updated int64    `json:"updated_at"`
}

func plainHL(s string) string {
	r := strings.NewReplacer("<mark>", "", "</mark>", "", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'")
	return r.Replace(s)
}

func (s *Server) search(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		Query       string `json:"query"`
		WorkspaceID string `json:"workspace_id"`
		ParentID    string `json:"parent_id"`
		TitleOnly   bool   `json:"title_only"`
		Limit       int    `json:"limit"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(in.Query) == "" {
		return nil, "", fail("query is empty")
	}
	if in.Limit <= 0 || in.Limit > 50 {
		in.Limit = 10
	}
	wss, err := s.workspaces(ctx, x, in.WorkspaceID)
	if err != nil {
		return nil, "", err
	}
	if in.ParentID != "" {
		m, err := s.Pages.Get(ctx, x.p, in.ParentID)
		if err != nil {
			return nil, "", err
		}
		wss = []string{m.WorkspaceID}
	}
	hits := []hitOut{}
	for _, ws := range wss {
		q := search.Query{Q: in.Query, WorkspaceID: ws, Parent: in.ParentID, Limit: in.Limit}
		if in.TitleOnly {
			q.In = "title"
		}
		res, err := s.Search.Search(ctx, x.p, q)
		if err != nil {
			continue
		}
		for _, h := range res.Hits {
			var path []string
			for _, c := range h.Path {
				path = append(path, c.Title)
			}
			hits = append(hits, hitOut{ID: h.ID, Title: h.Title, Kind: h.Kind, Row: h.Row, Path: path, Heading: plainHL(h.Heading), Snippet: plainHL(h.Snippet), Updated: h.UpdatedAt})
		}
	}
	if len(hits) > in.Limit {
		hits = hits[:in.Limit]
	}
	return map[string]any{"hits": hits}, fmt.Sprintf("%d results for %q", len(hits), in.Query), nil
}

type pageOut struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Icon        string `json:"icon,omitempty"`
	HasChildren bool   `json:"has_children,omitempty"`
	Updated     int64  `json:"updated_at"`
}

func (s *Server) listPages(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		ParentID    string `json:"parent_id"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	ws := in.WorkspaceID
	if in.ParentID != "" {
		m, err := s.Pages.Get(ctx, x.p, in.ParentID)
		if err != nil {
			return nil, "", err
		}
		ws = m.WorkspaceID
	}
	if ws == "" {
		list, err := s.workspaces(ctx, x, "")
		if err != nil {
			return nil, "", err
		}
		if len(list) != 1 {
			return nil, "", fail("this credential reaches several workspaces; pass workspace_id (see whoami)")
		}
		ws = list[0]
	}
	tree, err := s.Pages.Tree(ctx, x.p, ws)
	if err != nil {
		return nil, "", err
	}
	parents := map[string]bool{}
	for _, m := range tree {
		parents[m.ParentID] = true
	}
	out := []pageOut{}
	for _, m := range tree {
		if m.ParentID != in.ParentID || m.Row || m.IsTemplate {
			continue
		}
		out = append(out, pageOut{ID: m.ID, Title: m.Title, Kind: m.Kind, Icon: m.Icon, HasChildren: parents[m.ID], Updated: m.UpdatedAt})
	}
	return map[string]any{"workspace_id": ws, "parent_id": in.ParentID, "pages": out}, fmt.Sprintf("%d pages", len(out)), nil
}

func crumbs(path []page.Crumb) []map[string]string {
	out := []map[string]string{}
	for _, c := range path {
		out = append(out, map[string]string{"id": c.ID, "title": c.Title})
	}
	return out
}

func (s *Server) getPage(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID string `json:"page_id"`
		Nested bool   `json:"nested"`
		Format string `json:"format"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	d, err := s.Pages.Get(ctx, x.p, in.PageID)
	if err != nil {
		return nil, "", err
	}
	res := map[string]any{"id": d.ID, "title": d.Title, "kind": d.Kind, "workspace_id": d.WorkspaceID, "parent_id": d.ParentID, "path": crumbs(d.Path),
		"description": d.Description, "tags": d.Tags, "level": d.Level, "can_edit": d.CanWrite, "updated_at": d.UpdatedAt}
	if d.Kind == "collection" {
		res["note"] = "this page is a database; use get_database and query_database for its rows"
	}
	if d.Row {
		if rd, err := s.Cols.Row(ctx, x.p, d.ID); err == nil {
			res["database_id"] = rd.Collection.ID
			res["properties"] = s.namedValues(ctx, x, rd.Collection, rd.Row, rd.Refs)
		}
	}
	if in.Format == "markdown" {
		md, err := s.Docs.Markdown(ctx, x.p, d.ID)
		if err != nil {
			return nil, "", err
		}
		res["markdown"] = md
		return res, fmt.Sprintf("Page %q (%s)", d.Title, d.ID), nil
	}
	blocks, err := s.Docs.Outline(ctx, x.p, d.ID, in.Nested)
	if err != nil {
		return nil, "", err
	}
	if in.Limit <= 0 || in.Limit > 500 {
		in.Limit = 200
	}
	total := len(blocks)
	if in.Offset > 0 {
		if in.Offset > len(blocks) {
			in.Offset = len(blocks)
		}
		blocks = blocks[in.Offset:]
	}
	more := false
	if len(blocks) > in.Limit {
		blocks, more = blocks[:in.Limit], true
	}
	res["blocks"] = blocks
	res["total_blocks"] = total
	if more {
		res["next_offset"] = in.Offset + in.Limit
	}
	return res, fmt.Sprintf("Page %q (%s), %d of %d blocks", d.Title, d.ID, len(blocks), total), nil
}

type propOut struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Options []string `json:"options,omitempty"`
	Target  string   `json:"database,omitempty"`
	Expr    string   `json:"formula,omitempty"`
}

func schemaOut(sc collection.Schema) []propOut {
	out := []propOut{}
	for _, p := range sc {
		po := propOut{ID: p.ID, Name: p.Name, Type: p.Type, Expr: p.Expr}
		for _, o := range p.Options {
			po.Options = append(po.Options, o.Name)
		}
		if p.Type == collection.TRelation {
			po.Target = p.Collection
		}
		out = append(out, po)
	}
	return out
}

func (s *Server) getDatabase(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		DatabaseID string `json:"database_id"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	c, err := s.Cols.Get(ctx, x.p, in.DatabaseID)
	if err != nil {
		return nil, "", err
	}
	views := []map[string]string{}
	for _, v := range c.Views {
		views = append(views, map[string]string{"id": v.ID, "name": v.Name, "type": v.Type})
	}
	return map[string]any{"id": c.ID, "title": c.Title, "workspace_id": c.WorkspaceID, "properties": schemaOut(c.Schema), "views": views, "rows": c.Rows, "can_edit": c.CanWrite, "can_change_properties": c.CanDesign},
		fmt.Sprintf("Database %q with %d properties and %d rows", c.Title, len(c.Schema), c.Rows), nil
}

func (s *Server) people(ctx context.Context, x *exec, workspaceID string) map[string]string {
	out := map[string]string{}
	list, err := s.Workspace.Members(ctx, x.p, workspaceID)
	if err != nil {
		return out
	}
	for _, m := range list {
		out[m.UserID] = m.Name
	}
	return out
}

func (s *Server) namedValues(ctx context.Context, x *exec, c *collection.Out, r *collection.RowOut, refs map[string]collection.Ref) map[string]any {
	people := s.people(ctx, x, c.WorkspaceID)
	out := map[string]any{}
	for _, p := range c.Schema {
		if p.Type == collection.TTitle {
			out[p.Name] = r.Title
			continue
		}
		v, ok := r.Values[p.ID]
		if !ok || v == nil {
			continue
		}
		out[p.Name] = readable(p, v, people, refs)
	}
	return out
}

func flatCell(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	inner := m["value"]
	switch m["type"] {
	case "list":
		items, _ := inner.([]any)
		out := make([]any, 0, len(items))
		for _, it := range items {
			out = append(out, flatCell(it))
		}
		return out
	case "person":
		if pm, ok := inner.(map[string]any); ok {
			return pm["name"]
		}
	case "page":
		if pm, ok := inner.(map[string]any); ok {
			return pm["title"]
		}
	}
	return inner
}

func readable(p *collection.Property, v any, people map[string]string, refs map[string]collection.Ref) any {
	if raw, ok := v.(json.RawMessage); ok {
		var d any
		if json.Unmarshal(raw, &d) == nil {
			v = d
		}
	}
	if p.Type == collection.TFormula || p.Type == collection.TRollup {
		return flatCell(v)
	}
	name := func(id string) string {
		if o := p.Option(id); o != nil {
			return o.Name
		}
		return id
	}
	list := func(v any) []string {
		var out []string
		switch xs := v.(type) {
		case []string:
			out = xs
		case []any:
			for _, e := range xs {
				if s, ok := e.(string); ok {
					out = append(out, s)
				}
			}
		case string:
			out = []string{xs}
		}
		return out
	}
	switch p.Type {
	case collection.TSelect, collection.TStatus:
		if id, ok := v.(string); ok {
			return name(id)
		}
	case collection.TMulti:
		var names []string
		for _, id := range list(v) {
			names = append(names, name(id))
		}
		return names
	case collection.TPerson, collection.TCreatedBy, collection.TEditedBy:
		var names []string
		for _, id := range list(v) {
			if n := people[id]; n != "" {
				names = append(names, n)
			} else {
				names = append(names, id)
			}
		}
		if p.Type != collection.TPerson && len(names) == 1 {
			return names[0]
		}
		return names
	case collection.TRelation:
		var out []map[string]string
		for _, id := range list(v) {
			out = append(out, map[string]string{"id": id, "title": refs[id].Title})
		}
		return out
	}
	if raw, ok := v.(json.RawMessage); ok {
		var any_ any
		if json.Unmarshal(raw, &any_) == nil {
			return any_
		}
	}
	return v
}

func (s *Server) queryDatabase(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		DatabaseID string               `json:"database_id"`
		Filter     *collection.Filter   `json:"filter"`
		Sorts      []collection.SortKey `json:"sorts"`
		Search     string               `json:"search"`
		ViewID     string               `json:"view_id"`
		Offset     int                  `json:"offset"`
		Limit      int                  `json:"limit"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	c, err := s.Cols.Get(ctx, x.p, in.DatabaseID)
	if err != nil {
		return nil, "", err
	}
	byName := func(k string) string {
		if p := c.Schema.ByID(k); p != nil {
			return p.ID
		}
		if p := c.Schema.ByName(k); p != nil {
			return p.ID
		}
		for _, p := range c.Schema {
			if strings.EqualFold(p.Name, k) {
				return p.ID
			}
		}
		return k
	}
	optionIDs := func(p *collection.Property, raw json.RawMessage) json.RawMessage {
		if p == nil || len(p.Options) == 0 || len(raw) == 0 {
			return raw
		}
		conv := func(v string) string {
			if p.Option(v) != nil {
				return v
			}
			for _, o := range p.Options {
				if strings.EqualFold(o.Name, v) {
					return o.ID
				}
			}
			return v
		}
		var one string
		if json.Unmarshal(raw, &one) == nil {
			b, _ := json.Marshal(conv(one))
			return b
		}
		var many []string
		if json.Unmarshal(raw, &many) == nil {
			for i := range many {
				many[i] = conv(many[i])
			}
			b, _ := json.Marshal(many)
			return b
		}
		return raw
	}
	var fix func(f *collection.Filter)
	fix = func(f *collection.Filter) {
		if f == nil {
			return
		}
		if f.Prop != "" {
			f.Prop = byName(f.Prop)
			f.Value = optionIDs(c.Schema.ByID(f.Prop), f.Value)
		}
		for i := range f.Items {
			fix(&f.Items[i])
		}
	}
	fix(in.Filter)
	for i := range in.Sorts {
		in.Sorts[i].Prop = byName(in.Sorts[i].Prop)
	}
	if in.Limit <= 0 || in.Limit > 200 {
		in.Limit = 50
	}
	q := collection.QueryInput{ViewID: in.ViewID, Filter: in.Filter, Sorts: in.Sorts, Search: in.Search, Offset: in.Offset, Limit: in.Limit, NoGroups: true}
	res, err := s.Cols.Query(ctx, x.p, c.ID, q)
	if err != nil {
		return nil, "", err
	}
	rows := []map[string]any{}
	for i := range res.Rows {
		r := &res.Rows[i]
		rows = append(rows, map[string]any{"id": r.ID, "title": r.Title, "values": s.namedValues(ctx, x, c, r, res.Refs), "updated_at": r.UpdatedAt})
	}
	out := map[string]any{"database_id": c.ID, "total": res.Total, "offset": in.Offset, "rows": rows}
	if in.Offset+len(rows) < res.Total {
		out["next_offset"] = in.Offset + len(rows)
	}
	return out, fmt.Sprintf("%d of %d rows from %q", len(rows), res.Total, c.Title), nil
}

func (s *Server) listComments(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID          string `json:"page_id"`
		IncludeResolved bool   `json:"include_resolved"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	l, err := s.Comments.List(ctx, x.p, in.PageID)
	if err != nil {
		return nil, "", err
	}
	type cOut struct {
		ID     string `json:"id"`
		Author string `json:"author"`
		Body   string `json:"body"`
		At     int64  `json:"created_at"`
	}
	type tOut struct {
		ID       string `json:"id"`
		Anchor   string `json:"anchor"`
		Quote    string `json:"quote,omitempty"`
		Resolved bool   `json:"resolved"`
		Comments []cOut `json:"comments"`
	}
	out := []tOut{}
	for _, t := range l.Threads {
		if t.ResolvedAt > 0 && !in.IncludeResolved {
			continue
		}
		to := tOut{ID: t.ID, Anchor: t.Anchor, Quote: t.Quote, Resolved: t.ResolvedAt > 0, Comments: []cOut{}}
		for _, c := range t.Comments {
			to.Comments = append(to.Comments, cOut{ID: c.ID, Author: c.AuthorName, Body: c.Body, At: c.CreatedAt})
		}
		out = append(out, to)
	}
	return map[string]any{"page_id": in.PageID, "threads": out, "can_comment": l.CanComment}, fmt.Sprintf("%d threads", len(out)), nil
}

func (s *Server) listRuns(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		Limit int `json:"limit"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 20
	}
	list, err := s.Runs.List(ctx, x.p, "", in.Limit)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"runs": list}, fmt.Sprintf("%d runs", len(list)), nil
}
