export type SimIn =
  | { type: "init"; n: number; links: Int32Array; groups: Int32Array; radius: Float32Array; pos: Float32Array; alpha: number }
  | { type: "drag"; i: number; x: number; y: number }
  | { type: "release"; i: number }
  | { type: "reheat"; alpha: number }
  | { type: "stop" };

export type SimOut = { type: "tick"; pos: Float32Array; alpha: number; done: boolean };

const scope = self as unknown as { postMessage(m: SimOut, t?: Transferable[]): void; onmessage: ((e: MessageEvent<SimIn>) => void) | null };

let n = 0;
let x = new Float32Array(0);
let y = new Float32Array(0);
let vx = new Float32Array(0);
let vy = new Float32Array(0);
let fx = new Float32Array(0);
let fy = new Float32Array(0);
let rad: Float32Array = new Float32Array(0);
let group: Int32Array = new Int32Array(0);
let links: Int32Array = new Int32Array(0);
let strength = new Float32Array(0);
let bias = new Float32Array(0);
let alpha = 1;
let timer: ReturnType<typeof setTimeout> | undefined;

const ALPHA_MIN = 0.002;
const DECAY = 1 - Math.pow(ALPHA_MIN, 1 / 320);
const VELOCITY = 0.58;
const THETA2 = 0.81;

let cap = 0;
let child = new Int32Array(0);
let point = new Int32Array(0);
let mass = new Float32Array(0);
let mx = new Float32Array(0);
let my = new Float32Array(0);
let size = new Float32Array(0);
let used = 0;

function pool(need: number): void {
  if (need <= cap) return;
  cap = need;
  child = new Int32Array(cap * 4);
  point = new Int32Array(cap);
  mass = new Float32Array(cap);
  mx = new Float32Array(cap);
  my = new Float32Array(cap);
  size = new Float32Array(cap);
}

function alloc(s: number): number {
  if (used >= cap) return -1;
  const q = used++;
  child.fill(0, q * 4, q * 4 + 4);
  point[q] = -1;
  size[q] = s;
  return q;
}

function insert(q: number, x0: number, y0: number, s: number, i: number, depth: number): void {
  if (point[q] === -1) {
    point[q] = i;
    return;
  }
  if ((point[q] ?? -1) >= 0) {
    const j = point[q] as number;
    if (depth > 28 || (Math.abs((x[j] ?? 0) - (x[i] ?? 0)) < 1e-3 && Math.abs((y[j] ?? 0) - (y[i] ?? 0)) < 1e-3)) {
      x[i] = (x[i] ?? 0) + ((i * 7919) % 13) * 0.05 - 0.3;
      y[i] = (y[i] ?? 0) + ((i * 104729) % 17) * 0.05 - 0.4;
      if (depth > 28) return;
    }
    point[q] = -2;
    down(q, x0, y0, s, j, depth);
  }
  down(q, x0, y0, s, i, depth);
}

function down(q: number, x0: number, y0: number, s: number, i: number, depth: number): void {
  const h = s / 2;
  const right = (x[i] ?? 0) >= x0 + h ? 1 : 0;
  const lower = (y[i] ?? 0) >= y0 + h ? 1 : 0;
  const slot = q * 4 + right + lower * 2;
  let c = (child[slot] ?? 0) - 1;
  if (c < 0) {
    c = alloc(h);
    if (c < 0) return;
    child[slot] = c + 1;
  }
  insert(c, x0 + right * h, y0 + lower * h, h, i, depth + 1);
}

function accumulate(q: number): void {
  const p = point[q] as number;
  if (p >= 0) {
    mass[q] = 1;
    mx[q] = x[p] ?? 0;
    my[q] = y[p] ?? 0;
    return;
  }
  let m = 0;
  let sx = 0;
  let sy = 0;
  for (let k = 0; k < 4; k++) {
    const c = (child[q * 4 + k] ?? 0) - 1;
    if (c < 0) continue;
    accumulate(c);
    const cm = mass[c] ?? 0;
    m += cm;
    sx += (mx[c] ?? 0) * cm;
    sy += (my[c] ?? 0) * cm;
  }
  mass[q] = m;
  mx[q] = m ? sx / m : 0;
  my[q] = m ? sy / m : 0;
}

