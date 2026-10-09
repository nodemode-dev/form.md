import { locale } from "../../../engine/format";
import { t } from "../../../engine/i18n";
import type { ChartKind, ChartResult, ChartSpec, Collection, Grouping, Property } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { type IconName, iconNode } from "../../../ui/icons";
import { toast } from "../../../ui/toast";
import type { DbCtx } from "../cells";
import { openPanel } from "../panel";
import { formatNumber } from "../props";
import type { ViewHost, ViewRenderer } from "../surface";
import { groupable, propSelect, saveConfig } from "../toolbar";
import { drawChart, exportChart } from "./chartdraw";
import { openPickList } from "./picklist";

const KIND_PATHS: Record<string, string> = {
  hbar: '<path d="M4 4.5v15M4 7h6M4 12h13M4 17h9.5"/>',
  line: '<path d="M4.5 20h15"/><path d="m4.5 15.5 4.5-5 4 3 6-7.5"/>',
  donut: '<circle cx="12" cy="12" r="7.5"/><circle cx="12" cy="12" r="3.5"/><path d="M12 4.5v4M19.5 12h-4"/>',
  area: '<path d="M4.5 19.5h15"/><path d="M4.5 17 9 11l4 3 6.5-8v13.5h-15Z" fill="currentColor" fill-opacity=".25"/>',
  radial: '<path d="M12 4.5a7.5 7.5 0 1 1-7.5 7.5"/><path d="M12 8.5a3.5 3.5 0 1 1-3.5 3.5"/>',
  gauge: '<path d="M4.5 16.5a7.5 7.5 0 0 1 15 0"/><path d="m12 16.5 3.5-4.5"/>',
  histogram: '<path d="M4.5 19.5h15"/><path d="M6 19.5v-5h3v5M9 19.5V9h3v10.5M12 19.5v-7h3v7M15 19.5v-3h3v3"/>',
  dots: '<circle cx="7" cy="9" r="1.5"/><circle cx="11" cy="12" r="1.5"/><circle cx="15.5" cy="8.5" r="1.5"/><circle cx="12" cy="16.5" r="1.5"/><circle cx="17" cy="14" r="1.5"/>',
  compare: '<path d="M4.5 20h15"/><path d="m4.5 15.5 4.5-5 4 3 6-7.5"/><path d="m4.5 17.5 5-3 4 1.5 6-4" stroke-dasharray="1.5 2"/>',
  heatmap: '<rect x="4.5" y="4.5" width="15" height="15" rx="1.5"/><path d="M4.5 9.5h15M4.5 14.5h15M9.5 4.5v15M14.5 4.5v15"/>',
  calendar: '<rect x="4.5" y="5.5" width="15" height="14" rx="2"/><path d="M8 3.5v4M16 3.5v4M4.5 10h15M8 13.5h.01M12 13.5h.01M16 13.5h.01M8 16.5h.01M12 16.5h.01"/>',
};

function kindIcon(kind: ChartKind, fallback: IconName): Element {
  const path = KIND_PATHS[kind];
  if (!path) return iconNode(fallback);
  const tpl = document.createElement("template");
  tpl.innerHTML = `<svg class="icon" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${path}</svg>`;
  return tpl.content.firstElementChild as Element;
}

const KINDS: [ChartKind, string, IconName][] = [
  ["bar", "Vertical bar", "chart"],
  ["hbar", "Horizontal bar", "chart"],
  ["line", "Line", "chart"],
  ["donut", "Donut", "chart"],
  ["number", "Number", "hash"],
  ["area", "Area", "chart"],
  ["radial", "Radial bar", "chart"],
  ["gauge", "Gauge", "chart"],
  ["histogram", "Histogram", "chart"],
  ["dots", "Dot plot", "chart"],
  ["compare", "Compare periods", "chart"],
  ["heatmap", "Heatmap", "chart"],
  ["calendar", "Calendar heatmap", "calendar"],
];

const NO_AXIS = new Set<ChartKind>(["number", "gauge"]);
const ONE_SERIES = new Set<ChartKind>(["donut", "radial", "histogram", "calendar", "compare", "dots"]);

