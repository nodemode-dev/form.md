export interface Pt { x: number; y: number }

export type CornerStyle = "round" | "inverted" | "chamfer";

export interface Anchor {
  x: number;
  y: number;
  in: Pt | null;
  out: Pt | null;
  smooth: boolean;
  corner?: { r: number; style: CornerStyle };
}

export interface Path {
  anchors: Anchor[];
  closed: boolean;
}

export type Seg =
  | { kind: "M"; to: Pt }
  | { kind: "L"; to: Pt }
  | { kind: "C"; c1: Pt; c2: Pt; to: Pt }
  | { kind: "Z" };

export const add = (a: Pt, b: Pt): Pt => ({ x: a.x + b.x, y: a.y + b.y });
export const sub = (a: Pt, b: Pt): Pt => ({ x: a.x - b.x, y: a.y - b.y });
export const mul = (a: Pt, k: number): Pt => ({ x: a.x * k, y: a.y * k });
export const len = (a: Pt): number => Math.hypot(a.x, a.y);
export const lerp = (a: Pt, b: Pt, t: number): Pt => ({ x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t });
export const norm = (a: Pt): Pt => {
  const l = len(a);
  return l < 1e-12 ? { x: 0, y: 0 } : { x: a.x / l, y: a.y / l };
};
export const dist = (a: Pt, b: Pt): number => Math.hypot(a.x - b.x, a.y - b.y);

export function anchor(x: number, y: number, inH: Pt | null = null, outH: Pt | null = null, smooth = false): Anchor {
  return { x, y, in: inH, out: outH, smooth };
}

function neighbour(p: Path, i: number, step: 1 | -1): Anchor | null {
  const n = p.anchors.length;
  const j = i + step;
  if (j >= 0 && j < n) return p.anchors[j]!;
  return p.closed && n > 2 ? p.anchors[(j + n) % n]! : null;
}

function tangentIn(p: Path, i: number): Pt | null {
  const a = p.anchors[i]!;
  if (a.in && dist(a.in, a) > 1e-9) return norm(sub(a.in, a));
  const prev = neighbour(p, i, -1);
  if (!prev) return null;
  const from = prev.out && dist(prev.out, prev) > 1e-9 ? prev.out : prev;
  return norm(sub(from, a));
}

function tangentOut(p: Path, i: number): Pt | null {
  const a = p.anchors[i]!;
  if (a.out && dist(a.out, a) > 1e-9) return norm(sub(a.out, a));
  const next = neighbour(p, i, 1);
  if (!next) return null;
  const to = next.in && dist(next.in, next) > 1e-9 ? next.in : next;
  return norm(sub(to, a));
}

export function maxCornerRadius(p: Path, i: number): number {
  const a = p.anchors[i]!;
  const prev = neighbour(p, i, -1);
  const next = neighbour(p, i, 1);
  const u = tangentIn(p, i);
  const v = tangentOut(p, i);
  if (!prev || !next || !u || !v) return 0;
  const cos = Math.max(-1, Math.min(1, u.x * v.x + u.y * v.y));
  const theta = Math.acos(cos);
  if (theta < 1e-3 || Math.PI - theta < 1e-3) return 0;
  const room = Math.min(dist(prev, a), dist(next, a)) / 2;
  return room * Math.tan(theta / 2);
}

export function isCorner(p: Path, i: number): boolean {
  const u = tangentIn(p, i);
  const v = tangentOut(p, i);
  if (!u || !v) return false;
  return u.x * v.x + u.y * v.y > -0.9995;
}

interface Expanded { pts: Anchor[] }

function expandCorners(p: Path): Expanded {
  const out: Anchor[] = [];
  const n = p.anchors.length;
  for (let i = 0; i < n; i++) {
    const a = p.anchors[i]!;
    const c = a.corner;
    const u = tangentIn(p, i);
    const v = tangentOut(p, i);
    if (!c || c.r <= 0 || !u || !v || !isCorner(p, i)) {
      out.push(a);
      continue;
    }
    const theta = Math.acos(Math.max(-1, Math.min(1, u.x * v.x + u.y * v.y)));
    const r = Math.min(c.r, maxCornerRadius(p, i));
    if (r <= 0) {
      out.push(a);
      continue;
    }
    const d = r / Math.tan(theta / 2);
    const start = add(a, mul(u, d));
    const end = add(a, mul(v, d));
    if (c.style === "chamfer") {
      out.push(anchor(start.x, start.y, a.in ? lerp(start, a.in, 0) : null, null));
      out.push(anchor(end.x, end.y, null, a.out ? lerp(end, a.out, 0) : null));
      continue;
    }
    if (c.style === "round") {
      const turn = Math.PI - theta;
      const h = (4 / 3) * Math.tan(turn / 4) * r;
      out.push(anchor(start.x, start.y, null, sub(start, mul(u, h))));
      out.push(anchor(end.x, end.y, sub(end, mul(v, h)), null));
      continue;
    }
    const h = (4 / 3) * Math.tan(theta / 4) * d;
    const bis = norm(add(u, v));
    const side = (w: Pt) => {
      const perp = { x: -w.y, y: w.x };
      return perp.x * bis.x + perp.y * bis.y > 0 ? perp : mul(perp, -1);
    };
    out.push(anchor(start.x, start.y, null, add(start, mul(side(u), h))));
    out.push(anchor(end.x, end.y, add(end, mul(side(v), h)), null));
  }
  return { pts: out };
}

