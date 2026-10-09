import { keymap } from "prosemirror-keymap";
import { DOMSerializer, Fragment, type Node as PMNode, Slice } from "prosemirror-model";
import { type EditorState, Plugin, PluginKey, TextSelection } from "prosemirror-state";
import { Decoration, DecorationSet, type EditorView, type NodeView, type ViewMutationRecord } from "prosemirror-view";
import { ApiError } from "../../engine/api";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import type { AIAction, AIAsk, AIChoices } from "../../engine/types";
import { byId, h, setAttr, setText } from "../../ui/dom";
import { anchorToRect, type Rect } from "../../ui/floating";
import { type IconName, iconNode } from "../../ui/icons";
import { openLayer } from "../../ui/layers";
import { toast } from "../../ui/toast";
import { blockRect, caretRect, type EditorCtx } from "./context";
import { markdownReady, parseMarkdownSlice, plainTextSlice } from "./markdown";
import { registerNodeView } from "./nodeviews";
import { openPrompt } from "./popover";

interface Target {
  from: number;
  to: number;
}

interface AIState {
  target: Target | null;
  deco: DecorationSet;
}

const key = new PluginKey<AIState>("formmd-ai");
const MEDIA = new Set(["image", "video", "audio", "embed", "file", "bookmark"]);
let lastModel = "";
let choices: { ws: string; at: number; job: Promise<AIChoices> } | null = null;
let panel: AIPanel | null = null;

function loadChoices(ctx: EditorCtx): Promise<AIChoices> {
  const ws = ctx.workspaceId();
  if (choices && choices.ws === ws && Date.now() - choices.at < 60000) return choices.job;
  const job = ctx.engine.api.ai.models(ws);
  choices = { ws, at: Date.now(), job };
  job.catch(() => {
    choices = null;
  });
  return job;
}

function clean(f: Fragment): Fragment {
  const out: PMNode[] = [];
  f.forEach((n) => {
    if (MEDIA.has(n.type.name)) return;
    if (n.isLeaf || n.isTextblock) {
      out.push(n);
      return;
    }
    const inner = clean(n.content);
    if (inner.childCount && n.type.validContent(inner)) out.push(n.copy(inner));
  });
  return Fragment.fromArray(out);
}

function sliceFor(text: string, state: EditorState, at: number, ready: boolean): Slice {
  const $at = state.doc.resolve(at);
  const raw = ready ? parseMarkdownSlice(text, $at) : plainTextSlice(text);
  const content = clean(raw.content);
  return content.eq(raw.content) ? raw : new Slice(content, 0, 0);
}

const blank = schema.topNodeType.createAndFill() as PMNode;

function render(text: string, ready: boolean): DocumentFragment {
  const slice = ready ? parseMarkdownSlice(text, blank.resolve(0)) : plainTextSlice(text);
  return DOMSerializer.fromSchema(schema).serializeFragment(clean(slice.content)) as DocumentFragment;
}

function sameShell(a: Element, b: Element): boolean {
  if (a.tagName !== b.tagName || a.attributes.length !== b.attributes.length) return false;
  for (const at of Array.from(b.attributes)) if (a.getAttribute(at.name) !== at.value) return false;
  return true;
}

function reconcile(target: Node, fresh: Node): void {
  const next = Array.from(fresh.childNodes);
  next.forEach((n, i) => {
    const cur = target.childNodes[i];
    if (!cur) target.appendChild(n);
    else if (cur.isEqualNode(n)) return;
    else if (cur instanceof Text && n instanceof Text) cur.data = n.data;
    else if (cur instanceof Element && n instanceof Element && sameShell(cur, n)) reconcile(cur, n);
    else target.replaceChild(n, cur);
  });
  while (target.childNodes.length > next.length) target.lastChild?.remove();
}

function targetOf(state: EditorState): Target | null {
  return key.getState(state)?.target ?? null;
}

function rangeRect(view: EditorView, r: Target): Rect | null {
  try {
    const a = view.coordsAtPos(r.from);
    const b = view.coordsAtPos(r.to);
    const box = view.dom.getBoundingClientRect();
    return { x: box.left, y: Math.min(a.top, b.top), width: Math.min(box.width, 640), height: Math.max(a.bottom, b.bottom) - Math.min(a.top, b.top) };
  } catch {
    return caretRect(view);
  }
}

