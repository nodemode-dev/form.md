package collection

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/httpx"
)

type DateVal struct {
	Start  string `json:"start"`
	End    string `json:"end,omitempty"`
	Remind string `json:"remind,omitempty"`
}

type FileVal struct {
	ID   string `json:"id,omitempty"`
	URL  string `json:"url,omitempty"`
	Name string `json:"name"`
	Mime string `json:"mime,omitempty"`
}

type PlaceVal struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Name    string  `json:"name,omitempty"`
	Address string  `json:"address,omitempty"`
}

func badValue(prop *Property, msg string) error {
	return httpx.Fail(http.StatusBadRequest, "bad_value", prop.Name+": "+msg)
}

func isNull(raw json.RawMessage) bool {
	t := bytes.TrimSpace(raw)
	return len(t) == 0 || bytes.Equal(t, []byte("null"))
}

func parseDay(s string) (time.Time, bool, bool) {
	if len(s) == 10 {
		t, err := time.Parse("2006-01-02", s)
		return t, false, err == nil
	}
	for _, l := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(l, s); err == nil {
			return t, true, true
		}
	}
	return time.Time{}, false, false
}

func (d DateVal) Times(loc *time.Location) (time.Time, time.Time, bool, bool) {
	a, ha, ok := parseDay(d.Start)
	if !ok {
		return time.Time{}, time.Time{}, false, false
	}
	a = time.Date(a.Year(), a.Month(), a.Day(), a.Hour(), a.Minute(), a.Second(), 0, loc)
	var b time.Time
	hb := false
	if d.End != "" {
		if e, h, ok := parseDay(d.End); ok {
			b = time.Date(e.Year(), e.Month(), e.Day(), e.Hour(), e.Minute(), e.Second(), 0, loc)
			hb = h
		}
	}
	return a, b, ha || hb, true
}

type normCtx struct {
	members  map[string]bool
	fileOK   func(id string) bool
	targets  func(collectionID string, ids []string) []string
	changed  bool
	assigned []string
}

func trimText(s string, limit int) string {
	if utf8.RuneCountInString(s) > limit {
		return string([]rune(s)[:limit])
	}
	return s
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func (nc *normCtx) option(p *Property, raw json.RawMessage) (string, error) {
	var id string
	if json.Unmarshal(raw, &id) == nil {
		if p.Option(id) != nil {
			return id, nil
		}
		if o := p.OptionByName(id); o != nil {
			return o.ID, nil
		}
		return "", badValue(p, "that option does not exist")
	}
	var obj struct {
		New   string `json:"new"`
		Color string `json:"color"`
	}
	if json.Unmarshal(raw, &obj) != nil || strings.TrimSpace(obj.New) == "" {
		return "", badValue(p, "expected an option id or {\"new\": name}")
	}
	if o := p.OptionByName(obj.New); o != nil {
		return o.ID, nil
	}
	if len(p.Options) >= maxOptions {
		return "", badValue(p, "too many options")
	}
	name, err := cleanName(obj.New)
	if err != nil {
		return "", badValue(p, "an option needs a name")
	}
	color := obj.Color
	if !validColor(color) {
		color = Colors[(len(p.Options)+1)%len(Colors)]
	}
	o := Option{ID: shortID(), Name: name, Color: color}
	if p.Type == TStatus {
		o.Group = "todo"
	}
	p.Options = append(p.Options, o)
	nc.changed = true
	return o.ID, nil
}

func (nc *normCtx) normalize(p *Property, raw json.RawMessage) (json.RawMessage, error) {
	if computed(p.Type) {
		return nil, badValue(p, "this property is calculated and cannot be set")
	}
	if isNull(raw) {
		return nil, nil
	}
	switch p.Type {
	case TText, TURL, TEmail, TPhone, TTitle:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			var f float64
			if json.Unmarshal(raw, &f) != nil {
				return nil, badValue(p, "expected text")
			}
			s = formula.FormatNumber(f)
		}
		limit := maxText
		if p.Type != TText {
			limit = 2000
			s = strings.TrimSpace(s)
		}
		s = trimText(s, limit)
		if s == "" {
			return nil, nil
		}
		if p.Type == TEmail {
			if _, err := mail.ParseAddress(s); err != nil {
				return nil, badValue(p, "that is not an email address")
			}
		}
		return mustJSON(s), nil
	case TNumber:
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			var s string
			if json.Unmarshal(raw, &s) != nil {
				return nil, badValue(p, "expected a number")
			}
			v, ok := formula.Text(s).ToNumber()
			if !ok {
				if strings.TrimSpace(s) == "" {
					return nil, nil
				}
				return nil, badValue(p, "expected a number")
			}
			f = v
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, badValue(p, "expected a finite number")
		}
		return mustJSON(f), nil
	case TSelect, TStatus:
		id, err := nc.option(p, raw)
		if err != nil {
			return nil, err
		}
		return mustJSON(id), nil
	case TMulti:
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, badValue(p, "expected a list of options")
		}
		out := []string{}
		seen := map[string]bool{}
		for _, it := range items {
			id, err := nc.option(p, it)
			if err != nil {
				return nil, err
			}
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return mustJSON(out), nil
	case TDate:
		var d DateVal
		if err := json.Unmarshal(raw, &d); err != nil {
			var s string
			if json.Unmarshal(raw, &s) != nil {
				return nil, badValue(p, "expected a date")
			}
			d.Start = s
		}
		if _, _, ok := parseDay(d.Start); !ok {
			return nil, badValue(p, "a date is written YYYY-MM-DD or YYYY-MM-DDTHH:MM")
		}
		if d.End != "" {
			if _, _, ok := parseDay(d.End); !ok {
				return nil, badValue(p, "the end date is not valid")
			}
			if d.End < d.Start {
				d.Start, d.End = d.End, d.Start
			}
		}
		if d.Remind != "" {
			if _, _, ok := parseDay(d.Remind); !ok {
				d.Remind = ""
			}
		}
		return mustJSON(d), nil
	case TPerson:
		var list []string
		if err := json.Unmarshal(raw, &list); err != nil {
			var one string
			if json.Unmarshal(raw, &one) != nil {
				return nil, badValue(p, "expected a list of people")
			}
			list = []string{one}
		}
		out := []string{}
		seen := map[string]bool{}
		for _, id := range list {
			if seen[id] {
				continue
			}
			if nc.members != nil && !nc.members[id] {
				return nil, badValue(p, "that person is not in this workspace")
			}
			seen[id] = true
			out = append(out, id)
		}
		if len(out) == 0 {
			return nil, nil
		}
		return mustJSON(out), nil
	case TFiles:
		var list []FileVal
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, badValue(p, "expected a list of files")
		}
		out := []FileVal{}
		for _, f := range list {
			f.Name = trimText(strings.TrimSpace(f.Name), 300)
			switch {
			case f.ID != "":
				if nc.fileOK != nil && !nc.fileOK(f.ID) {
					return nil, badValue(p, "that file is not in this workspace")
				}
				f.URL = ""
			case strings.HasPrefix(f.URL, "https://") || strings.HasPrefix(f.URL, "http://"):
				f.URL = trimText(f.URL, 2000)
				if f.Name == "" {
					f.Name = f.URL
				}
			default:
				continue
			}
			out = append(out, f)
			if len(out) == 100 {
				break
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return mustJSON(out), nil
	case TCheckbox:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			var s string
			if json.Unmarshal(raw, &s) != nil {
				return nil, badValue(p, "expected true or false")
			}
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "true", "yes", "1", "checked", "x":
				b = true
			}
		}
		if !b {
			return nil, nil
		}
		return mustJSON(true), nil
	case TRelation:
		var list []string
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, badValue(p, "expected a list of pages")
		}
		seen := map[string]bool{}
		uniq := []string{}
		for _, id := range list {
			if !seen[id] {
				seen[id] = true
				uniq = append(uniq, id)
			}
		}
		if len(uniq) > maxRelationRefs {
			return nil, badValue(p, "a relation can hold up to 2,000 pages")
		}
		if nc.targets != nil {
			uniq = nc.targets(p.Collection, uniq)
		}
		if p.Limit == 1 && len(uniq) > 1 {
			uniq = uniq[len(uniq)-1:]
		}
		if len(uniq) == 0 {
			return nil, nil
		}
		return mustJSON(uniq), nil
	case TPlace:
		var pl PlaceVal
		if err := json.Unmarshal(raw, &pl); err != nil {
			return nil, badValue(p, "expected a place")
		}
		if pl.Lat < -90 || pl.Lat > 90 || pl.Lng < -180 || pl.Lng > 180 {
			return nil, badValue(p, "the coordinates are out of range")
		}
		pl.Name = trimText(strings.TrimSpace(pl.Name), 300)
		pl.Address = trimText(strings.TrimSpace(pl.Address), 500)
		return mustJSON(pl), nil
	}
	return nil, badValue(p, "cannot be set")
}

