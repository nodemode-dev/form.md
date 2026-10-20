package templates

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

func xbDay(n int) string { return fmt.Sprintf("%+dd", n) }

func xbSpan(a, b int) map[string]any { return map[string]any{"start": xbDay(a), "end": xbDay(b)} }

func xbAt(lat, lng float64, name string) map[string]any {
	return map[string]any{"lat": lat, "lng": lng, "name": name}
}

func xbRound(x float64, d int) float64 {
	m := math.Pow(10, float64(d))
	return math.Round(x*m) / m
}

func xbShow(ids ...string) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf(`{"id":%q,"hidden":false}`, id)
	}
	return `"props":[` + strings.Join(parts, ",") + `]`
}

func xbHide(ids ...string) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf(`{"id":%q,"hidden":true}`, id)
	}
	return `"props":[` + strings.Join(parts, ",") + `]`
}

func xbWeekday(d int) int {
	switch ((int(time.Now().Weekday())+d)%7 + 7) % 7 {
	case 6:
		return d - 1
	case 0:
		return d + 1
	}
	return d
}

func xbCols(items ...string) string {
	parts := make([]string, len(items))
	for i, it := range items {
		if id, ok := strings.CutPrefix(it, "-"); ok {
			parts[i] = fmt.Sprintf(`{"id":%q,"hidden":true}`, id)
			continue
		}
		id, w, _ := strings.Cut(it, ":")
		if w == "" {
			parts[i] = fmt.Sprintf(`{"id":%q,"hidden":false}`, id)
			continue
		}
		parts[i] = fmt.Sprintf(`{"id":%q,"hidden":false,"width":%s}`, id, w)
	}
	return `"props":[` + strings.Join(parts, ",") + `]`
}

func xbWhere(rows []row, key, want string) []string {
	out := []string{}
	for _, r := range rows {
		switch v := r.Values[key].(type) {
		case string:
			if v == want {
				out = append(out, r.Title)
			}
		case []string:
			for _, x := range v {
				if x == want {
					out = append(out, r.Title)
					break
				}
			}
		}
	}
	return out
}

type xbRand struct{ s uint32 }

func (r *xbRand) f() float64 {
	r.s = r.s*1664525 + 1013904223
	return float64(r.s>>8) / 16777216
}

func (r *xbRand) in(a, b float64) float64 { return a + (b-a)*r.f() }

func (r *xbRand) n(a, b int) int { return a + int(r.f()*float64(b-a+1)) }

var xbGroupRE = regexp.MustCompile(`"group":\{"prop":"([a-z]+)"\}`)

func xbTidy(list []Starter) []Starter {
	for i := range list {
		for j := range list[i].views {
			list[i].views[j].Config = xbGroupRE.ReplaceAllString(list[i].views[j].Config, `"group":{"prop":"$1","hide_empty":true}`)
		}
		list[i].children = xbTidy(list[i].children)
	}
	return list
}

var exampleStartersB = xbTidy([]Starter{
	xbFinance(),
	xbFitness(),
	xbTrip(),
	xbEvent(),
	xbRenovation(),
	xbCourse(),
	xbCafe(),
})

func xbCafe() Starter {
	rnd := &xbRand{s: 314}
	type sup struct {
		name, cat, town string
		lat, lng        float64
		lead            float64
		day             string
		monthly         float64
	}
	sups := []sup{
		{"Rose City Roasters", "coffee", "Portland", 45.5152, -122.6784, 2, "tue", 2400},
		{"Coastal Meadow Dairy", "dairy", "Tillamook", 45.4562, -123.8440, 1, "mon", 1850},
		{"Beaverton Bread Works", "bakery", "Beaverton", 45.4871, -122.8037, 1, "daily", 2100},
		{"Hood River Orchards", "produce", "Hood River", 45.7054, -121.5215, 3, "thu", 640},
		{"Hillsboro Greens Farm", "produce", "Hillsboro", 45.5229, -122.9898, 2, "wed", 720},
		{"Salem Egg Farm", "dairy", "Salem", 44.9429, -123.0351, 2, "mon", 380},
		{"Cascade Tea Traders", "tea", "Seattle", 47.6062, -122.3321, 5, "fri", 310},
		{"Willamette Valley Honey", "pantry", "Newberg", 45.3001, -122.9732, 4, "thu", 220},
		{"Columbia Oat Mill", "dairy", "Eugene", 44.0521, -123.0868, 4, "tue", 560},
		{"Pacific Paper Goods", "packaging", "Vancouver, WA", 45.6387, -122.6615, 3, "fri", 450},
	}
	suprows := make([]row, 0, len(sups))
	for _, s := range sups {
		slug := strings.ToLower(strings.Fields(s.name)[0])
		suprows = append(suprows, row{Title: s.name, Values: map[string]any{"cat": s.cat, "where": xbAt(s.lat, s.lng, s.town), "lead": s.lead, "day": s.day, "monthly": s.monthly, "email": "orders@" + slug + ".example.com"}})
	}
	suprows[0].Markdown = "House espresso blend (Brazil and Ethiopia) and a rotating single origin for filter. Roast date on every bag; we use beans 5 to 21 days after roasting.\n"
	type stock struct {
		name, sup, cat, unit string
		have, par, cost      float64
	}
	stocks := []stock{
		{"Espresso beans, house blend", "Rose City Roasters", "coffee", "kg", 9, 20, 22},
		{"Filter coffee, single origin", "Rose City Roasters", "coffee", "kg", 4, 6, 26},
		{"Decaf beans", "Rose City Roasters", "coffee", "kg", 2.5, 3, 24},
		{"Whole milk", "Coastal Meadow Dairy", "dairy", "litre", 38, 60, 1.2},
		{"Oat milk", "Columbia Oat Mill", "dairy", "litre", 9, 30, 2.1},
		{"Butter", "Coastal Meadow Dairy", "dairy", "kg", 6, 8, 9.5},
		{"Cheddar", "Coastal Meadow Dairy", "dairy", "kg", 3.5, 5, 11},
		{"Free-range eggs", "Salem Egg Farm", "dairy", "dozen", 14, 20, 4.8},
		{"Sourdough loaves", "Beaverton Bread Works", "bakery", "loaf", 12, 18, 4.5},
		{"Croissants, frozen", "Beaverton Bread Works", "bakery", "piece", 70, 120, 0.9},
		{"Bagels", "Beaverton Bread Works", "bakery", "piece", 34, 48, 0.6},
		{"Avocados", "Hillsboro Greens Farm", "produce", "piece", 18, 40, 1.1},
		{"Salad leaves", "Hillsboro Greens Farm", "produce", "kg", 2, 4, 8},
		{"Tomatoes", "Hillsboro Greens Farm", "produce", "kg", 5, 6, 3.5},
		{"Apples", "Hood River Orchards", "produce", "kg", 9, 10, 2.4},
		{"Berries, frozen", "Hood River Orchards", "produce", "kg", 6, 8, 7},
		{"Earl grey tea", "Cascade Tea Traders", "tea", "box of 100", 3, 4, 18},
		{"Green tea", "Cascade Tea Traders", "tea", "box of 100", 1, 3, 16},
		{"Chai concentrate", "Cascade Tea Traders", "tea", "litre", 7, 8, 9},
		{"Local honey", "Willamette Valley Honey", "pantry", "jar", 10, 12, 6.5},
		{"Rolled oats", "Columbia Oat Mill", "pantry", "kg", 11, 15, 2.2},
		{"Takeaway cups, 12 oz", "Pacific Paper Goods", "packaging", "sleeve of 50", 9, 30, 7.5},
		{"Takeaway lids", "Pacific Paper Goods", "packaging", "sleeve of 100", 11, 15, 6},
		{"Paper bags", "Pacific Paper Goods", "packaging", "pack of 250", 3, 4, 12},
		{"Napkins", "Pacific Paper Goods", "packaging", "case", 2, 3, 22},
	}
	strows := make([]row, 0, len(stocks))
	for _, s := range stocks {
		strows = append(strows, row{Title: s.name, Values: map[string]any{"supplier": s.sup, "cat": s.cat, "unit": s.unit, "have": s.have, "par": s.par, "cost": s.cost}})
	}
	type dish struct {
		name, cat   string
		price, cost float64
		vegan       bool
		sold        float64
		uses        []string
	}
	dishes := []dish{
		{"Espresso", "coffee", 3.25, 0.42, true, 140, []string{"Espresso beans, house blend"}},
		{"Flat white", "coffee", 4.50, 0.78, false, 310, []string{"Espresso beans, house blend", "Whole milk"}},
		{"Latte", "coffee", 4.75, 0.82, false, 360, []string{"Espresso beans, house blend", "Whole milk", "Takeaway cups, 12 oz"}},
		{"Oat latte", "coffee", 5.25, 1.05, true, 240, []string{"Espresso beans, house blend", "Oat milk"}},
		{"Cappuccino", "coffee", 4.50, 0.74, false, 220, []string{"Espresso beans, house blend", "Whole milk"}},
		{"Filter coffee", "coffee", 3.50, 0.55, true, 180, []string{"Filter coffee, single origin"}},
		{"Mocha", "coffee", 5.25, 1.10, false, 95, []string{"Espresso beans, house blend", "Whole milk"}},
		{"Chai latte", "tea", 4.95, 1.15, false, 130, []string{"Chai concentrate", "Whole milk"}},
		{"Pot of earl grey", "tea", 3.75, 0.40, true, 70, []string{"Earl grey tea"}},
		{"Green tea", "tea", 3.50, 0.38, true, 45, []string{"Green tea"}},
		{"Iced latte", "cold", 5.25, 0.95, false, 85, []string{"Espresso beans, house blend", "Whole milk", "Takeaway cups, 12 oz"}},
		{"Berry smoothie", "cold", 6.50, 2.10, true, 60, []string{"Berries, frozen", "Oat milk"}},
		{"Butter croissant", "bakery", 3.95, 1.05, false, 260, []string{"Croissants, frozen", "Butter"}},
		{"Apple cinnamon muffin", "bakery", 3.75, 0.95, false, 150, []string{"Apples"}},
		{"Honey oat bar", "bakery", 3.50, 0.80, true, 110, []string{"Rolled oats", "Local honey"}},
		{"Avocado on sourdough", "breakfast", 11.50, 3.40, true, 140, []string{"Avocados", "Sourdough loaves", "Tomatoes"}},
		{"Eggs any way on toast", "breakfast", 10.50, 2.60, false, 120, []string{"Free-range eggs", "Sourdough loaves", "Butter"}},
		{"Overnight oats with berries", "breakfast", 8.50, 1.90, true, 75, []string{"Rolled oats", "Berries, frozen", "Oat milk"}},
		{"Bagel with egg and cheddar", "breakfast", 9.25, 2.30, false, 105, []string{"Bagels", "Free-range eggs", "Cheddar"}},
		{"Grilled cheese and tomato soup", "lunch", 13.50, 3.90, false, 95, []string{"Sourdough loaves", "Cheddar", "Tomatoes", "Butter"}},
		{"Harvest salad bowl", "lunch", 13.00, 4.20, true, 70, []string{"Salad leaves", "Apples", "Tomatoes"}},
		{"Soup of the day with bread", "lunch", 9.50, 2.20, true, 88, []string{"Sourdough loaves", "Tomatoes"}},
	}
	mrows := make([]row, 0, len(dishes))
	for _, d := range dishes {
		mrows = append(mrows, row{Title: d.name, Values: map[string]any{"cat": d.cat, "price": d.price, "cost": d.cost, "vegan": d.vegan, "sold": d.sold, "uses": d.uses}})
	}
	mrows[15].Markdown = "## Build\n\n1. Toast one thick slice of sourdough.\n2. Half an avocado, smashed with lemon, salt and chilli flakes.\n3. Sliced tomatoes, a drizzle of oil, black pepper.\n\nAdd a poached egg for 2.00.\n"
	popular := []string{"Latte", "Flat white", "Oat latte", "Butter croissant", "Avocado on sourdough"}
	wd := int(time.Now().Weekday())
	dayFactor := []float64{1.25, 0.82, 0.86, 0.92, 0.97, 1.12, 1.38}
	var sales []row
	for d := -60; d <= -1; d++ {
		w := ((wd+d)%7 + 7) % 7
		weather := "cloudy"
		wf := 1.0
		switch x := rnd.f(); {
		case x < 0.32:
			weather, wf = "rainy", 0.9
		case x > 0.72:
			weather, wf = "sunny", 1.08
		}
		trend := 1 + float64(d+60)*0.003
		rev := 1380 * dayFactor[w] * wf * trend * (0.94 + rnd.f()*0.12)
		ticket := 9.4 + rnd.in(-0.5, 0.7)
		if w == 0 || w == 6 {
			ticket += 1.6
		}
		cust := math.Round(rev / ticket)
		staff := 3.0
		if w == 0 || w == 6 || w == 5 {
			staff = 5
		}
		sales = append(sales, row{Title: fmt.Sprintf("Day %d", d+61), Values: map[string]any{"date": xbDay(d), "revenue": xbRound(rev, 2), "customers": cust, "weather": weather, "staff": staff, "top": popular[rnd.n(0, len(popular)-1)]}})
	}
	sales[len(sales)-1].Markdown = "Busy morning, the oat milk ran out at 14:00. Order more on Monday.\n"
	return Starter{
		ID: "ex-neighbourhood-cafe", Name: "Corner Cup Cafe", Description: "A small cafe run from one workspace: menu margins, two months of daily sales, stock levels with gauges, suppliers on a map and the opening checklist.", Icon: "i:coffee:amber", Kind: "doc", Category: catExamples, Tags: []string{"cafe", "sales", "inventory", "small business"},
		markdown: `Corner Cup is a twelve-table cafe on a corner in Portland, Oregon, run by Lucia and her team of six. This space replaced a folder of spreadsheets: the menu knows its margins, the stock list knows when to reorder and the till numbers go in every evening.

> [!TIP]
> Each evening, add one row to Daily sales. The charts, weekday heatmap and week-on-week comparison update themselves.

## How the cafe is doing

{{db:cafe-sales}}

## What earns the money

Every dish with its price, ingredient cost and margin. Coffee carries the business; lunch brings in fewer, bigger tickets.

{{db:cafe-menu}}

## Stock

Gauges for the items that run out fastest, and a reorder flag on everything else.

{{db:cafe-stock}}

## Suppliers

All local, all within a morning's drive.

{{db:cafe-suppliers}}

## Running the place

- {{page:cafe-opening}}: the opening and closing checklists
- {{page:cafe-recipes}}: drink specs and kitchen builds
- {{page:cafe-espresso}}: how to dial in the espresso each morning
`,
		children: []Starter{
			{
				key: "cafe-suppliers", Name: "Suppliers", Icon: "i:pin:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Supplier", Type: "title"},
					{ID: "cat", Name: "Supplies", Type: "select", Options: []opt{o("coffee", "Coffee", "brown"), o("dairy", "Dairy and eggs", "yellow"), o("bakery", "Bakery", "orange"), o("produce", "Fruit and veg", "green"), o("tea", "Tea", "purple"), o("pantry", "Pantry", "pink"), o("packaging", "Packaging", "gray")}},
					{ID: "where", Name: "Location", Type: "place"},
					{ID: "day", Name: "Delivers on", Type: "select", Options: []opt{o("daily", "Every day", "green"), o("mon", "Monday", "blue"), o("tue", "Tuesday", "blue"), o("wed", "Wednesday", "blue"), o("thu", "Thursday", "blue"), o("fri", "Friday", "blue")}},
					{ID: "lead", Name: "Lead time (days)", Type: "number"},
					{ID: "monthly", Name: "Spend per month", Type: "number", Format: "dollar"},
					{ID: "email", Name: "Orders email", Type: "email"},
				},
				views: []view{
					{Name: "Map", Type: "map", Config: `{"layout":{"place":"where"},` + xbShow("cat", "day") + `}`},
					{Name: "All suppliers", Type: "table", Config: `{"sorts":[{"prop":"monthly","dir":"desc"}],"calcs":{"monthly":"sum"}}`},
					{Name: "Spend by supplier", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"cat"},"fn":"sum","prop":"monthly","labels":true,"legend":true}}}`},
				},
				rows: suprows,
			},
			{
				key: "cafe-stock", Name: "Stock", Icon: "i:box:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Item", Type: "title"},
					{ID: "supplier", Name: "Supplier", Type: "relation", Collection: "@cafe-suppliers"},
					{ID: "cat", Name: "Category", Type: "select", Options: []opt{o("coffee", "Coffee", "brown"), o("dairy", "Dairy and eggs", "yellow"), o("bakery", "Bakery", "orange"), o("produce", "Fruit and veg", "green"), o("tea", "Tea", "purple"), o("pantry", "Pantry", "pink"), o("packaging", "Packaging", "gray")}},
					{ID: "unit", Name: "Unit", Type: "text"},
					{ID: "have", Name: "In stock", Type: "number"},
					{ID: "par", Name: "Par level", Type: "number"},
					{ID: "level", Name: "Stock level", Type: "formula", Expr: `round(prop("In stock") / prop("Par level"), 2)`, Format: "percent", Display: "bar", Max: 1},
					{ID: "action", Name: "Action", Type: "formula", Expr: `if(prop("Stock level") < 0.4, "Reorder now", if(prop("Stock level") < 0.7, "Order this week", "Fine"))`},
					{ID: "cost", Name: "Cost per unit", Type: "number", Format: "dollar", Precision: 2},
					{ID: "value", Name: "Stock value", Type: "formula", Expr: `round(prop("In stock") * prop("Cost per unit"), 2)`, Format: "dollar"},
					{ID: "lead", Name: "Lead time", Type: "rollup", Relation: "supplier", Target: "lead", Fn: "max"},
				},
				views: []view{
					{Name: "Stock gauges", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"s1","size":"s","title":"Espresso beans (kg)","chart":{"kind":"gauge","fn":"sum","prop":"have","target":20,"filter":{"prop":"title","cond":"is","value":"Espresso beans, house blend"}}},` +
						`{"id":"s2","size":"s","title":"Whole milk (litres)","chart":{"kind":"gauge","fn":"sum","prop":"have","target":60,"filter":{"prop":"title","cond":"is","value":"Whole milk"}}},` +
						`{"id":"s3","size":"s","title":"Oat milk (litres)","chart":{"kind":"gauge","fn":"sum","prop":"have","target":30,"filter":{"prop":"title","cond":"is","value":"Oat milk"}}},` +
						`{"id":"s4","size":"s","title":"Takeaway cups (sleeves)","chart":{"kind":"gauge","fn":"sum","prop":"have","target":30,"filter":{"prop":"title","cond":"is","value":"Takeaway cups, 12 oz"}}},` +
						`{"id":"s5","wide":true,"title":"Running low: items under 70% of par","chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"level","sort":"value_asc","labels":true,"filter":{"prop":"level","cond":"lt","value":0.7}}},` +
						`{"id":"s6","size":"s","title":"Stock on the shelves","chart":{"kind":"number","fn":"sum","prop":"value"}},` +
						`{"id":"s7","size":"s","title":"Items to reorder now","chart":{"kind":"number","fn":"count","filter":{"prop":"action","cond":"is","value":"Reorder now"}}},` +
						`{"id":"s8","title":"Stock value by category","chart":{"kind":"donut","x":{"prop":"cat"},"fn":"sum","prop":"value","legend":true}}` +
						`]}}`},
					{Name: "Reorder board", Type: "board", Config: `{"group":{"prop":"cat"},"layout":{"size":"small"},` + xbShow("level", "action", "supplier") + `}`},
					{Name: "Stock list", Type: "table", Config: `{"sorts":[{"prop":"level","dir":"asc"}],` + xbCols("title:230", "have:90", "par:90", "unit:110", "level:150", "action:140", "supplier:200", "cost:110", "value:110", "lead:90", "cat:130") + `,"calcs":{"value":"sum"}}`},
				},
				rows: strows,
			},
			{
				key: "cafe-menu", Name: "Menu", Icon: "i:list:orange", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Item", Type: "title"},
					{ID: "cat", Name: "Section", Type: "select", Options: []opt{o("coffee", "Coffee", "brown"), o("tea", "Tea", "purple"), o("cold", "Cold drinks", "blue"), o("bakery", "Bakery", "orange"), o("breakfast", "Breakfast", "yellow"), o("lunch", "Lunch", "green")}},
					{ID: "price", Name: "Price", Type: "number", Format: "dollar", Precision: 2},
					{ID: "cost", Name: "Ingredient cost", Type: "number", Format: "dollar", Precision: 2},
					{ID: "margin", Name: "Margin", Type: "formula", Expr: `round((prop("Price") - prop("Ingredient cost")) / prop("Price"), 2)`, Format: "percent", Display: "bar", Max: 1},
					{ID: "profit", Name: "Profit each", Type: "formula", Expr: `round(prop("Price") - prop("Ingredient cost"), 2)`, Format: "dollar"},
					{ID: "sold", Name: "Sold last week", Type: "number"},
					{ID: "weekly", Name: "Weekly profit", Type: "formula", Expr: `round(prop("Profit each") * prop("Sold last week"), 2)`, Format: "dollar"},
					{ID: "vegan", Name: "Vegan", Type: "checkbox"},
					{ID: "uses", Name: "Uses", Type: "relation", Collection: "@cafe-stock"},
				},
				views: []view{
					{Name: "Menu and margins", Type: "table", Config: `{"group":{"prop":"cat"},` + xbCols("title:230", "price:90", "cost:130", "margin:150", "profit:110", "sold:120", "weekly:120", "vegan:80", "uses:240") + `,"calcs":{"weekly":"sum","sold":"sum"}}`},
					{Name: "Weekly profit", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"series":{"prop":"cat"},"fn":"sum","prop":"weekly","sort":"value_desc","stacked":true,"legend":true}}}`},
					{Name: "Margins by section", Type: "chart", Config: `{"layout":{"chart":{"kind":"dots","x":{"prop":"cat"},"prop":"margin"}}}`},
					{Name: "Profit share", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"cat"},"fn":"sum","prop":"weekly","labels":true,"legend":true}}}`},
					{Name: "Menu board", Type: "gallery", Config: `{"layout":{"cover":"none","size":"small"},"group":{"prop":"cat"},` + xbShow("price", "margin", "vegan") + `}`},
					{Name: "What goes into what", Type: "graph", Config: `{"layout":{"graph":{"color":"cat","links":["uses"],"external":true,"labels":"auto"}}}`},
				},
				rows: mrows,
			},
			{
				key: "cafe-sales", Name: "Daily sales", Icon: "i:chart-line:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Day", Type: "title"},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "revenue", Name: "Takings", Type: "number", Format: "dollar", Precision: 2},
					{ID: "customers", Name: "Customers", Type: "number"},
					{ID: "ticket", Name: "Average spend", Type: "formula", Expr: `round(prop("Takings") / prop("Customers"), 2)`, Format: "dollar"},
					{ID: "weather", Name: "Weather", Type: "select", Options: []opt{o("sunny", "Sunny", "yellow"), o("cloudy", "Cloudy", "gray"), o("rainy", "Rainy", "blue")}},
					{ID: "staff", Name: "Staff on shift", Type: "number"},
					{ID: "top", Name: "Best seller", Type: "relation", Collection: "@cafe-menu"},
				},
				views: []view{
					{Name: "Sales dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"d1","title":"Takings, last 60 days","kpi":{"date":"date","by":"day"},"chart":{"kind":"number","fn":"sum","prop":"revenue"}},` +
						`{"id":"d2","size":"s","title":"Customers","chart":{"kind":"number","fn":"sum","prop":"customers"}},` +
						`{"id":"d3","size":"s","title":"Typical day","chart":{"kind":"number","fn":"median","prop":"revenue"}},` +
						`{"id":"d4","wide":true,"title":"Takings per day","chart":{"kind":"line","x":{"prop":"date"},"fn":"sum","prop":"revenue","smooth":true}},` +
						`{"id":"d5","title":"This week against last week","chart":{"kind":"compare","x":{"prop":"date"},"fn":"sum","prop":"revenue","window":7,"smooth":true}},` +
						`{"id":"d6","title":"Busy days of the week","chart":{"kind":"heatmap","x":{"prop":"date","by":"weekday"},"series":{"prop":"date","by":"week"},"fn":"sum","prop":"revenue"}},` +
						`{"id":"d7","title":"Takings by weather","chart":{"kind":"bar","x":{"prop":"weather"},"fn":"average","prop":"revenue","labels":true}},` +
						`{"id":"d8","title":"Best sellers","chart":{"kind":"donut","x":{"prop":"top"},"fn":"count","legend":true}}` +
						`]}}`},
					{Name: "Takings calendar", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":3,"fn":"sum","value":"revenue","color":"amber"}}}`},
					{Name: "By week", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"week"},"series":{"prop":"weather"},"fn":"sum","prop":"revenue","stacked":true,"legend":true}}}`},
					{Name: "Weekday averages", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"weekday"},"fn":"average","prop":"revenue","labels":true}}}`},
					{Name: "Till log", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"revenue":"sum","customers":"sum"}}`},
				},
				rows: sales,
			},
			{
				key: "cafe-opening", Name: "Opening and closing checklist", Icon: "i:checklist:green", Kind: "doc",
				markdown: `First person in does the opening list. Last person out does the closing list. Tick as you go, untick everything at the end of the day.

