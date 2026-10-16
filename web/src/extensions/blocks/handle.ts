import { Fragment, type NodeType } from "prosemirror-model";
import { NodeSelection, TextSelection } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";
import { newBlockId, schema, TONES } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { h, setAttr } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { toast } from "../../ui/toast";
import { CATALOG } from "./catalog";
import { addColumn, removeLastColumn, resetWidths } from "./columns";
import { commentUI } from "./comments";
import { type BlockRef, blockAt, blockFromPos, deleteBlock, duplicateBlock, moveBlock, tableAddColumn, tableAddRow, tableRemoveColumn, tableRemoveRow, turnInto } from "./commands";
import { blockRect, type EditorCtx } from "./context";
import { openPrompt } from "./popover";
import { createSynced } from "./synced";
import { triggerKey } from "./triggers";

const N = schema.nodes as Record<string, NodeType>;
const ITEMS = new Set(["list_item", "task_item"]);

export class BlockHandle {
  readonly el: HTMLElement;
  private add: HTMLButtonElement;
  private grip: HTMLButtonElement;
  private target: BlockRef | null = null;
  private hideTimer: ReturnType<typeof setTimeout> | undefined;
  private frame = 0;
  private lastY = 0;
  private lastX = 0;
  private phone: HTMLElement;

  constructor(
    private readonly ctx: EditorCtx,
    private readonly wrap: HTMLElement,
  ) {
    this.add = h("button", { class: "block-handle__btn", attrs: { type: "button", "aria-label": t("Add a block below"), "data-tip": t("Add a block below") } }, iconNode("plus"));
    this.grip = h("button", { class: "block-handle__btn block-handle__grip", attrs: { type: "button", draggable: "true", "aria-label": t("Drag to move, click for options"), "data-tip": t("Drag to move, click for options") } }, iconNode("grip"));
    this.el = h("div", { class: "block-handle", attrs: { contenteditable: "false" } }, this.add, this.grip);
    wrap.prepend(this.el);
    wrap.addEventListener("pointermove", (e) => {
      if (e.pointerType === "touch" || this.el.contains(e.target as Node)) return;
      if ((e.target as Element).closest?.(".column__resize")) return this.hide();
      this.lastY = e.clientY;
      this.lastX = e.clientX;
      if (!this.frame) this.frame = requestAnimationFrame(() => this.track());
    });
    wrap.addEventListener("pointerleave", () => this.hideSoon());
    this.el.addEventListener("pointerenter", () => clearTimeout(this.hideTimer));
    this.add.addEventListener("mousedown", (e) => e.preventDefault());
    this.add.addEventListener("click", () => this.addBelow());
    this.grip.addEventListener("click", () => this.menu());
    this.grip.addEventListener("dragstart", (e) => this.dragStart(e));
    this.grip.addEventListener("dragend", () => this.hide());
    const tool = (icon: Parameters<typeof iconNode>[0], label: string, run: (b: HTMLButtonElement) => void): HTMLButtonElement => {
      const b = h("button", { class: "icon-btn block-phonebar__btn", attrs: { type: "button", "aria-label": label } }, iconNode(icon));
      b.addEventListener("pointerdown", (e) => e.preventDefault());
      b.addEventListener("click", () => {
        if (this.caretBlock()) run(b);
      });
      return b;
    };
    this.phone = h(
      "div",
      { class: "block-phonebar", attrs: { role: "toolbar", "aria-label": t("Block"), contenteditable: "false" } },
      tool("plus", t("Add a block below"), () => this.addBelow()),
      tool("grip", t("Block options"), (b) => this.menu(b)),
      tool("arrow-up", t("Move up"), () => this.moveCaretBlock(-1)),
      tool("arrow-down", t("Move down"), () => this.moveCaretBlock(1)),
    );
    wrap.append(this.phone);
    const vv = window.visualViewport;
    if (vv) {
      const keyboard = () => document.documentElement.style.setProperty("--keyboard-h", `${Math.max(0, Math.round(window.innerHeight - vv.height - vv.offsetTop))}px`);
      vv.addEventListener("resize", keyboard);
      vv.addEventListener("scroll", keyboard);
    }
  }

  private caretBlock(): BlockRef | null {
    const view = this.ctx.view();
    if (!view || !view.editable) return null;
    let ref = blockAt(view.state.selection.$head);
    if (!ref) return null;
    const $ref = view.state.doc.resolve(ref.pos);
    if (ITEMS.has($ref.parent.type.name) && $ref.index() === 0) ref = { node: $ref.parent, pos: $ref.before(), depth: $ref.depth };
    if ($ref.parent.type.name === "toggle" && ref.node.type.name === "toggle_title") ref = { node: $ref.parent, pos: $ref.before(), depth: $ref.depth };
    this.target = ref;
    return ref;
  }

