import type { GroupResult, Property, RowData } from "../../web/src/engine/types";
import { formatCalc, formatDay, type DateValue } from "../../web/src/extensions/database/props";
import { cellText } from "./csv";

export interface Rule {
  prop: string;
  cond: string;
  value: string;
}

export interface SortSpec {
  prop: string;
  dir: "asc" | "desc";
}

export interface ViewState {
  type: "table" | "board";
  rules: Rule[];
  sorts: SortSpec[];
  group: string;
  boardBy: string;
  calcs: Record<string, string>;
  search: string;
}

export const COND_LABELS: Record<string, string> = {
  contains: "Contains",
  not_contains: "Does not contain",
  is: "Is",
  is_not: "Is not",
  eq: "=",
  ne: "Not equal",
  gt: "Greater than",
  gte: "At least",
  lt: "Less than",
  lte: "At most",
  before: "Is before",
  after: "Is after",
  on: "Is on",
  empty: "Is empty",
  not_empty: "Is not empty",
  checked: "Is checked",
  unchecked: "Is unchecked",
};

export function condsFor(p: Property): string[] {
  switch (p.type) {
    case "number":
      return ["eq", "ne", "gt", "gte", "lt", "lte", "empty", "not_empty"];
    case "select":
    case "status":
      return ["is", "is_not", "empty", "not_empty"];
    case "multi_select":
      return ["contains", "not_contains", "empty", "not_empty"];
    case "checkbox":
      return ["checked", "unchecked"];
    case "date":
      return ["on", "before", "after", "empty", "not_empty"];
    default:
      return ["contains", "not_contains", "is", "is_not", "empty", "not_empty"];
  }
}

export function needsValue(cond: string): boolean {
  return !["empty", "not_empty", "checked", "unchecked"].includes(cond);
}

export function calcOptions(p: Property): [string, string][] {
  const common: [string, string][] = [["", "None"], ["count_all", "Count all"], ["count_empty", "Count empty"], ["percent_not_empty", "Percent filled"]];
  if (p.type === "number") return [...common, ["sum", "Sum"], ["average", "Average"], ["median", "Median"], ["min", "Min"], ["max", "Max"]];
  if (p.type === "date") return [...common, ["earliest", "Earliest date"], ["latest", "Latest date"]];
  if (p.type === "checkbox") return [["", "None"], ["count_all", "Count all"], ["checked", "Checked"], ["percent_checked", "Percent checked"]];
  return common;
}

function ruleMatches(p: Property, row: RowData, r: Rule): boolean {
  const v = row.values[p.id];
  const text = cellText(p, row).toLowerCase();
  const want = r.value.trim().toLowerCase();
  const blank = p.type === "checkbox" ? false : v === undefined || v === null || v === "" || (Array.isArray(v) && v.length === 0);
  switch (r.cond) {
    case "empty":
      return p.type === "title" ? row.title === "" : blank;
    case "not_empty":
      return p.type === "title" ? row.title !== "" : !blank;
    case "checked":
      return v === true;
    case "unchecked":
      return v !== true;
    case "contains":
      if (p.type === "multi_select") return Array.isArray(v) && v.includes(r.value);
      return text.includes(want);
    case "not_contains":
      if (p.type === "multi_select") return !(Array.isArray(v) && v.includes(r.value));
      return !text.includes(want);
    case "is":
      if (p.type === "select" || p.type === "status") return v === r.value;
      return text === want;
    case "is_not":
      if (p.type === "select" || p.type === "status") return v !== r.value;
      return text !== want;
  }
  if (p.type === "number") {
    if (typeof v !== "number") return false;
    const n = Number(r.value);
    if (!Number.isFinite(n)) return true;
    switch (r.cond) {
      case "eq":
        return v === n;
      case "ne":
        return v !== n;
      case "gt":
        return v > n;
      case "gte":
        return v >= n;
      case "lt":
        return v < n;
      case "lte":
        return v <= n;
    }
  }
  if (p.type === "date") {
    const d = (v as DateValue | undefined)?.start?.slice(0, 10);
    if (!d || !r.value) return !r.value;
    if (r.cond === "on") return d === r.value;
    if (r.cond === "before") return d < r.value;
    if (r.cond === "after") return d > r.value;
  }
  return true;
}

export function filterRows(schema: Property[], rows: RowData[], view: ViewState): RowData[] {
  const q = view.search.trim().toLowerCase();
  return rows.filter((row) => {
    for (const r of view.rules) {
      const p = schema.find((x) => x.id === r.prop);
      if (p && !ruleMatches(p, row, r)) return false;
    }
    if (!q) return true;
    return schema.some((p) => cellText(p, row).toLowerCase().includes(q));
  });
}

