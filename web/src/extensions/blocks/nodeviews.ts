import type { Node as PMNode } from "prosemirror-model";
import { NodeSelection } from "prosemirror-state";
import { TableView } from "prosemirror-tables";
import type { EditorView, NodeView, NodeViewConstructor, ViewMutationRecord } from "prosemirror-view";
import { EMBED_ALLOW, EMBED_SANDBOX, embedSrc, humanSize, isRemote, safeHref } from "../../engine/doc/schema";
import { formatDate } from "../../engine/format";
import { t } from "../../engine/i18n";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { reconcile } from "../../ui/list";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { fallbackGlyph, paintIcon } from "../../ui/pageicon";
import { openIconPicker } from "../../ui/pickers";
import type { EditorCtx, Refreshable } from "./context";
import { LANGUAGES, languageLabel } from "./languages";
import { openLightbox } from "./lightbox";
import { importMedia, importPhase, watchImport } from "./links";
import { ColumnView } from "./columns";
import { MathView } from "./math";

type GetPos = () => number | undefined;

function blockId(dom: HTMLElement, node: PMNode): void {
  setAttr(dom, "data-block", node.attrs.id ? String(node.attrs.id) : null);
}

class CodeView implements NodeView {
  dom: HTMLElement;
  contentDOM: HTMLElement;
  private bar: HTMLElement;
  private lang: HTMLButtonElement;
  private copy: HTMLButtonElement;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
  ) {
    this.lang = h("button", { class: "code__lang", attrs: { type: "button" } });
    this.copy = h("button", { class: "code__copy icon-btn", attrs: { type: "button", "aria-label": t("Copy code"), "data-tip": t("Copy code") } }, iconNode("copy"));
    this.bar = h("div", { class: "code__bar", attrs: { contenteditable: "false" } }, this.lang, this.copy);
    this.contentDOM = h("code");
    const pre = h("pre", { class: "code__pre", attrs: { spellcheck: "false" } }, this.contentDOM);
    this.dom = h("div", { class: "code" }, this.bar, pre);
    this.paint();
    this.lang.addEventListener("click", () => this.pickLanguage());
    this.copy.addEventListener("click", () => {
      void navigator.clipboard?.writeText(this.node.textContent).catch(() => undefined);
    });
  }

  private paint(): void {
    blockId(this.dom, this.node);
    const language = String(this.node.attrs.language ?? "");
    setAttr(this.dom, "data-language", language || null);
    setText(this.lang, languageLabel(language) || t("Plain text"));
    this.lang.disabled = !this.view.editable;
  }

  private pickLanguage(): void {
    if (!this.view.editable) return;
    const current = String(this.node.attrs.language ?? "");
    const entries: MenuEntry[] = [{ label: t("Plain text"), checked: current === "", run: () => this.setLanguage("") }];
    for (const [id, label] of LANGUAGES) entries.push({ label, checked: current === id, run: () => this.setLanguage(id) });
    openMenu(this.lang, entries, { side: "bottom", align: "start", label: t("Language") });
  }

  private setLanguage(language: string): void {
    const pos = this.getPos();
    if (pos === undefined) return;
    this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...this.node.attrs, language }));
    this.view.focus();
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    return this.bar.contains(e.target as Node);
  }

  ignoreMutation(m: ViewMutationRecord): boolean {
    return m.type !== "selection" && !this.contentDOM.contains(m.target);
  }
}

