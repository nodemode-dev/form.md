import { type EditorState, Plugin, PluginKey, TextSelection, type Transaction } from "prosemirror-state";
import { Decoration, DecorationSet, type EditorView } from "prosemirror-view";
import { ySyncPluginKey } from "y-prosemirror";
import type { Suggestion } from "../../engine";
import { t } from "../../engine/i18n";
import { suggestionView } from "../../ui/comment-view";
import { byId, h } from "../../ui/dom";
import { errorMessage } from "../../ui/errors";
import { anchorToRect, type Rect } from "../../ui/floating";
import { openLayer } from "../../ui/layers";
import { toast } from "../../ui/toast";
import { encodeRange, resolveAnchor } from "./comments";
import type { EditorCtx } from "./context";

interface Draft {
  id: string;
  from: number;
  to: number;
  text: string;
  start: string;
  end: string;
}

interface SuggestState {
  set: DecorationSet;
  draft: Draft | null;
}

type Meta = { draft?: Draft | null; rebuild?: true; apply?: true };

export const suggestKey = new PluginKey<SuggestState>("formmd-suggest");

const IDLE_MS = 1500;

export function suggestionRange(state: EditorState, s: { start: string; end: string }): { from: number; to: number } | null {
  const a = resolveAnchor(state, s.start);
  const z = resolveAnchor(state, s.end);
  if (a === null || z === null) return null;
  const size = state.doc.content.size;
  const from = Math.max(0, Math.min(a, z, size));
  return { from, to: Math.min(size, Math.max(a, z, from)) };
}

function insertWidget(id: string, text: string): HTMLElement {
  return h("span", { class: "s-ins", attrs: { "data-suggestion": id, contenteditable: "false" }, text });
}

function build(state: EditorState, list: Suggestion[], draft: Draft | null): DecorationSet {
  const decos: Decoration[] = [];
  const add = (id: string, from: number, to: number, text: string) => {
    if (to > from) decos.push(Decoration.inline(from, to, { class: "s-del", "data-suggestion": id }, { suggestion: id }));
    if (text) decos.push(Decoration.widget(to, () => insertWidget(id, text), { side: -1, key: `${id}:${text}`, suggestion: id, ignoreSelection: true }));
  };
  for (const s of list) {
    if (s.status !== "open" || (draft && draft.id === s.id)) continue;
    const r = suggestionRange(state, s);
    if (r) add(s.id, r.from, r.to, s.text);
  }
  if (draft && draft.to <= state.doc.content.size) add(draft.id || "draft", draft.from, draft.to, draft.text);
  return DecorationSet.create(state.doc, decos);
}

function sameBlock(state: EditorState, from: number, to: number): boolean {
  if (from < 0 || to > state.doc.content.size || from > to) return false;
  const $a = state.doc.resolve(from);
  const $b = state.doc.resolve(to);
  return $a.sameParent($b) && $a.parent.isTextblock && !$a.parent.type.spec.code;
}

class SuggestUI {
  private timer: ReturnType<typeof setTimeout> | undefined;
  private saving: Promise<void> | null = null;
  private pop: { el: HTMLElement; close: () => void } | null = null;
  private off: () => void;

  constructor(
    private readonly view: EditorView,
    private readonly ctx: EditorCtx,
  ) {
    this.off = ctx.engine.comments.subscribe((id) => {
      if (id === this.ctx.pageId() && !this.view.isDestroyed) this.view.dispatch(this.view.state.tr.setMeta(suggestKey, { rebuild: true } satisfies Meta));
    });
    this.view.dom.addEventListener("blur", this.onBlur);
  }

  private list(): Suggestion[] {
    return this.ctx.engine.comments.get(this.ctx.pageId()).suggestions;
  }

  private draft(): Draft | null {
    return suggestKey.getState(this.view.state)?.draft ?? null;
  }

  private setDraft(d: Draft | null, caret?: number): void {
    let tr = this.view.state.tr.setMeta(suggestKey, { draft: d } satisfies Meta);
    if (caret !== undefined) tr = tr.setSelection(TextSelection.create(tr.doc, caret));
    this.view.dispatch(tr);
    clearTimeout(this.timer);
    if (d) this.timer = setTimeout(() => void this.commit(), IDLE_MS);
  }

  private onBlur = () => void this.commit();

