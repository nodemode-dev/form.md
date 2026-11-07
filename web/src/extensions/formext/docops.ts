import { DOMParser as PMDOMParser, Fragment, type Node as PMNode } from "prosemirror-model";
import { TextSelection, type Transaction } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";
import { redo, undo, yUndoPluginKey } from "y-prosemirror";
import type { Engine } from "../../engine";
import type { CommandParam } from "../../engine/commands";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import { editorHost } from "../blocks";
import { MarkdownWriter, markdownReady, markdownToDOM } from "../blocks/markdown";
import { ERR, HostError } from "./host-bridge";

type Args = Record<string, unknown>;
type Op = { command: string; args?: unknown };

interface DocOp {
  id: string;
  label: () => string;
  description: string;
  params: Record<string, CommandParam>;
  markdown?: boolean;
  apply: (tr: Transaction, args: Args) => unknown;
}

const TEXT_CAP = 4000;
const SELECTION_CAP = 2000;
const MARKS = ["strong", "em", "underline", "strike", "code", "highlight"];
const WHERE = ["end", "start", "cursor", "after"];

function fail(message: string): never {
  throw new HostError(ERR.REFUSED, message);
}

function findBlock(doc: PMNode, id: string): { node: PMNode; pos: number } {
  let hit: { node: PMNode; pos: number } | null = null;
  doc.descendants((node, pos) => {
    if (hit) return false;
    if (node.isBlock && node.attrs.id === id) {
      hit = { node, pos };
      return false;
    }
    return true;
  });
  if (!hit) fail(`no block ${id} on this page`);
  return hit;
}

function withIds(frag: Fragment): { frag: Fragment; ids: string[] } {
  const ids: string[] = [];
  const nodes: PMNode[] = [];
  frag.forEach((n) => {
    if ("id" in (n.type.spec.attrs ?? {})) {
      const id = String(n.attrs.id ?? "") || newBlockId();
      ids.push(id);
      nodes.push(n.attrs.id === id ? n : n.type.create({ ...n.attrs, id }, n.content, n.marks));
    } else nodes.push(n);
  });
  return { frag: Fragment.from(nodes), ids };
}

function fromMarkdown(text: string): Fragment {
  return PMDOMParser.fromSchema(schema).parse(markdownToDOM(text)).content;
}

function fromText(text: string): Fragment {
  const paras = text.replace(/\r\n?/g, "\n").split(/\n{2,}/);
  return Fragment.from(paras.map((p) => schema.nodes.paragraph!.create(null, p ? schema.text(p) : null)));
}

function blank(doc: PMNode): boolean {
  return doc.childCount === 1 && doc.firstChild?.type.name === "paragraph" && doc.firstChild.content.size === 0;
}

function insertAt(tr: Transaction, args: Args, content: Fragment): { blocks: string[] } {
  const { frag, ids } = withIds(content);
  if (!frag.size) return { blocks: [] };
  const at = String(args.at ?? (args.block ? "after" : "end"));
  if (blank(tr.doc) && at !== "after") {
    tr.replaceWith(0, tr.doc.content.size, frag);
    return { blocks: ids };
  }
  let pos = tr.doc.content.size;
  if (at === "start") pos = 0;
  else if (at === "cursor") {
    const $from = tr.selection.$from;
    pos = $from.depth >= 1 ? $from.after(1) : tr.doc.content.size;
  } else if (at === "after") {
    if (typeof args.block !== "string") fail("block is required with at: after");
    const { node, pos: start } = findBlock(tr.doc, args.block);
    pos = start + node.nodeSize;
  }
  tr.insert(pos, frag);
  return { blocks: ids };
}

function textPos(doc: PMNode, start: number, node: PMNode, offset: number): number {
  let lo = start + 1;
  let hi = start + node.nodeSize - 1;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (doc.textBetween(start, mid, "\n").length < offset) lo = mid + 1;
    else hi = mid;
  }
  return lo;
}

function matches(doc: PMNode, find: string, all: boolean, caseSensitive: boolean): { from: number; to: number }[] {
  const out: { from: number; to: number }[] = [];
  if (!find) fail("find must not be empty");
  const needle = caseSensitive ? find : find.toLowerCase();
  doc.descendants((node, pos) => {
    if (!all && out.length) return false;
    if (!node.isText || !node.text) return true;
    const hay = caseSensitive ? node.text : node.text.toLowerCase();
    let i = hay.indexOf(needle);
    while (i >= 0) {
      out.push({ from: pos + i, to: pos + i + find.length });
      if (!all) break;
      i = hay.indexOf(needle, i + find.length);
    }
    return false;
  });
  return out;
}

