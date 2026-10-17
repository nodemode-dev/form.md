package formula

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

type builtin struct {
	min, max int
	fn       func(env Env, a []Value) (Value, error)
}

var builtins map[string]builtin

func Names() []string {
	out := make([]string, 0, len(builtins)+len(special)+len(lambdaFns)+1)
	for k := range builtins {
		out = append(out, k)
	}
	for k := range special {
		out = append(out, k)
	}
	for k := range lambdaFns {
		out = append(out, k)
	}
	out = append(out, "prop")
	sort.Strings(out)
	return out
}

func num(v Value) (float64, error) {
	if v.IsEmpty() {
		return 0, errEmpty
	}
	f, ok := v.ToNumber()
	if !ok || v.K == KDate {
		return 0, fmt.Errorf("needs a number, not a %s", v.K)
	}
	return f, nil
}

var errEmpty = errors.New("empty")

func numFn(f func(x float64) float64) func(Env, []Value) (Value, error) {
	return func(_ Env, a []Value) (Value, error) {
		x, err := num(a[0])
		if err == errEmpty {
			return Empty, nil
		}
		if err != nil {
			return Empty, err
		}
		return Num(f(x)), nil
	}
}

func num2(f func(x, y float64) Value) func(Env, []Value) (Value, error) {
	return func(_ Env, a []Value) (Value, error) {
		x, err := num(a[0])
		if err == errEmpty {
			return Empty, nil
		}
		if err != nil {
			return Empty, err
		}
		y, err := num(a[1])
		if err == errEmpty {
			return Empty, nil
		}
		if err != nil {
			return Empty, err
		}
		return f(x, y), nil
	}
}

func numbers(a []Value) []float64 {
	var out []float64
	for _, v := range Flatten(a) {
		if v.IsEmpty() {
			continue
		}
		if f, ok := v.ToNumber(); ok && v.K != KDate && v.K != KText {
			out = append(out, f)
		} else if v.K == KText {
			if f, err := strconv.ParseFloat(strings.TrimSpace(v.S), 64); err == nil {
				out = append(out, f)
			}
		}
	}
	return out
}

func text(v Value) string { return v.String() }

func date(v Value) (Value, error) {
	if v.K == KDate {
		return v, nil
	}
	if v.IsEmpty() {
		return Empty, errEmpty
	}
	if v.K == KText {
		if t, has, ok := ParseDate(v.S, time.UTC); ok {
			return Date(t, has), nil
		}
	}
	return Empty, fmt.Errorf("needs a date, not a %s", v.K)
}

var (
	reCache sync.Map
	reCount atomic.Int64
)

const reCacheMax = 512

func compileRe(pattern string) (*regexp.Regexp, error) {
	if re, ok := reCache.Load(pattern); ok {
		return re.(*regexp.Regexp), nil
	}
	if len(pattern) > 500 {
		return nil, errors.New("the pattern is too long")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("the pattern is not valid: %v", err)
	}
	if reCount.Add(1) > reCacheMax {
		reCache.Clear()
		reCount.Store(1)
	}
	reCache.Store(pattern, re)
	return re, nil
}

func runes(s string) []rune { return []rune(s) }

func sliceBounds(n int, a []Value) (int, int) {
	start, end := 0, n
	if len(a) > 1 {
		if f, err := num(a[1]); err == nil {
			start = int(f)
		}
	}
	if len(a) > 2 {
		if f, err := num(a[2]); err == nil {
			end = int(f)
		}
	}
	if start < 0 {
		start += n
	}
	if end < 0 {
		end += n
	}
	start = max(0, min(start, n))
	end = max(start, min(end, n))
	return start, end
}

func unitOf(s string) (string, error) {
	u := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), "s")
	switch u {
	case "year", "quarter", "month", "week", "day", "hour", "minute", "second", "millisecond":
		return u, nil
	}
	return "", fmt.Errorf("%q is not a unit; use years, quarters, months, weeks, days, hours or minutes", s)
}

