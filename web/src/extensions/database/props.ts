import type { ComputedValue, Property, PropType, RowData, ViewType } from "../../engine/types";
import { t } from "../../engine/i18n";
import { locale } from "../../engine/format";
import type { IconName } from "../../ui/icons";

export const TYPE_INFO: Record<PropType, { label: string; icon: IconName }> = {
  title: { label: "Title", icon: "text" },
  text: { label: "Text", icon: "align-left" },
  number: { label: "Number", icon: "hash" },
  select: { label: "Select", icon: "select" },
  multi_select: { label: "Multi-select", icon: "multi" },
  status: { label: "Status", icon: "status" },
  date: { label: "Date", icon: "calendar" },
  person: { label: "Person", icon: "users" },
  files: { label: "Files and media", icon: "attachment" },
  checkbox: { label: "Checkbox", icon: "checkbox" },
  url: { label: "URL", icon: "link" },
  email: { label: "Email", icon: "mail" },
  phone: { label: "Phone", icon: "phone" },
  relation: { label: "Relation", icon: "relation" },
  rollup: { label: "Rollup", icon: "rollup" },
  formula: { label: "Formula", icon: "formula" },
  created_time: { label: "Created time", icon: "clock" },
  created_by: { label: "Created by", icon: "user" },
  last_edited_time: { label: "Last edited time", icon: "clock" },
  last_edited_by: { label: "Last edited by", icon: "user" },
  unique_id: { label: "ID", icon: "id" },
  place: { label: "Place", icon: "pin" },
  button: { label: "Button", icon: "button" },
};

export const CREATABLE: PropType[] = [
  "text", "number", "select", "multi_select", "status", "date", "person", "files", "checkbox", "url", "email", "phone",
  "relation", "rollup", "formula", "created_time", "created_by", "last_edited_time", "last_edited_by", "unique_id", "place", "button",
];

export const VIEW_INFO: Record<ViewType, { label: string; icon: IconName }> = {
  table: { label: "Table", icon: "table" },
  board: { label: "Board", icon: "board" },
  list: { label: "List", icon: "rows" },
  gallery: { label: "Gallery", icon: "gallery" },
  calendar: { label: "Calendar", icon: "calendar" },
  timeline: { label: "Timeline", icon: "timeline" },
  form: { label: "Form", icon: "form" },
  chart: { label: "Chart", icon: "chart" },
  map: { label: "Map", icon: "pin" },
  feed: { label: "Feed", icon: "feed" },
  dashboard: { label: "Dashboard", icon: "dashboard" },
};

export const PHASE5_VIEWS: ViewType[] = ["table", "board", "list", "gallery", "calendar", "timeline", "form", "chart", "dashboard"];

export const OPTION_COLORS = ["default", "gray", "brown", "orange", "yellow", "green", "blue", "purple", "pink", "red"] as const;

export function computed(type: PropType): boolean {
  return ["rollup", "formula", "created_time", "created_by", "last_edited_time", "last_edited_by", "unique_id", "button"].includes(type);
}

export function typeLabel(type: PropType): string {
  return t(TYPE_INFO[type]?.label ?? type);
}

export type CondKind = "text" | "number" | "option" | "list" | "files" | "date" | "checkbox" | "valuelist";

