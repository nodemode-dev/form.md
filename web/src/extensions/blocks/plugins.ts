import { baseKeymap, chainCommands, createParagraphNear, deleteSelection, exitCode, joinBackward, liftEmptyBlock, newlineInCode, selectNodeBackward, setBlockType, splitBlock } from "prosemirror-commands";
import { dropCursor } from "prosemirror-dropcursor";
import { gapCursor } from "prosemirror-gapcursor";
import { keymap } from "prosemirror-keymap";
import type { Node as PMNode, NodeType } from "prosemirror-model";
import { liftListItem, sinkListItem, splitListItem } from "prosemirror-schema-list";
import { type Command, type EditorState, Plugin, PluginKey, Selection, type Transaction } from "prosemirror-state";
import { columnResizing, goToNextCell, tableEditing } from "prosemirror-tables";
import { Decoration, DecorationSet, type EditorView } from "prosemirror-view";
import { ySyncPluginKey } from "y-prosemirror";
import { BLOCKS_WITH_ID, newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import type { EditorCtx } from "./context";
import { normaliseLanguage } from "./languages";

const N = schema.nodes as Record<string, NodeType>;

export function idsPlugin(): Plugin {
  return new Plugin({
    key: new PluginKey("formmd-ids"),
    appendTransaction(trs, _old, state) {
      if (!trs.some((tr) => tr.docChanged)) return null;
      if (trs.every((tr) => (tr.getMeta(ySyncPluginKey) as { isChangeOrigin?: boolean } | undefined)?.isChangeOrigin)) return null;
      const seen = new Set<string>();
      let tr: Transaction | null = null;
      state.doc.descendants((node, pos) => {
        if (BLOCKS_WITH_ID.has(node.type.name)) {
          const id = node.attrs.id as string | null;
          if (!id || seen.has(id)) {
            tr ??= state.tr;
            tr.setNodeMarkup(pos, undefined, { ...node.attrs, id: newBlockId() });
          } else seen.add(id);
        }
        return !node.isTextblock;
      });
      if (tr) (tr as Transaction).setMeta("addToHistory", false);
      return tr;
    },
  });
}

function placeholderFor(node: PMNode, parent: PMNode | null): string {
  if (node.type.name === "heading") return t("Heading {n}", { n: Number(node.attrs.level) || 1 });
  if (node.type.name === "toggle_title") return t("Toggle");
  if (parent && (parent.type.name === "list_item" || parent.type.name === "task_item")) return t("List");
  if (parent && parent.type.name === "callout") return t("Note");
  return t("Write, or press / for blocks");
}

export function placeholderPlugin(): Plugin {
  return new Plugin({
    key: new PluginKey("formmd-placeholder"),
    props: {
      decorations(state: EditorState) {
        const decos: Decoration[] = [];
        const doc = state.doc;
        const only = doc.childCount === 1 ? doc.firstChild : null;
        if (only && only.type === N.paragraph && only.content.size === 0) {
          decos.push(Decoration.node(0, only.nodeSize, { class: "is-empty is-sole", "data-placeholder": t("Write, or press / for blocks") }));
          return DecorationSet.create(doc, decos);
        }
        const { $from, empty } = state.selection;
        if (empty && $from.parent.isTextblock && $from.parent.content.size === 0 && !$from.parent.type.spec.code && $from.depth > 0) {
          const parent = $from.depth > 1 ? $from.node($from.depth - 1) : null;
          decos.push(Decoration.node($from.before(), $from.after(), { class: "is-empty", "data-placeholder": placeholderFor($from.parent, parent) }));
        }
        doc.descendants((node, pos) => {
          if ((node.type.name === "heading" || node.type.name === "toggle_title") && node.content.size === 0 && !(empty && $from.parent === node)) {
            decos.push(Decoration.node(pos, pos + node.nodeSize, { class: "is-empty is-always", "data-placeholder": placeholderFor(node, null) }));
          }
          return !node.isTextblock;
        });
        return DecorationSet.create(doc, decos);
      },
    },
  });
}

type HL = typeof import("./hl");
let hl: HL | null = null;
let hlLoading: Promise<void> | null = null;
const tokenCache = new WeakMap<PMNode, ReturnType<HL["tokens"]>>();
const hlKey = new PluginKey<DecorationSet>("formmd-highlight");

function hasCode(doc: PMNode): boolean {
  let found = false;
  doc.descendants((n) => {
    if (found) return false;
    if (n.type.name === "code_block" && n.attrs.language) found = true;
    return !n.isTextblock;
  });
  return found;
}

function highlight(doc: PMNode): DecorationSet {
  const mod = hl;
  if (!mod) return DecorationSet.empty;
  const decos: Decoration[] = [];
  doc.descendants((node, pos) => {
    if (node.type.name !== "code_block") return !node.isTextblock;
    const lang = normaliseLanguage(String(node.attrs.language ?? ""));
    if (!lang || lang === "mermaid") return false;
    let toks = tokenCache.get(node);
    if (!toks) {
      toks = mod.tokens(node.textContent, lang);
      tokenCache.set(node, toks);
    }
    for (const tk of toks) decos.push(Decoration.inline(pos + 1 + tk.from, pos + 1 + tk.to, { class: tk.cls }));
    return false;
  });
  return DecorationSet.create(doc, decos);
}

export function highlightPlugin(): Plugin<DecorationSet> {
  return new Plugin<DecorationSet>({
    key: hlKey,
    state: {
      init: (_c, state) => highlight(state.doc),
      apply: (tr, set, _old, state) => (tr.docChanged || tr.getMeta(hlKey) ? highlight(state.doc) : set),
    },
    props: {
      decorations: (state) => hlKey.getState(state),
    },
    view(view) {
      const ensure = () => {
        if (hl || !hasCode(view.state.doc)) return;
        hlLoading ??= import("./hl").then(
          (m) => {
            hl = m;
          },
          () => {
            hlLoading = null;
          },
        );
        void hlLoading.then(() => {
          if (hl && !view.isDestroyed) view.dispatch(view.state.tr.setMeta(hlKey, true));
        });
      };
      ensure();
      return { update: ensure };
    },
  });
}

export const uploadKey = new PluginKey<DecorationSet>("formmd-uploads");

export interface UploadMeta {
  add?: { id: string; pos: number; el: HTMLElement };
  remove?: string;
}

export function uploadsPlugin(): Plugin<DecorationSet> {
  return new Plugin<DecorationSet>({
    key: uploadKey,
    state: {
      init: () => DecorationSet.empty,
      apply(tr, set) {
        let next = set.map(tr.mapping, tr.doc);
        const meta = tr.getMeta(uploadKey) as UploadMeta | undefined;
        if (meta?.add) next = next.add(tr.doc, [Decoration.widget(meta.add.pos, meta.add.el, { id: meta.add.id, side: -1 })]);
        if (meta?.remove) next = next.remove(next.find(undefined, undefined, (spec) => spec.id === meta.remove));
        return next;
      },
    },
    props: {
      decorations: (state) => uploadKey.getState(state),
    },
  });
}

export function uploadPos(state: EditorState, id: string): number | null {
  const found = uploadKey.getState(state)?.find(undefined, undefined, (spec) => spec.id === id);
  return found?.[0]?.from ?? null;
}

export function livePlugin(ctx: EditorCtx): Plugin {
  return new Plugin({
    key: new PluginKey("formmd-live"),
    view: () => ({
      update: (view: EditorView, prev: EditorState) => {
        if (view.state.doc === prev.doc) return;
        for (const v of ctx.live) v.refresh();
      },
    }),
  });
}

const resetEmptyBlock: Command = (state, dispatch) => {
  const { $from, empty } = state.selection;
  if (!empty || $from.parentOffset > 0) return false;
  const parent = $from.parent;
  if (parent.type === N.heading || (parent.type === N.code_block && parent.content.size === 0)) {
    return setBlockType(N.paragraph as NodeType)(state, dispatch);
  }
  return false;
};

const liftAtStart: Command = (state, dispatch) => {
  const { $from, empty } = state.selection;
  if (!empty || $from.parentOffset > 0 || $from.depth < 2) return false;
  const item = $from.node($from.depth - 1);
  if ($from.index($from.depth - 1) !== 0) return false;
  if (item.type === N.list_item) return liftListItem(N.list_item as NodeType)(state, dispatch);
  if (item.type === N.task_item) return liftListItem(N.task_item as NodeType)(state, dispatch);
  return false;
};

const hardBreak: Command = (state, dispatch) => {
  if (state.selection.$from.parent.type.spec.code) return newlineInCode(state, dispatch);
  dispatch?.(state.tr.replaceSelectionWith((N.hard_break as NodeType).create()).scrollIntoView());
  return true;
};

const codeTab: Command = (state, dispatch) => {
  if (!state.selection.$from.parent.type.spec.code) return false;
  dispatch?.(state.tr.insertText("  ").scrollIntoView());
  return true;
};

function toggleEnter(ctx: EditorCtx): Command {
  return (state, dispatch) => {
    const { $from } = state.selection;
    if ($from.parent.type !== N.toggle_title) return false;
    const toggle = $from.node($from.depth - 1);
    const id = String(toggle.attrs.id ?? "");
    if (!ctx.openToggles.has(id)) {
      const after = $from.after($from.depth - 1);
      if (dispatch) {
        const tr = state.tr.insert(after, (N.paragraph as NodeType).create({ id: newBlockId() }));
        tr.setSelection(Selection.near(tr.doc.resolve(after + 1)));
        dispatch(tr.scrollIntoView());
      }
      return true;
    }
    return splitBlock(state, dispatch);
  };
}

export function keysPlugin(ctx: EditorCtx): Plugin[] {
  return [
    keymap({
      Enter: chainCommands(newlineInCode, toggleEnter(ctx), splitListItem(N.list_item as NodeType), splitListItem(N.task_item as NodeType), createParagraphNear, liftEmptyBlock, splitBlock),
      "Mod-Enter": exitCode,
      "Shift-Enter": hardBreak,
      Backspace: chainCommands(deleteSelection, resetEmptyBlock, liftAtStart, joinBackward, selectNodeBackward),
      Tab: chainCommands(codeTab, sinkListItem(N.list_item as NodeType), sinkListItem(N.task_item as NodeType), goToNextCell(1)),
      "Shift-Tab": chainCommands(liftListItem(N.list_item as NodeType), liftListItem(N.task_item as NodeType), goToNextCell(-1)),
    }),
    keymap(baseKeymap),
  ];
}

export function structurePlugins(): Plugin[] {
  return [dropCursor({ class: "drop-cursor", width: 2 }), gapCursor(), columnResizing({ cellMinWidth: 80 }), tableEditing()];
}
