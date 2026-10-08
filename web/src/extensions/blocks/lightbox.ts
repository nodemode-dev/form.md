import { t } from "../../engine/i18n";
import { byId, h } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { openLayer } from "../../ui/layers";

export function openLightbox(src: string, alt: string): void {
  const img = h("img", { class: "lightbox__img", attrs: { src, alt, decoding: "async" } });
  const close = h("button", { class: "icon-btn lightbox__close", attrs: { type: "button", "aria-label": t("Close"), "data-tip": t("Close") } }, iconNode("close"));
  const box = h("div", { class: "lightbox", attrs: { role: "dialog", "aria-modal": "true", "aria-label": alt || t("Image") } }, img, close);
  byId("overlay-root").append(box);
  requestAnimationFrame(() => box.classList.add("is-visible"));
  let closed = false;
  const done = openLayer({
    el: box,
    modal: true,
    dismissOnOutside: false,
    initialFocus: close,
    onClose: () => {
      if (closed) return;
      closed = true;
      box.remove();
    },
  });
  box.addEventListener("click", (e) => {
    if (e.target === box || (e.target as Element).closest(".lightbox__close")) done();
  });
}