func addUnits(t time.Time, n float64, unit string) time.Time {
	i := int(n)
	switch unit {
	case "year":
		return t.AddDate(i, 0, 0)
	case "quarter":
		return t.AddDate(0, 3*i, 0)
	case "month":
		return t.AddDate(0, i, 0)
	case "week":
		return t.AddDate(0, 0, 7*i)
	case "day":
		return t.AddDate(0, 0, i)
	case "hour":
		return t.Add(time.Duration(n * float64(time.Hour)))
	case "minute":
		return t.Add(time.Duration(n * float64(time.Minute)))
	case "second":
		return t.Add(time.Duration(n * float64(time.Second)))
	}
	return t.Add(time.Duration(n * float64(time.Millisecond)))
}

func monthsBetween(a, b time.Time) int {
	m := (a.Year()-b.Year())*12 + int(a.Month()) - int(b.Month())
	if m > 0 && a.AddDate(0, -m, 0).Before(b) {
		m--
	} else if m < 0 && a.AddDate(0, -m, 0).After(b) {
		m++
	}
	return m
}

func between(a, b time.Time, unit string) float64 {
	d := a.Sub(b)
	switch unit {
	case "year":
		return float64(monthsBetween(a, b) / 12)
	case "quarter":
		return float64(monthsBetween(a, b) / 3)
	case "month":
		return float64(monthsBetween(a, b))
	case "week":
		return math.Trunc(d.Hours() / (24 * 7))
	case "day":
		return math.Trunc(d.Hours() / 24)
	case "hour":
		return math.Trunc(d.Hours())
	case "minute":
		return math.Trunc(d.Minutes())
	case "second":
		return math.Trunc(d.Seconds())
	}
	return float64(d.Milliseconds())
}

var layouts = []struct {
	l    string
	time bool
}{
	{"2006-01-02T15:04:05Z07:00", true},
	{"2006-01-02T15:04:05", true},
	{"2006-01-02T15:04", true},
	{"2006-01-02 15:04:05", true},
	{"2006-01-02 15:04", true},
	{"2006-01-02", false},
	{"2006/01/02", false},
	{"January 2, 2006 3:04 PM", true},
	{"January 2, 2006", false},
	{"Jan 2, 2006", false},
	{"2 January 2006", false},
	{"01/02/2006", false},
}

func ParseDate(s string, loc *time.Location) (time.Time, bool, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false, false
	}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l.l, s, loc); err == nil {
			return t, l.time, true
		}
	}
	return time.Time{}, false, false
}

var momentTokens = []string{"YYYY", "YY", "MMMM", "MMM", "MM", "M", "Do", "DD", "D", "dddd", "ddd", "d", "HH", "H", "hh", "h", "mm", "m", "ss", "s", "A", "a", "Q", "WW", "W", "ww", "w", "X", "x"}

func ordinal(n int) string {
	suf := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suf = "st"
		case 2:
			suf = "nd"
		case 3:
			suf = "rd"
		}
	}
	return strconv.Itoa(n) + suf
}

func FormatMoment(t time.Time, f string) string {
	var b strings.Builder
	for i := 0; i < len(f); {
		if f[i] == '[' {
			j := strings.IndexByte(f[i:], ']')
			if j > 0 {
				b.WriteString(f[i+1 : i+j])
				i += j + 1
				continue
			}
		}
		matched := ""
		for _, tok := range momentTokens {
			if strings.HasPrefix(f[i:], tok) {
				matched = tok
				break
			}
		}
		if matched == "" {
			r, w := utf8.DecodeRuneInString(f[i:])
			b.WriteRune(r)
			i += w
			continue
		}
		_, wk := t.ISOWeek()
		h12 := t.Hour() % 12
		if h12 == 0 {
			h12 = 12
		}
		switch matched {
		case "YYYY":
			b.WriteString(strconv.Itoa(t.Year()))
		case "YY":
			fmt.Fprintf(&b, "%02d", t.Year()%100)
		case "MMMM":
			b.WriteString(t.Month().String())
		case "MMM":
			b.WriteString(t.Month().String()[:3])
		case "MM":
			fmt.Fprintf(&b, "%02d", int(t.Month()))
		case "M":
			b.WriteString(strconv.Itoa(int(t.Month())))
		case "Do":
			b.WriteString(ordinal(t.Day()))
		case "DD":
			fmt.Fprintf(&b, "%02d", t.Day())
		case "D":
			b.WriteString(strconv.Itoa(t.Day()))
		case "dddd":
			b.WriteString(t.Weekday().String())
		case "ddd":
			b.WriteString(t.Weekday().String()[:3])
		case "d":
			b.WriteString(strconv.Itoa(int(t.Weekday())))
		case "HH":
			fmt.Fprintf(&b, "%02d", t.Hour())
		case "H":
			b.WriteString(strconv.Itoa(t.Hour()))
		case "hh":
			fmt.Fprintf(&b, "%02d", h12)
		case "h":
			b.WriteString(strconv.Itoa(h12))
		case "mm":
			fmt.Fprintf(&b, "%02d", t.Minute())
		case "m":
			b.WriteString(strconv.Itoa(t.Minute()))
		case "ss":
			fmt.Fprintf(&b, "%02d", t.Second())
		case "s":
			b.WriteString(strconv.Itoa(t.Second()))
		case "A":
			if t.Hour() < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case "a":
			if t.Hour() < 12 {
				b.WriteString("am")
			} else {
				b.WriteString("pm")
			}
		case "Q":
			b.WriteString(strconv.Itoa((int(t.Month())-1)/3 + 1))
		case "WW", "ww":
			fmt.Fprintf(&b, "%02d", wk)
		case "W", "w":
			b.WriteString(strconv.Itoa(wk))
		case "X":
			b.WriteString(strconv.FormatInt(t.Unix(), 10))
		case "x":
			b.WriteString(strconv.FormatInt(t.UnixMilli(), 10))
		}
		i += len(matched)
	}
	return b.String()
}