  private reopen(pos: number): Draft | null {
    const me = this.ctx.engine.state.get("me")?.user.id ?? "";
    for (const s of this.list()) {
      if (s.status !== "open" || s.author_id !== me) continue;
      const r = suggestionRange(this.view.state, s);
      if (r && r.to === pos) return { id: s.id, from: r.from, to: r.to, text: s.text, start: s.start, end: s.end };
    }
    return null;
  }

  private current(from: number, to: number): Draft | null {
    const d = this.draft();
    if (from !== to) return null;
    if (d && (to === d.to || to === d.from)) return d;
    return this.reopen(to);
  }

  private anchor(d: Draft): Draft | null {
    const r = encodeRange(this.view.state, d.from, d.to);
    return r ? { ...d, start: r.start, end: r.end } : null;
  }

  input(from: number, to: number, text: string): boolean {
    if (!sameBlock(this.view.state, from, to)) {
      toast({ id: "suggest-scope", message: t("Suggestions change text inside one block at a time.") });
      return true;
    }
    const cur = this.current(from, to);
    const next = cur ? { ...cur, text: cur.text + text } : this.anchor({ id: "", from, to, text, start: "", end: "" });
    if (next) this.setDraft(next, next.to);
    return true;
  }

  erase(forward: boolean): boolean {
    const sel = this.view.state.selection;
    if (!sel.empty) return this.input(sel.from, sel.to, "");
    const pos = sel.head;
    const cur = this.current(pos, pos);
    if (cur && !forward && cur.text) {
      this.setDraft({ ...cur, text: cur.text.slice(0, -[...cur.text].at(-1)!.length) }, cur.to);
      return true;
    }
    const base = cur ?? { id: "", from: pos, to: pos, text: "", start: "", end: "" };
    const from = forward ? base.from : base.from - 1;
    const to = forward ? base.to + 1 : base.to;
    if (!sameBlock(this.view.state, from, to)) return true;
    const next = this.anchor({ ...base, from, to });
    if (next) this.setDraft(next, forward ? base.from : next.to);
    return true;
  }

  async commit(): Promise<void> {
    clearTimeout(this.timer);
    if (this.saving) await this.saving;
    const d = this.draft();
    if (!d || this.view.isDestroyed) return;
    const pageId = this.ctx.pageId();
    const original = this.view.state.doc.textBetween(d.from, d.to, " ", " ");
    const body = { start: d.start, end: d.end, original, text: d.text };
    const empty = !d.text && d.from === d.to;
    const api = this.ctx.engine.api.comments;
    const job = (async () => {
      let id = d.id;
      try {
        if (d.id && empty) await api.decide(d.id, false);
        else if (d.id) await api.revise(d.id, body);
        else if (!empty) {
          id = (await api.suggest(pageId, body)).suggestion.id;
          const live = this.draft();
          if (!this.view.isDestroyed && live && !live.id && (live.start === d.start || live.end === d.end)) this.view.dispatch(this.view.state.tr.setMeta(suggestKey, { draft: { ...live, id } } satisfies Meta));
        }
        await this.ctx.engine.comments.load(pageId, true);
      } catch (e) {
        toast({ tone: "danger", message: errorMessage(e) });
      }
      const now = this.draft();
      if (!this.view.isDestroyed && now && (now === d || (id && now.id === id && now.text === d.text && now.start === d.start && now.end === d.end))) this.view.dispatch(this.view.state.tr.setMeta(suggestKey, { draft: null } satisfies Meta));
    })();
    this.saving = job;
    await job;
    this.saving = null;
  }

  selectionMoved(): void {
    const d = this.draft();
    const sel = this.view.state.selection;
    if (d && (!sel.empty || (sel.head !== d.to && sel.head !== d.from))) void this.commit();
  }

  async decide(s: Suggestion, accept: boolean): Promise<void> {
    const pageId = this.ctx.pageId();
    try {
      await this.ctx.engine.api.comments.decide(s.id, accept);
      if (accept) {
        const r = suggestionRange(this.view.state, s);
        if (r) {
          let tr = this.view.state.tr;
          tr = s.text ? tr.insertText(s.text, r.from, r.to) : tr.delete(r.from, r.to);
          this.view.dispatch(tr.setMeta(suggestKey, { apply: true } satisfies Meta));
        }
      }
      await this.ctx.engine.comments.load(pageId, true);
    } catch (e) {
      toast({ tone: "danger", message: errorMessage(e) });
    }
  }