class TaskView implements NodeView {
  dom: HTMLElement;
  contentDOM: HTMLElement;
  private check: HTMLElement;
  private box: HTMLInputElement;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
  ) {
    this.box = h("input", { attrs: { type: "checkbox", "aria-label": t("Done") } });
    this.check = h("span", { class: "task__check", attrs: { contenteditable: "false" } }, this.box);
    this.contentDOM = h("div", { class: "task__body" });
    this.dom = h("li", { class: "task" }, this.check, this.contentDOM);
    this.box.addEventListener("mousedown", (e) => e.preventDefault());
    this.box.addEventListener("click", (e) => {
      e.preventDefault();
      if (!this.view.editable) return;
      const pos = this.getPos();
      if (pos === undefined) return;
      this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...this.node.attrs, checked: !this.node.attrs.checked }));
    });
    this.paint();
  }

  private paint(): void {
    blockId(this.dom, this.node);
    const checked = !!this.node.attrs.checked;
    setAttr(this.dom, "data-checked", checked ? "true" : "false");
    if (this.box.checked !== checked) this.box.checked = checked;
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    return this.check.contains(e.target as Node);
  }

  ignoreMutation(m: ViewMutationRecord): boolean {
    return m.type !== "selection" && this.check.contains(m.target);
  }
}

class ToggleView implements NodeView {
  dom: HTMLElement;
  contentDOM: HTMLElement;
  private btn: HTMLButtonElement;

  constructor(
    private node: PMNode,
    private readonly ctx: EditorCtx,
  ) {
    this.btn = h("button", { class: "toggle__btn", attrs: { type: "button", contenteditable: "false", "aria-label": t("Show or hide") } }, iconNode("chevron-right"));
    this.contentDOM = h("div", { class: "toggle__content" });
    this.dom = h("div", { class: "toggle" }, this.btn, this.contentDOM);
    this.btn.addEventListener("mousedown", (e) => e.preventDefault());
    this.btn.addEventListener("click", () => {
      const id = String(this.node.attrs.id ?? "");
      if (!id) return;
      if (ctx.openToggles.has(id)) ctx.openToggles.delete(id);
      else ctx.openToggles.add(id);
      this.paint();
    });
    this.paint();
  }

  private paint(): void {
    blockId(this.dom, this.node);
    const level = Number(this.node.attrs.level) || 0;
    setAttr(this.dom, "data-level", level ? String(level) : null);
    const open = this.ctx.openToggles.has(String(this.node.attrs.id ?? ""));
    this.dom.classList.toggle("is-open", open);
    setAttr(this.btn, "aria-expanded", open ? "true" : "false");
    setAttr(this.dom, "data-empty", this.node.childCount <= 1 ? "true" : null);
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    return this.btn.contains(e.target as Node);
  }

  ignoreMutation(m: ViewMutationRecord): boolean {
    return m.type !== "selection" && (m.target === this.dom || this.btn.contains(m.target));
  }
}

class CalloutView implements NodeView {
  dom: HTMLElement;
  contentDOM: HTMLElement;
  private icon: HTMLElement;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
    private readonly ctx: EditorCtx,
  ) {
    this.icon = h("span", { class: "callout__icon", attrs: { contenteditable: "false", role: "button", tabindex: "-1", "aria-label": t("Change icon") } });
    this.contentDOM = h("div", { class: "callout__body" });
    this.dom = h("div", { class: "doc-callout" }, this.icon, this.contentDOM);
    this.icon.addEventListener("mousedown", (e) => e.preventDefault());
    this.icon.addEventListener("click", () => this.pick());
    this.paint();
  }

  private pick(): void {
    if (!this.view.editable) return;
    openIconPicker(this.ctx.engine, this.icon, {
      current: String(this.node.attrs.icon ?? ""),
      upload: { purpose: "icon", page_id: this.ctx.pageId() },
      allowRemove: !!this.node.attrs.icon,
      onPick: (ref) => {
        const tint = ref.startsWith("i:") ? (ref.split(":")[2] ?? "") : "";
        this.set({ icon: ref, tone: tint || this.node.attrs.tone });
      },
      onRemove: () => this.set({ icon: "" }),
    });
  }

  private set(attrs: Record<string, unknown>): void {
    const pos = this.getPos();
    if (pos === undefined) return;
    this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...this.node.attrs, ...attrs }));
  }

  private paint(): void {
    blockId(this.dom, this.node);
    setAttr(this.dom, "data-tone", this.node.attrs.tone ? String(this.node.attrs.tone) : null);
    const ref = String(this.node.attrs.icon ?? "");
    setAttr(this.dom, "data-icon", ref || null);
    paintIcon(this.icon, ref.startsWith("i:") || ref.startsWith("f:") ? ref : "", "lightbulb");
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    return this.icon.contains(e.target as Node);
  }

  ignoreMutation(m: ViewMutationRecord): boolean {
    return m.type !== "selection" && this.icon.contains(m.target);
  }
}

