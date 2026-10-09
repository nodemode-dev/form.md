import { Fragment, type Node as PMNode } from "prosemirror-model";
import type { EditorView } from "prosemirror-view";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import type { AutomationAction } from "../../engine/types";
import { confirmDialog, simpleDialog } from "../../ui/dialog";
import { h } from "../../ui/dom";
import { actions as formActions, button as formButton, field, input } from "../../ui/form";
import { toast } from "../../ui/toast";
import { automationApi } from "../automation/api";
import { readActions } from "./button";
import type { EditorCtx } from "./context";
import { markdownReady, parseMarkdownSlice } from "./markdown";

const SERVER = ["edit", "add", "notify", "mail", "webhook"];

function fill(tpl: string, ctx: EditorCtx): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  const date = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
  const me = ctx.engine.state.get("me")?.user.name ?? "";
  const title = ctx.engine.pages.get(ctx.pageId())?.title ?? "";
  return tpl.replace(/\{\{\s*([^{}]+?)\s*\}\}/g, (m, k: string) => {
    switch (k.toLowerCase()) {
      case "date":
      case "today":
        return date;
      case "now":
      case "time":
        return `${date} ${pad(now.getHours())}:${pad(now.getMinutes())}`;
      case "user":
      case "me":
        return me;
      case "title":
      case "page":
        return title;
    }
    return m;
  });
}

export function editButton(ctx: EditorCtx, view: EditorView, getPos: () => number | undefined): void {
  const pos = getPos();
  const node = pos === undefined ? null : view.state.doc.nodeAt(pos);
  if (!node || node.type.name !== "button") return;
  const page = ctx.engine.pages.get(ctx.pageId());
  simpleDialog(
    t("Edit button"),
    (body, close) => {
      const label = input({ value: String(node.attrs.label ?? ""), placeholder: t("Button label"), maxlength: 80 });
      const slot = h("div", { class: "acts__slot" });
      const save = formButton(t("Save"), "primary", undefined, "submit");
      const cancel = formButton(t("Cancel"), "ghost");
      cancel.addEventListener("click", close);
      let read: (() => AutomationAction[]) | null = null;
      void import("../automation/actions").then(async (m) => {
        const ed = await m.actionEditor({ engine: ctx.engine, workspaceId: ctx.workspaceId(), ...(page?.row ? { collectionId: page.parent_id } : {}), automation: false }, readActions(node.attrs.actions), ["insert", "add", "edit", "notify", "mail", "webhook", "open", "confirm"]);
        read = ed.value;
        slot.append(ed.el);
      });
      const form = h("form", { class: "form-grid automation-form", attrs: { novalidate: "" } }, field(t("Label"), label), field(t("When clicked"), slot, t("Actions run in order, as the person who clicks. {{date}}, {{user}} and {{title}} are filled in.")), formActions(cancel, save));
      form.addEventListener("submit", (e) => {
        e.preventDefault();
        const at = getPos();
        const cur = at === undefined ? null : view.state.doc.nodeAt(at);
        if (at === undefined || !cur || cur.type.name !== "button") return close();
        view.dispatch(view.state.tr.setNodeMarkup(at, undefined, { ...cur.attrs, label: label.value.trim(), actions: JSON.stringify(read ? read() : readActions(cur.attrs.actions)) }));
        close();
      });
      body.append(form);
      return label;
    },
    undefined,
    "md",
  );
}

async function insert(ctx: EditorCtx, view: EditorView, getPos: () => number | undefined, a: AutomationAction): Promise<void> {
  if (!ctx.writable()) {
    toast({ id: "button-insert", tone: "info", message: t("You can view this page but not add to it.") });
    return;
  }
  const text = fill(a.markdown ?? "", ctx);
  if (!text.trim()) return;
  const ready = await markdownReady();
  const pos = getPos();
  const node = pos === undefined ? null : view.state.doc.nodeAt(pos);
  if (pos === undefined || !node) return;
  const at = a.where === "above" ? pos : pos + node.nodeSize;
  const nodes: PMNode[] = [];
  if (ready) parseMarkdownSlice(text, view.state.doc.resolve(at)).content.forEach((n) => void (n.isBlock && nodes.push(n)));
  else for (const line of text.split(/\n{2,}/)) nodes.push(schema.nodes.paragraph!.create({ id: newBlockId() }, line.trim() ? schema.text(line.trim()) : null));
  if (!nodes.length) return;
  view.dispatch(view.state.tr.insert(at, Fragment.fromArray(nodes)));
}

export async function runButton(ctx: EditorCtx, view: EditorView, getPos: () => number | undefined, node: PMNode): Promise<void> {
  const list = readActions(node.attrs.actions);
  const name = String(node.attrs.label || t("Button"));
  if (!list.length) {
    if (ctx.writable()) editButton(ctx, view, getPos);
    return;
  }
  const ask = list.find((a) => a.kind === "confirm");
  if (ask && !(await confirmDialog({ title: fill(ask.message || t("Run {name}?", { name }), ctx), body: t("{n} actions will run.", { n: list.filter((a) => a.kind !== "confirm").length }), confirm: name }))) return;
  for (const a of list) if (a.kind === "insert") await insert(ctx, view, getPos, a);
  const server = list.filter((a) => SERVER.includes(a.kind));
  if (server.length) {
    const r = await automationApi(ctx.engine).button({ page_id: ctx.pageId(), name, actions: server });
    const failed = r.outcome.failed ?? [];
    const done = r.outcome.done ?? [];
    toast({ id: "button-run", tone: failed.length ? "danger" : "success", message: failed.length ? failed.join("; ") : done.join("; ") || t("Done") });
  }
  const open = list.find((a) => a.kind === "open");
  if (open?.page) ctx.engine.openPage(open.page);
  else if (open?.url && /^https?:\/\//i.test(open.url)) window.open(open.url, "_blank", "noopener,noreferrer");
}
