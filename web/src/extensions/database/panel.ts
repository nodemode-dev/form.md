import { byId, h } from "../../ui/dom";
import { type Align, anchorTo, anchorToRect, type Rect, type Side } from "../../ui/floating";
import { openLayer } from "../../ui/layers";

export interface PanelOptions {
  side?: Side;
  align?: Align;
  className?: string;
  label: string;
  onClose?: () => void;
  focus?: boolean;
}

export interface PanelHandle {
  el: HTMLElement;
  close: () => void;
  reposition: () => void;
}

export function openPanel(anchor: HTMLElement | (() => Rect | null), build: (panel: HTMLElement, close: () => void) => void, o: PanelOptions): PanelHandle {
  const panel = h("div", { class: `popover dbpanel ${o.className ?? ""}`.trim(), attrs: { role: "dialog", "aria-label": o.label } });
  let closed = false;
  let release: () => void = () => undefined;
  let reposition: () => void = () => undefined;
  let closeLayer: () => void = () => undefined;
  const close = () => closeLayer();
  build(panel, close);
  byId("overlay-root").append(panel);
  if (typeof anchor === "function") {
    const placed = anchorToRect(panel, anchor, { side: o.side ?? "bottom", align: o.align ?? "start", offset: 4 });
    release = placed.release;
    reposition = placed.update;
  } else {
    release = anchorTo(panel, anchor, { side: o.side ?? "bottom", align: o.align ?? "start", offset: 4 });
    reposition = () => {
      release();
      release = anchorTo(panel, anchor, { side: o.side ?? "bottom", align: o.align ?? "start", offset: 4 });
    };
    anchor.setAttribute("aria-expanded", "true");
  }
  requestAnimationFrame(() => panel.classList.add("is-visible"));
  const first = panel.querySelector<HTMLElement>("input:not([type=checkbox]), textarea, [data-autofocus]");
  closeLayer = openLayer({
    el: panel,
    modal: false,
    dismissOnOutside: true,
    returnFocus: typeof anchor === "function" ? null : anchor,
    ...(o.focus !== false && first ? { initialFocus: first } : {}),
    onClose: () => {
      if (closed) return;
      closed = true;
      release();
      if (typeof anchor !== "function") anchor.setAttribute("aria-expanded", "false");
      panel.remove();
      o.onClose?.();
    },
  });
  return { el: panel, close, reposition: () => reposition() };
}

export function rectOf(el: Element): () => Rect | null {
  return () => {
    if (!el.isConnected) return null;
    const r = el.getBoundingClientRect();
    return { x: r.left, y: r.top, width: r.width, height: r.height };
  };
}
