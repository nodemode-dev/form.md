import { type EditorState, TextSelection } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";
import { schema, TONES } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { byId, h, setAttr } from "../../ui/dom";
import { anchorToRect, type Rect } from "../../ui/floating";
import { type IconName, iconNode } from "../../ui/icons";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { openAI } from "./ai";
import { type Align, alignAt, setAlign } from "./align";
import { CATALOG } from "./catalog";
import { markActive, markCommand, setMark, turnInto } from "./commands";
import { commentUI } from "./comments";
import type { EditorCtx } from "./context";
import { openPrompt } from "./popover";

interface Tool {
  id: string;
  icon: IconName;
  label: () => string;
  mark?: string;
  run: (view: EditorView, btn: HTMLButtonElement) => void;
}

function selectionRect(view: EditorView): Rect | null {
  const { from, to } = view.state.selection;
  try {
    const a = view.coordsAtPos(from);
    const b = view.coordsAtPos(to);
    const left = Math.min(a.left, b.left);
    const right = Math.max(a.right, b.right);
    const top = Math.min(a.top, b.top);
    const bottom = Math.max(a.bottom, b.bottom);
    if (a.top !== b.top) {
      const box = view.dom.getBoundingClientRect();
      return { x: box.left, y: top, width: box.width, height: bottom - top };
    }
    return { x: left, y: top, width: Math.max(1, right - left), height: bottom - top };
  } catch {
    return null;
  }
}

export class FormatBar {
  private el: HTMLElement | null = null;
  private place: { update: () => void; release: () => void } | null = null;
  private buttons = new Map<string, HTMLButtonElement>();
  private pointerDown = false;
  private readonly tools: Tool[];

  constructor(private readonly ctx: EditorCtx) {
    const mark = (id: string, icon: IconName, label: () => string, name: string): Tool => ({
      id,
      icon,
      label,
      mark: name,
      run: (view) => {
        markCommand(name)(view.state, view.dispatch, view);
        view.focus();
      },
    });
    this.tools = [
      {
        id: "ai",
        icon: "sparkle",
        label: () => t("Ask AI"),
        run: (view) => {
          this.hide();
          openAI(this.ctx, view);
        },
      },
      {
        id: "turn",
        icon: "turn",
        label: () => t("Turn into"),
        run: (view, btn) => {
          const entries: MenuEntry[] = CATALOG.filter((c) => c.turn).map((c) => ({
            label: c.label(),
            icon: c.icon,
            run: () => {
              turnInto(view, c.turn as Parameters<typeof turnInto>[1]);
              view.focus();
            },
          }));
          openMenu(btn, entries, { side: "bottom", align: "start", label: t("Turn into") });
        },
      },
      mark("bold", "bold", () => t("Bold"), "strong"),
      mark("italic", "italic", () => t("Italic"), "em"),
      mark("underline", "underline", () => t("Underline"), "underline"),
      mark("strike", "strike", () => t("Strikethrough"), "strike"),
      mark("code", "code", () => t("Inline code"), "code"),
      {
        id: "link",
        icon: "link",
        label: () => t("Link"),
        mark: "link",
        run: (view) => this.editLink(view),
      },
      {
        id: "comment",
        icon: "comment-plus",
        label: () => t("Comment"),
        run: (view) => {
          const { from, to } = view.state.selection;
          this.hide();
          commentUI(view)?.startThread(from, to, "text");
        },
      },
      {
        id: "align",
        icon: "align",
        label: () => t("Text alignment"),
        run: (view, btn) => {
          const cur = alignAt(view);
          const opts: [Align, string][] = [["", "Left"], ["center", "Centred"], ["right", "Right"], ["justify", "Justified"]];
          openMenu(btn, opts.map(([v, label]) => ({ label: t(label), checked: cur === v, run: () => (setAlign(view, v), view.focus()) })), { side: "bottom", align: "start", label: t("Text alignment") });
        },
      },
      {
        id: "color",
        icon: "text-color",
        label: () => t("Colour"),
        run: (view, btn) => {
          const dot = (c: string) => h("span", { class: "tag-dot", attrs: { "data-color": c || "none" } });
          const name = (c: string) => (c ? c.charAt(0).toUpperCase() + c.slice(1) : t("Default"));
          const entries: MenuEntry[] = [{ kind: "heading", label: t("Text") }];
          for (const c of TONES) entries.push({ label: name(c), lead: dot(c), run: () => setMark(view, "color", c ? { value: c } : null) });
          entries.push({ kind: "heading", label: t("Highlight") });
          for (const c of TONES) entries.push({ label: name(c), lead: dot(c), run: () => setMark(view, "highlight", c ? { value: c } : null) });
          openMenu(btn, entries, { side: "bottom", align: "start", label: t("Colour") });
        },
      },
    ];
  }

