import { yXmlFragmentToProseMirrorRootNode } from "y-prosemirror";
import { ROOT, schema } from "../../../../web/src/engine/doc/schema";
import { MarkdownWriter } from "../../../../web/src/extensions/blocks/markdown";
import type { Page } from "../../../../web/src/engine/types";
import { serverDoc } from "./db";
import { FILES, type FileRec } from "./files";
import { loadPictures, pictureBytes } from "./packs";
import { collect, exportSrc, renderPage, type Ctx } from "./site";
import { zip, type Entry } from "./zip";

const ORIGIN = "https://fielddesk.example";

const safe = (s: string): string => (s.trim() || "Untitled").replace(/[\\/:*?"<>|]+/g, "-");

function fileOf(n: Page, all: Page[]): string {
  const parent = n.parent_id ? all.find((p) => p.id === n.parent_id) : undefined;
  return parent ? `${fileOf(parent, all)}/${safe(n.title)}` : safe(n.title);
}

function rel(from: string, to: string): string {
  const depth = from.split("/").length - 1;
  return `${"../".repeat(depth)}${to}`;
}

function copyPath(f: FileRec): string {
  return `files/${f.id.slice(0, 8)}-${f.name}${f.ext}`;
}

function byPath(path: string): FileRec | undefined {
  return [...FILES.values()].find((f) => f.path === path);
}

export function saveBlob(name: string, blob: Blob): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.append(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 4000);
}

function markdownFor(n: Page, all: Page[], used: Map<string, FileRec>): string {
  const from = `${fileOf(n, all)}.md`;
  const doc = yXmlFragmentToProseMirrorRootNode(serverDoc(n.id).getXmlFragment(ROOT), schema);
  let md = new MarkdownWriter((id) => all.find((p) => p.id === id)?.title ?? "").doc(doc.content);
  md = md.replace(/\]\(\/p\/([a-z2-7]+)((?:#[^)\s]*)?)([ )])/g, (_m, id: string, frag: string, tail: string) => {
    const target = all.find((p) => p.id === id);
    return target ? `](<${rel(from, `${fileOf(target, all)}.md`)}>${tail}` : `](${ORIGIN}/p/${id}${frag}${tail}`;
  });
  md = md.replace(/\]\((assets\/media\/[a-z0-9-]+\.webp)([ )])/g, (m, path: string, tail: string) => {
    const f = byPath(path);
    if (!f) return m;
    used.set(f.id, f);
    return `](<${rel(from, copyPath(f))}>${tail}`;
  });
  const head = `# ${n.title || "Untitled"}\n\n${n.description ? `${n.description}\n\n` : ""}`;
  return head + md;
}

export async function exportMarkdown(rootId: string, subpages: boolean): Promise<{ name: string; blob: Blob }> {
  await loadPictures();
  const all = collect(rootId, subpages);
  const used = new Map<string, FileRec>();
  const enc = new TextEncoder();
  const entries: Entry[] = all.map((n) => ({ name: `${fileOf(n, all)}.md`, data: enc.encode(markdownFor(n, all, used)) }));
  for (const f of used.values()) entries.push({ name: copyPath(f), data: pictureBytes(f.name) });
  const root = all[0] as Page;
  return { name: `${safe(root.title)}.zip`, blob: zip(entries) };
}

function context(all: Page[], root: Page, current: string | null, used: Map<string, FileRec>): Ctx {
  const single = current === null;
  return {
    mode: "export",
    root: root.id,
    nodes: all,
    view: { theme: "auto", font: "sans", indexing: false, title: "", description: "" },
    origin: ORIGIN,
    href: (id) => {
      const target = all.find((p) => p.id === id);
      if (single || !target) return `${ORIGIN}/p/${id}`;
      return rel(`${current}.html`, `${fileOf(target, all)}.html`);
    },
    src: (path) => {
      if (single) return exportSrc(path);
      const f = byPath(path);
      if (!f) return path;
      used.set(f.id, f);
      return rel(`${current}.html`, copyPath(f));
    },
  };
}

export async function exportHtml(rootId: string, subpages: boolean): Promise<{ name: string; blob: Blob }> {
  await loadPictures();
  const all = collect(rootId, subpages);
  const root = all[0] as Page;
  const enc = new TextEncoder();
  if (all.length === 1) {
    const html = renderPage(root.id, context(all, root, null, new Map()));
    return { name: `${safe(root.title)}.html`, blob: new Blob([html], { type: "text/html" }) };
  }
  const used = new Map<string, FileRec>();
  const entries: Entry[] = all.map((n) => {
    const file = fileOf(n, all);
    return { name: `${file}.html`, data: enc.encode(renderPage(n.id, context(all, root, file, used))) };
  });
  for (const f of used.values()) entries.push({ name: copyPath(f), data: pictureBytes(f.name) });
  return { name: `${safe(root.title)}.zip`, blob: zip(entries) };
}