func strList(raw json.RawMessage) []string {
	var out []string
	if isNull(raw) {
		return nil
	}
	if json.Unmarshal(raw, &out) != nil {
		var one string
		if json.Unmarshal(raw, &one) == nil && one != "" {
			return []string{one}
		}
	}
	return out
}

func str(raw json.RawMessage) string {
	var s string
	if !isNull(raw) {
		if json.Unmarshal(raw, &s) != nil {
			var f float64
			if json.Unmarshal(raw, &f) == nil {
				return formula.FormatNumber(f)
			}
		}
	}
	return s
}

func encodeValue(v formula.Value) any {
	switch v.K {
	case formula.KEmpty:
		return nil
	case formula.KNumber:
		return map[string]any{"type": "number", "value": v.N}
	case formula.KText:
		out := map[string]any{"type": "text", "value": v.S}
		if v.Href != "" {
			out["href"] = v.Href
		}
		if v.Style != "" {
			out["style"] = v.Style
		}
		return out
	case formula.KBool:
		return map[string]any{"type": "boolean", "value": v.B}
	case formula.KDate:
		d := map[string]any{"start": dateString(v.T, v.HasTime), "time": v.HasTime}
		if !v.End.IsZero() && !v.End.Equal(v.T) {
			d["end"] = dateString(v.End, v.HasTime)
		}
		return map[string]any{"type": "date", "value": d}
	case formula.KList:
		items := make([]any, len(v.L))
		for i, x := range v.L {
			items[i] = encodeValue(x)
		}
		return map[string]any{"type": "list", "value": items}
	case formula.KPerson:
		return map[string]any{"type": "person", "value": map[string]any{"id": v.S, "name": v.Name}}
	case formula.KPage:
		return map[string]any{"type": "page", "value": map[string]any{"id": v.S, "title": v.Name}}
	}
	return nil
}

func dateString(t time.Time, hasTime bool) string {
	if hasTime {
		return t.Format("2006-01-02T15:04")
	}
	return t.Format("2006-01-02")
}
