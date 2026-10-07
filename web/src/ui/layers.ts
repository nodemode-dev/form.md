import { byId } from "./dom";

export interface LayerOptions {
  el: HTMLElement;
  modal: boolean;
  dismissOnOutside?: boolean;
  initialFocus?: HTMLElement;
  returnFocus?: HTMLElement | null;
  onClose: () => void;
}

interface Layer extends LayerOptions {
  restoreFocus: HTMLElement | null;
}

const stack: Layer[] = [];
let listening = false;

function onPointerDown(e: PointerEvent): void {
  const top = stack.at(-1);
  if (!top || !top.dismissOnOutside) return;
  const target = e.target as Node | null;
  if (target && top.el.contains(target)) return;
  close(top);
}

function syncInert(): void {
  const app = byId("app");
  const modal = stack.some((l) => l.modal);
  if (app.inert !== modal) app.inert = modal;
}

function close(layer: Layer): void {
  const i = stack.indexOf(layer);
  if (i === -1) return;
  stack.splice(i, 1);
  syncInert();
  if (stack.length === 0 && listening) {
    document.removeEventListener("pointerdown", onPointerDown, true);
    listening = false;
  }
  layer.onClose();
  const back = layer.restoreFocus;
  if (back && back.isConnected && !back.closest("[inert]")) back.focus({ preventScroll: true });
}

export function openLayer(options: LayerOptions): () => void {
  const active = document.activeElement;
  const fallback = active instanceof HTMLElement && active !== document.body ? active : null;
  const layer: Layer = { ...options, restoreFocus: options.returnFocus ?? fallback };
  stack.push(layer);
  syncInert();
  if (!listening) {
    document.addEventListener("pointerdown", onPointerDown, true);
    listening = true;
  }
  options.initialFocus?.focus({ preventScroll: true });
  return () => close(layer);
}

export function closeTop(): boolean {
  const top = stack.at(-1);
  if (!top) return false;
  close(top);
  return true;
}

export function hasLayers(): boolean {
  return stack.length > 0;
}

export function hasModal(): boolean {
  return stack.some((l) => l.modal);
}