function sortValue(p: Property, row: RowData): string | number {
  const v = row.values[p.id];
  if (p.type === "title") return row.title.toLowerCase();
  if (p.type === "number") return typeof v === "number" ? v : Number.NEGATIVE_INFINITY;
  if (p.type === "checkbox") return v === true ? 1 : 0;
  if (p.type === "select" || p.type === "status") {
    const i = p.options?.findIndex((o) => o.id === v) ?? -1;
    return i < 0 ? Number.MAX_SAFE_INTEGER : i;
  }
  if (p.type === "date") return (v as DateValue | undefined)?.start ?? "";
  return cellText(p, row).toLowerCase();
}

export function sortRows(schema: Property[], rows: RowData[], sorts: SortSpec[]): RowData[] {
  const keys = sorts.map((s) => ({ p: schema.find((x) => x.id === s.prop), dir: s.dir === "asc" ? 1 : -1 })).filter((k): k is { p: Property; dir: number } => !!k.p);
  if (!keys.length) return [...rows].sort((a, b) => a.sort_key.localeCompare(b.sort_key));
  return [...rows].sort((a, b) => {
    for (const k of keys) {
      const x = sortValue(k.p, a);
      const y = sortValue(k.p, b);
      if (x < y) return -k.dir;
      if (x > y) return k.dir;
    }
    return a.sort_key.localeCompare(b.sort_key);
  });
}

export function groupRows(p: Property, rows: RowData[]): GroupResult[] {
  const buckets = new Map<string, GroupResult>();
  const put = (key: string, label: string, color: string | undefined, id: string) => {
    let g = buckets.get(key);
    if (!g) {
      g = { key, label, count: 0, rows: [] };
      if (color) g.color = color;
      buckets.set(key, g);
    }
    g.rows.push(id);
    g.count++;
  };
  if (p.type === "select" || p.type === "status") {
    for (const o of p.options ?? []) buckets.set(o.id, { key: o.id, label: o.name, color: o.color, count: 0, rows: [] });
    buckets.set("", { key: "", label: "", count: 0, rows: [] });
    for (const r of rows) {
      const v = r.values[p.id];
      const o = p.options?.find((x) => x.id === v);
      put(o ? o.id : "", o?.name ?? "", o?.color, r.id);
    }
    const out = [...buckets.values()];
    const empty = out.find((g) => g.key === "");
    return out.filter((g) => g.key !== "").concat(empty && empty.count ? [empty] : []);
  }
  if (p.type === "checkbox") {
    for (const r of rows) put(r.values[p.id] === true ? "1" : "0", r.values[p.id] === true ? "Checked" : "Unchecked", undefined, r.id);
    return [...buckets.values()].sort((a, b) => b.key.localeCompare(a.key));
  }
  for (const r of rows) {
    const t = cellText(p, r);
    put(t, t, undefined, r.id);
  }
  return [...buckets.values()].sort((a, b) => a.label.localeCompare(b.label));
}

export function calcValue(p: Property, rows: RowData[], fn: string): string {
  if (!fn) return "";
  const vals = rows.map((r) => (p.type === "title" ? r.title : r.values[p.id]));
  const filled = vals.filter((v) => !(v === undefined || v === null || v === "" || v === false));
  const nums = vals.filter((v): v is number => typeof v === "number").sort((a, b) => a - b);
  const total = rows.length || 1;
  switch (fn) {
    case "count_all":
      return formatCalc(rows.length, fn);
    case "count_empty":
      return formatCalc(rows.length - filled.length, fn);
    case "percent_not_empty":
      return formatCalc(filled.length / total, fn);
    case "checked":
      return formatCalc(vals.filter((v) => v === true).length, fn);
    case "percent_checked":
      return formatCalc(vals.filter((v) => v === true).length / total, fn);
    case "sum":
      return nums.length ? formatCalc(nums.reduce((a, b) => a + b, 0), fn, p) : "";
    case "average":
      return nums.length ? formatCalc(nums.reduce((a, b) => a + b, 0) / nums.length, fn, p) : "";
    case "median": {
      if (!nums.length) return "";
      const mid = Math.floor(nums.length / 2);
      return formatCalc(nums.length % 2 ? (nums[mid] as number) : ((nums[mid - 1] as number) + (nums[mid] as number)) / 2, fn, p);
    }
    case "min":
      return nums.length ? formatCalc(nums[0] as number, fn, p) : "";
    case "max":
      return nums.length ? formatCalc(nums[nums.length - 1] as number, fn, p) : "";
    case "earliest":
    case "latest": {
      const days = vals.map((v) => (v as DateValue | undefined)?.start).filter((s): s is string => !!s).sort();
      const pick = fn === "earliest" ? days[0] : days[days.length - 1];
      return pick ? formatDay(pick) : "";
    }
  }
  return "";
}

export function freshView(type: ViewState["type"] = "table"): ViewState {
  return { type, rules: [], sorts: [], group: "", boardBy: "", calcs: {}, search: "" };
}