## Opening, 6:30 to 7:00

` + fence + `mermaid
flowchart TD
    A[Alarm off, lights on] --> B[Machine on, grinders on]
    B --> C[Dial in espresso]
    B --> D[Bake croissants]
    C --> E[Taste test]
    D --> F[Fill the counter]
    E --> G{Ready at 7:00?}
    F --> G
    G -->|yes| H[Doors open]
    G -->|no| I[Call the manager]
` + fence + `

- [ ] Disarm the alarm, lights and music on
- [ ] Espresso machine and both grinders on (needs 20 minutes to heat)
- [ ] Croissants in the oven: 18 minutes at 180 °C
- [ ] Fridge temperatures logged (must be 5 °C or below)
- [ ] Dial in the espresso, see {{page:cafe-espresso}}
- [ ] Milk, oat milk and syrups stocked at the bar
- [ ] Float counted into the till
- [ ] Chairs down, tables wiped, sign out on the pavement

> [!WARNING]
> If any fridge reads above 8 °C, move the dairy to the back fridge and call Lucia before opening.

## Closing, 17:00 to 17:45

- [ ] Backflush the machine with cleaner, soak the portafilters
- [ ] Empty and brush the grinders' chutes
- [ ] Count the till and enter today's numbers in {{page:cafe-sales}}
- [ ] Check stock against par and update {{page:cafe-stock}}
- [ ] Floors swept and mopped, bins out
- [ ] Alarm on, door double locked

## Weekly

| Day | Job | Who |
| --- | --- | --- |
| Monday | Deep clean the fridges | Opening shift |
| Tuesday | Coffee delivery, rotate stock | Lucia |
| Thursday | Descale the kettle and dishwasher | Closing shift |
| Saturday | Windows and outside tables | Opening shift |
`,
			},
			{
				key: "cafe-recipes", Name: "Recipes and drink specs", Icon: "i:book:amber", Kind: "doc",
				markdown: `Every drink and plate made the same way, whoever is on shift.

## Coffee

| Drink | Espresso | Milk | Cup |
| --- | --- | --- | --- |
| Espresso | 18 g in, 36 g out, 27 to 30 s | none | 3 oz |
| Flat white | Double | 120 ml, thin silky foam | 6 oz |
| Cappuccino | Double | 150 ml, 1.5 cm foam | 8 oz |
| Latte | Double | 240 ml, 1 cm foam | 12 oz |
| Mocha | Double plus 20 g chocolate | 220 ml | 12 oz |

## Kitchen

<details>
<summary>Grilled cheese and tomato soup</summary>

Two slices of sourdough, butter outside, 60 g cheddar inside. Press for 4 minutes. Serve with a cup of the day's tomato soup and a pickle.

</details>

<details>
<summary>Harvest salad bowl</summary>

Leaves, sliced apple, roast squash, toasted seeds, honey mustard dressing. Ask about allergies, the seeds contain sesame.

</details>

> [!CAUTION]
> Allergen sheet is behind the counter. Always check it, never guess.
`,
				children: []Starter{
					{
						key: "cafe-espresso", Name: "Dialling in the espresso", Icon: "i:coffee:orange", Kind: "doc",
						markdown: `Do this every morning and whenever a new bag is opened.

1. Weigh 18 g of ground coffee into the basket.
2. Pull the shot and stop at 36 g in the cup.
3. Time it. The target is 27 to 30 seconds.

| Shot time | Taste | Change |
| --- | --- | --- |
| Under 25 s | Sour, thin | Grind finer |
| 27 to 30 s | Sweet, balanced | Leave it |
| Over 32 s | Bitter, dry | Grind coarser |

> [!TIP]
> Change the grinder one small step at a time and pull a new shot after each change.

Write the setting on the whiteboard so the next barista starts from it.
`,
					},
				},
			},
		},
	}
}

func xbCourse() Starter {
	rnd := &xbRand{s: 2026}
	type stu struct {
		name, group string
		ability     float64
		support     bool
	}
	students := []stu{
		{"Amara Okafor", "a", 88, false}, {"Ben Hollis", "a", 64, false}, {"Chloe Martin", "a", 79, false}, {"Daniel Kowalski", "a", 52, true},
		{"Esme Park", "b", 93, false}, {"Farid Haddad", "b", 71, false}, {"Grace Lindqvist", "b", 67, false}, {"Harvey Bell", "b", 58, false},
		{"Isla Mensah", "c", 82, false}, {"Jacob Rees", "c", 47, true}, {"Kiara Singh", "c", 76, false}, {"Leo Fischer", "c", 69, false},
		{"Maya Duarte", "d", 85, false}, {"Noah Brennan", "d", 61, false}, {"Olivia Chen", "d", 74, false},
	}
	srows := make([]row, 0, len(students))
	for _, s := range students {
		first := strings.ToLower(strings.Fields(s.name)[0])
		srows = append(srows, row{Title: s.name, Values: map[string]any{"group": s.group, "email": first + ".student@example.com", "parent": first + ".family@example.com", "support": s.support}})
	}
	srows[3].Markdown = "Extra time in tests (25%). Prefers a printed copy of the slides.\n"
	srows[9].Markdown = "Reading support: give instructions out loud as well as written. Sits near the front.\n"
	type asg struct {
		title, kind, status string
		due                 int
		points, weight      float64
		diff                float64
	}
	asgs := []asg{
		{"Homework 1: Speed and velocity", "homework", "graded", -40, 20, 0.05, 6},
		{"Lab 1: Measuring speed with light gates", "lab", "graded", -33, 30, 0.10, 2},
		{"Quiz 1: Forces and Newton's laws", "quiz", "graded", -31, 25, 0.10, -6},
		{"Homework 2: Free body diagrams", "homework", "graded", -24, 20, 0.05, 1},
		{"Lab 2: Stretching springs", "lab", "marking", -10, 30, 0.10, 0},
		{"Test: Forces and motion", "test", "set", 6, 60, 0.25, 0},
		{"Project: Egg drop challenge", "project", "set", 20, 40, 0.20, 0},
		{"Quiz 2: Energy stores", "quiz", "set", 27, 25, 0.15, 0},
	}
	setOn := []int{-43, -38, -34, -27, -15, -6, 4, 22}
	arows := make([]row, 0, len(asgs))
	for i, a := range asgs {
		arows = append(arows, row{Title: a.title, Values: map[string]any{"type": a.kind, "status": a.status, "due": xbSpan(xbWeekday(setOn[i]), xbWeekday(a.due)), "points": a.points, "weight": a.weight}})
	}
	arows[4].Markdown = "## Task\n\nHang masses from a spring, measure the extension and plot force against extension. Find the spring constant from the gradient.\n\n$$F = k\\,x$$\n\n## Marking\n\n| Part | Marks |\n| --- | --- |\n| Table of results | 8 |\n| Graph with line of best fit | 10 |\n| Spring constant with units | 6 |\n| Evaluation | 6 |\n"
	feedback := []string{"Clear working, well done.", "Check your units.", "Show every step of the calculation.", "Great graph, label the axes next time.", "Careful with significant figures.", "Excellent explanation.", "Re-read the question before answering.", "Good effort, see me about part 3."}
	var grades []row
	for _, a := range asgs[:4] {
		for _, s := range students {
			pct := s.ability + a.diff + rnd.in(-9, 9)
			pct = math.Max(18, math.Min(100, pct))
			score := math.Round(a.points * pct / 100)
			grades = append(grades, row{Title: strings.Fields(s.name)[0] + " · " + strings.SplitN(a.title, ":", 2)[0], Values: map[string]any{"student": s.name, "assignment": a.title, "score": score, "late": rnd.f() < 0.08, "feedback": feedback[rnd.n(0, len(feedback)-1)]}})
		}
	}
	type lesson struct {
		day                int
		title, kind, topic string
		hw                 string
	}
	lessons := []lesson{
		{-45, "Distance, displacement and speed", "lesson", "motion", ""},
		{-43, "Velocity and speed-time graphs", "lesson", "motion", "Homework 1: Speed and velocity"},
		{-41, "Acceleration", "lesson", "motion", ""},
		{-38, "Practical: light gates", "lab", "motion", "Lab 1: Measuring speed with light gates"},
		{-36, "Distance-time and velocity-time graphs", "lesson", "motion", ""},
		{-34, "Forces as vectors", "lesson", "forces", ""},
		{-31, "Quiz and Newton's first law", "test", "forces", "Quiz 1: Forces and Newton's laws"},
		{-29, "Newton's second law, F = ma", "lesson", "forces", ""},
		{-27, "Free body diagrams", "lesson", "forces", "Homework 2: Free body diagrams"},
		{-24, "Weight, mass and gravity", "lesson", "forces", ""},
		{-22, "Friction and air resistance", "lesson", "forces", ""},
		{-20, "Trip: science museum forces gallery", "trip", "forces", ""},
		{-17, "Terminal velocity", "lesson", "forces", ""},
		{-15, "Practical: stretching springs", "lab", "forces", "Lab 2: Stretching springs"},
		{-13, "Hooke's law and elastic energy", "lesson", "forces", ""},
		{-10, "Momentum", "lesson", "forces", ""},
		{-8, "Stopping distances and road safety", "lesson", "forces", ""},
		{-6, "Revision: motion", "revision", "motion", ""},
		{-3, "Revision: forces", "revision", "forces", ""},
		{-1, "Exam technique and past questions", "revision", "forces", ""},
		{1, "Practice test and feedback", "revision", "forces", ""},
		{4, "Egg drop challenge: design brief", "lab", "energy", "Project: Egg drop challenge"},
		{6, "Test: forces and motion", "test", "forces", "Test: Forces and motion"},
		{8, "Energy stores and transfers", "lesson", "energy", ""},
		{11, "Kinetic and potential energy", "lesson", "energy", ""},
		{13, "Work done and power", "lesson", "energy", ""},
		{15, "Efficiency", "lesson", "energy", ""},
		{18, "Egg drop challenge: build day", "lab", "energy", ""},
		{20, "Egg drop challenge: the drop", "lab", "energy", ""},
		{22, "Conservation of energy", "lesson", "energy", "Quiz 2: Energy stores"},
		{25, "Renewable and non-renewable resources", "lesson", "energy", ""},
		{27, "Quiz 2 and review", "test", "energy", ""},
	}
	lrows := make([]row, 0, len(lessons))
	for _, l := range lessons {
		room := "lab2"
		if l.kind == "lesson" || l.kind == "revision" {
			room = "r14"
		}
		if l.kind == "trip" {
			room = "out"
		}
		v := map[string]any{"date": xbDay(xbWeekday(l.day)), "type": l.kind, "topic": l.topic, "room": room}
		if l.hw != "" {
			v["hw"] = l.hw
		}
		lrows = append(lrows, row{Title: l.title, Values: v})
	}
	lrows[7].Markdown = "## Learning goals\n\n- State Newton's second law\n- Use $F = m a$ to find force, mass or acceleration\n\n## Starter\n\nA 1,200 kg car accelerates at 2 m/s². What is the resultant force?\n\n> [!TIP]\n> Answer: 2,400 N. Ask students to rearrange the equation for each variable.\n"
	return Starter{
		ID: "ex-school-course", Name: "Physics 10: Forces and energy", Description: "One term of a school physics class: students, assignments, grades with averages and a histogram, the lesson calendar and a syllabus with equations.", Icon: "i:cap:blue", Kind: "doc", Category: catExamples, Tags: []string{"school", "grades", "teaching", "syllabus"},
		markdown: `Ms Adeyemi teaches Physics 10 to class 10B. This space holds the whole term: who is in the class, what they have handed in, how they are doing and what is coming up next.

> [!NOTE]
> Grades are worked out from each score and the points the assignment is out of, so the averages and charts stay correct as new marks come in.

## How the class is doing

{{db:sch-grades}}

## Coming up

Lessons, labs and tests for the term. Today's lesson is highlighted.

{{db:sch-lessons}}

## Assignments

{{db:sch-assign}}

## The class

{{db:sch-students}}

## Course pages

- {{page:sch-syllabus}}: what we cover, how it is graded and the key equations
- {{page:sch-safety}}: rules for the lab
- {{page:sch-letter}}: the start-of-term letter to families
`,
		children: []Starter{
			{
				key: "sch-students", Name: "Students", Icon: "i:users:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Student", Type: "title"},
					{ID: "group", Name: "Lab group", Type: "select", Options: []opt{o("a", "Group A", "blue"), o("b", "Group B", "green"), o("c", "Group C", "orange"), o("d", "Group D", "purple")}},
					{ID: "email", Name: "School email", Type: "email"},
					{ID: "parent", Name: "Family contact", Type: "email"},
					{ID: "support", Name: "Learning support plan", Type: "checkbox"},
				},
				views: []view{
					{Name: "Lab groups", Type: "board", Config: `{"group":{"prop":"group"},` + xbShow("support") + `}`},
					{Name: "Class list", Type: "table", Config: `{"sorts":[{"prop":"title","dir":"asc"}]}`},
					{Name: "New student", Type: "form", Config: `{"form":{"title":"Join Physics 10","description":"For students who join the class during the term.","questions":[{"prop":"title","label":"Full name","required":true},{"prop":"email","label":"School email","required":true},{"prop":"parent","label":"Parent or carer email"},{"prop":"group","label":"Lab group (your teacher will tell you)"},{"prop":"support","label":"I have a learning support plan"}],"submit":"Send","done":"Welcome to Physics 10!"}}`},
				},
				rows: srows,
			},
			{
				key: "sch-assign", Name: "Assignments", Icon: "i:clipboard:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Assignment", Type: "title"},
					{ID: "type", Name: "Type", Type: "select", Options: []opt{o("homework", "Homework", "blue"), o("lab", "Lab report", "green"), o("quiz", "Quiz", "yellow"), o("test", "Test", "red"), o("project", "Project", "purple")}},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("set", "Set", "gray", "todo"), so("marking", "Marking", "yellow", "doing"), so("graded", "Graded", "green", "done")}},
					{ID: "due", Name: "Set and due", Type: "date"},
					{ID: "points", Name: "Points", Type: "number"},
					{ID: "weight", Name: "Weight", Type: "number", Format: "percent"},
				},
				views: []view{
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"due","zoom":"week","table":true},` + xbShow("type") + `}`},
					{Name: "Status", Type: "board", Config: `{"group":{"prop":"status"},` + xbShow("type", "due", "weight") + `}`},
					{Name: "All assignments", Type: "table", Config: `{"sorts":[{"prop":"due","dir":"asc"}],"calcs":{"weight":"sum","points":"sum"}}`},
					{Name: "Weighting", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"type"},"fn":"sum","prop":"weight","labels":true,"legend":true}}}`},
				},
				rows: arows,
			},
			{
				key: "sch-grades", Name: "Grades", Icon: "i:chart-bar:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Entry", Type: "title"},
					{ID: "student", Name: "Student", Type: "relation", Collection: "@sch-students"},
					{ID: "assignment", Name: "Assignment", Type: "relation", Collection: "@sch-assign"},
					{ID: "score", Name: "Score", Type: "number"},
					{ID: "outof", Name: "Out of", Type: "rollup", Relation: "assignment", Target: "points", Fn: "max"},
					{ID: "pct", Name: "Percent", Type: "formula", Expr: `round(prop("Score") / prop("Out of") * 100)`, Display: "bar", Max: 100},
					{ID: "grade", Name: "Grade", Type: "formula", Expr: `ifs(prop("Percent") >= 85, "A", prop("Percent") >= 70, "B", prop("Percent") >= 55, "C", prop("Percent") >= 40, "D", "E")`},
					{ID: "late", Name: "Handed in late", Type: "checkbox"},
					{ID: "feedback", Name: "Feedback", Type: "text"},
				},
				views: []view{
					{Name: "Class overview", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"c1","size":"s","title":"Class median","chart":{"kind":"number","fn":"median","prop":"pct"}},` +
						`{"id":"c2","size":"s","title":"Marks entered","chart":{"kind":"number","fn":"count"}},` +
						`{"id":"c3","size":"s","title":"Handed in late","chart":{"kind":"number","fn":"percent_checked","prop":"late"}},` +
						`{"id":"c4","size":"s","title":"Top score","chart":{"kind":"number","fn":"max","prop":"pct"}},` +
						`{"id":"c5","title":"Spread of marks","chart":{"kind":"histogram","x":{"prop":"pct","step":10},"fn":"count","labels":true}},` +
						`{"id":"c6","title":"Average by assignment","chart":{"kind":"bar","x":{"prop":"assignment"},"fn":"average","prop":"pct","labels":true}},` +
						`{"id":"c7","wide":true,"title":"Each student's average","chart":{"kind":"hbar","x":{"prop":"student"},"fn":"average","prop":"pct","sort":"value_desc","labels":true}},` +
						`{"id":"c8","title":"Grades","chart":{"kind":"donut","x":{"prop":"grade"},"fn":"count","labels":true,"legend":true}},` +
						`{"id":"c9","title":"Marks by assignment","chart":{"kind":"dots","x":{"prop":"assignment"},"prop":"pct"}}` +
						`]}}`},
					{Name: "Gradebook", Type: "table", Config: `{"group":{"prop":"assignment"},` + xbCols("title:180", "student:170", "score:80", "outof:80", "pct:160", "grade:80", "late:120", "feedback:260", "-assignment") + `,"calcs":{"pct":"median"}}`},
					{Name: "Student progress", Type: "chart", Config: `{"layout":{"chart":{"kind":"heatmap","x":{"prop":"assignment"},"series":{"prop":"student"},"fn":"average","prop":"pct","labels":true}}}`},
					{Name: "By student", Type: "board", Config: `{"group":{"prop":"student"},"layout":{"total":{"fn":"average","prop":"pct"},"size":"small"},` + xbShow("pct", "grade") + `}`},
				},
				rows: grades,
			},
			{
				key: "sch-lessons", Name: "Class schedule", Icon: "i:calendar:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Lesson", Type: "title"},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "type", Name: "Type", Type: "select", Options: []opt{o("lesson", "Lesson", "blue"), o("lab", "Practical", "green"), o("test", "Test", "red"), o("revision", "Revision", "yellow"), o("trip", "Trip", "purple")}},
					{ID: "topic", Name: "Unit", Type: "select", Options: []opt{o("motion", "Motion", "orange"), o("forces", "Forces", "blue"), o("energy", "Energy", "green")}},
					{ID: "room", Name: "Room", Type: "select", Options: []opt{o("r14", "Room 14", "gray"), o("lab2", "Science lab 2", "green"), o("out", "Off site", "purple")}},
					{ID: "hw", Name: "Assignment set", Type: "relation", Collection: "@sch-assign"},
				},
				views: []view{
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"},` + xbShow("type") + `}`},
					{Name: "Term plan", Type: "table", Config: `{"group":{"prop":"topic"},"sorts":[{"prop":"date","dir":"asc"}]}`},
					{Name: "Lessons by unit", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"topic"},"series":{"prop":"type"},"fn":"count","stacked":true,"legend":true}}}`},
				},
				rows: lrows,
			},
			{
				key: "sch-syllabus", Name: "Syllabus", Icon: "i:book:blue", Kind: "doc",
				markdown: `**Course:** Physics 10 · **Teacher:** Ms Adeyemi · **Room:** 14 and Science lab 2 · **Term:** autumn

<!-- toc -->

## What we will learn

` + fence + `mermaid
flowchart LR
    M[Unit 1: Motion] --> F[Unit 2: Forces]
    F --> E[Unit 3: Energy]
    M -.->|graphs| E
    F -.->|work done| E
` + fence + `

| Unit | Weeks | Big question |
| --- | --- | --- |
| Motion | 1 to 3 | How do we describe how things move? |
| Forces | 4 to 9 | Why do things speed up, slow down or change direction? |
| Energy | 10 to 13 | Where does energy go when something moves? |

## Key equations

| Quantity | Equation | Units |
| --- | --- | --- |
| Speed | $v = \dfrac{d}{t}$ | m/s |
| Acceleration | $a = \dfrac{\Delta v}{t}$ | m/s² |
| Newton's second law | $F = m\,a$ | N |
| Weight | $W = m\,g$ | N |
| Spring force | $F = k\,x$ | N |
| Kinetic energy | $E_k = \tfrac{1}{2} m v^2$ | J |
| Gravitational potential energy | $E_p = m\,g\,h$ | J |

Combining the last two for a falling object with no air resistance:

$$m g h = \frac{1}{2} m v^2 \quad\Rightarrow\quad v = \sqrt{2 g h}$$

## How you are graded

| Type | Share of the final grade |
| --- | --- |
| Homework | 10% |
| Lab reports | 20% |
| Quizzes | 25% |
| Test | 25% |
| Project | 20% |

| Grade | Percent |
| --- | --- |
| A | 85 and above |
| B | 70 to 84 |
| C | 55 to 69 |
| D | 40 to 54 |
| E | below 40 |

> [!IMPORTANT]
> Late work loses 10% per day unless you agreed an extension before the deadline.

## Frequently asked

<details>
<summary>What do I need to bring?</summary>

A scientific calculator, a ruler, a pencil and your exercise book. Lab coats and goggles are provided.

</details>

<details>
<summary>I missed a lesson. What now?</summary>

Check the class schedule for the lesson notes, then come to the Thursday lunchtime catch-up in Room 14.

</details>
`,
				children: []Starter{
					{
						key: "sch-safety", Name: "Lab safety rules", Icon: "i:shield:rose", Kind: "doc",
						markdown: `Read these before your first practical and sign the sheet on the lab door.

> [!CAUTION]
> Goggles on whenever anyone in the room is doing a practical, not just you.

- [ ] Bags under the benches, never in the walkways
- [ ] Long hair tied back
- [ ] No food or drink in the lab
- [ ] Report every breakage and spill to the teacher at once
- [ ] Masses on springs: stand up, and keep feet clear of the hanging weights
- [ ] Wash your hands before you leave

| If this happens | Do this |
| --- | --- |
| Something breaks | Step back, tell the teacher, do not pick up glass |
| Chemical in the eye | Eye wash station by the door, rinse for 10 minutes |
| Fire alarm | Leave everything, exit by the back door, line up on the field |
`,
					},
				},
			},
			{
				key: "sch-letter", Name: "Letter to families", Icon: "i:mail:amber", Kind: "doc",
				markdown: `Dear families,

Welcome to Physics 10. This term we study motion, forces and energy, with plenty of hands-on practical work and a class egg drop challenge near the end of term.

**How you can help at home**

- Ask your child to explain one thing they learned this week. Teaching it is the best way to remember it.
- Make sure they have a scientific calculator.
- Check the assignment list together on Sunday evenings.

**Dates for your diary**

| What | When |
| --- | --- |
| Test on forces and motion | In about a week |
| Egg drop challenge | In three weeks, families welcome to watch |
| Parents' evening | End of term |

If you have any questions, reply to this letter or email physics@example.com.

Kind regards,

