package doc

import (
	"context"

	"github.com/reearth/ygo/crdt"

	"github.com/nodemode-ai/form.md/internal/policy"
)

func SyncKey(s string) bool {
	if len(s) < 6 || len(s) > 26 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z') && !(r >= '2' && r <= '7') {
			return false
		}
	}
	return true
}

func SyncRoot(key string) string {
	return "s:" + key
}

type syncedRef struct {
	page string
	key  string
}

func syncedRefs(nodes []any, out map[syncedRef]bool) {
	for _, n := range nodes {
		el, ok := n.(*crdt.YXmlElement)
		if !ok {
			continue
		}
		if el.NodeName == "synced" {
			if k := attrString(el, "key"); SyncKey(k) {
				out[syncedRef{attrString(el, "page"), k}] = true
			}
			continue
		}
		if !textblocks[el.NodeName] && !atoms[el.NodeName] {
			syncedRefs(children(el), out)
		}
	}
}

const syncedDenied = "\x00denied"

func (s *Service) renderSynced(ctx context.Context, p *policy.Principal, self string, refs map[syncedRef]bool) map[syncedRef]string {
	out := map[syncedRef]string{}
	for ref := range refs {
		if ref.page == self {
			continue
		}
		if err := s.pages.CanRead(ctx, p, ref.page); err != nil {
			out[ref] = syncedDenied
			continue
		}
		var titles []string
		_ = s.withRoom(ctx, ref.page, func(r *room) {
			titles = Summarise(r.doc.GetXmlFragment(SyncRoot(ref.key)), nil).Pages
		})
		names := s.pages.Titles(ctx, p, titles)
		_ = s.withRoom(ctx, ref.page, func(r *room) {
			out[ref] = renderFragment(r.doc.GetXmlFragment(SyncRoot(ref.key)), func(id string) string { return names[id] })
		})
	}
	return out
}

func renderFragment(frag *crdt.YXmlFragment, title func(id string) string) string {
	w := &htmlWriter{title: title, nested: true}
	for _, c := range children(frag) {
		if el, ok := c.(*crdt.YXmlElement); ok {
			w.node(el)
		}
	}
	return w.b.String()
}

func (w *htmlWriter) synced(el *crdt.YXmlElement, id string) {
	pg, key := attrString(el, "page"), attrString(el, "key")
	w.open("div", "class", "synced", "data-block", id, "data-page", pg, "data-key", key, "contenteditable", "false")
	w.b.WriteString(`<div class="synced__body">`)
	switch {
	case w.nested || !SyncKey(key):
	case w.local != nil && pg == w.self:
		w.b.WriteString(w.child(w.local(SyncRoot(key))))
	default:
		if html, ok := w.foreign[syncedRef{pg, key}]; ok && html != syncedDenied {
			w.b.WriteString(html)
		} else if ok {
			w.b.WriteString(`<p class="synced__notice">This synced block comes from a page you cannot open.</p>`)
		}
	}
	w.b.WriteString("</div></div>")
}
