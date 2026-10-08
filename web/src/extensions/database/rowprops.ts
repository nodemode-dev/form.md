import type { Engine } from "../../engine";
import type { QuerySession } from "../../engine/collections";
import { plural, t } from "../../engine/i18n";
import type { Collection, Property, RowDetail } from "../../engine/types";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { reconcile } from "../../ui/list";
import { type DbCtx, paintCell } from "./cells";
import { editCell } from "./editors";
import { openAddProperty, propertyMenu } from "./propedit";
import { computed, isEmptyValue, TYPE_INFO } from "./props";

export interface RowPropsHandle {
  show(rowId: string, detail?: RowDetail | null): Promise<void>;
  hide(): void;
  ready(rowId: string): boolean;
}

export function mountRowProps(engine: Engine, host: HTMLElement): RowPropsHandle {
  const el = h("section", { class: "rowprops", attrs: { "aria-label": t("Properties") } });
  const list = h("div", { class: "rowprops__list" });
  const toggle = h("button", { class: "link-btn rowprops__toggle", attrs: { type: "button" } });
  const add = h("button", { class: "link-btn rowprops__add", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add a property") }));
  const source = h("a", { class: "rowprops__source" });
  el.append(source, list, toggle, add);
  host.append(el);
  el.hidden = true;

  let rowId = "";
  let colId = "";
  let session: QuerySession | null = null;
  let showAll = false;
  let offCol: (() => void) | null = null;
  let seq = 0;

  const ctx = (): DbCtx | null => {
    const col = engine.collections.get(colId);
    if (!session || !col) return null;
    return { engine, colId, col: () => engine.collections.get(colId) as Collection, session, canWrite: () => !!engine.collections.get(colId)?.can_write, canDesign: () => !!engine.collections.get(colId)?.can_design, workspaceId: () => col.workspace_id };
  };

  const paint = () => {
    const c = ctx();
    const row = session?.rows.get(rowId);
    if (!c || !row) return;
    const col = c.col();
    setText(source, col.title || t("Untitled database"));
    source.setAttribute("href", `/p/${col.id}`);
    const props = col.schema.filter((p) => p.type !== "title");
    const empty = props.filter((p) => isEmptyValue(row.values[p.id]) && p.type !== "checkbox" && !computed(p.type));
    const shown = showAll ? props : props.filter((p) => !empty.includes(p));
    reconcile(
      list,
      shown,
      (p) => p.id,
      (p) => h("div", { class: "rowprops__row", attrs: { "data-prop": p.id } }, h("button", { class: "rowprops__name", attrs: { type: "button" } }, h("span", { class: "rowprops__icon" }), h("span", { class: "rowprops__label" })), h("div", { class: "rowprops__value", attrs: { tabindex: "0", role: "button" } })),
      (node, p: Property) => {
        const icon = node.querySelector(".rowprops__icon") as HTMLElement;
        if (icon.dataset.icon !== TYPE_INFO[p.type].icon) {
          icon.dataset.icon = TYPE_INFO[p.type].icon;
          icon.replaceChildren(iconNode(TYPE_INFO[p.type].icon));
        }
        setText(node.querySelector(".rowprops__label") as HTMLElement, p.name);
        const value = node.querySelector(".rowprops__value") as HTMLElement;
        paintCell(value, p, row, c, "panel");
        setAttr(value, "data-placeholder", c.canWrite() && !computed(p.type) ? t("Empty") : null);
        setAttr(value, "data-readonly", c.canWrite() && !computed(p.type) ? null : "true");
      },
    );
    toggle.hidden = empty.length === 0;
    setText(toggle, showAll ? t("Hide empty properties") : plural(empty.length, "{count} more property", "{count} more properties"));
    add.hidden = !c.canDesign();
    el.hidden = false;
  };

  list.addEventListener("click", (e) => {
    const c = ctx();
    const row = session?.rows.get(rowId);
    const node = (e.target as Element).closest<HTMLElement>(".rowprops__row");
    const p = node ? c?.col().schema.find((x) => x.id === node.dataset.prop) : undefined;
    if (!c || !row || !node || !p) return;
    const ref = (e.target as Element).closest<HTMLElement>("a.ref");
    if (ref?.dataset.id) {
      e.preventDefault();
      engine.openPage(ref.dataset.id);
      return;
    }
    if ((e.target as Element).closest("a")) return;
    if ((e.target as Element).closest(".rowprops__name")) {
      if (c.canDesign()) propertyMenu(node.querySelector(".rowprops__name") as HTMLElement, p, c, []);
      return;
    }
    editCell(node.querySelector(".rowprops__value") as HTMLElement, p, row, c);
  });
  list.addEventListener("keydown", (e) => {
    if (e.key !== "Enter") return;
    const v = (e.target as Element).closest<HTMLElement>(".rowprops__value");
    v?.click();
  });
  toggle.addEventListener("click", () => {
    showAll = !showAll;
    paint();
  });
  add.addEventListener("click", () => {
    const c = ctx();
    if (!c) return;
    showAll = true;
    openAddProperty(add, c);
  });
  source.addEventListener("click", (e) => {
    e.preventDefault();
    if (colId) engine.openPage(colId);
  });

  const release = () => {
    session?.close();
    session = null;
    offCol?.();
    offCol = null;
  };

  return {
    ready(id) {
      return id === rowId && !!session?.rows.get(id);
    },
    async show(id, detail) {
      if (id === rowId && session) return;
      const mine = ++seq;
      let d = detail ?? null;
      if (!d) {
        try {
          d = await engine.api.collections.row(id);
        } catch (e) {
          engine.reportError(e);
          return;
        }
      }
      if (mine !== seq || !d.row) return;
      release();
      rowId = id;
      colId = d.collection.id;
      if (!engine.collections.get(colId)) engine.collections.seed(d.collection);
      session = engine.collections.session(colId, "", { ids: [id] });
      session.seed({ total: 1, rows: [d.row], order: [id], refs: d.refs, epoch: d.collection.epoch, offset: 0 });
      session.on(() => paint());
      offCol = engine.collections.subscribe(colId, () => paint());
      showAll = false;
      paint();
      void engine.collections.people(d.collection.workspace_id).then(paint, () => undefined);
    },
    hide() {
      seq++;
      rowId = "";
      release();
      el.hidden = true;
    },
  };
}
