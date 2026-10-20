package templates

var teamStarters = []Starter{
	{
		ID: "onboarding-checklist", Name: "Onboarding checklist", Description: "A new joiner's first 30 days: accounts, people to meet, reading and first goals.", Icon: "i:checklist:teal", Kind: "doc", Category: catTeam, Tags: []string{"new hire", "first week", "welcome"},
		markdown: `**Welcome,** ! · **Start date:** {{date}} · **Manager:**  · **Buddy:**

> [!NOTE]
> Tick things off as you go. Your buddy checks in at the end of each week.

## Before day one

- [ ] Laptop shipped and set up
- [ ] Email and workspace accounts created
- [ ] Calendar invites for the first week sent

## Week one

- [ ] Meet your manager and agree on 30-day goals
- [ ] Coffee with your buddy
- [ ] Read the company handbook and team wiki
- [ ] Set up the password manager and two-factor sign-in
- [ ] Ship one small change, however tiny

## People to meet

| Who | Why | Done |
| --- | --- | --- |
|  | Runs the team you work with most |  |
|  | Knows the history of the product |  |
|  | Your first customer contact |  |

## Weeks two to four

- [ ] Shadow a customer call
- [ ] Take on your first project
- [ ] Share what surprised you so far in a short note

<details>
<summary>Useful links</summary>

- Team directory
- Time off tracker
- Expense policy

</details>

## 30-day check-in

- **What is going well:**
- **What is confusing:**
- **Goals for the next 60 days:**
`,
	},
	{
		ID: "hiring-pipeline", Name: "Hiring pipeline", Description: "Candidates by stage and role with interviewers, scores and next steps.", Icon: "i:briefcase:blue", Kind: "database", Category: catTeam, Tags: []string{"recruiting", "candidates", "ats"},
		schema: []prop{
			{ID: "title", Name: "Candidate", Type: "title"},
			{ID: "stage", Name: "Stage", Type: "status", Options: []opt{so("applied", "Applied", "gray", "todo"), so("screen", "Screen", "yellow", "doing"), so("interview", "Interviews", "blue", "doing"), so("offer", "Offer", "purple", "doing"), so("hired", "Hired", "green", "done"), so("rejected", "Not moving forward", "red", "done")}},
			{ID: "role", Name: "Role", Type: "select", Options: []opt{o("eng", "Backend engineer", "blue"), o("design", "Product designer", "pink"), o("sales", "Account executive", "green")}},
			{ID: "source", Name: "Source", Type: "select", Options: []opt{o("referral", "Referral", "green"), o("board", "Job board", "blue"), o("outreach", "Outreach", "purple")}},
			{ID: "owner", Name: "Hiring manager", Type: "person"},
			{ID: "score", Name: "Score", Type: "number"},
			{ID: "next", Name: "Next step", Type: "date"},
			{ID: "email", Name: "Email", Type: "email"},
		},
		views: []view{
			{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"stage"}}`},
			{Name: "By role", Type: "table", Config: `{"group":{"prop":"role"},"sorts":[{"prop":"score","dir":"desc"}]}`},
			{Name: "Upcoming", Type: "calendar", Config: `{"layout":{"date":"next"}}`},
		},
		rows: []row{
			{Title: "Priya Raman", Values: map[string]any{"stage": "interview", "role": "eng", "source": "referral", "owner": "@me", "score": 4, "next": "+2d", "email": "priya@example.com"}},
			{Title: "Tom Okafor", Values: map[string]any{"stage": "offer", "role": "design", "source": "board", "score": 5, "next": "+1d", "email": "tom@example.com"}},
			{Title: "Lena Fischer", Values: map[string]any{"stage": "screen", "role": "sales", "source": "outreach", "next": "+4d", "email": "lena@example.com"}},
			{Title: "Marco Silva", Values: map[string]any{"stage": "applied", "role": "eng", "source": "board", "email": "marco@example.com"}},
		},
		templates: []row{
			{Title: "Candidate", Values: map[string]any{"stage": "applied"}, Markdown: "## Summary\n\n## Interview notes\n\n## Decision\n"},
		},
	},
	{
		ID: "interview-scorecard", Name: "Interview scorecard", Description: "A structured scorecard: competencies, evidence, scores and a clear recommendation.", Icon: "i:star:violet", Kind: "doc", Category: catTeam, Tags: []string{"interview", "hiring", "evaluation"},
		markdown: `| Candidate | Role | Interviewer | Date |
| --- | --- | --- | --- |
|  |  |  | {{date}} |

> [!IMPORTANT]
> Write your scorecard before you discuss the candidate with anyone else.

## Scoring guide

| Score | Meaning |
| --- | --- |
| 1 | Strong no: clear evidence against |
| 2 | Leaning no |
| 3 | Leaning yes |
| 4 | Strong yes: clear evidence for |

## Competencies

| Competency | Evidence | Score |
| --- | --- | --- |
| Problem solving |  |  |
| Communication |  |  |
| Craft and depth in the role |  |  |
| Collaboration |  |  |
| Ownership |  |  |

## Questions asked

1.
2.
3.

## Strengths

-

## Concerns

-

## Recommendation

- [ ] Strong yes
- [ ] Yes
- [ ] No
- [ ] Strong no
`,
	},
	{
		ID: "team-directory", Name: "Team directory", Description: "Everyone on the team with role, team, location and contact details, as cards and a map.", Icon: "i:users:violet", Kind: "database", Category: catTeam, Tags: []string{"people", "org chart", "contacts"},
		schema: []prop{
			{ID: "title", Name: "Name", Type: "title"},
			{ID: "role", Name: "Role", Type: "text"},
			{ID: "team", Name: "Team", Type: "select", Options: []opt{o("product", "Product", "blue"), o("eng", "Engineering", "purple"), o("sales", "Sales", "green"), o("ops", "Operations", "brown")}},
			{ID: "email", Name: "Email", Type: "email"},
			{ID: "phone", Name: "Phone", Type: "phone"},
			{ID: "where", Name: "Based in", Type: "place"},
			{ID: "started", Name: "Start date", Type: "date"},
		},
		views: []view{
			{Name: "People", Type: "gallery", Config: `{"layout":{"cover":"none","size":"small"}}`},
			{Name: "By team", Type: "table", Config: `{"group":{"prop":"team"}}`},
			{Name: "Map", Type: "map", Config: `{"layout":{"place":"where"}}`},
		},
		rows: []row{
			{Title: "Amara Nwosu", Values: map[string]any{"role": "Head of Product", "team": "product", "email": "amara@example.com", "phone": "+44 20 7946 0101", "where": map[string]any{"lat": 51.5072, "lng": -0.1276, "name": "London"}, "started": "-700d"}},
			{Title: "Kenji Watanabe", Values: map[string]any{"role": "Staff Engineer", "team": "eng", "email": "kenji@example.com", "where": map[string]any{"lat": 35.6762, "lng": 139.6503, "name": "Tokyo"}, "started": "-420d"}},
			{Title: "Sofia Moreno", Values: map[string]any{"role": "Account Executive", "team": "sales", "email": "sofia@example.com", "where": map[string]any{"lat": 40.4168, "lng": -3.7038, "name": "Madrid"}, "started": "-120d"}},
			{Title: "Noah Becker", Values: map[string]any{"role": "Operations Lead", "team": "ops", "email": "noah@example.com", "where": map[string]any{"lat": 52.52, "lng": 13.405, "name": "Berlin"}, "started": "-30d"}},
		},
	},
	{
		ID: "time-off", Name: "PTO tracker", Description: "Leave requests with type, dates and approval, on a team calendar and a timeline.", Icon: "i:plane:cyan", Kind: "database", Category: catTeam, Tags: []string{"leave", "vacation", "time off"},
		schema: []prop{
			{ID: "title", Name: "Request", Type: "title"},
			{ID: "person", Name: "Person", Type: "person"},
			{ID: "type", Name: "Type", Type: "select", Options: []opt{o("vacation", "Vacation", "blue"), o("sick", "Sick", "red"), o("parental", "Parental", "pink"), o("other", "Other", "gray")}},
			{ID: "start", Name: "First day", Type: "date"},
			{ID: "end", Name: "Last day", Type: "date"},
			{ID: "days", Name: "Working days", Type: "number"},
			{ID: "status", Name: "Approval", Type: "status", Options: []opt{so("requested", "Requested", "yellow", "todo"), so("approved", "Approved", "green", "done"), so("declined", "Declined", "red", "done")}},
		},
		views: []view{
			{Name: "Team calendar", Type: "calendar", Config: `{"layout":{"date":"start"}}`},
			{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"start","end":"end","zoom":"week"}}`},
			{Name: "Requests", Type: "table", Config: `{"sorts":[{"prop":"start","dir":"asc"}],"calcs":{"days":"sum"}}`},
		},
		rows: []row{
			{Title: "Summer holiday", Values: map[string]any{"person": "@me", "type": "vacation", "start": "+21d", "end": "+32d", "days": 8, "status": "approved"}},
			{Title: "Long weekend", Values: map[string]any{"type": "vacation", "start": "+9d", "end": "+10d", "days": 2, "status": "requested"}},
			{Title: "Doctor's appointment", Values: map[string]any{"type": "other", "start": "+3d", "end": "+3d", "days": 1, "status": "approved"}},
			{Title: "Flu", Values: map[string]any{"type": "sick", "start": "-6d", "end": "-4d", "days": 3, "status": "approved"}},
		},
	},
	{
		ID: "performance-review", Name: "Performance review", Description: "A fair, evidence-based review: goals, achievements, feedback and a growth plan.", Icon: "i:trophy:amber", Kind: "doc", Category: catTeam, Tags: []string{"review", "feedback", "growth"},
		markdown: `| Person | Reviewer | Period | Date |
| --- | --- | --- | --- |
|  |  | H2 | {{date}} |

## Goals for the period

| Goal | Result | Rating |
| --- | --- | --- |
|  |  | Met |
|  |  | Exceeded |
|  |  | Partly met |

## Self-reflection

<details>
<summary>Questions for the person being reviewed</summary>

- What are you most proud of this period?
- Where did you get stuck, and what helped?
- What do you want to learn next?

</details>

-

## Achievements

-

## Strengths

-

## Areas to grow

> [!TIP]
> Give one concrete example for every piece of feedback.

-

## Peer feedback

| From | Highlights |
| --- | --- |
|  |  |

## Growth plan for next period

- [ ]
- [ ]

## Overall

- **Rating:**
- **Summary in two sentences:**
`,
	},
}
