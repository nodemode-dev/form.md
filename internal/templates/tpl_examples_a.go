package templates

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

var exampleStartersA = []Starter{xaLaunch(), xaStartup(), xaMarketing(), xaSales(), xaEngineering(), xaSuccess(), xaResearch()}

const xaThisMonth = `{"prop":"month","cond":"on_or_after","value":{"relative":"one_week_ago"}}`

func xaDay(n int) string { return fmt.Sprintf("%+dd", n) }

func xaSpan(a, b int) map[string]any { return map[string]any{"start": xaDay(a), "end": xaDay(b)} }

func xaNoise(seed, i int) float64 {
	x := uint32(seed)*2654435761 ^ uint32(i+7)*2246822519
	x ^= x >> 15
	x *= 2246822519
	x ^= x >> 13
	x *= 3266489917
	x ^= x >> 16
	return float64(x%100000) / 100000
}

func xaMonthAgo(k int) int { return -int(math.Round(float64(k) * 30.4375)) }

func xaInt(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func xaMoney(n float64) string { return "$" + xaInt(int(math.Round(n))) }

func xaFill(md string, kv ...string) string { return strings.NewReplacer(kv...).Replace(md) }

func xaCards(ids ...string) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = `{"id":"` + id + `","hidden":false}`
	}
	return `"props":[` + strings.Join(parts, ",") + `]`
}

var xaStatus = []opt{so("todo", "Not started", "gray", "todo"), so("doing", "In progress", "blue", "doing"), so("done", "Done", "green", "done")}

type xaLaunchTask struct {
	title  string
	ms     int
	owner  string
	status string
	due    int
	hours  int
	pri    string
}

type xaFeedback struct {
	title, tester, company, mood string
	score                        int
	area                         string
	day                          int
	source                       string
}

func xaLaunch() Starter {
	people := []opt{o("maya", "Maya Chen", "purple"), o("diego", "Diego Alvarez", "blue"), o("priya", "Priya Raman", "pink"), o("tom", "Tom Becker", "orange"), o("aisha", "Aisha Okafor", "green"), o("lars", "Lars Nilsson", "brown")}
	team := map[string]string{"maya": "product", "diego": "eng", "priya": "design", "tom": "marketing", "aisha": "support", "lars": "sales"}
	milestones := []struct {
		title, phase, owner, status string
		from, to                    int
		goal                        string
	}{
		{"Feature freeze", "build", "diego", "done", -42, -35, "Everything in 2.0 is merged and the release branch is cut."},
		{"Private beta", "beta", "maya", "done", -41, -14, "Forty invited teams use Beacon 2.0 every day and tell us what breaks."},
		{"Public beta", "beta", "maya", "doing", -13, 10, "Anyone on the waitlist can try 2.0; we fix what blocks a paid launch."},
		{"Pricing and packaging", "prep", "lars", "doing", -20, 6, "New plans, limits and a fair migration offer for 1.x customers."},
		{"Help center and training", "prep", "aisha", "doing", -10, 13, "Articles, videos and a support team ready for launch-week volume."},
		{"Press and analyst briefings", "prep", "tom", "todo", 4, 15, "Embargoed briefings so the story lands on launch morning."},
		{"Launch day", "launch", "maya", "todo", 18, 18, "Beacon 2.0 is on for every workspace and announced everywhere."},
		{"Post-launch review", "after", "maya", "todo", 25, 32, "Measure results against the brief and decide what comes next."},
	}
	tasks := []xaLaunchTask{
		{"Merge the shared inbox rewrite", 0, "diego", "done", -40, 24, "high"},
		{"Finish the rules engine API", 0, "diego", "done", -38, 16, "high"},
		{"Freeze the database schema", 0, "diego", "done", -36, 6, "medium"},
		{"Cut the 2.0 release branch", 0, "diego", "done", -35, 2, "medium"},
		{"Invite 40 beta teams", 1, "maya", "done", -40, 4, "high"},
		{"Set up the beta feedback form", 1, "aisha", "done", -39, 3, "medium"},
		{"Ship the in-app feedback widget", 1, "priya", "done", -34, 10, "medium"},
		{"Run five beta interviews", 1, "maya", "done", -22, 10, "high"},
		{"Fix the top 10 beta bugs", 1, "diego", "done", -15, 30, "high"},
		{"Open the public beta waitlist", 2, "tom", "done", -13, 4, "high"},
		{"Load test at five times expected traffic", 2, "diego", "done", -9, 12, "high"},
		{"Polish the onboarding checklist", 2, "priya", "doing", 2, 14, "high"},
		{"Fix delayed push notifications on Android", 2, "diego", "doing", 4, 12, "high"},
		{"Weekly beta digest email", 2, "tom", "doing", 3, 3, "low"},
		{"Triage public beta feedback", 2, "aisha", "doing", 6, 8, "medium"},
		{"Accessibility review", 2, "priya", "todo", 8, 8, "medium"},
		{"Finalise plan limits", 3, "lars", "done", -12, 6, "high"},
		{"Billing system plan codes", 3, "diego", "done", -6, 8, "medium"},
		{"Update the pricing page", 3, "tom", "doing", 3, 6, "high"},
		{"Migration offer for 1.x customers", 3, "lars", "doing", 5, 5, "medium"},
		{"Update in-app tooltips", 4, "priya", "done", -4, 5, "low"},
		{"Write 12 new help articles", 4, "aisha", "doing", 8, 20, "high"},
		{"Record three walkthrough videos", 4, "tom", "todo", 10, 9, "medium"},
		{"Train the support team", 4, "aisha", "todo", 12, 6, "high"},
		{"Draft the press release", 5, "tom", "doing", 6, 6, "high"},
		{"Book analyst briefings", 5, "tom", "todo", 9, 4, "medium"},
		{"Prepare the demo workspace", 5, "priya", "todo", 11, 6, "medium"},
		{"Collect customer quotes", 5, "lars", "todo", 12, 4, "medium"},
		{"Sales demo script", 6, "lars", "todo", 15, 5, "medium"},
		{"Publish the launch blog post", 6, "tom", "todo", 18, 3, "high"},
		{"Send the launch email", 6, "tom", "todo", 18, 2, "high"},
		{"Turn on 2.0 for every workspace", 6, "diego", "todo", 18, 1, "high"},
		{"Staff the launch-day war room", 6, "maya", "todo", 18, 8, "medium"},
		{"Collect launch metrics", 7, "maya", "todo", 25, 4, "medium"},
		{"Send the post-launch survey", 7, "aisha", "todo", 26, 3, "low"},
		{"Write the launch retrospective", 7, "maya", "todo", 30, 4, "medium"},
	}
	var taskRows []row
	byMs := map[int][]string{}
	for _, t := range tasks {
		byMs[t.ms] = append(byMs[t.ms], t.title)
		taskRows = append(taskRows, row{Title: t.title, Values: map[string]any{"status": t.status, "owner": t.owner, "team": team[t.owner], "due": xaDay(t.due), "hours": t.hours, "priority": t.pri}})
	}
	taskRows[11].Markdown = "## What good looks like\n\nA new team reaches its first shared conversation in under five minutes.\n\n- [x] Cut the checklist from nine steps to five\n- [x] Pre-fill the team name from the sign-up email\n- [ ] Add a sample conversation so the inbox is never empty\n- [ ] Test with three teams from the public beta\n"
	taskRows[12].Markdown = "> [!WARNING]\n> Four beta teams report replies arriving two to five minutes late on Android. This blocks launch.\n\n## Findings so far\n\n- The delay only happens when battery saver is on\n- High-priority messages arrive on time\n\n## Plan\n\n1. Send conversation replies as high-priority messages\n2. Add a fallback poll every 60 seconds while the app is open\n3. Ship to the beta channel and confirm with Mensah Couriers\n"
	var msRows []row
	for i, m := range milestones {
		msRows = append(msRows, row{Title: m.title, Values: map[string]any{"phase": m.phase, "owner": m.owner, "status": m.status, "when": xaSpan(m.from, m.to), "tasks": byMs[i], "goal": m.goal}})
	}
	msRows[6].Markdown = "Launch day runs to the minute in {{page:lt_runofshow}}. The success measures are in {{page:lt_brief}}.\n"
	feedback := []xaFeedback{
		{"Shared inbox feels instant now", "Rosa Delgado", "Pinecrest Dental", "pos", 9, "inbox", -38, "interview"},
		{"Auto-assign rules save us an hour a day", "Ben Okoro", "Harbor Bikes", "pos", 10, "rules", -36, "widget"},
		{"Setup took less than ten minutes", "Hana Sato", "Little Fern Bakery", "pos", 9, "onboarding", -35, "survey"},
		{"Would like more report templates", "Erik Holm", "Holm Builders", "neu", 6, "reports", -34, "survey"},
		{"Love the new keyboard shortcuts", "Owen Price", "Northgate Realty", "pos", 8, "inbox", -33, "widget"},
		{"Push notifications arrive minutes late on Android", "Kwame Mensah", "Mensah Couriers", "neg", 3, "mobile", -32, "widget"},
		{"Mobile replies are much faster", "Leila Haddad", "Cedar Yoga Studio", "pos", 8, "mobile", -31, "email"},
		{"Internal notes with mentions are a hit", "Marco Bianchi", "Bianchi Tiles", "pos", 9, "inbox", -30, "interview"},
		{"Calendar integration just worked", "Grace Liu", "Brightpath Tutors", "pos", 8, "integrations", -29, "widget"},
		{"Rules editor takes a while to learn", "Sofia Marin", "Marin Pilates", "neu", 6, "rules", -28, "interview"},
		{"Search finds old threads in a second", "Samuel Adeyemi", "Kora Logistics", "pos", 9, "performance", -27, "survey"},
		{"Inbox slowed down with 20,000 threads", "Greta Novak", "Novak Hardware", "neg", 4, "performance", -26, "email"},
		{"Saved replies cut our response time", "Ines Varga", "Varga Travel", "pos", 9, "rules", -25, "interview"},
		{"The onboarding checklist kept us on track", "Felix Wagner", "Wagner Auto Repair", "pos", 8, "onboarding", -24, "widget"},
		{"Missing an integration with our booking tool", "Daniel Kim", "Kim Physio", "neu", 5, "integrations", -23, "widget"},
		{"Customers noticed we reply faster", "Amara Nwosu", "Nwosu Legal", "pos", 10, "inbox", -22, "email"},
		{"Attachments over 10 MB fail to upload", "Ali Rahman", "Rahman Print Shop", "neg", 3, "inbox", -21, "widget"},
		{"Webhooks are exactly what we needed", "Kenji Mori", "Mori Studio", "pos", 9, "integrations", -20, "widget"},
		{"Price per seat adds up for part-time staff", "Abby Turner", "Turner Cafe", "neu", 5, "pricing", -19, "email"},
		{"Collision warnings stopped double replies", "Clara Jensen", "Fjord Outfitters", "pos", 9, "inbox", -18, "interview"},
		{"Calendar sync created duplicate events", "Ivy Walsh", "Walsh Events", "neg", 2, "integrations", -17, "interview"},
		{"Mobile app needs a unified inbox view", "Ravi Shah", "Shah Electronics", "neu", 6, "mobile", -15, "widget"},
		{"Lost a draft when switching conversations", "Pablo Ortiz", "Ortiz Motors", "neg", 3, "inbox", -14, "widget"},
		{"Export to a spreadsheet would help", "Zoe Clarke", "Clarke Accounting", "neu", 7, "reports", -13, "survey"},
		{"Push notifications are reliable on iPhone", "Diana Petrova", "Petrova Design", "pos", 8, "mobile", -12, "survey"},
		{"Pricing is fair for a team of five", "Joel Martin", "Martin and Sons Plumbing", "pos", 8, "pricing", -11, "email"},
		{"Dark mode looks great", "Yara Costa", "Costa Florals", "pos", 8, "mobile", -10, "widget"},
		{"Reports show exactly what our manager asks for", "Hassan Ali", "Ali Pharmacy", "pos", 9, "reports", -9, "survey"},
		{"Onboarding emails felt a bit frequent", "Liam Byrne", "Byrne Gardens", "neu", 6, "onboarding", -9, "email"},
		{"Importing from our old helpdesk was painless", "Nora Lindqvist", "Lindqvist Ceramics", "pos", 9, "onboarding", -8, "interview"},
		{"Good so far, still testing with one team", "Elena Popescu", "Popescu Clinic", "neu", 7, "inbox", -7, "survey"},
		{"Tags and filters make triage easy", "Paolo Greco", "Greco Wines", "pos", 8, "inbox", -6, "widget"},
		{"Push notifications still late for two of our drivers", "Kwame Mensah", "Mensah Couriers", "neg", 4, "mobile", -6, "email"},
		{"Customer satisfaction scores went up", "Lucy Evans", "Evans Pet Care", "pos", 9, "reports", -5, "survey"},
		{"Billing page shows the wrong plan", "Ruth Adler", "Adler Optics", "neg", 3, "pricing", -5, "email"},
		{"Wish rules could run on a schedule", "Omar Farouk", "Farouk Imports", "neu", 6, "rules", -4, "widget"},
		{"Assignment rules handle our night shift", "Victor Ramos", "Ramos Security", "pos", 8, "rules", -3, "widget"},
		{"Search misses results with accents", "Joana Silva", "Silva Translations", "neg", 4, "performance", -3, "widget"},
		{"The team adopted it without any training", "Mei Tanaka", "Tanaka Language School", "pos", 10, "onboarding", -2, "interview"},
		{"Unsure which plan fits a seasonal business", "Chloe Dubois", "Dubois Ski School", "neu", 5, "pricing", -1, "email"},
	}
	var fbRows []row
	scoreSum := 0
	moods := map[string]int{}
	for _, f := range feedback {
		ms := "Public beta"
		if f.day < -13 {
			ms = "Private beta"
		}
		scoreSum += f.score
		moods[f.mood]++
		fbRows = append(fbRows, row{Title: f.title, Values: map[string]any{"tester": f.tester, "company": f.company, "sentiment": f.mood, "score": f.score, "area": f.area, "received": xaDay(f.day), "source": f.source, "milestone": ms}})
	}
	var metricRows []row
	signups, paid, active := 0, 0, 40
	for d := 0; d < 42; d++ {
		off := d - 41
		var s, p int
		if off < -13 {
			s = 5 + int(7*xaNoise(11, d)) + d/6
			p = int(1.6 * xaNoise(12, d))
		} else {
			s = 38 + 4*(d-28) + int(26*xaNoise(13, d))
			p = 2 + int(5*xaNoise(14, d)) + (d-28)/5
		}
		trials := int(math.Round(float64(s) * (0.5 + 0.12*xaNoise(15, d))))
		active += int(math.Round(float64(s)*0.42)) - int(2*xaNoise(16, d))
		uptime := 99.9 + 0.1*xaNoise(17, d)
		if d == 19 {
			uptime = 98.6
		}
		if d == 33 {
			uptime = 99.4
		}
		signups += s
		paid += p
		metricRows = append(metricRows, row{Title: "Beta day " + strconv.Itoa(d+1), Values: map[string]any{"date": xaDay(off), "signups": s, "trials": trials, "paid": p, "active": active, "mrr": p * 120, "uptime": math.Round(uptime*100) / 100}})
	}
	avgScore := float64(scoreSum) / float64(len(feedback))
	nums := []string{"[[signups]]", xaInt(signups), "[[paid]]", xaInt(paid), "[[active]]", xaInt(active), "[[mrr]]", xaMoney(float64(paid * 120)), "[[score]]", strconv.FormatFloat(math.Round(avgScore*10)/10, 'f', 1, 64), "[[pos]]", strconv.Itoa(moods["pos"]), "[[neg]]", strconv.Itoa(moods["neg"]), "[[fb]]", strconv.Itoa(len(feedback)), "[[tasks]]", strconv.Itoa(len(tasks))}
	return Starter{
		ID: "product-launch-hq", Name: "Product launch HQ", Description: "Run a software launch end to end: milestones, tasks, beta feedback and a live metrics dashboard.", Icon: "i:rocket:orange", Kind: "doc", Category: catExamples, Tags: []string{"launch", "milestones", "beta", "metrics"},
		markdown: xaFill(`Lanternfish Software ships **Beacon 2.0**, a faster shared inbox for small teams, in about two and a half weeks. This page is the launch headquarters: everything the team needs is one click away.

> [!TIP]
> Start with the {{page:lt_brief}} for the why, then use the timeline below to see where we are today.

## Where we are

The private beta is done and the public beta is running. [[signups]] teams have signed up so far, [[active]] teams are active and [[paid]] have already moved to a paid plan. Beta testers rate 2.0 at [[score]] out of 10 on average.

{{db:lt_ms}}

## Launch numbers

The dashboard updates as the team logs each day. Hover any chart for exact values.

{{db:lt_metrics}}

## How the pieces fit

| Page | What it is for |
| --- | --- |
| {{page:lt_ms}} | Eight milestones on a timeline, each rolling up progress from its tasks |
| {{page:lt_tasks}} | [[tasks]] launch tasks on a board, a calendar and a workload chart |
| {{page:lt_fb}} | [[fb]] pieces of beta feedback with sentiment, scores and an intake form |
| {{page:lt_metrics}} | Daily sign-ups, trials, conversions and uptime |
| {{page:lt_brief}} | The one-page launch brief and success measures |
| {{page:lt_gtm}} | Audience, channels, messaging and the launch-day run of show |

## This week

- [x] Load test at five times expected traffic
- [ ] Fix delayed push notifications on Android (blocks launch)
- [ ] Finish the onboarding checklist polish
- [ ] Draft the press release
`, nums...),
		children: []Starter{
			{
				key: "lt_tasks", Name: "Launch tasks", Icon: "i:checklist:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Task", Type: "title"},
					{ID: "status", Name: "Status", Type: "status", Options: xaStatus},
					{ID: "owner", Name: "Owner", Type: "select", Options: people},
					{ID: "team", Name: "Team", Type: "select", Options: []opt{o("product", "Product", "purple"), o("eng", "Engineering", "blue"), o("design", "Design", "pink"), o("marketing", "Marketing", "orange"), o("support", "Support", "green"), o("sales", "Sales", "brown")}},
					{ID: "priority", Name: "Priority", Type: "select", Options: []opt{o("high", "High", "red"), o("medium", "Medium", "yellow"), o("low", "Low", "gray")}},
					{ID: "due", Name: "Due", Type: "date"},
					{ID: "hours", Name: "Hours", Type: "number"},
					{ID: "complete", Name: "Complete", Type: "formula", Expr: `if(prop("Status") == "Done", 1, 0)`},
				},
				views: []view{
					{Name: "Board", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},` + xaCards("owner", "priority", "due") + `,"layout":{"total":{"prop":"hours","fn":"sum"}}}`},
					{Name: "By owner", Type: "table", Config: `{"group":{"prop":"owner"},"sorts":[{"prop":"due","dir":"asc"}],"calcs":{"hours":"sum"},"props":[{"id":"complete","hidden":true}]}`},
					{Name: "Due dates", Type: "calendar", Config: `{"layout":{"date":"due"}}`},
					{Name: "Workload", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"owner"},"series":{"prop":"status"},"fn":"sum","prop":"hours","stacked":true,"legend":true}}}`},
				},
				rows: taskRows,
			},
			{
				key: "lt_ms", Name: "Milestones", Icon: "i:flag:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Milestone", Type: "title"},
					{ID: "when", Name: "Dates", Type: "date"},
					{ID: "status", Name: "Status", Type: "status", Options: xaStatus},
					{ID: "phase", Name: "Phase", Type: "select", Options: []opt{o("build", "Build", "blue"), o("beta", "Beta", "purple"), o("prep", "Launch prep", "orange"), o("launch", "Launch", "red"), o("after", "After launch", "green")}},
					{ID: "owner", Name: "Owner", Type: "select", Options: people},
					{ID: "progress", Name: "Progress", Type: "formula", Expr: `round(prop("Share done") * 100) / 100`, Display: "ring", Max: 1, Format: "percent"},
					{ID: "share", Name: "Share done", Type: "rollup", Relation: "tasks", Target: "complete", Fn: "average"},
					{ID: "count", Name: "Tasks", Type: "rollup", Relation: "tasks", Target: "title", Fn: "count_all"},
					{ID: "effort", Name: "Hours", Type: "rollup", Relation: "tasks", Target: "hours", Fn: "sum"},
					{ID: "tasks", Name: "Task list", Type: "relation", Collection: "@lt_tasks"},
					{ID: "goal", Name: "Goal", Type: "text"},
				},
				views: []view{
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"when","zoom":"month","table":false}}`},
					{Name: "By phase", Type: "board", Config: `{"group":{"prop":"phase","hide_empty":true},` + xaCards("progress", "owner", "when") + `}`},
					{Name: "All milestones", Type: "table", Config: `{"sorts":[{"prop":"when","dir":"asc"}],"calcs":{"effort":"sum"},"props":[{"id":"share","hidden":true},{"id":"when","hidden":false,"width":250}]}`},
				},
				rows: msRows,
			},
			{
				key: "lt_fb", Name: "Beta feedback", Icon: "i:chat:cyan", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Feedback", Type: "title"},
					{ID: "sentiment", Name: "Sentiment", Type: "select", Options: []opt{o("pos", "Positive", "green"), o("neu", "Neutral", "yellow"), o("neg", "Negative", "red")}},
					{ID: "score", Name: "Score", Type: "number", Display: "bar", Max: 10},
					{ID: "area", Name: "Area", Type: "select", Options: []opt{o("onboarding", "Onboarding", "purple"), o("inbox", "Inbox", "blue"), o("rules", "Rules", "orange"), o("reports", "Reports", "brown"), o("mobile", "Mobile app", "pink"), o("integrations", "Integrations", "green"), o("pricing", "Pricing", "yellow"), o("performance", "Performance", "red")}},
					{ID: "tester", Name: "Tester", Type: "text"},
					{ID: "company", Name: "Company", Type: "text"},
					{ID: "source", Name: "Source", Type: "select", Options: []opt{o("interview", "Interview", "purple"), o("widget", "In-app widget", "blue"), o("survey", "Survey", "green"), o("email", "Email", "gray")}},
					{ID: "received", Name: "Received", Type: "date"},
					{ID: "milestone", Name: "Milestone", Type: "relation", Collection: "@lt_ms"},
				},
				views: []view{
					{Name: "Overview", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"fb1","title":"Average score","size":"s","chart":{"kind":"number","fn":"average","prop":"score"}},{"id":"fb2","title":"Sentiment","size":"s","chart":{"kind":"donut","x":{"prop":"sentiment"},"fn":"count","legend":true}},{"id":"fb3","title":"Feedback per week","size":"m","chart":{"kind":"line","x":{"prop":"received","by":"week"},"fn":"count","smooth":true}},{"id":"fb4","title":"Sentiment by area","size":"l","chart":{"kind":"bar","x":{"prop":"area"},"series":{"prop":"sentiment"},"fn":"count","stacked":true,"legend":true}},{"id":"fb5","title":"Every score, by area","size":"l","chart":{"kind":"dots","x":{"prop":"area"},"prop":"score"}}]}}`},
					{Name: "By sentiment", Type: "board", Config: `{"group":{"prop":"sentiment","hide_empty":true},` + xaCards("area", "company", "score") + `,"layout":{"total":{"prop":"score","fn":"average"}}}`},
					{Name: "All feedback", Type: "table", Config: `{"sorts":[{"prop":"received","dir":"desc"}],"calcs":{"score":"average"}}`},
					{Name: "Share feedback", Type: "form", Config: `{"form":{"title":"Tell us about Beacon 2.0","description":"Two minutes of your time shapes what we fix before launch. Thank you for testing.","questions":[{"prop":"title","label":"In one sentence, what stood out?","required":true},{"prop":"area","label":"Which part of the product is this about?","required":true},{"prop":"sentiment","label":"How do you feel about it?"},{"prop":"score","label":"How likely are you to recommend Beacon, from 1 to 10?"},{"prop":"tester","label":"Your name"},{"prop":"company","label":"Your company"}],"submit":"Send feedback","done":"Thanks! The launch team reads every response."}}`},
				},
				rows: fbRows,
			},
			{
				key: "lt_metrics", Name: "Launch metrics", Icon: "i:chart-line:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Day", Type: "title"},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "signups", Name: "Sign-ups", Type: "number"},
					{ID: "trials", Name: "Trials started", Type: "number"},
					{ID: "paid", Name: "Paid conversions", Type: "number"},
					{ID: "active", Name: "Active teams", Type: "number"},
					{ID: "mrr", Name: "New MRR", Type: "number", Format: "dollar"},
					{ID: "uptime", Name: "Uptime", Type: "number", Precision: 2},
					{ID: "conv", Name: "Trial to paid", Type: "formula", Expr: `if(prop("Trials started") > 0, prop("Paid conversions") / prop("Trials started"), 0)`, Format: "percent", Precision: 1},
				},
				views: []view{
					{Name: "Dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"m1","title":"Sign-ups so far","size":"s","kpi":{"date":"date","by":"day"},"chart":{"kind":"number","fn":"sum","prop":"signups"}},{"id":"m2","title":"Paid teams vs goal of 150","size":"s","chart":{"kind":"gauge","fn":"sum","prop":"paid","target":150}},{"id":"m3","title":"Trials started","size":"s","kpi":{"date":"date","by":"day"},"chart":{"kind":"number","fn":"sum","prop":"trials"}},{"id":"m3b","title":"Active teams today","size":"s","kpi":{"date":"date","by":"day"},"chart":{"kind":"number","fn":"max","prop":"active"}},{"id":"m4","title":"Cumulative sign-ups","size":"l","chart":{"kind":"area","x":{"prop":"date","by":"day"},"fn":"sum","prop":"signups","cumulative":true,"smooth":true}},{"id":"m5","title":"Active teams","size":"m","chart":{"kind":"line","x":{"prop":"date","by":"week"},"fn":"max","prop":"active","smooth":true,"labels":true}},{"id":"m6","title":"Sign-ups per week","size":"m","chart":{"kind":"bar","x":{"prop":"date","by":"week"},"fn":"sum","prop":"signups","labels":true}},{"id":"m6b","title":"New MRR per week","size":"l","chart":{"kind":"bar","x":{"prop":"date","by":"week"},"fn":"sum","prop":"mrr"}},{"id":"m7","title":"Daily sign-ups","size":"l","chart":{"kind":"calendar","x":{"prop":"date"},"fn":"sum","prop":"signups"}}]}}`},
					{Name: "Daily numbers", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"signups":"sum","trials":"sum","paid":"sum","mrr":"sum","uptime":"average"}}`},
					{Name: "This fortnight vs last", Type: "chart", Config: `{"layout":{"chart":{"kind":"compare","x":{"prop":"date","by":"day"},"fn":"sum","prop":"signups","window":14,"smooth":true}}}`},
					{Name: "Sign-up heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":3,"fn":"sum","value":"signups","color":"green"}}}`},
					{Name: "Paid conversions", Type: "chart", Config: `{"layout":{"chart":{"kind":"area","x":{"prop":"date","by":"week"},"fn":"sum","prop":"paid","labels":true,"smooth":true}}}`},
				},
				rows: metricRows,
			},
			{
				key: "lt_brief", Name: "Launch brief", Icon: "i:page:orange", Kind: "doc",
				markdown: xaFill(`| Launch owner | Product | Launch | Status |
| --- | --- | --- | --- |
| Maya Chen | Beacon 2.0 | In about two and a half weeks | Public beta |

<!-- toc -->

## The one-line pitch

> [!IMPORTANT]
> Beacon 2.0 is a shared inbox that is fast enough for a busy front desk and simple enough that a five-person team sets it up in an afternoon.

## Why now

Small teams outgrew 1.x. In our last survey, 41% of churned customers named speed with large inboxes as the reason they left, and 28% asked for rules that assign conversations automatically. Version 2.0 fixes both and adds a mobile app that replies as fast as the desktop.

## Who it is for

- **Primary:** service businesses with 3 to 25 people answering customers by email and chat (clinics, studios, repair shops, agencies)
- **Secondary:** existing 1.x customers, who move to 2.0 at no extra cost for six months

## Goals and measures

| Goal | Measure | Target 30 days after launch | Today |
| --- | --- | --- | --- |
| Adoption | Teams signed up | 3,000 | [[signups]] |
| Revenue | Teams on a paid plan | 150 | [[paid]] |
| Revenue | New monthly recurring revenue | $18,000 | [[mrr]] |
| Quality | Average beta score | 8.0 or higher | [[score]] |
| Reliability | Uptime | 99.9% | See {{page:lt_metrics}} |

The success maths is simple: each paid team pays about $120 a month, so 150 teams is $18,000 in new monthly revenue.

## What is in 2.0

1. A rebuilt shared inbox that opens 20,000 conversations in under a second
2. Rules that assign, tag and reply automatically
3. A native mobile app with reliable push notifications
4. Reports for response time and customer satisfaction

## What is not in 2.0

- Phone calls and voicemail
- A public knowledge base (planned for 2.1)

## Risks

| Risk | Likelihood | Plan |
| --- | --- | --- |
| Android push delays are not fixed in time | Medium | Fallback polling while the app is open; launch day can move by one week |
| Launch-week support volume doubles | High | Twelve new help articles and two extra people on chat |
| Price change upsets 1.x customers | Low | Six months at the old price, announced two weeks ahead |

## Go or no-go checklist

- [x] Load test passed at five times expected traffic
- [x] Plan limits signed off
- [ ] Android push notification fix confirmed by beta teams
- [ ] Help center articles published
- [ ] Support team trained
- [ ] Press release approved

See the {{page:lt_gtm}} for channels and messaging.
`, nums...),
			},
			{
				key: "lt_gtm", Name: "Go-to-market plan", Icon: "i:megaphone:rose", Kind: "doc",
				markdown: `**Owner:** Tom Becker · **Reviewers:** Maya Chen, Lars Nilsson

## The path from stranger to paying team

Every channel feeds one path. We measure each step in {{page:lt_metrics}}.

` + fence + `mermaid
flowchart LR
  A[Blog, press and social] --> B[Launch page]
  P[Partner newsletters] --> B
  C[1.x customers] --> E
  B --> D[Waitlist and public beta]
  D --> E[14-day trial]
  E -->|activates a shared inbox| F[Paid plan]
  E -->|no activity in 7 days| G[Nudge email from Aisha]
  G --> E
  F --> H[Referral offer]
  H --> B
` + fence + `

## Audience and message

| Audience | What they care about | Message |
| --- | --- | --- |
| Front-desk teams | Never missing a customer | Every message answered, by the right person, fast |
| Owners and managers | Seeing how the team is doing | Response times and satisfaction on one page |
| 1.x customers | Not being disrupted | Everything you have, twice as fast, same price for six months |

## Channels

| Channel | Owner | Launch-week action | Budget |
| --- | --- | --- | --- |
| Blog and launch page | Tom Becker | Launch post and a two-minute video | $0 |
| Email | Tom Becker | Three emails to the waitlist and 1.x customers | $400 |
| Press | Tom Becker | Six embargoed briefings | $2,500 |
| Partners | Lars Nilsson | Feature in four partner newsletters | $1,200 |
| Paid social | Tom Becker | Two weeks of small-business targeting | $6,000 |
| In-product | Priya Raman | Upgrade banner for 1.x workspaces | $0 |

## Messaging details

<details>
<summary>Headline and proof points</summary>

- **Headline:** The shared inbox that keeps up with you
- **Proof 1:** Opens 20,000 conversations in under a second
- **Proof 2:** Rules assign and tag conversations for you
- **Proof 3:** Rated 8 out of 10 or higher by most beta teams

</details>

<details>
<summary>Answers to tough questions</summary>

- **Why are prices changing?** 2.0 costs more to run; 1.x customers keep their price for six months.
- **Is my data moved automatically?** Yes, in the background, with no downtime.
- **What about Android notifications?** Fixed before launch; see the launch task for status.

</details>

## Launch week at a glance

` + fence + `mermaid
timeline
  title Launch week
  Five days before : Embargoed press briefings
  Three days before : Help articles go live : Support team trained
  Launch day : 2.0 on for everyone : Blog post and launch email
  Days 1 to 7 : Paid social runs : Daily numbers check
  Day 7 : Results review
` + fence + `

The hour-by-hour plan for the day itself lives in {{page:lt_runofshow}}.
`,
				children: []Starter{
					{
						key: "lt_runofshow", Name: "Launch day run of show", Icon: "i:clock:amber", Kind: "doc",
						markdown: `> [!CAUTION]
> If anything customer-facing breaks, Diego Alvarez decides on a rollback within 15 minutes. Do not wait for a meeting.

## Timeline (all times Pacific)

| Time | What happens | Who |
| --- | --- | --- |
| 06:00 | Final health check of all services | Diego Alvarez |
| 06:30 | Turn on 2.0 for every workspace | Diego Alvarez |
| 07:00 | Press embargo lifts, blog post goes live | Tom Becker |
| 07:15 | Launch email to the waitlist | Tom Becker |
| 08:00 | War room opens, support doubles chat coverage | Aisha Okafor |
| 10:00 | First numbers check: sign-ups, errors, ticket volume | Maya Chen |
| 13:00 | Sales team demos for booked prospects | Lars Nilsson |
| 16:00 | Second numbers check and decision on paid social | Maya Chen |
| 17:30 | Wrap-up note to the whole company | Maya Chen |

## Before you go to bed the night before

- [ ] Feature flags tested in staging
- [ ] Rollback steps rehearsed
- [ ] Blog post scheduled and previewed
- [ ] Support macros for the ten most likely questions
- [ ] War room video link shared

## Numbers to watch

1. Sign-ups per hour (expect 40 to 80 in the morning)
2. Error rate on the inbox API (must stay under 0.5%)
3. Chat wait time (must stay under 3 minutes)
`,
					},
				},
			},
		},
	}
}