const MEASURES: [string, string][] = [
  ["count", "Count rows"],
  ["sum", "Sum"],
  ["average", "Average"],
  ["median", "Median"],
  ["min", "Min"],
  ["max", "Max"],
  ["range", "Range"],
  ["count_values", "Count values"],
  ["count_unique", "Count unique values"],
  ["count_empty", "Count empty"],
  ["percent_not_empty", "Percent not empty"],
  ["checked", "Count checked"],
  ["percent_checked", "Percent checked"],
];

const NUMERIC = ["sum", "average", "median", "min", "max", "range"];
const DATEISH = ["date", "created_time", "last_edited_time", "formula", "rollup"];

export function defaultSpec(col: Collection): ChartSpec {
  const pick = (types: string[]) => col.schema.find((p) => types.includes(p.type));
  const p = pick(["status"]) ?? pick(["select"]) ?? pick(["multi_select"]) ?? pick(["checkbox"]) ?? pick(["person"]) ?? pick(["date", "created_time"]);
  if (!p) return { kind: "number", fn: "count" };
  return { kind: "bar", x: { prop: p.id, ...(DATEISH.includes(p.type) ? { by: "month" } : {}) }, fn: "count" };
}

export function chartOf(col: Collection, layout: Record<string, unknown> | undefined): ChartSpec {
  const c = layout?.chart as ChartSpec | undefined;
  return c?.kind ? c : defaultSpec(col);
}

export function measureProp(col: Collection, spec: ChartSpec): Property | undefined {
  if (!spec.fn || spec.fn === "count") return undefined;
  return col.schema.find((p) => p.id === spec.prop);
}

export function chartCaption(col: Collection, spec: ChartSpec): string {
  const m = MEASURES.find((x) => x[0] === (spec.fn || "count"));
  const p = measureProp(col, spec);
  return p ? t("{fn} of {prop}", { fn: t(m?.[1] ?? "Count rows"), prop: p.name }) : t(m?.[1] ?? "Count rows");
}

export function chartFormat(col: Collection, spec: ChartSpec): { fmt: (n: number) => string; axisFmt: (n: number) => string } {
  const pct = (spec.fn ?? "").startsWith("percent_");
  const p = measureProp(col, spec);
  const numeric = !!p && p.type === "number" && NUMERIC.includes(spec.fn ?? "");
  let compact: Intl.NumberFormat;
  try {
    compact = new Intl.NumberFormat(locale(), { notation: "compact", maximumFractionDigits: 1 });
  } catch {
    compact = new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 });
  }
  const plain = (n: number) => (Number.isInteger(n) ? n.toLocaleString(locale()) : (Math.round(n * 100) / 100).toLocaleString(locale()));
  if (pct) {
    const f = (n: number) => `${Math.round(n * 1000) / 10}%`;
    return { fmt: f, axisFmt: f };
  }
  if (numeric && p) return { fmt: (n) => formatNumber(n, p), axisFmt: (n) => (p.format && p.format !== "number" && p.format !== "comma" && Math.abs(n) < 1e4 ? formatNumber(n, { format: p.format, precision: 0 }) : compact.format(n)) };
  return { fmt: plain, axisFmt: (n) => compact.format(n) };
}

function field(label: string, control: HTMLElement): HTMLElement {
  return h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: label }), control);
}

function toggle(label: string, on: boolean, change: (on: boolean) => void): HTMLElement {
  const row = h("label", { class: "dpick__toggle" }, h("span", { text: label }), h("input", { attrs: { type: "checkbox", role: "switch" } }));
  const input = row.querySelector("input") as HTMLInputElement;
  input.checked = on;
  input.addEventListener("change", () => change(input.checked));
  return row;
}

function select(label: string, options: [string, string][], value: string, change: (v: string) => void): HTMLSelectElement {
  const s = h("select", { class: "input input--sm", attrs: { "aria-label": label } });
  for (const [v, l] of options) s.append(h("option", { attrs: { value: v }, text: t(l) }));
  s.value = value;
  s.addEventListener("change", () => change(s.value));
  return s;
}

