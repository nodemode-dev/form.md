import { baseKeymap } from "prosemirror-commands";
import { keymap } from "prosemirror-keymap";
import { Fragment, type Node as PMNode, type NodeType } from "prosemirror-model";
import { EditorState } from "prosemirror-state";
import { EditorView, type NodeView } from "prosemirror-view";
import { initProseMirrorDoc, prosemirrorToYXmlFragment, redo, undo, ySyncPlugin, yUndoPlugin } from "y-prosemirror";
import type * as Y from "yjs";
import type { DocSession } from "../../engine/doc/docs";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { toast } from "../../ui/toast";
import { markCommand } from "./commands";
import type { EditorCtx } from "./context";
import { nodeViews } from "./nodeviews";
import { highlightPlugin, idsPlugin, keysPlugin, placeholderPlugin, structurePlugins } from "./plugins";
import { rulesPlugin } from "./rules";

type GetPos = () => number | undefined;

const N = schema.nodes as Record<string, NodeType>;
const META = "synced";

export function syncRoot(key: string): string {
  return `s:${key}`;
}

function released(doc: Y.Doc, key: string): boolean {
  const m = doc.getMap(META).get(key) as { released?: boolean } | undefined;
  return !!m?.released;
}

export function contentOf(doc: Y.Doc, key: string): Fragment {
  const frag = doc.getXmlFragment(syncRoot(key));
  if (!frag.length) return Fragment.empty;
  try {
    return initProseMirrorDoc(frag, schema).doc.content;
  } catch {
    return Fragment.empty;
  }
}

export function createSynced(ctx: EditorCtx, content: Fragment): PMNode | null {
  const doc = ctx.doc();
  const page = ctx.pageId();
  if (!doc || !page) return null;
  const key = newBlockId();
  const body = content.size ? content : Fragment.from((N.paragraph as NodeType).create({ id: newBlockId() }));
  doc.transact(() => {
    prosemirrorToYXmlFragment(schema.topNodeType.create(null, body), doc.getXmlFragment(syncRoot(key)));
  });
  return (N.synced as NodeType).create({ id: newBlockId(), page, key });
}

