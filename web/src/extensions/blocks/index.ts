import { redo, undo } from "y-prosemirror";
import type { Engine } from "../../engine";
import { t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import { toast } from "../../ui/toast";
import type { Extension } from "../registry";
import { type BlockKind, duplicateBlock, markCommand, moveBlock, toggleCheck, turnInto } from "./commands";
import { EditorHost } from "./editor";
import { MarkdownWriter } from "./markdown";
import { insertInlineMath } from "./math";

const EDITING = () => t("Editing");
const BLOCKS = () => t("Blocks");

let host: EditorHost | null = null;
let teardown: Unsubscribe[] = [];

export function mountEditor(engine: Engine, mount: HTMLElement, wrap: HTMLElement): EditorHost {
  host ??= new EditorHost(engine, mount, wrap);
  return host;
}

export function editorHost(): EditorHost | null {
  return host;
}

function prefetchOnHover(engine: Engine): void {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let last = "";
  document.addEventListener("pointerover", (e) => {
    const el = (e.target as Element).closest?.<HTMLElement>("[data-id]");
    const id = el?.dataset.id ?? "";
    if (!/^[a-z2-7]{26}$/.test(id) || id === last) return;
    clearTimeout(timer);
    timer = setTimeout(() => {
      last = id;
      if (engine.state.get("auth") === "app") engine.docs.prefetch(id);
    }, 90);
  });
}

function setup(engine: Engine): void {
  const { commands, keymap } = engine;
  prefetchOnHover(engine);
  const focused = () => {
    const v = host?.view();
    return !!v && v.hasFocus();
  };
  const editing = () => focused() && !!host?.writable();
  const withView = (fn: (v: NonNullable<ReturnType<EditorHost["view"]>>) => void) => () => {
    const v = host?.view();
    if (v) fn(v);
  };

  const mark = (id: string, label: () => string, name: string, chords: string[]) => {
    teardown.push(
      commands.register({ id, label, group: EDITING, enabled: editing, run: withView((v) => void markCommand(name)(v.state, v.dispatch, v)) }),
      keymap.register({ id, command: id, chords, scope: "editor", whileTyping: true }),
    );
  };
  const block = (id: string, label: () => string, kind: BlockKind, chords: string[]) => {
    teardown.push(
      commands.register({ id, label, group: BLOCKS, enabled: editing, run: withView((v) => void turnInto(v, kind)) }),
      keymap.register({ id, command: id, chords, scope: "editor", whileTyping: true }),
    );
  };

  mark("editor.bold", () => t("Bold"), "strong", ["mod+b"]);
  mark("editor.italic", () => t("Italic"), "em", ["mod+i"]);
  mark("editor.underline", () => t("Underline"), "underline", ["mod+u"]);
  mark("editor.strike", () => t("Strikethrough"), "strike", ["mod+shift+x"]);
  mark("editor.code", () => t("Inline code"), "code", ["mod+e"]);
  block("editor.text", () => t("Turn into text"), "paragraph", ["mod+alt+0"]);
  block("editor.h1", () => t("Turn into heading 1"), "h1", ["mod+alt+1"]);
  block("editor.h2", () => t("Turn into heading 2"), "h2", ["mod+alt+2"]);
  block("editor.h3", () => t("Turn into heading 3"), "h3", ["mod+alt+3"]);
  block("editor.bullet", () => t("Turn into a bulleted list"), "bullet_list", ["mod+shift+8"]);
  block("editor.ordered", () => t("Turn into a numbered list"), "ordered_list", ["mod+shift+7"]);
  block("editor.task", () => t("Turn into a to-do list"), "task_list", ["mod+shift+9"]);
  block("editor.codeBlock", () => t("Turn into code"), "code_block", ["mod+alt+c"]);

  teardown.push(
    commands.register({
      id: "editor.link",
      label: () => t("Link"),
      group: EDITING,
      enabled: () => editing() && !host?.view()?.state.selection.empty,
      run: () => {
        const v = host?.view();
        if (v) host?.formatBar().editLink(v);
      },
    }),
    keymap.register({ id: "editor.link", command: "editor.link", chords: ["mod+k"], scope: "editor", whileTyping: true }),
    commands.register({ id: "editor.inlineMath", label: () => t("Inline equation"), group: EDITING, enabled: editing, run: withView((v) => void insertInlineMath(v)) }),
    keymap.register({ id: "editor.inlineMath", command: "editor.inlineMath", chords: ["mod+shift+e"], scope: "editor", whileTyping: true }),
    commands.register({ id: "editor.undo", label: () => t("Undo"), group: EDITING, enabled: editing, run: withView((v) => void undo(v.state)) }),
    keymap.register({ id: "editor.undo", command: "editor.undo", chords: ["mod+z"], scope: "editor", whileTyping: true }),
    commands.register({ id: "editor.redo", label: () => t("Redo"), group: EDITING, enabled: editing, run: withView((v) => void redo(v.state)) }),
    keymap.register({ id: "editor.redo", command: "editor.redo", chords: ["mod+shift+z", "mod+y"], scope: "editor", whileTyping: true }),
    commands.register({ id: "editor.moveUp", label: () => t("Move block up"), group: BLOCKS, enabled: editing, run: withView((v) => void moveBlock(v, -1)) }),
    keymap.register({ id: "editor.moveUp", command: "editor.moveUp", chords: ["alt+arrowup"], scope: "editor", whileTyping: true }),
    commands.register({ id: "editor.moveDown", label: () => t("Move block down"), group: BLOCKS, enabled: editing, run: withView((v) => void moveBlock(v, 1)) }),
    keymap.register({ id: "editor.moveDown", command: "editor.moveDown", chords: ["alt+arrowdown"], scope: "editor", whileTyping: true }),
    commands.register({ id: "editor.duplicate", label: () => t("Duplicate block"), group: BLOCKS, enabled: editing, run: withView((v) => void duplicateBlock(v)) }),
    keymap.register({ id: "editor.duplicate", command: "editor.duplicate", chords: ["mod+d"], scope: "editor", whileTyping: true }),
    commands.register({
      id: "editor.toggleCheck",
      label: () => t("Tick or untick a to-do"),
      group: BLOCKS,
      enabled: () => {
        const v = host?.view();
        if (!v || !editing()) return false;
        const $from = v.state.selection.$from;
        for (let d = $from.depth; d > 0; d--) if ($from.node(d).type.name === "task_item") return true;
        return false;
      },
      run: withView((v) => void toggleCheck(v)),
    }),
    keymap.register({ id: "editor.toggleCheck", command: "editor.toggleCheck", chords: ["mod+enter"], scope: "editor", whileTyping: true }),
    commands.register({
      id: "page.copyMarkdown",
      label: () => t("Copy page as Markdown"),
      group: () => t("Pages"),
      enabled: () => !!host?.view() && !!host.current(),
      run: () => {
        const v = host?.view();
        if (!v) return;
        const md = new MarkdownWriter((id) => engine.pages.get(id)?.title ?? "").doc(v.state.doc.content);
        void navigator.clipboard
          ?.writeText(md)
          .then(() => toast({ id: "copy-md", tone: "success", message: t("Markdown copied") }))
          .catch(engine.reportError);
      },
    }),
    keymap.register({ id: "page.copyMarkdown", command: "page.copyMarkdown", chords: ["mod+shift+c"], scope: "global", whileTyping: true }),
  );
}

const extension: Extension = {
  id: "blocks",
  setup,
  dispose() {
    for (const off of teardown) off();
    teardown = [];
  },
};

export default extension;
