import { Fragment, type NodeType } from "prosemirror-model";
import type { EditorView } from "prosemirror-view";
import { embedProvider, newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import type { IconName } from "../../ui/icons";
import { newAIBlock, openAI } from "./ai";
import { newBoard } from "./board";
import { editButton, newButton } from "./button";
import { newMeeting } from "./meeting";
import { type BlockKind, insertBlock, turnInto } from "./commands";
import { caretRect, type EditorCtx } from "./context";
import { embedFor, mediaFromLink } from "./links";
import { insertInlineMath, newMathBlock } from "./math";
import { createSynced } from "./synced";
import { openPrompt } from "./popover";

export interface CatalogEntry {
  id: string;
  label: () => string;
  section: () => string;
  icon: IconName;
  keywords: string;
  hint?: string;
  turn?: BlockKind;
  run: (view: EditorView, ctx: EditorCtx) => void;
}

const N = schema.nodes as Record<string, NodeType>;
const BASIC = () => t("Basic blocks");
const MEDIA = () => t("Media");
const LAYOUT = () => t("Layout");
const LINKS = () => t("Links");
const AI = () => t("AI");
const ADVANCED = () => t("Actions");

const turn = (kind: BlockKind) => (view: EditorView) => {
  turnInto(view, kind);
};

function para(): ReturnType<NodeType["create"]> {
  return (N.paragraph as NodeType).create({ id: newBlockId() });
}

function askUrl(view: EditorView, label: string, placeholder: string, done: (url: string) => void): void {
  const at = view.state.selection.head;
  openPrompt({
    label,
    placeholder,
    action: t("Add"),
    rect: () => caretRect(view, Math.min(at, view.state.doc.content.size)),
    onSubmit: (raw) => {
      let url = raw.trim();
      if (!/^https?:\/\//i.test(url)) url = `https://${url}`;
      done(url);
    },
    onClose: () => view.focus(),
  });
}

export const CATALOG: CatalogEntry[] = [
  { id: "text", label: () => t("Text"), section: BASIC, icon: "text", keywords: "paragraph plain", turn: "paragraph", run: turn("paragraph") },
  { id: "h1", label: () => t("Heading 1"), section: BASIC, icon: "h1", keywords: "title big h1", hint: "#", turn: "h1", run: turn("h1") },
  { id: "h2", label: () => t("Heading 2"), section: BASIC, icon: "h2", keywords: "subtitle h2", hint: "##", turn: "h2", run: turn("h2") },
  { id: "h3", label: () => t("Heading 3"), section: BASIC, icon: "h3", keywords: "h3 small heading", hint: "###", turn: "h3", run: turn("h3") },
  { id: "bullet", label: () => t("Bulleted list"), section: BASIC, icon: "list-bullet", keywords: "unordered bullets ul", hint: "-", turn: "bullet_list", run: turn("bullet_list") },
  { id: "ordered", label: () => t("Numbered list"), section: BASIC, icon: "list-number", keywords: "ordered numbers ol", hint: "1.", turn: "ordered_list", run: turn("ordered_list") },
  { id: "task", label: () => t("To-do list"), section: BASIC, icon: "checkbox", keywords: "todo task checklist checkbox", hint: "[ ]", turn: "task_list", run: turn("task_list") },
  { id: "toggle", label: () => t("Toggle"), section: BASIC, icon: "chevron-right", keywords: "collapse details fold", turn: "toggle", run: turn("toggle") },
  {
    id: "toggle-heading",
    label: () => t("Toggle heading"),
    section: BASIC,
    icon: "chevron-right",
    keywords: "collapse section heading fold",
    run: (view) => {
      const node = (N.toggle as NodeType).create({ id: newBlockId(), level: 2 }, [(N.toggle_title as NodeType).create()]);
      insertBlock(view, node);
    },
  },
  { id: "quote", label: () => t("Quote"), section: BASIC, icon: "quote", keywords: "blockquote citation", hint: ">", turn: "blockquote", run: turn("blockquote") },
  { id: "callout", label: () => t("Callout"), section: BASIC, icon: "callout", keywords: "note tip warning info box", turn: "callout", run: turn("callout") },
  { id: "code", label: () => t("Code"), section: BASIC, icon: "code", keywords: "snippet pre programming", hint: "```", turn: "code_block", run: turn("code_block") },
  { id: "mermaid", label: () => t("Diagram"), section: BASIC, icon: "diagram", keywords: "mermaid flowchart chart graph sequence", turn: "mermaid", run: turn("mermaid") },
  {
    id: "divider",
    label: () => t("Divider"),
    section: BASIC,
    icon: "divider",
    keywords: "hr rule line separator",
    hint: "---",
    run: (view) => insertBlock(view, (N.horizontal_rule as NodeType).create({ id: newBlockId() })),
  },
  {
    id: "table",
    label: () => t("Table"),
    section: LAYOUT,
    icon: "table",
    keywords: "grid rows columns cells",
    run: (view) => {
      const cell = (head: boolean) => (head ? (N.table_header as NodeType) : (N.table_cell as NodeType)).create(null, [para()]);
      const row = (head: boolean) => (N.table_row as NodeType).create(null, [cell(head), cell(head), cell(head)]);
      insertBlock(view, (N.table as NodeType).create({ id: newBlockId() }, [row(true), row(false), row(false)]));
    },
  },
  {
    id: "columns2",
    label: () => t("Two columns"),
    section: LAYOUT,
    icon: "columns",
    keywords: "columns side by side layout split",
    run: (view) => insertBlock(view, (N.columns as NodeType).create({ id: newBlockId() }, [(N.column as NodeType).create(null, [para()]), (N.column as NodeType).create(null, [para()])])),
  },
  {
    id: "columns3",
    label: () => t("Three columns"),
    section: LAYOUT,
    icon: "columns",
    keywords: "columns layout split three",
    run: (view) =>
      insertBlock(view, (N.columns as NodeType).create({ id: newBlockId() }, [(N.column as NodeType).create(null, [para()]), (N.column as NodeType).create(null, [para()]), (N.column as NodeType).create(null, [para()])])),
  },
  {
    id: "synced",
    label: () => t("Synced block"),
    section: LAYOUT,
    icon: "refresh",
    keywords: "synced sync mirror shared reuse transclude same content",
    run: (view, ctx) => {
      const node = createSynced(ctx, Fragment.empty);
      if (node) insertBlock(view, node);
    },
  },
  { id: "toc", label: () => t("Table of contents"), section: LAYOUT, icon: "toc", keywords: "toc outline headings contents", run: (view) => insertBlock(view, (N.toc as NodeType).create({ id: newBlockId() })) },
  { id: "math", label: () => t("Block equation"), section: LAYOUT, icon: "sigma", keywords: "math equation latex tex formula katex block", run: (view) => insertBlock(view, newMathBlock()) },
  { id: "math-inline", label: () => t("Inline equation"), section: LAYOUT, icon: "sigma", hint: "Ctrl+Shift+E", keywords: "math equation latex tex formula inline", run: (view) => void insertInlineMath(view) },
  { id: "breadcrumb", label: () => t("Breadcrumb"), section: LAYOUT, icon: "turn", keywords: "breadcrumb bread path parents trail", run: (view) => insertBlock(view, (N.breadcrumb as NodeType).create({ id: newBlockId() })) },
  { id: "image", label: () => t("Image"), section: MEDIA, icon: "image", keywords: "picture photo png jpg upload", run: (view, ctx) => ctx.pickFiles(view, "image/*") },
  { id: "video", label: () => t("Video"), section: MEDIA, icon: "video", keywords: "movie mp4 webm upload", run: (view, ctx) => ctx.pickFiles(view, "video/*") },
  { id: "audio", label: () => t("Audio"), section: MEDIA, icon: "audio", keywords: "sound music mp3 recording", run: (view, ctx) => ctx.pickFiles(view, "audio/*") },
  { id: "file", label: () => t("File"), section: MEDIA, icon: "attachment", keywords: "attachment pdf upload document", run: (view, ctx) => ctx.pickFiles(view, "") },
  {
    id: "bookmark",
    label: () => t("Link card"),
    section: LINKS,
    icon: "link",
    keywords: "bookmark url web preview",
    run: (view) =>
      askUrl(view, t("Link card"), t("Paste a link"), (url) => {
        const provider = embedProvider(url);
        insertBlock(view, provider ? (N.embed as NodeType).create({ id: newBlockId(), url, provider }) : (N.bookmark as NodeType).create({ id: newBlockId(), url }));
      }),
  },
  {
    id: "embed",
    label: () => t("Embed"),
    section: LINKS,
    icon: "globe",
    keywords: "embed iframe youtube vimeo loom figma miro codepen codesandbox spotify soundcloud google docs drive maps airtable typeform excalidraw tiktok canva behance sketchfab pinterest apple music website",
    run: (view, ctx) => askUrl(view, t("Embed"), t("Paste any link to show it inside the page"), (url) => void embedFor(ctx, url).then((node) => insertBlock(view, node))),
  },
  {
    id: "media-link",
    label: () => t("Media from a link"),
    section: MEDIA,
    icon: "image",
    keywords: "image video audio pdf url link import web picture",
    run: (view) => askUrl(view, t("Media from a link"), t("Paste a link to an image, video, audio or PDF file"), (url) => insertBlock(view, mediaFromLink(url))),
  },
  {
    id: "board",
    label: () => t("Board"),
    section: MEDIA,
    icon: "pin",
    keywords: "board moodboard mood inspiration references pin pinboard collage canvas notes pictures links",
    run: (view) => insertBlock(view, newBoard()),
  },
  {
    id: "page",
    label: () => t("Page inside"),
    section: LINKS,
    icon: "page",
    keywords: "subpage child new page",
    run: (view, ctx) => {
      const ws = ctx.workspaceId();
      const parent = ctx.pageId();
      if (!ws || !parent) return;
      void ctx.engine.pages
        .create(ws, parent)
        .then((p) => {
          const mention = (N.mention as NodeType).create({ kind: "page", ref: p.id, label: "" });
          insertBlock(view, (N.paragraph as NodeType).create({ id: newBlockId() }, [mention]));
        })
        .catch(ctx.engine.reportError);
    },
  },
  {
    id: "mention-page",
    label: () => t("Link to a page"),
    section: LINKS,
    icon: "at",
    keywords: "mention reference link page",
    hint: "[[",
    run: (view) => {
      view.dispatch(view.state.tr.insertText("[["));
    },
  },
  {
    id: "ask-ai",
    label: () => t("Ask AI"),
    section: AI,
    icon: "sparkle",
    keywords: "ai write continue summarize translate assistant generate draft",
    run: (view, ctx) => {
      requestAnimationFrame(() => openAI(ctx, view));
    },
  },
  {
    id: "ai-block",
    label: () => t("AI block"),
    section: AI,
    icon: "sparkle",
    keywords: "ai block generate prompt regenerate summary",
    run: (view) => insertBlock(view, newAIBlock()),
  },
  {
    id: "meeting",
    label: () => t("Meeting notes"),
    section: AI,
    icon: "audio",
    keywords: "meeting notes record transcribe transcript call summary audio",
    run: (view) => insertBlock(view, newMeeting()),
  },
  {
    id: "button",
    label: () => t("Button"),
    section: ADVANCED,
    icon: "button",
    keywords: "button action automation click insert add page notify webhook",
    run: (view, ctx) => placeButton(view, ctx, newButton(t("New button"))),
  },
  {
    id: "template-button",
    label: () => t("Template button"),
    section: ADVANCED,
    icon: "button",
    keywords: "template button repeat insert checklist duplicate blocks",
    run: (view, ctx) => placeButton(view, ctx, newButton(t("Add a checklist"), [{ kind: "insert", where: "below", markdown: "- [ ] To do on {{date}}" }])),
  },
];

function placeButton(view: EditorView, ctx: EditorCtx, node: ReturnType<typeof newButton>): void {
  insertBlock(view, node);
  const id = String(node.attrs.id);
  requestAnimationFrame(() => {
    let at: number | undefined;
    view.state.doc.descendants((n, pos) => {
      if (at === undefined && n.type.name === "button" && n.attrs.id === id) at = pos;
      return at === undefined;
    });
    if (at !== undefined) editButton(ctx, view, () => at);
  });
}

export function filterCatalog(query: string, entries: CatalogEntry[] = CATALOG): CatalogEntry[] {
  const q = query.trim().toLowerCase();
  if (!q) return entries;
  const scored: { e: CatalogEntry; s: number }[] = [];
  for (const e of entries) {
    const label = e.label().toLowerCase();
    const words = `${label} ${e.keywords} ${e.id}`;
    let s = -1;
    if (label.startsWith(q)) s = 0;
    else if (label.split(/\s+/).some((w) => w.startsWith(q))) s = 1;
    else if (words.split(/\s+/).some((w) => w.startsWith(q))) s = 2;
    else if (words.includes(q)) s = 3;
    if (s >= 0) scored.push({ e, s });
  }
  return scored.sort((a, b) => a.s - b.s).map((x) => x.e);
}
