package templates

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type chartRng struct{ s uint64 }

func chartSeed(seed uint64) *chartRng { return &chartRng{s: seed*0x9E3779B97F4A7C15 | 1} }

func (r *chartRng) f() float64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return float64(r.s>>11) / float64(uint64(1)<<53)
}

func (r *chartRng) span(lo, hi float64) float64 { return lo + r.f()*(hi-lo) }

func (r *chartRng) num(lo, hi int) int { return lo + int(r.f()*float64(hi-lo+1)) }

func (r *chartRng) chance(p float64) bool { return r.f() < p }

func (r *chartRng) norm() float64 { return (r.f() + r.f() + r.f() + r.f() - 2) * 1.732 }

func (r *chartRng) weigh(ids []string, ws []float64) string {
	total := 0.0
	for _, w := range ws {
		total += w
	}
	x := r.f() * total
	for i, w := range ws {
		if x < w {
			return ids[i]
		}
		x -= w
	}
	return ids[len(ids)-1]
}

var chartToday = func() time.Time {
	n := time.Now().UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}()

func chartAgo(d int) string {
	if d < 0 {
		return fmt.Sprintf("+%dd", -d)
	}
	return fmt.Sprintf("-%dd", d)
}

func chartDay(d int) time.Time { return chartToday.AddDate(0, 0, -d) }

func chartBack(t time.Time) int { return int(math.Round(chartToday.Sub(t).Hours() / 24)) }

func chartMonth(k int) (first, last int, start time.Time) {
	start = time.Date(chartToday.Year(), chartToday.Month()-time.Month(k), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	if end.After(chartToday) {
		end = chartToday
	}
	return chartBack(start), chartBack(end), start
}

func chartR(x float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(x*p) / p
}

func chartClamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

func chartPlace(name string, lat, lng float64) map[string]any {
	return map[string]any{"lat": lat, "lng": lng, "name": name}
}

func chartIDs(opts []opt) []string {
	out := make([]string, len(opts))
	for i, x := range opts {
		out[i] = x.ID
	}
	return out
}

func chartNames(a, b []string, n int) []string {
	out := make([]string, 0, n)
	total := len(a) * len(b)
	for i := 0; len(out) < n && i < total; i++ {
		k := (i * 37) % total
		out = append(out, a[k%len(a)]+" "+b[k/len(a)])
	}
	return out
}

var chartStarters = []Starter{
	chartSales(),
	chartAnalytics(),
	chartPortfolio(),
	chartFinance(),
	chartSurvey(),
	chartInventory(),
	chartCapacity(),
	chartCustomers(),
	chartFitness(),
	chartHabits(),
	chartKnowledge(),
	chartFormulas(),
	chartLibrary(),
	chartReport(),
	chartDiagrams(),
}

func chartSales() Starter {
	r := chartSeed(101)
	products := []opt{o("backpack", "Trail backpack", "green"), o("shell", "Rain shell", "blue"), o("stove", "Camp stove", "orange"), o("jacket", "Down jacket", "purple"), o("boots", "Hiking boots", "brown"), o("filter", "Water filter", "gray"), o("headlamp", "Headlamp", "yellow"), o("tent", "Two-person tent", "red")}
	price := map[string]float64{"backpack": 129, "shell": 189, "stove": 89, "jacket": 249, "boots": 169, "filter": 49, "headlamp": 39, "tent": 329}
	regions := []opt{o("na", "North America", "blue"), o("eu", "Europe", "purple"), o("apac", "Asia Pacific", "green"), o("latam", "Latin America", "orange")}
	channels := []opt{o("portal", "Online portal", "blue"), o("rep", "Sales rep", "green"), o("show", "Trade show", "yellow")}
	type city struct {
		name     string
		lat, lng float64
	}
	cities := map[string][]city{
		"na":    {{"Seattle", 47.6062, -122.3321}, {"Denver", 39.7392, -104.9903}, {"Toronto", 43.6532, -79.3832}, {"Chicago", 41.8781, -87.6298}, {"Austin", 30.2672, -97.7431}},
		"eu":    {{"Munich", 48.1351, 11.582}, {"Lyon", 45.764, 4.8357}, {"Edinburgh", 55.9533, -3.1883}, {"Oslo", 59.9139, 10.7522}, {"Zurich", 47.3769, 8.5417}},
		"apac":  {{"Sydney", -33.8688, 151.2093}, {"Tokyo", 35.6762, 139.6503}, {"Auckland", -36.8485, 174.7633}, {"Seoul", 37.5665, 126.978}, {"Singapore", 1.3521, 103.8198}},
		"latam": {{"Santiago", -33.4489, -70.6693}, {"Mexico City", 19.4326, -99.1332}, {"Bogotá", 4.711, -74.0721}, {"São Paulo", -23.5505, -46.6333}},
	}
	stores := []string{"Summit Supply", "Ridgeline Outfitters", "Fjord and Pine", "Switchback Co-op", "Tidewater Sports", "Alpenglow Gear", "Northstar Trading", "Cedar Trail Shop", "Kestrel Outdoor", "Bluebird Mercantile", "Granite Peak Store", "Harbour Line Goods"}
	season := map[time.Month]float64{1: .85, 2: .8, 3: .95, 4: 1, 5: 1.05, 6: 1, 7: .76, 8: .9, 9: 1.08, 10: 1.15, 11: 1.35, 12: 1.55}
	prodIDs := chartIDs(products)
	var rows []row
	n := 1001
	target := 0.0
	for k := 11; k >= 0; k-- {
		first, last, start := chartMonth(k)
		goal := 42000 * math.Pow(1.03, float64(11-k)) * season[start.Month()]
		if k == 0 {
			target = math.Round(goal/1000) * 1000
			goal *= 0.85
		}
		type ord struct {
			d            int
			w            float64
			ch, prod, rg string
		}
		var list []ord
		sum := 0.0
		t := float64(11 - k)
		count := int(math.Round(5 * season[start.Month()] * math.Pow(1.025, t)))
		if k == 0 {
			count = max(3, count-2)
		}
		for i := 0; i < count; i++ {
			pw := []float64{1, 1, 1, 1, 1, .8, .8, 1}
			switch start.Month() {
			case 11, 12, 1, 2:
				pw[3], pw[4] = 2.2, 1.8
			case 5, 6, 7, 8:
				pw[2], pw[7] = 2, 2.2
			}
			ch := r.weigh([]string{"portal", "rep", "show"}, []float64{.55, .35, .1})
			w := r.span(.6, 1.4)
			if ch == "rep" {
				w *= 1.8
			}
			list = append(list, ord{d: r.num(last, first), w: w, ch: ch, prod: r.weigh(prodIDs, pw), rg: r.weigh([]string{"na", "eu", "apac", "latam"}, []float64{.42 - .012*t, .3, .12 + .02*t, .1})})
			sum += w
		}
		sort.SliceStable(list, func(a, b int) bool { return list[a].d > list[b].d })
		for _, x := range list {
			units := math.Max(1, math.Round(goal*x.w/sum/price[x.prod]))
			c := cities[x.rg][r.num(0, len(cities[x.rg])-1)]
			status := "paid"
			if r.chance(.05) {
				status = "refunded"
			}
			rows = append(rows, row{Title: fmt.Sprintf("Order %d", n), Values: map[string]any{"date": chartAgo(x.d), "customer": stores[r.num(0, len(stores)-1)], "product": x.prod, "region": x.rg, "channel": x.ch, "units": units, "revenue": units * price[x.prod], "city": chartPlace(c.name, c.lat, c.lng), "status": status}})
			n++
		}
	}
	paid := `{"prop":"status","cond":"is_not","value":["refunded"]}`
	dash := `{"layout":{"widgets":[` +
		`{"id":"rev","title":"Revenue","kpi":{"date":"date","by":"month"},"chart":{"kind":"number","fn":"sum","prop":"revenue","filter":` + paid + `}},` +
		`{"id":"orders","size":"s","title":"Orders","kpi":{"date":"date","by":"month"},"chart":{"kind":"number","fn":"count"}},` +
		fmt.Sprintf(`{"id":"goal","size":"s","title":"This month against target","chart":{"kind":"gauge","fn":"sum","prop":"revenue","target":%.0f,"filter":{"op":"and","items":[{"prop":"date","cond":"within","value":{"within":"this_month"}},`+paid+`]}}},`, target) +
		`{"id":"trend","wide":true,"title":"Revenue by month","chart":{"kind":"line","x":{"prop":"date","by":"month"},"fn":"sum","prop":"revenue","smooth":true,"labels":true,"filter":` + paid + `}},` +
		`{"id":"regions","title":"Revenue by region","chart":{"kind":"donut","x":{"prop":"region"},"fn":"sum","prop":"revenue","labels":true,"filter":` + paid + `}},` +
		`{"id":"top","title":"Top products","chart":{"kind":"hbar","x":{"prop":"product"},"fn":"sum","prop":"revenue","sort":"value_desc","labels":true,"filter":` + paid + `}},` +
		`{"id":"stack","wide":true,"title":"Monthly revenue by region","chart":{"kind":"area","x":{"prop":"date","by":"month"},"series":{"prop":"region"},"fn":"sum","prop":"revenue","stacked":true,"smooth":true,"legend":true,"filter":` + paid + `}},` +
		`{"id":"half","wide":true,"title":"Last six months against the six before","chart":{"kind":"compare","x":{"prop":"date","by":"month"},"fn":"sum","prop":"revenue","window":6,"smooth":true,"filter":` + paid + `}}` +
		`]}}`
	return Starter{
		ID: "sales-dashboard", Name: "Sales dashboard", Description: "A year of wholesale orders on one dashboard: revenue KPIs with trends, a monthly line, regions as a stacked area, a donut and a map, top products and a target gauge.", Icon: "i:chart-line:green", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"revenue", "kpi", "dashboard", "map", "gauge"},
		schema: []prop{
			{ID: "title", Name: "Order", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "customer", Name: "Customer", Type: "text"},
			{ID: "product", Name: "Product", Type: "select", Options: products},
			{ID: "region", Name: "Region", Type: "select", Options: regions},
			{ID: "channel", Name: "Channel", Type: "select", Options: channels},
			{ID: "units", Name: "Units", Type: "number", Display: "bar", Max: 150},
			{ID: "revenue", Name: "Revenue", Type: "number", Format: "dollar", Precision: -1},
			{ID: "city", Name: "Store", Type: "place"},
			{ID: "status", Name: "Payment", Type: "select", Options: []opt{o("paid", "Paid", "green"), o("refunded", "Refunded", "red")}},
		},
		views: []view{
			{Name: "Dashboard", Type: "dashboard", Config: dash},
			{Name: "Orders", Type: "table", Config: `{"group":{"prop":"date","by":"month","sort":"desc","hide_empty":true},"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"title":"count_all","units":"sum","revenue":"sum"}}`},
			{Name: "Store map", Type: "map", Config: `{"layout":{"place":"city"}}`},
			{Name: "By region", Type: "board", Config: `{"group":{"prop":"region","hide_empty":true},"sorts":[{"prop":"revenue","dir":"desc"}],"props":[{"id":"revenue","hidden":false},{"id":"product","hidden":false},{"id":"date","hidden":false}],"layout":{"total":{"prop":"revenue","fn":"sum"}}}`},
			{Name: "Channels", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"month"},"series":{"prop":"channel"},"fn":"sum","prop":"revenue","stacked":true,"legend":true,"filter":` + paid + `}}}`},
		},
		rows: rows,
	}
}

func chartAnalytics() Starter {
	r := chartSeed(202)
	var traffic []row
	for d := 84; d >= 1; d-- {
		t := float64(84 - d)
		base := 1150 + t*11
		switch chartDay(d).Weekday() {
		case time.Saturday:
			base *= .62
		case time.Sunday:
			base *= .58
		case time.Friday:
			base *= .9
		case time.Monday:
			base *= 1.05
		}
		conv := r.span(.019, .025)
		if d <= 40 {
			conv = r.span(.027, .034)
		}
		bounce := r.span(.36, .47)
		switch d {
		case 30:
			base *= 2.3
			bounce = .55
		case 29:
			base *= 1.6
		case 52:
			base *= .35
			bounce = .71
		}
		visits := math.Round(base * r.span(.92, 1.08))
		traffic = append(traffic, row{Title: chartDay(d).Format("Mon Jan 2"), Values: map[string]any{"date": chartAgo(d), "visits": visits, "signups": math.Round(visits * conv), "bounce": chartR(bounce, 3), "minutes": chartR(r.span(2.1, 3.4), 1)}})
	}
	sources := []opt{o("search", "Search", "blue"), o("social", "Social", "pink"), o("referral", "Referral", "green"), o("email", "Email", "orange"), o("direct", "Direct", "gray"), o("ads", "Ads", "yellow")}
	plans := []opt{o("free", "Free", "gray"), o("team", "Team", "blue"), o("business", "Business", "purple")}
	names := chartNames([]string{"Bright", "Copper", "Quiet", "Swift", "Maple", "Harbor", "Lumen", "Orchid", "Pebble", "Juniper", "Atlas", "Cobalt"}, []string{"Studio", "Labs", "Bakery", "Clinic", "Collective", "Works", "Partners", "Press", "Foods", "Design"}, 60)
	type su struct {
		d    int
		name string
	}
	var list []su
	for i := 0; i < 60; i++ {
		list = append(list, su{d: int(83 * math.Pow(r.f(), 1.35)), name: names[i]})
	}
	sort.SliceStable(list, func(a, b int) bool { return list[a].d > list[b].d })
	var signups []row
	for _, x := range list {
		social := .18
		if x.d <= 31 {
			social = .32
		}
		src := r.weigh(chartIDs(sources), []float64{.32, social, .15, .1, .17, .08})
		plan := r.weigh(chartIDs(plans), []float64{.55, .33, .12})
		seats := 1
		switch plan {
		case "team":
			seats = r.num(3, 12)
		case "business":
			seats = r.num(15, 60)
		}
		signups = append(signups, row{Title: x.name, Values: map[string]any{"date": chartAgo(x.d), "source": src, "plan": plan, "seats": seats, "active": x.d > 3 && r.chance(.7)}})
	}
	dash := `{"layout":{"widgets":[` +
		`{"id":"visits","title":"Visits","kpi":{"date":"date","by":"day"},"chart":{"kind":"number","fn":"sum","prop":"visits"}},` +
		`{"id":"signups","title":"Sign-ups","kpi":{"date":"date","by":"day"},"chart":{"kind":"number","fn":"sum","prop":"signups"}},` +
		`{"id":"bounce","size":"s","title":"Average bounce rate","chart":{"kind":"number","fn":"average","prop":"bounce"}},` +
		`{"id":"conv","size":"s","title":"Average conversion, %","chart":{"kind":"number","fn":"average","prop":"conv"}},` +
		`{"id":"wd","title":"Average visits by weekday","chart":{"kind":"bar","x":{"prop":"date","by":"weekday"},"fn":"average","prop":"visits","labels":true}},` +
		`{"id":"cal","wide":true,"title":"Visits per day","chart":{"kind":"calendar","x":{"prop":"date","by":"day"},"fn":"sum","prop":"visits","scale":"quantile"}},` +
		`{"id":"cmp","wide":true,"title":"Last 28 days against the 28 before","chart":{"kind":"compare","x":{"prop":"date","by":"day"},"fn":"sum","prop":"visits","window":28}},` +
		`{"id":"hist","wide":true,"title":"How busy are our days?","chart":{"kind":"histogram","x":{"prop":"visits","step":250},"fn":"count"}}` +
		`]}}`
	return Starter{
		ID: "website-analytics", Name: "Website analytics", Description: "Twelve weeks of daily traffic and sign-ups: a calendar heatmap, period comparison, histogram, weekday pattern and a source donut.", Icon: "i:globe:blue", Kind: "doc", Category: catCharts, Featured: true, Tags: []string{"traffic", "marketing", "heatmap", "conversion"},
		markdown: `A small website's last twelve weeks. Traffic dips every weekend, jumps on launch day about a month ago and drops for one day during an outage. A new landing page lifted conversion around six weeks ago.