export function chartSettings(pad: HTMLElement, ctx: DbCtx, start: ChartSpec, change: (spec: ChartSpec) => void, exporter?: (as: "svg" | "png") => void): void {
  const spec: ChartSpec = structuredClone(start);
  const emit = () => change(structuredClone(spec));
  const schemaProp = (id: string | undefined) => ctx.col().schema.find((p) => p.id === id);
  const firstOf = (types: string[]) => ctx.col().schema.find((p) => types.includes(p.type));
  const adapt = (k: ChartKind) => {
    const xp = schemaProp(spec.x?.prop);
    if (k === "calendar" || k === "compare") {
      const d = xp && DATEISH.includes(xp.type) ? xp : firstOf(["date", "created_time", "last_edited_time"]);
      if (d) spec.x = { prop: d.id, by: k === "calendar" ? "day" : (spec.x?.by ?? "month") };
    }
    if (k === "histogram" && xp?.type !== "number") {
      const n = firstOf(["number"]);
      if (n) spec.x = { prop: n.id, step: 10 };
    }
    if (k === "dots" && schemaProp(spec.prop)?.type !== "number") {
      const n = firstOf(["number"]);
      if (n) spec.prop = n.id;
    }
  };

  const kinds = h("div", { class: "segmented chartset__kinds", attrs: { role: "group", "aria-label": t("Chart type") } });
  for (const [k, l, icon] of KINDS) {
    const b = h("button", { class: "segmented__btn", attrs: { type: "button", "data-kind": k, "aria-label": t(l), title: t(l), "aria-pressed": "false" } }, kindIcon(k, icon));
    b.addEventListener("click", () => {
      spec.kind = k;
      adapt(k);
      xSel.dispatchEvent(new CustomEvent("prop:set", { detail: spec.x?.prop ?? "" }));
      by.value = spec.x?.by ?? "month";
      step.value = String(spec.x?.step ?? 10);
      yProp.dispatchEvent(new CustomEvent("prop:set", { detail: spec.prop ?? "" }));
      emit();
      sync();
    });
    kinds.append(b);
  }

  const xSel = propSelect(ctx, spec.x?.prop ?? "", (id) => {
    const p = schemaProp(id);
    spec.x = { prop: id, ...(p && DATEISH.includes(p.type) ? { by: "month" } : {}) };
    by.value = spec.x.by ?? "relative";
    emit();
    sync();
  }, groupable);
  const by = select(t("Date by"), [["day", "Day"], ["week", "Week"], ["month", "Month"], ["year", "Year"], ["relative", "Relative"], ["weekday", "Day of the week"], ["hour", "Hour of the day"]], spec.x?.by ?? "month", (v) => {
    if (spec.x) spec.x.by = v;
    emit();
  });
  const step = h("input", { class: "input input--sm", attrs: { type: "number", min: "0", "aria-label": t("Range size") } });
  step.value = String(spec.x?.step ?? 10);
  step.addEventListener("change", () => {
    if (spec.x) spec.x.step = Number(step.value) || 10;
    emit();
  });
  const byRow = field(t("Date by"), by);
  const stepRow = field(t("Range size"), step);

  const fn = select(t("Measure"), MEASURES, spec.fn || "count", (v) => {
    spec.fn = v;
    if (v === "count") delete spec.prop;
    emit();
    sync();
  });
  const yProp = propSelect(ctx, spec.prop ?? "", (id) => {
    spec.prop = id;
    emit();
  }, (p) => (spec.kind === "dots" || NUMERIC.includes(spec.fn ?? "") ? ["number", "formula", "rollup"].includes(p.type) : p.type !== "title" || spec.fn !== "checked"));
  const yRow = field(t("Property"), yProp);
  const fnRow = field(t("Measure"), fn);
  const num = (label: string, key: "target" | "window", min: string) => {
    const i = h("input", { class: "input input--sm", attrs: { type: "number", min, step: "any", "aria-label": label } });
    i.value = spec[key] ? String(spec[key]) : "";
    i.addEventListener("change", () => {
      const n = Number(i.value);
      if (Number.isFinite(n) && n > 0) spec[key] = n;
      else delete spec[key];
      emit();
    });
    return field(label, i);
  };
  const targetRow = num(t("Target"), "target", "0");
  const windowRow = num(t("Buckets per period"), "window", "1");
  const scaleRow = field(
    t("Colour scale"),
    select(t("Colour scale"), [["", "Linear"], ["quantile", "Quantiles"]], spec.scale ?? "", (v) => {
      spec.scale = v as NonNullable<ChartSpec["scale"]>;
      emit();
    }),
  );

  const sSel = propSelect(ctx, spec.series?.prop ?? "", (id) => {
    const p = schemaProp(id);
    spec.series = { prop: id, ...(p && DATEISH.includes(p.type) ? { by: "month" } : {}) } as Grouping;
    emit();
    sync();
  }, (p) => groupable(p) && p.type !== "title");
  const sClear = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("close"), h("span", { text: t("No grouping") }));
  sClear.addEventListener("click", () => {
    delete spec.series;
    emit();
    sync();
  });
  const seriesRow = h("div", { class: "chartset__series" }, field(t("Group by"), sSel), sClear);

  const order = select(t("Sort"), [["", "Axis order"], ["value_desc", "Largest first"], ["value_asc", "Smallest first"]], spec.sort ?? "", (v) => {
    spec.sort = v as NonNullable<ChartSpec["sort"]>;
    emit();
  });
  const orderRow = field(t("Sort"), order);
  const flag = (key: "labels" | "legend" | "smooth" | "cumulative" | "stacked" | "hide_empty", label: string, dflt = false) =>
    toggle(label, spec[key] ?? dflt, (on) => {
      spec[key] = on;
      emit();
    });
  const labels = flag("labels", t("Data labels"));
  const legend = flag("legend", t("Legend"), true);
  const smooth = flag("smooth", t("Smooth line"));
  const cumulative = flag("cumulative", t("Cumulative"));
  const stacked = flag("stacked", t("Stack groups"));
  const hideEmpty = flag("hide_empty", t("Omit empty groups"));
  const axisHead = h("div", { class: "dbpanel__head", text: t("X axis") });
  const xRow = field(t("Property"), xSel);
  const valueHead = h("div", { class: "dbpanel__head", text: t("Value") });
  const styleHead = h("div", { class: "dbpanel__head", text: t("Style") });
  pad.append(kinds, axisHead, xRow, byRow, stepRow, seriesRow, valueHead, fnRow, yRow, targetRow, windowRow, orderRow, styleHead, scaleRow, labels, legend, smooth, cumulative, stacked, hideEmpty);
  if (exporter) {
    const svgBtn = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("download"), h("span", { text: t("SVG") }));
    const pngBtn = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("image"), h("span", { text: t("PNG") }));
    svgBtn.addEventListener("click", () => exporter("svg"));
    pngBtn.addEventListener("click", () => exporter("png"));
    pad.append(h("div", { class: "dbpanel__head", text: t("Export") }), h("div", { class: "chartset__export" }, svgBtn, pngBtn));
  }

  const sync = () => {
    for (const b of Array.from(kinds.children) as HTMLElement[]) setAttr(b, "aria-pressed", b.dataset.kind === spec.kind ? "true" : "false");
    const k = spec.kind;
    const number = NO_AXIS.has(k);
    const xp = schemaProp(spec.x?.prop);
    for (const n of [axisHead, xRow, orderRow, labels, hideEmpty]) n.hidden = number;
    orderRow.hidden = number || k === "calendar" || k === "compare" || k === "histogram";
    byRow.hidden = number || k === "calendar" || !xp || !DATEISH.includes(xp.type);
    stepRow.hidden = number || xp?.type !== "number";
    seriesRow.hidden = number || ONE_SERIES.has(k);
    sClear.hidden = !spec.series;
    legend.hidden = number || k === "heatmap" || k === "calendar" || k === "histogram" || k === "dots";
    smooth.hidden = !(k === "line" || k === "area" || k === "compare" || k === "histogram");
    cumulative.hidden = !(k === "line" || k === "area");
    stacked.hidden = !(((k === "bar" || k === "hbar") && spec.series) || (k === "area" && spec.series));
    fnRow.hidden = k === "dots";
    yRow.hidden = k !== "dots" && (!spec.fn || spec.fn === "count");
    targetRow.hidden = k !== "gauge";
    windowRow.hidden = k !== "compare";
    scaleRow.hidden = !(k === "heatmap" || k === "calendar");
  };
  sync();
}

