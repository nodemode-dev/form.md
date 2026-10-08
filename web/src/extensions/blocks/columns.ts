import { Fragment, type Node as PMNode, type NodeType } from "prosemirror-model";
import { NodeSelection, Plugin, PluginKey, type Transaction } from "prosemirror-state";
import type { EditorView, NodeView, ViewMutationRecord } from "prosemirror-view";
import { newBlockId, schema } from "../../engine/doc/schema";
import { h, setAttr } from "../../ui/dom";

type GetPos = () => number | undefined;

const N = schema.nodes as Record<string, NodeType>;
const STEP = 5;
const MIN = 10;

function weights(columns: PMNode): number[] {
  const n = columns.childCount;
  const out: number[] = [];
  columns.forEach((c) => out.push(Number(c.attrs.width) || Math.round(100 / n)));
  return out;
}

export class ColumnView implements NodeView {
  dom: HTMLElement;
  contentDOM: HTMLElement;
  private grip: HTMLElement;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
  ) {
    this.contentDOM = h("div", { class: "column__body" });
    this.grip = h("div", { class: "column__resize", attrs: { contenteditable: "false", "aria-hidden": "true" } });
    this.dom = h("div", { class: "column" }, this.contentDOM, this.grip);
    this.paint();
    this.grip.addEventListener("pointerdown", (e) => this.drag(e));
  }

  private paint(): void {
    setAttr(this.dom, "data-w", this.node.attrs.width ? String(this.node.attrs.width) : null);
  }

  private drag(e: PointerEvent): void {
    const pos = this.getPos();
    const row = this.dom.parentElement;
    if (pos === undefined || !row || !this.view.editable || e.button !== 0) return;
    const $pos = this.view.state.doc.resolve(pos);
    const columns = $pos.parent;
    const index = $pos.index();
    if (columns.type !== N.columns || index >= columns.childCount - 1) return;
    e.preventDefault();
    this.grip.setPointerCapture(e.pointerId);
    const w = weights(columns);
    const total = w.reduce((a, b) => a + b, 0);
    const scale = 100 / total;
    const pct = w.map((x) => x * scale);
    const before = pct.slice(0, index).reduce((a, b) => a + b, 0);
    const pair = (pct[index] ?? 0) + (pct[index + 1] ?? 0);
    const cells = Array.from(row.children) as HTMLElement[];
    let next = pct.map((x) => Math.max(STEP, Math.round(x / STEP) * STEP));
    const paint = () => cells.forEach((c, i) => setAttr(c, "data-w", String(next[i] ?? STEP)));
    paint();
    this.dom.classList.add("is-resizing");
    const move = (ev: PointerEvent) => {
      const rect = row.getBoundingClientRect();
      const at = ((ev.clientX - rect.left) / rect.width) * 100 - before;
      const left = Math.min(Math.max(Math.round(at / STEP) * STEP, MIN), Math.round(pair / STEP) * STEP - MIN);
      const right = Math.max(MIN, Math.round(pair / STEP) * STEP - left);
      next = next.map((x, i) => (i === index ? left : i === index + 1 ? right : x));
      paint();
    };
    const up = () => {
      this.grip.removeEventListener("pointermove", move);
      this.grip.removeEventListener("pointerup", up);
      this.grip.removeEventListener("pointercancel", up);
      this.dom.classList.remove("is-resizing");
      const start = this.getPos();
      if (start === undefined) return;
      const $start = this.view.state.doc.resolve(start);
      const parentPos = $start.before();
      const parent = this.view.state.doc.nodeAt(parentPos);
      if (!parent || parent.type !== N.columns) return;
      let tr = this.view.state.tr;
      let p = parentPos + 1;
      parent.forEach((c, _o, i) => {
        if (Number(c.attrs.width) !== next[i]) tr = tr.setNodeMarkup(p, undefined, { ...c.attrs, width: next[i] ?? null });
        p += c.nodeSize;
      });
      if (tr.docChanged) this.view.dispatch(tr);
    };
    this.grip.addEventListener("pointermove", move);
    this.grip.addEventListener("pointerup", up);
    this.grip.addEventListener("pointercancel", up);
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    return this.grip.contains(e.target as Node);
  }

  ignoreMutation(m: ViewMutationRecord): boolean {
    if (m.type === "selection") return false;
    return this.grip.contains(m.target) || (m.type === "attributes" && m.target === this.dom);
  }
}

export function addColumn(view: EditorView, pos: number): void {
  const node = view.state.doc.nodeAt(pos);
  if (!node || node.type !== N.columns) return;
  let tr = view.state.tr.insert(pos + node.nodeSize - 1, (N.column as NodeType).create(null, [(N.paragraph as NodeType).create({ id: newBlockId() })]));
  tr = resetWidths(tr, pos);
  view.dispatch(tr);
}

