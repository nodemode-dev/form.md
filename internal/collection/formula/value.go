package formula

import (
	"math"
	"strconv"
	"strings"
	"time"
)

type Kind uint8

const (
	KEmpty Kind = iota
	KNumber
	KText
	KBool
	KDate
	KList
	KPerson
	KPage
)

var kindNames = [...]string{"empty", "number", "text", "boolean", "date", "list", "person", "page"}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "unknown"
}

type Value struct {
	K       Kind
	N       float64
	S       string
	B       bool
	T       time.Time
	End     time.Time
	HasTime bool
	L       []Value
	Name    string
	Href    string
	Style   string
}

var Empty = Value{}

func Num(f float64) Value {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Empty
	}
	return Value{K: KNumber, N: f}
}

func Text(s string) Value          { return Value{K: KText, S: s} }
func Bool(b bool) Value            { return Value{K: KBool, B: b} }
func List(vs []Value) Value        { return Value{K: KList, L: vs} }
func Person(id, name string) Value { return Value{K: KPerson, S: id, Name: name} }
func Page(id, title string) Value  { return Value{K: KPage, S: id, Name: title} }

func Date(t time.Time, hasTime bool) Value {
	if t.IsZero() {
		return Empty
	}
	return Value{K: KDate, T: t, HasTime: hasTime}
}

func Range(a, b time.Time, hasTime bool) Value {
	if a.IsZero() {
		return Empty
	}
	if b.Before(a) {
		a, b = b, a
	}
	return Value{K: KDate, T: a, End: b, HasTime: hasTime}
}

func (v Value) IsEmpty() bool {
	switch v.K {
	case KEmpty:
		return true
	case KText:
		return v.S == ""
	case KList:
		return len(v.L) == 0
	}
	return false
}

func (v Value) Truthy() bool {
	switch v.K {
	case KBool:
		return v.B
	case KNumber:
		return v.N != 0
	case KText:
		return v.S != ""
	case KList:
		return len(v.L) > 0
	case KDate, KPerson, KPage:
		return true
	}
	return false
}

func FormatNumber(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if len(s) > 18 {
		s = strconv.FormatFloat(f, 'g', 12, 64)
	}
	return s
}

func FormatDate(t time.Time, hasTime bool) string {
	if hasTime {
		return t.Format("January 2, 2006 3:04 PM")
	}
	return t.Format("January 2, 2006")
}

func (v Value) String() string {
	switch v.K {
	case KNumber:
		return FormatNumber(v.N)
	case KText:
		return v.S
	case KBool:
		if v.B {
			return "true"
		}
		return "false"
	case KDate:
		s := FormatDate(v.T, v.HasTime)
		if !v.End.IsZero() && !v.End.Equal(v.T) {
			s += " → " + FormatDate(v.End, v.HasTime)
		}
		return s
	case KList:
		parts := make([]string, len(v.L))
		for i, x := range v.L {
			parts[i] = x.String()
		}
		return strings.Join(parts, ", ")
	case KPerson, KPage:
		return v.Name
	}
	return ""
}

func Equal(a, b Value) bool {
	if a.K != b.K {
		if a.IsEmpty() && b.IsEmpty() {
			return true
		}
		return false
	}
	switch a.K {
	case KEmpty:
		return true
	case KNumber:
		return a.N == b.N
	case KText:
		return a.S == b.S
	case KBool:
		return a.B == b.B
	case KDate:
		return a.T.Equal(b.T) && a.End.Equal(b.End)
	case KPerson, KPage:
		return a.S == b.S
	case KList:
		if len(a.L) != len(b.L) {
			return false
		}
		for i := range a.L {
			if !Equal(a.L[i], b.L[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func Compare(a, b Value) (int, bool) {
	if a.IsEmpty() || b.IsEmpty() {
		switch {
		case a.IsEmpty() && b.IsEmpty():
			return 0, true
		case a.IsEmpty():
			return 1, true
		default:
			return -1, true
		}
	}
	if a.K != b.K {
		return 0, false
	}
	switch a.K {
	case KNumber:
		return cmp3(a.N < b.N, a.N > b.N), true
	case KText:
		return strings.Compare(strings.ToLower(a.S), strings.ToLower(b.S)), true
	case KBool:
		return cmp3(!a.B && b.B, a.B && !b.B), true
	case KDate:
		return cmp3(a.T.Before(b.T), a.T.After(b.T)), true
	case KPerson, KPage:
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), true
	case KList:
		for i := 0; i < len(a.L) && i < len(b.L); i++ {
			if c, ok := Compare(a.L[i], b.L[i]); ok && c != 0 {
				return c, true
			}
		}
		return cmp3(len(a.L) < len(b.L), len(a.L) > len(b.L)), true
	}
	return 0, false
}

func cmp3(less, more bool) int {
	switch {
	case less:
		return -1
	case more:
		return 1
	}
	return 0
}

func (v Value) ToNumber() (float64, bool) {
	switch v.K {
	case KNumber:
		return v.N, true
	case KBool:
		if v.B {
			return 1, true
		}
		return 0, true
	case KText:
		f, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(v.S), ",", ""), 64)
		return f, err == nil
	case KDate:
		return float64(v.T.UnixMilli()), true
	}
	return 0, false
}

func Flatten(vs []Value) []Value {
	var out []Value
	for _, v := range vs {
		if v.K == KList {
			out = append(out, Flatten(v.L)...)
		} else {
			out = append(out, v)
		}
	}
	return out
}
