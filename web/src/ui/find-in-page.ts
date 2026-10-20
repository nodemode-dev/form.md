import type { Node as PMNode } from "prosemirror-model";
import { type EditorState, TextSelection } from "prosemirror-state";
import { Decoration, DecorationSet, type EditorView } from "prosemirror-view";
import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import { editorHost } from "../extensions/blocks";
import { h, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { openLayer } from "./layers";

interface Hit {
  from: number;
  to: number;
}

const LIMIT = 5000;

function findHits(doc: PMNode, re: RegExp): Hit[] {
  const out: Hit[] = [];
  doc.descendants((node, pos) => {
    if (out.length >= LIMIT) return false;
    if (!node.isTextblock) return true;
    let text = "";
    const at: number[] = [];
    node.forEach((child, offset) => {
      const start = pos + 1 + offset;
      if (child.isText) {
        const s = child.text ?? "";
        for (let i = 0; i < s.length; i++) at.push(start + i);
        text += s;
      } else {
        at.push(start);
        text += "￼";
      }
    });
    re.lastIndex = 0;
    for (let m = re.exec(text); m && out.length < LIMIT; m = re.exec(text)) {
      if (!m[0]) {
        re.lastIndex++;
        continue;
      }
      const from = at[m.index];
      const last = at[m.index + m[0].length - 1];
      if (from !== undefined && last !== undefined) out.push({ from, to: last + 1 });
    }
    return false;
  });
  return out;
}

function chip(label: string): HTMLButtonElement {
  return h("button", { class: "chip-btn", attrs: { type: "button", "aria-pressed": "false" } }, h("span", { text: label }));
}

function iconBtn(icon: "chevron-up" | "chevron-down" | "close", label: string): HTMLButtonElement {
  return h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": label, "data-tip": label } }, iconNode(icon));
}

class FindBar {
  readonly el: HTMLElement;
  private readonly find: HTMLInputElement;
  private readonly repl: HTMLInputElement;
  private readonly count: HTMLElement;
  private readonly matchCase: HTMLButtonElement;
  private readonly wholeWords: HTMLButtonElement;
  private readonly replRow: HTMLElement;
  private readonly stepBtns: HTMLButtonElement[];
  private readonly replBtns: HTMLButtonElement[];
  private view: EditorView | null = null;
  private active = false;
  private hits: Hit[] = [];
  private cur = -1;
  private anchor = -1;
  private seek = false;
  private doc: PMNode | null = null;
  private key = "";
  private set: DecorationSet = DecorationSet.empty;
  private frame = 0;
  private closeLayer: (() => void) | null = null;
  private selectOnClose = true;
  private readonly deco = (state: EditorState): DecorationSet | null => this.decorations(state);

  constructor(private readonly engine: Engine) {
    this.find = h("input", { class: "input input--sm findbar__input", attrs: { type: "search", placeholder: t("Find"), "aria-label": t("Find"), spellcheck: "false", autocomplete: "off" } });
    this.repl = h("input", { class: "input input--sm findbar__input", attrs: { type: "text", placeholder: t("Replace with"), "aria-label": t("Replace with"), spellcheck: "false", autocomplete: "off" } });
    this.count = h("span", { class: "findbar__count", attrs: { "aria-live": "polite" } });
    const prev = iconBtn("chevron-up", t("Previous match"));
    const next = iconBtn("chevron-down", t("Next match"));
    const close = iconBtn("close", t("Close"));
    this.matchCase = chip(t("Match case"));
    this.wholeWords = chip(t("Whole words"));
    const one = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Replace") });
    const all = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Replace all") });
    this.stepBtns = [prev, next];
    this.replBtns = [one, all];
    const findRow = h("div", { class: "findbar__row" }, this.find, this.count, prev, next, close);
    this.replRow = h("div", { class: "findbar__row" }, this.repl, one, all);
    const opts = h("div", { class: "findbar__row findbar__opts" }, this.matchCase, this.wholeWords);
    this.el = h("div", { class: "findbar", attrs: { role: "search", "aria-label": t("Find and replace") } }, findRow, this.replRow, opts);
    this.el.hidden = true;
    this.find.addEventListener("input", () => this.requery());
    for (const b of [this.matchCase, this.wholeWords])
      b.addEventListener("click", () => {
        setAttr(b, "aria-pressed", String(b.getAttribute("aria-pressed") !== "true"));
        this.requery();
      });
    prev.addEventListener("click", () => this.step(-1));
    next.addEventListener("click", () => this.step(1));
    close.addEventListener("click", () => this.close());
    one.addEventListener("click", () => this.replaceOne());
    all.addEventListener("click", () => this.replaceAll());
    this.find.addEventListener("keydown", (e) => {
      if (e.key !== "Enter") return;
      e.preventDefault();
      this.step(e.shiftKey ? -1 : 1);
    });
    this.repl.addEventListener("keydown", (e) => {
      if (e.key !== "Enter") return;
      e.preventDefault();
      if (e.metaKey || e.ctrlKey) this.replaceAll();
      else this.replaceOne();
    });
    window.addEventListener("resize", () => this.active && this.place());
    engine.state.watch("route", () => this.active && this.close(false));
  }

  open(): void {
    const host = editorHost();
    const view = host?.view();
    if (!host || !view) return;
    if (this.view !== view) {
      this.view = view;
      view.setProps({ decorations: this.deco });
    }
    const wasActive = this.active;
    this.active = true;
    const sel = view.state.selection;
    const picked = sel.empty ? "" : view.state.doc.textBetween(sel.from, sel.to, "\n");
    if (picked && !picked.includes("\n") && picked.length < 200) this.find.value = picked;
    const writable = host.writable();
    if (this.replRow.hidden === writable) this.replRow.hidden = !writable;
    const mount = view.dom.closest<HTMLElement>("[data-ref='scroller']")?.parentElement ?? view.dom.parentElement;
    if (mount && this.el.parentElement !== mount) mount.append(this.el);
    this.place();
    this.el.hidden = false;
    if (!wasActive) {
      this.selectOnClose = true;
      this.closeLayer = openLayer({
        el: this.el,
        modal: false,
        dismissOnOutside: false,
        initialFocus: this.find,
        returnFocus: view.hasFocus() ? (view.dom as HTMLElement) : null,
        onClose: () => {
          this.closeLayer = null;
          this.deactivate();
        },
      });
    } else this.find.focus();
    this.find.select();
    this.requery();
  }

  close(select = true): void {
    this.selectOnClose = select;
    const c = this.closeLayer;
    this.closeLayer = null;
    if (c) c();
    else this.deactivate();
  }

  private deactivate(): void {
    if (!this.active) return;
    const view = this.view;
    const hit = this.hits[this.cur];
    this.active = false;
    this.el.hidden = true;
    cancelAnimationFrame(this.frame);
    this.frame = 0;
    this.doc = null;
    this.hits = [];
    this.cur = -1;
    if (!view) return;
    view.setProps({});
    if (this.selectOnClose && hit && hit.to <= view.state.doc.content.size) {
      view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, hit.from, hit.to)));
      view.focus();
    }
  }

  private place(): void {
    const mount = this.el.parentElement;
    const scroller = mount?.querySelector<HTMLElement>("[data-ref='scroller']");
    if (!scroller) return;
    this.el.style.insetBlockStart = `${scroller.offsetTop + 8}px`;
    this.el.style.insetInlineEnd = `${Math.max(0, scroller.offsetWidth - scroller.clientWidth) + 16}px`;
  }

  private regex(): RegExp | null {
    const f = this.find.value;
    if (!f) return null;
    const src = f.replace(/[.*+?^${}()|[\]\\/]/g, "\\$&");
    const whole = this.wholeWords.getAttribute("aria-pressed") === "true";
    const exact = this.matchCase.getAttribute("aria-pressed") === "true";
    return new RegExp(whole ? `(?<![\\p{L}\\p{N}_])${src}(?![\\p{L}\\p{N}_])` : src, `gu${exact ? "" : "i"}`);
  }

  private requery(): void {
    const view = this.view;
    if (!view || !this.active) return;
    this.anchor = view.state.selection.from;
    this.seek = true;
    this.doc = null;
    view.setProps({});
    this.paint();
  }

  private decorations(state: EditorState): DecorationSet | null {
    if (!this.active) return null;
    const re = this.regex();
    if (!re) {
      this.hits = [];
      this.cur = -1;
      this.schedulePaint();
      return null;
    }
    const key = `${re.source}|${re.flags}|${this.cur}`;
    if (state.doc === this.doc && key === this.key) return this.set;
    if (state.doc !== this.doc) {
      const before = this.hits[this.cur]?.from ?? this.anchor;
      this.hits = findHits(state.doc, re);
      const i = this.hits.findIndex((h) => h.from >= before);
      this.cur = !this.hits.length ? -1 : i >= 0 ? i : 0;
      if (this.seek && this.cur >= 0) requestAnimationFrame(() => this.reveal());
      this.seek = false;
      this.doc = state.doc;
    }
    this.key = `${re.source}|${re.flags}|${this.cur}`;
    this.set = DecorationSet.create(
      state.doc,
      this.hits.map((h, i) => Decoration.inline(h.from, h.to, { class: i === this.cur ? "find-hit find-hit--current" : "find-hit" })),
    );
    this.schedulePaint();
    return this.set;
  }

  private schedulePaint(): void {
    if (this.frame) return;
    this.frame = requestAnimationFrame(() => {
      this.frame = 0;
      this.paint();
    });
  }

  private paint(): void {
    const n = this.hits.length;
    const q = this.find.value;
    setText(this.count, !q ? "" : !n ? t("No results") : t("{n} of {total}", { n: this.cur + 1, total: n >= LIMIT ? `${LIMIT}+` : n }));
    this.count.classList.toggle("is-none", !!q && !n);
    for (const b of [...this.stepBtns, ...this.replBtns]) if (b.disabled !== !n) b.disabled = !n;
  }

  private step(dir: 1 | -1): void {
    const view = this.view;
    const n = this.hits.length;
    if (!view || !n) return;
    this.cur = this.cur < 0 ? (dir > 0 ? 0 : n - 1) : (this.cur + dir + n) % n;
    view.setProps({});
    this.paint();
    this.reveal();
  }

  private reveal(): void {
    const view = this.view;
    const hit = this.hits[this.cur];
    if (!view || !hit) return;
    let c: { top: number; bottom: number };
    try {
      c = view.coordsAtPos(hit.from);
    } catch {
      return;
    }
    const scroller = view.dom.closest<HTMLElement>("[data-ref='scroller']");
    if (!scroller) return;
    const r = scroller.getBoundingClientRect();
    const bar = this.el.getBoundingClientRect();
    const top = Math.max(r.top, bar.bottom) + 16;
    if (c.top >= top && c.bottom <= r.bottom - 24) return;
    const smooth = !matchMedia("(prefers-reduced-motion: reduce)").matches;
    scroller.scrollTo({ top: scroller.scrollTop + c.top - r.top - r.height / 3, behavior: smooth ? "smooth" : "auto" });
  }

  private writable(): boolean {
    return !!editorHost()?.writable() && !this.replRow.hidden;
  }

  private replaceOne(): void {
    const view = this.view;
    if (!view || !this.writable()) return;
    const hit = this.hits[this.cur] ?? this.hits[0];
    if (!hit) return;
    const text = this.repl.value;
    this.anchor = hit.from + text.length;
    this.cur = -1;
    this.seek = true;
    const tr = view.state.tr;
    if (text) tr.insertText(text, hit.from, hit.to);
    else tr.delete(hit.from, hit.to);
    view.dispatch(tr);
  }

  private replaceAll(): void {
    const view = this.view;
    if (!view || !this.writable() || !this.hits.length) return;
    const text = this.repl.value;
    const tr = view.state.tr;
    for (let i = this.hits.length - 1; i >= 0; i--) {
      const hit = this.hits[i];
      if (!hit) continue;
      if (text) tr.insertText(text, hit.from, hit.to);
      else tr.delete(hit.from, hit.to);
    }
    this.cur = -1;
    this.anchor = 0;
    this.seek = false;
    view.dispatch(tr);
  }
}

let bar: FindBar | null = null;

export function openFind(engine: Engine): void {
  bar ??= new FindBar(engine);
  bar.open();
}
