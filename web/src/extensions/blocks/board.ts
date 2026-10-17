import type { Node as PMNode } from "prosemirror-model";
import type { EditorView, NodeView } from "prosemirror-view";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import type { EditorCtx } from "./context";
import { registerNodeView } from "./nodeviews";
import { openPrompt } from "./popover";

const SVG = "http://www.w3.org/2000/svg";
const WIDTH = 1000;
const MIN_H = 200;
const MAX_H = 8000;
const MAX_ITEMS = 200;

export interface BoardItem {
  id: string;
  k: "image" | "note" | "link";
  x: number;
  y: number;
  w: number;
  h: number;
  src?: string;
  text?: string;
  href?: string;
}

export function newBoard(): PMNode {
  return schema.nodes.board!.create({ id: newBlockId(), items: "[]", height: 600 });
}

export function readBoard(raw: unknown): BoardItem[] {
  try {
    const v = JSON.parse(String(raw ?? "[]"));
    return Array.isArray(v) ? (v as BoardItem[]).filter((i) => i && typeof i.id === "string" && ["image", "note", "link"].includes(i.k) && (i.k !== "link" || /^https?:\/\//i.test(i.href ?? ""))) : [];
  } catch {
    return [];
  }
}

const clamp = (v: number, lo: number, hi: number): number => Math.min(hi, Math.max(lo, v));
const round = (v: number): number => Math.round(v);
const fileSrc = (ref: string): string => (ref.startsWith("f:") ? `/files/${ref.slice(2)}` : ref);
const hostOf = (href: string): string => {
  try {
    return new URL(href).host.replace(/^www\./, "");
  } catch {
    return href;
  }
};

function imageRatio(file: File): Promise<number> {
  return new Promise((resolve) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      resolve(img.naturalWidth > 0 ? img.naturalHeight / img.naturalWidth : 0.75);
      URL.revokeObjectURL(url);
    };
    img.onerror = () => {
      resolve(0.75);
      URL.revokeObjectURL(url);
    };
    img.src = url;
  });
}

interface Slot {
  fo: SVGForeignObjectElement;
  card: HTMLElement;
  key: string;
}