type xaCandidate struct {
	name, role, stage, source string
	day, score, ask           int
}

func xaStartup() Starter {
	people := []opt{o("nadia", "Nadia Brooks", "purple"), o("owen", "Owen Patel", "blue"), o("lucia", "Lucia Romero", "orange"), o("sam", "Sam Whitaker", "green"), o("rina", "Rina Takahashi", "brown"), o("jonah", "Jonah Ellis", "pink")}
	const months = 18
	type month struct {
		mrr, payroll, software, marketing, other, cash, funding, firms, burn int
	}
	var fin []month
	cash := 1250000
	for i := 0; i < months; i++ {
		m := month{}
		m.mrr = int(math.Round(21000*math.Pow(1.064, float64(i))*(0.99+0.02*xaNoise(21, i))/10) * 10)
		m.payroll = 52000 + 8500*min(max(i-2, 0), 10)
		if i >= months-2 {
			m.payroll += 14000
		}
		if i >= months-1 {
			m.payroll += 13000
		}
		m.software = 4000 + 300*i
		m.marketing = int(math.Round((6000+900*float64(i)+1500*xaNoise(22, i))/10) * 10)
		m.other = 7000
		if i >= 4 {
			m.other += 4000
		}
		m.burn = m.payroll + m.software + m.marketing + m.other - m.mrr
		if i == 3 {
			m.funding = 4500000
		}
		cash += m.funding - m.burn
		m.cash = cash
		m.firms = int(math.Round(float64(m.mrr) / 455))
		fin = append(fin, m)
	}
	last, prev := fin[months-1], fin[months-2]
	runway := float64(last.cash) / float64(last.burn)
	growth := (float64(last.mrr)/float64(prev.mrr) - 1) * 100
	var finRows []row
	for i, m := range fin {
		vals := map[string]any{"month": xaDay(xaMonthAgo(months - 1 - i)), "mrr": m.mrr, "payroll": m.payroll, "software": m.software, "marketing": m.marketing, "other": m.other, "cash": m.cash, "firms": m.firms}
		if m.funding > 0 {
			vals["funding"] = m.funding
		}
		finRows = append(finRows, row{Title: "Month " + strconv.Itoa(i+1), Values: vals})
	}
	finRows[3].Markdown = "> [!NOTE]\n> The seed round closed this month: $4.5M led by Harbor Light Ventures, with Quay Street Capital and six angels.\n"
	var table strings.Builder
	table.WriteString("| Month | MRR | Net burn | Cash at month end | Paying firms |\n| --- | --- | --- | --- | --- |\n")
	for i := months - 6; i < months; i++ {
		m := fin[i]
		fmt.Fprintf(&table, "| Month %d | %s | %s | %s | %d |\n", i+1, xaMoney(float64(m.mrr)), xaMoney(float64(m.burn)), xaMoney(float64(m.cash)), m.firms)
	}
	var chart strings.Builder
	chart.WriteString(fence + "mermaid\nxychart-beta\n  title \"Monthly recurring revenue, thousands of dollars\"\n  x-axis [")
	for i := months - 12; i < months; i++ {
		if i > months-12 {
			chart.WriteString(", ")
		}
		fmt.Fprintf(&chart, "M%d", i+1)
	}
	chart.WriteString("]\n  y-axis \"MRR\" 0 --> 80\n  bar [")
	for i := months - 12; i < months; i++ {
		if i > months-12 {
			chart.WriteString(", ")
		}
		fmt.Fprintf(&chart, "%.1f", float64(fin[i].mrr)/1000)
	}
	chart.WriteString("]\n" + fence)
	type kr struct {
		title, obj, owner, conf string
		start, target, current  float64
		unit                    string
	}
	krs := []kr{
		{"Grow MRR to $75,000", "o1", "lucia", "on", float64(prev.mrr), 75000, float64(last.mrr), "dollars"},
		{"Win 60 new paying firms", "o1", "lucia", "on", 0, 60, 34, "firms"},
		{"Net revenue retention of 110%", "o1", "sam", "risk", 100, 110, 106, "percent"},
		{"Median time to first invoice under 1 day", "o2", "jonah", "on", 4, 1, 2.2, "days"},
		{"Activation rate from 42% to 60%", "o2", "jonah", "risk", 42, 60, 51, "percent"},
		{"Onboarding tickets down to 20 a month", "o2", "sam", "on", 64, 20, 38, "tickets"},
		{"Hire five people", "o3", "nadia", "risk", 0, 5, 2, "hires"},
		{"80% of offers accepted", "o3", "nadia", "off", 0, 80, 67, "percent"},
		{"Every new hire ships in their first 30 days", "o3", "owen", "on", 0, 100, 100, "percent"},
		{"Monthly net burn under $120,000", "o4", "rina", "risk", 165000, 120000, float64(last.burn), "dollars"},
		{"Gross margin from 71% to 78%", "o4", "rina", "on", 71, 78, 75, "percent"},
		{"Commitments for a $1.5M extension", "o4", "nadia", "risk", 0, 1.5, 0.6, "million dollars"},
		{"Uptime of 99.95%", "o5", "owen", "on", 99.8, 99.95, 99.93, "percent"},
		{"Payment errors under 0.1%", "o5", "owen", "on", 0.4, 0.1, 0.15, "percent"},
		{"Invoice pages load in under 1.5 seconds", "o5", "owen", "risk", 2.8, 1.5, 1.9, "seconds"},
	}
	byObj := map[string][]string{}
	var krRows []row
	for _, k := range krs {
		byObj[k.obj] = append(byObj[k.obj], k.title)
		krRows = append(krRows, row{Title: k.title, Values: map[string]any{"objective": k.obj, "owner": k.owner, "confidence": k.conf, "start": k.start, "target": k.target, "current": k.current, "unit": k.unit}})
	}
	krRows[6].Markdown = "Two of five hires have started: a senior backend engineer and a customer success manager. See {{page:so_hire}} for the live pipeline.\n"
	objectives := []struct{ id, title, owner, why string }{
		{"o1", "Reach $75,000 in monthly recurring revenue", "lucia", "Revenue growth of 6% a month keeps us on plan for the Series A."},
		{"o2", "Make onboarding effortless for new firms", "jonah", "Firms that send an invoice on day one are three times as likely to pay."},
		{"o3", "Build a team that can double", "nadia", "Five key hires this quarter, without lowering the bar."},
		{"o4", "Keep two years of runway while we grow", "rina", "Control burn so the next raise happens on our timeline."},
		{"o5", "Earn trust with a reliable product", "owen", "Accountants move money with Fernway; it has to just work."},
	}
	var objRows []row
	for _, ob := range objectives {
		objRows = append(objRows, row{Title: ob.title, Values: map[string]any{"owner": ob.owner, "krs": byObj[ob.id], "why": ob.why, "quarter": "q4"}})
	}
	roles := []struct {
		id, title, team, manager string
		openings, low, high      int
	}{
		{"be", "Senior backend engineer", "eng", "owen", 2, 150000, 175000},
		{"design", "Product designer", "product", "jonah", 1, 125000, 145000},
		{"ae", "Account executive", "sales", "lucia", 1, 120000, 150000},
		{"csm", "Customer success manager", "cs", "sam", 1, 95000, 110000},
	}
	var roleRows []row
	for _, r := range roles {
		roleRows = append(roleRows, row{Title: r.title, Values: map[string]any{"team": r.team, "manager": r.manager, "openings": r.openings, "low": r.low, "high": r.high}})
	}
	roleTitle := map[string]string{}
	for _, r := range roles {
		roleTitle[r.id] = r.title
	}
	cands := []xaCandidate{
		{"Tariq Hassan", "be", "hired", "referral", -48, 5, 168000},
		{"Megan Doyle", "csm", "hired", "referral", -40, 5, 105000},
		{"Peter Novak", "ae", "declined", "referral", -35, 4, 145000},
		{"Kevin O'Brien", "be", "declined", "board", -30, 2, 150000},
		{"Ana Lima", "design", "offer", "referral", -28, 5, 140000},
		{"Oliver Grant", "csm", "declined", "board", -27, 2, 98000},
		{"Yuki Tanaka", "be", "declined", "agency", -26, 3, 190000},
		{"Gabriel Souza", "design", "declined", "board", -25, 2, 125000},
		{"Lena Vogel", "be", "interview", "inbound", -20, 4, 160000},
		{"Marcus Reid", "be", "interview", "board", -18, 4, 172000},
		{"Mila Petrova", "be", "interview", "referral", -16, 5, 170000},
		{"Jasper Holt", "design", "interview", "board", -15, 4, 132000},
		{"Hannah Reyes", "ae", "interview", "agency", -14, 4, 150000},
		{"Rachel Kim", "csm", "interview", "inbound", -13, 4, 102000},
		{"Arjun Mehta", "be", "screen", "board", -12, 3, 155000},
		{"Luis Ortega", "ae", "interview", "board", -11, 3, 138000},
		{"Noor Haddad", "design", "screen", "inbound", -10, 3, 128000},
		{"Chiara Conti", "be", "screen", "referral", -9, 4, 165000},
		{"Hugo Chen", "design", "screen", "agency", -8, 3, 135000},
		{"Dana Whitfield", "ae", "screen", "board", -7, 3, 130000},
		{"Theo Laurent", "csm", "screen", "board", -6, 3, 99000},
		{"Grace Mwangi", "ae", "screen", "referral", -5, 4, 142000},
		{"Sven Larsen", "be", "applied", "board", -5, 0, 158000},
		{"Ben Carter", "be", "applied", "board", -4, 0, 150000},
		{"Fatima Zahra", "be", "applied", "inbound", -3, 0, 162000},
		{"Amina Diallo", "csm", "applied", "referral", -3, 0, 100000},
		{"Elise Martin", "design", "applied", "board", -2, 0, 130000},
		{"Sophie Lambert", "ae", "applied", "inbound", -2, 0, 135000},
		{"Isaac Moreno", "ae", "applied", "board", -1, 0, 128000},
		{"Priscilla Wong", "design", "applied", "inbound", -1, 0, 138000},
	}
	var candRows []row
	for _, c := range cands {
		v := map[string]any{"role": roleTitle[c.role], "stage": c.stage, "source": c.source, "applied": xaDay(c.day), "ask": c.ask, "email": strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(c.name, " ", "."), "'", "")) + "@example.com"}
		if c.score > 0 {
			v["score"] = c.score
		}
		candRows = append(candRows, row{Title: c.name, Values: v})
	}
	candRows[4].Markdown = "## Offer\n\n- Base salary: $140,000\n- Equity: 0.25% over four years\n- Start date: in three weeks\n\n> [!TIP]\n> Ana asked for a portfolio review with the whole team before signing. Jonah is setting it up for Thursday.\n"
	nums := []string{"[[mrr]]", xaMoney(float64(last.mrr)), "[[prevmrr]]", xaMoney(float64(prev.mrr)), "[[growth]]", strconv.FormatFloat(math.Round(growth*10)/10, 'f', 1, 64), "[[burn]]", xaMoney(float64(last.burn)), "[[cash]]", xaMoney(float64(last.cash)), "[[runway]]", strconv.FormatFloat(math.Round(runway*10)/10, 'f', 1, 64), "[[firms]]", strconv.Itoa(last.firms), "[[arr]]", xaMoney(float64(last.mrr * 12)), "[[table]]", table.String(), "[[chart]]", chart.String(), "[[first]]", xaMoney(float64(fin[0].mrr))}
	return Starter{
		ID: "startup-os", Name: "Startup operating system", Description: "One home for a young company: goals and key results, hiring, monthly finances with runway, and the investor update.", Icon: "i:compass:violet", Kind: "doc", Category: catExamples, Tags: []string{"okrs", "hiring", "runway", "investors"},
		markdown: xaFill(`**Fernway** builds invoicing and payments software for small accounting firms. We are 18 months old, 14 people, and this page is how we run the company: what we are aiming for, who we are hiring, and how much money we have.

> [!NOTE]
> Monthly recurring revenue is **[[mrr]]** ([[growth]]% up on last month), we burn **[[burn]]** a month and have **[[runway]] months** of runway.

## Goals this quarter

Five objectives, each measured by key results. Progress rings fill up as owners update the current value of each key result.

{{db:so_obj}}

## Money

{{db:so_fin}}

## Hiring

{{db:so_hire}}

## Pages

- {{page:so_kr}}: every key result with its start, target and current value
- {{page:so_roles}}: open roles, salary bands and hiring managers
- {{page:so_update}}: this month's letter to investors
- {{page:so_cadence}}: how we plan, meet and decide, with {{page:so_board}} from the last board meeting
`, nums...),
		children: []Starter{
			{
				key: "so_kr", Name: "Key results", Icon: "i:target:rose", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Key result", Type: "title"},
					{ID: "progress", Name: "Progress", Type: "formula", Expr: `round(min(1, max(0, (prop("Current") - prop("Start")) / (prop("Target") - prop("Start")))) * 100) / 100`, Display: "ring", Max: 1, Format: "percent"},
					{ID: "pct", Name: "Percent done", Type: "formula", Expr: `round(prop("Progress") * 100)`},
					{ID: "objective", Name: "Objective", Type: "select", Options: []opt{o("o1", "Revenue", "green"), o("o2", "Onboarding", "blue"), o("o3", "Team", "purple"), o("o4", "Runway", "brown"), o("o5", "Reliability", "orange")}},
					{ID: "owner", Name: "Owner", Type: "select", Options: people},
					{ID: "confidence", Name: "Confidence", Type: "select", Options: []opt{o("on", "On track", "green"), o("risk", "At risk", "yellow"), o("off", "Off track", "red")}},
					{ID: "start", Name: "Start", Type: "number"},
					{ID: "target", Name: "Target", Type: "number"},
					{ID: "current", Name: "Current", Type: "number"},
					{ID: "unit", Name: "Unit", Type: "text"},
				},
				views: []view{
					{Name: "By objective", Type: "table", Config: `{"group":{"prop":"objective"},"calcs":{"progress":"average"},"props":[{"id":"pct","hidden":true}]}`},
					{Name: "Confidence", Type: "board", Config: `{"group":{"prop":"confidence","hide_empty":true},` + xaCards("progress", "owner", "objective") + `}`},
					{Name: "Percent done", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"objective"},"fn":"average","prop":"pct","labels":true,"target":70}}}`},
					{Name: "Key results per owner", Type: "chart", Config: `{"layout":{"chart":{"kind":"radial","x":{"prop":"owner"},"fn":"count","legend":true}}}`},
					{Name: "Who owns what", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"owner"},"series":{"prop":"confidence"},"fn":"count","stacked":true,"legend":true,"hide_empty":true}}}`},
				},
				rows: krRows,
			},
			{
				key: "so_obj", Name: "Objectives", Icon: "i:flag:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Objective", Type: "title"},
					{ID: "progress", Name: "Progress", Type: "formula", Expr: `round(prop("Average progress") * 100) / 100`, Display: "ring", Max: 1, Format: "percent"},
					{ID: "owner", Name: "Owner", Type: "select", Options: people},
					{ID: "count", Name: "Key results", Type: "rollup", Relation: "krs", Target: "title", Fn: "count_all"},
					{ID: "avg", Name: "Average progress", Type: "rollup", Relation: "krs", Target: "progress", Fn: "average"},
					{ID: "pct", Name: "Percent done", Type: "formula", Expr: `round(prop("Average progress") * 100)`},
					{ID: "krs", Name: "Key result list", Type: "relation", Collection: "@so_kr"},
					{ID: "quarter", Name: "Quarter", Type: "select", Options: []opt{o("q4", "This quarter", "blue"), o("next", "Next quarter", "gray")}},
					{ID: "why", Name: "Why it matters", Type: "text"},
				},
				views: []view{
					{Name: "Cards", Type: "gallery", Config: `{"layout":{"cover":"none","size":"medium"},` + xaCards("progress", "owner", "count") + `}`},
					{Name: "Table", Type: "table", Config: `{"calcs":{"progress":"average"},"props":[{"id":"avg","hidden":true},{"id":"pct","hidden":true}]}`},
					{Name: "How far along", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"pct","labels":true,"sort":"value_desc"}}}`},
				},
				rows: objRows,
			},
			{
				key: "so_roles", Name: "Open roles", Icon: "i:briefcase:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Role", Type: "title"},
					{ID: "team", Name: "Team", Type: "select", Options: []opt{o("eng", "Engineering", "blue"), o("product", "Product", "pink"), o("sales", "Sales", "orange"), o("cs", "Customer success", "green")}},
					{ID: "manager", Name: "Hiring manager", Type: "select", Options: people},
					{ID: "openings", Name: "Openings", Type: "number"},
					{ID: "low", Name: "Salary from", Type: "number", Format: "dollar"},
					{ID: "high", Name: "Salary to", Type: "number", Format: "dollar"},
				},
				views: []view{
					{Name: "Roles", Type: "table", Config: `{"calcs":{"openings":"sum"}}`},
					{Name: "Cards", Type: "gallery", Config: `{"layout":{"cover":"none","size":"small"},` + xaCards("team", "manager", "openings") + `}`},
				},
				rows: roleRows,
			},
			{
				key: "so_hire", Name: "Hiring pipeline", Icon: "i:users:cyan", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Candidate", Type: "title"},
					{ID: "role", Name: "Role", Type: "relation", Collection: "@so_roles"},
					{ID: "stage", Name: "Stage", Type: "status", Options: []opt{so("applied", "Applied", "gray", "todo"), so("screen", "Phone screen", "yellow", "doing"), so("interview", "Interviews", "blue", "doing"), so("offer", "Offer out", "purple", "doing"), so("hired", "Hired", "green", "done"), so("declined", "Not moving on", "red", "done")}},
					{ID: "score", Name: "Score", Type: "number", Display: "bar", Max: 5},
					{ID: "source", Name: "Source", Type: "select", Options: []opt{o("referral", "Referral", "green"), o("inbound", "Careers page", "blue"), o("board", "Job board", "gray"), o("agency", "Agency", "orange")}},
					{ID: "applied", Name: "Applied", Type: "date"},
					{ID: "ask", Name: "Salary ask", Type: "number", Format: "dollar"},
					{ID: "email", Name: "Email", Type: "email"},
				},
				views: []view{
					{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"stage","hide_empty":true},` + xaCards("role", "score", "source") + `,"layout":{"total":{"fn":"count"}}}`},
					{Name: "All candidates", Type: "table", Config: `{"sorts":[{"prop":"applied","dir":"desc"}],"group":{"prop":"role"}}`},
					{Name: "Funnel", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"stage"},"series":{"prop":"role"},"fn":"count","stacked":true,"legend":true}}}`},
					{Name: "Sources", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"source"},"fn":"count","legend":true,"labels":true}}}`},
				},
				rows: candRows,
			},
			{
				key: "so_fin", Name: "Monthly finance", Icon: "i:wallet:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Month", Type: "title"},
					{ID: "month", Name: "Month starting", Type: "date"},
					{ID: "mrr", Name: "MRR", Type: "number", Format: "dollar"},
					{ID: "payroll", Name: "Payroll", Type: "number", Format: "dollar"},
					{ID: "software", Name: "Software", Type: "number", Format: "dollar"},
					{ID: "marketing", Name: "Marketing", Type: "number", Format: "dollar"},
					{ID: "other", Name: "Rent and other", Type: "number", Format: "dollar"},
					{ID: "expenses", Name: "Total spend", Type: "formula", Expr: `prop("Payroll") + prop("Software") + prop("Marketing") + prop("Rent and other")`, Format: "dollar"},
					{ID: "burn", Name: "Net burn", Type: "formula", Expr: `prop("Total spend") - prop("MRR")`, Format: "dollar"},
					{ID: "cash", Name: "Cash at month end", Type: "number", Format: "dollar"},
					{ID: "runway", Name: "Runway in months", Type: "formula", Expr: `round(prop("Cash at month end") / prop("Net burn") * 10) / 10`},
					{ID: "funding", Name: "Money raised", Type: "number", Format: "dollar"},
					{ID: "firms", Name: "Paying firms", Type: "number"},
				},
				views: []view{
					{Name: "Dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"f1","title":"MRR this month","size":"s","chart":{"kind":"number","fn":"max","prop":"mrr"}},{"id":"f2","title":"Runway vs 24-month goal","size":"s","chart":{"kind":"gauge","fn":"average","prop":"runway","target":24,"filter":` + xaThisMonth + `}},{"id":"f3","title":"Net burn this month","size":"s","chart":{"kind":"number","fn":"sum","prop":"burn","filter":` + xaThisMonth + `}},{"id":"f4","title":"Paying firms","size":"s","kpi":{"date":"month","by":"month"},"chart":{"kind":"number","fn":"max","prop":"firms"}},{"id":"f5","title":"Monthly recurring revenue","size":"m","chart":{"kind":"line","x":{"prop":"month","by":"month"},"fn":"sum","prop":"mrr","smooth":true}},{"id":"f6","title":"Net burn","size":"m","chart":{"kind":"bar","x":{"prop":"month","by":"month"},"fn":"sum","prop":"burn"}},{"id":"f7","title":"Cash in the bank","size":"m","chart":{"kind":"area","x":{"prop":"month","by":"month"},"fn":"sum","prop":"cash"}},{"id":"f8","title":"Revenue earned to date","size":"m","chart":{"kind":"area","x":{"prop":"month","by":"month"},"fn":"sum","prop":"mrr","cumulative":true,"smooth":true}}]}}`},
					{Name: "Months", Type: "table", Config: `{"sorts":[{"prop":"month","dir":"desc"}],"calcs":{"mrr":"sum","burn":"sum","funding":"sum"}}`},
					{Name: "Spend over time", Type: "chart", Config: `{"layout":{"chart":{"kind":"area","x":{"prop":"month","by":"month"},"fn":"sum","prop":"expenses","smooth":true}}}`},
					{Name: "Runway", Type: "chart", Config: `{"layout":{"chart":{"kind":"line","x":{"prop":"month","by":"month"},"fn":"average","prop":"runway","labels":true}}}`},
				},
				rows: finRows,
			},
			{
				key: "so_update", Name: "Investor update", Icon: "i:mail:blue", Kind: "doc",
				markdown: xaFill(`**To:** Fernway investors and advisors · **From:** Nadia Brooks · **Sent:** {{date}}

Hi everyone,

Short version: revenue grew [[growth]]% last month to **[[mrr]]** in monthly recurring revenue ([[arr]] a year), [[firms]] accounting firms now pay us, and we have [[runway]] months of runway.

## Numbers

| Metric | This month | Last month |
| --- | --- | --- |
| Monthly recurring revenue | [[mrr]] | [[prevmrr]] |
| Net burn | [[burn]] |  |
| Cash in the bank | [[cash]] |  |
| Runway | [[runway]] months |  |
| Paying firms | [[firms]] |  |

### The last six months

[[table]]
We started 18 months ago at [[first]] a month. The chart below shows the last twelve months.

[[chart]]

## Wins

- **Activation is up.** Median time to a firm's first invoice fell from 4 days to 2.2 days after the new setup wizard.
- **Two great hires.** Tariq Hassan joined as a senior backend engineer and Megan Doyle as our first customer success manager.
- **Payments reliability.** Payment errors are down to 0.15%, close to our 0.1% goal.

## Challenges

- **Hiring is slow.** We have five roles to fill this quarter and two are done. One offer was declined over salary.
- **Burn is higher than planned** because the new hires started earlier than budgeted.

## Asks

1. Introductions to accounting firms with 10 to 50 staff in the Midwest
2. Senior product designers who love dense, data-heavy interfaces
3. Anyone who has run a usage-based pricing change for small businesses

> [!TIP]
> Reply to this email or book 20 minutes with me any time. The goals behind these numbers live in {{page:so_kr}}.

Thank you for backing us,
Nadia
`, nums...),
			},
			{
				key: "so_cadence", Name: "Operating cadence", Icon: "i:calendar:teal", Kind: "doc",
				markdown: `How Fernway plans, meets and decides. Fourteen people can stay in sync with very few meetings if every meeting has one job.

## The rhythm

| When | Meeting | Length | Output |
| --- | --- | --- | --- |
| Monday | Company kickoff | 20 min | Top three priorities for the week |
| Tuesday | Pipeline review | 30 min | Next step on every open deal |
| Wednesday | No-meeting day | All day | Deep work |
| Thursday | Product review | 45 min | Decisions on what ships next |
| Friday | Numbers and demos | 30 min | Key result updates in {{page:so_kr}} |
| Monthly | Investor update | Written | {{page:so_update}} |
| Quarterly | Board meeting | 2 hours | Approved plan and budget |

## How a quarter flows

` + fence + `mermaid
flowchart TD
  A[Draft objectives with the leads] --> B[Pick three to five key results each]
  B --> C[Budget check against runway]
  C -->|fits| D[Board approves the plan]
  C -->|too expensive| B
  D --> E[Weekly updates every Friday]
  E --> F{Off track for two weeks?}
  F -->|yes| G[Owner proposes a fix on Monday]
  F -->|no| E
  G --> E
  E --> H[Quarter review and scoring]
  H --> A
` + fence + `

## How we decide

> [!IMPORTANT]
> One person owns each decision. Others advise; the owner decides and writes it down.

<details>
<summary>Decisions the CEO owns</summary>

- Hiring plan and compensation bands
- Fundraising and anything that changes the cap table
- Pricing changes

</details>

<details>
<summary>Decisions team leads own</summary>

- What their team builds in a sprint
- Who they hire, within the approved plan
- Vendors under $2,000 a month

</details>

## Every Friday

- [ ] Update the current value of your key results
- [ ] Move candidates forward in the hiring pipeline
- [ ] Rina closes the books for the week
`,
				children: []Starter{
					{
						key: "so_board", Name: "Board meeting notes", Icon: "i:chat:violet", Kind: "doc",
						markdown: xaFill(`**Attendees:** Nadia Brooks, Rina Takahashi, Elena Marsh (Harbor Light Ventures), David Okoye (independent) · **Date:** {{date}}

## Agenda

1. Numbers and runway (Rina, 20 min)
2. Hiring plan (Nadia, 20 min)
3. Pricing experiment (Lucia, 30 min)
4. Closed session (30 min)

## Notes

- MRR is [[mrr]] and growing [[growth]]% a month; the board is comfortable with the plan
- Net burn of [[burn]] is above the $120,000 goal because two hires started early
- Runway of [[runway]] months gives room to raise in the second half of next year

## Decisions

| Decision | Owner | Why |
| --- | --- | --- |
| Approve hiring a fifth engineer next quarter | Nadia Brooks | Backlog for payments work keeps growing |
| Run the usage-based pricing test with 20 firms | Lucia Romero | Learn before changing prices for everyone |
| Keep marketing spend flat until activation reaches 60% | Rina Takahashi | Spend where conversion is proven |

## Action items

- [ ] Share the pricing test design with the board (Lucia Romero)
- [ ] Send updated runway scenarios at 5% and 8% monthly growth (Rina Takahashi)
- [ ] Introduce two design candidates from the portfolio network (Elena Marsh)
`, nums...),
					},
				},
			},
		},
	}
}