Ms Adeyemi
`,
			},
		},
	}
}

func xbRenovation() Starter {
	type firm struct {
		name, trade, phone string
		rating             float64
		insured            bool
	}
	firms := []firm{
		{"Brightwire Electrical", "electric", "0161 496 0101", 5, true},
		{"Oakline Joinery", "joiner", "0161 496 0102", 4.5, true},
		{"Clearflow Plumbing", "plumber", "0161 496 0103", 4.5, true},
		{"Riverside Plumbing and Heating", "plumber", "0161 496 0104", 4, true},
		{"Smooth Finish Plastering", "plaster", "0161 496 0105", 4.5, true},
		{"Tilecraft Studio", "tiler", "0161 496 0106", 5, true},
		{"Hartley Building Services", "builder", "0161 496 0107", 4, true},
		{"Fresh Coat Decorators", "decorator", "0161 496 0108", 4, false},
		{"Greenleaf Gardens", "garden", "0161 496 0109", 4.5, true},
		{"Northside Skip Hire", "waste", "0161 496 0110", 3.5, true},
	}
	frows := make([]row, 0, len(firms))
	for _, f := range firms {
		slug := strings.ToLower(strings.Fields(f.name)[0])
		frows = append(frows, row{Title: f.name, Values: map[string]any{"trade": f.trade, "phone": f.phone, "email": "office@" + slug + ".example.com", "rating": f.rating, "insured": f.insured}})
	}
	frows[0].Markdown = "Did the whole-house rewire survey. Tidy, on time, explains everything. Ask for Gemma.\n"
	frows[5].Markdown = "Booked six weeks ahead. Needs the room tanked and dry before starting the bathroom.\n"
	quotes := []row{
		{Title: "Kitchen fitting: Oakline", Values: map[string]any{"firm": "Oakline Joinery", "job": "kitchen", "amount": 3400, "status": "accepted", "received": "-75d", "valid": "-15d"}},
		{Title: "Kitchen fitting: Hartley", Values: map[string]any{"firm": "Hartley Building Services", "job": "kitchen", "amount": 3950, "status": "declined", "received": "-72d", "valid": "-12d"}},
		{Title: "Rewire kitchen and hall: Brightwire", Values: map[string]any{"firm": "Brightwire Electrical", "job": "electric", "amount": 2100, "status": "accepted", "received": "-80d", "valid": "-20d"}},
		{Title: "Rewire kitchen and hall: Hartley", Values: map[string]any{"firm": "Hartley Building Services", "job": "electric", "amount": 2450, "status": "declined", "received": "-77d", "valid": "-17d"}},
		{Title: "Bathroom plumbing: Clearflow", Values: map[string]any{"firm": "Clearflow Plumbing", "job": "plumbing", "amount": 1850, "status": "accepted", "received": "-50d", "valid": "+10d"}},
		{Title: "Bathroom plumbing: Riverside", Values: map[string]any{"firm": "Riverside Plumbing and Heating", "job": "plumbing", "amount": 1700, "status": "declined", "received": "-48d", "valid": "+12d"}, Markdown: "Cheapest, but cannot start until three weeks after the date we need.\n"},
		{Title: "Plastering: Smooth Finish", Values: map[string]any{"firm": "Smooth Finish Plastering", "job": "plaster", "amount": 1600, "status": "accepted", "received": "-70d", "valid": "-10d"}},
		{Title: "Plastering: Hartley", Values: map[string]any{"firm": "Hartley Building Services", "job": "plaster", "amount": 1750, "status": "declined", "received": "-69d", "valid": "-9d"}},
		{Title: "Bathroom tiling: Tilecraft", Values: map[string]any{"firm": "Tilecraft Studio", "job": "tiling", "amount": 1450, "status": "accepted", "received": "-45d", "valid": "+15d"}},
		{Title: "Bathroom tiling: Hartley", Values: map[string]any{"firm": "Hartley Building Services", "job": "tiling", "amount": 1300, "status": "declined", "received": "-44d", "valid": "+16d"}},
		{Title: "Decorating downstairs: Fresh Coat", Values: map[string]any{"firm": "Fresh Coat Decorators", "job": "decor", "amount": 1900, "status": "received", "received": "-6d", "valid": "+24d"}},
		{Title: "Decorating downstairs: Hartley", Values: map[string]any{"firm": "Hartley Building Services", "job": "decor", "amount": 2300, "status": "received", "received": "-4d", "valid": "+26d"}},
		{Title: "New patio: Greenleaf", Values: map[string]any{"firm": "Greenleaf Gardens", "job": "garden", "amount": 2200, "status": "received", "received": "-3d", "valid": "+27d"}},
		{Title: "New patio: Hartley", Values: map[string]any{"firm": "Hartley Building Services", "job": "garden", "amount": 2600, "status": "requested", "received": "-1d", "valid": "+29d"}},
		{Title: "Built-in wardrobes: Oakline", Values: map[string]any{"firm": "Oakline Joinery", "job": "wardrobe", "amount": 1800, "status": "received", "received": "-9d", "valid": "+21d"}},
		{Title: "Skips for the whole job", Values: map[string]any{"firm": "Northside Skip Hire", "job": "waste", "amount": 780, "status": "accepted", "received": "-82d", "valid": "-22d"}},
	}
	type tk struct {
		title, room, firm, status string
		a, b                      int
		cost                      float64
	}
	tlist := []tk{
		{"Strip out the old kitchen", "kitchen", "Hartley Building Services", "done", -42, -39, 450},
		{"First fix electrics, kitchen", "kitchen", "Brightwire Electrical", "done", -38, -35, 900},
		{"First fix plumbing, kitchen", "kitchen", "Clearflow Plumbing", "done", -37, -35, 380},
		{"Replaster kitchen walls", "kitchen", "Smooth Finish Plastering", "done", -32, -29, 950},
		{"Level the floor and lay tiles", "kitchen", "Tilecraft Studio", "done", -27, -23, 620},
		{"Fit kitchen units", "kitchen", "Oakline Joinery", "done", -21, -15, 2400},
		{"Template and fit oak worktops", "kitchen", "Oakline Joinery", "done", -13, -7, 1000},
		{"Tile the splashback", "kitchen", "Tilecraft Studio", "doing", -4, -2, 280},
		{"Second fix electrics, kitchen", "kitchen", "Brightwire Electrical", "doing", -1, 1, 450},
		{"Connect sink, dishwasher and hob", "kitchen", "Clearflow Plumbing", "todo", 1, 2, 220},
		{"Paint the kitchen", "kitchen", "Fresh Coat Decorators", "todo", 3, 6, 520},
		{"Rewire the hallway", "hall", "Brightwire Electrical", "done", -34, -33, 750},
		{"Patch hallway plaster", "hall", "Smooth Finish Plastering", "done", -28, -27, 300},
		{"Paint hall, stairs and landing", "hall", "Fresh Coat Decorators", "todo", 34, 38, 780},
		{"Overboard the living room ceiling", "living", "Smooth Finish Plastering", "done", -19, -17, 350},
		{"Swap the living room radiator", "living", "Riverside Plumbing and Heating", "done", -15, -15, 160},
		{"Sand and varnish the floorboards", "living", "Hartley Building Services", "todo", 27, 30, 650},
		{"Paint the living room", "living", "Fresh Coat Decorators", "todo", 31, 33, 600},
		{"Strip out the old bathroom", "bath", "Hartley Building Services", "todo", 7, 8, 380},
		{"First fix plumbing, bathroom", "bath", "Clearflow Plumbing", "todo", 9, 11, 950},
		{"Extractor fan and lights, bathroom", "bath", "Brightwire Electrical", "todo", 10, 11, 380},
		{"Board and tank the shower area", "bath", "Smooth Finish Plastering", "todo", 12, 14, 420},
		{"Tile the bathroom", "bath", "Tilecraft Studio", "todo", 15, 20, 1450},
		{"Fit bath, toilet and vanity", "bath", "Clearflow Plumbing", "todo", 21, 23, 900},
		{"Paint the bathroom", "bath", "Fresh Coat Decorators", "todo", 24, 25, 280},
		{"Build the wardrobes", "bedroom", "Oakline Joinery", "todo", 39, 43, 1800},
		{"Paint the main bedroom", "bedroom", "Fresh Coat Decorators", "todo", 44, 46, 520},
		{"Dig out and lay the patio", "garden", "Greenleaf Gardens", "todo", 41, 48, 2200},
		{"Outdoor sockets and lights", "garden", "Brightwire Electrical", "todo", 47, 48, 420},
		{"Skips, first two loads", "garden", "Northside Skip Hire", "done", -42, -20, 520},
		{"Skip, final load", "garden", "Northside Skip Hire", "todo", 45, 50, 260},
	}
	tasks := make([]row, 0, len(tlist))
	for _, t := range tlist {
		tasks = append(tasks, row{Title: t.title, Values: map[string]any{"room": t.room, "firm": t.firm, "status": t.status, "when": xbSpan(t.a, t.b), "cost": t.cost}})
	}
	after := map[string][]string{
		"First fix electrics, kitchen":       {"Strip out the old kitchen"},
		"First fix plumbing, kitchen":        {"Strip out the old kitchen"},
		"Replaster kitchen walls":            {"First fix electrics, kitchen", "First fix plumbing, kitchen"},
		"Level the floor and lay tiles":      {"Replaster kitchen walls"},
		"Fit kitchen units":                  {"Level the floor and lay tiles"},
		"Template and fit oak worktops":      {"Fit kitchen units"},
		"Tile the splashback":                {"Template and fit oak worktops"},
		"Second fix electrics, kitchen":      {"Tile the splashback"},
		"Connect sink, dishwasher and hob":   {"Second fix electrics, kitchen"},
		"Paint the kitchen":                  {"Connect sink, dishwasher and hob"},
		"First fix plumbing, bathroom":       {"Strip out the old bathroom"},
		"Extractor fan and lights, bathroom": {"Strip out the old bathroom"},
		"Board and tank the shower area":     {"First fix plumbing, bathroom", "Extractor fan and lights, bathroom"},
		"Tile the bathroom":                  {"Board and tank the shower area"},
		"Fit bath, toilet and vanity":        {"Tile the bathroom"},
		"Paint the bathroom":                 {"Fit bath, toilet and vanity"},
		"Paint the living room":              {"Sand and varnish the floorboards"},
		"Outdoor sockets and lights":         {"Dig out and lay the patio"},
	}
	for i := range tasks {
		if a, ok := after[tasks[i].Title]; ok {
			tasks[i].Values["after"] = a
		}
	}
	tasks[7].Markdown = "Metro tiles, 75 by 150, white with a pale grey grout. Tilecraft brings their own spacers.\n\n- [x] Tiles delivered\n- [ ] Grout colour confirmed\n"
	type buy struct {
		item, room, shop, status string
		price, qty               float64
		day                      int
	}
	blist := []buy{
		{"Shaker kitchen units, sage green", "kitchen", "Hearth and Home Kitchens", "delivered", 4200, 1, -55},
		{"Solid oak worktops", "kitchen", "Timber Yard Direct", "delivered", 1150, 1, -30},
		{"Induction hob", "kitchen", "Appliance Hub", "delivered", 520, 1, -28},
		{"Built-in oven", "kitchen", "Appliance Hub", "delivered", 610, 1, -28},
		{"Extractor hood", "kitchen", "Appliance Hub", "delivered", 280, 1, -28},
		{"Butler sink and mixer tap", "kitchen", "Bathstore Outlet", "delivered", 340, 1, -26},
		{"Metro wall tiles (m2)", "kitchen", "Tile Warehouse", "delivered", 32, 6, -12},
		{"Porcelain floor tiles (m2)", "kitchen", "Tile Warehouse", "delivered", 38, 14, -33},
		{"Pendant lights", "kitchen", "Lamp and Shade", "ordered", 85, 3, -6},
		{"Brass cabinet handles", "kitchen", "Hearth and Home Kitchens", "delivered", 6.5, 24, -20},
		{"Wall paint, 5 litre", "kitchen", "Paint Place", "to_order", 48, 2, 0},
		{"Freestanding bath", "bath", "Bathstore Outlet", "ordered", 480, 1, -8},
		{"Shower screen", "bath", "Bathstore Outlet", "ordered", 260, 1, -8},
		{"Close-coupled toilet", "bath", "Bathstore Outlet", "ordered", 310, 1, -8},
		{"Vanity unit with basin", "bath", "Bathstore Outlet", "to_order", 540, 1, 0},
		{"Wall tiles, bathroom (m2)", "bath", "Tile Warehouse", "ordered", 42, 22, -5},
		{"Floor tiles, bathroom (m2)", "bath", "Tile Warehouse", "ordered", 45, 5, -5},
		{"Heated towel rail", "bath", "Radiator Centre", "to_order", 190, 1, 0},
		{"Mirror cabinet", "bath", "Lamp and Shade", "to_order", 160, 1, 0},
		{"Column radiator", "living", "Radiator Centre", "delivered", 240, 1, -18},
		{"Floor varnish, 2.5 litre", "living", "Paint Place", "to_order", 45, 3, 0},
		{"Living room paint, 5 litre", "living", "Paint Place", "to_order", 52, 2, 0},
		{"Hallway paint and primer", "hall", "Paint Place", "to_order", 46, 3, 0},
		{"Hall pendant and switches", "hall", "Lamp and Shade", "delivered", 120, 1, -36},
		{"Wardrobe doors", "bedroom", "Timber Yard Direct", "to_order", 95, 6, 0},
		{"Bedroom paint, 5 litre", "bedroom", "Paint Place", "to_order", 48, 2, 0},
		{"Patio slabs (m2)", "garden", "Garden Supplies Co", "to_order", 28, 30, 0},
		{"Sand and cement", "garden", "Garden Supplies Co", "to_order", 180, 1, 0},
		{"Outdoor wall lights", "garden", "Lamp and Shade", "to_order", 35, 4, 0},
	}
	buys := make([]row, 0, len(blist))
	for _, b := range blist {
		v := map[string]any{"room": b.room, "shop": b.shop, "status": b.status, "price": b.price, "qty": b.qty}
		if b.day < 0 {
			v["ordered"] = xbDay(b.day)
		}
		buys = append(buys, row{Title: b.item, Values: v})
	}
	type rm struct {
		id, name string
		budget   float64
	}
	rooms := []rm{{"kitchen", "Kitchen", 18500}, {"bath", "Bathroom", 10500}, {"living", "Living room", 2600}, {"hall", "Hallway and stairs", 2000}, {"bedroom", "Main bedroom", 2900}, {"garden", "Garden", 3800}}
	rrows := make([]row, 0, len(rooms))
	for _, r := range rooms {
		rrows = append(rrows, row{Title: r.name, Values: map[string]any{"budget": r.budget, "tasks": xbWhere(tasks, "room", r.id), "buys": xbWhere(buys, "room", r.id)}})
	}
	rrows[0].Markdown = "Units in, worktops oiled. Waiting on the splashback and second fix. Light switch by the back door moved 30 cm left.\n"
	decisions := []row{
		{Title: "Keep the original floorboards in the living room", Values: map[string]any{"date": "-90d", "room": "living", "by": "both", "impact": -1200, "state": "agreed"}, Markdown: "Sanding and varnishing is cheaper than new flooring and keeps the character of the house.\n"},
		{Title: "Sage green shaker units instead of white gloss", Values: map[string]any{"date": "-85d", "room": "kitchen", "by": "both", "impact": 600, "state": "agreed"}, Markdown: "Costs more but we love it, and it hides marks better.\n"},
		{Title: "Brightwire does all the electrics", Values: map[string]any{"date": "-80d", "room": "hall", "by": "raj", "impact": -350, "state": "agreed"}, Markdown: "Cheaper than Hartley and they found the old fuse box problem on the survey.\n"},
		{Title: "Move the boiler cupboard? No", Values: map[string]any{"date": "-78d", "room": "kitchen", "by": "both", "impact": -1800, "state": "agreed"}, Markdown: "Moving it would add about 1,800 and a week. We redesigned the corner instead.\n"},
		{Title: "Oak worktops, not quartz", Values: map[string]any{"date": "-70d", "room": "kitchen", "by": "hannah", "impact": -900, "state": "agreed"}, Markdown: "Warmer, cheaper and can be sanded if they get marked. Oil every three months.\n"},
		{Title: "Clearflow over Riverside for the bathroom", Values: map[string]any{"date": "-46d", "room": "bath", "by": "both", "impact": 150, "state": "agreed"}, Markdown: "150 more, but Riverside could not start until three weeks after the tiler.\n"},
		{Title: "Freestanding bath instead of a walk-in shower", Values: map[string]any{"date": "-40d", "room": "bath", "by": "hannah", "impact": 300, "state": "agreed"}, Markdown: "With a shower screen over one end so we still get a proper shower.\n"},
		{Title: "Underfloor heating in the bathroom", Values: map[string]any{"date": "-38d", "room": "bath", "by": "raj", "impact": 650, "state": "dropped"}, Markdown: "Nice to have, but the heated towel rail is enough for a small room.\n"},
		{Title: "Pale grey grout for the splashback", Values: map[string]any{"date": "-10d", "room": "kitchen", "by": "hannah", "impact": 0, "state": "agreed"}, Markdown: "White grout showed every coffee splash on the sample board. Pale grey hides it and still looks crisp.\n"},
		{Title: "Paint colours for downstairs", Values: map[string]any{"date": "-7d", "room": "living", "by": "both", "impact": 0, "state": "open"}, Markdown: "Two samples on the wall. Decide by the weekend so Fresh Coat can order.\n"},
		{Title: "Which decorator?", Values: map[string]any{"date": "-3d", "room": "hall", "by": "both", "impact": 400, "state": "open"}, Markdown: "Fresh Coat is 400 cheaper but not insured. Ask for proof of insurance before choosing.\n"},
		{Title: "Patio: slabs or gravel", Values: map[string]any{"date": "-2d", "room": "garden", "by": "raj", "impact": -900, "state": "open"}, Markdown: "Gravel saves about 900 but is harder with a pushchair.\n"},
		{Title: "Wardrobes built in, not freestanding", Values: map[string]any{"date": "-12d", "room": "bedroom", "by": "both", "impact": 800, "state": "agreed"}, Markdown: "The alcoves either side of the chimney are perfect for it, and built-in adds value when we sell.\n"},
		{Title: "Keep the picture rail in the hallway", Values: map[string]any{"date": "-30d", "room": "hall", "by": "hannah", "impact": 0, "state": "agreed"}, Markdown: "It is original to the house. Paint below it dark green and above it pale stone.\n"},
		{Title: "Live upstairs during the kitchen works", Values: map[string]any{"date": "-60d", "room": "kitchen", "by": "both", "impact": 0, "state": "agreed"}, Markdown: "Microwave and kettle in the bedroom, washing up in the bath. Four weeks, we can do it.\n"},
	}
	roomOpts := []opt{o("kitchen", "Kitchen", "green"), o("bath", "Bathroom", "blue"), o("living", "Living room", "orange"), o("hall", "Hallway and stairs", "brown"), o("bedroom", "Main bedroom", "purple"), o("garden", "Garden", "green")}
	return Starter{
		ID: "ex-home-renovation", Name: "Home renovation", Description: "Renovating a 1930s house room by room: tasks on a timeline, purchases, contractor quotes, costs rolled up against each room's budget and a decisions log.", Icon: "i:home:orange", Kind: "doc", Category: catExamples, Tags: []string{"renovation", "budget", "contractors", "home"},
		markdown: `Hannah and Raj bought a tired 1930s semi and are doing it up one room at a time while living in it. The kitchen is nearly finished, the bathroom starts next week and the garden comes last. Three rooms are already heading over budget, which is exactly why this page exists.

## Budget by room

Each room adds up the labour from its tasks and the materials from its purchases, then compares the total with the budget. The bar turns full when a room has spent everything.

{{db:reno-rooms}}

## Schedule

Every job on one timeline, coloured by room. Drag a bar to move it and everything else stays put.

{{db:reno-tasks}}

## Shopping

{{db:reno-buys}}

## Who is doing the work

Quotes side by side, so it is easy to see who was cheapest and who we chose.

{{db:reno-quotes}}

{{db:reno-firms}}

## Why we chose what we chose

{{db:reno-decisions}}

## Pages

- {{page:reno-plan}}: the order of works and who to call first
- {{page:reno-finishes}}: paint colours, tiles and handles
- {{page:reno-punch}}: the snagging list for each room
`,
		children: []Starter{
			{
				key: "reno-firms", Name: "Contractors", Icon: "i:wrench:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Company", Type: "title"},
					{ID: "trade", Name: "Trade", Type: "select", Options: []opt{o("electric", "Electrician", "yellow"), o("joiner", "Joiner", "brown"), o("plumber", "Plumber", "blue"), o("plaster", "Plasterer", "gray"), o("tiler", "Tiler", "purple"), o("builder", "General builder", "orange"), o("decorator", "Decorator", "pink"), o("garden", "Landscaper", "green"), o("waste", "Skip hire", "gray")}},
					{ID: "phone", Name: "Phone", Type: "phone"},
					{ID: "email", Name: "Email", Type: "email"},
					{ID: "rating", Name: "Our rating", Type: "number", Display: "bar", Max: 5},
					{ID: "insured", Name: "Insured", Type: "checkbox"},
				},
				views: []view{
					{Name: "Contractors", Type: "gallery", Config: `{"layout":{"cover":"none","size":"small"},` + xbShow("trade", "rating", "phone") + `}`},
					{Name: "Contact list", Type: "table", Config: `{"sorts":[{"prop":"trade","dir":"asc"}]}`},
				},
				rows: frows,
			},
			{
				key: "reno-quotes", Name: "Quotes", Icon: "i:clipboard:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Quote", Type: "title"},
					{ID: "firm", Name: "Contractor", Type: "relation", Collection: "@reno-firms"},
					{ID: "job", Name: "Job", Type: "select", Options: []opt{o("kitchen", "Kitchen fitting", "green"), o("electric", "Rewire", "yellow"), o("plumbing", "Bathroom plumbing", "blue"), o("plaster", "Plastering", "gray"), o("tiling", "Tiling", "purple"), o("decor", "Decorating", "pink"), o("garden", "Patio", "green"), o("wardrobe", "Wardrobes", "brown"), o("waste", "Skips", "gray")}},
					{ID: "amount", Name: "Amount", Type: "number", Format: "pound"},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("requested", "Requested", "gray", "todo"), so("received", "Received", "yellow", "doing"), so("accepted", "Accepted", "green", "done"), so("declined", "Declined", "red", "done")}},
					{ID: "received", Name: "Received", Type: "date"},
					{ID: "valid", Name: "Valid until", Type: "date"},
					{ID: "rating", Name: "Their rating", Type: "rollup", Relation: "firm", Target: "rating", Fn: "max"},
				},
				views: []view{
					{Name: "Compare quotes", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"series":{"prop":"status"},"fn":"sum","prop":"amount","stacked":true,"labels":true,"legend":true}}}`},
					{Name: "Quotes", Type: "board", Config: `{"group":{"prop":"status"},"layout":{"total":{"fn":"sum","prop":"amount"}},` + xbShow("amount", "firm", "valid") + `}`},
					{Name: "All quotes", Type: "table", Config: `{"group":{"prop":"job"},"sorts":[{"prop":"amount","dir":"asc"}]}`},
				},
				rows: quotes,
			},
			{
				key: "reno-tasks", Name: "Tasks and schedule", Icon: "i:calendar:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Job", Type: "title"},
					{ID: "room", Name: "Room", Type: "select", Options: roomOpts},
					{ID: "firm", Name: "Who", Type: "relation", Collection: "@reno-firms"},
					{ID: "when", Name: "Dates", Type: "date"},
					{ID: "status", Name: "Status", Type: "status", Options: statusOpts},
					{ID: "cost", Name: "Labour cost", Type: "number", Format: "pound"},
					{ID: "after", Name: "Starts after", Type: "relation", Collection: "@self"},
				},
				views: []view{
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"when","zoom":"week","table":true,"deps":"after"},"sorts":[{"prop":"when","dir":"asc"}],` + xbShow("room") + `}`},
					{Name: "Board", Type: "board", Config: `{"group":{"prop":"status"},"sub_group":{"prop":"room"},"layout":{"total":{"fn":"sum","prop":"cost"}},` + xbShow("when", "firm") + `}`},
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"when"},` + xbShow("room") + `}`},
					{Name: "Labour by room", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"room"},"series":{"prop":"status"},"fn":"sum","prop":"cost","stacked":true,"legend":true,"labels":true}}}`},
					{Name: "All jobs", Type: "table", Config: `{"sorts":[{"prop":"when","dir":"asc"}],"calcs":{"cost":"sum"}}`},
				},
				rows: tasks,
			},
			{
				key: "reno-buys", Name: "Purchases", Icon: "i:cart:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Item", Type: "title"},
					{ID: "room", Name: "Room", Type: "select", Options: roomOpts},
					{ID: "shop", Name: "Supplier", Type: "text"},
					{ID: "price", Name: "Unit price", Type: "number", Format: "pound", Precision: 2},
					{ID: "qty", Name: "Quantity", Type: "number"},
					{ID: "total", Name: "Total", Type: "formula", Expr: `round(prop("Unit price") * prop("Quantity"), 2)`, Format: "pound", Precision: 2},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("to_order", "To order", "gray", "todo"), so("ordered", "Ordered", "yellow", "doing"), so("delivered", "Delivered", "green", "done")}},
					{ID: "ordered", Name: "Ordered on", Type: "date"},
				},
				views: []view{
					{Name: "Spending", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"p1","size":"s","title":"Materials total","chart":{"kind":"number","fn":"sum","prop":"total"}},` +
						`{"id":"p2","size":"s","title":"Still to order","chart":{"kind":"number","fn":"sum","prop":"total","filter":{"prop":"status","cond":"is","value":"to_order"}}},` +
						`{"id":"p3","size":"s","title":"Items delivered","chart":{"kind":"number","fn":"count","filter":{"prop":"status","cond":"is","value":"delivered"}}},` +
						`{"id":"p4","size":"s","title":"Items waiting","chart":{"kind":"number","fn":"count","filter":{"prop":"status","cond":"is_not","value":"delivered"}}},` +
						`{"id":"p5","title":"Materials by room","chart":{"kind":"donut","x":{"prop":"room"},"fn":"sum","prop":"total","legend":true}},` +
						`{"id":"p6","title":"By status","chart":{"kind":"hbar","x":{"prop":"room"},"series":{"prop":"status"},"fn":"sum","prop":"total","stacked":true,"legend":true}}` +
						`]}}`},
					{Name: "Shopping list", Type: "table", Config: `{"group":{"prop":"status"},"calcs":{"total":"sum"}}`},
					{Name: "By room", Type: "board", Config: `{"group":{"prop":"room"},"layout":{"total":{"fn":"sum","prop":"total"}},` + xbShow("total", "status") + `}`},
				},
				rows: buys,
			},
			{
				key: "reno-rooms", Name: "Rooms", Icon: "i:home:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Room", Type: "title"},
					{ID: "budget", Name: "Budget", Type: "number", Format: "pound"},
					{ID: "tasks", Name: "Jobs", Type: "relation", Collection: "@reno-tasks"},
					{ID: "buys", Name: "Purchases", Type: "relation", Collection: "@reno-buys"},
					{ID: "labour", Name: "Labour", Type: "rollup", Relation: "tasks", Target: "cost", Fn: "sum", Format: "pound"},
					{ID: "materials", Name: "Materials", Type: "rollup", Relation: "buys", Target: "total", Fn: "sum", Format: "pound"},
					{ID: "cost", Name: "Total cost", Type: "formula", Expr: `prop("Labour") + prop("Materials")`, Format: "pound"},
					{ID: "used", Name: "Of budget", Type: "formula", Expr: `round(prop("Total cost") / prop("Budget"), 2)`, Format: "percent", Display: "bar", Max: 1},
					{ID: "left", Name: "Left", Type: "formula", Expr: `prop("Budget") - prop("Total cost")`, Format: "pound"},
				},
				views: []view{
					{Name: "Budget by room", Type: "table", Config: `{` + xbCols("title:170", "budget:110", "labour:110", "materials:120", "cost:120", "used:160", "left:110", "-tasks", "-buys") + `,"calcs":{"budget":"sum","labour":"sum","materials":"sum","cost":"sum"}}`},
					{Name: "Total cost by room", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"cost","sort":"value_desc","labels":true}}}`},
					{Name: "Where the money goes", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"title"},"fn":"sum","prop":"cost","labels":true,"legend":true}}}`},
				},
				rows: rrows,
			},
			{
				key: "reno-decisions", Name: "Decisions log", Icon: "i:scale:rose", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Decision", Type: "title"},
					{ID: "date", Name: "Decided on", Type: "date"},
					{ID: "room", Name: "Room", Type: "select", Options: roomOpts},
					{ID: "by", Name: "Decided by", Type: "select", Options: []opt{o("both", "Both of us", "purple"), o("hannah", "Hannah", "pink"), o("raj", "Raj", "blue")}},
					{ID: "impact", Name: "Cost impact", Type: "number", Format: "pound"},
					{ID: "state", Name: "Status", Type: "select", Options: []opt{o("open", "Still deciding", "yellow"), o("agreed", "Agreed", "green"), o("dropped", "Dropped", "gray")}},
				},
				views: []view{
					{Name: "Log", Type: "feed", Config: `{"sorts":[{"prop":"date","dir":"desc"}],` + xbShow("date", "room", "state", "impact") + `}`},
					{Name: "Still deciding", Type: "list", Config: `{"filter":{"prop":"state","cond":"is","value":"open"},` + xbShow("room", "impact") + `}`},
					{Name: "All decisions", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"impact":"sum"}}`},
				},
				rows: decisions,
			},
			{
				key: "reno-plan", Name: "Plan of works", Icon: "i:layers:blue", Kind: "doc",
				markdown: `<!-- toc -->

