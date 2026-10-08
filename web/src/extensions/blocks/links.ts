import type { Node as PMNode, NodeType, Slice } from "prosemirror-model";
import { type EditorState, Plugin, PluginKey, type Transaction } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";
import { ySyncPluginKey } from "y-prosemirror";
import type { FileInfo } from "../../engine";
import { embedProvider, isRemote, newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { h } from "../../ui/dom";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { toast } from "../../ui/toast";
import { caretRect, type EditorCtx } from "./context";

const N = schema.nodes as Record<string, NodeType>;
const MEDIA = new Set(["image", "video", "audio"]);
const MEDIA_EXT = /\.(?:png|jpe?g|gif|webp|avif|svg|mp4|webm|mov|m4v|mp3|m4a|ogg|oga|wav|flac|pdf)(?:[?#]|$)/i;

type Phase = "importing" | "failed";
const phases = new Map<string, Phase>();
const watchers = new Map<string, Set<() => void>>();
const unfurled = new Set<string>();

function setPhase(id: string, p: Phase | null): void {
  if (p) phases.set(id, p);
  else phases.delete(id);
  for (const fn of watchers.get(id) ?? []) fn();
}

export function importPhase(id: string): Phase | null {
  return phases.get(id) ?? null;
}

export function watchImport(id: string, fn: () => void): () => void {
  let set = watchers.get(id);
  if (!set) watchers.set(id, (set = new Set()));
  set.add(fn);
  return () => {
    set.delete(fn);
    if (!set.size) watchers.delete(id);
  };
}

function findBlock(state: EditorState, id: string): { node: PMNode; pos: number } | null {
  let out: { node: PMNode; pos: number } | null = null;
  state.doc.descendants((node, pos) => {
    if (out) return false;
    if (node.attrs.id === id) {
      out = { node, pos };
      return false;
    }
    return true;
  });
  return out;
}

function stem(name: string): string {
  return name.replace(/\.[^.]+$/, "");
}

function mediaNode(info: FileInfo, keep: PMNode): PMNode {
  const id = String(keep.attrs.id || newBlockId());
  const caption = String(keep.attrs.caption ?? "");
  if (info.mime.startsWith("image/") && info.mime !== "image/svg+xml")
    return (N.image as NodeType).create({ id, src: info.ref, alt: String(keep.attrs.alt || stem(info.name)), caption, w: info.w || keep.attrs.w || null, h: info.h || keep.attrs.h || null });
  if (info.mime.startsWith("video/")) return (N.video as NodeType).create({ id, src: info.ref, caption });
  if (info.mime.startsWith("audio/")) return (N.audio as NodeType).create({ id, src: info.ref, caption });
  return (N.file as NodeType).create({ id, src: info.ref, name: info.name, size: info.size, mime: info.mime });
}

function dataBlob(src: string): Blob | null {
  const m = /^data:([\w/+.-]+);base64,(.*)$/s.exec(src);
  if (!m?.[1] || !m[2]) return null;
  try {
    const bin = atob(m[2]);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return new Blob([bytes], { type: m[1] });
  } catch {
    return null;
  }
}

export function importMedia(ctx: EditorCtx, view: EditorView, id: string, loud = false): void {
  if (phases.get(id) === "importing" || !ctx.writable()) return;
  const found = findBlock(view.state, id);
  const pageId = ctx.pageId();
  if (!found || !pageId) return;
  const src = String(found.node.attrs.src ?? "");
  let job: Promise<FileInfo>;
  if (src.startsWith("data:")) {
    const blob = dataBlob(src);
    if (!blob) return setPhase(id, "failed");
    const ext = blob.type.split("/")[1]?.replace("jpeg", "jpg").replace(/\W.*$/, "") || "bin";
    job = ctx.engine.api.files.upload(blob, "attachment", { page_id: pageId, name: `pasted.${ext}` }).then((r) => r.file);
  } else if (isRemote(src)) job = ctx.engine.api.files.importUrl(src, pageId).then((r) => r.file);
  else return;
  setPhase(id, "importing");
  void job
    .then((info) => {
      setPhase(id, null);
      if (view.isDestroyed) return;
      const at = findBlock(view.state, id);
      if (!at || String(at.node.attrs.src ?? "") !== src) return;
      view.dispatch(view.state.tr.replaceWith(at.pos, at.pos + at.node.nodeSize, mediaNode(info, at.node)));
    })
    .catch((e: unknown) => {
      setPhase(id, "failed");
      if (loud) ctx.engine.reportError(e);
    });
}

export function unfurlBookmark(ctx: EditorCtx, view: EditorView, id: string): void {
  const found = findBlock(view.state, id);
  const pageId = ctx.pageId();
  if (!found || !pageId || !ctx.writable() || unfurled.has(id)) return;
  const url = String(found.node.attrs.url ?? "");
  if (!isRemote(url)) return;
  unfurled.add(id);
  view.dispatch(view.state.tr.setNodeMarkup(found.pos, undefined, { ...found.node.attrs, fetched: true }).setMeta("addToHistory", false));
  void ctx.engine.api.files
    .unfurl(url, pageId)
    .then((pv) => {
      if (view.isDestroyed) return;
      const at = findBlock(view.state, id);
      if (!at || at.node.type !== N.bookmark || String(at.node.attrs.url) !== url) return;
      const attrs = { ...at.node.attrs, title: pv.title, description: pv.description, site: pv.site, image: pv.image || at.node.attrs.image, icon: pv.icon || at.node.attrs.icon, fetched: true };
      view.dispatch(view.state.tr.setNodeMarkup(at.pos, undefined, attrs).setMeta("addToHistory", false));
    })
    .catch(() => undefined);
}

function touches(tr: Transaction): boolean {
  for (const step of tr.steps) {
    const slice = (step as { slice?: Slice }).slice;
    if (!slice) continue;
    let hit = false;
    slice.content.descendants((n) => {
      if (hit) return false;
      if (MEDIA.has(n.type.name) || n.type === N.bookmark) hit = true;
      return !hit;
    });
    if (hit) return true;
  }
  return false;
}

function scan(ctx: EditorCtx, view: EditorView): void {
  if (view.isDestroyed || !ctx.writable()) return;
  view.state.doc.descendants((n) => {
    const id = String(n.attrs.id ?? "");
    if (!id) return true;
    if (MEDIA.has(n.type.name)) {
      const src = String(n.attrs.src ?? "");
      if ((isRemote(src) || src.startsWith("data:")) && !phases.has(id)) importMedia(ctx, view, id);
      return false;
    }
    if (n.type === N.bookmark) {
      if (!n.attrs.fetched && !n.attrs.title && n.attrs.url) unfurlBookmark(ctx, view, id);
      return false;
    }
    return true;
  });
}

export function linksPlugin(ctx: EditorCtx): Plugin {
  let dirty = true;
  let timer = 0;
  const later = (view: EditorView) => {
    if (timer) return;
    timer = window.setTimeout(() => {
      timer = 0;
      if (!dirty) return;
      dirty = false;
      scan(ctx, view);
    }, 60);
  };
  return new Plugin({
    key: new PluginKey("formmd-links"),
    appendTransaction(trs) {
      for (const tr of trs) {
        if (!tr.docChanged || (tr.getMeta(ySyncPluginKey) as { isChangeOrigin?: boolean } | undefined)?.isChangeOrigin) continue;
        if (touches(tr)) dirty = true;
      }
      return null;
    },
    view(view) {
      later(view);
      return {
        update(v) {
          if (dirty) later(v);
        },
        destroy() {
          clearTimeout(timer);
        },
      };
    },
  });
}

export function looksLikeMedia(url: string): boolean {
  try {
    return MEDIA_EXT.test(new URL(url).pathname);
  } catch {
    return false;
  }
}

function replaceBlock(view: EditorView, pos: number, node: PMNode): void {
  const cur = view.state.doc.nodeAt(pos);
  if (!cur) return;
  view.dispatch(view.state.tr.replaceWith(pos, pos + cur.nodeSize, node).scrollIntoView());
}

export function mediaFromLink(url: string): PMNode {
  return (N.image as NodeType).create({ id: newBlockId(), src: url });
}

export function bookmarkFor(url: string): PMNode {
  return (N.bookmark as NodeType).create({ id: newBlockId(), url });
}

export async function embedFor(ctx: EditorCtx, url: string): Promise<PMNode> {
  const provider = embedProvider(url);
  if (provider) return (N.embed as NodeType).create({ id: newBlockId(), url, provider });
  const pv = await ctx.engine.api.files.unfurl(url, ctx.pageId()).catch(() => null);
  if (pv?.frameable && pv.kind === "page" && url.startsWith("https://")) return (N.embed as NodeType).create({ id: newBlockId(), url, provider: "web" });
  if (pv && pv.kind !== "page") return mediaFromLink(url);
  toast({ id: "embed-refused", tone: "info", message: t("That site does not allow embedding, so it was added as a link card.") });
  return bookmarkFor(url);
}

let anchor: HTMLElement | null = null;

export function offerLinkChoice(ctx: EditorCtx, view: EditorView, blockPos: number, url: string, wrap: HTMLElement): void {
  const rect = caretRect(view);
  if (!rect) return;
  if (!anchor || !anchor.isConnected) {
    anchor = h("span", { class: "paste-anchor", attrs: { "aria-hidden": "true" } });
    wrap.append(anchor);
  }
  const box = wrap.getBoundingClientRect();
  anchor.style.transform = `translate(${Math.round(rect.x - box.left)}px, ${Math.round(rect.y + rect.height - box.top)}px)`;
  const still = () => {
    const node = view.state.doc.nodeAt(blockPos);
    return !!node && node.type === N.paragraph && node.textContent === url;
  };
  const swap = (make: () => PMNode | Promise<PMNode>) => () => {
    if (!still()) return;
    void Promise.resolve(make()).then((node) => {
      if (still()) replaceBlock(view, blockPos, node);
      view.focus();
    });
  };
  const entries: MenuEntry[] = [{ kind: "heading", label: t("Paste as") }];
  const media: MenuEntry = { label: t("Media from the link"), icon: "image", run: swap(() => mediaFromLink(url)) };
  if (looksLikeMedia(url)) entries.push(media);
  entries.push(
    { label: t("Link"), icon: "link", run: () => view.focus() },
    { label: t("Link card"), icon: "cards", run: swap(() => bookmarkFor(url)) },
    { label: t("Embed"), icon: "globe", run: swap(() => embedFor(ctx, url)) },
  );
  openMenu(anchor, entries, { side: "bottom", align: "start", label: t("Paste as") });
}
