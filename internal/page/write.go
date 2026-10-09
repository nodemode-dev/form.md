package page

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/icon"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/order"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

func lastChildKey(ctx context.Context, q store.Querier, workspaceID, parentID, exclude string) (string, error) {
	var key sql.NullString
	err := q.QueryRowContext(ctx, `SELECT MAX(sort_key) FROM pages WHERE workspace_id = ? AND parent_id IS ? AND id != ?`,
		workspaceID, nullable(parentID), exclude).Scan(&key)
	return key.String, err
}

func nullable(id string) any {
	if id == "" {
		return nil
	}
	return id
}

func (s *Service) slot(ctx context.Context, q store.Querier, workspaceID, parentID, beforeID, afterID, moving string) (string, error) {
	sibling := func(id string) (string, error) {
		var key, parent string
		err := q.QueryRowContext(ctx, "SELECT sort_key, COALESCE(parent_id, '') FROM pages WHERE id = ? AND workspace_id = ?", id, workspaceID).Scan(&key, &parent)
		if err != nil {
			return "", httpx.Fail(http.StatusBadRequest, "bad_sibling", "the neighbouring page was not found")
		}
		if parent != parentID {
			return "", httpx.Fail(http.StatusBadRequest, "bad_sibling", "the neighbouring page has another parent")
		}
		return key, nil
	}
	neighbour := func(key string, next bool) (string, error) {
		op, dir := ">", "ASC"
		if !next {
			op, dir = "<", "DESC"
		}
		var k sql.NullString
		err := q.QueryRowContext(ctx, `SELECT sort_key FROM pages WHERE workspace_id = ? AND parent_id IS ? AND id != ? AND sort_key `+op+` ?
			ORDER BY sort_key `+dir+` LIMIT 1`, workspaceID, nullable(parentID), moving, key).Scan(&k)
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return k.String, err
	}
	switch {
	case afterID != "":
		a, err := sibling(afterID)
		if err != nil {
			return "", err
		}
		b, err := neighbour(a, true)
		if err != nil {
			return "", err
		}
		return order.Between(a, b)
	case beforeID != "":
		b, err := sibling(beforeID)
		if err != nil {
			return "", err
		}
		a, err := neighbour(b, false)
		if err != nil {
			return "", err
		}
		return order.Between(a, b)
	}
	last, err := lastChildKey(ctx, q, workspaceID, parentID, moving)
	if err != nil {
		return "", err
	}
	return order.After(last), nil
}

type CreateInput struct {
	WorkspaceID string          `json:"workspace_id"`
	ParentID    string          `json:"parent_id"`
	AfterID     string          `json:"after_id"`
	Title       string          `json:"title"`
	Icon        string          `json:"icon"`
	Kind        string          `json:"kind"`
	TeamspaceID string          `json:"teamspace_id"`
	Extra       json.RawMessage `json:"extra,omitempty"`
}

