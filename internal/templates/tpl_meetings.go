package templates

var meetingStarters = []Starter{
	{
		ID: "standup-notes", Name: "Standup notes", Description: "A quick daily check-in: yesterday, today and anything blocking, per person.", Icon: "i:sun:amber", Kind: "doc", Category: catMeetings, Tags: []string{"daily", "scrum", "check-in"},
		markdown: `**Date:** {{date}} · **Facilitator:**

> [!TIP]
> Keep it to 15 minutes. Anything that needs discussion goes to the parking lot.

## Updates

| Person | Yesterday | Today | Blockers |
| --- | --- | --- | --- |
|  |  |  |  |
|  |  |  |  |
|  |  |  |  |

## Parking lot

-

## Follow-ups

- [ ]
- [ ]
`,
	},
	{
		ID: "retrospective", Name: "Retrospective", Description: "Look back on a sprint or project: what went well, what did not, and what to change.", Icon: "i:clock:violet", Kind: "doc", Category: catMeetings, Tags: []string{"retro", "sprint", "lessons"},
		markdown: `**Sprint or project:**  · **Date:** {{date}}

## Check-in

In one word, how did this stretch feel?

## What went well

-
-

## What did not go well

-
-

## What puzzled us

-

## Actions

| Action | Owner | Due |
| --- | --- | --- |
|  |  |  |
|  |  |  |

> [!NOTE]
> Start next retro by checking these actions. Unfinished ones carry over or get dropped on purpose.

<details>
<summary>Prompts if the room is quiet</summary>

- What would you do differently with the same time again?
- What slowed you down that nobody saw?
- Who helped you, and how?

</details>
`,
	},
	{
		ID: "all-hands-agenda", Name: "All-hands agenda", Description: "A company meeting run of show with timings, speakers, updates and open questions.", Icon: "i:megaphone:violet", Kind: "doc", Category: catMeetings, Tags: []string{"company", "town hall", "agenda"},
		markdown: `**Date:** {{date}} · **Host:**  · **Recording:**

## Run of show

| Time | Segment | Speaker |
| --- | --- | --- |
| 0:00 | Welcome and new joiners |  |
| 0:05 | Company numbers |  |
| 0:15 | Team spotlights |  |
| 0:35 | Questions from everyone |  |
| 0:55 | Shout-outs and close |  |

## Company numbers

- **Customers:**
- **Revenue:**
- **Hiring:**

## Team spotlights

### Team one

- What shipped
- What is next

### Team two

- What shipped
- What is next

## Questions

> [!NOTE]
> Add questions below before the meeting. Upvote others with a comment.

-

## Shout-outs

-
`,
	},
	{
		ID: "one-on-one", Name: "1:1 notes", Description: "A running page for one-to-ones: topics, feedback, growth and agreed next steps.", Icon: "i:users:blue", Kind: "doc", Category: catMeetings, Tags: []string{"one-on-one", "manager", "feedback"},
		markdown: `**With:**  · **Cadence:** weekly, 30 minutes

> [!TIP]
> Either of you can add topics during the week. Newest meeting goes on top.

## {{date}}

### Their topics

-

### My topics

-

### Feedback

- **Going well:**
- **To work on:**

### Next steps

- [ ]
- [ ]

<details>
<summary>Growth goals</summary>

| Goal | Why it matters | Progress |
| --- | --- | --- |
|  |  |  |

</details>

<details>
<summary>Questions to try</summary>

- What is energising you at the moment, and what is draining you?
- Is there anything you need from me that you are not getting?
- What would make next week a great week?

</details>
`,
	},
	{
		ID: "meeting-log", Name: "Meeting log", Description: "Every meeting in one database with type, date and attendees, on a calendar and a list.", Icon: "i:calendar:violet", Kind: "database", Category: catMeetings, Tags: []string{"meetings", "minutes", "calendar"},
		schema: []prop{
			{ID: "title", Name: "Meeting", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "type", Name: "Type", Type: "select", Options: []opt{o("planning", "Planning", "blue"), o("review", "Review", "purple"), o("customer", "Customer", "green"), o("oneone", "1:1", "pink")}},
			{ID: "attendees", Name: "Attendees", Type: "person"},
			{ID: "done", Name: "Notes written", Type: "checkbox"},
		},
		views: []view{
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"}}`},
			{Name: "All meetings", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}]}`},
			{Name: "By type", Type: "list", Config: `{"group":{"prop":"type"}}`},
		},
		rows: []row{
			{Title: "Sprint planning", Values: map[string]any{"date": "-1d", "type": "planning", "attendees": "@me", "done": true}, Markdown: "## Agenda\n\n1. Review last sprint\n2. Pick stories\n\n## Decisions\n\n- Sprint goal: magic link sign-in\n\n## Action items\n\n- [x] Share the sprint goal\n- [ ] Book the demo slot\n"},
			{Title: "Quarterly review with Northwind", Values: map[string]any{"date": "+2d", "type": "customer", "done": false}},
			{Title: "Design review: onboarding", Values: map[string]any{"date": "+4d", "type": "review", "attendees": "@me", "done": false}},
			{Title: "Weekly 1:1", Values: map[string]any{"date": "-3d", "type": "oneone", "done": true}},
		},
		templates: []row{
			{Title: "Meeting", Markdown: "## Agenda\n\n1. \n\n## Notes\n\n- \n\n## Action items\n\n- [ ] \n"},
		},
	},
}
