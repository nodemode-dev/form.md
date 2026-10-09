import { type EditorState, Plugin, PluginKey, type Transaction } from "prosemirror-state";
import { Decoration, DecorationSet, type EditorView } from "prosemirror-view";
import { absolutePositionToRelativePosition, relativePositionToAbsolutePosition, ySyncPluginKey } from "y-prosemirror";
import * as Y from "yjs";
import type { CommentThread } from "../../engine";
import { t } from "../../engine/i18n";
import { composer, threadView, type ThreadView } from "../../ui/comment-view";
import { byId, h } from "../../ui/dom";
import { anchorToRect, type Rect } from "../../ui/floating";
import { iconNode } from "../../ui/icons";
import { openLayer } from "../../ui/layers";
import type { EditorCtx } from "./context";

interface Binding {
  doc: Y.Doc;
  type: Y.XmlFragment;
  mapping: Map<unknown, unknown>;
}

interface CommentState {
  set: DecorationSet;
  pending: { from: number; to: number } | null;
  active: string;
}

type Meta = { rebuild?: true; pending?: { from: number; to: number } | null; active?: string };

export const commentKey = new PluginKey<CommentState>("formmd-comments");

const b64 = (u: Uint8Array) => {
  let s = "";
  for (const c of u) s += String.fromCharCode(c);
  return btoa(s);
};
const unb64 = (s: string) => Uint8Array.from(atob(s), (c) => c.charCodeAt(0));

function binding(state: EditorState): Binding | null {
  const st = ySyncPluginKey.getState(state) as { binding?: Binding } | undefined;
  return st?.binding?.type ? st.binding : null;
}

export function encodeRange(state: EditorState, from: number, to: number): { start: string; end: string } | null {
  const b = binding(state);
  if (!b) return null;
  try {
    const s = absolutePositionToRelativePosition(from, b.type, b.mapping as never);
    const e = absolutePositionToRelativePosition(to, b.type, b.mapping as never);
    return { start: b64(Y.encodeRelativePosition(s)), end: b64(Y.encodeRelativePosition(e)) };
  } catch {
    return null;
  }
}

function resolve(b: Binding, enc: string): number | null {
  try {
    return relativePositionToAbsolutePosition(b.doc, b.type, Y.decodeRelativePosition(unb64(enc)), b.mapping as never);
  } catch {
    return null;
  }
}

export function threadRange(state: EditorState, th: CommentThread): { from: number; to: number } | null {
  const b = binding(state);
  if (!b || th.anchor === "page") return null;
  const a = resolve(b, th.start);
  const z = resolve(b, th.end);
  if (a === null || z === null) return null;
  const size = state.doc.content.size;
  const from = Math.max(0, Math.min(a, z, size));
  const to = Math.min(size, Math.max(a, z));
  return to > from ? { from, to } : null;
}

function build(state: EditorState, threads: CommentThread[], active: string, pending: CommentState["pending"]): DecorationSet {
  const decos: Decoration[] = [];
  for (const th of threads) {
    if (th.anchor === "page" || th.resolved_at) continue;
    const r = threadRange(state, th);
    if (!r) continue;
    const cls = th.id === active ? "c-mark is-active" : "c-mark";
    if (th.anchor === "block") {
      const node = state.doc.nodeAt(r.from);
      if (node?.isBlock) decos.push(Decoration.node(r.from, r.from + node.nodeSize, { class: th.id === active ? "c-block is-active" : "c-block", "data-thread": th.id }, { thread: th.id }));
      continue;
    }
    decos.push(Decoration.inline(r.from, r.to, { class: cls, "data-thread": th.id }, { thread: th.id, inclusiveStart: false, inclusiveEnd: false }));
  }
  if (pending && pending.to > pending.from && pending.to <= state.doc.content.size) decos.push(Decoration.inline(pending.from, pending.to, { class: "c-mark is-pending" }));
  return DecorationSet.create(state.doc, decos);
}

function threadsAt(state: EditorState, pos: number): string[] {
  const st = commentKey.getState(state);
  if (!st) return [];
  return st.set
    .find(Math.max(0, pos - 1), pos + 1)
    .map((d) => (d.spec as { thread?: string }).thread ?? "")
    .filter(Boolean);
}

function rangeRect(view: EditorView, from: number, to: number): Rect | null {
  try {
    const a = view.coordsAtPos(from);
    const b = view.coordsAtPos(to);
    if (a.top !== b.top) {
      const box = view.dom.getBoundingClientRect();
      return { x: box.left, y: a.top, width: box.width, height: b.bottom - a.top };
    }
    return { x: a.left, y: a.top, width: Math.max(1, b.right - a.left), height: b.bottom - a.top };
  } catch {
    return null;
  }
}