func (s *Service) Create(ctx context.Context, p *policy.Principal, in CreateInput) (*Meta, error) {
	title, err := cleanTitle(in.Title)
	if err != nil {
		return nil, err
	}
	kind := in.Kind
	if kind == "" {
		kind = "doc"
	}
	if kind != "doc" && kind != "collection" {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_kind", "a page is a doc or a collection")
	}
	if !icon.Valid(in.Icon) || strings.HasPrefix(in.Icon, "f:") {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_icon", "an icon is a name from the icon set or an uploaded image")
	}
	workspaceID := in.WorkspaceID
	teamspace := ""
	if in.ParentID != "" {
		parent, err := s.load(ctx, s.st.R, in.ParentID)
		if err != nil {
			return nil, err
		}
		if _, err := s.check(ctx, s.st.R, p, parent, LevelContent); err != nil {
			return nil, err
		}
		if parent.TrashedAt > 0 {
			return nil, errTrashed
		}
		workspaceID = parent.WorkspaceID
	} else {
		a, err := s.gate.Workspace(ctx, p, workspaceID)
		if err != nil {
			return nil, err
		}
		if !a.CanWrite() {
			return nil, httpx.ErrForbidden
		}
		if in.TeamspaceID != "" {
			if err := s.teamWritable(ctx, p, workspaceID, in.TeamspaceID); err != nil {
				return nil, err
			}
			teamspace = in.TeamspaceID
		}
	}
	id := ids.New()
	now := store.NowMS()
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		key, err := s.slot(ctx, tx, workspaceID, in.ParentID, "", in.AfterID, id)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO pages (id, workspace_id, parent_id, kind, sort_key, title, icon, owner_id, created_at, created_by, updated_at, updated_by, teamspace_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceID, nullable(in.ParentID), kind, key, title, in.Icon, p.UserID, now, p.UserID, now, p.UserID, nullable(teamspace)); err != nil {
			return err
		}
		for _, fn := range s.creators {
			if err := fn(ctx, tx, id, workspaceID, kind, in.Extra); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.create", WorkspaceID: workspaceID, PageID: id, Detail: title})
	})
	if err != nil {
		return nil, err
	}
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	s.emit(ctx, m)
	return m, nil
}

type UpdateInput struct {
	Title       *string   `json:"title"`
	Icon        *string   `json:"icon"`
	Cover       *string   `json:"cover"`
	Description *string   `json:"description"`
	Tags        *[]string `json:"tags"`
	Visibility  *string   `json:"visibility"`
	IsTemplate  *bool     `json:"is_template"`
}

func (s *Service) Update(ctx context.Context, p *policy.Principal, id string, in UpdateInput) (*Meta, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	a, err := s.check(ctx, s.st.R, p, m, LevelContent)
	if err != nil {
		return nil, err
	}
	if m.TrashedAt > 0 {
		return nil, errTrashed
	}
	if in.Visibility != nil && (a.level < LevelEdit || !a.viewer.member) {
		return nil, httpx.ErrForbidden
	}
	sets, args := []string{}, []any{}
	var changes []audit.Change
	if in.Title != nil {
		t, err := cleanTitle(*in.Title)
		if err != nil {
			return nil, err
		}
		sets, args = append(sets, "title = ?"), append(args, t)
		changes = append(changes, audit.Change{Field: "title", Before: m.Title, After: t})
	}
	if in.Icon != nil {
		v := strings.TrimSpace(*in.Icon)
		if !icon.Valid(v) {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_icon", "an icon is a name from the icon set or an uploaded image")
		}
		if err := s.fileInWorkspace(ctx, s.st.R, v, m.WorkspaceID); err != nil {
			return nil, err
		}
		sets, args = append(sets, "icon = ?"), append(args, v)
		changes = append(changes, audit.Change{Field: "icon", Before: m.Icon, After: v})
	}
	if in.Cover != nil {
		v := strings.TrimSpace(*in.Cover)
		if !validCover(v) {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_cover", "a cover is one of the gradients or an uploaded image")
		}
		if err := s.fileInWorkspace(ctx, s.st.R, v, m.WorkspaceID); err != nil {
			return nil, err
		}
		sets, args = append(sets, "cover = ?"), append(args, v)
		changes = append(changes, audit.Change{Field: "cover", Before: m.Cover, After: v})
	}
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if utf8.RuneCountInString(d) > MaxDescription {
			return nil, httpx.Fail(http.StatusBadRequest, "description_too_long", "a description can be up to 2,000 characters")
		}
		sets, args = append(sets, "description = ?"), append(args, d)
		changes = append(changes, audit.Change{Field: "description", Before: m.Description, After: d})
	}
	if in.Tags != nil {
		tags := cleanTags(*in.Tags)
		b, _ := json.Marshal(tags)
		sets, args = append(sets, "tags = ?"), append(args, string(b))
		before, _ := json.Marshal(m.Tags)
		changes = append(changes, audit.Change{Field: "tags", Before: string(before), After: string(b)})
	}
	if in.IsTemplate != nil && *in.IsTemplate != m.IsTemplate {
		if a.level < LevelEdit || !a.viewer.member || m.Row {
			return nil, httpx.ErrForbidden
		}
		sets, args = append(sets, "is_template = ?"), append(args, store.Bool(*in.IsTemplate))
		changes = append(changes, audit.Change{Field: "is_template", Before: strconv.FormatBool(m.IsTemplate), After: strconv.FormatBool(*in.IsTemplate)})
	}
	movedPrivacy := false
	if in.Visibility != nil {
		v := *in.Visibility
		if v != "workspace" && v != "private" {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_visibility", "visibility is workspace or private")
		}
		if v != m.Visibility {
			if v == "private" && a.ws.Personal {
				return nil, httpx.Fail(http.StatusConflict, "personal_space", "everything in a personal space is already private")
			}
			sets, args = append(sets, "visibility = ?", "owner_id = ?"), append(args, v, p.UserID)
			changes = append(changes, audit.Change{Field: "visibility", Before: m.Visibility, After: v})
			movedPrivacy = true
		}
	}
	if len(sets) == 0 {
		return m, nil
	}
	before, err := s.Audience(ctx, m)
	if err != nil {
		return nil, err
	}
	sets = append(sets, "updated_at = ?", "updated_by = ?")
	args = append(args, store.NowMS(), p.UserID, id)
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE pages SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.update", WorkspaceID: m.WorkspaceID, PageID: id, Changes: changes})
	})
	if err != nil {
		return nil, err
	}
	if movedPrivacy {
		return s.republish(ctx, m.WorkspaceID, id, before)
	}
	out, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	s.emit(ctx, out)
	return out, nil
}

