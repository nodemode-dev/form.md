import { lift, setBlockType, toggleMark, wrapIn } from "prosemirror-commands";
import { Fragment, type Node as PMNode, type NodeType, type ResolvedPos } from "prosemirror-model";
import { liftListItem, wrapInList } from "prosemirror-schema-list";
import { type Command, type EditorState, NodeSelection, Selection, TextSelection, type Transaction } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";
import { newBlockId, schema } from "../../engine/doc/schema";

const N = schema.nodes as Record<string, NodeType>;
const LIST_TYPES = new Set(["bullet_list", "ordered_list", "task_list"]);
const CONTAINERS = new Set(["doc", "blockquote", "callout", "toggle", "list_item", "task_item", "column"]);

export interface BlockRef {
  node: PMNode;
  pos: number;
  depth: number;
}

export function blockAt($pos: ResolvedPos): BlockRef | null {
  for (let d = $pos.depth; d > 0; d--) {
    const parent = $pos.node(d - 1);
    if (CONTAINERS.has(parent.type.name)) {
      const node = $pos.node(d);
      if (node.type.name === "toggle_title") continue;
      return { node, pos: $pos.before(d), depth: d };
    }
  }
  return null;
}

export function currentBlock(state: EditorState): BlockRef | null {
  const sel = state.selection;
  if (sel instanceof NodeSelection && sel.node.isBlock) return { node: sel.node, pos: sel.from, depth: sel.$from.depth + 1 };
  return blockAt(sel.$head);
}

export function blockFromPos(state: EditorState, pos: number): BlockRef | null {
  const node = state.doc.nodeAt(pos);
  if (!node) return null;
  const $pos = state.doc.resolve(pos);
  return { node, pos, depth: $pos.depth + 1 };
}

function listItemType(list: string): NodeType {
  return list === "task_list" ? (N.task_item as NodeType) : (N.list_item as NodeType);
}

function liftAll(state: EditorState, dispatch?: (tr: Transaction) => void): EditorState {
  let s = state;
  for (let i = 0; i < 8; i++) {
    const $from = s.selection.$from;
    let inWrapper = false;
    for (let d = $from.depth; d > 0; d--) {
      const t = $from.node(d).type.name;
      if (t === "list_item" || t === "task_item" || t === "blockquote") inWrapper = true;
    }
    if (!inWrapper) break;
    let next: Transaction | null = null;
    const capture = (tr: Transaction) => (next = tr);
    if (!(liftListItem(N.list_item as NodeType)(s, capture) || liftListItem(N.task_item as NodeType)(s, capture) || lift(s, capture))) break;
    if (!next) break;
    dispatch?.(next);
    s = s.apply(next);
  }
  return s;
}

export type BlockKind =
  | "paragraph"
  | "h1"
  | "h2"
  | "h3"
  | "h4"
  | "h5"
  | "h6"
  | "bullet_list"
  | "ordered_list"
  | "task_list"
  | "toggle"
  | "blockquote"
  | "callout"
  | "code_block"
  | "mermaid";

export function turnInto(view: EditorView, kind: BlockKind): boolean {
  const run = (cmd: Command) => cmd(view.state, view.dispatch, view);
  const block = currentBlock(view.state);
  if (block && !block.node.isTextblock && !LIST_TYPES.has(block.node.type.name) && block.node.type.name !== "list_item" && block.node.type.name !== "task_item") {
    const inner = block.node.firstChild;
    if (inner?.isTextblock) view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, block.pos + 2)));
  }
  const heading = /^h([1-6])$/.exec(kind);
  if (kind === "paragraph" || heading || kind === "code_block" || kind === "mermaid") {
    liftAll(view.state, view.dispatch);
    if (kind === "paragraph") return run(setBlockType(N.paragraph as NodeType));
    if (heading) return run(setBlockType(N.heading as NodeType, { level: Number(heading[1]) }));
    return run(setBlockType(N.code_block as NodeType, { language: kind === "mermaid" ? "mermaid" : "" }));
  }
  if (LIST_TYPES.has(kind)) {
    const $from = view.state.selection.$from;
    for (let d = $from.depth; d > 0; d--) {
      const node = $from.node(d);
      if (!LIST_TYPES.has(node.type.name)) continue;
      if (node.type.name === kind) return true;
      const pos = $from.before(d);
      const itemType = listItemType(kind);
      const items: PMNode[] = [];
      node.forEach((item) => items.push(itemType.create(itemType === N.task_item ? { id: item.attrs.id, checked: false } : { id: item.attrs.id }, item.content)));
      const list = (N[kind] as NodeType).create({ id: node.attrs.id }, items);
      const tr = view.state.tr.replaceWith(pos, pos + node.nodeSize, list);
      tr.setSelection(Selection.near(tr.doc.resolve(Math.min(view.state.selection.from, tr.doc.content.size - 1))));
      view.dispatch(tr);
      return true;
    }
    liftAll(view.state, view.dispatch);
    return run(wrapInList(N[kind] as NodeType));
  }
  if (kind === "blockquote" || kind === "callout") return run(wrapIn(N[kind] as NodeType));
  if (kind === "toggle") {
    const b = currentBlock(view.state);
    if (!b) return false;
    const text = b.node.isTextblock ? b.node.content : Fragment.empty;
    const toggle = (N.toggle as NodeType).create({ id: newBlockId() }, [(N.toggle_title as NodeType).create(null, text)]);
    const tr = view.state.tr.replaceWith(b.pos, b.pos + b.node.nodeSize, toggle);
    tr.setSelection(TextSelection.create(tr.doc, b.pos + 2 + text.size));
    view.dispatch(tr.scrollIntoView());
    return true;
  }
  return false;
}

