package doc

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"strings"
	"time"

	"github.com/reearth/ygo/crdt"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type Node struct {
	Name  string         `json:"n"`
	Attrs map[string]any `json:"a,omitempty"`
	Kids  []*Node        `json:"k,omitempty"`
	Runs  []Run          `json:"r,omitempty"`
}

type Run struct {
	Text  string         `json:"t,omitempty"`
	Marks map[string]any `json:"m,omitempty"`
	Atom  *Node          `json:"x,omitempty"`
}

type Block struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Parent   string   `json:"parent,omitempty"`
	Depth    int      `json:"depth"`
	Section  []string `json:"section,omitempty"`
	Markdown string   `json:"markdown"`
}

type Edit struct {
	Op       string   `json:"op"`
	Markdown string   `json:"markdown,omitempty"`
	Nodes    []*Node  `json:"nodes,omitempty"`
	Block    string   `json:"block,omitempty"`
	Blocks   []string `json:"blocks,omitempty"`
	Where    string   `json:"where,omitempty"`
	Anchor   string   `json:"anchor,omitempty"`
	Section  string   `json:"section,omitempty"`
}

type EditResult struct {
	Blocks  []string `json:"blocks"`
	Inverse []Edit   `json:"-"`
}

type Agent struct {
	Key   string
	Name  string
	Color string
}

var listKinds = map[string]bool{"bullet_list": true, "ordered_list": true, "task_list": true}
var itemKinds = map[string]bool{"list_item": true, "task_item": true}
var flowParents = map[string]bool{"doc": true, "blockquote": true, "callout": true, "column": true, "toggle": true, "list_item": true, "task_item": true}

type parentNode interface {
	container
	Delete(txn *crdt.Transaction, index, length int)
	Len() int
}

func asParent(x any) parentNode {
	switch v := x.(type) {
	case *crdt.YXmlFragment:
		return v
	case *crdt.YXmlElement:
		return v
	}
	return nil
}

func toNode(n *pmNode) *Node {
	out := &Node{Name: n.name}
	if len(n.attrs) > 0 {
		out.Attrs = map[string]any{}
		for k, v := range n.attrs {
			out.Attrs[k] = v
		}
	}
	for _, k := range n.kids {
		out.Kids = append(out.Kids, toNode(k))
	}
	for _, r := range n.runs {
		x := Run{Text: r.text, Marks: r.marks}
		if r.atom != nil {
			x.Atom = toNode(r.atom)
		}
		out.Runs = append(out.Runs, x)
	}
	return out
}

func fromNode(n *Node) *pmNode {
	out := &pmNode{name: n.Name, attrs: map[string]any{}}
	for k, v := range n.Attrs {
		out.attrs[k] = v
	}
	for _, k := range n.Kids {
		if k != nil {
			out.kids = append(out.kids, fromNode(k))
		}
	}
	for _, r := range n.Runs {
		x := pmRun{text: r.Text, marks: r.Marks}
		if r.Atom != nil {
			x.atom = fromNode(r.Atom)
		}
		out.runs = append(out.runs, x)
	}
	return out
}

func readNode(el *crdt.YXmlElement) *pmNode {
	n := &pmNode{name: el.NodeName, attrs: map[string]any{}}
	for k, v := range el.GetAttributeValues() {
		n.attrs[k] = v
	}
	kids := children(el)
	inline := textblocks[el.NodeName]
	for _, k := range kids {
		if _, ok := k.(*crdt.YXmlText); ok {
			inline = true
		}
	}
	if !inline {
		for _, k := range kids {
			if ke, ok := k.(*crdt.YXmlElement); ok {
				n.kids = append(n.kids, readNode(ke))
			}
		}
		return n
	}
	for _, k := range kids {
		switch x := k.(type) {
		case *crdt.YXmlText:
			for _, d := range x.ToDelta() {
				s, ok := d.Insert.(string)
				if !ok || s == "" {
					continue
				}
				var marks map[string]any
				for mk, mv := range d.Attributes {
					if mv == nil {
						continue
					}
					if marks == nil {
						marks = map[string]any{}
					}
					marks[mk] = mv
				}
				n.runs = append(n.runs, pmRun{text: s, marks: marks})
			}
		case *crdt.YXmlElement:
			atom := &pmNode{name: x.NodeName, attrs: map[string]any{}}
			for ak, av := range x.GetAttributeValues() {
				atom.attrs[ak] = av
			}
			n.runs = append(n.runs, pmRun{atom: atom})
		}
	}
	return n
}

