import { currentLang } from "./i18n";
import type { Prefs } from "./types";

let prefs: Partial<Prefs> = {};

export function setFormatPrefs(next: Partial<Prefs>): void {
  prefs = next;
}

function canonical(tag: string, fallback: string): string {
  try {
    return Intl.getCanonicalLocales(tag.split("@")[0] ?? "")[0] ?? fallback;
  } catch {
    return fallback;
  }
}

export function locale(): string {
  const region = prefs.region ?? "";
  const lang = currentLang();
  if (!region) return lang === "en" && typeof navigator !== "undefined" ? canonical(navigator.language || "en", "en") : lang;
  return canonical(region.includes("-") ? region : `${lang}-${region}`, lang);
}

function zone(): string | undefined {
  return prefs.time_zone || undefined;
}

function cycle(): "h12" | "h23" | undefined {
  return prefs.hour_cycle || undefined;
}

export function formatDateTime(ms: number): string {
  if (!ms) return "";
  const opts: Intl.DateTimeFormatOptions = { dateStyle: "medium", timeStyle: "short" };
  const tz = zone();
  if (tz) opts.timeZone = tz;
  const hc = cycle();
  if (hc) opts.hourCycle = hc;
  return new Intl.DateTimeFormat(locale(), opts).format(ms);
}

export function formatDate(ms: number): string {
  if (!ms) return "";
  const opts: Intl.DateTimeFormatOptions = { dateStyle: "medium" };
  const tz = zone();
  if (tz) opts.timeZone = tz;
  return new Intl.DateTimeFormat(locale(), opts).format(ms);
}

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 365 * 86400],
  ["month", 30 * 86400],
  ["week", 7 * 86400],
  ["day", 86400],
  ["hour", 3600],
  ["minute", 60],
];

export function formatRelative(ms: number, now = Date.now()): string {
  if (!ms) return "";
  const diff = (ms - now) / 1000;
  const rtf = new Intl.RelativeTimeFormat(locale(), { numeric: "auto" });
  for (const [unit, secs] of UNITS) {
    if (Math.abs(diff) >= secs) return rtf.format(Math.round(diff / secs), unit);
  }
  return rtf.format(0, "minute");
}

export function weekStart(): number {
  const pref = prefs.week_start;
  if (pref === "sunday") return 0;
  if (pref === "saturday") return 6;
  if (pref === "monday") return 1;
  try {
    const info = (new Intl.Locale(locale()) as Intl.Locale & { getWeekInfo?: () => { firstDay: number } }).getWeekInfo?.();
    if (info) return info.firstDay % 7;
  } catch {
    return 1;
  }
  return 1;
}
