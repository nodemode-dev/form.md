import { byId, h, setAttr } from "./dom";
import { type Align, anchorTo, type Side } from "./floating";
import { type IconName, iconNode } from "./icons";
import { openLayer } from "./layers";

export type MenuEntry =
  | { kind?: "item"; label: string; icon?: IconName; lead?: HTMLElement; hint?: string; checked?: boolean; danger?: boolean; run: () => void }
  | { kind: "separator" }
  | { kind: "heading"; label: string };

export interface MenuOptions {
  side?: Side;
  align?: Align;
  label?: string;
}

export function openMenu(anchor: HTMLElement, entries: MenuEntry[], options: MenuOptions = {}): () => void {
  const menu = h("div", { class: "menu", attrs: { role: "menu", "aria-label": options.label ?? "" } });
  const items: HTMLButtonElement[] = [];
  let close: () => void = () => undefined;

  for (const entry of entries) {
    if (entry.kind === "separator") {
      menu.append(h("div", { class: "menu__sep", attrs: { role: "separator" } }));
      continue;
    }
    if (entry.kind === "heading") {
      menu.append(h("div", { class: "menu__heading", text: entry.label }));
      continue;
    }
    const lead = entry.lead ?? (entry.icon ? iconNode(entry.icon) : h("span"));
    const trail = entry.checked ? h("span", { class: "menu__check" }, iconNode("check")) : h("span", { class: "menu__hint", text: entry.hint ?? "" });
    const btn = h(
      "button",
      { class: "menu__item", attrs: { type: "button", role: entry.checked === undefined ? "menuitem" : "menuitemradio", tabindex: "-1" } },
      lead,
      h("span", { class: "menu__label", text: entry.label }),
      trail,
    );
    if (entry.checked !== undefined) setAttr(btn, "aria-checked", entry.checked ? "true" : "false");
    if (entry.danger) btn.dataset.danger = "true";
    btn.addEventListener("click", () => {
      close();
      delete anchor.dataset.menuClosedAt;
      entry.run();
    });
    items.push(btn);
    menu.append(btn);
  }

  const focusAt = (i: number) => {
    const n = items.length;
    if (n === 0) return;
    items[((i % n) + n) % n]?.focus();
  };
  menu.addEventListener("keydown", (e) => {
    const i = items.indexOf(document.activeElement as HTMLButtonElement);
    if (e.key === "ArrowDown") focusAt(i + 1);
    else if (e.key === "ArrowUp") focusAt(i - 1);
    else if (e.key === "Home") focusAt(0);
    else if (e.key === "End") focusAt(items.length - 1);
    else if (e.key === "Tab") close();
    else return;
    e.preventDefault();
  });
  menu.addEventListener("pointermove", (e) => {
    const btn = (e.target as Element).closest<HTMLButtonElement>(".menu__item");
    if (btn && document.activeElement !== btn) btn.focus({ preventScroll: true });
  });

  byId("overlay-root").append(menu);
  const release = anchorTo(menu, anchor, { side: options.side ?? "bottom", align: options.align ?? "start", offset: 4 });
  setAttr(anchor, "aria-expanded", "true");
  requestAnimationFrame(() => menu.classList.add("is-visible"));
  let closed = false;
  const closeLayer = openLayer({
    el: menu,
    modal: false,
    dismissOnOutside: true,
    returnFocus: anchor,
    onClose: () => {
      if (closed) return;
      closed = true;
      release();
      setAttr(anchor, "aria-expanded", "false");
      anchor.dataset.menuClosedAt = String(performance.now());
      menu.remove();
    },
  });
  close = closeLayer;
  focusAt(Math.max(0, items.findIndex((b) => b.getAttribute("aria-checked") === "true")));
  return close;
}

export function menuButton(anchor: HTMLElement, entries: () => MenuEntry[], options: MenuOptions = {}): void {
  anchor.addEventListener("click", () => {
    const closedAt = anchor.dataset.menuClosedAt;
    if (anchor.getAttribute("aria-expanded") === "true") return;
    if (closedAt !== undefined && performance.now() - Number(closedAt) < 350) return;
    openMenu(anchor, entries(), options);
  });
}
