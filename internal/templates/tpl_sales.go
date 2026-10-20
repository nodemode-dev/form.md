package templates

var salesStarters = []Starter{
	{
		ID: "deal-pipeline", Name: "Deal pipeline", Description: "Deals by stage with value, close date and owner, on a board, a forecast and a dashboard.", Icon: "i:wallet:green", Kind: "database", Category: catSales, Tags: []string{"crm", "deals", "forecast"},
		schema: []prop{
			{ID: "title", Name: "Deal", Type: "title"},
			{ID: "stage", Name: "Stage", Type: "status", Options: []opt{so("qualified", "Qualified", "gray", "todo"), so("demo", "Demo", "blue", "doing"), so("proposal", "Proposal", "purple", "doing"), so("negotiation", "Negotiation", "orange", "doing"), so("won", "Won", "green", "done"), so("lost", "Lost", "red", "done")}},
			{ID: "value", Name: "Value", Type: "number", Format: "dollar"},
			{ID: "close", Name: "Expected close", Type: "date"},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "company", Name: "Company", Type: "text"},
			{ID: "source", Name: "Source", Type: "select", Options: []opt{o("inbound", "Inbound", "green"), o("outbound", "Outbound", "blue"), o("partner", "Partner", "purple")}},
		},
		views: []view{
			{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"stage"},"calcs":{"value":"sum"}}`},
			{Name: "Forecast", Type: "table", Config: `{"sorts":[{"prop":"close","dir":"asc"}],"calcs":{"value":"sum"}}`},
			{Name: "Overview", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"w1","title":"Open pipeline","chart":{"kind":"number","fn":"sum","prop":"value"}},{"id":"w2","title":"Deals by source","chart":{"kind":"donut","x":{"prop":"source"},"fn":"count"}},{"id":"w3","wide":true,"title":"Value by stage","chart":{"kind":"bar","x":{"prop":"stage"},"fn":"sum","prop":"value","labels":true}}]}}`},
		},
		rows: []row{
			{Title: "Northwind · 50 seats", Values: map[string]any{"stage": "proposal", "value": 18000, "close": "+14d", "owner": "@me", "company": "Northwind Traders", "source": "inbound"}, Markdown: "## Next step\n\nSend the security questionnaire back by Friday.\n\n## Stakeholders\n\n| Name | Role | View |\n| --- | --- | --- |\n| Dana | Champion | Keen |\n| Lee | Budget holder | Unsure about price |\n"},
			{Title: "Bluebell Labs · pilot", Values: map[string]any{"stage": "demo", "value": 6000, "close": "+30d", "company": "Bluebell Labs", "source": "outbound"}},
			{Title: "Contoso · renewal and upgrade", Values: map[string]any{"stage": "negotiation", "value": 42000, "close": "+7d", "owner": "@me", "company": "Contoso", "source": "partner"}},
			{Title: "Fabrikam · team plan", Values: map[string]any{"stage": "won", "value": 9500, "close": "-5d", "company": "Fabrikam", "source": "inbound"}},
			{Title: "Tailspin · 10 seats", Values: map[string]any{"stage": "lost", "value": 3000, "close": "-12d", "company": "Tailspin Toys", "source": "outbound"}},
		},
	},
	{
		ID: "sales-call-notes", Name: "Sales call notes", Description: "Discovery call notes: the customer's situation, needs, decision process and next step.", Icon: "i:phone:blue", Kind: "doc", Category: catSales, Tags: []string{"discovery", "call", "qualification"},
		markdown: `**Company:**  · **Date:** {{date}} · **On the call:**

## Their situation

- What they use today:
- Team size and who would use it:
- What triggered the conversation now:

## Needs and pains

| Need | Why it matters to them | Priority |
| --- | --- | --- |
|  |  | High |
|  |  | Medium |

## Decision process

- **Decision maker:**
- **Budget:**
- **Timeline:**
- **Other options they are looking at:**

> [!TIP]
> Before you hang up, agree on one concrete next step with a date.

## Objections

<details>
<summary>What they raised and how we answered</summary>

-

</details>

## Next steps

- [ ] Send a recap email today
- [ ]
`,
	},
	{
		ID: "account-plan", Name: "Account plan", Description: "A plan for a key customer: goals, stakeholders, health, risks and growth opportunities.", Icon: "i:building:cyan", Kind: "doc", Category: catSales, Tags: []string{"key account", "customer success", "expansion"},
		markdown: `| Account | Owner | Renewal date | Annual value |
| --- | --- | --- | --- |
|  |  |  |  |

**Plan updated:** {{date}}

## Their goals this year

1.
2.
3.

## Stakeholders

| Name | Role | Relationship | Last contact |
| --- | --- | --- | --- |
|  | Executive sponsor | Strong |  |
|  | Day-to-day admin | Good |  |
|  | Finance | New |  |

## Health

| Signal | Status | Notes |
| --- | --- | --- |
| Usage trend | Growing |  |
| Support tickets | Normal |  |
| Sentiment | Positive |  |

## Risks

> [!WARNING]
> List anything that could put the renewal at risk, with an owner for each.

-

## Growth opportunities

| Opportunity | Value | Next step |
| --- | --- | --- |
| New team rollout |  |  |
| Upgrade to a higher plan |  |  |

## Actions this quarter

- [ ] Quarterly business review booked
- [ ] Training session for new users
- [ ]
`,
	},
	{
		ID: "customer-feedback", Name: "Customer feedback", Description: "Feedback from calls, tickets and surveys, tagged by theme and sentiment, with trends.", Icon: "i:heart:rose", Kind: "database", Category: catSales, Tags: []string{"voice of customer", "nps", "insights"},
		schema: []prop{
			{ID: "title", Name: "Feedback", Type: "title"},
			{ID: "customer", Name: "Customer", Type: "text"},
			{ID: "source", Name: "Source", Type: "select", Options: []opt{o("call", "Call", "blue"), o("ticket", "Support ticket", "orange"), o("survey", "Survey", "purple"), o("social", "Social", "pink")}},
			{ID: "sentiment", Name: "Sentiment", Type: "select", Options: []opt{o("positive", "Positive", "green"), o("neutral", "Neutral", "gray"), o("negative", "Negative", "red")}},
			{ID: "theme", Name: "Theme", Type: "multi_select", Options: []opt{o("speed", "Speed", "blue"), o("pricing", "Pricing", "orange"), o("ux", "Ease of use", "green"), o("integrations", "Integrations", "purple")}},
			{ID: "date", Name: "Received", Type: "date"},
		},
		views: []view{
			{Name: "Inbox", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}]}`},
			{Name: "By sentiment", Type: "board", Config: `{"group":{"prop":"sentiment"}}`},
			{Name: "Themes", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"theme"},"series":{"prop":"sentiment"},"fn":"count","stacked":true}}}`},
		},
		rows: []row{
			{Title: "Love how fast pages open now", Values: map[string]any{"customer": "Northwind Traders", "source": "survey", "sentiment": "positive", "theme": []string{"speed"}, "date": "-1d"}},
			{Title: "Hard to find where templates live", Values: map[string]any{"customer": "Contoso", "source": "call", "sentiment": "negative", "theme": []string{"ux"}, "date": "-3d"}},
			{Title: "Would pay more for calendar sync", Values: map[string]any{"customer": "Fabrikam", "source": "ticket", "sentiment": "neutral", "theme": []string{"pricing", "integrations"}, "date": "-6d"}},
			{Title: "Setup took our team ten minutes", Values: map[string]any{"customer": "Bluebell Labs", "source": "social", "sentiment": "positive", "theme": []string{"ux"}, "date": "-8d"}},
		},
	},
	{
		ID: "support-tickets", Name: "Support tickets", Description: "Customer issues with priority, status and assignee, with a queue board and response times.", Icon: "i:mail:blue", Kind: "database", Category: catSales, Tags: []string{"helpdesk", "support", "queue"},
		schema: []prop{
			{ID: "title", Name: "Subject", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("open", "Open", "red", "todo"), so("waiting", "Waiting on customer", "yellow", "doing"), so("progress", "In progress", "blue", "doing"), so("solved", "Solved", "green", "done")}},
			{ID: "priority", Name: "Priority", Type: "select", Options: []opt{o("urgent", "Urgent", "red"), o("high", "High", "orange"), o("normal", "Normal", "blue"), o("low", "Low", "gray")}},
			{ID: "customer", Name: "Customer email", Type: "email"},
			{ID: "assignee", Name: "Assignee", Type: "person"},
			{ID: "opened", Name: "Opened", Type: "date"},
			{ID: "channel", Name: "Channel", Type: "select", Options: []opt{o("email", "Email", "blue"), o("chat", "Chat", "green"), o("phone", "Phone", "purple")}},
		},
		views: []view{
			{Name: "Queue", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "All tickets", Type: "table", Config: `{"sorts":[{"prop":"opened","dir":"desc"}]}`},
			{Name: "By priority", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"priority"},"fn":"count"}}}`},
		},
		rows: []row{
			{Title: "Cannot invite a teammate", Values: map[string]any{"status": "open", "priority": "high", "customer": "dana@northwind.example", "opened": "-1d", "channel": "email"}},
			{Title: "Invoice shows the wrong company name", Values: map[string]any{"status": "waiting", "priority": "normal", "customer": "billing@contoso.example", "assignee": "@me", "opened": "-3d", "channel": "chat"}},
			{Title: "Workspace down for all users", Values: map[string]any{"status": "solved", "priority": "urgent", "customer": "it@fabrikam.example", "assignee": "@me", "opened": "-7d", "channel": "phone"}, Markdown: "## Resolution\n\nA certificate expired on the customer's proxy. Walked their IT team through renewing it.\n\n## Follow-up\n\n- [x] Sent a summary\n- [ ] Add a help article about proxies\n"},
			{Title: "How do I export a page to PDF?", Values: map[string]any{"status": "progress", "priority": "low", "customer": "sam@tailspin.example", "opened": "-2d", "channel": "chat"}},
		},
		templates: []row{
			{Title: "New ticket", Values: map[string]any{"status": "open", "priority": "normal"}, Markdown: "## Problem\n\n## Steps tried\n\n- \n\n## Resolution\n"},
		},
	},
}