type xaCampaign struct {
	title, channel, status string
	from, to               int
	spend, revenue, budget int
	goal                   string
}

type xaKeyword struct {
	kw                      string
	volume, diff, pos, last int
	clicks                  int
	intent, topic           string
}

type xaPost struct {
	title                string
	day                  int
	kind, status, author string
	keyword, campaign    string
}

func xaMarketing() Starter {
	team := []opt{o("camille", "Camille Laurent", "purple"), o("jordan", "Jordan Price", "blue"), o("ruby", "Ruby Nakamura", "pink"), o("mateo", "Mateo Silva", "green"), o("hollis", "Hollis Grant", "orange")}
	channels := []struct {
		title, kind, owner string
		budget             int
		spend, roas, aov   float64
	}{
		{"Paid search", "paid", "jordan", 18000, 4200, 4.1, 92},
		{"Paid social", "paid", "jordan", 14000, 3300, 2.5, 78},
		{"Email", "owned", "hollis", 2500, 600, 13.5, 84},
		{"Organic search", "owned", "mateo", 4000, 900, 6.4, 88},
		{"Affiliates", "paid", "camille", 6000, 1400, 4.8, 96},
		{"Influencers", "earned", "ruby", 8000, 1800, 1.7, 74},
	}
	var chRows []row
	for _, c := range channels {
		chRows = append(chRows, row{Title: c.title, Values: map[string]any{"kind": c.kind, "owner": c.owner, "budget": c.budget}})
	}
	const weeks = 10
	var perfRows []row
	var revW, spendW [weeks]float64
	recentOrders, recentSessions := 0, 0
	for w := 0; w < weeks; w++ {
		for ci, c := range channels {
			lift := 1 + 0.035*float64(w)
			spend := math.Round(c.spend * lift * (0.85 + 0.3*xaNoise(31+ci, w)))
			rev := math.Round(spend * c.roas * (0.82 + 0.36*xaNoise(41+ci, w)))
			orders := int(math.Round(rev / c.aov))
			sessions := int(math.Round(float64(orders) / 0.0264 * (0.9 + 0.2*xaNoise(51+ci, w))))
			revW[w] += rev
			spendW[w] += spend
			if w >= weeks-4 {
				recentOrders += orders
				recentSessions += sessions
			}
			perfRows = append(perfRows, row{Title: fmt.Sprintf("Week %d · %s", w+1, c.title), Values: map[string]any{"week": xaDay(-7 * (weeks - 1 - w)), "channel": c.title, "spend": spend, "revenue": rev, "orders": orders, "sessions": sessions}})
		}
	}
	thisRev, lastRev, thisSpend := revW[weeks-1], revW[weeks-2], spendW[weeks-1]
	camps := []xaCampaign{
		{"Brand search always on", "Paid search", "live", -90, 60, 14100, 71800, 24000, "acquire"},
		{"Abandoned cart reminders", "Email", "live", -90, 60, 400, 22600, 800, "retain"},
		{"Spring trail sale", "Paid search", "ended", -70, -56, 9800, 41200, 10000, "acquire"},
		{"Ultralight tents launch", "Paid social", "ended", -66, -45, 12400, 30500, 12000, "acquire"},
		{"Welcome series refresh", "Email", "live", -60, 30, 1200, 18400, 1500, "retain"},
		{"Camp cooking guide", "Organic search", "live", -58, 30, 2600, 11800, 3000, "aware"},
		{"Affiliate summer push", "Affiliates", "ended", -55, -30, 5200, 24100, 5000, "acquire"},
		{"Trail creator collab", "Influencers", "ended", -50, -36, 7800, 9900, 8000, "aware"},
		{"Back to school day packs", "Paid social", "ended", -42, -28, 6900, 15800, 7000, "acquire"},
		{"Win-back for lapsed buyers", "Email", "ended", -40, -26, 600, 9400, 600, "retain"},
		{"Hiking boots comparison", "Organic search", "live", -35, 40, 1800, 8700, 2500, "aware"},
		{"Gear review partners", "Affiliates", "live", -28, 20, 3900, 17600, 6000, "acquire"},
		{"Shopping ads for stoves", "Paid search", "live", -24, 15, 6200, 25900, 9000, "acquire"},
		{"Fall layering lookbook", "Paid social", "live", -21, 10, 5400, 11900, 7000, "aware"},
		{"Weekend camping reels", "Influencers", "live", -14, 7, 4600, 4100, 6000, "aware"},
		{"Loyalty points launch", "Email", "live", -10, 20, 300, 5200, 500, "retain"},
		{"Early holiday gift guide", "Paid social", "live", -5, 25, 2100, 3600, 9000, "acquire"},
		{"Winter sleeping bags", "Paid search", "planned", 5, 35, 0, 0, 12000, "acquire"},
		{"Holiday gift guide", "Organic search", "planned", 8, 45, 0, 0, 3000, "aware"},
		{"Creator gift boxes", "Influencers", "planned", 12, 30, 0, 0, 9000, "aware"},
		{"Holiday affiliate bonus", "Affiliates", "planned", 20, 45, 0, 0, 6000, "acquire"},
		{"Black Friday early access", "Email", "planned", 28, 36, 0, 0, 800, "retain"},
		{"Black Friday doorbusters", "Paid social", "planned", 30, 38, 0, 0, 15000, "acquire"},
		{"Cyber week search boost", "Paid search", "planned", 33, 40, 0, 0, 10000, "acquire"},
	}
	var campRows []row
	for _, c := range camps {
		v := map[string]any{"channel": c.channel, "status": c.status, "dates": xaSpan(c.from, c.to), "budget": c.budget, "goal": c.goal}
		if c.spend > 0 {
			v["spend"], v["revenue"] = c.spend, c.revenue
		}
		campRows = append(campRows, row{Title: c.title, Values: v})
	}
	campRows[16].Markdown = "> [!TIP]\n> The full brief for the holiday push is in {{page:ma_brief}}.\n\n## Early read\n\n- Click-through is 1.9%, above the 1.4% account average\n- Return on spend is still low because most gift buyers wait for Black Friday\n- Keep the budget flat until the doorbuster creative is ready\n"
	kws := []xaKeyword{
		{"best ultralight tent", 18100, 62, 7, 11, 1240, "commercial", "tents"},
		{"2 person backpacking tent", 12100, 55, 4, 5, 1980, "commercial", "tents"},
		{"how to pitch a tent in the rain", 2900, 28, 3, 6, 610, "info", "tents"},
		{"tent footprint vs tarp", 1900, 22, 2, 2, 540, "info", "tents"},
		{"warmest sleeping bag", 9900, 58, 14, 19, 310, "commercial", "sleeping"},
		{"sleeping bag temperature ratings", 4400, 35, 5, 9, 720, "info", "sleeping"},
		{"down vs synthetic sleeping bag", 6600, 41, 8, 8, 590, "info", "sleeping"},
		{"inflatable sleeping pad", 8100, 49, 12, 15, 280, "buy", "sleeping"},
		{"camp stove", 27100, 71, 18, 22, 260, "buy", "cooking"},
		{"backpacking stove", 14800, 60, 9, 13, 690, "commercial", "cooking"},
		{"easy camping meals", 22200, 38, 6, 10, 2310, "info", "cooking"},
		{"how to clean a camp pot", 1300, 15, 1, 3, 470, "info", "cooking"},
		{"hiking boots for wide feet", 5400, 33, 4, 7, 960, "commercial", "hiking"},
		{"best day pack", 12100, 57, 11, 10, 520, "commercial", "hiking"},
		{"what to bring on a day hike", 8100, 26, 3, 4, 1650, "info", "hiking"},
		{"trekking poles worth it", 3600, 24, 5, 8, 480, "info", "hiking"},
		{"waterproof hiking jacket", 18100, 66, 21, 27, 140, "buy", "hiking"},
		{"camping gift ideas", 14800, 44, 9, 16, 880, "commercial", "gifts"},
		{"tidewater goods discount code", 1600, 5, 1, 1, 820, "buy", "brand"},
		{"tidewater goods reviews", 1300, 8, 2, 2, 390, "commercial", "brand"},
	}
	var kwRows []row
	for _, k := range kws {
		kwRows = append(kwRows, row{Title: k.kw, Values: map[string]any{"volume": k.volume, "difficulty": k.diff, "position": k.pos, "last": k.last, "clicks": k.clicks, "intent": k.intent, "topic": k.topic}})
	}
	posts := []xaPost{
		{"How to pitch a tent in the rain", -28, "blog", "live", "ruby", "how to pitch a tent in the rain", "Ultralight tents launch"},
		{"Ultralight tent buyer's guide", -25, "guide", "live", "mateo", "best ultralight tent", ""},
		{"Five easy camping meals under 20 minutes", -22, "blog", "live", "ruby", "easy camping meals", "Camp cooking guide"},
		{"October newsletter: layering for fall", -19, "newsletter", "live", "hollis", "", "Fall layering lookbook"},
		{"Reel: one-minute tent setup", -17, "social", "live", "ruby", "2 person backpacking tent", ""},
		{"Down vs synthetic, explained", -14, "video", "live", "ruby", "down vs synthetic sleeping bag", ""},
		{"What to bring on a day hike: a checklist", -12, "blog", "live", "mateo", "what to bring on a day hike", ""},
		{"Hiking boots for wide feet, tested", -9, "guide", "live", "mateo", "hiking boots for wide feet", "Hiking boots comparison"},
		{"Stove showdown: four backpacking stoves", -6, "video", "live", "ruby", "backpacking stove", "Shopping ads for stoves"},
		{"Loyalty points are here", -4, "newsletter", "live", "hollis", "", "Loyalty points launch"},
		{"Reel: packing a 30-liter day pack", -2, "social", "live", "ruby", "best day pack", ""},
		{"Sleeping bag temperature ratings decoded", 1, "blog", "scheduled", "mateo", "sleeping bag temperature ratings", "Winter sleeping bags"},
		{"November newsletter: first frost checklist", 3, "newsletter", "scheduled", "hollis", "", ""},
		{"Warmest sleeping bags for winter camping", 6, "guide", "review", "mateo", "warmest sleeping bag", "Winter sleeping bags"},
		{"Camping gift ideas for every budget", 9, "guide", "drafting", "ruby", "camping gift ideas", "Holiday gift guide"},
		{"Reel: stove-top pancakes", 11, "social", "drafting", "ruby", "easy camping meals", ""},
		{"Inflatable pads: what the R-value means", 14, "blog", "drafting", "mateo", "inflatable sleeping pad", ""},
		{"Trekking poles: worth it?", 17, "video", "idea", "ruby", "trekking poles worth it", ""},
		{"Gift guide newsletter", 21, "newsletter", "idea", "hollis", "camping gift ideas", "Holiday gift guide"},
		{"Waterproof jackets field test", 24, "video", "idea", "ruby", "waterproof hiking jacket", ""},
		{"Black Friday early access email", 28, "newsletter", "idea", "hollis", "", "Black Friday early access"},
		{"How to clean a camp pot without soap", 30, "blog", "idea", "mateo", "how to clean a camp pot", ""},
	}
	var postRows []row
	for _, p := range posts {
		v := map[string]any{"publish": xaDay(p.day), "kind": p.kind, "status": p.status, "author": p.author}
		if p.keyword != "" {
			v["keyword"] = p.keyword
		}
		if p.campaign != "" {
			v["campaign"] = p.campaign
		}
		postRows = append(postRows, row{Title: p.title, Values: v})
	}
	postRows[14].Markdown = "## Angle\n\nThree budgets, twelve gifts, every one tested on a real trip by the team.\n\n## Outline\n\n1. Under $50: stocking stuffers that get used\n2. $50 to $150: the upgrade they will not buy themselves\n3. Over $150: the big one\n\n- [x] Pick the twelve products with the buying team\n- [ ] Photos at the lake on Saturday\n- [ ] First draft to Camille by Friday\n"
	views := int(math.Round(float64(recentSessions) * 0.41))
	cart := int(math.Round(float64(views) * 0.19))
	checkout := int(math.Round(float64(cart) * 0.53))
	repeat := int(math.Round(float64(recentOrders) * 0.26))
	stages := []struct {
		id, title string
		count     int
		bench     float64
	}{
		{"s1", "Store visits", recentSessions, 0},
		{"s2", "Viewed a product", views, 0.38},
		{"s3", "Added to cart", cart, 0.17},
		{"s4", "Started checkout", checkout, 0.5},
		{"s5", "Bought", recentOrders, 0.6},
		{"s6", "Bought again within 60 days", repeat, 0.25},
	}
	var stageRows []row
	var funnelTable strings.Builder
	funnelTable.WriteString("| Step | People | Share of the step before |\n| --- | --- | --- |\n")
	for i, s := range stages {
		v := map[string]any{"step": s.id, "count": s.count}
		share := "·"
		if i > 0 {
			r := float64(s.count) / float64(stages[i-1].count)
			v["rate"] = math.Round(r*1000) / 1000
			v["bench"] = s.bench
			share = strconv.FormatFloat(math.Round(r*1000)/10, 'f', 1, 64) + "%"
		}
		stageRows = append(stageRows, row{Title: s.title, Values: v})
		fmt.Fprintf(&funnelTable, "| %s | %s | %s |\n", s.title, xaInt(s.count), share)
	}
	stageOpts := make([]opt, len(stages))
	colors := []string{"blue", "purple", "pink", "orange", "green", "brown"}
	for i, s := range stages {
		stageOpts[i] = o(s.id, s.title, colors[i])
	}
	chSpend := make([]float64, len(channels))
	chRev := make([]float64, len(channels))
	for i, r := range perfRows {
		chSpend[i%len(channels)] += r.Values["spend"].(float64)
		chRev[i%len(channels)] += r.Values["revenue"].(float64)
	}
	var xy strings.Builder
	xy.WriteString(fence + "mermaid\nxychart-beta\n  title \"Ten weeks: spend (bars) against revenue (line), thousands of dollars\"\n  x-axis [")
	var sp, rv []string
	for i, c := range channels {
		if i > 0 {
			xy.WriteString(", ")
		}
		xy.WriteString("\"" + c.title + "\"")
		sp = append(sp, strconv.FormatFloat(math.Round(chSpend[i]/100)/10, 'f', 1, 64))
		rv = append(rv, strconv.FormatFloat(math.Round(chRev[i]/100)/10, 'f', 1, 64))
	}
	xy.WriteString("]\n  y-axis \"Thousands of dollars\"\n  bar [" + strings.Join(sp, ", ") + "]\n  line [" + strings.Join(rv, ", ") + "]\n" + fence)
	change := (thisRev/lastRev - 1) * 100
	trend := "up " + strconv.FormatFloat(math.Round(change*10)/10, 'f', 1, 64) + "%"
	if change < 0 {
		trend = "down " + strconv.FormatFloat(math.Round(-change*10)/10, 'f', 1, 64) + "%"
	}
	nums := []string{"[[rev]]", xaMoney(thisRev), "[[lastrev]]", xaMoney(lastRev), "[[spend]]", xaMoney(thisSpend), "[[roas]]", strconv.FormatFloat(math.Round(thisRev/thisSpend*10)/10, 'f', 1, 64), "[[change]]", strconv.FormatFloat(math.Round(change*10)/10, 'f', 1, 64), "[[orders]]", xaInt(recentOrders), "[[sessions]]", xaInt(recentSessions), "[[conv]]", strconv.FormatFloat(math.Round(float64(recentOrders)/float64(recentSessions)*1000)/10, 'f', 1, 64), "[[funnel]]", funnelTable.String(), "[[xy]]", xy.String(), "[[trend]]", trend}
	cardProps := xaCards("channel", "dates", "roi")
	return Starter{
		ID: "marketing-analytics", Name: "Marketing analytics", Description: "See which channels and campaigns pay off, with spend against revenue, a content calendar, SEO keywords and a funnel.", Icon: "i:chart-pie:pink", Kind: "doc", Category: catExamples, Tags: []string{"campaigns", "roi", "seo", "funnel"},
		markdown: xaFill(`**Tidewater Goods** sells camping and hiking gear online. The five-person marketing team runs paid ads, email, content and partnerships, and this page answers one question every Monday: *what is working?*

> [!NOTE]
> Last week brought in **[[rev]]** from **[[spend]]** of marketing spend, a return of **[[roas]]x**. Revenue is [[trend]] on the week before.

## Weekly performance

Every channel, every week, with this period compared to the one before.

{{db:ma_perf}}

## The funnel, last four weeks

[[conv]]% of store visits ended in an order.

{{db:ma_funnel}}

## What we publish

{{db:ma_content}}

## Everything else

- {{page:ma_camp}}: every campaign with spend, revenue and return on investment
- {{page:ma_ch}}: the six channels, their owners and monthly budgets
- {{page:ma_kw}}: the search terms we want to rank for
- {{page:ma_plan}}: the quarter's plan, with the {{page:ma_brief}} for the holiday push
- {{page:ma_report}}: Monday's numbers in words
`, nums...),
		children: []Starter{
			{
				key: "ma_ch", Name: "Channels", Icon: "i:layers:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Channel", Type: "title"},
					{ID: "kind", Name: "Type", Type: "select", Options: []opt{o("paid", "Paid", "orange"), o("owned", "Owned", "blue"), o("earned", "Earned", "green")}},
					{ID: "owner", Name: "Owner", Type: "select", Options: team},
					{ID: "budget", Name: "Monthly budget", Type: "number", Format: "dollar"},
				},
				views: []view{
					{Name: "Channels", Type: "table", Config: `{"calcs":{"budget":"sum"}}`},
					{Name: "Budget split", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"title"},"fn":"sum","prop":"budget","legend":true,"labels":true}}}`},
				},
				rows: chRows,
			},
			{
				key: "ma_camp", Name: "Campaigns", Icon: "i:megaphone:orange", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Campaign", Type: "title"},
					{ID: "channel", Name: "Channel", Type: "relation", Collection: "@ma_ch"},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("planned", "Planned", "gray", "todo"), so("live", "Live", "blue", "doing"), so("ended", "Ended", "green", "done")}},
					{ID: "dates", Name: "Dates", Type: "date"},
					{ID: "goal", Name: "Goal", Type: "select", Options: []opt{o("aware", "Awareness", "purple"), o("acquire", "New customers", "blue"), o("retain", "Repeat buyers", "green")}},
					{ID: "budget", Name: "Budget", Type: "number", Format: "dollar"},
					{ID: "spend", Name: "Spend", Type: "number", Format: "dollar"},
					{ID: "revenue", Name: "Revenue", Type: "number", Format: "dollar"},
					{ID: "roi", Name: "ROI", Type: "formula", Expr: `if(prop("Spend") > 0, round((prop("Revenue") - prop("Spend")) / prop("Spend") * 100) / 100, 0)`, Format: "percent"},
					{ID: "roas", Name: "Return on spend", Type: "formula", Expr: `if(prop("Spend") > 0, round(prop("Revenue") / prop("Spend") * 10) / 10, 0)`},
					{ID: "used", Name: "Budget used", Type: "formula", Expr: `if(prop("Budget") > 0, min(1, round(prop("Spend") / prop("Budget") * 100) / 100), 0)`, Display: "bar", Max: 1, Format: "percent"},
				},
				views: []view{
					{Name: "All campaigns", Type: "table", Config: `{"sorts":[{"prop":"dates","dir":"desc"}],"calcs":{"spend":"sum","revenue":"sum","budget":"sum"}}`},
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"dates","zoom":"month","table":false}}`},
					{Name: "By status", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},` + cardProps + `,"layout":{"total":{"prop":"spend","fn":"sum"}}}`},
					{Name: "Revenue by channel", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"channel"},"fn":"sum","prop":"revenue","labels":true,"sort":"value_desc","hide_empty":true}}}`},
					{Name: "Return on spend", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"roas","labels":true,"sort":"value_desc","limit":12,"filter":{"prop":"spend","cond":"gt","value":0}}}}`},
				},
				rows: campRows,
			},
			{
				key: "ma_perf", Name: "Weekly performance", Icon: "i:chart-line:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Entry", Type: "title"},
					{ID: "week", Name: "Week", Type: "date"},
					{ID: "channel", Name: "Channel", Type: "relation", Collection: "@ma_ch"},
					{ID: "spend", Name: "Spend", Type: "number", Format: "dollar"},
					{ID: "revenue", Name: "Revenue", Type: "number", Format: "dollar"},
					{ID: "orders", Name: "Orders", Type: "number"},
					{ID: "sessions", Name: "Sessions", Type: "number"},
					{ID: "roas", Name: "Return on spend", Type: "formula", Expr: `round(prop("Revenue") / prop("Spend") * 10) / 10`},
					{ID: "cpa", Name: "Cost per order", Type: "formula", Expr: `round(prop("Spend") / prop("Orders") * 100) / 100`, Format: "dollar"},
				},
				views: []view{
					{Name: "Marketing dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"p1","title":"Revenue, ten weeks","size":"s","chart":{"kind":"number","fn":"sum","prop":"revenue"}},{"id":"p2","title":"Orders","size":"s","kpi":{"date":"week","by":"week"},"chart":{"kind":"number","fn":"sum","prop":"orders"}},{"id":"p3","title":"Sessions","size":"s","kpi":{"date":"week","by":"week"},"chart":{"kind":"number","fn":"sum","prop":"sessions"}},{"id":"p4","title":"Average return on spend","size":"s","chart":{"kind":"gauge","fn":"average","prop":"roas","target":4}},{"id":"p5","title":"Revenue: last five weeks vs the five before","size":"l","chart":{"kind":"compare","x":{"prop":"week","by":"week"},"fn":"sum","prop":"revenue","window":5,"smooth":true}},{"id":"p6","title":"Revenue by channel","size":"m","chart":{"kind":"line","x":{"prop":"week","by":"week"},"series":{"prop":"channel"},"fn":"sum","prop":"revenue","smooth":true}},{"id":"p7","title":"Spend by channel","size":"m","chart":{"kind":"area","x":{"prop":"week","by":"week"},"series":{"prop":"channel"},"fn":"sum","prop":"spend","stacked":true}},{"id":"p8","title":"Share of revenue","size":"m","chart":{"kind":"donut","x":{"prop":"channel"},"fn":"sum","prop":"revenue","legend":true}},{"id":"p9","title":"Return on spend by channel","size":"m","chart":{"kind":"hbar","x":{"prop":"channel"},"fn":"average","prop":"roas","labels":true,"sort":"value_desc"}},{"id":"p10","title":"Orders by channel and week","size":"l","chart":{"kind":"heatmap","x":{"prop":"week","by":"week"},"series":{"prop":"channel"},"fn":"sum","prop":"orders","labels":true}}]}}`},
					{Name: "Entries", Type: "table", Config: `{"sorts":[{"prop":"week","dir":"desc"}],"group":{"prop":"channel"},"calcs":{"spend":"sum","revenue":"sum","orders":"sum"}}`},
					{Name: "Spend, this period vs last", Type: "chart", Config: `{"layout":{"chart":{"kind":"compare","x":{"prop":"week","by":"week"},"fn":"sum","prop":"spend","window":5}}}`},
					{Name: "Cost per order", Type: "chart", Config: `{"layout":{"chart":{"kind":"dots","x":{"prop":"channel"},"prop":"cpa"}}}`},
				},
				rows: perfRows,
			},
			{
				key: "ma_kw", Name: "SEO keywords", Icon: "i:globe:teal", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Keyword", Type: "title"},
					{ID: "volume", Name: "Monthly searches", Type: "number", Format: "comma"},
					{ID: "difficulty", Name: "Difficulty", Type: "number", Display: "bar", Max: 100},
					{ID: "position", Name: "Position", Type: "number"},
					{ID: "last", Name: "Position last month", Type: "number"},
					{ID: "change", Name: "Places gained", Type: "formula", Expr: `prop("Position last month") - prop("Position")`},
					{ID: "clicks", Name: "Clicks", Type: "number", Format: "comma"},
					{ID: "intent", Name: "Intent", Type: "select", Options: []opt{o("info", "Learn", "blue"), o("commercial", "Compare", "purple"), o("buy", "Buy", "green")}},
					{ID: "topic", Name: "Topic", Type: "select", Options: []opt{o("tents", "Tents", "orange"), o("sleeping", "Sleeping", "purple"), o("cooking", "Cooking", "red"), o("hiking", "Hiking", "green"), o("gifts", "Gifts", "pink"), o("brand", "Brand", "gray")}},
				},
				views: []view{
					{Name: "Keywords", Type: "table", Config: `{"sorts":[{"prop":"volume","dir":"desc"}],"calcs":{"clicks":"sum","position":"average"}}`},
					{Name: "Search demand", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"volume","sort":"value_desc","limit":10,"labels":true}}}`},
					{Name: "Rankings by topic", Type: "chart", Config: `{"layout":{"chart":{"kind":"dots","x":{"prop":"topic"},"prop":"position"}}}`},
					{Name: "Clicks by intent", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"topic"},"series":{"prop":"intent"},"fn":"sum","prop":"clicks","stacked":true,"legend":true}}}`},
				},
				rows: kwRows,
			},
			{
				key: "ma_content", Name: "Content calendar", Icon: "i:calendar:pink", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Piece", Type: "title"},
					{ID: "publish", Name: "Publish date", Type: "date"},
					{ID: "kind", Name: "Format", Type: "select", Options: []opt{o("blog", "Blog post", "blue"), o("guide", "Buying guide", "purple"), o("video", "Video", "red"), o("social", "Social reel", "pink"), o("newsletter", "Newsletter", "green")}},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("idea", "Idea", "gray", "todo"), so("drafting", "Drafting", "yellow", "doing"), so("review", "In review", "orange", "doing"), so("scheduled", "Scheduled", "blue", "doing"), so("live", "Published", "green", "done")}},
					{ID: "author", Name: "Author", Type: "select", Options: team},
					{ID: "keyword", Name: "Target keyword", Type: "relation", Collection: "@ma_kw"},
					{ID: "campaign", Name: "Campaign", Type: "relation", Collection: "@ma_camp"},
				},
				views: []view{
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"publish"}}`},
					{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},` + xaCards("kind", "author", "publish") + `}`},
					{Name: "By format", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"kind"},"series":{"prop":"status"},"fn":"count","stacked":true,"legend":true}}}`},
					{Name: "All pieces", Type: "table", Config: `{"sorts":[{"prop":"publish","dir":"asc"}]}`},
				},
				rows: postRows,
			},
			{
				key: "ma_funnel", Name: "Funnel", Icon: "i:chart-bar:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Step", Type: "title"},
					{ID: "step", Name: "Stage", Type: "select", Options: stageOpts},
					{ID: "count", Name: "People", Type: "number", Format: "comma"},
					{ID: "rate", Name: "Share of step before", Type: "number", Format: "percent", Precision: 1},
					{ID: "bench", Name: "Industry benchmark", Type: "number", Format: "percent"},
					{ID: "gap", Name: "Against benchmark", Type: "formula", Expr: `if(prop("Industry benchmark") > 0, round((prop("Share of step before") - prop("Industry benchmark")) * 1000) / 10, 0)`},
				},
				views: []view{
					{Name: "Funnel", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"step"},"fn":"sum","prop":"count","labels":true}}}`},
					{Name: "Steps", Type: "table", Config: `{}`},
				},
				rows: stageRows,
			},
			{
				key: "ma_plan", Name: "Q4 marketing plan", Icon: "i:map:orange", Kind: "doc",
				markdown: `**Owner:** Camille Laurent · **Period:** this quarter · **Status:** Approved

<!-- toc -->

## Goals

| Goal | Measure | Target |
| --- | --- | --- |
| Grow revenue from marketing | Weekly revenue | $95,000 by Black Friday week |
| Spend efficiently | Return on spend across paid channels | 4x or better |
| Build owned audience | Email subscribers | 48,000 (from 39,500) |
| Win search | Keywords in the top 5 | 12 of 20 |

## How customers find us

` + fence + `mermaid
flowchart LR
  S[Search] --> P[Product page]
  SO[Social and creators] --> G[Guides and reels]
  G --> P
  AF[Affiliate reviews] --> P
  P --> C{Add to cart?}
  C -->|yes| K[Checkout]
  C -->|no| R[Retargeting ad]
  R --> P
  K --> B[First order]
  B --> E[Welcome emails]
  E --> L[Loyalty points]
  L --> B2[Second order]
` + fence + `

## Budget by channel

| Channel | Monthly budget | Why |
| --- | --- | --- |
| Paid search | $18,000 | Highest-intent traffic, steady 4x return |
| Paid social | $14,000 | Reach for new gift buyers in November |
| Influencers | $8,000 | Trust for new categories, slow to pay back |
| Affiliates | $6,000 | Pay only for sales |
| Organic search | $4,000 | Freelance writers and photos |
| Email | $2,500 | Tools and design |
| **Total** | **$52,500** | |

## Bets for the quarter

<details>
<summary>1. Buying guides beat product ads for new categories</summary>

Our guides convert at twice the rate of category pages. We will publish one guide a week and send paid social traffic to guides instead of products.

</details>

<details>
<summary>2. Loyalty points raise second orders</summary>

Repeat purchase within 60 days is 26%. The points program aims for 32% by the end of the quarter.

</details>

<details>
<summary>3. Influencers need a longer runway</summary>

Creator campaigns return less than 2x in their first month. We are testing gift boxes with longer usage periods before cutting the budget.

</details>

> [!WARNING]
> Paid social costs rise about 40% in late November. Front-load creative testing so only proven ads run during Black Friday week.

See {{page:ma_brief}} for the holiday campaign.
`,
				children: []Starter{
					{
						key: "ma_brief", Name: "Holiday campaign brief", Icon: "i:gift:rose", Kind: "doc",
						markdown: `> [!IMPORTANT]
> **One line:** Gear they will actually use, chosen by people who camp every weekend.

## Audience

- People buying for an outdoorsy friend or partner, mostly 28 to 45
- Existing customers buying for themselves during sales

## Offer

| Window | Offer | Channels |
| --- | --- | --- |
| Early access | 20% off for loyalty members | Email |
| Black Friday to Cyber Monday | Doorbusters on six products | Paid social, search, affiliates |
| Last shipping week | Free express shipping over $100 | All |

## Deliverables

- [x] Gift guide outline approved
- [ ] Twelve product photos at the lake
- [ ] Six reels from three creators
- [ ] Four emails written and tested
- [ ] Landing page with budget filters

## Success looks like

1. $310,000 in revenue across the four weeks
2. Return on spend of 4x or better on paid channels
3. 6,000 new email subscribers

Track live numbers in {{page:ma_perf}} and the campaigns in {{page:ma_camp}}.
`,
					},
				},
			},
			{
				key: "ma_report", Name: "Weekly marketing report", Icon: "i:note:green", Kind: "doc",
				markdown: xaFill(`**Week ending** {{date}} · **Prepared by** Jordan Price

## Headline numbers

| | Last week | Week before | Change |
| --- | --- | --- | --- |
| Revenue from marketing | [[rev]] | [[lastrev]] | [[change]]% |
| Marketing spend | [[spend]] | | |
| Return on spend | [[roas]]x | | |

## Spend against revenue by channel

[[xy]]

Email and organic search return the most per dollar; influencers return the least.

## Funnel, last four weeks

[[funnel]]
## What worked

- **Shopping ads for stoves** keep returning more than 4x; the video review lifted click-through
- **Abandoned cart emails** recovered more orders than any paid campaign for the cost

## What did not

- **Weekend camping reels** spent $4,600 for $4,100 in revenue. Pausing two of three creators.

## Next week

- [ ] Launch the winter sleeping bags search campaign
- [ ] Publish the temperature ratings post
- [ ] Brief creators for the gift boxes
`, nums...),
			},
		},
	}
}

