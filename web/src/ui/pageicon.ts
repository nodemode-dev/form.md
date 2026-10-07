import { h, setAttr } from "./dom";
import { glyphNode, hasGlyph, parseIcon } from "./glyphs";

export function fileUrl(id: string): string {
  return `/files/${id}`;
}

export function paintGlyph(el: HTMLElement, name: string): void {
  const want = hasGlyph(name) ? name : "page";
  if (el.dataset.glyph === want && el.firstElementChild instanceof SVGSVGElement) return;
  el.dataset.glyph = want;
  el.replaceChildren(glyphNode(want));
}

export function paintPicture(el: HTMLElement, id: string): void {
  const src = fileUrl(id);
  const img = el.firstElementChild;
  if (img instanceof HTMLImageElement) {
    if (img.getAttribute("src") !== src) img.src = src;
    return;
  }
  delete el.dataset.glyph;
  el.replaceChildren(h("img", { attrs: { src, alt: "", decoding: "async", draggable: "false" } }));
}

export function paintIcon(el: HTMLElement, ref: string, fallback: string): void {
  const icon = parseIcon(ref);
  if (icon.kind === "file") {
    setAttr(el, "data-color", null);
    setAttr(el, "data-kind", "picture");
    paintPicture(el, icon.id);
    return;
  }
  setAttr(el, "data-kind", icon.kind === "glyph" ? "glyph" : "default");
  setAttr(el, "data-color", icon.kind === "glyph" && icon.tint ? icon.tint : null);
  paintGlyph(el, icon.kind === "glyph" ? icon.name : fallback);
}

export function pageIcon(ref: string, fallback: string, extra = ""): HTMLSpanElement {
  const el = h("span", { class: extra ? `page-icon ${extra}` : "page-icon", attrs: { "aria-hidden": "true" } });
  paintIcon(el, ref, fallback);
  return el;
}

export function fallbackGlyph(kind: string): string {
  return kind === "collection" ? "table" : "page";
}
