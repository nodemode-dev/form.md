export type ThemePref = "dark" | "light" | "auto";
export type SidebarMode = "expanded" | "collapsed";
export type Density = "compact" | "standard" | "comfortable";
export type ContrastPref = "auto" | "standard" | "high";
export type StartupPref = "continue" | "home";

export interface Look {
  dark: string;
  light: string;
  accent: string;
  density: Density;
  contrast: ContrastPref;
  at: number;
}

export interface ServerLook extends Look {
  mode: ThemePref;
}

export const DARK_THEMES = ["dark", "studio", "night", "graphite", "midnight", "forest", "ember", "violet", "slate", "mono", "ocean", "rose", "terminal", "contrast"] as const;
export const LIGHT_THEMES = ["light", "paper", "sand", "sage", "mist", "contrast"] as const;
export const ACCENTS = ["#f97316", "#ef4444", "#ec4899", "#a855f7", "#6366f1", "#3b82f6", "#14b8a6", "#22c55e", "#eab308"] as const;
export const DENSITIES: readonly Density[] = ["compact", "standard", "comfortable"];
export const CONTRASTS: readonly ContrastPref[] = ["auto", "standard", "high"];
export const DEFAULT_LOOK: Look = { dark: "dark", light: "light", accent: "", density: "standard", contrast: "auto", at: 0 };

export interface KeyValueStore {
  get(key: string): string | null;
  set(key: string, value: string): void;
}

export const PREF_KEYS = {
  theme: "formmd.theme",
  sidebar: "formmd.sidebar",
  look: "formmd.look",
  startup: "formmd.startup",
} as const;

export function readThemePref(store: KeyValueStore): ThemePref {
  const v = store.get(PREF_KEYS.theme);
  return v === "light" || v === "auto" || v === "dark" ? v : "auto";
}

export function readSidebar(store: KeyValueStore): SidebarMode {
  return store.get(PREF_KEYS.sidebar) === "collapsed" ? "collapsed" : "expanded";
}

function pick<T extends string>(v: unknown, list: readonly T[], fallback: T): T {
  return list.includes(v as T) ? (v as T) : fallback;
}

export function normalizeLook(raw: unknown): Look {
  const r = (raw && typeof raw === "object" ? raw : {}) as Record<string, unknown>;
  const accent = typeof r.accent === "string" && /^#[0-9a-f]{6}$/i.test(r.accent) ? r.accent.toLowerCase() : "";
  const at = typeof r.at === "number" && Number.isFinite(r.at) && r.at > 0 ? Math.floor(r.at) : 0;
  return {
    dark: pick(r.dark, DARK_THEMES, "dark"),
    light: pick(r.light, LIGHT_THEMES, "light"),
    accent,
    density: pick(r.density, DENSITIES, "standard"),
    contrast: pick(r.contrast, CONTRASTS, "auto"),
    at,
  };
}

export function normalizeServerLook(raw: unknown): ServerLook | null {
  if (!raw || typeof raw !== "object") return null;
  const mode = (raw as { mode?: unknown }).mode;
  return { ...normalizeLook(raw), mode: mode === "light" || mode === "dark" ? mode : "auto" };
}

export function readLook(store: KeyValueStore): Look {
  try {
    return normalizeLook(JSON.parse(store.get(PREF_KEYS.look) ?? "null"));
  } catch {
    return { ...DEFAULT_LOOK };
  }
}

export function writeLook(store: KeyValueStore, look: Look): void {
  store.set(PREF_KEYS.look, JSON.stringify(look));
}

export function accentTone(hex: string): "light" | "dark" {
  const n = parseInt(hex.slice(1), 16);
  return 0.2126 * (n >> 16) + 0.7152 * ((n >> 8) & 255) + 0.0722 * (n & 255) > 132 ? "light" : "dark";
}

export function readStartup(store: KeyValueStore): StartupPref {
  return store.get(PREF_KEYS.startup) === "home" ? "home" : "continue";
}

export function writeStartup(store: KeyValueStore, value: StartupPref): void {
  store.set(PREF_KEYS.startup, value);
}

export function startOnHome(store: KeyValueStore): boolean {
  return readStartup(store) === "home";
}
