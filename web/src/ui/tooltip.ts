import type { Engine } from "../engine";
import { byId, h, setAttr, setText } from "./dom";
import { anchorTo, type Side } from "./floating";

const SHOW_DELAY = 450;
const WARM_WINDOW = 300;

export function mountTooltip(engine: Engine): { hide(): void; visible(): boolean } {
  const label = h("span", { class: "tooltip__label" });
  const keys = h("span", { class: "tooltip__keys" });
  const tip = h("div", { class: "tooltip", attrs: { role: "tooltip", id: "tooltip" } }, label, keys);
  tip.hidden = true;
  byId("overlay-root").append(tip);

  let current: HTMLElement | null = null;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let release: (() => void) | null = null;
  let lastHidden = 0;

  const renderKeys = (parts: string[] | undefined) => {
    const wanted = parts ?? [];
    while (keys.childElementCount > wanted.length) keys.lastElementChild?.remove();
    wanted.forEach((part, i) => {
      const existing = keys.children[i];
      if (existing) setText(existing, part);
      else keys.append(h("kbd", { text: part }));
    });
  };

  const show = (el: HTMLElement) => {
    const text = el.dataset.tip;
    if (!text) return;
    current = el;
    setText(label, text);
    renderKeys(el.dataset.command ? engine.keymap.chordFor(el.dataset.command) : undefined);
    tip.classList.remove("is-visible");
    tip.hidden = false;
    release?.();
    release = anchorTo(tip, el, { side: (el.dataset.tipSide as Side | undefined) ?? "bottom" });
    setAttr(el, "aria-describedby", "tooltip");
    requestAnimationFrame(() => {
      if (current === el) tip.classList.add("is-visible");
    });
  };

  const hide = () => {
    clearTimeout(timer);
    if (!current) return;
    setAttr(current, "aria-describedby", null);
    current = null;
    release?.();
    release = null;
    tip.classList.remove("is-visible");
    tip.hidden = true;
    lastHidden = performance.now();
  };

  const schedule = (el: HTMLElement) => {
    clearTimeout(timer);
    const warm = performance.now() - lastHidden < WARM_WINDOW;
    timer = setTimeout(() => show(el), warm ? 0 : SHOW_DELAY);
  };

  const tipTarget = (t: EventTarget | null) => (t instanceof Element ? t.closest<HTMLElement>("[data-tip]") : null);

  document.addEventListener("pointerover", (e) => {
    if (e.pointerType === "touch") return;
    const el = tipTarget(e.target);
    if (el && el !== current) schedule(el);
  });
  document.addEventListener("pointerout", (e) => {
    const el = tipTarget(e.target);
    if (!el) return;
    const to = e.relatedTarget as Node | null;
    if (to && el.contains(to)) return;
    if (el === current) hide();
    else clearTimeout(timer);
  });
  document.addEventListener("focusin", (e) => {
    const el = tipTarget(e.target);
    if (el && el.matches(":focus-visible")) schedule(el);
  });
  document.addEventListener("focusout", (e) => {
    if (tipTarget(e.target)) hide();
  });
  document.addEventListener("pointerdown", hide, true);
  document.addEventListener("keydown", (e) => e.key === "Escape" && hide(), true);
  window.addEventListener("blur", hide);

  return { hide, visible: () => current !== null };
}
