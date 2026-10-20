import type { Engine, Platform } from "../engine";
import type { KeyValueStore } from "../engine/prefs";
import { mountKeyboard } from "./keyboard";
import { mountShortcutsSheet } from "./shortcuts-sheet";
import { mountSidebar, NARROW_QUERY } from "./sidebar";
import { mountStatus } from "./status";
import { mountTheme } from "./theme";
import { mountTooltip } from "./tooltip";

const NAV_TOP_KEY = "formmd.navTop";
const navOf = (s: unknown): number => {
  const n = (s as { nav?: unknown } | null)?.nav;
  return typeof n === "number" ? n : 0;
};
let navAt = navOf(history.state);
let navTop = navAt;
try {
  if (history.state) navTop = Math.max(navAt, Number(sessionStorage.getItem(NAV_TOP_KEY)) || 0);
} catch {
  navTop = navAt;
}

export function navSpan(): { back: boolean; forward: boolean } {
  return { back: navAt > 0, forward: navAt < navTop };
}

export function platformAdapter(): Platform {
  const narrow = matchMedia(NARROW_QUERY);
  const light = matchMedia("(prefers-color-scheme: light)");
  return {
    mac: /Mac|iPhone|iPad|iPod/.test(navigator.userAgent),
    isNarrow: () => narrow.matches,
    prefersLight: () => light.matches,
    navigate: (path) => location.assign(path),
    location: () => ({ path: location.pathname, search: location.search, state: history.state as unknown }),
    pushHistory: (state, url) => {
      navTop = ++navAt;
      try {
        sessionStorage.setItem(NAV_TOP_KEY, String(navTop));
      } catch {
        navTop = navAt;
      }
      history.pushState({ ...(state as object), nav: navAt }, "", url);
    },
    replaceHistory: (state, url) => history.replaceState({ ...(state as object), nav: navAt }, "", url),
    eventSource: (url) => new EventSource(url, { withCredentials: true }),
    webSocket: (path) => new WebSocket(`${location.protocol === "https:" ? "wss" : "ws"}://${location.host}${path}`),
    persistDocs: typeof indexedDB !== "undefined",
  };
}

export function storageAdapter(): KeyValueStore {
  return {
    get(key) {
      try {
        return localStorage.getItem(key);
      } catch {
        return null;
      }
    },
    set(key, value) {
      try {
        localStorage.setItem(key, value);
      } catch {
        return;
      }
    },
  };
}

export function mountUI(engine: Engine): void {
  mountTheme(engine);
  mountSidebar(engine);
  mountKeyboard(engine);
  mountTooltip(engine);
  mountShortcutsSheet(engine);
  mountStatus(engine);

  document.addEventListener("click", (e) => {
    const el = e.target instanceof Element ? e.target.closest<HTMLElement>("[data-command]") : null;
    const id = el?.dataset.command;
    if (!el || !id || el.closest("[inert]")) return;
    e.preventDefault();
    engine.commands.run(id);
  });

  window.addEventListener("popstate", (e) => {
    navAt = navOf(e.state);
    engine.restoreNav(e.state);
  });
  window.addEventListener("online", () => engine.reconnect());
  window.addEventListener("pagehide", () => engine.pages.flushAll());
  document.addEventListener("visibilitychange", () => {
    const visible = document.visibilityState === "visible";
    engine.docs.setHidden(!visible);
    if (visible) engine.reconnect();
    else engine.pages.flushAll();
  });
}
