import type { Engine } from "../../engine";
import type { QuerySession, SessionOptions } from "../../engine/collections";
import { t } from "../../engine/i18n";
import type { Collection, CollectionView, RowData } from "../../engine/types";
import { confirmDialog } from "../../ui/dialog";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { toast } from "../../ui/toast";
import type { DbCtx } from "./cells";
import { openCSVImport } from "./csv";
import { openPanel } from "./panel";
import { PHASE5_VIEWS, VIEW_INFO } from "./props";
import { openFilterPanel, openGroupPanel, openPropsPanel, openSortPanel, viewOf } from "./toolbar";
import { renderers } from "./views";
import { defaultSpec } from "./views/chart";
import { defaultWidgets } from "./views/dashboard";

export interface ViewHost {
  engine: Engine;
  colId: string;
  viewId: string;
  inline: boolean;
  col(): Collection;
  view(): CollectionView;
  canWrite(): boolean;
  canDesign(): boolean;
  workspaceId(): string;
  makeSession(options?: SessionOptions): QuerySession;
  ctx(session: QuerySession): DbCtx;
  openRow(id: string, e?: MouseEvent | KeyboardEvent): void;
  search(): string;
}

export interface ViewRenderer {
  el: HTMLElement;
  ready: Promise<void>;
  ctx: DbCtx;
  configChanged(prev: CollectionView, next: CollectionView): void;
  schemaChanged(): void;
  setSearch(q: string): void;
  addNew?(): void;
  settings?(anchor: HTMLElement): void;
  destroy(): void;
}

export type ViewFactory = (host: ViewHost) => ViewRenderer;

export interface DatabaseOptions {
  inline: boolean;
  viewId?: string;
  onTitleFocus?: () => void;
}

export interface DatabaseHandle {
  el: HTMLElement;
  ready: Promise<void>;
  destroy(): void;
}

const VIEW_KEY = "formmd.view.";

function rememberView(colId: string): string {
  try {
    return localStorage.getItem(VIEW_KEY + colId) ?? "";
  } catch {
    return "";
  }
}

function storeView(colId: string, viewId: string): void {
  try {
    localStorage.setItem(VIEW_KEY + colId, viewId);
  } catch {
    return;
  }
}

function filterCount(v: CollectionView | undefined): number {
  const f = v?.config.filter;
  if (!f) return 0;
  if (f.prop) return 1;
  return f.items?.length ?? 0;
}

