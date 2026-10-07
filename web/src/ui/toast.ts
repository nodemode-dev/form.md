import { byId, h, setText } from "./dom";
import { iconNode } from "./icons";

export type Tone = "info" | "success" | "warning" | "danger";

export interface ToastOptions {
  message: string;
  tone?: Tone;
  id?: string;
  durationMs?: number;
  action?: { label: string; run: () => void };
}

export interface ToastHandle {
  dismiss(): void;
}

interface Live {
  el: HTMLElement;
  msg: HTMLElement;
  timer: ReturnType<typeof setTimeout> | undefined;
  remaining: number;
  startedAt: number;
  leaving: boolean;
}

const live = new Map<string, Live>();
let counter = 0;

function arm(t: Live): void {
  clearTimeout(t.timer);
  if (t.remaining <= 0) return;
  t.startedAt = performance.now();
  t.timer = setTimeout(() => leave(t), t.remaining);
}

function pause(t: Live): void {
  if (t.timer === undefined) return;
  clearTimeout(t.timer);
  t.timer = undefined;
  t.remaining = Math.max(1000, t.remaining - (performance.now() - t.startedAt));
}

function leave(t: Live): void {
  if (t.leaving) return;
  t.leaving = true;
  clearTimeout(t.timer);
  for (const [key, value] of live) if (value === t) live.delete(key);
  t.el.classList.add("is-leaving");
  const done = () => t.el.remove();
  t.el.addEventListener("transitionend", (e) => e.target === t.el && done(), { once: true });
  setTimeout(done, 400);
}

export function toast(options: ToastOptions): ToastHandle {
  const tone = options.tone ?? "info";
  const id = options.id ?? `toast-${++counter}`;
  const duration = options.durationMs ?? (options.action ? 8000 : 5000);

  const existing = live.get(id);
  if (existing) {
    setText(existing.msg, options.message);
    if (existing.el.dataset.tone !== tone) existing.el.dataset.tone = tone;
    existing.remaining = duration;
    arm(existing);
    return { dismiss: () => leave(existing) };
  }

  const msg = h("p", { class: "toast__msg", text: options.message });
  const close = h("button", { class: "icon-btn toast__close", attrs: { type: "button", "aria-label": "Dismiss" } }, iconNode("close"));
  const iconWrap = h("span", { class: "toast__icon" }, iconNode(tone));
  const inner = h("div", { class: "toast__inner" }, iconWrap, msg);
  if (options.action) {
    const { label, run } = options.action;
    const btn = h("button", { class: "toast__action", attrs: { type: "button" }, text: label });
    btn.addEventListener("click", () => {
      run();
      leave(t);
    });
    inner.append(btn);
  } else {
    inner.append(h("span"));
  }
  inner.append(close);

  const el = h(
    "div",
    { class: "toast is-entering", attrs: { "data-tone": tone, role: tone === "danger" ? "alert" : "status" } },
    h("div", { class: "toast__clip" }, inner),
  );
  const t: Live = { el, msg, timer: undefined, remaining: duration, startedAt: 0, leaving: false };
  live.set(id, t);

  close.addEventListener("click", () => leave(t));
  el.addEventListener("pointerenter", () => pause(t));
  el.addEventListener("pointerleave", () => arm(t));
  el.addEventListener("focusin", () => pause(t));
  el.addEventListener("focusout", () => arm(t));

  byId("toast-root").append(el);
  requestAnimationFrame(() => requestAnimationFrame(() => el.classList.remove("is-entering")));
  arm(t);
  return { dismiss: () => leave(t) };
}

export function dismissToast(id: string): void {
  const t = live.get(id);
  if (t) leave(t);
}
