import { type FetchFn, request } from "./api";
import { type Client, createClient } from "./client";
import { CollectionStore } from "./collections";
import { CommentStore } from "./comments";
import { InboxStore } from "./inbox";
import { Commands } from "./commands";
import { DocStore } from "./doc/docs";
import { DocSync } from "./doc/sync";
import { setFormatPrefs } from "./format";
import type { AccessLevel, Boot, LiveEvent, Me, Page, PageDetail, Workspace } from "./types";
import { type Health, HealthMonitor, initialHealth } from "./health";
import { Keymap } from "./keymap";
import { Live, type LiveStatus } from "./live";
import { PageStore } from "./pages";
import { type KeyValueStore, PREF_KEYS, readSidebar, readThemePref, type SidebarMode, type ThemePref } from "./prefs";
import { closeTab, cycleTab, moveTab, type NavState, openNewTab, openTab, parseRoute, type Route, routePath, sameRoute } from "./router";
import { State } from "./state";

export type Overlay = null | "shortcuts" | "account" | "admin" | "workspace" | "new-workspace" | "search";
export type AuthState = "pending" | "setup" | "signed-out" | "signup" | "invite" | "app" | "form" | "consent";

export interface Platform {
  mac: boolean;
  isNarrow(): boolean;
  prefersLight(): boolean;
  navigate(path: string): void;
  location(): { path: string; search: string; state: unknown };
  pushHistory(state: unknown, url: string): void;
  replaceHistory(state: unknown, url: string): void;
  eventSource(url: string): EventSource;
  webSocket(path: string): WebSocket;
  persistDocs: boolean;
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
  route: Route;
  tabs: string[];
  detail: PageDetail | null;
  live: LiveStatus;
  newVersion: string;
  tagFilter: string;
}

export interface OpenOptions {
  newTab?: boolean;
  replace?: boolean;
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
  readonly pages: PageStore;
  readonly docs: DocStore;
  readonly collections: CollectionStore;
  readonly comments: CommentStore;
  readonly inbox: InboxStore;
  resolvedTheme(): "dark" | "light";
  workspace(): Workspace | null;
  workspaceById(id: string): Workspace | null;
  canWrite(workspaceId?: string): boolean;
  can(pageId: string, need: AccessLevel): boolean;
  setMe(me: Me | null): void;
  refreshMe(): Promise<void>;
  openOverlay(overlay: Exclude<Overlay, null>, tab?: string): void;
  navigate(route: Route, options?: OpenOptions): void;
  openPage(id: string, options?: OpenOptions): void;
  closeTab(id: string): void;
  cycleTab(d: 1 | -1): boolean;
  moveTab(id: string, before: string): void;
  restoreNav(state: unknown): void;
  reportError(e: unknown): void;
  onError(fn: (e: unknown) => void): void;
  start(): void;
  reconnect(): void;
}

const LEVELS: readonly AccessLevel[] = ["view", "comment", "create", "content", "edit", "full"];
const WORKSPACE_KEY = "formmd.workspace";
const TABS_KEY = "formmd.tabs";

function readTabs(storage: KeyValueStore): string[] {
  try {
    const v = JSON.parse(storage.get(TABS_KEY) ?? "[]") as unknown;
    return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string" && /^[a-z2-7]{26}$/.test(x)).slice(0, 30) : [];
  } catch {
    return [];
  }
}

function isNavState(v: unknown): v is NavState {
  const s = v as NavState | null;
  return !!s && Array.isArray(s.tabs) && typeof s.route === "object" && s.route !== null && typeof (s.route as { name?: unknown }).name === "string";
}

function pickWorkspace(me: Me | null, preferred: string): string {
  if (!me || me.workspaces.length === 0) return "";
  if (me.workspaces.some((w) => w.id === preferred)) return preferred;
  return (me.workspaces.find((w) => !w.personal) ?? me.workspaces[0])?.id ?? "";
}