## Order of works

Trades follow each other in a fixed order. Getting this wrong means paying someone twice.

` + fence + `mermaid
flowchart TD
    A[Strip out] --> B[First fix: electrics and plumbing]
    B --> C[Plaster walls and ceilings]
    C --> D[Floors: level and lay]
    D --> E[Fit units, bath, wardrobes]
    E --> F[Tile]
    F --> G[Second fix: sockets, taps, lights]
    G --> H[Decorate]
    H --> I[Snagging and clean]
` + fence + `

## Before anything starts

- [x] Building control notice for the rewire
- [x] Asbestos check on the old artex ceiling (clear)
- [x] Tell the neighbours and agree skip position
- [ ] Party wall letter for the patio

## Money rules

| Rule | Why |
| --- | --- |
| Keep 10% of every room budget as contingency | Old houses always hide something |
| Pay in stages, never more than 30% up front | Protects us if a firm disappears |
| Get three quotes for anything over 1,000 | Prices vary a lot for the same job |

> [!WARNING]
> Never let a trade start without a written quote that lists what is included. "Supply and fit" and "fit only" can differ by thousands.

<details>
<summary>What counts as first fix and second fix?</summary>

First fix is everything hidden inside walls and floors: cables, pipes, back boxes. Second fix is everything you can see afterwards: sockets, switches, taps, lights. Plastering happens in between.

</details>
`,
				children: []Starter{
					{
						key: "reno-finishes", Name: "Colours and finishes", Icon: "i:palette:pink", Kind: "doc",
						markdown: `## Paint

| Room | Walls | Woodwork | Ceiling |
| --- | --- | --- | --- |
| Kitchen | Warm white | Sage green to match units | Brilliant white |
| Living room | Soft clay (sample A) or stone (sample B) | Off white | Brilliant white |
| Hallway | Pale stone | Deep green below the picture rail | Brilliant white |
| Bathroom | Bathroom-grade warm white | White | Bathroom-grade white |

## Tiles

- Kitchen splashback: white metro, 75 by 150, pale grey grout
- Kitchen floor: porcelain, warm grey, 600 by 600
- Bathroom walls: handmade-look green, 100 by 100, white grout

## Hardware

- [x] Brass cup handles for the kitchen
- [ ] Brass light switches for the hallway
- [ ] Black taps for the bathroom

> [!TIP]
> Paint a large sample on two walls and look at it morning and evening before choosing.
`,
					},
				},
			},
			{
				key: "reno-punch", Name: "Punch list", Icon: "i:checklist:orange", Kind: "doc",
				markdown: `The snagging list: small things to fix before each trade gets their final payment.

## Kitchen

- [x] Cupboard above the fridge does not close flush
- [x] Gap between worktop and wall at the window end
- [ ] Silicone around the sink is patchy
- [ ] Plinth missing under the dishwasher
- [ ] Light switch plate not straight

## Hallway

- [x] Hairline crack above the front door
- [ ] Paint drip on the third stair

## Living room

- [ ] Radiator valve drips when turned fully on
- [ ] Ceiling rose not centred over the light

## Bathroom

- [ ] (Starts next week)

> [!IMPORTANT]
> Hold back the last 10% of each payment until every item for that trade is ticked.

| Trade | Open items | Final payment held |
| --- | --- | --- |
| Oakline Joinery | 1 | 340 |
| Clearflow Plumbing | 2 | 185 |
| Brightwire Electrical | 1 | 210 |
`,
			},
		},
	}
}

func xbEvent() Starter {
	rnd := &xbRand{s: 42}
	type spk struct {
		name, org, country, topic, bio string
		travel                         bool
	}
	speakers := []spk{
		{"Marieke Jansen", "Rotterdam Repair Network", "nl", "repair", "Started the city's first repair cafe in a church hall in 2014. Now coordinates forty of them.", false},
		{"Tomasz Wolski", "Open Bench Collective", "pl", "electronics", "Hardware engineer who designs open-source test equipment for community workshops.", true},
		{"Aisha Mwangi", "Nairobi Fab Hub", "ke", "community", "Runs a fab lab that trains young people to build solar kits and repair phones.", true},
		{"Lena Vogt", "Textile Futures Studio", "de", "textiles", "Designer and teacher. Turns old jeans into new jackets and teaches others to do the same.", true},
		{"Pieter de Graaf", "Grachten Woodworks", "nl", "woodwork", "Third-generation cabinet maker who restores canal house furniture.", false},
		{"Camille Laurent", "Atelier Circulaire", "fr", "repair", "Researches how long household appliances last and why some are impossible to open.", true},
		{"Ravi Iyer", "Bengaluru Makers Guild", "in", "electronics", "Builds low-cost air quality sensors with school groups.", true},
		{"Sofie Peeters", "Stadslab Foundation", "be", "community", "Helps cities turn empty shops into shared workshops.", false},
		{"Jonas Berg", "Fixit Stockholm", "se", "repair", "Bicycle mechanic and author of a free guide to fixing anything with gears.", true},
		{"Grace O'Neill", "Dublin Tool Library", "ie", "community", "Founded a library where neighbours borrow drills instead of buying them.", true},
		{"Marta Costa", "Lisbon Ceramics Lab", "pt", "craft", "Potter who teaches kintsugi-style repair of broken ceramics.", true},
		{"Daan Visser", "Delta Circuits", "nl", "electronics", "Teaches microcontrollers to beginners using projects you can finish in an evening.", false},
		{"Elena Rossi", "Milano Riuso", "it", "textiles", "Runs a clothing swap and mending studio in a former tram depot.", true},
		{"Kwame Mensah", "Sunworks Cooperative", "uk", "energy", "Installs community solar and teaches people to read their own energy data.", true},
		{"Ana Ruiz", "Barcelona Maker Faire Club", "es", "craft", "Organises family workshops where kids take apart old toys and build new ones.", true},
		{"Femke Bakker", "Harbour Makers Week", "nl", "community", "Lead organiser of this week. Will be the one with the clipboard.", false},
	}
	srows := make([]row, 0, len(speakers))
	for _, s := range speakers {
		srows = append(srows, row{Title: s.name, Values: map[string]any{"org": s.org, "country": s.country, "topic": s.topic, "travel": s.travel, "email": strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s.name, " ", "."), "'", "")) + "@example.com"}, Markdown: s.bio + "\n"})
	}
	type sess struct {
		title      string
		day, until int
		time, room string
		track      string
		speakers   []string
		cap        float64
	}
	sessions := []sess{
		{"Opening: why we fix things", 0, 0, "09:30", "main", "community", []string{"Femke Bakker", "Marieke Jansen"}, 300},
		{"Repair cafe, open all week", 0, 4, "10:00", "courtyard", "repair", []string{"Marieke Jansen", "Jonas Berg"}, 80},
		{"Soldering for complete beginners", 0, 0, "11:00", "workshop", "electronics", []string{"Daan Visser"}, 24},
		{"Panel: makerspaces that last", 0, 0, "14:00", "main", "community", []string{"Sofie Peeters", "Grace O'Neill", "Aisha Mwangi"}, 300},
		{"Visible mending with darning and patches", 0, 0, "15:30", "studio", "textiles", []string{"Elena Rossi"}, 20},
		{"Student project showcase", 1, 4, "10:00", "gallery", "community", []string{"Femke Bakker"}, 120},
		{"Build an air quality sensor", 1, 1, "10:00", "workshop", "electronics", []string{"Ravi Iyer"}, 24},
		{"Why appliances break, and how to open them", 1, 1, "11:30", "main", "repair", []string{"Camille Laurent"}, 300},
		{"From jeans to jacket", 1, 1, "13:30", "studio", "textiles", []string{"Lena Vogt"}, 18},
		{"Tool libraries: lending instead of buying", 1, 1, "15:00", "main", "community", []string{"Grace O'Neill"}, 300},
		{"Restoring old furniture", 2, 2, "10:00", "workshop", "woodwork", []string{"Pieter de Graaf"}, 16},
		{"Solar kits for a whole street", 2, 2, "11:30", "main", "energy", []string{"Kwame Mensah", "Aisha Mwangi"}, 300},
		{"Kids take apart toys", 2, 2, "13:30", "studio", "craft", []string{"Ana Ruiz"}, 30},
		{"Open-source test gear", 2, 2, "15:00", "workshop", "electronics", []string{"Tomasz Wolski"}, 24},
		{"Bike repair basics", 3, 3, "10:00", "courtyard", "repair", []string{"Jonas Berg"}, 20},
		{"Golden repair for broken pottery", 3, 3, "11:00", "studio", "craft", []string{"Marta Costa"}, 16},
		{"Reading your energy bill", 3, 3, "13:30", "main", "energy", []string{"Kwame Mensah"}, 300},
		{"Microcontrollers in one evening", 3, 3, "18:30", "workshop", "electronics", []string{"Daan Visser", "Tomasz Wolski"}, 24},
		{"Clothing swap and mending bar", 4, 4, "10:00", "studio", "textiles", []string{"Elena Rossi", "Lena Vogt"}, 60},
		{"Spoon carving by the water", 4, 4, "11:00", "courtyard", "woodwork", []string{"Pieter de Graaf"}, 16},
		{"Empty shops into workshops", 4, 4, "13:30", "main", "community", []string{"Sofie Peeters"}, 300},
		{"Closing: what we fixed this week", 4, 4, "16:30", "main", "community", []string{"Femke Bakker", "Marieke Jansen"}, 300},
	}
	ev := 8 + ((1-(int(time.Now().Weekday())+8))%7+7)%7
	sessRows := make([]row, 0, len(sessions))
	for _, s := range sessions {
		reg := math.Round(s.cap * (0.55 + rnd.f()*0.5))
		if reg > s.cap {
			reg = s.cap
		}
		v := map[string]any{"when": xbSpan(ev+s.day, ev+s.until), "time": s.time, "room": s.room, "track": s.track, "speakers": s.speakers, "capacity": s.cap, "registered": reg}
		if s.day == s.until {
			v["when"] = xbDay(ev + s.day)
		}
		sessRows = append(sessRows, row{Title: s.title, Values: v})
	}
	sessRows[0].Markdown = "Welcome from the organisers, a short history of repair in Rotterdam, and what to expect this week.\n\n> [!NOTE]\n> Doors open at 08:45. Coffee is in the courtyard.\n"
	type city struct {
		name, country string
		lat, lng      float64
		n             int
	}
	cities := []city{
		{"Rotterdam", "nl", 51.9244, 4.4777, 7}, {"Amsterdam", "nl", 52.3676, 4.9041, 4}, {"Utrecht", "nl", 52.0907, 5.1214, 3},
		{"Antwerp", "be", 51.2194, 4.4025, 3}, {"Brussels", "be", 50.8503, 4.3517, 2}, {"Berlin", "de", 52.5200, 13.4050, 3}, {"Hamburg", "de", 53.5511, 9.9937, 2},
		{"Paris", "fr", 48.8566, 2.3522, 3}, {"Lyon", "fr", 45.7640, 4.8357, 1}, {"London", "uk", 51.5072, -0.1276, 3}, {"Manchester", "uk", 53.4808, -2.2426, 1},
		{"Madrid", "es", 40.4168, -3.7038, 1}, {"Barcelona", "es", 41.3874, 2.1686, 2}, {"Milan", "it", 45.4642, 9.1900, 2}, {"Warsaw", "pl", 52.2297, 21.0122, 2},
		{"Stockholm", "se", 59.3293, 18.0686, 1}, {"Lisbon", "pt", 38.7223, -9.1393, 1}, {"Copenhagen", "dk", 55.6761, 12.5683, 1}, {"Dublin", "ie", 53.3498, -6.2603, 1},
		{"New York", "us", 40.7128, -74.0060, 1}, {"Nairobi", "ke", -1.2921, 36.8219, 1}, {"Bengaluru", "in", 12.9716, 77.5946, 1}, {"Sao Paulo", "br", -23.5505, -46.6333, 1},
	}
	first := []string{"Noor", "Bas", "Lotte", "Sem", "Yara", "Thijs", "Mila", "Ruben", "Eva", "Jesse", "Fleur", "Milan", "Elif", "Hugo", "Sara", "Luca", "Ines", "Felix", "Hanna", "Mats", "Chloe", "Louis", "Amira", "Oliver", "Zoe", "Ben", "Lucia", "Pablo", "Giulia", "Marco", "Kasia", "Piotr", "Freja", "Joao", "Nina", "Aoife", "Maya", "Wanjiru", "Arjun", "Beatriz", "Tess", "Emre", "Lina", "Jan", "Iris", "Omar", "Rosa", "Kai"}
	letters := "BDKMVSTLGHRWPNJC"
	var att []row
	i := 0
	for _, c := range cities {
		for k := 0; k < c.n; k++ {
			name := fmt.Sprintf("%s %c.", first[i%len(first)], letters[(i*7)%len(letters)])
			ticket := "day"
			switch {
			case i%6 == 0:
				ticket = "week"
			case i%9 == 4:
				ticket = "volunteer"
			case i%4 == 1:
				ticket = "week"
			}
			picks := []string{}
			for len(picks) < 2+i%3 {
				t := sessions[rnd.n(0, len(sessions)-1)].title
				dup := false
				for _, p := range picks {
					dup = dup || p == t
				}
				if !dup {
					picks = append(picks, t)
				}
			}
			reg := -58 + int(math.Pow(rnd.f(), 0.6)*56)
			att = append(att, row{Title: name, Values: map[string]any{"country": c.country, "city": xbAt(c.lat+rnd.in(-0.03, 0.03), c.lng+rnd.in(-0.04, 0.04), c.name), "ticket": ticket, "registered": xbDay(reg), "sessions": picks, "email": fmt.Sprintf("%s.%c@example.com", strings.ToLower(first[i%len(first)]), letters[(i*7)%len(letters)]+32), "newsletter": i%3 != 2}})
			i++
		}
	}
	sponsors := []row{
		{Title: "Maasbrug Energy", Values: map[string]any{"tier": "gold", "amount": 12000, "status": "paid", "perks": []string{"logo", "stand", "talk", "newsletter"}, "contact": "partners@example.com"}},
		{Title: "Delta Circuits", Values: map[string]any{"tier": "gold", "amount": 10000, "status": "invoiced", "perks": []string{"logo", "stand", "talk"}, "contact": "events@example.com"}},
		{Title: "Stadslab Foundation", Values: map[string]any{"tier": "gold", "amount": 10000, "status": "paid", "perks": []string{"logo", "talk", "newsletter"}, "contact": "grants@example.com"}},
		{Title: "Polder Tools", Values: map[string]any{"tier": "silver", "amount": 5000, "status": "paid", "perks": []string{"logo", "stand"}, "contact": "marketing@example.com"}},
		{Title: "Harbourlight Printing", Values: map[string]any{"tier": "silver", "amount": 4000, "status": "agreed", "perks": []string{"logo", "newsletter"}, "contact": "hello@example.com"}},
		{Title: "Spoorwerk Bikes", Values: map[string]any{"tier": "silver", "amount": 4000, "status": "invoiced", "perks": []string{"logo", "stand"}, "contact": "team@example.com"}},
		{Title: "Nordzee Textiles", Values: map[string]any{"tier": "silver", "amount": 3500, "status": "agreed", "perks": []string{"logo", "stand"}, "contact": "studio@example.com"}},
		{Title: "Lumen Lighting Co", Values: map[string]any{"tier": "bronze", "amount": 1500, "status": "paid", "perks": []string{"logo"}, "contact": "info@example.com"}},
		{Title: "Grachten Woodworks", Values: map[string]any{"tier": "bronze", "amount": 1500, "status": "paid", "perks": []string{"logo"}, "contact": "workshop@example.com"}},
		{Title: "Kade Coffee Roasters", Values: map[string]any{"tier": "bronze", "amount": 1200, "status": "invoiced", "perks": []string{"logo", "stand"}, "contact": "coffee@example.com"}},
		{Title: "Brightwork Hardware", Values: map[string]any{"tier": "bronze", "amount": 1000, "status": "agreed", "perks": []string{"logo"}, "contact": "store@example.com"}},
		{Title: "Ferro Metals", Values: map[string]any{"tier": "bronze", "amount": 1000, "status": "talking", "perks": []string{"logo"}, "contact": "sales@example.com"}},
		{Title: "Windmolen Software", Values: map[string]any{"tier": "bronze", "amount": 1000, "status": "talking", "perks": []string{"logo", "newsletter"}, "contact": "community@example.com"}},
		{Title: "Open Bench Collective", Values: map[string]any{"tier": "community", "amount": 300, "status": "paid", "perks": []string{"logo", "stand"}, "contact": "bench@example.com"}},
		{Title: "Tulip Neighbourhood Radio", Values: map[string]any{"tier": "community", "amount": 0, "status": "agreed", "perks": []string{"newsletter"}, "contact": "radio@example.com"}},
	}
	type task struct {
		title, area, owner, status string
		due                        int
		prog                       float64
	}
	tasks := []task{
		{"Sign the venue contract", "venue", "femke", "done", -60, 100}, {"Floor plan for stands and workshops", "venue", "bram", "done", -25, 100}, {"Hire chairs and tables", "venue", "bram", "doing", 2, 60}, {"Wi-Fi and power for the workshop room", "venue", "lukas", "doing", 4, 40}, {"Signs and wayfinding", "venue", "ines", "todo", 6, 10}, {"First aid kit and trained first aider", "venue", "fatima", "doing", 5, 50},
		{"Confirm all speakers", "speakers", "femke", "done", -20, 100}, {"Book speaker travel", "speakers", "ines", "doing", 1, 75}, {"Collect slides by the Friday before", "speakers", "ines", "todo", 5, 20}, {"Speaker dinner booking", "speakers", "femke", "done", -5, 100}, {"Hotel block for speakers", "speakers", "ines", "done", -14, 100},
		{"Sponsor packages and prices", "sponsors", "sofie", "done", -45, 100}, {"Chase unpaid invoices", "sponsors", "sofie", "doing", 3, 50}, {"Sponsor logos on the website", "sponsors", "lukas", "doing", 0, 70}, {"Stand allocation for sponsors", "sponsors", "bram", "todo", 4, 0},
		{"Launch the website", "comms", "lukas", "done", -50, 100}, {"Registration form live", "comms", "lukas", "done", -58, 100}, {"Newsletter: programme announced", "comms", "sofie", "done", -18, 100}, {"Social posts for each speaker", "comms", "sofie", "doing", 6, 55}, {"Press release to local media", "comms", "femke", "todo", 3, 15}, {"Photo and video volunteers", "comms", "fatima", "doing", 4, 50},
		{"Recruit 20 volunteers", "volunteers", "fatima", "doing", 1, 80}, {"Volunteer briefing session", "volunteers", "fatima", "todo", 7, 0}, {"Rota for the registration desk", "volunteers", "fatima", "todo", 5, 25}, {"Volunteer T-shirts", "volunteers", "bram", "doing", 4, 60},
		{"Workshop materials order", "programme", "bram", "doing", 2, 65}, {"Print the programme booklet", "programme", "lukas", "todo", 5, 30}, {"Accessibility check of every room", "programme", "ines", "todo", 6, 0}, {"Lunch caterer and dietary list", "programme", "femke", "doing", 3, 70}, {"Feedback survey for after the event", "programme", "sofie", "todo", 13, 0},
	}
	trows := make([]row, 0, len(tasks))
	for _, t := range tasks {
		trows = append(trows, row{Title: t.title, Values: map[string]any{"area": t.area, "owner": t.owner, "status": t.status, "due": xbDay(t.due), "progress": t.prog}})
	}
	countryOpts := []opt{o("nl", "Netherlands", "orange"), o("be", "Belgium", "yellow"), o("de", "Germany", "gray"), o("fr", "France", "blue"), o("uk", "United Kingdom", "red"), o("es", "Spain", "red"), o("it", "Italy", "green"), o("pl", "Poland", "pink"), o("se", "Sweden", "blue"), o("pt", "Portugal", "green"), o("dk", "Denmark", "red"), o("ie", "Ireland", "green"), o("us", "United States", "blue"), o("ke", "Kenya", "brown"), o("in", "India", "orange"), o("br", "Brazil", "yellow")}
	trackOpts := []opt{o("repair", "Repair", "orange"), o("electronics", "Electronics", "blue"), o("textiles", "Textiles", "pink"), o("woodwork", "Woodwork", "brown"), o("energy", "Energy", "yellow"), o("craft", "Craft", "purple"), o("community", "Community", "green")}
	return Starter{
		ID: "ex-community-event", Name: "Harbour Makers Week", Description: "A five-day community festival: sessions, speakers, attendees on a map, sponsors, tasks and a run-of-show, all connected.", Icon: "i:megaphone:violet", Kind: "doc", Category: catExamples, Tags: []string{"event", "conference", "community", "volunteers"},
		markdown: `Harbour Makers Week is a free festival of repair, making and sharing in an old warehouse on the Rotterdam waterfront. Five days, twenty-two sessions, sixteen speakers from nine countries and a team of five volunteers who organise it in their evenings.

