import { h, setAttr, setText } from "./dom";

export const COLORS = ["violet", "blue", "cyan", "teal", "green", "lime", "amber", "orange", "rose", "pink"] as const;

export function initials(name: string): string {
  const out: string[] = [];
  for (const word of name.trim().split(/\s+/)) {
    const first = [...word][0];
    if (first) out.push(first.toUpperCase());
    if (out.length === 2) break;
  }
  return out.join("");
}

export function tileColor(name: string): string {
  let hash = 0;
  for (const ch of name) hash = (Math.imul(hash, 31) + (ch.codePointAt(0) ?? 0)) >>> 0;
  return COLORS[hash % COLORS.length] ?? "violet";
}

export function avatar(name: string, color: string, large = false): HTMLSpanElement {
  const el = h("span", { class: large ? "avatar avatar--lg" : "avatar", attrs: { "aria-hidden": "true" } });
  updateAvatar(el, name, color);
  return el;
}

export function updateAvatar(el: HTMLElement, name: string, color: string): void {
  setAttr(el, "data-color", color);
  setText(el, initials(name));
}

export function tile(name: string): HTMLSpanElement {
  const el = h("span", { class: "tile", attrs: { "aria-hidden": "true" } });
  updateTile(el, name);
  return el;
}

export function updateTile(el: HTMLElement, name: string): void {
  setAttr(el, "data-color", tileColor(name));
  setText(el, [...initials(name)][0] ?? "");
}
