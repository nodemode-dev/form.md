import { locale, weekStart } from "../../../engine/format";
import { t } from "../../../engine/i18n";
import type { ChartSpec, Collection, Filter } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { openPanel } from "../panel";
import type { ViewHost, ViewRenderer } from "../surface";
import { propSelect, saveConfig } from "../toolbar";
import { chartFormat } from "./chart";
import { layer } from "./chartdraw";
import { type CalDomain, calendarRange, dayKey, layoutCalendar, paintCells } from "./chartkinds";
import { openPickList } from "./picklist";

export interface HeatConfig {
  prop?: string;
  domain?: CalDomain;
  count?: number;
  fn?: string;
  value?: string;
  color?: string;
  scale?: "" | "quantile";
  labels?: boolean;
}

const DATE_TYPES = ["date", "created_time", "last_edited_time"];
const COLORS: [string, string][] = [
  ["", "Accent"],
  ["green", "Green"],
  ["teal", "Teal"],
  ["amber", "Amber"],
  ["rose", "Rose"],
  ["violet", "Violet"],
];
const DOMAINS: [CalDomain, string, number][] = [
  ["year", "Year, by day", 1],
  ["month", "Months, by day", 12],
  ["week", "Weeks, by hour", 4],
  ["day", "Days, by hour", 14],
];

export function heatOf(col: Collection, layout: Record<string, unknown> | undefined): Required<Omit<HeatConfig, "value">> & { value: string } {
  const c = (layout?.heatmap ?? {}) as HeatConfig;
  const prop = col.schema.find((p) => p.id === c.prop && DATE_TYPES.includes(p.type)) ?? col.schema.find((p) => p.type === "date") ?? col.schema.find((p) => DATE_TYPES.includes(p.type));
  const domain = DOMAINS.some((d) => d[0] === c.domain) ? (c.domain as CalDomain) : "year";
  return { prop: prop?.id ?? "", domain, count: Math.max(1, Math.min(domain === "year" ? 3 : 31, c.count ?? DOMAINS.find((d) => d[0] === domain)?.[2] ?? 1)), fn: c.fn || "count", value: c.value ?? "", color: c.color ?? "", scale: c.scale ?? "", labels: !!c.labels };
}

const hourly = (d: CalDomain) => d === "week" || d === "day";

function shifted(domain: CalDomain, count: number, offset: number): Date {
  const now = new Date();
  switch (domain) {
    case "year":
      return new Date(now.getFullYear() - count + 1 + offset * count, 0, 1);
    case "month":
      return new Date(now.getFullYear(), now.getMonth() - count + 1 + offset * count, 1);
    case "week":
      return new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7 * (count - 1) + offset * 7 * count);
    default:
      return new Date(now.getFullYear(), now.getMonth(), now.getDate() - count + 1 + offset * count);
  }
}

function fmt(d: Date, o: Intl.DateTimeFormatOptions): string {
  try {
    return new Intl.DateTimeFormat(locale(), o).format(d);
  } catch {
    return new Intl.DateTimeFormat("en", o).format(d);
  }
}