> [!IMPORTANT]
> The festival opens on Monday week. The task board below shows what is still open.

## The programme

Every session, with its speakers, room and how full it is.

{{db:ev-sessions}}

## Who is coming

Attendees pinned where they travel from. Switch to the dashboard tab for countries and registrations over time, or to the form to register someone at the door.

{{db:ev-attendees}}

## Speakers

{{db:ev-speakers}}

## Sponsors

{{db:ev-sponsors}}

## Getting it done

{{db:ev-tasks}}

## Organiser pages

- {{page:ev-runofshow}}: minute by minute for opening day
- {{page:ev-volunteers}}: what every volunteer needs to know
- {{page:ev-venue}}: rooms, access, power and the code of conduct
`,
		children: []Starter{
			{
				key: "ev-speakers", Name: "Speakers", Icon: "i:users:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Name", Type: "title"},
					{ID: "org", Name: "Organisation", Type: "text"},
					{ID: "country", Name: "Country", Type: "select", Options: countryOpts},
					{ID: "topic", Name: "Topic", Type: "select", Options: append([]opt{}, trackOpts...)},
					{ID: "travel", Name: "Needs travel", Type: "checkbox"},
					{ID: "email", Name: "Email", Type: "email"},
				},
				views: []view{
					{Name: "Gallery", Type: "gallery", Config: `{"layout":{"cover":"content","size":"medium"},` + xbShow("org", "topic", "country") + `}`},
					{Name: "All speakers", Type: "table", Config: `{"sorts":[{"prop":"title","dir":"asc"}]}`},
					{Name: "By topic", Type: "board", Config: `{"group":{"prop":"topic"},` + xbShow("org", "country") + `}`},
				},
				rows: srows,
			},
			{
				key: "ev-sessions", Name: "Sessions", Icon: "i:calendar:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Session", Type: "title"},
					{ID: "when", Name: "Day", Type: "date"},
					{ID: "time", Name: "Starts", Type: "text"},
					{ID: "room", Name: "Room", Type: "select", Options: []opt{o("main", "Main hall", "blue"), o("workshop", "Workshop room", "orange"), o("studio", "Craft studio", "pink"), o("courtyard", "Courtyard", "green"), o("gallery", "Gallery", "purple")}},
					{ID: "track", Name: "Track", Type: "select", Options: trackOpts},
					{ID: "speakers", Name: "Speakers", Type: "relation", Collection: "@ev-speakers"},
					{ID: "capacity", Name: "Places", Type: "number"},
					{ID: "registered", Name: "Signed up", Type: "number"},
					{ID: "full", Name: "Full", Type: "formula", Expr: `round(prop("Signed up") / prop("Places"), 2)`, Format: "percent", Display: "bar", Max: 1},
				},
				views: []view{
					{Name: "Schedule", Type: "timeline", Config: `{"layout":{"date":"when","zoom":"week","table":true},"sorts":[{"prop":"when","dir":"asc"},{"prop":"time","dir":"asc"}],` + xbShow("time", "room") + `}`},
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"when"},` + xbShow("time", "room") + `}`},
					{Name: "Programme", Type: "table", Config: `{"group":{"prop":"when"},"sorts":[{"prop":"time","dir":"asc"}],` + xbCols("title:300", "time:80", "room:140", "track:120", "speakers:220", "full:150", "capacity:90", "registered:100") + `}`},
					{Name: "Sign-ups by track", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"track"},"fn":"sum","prop":"registered","labels":true,"legend":true}}}`},
					{Name: "Rooms by day", Type: "chart", Config: `{"layout":{"chart":{"kind":"heatmap","x":{"prop":"when"},"series":{"prop":"room"},"fn":"sum","prop":"registered","labels":true}}}`},
					{Name: "Speakers network", Type: "graph", Config: `{"layout":{"graph":{"color":"track","links":["speakers"],"external":true,"labels":"auto"}}}`},
				},
				rows: sessRows,
			},
			{
				key: "ev-attendees", Name: "Attendees", Icon: "i:globe:teal", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Name", Type: "title"},
					{ID: "email", Name: "Email", Type: "email"},
					{ID: "country", Name: "Country", Type: "select", Options: countryOpts},
					{ID: "city", Name: "Travelling from", Type: "place"},
					{ID: "ticket", Name: "Ticket", Type: "select", Options: []opt{o("day", "Day pass", "blue"), o("week", "Full week", "purple"), o("volunteer", "Volunteer", "green")}},
					{ID: "sessions", Name: "Sessions", Type: "relation", Collection: "@ev-sessions"},
					{ID: "registered", Name: "Registered", Type: "date"},
					{ID: "newsletter", Name: "Newsletter", Type: "checkbox"},
				},
				views: []view{
					{Name: "Map", Type: "map", Config: `{"layout":{"place":"city"},` + xbShow("country", "ticket") + `}`},
					{Name: "Dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"e1","size":"s","title":"Registered","chart":{"kind":"number","fn":"count"}},` +
						`{"id":"e2","size":"s","title":"Countries","chart":{"kind":"number","fn":"count_unique","prop":"country"}},` +
						`{"id":"e3","size":"s","title":"Volunteers","chart":{"kind":"number","fn":"count","filter":{"prop":"ticket","cond":"is","value":"volunteer"}}},` +
						`{"id":"e4","size":"s","title":"Want the newsletter","chart":{"kind":"number","fn":"percent_checked","prop":"newsletter"}},` +
						`{"id":"e5","title":"Where people come from","chart":{"kind":"donut","x":{"prop":"country"},"fn":"count","sort":"value_desc","hide_empty":true}},` +
						`{"id":"e6","title":"Tickets","chart":{"kind":"bar","x":{"prop":"ticket"},"fn":"count","labels":true}},` +
						`{"id":"e7","wide":true,"title":"Registrations over time","chart":{"kind":"area","x":{"prop":"registered","by":"week"},"fn":"count","cumulative":true,"smooth":true,"labels":true}}` +
						`]}}`},
					{Name: "Register at the door", Type: "form", Config: `{"form":{"title":"Register for Harbour Makers Week","description":"Free entry. Tell us who you are and which sessions you want, and we will save you a seat.","questions":[{"prop":"title","label":"Your name","required":true},{"prop":"email","label":"Email","required":true},{"prop":"country","label":"Where do you live?"},{"prop":"ticket","label":"Which ticket?","required":true},{"prop":"sessions","label":"Sessions you want to join","help":"Workshops have limited places."},{"prop":"newsletter","label":"Send me the newsletter"}],"submit":"Register","done":"Thank you! See you at the warehouse."}}`},
					{Name: "Who is in which session", Type: "graph", Config: `{"layout":{"graph":{"color":"ticket","links":["sessions"],"external":true,"labels":"auto"}}}`},
					{Name: "All attendees", Type: "table", Config: `{"sorts":[{"prop":"registered","dir":"desc"}],` + xbHide("city") + `}`},
				},
				rows: att,
			},
			{
				key: "ev-sponsors", Name: "Sponsors", Icon: "i:star:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Sponsor", Type: "title"},
					{ID: "tier", Name: "Tier", Type: "select", Options: []opt{o("gold", "Gold", "yellow"), o("silver", "Silver", "gray"), o("bronze", "Bronze", "brown"), o("community", "Community partner", "green")}},
					{ID: "amount", Name: "Amount", Type: "number", Format: "euro"},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("talking", "Talking", "gray", "todo"), so("agreed", "Agreed", "yellow", "doing"), so("invoiced", "Invoiced", "blue", "doing"), so("paid", "Paid", "green", "done")}},
					{ID: "perks", Name: "Includes", Type: "multi_select", Options: []opt{o("logo", "Logo on site", "blue"), o("stand", "Stand", "orange"), o("talk", "Sponsor talk", "purple"), o("newsletter", "Newsletter mention", "green")}},
					{ID: "contact", Name: "Contact", Type: "email"},
				},
				views: []view{
					{Name: "Pipeline", Type: "board", Config: `{"group":{"prop":"status"},"layout":{"total":{"fn":"sum","prop":"amount"}},` + xbShow("tier", "amount") + `}`},
					{Name: "Money by tier", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"tier"},"series":{"prop":"status"},"fn":"sum","prop":"amount","stacked":true,"labels":true,"legend":true}}}`},
					{Name: "Share", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"tier"},"fn":"sum","prop":"amount","labels":true,"legend":true}}}`},
					{Name: "All sponsors", Type: "table", Config: `{"sorts":[{"prop":"amount","dir":"desc"}],"calcs":{"amount":"sum"}}`},
				},
				rows: sponsors,
			},
			{
				key: "ev-tasks", Name: "Organiser tasks", Icon: "i:kanban:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Task", Type: "title"},
					{ID: "status", Name: "Status", Type: "status", Options: statusOpts},
					{ID: "area", Name: "Area", Type: "select", Options: []opt{o("venue", "Venue", "blue"), o("speakers", "Speakers", "purple"), o("sponsors", "Sponsors", "yellow"), o("comms", "Comms", "pink"), o("volunteers", "Volunteers", "green"), o("programme", "Programme", "orange")}},
					{ID: "owner", Name: "Owner", Type: "select", Options: []opt{o("femke", "Femke", "purple"), o("bram", "Bram", "blue"), o("ines", "Ines", "pink"), o("lukas", "Lukas", "orange"), o("sofie", "Sofie", "green"), o("fatima", "Fatima", "yellow")}},
					{ID: "due", Name: "Due", Type: "date"},
					{ID: "progress", Name: "Progress", Type: "number", Display: "bar", Max: 100},
				},
				views: []view{
					{Name: "Board", Type: "board", Config: `{"group":{"prop":"area"},"layout":{"progress":"progress","size":"small"},"sorts":[{"prop":"due","dir":"asc"}],` + xbShow("owner", "due", "status") + `}`},
					{Name: "By status", Type: "board", Config: `{"group":{"prop":"status"},` + xbShow("area", "owner", "progress") + `}`},
					{Name: "Due dates", Type: "timeline", Config: `{"layout":{"date":"due","zoom":"week"},` + xbShow("owner") + `}`},
					{Name: "Who is doing what", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"owner"},"series":{"prop":"status"},"fn":"count","stacked":true,"legend":true}}}`},
					{Name: "All tasks", Type: "table", Config: `{"sorts":[{"prop":"due","dir":"asc"}],"calcs":{"progress":"average"}}`},
				},
				rows: trows,
			},
			{
				key: "ev-runofshow", Name: "Run of show, opening day", Icon: "i:clock:blue", Kind: "doc",
				markdown: `**Day:** opening day · **Doors:** 08:45 · **Stage manager:** Femke · **Desk lead:** Fatima

> [!IMPORTANT]
> One radio channel for everyone. Say your name, then the room, then the message.

## Timeline

| Time | What | Where | Who |
| --- | --- | --- | --- |
| 07:30 | Crew arrives, coffee on | Courtyard | Bram, volunteers |
| 08:00 | Sound check with the opening speakers | Main hall | Lukas |
| 08:45 | Doors open, registration desk live | Entrance | Fatima and four volunteers |
| 09:30 | Opening: why we fix things | Main hall | Femke, Marieke |
| 10:00 | Repair cafe opens for the week | Courtyard | Marieke, Jonas |
| 11:00 | Soldering for beginners | Workshop room | Daan |
| 12:30 | Lunch | Courtyard | Caterer |
| 14:00 | Panel: makerspaces that last | Main hall | Sofie, Grace, Aisha |
| 15:30 | Visible mending | Craft studio | Elena |
| 17:30 | Doors close, quick debrief | Courtyard | Everyone |

## How a speaker change-over works

` + fence + `mermaid
sequenceDiagram
    participant SM as Stage manager
    participant AV as Sound desk
    participant S1 as Speaker on stage
    participant S2 as Next speaker
    participant V as Room volunteer
    SM->>S2: 10 minute call, meet at the side door
    V->>S2: Walk to the stage side, mic check
    S1->>SM: Last slide
    SM->>AV: Fade music in
    AV->>S2: Clicker and mic live
    SM->>S2: Go
    V->>SM: Room is full, close the doors
` + fence + `

## If something goes wrong

<details>
<summary>A speaker is late</summary>

Swap with the next slot in the same room and tell the desk so the screens update. If nobody can swap, run the repair cafe live stream in the main hall.

</details>

<details>
<summary>The projector fails</summary>

Spare projector and cables are in the black crate behind the stage. Lukas has the key.

</details>

<details>
<summary>Someone is unwell</summary>

Call Fatima on the radio, she is the first aider. The defibrillator is next to the entrance. For anything serious call 112 first.

</details>
`,
				children: []Starter{
					{
						key: "ev-volunteers", Name: "Volunteer guide", Icon: "i:heart:green", Kind: "doc",
						markdown: `Thank you for helping. This page has everything you need for your shift.

## Before your shift

- [ ] Read this page and the {{page:ev-runofshow}}
- [ ] Bring a refillable bottle, we have water and coffee
- [ ] Wear comfortable shoes, the warehouse floor is concrete

## At the registration desk

1. Greet people and ask for their name.
2. Find them in the attendee list, or register them on the spot with the form.
3. Hand over a badge and the programme booklet.

> [!TIP]
> If the queue is longer than ten people, open the second laptop and call another volunteer on the radio.

## Shifts

| Shift | Time | Volunteers needed |
| --- | --- | --- |
| Morning | 08:30 to 13:00 | 8 |
| Afternoon | 12:30 to 17:30 | 8 |
| Evening workshops | 18:00 to 21:00 | 4 |
`,
					},
				},
			},
			{
				key: "ev-venue", Name: "Venue and code of conduct", Icon: "i:building:amber", Kind: "doc",
				markdown: `**Venue:** the Old Harbour Warehouse, on the waterfront in Rotterdam · **Capacity:** 400 people

## Rooms

| Room | Seats | Good for | Power |
| --- | --- | --- | --- |
| Main hall | 300 | Talks and panels | Stage and 6 floor sockets |
| Workshop room | 24 | Soldering, electronics | Every bench, extraction fan |
| Craft studio | 30 | Textiles, ceramics | Sewing machines on the left wall |
| Courtyard | 80 | Repair cafe, lunch | Outdoor sockets, covered area |
| Gallery | 120 standing | Student showcase | Lighting track only |

## Getting here

- 10 minutes walk from the central station, signposted from the harbour bridge
- Bike parking for 150 bikes at the side entrance
- Step-free access through the courtyard gate; lift to the gallery

## Code of conduct

> [!CAUTION]
> Harbour Makers Week is for everyone. Harassment of any kind is not tolerated, and anyone who breaks this code may be asked to leave without a refund.

- Be kind and patient, especially with beginners.
- Ask before you touch someone else's project.
- Safety glasses on in the workshop room, always.
- To report a problem, talk to anyone in a green T-shirt or email safety@example.com.
`,
			},
		},
	}
}

func xbTrip() Starter {
	type place struct {
		name, city, kind string
		lat, lng         float64
		fee              float64
		hours            float64
		must             bool
	}
	places := []place{
		{"Senso-ji Temple", "tokyo", "temple", 35.7148, 139.7967, 0, 2, true},
		{"Tokyo Skytree", "tokyo", "view", 35.7101, 139.8107, 3100, 2, false},
		{"Ueno Park", "tokyo", "park", 35.7156, 139.7745, 0, 2, false},
		{"Shibuya Crossing", "tokyo", "city", 35.6595, 139.7005, 0, 1, true},
		{"Meiji Jingu", "tokyo", "shrine", 35.6764, 139.6993, 0, 1.5, true},
		{"Tsukiji Outer Market", "tokyo", "food", 35.6655, 139.7707, 0, 2, true},
		{"Imperial Palace East Gardens", "tokyo", "park", 35.6852, 139.7528, 0, 1.5, false},
		{"Hakone Open-Air Museum", "hakone", "museum", 35.2446, 139.0517, 2000, 2.5, true},
		{"Lake Ashi", "hakone", "nature", 35.2056, 139.0270, 1200, 2, true},
		{"Owakudani", "hakone", "nature", 35.2440, 139.0196, 0, 1, false},
		{"Fushimi Inari Taisha", "kyoto", "shrine", 34.9671, 135.7727, 0, 3, true},
		{"Kiyomizu-dera", "kyoto", "temple", 34.9949, 135.7850, 500, 1.5, true},
		{"Gion", "kyoto", "city", 35.0037, 135.7788, 0, 2, true},
		{"Arashiyama Bamboo Grove", "kyoto", "nature", 35.0170, 135.6713, 0, 1.5, true},
		{"Kinkaku-ji", "kyoto", "temple", 35.0394, 135.7292, 500, 1, true},
		{"Nishiki Market", "kyoto", "food", 35.0050, 135.7649, 0, 1.5, false},
		{"Todai-ji", "nara", "temple", 34.6890, 135.8398, 800, 1.5, true},
		{"Nara Park", "nara", "park", 34.6851, 135.8430, 0, 2, true},
		{"Osaka Castle", "osaka", "castle", 34.6873, 135.5262, 600, 2, true},
		{"Dotonbori", "osaka", "food", 34.6687, 135.5013, 0, 2.5, true},
		{"Kuromon Market", "osaka", "food", 34.6654, 135.5066, 0, 1.5, false},
		{"Hiroshima Peace Memorial Park", "hiroshima", "memorial", 34.3955, 132.4536, 200, 3, true},
		{"Itsukushima Shrine", "hiroshima", "shrine", 34.2959, 132.3199, 300, 2.5, true},
	}
	prows := make([]row, 0, len(places))
	for _, p := range places {
		prows = append(prows, row{Title: p.name, Values: map[string]any{"where": xbAt(p.lat, p.lng, p.name), "city": p.city, "kind": p.kind, "fee": p.fee, "hours": p.hours, "must": p.must}})
	}
	prows[10].Markdown = "Go at 7am, before the tour groups. The full loop to the summit takes about 2.5 hours; most people turn back at the Yotsutsuji crossing, which has the best view over Kyoto.\n"
	prows[21].Markdown = "Allow at least two hours for the museum. It is moving and heavy, so plan a quiet evening afterwards.\n"
	type day struct {
		title, city string
		places      []string
		stay        string
	}
	days := []day{
		{"Arrive in Tokyo, settle into Shinjuku", "tokyo", nil, "Hotel in Shinjuku"},
		{"Old Tokyo: Asakusa and Ueno", "tokyo", []string{"Senso-ji Temple", "Tokyo Skytree", "Ueno Park"}, "Hotel in Shinjuku"},
		{"Shibuya, Harajuku and Meiji Jingu", "tokyo", []string{"Shibuya Crossing", "Meiji Jingu"}, "Hotel in Shinjuku"},
		{"Market breakfast and the palace gardens", "tokyo", []string{"Tsukiji Outer Market", "Imperial Palace East Gardens"}, "Hotel in Shinjuku"},
		{"Into the mountains: Hakone", "hakone", []string{"Hakone Open-Air Museum", "Owakudani"}, "Ryokan in Hakone"},
		{"Lake Ashi, then the bullet train to Kyoto", "kyoto", []string{"Lake Ashi"}, "Guesthouse in Kyoto"},
		{"Fushimi Inari at dawn and an evening in Gion", "kyoto", []string{"Fushimi Inari Taisha", "Kiyomizu-dera", "Gion"}, "Guesthouse in Kyoto"},
		{"Bamboo and the Golden Pavilion", "kyoto", []string{"Arashiyama Bamboo Grove", "Kinkaku-ji"}, "Guesthouse in Kyoto"},
		{"Day trip to Nara", "nara", []string{"Todai-ji", "Nara Park"}, "Guesthouse in Kyoto"},
		{"Nishiki Market, tea ceremony, on to Osaka", "osaka", []string{"Nishiki Market"}, "Hotel in Namba, Osaka"},
		{"Osaka Castle and Dotonbori by night", "osaka", []string{"Osaka Castle", "Dotonbori"}, "Hotel in Namba, Osaka"},
		{"Hiroshima and Miyajima", "hiroshima", []string{"Hiroshima Peace Memorial Park", "Itsukushima Shrine"}, "Hotel near Hiroshima station"},
		{"Back to Osaka for a last food crawl", "osaka", []string{"Kuromon Market", "Dotonbori"}, "Airport hotel, Kansai"},
		{"Fly home", "osaka", nil, ""},
	}
	drows := make([]row, 0, len(days))
	for i, d := range days {
		v := map[string]any{"date": xbDay(3 + i), "city": d.city, "stay": d.stay}
		if d.places != nil {
			v["places"] = d.places
		}
		drows = append(drows, row{Title: fmt.Sprintf("Day %d · %s", i+1, d.title), Values: v})
	}
	drows[0].Markdown = "Land at Haneda at 16:40. Pick up the Wi-Fi router at arrivals, top up the IC cards and take the train to Shinjuku. Early dinner, early night.\n"
	drows[6].Markdown = "Alarm at 6:00. Fushimi Inari before breakfast, Kiyomizu-dera late morning, nap, then Gion at dusk.\n\n- [ ] Book dinner in Pontocho\n- [ ] Cash for the temple stalls\n"
	bookings := []row{
		{Title: "Flight out: Heathrow to Haneda", Values: map[string]any{"type": "flight", "dates": xbSpan(2, 3), "status": "paid", "cost": 896, "provider": "Skyline Air", "ref": "SKL7Q2", "city": "tokyo"}},
		{Title: "Hotel in Shinjuku, 4 nights", Values: map[string]any{"type": "stay", "dates": xbSpan(3, 7), "status": "paid", "cost": 860, "provider": "Shinjuku Garden Hotel", "ref": "SGH-48213", "city": "tokyo"}},
		{Title: "Pocket Wi-Fi rental", Values: map[string]any{"type": "other", "dates": xbSpan(3, 16), "status": "paid", "cost": 64, "provider": "GoConnect Rentals", "ref": "GC-99172", "city": "tokyo"}},
		{Title: "Travel insurance", Values: map[string]any{"type": "other", "dates": xbSpan(2, 16), "status": "paid", "cost": 142, "provider": "Safe Travels Mutual", "ref": "POL-3381-22"}},
		{Title: "Japan Rail Passes for two, 7 days", Values: map[string]any{"type": "train", "dates": xbSpan(7, 13), "status": "paid", "cost": 660, "provider": "Rail pass agent", "ref": "JRP-20441", "city": "hakone"}},
		{Title: "Ryokan in Hakone with dinner", Values: map[string]any{"type": "stay", "dates": xbSpan(7, 8), "status": "paid", "cost": 520, "provider": "Yumoto Hot Spring Inn", "ref": "YH-0712", "city": "hakone"}},
		{Title: "Luggage forwarding to Kyoto", Values: map[string]any{"type": "other", "dates": xbSpan(7, 8), "status": "todo", "cost": 40, "provider": "Hotel front desk", "city": "kyoto"}},
		{Title: "Guesthouse in Kyoto, 4 nights", Values: map[string]any{"type": "stay", "dates": xbSpan(8, 12), "status": "booked", "cost": 780, "provider": "Machiya Stay Kyoto", "ref": "MSK-5521", "city": "kyoto"}},
		{Title: "Tea ceremony in a Kyoto townhouse", Values: map[string]any{"type": "tour", "dates": xbSpan(12, 12), "status": "paid", "cost": 96, "provider": "Uji Tea House", "ref": "UTH-317", "city": "kyoto"}},
		{Title: "Hotel in Namba, Osaka, 2 nights", Values: map[string]any{"type": "stay", "dates": xbSpan(12, 14), "status": "booked", "cost": 410, "provider": "Namba River Hotel", "ref": "NRH-77004", "city": "osaka"}},
		{Title: "Street food cooking class", Values: map[string]any{"type": "tour", "dates": xbSpan(13, 13), "status": "booked", "cost": 140, "provider": "Osaka Kitchen Studio", "ref": "OKS-2208", "city": "osaka"}},
		{Title: "Bullet train to Hiroshima and back", Values: map[string]any{"type": "train", "dates": xbSpan(14, 15), "status": "todo", "cost": 230, "provider": "Station ticket office", "city": "hiroshima"}},
		{Title: "Hotel near Hiroshima station", Values: map[string]any{"type": "stay", "dates": xbSpan(14, 15), "status": "booked", "cost": 150, "provider": "Peace Boulevard Hotel", "ref": "PBH-1190", "city": "hiroshima"}},
		{Title: "Ferry to Miyajima", Values: map[string]any{"type": "train", "dates": xbSpan(14, 14), "status": "todo", "cost": 12, "provider": "Pay at the pier", "city": "hiroshima"}},
		{Title: "Airport hotel, Kansai", Values: map[string]any{"type": "stay", "dates": xbSpan(15, 16), "status": "todo", "cost": 140, "provider": "Kansai Gateway Hotel", "city": "osaka"}},
		{Title: "Flight home: Kansai to Heathrow", Values: map[string]any{"type": "flight", "dates": xbSpan(16, 16), "status": "paid", "cost": 896, "provider": "Skyline Air", "ref": "SKL7Q2", "city": "osaka"}},
	}
	budget := []row{
		{Title: "Flights", Values: map[string]any{"kind": "plan", "category": "flights", "amount": 1800}},
		{Title: "Places to stay", Values: map[string]any{"kind": "plan", "category": "stay", "amount": 2900}},
		{Title: "Trains and local transport", Values: map[string]any{"kind": "plan", "category": "transport", "amount": 1050}},
		{Title: "Food and drink", Values: map[string]any{"kind": "plan", "category": "food", "amount": 1300}},
		{Title: "Sights and activities", Values: map[string]any{"kind": "plan", "category": "sights", "amount": 450}},
		{Title: "Shopping and gifts", Values: map[string]any{"kind": "plan", "category": "shopping", "amount": 400}},
		{Title: "Insurance and phone", Values: map[string]any{"kind": "plan", "category": "admin", "amount": 210}},
		{Title: "Spare money", Values: map[string]any{"kind": "plan", "category": "spare", "amount": 300}},
		{Title: "Paid: outbound flight", Values: map[string]any{"kind": "spent", "category": "flights", "amount": 896, "date": "-74d", "booking": "Flight out: Heathrow to Haneda"}},
		{Title: "Paid: flight home", Values: map[string]any{"kind": "spent", "category": "flights", "amount": 896, "date": "-74d", "booking": "Flight home: Kansai to Heathrow"}},
		{Title: "Paid: Shinjuku hotel", Values: map[string]any{"kind": "spent", "category": "stay", "amount": 860, "date": "-60d", "booking": "Hotel in Shinjuku, 4 nights"}},
		{Title: "Paid: Hakone ryokan", Values: map[string]any{"kind": "spent", "category": "stay", "amount": 520, "date": "-55d", "booking": "Ryokan in Hakone with dinner"}},
		{Title: "Deposit: Kyoto guesthouse", Values: map[string]any{"kind": "spent", "category": "stay", "amount": 150, "date": "-52d", "booking": "Guesthouse in Kyoto, 4 nights"}},
		{Title: "Paid: rail passes", Values: map[string]any{"kind": "spent", "category": "transport", "amount": 660, "date": "-30d", "booking": "Japan Rail Passes for two, 7 days"}},
		{Title: "Paid: tea ceremony", Values: map[string]any{"kind": "spent", "category": "sights", "amount": 96, "date": "-21d", "booking": "Tea ceremony in a Kyoto townhouse"}},
		{Title: "Deposit: cooking class", Values: map[string]any{"kind": "spent", "category": "sights", "amount": 70, "date": "-18d", "booking": "Street food cooking class"}},
		{Title: "Paid: travel insurance", Values: map[string]any{"kind": "spent", "category": "admin", "amount": 142, "date": "-40d", "booking": "Travel insurance"}},
		{Title: "Paid: Wi-Fi router", Values: map[string]any{"kind": "spent", "category": "admin", "amount": 64, "date": "-12d", "booking": "Pocket Wi-Fi rental"}},
		{Title: "Guidebook and phrasebook", Values: map[string]any{"kind": "spent", "category": "shopping", "amount": 32, "date": "-25d"}},
	}
	type item struct {
		name, cat, who string
		qty            float64
		packed         bool
	}
	items := []item{
		{"Passports", "docs", "shared", 2, true}, {"Printed booking confirmations", "docs", "shared", 1, true}, {"Rail pass vouchers", "docs", "shared", 2, true}, {"Credit cards and some yen", "docs", "shared", 1, false}, {"Driving licences", "docs", "shared", 2, true},
		{"T-shirts for Alex", "clothes", "alex", 6, true}, {"T-shirts for Jordan", "clothes", "jordan", 6, false}, {"Light jumper", "clothes", "alex", 1, true}, {"Rain jacket", "clothes", "jordan", 1, true}, {"Comfortable walking shoes", "clothes", "shared", 2, true}, {"Socks without holes (for temples)", "clothes", "shared", 10, false}, {"Smart outfit for the ryokan dinner", "clothes", "jordan", 1, false},
		{"Toothbrushes and toothpaste", "toiletries", "shared", 2, false}, {"Sun cream", "toiletries", "shared", 1, true}, {"Travel towel", "toiletries", "alex", 1, true}, {"Small hand towel (Japan has few dryers)", "toiletries", "shared", 2, false},
		{"Phone chargers", "tech", "shared", 2, false}, {"Plug adapters, type A", "tech", "shared", 2, true}, {"Power bank", "tech", "alex", 1, true}, {"Camera and spare battery", "tech", "jordan", 1, false}, {"Headphones", "tech", "shared", 2, true},
		{"Plasters and blister pads", "health", "shared", 1, true}, {"Painkillers", "health", "shared", 1, true}, {"Prescription medicine in original box", "health", "jordan", 1, false}, {"Motion sickness tablets for the ferry", "health", "alex", 1, false},
		{"Day backpack", "misc", "alex", 1, true}, {"Coin purse", "misc", "jordan", 1, true}, {"Reusable water bottles", "misc", "shared", 2, true}, {"Folding tote bag for shopping", "misc", "shared", 1, false}, {"Small gifts for the ryokan hosts", "misc", "jordan", 2, false}, {"Snacks for the flight", "misc", "shared", 1, false},
	}
	irows := make([]row, 0, len(items))
	seen := map[string]bool{}
	for _, it := range items {
		t := it.name
		if seen[t] {
			t += " (Jordan)"
		}
		seen[t] = true
		irows = append(irows, row{Title: t, Values: map[string]any{"cat": it.cat, "who": it.who, "qty": it.qty, "packed": it.packed}})
	}
	cityOpts := []opt{o("tokyo", "Tokyo", "red"), o("hakone", "Hakone", "green"), o("kyoto", "Kyoto", "purple"), o("nara", "Nara", "brown"), o("osaka", "Osaka", "orange"), o("hiroshima", "Hiroshima", "blue")}
	catOpts := []opt{o("flights", "Flights", "blue"), o("stay", "Places to stay", "purple"), o("transport", "Trains and transport", "green"), o("food", "Food and drink", "orange"), o("sights", "Sights and activities", "pink"), o("shopping", "Shopping", "yellow"), o("admin", "Insurance and phone", "gray"), o("spare", "Spare money", "brown")}
	return Starter{
		ID: "ex-trip-planner", Name: "Two weeks in Japan", Description: "A complete trip plan: sights on a map, a day-by-day itinerary, bookings on a timeline, a budget and a packing board.", Icon: "i:plane:cyan", Kind: "doc", Category: catExamples, Tags: []string{"travel", "itinerary", "map", "budget"},
		markdown: `Alex and Jordan leave for Japan in three days. Fourteen days, six places, one rail pass and a list of temples longer than the time they have. Everything for the trip lives here, so either of them can answer "where are we sleeping on Thursday?" from a phone.