  open(id: string, rect: () => Rect | null): void {
    const s = this.list().find((x) => x.id === id);
    if (!s) return;
    this.pop?.close();
    const v = suggestionView(this.ctx.engine, this.ctx.pageId(), s, {
      onDecide: (x, accept) => {
        this.pop?.close();
        void this.decide(x, accept);
      },
    });
    const el = h("div", { class: "popover cpop" }, v.el);
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
        this.pop = null;
      },
    });
    this.pop = { el, close };
  }

  destroy(): void {
    void this.commit();
    this.off();
    this.view.dom.removeEventListener("blur", this.onBlur);
    this.pop?.close();
  }
}

const uis = new WeakMap<EditorView, SuggestUI>();

export function suggestUI(view: EditorView): SuggestUI | undefined {
  return uis.get(view);
}

export function suggestPlugin(ctx: EditorCtx): Plugin<SuggestState> {
  const list = () => ctx.engine.comments.get(ctx.pageId()).suggestions;
  const on = () => ctx.suggesting();
  return new Plugin<SuggestState>({
    key: suggestKey,
    state: {
      init: (_, state) => ({ set: build(state, list(), null), draft: null }),
      apply: (tr: Transaction, value: SuggestState, _old: EditorState, next: EditorState): SuggestState => {
        const meta = tr.getMeta(suggestKey) as Meta | undefined;
        const sync = tr.getMeta(ySyncPluginKey) as { isChangeOrigin?: boolean; binding?: unknown } | undefined;
        let draft = meta?.draft !== undefined ? meta.draft : value.draft;
        if (draft && meta?.draft === undefined && tr.docChanged) {
          const r = sync?.isChangeOrigin ? suggestionRange(next, draft) : { from: tr.mapping.map(draft.from, -1), to: tr.mapping.map(draft.to, 1) };
          draft = r ? { ...draft, ...r } : null;
        }
        if (meta || sync?.isChangeOrigin || sync?.binding) return { set: build(next, list(), draft), draft };
        if (!tr.docChanged) return value;
        return { set: value.set.map(tr.mapping, tr.doc), draft };
      },
    },
    filterTransaction: (tr) => {
      if (!on() || !tr.docChanged) return true;
      if (tr.getMeta(ySyncPluginKey) !== undefined) return true;
      return !!(tr.getMeta(suggestKey) as Meta | undefined)?.apply;
    },
    props: {
      decorations: (state) => suggestKey.getState(state)?.set ?? DecorationSet.empty,
      handleTextInput: (view, from, to, text) => (on() ? (uis.get(view)?.input(from, to, text) ?? true) : false),
      handleKeyDown: (view, e) => {
        if (!on()) return false;
        const ui = uis.get(view);
        if (e.key === "Backspace" || e.key === "Delete") {
          e.preventDefault();
          return ui?.erase(e.key === "Delete") ?? true;
        }
        if (e.key === "Enter") {
          e.preventDefault();
          void ui?.commit();
          return true;
        }
        if (e.key === "Tab") {
          e.preventDefault();
          return true;
        }
        return false;
      },
      handlePaste: (view, e) => {
        if (!on()) return false;
        const text = (e.clipboardData?.getData("text/plain") ?? "").replace(/\s*\n\s*/g, " ");
        const { from, to } = view.state.selection;
        if (text) uis.get(view)?.input(from, to, text);
        return true;
      },
      handleDrop: () => on(),
      handleClick: (view, _pos, e) => {
        const el = (e.target as Element).closest?.<HTMLElement>("[data-suggestion]");
        const id = el?.dataset.suggestion;
        if (!el || !id || id === "draft") return false;
        uis.get(view)?.open(id, () => {
          const r = el.getBoundingClientRect();
          return { x: r.left, y: r.top, width: r.width, height: r.height };
        });
        return true;
      },
    },
    view: (view) => {
      const ui = new SuggestUI(view, ctx);
      uis.set(view, ui);
      return {
        update: (v, prev) => {
          if (!v.state.selection.eq(prev.selection)) ui.selectionMoved();
        },
        destroy: () => {
          ui.destroy();
          uis.delete(view);
        },
      };
    },
  });
}
