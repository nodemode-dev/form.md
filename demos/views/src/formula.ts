export type V =
  | { k: "e" }
  | { k: "n"; n: number }
  | { k: "s"; s: string }
  | { k: "b"; b: boolean }
  | { k: "d"; t: number; end: number; time: boolean }
  | { k: "l"; l: V[] }
  | { k: "p"; id: string; name: string };

export const EMPTY: V = { k: "e" };
export const num = (n: number): V => (Number.isFinite(n) ? { k: "n", n } : EMPTY);
export const str = (s: string): V => ({ k: "s", s });
export const bool = (b: boolean): V => ({ k: "b", b });
export const list = (l: V[]): V => ({ k: "l", l });
export const date = (t: number, time = false, end = 0): V => ({ k: "d", t, end, time });

export class FormulaError extends Error {
  constructor(
    message: string,
    readonly pos: number,
  ) {
    super(message);
  }
}

export interface Env {
  prop(name: string): V;
  now(): number;
}

export function fmtNum(n: number): string {
  if (Number.isInteger(n) && Math.abs(n) < 1e15) return String(n);
  const s = String(n);
  return s.length > 18 ? n.toPrecision(12) : s;
}

const MONTHS = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];

export function fmtDate(t: number, time: boolean): string {
  const d = new Date(t);
  const base = `${MONTHS[d.getUTCMonth()]} ${d.getUTCDate()}, ${d.getUTCFullYear()}`;
  if (!time) return base;
  const h = d.getUTCHours();
  return `${base} ${h % 12 === 0 ? 12 : h % 12}:${String(d.getUTCMinutes()).padStart(2, "0")} ${h < 12 ? "AM" : "PM"}`;
}

export function asText(v: V): string {
  switch (v.k) {
    case "n":
      return fmtNum(v.n);
    case "s":
      return v.s;
    case "b":
      return v.b ? "true" : "false";
    case "d":
      return v.end && v.end !== v.t ? `${fmtDate(v.t, v.time)} to ${fmtDate(v.end, v.time)}` : fmtDate(v.t, v.time);
    case "l":
      return v.l.map(asText).join(", ");
    case "p":
      return v.name;
  }
  return "";
}

export function truthy(v: V): boolean {
  switch (v.k) {
    case "n":
      return v.n !== 0;
    case "s":
      return v.s !== "";
    case "b":
      return v.b;
    case "l":
      return v.l.length > 0;
    case "d":
    case "p":
      return true;
  }
  return false;
}

export function isEmpty(v: V): boolean {
  return v.k === "e" || (v.k === "s" && v.s === "") || (v.k === "l" && v.l.length === 0);
}

export function flatten(vs: V[]): V[] {
  const out: V[] = [];
  for (const v of vs) {
    if (v.k === "l") out.push(...flatten(v.l));
    else out.push(v);
  }
  return out;
}

export function same(a: V, b: V): boolean {
  if (a.k !== b.k) return isEmpty(a) && isEmpty(b);
  switch (a.k) {
    case "n":
      return a.n === (b as typeof a).n;
    case "s":
      return a.s.toLowerCase() === (b as typeof a).s.toLowerCase();
    case "b":
      return a.b === (b as typeof a).b;
    case "d":
      return a.t === (b as typeof a).t;
    case "p":
      return a.id === (b as typeof a).id;
    case "l":
      return a.l.length === (b as typeof a).l.length && a.l.every((x, i) => same(x, (b as typeof a).l[i] as V));
  }
  return true;
}

export function compare(a: V, b: V): number | null {
  if (a.k === "n" && b.k === "n") return a.n - b.n;
  if (a.k === "d" && b.k === "d") return a.t - b.t;
  if (a.k === "s" && b.k === "s") return a.s.toLowerCase() < b.s.toLowerCase() ? -1 : a.s.toLowerCase() > b.s.toLowerCase() ? 1 : 0;
  if (a.k === "b" && b.k === "b") return Number(a.b) - Number(b.b);
  return null;
}

type Tok = { k: "num" | "str" | "id" | "op" | "eof"; s: string; n: number; pos: number };