type located struct {
	parent any
	pname  string
	pid    string
	index  int
	el     *crdt.YXmlElement
	depth  int
}

func locate(frag *crdt.YXmlFragment, id string) *located {
	var walk func(parent any, pname, pid string, depth int) *located
	walk = func(parent any, pname, pid string, depth int) *located {
		for i, c := range children(parent) {
			el, ok := c.(*crdt.YXmlElement)
			if !ok {
				continue
			}
			eid := attrString(el, "id")
			if eid == id {
				return &located{parent: parent, pname: pname, pid: pid, index: i, el: el, depth: depth}
			}
			if atoms[el.NodeName] || textblocks[el.NodeName] {
				continue
			}
			next := pid
			if eid != "" {
				next = eid
			}
			if l := walk(el, el.NodeName, next, depth+1); l != nil {
				return l
			}
		}
		return nil
	}
	return walk(frag, "doc", "", 0)
}

func (l *located) self(frag *crdt.YXmlFragment) *located {
	if l.pid == "" {
		return nil
	}
	return locate(frag, l.pid)
}

func badEdit(msg string) error { return httpx.Fail(http.StatusBadRequest, "bad_edit", msg) }

func missingBlock(id string) error {
	return httpx.Fail(http.StatusNotFound, "block_not_found", "no block has the id "+id)
}

