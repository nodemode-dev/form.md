import { t } from "../../../engine/i18n";
import type { Property, RowData, RowRef } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import { toast } from "../../../ui/toast";
import { openPanel } from "../panel";
import type { ViewHost, ViewRenderer } from "../surface";
import { propSelect, saveConfig } from "../toolbar";
import type { SimIn, SimOut } from "./graphsim";

export interface GraphConfig {
  color?: string;
  size?: string;
  links?: string[];
  external?: boolean;
  labels?: "auto" | "all" | "none";
  cluster?: boolean;
}

interface GNode {
  id: string;
  title: string;
  ext: boolean;
  col: string;
  color: string;
  group: number;
  val: number;
  deg: number;
  r: number;
  x: number;
  y: number;
}

interface GEdge {
  a: number;
  b: number;
  prop: string;
  owner: number;
  other: number;
}

const MAX_ROWS = 2000;
const MAX_EXTERNAL = 600;
const COLORS = ["default", "gray", "brown", "orange", "yellow", "green", "blue", "purple", "pink", "red"];

const asIds = (v: unknown): string[] => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : typeof v === "string" && v ? [v] : []);

// Theme tones for rows linked in from other databases, with their rough hue so
// the pick is the same in every theme.
const EXT_TONES: [string, number][] = [
  ["--c-teal", 175],
  ["--c-pink", 330],
  ["--c-cyan", 192],
  ["--c-lime", 85],
  ["--c-amber", 38],
  ["--c-blue", 221],
  ["--c-orange", 22],
  ["--c-violet", 262],
  ["--c-rose", 350],
  ["--c-green", 142],
];
const OPTION_HUES: Record<string, number> = { red: 350, pink: 330, orange: 22, brown: 25, yellow: 38, green: 142, blue: 221, purple: 262 };
const hueGap = (a: number, b: number) => {
  const d = Math.abs(a - b) % 360;
  return Math.min(d, 360 - d);
};

const toRgb = (c: string): string => {
  const m = /^color\(srgb ([\d.]+) ([\d.]+) ([\d.]+)(?: \/ ([\d.]+))?\)$/.exec(c);
  if (m) return `rgb(${Math.round(Number(m[1]) * 255)}, ${Math.round(Number(m[2]) * 255)}, ${Math.round(Number(m[3]) * 255)})`;
  return c;
};

const rgbParts = (c: string): [number, number, number] | null => {
  const m = /rgba?\(\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)/.exec(c);
  return m ? [Number(m[1]), Number(m[2]), Number(m[3])] : null;
};

const hueOf = (c: string): number | null => {
  const p = rgbParts(c);
  if (!p) return null;
  const [r, g, b] = p.map((v) => v / 255) as [number, number, number];
  const mx = Math.max(r, g, b);
  const d = mx - Math.min(r, g, b);
  if (d < 0.08) return null;
  const h = mx === r ? ((g - b) / d) % 6 : mx === g ? (b - r) / d + 2 : (r - g) / d + 4;
  return (h * 60 + 360) % 360;
};

function resolveColor(probe: HTMLElement, name: string): string {
  probe.setAttribute("data-color", name);
  return toRgb(getComputedStyle(probe).color);
}

function resolveVar(probe: HTMLElement, token: string): string {
  probe.style.color = `var(${token})`;
  const c = toRgb(getComputedStyle(probe).color);
  probe.style.color = "";
  return c;
}

export function graphOf(layout: Record<string, unknown> | undefined): GraphConfig {
  return { external: true, labels: "auto", cluster: true, ...((layout?.graph as GraphConfig | undefined) ?? {}) };
}

