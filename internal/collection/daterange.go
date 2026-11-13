package collection

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

type DateRange struct {
	Prop   string `json:"prop"`
	Within string `json:"within,omitempty"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
}

func badDateRange(msg string) error {
	return httpx.Fail(http.StatusBadRequest, "bad_date_range", msg)
}

func isDateProp(p *Property) bool {
	return p != nil && (p.Type == TDate || p.Type == TCreatedTime || p.Type == TEditedTime)
}

func strictDay(s string) (time.Time, bool) {
	if len(s) != 10 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

func (d *DateRange) compile(schema Schema) (*Filter, error) {
	if d == nil {
		return nil, nil
	}
	if d.Prop == "" {
		return nil, badDateRange("date_range needs a prop")
	}
	if !isDateProp(schema.ByID(d.Prop)) {
		return nil, badDateRange("date_range prop must be a date property of this database")
	}
	if d.Within != "" {
		if d.From != "" || d.To != "" {
			return nil, badDateRange("date_range takes within or from and to, not both")
		}
		if _, _, ok := withinSpan(d.Within, dayStart(time.Now()), func() bool { return true }); !ok {
			return nil, badDateRange("unknown date_range within " + d.Within)
		}
		v, _ := json.Marshal(map[string]string{"within": d.Within})
		return &Filter{Prop: d.Prop, Cond: "within", Value: v}, nil
	}
	from, ok1 := strictDay(d.From)
	to, ok2 := strictDay(d.To)
	if !ok1 || !ok2 {
		return nil, badDateRange("date_range from and to must be dates as YYYY-MM-DD")
	}
	if to.Before(from) {
		return nil, badDateRange("date_range from must be on or before to")
	}
	v, _ := json.Marshal(map[string]string{"date": d.From, "to": d.To})
	return &Filter{Prop: d.Prop, Cond: "between", Value: v}, nil
}
