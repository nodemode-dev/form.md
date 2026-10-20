package templates

import (
	"fmt"
	"sort"
	"strings"
)

const (
	catCharts    = "Data & charts"
	catExamples  = "Example projects"
	catProjects  = "Projects"
	catDocs      = "Docs & wikis"
	catMeetings  = "Meetings"
	catEng       = "Engineering"
	catProduct   = "Product & design"
	catMarketing = "Marketing"
	catSales     = "Sales & customers"
	catTeam      = "Team & HR"
	catPersonal  = "Personal & school"
)

type Category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

var categoryOrder = []Category{
	{ID: "charts", Name: catCharts},
	{ID: "examples", Name: catExamples},
	{ID: "projects", Name: catProjects},
	{ID: "docs", Name: catDocs},
	{ID: "meetings", Name: catMeetings},
	{ID: "engineering", Name: catEng},
	{ID: "product", Name: catProduct},
	{ID: "marketing", Name: catMarketing},
	{ID: "sales", Name: catSales},
	{ID: "team", Name: catTeam},
	{ID: "personal", Name: catPersonal},
}

var categories []Category

func o(id, name, color string) opt { return opt{ID: id, Name: name, Color: color} }

func so(id, name, color, group string) opt {
	return opt{ID: id, Name: name, Color: color, Group: group}
}

func checkTree(root *Starter) {
	seen := map[string]bool{}
	var walk func(s *Starter)
	walk = func(s *Starter) {
		if s.Kind == "database" && len(s.views) == 0 {
			panic(fmt.Sprintf("template %s/%s: a database needs at least one view", root.ID, s.Name))
		}
		for _, p := range s.schema {
			if k, ok := strings.CutPrefix(p.Collection, "@"); ok && k != "self" && k != s.key && !seen[k] {
				panic(fmt.Sprintf("template %s/%s: relation %s points at %q before it is created", root.ID, s.Name, p.ID, k))
			}
		}
		if s.key != "" {
			if seen[s.key] {
				panic(fmt.Sprintf("template %s: key %q used twice", root.ID, s.key))
			}
			seen[s.key] = true
		}
		for i := range s.children {
			walk(&s.children[i])
		}
	}
	walk(root)
	var refs func(s *Starter)
	refs = func(s *Starter) {
		texts := []string{s.markdown}
		for _, r := range s.rows {
			texts = append(texts, r.Markdown)
		}
		for _, t := range texts {
			for _, m := range placeholder.FindAllStringSubmatch(t, -1) {
				if !seen[m[2]] {
					panic(fmt.Sprintf("template %s/%s: %s names key %q that the template does not have", root.ID, s.Name, m[0], m[2]))
				}
			}
		}
		for i := range s.children {
			refs(&s.children[i])
		}
	}
	refs(root)
}

func init() {
	for _, set := range [][]Starter{chartStarters, exampleStartersA, exampleStartersB, projectStarters, docStarters, meetingStarters, engStarters, productStarters, marketingStarters, salesStarters, teamStarters, personalStarters} {
		starters = append(starters, set...)
	}
	ids := map[string]bool{}
	for i := range starters {
		s := &starters[i]
		if ids[s.ID] {
			panic("template id used twice: " + s.ID)
		}
		ids[s.ID] = true
		checkTree(s)
		if len(s.children) > 0 {
			s.Pages = s.count()
			s.Databases = s.databases()
			s.Docs = s.Pages - s.Databases
		}
	}
	rank := map[string]int{}
	for i, c := range categoryOrder {
		rank[c.Name] = i
	}
	sort.SliceStable(starters, func(i, j int) bool { return rank[starters[i].Category] < rank[starters[j].Category] })
	count := map[string]int{}
	for _, s := range starters {
		count[s.Category]++
	}
	for _, c := range categoryOrder {
		c.Count = count[c.Name]
		categories = append(categories, c)
	}
}