  private moveCaretBlock(dir: -1 | 1): void {
    const view = this.ctx.view();
    if (view) moveBlock(view, dir);
  }

  hide(): void {
    clearTimeout(this.hideTimer);
    this.el.classList.remove("is-visible");
    this.target = null;
  }

  private hideSoon(): void {
    clearTimeout(this.hideTimer);
    this.hideTimer = setTimeout(() => this.hide(), 250);
  }

  private track(): void {
    this.frame = 0;
    const view = this.ctx.view();
    if (!view || !view.editable || !view.dom.isConnected) return this.hide();
    const under = document.elementFromPoint(this.lastX, this.lastY)?.closest(".column");
    const box = under instanceof HTMLElement && view.dom.contains(under) ? under.getBoundingClientRect() : view.dom.getBoundingClientRect();
    const hit = view.posAtCoords({ left: box.left + Math.min(48, box.width / 2), top: this.lastY });
    if (!hit) return this.hideSoon();
    const $pos = view.state.doc.resolve(hit.inside >= 0 ? hit.inside : hit.pos);
    let ref = blockAt(hit.inside >= 0 ? view.state.doc.resolve(Math.min(hit.inside + 1, view.state.doc.content.size)) : $pos);
    if (!ref) return this.hideSoon();
    const $ref = view.state.doc.resolve(ref.pos);
    if (ITEMS.has($ref.parent.type.name) && $ref.index() === 0) ref = { node: $ref.parent, pos: $ref.before(), depth: $ref.depth };
    if ($ref.parent.type.name === "toggle" && ref.node.type.name === "toggle_title") ref = { node: $ref.parent, pos: $ref.before(), depth: $ref.depth };
    const dom = view.nodeDOM(ref.pos);
    if (!(dom instanceof HTMLElement)) return this.hideSoon();
    clearTimeout(this.hideTimer);
    this.target = ref;
    const r = dom.getBoundingClientRect();
    const w = this.wrap.getBoundingClientRect();
    const line = Math.min(r.height, ref.node.type.name === "heading" ? 40 : 28);
    const x = r.left - w.left - this.el.offsetWidth - 6;
    const y = r.top - w.top + line / 2 - this.el.offsetHeight / 2;
    this.el.style.transform = `translate(${Math.round(x)}px, ${Math.round(y)}px)`;
    this.el.classList.add("is-visible");
  }

  private addBelow(): void {
    const view = this.ctx.view();
    const b = this.target;
    if (!view || !b) return;
    const end = b.pos + b.node.nodeSize;
    const tr = view.state.tr.insert(end, (N.paragraph as NodeType).create({ id: newBlockId() }));
    tr.setSelection(TextSelection.create(tr.doc, end + 1));
    view.dispatch(tr);
    view.focus();
    view.dispatch(view.state.tr.insertText("/").setMeta(triggerKey, { active: true, kind: "slash", from: end + 1, query: "" }));
  }

  private dragStart(e: DragEvent): void {
    const view = this.ctx.view();
    const b = this.target;
    if (!view || !b || !e.dataTransfer) return;
    const sel = NodeSelection.create(view.state.doc, b.pos);
    view.dispatch(view.state.tr.setSelection(sel));
    const slice = sel.content();
    const { dom, text } = view.serializeForClipboard(slice);
    e.dataTransfer.clearData();
    e.dataTransfer.setData("text/html", dom.innerHTML);
    e.dataTransfer.setData("text/plain", text);
    e.dataTransfer.effectAllowed = "copyMove";
    const node = view.nodeDOM(b.pos);
    if (node instanceof HTMLElement) e.dataTransfer.setDragImage(node, 0, 0);
    view.dragging = { slice, move: true };
  }