export function condKind(p: Property, sample?: ComputedValue | null): CondKind {
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
  switch (sample?.type) {
    case "number":
      return "number";
    case "date":
      return "date";
    case "boolean":
      return "checkbox";
    case "list":
      return "valuelist";
  }
  return p.type === "formula" && p.expr && /\b(now|today|date|dateadd|datesubtract|daterange|parsedate|fromtimestamp)\(/i.test(p.expr) ? "date" : "text";
}

export const CONDS: Record<CondKind, [string, string, boolean][]> = {
  text: [["contains", "Contains", true], ["not_contains", "Does not contain", true], ["is", "Is", true], ["is_not", "Is not", true], ["starts_with", "Starts with", true], ["ends_with", "Ends with", true], ["empty", "Is empty", false], ["not_empty", "Is not empty", false]],
  number: [["eq", "=", true], ["ne", "≠", true], ["gt", ">", true], ["lt", "<", true], ["gte", "≥", true], ["lte", "≤", true], ["empty", "Is empty", false], ["not_empty", "Is not empty", false]],
  option: [["is", "Is", true], ["is_not", "Is not", true], ["empty", "Is empty", false], ["not_empty", "Is not empty", false]],
  list: [["contains", "Contains", true], ["not_contains", "Does not contain", true], ["empty", "Is empty", false], ["not_empty", "Is not empty", false]],
  files: [["contains", "Name contains", true], ["empty", "Is empty", false], ["not_empty", "Is not empty", false]],
  valuelist: [["contains", "Contains", true], ["not_contains", "Does not contain", true], ["empty", "Is empty", false], ["not_empty", "Is not empty", false]],
  date: [["is", "Is", true], ["before", "Is before", true], ["after", "Is after", true], ["on_or_before", "Is on or before", true], ["on_or_after", "Is on or after", true], ["between", "Is between", true], ["within", "Is within", true], ["empty", "Is empty", false], ["not_empty", "Is not empty", false]],
  checkbox: [["is", "Is", true]],
};

export const RELATIVE_DAYS: [string, string][] = [
  ["today", "Today"], ["tomorrow", "Tomorrow"], ["yesterday", "Yesterday"], ["one_week_ago", "One week ago"], ["one_week_from_now", "One week from now"],
  ["one_month_ago", "One month ago"], ["one_month_from_now", "One month from now"],
];

export const WITHIN: [string, string][] = [
  ["past_week", "The past week"], ["next_week", "The next week"], ["past_month", "The past month"], ["next_month", "The next month"],
  ["past_year", "The past year"], ["next_year", "The next year"], ["this_week", "This week"], ["this_month", "This month"], ["this_year", "This year"],
];

export const CALCS: [string, string][] = [
  ["count_all", "Count all"], ["count_values", "Count values"], ["count_unique", "Count unique values"], ["count_empty", "Count empty"],
  ["count_not_empty", "Count not empty"], ["percent_empty", "Percent empty"], ["percent_not_empty", "Percent not empty"],
  ["sum", "Sum"], ["average", "Average"], ["median", "Median"], ["min", "Min"], ["max", "Max"], ["range", "Range"],
  ["earliest", "Earliest date"], ["latest", "Latest date"], ["date_range", "Date range"],
  ["checked", "Checked"], ["unchecked", "Unchecked"], ["percent_checked", "Percent checked"], ["percent_unchecked", "Percent unchecked"],
];

export function calcsFor(p: Property): [string, string][] {
  const base = CALCS.slice(0, 7);
  if (p.type === "number" || p.type === "formula" || p.type === "rollup" || p.type === "unique_id") return [...base, ...CALCS.slice(7, 13)];
  if (p.type === "date" || p.type === "created_time" || p.type === "last_edited_time") return [...base, ...CALCS.slice(13, 16)];
  if (p.type === "checkbox") return [CALCS[0] as [string, string], ...CALCS.slice(16)];
  return base;
}

export const ROLLUP_FNS: [string, string][] = [["show_original", "Show original"], ["show_unique", "Show unique values"], ...CALCS];

const CURRENCY: Record<string, string> = {
  dollar: "USD", euro: "EUR", pound: "GBP", yen: "JPY", rupee: "INR", won: "KRW", yuan: "CNY", franc: "CHF", real: "BRL", krona: "SEK",
  ruble: "RUB", peso: "MXN", lira: "TRY", zloty: "PLN", rand: "ZAR",
};

export const NUMBER_FORMATS: [string, string][] = [
  ["", "Number"], ["comma", "Number with commas"], ["percent", "Percent"], ["dollar", "US dollar"], ["euro", "Euro"], ["pound", "Pound"],
  ["yen", "Yen"], ["rupee", "Rupee"], ["won", "Won"], ["yuan", "Yuan"], ["franc", "Franc"], ["real", "Real"], ["krona", "Krona"],
  ["ruble", "Ruble"], ["peso", "Peso"], ["lira", "Lira"], ["zloty", "Zloty"], ["rand", "Rand"],
];

export const DATE_FORMATS: [string, string][] = [["", "Default"], ["full", "Full date"], ["short", "Month and day"], ["us", "Month/day/year"], ["eu", "Day/month/year"], ["iso", "Year-month-day"], ["relative", "Relative"]];

export function formatNumber(n: number, p: Pick<Property, "format" | "precision">): string {
  const fmt = p.format ?? "";
  const digits = p.precision !== undefined && p.precision > 0 ? p.precision : undefined;
  const base: Intl.NumberFormatOptions = digits !== undefined ? { minimumFractionDigits: digits, maximumFractionDigits: digits } : { maximumFractionDigits: 8 };
  try {
    if (fmt === "percent") return new Intl.NumberFormat(locale(), { ...base, style: "percent" }).format(n);
    if (CURRENCY[fmt]) return new Intl.NumberFormat(locale(), { ...base, style: "currency", currency: CURRENCY[fmt] }).format(n);
    if (fmt === "comma") return new Intl.NumberFormat(locale(), base).format(n);
  } catch {
    return String(n);
  }
  if (digits !== undefined) return n.toFixed(digits);
  return Number.isInteger(n) ? String(n) : String(Math.round(n * 1e8) / 1e8);
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

export function parseISO(s: string): { date: Date; time: boolean } | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}))?/.exec(s);
  if (!m) return null;
  const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]), m[4] ? Number(m[4]) : 0, m[5] ? Number(m[5]) : 0);
  return { date: d, time: !!m[4] };
}