export function graphView(host: ViewHost): ViewRenderer {
  const engine = host.engine;
  const session = host.makeSession({ watch: () => schedule() });
  const ctx = host.ctx(session);
  let destroyed = false;
  let nodes: GNode[] = [];
  let edges: GEdge[] = [];
  const rowVals = new Map<string, Record<string, unknown>>();
  const index = new Map<string, number>();
  let adj: number[][] = [];
  let worker: Worker | null = null;
  let settled = false;
  let k = 1;
  let tx = 0;
  let ty = 0;
  let userView = false;
  let hover = -1;
  let selected = -1;
  let query = "";
  let mode: "move" | "link" = "move";
  let rubber: { from: number; x: number; y: number } | null = null;
  let palette = new Map<string, string>();
  let accent = "#8b7cf6";
  let textColor = "#ddd";
  let faint = "#666";
  let muted = "#999";
  let sheet = "#111";
  let fontFamily = "sans-serif";
  let tones: string[] = [];
  let extNames = new Map<string, string>();
  let extFill = new Map<string, string>();
  let usedColors: string[] = [];
  let accentUsed = false;
  const widths = new Map<string, number>();

  const canvas = h("canvas", { class: "graphv__canvas", attrs: { tabindex: "0", role: "img", "aria-label": t("Graph of rows and their links") } });
  const probe = h("span", { class: "graphv__probe", attrs: { "aria-hidden": "true" } });
  const anchor = h("span", { class: "graphv__anchor", attrs: { "aria-hidden": "true" } });
  const side = h("aside", { class: "graphv__side", attrs: { "aria-label": t("Selected row") } });
  side.hidden = true;
  const a11y = h("ul", { class: "graphv__a11y", attrs: { "aria-label": t("Rows in this graph") } });
  const frame = h("div", { class: "graphv__frame" }, canvas, probe, anchor, side, a11y);
  const search = h("input", { class: "input input--sm graphv__search", attrs: { type: "search", placeholder: t("Find a row"), "aria-label": t("Find a row") } });
  const moveBtn = h("button", { class: "segmented__btn", attrs: { type: "button", "aria-pressed": "true", title: t("Move nodes"), "aria-label": t("Move nodes") } }, iconNode("grip"));
  const linkBtn = h("button", { class: "segmented__btn", attrs: { type: "button", "aria-pressed": "false", title: t("Draw links: drag from one row to another"), "aria-label": t("Draw links") } }, iconNode("relation"));
  const modes = h("div", { class: "segmented", attrs: { role: "group", "aria-label": t("Mode") } }, moveBtn, linkBtn);
  const zoomIn = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Zoom in") } }, iconNode("plus"));
  const zoomOut = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Zoom out") } }, h("span", { class: "mapv__minus" }));
  const fitBtn = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Fit to view") } }, iconNode("expand"));
  const count = h("span", { class: "graphv__count" });
  const legend = h("div", { class: "graphv__legend" });
  const el = h("div", { class: "graphv", attrs: { "data-inline": host.inline ? "true" : "false" } }, h("div", { class: "graphv__bar" }, search, modes, h("span", { class: "graphv__spacer" }), count, zoomOut, zoomIn, fitBtn), frame, legend);

  const conf = () => graphOf(host.view().config.layout);
  const schema = () => host.col().schema;
  const relProps = (): Property[] => {
    const c = conf();
    return schema().filter((p) => p.type === "relation" && (!c.links?.length || c.links.includes(p.id)));
  };

  const readTheme = () => {
    const cs = getComputedStyle(el);
    accent = cs.getPropertyValue("--accent").trim() || accent;
    textColor = cs.getPropertyValue("--text").trim() || textColor;
    faint = cs.getPropertyValue("--text-faint").trim() || faint;
    muted = cs.getPropertyValue("--text-muted").trim() || muted;
    fontFamily = cs.fontFamily || fontFamily;
    widths.clear();
    let behind: Element | null = frame;
    let bg = "";
    while (behind && !bg) {
      const c = getComputedStyle(behind).backgroundColor;
      if (c && c !== "transparent" && !/rgba\([^)]*,\s*0\)$/.test(c) && !/ \/ 0\)$/.test(c)) bg = toRgb(c);
      behind = behind.parentElement;
    }
    sheet = bg || cs.getPropertyValue("--sheet").trim() || sheet;
    palette = new Map(COLORS.map((c) => [c, resolveColor(probe, c)] as [string, string]).filter(([, v]) => !!v));
    tones = EXT_TONES.map(([v]) => resolveVar(probe, v));
    assignExt();
  };

  // Rows linked in from other databases get one fill per source database:
  // the tone whose hue is farthest from every hue already on the canvas.
  const assignExt = () => {
    const taken = usedColors.map((c) => OPTION_HUES[c]).filter((x): x is number => x !== undefined);
    const ah = accentUsed ? hueOf(resolveVar(probe, "--accent")) : null;
    if (ah !== null) taken.push(ah);
    const free = EXT_TONES.map((_, j) => j);
    extFill = new Map();
    for (const col of extNames.keys()) {
      let best = 0;
      let bd = -1;
      free.forEach((j, at) => {
        const hue = (EXT_TONES[j] as [string, number])[1];
        const d = taken.length ? Math.min(...taken.map((x) => hueGap(hue, x))) : 180;
        if (d > bd) {
          bd = d;
          best = at;
        }
      });
      const j = free.splice(best, 1)[0];
      const tone = j === undefined ? "" : (tones[j] ?? "");
      extFill.set(col, rgbParts(tone) ? tone : (palette.get("gray") ?? accent));
      if (j !== undefined) taken.push((EXT_TONES[j] as [string, number])[1]);
    }
    for (const i of legend.querySelectorAll<HTMLElement>("i[data-ext]")) i.style.background = extFill.get(i.dataset.ext ?? "") ?? "";
  };

  const size = () => [frame.clientWidth, frame.clientHeight] as const;
  const toWorld = (px: number, py: number) => [(px - tx) / k, (py - ty) / k] as const;

  let raf = 0;
  const redraw = () => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(paint);
  };

  const fit = (animate = false) => {
    const [W, H] = size();
    if (!nodes.length || !W || !H) return;
    let x0 = Infinity;
    let y0 = Infinity;
    let x1 = -Infinity;
    let y1 = -Infinity;
    for (const n of nodes) {
      x0 = Math.min(x0, n.x - n.r);
      y0 = Math.min(y0, n.y - n.r);
      x1 = Math.max(x1, n.x + n.r);
      y1 = Math.max(y1, n.y + n.r);
    }
    const nk = Math.max(0.05, Math.min(2.5, Math.min((W - 40) / (x1 - x0 || 1), (H - 40) / (y1 - y0 || 1))));
    const ntx = W / 2 - ((x0 + x1) / 2) * nk;
    const nty = H / 2 - ((y0 + y1) / 2) * nk;
    if (animate) {
      k += (nk - k) * 0.25;
      tx += (ntx - tx) * 0.25;
      ty += (nty - ty) * 0.25;
    } else {
      k = nk;
      tx = ntx;
      ty = nty;
    }
  };

  const neighbours = (i: number) => new Set(i >= 0 ? (adj[i] ?? []) : []);

  function paint(): void {
    if (destroyed) return;
    const [W, H] = size();
    if (!W || !H) return;
    const dpr = window.devicePixelRatio || 1;
    if (canvas.width !== Math.round(W * dpr) || canvas.height !== Math.round(H * dpr)) {
      canvas.width = Math.round(W * dpr);
      canvas.height = Math.round(H * dpr);
    }
    const g = canvas.getContext("2d");
    if (!g) return;
    if (!userView && !settled) fit(true);
    g.setTransform(dpr, 0, 0, dpr, 0, 0);
    g.clearRect(0, 0, W, H);
    g.setTransform(dpr * k, 0, 0, dpr * k, dpr * tx, dpr * ty);
    const focus = hover >= 0 ? hover : selected;
    const near = neighbours(focus);
    const q = query.trim().toLowerCase();
    const match = (n: GNode) => !q || n.title.toLowerCase().includes(q);
    g.lineWidth = 1 / k;
    for (const e of edges) {
      const a = nodes[e.a];
      const b = nodes[e.b];
      if (!a || !b) continue;
      const lit = focus >= 0 && (e.a === focus || e.b === focus);
      g.globalAlpha = focus >= 0 ? (lit ? 0.9 : 0.08) : q ? 0.12 : 0.35;
      g.strokeStyle = lit ? accent : faint;
      g.lineWidth = (lit ? 1.6 : 1) / k;
      g.beginPath();
      g.moveTo(a.x, a.y);
      g.lineTo(b.x, b.y);
      g.stroke();
    }
    nodes.forEach((n, i) => {
      const dim = (focus >= 0 && i !== focus && !near.has(i)) || (q && !match(n));
      g.globalAlpha = dim ? 0.18 : 1;
      g.fillStyle = n.ext ? (extFill.get(n.col) ?? palette.get("gray") ?? accent) : (palette.get(n.color || (n.group >= 0 ? "default" : "")) ?? accent);
      g.beginPath();
      g.arc(n.x, n.y, n.r, 0, Math.PI * 2);
      g.fill();
      g.strokeStyle = sheet;
      g.lineWidth = 1.25 / k;
      g.stroke();
      if (i === selected) {
        g.strokeStyle = textColor;
        g.lineWidth = 2.5 / k;
        g.beginPath();
        g.arc(n.x, n.y, n.r + 3 / k, 0, Math.PI * 2);
        g.stroke();
      }
    });
    if (rubber) {
      const a = nodes[rubber.from];
      if (a) {
        g.globalAlpha = 1;
        g.strokeStyle = accent;
        g.lineWidth = 2 / k;
        g.setLineDash([6 / k, 4 / k]);
        g.beginPath();
        g.moveTo(a.x, a.y);
        g.lineTo(rubber.x, rubber.y);
        g.stroke();
        g.setLineDash([]);
      }
    }
    const lab = conf().labels ?? "auto";
    if (lab !== "none") paintLabels(g, dpr, W, H, lab, focus, near, q, match);
    g.globalAlpha = 1;
  }

  // Labels are placed in screen space in order of importance (focus, selection,
  // neighbours, search matches, then node size and degree). Each tries below,
  // above, right and left of its node; a label that would overlap one already
  // placed is skipped and shows on hover, focus or selection instead.
  function paintLabels(g: CanvasRenderingContext2D, dpr: number, W: number, H: number, lab: "auto" | "all", focus: number, near: Set<number>, q: string, match: (n: GNode) => boolean): void {
    const LH = 15;
    const CELL = 48;
    const PAD = 3;
    g.setTransform(dpr, 0, 0, dpr, 0, 0);
    g.font = `12px ${fontFamily}`;
    g.textAlign = "center";
    g.textBaseline = "middle";
    g.lineJoin = "round";
    const cellsOf = (x0: number, y0: number, x1: number, y1: number): number[] => {
      const out: number[] = [];
      for (let cx = Math.floor(x0 / CELL); cx <= Math.floor(x1 / CELL); cx++) for (let cy = Math.floor(y0 / CELL); cy <= Math.floor(y1 / CELL); cy++) out.push((cx + 4096) * 8192 + cy + 4096);
      return out;
    };
    const push = (grid: Map<number, number[]>, key: number, v: number) => {
      let a = grid.get(key);
      if (!a) grid.set(key, (a = []));
      a.push(v);
    };
    const circles = new Map<number, number[]>();
    const vis: number[] = [];
    nodes.forEach((n, i) => {
      const x = n.x * k + tx;
      const y = n.y * k + ty;
      const r = n.r * k;
      if (x + r < -240 || x - r > W + 240 || y + r < -40 || y - r > H + 40) return;
      vis.push(i);
      for (const key of cellsOf(x - r, y - r, x + r, y + r)) push(circles, key, i);
    });
    const rank = (i: number): number => {
      const n = nodes[i] as GNode;
      return i === focus ? 0 : i === selected ? 1 : near.has(i) ? 2 : q.length > 1 && match(n) ? 3 : 4;
    };
    const roomy = vis.length <= 800 || k >= 1.6;
    const order = vis
      .filter((i) => lab === "all" || rank(i) < 4 || roomy || (nodes[i] as GNode).r * k >= 9)
      .map((i) => [i, rank(i)] as const)
      .sort((a, b) => a[1] - b[1] || (nodes[b[0]] as GNode).r - (nodes[a[0]] as GNode).r || (nodes[b[0]] as GNode).deg - (nodes[a[0]] as GNode).deg || Number((nodes[a[0]] as GNode).ext) - Number((nodes[b[0]] as GNode).ext) || a[0] - b[0]);
    const boxes: number[][] = [];
    const placed = new Map<number, number[]>();
    const drawn: { i: number; text: string; x: number; y: number }[] = [];
    const hitsLabel = (b: number[]) =>
      cellsOf(b[0] as number, b[1] as number, b[2] as number, b[3] as number).some((key) =>
        (placed.get(key) ?? []).some((j) => {
          const o = boxes[j] as number[];
          return (b[0] as number) < (o[2] as number) && (b[2] as number) > (o[0] as number) && (b[1] as number) < (o[3] as number) && (b[3] as number) > (o[1] as number);
        }),
      );
    const hitsNode = (b: number[], self: number) =>
      cellsOf(b[0] as number, b[1] as number, b[2] as number, b[3] as number).some((key) =>
        (circles.get(key) ?? []).some((j) => {
          if (j === self) return false;
          const m = nodes[j] as GNode;
          const x = m.x * k + tx;
          const y = m.y * k + ty;
          const dx = x - Math.max(b[0] as number, Math.min(x, b[2] as number));
          const dy = y - Math.max(b[1] as number, Math.min(y, b[3] as number));
          return dx * dx + dy * dy < (m.r * k) ** 2;
        }),
      );
    for (const [i, rk] of order) {
      const n = nodes[i] as GNode;
      const raw = n.title || t("Untitled");
      const text = raw.length > 40 ? `${raw.slice(0, 39)}…` : raw;
      let w = widths.get(text);
      if (w === undefined) {
        if (widths.size > 5000) widths.clear();
        w = g.measureText(text).width;
        widths.set(text, w);
      }
      const x = n.x * k + tx;
      const y = n.y * k + ty;
      const r = n.r * k;
      const spots = [
        [x, y + r + PAD + LH / 2],
        [x, y - r - PAD - LH / 2],
        [x + r + PAD + 2 + w / 2, y],
        [x - r - PAD - 2 - w / 2, y],
      ].map(([cx, cy]) => [(cx as number) - w / 2 - 2, (cy as number) - LH / 2, (cx as number) + w / 2 + 2, (cy as number) + LH / 2]);
      const clear = spots.filter((b) => !hitsLabel(b));
      const soft = rk < 4 || lab === "all";
      const box = clear.find((b) => !hitsNode(b, i)) ?? (soft ? clear[0] : undefined) ?? (rk <= 1 || lab === "all" ? spots[0] : undefined);
      if (!box) continue;
      const bi = boxes.push(box) - 1;
      for (const key of cellsOf(box[0] as number, box[1] as number, box[2] as number, box[3] as number)) push(placed, key, bi);
      drawn.push({ i, text, x: ((box[0] as number) + (box[2] as number)) / 2, y: ((box[1] as number) + (box[3] as number)) / 2 });
    }
    g.lineWidth = 3.5;
    g.strokeStyle = sheet;
    for (let d = drawn.length - 1; d >= 0; d--) {
      const { i, text, x, y } = drawn[d] as (typeof drawn)[number];
      const n = nodes[i] as GNode;
      const dim = (focus >= 0 && i !== focus && !near.has(i)) || (q && !match(n));
      g.globalAlpha = dim ? 0.35 : 1;
      g.strokeText(text, x, y);
      g.fillStyle = i === focus || i === selected ? textColor : muted;
      g.fillText(text, x, y);
    }
    g.globalAlpha = 1;
  }

  const nodeAt = (px: number, py: number): number => {
    const [wx, wy] = toWorld(px, py);
    let best = -1;
    let bd = Infinity;
    nodes.forEach((n, i) => {
      const d = (n.x - wx) ** 2 + (n.y - wy) ** 2;
      const rr = n.r + 4 / k;
      if (d <= rr * rr && d < bd) {
        bd = d;
        best = i;
      }
    });
    return best;
  };

  const edgeAt = (px: number, py: number): number => {
    const [wx, wy] = toWorld(px, py);
    const tol = 5 / k;
    let best = -1;
    let bd = tol;
    edges.forEach((e, i) => {
      const a = nodes[e.a];
      const b = nodes[e.b];
      if (!a || !b) return;
      const dx = b.x - a.x;
      const dy = b.y - a.y;
      const len = dx * dx + dy * dy || 1;
      const s = Math.max(0, Math.min(1, ((wx - a.x) * dx + (wy - a.y) * dy) / len));
      const d = Math.hypot(wx - (a.x + s * dx), wy - (a.y + s * dy));
      if (d < bd) {
        bd = d;
        best = i;
      }
    });
    return best;
  };

  const post = (m: SimIn, transfer: Transferable[] = []) => worker?.postMessage(m, transfer);

  const startSim = (alpha: number) => {
    if (!worker) {
      worker = new Worker(new URL("./graphsim.ts", import.meta.url), { type: "module", name: "graph-layout" });
      worker.onmessage = (e: MessageEvent<SimOut>) => {
        const m = e.data;
        if (m.pos.length !== nodes.length * 2) return;
        nodes.forEach((n, i) => {
          n.x = m.pos[i * 2] ?? n.x;
          n.y = m.pos[i * 2 + 1] ?? n.y;
        });
        if (m.done && !settled) {
          settled = true;
          setAttr(el, "data-settled", "true");
          if (!userView) fit(false);
        }
        redraw();
      };
    }
    settled = false;
    setAttr(el, "data-settled", null);
    const pos = new Float32Array(nodes.length * 2);
    nodes.forEach((n, i) => {
      pos[i * 2] = n.x;
      pos[i * 2 + 1] = n.y;
    });
    const links = new Int32Array(edges.length * 2);
    edges.forEach((e, i) => {
      links[i * 2] = e.a;
      links[i * 2 + 1] = e.b;
    });
    const groups = new Int32Array(nodes.map((n) => (conf().cluster === false ? -1 : n.group)));
    const radius = new Float32Array(nodes.map((n) => n.r));
    post({ type: "init", n: nodes.length, links, groups, radius, pos, alpha }, [links.buffer, groups.buffer, radius.buffer, pos.buffer]);
  };

  const build = (rows: RowData[], refs: Map<string, RowRef>, more: boolean) => {
    const c = conf();
    const old = new Map(nodes.map((n) => [n.id, n] as const));
    const colorProp = schema().find((p) => p.id === c.color && (p.type === "select" || p.type === "status"));
    const sizeProp = schema().find((p) => p.id === c.size && p.type === "number");
    const props = relProps();
    const next: GNode[] = [];
    index.clear();
    const add = (id: string, title: string, ext: boolean, col: string, color: string, group: number, val: number) => {
      const prev = old.get(id);
      index.set(id, next.length);
      next.push({ id, title, ext, col, color, group, val, deg: 0, r: 5, x: prev?.x ?? Number.NaN, y: prev?.y ?? Number.NaN });
    };
    for (const r of rows) {
      let color = "";
      let group = -1;
      if (colorProp) {
        const v = asIds(r.values[colorProp.id])[0] ?? "";
        const oi = colorProp.options?.findIndex((o) => o.id === v) ?? -1;
        if (oi >= 0) {
          color = colorProp.options?.[oi]?.color ?? "";
          group = oi;
        }
      }
      const raw = sizeProp ? r.values[sizeProp.id] : undefined;
      add(r.id, r.title, false, host.colId, color, group, typeof raw === "number" ? raw : 0);
    }
    const ne: GEdge[] = [];
    const seen = new Set<string>();
    const names = new Map<string, string>();
    let extCount = 0;
    for (const r of rows) {
      const ai = index.get(r.id) as number;
      for (const p of props) {
        const other = p.collection && p.collection !== host.colId ? p.collection : host.colId;
        for (const id of asIds(r.values[p.id])) {
          let bi = index.get(id);
          if (bi === undefined) {
            if (other === host.colId || c.external === false || extCount >= MAX_EXTERNAL) continue;
            const ref = refs.get(id);
            add(id, ref?.title ?? "", true, other, "", -1, 0);
            if (!names.has(other)) names.set(other, engine.pages.get(other)?.title || p.name);
            extCount++;
            bi = index.get(id) as number;
          }
          if (bi === ai) continue;
          const key = ai < bi ? `${ai}|${bi}` : `${bi}|${ai}`;
          if (seen.has(key)) continue;
          seen.add(key);
          ne.push({ a: ai, b: bi, prop: p.id, owner: ai, other: bi });
        }
      }
    }
    adj = next.map(() => []);
    for (const e of ne) {
      (adj[e.a] as number[]).push(e.b);
      (adj[e.b] as number[]).push(e.a);
    }
    const maxVal = Math.max(1, ...next.map((n) => Math.abs(n.val)));
    next.forEach((n, i) => {
      n.deg = adj[i]?.length ?? 0;
      n.r = sizeProp && !n.ext ? 4 + Math.sqrt(Math.abs(n.val) / maxVal) * 14 : Math.min(18, 4 + Math.sqrt(n.deg) * 2.2);
    });
    const selId = nodes[selected]?.id;
    nodes = next;
    edges = ne;
    selected = selId ? (index.get(selId) ?? -1) : -1;
    hover = -1;
    extNames = names;
    usedColors = (colorProp?.options ?? []).map((o) => o.color || "default");
    accentUsed = next.some((n) => !n.ext && n.group < 0);
    setText(count, more ? t("First {n} rows, {m} links", { n: rows.length, m: edges.length }) : t("{n} rows, {m} links", { n: rows.length, m: edges.length }));
    paintLegend(colorProp);
    paintList();
  };

  const paintList = () => {
    reconcile(
      a11y,
      nodes.map((n, i) => ({ n, i })),
      (x) => x.n.id,
      () => {
        const b = h("button", { class: "graphv__a11y-row", attrs: { type: "button" } });
        const li = h("li", {}, b);
        b.addEventListener("click", () => showNode(index.get(li.dataset.key ?? "") ?? -1));
        return li;
      },
      (li, { n, i }) => {
        const near = (adj[i] ?? []).map((j) => nodes[j]?.title || t("Untitled"));
        const title = n.title || t("Untitled");
        const names = near.slice(0, 5).join(", ") + (near.length > 5 ? ` ${t("and {n} more", { n: near.length - 5 })}` : "");
        setText(li.firstElementChild as HTMLElement, near.length ? t("{title}: linked to {names}", { title, names }) : t("{title}: no links", { title }));
      },
    );
  };

  const paintLegend = (p: Property | undefined) => {
    const items = (p?.options ?? []).map((o) => h("span", { class: "graphv__key" }, h("i", { attrs: { "data-color": o.color || "default" } }), h("span", { text: o.name })));
    const ext = [...extNames].map(([col, name]) => h("span", { class: "graphv__key graphv__key--ext" }, h("i", { attrs: { "data-ext": col } }), h("span", { text: name || t("Untitled") })));
    legend.replaceChildren(...items, ...ext);
    assignExt();
  };

  let seq = 0;
  const load = async (alpha = 1) => {
    const mine = ++seq;
    const rows: RowData[] = [];
    const refs = new Map<string, RowRef>();
    let total = 0;
    try {
      for (let offset = 0; offset < MAX_ROWS; offset += 1000) {
        const r = await engine.api.collections.query(host.colId, { view_id: host.viewId, offset, limit: Math.min(1000, MAX_ROWS - offset), no_groups: true, ...(host.search() ? { search: host.search() } : {}) });
        if (mine !== seq || destroyed) return;
        total = r.total;
        rows.push(...r.rows);
        for (const [id, ref] of Object.entries(r.refs ?? {})) refs.set(id, ref);
        if (rows.length >= r.total || !r.rows.length) break;
      }
    } catch (e) {
      if (mine === seq && !destroyed) engine.reportError(e);
      return;
    }
    if (mine !== seq || destroyed) return;
    rowVals.clear();
    for (const r of rows) rowVals.set(r.id, r.values as Record<string, unknown>);
    build(rows, refs, total > rows.length);
    paintSide();
    startSim(alpha);
  };

  let timer: ReturnType<typeof setTimeout> | undefined;
  const schedule = () => {
    clearTimeout(timer);
    timer = setTimeout(() => void load(0.25), 300);
  };

  const relationFor = (a: GNode, b: GNode): Property[] => schema().filter((p) => p.type === "relation" && (p.collection || host.colId) === b.col && !a.ext);

  const link = (ai: number, bi: number, e: PointerEvent) => {
    let a = nodes[ai];
    let b = nodes[bi];
    if (!a || !b || ai === bi) return;
    if (a.ext && !b.ext) [a, b] = [b, a];
    if (a.ext) return;
    const options = relationFor(a, b);
    const apply = (p: Property) => {
      const cur = new Set(rowValues(a as GNode, p.id));
      if (cur.has((b as GNode).id)) return;
      cur.add((b as GNode).id);
      engine.collections.setValue(host.colId, (a as GNode).id, p.id, [...cur]);
      remember((a as GNode).id, p.id, [...cur]);
      const x = index.get((a as GNode).id) as number;
      const y = index.get((b as GNode).id) as number;
      edges.push({ a: x, b: y, prop: p.id, owner: x, other: y });
      (adj[x] as number[]).push(y);
      (adj[y] as number[]).push(x);
      redraw();
    };
    if (!host.canWrite()) return;
    if (!options.length) {
      toast({ id: "graph-link", tone: "warning", message: t("Add a relation property to this database to draw links") });
      return;
    }
    if (options.length === 1) return apply(options[0] as Property);
    placeAnchor(e);
    openMenu(anchor, options.map((p) => ({ label: p.name, icon: "relation", run: () => apply(p) })), { label: t("Link with") });
  };

  const rowValues = (n: GNode, propId: string): string[] => asIds(rowVals.get(n.id)?.[propId]);
  const remember = (rowId: string, propId: string, value: unknown) => {
    const cur = { ...(rowVals.get(rowId) ?? {}) };
    if (value === null || (Array.isArray(value) && !value.length)) delete cur[propId];
    else cur[propId] = value;
    rowVals.set(rowId, cur);
  };

  const unlink = (ei: number) => {
    const e = edges[ei];
    if (!e) return;
    const a = nodes[e.owner];
    const b = nodes[e.other];
    if (!a || !b || a.ext) return;
    const cur = rowValues(a, e.prop).filter((id) => id !== b.id);
    engine.collections.setValue(host.colId, a.id, e.prop, cur);
    remember(a.id, e.prop, cur);
    edges.splice(ei, 1);
    adj[e.a] = (adj[e.a] ?? []).filter((x) => x !== e.b);
    adj[e.b] = (adj[e.b] ?? []).filter((x) => x !== e.a);
    redraw();
  };

  const placeAnchor = (e: { clientX: number; clientY: number }) => {
    const r = frame.getBoundingClientRect();
    anchor.style.transform = `translate(${Math.round(e.clientX - r.left)}px, ${Math.round(e.clientY - r.top)}px)`;
  };

  const addAt = async (wx: number, wy: number) => {
    if (!host.canWrite()) return;
    try {
      const row = await engine.collections.createRow(host.colId, { title: "" });
      const n: GNode = { id: row.id, title: "", ext: false, col: host.colId, color: "", group: -1, val: 0, deg: 0, r: 5, x: wx, y: wy };
      index.set(row.id, nodes.length);
      nodes.push(n);
      adj.push([]);
      selected = nodes.length - 1;
      paintSide(true);
      startSim(0.15);
    } catch (err) {
      engine.reportError(err);
    }
  };

  const titleInput = h("input", { class: "input graphv__title", attrs: { type: "text", "aria-label": t("Title"), placeholder: t("Untitled") } });
  const openBtn = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("expand"), h("span", { text: t("Open") }));
  const closeBtn = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Close") } }, iconNode("close"));
  const colorSel = h("select", { class: "input input--sm", attrs: { "aria-label": t("Colour property") } });
  const colorRow = h("label", { class: "field field--compact" }, h("span", { class: "field__label" }), colorSel);
  const linksList = h("ul", { class: "graphv__links", attrs: { role: "list" } });
  side.append(h("div", { class: "graphv__sidehead" }, titleInput, closeBtn), colorRow, h("div", { class: "graphv__sublabel", text: t("Links") }), linksList, h("div", { class: "graphv__sidefoot" }, openBtn));
  let titleTimer: ReturnType<typeof setTimeout> | undefined;
  titleInput.addEventListener("input", () => {
    const n = nodes[selected];
    if (!n || n.ext) return;
    n.title = titleInput.value;
    redraw();
    clearTimeout(titleTimer);
    titleTimer = setTimeout(() => engine.collections.setValue(host.colId, n.id, "title", titleInput.value), 350);
  });
  colorSel.addEventListener("change", () => {
    const n = nodes[selected];
    const p = schema().find((x) => x.id === conf().color);
    if (!n || !p) return;
    engine.collections.setValue(host.colId, n.id, p.id, colorSel.value || null);
    remember(n.id, p.id, colorSel.value || null);
    const oi = p.options?.findIndex((o) => o.id === colorSel.value) ?? -1;
    n.color = oi >= 0 ? (p.options?.[oi]?.color ?? "") : "";
    n.group = oi;
    redraw();
  });
  openBtn.addEventListener("click", (e) => {
    const n = nodes[selected];
    if (n) host.openRow(n.id, e);
  });
  closeBtn.addEventListener("click", () => {
    selected = -1;
    paintSide();
    redraw();
  });

  function paintSide(focusTitle = false): void {
    const n = nodes[selected];
    side.hidden = !n;
    if (!n) return;
    if (document.activeElement !== titleInput) titleInput.value = n.title;
    titleInput.disabled = n.ext || !host.canWrite();
    const p = schema().find((x) => x.id === conf().color && (x.type === "select" || x.type === "status"));
    colorRow.hidden = !p || n.ext;
    if (p) {
      setText(colorRow.firstElementChild as HTMLElement, p.name);
      const cur = asIds(rowVals.get(n.id)?.[p.id])[0] ?? "";
      colorSel.replaceChildren(h("option", { attrs: { value: "" }, text: t("None") }), ...(p.options ?? []).map((o) => h("option", { attrs: { value: o.id }, text: o.name })));
      colorSel.value = (p.options ?? []).find((o) => o.color === n.color && o.id === cur)?.id ?? cur;
      colorSel.disabled = !host.canWrite();
    }
    const items = (adj[selected] ?? []).slice(0, 50).map((j) => {
      const m = nodes[j] as GNode;
      const go = h("button", { class: "graphv__link", attrs: { type: "button" }, text: m.title || t("Untitled") });
      go.addEventListener("click", () => {
        selected = j;
        paintSide();
        redraw();
      });
      const ei = edges.findIndex((e) => (e.a === selected && e.b === j) || (e.b === selected && e.a === j));
      const rm = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Remove link") } }, iconNode("close"));
      rm.hidden = !host.canWrite() || ei < 0 || !!nodes[edges[ei]?.owner ?? -1]?.ext;
      rm.addEventListener("click", () => {
        unlink(edges.findIndex((e) => (e.a === selected && e.b === j) || (e.b === selected && e.a === j)));
        paintSide();
      });
      return h("li", {}, go, rm);
    });
    linksList.replaceChildren(...items);
    if (focusTitle) titleInput.focus();
  }

  let drag: { kind: "node" | "pan" | "link"; i: number; x: number; y: number; moved: boolean; id: number } | null = null;
  const local = (e: { clientX: number; clientY: number }) => {
    const r = canvas.getBoundingClientRect();
    return [e.clientX - r.left, e.clientY - r.top] as const;
  };
  canvas.addEventListener("pointerdown", (e) => {
    if (e.button !== 0) return;
    const [px, py] = local(e);
    const i = nodeAt(px, py);
    canvas.setPointerCapture(e.pointerId);
    if (i >= 0 && (mode === "link" || e.shiftKey) && host.canWrite()) {
      const [wx, wy] = toWorld(px, py);
      rubber = { from: i, x: wx, y: wy };
      drag = { kind: "link", i, x: px, y: py, moved: false, id: e.pointerId };
    } else if (i >= 0) drag = { kind: "node", i, x: px, y: py, moved: false, id: e.pointerId };
    else drag = { kind: "pan", i: -1, x: px, y: py, moved: false, id: e.pointerId };
  });
  canvas.addEventListener("pointermove", (e) => {
    const [px, py] = local(e);
    if (!drag) {
      const i = nodeAt(px, py);
      if (i !== hover) {
        hover = i;
        canvas.style.cursor = i >= 0 ? (mode === "link" ? "crosshair" : "pointer") : edgeAt(px, py) >= 0 ? "pointer" : "";
        redraw();
      }
      return;
    }
    if (Math.abs(px - drag.x) + Math.abs(py - drag.y) > 3) drag.moved = true;
    if (drag.kind === "pan") {
      tx += px - drag.x;
      ty += py - drag.y;
      drag.x = px;
      drag.y = py;
      userView = true;
      redraw();
    } else if (drag.kind === "node" && drag.moved) {
      const [wx, wy] = toWorld(px, py);
      const n = nodes[drag.i];
      if (n) {
        n.x = wx;
        n.y = wy;
      }
      post({ type: "drag", i: drag.i, x: wx, y: wy });
      redraw();
    } else if (drag.kind === "link" && rubber) {
      const [wx, wy] = toWorld(px, py);
      rubber.x = wx;
      rubber.y = wy;
      hover = nodeAt(px, py);
      redraw();
    }
  });
  canvas.addEventListener("pointerup", (e) => {
    const d = drag;
    drag = null;
    if (!d) return;
    const [px, py] = local(e);
    if (d.kind === "link") {
      rubber = null;
      const j = nodeAt(px, py);
      if (j >= 0 && j !== d.i) link(d.i, j, e);
      redraw();
      return;
    }
    if (d.kind === "node") {
      if (d.moved) post({ type: "release", i: d.i });
      else {
        selected = d.i;
        paintSide();
        redraw();
      }
      return;
    }
    if (!d.moved) {
      const ei = edgeAt(px, py);
      if (ei >= 0 && host.canWrite()) {
        const ed = edges[ei] as GEdge;
        placeAnchor(e);
        const entries: MenuEntry[] = [
          { kind: "heading", label: `${nodes[ed.a]?.title || t("Untitled")} · ${nodes[ed.b]?.title || t("Untitled")}` },
          { label: t("Remove link"), icon: "trash", danger: true, run: () => unlink(edges.indexOf(ed)) },
        ];
        openMenu(anchor, entries, { label: t("Link") });
        return;
      }
      if (selected >= 0) {
        selected = -1;
        paintSide();
        redraw();
      }
    }
  });
  canvas.addEventListener("pointerleave", () => {
    if (!drag && hover >= 0) {
      hover = -1;
      redraw();
    }
  });
  canvas.addEventListener("dblclick", (e) => {
    const [px, py] = local(e);
    if (nodeAt(px, py) >= 0) {
      const n = nodes[nodeAt(px, py)];
      if (n) host.openRow(n.id, e);
      return;
    }
    const [wx, wy] = toWorld(px, py);
    void addAt(wx, wy);
  });
  canvas.addEventListener(
    "wheel",
    (e) => {
      e.preventDefault();
      const [px, py] = local(e);
      const f = Math.exp(-e.deltaY * 0.0015);
      const nk = Math.max(0.05, Math.min(6, k * f));
      tx = px - ((px - tx) * nk) / k;
      ty = py - ((py - ty) * nk) / k;
      k = nk;
      userView = true;
      redraw();
    },
    { passive: false },
  );
  canvas.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && selected >= 0) {
      selected = -1;
      paintSide();
      redraw();
    }
  });
  const zoomBy = (f: number) => {
    const [W, H] = size();
    const nk = Math.max(0.05, Math.min(6, k * f));
    tx = W / 2 - ((W / 2 - tx) * nk) / k;
    ty = H / 2 - ((H / 2 - ty) * nk) / k;
    k = nk;
    userView = true;
    redraw();
  };
  zoomIn.addEventListener("click", () => zoomBy(1.3));
  zoomOut.addEventListener("click", () => zoomBy(1 / 1.3));
  fitBtn.addEventListener("click", () => {
    userView = false;
    fit(false);
    redraw();
  });
  const setMode = (m: "move" | "link") => {
    mode = m;
    setAttr(moveBtn, "aria-pressed", m === "move" ? "true" : "false");
    setAttr(linkBtn, "aria-pressed", m === "link" ? "true" : "false");
    setAttr(el, "data-mode", m);
  };
  moveBtn.addEventListener("click", () => setMode("move"));
  linkBtn.addEventListener("click", () => setMode("link"));
  search.addEventListener("input", () => {
    query = search.value;
    redraw();
  });
  const showNode = (i: number) => {
    const n = nodes[i];
    if (!n) return;
    selected = i;
    const [W, H] = size();
    k = Math.max(k, 1.4);
    tx = W / 2 - n.x * k;
    ty = H / 2 - n.y * k;
    userView = true;
    paintSide();
    redraw();
  };
  search.addEventListener("keydown", (e) => {
    if (e.key !== "Enter") return;
    const q = search.value.trim().toLowerCase();
    showNode(nodes.findIndex((n) => q && n.title.toLowerCase().includes(q)));
  });

  const theme = new MutationObserver(() => {
    readTheme();
    redraw();
  });
  theme.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme", "data-dark-theme", "data-light-theme", "data-accent", "data-contrast"] });
  let themed = false;
  const ro = new ResizeObserver(() => {
    if (!themed && frame.isConnected) {
      themed = true;
      readTheme();
    }
    redraw();
  });
  ro.observe(frame);
  requestAnimationFrame(() => {
    if (frame.isConnected) {
      themed = true;
      readTheme();
    }
    redraw();
  });
  const ready = load(1);

  const save = (change: (c: GraphConfig) => GraphConfig) => saveConfig(ctx, host.viewId, (v) => ({ ...v, layout: { ...(v.layout ?? {}), graph: change({ ...((v.layout?.graph as GraphConfig | undefined) ?? {}) }) } }), 150);

  return {
    el,
    ready,
    ctx,
    configChanged(a, b) {
      const x = JSON.stringify([a.config.filter ?? null, a.config.layout?.graph ?? null]);
      const y = JSON.stringify([b.config.filter ?? null, b.config.layout?.graph ?? null]);
      if (x !== y) void load(0.6);
    },
    schemaChanged: () => void load(0.3),
    setSearch: () => void load(0.6),
    addNew() {
      const [W, H] = size();
      const [wx, wy] = toWorld(W / 2, H / 2);
      void addAt(wx, wy);
    },
    settings(anchorEl) {
      openPanel(
        anchorEl,
        (panel) => {
          const c = conf();
          const pad = h("div", { class: "dbpanel__pad dbpanel__stack chartset" });
          const field = (label: string, control: HTMLElement) => h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: label }), control);
          const toggle = (label: string, on: boolean, change: (v: boolean) => void) => {
            const row = h("label", { class: "dpick__toggle" }, h("span", { text: label }), h("input", { attrs: { type: "checkbox", role: "switch" } }));
            const i = row.querySelector("input") as HTMLInputElement;
            i.checked = on;
            i.addEventListener("change", () => change(i.checked));
            return row;
          };
          const colorPick = propSelect(ctx, c.color ?? "", (id) => save((x) => ({ ...x, color: id })), (p) => p.type === "select" || p.type === "status");
          const sizeSel = h("select", { class: "input input--sm", attrs: { "aria-label": t("Size by") } });
          sizeSel.append(h("option", { attrs: { value: "" }, text: t("Number of links") }), ...schema().filter((p) => p.type === "number").map((p) => h("option", { attrs: { value: p.id }, text: p.name })));
          sizeSel.value = c.size ?? "";
          sizeSel.addEventListener("change", () => save((x) => ({ ...x, size: sizeSel.value })));
          const rels = schema().filter((p) => p.type === "relation");
          const linkBoxes = rels.map((p) =>
            toggle(p.name, !c.links?.length || c.links.includes(p.id), (on) =>
              save((x) => {
                const cur = new Set(x.links?.length ? x.links : rels.map((r) => r.id));
                if (on) cur.add(p.id);
                else cur.delete(p.id);
                return { ...x, links: [...cur] };
              }),
            ),
          );
          const labels = h("select", { class: "input input--sm", attrs: { "aria-label": t("Labels") } });
          for (const [v, l] of [["auto", "When there is room"], ["all", "Always"], ["none", "Never"]] as const) labels.append(h("option", { attrs: { value: v }, text: t(l) }));
          labels.value = c.labels ?? "auto";
          labels.addEventListener("change", () => save((x) => ({ ...x, labels: labels.value as GraphConfig["labels"] & string })));
          pad.append(
            h("div", { class: "dbpanel__head", text: t("Nodes") }),
            field(t("Colour by"), colorPick),
            field(t("Size by"), sizeSel),
            toggle(t("Cluster by colour"), c.cluster !== false, (on) => save((x) => ({ ...x, cluster: on }))),
            field(t("Labels"), labels),
            h("div", { class: "dbpanel__head", text: t("Links") }),
            ...(linkBoxes.length ? linkBoxes : [h("p", { class: "dbpanel__note", text: t("Add a relation property to draw links between rows.") })]),
            toggle(t("Show linked rows from other databases"), c.external !== false, (on) => save((x) => ({ ...x, external: on }))),
          );
          panel.append(pad);
        },
        { label: t("Graph"), focus: false, align: "end" },
      );
    },
    destroy() {
      destroyed = true;
      clearTimeout(timer);
      clearTimeout(titleTimer);
      cancelAnimationFrame(raf);
      ro.disconnect();
      theme.disconnect();
      post({ type: "stop" });
      worker?.terminate();
      session.close();
      el.remove();
    },
  };
}
