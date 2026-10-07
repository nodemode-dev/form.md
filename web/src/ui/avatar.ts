import { h, setAttr, setText } from "./dom";
import { parseIcon } from "./glyphs";
import { paintGlyph, paintPicture } from "./pageicon";

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

export function avatar(name: string, color: string, large = false, picture = ""): HTMLSpanElement {
  const el = h("span", { class: large ? "avatar avatar--lg" : "avatar", attrs: { "aria-hidden": "true" } });
  updateAvatar(el, name, color, picture);
  return el;
}

function paintLetters(el: HTMLElement, text: string): void {
  if (el.firstElementChild) el.replaceChildren();
  delete el.dataset.glyph;
  setText(el, text);
}

export function updateAvatar(el: HTMLElement, name: string, color: string, picture = ""): void {
  setAttr(el, "data-color", color);
  const ref = parseIcon(picture);
  setAttr(el, "data-kind", ref.kind === "file" ? "picture" : null);
  if (ref.kind === "file") paintPicture(el, ref.id);
  else paintLetters(el, initials(name));
}

export function tile(name: string, icon = "", image = ""): HTMLSpanElement {
  const el = h("span", { class: "tile", attrs: { "aria-hidden": "true" } });
  updateTile(el, name, icon, image);
  return el;
}

export function updateTile(el: HTMLElement, name: string, icon = "", image = ""): void {
  const picture = parseIcon(image);
  const glyph = parseIcon(icon);
  if (picture.kind === "file" || glyph.kind === "file") {
    setAttr(el, "data-kind", "picture");
    setAttr(el, "data-color", null);
    paintPicture(el, picture.kind === "file" ? picture.id : glyph.kind === "file" ? glyph.id : "");
    return;
  }
  if (glyph.kind === "glyph") {
    setAttr(el, "data-kind", "glyph");
    setAttr(el, "data-color", glyph.tint || tileColor(name));
    paintGlyph(el, glyph.name);
    return;
  }
  setAttr(el, "data-kind", null);
  setAttr(el, "data-color", tileColor(name));
  paintLetters(el, [...initials(name)][0] ?? "");
}
