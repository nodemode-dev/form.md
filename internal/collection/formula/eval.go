package formula

import (
	"math"
	"sort"
	"strings"
	"time"
)

type Env interface {
	Prop(name string) (Value, error)
	PageProp(pageID, name string) (Value, error)
	Now() time.Time
	Location() *time.Location
	RowID() string
	PersonEmail(id string) string
}

type Program struct {
	src   string
	root  node
	props []string
}

func (p *Program) Source() string  { return p.src }
func (p *Program) Props() []string { return p.props }

type Schema interface {
	HasProp(name string) bool
}

func Compile(src string, schema Schema) (*Program, error) {
	root, err := parse(src)
	if err != nil {
		return nil, err
	}
	c := &checker{schema: schema, seen: map[string]bool{}}
	if err := c.walk(root, map[string]bool{}); err != nil {
		return nil, err
	}
	return &Program{src: src, root: root, props: c.props}, nil
}

type checker struct {
	schema Schema
	seen   map[string]bool
	props  []string
}

var lambdaFns = map[string]bool{"map": true, "filter": true, "find": true, "findindex": true, "some": true, "every": true, "sort": true, "count": true}

func (c *checker) walk(n node, vars map[string]bool) error {
	switch x := n.(type) {
	case *listLit:
		for _, it := range x.items {
			if err := c.walk(it, vars); err != nil {
				return err
			}
		}
	case *unaryNode:
		return c.walk(x.x, vars)
	case *binaryNode:
		if err := c.walk(x.l, vars); err != nil {
			return err
		}
		return c.walk(x.r, vars)
	case *condNode:
		for _, y := range []node{x.c, x.a, x.b} {
			if err := c.walk(y, vars); err != nil {
				return err
			}
		}
	case *ident:
		low := strings.ToLower(x.name)
		if vars[x.name] || low == "current" || low == "index" || low == "pi" || low == "e" {
			if (low == "current" || low == "index") && !vars["#lambda"] && !vars[x.name] {
				return errAt(x.pos, "%s can only be used inside map, filter and similar functions", low)
			}
			return nil
		}
		return errAt(x.pos, "%q is not a property, variable or function; properties are written prop(\"Name\")", x.name)
	case *callNode:
		switch x.name {
		case "prop":
			if x.method {
				if len(x.args) != 2 {
					return errAt(x.pos, "prop takes one property name")
				}
				if _, ok := x.args[1].(*strLit); !ok {
					return errAt(x.pos, "prop needs a property name in quotes")
				}
				return c.walk(x.args[0], vars)
			}
			if len(x.args) != 1 {
				return errAt(x.pos, "prop takes one property name")
			}
			lit, ok := x.args[0].(*strLit)
			if !ok {
				return errAt(x.pos, "prop needs a property name in quotes")
			}
			if c.schema != nil && !c.schema.HasProp(lit.v) {
				return errAt(x.pos, "there is no property called %q", lit.v)
			}
			if !c.seen[lit.v] {
				c.seen[lit.v] = true
				c.props = append(c.props, lit.v)
			}
			return nil
		case "let":
			if len(x.args) != 3 {
				return errAt(x.pos, "let takes a name, a value and an expression")
			}
			name, ok := x.args[0].(*ident)
			if !ok {
				return errAt(x.pos, "let needs a variable name first")
			}
			if err := c.walk(x.args[1], vars); err != nil {
				return err
			}
			inner := copyVars(vars)
			inner[name.name] = true
			return c.walk(x.args[2], inner)
		case "lets":
			if len(x.args) < 3 || len(x.args)%2 == 0 {
				return errAt(x.pos, "lets takes pairs of names and values, then an expression")
			}
			inner := copyVars(vars)
			for i := 0; i+1 < len(x.args); i += 2 {
				name, ok := x.args[i].(*ident)
				if !ok {
					return errAt(x.args[i].at(), "lets needs a variable name here")
				}
				if err := c.walk(x.args[i+1], inner); err != nil {
					return err
				}
				inner[name.name] = true
			}
			return c.walk(x.args[len(x.args)-1], inner)
		}
		spec, ok := builtins[x.name]
		if !ok && !lambdaFns[x.name] && !special[x.name] {
			return errAt(x.pos, "%s is not a function", x.name)
		}
		if ok && (len(x.args) < spec.min || (spec.max >= 0 && len(x.args) > spec.max)) {
			return errAt(x.pos, "%s takes %s", x.name, arity(spec.min, spec.max))
		}
		for i, a := range x.args {
			v := vars
			if lambdaFns[x.name] && i == 1 {
				v = copyVars(vars)
				v["#lambda"] = true
			}
			if err := c.walk(a, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func arity(min, max int) string {
	switch {
	case max < 0:
		return plural(min, "argument") + " or more"
	case min == max:
		return plural(min, "argument")
	}
	return FormatNumber(float64(min)) + " to " + plural(max, "argument")
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return FormatNumber(float64(n)) + " " + word + "s"
}

func copyVars(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

var special = map[string]bool{"if": true, "ifs": true, "and": true, "or": true, "let": true, "lets": true}

type scope struct {
	parent *scope
	vars   map[string]Value
}

func (s *scope) get(name string) (Value, bool) {
	for c := s; c != nil; c = c.parent {
		if v, ok := c.vars[name]; ok {
			return v, true
		}
	}
	return Empty, false
}

func (s *scope) with(kv map[string]Value) *scope { return &scope{parent: s, vars: kv} }

type run struct {
	env   Env
	steps int
}

const maxSteps = 200000

func (p *Program) Eval(env Env) (v Value, err error) {
	r := &run{env: env}
	defer func() {
		if rec := recover(); rec != nil {
			if fe, ok := rec.(*Error); ok {
				v, err = Empty, fe
				return
			}
			panic(rec)
		}
	}()
	return r.eval(p.root, &scope{}), nil
}

func fail(n node, format string, args ...any) {
	panic(errAt(n.at(), format, args...))
}

func (r *run) eval(n node, sc *scope) Value {
	r.steps++
	if r.steps > maxSteps {
		fail(n, "the formula takes too many steps")
	}
	switch x := n.(type) {
	case *numLit:
		return Num(x.v)
	case *strLit:
		return Text(x.v)
	case *boolLit:
		return Bool(x.v)
	case *listLit:
		out := make([]Value, len(x.items))
		for i, it := range x.items {
			out[i] = r.eval(it, sc)
		}
		return List(out)
	case *ident:
		if v, ok := sc.get(x.name); ok {
			return v
		}
		switch strings.ToLower(x.name) {
		case "current":
			v, _ := sc.get("#current")
			return v
		case "index":
			v, _ := sc.get("#index")
			return v
		case "pi":
			return Num(math.Pi)
		case "e":
			return Num(math.E)
		}
		fail(x, "%q is not defined", x.name)
	case *unaryNode:
		v := r.eval(x.x, sc)
		if x.op == "!" {
			return Bool(!v.Truthy())
		}
		f, ok := v.ToNumber()
		if !ok {
			if v.IsEmpty() {
				return Empty
			}
			fail(x, "cannot negate a %s", v.K)
		}
		return Num(-f)
	case *condNode:
		if r.eval(x.c, sc).Truthy() {
			return r.eval(x.a, sc)
		}
		return r.eval(x.b, sc)
	case *binaryNode:
		return r.binary(x, sc)
	case *callNode:
		return r.call(x, sc)
	}
	fail(n, "cannot evaluate this part")
	return Empty
}

func (r *run) binary(x *binaryNode, sc *scope) Value {
	switch x.op {
	case "&&":
		l := r.eval(x.l, sc)
		if !l.Truthy() {
			return Bool(false)
		}
		return Bool(r.eval(x.r, sc).Truthy())
	case "||":
		l := r.eval(x.l, sc)
		if l.Truthy() {
			return Bool(true)
		}
		return Bool(r.eval(x.r, sc).Truthy())
	}
	l, rv := r.eval(x.l, sc), r.eval(x.r, sc)
	switch x.op {
	case "==":
		return Bool(Equal(l, rv))
	case "!=":
		return Bool(!Equal(l, rv))
	case "<", ">", "<=", ">=":
		c, ok := Compare(l, rv)
		if !ok {
			fail(x, "cannot compare a %s with a %s", l.K, rv.K)
		}
		switch x.op {
		case "<":
			return Bool(c < 0)
		case ">":
			return Bool(c > 0)
		case "<=":
			return Bool(c <= 0)
		}
		return Bool(c >= 0)
	case "+":
		if l.K == KText || rv.K == KText {
			return Text(l.String() + rv.String())
		}
		if l.K == KList && rv.K == KList {
			return List(append(append([]Value{}, l.L...), rv.L...))
		}
	}
	if l.IsEmpty() || rv.IsEmpty() {
		if x.op == "+" || x.op == "-" {
			if l.IsEmpty() && rv.K == KNumber {
				if x.op == "-" {
					return Num(-rv.N)
				}
				return rv
			}
			if rv.IsEmpty() && l.K == KNumber {
				return l
			}
		}
		return Empty
	}
	a, ok1 := l.ToNumber()
	b, ok2 := rv.ToNumber()
	if !ok1 || !ok2 || l.K == KDate || rv.K == KDate {
		fail(x, "%s needs numbers, not a %s and a %s", x.op, l.K, rv.K)
	}
	switch x.op {
	case "+":
		return Num(a + b)
	case "-":
		return Num(a - b)
	case "*":
		return Num(a * b)
	case "/":
		if b == 0 {
			return Empty
		}
		return Num(a / b)
	case "%":
		if b == 0 {
			return Empty
		}
		return Num(math.Mod(a, b))
	case "^":
		return Num(math.Pow(a, b))
	}
	fail(x, "unknown operator %s", x.op)
	return Empty
}

func (r *run) lambda(x *callNode, sc *scope, item Value, i int) Value {
	return r.eval(x.args[1], sc.with(map[string]Value{"#current": item, "#index": Num(float64(i))}))
}

func (r *run) listArg(x *callNode, sc *scope) []Value {
	v := r.eval(x.args[0], sc)
	if v.K == KList {
		return v.L
	}
	if v.IsEmpty() {
		return nil
	}
	return []Value{v}
}

func (r *run) call(x *callNode, sc *scope) Value {
	switch x.name {
	case "if":
		if len(x.args) != 3 {
			fail(x, "if takes a condition, a value when true and a value when false")
		}
		if r.eval(x.args[0], sc).Truthy() {
			return r.eval(x.args[1], sc)
		}
		return r.eval(x.args[2], sc)
	case "ifs":
		if len(x.args) < 2 {
			fail(x, "ifs takes pairs of conditions and values")
		}
		i := 0
		for ; i+1 < len(x.args); i += 2 {
			if r.eval(x.args[i], sc).Truthy() {
				return r.eval(x.args[i+1], sc)
			}
		}
		if i < len(x.args) {
			return r.eval(x.args[i], sc)
		}
		return Empty
	case "and":
		for _, a := range x.args {
			if !r.eval(a, sc).Truthy() {
				return Bool(false)
			}
		}
		return Bool(true)
	case "or":
		for _, a := range x.args {
			if r.eval(a, sc).Truthy() {
				return Bool(true)
			}
		}
		return Bool(false)
	case "let":
		name := x.args[0].(*ident).name
		v := r.eval(x.args[1], sc)
		return r.eval(x.args[2], sc.with(map[string]Value{name: v}))
	case "lets":
		inner := sc
		for i := 0; i+1 < len(x.args); i += 2 {
			name := x.args[i].(*ident).name
			inner = inner.with(map[string]Value{name: r.eval(x.args[i+1], inner)})
		}
		return r.eval(x.args[len(x.args)-1], inner)
	case "prop":
		if x.method {
			target := r.eval(x.args[0], sc)
			name := x.args[1].(*strLit).v
			if target.K != KPage {
				if target.IsEmpty() {
					return Empty
				}
				fail(x, "prop after a dot needs a page, not a %s", target.K)
			}
			v, err := r.env.PageProp(target.S, name)
			if err != nil {
				fail(x, "%s", err.Error())
			}
			return v
		}
		v, err := r.env.Prop(x.args[0].(*strLit).v)
		if err != nil {
			fail(x, "%s", err.Error())
		}
		return v
	case "map", "filter", "find", "findindex", "some", "every", "count":
		items := r.listArg(x, sc)
		if len(x.args) < 2 {
			if x.name == "count" {
				return Num(float64(len(items)))
			}
			fail(x, "%s takes a list and an expression using current", x.name)
		}
		switch x.name {
		case "map":
			out := make([]Value, len(items))
			for i, it := range items {
				out[i] = r.lambda(x, sc, it, i)
			}
			return List(out)
		case "filter":
			var out []Value
			for i, it := range items {
				if r.lambda(x, sc, it, i).Truthy() {
					out = append(out, it)
				}
			}
			return List(out)
		case "count":
			n := 0
			for i, it := range items {
				if r.lambda(x, sc, it, i).Truthy() {
					n++
				}
			}
			return Num(float64(n))
		case "find":
			for i, it := range items {
				if r.lambda(x, sc, it, i).Truthy() {
					return it
				}
			}
			return Empty
		case "findindex":
			for i, it := range items {
				if r.lambda(x, sc, it, i).Truthy() {
					return Num(float64(i))
				}
			}
			return Num(-1)
		case "some":
			for i, it := range items {
				if r.lambda(x, sc, it, i).Truthy() {
					return Bool(true)
				}
			}
			return Bool(false)
		case "every":
			for i, it := range items {
				if !r.lambda(x, sc, it, i).Truthy() {
					return Bool(false)
				}
			}
			return Bool(true)
		}
	case "sort":
		items := append([]Value{}, r.listArg(x, sc)...)
		keys := items
		if len(x.args) >= 2 {
			keys = make([]Value, len(items))
			for i, it := range items {
				keys[i] = r.lambda(x, sc, it, i)
			}
		}
		idx := make([]int, len(items))
		for i := range idx {
			idx[i] = i
		}
		stableSort(idx, func(a, b int) bool {
			c, _ := Compare(keys[a], keys[b])
			return c < 0
		})
		out := make([]Value, len(items))
		for i, j := range idx {
			out[i] = items[j]
		}
		return List(out)
	}
	spec, ok := builtins[x.name]
	if !ok {
		fail(x, "%s is not a function", x.name)
	}
	args := make([]Value, len(x.args))
	for i, a := range x.args {
		args[i] = r.eval(a, sc)
	}
	v, err := spec.fn(r.env, args)
	if err != nil {
		fail(x, "%s: %s", x.name, err.Error())
	}
	return v
}

func stableSort(idx []int, less func(a, b int) bool) {
	sort.SliceStable(idx, func(i, j int) bool { return less(idx[i], idx[j]) })
}
