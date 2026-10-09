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
import { chartCaption, chartFormat, chartSettings, defaultSpec } from "./chart";
import { drawChart, exportChart } from "./chartdraw";

export interface Widget {
  id: string;
  title?: string;
  wide?: boolean;
  chart: ChartSpec;
}

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
  seq: number;
  abort?: AbortController;
  sig: string;
}

export function dashboardView(host: ViewHost): ViewRenderer {
  const engine = host.engine;
  let destroyed = false;
  const session = host.makeSession({ watch: () => refetchAll() });
  const ctx = host.ctx(session);
  const grid = h("div", { class: "dash__grid" });
  const add = h("button", { class: "dash__add", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add a chart") }));
  const el = h("div", { class: "dash", attrs: { "data-inline": host.inline ? "true" : "false" } }, grid, add);
  const cards = new Map<string, Card>();

  const widgets = (): Widget[] => {
    const w = host.view().config.layout?.widgets as Widget[] | undefined;
    return Array.isArray(w) ? w : [];
  };
  const save = (next: Widget[]) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), widgets: next } }), 150);

  const draw = (c: Card) => {
    if (!c.w || destroyed) return;
    const w = c.body.clientWidth;
    const hh = c.body.clientHeight;
    if (!w || !hh) return;
    const f = chartFormat(host.col(), c.w.chart);
    drawChart(c.svg, c.result, c.w.chart, { width: w, height: hh, fmt: f.fmt, axisFmt: f.axisFmt, empty: t("No rows to chart"), caption: chartCaption(host.col(), c.w.chart) });
  };
  const fetch = async (c: Card) => {
    if (!c.w) return;
    const mine = ++c.seq;
    c.abort?.abort();
    c.abort = new AbortController();
    try {
      const r = await engine.api.collections.chart(host.colId, { view_id: host.viewId, chart: c.w.chart, search: host.search() || undefined }, c.abort.signal);
      if (mine !== c.seq || destroyed) return;
      c.result = r;
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
    const body = h("div", { class: "dash__body" });
    body.append(svg);
    const card = h("section", { class: "dash__card", attrs: { "data-widget": w.id } }, h("header", { class: "dash__head" }, title, menu), body);
    const c: Card = { el: card, svg, title, body, result: null, seq: 0, sig: "" };
    cards.set(w.id, c);
    ro.observe(body);
    menu.addEventListener("click", () => cardMenu(menu, w.id));
    return card;
  };

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
      setAttr(node, "data-wide", w.wide ? "true" : null);
      setAttr(node, "data-kind", w.chart.kind);
      setText(c.title, w.title || chartCaption(host.col(), w.chart));
      setAttr(c.svg, "aria-label", c.title.textContent ?? "");
      const sig = JSON.stringify(w.chart);
      if (sig !== c.sig) {
        c.sig = sig;
        void fetch(c);
      } else draw(c);
    });
    add.hidden = !host.canDesign();
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
              { label: w.wide ? t("Half width") : t("Full width"), icon: "expand" as const, run: () => update(id, (x) => ({ ...x, wide: !x.wide })) },
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
