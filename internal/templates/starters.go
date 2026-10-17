package templates

import "encoding/json"

type opt struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Group string `json:"group,omitempty"`
}

type prop struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Options []opt  `json:"options,omitempty"`
	Format  string `json:"format,omitempty"`
}

type view struct {
	Name   string
	Type   string
	Config string
}

type row struct {
	Title    string
	Values   map[string]any
	Markdown string
}

type Starter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Kind        string `json:"kind"`
	Category    string `json:"category"`
	markdown    string
	schema      []prop
	views       []view
	rows        []row
	templates   []row
}

func (s Starter) schemaJSON() json.RawMessage {
	b, _ := json.Marshal(s.schema)
	return b
}

var statusOpts = []opt{{ID: "todo", Name: "Not started", Color: "gray", Group: "todo"}, {ID: "doing", Name: "In progress", Color: "blue", Group: "doing"}, {ID: "done", Name: "Done", Color: "green", Group: "done"}}

var starters = []Starter{
	{
		ID: "meeting-notes", Name: "Meeting notes", Description: "Agenda, notes, decisions and action items for one meeting.", Icon: "i:chat:violet", Kind: "doc", Category: "Docs",
		markdown: "**Date:** {{date}}\n\n**Attendees:** \n\n## Agenda\n\n1. \n2. \n3. \n\n## Notes\n\n- \n\n## Decisions\n\n> [!TIP]\n> Write down what was agreed and who agreed it.\n\n## Action items\n\n- [ ] \n- [ ] \n",
	},
	{
		ID: "project-brief", Name: "Project brief", Description: "Problem, goals, scope, milestones and risks on one page.", Icon: "i:rocket:blue", Kind: "doc", Category: "Docs",
		markdown: "## Problem\n\nWhat is wrong today, and for whom?\n\n## Goals\n\n- \n\n## Out of scope\n\n- \n\n## Milestones\n\n| Milestone | Owner | Date |\n| --- | --- | --- |\n| Kickoff |  | {{date}} |\n| First release |  |  |\n\n## Risks\n\n| Risk | Likelihood | Plan |\n| --- | --- | --- |\n|  |  |  |\n\n## Open questions\n\n- [ ] \n",
	},
	{
		ID: "weekly-review", Name: "Weekly review", Description: "A short reflection: wins, misses, lessons and next week's focus.", Icon: "i:calendar:teal", Kind: "doc", Category: "Personal",
		markdown: "Week of {{date}}\n\n## Wins\n\n- \n\n## What did not go to plan\n\n- \n\n## Lessons\n\n- \n\n## Focus for next week\n\n- [ ] \n- [ ] \n- [ ] \n",
	},
	{
		ID: "team-wiki", Name: "Team wiki", Description: "A home page for a team: who we are, how we work, where things live.", Icon: "i:book:amber", Kind: "doc", Category: "Docs",
		markdown: "<!-- toc -->\n\n## Who we are\n\nOne paragraph on what this team owns.\n\n## How we work\n\n- Planning happens on Mondays.\n- Decisions are written down in the project brief.\n\n## Where things live\n\n| What | Where |\n| --- | --- |\n| Roadmap |  |\n| Runbooks |  |\n\n## Getting help\n\n> [!NOTE]\n> Mention the team in a comment and someone will pick it up.\n",
	},
	{
		ID: "task-tracker", Name: "Task tracker", Description: "Tasks with status, owner, priority and due date, on a board, a table, a calendar and a chart.", Icon: "i:checklist:green", Kind: "database", Category: "Projects",
		schema: []prop{
			{ID: "title", Name: "Task", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: statusOpts},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "priority", Name: "Priority", Type: "select", Options: []opt{{ID: "high", Name: "High", Color: "red"}, {ID: "medium", Name: "Medium", Color: "yellow"}, {ID: "low", Name: "Low", Color: "gray"}}},
			{ID: "due", Name: "Due", Type: "date"},
		},
		views: []view{
			{Name: "Board", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "All tasks", Type: "table", Config: `{"sorts":[{"prop":"due","dir":"asc"}]}`},
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"due"}}`},
			{Name: "By status", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"status"},"series":{"prop":"priority"},"fn":"count","stacked":true}}}`},
		},
		rows: []row{
			{Title: "Write the project brief", Values: map[string]any{"status": "done", "owner": "@me", "priority": "high", "due": "-2d"}},
			{Title: "Collect feedback from the pilot", Values: map[string]any{"status": "doing", "owner": "@me", "priority": "medium", "due": "+3d"}},
			{Title: "Plan the launch", Values: map[string]any{"status": "todo", "priority": "high", "due": "+7d"}},
			{Title: "Tidy the backlog", Values: map[string]any{"status": "todo", "priority": "low", "due": "+10d"}},
		},
		templates: []row{
			{Title: "Bug report", Values: map[string]any{"priority": "high", "status": "todo"}, Markdown: "## What happened\n\n## What should have happened\n\n## Steps to reproduce\n\n1. \n2. \n\n## Notes\n"},
		},
	},
	{
		ID: "reading-list", Name: "Reading list", Description: "Books, articles and podcasts with status, rating and links, shown as a gallery.", Icon: "i:bookmark:rose", Kind: "database", Category: "Personal",
		schema: []prop{
			{ID: "title", Name: "Title", Type: "title"},
			{ID: "author", Name: "Author", Type: "text"},
			{ID: "kind", Name: "Type", Type: "select", Options: []opt{{ID: "book", Name: "Book", Color: "brown"}, {ID: "article", Name: "Article", Color: "blue"}, {ID: "podcast", Name: "Podcast", Color: "purple"}}},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{{ID: "todo", Name: "To read", Color: "gray", Group: "todo"}, {ID: "doing", Name: "Reading", Color: "blue", Group: "doing"}, {ID: "done", Name: "Finished", Color: "green", Group: "done"}}},
			{ID: "rating", Name: "Rating", Type: "number"},
			{ID: "link", Name: "Link", Type: "url"},
		},
		views: []view{
			{Name: "Gallery", Type: "gallery", Config: `{"layout":{"cover":"content","size":"medium"}}`},
			{Name: "By type", Type: "table", Config: `{"group":{"prop":"kind"}}`},
		},
		rows: []row{
			{Title: "The Design of Everyday Things", Values: map[string]any{"author": "Don Norman", "kind": "book", "status": "done", "rating": 5}},
			{Title: "How to write a good bug report", Values: map[string]any{"kind": "article", "status": "doing"}},
			{Title: "An episode worth hearing", Values: map[string]any{"kind": "podcast", "status": "todo"}},
		},
	},
	{
		ID: "content-calendar", Name: "Content calendar", Description: "Posts by publish date and channel, on a calendar and a board.", Icon: "i:megaphone:orange", Kind: "database", Category: "Marketing",
		schema: []prop{
			{ID: "title", Name: "Post", Type: "title"},
			{ID: "publish", Name: "Publish date", Type: "date"},
			{ID: "channel", Name: "Channel", Type: "multi_select", Options: []opt{{ID: "blog", Name: "Blog", Color: "blue"}, {ID: "newsletter", Name: "Newsletter", Color: "green"}, {ID: "social", Name: "Social", Color: "pink"}}},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{{ID: "idea", Name: "Idea", Color: "gray", Group: "todo"}, {ID: "draft", Name: "Drafting", Color: "yellow", Group: "doing"}, {ID: "live", Name: "Published", Color: "green", Group: "done"}}},
			{ID: "owner", Name: "Writer", Type: "person"},
		},
		views: []view{
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"publish"}}`},
			{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "Posts", Type: "feed", Config: `{}`},
		},
		rows: []row{
			{Title: "Launch announcement", Values: map[string]any{"publish": "+5d", "channel": []string{"blog", "social"}, "status": "draft", "owner": "@me"}, Markdown: "A short draft of the launch post.\n\n- What is new\n- Who it is for\n- How to try it\n"},
			{Title: "Monthly newsletter", Values: map[string]any{"publish": "+12d", "channel": []string{"newsletter"}, "status": "idea"}},
		},
	},
	{
		ID: "contacts", Name: "Customer contacts", Description: "People and companies with a stage, last contact and a pipeline chart.", Icon: "i:users:cyan", Kind: "database", Category: "Sales",
		schema: []prop{
			{ID: "title", Name: "Name", Type: "title"},
			{ID: "company", Name: "Company", Type: "text"},
			{ID: "email", Name: "Email", Type: "email"},
			{ID: "stage", Name: "Stage", Type: "select", Options: []opt{{ID: "lead", Name: "Lead", Color: "gray"}, {ID: "talking", Name: "Talking", Color: "blue"}, {ID: "won", Name: "Won", Color: "green"}, {ID: "lost", Name: "Lost", Color: "red"}}},
			{ID: "value", Name: "Value", Type: "number", Format: "dollar"},
			{ID: "last", Name: "Last contact", Type: "date"},
		},
		views: []view{
			{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"stage"}}`},
			{Name: "All contacts", Type: "table", Config: `{}`},
			{Name: "Overview", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"w1","title":"Pipeline value","chart":{"kind":"number","fn":"sum","prop":"value"}},{"id":"w2","title":"Contacts by stage","chart":{"kind":"donut","x":{"prop":"stage"},"fn":"count"}},{"id":"w3","wide":true,"title":"Value by stage","chart":{"kind":"bar","x":{"prop":"stage"},"fn":"sum","prop":"value","labels":true}}]}}`},
		},
		rows: []row{
			{Title: "Ada Byron", Values: map[string]any{"company": "Analytical Ltd", "email": "ada@example.com", "stage": "talking", "value": 12000, "last": "-3d"}},
			{Title: "Grace Hopper", Values: map[string]any{"company": "Compiler Co", "email": "grace@example.com", "stage": "won", "value": 30000, "last": "-10d"}},
			{Title: "Alan Turing", Values: map[string]any{"company": "Enigma Partners", "email": "alan@example.com", "stage": "lead", "value": 8000, "last": "-1d"}},
		},
	},
	{
		ID: "places", Name: "Places to visit", Description: "Cities and spots with a place, shown on a map and a gallery.", Icon: "i:map:teal", Kind: "database", Category: "Personal",
		schema: []prop{
			{ID: "title", Name: "Name", Type: "title"},
			{ID: "where", Name: "Place", Type: "place"},
			{ID: "when", Name: "When", Type: "date"},
			{ID: "been", Name: "Visited", Type: "checkbox"},
		},
		views: []view{
			{Name: "Map", Type: "map", Config: `{"layout":{"place":"where"}}`},
			{Name: "List", Type: "list", Config: `{}`},
		},
		rows: []row{
			{Title: "Lisbon", Values: map[string]any{"where": map[string]any{"lat": 38.7223, "lng": -9.1393, "name": "Lisbon"}, "been": true}},
			{Title: "Kyoto", Values: map[string]any{"where": map[string]any{"lat": 35.0116, "lng": 135.7681, "name": "Kyoto"}, "when": "+60d"}},
			{Title: "Cape Town", Values: map[string]any{"where": map[string]any{"lat": -33.9249, "lng": 18.4241, "name": "Cape Town"}}},
		},
	},
}