interface ActionDef {
  action: AIAction;
  label: () => string;
  icon: IconName;
  section: () => string;
}

const EDIT = () => t("Edit or review");
const WRITE = () => t("Write");
const SELECTED: ActionDef[] = [
  { action: "improve", label: () => t("Improve writing"), icon: "sparkle", section: EDIT },
  { action: "fix", label: () => t("Fix spelling and grammar"), icon: "check", section: EDIT },
  { action: "shorter", label: () => t("Make shorter"), icon: "text", section: EDIT },
  { action: "longer", label: () => t("Make longer"), icon: "text", section: EDIT },
  { action: "simplify", label: () => t("Simplify language"), icon: "lightbulb", section: EDIT },
  { action: "professional", label: () => t("Professional tone"), icon: "building", section: EDIT },
  { action: "casual", label: () => t("Casual tone"), icon: "smile", section: EDIT },
  { action: "translate", label: () => t("Translate"), icon: "globe", section: EDIT },
  { action: "explain", label: () => t("Explain this"), icon: "info", section: WRITE },
  { action: "summarize", label: () => t("Summarize"), icon: "rows", section: WRITE },
  { action: "actions", label: () => t("Find action items"), icon: "checkbox", section: WRITE },
];
const CURSOR: ActionDef[] = [
  { action: "continue", label: () => t("Continue writing"), icon: "pencil", section: WRITE },
  { action: "summarize", label: () => t("Summarize this page"), icon: "rows", section: WRITE },
  { action: "actions", label: () => t("Find action items"), icon: "checkbox", section: WRITE },
  { action: "translate", label: () => t("Translate this page"), icon: "globe", section: WRITE },
];

type Phase = "idle" | "running" | "done";

class AIPanel {
  readonly el: HTMLFormElement;
  private input: HTMLInputElement;
  private model: HTMLSelectElement;
  private go: HTMLButtonElement;
  private stop: HTMLButtonElement;
  private out: HTMLElement;
  private note: HTMLElement;
  private list: HTMLElement;
  private done: HTMLElement;
  private replaceBtn: HTMLButtonElement;
  private place: { update: () => void; release: () => void };
  private closeLayer: () => void;
  private phase: Phase = "idle";
  private text = "";
  private last: AIAsk | null = null;
  private pending: AIAction | null = null;
  private abort: AbortController | null = null;
  private frame = 0;
  private ready = false;
  private closed = false;
  private readonly selected: boolean;