  private menu(anchor: HTMLElement = this.grip): void {
    const view = this.ctx.view();
    const b = this.target;
    if (!view || !b) return;
    const ref = blockFromPos(view.state, b.pos) ?? b;
    view.dispatch(view.state.tr.setSelection(NodeSelection.create(view.state.doc, ref.pos)));
    const inner = ref.node.isTextblock ? ref.pos + 1 : ref.pos + 2;
    const focusInner = () => {
      const max = view.state.doc.content.size;
      view.dispatch(view.state.tr.setSelection(TextSelection.near(view.state.doc.resolve(Math.min(inner, max)))));
    };
    const entries: MenuEntry[] = [{ kind: "heading", label: t("Turn into") }];
    for (const c of CATALOG.filter((x) => x.turn)) {
      entries.push({
        label: c.label(),
        icon: c.icon,
        run: () => {
          focusInner();
          turnInto(view, c.turn as Parameters<typeof turnInto>[1]);
          view.focus();
        },
      });
    }
    if (ref.node.type.name === "callout") {
      entries.push({ kind: "heading", label: t("Colour") });
      for (const tone of TONES) {
        entries.push({
          label: tone ? tone.charAt(0).toUpperCase() + tone.slice(1) : t("Default"),
          lead: h("span", { class: "tag-dot", attrs: { "data-color": tone || "none" } }),
          checked: String(ref.node.attrs.tone ?? "") === tone,
          run: () => view.dispatch(view.state.tr.setNodeMarkup(ref.pos, undefined, { ...ref.node.attrs, tone })),
        });
      }
    }
    if (ref.node.type.name === "table") {
      entries.push(
        { kind: "heading", label: t("Table") },
        { label: t("Add a row"), icon: "plus", run: () => tableAddRow(view, ref) },
        { label: t("Add a column"), icon: "plus", run: () => tableAddColumn(view, ref) },
        { label: t("Remove the last row"), icon: "close", run: () => tableRemoveRow(view, ref) },
        { label: t("Remove the last column"), icon: "close", run: () => tableRemoveColumn(view, ref) },
      );
    }
    let rowPos = ref.node.type.name === "columns" ? ref.pos : -1;
    const $ref = view.state.doc.resolve(ref.pos);
    for (let d = $ref.depth; d > 0 && rowPos < 0; d--) if ($ref.node(d).type.name === "columns") rowPos = $ref.before(d);
    if (rowPos >= 0) {
      entries.push(
        { kind: "heading", label: t("Columns") },
        { label: t("Add a column"), icon: "plus", run: () => addColumn(view, rowPos) },
        { label: t("Remove the last column"), icon: "close", run: () => removeLastColumn(view, rowPos) },
        { label: t("Make columns equal"), icon: "columns", run: () => view.dispatch(resetWidths(view.state.tr, rowPos)) },
      );
    }
    if (ref.node.type.name === "image" || ref.node.type.name === "video" || ref.node.type.name === "audio") {
      entries.push({ kind: "separator" }, { label: t("Caption"), icon: "align-left", run: () => this.caption(view, ref) });
    }
    entries.push(
      { kind: "separator" },
      { label: t("Move up"), icon: "arrow-up", hint: "Alt+Up", run: () => moveBlock(view, -1) },
      { label: t("Move down"), icon: "arrow-down", hint: "Alt+Down", run: () => moveBlock(view, 1) },
      { label: t("Duplicate"), icon: "copy", run: () => duplicateBlock(view, ref) },
      ...(ref.node.type.name === "synced" || ITEMS.has(ref.node.type.name) || !view.editable
        ? []
        : [{ label: t("Turn into a synced block"), icon: "refresh" as const, run: () => this.makeSynced(view, ref) }]),
      { label: t("Copy link to block"), icon: "link", run: () => this.copyLink(ref) },
      { label: t("Comment"), icon: "comment-plus", run:() => commentUI(view)?.startThread(ref.pos, ref.pos + ref.node.nodeSize, "block") },
      { kind: "separator" },
      { label: t("Delete"), icon: "trash", danger: true, run: () => deleteBlock(view, ref) },
    );
    openMenu(anchor, entries, { side: anchor === this.grip ? "bottom" : "top", align: "start", label: t("Block") });
  }

  private makeSynced(view: EditorView, ref: BlockRef): void {
    const cur = view.state.doc.nodeAt(ref.pos);
    if (!cur || cur !== ref.node) return;
    const node = createSynced(this.ctx, Fragment.from(cur));
    if (!node) return;
    try {
      view.dispatch(view.state.tr.replaceWith(ref.pos, ref.pos + cur.nodeSize, node));
    } catch {
      toast({ id: "synced-fail", tone: "danger", message: t("This block cannot become a synced block here.") });
    }
  }

  private caption(view: EditorView, ref: BlockRef): void {
    openPrompt({
      label: t("Caption"),
      placeholder: t("Write a caption"),
      value: String(ref.node.attrs.caption ?? ""),
      action: t("Save"),
      allowEmpty: true,
      rect: () => blockRect(view, ref.pos),
      onSubmit: (value) => {
        const node = view.state.doc.nodeAt(ref.pos);
        if (!node || node.type !== ref.node.type) return;
        view.dispatch(view.state.tr.setNodeMarkup(ref.pos, undefined, { ...node.attrs, caption: value }));
      },
      onClose: () => view.focus(),
    });
  }

  private copyLink(ref: BlockRef): void {
    const id = String(ref.node.attrs.id ?? "");
    const url = `${location.origin}/p/${this.ctx.pageId()}${id ? `#${id}` : ""}`;
    void navigator.clipboard
      ?.writeText(url)
      .then(() => toast({ id: "block-link", tone: "success", message: t("Link copied") }))
      .catch(() => undefined);
  }

  setVisible(on: boolean): void {
    setAttr(this.el, "hidden", on ? null : "");
  }
}
