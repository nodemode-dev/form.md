import { locale } from "../../../engine/format";
import type { ChartResult, ChartSpec } from "../../../engine/types";
import { arc, catLabel, chord, clip, type DrawOpts, fitText, keyed, layer, legendRows, linear, monotone, niceTicks, paintSwatch, type Swatch, set, text, tickLabel, tip, width } from "./chartdraw";

export const EXTRA_KINDS: ReadonlySet<string> = new Set(["area", "radial", "gauge", "histogram", "dots", "compare", "heatmap", "calendar"]);

type Series = ChartResult["series"][number];
type Point = ChartResult["x"][number];

interface Frame {
  W: number;
  H: number;
  grid: SVGGElement;
  axis: SVGGElement;
  marks: SVGGElement;
  labels: SVGGElement;
  legend: SVGGElement;
  center: SVGGElement;
}

const v = (s: Series | undefined, i: number) => s?.values[i] ?? 0;

function clear(...gs: SVGGElement[]): void {
  for (const g of gs) if (g.firstChild) g.replaceChildren();
}

function centerText(f: Frame, lines: { t: string; size: number; cls: string }[], cy = f.H / 2): void {
  const total = lines.reduce((a, l) => a + l.size * 1.3, 0);
  let y = cy - total / 2;
  keyed(f.center, lines, (l) => l.cls, "text", (e, l) => {
    y += l.size * 1.3;
    set(e, { x: f.W / 2, y: y - l.size * 0.3, "text-anchor": "middle", class: l.cls, "font-size": l.size });
    text(e, l.t);
  });
}

function legendRow(f: Frame, items: (Swatch & { tail?: string })[], show: boolean): number {
  return legendRows(f.legend, show ? items : [], f.W, f.H);
}

// A value for a legend or caption: the full format when it is short, else the compact axis format.
function brief(o: DrawOpts, n: number): string {
  const full = o.fmt(n);
  return full.length <= 8 ? full : o.axisFmt(n);
}

interface Axes {
  left: number;
  top: number;
  pw: number;
  ph: number;
  band: number;
  scale: (n: number) => number;
  px: (i: number) => number;
  d0: number;
  d1: number;
}

function cartesian(f: Frame, o: DrawOpts, xs: Point[], lo: number, hi: number, top: number, right = 16, rightTicks?: { ticks: number[]; fmt: (n: number) => string }): Axes {
  const ticks = niceTicks(lo, hi, 5);
  const d0 = ticks[0] ?? 0;
  const d1 = ticks[ticks.length - 1] ?? 1;
  const left = Math.max(...ticks.map((t) => width(o.axisFmt(t)))) + 14;
  const bottom = 26;
  const pw = Math.max(10, f.W - left - right);
  const ph = Math.max(10, f.H - top - bottom);
  const n = Math.max(1, xs.length);
  const band = pw / n;
  const scale = (x: number) => top + ph - ((x - d0) / (d1 - d0 || 1)) * ph;
  keyed(f.grid, ticks, (t) => String(t), "line", (e, t) => {
    const p = scale(t);
    set(e, { x1: left, x2: left + pw, y1: p, y2: p, class: t === 0 ? "chart__base" : "chart__grid" });
  });
  const labelChars = Math.max(1, ...xs.map((x) => tickLabel(x.label || "-").length));
  const every = Math.max(1, Math.ceil((Math.min(labelChars, 14) * 6.6 + 10) / band));
  const cat = xs
    .map((x, i) => ({ x, i }))
    .filter(({ i }) => i % every === 0)
    .map(({ x, i }) => ({ k: `c${x.key}`, t: catLabel(x.label || "-", band * every - 6), x: left + band * (i + 0.5), y: top + ph + 16, a: "middle" }));
  const val = ticks.map((t) => ({ k: `v${t}`, t: o.axisFmt(t), x: left - 8, y: scale(t) + 4, a: "end" }));
  const alt = rightTicks ? rightTicks.ticks.map((t) => ({ k: `r${t}`, t: rightTicks.fmt(t), x: left + pw + 8, y: top + ph - (t / (rightTicks.ticks[rightTicks.ticks.length - 1] || 1)) * ph + 4, a: "start" })) : [];
  keyed(f.axis, [...val, ...cat, ...alt], (t) => t.k, "text", (e, t) => {
    set(e, { x: t.x, y: t.y, "text-anchor": t.a, class: "chart__tick" });
    text(e, t.t);
  });
  return { left, top, pw, ph, band, scale, px: (i) => left + band * (i + 0.5), d0, d1 };
}

