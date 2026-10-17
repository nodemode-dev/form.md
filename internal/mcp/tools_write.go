package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/comment"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
)

var whereProp = enum("Where to put the blocks: end of the page (default), start, before or after the anchor block, or inside_start or inside_end of a container block such as a toggle, quote, callout or list", "end", "start", "before", "after", "inside_start", "inside_end")

func (s *Server) registerWrites() {
	s.add(&Tool{Name: "create_page", Title: "Create a page", Write: true,
		Description: "A new page at the top of a workspace or inside a page, optionally with a Markdown body.",
		Props:       map[string]any{"title": str("Title"), "workspace_id": str("Workspace (needed without parent_id when several are reachable)"), "parent_id": str("Create inside this page"), "markdown": str("Body in Markdown")},
		Required:    []string{"title"}, Run: s.createPage})
	s.add(&Tool{Name: "update_page", Title: "Change a page's title, description or tags", Write: true, Idempotent: true,
		Description: "Change the title, description or tags of a page; fields left out stay as they are.",
		Props:       map[string]any{"page_id": str("Page id"), "title": str("New title"), "description": str("New description"), "tags": strs("Replace the tags")},
		Required:    []string{"page_id"}, Run: s.updatePage})
	s.add(&Tool{Name: "move_page", Title: "Move a page", Write: true, Idempotent: true,
		Description: "Move a page and everything inside it under another page, or to the top level of its workspace when parent_id is empty.",
		Props:       map[string]any{"page_id": str("Page id"), "parent_id": str("New parent page id; empty string for the top level"), "after_id": str("Place after this sibling")},
		Required:    []string{"page_id", "parent_id"}, Run: s.movePage})
	s.add(&Tool{Name: "trash_page", Title: "Move a page to the trash", Write: true, Destructive: true,
		Description: "Move a page and everything inside it to the trash, where people can restore it. undo_run restores it too.",
		Props:       map[string]any{"page_id": str("Page id")}, Required: []string{"page_id"}, Run: s.trashPage})
	s.add(&Tool{Name: "insert_blocks", Title: "Insert blocks", Write: true,
		Description: "Insert Markdown as new blocks into a page, merged with whatever people are typing. Place them with where and anchor (a block id from get_page), or at the end of a section named by its heading text (\"Heading\" or \"Parent > Child\").",
		Props:       map[string]any{"page_id": str("Page id"), "markdown": str("Markdown to insert"), "where": whereProp, "anchor": str("Block id that where refers to"), "section": str("Insert at the end of the section under this heading")},
		Required:    []string{"page_id", "markdown"}, Run: s.insertBlocks})
	s.add(&Tool{Name: "replace_block", Title: "Replace a block", Write: true, Destructive: true, Idempotent: true,
		Description: "Replace one block with Markdown. The first new block keeps the block id. Text that people type elsewhere in the page is kept. Refused (person_editing) while a person's cursor is in that block; insert next to it instead or try again later.",
		Props:       map[string]any{"page_id": str("Page id"), "block_id": str("Block id"), "markdown": str("New Markdown for the block")},
		Required:    []string{"page_id", "block_id", "markdown"}, Run: s.replaceBlock})
	s.add(&Tool{Name: "delete_blocks", Title: "Delete blocks", Write: true, Destructive: true, Idempotent: true,
		Description: "Delete blocks by id. Deleting the only item of a list removes the list; deleting the first paragraph of a list item removes the item.",
		Props:       map[string]any{"page_id": str("Page id"), "block_ids": strs("Block ids")},
		Required:    []string{"page_id", "block_ids"}, Run: s.deleteBlocks})
	s.add(&Tool{Name: "move_block", Title: "Move a block", Write: true, Idempotent: true,
		Description: "Move a block, with everything inside it, to another place in the same page.",
		Props:       map[string]any{"page_id": str("Page id"), "block_id": str("Block to move"), "where": whereProp, "anchor": str("Block id that where refers to"), "section": str("Move to the end of the section under this heading")},
		Required:    []string{"page_id", "block_id"}, Run: s.moveBlock})
	s.add(&Tool{Name: "replace_page_content", Title: "Replace a page's whole body", Write: true, Destructive: true, Idempotent: true,
		Description: "Replace every block of a page with new Markdown. Prefer insert_blocks and replace_block for targeted edits. Refused (person_editing) while a person is editing the page.",
		Props:       map[string]any{"page_id": str("Page id"), "markdown": str("The new body")},
		Required:    []string{"page_id", "markdown"}, Run: s.replaceContent})
	s.add(&Tool{Name: "create_database", Title: "Create a database", Write: true,
		Description: "A new database page. Each property has a name and a type: text, number, select, multi_select, status, date, person, files, checkbox, url, email, phone, relation (with database, the target database id, or self), formula (with formula), place. A title property called Name is added when none is given. select, multi_select and status take options.",
		Props: map[string]any{"title": str("Database title"), "workspace_id": str("Workspace"), "parent_id": str("Create inside this page"),
			"properties": map[string]any{"type": "array", "items": obj(map[string]any{"name": str("Property name"), "type": str("Property type"), "options": strs("Option names"), "database": str("Relation target database id, or self"), "formula": str("Formula expression")}, "name", "type")},
			"view":       enum("First view", "table", "board", "list", "gallery", "calendar", "timeline")},
		Required: []string{"title"}, Run: s.createDatabase})
	s.add(&Tool{Name: "update_database_properties", Title: "Add, rename or remove database properties", Write: true, Destructive: true,
		Description: "Add properties, rename them, or remove them (removing deletes their values in every row and cannot be undone).",
		Props: map[string]any{"database_id": str("Database id"),
			"add":    map[string]any{"type": "array", "items": obj(map[string]any{"name": str("Name"), "type": str("Type"), "options": strs("Option names"), "database": str("Relation target"), "formula": str("Formula")}, "name", "type")},
			"rename": map[string]any{"type": "array", "items": obj(map[string]any{"property": str("Current name or id"), "name": str("New name")}, "property", "name")},
			"remove": strs("Property names or ids to remove")},
		Required: []string{"database_id"}, Run: s.updateSchema})
	s.add(&Tool{Name: "add_rows", Title: "Add database rows", Write: true,
		Description: "Add up to 100 rows. values is keyed by property name: text and numbers as is, select and status by option name (a new name adds the option), multi_select as a list of names, date as YYYY-MM-DD or YYYY-MM-DDTHH:MM or {\"start\",\"end\"}, person as a list of member emails or ids, checkbox as true or false, relation as a list of row page ids. markdown becomes the row page's body.",
		Props:       map[string]any{"database_id": str("Database id"), "rows": map[string]any{"type": "array", "items": obj(map[string]any{"title": str("Row title"), "values": map[string]any{"type": "object"}, "markdown": str("Row page body")})}},
		Required:    []string{"database_id", "rows"}, Run: s.addRows})
	s.add(&Tool{Name: "update_rows", Title: "Change database rows", Write: true, Idempotent: true,
		Description: "Change the title or property values of up to 100 rows; values work as in add_rows and null clears a value.",
		Props:       map[string]any{"database_id": str("Database id"), "rows": map[string]any{"type": "array", "items": obj(map[string]any{"row_id": str("Row page id"), "title": str("New title"), "values": map[string]any{"type": "object"}}, "row_id")}},
		Required:    []string{"database_id", "rows"}, Run: s.updateRows})
	s.add(&Tool{Name: "add_comment", Title: "Comment on a page", Write: true,
		Description: "Start a discussion on a page, or reply to a thread from list_comments.",
		Props:       map[string]any{"page_id": str("Page id"), "body": str("Comment text"), "thread_id": str("Reply to this thread")},
		Required:    []string{"page_id", "body"}, Run: s.addComment})
	s.add(&Tool{Name: "undo_run", Title: "Undo a run", Write: true, Destructive: true,
		Description: "Revert every change of a run made by this person's agents, newest first: block edits are reversed, created pages and rows go to the trash, changed values return. Removing a property cannot be undone.",
		Props:       map[string]any{"target_run_id": str("The run to undo")}, Required: []string{"target_run_id"}, Run: s.undoRun})
	s.add(&Tool{Name: "propose_rules", Title: "Propose workspace rules", Write: true,
		Description: "Propose new written rules for a workspace. Only an admin's agent may propose; the rules change only when an admin accepts the proposal in the app.",
		Props:       map[string]any{"workspace_id": str("Workspace id"), "rules": str("The proposed rules, in full")},
		Required:    []string{"workspace_id", "rules"}, Run: s.proposeRules})
	s.registerUndo()
}