type xaAccount struct {
	name, city     string
	lat, lng       float64
	industry, tier string
	owner          string
	employees      int
}

type xaDeal struct {
	account, label, stage  string
	amount, close          int
	owner, source, product string
}

func xaSales() Starter {
	reps := []opt{o("hannah", "Hannah Cole", "purple"), o("marcus", "Marcus Lee", "blue"), o("jamal", "Jamal Wright", "orange"), o("sofia", "Sofia Rossi", "green"), o("emma", "Emma Lindgren", "pink")}
	accounts := []xaAccount{
		{"Cascade Cold Storage", "Seattle", 47.6062, -122.3321, "food", "ent", "marcus", 1800},
		{"Golden Gate Bottling", "Oakland", 37.8044, -122.2712, "beverage", "mid", "marcus", 640},
		{"Sonoran Parts Supply", "Phoenix", 33.4484, -112.0740, "industrial", "mid", "marcus", 420},
		{"Front Range Outfitters", "Denver", 39.7392, -104.9903, "retail", "smb", "marcus", 210},
		{"Lone Star Freight Partners", "Dallas", 32.7767, -96.7970, "logistics", "ent", "jamal", 2600},
		{"Gulf Coast Marine Parts", "Houston", 29.7604, -95.3698, "industrial", "mid", "jamal", 700},
		{"Lakeshore Medical Supply", "Chicago", 41.8781, -87.6298, "medical", "ent", "jamal", 1500},
		{"Motor City Components", "Detroit", 42.3314, -83.0458, "industrial", "mid", "jamal", 900},
		{"Bluegrass Bourbon Supply", "Louisville", 38.2527, -85.7585, "beverage", "smb", "jamal", 150},
		{"Peachtree Home Goods", "Atlanta", 33.7490, -84.3880, "retail", "mid", "jamal", 560},
		{"Harborview Pharma Logistics", "Boston", 42.3601, -71.0589, "medical", "ent", "jamal", 1200},
		{"Empire Book Distributors", "New York", 40.7128, -74.0060, "retail", "mid", "jamal", 380},
		{"Maple Leaf Grocers Co-op", "Toronto", 43.6532, -79.3832, "food", "ent", "jamal", 3100},
		{"Thames Valley Wholesale", "London", 51.5074, -0.1278, "food", "ent", "sofia", 2200},
		{"Dutch Flower Logistics", "Rotterdam", 51.9244, 4.4777, "logistics", "ent", "sofia", 1400},
		{"Rhine Industrial Supply", "Cologne", 50.9375, 6.9603, "industrial", "mid", "sofia", 850},
		{"Nordic Cycle Parts", "Copenhagen", 55.6761, 12.5683, "retail", "smb", "sofia", 120},
		{"Alpine Sports Wholesale", "Zurich", 47.3769, 8.5417, "retail", "mid", "sofia", 330},
		{"Lyon Pharma Distribution", "Lyon", 45.7640, 4.8357, "medical", "mid", "sofia", 610},
		{"Iberia Fresh Foods", "Madrid", 40.4168, -3.7038, "food", "mid", "sofia", 990},
	}
	deals := []xaDeal{
		{"Cascade Cold Storage", "platform", "won", 186000, -150, "marcus", "outbound", "core"},
		{"Lone Star Freight Partners", "three sites", "won", 240000, -128, "jamal", "outbound", "core"},
		{"Thames Valley Wholesale", "rollout", "won", 210000, -110, "sofia", "partner", "core"},
		{"Motor City Components", "first site", "won", 92000, -95, "jamal", "inbound", "core"},
		{"Sonoran Parts Supply", "core platform", "won", 64000, -80, "marcus", "inbound", "core"},
		{"Peachtree Home Goods", "pilot", "lost", 54000, -70, "jamal", "outbound", "core"},
		{"Empire Book Distributors", "core platform", "won", 48000, -66, "jamal", "inbound", "core"},
		{"Nordic Cycle Parts", "starter plan", "won", 28000, -52, "sofia", "inbound", "core"},
		{"Rhine Industrial Supply", "core platform", "lost", 88000, -45, "sofia", "outbound", "core"},
		{"Cascade Cold Storage", "analytics add-on", "won", 38000, -40, "marcus", "expansion", "analytics"},
		{"Bluegrass Bourbon Supply", "starter plan", "lost", 22000, -30, "jamal", "inbound", "core"},
		{"Lone Star Freight Partners", "robotics integration", "won", 72000, -24, "jamal", "expansion", "robotics"},
		{"Front Range Outfitters", "starter plan", "lost", 26000, -18, "marcus", "inbound", "core"},
		{"Thames Valley Wholesale", "analytics add-on", "won", 41000, -12, "sofia", "expansion", "analytics"},
		{"Motor City Components", "second site", "won", 46000, -6, "jamal", "expansion", "core"},
		{"Sonoran Parts Supply", "analytics add-on", "negotiation", 24000, 8, "marcus", "expansion", "analytics"},
		{"Lakeshore Medical Supply", "four hospitals", "negotiation", 168000, 12, "jamal", "outbound", "core"},
		{"Dutch Flower Logistics", "rollout", "negotiation", 142000, 20, "sofia", "partner", "core"},
		{"Golden Gate Bottling", "core platform", "negotiation", 96000, 25, "marcus", "inbound", "core"},
		{"Empire Book Distributors", "robotics integration", "proposal", 36000, 22, "jamal", "expansion", "robotics"},
		{"Cascade Cold Storage", "second warehouse", "proposal", 94000, 28, "marcus", "expansion", "core"},
		{"Gulf Coast Marine Parts", "core platform", "proposal", 76000, 30, "jamal", "inbound", "core"},
		{"Harborview Pharma Logistics", "cold chain", "proposal", 210000, 35, "jamal", "outbound", "core"},
		{"Iberia Fresh Foods", "two sites", "proposal", 118000, 40, "sofia", "partner", "core"},
		{"Lone Star Freight Partners", "analytics add-on", "discovery", 58000, 45, "jamal", "expansion", "analytics"},
		{"Alpine Sports Wholesale", "core platform", "discovery", 52000, 48, "sofia", "inbound", "core"},
		{"Lyon Pharma Distribution", "core platform", "discovery", 84000, 55, "sofia", "inbound", "core"},
		{"Thames Valley Wholesale", "robotics integration", "discovery", 88000, 60, "sofia", "expansion", "robotics"},
		{"Maple Leaf Grocers Co-op", "network", "discovery", 320000, 70, "jamal", "outbound", "core"},
		{"Rhine Industrial Supply", "second look", "discovery", 70000, 80, "sofia", "outbound", "core"},
		{"Nordic Cycle Parts", "analytics add-on", "prospect", 12000, 65, "sofia", "expansion", "analytics"},
		{"Motor City Components", "robotics integration", "prospect", 64000, 85, "jamal", "expansion", "robotics"},
		{"Peachtree Home Goods", "fresh start", "prospect", 48000, 95, "jamal", "outbound", "core"},
		{"Gulf Coast Marine Parts", "second site", "prospect", 40000, 100, "jamal", "outbound", "core"},
		{"Golden Gate Bottling", "robotics integration", "prospect", 60000, 110, "marcus", "outbound", "robotics"},
		{"Dutch Flower Logistics", "analytics add-on", "prospect", 30000, 120, "sofia", "expansion", "analytics"},
	}
	prob := map[string]int{"prospect": 10, "discovery": 25, "proposal": 50, "negotiation": 75, "won": 100, "lost": 0}
	arr := map[string]int{}
	won, open, weighted := 0, 0, 0.0
	var dealRows []row
	nextStep := map[string]string{"prospect": "Book a discovery call", "discovery": "Site visit and requirements workshop", "proposal": "Walk the buying group through the proposal", "negotiation": "Agree final terms with procurement"}
	for _, d := range deals {
		title := d.account + " · " + d.label
		v := map[string]any{"account": d.account, "stage": d.stage, "amount": d.amount, "probability": prob[d.stage], "close": xaDay(d.close), "owner": d.owner, "source": d.source, "product": d.product}
		if s, ok := nextStep[d.stage]; ok {
			v["next"] = s
		}
		switch d.stage {
		case "won":
			arr[d.account] += d.amount
			won += d.amount
		case "lost":
		default:
			open += d.amount
			weighted += float64(d.amount*prob[d.stage]) / 100
		}
		dealRows = append(dealRows, row{Title: title, Values: v})
	}
	dealRows[16].Markdown = "The full plan for this deal is in {{page:sr_plan}}.\n\n## Where it stands\n\n- [x] Security review passed\n- [x] Pilot at the Evanston hospital hit 99.2% pick accuracy\n- [ ] Procurement signs the master agreement\n- [ ] Board approves the budget line\n"
	var accRows []row
	for _, a := range accounts {
		v := map[string]any{"hq": map[string]any{"lat": a.lat, "lng": a.lng, "name": a.city}, "industry": a.industry, "tier": a.tier, "owner": a.owner, "employees": a.employees, "status": "prospect", "website": "https://" + strings.ToLower(strings.ReplaceAll(strings.Fields(a.name)[0], "'", "")) + ".example.com"}
		if arr[a.name] > 0 {
			v["arr"] = arr[a.name]
			v["status"] = "customer"
		}
		accRows = append(accRows, row{Title: a.name, Values: v})
	}
	types := []string{"call", "demo", "meeting", "email"}
	verbs := map[string]string{"call": "Call with", "demo": "Demo for", "meeting": "On-site meeting at", "email": "Proposal follow-up to"}
	var actRows []row
	k := 0
	for _, d := range deals {
		if d.stage == "won" || d.stage == "lost" {
			continue
		}
		for j := 0; j < 2; j++ {
			if j == 1 && (d.stage == "prospect") {
				continue
			}
			t := types[(k+j)%len(types)]
			day := -2 - int(26*xaNoise(61, k*2+j))
			if j == 1 && (d.stage == "negotiation" || d.stage == "proposal") {
				day = 1 + int(8*xaNoise(62, k))
			}
			mins := map[string]int{"call": 30, "demo": 60, "meeting": 90, "email": 10}[t]
			actRows = append(actRows, row{Title: verbs[t] + " " + d.account + " (" + d.label + ")", Values: map[string]any{"kind": t, "date": xaDay(day), "deal": d.account + " · " + d.label, "owner": d.owner, "minutes": mins}})
		}
		k++
	}
	for i := 0; i < 8; i++ {
		a := accounts[(i*3)%len(accounts)]
		actRows = append(actRows, row{Title: "Prospecting email to " + a.name, Values: map[string]any{"kind": "email", "date": xaDay(-1 - int(20*xaNoise(63, i))), "owner": "emma", "minutes": 10}})
	}
	quota := 1400000
	nums := []string{"[[won]]", xaMoney(float64(won)), "[[open]]", xaMoney(float64(open)), "[[weighted]]", xaMoney(weighted), "[[quota]]", xaMoney(float64(quota)), "[[pct]]", strconv.Itoa(int(math.Round(float64(won) / float64(quota) * 100))), "[[accounts]]", strconv.Itoa(len(accounts)), "[[deals]]", strconv.Itoa(len(deals))}
	openFilter := `{"prop":"stage","cond":"group_is","value":["todo","doing"]}`
	wonFilter := `{"prop":"stage","cond":"is","value":["won"]}`
	return Starter{
		ID: "sales-revenue", Name: "Sales and revenue", Description: "A complete sales workspace: accounts on a map, a deal pipeline with weighted forecast, activities and an account plan.", Icon: "i:coin:amber", Kind: "doc", Category: catExamples, Tags: []string{"pipeline", "forecast", "crm", "accounts"},
		markdown: xaFill(`**Ridgeline Software** sells Stackwise, a warehouse management system, to distributors in North America and Europe. Five people sell it, and this page is where they work.

> [!NOTE]
> We have closed **[[won]]** this year against a quota of [[quota]] ([[pct]]%). The open pipeline is **[[open]]**, worth **[[weighted]]** once each deal is weighted by its chance of closing.

## Forecast

{{db:sr_deals}}

## Where our accounts are

[[accounts]] accounts across two continents. Click a pin to open the account.

{{db:sr_acc}}

## Working the pipeline

| Page | Use it to |
| --- | --- |
| {{page:sr_deals}} | Move deals across the board; column totals show the money in each stage |
| {{page:sr_acc}} | See every account by region, tier and industry |
| {{page:sr_act}} | Log calls, demos and meetings, and see who is busy when |
| {{page:sr_playbook}} | Learn the stages, exit criteria and questions to ask |
| {{page:sr_plan}} | Follow the plan for our biggest open deal |
`, nums...),
		children: []Starter{
			{
				key: "sr_acc", Name: "Accounts", Icon: "i:building:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Account", Type: "title"},
					{ID: "hq", Name: "Headquarters", Type: "place"},
					{ID: "status", Name: "Status", Type: "select", Options: []opt{o("customer", "Customer", "green"), o("prospect", "Prospect", "blue")}},
					{ID: "tier", Name: "Tier", Type: "select", Options: []opt{o("ent", "Enterprise", "purple"), o("mid", "Mid-market", "blue"), o("smb", "Small business", "gray")}},
					{ID: "industry", Name: "Industry", Type: "select", Options: []opt{o("food", "Food and grocery", "green"), o("beverage", "Beverage", "brown"), o("industrial", "Industrial parts", "orange"), o("retail", "Retail", "pink"), o("logistics", "Logistics", "blue"), o("medical", "Medical", "red")}},
					{ID: "owner", Name: "Account owner", Type: "select", Options: reps},
					{ID: "arr", Name: "Annual revenue", Type: "number", Format: "dollar"},
					{ID: "employees", Name: "Employees", Type: "number", Format: "comma"},
					{ID: "website", Name: "Website", Type: "url"},
				},
				views: []view{
					{Name: "Map", Type: "map", Config: `{"layout":{"place":"hq"}}`},
					{Name: "All accounts", Type: "table", Config: `{"sorts":[{"prop":"arr","dir":"desc"}],"calcs":{"arr":"sum"}}`},
					{Name: "By tier", Type: "board", Config: `{"group":{"prop":"tier","hide_empty":true},` + xaCards("status", "industry", "arr") + `}`},
					{Name: "Revenue by industry", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"industry"},"fn":"sum","prop":"arr","legend":true}}}`},
				},
				rows: accRows,
			},
			{
				key: "sr_deals", Name: "Deals", Icon: "i:briefcase:orange", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Deal", Type: "title"},
					{ID: "account", Name: "Account", Type: "relation", Collection: "@sr_acc"},
					{ID: "stage", Name: "Stage", Type: "status", Options: []opt{so("prospect", "Prospecting", "gray", "todo"), so("discovery", "Discovery", "blue", "doing"), so("proposal", "Proposal", "purple", "doing"), so("negotiation", "Negotiation", "orange", "doing"), so("won", "Closed won", "green", "done"), so("lost", "Closed lost", "red", "done")}},
					{ID: "amount", Name: "Amount (USD)", Type: "number", Format: "comma"},
					{ID: "probability", Name: "Probability", Type: "number", Display: "bar", Max: 100},
					{ID: "weighted", Name: "Weighted value (USD)", Type: "formula", Expr: `prop("Amount (USD)") * prop("Probability") / 100`, Format: "comma"},
					{ID: "close", Name: "Close date", Type: "date"},
					{ID: "owner", Name: "Owner", Type: "select", Options: reps},
					{ID: "source", Name: "Source", Type: "select", Options: []opt{o("inbound", "Inbound", "blue"), o("outbound", "Outbound", "purple"), o("partner", "Partner", "green"), o("expansion", "Expansion", "orange")}},
					{ID: "product", Name: "Product", Type: "select", Options: []opt{o("core", "Stackwise core", "blue"), o("analytics", "Analytics add-on", "green"), o("robotics", "Robotics integration", "pink")}},
					{ID: "next", Name: "Next step", Type: "text"},
				},
				views: []view{
					{Name: "Sales dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"d1","title":"Closed won vs quota","size":"s","chart":{"kind":"gauge","fn":"sum","prop":"amount","target":` + strconv.Itoa(quota) + `,"filter":` + wonFilter + `}},{"id":"d2","title":"Open pipeline","size":"s","chart":{"kind":"number","fn":"sum","prop":"amount","filter":` + openFilter + `}},{"id":"d3","title":"Weighted forecast","size":"s","chart":{"kind":"number","fn":"sum","prop":"weighted","filter":` + openFilter + `}},{"id":"d4","title":"Open deals","size":"s","chart":{"kind":"number","fn":"count","filter":` + openFilter + `}},{"id":"d5","title":"Forecast by close month","size":"l","chart":{"kind":"bar","x":{"prop":"close","by":"month"},"series":{"prop":"stage"},"fn":"sum","prop":"weighted","stacked":true,"legend":true,"filter":` + openFilter + `}},{"id":"d6","title":"Revenue closed to date","size":"m","chart":{"kind":"area","x":{"prop":"close","by":"month"},"fn":"sum","prop":"amount","cumulative":true,"smooth":true,"filter":` + wonFilter + `}},{"id":"d7","title":"Pipeline by owner","size":"m","chart":{"kind":"hbar","x":{"prop":"owner"},"series":{"prop":"stage"},"fn":"sum","prop":"amount","stacked":true,"hide_empty":true,"filter":` + openFilter + `}},{"id":"d8","title":"Where deals come from","size":"m","chart":{"kind":"donut","x":{"prop":"source"},"fn":"sum","prop":"amount","legend":true}},{"id":"d9","title":"Deal sizes by product","size":"m","chart":{"kind":"dots","x":{"prop":"product"},"prop":"amount"}}]}}`},
					{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"stage","hide_empty":true},` + xaCards("account", "amount", "close", "owner") + `,"layout":{"total":{"prop":"amount","fn":"sum"},"progress":"probability"}}`},
					{Name: "Close dates", Type: "calendar", Config: `{"layout":{"date":"close"},"filter":` + openFilter + `}`},
					{Name: "By owner", Type: "table", Config: `{"group":{"prop":"owner","hide_empty":true},"sorts":[{"prop":"close","dir":"asc"}],"calcs":{"amount":"sum","weighted":"sum"}}`},
					{Name: "Won by month", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"close","by":"month"},"series":{"prop":"product"},"fn":"sum","prop":"amount","stacked":true,"legend":true,"filter":` + wonFilter + `}}}`},
				},
				rows: dealRows,
			},
			{
				key: "sr_act", Name: "Activities", Icon: "i:phone:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Activity", Type: "title"},
					{ID: "kind", Name: "Type", Type: "select", Options: []opt{o("call", "Call", "blue"), o("demo", "Demo", "purple"), o("meeting", "Meeting", "orange"), o("email", "Email", "gray")}},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "deal", Name: "Deal", Type: "relation", Collection: "@sr_deals"},
					{ID: "owner", Name: "Owner", Type: "select", Options: reps},
					{ID: "minutes", Name: "Minutes", Type: "number"},
				},
				views: []view{
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"}}`},
					{Name: "Activity heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":2,"color":"teal"}}}`},
					{Name: "Who is busy", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"owner"},"series":{"prop":"kind"},"fn":"sum","prop":"minutes","stacked":true,"legend":true,"hide_empty":true}}}`},
					{Name: "Log", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}]}`},
				},
				rows: actRows,
			},
			{
				key: "sr_playbook", Name: "Sales playbook", Icon: "i:book:violet", Kind: "doc",
				markdown: `How a deal moves at Ridgeline, what must be true to move it forward, and how we forecast.