  editLink(view: EditorView): void {
    const { from, to, empty } = view.state.selection;
    if (empty) return;
    const type = schema.marks.link;
    if (!type) return;
    let current = "";
    view.state.doc.nodesBetween(from, to, (n) => {
      const m = type.isInSet(n.marks);
      if (m && !current) current = String(m.attrs.href);
    });
    if (current) {
      view.dispatch(view.state.tr.removeMark(from, to, type));
      view.focus();
      return;
    }
    openPrompt({
      label: t("Link"),
      placeholder: t("Paste a link or a page path"),
      action: t("Link"),
      rect: () => selectionRect(view),
      onSubmit: (raw) => {
        let href = raw.trim();
        if (!/^(https?:\/\/|mailto:|\/|#)/i.test(href)) href = `https://${href}`;
        view.dispatch(view.state.tr.addMark(from, to, type.create({ href })));
      },
      onClose: () => view.focus(),
    });
  }

  attach(view: EditorView): void {
    view.dom.addEventListener("mousedown", () => {
      this.pointerDown = true;
      this.hide();
    });
    window.addEventListener("mouseup", () => {
      if (!this.pointerDown) return;
      this.pointerDown = false;
      requestAnimationFrame(() => {
        const v = this.ctx.view();
        if (v) this.update(v);
      });
    });
    view.dom.addEventListener("focusout", (e) => {
      const to = e.relatedTarget as Node | null;
      if (to && this.el?.contains(to)) return;
      setTimeout(() => {
        const v = this.ctx.view();
        if (v && !v.hasFocus() && !this.el?.contains(document.activeElement)) this.hide();
      }, 0);
    });
  }

  private wanted(view: EditorView, state: EditorState): boolean {
    const sel = state.selection;
    if (this.pointerDown || sel.empty || !(sel instanceof TextSelection) || !view.editable || !view.hasFocus()) return false;
    if (sel.$from.parent.type.spec.code || sel.$to.parent.type.spec.code) return false;
    return state.doc.textBetween(sel.from, sel.to, " ", " ").trim().length > 0;
  }

  update(view: EditorView): void {
    if (!this.wanted(view, view.state)) return this.hide();
    if (!this.el) this.build(view);
    const suggesting = this.ctx.suggesting();
    for (const tool of this.tools) {
      const b = this.buttons.get(tool.id);
      if (b) b.hidden = suggesting && tool.id !== "comment";
      if (!tool.mark) continue;
      if (b) setAttr(b, "aria-pressed", markActive(view.state, tool.mark) ? "true" : "false");
    }
    const sep = this.el?.querySelector<HTMLElement>(".formatbar__sep");
    if (sep) sep.hidden = suggesting;
    this.place?.update();
  }

  private build(view: EditorView): void {
    const el = h("div", { class: "formatbar", attrs: { role: "toolbar", "aria-label": t("Formatting") } });
    for (const tool of this.tools) {
      const b = h("button", { class: "formatbar__btn", attrs: { type: "button", "aria-label": tool.label(), "data-tip": tool.label() } }, iconNode(tool.icon));
      if (tool.mark) setAttr(b, "aria-pressed", "false");
      b.addEventListener("mousedown", (e) => e.preventDefault());
      b.addEventListener("click", () => {
        const v = this.ctx.view();
        if (v) tool.run(v, b);
      });
      this.buttons.set(tool.id, b);
      el.append(b);
      if (tool.id === "turn") el.append(h("span", { class: "formatbar__sep" }));
    }
    byId("overlay-root").append(el);
    this.el = el;
    this.place = anchorToRect(el, () => selectionRect(this.ctx.view() ?? view), { side: "top", align: "center", offset: 8 });
    requestAnimationFrame(() => el.classList.add("is-visible"));
  }

  hide(): void {
    this.place?.release();
    this.place = null;
    this.el?.remove();
    this.el = null;
    this.buttons.clear();
  }
}
