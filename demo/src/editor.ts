import { baseKeymap, chainCommands, createParagraphNear, deleteSelection, exitCode, joinBackward, liftEmptyBlock, newlineInCode, selectNodeBackward, splitBlock } from "prosemirror-commands";
import { keymap } from "prosemirror-keymap";
import type { Node as PMNode, NodeType } from "prosemirror-model";
import { liftListItem, sinkListItem, splitListItem } from "prosemirror-schema-list";
import { EditorState, type Plugin, TextSelection } from "prosemirror-state";
import { goToNextCell } from "prosemirror-tables";
import { EditorView } from "prosemirror-view";
import { initProseMirrorDoc, redo, undo, yUndoPlugin, ySyncPlugin } from "y-prosemirror";
import { ROOT, schema } from "../../web/src/engine/doc/schema";
import { markActive, markCommand, setMark, tabOutOfLastCell, turnInto, type BlockKind } from "../../web/src/extensions/blocks/commands";
import { idsPlugin, placeholderPlugin, structurePlugins } from "../../web/src/extensions/blocks/plugins";
import { rulesPlugin } from "../../web/src/extensions/blocks/rules";
import { looksLikeMarkdown, parseMarkdownSlice, preloadMarkdown } from "../../web/src/extensions/blocks/markdown";
import { h } from "../../web/src/ui/dom";
import { simpleDialog } from "../../web/src/ui/dialog";
import { actions, button, field, input } from "../../web/src/ui/form";
import type { PageItem } from "./store";

const N = schema.nodes as Record<string, NodeType>;

function keys(): Plugin[] {
  const list = N.list_item as NodeType;
  const task = N.task_item as NodeType;
  return [
    keymap({
      Enter: chainCommands(newlineInCode, splitListItem(list), splitListItem(task), createParagraphNear, liftEmptyBlock, splitBlock),
      "Mod-Enter": exitCode,
      Backspace: chainCommands(deleteSelection, joinBackward, selectNodeBackward),
      Tab: chainCommands(sinkListItem(list), sinkListItem(task), goToNextCell(1), tabOutOfLastCell),
      "Shift-Tab": chainCommands(liftListItem(list), liftListItem(task), goToNextCell(-1)),
      "Mod-b": markCommand("strong"),
      "Mod-i": markCommand("em"),
      "Mod-u": markCommand("underline"),
      "Mod-e": markCommand("code"),
      "Mod-Shift-x": markCommand("strike"),
      "Mod-z": undo,
      "Mod-y": redo,
      "Mod-Shift-z": redo,
    }),
    keymap(baseKeymap),
  ];
}

export interface FormatTool {
  id: string;
  label: string;
  icon: string;
  mark?: string;
  block?: BlockKind;
}

export const FORMAT_TOOLS: FormatTool[] = [
  { id: "bold", label: "Bold", icon: "bold", mark: "strong" },
  { id: "italic", label: "Italic", icon: "italic", mark: "em" },
  { id: "underline", label: "Underline", icon: "underline", mark: "underline" },
  { id: "strike", label: "Strikethrough", icon: "strike", mark: "strike" },
  { id: "code", label: "Inline code", icon: "code", mark: "code" },
  { id: "link", label: "Link", icon: "link", mark: "link" },
  { id: "h1", label: "Heading 1", icon: "h1", block: "h1" },
  { id: "h2", label: "Heading 2", icon: "h2", block: "h2" },
  { id: "h3", label: "Heading 3", icon: "h3", block: "h3" },
  { id: "bullet", label: "Bulleted list", icon: "list-bullet", block: "bullet_list" },
  { id: "number", label: "Numbered list", icon: "list-number", block: "ordered_list" },
  { id: "task", label: "To-do list", icon: "checkbox", block: "task_list" },
  { id: "quote", label: "Quote", icon: "quote", block: "blockquote" },
  { id: "callout", label: "Callout", icon: "callout", block: "callout" },
  { id: "codeblock", label: "Code block", icon: "code", block: "code_block" },
];

export interface Hit {
  from: number;
  to: number;
}

export class DemoEditor {
  private view: EditorView | null = null;
  private page: PageItem | null = null;
  private readonly changeListeners = new Set<() => void>();

  constructor(private readonly mount: HTMLElement) {}

  onChange(fn: () => void): void {
    this.changeListeners.add(fn);
  }

  get current(): PageItem | null {
    return this.page;
  }

  get editor(): EditorView | null {
    return this.view;
  }

  private plugins(fragment: ReturnType<PageItem["ydoc"]["getXmlFragment"]>, mapping: unknown): Plugin[] {
    return [ySyncPlugin(fragment, { mapping: mapping as never }), yUndoPlugin(), rulesPlugin(), idsPlugin(), placeholderPlugin(), ...keys(), ...structurePlugins()];
  }