  constructor(
    private readonly ctx: EditorCtx,
    view: EditorView,
  ) {
    const r = targetOf(view.state);
    this.selected = !!r && r.to > r.from;
    this.input = h("input", { class: "ai-panel__input", attrs: { type: "text", "aria-label": t("Ask AI"), placeholder: this.selected ? t("Ask AI to edit or explain the selection") : t("Ask AI to write anything"), autocomplete: "off", spellcheck: "false" } });
    this.model = h("select", { class: "ai-panel__model", attrs: { "aria-label": t("Model") } });
    this.model.hidden = true;
    this.go = h("button", { class: "btn btn--primary ai-panel__go", attrs: { type: "submit" }, text: t("Ask") });
    this.stop = h("button", { class: "btn ai-panel__stop", attrs: { type: "button" }, text: t("Stop") });
    this.stop.hidden = true;
    this.out = h("div", { class: "ai-panel__out doc-body", attrs: { "aria-live": "polite", "aria-label": t("AI response") } });
    this.out.hidden = true;
    this.note = h("p", { class: "ai-panel__note", attrs: { role: "status" } });
    this.note.hidden = true;
    this.list = h("div", { class: "ai-panel__list", attrs: { role: "group", "aria-label": t("Suggestions") } });
    this.replaceBtn = h("button", { class: "btn btn--primary", attrs: { type: "button", "data-do": "replace" }, text: t("Replace selection") });
    this.done = h(
      "div",
      { class: "ai-panel__done" },
      this.replaceBtn,
      h("button", { class: this.selected ? "btn" : "btn btn--primary", attrs: { type: "button", "data-do": "below" }, text: this.selected ? t("Insert below") : t("Insert") }),
      h("button", { class: "btn", attrs: { type: "button", "data-do": "again" }, text: t("Try again") }),
      h("button", { class: "btn btn--ghost", attrs: { type: "button", "data-do": "discard" }, text: t("Discard") }),
    );
    this.replaceBtn.hidden = !this.selected;
    this.done.hidden = true;
    const bar = h("div", { class: "ai-panel__bar" }, h("span", { class: "ai-panel__icon" }, iconNode("sparkle")), this.input, this.model, this.go, this.stop);
    this.el = h("form", { class: "popover ai-panel", attrs: { role: "dialog", "aria-label": t("Ask AI") } }, this.out, bar, this.note, this.list, this.done);
    this.fillList(this.selected ? SELECTED : CURSOR);
    this.el.addEventListener("submit", (e) => {
      e.preventDefault();
      this.submit();
    });
    this.list.addEventListener("click", (e) => {
      const b = (e.target as Element).closest<HTMLElement>("[data-action]");
      if (b) this.pick(b.dataset.action as AIAction);
    });
    this.done.addEventListener("click", (e) => {
      const b = (e.target as Element).closest<HTMLElement>("[data-do]");
      if (!b) return;
      const what = b.dataset.do;
      if (what === "replace" || what === "below") this.accept(what);
      else if (what === "again" && this.last) this.run(this.last);
      else if (what === "discard") this.close();
    });
    this.stop.addEventListener("click", () => this.abort?.abort());
    this.model.addEventListener("change", () => {
      lastModel = this.model.value;
    });
    byId("overlay-root").append(this.el);
    this.place = anchorToRect(
      this.el,
      () => {
        const v = this.ctx.view();
        const tr = v && targetOf(v.state);
        return v && tr ? rangeRect(v, tr) : null;
      },
      { side: "bottom", align: "start", offset: 8 },
    );
    requestAnimationFrame(() => this.el.classList.add("is-visible"));
    this.closeLayer = openLayer({ el: this.el, modal: false, dismissOnOutside: true, initialFocus: this.input, returnFocus: null, onClose: () => this.teardown() });
    void markdownReady().then((ok) => {
      this.ready = ok;
    });
    void loadChoices(ctx).then(
      (c) => this.setChoices(c),
      () => this.say(t("AI models could not be loaded."), true),
    );
  }

  private fillList(defs: ActionDef[]): void {
    let section = "";
    for (const d of defs) {
      if (d.section() !== section) {
        section = d.section();
        this.list.append(h("div", { class: "listbox__section", text: section }));
      }
      const b = h("button", { class: "listbox__item ai-panel__item", attrs: { type: "button", "data-action": d.action } }, h("span", { class: "listbox__icon" }, iconNode(d.icon)), h("span", { class: "listbox__label", text: d.label() }), h("span", { class: "listbox__hint" }));
      this.list.append(b);
    }
  }

  private setChoices(c: AIChoices): void {
    if (this.closed) return;
    const models = c.models.filter((m) => !m.audio);
    if (!c.enabled || !models.length) {
      this.say(models.length ? t("AI is turned off here. An admin can turn it on.") : t("No AI model is set up yet. An admin can add one under Admin settings, AI."), true);
      this.input.disabled = true;
      this.go.disabled = true;
      this.list.hidden = true;
      return;
    }
    for (const m of models) this.model.append(h("option", { attrs: { value: m.ref }, text: m.label }));
    const want = models.some((m) => m.ref === lastModel) ? lastModel : models.some((m) => m.ref === c.default) ? c.default : (models[0]?.ref ?? "");
    this.model.value = want;
    lastModel = want;
    this.model.hidden = models.length < 2;
    this.place.update();
  }

  private say(text: string, bad = false): void {
    setText(this.note, text);
    this.note.hidden = !text;
    this.note.classList.toggle("is-error", bad);
  }

  private pick(action: AIAction): void {
    if (action === "translate") {
      this.pending = "translate";
      this.input.value = "";
      setAttr(this.input, "placeholder", t("Translate into which language? For example, Spanish"));
      this.input.focus();
      return;
    }
    this.run(this.ask(action));
  }

