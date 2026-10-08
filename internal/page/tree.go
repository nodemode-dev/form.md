package page

import (
	"context"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

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

func (s *Service) workspacePages(ctx context.Context, workspaceID string, trashed, withRows bool) ([]*Meta, error) {
	cond := "p.trashed_at = 0"
	if trashed {
		cond = "1 = 1"
	}
	if !withRows {
		cond += " AND p.is_row = 0"
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
	v, err := s.viewer(ctx, s.st.R, p, workspaceID)
	if err != nil {
		return nil, err
	}
	all, err := s.workspacePages(ctx, workspaceID, false, true)
	if err != nil {
		return nil, err
	}
	levels := v.levels(all)
	tpl := inTemplates(all)
	out := make([]*Meta, 0, len(all))
	for _, m := range all {
		if levels[m.ID] >= LevelView && !tpl[m.ID] && !m.Row {
			out = append(out, m)
		}
	}
	for _, m := range out {
		m.Access = levels[m.ID].String()
		switch {
		case m.ParentID != "" && levels[m.ParentID] < LevelView:
		case m.ParentID == "" && (!v.member || (m.Visibility == "private" && m.OwnerID != v.me) || (m.TeamspaceID != "" && v.root(chainNode{teamspace: m.TeamspaceID}) == LevelNone)):
		default:
			continue
		}
		m.Detached = true
	}
	return out, nil
}

func marks(n int) string {
	if n == 0 {
		return "SELECT NULL WHERE 0"
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func (s *Service) Filter(ctx context.Context, p *policy.Principal, workspaceID string) (func(id string) bool, error) {
	v, err := s.viewer(ctx, s.st.R, p, workspaceID)
	if err != nil {
		return nil, err
	}
	shared := make([]any, 0, len(v.shares))
	for id := range v.shares {
		shared = append(shared, id)
	}
	var q string
	var args []any
	if v.member {
		var teams []any
		for id := range v.teams {
			if v.root(chainNode{teamspace: id}) == LevelNone {
				teams = append(teams, id)
			}
		}
		q = `WITH RECURSIVE hid(id) AS (
			SELECT id FROM pages WHERE workspace_id = ? AND ((visibility = 'private' AND COALESCE(owner_id, '') != ?) OR (parent_id IS NULL AND teamspace_id IN (` + marks(len(teams)) + `))) AND id NOT IN (` + marks(len(shared)) + `)
			UNION SELECT c.id FROM pages c JOIN hid ON c.parent_id = hid.id WHERE c.id NOT IN (` + marks(len(shared)) + `))
			SELECT id FROM hid`
		args = append([]any{workspaceID, v.me}, teams...)
		args = append(append(args, shared...), shared...)
	} else {
		q = `WITH RECURSIVE vis(id) AS (
			SELECT id FROM pages WHERE workspace_id = ? AND id IN (` + marks(len(shared)) + `)
			UNION SELECT c.id FROM pages c JOIN vis ON c.parent_id = vis.id WHERE c.visibility != 'private' OR c.id IN (` + marks(len(shared)) + `))
			SELECT id FROM vis`
		args = append(append([]any{workspaceID}, shared...), shared...)
	}
	rows, err := s.st.R.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	set := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		set[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if v.member {
		return func(id string) bool { return !set[id] }, nil
	}
	return func(id string) bool { return set[id] }, nil
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
	all, err := s.workspacePages(ctx, workspaceID, true, true)
	if err != nil {
		return nil, err
	}
	v, err := s.viewer(ctx, s.st.R, p, workspaceID)
	if err != nil {
		return nil, err
	}
	levels := v.levels(all)
	seen := make(map[string]bool, len(levels))
	for id, l := range levels {
		seen[id] = l >= LevelEdit
	}
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
	return s.require(ctx, p, id, LevelContent)
}

type Crumb struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
}

type Detail struct {
	*Meta
	Path       []Crumb `json:"path"`
	CanWrite   bool    `json:"can_write"`
	Level      string  `json:"level"`
	CanEdit    bool    `json:"can_edit"`
	CanComment bool    `json:"can_comment"`
	CanShare   bool    `json:"can_share"`
	Guest      bool    `json:"guest,omitempty"`
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
	if !a.viewer.member || len(a.viewer.shares) > 0 {
		visible := path[:0]
		l := LevelNone
		for i, n := range a.chain {
			l = a.viewer.step(l, n, i == 0)
			if i < len(path) && l >= LevelView {
				visible = append(visible, path[i])
			}
		}
		path = visible
	}
	level := a.level
	if !p.CanWrite() && level > LevelView {
		level = LevelView
	}
	live := m.TrashedAt == 0
	return &Detail{Meta: m, Path: path, CanWrite: live && level >= LevelContent, Level: level.String(), CanEdit: live && level >= LevelEdit,
		CanComment: live && level >= LevelComment, CanShare: live && level >= LevelFull, Guest: !a.viewer.member}, nil
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