class BoardView implements NodeView {
  readonly dom: HTMLElement;
  private svg: SVGSVGElement;
  private bar: HTMLElement;
  private grip: HTMLElement;
  private slots = new Map<string, Slot>();
  private items: BoardItem[] = [];
  private height = 600;
  private moving = false;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: () => number | undefined,
    private readonly ctx: EditorCtx,
  ) {
    this.svg = document.createElementNS(SVG, "svg");
    this.svg.setAttribute("class", "board__canvas");
    this.svg.setAttribute("preserveAspectRatio", "xMinYMin meet");
    this.svg.setAttribute("role", "group");
    this.svg.setAttribute("aria-label", t("Board"));
    const tool = (icon: Parameters<typeof iconNode>[0], label: string, run: () => void): HTMLButtonElement => {
      const b = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": label, "data-tip": label } }, iconNode(icon));
      b.addEventListener("click", run);
      return b;
    };
    this.bar = h(
      "div",
      { class: "board__bar" },
      tool("text", t("Add note"), () => this.add({ k: "note", w: 220, h: 160, text: "" }, true)),
      tool("image", t("Add picture"), () => void this.pickPicture()),
      tool("link", t("Add link"), () => this.askLink()),
    );
    this.grip = h("div", { class: "board__resize", attrs: { "aria-hidden": "true" } });
    this.dom = h("div", { class: "board", attrs: { contenteditable: "false" } }, this.bar, this.svg as unknown as HTMLElement, this.grip);
    this.grip.addEventListener("pointerdown", (e) => this.resizeBoard(e));
    this.svg.addEventListener("dragstart", (e) => e.preventDefault());
    this.dom.addEventListener("dragover", (e) => {
      if (this.ctx.writable() && e.dataTransfer?.types.includes("Files")) {
        e.preventDefault();
        e.stopPropagation();
      }
    });
    this.dom.addEventListener("drop", (e) => this.drop(e));
    this.sync();
  }

  private sync(): void {
    this.items = readBoard(this.node.attrs.items);
    this.height = clamp(Number(this.node.attrs.height) || 600, MIN_H, MAX_H);
    setAttr(this.dom, "data-block", this.node.attrs.id ? String(this.node.attrs.id) : null);
    setAttr(this.dom, "data-empty", this.items.length ? null : "true");
    this.svg.setAttribute("viewBox", `0 0 ${WIDTH} ${this.height}`);
    const writable = this.ctx.writable();
    this.bar.hidden = !writable;
    this.grip.hidden = !writable;
    const seen = new Set<string>();
    let prev: ChildNode | null = null;
    for (const it of this.items) {
      seen.add(it.id);
      const key = `${it.k}|${it.src ?? ""}|${it.href ?? ""}|${it.k === "note" ? "" : (it.text ?? "")}|${writable}`;
      let slot = this.slots.get(it.id);
      if (!slot || slot.key !== key) {
        const fresh = this.build(it, writable);
        if (slot) slot.fo.replaceWith(fresh.fo);
        else this.svg.append(fresh.fo);
        slot = { ...fresh, key };
        this.slots.set(it.id, slot);
      } else if (it.k === "note") {
        const text = slot.card.querySelector<HTMLElement>(".board__text");
        if (text && document.activeElement !== text && text.textContent !== (it.text ?? "")) setText(text, it.text ?? "");
      }
      this.place(slot.fo, it);
      const want: ChildNode | null = prev ? prev.nextSibling : this.svg.firstChild;
      if (want !== slot.fo) this.svg.insertBefore(slot.fo, want);
      prev = slot.fo;
    }
    for (const [id, slot] of this.slots) {
      if (!seen.has(id)) {
        slot.fo.remove();
        this.slots.delete(id);
      }
    }
  }

  private place(fo: SVGForeignObjectElement, it: BoardItem): void {
    fo.setAttribute("x", String(it.x));
    fo.setAttribute("y", String(it.y));
    fo.setAttribute("width", String(it.w));
    fo.setAttribute("height", String(it.h));
  }

  private build(it: BoardItem, writable: boolean): { fo: SVGForeignObjectElement; card: HTMLElement } {
    const fo = document.createElementNS(SVG, "foreignObject");
    fo.setAttribute("class", "board__slot");
    let card: HTMLElement;
    if (it.k === "image") {
      card = h("div", { class: "board__card board__card--image" }, h("img", { attrs: { src: fileSrc(it.src ?? ""), alt: "", decoding: "async", draggable: "false" } }));
    } else if (it.k === "note") {
      const text = h("div", { class: "board__text", attrs: { "aria-label": t("Note") } });
      setText(text, it.text ?? "");
      if (writable) {
        text.contentEditable = "plaintext-only";
        setAttr(text, "data-placeholder", t("Write a note"));
        text.addEventListener("blur", () => this.patch(it.id, { text: (text.textContent ?? "").slice(0, 4000) }));
        text.addEventListener("keydown", (e) => {
          if (e.key === "Escape") text.blur();
        });
      }
      card = h("div", { class: "board__card board__card--note" }, text);
    } else {
      card = h(
        "a",
        { class: "board__card board__card--link", attrs: { href: it.href ?? "", target: "_blank", rel: "noopener noreferrer nofollow", draggable: "false" } },
        h("span", { class: "board__host", text: hostOf(it.href ?? "") }),
        h("span", { class: "board__text", text: it.text || it.href || "" }),
      );
    }
    if (writable) {
      const remove = h("button", { class: "icon-btn icon-btn--sm board__remove", attrs: { type: "button", "aria-label": t("Remove"), "data-tip": t("Remove") } }, iconNode("close"));
      remove.addEventListener("click", (e) => {
        e.preventDefault();
        this.commit(this.items.filter((i) => i.id !== it.id));
      });
      const size = h("div", { class: "board__size", attrs: { "aria-hidden": "true" } });
      size.addEventListener("pointerdown", (e) => this.drag(e, it.id, "size"));
      card.addEventListener("pointerdown", (e) => {
        const target = e.target as Element;
        if (target.closest(".board__remove, .board__size") || (target.closest(".board__text") && it.k === "note" && card.contains(document.activeElement))) return;
        this.drag(e, it.id, "move");
      });
      card.append(remove, size);
    }
    fo.append(card);
    return { fo, card };
  }

  private scale(): number {
    const w = this.svg.getBoundingClientRect().width;
    return w > 0 ? WIDTH / w : 1;
  }

  private drag(e: PointerEvent, id: string, mode: "move" | "size"): void {
    const it = this.items.find((i) => i.id === id);
    const slot = this.slots.get(id);
    if (!it || !slot || e.button !== 0 || !this.ctx.writable()) return;
    const target = e.currentTarget as HTMLElement;
    e.preventDefault();
    target.setPointerCapture(e.pointerId);
    const k = this.scale();
    const sx = e.clientX;
    const sy = e.clientY;
    const next = { ...it };
    let moved = false;
    this.moving = true;
    slot.fo.classList.add("is-moving");
    const move = (ev: PointerEvent) => {
      const dx = (ev.clientX - sx) * k;
      const dy = (ev.clientY - sy) * k;
      if (!moved && Math.hypot(dx, dy) < 4 * k) return;
      moved = true;
      if (mode === "move") {
        next.x = round(clamp(it.x + dx, 0, WIDTH - it.w));
        next.y = round(clamp(it.y + dy, 0, this.height - it.h));
      } else {
        next.w = round(clamp(it.w + dx, 40, WIDTH - it.x));
        next.h = round(clamp(it.h + dy, 40, this.height - it.y));
      }
      this.place(slot.fo, next);
    };
    const up = () => {
      target.removeEventListener("pointermove", move);
      target.removeEventListener("pointerup", up);
      target.removeEventListener("pointercancel", up);
      slot.fo.classList.remove("is-moving");
      this.moving = false;
      if (!moved) {
        if (it.k === "note" && mode === "move") slot.card.querySelector<HTMLElement>(".board__text")?.focus();        return;
      }
      const rest = this.items.filter((i) => i.id !== id);
      this.commit([...rest, next]);
    };
    target.addEventListener("pointermove", move);
    target.addEventListener("pointerup", up);
    target.addEventListener("pointercancel", up);
    target.addEventListener("click", (ev) => moved && ev.preventDefault(), { once: true, capture: true });
  }

  private resizeBoard(e: PointerEvent): void {
    if (e.button !== 0 || !this.ctx.writable()) return;
    e.preventDefault();
    this.grip.setPointerCapture(e.pointerId);
    const k = this.scale();
    const sy = e.clientY;
    const start = this.height;
    const floor = Math.max(MIN_H, ...this.items.map((i) => i.y + i.h));
    let height = start;
    const move = (ev: PointerEvent) => {
      height = round(clamp(start + (ev.clientY - sy) * k, floor, MAX_H) / 10) * 10;
      this.svg.setAttribute("viewBox", `0 0 ${WIDTH} ${height}`);
    };
    const up = () => {
      this.grip.removeEventListener("pointermove", move);
      this.grip.removeEventListener("pointerup", up);
      this.grip.removeEventListener("pointercancel", up);
      if (height !== start) this.commit(this.items, height);
    };
    this.grip.addEventListener("pointermove", move);
    this.grip.addEventListener("pointerup", up);
    this.grip.addEventListener("pointercancel", up);
  }

  private slotFor(w: number, h: number): { x: number; y: number; height: number } {
    const gap = 24;
    for (let y = gap; y + h <= this.height - gap; y += 40) {
      for (let x = gap; x + w <= WIDTH - gap; x += 40) {
        const hit = this.items.some((i) => x < i.x + i.w + gap / 2 && x + w + gap / 2 > i.x && y < i.y + i.h + gap / 2 && y + h + gap / 2 > i.y);
        if (!hit) return { x, y, height: this.height };
      }
    }
    const y = Math.max(gap, ...this.items.map((i) => i.y + i.h + gap));
    return { x: gap, y, height: clamp(y + h + gap, MIN_H, MAX_H) };
  }

  private add(part: Omit<BoardItem, "id" | "x" | "y">, focus = false): void {
    if (this.items.length >= MAX_ITEMS) return;
    const at = this.slotFor(part.w, part.h);
    if (at.y + part.h > MAX_H) return;
    const it: BoardItem = { ...part, id: newBlockId(), x: at.x, y: at.y };
    this.commit([...this.items, it], at.height);
    if (focus) requestAnimationFrame(() => this.slots.get(it.id)?.card.querySelector<HTMLElement>(".board__text")?.focus());
  }

  private patch(id: string, part: Partial<BoardItem>): void {
    const it = this.items.find((i) => i.id === id);
    if (!it || Object.entries(part).every(([k, v]) => it[k as keyof BoardItem] === v)) return;
    this.commit(this.items.map((i) => (i.id === id ? { ...i, ...part } : i)));
  }

  private async pickPicture(): Promise<void> {
    const { pickFile } = await import("../../ui/pickers");
    const file = await pickFile("image/png,image/jpeg,image/gif,image/webp");
    if (file) await this.uploadPicture(file);
  }

  private async uploadPicture(file: File): Promise<void> {
    const pageId = this.ctx.pageId();
    if (!pageId || !file.type.startsWith("image/") || file.type === "image/svg+xml") return;
    try {
      const [ratio, r] = await Promise.all([imageRatio(file), this.ctx.engine.api.files.upload(file, "attachment", { page_id: pageId, name: file.name })]);
      if (this.view.isDestroyed) return;
      const w = 280;
      this.add({ k: "image", w, h: round(clamp(w * ratio, 60, 900)), src: `f:${r.file.id}` });
    } catch (e) {
      this.ctx.engine.reportError(e);
    }
  }

  private drop(e: DragEvent): void {
    const files = [...(e.dataTransfer?.files ?? [])].filter((f) => f.type.startsWith("image/"));
    if (!files.length || !this.ctx.writable()) return;
    e.preventDefault();
    e.stopPropagation();
    void files.reduce((p, f) => p.then(() => this.uploadPicture(f)), Promise.resolve());
  }

  private askLink(): void {
    openPrompt({
      label: t("Add link"),
      placeholder: t("Paste a link"),
      action: t("Add"),
      rect: () => this.bar.getBoundingClientRect(),
      onSubmit: (raw) => {
        let href = raw.trim();
        if (!href) return;
        if (!/^https?:\/\//i.test(href)) href = `https://${href}`;
        try {
          href = new URL(href).toString();
        } catch {
          return;
        }
        this.add({ k: "link", w: 260, h: 96, href });
      },
    });
  }

  private commit(items: BoardItem[], height = this.height): void {
    const pos = this.getPos();
    if (pos === undefined) return;
    this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...this.node.attrs, items: JSON.stringify(items), height }));
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    if (!this.moving) this.sync();
    return true;
  }

  selectNode(): void {
    this.dom.classList.add("is-selected");
  }

  deselectNode(): void {
    this.dom.classList.remove("is-selected");
  }

  stopEvent(e: Event): boolean {
    return e.target instanceof Element && !!e.target.closest(".board__canvas, .board__bar, .board__resize");
  }

  ignoreMutation(): boolean {
    return true;
  }
}

registerNodeView("board", (ctx) => (node, view, getPos) => new BoardView(node, view, getPos, ctx));
