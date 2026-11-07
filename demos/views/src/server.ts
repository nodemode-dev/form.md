import type { CollectionStore } from "../../../web/src/engine/collections";
import type {
  ChartResult,
  ChartSpec,
  Collection,
  CollectionView,
  ComputedValue,
  Filter,
  GroupResult,
  Grouping,
  LaneResult,
  Property,
  QueryInput,
  QueryResult,
  RowData,
  RowRef,
  SchemaOp,
  SortKey,
  ViewConfig,
  ViewType,
} from "../../../web/src/engine/types";
import * as F from "./formula";
import type { V } from "./formula";
import type { SeedRow } from "./data";

export const WORKSPACE = "ws_demo";
export const USER = "u_demo";
const DAY = 86400000;

interface Row {
  id: string;
  title: string;
  icon: string;
  cover: string;
  vals: Record<string, unknown>;
  number: number;
  version: number;
  created: number;
  updated: number;
}

interface Col {
  id: string;
  title: string;
  icon: string;
  schema: Property[];
  views: CollectionView[];
  rows: Row[];
  epoch: number;
  next: number;
}

type DateVal = { start: string; end?: string };

function parseDay(s: string): { t: number; time: boolean } | null {
  if (!s) return null;
  const m = /^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2}))?/.exec(s);
  if (!m) return null;
  const t = Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]), Number(m[4] ?? 0), Number(m[5] ?? 0));
  return { t, time: m[4] !== undefined };
}

function dayStr(t: number, time: boolean): string {
  const d = new Date(t);
  const base = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
  return time ? `${base}T${String(d.getUTCHours()).padStart(2, "0")}:${String(d.getUTCMinutes()).padStart(2, "0")}` : base;
}

const dayStart = (t: number) => Math.floor(t / DAY) * DAY;

function asList(v: unknown): string[] {
  if (Array.isArray(v)) return v.filter((x): x is string => typeof x === "string");
  if (typeof v === "string" && v) return [v];
  return [];
}

let idSeq = 0;
const newId = (p: string) => `${p}_${Date.now().toString(36)}${(idSeq++).toString(36)}`;

export interface Backend {
  fetch: typeof fetch;
  attach(store: CollectionStore): void;
  collection(id: string): Col | undefined;
  addCollection(c: { id: string; title: string; icon: string; schema: Property[]; views: Omit<CollectionView, "collection_id" | "created_at" | "updated_at" | "sort_key">[]; rows: SeedRow[] }): void;
  reset(): void;
}

