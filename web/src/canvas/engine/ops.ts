import { type Anchor, anchor, type CornerStyle, dist, isCorner, lerp, maxCornerRadius, norm, type Path, type Pt, splitCubic, sub, add, mul } from "./geometry";

const K = 0.5522847498;

export function rect(x: number, y: number, w: number, h: number, r = 0): Path {
  const p: Path = { closed: true, anchors: [anchor(x, y), anchor(x + w, y), anchor(x + w, y + h), anchor(x, y + h)] };
  if (r > 0) setCorners(p, [0, 1, 2, 3], r, "round");
  return p;
}

export function ellipse(cx: number, cy: number, rx: number, ry: number): Path {
  const ox = rx * K, oy = ry * K;
  return {
    closed: true,
    anchors: [
      anchor(cx, cy - ry, { x: cx - ox, y: cy - ry }, { x: cx + ox, y: cy - ry }, true),
      anchor(cx + rx, cy, { x: cx + rx, y: cy - oy }, { x: cx + rx, y: cy + oy }, true),
      anchor(cx, cy + ry, { x: cx + ox, y: cy + ry }, { x: cx - ox, y: cy + ry }, true),
      anchor(cx - rx, cy, { x: cx - rx, y: cy + oy }, { x: cx - rx, y: cy - oy }, true),
    ],
  };
}

export function polygon(cx: number, cy: number, r: number, sides: number): Path {
  const n = Math.max(3, Math.min(1000, Math.round(sides)));
  const anchors: Anchor[] = [];
  for (let i = 0; i < n; i++) {
    const a = -Math.PI / 2 + (i * 2 * Math.PI) / n;
    anchors.push(anchor(cx + r * Math.cos(a), cy + r * Math.sin(a)));
  }
  return { closed: true, anchors };
}

export function star(cx: number, cy: number, r1: number, r2: number, points: number): Path {
  const n = Math.max(3, Math.min(1000, Math.round(points)));
  const anchors: Anchor[] = [];
  for (let i = 0; i < n * 2; i++) {
    const a = -Math.PI / 2 + (i * Math.PI) / n;
    const r = i % 2 === 0 ? r1 : r2;
    anchors.push(anchor(cx + r * Math.cos(a), cy + r * Math.sin(a)));
  }
  return { closed: true, anchors };
}

export function line(a: Pt, b: Pt): Path {
  return { closed: false, anchors: [anchor(a.x, a.y), anchor(b.x, b.y)] };
}

export function spiral(cx: number, cy: number, r: number, turns: number, decay = 0.8): Path {
  const anchors: Anchor[] = [];
  const steps = Math.max(2, Math.round(turns * 4));
  for (let i = 0; i <= steps; i++) {
    const a = (i * Math.PI) / 2;
    const rr = r * Math.pow(decay, i / 4);
    anchors.push(anchor(cx + rr * Math.cos(a), cy + rr * Math.sin(a)));
  }
  return smoothAll({ closed: false, anchors });
}

export function setCorners(p: Path, which: number[], r: number, style: CornerStyle = "round"): number {
  let changed = 0;
  for (const i of which) {
    const a = p.anchors[i];
    if (!a || !isCorner(p, i)) continue;
    if (r <= 0) delete a.corner;
    else a.corner = { r: Math.min(r, maxCornerRadius(p, i) || r), style };
    changed++;
  }
  return changed;
}

export function roundAll(p: Path, r: number, style: CornerStyle = "round"): number {
  return setCorners(p, p.anchors.map((_, i) => i), r, style);
}

export function convertAnchor(p: Path, i: number, to: "smooth" | "corner"): void {
  const a = p.anchors[i];
  if (!a) return;
  if (to === "corner") {
    a.in = null;
    a.out = null;
    a.smooth = false;
    return;
  }
  const n = p.anchors.length;
  const prev = p.anchors[i - 1] ?? (p.closed ? p.anchors[(i - 1 + n) % n] : undefined);
  const next = p.anchors[i + 1] ?? (p.closed ? p.anchors[(i + 1) % n] : undefined);
  const dir = norm(sub(next ?? { x: a.x + 1, y: a.y }, prev ?? { x: a.x - 1, y: a.y }));
  const lin = prev ? dist(prev, a) / 3 : 20;
  const lout = next ? dist(next, a) / 3 : 20;
  a.in = sub(a, mul(dir, lin));
  a.out = add(a, mul(dir, lout));
  a.smooth = true;
  delete a.corner;
}

export function smoothAll(p: Path): Path {
  for (let i = 0; i < p.anchors.length; i++) convertAnchor(p, i, "smooth");
  return p;
}

export function moveHandle(p: Path, i: number, which: "in" | "out", to: Pt, breakSymmetry = false): void {
  const a = p.anchors[i];
  if (!a) return;
  a[which] = { x: to.x, y: to.y };
  if (a.smooth && !breakSymmetry) {
    const other = which === "in" ? "out" : "in";
    const keep = a[other] ? dist(a[other]!, a) : dist(to, a);
    a[other] = add(a, mul(norm(sub(a, to)), keep));
  } else if (breakSymmetry) a.smooth = false;
}