func (s *Service) subtree(ctx context.Context, q store.Querier, id string, liveOnly bool) ([]*Meta, error) {
	cond := ""
	if liveOnly {
		cond = " WHERE p.trashed_at = 0"
	}
	rows, err := q.QueryContext(ctx, `WITH RECURSIVE sub(id, depth) AS (
			SELECT id, 0 FROM pages WHERE id = ?
			UNION ALL
			SELECT c.id, s.depth + 1 FROM pages c JOIN sub s ON c.parent_id = s.id WHERE s.depth < 512
		) SELECT `+metaColumns+` FROM sub JOIN pages p ON p.id = sub.id`+cond+` ORDER BY sub.depth, p.sort_key`, id)
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

func (s *Service) republish(ctx context.Context, workspaceID, id string, before events.Audience) (*Meta, error) {
	pages, err := s.subtree(ctx, s.st.R, id, true)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, store.ErrNotFound
	}
	after, err := s.Audience(ctx, pages[0])
	if err != nil {
		return nil, err
	}
	same := after.Scope == before.Scope && strings.Join(after.Grantees, ",") == strings.Join(before.Grantees, ",")
	if !same && after.Scope != "" {
		ids := make([]string, len(pages))
		for i, m := range pages {
			ids[i] = m.ID
		}
		s.hub.Removed(workspaceID, ids, &after)
	}
	for _, m := range pages {
		if same && m.ID != id {
			continue
		}
		s.emit(ctx, m)
	}
	return pages[0], nil
}

type MoveInput struct {
	WorkspaceID string  `json:"workspace_id"`
	ParentID    *string `json:"parent_id"`
	TeamspaceID *string `json:"teamspace_id"`
	BeforeID    string  `json:"before_id"`
	AfterID     string  `json:"after_id"`
}