const stack: number[] = [];

function charge(k: number): void {
  let x0 = Infinity;
  let y0 = Infinity;
  let x1 = -Infinity;
  let y1 = -Infinity;
  for (let i = 0; i < n; i++) {
    const a = x[i] ?? 0;
    const b = y[i] ?? 0;
    if (a < x0) x0 = a;
    if (a > x1) x1 = a;
    if (b < y0) y0 = b;
    if (b > y1) y1 = b;
  }
  const s = Math.max(x1 - x0, y1 - y0, 1) + 1;
  pool(n * 6 + 64);
  used = 0;
  const root = alloc(s);
  for (let i = 0; i < n; i++) insert(root, x0, y0, s, i, 0);
  accumulate(root);
  const str = -42 * k;
  for (let i = 0; i < n; i++) {
    const xi = x[i] ?? 0;
    const yi = y[i] ?? 0;
    let ax = 0;
    let ay = 0;
    stack.length = 0;
    stack.push(root);
    while (stack.length) {
      const q = stack.pop() as number;
      const m = mass[q] ?? 0;
      if (!m) continue;
      const dx = (mx[q] ?? 0) - xi;
      const dy = (my[q] ?? 0) - yi;
      let d2 = dx * dx + dy * dy;
      const p = point[q] as number;
      const sq = size[q] ?? 0;
      if (p === i) continue;
      if (p >= 0 || (sq * sq) / (d2 || 1e-9) < THETA2) {
        if (d2 < 1) d2 = 1;
        const f = (str * m) / d2;
        ax += dx * f;
        ay += dy * f;
        continue;
      }
      for (let c = 0; c < 4; c++) {
        const ch = (child[q * 4 + c] ?? 0) - 1;
        if (ch >= 0) stack.push(ch);
      }
    }
    vx[i] = (vx[i] ?? 0) + ax;
    vy[i] = (vy[i] ?? 0) + ay;
  }
}

function linkForce(k: number): void {
  const m = links.length / 2;
  for (let e = 0; e < m; e++) {
    const a = links[e * 2] as number;
    const b = links[e * 2 + 1] as number;
    const dx = (x[b] ?? 0) + (vx[b] ?? 0) - (x[a] ?? 0) - (vx[a] ?? 0) || 1e-3;
    const dy = (y[b] ?? 0) + (vy[b] ?? 0) - (y[a] ?? 0) - (vy[a] ?? 0) || 1e-3;
    const d = Math.sqrt(dx * dx + dy * dy);
    const rest = 34 + (rad[a] ?? 4) + (rad[b] ?? 4);
    const f = ((d - rest) / d) * k * (strength[e] ?? 0.5);
    const bb = bias[e] ?? 0.5;
    vx[b] = (vx[b] ?? 0) - dx * f * bb;
    vy[b] = (vy[b] ?? 0) - dy * f * bb;
    vx[a] = (vx[a] ?? 0) + dx * f * (1 - bb);
    vy[a] = (vy[a] ?? 0) + dy * f * (1 - bb);
  }
}

function clusterForce(k: number): void {
  const sums = new Map<number, [number, number, number]>();
  for (let i = 0; i < n; i++) {
    const g = group[i] ?? -1;
    if (g < 0) continue;
    const s = sums.get(g) ?? [0, 0, 0];
    s[0] += x[i] ?? 0;
    s[1] += y[i] ?? 0;
    s[2]++;
    sums.set(g, s);
  }
  if (sums.size < 2) return;
  for (let i = 0; i < n; i++) {
    const s = sums.get(group[i] ?? -1);
    if (!s) continue;
    vx[i] = (vx[i] ?? 0) + (s[0] / s[2] - (x[i] ?? 0)) * 0.06 * k;
    vy[i] = (vy[i] ?? 0) + (s[1] / s[2] - (y[i] ?? 0)) * 0.06 * k;
  }
}

