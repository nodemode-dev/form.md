import type { EditorView } from "prosemirror-view";
import type { Engine } from "../../engine";
import type { Rect } from "../../ui/floating";

export interface Refreshable {
  refresh(): void;
}

export interface EditorCtx {
  engine: Engine;
  view(): EditorView | null;
  pageId(): string;
  workspaceId(): string;
  writable(): boolean;
  openToggles: Set<string>;
  live: Set<Refreshable>;
  pickFiles(view: EditorView, accept: string, at?: number): void;
  uploadFiles(view: EditorView, files: File[], at?: number): void;
}

export function caretRect(view: EditorView, pos = view.state.selection.head): Rect | null {
  try {
    const c = view.coordsAtPos(pos);
    return { x: c.left, y: c.top, width: Math.max(1, c.right - c.left), height: c.bottom - c.top };
  } catch {
    return null;
  }
}

export function blockRect(view: EditorView, pos: number): Rect | null {
  const dom = view.nodeDOM(pos);
  if (!(dom instanceof HTMLElement)) return caretRect(view, pos + 1);
  const r = dom.getBoundingClientRect();
  return { x: r.left, y: r.top, width: r.width, height: r.height };
}