var styles = map[string]bool{"b": true, "i": true, "u": true, "s": true, "c": true,
	"gray": true, "brown": true, "orange": true, "yellow": true, "green": true, "blue": true, "purple": true, "pink": true, "red": true,
	"gray_background": true, "brown_background": true, "orange_background": true, "yellow_background": true, "green_background": true,
	"blue_background": true, "purple_background": true, "pink_background": true, "red_background": true}

func aggregate(a []Value, f func(xs []float64) float64) (Value, error) {
	xs := numbers(a)
	if len(xs) == 0 {
		return Empty, nil
	}
	return Num(f(xs)), nil
}

func listOf(v Value) []Value {
	if v.K == KList {
		return v.L
	}
	if v.IsEmpty() {
		return nil
	}
	return []Value{v}
}

func init() {
	d1 := func(f func(t time.Time, v Value) Value) func(Env, []Value) (Value, error) {
		return func(env Env, a []Value) (Value, error) {
			v, err := date(a[0])
			if err == errEmpty {
				return Empty, nil
			}
			if err != nil {
				return Empty, err
			}
			return f(v.T, v), nil
		}
	}
	builtins = map[string]builtin{
		"not": {1, 1, func(_ Env, a []Value) (Value, error) { return Bool(!a[0].Truthy()), nil }},
		"empty": {1, 1, func(_ Env, a []Value) (Value, error) {
			return Bool(a[0].IsEmpty() || (a[0].K == KNumber && a[0].N == 0) || (a[0].K == KBool && !a[0].B)), nil
		}},
		"equal":   {2, 2, func(_ Env, a []Value) (Value, error) { return Bool(Equal(a[0], a[1])), nil }},
		"unequal": {2, 2, func(_ Env, a []Value) (Value, error) { return Bool(!Equal(a[0], a[1])), nil }},

		"concat": {1, -1, func(_ Env, a []Value) (Value, error) {
			allLists := true
			for _, v := range a {
				if v.K != KList && !v.IsEmpty() {
					allLists = false
				}
			}
			if allLists {
				var out []Value
				for _, v := range a {
					out = append(out, v.L...)
				}
				return List(out), nil
			}
			var b strings.Builder
			for _, v := range a {
				b.WriteString(text(v))
			}
			return Text(b.String()), nil
		}},
		"join": {1, -1, func(_ Env, a []Value) (Value, error) {
			if a[0].K == KList {
				sep := ""
				if len(a) > 1 {
					sep = text(a[1])
				}
				parts := make([]string, len(a[0].L))
				for i, v := range a[0].L {
					parts[i] = text(v)
				}
				return Text(strings.Join(parts, sep)), nil
			}
			sep := text(a[0])
			parts := make([]string, 0, len(a)-1)
			for _, v := range a[1:] {
				parts = append(parts, text(v))
			}
			return Text(strings.Join(parts, sep)), nil
		}},
		"slice":     {2, 3, sliceFn},
		"substring": {2, 3, sliceFn},
		"length": {1, 1, func(_ Env, a []Value) (Value, error) {
			if a[0].K == KList {
				return Num(float64(len(a[0].L))), nil
			}
			return Num(float64(utf8.RuneCountInString(text(a[0])))), nil
		}},
		"contains": {2, 2, func(_ Env, a []Value) (Value, error) {
			if a[0].K == KList {
				for _, v := range a[0].L {
					if Equal(v, a[1]) || (v.K == KText && a[1].K == KText && v.S == a[1].S) {
						return Bool(true), nil
					}
				}
				return Bool(false), nil
			}
			return Bool(strings.Contains(text(a[0]), text(a[1]))), nil
		}},
		"test": {2, 2, func(_ Env, a []Value) (Value, error) {
			re, err := compileRe(text(a[1]))
			if err != nil {
				return Empty, err
			}
			return Bool(re.MatchString(text(a[0]))), nil
		}},
		"match": {2, 2, func(_ Env, a []Value) (Value, error) {
			re, err := compileRe(text(a[1]))
			if err != nil {
				return Empty, err
			}
			var out []Value
			for _, m := range re.FindAllString(text(a[0]), 1000) {
				out = append(out, Text(m))
			}
			return List(out), nil
		}},
		"replace": {3, 3, func(_ Env, a []Value) (Value, error) {
			re, err := compileRe(text(a[1]))
			if err != nil {
				return Empty, err
			}
			s := text(a[0])
			loc := re.FindStringSubmatchIndex(s)
			if loc == nil {
				return Text(s), nil
			}
			var dst []byte
			dst = re.ExpandString(dst, text(a[2]), s, loc)
			return Text(s[:loc[0]] + string(dst) + s[loc[1]:]), nil
		}},
		"replaceall": {3, 3, func(_ Env, a []Value) (Value, error) {
			re, err := compileRe(text(a[1]))
			if err != nil {
				return Empty, err
			}
			return Text(re.ReplaceAllString(text(a[0]), text(a[2]))), nil
		}},
		"lower": {1, 1, func(_ Env, a []Value) (Value, error) { return Text(strings.ToLower(text(a[0]))), nil }},
		"upper": {1, 1, func(_ Env, a []Value) (Value, error) { return Text(strings.ToUpper(text(a[0]))), nil }},
		"repeat": {2, 2, func(_ Env, a []Value) (Value, error) {
			n, err := num(a[1])
			if err != nil {
				return Empty, nil
			}
			s := text(a[0])
			if n < 0 || float64(len(s))*n > 100000 {
				return Empty, errors.New("the result would be too long")
			}
			return Text(strings.Repeat(s, int(n))), nil
		}},
		"padstart": {3, 3, padFn(true)},
		"padend":   {3, 3, padFn(false)},
		"link": {2, 2, func(_ Env, a []Value) (Value, error) {
			v := Text(text(a[0]))
			v.Href = text(a[1])
			return v, nil
		}},
		"style": {1, -1, func(_ Env, a []Value) (Value, error) {
			v := a[0]
			if v.K != KText {
				v = Text(text(v))
			}
			var list []string
			if v.Style != "" {
				list = strings.Split(v.Style, ",")
			}
			for _, s := range a[1:] {
				k := strings.ToLower(text(s))
				if !styles[k] {
					return Empty, fmt.Errorf("%q is not a style", k)
				}
				list = append(list, k)
			}
			v.Style = strings.Join(list, ",")
			return v, nil
		}},
		"unstyle": {1, -1, func(_ Env, a []Value) (Value, error) {
			v := a[0]
			if v.K != KText {
				return Text(text(v)), nil
			}
			if len(a) == 1 {
				v.Style = ""
				return v, nil
			}
			drop := map[string]bool{}
			for _, s := range a[1:] {
				drop[strings.ToLower(text(s))] = true
			}
			var keep []string
			for _, s := range strings.Split(v.Style, ",") {
				if s != "" && !drop[s] {
					keep = append(keep, s)
				}
			}
			v.Style = strings.Join(keep, ",")
			return v, nil
		}},
		"format": {1, 1, func(_ Env, a []Value) (Value, error) { return Text(text(a[0])), nil }},
		"split": {1, 2, func(_ Env, a []Value) (Value, error) {
			sep := ""
			if len(a) > 1 {
				sep = text(a[1])
			}
			parts := strings.Split(text(a[0]), sep)
			out := make([]Value, len(parts))
			for i, p := range parts {
				out[i] = Text(p)
			}
			return List(out), nil
		}},
		"trim": {1, 1, func(_ Env, a []Value) (Value, error) { return Text(strings.TrimSpace(text(a[0]))), nil }},

		"add":      {2, 2, func(e Env, a []Value) (Value, error) { return addFn(a) }},
		"subtract": {2, 2, num2(func(x, y float64) Value { return Num(x - y) })},
		"multiply": {2, 2, num2(func(x, y float64) Value { return Num(x * y) })},
		"divide": {2, 2, num2(func(x, y float64) Value {
			if y == 0 {
				return Empty
			}
			return Num(x / y)
		})},
		"pow": {2, 2, num2(func(x, y float64) Value { return Num(math.Pow(x, y)) })},
		"mod": {2, 2, num2(func(x, y float64) Value {
			if y == 0 {
				return Empty
			}
			return Num(math.Mod(x, y))
		})},
		"abs":   {1, 1, numFn(math.Abs)},
		"ceil":  {1, 1, numFn(math.Ceil)},
		"floor": {1, 1, numFn(math.Floor)},
		"round": {1, 2, func(_ Env, a []Value) (Value, error) {
			x, err := num(a[0])
			if err == errEmpty {
				return Empty, nil
			}
			if err != nil {
				return Empty, err
			}
			places := 0.0
			if len(a) > 1 {
				if p, err := num(a[1]); err == nil {
					places = p
				}
			}
			m := math.Pow(10, places)
			return Num(math.Round(x*m) / m), nil
		}},
		"sqrt":  {1, 1, numFn(math.Sqrt)},
		"cbrt":  {1, 1, numFn(math.Cbrt)},
		"exp":   {1, 1, numFn(math.Exp)},
		"ln":    {1, 1, numFn(math.Log)},
		"log10": {1, 1, numFn(math.Log10)},
		"log2":  {1, 1, numFn(math.Log2)},
		"sign": {1, 1, numFn(func(x float64) float64 {
			switch {
			case x > 0:
				return 1
			case x < 0:
				return -1
			}
			return 0
		})},
		"min": {1, -1, func(_ Env, a []Value) (Value, error) {
			return aggregate(a, func(xs []float64) float64 {
				m := xs[0]
				for _, x := range xs {
					m = math.Min(m, x)
				}
				return m
			})
		}},
		"max": {1, -1, func(_ Env, a []Value) (Value, error) {
			return aggregate(a, func(xs []float64) float64 {
				m := xs[0]
				for _, x := range xs {
					m = math.Max(m, x)
				}
				return m
			})
		}},
		"sum": {1, -1, func(_ Env, a []Value) (Value, error) {
			xs := numbers(a)
			s := 0.0
			for _, x := range xs {
				s += x
			}
			return Num(s), nil
		}},
		"mean":    {1, -1, func(_ Env, a []Value) (Value, error) { return aggregate(a, Mean) }},
		"average": {1, -1, func(_ Env, a []Value) (Value, error) { return aggregate(a, Mean) }},
		"median":  {1, -1, func(_ Env, a []Value) (Value, error) { return aggregate(a, Median) }},
		"tonumber": {1, 1, func(_ Env, a []Value) (Value, error) {
			if a[0].K == KDate {
				return Num(float64(a[0].T.UnixMilli())), nil
			}
			f, ok := a[0].ToNumber()
			if !ok {
				return Empty, nil
			}
			return Num(f), nil
		}},
		"pi": {0, 0, func(Env, []Value) (Value, error) { return Num(math.Pi), nil }},
		"e":  {0, 0, func(Env, []Value) (Value, error) { return Num(math.E), nil }},

		"now": {0, 0, func(env Env, _ []Value) (Value, error) { return Date(env.Now().In(env.Location()), true), nil }},
		"today": {0, 0, func(env Env, _ []Value) (Value, error) {
			n := env.Now().In(env.Location())
			return Date(time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, env.Location()), false), nil
		}},
		"minute": {1, 1, d1(func(t time.Time, _ Value) Value { return Num(float64(t.Minute())) })},
		"hour":   {1, 1, d1(func(t time.Time, _ Value) Value { return Num(float64(t.Hour())) })},
		"day":    {1, 1, d1(func(t time.Time, _ Value) Value { return Num(float64(t.Weekday())) })},
		"date":   {1, 1, d1(func(t time.Time, _ Value) Value { return Num(float64(t.Day())) })},
		"week": {1, 1, d1(func(t time.Time, _ Value) Value {
			_, w := t.ISOWeek()
			return Num(float64(w))
		})},
		"month":     {1, 1, d1(func(t time.Time, _ Value) Value { return Num(float64(t.Month())) })},
		"year":      {1, 1, d1(func(t time.Time, _ Value) Value { return Num(float64(t.Year())) })},
		"datestart": {1, 1, d1(func(t time.Time, v Value) Value { return Date(t, v.HasTime) })},
		"dateend": {1, 1, d1(func(t time.Time, v Value) Value {
			if v.End.IsZero() {
				return Date(t, v.HasTime)
			}
			return Date(v.End, v.HasTime)
		})},
		"timestamp": {1, 1, d1(func(t time.Time, _ Value) Value { return Num(float64(t.UnixMilli())) })},
		"fromtimestamp": {1, 1, func(env Env, a []Value) (Value, error) {
			ms, err := num(a[0])
			if err != nil {
				return Empty, nil
			}
			return Date(time.UnixMilli(int64(ms)).In(env.Location()), true), nil
		}},
		"dateadd":      {3, 3, dateShift(1)},
		"datesubtract": {3, 3, dateShift(-1)},
		"datebetween": {3, 3, func(_ Env, a []Value) (Value, error) {
			x, err := date(a[0])
			if err == errEmpty {
				return Empty, nil
			}
			if err != nil {
				return Empty, err
			}
			y, err := date(a[1])
			if err == errEmpty {
				return Empty, nil
			}
			if err != nil {
				return Empty, err
			}
			u, err := unitOf(text(a[2]))
			if err != nil {
				return Empty, err
			}
			return Num(between(x.T, y.T, u)), nil
		}},
		"daterange": {2, 2, func(_ Env, a []Value) (Value, error) {
			x, err := date(a[0])
			if err != nil {
				return Empty, nil
			}
			y, err := date(a[1])
			if err != nil {
				return Date(x.T, x.HasTime), nil
			}
			return Range(x.T, y.T, x.HasTime || y.HasTime), nil
		}},
		"formatdate": {1, 3, func(env Env, a []Value) (Value, error) {
			v, err := date(a[0])
			if err == errEmpty {
				return Empty, nil
			}
			if err != nil {
				return Empty, err
			}
			if len(a) == 1 {
				return Text(FormatDate(v.T, v.HasTime)), nil
			}
			t := v.T
			if len(a) == 3 {
				if loc, err := time.LoadLocation(text(a[2])); err == nil {
					t = t.In(loc)
				}
			}
			return Text(FormatMoment(t, text(a[1]))), nil
		}},
		"parsedate": {1, 1, func(env Env, a []Value) (Value, error) {
			t, has, ok := ParseDate(text(a[0]), env.Location())
			if !ok {
				return Empty, nil
			}
			return Date(t, has), nil
		}},

		"at": {2, 2, func(_ Env, a []Value) (Value, error) {
			l := listOf(a[0])
			i, err := num(a[1])
			if err != nil {
				return Empty, nil
			}
			k := int(i)
			if k < 0 {
				k += len(l)
			}
			if k < 0 || k >= len(l) {
				return Empty, nil
			}
			return l[k], nil
		}},
		"first": {1, 1, func(_ Env, a []Value) (Value, error) {
			if l := listOf(a[0]); len(l) > 0 {
				return l[0], nil
			}
			return Empty, nil
		}},
		"last": {1, 1, func(_ Env, a []Value) (Value, error) {
			if l := listOf(a[0]); len(l) > 0 {
				return l[len(l)-1], nil
			}
			return Empty, nil
		}},
		"reverse": {1, 1, func(_ Env, a []Value) (Value, error) {
			if a[0].K == KText {
				r := runes(a[0].S)
				for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
					r[i], r[j] = r[j], r[i]
				}
				return Text(string(r)), nil
			}
			l := listOf(a[0])
			out := make([]Value, len(l))
			for i, v := range l {
				out[len(l)-1-i] = v
			}
			return List(out), nil
		}},
		"unique": {1, 1, func(_ Env, a []Value) (Value, error) {
			var out []Value
			for _, v := range listOf(a[0]) {
				dup := false
				for _, o := range out {
					if Equal(o, v) {
						dup = true
						break
					}
				}
				if !dup {
					out = append(out, v)
				}
			}
			return List(out), nil
		}},
		"includes": {2, 2, func(_ Env, a []Value) (Value, error) {
			for _, v := range listOf(a[0]) {
				if Equal(v, a[1]) || (v.K != a[1].K && v.String() == a[1].String()) {
					return Bool(true), nil
				}
			}
			return Bool(false), nil
		}},
		"flat": {1, 1, func(_ Env, a []Value) (Value, error) { return List(Flatten(listOf(a[0]))), nil }},

		"name": {1, 1, func(_ Env, a []Value) (Value, error) {
			if a[0].K == KList {
				out := make([]Value, len(a[0].L))
				for i, v := range a[0].L {
					out[i] = Text(v.Name)
				}
				return List(out), nil
			}
			return Text(a[0].Name), nil
		}},
		"email": {1, 1, func(env Env, a []Value) (Value, error) {
			if a[0].K == KPerson {
				return Text(env.PersonEmail(a[0].S)), nil
			}
			if a[0].K == KList {
				out := make([]Value, 0, len(a[0].L))
				for _, v := range a[0].L {
					if v.K == KPerson {
						out = append(out, Text(env.PersonEmail(v.S)))
					}
				}
				return List(out), nil
			}
			return Empty, nil
		}},
		"id": {0, 1, func(env Env, a []Value) (Value, error) {
			if len(a) == 1 && a[0].K == KPage {
				return Text(a[0].S), nil
			}
			return Text(env.RowID()), nil
		}},
	}
}