  open(page: PageItem): void {
    this.page = page;
    const fragment = page.ydoc.getXmlFragment(ROOT);
    let doc: PMNode;
    let mapping: unknown = null;
    if (fragment.length > 0) {
      const init = initProseMirrorDoc(fragment, schema);
      doc = init.doc;
      mapping = init.mapping;
    } else doc = schema.topNodeType.createAndFill() as PMNode;
    const state = EditorState.create({ schema, doc, plugins: this.plugins(fragment, mapping) });
    if (!this.view) {
      this.view = new EditorView(
        { mount: this.mount },
        {
          state,
          attributes: { class: "doc-body", "aria-label": "Page content", spellcheck: "true" },
          dispatchTransaction: (tr) => {
            const view = this.view;
            if (!view) return;
            const next = view.state.apply(tr);
            view.updateState(next);
            if (tr.docChanged) {
              if (this.page) this.page.edited = Date.now();
              for (const fn of this.changeListeners) fn();
            }
          },
          clipboardTextParser: ((text: string, $context: never, plain: boolean) => (!plain && looksLikeMarkdown(text) ? parseMarkdownSlice(text, $context) : null)) as never,
        },
      );
      this.view.dom.addEventListener("focusin", preloadMarkdown);
      preloadMarkdown();
    } else this.view.updateState(state);
    for (const fn of this.changeListeners) fn();
  }

  focus(): void {
    this.view?.focus();
  }

  run(tool: FormatTool): void {
    const view = this.view;
    if (!view) return;
    view.focus();
    if (tool.id === "link") {
      this.linkDialog();
      return;
    }
    if (tool.mark) markCommand(tool.mark)(view.state, view.dispatch, view);
    else if (tool.block) turnInto(view, tool.block);
    view.focus();
  }

  active(tool: FormatTool): boolean {
    const view = this.view;
    return !!view && !!tool.mark && markActive(view.state, tool.mark);
  }

  linkDialog(): void {
    const view = this.view;
    if (!view) return;
    simpleDialog("Link", (body, close) => {
      const url = input({ type: "url", placeholder: "https://example.org", autocomplete: "off" });
      const apply = button("Apply", "primary", undefined, "submit");
      const cancel = button("Cancel", "ghost");
      cancel.addEventListener("click", close);
      const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, field("Address", url), actions(cancel, apply));
      form.addEventListener("submit", (e) => {
        e.preventDefault();
        const href = url.value.trim();
        close();
        setMark(view, "link", href ? { href } : null);
        view.focus();
      });
      body.append(form);
      return url;
    });
  }

  outline(): { level: number; text: string; pos: number }[] {
    const out: { level: number; text: string; pos: number }[] = [];
    this.view?.state.doc.descendants((node, pos) => {
      if (node.type.name === "heading") out.push({ level: Number(node.attrs.level) || 1, text: node.textContent, pos });
      return !node.isTextblock;
    });
    return out;
  }

  scrollTo(pos: number): void {
    const view = this.view;
    if (!view) return;
    const clamped = Math.min(pos + 1, view.state.doc.content.size);
    view.dispatch(view.state.tr.setSelection(TextSelection.near(view.state.doc.resolve(clamped))).scrollIntoView());
    view.focus();
  }

  text(): string {
    return this.view?.state.doc.textBetween(0, this.view.state.doc.content.size, " ", " ") ?? "";
  }

  stats(): { words: number; chars: number; minutes: number } {
    const text = this.text().trim();
    const words = text ? text.split(/\s+/).length : 0;
    return { words, chars: text.length, minutes: Math.max(1, Math.round(words / 220)) };
  }

  find(query: string, caseSensitive: boolean): Hit[] {
    const view = this.view;
    if (!view || !query) return [];
    const hits: Hit[] = [];
    const needle = caseSensitive ? query : query.toLowerCase();
    view.state.doc.descendants((node, pos) => {
      if (!node.isText || !node.text) return true;
      const hay = caseSensitive ? node.text : node.text.toLowerCase();
      let i = hay.indexOf(needle);
      while (i >= 0) {
        hits.push({ from: pos + i, to: pos + i + needle.length });
        i = hay.indexOf(needle, i + needle.length);
      }
      return true;
    });
    return hits;
  }

  replace(hits: Hit[], text: string): number {
    const view = this.view;
    if (!view || !hits.length) return 0;
    const tr = view.state.tr;
    for (const hit of [...hits].reverse()) {
      if (text) tr.replaceWith(hit.from, hit.to, schema.text(text, tr.doc.resolve(hit.from).marks()));
      else tr.delete(hit.from, hit.to);
    }
    view.dispatch(tr);
    return hits.length;
  }

  insertMarkdown(md: string, parse: (md: string) => Promise<PMNode>): void {
    const view = this.view;
    if (!view) return;
    void parse(md).then((doc) => {
      const end = view.state.doc.content.size;
      view.dispatch(view.state.tr.insert(end, doc.content).scrollIntoView());
    });
  }
}