> [!TIP]
> Each chart is a saved view. Open **Daily traffic** or **Sign-ups** and switch between the view tabs to see the same rows as a calendar heatmap, a histogram or a period comparison.

## Traffic

{{db:traffic}}

## Sign-ups

{{db:signups}}

## What to look for

| Chart | What it shows |
| --- | --- |
| Visits per day | A calendar heatmap: darker days had more visits, weekends are lighter |
| Last 28 days against the 28 before | A period comparison with the change in percent |
| How busy are our days? | A histogram of days by visit count, with a cumulative line |
| Sources | Where sign-ups came from, as a donut |
| Plans by week | Free, Team and Business sign-ups stacked per week |

<details>
<summary>How conversion is worked out</summary>

Conversion is a formula on each day: $\text{conversion} = \frac{\text{sign-ups}}{\text{visits}} \times 100$, rounded to one decimal place. The dashboard shows its average.

</details>
`,
		children: []Starter{
			{
				key: "traffic", Name: "Daily traffic", Icon: "i:chart-line:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Day", Type: "title"},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "visits", Name: "Visits", Type: "number", Format: "comma", Precision: -1},
					{ID: "signups", Name: "Sign-ups", Type: "number"},
					{ID: "conv", Name: "Conversion, %", Type: "formula", Expr: `round(prop("Sign-ups") / prop("Visits") * 100, 1)`, Display: "bar", Max: 5},
					{ID: "bounce", Name: "Bounce rate", Type: "number", Format: "percent", Precision: 1, Display: "ring", Max: 1},
					{ID: "minutes", Name: "Minutes on site", Type: "number", Precision: 1},
				},
				views: []view{
					{Name: "Overview", Type: "dashboard", Config: dash},
					{Name: "Daily log", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"visits":"sum","signups":"sum","conv":"average","bounce":"average","minutes":"median"}}`},
					{Name: "Heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":4,"fn":"sum","value":"visits","color":"teal"}}}`},
					{Name: "Weekly visits", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"week"},"fn":"sum","prop":"visits","labels":true}}}`},
				},
				rows: traffic,
			},
			{
				key: "signups", Name: "Sign-ups", Icon: "i:users:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Account", Type: "title"},
					{ID: "date", Name: "Signed up", Type: "date"},
					{ID: "source", Name: "Source", Type: "select", Options: sources},
					{ID: "plan", Name: "Plan", Type: "select", Options: plans},
					{ID: "seats", Name: "Seats", Type: "number"},
					{ID: "active", Name: "Active this week", Type: "checkbox"},
				},
				views: []view{
					{Name: "Sources", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"source"},"fn":"count","labels":true}}}`},
					{Name: "Plans by week", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"week"},"series":{"prop":"plan"},"fn":"count","stacked":true,"legend":true}}}`},
					{Name: "Seats by source", Type: "chart", Config: `{"layout":{"chart":{"kind":"radial","x":{"prop":"source"},"fn":"sum","prop":"seats","labels":true}}}`},
					{Name: "All sign-ups", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"title":"count_all","seats":"sum","active":"percent_checked"}}`},
				},
				rows: signups,
			},
		},
	}
}

func chartPortfolio() Starter {
	leads := []opt{o("maya", "Maya Chen", "blue"), o("tomas", "Tomás Silva", "orange"), o("ines", "Ines Laurent", "purple"), o("kwame", "Kwame Mensah", "green"), o("hana", "Hana Sato", "pink"), o("lea", "Lea Novak", "yellow")}
	type proj struct {
		name, status, team, lead, health string
		start, end, progress             int
		budget, burn                     float64
	}
	projects := []proj{
		{"Customer portal redesign", "active", "product", "maya", "on", 75, -40, 58, 120000, .95},
		{"Warehouse scanner rollout", "active", "ops", "tomas", "risk", 40, -25, 70, 85000, 1.1},
		{"Mobile app 3.0", "active", "eng", "ines", "risk", 60, -75, 35, 210000, 1.25},
		{"Data warehouse migration", "review", "eng", "kwame", "on", 120, 5, 92, 160000, .9},
		{"Self-serve billing", "planned", "product", "maya", "on", -15, -110, 0, 95000, 0},
		{"Support chatbot pilot", "active", "product", "ines", "off", 20, -30, 25, 40000, 1.45},
		{"Office move to Pier 9", "done", "ops", "tomas", "on", 150, 60, 100, 70000, .97},
		{"Security audit", "active", "eng", "kwame", "on", 30, -20, 50, 55000, .9},
		{"Partner API", "planned", "eng", "hana", "on", -30, -140, 0, 130000, 0},
		{"Brand refresh", "review", "marketing", "lea", "on", 90, -5, 85, 65000, 1},
		{"Onboarding emails", "done", "marketing", "lea", "on", 110, 70, 100, 15000, .85},
		{"Accessibility fixes", "active", "eng", "hana", "risk", 45, -15, 66, 30000, 1.05},
	}
	var prow, spend []row
	for _, p := range projects {
		prow = append(prow, row{Title: p.name, Values: map[string]any{"status": p.status, "team": p.team, "lead": p.lead, "health": p.health, "when": map[string]any{"start": chartAgo(p.start), "end": chartAgo(p.end)}, "progress": p.progress, "budget": p.budget}})
		spend = append(spend, row{Title: "Budget · " + p.name, Values: map[string]any{"project": p.name, "kind": "budget", "amount": p.budget, "month": chartAgo(p.start)}})
		if p.progress == 0 {
			continue
		}
		spent := p.budget * float64(p.progress) / 100 * p.burn
		lastDay := max(p.end, 3)
		parts := 3
		if p.start-lastDay < 45 {
			parts = 2
		}
		for i := 0; i < parts; i++ {
			d := p.start - (p.start-lastDay)*(i+1)/parts
			share := []float64{.25, .35, .4}[i]
			if parts == 2 {
				share = []float64{.45, .55}[i]
			}
			spend = append(spend, row{Title: fmt.Sprintf("Actual · %s · %s", p.name, chartDay(d).Format("Jan")), Values: map[string]any{"project": p.name, "kind": "actual", "amount": math.Round(spent*share/100) * 100, "month": chartAgo(d)}})
		}
	}
	type risk struct {
		title, project, status, owner string
		likelihood, impact            int
	}
	risks := []risk{
		{"App store review rejects the new sign-in", "Mobile app 3.0", "open", "Ines Laurent", 3, 4},
		{"Offline mode needs a rewrite of sync", "Mobile app 3.0", "mitigating", "Ines Laurent", 4, 5},
		{"Two senior engineers on leave in December", "Mobile app 3.0", "open", "Kwame Mensah", 4, 3},
		{"Scanner batteries last less than a shift", "Warehouse scanner rollout", "mitigating", "Tomás Silva", 4, 4},
		{"Wi-Fi dead zones in aisle 14", "Warehouse scanner rollout", "open", "Tomás Silva", 3, 3},
		{"Night shift not trained before go-live", "Warehouse scanner rollout", "closed", "Tomás Silva", 2, 3},
		{"Old reports read from the legacy database", "Data warehouse migration", "mitigating", "Kwame Mensah", 3, 4},
		{"Row counts differ after the final sync", "Data warehouse migration", "closed", "Kwame Mensah", 2, 5},
		{"Chatbot gives wrong refund answers", "Support chatbot pilot", "open", "Ines Laurent", 4, 5},
		{"Support team does not trust the handover", "Support chatbot pilot", "open", "Maya Chen", 3, 3},
		{"Portal search is slower than the old one", "Customer portal redesign", "mitigating", "Maya Chen", 3, 3},
		{"Single sign-on partner delays their release", "Customer portal redesign", "open", "Maya Chen", 2, 4},
		{"Pen test finds a critical issue late", "Security audit", "open", "Kwame Mensah", 2, 5},
		{"Vendor questionnaire backlog", "Security audit", "closed", "Hana Sato", 3, 2},
		{"Screen reader testing needs outside help", "Accessibility fixes", "mitigating", "Hana Sato", 3, 3},
		{"Colour changes clash with the brand refresh", "Accessibility fixes", "open", "Lea Novak", 2, 2},
		{"New logo trademark not cleared", "Brand refresh", "mitigating", "Lea Novak", 2, 4},
		{"Printed materials reordered too early", "Brand refresh", "closed", "Lea Novak", 1, 2},
		{"Partners want webhooks on day one", "Partner API", "open", "Hana Sato", 3, 3},
		{"Tax rules differ in three new countries", "Self-serve billing", "open", "Maya Chen", 4, 4},
	}
	var rrows []row
	for _, x := range risks {
		rrows = append(rrows, row{Title: x.title, Values: map[string]any{"project": x.project, "status": x.status, "owner": x.owner, "likelihood": x.likelihood, "impact": x.impact}})
	}
	return Starter{
		ID: "project-portfolio", Name: "Project portfolio", Description: "Twelve projects on a timeline and a status board with progress meters and budget totals, a risk matrix heatmap, risk dots, a link graph and budget against actual spend.", Icon: "i:briefcase:violet", Kind: "doc", Category: catCharts, Featured: true, Tags: []string{"projects", "timeline", "risks", "budget", "graph"},
		markdown: `<!-- toc -->

Every project the company is running this year, with its risks and its money. The three databases below are linked: each risk and each spend line points at a project.

## Timeline

{{db:projects}}

> [!NOTE]
> Open **Projects** and switch to **Status board** to see progress meters on each column and the budget total at the bottom.

## Risks

Risks are scored as likelihood times impact:

$$\text{score} = \text{likelihood} \times \text{impact}, \quad 1 \le \text{score} \le 25$$

{{db:risks}}

## Budget against actual

{{db:spend}}

> [!WARNING]
> **Mobile app 3.0** and **Support chatbot pilot** are spending faster than they progress. Both are on the agenda for the next steering meeting.
`,
		children: []Starter{
			{
				key: "projects", Name: "Projects", Icon: "i:rocket:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Project", Type: "title"},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("planned", "Planned", "gray", "todo"), so("active", "In progress", "blue", "doing"), so("review", "In review", "purple", "doing"), so("done", "Done", "green", "done")}},
					{ID: "team", Name: "Team", Type: "select", Options: []opt{o("product", "Product", "blue"), o("eng", "Engineering", "purple"), o("ops", "Operations", "orange"), o("marketing", "Marketing", "pink")}},
					{ID: "lead", Name: "Lead", Type: "select", Options: leads},
					{ID: "when", Name: "Dates", Type: "date"},
					{ID: "progress", Name: "Progress", Type: "number", Display: "bar", Max: 100},
					{ID: "budget", Name: "Budget", Type: "number", Format: "dollar", Precision: -1},
					{ID: "health", Name: "Health", Type: "select", Options: []opt{o("on", "On track", "green"), o("risk", "At risk", "yellow"), o("off", "Off track", "red")}},
				},
				views: []view{
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"when","zoom":"month"},"props":[{"id":"lead","hidden":false},{"id":"progress","hidden":false}]}`},
					{Name: "Status board", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},"props":[{"id":"progress","hidden":false},{"id":"lead","hidden":false},{"id":"health","hidden":false},{"id":"budget","hidden":false}],"layout":{"progress":"progress","total":{"prop":"budget","fn":"sum"}}}`},
					{Name: "By team", Type: "table", Config: `{"group":{"prop":"team","hide_empty":true},"calcs":{"budget":"sum","progress":"average","title":"count_all"}}`},
					{Name: "Health", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"team"},"series":{"prop":"health"},"fn":"count","stacked":true,"legend":true,"labels":true}}}`},
				},
				rows: prow,
			},
			{
				key: "risks", Name: "Risks", Icon: "i:shield:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Risk", Type: "title"},
					{ID: "project", Name: "Project", Type: "relation", Collection: "@projects"},
					{ID: "likelihood", Name: "Likelihood", Type: "number", Display: "bar", Max: 5},
					{ID: "impact", Name: "Impact", Type: "number", Display: "bar", Max: 5},
					{ID: "score", Name: "Score", Type: "formula", Expr: `prop("Likelihood") * prop("Impact")`, Display: "ring", Max: 25},
					{ID: "lband", Name: "Likelihood level", Type: "formula", Expr: `if(prop("Likelihood") >= 5, "5 Almost certain", if(prop("Likelihood") >= 4, "4 Likely", if(prop("Likelihood") >= 3, "3 Possible", if(prop("Likelihood") >= 2, "2 Unlikely", "1 Rare"))))`},
					{ID: "iband", Name: "Impact level", Type: "formula", Expr: `if(prop("Impact") >= 5, "5 Severe", if(prop("Impact") >= 4, "4 Major", if(prop("Impact") >= 3, "3 Moderate", if(prop("Impact") >= 2, "2 Minor", "1 Slight"))))`},
					{ID: "status", Name: "Status", Type: "select", Options: []opt{o("open", "Open", "red"), o("mitigating", "Mitigating", "yellow"), o("closed", "Closed", "green")}},
					{ID: "owner", Name: "Owner", Type: "text"},
				},
				views: []view{
					{Name: "Risk matrix", Type: "chart", Config: `{"layout":{"chart":{"kind":"heatmap","x":{"prop":"lband"},"series":{"prop":"iband","sort":"desc"},"fn":"count","labels":true}}}`},
					{Name: "Scores by project", Type: "chart", Config: `{"layout":{"chart":{"kind":"dots","x":{"prop":"project"},"prop":"score"}}}`},
					{Name: "Register", Type: "table", Config: `{"sorts":[{"prop":"score","dir":"desc"}],"calcs":{"score":"average","title":"count_all"}}`},
					{Name: "Links", Type: "graph", Config: `{"layout":{"graph":{"color":"status","labels":"auto"}}}`},
					{Name: "By status", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},"props":[{"id":"score","hidden":false},{"id":"project","hidden":false}]}`},
				},
				rows: rrows,
			},
			{
				key: "spend", Name: "Budget and spend", Icon: "i:coin:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Line", Type: "title"},
					{ID: "project", Name: "Project", Type: "relation", Collection: "@projects"},
					{ID: "kind", Name: "Type", Type: "select", Options: []opt{o("budget", "Budget", "gray"), o("actual", "Actual", "blue")}},
					{ID: "amount", Name: "Amount", Type: "number", Format: "dollar", Precision: -1},
					{ID: "month", Name: "Month", Type: "date"},
				},
				views: []view{
					{Name: "Budget against actual", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"project"},"series":{"prop":"kind"},"fn":"sum","prop":"amount","legend":true}}}`},
					{Name: "Spend to date", Type: "chart", Config: `{"layout":{"chart":{"kind":"line","x":{"prop":"month","by":"month"},"fn":"sum","prop":"amount","cumulative":true,"labels":true,"filter":{"prop":"kind","cond":"is","value":["actual"]}}}}`},
					{Name: "All lines", Type: "table", Config: `{"group":{"prop":"kind","hide_empty":true},"sorts":[{"prop":"month","dir":"asc"}],"calcs":{"amount":"sum"}}`},
				},
				rows: spend,
			},
		},
	}
}

