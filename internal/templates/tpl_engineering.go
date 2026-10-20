package templates

const fence = "```"

var engStarters = []Starter{
	{
		ID: "bug-tracker", Name: "Bug tracker", Description: "Bugs with severity, status, component and assignee, on a triage board and a table.", Icon: "i:bug:rose", Kind: "database", Category: catEng, Tags: []string{"issues", "triage", "qa"},
		schema: []prop{
			{ID: "title", Name: "Bug", Type: "title"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("new", "New", "gray", "todo"), so("triaged", "Triaged", "yellow", "todo"), so("fixing", "Fixing", "blue", "doing"), so("fixed", "Fixed", "green", "done"), so("wontfix", "Won't fix", "brown", "done")}},
			{ID: "severity", Name: "Severity", Type: "select", Options: []opt{o("s1", "S1 · Critical", "red"), o("s2", "S2 · Major", "orange"), o("s3", "S3 · Minor", "yellow"), o("s4", "S4 · Cosmetic", "gray")}},
			{ID: "component", Name: "Component", Type: "select", Options: []opt{o("web", "Web app", "blue"), o("api", "API", "purple"), o("mobile", "Mobile", "pink"), o("billing", "Billing", "orange")}},
			{ID: "assignee", Name: "Assignee", Type: "person"},
			{ID: "reported", Name: "Reported", Type: "date"},
			{ID: "id", Name: "ID", Type: "unique_id", Prefix: "BUG"},
		},
		views: []view{
			{Name: "Triage", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "All bugs", Type: "table", Config: `{"sorts":[{"prop":"severity","dir":"asc"}]}`},
			{Name: "By component", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"component"},"series":{"prop":"severity"},"fn":"count","stacked":true}}}`},
		},
		rows: []row{
			{Title: "Checkout fails when the card needs 3-D Secure", Values: map[string]any{"status": "fixing", "severity": "s1", "component": "billing", "assignee": "@me", "reported": "-1d"}, Markdown: "## What happened\n\nThe payment step spins forever after the bank challenge.\n\n## Steps to reproduce\n\n1. Use a test card that requires a challenge\n2. Complete the challenge\n3. Wait on the confirmation screen\n\n## Expected\n\nThe order is confirmed and a receipt is emailed.\n"},
			{Title: "Avatar upload rotates photos", Values: map[string]any{"status": "triaged", "severity": "s3", "component": "mobile", "reported": "-4d"}},
			{Title: "API returns 500 on an empty filter", Values: map[string]any{"status": "new", "severity": "s2", "component": "api", "reported": "-2d"}},
			{Title: "Tooltip overlaps the menu", Values: map[string]any{"status": "fixed", "severity": "s4", "component": "web", "reported": "-9d"}},
		},
		templates: []row{
			{Title: "Bug report", Values: map[string]any{"status": "new"}, Markdown: "## What happened\n\n## What should have happened\n\n## Steps to reproduce\n\n1. \n2. \n\n## Environment\n\n- Browser or device:\n- Version:\n"},
		},
	},
	{
		ID: "design-doc", Name: "Design doc", Description: "A technical proposal or RFC: context, goals, the design, alternatives and rollout.", Icon: "i:layers:blue", Kind: "doc", Category: catEng, Tags: []string{"rfc", "architecture", "proposal"},
		markdown: `| Author | Reviewers | Status | Updated |
| --- | --- | --- | --- |
|  |  | Draft | {{date}} |

<!-- toc -->

## Summary

Three sentences: the problem, the proposed change, and the expected result.

## Context

What exists today and why it is no longer enough. Link to data, tickets or incidents.

## Goals and non-goals

**Goals**

-

**Non-goals**

-

## Proposed design

Describe the components and how data flows between them.

` + fence + `text
client -> api -> queue -> worker -> store
` + fence + `

### Data model

| Field | Type | Notes |
| --- | --- | --- |
| id | string | Primary key |
| created_at | timestamp |  |

### API changes

-

## Alternatives considered

| Option | Pros | Cons |
| --- | --- | --- |
| Do nothing |  |  |
| Option B |  |  |

## Risks and open questions

> [!WARNING]
> List anything that could make this design wrong, and how you will find out early.

- [ ]

## Rollout plan

1. Behind a flag for internal users
2. 10% of workspaces, watch error rates for a week
3. Everyone, then remove the flag
`,
	},
	{
		ID: "incident-postmortem", Name: "Incident postmortem", Description: "A blameless write-up: impact, timeline, root cause and the actions that prevent a repeat.", Icon: "i:flame:orange", Kind: "doc", Category: catEng, Tags: []string{"incident", "outage", "blameless"},
		markdown: `| Severity | Date | Duration | Author |
| --- | --- | --- | --- |
| SEV 2 | {{date}} |  |  |

> [!NOTE]
> This review is blameless. We look for causes in systems and processes, not people.

## Summary

What broke, who was affected, and how it was fixed, in a short paragraph.

## Impact

- **Customers affected:**
- **Requests failed:**
- **Revenue or data impact:**

## Timeline

| Time (UTC) | Event |
| --- | --- |
| 09:12 | Alert fires for raised error rate |
| 09:15 | On-call engineer acknowledges |
| 09:31 | Cause identified |
| 09:40 | Fix deployed, errors return to normal |

## Root cause

Ask "why" until you reach something you can change.

1. Why did requests fail?
2. Why did that happen?
3. Why was it not caught earlier?