export function removeLastColumn(view: EditorView, pos: number): void {
  const node = view.state.doc.nodeAt(pos);
  if (!node || node.type !== N.columns) return;
  const last = node.lastChild;
  if (!last) return;
  if (node.childCount <= 2) {
    const first = node.firstChild as PMNode;
    view.dispatch(view.state.tr.replaceWith(pos, pos + node.nodeSize, first.content.append(last.content)));
    return;
  }
  const end = pos + node.nodeSize - 1;
  let tr = view.state.tr.delete(end - last.nodeSize, end);
  tr = resetWidths(tr, pos);
  const row = tr.doc.nodeAt(pos);
  if (row) tr = tr.insert(pos + row.nodeSize, last.content);
  view.dispatch(tr);
}

export function resetWidths(tr: Transaction, pos: number): Transaction {
  const node = tr.doc.nodeAt(pos);
  if (!node || node.type !== N.columns) return tr;
  let p = pos + 1;
  node.forEach((c) => {
    if (c.attrs.width !== null) tr = tr.setNodeMarkup(p, undefined, { ...c.attrs, width: null });
    p += c.nodeSize;
  });
  return tr;
}

interface SideTarget {
  pos: number;
  side: "left" | "right";
  column: boolean;
  rect: DOMRect;
}

function sideTarget(view: EditorView, e: DragEvent): SideTarget | null {
  const sel = view.state.selection;
  if (!(sel instanceof NodeSelection)) return null;
  const hit = view.posAtCoords({ left: e.clientX, top: e.clientY });
  if (!hit) return null;
  const inside = hit.inside >= 0 ? view.state.doc.nodeAt(hit.inside) : null;
  const leaf = inside?.isLeaf ? hit.inside : -1;
  const $p = view.state.doc.resolve(inside && !inside.isLeaf ? hit.inside + 1 : hit.inside >= 0 ? hit.inside : hit.pos);
  for (let pd = $p.depth; pd >= 0; pd--) {
    const child = pd === $p.depth ? leaf : $p.before(pd + 1);
    if (child < 0) continue;
    const name = $p.node(pd).type.name;
    if (name !== "doc" && name !== "column") continue;
    const inColumn = name === "column";
    const pos = inColumn ? $p.before(pd) : child;
    const node = view.state.doc.nodeAt(pos);
    if (!node) return null;
    if (pos <= sel.from && pos + node.nodeSize >= sel.to) return null;
    const dom = view.nodeDOM(pos);
    if (!(dom instanceof HTMLElement)) return null;
    const rect = dom.getBoundingClientRect();
    if (e.clientY < rect.top || e.clientY > rect.bottom) return null;
    const edge = Math.min(64, rect.width * 0.2);
    const side = e.clientX < rect.left + edge ? "left" : e.clientX > rect.right - edge ? "right" : null;
    if (!side) return null;
    if (node.type === N.columns && node.childCount > 0) {
      const last = node.lastChild as PMNode;
      const colPos = side === "left" ? pos + 1 : pos + node.nodeSize - 1 - last.nodeSize;
      return { pos: colPos, side, column: true, rect };
    }
    return { pos, side, column: inColumn, rect };
  }
  return null;
}

function removeDragged(tr: Transaction, from: number): Transaction {
  const $from = tr.doc.resolve(from);
  const node = tr.doc.nodeAt(from);
  if (!node) return tr;
  if ($from.parent.type === N.column && $from.parent.childCount === 1 && $from.depth >= 2) {
    const colPos = $from.before();
    const col = $from.parent;
    const row = $from.node($from.depth - 1);
    const rowPos = $from.before($from.depth - 1);
    if (row.type === N.columns && row.childCount === 2) {
      const other = row.firstChild === col ? row.lastChild : row.firstChild;
      if (other) return tr.replaceWith(rowPos, rowPos + row.nodeSize, other.content);
    }
    return resetWidths(tr.delete(colPos, colPos + col.nodeSize), rowPos);
  }
  return tr.delete(from, from + node.nodeSize);
}