func (s *Server) defaultWorkspace(ctx context.Context, x *exec, ws, parent string) (string, error) {
	if parent != "" {
		m, err := s.Pages.Get(ctx, x.p, parent)
		if err != nil {
			return "", err
		}
		return m.WorkspaceID, nil
	}
	if ws != "" {
		return ws, nil
	}
	list, err := s.workspaces(ctx, x, "")
	if err != nil {
		return "", err
	}
	if len(list) != 1 {
		return "", fail("this credential reaches several workspaces; pass workspace_id or parent_id (see whoami)")
	}
	return list[0], nil
}

func (s *Server) createPage(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		Title       string `json:"title"`
		WorkspaceID string `json:"workspace_id"`
		ParentID    string `json:"parent_id"`
		Markdown    string `json:"markdown"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	ws, err := s.defaultWorkspace(ctx, x, in.WorkspaceID, in.ParentID)
	if err != nil {
		return nil, "", err
	}
	m, err := s.Pages.Create(ctx, x.p, page.CreateInput{WorkspaceID: ws, ParentID: in.ParentID, Title: in.Title})
	if err != nil {
		return nil, "", err
	}
	if err := x.record(ctx, "page.create", m.ID, "created "+m.Title, map[string]any{}); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(in.Markdown) != "" {
		if err := s.Docs.Write(ctx, x.p, m.ID, doc.WriteInput{Markdown: in.Markdown, Mode: "replace"}); err != nil {
			return nil, "", err
		}
	}
	return map[string]any{"page_id": m.ID, "workspace_id": m.WorkspaceID, "title": m.Title}, fmt.Sprintf("Created page %q (%s)", m.Title, m.ID), nil
}

func (s *Server) updatePage(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID      string    `json:"page_id"`
		Title       *string   `json:"title"`
		Description *string   `json:"description"`
		Tags        *[]string `json:"tags"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	before, err := s.Pages.Get(ctx, x.p, in.PageID)
	if err != nil {
		return nil, "", err
	}
	m, err := s.Pages.Update(ctx, x.p, in.PageID, page.UpdateInput{Title: in.Title, Description: in.Description, Tags: in.Tags})
	if err != nil {
		return nil, "", err
	}
	undo := map[string]any{}
	if in.Title != nil {
		undo["title"] = before.Title
	}
	if in.Description != nil {
		undo["description"] = before.Description
	}
	if in.Tags != nil {
		undo["tags"] = before.Tags
	}
	if err := x.record(ctx, "page.update", m.ID, "changed "+m.Title, undo); err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": m.ID, "title": m.Title, "description": m.Description, "tags": m.Tags}, fmt.Sprintf("Updated %q", m.Title), nil
}

