import { h } from "../../ui/dom";

export function cursorBuilder(user: { name?: string; tint?: string }): HTMLElement {
  const label = h("span", { class: "collab-cursor__label", text: user.name ?? "" });
  const el = h("span", { class: "collab-cursor" }, document.createTextNode("⁠"), label, document.createTextNode("⁠"));
  el.dataset.color = user.tint ?? "violet";
  return el;
}

export function selectionBuilder(user: { tint?: string }): { class: string; "data-color": string } {
  return { class: "collab-selection", "data-color": user.tint ?? "violet" };
}
