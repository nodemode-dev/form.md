import { t } from "../../../engine/i18n";
import type { CollectionView, GroupResult, Property, RowData, ViewConfig, ViewProp } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import { optionChip, paintCell } from "../cells";
import { editCell } from "../editors";
import { openAddProperty, propertyMenu } from "../propedit";
import { calcsFor, computedText, TYPE_INFO } from "../props";
import type { ViewHost, ViewRenderer } from "../surface";
import { openFilterPanel, orderedProps, saveConfig } from "../toolbar";
import { configKey, prefillFromFilter } from "./shared";

const ROW_H = 34;
const GUTTER = 44;
const OVERSCAN = 10;

function defaultWidth(p: Property): number {
  if (p.type === "title") return 280;
  if (p.type === "checkbox") return 110;
  if (p.type === "number" || p.type === "unique_id") return 140;
  if (p.type === "text" || p.type === "relation" || p.type === "multi_select") return 220;
  return 180;
}

interface Col {
  p: Property;
  vp: ViewProp;
  width: number;
  left: number;
}

export function tableView(host: ViewHost): ViewRenderer {
  const groupLimit = host.inline ? 25 : 50;
  const session = host.makeSession({ limit: host.inline ? 25 : 100, groupLimit, search: host.search() });
  const ctx = host.ctx(session);
  const el = h("div", { class: "tbl", attrs: { role: "grid", "aria-label": host.view().name } });
  const headWrap = h("div", { class: "tbl__headwrap" });
  const head = h("div", { class: "tbl__row tbl__row--head", attrs: { role: "row" } });
  headWrap.append(head);
  const scroller = h("div", { class: "tbl__scroll" });
  const body = h("div", { class: "tbl__body", attrs: { role: "rowgroup", tabindex: "0" } });
  const newRow = h("button", { class: "tbl__new", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("New") }));
  const more = h("button", { class: "tbl__more", attrs: { type: "button" } });
  const foot = h("div", { class: "tbl__row tbl__row--foot", attrs: { role: "row" } });
  const bulk = h("div", { class: "tbl__bulk" });
  scroller.append(body, more, newRow, foot);
  el.append(bulk, headWrap, scroller);
  more.hidden = true;
  bulk.hidden = true;

  let cols: Col[] = [];
  let frozen = 0;
  const selected = new Set<string>();
  const collapsed = new Set<string>();
  let active: { row: string; prop: string } | null = null;
  let destroyed = false;
  let frame = 0;
  let pendingFocus = "";

  const view = (): CollectionView => host.view();

  const computeCols = () => {
    const cfg = view().config;
    frozen = Math.max(0, Math.min(6, Number((cfg.layout?.frozen as number | undefined) ?? 0)));
    let left = GUTTER;
    cols = orderedProps(ctx, cfg)
      .filter((x) => !x.vp.hidden && x.p.type !== "button")
      .map(({ p, vp }) => {
        const width = Math.max(60, vp.width ?? defaultWidth(p));
        const c = { p, vp, width, left };
        left += width;
        return c;
      });
    const template = `${GUTTER}px ${cols.map((c) => `${c.width}px`).join(" ")} 44px`;
    el.style.setProperty("--tbl-cols", template);
    el.style.setProperty("--tbl-w", `${left + 44}px`);
  };

  const frozenStyle = (cell: HTMLElement, i: number) => {
    const c = cols[i];
    if (i < frozen && c) {
      cell.classList.add("tbl__cell--frozen");
      cell.style.left = `${c.left}px`;
    } else {
      cell.classList.remove("tbl__cell--frozen");
      cell.style.left = "";
    }
  };

  const paintHead = () => {
    reconcile(
      head,
      [{ id: "#gutter" }, ...cols.map((c) => ({ id: c.p.id })), { id: "#add" }],
      (x) => x.id,
      (x) => {
        if (x.id === "#gutter") return h("div", { class: "tbl__cell tbl__cell--gutter" });
        if (x.id === "#add") {
          const b = h("div", { class: "tbl__cell tbl__cell--add" }, h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("New property") } }, iconNode("plus")));
          b.querySelector("button")?.addEventListener("click", (e) => openAddProperty(e.currentTarget as HTMLElement, ctx, cols.at(-1)?.p.id ?? ""));
          return b;
        }
        const cell = h("div", { class: "tbl__cell tbl__cell--head", attrs: { role: "columnheader", tabindex: "-1" } }, h("span", { class: "tbl__hicon" }), h("span", { class: "tbl__hname" }), h("span", { class: "tbl__resize", attrs: { "aria-hidden": "true" } }));
        return cell;
      },
      (cell, x) => {
        if (x.id === "#gutter" || x.id === "#add") {
          if (x.id === "#add") cell.hidden = !host.canDesign();
          return;
        }
        const i = cols.findIndex((c) => c.p.id === x.id);
        const c = cols[i];
        if (!c) return;
        cell.dataset.prop = c.p.id;
        const icon = cell.firstElementChild as HTMLElement;
        if (icon.dataset.icon !== TYPE_INFO[c.p.type].icon) {
          icon.replaceChildren(iconNode(TYPE_INFO[c.p.type].icon));
          icon.dataset.icon = TYPE_INFO[c.p.type].icon;
        }
        setText(cell.children[1] as HTMLElement, c.p.name);
        frozenStyle(cell, i);
      },
    );
    if (!head.lastElementChild?.classList.contains("tbl__cell--add")) return;
  };

  const ensureCells = (row: HTMLElement) => {
    reconcile(
      row,
      [{ id: "#gutter" }, ...cols.map((c) => ({ id: c.p.id }))],
      (x) => x.id,
      (x) => {
        if (x.id === "#gutter") {
          const g = h("div", { class: "tbl__cell tbl__cell--gutter" }, h("span", { class: "tbl__grip", attrs: { "aria-hidden": "true" } }, iconNode("grip")), h("input", { class: "tbl__check", attrs: { type: "checkbox", "aria-label": t("Select row") } }));
          return g;
        }
        const c = cols.find((y) => y.p.id === x.id);
        const cell = h("div", { class: "tbl__cell", attrs: { role: "gridcell", "data-prop": x.id } }, h("span", { class: "cell-value" }));
        if (c?.p.type === "title") cell.append(h("button", { class: "tbl__open", attrs: { type: "button" } }, iconNode("expand"), h("span", { text: t("Open") })));
        return cell;
      },
      (cell, x) => {
        if (x.id === "#gutter") return;
        frozenStyle(cell, cols.findIndex((c) => c.p.id === x.id));
      },
    );
  };

  const paintRow = (rowEl: HTMLElement, row: RowData) => {
    ensureCells(rowEl);
    setAttr(rowEl, "data-selected", selected.has(row.id) ? "true" : null);
    const check = rowEl.querySelector<HTMLInputElement>(".tbl__check");
    if (check) check.checked = selected.has(row.id);
    for (const c of cols) {
      const cell = rowEl.querySelector<HTMLElement>(`:scope > [data-prop="${c.p.id}"]`);
      const value = cell?.firstElementChild as HTMLElement | null;
      if (cell && value) {
        paintCell(value, c.p, row, ctx, "table");
        setAttr(cell, "data-active", active && active.row === row.id && active.prop === c.p.id ? "true" : null);
      }
    }
  };

  const makeRow = (id: string) => {
    const r = h("div", { class: "tbl__row", attrs: { role: "row", "data-row": id } });
    return r;
  };

  const flat = () => !session.groups && !session.lanes;

  const visibleRange = (): [number, number] => {
    const rect = body.getBoundingClientRect();
    const vh = document.documentElement.clientHeight;
    const start = Math.max(0, Math.floor(-rect.top / ROW_H) - OVERSCAN);
    const end = Math.min(session.total, Math.ceil((vh - rect.top) / ROW_H) + OVERSCAN);
    return [start, Math.max(start, end)];
  };

  const paintFlat = () => {
    const limit = host.inline ? Math.min(session.total, shownInline) : session.total;
    body.style.height = `${limit * ROW_H}px`;
    body.classList.remove("tbl__body--flow");
    const [start, endRaw] = visibleRange();
    const end = Math.min(endRaw, limit);
    session.setVisible(start, end);
    const want = new Map<string, number>();
    for (let i = start; i < end; i++) want.set(session.slots[i] ?? `ph:${i}`, i);
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(body.children) as HTMLElement[]) existing.set(c.dataset.row ?? "", c);
    for (const [key, i] of want) {
      let r = existing.get(key);
      if (r) existing.delete(key);
      else {
        r = makeRow(key);
        if (key.startsWith("ph:")) {
          r.classList.add("tbl__row--ph");
          r.append(h("div", { class: "tbl__cell tbl__cell--gutter" }), h("div", { class: "tbl__ph" }));
        }
        body.append(r);
      }
      r.style.transform = `translateY(${i * ROW_H}px)`;
      r.dataset.index = String(i);
      const row = session.rows.get(key);
      if (row) paintRow(r, row);
    }
    for (const r of existing.values()) r.remove();
    more.hidden = !host.inline || session.total <= shownInline;
    if (!more.hidden) setText(more, t("Load more ({n} more)", { n: session.total - shownInline }));
  };

  let shownInline = 25;

  const paintGroups = () => {
    body.style.height = "";
    body.classList.add("tbl__body--flow");
    more.hidden = true;
    const groups: (GroupResult & { lane?: string })[] = session.lanes ? session.lanes.flatMap((l) => l.groups.map((g) => ({ ...g, lane: l.key, label: `${l.label} / ${g.label}` }))) : (session.groups ?? []);
    const items: { key: string; kind: "group" | "row" | "more"; g: GroupResult & { lane?: string }; id?: string }[] = [];
    for (const g of groups) {
      if (g.hidden) continue;
      const gk = `${g.lane ?? ""}|${g.key}`;
      items.push({ key: `g:${gk}`, kind: "group", g });
      if (collapsed.has(gk)) continue;
      for (const id of g.rows) items.push({ key: `r:${gk}:${id}`, kind: "row", g, id });
      if (g.count > g.rows.length) items.push({ key: `m:${gk}`, kind: "more", g });
    }
    reconcile(
      body,
      items,
      (x) => x.key,
      (x) => {
        if (x.kind === "group") {
          const toggle = h("button", { class: "icon-btn icon-btn--sm tbl__gtoggle", attrs: { type: "button", "aria-label": t("Collapse group") } }, iconNode("chevron-down"));
          return h("div", { class: "tbl__group" }, toggle, h("span", { class: "tbl__glabel" }), h("span", { class: "tbl__gcount" }), h("span", { class: "tbl__gcalc" }));
        }
        if (x.kind === "more") return h("button", { class: "tbl__more", attrs: { type: "button" } });
        const r = makeRow(x.id ?? "");
        r.classList.add("tbl__row--flow");
        return r;
      },
      (node, x) => {
        const gk = `${x.g.lane ?? ""}|${x.g.key}`;
        node.dataset.group = gk;
        if (x.kind === "group") {
          const label = node.children[1] as HTMLElement;
          const p = ctx.col().schema.find((y) => y.id === view().config.group?.prop);
          const sig = `${x.g.label}|${x.g.color ?? ""}`;
          if (label.dataset.sig !== sig) {
            label.dataset.sig = sig;
            if (p && (p.type === "select" || p.type === "status" || p.type === "multi_select") && x.g.key) label.replaceChildren(optionChip({ id: x.g.key, name: x.g.label, color: x.g.color ?? "default" }, p.type === "status"));
            else label.replaceChildren(document.createTextNode(x.g.label || t("Empty")));
          }
          setText(node.children[2] as HTMLElement, String(x.g.count));
          setAttr(node, "data-collapsed", collapsed.has(gk) ? "true" : null);
          return;
        }
        if (x.kind === "more") {
          setText(node, t("Load more ({n} more)", { n: x.g.count - x.g.rows.length }));
          node.dataset.key2 = x.g.key;
          if (x.g.lane !== undefined) node.dataset.lane = x.g.lane;
          return;
        }
        const row = session.rows.get(x.id ?? "");
        if (row) paintRow(node, row);
      },
    );
  };

  const paintFoot = () => {
    const calcs = view().config.calcs ?? {};
    reconcile(
      foot,
      [{ id: "#gutter" }, ...cols.map((c) => ({ id: c.p.id }))],
      (x) => x.id,
      (x) => h("div", { class: x.id === "#gutter" ? "tbl__cell tbl__cell--gutter" : "tbl__cell tbl__cell--calc", attrs: { "data-prop": x.id } }),
      (cell, x) => {
        if (x.id === "#gutter") return;
        const fn = calcs[x.id];
        const c = cols.find((y) => y.p.id === x.id);
        const label = fn ? (calcsFor(c?.p as Property).find((k) => k[0] === fn)?.[1] ?? fn) : "";
        const v = fn ? session.calcs?.[x.id] : null;
        const text = fn ? `${t(label)} ${computedText(v ?? null, c?.p)}`.trim() : "";
        setText(cell, text || (host.canWrite() ? t("Calculate") : ""));
        setAttr(cell, "data-empty", fn ? null : "true");
        frozenStyle(cell, cols.findIndex((y) => y.p.id === x.id));
      },
    );
  };

  const paintBulk = () => {
    bulk.hidden = selected.size === 0;
    if (bulk.hidden) return;
    bulk.replaceChildren(
      h("span", { text: t("{n} selected", { n: selected.size }) }),
      h("button", { class: "btn btn--sm", attrs: { type: "button", "data-bulk": "delete" } }, iconNode("trash"), h("span", { text: t("Delete") })),
      h("button", { class: "btn btn--sm", attrs: { type: "button", "data-bulk": "clear" } }, h("span", { text: t("Clear selection") })),
    );
  };

  const paint = () => {
    if (destroyed) return;
    paintHead();
    if (flat()) paintFlat();
    else paintGroups();
    paintFoot();
    foot.hidden = cols.length === 0;
    newRow.hidden = !host.canWrite();
    paintBulk();
    setAttr(el, "data-grouped", flat() ? null : "true");
    if (pendingFocus) {
      const cell = body.querySelector<HTMLElement>(`[data-row="${pendingFocus}"] [data-prop="title"]`);
      const row = session.rows.get(pendingFocus);
      if (cell && row) {
        const p = ctx.col().schema.find((x) => x.type === "title");
        pendingFocus = "";
        if (p) editCell(cell, p, row, ctx, {});
      }
    }
  };

  const schedule = () => {
    if (frame) return;
    frame = requestAnimationFrame(() => {
      frame = 0;
      if (flat()) paintFlat();
    });
  };

  session.on((c) => {
    if (c.kind === "loading") {
      setAttr(el, "data-loading", c.on ? "true" : null);
      return;
    }
    if (c.kind === "rows") {
      for (const id of c.ids) {
        const row = session.rows.get(id);
        if (!row) continue;
        for (const r of Array.from(body.querySelectorAll<HTMLElement>(`[data-row="${id}"]`))) paintRow(r, row);
      }
      paintFoot();
      return;
    }
    paint();
  });

  const onScroll = () => schedule();
  window.addEventListener("scroll", onScroll, { capture: true, passive: true });
  window.addEventListener("resize", onScroll, { passive: true });
  scroller.addEventListener("scroll", () => {
    head.style.transform = `translateX(${-scroller.scrollLeft}px)`;
  }, { passive: true });

  computeCols();
  const boot = host.engine.collections.takeBootQuery(host.viewId);
  let ready: Promise<void>;
  if (boot && !host.search()) {
    session.seed(boot);
    paint();
    ready = Promise.resolve();
  } else ready = session.load();

  const rowOf = (target: Element): RowData | undefined => {
    const r = target.closest<HTMLElement>("[data-row]");
    return r?.dataset.row ? session.rows.get(r.dataset.row) : undefined;
  };

  const orderedRows = (): string[] => session.ordered();

  const setActive = (rowId: string, propId: string) => {
    active = { row: rowId, prop: propId };
    for (const c of Array.from(body.querySelectorAll("[data-active]"))) c.removeAttribute("data-active");
    const cell = body.querySelector<HTMLElement>(`[data-row="${rowId}"] [data-prop="${propId}"]`);
    if (cell) {
      cell.setAttribute("data-active", "true");
      cell.scrollIntoView({ block: "nearest", inline: "nearest" });
    }
  };

  const edit = (cell: HTMLElement, row: RowData, p: Property, initial?: string) => {
    setActive(row.id, p.id);
    editCell(cell, p, row, ctx, { ...(initial !== undefined ? { initial } : {}), onClose: () => body.focus({ preventScroll: true }) });
  };

  body.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;
    const moreBtn = target.closest<HTMLElement>(".tbl__more");
    if (moreBtn?.dataset.key2) {
      void session.moreInGroup(moreBtn.dataset.key2, moreBtn.dataset.lane).catch(host.engine.reportError);
      return;
    }
    const g = target.closest<HTMLElement>(".tbl__group");
    if (g?.dataset.group) {
      if (collapsed.has(g.dataset.group)) collapsed.delete(g.dataset.group);
      else collapsed.add(g.dataset.group);
      paint();
      return;
    }
    const ref = target.closest<HTMLElement>("a.ref");
    if (ref?.dataset.id) {
      e.preventDefault();
      host.openRow(ref.dataset.id, e);
      return;
    }
    if (target.closest("a")) return;
    const row = rowOf(target);
    if (!row) return;
    if (target.closest(".tbl__open")) {
      host.openRow(row.id, e);
      return;
    }
    if (target.closest(".tbl__check")) {
      if ((target as HTMLInputElement).checked) selected.add(row.id);
      else selected.delete(row.id);
      paintBulk();
      setAttr(target.closest("[data-row]") as HTMLElement, "data-selected", selected.has(row.id) ? "true" : null);
      return;
    }
    const cell = target.closest<HTMLElement>("[data-prop]");
    const p = cell ? ctx.col().schema.find((x) => x.id === cell.dataset.prop) : undefined;
    if (!cell || !p) return;
    if (!host.canWrite()) {
      setActive(row.id, p.id);
      return;
    }
    edit(cell, row, p);
  });

  body.addEventListener("keydown", (e) => {
    if (!active) return;
    const rows = orderedRows();
    const ri = rows.indexOf(active.row);
    const ci = cols.findIndex((c) => c.p.id === active?.prop);
    const move = (dr: number, dc: number) => {
      const nr = rows[Math.max(0, Math.min(rows.length - 1, ri + dr))];
      const nc = cols[Math.max(0, Math.min(cols.length - 1, ci + dc))];
      if (nr && nc) setActive(nr, nc.p.id);
    };
    const row = session.rows.get(active.row);
    const c = cols[ci];
    const cell = body.querySelector<HTMLElement>(`[data-row="${active.row}"] [data-prop="${active.prop}"]`);
    switch (e.key) {
      case "ArrowDown":
        move(1, 0);
        break;
      case "ArrowUp":
        move(-1, 0);
        break;
      case "ArrowRight":
      case "Tab":
        move(0, e.shiftKey && e.key === "Tab" ? -1 : 1);
        break;
      case "ArrowLeft":
        move(0, -1);
        break;
      case "Enter":
        if (row && c && cell && host.canWrite()) edit(cell, row, c.p);
        break;
      case "Escape":
        active = null;
        for (const x of Array.from(body.querySelectorAll("[data-active]"))) x.removeAttribute("data-active");
        break;
      case "Backspace":
      case "Delete":
        if (row && c && host.canWrite() && c.p.type !== "title" && !["rollup", "formula", "created_time", "created_by", "last_edited_time", "last_edited_by", "unique_id"].includes(c.p.type)) {
          host.engine.collections.setValue(ctx.colId, row.id, c.p.id, null);
        }
        break;
      default:
        if (e.key.length === 1 && !e.metaKey && !e.ctrlKey && !e.altKey && row && c && cell && host.canWrite() && ["title", "text", "number", "url", "email", "phone", "select", "multi_select", "status"].includes(c.p.type)) {
          e.preventDefault();
          edit(cell, row, c.p, e.key);
        }
        return;
    }
    e.preventDefault();
  });

  bulk.addEventListener("click", (e) => {
    const b = (e.target as Element).closest<HTMLElement>("[data-bulk]");
    if (!b) return;
    if (b.dataset.bulk === "clear") {
      selected.clear();
      paint();
      return;
    }
    const ids = [...selected];
    selected.clear();
    paint();
    void host.engine.collections.deleteRows(ctx.colId, ids).catch(host.engine.reportError);
  });

  more.addEventListener("click", () => {
    if (!host.inline) return;
    shownInline += 50;
    paintFlat();
  });

  newRow.addEventListener("click", () => addNew(false));
  const addNew = (top: boolean) => {
    const values = prefillFromFilter(ctx, view());
    const last = session.ordered().at(-1);
    const body: { values: Record<string, unknown>; after_id?: string; top?: boolean } = { values };
    if (top) body.top = true;
    else if (last && !view().config.sorts?.length) body.after_id = last;
    void host.engine.collections
      .createRow(ctx.colId, body)
      .then((row) => {
        pendingFocus = row.id;
        if (host.inline && session.total + 1 > shownInline) shownInline = session.total + 1;
      })
      .catch(host.engine.reportError);
  };

  head.addEventListener("click", (e) => {
    const cell = (e.target as Element).closest<HTMLElement>(".tbl__cell--head");
    if (!cell?.dataset.prop || (e.target as Element).closest(".tbl__resize") || dragged) return;
    const p = ctx.col().schema.find((x) => x.id === cell.dataset.prop);
    if (!p) return;
    const idx = cols.findIndex((c) => c.p.id === p.id);
    const extra: MenuEntry[] = [
      { label: t("Sort ascending"), icon: "arrow-up", run: () => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, sorts: [{ prop: p.id, dir: "asc" }] }), 0) },
      { label: t("Sort descending"), icon: "arrow-down", run: () => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, sorts: [{ prop: p.id, dir: "desc" }] }), 0) },
      { label: t("Filter"), icon: "filter", run: () => openFilterPanel(cell, ctx, host.viewId, p.id) },
    ];
    if (p.type !== "title") extra.push({ label: t("Hide in view"), icon: "eye-off", run: () => setHidden(p.id, true) });
    extra.push(
      idx + 1 === frozen
        ? { label: t("Unfreeze columns"), icon: "close", run: () => setLayout({ frozen: 0 }) }
        : { label: t("Freeze up to this column"), icon: "columns", run: () => setLayout({ frozen: idx + 1 }) },
    );
    if (host.canDesign()) {
      extra.push(
        { label: t("Insert left"), icon: "plus", run: () => openAddProperty(cell, ctx, cols[idx - 1]?.p.id ?? "", (id) => moveProp(id, idx)) },
        { label: t("Insert right"), icon: "plus", run: () => openAddProperty(cell, ctx, p.id, (id) => moveProp(id, idx + 1)) },
      );
      propertyMenu(cell, p, ctx, extra);
    } else openMenu(cell, extra, { side: "bottom", align: "start", label: p.name });
  });

  const setHidden = (id: string, hidden: boolean) =>
    saveConfig(ctx, host.viewId, (cfg) => {
      const ordered = orderedProps(ctx, cfg).map((x) => ({ ...x.vp }));
      const me = ordered.find((x) => x.id === id);
      if (me) me.hidden = hidden;
      return { ...cfg, props: ordered };
    }, 0);
  const setLayout = (change: Record<string, unknown>) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), ...change } }), 0);
  const moveProp = (id: string, at: number) =>
    saveConfig(ctx, host.viewId, (cfg) => {
      const ordered = orderedProps(ctx, cfg).map((x) => ({ ...x.vp }));
      const from = ordered.findIndex((x) => x.id === id);
      const visible = ordered.filter((x) => !x.hidden);
      const targetId = visible[at]?.id;
      if (from < 0) return cfg;
      const [item] = ordered.splice(from, 1);
      const to = targetId ? ordered.findIndex((x) => x.id === targetId) : ordered.length;
      if (item) ordered.splice(Math.max(1, to), 0, item);
      return { ...cfg, props: ordered };
    }, 0);

  foot.addEventListener("click", (e) => {
    const cell = (e.target as Element).closest<HTMLElement>(".tbl__cell--calc");
    const p = cell?.dataset.prop ? ctx.col().schema.find((x) => x.id === cell.dataset.prop) : undefined;
    if (!cell || !p) return;
    const now = view().config.calcs?.[p.id];
    const entries: MenuEntry[] = [
      { label: t("None"), checked: !now, run: () => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, calcs: Object.fromEntries(Object.entries(cfg.calcs ?? {}).filter(([k]) => k !== p.id)) }), 0) },
      ...calcsFor(p).map(([k, label]) => ({ label: t(label), checked: now === k, run: () => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, calcs: { ...(cfg.calcs ?? {}), [p.id]: k } }), 0) })),
    ];
    openMenu(cell, entries, { side: "top", align: "start", label: t("Calculate") });
  });

  let dragged = false;
  head.addEventListener("pointerdown", (e) => {
    const handle = (e.target as Element).closest<HTMLElement>(".tbl__resize");
    const cell = (e.target as Element).closest<HTMLElement>(".tbl__cell--head");
    if (!cell?.dataset.prop) return;
    const id = cell.dataset.prop;
    const c = cols.find((x) => x.p.id === id);
    if (!c) return;
    const startX = e.clientX;
    dragged = false;
    if (handle) {
      e.preventDefault();
      handle.setPointerCapture(e.pointerId);
      const base = c.width;
      const moveH = (ev: PointerEvent) => {
        c.width = Math.max(60, base + ev.clientX - startX);
        let left = GUTTER;
        for (const x of cols) {
          x.left = left;
          left += x.width;
        }
        el.style.setProperty("--tbl-cols", `${GUTTER}px ${cols.map((x) => `${x.width}px`).join(" ")} 44px`);
        el.style.setProperty("--tbl-w", `${left + 44}px`);
        dragged = true;
      };
      const up = () => {
        handle.removeEventListener("pointermove", moveH);
        handle.removeEventListener("pointerup", up);
        if (dragged)
          saveConfig(ctx, host.viewId, (cfg) => {
            const ordered = orderedProps(ctx, cfg).map((x) => ({ ...x.vp }));
            const me = ordered.find((x) => x.id === id);
            if (me) me.width = Math.round(c.width);
            return { ...cfg, props: ordered };
          }, 0);
        setTimeout(() => (dragged = false), 0);
      };
      handle.addEventListener("pointermove", moveH);
      handle.addEventListener("pointerup", up);
      return;
    }
    if (c.p.type === "title" || !host.canDesign()) return;
    let indicator: HTMLElement | null = null;
    let target = -1;
    const moveC = (ev: PointerEvent) => {
      if (!dragged && Math.abs(ev.clientX - startX) < 5) return;
      dragged = true;
      const rect = headWrap.getBoundingClientRect();
      const x = ev.clientX - rect.left + scroller.scrollLeft;
      target = cols.findIndex((k) => x < k.left + k.width / 2);
      if (target < 1) target = target === -1 ? cols.length : 1;
      if (!indicator) {
        indicator = h("div", { class: "tbl__drop" });
        el.append(indicator);
      }
      const k = cols[target];
      indicator.style.transform = `translateX(${(k ? k.left : (cols.at(-1)?.left ?? 0) + (cols.at(-1)?.width ?? 0)) - scroller.scrollLeft}px)`;
    };
    const upC = () => {
      window.removeEventListener("pointermove", moveC);
      window.removeEventListener("pointerup", upC);
      indicator?.remove();
      if (dragged && target >= 0) {
        const from = cols.findIndex((k) => k.p.id === id);
        moveProp(id, target > from ? target - 1 : target);
      }
      setTimeout(() => (dragged = false), 0);
    };
    window.addEventListener("pointermove", moveC);
    window.addEventListener("pointerup", upC);
  });

  body.addEventListener("pointerdown", (e) => {
    const grip = (e.target as Element).closest(".tbl__grip");
    if (!grip || !host.canWrite() || view().config.sorts?.length) return;
    const rowEl = grip.closest<HTMLElement>("[data-row]");
    const id = rowEl?.dataset.row;
    if (!rowEl || !id) return;
    e.preventDefault();
    const startY = e.clientY;
    let indicator: HTMLElement | null = null;
    let over: { id: string; after: boolean } | null = null;
    const groupKey = rowEl.dataset.group;
    const move = (ev: PointerEvent) => {
      if (!indicator && Math.abs(ev.clientY - startY) < 4) return;
      if (!indicator) {
        indicator = h("div", { class: "tbl__rowdrop" });
        body.append(indicator);
        rowEl.classList.add("is-dragging");
      }
      const rows = Array.from(body.querySelectorAll<HTMLElement>(".tbl__row[data-row]")).filter((r) => !r.classList.contains("tbl__row--ph") && (!groupKey || r.dataset.group === groupKey));
      let best: HTMLElement | null = null;
      let after = false;
      for (const r of rows) {
        const rr = r.getBoundingClientRect();
        if (ev.clientY < rr.top + rr.height / 2) {
          best = r;
          after = false;
          break;
        }
        best = r;
        after = true;
      }
      if (!best?.dataset.row) return;
      over = { id: best.dataset.row, after };
      const br = best.getBoundingClientRect();
      const bodyRect = body.getBoundingClientRect();
      indicator.style.transform = `translateY(${(after ? br.bottom : br.top) - bodyRect.top}px)`;
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      indicator?.remove();
      rowEl.classList.remove("is-dragging");
      if (!over || over.id === id) return;
      const target = over.after ? { after_id: over.id } : { before_id: over.id };
      void host.engine.collections.moveRow(ctx.colId, id, target).then(() => session.scheduleRefresh([id]), host.engine.reportError);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  });

  return {
    el,
    ready,
    ctx,
    configChanged(prev, next) {
      computeCols();
      if (configKey(prev.config) !== configKey(next.config)) void session.load().catch(host.engine.reportError);
      else paint();
    },
    schemaChanged() {
      computeCols();
      paint();
    },
    setSearch(q) {
      session.options.search = q;
      void session.load().catch(host.engine.reportError);
    },
    addNew() {
      addNew(true);
    },
    settings(anchor) {
      const cfg: ViewConfig = view().config;
      openMenu(
        anchor,
        [
          { kind: "heading", label: t("Table") },
          { label: t("Freeze the title column"), checked: (cfg.layout?.frozen as number | undefined) === 1, run: () => setLayout({ frozen: (cfg.layout?.frozen as number | undefined) === 1 ? 0 : 1 }) },
        ],
        { side: "bottom", align: "end", label: t("Layout") },
      );
    },
    destroy() {
      destroyed = true;
      cancelAnimationFrame(frame);
      window.removeEventListener("scroll", onScroll, { capture: true });
      window.removeEventListener("resize", onScroll);
      session.close();
      el.remove();
    },
  };
}
