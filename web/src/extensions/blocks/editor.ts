import { Fragment, type Node as PMNode, type ResolvedPos, type Slice } from "prosemirror-model";
import { EditorState, Plugin, PluginKey } from "prosemirror-state";
import { EditorView } from "prosemirror-view";
import { initProseMirrorDoc, relativePositionToAbsolutePosition, yCursorPlugin, ySyncPlugin, ySyncPluginKey, yUndoPlugin } from "y-prosemirror";
import * as Y from "yjs";
import type { Engine } from "../../engine";
import type { DocSession } from "../../engine/doc/docs";
import { embedProvider, newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { h } from "../../ui/dom";
import type { EditorCtx, Refreshable } from "./context";
import { FormatBar } from "./formatbar";
import { BlockHandle } from "./handle";
import { looksLikeMarkdown, MarkdownWriter, parseMarkdownSlice, preloadMarkdown } from "./markdown";
import { pickFiles, uploadFiles } from "./media";
import { nodeViews } from "./nodeviews";
import { highlightPlugin, idsPlugin, keysPlugin, livePlugin, placeholderPlugin, structurePlugins, uploadsPlugin } from "./plugins";
import { columnsPlugins } from "./columns";
import { commentsPlugin } from "./comments";
import { suggestKey, suggestPlugin, suggestUI } from "./suggest";
import { linksPlugin, offerLinkChoice } from "./links";
import { mathKeys } from "./math";
import { rulesPlugin } from "./rules";
import { triggerPlugin } from "./triggers";

const URL_ONLY = /^https?:\/\/\S+$/i;

export type EditMode = "edit" | "suggest" | "view";

function cursorBuilder(user: { name?: string; tint?: string }): HTMLElement {
  const label = h("span", { class: "collab-cursor__label", text: user.name ?? "" });
  const el = h("span", { class: "collab-cursor" }, document.createTextNode("⁠"), label, document.createTextNode("⁠"));
  el.dataset.color = user.tint ?? "violet";
  return el;
}

function selectionBuilder(user: { tint?: string }): { class: string; "data-color": string } {
  return { class: "collab-selection", "data-color": user.tint ?? "violet" };
}

export class EditorHost implements EditorCtx {
  readonly openToggles = new Set<string>();
  readonly live = new Set<Refreshable>();
  private ed: EditorView | null = null;
  private session: DocSession | null = null;
  private wanted = "";
  private unsub: (() => void) | null = null;
  private handle: BlockHandle;
  private bar: FormatBar;
  private md = new MarkdownWriter((id) => this.engine.pages.get(id)?.title ?? "");
  private notice: HTMLElement | null = null;
  private listeners = new Set<(id: string) => void>();
  private modeListeners = new Set<(m: EditMode) => void>();
  private mode: EditMode = "edit";
  private snaps = new Map<string, string>();

  constructor(
    readonly engine: Engine,
    readonly mount: HTMLElement,
    readonly wrap: HTMLElement,
  ) {
    this.handle = new BlockHandle(this, wrap);
    this.bar = new FormatBar(this);
    engine.pages.subscribe((c) => {
      if (c.kind === "upsert" || c.kind === "remove" || c.kind === "loaded") for (const v of this.live) v.refresh();
    });
    const recheck = () => {
      if (this.ed && this.session?.phase === "ready") this.ed.setProps({});
    };
    engine.state.watch("me", recheck);
    engine.state.watch("detail", recheck);
    engine.comments.subscribe((id) => {
      if (id === this.session?.id) recheck();
    });
    mount.addEventListener("pointerover", (e) => {
      const m = (e.target as Element).closest?.<HTMLElement>(".mention[data-kind=page]");
      if (m?.dataset.ref) engine.docs.prefetch(m.dataset.ref);
    });
  }

  view(): EditorView | null {
    return this.ed;
  }

  formatBar(): FormatBar {
    return this.bar;
  }

  pageId(): string {
    return this.session?.id ?? "";
  }

  workspaceId(): string {
    const id = this.pageId();
    const p = this.engine.pages.get(id);
    const d = this.engine.state.get("detail");
    return p?.workspace_id ?? (d && d.id === id ? d.workspace_id : this.engine.state.get("workspaceId"));
  }

  writable(): boolean {
    return this.mode === "edit" && this.canEdit();
  }

  canEdit(): boolean {
    const s = this.session;
    if (!s || s.phase !== "ready" || !s.write) return false;
    const d = this.engine.state.get("detail");
    if (d && d.id === s.id && !d.can_write) return false;
    return this.engine.can(s.id, "content");
  }

  canSuggest(): boolean {
    const s = this.session;
    return !!s && s.phase === "ready" && this.engine.comments.get(s.id).can_comment;
  }

  suggesting(): boolean {
    return this.mode === "suggest" && this.canSuggest();
  }

  editMode(): EditMode {
    if (this.mode === "edit" && this.canEdit()) return "edit";
    if (this.mode === "suggest" && this.canSuggest()) return "suggest";
    return "view";
  }

  setMode(mode: EditMode): void {
    if (this.mode === mode) return;
    const v = this.ed;
    if (v && this.mode === "suggest") void suggestUI(v)?.commit();
    this.mode = mode;
    this.ed?.setProps({});
    for (const fn of this.modeListeners) fn(mode);
  }

  restoreDoc(doc: PMNode): { kept: number } | null {
    const v = this.ed;
    const s = this.session;
    if (!v || !s || !this.canEdit() || s.phase !== "ready") return null;
    const busy = this.remoteBlocks(v, s);
    const live = new Map<string, PMNode>();
    v.state.doc.forEach((n) => {
      const id = String(n.attrs.id ?? "");
      if (id) live.set(id, n);
    });
    const out: PMNode[] = [];
    const placed = new Set<string>();
    doc.forEach((n) => {
      const id = String(n.attrs.id ?? "");
      if (id) placed.add(id);
      out.push((id && busy.has(id) && live.get(id)) || n);
    });
    let prev = "";
    v.state.doc.forEach((n) => {
      const id = String(n.attrs.id ?? "");
      if (id && busy.has(id) && !placed.has(id)) {
        const at = prev ? out.findIndex((x) => String(x.attrs.id ?? "") === prev) + 1 : 0;
        out.splice(at, 0, n);
        placed.add(id);
      }
      if (id && placed.has(id)) prev = id;
    });
    const tr = v.state.tr.replaceWith(0, v.state.doc.content.size, Fragment.from(out));
    v.dispatch(tr.setMeta(suggestKey, { apply: true }));
    return { kept: [...busy].filter((id) => live.has(id)).length };
  }

  private remoteBlocks(v: EditorView, s: DocSession): Set<string> {
    const out = new Set<string>();
    const binding = ySyncPluginKey.getState(v.state)?.binding;
    if (!binding || s.phase !== "ready") return out;
    s.awareness.getStates().forEach((st, client) => {
      const cur = (st as { cursor?: { anchor?: unknown; head?: unknown } }).cursor;
      if (client === s.awareness.clientID || !cur) return;
      for (const rel of [cur.anchor, cur.head]) {
        if (!rel) continue;
        const pos = relativePositionToAbsolutePosition(s.doc, s.fragment, Y.createRelativePositionFromJSON(rel), binding.mapping);
        if (pos === null || pos < 0 || pos > v.state.doc.content.size) continue;
        const block = v.state.doc.maybeChild(v.state.doc.resolve(pos).index(0));
        const id = block ? String(block.attrs.id ?? "") : "";
        if (id) out.add(id);
      }
    });
    return out;
  }

  onMode(fn: (m: EditMode) => void): () => void {
    this.modeListeners.add(fn);
    return () => this.modeListeners.delete(fn);
  }

  pickFiles(view: EditorView, accept: string, at?: number): void {
    pickFiles(this, view, accept, at);
  }

  uploadFiles(view: EditorView, files: File[], at?: number): void {
    uploadFiles(this, view, files, at);
  }

  doc(): Y.Doc | null {
    return this.session?.phase === "ready" ? this.session.doc : null;
  }

  syncedSnapshot(block: string): string {
    return this.snaps.get(block) ?? "";
  }

  onShown(fn: (id: string) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  ssrPage(): string {
    return this.ed ? "" : (this.mount.dataset.page ?? "");
  }

  current(): string {
    return this.ed && this.session?.phase === "ready" ? this.session.id : "";
  }

  isReady(id: string): boolean {
    const s = this.engine.docs.peek(id);
    return !!s && s.phase !== "loading";
  }

  prepare(id: string): Promise<void> {
    const s = this.engine.docs.prefetch(id);
    return s.ready.then(() => {
      if (s.phase !== "ready") return;
      const sources = new Set<string>();
      const walk = (el: Y.XmlFragment | Y.XmlElement) => {
        for (const c of el.toArray()) {
          if (!(c instanceof Y.XmlElement)) continue;
          if (c.nodeName === "synced") {
            const page = String(c.getAttribute("page") ?? "");
            if (page && page !== id && sources.size < 6) sources.add(page);
          } else walk(c);
        }
      };
      walk(s.fragment);
      if (!sources.size) return;
      const ready = Promise.all([...sources].map((p) => this.engine.docs.prefetch(p).ready));
      return Promise.race([ready.then(() => undefined), new Promise<void>((r) => setTimeout(r, 1500))]);
    });
  }

  show(id: string): void {
    this.wanted = id;
    if (this.session?.id === id) return;
    const next = this.engine.docs.open(id);
    const prev = this.session;
    if (prev) {
      prev.awareness.setLocalStateField("cursor", null);
      this.engine.docs.release(prev);
    }
    this.session = next;
    this.unsub?.();
    this.unsub = next.on((what) => {
      if (this.session !== next) return;
      if (what === "ready" || what === "reset") this.bind(next);
      else if (what === "write") this.ed?.setProps({});
      else if (what === "denied") this.deny(next);
    });
    if (next.phase === "ready") this.bind(next);
    else if (next.phase === "denied") this.deny(next);
    else this.loading();
  }

  private loading(): void {
    this.handle.hide();
    this.bar.hide();
    this.clearNotice();
    if (this.mount.dataset.page === this.wanted && !this.ed) return;
    this.mount.classList.add("is-loading");
    if (this.ed) {
      this.ed.updateState(EditorState.create({ schema, doc: schema.topNodeType.createAndFill() as PMNode }));
      this.ed.setProps({ editable: () => false });
    } else this.mount.replaceChildren();
  }

  private deny(s: DocSession): void {
    this.loading();
    this.mount.classList.remove("is-loading");
    const msg = s.denied === "not_found" ? t("This page's content is not available to you.") : t("The content could not be opened. Reload to try again.");
    this.notice = h("p", { class: "doc-notice", text: msg });
    this.mount.after(this.notice);
  }

  private clearNotice(): void {
    this.notice?.remove();
    this.notice = null;
  }

  private plugins(s: DocSession, mapping: unknown): Plugin[] {
    const host = this;
    return [
      ySyncPlugin(s.fragment, mapping ? ({ mapping } as Parameters<typeof ySyncPlugin>[1]) : {}),
      yCursorPlugin(s.awareness, { cursorBuilder, selectionBuilder }),
      suggestPlugin(this),
      yUndoPlugin(),
      triggerPlugin(this),
      rulesPlugin(),
      mathKeys(),
      ...keysPlugin(this),
      idsPlugin(),
      placeholderPlugin(),
      highlightPlugin(),
      uploadsPlugin(),
      livePlugin(this),
      linksPlugin(this),
      commentsPlugin(this),
      ...columnsPlugins(() => this.wrap),
      ...structurePlugins(),
      new Plugin({
        key: new PluginKey("formmd-chrome"),
        view: () => ({
          update(view) {
            host.bar.update(view);
          },
        }),
      }),
    ];
  }

  private bind(s: DocSession): void {
    if (this.session !== s || s.phase !== "ready") return;
    this.clearNotice();
    const frag = s.fragment;
    let doc: PMNode;
    let mapping: unknown = null;
    if (frag.length > 0) {
      const init = initProseMirrorDoc(frag, schema);
      doc = init.doc;
      mapping = init.mapping;
      try {
        doc.check();
      } catch {
        doc = schema.topNodeType.createAndFill() as PMNode;
        mapping = null;
      }
    } else doc = schema.topNodeType.createAndFill() as PMNode;
    const state = EditorState.create({ schema, doc, plugins: this.plugins(s, mapping) });
    this.snaps.clear();
    if (!this.ed && this.mount.dataset.page === s.id) {
      for (const el of Array.from(this.mount.querySelectorAll<HTMLElement>(".synced[data-block]"))) this.snaps.set(el.dataset.block ?? "", el.querySelector(".synced__body")?.innerHTML ?? "");
    }
    if (!this.ed) {
      this.ed = new EditorView(
        { mount: this.mount },
        {
          state,
          nodeViews: nodeViews(this),
          editable: () => this.writable() || this.suggesting(),
          attributes: { class: "doc-body", "aria-label": t("Page content"), spellcheck: "true" },
          clipboardTextSerializer: (slice: Slice) => this.md.doc(slice.content),
          clipboardTextParser: ((text: string, $context: ResolvedPos, plain: boolean) => (!plain && looksLikeMarkdown(text) ? parseMarkdownSlice(text, $context) : null)) as never,
          transformPastedHTML: (html) => html.replace(/<img\b[^>]*\b(?:width|height)="[0-2](?:px)?"[^>]*>/gi, ""),
          handlePaste: (view, e) => this.paste(view, e),
          handleDrop: (view, e, _slice, moved) => {
            const files = Array.from((e as DragEvent).dataTransfer?.files ?? []);
            if (moved || !files.length || !this.writable()) return false;
            const at = view.posAtCoords({ left: (e as DragEvent).clientX, top: (e as DragEvent).clientY });
            e.preventDefault();
            this.uploadFiles(view, files, at?.pos);
            return true;
          },
          handleClick: (view, pos, e) => this.click(view, pos, e),
        },
      );
      this.bar.attach(this.ed);
      this.ed.dom.addEventListener("keydown", () => this.handle.hide());
      this.ed.dom.addEventListener("focusin", preloadMarkdown);
    } else {
      this.ed.updateState(state);
      this.ed.setProps({ editable: () => this.writable() || this.suggesting() });
    }
    this.mount.classList.remove("is-loading");
    delete this.mount.dataset.page;
    for (const v of this.live) v.refresh();
    for (const fn of this.listeners) fn(s.id);
  }

  private paste(view: EditorView, e: ClipboardEvent): boolean {
    const data = e.clipboardData;
    if (!data || this.suggesting()) return false;
    const files = Array.from(data.files ?? []);
    if (files.length && !data.types.includes("text/html") && this.writable()) {
      this.uploadFiles(view, files);
      return true;
    }
    const text = data.getData("text/plain").trim();
    if (!URL_ONLY.test(text) || data.types.includes("text/html")) return false;
    const { from, to, empty, $from } = view.state.selection;
    const link = schema.marks.link;
    if (!empty && link) {
      view.dispatch(view.state.tr.addMark(from, to, link.create({ href: text })));
      return true;
    }
    const provider = embedProvider(text);
    if (provider && $from.parent.type.name === "paragraph" && $from.parent.content.size === 0 && $from.depth >= 1) {
      const node = schema.nodes.embed?.create({ id: newBlockId(), url: text, provider });
      if (!node) return false;
      const pos = $from.before();
      view.dispatch(view.state.tr.replaceWith(pos, pos + $from.parent.nodeSize, node).scrollIntoView());
      return true;
    }
    if (link) {
      const alone = $from.parent.type.name === "paragraph" && $from.parent.content.size === 0 && $from.depth >= 1 ? $from.before() : -1;
      view.dispatch(view.state.tr.replaceSelectionWith(schema.text(text, [link.create({ href: text })]), false).scrollIntoView());
      if (alone >= 0 && this.writable()) offerLinkChoice(this, view, alone, text, this.wrap);
      return true;
    }
    return false;
  }

  private click(view: EditorView, pos: number, e: MouseEvent): boolean {
    const target = e.target as Element;
    const mention = target.closest?.<HTMLElement>(".mention[data-kind=page]");
    const newTab = this.engine.platform.mac ? e.metaKey : e.ctrlKey;
    if (mention?.dataset.ref) {
      e.preventDefault();
      this.engine.openPage(mention.dataset.ref, { newTab });
      return true;
    }
    const link = schema.marks.link;
    const $pos: ResolvedPos = view.state.doc.resolve(pos);
    const mark = link?.isInSet($pos.marks()) ?? (target.closest?.("a[href]") && link ? link.isInSet(view.state.doc.resolve(Math.min(pos + 1, view.state.doc.content.size)).marks()) : undefined);
    if (!mark) return false;
    const href = String(mark.attrs.href ?? "");
    if (!href || (view.editable && !newTab)) return false;
    e.preventDefault();
    const page = /^\/p\/([a-z2-7]{26})/.exec(href);
    if (page?.[1]) this.engine.openPage(page[1], { newTab });
    else window.open(href, "_blank", "noopener,noreferrer");
    return true;
  }
}
