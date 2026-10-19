export type ThemePref = "dark" | "light" | "auto";
export type SidebarMode = "expanded" | "collapsed";

export interface KeyValueStore {
  get(key: string): string | null;
  set(key: string, value: string): void;
}

export const PREF_KEYS = {
  theme: "formmd.theme",
  sidebar: "formmd.sidebar",
} as const;

export function readThemePref(store: KeyValueStore): ThemePref {
  const v = store.get(PREF_KEYS.theme);
  return v === "light" || v === "auto" || v === "dark" ? v : "auto";
}

export function readSidebar(store: KeyValueStore): SidebarMode {
  return store.get(PREF_KEYS.sidebar) === "collapsed" ? "collapsed" : "expanded";
}
