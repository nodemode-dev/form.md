import { InputRule } from "prosemirror-inputrules";
import type { Node as PMNode, NodeType } from "prosemirror-model";
import { NodeSelection, Plugin, TextSelection } from "prosemirror-state";
import type { EditorView, NodeView, ViewMutationRecord } from "prosemirror-view";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { byId, h, setAttr, setText } from "../../ui/dom";
import { anchorToRect } from "../../ui/floating";
import { openLayer } from "../../ui/layers";
import type { EditorCtx } from "./context";
import { mathElement } from "./mathml";

type Renderer = typeof import("./math-render");
type GetPos = () => number | undefined;

const N = schema.nodes as Record<string, NodeType>;

let job: Promise<Renderer> | null = null;
let ready: Renderer | null = null;

function loadMath(): Promise<Renderer> {
  job ??= import("./math-render").then(
    (m) => (ready = m),
    (e: unknown) => {
      job = null;
      throw e;
    },
  );
  return job;
}

let editNextBlock = "";
let editNextInline = false;

export function newMathBlock(): PMNode {
  const id = newBlockId();
  editNextBlock = id;
  return (N.math_block as NodeType).create({ id, tex: "", html: "" });
}

export function insertInlineMath(view: EditorView): boolean {
  const { state } = view;
  const { from, to, empty } = state.selection;
  const $from = state.doc.resolve(from);
  if (!$from.parent.inlineContent || $from.parent.type.spec.code) return false;
  const tex = empty ? "" : state.doc.textBetween(from, to, " ");
  if (!tex) editNextInline = true;
  const node = (N.math_inline as NodeType).create({ tex, html: "" });
  const tr = state.tr.replaceRangeWith(from, to, node);
  view.dispatch(tr.setSelection(TextSelection.create(tr.doc, Math.min(from + 1, tr.doc.content.size))).scrollIntoView());
  view.focus();
  return true;
}

interface PanelOptions {
  anchor: HTMLElement;
  block: boolean;
  value: string;
  onInput: (tex: string) => void;
  onDone: (tex: string) => void;
  onCancel: () => void;
}

function openPanel(o: PanelOptions): () => void {
  const field = o.block
    ? h("textarea", { class: "math-panel__field", attrs: { rows: "3", spellcheck: "false", "aria-label": t("TeX equation"), placeholder: "E = mc^2" } })
    : h("input", { class: "math-panel__field", attrs: { type: "text", spellcheck: "false", "aria-label": t("TeX equation"), placeholder: "x^2" } });
  field.value = o.value;
  const status = h("span", { class: "math-panel__status", attrs: { "aria-live": "polite" } });
  const done = h("button", { class: "btn btn--primary btn--sm", attrs: { type: "button" }, text: t("Done") });
  const panel = h("div", { class: "popover math-panel", attrs: { role: "dialog", "aria-label": t("Edit equation") } }, field, h("div", { class: "math-panel__foot" }, status, done));
  byId("overlay-root").append(panel);
  const place = anchorToRect(
    panel,
    () => {
      const r = o.anchor.getBoundingClientRect();
      return { x: r.left, y: r.top, width: r.width, height: r.height };
    },
    { side: "bottom", align: "start", offset: 6 },
  );
  requestAnimationFrame(() => panel.classList.add("is-visible"));
  let finished = false;
  const finish = (commit: boolean) => {
    if (finished) return;
    finished = true;
    if (commit) o.onDone(field.value.trim());
    else o.onCancel();
  };
  const close = openLayer({
    el: panel,
    modal: false,
    dismissOnOutside: true,
    initialFocus: field,
    returnFocus: null,
    onClose: () => {
      place.release();
      panel.remove();
      finish(true);
    },
  });
  field.addEventListener("input", () => o.onInput(field.value.trim()));
  field.addEventListener("keydown", (ev) => {
    const e = ev as KeyboardEvent;
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      finish(false);
      close();
    } else if (e.key === "Enter" && (!o.block || e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      close();
    }
  });
  done.addEventListener("click", () => close());
  setText(status, o.block ? t("Ctrl+Enter to finish") : t("Enter to finish"));
  return close;
}

