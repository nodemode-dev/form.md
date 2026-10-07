import type { Engine, LibraryView, Page } from "../engine";
import { formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { h, ref, setAttr, setText } from "./dom";
import { copyText, select } from "./form";
import { type IconName, iconNode } from "./icons";
import { pageLink } from "./page-actions";
import { fallbackGlyph, pageIcon, paintIcon } from "./pageicon";
import { toast } from "./toast";

type Sort = "changed" | "name" | "created" | "opened";

const VIEWS: { id: LibraryView; label: () => string; icon: IconName }[] = [
  { id: "recent", label: () => t("Recently used"), icon: "clock" },
  { id: "favorites", label: () => t("Favorites"), icon: "star" },
  { id: "shared", label: () => t("Shared"), icon: "users" },
  { id: "private", label: () => t("Private"), icon: "lock" },
  { id: "all", label: () => t("All pages"), icon: "library" },
  { id: "tree", label: () => t("Tree as agents read it"), icon: "list-tree" },
];

interface Row {
  page: Page;
  visited: number;
}

export function mountLibraryView(engine: Engine): void {
  const { state, pages } = engine;
  const view = ref("view-library");

  const tabs = h("div", { class: "lib-tabs", attrs: { role: "tablist", "aria-label": t("Library views") } });
  for (const v of VIEWS) {
    tabs.append(h("button", { class: "lib-tab", attrs: { type: "button", role: "tab", "data-view": v.id, "aria-selected": "false" } }, iconNode(v.icon), h("span", { text: v.label() })));
  }
  const search = h("input", { class: "input lib-search", attrs: { type: "search", placeholder: t("Filter pages"), "aria-label": t("Filter pages"), autocomplete: "off", spellcheck: "false" } });
  const scope = select<"here" | "all">([["here", t("This workspace")], ["all", t("All workspaces")]], "here");
  scope.setAttribute("aria-label", t("Workspace"));
  const sort = select<Sort>([["changed", t("Recently changed")], ["opened", t("Recently opened")], ["created", t("Recently created")], ["name", t("Name (A to Z)")]], "changed");
  sort.setAttribute("aria-label", t("Sort"));
  const toolbar = h("div", { class: "lib-toolbar" }, h("span", { class: "search-field lib-search-field" }, iconNode("search"), search), scope, sort);
  const count = h("p", { class: "lib-count" });
  const head = h("div", { class: "lib-row lib-row--head", attrs: { "aria-hidden": "true" } }, h("span", { text: t("Page") }), h("span", { text: t("Workspace") }), h("span", { text: t("Changed") }), h("span", { text: t("Created by") }), h("span"));
  const list = h("div", { class: "lib-list", attrs: { role: "list" } });
  const empty = h("p", { class: "lib-empty", text: t("No pages.") });
  const treeText = h("pre", { class: "lib-tree" });
  const treeCopy = h("button", { class: "btn btn--ghost", attrs: { type: "button" } }, iconNode("copy"), h("span", { text: t("Copy") }));
  const treeLead = h("p", { class: "section__lead", text: t("This is the structure an agent reads over MCP: every page as a stable Markdown link, nested like the sidebar.") });
  const treeBox = h("div", { class: "lib-treebox" }, h("div", { class: "lib-treebox__bar" }, treeLead, treeCopy), treeText);
  const table = h("div", { class: "lib-table" }, head, list, empty);
  view.append(h("div", { class: "lib" }, h("h1", { class: "lib-title", text: t("Library") }), tabs, toolbar, count, table, treeBox));

  let recent: Row[] = [];
  let recentLoaded = false;
  const people = new Map<string, string>();
  const loadedPeople = new Set<string>();

  const active = (): LibraryView => {
    const r = state.get("route");
    return r.name === "library" ? r.view : "recent";
  };

  const loadPeople = (ws: string) => {
    if (loadedPeople.has(ws)) return;
    loadedPeople.add(ws);
    void engine.api.workspaces
      .members(ws)
      .then((r) => {
        for (const m of r.members) people.set(m.user_id, m.name);
        paint();
      })
      .catch(() => undefined);
  };

  const ensureData = () => {
    const v = active();
    const all = scope.value === "all";
    const ws = state.get("workspaceId");
    const spaces = all ? (state.get("me")?.workspaces ?? []).map((w) => w.id) : [ws];
    for (const id of spaces) {
      if (!pages.isLoaded(id)) void pages.load(id).then(paint, engine.reportError);
      loadPeople(id);
    }
    if (v === "recent" && !recentLoaded) {
      recentLoaded = true;
      void engine.api.pages
        .recent(100)
        .then((r) => {
          recent = r.pages.map((p) => ({ page: p, visited: p.visited_at }));
          paint();
        })
        .catch(engine.reportError);
    }
  };

  const rows = (): Row[] => {
    const v = active();
    const ws = state.get("workspaceId");
    const all = scope.value === "all";
    const inScope = (p: Page) => all || p.workspace_id === ws;
    const visitedAt = new Map(recent.map((r) => [r.page.id, r.visited]));
    let out: Row[];
    if (v === "recent") out = recent.map((r) => ({ page: pages.get(r.page.id) ?? r.page, visited: r.visited })).filter((r) => inScope(r.page) && !!pages.get(r.page.id));
    else if (v === "favorites") out = pages.favorites().filter(inScope).map((p) => ({ page: p, visited: visitedAt.get(p.id) ?? 0 }));
    else {
      const spaces = all ? (state.get("me")?.workspaces ?? []).map((w) => w.id) : [ws];
      out = spaces.flatMap((id) => pages.all(id)).map((p) => ({ page: p, visited: visitedAt.get(p.id) ?? 0 }));
      if (v === "shared") out = out.filter((r) => r.page.visibility !== "private" && !engine.workspaceById(r.page.workspace_id)?.personal);
      if (v === "private") out = out.filter((r) => r.page.visibility === "private" || !!engine.workspaceById(r.page.workspace_id)?.personal);
    }
    const q = search.value.trim().toLowerCase();
    if (q) out = out.filter((r) => (r.page.title || t("Untitled")).toLowerCase().includes(q) || r.page.tags.some((x) => x.toLowerCase().includes(q)));
    const s = sort.value as Sort;
    if (v === "recent" && s === "changed") out.sort((a, b) => b.visited - a.visited);
    else if (s === "name") out.sort((a, b) => (a.page.title || "").localeCompare(b.page.title || ""));
    else if (s === "created") out.sort((a, b) => b.page.created_at - a.page.created_at);
    else if (s === "opened") out.sort((a, b) => b.visited - a.visited || b.page.updated_at - a.page.updated_at);
    else out.sort((a, b) => b.page.updated_at - a.page.updated_at);
    return out;
  };

  const pathOf = (p: Page) =>
    pages
      .ancestors(p.id)
      .map((a) => a.title || t("Untitled"))
      .join(" / ");

  const treeMarkdown = () => {
    const ws = state.get("workspaceId");
    const lines: string[] = [];
    const walk = (parent: string, depth: number) => {
      for (const p of pages.children(ws, parent)) {
        lines.push(`${"  ".repeat(depth)}- [${(p.title || t("Untitled")).replace(/[[\]]/g, "")}](/p/${p.id})${p.visibility === "private" ? " (private)" : ""}`);
        walk(p.id, depth + 1);
      }
    };
    walk("", 0);
    return lines.join("\n");
  };

  const paint = () => {
    if (state.get("route").name !== "library") return;
    const v = active();
    for (const b of tabs.children) {
      const on = (b as HTMLElement).dataset.view === v;
      setAttr(b, "aria-selected", on ? "true" : "false");
      (b as HTMLElement).tabIndex = on ? 0 : -1;
    }
    const isTree = v === "tree";
    toolbar.hidden = isTree;
    table.hidden = isTree;
    count.hidden = isTree;
    treeBox.hidden = !isTree;
    if (isTree) {
      const text = treeMarkdown() || t("No pages.");
      setText(treeText, text);
      return;
    }
    const list_ = rows();
    setText(count, plural(list_.length, "{count} page", "{count} pages"));
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(list.children) as HTMLElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = list.firstElementChild;
    for (const r of list_.slice(0, 500)) {
      const p = r.page;
      let el = existing.get(p.id);
      if (el) existing.delete(p.id);
      else {
        el = h(
          "div",
          { class: "lib-row", attrs: { role: "listitem", tabindex: "-1", "data-id": p.id } },
          h("span", { class: "lib-row__page" }, pageIcon(p.icon, fallbackGlyph(p.kind)), h("span", { class: "lib-row__text" }, h("span", { class: "lib-row__title" }), h("span", { class: "lib-row__path" }))),
          h("span", { class: "lib-row__ws" }),
          h("span", { class: "lib-row__when" }),
          h("span", { class: "lib-row__who" }),
          h("button", { class: "icon-btn lib-row__copy", attrs: { type: "button", "data-action": "copy", "aria-label": t("Copy Markdown link"), "data-tip": t("Copy Markdown link") } }, iconNode("link")),
        );
        el.dataset.key = p.id;
      }
      paintIcon(el.querySelector(".page-icon") as HTMLElement, p.icon, fallbackGlyph(p.kind));
      const title = el.querySelector(".lib-row__title") as HTMLElement;
      setText(title, p.title || t("Untitled"));
      setAttr(title, "data-untitled", p.title ? null : "true");
      setText(el.querySelector(".lib-row__path") as HTMLElement, pathOf(p));
      setText(el.querySelector(".lib-row__ws") as HTMLElement, engine.workspaceById(p.workspace_id)?.name ?? "");
      setText(el.querySelector(".lib-row__when") as HTMLElement, v === "recent" && r.visited ? formatRelative(r.visited) : formatRelative(p.updated_at));
      setText(el.querySelector(".lib-row__who") as HTMLElement, people.get(p.created_by) ?? "");
      if (el !== cursor) list.insertBefore(el, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
    empty.hidden = list_.length > 0;
    if (!list.querySelector('.lib-row[tabindex="0"]')) {
      const first = list.firstElementChild as HTMLElement | null;
      if (first) first.tabIndex = 0;
    }
  };

  list.addEventListener("keydown", (e) => {
    const row = (e.target as HTMLElement).closest<HTMLElement>(".lib-row");
    if (!row || e.target !== row) return;
    const all = [...list.querySelectorAll<HTMLElement>(".lib-row")];
    const i = all.indexOf(row);
    let next: HTMLElement | undefined;
    if (e.key === "ArrowDown") next = all[i + 1];
    else if (e.key === "ArrowUp") next = all[i - 1];
    else if (e.key === "Home") next = all[0];
    else if (e.key === "End") next = all.at(-1);
    else if (e.key === "Enter") {
      e.preventDefault();
      engine.openPage(row.dataset.id ?? "", { newTab: engine.platform.mac ? e.metaKey : e.ctrlKey });
      return;
    } else return;
    e.preventDefault();
    if (!next) return;
    row.tabIndex = -1;
    next.tabIndex = 0;
    next.focus();
  });

  tabs.addEventListener("click", (e) => {
    const b = (e.target as HTMLElement).closest<HTMLElement>(".lib-tab");
    const v = b?.dataset.view as LibraryView | undefined;
    if (v) engine.navigate({ name: "library", view: v }, { replace: true });
  });
  tabs.addEventListener("keydown", (e) => {
    const all = [...tabs.querySelectorAll<HTMLElement>(".lib-tab")];
    const i = all.findIndex((b) => b.getAttribute("aria-selected") === "true");
    const d = e.key === "ArrowRight" ? 1 : e.key === "ArrowLeft" ? -1 : 0;
    if (!d) return;
    e.preventDefault();
    const next = all[(i + d + all.length) % all.length];
    const v = next?.dataset.view as LibraryView | undefined;
    if (v) {
      engine.navigate({ name: "library", view: v }, { replace: true });
      next?.focus();
    }
  });
  list.addEventListener("click", (e) => {
    const row = (e.target as HTMLElement).closest<HTMLElement>(".lib-row");
    const id = row?.dataset.id;
    if (!id) return;
    e.preventDefault();
    if ((e.target as HTMLElement).closest('[data-action="copy"]')) {
      const p = pages.get(id);
      void copyText(`[${p?.title || t("Untitled")}](${pageLink(id)})`).then((ok) => toast({ tone: ok ? "success" : "danger", message: ok ? t("Markdown link copied") : t("Could not copy") }));
      return;
    }
    engine.openPage(id, { newTab: engine.platform.mac ? e.metaKey : e.ctrlKey });
  });
  list.addEventListener("auxclick", (e) => {
    const id = (e.target as HTMLElement).closest<HTMLElement>(".lib-row")?.dataset.id;
    if (e.button !== 1 || !id) return;
    e.preventDefault();
    engine.openPage(id, { newTab: true });
  });
  treeCopy.addEventListener("click", () => void copyText(treeMarkdown()).then((ok) => toast({ tone: ok ? "success" : "danger", message: ok ? t("Copied") : t("Could not copy") })));
  search.addEventListener("input", paint);
  scope.addEventListener("change", () => {
    ensureData();
    paint();
  });
  sort.addEventListener("change", paint);

  state.watch("route", (r) => {
    if (r.name !== "library") return;
    if (r.view === "recent") recentLoaded = false;
    ensureData();
    paint();
  });
  state.watch("workspaceId", () => {
    ensureData();
    paint();
  });
  pages.subscribe((c) => {
    if (state.get("route").name !== "library") return;
    if (c.kind === "upsert" || c.kind === "remove" || c.kind === "loaded" || c.kind === "favorites") paint();
  });
  if (state.get("route").name === "library") {
    ensureData();
    paint();
  }
}
