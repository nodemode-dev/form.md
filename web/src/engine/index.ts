import type { FetchFn } from "./api";
import { type Client, createClient } from "./client";
import { Commands } from "./commands";
import { setFormatPrefs } from "./format";
import type { Boot, Me, Workspace } from "./types";
import { type Health, HealthMonitor, initialHealth } from "./health";
import { Keymap } from "./keymap";
import { type KeyValueStore, PREF_KEYS, readSidebar, readThemePref, type SidebarMode, type ThemePref } from "./prefs";
import { State } from "./state";

export type Overlay = null | "shortcuts" | "account" | "admin" | "workspace" | "new-workspace";
export type AuthState = "pending" | "setup" | "signed-out" | "signup" | "invite" | "app";

export interface Platform {
  mac: boolean;
  isNarrow(): boolean;
  prefersLight(): boolean;
  navigate(path: string): void;
}

export interface EngineState {
  themePref: ThemePref;
  sidebar: SidebarMode;
  drawer: boolean;
  overlay: Overlay;
  overlayTab: string;
  health: Health;
  checking: boolean;
  auth: AuthState;
  me: Me | null;
  workspaceId: string;
}

export interface EngineDeps {
  storage: KeyValueStore;
  fetch: FetchFn;
  platform: Platform;
  now: () => number;
  boot: Boot;
  auth: AuthState;
}

export interface Engine {
  readonly state: State<EngineState>;
  readonly commands: Commands;
  readonly keymap: Keymap;
  readonly health: HealthMonitor;
  readonly platform: Platform;
  readonly api: Client;
  readonly boot: Boot;
  resolvedTheme(): "dark" | "light";
  workspace(): Workspace | null;
  setMe(me: Me | null): void;
  refreshMe(): Promise<void>;
  openOverlay(overlay: Exclude<Overlay, null>, tab?: string): void;
  start(): void;
}

const WORKSPACE_KEY = "formmd.workspace";

function pickWorkspace(me: Me | null, preferred: string): string {
  if (!me || me.workspaces.length === 0) return "";
  if (me.workspaces.some((w) => w.id === preferred)) return preferred;
  return (me.workspaces.find((w) => !w.personal) ?? me.workspaces[0])?.id ?? "";
}

export function createEngine(deps: EngineDeps): Engine {
  const state = new State<EngineState>({
    themePref: readThemePref(deps.storage),
    sidebar: readSidebar(deps.storage),
    drawer: false,
    overlay: null,
    overlayTab: "",
    health: initialHealth,
    checking: false,
    auth: deps.auth,
    me: deps.boot.me ?? null,
    workspaceId: pickWorkspace(deps.boot.me ?? null, deps.boot.workspace_id || deps.storage.get(WORKSPACE_KEY) || ""),
  });
  state.watch("themePref", (v) => deps.storage.set(PREF_KEYS.theme, v));
  state.watch("sidebar", (v) => deps.storage.set(PREF_KEYS.sidebar, v));
  state.watch("workspaceId", (v) => v && deps.storage.set(WORKSPACE_KEY, v));
  if (deps.boot.me) setFormatPrefs(deps.boot.me.user.prefs);
  const api = createClient(deps.fetch);
  const setMe = (me: Me | null) => {
    state.set("me", me);
    if (me) setFormatPrefs(me.user.prefs);
    state.set("workspaceId", pickWorkspace(me, state.get("workspaceId")));
  };

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
    api,
    boot: deps.boot,
    resolvedTheme() {
      const pref = state.get("themePref");
      if (pref !== "auto") return pref;
      return deps.platform.prefersLight() ? "light" : "dark";
    },
    workspace() {
      const id = state.get("workspaceId");
      return state.get("me")?.workspaces.find((w) => w.id === id) ?? null;
    },
    setMe,
    async refreshMe() {
      setMe(await api.me.get());
    },
    openOverlay(overlay, tab = "") {
      state.set("overlayTab", "");
      state.set("overlayTab", tab);
      state.set("overlay", overlay);
    },
    start() {
      health.start();
    },
  };
}

export type { Health, HealthPhase } from "./health";
export type * from "./types";
export type { Scope } from "./keymap";
export type { ThemePref, SidebarMode } from "./prefs";