export function sideDrop(view: EditorView, target: SideTarget): void {
  const sel = view.state.selection as NodeSelection;
  const dragged = sel.node;
  const col = (N.column as NodeType).create(null, [dragged]);
  if (target.column) {
    const $t = view.state.doc.resolve(target.pos);
    const row = $t.parent;
    const rowPos = $t.before();
    if (row.type === N.columns && sel.from > rowPos && sel.to < rowPos + row.nodeSize) {
      const cols: PMNode[] = [];
      row.forEach((c, offset) => {
        const cpos = rowPos + 1 + offset;
        let keep: PMNode | null = c;
        if (sel.from > cpos && sel.to < cpos + c.nodeSize) {
          const inner = sel.from - (cpos + 1);
          const content = c.content.cut(0, inner).append(c.content.cut(inner + dragged.nodeSize));
          keep = content.childCount ? c.copy(content) : null;
        }
        if (cpos === target.pos && target.side === "left") cols.push(col);
        if (keep) cols.push(keep);
        if (cpos === target.pos && target.side === "right") cols.push(col);
      });
      const even = cols.map((c) => (N.column as NodeType).create({ width: null }, c.content));
      const tr = view.state.tr;
      if (even.length >= 2) tr.replaceWith(rowPos, rowPos + row.nodeSize, row.copy(Fragment.from(even)));
      else tr.replaceWith(rowPos, rowPos + row.nodeSize, even[0]?.content ?? Fragment.empty);
      view.dispatch(tr.scrollIntoView());
      return;
    }
  }
  let tr = removeDragged(view.state.tr, sel.from);
  const pos = tr.mapping.map(target.pos, target.side === "left" ? 1 : -1);
  const node = tr.doc.nodeAt(pos);
  if (!node) return;
  if (target.column && node.type === N.column) {
    const at = target.side === "left" ? pos : pos + node.nodeSize;
    tr = tr.insert(at, col);
    const $col = tr.doc.resolve(at);
    tr = resetWidths(tr, $col.before());
  } else {
    const own = (N.column as NodeType).create(null, [node]);
    const cols = (N.columns as NodeType).create({ id: newBlockId() }, target.side === "left" ? [col, own] : [own, col]);
    tr = tr.replaceWith(pos, pos + node.nodeSize, cols);
  }
  view.dispatch(tr.scrollIntoView());
}

const key = new PluginKey("formmd-columns");

export function columnsPlugins(wrap: () => HTMLElement | null): Plugin[] {
  let marker: HTMLElement | null = null;
  let target: SideTarget | null = null;
  let lifted = -1;
  const hide = () => {
    target = null;
    marker?.classList.remove("is-visible");
    wrap()?.classList.remove("is-side-drop");
  };
  const show = (t: SideTarget) => {
    const host = wrap();
    if (!host) return;
    if (!marker || !marker.isConnected) {
      marker = h("div", { class: "side-drop", attrs: { "aria-hidden": "true" } });
      host.append(marker);
    }
    const box = host.getBoundingClientRect();
    const x = (t.side === "left" ? t.rect.left - 6 : t.rect.right + 4) - box.left;
    marker.style.transform = `translate(${Math.round(x)}px, ${Math.round(t.rect.top - box.top)}px)`;
    marker.style.height = `${Math.round(t.rect.height)}px`;
    marker.classList.add("is-visible");
    host.classList.add("is-side-drop");
  };
  const drop = new Plugin({
    key,
    props: {
      handleDOMEvents: {
        dragover(view, e) {
          if (!view.dragging || !view.editable) {
            hide();
            return false;
          }
          const t = sideTarget(view, e);
          if (!t) {
            hide();
            return false;
          }
          e.preventDefault();
          target = t;
          show(t);
          return true;
        },
        dragleave(view, e) {
          if (!view.dom.contains(e.relatedTarget as Node | null)) hide();
          return false;
        },
        dragend() {
          hide();
          lifted = -1;
          return false;
        },
        drop(view, e) {
          const t = target;
          hide();
          const sel = view.state.selection;
          lifted = view.dragging?.move && sel instanceof NodeSelection && sel.$from.parent.type === N.column && sel.$from.parent.childCount === 1 ? sel.$from.before() : -1;
          if (!t || !view.dragging) return false;
          lifted = -1;
          e.preventDefault();
          sideDrop(view, t);
          view.dragging = null;
          return true;
        },
      },
    },
  });
  const normalize = new Plugin({
    appendTransaction(trs, _old, state) {
      if (!trs.some((t) => t.docChanged)) return null;
      if (lifted >= 0 && trs.some((t) => t.getMeta("uiEvent") === "drop")) {
        let at = lifted;
        lifted = -1;
        for (const t of trs) at = t.mapping.map(at, 1);
        const col = state.doc.nodeAt(at);
        const only = col?.childCount === 1 ? col.firstChild : null;
        if (col && col.type === N.column && only?.isTextblock && only.content.size === 0) {
          const $c = state.doc.resolve(at);
          const row = $c.parent;
          const rowPos = $c.before();
          const other = row.childCount === 2 ? (row.firstChild === col ? row.lastChild : row.firstChild) : null;
          if (other) return state.tr.replaceWith(rowPos, rowPos + row.nodeSize, other.content);
          return resetWidths(state.tr.delete(at, at + col.nodeSize), rowPos);
        }
      }
      const lonely: { pos: number; node: PMNode }[] = [];
      state.doc.descendants((n, pos) => {
        if (n.type === N.columns && n.childCount < 2) lonely.push({ pos, node: n });
        return n.type !== N.columns || n.childCount < 2;
      });
      if (!lonely.length) return null;
      let tr = state.tr;
      for (const { pos, node } of lonely.reverse()) {
        const inner = node.firstChild;
        if (inner) tr = tr.replaceWith(pos, pos + node.nodeSize, inner.content);
        else tr = tr.delete(pos, pos + node.nodeSize);
      }
      return tr;
    },
  });
  return [drop, normalize];
}