  private ask(action: AIAction, extra: Partial<AIAsk> = {}): AIAsk {
    const view = this.ctx.view();
    const r = view ? targetOf(view.state) : null;
    const body: AIAsk = { action, workspace_id: this.ctx.workspaceId(), page_id: this.ctx.pageId(), ...extra };
    if (this.model.value) body.model = this.model.value;
    if (view && r) {
      if (r.to > r.from) body.selection = this.ctx.markdownBetween(r.from, r.to);
      body.before = this.ctx.markdownBetween(0, r.from).slice(-12000);
    }
    return body;
  }

  private submit(): void {
    const value = this.input.value.trim();
    if (this.phase === "running") return;
    if (this.pending === "translate") {
      if (!value) return;
      this.pending = null;
      this.run(this.ask("translate", { language: value }));
      return;
    }
    if (!value) return;
    if (this.phase === "done" && this.text) {
      const body = this.ask("ask", { instruction: value });
      body.selection = this.text;
      this.run(body);
      return;
    }
    this.run(this.ask("ask", { instruction: value }));
  }

  private run(body: AIAsk): void {
    this.abort?.abort();
    const abort = new AbortController();
    this.abort = abort;
    this.last = body;
    this.text = "";
    this.setPhase("running");
    this.say(t("Writing..."));
    this.input.value = "";
    let got = false;
    void this.ctx.engine.api.ai
      .complete(
        body,
        (chunk) => {
          if (this.abort !== abort) return;
          this.text += chunk;
          if (!got) {
            got = true;
            this.out.hidden = false;
            this.say("");
          }
          this.paint();
        },
        abort.signal,
      )
      .then(
        () => {
          if (this.abort !== abort) return;
          this.finish(this.text.trim() ? "" : t("The model returned no text."));
        },
        (e: unknown) => {
          if (this.abort !== abort) return;
          const cancelled = e instanceof ApiError && e.code === "cancelled";
          this.finish(cancelled ? "" : e instanceof ApiError ? e.message : t("The AI request failed."));
        },
      );
  }

  private finish(error: string): void {
    this.abort = null;
    cancelAnimationFrame(this.frame);
    this.frame = 0;
    if (this.text) reconcile(this.out, render(this.text, this.ready));
    this.setPhase(this.text.trim() ? "done" : "idle");
    this.say(error, !!error);
    this.place.update();
  }

  private paint(): void {
    if (this.frame) return;
    this.frame = requestAnimationFrame(() => {
      this.frame = 0;
      reconcile(this.out, render(this.text, this.ready));
      this.out.scrollTop = this.out.scrollHeight;
      this.place.update();
    });
  }

  private setPhase(p: Phase): void {
    this.phase = p;
    this.go.hidden = p === "running";
    this.stop.hidden = p !== "running";
    this.list.hidden = p !== "idle";
    this.done.hidden = p !== "done";
    this.input.disabled = p === "running";
    if (p === "idle") this.out.hidden = true;
    setAttr(this.input, "placeholder", p === "done" ? t("Tell AI what to change") : this.selected ? t("Ask AI to edit or explain the selection") : t("Ask AI to write anything"));
    if (p !== "running") this.input.focus({ preventScroll: true });
  }

  private accept(mode: "replace" | "below"): void {
    const view = this.ctx.view();
    const r = view ? targetOf(view.state) : null;
    const text = this.text.trim();
    if (!view || !r || !text) return;
    const { state } = view;
    const tr = state.tr;
    try {
      if (mode === "replace" && r.to > r.from) {
        tr.replaceRange(r.from, r.to, sliceFor(text, state, r.from, this.ready));
      } else {
        const $p = state.doc.resolve(r.to);
        const empty = r.from === r.to && $p.parent.isTextblock && $p.parent.content.size === 0 && $p.depth > 0;
        if (empty) {
          const from = $p.before();
          tr.replaceRange(from, $p.after(), sliceFor(text, state, from, this.ready));
        } else {
          const at = $p.depth > 0 ? $p.after() : r.to;
          tr.replaceRange(at, at, sliceFor(text, state, at, this.ready));
        }
      }
    } catch {
      toast({ message: t("The answer could not be placed here."), tone: "warning" });
      return;
    }
    const end = Math.min(tr.mapping.map(r.to, 1), tr.doc.content.size);
    tr.setSelection(TextSelection.near(tr.doc.resolve(end), -1)).scrollIntoView();
    view.dispatch(tr);
    this.close();
    view.focus();
  }

