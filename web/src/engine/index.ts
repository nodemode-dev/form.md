import type { FetchFn } from "./api";
import { Commands } from "./commands";
import { type Health, HealthMonitor, initialHealth } from "./health";
import { Keymap } from "./keymap";
import { type KeyValueStore, PREF_KEYS, readSidebar, readThemePref, type SidebarMode, type ThemePref } from "./prefs";
import { State } from "./state";

export type Overlay = null | "shortcuts";

export interface Platform {
  mac: boolean;
  isNarrow(): boolean;
  prefersLight(): boolean;
}

export interface EngineState {
  themePref: ThemePref;
  sidebar: SidebarMode;
  drawer: boolean;
  overlay: Overlay;
  health: Health;
  checking: boolean;
}

export interface EngineDeps {
  storage: KeyValueStore;
  fetch: FetchFn;
  platform: Platform;
  now: () => number;
}

export interface Engine {
  readonly state: State<EngineState>;
  readonly commands: Commands;
  readonly keymap: Keymap;
  readonly health: HealthMonitor;
  readonly platform: Platform;
  resolvedTheme(): "dark" | "light";
  start(): void;
}

export function createEngine(deps: EngineDeps): Engine {
  const state = new State<EngineState>({
    themePref: readThemePref(deps.storage),
    sidebar: readSidebar(deps.storage),
    drawer: false,
    overlay: null,
    health: initialHealth,
    checking: false,
  });
  state.watch("themePref", (v) => deps.storage.set(PREF_KEYS.theme, v));
  state.watch("sidebar", (v) => deps.storage.set(PREF_KEYS.sidebar, v));

  const commands = new Commands();
  const keymap = new Keymap(commands, deps.platform.mac);
  const health = new HealthMonitor(
    deps.fetch,
    {
      get: () => state.get("health"),
      set: (h) => state.set("health", h),
      setChecking: (c) => state.set("checking", c),
    },
    deps.now,
  );

  return {
    state,
    commands,
    keymap,
    health,
    platform: deps.platform,
    resolvedTheme() {
      const pref = state.get("themePref");
      if (pref !== "auto") return pref;
      return deps.platform.prefersLight() ? "light" : "dark";
    },
    start() {
      health.start();
    },
  };
}

export type { Health, HealthPhase } from "./health";
export type { Scope } from "./keymap";
export type { ThemePref, SidebarMode } from "./prefs";
