package collection

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
)

const (
	TTitle          = "title"
	TText           = "text"
	TNumber         = "number"
	TSelect         = "select"
	TMulti          = "multi_select"
	TStatus         = "status"
	TDate           = "date"
	TPerson         = "person"
	TFiles          = "files"
	TCheckbox       = "checkbox"
	TURL            = "url"
	TEmail          = "email"
	TPhone          = "phone"
	TRelation       = "relation"
	TRollup         = "rollup"
	TFormula        = "formula"
	TCreatedTime    = "created_time"
	TCreatedBy      = "created_by"
	TEditedTime     = "last_edited_time"
	TEditedBy       = "last_edited_by"
	TUniqueID       = "unique_id"
	TPlace          = "place"
	TButton         = "button"
	maxProps        = 200
	maxOptions      = 500
	maxName         = 200
	maxText         = 20000
	maxRelationRefs = 2000
)

var types = map[string]bool{
	TTitle: true, TText: true, TNumber: true, TSelect: true, TMulti: true, TStatus: true, TDate: true, TPerson: true,
	TFiles: true, TCheckbox: true, TURL: true, TEmail: true, TPhone: true, TRelation: true, TRollup: true, TFormula: true,
	TCreatedTime: true, TCreatedBy: true, TEditedTime: true, TEditedBy: true, TUniqueID: true, TPlace: true, TButton: true,
}

func computed(t string) bool {
	switch t {
	case TRollup, TFormula, TCreatedTime, TCreatedBy, TEditedTime, TEditedBy, TUniqueID, TButton:
		return true
	}
	return false
}

var Colors = []string{"default", "gray", "brown", "orange", "yellow", "green", "blue", "purple", "pink", "red"}

func validColor(c string) bool {
	for _, x := range Colors {
		if x == c {
			return true
		}
	}
	return false
}

var numberFormats = map[string]bool{"": true, "number": true, "comma": true, "percent": true, "dollar": true, "euro": true, "pound": true, "yen": true,
	"rupee": true, "won": true, "yuan": true, "franc": true, "real": true, "krona": true, "ruble": true, "peso": true, "lira": true, "zloty": true, "rand": true}

var dateFormats = map[string]bool{"": true, "full": true, "short": true, "us": true, "eu": true, "iso": true, "relative": true}

var rollupFns = map[string]bool{
	"show_original": true, "show_unique": true, "count_all": true, "count_values": true, "count_unique": true, "count_empty": true, "count_not_empty": true,
	"percent_empty": true, "percent_not_empty": true, "sum": true, "average": true, "median": true, "min": true, "max": true, "range": true,
	"earliest": true, "latest": true, "date_range": true, "checked": true, "unchecked": true, "percent_checked": true, "percent_unchecked": true,
}

var StatusGroups = []string{"todo", "doing", "done"}

type Option struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Group string `json:"group,omitempty"`
}

type Action struct {
	Kind   string         `json:"kind"`
	Config map[string]any `json:"config,omitempty"`
}

