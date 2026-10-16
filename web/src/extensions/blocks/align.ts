import type { EditorView } from "prosemirror-view";
import { ALIGNS } from "../../engine/doc/schema";

export type Align = "" | (typeof ALIGNS)[number];

export function setAlign(view: EditorView, align: Align): boolean {
  const { from, to } = view.state.selection;
  const tr = view.state.tr;
  view.state.doc.nodesBetween(from, to, (node, pos) => {
    if (node.type.name === "paragraph" || node.type.name === "heading") {
      if ((node.attrs.align ?? "") !== align) tr.setNodeMarkup(pos, undefined, { ...node.attrs, align });
      return false;
    }
    return true;
  });
  if (!tr.docChanged) return false;
  view.dispatch(tr);
  return true;
}

export function alignAt(view: EditorView): Align {
  const node = view.state.selection.$from.parent;
  return (node.attrs.align as Align | undefined) ?? "";
}