## What went well

-

## What could have gone better

-

## Action items

| Action | Type | Owner | Due |
| --- | --- | --- | --- |
|  | Prevent |  |  |
|  | Detect |  |  |
|  | Mitigate |  |  |
`,
	},
	{
		ID: "runbook", Name: "Runbook", Description: "Operational steps for a service: health checks, common alerts and how to recover.", Icon: "i:terminal:green", Kind: "doc", Category: catEng, Tags: []string{"operations", "on-call", "sre"},
		markdown: `**Service:**  · **Owning team:**  · **Last tested:** {{date}}

> [!IMPORTANT]
> If customers are affected, open an incident first, then follow this page.

## Quick links

| What | Where |
| --- | --- |
| Dashboard |  |
| Logs |  |
| Alerts |  |
| Escalation |  |

## Health check

` + fence + `sh
curl -fsS https://service.internal/healthz
` + fence + `

A healthy service answers within 200 ms with status ok.

## Common alerts

<details>
<summary>High error rate</summary>

1. Check the latest deploy. If it is under an hour old, roll back.
2. Look for a failing dependency on the dashboard.
3. Escalate if errors stay above 2% for 10 minutes.

</details>

<details>
<summary>Queue backlog growing</summary>

1. Check worker count and restart stuck workers.
2. Scale workers up by two and watch the backlog for 5 minutes.

</details>

<details>
<summary>Disk almost full</summary>

1. Clear old logs older than 7 days.
2. If still above 85%, expand the volume and file a follow-up.

</details>

## Roll back a deploy

` + fence + `sh
deploy rollback --service api --to previous
` + fence + `

## Escalation

- [ ] Page the secondary on-call after 15 minutes without progress
- [ ] Post an update in the incident channel every 30 minutes
`,
	},
	{
		ID: "api-reference", Name: "API reference", Description: "Endpoint docs with authentication, parameters, example requests and error codes.", Icon: "i:code:violet", Kind: "doc", Category: catEng, Tags: []string{"api", "endpoints", "developer docs"},
		markdown: `<!-- toc -->

## Overview

Base URL: **https://api.example.com/v1** · All requests and responses use JSON.

## Authentication

Send a bearer token in the Authorization header.

` + fence + `http
Authorization: Bearer YOUR_TOKEN
` + fence + `

> [!WARNING]
> Keep tokens secret. Rotate a token at once if it appears in logs or a repository.

## List projects

**GET /projects**

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| limit | integer | No | Items per page, 1 to 100, default 20 |
| cursor | string | No | Cursor from the previous page |

` + fence + `json
{
  "items": [{ "id": "prj_123", "name": "Website" }],
  "next_cursor": null
}
` + fence + `

## Create a project

**POST /projects**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| name | string | Yes | 1 to 200 characters |
| color | string | No | One of blue, green, orange |

## Errors

| Status | Code | Meaning |
| --- | --- | --- |
| 400 | bad_request | The body or a parameter is not valid |
| 401 | unauthorized | Missing or expired token |
| 404 | not_found | No such resource |
| 429 | rate_limited | Too many requests, retry after the given seconds |

## Changelog

- {{date}}: first version of this page
`,
	},
	{
		ID: "code-review-checklist", Name: "Code review checklist", Description: "What to check before approving a change: correctness, tests, security and readability.", Icon: "i:checklist:blue", Kind: "doc", Category: catEng, Tags: []string{"pull request", "review", "quality"},
		markdown: `Use this list as a guide, not a gate. Comment on what matters and skip what does not apply.

## Before you start

- [ ] I understand what the change is for and have read the linked issue
- [ ] The change is small enough to review in one sitting

## Correctness

- [ ] It does what the description says
- [ ] Edge cases are handled: empty input, large input, errors, retries
- [ ] No behaviour changes slipped in that the description does not mention

## Tests

- [ ] New behaviour is covered by tests that would fail without the change
- [ ] Tests are readable and do not depend on timing or order

## Security and data

- [ ] Input from users is validated and escaped
- [ ] No secrets, tokens or personal data in code or logs
- [ ] Permission checks happen on the server

## Readability

- [ ] Names say what things are
- [ ] Complex parts have a short explanation where a reader will need it
- [ ] Dead code and debugging output are removed

## Operations

- [ ] Migrations are safe to run on a live database
- [ ] Errors are logged with enough context to debug

> [!TIP]
> Prefix comments with "nit:" when they are optional, so authors know what blocks the merge.
`,
	},
	{
		ID: "release-notes", Name: "Release notes", Description: "A release write-up for users: highlights, improvements, fixes and anything to act on.", Icon: "i:tag:green", Kind: "doc", Category: catEng, Tags: []string{"changelog", "release", "announcement"},
		markdown: `**Version:** 2.4.0 · **Released:** {{date}}

## Highlights

### Faster search

Results now appear as you type, and filters remember your last choice.

### Shared templates

Save any page as a template and your whole team can start from it.

## Improvements

- Tables scroll smoothly with thousands of rows
- Dates follow your language and region settings
- Keyboard shortcuts work inside dialogs

## Fixes

- Fixed a crash when pasting a very large image
- Comments no longer jump when someone else replies
- Exported files keep their original names

> [!WARNING]
> **Action needed:** API tokens created before version 2.0 stop working next month. Create a new token in settings.

## Thanks

Thanks to everyone who reported issues and tested the beta.
`,
	},
}