export class MathView implements NodeView {
  dom: HTMLElement;
  private out: HTMLElement;
  private sig = "";
  private editing: (() => void) | null = null;
  private previewSeq = 0;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
    private readonly ctx: EditorCtx,
    private readonly block: boolean,
  ) {
    this.out = h("span", { class: "math__out" });
    this.dom = block
      ? h("div", { class: "math-block", attrs: { contenteditable: "false" } }, this.out)
      : h("span", { class: "math-inline", attrs: { contenteditable: "false" } }, this.out);
    if (block) this.dom.dataset.block = String(node.attrs.id ?? "");
    this.dom.addEventListener("click", (e) => {
      e.preventDefault();
      this.edit();
    });
    this.dom.addEventListener("math-edit", () => this.edit());
    this.paint();
    const wantEdit = block ? editNextBlock !== "" && editNextBlock === node.attrs.id : editNextInline && !node.attrs.tex;
    if (wantEdit) {
      editNextBlock = "";
      editNextInline = false;
      requestAnimationFrame(() => this.edit());
    }
  }

  private show(tex: string, markup: string, error = ""): void {
    setAttr(this.dom, "data-empty", tex ? null : "true");
    setAttr(this.dom, "data-error", error ? "true" : null);
    if (!tex) {
      setText(this.out, this.block ? t("Add a TeX equation") : t("New equation"));
      return;
    }
    const el = markup ? mathElement(markup) : null;
    if (el) {
      this.out.replaceChildren(el);
      return;
    }
    this.out.replaceChildren(h("code", { class: "math-src", text: tex }));
    if (error) setAttr(this.out.firstElementChild as HTMLElement, "title", error);
  }

  private paint(): void {
    const tex = String(this.node.attrs.tex ?? "");
    const cache = String(this.node.attrs.html ?? "");
    const sig = `${tex}\u0000${cache}`;
    if (sig === this.sig || this.editing) return;
    this.sig = sig;
    if (!tex || (cache && mathElement(cache))) {
      this.show(tex, cache);
      return;
    }
    if (!this.out.firstChild) this.show(tex, "");
    const mine = ++this.previewSeq;
    void loadMath().then((r) => {
      if (mine !== this.previewSeq || this.editing) return;
      const res = r.renderTex(tex, this.block);
      if ("mathml" in res) {
        this.show(tex, res.mathml);
        if (this.ctx.writable() && this.view.editable) this.store(tex, res.mathml);
      } else this.show(tex, "", res.error);
    }, this.ctx.engine.reportError);
  }

  private store(tex: string, html: string): void {
    const pos = this.getPos();
    if (pos === undefined) return;
    const cur = this.view.state.doc.nodeAt(pos);
    if (!cur || cur.type !== this.node.type || cur.attrs.tex !== tex || cur.attrs.html === html) return;
    this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...cur.attrs, tex, html }).setMeta("addToHistory", false));
  }

  private edit(): void {
    if (this.editing || !this.view.editable || !this.ctx.writable()) return;
    const original = String(this.node.attrs.tex ?? "");
    let latest = { tex: original, html: String(this.node.attrs.html ?? "") };
    this.dom.classList.add("is-editing");
    const preview = (tex: string) => {
      const mine = ++this.previewSeq;
      if (!tex) {
        this.show("", "");
        latest = { tex: "", html: "" };
        return;
      }
      void loadMath().then((r) => {
        if (mine !== this.previewSeq) return;
        const res = r.renderTex(tex, this.block);
        if ("mathml" in res) {
          latest = { tex, html: res.mathml };
          this.show(tex, res.mathml);
        } else {
          latest = { tex, html: "" };
          this.show(tex, "", res.error);
        }
      }, this.ctx.engine.reportError);
    };
    void loadMath();
    this.editing = openPanel({
      anchor: this.dom,
      block: this.block,
      value: original,
      onInput: preview,
      onDone: (tex) => {
        this.editing = null;
        this.dom.classList.remove("is-editing");
        this.commit(tex, tex === latest.tex ? latest.html : "");
      },
      onCancel: () => {
        this.editing = null;
        this.dom.classList.remove("is-editing");
        this.sig = "";
        this.paint();
        this.view.focus();
      },
    });
  }

  private commit(tex: string, html: string): void {
    const pos = this.getPos();
    if (pos === undefined) return;
    const { state } = this.view;
    if (!tex && !this.block) {
      this.view.dispatch(state.tr.delete(pos, pos + this.node.nodeSize));
      this.view.focus();
      return;
    }
    this.sig = "";
    let tr = state.tr.setNodeMarkup(pos, undefined, { ...this.node.attrs, tex, html });
    if (this.block) tr = tr.setSelection(NodeSelection.create(tr.doc, pos));
    else tr = tr.setSelection(TextSelection.create(tr.doc, pos + 1));
    this.view.dispatch(tr);
    this.view.focus();
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    if (this.block) this.dom.dataset.block = String(node.attrs.id ?? "");
    this.paint();
    return true;
  }

  selectNode(): void {
    this.dom.classList.add("is-selected");
  }

  deselectNode(): void {
    this.dom.classList.remove("is-selected");
  }

  stopEvent(e: Event): boolean {
    return e.type === "click" || e.type === "mousedown";
  }

  ignoreMutation(_m: ViewMutationRecord): boolean {
    return true;
  }

  destroy(): void {
    this.previewSeq++;
    this.editing?.();
    this.editing = null;
  }
}

export function mathRules(): InputRule[] {
  return [
    new InputRule(/^\$\$\s$/, (state, _m, start, end) => {
      const $start = state.doc.resolve(start);
      if ($start.parent.type !== N.paragraph || $start.parent.content.size !== end - start) return null;
      const id = newBlockId();
      editNextBlock = id;
      const from = $start.before();
      const to = $start.after();
      const tr = state.tr.replaceWith(from, to, (N.math_block as NodeType).create({ id, tex: "", html: "" }));
      return tr.setSelection(NodeSelection.create(tr.doc, from));
    }),
    new InputRule(/\$\$([^$\n]+)\$\$$/, (state, m, start, end) => {
      const $start = state.doc.resolve(start);
      if ($start.parent.type.spec.code) return null;
      const tex = (m[1] ?? "").trim();
      if (!tex) return null;
      return state.tr.replaceRangeWith(start, end, (N.math_inline as NodeType).create({ tex, html: "" }));
    }),
  ];
}

export function mathKeys(): Plugin {
  return new Plugin({
    props: {
      handleKeyDown(view, e) {
        const sel = view.state.selection;
        if (e.key !== "Enter" || !(sel instanceof NodeSelection)) return false;
        const name = sel.node.type.name;
        if (name !== "math_block" && name !== "math_inline") return false;
        const dom = view.nodeDOM(sel.from);
        if (!(dom instanceof HTMLElement)) return false;
        dom.dispatchEvent(new CustomEvent("math-edit"));
        return true;
      },
    },
  });
}