function tick(): void {
  alpha += (0 - alpha) * DECAY;
  if (n > 1) charge(alpha);
  linkForce(alpha);
  clusterForce(alpha);
  for (let i = 0; i < n; i++) {
    vx[i] = (vx[i] ?? 0) - (x[i] ?? 0) * 0.025 * alpha;
    vy[i] = (vy[i] ?? 0) - (y[i] ?? 0) * 0.025 * alpha;
    if (!Number.isNaN(fx[i])) {
      x[i] = fx[i] as number;
      y[i] = fy[i] as number;
      vx[i] = 0;
      vy[i] = 0;
      continue;
    }
    vx[i] = (vx[i] ?? 0) * VELOCITY;
    vy[i] = (vy[i] ?? 0) * VELOCITY;
    x[i] = (x[i] ?? 0) + (vx[i] ?? 0);
    y[i] = (y[i] ?? 0) + (vy[i] ?? 0);
  }
}

function post(done: boolean): void {
  const out = new Float32Array(n * 2);
  for (let i = 0; i < n; i++) {
    out[i * 2] = x[i] ?? 0;
    out[i * 2 + 1] = y[i] ?? 0;
  }
  scope.postMessage({ type: "tick", pos: out, alpha, done }, [out.buffer]);
}

function run(): void {
  clearTimeout(timer);
  const start = performance.now();
  while (performance.now() - start < 14 && alpha > ALPHA_MIN) tick();
  const done = alpha <= ALPHA_MIN;
  post(done);
  if (!done) timer = setTimeout(run, 0);
}

scope.onmessage = (e) => {
  const m = e.data;
  if (m.type === "init") {
    n = m.n;
    x = new Float32Array(n);
    y = new Float32Array(n);
    vx = new Float32Array(n);
    vy = new Float32Array(n);
    fx = new Float32Array(n).fill(Number.NaN);
    fy = new Float32Array(n).fill(Number.NaN);
    rad = m.radius;
    group = m.groups;
    links = m.links;
    const deg = new Int32Array(n);
    for (const v of links) deg[v] = (deg[v] ?? 0) + 1;
    const ml = links.length / 2;
    strength = new Float32Array(ml);
    bias = new Float32Array(ml);
    for (let e2 = 0; e2 < ml; e2++) {
      const a = links[e2 * 2] as number;
      const b = links[e2 * 2 + 1] as number;
      const da = deg[a] ?? 1;
      const db = deg[b] ?? 1;
      strength[e2] = 1 / Math.min(da, db);
      bias[e2] = da / (da + db);
    }
    for (let i = 0; i < n; i++) {
      const px = m.pos[i * 2];
      const py = m.pos[i * 2 + 1];
      if (px !== undefined && py !== undefined && !Number.isNaN(px) && !Number.isNaN(py)) {
        x[i] = px;
        y[i] = py;
      } else {
        const r = 10 * Math.sqrt(0.5 + i);
        const a = i * Math.PI * (3 - Math.sqrt(5));
        x[i] = r * Math.cos(a);
        y[i] = r * Math.sin(a);
      }
    }
    alpha = m.alpha;
    run();
  } else if (m.type === "drag") {
    fx[m.i] = m.x;
    fy[m.i] = m.y;
    x[m.i] = m.x;
    y[m.i] = m.y;
    if (alpha < 0.12) alpha = 0.12;
    run();
  } else if (m.type === "release") {
    fx[m.i] = Number.NaN;
    fy[m.i] = Number.NaN;
  } else if (m.type === "reheat") {
    alpha = Math.max(alpha, m.alpha);
    run();
  } else if (m.type === "stop") clearTimeout(timer);
};
