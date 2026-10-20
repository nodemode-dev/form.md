package templates

import (
	"regexp"
	"strconv"
	"strings"
)

type PreviewNode struct {
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Kind  string `json:"kind"`
	Depth int    `json:"depth"`
	Rows  int    `json:"rows,omitempty"`
	View  string `json:"view,omitempty"`
}

func treeOf(root *Starter) (nodes []PreviewNode, pages, dbs, rows int) {
	var walk func(s *Starter, depth int)
	walk = func(s *Starter, depth int) {
		if s.Kind == "database" {
			dbs++
			rows += len(s.rows)
		} else {
			pages++
		}
		if depth > 0 {
			n := PreviewNode{Name: s.Name, Icon: s.Icon, Kind: s.Kind, Depth: depth - 1}
			if s.Kind == "database" {
				n.Rows = len(s.rows)
				n.View = s.views[0].Type
			}
			nodes = append(nodes, n)
		}
		for i := range s.children {
			walk(&s.children[i], depth+1)
		}
	}
	walk(root, 0)
	return nodes, pages, dbs, rows
}

var (
	dbToken   = regexp.MustCompile(`\{\{db:([A-Za-z0-9_-]+)\}\}`)
	pageToken = regexp.MustCompile(`\{\{page:([A-Za-z0-9_-]+)\}\}`)
	dbMarker  = regexp.MustCompile(`(?:<p>)?@@pvdb:([A-Za-z0-9_-]+)@@(?:</p>)?`)
	mathBlock = regexp.MustCompile(`\$\$([^$<]+)\$\$`)
)

func keyed(root *Starter) map[string]*Starter {
	out := map[string]*Starter{}
	var walk func(s *Starter)
	walk = func(s *Starter) {
		if s.key != "" {
			out[s.key] = s
		}
		for i := range s.children {
			walk(&s.children[i])
		}
	}
	walk(root)
	return out
}

func (t *pvTree) docHTML(root *Starter, md string) (string, []string) {
	names := keyed(root)
	md = fill(md)
	md = pageToken.ReplaceAllStringFunc(md, func(m string) string {
		s := names[pageToken.FindStringSubmatch(m)[1]]
		if s == nil {
			return ""
		}
		return "[" + strings.NewReplacer("[", "", "]", "").Replace(s.Name) + "](#)"
	})
	md = dbToken.ReplaceAllStringFunc(md, func(m string) string {
		return "\n\n@@pvdb:" + dbToken.FindStringSubmatch(m)[1] + "@@\n\n"
	})
	out, sections := markdownHTML(md)
	out = diagramHTML(out)
	out = mathBlock.ReplaceAllString(out, `<span class="tplp-math">$1</span>`)
	n := 0
	out = dbMarker.ReplaceAllStringFunc(out, func(m string) string {
		s := names[dbMarker.FindStringSubmatch(m)[1]]
		if s == nil || s.Kind != "database" {
			return ""
		}
		n++
		return t.inlineHTML(s, n)
	})
	return out, sections
}

func (t *pvTree) inlineHTML(s *Starter, n int) string {
	head := `<div class="tplp-inline"><div class="tplp-inline__head">` + esc(s.Name) + `<span>` + strconv.Itoa(len(s.rows)) + `</span></div>`
	if n > 8 {
		return head + `</div>`
	}
	return head + databaseHTML(t, s, t.now) + `</div>`
}