  close(): void {
    this.closeLayer();
  }

  private teardown(): void {
    if (this.closed) return;
    this.closed = true;
    this.abort?.abort();
    this.abort = null;
    cancelAnimationFrame(this.frame);
    this.place.release();
    this.el.remove();
    if (panel === this) panel = null;
    const view = this.ctx.view();
    if (view && targetOf(view.state)) view.dispatch(view.state.tr.setMeta(key, null));
  }
}

function domSelection(view: EditorView): { anchor: number; head: number } | null {
  const dom = document.getSelection();
  if (!dom?.anchorNode || !dom.focusNode || !view.dom.contains(dom.anchorNode) || !view.dom.contains(dom.focusNode)) return null;
  try {
    const anchor = view.posAtDOM(dom.anchorNode, dom.anchorOffset);
    const head = view.posAtDOM(dom.focusNode, dom.focusOffset);
    const doc = view.state.doc;
    return doc.resolve(anchor).parent.inlineContent && doc.resolve(head).parent.inlineContent ? { anchor, head } : null;
  } catch {
    return null;
  }
}

export function openAI(ctx: EditorCtx, view: EditorView): boolean {
  if (!ctx.writable() && !ctx.suggesting()) return false;
  panel?.close();
  const live = domSelection(view);
  const cur = view.state.selection;
  if (live && cur instanceof TextSelection && (cur.anchor !== live.anchor || cur.head !== live.head)) view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, live.anchor, live.head)));
  const { from, to } = view.state.selection;
  view.dispatch(view.state.tr.setMeta(key, { from, to }));
  panel = new AIPanel(ctx, view);
  return true;
}

export function aiPlugins(ctx: EditorCtx): Plugin[] {
  return [
    new Plugin<AIState>({
      key,
      state: {
        init: () => ({ target: null, deco: DecorationSet.empty }),
        apply(tr, prev) {
          const meta = tr.getMeta(key) as Target | null | undefined;
          if (meta === null) return { target: null, deco: DecorationSet.empty };
          if (meta) return { target: meta, deco: meta.to > meta.from ? DecorationSet.create(tr.doc, [Decoration.inline(meta.from, meta.to, { class: "ai-target" })]) : DecorationSet.empty };
          if (!prev.target || !tr.docChanged) return prev;
          const from = tr.mapping.map(prev.target.from, 1);
          const to = Math.max(from, tr.mapping.map(prev.target.to, -1));
          return { target: { from, to }, deco: prev.deco.map(tr.mapping, tr.doc) };
        },
      },
      props: { decorations: (s) => key.getState(s)?.deco ?? null },
    }),
    keymap({ "Mod-j": (_s, _d, view) => (view ? openAI(ctx, view) : false) }),
  ];
}

export function newAIBlock(): PMNode {
  return schema.nodes.ai_block!.create({ id: newBlockId(), prompt: "" }, [schema.nodes.paragraph!.create({ id: newBlockId() })]);
}

