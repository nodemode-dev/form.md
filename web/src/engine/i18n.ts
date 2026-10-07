type Catalog = Record<string, string>;
type Params = Record<string, string | number>;

const loaders = import.meta.glob<Catalog>("../i18n/*.json", { import: "default" });

let lang = "en";
let catalog: Catalog = {};
let rules = new Intl.PluralRules("en");
const listeners = new Set<() => void>();

export const SUPPORTED = ["en", "de"] as const;

export function currentLang(): string {
  return lang;
}

export function resolveLang(pref: string, fallback: string): string {
  const pick = (tag: string) => tag.toLowerCase().split("-")[0] ?? "";
  for (const candidate of [pref, fallback]) {
    const base = pick(candidate);
    if ((SUPPORTED as readonly string[]).includes(base)) return base;
  }
  return "en";
}

export async function setLang(next: string): Promise<void> {
  if (next === lang && (next === "en" || Object.keys(catalog).length > 0)) return;
  const load = loaders[`../i18n/${next}.json`];
  catalog = next !== "en" && load ? await load() : {};
  lang = next !== "en" && load ? next : "en";
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
