package templates

var productStarters = []Starter{
	{
		ID: "product-spec", Name: "Product spec", Description: "A feature spec: problem, users, requirements, design notes, success metrics and launch steps.", Icon: "i:clipboard:blue", Kind: "doc", Category: catProduct, Tags: []string{"prd", "requirements", "feature"},
		markdown: `| Owner | Designer | Engineering lead | Status |
| --- | --- | --- | --- |
|  |  |  | Discovery |

<!-- toc -->

## Problem

Who has the problem, how often, and what it costs them today.

> [!NOTE]
> Link the research, tickets or data that show this is real.

## Who it is for

- **Primary:**
- **Not for:**

## Requirements

| Priority | Requirement | Notes |
| --- | --- | --- |
| Must | People can ... |  |
| Must | ... |  |
| Should | ... |  |
| Could | ... |  |

## User flow

1.
2.
3.

## Design

Link the latest designs and note any decisions made along the way.

## Success metrics

| Metric | Today | Target |
| --- | --- | --- |
|  |  |  |

## Open questions

- [ ]

<details>
<summary>Out of scope for now</summary>

-

</details>

## Launch

- [ ] Beta with five customers
- [ ] Help article and release note
- [ ] General availability
`,
	},
	{
		ID: "research-notes", Name: "User research notes", Description: "Notes from one research session: goals, participant, observations, quotes and insights.", Icon: "i:eye:violet", Kind: "doc", Category: catProduct, Tags: []string{"ux research", "interview", "insights"},
		markdown: `**Date:** {{date}} · **Researcher:**  · **Method:** interview, 45 minutes

## Research goals

- What are we trying to learn?
- What decision will this inform?

## Participant

| Role | Company size | Uses the product for |
| --- | --- | --- |
|  |  |  |

## Observations

-
-

## Quotes

> "Paste the participant's exact words here."

> "One quote per block makes them easy to find later."

## Pain points

| Pain point | How severe | How often |
| --- | --- | --- |
|  | High |  |
|  | Medium |  |

## Insights

> [!TIP]
> An insight explains why something happens, not just what happened.

1.
2.

## Follow-ups

- [ ] Share the highlights with the team
- [ ] Add tagged notes to the research database
`,
	},
	{
		ID: "design-system-tracker", Name: "Design system tracker", Description: "Components with design and code status, owners and documentation links.", Icon: "i:palette:pink", Kind: "database", Category: catProduct, Tags: []string{"components", "ui kit", "design ops"},
		schema: []prop{
			{ID: "title", Name: "Component", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("proposed", "Proposed", "gray", "todo"), so("design", "In design", "purple", "doing"), so("build", "In build", "blue", "doing"), so("stable", "Stable", "green", "done"), so("deprecated", "Deprecated", "red", "done")}},
			{ID: "category", Name: "Category", Type: "select", Options: []opt{o("inputs", "Inputs", "blue"), o("nav", "Navigation", "purple"), o("feedback", "Feedback", "orange"), o("layout", "Layout", "brown")}},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "a11y", Name: "Accessibility checked", Type: "checkbox"},
			{ID: "docs", Name: "Docs", Type: "url"},
		},
		views: []view{
			{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "Components", Type: "gallery", Config: `{"layout":{"cover":"content","size":"small"}}`},
			{Name: "By category", Type: "table", Config: `{"group":{"prop":"category"}}`},
		},
		rows: []row{
			{Title: "Button", Values: map[string]any{"status": "stable", "category": "inputs", "owner": "@me", "a11y": true, "docs": "https://example.com/design/button"}, Markdown: "## Variants\n\n| Variant | Use for |\n| --- | --- |\n| Primary | The one main action on a screen |\n| Ghost | Secondary actions |\n| Danger | Actions that delete or cannot be undone |\n"},
			{Title: "Date picker", Values: map[string]any{"status": "build", "category": "inputs", "a11y": false}},
			{Title: "Toast", Values: map[string]any{"status": "design", "category": "feedback", "a11y": false}},
			{Title: "Tabs", Values: map[string]any{"status": "stable", "category": "nav", "a11y": true}},
			{Title: "Old modal", Values: map[string]any{"status": "deprecated", "category": "layout", "a11y": false}},
		},
	},
	{
		ID: "feature-requests", Name: "Feature request log", Description: "Requests from customers and teammates with votes, segment and a decision status.", Icon: "i:inbox:green", Kind: "database", Category: catProduct, Tags: []string{"feedback", "ideas", "backlog"},
		schema: []prop{
			{ID: "title", Name: "Request", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("new", "New", "gray", "todo"), so("considering", "Considering", "yellow", "doing"), so("planned", "Planned", "blue", "doing"), so("shipped", "Shipped", "green", "done"), so("declined", "Declined", "red", "done")}},
			{ID: "votes", Name: "Votes", Type: "number"},
			{ID: "segment", Name: "Segment", Type: "multi_select", Options: []opt{o("smb", "Small teams", "green"), o("mid", "Mid-market", "blue"), o("ent", "Enterprise", "purple")}},
			{ID: "area", Name: "Area", Type: "select", Options: []opt{o("editor", "Editor", "blue"), o("data", "Databases", "purple"), o("admin", "Admin", "brown"), o("integrations", "Integrations", "orange")}},
			{ID: "requested", Name: "First asked", Type: "date"},
		},
		views: []view{
			{Name: "Most wanted", Type: "table", Config: `{"sorts":[{"prop":"votes","dir":"desc"}]}`},
			{Name: "Status", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "Votes by area", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"area"},"fn":"sum","prop":"votes","labels":true}}}`},
		},
		rows: []row{
			{Title: "Export a database to a spreadsheet", Values: map[string]any{"status": "planned", "votes": 42, "segment": []string{"smb", "mid"}, "area": "data", "requested": "-60d"}},
			{Title: "Single sign-on with our identity provider", Values: map[string]any{"status": "considering", "votes": 31, "segment": []string{"ent"}, "area": "admin", "requested": "-45d"}},
			{Title: "Dark mode for published pages", Values: map[string]any{"status": "new", "votes": 9, "segment": []string{"smb"}, "area": "editor", "requested": "-8d"}},
			{Title: "Calendar sync", Values: map[string]any{"status": "shipped", "votes": 27, "segment": []string{"smb", "mid"}, "area": "integrations", "requested": "-120d"}},
		},
	},
	{
		ID: "interview-log", Name: "Research repository", Description: "Every research session in one place, tagged by theme and persona, ready to search.", Icon: "i:archive:violet", Kind: "database", Category: catProduct, Tags: []string{"ux research", "interviews", "personas"},
		schema: []prop{
			{ID: "title", Name: "Session", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "persona", Name: "Persona", Type: "select", Options: []opt{o("admin", "Workspace admin", "purple"), o("member", "Team member", "blue"), o("guest", "Guest", "gray")}},
			{ID: "themes", Name: "Themes", Type: "multi_select", Options: []opt{o("onboarding", "Onboarding", "green"), o("search", "Search", "blue"), o("sharing", "Sharing", "orange"), o("pricing", "Pricing", "red")}},
			{ID: "researcher", Name: "Researcher", Type: "person"},
			{ID: "method", Name: "Method", Type: "select", Options: []opt{o("interview", "Interview", "purple"), o("usability", "Usability test", "pink"), o("survey", "Survey", "brown")}},
		},
		views: []view{
			{Name: "Sessions", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}]}`},
			{Name: "By persona", Type: "board", Config: `{"group":{"prop":"persona"}}`},
			{Name: "Themes", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"themes"},"fn":"count"}}}`},
		},
		rows: []row{
			{Title: "Admin at a 40-person agency", Values: map[string]any{"date": "-3d", "persona": "admin", "themes": []string{"onboarding", "pricing"}, "researcher": "@me", "method": "interview"}, Markdown: "## Key points\n\n- Sets up every new client space by hand\n- Wants to copy a whole space as a template\n\n> \"I spend my Monday mornings duplicating pages.\"\n"},
			{Title: "New member, first week", Values: map[string]any{"date": "-10d", "persona": "member", "themes": []string{"onboarding", "search"}, "method": "usability"}},
			{Title: "Guest reviewer from a partner", Values: map[string]any{"date": "-17d", "persona": "guest", "themes": []string{"sharing"}, "method": "interview"}},
		},
	},
	{
		ID: "competitive-analysis", Name: "Competitive analysis", Description: "Compare alternatives side by side on features, pricing, strengths and where you win.", Icon: "i:chart-bar:orange", Kind: "doc", Category: catProduct, Tags: []string{"market", "competitors", "positioning"},
		markdown: `**Updated:** {{date}} · **Owner:**

## Summary

> [!IMPORTANT]
> Two or three sentences: where we win, where we lose, and what to do about it.

## The landscape

| | Us | Alternative A | Alternative B | Doing nothing |
| --- | --- | --- | --- | --- |
| Best for |  |  |  | Very small teams |
| Starting price |  |  |  | Free |
| Self-hosted | Yes |  |  | n/a |
| Real-time editing | Yes |  |  | No |
| Databases | Yes |  |  | Spreadsheets |

## Strengths and weaknesses

### Alternative A

- **Strengths:**
- **Weaknesses:**
- **Why customers pick it:**

### Alternative B

- **Strengths:**
- **Weaknesses:**
- **Why customers pick it:**

## Win and loss notes

| Deal | Outcome | Why |
| --- | --- | --- |
|  | Won |  |
|  | Lost |  |

## What we will do

- [ ]
- [ ]
`,
	},
}
