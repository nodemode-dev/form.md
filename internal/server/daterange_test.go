package server_test

import (
	"net/http"
	"sort"
	"testing"
	"time"
)

type dateRangeFixture struct {
	owner  *client
	col    string
	view   string
	seen   string
	keep   string
	note   string
	titles map[string]string
}

func propID(t *testing.T, schema []any, name string) string {
	t.Helper()
	for _, p := range schema {
		m := p.(map[string]any)
		if m["name"] == name {
			return m["id"].(string)
		}
	}
	t.Fatalf("no property %q in %v", name, schema)
	return ""
}

func newDateRangeFixture(t *testing.T) *dateRangeFixture {
	t.Helper()
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	page := owner.mustJSON("POST", "/api/pages", map[string]any{"workspace_id": ws, "title": "Site traffic", "kind": "collection", "extra": map[string]any{"view": "table"}}, http.StatusCreated)
	col := page["page"].(map[string]any)["id"].(string)
	res := owner.mustJSON("POST", "/api/collections/"+col+"/schema", map[string]any{"ops": []any{
		map[string]any{"op": "add", "prop": map[string]any{"id": "", "name": "Seen", "type": "date"}},
		map[string]any{"op": "add", "prop": map[string]any{"id": "", "name": "Keep", "type": "checkbox"}},
		map[string]any{"op": "add", "prop": map[string]any{"id": "", "name": "Note", "type": "text"}},
	}}, http.StatusOK)
	schema := res["schema"].([]any)
	f := &dateRangeFixture{owner: owner, col: col, seen: propID(t, schema, "Seen"), keep: propID(t, schema, "Keep"), note: propID(t, schema, "Note"), titles: map[string]string{}}
	got := owner.mustJSON("GET", "/api/collections/"+col, nil, http.StatusOK)
	f.view = got["collection"].(map[string]any)["views"].([]any)[0].(map[string]any)["id"].(string)
	today := time.Now()
	day := func(back int) string { return today.AddDate(0, 0, -back).Format("2006-01-02") }
	rows := []struct {
		title string
		back  int
		keep  bool
	}{
		{"today", 0, true},
		{"six days ago", 6, true},
		{"seven days ago", 7, true},
		{"twenty nine days ago", 29, false},
		{"thirty days ago", 30, true},
		{"tomorrow", -1, true},
	}
	for _, r := range rows {
		owner.mustJSON("POST", "/api/collections/"+col+"/rows", map[string]any{"title": r.title, "values": map[string]any{f.seen: map[string]string{"start": day(r.back)}, f.keep: r.keep}}, http.StatusCreated)
	}
	owner.mustJSON("POST", "/api/collections/"+col+"/rows", map[string]any{"title": "no date", "values": map[string]any{f.keep: true}}, http.StatusCreated)
	return f
}

func (f *dateRangeFixture) titlesOf(t *testing.T, body map[string]any) []string {
	t.Helper()
	var out []string
	for _, r := range body["rows"].([]any) {
		out = append(out, r.(map[string]any)["title"].(string))
	}
	sort.Strings(out)
	return out
}

func (f *dateRangeFixture) query(t *testing.T, dr map[string]any) []string {
	t.Helper()
	body := map[string]any{"view_id": f.view, "limit": 100}
	if dr != nil {
		body["date_range"] = dr
	}
	return f.titlesOf(t, f.owner.mustJSON("POST", "/api/collections/"+f.col+"/query", body, http.StatusOK))
}

func (f *dateRangeFixture) chartCount(t *testing.T, dr map[string]any) float64 {
	t.Helper()
	body := map[string]any{"view_id": f.view, "chart": map[string]any{"kind": "number", "fn": "count"}}
	if dr != nil {
		body["date_range"] = dr
	}
	return f.owner.mustJSON("POST", "/api/collections/"+f.col+"/chart", body, http.StatusOK)["value"].(float64)
}

