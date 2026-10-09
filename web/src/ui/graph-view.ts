import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import { simpleDialog } from "./dialog";
import { h, setText } from "./dom";

interface GNode {
  id: string;
  title: string;
  kind: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  deg: number;
  fixed: boolean;
}

interface GEdge {
  from: string;
  to: string;
  kind: string;
}

function layout(nodes: GNode[], edges: { a: GNode; b: GNode }[], ticks: number): void {
  const n = nodes.length;
  if (!n) return;
  const cell = 90;
  for (let step = 0; step < ticks; step++) {
    const alpha = 1 - step / ticks;
    const grid = new Map<string, GNode[]>();
    for (const p of nodes) {
      const k = `${Math.floor(p.x / cell)}:${Math.floor(p.y / cell)}`;
      let bucket = grid.get(k);
      if (!bucket) grid.set(k, (bucket = []));
      bucket.push(p);
    }
    for (const p of nodes) {
      const cx = Math.floor(p.x / cell);
      const cy = Math.floor(p.y / cell);
      for (let dx = -2; dx <= 2; dx++)
        for (let dy = -2; dy <= 2; dy++) {
          for (const q of grid.get(`${cx + dx}:${cy + dy}`) ?? []) {
            if (q === p) continue;
            let ux = p.x - q.x;
            let uy = p.y - q.y;
            let d2 = ux * ux + uy * uy;
            if (d2 < 0.01) {
              ux = Math.random() - 0.5;
              uy = Math.random() - 0.5;
              d2 = 0.25;
            }
            if (d2 > cell * cell * 4) continue;
            const f = (900 * alpha) / d2;
            p.vx += ux * f;
            p.vy += uy * f;
          }
        }
      p.vx -= p.x * 0.004 * alpha;
      p.vy -= p.y * 0.004 * alpha;
    }
    for (const { a, b } of edges) {
      const ux = b.x - a.x;
      const uy = b.y - a.y;
      const d = Math.sqrt(ux * ux + uy * uy) || 1;
      const f = ((d - 60) / d) * 0.06 * alpha;
      a.vx += ux * f;
      a.vy += uy * f;
      b.vx -= ux * f;
      b.vy -= uy * f;
    }
    for (const p of nodes) {
      if (p.fixed) {
        p.vx = p.vy = 0;
        continue;
      }
      p.x += Math.max(-30, Math.min(30, p.vx));
      p.y += Math.max(-30, Math.min(30, p.vy));
      p.vx *= 0.55;
      p.vy *= 0.55;
    }
  }
}