class AIBlockView implements NodeView {
  readonly dom: HTMLElement;
  readonly contentDOM: HTMLElement;
  private promptEl: HTMLElement;
  private runBtn: HTMLButtonElement;
  private live: HTMLElement | null = null;
  private abort: AbortController | null = null;
  private frame = 0;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: () => number | undefined,
    private readonly ctx: EditorCtx,
  ) {
    this.promptEl = h("span", { class: "ai-block__prompt", attrs: { role: "button", tabindex: "0" } });
    this.runBtn = h("button", { class: "ai-block__run", attrs: { type: "button" }, text: t("Regenerate") });
    const head = h("div", { class: "ai-block__head", attrs: { contenteditable: "false" } }, h("span", { class: "ai-block__icon" }, iconNode("sparkle")), this.promptEl, this.runBtn);
    this.contentDOM = h("div", { class: "ai-block__body" });
    this.dom = h("div", { class: "ai-block" }, head, this.contentDOM);
    this.promptEl.addEventListener("click", () => this.editPrompt());
    this.promptEl.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        this.editPrompt();
      }
    });
    this.runBtn.addEventListener("click", () => {
      if (this.abort) this.abort.abort();
      else void this.generate();
    });
    this.sync();
    if (!String(node.attrs.prompt ?? "") && ctx.writable() && node.textContent === "") requestAnimationFrame(() => this.editPrompt());
  }

  private sync(): void {
    const prompt = String(this.node.attrs.prompt ?? "");
    setAttr(this.dom, "data-block", String(this.node.attrs.id ?? "") || null);
    setAttr(this.dom, "data-prompt", prompt);
    setText(this.promptEl, prompt || t("Describe what this block should write"));
    this.promptEl.classList.toggle("is-empty", !prompt);
    setText(this.runBtn, this.abort ? t("Stop") : t("Regenerate"));
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.sync();
    return true;
  }

  stopEvent(e: Event): boolean {
    return !!(e.target as Element | null)?.closest?.(".ai-block__head");
  }

  ignoreMutation(m: ViewMutationRecord): boolean {
    if (m.type === "selection") return false;
    return !this.contentDOM.contains(m.target);
  }

  destroy(): void {
    this.abort?.abort();
    cancelAnimationFrame(this.frame);
  }

  private editPrompt(): void {
    if (!this.ctx.writable() || this.abort) return;
    openPrompt({
      label: t("AI block"),
      placeholder: t("For example, summarize this page in three bullets"),
      value: String(this.node.attrs.prompt ?? ""),
      action: t("Generate"),
      rect: () => {
        const pos = this.getPos();
        return pos === undefined ? null : blockRect(this.view, pos);
      },
      onSubmit: (prompt) => {
        const pos = this.getPos();
        if (pos === undefined) return;
        this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...this.node.attrs, prompt: prompt.slice(0, 2000) }));
        void this.generate();
      },
    });
  }

  private async generate(): Promise<void> {
    const prompt = String(this.node.attrs.prompt ?? "");
    if (!prompt) return this.editPrompt();
    const start = this.getPos();
    if (start === undefined || this.abort) return;
    const abort = new AbortController();
    this.abort = abort;
    this.sync();
    this.dom.classList.add("is-running");
    const ready = await markdownReady();
    let text = "";
    const live = h("div", { class: "ai-block__live" });
    const body: AIAsk = { action: "block", workspace_id: this.ctx.workspaceId(), page_id: this.ctx.pageId(), instruction: prompt, before: this.ctx.markdownBetween(0, start).slice(-12000) };
    if (lastModel) body.model = lastModel;
    let error = "";
    try {
      await this.ctx.engine.api.ai.complete(
        body,
        (chunk) => {
          text += chunk;
          if (!this.live) {
            this.live = live;
            this.dom.append(live);
            this.dom.classList.add("is-live");
          }
          if (!this.frame)
            this.frame = requestAnimationFrame(() => {
              this.frame = 0;
              reconcile(live, render(text, ready));
            });
        },
        abort.signal,
      );
    } catch (e) {
      if (!(e instanceof ApiError && e.code === "cancelled")) error = e instanceof ApiError ? e.message : t("The AI request failed.");
    }
    cancelAnimationFrame(this.frame);
    this.frame = 0;
    const pos = this.getPos();
    const node = pos === undefined ? null : this.view.state.doc.nodeAt(pos);
    if (text.trim() && pos !== undefined && node?.type === this.node.type) {
      const state = this.view.state;
      try {
        this.view.dispatch(state.tr.replaceRange(pos + 1, pos + node.nodeSize - 1, sliceFor(text.trim(), state, pos + 1, ready)));
      } catch {
        error = t("The answer could not be placed here.");
      }
    }
    live.remove();
    this.live = null;
    this.dom.classList.remove("is-live", "is-running");
    this.abort = null;
    this.sync();
    if (error) toast({ message: error, tone: "warning" });
  }
}

registerNodeView("ai_block", (ctx) => (node, view, getPos) => new AIBlockView(node, view, getPos, ctx));
