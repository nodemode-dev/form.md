export type Side = "top" | "bottom" | "left" | "right";
export type Align = "start" | "center" | "end";

export interface PlaceOptions {
  side?: Side;
  align?: Align;
  offset?: number;
  padding?: number;
}

export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Placement {
  x: number;
  y: number;
  side: Side;
}

const OPPOSITE: Record<Side, Side> = { top: "bottom", bottom: "top", left: "right", right: "left" };

function along(anchorStart: number, anchorSize: number, size: number, align: Align): number {
  if (align === "start") return anchorStart;
  if (align === "end") return anchorStart + anchorSize - size;
  return anchorStart + (anchorSize - size) / 2;
}

function place(a: Rect, f: { width: number; height: number }, side: Side, align: Align, offset: number): { x: number; y: number } {
  switch (side) {
    case "top":
      return { x: along(a.x, a.width, f.width, align), y: a.y - f.height - offset };
    case "bottom":
      return { x: along(a.x, a.width, f.width, align), y: a.y + a.height + offset };
    case "left":
      return { x: a.x - f.width - offset, y: along(a.y, a.height, f.height, align) };
    case "right":
      return { x: a.x + a.width + offset, y: along(a.y, a.height, f.height, align) };
  }
}

function overflow(p: { x: number; y: number }, f: { width: number; height: number }, vw: number, vh: number, pad: number, side: Side): number {
  if (side === "top") return Math.max(0, pad - p.y);
  if (side === "bottom") return Math.max(0, p.y + f.height - (vh - pad));
  if (side === "left") return Math.max(0, pad - p.x);
  return Math.max(0, p.x + f.width - (vw - pad));
}

export function computePlacement(anchor: Rect, float: { width: number; height: number }, viewport: { width: number; height: number }, options: PlaceOptions = {}): Placement {
  const { side = "bottom", align = "center", offset = 6, padding = 8 } = options;
  let chosen = side;
  let pos = place(anchor, float, side, align, offset);
  const over = overflow(pos, float, viewport.width, viewport.height, padding, side);
  if (over > 0) {
    const flipped = OPPOSITE[side];
    const alt = place(anchor, float, flipped, align, offset);
    if (overflow(alt, float, viewport.width, viewport.height, padding, flipped) < over) {
      chosen = flipped;
      pos = alt;
    }
  }
  const clamp = (v: number, size: number, max: number) => Math.min(Math.max(v, padding), Math.max(padding, max - size - padding));
  if (chosen === "top" || chosen === "bottom") pos.x = clamp(pos.x, float.width, viewport.width);
  else pos.y = clamp(pos.y, float.height, viewport.height);
  return { x: Math.round(pos.x), y: Math.round(pos.y), side: chosen };
}

export function anchorTo(floating: HTMLElement, anchor: Element, options: PlaceOptions = {}): () => void {
  let frame = 0;
  const update = () => {
    frame = 0;
    if (!anchor.isConnected) return;
    const r = anchor.getBoundingClientRect();
    const p = computePlacement(
      { x: r.left, y: r.top, width: r.width, height: r.height },
      { width: floating.offsetWidth, height: floating.offsetHeight },
      { width: document.documentElement.clientWidth, height: document.documentElement.clientHeight },
      options,
    );
    floating.style.left = `${p.x}px`;
    floating.style.top = `${p.y}px`;
    if (floating.dataset.side !== p.side) floating.dataset.side = p.side;
  };
  const schedule = () => {
    if (!frame) frame = requestAnimationFrame(update);
  };
  update();
  window.addEventListener("scroll", schedule, { capture: true, passive: true });
  window.addEventListener("resize", schedule, { passive: true });
  return () => {
    cancelAnimationFrame(frame);
    window.removeEventListener("scroll", schedule, { capture: true });
    window.removeEventListener("resize", schedule);
  };
}