export function createBackend(seed: () => Parameters<Backend["addCollection"]>[0][]): Backend {
  const cols = new Map<string, Col>();
  let engine: { collections: CollectionStore } | null = null;
  const now0 = Date.now();

  const addCollection: Backend["addCollection"] = (c) => {
    const col: Col = {
      id: c.id,
      title: c.title,
      icon: c.icon,
      schema: structuredClone(c.schema),
      views: c.views.map((v, i) => ({ ...structuredClone(v), collection_id: c.id, sort_key: String(i).padStart(4, "0"), created_at: now0, updated_at: now0 })),
      rows: [],
      epoch: 1,
      next: 1,
    };
    c.rows.forEach((r) => {
      col.rows.push({ id: r.id, title: r.title, icon: "", cover: r.cover ?? "", vals: structuredClone(r.vals), number: col.next++, version: 1, created: now0, updated: now0 });
    });
    cols.set(c.id, col);
  };

  const reset = () => {
    cols.clear();
    for (const c of seed()) addCollection(c);
  };
  reset();

  const notify = (colId: string, what: "rows" | "schema" | "views", ids: string[] = ["*"]) => {
    if (!engine) return;
    const ws = WORKSPACE;
    const emit = (cid: string, w: typeof what) => engine?.collections.apply({ type: "collection", seq: 0, workspace_id: ws, collection_id: cid, what: w, ids: w === "rows" ? ids : [], epoch: cols.get(cid)?.epoch ?? 0 } as never);
    emit(colId, what);
    if (what === "rows" || what === "schema") for (const c of cols.values()) if (c.id !== colId && c.schema.some((p) => p.type === "rollup" || p.type === "formula" || p.type === "relation")) emit(c.id, "rows");
  };

  class Evaluator {
    memo = new Map<string, V>();
    errs = new Map<string, string>();
    visiting = new Set<string>();
    progs = new Map<string, F.Program | string>();
    now = Date.now();

    col(id: string): Col | undefined {
      return cols.get(id);
    }

    prog(col: Col, p: Property): F.Program | string {
      const key = `${col.id}|${p.id}|${p.expr ?? ""}`;
      let hit = this.progs.get(key);
      if (hit === undefined) {
        try {
          hit = F.compile(p.expr ?? "");
        } catch (e) {
          hit = (e as Error).message;
        }
        this.progs.set(key, hit);
      }
      return hit;
    }

    value(col: Col, row: Row, p: Property): V {
      const raw = row.vals[p.id];
      switch (p.type) {
        case "title":
          return F.str(row.title);
        case "text":
        case "url":
        case "email":
        case "phone":
          return typeof raw === "string" ? F.str(raw) : F.EMPTY;
        case "number":
          return typeof raw === "number" ? F.num(raw) : F.EMPTY;
        case "select":
        case "status": {
          const o = p.options?.find((x) => x.id === raw);
          return o ? F.str(o.name) : F.EMPTY;
        }
        case "multi_select":
          return F.list(asList(raw).flatMap((id) => {
            const o = p.options?.find((x) => x.id === id);
            return o ? [F.str(o.name)] : [];
          }));
        case "date": {
          const d = raw as DateVal | undefined;
          const a = d?.start ? parseDay(d.start) : null;
          if (!a) return F.EMPTY;
          const b = d?.end ? parseDay(d.end) : null;
          return F.date(a.t, a.time, b ? b.t : 0);
        }
        case "checkbox":
          return F.bool(raw === true);
        case "files":
          return F.list(((raw as { name: string }[] | undefined) ?? []).map((f) => F.str(f.name)));
        case "relation": {
          const target = this.col(p.collection ?? "");
          return F.list(asList(raw).flatMap((id) => {
            const r = target?.rows.find((x) => x.id === id);
            return r ? [{ k: "p", id, name: r.title } as V] : [];
          }));
        }
        case "created_time":
          return F.date(row.created, true);
        case "last_edited_time":
          return F.date(row.updated, true);
        case "unique_id":
          return F.num(row.number);
        case "formula":
        case "rollup": {
          const k = `${col.id}|${row.id}|${p.id}`;
          const m = this.memo.get(k);
          if (m) return m;
          if (this.visiting.has(k)) {
            this.errs.set(k, "this refers back to itself");
            return F.EMPTY;
          }
          this.visiting.add(k);
          const v = p.type === "formula" ? this.formula(col, row, p, k) : this.rollup(col, row, p, k);
          this.visiting.delete(k);
          this.memo.set(k, v);
          return v;
        }
      }
      return F.EMPTY;
    }

    formula(col: Col, row: Row, p: Property, k: string): V {
      const prog = this.prog(col, p);
      if (typeof prog === "string") {
        this.errs.set(k, prog);
        return F.EMPTY;
      }
      try {
        return F.run(prog, {
          now: () => this.now,
          prop: (name) => {
            const q = col.schema.find((x) => x.name === name);
            if (!q) throw new F.FormulaError(`that page has no property called "${name}"`, 0);
            return this.value(col, row, q);
          },
        });
      } catch (e) {
        this.errs.set(k, (e as Error).message);
        return F.EMPTY;
      }
    }

    rollup(col: Col, row: Row, p: Property, k: string): V {
      const rel = col.schema.find((x) => x.id === p.relation);
      if (!rel || rel.type !== "relation") {
        this.errs.set(k, "choose a relation to roll up");
        return F.EMPTY;
      }
      const target = this.col(rel.collection ?? "");
      if (!target) return F.EMPTY;
      const tp = target.schema.find((x) => x.id === p.target) ?? target.schema.find((x) => x.type === "title");
      if (!tp) return F.EMPTY;
      const related = asList(row.vals[rel.id]).flatMap((id) => {
        const r = target.rows.find((x) => x.id === id);
        return r ? [r] : [];
      });
      return aggregate(p.fn ?? "", related.map((r) => this.value(target, r, tp)), related.length);
    }

    cell(col: Col, row: Row, p: Property): ComputedValue | null {
      const v = this.value(col, row, p);
      const err = this.errs.get(`${col.id}|${row.id}|${p.id}`);
      if (err) return { type: "error", value: err };
      return encode(v);
    }

    rowOut(col: Col, row: Row, refs: Record<string, RowRef> | null): RowData {
      const values: Record<string, unknown> = {};
      for (const p of col.schema) {
        if (p.type === "title" || p.type === "button") continue;
        if (p.type === "formula" || p.type === "rollup") values[p.id] = this.cell(col, row, p);
        else if (p.type === "created_time") values[p.id] = row.created;
        else if (p.type === "last_edited_time") values[p.id] = row.updated;
        else if (p.type === "created_by" || p.type === "last_edited_by") values[p.id] = USER;
        else if (p.type === "unique_id") values[p.id] = row.number;
        else if (p.type === "relation") {
          const target = this.col(p.collection ?? "");
          const kept = asList(row.vals[p.id]).filter((id) => target?.rows.some((r) => r.id === id));
          for (const id of kept) {
            const t = target?.rows.find((r) => r.id === id);
            if (refs && t && target) refs[id] = { title: t.title, icon: t.icon, collection: target.id };
          }
          if (kept.length) values[p.id] = kept;
        } else if (row.vals[p.id] !== undefined && row.vals[p.id] !== null) values[p.id] = row.vals[p.id];
      }
      const idx = col.rows.indexOf(row);
      const out: RowData = { id: row.id, title: row.title, icon: row.icon, sort_key: String(idx).padStart(6, "0"), values, number: row.number, version: row.version, created_at: row.created, created_by: USER, updated_at: row.updated, updated_by: USER };
      if (row.cover) out.cover = row.cover;
      return out;
    }

    option(p: Property, row: Row): string[] {
      if (p.type === "select" || p.type === "status") return typeof row.vals[p.id] === "string" && row.vals[p.id] ? [row.vals[p.id] as string] : [];
      if (p.type === "multi_select" || p.type === "relation" || p.type === "person") return asList(row.vals[p.id]);
      return [];
    }

    kind(p: Property, v: V): string {
      switch (p.type) {
        case "title":
        case "text":
        case "url":
        case "email":
        case "phone":
        case "place":
          return "text";
        case "number":
        case "unique_id":
          return "number";
        case "select":
        case "status":
          return "option";
        case "multi_select":
        case "person":
        case "relation":
        case "created_by":
        case "last_edited_by":
          return "list";
        case "files":
          return "files";
        case "date":
        case "created_time":
        case "last_edited_time":
          return "date";
        case "checkbox":
          return "checkbox";
      }
      if (v.k === "n") return "number";
      if (v.k === "d") return "date";
      if (v.k === "b") return "checkbox";
      if (v.k === "l") return "valuelist";
      return "text";
    }

    filterDay(fv: { date?: string; relative?: string; s?: string }): number | null {
      if (fv.relative) {
        const t = dayStart(this.now);
        const map: Record<string, number> = { today: 0, tomorrow: 1, yesterday: -1, one_week_ago: -7, one_week_from_now: 7 };
        if (fv.relative in map) return t + (map[fv.relative] as number) * DAY;
        if (fv.relative === "one_month_ago") return shiftMonth(t, -1);
        if (fv.relative === "one_month_from_now") return shiftMonth(t, 1);
        return null;
      }
      const d = parseDay(fv.date || fv.s || "");
      return d ? dayStart(d.t) : null;
    }

    within(name: string): [number, number] | null {
      const t = dayStart(this.now);
      const end = t + DAY;
      const d = new Date(t);
      switch (name) {
        case "past_week":
          return [t - 7 * DAY, end];
        case "next_week":
          return [t, t + 8 * DAY];
        case "past_month":
          return [shiftMonth(t, -1), end];
        case "next_month":
          return [t, shiftMonth(t, 1) + DAY];
        case "past_year":
          return [shiftMonth(t, -12), end];
        case "next_year":
          return [t, shiftMonth(t, 12) + DAY];
        case "this_week": {
          const s = t - d.getUTCDay() * DAY;
          return [s, s + 7 * DAY];
        }
        case "this_month": {
          const s = Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), 1);
          return [s, shiftMonth(s, 1)];
        }
        case "this_year":
          return [Date.UTC(d.getUTCFullYear(), 0, 1), Date.UTC(d.getUTCFullYear() + 1, 0, 1)];
      }
      return null;
    }

    matchOne(col: Col, row: Row, f: Filter): boolean {
      const p = col.schema.find((x) => x.id === f.prop);
      if (!p) return true;
      const v = this.value(col, row, p);
      const raw = f.value;
      const sval = typeof raw === "string" ? raw : typeof raw === "number" ? String(raw) : "";
      const nval = typeof raw === "number" ? raw : typeof raw === "string" && raw.trim() !== "" && !Number.isNaN(Number(raw)) ? Number(raw) : null;
      const cond = f.cond ?? "";
      if (cond === "empty" || cond === "is_empty") return p.type === "checkbox" ? !F.truthy(v) : F.isEmpty(v);
      if (cond === "not_empty" || cond === "is_not_empty") return p.type === "checkbox" ? F.truthy(v) : !F.isEmpty(v);
      switch (this.kind(p, v)) {
        case "text": {
          const hay = F.asText(v).toLowerCase();
          const needle = sval.toLowerCase();
          switch (cond) {
            case "is":
              return hay === needle;
            case "is_not":
              return hay !== needle;
            case "contains":
              return hay.includes(needle);
            case "not_contains":
              return !hay.includes(needle);
            case "starts_with":
              return hay.startsWith(needle);
            case "ends_with":
              return hay.endsWith(needle);
          }
          break;
        }
        case "number": {
          if (v.k !== "n") return cond === "ne";
          if (nval === null) return true;
          switch (cond) {
            case "eq":
            case "is":
              return v.n === nval;
            case "ne":
            case "is_not":
              return v.n !== nval;
            case "gt":
              return v.n > nval;
            case "lt":
              return v.n < nval;
            case "gte":
              return v.n >= nval;
            case "lte":
              return v.n <= nval;
          }
          break;
        }
        case "option": {
          const have = this.option(p, row)[0] ?? "";
          const want = asList(raw);
          const inn = want.includes(have);
          if (cond === "is" || cond === "any") return inn;
          if (cond === "is_not" || cond === "none") return !inn;
          if (cond === "group_is") {
            const o = p.options?.find((x) => x.id === have);
            return !!o && want.includes(o.group ?? "");
          }
          break;
        }
        case "list": {
          const have = this.option(p, row);
          const any = asList(raw).some((w) => have.includes(w));
          if (cond === "contains" || cond === "is" || cond === "any") return any;
          if (cond === "not_contains" || cond === "is_not" || cond === "none") return !any;
          break;
        }
        case "files":
        case "valuelist": {
          const hay = F.asText(v).toLowerCase();
          if (cond === "contains") return hay.includes(sval.toLowerCase());
          if (cond === "not_contains") return !hay.includes(sval.toLowerCase());
          break;
        }
        case "checkbox": {
          const want = raw === true || raw === "true";
          if (cond === "is" || cond === "eq") return F.truthy(v) === want;
          if (cond === "is_not" || cond === "ne") return F.truthy(v) !== want;
          break;
        }
        case "date": {
          if (v.k !== "d") return false;
          const start = dayStart(v.t);
          const end = v.end ? dayStart(v.end) : start;
          const o = (raw && typeof raw === "object" && !Array.isArray(raw) ? raw : {}) as { date?: string; to?: string; relative?: string; within?: string };
          if (cond === "within") {
            const w = this.within(o.within ?? "");
            return w ? end >= w[0] && start < w[1] : true;
          }
          const day = this.filterDay({ ...o, s: sval });
          if (day === null) return true;
          switch (cond) {
            case "is":
              return day >= start && day <= end;
            case "is_not":
              return day < start || day > end;
            case "before":
              return start < day;
            case "after":
              return end > day;
            case "on_or_before":
              return start <= day;
            case "on_or_after":
              return end >= day;
            case "between": {
              const to = this.filterDay({ date: o.to });
              return to === null ? end >= day : end >= day && start <= to;
            }
          }
          break;
        }
      }
      return true;
    }

    match(col: Col, row: Row, f: Filter | undefined, depth = 0): boolean {
      if (!f || depth > 4) return true;
      if (!f.prop) {
        const items = f.items ?? [];
        if (!items.length) return true;
        const or = f.op === "or";
        for (const it of items) {
          const ok = this.match(col, row, it, depth + 1);
          if (or && ok) return true;
          if (!or && !ok) return false;
        }
        return !or;
      }
      return this.matchOne(col, row, f);
    }

    search(col: Col, row: Row, q: string): boolean {
      if (row.title.toLowerCase().includes(q)) return true;
      for (const p of col.schema) {
        if (["text", "url", "email", "phone", "select", "multi_select", "status", "formula"].includes(p.type) && F.asText(this.value(col, row, p)).toLowerCase().includes(q)) return true;
      }
      return false;
    }

    overlaps(col: Col, row: Row, rg: NonNullable<QueryInput["range"]>): boolean {
      const p = col.schema.find((x) => x.id === rg.prop);
      if (!p) return false;
      const v = this.value(col, row, p);
      if (v.k !== "d") return false;
      const start = v.t;
      let end = v.end;
      if (rg.end_prop) {
        const ep = col.schema.find((x) => x.id === rg.end_prop);
        const e = ep ? this.value(col, row, ep) : F.EMPTY;
        if (e.k === "d") end = e.end || e.t;
      }
      if (!end || end < start) end = start;
      const from = parseDay(rg.from);
      const to = parseDay(rg.to);
      if (!from || !to) return true;
      return end >= dayStart(from.t) && start <= dayStart(to.t) + DAY - 1000;
    }

    sort(col: Col, rows: Row[], sorts: SortKey[]): Row[] {
      const keys = sorts.flatMap((s) => {
        const p = col.schema.find((x) => x.id === s.prop);
        return p ? [{ p, desc: s.dir === "desc" }] : [];
      });
      if (!keys.length) return rows;
      const keyed = rows.map((r) => ({
        r,
        k: keys.map(({ p }) => {
          if (p.type === "select" || p.type === "status") {
            const id = this.option(p, r)[0];
            const idx = p.options?.findIndex((o) => o.id === id) ?? -1;
            if (idx >= 0) return F.num(idx);
          }
          return this.value(col, r, p);
        }),
      }));
      keyed.sort((a, b) => {
        for (let i = 0; i < keys.length; i++) {
          const x = a.k[i] as V;
          const y = b.k[i] as V;
          if (F.isEmpty(x) !== F.isEmpty(y)) return F.isEmpty(y) ? -1 : 1;
          let c = F.compare(x, y);
          if (c === null) c = F.asText(x).localeCompare(F.asText(y));
          if (c !== 0) return (keys[i] as { desc: boolean }).desc ? -c : c;
        }
        return 0;
      });
      return keyed.map((x) => x.r);
    }

    groupKeys(col: Col, row: Row, g: Grouping): { key: string; label: string; color?: string; rank: number }[] {
      const p = col.schema.find((x) => x.id === g.prop);
      if (!p) return [{ key: "", label: "", rank: 0 }];
      const none = { key: "", label: `No ${p.name}`, rank: Infinity };
      if (p.type === "select" || p.type === "status" || p.type === "multi_select") {
        if (p.type === "status" && g.by === "group") {
          const o = p.options?.find((x) => x.id === this.option(p, row)[0]);
          const groups = ["todo", "doing", "done"];
          const gi = groups.indexOf(o?.group ?? "");
          return gi < 0 ? [none] : [{ key: groups[gi] as string, label: ["To do", "In progress", "Complete"][gi] as string, rank: gi }];
        }
        const out = this.option(p, row).flatMap((id) => {
          const i = p.options?.findIndex((o) => o.id === id) ?? -1;
          const o = p.options?.[i];
          return o ? [{ key: id, label: o.name, color: o.color, rank: i }] : [];
        });
        return out.length ? out : [none];
      }
      if (p.type === "relation" || p.type === "person") {
        const target = this.col(p.collection ?? "");
        const out = this.option(p, row).flatMap((id) => {
          const t = target?.rows.find((r) => r.id === id);
          return t ? [{ key: id, label: t.title, rank: NaN }] : [];
        });
        return out.length ? out : [none];
      }
      if (p.type === "checkbox") return F.truthy(this.value(col, row, p)) ? [{ key: "true", label: "Checked", rank: 0 }] : [{ key: "false", label: "Unchecked", rank: 1 }];
      const v = this.value(col, row, p);
      if (F.isEmpty(v)) return [none];
      if (v.k === "d") {
        const d = new Date(v.t);
        const y = d.getUTCFullYear();
        const mon = F.fmtDate(Date.UTC(y, d.getUTCMonth(), 1), false);
        switch (g.by) {
          case "year":
            return [{ key: String(y), label: String(y), rank: y }];
          case "month":
            return [{ key: `${y}-${String(d.getUTCMonth() + 1).padStart(2, "0")}`, label: mon.replace(" 1,", ""), rank: Date.UTC(y, d.getUTCMonth(), 1) }];
          case "weekday":
            return [{ key: `d${d.getUTCDay()}`, label: ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"][d.getUTCDay()] as string, rank: d.getUTCDay() }];
          case "hour":
            return [{ key: `h${String(d.getUTCHours()).padStart(2, "0")}`, label: `${String(d.getUTCHours()).padStart(2, "0")}:00`, rank: d.getUTCHours() }];
          case "week": {
            const s = dayStart(v.t) - d.getUTCDay() * DAY;
            return [{ key: `${dayStr(s, false)}w`, label: `Week of ${F.fmtDate(s, false)}`, rank: s }];
          }
        }
        const s = dayStart(v.t);
        return [{ key: dayStr(s, false), label: F.fmtDate(s, false), rank: s }];
      }
      if (v.k === "n") {
        const step = g.step && g.step > 0 ? g.step : 10;
        const lo = Math.floor(v.n / step) * step;
        return [{ key: F.fmtNum(lo), label: `${F.fmtNum(lo)} to ${F.fmtNum(lo + step)}`, rank: lo }];
      }
      if (v.k === "b") return [{ key: String(v.b), label: v.b ? "True" : "False", rank: v.b ? 0 : 1 }];
      const s = F.asText(v);
      return [{ key: s.toLowerCase(), label: s, rank: NaN }];
    }

    bucketise(col: Col, rows: Row[], g: Grouping) {
      type B = { key: string; label: string; color?: string; rank: number; rows: Row[] };
      const by = new Map<string, B>();
      const list: B[] = [];
      const p = col.schema.find((x) => x.id === g.prop);
      const seedB = (b: B) => {
        by.set(b.key, b);
        list.push(b);
      };
      if (p && (p.type === "select" || p.type === "status" || p.type === "multi_select") && g.by !== "group") (p.options ?? []).forEach((o, i) => seedB({ key: o.id, label: o.name, color: o.color, rank: i, rows: [] }));
      if (p?.type === "status" && g.by === "group") ["todo", "doing", "done"].forEach((k, i) => seedB({ key: k, label: ["To do", "In progress", "Complete"][i] as string, rank: i, rows: [] }));
      if (p?.type === "checkbox") [["true", "Checked"], ["false", "Unchecked"]].forEach(([k, l], i) => seedB({ key: k as string, label: l as string, rank: i, rows: [] }));
      for (const r of rows) {
        for (const k of this.groupKeys(col, r, g)) {
          let b = by.get(k.key);
          if (!b) {
            b = { ...k, rows: [] };
            seedB(b);
          }
          b.rows.push(r);
        }
      }
      if (!by.has("") && p) seedB({ key: "", label: `No ${p.name}`, rank: Infinity, rows: [] });
      const desc = g.sort === "desc";
      const less = (a: B, b: B) => (Number.isNaN(a.rank) || Number.isNaN(b.rank) ? a.label.toLowerCase() < b.label.toLowerCase() : a.rank < b.rank);
      list.sort((a, b) => {
        if (a.key === "" || b.key === "") return a.key === "" ? (b.key === "" ? 0 : 1) : -1;
        const x = desc ? less(b, a) : less(a, b);
        const y = desc ? less(a, b) : less(b, a);
        return x ? -1 : y ? 1 : 0;
      });
      if (g.order?.length && g.sort === "manual") {
        const pos = new Map(g.order.map((k, i) => [k, i]));
        list.sort((a, b) => {
          const pa = pos.get(a.key);
          const pb = pos.get(b.key);
          if (pa !== undefined && pb !== undefined) return pa - pb;
          if (pa !== undefined) return -1;
          if (pb !== undefined) return 1;
          return 0;
        });
      }
      return list;
    }

    calcs(col: Col, rows: Row[], calcs: Record<string, string> | undefined): Record<string, ComputedValue | null> | undefined {
      if (!calcs || !Object.keys(calcs).length) return undefined;
      const out: Record<string, ComputedValue | null> = {};
      for (const [pid, fn] of Object.entries(calcs)) {
        const p = col.schema.find((x) => x.id === pid);
        if (!p) continue;
        let vals = rows.map((r) => this.value(col, r, p));
        if (["checked", "unchecked", "percent_checked", "percent_unchecked"].includes(fn)) vals = vals.map((v) => F.bool(F.truthy(v)));
        out[pid] = encode(aggregate(fn, vals, rows.length));
      }
      return out;
    }

    ordered(col: Col, cfg: ViewConfig, input: QueryInput): Row[] {
      const q = (input.search ?? "").trim().toLowerCase();
      const rows = col.rows.filter((r) => (!q || this.search(col, r, q)) && (!input.range || this.overlaps(col, r, input.range)) && this.match(col, r, cfg.filter));
      return this.sort(col, rows, cfg.sorts ?? []);
    }
  }

  function shiftMonth(t: number, n: number): number {
    const d = new Date(t);
    d.setUTCMonth(d.getUTCMonth() + n);
    return d.getTime();
  }

  function encode(v: V): ComputedValue | null {
    switch (v.k) {
      case "e":
        return null;
      case "n":
        return { type: "number", value: v.n };
      case "s":
        return { type: "text", value: v.s };
      case "b":
        return { type: "boolean", value: v.b };
      case "d":
        return { type: "date", value: { start: dayStr(v.t, v.time), time: v.time, ...(v.end && v.end !== v.t ? { end: dayStr(v.end, v.time) } : {}) } };
      case "l":
        return { type: "list", value: v.l.map(encode) };
      case "p":
        return { type: "page", value: { id: v.id, title: v.name } };
    }
  }

  function aggregate(fn: string, vals: V[], n: number): V {
    const flat = F.flatten(vals);
    const nonEmpty = vals.filter((v) => !F.isEmpty(v) && !(v.k === "b" && !v.b)).length;
    const pct = (x: number) => F.num(n === 0 ? 0 : x / n);
    const nums = flat.flatMap((v) => (v.k === "n" ? [v.n] : []));
    const checked = flat.filter((v) => v.k === "b" && v.b).length;
    switch (fn) {
      case "show_unique": {
        const out: V[] = [];
        for (const v of flat) if (!F.isEmpty(v) && !out.some((o) => F.same(o, v))) out.push(v);
        return F.list(out);
      }
      case "count_all":
        return F.num(n);
      case "count_values":
        return F.num(flat.filter((v) => !F.isEmpty(v)).length);
      case "count_unique":
        return F.num(new Set(flat.filter((v) => !F.isEmpty(v)).map((v) => F.asText(v).toLowerCase())).size);
      case "count_empty":
        return F.num(n - nonEmpty);
      case "count_not_empty":
        return F.num(nonEmpty);
      case "percent_empty":
        return pct(n - nonEmpty);
      case "percent_not_empty":
        return pct(nonEmpty);
      case "sum":
        return F.num(nums.reduce((s, x) => s + x, 0));
      case "average":
        return nums.length ? F.num(nums.reduce((s, x) => s + x, 0) / nums.length) : F.EMPTY;
      case "median": {
        if (!nums.length) return F.EMPTY;
        const s = [...nums].sort((a, b) => a - b);
        const m = Math.floor(s.length / 2);
        return F.num(s.length % 2 ? (s[m] as number) : ((s[m - 1] as number) + (s[m] as number)) / 2);
      }
      case "min":
        return nums.length ? F.num(Math.min(...nums)) : F.EMPTY;
      case "max":
        return nums.length ? F.num(Math.max(...nums)) : F.EMPTY;
      case "range":
        return nums.length ? F.num(Math.max(...nums) - Math.min(...nums)) : F.EMPTY;
      case "earliest":
      case "latest":
      case "date_range": {
        const ds = flat.filter((v): v is Extract<V, { k: "d" }> => v.k === "d");
        if (!ds.length) return F.EMPTY;
        const lo = Math.min(...ds.map((d) => d.t));
        const hi = Math.max(...ds.map((d) => d.end || d.t));
        const time = ds.some((d) => d.time);
        return fn === "earliest" ? F.date(lo, time) : fn === "latest" ? F.date(hi, time) : F.date(lo, time, hi);
      }
      case "checked":
        return F.num(checked);
      case "unchecked":
        return F.num(flat.length - checked);
      case "percent_checked":
        return F.num(flat.length ? checked / flat.length : 0);
      case "percent_unchecked":
        return F.num(flat.length ? (flat.length - checked) / flat.length : 0);
    }
    return F.list(flat);
  }

  const ev = () => new Evaluator();

  const colOut = (c: Col): Collection => ({
    id: c.id,
    workspace_id: WORKSPACE,
    title: c.title,
    icon: c.icon,
    inline: false,
    schema: c.schema,
    views: c.views,
    can_write: true,
    can_design: true,
    rows: c.rows.length,
    epoch: c.epoch,
  });

  const query = (c: Col, input: QueryInput): QueryResult => {
    const e = ev();
    const view = input.view_id ? c.views.find((v) => v.id === input.view_id) : undefined;
    const cfg: ViewConfig = { ...(view?.config ?? {}) };
    if (input.filter) cfg.filter = input.filter;
    if (input.sorts) cfg.sorts = input.sorts;
    const refs: Record<string, RowRef> = {};
    const res: QueryResult = { total: 0, rows: [], order: [], refs, epoch: c.epoch, offset: 0 };
    if (input.ids?.length) {
      for (const id of input.ids) {
        const r = c.rows.find((x) => x.id === id);
        if (r) {
          res.rows.push(e.rowOut(c, r, refs));
          res.order.push(id);
        }
      }
      res.total = res.rows.length;
      return res;
    }
    const all = e.ordered(c, cfg, input);
    const calcs = e.calcs(c, all, cfg.calcs);
    if (calcs) res.calcs = calcs;
    res.total = all.length;
    const limit = Math.min(input.limit && input.limit > 0 ? input.limit : 100, 1000);
    const groupLimit = Math.min(input.group_limit && input.group_limit > 0 ? input.group_limit : 50, 500);
    const emitted = new Set<string>();
    const add = (r: Row) => {
      if (!emitted.has(r.id)) {
        emitted.add(r.id);
        res.rows.push(e.rowOut(c, r, refs));
      }
    };
    const makeGroups = (rows: Row[], g: Grouping, laneKey: string): GroupResult[] => {
      const hidden = new Set(g.hidden ?? []);
      const out: GroupResult[] = [];
      for (const b of e.bucketise(c, rows, g)) {
        if (g.hide_empty && !b.rows.length) continue;
        const gr: GroupResult = { key: b.key, label: b.label, count: b.rows.length, rows: [] };
        if (b.color) gr.color = b.color;
        if (hidden.has(b.key)) gr.hidden = true;
        let off = 0;
        let lim = groupLimit;
        if (input.group_key !== undefined && input.group_key === b.key && (input.lane_key === undefined || input.lane_key === laneKey)) {
          off = Math.max(0, input.offset ?? 0);
          lim = limit;
        } else if (input.group_key !== undefined) continue;
        if (!gr.hidden || input.group_key !== undefined) {
          for (let i = off; i < b.rows.length && i < off + lim; i++) {
            add(b.rows[i] as Row);
            gr.rows.push((b.rows[i] as Row).id);
          }
        }
        const gc = e.calcs(c, b.rows, cfg.calcs);
        if (gc) gr.calcs = gc;
        out.push(gr);
      }
      return out;
    };
    if (cfg.group?.prop && !input.no_groups && cfg.sub_group?.prop) {
      const hidden = new Set(cfg.sub_group.hidden ?? []);
      const lanes: LaneResult[] = [];
      for (const lane of e.bucketise(c, all, cfg.sub_group)) {
        if (cfg.sub_group.hide_empty && !lane.rows.length) continue;
        if (input.lane_key !== undefined && input.lane_key !== lane.key) continue;
        const lo: LaneResult = { key: lane.key, label: lane.label, count: lane.rows.length, groups: makeGroups(lane.rows, cfg.group, lane.key) };
        if (lane.color) lo.color = lane.color;
        if (hidden.has(lane.key)) lo.hidden = true;
        lanes.push(lo);
      }
      res.lanes = lanes;
    } else if (cfg.group?.prop && !input.no_groups) res.groups = makeGroups(all, cfg.group, "");
    else {
      const off = Math.max(0, input.offset ?? 0);
      res.offset = off;
      for (let i = off; i < all.length && i < off + limit; i++) {
        add(all[i] as Row);
        res.order.push((all[i] as Row).id);
      }
    }
    return res;
  };

  const chart = (c: Col, body: { view_id?: string; chart: ChartSpec; search?: string; pick?: { x: string; series?: string } }): ChartResult => {
    const e = ev();
    const spec = body.chart;
    const cfg = c.views.find((v) => v.id === body.view_id)?.config ?? {};
    const rows = e.ordered(c, { ...cfg, ...(spec.filter ? { filter: spec.filter } : {}) }, { search: body.search ?? "" });
    const xg: Grouping = spec.x ?? { prop: "title" };
    const buckets = e.bucketise(c, rows, { ...xg, hidden: [] });
    const fn = spec.fn ?? "count";
    const measure = (rs: Row[]): number => {
      const p = c.schema.find((x) => x.id === spec.prop);
      if (fn === "count" || !p) return rs.length;
      const vals = rs.map((r) => e.value(c, r, p));
      const out = aggregate(fn === "avg" ? "average" : fn, vals, rs.length);
      return out.k === "n" ? out.n : 0;
    };
    const seriesList = spec.series?.prop ? e.bucketise(c, rows, { ...spec.series, hidden: [] }) : null;
    const xs = buckets.filter((b) => !(spec.hide_empty && !b.rows.length));
    const result: ChartResult = {
      kind: spec.kind,
      x: xs.map((b) => ({ key: b.key, label: b.label, ...(b.color ? { color: b.color } : {}) })),
      series: seriesList
        ? seriesList.map((s) => ({ key: s.key, label: s.label, ...(s.color ? { color: s.color } : {}), values: xs.map((b) => measure(b.rows.filter((r) => s.rows.includes(r)))) }))
        : [{ key: "", label: "", values: xs.map((b) => measure(b.rows)) }],
      value: measure(rows),
      rows: rows.length,
      epoch: c.epoch,
    };
    if (body.pick) {
      const b = xs.find((x) => x.key === body.pick?.x);
      result.picked = (b?.rows ?? []).map((r) => r.id);
    }
    return result;
  };

  const json = (data: unknown, status = 200) => (status === 204 ? new Response(null, { status }) : new Response(JSON.stringify(data), { status, headers: { "Content-Type": "application/json" } }));
  const fail = (status: number, code: string, message: string) => json({ error: { code, message } }, status);

  const normalise = (c: Col, vals: Record<string, unknown>): Record<string, unknown> => {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(vals)) {
      const p = c.schema.find((x) => x.id === k);
      if (!p || ["formula", "rollup", "created_time", "last_edited_time", "created_by", "last_edited_by", "unique_id", "title"].includes(p.type)) continue;
      if (v === null || v === undefined || v === "" || (Array.isArray(v) && v.length === 0)) continue;
      out[k] = v;
    }
    return out;
  };

  const syncPairs = (c: Col, row: Row, before: Record<string, unknown>) => {
    for (const p of c.schema) {
      if (p.type !== "relation" || !p.pair || !p.collection) continue;
      const target = cols.get(p.collection);
      const pairProp = target?.schema.find((x) => x.id === p.pair);
      if (!target || !pairProp) continue;
      const was = new Set(asList(before[p.id]));
      const now = new Set(asList(row.vals[p.id]));
      for (const id of new Set([...was, ...now])) {
        if (was.has(id) === now.has(id)) continue;
        const tr = target.rows.find((r) => r.id === id);
        if (!tr) continue;
        const list = new Set(asList(tr.vals[pairProp.id]));
        if (now.has(id)) list.add(row.id);
        else list.delete(row.id);
        if (list.size) tr.vals[pairProp.id] = [...list];
        else delete tr.vals[pairProp.id];
        if (pairProp.limit === 1 && now.has(id)) {
          const others = [...list].filter((x) => x !== row.id);
          for (const oid of others) {
            const o = c.rows.find((r) => r.id === oid);
            if (o) delete o.vals[p.id];
          }
          tr.vals[pairProp.id] = [row.id];
        }
      }
    }
  };

  const place = (c: Col, row: Row, body: { before_id?: string; after_id?: string; top?: boolean }) => {
    const cur = c.rows.indexOf(row);
    if (cur >= 0) c.rows.splice(cur, 1);
    let at = c.rows.length;
    if (body.top) at = 0;
    if (body.before_id) at = Math.max(0, c.rows.findIndex((r) => r.id === body.before_id));
    if (body.after_id) at = c.rows.findIndex((r) => r.id === body.after_id) + 1;
    c.rows.splice(at, 0, row);
  };

  const handle = async (method: string, url: URL, body: Record<string, unknown>): Promise<Response> => {
    const path = url.pathname;
    let m: RegExpExecArray | null;
    if ((m = /^\/api\/collections\/([^/]+)$/.exec(path)) && method === "GET") {
      const c = cols.get(decodeURIComponent(m[1] as string));
      return c ? json({ collection: colOut(c) }) : fail(404, "not_found", "no such database");
    }
    if ((m = /^\/api\/collections\/([^/]+)\/(query|chart|schema|formula|rows|rows\/delete|search|views)$/.exec(path))) {
      const c = cols.get(decodeURIComponent(m[1] as string));
      if (!c) return fail(404, "not_found", "no such database");
      const op = m[2];
      if (op === "query") return json(query(c, body as QueryInput));
      if (op === "chart") return json(chart(c, body as never));
      if (op === "search") {
        const q = (url.searchParams.get("q") ?? "").toLowerCase();
        return json({ rows: c.rows.filter((r) => !q || r.title.toLowerCase().includes(q)).slice(0, Number(url.searchParams.get("limit") ?? 50)).map((r) => ({ id: r.id, title: r.title, icon: r.icon })) });
      }
      if (op === "formula") {
        try {
          const prog = F.compile(String(body.expr ?? ""));
          const unknown = prog.props.find((n) => !c.schema.some((p) => p.name === n));
          if (unknown) return json({ ok: false, error: `there is no property called "${unknown}"`, pos: 0 });
          return json({ ok: true, pos: 0, props: prog.props });
        } catch (err) {
          return json({ ok: false, error: (err as Error).message, pos: err instanceof F.FormulaError ? err.pos : 0 });
        }
      }
      if (op === "schema") {
        for (const o of (body.ops as SchemaOp[]) ?? []) applySchema(c, o);
        c.epoch++;
        notify(c.id, "schema");
        return json({ schema: c.schema, epoch: c.epoch });
      }
      if (op === "rows") {
        const row: Row = { id: newId("r"), title: String(body.title ?? ""), icon: "", cover: "", vals: {}, number: c.next++, version: 1, created: Date.now(), updated: Date.now() };
        row.vals = normalise(c, (body.values as Record<string, unknown>) ?? {});
        place(c, row, body as never);
        syncPairs(c, row, {});
        c.epoch++;
        notify(c.id, "rows");
        return json({ row: ev().rowOut(c, row, null), epoch: c.epoch });
      }
      if (op === "rows/delete") {
        const ids = new Set((body.ids as string[]) ?? []);
        for (const r of c.rows.filter((x) => ids.has(x.id))) syncPairs(c, { ...r, vals: {} }, r.vals);
        c.rows = c.rows.filter((r) => !ids.has(r.id));
        c.epoch++;
        notify(c.id, "rows");
        return json({ trashed: ids.size });
      }
      if (op === "views") {
        const v: CollectionView = {
          id: newId("v"),
          collection_id: c.id,
          sort_key: String(c.views.length).padStart(4, "0"),
          name: String(body.name || defaultViewName(String(body.type ?? "table"))),
          type: (body.type as ViewType) ?? "table",
          config: (body.config as ViewConfig) ?? {},
          created_at: Date.now(),
          updated_at: Date.now(),
        };
        const after = c.views.findIndex((x) => x.id === body.after_id);
        c.views.splice(after >= 0 ? after + 1 : c.views.length, 0, v);
        notify(c.id, "views");
        return json({ view: v });
      }
    }
    if ((m = /^\/api\/rows\/([^/]+)$/.exec(path))) {
      const id = decodeURIComponent(m[1] as string);
      for (const c of cols.values()) {
        const row = c.rows.find((r) => r.id === id);
        if (!row) continue;
        if (method === "GET") {
          const refs: Record<string, RowRef> = {};
          const out = ev().rowOut(c, row, refs);
          return json({ collection: colOut(c), row: out, refs });
        }
        if (method === "PATCH") {
          const before = structuredClone(row.vals);
          if (typeof body.title === "string") row.title = body.title;
          const vals = (body.values as Record<string, unknown>) ?? {};
          for (const [k, v] of Object.entries(vals)) {
            const p = c.schema.find((x) => x.id === k);
            if (!p || ["formula", "rollup", "title"].includes(p.type)) continue;
            if (v === null || v === undefined || v === "" || (Array.isArray(v) && v.length === 0)) delete row.vals[k];
            else row.vals[k] = v;
          }
          if (body.before_id || body.after_id) place(c, row, body as never);
          row.version++;
          row.updated = Date.now();
          syncPairs(c, row, before);
          c.epoch++;
          notify(c.id, "rows");
          return json({ row: ev().rowOut(c, row, null), epoch: c.epoch });
        }
      }
      return fail(404, "not_found", "no such row");
    }
    if ((m = /^\/api\/views\/([^/]+)\/export\.csv$/.exec(path))) {
      const id = decodeURIComponent(m[1] as string);
      for (const c of cols.values()) {
        const view = c.views.find((x) => x.id === id);
        if (!view) continue;
        const e = ev();
        const shown = view.config.props?.length ? view.config.props.filter((p) => !p.hidden).flatMap((p) => c.schema.filter((s) => s.id === p.id)) : c.schema;
        const quote = (s: string) => (/[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s);
        const cell = (row: Row, p: Property): string => {
          const v = e.value(c, row, p);
          if (v.k === "b") return v.b ? "true" : "false";
          return F.asText(v);
        };
        const lines = [shown.map((p) => quote(p.name)).join(",")];
        for (const row of e.ordered(c, view.config, {})) lines.push(shown.map((p) => quote(cell(row, p))).join(","));
        return new Response(lines.join("\n") + "\n", { status: 200, headers: { "Content-Type": "text/csv" } });
      }
      return fail(404, "not_found", "no such view");
    }
    if ((m = /^\/api\/views\/([^/]+)(?:\/(move|duplicate))?$/.exec(path))) {
      const id = decodeURIComponent(m[1] as string);
      for (const c of cols.values()) {
        const i = c.views.findIndex((v) => v.id === id);
        if (i < 0) continue;
        const v = c.views[i] as CollectionView;
        if (m[2] === "move") {
          c.views.splice(i, 1);
          const after = c.views.findIndex((x) => x.id === body.after_id);
          c.views.splice(after + 1, 0, v);
          notify(c.id, "views");
          return json(null, 204);
        }
        if (m[2] === "duplicate") {
          const copy: CollectionView = { ...structuredClone(v), id: newId("v"), name: `${v.name} copy` };
          c.views.splice(i + 1, 0, copy);
          notify(c.id, "views");
          return json({ view: copy });
        }
        if (method === "PATCH") {
          if (typeof body.name === "string") v.name = body.name;
          if (body.type) v.type = body.type as ViewType;
          if (body.config) v.config = body.config as ViewConfig;
          v.updated_at = Date.now();
          return json({ view: v });
        }
        if (method === "DELETE") {
          c.views.splice(i, 1);
          notify(c.id, "views");
          return json(null, 204);
        }
      }
      return fail(404, "not_found", "no such view");
    }
    if (/^\/api\/workspaces\/[^/]+\/members/.test(path)) return json({ members: [] });
    return json({});
  };

  const defaultViewName = (t: string) => t.charAt(0).toUpperCase() + t.slice(1);

  const applySchema = (c: Col, o: SchemaOp) => {
    if (o.op === "add" && o.prop) {
      const p = { ...o.prop, id: o.prop.id || newId("p") } as Property;
      if (p.type === "select" || p.type === "multi_select" || p.type === "status") p.options = (p.options ?? []).map((x) => ({ ...x, id: x.id || newId("o") }));
      const at = o.after ? c.schema.findIndex((x) => x.id === o.after) + 1 : c.schema.length;
      c.schema.splice(at, 0, p);
      if (p.type === "relation" && o.two_way && p.collection) {
        const target = cols.get(p.collection);
        if (target) {
          const pair: Property = { id: newId("p"), name: o.pair_name || c.title, type: "relation", collection: c.id, pair: p.id };
          target.schema.push(pair);
          p.pair = pair.id;
        }
      }
    } else if (o.op === "update" && o.id && o.prop) {
      const i = c.schema.findIndex((x) => x.id === o.id);
      if (i >= 0) {
        const p = { ...(c.schema[i] as Property), ...o.prop, id: o.id } as Property;
        if (p.options) p.options = p.options.map((x) => ({ ...x, id: x.id || newId("o") }));
        c.schema[i] = p;
      }
    } else if (o.op === "delete" && o.id) {
      c.schema = c.schema.filter((x) => x.id !== o.id);
      for (const r of c.rows) delete r.vals[o.id];
      for (const v of c.views) if (v.config.props) v.config.props = v.config.props.filter((x) => x.id !== o.id);
    } else if (o.op === "move" && o.id) {
      const i = c.schema.findIndex((x) => x.id === o.id);
      if (i >= 0) {
        const [p] = c.schema.splice(i, 1);
        const at = o.after ? c.schema.findIndex((x) => x.id === o.after) + 1 : 0;
        c.schema.splice(at, 0, p as Property);
      }
    }
  };

  const fetchFn: typeof fetch = async (input, init) => {
    const req = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(req, "http://demo.local");
    const method = (init?.method ?? "GET").toUpperCase();
    let body: Record<string, unknown> = {};
    if (typeof init?.body === "string") {
      try {
        body = JSON.parse(init.body) as Record<string, unknown>;
      } catch {
        body = {};
      }
    }
    await Promise.resolve();
    return handle(method, url, body);
  };

  return {
    fetch: fetchFn,
    attach(store) {
      engine = { collections: store };
    },
    collection: (id) => cols.get(id),
    addCollection,
    reset,
  };
}