function lex(src: string): Tok[] {
  const out: Tok[] = [];
  let i = 0;
  while (i < src.length) {
    const c = src[i] as string;
    if (/\s/.test(c)) {
      i++;
      continue;
    }
    if (/[0-9]/.test(c) || (c === "." && /[0-9]/.test(src[i + 1] ?? ""))) {
      let j = i;
      while (j < src.length && /[0-9._]/.test(src[j] as string)) j++;
      const text = src.slice(i, j).replace(/_/g, "");
      const n = Number(text);
      if (Number.isNaN(n)) throw new FormulaError(`"${text}" is not a number`, i);
      out.push({ k: "num", s: text, n, pos: i });
      i = j;
      continue;
    }
    if (c === '"' || c === "'" || c === "“" || c === "‘") {
      const closer = c === "“" ? "”" : c === "‘" ? "’" : c;
      let j = i + 1;
      let s = "";
      let closed = false;
      while (j < src.length) {
        const d = src[j] as string;
        if (d === "\\" && j + 1 < src.length) {
          const e = src[j + 1] as string;
          s += e === "n" ? "\n" : e === "t" ? "\t" : e;
          j += 2;
          continue;
        }
        if (d === closer || d === c) {
          closed = true;
          j++;
          break;
        }
        s += d;
        j++;
      }
      if (!closed) throw new FormulaError("a text value is missing its closing quote", i);
      out.push({ k: "str", s, n: 0, pos: i });
      i = j;
      continue;
    }
    if (/[\p{L}_]/u.test(c)) {
      let j = i;
      while (j < src.length && /[\p{L}\p{N}_]/u.test(src[j] as string)) j++;
      out.push({ k: "id", s: src.slice(i, j), n: 0, pos: i });
      i = j;
      continue;
    }
    const two = src.slice(i, i + 2);
    if (["==", "!=", ">=", "<=", "&&", "||"].includes(two)) {
      out.push({ k: "op", s: two, n: 0, pos: i });
      i += 2;
      continue;
    }
    if ("+-*/%^><!()[],.?:=".includes(c)) {
      out.push({ k: "op", s: c, n: 0, pos: i });
      i++;
      continue;
    }
    throw new FormulaError(`unexpected character "${c}"`, i);
  }
  out.push({ k: "eof", s: "", n: 0, pos: src.length });
  return out;
}

type Node =
  | { t: "num"; v: number; pos: number }
  | { t: "str"; v: string; pos: number }
  | { t: "bool"; v: boolean; pos: number }
  | { t: "list"; items: Node[]; pos: number }
  | { t: "id"; name: string; pos: number }
  | { t: "call"; name: string; args: Node[]; pos: number; method: boolean }
  | { t: "un"; op: string; x: Node; pos: number }
  | { t: "bin"; op: string; l: Node; r: Node; pos: number }
  | { t: "cond"; c: Node; a: Node; b: Node; pos: number };

