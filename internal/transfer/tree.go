package transfer

import (
	"bytes"
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type Node struct {
	Title    string
	Body     string
	Children []*Node
	Table    *Table
	ref      int
	id       string
}

type Table struct {
	Headers []string
	Types   []string
	Rows    [][]string
	Bodies  []*Node
	Board   string
}

type Asset struct {
	Name string
	Data []byte
	ref  int
	file string
	bad  bool
}

type Bundle struct {
	Roots    []*Node
	assets   []*Asset
	pages    []*Node
	warnings []string
	consumed map[string]bool
}

func (b *Bundle) Warn(msg string) {
	if len(b.warnings) < 200 {
		b.warnings = append(b.warnings, msg)
	}
}

func (b *Bundle) used(p string) {
	if b.consumed == nil {
		b.consumed = map[string]bool{}
	}
	b.consumed[p] = true
}

func (b *Bundle) AssetRef(a *Asset) string {
	if a.ref == 0 {
		b.assets = append(b.assets, a)
		a.ref = len(b.assets)
	}
	return "asset:" + strconv.Itoa(a.ref)
}

func (b *Bundle) PageRef(n *Node) string {
	if n.ref == 0 {
		b.pages = append(b.pages, n)
		n.ref = len(b.pages)
	}
	return "page:" + strconv.Itoa(n.ref)
}

func (b *Bundle) count() int {
	n := 0
	var walk func(list []*Node)
	walk = func(list []*Node) {
		for _, x := range list {
			if x == nil {
				continue
			}
			n++
			if x.Table != nil {
				n += len(x.Table.Rows)
				for _, r := range x.Table.Bodies {
					if r != nil {
						walk(r.Children)
					}
				}
			}
			walk(x.Children)
		}
	}
	walk(b.Roots)
	return n
}

type writer struct {
	s       *Service
	p       *policy.Principal
	ws      string
	b       *Bundle
	job     *Job
	written []*Node
}

const maxTitle = 400

func cleanTitle(t string) string {
	t = strings.Join(strings.Fields(t), " ")
	if t == "" {
		return ""
	}
	r := []rune(t)
	if len(r) > maxTitle {
		t = string(r[:maxTitle])
	}
	return t
}

func (w *writer) create(ctx context.Context, n *Node, parentID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	title := cleanTitle(n.Title)
	if n.Table != nil {
		t := n.Table
		if title == "" {
			title = "Untitled database"
		}
		id, rowIDs, err := w.s.cols.ImportTable(ctx, w.p, collection.TableInput{WorkspaceID: w.ws, ParentID: parentID, Title: title, Headers: t.Headers, Rows: t.Rows, Types: t.Types})
		if err != nil {
			w.b.Warn("Could not create the database " + title + ": " + err.Error())
			return nil
		}
		n.id = id
		w.job.addPage(id, parentID == w.job.ParentID)
		w.job.progress(1+len(rowIDs), len(rowIDs))
		if t.Board != "" {
			_ = w.s.cols.AddBoardView(ctx, w.p, id, "Board", t.Board)
		}
		for i, rid := range rowIDs {
			if i < len(t.Bodies) && t.Bodies[i] != nil {
				t.Bodies[i].id = rid
				w.written = append(w.written, t.Bodies[i])
				for _, k := range t.Bodies[i].Children {
					if err := w.create(ctx, k, rid); err != nil {
						return err
					}
				}
			}
		}
	} else {
		if title == "" {
			title = "Untitled"
		}
		m, err := w.s.pages.Create(ctx, w.p, page.CreateInput{WorkspaceID: w.ws, ParentID: parentID, Title: title})
		if err != nil {
			w.b.Warn("Could not create the page " + title + ": " + err.Error())
			return nil
		}
		n.id = m.ID
		w.job.addPage(m.ID, parentID == w.job.ParentID)
		w.job.progress(1, 0)
		w.written = append(w.written, n)
	}
	for _, k := range n.Children {
		if err := w.create(ctx, k, n.id); err != nil {
			return err
		}
	}
	return nil
}

var refRE = regexp.MustCompile(`\((asset|page):(\d+)([ )])`)

func (w *writer) body(ctx context.Context, n *Node) {
	if strings.TrimSpace(n.Body) == "" || n.id == "" {
		return
	}
	md := refRE.ReplaceAllStringFunc(n.Body, func(m string) string {
		sub := refRE.FindStringSubmatch(m)
		i, _ := strconv.Atoi(sub[2])
		if sub[1] == "page" {
			if i >= 1 && i <= len(w.b.pages) && w.b.pages[i-1].id != "" {
				return "(/p/" + w.b.pages[i-1].id + sub[3]
			}
			return "(#" + sub[3]
		}
		if i < 1 || i > len(w.b.assets) {
			return "(#" + sub[3]
		}
		a := w.b.assets[i-1]
		if a.file == "" && !a.bad {
			f, err := w.s.files.StoreBytes(ctx, w.p, w.ws, n.id, a.Name, bytes.NewReader(a.Data))
			if err != nil {
				a.bad = true
				w.b.Warn("Skipped the file " + a.Name + ": " + err.Error())
			} else {
				a.file = f.ID
				w.job.files()
			}
		}
		if a.bad {
			return "(#" + sub[3]
		}
		return "(/files/" + a.file + sub[3]
	})
	if err := w.s.docs.Write(ctx, w.p, n.id, doc.WriteInput{Markdown: md, Mode: "replace"}); err != nil {
		w.b.Warn("Could not write the body of " + cleanTitle(n.Title) + ": " + err.Error())
	}
}

func (s *Service) materialise(ctx context.Context, p *policy.Principal, job *Job, b *Bundle) error {
	w := &writer{s: s, p: p, ws: job.WorkspaceID, b: b, job: job}
	job.setTotal(b.count())
	job.setState("writing")
	for _, r := range b.Roots {
		if err := w.create(ctx, r, job.ParentID); err != nil {
			return err
		}
	}
	for _, n := range w.written {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.body(ctx, n)
	}
	return nil
}
