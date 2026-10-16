import { t } from "../../engine/i18n";
import { h } from "../../ui/dom";
import { openPrompt } from "./popover";

export type MarkKind = "arrow" | "rect" | "ellipse" | "pen" | "text";

export interface Mark {
  k: MarkKind;
  c: string;
  s: number;
  p: number[];
  t?: string;
}

export interface Marks {
  v: 1;
  w: number;
  h: number;
  items: Mark[];
}

export const COLORS = ["#ef4444", "#f97316", "#facc15", "#22c55e", "#3b82f6", "#ffffff", "#111111"];
const NS = "http://www.w3.org/2000/svg";
const KINDS: MarkKind[] = ["arrow", "rect", "ellipse", "pen", "text"];

export function parseMarks(raw: unknown): Marks | null {
  if (typeof raw !== "string" || !raw) return null;
  try {
    const m = JSON.parse(raw) as Marks;
    if (m?.v !== 1 || !(m.w > 0) || !(m.h > 0) || !Array.isArray(m.items)) return null;
    m.items = m.items.filter((x) => KINDS.includes(x.k) && Array.isArray(x.p) && x.p.every((n) => Number.isFinite(n)) && /^#[0-9a-fA-F]{6}$/.test(x.c));
    return m;
  } catch {
    return null;
  }
}

function el<K extends keyof SVGElementTagNameMap>(tag: K, attrs: Record<string, string | number>): SVGElementTagNameMap[K] {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, String(v));
  return e;
}

export function arrowHead(x1: number, y1: number, x2: number, y2: number, s: number): string {
  const a = Math.atan2(y2 - y1, x2 - x1);
  const len = Math.max(10, s * 4);
  const p1 = [x2 - len * Math.cos(a - 0.45), y2 - len * Math.sin(a - 0.45)];
  const p2 = [x2 - len * Math.cos(a + 0.45), y2 - len * Math.sin(a + 0.45)];
  return `${x2},${y2} ${p1[0]},${p1[1]} ${p2[0]},${p2[1]}`;
}

export function markNodes(m: Mark): SVGElement[] {
  const base = { stroke: m.c, "stroke-width": m.s, fill: "none", "stroke-linecap": "round", "stroke-linejoin": "round" };
  const [x1 = 0, y1 = 0, x2 = 0, y2 = 0] = m.p;
  if (m.k === "arrow") return [el("line", { ...base, x1, y1, x2, y2 }), el("polygon", { points: arrowHead(x1, y1, x2, y2, m.s), fill: m.c, stroke: m.c, "stroke-width": 1, "stroke-linejoin": "round" })];
  if (m.k === "rect") return [el("rect", { ...base, x: Math.min(x1, x2), y: Math.min(y1, y2), width: Math.abs(x2 - x1), height: Math.abs(y2 - y1), rx: m.s })];
  if (m.k === "ellipse") return [el("ellipse", { ...base, cx: (x1 + x2) / 2, cy: (y1 + y2) / 2, rx: Math.abs(x2 - x1) / 2, ry: Math.abs(y2 - y1) / 2 })];
  if (m.k === "pen") {
    let d = "";
    for (let i = 0; i + 1 < m.p.length; i += 2) d += `${i ? "L" : "M"}${m.p[i]} ${m.p[i + 1]}`;
    return [el("path", { ...base, d })];
  }
  const text = el("text", { x: x1, y: y1, fill: m.c, "font-size": m.s * 6, "font-family": "Inter Variable, system-ui, sans-serif", "font-weight": 600, "paint-order": "stroke", stroke: m.c === "#111111" ? "#ffffff" : "#111111", "stroke-width": m.s * 0.6 });
  text.textContent = m.t ?? "";
  return [text];
}

export function paintMarks(svg: SVGSVGElement, marks: Marks | null): void {
  if (!marks || !marks.items.length) {
    svg.setAttribute("hidden", "");
    if (svg.childNodes.length) svg.replaceChildren();
    return;
  }
  svg.removeAttribute("hidden");
  const box = `0 0 ${marks.w} ${marks.h}`;
  if (svg.getAttribute("viewBox") !== box) svg.setAttribute("viewBox", box);
  const key = JSON.stringify(marks.items);
  if (svg.dataset.key === key) return;
  svg.dataset.key = key;
  svg.replaceChildren(...marks.items.flatMap(markNodes));
}

export function marksSvg(): SVGSVGElement {
  const svg = el("svg", { class: "media__marks", "aria-hidden": "true", preserveAspectRatio: "xMidYMid meet" }) as SVGSVGElement;
  svg.setAttribute("hidden", "");
  return svg;
}

export interface AnnotatorHost {
  frame: HTMLElement;
  img: HTMLImageElement;
  svg: SVGSVGElement;
  read(): Marks | null;
  write(next: Marks | null): void;
}