func chartFinance() Starter {
	r := chartSeed(303)
	cats := []opt{o("salary", "Salary", "green"), o("freelance", "Freelance", "blue"), o("rent", "Rent", "purple"), o("utilities", "Utilities", "gray"), o("groceries", "Groceries", "orange"), o("transport", "Transport", "yellow"), o("dining", "Eating out", "pink"), o("fun", "Fun", "red"), o("travel", "Travel", "brown")}
	shops := []string{"Green Basket market", "Corner grocer", "Weekly shop", "Farmers market", "Bulk store"}
	dining := []string{"Lunch with Sam", "Pizza night", "Coffee and cake", "Dinner at Casa Lupe", "Noodle bar", "Brunch"}
	fun := []string{"Cinema tickets", "Concert", "Board game", "Climbing gym", "Museum and lunch", "Books"}
	var rows []row
	add := func(title string, d int, cat string, amount float64, account string) {
		typ := "expense"
		if cat == "salary" || cat == "freelance" {
			typ = "income"
		}
		rows = append(rows, row{Title: title, Values: map[string]any{"date": chartAgo(d), "type": typ, "category": cat, "amount": chartR(amount, 2), "account": account}})
	}
	for k := 5; k >= 0; k-- {
		first, last, start := chartMonth(k)
		at := func(day int) int { return chartBack(start.AddDate(0, 0, day-1)) }
		in := func(day int) bool { d := at(day); return d >= last && d <= first }
		mon := start.Format("January")
		if in(1) {
			salary := 5200.0
			if k <= 2 {
				salary = 5450
			}
			add("Salary · "+mon, at(1), "salary", salary, "checking")
		}
		if in(2) {
			add("Rent · "+mon, at(2), "rent", 1850, "checking")
		}
		if in(6) {
			u := 140.0
			switch start.Month() {
			case 12, 1, 2:
				u = 225
			case 6, 7, 8:
				u = 165
			}
			add("Power and water", at(6), "utilities", u+r.span(-12, 12), "checking")
		}
		for i := 0; i < r.num(0, 2); i++ {
			if day := r.num(8, 26); in(day) {
				add([]string{"Logo design job", "Website fixes", "Photo shoot", "Workshop fee"}[r.num(0, 3)], at(day), "freelance", math.Round(r.span(400, 1500)/50)*50, "checking")
			}
		}
		for _, day := range []int{4, 11, 18, 25} {
			if in(day) {
				add(shops[r.num(0, len(shops)-1)], at(day), "groceries", r.span(85, 160), "card")
			}
		}
		for _, day := range []int{3, 17} {
			if in(day) {
				add([]string{"Train pass top-up", "Fuel", "Bike repair", "Taxi home"}[r.num(0, 3)], at(day), "transport", r.span(40, 90), "card")
			}
		}
		for i := 0; i < r.num(2, 3); i++ {
			if day := r.num(5, 28); in(day) {
				amt := r.span(25, 90)
				if k == 3 && i == 0 {
					amt = 210
				}
				add(dining[r.num(0, len(dining)-1)], at(day), "dining", amt, "card")
			}
		}
		for i := 0; i < r.num(1, 2); i++ {
			if day := r.num(6, 27); in(day) {
				add(fun[r.num(0, len(fun)-1)], at(day), "fun", r.span(20, 120), "card")
			}
		}
		if k == 2 && in(14) {
			add("Train tickets and hotel, coast weekend", at(14), "travel", 920, "card")
		}
	}
	income := `{"prop":"type","cond":"is","value":["income"]}`
	expense := `{"prop":"type","cond":"is","value":["expense"]}`
	dash := `{"layout":{"widgets":[` +
		`{"id":"in","title":"Income","kpi":{"date":"date","by":"month"},"chart":{"kind":"number","fn":"sum","prop":"amount","filter":` + income + `}},` +
		`{"id":"out","title":"Spending","kpi":{"date":"date","by":"month"},"chart":{"kind":"number","fn":"sum","prop":"amount","filter":` + expense + `}},` +
		`{"id":"net","size":"s","title":"Saved over six months","chart":{"kind":"number","fn":"sum","prop":"net"}},` +
		`{"id":"count","size":"s","title":"Transactions","chart":{"kind":"number","fn":"count"}},` +
		`{"id":"io","title":"Income and spending","chart":{"kind":"bar","x":{"prop":"date","by":"month"},"series":{"prop":"type"},"fn":"sum","prop":"amount","legend":true}},` +
		`{"id":"save","wide":true,"title":"Savings over time","chart":{"kind":"line","x":{"prop":"date","by":"month"},"fn":"sum","prop":"net","cumulative":true,"labels":true,"smooth":true}},` +
		`{"id":"where","title":"Where the money goes","chart":{"kind":"donut","x":{"prop":"category"},"fn":"sum","prop":"amount","labels":true,"hide_empty":true,"filter":` + expense + `}},` +
		`{"id":"cats","title":"Everyday spending, without rent and bills","chart":{"kind":"area","x":{"prop":"date","by":"month"},"series":{"prop":"category"},"fn":"sum","prop":"amount","stacked":true,"smooth":true,"legend":true,"filter":{"op":"and","items":[` + expense + `,{"prop":"category","cond":"is_not","value":["rent","utilities"]}]}}}` +
		`]}}`
	return Starter{
		ID: "finance-overview", Name: "Finance overview", Description: "Six months of household money: income and spending KPIs, a cumulative savings line, a stacked area by category, a donut and a calendar of transactions.", Icon: "i:wallet:green", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"budget", "money", "savings", "dashboard"},
		schema: []prop{
			{ID: "title", Name: "Description", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "type", Name: "Type", Type: "select", Options: []opt{o("income", "Income", "green"), o("expense", "Expense", "red")}},
			{ID: "category", Name: "Category", Type: "select", Options: cats},
			{ID: "amount", Name: "Amount", Type: "number", Format: "dollar", Precision: 2},
			{ID: "net", Name: "Net", Type: "formula", Expr: `if(prop("Type") == "Income", prop("Amount"), -prop("Amount"))`, Format: "dollar", Precision: 2},
			{ID: "account", Name: "Account", Type: "select", Options: []opt{o("checking", "Checking", "blue"), o("card", "Credit card", "purple")}},
		},
		views: []view{
			{Name: "Overview", Type: "dashboard", Config: dash},
			{Name: "Transactions", Type: "table", Config: `{"group":{"prop":"date","by":"month","sort":"desc","hide_empty":true},"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"title":"count_all","net":"sum"}}`},
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"},"props":[{"id":"amount","hidden":false}]}`},
			{Name: "Biggest costs", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"category"},"fn":"sum","prop":"amount","sort":"value_desc","labels":true,"hide_empty":true,"filter":` + expense + `}}}`},
		},
		rows: rows,
	}
}

func chartSurvey() Starter {
	r := chartSeed(404)
	teams := []opt{o("eng", "Engineering", "purple"), o("design", "Design", "pink"), o("sales", "Sales", "green"), o("support", "Support", "orange"), o("ops", "Operations", "blue")}
	perks := []opt{o("flex", "Flexible hours", "green"), o("remote", "Remote days", "blue"), o("learning", "Learning budget", "purple"), o("health", "Health plan", "red"), o("equipment", "Good equipment", "gray"), o("offsites", "Team offsites", "yellow")}
	counts := map[string]int{"eng": 14, "design": 8, "sales": 12, "support": 14, "ops": 12}
	mean := map[string]float64{"eng": 8.6, "design": 9.2, "sales": 8.1, "ops": 7.6, "support": 6}
	praise := []string{"My manager really listens.", "Love the flexible hours.", "Great people, interesting work.", "The learning budget made a real difference this year.", "Best team I have worked in."}
	neutral := []string{"Mostly good, meetings could be shorter.", "Would like clearer goals for next quarter.", "Tools are fine, onboarding was slow.", "", ""}
	worry := []string{"Too many tickets for the size of the team.", "Weekend rota is wearing people out.", "We need another hire before the holidays.", "Hard to plan when priorities change weekly.", "Pay has not kept up."}
	var rows []row
	n := 1
	for _, tm := range chartIDs(teams) {
		for i := 0; i < counts[tm]; i++ {
			score := int(chartClamp(math.Round(mean[tm]+r.norm()*1.4), 0, 10))
			workload := "right"
			if tm == "support" && r.chance(.65) || r.chance(.12) {
				workload = "heavy"
			} else if r.chance(.1) {
				workload = "light"
			}
			var picked []string
			for j, p := range chartIDs(perks) {
				if r.chance([]float64{.7, .6, .45, .4, .3, .22}[j]) {
					picked = append(picked, p)
				}
			}
			if len(picked) == 0 {
				picked = []string{"flex"}
			}
			comment := neutral[r.num(0, len(neutral)-1)]
			if score >= 9 {
				comment = praise[r.num(0, len(praise)-1)]
			} else if score <= 6 {
				comment = worry[r.num(0, len(worry)-1)]
			}
			level := "ic"
			if r.chance(.2) {
				level = "mgr"
			}
			group := "passive"
			if score >= 9 {
				group = "promoter"
			} else if score <= 6 {
				group = "detractor"
			}
			v := map[string]any{"group": group, "team": tm, "level": level, "tenure": r.weigh([]string{"new", "mid", "long"}, []float64{.25, .45, .3}), "score": score, "workload": workload, "perks": picked, "date": chartAgo(r.num(0, 13))}
			if comment != "" {
				v["comment"] = comment
			}
			rows = append(rows, row{Title: fmt.Sprintf("Response %02d", n), Values: v})
			n++
		}
	}
	dash := `{"layout":{"widgets":[` +
		`{"id":"n","size":"s","title":"Responses","chart":{"kind":"number","fn":"count"}},` +
		`{"id":"enps","size":"s","title":"Employee NPS","chart":{"kind":"number","fn":"average","prop":"points"}},` +
		`{"id":"avg","size":"s","title":"Median score","chart":{"kind":"number","fn":"median","prop":"score"}},` +
		`{"id":"promoters","size":"s","title":"Promoters","chart":{"kind":"number","fn":"count","filter":{"prop":"group","cond":"is","value":["promoter"]}}},` +
		`{"id":"hist","wide":true,"title":"Score distribution","chart":{"kind":"histogram","x":{"prop":"score","step":1},"fn":"count"}},` +
		`{"id":"teams","wide":true,"title":"Promoters and detractors by team","chart":{"kind":"bar","x":{"prop":"team"},"series":{"prop":"group"},"fn":"count","stacked":true,"legend":true,"labels":true}},` +
		`{"id":"perks","wide":true,"title":"What people value","chart":{"kind":"radial","x":{"prop":"perks"},"fn":"count","sort":"value_desc","legend":true}},` +
		`{"id":"load","title":"Workload","chart":{"kind":"donut","x":{"prop":"workload"},"fn":"count","labels":true}},` +
		`{"id":"tenure","title":"Tenure","chart":{"kind":"donut","x":{"prop":"tenure"},"fn":"count","labels":true}}` +
		`]}}`
	form := `{"form":{"title":"Team pulse survey","description":"Seven short questions. Answers are shared as totals only, never one by one.","questions":[` +
		`{"prop":"team","label":"Which team are you in?","required":true},` +
		`{"prop":"level","label":"Your role"},` +
		`{"prop":"tenure","label":"How long have you been here?"},` +
		`{"prop":"score","label":"How likely are you to recommend working here to a friend?","help":"0 means not at all likely, 10 means extremely likely.","required":true},` +
		`{"prop":"workload","label":"How does your workload feel right now?"},` +
		`{"prop":"perks","label":"Which benefits matter most to you?"},` +
		`{"prop":"comment","label":"Anything else we should know?"}` +
		`],"submit":"Send my answers","done":"Thank you. Results are shared at the next all-hands.","anonymous":true}}`
	return Starter{
		ID: "survey-results", Name: "Survey results", Description: "A team pulse survey with its form and 60 answers: a score histogram, eNPS from a formula, promoters by team as stacked bars, a radial of valued benefits and a workload donut.", Icon: "i:chat:pink", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"survey", "form", "enps", "histogram", "radial"},
		schema: []prop{
			{ID: "title", Name: "Response", Type: "title"},
			{ID: "team", Name: "Team", Type: "select", Options: teams},
			{ID: "level", Name: "Role", Type: "select", Options: []opt{o("ic", "Individual contributor", "gray"), o("mgr", "Manager", "brown")}},
			{ID: "tenure", Name: "Tenure", Type: "select", Options: []opt{o("new", "Under 1 year", "yellow"), o("mid", "1 to 3 years", "blue"), o("long", "Over 3 years", "purple")}},
			{ID: "score", Name: "Score", Type: "number", Display: "bar", Max: 10},
			{ID: "group", Name: "Group", Type: "select", Options: []opt{o("promoter", "Promoter", "green"), o("passive", "Passive", "gray"), o("detractor", "Detractor", "red")}},
			{ID: "points", Name: "eNPS points", Type: "formula", Expr: `if(prop("Score") >= 9, 100, if(prop("Score") >= 7, 0, -100))`},
			{ID: "workload", Name: "Workload", Type: "select", Options: []opt{o("light", "Too light", "blue"), o("right", "About right", "green"), o("heavy", "Too heavy", "red")}},
			{ID: "perks", Name: "Valued benefits", Type: "multi_select", Options: perks},
			{ID: "date", Name: "Submitted", Type: "date"},
			{ID: "comment", Name: "Comment", Type: "text"},
		},
		views: []view{
			{Name: "Results", Type: "dashboard", Config: dash},
			{Name: "Survey form", Type: "form", Config: form},
			{Name: "By team", Type: "table", Config: `{"group":{"prop":"team","hide_empty":true},"calcs":{"title":"count_all","score":"average","points":"average"}}`},
			{Name: "Comments", Type: "list", Config: `{"filter":{"prop":"comment","cond":"not_empty"},"props":[{"id":"comment","hidden":false},{"id":"team","hidden":false},{"id":"group","hidden":false}]}`},
			{Name: "Workload by team", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"team"},"series":{"prop":"workload"},"fn":"count","stacked":true,"legend":true}}}`},
		},
		rows: rows,
	}
}