func (s *Server) siblingBefore(ctx context.Context, p *policy.Principal, m *page.Meta) string {
	tree, err := s.Pages.Tree(ctx, p, m.WorkspaceID)
	if err != nil {
		return ""
	}
	var sibs []*page.Meta
	for _, t := range tree {
		if t.ParentID == m.ParentID && !t.Row {
			sibs = append(sibs, t)
		}
	}
	sort.Slice(sibs, func(i, j int) bool { return sibs[i].SortKey < sibs[j].SortKey })
	prev := ""
	for _, t := range sibs {
		if t.ID == m.ID {
			return prev
		}
		prev = t.ID
	}
	return ""
}

func (s *Server) movePage(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID   string `json:"page_id"`
		ParentID string `json:"parent_id"`
		AfterID  string `json:"after_id"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	before, err := s.Pages.Get(ctx, x.p, in.PageID)
	if err != nil {
		return nil, "", err
	}
	after := s.siblingBefore(ctx, x.p, before.Meta)
	parent := in.ParentID
	m, err := s.Pages.Move(ctx, x.p, in.PageID, page.MoveInput{ParentID: &parent, AfterID: in.AfterID})
	if err != nil {
		return nil, "", err
	}
	if err := x.record(ctx, "page.move", m.ID, "moved "+m.Title, map[string]any{"parent_id": before.ParentID, "after_id": after}); err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": m.ID, "parent_id": m.ParentID}, fmt.Sprintf("Moved %q", m.Title), nil
}

func (s *Server) trashPage(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID string `json:"page_id"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	trashed, err := s.Pages.TrashPage(ctx, x.p, in.PageID)
	if err != nil {
		return nil, "", err
	}
	if err := x.record(ctx, "page.trash", in.PageID, "trashed a page", map[string]any{}); err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": in.PageID, "trashed": len(trashed)}, fmt.Sprintf("Moved %d pages to the trash", len(trashed)), nil
}