export function isoDay(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function isoTime(d: Date): string {
  return `${isoDay(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function formatDay(iso: string, p?: Pick<Property, "format" | "time_format">): string {
  const v = parseISO(iso);
  if (!v) return iso;
  const d = v.date;
  const fmt = p?.format ?? "";
  let s: string;
  switch (fmt) {
    case "short":
      s = new Intl.DateTimeFormat(locale(), { month: "short", day: "numeric" }).format(d);
      break;
    case "us":
      s = `${pad(d.getMonth() + 1)}/${pad(d.getDate())}/${d.getFullYear()}`;
      break;
    case "eu":
      s = `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()}`;
      break;
    case "iso":
      s = isoDay(d);
      break;
    case "full":
      s = new Intl.DateTimeFormat(locale(), { dateStyle: "long" }).format(d);
      break;
    case "relative": {
      const today = new Date();
      today.setHours(0, 0, 0, 0);
      const day = new Date(d);
      day.setHours(0, 0, 0, 0);
      const diff = Math.round((day.getTime() - today.getTime()) / 86400000);
      s = Math.abs(diff) <= 6 ? new Intl.RelativeTimeFormat(locale(), { numeric: "auto" }).format(diff, "day") : new Intl.DateTimeFormat(locale(), { dateStyle: "medium" }).format(d);
      break;
    }
    default:
      s = new Intl.DateTimeFormat(locale(), { dateStyle: "medium" }).format(d);
  }
  if (v.time) {
    const opts: Intl.DateTimeFormatOptions = { hour: "numeric", minute: "2-digit" };
    if (p?.time_format === "24") opts.hourCycle = "h23";
    if (p?.time_format === "12") opts.hourCycle = "h12";
    s += ` ${new Intl.DateTimeFormat(locale(), opts).format(d)}`;
  }
  return s;
}

export interface DateValue {
  start: string;
  end?: string;
  remind?: string;
}

export function formatRange(v: DateValue, p?: Pick<Property, "format" | "time_format">): string {
  if (!v.start) return "";
  return v.end && v.end !== v.start ? `${formatDay(v.start, p)} → ${formatDay(v.end, p)}` : formatDay(v.start, p);
}

export function isEmptyValue(v: unknown): boolean {
  return v === undefined || v === null || v === "" || v === false || (Array.isArray(v) && v.length === 0);
}

export function asList(v: unknown): string[] {
  if (Array.isArray(v)) return v.filter((x): x is string => typeof x === "string");
  if (typeof v === "string" && v) return [v];
  return [];
}

export function computedText(v: ComputedValue | null | undefined, p?: Property): string {
  if (!v) return "";
  switch (v.type) {
    case "number":
      return formatNumber(v.value as number, p ?? {});
    case "boolean":
      return v.value ? t("Yes") : t("No");
    case "date": {
      const d = v.value as { start: string; end?: string };
      return formatRange({ start: d.start, ...(d.end ? { end: d.end } : {}) }, p);
    }
    case "list":
      return (v.value as (ComputedValue | null)[]).map((x) => computedText(x, p)).filter(Boolean).join(", ");
    case "person":
      return (v.value as { name: string }).name;
    case "page":
      return (v.value as { title: string }).title || t("Untitled");
    case "error":
      return String(v.value);
  }
  return String(v.value ?? "");
}

export function rowValue(row: RowData, p: Property): unknown {
  if (p.type === "title") return row.title;
  return row.values[p.id];
}