class CommentUI {
  private pop: { el: HTMLElement; close: () => void; thread: string; view: ThreadView | null } | null = null;
  private bubble: { el: HTMLElement; update: () => void; release: () => void } | null = null;
  private off: () => void;
  private page = "";

  constructor(
    private readonly view: EditorView,
    private readonly ctx: EditorCtx,
  ) {
    this.off = ctx.engine.comments.subscribe((id) => {
      if (id !== this.ctx.pageId()) return;
      this.refresh();
    });
    document.addEventListener("selectionchange", this.onSelection);
    this.view.dom.addEventListener("mousedown", this.onDown);
  }

  private threads(): CommentThread[] {
    return this.ctx.engine.comments.get(this.ctx.pageId()).threads;
  }

  refresh(): void {
    if (this.view.isDestroyed) return;
    this.view.dispatch(this.view.state.tr.setMeta(commentKey, { rebuild: true } satisfies Meta));
    if (this.pop?.view) {
      const th = this.threads().find((x) => x.id === this.pop?.thread);
      if (th) this.pop.view.update(th, this.ctx.engine.comments.get(this.ctx.pageId()));
      else this.closePop();
    }
  }

  update(): void {
    const id = this.ctx.pageId();
    if (id !== this.page) {
      this.page = id;
      this.closePop();
      this.hideBubble();
    }
  }

  private canComment(): boolean {
    return this.ctx.engine.comments.get(this.ctx.pageId()).can_comment;
  }

  private onDown = () => this.hideBubble();

  private onSelection = () => {
    if (this.view.editable || !this.canComment()) return this.hideBubble();
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || !sel.anchorNode || !sel.focusNode || !this.view.dom.contains(sel.anchorNode) || !this.view.dom.contains(sel.focusNode)) return this.hideBubble();
    let a: number;
    let b: number;
    try {
      a = this.view.posAtDOM(sel.anchorNode, sel.anchorOffset);
      b = this.view.posAtDOM(sel.focusNode, sel.focusOffset);
    } catch {
      return this.hideBubble();
    }
    const from = Math.min(a, b);
    const to = Math.max(a, b);
    if (!this.view.state.doc.textBetween(from, to, " ", " ").trim()) return this.hideBubble();
    this.showBubble(from, to);
  };

  private showBubble(from: number, to: number): void {
    if (this.bubble) {
      this.bubble.el.dataset.range = JSON.stringify({ from, to });
      this.bubble.update();
      return;
    }
    const btn = h("button", { class: "formatbar__btn formatbar__btn--text", attrs: { type: "button" } }, iconNode("comment-plus"), h("span", { text: t("Comment") }));
    const el = h("div", { class: "formatbar is-visible", attrs: { role: "toolbar", "aria-label": t("Comment") } }, btn);
    btn.addEventListener("mousedown", (e) => e.preventDefault());
    btn.addEventListener("click", () => {
      const r = this.bubble ? (JSON.parse(el.dataset.range ?? "null") as { from: number; to: number } | null) : null;
      this.hideBubble();
      if (r) this.startThread(r.from, r.to, "text");
    });
    el.dataset.range = JSON.stringify({ from, to });
    byId("overlay-root").append(el);
    const place = anchorToRect(el, () => {
      const r = JSON.parse(el.dataset.range ?? "null") as { from: number; to: number } | null;
      return r ? rangeRect(this.view, r.from, r.to) : null;
    }, { side: "top", align: "center", offset: 8 });
    this.bubble = { el, update: place.update, release: place.release };
  }

  private hideBubble(): void {
    this.bubble?.release();
    this.bubble?.el.remove();
    this.bubble = null;
  }

  startThread(from: number, to: number, anchor: "text" | "block"): void {
    const state = this.view.state;
    const range = encodeRange(state, from, to);
    if (!range) return;
    const quote = state.doc.textBetween(from, to, " ", " ").trim().slice(0, 500);
    this.closePop();
    this.view.dispatch(state.tr.setMeta(commentKey, { pending: { from, to } } satisfies Meta));
    const pageId = this.ctx.pageId();
    const c = composer(this.ctx.engine, {
      workspaceId: () => this.ctx.workspaceId(),
      placeholder: t("Add a comment"),
      submitLabel: t("Comment"),
      onSubmit: async (body) => {
        const r = await this.ctx.engine.comments.run(pageId, () => this.ctx.engine.api.comments.create(pageId, { anchor, start: range.start, end: range.end, quote, body }));
        this.closePop();
        this.focusThread(r.thread.id);
      },
      onCancel: () => this.closePop(),
    });
    const el = h("div", { class: "popover cpop" }, h("div", { class: "cpop__quote", text: quote || t("A block") }), c.el);
    this.openPop(el, "", null, () => rangeRect(this.view, from, to));
    c.focus();
  }

  focusThread(id: string): void {
    const th = this.threads().find((x) => x.id === id);
    if (!th) return;
    const r = threadRange(this.view.state, th);
    this.view.dispatch(this.view.state.tr.setMeta(commentKey, { active: id } satisfies Meta));
    if (!r) return;
    const dom = this.view.domAtPos(r.from).node;
    (dom instanceof Element ? dom : dom.parentElement)?.scrollIntoView({ block: "center", behavior: "smooth" });
    this.openThread(th, r);
  }

  private openThread(th: CommentThread, r: { from: number; to: number }): void {
    const pageId = this.ctx.pageId();
    const v = threadView(this.ctx.engine, pageId, th, { workspaceId: () => this.ctx.workspaceId(), showQuote: false });
    const el = h("div", { class: "popover cpop" }, v.el);
    this.openPop(el, th.id, v, () => rangeRect(this.view, r.from, r.to));
  }

  private openPop(el: HTMLElement, thread: string, view: ThreadView | null, rect: () => Rect | null): void {
    byId("overlay-root").append(el);
    const place = anchorToRect(el, rect, { side: "bottom", align: "start", offset: 6 });
    requestAnimationFrame(() => el.classList.add("is-visible"));
    let closed = false;
    const close = openLayer({
      el,
      modal: false,
      dismissOnOutside: true,
      returnFocus: null,
      onClose: () => {
        if (closed) return;
        closed = true;
        place.release();
        el.remove();
        if (this.pop?.el === el) this.pop = null;
        if (!this.view.isDestroyed) this.view.dispatch(this.view.state.tr.setMeta(commentKey, { pending: null, active: "" } satisfies Meta));
      },
    });
    this.pop = { el, close, thread, view };
  }

  closePop(): void {
    this.pop?.close();
    this.pop = null;
  }

  click(pos: number): boolean {
    const ids = threadsAt(this.view.state, pos);
    const id = ids[0];
    if (!id) return false;
    const th = this.threads().find((x) => x.id === id);
    const r = th ? threadRange(this.view.state, th) : null;
    if (!th || !r) return false;
    this.view.dispatch(this.view.state.tr.setMeta(commentKey, { active: id } satisfies Meta));
    this.openThread(th, r);
    return true;
  }

  destroy(): void {
    this.off();
    document.removeEventListener("selectionchange", this.onSelection);
    this.view.dom.removeEventListener("mousedown", this.onDown);
    this.closePop();
    this.hideBubble();
  }
}

