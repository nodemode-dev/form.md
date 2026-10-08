import type { NodeType, Node as PMNode } from "prosemirror-model";
import type { EditorView } from "prosemirror-view";
import { ApiError } from "../../engine/api";
import { newBlockId, schema } from "../../engine/doc/schema";
import type { FileInfo } from "../../engine";
import { t } from "../../engine/i18n";
import { h, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import type { EditorCtx } from "./context";
import { type UploadMeta, uploadKey, uploadPos } from "./plugins";

const N = schema.nodes as Record<string, NodeType>;

function upload(form: FormData, onProgress: (p: number) => void): Promise<FileInfo> {
  return new Promise((resolve, reject) => {
    const x = new XMLHttpRequest();
    x.open("POST", "/api/files");
    x.responseType = "json";
    x.setRequestHeader("Accept", "application/json");
    x.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total);
    };
    x.onload = () => {
      const body = x.response as { file?: FileInfo; error?: { code?: string; message?: string } } | null;
      if (x.status >= 200 && x.status < 300 && body?.file) resolve(body.file);
      else reject(new ApiError(x.status, body?.error?.code ?? "http_error", body?.error?.message ?? `HTTP ${x.status}`));
    };
    x.onerror = () => reject(new ApiError(0, "network", "network unreachable"));
    x.ontimeout = () => reject(new ApiError(0, "timeout", "request timed out"));
    x.timeout = 30 * 60 * 1000;
    x.send(form);
  });
}

async function dims(file: File): Promise<{ w: number; h: number } | null> {
  if (!file.type.startsWith("image/") || typeof createImageBitmap !== "function") return null;
  try {
    const bmp = await createImageBitmap(file);
    const out = { w: bmp.width, h: bmp.height };
    bmp.close();
    return out;
  } catch {
    return null;
  }
}

function boundary(view: EditorView, at: number): number {
  const doc = view.state.doc;
  const $pos = doc.resolve(Math.max(0, Math.min(at, doc.content.size)));
  for (let d = $pos.depth; d > 0; d--) {
    const parent = $pos.node(d - 1);
    if (parent.type.contentMatch.matchType(N.image as NodeType)) {
      const node = $pos.node(d);
      return node.isTextblock && node.content.size === 0 ? $pos.before(d) : $pos.after(d);
    }
  }
  return doc.content.size;
}

function nodeFor(file: FileInfo, size: { w: number; h: number } | null): PMNode {
  const src = `f:${file.id}`;
  const id = newBlockId();
  if (file.mime.startsWith("image/") && file.mime !== "image/svg+xml") return (N.image as NodeType).create({ id, src, alt: file.name.replace(/\.[^.]+$/, ""), w: size?.w ?? null, h: size?.h ?? null });
  if (file.mime.startsWith("video/")) return (N.video as NodeType).create({ id, src });
  if (file.mime.startsWith("audio/")) return (N.audio as NodeType).create({ id, src });
  return (N.file as NodeType).create({ id, src, name: file.name, size: file.size, mime: file.mime });
}

let seq = 0;

export function uploadFiles(ctx: EditorCtx, view: EditorView, files: File[], at?: number): void {
  const pageId = ctx.pageId();
  if (!pageId || !ctx.writable()) return;
  const pos = boundary(view, at ?? view.state.selection.from);
  for (const file of files) {
    const id = `up${++seq}`;
    const bar = h("progress", { class: "upload__bar", attrs: { max: "100", value: "0" } });
    const label = h("span", { class: "upload__name", text: file.name || t("File") });
    const el = h("div", { class: "upload", attrs: { contenteditable: "false", role: "status", "aria-label": t("Uploading {name}", { name: file.name }) } }, iconNode("upload"), label, bar);
    view.dispatch(view.state.tr.setMeta(uploadKey, { add: { id, pos, el } } satisfies UploadMeta));
    const form = new FormData();
    form.append("purpose", "attachment");
    form.append("page_id", pageId);
    form.append("name", file.name);
    form.append("file", file);
    const sized = dims(file);
    void upload(form, (p) => {
      bar.value = Math.round(p * 100);
    })
      .then(async (info) => {
        const size = await sized;
        if (view.isDestroyed) return;
        const at = uploadPos(view.state, id);
        const tr = view.state.tr.setMeta(uploadKey, { remove: id } satisfies UploadMeta);
        if (at !== null) {
          try {
            tr.insert(at, nodeFor(info, size));
          } catch {
            tr.insert(tr.doc.content.size, nodeFor(info, size));
          }
        }
        view.dispatch(tr);
      })
      .catch((e: unknown) => {
        if (!view.isDestroyed) view.dispatch(view.state.tr.setMeta(uploadKey, { remove: id } satisfies UploadMeta));
        setText(label, file.name);
        ctx.engine.reportError(e);
      });
  }
}

export function pickFiles(ctx: EditorCtx, view: EditorView, accept: string, at?: number): void {
  const input = h("input", { attrs: { type: "file", accept, multiple: "multiple" } });
  input.hidden = true;
  const where = at ?? view.state.selection.from;
  input.addEventListener("change", () => {
    const files = Array.from(input.files ?? []);
    input.remove();
    if (files.length) uploadFiles(ctx, view, files, where);
  });
  document.body.append(input);
  input.click();
}