## Stages

` + fence + `mermaid
stateDiagram-v2
  [*] --> Prospecting
  Prospecting --> Discovery: pain confirmed
  Discovery --> Proposal: site visit done
  Proposal --> Negotiation: champion and budget confirmed
  Negotiation --> ClosedWon: contract signed
  Negotiation --> ClosedLost
  Proposal --> ClosedLost
  Discovery --> ClosedLost
  ClosedWon --> [*]
  ClosedLost --> [*]
` + fence + `

| Stage | Probability | Exit criteria |
| --- | --- | --- |
| Prospecting | 10% | A named person agrees to a discovery call |
| Discovery | 25% | We know the pain, the cost of the pain and who decides |
| Proposal | 50% | Proposal sent after a site visit; budget range confirmed |
| Negotiation | 75% | Verbal yes from the economic buyer; legal review started |
| Closed won | 100% | Signed order form |

## How we forecast

The weighted value of a deal is its amount times its probability:

$$
\text{Forecast} = \sum_{i} \text{Amount}_i \times p_i
$$

> [!TIP]
> Only move a deal forward when its exit criteria are met. A forecast built on hope is worse than no forecast.

## Discovery questions

<details>
<summary>About the problem</summary>

- What happens today when an order is picked wrong?
- How many people work in the warehouse per shift?
- What does a mis-ship cost you, roughly?

</details>

<details>
<summary>About the decision</summary>

- Who else needs to say yes?
- What is the deadline, and what happens if you miss it?
- How did you buy your last system?

</details>

## Weekly rhythm

- [ ] Monday: update stages and close dates before the {{page:sr_call}}
- [ ] Wednesday: log every customer touch in {{page:sr_act}}
- [ ] Friday: review deals with no activity in 14 days
`,
				children: []Starter{
					{
						key: "sr_call", Name: "Monday forecast call", Icon: "i:chat:orange", Kind: "doc",
						markdown: xaFill(`**Date:** {{date}} · **Attendees:** Hannah Cole, Marcus Lee, Jamal Wright, Sofia Rossi, Emma Lindgren

## Numbers

- Closed won this year: [[won]] of [[quota]] ([[pct]]%)
- Open pipeline: [[open]]; weighted forecast: [[weighted]]

## Deal reviews

| Deal | Owner | Change this week | Risk |
| --- | --- | --- | --- |
| Lakeshore Medical Supply · four hospitals | Jamal Wright | Moved to negotiation | Procurement timing |
| Dutch Flower Logistics · rollout | Sofia Rossi | Partner joined the call | Low |
| Golden Gate Bottling · core platform | Marcus Lee | Pricing approved | Competitor discounting |
| Harborview Pharma Logistics · cold chain | Jamal Wright | Proposal sent | New champion |

## Decisions

- Offer a 60-day pilot to Maple Leaf Grocers Co-op instead of a discount
- Emma books five new discovery calls in the Midwest this week

## Action items

- [ ] Send the redlined master agreement to Lakeshore (Jamal Wright)
- [ ] Reference call between Golden Gate and Cascade Cold Storage (Marcus Lee)
- [ ] Update close dates for every deal past due (everyone)
`, nums...),
					},
				},
			},
			{
				key: "sr_plan", Name: "Account plan: Lakeshore Medical Supply", Icon: "i:target:rose", Kind: "doc",
				markdown: `| Owner | Deal size | Stage | Target close |
| --- | --- | --- | --- |
| Jamal Wright | $168,000 a year | Negotiation | In about two weeks |

<!-- toc -->

## Why they buy

Lakeshore supplies four hospitals from one Chicago warehouse. Wrong picks cost them about $410,000 last year in rush deliveries and write-offs, and their current system ends support next spring.

## The buying group

| Person | Role | Stance | What they care about |
| --- | --- | --- | --- |
| Dana Ruiz | VP Operations (economic buyer) | Supportive | Cutting rush deliveries |
| Kofi Mensah | Warehouse manager (champion) | Strong supporter | Easier training for night shift |
| Laura Becker | IT director | Neutral | Security and integration effort |
| Sam Ortiz | Procurement | Cautious | Multi-year price protection |

` + fence + `mermaid
flowchart TD
  K[Kofi Mensah, champion] --> D[Dana Ruiz signs off on value]
  L[Laura Becker approves security] --> D
  D --> P[Sam Ortiz negotiates terms]
  P --> B[Board budget approval]
  B --> S[Signature]
` + fence + `

## Mutual action plan

- [x] Discovery workshop at the warehouse
- [x] Pilot at one hospital site (99.2% pick accuracy, up from 96.8%)
- [x] Security questionnaire returned
- [ ] Master agreement redlines agreed
- [ ] Budget approved at the board meeting
- [ ] Signature and kickoff date set

## Risks

> [!WARNING]
> Procurement wants a three-year price lock. We can offer two years with a 5% cap in year three.

| Risk | Plan |
| --- | --- |
| Board meeting moves | Ask Dana for a written approval by email instead |
| Competitor drops price | Lean on the pilot results and the night-shift training time |

See the deal in {{page:sr_deals}}.
`,
			},
		},
	}
}

type xaIssue struct {
	title, kind, service, status string
	points, sprint               int
}

type xaIncident struct {
	title, service, cause, sev string
	day, minutes               int
}

func xaEngineering() Starter {
	engineers := []opt{o("rosa", "Rosa Kim", "purple"), o("dev", "Dev Malhotra", "blue"), o("femi", "Femi Adebayo", "orange"), o("yui", "Yui Sato", "pink"), o("tomas", "Tomas Berg", "green"), o("lina", "Lina Haddad", "brown")}
	deps := []struct{ title, kind string }{
		{"Postgres primary", "data"}, {"Redis cache", "data"}, {"Kafka event bus", "queue"}, {"Object storage", "data"}, {"Search cluster", "data"},
		{"Card processor API", "vendor"}, {"Email provider", "vendor"}, {"SMS gateway", "vendor"}, {"CDN", "vendor"}, {"Feature flags", "vendor"},
	}
	var depRows []row
	for _, d := range deps {
		depRows = append(depRows, row{Title: d.title, Values: map[string]any{"kind": d.kind}})
	}
	services := []struct {
		title, team, tier, lang, oncall string
		rpm, p95                        int
		uptime                          float64
		uses                            []string
	}{
		{"Web app", "discovery", "t1", "ts", "femi", 4200, 180, 99.97, []string{"CDN", "Feature flags", "Redis cache"}},
		{"Checkout API", "checkout", "t1", "go", "rosa", 1800, 240, 99.93, []string{"Postgres primary", "Redis cache", "Kafka event bus", "Feature flags"}},
		{"Payments", "checkout", "t1", "go", "dev", 650, 410, 99.95, []string{"Postgres primary", "Card processor API", "Kafka event bus"}},
		{"Pricing engine", "checkout", "t2", "go", "rosa", 2600, 35, 99.98, []string{"Redis cache", "Postgres primary"}},
		{"Fraud check", "checkout", "t2", "python", "dev", 640, 160, 99.9, []string{"Redis cache", "Kafka event bus"}},
		{"Search", "discovery", "t2", "go", "yui", 3100, 95, 99.94, []string{"Search cluster", "Kafka event bus"}},
		{"Seat map", "discovery", "t1", "ts", "femi", 2400, 120, 99.96, []string{"Redis cache", "Postgres primary", "CDN"}},
		{"Notifications", "platform", "t3", "kotlin", "lina", 900, 300, 99.8, []string{"Kafka event bus", "Email provider", "SMS gateway"}},
		{"Ticket issuer", "platform", "t2", "kotlin", "tomas", 700, 220, 99.92, []string{"Kafka event bus", "Object storage", "Postgres primary"}},
		{"Auth", "platform", "t1", "go", "tomas", 5200, 40, 99.99, []string{"Postgres primary", "Redis cache"}},
		{"Reporting", "platform", "t3", "python", "lina", 120, 1900, 99.5, []string{"Postgres primary", "Object storage"}},
		{"Mobile gateway", "platform", "t1", "go", "tomas", 2900, 140, 99.95, []string{"Redis cache", "Feature flags"}},
	}
	var svcRows []row
	for _, s := range services {
		svcRows = append(svcRows, row{Title: s.title, Values: map[string]any{"team": s.team, "tier": s.tier, "lang": s.lang, "oncall": s.oncall, "rpm": s.rpm, "p95": s.p95, "uptime": s.uptime, "uses": s.uses}})
	}
	svcRows[1].Markdown = "Runbook: {{page:en_runbook}}. Last major incident: {{page:en_pm}}.\n"
	owner := map[string][]string{"Web app": {"femi", "yui"}, "Search": {"yui", "femi"}, "Seat map": {"femi", "yui"}, "Checkout API": {"rosa", "dev"}, "Payments": {"dev", "rosa"}, "Pricing engine": {"rosa", "dev"}, "Fraud check": {"dev", "rosa"}, "Notifications": {"lina", "tomas"}, "Ticket issuer": {"tomas", "lina"}, "Auth": {"tomas", "lina"}, "Reporting": {"lina", "tomas"}, "Mobile gateway": {"tomas", "lina"}, "": {"tomas", "lina"}}
	issues := []xaIssue{
		{"Move checkout sessions to Redis", "feature", "Checkout API", "done", 5, 0},
		{"Fix double charge on retry", "bug", "Payments", "done", 3, 0},
		{"Seat map renders slowly on old phones", "bug", "Seat map", "done", 5, 0},
		{"Upgrade the Postgres driver", "chore", "", "done", 2, 0},
		{"Add idempotency keys to refunds", "feature", "Payments", "done", 3, 0},
		{"Alert on queue lag", "chore", "", "done", 2, 0},
		{"Search by venue and date", "feature", "Search", "done", 8, 1},
		{"Cache event pages at the edge", "feature", "Web app", "done", 5, 1},
		{"Rate limit the public API", "feature", "Mobile gateway", "done", 3, 1},
		{"Retry failed SMS with backoff", "bug", "Notifications", "done", 2, 1},
		{"Remove the legacy coupon code", "debt", "Pricing engine", "done", 3, 1},
		{"Dashboard for checkout errors", "chore", "Reporting", "done", 2, 1},
		{"Dynamic pricing rules", "feature", "Pricing engine", "done", 8, 2},
		{"Handle fraud check timeouts", "bug", "Fraud check", "done", 3, 2},
		{"Ticket PDF uses the wrong time zone", "bug", "Ticket issuer", "done", 2, 2},
		{"Split the notifications worker", "debt", "Notifications", "done", 5, 2},
		{"Wallet passes for phones", "feature", "Ticket issuer", "done", 5, 2},
		{"Load test for on-sale spikes", "chore", "", "todo", 3, 2},
		{"Queue page for big on-sales", "feature", "Web app", "done", 8, 3},
		{"Fix Kafka consumer lag", "bug", "Ticket issuer", "done", 3, 3},
		{"Hold seats for ten minutes", "feature", "Seat map", "done", 5, 3},
		{"Search typo tolerance", "feature", "Search", "done", 3, 3},
		{"Refund status emails", "feature", "Notifications", "done", 2, 3},
		{"Drop unused indexes", "debt", "", "done", 2, 3},
		{"Checkout in three taps", "feature", "Checkout API", "done", 8, 4},
		{"Stop the 3-D Secure challenge loop", "bug", "Payments", "done", 5, 4},
		{"Price breakdown before checkout", "feature", "Pricing engine", "done", 3, 4},
		{"Rotate session tokens on refresh", "chore", "Auth", "done", 3, 4},
		{"Sales report export", "feature", "Reporting", "done", 2, 4},
		{"Move images to object storage", "debt", "Web app", "progress", 5, 4},
		{"Waitlist when an event sells out", "feature", "Web app", "done", 5, 5},
		{"Fix stale seat availability", "bug", "Seat map", "done", 5, 5},
		{"Payment provider failover", "feature", "Payments", "done", 8, 5},
		{"Organizer payout report", "feature", "Reporting", "done", 3, 5},
		{"Sign in with a magic link", "feature", "Auth", "done", 3, 5},
		{"Trim the mobile gateway payload", "debt", "Mobile gateway", "done", 2, 5},
		{"Group bookings up to 20 seats", "feature", "Checkout API", "progress", 8, 6},
		{"Fix push for ticket transfers", "bug", "Notifications", "review", 3, 6},
		{"Rank search by popularity", "feature", "Search", "progress", 5, 6},
		{"Postgres failover drill", "chore", "", "done", 3, 6},
		{"Promo codes per organizer", "feature", "Pricing engine", "todo", 5, 6},
		{"Reduce fraud false positives", "feature", "Fraud check", "review", 5, 6},
		{"Event page accessibility fixes", "bug", "Web app", "done", 3, 6},
		{"Resale marketplace spike", "feature", "Web app", "todo", 5, 7},
		{"Seat map zoom on tablets", "feature", "Seat map", "todo", 3, 7},
		{"Archive old orders", "debt", "", "todo", 3, 7},
		{"Organizer team roles", "feature", "Auth", "todo", 5, 7},
		{"Price drop alerts", "feature", "Notifications", "todo", 3, 7},
		{"Dark mode for the organizer app", "feature", "Web app", "backlog", 5, -1},
		{"Offline tickets in the app", "feature", "Mobile gateway", "backlog", 8, -1},
		{"Multi-currency checkout", "feature", "Checkout API", "backlog", 8, -1},
		{"Replace cron with scheduled jobs", "debt", "", "backlog", 3, -1},
		{"Partner API for resellers", "feature", "Mobile gateway", "backlog", 8, -1},
		{"Delete stale feature flags", "chore", "", "backlog", 1, -1},
	}
	bySprint := map[int][]string{}
	planned := map[int]int{}
	completed := map[int]int{}
	var issueRows []row
	for i, is := range issues {
		s := is.sprint
		if s < 0 {
			s = 6
		}
		v := map[string]any{"type": is.kind, "status": is.status, "points": is.points, "assignee": owner[is.service][i%2], "created": xaDay(-91 + 14*s - i%9)}
		if is.service != "" {
			v["service"] = is.service
		}
		if is.sprint >= 0 {
			bySprint[is.sprint] = append(bySprint[is.sprint], is.title)
			planned[is.sprint] += is.points
			if is.status == "done" {
				completed[is.sprint] += is.points
			}
		}
		issueRows = append(issueRows, row{Title: is.title, Values: v})
	}
	issueRows[36].Markdown = "## Acceptance criteria\n\n- [x] Buyers can pick up to 20 seats in one order\n- [x] Seats stay together when possible\n- [ ] One payment, one confirmation email, tickets split by name\n- [ ] Load test with 500 group checkouts a minute\n"
	sprintGoals := []string{"Faster, safer checkout sessions", "Find events by venue and date", "Smarter prices and wallet passes", "Survive big on-sales", "Checkout in three taps", "Never sell a seat twice", "Group bookings", "Resale groundwork"}
	var sprintRows []row
	velSum, velN := 0, 0
	for s := 0; s < 8; s++ {
		state := "done"
		if s == 6 {
			state = "current"
		} else if s == 7 {
			state = "next"
		}
		if state == "done" {
			velSum += completed[s]
			velN++
		}
		start := -88 + 14*s
		sprintRows = append(sprintRows, row{Title: "Sprint " + strconv.Itoa(21+s), Values: map[string]any{"dates": xaSpan(start, start+13), "state": state, "goal": sprintGoals[s], "issues": bySprint[s]}})
	}
	sprintRows[6].Markdown = "## Goal\n\nGroup bookings of up to 20 seats, end to end.\n\n## Risks\n\n> [!WARNING]\n> Rosa is on call during the Lakeside Festival follow-up work. Keep scope tight.\n"
	incidents := []xaIncident{
		{"Card payments failing for one bank", "Payments", "dependency", "sev2", -352, 64},
		{"Search returns empty results", "Search", "deploy", "sev2", -338, 38},
		{"Push notifications delayed", "Notifications", "dependency", "sev3", -322, 120},
		{"Login loop on mobile", "Auth", "bug", "sev2", -301, 52},
		{"Seat map not loading in one browser", "Seat map", "deploy", "sev3", -289, 45},
		{"Reports a day behind", "Reporting", "capacity", "sev3", -270, 300},
		{"Checkout slow during an on-sale", "Checkout API", "capacity", "sev2", -255, 33},
		{"Ticket emails sent twice", "Notifications", "bug", "sev3", -241, 90},
		{"Pricing shows old fees", "Pricing engine", "config", "sev2", -230, 41},
		{"Database failover took too long", "Checkout API", "dependency", "sev1", -214, 26},
		{"Wallet passes fail to install", "Ticket issuer", "bug", "sev3", -199, 180},
		{"Mobile gateway returning errors", "Mobile gateway", "deploy", "sev2", -187, 22},
		{"Fraud check blocking real buyers", "Fraud check", "config", "sev2", -171, 75},
		{"Image uploads failing for organizers", "Web app", "dependency", "sev3", -160, 95},
		{"Search index out of date", "Search", "capacity", "sev3", -148, 140},
		{"3-D Secure challenge loops", "Payments", "bug", "sev1", -133, 58},
		{"Auth tokens expiring early", "Auth", "config", "sev3", -120, 35},
		{"Seat holds released too soon", "Seat map", "bug", "sev2", -108, 40},
		{"SMS codes not arriving in Canada", "Notifications", "dependency", "sev3", -97, 210},
		{"Event pages missing prices", "Pricing engine", "deploy", "sev2", -86, 18},
		{"Checkout errors after a config push", "Checkout API", "config", "sev2", -71, 24},
		{"Queue page let everyone in at once", "Web app", "bug", "sev2", -63, 31},
		{"Payouts report times out", "Reporting", "capacity", "sev3", -55, 160},
		{"Kafka lag delays ticket issuing", "Ticket issuer", "capacity", "sev2", -44, 67},
		{"Search slow for large venues", "Search", "capacity", "sev3", -36, 85},
		{"Duplicate charges on retry", "Payments", "bug", "sev2", -28, 44},
		{"Mobile app cannot load tickets", "Mobile gateway", "dependency", "sev2", -19, 29},
		{"Checkout errors during the Lakeside Festival on-sale", "Checkout API", "capacity", "sev1", -9, 47},
		{"Fraud check slow after a model update", "Fraud check", "deploy", "sev3", -5, 55},
		{"Notification queue backing up", "Notifications", "capacity", "sev3", -2, 70},
	}
	var incRows []row
	totalMin := 0
	for i, in := range incidents {
		state := "resolved"
		if i == len(incidents)-1 {
			state = "monitoring"
		}
		totalMin += in.minutes
		incRows = append(incRows, row{Title: in.title, Values: map[string]any{"severity": in.sev, "service": in.service, "cause": in.cause, "started": xaDay(in.day), "minutes": in.minutes, "state": state, "commander": owner[in.service][0]}})
	}
	incRows[27].Markdown = "Full write-up: {{page:en_pm}}.\n"
	nums := []string{"[[vel]]", strconv.Itoa(int(math.Round(float64(velSum) / float64(velN)))), "[[cur]]", strconv.Itoa(planned[6]), "[[curdone]]", strconv.Itoa(completed[6]), "[[inc]]", strconv.Itoa(len(incidents)), "[[mttr]]", strconv.Itoa(int(math.Round(float64(totalMin) / float64(len(incidents))))), "[[svc]]", strconv.Itoa(len(services))}
	sevFilter := `{"prop":"severity","cond":"is","value":["sev1"]}`
	return Starter{
		ID: "engineering-team", Name: "Engineering team", Description: "Sprints and velocity, issues, incidents on a calendar heatmap, a service dependency graph, a postmortem and a runbook.", Icon: "i:code:cyan", Kind: "doc", Category: catExamples, Tags: []string{"sprints", "incidents", "services", "on-call"},
		markdown: xaFill(`The **Harbor Tickets** engineering team builds the platform that sells concert and festival tickets. Six engineers in three squads own [[svc]] services. This page is where we plan sprints, track work and learn from incidents.