## The route

Every sight they want to see, pinned on the map. Click a pin to open its notes, or switch to the gallery.

{{db:trip-places}}

> [!NOTE]
> The route runs west: Tokyo, Hakone, Kyoto with a day in Nara, Osaka, a night in Hiroshima, then home from Kansai airport.

## Day by day

{{db:trip-days}}

## Bookings

Hotels show as bars across the trip, so a missing night stands out straight away.

{{db:trip-bookings}}

## Money

What they planned to spend, next to what is already paid.

{{db:trip-budget}}

## Packing

The bar on each column shows how much of it is already in the case.

{{db:trip-packing}}

## Notes

- {{page:trip-notes}}: phrases, manners, money and transport tips
- {{page:trip-food}}: the eat-it-before-we-leave list
- {{page:trip-before}}: jobs to finish before the taxi comes
`,
		children: []Starter{
			{
				key: "trip-places", Name: "Places to see", Icon: "i:pin:rose", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Place", Type: "title"},
					{ID: "where", Name: "Location", Type: "place"},
					{ID: "city", Name: "City", Type: "select", Options: cityOpts},
					{ID: "kind", Name: "Kind", Type: "select", Options: []opt{o("temple", "Temple", "orange"), o("shrine", "Shrine", "red"), o("park", "Park", "green"), o("nature", "Nature", "green"), o("museum", "Museum", "purple"), o("food", "Food", "yellow"), o("city", "City life", "blue"), o("view", "Viewpoint", "blue"), o("castle", "Castle", "brown"), o("memorial", "Memorial", "gray")}},
					{ID: "fee", Name: "Entry fee", Type: "number", Format: "yen"},
					{ID: "hours", Name: "Hours needed", Type: "number"},
					{ID: "must", Name: "Must see", Type: "checkbox"},
				},
				views: []view{
					{Name: "Map", Type: "map", Config: `{"layout":{"place":"where"},` + xbShow("city", "kind") + `}`},
					{Name: "Gallery", Type: "gallery", Config: `{"layout":{"cover":"none","size":"small"},` + xbShow("city", "kind", "fee", "must") + `}`},
					{Name: "By city", Type: "table", Config: `{"group":{"prop":"city"},"calcs":{"hours":"sum","fee":"sum"}}`},
					{Name: "Hours per city", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"city"},"fn":"sum","prop":"hours","labels":true}}}`},
				},
				rows: prows,
			},
			{
				key: "trip-days", Name: "Itinerary", Icon: "i:compass:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Day", Type: "title"},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "city", Name: "City", Type: "select", Options: cityOpts},
					{ID: "places", Name: "Places", Type: "relation", Collection: "@trip-places"},
					{ID: "stay", Name: "Sleeping at", Type: "text"},
					{ID: "count", Name: "Stops", Type: "rollup", Relation: "places", Target: "hours", Fn: "count_all"},
					{ID: "hours", Name: "Sightseeing hours", Type: "rollup", Relation: "places", Target: "hours", Fn: "sum"},
				},
				views: []view{
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"},` + xbShow("city") + `}`},
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"date","zoom":"week"},` + xbShow("city") + `}`},
					{Name: "Days and places", Type: "graph", Config: `{"layout":{"graph":{"color":"city","links":["places"],"external":true,"labels":"auto"}}}`},
					{Name: "Plan", Type: "list", Config: `{"sorts":[{"prop":"date","dir":"asc"}],` + xbShow("date", "city", "stay", "places") + `}`},
					{Name: "Busy days", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date"},"series":{"prop":"city"},"fn":"sum","prop":"hours","stacked":true,"legend":true}}}`},
				},
				rows: drows,
			},
			{
				key: "trip-bookings", Name: "Bookings", Icon: "i:tag:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Booking", Type: "title"},
					{ID: "type", Name: "Type", Type: "select", Options: []opt{o("flight", "Flight", "blue"), o("stay", "Stay", "purple"), o("train", "Train or ferry", "green"), o("tour", "Tour or class", "pink"), o("other", "Other", "gray")}},
					{ID: "dates", Name: "Dates", Type: "date"},
					{ID: "status", Name: "Status", Type: "status", Options: []opt{so("todo", "To book", "red", "todo"), so("booked", "Booked, pay later", "yellow", "doing"), so("paid", "Paid", "green", "done")}},
					{ID: "cost", Name: "Cost", Type: "number", Format: "dollar"},
					{ID: "provider", Name: "With", Type: "text"},
					{ID: "ref", Name: "Reference", Type: "text"},
					{ID: "city", Name: "City", Type: "select", Options: cityOpts},
				},
				views: []view{
					{Name: "Timeline", Type: "timeline", Config: `{"layout":{"date":"dates","zoom":"week","table":false},` + xbShow("status") + `}`},
					{Name: "To do", Type: "board", Config: `{"group":{"prop":"status"},"layout":{"total":{"fn":"sum","prop":"cost"}},` + xbShow("dates", "cost", "ref") + `}`},
					{Name: "All bookings", Type: "table", Config: `{"sorts":[{"prop":"dates","dir":"asc"}],"calcs":{"cost":"sum"}}`},
				},
				rows: bookings,
			},
			{
				key: "trip-budget", Name: "Trip budget", Icon: "i:wallet:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Item", Type: "title"},
					{ID: "kind", Name: "Planned or spent", Type: "select", Options: []opt{o("plan", "Planned", "gray"), o("spent", "Spent", "green")}},
					{ID: "category", Name: "Category", Type: "select", Options: catOpts},
					{ID: "amount", Name: "Amount", Type: "number", Format: "dollar"},
					{ID: "date", Name: "Paid on", Type: "date"},
					{ID: "booking", Name: "Booking", Type: "relation", Collection: "@trip-bookings"},
				},
				views: []view{
					{Name: "Budget", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"b1","size":"s","title":"Planned","chart":{"kind":"number","fn":"sum","prop":"amount","filter":{"prop":"kind","cond":"is","value":"plan"}}},` +
						`{"id":"b2","size":"s","title":"Paid so far","chart":{"kind":"number","fn":"sum","prop":"amount","filter":{"prop":"kind","cond":"is","value":"spent"}}},` +
						`{"id":"b3","title":"Paid against the plan","chart":{"kind":"gauge","fn":"sum","prop":"amount","target":8410,"filter":{"prop":"kind","cond":"is","value":"spent"}}},` +
						`{"id":"b4","wide":true,"title":"Planned and paid by category","chart":{"kind":"bar","x":{"prop":"category"},"series":{"prop":"kind"},"fn":"sum","prop":"amount","legend":true,"labels":true}},` +
						`{"id":"b5","title":"The plan","chart":{"kind":"donut","x":{"prop":"category"},"fn":"sum","prop":"amount","filter":{"prop":"kind","cond":"is","value":"plan"},"legend":true}},` +
						`{"id":"b6","title":"Paid so far, by week","chart":{"kind":"area","x":{"prop":"date","by":"week"},"fn":"sum","prop":"amount","cumulative":true,"smooth":true,"filter":{"prop":"kind","cond":"is","value":"spent"}}}` +
						`]}}`},
					{Name: "All items", Type: "table", Config: `{"group":{"prop":"kind"},"calcs":{"amount":"sum"}}`},
				},
				rows: budget,
			},
			{
				key: "trip-packing", Name: "Packing list", Icon: "i:box:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Item", Type: "title"},
					{ID: "cat", Name: "Category", Type: "select", Options: []opt{o("docs", "Documents and money", "red"), o("clothes", "Clothes", "blue"), o("toiletries", "Toiletries", "green"), o("tech", "Tech", "purple"), o("health", "Health", "pink"), o("misc", "Everything else", "gray")}},
					{ID: "who", Name: "Whose bag", Type: "select", Options: []opt{o("alex", "Alex", "orange"), o("jordan", "Jordan", "blue"), o("shared", "Shared", "gray")}},
					{ID: "qty", Name: "How many", Type: "number"},
					{ID: "packed", Name: "Packed", Type: "checkbox"},
				},
				views: []view{
					{Name: "Packing board", Type: "board", Config: `{"group":{"prop":"cat"},"layout":{"progress":"packed","size":"small"},` + xbShow("packed", "who") + `}`},
					{Name: "Still to pack", Type: "list", Config: `{"filter":{"prop":"packed","cond":"is","value":false},` + xbShow("who", "cat") + `}`},
					{Name: "By bag", Type: "table", Config: `{"group":{"prop":"who"},"calcs":{"packed":"percent_checked"}}`},
				},
				rows: irows,
			},
			{
				key: "trip-notes", Name: "Trip notes", Icon: "i:book:rose", Kind: "doc",
				markdown: `<!-- toc -->

## The route

` + fence + `mermaid
flowchart TD
    LHR([Heathrow]) -->|overnight flight| TYO[Tokyo, 4 nights]
    TYO -->|romance car| HAK[Hakone, 1 night]
    HAK -->|bullet train| KYO[Kyoto, 4 nights]
    KYO -.->|day trip| NAR[Nara]
    KYO -->|30 minutes| OSA[Osaka, 2 nights]
    OSA -->|bullet train| HIR[Hiroshima, 1 night]
    HIR -.->|ferry| MIY[Miyajima]
    HIR --> KIX[Kansai airport hotel]
    KIX -->|flight home| LHR
` + fence + `

## Useful phrases

| English | Japanese | Say it like |
| --- | --- | --- |
| Hello | Konnichiwa | kon-nee-chee-wah |
| Thank you | Arigatou gozaimasu | ah-ree-gah-toh go-zai-mass |
| Excuse me, sorry | Sumimasen | soo-mee-mah-sen |
| This one, please | Kore o kudasai | koh-reh oh koo-dah-sigh |
| How much is it? | Ikura desu ka? | ee-koo-rah dess-kah |
| The bill, please | Okaikei onegaishimasu | oh-kai-kay oh-neh-guy-shee-mass |

## Manners worth knowing

> [!IMPORTANT]
> Shoes off when you see a raised floor or a row of slippers. Toilet slippers stay in the toilet.

- No tipping. Good service is part of the price.
- Queue on the marked lines at train doors and keep phone calls off the train.
- Carry your rubbish home; public bins are rare.
- At shrines: bow, two claps, bow. At temples: no clapping.

## Money and transport

<details>
<summary>Cash or card?</summary>

Cards work in cities, but small temples, markets and buses often want cash. Take out yen at convenience store machines, which accept foreign cards.

</details>

<details>
<summary>IC cards</summary>

Add a transport card to each phone wallet before landing. It pays for metro, buses and convenience stores. Top it up at any station machine.

</details>

<details>
<summary>The rail pass</summary>

Seven days, starting the day we leave Hakone. Seat reservations are free; book them at the ticket office the day before long rides.

</details>

> [!WARNING]
> Mid-November is autumn leaf season in Kyoto. Temples get very busy after 10:00, so plan the famous ones early.
`,
				children: []Starter{
					{
						key: "trip-food", Name: "Food to try", Icon: "i:coffee:amber", Kind: "doc",
						markdown: `Tick them off as we go. Anything left over is a reason to come back.

## Tokyo

- [ ] Sushi breakfast at the outer market
- [ ] Ramen at a ticket-machine counter shop
- [ ] Convenience store egg sandwich, apparently life-changing

## Kyoto and Nara

- [ ] Matcha everything: tea, soft serve, cake
- [ ] Yudofu, simmered tofu, near the temples
- [ ] Kaiseki dinner at the ryokan (Hakone, technically)

## Osaka and Hiroshima

- [ ] Takoyaki in Dotonbori
- [ ] Okonomiyaki, both Osaka style and Hiroshima style, then argue about it
- [ ] Grilled oysters on Miyajima

| Dish | Rough price | Where |
| --- | --- | --- |
| Ramen | 1,000 yen | Anywhere with a queue |
| Takoyaki, 8 pieces | 600 yen | Street stalls |
| Okonomiyaki | 1,200 yen | Hiroshima station food hall |
`,
					},
				},
			},
			{
				key: "trip-before", Name: "Before we go", Icon: "i:checklist:green", Kind: "doc",
				markdown: `Three days to go. One list, two people, no surprises at the airport.

## Admin

- [x] Passports valid for the whole trip
- [x] Travel insurance bought and the policy saved offline
- [x] Tell the banks we are travelling
- [ ] Download offline maps for all six cities
- [ ] Add IC cards to both phones

## Bookings still open

- [ ] Bullet train seats to Hiroshima
- [ ] Airport hotel at Kansai for the last night
- [ ] Luggage forwarding from Hakone to Kyoto

{{db:trip-bookings}}

## House

- [ ] Plants to the neighbours
- [ ] Bins out on the morning we leave
- [ ] Heating on the holiday setting
`,
			},
		},
	}
}

func xbFitness() Starter {
	rnd := &xbRand{s: 7}
	type plan struct {
		phase string
		km    float64
		long  float64
	}
	plans := []plan{{"base", 22, 10}, {"base", 24, 11}, {"base", 26, 12}, {"build", 28, 13}, {"build", 30, 14}, {"build", 27, 12}, {"peak", 33, 16}, {"peak", 35, 18}, {"peak", 30, 15}, {"taper", 26, 12}, {"taper", 20, 8}, {"race", 26, 21.1}}
	routes := []string{"river", "park", "canal", "track"}
	first := -56 + ((1-(int(time.Now().Weekday())-56))%7+7)%7
	feels := []string{"great", "good", "good", "ok", "tired"}
	var workouts, runs []row
	weekRuns := map[int][]string{}
	for w := 0; w < 12; w++ {
		start := first + 7*w
		if start > -1 {
			break
		}
		p := plans[w]
		fit := float64(w) * 0.035
		add := func(day int, title, kind string, mins, effort, kcal float64) {
			if day > -1 {
				return
			}
			workouts = append(workouts, row{Title: title, Values: map[string]any{"type": kind, "date": xbDay(day), "minutes": mins, "effort": effort, "kcal": kcal, "feel": feels[rnd.n(0, 4)]}})
		}
		run := func(day int, title, route string, km, pace, hr, effort float64) {
			if day > -1 {
				return
			}
			km = xbRound(km, 1)
			mins := xbRound(km*pace, 1)
			add(day, title, "run", mins, effort, xbRound(km*68, 0))
			runs = append(runs, row{Title: title, Values: map[string]any{"workout": title, "date": xbDay(day), "distance": km, "minutes": mins, "hr": xbRound(hr, 0), "route": route, "shoes": map[bool]string{true: "trail", false: "road"}[route == "park"]}})
			weekRuns[w] = append(weekRuns[w], title)
		}
		n := w + 1
		easy := p.km - p.long - 7
		run(start+1, fmt.Sprintf("Easy run, week %d", n), routes[rnd.n(0, 2)], math.Max(5, easy*0.55+rnd.in(-0.4, 0.4)), 6.25-fit+rnd.in(-0.08, 0.08), 142+rnd.in(-4, 4), 4)
		if rnd.f() > 0.15 {
			add(start+2, fmt.Sprintf("Strength session, week %d", n), "strength", 40+float64(rnd.n(0, 3))*5, 6, 260+rnd.in(-20, 30))
		}
		if p.phase == "peak" || p.phase == "build" {
			run(start+3, fmt.Sprintf("Intervals on the track, week %d", n), "track", 7+rnd.in(-0.5, 0.8), 5.05-fit+rnd.in(-0.06, 0.06), 163+rnd.in(-3, 3), 8)
		} else {
			run(start+3, fmt.Sprintf("Tempo run, week %d", n), routes[rnd.n(0, 2)], 6+rnd.in(-0.3, 0.9), 5.35-fit+rnd.in(-0.06, 0.06), 156+rnd.in(-3, 3), 7)
		}
		if rnd.f() > 0.3 {
			add(start+4, fmt.Sprintf("Yoga and stretching, week %d", n), "yoga", 30+float64(rnd.n(0, 2))*15, 3, 120+rnd.in(-10, 20))
		}
		run(start+5, fmt.Sprintf("Long run, week %d", n), routes[rnd.n(0, 3)%3], p.long+rnd.in(-0.3, 0.3), 6.45-fit+rnd.in(-0.08, 0.08), 148+rnd.in(-3, 4), 7)
		switch w % 3 {
		case 0:
			add(start+6, fmt.Sprintf("Bike ride to the lake, week %d", n), "cycle", 70+float64(rnd.n(0, 4))*10, 5, 520+rnd.in(-40, 60))
		case 1:
			add(start+6, fmt.Sprintf("Swim at the leisure centre, week %d", n), "swim", 35+float64(rnd.n(0, 2))*5, 5, 300+rnd.in(-20, 30))
		default:
			add(start+6, fmt.Sprintf("Family walk in the woods, week %d", n), "walk", 60+float64(rnd.n(0, 3))*15, 2, 240+rnd.in(-20, 40))
		}
	}
	for i := range workouts {
		workouts[i].Values["kcal"] = xbRound(workouts[i].Values["kcal"].(float64), 0)
	}
	phaseName := map[string]string{"base": "Base", "build": "Build", "peak": "Peak", "taper": "Taper", "race": "Race week"}
	weeks := make([]row, 0, 12)
	for w, p := range plans {
		start := first + 7*w
		weeks = append(weeks, row{Title: fmt.Sprintf("Week %d · %s", w+1, phaseName[p.phase]), Values: map[string]any{"phase": p.phase, "dates": xbSpan(start, start+6), "planned": p.km, "long": p.long, "runs": weekRuns[w]}})
	}
	weeks[11].Markdown = "Race week. Two short easy runs, lots of sleep, and the half marathon on Sunday.\n\n- [ ] Pick up the race number\n- [ ] Lay out kit the night before\n- [ ] Porridge and a banana three hours before the start\n"
	type meal struct {
		meal, title          string
		kcal, pro, carb, fat float64
		veg                  float64
	}
	menu := map[string][]meal{
		"breakfast": {{"breakfast", "Porridge with berries and seeds", 420, 15, 62, 12, 1}, {"breakfast", "Greek yoghurt, granola and banana", 390, 22, 55, 9, 1}, {"breakfast", "Eggs on wholegrain toast", 450, 24, 38, 21, 0}, {"breakfast", "Smoothie with oats and peanut butter", 480, 20, 60, 16, 1}},
		"lunch":     {{"lunch", "Chicken and quinoa salad", 560, 42, 48, 18, 2}, {"lunch", "Lentil soup with bread", 510, 24, 72, 11, 2}, {"lunch", "Tuna wrap and an apple", 530, 34, 58, 15, 1}, {"lunch", "Leftover vegetable curry with rice", 620, 18, 92, 17, 3}, {"lunch", "Hummus and falafel bowl", 590, 21, 70, 22, 2}},
		"dinner":    {{"dinner", "Salmon, potatoes and greens", 680, 44, 52, 28, 2}, {"dinner", "Wholewheat pasta with turkey ragu", 720, 46, 85, 18, 2}, {"dinner", "Tofu stir-fry with noodles", 640, 30, 78, 20, 3}, {"dinner", "Bean chilli with rice", 690, 28, 98, 14, 3}, {"dinner", "Roast chicken traybake", 650, 48, 45, 26, 3}, {"dinner", "Homemade pizza night", 820, 32, 96, 30, 1}},
		"snack":     {{"snack", "Banana and almonds", 260, 7, 30, 13, 1}, {"snack", "Recovery chocolate milk", 220, 12, 34, 5, 0}, {"snack", "Rice cakes with peanut butter", 240, 8, 24, 13, 0}},
	}
	var meals []row
	for d := -14; d <= -1; d++ {
		for _, m := range []string{"breakfast", "lunch", "dinner", "snack"} {
			if m == "snack" && rnd.f() < 0.45 {
				continue
			}
			opts := menu[m]
			x := opts[rnd.n(0, len(opts)-1)]
			s := 0.92 + rnd.f()*0.16
			meals = append(meals, row{Title: x.title, Values: map[string]any{"meal": m, "date": xbDay(d), "kcal": xbRound(x.kcal*s, 0), "protein": xbRound(x.pro*s, 0), "carbs": xbRound(x.carb*s, 0), "fat": xbRound(x.fat*s, 0), "veg": x.veg, "home": rnd.f() > 0.15}})
		}
	}
	goals := []row{
		{Title: "Run 30 km a week", Values: map[string]any{"area": "running", "target": 30, "current": 27.5, "unit": "km a week", "due": "+30d"}},
		{Title: "Half marathon in under 2 hours 5", Values: map[string]any{"area": "running", "target": 125, "current": 128, "unit": "minutes, predicted", "lower": true, "due": "+30d"}, Markdown: "Predicted from the last long run and the 10K time trial. Pace needed: 5:55 per km.\n"},
		{Title: "Strength train twice a week", Values: map[string]any{"area": "strength", "target": 2, "current": 1.6, "unit": "sessions a week", "due": "+60d"}},
		{Title: "Five portions of veg a day", Values: map[string]any{"area": "food", "target": 5, "current": 4.2, "unit": "portions a day", "due": "+45d"}},
		{Title: "Protein with every meal", Values: map[string]any{"area": "food", "target": 25, "current": 26.5, "unit": "grams per meal", "due": "+45d"}},
		{Title: "Sleep seven and a half hours", Values: map[string]any{"area": "sleep", "target": 7.5, "current": 7.1, "unit": "hours a night", "due": "+30d"}},
		{Title: "Resting heart rate under 58", Values: map[string]any{"area": "health", "target": 58, "current": 61, "unit": "beats a minute", "lower": true, "due": "+90d"}},
		{Title: "Get back to 68 kg", Values: map[string]any{"area": "health", "target": 68, "current": 69.4, "unit": "kg", "lower": true, "due": "+90d"}},
	}
	gauge := func(id, title, goal string, target float64) string {
		return fmt.Sprintf(`{"id":%q,"size":"s","title":%q,"chart":{"kind":"gauge","fn":"sum","prop":"current","target":%g,"filter":{"prop":"title","cond":"is","value":%q}}}`, id, title, target, goal)
	}
	return Starter{
		ID: "ex-health-fitness", Name: "Health and fitness", Description: "Twelve weeks of training for a half marathon: workouts, runs, meals and goals, with heatmaps, pace charts and gauges.", Icon: "i:heart:rose", Kind: "doc", Category: catExamples, Tags: []string{"fitness", "running", "nutrition", "habits"},
		markdown: `Priya is training for the Riverside Half Marathon in a month. This page keeps the whole plan in one place: what she did, how fast she ran, what she ate and whether the goals are moving.