func chartInventory() Starter {
	r := chartSeed(505)
	type item struct {
		name, cat, sup      string
		capacity, reorder   float64
		cost, perWeek, fill float64
	}
	items := []item{
		{"House espresso blend, 1 kg", "beans", "lakeside", 120, 30, 18.5, 28, .22},
		{"Single origin Ethiopia, 1 kg", "beans", "lakeside", 60, 15, 24, 9, .65},
		{"Single origin Colombia, 1 kg", "beans", "lakeside", 60, 15, 21.5, 11, .48},
		{"Decaf blend, 1 kg", "beans", "lakeside", 40, 10, 19, 5, .8},
		{"Cold brew coarse grind, 1 kg", "beans", "lakeside", 50, 12, 16.75, 10, .3},
		{"Seasonal blend, 1 kg", "beans", "lakeside", 40, 10, 22, 6, .9},
		{"English breakfast tea, 100 bags", "tea", "lakeside", 80, 15, 6.2, 8, .7},
		{"Green sencha, 100 bags", "tea", "lakeside", 50, 10, 8.4, 4, .55},
		{"Chai concentrate, 1 l", "tea", "lakeside", 48, 12, 7.9, 14, .2},
		{"Peppermint tea, 100 bags", "tea", "lakeside", 40, 8, 5.8, 3, .75},
		{"Whole milk, 2 l", "milk", "dairy", 180, 60, 2.4, 140, .55},
		{"Oat milk, 1 l", "milk", "dairy", 240, 80, 2.1, 190, .38},
		{"Almond milk, 1 l", "milk", "dairy", 96, 30, 2.3, 40, .62},
		{"Soy milk, 1 l", "milk", "dairy", 72, 24, 1.95, 22, .7},
		{"Whipping cream, 1 l", "milk", "dairy", 36, 10, 3.8, 9, .25},
		{"8 oz hot cups, sleeve of 50", "cups", "paper", 400, 100, 4.2, 120, .6},
		{"12 oz hot cups, sleeve of 50", "cups", "paper", 500, 120, 5.1, 180, .18},
		{"16 oz cold cups, sleeve of 50", "cups", "paper", 300, 80, 5.6, 70, .5},
		{"Sip lids, sleeve of 100", "cups", "paper", 400, 120, 3.9, 150, .45},
		{"Paper straws, box of 250", "cups", "paper", 60, 15, 6.5, 6, .85},
		{"Compostable napkins, pack of 500", "cups", "paper", 80, 20, 7.2, 10, .7},
		{"Vanilla syrup, 750 ml", "syrup", "lakeside", 36, 8, 6.9, 7, .5},
		{"Caramel syrup, 750 ml", "syrup", "lakeside", 36, 8, 6.9, 6, .3},
		{"Hazelnut syrup, 750 ml", "syrup", "lakeside", 24, 6, 7.1, 3, .75},
		{"Chocolate sauce, 2 l", "syrup", "lakeside", 20, 5, 12.4, 4, .6},
		{"Group head gaskets", "parts", "parts", 24, 6, 9.5, 1, .5},
		{"Grinder burr set", "parts", "parts", 6, 2, 64, .25, .33},
		{"Water filter cartridge", "parts", "parts", 12, 4, 38, 1, .25},
		{"Cleaning tablets, tub of 100", "parts", "parts", 20, 5, 14.8, 2, .6},
		{"Milk jug thermometer", "parts", "parts", 10, 3, 8.2, .5, .8},
	}
	capOf := map[string]float64{}
	var rows []row
	for _, it := range items {
		capOf[it.cat] += it.capacity
		rows = append(rows, row{Title: it.name, Values: map[string]any{"category": it.cat, "supplier": it.sup, "on_hand": math.Round(it.capacity * it.fill), "capacity": it.capacity, "reorder": it.reorder, "cost": it.cost, "per_week": it.perWeek, "counted": chartAgo(r.num(0, 6))}})
	}
	status := `{"prop":"state","sort":"manual","order":["reorder now","running low","in stock"],"hide_empty":true}`
	dash := `{"layout":{"widgets":[` +
		`{"id":"value","size":"s","title":"Stock value","chart":{"kind":"number","fn":"sum","prop":"value"}},` +
		`{"id":"reorder","size":"s","title":"Items to reorder now","chart":{"kind":"number","fn":"count","filter":{"prop":"state","cond":"is","value":"Reorder now"}}},` +
		fmt.Sprintf(`{"id":"beans","size":"s","title":"Coffee beans on hand","chart":{"kind":"gauge","fn":"sum","prop":"on_hand","target":%.0f,"filter":{"prop":"category","cond":"is","value":["beans"]}}},`, math.Round(capOf["beans"]*.75)) +
		fmt.Sprintf(`{"id":"milk","size":"s","title":"Milk on hand","chart":{"kind":"gauge","fn":"sum","prop":"on_hand","target":%.0f,"filter":{"prop":"category","cond":"is","value":["milk"]}}},`, math.Round(capOf["milk"]*.75)) +
		`{"id":"empty","wide":true,"title":"Emptiest shelves, fill %","chart":{"kind":"hbar","x":{"prop":"title"},"fn":"average","prop":"fill","sort":"value_asc","limit":10,"labels":true}},` +
		`{"id":"bycat","title":"Stock value by category","chart":{"kind":"donut","x":{"prop":"category"},"fn":"sum","prop":"value","labels":true}},` +
		`{"id":"state","title":"Stock status by category","chart":{"kind":"bar","x":{"prop":"category"},"series":` + status + `,"fn":"count","stacked":true,"legend":true}}` +
		`]}}`
	return Starter{
		ID: "inventory-stock", Name: "Inventory and stock levels", Description: "Thirty café supplies with gauges against target, an emptiest-shelves bar chart, fill rings and stock bars in the table, and a reorder board driven by a formula.", Icon: "i:box:amber", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"inventory", "gauge", "reorder", "formulas"},
		schema: []prop{
			{ID: "title", Name: "Item", Type: "title"},
			{ID: "sku", Name: "SKU", Type: "unique_id", Prefix: "SKU"},
			{ID: "category", Name: "Category", Type: "select", Options: []opt{o("beans", "Coffee beans", "brown"), o("tea", "Tea", "green"), o("milk", "Milk and alternatives", "blue"), o("cups", "Cups and lids", "gray"), o("syrup", "Syrups", "pink"), o("parts", "Equipment parts", "purple")}},
			{ID: "supplier", Name: "Supplier", Type: "select", Options: []opt{o("lakeside", "Lakeside Imports", "orange"), o("dairy", "Pacific Dairy", "blue"), o("paper", "Valley Paper Co", "gray"), o("parts", "Brewline Parts", "purple")}},
			{ID: "on_hand", Name: "On hand", Type: "number"},
			{ID: "capacity", Name: "Shelf capacity", Type: "number"},
			{ID: "fill", Name: "Fill, %", Type: "formula", Expr: `round(prop("On hand") / prop("Shelf capacity") * 100)`, Display: "ring", Max: 100},
			{ID: "reorder", Name: "Reorder at", Type: "number"},
			{ID: "per_week", Name: "Used per week", Type: "number"},
			{ID: "weeks", Name: "Weeks left", Type: "formula", Expr: `round(prop("On hand") / prop("Used per week"), 1)`, Display: "bar", Max: 8},
			{ID: "state", Name: "Stock status", Type: "formula", Expr: `if(prop("On hand") <= prop("Reorder at"), "Reorder now", if(prop("On hand") <= prop("Reorder at") * 1.5, "Running low", "In stock"))`},
			{ID: "cost", Name: "Unit cost", Type: "number", Format: "dollar", Precision: 2},
			{ID: "value", Name: "Stock value", Type: "formula", Expr: `round(prop("On hand") * prop("Unit cost"), 2)`, Format: "dollar", Precision: 2},
			{ID: "counted", Name: "Last counted", Type: "date"},
		},
		views: []view{
			{Name: "Overview", Type: "dashboard", Config: dash},
			{Name: "All items", Type: "table", Config: `{"group":{"prop":"category","hide_empty":true},"calcs":{"on_hand":"sum","fill":"average","value":"sum","title":"count_all"}}`},
			{Name: "Reorder board", Type: "board", Config: `{"group":` + status + `,"sorts":[{"prop":"weeks","dir":"asc"}],"props":[{"id":"fill","hidden":false},{"id":"weeks","hidden":false},{"id":"supplier","hidden":false}],"layout":{"total":{"prop":"value","fn":"sum"}}}`},
			{Name: "Weeks of stock", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"average","prop":"weeks","sort":"value_asc","limit":12,"labels":true}}}`},
		},
		rows: rows,
	}
}

func chartCapacity() Starter {
	r := chartSeed(606)
	type person struct {
		id, name, team string
		capacity       float64
	}
	people := []person{{"amara", "Amara Okafor", "design", 40}, {"ben", "Ben Carter", "eng", 40}, {"chloe", "Chloé Martin", "eng", 40}, {"diego", "Diego Ramos", "eng", 40}, {"elif", "Elif Kaya", "research", 40}, {"farah", "Farah Haddad", "design", 24}}
	var popts []opt
	for i, p := range people {
		popts = append(popts, o(p.id, p.name, []string{"pink", "blue", "purple", "green", "orange", "yellow"}[i]))
	}
	monday := (int(chartToday.Weekday()) + 6) % 7
	var rows []row
	for w := -8; w <= 3; w++ {
		d := monday - 7*w
		for _, p := range people {
			hours := 0.0
			project := "support"
			switch p.team {
			case "eng":
				project = "app"
				hours = r.span(33, 38)
				if w >= -3 && w <= -1 {
					hours = r.span(43, 49)
				}
			case "design":
				project = "portal"
				if p.id == "farah" {
					project = "support"
				}
				hours = (24 + float64(w+8)*1.3 + r.span(-2, 2)) * p.capacity / 40
			case "research":
				project = "research"
				hours = r.span(29, 35)
			}
			leave := p.id == "elif" && (w == -5 || w == -4)
			if leave {
				hours = 0
			}
			if w >= 2 {
				hours *= .7
			}
			rows = append(rows, row{Title: fmt.Sprintf("%s · %s", p.name, chartDay(d).Format("Jan 2")), Values: map[string]any{"person": p.id, "team": p.team, "week": chartAgo(d), "booked": math.Round(hours), "capacity": p.capacity, "project": project, "leave": leave}})
		}
	}
	return Starter{
		ID: "team-capacity", Name: "Team capacity", Description: "Six people over twelve weeks: hours as a stacked area by team, a people-by-week heatmap, utilisation bars and a board by project.", Icon: "i:users:teal", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"resourcing", "heatmap", "utilisation", "planning"},
		schema: []prop{
			{ID: "title", Name: "Booking", Type: "title"},
			{ID: "person", Name: "Person", Type: "select", Options: popts},
			{ID: "team", Name: "Team", Type: "select", Options: []opt{o("design", "Design", "pink"), o("eng", "Engineering", "purple"), o("research", "Research", "green")}},
			{ID: "week", Name: "Week of", Type: "date"},
			{ID: "booked", Name: "Hours booked", Type: "number"},
			{ID: "capacity", Name: "Capacity", Type: "number"},
			{ID: "util", Name: "Utilisation, %", Type: "formula", Expr: `round(prop("Hours booked") / prop("Capacity") * 100)`, Display: "bar", Max: 100},
			{ID: "project", Name: "Project", Type: "select", Options: []opt{o("app", "Mobile app 3.0", "blue"), o("portal", "Customer portal", "pink"), o("research", "Research sprint", "green"), o("support", "Support", "gray")}},
			{ID: "leave", Name: "On leave", Type: "checkbox"},
		},
		views: []view{
			{Name: "Hours by team", Type: "chart", Config: `{"layout":{"chart":{"kind":"area","x":{"prop":"week","by":"week"},"series":{"prop":"team"},"fn":"sum","prop":"booked","stacked":true,"smooth":true,"legend":true}}}`},
			{Name: "People by week", Type: "chart", Config: `{"layout":{"chart":{"kind":"heatmap","x":{"prop":"week","by":"week"},"series":{"prop":"person"},"fn":"sum","prop":"booked","labels":true}}}`},
			{Name: "Load by person", Type: "table", Config: `{"group":{"prop":"person","hide_empty":true},"sorts":[{"prop":"week","dir":"asc"}],"calcs":{"booked":"sum","util":"average","leave":"checked"}}`},
			{Name: "Utilisation", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"person"},"fn":"average","prop":"util","labels":true,"filter":{"prop":"week","cond":"on_or_before","value":{"relative":"today"}}}}}`},
			{Name: "By project", Type: "board", Config: `{"group":{"prop":"project","hide_empty":true},"props":[{"id":"booked","hidden":false},{"id":"util","hidden":false}],"layout":{"total":{"prop":"booked","fn":"sum"}}}`},
		},
		rows: rows,
	}
}

