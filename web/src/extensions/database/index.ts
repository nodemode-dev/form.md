import type { Node as PMNode, NodeType } from "prosemirror-model";
import type { EditorView, NodeView } from "prosemirror-view";
import type { Engine } from "../../engine";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import type { ViewType } from "../../engine/types";
import { h } from "../../ui/dom";
import { openMenu } from "../../ui/menu";
import { pageIcon } from "../../ui/pageicon";
import { CATALOG } from "../blocks/catalog";
import { insertBlock } from "../blocks/commands";
import type { EditorCtx } from "../blocks/context";
import { registerNodeView } from "../blocks/nodeviews";
import type { Extension } from "../registry";
import { databaseViews, loadDatabase } from "./load";
import { VIEW_INFO } from "./props";
import type { DatabaseHandle } from "./surface";

class DatabaseBlockView implements NodeView {
  dom: HTMLElement;
  private handle: DatabaseHandle | null = null;
  private gone = false;

  constructor(
    private node: PMNode,
    private readonly ctx: EditorCtx,
  ) {
    this.dom = h("div", { class: "dbblock", attrs: { contenteditable: "false" } });
    this.dom.dataset.block = String(node.attrs.id ?? "");
    this.mount();
  }

  private mount(): void {
    const col = String(this.node.attrs.collection ?? "");
    this.dom.dataset.collection = col;
    if (!col) {
      this.dom.replaceChildren(h("p", { class: "faint", text: t("This database is missing.") }));
      return;
    }
    const view = String(this.node.attrs.view ?? "");
    const attach = (m: NonNullable<ReturnType<typeof databaseViews>>) => {
      if (this.gone) return;
      this.handle = m.mountDatabase(this.ctx.engine, this.dom, col, { inline: true, ...(view ? { viewId: view } : {}) });
      void this.handle.ready.then(() => {
        this.dom.setAttribute("data-ready", "true");
        if (focusNext.delete(col)) this.handle?.focusNew();
      });
    };
    const ready = databaseViews();
    if (ready) attach(ready);
    else void loadDatabase().then(attach, this.ctx.engine.reportError);
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    if (node.attrs.collection !== this.node.attrs.collection) return false;
    this.node = node;
    this.dom.dataset.block = String(node.attrs.id ?? "");
    return true;
  }

  selectNode(): void {
    this.dom.classList.add("is-selected");
  }

  deselectNode(): void {
    this.dom.classList.remove("is-selected");
  }

  stopEvent(): boolean {
    return true;
  }

  ignoreMutation(): boolean {
    return true;
  }

  destroy(): void {
    this.gone = true;
    this.handle?.destroy();
    this.handle = null;
  }
}

const N = schema.nodes as Record<string, NodeType>;
const DATA = () => t("Databases");
const focusNext = new Set<string>();

function createInline(view: EditorView, ctx: EditorCtx, type: ViewType): void {
  const ws = ctx.workspaceId();
  const parent = ctx.pageId();
  if (!ws || !parent) return;
  void ctx.engine.api.pages
    .createCollection({ workspace_id: ws, parent_id: parent, inline: true, view: type, ...(type === "table" ? { title: t("New table") } : {}) })
    .then((r) => {
      ctx.engine.pages.upsert(r.page);
      focusNext.add(r.page.id);
      insertBlock(view, (N.database as NodeType).create({ id: newBlockId(), collection: r.page.id, view: "" }));
    })
    .catch(ctx.engine.reportError);
}

function linkExisting(view: EditorView, ctx: EditorCtx): void {
  const ws = ctx.workspaceId();
  const all = ws && ctx.engine.pages.isLoaded(ws) ? ctx.engine.pages.all(ws).filter((p) => p.kind === "collection") : [];
  const coords = view.coordsAtPos(view.state.selection.head);
  const pin = h("span", { class: "caret-anchor" });
  pin.style.left = `${coords.left}px`;
  pin.style.top = `${coords.bottom}px`;
  document.body.append(pin);
  openMenu(
    pin,
    all.length
      ? all.map((p) => ({
          label: p.title || t("Untitled database"),
          lead: pageIcon(p.icon, "database"),
          run: () => insertBlock(view, (N.database as NodeType).create({ id: newBlockId(), collection: p.id, view: "" })),
        }))
      : [{ label: t("No databases in this workspace yet"), run: () => undefined }],
    { side: "bottom", align: "start", label: t("Linked view of a database") },
  );
  setTimeout(() => pin.remove(), 0);
}

function setup(engine: Engine): void {
  void engine;
  registerNodeView("database", (ctx) => (node) => new DatabaseBlockView(node, ctx));
  const types: ViewType[] = ["table", "board", "list", "gallery", "calendar", "timeline"];
  for (const type of types) {
    CATALOG.push({
      id: `db-${type}`,
      label: () => (type === "table" ? t("Database table") : t("{view} database", { view: t(VIEW_INFO[type].label) })),
      section: DATA,
      icon: VIEW_INFO[type].icon,
      get hint() {
        return type === "table" ? t("Sort, filter and add columns") : undefined;
      },
      keywords: `database inline ${type} view collection`,
      run: (view, ctx) => createInline(view, ctx, type),
    });
  }
  CATALOG.push(
    {
      id: "db-page",
      label: () => t("Database page"),
      section: DATA,
      icon: "database",
      keywords: "database full page collection table new",
      run: (view, ctx) => {
        const ws = ctx.workspaceId();
        const parent = ctx.pageId();
        if (!ws || !parent) return;
        void ctx.engine.api.pages
          .createCollection({ workspace_id: ws, parent_id: parent })
          .then((r) => {
            ctx.engine.pages.upsert(r.page);
            const mention = (N.mention as NodeType).create({ kind: "page", ref: r.page.id, label: "" });
            insertBlock(view, (N.paragraph as NodeType).create({ id: newBlockId() }, [mention]));
          })
          .catch(ctx.engine.reportError);
      },
    },
    { id: "db-linked", label: () => t("Linked view of a database"), section: DATA, icon: "relation", keywords: "linked database view existing", run: (view, ctx) => linkExisting(view, ctx) },
  );
}

const extension: Extension = { id: "database", setup };
export default extension;