export function openGraph(engine: Engine): void {
  const ws = engine.state.get("workspaceId");
  const handle = simpleDialog(
    t("Link graph"),
    (body) => {
      const canvas = h("canvas", { class: "graphv__canvas", attrs: { role: "img", "aria-label": t("Pages and the links between them") } });
      const info = h("div", { class: "graphv__info" });
      const legend = h("div", { class: "graphv__legend" }, h("span", { class: "graphv__key graphv__key--link", text: t("Links") }), h("span", { class: "graphv__key graphv__key--child", text: t("Inside") }));
      const frame = h("div", { class: "graphv__frame" }, canvas, h("div", { class: "graphv__bar" }, info, legend));
      body.append(frame);
      let nodes: GNode[] = [];
      let edges: { a: GNode; b: GNode; kind: string }[] = [];
      let scale = 1;
      let ox = 0;
      let oy = 0;
      let hover: GNode | null = null;
      let raf = 0;
      const css = getComputedStyle(document.documentElement);
      const color = (v: string, d: string) => css.getPropertyValue(v).trim() || d;
      const draw = () => {
        raf = 0;
        const dpr = window.devicePixelRatio || 1;
        const w = frame.clientWidth;
        const hh = canvas.clientHeight;
        if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(hh * dpr)) {
          canvas.width = Math.round(w * dpr);
          canvas.height = Math.round(hh * dpr);
        }
        const g = canvas.getContext("2d");
        if (!g) return;
        g.setTransform(dpr, 0, 0, dpr, 0, 0);
        g.clearRect(0, 0, w, hh);
        g.translate(w / 2 + ox, hh / 2 + oy);
        g.scale(scale, scale);
        const near = hover ? new Set(edges.filter((e) => e.a === hover || e.b === hover).flatMap((e) => [e.a, e.b])) : null;
        g.lineWidth = 1 / scale;
        for (const e of edges) {
          const lit = !near || (near.has(e.a) && near.has(e.b) && (e.a === hover || e.b === hover));
          g.strokeStyle = e.kind === "child" ? color("--border-strong", "#555") : color("--accent", "#88f");
          g.globalAlpha = lit ? (e.kind === "child" ? 0.7 : 0.55) : 0.08;
          g.beginPath();
          g.moveTo(e.a.x, e.a.y);
          g.lineTo(e.b.x, e.b.y);
          g.stroke();
        }
        g.globalAlpha = 1;
        for (const p of nodes) {
          const r = 3 + Math.min(9, Math.sqrt(p.deg) * 1.6);
          const dim = near && !near.has(p);
          g.globalAlpha = dim ? 0.2 : 1;
          g.fillStyle = p === hover ? color("--c-rose", "#f77") : p.kind === "collection" ? color("--c-teal", "#2dd") : color("--text-muted", "#aaa");
          g.beginPath();
          g.arc(p.x, p.y, r / Math.max(0.6, Math.sqrt(scale)), 0, Math.PI * 2);
          g.fill();
        }
        g.globalAlpha = 1;
        g.fillStyle = color("--text", "#eee");
        g.font = `${12 / scale}px ${color("--font-sans", "sans-serif")}`;
        g.textAlign = "center";
        for (const p of nodes) {
          const show = p === hover || (near?.has(p) ?? false) || (scale > 1.4 && !near) || (p.deg >= 4 && !near);
          if (!show) continue;
          g.fillText(p.title || t("Untitled"), p.x, p.y - 12 / scale);
        }
      };
      const paint = () => {
        if (!raf) raf = requestAnimationFrame(draw);
      };
      const toWorld = (cx: number, cy: number) => {
        const r = canvas.getBoundingClientRect();
        return { x: (cx - r.left - r.width / 2 - ox) / scale, y: (cy - r.top - r.height / 2 - oy) / scale };
      };
      const hit = (cx: number, cy: number): GNode | null => {
        const w = toWorld(cx, cy);
        let best: GNode | null = null;
        let bd = (14 / scale) ** 2;
        for (const p of nodes) {
          const d = (p.x - w.x) ** 2 + (p.y - w.y) ** 2;
          if (d < bd) {
            bd = d;
            best = p;
          }
        }
        return best;
      };
      const pointers = new Map<number, { x: number; y: number }>();
      let drag: { node: GNode | null; x: number; y: number; ox: number; oy: number; moved: boolean } | null = null;
      let pinch: { d: number; scale: number } | null = null;
      canvas.addEventListener("pointerdown", (e) => {
        canvas.setPointerCapture(e.pointerId);
        pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
        if (pointers.size === 2) {
          const [a, b] = [...pointers.values()];
          pinch = { d: Math.hypot((a?.x ?? 0) - (b?.x ?? 0), (a?.y ?? 0) - (b?.y ?? 0)), scale };
          drag = null;
          return;
        }
        const node = hit(e.clientX, e.clientY);
        if (node) node.fixed = true;
        drag = { node, x: e.clientX, y: e.clientY, ox, oy, moved: false };
      });
      canvas.addEventListener("pointermove", (e) => {
        if (pointers.has(e.pointerId)) pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
        if (pinch && pointers.size === 2) {
          const [a, b] = [...pointers.values()];
          const d = Math.hypot((a?.x ?? 0) - (b?.x ?? 0), (a?.y ?? 0) - (b?.y ?? 0));
          scale = Math.max(0.2, Math.min(5, (pinch.scale * d) / (pinch.d || 1)));
          return paint();
        }
        if (!drag) {
          const over = hit(e.clientX, e.clientY);
          if (over !== hover) {
            hover = over;
            canvas.style.cursor = over ? "pointer" : "grab";
            setText(info, over ? over.title || t("Untitled") : t("{n} pages, {m} links", { n: nodes.length, m: edges.length }));
            paint();
          }
          return;
        }
        const dx = e.clientX - drag.x;
        const dy = e.clientY - drag.y;
        if (!drag.moved && Math.hypot(dx, dy) < 4) return;
        drag.moved = true;
        if (drag.node) {
          const w = toWorld(e.clientX, e.clientY);
          drag.node.x = w.x;
          drag.node.y = w.y;
        } else {
          ox = drag.ox + dx;
          oy = drag.oy + dy;
        }
        paint();
      });
      const end = (e: PointerEvent) => {
        pointers.delete(e.pointerId);
        if (pointers.size < 2) pinch = null;
        if (!drag) return;
        const d = drag;
        drag = null;
        if (d.node && !d.moved) {
          handle.close();
          engine.openPage(d.node.id);
        }
      };
      canvas.addEventListener("pointerup", end);
      canvas.addEventListener("pointercancel", end);
      canvas.addEventListener(
        "wheel",
        (e) => {
          e.preventDefault();
          const r = canvas.getBoundingClientRect();
          const px = e.clientX - r.left - r.width / 2;
          const py = e.clientY - r.top - r.height / 2;
          const next = Math.max(0.2, Math.min(5, scale * Math.exp(-e.deltaY * 0.0015)));
          ox = px - ((px - ox) * next) / scale;
          oy = py - ((py - oy) * next) / scale;
          scale = next;
          paint();
        },
        { passive: false },
      );
      const ro = new ResizeObserver(paint);
      ro.observe(frame);
      setText(info, t("Loading"));
      engine.api.http
        .get<{ nodes: { id: string; title: string; kind: string }[]; edges: GEdge[]; truncated?: boolean }>(`/api/workspaces/${encodeURIComponent(ws)}/graph`)
        .then((g) => {
          const by = new Map<string, GNode>();
          const n = g.nodes.length;
          nodes = g.nodes.map((x, i) => {
            const a = (i / Math.max(1, n)) * Math.PI * 2;
            const r = 40 + Math.sqrt(n) * 18;
            const node: GNode = { id: x.id, title: x.title, kind: x.kind, x: Math.cos(a) * r, y: Math.sin(a) * r, vx: 0, vy: 0, deg: 0, fixed: false };
            by.set(x.id, node);
            return node;
          });
          edges = g.edges.flatMap((e) => {
            const a = by.get(e.from);
            const b = by.get(e.to);
            if (!a || !b) return [];
            a.deg++;
            b.deg++;
            return [{ a, b, kind: e.kind }];
          });
          layout(nodes, edges, n > 1500 ? 120 : 260);
          let x0 = Infinity;
          let x1 = -Infinity;
          let y0 = Infinity;
          let y1 = -Infinity;
          for (const p of nodes) {
            x0 = Math.min(x0, p.x);
            x1 = Math.max(x1, p.x);
            y0 = Math.min(y0, p.y);
            y1 = Math.max(y1, p.y);
          }
          if (nodes.length) {
            scale = Math.max(0.2, Math.min(2, Math.min((frame.clientWidth - 60) / Math.max(1, x1 - x0), (canvas.clientHeight - 60) / Math.max(1, y1 - y0))));
            ox = (-(x0 + x1) / 2) * scale;
            oy = (-(y0 + y1) / 2) * scale;
          }
          setText(info, t("{n} pages, {m} links", { n: nodes.length, m: edges.length }) + (g.truncated ? ` ${t("(the largest workspaces show their first pages)")}` : ""));
          frame.dataset.ready = "true";
          paint();
        })
        .catch((e: unknown) => {
          setText(info, String((e as Error).message));
        });
    },
    undefined,
    "md",
  );
  handle.body.closest(".dialog")?.classList.add("dialog--graph");
}
