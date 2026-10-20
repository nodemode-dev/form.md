package templates

var docStarters = []Starter{
	{
		ID: "company-handbook", Name: "Company handbook", Description: "Mission, values, policies and the practical details every new person asks about.", Icon: "i:book:violet", Kind: "doc", Category: catDocs, Tags: []string{"policies", "culture", "values"},
		markdown: `<!-- toc -->

## Our mission

One or two sentences on why the company exists and who it serves.

## What we value

1. **Write it down.** Decisions, plans and answers live in pages anyone can find.
2. **Default to trust.** Give people context and room, then ask how it went.
3. **Small and often.** Ship in slices, learn, adjust.
4. **Customers in the room.** Every big call names the customer it helps.

## How we work

| Topic | The short version |
| --- | --- |
| Working hours | Core hours 10:00 to 15:00 in your time zone |
| Meetings | Agenda in the invite, notes in the workspace, none on Fridays |
| Communication | Pages for decisions, chat for quick questions |
| Tools | This workspace, email, the team calendar |

## Time off and leave

- 25 days of paid leave plus public holidays
- Book in the time off tracker at least two weeks ahead for anything over three days
- Sick days do not need approval, just let your manager know

## Expenses

> [!NOTE]
> Keep receipts. Anything over 500 needs a manager's approval before you spend it.

<details>
<summary>What counts as a business expense</summary>

- Travel to meet customers or teammates
- Books and courses tied to your role
- Equipment from the approved list

</details>

<details>
<summary>Equipment and security</summary>

- Turn on disk encryption and a screen lock
- Use the password manager for every work account
- Report a lost device within 24 hours

</details>

## Who to ask

| Question about | Ask |
| --- | --- |
| Pay and benefits | People team |
| Accounts and laptops | IT |
| Anything else | Your manager or buddy |
`,
	},
	{
		ID: "decision-log", Name: "Decision log", Description: "A searchable record of decisions: context, options, the call made and who made it.", Icon: "i:scale:blue", Kind: "database", Category: catDocs, Tags: []string{"decisions", "adr", "record"},
		schema: []prop{
			{ID: "title", Name: "Decision", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("proposed", "Proposed", "yellow", "todo"), so("decided", "Decided", "green", "done"), so("superseded", "Superseded", "gray", "done")}},
			{ID: "area", Name: "Area", Type: "select", Options: []opt{o("product", "Product", "blue"), o("eng", "Engineering", "purple"), o("ops", "Operations", "brown"), o("people", "People", "pink")}},
			{ID: "owner", Name: "Decided by", Type: "person"},
			{ID: "date", Name: "Date", Type: "date"},
		},
		views: []view{
			{Name: "All decisions", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}]}`},
			{Name: "By area", Type: "board", Config: `{"group":{"prop":"area"}}`},
			{Name: "Recent", Type: "feed", Config: `{}`},
		},
		rows: []row{
			{Title: "Move weekly planning to Tuesdays", Values: map[string]any{"status": "decided", "area": "ops", "owner": "@me", "date": "-6d"}, Markdown: "## Context\n\nMonday planning kept losing people to weekend catch-up.\n\n## Options\n\n1. Keep Mondays and shorten the meeting\n2. Move to Tuesdays\n\n## Decision\n\nOption 2, reviewed after six weeks.\n"},
			{Title: "Use one database for every bug", Values: map[string]any{"status": "decided", "area": "eng", "date": "-20d"}},
			{Title: "Offer an annual plan", Values: map[string]any{"status": "proposed", "area": "product", "date": "+4d"}},
			{Title: "Quarterly team offsites", Values: map[string]any{"status": "superseded", "area": "people", "date": "-90d"}},
		},
		templates: []row{
			{Title: "New decision", Values: map[string]any{"status": "proposed"}, Markdown: "## Context\n\n## Options\n\n1. \n2. \n\n## Decision\n\n## Consequences\n"},
		},
	},
	{
		ID: "knowledge-base", Name: "Knowledge base", Description: "Articles grouped by topic with an owner and a review date, so answers stay current.", Icon: "i:lightbulb:amber", Kind: "database", Category: catDocs, Tags: []string{"help center", "articles", "docs"},
		schema: []prop{
			{ID: "title", Name: "Article", Type: "title"},
			{ID: "topic", Name: "Topic", Type: "select", Options: []opt{o("start", "Getting started", "green"), o("accounts", "Accounts", "blue"), o("billing", "Billing", "orange"), o("trouble", "Troubleshooting", "red")}},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "reviewed", Name: "Last reviewed", Type: "date"},
			{ID: "audience", Name: "Audience", Type: "multi_select", Options: []opt{o("customers", "Customers", "blue"), o("staff", "Staff", "purple")}},
		},
		views: []view{
			{Name: "Articles", Type: "gallery", Config: `{"layout":{"cover":"content","size":"medium"}}`},
			{Name: "By topic", Type: "list", Config: `{"group":{"prop":"topic"}}`},
			{Name: "Review queue", Type: "table", Config: `{"sorts":[{"prop":"reviewed","dir":"asc"}]}`},
		},
		rows: []row{
			{Title: "Set up your workspace", Values: map[string]any{"topic": "start", "owner": "@me", "reviewed": "-12d", "audience": []string{"customers"}}, Markdown: "1. Create a workspace and give it a name.\n2. Invite two teammates.\n3. Start from a template.\n\n> [!TIP]\n> Most teams start with a task tracker and a team wiki.\n"},
			{Title: "Reset a password", Values: map[string]any{"topic": "accounts", "reviewed": "-40d", "audience": []string{"customers", "staff"}}},
			{Title: "Change the billing email", Values: map[string]any{"topic": "billing", "reviewed": "-75d", "audience": []string{"customers"}}},
			{Title: "Pages load slowly", Values: map[string]any{"topic": "trouble", "reviewed": "-5d", "audience": []string{"staff"}}},
		},
	},
	{
		ID: "how-to-guide", Name: "How-to guide", Description: "Step-by-step instructions with prerequisites, numbered steps and troubleshooting.", Icon: "i:compass:teal", Kind: "doc", Category: catDocs, Tags: []string{"instructions", "tutorial", "process"},
		markdown: `**Time needed:** 15 minutes · **Last checked:** {{date}}