class MentionView implements NodeView, Refreshable {
  dom: HTMLElement;
  private icon: HTMLElement | null = null;
  private label: HTMLElement | null = null;

  constructor(
    private node: PMNode,
    private readonly ctx: EditorCtx,
  ) {
    const kind = String(node.attrs.kind);
    if (kind === "page") {
      this.icon = h("span", { class: "mention__icon" });
      this.label = h("span", { class: "mention__label" });
      this.dom = h("a", { class: "mention", attrs: { contenteditable: "false" } }, this.icon, this.label);
    } else {
      this.dom = h("span", { class: "mention", attrs: { contenteditable: "false" } });
    }
    this.dom.dataset.kind = kind;
    ctx.live.add(this);
    this.refresh();
  }

  refresh(): void {
    const kind = String(this.node.attrs.kind);
    const ref = String(this.node.attrs.ref);
    this.dom.dataset.ref = ref;
    if (kind === "page" && this.icon && this.label) {
      const p = this.ctx.engine.pages.get(ref);
      setAttr(this.dom, "href", `/p/${ref}`);
      paintIcon(this.icon, p?.icon ?? "", fallbackGlyph(p?.kind ?? "doc"));
      setText(this.label, p ? p.title || t("Untitled") : String(this.node.attrs.label || t("Untitled")));
      setAttr(this.dom, "data-missing", p || this.node.attrs.label ? null : "true");
    } else if (kind === "user") {
      setText(this.dom, `@${String(this.node.attrs.label ?? "")}`);
    } else {
      const ms = new Date(`${ref}T12:00:00`).getTime();
      setText(this.dom, Number.isFinite(ms) ? formatDate(ms) : ref);
    }
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type || node.attrs.kind !== this.node.attrs.kind) return false;
    this.node = node;
    this.refresh();
    return true;
  }

  ignoreMutation(): boolean {
    return true;
  }

  destroy(): void {
    this.ctx.live.delete(this);
  }
}

class CrumbView implements NodeView, Refreshable {
  dom: HTMLElement;
  private sig = "";

  constructor(
    private node: PMNode,
    private readonly ctx: EditorCtx,
  ) {
    this.dom = h("nav", { class: "crumbline", attrs: { contenteditable: "false", "aria-label": t("Breadcrumb") } });
    blockId(this.dom, node);
    this.dom.addEventListener("mousedown", (e) => {
      if ((e.target as Element).closest(".crumbline__item")) e.preventDefault();
    });
    this.dom.addEventListener("click", (e) => {
      const a = (e.target as Element).closest<HTMLAnchorElement>(".crumbline__item");
      if (!a?.dataset.id) return;
      e.preventDefault();
      ctx.engine.openPage(a.dataset.id, { newTab: e.ctrlKey || e.metaKey });
    });
    ctx.live.add(this);
    this.refresh();
  }

  refresh(): void {
    const pages = this.ctx.engine.pages;
    const chain: { id: string; title: string; icon: string; kind: string }[] = [];
    let cur = pages.get(this.ctx.pageId());
    for (let i = 0; cur && i < 64; i++) {
      chain.unshift({ id: cur.id, title: cur.title, icon: cur.icon, kind: cur.kind });
      cur = cur.parent_id ? pages.get(cur.parent_id) : undefined;
    }
    const sig = JSON.stringify(chain);
    if (sig === this.sig) return;
    this.sig = sig;
    reconcile(
      this.dom,
      chain,
      (c) => c.id,
      () => h("a", { class: "crumbline__item" }, h("span", { class: "crumbline__icon" }), h("span", { class: "crumbline__label" })),
      (a, c) => {
        a.dataset.id = c.id;
        setAttr(a, "href", `/p/${c.id}`);
        setAttr(a, "aria-current", c.id === this.ctx.pageId() ? "page" : null);
        paintIcon(a.firstElementChild as HTMLElement, c.icon, fallbackGlyph(c.kind));
        setText(a.lastElementChild as HTMLElement, c.title || t("Untitled"));
      },
    );
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    blockId(this.dom, node);
    return true;
  }