> [!TIP]
> We finish about **[[vel]] points** a sprint. The current sprint has [[cur]] points planned and [[curdone]] done.

## Sprints and velocity

{{db:en_sprint}}

## Incidents this year

[[inc]] incidents in the last twelve months, resolved in about [[mttr]] minutes on average.

{{db:en_inc}}

## How our services connect

Each dot is a service or a shared dependency; lines show what depends on what. Bigger dots handle more traffic.

{{db:en_svc}}

## Read next

- {{page:en_pm}}: what happened during the Lakeside Festival on-sale and what we changed
- {{page:en_oncall}}: how on-call works, with the {{page:en_runbook}}
- {{page:en_issues}}: every story, bug and chore
`, nums...),
		children: []Starter{
			{
				key: "en_deps", Name: "Shared dependencies", Icon: "i:server:teal", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Dependency", Type: "title"},
					{ID: "kind", Name: "Kind", Type: "select", Options: []opt{o("data", "Data store", "blue"), o("queue", "Queue", "purple"), o("vendor", "Vendor", "orange")}},
				},
				views: []view{{Name: "All", Type: "table", Config: `{}`}},
				rows:  depRows,
			},
			{
				key: "en_svc", Name: "Services", Icon: "i:layers:cyan", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Service", Type: "title"},
					{ID: "tier", Name: "Tier", Type: "select", Options: []opt{o("t1", "Tier 1 · critical", "red"), o("t2", "Tier 2", "orange"), o("t3", "Tier 3", "gray")}},
					{ID: "team", Name: "Squad", Type: "select", Options: []opt{o("checkout", "Checkout", "purple"), o("discovery", "Discovery", "blue"), o("platform", "Platform", "green")}},
					{ID: "lang", Name: "Language", Type: "select", Options: []opt{o("go", "Go", "blue"), o("ts", "TypeScript", "yellow"), o("python", "Python", "green"), o("kotlin", "Kotlin", "purple")}},
					{ID: "oncall", Name: "On call", Type: "select", Options: engineers},
					{ID: "rpm", Name: "Requests per minute", Type: "number", Format: "comma"},
					{ID: "p95", Name: "p95 latency (ms)", Type: "number"},
					{ID: "uptime", Name: "Uptime (%)", Type: "number", Precision: 2},
					{ID: "uses", Name: "Depends on", Type: "relation", Collection: "@en_deps"},
				},
				views: []view{
					{Name: "Dependency graph", Type: "graph", Config: `{"layout":{"graph":{"color":"tier","size":"rpm","links":["uses"],"labels":"auto"}}}`},
					{Name: "Catalog", Type: "table", Config: `{"group":{"prop":"team"},"calcs":{"rpm":"sum","uptime":"average"}}`},
					{Name: "Cards", Type: "gallery", Config: `{"layout":{"cover":"none","size":"small"},` + xaCards("tier", "oncall", "uptime") + `}`},
					{Name: "Traffic", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"rpm","sort":"value_desc","labels":true}}}`},
				},
				rows: svcRows,
			},
			{
				key: "en_issues", Name: "Issues", Icon: "i:bug:rose", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Issue", Type: "title"},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("backlog", "Backlog", "gray", "todo"), so("todo", "To do", "brown", "todo"), so("progress", "In progress", "blue", "doing"), so("review", "In review", "purple", "doing"), so("done", "Done", "green", "done")}},
					{ID: "type", Name: "Type", Type: "select", Options: []opt{o("feature", "Feature", "green"), o("bug", "Bug", "red"), o("chore", "Chore", "gray"), o("debt", "Tech debt", "orange")}},
					{ID: "points", Name: "Points", Type: "number"},
					{ID: "assignee", Name: "Assignee", Type: "select", Options: engineers},
					{ID: "service", Name: "Service", Type: "relation", Collection: "@en_svc"},
					{ID: "created", Name: "Created", Type: "date"},
					{ID: "donepts", Name: "Done points", Type: "formula", Expr: `if(prop("Status") == "Done", prop("Points"), 0)`},
					{ID: "id", Name: "ID", Type: "unique_id", Prefix: "HT"},
				},
				views: []view{
					{Name: "Open work", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},"filter":{"prop":"status","cond":"group_is","value":["todo","doing"]},` + xaCards("type", "points", "assignee") + `,"layout":{"total":{"prop":"points","fn":"sum"}}}`},
					{Name: "All issues", Type: "table", Config: `{"group":{"prop":"type"},"calcs":{"points":"sum"},"props":[{"id":"donepts","hidden":true}]}`},
					{Name: "By assignee", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"assignee"},"series":{"prop":"type"},"fn":"sum","prop":"points","stacked":true,"legend":true}}}`},
					{Name: "Issue sizes", Type: "chart", Config: `{"layout":{"chart":{"kind":"histogram","x":{"prop":"points","step":1},"fn":"count"}}}`},
				},
				rows: issueRows,
			},
			{
				key: "en_sprint", Name: "Sprints", Icon: "i:kanban:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Sprint", Type: "title"},
					{ID: "dates", Name: "Dates", Type: "date"},
					{ID: "state", Name: "State", Type: "select", Options: []opt{o("done", "Finished", "green"), o("current", "Current", "blue"), o("next", "Next", "gray")}},
					{ID: "goal", Name: "Goal", Type: "text"},
					{ID: "planned", Name: "Planned points", Type: "rollup", Relation: "issues", Target: "points", Fn: "sum"},
					{ID: "velocity", Name: "Completed points", Type: "rollup", Relation: "issues", Target: "donepts", Fn: "sum"},
					{ID: "completion", Name: "Completion", Type: "formula", Expr: `if(prop("Planned points") > 0, round(prop("Completed points") / prop("Planned points") * 100) / 100, 0)`, Display: "bar", Max: 1, Format: "percent"},
					{ID: "issues", Name: "Issues", Type: "relation", Collection: "@en_issues"},
				},
				views: []view{
					{Name: "Velocity", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"s1","title":"Average velocity","size":"s","chart":{"kind":"number","fn":"average","prop":"velocity","filter":{"prop":"state","cond":"is","value":["done"]}}},{"id":"s2","title":"Current sprint done","size":"s","chart":{"kind":"gauge","fn":"sum","prop":"velocity","target":` + strconv.Itoa(planned[6]) + `,"filter":{"prop":"state","cond":"is","value":["current"]}}},{"id":"s3","title":"Points finished per sprint","size":"l","chart":{"kind":"bar","x":{"prop":"title"},"fn":"sum","prop":"velocity","labels":true,"target":` + strconv.Itoa(int(math.Round(float64(velSum)/float64(velN)))) + `}},{"id":"s4","title":"Points planned per sprint","size":"m","chart":{"kind":"line","x":{"prop":"title"},"fn":"sum","prop":"planned","labels":true}},{"id":"s5","title":"Total points delivered","size":"m","chart":{"kind":"area","x":{"prop":"title"},"fn":"sum","prop":"velocity","cumulative":true,"smooth":true}}]}}`},
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"dates","zoom":"month","table":false}}`},
					{Name: "Sprints", Type: "table", Config: `{"sorts":[{"prop":"dates","dir":"desc"}],"calcs":{"velocity":"average"}}`},
				},
				rows: sprintRows,
			},
			{
				key: "en_inc", Name: "Incidents", Icon: "i:flame:orange", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Incident", Type: "title"},
					{ID: "severity", Name: "Severity", Type: "select", Options: []opt{o("sev1", "SEV 1", "red"), o("sev2", "SEV 2", "orange"), o("sev3", "SEV 3", "yellow")}},
					{ID: "started", Name: "Started", Type: "date"},
					{ID: "minutes", Name: "Minutes to resolve", Type: "number", Precision: 1},
					{ID: "service", Name: "Service", Type: "relation", Collection: "@en_svc"},
					{ID: "cause", Name: "Cause", Type: "select", Options: []opt{o("deploy", "Bad deploy", "purple"), o("dependency", "Dependency", "blue"), o("capacity", "Capacity", "orange"), o("config", "Config change", "brown"), o("bug", "Code bug", "red")}},
					{ID: "state", Name: "State", Type: "status", Options: []opt{so("open", "Open", "red", "todo"), so("monitoring", "Monitoring", "yellow", "doing"), so("resolved", "Resolved", "green", "done")}},
					{ID: "commander", Name: "Incident lead", Type: "select", Options: engineers},
				},
				views: []view{
					{Name: "Overview", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"i1","title":"Incidents","size":"s","chart":{"kind":"number","fn":"count"}},{"id":"i2","title":"Average minutes to resolve","size":"s","chart":{"kind":"number","fn":"average","prop":"minutes"}},{"id":"i3","title":"SEV 1 incidents","size":"s","chart":{"kind":"number","fn":"count","filter":` + sevFilter + `}},{"id":"i4","title":"Minutes of impact","size":"s","chart":{"kind":"number","fn":"sum","prop":"minutes"}},{"id":"i5","title":"Incident days","size":"l","chart":{"kind":"calendar","x":{"prop":"started"},"fn":"count"}},{"id":"i6","title":"By month and severity","size":"m","chart":{"kind":"bar","x":{"prop":"started","by":"month"},"series":{"prop":"severity"},"fn":"count","stacked":true,"legend":true}},{"id":"i7","title":"Time to resolve","size":"m","chart":{"kind":"histogram","x":{"prop":"minutes","step":30},"fn":"count"}},{"id":"i8","title":"Causes","size":"m","chart":{"kind":"donut","x":{"prop":"cause"},"fn":"count","legend":true}},{"id":"i9","title":"Incidents per service","size":"m","chart":{"kind":"hbar","x":{"prop":"service"},"fn":"count","sort":"value_desc","labels":true}}]}}`},
					{Name: "Heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"started","domain":"month","count":12,"color":"rose"}}}`},
					{Name: "Log", Type: "table", Config: `{"sorts":[{"prop":"started","dir":"desc"}],"calcs":{"minutes":"average"}}`},
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"started"}}`},
				},
				rows: incRows,
			},
			{
				key: "en_pm", Name: "Postmortem: Lakeside Festival on-sale", Icon: "i:flame:rose", Kind: "doc",
				markdown: `| Severity | Duration | Incident lead | Status |
| --- | --- | --- | --- |
| SEV 1 | 47 minutes | Rosa Kim | Resolved, actions in progress |

> [!NOTE]
> This review is blameless. We look for causes in systems and processes, not in people.

<!-- toc -->

## Summary

When tickets for the Lakeside Festival went on sale nine days ago, the Checkout API ran out of database connections within two minutes. For 47 minutes, about one in three checkout attempts failed. We sold out four hours later than planned and refunded 312 duplicate payment holds.

## Impact

- **Checkout attempts failed:** 18,400 of 55,900 (33%)
- **Buyers affected:** about 9,800
- **Duplicate payment holds:** 312, all released within 24 hours
- **Support tickets:** 1,140 in the first day

## Timeline

| Time | What happened |
| --- | --- |
| 10:00 | Sale opens; queue page admits 4,000 buyers a minute |
| 10:02 | Checkout API connection pool saturated; error rate 31% |
| 10:04 | Alert fires; Rosa Kim takes the lead |
| 10:12 | Queue admission rate cut to 1,500 a minute |
| 10:25 | Read traffic moved to the replica; errors fall to 12% |
| 10:38 | Pool size raised and Checkout API scaled from 12 to 30 pods |
| 10:47 | Error rate back under 0.5% |

## What happened, step by step

` + fence + `mermaid
sequenceDiagram
  participant Q as Queue page
  participant C as Checkout API
  participant DB as Postgres primary
  participant P as Payments
  Q->>C: 4,000 buyers a minute
  C->>DB: hold seats (one connection each)
  DB-->>C: pool exhausted, wait
  C->>P: payment hold
  P-->>C: hold placed
  C-->>Q: timeout error
  Q->>C: buyer retries
  C->>P: second payment hold
` + fence + `

## Root cause

1. **Why did checkouts fail?** The Checkout API could not get a database connection.
2. **Why?** Each seat hold kept a connection open while it waited for the payment service.
3. **Why was that not caught?** Our load test used 1,000 buyers a minute, a quarter of real demand.
4. **Why did buyers get two payment holds?** Retries after a timeout did not reuse the idempotency key.

## Action items

| Action | Type | Owner | Status |
| --- | --- | --- | --- |
| Release the connection before calling Payments | Prevent | Rosa Kim | Done |
| Load test at 5,000 buyers a minute before every big on-sale | Detect | Tomas Berg | In progress |
| Reuse idempotency keys on retry | Prevent | Dev Malhotra | Done |
| Queue admission adapts to error rate | Mitigate | Femi Adebayo | Planned |

- [x] Share this write-up with support and the festival organizer
- [ ] Add the load test to the on-sale checklist in {{page:en_runbook}}
`,
			},
			{
				key: "en_oncall", Name: "On-call handbook", Icon: "i:bell:amber", Kind: "doc",
				markdown: `One engineer per squad is on call for a week, Monday 10:00 to Monday 10:00. The rotation and service owners are in {{page:en_svc}}.

## Severity levels

| Level | Meaning | Response | Example |
| --- | --- | --- | --- |
| SEV 1 | Buyers cannot buy, or money is wrong | Page now, all hands | Checkout failing for everyone |
| SEV 2 | A major feature is broken for many | Page now | Search returns nothing |
| SEV 3 | Degraded, with a workaround | Next working hour | Reports a day late |

## When the pager goes off

` + fence + `mermaid
flowchart TD
  A[Alert fires] --> B{Customers affected?}
  B -->|yes| C[Declare an incident and post in the incident channel]
  B -->|no| D[Fix within the working day]
  C --> E{Fixed in 15 minutes?}
  E -->|no| F[Page the secondary and the squad lead]
  E -->|yes| G[Write a short summary]
  F --> H[Status page update every 30 minutes]
  H --> G
  G --> I[Postmortem for SEV 1 and SEV 2]
` + fence + `

## Handover checklist

- [ ] Read the incidents from the past week in {{page:en_inc}}
- [ ] Check that your phone gets test pages
- [ ] Know where the runbooks are: {{page:en_runbook}}
- [ ] Confirm the secondary on-call engineer

> [!TIP]
> Nobody gets in trouble for paging too early. Page when unsure.
`,
				children: []Starter{
					{
						key: "en_runbook", Name: "Runbook: Checkout API", Icon: "i:terminal:green", Kind: "doc",
						markdown: `**Service:** Checkout API · **Squad:** Checkout · **Tier:** 1 · **Last tested:** {{date}}

> [!IMPORTANT]
> If buyers cannot complete checkout, declare a SEV 1 first, then follow this page.

## Quick checks

` + fence + `sh
curl -fsS https://checkout.internal.example.com/healthz
kubectl -n checkout get pods
` + fence + `

A healthy service answers in under 250 ms and runs at least 12 pods.

## Common alerts

<details>
<summary>Error rate above 2%</summary>

1. Check the last deploy. If it is under an hour old, roll back.
2. Check the database connection pool panel. If it is above 90%, see the next section.
3. Check the card processor status page.

</details>

<details>
<summary>Connection pool saturated</summary>

1. Lower the queue admission rate to 1,500 buyers a minute.
2. Scale the service: ` + "`kubectl -n checkout scale deploy/checkout-api --replicas=30`" + `
3. Move read traffic to the replica with the feature flag ` + "`checkout.read_replica`" + `.

</details>

<details>
<summary>Payment holds not released</summary>

1. Run the release job for holds older than 30 minutes.
2. Post the count in the incident channel for support.

</details>

## Before a big on-sale

- [ ] Load test at 5,000 buyers a minute
- [ ] Pre-scale to 30 pods an hour before
- [ ] Queue admission set to the agreed rate
- [ ] Someone from the Checkout squad watching the dashboards

## Roll back

