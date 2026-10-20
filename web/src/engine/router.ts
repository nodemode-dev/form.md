export type LibraryView = "recent" | "favorites" | "shared" | "private" | "all" | "tree" | "graph";

export const LIBRARY_VIEWS: readonly LibraryView[] = ["recent", "favorites", "shared", "private", "all", "tree", "graph"];

export type Route = { name: "home" } | { name: "page"; id: string } | { name: "library"; view: LibraryView } | { name: "trash" } | { name: "inbox" };

const ID = /^[a-z2-7]{26}$/;

export function parseRoute(path: string, search: string): Route {
  const m = /^\/p\/([^/]+)\/?$/.exec(path);
  if (m?.[1] && ID.test(m[1])) return { name: "page", id: m[1] };
  if (path === "/library" || path === "/library/") {
    const v = new URLSearchParams(search).get("view") as LibraryView | null;
    return { name: "library", view: v && LIBRARY_VIEWS.includes(v) ? v : "recent" };
  }
  if (path === "/trash" || path === "/trash/") return { name: "trash" };
  if (path === "/inbox" || path === "/inbox/") return { name: "inbox" };
  return { name: "home" };
}

export function routePath(r: Route): string {
  switch (r.name) {
    case "page":
      return `/p/${r.id}`;
    case "library":
      return r.view === "recent" ? "/library" : `/library?view=${r.view}`;
    case "trash":
      return "/trash";
    case "inbox":
      return "/inbox";
    default:
      return "/home";
  }
}

export function sameRoute(a: Route, b: Route): boolean {
  return routePath(a) === routePath(b);
}

export interface NavState {
  tabs: string[];
  route: Route;
}

export function openTab(s: NavState, id: string): NavState {
  if (s.tabs.includes(id)) return { tabs: s.tabs, route: { name: "page", id } };
  const active = s.route.name === "page" ? s.route.id : "";
  const tabs = active && s.tabs.includes(active) ? s.tabs.map((t) => (t === active ? id : t)) : [...s.tabs, id];
  return { tabs, route: { name: "page", id } };
}

export function openNewTab(s: NavState, id: string): NavState {
  if (s.tabs.includes(id)) return { tabs: s.tabs, route: { name: "page", id } };
  const active = s.route.name === "page" ? s.tabs.indexOf(s.route.id) : -1;
  const tabs = active < 0 ? [...s.tabs, id] : [...s.tabs.slice(0, active + 1), id, ...s.tabs.slice(active + 1)];
  return { tabs, route: { name: "page", id } };
}

export function closeTab(s: NavState, id: string): NavState {
  const i = s.tabs.indexOf(id);
  if (i < 0) return s;
  const tabs = s.tabs.filter((t) => t !== id);
  if (s.route.name !== "page" || s.route.id !== id) return { tabs, route: s.route };
  const next = tabs[i] ?? tabs[i - 1];
  return { tabs, route: next ? { name: "page", id: next } : { name: "home" } };
}

export function cycleTab(s: NavState, d: 1 | -1): NavState | null {
  if (s.tabs.length < 2) return null;
  const i = s.route.name === "page" ? s.tabs.indexOf(s.route.id) : -1;
  const next = s.tabs[(i + d + s.tabs.length) % s.tabs.length];
  return next ? { tabs: s.tabs, route: { name: "page", id: next } } : null;
}

export function moveTab(s: NavState, id: string, before: string): NavState {
  const rest = s.tabs.filter((t) => t !== id);
  const at = before ? rest.indexOf(before) : rest.length;
  rest.splice(at < 0 ? rest.length : at, 0, id);
  return { tabs: rest, route: s.route };
}