  stopEvent(e: Event): boolean {
    return !!(e.target as Element).closest?.(".crumbline__item");
  }

  ignoreMutation(): boolean {
    return true;
  }

  destroy(): void {
    this.ctx.live.delete(this);
  }
}

class TocView implements NodeView, Refreshable {
  dom: HTMLElement;
  private sig = "";

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly ctx: EditorCtx,
  ) {
    this.dom = h("nav", { class: "toc", attrs: { contenteditable: "false", "aria-label": t("Contents") } });
    blockId(this.dom, node);
    this.dom.addEventListener("mousedown", (e) => {
      if ((e.target as Element).closest(".toc__item")) e.preventDefault();
    });
    this.dom.addEventListener("click", (e) => {
      const a = (e.target as Element).closest<HTMLAnchorElement>(".toc__item");
      if (!a) return;
      e.preventDefault();
      const id = a.hash.slice(1);
      this.view.dom.querySelector(`[data-block="${CSS.escape(id)}"]`)?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
    ctx.live.add(this);
    this.refresh();
  }

  refresh(): void {
    const heads: { id: string; level: number; text: string }[] = [];
    this.view.state.doc.descendants((n) => {
      if (n.type.name === "heading") heads.push({ id: String(n.attrs.id ?? ""), level: Number(n.attrs.level) || 1, text: n.textContent });
      return !n.isTextblock;
    });
    const sig = JSON.stringify(heads);
    if (sig === this.sig) return;
    this.sig = sig;
    if (!heads.length) {
      this.dom.replaceChildren(h("span", { class: "toc__empty", text: t("Add headings to build the contents.") }));
      setAttr(this.dom, "data-empty", "true");
      return;
    }
    this.dom.querySelector(".toc__empty")?.remove();
    const existing = Array.from(this.dom.children) as HTMLAnchorElement[];
    heads.forEach((hd, i) => {
      let a = existing[i];
      if (!a) {
        a = h("a", { class: "toc__item" });
        this.dom.append(a);
      }
      setAttr(a, "href", `#${hd.id}`);
      setAttr(a, "data-level", String(hd.level));
      setText(a, hd.text || t("Untitled"));
    });
    for (const a of existing.slice(heads.length)) a.remove();
    setAttr(this.dom, "data-empty", heads.length ? null : "true");
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    blockId(this.dom, node);
    return true;
  }

  stopEvent(e: Event): boolean {
    return !!(e.target as Element).closest?.(".toc__item");
  }

  ignoreMutation(): boolean {
    return true;
  }

  destroy(): void {
    this.ctx.live.delete(this);
  }
}

class FileView implements NodeView {
  dom: HTMLElement;
  private icon: HTMLElement;
  private name: HTMLAnchorElement;
  private size: HTMLElement;
  private frame: HTMLIFrameElement | null = null;

  constructor(private node: PMNode) {
    this.icon = h("span", { class: "file__icon" }, iconNode("attachment"));
    this.name = h("a", { class: "file__name", attrs: { target: "_blank", rel: "noopener" } });
    this.size = h("span", { class: "file__size" });
    this.dom = h("div", { class: "file" }, h("div", { class: "file__card" }, this.icon, this.name, this.size));
    this.paint();
  }

  private paint(): void {
    blockId(this.dom, this.node);
    const src = String(this.node.attrs.src ?? "");
    const url = src.startsWith("f:") ? `/files/${src.slice(2)}` : src;
    const mime = String(this.node.attrs.mime ?? "");
    setAttr(this.dom, "data-mime", mime || null);
    setAttr(this.name, "href", url);
    setText(this.name, String(this.node.attrs.name || t("File")));
    setText(this.size, humanSize(Number(this.node.attrs.size) || 0));
    if (mime === "application/pdf") {
      if (!this.frame) {
        this.frame = h("iframe", { class: "file__pdf", attrs: { loading: "lazy" } });
        this.dom.append(this.frame);
      }
      setAttr(this.frame, "title", String(this.node.attrs.name ?? ""));
      if (this.frame.getAttribute("src") !== url) this.frame.src = url;
    } else if (this.frame) {
      this.frame.remove();
      this.frame = null;
    }
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    return e.type === "click" && !!(e.target as Element).closest?.(".file__name");
  }

  ignoreMutation(): boolean {
    return true;
  }
}

class MediaView implements NodeView {
  dom: HTMLElement;
  private frame: HTMLElement | null = null;
  private el: HTMLImageElement | HTMLVideoElement | HTMLAudioElement;
  private caption: HTMLElement;
  private pending: HTMLElement | null = null;
  private unwatch: (() => void) | null = null;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
    private readonly ctx: EditorCtx,
  ) {
    const kind = node.type.name;
    this.caption = h("figcaption", { class: "media__caption" });
    if (kind === "image") this.el = h("img", { attrs: { loading: "lazy", decoding: "async", draggable: "false" } });
    else if (kind === "video") this.el = h("video", { attrs: { controls: "controls", preload: "metadata" } });
    else this.el = h("audio", { attrs: { controls: "controls", preload: "metadata" } });
    if (kind === "audio") this.dom = h("figure", { class: "media media--audio" }, this.el, this.caption);
    else {
      this.frame = h("div", { class: "media__frame" }, this.el);
      this.dom = h("figure", { class: `media media--${kind}` }, this.frame, this.caption);
    }
    if (kind === "image") {
      this.el.addEventListener("click", () => {
        const pos = this.getPos();
        if (pos === undefined) return;
        const sel = this.view.state.selection;
        if (sel instanceof NodeSelection && sel.from === pos) openLightbox(String((this.el as HTMLImageElement).src), String(this.node.attrs.alt ?? ""));
      });
    }
    this.paint();
  }

  private remote(src: string): void {
    const id = String(this.node.attrs.id ?? "");
    if (!this.pending) {
      const text = h("span", { class: "media__pending-text" });
      const retry = h("button", { class: "btn btn--ghost btn--sm media__retry", attrs: { type: "button" }, text: t("Try again") });
      retry.addEventListener("click", () => importMedia(this.ctx, this.view, String(this.node.attrs.id ?? ""), true));
      this.pending = h("div", { class: "media__pending", attrs: { contenteditable: "false" } }, iconNode(this.node.type.name === "audio" ? "audio" : this.node.type.name === "video" ? "video" : "image"), text, retry);
      this.dom.insertBefore(this.pending, this.dom.firstChild);
    }
    if (!this.unwatch && id) this.unwatch = watchImport(id, () => this.remote(String(this.node.attrs.src ?? "")));
    let host = src.startsWith("data:") ? t("the clipboard") : src;
    try {
      if (!src.startsWith("data:")) host = new URL(src).host;
    } catch {
      host = src;
    }
    const phase = id ? importPhase(id) : null;
    setText(this.pending.children[1] as HTMLElement, phase === "failed" ? t("Could not bring in the file from {host}", { host }) : t("Bringing in the file from {host}", { host }));
    (this.pending.children[2] as HTMLElement).hidden = phase !== "failed" || !this.ctx.writable();
  }

  private paint(): void {
    blockId(this.dom, this.node);
    const src = String(this.node.attrs.src ?? "");
    const far = isRemote(src) || src.startsWith("data:");
    this.dom.classList.toggle("is-remote", far);
    if (far) {
      this.remote(src);
      setText(this.caption, String(this.node.attrs.caption ?? ""));
      return;
    }
    this.pending?.remove();
    this.pending = null;
    const url = src.startsWith("f:") ? `/files/${src.slice(2)}` : src;
    if (this.el.getAttribute("src") !== url) this.el.setAttribute("src", url);
    if (this.el instanceof HTMLImageElement) setAttr(this.el, "alt", String(this.node.attrs.alt ?? ""));
    if (!(this.el instanceof HTMLAudioElement)) {
      setAttr(this.el, "width", this.node.attrs.w ? String(this.node.attrs.w) : null);
      setAttr(this.el, "height", this.node.attrs.h ? String(this.node.attrs.h) : null);
    }
    setText(this.caption, String(this.node.attrs.caption ?? ""));
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    if (this.pending?.contains(e.target as Node) && (e.target as Element).closest?.("button")) return true;
    return this.node.type.name !== "image" && (e.target as Node) === this.el && e.type !== "dragstart";
  }

  destroy(): void {
    this.unwatch?.();
  }

  ignoreMutation(): boolean {
    return true;
  }
}