func elementIDs(nodes []*pmNode) []string {
	var out []string
	for _, n := range nodes {
		if id, _ := n.attrs["id"].(string); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func ensureIDs(nodes []*pmNode) {
	for _, n := range nodes {
		if withID[n.name] {
			if id, _ := n.attrs["id"].(string); id == "" {
				n.attrs["id"] = NewBlockID()
			}
		}
		ensureIDs(n.kids)
	}
}

func asItems(nodes []*pmNode, list string) ([]*pmNode, error) {
	item := "list_item"
	if list == "task_list" {
		item = "task_item"
	}
	var out []*pmNode
	for _, n := range nodes {
		switch {
		case listKinds[n.name]:
			for _, k := range n.kids {
				if k.name != item {
					k.name = item
					if item == "task_item" {
						k.attrs["checked"] = false
					} else {
						delete(k.attrs, "checked")
					}
				}
				out = append(out, k)
			}
		case n.name == "paragraph":
			it := newNode(item)
			if item == "task_item" {
				it.attrs["checked"] = false
			}
			it.kids = []*pmNode{n}
			out = append(out, it)
		default:
			return nil, badEdit("only list items can sit beside a list item; anchor on the list itself to place other blocks")
		}
	}
	return out, nil
}

func wrapItems(nodes []*pmNode) []*pmNode {
	var out []*pmNode
	var list *pmNode
	for _, n := range nodes {
		if !itemKinds[n.name] {
			list = nil
			out = append(out, n)
			continue
		}
		kind := "bullet_list"
		if n.name == "task_item" {
			kind = "task_list"
		}
		if list == nil || list.name != kind {
			list = newNode(kind)
			out = append(out, list)
		}
		list.kids = append(list.kids, n)
	}
	return out
}

func sectionIndex(frag *crdt.YXmlFragment, path string) (int, bool) {
	want := []string{}
	for _, s := range strings.Split(path, ">") {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			want = append(want, s)
		}
	}
	if len(want) == 0 {
		return 0, false
	}
	type head struct {
		level int
		text  string
	}
	var stack []head
	w := &mdWriter{pages: map[string]bool{}, files: map[string]bool{}, users: map[string]bool{}}
	kids := children(frag)
	start, level := -1, 0
	for i, c := range kids {
		el, ok := c.(*crdt.YXmlElement)
		if !ok || el.NodeName != "heading" {
			continue
		}
		lv := attrInt(el, "level", 1)
		if start >= 0 && lv <= level {
			return i, true
		}
		for len(stack) > 0 && stack[len(stack)-1].level >= lv {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, head{lv, strings.ToLower(strings.TrimSpace(w.plain(el)))})
		if start < 0 && len(stack) >= len(want) {
			match := true
			for j := range want {
				if stack[len(stack)-len(want)+j].text != want[j] {
					match = false
					break
				}
			}
			if match {
				start, level = i, lv
			}
		}
	}
	if start >= 0 {
		return len(kids), true
	}
	return 0, false
}

type editor struct {
	r     *room
	frag  *crdt.YXmlFragment
	who   any
	files FileLookup
	out   []string
	inv   []Edit
	last  string
}

func (e *editor) nodes(ed Edit) ([]*pmNode, error) {
	var nodes []*pmNode
	if len(ed.Nodes) > 0 {
		for _, n := range ed.Nodes {
			if n != nil && n.Name != "" {
				nodes = append(nodes, fromNode(n))
			}
		}
	} else {
		if len(ed.Markdown) > 4<<20 {
			return nil, httpx.Fail(http.StatusRequestEntityTooLarge, "too_large", "Markdown can be up to 4 MB per edit")
		}
		nodes = ParseMarkdown(ed.Markdown, e.files)
	}
	if len(nodes) == 0 {
		return nil, badEdit("the edit has no content")
	}
	ensureIDs(nodes)
	return nodes, nil
}

func (e *editor) anchorOf(l *located) Edit {
	kids := children(l.parent)
	for i := l.index - 1; i >= 0; i-- {
		if el, ok := kids[i].(*crdt.YXmlElement); ok {
			if id := attrString(el, "id"); id != "" {
				return Edit{Where: "after", Anchor: id}
			}
		}
	}
	for i := l.index + 1; i < len(kids); i++ {
		if el, ok := kids[i].(*crdt.YXmlElement); ok {
			if id := attrString(el, "id"); id != "" {
				return Edit{Where: "before", Anchor: id}
			}
		}
	}
	if l.pname == "doc" {
		return Edit{Where: "start"}
	}
	if l.pid != "" && l.pname != "column" {
		return Edit{Where: "inside_start", Anchor: l.pid}
	}
	return Edit{}
}

func (e *editor) target(ed Edit) (parentNode, string, int, error) {
	switch ed.Where {
	case "", "end":
		if ed.Section != "" {
			i, ok := sectionIndex(e.frag, ed.Section)
			if !ok {
				return nil, "", 0, httpx.Fail(http.StatusNotFound, "section_not_found", "no heading matches "+ed.Section)
			}
			return e.frag, "doc", i, nil
		}
		if ed.Anchor == "" {
			return e.frag, "doc", e.frag.Len(), nil
		}
	case "start":
		if ed.Anchor == "" {
			return e.frag, "doc", 0, nil
		}
	}
	if ed.Anchor == "" {
		return nil, "", 0, badEdit("where " + ed.Where + " needs an anchor block id")
	}
	l := locate(e.frag, ed.Anchor)
	if l == nil {
		return nil, "", 0, missingBlock(ed.Anchor)
	}
	switch ed.Where {
	case "before", "after":
		if itemKinds[l.pname] && l.index == 0 {
			if ed.Where == "before" {
				return nil, "", 0, badEdit("the first paragraph of a list item has nothing before it; anchor on the list item")
			}
		}
		i := l.index
		if ed.Where == "after" {
			i++
		}
		return asParent(l.parent), l.pname, i, nil
	case "inside_start", "inside_end", "start", "end":
		el := l.el
		name := el.NodeName
		if atoms[name] || textblocks[name] {
			return nil, "", 0, badEdit("a " + name + " block cannot hold other blocks")
		}
		if name == "columns" || name == "table" {
			return nil, "", 0, badEdit("anchor on a block inside the " + name + " instead")
		}
		end := ed.Where == "inside_end" || ed.Where == "end"
		i := 0
		if end {
			i = el.Len()
		} else if name == "toggle" || itemKinds[name] {
			i = 1
		}
		return el, name, i, nil
	}
	return nil, "", 0, badEdit("where is before, after, inside_start, inside_end, start or end")
}

func (e *editor) insert(ed Edit) error {
	nodes, err := e.nodes(ed)
	if err != nil {
		return err
	}
	parent, pname, at, err := e.target(ed)
	if err != nil {
		return err
	}
	if listKinds[pname] {
		if nodes, err = asItems(nodes, pname); err != nil {
			return err
		}
	} else if !flowParents[pname] {
		return badEdit("blocks cannot be placed inside a " + pname)
	} else {
		nodes = wrapItems(nodes)
	}
	e.tx(func(txn *crdt.Transaction) { writeNodes(txn, parent, at, nodes) })
	ids := elementIDs(nodes)
	e.out = append(e.out, ids...)
	if len(ids) > 0 {
		e.inv = append(e.inv, Edit{Op: "delete", Blocks: ids})
		e.last = ids[len(ids)-1]
	}
	return nil
}

func (e *editor) tx(fn func(txn *crdt.Transaction)) {
	e.r.doc.Transact(fn, e.who)
}

func (e *editor) escalate(l *located) *located {
	for {
		if itemKinds[l.pname] && l.index == 0 {
			if up := l.self(e.frag); up != nil {
				l = up
				continue
			}
		}
		p := asParent(l.parent)
		if (listKinds[l.pname] || l.pname == "blockquote" || l.pname == "callout") && p.Len() == 1 {
			if up := l.self(e.frag); up != nil {
				l = up
				continue
			}
		}
		return l
	}
}

func (e *editor) remove(l *located) string {
	p := asParent(l.parent)
	filled := ""
	e.tx(func(txn *crdt.Transaction) {
		p.Delete(txn, l.index, 1)
		if p.Len() == 0 && (l.pname == "doc" || l.pname == "column") {
			filler := newNode("paragraph")
			writeNodes(txn, p, 0, []*pmNode{filler})
			filled, _ = filler.attrs["id"].(string)
		}
	})
	return filled
}

func (e *editor) removable(l *located) error {
	if !flowParents[l.pname] && !listKinds[l.pname] && asParent(l.parent).Len() == 1 {
		return badEdit("a " + l.pname + " keeps at least one paragraph; replace it instead")
	}
	return nil
}

func (e *editor) delete(ed Edit) error {
	ids := ed.Blocks
	if ed.Block != "" {
		ids = append([]string{ed.Block}, ids...)
	}
	if len(ids) == 0 {
		return badEdit("name the blocks to delete")
	}
	for _, id := range ids {
		l := locate(e.frag, id)
		if l == nil {
			return missingBlock(id)
		}
		l = e.escalate(l)
		if err := e.removable(l); err != nil {
			return err
		}
		anchor := e.anchorOf(l)
		snap := toNode(readNode(l.el))
		filler := e.remove(l)
		switch {
		case anchor.Where != "":
			e.inv = append(e.inv, Edit{Op: "insert", Nodes: []*Node{snap}, Where: anchor.Where, Anchor: anchor.Anchor})
		case filler != "":
			e.inv = append(e.inv, Edit{Op: "replace", Block: filler, Nodes: []*Node{snap}})
		default:
			return badEdit("that block cannot be removed on its own")
		}
	}
	return nil
}

func (e *editor) replace(ed Edit) error {
	if ed.Block == "" {
		return badEdit("name the block to replace")
	}
	nodes, err := e.nodes(ed)
	if err != nil {
		return err
	}
	l := locate(e.frag, ed.Block)
	if l == nil {
		return missingBlock(ed.Block)
	}
	if itemKinds[l.pname] && l.index == 0 && (len(nodes) != 1 || nodes[0].name != "paragraph") {
		if up := l.self(e.frag); up != nil {
			l = up
		}
	}
	if listKinds[l.pname] {
		if nodes, err = asItems(nodes, l.pname); err != nil {
			return err
		}
	} else if !flowParents[l.pname] {
		for _, n := range nodes {
			if n.name != "paragraph" {
				return badEdit("a " + l.pname + " holds paragraphs only")
			}
		}
	}
	old := toNode(readNode(l.el))
	oldID := attrString(l.el, "id")
	if withID[nodes[0].name] && oldID != "" {
		nodes[0].attrs["id"] = oldID
	}
	p := asParent(l.parent)
	e.tx(func(txn *crdt.Transaction) {
		p.Delete(txn, l.index, 1)
		writeNodes(txn, p, l.index, nodes)
	})
	ids := elementIDs(nodes)
	e.out = append(e.out, ids...)
	if len(ids) > 0 {
		e.last = ids[len(ids)-1]
	}
	var extra []string
	for _, id := range ids {
		if id != oldID {
			extra = append(extra, id)
		}
	}
	if len(extra) > 0 {
		e.inv = append(e.inv, Edit{Op: "delete", Blocks: extra})
	}
	if oldID != "" && len(ids) > 0 && ids[0] == oldID {
		e.inv = append(e.inv, Edit{Op: "replace", Block: oldID, Nodes: []*Node{old}})
	}
	return nil
}

func (e *editor) move(ed Edit) error {
	if ed.Block == "" {
		return badEdit("name the block to move")
	}
	l := locate(e.frag, ed.Block)
	if l == nil {
		return missingBlock(ed.Block)
	}
	if itemKinds[l.pname] && l.index == 0 {
		if up := l.self(e.frag); up != nil {
			l = up
		}
	}
	if ed.Anchor == ed.Block || ed.Anchor == attrString(l.el, "id") {
		return badEdit("a block cannot move next to itself")
	}
	if ed.Anchor != "" {
		if a := locate(e.frag, ed.Anchor); a == nil {
			return missingBlock(ed.Anchor)
		} else if inside(l.el, ed.Anchor) {
			return badEdit("a block cannot move inside itself")
		}
	}
	if err := e.removable(l); err != nil {
		return err
	}
	back := e.anchorOf(l)
	if back.Where == "" {
		return badEdit("that block cannot be moved on its own")
	}
	snap := toNode(readNode(l.el))
	movedID := attrString(l.el, "id")
	saved := e.inv
	e.inv = append(e.inv, Edit{Op: "insert", Nodes: []*Node{snap}, Where: back.Where, Anchor: back.Anchor})
	e.remove(l)
	if err := e.insert(Edit{Nodes: []*Node{snap}, Where: ed.Where, Anchor: ed.Anchor, Section: ed.Section}); err != nil {
		return err
	}
	e.inv = append(saved, Edit{Op: "move", Block: movedID, Where: back.Where, Anchor: back.Anchor})
	return nil
}

func inside(el *crdt.YXmlElement, id string) bool {
	for _, c := range children(el) {
		if ce, ok := c.(*crdt.YXmlElement); ok {
			if attrString(ce, "id") == id || inside(ce, id) {
				return true
			}
		}
	}
	return false
}

func (s *Service) Apply(ctx context.Context, p *policy.Principal, pageID string, edits []Edit, agent *Agent) (*EditResult, error) {
	workspaceID, err := s.pages.CanWrite(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	if len(edits) == 0 {
		return nil, badEdit("no edits")
	}
	if len(edits) > 200 {
		return nil, badEdit("up to 200 edits per call")
	}
	files := s.fileLookup(ctx, workspaceID)
	res := &EditResult{Blocks: []string{}}
	var applyErr error
	err = s.withRoom(ctx, pageID, func(r *room) {
		who := &origin{user: p.UserID}
		e := &editor{r: r, frag: r.fragment(), files: files, who: who}
		for _, ed := range edits {
			var err error
			switch ed.Op {
			case "insert", "append":
				err = e.insert(ed)
			case "delete":
				err = e.delete(ed)
			case "replace":
				err = e.replace(ed)
			case "move":
				err = e.move(ed)
			default:
				err = badEdit("op is insert, replace, delete or move")
			}
			if err != nil {
				applyErr = err
				break
			}
		}
		if applyErr != nil {
			undo := &editor{r: r, frag: r.fragment(), files: files, who: who}
			for i := len(e.inv) - 1; i >= 0; i-- {
				_ = undo.run(e.inv[i])
			}
			return
		}
		res.Blocks = e.out
		for i := len(e.inv) - 1; i >= 0; i-- {
			res.Inverse = append(res.Inverse, e.inv[i])
		}
		if agent != nil {
			r.agentHere(agent, e.last)
		}
	})
	if err != nil {
		return nil, err
	}
	if applyErr != nil {
		return nil, applyErr
	}
	s.importRemote(p, pageID)
	return res, nil
}

func (e *editor) run(ed Edit) error {
	switch ed.Op {
	case "insert":
		return e.insert(ed)
	case "delete":
		return e.delete(ed)
	case "replace":
		return e.replace(ed)
	case "move":
		return e.move(ed)
	}
	return badEdit("unknown op")
}

func (s *Service) Revert(ctx context.Context, p *policy.Principal, pageID string, inverse []Edit) error {
	if _, err := s.pages.CanWrite(ctx, p, pageID); err != nil {
		return err
	}
	return s.withRoom(ctx, pageID, func(r *room) {
		e := &editor{r: r, frag: r.fragment(), files: func(string) (FileInfo, bool) { return FileInfo{}, false }, who: &origin{user: p.UserID}}
		for _, ed := range inverse {
			_ = e.run(ed)
		}
	})
}

func (s *Service) Outline(ctx context.Context, p *policy.Principal, pageID string, nested bool) ([]Block, error) {
	if err := s.pages.CanRead(ctx, p, pageID); err != nil {
		return nil, err
	}
	var refs []string
	if err := s.withRoom(ctx, pageID, func(r *room) { refs = r.summary(nil).Pages }); err != nil {
		return nil, err
	}
	titles := s.pages.Titles(ctx, p, refs)
	title := func(id string) string { return titles[id] }
	out := []Block{}
	err := s.withRoom(ctx, pageID, func(r *room) {
		w := &mdWriter{pages: map[string]bool{}, files: map[string]bool{}, users: map[string]bool{}, title: title, self: r.id, doc: r.doc}
		type head struct {
			level int
			text  string
		}
		var stack []head
		var walk func(parent any, pid string, depth int)
		walk = func(parent any, pid string, depth int) {
			for _, c := range children(parent) {
				el, ok := c.(*crdt.YXmlElement)
				if !ok {
					continue
				}
				id := attrString(el, "id")
				if id == "" {
					if nested && !atoms[el.NodeName] && !textblocks[el.NodeName] {
						walk(el, pid, depth)
					}
					continue
				}
				b := Block{ID: id, Type: el.NodeName, Parent: pid, Depth: depth}
				if depth == 0 {
					if el.NodeName == "heading" {
						lv := attrInt(el, "level", 1)
						for len(stack) > 0 && stack[len(stack)-1].level >= lv {
							stack = stack[:len(stack)-1]
						}
						for _, h := range stack {
							b.Section = append(b.Section, h.text)
						}
						stack = append(stack, head{lv, strings.TrimSpace(w.plain(el))})
					} else {
						for _, h := range stack {
							b.Section = append(b.Section, h.text)
						}
					}
				}
				b.Markdown = strings.TrimRight(strings.Join(w.block(el), "\n"), "\n")
				out = append(out, b)
				if nested && !atoms[el.NodeName] && !textblocks[el.NodeName] {
					walk(el, id, depth+1)
				}
			}
		}
		walk(r.fragment(), "", 0)
	})
	return out, err
}

func agentClient(key string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return (1 << 40) | (h.Sum64() & ((1 << 40) - 1))
}

func (r *room) agentHere(a *Agent, block string) {
	if a == nil || a.Key == "" {
		return
	}
	id := agentClient(a.Key)
	color := a.Color
	if color == "" {
		color = tintHex["teal"]
		if color == "" {
			color = "#2a9d8f"
		}
	}
	state := map[string]any{"user": map[string]any{"id": "agent:" + a.Key, "name": a.Name, "color": color, "agent": true}}
	if block != "" {
		if l := locate(r.fragment(), block); l != nil {
			var txt *crdt.YXmlText
			var last func(el *crdt.YXmlElement)
			last = func(el *crdt.YXmlElement) {
				for _, c := range children(el) {
					switch x := c.(type) {
					case *crdt.YXmlText:
						txt = x
					case *crdt.YXmlElement:
						last(x)
					}
				}
			}
			last(l.el)
			if txt != nil && txt.Len() > 0 {
				rp := crdt.CreateRelativePositionFromIndex(txt, txt.Len(), -1)
				if rp.Item != nil {
					pos := map[string]any{"type": nil, "tname": nil, "item": map[string]any{"client": uint64(rp.Item.Client), "clock": rp.Item.Clock}, "assoc": rp.Assoc}
					state["cursor"] = map[string]any{"anchor": pos, "head": pos}
				}
			}
		}
	}
	b, err := json.Marshal(state)
	if err != nil {
		return
	}
	e := r.aware[id]
	clock := uint64(1)
	if e != nil {
		clock = e.clock + 1
	}
	r.aware[id] = &awareEntry{clock: clock, state: b}
	frame := awarenessFrame(r.id, encodeAwareness([]awareUpdate{{id: id, clock: clock, state: b}}))
	for c := range r.subs {
		c.send(frame)
	}
	time.AfterFunc(20*time.Second, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		cur := r.aware[id]
		if cur == nil || cur.clock != clock {
			return
		}
		delete(r.aware, id)
		gone := awarenessFrame(r.id, encodeAwareness([]awareUpdate{{id: id, clock: clock + 1, state: []byte("null")}}))
		for c := range r.subs {
			c.send(gone)
		}
	})
}