class Parser {
  i = 0;
  constructor(private readonly toks: Tok[]) {}
  peek(): Tok {
    return this.toks[this.i] as Tok;
  }
  next(): Tok {
    const t = this.peek();
    if (this.i < this.toks.length - 1) this.i++;
    return t;
  }
  isOp(s: string): boolean {
    const t = this.peek();
    return t.k === "op" && t.s === s;
  }
  expect(s: string): void {
    if (!this.isOp(s)) {
      const t = this.peek();
      throw new FormulaError(t.k === "eof" ? `expected "${s}" before the end` : `expected "${s}"`, t.pos);
    }
    this.next();
  }
  power(): [string, number] | null {
    const t = this.peek();
    if (t.k === "id") {
      const w = t.s.toLowerCase();
      if (w === "or") return ["||", 2];
      if (w === "and") return ["&&", 3];
      return null;
    }
    if (t.k !== "op") return null;
    switch (t.s) {
      case "?":
        return ["?", 1];
      case "||":
        return ["||", 2];
      case "&&":
        return ["&&", 3];
      case "==":
      case "!=":
        return [t.s, 4];
      case "=":
        return ["==", 4];
      case "<":
      case ">":
      case "<=":
      case ">=":
        return [t.s, 5];
      case "+":
      case "-":
        return [t.s, 6];
      case "*":
      case "/":
      case "%":
        return [t.s, 7];
      case "^":
        return ["^", 8];
    }
    return null;
  }
  expr(min: number): Node {
    let left = this.unary();
    for (;;) {
      const p = this.power();
      if (!p || p[1] < min) return left;
      const t = this.next();
      if (p[0] === "?") {
        const a = this.expr(1);
        this.expect(":");
        const b = this.expr(1);
        left = { t: "cond", c: left, a, b, pos: t.pos };
        continue;
      }
      const right = this.expr(p[0] === "^" ? p[1] : p[1] + 1);
      left = { t: "bin", op: p[0], l: left, r: right, pos: t.pos };
    }
  }
  unary(): Node {
    const t = this.peek();
    if (t.k === "op" && (t.s === "-" || t.s === "!" || t.s === "+")) {
      this.next();
      const x = this.expr(9);
      return t.s === "+" ? x : { t: "un", op: t.s, x, pos: t.pos };
    }
    if (t.k === "id" && t.s.toLowerCase() === "not" && !(this.toks[this.i + 1]?.k === "op" && this.toks[this.i + 1]?.s === "(")) {
      this.next();
      return { t: "un", op: "!", x: this.expr(9), pos: t.pos };
    }
    return this.postfix();
  }
  args(): Node[] {
    this.expect("(");
    const out: Node[] = [];
    if (this.isOp(")")) {
      this.next();
      return out;
    }
    for (;;) {
      out.push(this.expr(1));
      if (this.isOp(",")) {
        this.next();
        continue;
      }
      this.expect(")");
      return out;
    }
  }
  postfix(): Node {
    let n = this.primary();
    while (this.isOp(".")) {
      const dot = this.next();
      const name = this.next();
      if (name.k !== "id") throw new FormulaError("expected a function name after the dot", dot.pos);
      const args = this.isOp("(") ? this.args() : [];
      n = { t: "call", name: name.s.toLowerCase(), args: [n, ...args], pos: name.pos, method: true };
    }
    return n;
  }
  primary(): Node {
    const t = this.next();
    if (t.k === "num") return { t: "num", v: t.n, pos: t.pos };
    if (t.k === "str") return { t: "str", v: t.s, pos: t.pos };
    if (t.k === "id") {
      const low = t.s.toLowerCase();
      if (low === "true") return { t: "bool", v: true, pos: t.pos };
      if (low === "false") return { t: "bool", v: false, pos: t.pos };
      if (this.isOp("(")) return { t: "call", name: low, args: this.args(), pos: t.pos, method: false };
      return { t: "id", name: t.s, pos: t.pos };
    }
    if (t.k === "op") {
      if (t.s === "(") {
        const n = this.expr(0);
        this.expect(")");
        return n;
      }
      if (t.s === "[") {
        const items: Node[] = [];
        if (this.isOp("]")) {
          this.next();
          return { t: "list", items, pos: t.pos };
        }
        for (;;) {
          items.push(this.expr(1));
          if (this.isOp(",")) {
            this.next();
            continue;
          }
          this.expect("]");
          return { t: "list", items, pos: t.pos };
        }
      }
      throw new FormulaError(`unexpected "${t.s}"`, t.pos);
    }
    throw new FormulaError("the formula ends too early", t.pos);
  }
}

export interface Program {
  node: Node;
  props: string[];
}

function collect(n: Node, out: Set<string>): void {
  switch (n.t) {
    case "list":
      for (const x of n.items) collect(x, out);
      break;
    case "call":
      if (n.name === "prop" && !n.method && n.args[0]?.t === "str") out.add(n.args[0].v);
      for (const x of n.args) collect(x, out);
      break;
    case "un":
      collect(n.x, out);
      break;
    case "bin":
      collect(n.l, out);
      collect(n.r, out);
      break;
    case "cond":
      collect(n.c, out);
      collect(n.a, out);
      collect(n.b, out);
      break;
  }
}