class BookmarkView implements NodeView {
  dom: HTMLElement;
  private link: HTMLAnchorElement;

  constructor(private node: PMNode) {
    this.link = h("a", { class: "bookmark__link", attrs: { target: "_blank", rel: "noopener noreferrer nofollow" } });
    this.dom = h("div", { class: "bookmark" }, this.link);
    this.paint();
  }

  private paint(): void {
    blockId(this.dom, this.node);
    const url = safeHref(String(this.node.attrs.url ?? ""));
    setAttr(this.link, "href", url);
    let host = url;
    try {
      host = new URL(url).host;
    } catch {
      host = url;
    }
    const title = String(this.node.attrs.title || host);
    const icon = String(this.node.attrs.icon ?? "");
    const line = h("span", { class: "bookmark__url" });
    if (icon.startsWith("f:")) line.append(h("img", { class: "bookmark__icon", attrs: { src: `/files/${icon.slice(2)}`, alt: "", width: "16", height: "16" } }));
    line.append(url);
    const text = h(
      "span",
      { class: "bookmark__text" },
      h("span", { class: "bookmark__title", text: title }),
      h("span", { class: "bookmark__desc", text: String(this.node.attrs.description ?? "") }),
      line,
    );
    const img = String(this.node.attrs.image ?? "");
    const kids: Node[] = [text];
    if (img.startsWith("f:")) kids.push(h("img", { class: "bookmark__image", attrs: { src: `/files/${img.slice(2)}`, alt: "", loading: "lazy" } }));
    this.link.replaceChildren(...kids);
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    const changed = ["url", "title", "description", "image", "icon"].some((k) => node.attrs[k] !== this.node.attrs[k]);
    this.node = node;
    if (changed) this.paint();
    else blockId(this.dom, node);
    return true;
  }