const str = (description: string, required = false): CommandParam => ({ type: "string", description, ...(required ? { required: true } : {}) });

export const DOC_OPS: DocOp[] = [
  {
    id: "doc.insertMarkdown",
    label: () => t("Insert Markdown"),
    description: "Inserts Markdown as blocks into the open page: at the end (default), the start, after the block holding the cursor, or after a given block id. Returns the new block ids.",
    params: { markdown: str("Markdown to insert", true), at: { type: "string", enum: WHERE, description: "Where to insert" }, block: str("Block id, with at: after") },
    markdown: true,
    apply: (tr, a) => insertAt(tr, a, fromMarkdown(String(a.markdown))),
  },
  {
    id: "doc.insertText",
    label: () => t("Insert text"),
    description: "Inserts plain text into the open page. With at: cursor and text with no blank line, inserts it inline at the selection in its text block, replacing selected text. Otherwise inserts it as paragraphs (split on blank lines) where doc.insertMarkdown would.",
    params: { text: str("Text to insert", true), at: { type: "string", enum: WHERE, description: "Where to insert" }, block: str("Block id, with at: after") },
    apply: (tr, a) => {
      const text = String(a.text);
      const { $from, from, to } = tr.selection;
      if (a.at === "cursor" && text && !/\n\s*\n/.test(text) && $from.parent.isTextblock && $from.depth >= 1) {
        tr.replaceWith(from, to, schema.text(text.replace(/\s*\n\s*/g, " ")));
        return { blocks: [], inline: String(tr.doc.child($from.index(0)).attrs.id ?? "") };
      }
      return insertAt(tr, a, fromText(text));
    },
  },
  {
    id: "doc.replaceBlock",
    label: () => t("Replace a block"),
    description: "Replaces one block of the open page, found by id, with blocks written as Markdown. Returns the new block ids.",
    params: { block: str("Block id", true), markdown: str("Markdown for the new blocks", true) },
    markdown: true,
    apply: (tr, a) => {
      const { node, pos } = findBlock(tr.doc, String(a.block));
      const { frag, ids } = withIds(fromMarkdown(String(a.markdown)));
      if (!frag.size) fail("markdown is empty");
      tr.replaceWith(pos, pos + node.nodeSize, frag);
      return { blocks: ids };
    },
  },
  {
    id: "doc.setBlockText",
    label: () => t("Set a block's text"),
    description: "Replaces the text of one text block (paragraph, heading, code and the like) of the open page, keeping its type and id.",
    params: { block: str("Block id", true), text: { type: "string", required: true, description: "The new text" } },
    apply: (tr, a) => {
      const { node, pos } = findBlock(tr.doc, String(a.block));
      if (!node.isTextblock) fail(`block ${String(a.block)} is a ${node.type.name}, not a text block`);
      const text = String(a.text);
      tr.replaceWith(pos + 1, pos + node.nodeSize - 1, text ? schema.text(text) : Fragment.empty);
      return { block: a.block };
    },
  },
  {
    id: "doc.deleteBlock",
    label: () => t("Delete a block"),
    description: "Deletes one block of the open page, found by id.",
    params: { block: str("Block id", true) },
    apply: (tr, a) => {
      const { node, pos } = findBlock(tr.doc, String(a.block));
      if (tr.doc.childCount === 1 && pos === 0) tr.replaceWith(0, tr.doc.content.size, schema.nodes.paragraph!.create({ id: newBlockId() }));
      else tr.delete(pos, pos + node.nodeSize);
      return { deleted: a.block };
    },
  },
  {
    id: "doc.moveBlock",
    label: () => t("Move a block"),
    description: "Moves one top-level block of the open page to a position in the page (0 is first).",
    params: { block: str("Block id", true), index: { type: "integer", required: true, min: 0, description: "New position, 0 is first" } },
    apply: (tr, a) => {
      let from = -1;
      let start = 0;
      tr.doc.forEach((n, offset, i) => {
        if (n.attrs.id === a.block) {
          from = i;
          start = offset;
        }
      });
      if (from < 0) fail(`no top-level block ${String(a.block)} on this page`);
      const node = tr.doc.child(from);
      tr.delete(start, start + node.nodeSize);
      const index = Math.min(Number(a.index), tr.doc.childCount);
      let pos = 0;
      for (let i = 0; i < index; i++) pos += tr.doc.child(i).nodeSize;
      tr.insert(pos, node);
      return { block: a.block, index };
    },
  },
  {
    id: "doc.replaceText",
    label: () => t("Replace text"),
    description: "Finds text in the open page and replaces it, every match (default) or the first one. Matches within one run of text; case-insensitive unless caseSensitive.",
    params: { find: str("Text to find", true), replace: { type: "string", required: true, description: "Replacement" }, all: { type: "boolean", description: "Every match (default true)" }, caseSensitive: { type: "boolean", description: "Match case (default false)" } },
    apply: (tr, a) => {
      const hits = matches(tr.doc, String(a.find), a.all !== false, a.caseSensitive === true);
      for (const m of [...hits].reverse()) {
        if (a.replace) tr.insertText(String(a.replace), m.from, m.to);
        else tr.delete(m.from, m.to);
      }
      return { replaced: hits.length };
    },
  },
  {
    id: "doc.markText",
    label: () => t("Format text"),
    description: "Finds text in the open page and adds an inline mark to it: strong, em, underline, strike, code or highlight.",
    params: { find: str("Text to find", true), mark: { type: "string", required: true, enum: MARKS, description: "The mark" }, all: { type: "boolean", description: "Every match (default true)" } },
    apply: (tr, a) => {
      const type = schema.marks[String(a.mark)];
      if (!type) fail(`no mark ${String(a.mark)}`);
      const hits = matches(tr.doc, String(a.find), a.all !== false, false);
      for (const m of hits) tr.addMark(m.from, m.to, type.create());
      return { marked: hits.length };
    },
  },
  {
    id: "doc.select",
    label: () => t("Select in a block"),
    description: "Moves the selection of the open page into a block, optionally from and to offsets in the block's text.",
    params: { block: str("Block id", true), from: { type: "integer", min: 0, description: "Start offset" }, to: { type: "integer", min: 0, description: "End offset" } },
    apply: (tr, a) => {
      const { node, pos } = findBlock(tr.doc, String(a.block));
      const from = textPos(tr.doc, pos, node, Number(a.from ?? 0));
      const to = a.to === undefined ? from : textPos(tr.doc, pos, node, Number(a.to));
      tr.setSelection(TextSelection.create(tr.doc, from, Math.max(from, to)));
      return { block: a.block };
    },
  },
];