export function compile(src: string): Program {
  const p = new Parser(lex(src));
  if (p.peek().k === "eof") throw new FormulaError("the formula is empty", 0);
  const node = p.expr(0);
  const t = p.peek();
  if (t.k !== "eof") throw new FormulaError(`unexpected "${t.s}"`, t.pos);
  const props = new Set<string>();
  collect(node, props);
  return { node, props: [...props] };
}

const DAY = 86400000;

function toDate(v: V, pos: number): { t: number; end: number; time: boolean } {
  if (v.k === "d") return v;
  if (v.k === "s") {
    const t = Date.parse(v.s.length <= 10 ? `${v.s}T00:00:00Z` : v.s);
    if (!Number.isNaN(t)) return { t, end: 0, time: v.s.length > 10 };
  }
  throw new FormulaError(v.k === "e" ? "needs a date, but it is empty" : "needs a date", pos);
}

function shift(t: number, n: number, unit: string, pos: number): number {
  const d = new Date(t);
  switch (unit.replace(/s$/, "")) {
    case "year":
      d.setUTCFullYear(d.getUTCFullYear() + n);
      return d.getTime();
    case "month":
      d.setUTCMonth(d.getUTCMonth() + n);
      return d.getTime();
    case "week":
      return t + n * 7 * DAY;
    case "day":
      return t + n * DAY;
    case "hour":
      return t + n * 3600000;
    case "minute":
      return t + n * 60000;
    case "second":
      return t + n * 1000;
  }
  throw new FormulaError(`"${unit}" is not a unit; use years, months, weeks, days, hours, minutes or seconds`, pos);
}

function between(a: number, b: number, unit: string, pos: number): number {
  const u = unit.replace(/s$/, "");
  const ms = a - b;
  switch (u) {
    case "day":
      return Math.trunc(ms / DAY);
    case "week":
      return Math.trunc(ms / (7 * DAY));
    case "hour":
      return Math.trunc(ms / 3600000);
    case "minute":
      return Math.trunc(ms / 60000);
    case "second":
      return Math.trunc(ms / 1000);
    case "month":
    case "year": {
      const da = new Date(a);
      const db = new Date(b);
      let m = (da.getUTCFullYear() - db.getUTCFullYear()) * 12 + da.getUTCMonth() - db.getUTCMonth();
      const probe = shift(b, m, "month", pos);
      if (m > 0 && probe > a) m--;
      if (m < 0 && probe < a) m++;
      return u === "year" ? Math.trunc(m / 12) : m;
    }
  }
  throw new FormulaError(`"${unit}" is not a unit; use years, months, weeks, days, hours, minutes or seconds`, pos);
}

function nums(vs: V[], pos: number, who: string): number[] {
  const out: number[] = [];
  for (const v of flatten(vs)) {
    if (v.k === "n") out.push(v.n);
    else if (v.k === "e") continue;
    else throw new FormulaError(`${who} needs numbers`, pos);
  }
  return out;
}

function numArg(v: V, pos: number, who: string): number {
  if (v.k === "n") return v.n;
  if (v.k === "b") return Number(v.b);
  if (v.k === "s" && v.s.trim() !== "" && !Number.isNaN(Number(v.s))) return Number(v.s);
  if (v.k === "e") return 0;
  throw new FormulaError(`${who} needs a number`, pos);
}

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

function formatWith(t: number, pattern: string): string {
  const d = new Date(t);
  const h = d.getUTCHours();
  return pattern
    .replace(/YYYY/g, String(d.getUTCFullYear()))
    .replace(/MMMM/g, MONTHS[d.getUTCMonth()] as string)
    .replace(/MMM/g, (MONTHS[d.getUTCMonth()] as string).slice(0, 3))
    .replace(/MM/g, pad2(d.getUTCMonth() + 1))
    .replace(/DD/g, pad2(d.getUTCDate()))
    .replace(/HH/g, pad2(h))
    .replace(/mm/g, pad2(d.getUTCMinutes()));
}

interface Scope {
  vars: Map<string, V>;
}