func sliceFn(_ Env, a []Value) (Value, error) {
	if a[0].K == KList {
		s, e := sliceBounds(len(a[0].L), a)
		return List(append([]Value{}, a[0].L[s:e]...)), nil
	}
	r := runes(text(a[0]))
	s, e := sliceBounds(len(r), a)
	return Text(string(r[s:e])), nil
}

func padFn(start bool) func(Env, []Value) (Value, error) {
	return func(_ Env, a []Value) (Value, error) {
		s := text(a[0])
		n, err := num(a[1])
		if err != nil || n > 10000 {
			return Text(s), nil
		}
		pad := text(a[2])
		if pad == "" {
			return Text(s), nil
		}
		need := int(n) - utf8.RuneCountInString(s)
		if need <= 0 {
			return Text(s), nil
		}
		fill := []rune(strings.Repeat(pad, need/utf8.RuneCountInString(pad)+1))[:need]
		if start {
			return Text(string(fill) + s), nil
		}
		return Text(s + string(fill)), nil
	}
}

func addFn(a []Value) (Value, error) {
	if a[0].K == KText || a[1].K == KText {
		return Text(text(a[0]) + text(a[1])), nil
	}
	x, err := num(a[0])
	if err == errEmpty {
		return a[1], nil
	}
	if err != nil {
		return Empty, err
	}
	y, err := num(a[1])
	if err == errEmpty {
		return a[0], nil
	}
	if err != nil {
		return Empty, err
	}
	return Num(x + y), nil
}

func dateShift(sign float64) func(Env, []Value) (Value, error) {
	return func(_ Env, a []Value) (Value, error) {
		v, err := date(a[0])
		if err == errEmpty {
			return Empty, nil
		}
		if err != nil {
			return Empty, err
		}
		n, err := num(a[1])
		if err != nil {
			return v, nil
		}
		u, err := unitOf(text(a[2]))
		if err != nil {
			return Empty, err
		}
		t := addUnits(v.T, sign*n, u)
		has := v.HasTime || u == "hour" || u == "minute" || u == "second"
		if !v.End.IsZero() {
			return Range(t, addUnits(v.End, sign*n, u), has), nil
		}
		return Date(t, has), nil
	}
}

func Mean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func Median(xs []float64) float64 {
	c := append([]float64{}, xs...)
	sort.Float64s(c)
	n := len(c)
	if n%2 == 1 {
		return c[n/2]
	}
	return (c[n/2-1] + c[n/2]) / 2
}
