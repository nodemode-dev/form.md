package page

import (
	"context"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

func visibleIn(all []*Meta, me string) map[string]bool {
	byID := make(map[string]*Meta, len(all))
	for _, m := range all {
		byID[m.ID] = m
	}
	memo := make(map[string]bool, len(all))
	var visit func(m *Meta, depth int) bool
	visit = func(m *Meta, depth int) bool {
		if v, ok := memo[m.ID]; ok {
			return v
		}
		ok := !(m.Visibility == "private" && m.OwnerID != me)
		if ok && m.ParentID != "" && depth < 512 {
			if parent, found := byID[m.ParentID]; found {
				ok = visit(parent, depth+1)
			}
		}
		memo[m.ID] = ok
		return ok
	}
	out := make(map[string]bool, len(all))
	for _, m := range all {
		if visit(m, 0) {
			out[m.ID] = true
		}
	}
	return out
}

func inTemplates(all []*Meta) map[string]bool {
	byID := make(map[string]*Meta, len(all))
	for _, m := range all {
		byID[m.ID] = m
	}
	memo := map[string]bool{}
	var under func(m *Meta, depth int) bool
	under = func(m *Meta, depth int) bool {
		if v, ok := memo[m.ID]; ok {
			return v
		}
		v := m.IsTemplate
		if !v && m.ParentID != "" && depth < 512 {
			if parent, ok := byID[m.ParentID]; ok {
				v = under(parent, depth+1)
			}
		}
		memo[m.ID] = v
		return v
	}
	out := map[string]bool{}
	for _, m := range all {
		if under(m, 0) {
			out[m.ID] = true
		}
	}
	return out
}

func (s *Service) workspacePages(ctx context.Context, workspaceID string, trashed bool) ([]*Meta, error) {
	cond := "p.trashed_at = 0"
	if trashed {
		cond = "1 = 1"
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT "+metaColumns+" FROM pages p WHERE p.workspace_id = ? AND "+cond+" ORDER BY p.parent_id, p.sort_key", workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Meta
	for rows.Next() {
		m, err := scanMeta(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) Tree(ctx context.Context, p *policy.Principal, workspaceID string) ([]*Meta, error) {
	if _, err := s.gate.Workspace(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	all, err := s.workspacePages(ctx, workspaceID, false)
	if err != nil {
		return nil, err
	}
	seen := visibleIn(all, p.UserID)
	tpl := inTemplates(all)
	out := make([]*Meta, 0, len(all))
	for _, m := range all {
		if seen[m.ID] && !tpl[m.ID] {
			out = append(out, m)
		}
	}
	return out, nil
}

type TrashItem struct {
	*Meta
	TrashedByName string `json:"trashed_by_name"`
	Pages         int    `json:"pages"`
}

func (s *Service) Trash(ctx context.Context, p *policy.Principal, workspaceID, query string) ([]TrashItem, error) {
	if _, err := s.gate.Workspace(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	all, err := s.workspacePages(ctx, workspaceID, true)
	if err != nil {
		return nil, err
	}
	seen := visibleIn(all, p.UserID)
	byID := make(map[string]*Meta, len(all))
	for _, m := range all {
		byID[m.ID] = m
	}
	count := map[string]int{}
	rootOf := func(m *Meta) *Meta {
		for depth := 0; depth < 512; depth++ {
			parent, ok := byID[m.ParentID]
			if !ok || parent.trashBatch != m.trashBatch {
				return m
			}
			m = parent
		}
		return m
	}
	for _, m := range all {
		if m.TrashedAt > 0 && seen[m.ID] {
			count[rootOf(m).ID]++
		}
	}
	names, err := s.userNames(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []TrashItem
	for id, n := range count {
		m := byID[id]
		if q != "" && !strings.Contains(strings.ToLower(m.Title), q) {
			continue
		}
		out = append(out, TrashItem{Meta: m, TrashedByName: names[m.TrashedBy], Pages: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TrashedAt > out[j].TrashedAt })
	if out == nil {
		out = []TrashItem{}
	}
	return out, nil
}

func (s *Service) userNames(ctx context.Context) (map[string]string, error) {
	rows, err := s.st.R.QueryContext(ctx, "SELECT id, name FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

func (s *Service) CanRead(ctx context.Context, p *policy.Principal, id string) error {
	_, _, err := s.readable(ctx, p, id)
	return err
}

func (s *Service) CanWrite(ctx context.Context, p *policy.Principal, id string) (string, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return "", err
	}
	if _, err := s.check(ctx, s.st.R, p, m, true); err != nil {
		return "", err
	}
	if m.TrashedAt > 0 {
		return "", errTrashed
	}
	return m.WorkspaceID, nil
}

type Crumb struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
}

type Detail struct {
	*Meta
	Path     []Crumb `json:"path"`
	CanWrite bool    `json:"can_write"`
}

func (s *Service) Get(ctx context.Context, p *policy.Principal, id string) (*Detail, error) {
	m, a, err := s.readable(ctx, p, id)
	if err != nil {
		return nil, err
	}
	path, err := s.path(ctx, m)
	if err != nil {
		return nil, err
	}
	return &Detail{Meta: m, Path: path, CanWrite: a.ws.CanWrite() && m.TrashedAt == 0}, nil
}

func (s *Service) path(ctx context.Context, m *Meta) ([]Crumb, error) {
	rows, err := s.st.R.QueryContext(ctx, `WITH RECURSIVE chain(id, parent_id, title, icon, depth) AS (
			SELECT id, parent_id, title, icon, 0 FROM pages WHERE id = ?
			UNION ALL
			SELECT p.id, p.parent_id, p.title, p.icon, c.depth + 1 FROM pages p JOIN chain c ON p.id = c.parent_id WHERE c.depth < 512
		) SELECT id, title, icon FROM chain WHERE id != ? ORDER BY depth DESC`, m.ID, m.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Crumb{}
	for rows.Next() {
		var c Crumb
		if err := rows.Scan(&c.ID, &c.Title, &c.Icon); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type Tag struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
	Color string `json:"color"`
}

func (s *Service) Tags(ctx context.Context, p *policy.Principal, workspaceID string) ([]Tag, error) {
	pages, err := s.Tree(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	counts := map[string]*Tag{}
	for _, m := range pages {
		for _, t := range m.Tags {
			k := strings.ToLower(t)
			if c, ok := counts[k]; ok {
				c.Count++
			} else {
				counts[k] = &Tag{Tag: t, Count: 1}
			}
		}
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT tag, color FROM tag_colors WHERE workspace_id = ?", workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag, color string
		if err := rows.Scan(&tag, &color); err != nil {
			return nil, err
		}
		if c, ok := counts[strings.ToLower(tag)]; ok {
			c.Color = color
		}
	}
	out := make([]Tag, 0, len(counts))
	for _, c := range counts {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return strings.ToLower(out[i].Tag) < strings.ToLower(out[j].Tag)
	})
	return out, rows.Err()
}

var tagColors = map[string]bool{"": true, "violet": true, "blue": true, "cyan": true, "teal": true, "green": true, "lime": true, "amber": true, "orange": true, "rose": true, "pink": true}

func (s *Service) SetTagColor(ctx context.Context, p *policy.Principal, workspaceID, tag, color string) error {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.CanWrite() {
		return httpx.ErrForbidden
	}
	tags := cleanTags([]string{tag})
	if len(tags) == 0 || !tagColors[color] {
		return httpx.Fail(400, "bad_tag", "a tag and one of the palette colours are needed")
	}
	if color == "" {
		_, err = s.st.W.ExecContext(ctx, "DELETE FROM tag_colors WHERE workspace_id = ? AND tag = ?", workspaceID, tags[0])
	} else {
		_, err = s.st.W.ExecContext(ctx, `INSERT INTO tag_colors (workspace_id, tag, color) VALUES (?, ?, ?)
			ON CONFLICT (workspace_id, tag) DO UPDATE SET color = excluded.color`, workspaceID, tags[0], color)
	}
	if err != nil {
		return err
	}
	s.hub.Workspace(workspaceID, "tags")
	return nil
}
