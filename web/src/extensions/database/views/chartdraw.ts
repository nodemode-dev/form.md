import type { ChartResult, ChartSpec } from "../../../engine/types";
import { drawExtra, EXTRA_KINDS } from "./chartkinds";

const NS = "http://www.w3.org/2000/svg";
const LAYERS = ["grid", "axis", "marks", "labels", "legend", "center"] as const;
type Layer = (typeof LAYERS)[number];
type Attrs = Record<string, string | number | null>;

export interface DrawOpts {
  width: number;
  height: number;
  fmt: (n: number) => string;
  axisFmt: (n: number) => string;
  empty: string;
  caption: string;
}

function el<K extends keyof SVGElementTagNameMap>(tag: K): SVGElementTagNameMap[K] {
  return document.createElementNS(NS, tag);
}

function set(e: Element, attrs: Attrs): void {
  for (const [k, v] of Object.entries(attrs)) {
    if (v === null) {
      if (e.hasAttribute(k)) e.removeAttribute(k);
      continue;
    }
    const s = typeof v === "number" ? String(Math.round(v * 100) / 100) : v;
    if (e.getAttribute(k) !== s) e.setAttribute(k, s);
  }
}

function text(e: Element, s: string): void {
  if (e.textContent !== s) e.textContent = s;
}

function layer(svg: SVGSVGElement, name: Layer): SVGGElement {
  let g = svg.querySelector<SVGGElement>(`:scope > g[data-layer="${name}"]`);
  if (!g) {
    for (const n of LAYERS) {
      if (svg.querySelector(`:scope > g[data-layer="${n}"]`)) continue;
      const x = el("g");
      x.setAttribute("data-layer", n);
      svg.append(x);
    }
    g = svg.querySelector<SVGGElement>(`:scope > g[data-layer="${name}"]`) as SVGGElement;
  }
  return g;
}

function keyed<T, K extends keyof SVGElementTagNameMap>(g: SVGGElement, items: readonly T[], key: (t: T, i: number) => string, tag: K, paint: (e: SVGElementTagNameMap[K], t: T, i: number) => void): void {
  const existing = new Map<string, Element>();
  for (const c of Array.from(g.children)) existing.set(c.getAttribute("data-key") ?? "", c);
  let cursor: Element | null = g.firstElementChild;
  items.forEach((item, i) => {
    const k = `${tag}:${key(item, i)}`;
    let e = existing.get(k) as SVGElementTagNameMap[K] | undefined;
    if (e) existing.delete(k);
    else {
      e = el(tag);
      e.setAttribute("data-key", k);
    }
    paint(e, item, i);
    if (e !== cursor) g.insertBefore(e, cursor);
    else cursor = cursor.nextElementSibling;
  });
  for (const e of existing.values()) e.remove();
}

function tip(e: Element, s: string): void {
  let t = e.firstElementChild;
  if (!t || t.tagName !== "title") {
    t = el("title");
    e.prepend(t);
  }
  text(t, s);
}

function width(s: string, size = 11): number {
  return s.length * size * 0.6;
}

function clip(s: string, room: number, size = 11): string {
  const max = Math.max(1, Math.floor(room / (size * 0.6)));
  return s.length <= max ? s : `${s.slice(0, Math.max(1, max - 1))}…`;
}

export function niceTicks(lo: number, hi: number, count = 5): number[] {
  if (lo === hi) hi = lo + 1;
  const raw = (hi - lo) / count;
  const mag = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 2.5, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? raw;
  const out: number[] = [];
  for (let v = Math.floor(lo / step) * step; v <= hi + step * 1e-9; v += step) out.push(Math.round(v / step) * step);
  const last = out[out.length - 1] ?? lo;
  if (last < hi) out.push(last + step);
  return out;
}

