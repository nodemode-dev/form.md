import { h, setAttr, setText } from "../../../web/src/ui/dom";
import { iconNode } from "../../../web/src/ui/icons";
import { passwords, sites } from "./server/db";
import { collect, renderGate, renderMissing, renderPage, type Ctx } from "./server/site";

declare global {
  interface Window {
    __pubGo?: (id: string) => void;
    __pubUnlock?: (password: string) => boolean;
  }
}

const ORIGIN = "https://fielddesk.example";

let root = "";
let current = "";
let unlocked = new Set<string>();
let built: { el: HTMLElement; frame: HTMLIFrameElement; url: HTMLElement; phone: HTMLButtonElement; wide: HTMLButtonElement } | null = null;

function build() {
  const frame = h("iframe", { class: "pubview__frame", attrs: { title: "Published page" } });
  const url = h("span", { class: "pubview__url" });
  const wide = h("button", { class: "segmented__btn segmented__btn--text", attrs: { type: "button", "aria-pressed": "true" }, text: "Desktop" });
  const phone = h("button", { class: "segmented__btn segmented__btn--text", attrs: { type: "button", "aria-pressed": "false" }, text: "Phone" });
  const close = h("button", { class: "btn btn--ghost", attrs: { type: "button" } }, iconNode("close"), h("span", { text: "Close" }));
  const el = h(
    "div",
    { class: "pubview", attrs: { role: "dialog", "aria-label": "Published page" } },
    h("header", { class: "pubview__bar" }, h("span", { class: "pubview__dot", attrs: { "aria-hidden": "true" } }), url, h("div", { class: "segmented", attrs: { role: "group", "aria-label": "Width" } }, wide, phone), close),
    h("div", { class: "pubview__stage" }, frame),
  );
  el.hidden = true;
  document.body.append(el);
  const width = (narrow: boolean) => {
    setAttr(el, "data-width", narrow ? "phone" : "desktop");
    setAttr(wide, "aria-pressed", narrow ? "false" : "true");
    setAttr(phone, "aria-pressed", narrow ? "true" : "false");
  };
  wide.addEventListener("click", () => width(false));
  phone.addEventListener("click", () => width(true));
  close.addEventListener("click", closePreview);
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !el.hidden && !document.querySelector(".dialog, .menu")) closePreview();
  });
  return { el, frame, url, phone, wide };
}

function context(): Ctx | null {
  const site = sites.get(root);
  if (!site) return null;
  return {
    mode: "site",
    root,
    nodes: collect(root, site.subpages),
    view: { theme: site.theme, font: site.font, indexing: site.indexing, title: site.title, description: site.description },
    origin: ORIGIN,
    href: (id) => (id === root ? site.path : `${site.path}/${id}`),
    src: (path) => path,
  };
}

function paint(): void {
  if (!built) return;
  const site = sites.get(root);
  const ctx = context();
  if (!site || !ctx) {
    built.frame.srcdoc = renderMissing({ mode: "site", root, nodes: [], view: { theme: "auto", font: "sans", indexing: false, title: "", description: "" }, origin: ORIGIN, href: () => "", src: (p) => p });
    return;
  }
  setText(built.url, `${ORIGIN.replace("https://", "")}${current === root ? site.path : `${site.path}/${current}`}`);
  built.frame.srcdoc = site.password && !unlocked.has(root) ? renderGate(ctx, false) : ctx.nodes.some((n) => n.id === current) ? renderPage(current, ctx) : renderMissing(ctx);
}

export function openPreview(pageId: string): void {
  built ??= build();
  root = pageId;
  current = pageId;
  unlocked = new Set();
  window.__pubGo = (id) => {
    current = id;
    paint();
  };
  window.__pubUnlock = (pw) => {
    if (passwords.get(root) !== pw) return false;
    unlocked.add(root);
    paint();
    return true;
  };
  built.el.hidden = false;
  document.documentElement.setAttribute("data-pubview", "open");
  paint();
}

export function openPreviewBySlug(slug: string): boolean {
  for (const [pageId, s] of sites) {
    if (s.slug === slug) {
      openPreview(pageId);
      return true;
    }
  }
  return false;
}

export function closePreview(): void {
  if (!built) return;
  built.el.hidden = true;
  document.documentElement.removeAttribute("data-pubview");
}