func (s *Server) applyBlocks(ctx context.Context, x *exec, pageID string, edits []doc.Edit, label string) (*doc.EditResult, error) {
	if _, err := x.openRun(ctx); err != nil {
		return nil, err
	}
	res, err := s.Docs.Apply(ctx, x.p, pageID, edits, x.agent)
	if err != nil {
		return nil, err
	}
	if err := x.record(ctx, "blocks", pageID, label, map[string]any{"inverse": res.Inverse}); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Server) insertBlocks(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID   string `json:"page_id"`
		Markdown string `json:"markdown"`
		Where    string `json:"where"`
		Anchor   string `json:"anchor"`
		Section  string `json:"section"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	res, err := s.applyBlocks(ctx, x, in.PageID, []doc.Edit{{Op: "insert", Markdown: in.Markdown, Where: in.Where, Anchor: in.Anchor, Section: in.Section}}, "inserted blocks")
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": in.PageID, "blocks": res.Blocks}, fmt.Sprintf("Inserted %d blocks", len(res.Blocks)), nil
}

func (s *Server) replaceBlock(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID   string `json:"page_id"`
		BlockID  string `json:"block_id"`
		Markdown string `json:"markdown"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	res, err := s.applyBlocks(ctx, x, in.PageID, []doc.Edit{{Op: "replace", Block: in.BlockID, Markdown: in.Markdown}}, "replaced a block")
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": in.PageID, "blocks": res.Blocks}, "Replaced the block", nil
}

func (s *Server) deleteBlocks(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID   string   `json:"page_id"`
		BlockIDs []string `json:"block_ids"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if _, err := s.applyBlocks(ctx, x, in.PageID, []doc.Edit{{Op: "delete", Blocks: in.BlockIDs}}, "deleted blocks"); err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": in.PageID, "deleted": in.BlockIDs}, fmt.Sprintf("Deleted %d blocks", len(in.BlockIDs)), nil
}

func (s *Server) moveBlock(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID  string `json:"page_id"`
		BlockID string `json:"block_id"`
		Where   string `json:"where"`
		Anchor  string `json:"anchor"`
		Section string `json:"section"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if _, err := s.applyBlocks(ctx, x, in.PageID, []doc.Edit{{Op: "move", Block: in.BlockID, Where: in.Where, Anchor: in.Anchor, Section: in.Section}}, "moved a block"); err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": in.PageID, "block_id": in.BlockID}, "Moved the block", nil
}