func chartCustomers() Starter {
	r := chartSeed(707)
	type city struct {
		name, country string
		lat, lng      float64
	}
	cities := []city{
		{"New York", "us", 40.7128, -74.006}, {"San Francisco", "us", 37.7749, -122.4194}, {"Chicago", "us", 41.8781, -87.6298}, {"Austin", "us", 30.2672, -97.7431}, {"Boston", "us", 42.3601, -71.0589}, {"Seattle", "us", 47.6062, -122.3321}, {"Denver", "us", 39.7392, -104.9903}, {"Atlanta", "us", 33.749, -84.388},
		{"Toronto", "ca", 43.6532, -79.3832}, {"Vancouver", "ca", 49.2827, -123.1207}, {"Montreal", "ca", 45.5017, -73.5673},
		{"London", "uk", 51.5074, -0.1278}, {"Manchester", "uk", 53.4808, -2.2426}, {"Edinburgh", "uk", 55.9533, -3.1883}, {"Bristol", "uk", 51.4545, -2.5879},
		{"Berlin", "de", 52.52, 13.405}, {"Munich", "de", 48.1351, 11.582}, {"Hamburg", "de", 53.5511, 9.9937}, {"Cologne", "de", 50.9375, 6.9603},
		{"Amsterdam", "nl", 52.3676, 4.9041}, {"Rotterdam", "nl", 51.9244, 4.4777}, {"Utrecht", "nl", 52.0907, 5.1214},
		{"Tokyo", "jp", 35.6762, 139.6503}, {"Osaka", "jp", 34.6937, 135.5023}, {"Fukuoka", "jp", 33.5904, 130.4017},
		{"Sydney", "au", -33.8688, 151.2093}, {"Melbourne", "au", -37.8136, 144.9631}, {"Brisbane", "au", -27.4698, 153.0251}, {"Perth", "au", -31.9505, 115.8605},
		{"São Paulo", "br", -23.5505, -46.6333}, {"Rio de Janeiro", "br", -22.9068, -43.1729}, {"Curitiba", "br", -25.4284, -49.2733},
	}
	quota := map[string]int{"us": 12, "uk": 6, "de": 5, "ca": 4, "au": 4, "nl": 3, "jp": 3, "br": 3}
	names := chartNames([]string{"Aurora", "Beacon", "Cinder", "Delta", "Evergreen", "Foxglove", "Gulf", "Highland", "Ironwood", "Kite"}, []string{"Health", "Logistics", "Media", "Foods", "Robotics", "Learning", "Energy", "Legal"}, 40)
	countries := []opt{o("us", "United States", "blue"), o("uk", "United Kingdom", "purple"), o("de", "Germany", "yellow"), o("ca", "Canada", "red"), o("au", "Australia", "green"), o("nl", "Netherlands", "orange"), o("jp", "Japan", "pink"), o("br", "Brazil", "brown")}
	var rows []row
	n := 0
	for _, c := range chartIDs(countries) {
		var here []city
		for _, x := range cities {
			if x.country == c {
				here = append(here, x)
			}
		}
		for i := 0; i < quota[c]; i++ {
			x := here[i%len(here)]
			jit := float64(i/len(here)) * .08
			plan := r.weigh([]string{"starter", "growth", "enterprise"}, []float64{.45, .38, .17})
			mrr, seats := 0.0, 0
			switch plan {
			case "starter":
				mrr, seats = math.Round(r.span(49, 199)), r.num(2, 8)
			case "growth":
				mrr, seats = math.Round(r.span(300, 1200)/10)*10, r.num(10, 45)
			default:
				mrr, seats = math.Round(r.span(2000, 6500)/50)*50, r.num(60, 400)
			}
			health := r.weigh([]string{"healthy", "watch", "risk"}, []float64{.65, .23, .12})
			name := names[n]
			rows = append(rows, row{Title: name, Values: map[string]any{"location": chartPlace(x.name, x.lat+jit, x.lng+jit), "country": c, "plan": plan, "mrr": mrr, "seats": seats, "since": chartAgo(int(360 * math.Pow(r.f(), .8))), "health": health, "contact": strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "@example.com"}})
			n++
		}
	}
	return Starter{
		ID: "customer-map", Name: "Customer map", Description: "Forty customers in real cities on a map, with a donut by country, revenue by country, a gallery of accounts and new customers by month.", Icon: "i:map:cyan", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"map", "crm", "customers", "gallery"},
		schema: []prop{
			{ID: "title", Name: "Customer", Type: "title"},
			{ID: "location", Name: "Location", Type: "place"},
			{ID: "country", Name: "Country", Type: "select", Options: countries},
			{ID: "plan", Name: "Plan", Type: "select", Options: []opt{o("starter", "Starter", "yellow"), o("growth", "Growth", "blue"), o("enterprise", "Enterprise", "purple")}},
			{ID: "mrr", Name: "Monthly revenue", Type: "number", Format: "dollar", Precision: -1, Display: "bar", Max: 6500},
			{ID: "seats", Name: "Seats", Type: "number"},
			{ID: "since", Name: "Customer since", Type: "date"},
			{ID: "health", Name: "Health", Type: "select", Options: []opt{o("healthy", "Healthy", "green"), o("watch", "Watch", "yellow"), o("risk", "At risk", "red")}},
			{ID: "contact", Name: "Contact", Type: "email"},
		},
		views: []view{
			{Name: "Map", Type: "map", Config: `{"layout":{"place":"location"},"props":[{"id":"plan","hidden":false},{"id":"mrr","hidden":false}]}`},
			{Name: "By country", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"country"},"fn":"count","labels":true,"sort":"value_desc"}}}`},
			{Name: "Revenue by country", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"country"},"series":{"prop":"plan"},"fn":"sum","prop":"mrr","stacked":true,"legend":true,"sort":"value_desc"}}}`},
			{Name: "Accounts", Type: "gallery", Config: `{"sorts":[{"prop":"mrr","dir":"desc"}],"props":[{"id":"plan","hidden":false},{"id":"mrr","hidden":false},{"id":"health","hidden":false},{"id":"country","hidden":false}],"layout":{"cover":"none","size":"small"}}`},
			{Name: "Directory", Type: "list", Config: `{"sorts":[{"prop":"title","dir":"asc"}],"props":[{"id":"country","hidden":false},{"id":"plan","hidden":false},{"id":"contact","hidden":false}]}`},
			{Name: "New customers", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"since","by":"month"},"series":{"prop":"plan"},"fn":"count","stacked":true,"legend":true}}}`},
		},
		rows: rows,
	}
}

func chartFitness() Starter {
	r := chartSeed(808)
	var rows []row
	for d := 150; d >= 1; d-- {
		t := float64(150-d) / 150
		if d >= 58 && d <= 67 {
			continue
		}
		wd := chartDay(d).Weekday()
		add := func(title, act string, km, mins, kcal, hr float64, effort string) {
			v := map[string]any{"date": chartAgo(d), "activity": act, "minutes": math.Round(mins), "calories": math.Round(kcal), "hr": math.Round(hr), "effort": effort}
			if km > 0 {
				v["distance"] = chartR(km, 1)
			}
			rows = append(rows, row{Title: title, Values: v})
		}
		pace := 6.4 - .9*t + r.span(-.15, .15)
		switch wd {
		case time.Tuesday:
			if r.chance(.8) {
				km := 5 + 3*t + r.span(-.5, .8)
				add("Easy run", "run", km, km*pace, km*pace*10.5, r.span(138, 150), "easy")
			}
		case time.Thursday:
			if r.chance(.7) {
				km := 6 + 2*t + r.span(-.5, .5)
				p := pace - .35
				add("Tempo run", "run", km, km*p, km*p*12, r.span(155, 168), "hard")
			}
		case time.Saturday:
			if r.chance(.85) {
				km := 9 + 9*t + r.span(-1, 1)
				p := pace + .2
				add("Long run", "run", km, km*p, km*p*10.8, r.span(145, 156), "moderate")
			}
		case time.Monday:
			if r.chance(.55) {
				m := r.span(40, 60)
				add("Strength", "strength", 0, m, m*7, r.span(110, 128), "moderate")
			}
		case time.Wednesday:
			if r.chance(.45) {
				m := r.span(30, 60)
				add("Yoga", "yoga", 0, m, m*3.5, r.span(85, 100), "easy")
			} else if r.chance(.4) {
				km := r.span(1.5, 2.5)
				add("Pool swim", "swim", km, km*r.span(17, 21), km*r.span(17, 21)*9, r.span(128, 142), "moderate")
			}
		case time.Sunday:
			if r.chance(.5) {
				km := r.span(28, 60)
				add("Bike ride", "ride", km, km*r.span(2.2, 2.6), km*r.span(2.2, 2.6)*8.5, r.span(125, 140), "moderate")
			}
		}
	}
	runs := `{"prop":"activity","cond":"is","value":["run"]}`
	dash := `{"layout":{"widgets":[` +
		`{"id":"km","size":"s","title":"Distance, km","kpi":{"date":"date","by":"week"},"chart":{"kind":"number","fn":"sum","prop":"distance"}},` +
		`{"id":"n","size":"s","title":"Workouts","kpi":{"date":"date","by":"week"},"chart":{"kind":"number","fn":"count"}},` +
		`{"id":"kcal","size":"s","title":"Calories burned","chart":{"kind":"number","fn":"sum","prop":"calories"}},` +
		`{"id":"long","size":"s","title":"Longest run, km","chart":{"kind":"number","fn":"max","prop":"distance","filter":` + runs + `}},` +
		`{"id":"dist","wide":true,"title":"Weekly running distance","chart":{"kind":"line","x":{"prop":"date","by":"week"},"fn":"sum","prop":"distance","smooth":true,"filter":` + runs + `}},` +
		`{"id":"pace","title":"Run pace, minutes per km","chart":{"kind":"line","x":{"prop":"date","by":"month"},"fn":"average","prop":"pace","smooth":true,"labels":true,"filter":` + runs + `}},` +
		`{"id":"time","title":"Time by activity","chart":{"kind":"donut","x":{"prop":"activity"},"fn":"sum","prop":"minutes","labels":true}},` +
		`{"id":"cal","wide":true,"title":"Active minutes","chart":{"kind":"calendar","x":{"prop":"date","by":"day"},"fn":"sum","prop":"minutes"}},` +
		`{"id":"dots","wide":true,"title":"Calories per workout","chart":{"kind":"dots","x":{"prop":"activity"},"prop":"calories"}}` +
		`]}}`
	return Starter{
		ID: "fitness-tracker", Name: "Fitness tracker", Description: "Five months of training toward a half marathon: KPIs with weekly trends, smoothed distance lines, run pace, an active-minutes calendar heatmap and calories as a dot plot.", Icon: "i:heart:rose", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"running", "health", "heatmap", "habits"},
		schema: []prop{
			{ID: "title", Name: "Workout", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "activity", Name: "Activity", Type: "select", Options: []opt{o("run", "Run", "blue"), o("ride", "Ride", "orange"), o("swim", "Swim", "green"), o("strength", "Strength", "purple"), o("yoga", "Yoga", "pink")}},
			{ID: "distance", Name: "Distance, km", Type: "number", Precision: 1},
			{ID: "minutes", Name: "Minutes", Type: "number"},
			{ID: "pace", Name: "Pace, min per km", Type: "formula", Expr: `if(prop("Activity") == "Run", round(prop("Minutes") / prop("Distance, km"), 2), "")`},
			{ID: "calories", Name: "Calories", Type: "number", Format: "comma"},
			{ID: "hr", Name: "Average heart rate", Type: "number", Display: "bar", Max: 190},
			{ID: "effort", Name: "Effort", Type: "select", Options: []opt{o("easy", "Easy", "green"), o("moderate", "Moderate", "yellow"), o("hard", "Hard", "red")}},
		},
		views: []view{
			{Name: "Progress", Type: "dashboard", Config: dash},
			{Name: "Training log", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"title":"count_all","distance":"sum","minutes":"sum","calories":"sum","hr":"average"}}`},
			{Name: "Activity heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":6,"fn":"sum","value":"minutes","color":"rose"}}}`},
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"},"props":[{"id":"distance","hidden":false}]}`},
			{Name: "Effort by month", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"month"},"series":{"prop":"effort"},"fn":"count","stacked":true,"legend":true}}}`},
		},
		rows: rows,
	}
}

func chartHabits() Starter {
	r := chartSeed(909)
	var rows []row
	for d := 80; d >= 1; d-- {
		t := 80 - d
		wd := chartDay(d).Weekday()
		weekend := wd == time.Saturday || wd == time.Sunday
		gap := d >= 33 && d <= 38
		late := t >= 25
		p := func(early, later float64) bool {
			if gap {
				return false
			}
			if late {
				return r.chance(later)
			}
			return r.chance(early)
		}
		read := p(.55, .85)
		run := false
		if wd == time.Tuesday || wd == time.Thursday || wd == time.Saturday {
			run = p(.55, .85)
		} else {
			run = p(.05, .12)
		}
		meditate := p(.3, .8)
		guitar := p(.2, .35)
		if weekend {
			guitar = p(.5, .75)
		}
		sleep := r.span(6.2, 7.6)
		if late {
			sleep = r.span(6.8, 8.2)
		}
		if gap {
			sleep = r.span(8, 9.2)
		}
		done := 0
		for _, b := range []bool{read, run, meditate, guitar} {
			if b {
				done++
			}
		}
		mood := "low"
		switch s := float64(done) + (sleep-7)*1.2 + r.norm()*.5; {
		case gap:
			mood = "great"
		case s >= 3.2:
			mood = "great"
		case s >= 2.1:
			mood = "good"
		case s >= 1.1:
			mood = "okay"
		}
		rows = append(rows, row{Title: chartDay(d).Format("Mon Jan 2"), Values: map[string]any{"date": chartAgo(d), "read": read, "run": run, "meditate": meditate, "guitar": guitar, "sleep": chartR(sleep, 1), "mood": mood}})
	}
	return Starter{
		ID: "habit-heatmap", Name: "Habit heatmap", Description: "Eighty days of four habits with a calendar heatmap, a week-by-weekday heatmap, completion rates in the table footer and a sleep trend.", Icon: "i:flame:orange", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"habits", "heatmap", "streaks", "personal"},
		schema: []prop{
			{ID: "title", Name: "Day", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "read", Name: "Read 20 pages", Type: "checkbox"},
			{ID: "run", Name: "Run", Type: "checkbox"},
			{ID: "meditate", Name: "Meditate", Type: "checkbox"},
			{ID: "guitar", Name: "Practise guitar", Type: "checkbox"},
			{ID: "done", Name: "Habits done", Type: "formula", Expr: `if(prop("Read 20 pages"), 1, 0) + if(prop("Run"), 1, 0) + if(prop("Meditate"), 1, 0) + if(prop("Practise guitar"), 1, 0)`, Display: "bar", Max: 4},
			{ID: "score", Name: "Day score", Type: "formula", Expr: `prop("Habits done") / 4`, Format: "percent", Display: "ring", Max: 1},
			{ID: "sleep", Name: "Sleep, hours", Type: "number", Precision: 1},
			{ID: "mood", Name: "Mood", Type: "select", Options: []opt{o("great", "Great", "green"), o("good", "Good", "blue"), o("okay", "Okay", "yellow"), o("low", "Low", "red")}},
		},
		views: []view{
			{Name: "Heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":3,"fn":"sum","value":"done","color":"green","labels":true}}}`},
			{Name: "Year view", Type: "chart", Config: `{"layout":{"chart":{"kind":"calendar","x":{"prop":"date","by":"day"},"fn":"sum","prop":"done"}}}`},
			{Name: "Weeks by weekday", Type: "chart", Config: `{"layout":{"chart":{"kind":"heatmap","x":{"prop":"date","by":"week"},"series":{"prop":"date","by":"weekday"},"fn":"sum","prop":"done"}}}`},
			{Name: "Habit log", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"read":"percent_checked","run":"percent_checked","meditate":"percent_checked","guitar":"percent_checked","done":"average","sleep":"average"}}`},
			{Name: "Sleep trend", Type: "chart", Config: `{"layout":{"chart":{"kind":"line","x":{"prop":"date","by":"week"},"fn":"average","prop":"sleep","smooth":true}}}`},
			{Name: "Mood and habits", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"mood"},"fn":"average","prop":"done","labels":true}}}`},
		},
		rows: rows,
	}
}

