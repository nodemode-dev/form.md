import { t } from "../../../engine/i18n";
import type { ChartResult, ChartSpec, Collection } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { openMenu } from "../../../ui/menu";
import { toast } from "../../../ui/toast";
import { openPanel } from "../panel";
import type { ViewHost, ViewRenderer } from "../surface";
import { saveConfig } from "../toolbar";
import { chartCaption, chartClicks, chartFormat, chartSettings, defaultSpec } from "./chart";
import { drawChart, exportChart } from "./chartdraw";
import { drawKpi } from "./kpi";

export interface Widget {
  id: string;
  title?: string;
  wide?: boolean;
  size?: "s" | "m" | "l";
  kpi?: { date: string; by: "day" | "week" | "month" };
  chart: ChartSpec;
}

const sizeOf = (w: Widget): "s" | "m" | "l" => w.size ?? (w.wide ? "l" : "m");
const PERIOD: Record<string, string> = { day: "vs the day before", week: "vs last week", month: "vs last month" };

function swap<T>(list: T[], i: number, j: number): T[] {
  const out = list.slice();
  const a = out[i];
  const b = out[j];
  if (a === undefined || b === undefined) return out;
  out[i] = b;
  out[j] = a;
  return out;
}

function newId(): string {
  return Math.random().toString(36).slice(2, 10);
}

export function defaultWidgets(col: Collection): Widget[] {
  const out: Widget[] = [{ id: newId(), title: t("Total"), chart: { kind: "number", fn: "count" } }];
  const d = defaultSpec(col);
  if (d.kind !== "number") out.push({ id: newId(), chart: d });
  const date = col.schema.find((p) => p.type === "date") ?? col.schema.find((p) => p.type === "created_time");
  if (date) out.push({ id: newId(), wide: true, title: t("Over time"), chart: { kind: "line", x: { prop: date.id, by: "month" }, fn: "count", smooth: true } });
  return out;
}

interface Card {
  el: HTMLElement;
  svg: SVGSVGElement;
  title: HTMLElement;
  body: HTMLElement;
  w?: Widget;
  result: ChartResult | null;
  spark: ChartResult | null;
  seq: number;
  abort?: AbortController;
  sig: string;
}

export function widgetsOf(layout: Record<string, unknown> | undefined): Widget[] {
  const w = layout?.widgets as Widget[] | undefined;
  return Array.isArray(w) ? w : [];
}

export function sparkSpecOf(w: Widget): ChartSpec | null {
  return w.kpi && w.chart.kind === "number" ? { kind: "line", x: { prop: w.kpi.date, by: w.kpi.by }, fn: w.chart.fn ?? "count", ...(w.chart.prop ? { prop: w.chart.prop } : {}), ...(w.chart.filter ? { filter: w.chart.filter } : {}), limit: 500 } : null;
}