export function insertBlock(view: EditorView, node: PMNode, options: { select?: boolean } = {}): boolean {
  const { state } = view;
  const b = currentBlock(state);
  let tr = state.tr;
  let at: number;
  if (b && b.node.type.name === "paragraph" && b.node.content.size === 0) {
    tr = tr.replaceWith(b.pos, b.pos + b.node.nodeSize, node);
    at = b.pos;
  } else if (b) {
    at = b.pos + b.node.nodeSize;
    tr = tr.insert(at, node);
  } else {
    at = state.selection.to;
    tr = tr.replaceSelectionWith(node);
  }
  const after = at + node.nodeSize;
  const $after = tr.doc.resolve(Math.min(after, tr.doc.content.size));
  if (node.isAtom || node.isLeaf || !node.isTextblock) {
    const next = $after.nodeAfter;
    if (!next || !next.isTextblock) {
      tr = tr.insert(after, (N.paragraph as NodeType).create({ id: newBlockId() }));
    }
  }
  if (options.select && node.isAtom) tr.setSelection(NodeSelection.create(tr.doc, at));
  else if (node.isTextblock) tr.setSelection(TextSelection.create(tr.doc, at + 1));
  else {
    const inner = Selection.findFrom(tr.doc.resolve(at + 1), 1, true);
    tr.setSelection(inner && inner.from < after ? inner : TextSelection.near(tr.doc.resolve(Math.min(after + 1, tr.doc.content.size))));
  }
  view.dispatch(tr.scrollIntoView());
  view.focus();
  return true;
}

export function moveBlock(view: EditorView, dir: -1 | 1): boolean {
  const b = currentBlock(view.state);
  if (!b) return false;
  const $pos = view.state.doc.resolve(b.pos);
  const parent = $pos.parent;
  const index = $pos.index();
  const target = index + dir;
  if (target < 0 || target >= parent.childCount) return false;
  const sibling = parent.child(target);
  const { tr, selection } = view.state;
  const offset = selection.from - b.pos;
  if (dir === -1) {
    const start = b.pos - sibling.nodeSize;
    tr.delete(b.pos, b.pos + b.node.nodeSize).insert(start, b.node);
    const pos = start + offset;
    tr.setSelection(selection instanceof NodeSelection ? NodeSelection.create(tr.doc, start) : TextSelection.near(tr.doc.resolve(pos)));
  } else {
    const end = b.pos + b.node.nodeSize + sibling.nodeSize;
    tr.insert(end, b.node).delete(b.pos, b.pos + b.node.nodeSize);
    const start = end - b.node.nodeSize;
    tr.setSelection(selection instanceof NodeSelection ? NodeSelection.create(tr.doc, start) : TextSelection.near(tr.doc.resolve(start + offset)));
  }
  view.dispatch(tr.scrollIntoView());
  return true;
}

function freshIds(node: PMNode): PMNode {
  const map = (n: PMNode): PMNode => {
    const attrs = "id" in n.attrs ? { ...n.attrs, id: newBlockId() } : n.attrs;
    if (n.isText) return n;
    const kids: PMNode[] = [];
    n.forEach((c) => kids.push(map(c)));
    return n.type.create(attrs, kids, n.marks);
  };
  return map(node);
}

export function duplicateBlock(view: EditorView, at?: BlockRef | null): boolean {
  const b = at ?? currentBlock(view.state);
  if (!b) return false;
  const end = b.pos + b.node.nodeSize;
  const tr = view.state.tr.insert(end, freshIds(b.node));
  tr.setSelection(Selection.near(tr.doc.resolve(end + 1)));
  view.dispatch(tr.scrollIntoView());
  return true;
}