func (s *Server) replaceContent(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID   string `json:"page_id"`
		Markdown string `json:"markdown"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	old, err := s.Docs.Outline(ctx, x.p, in.PageID, false)
	if err != nil {
		return nil, "", err
	}
	edits := []doc.Edit{{Op: "insert", Markdown: in.Markdown, Where: "end"}}
	if len(old) > 0 {
		ids := make([]string, 0, len(old))
		for _, b := range old {
			ids = append(ids, b.ID)
		}
		edits = append(edits, doc.Edit{Op: "delete", Blocks: ids})
	}
	res, err := s.applyBlocks(ctx, x, in.PageID, edits, "replaced the body")
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"page_id": in.PageID, "blocks": res.Blocks}, fmt.Sprintf("Replaced the body with %d blocks", len(res.Blocks)), nil
}

type propIn struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Options  json.RawMessage `json:"options"`
	Database string          `json:"database"`
	Formula  string          `json:"formula"`
}

func (pi propIn) property() *collection.Property {
	p := &collection.Property{Name: pi.Name, Type: strings.ToLower(strings.TrimSpace(pi.Type)), Collection: pi.Database, Expr: pi.Formula}
	switch p.Type {
	case "multi-select", "multiselect", "tags":
		p.Type = collection.TMulti
	case "title", "name":
		p.Type = collection.TTitle
	}
	var names []string
	if json.Unmarshal(pi.Options, &names) == nil {
		for _, n := range names {
			p.Options = append(p.Options, collection.Option{Name: n})
		}
	} else {
		var objs []collection.Option
		if json.Unmarshal(pi.Options, &objs) == nil {
			p.Options = objs
		}
	}
	return p
}

func (s *Server) createDatabase(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		Title       string   `json:"title"`
		WorkspaceID string   `json:"workspace_id"`
		ParentID    string   `json:"parent_id"`
		Properties  []propIn `json:"properties"`
		View        string   `json:"view"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	ws, err := s.defaultWorkspace(ctx, x, in.WorkspaceID, in.ParentID)
	if err != nil {
		return nil, "", err
	}
	schema := collection.Schema{}
	for _, pi := range in.Properties {
		schema = append(schema, pi.property())
	}
	extra := map[string]any{"view": in.View}
	if len(schema) > 0 {
		extra["schema"] = schema
	}
	ex, _ := json.Marshal(extra)
	m, err := s.Pages.Create(ctx, x.p, page.CreateInput{WorkspaceID: ws, ParentID: in.ParentID, Title: in.Title, Kind: "collection", Extra: ex})
	if err != nil {
		return nil, "", err
	}
	if err := x.record(ctx, "page.create", m.ID, "created database "+m.Title, map[string]any{}); err != nil {
		return nil, "", err
	}
	c, err := s.Cols.Get(ctx, x.p, m.ID)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"database_id": m.ID, "title": m.Title, "properties": schemaOut(c.Schema)}, fmt.Sprintf("Created database %q (%s)", m.Title, m.ID), nil
}

func findProp(sc collection.Schema, key string) *collection.Property {
	if p := sc.ByID(key); p != nil {
		return p
	}
	if p := sc.ByName(key); p != nil {
		return p
	}
	for _, p := range sc {
		if strings.EqualFold(p.Name, key) {
			return p
		}
	}
	return nil
}