function heatLevel(values: number[], scale: string | undefined): (n: number) => number {
  const pos = values.filter((x) => x > 0).sort((a, b) => a - b);
  const max = pos[pos.length - 1] ?? 0;
  if (!max) return () => 0;
  if (scale === "quantile" && pos.length >= 5) {
    const cut = [0.2, 0.4, 0.6, 0.8].map((q) => pos[Math.min(pos.length - 1, Math.floor(q * pos.length))] ?? max);
    return (n) => (n <= 0 ? 0 : 1 + cut.filter((c) => n > c).length);
  }
  return (n) => (n <= 0 ? 0 : Math.max(1, Math.min(5, Math.ceil((n / max) * 5))));
}

export interface Cell {
  key: string;
  x: string;
  s?: string;
  px: number;
  py: number;
  w: number;
  h: number;
  v: number;
  tip: string;
}

export interface GridLabel {
  k: string;
  x: number;
  y: number;
  t: string;
  a: "start" | "middle" | "end";
}

export function paintCells(f: Pick<Frame, "marks" | "axis" | "labels">, cells: Cell[], gl: GridLabel[], scale: string | undefined, fmt: (n: number) => string, showValues = false): void {
  const level = heatLevel(
    cells.map((c) => c.v),
    scale,
  );
  keyed(f.marks, cells, (c) => c.key, "rect", (e, c) => {
    set(e, { x: c.px, y: c.py, width: Math.max(0, c.w), height: Math.max(0, c.h), rx: Math.min(3, c.w / 4), class: "chart__mark chart__cell", "data-level": String(level(c.v)), "data-x": c.x, "data-s": c.s ?? null });
    tip(e, c.tip);
  });
  keyed(f.axis, gl, (l) => l.k, "text", (e, l) => {
    set(e, { x: l.x, y: l.y, "text-anchor": l.a, class: "chart__tick" });
    text(e, l.t);
  });
  const vals = showValues ? cells.filter((c) => c.v > 0 && c.w >= 26 && c.h >= 16) : [];
  keyed(f.labels, vals, (c) => c.key, "text", (e, c) => {
    set(e, { x: c.px + c.w / 2, y: c.py + c.h / 2 + 4, "text-anchor": "middle", class: "chart__value chart__value--cell", "data-level": String(level(c.v)) });
    text(e, fmt(c.v));
  });
}