func chartKnowledge() Starter {
	type topic struct{ name, area, summary string }
	topics := []topic{
		{"Transit", "systems", "Buses, trams, metros and how people get around without a car."},
		{"Housing", "people", "Who can afford to live where, and how cities build homes."},
		{"Parks and trees", "environment", "Green space, street trees and the shade they give."},
		{"Water", "environment", "Rivers, rain, drinking water and flood protection."},
		{"Energy", "systems", "Heating, power and where it comes from."},
		{"Walkability", "people", "Whether daily errands can be done on foot."},
		{"Climate", "environment", "Heat, storms and how cities adapt."},
		{"Food", "people", "Markets, gardens and getting fresh food nearby."},
		{"Public health", "people", "How the shape of a city affects how long people live."},
		{"Street design", "systems", "Lanes, crossings and who the street is built for."},
	}
	type place struct {
		name, continent string
		lat, lng        float64
	}
	places := []place{
		{"Copenhagen", "europe", 55.6761, 12.5683}, {"Curitiba", "americas", -25.4284, -49.2733}, {"Singapore", "asia", 1.3521, 103.8198}, {"Vienna", "europe", 48.2082, 16.3738}, {"Tokyo", "asia", 35.6762, 139.6503},
		{"Barcelona", "europe", 41.3874, 2.1686}, {"Medellín", "americas", 6.2442, -75.5812}, {"Amsterdam", "europe", 52.3676, 4.9041}, {"Seoul", "asia", 37.5665, 126.978}, {"Paris", "europe", 48.8566, 2.3522},
	}
	type note struct {
		title, kind string
		topics      []string
		places      []string
		weight      int
		body        string
	}
	notes := []note{
		{"Curitiba's bus rapid transit", "example", []string{"Transit", "Street design"}, []string{"Curitiba"}, 5, "Curitiba gave buses their own lanes in the 1970s and later added raised tube stations where people pay before boarding. Buses behave like a metro for a fraction of the cost."},
		{"Medellín's cable cars", "example", []string{"Transit", "Housing"}, []string{"Medellín"}, 4, "Gondolas link steep hillside neighbourhoods to the metro, turning an hour's walk into a short ride."},
		{"Seoul's Cheonggyecheon stream", "example", []string{"Water", "Parks and trees", "Street design"}, []string{"Seoul"}, 5, "An elevated highway was taken down in the 2000s and the stream underneath was opened up again as a long linear park."},
		{"Barcelona's superblocks", "example", []string{"Street design", "Walkability", "Public health"}, []string{"Barcelona"}, 5, "Groups of nine blocks keep through-traffic on the edges. Inner streets become places to sit, play and walk."},
		{"Vienna's social housing", "fact", []string{"Housing"}, []string{"Vienna"}, 5, "A large share of Vienna's residents live in city-owned or subsidised flats, which keeps rents steady across the whole market."},
		{"Copenhagen's bike lanes", "fact", []string{"Transit", "Walkability", "Public health"}, []string{"Copenhagen"}, 4, "Wide, separated cycle tracks make riding the normal way to get to work for many residents."},
		{"District heating", "fact", []string{"Energy", "Climate"}, []string{"Copenhagen", "Vienna"}, 4, "One network of hot water pipes heats thousands of homes, using heat that would otherwise be wasted."},
		{"Singapore's recycled water", "fact", []string{"Water"}, []string{"Singapore"}, 4, "Treated used water is purified again and supplies part of the island's demand, mostly for industry."},
		{"Gardens on rooftops and walls", "example", []string{"Parks and trees", "Climate"}, []string{"Singapore"}, 3, "Building rules reward planting on roofs and facades, adding green space without using land."},
		{"Tokyo's rail network", "fact", []string{"Transit"}, []string{"Tokyo"}, 4, "Dense, frequent rail lets a very large city work with fewer car trips per person than most."},
		{"The 15-minute city", "idea", []string{"Walkability", "Food", "Public health"}, []string{"Paris"}, 5, "Most daily needs, from school to groceries to a doctor, within a short walk or ride from home."},
		{"Amsterdam's canals as infrastructure", "fact", []string{"Water", "Transit"}, []string{"Amsterdam"}, 3, "The canal rings drain the city, carry boats and shape where the streets run."},
		{"Urban heat islands", "fact", []string{"Climate", "Parks and trees", "Public health"}, []string{}, 5, "Dark roofs and roads soak up heat. Street trees and lighter surfaces can make a block noticeably cooler on hot afternoons."},
		{"Sponge city design", "idea", []string{"Water", "Climate", "Parks and trees"}, []string{}, 4, "Let rain soak in where it falls with parks, swales and permeable paving instead of piping it all away."},
		{"Car-free school streets", "question", []string{"Street design", "Public health", "Walkability"}, []string{"Barcelona", "Paris"}, 3, "Would closing the street outside a school at drop-off times work on our street? Who would need to agree?"},
		{"Markets as public space", "idea", []string{"Food", "Walkability"}, []string{"Barcelona", "Tokyo"}, 3, "Covered markets bring people out daily and keep small food sellers in the centre of town."},
		{"Community gardens", "idea", []string{"Food", "Parks and trees"}, []string{"Amsterdam"}, 2, "Shared plots turn empty land into food and a reason for neighbours to meet."},
		{"Rooftop solar on public buildings", "idea", []string{"Energy", "Climate"}, []string{}, 3, "Schools and libraries have big flat roofs and use most power in daytime, when panels produce it."},
		{"Level boarding saves minutes", "fact", []string{"Transit", "Street design"}, []string{"Curitiba"}, 3, "When the platform is level with the bus floor, everyone boards faster, including people with prams and wheelchairs."},
		{"Street trees and walking", "question", []string{"Parks and trees", "Walkability"}, []string{}, 2, "Do people choose shaded routes even when they are longer? Worth checking on our own walk to the station."},
		{"Cycling and health", "fact", []string{"Public health", "Transit"}, []string{"Copenhagen", "Amsterdam"}, 3, "Regular cycling to work builds everyday exercise into the commute."},
		{"Mixed-use ground floors", "idea", []string{"Walkability", "Housing"}, []string{"Vienna", "Paris"}, 3, "Shops and cafés below flats keep streets busy at all hours."},
		{"Flood parks", "example", []string{"Water", "Parks and trees", "Climate"}, []string{"Copenhagen"}, 3, "Some parks and squares are built to fill with water in a cloudburst and drain slowly afterwards."},
		{"Gondolas in the transit map", "question", []string{"Transit"}, []string{"Medellín"}, 2, "Should cable cars be shown on the same map as the metro? It changes how people think of them."},
		{"Heat and older residents", "fact", []string{"Public health", "Climate", "Housing"}, []string{}, 4, "Heat waves hit older people living alone hardest. Cool rooms in libraries can help."},
		{"Trams return", "example", []string{"Transit", "Street design"}, []string{"Paris"}, 3, "Several cities have rebuilt tram lines on boulevards that once carried only cars."},
		{"Tactical urbanism", "idea", []string{"Street design"}, []string{"Barcelona"}, 2, "Try a change with paint and planters first, then build it properly once it works."},
		{"Water-sensitive streets", "idea", []string{"Water", "Street design"}, []string{"Seoul"}, 2, "Planted strips beside the road clean runoff before it reaches the river."},
		{"Energy-efficient social housing", "example", []string{"Housing", "Energy"}, []string{"Vienna"}, 3, "Newer municipal blocks combine low rents with low heating bills."},
		{"Night buses", "question", []string{"Transit", "Public health"}, []string{"Tokyo"}, 2, "Who travels after midnight, and how do shift workers get home when trains stop?"},
		{"Food deserts", "fact", []string{"Food", "Public health"}, []string{}, 3, "Areas with no fresh food shop within walking distance tend to have worse diets."},
		{"Wide pavements", "idea", []string{"Walkability", "Street design"}, []string{"Copenhagen"}, 2, "Room for two prams to pass is a simple test of a good pavement."},
		{"Rain gardens at home", "idea", []string{"Water", "Parks and trees"}, []string{}, 1, "A small planted dip in the garden takes roof water and grows flowers."},
		{"Quiet zones", "question", []string{"Public health", "Housing"}, []string{"Amsterdam"}, 2, "Does traffic noise at night affect sleep enough to change where people choose to live?"},
		{"River swimming", "example", []string{"Water", "Public health"}, []string{"Copenhagen", "Paris"}, 3, "Cleaner rivers have opened harbour and river baths for summer swimming."},
		{"Shared heat pumps", "idea", []string{"Energy", "Housing"}, []string{}, 2, "One large heat pump for a block can be cheaper and quieter than one per flat."},
	}
	var trows, prows, nrows []row
	for _, t := range topics {
		trows = append(trows, row{Title: t.name, Values: map[string]any{"area": t.area, "summary": t.summary}})
	}
	for _, p := range places {
		prows = append(prows, row{Title: p.name, Values: map[string]any{"location": chartPlace(p.name, p.lat, p.lng), "continent": p.continent}})
	}
	related := map[string][]string{
		"Curitiba's bus rapid transit":      {"Level boarding saves minutes", "Medellín's cable cars"},
		"Barcelona's superblocks":           {"Car-free school streets", "Tactical urbanism", "The 15-minute city"},
		"The 15-minute city":                {"Mixed-use ground floors", "Markets as public space", "Wide pavements"},
		"Urban heat islands":                {"Street trees and walking", "Heat and older residents", "Gardens on rooftops and walls"},
		"Sponge city design":                {"Flood parks", "Water-sensitive streets", "Rain gardens at home"},
		"District heating":                  {"Shared heat pumps", "Energy-efficient social housing"},
		"Vienna's social housing":           {"Energy-efficient social housing", "Mixed-use ground floors"},
		"Copenhagen's bike lanes":           {"Cycling and health", "Wide pavements"},
		"Seoul's Cheonggyecheon stream":     {"Water-sensitive streets", "River swimming"},
		"Medellín's cable cars":             {"Gondolas in the transit map"},
		"Food deserts":                      {"Markets as public space", "Community gardens"},
		"Rooftop solar on public buildings": {"Shared heat pumps"},
		"Tokyo's rail network":              {"Night buses"},
	}
	for i, n := range notes {
		v := map[string]any{"type": n.kind, "topics": n.topics, "places": n.places, "importance": n.weight, "added": chartAgo(120 - i*3)}
		if rel, ok := related[n.title]; ok {
			v["related"] = rel
		}
		nrows = append(nrows, row{Title: n.title, Values: v, Markdown: n.body + "\n"})
	}
	return Starter{
		ID: "knowledge-graph", Name: "Knowledge graph", Description: "Thirty-six linked notes on how cities work: a graph view of notes, topics and places, a map of the places, a feed of notes and a topic chart.", Icon: "i:atom:violet", Kind: "doc", Category: catCharts, Featured: true, Tags: []string{"notes", "graph", "research", "relations"},
		markdown: `A research notebook about how cities work. Each note links to the topics it touches and the places it mentions, so the graph below shows which ideas cluster together.

> [!TIP]
> Drag a node to move it, scroll to zoom, and click a note to open it. Topics and places show up as linked nodes from their own databases.

{{db:notes}}

## Browse

- {{page:topics}}: ten themes, from transit to public health.
- {{page:places}}: the cities mentioned in the notes, on a map.

<details>
<summary>How to add a note</summary>

1. Open **Notes** and press **New**.
2. Pick a type: idea, fact, question or example.
3. Link one or more topics and places. The graph updates as you type.

</details>
`,
		children: []Starter{
			{
				key: "topics", Name: "Topics", Icon: "i:tag:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Topic", Type: "title"},
					{ID: "area", Name: "Area", Type: "select", Options: []opt{o("systems", "Systems", "blue"), o("people", "People", "pink"), o("environment", "Environment", "green")}},
					{ID: "summary", Name: "Summary", Type: "text"},
				},
				views: []view{{Name: "Topics", Type: "table", Config: `{"group":{"prop":"area","hide_empty":true}}`}},
				rows:  trows,
			},
			{
				key: "places", Name: "Places", Icon: "i:pin:teal", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "City", Type: "title"},
					{ID: "location", Name: "Location", Type: "place"},
					{ID: "continent", Name: "Continent", Type: "select", Options: []opt{o("europe", "Europe", "blue"), o("asia", "Asia", "red"), o("americas", "Americas", "green")}},
				},
				views: []view{{Name: "Map", Type: "map", Config: `{"layout":{"place":"location"}}`}, {Name: "All places", Type: "table", Config: `{}`}},
				rows:  prows,
			},
			{
				key: "notes", Name: "Notes", Icon: "i:note:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Note", Type: "title"},
					{ID: "type", Name: "Type", Type: "select", Options: []opt{o("idea", "Idea", "purple"), o("fact", "Fact", "blue"), o("question", "Question", "yellow"), o("example", "Example", "green")}},
					{ID: "topics", Name: "Topics", Type: "relation", Collection: "@topics"},
					{ID: "places", Name: "Places", Type: "relation", Collection: "@places"},
					{ID: "related", Name: "Related notes", Type: "relation", Collection: "@self"},
					{ID: "importance", Name: "Importance", Type: "number", Display: "bar", Max: 5},
					{ID: "links", Name: "Links", Type: "formula", Expr: `length(prop("Topics")) + length(prop("Places")) + length(prop("Related notes"))`},
					{ID: "added", Name: "Added", Type: "date"},
				},
				views: []view{
					{Name: "Graph", Type: "graph", Config: `{"layout":{"graph":{"color":"type","labels":"auto","cluster":true}}}`},
					{Name: "Feed", Type: "feed", Config: `{"sorts":[{"prop":"added","dir":"desc"}]}`},
					{Name: "All notes", Type: "table", Config: `{"sorts":[{"prop":"importance","dir":"desc"}],"calcs":{"title":"count_all","links":"sum"}}`},
					{Name: "Notes per topic", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"topics"},"series":{"prop":"type"},"fn":"count","stacked":true,"legend":true,"sort":"value_desc"}}}`},
					{Name: "By type", Type: "board", Config: `{"group":{"prop":"type","hide_empty":true},"props":[{"id":"topics","hidden":false},{"id":"importance","hidden":false}]}`},
				},
				rows: nrows,
			},
		},
	}
}

func chartFormulas() Starter {
	r := chartSeed(1010)
	type client struct {
		name     string
		rate     float64
		slow     bool
		invoices []string
	}
	clients := []*client{{name: "Bluebird Bakery", rate: 85}, {name: "Copperleaf Dental", rate: 120}, {name: "Harbor Yoga Studio", rate: 85}, {name: "Northgate Architects", rate: 120}, {name: "Pinecrest School", rate: 95}, {name: "Riverside Vet Clinic", rate: 110}, {name: "Saltmarsh Brewing", rate: 95, slow: true}, {name: "Willow and Oak Florist", rate: 85}}
	var inv []row
	n := 2041
	for d := 178; d >= 2; d -= 4 + r.num(0, 3) {
		c := clients[(len(inv)*3+r.num(0, 1))%len(clients)]
		hours := float64(r.num(6, 40))
		paid := d > 45 && r.chance(.95) || d <= 45 && r.chance(.45)
		if c.slow && d > 20 && d < 120 {
			paid = r.chance(.25)
		}
		title := fmt.Sprintf("INV-%d", n)
		c.invoices = append(c.invoices, title)
		inv = append(inv, row{Title: title, Values: map[string]any{"client": strings.ToLower(strings.Fields(c.name)[0]), "issued": chartAgo(d), "hours": hours, "rate": c.rate, "paid": paid}})
		n++
		if len(inv) == 40 {
			break
		}
	}
	var copts []opt
	var crows []row
	for i, c := range clients {
		copts = append(copts, o(strings.ToLower(strings.Fields(c.name)[0]), c.name, []string{"blue", "pink", "green", "purple", "yellow", "orange", "brown", "red"}[i]))
		crows = append(crows, row{Title: c.name, Values: map[string]any{"invoices": c.invoices}})
	}
	return Starter{
		ID: "formula-playground", Name: "Formula and rollup playground", Description: "Clients and invoices linked by a relation: rollups for totals, paid share and outstanding money, formulas for tax, due dates and lateness, shown as bars, rings and charts.", Icon: "i:flask:lime", Kind: "doc", Category: catCharts, Featured: true, Tags: []string{"formulas", "rollups", "relations", "invoices"},
		markdown: `<!-- toc -->

A small studio's invoices and clients, built to show how formulas and rollups work together. Change an invoice's hours or tick **Paid**, and every total, ring and chart that depends on it updates.

## Clients, with rollups

Each client links to its invoices. The columns after **Invoices** are rollups: they read the linked invoices and add them up.

{{db:clients}}

| Column | Kind | How it is worked out |
| --- | --- | --- |
| Billed | Rollup, sum | Adds up **Total** on every linked invoice |
| Invoices sent | Rollup, count | Counts the linked invoices |
| Paid share | Rollup, percent checked | Share of linked invoices with **Paid** ticked |
| Outstanding | Rollup, sum | Adds up **Unpaid** on linked invoices |
| Last invoice | Rollup, latest | The newest **Issued** date |
| Payment health | Formula | Reads **Paid share** and returns Good, Watch or Chase |

## Invoices, with formulas

{{db:invoices}}

The tax and total on each invoice:

$$\text{Amount} = \text{Hours} \times \text{Rate}, \qquad \text{Total} = \text{Amount} \times 1.2$$

