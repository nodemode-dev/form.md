import { type AuthState, type Boot, createEngine, type Engine } from "./engine";
import { ApiError } from "./engine/api";
import { createClient } from "./engine/client";
import { plural, resolveLang, setLang, t } from "./engine/i18n";
import account from "./extensions/account";
import blocks from "./extensions/blocks";
import database from "./extensions/database";
import { loadDatabase } from "./extensions/database/load";
import pagesExtension from "./extensions/pages";
import { activateAll, register } from "./extensions/registry";
import harness from "./extensions/harness";
import writing from "./extensions/writing";
import shell from "./extensions/shell";
import { mountUI, platformAdapter, storageAdapter } from "./ui/app";
import { mountAuth } from "./ui/auth";
import { mountCommentsPane } from "./ui/comments-pane";
import { mountInboxView } from "./ui/inbox-view";
import { listenForPushClicks } from "./ui/push";
import { errorMessage } from "./ui/errors";
import { prefetchPageDialogs } from "./ui/page-actions";
import { mountPageView } from "./ui/page-view";
import { mountHome, mountRoutes } from "./ui/routes";
import { mountSession } from "./ui/session";
import { toast } from "./ui/toast";
import { mountTopbar } from "./ui/topbar";
import { mountTrashView } from "./ui/trash-view";
import { mountTree } from "./ui/tree";
import { wipeLocalData } from "./engine/doc/docs";

const LAST_USER_KEY = "formmd.lastUser";

const AUTH_STATES: readonly AuthState[] = ["pending", "setup", "signed-out", "signup", "invite", "app", "form", "consent"];

function readBoot(): Boot {
  try {
    return (JSON.parse(document.getElementById("boot")?.textContent ?? "null") as Boot | null) ?? {};
  } catch {
    return {};
  }
}

function readAuth(): AuthState {
  const v = document.documentElement.dataset.auth as AuthState | undefined;
  return v && AUTH_STATES.includes(v) ? v : "pending";
}

async function devBoot(fetchFn: typeof fetch): Promise<{ auth: AuthState; boot: Boot }> {
  const api = createClient(fetchFn);
  const root = document.documentElement;
  const path = location.pathname;
  if (path.startsWith("/form/")) return { auth: "form", boot: {} };
  const policy = await api.auth.policy();
  if (policy.setup_required) return { auth: "setup", boot: { setup: true } };
  const signup = policy.signup === "invite" ? "closed" : "open";
  root.dataset.signup = signup;
  root.dataset.sso = policy.sso.map((p) => p.id).join(" ");
  const boot: Boot = { sso: policy.sso, signup, instance_name: policy.instance_name };
  try {
    boot.me = await api.me.get();
    return { auth: path.startsWith("/invite/") ? "invite" : "app", boot };
  } catch {
    if (path.startsWith("/invite/")) return { auth: "invite", boot };
    return { auth: path === "/signup" && signup === "open" ? "signup" : "signed-out", boot };
  }
}

function guardSession(base: typeof fetch, engine: () => Engine | null): typeof fetch {
  let leaving = false;
  return async (input, init) => {
    const res = await base(input, init);
    if (res.status === 401 && !leaving && engine()?.state.get("auth") === "app") {
      const body = (await res
        .clone()
        .json()
        .catch(() => null)) as { error?: { code?: string } } | null;
      if (body?.error?.code === "unauthorized") {
        leaving = true;
        const next = `/login?next=${encodeURIComponent(location.pathname + location.search)}`;
        const clear = typeof caches === "undefined" ? Promise.resolve([]) : Promise.all(["formmd-shell", "formmd-api", "formmd-files"].map((n) => caches.delete(n).catch(() => false)));
        void clear.finally(() => location.assign(next));
      }
    }
    return res;
  };
}