> [!TIP]
> Log a workout after every session. The heatmap fills in, the weekly totals roll up and the gauges move without any extra work.

## Every session

Each square is a day, darker means more minutes of exercise. Switch tabs for the calendar, weekly minutes by type and the full log.

{{db:fit-workouts}}

## Running

Distance, pace and heart rate for every run. Pace is worked out from distance and time, so you only type two numbers.

{{db:fit-runs}}

## The 12-week plan

Each week rolls up the runs inside it and compares the distance with the plan.

{{db:fit-weeks}}

## Goals

{{db:fit-goals}}

## Food

{{db:fit-meals}}

## Read next

- {{page:fit-plan}}: phases, pace zones and what to do when a run is missed
- {{page:fit-raceday}}: the checklist for the big day
- {{page:fit-plate}}: a simple way to build a meal
`,
		children: []Starter{
			{
				key: "fit-workouts", Name: "Workouts", Icon: "i:flame:orange", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Workout", Type: "title"},
					{ID: "type", Name: "Type", Type: "select", Options: []opt{o("run", "Run", "orange"), o("strength", "Strength", "purple"), o("yoga", "Yoga", "green"), o("cycle", "Cycle", "blue"), o("swim", "Swim", "blue"), o("walk", "Walk", "brown")}},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "minutes", Name: "Minutes", Type: "number"},
					{ID: "effort", Name: "Effort", Type: "number", Display: "bar", Max: 10},
					{ID: "kcal", Name: "Calories", Type: "number", Format: "comma"},
					{ID: "feel", Name: "How it felt", Type: "select", Options: []opt{o("great", "Great", "green"), o("good", "Good", "blue"), o("ok", "OK", "yellow"), o("tired", "Tired", "red")}},
				},
				views: []view{
					{Name: "Activity heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":3,"fn":"sum","value":"minutes","color":"green"}}}`},
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"},` + xbShow("type", "minutes") + `}`},
					{Name: "Minutes by week", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"week"},"series":{"prop":"type"},"fn":"sum","prop":"minutes","stacked":true,"legend":true}}}`},
					{Name: "By type", Type: "board", Config: `{"group":{"prop":"type"},"layout":{"total":{"fn":"sum","prop":"minutes"},"progress":"effort","size":"small"},` + xbShow("date", "minutes", "feel") + `}`},
					{Name: "Log", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"minutes":"sum","kcal":"sum","effort":"median"}}`},
					{Name: "Weekday habits", Type: "chart", Config: `{"layout":{"chart":{"kind":"heatmap","x":{"prop":"date","by":"weekday"},"series":{"prop":"type"},"fn":"count","labels":true}}}`},
				},
				rows: workouts,
			},
			{
				key: "fit-runs", Name: "Runs", Icon: "i:bolt:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Run", Type: "title"},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "distance", Name: "Distance (km)", Type: "number", Precision: 1},
					{ID: "minutes", Name: "Time (min)", Type: "number", Precision: 1},
					{ID: "pace", Name: "Pace (min per km)", Type: "formula", Expr: `round(prop("Time (min)") / prop("Distance (km)"), 2)`, Precision: 2},
					{ID: "pacetext", Name: "Pace", Type: "formula", Expr: `floor(prop("Pace (min per km)")) + ":" + padStart(format(round((prop("Pace (min per km)") - floor(prop("Pace (min per km)"))) * 60)), 2, "0") + " /km"`},
					{ID: "hr", Name: "Average heart rate", Type: "number", Precision: -1},
					{ID: "route", Name: "Route", Type: "select", Options: []opt{o("river", "River loop", "blue"), o("park", "Park hills", "green"), o("canal", "Canal path", "brown"), o("track", "Running track", "orange")}},
					{ID: "shoes", Name: "Shoes", Type: "select", Options: []opt{o("road", "Road shoes", "gray"), o("trail", "Trail shoes", "brown")}},
					{ID: "workout", Name: "Workout", Type: "relation", Collection: "@fit-workouts"},
					{ID: "effort", Name: "Effort", Type: "rollup", Relation: "workout", Target: "effort", Fn: "max"},
				},
				views: []view{
					{Name: "Running dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"r1","size":"s","title":"Total distance","chart":{"kind":"number","fn":"sum","prop":"distance"}},` +
						`{"id":"r2","size":"s","title":"Runs","chart":{"kind":"number","fn":"count"}},` +
						`{"id":"r3","title":"Last 7 days against 30 km","chart":{"kind":"gauge","fn":"sum","prop":"distance","target":30,"filter":{"prop":"date","cond":"within","value":{"within":"past_week"}}}},` +
						`{"id":"r4","wide":true,"title":"Kilometres per week","chart":{"kind":"line","x":{"prop":"date","by":"week"},"fn":"sum","prop":"distance","smooth":true,"labels":true}},` +
						`{"id":"r5","title":"Average pace per week (lower is faster)","chart":{"kind":"line","x":{"prop":"date","by":"week"},"fn":"average","prop":"pace","smooth":true}},` +
						`{"id":"r6","title":"Pace by distance","chart":{"kind":"dots","x":{"prop":"distance","step":4},"prop":"pace"}}` +
						`]}}`},
					{Name: "All runs", Type: "table", Config: `{` + xbCols("title:230", "date:120", "distance:120", "minutes:110", "pacetext:120", "hr:150", "route:140", "-pace", "-workout") + `,"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"distance":"sum","minutes":"sum","hr":"median"}}`},
					{Name: "Pace by route", Type: "chart", Config: `{"layout":{"chart":{"kind":"dots","x":{"prop":"route"},"prop":"pace"}}}`},
					{Name: "Distance histogram", Type: "chart", Config: `{"layout":{"chart":{"kind":"histogram","x":{"prop":"distance","step":2},"fn":"count","labels":true}}}`},
					{Name: "Heart rate", Type: "chart", Config: `{"layout":{"chart":{"kind":"area","x":{"prop":"date","by":"week"},"fn":"average","prop":"hr","smooth":true}}}`},
				},
				rows: runs,
			},
			{
				key: "fit-weeks", Name: "Training weeks", Icon: "i:calendar:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Week", Type: "title"},
					{ID: "phase", Name: "Phase", Type: "select", Options: []opt{o("base", "Base", "blue"), o("build", "Build", "green"), o("peak", "Peak", "orange"), o("taper", "Taper", "purple"), o("race", "Race week", "red")}},
					{ID: "dates", Name: "Dates", Type: "date"},
					{ID: "planned", Name: "Planned km", Type: "number"},
					{ID: "long", Name: "Long run km", Type: "number"},
					{ID: "runs", Name: "Runs", Type: "relation", Collection: "@fit-runs"},
					{ID: "done", Name: "Km run", Type: "rollup", Relation: "runs", Target: "distance", Fn: "sum", Precision: 1},
					{ID: "count", Name: "Runs done", Type: "rollup", Relation: "runs", Target: "distance", Fn: "count_all"},
					{ID: "share", Name: "Of plan", Type: "formula", Expr: `round(prop("Km run") / prop("Planned km"), 2)`, Format: "percent", Display: "bar", Max: 1},
				},
				views: []view{
					{Name: "Plan timeline", Type: "timeline", Config: `{"layout":{"date":"dates","zoom":"month","table":false},` + xbShow("share") + `}`},
					{Name: "Weeks", Type: "table", Config: `{` + xbCols("title:170", "phase:110", "dates:200", "planned:110", "done:100", "share:160", "count:100", "long:120", "-runs") + `}`},
					{Name: "Kilometres run", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"dates","by":"week"},"series":{"prop":"phase"},"fn":"sum","prop":"done","labels":true,"legend":true,"stacked":true}}}`},
					{Name: "By phase", Type: "board", Config: `{"group":{"prop":"phase"},"layout":{"total":{"fn":"sum","prop":"planned"}},` + xbShow("dates", "share") + `}`},
				},
				rows: weeks,
			},
			{
				key: "fit-meals", Name: "Meals", Icon: "i:apple:green", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Meal", Type: "title"},
					{ID: "meal", Name: "When", Type: "select", Options: []opt{o("breakfast", "Breakfast", "yellow"), o("lunch", "Lunch", "green"), o("dinner", "Dinner", "blue"), o("snack", "Snack", "pink")}},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "kcal", Name: "Calories", Type: "number", Precision: -1},
					{ID: "protein", Name: "Protein (g)", Type: "number", Precision: -1},
					{ID: "carbs", Name: "Carbs (g)", Type: "number"},
					{ID: "fat", Name: "Fat (g)", Type: "number"},
					{ID: "veg", Name: "Veg portions", Type: "number"},
					{ID: "home", Name: "Cooked at home", Type: "checkbox"},
				},
				views: []view{
					{Name: "Nutrition", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"m1","size":"s","title":"Typical meal, calories","chart":{"kind":"number","fn":"median","prop":"kcal"}},` +
						`{"id":"m2","size":"s","title":"Typical meal, protein (g)","chart":{"kind":"number","fn":"median","prop":"protein"}},` +
						`{"id":"m3","size":"s","title":"Veg portions, 2 weeks","chart":{"kind":"number","fn":"sum","prop":"veg"}},` +
						`{"id":"m4","size":"s","title":"Cooked at home","chart":{"kind":"number","fn":"percent_checked","prop":"home"}},` +
						`{"id":"m5","wide":true,"title":"Calories per day","chart":{"kind":"bar","x":{"prop":"date"},"series":{"prop":"meal"},"fn":"sum","prop":"kcal","stacked":true,"legend":true,"target":2300}},` +
						`{"id":"m6","title":"Where the calories come from","chart":{"kind":"donut","x":{"prop":"meal"},"fn":"sum","prop":"kcal","labels":true}},` +
						`{"id":"m7","title":"Protein per meal","chart":{"kind":"dots","x":{"prop":"meal"},"prop":"protein"}}` +
						`]}}`},
					{Name: "Food diary", Type: "table", Config: `{"group":{"prop":"date","sort":"desc"},"calcs":{"kcal":"sum","protein":"sum","veg":"sum"}}`},
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"},` + xbShow("kcal") + `}`},
				},
				rows: meals,
			},
			{
				key: "fit-goals", Name: "Goals", Icon: "i:trophy:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Goal", Type: "title"},
					{ID: "area", Name: "Area", Type: "select", Options: []opt{o("running", "Running", "orange"), o("strength", "Strength", "purple"), o("food", "Food", "green"), o("sleep", "Sleep", "blue"), o("health", "Health", "red")}},
					{ID: "current", Name: "Now", Type: "number"},
					{ID: "target", Name: "Target", Type: "number"},
					{ID: "unit", Name: "Measured in", Type: "text"},
					{ID: "lower", Name: "Lower is better", Type: "checkbox"},
					{ID: "progress", Name: "Progress", Type: "formula", Expr: `round(min(1, if(prop("Lower is better"), prop("Target") / prop("Now"), prop("Now") / prop("Target"))), 2)`, Format: "percent", Display: "ring", Max: 1},
					{ID: "due", Name: "By", Type: "date"},
				},
				views: []view{
					{Name: "Goal gauges", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						gauge("g1", "Km a week", "Run 30 km a week", 30) + "," +
						gauge("g2", "Strength sessions a week", "Strength train twice a week", 2) + "," +
						gauge("g3", "Veg portions a day", "Five portions of veg a day", 5) + "," +
						gauge("g4", "Hours of sleep", "Sleep seven and a half hours", 7.5) + "," +
						`{"id":"g5","wide":true,"title":"Progress on every goal","chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"progress","sort":"value_desc","labels":true}}` +
						`]}}`},
					{Name: "All goals", Type: "table", Config: `{` + xbCols("title:260", "progress:120", "current:90", "target:90", "unit:170", "area:110", "due:120", "lower:130") + `}`},
					{Name: "By area", Type: "gallery", Config: `{"layout":{"cover":"none","size":"small"},` + xbShow("progress", "area", "unit") + `}`},
				},
				rows: goals,
			},
			{
				key: "fit-plan", Name: "Training plan", Icon: "i:flag:orange", Kind: "doc",
				markdown: `**Race:** Riverside Half Marathon · **Goal time:** under 2 hours 5 minutes · **Plan started:** eight weeks ago

<!-- toc -->

## The shape of the plan

` + fence + `mermaid
timeline
    title Twelve weeks to race day
    Weeks 1 to 3 : Base : Easy kilometres, one tempo run
    Weeks 4 to 6 : Build : Intervals start, long run grows to 14 km
    Weeks 7 to 9 : Peak : Biggest weeks, long run up to 18 km
    Weeks 10 and 11 : Taper : Less volume, keep some speed
    Week 12 : Race week : Rest, carbs and 21.1 km
` + fence + `

## A normal week

| Day | Session | Why |
| --- | --- | --- |
| Monday | Rest | Let the legs absorb the work |
| Tuesday | Easy run | Builds the engine without stress |
| Wednesday | Strength, 40 minutes | Strong hips and calves prevent injuries |
| Thursday | Tempo or intervals | Teaches the body to hold race pace |
| Friday | Yoga or rest | Mobility and recovery |
| Saturday | Long run | The most important run of the week |
| Sunday | Bike, swim or walk | Easy movement with the family |

## Pace zones

Pace is minutes per kilometre: $\text{pace} = \dfrac{\text{time in minutes}}{\text{distance in km}}$. For the goal time the race pace is

$$\frac{125\ \text{min}}{21.1\ \text{km}} \approx 5.92\ \text{min per km} = 5\!:\!55\ \text{per km}$$

| Zone | Pace per km | Feels like | Used for |
| --- | --- | --- | --- |
| Easy | 6:10 to 6:40 | Can chat in full sentences | Easy and long runs |
| Steady | 5:50 to 6:05 | Short sentences | End of long runs |
| Tempo | 5:15 to 5:30 | A few words at a time | Tempo runs |
| Intervals | 4:50 to 5:05 | No talking | Track repeats |

> [!IMPORTANT]
> Most runs should feel easy. If every run is hard, the hard runs stop working.

## When things go wrong

<details>
<summary>I missed a run</summary>

Skip it and carry on with the plan. Never squeeze two hard days together to catch up.

</details>

<details>
<summary>Something hurts</summary>

Pain that changes how you run means stop. Two days off now beats two weeks off later. If it is still there after three days, book a physio.

</details>

<details>
<summary>I feel tired all the time</summary>

Check sleep and food first. Swap the next intervals for an easy run and look at the effort column in the workout log.

</details>
`,
				children: []Starter{
					{
						key: "fit-raceday", Name: "Race day checklist", Icon: "i:checklist:green", Kind: "doc",
						markdown: `## The week before

- [ ] Confirm the start time and wave
- [ ] Plan the journey and parking
- [ ] Two short easy runs only
- [ ] Extra carbs for the last two days

## The night before

- [ ] Pin the number on the shirt
- [ ] Charge the watch and headphones
- [ ] Lay out shoes, socks, shorts, gels and a bin bag for the start
- [ ] In bed by 22:00

## Race morning

| Time before start | What |
| --- | --- |
| 3 hours | Porridge, banana, coffee |
| 90 minutes | Leave home |
| 30 minutes | Toilet queue, short warm-up jog |
| 10 minutes | Into the start pen near the 2:05 pacer |

> [!TIP]
> Start slower than you want to. The first 5 km at 6:00 per km feels too easy, and that is the point.

## After

- [ ] Walk for 10 minutes, then eat
- [ ] Log the run and how it felt
- [ ] Rest for three days, then an easy jog
`,
					},
				},
			},
			{
				key: "fit-plate", Name: "The everyday plate", Icon: "i:leaf:green", Kind: "doc",
				markdown: `A simple way to build a meal that fuels training without counting everything.

` + fence + `mermaid
pie showData
    title Half a plate of vegetables
    "Vegetables and fruit" : 50
    "Protein" : 25
    "Starchy carbs" : 25
` + fence + `

## On hard days

Add a fist of extra carbs: more rice, a second slice of bread, a banana after the run.

## Shopping list staples

- [ ] Oats, wholegrain bread, rice, pasta
- [ ] Eggs, chicken, tinned fish, tofu, beans and lentils
- [ ] Frozen berries and frozen peas
- [ ] Greek yoghurt and milk
- [ ] Bananas, apples, whatever veg is in season

> [!NOTE]
> This is a sensible starting point, not medical advice. Talk to a doctor or dietitian about special diets.
`,
			},
		},
	}
}