func (s *Server) updateSchema(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		DatabaseID string   `json:"database_id"`
		Add        []propIn `json:"add"`
		Rename     []struct {
			Property string `json:"property"`
			Name     string `json:"name"`
		} `json:"rename"`
		Remove []string `json:"remove"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	c, err := s.Cols.Get(ctx, x.p, in.DatabaseID)
	if err != nil {
		return nil, "", err
	}
	var ops []collection.SchemaOp
	var renames []map[string]string
	for _, pi := range in.Add {
		ops = append(ops, collection.SchemaOp{Op: "add", Prop: pi.property()})
	}
	for _, rn := range in.Rename {
		p := findProp(c.Schema, rn.Property)
		if p == nil {
			return nil, "", fail("no property " + rn.Property)
		}
		np := *p
		np.Name = rn.Name
		ops = append(ops, collection.SchemaOp{Op: "update", ID: p.ID, Prop: &np})
		renames = append(renames, map[string]string{"id": p.ID, "name": p.Name})
	}
	removed := 0
	for _, key := range in.Remove {
		p := findProp(c.Schema, key)
		if p == nil {
			return nil, "", fail("no property " + key)
		}
		ops = append(ops, collection.SchemaOp{Op: "delete", ID: p.ID})
		removed++
	}
	if len(ops) == 0 {
		return nil, "", fail("nothing to change")
	}
	before := map[string]bool{}
	for _, p := range c.Schema {
		before[p.ID] = true
	}
	sc, err := s.Cols.UpdateSchema(ctx, x.p, c.ID, ops)
	if err != nil {
		return nil, "", err
	}
	var added []string
	for _, p := range sc {
		if !before[p.ID] {
			added = append(added, p.ID)
		}
	}
	if err := x.record(ctx, "schema", c.ID, "changed properties of "+c.Title, map[string]any{"added": added, "renames": renames, "removed": removed}); err != nil {
		return nil, "", err
	}
	return map[string]any{"database_id": c.ID, "properties": schemaOut(sc)}, fmt.Sprintf("Changed the properties of %q", c.Title), nil
}

func addsOptions(c *collection.Out, vals map[string]json.RawMessage) bool {
	for _, p := range c.Schema {
		if _, ok := vals[p.ID]; ok && (p.Type == collection.TSelect || p.Type == collection.TMulti || p.Type == collection.TStatus) {
			return true
		}
	}
	return false
}

func (s *Server) rowValues(ctx context.Context, x *exec, c *collection.Out, in map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	var people map[string]string
	for k, v := range in {
		p := findProp(c.Schema, k)
		if p == nil {
			return nil, fail("no property " + k + " in this database")
		}
		if p.Type == collection.TPerson && !isNullJSON(v) {
			if people == nil {
				people = map[string]string{}
				if list, err := s.Workspace.Members(ctx, x.p, c.WorkspaceID); err == nil {
					for _, m := range list {
						people[strings.ToLower(m.Name)] = m.UserID
						people[m.UserID] = m.UserID
					}
					if rows, err := s.Store.R.QueryContext(ctx, "SELECT u.id, u.email FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = ?", c.WorkspaceID); err == nil {
						for rows.Next() {
							var id, email string
							if rows.Scan(&id, &email) == nil {
								people[strings.ToLower(email)] = id
							}
						}
						rows.Close()
					}
				}
			}
			var list []string
			if json.Unmarshal(v, &list) != nil {
				var one string
				if json.Unmarshal(v, &one) != nil {
					return nil, fail(p.Name + " takes a list of member emails")
				}
				list = []string{one}
			}
			ids := []string{}
			for _, e := range list {
				id := people[strings.ToLower(strings.TrimSpace(e))]
				if id == "" {
					return nil, fail(e + " is not a member of this workspace")
				}
				ids = append(ids, id)
			}
			v, _ = json.Marshal(ids)
		}
		if (p.Type == collection.TSelect || p.Type == collection.TStatus) && !isNullJSON(v) {
			var name string
			if json.Unmarshal(v, &name) == nil && p.Option(name) == nil {
				found := false
				for _, o := range p.Options {
					if strings.EqualFold(o.Name, name) {
						v, _ = json.Marshal(o.ID)
						found = true
					}
				}
				if !found {
					v, _ = json.Marshal(map[string]string{"new": name})
				}
			}
		}
		if p.Type == collection.TMulti && !isNullJSON(v) {
			var names []string
			if json.Unmarshal(v, &names) == nil {
				items := []any{}
				for _, n := range names {
					switch {
					case p.Option(n) != nil:
						items = append(items, n)
					default:
						id := ""
						for _, o := range p.Options {
							if strings.EqualFold(o.Name, n) {
								id = o.ID
							}
						}
						if id != "" {
							items = append(items, id)
						} else {
							items = append(items, map[string]string{"new": n})
						}
					}
				}
				v, _ = json.Marshal(items)
			}
		}
		out[p.ID] = v
	}
	return out, nil
}

func isNullJSON(v json.RawMessage) bool {
	t := strings.TrimSpace(string(v))
	return t == "" || t == "null"
}

func (s *Server) addRows(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		DatabaseID string `json:"database_id"`
		Rows       []struct {
			Title    string                     `json:"title"`
			Values   map[string]json.RawMessage `json:"values"`
			Markdown string                     `json:"markdown"`
		} `json:"rows"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if len(in.Rows) == 0 || len(in.Rows) > 100 {
		return nil, "", fail("send between 1 and 100 rows")
	}
	c, err := s.Cols.Get(ctx, x.p, in.DatabaseID)
	if err != nil {
		return nil, "", err
	}
	var made []string
	for _, r := range in.Rows {
		vals, err := s.rowValues(ctx, x, c, r.Values)
		if err != nil {
			return s.partialRows(ctx, x, c, made, err)
		}
		out, err := s.Cols.CreateRow(ctx, x.p, c.ID, collection.RowInput{Title: r.Title, Values: vals})
		if err != nil {
			return s.partialRows(ctx, x, c, made, err)
		}
		made = append(made, out.ID)
		if strings.TrimSpace(r.Markdown) != "" {
			if err := s.Docs.Write(ctx, x.p, out.ID, doc.WriteInput{Markdown: r.Markdown, Mode: "replace"}); err != nil {
				return s.partialRows(ctx, x, c, made, err)
			}
		}
		if addsOptions(c, vals) {
			c, _ = s.Cols.Get(ctx, x.p, c.ID)
		}
	}
	if err := x.record(ctx, "rows.add", c.ID, fmt.Sprintf("added %d rows to %s", len(made), c.Title), map[string]any{"rows": made}); err != nil {
		return nil, "", err
	}
	return map[string]any{"database_id": c.ID, "rows": made}, fmt.Sprintf("Added %d rows to %q", len(made), c.Title), nil
}