async function start(): Promise<void> {
  let engine: Engine | null = null;
  const fetchFn = guardSession(window.fetch.bind(window), () => engine);
  let auth = readAuth();
  let boot = readBoot();
  if (auth === "pending") ({ auth, boot } = await devBoot(fetchFn));

  await setLang(resolveLang(boot.me?.user.prefs.lang ?? "", document.documentElement.lang || navigator.language));

  const uid = boot.me?.user.id ?? "";
  if (uid) {
    let last: string | null = null;
    try {
      last = localStorage.getItem(LAST_USER_KEY);
      if (last && last !== uid) for (const k of ["formmd.rowOutbox", "formmd.rowCreates", "formmd.rowDeletes"]) localStorage.removeItem(k);
    } catch {
      last = null;
    }
    if (last && last !== uid) await wipeLocalData();
    try {
      localStorage.setItem(LAST_USER_KEY, uid);
    } catch {
      last = null;
    }
  }

  engine = createEngine({
    storage: storageAdapter(),
    fetch: fetchFn,
    platform: platformAdapter(),
    now: () => performance.now(),
    boot,
    auth,
  });

  register(shell);
  register(account);
  register(pagesExtension);
  register(blocks);
  register(database);
  register(harness);
  register(writing);
  activateAll(engine);
  const commands = engine.commands;
  (window as unknown as { __formmdRun?: (id: string) => boolean }).__formmdRun = (id: string) => commands.run(id);
  if (auth === "form" || boot.collection || boot.row || document.querySelector("[data-ref='doc-body'] .dbblock")) await loadDatabase().catch(() => undefined);
  mountUI(engine);
  if (auth === "form") {
    document.documentElement.dataset.auth = "form";
    void loadDatabase().then((m) => m.mountPublicForm(engine as Engine), (e: unknown) => toast({ id: "error", tone: "danger", message: errorMessage(e) }));
    return;
  }
  mountAuth(engine);
  if (auth === "app") {
    const live = engine;
    engine.onError((e) => {
      if (e instanceof ApiError && e.code === "network") void live.health.check();
      else toast({ id: "error", tone: "danger", message: errorMessage(e) });
    });
    mountRoutes(engine);
    mountSession(engine);
    mountTree(engine);
    mountTopbar(engine);
    mountPageView(engine);
    mountCommentsPane(engine);
    mountInboxView(engine);
    listenForPushClicks(engine);
    let library: Promise<void> | null = null;
    const loadLibrary = () => (library ??= import("./ui/library-view").then((m) => m.mountLibraryView(live)));
    if (engine.state.get("route").name === "library") void loadLibrary();
    else {
      engine.state.watch("route", (r) => {
        if (r.name === "library") void loadLibrary();
      });
      setTimeout(() => (window.requestIdleCallback ?? ((f: () => void) => f()))(() => void loadLibrary()), 3000);
    }
    setTimeout(() => (window.requestIdleCallback ?? ((f: () => void) => f()))(() => void import("./ui/pickers")), 3000);
    mountTrashView(engine);
    mountHome(engine);
    const app = engine;
    engine.collections.onQueued = (n) =>
      toast({ id: "row-outbox", message: n ? plural(n, "{count} database change is saved on this device and will be sent when the connection is back", "{count} database changes are saved on this device and will be sent when the connection is back") : t("Database changes made offline were sent") });
    if (navigator.onLine && engine.collections.queued()) void engine.collections.replay();
    window.addEventListener("online", () => {
      if (engine.collections.queued()) void engine.collections.replay();
    });
    engine.state.watch("live", (s) => {
      if (s === "live" && app.collections.queued()) void app.collections.replay();
    });
    setTimeout(() => void import("./ui/offline").then((m) => m.startOffline(app)).catch(() => undefined), 2500);
    if (matchMedia("(display-mode: standalone)").matches && "ontouchstart" in window) void import("./ui/pull").then((m) => m.mountPullToRefresh(app));
    const settingsOverlays = new Set<string | null>(["account", "workspace", "new-workspace", "admin"]);
    let settingsJob: Promise<void> | null = null;
    const ensureSettings = () =>
      (settingsJob ??= import("./ui/settings-entry")
        .then((m) => m.mountSettings(app))
        .catch((e: unknown) => {
          settingsJob = null;
          if (settingsOverlays.has(app.state.get("overlay"))) app.state.set("overlay", null);
          app.reportError(e);
        }));
    engine.state.watch("overlay", (v) => void (settingsOverlays.has(v) && ensureSettings()));
    const rootEl = document.documentElement;
    const syncDev = () => {
      try {
        rootEl.toggleAttribute("data-dev", localStorage.getItem("fm.dev") === "1");
      } catch {
        rootEl.removeAttribute("data-dev");
      }
    };
    syncDev();
    window.addEventListener("storage", (e) => (e.key === null || e.key === "fm.dev") && syncDev());
    if (boot.me?.user.role === "owner") {
      rootEl.setAttribute("data-owner", "");
      fetchFn("/api/claude")
        .then((r) => r.json())
        .then((j: { enabled?: boolean }) => j.enabled === false && rootEl.setAttribute("data-harness", "off"))
        .catch(() => undefined);
    }
    const agentBtn = document.querySelector<HTMLButtonElement>("[data-ref='act-agent']");
    let agentJob: Promise<{ toggle(): void }> | null = null;
    const ensureAgent = () => {
      if (!agentBtn) return Promise.reject(new Error("no agent button"));
      agentJob ??= import("./ui/agent-pane").then((m) => m.mountAgentPane(app, agentBtn));
      agentJob.catch(() => (agentJob = null));
      return agentJob;
    };
    const toggleAgent = () => void ensureAgent().then((p) => p.toggle(), app.reportError);
    agentBtn?.addEventListener("click", toggleAgent);
    engine.commands.register({ id: "agent.toggle", label: () => t("Show or hide the agent"), group: () => t("AI"), description: "Shows or hides the agent pane in this window. Does not send anything to the agent.", mutates: false, run: toggleAgent });
    engine.keymap.register({ id: "agent.toggle", command: "agent.toggle", chords: ["mod+shift+j"], scope: "global", whileTyping: true });
    const loadSearch = () => import("./ui/search").then((m) => m.mountSearch(app));
    let searchJob: Promise<void> | null = null;
    const ensureSearch = () => (searchJob ??= loadSearch().catch((e: unknown) => ((searchJob = null), app.reportError(e))));
    engine.state.watch("overlay", (v) => void (v === "search" && ensureSearch()));
    const prefetch = () => {
      void ensureSearch();
      void ensureSettings();
      prefetchPageDialogs();
    };
    setTimeout(() => ("requestIdleCallback" in window ? requestIdleCallback(prefetch) : prefetch()), 2500);
    engine.start();
  }
}

void start();