func xbFinance() Starter {
	accounts := []row{
		{Title: "Joint everyday account", Values: map[string]any{"type": "everyday", "owner": "joint", "bank": "Harbour Bank", "balance": 3240.55, "rate": 0.001, "updated": "-1d"}},
		{Title: "Maya's current account", Values: map[string]any{"type": "everyday", "owner": "maya", "bank": "Harbour Bank", "balance": 812.20, "rate": 0.0, "updated": "-1d"}},
		{Title: "Sam's current account", Values: map[string]any{"type": "everyday", "owner": "sam", "bank": "Northgate Credit Union", "balance": 655.90, "rate": 0.0, "updated": "-2d"}},
		{Title: "Shared credit card", Values: map[string]any{"type": "credit", "owner": "joint", "bank": "Harbour Bank", "balance": -486.30, "rate": 0.219, "updated": "-1d"}},
		{Title: "Rainy-day savings", Values: map[string]any{"type": "savings", "owner": "joint", "bank": "Northgate Credit Union", "balance": 9850.00, "rate": 0.042, "updated": "-3d"}},
		{Title: "Holiday pot", Values: map[string]any{"type": "savings", "owner": "joint", "bank": "Harbour Bank", "balance": 1420.00, "rate": 0.038, "updated": "-3d"}},
		{Title: "Home deposit fund", Values: map[string]any{"type": "savings", "owner": "joint", "bank": "Brightwater Savings", "balance": 18600.00, "rate": 0.045, "updated": "-3d"}},
		{Title: "Car fund", Values: map[string]any{"type": "savings", "owner": "joint", "bank": "Brightwater Savings", "balance": 3600.00, "rate": 0.045, "updated": "-3d"}},
		{Title: "Kids' savings", Values: map[string]any{"type": "savings", "owner": "joint", "bank": "Northgate Credit Union", "balance": 2150.00, "rate": 0.040, "updated": "-5d"}},
		{Title: "Christmas club", Values: map[string]any{"type": "savings", "owner": "maya", "bank": "Harbour Bank", "balance": 340.00, "rate": 0.020, "updated": "-5d"}},
		{Title: "Retirement fund", Values: map[string]any{"type": "invest", "owner": "joint", "bank": "Evergreen Pensions", "balance": 42150.00, "rate": 0.0, "updated": "-7d"}},
	}
	need := map[string]string{"housing": "need", "utilities": "need", "phone": "need", "insurance": "need", "groceries": "need", "transport": "need", "health": "need", "kids": "need", "eating": "want", "subs": "want", "clothing": "want", "fun": "want", "gifts": "want", "home": "want", "care": "want"}
	acct := map[string]string{"j": "Joint everyday account", "c": "Shared credit card", "m": "Maya's current account", "s": "Sam's current account"}
	type tx struct {
		day   int
		title string
		amt   float64
		cat   string
		acct  string
		payee string
	}
	list := []tx{
		{-87, "Maya's salary, month 1", 2680, "pay", "m", "Lindenwood Clinic"},
		{-84, "Sam's salary, month 1", 2420, "pay", "s", "Fairhaven Schools"},
		{-57, "Maya's salary, month 2", 2680, "pay", "m", "Lindenwood Clinic"},
		{-54, "Sam's salary, month 2", 2420, "pay", "s", "Fairhaven Schools"},
		{-27, "Maya's salary, month 3", 2680, "pay", "m", "Lindenwood Clinic"},
		{-24, "Sam's salary, month 3", 2420, "pay", "s", "Fairhaven Schools"},
		{-86, "Savings transfers, month 1", 1500, "savings", "j", "Own accounts"},
		{-56, "Savings transfers, month 2", 1500, "savings", "j", "Own accounts"},
		{-26, "Savings transfers, month 3", 1500, "savings", "j", "Own accounts"},
		{-86, "Rent, month 1", 1650, "housing", "j", "Oakfield Lettings"},
		{-56, "Rent, month 2", 1650, "housing", "j", "Oakfield Lettings"},
		{-26, "Rent, month 3", 1650, "housing", "j", "Oakfield Lettings"},
		{-80, "Electricity bill, month 1", 104.20, "utilities", "j", "City Power"},
		{-50, "Electricity bill, month 2", 96.75, "utilities", "j", "City Power"},
		{-20, "Electricity bill, month 3", 88.40, "utilities", "j", "City Power"},
		{-45, "Water bill for the quarter", 126.30, "utilities", "j", "Riverside Water"},
		{-78, "Broadband and phones, month 1", 79.99, "phone", "j", "Linkwave"},
		{-48, "Broadband and phones, month 2", 79.99, "phone", "j", "Linkwave"},
		{-18, "Broadband and phones, month 3", 79.99, "phone", "j", "Linkwave"},
		{-85, "Car and home insurance, month 1", 112.40, "insurance", "j", "Steadfast Insurance"},
		{-55, "Car and home insurance, month 2", 112.40, "insurance", "j", "Steadfast Insurance"},
		{-25, "Car and home insurance, month 3", 112.40, "insurance", "j", "Steadfast Insurance"},
		{-82, "Streaming and music, month 1", 26.98, "subs", "c", "StreamBox"},
		{-52, "Streaming and music, month 2", 26.98, "subs", "c", "StreamBox"},
		{-22, "Streaming and music, month 3", 26.98, "subs", "c", "StreamBox"},
		{-81, "Swimming lessons, month 1", 60, "kids", "j", "Riverside Pool"},
		{-51, "Swimming lessons, month 2", 60, "kids", "j", "Riverside Pool"},
		{-21, "Swimming lessons, month 3", 60, "kids", "j", "Riverside Pool"},
		{-40, "School trip to the science museum", 35, "kids", "m", "Greenhill Primary"},
		{-89, "Weekly shop at Green Basket", 168.40, "groceries", "j", "Green Basket"},
		{-82, "Fruit, veg and bread at the market", 46.80, "groceries", "s", "Saturday Market"},
		{-75, "Big stock-up at Price Mart", 212.35, "groceries", "c", "Price Mart"},
		{-68, "Weekly shop and cleaning supplies", 154.10, "groceries", "j", "Green Basket"},
		{-61, "Top-up shop: milk, eggs and fruit", 38.60, "groceries", "m", "Corner Store"},
		{-54, "Weekly shop for the barbecue", 181.25, "groceries", "j", "Green Basket"},
		{-47, "Groceries for the family visit", 196.90, "groceries", "c", "Price Mart"},
		{-40, "Back-to-school weekly shop", 162.70, "groceries", "j", "Green Basket"},
		{-33, "Farmers market and bakery", 52.40, "groceries", "s", "Saturday Market"},
		{-26, "Monthly stock-up at Price Mart", 205.80, "groceries", "c", "Price Mart"},
		{-19, "Weekly shop and lunchbox snacks", 149.60, "groceries", "j", "Green Basket"},
		{-6, "Rainy-week shop at Green Basket", 171.30, "groceries", "j", "Green Basket"},
		{-79, "Pizza night at Luigi's", 48.50, "eating", "c", "Luigi's"},
		{-71, "Coffee and cake with friends", 18.40, "eating", "m", "The Bean Room"},
		{-63, "Anniversary dinner at The Old Mill", 126.00, "eating", "c", "The Old Mill"},
		{-49, "Takeaway curry", 42.30, "eating", "s", "Spice Garden"},
		{-36, "Lunch out after the school play", 64.80, "eating", "c", "Harbour Cafe"},
		{-15, "Burgers at the harbour", 55.20, "eating", "c", "Dockside Grill"},
		{-4, "Friday fish and chips", 31.60, "eating", "j", "The Chip Shop"},
		{-83, "Fuel at the ring road station", 64.20, "transport", "c", "Ring Road Fuel"},
		{-60, "Fuel before the coast trip", 71.90, "transport", "c", "Ring Road Fuel"},
		{-58, "Bus passes for the kids", 48.00, "transport", "m", "City Buses"},
		{-38, "Fuel and car wash", 69.50, "transport", "c", "Ring Road Fuel"},
		{-17, "Fuel for the school run", 62.80, "transport", "c", "Ring Road Fuel"},
		{-9, "Parking at the hospital", 12.50, "transport", "s", "City Parking"},
		{-74, "Pharmacy: allergy tablets", 14.60, "health", "m", "High Street Pharmacy"},
		{-43, "Dentist check-ups for the kids", 95.00, "health", "j", "Smile Dental"},
		{-11, "Pharmacy: plasters and cold medicine", 22.35, "health", "s", "High Street Pharmacy"},
		{-66, "Kids' socks and pyjamas", 36.50, "clothing", "m", "Little Threads"},
		{-42, "School shoes for Ella and Noah", 84.00, "clothing", "c", "Step Ahead Shoes"},
		{-13, "Winter coat for Sam", 119.00, "clothing", "c", "Outdoor Depot"},
		{-69, "Cinema trip for four", 46.00, "fun", "c", "Picturehouse"},
		{-34, "Paint and canvases", 28.90, "fun", "m", "Art Supply Co"},
		{-29, "Football club fees", 45.00, "fun", "s", "Riverside Juniors FC"},
		{-62, "Birthday present for Grandma", 38.00, "gifts", "m", "Book Nook"},
		{-31, "Wedding gift for Jo and Ben", 75.00, "gifts", "j", "Homeware Hall"},
		{-77, "Plants and compost", 42.75, "home", "c", "Garden World"},
		{-46, "New kettle", 34.99, "home", "c", "Homeware Hall"},
		{-23, "Paint for the hallway", 67.40, "home", "c", "Paint Place"},
		{-65, "Haircuts for the kids", 30.00, "care", "m", "Clip Joint"},
		{-16, "Haircut for Sam", 22.00, "care", "s", "Clip Joint"},
	}
	txs := make([]row, 0, len(list))
	for _, t := range list {
		kind := "spend"
		switch t.cat {
		case "pay":
			kind = "income"
		case "savings":
			kind = "saving"
		}
		v := map[string]any{"amount": t.amt, "date": xbDay(t.day), "kind": kind, "category": t.cat, "account": acct[t.acct], "payee": t.payee}
		if n, ok := need[t.cat]; ok {
			v["need"] = n
		}
		txs = append(txs, row{Title: t.title, Values: v})
	}
	type budget struct {
		id, name string
		amt      float64
	}
	budgets := []budget{
		{"housing", "Housing", 1650}, {"groceries", "Groceries", 650}, {"utilities", "Utilities", 150}, {"transport", "Transport", 140},
		{"eating", "Eating out", 120}, {"insurance", "Insurance", 115}, {"kids", "Kids", 80}, {"phone", "Phone and internet", 80},
		{"clothing", "Clothing", 60}, {"fun", "Fun and hobbies", 60}, {"health", "Health", 50}, {"home", "Home and garden", 50},
		{"gifts", "Gifts", 40}, {"care", "Personal care", 30}, {"subs", "Subscriptions", 25},
	}
	brows := make([]row, 0, len(budgets))
	for _, b := range budgets {
		brows = append(brows, row{Title: b.name, Values: map[string]any{"type": need[b.id], "budget": b.amt, "tx": xbWhere(txs, "category", b.id)}})
	}
	goals := []row{
		{Title: "Emergency fund", Values: map[string]any{"account": "Rainy-day savings", "target": 15000, "monthly": 300, "due": "+550d", "priority": "high"}, Markdown: "Six months of essential bills, kept somewhere we can reach in a day.\n\n- Essentials are about 2,500 a month, so the target is 15,000.\n- Only for real emergencies: job loss, a broken boiler, a big vet bill.\n"},
		{Title: "Summer holiday in Portugal", Values: map[string]any{"account": "Holiday pot", "target": 2400, "monthly": 150, "due": "+240d", "priority": "medium"}, Markdown: "Two weeks near Lagos in August. Flights are cheapest if we book in January.\n"},
		{Title: "Home deposit", Values: map[string]any{"account": "Home deposit fund", "target": 40000, "monthly": 900, "due": "+720d", "priority": "high"}},
		{Title: "Replace the family car", Values: map[string]any{"account": "Car fund", "target": 9000, "monthly": 200, "due": "+820d", "priority": "medium"}},
		{Title: "Kids' university fund", Values: map[string]any{"account": "Kids' savings", "target": 12000, "monthly": 100, "due": "+2900d", "priority": "low"}},
		{Title: "Christmas presents", Values: map[string]any{"account": "Christmas club", "target": 600, "monthly": 130, "due": "+66d", "priority": "medium"}},
	}
	catOpts := []opt{o("housing", "Housing", "blue"), o("groceries", "Groceries", "green"), o("utilities", "Utilities", "yellow"), o("transport", "Transport", "orange"), o("eating", "Eating out", "red"), o("insurance", "Insurance", "gray"), o("kids", "Kids", "pink"), o("phone", "Phone and internet", "purple"), o("clothing", "Clothing", "brown"), o("fun", "Fun and hobbies", "pink"), o("health", "Health", "green"), o("home", "Home and garden", "brown"), o("gifts", "Gifts", "red"), o("care", "Personal care", "purple"), o("subs", "Subscriptions", "yellow"), o("pay", "Pay", "green"), o("savings", "Savings", "blue")}
	spend := `{"prop":"kind","cond":"is","value":"spend"}`
	return Starter{
		ID: "ex-household-money", Name: "Household money", Description: "A family's accounts, three months of spending, budgets and savings goals, with charts that show where the money goes.", Icon: "i:wallet:green", Kind: "doc", Category: catExamples, Tags: []string{"budget", "savings", "spending", "family"},
		markdown: `Maya and Sam use this space to run the household money together. Every purchase lands in {{page:fin-tx}}, each one tagged with a category and the account it came from. Once a month they sit down with a cup of tea, follow {{page:fin-checkin}} and see how the quarter is going.

> [!TIP]
> Everything below is live. Add a transaction and the charts, budget bars and goal rings update straight away.

## Where the money went

Spending for the last three months, split into needs and wants, with a running total and the share for each category.

{{db:fin-tx}}

## Spending against budget

Each category rolls up its transactions, works out the average per month and compares it with the monthly budget. Red means over.

{{db:fin-budget}}

## Savings goals

Each goal is linked to the savings account that holds it, so the ring fills as the balance grows.

{{db:fin-goals}}

## Accounts

{{db:fin-accounts}}

## Our plan

- {{page:fin-plan}}: the rules we agreed and how pay day works
- {{page:fin-checkin}}: the monthly check-in, step by step
- {{page:fin-docs}}: where the important papers live
`,
		children: []Starter{
			{
				key: "fin-accounts", Name: "Accounts", Icon: "i:coin:amber", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Account", Type: "title"},
					{ID: "type", Name: "Type", Type: "select", Options: []opt{o("everyday", "Everyday", "blue"), o("savings", "Savings", "green"), o("credit", "Credit card", "red"), o("invest", "Investments", "purple")}},
					{ID: "owner", Name: "Owner", Type: "select", Options: []opt{o("joint", "Joint", "gray"), o("maya", "Maya", "pink"), o("sam", "Sam", "blue")}},
					{ID: "bank", Name: "Bank", Type: "text"},
					{ID: "balance", Name: "Balance", Type: "number", Format: "dollar", Precision: 2},
					{ID: "rate", Name: "Interest", Type: "number", Format: "percent", Precision: 1},
					{ID: "updated", Name: "Checked on", Type: "date"},
				},
				views: []view{
					{Name: "Overview", Type: "dashboard", Config: `{"layout":{"widgets":[{"id":"a1","size":"s","title":"Net worth","chart":{"kind":"number","fn":"sum","prop":"balance"}},{"id":"a2","size":"s","title":"Saved for goals","chart":{"kind":"number","fn":"sum","prop":"balance","filter":{"prop":"type","cond":"is","value":"savings"}}},{"id":"a5","size":"s","title":"Everyday money","chart":{"kind":"number","fn":"sum","prop":"balance","filter":{"prop":"type","cond":"is","value":"everyday"}}},{"id":"a6","size":"s","title":"Owed on the card","chart":{"kind":"number","fn":"sum","prop":"balance","filter":{"prop":"type","cond":"is","value":"credit"}}},{"id":"a3","title":"Where the money sits","chart":{"kind":"donut","x":{"prop":"type"},"fn":"sum","prop":"balance","filter":{"prop":"balance","cond":"gt","value":0},"legend":true}},{"id":"a7","title":"Whose name it is in","chart":{"kind":"bar","x":{"prop":"owner"},"fn":"sum","prop":"balance","labels":true}},{"id":"a4","wide":true,"title":"Balance by account","chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"balance","sort":"value_desc","labels":true}}]}}`},
					{Name: "All accounts", Type: "table", Config: `{"sorts":[{"prop":"balance","dir":"desc"}],"calcs":{"balance":"sum"}}`},
					{Name: "By owner", Type: "board", Config: `{"group":{"prop":"owner"},"layout":{"total":{"fn":"sum","prop":"balance"}},` + xbShow("balance", "type") + `}`},
				},
				rows: accounts,
			},
			{
				key: "fin-tx", Name: "Transactions", Icon: "i:list:blue", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Description", Type: "title"},
					{ID: "amount", Name: "Amount", Type: "number", Format: "dollar", Precision: 2},
					{ID: "date", Name: "Date", Type: "date"},
					{ID: "kind", Name: "Kind", Type: "select", Options: []opt{o("spend", "Spending", "orange"), o("income", "Income", "green"), o("saving", "To savings", "blue")}},
					{ID: "category", Name: "Category", Type: "select", Options: catOpts},
					{ID: "need", Name: "Need or want", Type: "select", Options: []opt{o("need", "Need", "blue"), o("want", "Want", "pink")}},
					{ID: "account", Name: "Account", Type: "relation", Collection: "@fin-accounts"},
					{ID: "payee", Name: "Paid to", Type: "text"},
				},
				views: []view{
					{Name: "Spending dashboard", Type: "dashboard", Config: `{"layout":{"widgets":[` +
						`{"id":"t1","size":"s","title":"Earned","chart":{"kind":"number","fn":"sum","prop":"amount","filter":{"prop":"kind","cond":"is","value":"income"}}},` +
						`{"id":"t2","size":"s","title":"Spent","chart":{"kind":"number","fn":"sum","prop":"amount","filter":` + spend + `}},` +
						`{"id":"t3","size":"s","title":"Moved to savings","chart":{"kind":"number","fn":"sum","prop":"amount","filter":{"prop":"kind","cond":"is","value":"saving"}}},` +
						`{"id":"t0","size":"s","title":"Purchases logged","chart":{"kind":"number","fn":"count","filter":` + spend + `}},` +
						`{"id":"t4","wide":true,"title":"Spending by week, needs and wants","chart":{"kind":"area","x":{"prop":"date","by":"week"},"series":{"prop":"need"},"fn":"sum","prop":"amount","stacked":true,"smooth":true,"legend":true,"filter":` + spend + `}},` +
						`{"id":"t5","title":"Share by category","chart":{"kind":"donut","x":{"prop":"category"},"fn":"sum","prop":"amount","sort":"value_desc","hide_empty":true,"filter":` + spend + `}},` +
						`{"id":"t6","title":"Running total this quarter","chart":{"kind":"line","x":{"prop":"date"},"fn":"sum","prop":"amount","cumulative":true,"filter":` + spend + `}}` +
						`]}}`},
					{Name: "All transactions", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"amount":"sum"}}`},
					{Name: "By month", Type: "chart", Config: `{"layout":{"chart":{"kind":"bar","x":{"prop":"date","by":"month"},"series":{"prop":"kind"},"fn":"sum","prop":"amount","legend":true,"labels":true}}}`},
					{Name: "Cumulative spend", Type: "chart", Config: `{"layout":{"chart":{"kind":"area","x":{"prop":"date","by":"week"},"fn":"sum","prop":"amount","cumulative":true,"smooth":true,"filter":` + spend + `}}}`},
					{Name: "By category", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"category"},"fn":"sum","prop":"amount","sort":"value_desc","labels":true,"hide_empty":true,"filter":` + spend + `}}}`},
					{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"},` + xbShow("amount") + `}`},
					{Name: "Daily heatmap", Type: "heatmap", Config: `{"layout":{"heatmap":{"prop":"date","domain":"month","count":3,"fn":"sum","value":"amount","color":"amber"}},"filter":` + spend + `}`},
				},
				rows: txs,
			},
			{
				key: "fin-budget", Name: "Monthly budget", Icon: "i:scale:violet", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Category", Type: "title"},
					{ID: "type", Name: "Need or want", Type: "select", Options: []opt{o("need", "Need", "blue"), o("want", "Want", "pink")}},
					{ID: "budget", Name: "Monthly budget", Type: "number", Format: "dollar"},
					{ID: "tx", Name: "Transactions", Type: "relation", Collection: "@fin-tx"},
					{ID: "spent", Name: "Spent in 3 months", Type: "rollup", Relation: "tx", Target: "amount", Fn: "sum", Format: "dollar"},
					{ID: "month", Name: "Per month", Type: "formula", Expr: `round(prop("Spent in 3 months") / 3, 2)`, Format: "dollar"},
					{ID: "used", Name: "Used", Type: "formula", Expr: `round(prop("Per month") / prop("Monthly budget"), 2)`, Format: "percent", Display: "bar", Max: 1},
					{ID: "left", Name: "Left each month", Type: "formula", Expr: `round(prop("Monthly budget") - prop("Per month"), 2)`, Format: "dollar"},
					{ID: "state", Name: "Status", Type: "formula", Expr: `if(prop("Used") > 1, "Over budget", if(prop("Used") >= 0.9, "Close to the limit", "On track"))`},
				},
				views: []view{
					{Name: "Spend against budget", Type: "table", Config: `{` + xbCols("title:170", "budget:130", "month:120", "used:170", "state:160", "left:140", "type:120", "spent:150", "-tx") + `,"sorts":[{"prop":"budget","dir":"desc"}],"calcs":{"budget":"sum","month":"sum","spent":"sum"}}`},
					{Name: "Per month", Type: "chart", Config: `{"layout":{"chart":{"kind":"hbar","x":{"prop":"title"},"fn":"sum","prop":"month","sort":"value_desc","labels":true}}}`},
					{Name: "Needs and wants", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"type"},"fn":"sum","prop":"month","labels":true,"legend":true}}}`},
					{Name: "Status", Type: "board", Config: `{"group":{"prop":"type"},"layout":{"total":{"fn":"sum","prop":"budget"}},` + xbShow("used", "month", "state") + `}`},
				},
				rows: brows,
			},
			{
				key: "fin-goals", Name: "Savings goals", Icon: "i:target:rose", Kind: "database",
				schema: []prop{
					{ID: "title", Name: "Goal", Type: "title"},
					{ID: "account", Name: "Saved in", Type: "relation", Collection: "@fin-accounts"},
					{ID: "saved", Name: "Saved", Type: "rollup", Relation: "account", Target: "balance", Fn: "sum", Format: "dollar"},
					{ID: "target", Name: "Target", Type: "number", Format: "dollar"},
					{ID: "progress", Name: "Progress", Type: "formula", Expr: `round(prop("Saved") / prop("Target"), 2)`, Format: "percent", Display: "ring", Max: 1},
					{ID: "monthly", Name: "Added each month", Type: "number", Format: "dollar"},
					{ID: "togo", Name: "Months to go", Type: "formula", Expr: `ceil((prop("Target") - prop("Saved")) / prop("Added each month")) + " months to go"`},
					{ID: "due", Name: "Target date", Type: "date"},
					{ID: "priority", Name: "Priority", Type: "select", Options: []opt{o("high", "High", "red"), o("medium", "Medium", "yellow"), o("low", "Low", "gray")}},
				},
				views: []view{
					{Name: "Goals", Type: "gallery", Config: `{"layout":{"cover":"none","size":"medium"},` + xbShow("progress", "saved", "togo", "account") + `}`},
					{Name: "Saved so far", Type: "chart", Config: `{"layout":{"chart":{"kind":"radial","x":{"prop":"title"},"fn":"sum","prop":"saved","sort":"value_desc","legend":true}}}`},
					{Name: "All goals", Type: "table", Config: `{"sorts":[{"prop":"due","dir":"asc"}],"calcs":{"saved":"sum","target":"sum"}}`},
				},
				rows: goals,
			},
			{
				key: "fin-plan", Name: "Our money plan", Icon: "i:compass:teal", Kind: "doc",
				markdown: `**Agreed by** Maya and Sam · **Last reviewed** {{date}}

<!-- toc -->

## The short version

We split every pay day into three pots: **needs**, **wants** and **future us**. Bills leave the joint account automatically, savings move on the same day, and whatever is left is ours to enjoy without guilt.

> [!IMPORTANT]
> If either of us wants to spend more than 250 on one thing that is not in the budget, we talk first.

## How pay day works

` + fence + `mermaid
flowchart LR
    A[Maya's pay] --> J[Joint everyday account]
    B[Sam's pay] --> J
    J -->|day 1| R[Rent and bills]
    J -->|day 1| S{Savings transfers}
    S --> E[Rainy-day savings]
    S --> H[Home deposit fund]
    S --> C[Car and holiday pots]
    J -->|rest of the month| W[Groceries and family spending]
    W --> CC[Shared credit card, paid in full]
` + fence + `

## Our split

| Pot | Share of take-home | Roughly per month | What goes in it |
| --- | --- | --- | --- |
| Needs | 50% | 2,550 | Rent, bills, groceries, transport, insurance |
| Wants | 20% | 1,020 | Eating out, hobbies, clothes, gifts |
| Future us | 30% | 1,530 | Emergency fund, home deposit, car, holidays |

` + fence + `mermaid
pie showData
    title Where each pay day goes
    "Needs" : 50
    "Wants" : 20
    "Future us" : 30
` + fence + `

## Rules we agreed

1. The credit card is paid off in full every month, no exceptions.
2. Any windfall (tax refund, bonus, birthday money) goes half to savings, half to fun.
3. Each of us gets 100 a month of no-questions-asked spending money.
4. Subscriptions get reviewed every three months. If nobody used it, it goes.

> [!WARNING]
> Eating out and clothing ran over budget this quarter. We agreed to plan two takeaway nights a month instead of deciding on the day.

## Questions we get asked

<details>
<summary>Why keep the emergency fund in a separate bank?</summary>

So it is not one tap away in the everyday app. It still arrives within a day when we need it.

</details>

<details>
<summary>Why track wants at all?</summary>

Not to cut them, but to choose them. Seeing the number helps us spend on what we actually enjoy.

</details>

## Next steps

- [x] Move the car fund to the higher-interest account
- [x] Set up the Christmas club transfer
- [ ] Compare broadband deals before the contract ends
- [ ] Book a pension review for Sam
`,
				children: []Starter{
					{
						key: "fin-checkin", Name: "Monthly money check-in", Icon: "i:checklist:green", Kind: "doc",
						markdown: `Thirty minutes, first Sunday of the month. Snacks are mandatory.

## Before we sit down

- [ ] Update every balance in {{page:fin-accounts}}
- [ ] Add any cash spending to {{page:fin-tx}}
- [ ] Check the credit card statement matches what we logged

## Look at the numbers

- [ ] Which categories are over budget? (the red bars below)
- [ ] Did any bill go up without warning?
- [ ] Are the savings goals still on track?

{{db:fin-budget}}

## Decide

| Question | This month's answer |
| --- | --- |
| Anything to cut back on? | Eating out: plan two takeaway nights |
| Anything to move money towards? | Christmas club, it is close |
| Anything big coming up? | Car service, school uniform |

> [!NOTE]
> Keep it kind. The point is to agree on the next month, not to blame anyone for the last one.
`,
					},
				},
			},
			{
				key: "fin-docs", Name: "Important papers", Icon: "i:archive:amber", Kind: "doc",
				markdown: `Where to find the paperwork that matters, so either of us can sort things out alone.

| Document | Where it lives | Renewal or check date |
| --- | --- | --- |
| Tenancy agreement | Blue folder, top drawer | Renews in spring |
| Car and home insurance | Email from Steadfast Insurance | Every 12 months |
| Pension statements | Evergreen Pensions online account | Yearly statement in April |
| Birth certificates and passports | Fire safe in the wardrobe | Passports: check expiry before booking flights |
| Wills | With our solicitor, copies in the fire safe | Review after big life changes |

> [!CAUTION]
> Never write passwords on this page. They live in the password manager we share.

## If something happens to one of us

1. Call the bank on the number on the back of the card.
2. The joint account keeps paying bills, nothing needs to change straight away.
3. The emergency fund covers six months of essentials.
`,
			},
		},
	}
}