function linear(pts: [number, number][]): string {
  return pts.map((p, i) => `${i ? "L" : "M"}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join("");
}

function monotone(pts: [number, number][]): string {
  const n = pts.length;
  if (n < 3) return linear(pts);
  const X = pts.map((p) => p[0]);
  const Y = pts.map((p) => p[1]);
  const at = (a: number[], i: number) => a[i] ?? 0;
  const dx: number[] = [];
  const m: number[] = [];
  for (let i = 0; i < n - 1; i++) {
    dx.push(at(X, i + 1) - at(X, i) || 1e-6);
    m.push((at(Y, i + 1) - at(Y, i)) / at(dx, i));
  }
  const t: number[] = [at(m, 0)];
  for (let i = 1; i < n - 1; i++) t.push(at(m, i - 1) * at(m, i) <= 0 ? 0 : (at(m, i - 1) + at(m, i)) / 2);
  t.push(at(m, n - 2));
  for (let i = 0; i < n - 1; i++) {
    const mi = at(m, i);
    if (mi === 0) {
      t[i] = 0;
      t[i + 1] = 0;
      continue;
    }
    const a = at(t, i) / mi;
    const b = at(t, i + 1) / mi;
    const s = a * a + b * b;
    if (s > 9) {
      const k = 3 / Math.sqrt(s);
      t[i] = k * a * mi;
      t[i + 1] = k * b * mi;
    }
  }
  const f = (v: number) => v.toFixed(1);
  let d = `M${f(at(X, 0))},${f(at(Y, 0))}`;
  for (let i = 0; i < n - 1; i++) {
    const h = at(dx, i) / 3;
    d += `C${f(at(X, i) + h)},${f(at(Y, i) + at(t, i) * h)},${f(at(X, i + 1) - h)},${f(at(Y, i + 1) - at(t, i + 1) * h)},${f(at(X, i + 1))},${f(at(Y, i + 1))}`;
  }
  return d;
}

function arc(cx: number, cy: number, r: number, ir: number, a0: number, a1: number): string {
  const sweep = Math.min(a1 - a0, Math.PI * 2 - 1e-4);
  const e = a0 + sweep;
  const large = sweep > Math.PI ? 1 : 0;
  const p = (rad: number, a: number) => `${(cx + rad * Math.sin(a)).toFixed(2)},${(cy - rad * Math.cos(a)).toFixed(2)}`;
  return `M${p(r, a0)}A${r},${r} 0 ${large} 1 ${p(r, e)}L${p(ir, e)}A${ir},${ir} 0 ${large} 0 ${p(ir, a0)}Z`;
}

interface Swatch {
  key: string;
  label: string;
  color?: string | undefined;
  pal: number;
}

type Point = ChartResult["x"][number];
type Series = ChartResult["series"][number];

const NONE: Point = { key: "", label: "" };

function val(s: Series, i: number): number {
  return s.values[i] ?? 0;
}

function paintSwatch(e: SVGElement, s: Swatch): void {
  set(e, { "data-color": s.color && s.color !== "default" ? s.color : null, "data-pal": s.color && s.color !== "default" ? null : String(s.pal % 10) });
}

export function drawChart(svg: SVGSVGElement, res: ChartResult | null, spec: ChartSpec, o: DrawOpts): void {
  const W = Math.max(120, o.width);
  const H = Math.max(120, o.height);
  set(svg, { viewBox: `0 0 ${W} ${H}`, width: W, height: H, "data-kind": spec.kind });
  const grid = layer(svg, "grid");
  const axis = layer(svg, "axis");
  const marks = layer(svg, "marks");
  const labels = layer(svg, "labels");
  const legend = layer(svg, "legend");
  const center = layer(svg, "center");
  const clear = (...gs: SVGGElement[]) => {
    for (const g of gs) if (g.firstChild) g.replaceChildren();
  };
  const centerText = (lines: { t: string; size: number; cls: string }[]) => {
    const total = lines.reduce((a, l) => a + l.size * 1.3, 0);
    let y = H / 2 - total / 2;
    keyed(center, lines, (l) => l.cls, "text", (e, l) => {
      y += l.size * 1.3;
      set(e, { x: W / 2, y: y - l.size * 0.3, "text-anchor": "middle", class: l.cls, "font-size": l.size });
      text(e, l.t);
    });
  };
  if (!res) {
    clear(grid, axis, marks, labels, legend, center);
    return;
  }
  if (EXTRA_KINDS.has(spec.kind)) {
    drawExtra(svg, res, spec, o, W, H);
    return;
  }
  if (spec.kind === "number") {
    clear(grid, axis, marks, labels, legend);
    centerText([
      { t: o.fmt(res.value), size: Math.min(56, Math.max(28, W / 8)), cls: "chart__big" },
      { t: o.caption, size: 13, cls: "chart__caption" },
    ]);
    return;
  }
  const xs = res.x;
  const series = res.series;
  if (!xs.length || !series.length) {
    clear(grid, axis, marks, labels, legend);
    centerText([{ t: o.empty, size: 13, cls: "chart__caption" }]);
    return;
  }
  const multi = series.length > 1;
  const donut = spec.kind === "donut";
  const swatches: Swatch[] = donut ? xs.map((x, i) => ({ key: x.key, label: x.label, color: x.color, pal: i })) : multi ? series.map((s, i) => ({ key: s.key, label: s.label, color: s.color, pal: i })) : [];
  const showLegend = swatches.length > 0 && spec.legend !== false;
  let legendH = 0;
  if (showLegend) {
    let lx = 0;
    const items: (Swatch & { x: number })[] = [];
    for (const s of swatches) {
      const w = 18 + width(clip(s.label || "—", 140)) + 14;
      if (lx + w > W - 8) break;
      items.push({ ...s, x: lx });
      lx += w;
    }
    legendH = 26;
    keyed(legend, items, (s) => s.key || "#none", "g", (g, s) => {
      set(g, { transform: `translate(${s.x + 4},8)`, class: "chart__legend-item" });
      if (g.childElementCount !== 2) g.replaceChildren(el("rect"), el("text"));
      const r = g.firstElementChild as SVGRectElement;
      set(r, { x: 0, y: 1, width: 10, height: 10, rx: 2, class: "chart__swatch" });
      paintSwatch(r, s);
      const t = g.lastElementChild as SVGTextElement;
      set(t, { x: 16, y: 10, class: "chart__tick" });
      text(t, clip(s.label || "—", 140));
    });
  } else clear(legend);
  clear(center);
  if (donut) {
    clear(grid, axis);
    const first = series[0];
    const vals = xs.map((_, i) => (first ? Math.max(0, val(first, i)) : 0));
    const total = vals.reduce((a, b) => a + b, 0);
    const top = legendH + 8;
    const r = Math.max(20, Math.min(W - 16, H - top - 8) / 2);
    const cx = W / 2;
    const cy = top + (H - top) / 2;
    const ir = r * 0.62;
    let a = 0;
    const slices = xs.map((x, i) => {
      const v = vals[i] ?? 0;
      const s = { x, i, v, a0: a, a1: a + (total ? (v / total) * Math.PI * 2 : 0) };
      a = s.a1;
      return s;
    });
    keyed(marks, slices.filter((s) => s.v > 0), (s) => `d:${s.x.key}`, "path", (e, s) => {
      set(e, { d: arc(cx, cy, r, ir, s.a0, s.a1), class: "chart__mark chart__slice", "data-x": s.x.key });
      paintSwatch(e, { key: s.x.key, label: s.x.label, color: s.x.color, pal: s.i });
      tip(e, `${s.x.label || "—"}: ${o.fmt(s.v)} (${total ? Math.round((s.v / total) * 100) : 0}%)`);
    });
    const shown = spec.labels ? slices.filter((s) => s.v > 0 && s.a1 - s.a0 > 0.25) : [];
    keyed(labels, shown, (s) => `d:${s.x.key}`, "text", (e, s) => {
      const mid = (s.a0 + s.a1) / 2;
      const rr = (r + ir) / 2;
      set(e, { x: cx + rr * Math.sin(mid), y: cy - rr * Math.cos(mid) + 4, "text-anchor": "middle", class: "chart__value chart__value--in" });
      text(e, `${Math.round((s.v / total) * 100)}%`);
    });
    keyed(center, [o.fmt(total), o.caption], (_, i) => String(i), "text", (e, t, i) => {
      set(e, { x: cx, y: cy + (i ? 16 : -2), "text-anchor": "middle", class: i ? "chart__caption" : "chart__total", "font-size": i ? 12 : Math.max(16, Math.min(28, ir / 2.4)) });
      text(e, clip(t, ir * 1.8, i ? 12 : 20));
    });
    return;
  }
  const horiz = spec.kind === "hbar";
  const line = spec.kind === "line";
  const stacked = !!spec.stacked && multi && !line;
  let lo = 0;
  let hi = 0;
  xs.forEach((_, i) => {
    if (stacked) {
      let pos = 0;
      let neg = 0;
      for (const s of series) {
        if (val(s, i) >= 0) pos += val(s, i);
        else neg += val(s, i);
      }
      hi = Math.max(hi, pos);
      lo = Math.min(lo, neg);
    } else
      for (const s of series) {
        hi = Math.max(hi, val(s, i));
        lo = Math.min(lo, val(s, i));
      }
  });
  const ticks = niceTicks(lo, hi, horiz ? 4 : 5);
  const d0 = ticks[0] ?? 0;
  const d1 = ticks[ticks.length - 1] ?? 1;
  const top = legendH + (spec.labels ? 22 : 12);
  const labelChars = Math.max(...xs.map((x) => (x.label || "—").length));
  const left = horiz ? Math.min(160, Math.max(40, labelChars * 6.6 + 14)) : Math.max(...ticks.map((t) => width(o.axisFmt(t)))) + 14;
  const right = horiz && spec.labels ? 44 : 16;
  const bottom = 26;
  const pw = Math.max(10, W - left - right);
  const ph = Math.max(10, H - top - bottom);
  const n = xs.length;
  const band = (horiz ? ph : pw) / n;
  const scale = (v: number) => (horiz ? left + ((v - d0) / (d1 - d0)) * pw : top + ph - ((v - d0) / (d1 - d0)) * ph);
  keyed(grid, ticks, (t) => String(t), "line", (e, t) => {
    const p = scale(t);
    set(e, horiz ? { x1: p, x2: p, y1: top, y2: top + ph, class: t === 0 ? "chart__base" : "chart__grid" } : { x1: left, x2: left + pw, y1: p, y2: p, class: t === 0 ? "chart__base" : "chart__grid" });
  });
  const valueTicks = ticks.map((t) => ({ k: `v${t}`, t: o.axisFmt(t), x: horiz ? scale(t) : left - 8, y: horiz ? top + ph + 16 : scale(t) + 4, anchor: horiz ? "middle" : "end" }));
  const every = Math.max(1, Math.ceil((horiz ? 14 : Math.min(labelChars, 14) * 6.6 + 10) / band));
  const room = horiz ? left - 10 : band * every - 6;
  const catTicks = xs
    .map((x, i) => ({ x, i }))
    .filter(({ i }) => i % every === 0)
    .map(({ x, i }) => ({ k: `c${x.key}`, t: clip(x.label || "—", room), x: horiz ? left - 8 : left + band * (i + 0.5), y: horiz ? top + band * (i + 0.5) + 4 : top + ph + 16, anchor: horiz ? "end" : "middle" }));
  keyed(axis, [...valueTicks, ...catTicks], (t) => t.k, "text", (e, t) => {
    set(e, { x: t.x, y: t.y, "text-anchor": t.anchor, class: "chart__tick" });
    text(e, t.t);
  });
  const color = (s: (typeof series)[number], si: number, x: (typeof xs)[number], xi: number): Swatch => (multi ? { key: s.key, label: s.label, color: s.color, pal: si } : { key: x.key, label: x.label, color: x.color, pal: x.color ? xi : 0 });
  const tipFor = (x: (typeof xs)[number], s: (typeof series)[number], v: number) => `${x.label || "—"}${multi ? ` · ${s.label || "—"}` : ""}: ${o.fmt(v)}`;
  if (line) {
    const px = (i: number) => left + band * (i + 0.5);
    const lines = series.map((s, si) => ({ s, si, pts: s.values.map((v, i) => [px(i), scale(v)] as [number, number]) }));
    const areas = multi ? [] : lines;
    keyed(marks, [...areas.map((l) => ({ ...l, area: true })), ...lines.map((l) => ({ ...l, area: false }))], (l) => `${l.area ? "a" : "l"}:${l.s.key}`, "path", (e, l) => {
      const d = spec.smooth ? monotone(l.pts) : linear(l.pts);
      const base = scale(Math.max(d0, 0));
      const x0 = l.pts[0]?.[0] ?? left;
      const x1 = l.pts[l.pts.length - 1]?.[0] ?? left;
      set(e, { d: l.area ? `${d}L${x1.toFixed(1)},${base.toFixed(1)}L${x0.toFixed(1)},${base.toFixed(1)}Z` : d, class: l.area ? "chart__area" : "chart__line" });
      paintSwatch(e, { key: l.s.key, label: l.s.label, color: multi ? l.s.color : undefined, pal: l.si });
    });
    const dots = lines.flatMap((l) => (n <= 60 ? l.pts.map((p, i) => ({ l, p, x: xs[i] ?? NONE, v: val(l.s, i) })) : []));
    keyed(labels, [...dots.map((d) => ({ ...d, kind: "dot" })), ...(spec.labels ? dots.map((d) => ({ ...d, kind: "val" })) : [])], (d) => `${d.kind}:${d.l.s.key}:${d.x.key}`, "g", (g, d) => {
      const want = d.kind === "dot" ? "circle" : "text";
      if (g.firstElementChild?.tagName !== want) g.replaceChildren(el(want as "circle"));
      const c = g.firstElementChild as SVGElement;
      if (d.kind === "dot") {
        set(c, { cx: d.p[0], cy: d.p[1], r: 3.5, class: "chart__dot", "data-x": d.x.key, "data-s": multi ? d.l.s.key : null });
        paintSwatch(c, { key: d.l.s.key, label: "", color: multi ? d.l.s.color : undefined, pal: d.l.si });
        tip(c, tipFor(d.x, d.l.s, d.v));
      } else {
        set(c, { x: d.p[0], y: d.p[1] - 9, "text-anchor": "middle", class: "chart__value" });
        text(c, o.fmt(d.v));
      }
    });
    return;
  }
  const inner = band * (n === 1 ? 0.4 : 0.72);
  const slots = stacked ? 1 : series.length;
  const sub = Math.min(inner / slots, horiz ? 40 : 64);
  const groupW = sub * slots;
  const bars: { key: string; x: number; y: number; w: number; h: number; s: (typeof series)[number]; si: number; xi: number; v: number }[] = [];
  xs.forEach((x, xi) => {
    let pos = 0;
    let neg = 0;
    series.forEach((s, si) => {
      const v = val(s, xi);
      const start = band * xi + (band - groupW) / 2 + (stacked ? 0 : sub * si);
      let a: number;
      let b: number;
      if (stacked) {
        a = v >= 0 ? pos : neg;
        b = a + v;
        if (v >= 0) pos = b;
        else neg = b;
      } else {
        a = 0;
        b = v;
      }
      const p0 = scale(a);
      const p1 = scale(b);
      const gap = slots > 1 && !stacked ? 2 : 0;
      if (horiz) bars.push({ key: `${x.key}|${s.key}`, x: Math.min(p0, p1), y: top + start + gap / 2, w: Math.abs(p1 - p0), h: sub - gap, s, si, xi, v });
      else bars.push({ key: `${x.key}|${s.key}`, x: left + start + gap / 2, y: Math.min(p0, p1), w: sub - gap, h: Math.abs(p1 - p0), s, si, xi, v });
    });
  });
  keyed(marks, bars, (b) => `b:${b.key}`, "rect", (e, b) => {
    const x = xs[b.xi] ?? NONE;
    set(e, { x: b.x, y: b.y, width: Math.max(0, b.w), height: Math.max(0, b.h), rx: Math.min(3, b.w / 2, b.h / 2), class: "chart__mark chart__bar", "data-x": x.key, "data-s": multi ? b.s.key : null });
    paintSwatch(e, color(b.s, b.si, x, b.xi));
    tip(e, tipFor(x, b.s, b.v));
  });
  const vals = spec.labels ? (stacked ? xs.map((x, xi) => ({ key: x.key, xi, v: series.reduce((a, s) => a + val(s, xi), 0), bar: bars.filter((b) => b.xi === xi) })) : bars.map((b) => ({ key: b.key, xi: b.xi, v: b.v, bar: [b] }))) : [];
  keyed(labels, vals, (v) => `v:${v.key}`, "text", (e, v) => {
    const b = v.bar;
    const b0 = b[0];
    if (!b0) return;
    if (horiz) {
      const end = Math.max(...b.map((r) => r.x + r.w));
      set(e, { x: end + 6, y: b0.y + b0.h / 2 + 4, "text-anchor": "start", class: "chart__value" });
    } else {
      const y = Math.min(...b.map((r) => r.y));
      set(e, { x: b0.x + b0.w / 2, y: y - 6, "text-anchor": "middle", class: "chart__value" });
    }
    text(e, o.fmt(v.v));
  });
}

export { arc, clip, el, keyed, layer, linear, monotone, paintSwatch, set, text, tip, width };
export type { Swatch };

function size(svg: SVGSVGElement): [number, number] {
  const vb = (svg.getAttribute("viewBox") ?? "").split(" ").map(Number);
  return [vb[2] || 300, vb[3] || 150];
}

const STYLE_PROPS = ["fill", "fill-opacity", "stroke", "stroke-width", "stroke-opacity", "stroke-dasharray", "opacity", "font-size", "font-family", "font-weight"];

export function standalone(svg: SVGSVGElement, title: string): string {
  const clone = svg.cloneNode(true) as SVGSVGElement;
  const src = [svg, ...Array.from(svg.querySelectorAll("*"))];
  const dst = [clone, ...Array.from(clone.querySelectorAll("*"))];
  src.forEach((s, i) => {
    const cs = getComputedStyle(s);
    const d = dst[i] as SVGElement;
    for (const p of STYLE_PROPS) {
      const v = cs.getPropertyValue(p).trim();
      if (v) d.setAttribute(p, v);
    }
    d.removeAttribute("class");
    d.removeAttribute("data-key");
    d.removeAttribute("data-color");
    d.removeAttribute("data-pal");
    d.removeAttribute("data-layer");
    for (const a of ["data-x", "data-s", "data-row", "data-level", "data-sign"]) d.removeAttribute(a);
  });
  clone.setAttribute("xmlns", NS);
  const [w, hgt] = size(svg);
  const bg = el("rect");
  set(bg, { x: 0, y: 0, width: w, height: hgt, fill: getComputedStyle(svg).getPropertyValue("--sheet").trim() || "#ffffff" });
  clone.prepend(bg);
  if (title) {
    const t = el("title");
    t.textContent = title;
    clone.prepend(t);
  }
  return `<?xml version="1.0" encoding="UTF-8"?>\n${new XMLSerializer().serializeToString(clone)}`;
}

export function exportChart(svg: SVGSVGElement, name: string, as: "svg" | "png"): Promise<void> {
  const xml = standalone(svg, name);
  const file = (name || "chart").replace(/[\\/:*?"<>|]+/g, " ").trim() || "chart";
  const save = (blob: Blob, ext: string) => {
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${file}.${ext}`;
    document.body.append(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
  const blob = new Blob([xml], { type: "image/svg+xml" });
  if (as === "svg") {
    save(blob, "svg");
    return Promise.resolve();
  }
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(blob);
    const img = new Image();
    img.onload = () => {
      const scale = 2;
      const [w, hgt] = size(svg);
      const c = document.createElement("canvas");
      c.width = w * scale;
      c.height = hgt * scale;
      const g = c.getContext("2d");
      if (!g) return reject(new Error("canvas unavailable"));
      g.scale(scale, scale);
      g.drawImage(img, 0, 0, w, hgt);
      URL.revokeObjectURL(url);
      c.toBlob((b) => {
        if (b) save(b, "png");
        resolve();
      }, "image/png");
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error("could not render the chart"));
    };
    img.src = url;
  });
}