` + fence + `sh
deploy rollback --service checkout-api --to previous
` + fence + `
`,
					},
				},
			},
		},
	}
}

type xaCustomer struct {
	name, plan       string
	arr, seats       int
	health           int
	csm, region, ind string
	renewal          int
	trend            string
}

func xaSuccess() Starter {
	csms := []opt{o("grace", "Grace Owusu", "purple"), o("ethan", "Ethan Brooks", "blue"), o("paula", "Paula Reyes", "green"), o("noah", "Noah Fischer", "orange")}
	agents := []opt{o("kenji", "Kenji Watanabe", "blue"), o("mira", "Mira Patel", "pink"), o("lucas", "Lucas Moreau", "green")}
	customers := []xaCustomer{
		{"Kestrel Airlines", "ent", 128000, 420, 86, "grace", "na", "travel", 74, "rising"},
		{"Meridian Health Group", "ent", 112000, 360, 78, "grace", "na", "health", 140, "flat"},
		{"Atlas Freight", "ent", 96000, 280, 72, "ethan", "na", "logistics", 33, "flat"},
		{"Brightwater Bank", "ent", 140000, 510, 81, "grace", "eu", "finance", 210, "rising"},
		{"Solstice Energy", "ent", 84000, 240, 69, "paula", "eu", "energy", 300, "rising"},
		{"Vantage Media", "ent", 88000, 260, 42, "ethan", "na", "media", 26, "falling"},
		{"Copperline Retail", "growth", 36000, 110, 74, "paula", "na", "retail", 95, "flat"},
		{"Bluefin Travel", "growth", 28000, 85, 81, "paula", "eu", "travel", 160, "rising"},
		{"Orchard Learning", "growth", 24000, 70, 88, "noah", "na", "education", 48, "rising"},
		{"Pinnacle Realty", "growth", 22000, 64, 66, "noah", "na", "realestate", 230, "flat"},
		{"Riverside Clinics", "growth", 32000, 96, 77, "ethan", "na", "health", 118, "flat"},
		{"Summit Outdoor", "growth", 26000, 78, 83, "paula", "eu", "retail", 62, "rising"},
		{"Tandem Bikes", "growth", 18000, 52, 38, "noah", "eu", "retail", 19, "falling"},
		{"Quarry Studios", "growth", 20000, 58, 71, "noah", "apac", "media", 280, "rising"},
		{"Lumen Insurance", "growth", 40000, 120, 55, "ethan", "na", "finance", 57, "falling"},
		{"Fable Games", "growth", 30000, 90, 79, "paula", "apac", "media", 310, "rising"},
		{"Harbor Coffee Co", "growth", 19000, 55, 84, "noah", "na", "retail", 175, "flat"},
		{"Pebble Pet Supplies", "starter", 6000, 15, 73, "noah", "na", "retail", 40, "flat"},
		{"Nimbus Labs", "starter", 8000, 20, 90, "noah", "eu", "software", 265, "rising"},
		{"Willow Florists", "starter", 4500, 8, 64, "noah", "na", "retail", 88, "flat"},
		{"Cobalt Dental", "starter", 5000, 10, 76, "noah", "apac", "health", 250, "flat"},
		{"Maple Yoga", "starter", 4200, 6, 81, "noah", "na", "health", 290, "rising"},
		{"Sprout Kids Clothing", "starter", 7000, 18, 68, "noah", "eu", "retail", 130, "flat"},
		{"Ember Bakery", "starter", 4800, 9, 45, "noah", "na", "retail", 22, "falling"},
	}
	totalARR, atRisk, healthSum := 0, 0, 0
	var custRows []row
	for _, c := range customers {
		totalARR += c.arr
		healthSum += c.health
		if c.health < 60 {
			atRisk++
		}
		custRows = append(custRows, row{Title: c.name, Values: map[string]any{"plan": c.plan, "arr": c.arr, "seats": c.seats, "health": c.health, "csm": c.csm, "region": c.region, "industry": c.ind, "renewal": xaDay(c.renewal), "trend": c.trend, "contact": "admin@" + strings.ToLower(strings.Fields(c.name)[0]) + ".example.com"}})
	}
	custRows[0].Markdown = "Quarterly review: {{page:cs_qbr}}.\n"
	custRows[5].Markdown = "> [!CAUTION]\n> Health dropped from 71 to 42 in two months. Their analytics lead left and weekly active users fell by half. Renewal is in under a month.\n\n## Save plan\n\n- [x] Executive call with their new head of data\n- [ ] Two training sessions for the new team\n- [ ] Offer a six-month renewal with an exit clause\n"
	cats := []string{"bug", "howto", "billing", "feature", "integration", "access"}
	titles := map[string][]string{
		"bug":         {"Dashboard export times out", "Funnel chart shows wrong totals", "Mobile SDK drops events offline", "Scheduled report sent twice", "Filters reset after refresh", "Retention chart blank for one cohort", "Time zone wrong in the daily digest", "CSV import skips rows with accents", "Alert fired with no data", "Chart legend overlaps on small screens"},
		"howto":       {"How to share a dashboard publicly", "Set up a weekly email report", "Compare two date ranges", "Track a sign-up funnel", "Create a custom event", "Invite a teammate as a viewer", "Group users by company", "Build a cohort of power users", "Exclude internal traffic", "Annotate a release on a chart"},
		"billing":     {"Invoice needs our VAT number", "Charged for removed seats", "Switch to annual billing", "Update the card on file", "Request a tax form", "Upgrade from Starter to Growth", "Refund for a duplicate charge", "Change the billing contact", "Question about a prorated charge", "Discount for a nonprofit"},
		"feature":     {"Request: dark mode for dashboards", "Request: export to slides", "Request: custom fiscal year", "Request: more than 10 funnel steps", "Request: single sign-on for contractors", "Request: alert on percentage change", "Request: comments on charts", "Request: data retention settings", "Request: higher API rate limit", "Request: shared filters across dashboards"},
		"integration": {"Warehouse sync failing nightly", "CRM sync missing new contacts", "Webhook returns 401", "Warehouse schema changed", "Event stream sync delayed", "Chat notifications stopped", "Spreadsheet connector times out", "Sign-on metadata expired", "Reverse sync overwrote fields", "Events stopped after a key rotation"},
		"access":      {"Locked out after a password reset", "Two-factor codes not arriving", "Former employee still has access", "Transfer admin rights", "Sign-on login loops", "Viewer cannot see a shared dashboard", "Account owner left the company", "Audit log request", "Remove a user from all projects", "Domain verification stuck"},
	}
	baseHours := map[string]float64{"bug": 30, "howto": 3, "billing": 7, "feature": 40, "integration": 22, "access": 2.5}
	weighted := []int{0, 0, 0, 0, 0, 1, 2, 3, 3, 4, 5, 5, 5, 6, 7, 8, 9, 10, 11, 12, 12, 13, 14, 14, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 23}
	var tixRows []row
	custTix := map[string]map[string]int{}
	var hoursSum float64
	solved, happy := 0, 0
	quota := map[string]int{"howto": 10, "bug": 10, "integration": 10, "billing": 9, "access": 8, "feature": 7}
	var order []string
	for len(order) < 54 {
		for _, c := range []string{"howto", "bug", "integration", "howto", "billing", "access", "bug", "integration", "feature", "billing"} {
			if quota[c] > 0 && len(order) < 54 {
				quota[c]--
				order = append(order, c)
			}
		}
	}
	used := map[string]int{}
	for i, cat := range order {
		cust := customers[weighted[(i*7+i/6)%len(weighted)]].name
		day := -1 - int(58*xaNoise(71, i))
		v := map[string]any{"customer": cust, "category": cat, "opened": xaDay(day), "agent": agents[i%3].ID, "channel": []string{"email", "chat", "chat", "email", "phone"}[i%5]}
		pr := xaNoise(72, i)
		switch {
		case pr < 0.08:
			v["priority"] = "urgent"
		case pr < 0.3:
			v["priority"] = "high"
		case pr < 0.85:
			v["priority"] = "normal"
		default:
			v["priority"] = "low"
		}
		if custTix[cust] == nil {
			custTix[cust] = map[string]int{}
		}
		custTix[cust][cat]++
		if day > -4 && i%3 != 0 {
			v["status"] = []string{"new", "open", "waiting"}[i%3]
		} else {
			h := math.Round(baseHours[cat]*(0.35+1.3*xaNoise(73, i))*10) / 10
			v["status"] = "solved"
			v["hours"] = h
			ok := h < baseHours[cat]*1.2 || xaNoise(74, i) < 0.3
			v["csat"] = ok
			hoursSum += h
			solved++
			if ok {
				happy++
			}
		}
		tixRows = append(tixRows, row{Title: titles[cat][used[cat]], Values: v})
		used[cat]++
	}
	tixRows[1].Markdown = "## Steps to reproduce\n\n1. Open a dashboard with more than 40 charts\n2. Choose Export, then PDF\n3. The export spins for two minutes, then fails\n\n## Workaround\n\nExport in two halves using the chart selector. Engineering ticket filed; fix expected in the next release.\n"
	comments := map[string][]string{
		"pro": {"Our weekly business review runs on these dashboards now.", "Support answers in minutes, not days.", "The funnel reports changed how we plan releases.", "Setup was quick and the team picked it up fast.", "Best analytics tool we have used.", "Cohorts finally make sense to our marketing team."},
		"pas": {"Good product, but exports are slow.", "Does what we need; pricing feels high.", "Solid, though the mobile view could be better.", "Useful, but onboarding new teammates takes time."},
		"det": {"Too many sync failures this quarter.", "We lost our champion and nobody else knows the tool.", "Hard to justify the cost for how much we use it.", "Bugs in reports made our numbers wrong twice."},
	}
	var npsRows []row
	custNPS := map[string][]int{}
	promoters, detractors := 0, 0
	for i := 0; i < 45; i++ {
		c := customers[(i*5+i/24)%len(customers)]
		score := int(math.Round(float64(c.health)/10 + 0.3 + 1.9*xaNoise(75, i)))
		score = min(10, max(0, score))
		band := "pas"
		if score >= 9 {
			band = "pro"
			promoters++
		} else if score <= 6 {
			band = "det"
			detractors++
		}
		custNPS[c.name] = append(custNPS[c.name], score)
		list := comments[band]
		npsRows = append(npsRows, row{Title: fmt.Sprintf("Response %d · %s", i+1, c.name), Values: map[string]any{"customer": c.name, "score": score, "comment": list[i%len(list)], "date": xaDay(-2 - int(85*xaNoise(76, i))), "role": []string{"admin", "analyst", "exec", "analyst"}[i%4]}})
	}
	nps := int(math.Round(float64(promoters-detractors) / 45 * 100))
	onb := []struct {
		cust, stage string
		from, to    int
		pct         int
		csm         string
	}{
		{"Solstice Energy", "live", -62, -8, 100, "paula"},
		{"Quarry Studios", "training", -30, 6, 70, "noah"},
		{"Fable Games", "setup", -18, 16, 45, "paula"},
		{"Nimbus Labs", "live", -45, -12, 100, "noah"},
		{"Maple Yoga", "training", -21, 4, 80, "noah"},
		{"Cobalt Dental", "setup", -10, 20, 30, "noah"},
		{"Bluefin Travel", "kickoff", -3, 30, 10, "paula"},
		{"Brightwater Bank", "setup", -14, 40, 35, "grace"},
	}
	var onbRows []row
	for _, x := range onb {
		onbRows = append(onbRows, row{Title: "Onboarding · " + x.cust, Values: map[string]any{"customer": x.cust, "stage": x.stage, "dates": xaSpan(x.from, x.to), "pct": x.pct, "csm": x.csm}})
	}
	k := custTix["Kestrel Airlines"]
	kTotal := 0
	for _, n := range k {
		kTotal += n
	}
	var pie strings.Builder
	pie.WriteString(fence + "mermaid\npie showData\n  title Kestrel Airlines tickets this quarter\n")
	catName := map[string]string{"bug": "Bugs", "howto": "How-to", "billing": "Billing", "feature": "Feature requests", "integration": "Integrations", "access": "Access"}
	for _, c := range cats {
		if k[c] > 0 {
			fmt.Fprintf(&pie, "  \"%s\" : %d\n", catName[c], k[c])
		}
	}
	pie.WriteString(fence)
	kScores := custNPS["Kestrel Airlines"]
	kAvg := 0.0
	for _, s := range kScores {
		kAvg += float64(s)
	}
	if len(kScores) > 0 {
		kAvg /= float64(len(kScores))
	}
	nums := []string{"[[arr]]", xaMoney(float64(totalARR)), "[[cust]]", strconv.Itoa(len(customers)), "[[risk]]", strconv.Itoa(atRisk), "[[health]]", strconv.Itoa(int(math.Round(float64(healthSum) / float64(len(customers))))), "[[nps]]", strconv.Itoa(nps), "[[hours]]", strconv.FormatFloat(math.Round(hoursSum/float64(solved)*10)/10, 'f', 1, 64), "[[csat]]", strconv.Itoa(int(math.Round(float64(happy) / float64(solved) * 100))), "[[ktix]]", strconv.Itoa(kTotal), "[[pie]]", pie.String(), "[[knps]]", strconv.FormatFloat(math.Round(kAvg*10)/10, 'f', 1, 64), "[[kresp]]", strconv.Itoa(len(kScores))}
	openFilter := `{"prop":"status","cond":"group_is","value":["todo","doing"]}`
	return Starter{
		ID: "customer-success", Name: "Customer success", Description: "Keep customers healthy: health scores, support tickets, NPS results, onboarding and a quarterly business review.", Icon: "i:heart:rose", Kind: "doc", Category: catExamples, Tags: []string{"health score", "support", "nps", "renewals"},
		markdown: xaFill(`**Lighthouse Analytics** makes product analytics software. The customer success team looks after [[cust]] customers worth **[[arr]]** a year. This page shows who is healthy, who needs help and what customers tell us.

> [!WARNING]
> [[risk]] customers have a health score under 60. Start with the ones renewing soonest.

## Customer health

{{db:cs_cust}}

## Support

Solved tickets took [[hours]] hours on average, and [[csat]]% of customers were happy with the answer.

{{db:cs_tix}}

## What customers say

Our Net Promoter Score is **[[nps]]**.

{{db:cs_nps}}

## More

- {{page:cs_onb}}: new customers getting set up, on a timeline
- {{page:cs_playbook}}: how we score health and when to escalate
- {{page:cs_qbr}}: the quarterly business review for Kestrel Airlines
`, nums...),
		children: []Starter{
			{
				key: "cs_cust", Name: "Customers", Icon: "i:building:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Customer", Type: "title"},
					{ID: "health", Name: "Health score", Type: "number", Display: "ring", Max: 100, Precision: 1},
					{ID: "plan", Name: "Plan", Type: "select", Options: []opt{o("starter", "Starter", "yellow"), o("growth", "Growth", "blue"), o("ent", "Enterprise", "purple")}},
					{ID: "arr", Name: "Annual value", Type: "number", Format: "dollar"},
					{ID: "seats", Name: "Seats", Type: "number"},
					{ID: "csm", Name: "Success manager", Type: "select", Options: csms},
					{ID: "renewal", Name: "Renewal", Type: "date"},
					{ID: "days", Name: "Days to renewal", Type: "formula", Expr: `dateBetween(prop("Renewal"), now(), "days")`},
					{ID: "trend", Name: "Usage trend", Type: "select", Options: []opt{o("rising", "Rising", "green"), o("flat", "Flat", "gray"), o("falling", "Falling", "red")}},
					{ID: "region", Name: "Region", Type: "select", Options: []opt{o("na", "North America", "blue"), o("eu", "Europe", "green"), o("apac", "Asia Pacific", "orange")}},
					{ID: "industry", Name: "Industry", Type: "select", Options: []opt{o("travel", "Travel", "blue"), o("health", "Health", "green"), o("logistics", "Logistics", "brown"), o("finance", "Finance", "purple"), o("energy", "Energy", "yellow"), o("media", "Media", "pink"), o("retail", "Retail", "orange"), o("education", "Education", "gray"), o("realestate", "Real estate", "red"), o("software", "Software", "default")}},
					{ID: "contact", Name: "Main contact", Type: "email"},
				},
				views: []view{
					{Name: "Health", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"c1","title":"Average health","size":"s","chart":{"kind":"gauge","fn":"average","prop":"health","target":80}},{"id":"c2","title":"Customers","size":"s","chart":{"kind":"number","fn":"count"}},{"id":"c3","title":"At risk (health under 60)","size":"s","chart":{"kind":"number","fn":"count","filter":{"prop":"health","cond":"lt","value":60}}},{"id":"c4","title":"Seats","size":"s","chart":{"kind":"number","fn":"sum","prop":"seats"}},{"id":"c5","title":"Annual value by plan","size":"m","chart":{"kind":"donut","x":{"prop":"plan"},"fn":"sum","prop":"arr","legend":true}},{"id":"c6","title":"Health of every customer","size":"m","chart":{"kind":"dots","x":{"prop":"plan"},"prop":"health"}},{"id":"c7","title":"Renewals by month","size":"l","chart":{"kind":"bar","x":{"prop":"renewal","by":"month"},"series":{"prop":"trend"},"fn":"sum","prop":"arr","stacked":true,"legend":true}},{"id":"c8","title":"Average health by manager","size":"m","chart":{"kind":"hbar","x":{"prop":"csm"},"fn":"average","prop":"health","labels":true,"target":80}},{"id":"c9","title":"Usage trend","size":"m","chart":{"kind":"radial","x":{"prop":"trend"},"fn":"count","legend":true}}]}}`},
					{Name: "All customers", Type: "table", Config: `{"sorts":[{"prop":"health","dir":"asc"}],"calcs":{"arr":"sum","health":"average"}}`},
					{Name: "By manager", Type: "board", Config: `{"group":{"prop":"csm","hide_empty":true},` + xaCards("health", "plan", "renewal") + `,"layout":{"total":{"prop":"arr","fn":"sum"}}}`},
					{Name: "Renewals", Type: "calendar", Config: `{"layout":{"date":"renewal"}}`},
				},
				rows: custRows,
			},
			{
				key: "cs_tix", Name: "Support tickets", Icon: "i:inbox:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Ticket", Type: "title"},
					{ID: "customer", Name: "Customer", Type: "relation", Collection: "@cs_cust"},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("new", "New", "gray", "todo"), so("open", "Open", "blue", "doing"), so("waiting", "Waiting on customer", "yellow", "doing"), so("solved", "Solved", "green", "done")}},
					{ID: "category", Name: "Category", Type: "select", Options: []opt{o("bug", "Bug", "red"), o("howto", "How-to", "blue"), o("billing", "Billing", "yellow"), o("feature", "Feature request", "purple"), o("integration", "Integration", "orange"), o("access", "Access", "gray")}},
					{ID: "priority", Name: "Priority", Type: "select", Options: []opt{o("urgent", "Urgent", "red"), o("high", "High", "orange"), o("normal", "Normal", "blue"), o("low", "Low", "gray")}},
					{ID: "channel", Name: "Channel", Type: "select", Options: []opt{o("email", "Email", "blue"), o("chat", "Chat", "green"), o("phone", "Phone", "purple")}},
					{ID: "agent", Name: "Agent", Type: "select", Options: agents},
					{ID: "opened", Name: "Opened", Type: "date"},
					{ID: "hours", Name: "Hours to resolve", Type: "number", Precision: 1},
					{ID: "csat", Name: "Happy with answer", Type: "checkbox"},
					{ID: "id", Name: "Ticket", Type: "unique_id", Prefix: "LH"},
				},
				views: []view{
					{Name: "Support overview", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"t1","title":"Open tickets","size":"s","chart":{"kind":"number","fn":"count","filter":` + openFilter + `}},{"id":"t2","title":"Median hours to resolve","size":"s","chart":{"kind":"number","fn":"median","prop":"hours"}},{"id":"t3","title":"Happy customers","size":"s","chart":{"kind":"gauge","fn":"percent_checked","prop":"csat","filter":{"prop":"status","cond":"is","value":["solved"]}}},{"id":"t4","title":"Tickets in the last 60 days","size":"s","chart":{"kind":"number","fn":"count"}},{"id":"t5","title":"How long tickets take (hours)","size":"m","chart":{"kind":"histogram","x":{"prop":"hours","step":6},"fn":"count"}},{"id":"t6","title":"By category","size":"m","chart":{"kind":"donut","x":{"prop":"category"},"fn":"count","legend":true,"labels":true}},{"id":"t7","title":"Tickets per week","size":"l","chart":{"kind":"area","x":{"prop":"opened","by":"week"},"series":{"prop":"category"},"fn":"count","stacked":true,"legend":true}},{"id":"t8","title":"Busiest customers","size":"m","chart":{"kind":"hbar","x":{"prop":"customer"},"fn":"count","sort":"value_desc","limit":8,"labels":true}},{"id":"t9","title":"Hours by priority","size":"m","chart":{"kind":"bar","x":{"prop":"priority"},"fn":"average","prop":"hours","labels":true}}]}}`},
					{Name: "Queue", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},` + xaCards("customer", "category", "priority") + `,"layout":{"total":{"fn":"count"}}}`},
					{Name: "All tickets", Type: "table", Config: `{"sorts":[{"prop":"opened","dir":"desc"}],"calcs":{"hours":"average","csat":"percent_checked"}}`},
					{Name: "Ticket heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"opened","domain":"month","count":3,"color":"violet"}}}`},
				},
				rows: tixRows,
			},
			{
				key: "cs_nps", Name: "NPS survey", Icon: "i:star:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Response", Type: "title"},
					{ID: "customer", Name: "Customer", Type: "relation", Collection: "@cs_cust"},
					{ID: "score", Name: "Score", Type: "number", Display: "bar", Max: 10},
					{ID: "group", Name: "Group", Type: "formula", Expr: `ifs(prop("Score") >= 9, "Promoter", prop("Score") >= 7, "Passive", "Detractor")`},
					{ID: "points", Name: "NPS points", Type: "formula", Expr: `ifs(prop("Score") >= 9, 100, prop("Score") >= 7, 0, -100)`},
					{ID: "comment", Name: "Comment", Type: "text"},
					{ID: "role", Name: "Role", Type: "select", Options: []opt{o("admin", "Admin", "blue"), o("analyst", "Analyst", "green"), o("exec", "Executive", "purple")}},
					{ID: "date", Name: "Answered", Type: "date"},
				},
				views: []view{
					{Name: "Results", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"n1","title":"Net Promoter Score","size":"s","chart":{"kind":"gauge","fn":"average","prop":"points","target":40}},{"id":"n2","title":"Responses","size":"s","chart":{"kind":"number","fn":"count"}},{"id":"n3","title":"Average score","size":"s","chart":{"kind":"number","fn":"average","prop":"score"}},{"id":"n4","title":"Promoters","size":"s","chart":{"kind":"number","fn":"count","filter":{"prop":"score","cond":"gte","value":9}}},{"id":"n5","title":"Scores","size":"m","chart":{"kind":"histogram","x":{"prop":"score","step":1},"fn":"count"}},{"id":"n6","title":"Promoters, passives, detractors","size":"m","chart":{"kind":"donut","x":{"prop":"group"},"fn":"count","legend":true,"labels":true}},{"id":"n7","title":"Score by role","size":"m","chart":{"kind":"dots","x":{"prop":"role"},"prop":"score"}},{"id":"n8","title":"NPS by month","size":"m","chart":{"kind":"line","x":{"prop":"date","by":"month"},"fn":"average","prop":"points","labels":true}}]}}`},
					{Name: "Comments", Type: "list", Config: `{"sorts":[{"prop":"date","dir":"desc"}],` + xaCards("score", "comment", "customer") + `}`},
					{Name: "All responses", Type: "table", Config: `{"sorts":[{"prop":"score","dir":"asc"}],"calcs":{"points":"average","score":"average"}}`},
					{Name: "Survey", Type: "form", Config: `{"form":{"title":"How likely are you to recommend Lighthouse?","description":"One question, one minute. Your answer goes straight to the customer success team.","questions":[{"prop":"score","label":"From 0 (not at all) to 10 (extremely likely)","required":true},{"prop":"comment","label":"What is the main reason for your score?"},{"prop":"role","label":"What best describes your role?"}],"submit":"Send","done":"Thank you. We read every answer."}}`},
				},
				rows: npsRows,
			},
			{
				key: "cs_onb", Name: "Onboarding", Icon: "i:rocket:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Project", Type: "title"},
					{ID: "customer", Name: "Customer", Type: "relation", Collection: "@cs_cust"},
					{ID: "dates", Name: "Dates", Type: "date"},
					{ID: "stage", Name: "Stage", Type: "status", Options: []opt{so("kickoff", "Kickoff", "gray", "todo"), so("setup", "Data setup", "yellow", "doing"), so("training", "Training", "blue", "doing"), so("live", "Live", "green", "done")}},
					{ID: "pct", Name: "Complete", Type: "number", Display: "bar", Max: 100},
					{ID: "csm", Name: "Success manager", Type: "select", Options: csms},
				},
				views: []view{
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"dates","zoom":"month","table":false}}`},
					{Name: "Board", Type: "board", Config: `{"group":{"prop":"stage","hide_empty":true},` + xaCards("customer", "pct", "csm") + `,"layout":{"progress":"pct"}}`},
					{Name: "Table", Type: "table", Config: `{"calcs":{"pct":"average"}}`},
				},
				rows: onbRows,
			},
			{
				key: "cs_playbook", Name: "Customer success playbook", Icon: "i:book:teal", Kind: "doc",
				markdown: `How we measure customer health, what we do at each score and when a problem becomes an escalation.

## The health score

Every customer gets a score from 0 to 100, updated weekly:

$$
H = 40 \cdot \frac{\text{weekly active users}}{\text{seats}} + 25 \cdot \text{feature depth} + 20 \cdot \text{support sentiment} + 15 \cdot \text{exec engagement}
$$

Each part is scaled from 0 to 1. A customer with half their seats active each week, using most features, with happy support tickets and a recent executive meeting scores about $40 \times 0.5 + 25 \times 0.8 + 20 \times 0.9 + 15 \times 1 = 73$.

| Score | Status | What we do |
| --- | --- | --- |
| 80 to 100 | Healthy | Ask for a case study or referral |
| 60 to 79 | Watch | Monthly check-in, training offer |
| Under 60 | At risk | Save plan within a week, manager involved |

## Touchpoints by plan

| Plan | Check-ins | Business reviews | Training |
| --- | --- | --- | --- |
| Enterprise | Every two weeks | Quarterly | On site, twice a year |
| Growth | Monthly | Twice a year | Live webinars |
| Starter | Automated emails | None | Self-serve videos |

## Escalation

` + fence + `mermaid
flowchart TD
  A[Health drops below 60 or an urgent ticket is open 24 hours] --> B[Success manager reviews within one day]
  B --> C{Product problem?}
  C -->|yes| D[Engineering escalation with the support lead]
  C -->|no| E{Champion left or usage falling?}
  E -->|yes| F[Executive call and training plan]
  E -->|no| G[Commercial issue: pricing review]
  D --> H[Weekly update to the customer until fixed]
  F --> H
  G --> H
` + fence + `

The details are in {{page:cs_escalation}}.

> [!TIP]
> Most saves start with a phone call, not a discount.
`,
				children: []Starter{
					{
						key: "cs_escalation", Name: "Escalation process", Icon: "i:bell:orange", Kind: "doc",
						markdown: `## When to escalate

- [ ] An urgent ticket is open for more than 24 hours
- [ ] A customer worth more than $50,000 a year drops below 60 health
- [ ] A customer mentions cancelling, in any channel
- [ ] Data shown to the customer was wrong

## Who does what

| Role | Responsibility | Response time |
| --- | --- | --- |
| Success manager | Owns the relationship and the plan | Same day |
| Support lead (Kenji Watanabe) | Owns the technical fix | 4 working hours |
| VP Customer Success (Grace Owusu) | Executive contact, commercial decisions | 1 working day |

## Writing the update

> [!IMPORTANT]
> Every escalation update says what happened, what we are doing, and when the next update comes. Never send an update without a next date.

<details>
<summary>Template</summary>

**What happened:** one or two sentences.

**What we are doing:** the fix and who is on it.

**What you can do now:** a workaround, if there is one.

**Next update:** day and time.

</details>
`,
					},
				},
			},
			{
				key: "cs_qbr", Name: "QBR: Kestrel Airlines", Icon: "i:chart-bar:blue", Kind: "doc",
				markdown: xaFill(`**Customer:** Kestrel Airlines · **Plan:** Enterprise, 420 seats · **Success manager:** Grace Owusu · **Date:** {{date}}

<!-- toc -->

## Where they are

| Measure | Last quarter | This quarter | Goal |
| --- | --- | --- | --- |
| Health score | 79 | 86 | 80 |
| Weekly active users | 188 | 241 | 250 |
| Dashboards in use | 34 | 52 | 50 |
| Support tickets | 9 | [[ktix]] | Fewer than 10 |
| Average survey score | 7.6 | [[knps]] ([[kresp]] answers) | 8.5 |

> [!TIP]
> Kestrel is our best example of a turnaround: one training day for the revenue team doubled weekly active users in that group.

## Support this quarter

[[pie]]

## Their goals for next quarter

1. Track on-time departures and rebooking rates in one dashboard for station managers
2. Connect the booking warehouse so marketing can build cohorts without engineering help
3. Roll out to the cargo division (about 120 more seats)

## Our commitments

- [ ] Warehouse connector set up and validated (Lighthouse solutions team)
- [ ] Two training sessions for station managers (Grace Owusu)
- [ ] Price proposal for the cargo division (Grace Owusu)
- [ ] Early access to custom fiscal year reporting (product team)

## Risks

| Risk | Plan |
| --- | --- |
| Cargo division uses a different warehouse | Scope a second connector in the proposal |
| Their analytics lead goes on leave next quarter | Train two deputies now |

Renewal is in about two and a half months; see {{page:cs_cust}}.
`, nums...),
			},
		},
	}
}

