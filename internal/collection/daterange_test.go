package collection

import (
	"testing"
	"time"
)

func TestRollingWithinSpansCountWholeDaysEndingToday(t *testing.T) {
	loc := time.FixedZone("UTC-5", -5*3600)
	today := time.Date(2026, 3, 1, 0, 0, 0, 0, loc)
	cases := []struct {
		name     string
		from, to time.Time
	}{
		{"today", today, today.AddDate(0, 0, 1)},
		{"past_7_days", time.Date(2026, 2, 23, 0, 0, 0, 0, loc), time.Date(2026, 3, 2, 0, 0, 0, 0, loc)},
		{"past_30_days", time.Date(2026, 1, 31, 0, 0, 0, 0, loc), time.Date(2026, 3, 2, 0, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		a, b, ok := withinSpan(c.name, today, func() bool { return true })
		if !ok || !a.Equal(c.from) || !b.Equal(c.to) {
			t.Errorf("%s: got %v to %v (%v), want %v to %v", c.name, a, b, ok, c.from, c.to)
		}
		if a.Location() != loc {
			t.Errorf("%s: span left the viewer's zone: %v", c.name, a.Location())
		}
	}
	if _, _, ok := withinSpan("past_decade", today, func() bool { return true }); ok {
		t.Fatal("an unknown option resolved")
	}
}

func TestDateRangeCompilesOnlyForDateProperties(t *testing.T) {
	schema := Schema{{ID: "d", Type: TDate}, {ID: "c", Type: TCreatedTime}, {ID: "t", Type: TText}}
	var none *DateRange
	if f, err := none.compile(schema); f != nil || err != nil {
		t.Fatalf("no range: %v %v", f, err)
	}
	ok := []DateRange{
		{Prop: "d", Within: "today"},
		{Prop: "c", Within: "past_30_days"},
		{Prop: "d", From: "2026-02-01", To: "2026-02-01"},
	}
	for _, d := range ok {
		f, err := d.compile(schema)
		if err != nil || f == nil || f.Prop != d.Prop {
			t.Errorf("%+v: %v %v", d, f, err)
		}
	}
	bad := []DateRange{
		{Prop: "t", Within: "today"},
		{Prop: "x", Within: "today"},
		{Within: "today"},
		{Prop: "d", Within: "soon"},
		{Prop: "d", Within: "today", To: "2026-02-01"},
		{Prop: "d", From: "2026-02-02", To: "2026-02-01"},
		{Prop: "d", From: "2026-02-30", To: "2026-03-01"},
		{Prop: "d", From: "2026-2-1", To: "2026-03-01"},
		{Prop: "d"},
	}
	for _, d := range bad {
		if f, err := d.compile(schema); err == nil {
			t.Errorf("%+v compiled to %v", d, f)
		}
	}
}
