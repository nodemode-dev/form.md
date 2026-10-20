package templates

var projectStarters = []Starter{
	{
		ID: "roadmap", Name: "Roadmap", Description: "Initiatives by quarter and theme, on a timeline, a board and a progress chart.", Icon: "i:map:blue", Kind: "database", Category: catProjects, Tags: []string{"planning", "quarters", "timeline"},
		schema: []prop{
			{ID: "title", Name: "Initiative", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("later", "Later", "gray", "todo"), so("next", "Next", "yellow", "todo"), so("now", "Now", "blue", "doing"), so("shipped", "Shipped", "green", "done")}},
			{ID: "theme", Name: "Theme", Type: "select", Options: []opt{o("growth", "Growth", "green"), o("quality", "Quality", "blue"), o("platform", "Platform", "purple"), o("revenue", "Revenue", "orange")}},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "start", Name: "Start", Type: "date"},
			{ID: "end", Name: "End", Type: "date"},
			{ID: "impact", Name: "Impact", Type: "select", Options: []opt{o("high", "High", "red"), o("medium", "Medium", "yellow"), o("low", "Low", "gray")}},
		},
		views: []view{
			{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"start","end":"end","zoom":"month"}}`},
			{Name: "Now, next, later", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "By theme", Type: "table", Config: `{"group":{"prop":"theme"},"sorts":[{"prop":"start","dir":"asc"}]}`},
			{Name: "Progress", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"theme"},"series":{"prop":"status"},"fn":"count","stacked":true}}}`},
		},
		rows: []row{
			{Title: "Self-serve onboarding", Values: map[string]any{"status": "now", "theme": "growth", "owner": "@me", "start": "-14d", "end": "+30d", "impact": "high"}, Markdown: "## Why\n\nNew teams stall in their first week. A guided setup should halve time to first value.\n\n## Measures\n\n- Activation rate in the first 7 days\n- Support tickets tagged onboarding\n"},
			{Title: "Faster search", Values: map[string]any{"status": "next", "theme": "quality", "start": "+20d", "end": "+60d", "impact": "medium"}},
			{Title: "Usage-based billing", Values: map[string]any{"status": "later", "theme": "revenue", "start": "+70d", "end": "+120d", "impact": "high"}},
			{Title: "Audit log export", Values: map[string]any{"status": "shipped", "theme": "platform", "start": "-60d", "end": "-20d", "impact": "low"}},
		},
	},
	{
		ID: "sprint-board", Name: "Sprint board", Description: "Stories with points, sprint and assignee, on a board, a table and a burn chart.", Icon: "i:kanban:violet", Kind: "database", Category: catProjects, Tags: []string{"agile", "scrum", "stories"},
		schema: []prop{
			{ID: "title", Name: "Story", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("backlog", "Backlog", "gray", "todo"), so("ready", "Ready", "brown", "todo"), so("progress", "In progress", "blue", "doing"), so("review", "In review", "purple", "doing"), so("done", "Done", "green", "done")}},
			{ID: "assignee", Name: "Assignee", Type: "person"},
			{ID: "points", Name: "Points", Type: "number"},
			{ID: "sprint", Name: "Sprint", Type: "select", Options: []opt{o("s14", "Sprint 14", "blue"), o("s15", "Sprint 15", "purple")}},
			{ID: "type", Name: "Type", Type: "select", Options: []opt{o("feature", "Feature", "green"), o("bug", "Bug", "red"), o("chore", "Chore", "gray")}},
		},
		views: []view{
			{Name: "Board", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "Backlog", Type: "table", Config: `{"group":{"prop":"sprint"},"calcs":{"points":"sum"}}`},
			{Name: "Points by status", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"status"},"fn":"sum","prop":"points","labels":true}}}`},
		},
		rows: []row{
			{Title: "Sign in with a magic link", Values: map[string]any{"status": "progress", "assignee": "@me", "points": 5, "sprint": "s14", "type": "feature"}, Markdown: "## Acceptance criteria\n\n- [ ] A link is emailed within a minute\n- [ ] The link works once and expires after 15 minutes\n- [ ] Expired links explain what to do next\n"},
			{Title: "Fix double submit on the invite form", Values: map[string]any{"status": "review", "points": 2, "sprint": "s14", "type": "bug"}},
			{Title: "Upgrade the image library", Values: map[string]any{"status": "ready", "points": 3, "sprint": "s14", "type": "chore"}},
			{Title: "Team switcher in the sidebar", Values: map[string]any{"status": "backlog", "points": 8, "sprint": "s15", "type": "feature"}},
			{Title: "Empty states for new workspaces", Values: map[string]any{"status": "done", "points": 3, "sprint": "s14", "type": "feature"}},
		},
		templates: []row{
			{Title: "User story", Values: map[string]any{"status": "backlog", "type": "feature"}, Markdown: "As a **who**, I want **what**, so that **why**.\n\n## Acceptance criteria\n\n- [ ] \n- [ ] \n\n## Notes\n"},
		},
	},
	{
		ID: "okrs", Name: "OKRs", Description: "Objectives and key results with owners, confidence and progress toward each target.", Icon: "i:target:rose", Kind: "database", Category: catProjects, Tags: []string{"goals", "objectives", "quarterly"},
		schema: []prop{
			{ID: "title", Name: "Key result", Type: "title"},
			{ID: "objective", Name: "Objective", Type: "select", Options: []opt{o("o1", "Make onboarding effortless", "blue"), o("o2", "Grow paying teams", "green"), o("o3", "Earn trust with reliability", "purple")}},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "progress", Name: "Progress", Type: "number"},
			{ID: "confidence", Name: "Confidence", Type: "select", Options: []opt{o("on", "On track", "green"), o("risk", "At risk", "yellow"), o("off", "Off track", "red")}},
			{ID: "quarter", Name: "Quarter", Type: "select", Options: []opt{o("q4", "Q4", "gray"), o("q1", "Q1", "brown")}},
		},
		views: []view{
			{Name: "By objective", Type: "table", Config: `{"group":{"prop":"objective"},"calcs":{"progress":"average"}}`},
			{Name: "Confidence", Type: "board", Config: `{"group":{"prop":"confidence"}}`},
			{Name: "Overview", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"w1","title":"Average progress","chart":{"kind":"number","fn":"average","prop":"progress"}},{"id":"w2","title":"Confidence","chart":{"kind":"donut","x":{"prop":"confidence"},"fn":"count"}},{"id":"w3","wide":true,"title":"Progress by objective","chart":{"kind":"bar","x":{"prop":"objective"},"fn":"average","prop":"progress","labels":true}}]}}`},
		},
		rows: []row{
			{Title: "Time to first page under 2 minutes", Values: map[string]any{"objective": "o1", "owner": "@me", "progress": 60, "confidence": "on", "quarter": "q4"}},
			{Title: "80% of new teams invite a teammate", Values: map[string]any{"objective": "o1", "progress": 35, "confidence": "risk", "quarter": "q4"}},
			{Title: "120 new paying teams", Values: map[string]any{"objective": "o2", "progress": 48, "confidence": "on", "quarter": "q4"}},
			{Title: "99.95% monthly uptime", Values: map[string]any{"objective": "o3", "progress": 90, "confidence": "on", "quarter": "q4"}},
			{Title: "No incident open longer than 4 hours", Values: map[string]any{"objective": "o3", "progress": 20, "confidence": "off", "quarter": "q4"}},
		},
	},
	{
		ID: "launch-plan", Name: "Launch plan", Description: "Everything a release needs: goals, audience, timeline, owners and a go or no-go checklist.", Icon: "i:rocket:orange", Kind: "doc", Category: catProjects, Tags: []string{"release", "go to market", "checklist"},
		markdown: `**Launch date:** {{date}} · **Launch owner:**

> [!IMPORTANT]
> One sentence on what is launching and why anyone should care.

## Goals

| Goal | Measure | Target |
| --- | --- | --- |
| Awareness | Visits to the launch page | 10,000 in week one |
| Adoption | Teams that try the feature | 500 in 30 days |
| Quality | Support tickets about it | Fewer than 20 |

## Audience

- **Primary:**
- **Secondary:**

## Timeline

| When | What | Owner |
| --- | --- | --- |
| T minus 3 weeks | Feature complete, docs drafted |  |
| T minus 1 week | Beta feedback in, press kit ready |  |
| Launch day | Announcement, email, social posts |  |
| T plus 1 week | Results review |  |

## Go or no-go checklist

- [ ] All launch-blocking bugs closed
- [ ] Help articles published
- [ ] Support team briefed
- [ ] Pricing and plans page updated
- [ ] Rollback plan written and tested

<details>
<summary>Messaging</summary>

- **Headline:**
- **Three proof points:**
- **Call to action:**

</details>

## After launch

- [ ] Share results with the team
- [ ] Collect the top five pieces of feedback
`,
	},
	{
		ID: "risk-register", Name: "Risk register", Description: "Project risks scored by likelihood and impact, with owners and mitigation plans.", Icon: "i:shield:amber", Kind: "database", Category: catProjects, Tags: []string{"risks", "mitigation", "governance"},
		schema: []prop{
			{ID: "title", Name: "Risk", Type: "title"},
			{ID: "likelihood", Name: "Likelihood", Type: "select", Options: []opt{o("high", "High", "red"), o("medium", "Medium", "yellow"), o("low", "Low", "green")}},
			{ID: "impact", Name: "Impact", Type: "select", Options: []opt{o("high", "High", "red"), o("medium", "Medium", "yellow"), o("low", "Low", "green")}},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("open", "Open", "red", "todo"), so("watching", "Watching", "yellow", "doing"), so("closed", "Closed", "green", "done")}},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "mitigation", Name: "Mitigation", Type: "text"},
			{ID: "review", Name: "Next review", Type: "date"},
		},
		views: []view{
			{Name: "All risks", Type: "table", Config: `{"sorts":[{"prop":"likelihood","dir":"asc"}]}`},
			{Name: "By status", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "Reviews", Type: "calendar", Config: `{"layout":{"date":"review"}}`},
		},
		rows: []row{
			{Title: "Key supplier misses the delivery date", Values: map[string]any{"likelihood": "medium", "impact": "high", "status": "watching", "owner": "@me", "mitigation": "Weekly check-in and a second supplier on standby", "review": "+7d"}},
			{Title: "Two engineers on leave in the same sprint", Values: map[string]any{"likelihood": "high", "impact": "medium", "status": "open", "mitigation": "Move the migration work forward a sprint", "review": "+3d"}},
			{Title: "Scope grows after the design review", Values: map[string]any{"likelihood": "medium", "impact": "medium", "status": "open", "mitigation": "Changes after review go to the next release", "review": "+10d"}},
			{Title: "Data migration loses records", Values: map[string]any{"likelihood": "low", "impact": "high", "status": "closed", "mitigation": "Dry run on a copy and row counts before and after", "review": "-5d"}},
		},
	},
}