const BY_ID = new Map(DOC_OPS.map((o) => [o.id, o]));

function checkArgs(op: DocOp, args: Args): void {
  for (const [name, p] of Object.entries(op.params)) {
    const v = args[name];
    if (v === undefined) {
      if (p.required) throw new HostError(ERR.INVALID, `${op.id}: ${name} is required`);
      continue;
    }
    const kind = Array.isArray(v) ? "array" : typeof v;
    const ok = p.type === "integer" ? Number.isInteger(v) : p.type === kind;
    if (!ok) throw new HostError(ERR.INVALID, `${op.id}: ${name} must be ${p.type}`);
    if (p.enum && !p.enum.includes(v as string)) throw new HostError(ERR.INVALID, `${op.id}: ${name} must be one of ${p.enum.join(", ")}`);
    if (typeof v === "number" && p.min !== undefined && v < p.min) throw new HostError(ERR.INVALID, `${op.id}: ${name} must be at least ${p.min}`);
  }
}

function liveView(): EditorView {
  const host = editorHost();
  const view = host?.view();
  if (!host || !view || !host.current()) fail("no page is open in the editor");
  if (!host.writable()) fail("the open page is not editable here: it is read-only for you, or in suggesting or viewing mode");
  return view;
}

export async function applyOps(label: string, ops: Op[]): Promise<{ applied: number; results: unknown[] }> {
  const steps = ops.map((o) => {
    const op = BY_ID.get(o.command);
    if (!op) throw new HostError(ERR.REFUSED, `${o.command} is not a document command; document.apply takes ${DOC_OPS.map((d) => d.id).join(", ")}`);
    const args = (o.args && typeof o.args === "object" ? o.args : {}) as Args;
    checkArgs(op, args);
    return { op, args };
  });
  if (steps.some((s) => s.op.markdown) && !(await markdownReady())) fail("the Markdown reader could not be loaded");
  const view = liveView();
  const tr = view.state.tr;
  const results: unknown[] = [];
  for (const { op, args } of steps) {
    try {
      results.push(op.apply(tr, args));
    } catch (e) {
      if (e instanceof HostError) throw new HostError(e.code, `${op.id}: ${e.message}`);
      throw new HostError(ERR.REFUSED, `${op.id}: ${e instanceof Error ? e.message : String(e)}`);
    }
  }
  if (tr.docChanged || tr.selectionSet) {
    const um = yUndoPluginKey.getState(view.state)?.undoManager;
    um?.stopCapturing();
    view.dispatch(tr.setMeta("formext", label));
    um?.stopCapturing();
  }
  return { applied: steps.length, results };
}