export function dashboardView(host: ViewHost): ViewRenderer {
  const engine = host.engine;
  let destroyed = false;
  const session = host.makeSession({ watch: () => refetchAll() });
  const ctx = host.ctx(session);
  const grid = h("div", { class: "dash__grid" });
  const add = h("button", { class: "dash__add", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add a chart") }));
  const addKpi = h("button", { class: "dash__add", attrs: { type: "button" } }, iconNode("hash"), h("span", { text: t("Add a KPI") }));
  const adds = h("div", { class: "dash__adds" }, add, addKpi);
  const announce = h("p", { class: "visually-hidden", attrs: { "aria-live": "polite" } });
  const el = h("div", { class: "dash", attrs: { "data-inline": host.inline ? "true" : "false" } }, grid, adds, announce);
  const cards = new Map<string, Card>();
  let refocus = "";

  const widgets = (): Widget[] => widgetsOf(host.view().config.layout);
  const save = (next: Widget[]) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), widgets: next } }), 150);

  const draw = (c: Card) => {
    if (!c.w || destroyed) return;
    const w = c.body.clientWidth;
    const hh = c.body.clientHeight;
    if (!w || !hh) return;
    const f = chartFormat(host.col(), c.w.chart);
    if (c.w.kpi && c.w.chart.kind === "number") {
      if (c.result) drawKpi(c.svg, { width: w, height: hh, value: c.result.value, fmt: f.fmt, caption: chartCaption(host.col(), c.w.chart), spark: c.spark, period: t(PERIOD[c.w.kpi.by] ?? "") });
      return;
    }
    drawChart(c.svg, c.result, c.w.chart, { width: w, height: hh, fmt: f.fmt, axisFmt: f.axisFmt, empty: t("No rows to chart"), caption: chartCaption(host.col(), c.w.chart) });
  };
  const fetch = async (c: Card) => {
    if (!c.w) return;
    const mine = ++c.seq;
    c.abort?.abort();
    c.abort = new AbortController();
    try {
      const w = c.w;
      const search = host.search() || undefined;
      const sparkSpec = sparkSpecOf(w);
      const [r, s] = await Promise.all([engine.api.collections.chart(host.colId, { view_id: host.viewId, chart: w.chart, search }, c.abort.signal), sparkSpec ? engine.api.collections.chart(host.colId, { view_id: host.viewId, chart: sparkSpec, search }, c.abort.signal) : Promise.resolve(null)]);
      if (mine !== c.seq || destroyed) return;
      c.result = r;
      c.spark = s ? { ...s, x: s.x.slice(-24), series: s.series.map((x) => ({ ...x, values: x.values.slice(-24) })) } : null;
      setAttr(c.el, "data-error", null);
      draw(c);
    } catch (e) {
      if (mine !== c.seq || destroyed || (e as Error).name === "AbortError") return;
      setAttr(c.el, "data-error", "true");
      setAttr(c.el, "title", (e as Error).message);
    }
  };
  const refetchAll = () => {
    for (const c of cards.values()) void fetch(c);
  };
  let raf = 0;
  const ro = new ResizeObserver(() => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(() => {
      for (const c of cards.values()) draw(c);
    });
  });

  const makeCard = (w: Widget): HTMLElement => {
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("class", "chart");
    svg.setAttribute("role", "img");
    const title = h("div", { class: "dash__title" });
    const menu = h("button", { class: "icon-btn icon-btn--sm dash__menu", attrs: { type: "button", "aria-label": t("Chart options") } }, iconNode("more"));
    const grip = h("button", { class: "icon-btn icon-btn--sm dash__grip", attrs: { type: "button", "aria-label": t("Drag to move"), "aria-keyshortcuts": "ArrowLeft ArrowRight" } }, iconNode("grip"));
    const body = h("div", { class: "dash__body" });
    body.append(svg);
    const card = h("section", { class: "dash__card", attrs: { "data-widget": w.id } }, h("header", { class: "dash__head" }, grip, title, menu), body);
    const c: Card = { el: card, svg, title, body, result: null, spark: null, seq: 0, sig: "" };
    cards.set(w.id, c);
    ro.observe(body);
    menu.addEventListener("click", () => cardMenu(menu, w.id));
    grip.addEventListener("pointerdown", (e) => startMove(e, w.id, card));
    grip.addEventListener("keydown", (e) => keyMove(e, w.id));
    chartClicks(svg, { engine, colId: host.colId, viewId: () => host.viewId, spec: () => cards.get(w.id)?.w?.chart ?? w.chart, search: host.search, openRow: host.openRow });
    return card;
  };

  let moving: { id: string; card: HTMLElement; target: HTMLElement | null; after: boolean; pointer: number } | null = null;
  const startMove = (e: PointerEvent, id: string, card: HTMLElement) => {
    if (!host.canDesign() || e.button !== 0) return;
    e.preventDefault();
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    moving = { id, card, target: null, after: false, pointer: e.pointerId };
    setAttr(card, "data-moving", "true");
  };
  const clearDrop = () => {
    for (const x of Array.from(grid.querySelectorAll("[data-drop]"))) x.removeAttribute("data-drop");
  };
  el.addEventListener("pointermove", (e) => {
    if (!moving || e.pointerId !== moving.pointer) return;
    const under = document.elementFromPoint(e.clientX, e.clientY)?.closest<HTMLElement>(".dash__card");
    clearDrop();
    if (!under || under === moving.card || !grid.contains(under)) {
      moving.target = null;
      return;
    }
    const r = under.getBoundingClientRect();
    moving.after = e.clientX > r.left + r.width / 2;
    moving.target = under;
    setAttr(under, "data-drop", moving.after ? "after" : "before");
  });
  const endMove = (e: PointerEvent) => {
    if (!moving || e.pointerId !== moving.pointer) return;
    const m = moving;
    moving = null;
    clearDrop();
    setAttr(m.card, "data-moving", null);
    const to = m.target?.dataset.widget;
    if (!to) return;
    const list = widgets();
    const item = list.find((x) => x.id === m.id);
    if (!item) return;
    const rest = list.filter((x) => x.id !== m.id);
    const at = rest.findIndex((x) => x.id === to);
    if (at < 0) return;
    rest.splice(m.after ? at + 1 : at, 0, item);
    save(rest);
  };
  const keyMove = (e: KeyboardEvent, id: string) => {
    const d = e.key === "ArrowLeft" || e.key === "ArrowUp" ? -1 : e.key === "ArrowRight" || e.key === "ArrowDown" ? 1 : 0;
    if (!d || !host.canDesign()) return;
    e.preventDefault();
    const list = widgets();
    const i = list.findIndex((x) => x.id === id);
    const j = i + d;
    if (i < 0 || j < 0 || j >= list.length) return;
    refocus = id;
    save(swap(list, i, j));
    setText(announce, t("Moved to position {n} of {total}", { n: j + 1, total: list.length }));
  };
  el.addEventListener("pointerup", endMove);
  el.addEventListener("pointercancel", endMove);

  const paint = () => {
    if (destroyed) return;
    const list = widgets();
    const live = new Set(list.map((w) => w.id));
    for (const [id, c] of cards) {
      if (!live.has(id)) {
        c.abort?.abort();
        ro.unobserve(c.body);
        cards.delete(id);
      }
    }
    reconcile(grid, list, (w) => w.id, makeCard, (node, w) => {
      const c = cards.get(w.id);
      if (!c) return;
      c.w = w;
      setAttr(node, "data-wide", sizeOf(w) === "l" ? "true" : null);
      setAttr(node, "data-size", sizeOf(w));
      setAttr(node, "data-kind", w.kpi && w.chart.kind === "number" ? "kpi" : w.chart.kind);
      setText(c.title, w.title || chartCaption(host.col(), w.chart));
      setAttr(c.svg, "aria-label", c.title.textContent ?? "");
      const sig = JSON.stringify([w.chart, w.kpi ?? null]);
      if (sig !== c.sig) {
        c.sig = sig;
        void fetch(c);
      } else draw(c);
    });
    adds.hidden = !host.canDesign();
    if (refocus) {
      const g = cards.get(refocus)?.el.querySelector<HTMLElement>(".dash__grip");
      refocus = "";
      if (g && document.activeElement !== g) g.focus();
    }
  };

  const edit = (anchor: HTMLElement, id: string) => {
    const w = widgets().find((x) => x.id === id);
    if (!w) return;
    openPanel(
      anchor,
      (panel) => {
        const pad = h("div", { class: "dbpanel__pad dbpanel__stack chartset" });
        const name = h("input", { class: "input input--sm", attrs: { type: "text", placeholder: chartCaption(host.col(), w.chart), "aria-label": t("Chart title"), maxlength: "120" } });
        name.value = w.title ?? "";
        name.addEventListener("input", () => update(id, (x) => ({ ...x, title: name.value })));
        pad.append(h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Title") }), name));
        chartSettings(pad, ctx, w.chart, (chart) => update(id, (x) => ({ ...x, chart })), (as) => doExport(id, as));
        const dates = host.col().schema.filter((p) => p.type === "date" || p.type === "created_time" || p.type === "last_edited_time");
        if (dates.length) {
          const trend = h("select", { class: "input input--sm", attrs: { "aria-label": t("Trend") } });
          trend.append(h("option", { attrs: { value: "" }, text: t("No trend") }));
          for (const p of dates) for (const by of ["day", "week", "month"] as const) trend.append(h("option", { attrs: { value: `${p.id}|${by}` }, text: `${p.name} · ${t(by === "day" ? "by day" : by === "week" ? "by week" : "by month")}` }));
          trend.value = w.kpi ? `${w.kpi.date}|${w.kpi.by}` : "";
          trend.addEventListener("change", () => {
            const [date, by] = trend.value.split("|");
            update(id, (x) => {
              const { kpi: _drop, ...rest } = x;
              return date && by ? { ...rest, kpi: { date, by: by as "day" | "week" | "month" } } : rest;
            });
          });
          const trendRow = h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Trend for a number") }), trend);
          pad.append(trendRow);
        }
        panel.append(pad);
      },
      { label: t("Edit chart"), focus: false, align: "end" },
    );
  };
  const update = (id: string, fn: (w: Widget) => Widget) => save(widgets().map((w) => (w.id === id ? fn(w) : w)));
  const doExport = (id: string, as: "svg" | "png") => {
    const c = cards.get(id);
    if (c) exportChart(c.svg, c.title.textContent || t("Chart"), as).catch((e: Error) => toast({ id: "chart-export", tone: "danger", message: e.message }));
  };
  const cardMenu = (anchor: HTMLElement, id: string) => {
    const list = widgets();
    const i = list.findIndex((w) => w.id === id);
    const w = list[i];
    if (!w) return;
    const design = host.canDesign();
    openMenu(
      anchor,
      [
        ...(design
          ? [
              { label: t("Edit chart"), icon: "pencil" as const, run: () => edit(anchor, id) },
              { kind: "heading" as const, label: t("Size") },
              ...(["s", "m", "l"] as const).map((s) => ({ label: t(s === "s" ? "Small" : s === "m" ? "Medium" : "Full width"), checked: sizeOf(w) === s, run: () => update(id, (x) => ({ ...x, size: s, wide: s === "l" })) })),
              { kind: "separator" as const },
              { label: t("Duplicate"), icon: "copy" as const, run: () => save([...list.slice(0, i + 1), { ...structuredClone(w), id: newId() }, ...list.slice(i + 1)]) },
              ...(i > 0 ? [{ label: t("Move earlier"), icon: "chevron-left" as const, run: () => save(swap(list, i, i - 1)) }] : []),
              ...(i < list.length - 1 ? [{ label: t("Move later"), icon: "chevron-right" as const, run: () => save(swap(list, i, i + 1)) }] : []),
            ]
          : []),
        { label: t("Export as SVG"), icon: "download", run: () => doExport(id, "svg") },
        { label: t("Export as PNG"), icon: "image", run: () => doExport(id, "png") },
        ...(design ? [{ kind: "separator" as const }, { label: t("Remove chart"), icon: "trash" as const, danger: true, run: () => save(list.filter((x) => x.id !== id)) }] : []),
      ],
      { side: "bottom", align: "end", label: t("Chart options") },
    );
  };

  add.addEventListener("click", () => {
    const w: Widget = { id: newId(), chart: defaultSpec(host.col()) };
    save([...widgets(), w]);
  });
  addKpi.addEventListener("click", () => {
    const date = host.col().schema.find((p) => p.type === "date") ?? host.col().schema.find((p) => p.type === "created_time");
    const w: Widget = { id: newId(), size: "s", chart: { kind: "number", fn: "count" }, ...(date ? { kpi: { date: date.id, by: "week" as const } } : {}) };
    save([...widgets(), w]);
  });

  paint();
  const ready = Promise.resolve();

  return {
    el,
    ready,
    ctx,
    configChanged(prev, next) {
      if (JSON.stringify(prev.config.filter ?? null) !== JSON.stringify(next.config.filter ?? null)) {
        for (const c of cards.values()) c.sig = "";
      }
      paint();
    },
    schemaChanged: () => {
      for (const c of cards.values()) c.sig = "";
      paint();
    },
    setSearch: () => refetchAll(),
    destroy() {
      destroyed = true;
      for (const c of cards.values()) c.abort?.abort();
      ro.disconnect();
      cancelAnimationFrame(raf);
      session.close();
      el.remove();
    },
  };
}