func (s *Server) partialRows(ctx context.Context, x *exec, c *collection.Out, made []string, err error) (any, string, error) {
	if len(made) > 0 {
		_ = x.record(ctx, "rows.add", c.ID, fmt.Sprintf("added %d rows to %s", len(made), c.Title), map[string]any{"rows": made})
		return nil, "", fail(fmt.Sprintf("added %d rows (%s), then stopped: %s", len(made), strings.Join(made, ", "), errText(err)))
	}
	return nil, "", err
}

func (s *Server) updateRows(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		DatabaseID string `json:"database_id"`
		Rows       []struct {
			RowID  string                     `json:"row_id"`
			Title  *string                    `json:"title"`
			Values map[string]json.RawMessage `json:"values"`
		} `json:"rows"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if len(in.Rows) == 0 || len(in.Rows) > 100 {
		return nil, "", fail("send between 1 and 100 rows")
	}
	c, err := s.Cols.Get(ctx, x.p, in.DatabaseID)
	if err != nil {
		return nil, "", err
	}
	ids := make([]string, 0, len(in.Rows))
	for _, r := range in.Rows {
		ids = append(ids, r.RowID)
	}
	found, err := s.Cols.Query(ctx, x.p, c.ID, collection.QueryInput{IDs: ids, Limit: len(ids)})
	if err != nil {
		return nil, "", err
	}
	current := map[string]collection.RowOut{}
	for _, row := range found.Rows {
		current[row.ID] = row
	}
	done := 0
	for _, r := range in.Rows {
		before, ok := current[r.RowID]
		if !ok {
			return nil, "", fail(r.RowID + " is not a row of this database")
		}
		vals, err := s.rowValues(ctx, x, c, r.Values)
		if err != nil {
			return nil, "", err
		}
		old := map[string]any{}
		for id := range vals {
			old[id] = before.Values[id]
		}
		undo := map[string]any{"values": old}
		if r.Title != nil {
			undo["title"] = before.Title
		}
		if _, err := s.Cols.UpdateRow(ctx, x.p, r.RowID, collection.RowPatch{Title: r.Title, Values: vals}); err != nil {
			return nil, "", err
		}
		if err := x.record(ctx, "row.update", r.RowID, "changed a row of "+c.Title, undo); err != nil {
			return nil, "", err
		}
		done++
		if addsOptions(c, vals) {
			c, _ = s.Cols.Get(ctx, x.p, c.ID)
		}
	}
	return map[string]any{"database_id": c.ID, "updated": done}, fmt.Sprintf("Changed %d rows of %q", done, c.Title), nil
}

func (s *Server) addComment(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		PageID   string `json:"page_id"`
		Body     string `json:"body"`
		ThreadID string `json:"thread_id"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if in.ThreadID != "" {
		c, err := s.Comments.Reply(ctx, x.p, in.ThreadID, in.Body)
		if err != nil {
			return nil, "", err
		}
		if err := x.record(ctx, "comment.add", in.PageID, "replied to a thread", map[string]any{"comment": c.ID}); err != nil {
			return nil, "", err
		}
		return map[string]any{"thread_id": in.ThreadID, "comment_id": c.ID}, "Replied", nil
	}
	t, err := s.Comments.Create(ctx, x.p, in.PageID, comment.CreateInput{Anchor: "page", Body: in.Body})
	if err != nil {
		return nil, "", err
	}
	cid := ""
	if len(t.Comments) > 0 {
		cid = t.Comments[0].ID
	}
	if err := x.record(ctx, "comment.add", in.PageID, "started a discussion", map[string]any{"comment": cid}); err != nil {
		return nil, "", err
	}
	return map[string]any{"thread_id": t.ID, "comment_id": cid}, "Started a discussion", nil
}

