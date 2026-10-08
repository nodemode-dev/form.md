import type { Node as PMNode, ResolvedPos, Slice } from "prosemirror-model";
import { EditorState, Plugin, PluginKey } from "prosemirror-state";
import { EditorView } from "prosemirror-view";
import { initProseMirrorDoc, yCursorPlugin, ySyncPlugin, yUndoPlugin } from "y-prosemirror";
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
import { rulesPlugin } from "./rules";
import { triggerPlugin } from "./triggers";

const URL_ONLY = /^https?:\/\/\S+$/i;

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
    const s = this.session;
    if (!s || s.phase !== "ready" || !s.write) return false;
    const d = this.engine.state.get("detail");
    if (d && d.id === s.id && !d.can_write) return false;
    return this.engine.canWrite(this.workspaceId());
  }

  pickFiles(view: EditorView, accept: string, at?: number): void {
    pickFiles(this, view, accept, at);
  }

  uploadFiles(view: EditorView, files: File[], at?: number): void {
    uploadFiles(this, view, files, at);
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
    return this.engine.docs.prefetch(id).ready;
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
      yUndoPlugin(),
      triggerPlugin(this),
      rulesPlugin(),
      ...keysPlugin(this),
      idsPlugin(),
      placeholderPlugin(),
      highlightPlugin(),
      uploadsPlugin(),
      livePlugin(this),
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
    if (!this.ed) {
      this.ed = new EditorView(
        { mount: this.mount },
        {
          state,
          nodeViews: nodeViews(this),
          editable: () => this.writable(),
          attributes: { class: "doc-body", "aria-label": t("Page content"), spellcheck: "true" },
          clipboardTextSerializer: (slice: Slice) => this.md.doc(slice.content),
          clipboardTextParser: ((text: string, $context: ResolvedPos, plain: boolean) => (!plain && looksLikeMarkdown(text) ? parseMarkdownSlice(text, $context) : null)) as never,
          transformPastedHTML: (html) => html.replace(/<img\b[^>]*\bsrc="(?:https?:)?\/\/[^"]*"[^>]*>/gi, ""),
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
      this.ed.setProps({ editable: () => this.writable() });
    }
    this.mount.classList.remove("is-loading");
    delete this.mount.dataset.page;
    for (const v of this.live) v.refresh();
    for (const fn of this.listeners) fn(s.id);
  }

  private paste(view: EditorView, e: ClipboardEvent): boolean {
    const data = e.clipboardData;
    if (!data) return false;
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
      view.dispatch(view.state.tr.replaceSelectionWith(schema.text(text, [link.create({ href: text })]), false).scrollIntoView());
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