const pad2 = (n: number) => String(n).padStart(2, "0");
export const dayKey = (d: Date) => `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
export function parseDay(k: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(k);
  return m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : null;
}
const addDays = (d: Date, n: number) => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);

function fmtDate(d: Date, o: Intl.DateTimeFormatOptions): string {
  try {
    return new Intl.DateTimeFormat(locale(), o).format(d);
  } catch {
    return new Intl.DateTimeFormat("en", o).format(d);
  }
}

export type CalDomain = "year" | "month" | "week" | "day";

export interface CalOpts {
  domain: CalDomain;
  start: Date;
  count: number;
  mondayFirst: boolean;
  scale?: string | undefined;
  labels?: boolean | undefined;
}

export function calendarRange(c: CalOpts): { from: Date; to: Date } {
  const s = c.start;
  switch (c.domain) {
    case "year":
      return { from: new Date(s.getFullYear(), 0, 1), to: new Date(s.getFullYear() + Math.max(1, c.count), 0, 1) };
    case "month":
      return { from: new Date(s.getFullYear(), s.getMonth(), 1), to: new Date(s.getFullYear(), s.getMonth() + Math.max(1, c.count), 1) };
    case "week": {
      const from = addDays(s, -((s.getDay() - (c.mondayFirst ? 1 : 0) + 7) % 7));
      return { from, to: addDays(from, 7 * Math.max(1, c.count)) };
    }
    default:
      return { from: new Date(s.getFullYear(), s.getMonth(), s.getDate()), to: addDays(s, Math.max(1, c.count)) };
  }
}

export function layoutCalendar(W: number, H: number, c: CalOpts, value: (day: string, hour?: string) => number, fmt: (n: number) => string): { cells: Cell[]; labels: GridLabel[] } {
  const cells: Cell[] = [];
  const labels: GridLabel[] = [];
  const { from, to } = calendarRange(c);
  const row = (d: Date) => (d.getDay() - (c.mondayFirst ? 1 : 0) + 7) % 7;
  const wdName = (i: number) => fmtDate(new Date(2024, 0, (c.mondayFirst ? 1 : 7) + i), { weekday: "short" });
  const dayTip = (d: Date, n: number, hour?: number) => `${fmtDate(d, { weekday: "short", month: "short", day: "numeric", year: "numeric" })}${hour !== undefined ? ` ${pad2(hour)}:00` : ""}: ${fmt(n)}`;
  if (c.domain === "year" || c.domain === "month") {
    const blocks: { from: Date; to: Date; label: string }[] = [];
    if (c.domain === "year") {
      for (let y = from.getFullYear(); y < to.getFullYear(); y++) blocks.push({ from: new Date(y, 0, 1), to: new Date(y + 1, 0, 1), label: String(y) });
    } else {
      for (let d = new Date(from); d < to; d = new Date(d.getFullYear(), d.getMonth() + 1, 1)) blocks.push({ from: d, to: new Date(d.getFullYear(), d.getMonth() + 1, 1), label: fmtDate(d, { month: "short", year: blocks.length === 0 || d.getMonth() === 0 ? "numeric" : undefined }) });
    }
    const weeksOf = (b: { from: Date; to: Date }) => Math.ceil((row(b.from) + Math.round((b.to.getTime() - b.from.getTime()) / 864e5)) / 7);
    const totalWeeks = blocks.reduce((a, b) => a + weeksOf(b), 0);
    const gapCols = c.domain === "month" ? blocks.length - 1 : 0;
    const left = 34;
    const top = c.domain === "year" && blocks.length > 1 ? 18 : 18;
    const rowsPer = 7;
    const yearRows = c.domain === "year" ? blocks.length : 1;
    const step = Math.max(6, Math.min(22, (W - left - 8) / (c.domain === "year" ? Math.max(...blocks.map(weeksOf)) : totalWeeks + gapCols * 0.6), (H - top - 6 - (yearRows - 1) * 22) / (rowsPer * yearRows)));
    const size = step - Math.max(1.5, step * 0.14);
    let colX = left;
    blocks.forEach((b, bi) => {
      const baseY = c.domain === "year" ? top + bi * (rowsPer * step + 22) : top;
      if (c.domain === "year") colX = left;
      const startRow = row(b.from);
      let lastMonth = -1;
      for (let d = new Date(b.from); d < b.to; d = addDays(d, 1)) {
        const offset = Math.round((d.getTime() - b.from.getTime()) / 864e5) + startRow;
        const wi = Math.floor(offset / 7);
        const k = dayKey(d);
        const n = value(k);
        const x = colX + wi * step;
        const y = baseY + row(d) * step;
        cells.push({ key: k, x: k, px: x, py: y, w: size, h: size, v: n, tip: dayTip(d, n) });
        if (c.domain === "year" && d.getMonth() !== lastMonth && row(d) <= 3) {
          lastMonth = d.getMonth();
          labels.push({ k: `m${b.label}-${lastMonth}`, x, y: baseY - 6, t: fmtDate(d, { month: "short" }), a: "start" });
        }
      }
      if (c.domain === "month") labels.push({ k: `b${bi}`, x: colX, y: baseY - 6, t: b.label, a: "start" });
      if (c.domain === "year" && blocks.length > 1) labels.push({ k: `y${b.label}`, x: 2, y: baseY - 6, t: b.label, a: "start" });
      for (const r of [0, 2, 4]) if (bi === 0 || c.domain === "year") labels.push({ k: `w${bi}-${r}`, x: left - 6, y: baseY + r * step + size * 0.75, t: wdName(r), a: "end" });
      colX += weeksOf(b) * step + (c.domain === "month" ? step * 0.6 : 0);
    });
    return { cells, labels };
  }
  const days: Date[] = [];
  for (let d = new Date(from); d < to; d = addDays(d, 1)) days.push(d);
  if (c.domain === "day") {
    const left = 92;
    const top = 18;
    const stepX = Math.max(6, Math.min(28, (W - left - 8) / 24));
    const stepY = Math.max(6, Math.min(24, (H - top - 6) / Math.max(1, days.length)));
    days.forEach((d, di) => {
      const k = dayKey(d);
      for (let hr = 0; hr < 24; hr++) {
        const hk = `h${pad2(hr)}`;
        const n = value(k, hk);
        cells.push({ key: `${k}|${hk}`, x: k, s: hk, px: left + hr * stepX, py: top + di * stepY, w: stepX - 2, h: stepY - 2, v: n, tip: dayTip(d, n, hr) });
      }
      labels.push({ k: `d${k}`, x: left - 8, y: top + di * stepY + stepY * 0.7, t: fmtDate(d, { weekday: "short", month: "short", day: "numeric" }), a: "end" });
    });
    for (let hr = 0; hr < 24; hr += 3) labels.push({ k: `h${hr}`, x: left + hr * stepX + stepX / 2, y: top - 6, t: pad2(hr), a: "middle" });
    return { cells, labels };
  }
  const weeks = Math.ceil(days.length / 7);
  const left = 34;
  const top = 34;
  const gap = 10;
  const stepX = Math.max(5, Math.min(26, (W - left - 8 - (weeks - 1) * gap) / (weeks * 7)));
  const stepY = Math.max(5, Math.min(14, (H - top - 6) / 24));
  for (let wi = 0; wi < weeks; wi++) {
    const bx = left + wi * (7 * stepX + gap);
    const first = days[wi * 7];
    if (first) labels.push({ k: `wk${wi}`, x: bx, y: 12, t: fmtDate(first, { month: "short", day: "numeric" }), a: "start" });
    for (let di = 0; di < 7; di++) {
      const d = days[wi * 7 + di];
      if (!d) continue;
      const k = dayKey(d);
      if (wi === 0) labels.push({ k: `wd${di}`, x: bx + di * stepX + stepX / 2, y: top - 6, t: wdName(di).slice(0, 1), a: "middle" });
      for (let hr = 0; hr < 24; hr++) {
        const hk = `h${pad2(hr)}`;
        const n = value(k, hk);
        cells.push({ key: `${k}|${hk}`, x: k, s: hk, px: bx + di * stepX, py: top + hr * stepY, w: stepX - 1.5, h: stepY - 1.5, v: n, tip: dayTip(d, n, hr) });
      }
    }
  }
  for (const hr of [0, 6, 12, 18]) labels.push({ k: `hh${hr}`, x: left - 6, y: top + hr * stepY + stepY * 0.8, t: pad2(hr), a: "end" });
  return { cells, labels };
}

export function drawExtra(svg: SVGSVGElement, res: ChartResult, spec: ChartSpec, o: DrawOpts, W: number, H: number): void {
  const f: Frame = { W, H, grid: layer(svg, "grid"), axis: layer(svg, "axis"), marks: layer(svg, "marks"), labels: layer(svg, "labels"), legend: layer(svg, "legend"), center: layer(svg, "center") };
  const empty = () => {
    clear(f.grid, f.axis, f.marks, f.labels, f.legend);
    centerText(f, [{ t: o.empty, size: 13, cls: "chart__caption" }]);
  };
  if (spec.kind === "gauge") return gauge(f, res, spec, o);
  if (spec.kind === "dots") return dots(f, res, spec, o, empty);
  if (spec.kind === "calendar") {
    clear(f.grid, f.legend, f.center);
    const map = new Map<string, number>();
    const s = res.series[0];
    res.x.forEach((x, i) => map.set(x.key, v(s, i)));
    let last: Date | null = null;
    for (const x of res.x) {
      const d = parseDay(x.key);
      if (d && (!last || d > last)) last = d;
    }
    const { cells, labels } = layoutCalendar(W, H, { domain: "year", start: last ?? new Date(), count: 1, mondayFirst: true, scale: spec.scale }, (k) => map.get(k) ?? 0, o.fmt);
    paintCells(f, cells, labels, spec.scale, o.fmt);
    return;
  }
  if (!res.x.length || !res.series.length) return empty();
  clear(f.center);
  if (spec.kind === "radial") return radial(f, res, spec, o);
  if (spec.kind === "heatmap") return matrix(f, res, spec, o);
  if (spec.kind === "histogram") return histogram(f, res, spec, o);
  if (spec.kind === "compare") return compare(f, res, spec, o, empty);
  return area(f, res, spec, o);
}

function area(f: Frame, res: ChartResult, spec: ChartSpec, o: DrawOpts): void {
  const xs = res.x;
  const series = res.series;
  const multi = series.length > 1;
  const stacked = !!spec.stacked && multi;
  const legendH = legendRow(
    f,
    multi ? series.map((s, i) => ({ key: s.key, label: s.label, color: s.color, pal: i })) : [],
    spec.legend !== false,
  );
  const base: number[] = xs.map(() => 0);
  const tops = series.map((s) =>
    xs.map((_, i) => {
      const b = stacked ? (base[i] ?? 0) : 0;
      const t = b + v(s, i);
      if (stacked) base[i] = t;
      return [b, t] as [number, number];
    }),
  );
  let lo = 0;
  let hi = 0;
  for (const s of tops) for (const [b, t] of s) (lo = Math.min(lo, b, t)), (hi = Math.max(hi, b, t));
  const ax = cartesian(f, o, xs, lo, hi, legendH + (spec.labels ? 22 : 12));
  const path = (pts: [number, number][]) => (spec.smooth ? monotone(pts) : linear(pts));
  const items = tops.map((pairs, si) => {
    const up = pairs.map(([, t], i) => [ax.px(i), ax.scale(t)] as [number, number]);
    const down = pairs.map(([b], i) => [ax.px(i), ax.scale(b)] as [number, number]).reverse();
    return { s: series[si] as Series, si, up, down };
  });
  keyed(f.marks, [...items.map((it) => ({ ...it, kind: "a" })), ...items.map((it) => ({ ...it, kind: "l" }))], (it) => `${it.kind}:${it.s.key}`, "path", (e, it) => {
    if (it.kind === "a") {
      const back = spec.smooth ? monotone(it.down).replace(/^M/, "L") : it.down.map((p) => `L${p[0].toFixed(1)},${p[1].toFixed(1)}`).join("");
      set(e, { d: `${path(it.up)}${back}Z`, class: "chart__area chart__area--solid" });
    } else set(e, { d: path(it.up), class: "chart__line" });
    paintSwatch(e, { key: it.s.key, label: it.s.label, color: multi ? it.s.color : undefined, pal: it.si });
  });
  const dots = xs.length <= 60 ? items.flatMap((it) => it.up.map((p, i) => ({ it, p, x: xs[i] as Point, val: v(it.s, i) }))) : [];
  keyed(f.labels, dots, (d) => `${d.it.s.key}:${d.x.key}`, "circle", (c, d) => {
    set(c, { cx: d.p[0], cy: d.p[1], r: 3, class: "chart__dot", "data-x": d.x.key, "data-s": multi ? d.it.s.key : null });
    paintSwatch(c, { key: d.it.s.key, label: "", color: multi ? d.it.s.color : undefined, pal: d.it.si });
    tip(c, `${d.x.label || "-"}${multi ? ` · ${d.it.s.label || "-"}` : ""}: ${o.fmt(d.val)}`);
  });
}

function radial(f: Frame, res: ChartResult, spec: ChartSpec, o: DrawOpts): void {
  clear(f.grid, f.axis);
  const s = res.series[0];
  const xs = res.x.slice(0, 12);
  const vals = xs.map((_, i) => Math.max(0, v(s, i)));
  const max = spec.target && spec.target > 0 ? Math.max(spec.target, ...vals) : Math.max(...vals, 0) || 1;
  const total = !vals.length
    ? 0
    : spec.fn === "average" || spec.fn === "median" || !!spec.fn?.startsWith("percent")
      ? vals.reduce((a, b) => a + b, 0) / vals.length
      : spec.fn === "min"
        ? Math.min(...vals)
        : spec.fn === "max"
          ? Math.max(...vals)
          : vals.reduce((a, b) => a + b, 0);
  const legendH = legendRow(
    f,
    xs.map((x, i) => ({ key: x.key, label: x.label || "-", tail: ` · ${brief(o, vals[i] ?? 0)}`, color: x.color, pal: i })),
    spec.legend !== false,
  );
  const top = legendH + 8;
  const R = Math.max(30, Math.min(f.W - 16, f.H - top - 8) / 2);
  const cx = f.W / 2;
  const cy = top + (f.H - top) / 2;
  const inner = R * 0.28;
  const ring = Math.max(4, Math.min(22, (R - inner) / xs.length - 3));
  const a0 = 0;
  const span = Math.PI * 1.5;
  const rings = xs.map((x, i) => ({ x, i, r: R - i * (ring + 3), val: vals[i] ?? 0 }));
  keyed(f.grid, rings, (r) => r.x.key, "path", (e, r) => set(e, { d: arc(cx, cy, r.r, r.r - ring, a0, a0 + span), class: "chart__track" }));
  keyed(f.marks, rings.filter((r) => r.val > 0), (r) => r.x.key, "path", (e, r) => {
    set(e, { d: arc(cx, cy, r.r, r.r - ring, a0, a0 + span * (r.val / max)), class: "chart__mark chart__ring", "data-x": r.x.key });
    paintSwatch(e, { key: r.x.key, label: r.x.label, color: r.x.color, pal: r.i });
    tip(e, `${r.x.label || "-"}: ${o.fmt(r.val)}`);
  });
  const shown = spec.labels ? rings : [];
  keyed(f.labels, shown, (r) => r.x.key, "text", (e, r) => {
    set(e, { x: cx - 6, y: cy - r.r + ring * 0.75, "text-anchor": "end", class: "chart__value" });
    text(e, clip(`${r.x.label || "-"} ${o.fmt(r.val)}`, cx - 12));
  });
  const big = fitText([o.fmt(total), o.axisFmt(total)], () => inner * 2.4, Math.max(14, Math.min(26, inner / 1.6)), 13, 9);
  keyed(f.center, [big.t, clip(o.caption, inner * 2.4, 12)], (_, i) => String(i), "text", (e, t, i) => {
    set(e, { x: cx, y: cy + (i ? 16 : -2) + R * 0.12, "text-anchor": "middle", class: i ? "chart__caption" : "chart__total", "font-size": i ? 12 : big.size });
    text(e, t);
  });
}

function gauge(f: Frame, res: ChartResult, spec: ChartSpec, o: DrawOpts): void {
  clear(f.grid, f.axis, f.legend);
  const value = res.value;
  const target = spec.target && spec.target > 0 ? spec.target : 0;
  const max = target ? Math.max(target, value) * (value > target ? 1.1 : 1) : (niceTicks(0, Math.max(1, value), 4).pop() ?? 1);
  const R = Math.max(30, Math.min(f.W / 2 - 12, f.H - 48));
  const cx = f.W / 2;
  const cy = Math.min(f.H - 34, f.H / 2 + R / 2);
  const thick = Math.max(10, R * 0.2);
  const a0 = -Math.PI / 2;
  const share = Math.max(0, Math.min(1, value / max));
  keyed(f.grid, ["track"], (k) => k, "path", (e) => set(e, { d: arc(cx, cy, R, R - thick, a0, a0 + Math.PI), class: "chart__track" }));
  keyed(f.marks, share > 0 ? ["value"] : [], (k) => k, "path", (e) => {
    set(e, { d: arc(cx, cy, R, R - thick, a0, a0 + Math.PI * share), class: "chart__mark chart__gauge", "data-state": target ? (value >= target ? "met" : "under") : null });
    tip(e, `${o.caption}: ${o.fmt(value)}${target ? ` / ${o.fmt(target)}` : ""}`);
  });
  const tickAt = target ? a0 + Math.PI * Math.min(1, target / max) : 0;
  keyed(f.labels, target ? ["target"] : [], (k) => k, "line", (e) => {
    set(e, { x1: cx + (R + 4) * Math.sin(tickAt), y1: cy - (R + 4) * Math.cos(tickAt), x2: cx + (R - thick - 4) * Math.sin(tickAt), y2: cy - (R - thick - 4) * Math.cos(tickAt), class: "chart__target" });
  });
  keyed(f.axis, [{ k: "lo", t: o.axisFmt(0), x: cx - R + thick / 2 }, { k: "hi", t: o.axisFmt(max), x: cx + R - thick / 2 }], (t) => t.k, "text", (e, t) => {
    set(e, { x: t.x, y: cy + 16, "text-anchor": "middle", class: "chart__tick" });
    text(e, t.t);
  });
  // centerText puts the big value's cap line about 0.65 * (size + 12) - 0.25 * size above the block centre,
  // which sits R * 0.28 above the hub; the value must fit the arc's inner chord at that height.
  const ri = R - thick;
  const room = ri * 1.7;
  const big = fitText([o.fmt(value), o.axisFmt(value)], (s) => Math.min(room, chord(ri, R * 0.28 + 0.65 * (s + 12) - 0.25 * s) - 14), Math.min(40, Math.max(20, R / 3)), 16, 11);
  const pct = target ? `${Math.round((value / target) * 100)}% of ` : "";
  const cap = target ? (width(`${pct}${o.fmt(target)}`, 12) <= room ? `${pct}${o.fmt(target)}` : `${pct}${o.axisFmt(target)}`) : clip(o.caption, room, 12);
  centerText(
    f,
    [
      { t: big.t, size: big.size, cls: "chart__big" },
      { t: cap, size: 12, cls: "chart__caption" },
    ],
    cy - R * 0.28,
  );
}

function histogram(f: Frame, res: ChartResult, spec: ChartSpec, o: DrawOpts): void {
  clear(f.legend);
  const xs = res.x;
  const s = res.series[0];
  const vals = xs.map((_, i) => v(s, i));
  const total = vals.reduce((a, b) => a + Math.max(0, b), 0) || 1;
  let run = 0;
  const cum = vals.map((x) => ((run += Math.max(0, x)), (run / total) * 100));
  const pctTicks = [0, 25, 50, 75, 100];
  const ax = cartesian(f, o, xs, 0, Math.max(...vals, 0), spec.labels ? 22 : 12, 44, { ticks: pctTicks, fmt: (n) => `${n}%` });
  keyed(f.marks, xs, (x) => `b:${x.key}`, "rect", (e, x) => {
    const i = xs.indexOf(x);
    const y = ax.scale(Math.max(0, vals[i] ?? 0));
    set(e, { x: ax.left + ax.band * i + 0.5, y, width: Math.max(0, ax.band - 1), height: Math.max(0, ax.scale(0) - y), class: "chart__mark chart__bar", "data-x": x.key, "data-pal": "0" });
    tip(e, `${x.label || "-"}: ${o.fmt(vals[i] ?? 0)} (${Math.round(((vals[i] ?? 0) / total) * 100)}%)`);
  });
  const pts = cum.map((c, i) => [ax.px(i), ax.top + ax.ph - (c / 100) * ax.ph] as [number, number]);
  keyed(f.labels, ["cum"], (k) => k, "path", (e) => {
    set(e, { d: spec.smooth ? monotone(pts) : linear(pts), class: "chart__line chart__line--alt", "data-pal": "3" });
    tip(e, "Cumulative share");
  });
}

function compare(f: Frame, res: ChartResult, spec: ChartSpec, o: DrawOpts, empty: () => void): void {
  const xs = res.x;
  const s = res.series[0];
  const n = xs.length;
  const N = Math.min(Math.floor(n / 2), spec.window && spec.window > 0 ? spec.window : Math.floor(n / 2));
  if (N < 1) return empty();
  const cur = Array.from({ length: N }, (_, i) => v(s, n - N + i));
  const prev = Array.from({ length: N }, (_, i) => v(s, n - 2 * N + i));
  const sum = (a: number[]) => a.reduce((x, y) => x + y, 0);
  const sc = sum(cur);
  const sp = sum(prev);
  const raw = sp ? ((sc - sp) / Math.abs(sp)) * 100 : null;
  const change = raw === null ? null : Math.abs(raw) >= 10 ? Math.round(raw) : Math.round(raw * 10) / 10;
  const legendH = legendRow(
    f,
    [
      { key: "cur", label: "This period", tail: ` · ${brief(o, sc)}${change === null ? "" : ` (${change > 0 ? "+" : ""}${change}%)`}`, pal: 0 },
      { key: "prev", label: "Previous", tail: ` · ${brief(o, sp)}`, pal: 1 },
    ],
    true,
  );
  const shown = xs.slice(n - N);
  const ax = cartesian(f, o, shown, Math.min(0, ...cur, ...prev), Math.max(0, ...cur, ...prev), legendH + 12);
  const line = (vals: number[]) => {
    const pts = vals.map((x, i) => [ax.px(i), ax.scale(x)] as [number, number]);
    return spec.smooth ? monotone(pts) : linear(pts);
  };
  keyed(f.marks, ["prev", "cur"], (k) => k, "path", (e, k) => {
    set(e, { d: line(k === "cur" ? cur : prev), class: k === "cur" ? "chart__line" : "chart__line chart__line--prev", "data-pal": k === "cur" ? "0" : "1" });
  });
  keyed(f.labels, shown.length <= 60 ? shown : [], (x) => x.key, "circle", (c, x) => {
    const i = shown.indexOf(x);
    set(c, { cx: ax.px(i), cy: ax.scale(cur[i] ?? 0), r: 3, class: "chart__dot", "data-x": x.key, "data-pal": "0" });
    tip(c, `${x.label || "-"}: ${o.fmt(cur[i] ?? 0)} (previous ${o.fmt(prev[i] ?? 0)})`);
  });
}

function matrix(f: Frame, res: ChartResult, spec: ChartSpec, o: DrawOpts): void {
  clear(f.grid, f.legend);
  const xs = res.x;
  const rows = res.series;
  const rowLabel = Math.min(140, Math.max(24, ...rows.map((s) => width(s.label || "-") + 12)));
  const left = rows.length > 1 || rows[0]?.label ? rowLabel : 8;
  const top = 8;
  const bottom = 26;
  const cw = Math.max(4, (f.W - left - 8) / Math.max(1, xs.length));
  const ch = Math.max(8, Math.min(40, (f.H - top - bottom) / Math.max(1, rows.length)));
  const cells: Cell[] = [];
  rows.forEach((s, ri) =>
    xs.forEach((x, xi) => {
      const val = v(s, xi);
      cells.push({ key: `${x.key}|${s.key}`, x: x.key, ...(s.key ? { s: s.key } : {}), px: left + xi * cw + 1, py: top + ri * ch + 1, w: cw - 2, h: ch - 2, v: val, tip: `${x.label || "-"}${s.label ? ` · ${s.label}` : ""}: ${o.fmt(val)}` });
    }),
  );
  const every = Math.max(1, Math.ceil((Math.min(12, Math.max(1, ...xs.map((x) => tickLabel(x.label || "-").length))) * 6.6 + 8) / cw));
  const gl: GridLabel[] = [
    ...rows.map((s, ri) => ({ k: `r${s.key}`, x: left - 6, y: top + ri * ch + ch / 2 + 4, t: clip(s.label || "-", left - 10), a: "end" as const })),
    ...xs.filter((_, i) => i % every === 0).map((x) => ({ k: `c${x.key}`, x: left + xs.indexOf(x) * cw + cw / 2, y: top + rows.length * ch + 16, t: catLabel(x.label || "-", cw * every - 4), a: "middle" as const })),
  ];
  paintCells(f, cells, gl, spec.scale, o.fmt, !!spec.labels);
}

function dots(f: Frame, res: ChartResult, spec: ChartSpec, o: DrawOpts, empty: () => void): void {
  const pts = res.points ?? [];
  if (!pts.length) return empty();
  clear(f.legend, f.center);
  const groups = res.x.length ? res.x : [{ key: "", label: "" }];
  const idx = new Map(groups.map((g, i) => [g.key, i]));
  const dmin = Math.min(...pts.map((p) => p.value));
  const dmax = Math.max(...pts.map((p) => p.value));
  const tight = dmin > 0 && dmin > dmax * 0.4;
  const lo = tight ? dmin - (dmax - dmin) * 0.1 : Math.min(0, dmin);
  const hi = Math.max(0, dmax);
  const diverging = lo < 0 && hi > 0;
  const ticks = niceTicks(lo, hi, 6);
  const d0 = ticks[0] ?? 0;
  const d1 = ticks[ticks.length - 1] ?? 1;
  const left = res.x.length ? Math.min(150, Math.max(40, ...groups.map((g) => width(g.label || "-") + 14))) : 16;
  const right = 16;
  const top = 10;
  const bottom = 26;
  const pw = Math.max(10, f.W - left - right);
  const ph = Math.max(10, f.H - top - bottom);
  const band = ph / groups.length;
  const sx = (n: number) => left + ((n - d0) / (d1 - d0 || 1)) * pw;
  const r = pts.length > 900 ? 2 : pts.length > 300 ? 2.8 : 3.6;
  keyed(f.grid, ticks, (t) => String(t), "line", (e, t) => set(e, { x1: sx(t), x2: sx(t), y1: top, y2: top + ph, class: t === 0 && diverging ? "chart__base" : "chart__grid" }));
  keyed(
    f.axis,
    [...ticks.map((t) => ({ k: `v${t}`, x: sx(t), y: top + ph + 16, t: o.axisFmt(t), a: "middle" })), ...(res.x.length ? groups.map((g, i) => ({ k: `g${g.key}`, x: left - 8, y: top + band * (i + 0.5) + 4, t: clip(g.label || "-", left - 12), a: "end" })) : [])],
    (t) => t.k,
    "text",
    (e, t) => {
      set(e, { x: t.x, y: t.y, "text-anchor": t.a, class: "chart__tick" });
      text(e, t.t);
    },
  );
  const placed: { id: string; x: number; y: number; p: (typeof pts)[number] }[] = [];
  const lanes = new Map<number, { x: number; y: number }[]>();
  const sorted = [...pts].sort((a, b) => a.value - b.value);
  for (const p of sorted) {
    const gi = idx.get(p.group) ?? 0;
    const mid = top + band * (gi + 0.5);
    const lane = lanes.get(gi) ?? [];
    lanes.set(gi, lane);
    const x = sx(p.value);
    const near = lane.filter((q) => Math.abs(q.x - x) < r * 2);
    let y = mid;
    for (let k = 0; k < 400; k++) {
      const off = k === 0 ? 0 : Math.ceil(k / 2) * (k % 2 ? 1 : -1) * r * 0.9;
      const cy = mid + off;
      if (Math.abs(off) > band / 2 - r) {
        let hsh = 0;
        for (let j = 0; j < p.id.length; j++) hsh = (hsh * 31 + p.id.charCodeAt(j)) >>> 0;
        y = mid + ((hsh % 1000) / 1000 - 0.5) * (band - 2 * r);
        break;
      }
      if (near.every((q) => (q.x - x) ** 2 + (q.y - cy) ** 2 >= (r * 2) ** 2)) {
        y = cy;
        break;
      }
    }
    lane.push({ x, y });
    placed.push({ id: p.id, x, y, p });
  }
  keyed(f.marks, placed, (d) => d.id, "circle", (c, d) => {
    set(c, { cx: d.x, cy: d.y, r, class: "chart__mark chart__bee", "data-row": d.id, "data-sign": diverging ? (d.p.value < 0 ? "neg" : "pos") : null, "data-pal": diverging ? null : String((idx.get(d.p.group) ?? 0) % 10) });
    tip(c, `${d.p.title || "Untitled"}: ${o.fmt(d.p.value)}`);
  });
  clear(f.labels);
}