export function chartClicks(svg: SVGSVGElement, o: { engine: ViewHost["engine"]; colId: string; viewId: () => string; spec: () => ChartSpec; search: () => string; openRow: ViewHost["openRow"] }): void {
  svg.addEventListener("click", (e) => {
    const target = e.target as Element;
    const row = target.closest("[data-row]");
    if (row) {
      o.openRow(row.getAttribute("data-row") ?? "", e);
      return;
    }
    const mark = target.closest("[data-x]");
    if (!mark) return;
    const x = mark.getAttribute("data-x") ?? "";
    const s = mark.getAttribute("data-s") ?? "";
    const title = (mark.querySelector("title")?.textContent ?? "").replace(/:[^:]*$/, "") || t("Rows");
    void o.engine.api.collections
      .chart(o.colId, { view_id: o.viewId(), chart: o.spec(), search: o.search() || undefined, pick: s ? { x, series: s } : { x } })
      .then((r) => openPickList({ engine: o.engine, colId: o.colId, anchor: mark, title, ids: r.picked ?? [], truncated: r.truncated, openRow: o.openRow }))
      .catch(o.engine.reportError);
  });
}

export function chartView(host: ViewHost): ViewRenderer {
  const engine = host.engine;
  let result: ChartResult | null = null;
  let seq = 0;
  let destroyed = false;
  let abort: AbortController | null = null;
  const session = host.makeSession({ watch: () => void fetch() });
  const ctx = host.ctx(session);
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "chart");
  svg.setAttribute("role", "img");
  const frame = h("div", { class: "chartv__frame" });
  frame.append(svg);
  const foot = h("div", { class: "chartv__foot" });
  const el = h("div", { class: "chartv", attrs: { "data-inline": host.inline ? "true" : "false" } }, frame, foot);

  const spec = () => chartOf(host.col(), host.view().config.layout);
  chartClicks(svg, { engine, colId: host.colId, viewId: () => host.viewId, spec, search: host.search, openRow: host.openRow });
  const draw = () => {
    if (destroyed) return;
    const s = spec();
    const f = chartFormat(host.col(), s);
    const w = frame.clientWidth;
    const hgt = frame.clientHeight;
    if (!w || !hgt) return;
    drawChart(svg, result, s, { width: w, height: hgt, fmt: f.fmt, axisFmt: f.axisFmt, empty: t("No rows to chart"), caption: chartCaption(host.col(), s) });
    setAttr(svg, "aria-label", `${host.view().name}: ${chartCaption(host.col(), s)}`);
    if (result) setText(foot, result.truncated ? t("{n} rows, showing the first groups", { n: result.rows }) : t("{n} rows", { n: result.rows }));
  };
  const fetch = async () => {
    const mine = ++seq;
    abort?.abort();
    abort = new AbortController();
    try {
      const r = await engine.api.collections.chart(host.colId, { view_id: host.viewId, chart: spec(), search: host.search() || undefined }, abort.signal);
      if (mine !== seq || destroyed) return;
      result = r;
      setAttr(el, "data-error", null);
      draw();
    } catch (e) {
      if (mine !== seq || destroyed || (e as Error).name === "AbortError") return;
      setAttr(el, "data-error", "true");
      setText(foot, (e as Error).message);
    }
  };
  let raf = 0;
  const ro = new ResizeObserver(() => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(draw);
  });
  ro.observe(frame);
  const ready = fetch();

  const doExport = (as: "svg" | "png") => {
    exportChart(svg, host.view().name || t("Chart"), as).catch((e: Error) => toast({ id: "chart-export", tone: "danger", message: e.message }));
  };

  return {
    el,
    ready,
    ctx,
    configChanged(prev, next) {
      const a = JSON.stringify([prev.config.filter ?? null, prev.config.layout?.chart ?? null]);
      const b = JSON.stringify([next.config.filter ?? null, next.config.layout?.chart ?? null]);
      if (a !== b) void fetch();
      else draw();
    },
    schemaChanged: () => void fetch(),
    setSearch: () => void fetch(),
    settings(anchor) {
      openPanel(
        anchor,
        (panel) => {
          const pad = h("div", { class: "dbpanel__pad dbpanel__stack chartset" });
          chartSettings(pad, ctx, spec(), (next) => {
            saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), chart: next } }), 150);
          }, doExport);
          panel.append(pad);
        },
        { label: t("Chart"), focus: false, align: "end" },
      );
    },
    destroy() {
      destroyed = true;
      abort?.abort();
      ro.disconnect();
      cancelAnimationFrame(raf);
      session.close();
      el.remove();
    },
  };
}