export function run(prog: Program, env: Env): V {
  const ev = (n: Node, sc: Scope): V => {
    switch (n.t) {
      case "num":
        return num(n.v);
      case "str":
        return str(n.v);
      case "bool":
        return bool(n.v);
      case "list":
        return list(n.items.map((x) => ev(x, sc)));
      case "id": {
        const hit = sc.vars.get(n.name) ?? sc.vars.get(n.name.toLowerCase());
        if (hit) return hit;
        const low = n.name.toLowerCase();
        if (low === "pi") return num(Math.PI);
        if (low === "e") return num(Math.E);
        throw new FormulaError(`"${n.name}" is not a property, variable or function; properties are written prop("Name")`, n.pos);
      }
      case "un": {
        const x = ev(n.x, sc);
        if (n.op === "!") return bool(!truthy(x));
        return num(-numArg(x, n.pos, "minus"));
      }
      case "cond":
        return truthy(ev(n.c, sc)) ? ev(n.a, sc) : ev(n.b, sc);
      case "bin":
        return binary(n, sc);
      case "call":
        return call(n, sc);
    }
  };

  const binary = (n: Extract<Node, { t: "bin" }>, sc: Scope): V => {
    if (n.op === "&&") {
      const l = ev(n.l, sc);
      return truthy(l) ? ev(n.r, sc) : l;
    }
    if (n.op === "||") {
      const l = ev(n.l, sc);
      return truthy(l) ? l : ev(n.r, sc);
    }
    const l = ev(n.l, sc);
    const r = ev(n.r, sc);
    switch (n.op) {
      case "==":
        return bool(same(l, r));
      case "!=":
        return bool(!same(l, r));
      case "<":
      case ">":
      case "<=":
      case ">=": {
        const c = compare(l, r);
        if (c === null) return bool(false);
        return bool(n.op === "<" ? c < 0 : n.op === ">" ? c > 0 : n.op === "<=" ? c <= 0 : c >= 0);
      }
      case "+":
        if (l.k === "d" && (r.k === "n" || r.k === "e")) return date(l.t + (r.k === "n" ? r.n : 0) * DAY, l.time);
        if (l.k === "s" || r.k === "s") return str(asText(l) + asText(r));
        return num(numArg(l, n.pos, "+") + numArg(r, n.pos, "+"));
      case "-":
        if (l.k === "d" && r.k === "d") return num((l.t - r.t) / DAY);
        if (l.k === "d" && r.k === "n") return date(l.t - r.n * DAY, l.time);
        return num(numArg(l, n.pos, "-") - numArg(r, n.pos, "-"));
      case "*":
        return num(numArg(l, n.pos, "*") * numArg(r, n.pos, "*"));
      case "/": {
        const d = numArg(r, n.pos, "/");
        if (d === 0) throw new FormulaError("cannot divide by zero", n.pos);
        return num(numArg(l, n.pos, "/") / d);
      }
      case "%": {
        const d = numArg(r, n.pos, "%");
        if (d === 0) throw new FormulaError("cannot divide by zero", n.pos);
        return num(numArg(l, n.pos, "%") % d);
      }
      case "^":
        return num(numArg(l, n.pos, "^") ** numArg(r, n.pos, "^"));
    }
    return EMPTY;
  };

  const lambda = (n: Extract<Node, { t: "call" }>, sc: Scope, items: V[], f: (v: V, i: number, r: () => V) => boolean | void): void => {
    const body = n.args[1];
    if (!body) throw new FormulaError(`${n.name} needs a function`, n.pos);
    items.forEach((v, i) => {
      const vars = new Map(sc.vars);
      vars.set("current", v);
      vars.set("index", num(i));
      f(v, i, () => ev(body, { vars }));
    });
  };

  const call = (n: Extract<Node, { t: "call" }>, sc: Scope): V => {
    const a = n.args;
    const name = n.name;
    const need = (lo: number, hi = lo) => {
      if (a.length < lo || (hi >= 0 && a.length > hi)) throw new FormulaError(`${name} takes ${lo === hi ? lo : `${lo} to ${hi < 0 ? "many" : hi}`} argument${hi === 1 ? "" : "s"}`, n.pos);
    };
    switch (name) {
      case "prop": {
        if (n.method) {
          need(2);
          const lit = a[1];
          if (lit?.t !== "str") throw new FormulaError("prop needs a property name in quotes", n.pos);
          const base = ev(a[0] as Node, sc);
          if (base.k === "p") return EMPTY;
          return EMPTY;
        }
        need(1);
        const lit = a[0];
        if (lit?.t !== "str") throw new FormulaError("prop needs a property name in quotes", n.pos);
        return env.prop(lit.v);
      }
      case "if": {
        need(2, 3);
        return truthy(ev(a[0] as Node, sc)) ? ev(a[1] as Node, sc) : a[2] ? ev(a[2], sc) : EMPTY;
      }
      case "ifs": {
        for (let i = 0; i + 1 < a.length; i += 2) if (truthy(ev(a[i] as Node, sc))) return ev(a[i + 1] as Node, sc);
        return a.length % 2 === 1 ? ev(a[a.length - 1] as Node, sc) : EMPTY;
      }
      case "and":
        return bool(a.every((x) => truthy(ev(x, sc))));
      case "or":
        return bool(a.some((x) => truthy(ev(x, sc))));
      case "let": {
        need(3);
        const id = a[0];
        if (id?.t !== "id") throw new FormulaError("let needs a variable name first", n.pos);
        const vars = new Map(sc.vars);
        vars.set(id.name, ev(a[1] as Node, sc));
        return ev(a[2] as Node, { vars });
      }
      case "lets": {
        const vars = new Map(sc.vars);
        for (let i = 0; i + 1 < a.length - 1; i += 2) {
          const id = a[i];
          if (id?.t !== "id") throw new FormulaError("lets needs a variable name here", n.pos);
          vars.set(id.name, ev(a[i + 1] as Node, { vars }));
        }
        return ev(a[a.length - 1] as Node, { vars });
      }
      case "map":
      case "filter":
      case "find":
      case "findindex":
      case "some":
      case "every":
      case "count": {
        const src = ev(a[0] as Node, sc);
        const items = src.k === "l" ? src.l : src.k === "e" ? [] : [src];
        if (name === "count" && !a[1]) return num(items.length);
        const out: V[] = [];
        let found: V = EMPTY;
        let idx = -1;
        let some = false;
        let every = true;
        let cnt = 0;
        lambda(n, sc, items, (v, i, r) => {
          const res = r();
          if (name === "map") out.push(res);
          else {
            const t = truthy(res);
            if (t) {
              cnt++;
              some = true;
              if (name === "filter") out.push(v);
              if (idx < 0) {
                idx = i;
                found = v;
              }
            } else every = false;
          }
        });
        if (name === "map" || name === "filter") return list(out);
        if (name === "find") return found;
        if (name === "findindex") return num(idx);
        if (name === "some") return bool(some);
        if (name === "every") return bool(every);
        return num(cnt);
      }
    }
    const v = a.map((x) => ev(x, sc));
    const arg = (i: number): V => v[i] ?? EMPTY;
    switch (name) {
      case "not":
        need(1);
        return bool(!truthy(arg(0)));
      case "empty":
        need(1);
        return bool(isEmpty(arg(0)) || (arg(0).k === "n" && (arg(0) as { n: number }).n === 0) || (arg(0).k === "b" && !(arg(0) as { b: boolean }).b));
      case "equal":
        need(2);
        return bool(same(arg(0), arg(1)));
      case "unequal":
        need(2);
        return bool(!same(arg(0), arg(1)));
      case "concat": {
        need(1, -1);
        if (v.every((x) => x.k === "l" || x.k === "e")) return list(v.flatMap((x) => (x.k === "l" ? x.l : [])));
        return str(v.map(asText).join(""));
      }
      case "join": {
        need(1, -1);
        if (arg(0).k === "l") return str((arg(0) as { l: V[] }).l.map(asText).join(v[1] ? asText(arg(1)) : ""));
        return str(v.slice(1).map(asText).join(asText(arg(0))));
      }
      case "slice":
      case "substring": {
        need(2, 3);
        const s = arg(0);
        const from = numArg(arg(1), n.pos, name);
        const to = v[2] ? numArg(arg(2), n.pos, name) : undefined;
        if (s.k === "l") return list(s.l.slice(from, to));
        return str(Array.from(asText(s)).slice(from, to).join(""));
      }
      case "length":
        need(1);
        return num(arg(0).k === "l" ? (arg(0) as { l: V[] }).l.length : Array.from(asText(arg(0))).length);
      case "contains":
        need(2);
        return bool(asText(arg(0)).toLowerCase().includes(asText(arg(1)).toLowerCase()));
      case "includes":
        need(2);
        return bool(arg(0).k === "l" ? (arg(0) as { l: V[] }).l.some((x) => same(x, arg(1))) : false);
      case "replace":
        need(3);
        return str(asText(arg(0)).replace(asText(arg(1)), asText(arg(2))));
      case "replaceall":
        need(3);
        return str(asText(arg(0)).split(asText(arg(1))).join(asText(arg(2))));
      case "lower":
        need(1);
        return str(asText(arg(0)).toLowerCase());
      case "upper":
        need(1);
        return str(asText(arg(0)).toUpperCase());
      case "trim":
        need(1);
        return str(asText(arg(0)).trim());
      case "repeat":
        need(2);
        return str(asText(arg(0)).repeat(Math.max(0, Math.min(200, numArg(arg(1), n.pos, name)))));
      case "padstart":
        need(2, 3);
        return str(asText(arg(0)).padStart(numArg(arg(1), n.pos, name), v[2] ? asText(arg(2)) : " "));
      case "padend":
        need(2, 3);
        return str(asText(arg(0)).padEnd(numArg(arg(1), n.pos, name), v[2] ? asText(arg(2)) : " "));
      case "format":
        need(1);
        return str(asText(arg(0)));
      case "split":
        need(1, 2);
        return list(asText(arg(0)).split(v[1] ? asText(arg(1)) : ",").map((x) => str(x.trim())));
      case "add":
      case "subtract":
      case "multiply":
      case "divide":
      case "pow":
      case "mod": {
        need(2);
        const x = numArg(arg(0), n.pos, name);
        const y = numArg(arg(1), n.pos, name);
        if ((name === "divide" || name === "mod") && y === 0) throw new FormulaError("cannot divide by zero", n.pos);
        return num(name === "add" ? x + y : name === "subtract" ? x - y : name === "multiply" ? x * y : name === "divide" ? x / y : name === "pow" ? x ** y : x % y);
      }
      case "abs":
        need(1);
        return num(Math.abs(numArg(arg(0), n.pos, name)));
      case "ceil":
        need(1);
        return num(Math.ceil(numArg(arg(0), n.pos, name)));
      case "floor":
        need(1);
        return num(Math.floor(numArg(arg(0), n.pos, name)));
      case "round": {
        need(1, 2);
        const x = numArg(arg(0), n.pos, name);
        const p = v[1] ? numArg(arg(1), n.pos, name) : 0;
        const f = 10 ** p;
        return num(Math.round((x + Number.EPSILON * Math.sign(x)) * f) / f);
      }
      case "sqrt":
        need(1);
        return num(Math.sqrt(numArg(arg(0), n.pos, name)));
      case "cbrt":
        need(1);
        return num(Math.cbrt(numArg(arg(0), n.pos, name)));
      case "exp":
        need(1);
        return num(Math.exp(numArg(arg(0), n.pos, name)));
      case "ln":
        need(1);
        return num(Math.log(numArg(arg(0), n.pos, name)));
      case "sign":
        need(1);
        return num(Math.sign(numArg(arg(0), n.pos, name)));
      case "min":
      case "max":
      case "sum":
      case "mean":
      case "average":
      case "median": {
        need(1, -1);
        const xs = nums(v, n.pos, name);
        if (!xs.length) return name === "sum" ? num(0) : EMPTY;
        if (name === "min") return num(Math.min(...xs));
        if (name === "max") return num(Math.max(...xs));
        if (name === "sum") return num(xs.reduce((s, x) => s + x, 0));
        if (name === "median") {
          const s = [...xs].sort((p, q) => p - q);
          const m = Math.floor(s.length / 2);
          return num(s.length % 2 ? (s[m] as number) : ((s[m - 1] as number) + (s[m] as number)) / 2);
        }
        return num(xs.reduce((s, x) => s + x, 0) / xs.length);
      }
      case "tonumber": {
        need(1);
        const x = arg(0);
        if (x.k === "n") return x;
        if (x.k === "b") return num(Number(x.b));
        const t = asText(x).replace(/[,$%\s]/g, "");
        return t === "" || Number.isNaN(Number(t)) ? EMPTY : num(Number(t));
      }
      case "now":
        return date(env.now(), true);
      case "today":
        return date(Math.floor(env.now() / DAY) * DAY, false);
      case "year":
      case "month":
      case "day":
      case "hour":
      case "minute": {
        need(1);
        const d = new Date(toDate(arg(0), n.pos).t);
        return num(name === "year" ? d.getUTCFullYear() : name === "month" ? d.getUTCMonth() + 1 : name === "day" ? d.getUTCDate() : name === "hour" ? d.getUTCHours() : d.getUTCMinutes());
      }
      case "date": {
        need(1);
        const d = toDate(arg(0), n.pos);
        return num(new Date(d.t).getUTCDate());
      }
      case "datestart": {
        need(1);
        const d = toDate(arg(0), n.pos);
        return date(d.t, d.time);
      }
      case "dateend": {
        need(1);
        const d = toDate(arg(0), n.pos);
        return date(d.end || d.t, d.time);
      }
      case "timestamp":
        need(1);
        return num(toDate(arg(0), n.pos).t);
      case "fromtimestamp":
        need(1);
        return date(numArg(arg(0), n.pos, name), true);
      case "dateadd":
      case "datesubtract": {
        need(3);
        const d = toDate(arg(0), n.pos);
        const k = numArg(arg(1), n.pos, name) * (name === "dateadd" ? 1 : -1);
        return date(shift(d.t, k, asText(arg(2)), n.pos), d.time);
      }
      case "datebetween": {
        need(3);
        return num(between(toDate(arg(0), n.pos).t, toDate(arg(1), n.pos).t, asText(arg(2)), n.pos));
      }
      case "daterange": {
        need(2);
        return date(toDate(arg(0), n.pos).t, false, toDate(arg(1), n.pos).t);
      }
      case "formatdate": {
        need(1, 3);
        const d = toDate(arg(0), n.pos);
        return str(v[1] ? formatWith(d.t, asText(arg(1))) : fmtDate(d.t, d.time));
      }
      case "parsedate": {
        need(1);
        const d = toDate(arg(0), n.pos);
        return date(d.t, d.time);
      }
      case "at":
        need(2);
        return arg(0).k === "l" ? ((arg(0) as { l: V[] }).l[numArg(arg(1), n.pos, name)] ?? EMPTY) : EMPTY;
      case "first":
        need(1);
        return arg(0).k === "l" ? ((arg(0) as { l: V[] }).l[0] ?? EMPTY) : arg(0);
      case "last":
        need(1);
        return arg(0).k === "l" ? ((arg(0) as { l: V[] }).l[(arg(0) as { l: V[] }).l.length - 1] ?? EMPTY) : arg(0);
      case "reverse":
        need(1);
        return arg(0).k === "l" ? list([...(arg(0) as { l: V[] }).l].reverse()) : arg(0);
      case "unique": {
        need(1);
        const out: V[] = [];
        for (const x of flatten([arg(0)])) if (!out.some((o) => same(o, x))) out.push(x);
        return list(out);
      }
      case "flat":
        need(1);
        return list(flatten([arg(0)]));
      case "name":
        need(1);
        return str(asText(arg(0)));
    }
    throw new FormulaError(`there is no function called ${name}`, n.pos);
  };

  return ev(prog.node, { vars: new Map() });
}