export function deleteBlock(view: EditorView, at?: BlockRef | null): boolean {
  const b = at ?? currentBlock(view.state);
  if (!b) return false;
  const tr = view.state.tr.delete(b.pos, b.pos + b.node.nodeSize);
  if (tr.doc.childCount === 0) tr.insert(0, (N.paragraph as NodeType).create({ id: newBlockId() }));
  tr.setSelection(Selection.near(tr.doc.resolve(Math.min(b.pos, tr.doc.content.size)), -1));
  view.dispatch(tr.scrollIntoView());
  return true;
}

export function toggleCheck(view: EditorView): boolean {
  const $from = view.state.selection.$from;
  for (let d = $from.depth; d > 0; d--) {
    const n = $from.node(d);
    if (n.type.name === "task_item") {
      view.dispatch(view.state.tr.setNodeMarkup($from.before(d), undefined, { ...n.attrs, checked: !n.attrs.checked }));
      return true;
    }
  }
  return false;
}

export function markCommand(name: string, attrs?: Record<string, unknown>): Command {
  const type = schema.marks[name];
  if (!type) return () => false;
  return toggleMark(type, attrs ?? null);
}

export function setMark(view: EditorView, name: string, attrs: Record<string, unknown> | null): boolean {
  const type = schema.marks[name];
  if (!type) return false;
  const { from, to, empty } = view.state.selection;
  if (empty) return false;
  const tr = view.state.tr.removeMark(from, to, type);
  if (attrs) tr.addMark(from, to, type.create(attrs));
  view.dispatch(tr);
  return true;
}

export function markActive(state: EditorState, name: string): boolean {
  const type = schema.marks[name];
  if (!type) return false;
  const { from, to, empty, $from } = state.selection;
  if (empty) return !!type.isInSet(state.storedMarks ?? $from.marks());
  return state.doc.rangeHasMark(from, to, type);
}

function emptyCell(header: boolean): PMNode {
  return ((header ? N.table_header : N.table_cell) as NodeType).create(null, [(N.paragraph as NodeType).create({ id: newBlockId() })]);
}

export function tableAddRow(view: EditorView, table: BlockRef): boolean {
  const last = table.node.lastChild;
  if (!last) return false;
  const cells: PMNode[] = [];
  last.forEach((c) => cells.push(emptyCell(false)));
  const at = table.pos + table.node.nodeSize - 1;
  const tr = view.state.tr.insert(at, (N.table_row as NodeType).create(null, cells));
  tr.setSelection(TextSelection.near(tr.doc.resolve(at + 3)));
  view.dispatch(tr.scrollIntoView());
  view.focus();
  return true;
}

export function tableAddColumn(view: EditorView, table: BlockRef): boolean {
  const rows: PMNode[] = [];
  table.node.forEach((row) => {
    const cells: PMNode[] = [];
    row.forEach((c) => cells.push(c));
    cells.push(emptyCell(row.firstChild?.type === N.table_header));
    rows.push(row.type.create(row.attrs, cells));
  });
  view.dispatch(view.state.tr.replaceWith(table.pos, table.pos + table.node.nodeSize, table.node.type.create(table.node.attrs, rows)));
  view.focus();
  return true;
}

export function tableRemoveRow(view: EditorView, table: BlockRef): boolean {
  if (table.node.childCount < 2) return false;
  const last = table.node.lastChild as PMNode;
  const end = table.pos + table.node.nodeSize - 1;
  view.dispatch(view.state.tr.delete(end - last.nodeSize, end));
  return true;
}

export function tableRemoveColumn(view: EditorView, table: BlockRef): boolean {
  if ((table.node.firstChild?.childCount ?? 0) < 2) return false;
  const rows: PMNode[] = [];
  table.node.forEach((row) => {
    const cells: PMNode[] = [];
    row.forEach((c, _o, i) => {
      if (i < row.childCount - 1) cells.push(c);
    });
    rows.push(row.type.create(row.attrs, cells));
  });
  view.dispatch(view.state.tr.replaceWith(table.pos, table.pos + table.node.nodeSize, table.node.type.create(table.node.attrs, rows)));
  return true;
}

export const tabOutOfLastCell: Command = (state, dispatch, view) => {
  const $from = state.selection.$from;
  for (let d = $from.depth; d > 0; d--) {
    if ($from.node(d).type !== N.table) continue;
    const table = $from.node(d);
    const rowIndex = $from.index(d);
    const row = table.child(rowIndex);
    const cellIndex = $from.index(d + 1);
    if (rowIndex !== table.childCount - 1 || cellIndex !== row.childCount - 1) return false;
    if (dispatch && view) tableAddRow(view, { node: table, pos: $from.before(d), depth: d });
    return true;
  }
  return false;
};

export function selectBlock(view: EditorView, b: BlockRef): void {
  view.dispatch(view.state.tr.setSelection(NodeSelection.create(view.state.doc, b.pos)));
}