type xaPaper struct {
	title, authors string
	year           int
	topic, venue   string
	citations      int
	status         string
	rating         int
	cites          []string
}

func xaResearch() Starter {
	classics := []struct {
		title, authors string
		year, cites    int
		field          string
	}{
		{"Spectrogram correlation for baleen whale call detection", "Harlow and Biggs", 2004, 1240, "bio"},
		{"Mel-frequency features for animal vocalisations", "Okonkwo, Hale and Sun", 2008, 980, "features"},
		{"Passive acoustic monitoring: a field guide", "Moreno", 2011, 760, "bio"},
		{"Energy measurement for embedded inference", "Kowalski", 2012, 410, "hardware"},
		{"Convolutional networks for audio event detection", "Varga and Lund", 2014, 3100, "deep"},
		{"Knowledge distillation for compact acoustic models", "Brandt and Ruiz", 2015, 2200, "compress"},
		{"Depthwise separable convolutions on small devices", "Tanaka, Weiss and Obi", 2016, 5400, "deep"},
		{"Time and frequency masking for robust audio classifiers", "Iqbal, Moreau and Chen", 2017, 1900, "deep"},
		{"Eight-bit integer quantization of neural networks", "Feld, Navarro and Ito", 2017, 2600, "compress"},
		{"Self-supervised audio representations from unlabeled sound", "Achterberg and Park", 2019, 840, "deep"},
	}
	var classicRows []row
	for _, c := range classics {
		classicRows = append(classicRows, row{Title: c.title, Values: map[string]any{"authors": c.authors, "year": c.year, "citations": c.cites, "field": c.field}})
	}
	c := func(i ...int) []string {
		out := make([]string, len(i))
		for k, n := range i {
			out[k] = classics[n].title
		}
		return out
	}
	papers := []xaPaper{
		{"A benchmark of humpback song detectors", "Lindahl et al.", 2020, "datasets", "journal", 310, "read", 4, c(0, 2, 4)},
		{"Tiny convolutional detectors for right whale up-calls", "Osei and Varga", 2021, "efficient", "workshop", 145, "key", 5, c(0, 4, 6)},
		{"Augmenting scarce marine audio with mixing and masking", "Rahimi et al.", 2021, "augment", "conference", 98, "key", 5, c(7, 1)},
		{"Ship noise as a confound in call detection", "Fontaine and Achterberg", 2021, "detection", "journal", 76, "read", 4, c(0, 2)},
		{"Duty-cycled inference on solar buoys", "Kowalski and Yeo", 2022, "deploy", "workshop", 52, "key", 5, c(3, 8)},
		{"Distilling large audio models into microcontrollers", "Brandt et al.", 2022, "efficient", "conference", 230, "read", 4, c(5, 6, 8)},
		{"Contrastive pretraining for bioacoustic events", "Park and Achterberg", 2022, "selfsup", "conference", 164, "read", 3, c(9, 4)},
		{"An open dataset of North Atlantic buoy recordings", "Hale et al.", 2022, "datasets", "journal", 188, "key", 5, c(2)},
		{"Transformers for long-range acoustic context", "Sun and Ibarra", 2023, "detection", "conference", 121, "reading", 3, c(4, 1)},
		{"Quantization-aware training for keyword-sized audio models", "Ito and Feld", 2023, "efficient", "conference", 87, "read", 4, c(8, 6)},
		{"Label noise in crowd-annotated whale calls", "Moreno and Diallo", 2023, "datasets", "journal", 43, "read", 3, c(2, 0)},
		{"Energy-aware neural architecture search for sensors", "Yeo et al.", 2023, "efficient", "conference", 66, "reading", 4, c(3, 6, 5)},
		{"Few-shot detection of rare call types", "Navarro and Park", 2024, "selfsup", "workshop", 29, "toread", 0, c(9, 7)},
		{"Field trial of edge detectors in the Gulf of Maine", "Achterberg et al.", 2024, "deploy", "journal", 38, "key", 5, c(3, 2, 6)},
		{"Mixing real and synthetic calls for training", "Rahimi and Sun", 2024, "augment", "workshop", 17, "reading", 3, c(7)},
		{"Pruning versus distillation for audio on microcontrollers", "Lund and Brandt", 2024, "efficient", "conference", 22, "toread", 0, c(5, 8)},
		{"Domain shift between buoys and seasons", "Fontaine et al.", 2024, "detection", "conference", 14, "read", 4, c(0, 1, 2)},
		{"Self-supervised pretraining on ten thousand hours of ocean sound", "Park et al.", 2025, "selfsup", "conference", 9, "toread", 0, c(9)},
		{"Low-power wake-up detectors for acoustic tags", "Kowalski and Osei", 2025, "deploy", "workshop", 6, "toread", 0, c(3, 6)},
		{"Uncertainty estimates for call detectors", "Ibarra and Hale", 2025, "detection", "journal", 4, "toread", 0, c(4)},
		{"Small models, big ocean: detecting whale calls on a buoy", "Moreau and Achterberg", 2025, "efficient", "workshop", 3, "ours", 0, c(4, 6, 7, 8)},
		{"Seasonal augmentation for robust buoy detectors", "Moreau, Fontaine and Achterberg", 2026, "augment", "conference", 0, "ours", 0, c(7, 1)},
	}
	var paperRows []row
	for _, p := range papers {
		v := map[string]any{"authors": p.authors, "year": p.year, "topic": p.topic, "venue": p.venue, "citations": p.citations, "status": p.status, "cites": p.cites}
		if p.rating > 0 {
			v["rating"] = p.rating
		}
		paperRows = append(paperRows, row{Title: p.title, Values: v})
	}
	paperRows[1].Markdown = "## Notes\n\n- Three convolution layers, 41,000 parameters, runs on a microcontroller at 4 mW\n- Reports F1 of 0.84 on their own buoy data; no cross-buoy test\n- **For the thesis:** our closest baseline. Reproduce in {{page:rl_exp}}\n\n> [!TIP]\n> Their augmentation section is thin. Our seasonal mixing should beat it on Buoy B.\n"
	models := []struct {
		id             string
		params         int
		f1, spread, mj float64
	}{
		{"svm", 0, 71, 3, 0.4}, {"cnn_small", 480, 88, 2, 9.5}, {"cnn_tiny", 62, 83, 2.5, 1.6}, {"dscnn", 38, 86, 2, 1.1}, {"trans", 610, 89.5, 1.5, 14}, {"distilled", 38, 88.5, 1.2, 1.1},
	}
	plan := []struct {
		model      int
		aug, quant bool
		dataset    string
	}{
		{0, false, false, "a"}, {2, false, false, "a"}, {1, false, false, "a"}, {3, false, false, "a"}, {0, false, false, "b"}, {2, false, false, "b"},
		{1, false, false, "b"}, {3, false, false, "b"}, {2, true, false, "a"}, {3, true, false, "a"}, {1, true, false, "a"}, {4, false, false, "a"},
		{4, true, false, "b"}, {3, true, false, "b"}, {2, true, true, "b"}, {3, true, true, "b"}, {1, true, true, "a"}, {4, true, false, "mixed"},
		{3, true, false, "mixed"}, {2, true, false, "mixed"}, {5, false, false, "a"}, {5, true, false, "a"}, {5, true, false, "b"}, {5, true, true, "b"},
		{5, true, true, "mixed"}, {3, true, true, "mixed"}, {1, true, false, "mixed"}, {4, true, true, "mixed"}, {5, true, true, "a"}, {0, true, false, "mixed"},
		{5, true, false, "mixed"}, {3, false, true, "a"},
	}
	modelName := map[string]string{"svm": "MFCC and SVM baseline", "cnn_small": "Small CNN", "cnn_tiny": "Tiny CNN", "dscnn": "Depthwise CNN", "trans": "Compact transformer", "distilled": "Distilled depthwise CNN"}
	var expRows []row
	best := 0.0
	bestRun := ""
	for i, p := range plan {
		m := models[p.model]
		f1 := m.f1 + m.spread*(xaNoise(81, i)*2-1)
		if p.aug {
			f1 += 1.4
		}
		if p.dataset == "b" {
			f1 -= 2.2
		}
		mj := m.mj * (0.9 + 0.2*xaNoise(82, i))
		if p.quant {
			f1 -= 0.4
			mj *= 0.4
		}
		f1 = math.Round(f1*10) / 10
		mj = math.Round(mj*100) / 100
		name := fmt.Sprintf("Run %02d · %s", i+1, modelName[m.id])
		if p.aug {
			name += ", augmented"
		}
		if p.quant {
			name += ", 8-bit"
		}
		if f1 > best && m.id != "trans" && m.id != "cnn_small" {
			best, bestRun = f1, name
		}
		expRows = append(expRows, row{Title: name, Values: map[string]any{"model": m.id, "params": m.params, "f1": f1, "energy": mj, "dataset": p.dataset, "augment": p.aug, "quantized": p.quant, "date": xaDay(-118 + i*37/10)}})
	}
	expRows[len(expRows)-8].Markdown = "## Setup\n\n- Teacher: compact transformer from run 28\n- Student: depthwise CNN, 38,000 parameters\n- Temperature 4, loss weight 0.7 on soft targets\n\n## Result\n\nThe student keeps most of the teacher's accuracy at less than a tenth of the energy. This is the headline result for chapter 5.\n"
	type thesisTask struct {
		title, chapter, status string
		from, to, done, target int
	}
	thesis := []thesisTask{
		{"Literature review draft", "ch2", "done", -150, -60, 9800, 10000},
		{"Data collection chapter", "ch3", "done", -90, -40, 7200, 7000},
		{"Methods: model design", "ch4", "doing", -60, -10, 6100, 8000},
		{"Experiments: accuracy", "ch5", "doing", -30, 20, 3400, 9000},
		{"Workshop paper submission", "paper", "todo", 25, 25, 0, 0},
		{"Experiments: energy on the buoy", "ch5", "todo", -5, 35, 800, 6000},
		{"Committee meeting", "milestone", "todo", 40, 40, 0, 0},
		{"Field deployment chapter", "ch6", "todo", 20, 70, 0, 7000},
		{"Introduction", "ch1", "todo", 60, 80, 1200, 4000},
		{"Conclusion", "ch7", "todo", 75, 90, 0, 3000},
		{"Full draft to advisor", "milestone", "todo", 95, 95, 0, 0},
		{"Revisions", "all", "todo", 96, 130, 0, 0},
		{"Thesis defense", "milestone", "todo", 150, 150, 0, 0},
	}
	var thesisRows []row
	written, target := 0, 0
	for _, t := range thesis {
		v := map[string]any{"chapter": t.chapter, "status": t.status, "dates": xaSpan(t.from, t.to)}
		if t.target > 0 {
			v["words"], v["target"] = t.done, t.target
			written += t.done
			target += t.target
		}
		thesisRows = append(thesisRows, row{Title: t.title, Values: v})
	}
	nums := []string{"[[papers]]", strconv.Itoa(len(papers)), "[[classics]]", strconv.Itoa(len(classics)), "[[runs]]", strconv.Itoa(len(plan)), "[[best]]", strconv.FormatFloat(best, 'f', 1, 64), "[[bestrun]]", bestRun, "[[written]]", xaInt(written), "[[target]]", xaInt(target), "[[pct]]", strconv.Itoa(int(math.Round(float64(written) / float64(target) * 100)))}
	return Starter{
		ID: "research-lab", Name: "Research lab", Description: "A PhD lab notebook: papers and the works they cite as a graph, experiments on a scatter chart, a thesis timeline and a literature review.", Icon: "i:flask:teal", Kind: "doc", Category: catExamples, Tags: []string{"research", "papers", "experiments", "thesis"},
		markdown: xaFill(`The **Coastal Acoustics Lab** at Northbridge University builds small neural networks that listen for whale calls on solar-powered buoys. This is Ines Moreau's PhD workspace: what she reads, what she runs and what she writes.

> [!NOTE]
> The question: *how small can a whale call detector be before it stops being useful?* The best low-power model so far reaches an F1 score of **[[best]]%** ([[bestrun]]).

## Reading map

[[papers]] recent papers and the [[classics]] classic works they build on. Lines are citations; bigger dots are cited more. Drag the graph, or click a dot to open the paper.

{{db:rl_papers}}

## Experiments

[[runs]] training runs so far. Each dot is one run; further right is more accurate.

{{db:rl_exp}}

## Thesis plan

[[written]] of [[target]] words written ([[pct]]%).

{{db:rl_thesis}}

## Notes

- {{page:rl_review}}: the field in one page, with a mind map and the maths
- {{page:rl_protocol}}: how every experiment is run, so results compare
- {{page:rl_meetings}}: weekly notes with Professor Achterberg
`, nums...),
		children: []Starter{
			{
				key: "rl_classics", Name: "Classic works", Icon: "i:book:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Title", Type: "title"},
					{ID: "authors", Name: "Authors", Type: "text"},
					{ID: "year", Name: "Year", Type: "number"},
					{ID: "citations", Name: "Citations", Type: "number", Format: "comma"},
					{ID: "field", Name: "Field", Type: "select", Options: []opt{o("bio", "Bioacoustics", "green"), o("features", "Audio features", "yellow"), o("deep", "Deep learning", "blue"), o("compress", "Model compression", "purple"), o("hardware", "Hardware", "gray")}},
				},
				views: []view{
					{Name: "Works", Type: "table", Config: `{"sorts":[{"prop":"year","dir":"asc"}]}`},
					{Name: "Most cited", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"citations","sort":"value_desc","labels":true}}}`},
				},
				rows: classicRows,
			},
			{
				key: "rl_papers", Name: "Papers", Icon: "i:bookmark:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Title", Type: "title"},
					{ID: "authors", Name: "Authors", Type: "text"},
					{ID: "year", Name: "Year", Type: "number"},
					{ID: "topic", Name: "Topic", Type: "select", Options: []opt{o("detection", "Detection", "blue"), o("efficient", "Efficient models", "purple"), o("augment", "Augmentation", "orange"), o("datasets", "Datasets", "green"), o("selfsup", "Self-supervised", "pink"), o("deploy", "Deployment", "brown")}},
					{ID: "venue", Name: "Venue", Type: "select", Options: []opt{o("journal", "Journal", "blue"), o("conference", "Conference", "purple"), o("workshop", "Workshop", "gray")}},
					{ID: "status", Name: "Reading status", Type: "status", Options: []opt{so("toread", "To read", "gray", "todo"), so("reading", "Reading", "yellow", "doing"), so("read", "Read", "blue", "done"), so("key", "Key paper", "green", "done"), so("ours", "Our paper", "purple", "done")}},
					{ID: "rating", Name: "Usefulness", Type: "number", Display: "bar", Max: 5},
					{ID: "citations", Name: "Citations", Type: "number", Format: "comma"},
					{ID: "cites", Name: "Cites", Type: "relation", Collection: "@rl_classics"},
					{ID: "refs", Name: "Classic works cited", Type: "rollup", Relation: "cites", Target: "title", Fn: "count_all"},
				},
				views: []view{
					{Name: "Citation graph", Type: "graph", Config: `{"layout":{"graph":{"color":"topic","size":"citations","links":["cites"],"external":true,"labels":"auto","cluster":true}}}`},
					{Name: "Reading list", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},` + xaCards("authors", "year", "topic", "rating") + `}`},
					{Name: "All papers", Type: "table", Config: `{"sorts":[{"prop":"year","dir":"desc"}],"group":{"prop":"topic"}}`},
					{Name: "Papers per year", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"year","step":1},"series":{"prop":"topic"},"fn":"count","stacked":true,"legend":true}}}`},
					{Name: "Gallery", Type: "gallery", Config: `{"layout":{"cover":"content","size":"medium"},` + xaCards("authors", "year", "status") + `}`},
				},
				rows: paperRows,
			},
			{
				key: "rl_exp", Name: "Experiments", Icon: "i:atom:cyan", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Run", Type: "title"},
					{ID: "model", Name: "Model", Type: "select", Options: []opt{o("svm", "MFCC and SVM baseline", "gray"), o("cnn_tiny", "Tiny CNN", "yellow"), o("dscnn", "Depthwise CNN", "blue"), o("distilled", "Distilled depthwise CNN", "green"), o("cnn_small", "Small CNN", "orange"), o("trans", "Compact transformer", "purple")}},
					{ID: "f1", Name: "F1 score (%)", Type: "number", Precision: 1},
					{ID: "energy", Name: "Energy per clip (mJ)", Type: "number", Precision: 2},
					{ID: "params", Name: "Parameters (thousands)", Type: "number"},
					{ID: "dataset", Name: "Test set", Type: "select", Options: []opt{o("a", "Buoy A", "blue"), o("b", "Buoy B", "orange"), o("mixed", "Both buoys", "green")}},
					{ID: "augment", Name: "Augmented", Type: "checkbox"},
					{ID: "quantized", Name: "8-bit", Type: "checkbox"},
					{ID: "date", Name: "Run date", Type: "date"},
					{ID: "eff", Name: "F1 per millijoule", Type: "formula", Expr: `round(prop("F1 score (%)") / prop("Energy per clip (mJ)") * 10) / 10`},
				},
				views: []view{
					{Name: "Results", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"e1","title":"Best F1 score (%)","size":"s","chart":{"kind":"number","fn":"max","prop":"f1"}},{"id":"e2","title":"Runs","size":"s","chart":{"kind":"number","fn":"count"}},{"id":"e3","title":"Lowest energy per clip (mJ)","size":"s","chart":{"kind":"number","fn":"min","prop":"energy"}},{"id":"e4","title":"Runs with augmentation","size":"s","chart":{"kind":"gauge","fn":"percent_checked","prop":"augment"}},{"id":"e5","title":"F1 score of every run, by model","size":"l","chart":{"kind":"dots","x":{"prop":"model"},"prop":"f1"}},{"id":"e6","title":"Energy per clip of every run (mJ)","size":"l","chart":{"kind":"dots","x":{"prop":"model"},"prop":"energy"}},{"id":"e7","title":"Average F1 by test set","size":"m","chart":{"kind":"bar","x":{"prop":"model"},"series":{"prop":"dataset"},"fn":"average","prop":"f1","legend":true}},{"id":"e8","title":"Best F1 so far","size":"m","chart":{"kind":"line","x":{"prop":"date","by":"week"},"fn":"max","prop":"f1","smooth":true}}]}}`},
					{Name: "All runs", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"f1":"max","energy":"min"}}`},
					{Name: "Accuracy for the energy", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"model"},"fn":"average","prop":"eff","labels":true,"sort":"value_desc"}}}`},
					{Name: "Score spread", Type: "chart", Config: `{"layout":{"chart":{"kind":"histogram","x":{"prop":"f1","step":2},"fn":"count"}}}`},
				},
				rows: expRows,
			},
			{
				key: "rl_thesis", Name: "Thesis plan", Icon: "i:cap:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Task", Type: "title"},
					{ID: "dates", Name: "Dates", Type: "date"},
					{ID: "chapter", Name: "Chapter", Type: "select", Options: []opt{o("ch1", "1 · Introduction", "gray"), o("ch2", "2 · Background", "blue"), o("ch3", "3 · Data", "green"), o("ch4", "4 · Methods", "purple"), o("ch5", "5 · Experiments", "orange"), o("ch6", "6 · Deployment", "brown"), o("ch7", "7 · Conclusion", "pink"), o("paper", "Paper", "yellow"), o("milestone", "Milestone", "red"), o("all", "Whole thesis", "default")}},
					{ID: "status", Name: "Status", Type: "status", Options: xaStatus},
					{ID: "words", Name: "Words written", Type: "number", Format: "comma"},
					{ID: "target", Name: "Word target", Type: "number", Format: "comma"},
					{ID: "progress", Name: "Progress", Type: "formula", Expr: `if(prop("Word target") > 0, min(1, round(prop("Words written") / prop("Word target") * 100) / 100), 0)`, Display: "bar", Max: 1, Format: "percent"},
				},
				views: []view{
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"dates","zoom":"month","table":false}}`},
					{Name: "Chapters", Type: "table", Config: `{"sorts":[{"prop":"dates","dir":"asc"}],"calcs":{"words":"sum","target":"sum"}}`},
					{Name: "Words by chapter", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"chapter"},"fn":"sum","prop":"words","labels":true,"hide_empty":true,"filter":{"prop":"words","cond":"gt","value":0}}}}`},
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"dates"}}`},
				},
				rows: thesisRows,
			},
			{
				key: "rl_review", Name: "Literature review", Icon: "i:lightbulb:amber", Kind: "doc",
				markdown: `<!-- toc -->

## The field at a glance

` + fence + `mermaid
mindmap
  root((Whale call detection on buoys))
    Signals
      Spectrograms
      Mel features
      Ship noise
    Models
      Classic baselines
      Convolutional networks
      Compact transformers
    Making models small
      Depthwise convolutions
      Distillation
      8-bit quantization
      Pruning
    Data
      Buoy recordings
      Label noise
      Augmentation
      Self-supervised pretraining
    Deployment
      Energy budgets
      Duty cycling
      Domain shift
` + fence + `

## How we measure

A detector is judged on precision $P$ and recall $R$. We report their harmonic mean:

$$
F_1 = \frac{2PR}{P + R}
$$

Audio is mapped to the mel scale before the model sees it, which spaces frequencies the way hearing does:

$$
m = 2595 \log_{10}\left(1 + \frac{f}{700}\right)
$$

On a buoy, what matters is energy. For a model that runs $n$ times a minute, each costing $E$ millijoules, the daily budget is $1440 \cdot n \cdot E$ millijoules, which must stay under what the solar panel delivers on a cloudy day.

## Themes

| Theme | What we know | Gap our work fills |
| --- | --- | --- |
| Small models | Depthwise CNNs match larger models on one buoy | Nobody has tested across buoys and seasons |
| Augmentation | Masking and mixing help when data is scarce | Seasonal noise is not modelled |
| Distillation | Students keep most of the teacher's accuracy | Not yet measured on real buoy hardware |
| Deployment | Duty cycling saves energy | Missed calls during sleep are rarely counted |

> [!IMPORTANT]
> The gap: small models are evaluated on the buoy they were trained on. Chapter 5 tests every model on a second buoy.

## Key papers

<details>
<summary>Tiny convolutional detectors for right whale up-calls (Osei and Varga, 2021)</summary>

The closest baseline: three convolution layers, 41,000 parameters, F1 of 84% on one buoy. No cross-buoy test.

</details>

<details>
<summary>Field trial of edge detectors in the Gulf of Maine (Achterberg et al., 2024)</summary>

The only field trial with energy measured on the buoy itself. Gives the power budget we design for.

</details>

<details>
<summary>Augmenting scarce marine audio with mixing and masking (Rahimi et al., 2021)</summary>

Mixing ship noise into clean calls adds two to three F1 points. We extend this with seasonal noise.

</details>

The full reading list, with every citation link, is in {{page:rl_papers}}.
`,
				children: []Starter{
					{
						key: "rl_meetings", Name: "Advisor meetings", Icon: "i:chat:teal", Kind: "doc",
						markdown: `## This week · {{date}}

**With:** Prof. Daniel Achterberg, Dr. Lea Fontaine

### Progress

- Distilled depthwise CNN matches the small CNN within half a point at a tenth of the energy
- Buoy B results are about two points lower across every model

### Decisions

> [!TIP]
> Lead chapter 5 with the cross-buoy result. It is the clearest new finding.

- Submit the workshop paper with the distillation results
- Drop pruning experiments; distillation already answers the question

### Next steps

- [ ] Re-run the three best models with three seeds each
- [ ] Measure energy on the buoy board, not the desktop estimate
- [ ] Send the methods chapter draft by Friday

---

## Last week

### Progress

- Quantized runs lose less than half an F1 point
- Data chapter approved

### Next steps

- [x] Distillation run with the transformer as teacher
- [x] Book the committee meeting
`,
					},
				},
			},
			{
				key: "rl_protocol", Name: "Experiment protocol", Icon: "i:checklist:green", Kind: "doc",
				markdown: `Every run follows the same steps so results in {{page:rl_exp}} can be compared.

## Data splits

| Split | Source | Clips | Calls |
| --- | --- | --- | --- |
| Train | Buoy A, spring and summer | 48,000 | 6,200 |
| Validation | Buoy A, autumn | 8,000 | 1,050 |
| Test A | Buoy A, winter | 9,500 | 1,180 |
| Test B | Buoy B, all seasons | 12,000 | 1,420 |

## Steps

1. Fix the random seed and log it with the run
2. Train for at most 60 epochs with early stopping on validation F1
3. Pick the threshold on validation, never on test
4. Report F1 on Test A, Test B, or both
5. Measure energy on the buoy board with the power logger

` + fence + `sh
python train.py --model dscnn --augment seasonal --seed 7
python evaluate.py --run latest --test a,b
python energy.py --run latest --board buoy-v3 --clips 500
` + fence + `

## Before you log a run

- [ ] Seed, commit and config recorded
- [ ] Threshold chosen on validation
- [ ] Energy measured on hardware, not estimated
- [ ] Row added to the experiments database with the test set

> [!WARNING]
> Never tune on Test B. It is the only honest measure of how a model travels to a new buoy.
`,
			},
		},
	}
}