## What this guide covers

One sentence on the outcome: by the end you will have ...

## Before you start

- [ ] You have access to
- [ ] You know the name of

## Steps

1. **Open the settings.** Where to click and what you should see.
2. **Fill in the details.** Which fields matter and which can stay as they are.
3. **Save and check.** How to tell that it worked.

> [!TIP]
> Add a screenshot under any step that is easy to get wrong.

## If something goes wrong

<details>
<summary>The button is greyed out</summary>

You may not have permission. Ask an admin to give you edit access.

</details>

<details>
<summary>The change does not show up</summary>

Refresh the page. Changes can take up to a minute to reach everyone.

</details>

## Related

-
`,
	},
	{
		ID: "glossary", Name: "Glossary", Description: "Shared definitions for the words your team uses, with examples and related terms.", Icon: "i:quote:cyan", Kind: "database", Category: catDocs, Tags: []string{"terms", "definitions", "vocabulary"},
		schema: []prop{
			{ID: "title", Name: "Term", Type: "title"},
			{ID: "definition", Name: "Definition", Type: "text"},
			{ID: "area", Name: "Area", Type: "select", Options: []opt{o("business", "Business", "orange"), o("product", "Product", "blue"), o("tech", "Technical", "purple")}},
			{ID: "owner", Name: "Owner", Type: "person"},
		},
		views: []view{
			{Name: "A to Z", Type: "table", Config: `{"sorts":[{"prop":"title","dir":"asc"}]}`},
			{Name: "By area", Type: "list", Config: `{"group":{"prop":"area"}}`},
		},
		rows: []row{
			{Title: "Activation", Values: map[string]any{"definition": "A new team that creates three pages and invites one person in its first week.", "area": "product"}},
			{Title: "ARR", Values: map[string]any{"definition": "Annual recurring revenue: the yearly value of active subscriptions.", "area": "business"}},
			{Title: "Churn", Values: map[string]any{"definition": "Paying customers who cancel in a period, divided by those at its start.", "area": "business"}},
			{Title: "Feature flag", Values: map[string]any{"definition": "A switch that turns code on for some people without a new release.", "area": "tech", "owner": "@me"}},
			{Title: "SLA", Values: map[string]any{"definition": "Service level agreement: the response and uptime promised to a customer.", "area": "tech"}},
		},
	},
	{
		ID: "faq", Name: "FAQ page", Description: "Questions and answers in collapsible sections, grouped by topic.", Icon: "i:chat:teal", Kind: "doc", Category: catDocs, Tags: []string{"questions", "answers", "help"},
		markdown: `Can't find your answer? Leave a comment on this page and we will add it.

## Getting started

<details>
<summary>Who is this for?</summary>

Anyone on the team who needs a quick answer without waiting for a reply.

</details>

<details>
<summary>Where do I start?</summary>

Read the team wiki first, then pick a buddy from the team directory.

</details>

## Accounts and access

<details>
<summary>How do I get access to a private page?</summary>

Open the page link and request access. The owner gets a notification.

</details>

<details>
<summary>Can I use my own laptop?</summary>

Yes, as long as disk encryption and a screen lock are on.

</details>

## Still stuck

> [!NOTE]
> Ask in the team channel and mention the page owner.
`,
	},
}