> [!IMPORTANT]
> Formulas refer to other properties by name, for example ` + "`prop(\"Hours\")`" + `. If you rename a property, formulas that use it follow the new name.

<details>
<summary>Every formula in this template</summary>

| Property | Formula |
| --- | --- |
| Amount | ` + "`prop(\"Hours\") * prop(\"Rate\")`" + ` |
| Tax | ` + "`round(prop(\"Amount\") * 0.2, 2)`" + ` |
| Total | ` + "`prop(\"Amount\") + prop(\"Tax\")`" + ` |
| Due | ` + "`dateAdd(prop(\"Issued\"), 30, \"days\")`" + ` |
| Unpaid | ` + "`if(prop(\"Paid\"), 0, prop(\"Total\"))`" + ` |
| Days late | ` + "`if(not prop(\"Paid\") and now() > prop(\"Due\"), dateBetween(now(), prop(\"Due\"), \"days\"), 0)`" + ` |
| Status | ` + "`if(prop(\"Paid\"), \"Paid\", if(now() > prop(\"Due\"), \"Overdue\", \"Open\"))`" + ` |
| Payment health | ` + "`if(prop(\"Paid share\") >= 0.9, \"Good\", if(prop(\"Paid share\") >= 0.7, \"Watch\", \"Chase\"))`" + ` |

</details>
`,
		children: []Starter{
			{
				key: "invoices", Name: "Invoices", Icon: "i:clipboard:lime", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Invoice", Type: "title"},
					{ID: "client", Name: "Billed to", Type: "select", Options: copts},
					{ID: "issued", Name: "Issued", Type: "date"},
					{ID: "due", Name: "Due", Type: "formula", Expr: `dateAdd(prop("Issued"), 30, "days")`},
					{ID: "hours", Name: "Hours", Type: "number"},
					{ID: "rate", Name: "Rate", Type: "number", Format: "dollar", Precision: -1},
					{ID: "amount", Name: "Amount", Type: "formula", Expr: `prop("Hours") * prop("Rate")`, Format: "dollar", Precision: -1},
					{ID: "tax", Name: "Tax", Type: "formula", Expr: `round(prop("Amount") * 0.2, 2)`, Format: "dollar", Precision: 2},
					{ID: "total", Name: "Total", Type: "formula", Expr: `prop("Amount") + prop("Tax")`, Format: "dollar", Precision: 2},
					{ID: "paid", Name: "Paid", Type: "checkbox"},
					{ID: "unpaid", Name: "Unpaid", Type: "formula", Expr: `if(prop("Paid"), 0, prop("Total"))`, Format: "dollar", Precision: 2},
					{ID: "late", Name: "Days late", Type: "formula", Expr: `if(not prop("Paid") and now() > prop("Due"), dateBetween(now(), prop("Due"), "days"), 0)`, Display: "bar", Max: 90},
					{ID: "status", Name: "Status", Type: "formula", Expr: `if(prop("Paid"), "Paid", if(now() > prop("Due"), "Overdue", "Open"))`},
				},
				views: []view{
					{Name: "Billing by month", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"issued","by":"month"},"series":{"prop":"status","sort":"manual","order":["paid","open","overdue"]},"fn":"sum","prop":"total","stacked":true,"legend":true,"labels":true}}}`},
					{Name: "All invoices", Type: "table", Config: `{"sorts":[{"prop":"issued","dir":"desc"}],"calcs":{"title":"count_all","hours":"sum","total":"sum","unpaid":"sum","late":"max","paid":"percent_checked"}}`},
					{Name: "By status", Type: "board", Config: `{"group":{"prop":"status","sort":"manual","order":["overdue","open","paid"],"hide_empty":true},"props":[{"id":"client","hidden":false},{"id":"total","hidden":false},{"id":"late","hidden":false}],"layout":{"total":{"prop":"total","fn":"sum"}}}`},
				},
				rows: inv,
			},
			{
				key: "clients", Name: "Clients", Icon: "i:building:lime", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Client", Type: "title"},
					{ID: "invoices", Name: "Invoices", Type: "relation", Collection: "@invoices"},
					{ID: "billed", Name: "Billed", Type: "rollup", Relation: "invoices", Target: "total", Fn: "sum", Format: "dollar", Display: "bar", Max: 30000},
					{ID: "count", Name: "Invoices sent", Type: "rollup", Relation: "invoices", Target: "title", Fn: "count_all"},
					{ID: "share", Name: "Paid share", Type: "rollup", Relation: "invoices", Target: "paid", Fn: "percent_checked", Format: "percent", Display: "ring", Max: 1},
					{ID: "outstanding", Name: "Outstanding", Type: "rollup", Relation: "invoices", Target: "unpaid", Fn: "sum", Format: "dollar"},
					{ID: "last", Name: "Last invoice", Type: "rollup", Relation: "invoices", Target: "issued", Fn: "latest"},
					{ID: "health", Name: "Payment health", Type: "formula", Expr: `if(prop("Paid share") >= 0.9, "Good", if(prop("Paid share") >= 0.7, "Watch", "Chase"))`},
				},
				views: []view{
					{Name: "Clients", Type: "table", Config: `{"sorts":[{"prop":"billed","dir":"desc"}],"calcs":{"billed":"sum","count":"sum","outstanding":"sum","share":"average"}}`},
					{Name: "Billed by client", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"billed","sort":"value_desc","labels":true}}}`},
					{Name: "Outstanding", Type: "chart", Config: `{"layout":{"chart":{"kind":"radial","x":{"prop":"title"},"fn":"sum","prop":"outstanding","sort":"value_desc","hide_empty":true,"labels":true}}}`},
					{Name: "Payment health", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"health"},"fn":"count","labels":true}}}`},
				},
				rows: crows,
			},
		},
	}
}

func chartLibrary() Starter {
	r := chartSeed(1111)
	type item struct{ title, kind, topic, summary string }
	items := []item{
		{"Your first week with a new tool", "guide", "start", "A day-by-day plan for getting a team comfortable with a new workspace without overwhelming anyone."},
		{"Five keyboard habits that save an hour a week", "article", "productivity", "Small shortcuts for moving, searching and formatting that add up quickly."},
		{"How to run a ten-minute standup", "video", "productivity", "A short walkthrough of a standup that stays on time: three questions, one board, no status essays."},
		{"Reading a chart without fooling yourself", "article", "data", "Axes that do not start at zero, averages that hide outliers and other traps, with examples."},
		{"Choosing colours for charts", "guide", "design", "How to pick colours that stay readable for colour-blind readers and in dark mode."},
		{"Writing headlines people click", "article", "writing", "Specific beats clever. Ten before-and-after headlines from real newsletters."},
		{"The two-list method for priorities", "podcast", "productivity", "A conversation about keeping one short list of things that matter and one long list of everything else."},
		{"What a median tells you that an average hides", "article", "data", "Why one huge order can make an average lie, and when to use the median instead."},
		{"Sketching before you design", "video", "design", "Paper first: how rough sketches save hours of polishing the wrong idea."},
		{"Plain language checklist", "guide", "writing", "Twelve checks for clearer writing, from short sentences to everyday words."},
		{"Setting up a team wiki", "guide", "start", "What to put on the home page, how to organise sub-pages and who keeps it tidy."},
		{"Running a bakery on spreadsheets", "podcast", "data", "A baker explains how simple tables track flour, orders and the Saturday rush."},
		{"Making forms people finish", "article", "design", "Fewer questions, clear labels and honest progress make the difference."},
		{"Weekly reviews that stick", "podcast", "productivity", "Two people compare the fifteen-minute review they have kept up for years."},
		{"Turning notes into a first draft", "video", "writing", "From scattered notes to a structured draft in three passes."},
		{"Dashboards for small teams", "guide", "data", "Three numbers, one trend and a target: a dashboard people actually open."},
		{"Accessible colour contrast in five minutes", "video", "design", "A quick test for text contrast and what to change when it fails."},
		{"Templates that save meetings", "article", "start", "Agendas, decision logs and follow-ups that turn hour-long calls into twenty minutes."},
		{"Editing your own writing", "podcast", "writing", "Reading aloud, cutting the first paragraph and other editing habits."},
		{"Spotting seasonality in sales", "article", "data", "How to tell a real trend from the usual December bump."},
		{"Inbox zero for shared inboxes", "guide", "productivity", "Labels, owners and a daily sweep for team inboxes."},
		{"Icon sets: when less is more", "article", "design", "Why a dozen well-chosen icons beat a library of thousands."},
		{"How to ask for feedback", "podcast", "writing", "Ask specific questions and you get useful answers."},
		{"Importing a spreadsheet the right way", "video", "start", "Clean the headers, pick property types and check dates before you import."},
		{"Heatmaps explained", "video", "data", "Calendar heatmaps, matrices and when colour beats bars."},
		{"Naming things in a workspace", "article", "start", "Page titles people can find a year later."},
		{"Story structure for case studies", "guide", "writing", "Problem, turning point, result: a simple frame for customer stories."},
		{"Focus blocks and calendar hygiene", "video", "productivity", "Protecting two hours a day without missing what matters."},
		{"Maps that answer one question", "article", "data", "Pins, regions and when a map is the wrong chart."},
		{"Typography basics for docs", "guide", "design", "Line length, headings and spacing for documents people read on screens."},
		{"Year-end review walkthrough", "video", "productivity", "A guided tour of a yearly review page."},
		{"Measuring what matters", "podcast", "data", "Choosing three numbers for a team and ignoring the rest."},
	}
	authors := []opt{o("nadia", "Nadia Rahman", "purple"), o("owen", "Owen Price", "blue"), o("sofia", "Sofia Marino", "pink"), o("jun", "Jun Park", "green")}
	var rows []row
	for i, it := range items {
		status, d := "published", 150-i*(147/26)
		switch {
		case i >= 30:
			status, d = "idea", -35-(i-30)*10
		case i >= 28:
			status, d = "draft", -18-(i-28)*7
		case i >= 26:
			status, d = "editing", -5-(i-26)*7
		}
		mins := map[string][2]int{"article": {4, 9}, "guide": {8, 15}, "video": {5, 14}, "podcast": {25, 48}}[it.kind]
		v := map[string]any{"kind": it.kind, "topic": it.topic, "status": status, "author": chartIDs(authors)[r.num(0, 3)], "publish": chartAgo(d), "minutes": r.num(mins[0], mins[1]), "checked": status == "published" || status == "editing" && r.chance(.5)}
		if status == "published" {
			mult := map[string]float64{"data": 1.6, "productivity": 1.3, "design": 1, "writing": .8, "start": 1.1}[it.topic]
			v["views"] = math.Round((800 + float64(d)*60*r.span(.6, 1.4)) * mult)
			v["rating"] = chartR(chartClamp(3.6+r.norm()*.5+(mult-1)*.6, 2.5, 5), 1)
		}
		kindLine := map[string]string{"article": "Read time", "guide": "Read time", "video": "Watch time", "podcast": "Listen time"}[it.kind]
		rows = append(rows, row{Title: it.title, Values: v, Markdown: fmt.Sprintf("%s\n\n**%s:** about %d minutes.\n", it.summary, kindLine, v["minutes"])})
	}
	return Starter{
		ID: "content-library", Name: "Content library", Description: "Thirty-two articles, videos and podcasts as a gallery, a feed, an editorial board with progress and totals, a publishing calendar and view charts.", Icon: "i:book:orange", Kind: "database", Category: catCharts, Featured: true, Tags: []string{"content", "feed", "gallery", "editorial"},
		schema: []prop{
			{ID: "title", Name: "Title", Type: "title"},
			{ID: "kind", Name: "Format", Type: "select", Options: []opt{o("article", "Article", "blue"), o("guide", "Guide", "green"), o("video", "Video", "red"), o("podcast", "Podcast", "purple")}},
			{ID: "topic", Name: "Topic", Type: "select", Options: []opt{o("start", "Getting started", "gray"), o("productivity", "Productivity", "orange"), o("data", "Data", "blue"), o("design", "Design", "pink"), o("writing", "Writing", "yellow")}},
			{ID: "status", Name: "Status", Type: "status", Options: []opt{so("idea", "Idea", "gray", "todo"), so("draft", "Drafting", "yellow", "doing"), so("editing", "Editing", "purple", "doing"), so("published", "Published", "green", "done")}},
			{ID: "author", Name: "Author", Type: "select", Options: authors},
			{ID: "publish", Name: "Publish date", Type: "date"},
			{ID: "minutes", Name: "Minutes", Type: "number"},
			{ID: "views", Name: "Views", Type: "number", Format: "comma", Display: "bar", Max: 15000},
			{ID: "rating", Name: "Rating", Type: "number", Precision: 1, Display: "ring", Max: 5},
			{ID: "checked", Name: "Fact-checked", Type: "checkbox"},
		},
		views: []view{
			{Name: "Library", Type: "gallery", Config: `{"sorts":[{"prop":"views","dir":"desc"}],"props":[{"id":"kind","hidden":false},{"id":"topic","hidden":false},{"id":"views","hidden":false},{"id":"rating","hidden":false}],"layout":{"cover":"none","size":"medium"}}`},
			{Name: "Feed", Type: "feed", Config: `{"filter":{"prop":"status","cond":"is","value":["published"]},"sorts":[{"prop":"publish","dir":"desc"}]}`},
			{Name: "Editorial board", Type: "board", Config: `{"group":{"prop":"status","hide_empty":true},"props":[{"id":"author","hidden":false},{"id":"kind","hidden":false},{"id":"views","hidden":false}],"layout":{"progress":"checked","total":{"prop":"minutes","fn":"sum"}}}`},
			{Name: "Publishing calendar", Type: "calendar", Config: `{"layout":{"date":"publish"},"props":[{"id":"kind","hidden":false}]}`},
			{Name: "Reading list", Type: "list", Config: `{"sorts":[{"prop":"publish","dir":"desc"}],"props":[{"id":"kind","hidden":false},{"id":"minutes","hidden":false},{"id":"author","hidden":false}]}`},
			{Name: "Views by format", Type: "chart", Config: `{"layout":{"chart":{"kind":"line","x":{"prop":"publish","by":"month"},"series":{"prop":"kind"},"fn":"sum","prop":"views","legend":true,"filter":{"prop":"status","cond":"is","value":["published"]}}}}`},
			{Name: "Top ten", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"views","sort":"value_desc","limit":10,"labels":true,"filter":{"prop":"status","cond":"is","value":["published"]}}}}`},
		},
		rows: rows,
	}
}