  stopEvent(e: Event): boolean {
    return e.type === "click" && (e as MouseEvent).button === 0 && !!(e.target as Element).closest?.("a");
  }

  ignoreMutation(): boolean {
    return true;
  }
}

class EmbedView implements NodeView {
  dom: HTMLElement;
  private frame: HTMLIFrameElement | null = null;
  private grip: HTMLElement;
  private fallback: HTMLAnchorElement | null = null;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
  ) {
    this.grip = h("div", { class: "embed__resize", attrs: { contenteditable: "false", "aria-hidden": "true" } });
    this.dom = h("figure", { class: "embed" }, this.grip);
    this.grip.addEventListener("pointerdown", (e) => this.resize(e));
    this.paint();
  }

  private resize(e: PointerEvent): void {
    const frame = this.frame;
    if (!frame || !this.view.editable || e.button !== 0) return;
    e.preventDefault();
    this.grip.setPointerCapture(e.pointerId);
    const top = frame.getBoundingClientRect().top;
    let height = Math.round(frame.getBoundingClientRect().height);
    this.dom.classList.add("is-resizing");
    const move = (ev: PointerEvent) => {
      height = Math.min(1600, Math.max(120, Math.round((ev.clientY - top) / 10) * 10));
      frame.height = String(height);
    };
    const up = () => {
      this.grip.removeEventListener("pointermove", move);
      this.grip.removeEventListener("pointerup", up);
      this.grip.removeEventListener("pointercancel", up);
      this.dom.classList.remove("is-resizing");
      const pos = this.getPos();
      if (pos === undefined || Number(this.node.attrs.height) === height) return;
      this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...this.node.attrs, height }));
    };
    this.grip.addEventListener("pointermove", move);
    this.grip.addEventListener("pointerup", up);
    this.grip.addEventListener("pointercancel", up);
  }

  private paint(): void {
    blockId(this.dom, this.node);
    const url = String(this.node.attrs.url ?? "");
    setAttr(this.dom, "data-provider", this.node.attrs.provider ? String(this.node.attrs.provider) : null);
    const { src, height } = embedSrc(url, String(this.node.attrs.provider ?? ""));
    if (!src) {
      this.frame?.remove();
      this.frame = null;
      if (!this.fallback) {
        this.fallback = h("a", { class: "embed__fallback", attrs: { target: "_blank", rel: "noopener noreferrer nofollow" } });
        this.dom.prepend(this.fallback);
      }
      setAttr(this.fallback, "href", safeHref(url));
      setText(this.fallback, url);
      return;
    }
    this.fallback?.remove();
    this.fallback = null;
    if (!this.frame) {
      this.frame = h("iframe", { attrs: { loading: "lazy", allow: EMBED_ALLOW, referrerpolicy: "strict-origin-when-cross-origin", sandbox: EMBED_SANDBOX } });
      this.dom.prepend(this.frame);
    }
    setAttr(this.frame, "title", url);
    const hh = Number(this.node.attrs.height) || height;
    setAttr(this.frame, "height", hh ? String(hh) : null);
    if (this.frame.getAttribute("src") !== src) this.frame.src = src;
  }

  stopEvent(e: Event): boolean {
    return this.grip.contains(e.target as Node);
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  ignoreMutation(): boolean {
    return true;
  }
}

