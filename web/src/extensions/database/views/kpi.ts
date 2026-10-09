import type { ChartResult } from "../../../engine/types";
import { keyed, layer, monotone, set, text, tip } from "./chartdraw";

export interface KpiDraw {
  width: number;
  height: number;
  value: number;
  fmt: (n: number) => string;
  caption: string;
  spark: ChartResult | null;
  period: string;
}

export function kpiChange(spark: ChartResult | null): { last: number; prev: number; pct: number | null } | null {
  const vals = spark?.series[0]?.values ?? [];
  if (vals.length < 2) return null;
  const last = vals[vals.length - 1] ?? 0;
  const prev = vals[vals.length - 2] ?? 0;
  return { last, prev, pct: prev ? ((last - prev) / Math.abs(prev)) * 100 : null };
}

export function drawKpi(svg: SVGSVGElement, o: KpiDraw): void {
  const W = Math.max(120, o.width);
  const H = Math.max(80, o.height);
  set(svg, { viewBox: `0 0 ${W} ${H}`, width: W, height: H, "data-kind": "kpi" });
  const grid = layer(svg, "grid");
  const marks = layer(svg, "marks");
  const labels = layer(svg, "labels");
  const center = layer(svg, "center");
  for (const g of [layer(svg, "axis"), layer(svg, "legend")]) if (g.firstChild) g.replaceChildren();
  const big = Math.min(40, Math.max(24, W / 9));
  const change = kpiChange(o.spark);
  const lines = [
    { k: "value", t: o.fmt(o.value), x: 4, y: big + 2, cls: "chart__big", size: big },
    { k: "caption", t: o.caption, x: 4, y: big + 22, cls: "chart__caption", size: 12 },
  ];
  keyed(center, lines, (l) => l.k, "text", (e, l) => {
    set(e, { x: l.x, y: l.y, class: l.cls, "font-size": l.size, "text-anchor": "start" });
    text(e, l.t);
  });
  const badge =
    change && change.pct !== null
      ? [{ k: "change", t: `${change.pct > 0 ? "+" : ""}${Math.round(change.pct * 10) / 10}% ${o.period}`, dir: change.pct > 0 ? "up" : change.pct < 0 ? "down" : "flat" }]
      : change
        ? [{ k: "change", t: `${o.fmt(change.last)} ${o.period}`, dir: "flat" }]
        : [];
  keyed(labels, badge, (b) => b.k, "text", (e, b) => {
    set(e, { x: W - 4, y: 18, "text-anchor": "end", class: "chart__kpi-change", "data-dir": b.dir });
    text(e, b.t);
  });
  const vals = o.spark?.series[0]?.values ?? [];
  const top = big + 34;
  const ph = Math.max(12, H - top - 4);
  if (vals.length < 2 || ph < 14) {
    if (marks.firstChild) marks.replaceChildren();
    if (grid.firstChild) grid.replaceChildren();
    return;
  }
  const lo = Math.min(0, ...vals);
  const hi = Math.max(...vals, lo + 1);
  const step = (W - 8) / (vals.length - 1);
  const pts = vals.map((v, i) => [4 + i * step, top + ph - ((v - lo) / (hi - lo)) * ph] as [number, number]);
  const d = monotone(pts);
  const base = top + ph;
  keyed(marks, ["area", "line"], (k) => k, "path", (e, k) => {
    if (k === "area") set(e, { d: `${d}L${(pts[pts.length - 1]?.[0] ?? W).toFixed(1)},${base}L${(pts[0]?.[0] ?? 0).toFixed(1)},${base}Z`, class: "chart__area chart__area--solid", "data-pal": "0" });
    else set(e, { d, class: "chart__line", "data-pal": "0" });
  });
  const last = pts[pts.length - 1];
  const x = o.spark?.x ?? [];
  keyed(grid, last ? ["end"] : [], (k) => k, "circle", (e) => {
    set(e, { cx: last?.[0] ?? 0, cy: last?.[1] ?? 0, r: 3, class: "chart__dot", "data-pal": "0" });
    tip(e, `${x[x.length - 1]?.label ?? ""}: ${o.fmt(vals[vals.length - 1] ?? 0)}`);
  });
}
