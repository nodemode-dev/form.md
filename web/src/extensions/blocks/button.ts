import type { Node as PMNode } from "prosemirror-model";
import type { EditorView, NodeView } from "prosemirror-view";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import type { AutomationAction } from "../../engine/types";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import type { EditorCtx } from "./context";
import { registerNodeView } from "./nodeviews";

type Runner = typeof import("./buttonrun");
let runner: Promise<Runner> | null = null;

export function buttonRunner(): Promise<Runner> {
  runner ??= import("./buttonrun").catch((e: unknown) => {
    runner = null;
    throw e;
  });
  return runner;
}

export function editButton(ctx: EditorCtx, view: EditorView, getPos: () => number | undefined): void {
  void buttonRunner().then((m) => m.editButton(ctx, view, getPos), ctx.engine.reportError);
}

export function newButton(label = "", actions: AutomationAction[] = []): PMNode {
  return schema.nodes.button!.create({ id: newBlockId(), label, actions: JSON.stringify(actions) });
}

export function readActions(raw: unknown): AutomationAction[] {
  try {
    const v = JSON.parse(String(raw ?? "[]"));
    return Array.isArray(v) ? (v as AutomationAction[]) : [];
  } catch {
    return [];
  }
}

class ButtonView implements NodeView {
  readonly dom: HTMLElement;
  private btn: HTMLButtonElement;
  private label: HTMLElement;
  private edit: HTMLButtonElement;
  private running = false;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: () => number | undefined,
    private readonly ctx: EditorCtx,
  ) {
    this.label = h("span", { class: "docbtn__label" });
    this.btn = h("button", { class: "docbtn__btn", attrs: { type: "button" } }, iconNode("button"), this.label);
    this.edit = h("button", { class: "icon-btn icon-btn--sm docbtn__edit", attrs: { type: "button", "aria-label": t("Edit button"), "data-tip": t("Edit button") } }, iconNode("pencil"));
    this.dom = h("div", { class: "docbtn", attrs: { contenteditable: "false" } }, this.btn, this.edit);
    this.btn.addEventListener("click", () => void this.run());
    this.dom.addEventListener("pointerenter", () => void buttonRunner().catch(() => undefined), { once: true });
    this.edit.addEventListener("click", () => editButton(this.ctx, this.view, this.getPos));
    this.sync();
  }

  private sync(): void {
    setText(this.label, String(this.node.attrs.label || t("Button")));
    setAttr(this.dom, "data-block", this.node.attrs.id ? String(this.node.attrs.id) : null);
    this.edit.hidden = !this.ctx.writable();
    setAttr(this.dom, "data-empty", readActions(this.node.attrs.actions).length ? null : "true");
  }

  private async run(): Promise<void> {
    if (this.running) return;
    this.running = true;
    setAttr(this.dom, "aria-busy", "true");
    try {
      const m = await buttonRunner();
      await m.runButton(this.ctx, this.view, this.getPos, this.node);
    } catch (e) {
      this.ctx.engine.reportError(e);
    } finally {
      this.running = false;
      setAttr(this.dom, "aria-busy", null);
    }
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.sync();
    return true;
  }

  stopEvent(e: Event): boolean {
    return e.target instanceof Element && !!e.target.closest(".docbtn__btn, .docbtn__edit");
  }

  ignoreMutation(): boolean {
    return true;
  }
}

registerNodeView("button", (ctx) => (node, view, getPos) => new ButtonView(node, view, getPos, ctx));