export function readDocument(engine: Engine, what = "all"): unknown {
  const host = editorHost();
  const view = host?.view();
  const id = host?.current() ?? "";
  const route = engine.state.get("route");
  const detail = engine.state.get("detail");
  const page = id ? (engine.pages.get(id) ?? (detail && detail.id === id ? detail : undefined)) : route.name === "page" ? (engine.pages.get(route.id) ?? (detail && detail.id === route.id ? detail : undefined)) : undefined;
  const out: Record<string, unknown> = {
    page: page ? { id: page.id, title: page.title, kind: page.kind, icon: page.icon, workspace_id: page.workspace_id, url: `/p/${page.id}` } : null,
    writable: !!host?.writable() && !!id,
    mode: id ? (host?.editMode() ?? "view") : "view",
  };
  if (!page) out.route = route.name;
  const doc = view && id ? view.state.doc : null;
  const want = (k: string) => what === "all" || what === k;
  if (want("blocks")) {
    const blocks: unknown[] = [];
    doc?.forEach((n) => {
      const b: Record<string, unknown> = { id: String(n.attrs.id ?? ""), type: n.type.name, text: n.textBetween(0, n.content.size, "\n", "\n").slice(0, TEXT_CAP) };
      if (n.type.name === "heading") b.level = n.attrs.level;
      if (n.type.name === "code_block") b.language = n.attrs.language;
      if (n.type.name === "task_list") {
        const checked: boolean[] = [];
        n.forEach((item) => checked.push(item.attrs.checked === true));
        b.checked = checked;
      }
      blocks.push(b);
    });
    out.blocks = blocks;
  }
  if (want("markdown")) out.markdown = doc ? new MarkdownWriter((pid) => engine.pages.get(pid)?.title ?? "").doc(doc.content) : "";
  if (want("selection")) out.selection = doc && view ? selectionOf(view) : null;
  if (want("outline")) {
    const outline: unknown[] = [];
    doc?.forEach((n) => {
      if (n.type.name === "heading") outline.push({ id: String(n.attrs.id ?? ""), level: n.attrs.level, text: n.textContent });
    });
    out.outline = outline;
  }
  return out;
}

export function selectionOf(view: EditorView): { block: string; from: number; to: number; empty: boolean; text: string } | null {
  const { selection, doc } = view.state;
  const $from = selection.$from;
  if ($from.depth < 1) return null;
  const start = $from.before(1);
  const node = doc.child($from.index(0));
  const from = doc.textBetween(start, selection.from, "\n").length;
  const to = from + doc.textBetween(selection.from, selection.to, "\n").length;
  return { block: String(node.attrs.id ?? ""), from, to, empty: selection.empty, text: doc.textBetween(selection.from, selection.to, "\n").slice(0, SELECTION_CAP) };
}

export function registerDocCommands(engine: Engine): Unsubscribe[] {
  const group = () => t("Document");
  const enabled = () => !!editorHost()?.current() && !!editorHost()?.writable();
  const offs = DOC_OPS.map((op) =>
    engine.commands.register({
      id: op.id,
      label: op.label,
      group,
      description: `${op.description} One undo step.`,
      params: op.params,
      mutates: true,
      hidden: () => true,
      enabled,
      run: async (params) => (await applyOps(op.label(), [{ command: op.id, args: params ?? {} }])).results[0],
    }),
  );
  const step = (fn: typeof undo) => () => {
    const view = liveView();
    yUndoPluginKey.getState(view.state)?.undoManager?.stopCapturing();
    return { done: fn(view.state) };
  };
  offs.push(
    engine.commands.register({ id: "doc.undo", label: () => t("Undo on the open page"), group, description: "Undoes the most recent change this person made on the open page, without the editor needing focus.", mutates: true, hidden: () => true, enabled, run: step(undo) }),
    engine.commands.register({ id: "doc.redo", label: () => t("Redo on the open page"), group, description: "Redoes the change most recently undone on the open page, without the editor needing focus.", mutates: true, hidden: () => true, enabled, run: step(redo) }),
  );
  return offs;
}