export function segments(p: Path): Seg[] {
  const pts = p.anchors.some((a) => a.corner && a.corner.r > 0) ? expandCorners(p).pts : p.anchors;
  if (!pts.length) return [];
  const segs: Seg[] = [{ kind: "M", to: { x: pts[0]!.x, y: pts[0]!.y } }];
  const link = (a: Anchor, b: Anchor) => {
    if (!a.out && !b.in) segs.push({ kind: "L", to: { x: b.x, y: b.y } });
    else segs.push({ kind: "C", c1: a.out ?? { x: a.x, y: a.y }, c2: b.in ?? { x: b.x, y: b.y }, to: { x: b.x, y: b.y } });
  };
  for (let i = 1; i < pts.length; i++) link(pts[i - 1]!, pts[i]!);
  if (p.closed && pts.length > 1) {
    link(pts[pts.length - 1]!, pts[0]!);
    segs.push({ kind: "Z" });
  }
  return segs;
}

const f = (n: number) => (Math.abs(n) < 1e-9 ? "0" : String(Math.round(n * 1000) / 1000));

export function toSvgD(p: Path): string {
  return segments(p)
    .map((s) => {
      if (s.kind === "M" || s.kind === "L") return `${s.kind}${f(s.to.x)} ${f(s.to.y)}`;
      if (s.kind === "C") return `C${f(s.c1.x)} ${f(s.c1.y)} ${f(s.c2.x)} ${f(s.c2.y)} ${f(s.to.x)} ${f(s.to.y)}`;
      return "Z";
    })
    .join("");
}

export function cubicAt(p0: Pt, c1: Pt, c2: Pt, p3: Pt, t: number): Pt {
  const m = 1 - t;
  return {
    x: m * m * m * p0.x + 3 * m * m * t * c1.x + 3 * m * t * t * c2.x + t * t * t * p3.x,
    y: m * m * m * p0.y + 3 * m * m * t * c1.y + 3 * m * t * t * c2.y + t * t * t * p3.y,
  };
}

export function splitCubic(p0: Pt, c1: Pt, c2: Pt, p3: Pt, t: number): [Pt, Pt, Pt, Pt, Pt, Pt, Pt] {
  const a = lerp(p0, c1, t);
  const b = lerp(c1, c2, t);
  const c = lerp(c2, p3, t);
  const d = lerp(a, b, t);
  const e = lerp(b, c, t);
  const m = lerp(d, e, t);
  return [p0, a, d, m, e, c, p3];
}

export function bounds(p: Path): { x: number; y: number; w: number; h: number } {
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
  const take = (q: Pt) => {
    x0 = Math.min(x0, q.x); y0 = Math.min(y0, q.y); x1 = Math.max(x1, q.x); y1 = Math.max(y1, q.y);
  };
  let cur: Pt = { x: 0, y: 0 };
  let start: Pt = cur;
  for (const s of segments(p)) {
    if (s.kind === "M") { cur = start = s.to; take(cur); }
    else if (s.kind === "L") { cur = s.to; take(cur); }
    else if (s.kind === "C") {
      for (let i = 1; i <= 16; i++) take(cubicAt(cur, s.c1, s.c2, s.to, i / 16));
      cur = s.to;
    } else cur = start;
  }
  if (!Number.isFinite(x0)) return { x: 0, y: 0, w: 0, h: 0 };
  return { x: x0, y: y0, w: x1 - x0, h: y1 - y0 };
}

export function nearestOnPath(p: Path, q: Pt): { seg: number; t: number; d: number } {
  let best = { seg: -1, t: 0, d: Infinity };
  const n = p.anchors.length;
  const count = p.closed ? n : n - 1;
  for (let i = 0; i < count; i++) {
    const a = p.anchors[i]!;
    const b = p.anchors[(i + 1) % n]!;
    const c1 = a.out ?? a;
    const c2 = b.in ?? b;
    for (let k = 0; k <= 64; k++) {
      const t = k / 64;
      const d = dist(cubicAt(a, c1, c2, b, t), q);
      if (d < best.d) best = { seg: i, t, d };
    }
  }
  return best;
}
