import { newBlockId } from "../../web/src/engine/doc/schema";
import type { Property, PropOption, RowData } from "../../web/src/engine/types";
import { OPTION_COLORS, formatDay, parseISO, isoDay, type DateValue } from "../../web/src/extensions/database/props";

export interface Table {
  schema: Property[];
  rows: RowData[];
}

const STATUS_GROUP: Record<string, "todo" | "doing" | "done"> = {
  todo: "todo", "to do": "todo", "not started": "todo", open: "todo", backlog: "todo", planned: "todo", new: "todo", idea: "todo", pending: "todo",
  doing: "doing", "in progress": "doing", started: "doing", active: "doing", review: "doing", "in review": "doing", "needs work": "doing", blocked: "doing", wip: "doing",
  done: "done", complete: "done", completed: "done", closed: "done", finished: "done", shipped: "done", resolved: "done", cancelled: "done",
};

const BOOL_TRUE = new Set(["true", "yes", "checked", "x", "y"]);
const BOOL_FALSE = new Set(["false", "no", "unchecked", "n"]);
const NUMBER = /^[-+]?[$€£]?\s?\d[\d,]*(\.\d+)?%?$|^[-+]?[$€£]?\s?\.\d+%?$/;
const ISO_DATE = /^\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2})?/;
const US_DATE = /^(\d{1,2})\/(\d{1,2})\/(\d{4})$/;
const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const URL_RE = /^https?:\/\/\S+$/i;

export function detectDelimiter(text: string): string {
  const head = text.split(/\r?\n/, 1)[0] ?? "";
  const count = (c: string) => head.split(c).length - 1;
  const tabs = count("\t");
  const semi = count(";");
  const commas = count(",");
  if (tabs > 0 && tabs >= commas) return "\t";
  if (semi > commas) return ";";
  return ",";
}

export function parseDelimited(text: string, delimiter = detectDelimiter(text)): string[][] {
  const out: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let quoted = false;
  const src = text.replace(/^﻿/, "");
  for (let i = 0; i < src.length; i++) {
    const c = src[i] as string;
    if (quoted) {
      if (c === '"') {
        if (src[i + 1] === '"') {
          cell += '"';
          i++;
        } else quoted = false;
      } else cell += c;
    } else if (c === '"' && cell === "") quoted = true;
    else if (c === delimiter) {
      row.push(cell);
      cell = "";
    } else if (c === "\n" || c === "\r") {
      if (c === "\r" && src[i + 1] === "\n") i++;
      row.push(cell);
      cell = "";
      out.push(row);
      row = [];
    } else cell += c;
  }
  if (cell !== "" || row.length) {
    row.push(cell);
    out.push(row);
  }
  return out.filter((r) => r.some((x) => x.trim() !== ""));
}

function toNumber(s: string): number {
  const pct = s.trim().endsWith("%");
  const n = Number(s.replace(/[^\d.+-]/g, ""));
  return pct ? n / 100 : n;
}

function toIso(s: string): string {
  const us = US_DATE.exec(s.trim());
  if (us) return `${us[3]}-${(us[1] as string).padStart(2, "0")}-${(us[2] as string).padStart(2, "0")}`;
  const iso = ISO_DATE.exec(s.trim());
  return iso ? iso[0].replace(" ", "T") : "";
}

function optionsFor(names: string[], status: boolean): PropOption[] {
  return names.map((name, i) => {
    const o: PropOption = { id: `o${i + 1}`, name, color: OPTION_COLORS[(i + 1) % OPTION_COLORS.length] as string };
    if (status) o.group = STATUS_GROUP[name.toLowerCase()] ?? "todo";
    return o;
  });
}

function inferProp(name: string, id: string, vals: string[], first: boolean): Property {
  const base = { id, name: name || "Column" };
  if (first) return { ...base, type: "title" };
  const filled = vals.filter((v) => v.trim() !== "");
  if (!filled.length) return { ...base, type: "text" };
  const lower = filled.map((v) => v.trim().toLowerCase());
  const distinct = [...new Set(filled.map((v) => v.trim()))];
  if (lower.every((v) => BOOL_TRUE.has(v) || BOOL_FALSE.has(v)) && lower.some((v) => BOOL_TRUE.has(v) || BOOL_FALSE.has(v))) return { ...base, type: "checkbox" };
  if (filled.every((v) => NUMBER.test(v.trim()))) {
    const p: Property = { ...base, type: "number" };
    if (filled.every((v) => v.trim().endsWith("%"))) p.format = "percent";
    return p;
  }
  if (filled.every((v) => toIso(v) !== "")) return { ...base, type: "date" };
  if (filled.every((v) => EMAIL.test(v.trim()))) return { ...base, type: "email" };
  if (filled.every((v) => URL_RE.test(v.trim()))) return { ...base, type: "url" };
  if (distinct.length >= 2 && distinct.length <= 8 && lower.every((v) => Object.hasOwn(STATUS_GROUP, v))) {
    const names = [...new Set(distinct.map((d) => d))].sort((a, b) => rankStatus(a) - rankStatus(b));
    return { ...base, type: "status", options: optionsFor(names, true) };
  }
  const longest = Math.max(...filled.map((v) => v.length));
  if (distinct.length <= 12 && longest <= 32 && (distinct.length * 2 <= filled.length || (distinct.length <= 6 && filled.length >= 6))) {
    return { ...base, type: "select", options: optionsFor(distinct, false) };
  }
  return { ...base, type: "text" };
}