export function moveAnchor(p: Path, i: number, dx: number, dy: number): void {
  const a = p.anchors[i];
  if (!a) return;
  a.x += dx; a.y += dy;
  if (a.in) a.in = { x: a.in.x + dx, y: a.in.y + dy };
  if (a.out) a.out = { x: a.out.x + dx, y: a.out.y + dy };
}

export function insertAnchor(p: Path, seg: number, t: number): number {
  const n = p.anchors.length;
  const a = p.anchors[seg]!;
  const b = p.anchors[(seg + 1) % n]!;
  if (!a.out && !b.in) {
    const m = lerp(a, b, t);
    p.anchors.splice(seg + 1, 0, anchor(m.x, m.y));
    return seg + 1;
  }
  const [, c1, c2, m, d1, d2] = splitCubic(a, a.out ?? a, b.in ?? b, b, t);
  a.out = c1;
  b.in = d2;
  p.anchors.splice(seg + 1, 0, anchor(m.x, m.y, c2, d1, true));
  return seg + 1;
}

export function deleteAnchor(p: Path, i: number): boolean {
  if (p.anchors.length <= 2) return false;
  p.anchors.splice(i, 1);
  return true;
}

export function reverse(p: Path): void {
  p.anchors.reverse();
  for (const a of p.anchors) [a.in, a.out] = [a.out, a.in];
}

export function average(p: Path, which: number[], axis: "both" | "x" | "y" = "both"): void {
  const pts = which.map((i) => p.anchors[i]).filter((a): a is Anchor => !!a);
  if (!pts.length) return;
  const cx = pts.reduce((s, a) => s + a.x, 0) / pts.length;
  const cy = pts.reduce((s, a) => s + a.y, 0) / pts.length;
  for (const i of which) {
    const a = p.anchors[i];
    if (!a) continue;
    moveAnchor(p, i, axis === "y" ? 0 : cx - a.x, axis === "x" ? 0 : cy - a.y);
  }
}

export function join(a: Path, b: Path): Path | null {
  if (a.closed || b.closed) return null;
  const aEnd = a.anchors.at(-1)!;
  const bStart = b.anchors[0]!;
  const bEnd = b.anchors.at(-1)!;
  if (dist(aEnd, bEnd) < dist(aEnd, bStart)) reverse(b);
  const merged: Path = { closed: false, anchors: [...a.anchors, ...b.anchors] };
  return merged;
}

export function closePath(p: Path): void {
  if (p.anchors.length > 2) p.closed = true;
}

export function cutAt(p: Path, i: number): Path[] {
  if (p.closed) {
    const rot = [...p.anchors.slice(i), ...p.anchors.slice(0, i + 1)].map((a) => ({ ...a }));
    return [{ closed: false, anchors: rot }];
  }
  if (i <= 0 || i >= p.anchors.length - 1) return [p];
  const head = p.anchors.slice(0, i + 1).map((a) => ({ ...a }));
  const tail = p.anchors.slice(i).map((a) => ({ ...a }));
  return [{ closed: false, anchors: head }, { closed: false, anchors: tail }];
}

export function simplify(p: Path, tolerance: number): void {
  if (p.anchors.length < 3) return;
  const pts = p.anchors;
  const keep = new Array<boolean>(pts.length).fill(false);
  keep[0] = keep[pts.length - 1] = true;
  const rdp = (s: number, e: number) => {
    let max = 0, at = -1;
    const A = pts[s]!, B = pts[e]!;
    const L = dist(A, B) || 1e-9;
    for (let i = s + 1; i < e; i++) {
      const P = pts[i]!;
      const d = Math.abs((B.x - A.x) * (A.y - P.y) - (A.x - P.x) * (B.y - A.y)) / L;
      if (d > max) { max = d; at = i; }
    }
    if (max > tolerance && at > 0) { keep[at] = true; rdp(s, at); rdp(at, e); }
  };
  rdp(0, pts.length - 1);
  p.anchors = pts.filter((_, i) => keep[i]);
}

export function transformPath(p: Path, m: [number, number, number, number, number, number]): void {
  const ap = (q: Pt): Pt => ({ x: m[0] * q.x + m[2] * q.y + m[4], y: m[1] * q.x + m[3] * q.y + m[5] });
  const scale = Math.sqrt(Math.abs(m[0] * m[3] - m[1] * m[2]));
  for (const a of p.anchors) {
    const q = ap(a);
    a.in = a.in ? ap(a.in) : null;
    a.out = a.out ? ap(a.out) : null;
    a.x = q.x; a.y = q.y;
    if (a.corner) a.corner = { ...a.corner, r: a.corner.r * scale };
  }
}