func (s *Service) Move(ctx context.Context, p *policy.Principal, id string, in MoveInput) (*Meta, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	a, err := s.check(ctx, s.st.R, p, m, LevelEdit)
	if err != nil {
		return nil, err
	}
	if m.TrashedAt > 0 {
		return nil, errTrashed
	}
	if in.WorkspaceID != "" && in.WorkspaceID != m.WorkspaceID {
		return s.moveAcross(ctx, p, m, in)
	}
	parentID := m.ParentID
	if in.ParentID != nil {
		parentID = *in.ParentID
	}
	teamspace := ""
	if parentID != "" {
		parent, err := s.load(ctx, s.st.R, parentID)
		if err != nil {
			return nil, err
		}
		if parent.WorkspaceID != m.WorkspaceID {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_parent", "use workspace_id to move a page to another workspace")
		}
		if _, err := s.check(ctx, s.st.R, p, parent, LevelContent); err != nil {
			return nil, err
		}
		if parent.TrashedAt > 0 {
			return nil, errTrashed
		}
	} else {
		if !a.viewer.member || !a.ws.CanWrite() {
			return nil, httpx.ErrForbidden
		}
		if len(a.chain) > 0 {
			teamspace = a.chain[0].teamspace
		}
		if in.TeamspaceID != nil {
			teamspace = *in.TeamspaceID
		}
		if teamspace != "" {
			if err := s.teamWritable(ctx, p, m.WorkspaceID, teamspace); err != nil {
				return nil, err
			}
		}
	}
	before, err := s.Audience(ctx, m)
	if err != nil {
		return nil, err
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if parentID != "" {
			var inside int
			if err := tx.QueryRowContext(ctx, `WITH RECURSIVE sub(id, depth) AS (
					SELECT id, 0 FROM pages WHERE id = ?
					UNION ALL
					SELECT c.id, s.depth + 1 FROM pages c JOIN sub s ON c.parent_id = s.id WHERE s.depth < 512
				) SELECT COUNT(*) FROM sub WHERE id = ?`, id, parentID).Scan(&inside); err != nil {
				return err
			}
			if inside > 0 {
				return httpx.Fail(http.StatusConflict, "move_into_self", "a page cannot move inside itself")
			}
		}
		key, err := s.slot(ctx, tx, m.WorkspaceID, parentID, in.BeforeID, in.AfterID, id)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE pages SET parent_id = ?, teamspace_id = ?, sort_key = ?, updated_at = ?, updated_by = ? WHERE id = ?",
			nullable(parentID), nullable(teamspace), key, store.NowMS(), p.UserID, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.move", WorkspaceID: m.WorkspaceID, PageID: id,
			Changes: []audit.Change{{Field: "parent", Before: m.ParentID, After: parentID}, {Field: "teamspace", Before: m.TeamspaceID, After: teamspace}}})
	})
	if err != nil {
		return nil, err
	}
	return s.republish(ctx, m.WorkspaceID, id, before)
}