type Property struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description,omitempty"`
	Options     []Option  `json:"options,omitempty"`
	Format      string    `json:"format,omitempty"`
	Display     string    `json:"display,omitempty"`
	Precision   int       `json:"precision,omitempty"`
	Max         float64   `json:"max,omitempty"`
	Color       string    `json:"color,omitempty"`
	TimeFormat  string    `json:"time_format,omitempty"`
	Collection  string    `json:"collection,omitempty"`
	Pair        string    `json:"pair,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Relation    string    `json:"relation,omitempty"`
	Target      string    `json:"target,omitempty"`
	Fn          string    `json:"fn,omitempty"`
	Expr        string    `json:"expr,omitempty"`
	Prefix      string    `json:"prefix,omitempty"`
	Actions     []Action  `json:"actions,omitempty"`
	Autofill    *Autofill `json:"autofill,omitempty"`
}

type Autofill struct {
	Prompt string `json:"prompt"`
	Auto   bool   `json:"auto"`
	By     string `json:"by,omitempty"`
}

var autofillTypes = map[string]bool{TText: true, TNumber: true, TSelect: true, TMulti: true, TCheckbox: true, TURL: true, TDate: true}

func AutofillType(t string) bool { return autofillTypes[t] }

type Schema []*Property

func (s Schema) ByID(id string) *Property {
	for _, p := range s {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (s Schema) ByName(name string) *Property {
	for _, p := range s {
		if p.Name == name {
			return p
		}
	}
	low := strings.ToLower(name)
	for _, p := range s {
		if strings.ToLower(p.Name) == low {
			return p
		}
	}
	return nil
}

func (s Schema) HasProp(name string) bool { return s.ByName(name) != nil }

func (s Schema) Clone() Schema {
	out := make(Schema, len(s))
	for i, p := range s {
		c := *p
		c.Options = append([]Option(nil), p.Options...)
		c.Actions = append([]Action(nil), p.Actions...)
		out[i] = &c
	}
	return out
}

func (p *Property) Option(id string) *Option {
	for i := range p.Options {
		if p.Options[i].ID == id {
			return &p.Options[i]
		}
	}
	return nil
}

func (p *Property) OptionByName(name string) *Option {
	low := strings.ToLower(strings.TrimSpace(name))
	for i := range p.Options {
		if strings.ToLower(p.Options[i].Name) == low {
			return &p.Options[i]
		}
	}
	return nil
}

func shortID() string { return ids.New()[:8] }

func badSchema(msg string) error { return httpx.Fail(http.StatusBadRequest, "bad_schema", msg) }

func cleanName(raw string) (string, error) {
	n := strings.Join(strings.Fields(raw), " ")
	if n == "" {
		return "", badSchema("a property needs a name")
	}
	if utf8.RuneCountInString(n) > maxName {
		return "", badSchema("a property name can be up to 200 characters")
	}
	return n, nil
}

func DefaultStatus() []Option {
	return []Option{
		{ID: shortID(), Name: "Not started", Color: "default", Group: "todo"},
		{ID: shortID(), Name: "In progress", Color: "blue", Group: "doing"},
		{ID: shortID(), Name: "Done", Color: "green", Group: "done"},
	}
}

func DefaultSchema() Schema {
	return Schema{
		{ID: TTitle, Name: "Name", Type: TTitle},
		{ID: shortID(), Name: "Tags", Type: TMulti},
	}
}

func cleanOptions(in []Option, status bool) ([]Option, error) {
	if len(in) > maxOptions {
		return nil, badSchema("a property can have up to 500 options")
	}
	seen := map[string]bool{}
	names := map[string]bool{}
	out := make([]Option, 0, len(in))
	for i, o := range in {
		name := strings.Join(strings.Fields(o.Name), " ")
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > maxName {
			return nil, badSchema("an option name can be up to 200 characters")
		}
		low := strings.ToLower(name)
		if names[low] {
			continue
		}
		names[low] = true
		if o.ID == "" || seen[o.ID] {
			o.ID = shortID()
		}
		seen[o.ID] = true
		if !validColor(o.Color) {
			o.Color = Colors[(i+1)%len(Colors)]
		}
		if status {
			if o.Group != "todo" && o.Group != "doing" && o.Group != "done" {
				o.Group = "todo"
			}
		} else {
			o.Group = ""
		}
		o.Name = name
		out = append(out, o)
	}
	return out, nil
}

func (p *Property) normalise() error {
	if !types[p.Type] {
		return badSchema("unknown property type " + p.Type)
	}
	name, err := cleanName(p.Name)
	if err != nil {
		return err
	}
	p.Name = name
	if utf8.RuneCountInString(p.Description) > 1000 {
		return badSchema("a description can be up to 1,000 characters")
	}
	keepOptions := p.Type == TSelect || p.Type == TMulti || p.Type == TStatus
	if keepOptions {
		if p.Type == TStatus && len(p.Options) == 0 {
			p.Options = DefaultStatus()
		}
		if p.Options, err = cleanOptions(p.Options, p.Type == TStatus); err != nil {
			return err
		}
	} else {
		p.Options = nil
	}
	if p.Type == TNumber || p.Type == TFormula || p.Type == TRollup {
		if !numberFormats[p.Format] && !(p.Type != TNumber && dateFormats[p.Format]) {
			p.Format = ""
		}
		if p.Display != "bar" && p.Display != "ring" {
			p.Display = ""
		}
		if p.Precision < -1 || p.Precision > 10 {
			p.Precision = 0
		}
	} else if p.Type == TDate || p.Type == TCreatedTime || p.Type == TEditedTime {
		if !dateFormats[p.Format] {
			p.Format = ""
		}
		p.Display, p.Precision, p.Max = "", 0, 0
	} else {
		p.Format, p.Display, p.Precision, p.Max = "", "", 0, 0
	}
	if p.TimeFormat != "12" && p.TimeFormat != "24" {
		p.TimeFormat = ""
	}
	if p.Color != "" && !validColor(p.Color) {
		p.Color = ""
	}
	if p.Type != TRelation {
		p.Collection, p.Pair, p.Limit = "", "", 0
	} else if p.Limit != 1 {
		p.Limit = 0
	}
	if p.Type != TRollup {
		p.Relation, p.Target, p.Fn = "", "", ""
	} else if !rollupFns[p.Fn] {
		p.Fn = "show_original"
	}
	if p.Type != TFormula {
		p.Expr = ""
	} else if utf8.RuneCountInString(p.Expr) > 10000 {
		return badSchema("a formula can be up to 10,000 characters")
	}
	if p.Type != TUniqueID {
		p.Prefix = ""
	} else {
		p.Prefix = strings.ToUpper(strings.TrimSpace(p.Prefix))
		if utf8.RuneCountInString(p.Prefix) > 10 {
			return badSchema("an ID prefix can be up to 10 characters")
		}
	}
	if p.Type != TButton {
		p.Actions = nil
	}
	if p.Autofill != nil {
		if !autofillTypes[p.Type] {
			p.Autofill = nil
		} else if utf8.RuneCountInString(p.Autofill.Prompt) > 2000 {
			return badSchema("an autofill instruction can be up to 2,000 characters")
		} else {
			p.Autofill.Prompt = strings.TrimSpace(p.Autofill.Prompt)
		}
	}
	return nil
}
