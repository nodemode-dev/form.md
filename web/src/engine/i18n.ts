import { loaders } from "virtual:i18n";

type Catalog = Record<string, string>;
type Params = Record<string, string | number>;

let lang = "en";
let catalog: Catalog = {};
let rules = new Intl.PluralRules("en");
const listeners = new Set<() => void>();

export const LANGUAGES: ReadonlyArray<readonly [string, string]> = [
  ["en", "English"], ["es", "Español (España)"], ["es-MX", "Español (México)"], ["pt-BR", "Português (Brasil)"],
  ["fr", "Français"], ["fr-CA", "Français (Canada)"], ["de", "Deutsch"], ["it", "Italiano"], ["nl", "Nederlands"],
  ["da", "Dansk"], ["fi", "Suomi"], ["nb", "Norsk bokmål"], ["sv", "Svenska"], ["ru", "Русский"], ["pl", "Polski"],
  ["cs", "Čeština"], ["hu", "Magyar"], ["uk", "Українська"], ["tr", "Türkçe"], ["zh-Hans", "简体中文"], ["zh-Hant", "繁體中文"],
  ["ja", "日本語"], ["ko", "한국어"], ["id", "Bahasa Indonesia"], ["th", "ไทย"], ["vi", "Tiếng Việt"],
];

export const SUPPORTED = LANGUAGES.map(([code]) => code);

const ALIASES: Record<string, string> = {
  zh: "zh-Hans", "zh-cn": "zh-Hans", "zh-sg": "zh-Hans", "zh-tw": "zh-Hant", "zh-hk": "zh-Hant", "zh-mo": "zh-Hant",
  pt: "pt-BR", "pt-pt": "pt-BR", "es-419": "es-MX", "es-us": "es-MX", no: "nb", nn: "nb",
};

export function currentLang(): string {
  return lang;
}

export function resolveLang(pref: string, fallback: string): string {
  for (const candidate of [pref, fallback]) {
    const tag = (candidate ?? "").trim().replace(/_/g, "-").toLowerCase();
    if (!tag) continue;
    const exact = SUPPORTED.find((c) => c.toLowerCase() === tag);
    if (exact) return exact;
    const prefix = tag.split("-").slice(0, 2).join("-");
    if (ALIASES[tag]) return ALIASES[tag];
    if (ALIASES[prefix]) return ALIASES[prefix];
    if (tag.startsWith("zh-hant")) return "zh-Hant";
    if (tag.startsWith("zh-hans")) return "zh-Hans";
    const base = tag.split("-")[0] ?? "";
    if (ALIASES[base]) return ALIASES[base];
    const hit = SUPPORTED.find((c) => c.toLowerCase() === base);
    if (hit) return hit;
  }
  return "en";
}

export async function setLang(next: string): Promise<void> {
  if (next === lang && (next === "en" || Object.keys(catalog).length > 0)) return;
  const load = next === "en" ? undefined : loaders[next];
  const found = load !== undefined;
  catalog = load ? { ...(await load()).default } : {};
  lang = found ? next : "en";
  rules = new Intl.PluralRules(lang);
  for (const fn of listeners) fn();
}

export function onLangChange(fn: () => void): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

function fill(text: string, params?: Params): string {
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (m, k: string) => (k in params ? String(params[k]) : m));
}

export function t(source: string, params?: Params): string {
  return fill(catalog[source] ?? source, params);
}

export function plural(count: number, one: string, other: string, params: Params = {}): string {
  const key = `${one}|${other}`;
  const translated = catalog[key];
  const [tOne, tOther] = translated ? (translated.split("|") as [string, string]) : [one, other];
  const form = rules.select(count) === "one" ? tOne : tOther;
  return fill(form, { count, ...params });
}