export function heatmapView(host: ViewHost): ViewRenderer {
  const engine = host.engine;
  let destroyed = false;
  let seq = 0;
  let offset = 0;
  let abort: AbortController | null = null;
  const values = new Map<string, number>();
  let loaded = false;
  const session = host.makeSession({ watch: () => void fetch() });
  const ctx = host.ctx(session);
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "chart heatv__svg");
  svg.setAttribute("role", "img");
  const layers = { marks: layer(svg, "marks"), axis: layer(svg, "axis"), labels: layer(svg, "labels") };
  const prev = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Earlier") } }, iconNode("chevron-left"));
  const next = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Later") } }, iconNode("chevron-right"));
  const today = h("button", { class: "btn btn--sm btn--ghost", attrs: { type: "button" }, text: t("Today") });
  const label = h("span", { class: "heatv__range" });
  const total = h("span", { class: "heatv__total" });
  const scaleKey = h("span", { class: "heatv__key", attrs: { "aria-hidden": "true" } }, h("span", { text: t("Less") }), ...[0, 1, 2, 3, 4, 5].map((l) => h("i", { attrs: { "data-level": String(l) } })), h("span", { text: t("More") }));
  const frame = h("div", { class: "heatv__frame" }, svg);
  const el = h("div", { class: "heatv", attrs: { "data-inline": host.inline ? "true" : "false" } }, h("div", { class: "heatv__bar" }, prev, label, next, today, h("span", { class: "heatv__spacer" }), total), frame, h("div", { class: "heatv__foot" }, scaleKey));

  const cfg = () => heatOf(host.col(), host.view().config.layout);
  const range = () => {
    const c = cfg();
    return calendarRange({ domain: c.domain, start: shifted(c.domain, c.count, offset), count: c.count, mondayFirst: weekStart() === 1 });
  };
  const spec = (): ChartSpec | null => {
    const c = cfg();
    if (!c.prop) return null;
    const { from, to } = range();
    const last = new Date(to.getFullYear(), to.getMonth(), to.getDate() - 1);
    const filter: Filter = { prop: c.prop, cond: "between", value: { date: dayKey(from), to: dayKey(last) } };
    const measure = c.fn !== "count" && c.value ? { fn: c.fn, prop: c.value } : { fn: "count" };
    return hourly(c.domain) ? { kind: "heatmap", x: { prop: c.prop, by: "day" }, series: { prop: c.prop, by: "hour" }, limit: 500, filter, ...measure } : { kind: "calendar", x: { prop: c.prop, by: "day" }, filter, ...measure };
  };
  const paintBar = () => {
    const c = cfg();
    const { from, to } = range();
    const last = new Date(to.getFullYear(), to.getMonth(), to.getDate() - 1);
    const text = c.domain === "year" ? (c.count > 1 ? `${from.getFullYear()} to ${last.getFullYear()}` : String(from.getFullYear())) : c.domain === "month" ? `${fmt(from, { month: "short", year: "numeric" })} to ${fmt(last, { month: "short", year: "numeric" })}` : `${fmt(from, { month: "short", day: "numeric" })} to ${fmt(last, { month: "short", day: "numeric", year: "numeric" })}`;
    setText(label, text);
    setAttr(el, "data-domain", c.domain);
    setAttr(svg, "data-heat", c.color || null);
    setAttr(el, "data-heat", c.color || null);
    next.disabled = offset >= 0;
    today.disabled = offset === 0;
  };
  const draw = () => {
    if (destroyed) return;
    paintBar();
    const W = frame.clientWidth;
    const H = frame.clientHeight;
    if (!W || !H) return;
    const c = cfg();
    const f = chartFormat(host.col(), { kind: "calendar", fn: c.fn, prop: c.value });
    svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
    svg.setAttribute("width", String(W));
    svg.setAttribute("height", String(H));
    const { cells, labels } = layoutCalendar(W, H, { domain: c.domain, start: range().from, count: c.count, mondayFirst: weekStart() === 1 }, (day, hour) => values.get(hour ? `${day}|${hour}` : day) ?? 0, f.fmt);
    paintCells(layers, loaded ? cells : cells.map((x) => ({ ...x, v: 0 })), labels, c.scale, f.fmt, c.labels);
    let sum = 0;
    for (const x of cells) sum += x.v;
    setText(total, loaded ? t("{n} in range", { n: f.fmt(sum) }) : "");
    setAttr(svg, "aria-label", `${host.view().name}: ${label.textContent ?? ""}`);
  };
  const fetch = async () => {
    const s = spec();
    const mine = ++seq;
    abort?.abort();
    if (!s) {
      values.clear();
      loaded = true;
      draw();
      return;
    }
    abort = new AbortController();
    try {
      const r = await engine.api.collections.chart(host.colId, { view_id: host.viewId, chart: s, search: host.search() || undefined, date_range: host.dateRange() }, abort.signal);
      if (mine !== seq || destroyed) return;
      values.clear();
      if (s.kind === "calendar") r.x.forEach((x, i) => values.set(x.key, r.series[0]?.values[i] ?? 0));
      else for (const se of r.series) r.x.forEach((x, i) => values.set(`${x.key}|${se.key}`, se.values[i] ?? 0));
      loaded = true;
      setAttr(el, "data-error", null);
      draw();
    } catch (e) {
      if (mine !== seq || destroyed || (e as Error).name === "AbortError") return;
      setAttr(el, "data-error", "true");
      setText(total, (e as Error).message);
    }
  };
  const move = (d: number) => {
    offset = d === 0 ? 0 : Math.min(0, offset + d);
    paintBar();
    void fetch();
  };
  prev.addEventListener("click", () => move(-1));
  next.addEventListener("click", () => move(1));
  today.addEventListener("click", () => move(0));

  const addOn = (day: string, hour: string) => async (): Promise<string | null> => {
    const c = cfg();
    const p = host.col().schema.find((x) => x.id === c.prop);
    if (!p || p.type !== "date") return null;
    const row = await engine.collections.createRow(host.colId, { values: { [c.prop]: { start: hour ? `${day}T${hour.slice(1)}:00` : day } } });
    return row.id;
  };
  svg.addEventListener("click", (e) => {
    const cell = (e.target as Element).closest("[data-x]");
    const s = spec();
    if (!cell || !s) return;
    const x = cell.getAttribute("data-x") ?? "";
    const hour = cell.getAttribute("data-s") ?? "";
    const title = (cell.querySelector("title")?.textContent ?? x).replace(/:[^:]*$/, "");
    const canAdd = host.canWrite() && host.col().schema.find((p) => p.id === cfg().prop)?.type === "date";
    void engine.api.collections
      .chart(host.colId, { view_id: host.viewId, chart: s, search: host.search() || undefined, date_range: host.dateRange(), pick: hour ? { x, series: hour } : { x } })
      .then((r) =>
        openPickList({
          engine,
          colId: host.colId,
          anchor: cell,
          title,
          ids: r.picked ?? [],
          truncated: r.truncated,
          openRow: host.openRow,
          add: canAdd ? { label: t("Add a row here"), run: addOn(x, hour) } : undefined,
        }),
      )
      .catch(engine.reportError);
  });

  let raf = 0;
  const ro = new ResizeObserver(() => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(draw);
  });
  ro.observe(frame);
  paintBar();
  const ready = fetch();

  const save = (change: (c: HeatConfig) => HeatConfig) => saveConfig(ctx, host.viewId, (v) => ({ ...v, layout: { ...(v.layout ?? {}), heatmap: change({ ...((v.layout?.heatmap as HeatConfig | undefined) ?? {}) }) } }), 150);

  return {
    el,
    ready,
    ctx,
    configChanged(a, b) {
      const x = JSON.stringify([a.config.filter ?? null, a.config.layout?.heatmap ?? null]);
      const y = JSON.stringify([b.config.filter ?? null, b.config.layout?.heatmap ?? null]);
      if (x !== y) {
        paintBar();
        void fetch();
      } else draw();
    },
    schemaChanged: () => void fetch(),
    setSearch: () => void fetch(),
    addNew() {
      void addOn(dayKey(new Date()), "")().then((id) => id && host.openRow(id), engine.reportError);
    },
    settings(anchor) {
      openPanel(
        anchor,
        (panel) => {
          const c = cfg();
          const pad = h("div", { class: "dbpanel__pad dbpanel__stack chartset" });
          const field = (text: string, control: HTMLElement) => h("label", { class: "field field--compact" }, h("span", { class: "field__label", text }), control);
          const select = (name: string, opts: [string, string][], value: string, change: (v: string) => void) => {
            const s = h("select", { class: "input input--sm", attrs: { "aria-label": name } });
            for (const [v, l] of opts) s.append(h("option", { attrs: { value: v }, text: t(l) }));
            s.value = value;
            s.addEventListener("change", () => change(s.value));
            return s;
          };
          const dateSel = propSelect(ctx, c.prop, (id) => save((x) => ({ ...x, prop: id })), (p) => DATE_TYPES.includes(p.type));
          const domainSel = select(t("Layout"), DOMAINS.map(([d, l]) => [d, l]), c.domain, (v) => {
            offset = 0;
            save((x) => ({ ...x, domain: v as CalDomain, count: DOMAINS.find((d) => d[0] === v)?.[2] ?? 1 }));
          });
          const count = h("input", { class: "input input--sm", attrs: { type: "number", min: "1", max: "31", "aria-label": t("How many") } });
          count.value = String(c.count);
          count.addEventListener("change", () => save((x) => ({ ...x, count: Math.max(1, Math.min(31, Number(count.value) || 1)) })));
          const fnSel = select(t("Value"), [["count", "Count rows"], ["sum", "Sum"], ["average", "Average"], ["max", "Max"]], c.fn, (v) => {
            save((x) => ({ ...x, fn: v }));
            valueRow.hidden = v === "count";
          });
          const valueSel = propSelect(ctx, c.value, (id) => save((x) => ({ ...x, value: id })), (p) => ["number", "formula", "rollup"].includes(p.type));
          const valueRow = field(t("Property"), valueSel);
          valueRow.hidden = c.fn === "count";
          const colorSel = select(t("Colour"), COLORS, c.color, (v) => save((x) => ({ ...x, color: v })));
          const scaleSel = select(t("Colour scale"), [["", "Linear"], ["quantile", "Quantiles"]], c.scale, (v) => save((x) => ({ ...x, scale: v === "quantile" ? "quantile" : "" })));
          const labels = h("label", { class: "dpick__toggle" }, h("span", { text: t("Show values") }), h("input", { attrs: { type: "checkbox", role: "switch" } }));
          const lab = labels.querySelector("input") as HTMLInputElement;
          lab.checked = c.labels;
          lab.addEventListener("change", () => save((x) => ({ ...x, labels: lab.checked })));
          pad.append(
            h("div", { class: "dbpanel__head", text: t("Dates") }),
            field(t("Date property"), dateSel),
            field(t("Layout"), domainSel),
            field(t("How many"), count),
            h("div", { class: "dbpanel__head", text: t("Value") }),
            field(t("Value"), fnSel),
            valueRow,
            h("div", { class: "dbpanel__head", text: t("Style") }),
            field(t("Colour"), colorSel),
            field(t("Colour scale"), scaleSel),
            labels,
          );
          panel.append(pad);
        },
        { label: t("Heatmap"), focus: false, align: "end" },
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