func (s *Server) undoRun(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		Target string `json:"target_run_id"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	n, err := s.Runs.Undo(ctx, x.p, in.Target)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"undone_run": in.Target, "steps": n}, fmt.Sprintf("Undid %d changes of %s", n, in.Target), nil
}

func (s *Server) proposeRules(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		Rules       string `json:"rules"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if err := s.Workspace.ProposeRules(ctx, x.p, in.WorkspaceID, in.Rules); err != nil {
		return nil, "", err
	}
	return map[string]any{"workspace_id": in.WorkspaceID, "proposed": true}, "Proposed new rules; an admin accepts them in the app", nil
}

func (s *Server) registerUndo() {
	r := s.Runs
	r.Handle("blocks", func(ctx context.Context, p *policy.Principal, pageID string, data json.RawMessage) error {
		var u struct {
			Inverse []doc.Edit `json:"inverse"`
		}
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		return s.Docs.Revert(ctx, p, pageID, u.Inverse)
	})
	r.Handle("page.create", func(ctx context.Context, p *policy.Principal, pageID string, _ json.RawMessage) error {
		_, err := s.Pages.TrashPage(ctx, p, pageID)
		return err
	})
	r.Handle("page.trash", func(ctx context.Context, p *policy.Principal, pageID string, _ json.RawMessage) error {
		_, err := s.Pages.Restore(ctx, p, pageID)
		return err
	})
	r.Handle("page.update", func(ctx context.Context, p *policy.Principal, pageID string, data json.RawMessage) error {
		var u page.UpdateInput
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		_, err := s.Pages.Update(ctx, p, pageID, u)
		return err
	})
	r.Handle("page.move", func(ctx context.Context, p *policy.Principal, pageID string, data json.RawMessage) error {
		var u struct {
			ParentID string `json:"parent_id"`
			AfterID  string `json:"after_id"`
		}
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		_, err := s.Pages.Move(ctx, p, pageID, page.MoveInput{ParentID: &u.ParentID, AfterID: u.AfterID})
		return err
	})
	r.Handle("rows.add", func(ctx context.Context, p *policy.Principal, colID string, data json.RawMessage) error {
		var u struct {
			Rows []string `json:"rows"`
		}
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		_, err := s.Cols.DeleteRows(ctx, p, colID, u.Rows)
		return err
	})
	r.Handle("row.update", func(ctx context.Context, p *policy.Principal, rowID string, data json.RawMessage) error {
		var u struct {
			Title  *string                    `json:"title"`
			Values map[string]json.RawMessage `json:"values"`
		}
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		_, err := s.Cols.UpdateRow(ctx, p, rowID, collection.RowPatch{Title: u.Title, Values: u.Values})
		return err
	})
	r.Handle("schema", func(ctx context.Context, p *policy.Principal, colID string, data json.RawMessage) error {
		var u struct {
			Added   []string            `json:"added"`
			Renames []map[string]string `json:"renames"`
		}
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		c, err := s.Cols.Get(ctx, p, colID)
		if err != nil {
			return err
		}
		var ops []collection.SchemaOp
		for _, id := range u.Added {
			if c.Schema.ByID(id) != nil {
				ops = append(ops, collection.SchemaOp{Op: "delete", ID: id})
			}
		}
		for _, rn := range u.Renames {
			if cur := c.Schema.ByID(rn["id"]); cur != nil {
				np := *cur
				np.Name = rn["name"]
				ops = append(ops, collection.SchemaOp{Op: "update", ID: cur.ID, Prop: &np})
			}
		}
		if len(ops) == 0 {
			return nil
		}
		_, err = s.Cols.UpdateSchema(ctx, p, colID, ops)
		return err
	})
	r.Handle("comment.add", func(ctx context.Context, p *policy.Principal, _ string, data json.RawMessage) error {
		var u struct {
			Comment string `json:"comment"`
		}
		if err := json.Unmarshal(data, &u); err != nil || u.Comment == "" {
			return err
		}
		return s.Comments.Delete(ctx, p, u.Comment)
	})
}