func (s *Service) moveAcross(ctx context.Context, p *policy.Principal, m *Meta, in MoveInput) (*Meta, error) {
	target, err := s.gate.Workspace(ctx, p, in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if !target.CanWrite() {
		return nil, httpx.ErrForbidden
	}
	parentID := ""
	if in.ParentID != nil {
		parentID = *in.ParentID
	}
	if parentID != "" {
		parent, err := s.load(ctx, s.st.R, parentID)
		if err != nil {
			return nil, err
		}
		if parent.WorkspaceID != in.WorkspaceID || parent.TrashedAt > 0 {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_parent", "the new parent is not a live page in that workspace")
		}
		if _, err := s.check(ctx, s.st.R, p, parent, LevelContent); err != nil {
			return nil, err
		}
	}
	pages, err := s.subtree(ctx, s.st.R, m.ID, false)
	if err != nil {
		return nil, err
	}
	all, err := s.workspacePages(ctx, m.WorkspaceID, true, true)
	if err != nil {
		return nil, err
	}
	v, err := s.viewer(ctx, s.st.R, p, m.WorkspaceID)
	if err != nil {
		return nil, err
	}
	seen := v.levels(all)
	ids := make([]string, len(pages))
	for i, x := range pages {
		if seen[x.ID] < LevelEdit {
			return nil, httpx.Fail(http.StatusConflict, "hidden_subpages", "this page holds pages you cannot edit, so it cannot change workspace")
		}
		ids[i] = x.ID
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		key, err := s.slot(ctx, tx, in.WorkspaceID, parentID, in.BeforeID, in.AfterID, m.ID)
		if err != nil {
			return err
		}
		now := store.NowMS()
		if _, err := tx.ExecContext(ctx, "UPDATE pages SET parent_id = ?, teamspace_id = NULL, sort_key = ?, updated_at = ?, updated_by = ? WHERE id = ?",
			nullable(parentID), key, now, p.UserID, m.ID); err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := tx.ExecContext(ctx, "UPDATE pages SET workspace_id = ? WHERE id = ?", in.WorkspaceID, id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM page_shares WHERE page_id = ? AND subject_kind = 'group'", id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE page_shares SET workspace_id = ? WHERE page_id = ?", in.WorkspaceID, id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE share_links SET workspace_id = ? WHERE page_id = ?", in.WorkspaceID, id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE files SET workspace_id = ? WHERE page_id = ?", in.WorkspaceID, id); err != nil {
				return err
			}
		}
		for _, fn := range s.movers {
			if err := fn(ctx, tx, ids, in.WorkspaceID); err != nil {
				return err
			}
		}
		if err := s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.move_out", WorkspaceID: m.WorkspaceID, PageID: m.ID, Detail: in.WorkspaceID}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.move_in", WorkspaceID: in.WorkspaceID, PageID: m.ID, Detail: m.WorkspaceID})
	})
	if err != nil {
		return nil, err
	}
	s.hub.Removed(m.WorkspaceID, ids, nil)
	moved, err := s.subtree(ctx, s.st.R, m.ID, true)
	if err != nil {
		return nil, err
	}
	var rowIDs []string
	for _, x := range moved {
		if x.Row {
			rowIDs = append(rowIDs, x.ID)
			continue
		}
		s.emit(ctx, x)
	}
	s.notify(ctx, rowIDs)
	if len(moved) == 0 {
		return nil, store.ErrNotFound
	}
	return moved[0], nil
}

type DuplicateInput struct {
	Title string `json:"title"`
}