export function createEngine(deps: EngineDeps): Engine {
  const here = deps.platform.location();
  const startRoute = deps.auth === "app" ? parseRoute(here.path, here.search) : ({ name: "home" } as Route);
  const savedNav = isNavState(here.state) ? here.state : null;
  const startTabs = savedNav ? savedNav.tabs : readTabs(deps.storage);
  const bootPage = deps.boot.page && startRoute.name === "page" && deps.boot.page.id === startRoute.id ? deps.boot.page : null;
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
    workspaceId: pickWorkspace(deps.boot.me ?? null, bootPage?.workspace_id || deps.boot.workspace_id || deps.storage.get(WORKSPACE_KEY) || ""),
    route: startRoute,
    tabs: startRoute.name === "page" && !startTabs.includes(startRoute.id) ? [...startTabs, startRoute.id] : startTabs,
    detail: bootPage,
    live: "connecting",
    newVersion: "",
    tagFilter: "",
  });
  state.watch("themePref", (v) => deps.storage.set(PREF_KEYS.theme, v));
  state.watch("sidebar", (v) => deps.storage.set(PREF_KEYS.sidebar, v));
  state.watch("workspaceId", (v) => v && deps.storage.set(WORKSPACE_KEY, v));
  state.watch("tabs", (v) => deps.storage.set(TABS_KEY, JSON.stringify(v)));
  if (deps.boot.me) setFormatPrefs(deps.boot.me.user.prefs);
  const api = createClient(deps.fetch);
  const setMe = (me: Me | null) => {
    const before = new Map((state.get("me")?.workspaces ?? []).map((w) => [w.id, w.role]));
    state.set("me", me);
    if (me) setFormatPrefs(me.user.prefs);
    state.set("workspaceId", pickWorkspace(me, state.get("workspaceId")));
    const moved = (me?.workspaces ?? []).filter((w) => before.has(w.id) && before.get(w.id) !== w.role).map((w) => w.id);
    for (const ws of moved) if (pages.isLoaded(ws)) void pages.load(ws, true).catch(reportError);
    const d = state.get("detail");
    if (d && moved.includes(d.workspace_id)) void loadDetail(d.id);
  };

  const commands = new Commands();
  const keymap = new Keymap(commands, deps.platform.mac);
  commands.keysOf = (id) => {
    const keys = keymap.chordFor(id);
    return keys ? [keys.join("+")] : [];
  };
  const health = new HealthMonitor(
    deps.fetch,
    {
      get: () => state.get("health"),
      set: (h) => state.set("health", h),
      setChecking: (c) => state.set("checking", c),
    },
    deps.now,
  );

  const errorHandlers = new Set<(e: unknown) => void>();
  const reportError = (e: unknown) => {
    for (const fn of errorHandlers) fn(e);
  };
  const pages = new PageStore(api, reportError);
  const collections = new CollectionStore(api, reportError);
  state.watch("me", (me) => collections.setOwner(me?.user.id ?? ""), { immediate: true });
  const comments = new CommentStore(api, reportError);
  const inbox = new InboxStore(api, reportError, () => typeof document !== "undefined" && document.visibilityState === "visible", deps.boot.inbox_unread ?? 0);
  if (bootPage && deps.boot.comments) comments.seed(bootPage.id, deps.boot.comments);
  if (deps.boot.collection) collections.seed(deps.boot.collection);
  if (deps.boot.query) collections.seedQuery(deps.boot.query);
  if (deps.boot.row) collections.seed(deps.boot.row.collection);
  if (bootPage?.row) pages.upsert(bootPage);
  const docs = new DocStore({
    sync: new DocSync({ url: "/sync", open: (url) => deps.platform.webSocket(url) }),
    user: () => state.get("me")?.user.id ?? "",
    persist: deps.platform.persistDocs,
  });
  const workspaceById = (id: string) => state.get("me")?.workspaces.find((w) => w.id === id) ?? null;
  const refreshMe = async () => setMe(await api.me.get());

  const lastPageKey = (ws: string) => `formmd.last.${ws}`;
  const commit = (next: NavState, replace = false) => {
    const prev = state.get("route");
    const snapshot: NavState = { tabs: next.tabs, route: next.route };
    const url = routePath(next.route);
    if (replace || sameRoute(prev, next.route)) deps.platform.replaceHistory(snapshot, url);
    else deps.platform.pushHistory(snapshot, url);
    state.set("tabs", next.tabs);
    state.set("route", next.route);
  };
  const nav = (): NavState => ({ tabs: state.get("tabs"), route: state.get("route") });

  const landing = () => {
    const ws = state.get("workspaceId");
    if (state.get("route").name !== "home" || !ws || !pages.isLoaded(ws)) return;
    const live = (id: string | null) => (id ? pages.get(id) : undefined);
    const remembered = live(deps.storage.get(lastPageKey(ws)));
    const fromTab = state.get("tabs").map((id) => pages.get(id)).find((p) => p?.workspace_id === ws);
    const target = (remembered?.workspace_id === ws ? remembered : undefined) ?? fromTab ?? pages.children(ws, "")[0];
    if (target) commit(openTab(nav(), target.id), true);
  };

  let detailSeq = 0;
  const loadDetail = async (id: string) => {
    const mine = ++detailSeq;
    try {
      const r = await api.pages.get(id);
      if (mine !== detailSeq) return;
      if (r.page.row) pages.upsert(r.page);
      state.set("detail", r.page);
      if (workspaceById(r.page.workspace_id) && r.page.workspace_id !== state.get("workspaceId")) state.set("workspaceId", r.page.workspace_id);
    } catch (e) {
      if (mine !== detailSeq) return;
      const code = (e as { code?: string }).code;
      if (code === "not_found" || code === "page_trashed") commit(closeTab(nav(), id), true);
      reportError(e);
    }
  };

  let visitTimer: ReturnType<typeof setTimeout> | undefined;
  state.watch("route", (r) => {
    clearTimeout(visitTimer);
    if (r.name !== "page") {
      detailSeq++;
      if (state.get("detail")) state.set("detail", null);
      landing();
      return;
    }
    const known = pages.get(r.id);
    if (known && known.workspace_id !== state.get("workspaceId") && workspaceById(known.workspace_id)) state.set("workspaceId", known.workspace_id);
    const d = state.get("detail");
    if (d && d.id !== r.id) state.set("detail", null);
    void loadDetail(r.id);
    visitTimer = setTimeout(() => void api.pages.visit(r.id).catch(() => undefined), 800);
    const ws = known?.workspace_id ?? state.get("detail")?.workspace_id;
    if (ws) deps.storage.set(lastPageKey(ws), r.id);
  });

  state.watch("workspaceId", (ws) => {
    if (!ws) return;
    state.set("tagFilter", "");
    void pages.load(ws).then(landing, reportError);
    void pages.loadTags(ws).catch(reportError);
    const r = state.get("route");
    if (r.name === "page") {
      const p = pages.get(r.id) ?? state.get("detail");
      if (p && p.workspace_id !== ws) commit({ tabs: state.get("tabs"), route: { name: "home" } });
    }
  });

  pages.subscribe((c) => {
    if (c.kind === "remove") {
      let next = nav();
      let changed = false;
      for (const p of c.pages) {
        if (!next.tabs.includes(p.id) && !(next.route.name === "page" && next.route.id === p.id)) continue;
        next = next.tabs.includes(p.id) ? closeTab(next, p.id) : { tabs: next.tabs, route: { name: "home" } };
        changed = true;
      }
      if (changed) commit(next, true);
    } else if (c.kind === "loaded") {
      landing();
    } else if (c.kind === "upsert") {
      const d = state.get("detail");
      if (d && d.id === c.page.id) state.set("detail", { ...d, ...c.page });
    }
  });

  let firstVersion = "";
  let buffering = false;
  let queue: LiveEvent[] = [];
  let needsLoad = false;
  const dispatch = (e: LiveEvent) => {
    if (e.type === "command.run") {
      const answer = (body: { result?: unknown; error?: string }) => request(deps.fetch, "POST", "/api/commands/result", { call_id: e.call_id, ...body }).catch(reportError);
      void commands.invoke(e.id, e.params ?? {}).then(
        (result) => answer({ result }),
        (err: unknown) => answer({ error: err instanceof Error ? err.message : String(err) }),
      );
      return;
    }
    pages.apply(e);
    collections.apply(e);
    comments.apply(e);
    inbox.apply(e);
    if (e.type === "workspace" && (e.what === "members" || e.what === "settings" || e.what === "rules" || e.what === "deleted")) void refreshMe().catch(reportError);
    if (e.type === "user" && (e.what === "workspaces" || e.what === "break_glass" || e.what === "shares")) void refreshMe().catch(reportError);
    if (e.type === "workspace" && e.what === "shares") void refreshMe().catch(reportError);
    const d = state.get("detail");
    if (d && (e.type === "user" || e.type === "workspace") && (e.what === "shares" || e.what === "teamspaces" || e.what === "pages")) void loadDetail(d.id);
  };
  const resync = async () => {
    buffering = true;
    try {
      const ws = state.get("workspaceId");
      const jobs: Promise<unknown>[] = [pages.reloadAll(), refreshMe(), pages.loadFavorites()];
      collections.resync();
      comments.resync();
      jobs.push(inbox.resync());
      if (ws && !pages.isLoaded(ws)) jobs.push(pages.load(ws), pages.loadTags(ws));
      const r = state.get("route");
      if (r.name === "page") jobs.push(loadDetail(r.id));
      await Promise.all(jobs.map((j) => j.catch(reportError)));
    } finally {
      buffering = false;
      const q = queue;
      queue = [];
      for (const e of q) dispatch(e);
    }
  };
  const live = new Live({
    url: "/api/events",
    open: (url) => deps.platform.eventSource(url),
    onStatus: (s) => state.set("live", s),
    onClosed: () => void api.me.get().catch(() => undefined),
    onEvent: (e) => {
      if (e.type === "hello") {
        if (!firstVersion) firstVersion = e.version;
        else if (e.version !== firstVersion) state.set("newVersion", e.version);
        if (e.reset || needsLoad) {
          needsLoad = false;
          void resync();
        }
        return;
      }
      if (e.type === "ping") return;
      if (buffering) queue.push(e);
      else dispatch(e);
    },
  });

  const openPage = (id: string, options: OpenOptions = {}) => {
    if (state.get("drawer")) state.set("drawer", false);
    commit(options.newTab ? openNewTab(nav(), id) : openTab(nav(), id), options.replace ?? false);
  };

  return {
    state,
    commands,
    keymap,
    health,
    platform: deps.platform,
    api,
    boot: deps.boot,
    pages,
    docs,
    collections,
    comments,
    inbox,
    resolvedTheme() {
      const pref = state.get("themePref");
      if (pref !== "auto") return pref;
      return deps.platform.prefersLight() ? "light" : "dark";
    },
    workspace() {
      return workspaceById(state.get("workspaceId"));
    },
    workspaceById,
    canWrite(workspaceId = state.get("workspaceId")) {
      const w = workspaceById(workspaceId);
      const me = state.get("me");
      return !!w && !w.break_glass && w.role !== "viewer" && w.role !== "guest" && me?.credential.scope !== "read";
    },
    can(pageId, need) {
      const d = state.get("detail");
      const p = pages.get(pageId);
      const level = d?.id === pageId ? d.level : p?.access;
      const ws = p?.workspace_id ?? (d?.id === pageId ? d.workspace_id : "");
      const w = workspaceById(ws);
      if (need !== "view" && (state.get("me")?.credential.scope === "read" || w?.break_glass)) return false;
      if (!level) return need === "view" || this.canWrite(ws);
      return LEVELS.indexOf(level) >= LEVELS.indexOf(need);
    },
    setMe,
    refreshMe,
    openOverlay(overlay, tab = "") {
      state.set("overlayTab", "");
      state.set("overlayTab", tab);
      state.set("overlay", overlay);
    },
    navigate(route, options = {}) {
      if (route.name === "page") return openPage(route.id, options);
      if (state.get("drawer")) state.set("drawer", false);
      commit({ tabs: state.get("tabs"), route }, options.replace ?? false);
    },
    openPage,
    closeTab(id) {
      commit(closeTab(nav(), id), true);
    },
    cycleTab(d) {
      const next = cycleTab(nav(), d);
      if (!next) return false;
      commit(next);
      return true;
    },
    moveTab(id, before) {
      commit(moveTab(nav(), id, before), true);
    },
    restoreNav(raw) {
      const here = deps.platform.location();
      const route = parseRoute(here.path, here.search);
      if (isNavState(raw)) {
        state.set("tabs", raw.tabs);
        state.set("route", route);
        return;
      }
      if (route.name === "page" && !state.get("tabs").includes(route.id)) state.set("tabs", [...state.get("tabs"), route.id]);
      state.set("route", route);
    },
    reportError,
    onError(fn) {
      errorHandlers.add(fn);
    },
    start() {
      health.start();
      if (state.get("auth") !== "app") return;
      const ws = state.get("workspaceId");
      const boot = deps.boot;
      if (boot.pages && ws && boot.workspace_id === ws) {
        if (boot.teamspaces) pages.seedTeams(ws, boot.teamspaces);
        pages.seed(ws, boot.pages);
        if (boot.favorites) pages.seedFavorites(boot.favorites);
        if (boot.tags) pages.seedTags(ws, boot.tags);
        live.start(boot.seq ?? 0);
      } else {
        needsLoad = true;
        live.start();
      }
      deps.platform.replaceHistory(nav(), routePath(state.get("route")));
      const r = state.get("route");
      if (r.name === "page") {
        void loadDetail(r.id);
        visitTimer = setTimeout(() => void api.pages.visit(r.id).catch(() => undefined), 800);
      } else {
        landing();
      }
    },
    reconnect() {
      if (state.get("live") !== "live") live.reconnectNow();
      docs.sync.reconnectNow();
    },
  };
}

export type { Route, LibraryView } from "./router";
export type { LiveStatus } from "./live";
export type { StoreChange } from "./pages";

export type { Health, HealthPhase } from "./health";
export type * from "./types";
export type { Scope } from "./keymap";
export type { ThemePref, SidebarMode } from "./prefs";
