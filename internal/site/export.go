package site

import (
	"context"
	"sort"
	"strings"
)

type ExportNode struct {
	ID          string
	Parent      string
	Title       string
	Icon        string
	Cover       string
	Kind        string
	Description string
	SortKey     string
	Row         bool
	Inline      bool
	UpdatedAt   int64
}

type Export struct {
	s  *Service
	sc *scope
}

const exportScript = `document.addEventListener("click",function(e){var b=e.target.closest&&e.target.closest(".toggle__btn");if(!b)return;e.preventDefault();var t=b.closest(".toggle");if(!t)return;var o=!t.classList.contains("is-open");t.classList.toggle("is-open",o);b.setAttribute("aria-expanded",o?"true":"false")});`

func (s *Service) NewExport(nodes []ExportNode, root, theme string, href, file func(id string) string) *Export {
	sc := &scope{root: root, theme: theme, pages: map[string]*node{}, kids: map[string][]*node{}, hrefFn: href, fileFn: file, export: true}
	for _, n := range nodes {
		sc.pages[n.ID] = &node{ID: n.ID, Parent: n.Parent, Title: n.Title, Icon: n.Icon, Cover: n.Cover, Kind: n.Kind, Description: n.Description, SortKey: n.SortKey, Row: n.Row, Inline: n.Inline, UpdatedAt: n.UpdatedAt}
	}
	for _, n := range sc.pages {
		if n.ID != root {
			sc.kids[n.Parent] = append(sc.kids[n.Parent], n)
		}
	}
	for _, list := range sc.kids {
		sort.Slice(list, func(i, j int) bool { return list[i].SortKey < list[j].SortKey })
	}
	return &Export{s: s, sc: sc}
}

func (e *Export) Page(ctx context.Context, id string) (string, error) {
	if _, ok := e.sc.pages[id]; !ok {
		return "", nil
	}
	out, err := e.s.renderPage(ctx, e.sc, id)
	if err != nil {
		return "", err
	}
	out = strings.Replace(out, " · Published with form.md</footer>", " · Exported from form.md</footer>", 1)
	return strings.Replace(out, "</body></html>", "<script>"+exportScript+"</script></body></html>", 1), nil
}