export function mountDatabase(engine: Engine, host: HTMLElement, colId: string, o: DatabaseOptions): DatabaseHandle {
  const store = engine.collections;
  const root = h("div", { class: "db", attrs: { "data-inline": o.inline ? "true" : "false" } });
  const title = h("input", { class: "db__title", attrs: { type: "text", placeholder: t("Untitled database"), "aria-label": t("Database title"), maxlength: "500" } });
  const tabs = h("div", { class: "db__tabs", attrs: { role: "tablist", "aria-label": t("Views") } });
  const addView = h("button", { class: "db__tab db__tab--add", attrs: { type: "button", "aria-label": t("Add a view") } }, iconNode("plus"));
  const tools = h("div", { class: "db__tools" });
  const btn = (icon: Parameters<typeof iconNode>[0], label: string, text = "") => {
    const b = h("button", { class: "db__tool", attrs: { type: "button", "aria-label": label, title: label } }, iconNode(icon));
    if (text) b.append(h("span", { class: "db__tool-label", text }));
    return b;
  };
  const filterBtn = btn("filter", t("Filter"));
  const filterBadge = h("span", { class: "db__badge" });
  filterBtn.append(filterBadge);
  const sortBtn = btn("sort", t("Sort"));
  const groupBtn = btn("group", t("Group"));
  const searchBtn = btn("search", t("Search"));
  const searchIn = h("input", { class: "db__search", attrs: { type: "search", placeholder: t("Type to search"), "aria-label": t("Search this view") } });
  const propsBtn = btn("sliders", t("Properties"));
  const layoutBtn = btn("settings", t("Layout"));
  const moreBtn = btn("more", t("More"));
  const newBtn = h("button", { class: "btn btn--primary btn--sm db__new", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("New") }));
  tools.append(filterBtn, sortBtn, groupBtn, searchBtn, searchIn, propsBtn, layoutBtn, moreBtn, newBtn);
  const bar = h("div", { class: "db__bar" }, tabs, tools);
  const head = h("div", { class: "db__head" });
  if (o.inline) head.append(title);
  head.append(bar);
  const stage = h("div", { class: "db__stage" });
  root.append(head, stage);
  host.append(root);
  searchIn.hidden = true;

  let destroyed = false;
  let col: Collection | undefined = store.get(colId);
  let activeId = "";
  let current: { id: string; r: ViewRenderer; view: CollectionView } | null = null;
  let switching = 0;
  let lastSchema = col ? JSON.stringify(col.schema) : "";
  let search = "";

  const canWrite = () => !!col?.can_write;
  const canDesign = () => !!col?.can_design;
  const makeHost = (viewId: string): ViewHost => ({
    engine,
    colId,
    viewId,
    inline: o.inline,
    col: () => store.get(colId) ?? (col as Collection),
    view: () => store.get(colId)?.views.find((v) => v.id === viewId) ?? (col?.views[0] as CollectionView),
    canWrite,
    canDesign,
    workspaceId: () => col?.workspace_id ?? engine.state.get("workspaceId"),
    makeSession: (options = {}) => store.session(colId, viewId, options),
    ctx: (session) => ({ engine, colId, col: () => store.get(colId) ?? (col as Collection), session, canWrite, canDesign, workspaceId: () => col?.workspace_id ?? "" }),
    openRow: (id, e) => {
      const mod = e && (engine.platform.mac ? e.metaKey : e.ctrlKey);
      engine.openPage(id, { newTab: !!mod });
    },
    search: () => search,
  });

  const paintTabs = () => {
    if (!col) return;
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(tabs.querySelectorAll<HTMLElement>("[data-view]"))) existing.set(c.dataset.view ?? "", c);
    let cursor: Element | null = tabs.firstElementChild;
    for (const v of col.views) {
      let b = existing.get(v.id);
      if (b) existing.delete(v.id);
      else {
        b = h("button", { class: "db__tab", attrs: { type: "button", role: "tab", "data-view": v.id } }, iconNode(VIEW_INFO[v.type]?.icon ?? "table"), h("span", { class: "db__tab-name" }));
      }
      const icon = b.firstElementChild as Element;
      const want = VIEW_INFO[v.type]?.icon ?? "table";
      if (b.dataset.icon !== want) {
        icon.replaceWith(iconNode(want));
        b.dataset.icon = want;
      }
      setText(b.lastElementChild as HTMLElement, v.name);
      setAttr(b, "aria-selected", v.id === activeId ? "true" : "false");
      if (b !== cursor) tabs.insertBefore(b, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
    if (addView.parentElement !== tabs || tabs.lastElementChild !== addView) tabs.append(addView);
    addView.hidden = !canDesign();
  };

  const paintTools = () => {
    const v = col?.views.find((x) => x.id === activeId);
    const n = filterCount(v);
    setText(filterBadge, n ? String(n) : "");
    setAttr(filterBtn, "data-on", n ? "true" : null);
    setAttr(sortBtn, "data-on", v?.config.sorts?.length ? "true" : null);
    setAttr(groupBtn, "data-on", v?.config.group?.prop ? "true" : null);
    const type = v?.type ?? "table";
    groupBtn.hidden = !["table", "board", "list", "gallery"].includes(type);
    sortBtn.hidden = ["form", "chart", "dashboard"].includes(type);
    propsBtn.hidden = ["chart", "dashboard"].includes(type);
    filterBtn.hidden = type === "form";
    searchBtn.hidden = type === "form";
    layoutBtn.hidden = !current?.r.settings;
    newBtn.hidden = !canWrite() || type === "form";
    if (o.inline) {
      const name = col?.title ?? "";
      if (document.activeElement !== title && title.value !== name) title.value = name;
      title.readOnly = !canWrite();
    }
    setAttr(root, "data-view-type", type);
  };

  const mountView = async (viewId: string, instant = false) => {
    if (!col) return;
    const view = col.views.find((v) => v.id === viewId) ?? col.views[0];
    if (!view) return;
    const factory = renderers[view.type] ?? renderers.table;
    if (!factory) return;
    const mine = ++switching;
    const r = factory(makeHost(view.id));
    const prev = current;
    activeId = view.id;
    storeView(colId, view.id);
    paintTabs();
    if (!prev || instant) {
      stage.replaceChildren(r.el);
      current = { id: view.id, r, view };
      paintTools();
      await r.ready.catch(engine.reportError);
      return;
    }
    await Promise.race([r.ready.catch(engine.reportError), new Promise((res) => setTimeout(res, 450))]);
    if (mine !== switching || destroyed) {
      r.destroy();
      return;
    }
    prev.r.destroy();
    stage.replaceChildren(r.el);
    current = { id: view.id, r, view };
    paintTools();
  };

  let readyResolve: () => void = () => undefined;
  const ready = new Promise<void>((res) => (readyResolve = res));

  const start = (c: Collection) => {
    col = c;
    lastSchema = JSON.stringify(c.schema);
    const want = o.viewId || rememberView(colId);
    const first = c.views.find((v) => v.id === want) ?? c.views[0];
    paintTabs();
    paintTools();
    if (first) void mountView(first.id, true).then(readyResolve, readyResolve);
    else readyResolve();
  };

  const off = store.subscribe(colId, (c) => {
    if (destroyed) return;
    const wasStarted = !!current;
    col = c;
    if (!wasStarted) return start(c);
    paintTabs();
    const schema = JSON.stringify(c.schema);
    if (schema !== lastSchema) {
      lastSchema = schema;
      current?.r.schemaChanged();
    }
    const v = c.views.find((x) => x.id === activeId);
    if (!v) {
      const first = c.views[0];
      if (first) void mountView(first.id);
    } else if (current && current.view !== v) {
      const prev = current.view;
      current.view = v;
      if (prev.type !== v.type) void mountView(v.id);
      else current.r.configChanged(prev, v);
    }
    paintTools();
  });
  if (col) start(col);
  else void store.load(colId, false, o.viewId || rememberView(colId)).catch((e) => {
    engine.reportError(e);
    readyResolve();
  });

  tabs.addEventListener("click", (e) => {
    const b = (e.target as Element).closest<HTMLElement>("[data-view]");
    if (!b?.dataset.view) return;
    if (b.dataset.view === activeId) {
      if (canDesign()) viewMenu(b, b.dataset.view);
      return;
    }
    void mountView(b.dataset.view);
  });
  tabs.addEventListener("keydown", (e) => {
    if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") return;
    const list = Array.from(tabs.querySelectorAll<HTMLElement>("[data-view]"));
    const i = list.findIndex((x) => x.dataset.view === activeId);
    const next = list[(i + (e.key === "ArrowRight" ? 1 : -1) + list.length) % list.length];
    if (next?.dataset.view) {
      void mountView(next.dataset.view);
      next.focus();
    }
  });

  const viewMenu = (anchor: HTMLElement, viewId: string) => {
    const v = col?.views.find((x) => x.id === viewId);
    if (!v) return;
    const idx = col?.views.findIndex((x) => x.id === viewId) ?? 0;
    const entries: MenuEntry[] = [
      { kind: "heading", label: v.name },
      {
        label: t("Rename"),
        icon: "text",
        run: () =>
          openPanel(
            anchor,
            (panel, close) => {
              const i = h("input", { class: "input input--sm", attrs: { type: "text", "aria-label": t("View name") } });
              i.value = v.name;
              i.addEventListener("keydown", (ev) => {
                if (ev.key === "Enter") {
                  void store.updateView(colId, viewId, { name: i.value.trim() || v.name });
                  close();
                }
              });
              panel.append(h("div", { class: "dbpanel__pad" }, i));
            },
            { label: t("Rename view") },
          ),
      },
      { kind: "heading", label: t("Layout") },
      ...PHASE5_VIEWS.map((type) => ({ label: t(VIEW_INFO[type].label), icon: VIEW_INFO[type].icon, checked: v.type === type, run: () => void store.updateView(colId, viewId, { type }) })),
      { kind: "separator" },
      { label: t("Duplicate view"), icon: "copy", run: () => void engine.api.collections.duplicateView(viewId).then((r) => store.load(colId, true).then(() => mountView(r.view.id))).catch(engine.reportError) },
    ];
    if (idx > 0) entries.push({ label: t("Move left"), icon: "chevron-left", run: () => void store.moveView(colId, viewId, col?.views[idx - 2]?.id ?? "").catch(engine.reportError) });
    if (col && idx < col.views.length - 1) entries.push({ label: t("Move right"), icon: "chevron-right", run: () => void store.moveView(colId, viewId, col?.views[idx + 1]?.id ?? "").catch(engine.reportError) });
    if ((col?.views.length ?? 0) > 1) {
      entries.push({ kind: "separator" }, {
        label: t("Delete view"),
        icon: "trash",
        danger: true,
        run: () =>
          void confirmDialog({ title: t("Delete the view {name}?", { name: v.name }), body: t("The rows stay; only this view and its settings go."), confirm: t("Delete view"), variant: "danger" }).then((ok) => {
            if (ok) void store.deleteView(colId, viewId).catch(engine.reportError);
          }),
      });
    }
    openMenu(anchor, entries, { side: "bottom", align: "start", label: v.name });
  };

  addView.addEventListener("click", () => {
    openMenu(
      addView,
      PHASE5_VIEWS.map((type) => ({
        label: t(VIEW_INFO[type].label),
        icon: VIEW_INFO[type].icon,
        run: () =>
          void store
            .createView(colId, type, "", type === "board" ? defaultBoard() : type === "dashboard" && col ? { layout: { widgets: defaultWidgets(col) } } : type === "chart" && col ? { layout: { chart: defaultSpec(col) } } : {})
            .then((v) => mountView(v.id))
            .catch(engine.reportError),
      })),
      { side: "bottom", align: "start", label: t("Add a view") },
    );
  });

  const defaultBoard = () => {
    const p = col?.schema.find((x) => x.type === "status") ?? col?.schema.find((x) => x.type === "select");
    return p ? { group: { prop: p.id } } : {};
  };

  const withCtx = (fn: (ctx: DbCtx) => void) => {
    if (current) fn(current.r.ctx);
  };
  filterBtn.addEventListener("click", () => withCtx((ctx) => openFilterPanel(filterBtn, ctx, activeId)));
  sortBtn.addEventListener("click", () => withCtx((ctx) => openSortPanel(sortBtn, ctx, activeId)));
  groupBtn.addEventListener("click", () => withCtx((ctx) => openGroupPanel(groupBtn, ctx, activeId)));
  propsBtn.addEventListener("click", () =>
    withCtx((ctx) => {
      const type = viewOf(ctx, activeId)?.type ?? "table";
      openPropsPanel(propsBtn, ctx, activeId, (p) => type !== "table" && p.type !== "title");
    }),
  );
  layoutBtn.addEventListener("click", () => current?.r.settings?.(layoutBtn));
  let searchTimer: ReturnType<typeof setTimeout> | undefined;
  searchBtn.addEventListener("click", () => {
    searchIn.hidden = !searchIn.hidden;
    if (!searchIn.hidden) searchIn.focus();
    else if (search) {
      search = "";
      searchIn.value = "";
      current?.r.setSearch("");
    }
  });
  searchIn.addEventListener("input", () => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      search = searchIn.value.trim();
      current?.r.setSearch(search);
    }, 200);
  });
  searchIn.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      searchIn.value = "";
      search = "";
      current?.r.setSearch("");
      searchIn.hidden = true;
      searchBtn.focus();
    }
  });
  newBtn.addEventListener("click", () => {
    if (current?.r.addNew) return current.r.addNew();
    void store
      .createRow(colId, {})
      .then((row: RowData) => engine.openPage(row.id))
      .catch(engine.reportError);
  });
  moreBtn.addEventListener("click", () => {
    const v = col?.views.find((x) => x.id === activeId);
    const entries: MenuEntry[] = [];
    if (v) {
      entries.push({
        label: t("Export as CSV"),
        icon: "download",
        run: () => {
          const a = h("a", { attrs: { href: engine.api.collections.exportUrl(v.id), download: "" } });
          document.body.append(a);
          a.click();
          a.remove();
        },
      });
    }
    if (canWrite()) entries.push({ label: t("Import CSV into this database"), icon: "upload", run: () => openCSVImport(engine, moreBtn, { collectionId: colId, workspaceId: col?.workspace_id ?? "" }) });
    entries.push({
      label: t("Copy link to view"),
      icon: "link",
      run: () => {
        const url = `${location.origin}/p/${colId}?v=${activeId}`;
        void navigator.clipboard.writeText(url).then(() => toast({ id: "copied", tone: "success", message: t("Link copied") }));
      },
    });
    if (o.inline) entries.push({ label: t("Open as full page"), icon: "expand", run: () => engine.openPage(colId) });
    openMenu(moreBtn, entries, { side: "bottom", align: "end", label: t("More") });
  });

  let titleTimer: ReturnType<typeof setTimeout> | undefined;
  title.addEventListener("input", () => {
    clearTimeout(titleTimer);
    titleTimer = setTimeout(() => {
      const p = engine.pages.get(colId);
      if (p) engine.pages.edit(colId, { title: title.value });
      else void engine.api.pages.update(colId, { title: title.value }).catch(engine.reportError);
    }, 300);
  });
  title.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      title.blur();
    }
    e.stopPropagation();
  });
  const offPages = engine.pages.subscribe((c) => {
    if (c.kind === "upsert" && c.page.id === colId && col) {
      col = { ...col, title: c.page.title, icon: c.page.icon };
      paintTools();
    }
  });

  return {
    el: root,
    ready,
    destroy() {
      destroyed = true;
      off();
      offPages();
      current?.r.destroy();
      current = null;
      root.remove();
    },
  };
}