class BlockTableView extends TableView {
  constructor(node: PMNode, cellMinWidth: number) {
    super(node, cellMinWidth);
    blockId(this.table, node);
  }

  override update(node: PMNode): boolean {
    const ok = super.update(node);
    if (ok) blockId(this.table, node);
    return ok;
  }
}

const extraViews = new Map<string, (ctx: EditorCtx) => NodeViewConstructor>();

export function registerNodeView(name: string, factory: (ctx: EditorCtx) => NodeViewConstructor): void {
  extraViews.set(name, factory);
}

export function nodeViews(ctx: EditorCtx): Record<string, NodeViewConstructor> {
  const extra: Record<string, NodeViewConstructor> = {};
  for (const [name, make] of extraViews) extra[name] = make(ctx);
  return {
    ...extra,
    code_block: (node, view, getPos) => new CodeView(node, view, getPos),
    task_item: (node, view, getPos) => new TaskView(node, view, getPos),
    toggle: (node) => new ToggleView(node, ctx),
    callout: (node, view, getPos) => new CalloutView(node, view, getPos, ctx),
    mention: (node) => new MentionView(node, ctx),
    toc: (node, view) => new TocView(node, view, ctx),
    breadcrumb: (node) => new CrumbView(node, ctx),
    column: (node, view, getPos) => new ColumnView(node, view, getPos),
    math_block: (node, view, getPos) => new MathView(node, view, getPos, ctx, true),
    math_inline: (node, view, getPos) => new MathView(node, view, getPos, ctx, false),
    file: (node) => new FileView(node),
    image: (node, view, getPos) => new MediaView(node, view, getPos, ctx),
    video: (node, view, getPos) => new MediaView(node, view, getPos, ctx),
    audio: (node, view, getPos) => new MediaView(node, view, getPos, ctx),
    bookmark: (node) => new BookmarkView(node),
    embed: (node, view, getPos) => new EmbedView(node, view, getPos),
    table: (node) => new BlockTableView(node, 80),
  };
}