const uis = new WeakMap<EditorView, CommentUI>();

export function commentUI(view: EditorView): CommentUI | undefined {
  return uis.get(view);
}

export function revealThread(view: EditorView, id: string): void {
  const found = commentKey.getState(view.state)?.set.find(undefined, undefined, (spec) => (spec as { thread?: string }).thread === id)[0];
  view.dispatch(view.state.tr.setMeta(commentKey, { active: id } satisfies Meta));
  if (!found) return;
  const dom = view.domAtPos(found.from + (found.to > found.from ? 1 : 0)).node;
  (dom instanceof Element ? dom : dom.parentElement)?.scrollIntoView({ block: "center", behavior: "smooth" });
}

export function commentsPlugin(ctx: EditorCtx): Plugin<CommentState> {
  const threads = () => ctx.engine.comments.get(ctx.pageId()).threads;
  return new Plugin<CommentState>({
    key: commentKey,
    state: {
      init: (_, state) => ({ set: build(state, threads(), "", null), pending: null, active: "" }),
      apply: (tr: Transaction, value: CommentState, _old: EditorState, next: EditorState): CommentState => {
        const meta = tr.getMeta(commentKey) as Meta | undefined;
        const sync = tr.getMeta(ySyncPluginKey) as { isChangeOrigin?: boolean; binding?: unknown } | undefined;
        let { pending, active } = value;
        if (meta?.pending !== undefined) pending = meta.pending;
        else if (pending && tr.docChanged) pending = { from: tr.mapping.map(pending.from), to: tr.mapping.map(pending.to) };
        if (meta?.active !== undefined) active = meta.active;
        if (meta || sync?.isChangeOrigin || sync?.binding) return { set: build(next, threads(), active, pending), pending, active };
        if (!tr.docChanged) return value;
        return { set: value.set.map(tr.mapping, tr.doc), pending, active };
      },
    },
    props: {
      decorations: (state) => commentKey.getState(state)?.set ?? DecorationSet.empty,
      handleClick: (view, pos, e) => {
        if (!(e.target as Element).closest?.(".c-mark, .c-block")) return false;
        if (view.editable && !view.state.selection.empty) return false;
        return uis.get(view)?.click(pos) ?? false;
      },
    },
    view: (view) => {
      const ui = new CommentUI(view, ctx);
      uis.set(view, ui);
      return {
        update: () => ui.update(),
        destroy: () => {
          ui.destroy();
          uis.delete(view);
        },
      };
    },
  });
}
