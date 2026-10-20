package templates

import "sort"

const (
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

func init() {
	for _, set := range [][]Starter{projectStarters, docStarters, meetingStarters, engStarters, productStarters, marketingStarters, salesStarters, teamStarters, personalStarters} {
		starters = append(starters, set...)
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
