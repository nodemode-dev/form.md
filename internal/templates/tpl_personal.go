package templates

var personalStarters = []Starter{
	{
		ID: "habit-tracker", Name: "Habit tracker", Description: "Tick off daily habits and watch streaks build on a calendar and a chart.", Icon: "i:leaf:green", Kind: "database", Category: catPersonal, Tags: []string{"habits", "routine", "daily"},
		schema: []prop{
			{ID: "title", Name: "Day", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "exercise", Name: "Exercise", Type: "checkbox"},
			{ID: "read", Name: "Read 20 minutes", Type: "checkbox"},
			{ID: "water", Name: "Drink water", Type: "checkbox"},
			{ID: "sleep", Name: "Hours of sleep", Type: "number"},
			{ID: "mood", Name: "Mood", Type: "select", Options: []opt{o("great", "Great", "green"), o("ok", "Okay", "yellow"), o("low", "Low", "red")}},
		},
		views: []view{
			{Name: "This week", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}],"calcs":{"exercise":"percent_checked","read":"percent_checked","water":"percent_checked","sleep":"average"}}`},
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"}}`},
			{Name: "Mood", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"mood"},"fn":"count"}}}`},
		},
		rows: []row{
			{Title: "Monday", Values: map[string]any{"date": "-3d", "exercise": true, "read": true, "water": true, "sleep": 7.5, "mood": "great"}},
			{Title: "Tuesday", Values: map[string]any{"date": "-2d", "exercise": false, "read": true, "water": true, "sleep": 6, "mood": "ok"}},
			{Title: "Wednesday", Values: map[string]any{"date": "-1d", "exercise": true, "read": false, "water": false, "sleep": 5.5, "mood": "low"}},
			{Title: "Thursday", Values: map[string]any{"date": "+0d", "exercise": true, "read": true, "water": true, "sleep": 8, "mood": "great"}},
		},
	},
	{
		ID: "journal", Name: "Journal", Description: "Daily entries with mood and tags, read back as a feed or browsed on a calendar.", Icon: "i:notebook:amber", Kind: "database", Category: catPersonal, Tags: []string{"diary", "reflection", "writing"},
		schema: []prop{
			{ID: "title", Name: "Entry", Type: "title"},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "mood", Name: "Mood", Type: "select", Options: []opt{o("bright", "Bright", "yellow"), o("calm", "Calm", "blue"), o("tired", "Tired", "gray"), o("stressed", "Stressed", "red")}},
			{ID: "tags", Name: "Tags", Type: "multi_select", Options: []opt{o("work", "Work", "blue"), o("family", "Family", "pink"), o("health", "Health", "green"), o("ideas", "Ideas", "purple")}},
		},
		views: []view{
			{Name: "Entries", Type: "feed", Config: `{}`},
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"date"}}`},
			{Name: "All entries", Type: "table", Config: `{"sorts":[{"prop":"date","dir":"desc"}]}`},
		},
		rows: []row{
			{Title: "A quiet start", Values: map[string]any{"date": "-2d", "mood": "calm", "tags": []string{"health"}}, Markdown: "Walked before breakfast and the whole day felt slower in a good way.\n\n**Grateful for:** the early light, a long call with an old friend.\n"},
			{Title: "Big presentation day", Values: map[string]any{"date": "-1d", "mood": "stressed", "tags": []string{"work"}}, Markdown: "Nervous all morning. It went better than I expected.\n\n**Next time:** rehearse out loud the night before.\n"},
			{Title: "Ideas from the weekend", Values: map[string]any{"date": "+0d", "mood": "bright", "tags": []string{"ideas", "family"}}, Markdown: "- Plan a picnic for the first warm Sunday\n- Start the herb garden on the balcony\n"},
		},
		templates: []row{
			{Title: "Today", Markdown: "## How I feel\n\n## What happened\n\n## Grateful for\n\n- \n"},
		},
	},
	{
		ID: "travel-planner", Name: "Travel planner", Description: "A trip on one page: itinerary by day, bookings, budget and a packing list.", Icon: "i:plane:blue", Kind: "doc", Category: catPersonal, Tags: []string{"trip", "itinerary", "packing"},
		markdown: `**Destination:**  · **Dates:** {{date}} to  · **Travellers:**

## Bookings

| What | Details | Confirmation | Paid |
| --- | --- | --- | --- |
| Flight out |  |  | No |
| Hotel |  |  | No |
| Flight back |  |  | No |

## Itinerary

### Day 1

- Arrive and check in
- Dinner near the hotel

### Day 2

- Morning:
- Afternoon:
- Evening:

### Day 3

- Morning:
- Afternoon:

## Budget

| Category | Planned | Spent |
| --- | --- | --- |
| Travel |  |  |
| Stay |  |  |
| Food |  |  |
| Activities |  |  |

## Packing list

- [ ] Passport and copies
- [ ] Chargers and adapter
- [ ] Medicines
- [ ] Comfortable shoes
- [ ] Rain jacket

<details>
<summary>Before you leave</summary>

- [ ] Tell the bank about travel
- [ ] Download offline maps
- [ ] Check visa and entry rules

</details>

> [!TIP]
> Save addresses in the local language for taxis.
`,
	},
	{
		ID: "budget", Name: "Budget", Description: "Income and spending by category and month, with totals and a breakdown chart.", Icon: "i:coin:green", Kind: "database", Category: catPersonal, Tags: []string{"money", "expenses", "finance"},
		schema: []prop{
			{ID: "title", Name: "Item", Type: "title"},
			{ID: "amount", Name: "Amount", Type: "number", Format: "dollar"},
			{ID: "kind", Name: "Type", Type: "select", Options: []opt{o("income", "Income", "green"), o("expense", "Expense", "red")}},
			{ID: "category", Name: "Category", Type: "select", Options: []opt{o("housing", "Housing", "brown"), o("food", "Food", "orange"), o("transport", "Transport", "blue"), o("fun", "Fun", "pink"), o("salary", "Salary", "green"), o("savings", "Savings", "purple")}},
			{ID: "date", Name: "Date", Type: "date"},
			{ID: "paid", Name: "Paid", Type: "checkbox"},
		},
		views: []view{
			{Name: "This month", Type: "table", Config: `{"group":{"prop":"kind"},"calcs":{"amount":"sum"}}`},
			{Name: "Spending", Type: "chart", Config: `{"layout":{"chart":{"kind":"donut","x":{"prop":"category"},"fn":"sum","prop":"amount"}}}`},
			{Name: "Bills", Type: "calendar", Config: `{"layout":{"date":"date"}}`},
		},
		rows: []row{
			{Title: "Salary", Values: map[string]any{"amount": 4200, "kind": "income", "category": "salary", "date": "-5d", "paid": true}},
			{Title: "Rent", Values: map[string]any{"amount": 1450, "kind": "expense", "category": "housing", "date": "-4d", "paid": true}},
			{Title: "Groceries", Values: map[string]any{"amount": 320, "kind": "expense", "category": "food", "date": "-1d", "paid": true}},
			{Title: "Train pass", Values: map[string]any{"amount": 95, "kind": "expense", "category": "transport", "date": "+2d", "paid": false}},
			{Title: "Concert tickets", Values: map[string]any{"amount": 80, "kind": "expense", "category": "fun", "date": "+12d", "paid": false}},
			{Title: "Savings transfer", Values: map[string]any{"amount": 600, "kind": "expense", "category": "savings", "date": "+1d", "paid": false}},
		},
	},
	{
		ID: "recipes", Name: "Recipes", Description: "A cookbook of recipes with course, time and rating, shown as a gallery of cards.", Icon: "i:apple:orange", Kind: "database", Category: catPersonal, Tags: []string{"cooking", "meal plan", "food"},
		schema: []prop{
			{ID: "title", Name: "Recipe", Type: "title"},
			{ID: "course", Name: "Course", Type: "select", Options: []opt{o("breakfast", "Breakfast", "yellow"), o("main", "Main", "orange"), o("side", "Side", "green"), o("dessert", "Dessert", "pink")}},
			{ID: "minutes", Name: "Minutes", Type: "number"},
			{ID: "diet", Name: "Diet", Type: "multi_select", Options: []opt{o("veg", "Vegetarian", "green"), o("vegan", "Vegan", "green"), o("gf", "Gluten free", "brown")}},
			{ID: "rating", Name: "Rating", Type: "number"},
			{ID: "source", Name: "Source", Type: "url"},
		},
		views: []view{
			{Name: "Cookbook", Type: "gallery", Config: `{"layout":{"cover":"content","size":"medium"}}`},
			{Name: "By course", Type: "board", Config: `{"group":{"prop":"course"}}`},
			{Name: "Quick meals", Type: "table", Config: `{"sorts":[{"prop":"minutes","dir":"asc"}]}`},
		},
		rows: []row{
			{Title: "Tomato and white bean soup", Values: map[string]any{"course": "main", "minutes": 30, "diet": []string{"veg", "vegan", "gf"}, "rating": 5}, Markdown: "## Ingredients\n\n- 1 onion, chopped\n- 2 cloves garlic\n- 1 tin chopped tomatoes\n- 1 tin white beans, drained\n- 500 ml vegetable stock\n\n## Method\n\n1. Soften the onion and garlic in olive oil.\n2. Add tomatoes, beans and stock. Simmer for 15 minutes.\n3. Blend half, stir back in, season and serve.\n"},
			{Title: "Overnight oats", Values: map[string]any{"course": "breakfast", "minutes": 5, "diet": []string{"veg"}, "rating": 4}},
			{Title: "Roast lemon potatoes", Values: map[string]any{"course": "side", "minutes": 50, "diet": []string{"veg", "vegan", "gf"}, "rating": 5}},
			{Title: "Dark chocolate mousse", Values: map[string]any{"course": "dessert", "minutes": 20, "diet": []string{"veg", "gf"}, "rating": 4}},
		},
	},
	{
		ID: "class-notes", Name: "Class notes", Description: "Lecture notes with key ideas, definitions, questions and a short summary to review later.", Icon: "i:cap:blue", Kind: "doc", Category: catPersonal, Tags: []string{"school", "lecture", "study"},
		markdown: `**Course:**  · **Lecture:**  · **Date:** {{date}}

## Key ideas

1.
2.
3.

## Notes

-
-

## Definitions

| Term | Meaning |
| --- | --- |
|  |  |
|  |  |

## Questions

> [!NOTE]
> Write questions as they come up, then answer them after class.

<details>
<summary>Question one</summary>

Answer:

</details>

<details>
<summary>Question two</summary>

Answer:

</details>

## Summary

Explain today's lecture in three sentences, as if to a friend who missed it.

## To do

- [ ] Review these notes within 24 hours
- [ ] Do the reading for next time
`,
	},
	{
		ID: "study-planner", Name: "Study planner", Description: "Assignments and exams by course with due dates, effort and status, on a calendar and a board.", Icon: "i:pencil:violet", Kind: "database", Category: catPersonal, Tags: []string{"homework", "exams", "student"},
		schema: []prop{
			{ID: "title", Name: "Assignment", Type: "title"},
			{ID: "course", Name: "Course", Type: "select", Options: []opt{o("math", "Calculus", "blue"), o("bio", "Biology", "green"), o("hist", "History", "brown"), o("lit", "Literature", "purple")}},
			{ID: "type", Name: "Type", Type: "select", Options: []opt{o("homework", "Homework", "gray"), o("essay", "Essay", "yellow"), o("exam", "Exam", "red"), o("project", "Project", "orange")}},
			{ID: "due", Name: "Due", Type: "date"},
			{ID: "status", Name: "Status", Type: "status", Options: statusOpts},
			{ID: "hours", Name: "Hours", Type: "number"},
			{ID: "grade", Name: "Grade", Type: "text"},
		},
		views: []view{
			{Name: "Calendar", Type: "calendar", Config: `{"layout":{"date":"due"}}`},
			{Name: "To do", Type: "board", Config: `{"group":{"prop":"status"}}`},
			{Name: "By course", Type: "table", Config: `{"group":{"prop":"course"},"sorts":[{"prop":"due","dir":"asc"}]}`},
		},
		rows: []row{
			{Title: "Problem set 6", Values: map[string]any{"course": "math", "type": "homework", "due": "+2d", "status": "doing", "hours": 3}},
			{Title: "Cell biology midterm", Values: map[string]any{"course": "bio", "type": "exam", "due": "+9d", "status": "todo", "hours": 8}, Markdown: "## Topics\n\n- [ ] Membranes and transport\n- [ ] Mitosis and meiosis\n- [ ] Respiration\n\n## Plan\n\n| Day | Focus |\n| --- | --- |\n| Mon | Flashcards |\n| Wed | Past paper |\n| Fri | Weak spots |\n"},
			{Title: "Essay on the industrial revolution", Values: map[string]any{"course": "hist", "type": "essay", "due": "+14d", "status": "todo", "hours": 6}},
			{Title: "Poetry reading response", Values: map[string]any{"course": "lit", "type": "homework", "due": "-2d", "status": "done", "hours": 1, "grade": "A-"}},
		},
	},
}
