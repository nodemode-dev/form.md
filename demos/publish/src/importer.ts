import { DOMParser as PMDOMParser, type Node as PMNode } from "prosemirror-model";
import { EditorState } from "prosemirror-state";
import { prosemirrorToYXmlFragment } from "y-prosemirror";
import type { Engine, Page } from "../../../web/src/engine";
import { BLOCKS_WITH_ID, newBlockId, ROOT, schema } from "../../../web/src/engine/doc/schema";
import { markdownReady, markdownToDOM } from "../../../web/src/extensions/blocks/markdown";
import { toast } from "../../../web/src/ui/toast";
import { pushLive } from "./platform";
import { pages, serverDoc, state } from "./server/db";
import { fresh } from "./server/ids";
import { WS_ID } from "./server/seed";

function withIds(doc: PMNode): PMNode {
  const tr = EditorState.create({ doc }).tr;
  doc.descendants((node, pos) => {
    if (BLOCKS_WITH_ID.has(node.type.name) && !node.attrs.id) tr.setNodeMarkup(pos, undefined, { ...node.attrs, id: newBlockId() });
  });
  return tr.doc;
}

function split(md: string, fallback: string): { title: string; body: string } {
  const m = /^\s*#\s+(.+?)\s*(?:\r?\n|$)/.exec(md);
  return m ? { title: m[1] as string, body: md.slice(m[0].length) } : { title: fallback, body: md };
}

export async function addMarkdownPage(title: string, md: string): Promise<Page> {
  await markdownReady();
  const parsed = PMDOMParser.fromSchema(schema).parse(markdownToDOM(md), { preserveWhitespace: false });
  const doc = withIds(parsed);
  const id = fresh("import");
  const ydoc = serverDoc(id);
  ydoc.transact(() => prosemirrorToYXmlFragment(doc, ydoc.getXmlFragment(ROOT)));
  const now = Date.now();
  const text = doc.textBetween(0, doc.content.size, " ", " ").trim();
  const page: Page = {
    id,
    workspace_id: WS_ID,
    parent_id: "",
    kind: "doc",
    sort_key: `c${now}`,
    title,
    icon: "",
    cover: "",
    description: "",
    tags: [],
    visibility: "workspace",
    owner_id: state.me!.id,
    is_template: false,
    snippet: text.slice(0, 160),
    thumb: "",
    created_at: now,
    created_by: state.me!.id,
    updated_at: now,
    updated_by: state.me!.id,
    access: "full",
  };
  pages.push(page);
  pushLive({ type: "page", seq: now, workspace_id: WS_ID, page });
  return page;
}

function readText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result ?? ""));
    r.onerror = () => reject(r.error ?? new Error("read failed"));
    r.readAsText(file);
  });
}

export async function importFiles(engine: Engine, files: File[]): Promise<void> {
  let last: Page | null = null;
  let skipped = 0;
  for (const file of files) {
    if (!/\.(md|markdown|mdown|txt)$/i.test(file.name) && !/^text\/(markdown|plain)$/.test(file.type)) {
      skipped++;
      continue;
    }
    const parts = split(await readText(file), file.name.replace(/\.[^.]+$/, ""));
    last = await addMarkdownPage(parts.title, parts.body);
  }
  if (skipped) toast({ tone: "warning", message: "This demo opens Markdown and text files." });
  if (last) {
    toast({ tone: "success", message: `Opened ${last.title || "your file"} as a page you can edit, publish and export.` });
    engine.openPage(last.id);
  }
}