export class SyncedView implements NodeView {
  dom: HTMLElement;
  private body: HTMLElement;
  private bar: HTMLElement;
  private label: HTMLElement;
  private session: DocSession | null = null;
  private inner: EditorView | null = null;
  private offSession: (() => void) | null = null;
  private offMeta: (() => void) | null = null;
  private boundDoc: Y.Doc | null = null;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: GetPos,
    private readonly ctx: EditorCtx,
  ) {
    this.label = h("span", { class: "synced__label" });
    const more = h("button", { class: "icon-btn synced__more", attrs: { type: "button", "aria-label": t("Synced block options"), "data-tip": t("Synced block options") } }, iconNode("more"));
    more.addEventListener("click", () => this.menu(more));
    this.bar = h("div", { class: "synced__bar", attrs: { contenteditable: "false" } }, iconNode("refresh"), this.label, more);
    this.body = h("div", { class: "synced__body" });
    const snap = ctx.syncedSnapshot(String(node.attrs.id ?? ""));
    if (snap) this.body.innerHTML = snap;
    this.dom = h("div", { class: "synced", attrs: { contenteditable: "false" } }, this.bar, this.body);
    this.paint();
    this.connect();
  }

  private original(): boolean {
    return String(this.node.attrs.page) === this.ctx.pageId();
  }

  private paint(): void {
    setAttr(this.dom, "data-block", this.node.attrs.id ? String(this.node.attrs.id) : null);
    setAttr(this.dom, "data-page", String(this.node.attrs.page ?? ""));
    setAttr(this.dom, "data-key", String(this.node.attrs.key ?? ""));
    this.dom.classList.toggle("is-original", this.original());
    const src = this.ctx.engine.pages.get(String(this.node.attrs.page));
    setText(this.label, this.original() ? t("Synced block, original") : t("Synced from {title}", { title: src?.title || t("Untitled") }));
  }

  private connect(): void {
    const page = String(this.node.attrs.page ?? "");
    const key = String(this.node.attrs.key ?? "");
    if (!page || !key) return;
    const s = this.ctx.engine.docs.open(page);
    this.session = s;
    this.offSession = s.on((what) => {
      if (what === "ready" || what === "reset") this.bind();
      else if (what === "write") this.inner?.setProps({});
      else if (what === "denied") this.deny();
    });
    if (s.phase === "ready") this.bind();
    else if (s.phase === "denied") this.deny();
  }

  private deny(): void {
    this.inner?.destroy();
    this.inner = null;
    this.body.replaceChildren(h("p", { class: "synced__notice", text: t("This synced block comes from a page you cannot open.") }));
  }

  private writable(): boolean {
    const s = this.session;
    if (!s || s.phase !== "ready" || !s.write || !this.view.editable) return false;
    return this.original() ? this.ctx.writable() : true;
  }

  private bind(): void {
    const s = this.session;
    if (!s || s.phase !== "ready") return;
    const key = String(this.node.attrs.key);
    const frag = s.doc.getXmlFragment(syncRoot(key));
    if (this.boundDoc !== s.doc) {
      this.offMeta?.();
      const meta = s.doc.getMap(META);
      const watch = () => this.checkReleased();
      meta.observe(watch);
      this.offMeta = () => meta.unobserve(watch);
      this.boundDoc = s.doc;
    }
    let doc: PMNode;
    let mapping: unknown = null;
    if (frag.length > 0) {
      const init = initProseMirrorDoc(frag, schema);
      doc = init.doc;
      mapping = init.mapping;
    } else doc = schema.topNodeType.createAndFill() as PMNode;
    const sub = Object.create(this.ctx, {
      pageId: { value: () => String(this.node.attrs.page) },
      view: { value: () => this.inner },
      writable: { value: () => this.writable() },
    }) as EditorCtx;
    const state = EditorState.create({
      schema,
      doc,
      plugins: [
        ySyncPlugin(frag, mapping ? ({ mapping } as Parameters<typeof ySyncPlugin>[1]) : {}),
        yUndoPlugin(),
        keymap({
          "Mod-z": undo,
          "Mod-y": redo,
          "Mod-Shift-z": redo,
          "Mod-b": markCommand("strong"),
          "Mod-i": markCommand("em"),
          "Mod-u": markCommand("underline"),
          "Mod-e": markCommand("code"),
          "Mod-Shift-x": markCommand("strike"),
        }),
        rulesPlugin(),
        ...keysPlugin(sub),
        keymap(baseKeymap),
        idsPlugin(),
        placeholderPlugin(),
        highlightPlugin(),
        ...structurePlugins(),
      ],
    });
    if (this.inner) {
      this.inner.updateState(state);
      this.inner.setProps({ editable: () => this.writable() });
    } else {
      this.body.replaceChildren();
      this.inner = new EditorView(
        { mount: this.body },
        { state, nodeViews: nodeViews(sub), editable: () => this.writable(), attributes: { class: "synced__body", "aria-label": t("Synced block") } },
      );
    }
    this.checkReleased();
  }

  private checkReleased(): void {
    const s = this.session;
    if (!s || s.phase !== "ready" || this.original()) return;
    if (released(s.doc, String(this.node.attrs.key)) && this.view.editable) queueMicrotask(() => this.unsync(false));
  }

  private menu(anchor: HTMLElement): void {
    const entries: MenuEntry[] = [{ label: t("Copy and sync"), icon: "copy", run: () => this.copy() }];
    if (!this.original()) {
      entries.push({ label: t("Go to the original"), icon: "external", run: () => this.ctx.engine.openPage(String(this.node.attrs.page)) });
      if (this.view.editable) entries.push({ kind: "separator" }, { label: t("Unsync"), icon: "close", run: () => this.unsync(true) });
    } else if (this.view.editable) entries.push({ kind: "separator" }, { label: t("Unsync all"), icon: "close", run: () => this.unsyncAll() });
    openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Synced block") });
  }

  private copy(): void {
    const a = this.node.attrs;
    const html = `<div class="synced" data-page="${a.page}" data-key="${a.key}"></div>`;
    const text = this.inner?.state.doc.textContent ?? "";
    const item = new ClipboardItem({ "text/html": new Blob([html], { type: "text/html" }), "text/plain": new Blob([text], { type: "text/plain" }) });
    void navigator.clipboard
      ?.write([item])
      .then(() => toast({ id: "synced-copy", tone: "success", message: t("Copied. Paste it on any page to sync it there.") }))
      .catch(() => undefined);
  }

  private unsync(loud: boolean): void {
    const pos = this.getPos();
    const s = this.session;
    if (pos === undefined || !s || s.phase !== "ready" || !this.view.editable) return;
    const content = this.inner?.state.doc.content ?? contentOf(s.doc, String(this.node.attrs.key));
    const cur = this.view.state.doc.nodeAt(pos);
    if (!cur || cur.type !== N.synced) return;
    const blocks = content.size ? content : Fragment.from((N.paragraph as NodeType).create({ id: newBlockId() }));
    this.view.dispatch(this.view.state.tr.replaceWith(pos, pos + cur.nodeSize, blocks));
    if (loud) toast({ id: "synced-unsync", tone: "success", message: t("Unsynced. This copy is now its own content.") });
  }

  private unsyncAll(): void {
    const s = this.session;
    if (!s || s.phase !== "ready") return;
    const key = String(this.node.attrs.key);
    s.doc.getMap(META).set(key, { released: true, at: Date.now() });
    this.unsync(false);
    toast({ id: "synced-unsync", tone: "success", message: t("Unsynced everywhere. Each copy keeps its content.") });
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type || node.attrs.key !== this.node.attrs.key || node.attrs.page !== this.node.attrs.page) return false;
    this.node = node;
    this.paint();
    return true;
  }

  stopEvent(e: Event): boolean {
    return this.body.contains(e.target as Node) || this.bar.contains(e.target as Node);
  }

  ignoreMutation(): boolean {
    return true;
  }

  destroy(): void {
    this.offSession?.();
    this.offMeta?.();
    this.inner?.destroy();
    this.inner = null;
    if (this.session) this.ctx.engine.docs.release(this.session);
    this.session = null;
  }
}