func chartReport() Starter {
	r := chartSeed(1212)
	var rows []row
	revenue := 128000.0
	var revs, costList, news, churns []float64
	var starts []time.Time
	for k := 18; k >= 1; k-- {
		_, _, start := chartMonth(k)
		revenue *= 1.025 + r.span(-.02, .025)
		if start.Month() == 12 {
			revenue *= 1.12
		} else if start.Month() == 1 {
			revenue /= 1.12
		}
		costs := revenue * r.span(.7, .8)
		newC := math.Round(revenue/9000 + r.span(-3, 4))
		churn := math.Round(r.span(3, 9))
		rv, cs := math.Round(revenue/100)*100, math.Round(costs/100)*100
		revs, costList, news, churns, starts = append(revs, rv), append(costList, cs), append(news, newC), append(churns, churn), append(starts, start)
		rows = append(rows, row{Title: start.Format("January 2006"), Values: map[string]any{"month": chartAgo(chartBack(start)), "revenue": rv, "costs": cs, "new": newC, "churned": churn}})
	}
	sum := func(xs []float64, from, to int) float64 {
		t := 0.0
		for _, x := range xs[from:to] {
			t += x
		}
		return t
	}
	n := len(revs)
	pct := func(a, b float64) string {
		v := (a - b) / b * 100
		sign := "+"
		if v < 0 {
			sign = "-"
		}
		return fmt.Sprintf("%s%.1f%%", sign, math.Abs(v))
	}
	money := func(x float64) string {
		s := fmt.Sprintf("%.0f", x)
		for i := len(s) - 3; i > 0; i -= 3 {
			s = s[:i] + "," + s[i:]
		}
		return "$" + s
	}
	r0, r1 := sum(revs, n-3, n), sum(revs, n-6, n-3)
	c0, c1 := sum(costList, n-3, n), sum(costList, n-6, n-3)
	m0, m1 := (r0-c0)/r0*100, (r1-c1)/r1*100
	n0, n1 := sum(news, n-3, n), sum(news, n-6, n-3)
	h0, h1 := sum(churns, n-3, n), sum(churns, n-6, n-3)
	trend := "grew"
	if r0 < r1 {
		trend = "dipped"
	}
	table := fmt.Sprintf("| Measure | Last three months | Three months before | Change |\n| --- | --- | --- | --- |\n| Revenue | %s | %s | %s |\n| Margin | %.1f%% | %.1f%% | %+.1f pts |\n| New customers | %.0f | %.0f | %s |\n| Churned customers | %.0f | %.0f | %s |\n\n> [!TIP]\n> Revenue %s %s on the three months before. Each number here is worked out from the **Monthly metrics** database.",
		money(r0), money(r1), pct(r0, r1), m0, m1, m0-m1, n0, n1, pct(n0, n1), h0, h1, pct(h0, h1), trend, strings.TrimLeft(pct(r0, r1), "+-"))
	var qLabels, qRev, qCost []string
	top := 0.0
	for q := 3; q >= 0; q-- {
		a, b := n-3*q-3, n-3*q
		qLabels = append(qLabels, fmt.Sprintf("%q", starts[a].Format("Jan")+" to "+starts[b-1].Format("Jan")))
		qRev = append(qRev, fmt.Sprintf("%.0f", sum(revs, a, b)/1000))
		top = math.Max(top, math.Ceil(sum(revs, a, b)/1000/100)*100)
		qCost = append(qCost, fmt.Sprintf("%.0f", sum(costList, a, b)/1000))
	}
	xy := fmt.Sprintf("xychart-beta\n  title \"Revenue and costs per quarter, thousands\"\n  x-axis [%s]\n  y-axis \"Thousands\" 0 --> %.0f\n  bar [%s]\n  line [%s]", strings.Join(qLabels, ", "), top, strings.Join(qRev, ", "), strings.Join(qCost, ", "))
	dash := `{"layout":{"widgets":[` +
		`{"id":"rev","title":"Revenue","kpi":{"date":"month","by":"month"},"chart":{"kind":"number","fn":"sum","prop":"revenue"}},` +
		`{"id":"profit","title":"Profit","kpi":{"date":"month","by":"month"},"chart":{"kind":"number","fn":"sum","prop":"profit"}},` +
		`{"id":"margin","size":"s","title":"Average margin, %","chart":{"kind":"number","fn":"average","prop":"margin"}},` +
		`{"id":"new","size":"s","title":"New customers","chart":{"kind":"number","fn":"sum","prop":"new"}},` +
		`{"id":"cust","title":"Customers, running total","chart":{"kind":"area","x":{"prop":"month","by":"month"},"fn":"sum","prop":"net","cumulative":true,"smooth":true}},` +
		`{"id":"bars","wide":true,"title":"Revenue by month","chart":{"kind":"bar","x":{"prop":"month","by":"month"},"fn":"sum","prop":"revenue"}},` +
		`{"id":"profitline","wide":true,"title":"Profit by month","chart":{"kind":"line","x":{"prop":"month","by":"month"},"fn":"sum","prop":"profit","smooth":true}}` +
		`]}}`
	return Starter{
		ID: "quarterly-report", Name: "Quarterly report", Description: "A written business review with a live dashboard inside the page, math, callouts, toggles, tables and Mermaid charts.", Icon: "i:chart-bar:blue", Kind: "doc", Category: catCharts, Featured: true, Tags: []string{"report", "dashboard", "math", "mermaid"},
		markdown: strings.NewReplacer("@@TABLE@@", table, "@@XY@@", xy).Replace(`<!-- toc -->

> [!NOTE]
> A sample quarterly review for a fictional company, Northwind Outdoor. Numbers in the dashboard come from the **Monthly metrics** database below, so editing a month changes the charts.

## Headlines

@@TABLE@@

## Dashboard

{{db:metrics}}

## How we measure growth

Quarter-on-quarter growth and compound monthly growth over $n$ months:

$$g = \frac{R_{\text{now}} - R_{\text{before}}}{R_{\text{before}}}, \qquad \text{CMGR} = \left(\frac{R_{\text{end}}}{R_{\text{start}}}\right)^{1/n} - 1$$

Margin is profit over revenue, $m = \frac{R - C}{R}$, shown as a percentage.

## Revenue mix

~~~mermaid
pie showData
  title Revenue by channel this quarter
  "Online portal" : 52
  "Sales reps" : 37
  "Trade shows" : 11
~~~

~~~mermaid
@@XY@@
~~~

## Risks and asks

> [!WARNING]
> Shipping costs rose 9% this quarter. If they keep rising, margin falls back below 23% by spring.

> [!CAUTION]
> Two of our five largest wholesale customers renew next quarter.

- [x] Hire a second warehouse lead
- [ ] Renegotiate the courier contract
- [ ] Launch the spring catalogue

<details>
<summary>Method notes</summary>

Revenue is booked when an order ships. Costs include goods, shipping and staff, but not one-off office costs. Customer counts exclude trial accounts.

</details>

<details>
<summary>Glossary</summary>

| Term | Meaning |
| --- | --- |
| Churn | Customers who did not renew in the month |
| Net new | New customers minus churned customers |
| Margin | Profit as a share of revenue |

</details>
`),
		children: []Starter{
			{
				key: "metrics", Name: "Monthly metrics", Icon: "i:chart-bar:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Month", Type: "title"},
					{ID: "month", Name: "Month start", Type: "date"},
					{ID: "revenue", Name: "Revenue", Type: "number", Format: "dollar", Precision: -1},
					{ID: "costs", Name: "Costs", Type: "number", Format: "dollar", Precision: -1},
					{ID: "profit", Name: "Profit", Type: "formula", Expr: `prop("Revenue") - prop("Costs")`, Format: "dollar", Precision: -1},
					{ID: "margin", Name: "Margin, %", Type: "formula", Expr: `round(prop("Profit") / prop("Revenue") * 100, 1)`, Display: "ring", Max: 40},
					{ID: "new", Name: "New customers", Type: "number"},
					{ID: "churned", Name: "Churned", Type: "number"},
					{ID: "net", Name: "Net new", Type: "formula", Expr: `prop("New customers") - prop("Churned")`},
				},
				views: []view{
					{Name: "Dashboard", Type: "dashboard", Config: dash},
					{Name: "Table", Type: "table", Config: `{"sorts":[{"prop":"month","dir":"asc"}],"calcs":{"revenue":"sum","costs":"sum","profit":"sum","margin":"average","new":"sum","churned":"sum"}}`},
				},
				rows: rows,
			},
		},
	}
}

func chartDiagrams() Starter {
	return Starter{
		ID: "diagram-gallery", Name: "Diagram gallery", Description: "Every kind of Mermaid diagram the editor draws, from flowcharts and sequences to Gantt, sankey, mind maps and charts, plus math, callouts and toggles.", Icon: "i:layers:blue", Kind: "doc", Category: catCharts, Featured: true, Tags: []string{"mermaid", "diagrams", "math", "flowchart"},
		markdown: `Diagrams are written as text in a ` + "`mermaid`" + ` code block and drawn as you type. Copy any of these into your own pages and change the words.

<!-- toc -->

> [!TIP]
> Type ` + "`/diagram`" + ` in any page to start a new one. Click a diagram to edit its source.

## Flowchart

Steps and decisions. Arrows can carry labels.

~~~mermaid
flowchart LR
  A[Order placed] --> B{In stock?}
  B -- Yes --> C[Pick and pack]
  B -- No --> D[Order from supplier]
  D --> C
  C --> E[Ship]
  E --> F((Delivered))
~~~

## Sequence diagram

Who talks to whom, in order.

~~~mermaid
sequenceDiagram
  participant C as Customer
  participant W as Web shop
  participant P as Payments
  participant S as Warehouse
  C->>W: Checkout
  W->>P: Charge card
  P-->>W: Approved
  W->>S: Create shipment
  S-->>C: Tracking email
~~~

## Class diagram

Things, their fields and how they relate.

~~~mermaid
classDiagram
  class Customer {
    +String name
    +String email
  }
  class Order {
    +Date placed
    +total() Money
  }
  class LineItem {
    +int quantity
    +Money price
  }
  Customer "1" --> "*" Order : places
  Order "1" *-- "*" LineItem : contains
~~~

## State diagram

The states something moves through.

~~~mermaid
stateDiagram-v2
  [*] --> Draft
  Draft --> Review : submit
  Review --> Draft : changes requested
  Review --> Approved : approve
  Approved --> Published : publish
  Published --> [*]
~~~

## Entity relationship diagram

Tables and how their rows link.

~~~mermaid
erDiagram
  CUSTOMER ||--o{ ORDER : places
  ORDER ||--|{ LINE_ITEM : contains
  PRODUCT ||--o{ LINE_ITEM : "appears in"
  CUSTOMER {
    string name
    string email
  }
  ORDER {
    date placed
    string status
  }
~~~

## User journey

How a person feels at each step, from 1 to 5.

~~~mermaid
journey
  title Joining a climbing gym
  section Discover
    See a poster: 3: Visitor
    Read reviews: 4: Visitor
  section Join
    Book a taster session: 4: Visitor, Staff
    Fill in the waiver: 2: Visitor
    Pay the first month: 3: Member
  section First week
    First class: 5: Member, Coach
~~~

## Gantt chart

Tasks over time, with dependencies.

~~~mermaid
gantt
  title Website relaunch
  dateFormat YYYY-MM-DD
  section Plan
    Research        :active, r1, {{date}}, 10d
    Wireframes      :w1, after r1, 8d
  section Build
    Design system   :d1, after w1, 12d
    Pages           :p1, after d1, 15d
  section Launch
    Testing         :t1, after p1, 7d
    Go live         :milestone, after t1, 0d
~~~

## Timeline

Events in order, grouped by period.

~~~mermaid
timeline
  title A short history of flight
  1903 : First powered aeroplane flight
  1927 : First solo nonstop flight across the Atlantic
  1969 : First crewed Moon landing
  1976 : Supersonic passenger service begins
  2021 : First powered flight on another planet
~~~

## Mind map

Ideas branching from one centre.

~~~mermaid
mindmap
  root((Garden plan))
    Vegetables
      Tomatoes
      Beans
      Lettuce
    Herbs
      Basil
      Mint
    Flowers
      Sunflowers
      Marigolds
    Jobs
      Compost
      Water daily
~~~

## Pie chart

Shares of a whole.

~~~mermaid
pie showData
  title Where a coffee's price goes, percent
  "Staff" : 31
  "Rent" : 22
  "Beans" : 18
  "Profit" : 14
  "Milk" : 9
  "Cups and lids" : 6
~~~

## Quadrant chart

Items placed on two scales.

~~~mermaid
quadrantChart
  title Feature requests by effort and impact
  x-axis Low effort --> High effort
  y-axis Low impact --> High impact
  quadrant-1 Plan carefully
  quadrant-2 Do first
  quadrant-3 Maybe later
  quadrant-4 Avoid
  CSV export: [0.2, 0.75]
  Dark mode: [0.35, 0.6]
  Offline sync: [0.85, 0.8]
  Keyboard shortcuts: [0.25, 0.4]
  Custom fonts: [0.65, 0.2]
~~~

## XY chart

Bars and lines on shared axes.

~~~mermaid
xychart-beta
  title "Monthly visitors"
  x-axis [Jan, Feb, Mar, Apr, May, Jun]
  y-axis "Visitors" 0 --> 6000
  bar [2100, 2400, 3100, 3600, 4200, 5100]
  line [2000, 2300, 2900, 3500, 4000, 4900]
~~~

## Sankey diagram

Flows from sources to destinations, sized by amount.

~~~mermaid
sankey-beta
Salary,Rent,1850
Salary,Groceries,520
Salary,Transport,180
Salary,Savings,900
Salary,Fun,350
Freelance,Savings,600
Freelance,Fun,200
~~~

## Git graph

Branches and merges.

~~~mermaid
gitGraph
  commit id: "Start"
  branch feature
  checkout feature
  commit id: "Add search"
  commit id: "Fix tests"
  checkout main
  commit id: "Hotfix"
  merge feature
  commit id: "Release"
~~~

## Kanban

Cards in columns.

~~~mermaid
kanban
  todo[To do]
    t1[Write the brief]
    t2[Book the venue]
  doing[In progress]
    t3[Design the poster]
  done[Done]
    t4[Pick a date]
~~~

## Radar chart

Several measures compared on one wheel.

~~~mermaid
radar-beta
  title Team skills
  axis design["Design"], code["Code"], writing["Writing"], data["Data"], support["Support"]
  curve a["Team A"]{4, 3, 5, 2, 4}
  curve b["Team B"]{2, 5, 3, 4, 3}
  max 5
~~~

## Treemap

Nested shares as rectangles.

~~~mermaid
treemap-beta
"Monthly budget"
    "Home"
        "Rent": 1850
        "Utilities": 180
    "Living"
        "Groceries": 520
        "Transport": 180
        "Fun": 350
    "Savings": 900
~~~

## Math

Inline math sits in a sentence, like the area of a circle $A = \pi r^2$ or Euler's identity $e^{i\pi} + 1 = 0$. Block math gets its own line:

$$x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$$

$$A = P\left(1 + \frac{r}{n}\right)^{nt}$$

## Callouts

> [!NOTE]
> Useful information that readers should know, even when skimming.

> [!TIP]
> Helpful advice for doing things better or more easily.

> [!IMPORTANT]
> Key information readers need to succeed.

> [!WARNING]
> Urgent information that needs immediate attention.

> [!CAUTION]
> Advice about risks or negative outcomes.

## Toggles

<details>
<summary>Which diagram should I use?</summary>

| You want to show | Try |
| --- | --- |
| Steps and decisions | Flowchart |
| Messages between people or systems | Sequence diagram |
| A plan over time | Gantt chart or timeline |
| Shares of a whole | Pie chart or treemap |
| Money or energy moving | Sankey diagram |
| Ideas around a theme | Mind map |

</details>

<details>
<summary>Where do live charts come from?</summary>

Mermaid charts are drawn from the numbers you type. For charts that update from data, add a database and give it a chart or dashboard view. The Data and charts templates show every kind.

</details>
`,
	}
}