export function startAnnotating(host: AnnotatorHost, done: () => void): () => void {
  const { frame, img, svg } = host;
  const w = img.naturalWidth || img.width || 1000;
  const hgt = img.naturalHeight || img.height || 1000;
  let marks: Marks = host.read() ?? { v: 1, w, h: hgt, items: [] };
  const sx = marks.w / w;
  const sy = marks.h / hgt;
  let kind: MarkKind = "arrow";
  let color = COLORS[0]!;
  const stroke = Math.max(2, Math.round(Math.max(marks.w, marks.h) / 250));

  const bar = h("div", { class: "annot__bar", attrs: { role: "toolbar", "aria-label": t("Annotate") } });
  const tools: [MarkKind, string][] = [["arrow", "Arrow"], ["rect", "Box"], ["ellipse", "Ellipse"], ["pen", "Draw"], ["text", "Text"]];
  const toolBtns = tools.map(([k, label]) => {
    const b = h("button", { class: "btn btn--sm annot__tool", attrs: { type: "button", "aria-pressed": k === kind ? "true" : "false" }, text: t(label) });
    b.addEventListener("click", () => {
      kind = k;
      for (const x of toolBtns) x.setAttribute("aria-pressed", x === b ? "true" : "false");
    });
    return b;
  });
  const swatches = COLORS.map((c) => {
    const b = h("button", { class: "annot__swatch", attrs: { type: "button", "aria-label": c, "aria-pressed": c === color ? "true" : "false" } });
    b.style.setProperty("--swatch", c);
    b.addEventListener("click", () => {
      color = c;
      for (const x of swatches) x.setAttribute("aria-pressed", x === b ? "true" : "false");
    });
    return b;
  });
  const undo = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Undo") });
  const clear = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Clear") });
  const finish = h("button", { class: "btn btn--primary btn--sm", attrs: { type: "button" }, text: t("Done") });
  bar.append(...toolBtns, h("span", { class: "annot__sep" }), ...swatches, h("span", { class: "annot__sep" }), undo, clear, finish);
  frame.appendChild(bar);
  frame.classList.add("is-annotating");
  svg.removeAttribute("hidden");
  if (!svg.getAttribute("viewBox")) svg.setAttribute("viewBox", `0 0 ${marks.w} ${marks.h}`);

  const commit = () => {
    host.write(marks.items.length ? marks : null);
    paintMarks(svg, marks);
    svg.removeAttribute("hidden");
  };
  undo.addEventListener("click", () => {
    marks = { ...marks, items: marks.items.slice(0, -1) };
    commit();
  });
  clear.addEventListener("click", () => {
    marks = { ...marks, items: [] };
    commit();
  });

  const at = (e: PointerEvent): [number, number] => {
    const r = img.getBoundingClientRect();
    return [Math.round(((e.clientX - r.left) / r.width) * w * sx), Math.round(((e.clientY - r.top) / r.height) * hgt * sy)];
  };
  let draft: Mark | null = null;
  let draftNodes: SVGElement[] = [];
  const redraft = () => {
    for (const n of draftNodes) n.remove();
    draftNodes = draft ? markNodes(draft) : [];
    svg.append(...draftNodes);
  };
  const down = (e: PointerEvent) => {
    if (e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();
    const [x, y] = at(e);
    if (kind === "text") {
      const at = { x: e.clientX, y: e.clientY };
      const c = color;
      openPrompt({
        label: t("Text"),
        placeholder: t("Text"),
        action: t("Add"),
        rect: () => ({ x: at.x, y: at.y, width: 0, height: 0 }),
        onSubmit: (value) => {
          const label = value.trim().slice(0, 200);
          if (!label) return;
          marks = { ...marks, items: [...marks.items, { k: "text", c, s: stroke, p: [x, y], t: label }] };
          commit();
        },
      });
      return;
    }
    svg.setPointerCapture(e.pointerId);
    draft = { k: kind, c: color, s: stroke, p: kind === "pen" ? [x, y] : [x, y, x, y] };
    redraft();
  };
  const move = (e: PointerEvent) => {
    if (!draft) return;
    const [x, y] = at(e);
    if (draft.k === "pen") draft.p.push(x, y);
    else {
      draft.p[2] = x;
      draft.p[3] = y;
    }
    redraft();
  };
  const up = () => {
    if (!draft) return;
    const d = draft;
    draft = null;
    redraft();
    const big = d.k === "pen" ? d.p.length >= 4 : Math.hypot((d.p[2] ?? 0) - (d.p[0] ?? 0), (d.p[3] ?? 0) - (d.p[1] ?? 0)) > stroke * 2;
    if (!big) return;
    if (d.k === "pen" && d.p.length > 400) d.p = d.p.filter((_, i) => Math.floor(i / 2) % 2 === 0 || i >= d.p.length - 2);
    marks = { ...marks, items: [...marks.items, d] };
    commit();
  };
  svg.addEventListener("pointerdown", down);
  svg.addEventListener("pointermove", move);
  svg.addEventListener("pointerup", up);
  const stop = () => {
    svg.removeEventListener("pointerdown", down);
    svg.removeEventListener("pointermove", move);
    svg.removeEventListener("pointerup", up);
    bar.remove();
    frame.classList.remove("is-annotating");
    paintMarks(svg, host.read());
  };
  finish.addEventListener("click", () => {
    stop();
    done();
  });
  return stop;
}
