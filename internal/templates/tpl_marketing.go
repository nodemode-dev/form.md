package templates

var marketingStarters = []Starter{
	{
		ID: "campaign-brief", Name: "Campaign brief", Description: "One page that aligns a campaign: objective, audience, message, channels, budget and dates.", Icon: "i:flag:orange", Kind: "doc", Category: catMarketing, Tags: []string{"campaign", "brief", "planning"},
		markdown: `| Campaign | Owner | Live dates | Budget |
| --- | --- | --- | --- |
|  |  | {{date}} to  |  |

## Objective

What should change because of this campaign? Make it measurable.

## Audience

- **Who:**
- **What they care about:**
- **Where they spend time:**

## Key message

> [!IMPORTANT]
> The one thing we want the audience to remember.

**Supporting points**

1.
2.
3.

## Channels and assets

| Channel | Asset | Owner | Due |
| --- | --- | --- | --- |
| Email | Announcement email |  |  |
| Social | Three posts and a short video |  |  |
| Website | Landing page |  |  |
| Paid | Two ad variants |  |  |

## Budget

| Item | Amount |
| --- | --- |
| Paid media |  |
| Design and video |  |
| **Total** |  |

## Measuring success

| Metric | Target |
| --- | --- |
| Sign-ups |  |
| Cost per sign-up |  |

## Approvals

- [ ] Marketing lead
- [ ] Product owner
- [ ] Legal, if claims or pricing are mentioned
`,
	},
	{
		ID: "seo-tracker", Name: "SEO tracker", Description: "Target keywords with intent, volume, current rank and the page that should win them.", Icon: "i:chart-line:green", Kind: "database", Category: catMarketing, Tags: []string{"keywords", "search", "rankings"},
		schema: []prop{
			{ID: "title", Name: "Keyword", Type: "title"},
			{ID: "intent", Name: "Intent", Type: "select", Options: []opt{o("info", "Informational", "blue"), o("commercial", "Commercial", "orange"), o("nav", "Navigational", "gray")}},
			{ID: "volume", Name: "Monthly volume", Type: "number", Format: "comma"},
			{ID: "rank", Name: "Current rank", Type: "number"},
			{ID: "target", Name: "Target page", Type: "url"},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("research", "Research", "gray", "todo"), so("writing", "Writing", "yellow", "doing"), so("live", "Live", "blue", "doing"), so("ranking", "Ranking", "green", "done")}},
			{ID: "checked", Name: "Last checked", Type: "date"},
		},
		views: []view{
			{Name: "Keywords", Type: "table", Config: `{"sorts":[{"prop":"volume","dir":"desc"}]}`},
			{Name: "Content status", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "Volume by intent", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"intent"},"fn":"sum","prop":"volume","labels":true}}}`},
		},
		rows: []row{
			{Title: "team wiki software", Values: map[string]any{"intent": "commercial", "volume": 2400, "rank": 14, "target": "https://example.com/wiki", "status": "live", "checked": "-2d"}},
			{Title: "how to write meeting notes", Values: map[string]any{"intent": "info", "volume": 5400, "rank": 6, "target": "https://example.com/blog/meeting-notes", "status": "ranking", "checked": "-2d"}},
			{Title: "self-hosted project management", Values: map[string]any{"intent": "commercial", "volume": 880, "status": "writing", "checked": "-9d"}},
			{Title: "okr template", Values: map[string]any{"intent": "info", "volume": 3600, "status": "research", "checked": "-9d"}},
		},
	},
	{
		ID: "social-posts", Name: "Social media planner", Description: "Posts by platform and publish date, with drafts, approvals and a calendar.", Icon: "i:chat:pink", Kind: "database", Category: catMarketing, Tags: []string{"social", "posts", "schedule"},
		schema: []prop{
			{ID: "title", Name: "Post", Type: "title"},
			{ID: "date", Name: "Publish date", Type: "date"},
			{ID: "platform", Name: "Platform", Type: "multi_select", Options: []opt{o("linkedin", "LinkedIn", "blue"), o("x", "X", "gray"), o("instagram", "Instagram", "pink"), o("youtube", "YouTube", "red")}},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("idea", "Idea", "gray", "todo"), so("draft", "Draft", "yellow", "doing"), so("approved", "Approved", "blue", "doing"), so("posted", "Posted", "green", "done")}},
			{ID: "owner", Name: "Owner", Type: "person"},
			{ID: "link", Name: "Live link", Type: "url"},
		},
		views: []view{
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"}}`},
			{Name: "Workflow", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "Posts", Type: "feed", Config: `{}`},
		},
		rows: []row{
			{Title: "Behind the scenes of the new release", Values: map[string]any{"date": "+2d", "platform": []string{"linkedin", "x"}, "status": "approved", "owner": "@me"}, Markdown: "We rebuilt search from the ground up. Here is what we learned along the way.\n\n1. Measure before you optimise\n2. Ship behind a flag\n3. Ask five customers first\n"},
			{Title: "Customer story: a week in their workspace", Values: map[string]any{"date": "+6d", "platform": []string{"linkedin", "youtube"}, "status": "draft"}},
			{Title: "Template of the week: OKRs", Values: map[string]any{"date": "+9d", "platform": []string{"instagram", "x"}, "status": "idea"}},
			{Title: "Hiring: product designer", Values: map[string]any{"date": "-3d", "platform": []string{"linkedin"}, "status": "posted", "link": "https://example.com/jobs/designer"}},
		},
	},
	{
		ID: "brand-guidelines", Name: "Brand guidelines", Description: "Voice, logo use, colours and type in one reference anyone can follow.", Icon: "i:brush:violet", Kind: "doc", Category: catMarketing, Tags: []string{"brand", "style guide", "voice"},
		markdown: `<!-- toc -->

## Who we are

One paragraph that anyone could read aloud to explain the brand.

## Voice and tone

| We are | We are not |
| --- | --- |
| Clear | Vague or clever for its own sake |
| Warm | Overly casual |
| Confident | Arrogant |

> [!TIP]
> Read your copy aloud. If it sounds like a press release, simplify it.

## Logo

- Leave clear space equal to the height of the mark on every side
- Use the full-colour logo on light backgrounds and the white logo on dark ones
- Never stretch, rotate, recolour or add effects

## Colour

| Name | Use | Value |
| --- | --- | --- |
| Primary | Buttons, links, highlights |  |
| Ink | Body text |  |
| Paper | Backgrounds |  |
| Accent | Charts and illustrations |  |

## Typography

- **Headings:** one family, semibold
- **Body:** one family, regular, 16 px on the web
- Use sentence case for headings and buttons

<details>
<summary>Words we use and avoid</summary>

| Use | Avoid |
| --- | --- |
| Workspace | Platform |
| Sign in | Log on |
| People | Users, in customer-facing copy |

</details>

## Assets

- [ ] Logo files
- [ ] Slide template
- [ ] Social image templates
`,
	},
	{
		ID: "event-plan", Name: "Event plan", Description: "Plan a meetup, webinar or conference: goals, logistics, run of show and follow-up.", Icon: "i:star:amber", Kind: "doc", Category: catMarketing, Tags: []string{"event", "webinar", "logistics"},
		markdown: `| Event | Date | Format | Owner |
| --- | --- | --- | --- |
|  | {{date}} | In person |  |

## Goals

- **Attendees:**
- **Qualified leads:**
- **Content produced:**

## Logistics

| Item | Details | Status |
| --- | --- | --- |
| Venue or platform |  | Booked |
| Catering |  | Not started |
| Speakers |  | Confirmed |
| Registration page |  | Live |
| Swag and signage |  | Not started |

## Run of show

| Time | Item | Who |
| --- | --- | --- |
| 18:00 | Doors and welcome |  |
| 18:30 | Talk one |  |
| 19:00 | Talk two |  |
| 19:30 | Networking |  |

## Promotion

- [ ] Announcement email four weeks out
- [ ] Social posts weekly until the event
- [ ] Reminder email the day before

> [!WARNING]
> Confirm the accessibility of the venue and share it on the registration page.

## After the event

- [ ] Send slides and recording to attendees
- [ ] Hand leads to sales within two days
- [ ] Write a short recap with numbers
`,
	},
}