func (s *Service) Duplicate(ctx context.Context, p *policy.Principal, id string, in DuplicateInput) (*Meta, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	a, err := s.check(ctx, s.st.R, p, m, LevelView)
	if err != nil {
		return nil, err
	}
	if m.ParentID != "" {
		parent, err := s.load(ctx, s.st.R, m.ParentID)
		if err != nil {
			return nil, err
		}
		if _, err := s.check(ctx, s.st.R, p, parent, LevelContent); err != nil {
			return nil, err
		}
	} else if !a.viewer.member || !a.ws.CanWrite() {
		return nil, httpx.ErrForbidden
	}
	if m.TrashedAt > 0 {
		return nil, errTrashed
	}
	title := in.Title
	if strings.TrimSpace(title) == "" {
		title = "Copy of " + m.Title
		if r := []rune(title); len(r) > MaxTitle {
			title = string(r[:MaxTitle])
		}
	}
	if title, err = cleanTitle(title); err != nil {
		return nil, err
	}
	pages, err := s.subtree(ctx, s.st.R, id, true)
	if err != nil {
		return nil, err
	}
	all, err := s.workspacePages(ctx, m.WorkspaceID, false, true)
	if err != nil {
		return nil, err
	}
	levels := a.viewer.levels(all)
	visible := func(id string) bool { return levels[id] >= LevelView }
	mapping := map[string]string{}
	var rootID string
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		now := store.NowMS()
		for i, x := range pages {
			if i > 0 && (!visible(x.ID) || mapping[x.ParentID] == "") {
				continue
			}
			nid := ids.New()
			mapping[x.ID] = nid
			parent := mapping[x.ParentID]
			key, name := x.SortKey, x.Title
			if i == 0 {
				rootID = nid
				parent = x.ParentID
				name = title
				k, err := s.slot(ctx, tx, x.WorkspaceID, x.ParentID, "", x.ID, nid)
				if err != nil {
					return err
				}
				key = k
			}
			tags, _ := json.Marshal(x.Tags)
			team := ""
			if i == 0 && parent == "" {
				team = x.TeamspaceID
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO pages (id, workspace_id, parent_id, kind, sort_key, title, icon, cover, description, tags,
				visibility, owner_id, is_template, snippet, thumb, created_at, created_by, updated_at, updated_by, is_row, teamspace_id)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?)`,
				nid, x.WorkspaceID, nullable(parent), x.Kind, key, name, x.Icon, x.Cover, x.Description, string(tags),
				x.Visibility, p.UserID, x.Snippet, x.Thumb, now, p.UserID, now, p.UserID, store.Bool(x.Row), nullable(team)); err != nil {
				return err
			}
			for _, fn := range s.copiers {
				if err := fn(ctx, tx, x.ID, nid); err != nil {
					return err
				}
			}
		}
		for _, fn := range s.remaps {
			if err := fn(ctx, tx, mapping); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.duplicate", WorkspaceID: m.WorkspaceID, PageID: rootID, Detail: id})
	})
	if err != nil {
		return nil, err
	}
	copies, err := s.subtree(ctx, s.st.R, rootID, true)
	if err != nil {
		return nil, err
	}
	for _, x := range copies {
		if !x.Row {
			s.emit(ctx, x)
		}
	}
	return copies[0], nil
}

func (s *Service) TrashPage(ctx context.Context, p *policy.Principal, id string) ([]string, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	need := LevelEdit
	if m.Row {
		need = LevelContent
	}
	if _, err := s.check(ctx, s.st.R, p, m, need); err != nil {
		return nil, err
	}
	if m.TrashedAt > 0 {
		return nil, errTrashed
	}
	pages, err := s.subtree(ctx, s.st.R, id, true)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(pages))
	for i, x := range pages {
		ids[i] = x.ID
	}
	kind := "human"
	if p.IsAgent() {
		kind = "agent"
	}
	batch := newBatch()
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		now := store.NowMS()
		for _, pid := range ids {
			if _, err := tx.ExecContext(ctx, `UPDATE pages SET trashed_at = ?, trash_batch = ?, trashed_by = ?, trashed_by_kind = ?, updated_at = ?, updated_by = ?
				WHERE id = ? AND trashed_at = 0`, now, batch, p.UserID, kind, now, p.UserID, pid); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.trash", WorkspaceID: m.WorkspaceID, PageID: id, Detail: m.Title})
	})
	if err != nil {
		return nil, err
	}
	s.hub.Removed(m.WorkspaceID, ids, nil)
	s.hub.Workspace(m.WorkspaceID, "trash")
	s.notify(ctx, ids)
	return ids, nil
}

func newBatch() string { return ids.New() }

func (s *Service) Restore(ctx context.Context, p *policy.Principal, id string) (*Meta, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.check(ctx, s.st.R, p, m, LevelEdit); err != nil {
		return nil, err
	}
	if m.TrashedAt == 0 {
		return nil, httpx.Fail(http.StatusConflict, "not_trashed", "the page is not in the trash")
	}
	pages, err := s.subtree(ctx, s.st.R, id, false)
	if err != nil {
		return nil, err
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		now := store.NowMS()
		parentID := m.ParentID
		if parentID != "" {
			var alive int
			if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM pages WHERE id = ? AND trashed_at = 0", parentID).Scan(&alive); err != nil {
				return err
			}
			if alive == 0 {
				parentID = ""
			}
		}
		if parentID != m.ParentID && m.Row {
			return httpx.Fail(http.StatusConflict, "database_trashed", "restore its database first")
		}
		if parentID != m.ParentID {
			key, err := s.slot(ctx, tx, m.WorkspaceID, "", "", "", id)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE pages SET parent_id = NULL, sort_key = ? WHERE id = ?", key, id); err != nil {
				return err
			}
		}
		for _, x := range pages {
			if x.trashBatch != m.trashBatch {
				continue
			}
			if _, err := tx.ExecContext(ctx, `UPDATE pages SET trashed_at = 0, trash_batch = '', trashed_by = '', trashed_by_kind = '', updated_at = ?, updated_by = ?
				WHERE id = ?`, now, p.UserID, x.ID); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.restore", WorkspaceID: m.WorkspaceID, PageID: id, Detail: m.Title})
	})
	if err != nil {
		return nil, err
	}
	restored, err := s.subtree(ctx, s.st.R, id, true)
	if err != nil {
		return nil, err
	}
	var rowIDs []string
	for i, x := range restored {
		if x.Row && i > 0 {
			rowIDs = append(rowIDs, x.ID)
			continue
		}
		s.emit(ctx, x)
	}
	s.notify(ctx, rowIDs)
	s.hub.Workspace(m.WorkspaceID, "trash")
	if len(restored) == 0 {
		return nil, store.ErrNotFound
	}
	return restored[0], nil
}

func (s *Service) DeleteForever(ctx context.Context, p *policy.Principal, id string) error {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return err
	}
	if _, err := s.check(ctx, s.st.R, p, m, LevelEdit); err != nil {
		return err
	}
	if m.TrashedAt == 0 {
		return httpx.Fail(http.StatusConflict, "not_trashed", "move the page to the trash first")
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM pages WHERE id = ?", id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "page.delete", WorkspaceID: m.WorkspaceID, PageID: id, Detail: m.Title})
	})
	if err != nil {
		return err
	}
	s.hub.Workspace(m.WorkspaceID, "trash")
	s.notify(ctx, []string{id})
	return nil
}

func (s *Service) EmptyTrash(ctx context.Context, p *policy.Principal, workspaceID string) (int, error) {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return 0, err
	}
	if !a.IsAdmin() && !a.Personal {
		return 0, httpx.Fail(http.StatusForbidden, "admins_only", "only an admin of this workspace can empty its trash")
	}
	items, err := s.Trash(ctx, p, workspaceID, "")
	if err != nil {
		return 0, err
	}
	n := 0
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		for _, it := range items {
			res, err := tx.ExecContext(ctx, "DELETE FROM pages WHERE id = ? AND trashed_at > 0", it.ID)
			if err != nil {
				return err
			}
			c, _ := res.RowsAffected()
			n += int(c)
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "trash.empty", WorkspaceID: workspaceID})
	})
	if err != nil {
		return 0, err
	}
	s.hub.Workspace(workspaceID, "trash")
	return n, nil
}

func (s *Service) Purge(ctx context.Context) (int, error) {
	days := s.settings.Int("trash.days")
	if days <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()
	rows, err := s.st.R.QueryContext(ctx, `SELECT p.id, p.workspace_id FROM pages p
		LEFT JOIN pages parent ON parent.id = p.parent_id
		WHERE p.trashed_at > 0 AND p.trashed_at < ? AND (parent.id IS NULL OR parent.trash_batch != p.trash_batch)`, cutoff)
	if err != nil {
		return 0, err
	}
	type root struct{ id, ws string }
	var roots []root
	for rows.Next() {
		var r root
		if err := rows.Scan(&r.id, &r.ws); err != nil {
			rows.Close()
			return 0, err
		}
		roots = append(roots, r)
	}
	rows.Close()
	touched := map[string]bool{}
	for _, r := range roots {
		if _, err := s.st.W.ExecContext(ctx, "DELETE FROM pages WHERE id = ?", r.id); err != nil {
			return 0, err
		}
		if err := s.audit.Record(ctx, nil, nil, audit.Entry{Action: "page.purge", WorkspaceID: r.ws, PageID: r.id}); err != nil {
			return 0, err
		}
		touched[r.ws] = true
	}
	for ws := range touched {
		s.hub.Workspace(ws, "trash")
	}
	return len(roots), nil
}