function rankStatus(name: string): number {
  const g = STATUS_GROUP[name.toLowerCase()];
  return g === "todo" ? 0 : g === "doing" ? 1 : 2;
}

function cellValue(p: Property, raw: string): unknown {
  const v = raw.trim();
  if (v === "") return undefined;
  switch (p.type) {
    case "number":
      return toNumber(v);
    case "checkbox":
      return BOOL_TRUE.has(v.toLowerCase());
    case "date":
      return { start: toIso(v) } satisfies DateValue;
    case "select":
    case "status":
      return p.options?.find((o) => o.name.toLowerCase() === v.toLowerCase())?.id;
    default:
      return v;
  }
}

export function buildTable(grid: string[][]): Table {
  const header = grid[0] ?? [];
  const body = grid.slice(1);
  const width = Math.max(header.length, ...body.map((r) => r.length), 1);
  const used = new Set<string>();
  const schema: Property[] = [];
  for (let c = 0; c < width; c++) {
    let name = (header[c] ?? "").trim() || `Column ${c + 1}`;
    while (used.has(name.toLowerCase())) name += " 2";
    used.add(name.toLowerCase());
    schema.push(inferProp(name, c === 0 ? "title" : `p${c}`, body.map((r) => r[c] ?? ""), c === 0));
  }
  const now = Date.now();
  const rows: RowData[] = body.map((r, i) => {
    const values: Record<string, unknown> = {};
    schema.forEach((p, c) => {
      if (p.type === "title") return;
      const v = cellValue(p, r[c] ?? "");
      if (v !== undefined) values[p.id] = v;
    });
    return { id: newBlockId(), title: (r[0] ?? "").trim(), icon: "", sort_key: String(i).padStart(6, "0"), values, number: i + 1, version: 1, created_at: now, created_by: "", updated_at: now, updated_by: "" };
  });
  return { schema, rows };
}

export function cellText(p: Property, row: RowData): string {
  if (p.type === "title") return row.title;
  const v = row.values[p.id];
  if (v === undefined || v === null) return "";
  switch (p.type) {
    case "number":
      return typeof v === "number" ? String(p.format === "percent" ? Number((v * 100).toPrecision(12)) + "%" : v) : "";
    case "checkbox":
      return v === true ? "true" : "false";
    case "date": {
      const d = v as DateValue;
      return d.start ?? "";
    }
    case "select":
    case "status":
      return p.options?.find((o) => o.id === v)?.name ?? "";
    case "multi_select":
      return Array.isArray(v) ? v.map((id) => p.options?.find((o) => o.id === id)?.name ?? "").join("; ") : "";
    default:
      return typeof v === "string" ? v : String(v);
  }
}

export function displayText(p: Property, row: RowData): string {
  if (p.type === "date") {
    const d = row.values[p.id] as DateValue | undefined;
    return d?.start ? formatDay(d.start) : "";
  }
  if (p.type === "checkbox") return row.values[p.id] === true ? "Yes" : "";
  return cellText(p, row);
}

function quote(s: string, delimiter: string): string {
  return /["\r\n]/.test(s) || s.includes(delimiter) ? `"${s.replace(/"/g, '""')}"` : s;
}

export function toDelimited(table: Table, delimiter = ","): string {
  const lines = [table.schema.map((p) => quote(p.name, delimiter)).join(delimiter)];
  for (const r of table.rows) lines.push(table.schema.map((p) => quote(cellText(p, r), delimiter)).join(delimiter));
  return `${lines.join("\n")}\n`;
}

export function toMarkdownTable(table: Table): string {
  const esc = (s: string) => s.replace(/\|/g, "\\|").replace(/\n/g, " ");
  const head = `| ${table.schema.map((p) => esc(p.name)).join(" | ")} |`;
  const sep = `| ${table.schema.map((p) => (p.type === "number" ? "---:" : "---")).join(" | ")} |`;
  const body = table.rows.map((r) => `| ${table.schema.map((p) => esc(displayText(p, r))).join(" | ")} |`);
  return [head, sep, ...body].join("\n");
}

export function todayIso(): string {
  return isoDay(new Date());
}

export function isoValid(s: string): boolean {
  return parseISO(s) !== null;
}
