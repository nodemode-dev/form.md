import { DOMParser as PMDOMParser, type Node as PMNode } from "prosemirror-model";
import { EditorState } from "prosemirror-state";
import { prosemirrorToYXmlFragment, yXmlFragmentToProseMirrorRootNode } from "y-prosemirror";
import * as Y from "yjs";
import { BLOCKS_WITH_ID, newBlockId, ROOT, schema } from "../../web/src/engine/doc/schema";
import type { Property, RowData } from "../../web/src/engine/types";
import { MarkdownWriter, markdownReady, markdownToDOM } from "../../web/src/extensions/blocks/markdown";
import { buildTable, parseDelimited, type Table } from "./csv";
import { freshView, type ViewState } from "./query";
import { EXOPLANETS, PAGES } from "./content";
import { loadPack } from "./packs";
import { trailCsv } from "./samples";

export interface PageItem {
  kind: "page";
  id: string;
  title: string;
  icon: string;
  cover: string;
  ydoc: Y.Doc;
  edited: number;
}

export interface DataItem {
  kind: "collection";
  id: string;
  title: string;
  icon: string;
  cover: string;
  schema: Property[];
  rows: RowData[];
  view: ViewState;
  edited: number;
}

export type Item = PageItem | DataItem;

const items: Item[] = [];
const listeners = new Set<() => void>();

export function subscribe(fn: () => void): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

export function emit(): void {
  for (const fn of listeners) fn();
}

export function list(): readonly Item[] {
  return items;
}

export function get(id: string): Item | undefined {
  return items.find((i) => i.id === id);
}

function withIds(doc: PMNode): PMNode {
  const tr = EditorState.create({ doc }).tr;
  doc.descendants((node, pos) => {
    if (BLOCKS_WITH_ID.has(node.type.name) && !node.attrs.id) tr.setNodeMarkup(pos, undefined, { ...node.attrs, id: newBlockId() });
  });
  return tr.doc;
}

export async function docFromMarkdown(md: string): Promise<PMNode> {
  await markdownReady();
  const parsed = PMDOMParser.fromSchema(schema).parse(markdownToDOM(md), { preserveWhitespace: false });
  return withIds(parsed);
}

export function docOf(page: PageItem): PMNode {
  return yXmlFragmentToProseMirrorRootNode(page.ydoc.getXmlFragment(ROOT), schema);
}

export function bodyMarkdown(page: PageItem): string {
  return new MarkdownWriter().doc(docOf(page).content);
}

export function fullMarkdown(page: PageItem): string {
  const body = bodyMarkdown(page);
  return page.title ? `# ${page.title}\n\n${body}` : body;
}

export async function addPage(title: string, md: string, icon = "", cover = ""): Promise<PageItem> {
  const doc = await docFromMarkdown(md);
  const ydoc = new Y.Doc();
  ydoc.transact(() => {
    prosemirrorToYXmlFragment(doc, ydoc.getXmlFragment(ROOT));
  });
  const page: PageItem = { kind: "page", id: newBlockId(), title, icon, cover, ydoc, edited: Date.now() };
  items.push(page);
  emit();
  return page;
}

export function addCollection(title: string, table: Table, icon = "", cover = ""): DataItem {
  const data: DataItem = { kind: "collection", id: newBlockId(), title, icon, cover, schema: table.schema, rows: table.rows, view: freshView(), edited: Date.now() };
  const board = table.schema.find((p) => p.type === "status") ?? table.schema.find((p) => p.type === "select");
  if (board) data.view.boardBy = board.id;
  items.push(data);
  emit();
  return data;
}

export function removeItem(id: string): void {
  const i = items.findIndex((x) => x.id === id);
  if (i < 0) return;
  items.splice(i, 1);
  emit();
}

export function touch(item: Item): void {
  item.edited = Date.now();
}

export function splitTitle(md: string, fallback: string): { title: string; body: string } {
  const m = /^\s*#\s+(.+?)\s*(?:\r?\n|$)/.exec(md);
  if (!m) return { title: fallback, body: md };
  return { title: m[1] as string, body: md.slice(m[0].length) };
}

export function baseName(file: string): string {
  return file.replace(/\.[^.]+$/, "");
}

export async function seed(): Promise<string> {
  items.length = 0;
  const csv = await loadPack(EXOPLANETS.pack, EXOPLANETS.file);
  let first = "";
  for (const p of PAGES) {
    const page = await addPage(p.title, p.md, p.icon, p.cover);
    first ||= page.id;
  }
  if (csv) {
    const data = addCollection(EXOPLANETS.title, buildTable(parseDelimited(csv)), "i:globe", EXOPLANETS.cover);
    data.view.calcs = { p7: "median", p9: "min" };
  } else addCollection("Trail survey", buildTable(parseDelimited(trailCsv())));
  return first;
}
