package doc

import (
	"context"
	"strings"

	"github.com/reearth/ygo/crdt"
)

type Public struct {
	Page     func(id string) string
	Title    func(id string) (string, bool)
	PageIcon func(id, class string) string
	File     func(id string) string
	Icon     func(ref, fallback, class string) string
	UI       func(name string) string
	Table    func(collectionID, viewID string) string
	Date     func(iso string) string
}

func (s *Service) RenderPublic(ctx context.Context, id string, pub *Public) (string, error) {
	refs := map[syncedRef]bool{}
	if err := s.withRoom(ctx, id, func(r *room) { syncedRefs(children(r.fragment()), refs) }); err != nil {
		return "", err
	}
	foreign := map[syncedRef]string{}
	for ref := range refs {
		if ref.page == id || pub.Page(ref.page) == "" {
			continue
		}
		_ = s.withRoom(ctx, ref.page, func(r *room) {
			w := &htmlWriter{nested: true, pub: pub}
			w.nodes(r.doc.GetXmlFragment(SyncRoot(ref.key)))
			foreign[ref] = w.b.String()
		})
	}
	var out string
	err := s.withRoom(ctx, id, func(r *room) {
		w := &htmlWriter{self: r.id, local: r.doc.GetXmlFragment, foreign: foreign, pub: pub}
		frag := r.fragment()
		w.collectHeadings(children(frag))
		w.nodes(frag)
		out = w.b.String()
	})
	return out, err
}

func (w *htmlWriter) nodes(frag *crdt.YXmlFragment) {
	for _, c := range children(frag) {
		if el, ok := c.(*crdt.YXmlElement); ok {
			w.node(el)
		}
	}
}

func (w *htmlWriter) child(frag *crdt.YXmlFragment) string {
	if w.pub == nil {
		return renderFragment(frag, w.title)
	}
	c := &htmlWriter{title: w.title, nested: true, pub: w.pub}
	c.nodes(frag)
	return c.b.String()
}

func (w *htmlWriter) src(ref string) string {
	if w.pub != nil {
		if id := fileID(ref); id != "" {
			return w.pub.File(id)
		}
	}
	return fileURL(ref)
}

func (w *htmlWriter) href(h string) string {
	if w.pub == nil || h == "" {
		return h
	}
	if id, ok := strings.CutPrefix(h, "/p/"); ok {
		id, frag, _ := strings.Cut(id, "#")
		if u := w.pub.Page(id); u != "" {
			if frag != "" {
				return u + "#" + frag
			}
			return u
		}
		return ""
	}
	if id := fileID(h); id != "" {
		return w.pub.File(id)
	}
	return h
}

func (w *htmlWriter) pubMention(ref string) {
	label, ok := w.pub.Title(ref)
	if !ok {
		w.open("span", "class", "mention", "data-kind", "page", "data-missing", "true")
		w.b.WriteString(w.pub.Icon("", "lock", "mention__icon") + `<span class="mention__label">Private page</span></span>`)
		return
	}
	if label == "" {
		label = "Untitled"
	}
	w.open("a", "class", "mention", "data-kind", "page", "href", w.pub.Page(ref))
	w.b.WriteString(w.pub.PageIcon(ref, "mention__icon") + `<span class="mention__label">` + esc(label) + "</span></a>")
}

func (w *htmlWriter) dateText(ref string) string {
	if w.pub != nil && w.pub.Date != nil {
		return w.pub.Date(ref)
	}
	return ref
}