func sameList(a, b []string) bool {
	sort.Strings(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDateRangeNarrowsRowsAndCharts(t *testing.T) {
	f := newDateRangeFixture(t)
	if all := f.query(t, nil); len(all) != 7 {
		t.Fatalf("without a range: %v", all)
	}
	cases := []struct {
		name string
		dr   map[string]any
		want []string
	}{
		{"today", map[string]any{"prop": f.seen, "within": "today"}, []string{"today"}},
		{"past_7_days", map[string]any{"prop": f.seen, "within": "past_7_days"}, []string{"today", "six days ago"}},
		{"past_30_days", map[string]any{"prop": f.seen, "within": "past_30_days"}, []string{"today", "six days ago", "seven days ago", "twenty nine days ago"}},
		{"custom", map[string]any{"prop": f.seen, "from": time.Now().AddDate(0, 0, -7).Format("2006-01-02"), "to": time.Now().AddDate(0, 0, -6).Format("2006-01-02")}, []string{"six days ago", "seven days ago"}},
	}
	for _, c := range cases {
		got := f.query(t, c.dr)
		if !sameList(got, c.want) {
			t.Errorf("%s rows: got %v, want %v", c.name, got, c.want)
		}
		if n := f.chartCount(t, c.dr); int(n) != len(c.want) {
			t.Errorf("%s chart count: got %v, want %d", c.name, n, len(c.want))
		}
	}
}

func TestDateRangeOnlyNarrowsTheSavedViewFilter(t *testing.T) {
	f := newDateRangeFixture(t)
	f.owner.mustJSON("PATCH", "/api/views/"+f.view, map[string]any{"config": map[string]any{"filter": map[string]any{"prop": f.keep, "cond": "is", "value": true}}}, http.StatusOK)
	dr := map[string]any{"prop": f.seen, "within": "past_30_days"}
	want := []string{"today", "six days ago", "seven days ago"}
	if got := f.query(t, dr); !sameList(got, want) {
		t.Fatalf("range with the view filter: got %v, want %v", got, want)
	}
	if n := f.chartCount(t, dr); int(n) != len(want) {
		t.Fatalf("chart with the view filter: got %v, want %d", n, len(want))
	}
}

func TestDateRangeIsValidatedStrictly(t *testing.T) {
	f := newDateRangeFixture(t)
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	bad := []map[string]any{
		{"within": "today"},
		{"prop": "nope", "within": "today"},
		{"prop": f.note, "within": "today"},
		{"prop": f.keep, "from": yesterday, "to": today},
		{"prop": f.seen, "within": "past_decade"},
		{"prop": f.seen, "within": "today", "from": yesterday, "to": today},
		{"prop": f.seen, "from": today, "to": yesterday},
		{"prop": f.seen, "from": yesterday},
		{"prop": f.seen, "from": "2026-13-01", "to": "2026-13-02"},
		{"prop": f.seen, "from": "2026-11-01T10:00", "to": "2026-11-02"},
		{"prop": f.seen},
	}
	for _, dr := range bad {
		for _, path := range []string{"/query", "/chart"} {
			body := map[string]any{"view_id": f.view, "date_range": dr}
			if path == "/chart" {
				body["chart"] = map[string]any{"kind": "number", "fn": "count"}
			}
			r := f.owner.do("POST", "/api/collections/"+f.col+path, body)
			if r.Status != http.StatusBadRequest || r.Code(t) != "bad_date_range" {
				t.Errorf("%s %v: status %d %s, want 400 bad_date_range", path, dr, r.Status, r.Body)
			}
		}
	}
	if r := f.owner.do("POST", "/api/collections/"+f.col+"/query", map[string]any{"view_id": f.view, "date_range": map[string]any{"prop": f.seen, "within": "today", "extra": true}}); r.Status != http.StatusBadRequest {
		t.Errorf("an unknown date_range key: status %d %s, want 400", r.Status, r.Body)
	}
	if got := f.query(t, map[string]any{"prop": f.seen, "from": today, "to": today}); !sameList(got, []string{"today"}) {
		t.Fatalf("a one-day custom range: %v", got)
	}
}
