import { newBlockId } from "../../web/src/engine/doc/schema";
import type { GroupResult, Property, RowData } from "../../web/src/engine/types";
import { optionChip, paintCell, type DbCtx } from "../../web/src/extensions/database/cells";
import { TYPE_INFO } from "../../web/src/extensions/database/props";
import { simpleDialog } from "../../web/src/ui/dialog";
import { h, setAttr, setText } from "../../web/src/ui/dom";
import { actions, button, select } from "../../web/src/ui/form";
import { type IconName, iconNode } from "../../web/src/ui/icons";
import { reconcile } from "../../web/src/ui/list";
import { type MenuEntry, openMenu } from "../../web/src/ui/menu";
import { toast } from "../../web/src/ui/toast";
import { cellText, toDelimited, toMarkdownTable } from "./csv";
import { COND_LABELS, calcOptions, calcValue, condsFor, filterRows, groupRows, needsValue, sortRows, type Rule } from "./query";
import { type DataItem, touch } from "./store";

const ctx = { session: { refs: new Map() }, engine: { collections: { person: () => null } }, workspaceId: () => "demo" } as unknown as DbCtx;
const PAGE_ROWS = 250;

export interface DbHooks {
  insertMarkdown(md: string): void;
  download(name: string, text: string, mime: string): void;
  changed(): void;
}

function widthOf(p: Property): number {
  if (p.type === "title") return 260;
  if (p.type === "checkbox") return 110;
  if (p.type === "number") return 140;
  if (p.type === "date") return 160;
  if (p.type === "text") return 240;
  return 170;
}

function sortedBy(item: DataItem, prop: string): "asc" | "desc" | "" {
  const s = item.view.sorts.find((x) => x.prop === prop);
  return s ? s.dir : "";
}

export class DbView {
  private item: DataItem | null = null;
  private readonly root = h("div", { class: "db", attrs: { "data-inline": "false" } });
  private readonly tabs = h("div", { class: "db__tabs", attrs: { role: "tablist", "aria-label": "Views" } });
  private readonly stage = h("div", { class: "db__stage" });
  private readonly summary = h("p", { class: "demo-dbsummary" });
  private readonly search = h("input", { class: "db__search", attrs: { type: "search", placeholder: "Type to search", "aria-label": "Search this view" } });
  private readonly filterBtn = this.tool("filter", "Filter", "Filter");
  private readonly sortBtn = this.tool("sort", "Sort", "Sort");
  private readonly groupBtn = this.tool("group", "Group by", "Group");
  private readonly exportBtn = this.tool("download", "Export", "Export");
  private readonly filterBadge = h("span", { class: "db__badge" });
  private readonly tbl = h("div", { class: "tbl", attrs: { role: "grid" } });
  private readonly headWrap = h("div", { class: "tbl__headwrap" });
  private readonly head = h("div", { class: "tbl__row tbl__row--head", attrs: { role: "row" } });
  private readonly scroller = h("div", { class: "tbl__scroll" });
  private readonly body = h("div", { class: "tbl__body tbl__body--flow", attrs: { role: "rowgroup" } });
  private readonly more = h("button", { class: "tbl__more", attrs: { type: "button" } });
  private readonly newRow = h("button", { class: "tbl__new", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: "New" }));
  private readonly foot = h("div", { class: "tbl__row tbl__row--foot", attrs: { role: "row" } });
  private readonly board = h("div", { class: "board" });
  private readonly lanes = h("div", { class: "board__lanes" });
  private readonly boardEmpty = h("div", { class: "board__empty" });
  private readonly cols = h("div", { class: "board__cols" });
  private shown = PAGE_ROWS;
  private colsKey = "";
  private collapsed = new Set<string>();
  private pendingTitle = "";

  constructor(
    private readonly host: HTMLElement,
    private readonly hooks: DbHooks,
  ) {
    const tabTable = this.viewTab("table", "Table", "table");
    const tabBoard = this.viewTab("board", "Board", "board");
    this.tabs.append(tabTable, tabBoard);
    const newBtn = h("button", { class: "btn btn--primary btn--sm db__new", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: "New" }));
    this.filterBtn.append(this.filterBadge);
    const tools = h("div", { class: "db__tools" }, this.filterBtn, this.sortBtn, this.groupBtn, this.search, this.exportBtn, newBtn);
    const bar = h("div", { class: "db__bar" }, this.tabs, tools);
    this.headWrap.append(this.head);
    this.scroller.append(this.body, this.more, this.newRow, this.foot);
    this.tbl.append(this.headWrap, this.scroller);
    this.lanes.append(this.cols);
    this.board.append(this.boardEmpty, this.lanes);
    this.stage.append(this.tbl, this.board);
    this.root.append(bar, this.summary, this.stage);
    host.append(this.root);

    this.search.addEventListener("input", () => {
      if (!this.item) return;
      this.item.view.search = this.search.value;
      this.shown = PAGE_ROWS;
      this.paint();
    });
    this.filterBtn.addEventListener("click", () => this.filterDialog());
    this.sortBtn.addEventListener("click", () => this.sortDialog());
    this.groupBtn.addEventListener("click", () => this.groupMenu());
    this.exportBtn.addEventListener("click", () => this.exportMenu());
    newBtn.addEventListener("click", () => this.addRow({}));
    this.newRow.addEventListener("click", () => this.addRow({}));
    this.more.addEventListener("click", () => {
      this.shown += PAGE_ROWS;
      this.paint();
    });
    this.scroller.addEventListener("scroll", () => {
      this.headWrap.scrollLeft = this.scroller.scrollLeft;
    });
    this.tabs.addEventListener("click", (e) => {
      const b = (e.target as HTMLElement).closest<HTMLElement>(".db__tab");
      if (!b || !this.item) return;
      this.item.view.type = b.dataset.view === "board" ? "board" : "table";
      this.paint();
    });
    this.body.addEventListener("click", (e) => this.onBodyClick(e));
    this.body.addEventListener("dblclick", (e) => this.onBodyDouble(e));
    this.body.addEventListener("keydown", (e) => this.onBodyKey(e));
    this.head.addEventListener("click", (e) => {
      const c = (e.target as HTMLElement).closest<HTMLElement>(".tbl__cell--head");
      if (c?.dataset.prop) this.cycleSort(c.dataset.prop);
    });
    this.foot.addEventListener("click", (e) => {
      const c = (e.target as HTMLElement).closest<HTMLElement>(".tbl__cell--calc");
      if (c?.dataset.prop) this.calcMenu(c, c.dataset.prop);
    });
    this.cols.addEventListener("click", (e) => this.onBoardClick(e));
    this.cols.addEventListener("dblclick", (e) => this.onBoardDouble(e));
    this.bindDrag();
  }

  private tool(icon: IconName, label: string, text: string): HTMLButtonElement {
    const b = h("button", { class: "db__tool", attrs: { type: "button", "aria-label": label, title: label } }, iconNode(icon));
    b.append(h("span", { class: "db__tool-label", text }));
    return b;
  }

  private viewTab(id: string, label: string, icon: IconName): HTMLButtonElement {
    return h("button", { class: "db__tab", attrs: { type: "button", role: "tab", "data-view": id } }, iconNode(icon), h("span", { class: "db__tab-name", text: label }));
  }

  open(item: DataItem): void {
    const changed = this.item?.id !== item.id;
    this.item = item;
    if (changed) {
      this.shown = PAGE_ROWS;
      this.collapsed.clear();
      this.colsKey = "";
      this.body.replaceChildren();
      this.cols.replaceChildren();
      this.search.value = item.view.search;
    }
    this.paint();
  }

  current(): DataItem | null {
    return this.item;
  }

  private props(): Property[] {
    return this.item?.schema ?? [];
  }

  private rows(): RowData[] {
    const item = this.item;
    if (!item) return [];
    return sortRows(item.schema, filterRows(item.schema, item.rows, item.view), item.view.sorts);
  }

  paint(): void {
    const item = this.item;
    if (!item) return;
    const v = item.view;
    const rows = this.rows();
    for (const b of Array.from(this.tabs.children) as HTMLElement[]) setAttr(b, "aria-selected", b.dataset.view === v.type ? "true" : "false");
    this.tbl.hidden = v.type !== "table";
    this.board.hidden = v.type !== "board";
    setAttr(this.filterBtn, "data-on", v.rules.length ? "true" : null);
    setText(this.filterBadge, v.rules.length ? String(v.rules.length) : "");
    setAttr(this.sortBtn, "data-on", v.sorts.length ? "true" : null);
    const groupId = v.type === "board" ? v.boardBy : v.group;
    setAttr(this.groupBtn, "data-on", groupId ? "true" : null);
    setText(this.summary, `${rows.length} of ${item.rows.length} rows${v.rules.length || v.search ? " match" : ""}`);
    if (v.type === "table") this.paintTable(rows);
    else this.paintBoard(rows);
  }

  private ensureCols(): void {
    const key = this.props().map((p) => p.id).join("|");
    if (key === this.colsKey) return;
    this.colsKey = key;
    const props = this.props();
    const widths = props.map(widthOf);
    this.tbl.style.setProperty("--tbl-cols", `44px ${widths.map((w) => `${w}px`).join(" ")}`);
    this.tbl.style.setProperty("--tbl-w", `${44 + widths.reduce((a, b) => a + b, 0)}px`);
    this.head.replaceChildren(
      h("div", { class: "tbl__cell tbl__cell--gutter" }),
      ...props.map((p) => h("div", { class: "tbl__cell tbl__cell--head", attrs: { role: "columnheader", "data-prop": p.id } }, h("span", { class: "tbl__hicon" }, iconNode(TYPE_INFO[p.type].icon)), h("span", { class: "tbl__hname", text: p.name }), h("span", { class: "demo-hsort" }))),
    );
    this.foot.replaceChildren(
      h("div", { class: "tbl__cell tbl__cell--gutter" }),
      ...props.map((p) => h("div", { class: "tbl__cell tbl__cell--calc", attrs: { "data-prop": p.id } }, h("span", { class: "tbl__calc-label" }), h("span", { class: "tbl__calc-value" }))),
    );
  }

  private makeRow(id: string): HTMLElement {
    const r = h("div", { class: "tbl__row tbl__row--flow", attrs: { role: "row", "data-row": id } });
    r.append(h("div", { class: "tbl__cell tbl__cell--gutter" }, h("button", { class: "icon-btn icon-btn--sm demo-rowdel", attrs: { type: "button", "aria-label": "Delete row", "data-action": "delete" } }, iconNode("trash"))));
    for (const p of this.props()) r.append(h("div", { class: "tbl__cell", attrs: { role: "gridcell", "data-prop": p.id } }, h("span", { class: "cell-value" })));
    return r;
  }

  private paintRow(el: HTMLElement, row: RowData): void {
    for (const p of this.props()) {
      const value = el.querySelector<HTMLElement>(`:scope > [data-prop="${p.id}"] > .cell-value`);
      if (value) paintCell(value, p, row, ctx, "table");
    }
  }

  private paintTable(rows: RowData[]): void {
    const item = this.item as DataItem;
    this.ensureCols();
    const v = item.view;
    const props = this.props();
    for (const c of Array.from(this.head.querySelectorAll<HTMLElement>(".tbl__cell--head"))) {
      const dir = sortedBy(item, c.dataset.prop ?? "");
      const mark = c.querySelector<HTMLElement>(".demo-hsort");
      if (mark && mark.dataset.dir !== dir) {
        mark.dataset.dir = dir;
        mark.replaceChildren(...(dir ? [iconNode(dir === "asc" ? "arrow-up" : "arrow-down")] : []));
      }
    }
    const group = props.find((p) => p.id === v.group);
    type Entry = { key: string; kind: "group" | "row"; g?: GroupResult; row?: RowData };
    const entries: Entry[] = [];
    const visible = rows.slice(0, this.shown);
    const byId = new Map(rows.map((r) => [r.id, r]));
    if (group) {
      let budget = this.shown;
      for (const g of groupRows(group, rows)) {
        entries.push({ key: `g:${g.key}`, kind: "group", g });
        if (this.collapsed.has(g.key)) continue;
        for (const id of g.rows) {
          if (budget-- <= 0) break;
          const row = byId.get(id);
          if (row) entries.push({ key: `r:${id}`, kind: "row", row });
        }
      }
    } else for (const row of visible) entries.push({ key: `r:${row.id}`, kind: "row", row });
    reconcile(
      this.body,
      entries,
      (x) => x.key,
      (x) => {
        if (x.kind === "group") return h("div", { class: "tbl__group" }, h("button", { class: "icon-btn icon-btn--sm tbl__gtoggle", attrs: { type: "button", "aria-label": "Collapse group" } }, iconNode("chevron-down")), h("span", { class: "tbl__glabel" }), h("span", { class: "tbl__gcount" }));
        return this.makeRow(x.row?.id ?? "");
      },
      (el, x) => {
        if (x.kind === "group" && x.g) {
          el.dataset.group = x.g.key;
          const label = el.children[1] as HTMLElement;
          const sig = `${x.g.label}|${x.g.color ?? ""}`;
          if (label.dataset.sig !== sig) {
            label.dataset.sig = sig;
            if (group && (group.type === "select" || group.type === "status") && x.g.key) label.replaceChildren(optionChip({ id: x.g.key, name: x.g.label, color: x.g.color ?? "default" }, group.type === "status"));
            else label.replaceChildren(document.createTextNode(x.g.label || "Empty"));
          }
          setText(el.children[2] as HTMLElement, String(x.g.count));
          setAttr(el, "data-collapsed", this.collapsed.has(x.g.key) ? "true" : null);
        } else if (x.row) this.paintRow(el, x.row);
      },
    );
    const total = group ? rows.length : visible.length;
    const hidden = rows.length - total;
    this.more.hidden = hidden <= 0 || !!group;
    if (!this.more.hidden) setText(this.more, `Load more (${rows.length - this.shown} more)`);
    for (const cell of Array.from(this.foot.querySelectorAll<HTMLElement>(".tbl__cell--calc"))) {
      const p = props.find((x) => x.id === cell.dataset.prop);
      const fn = p ? (v.calcs[p.id] ?? "") : "";
      const label = p ? (calcOptions(p).find((o) => o[0] === fn)?.[1] ?? "") : "";
      setText(cell.children[0] as HTMLElement, fn ? label : "Calculate");
      setText(cell.children[1] as HTMLElement, p && fn ? calcValue(p, rows, fn) : "");
      setAttr(cell, "data-empty", fn ? null : "true");
    }
    if (this.pendingTitle) {
      const cell = this.body.querySelector<HTMLElement>(`[data-row="${this.pendingTitle}"] [data-prop="title"]`);
      const row = item.rows.find((r) => r.id === this.pendingTitle);
      const p = props.find((x) => x.type === "title");
      this.pendingTitle = "";
      if (cell && row && p) this.edit(cell, p, row);
    }
  }

  private lookup(el: HTMLElement): { cell: HTMLElement; p: Property; row: RowData } | null {
    const cell = el.closest<HTMLElement>("[data-prop]");
    const rowEl = el.closest<HTMLElement>("[data-row]");
    if (!cell || !rowEl || !this.item) return null;
    const p = this.props().find((x) => x.id === cell.dataset.prop);
    const row = this.item.rows.find((r) => r.id === rowEl.dataset.row);
    return p && row ? { cell, p, row } : null;
  }

  private onBodyClick(e: MouseEvent): void {
    const target = e.target as HTMLElement;
    const del = target.closest<HTMLElement>("[data-action=delete]");
    if (del) {
      const id = del.closest<HTMLElement>("[data-row]")?.dataset.row;
      if (id && this.item) {
        this.item.rows = this.item.rows.filter((r) => r.id !== id);
        this.touched();
      }
      return;
    }
    const gt = target.closest<HTMLElement>(".tbl__group");
    if (gt?.dataset.group !== undefined) {
      const k = gt.dataset.group;
      if (this.collapsed.has(k)) this.collapsed.delete(k);
      else this.collapsed.add(k);
      this.paint();
      return;
    }
    const hit = this.lookup(target);
    if (!hit) return;
    for (const c of Array.from(this.body.querySelectorAll("[data-active]"))) c.removeAttribute("data-active");
    hit.cell.dataset.active = "true";
    if (hit.p.type === "checkbox") this.setValue(hit.row, hit.p, hit.row.values[hit.p.id] === true ? "" : "true");
    else if (hit.p.type === "select" || hit.p.type === "status") this.edit(hit.cell, hit.p, hit.row);
  }

  private onBodyDouble(e: MouseEvent): void {
    const hit = this.lookup(e.target as HTMLElement);
    if (hit && hit.p.type !== "checkbox") this.edit(hit.cell, hit.p, hit.row);
  }

  private onBodyKey(e: KeyboardEvent): void {
    if (e.key !== "Enter" || (e.target as HTMLElement).tagName === "INPUT") return;
    const hit = this.lookup(e.target as HTMLElement);
    if (hit) this.edit(hit.cell, hit.p, hit.row);
  }

  private touched(): void {
    if (this.item) touch(this.item);
    this.paint();
    this.hooks.changed();
  }

  setValue(row: RowData, p: Property, raw: string): void {
    const trimmed = raw.trim();
    if (p.type === "title") row.title = trimmed;
    else if (trimmed === "") delete row.values[p.id];
    else if (p.type === "number") {
      const n = Number(trimmed);
      if (Number.isFinite(n)) row.values[p.id] = n;
    } else if (p.type === "checkbox") row.values[p.id] = trimmed === "true";
    else if (p.type === "date") row.values[p.id] = { start: trimmed };
    else row.values[p.id] = trimmed;
    row.updated_at = Date.now();
    this.touched();
  }

  private edit(cell: HTMLElement, p: Property, row: RowData): void {
    if (cell.querySelector(".cell-input")) return;
    if (p.type === "select" || p.type === "status") {
      const entries: MenuEntry[] = (p.options ?? []).map((o) => ({ label: o.name, checked: row.values[p.id] === o.id, run: () => this.setValue(row, p, o.id) }));
      entries.push({ kind: "separator" }, { label: "Clear", run: () => this.setValue(row, p, "") });
      openMenu(cell, entries, { label: p.name });
      return;
    }
    if (p.type === "checkbox") {
      this.setValue(row, p, row.values[p.id] === true ? "" : "true");
      return;
    }
    const value = cell.querySelector<HTMLElement>(".cell-value");
    const field = h("input", { class: "cell-input", attrs: { type: p.type === "number" ? "number" : p.type === "date" ? "date" : "text", "aria-label": p.name } });
    if (p.type === "number") field.step = "any";
    field.value = p.type === "date" ? ((row.values[p.id] as { start?: string } | undefined)?.start ?? "").slice(0, 10) : cellText(p, row);
    if (value) value.hidden = true;
    cell.append(field);
    field.focus();
    field.select();
    let done = false;
    const finish = (commit: boolean) => {
      if (done) return;
      done = true;
      const text = field.value;
      field.remove();
      if (value) value.hidden = false;
      if (commit && text !== cellText(p, row)) this.setValue(row, p, text);
    };
    field.addEventListener("keydown", (e) => {
      if (e.key === "Enter") finish(true);
      else if (e.key === "Escape") finish(false);
    });
    field.addEventListener("blur", () => finish(true));
  }

  addRow(values: Record<string, unknown>): void {
    const item = this.item;
    if (!item) return;
    const n = item.rows.length + 1;
    const last = item.rows.reduce((m, r) => (r.sort_key > m ? r.sort_key : m), "");
    const row: RowData = { id: newBlockId(), title: "", icon: "", sort_key: String(Number(last || "0") + 1).padStart(6, "0"), values: { ...values }, number: n, version: 1, created_at: Date.now(), created_by: "", updated_at: Date.now(), updated_by: "" };
    item.rows.push(row);
    this.pendingTitle = item.view.type === "table" ? row.id : "";
    this.shown = Math.max(this.shown, item.rows.length);
    this.search.value = "";
    item.view.search = "";
    this.touched();
    if (item.view.type === "table") this.body.lastElementChild?.scrollIntoView({ block: "nearest" });
  }

  private cycleSort(prop: string): void {
    const item = this.item;
    if (!item) return;
    const cur = sortedBy(item, prop);
    item.view.sorts = cur === "" ? [{ prop, dir: "asc" }] : cur === "asc" ? [{ prop, dir: "desc" }] : [];
    this.paint();
  }

  private calcMenu(anchor: HTMLElement, prop: string): void {
    const item = this.item;
    const p = this.props().find((x) => x.id === prop);
    if (!item || !p) return;
    const cur = item.view.calcs[prop] ?? "";
    openMenu(
      anchor,
      calcOptions(p).map(([fn, label]) => ({
        label,
        checked: fn === cur,
        run: () => {
          if (fn) item.view.calcs[prop] = fn;
          else delete item.view.calcs[prop];
          this.paint();
        },
      })),
      { side: "top", label: "Calculate" },
    );
  }

  private groupMenu(): void {
    const item = this.item;
    if (!item) return;
    const board = item.view.type === "board";
    const cur = board ? item.view.boardBy : item.view.group;
    const set = (id: string) => {
      if (board) item.view.boardBy = id;
      else item.view.group = id;
      this.collapsed.clear();
      this.paint();
    };
    const entries: MenuEntry[] = item.schema.filter((p) => p.type !== "title" && p.type !== "text" && p.type !== "number").map((p) => ({ label: p.name, icon: TYPE_INFO[p.type].icon, checked: p.id === cur, run: () => set(p.id) }));
    if (!board) entries.unshift({ label: "No grouping", checked: cur === "", run: () => set("") });
    openMenu(this.groupBtn, entries, { label: "Group by" });
  }

  private exportMenu(): void {
    const item = this.item;
    if (!item) return;
    const table = () => ({ schema: item.schema, rows: this.rows() });
    openMenu(
      this.exportBtn,
      [
        { label: "Download CSV", icon: "download", run: () => this.hooks.download(`${item.title || "database"}.csv`, toDelimited(table()), "text/csv") },
        { label: "Download Markdown table", icon: "download", run: () => this.hooks.download(`${item.title || "database"}.md`, `${toMarkdownTable(table())}\n`, "text/markdown") },
        { label: "Copy Markdown table", icon: "copy", run: () => void navigator.clipboard?.writeText(toMarkdownTable(table())).then(() => toast({ message: "Table copied", tone: "success" }), () => toast({ message: "Clipboard not available here", tone: "warning" })) },
        { kind: "separator" },
        { label: "Insert into a page", icon: "page", run: () => this.hooks.insertMarkdown(toMarkdownTable(table())) },
      ],
      { label: "Export" },
    );
  }

  private valueControl(p: Property, rule: Rule, onChange: () => void): HTMLElement | null {
    if (!needsValue(rule.cond)) return null;
    if (p.type === "select" || p.type === "status" || p.type === "multi_select") {
      const opts: [string, string][] = [["", "Choose"], ...(p.options ?? []).map((o): [string, string] => [o.id, o.name])];
      const s = select(opts, rule.value);
      s.addEventListener("change", () => {
        rule.value = s.value;
        onChange();
      });
      return s;
    }
    const i = h("input", { class: "input input--sm", attrs: { type: p.type === "number" ? "number" : p.type === "date" ? "date" : "text", "aria-label": "Value", placeholder: "Value" } });
    i.value = rule.value;
    i.addEventListener("input", () => {
      rule.value = i.value;
      onChange();
    });
    return i;
  }

  private filterDialog(): void {
    const item = this.item;
    if (!item) return;
    simpleDialog(
      "Filter rows",
      (body, close) => {
        const list = h("div", { class: "demo-rules" });
        const draw = () => {
          list.replaceChildren();
          if (!item.view.rules.length) list.append(h("p", { class: "field__hint", text: "No rules yet. Add one to narrow the rows." }));
          for (const rule of item.view.rules) {
            const p = item.schema.find((x) => x.id === rule.prop) ?? (item.schema[0] as Property);
            const prop = select(item.schema.map((x): [string, string] => [x.id, x.name]), p.id);
            const cond = select(condsFor(p).map((c): [string, string] => [c, COND_LABELS[c] ?? c]), rule.cond);
            const value = this.valueControl(p, rule, () => this.paint());
            const rm = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": "Remove rule" } }, iconNode("close"));
            prop.addEventListener("change", () => {
              const np = item.schema.find((x) => x.id === prop.value) as Property;
              rule.prop = np.id;
              rule.cond = condsFor(np)[0] as string;
              rule.value = "";
              draw();
              this.paint();
            });
            cond.addEventListener("change", () => {
              rule.cond = cond.value;
              draw();
              this.paint();
            });
            rm.addEventListener("click", () => {
              item.view.rules = item.view.rules.filter((r) => r !== rule);
              draw();
              this.paint();
            });
            list.append(h("div", { class: "demo-rule" }, prop, cond, ...(value ? [value] : []), rm));
          }
        };
        const add = button("Add rule", "default", "plus");
        add.addEventListener("click", () => {
          const first = item.schema[0] as Property;
          item.view.rules.push({ prop: first.id, cond: condsFor(first)[0] as string, value: "" });
          draw();
          this.paint();
        });
        const clear = button("Clear all", "ghost");
        clear.addEventListener("click", () => {
          item.view.rules = [];
          draw();
          this.paint();
        });
        const done = button("Done", "primary");
        done.addEventListener("click", close);
        draw();
        body.append(list, actions(add, clear, done));
        return done;
      },
      undefined,
      "md",
    );
  }

  private sortDialog(): void {
    const item = this.item;
    if (!item) return;
    simpleDialog("Sort rows", (body, close) => {
      const prop = select([["", "No sort"], ...item.schema.map((x): [string, string] => [x.id, x.name])], item.view.sorts[0]?.prop ?? "");
      const dir = select<"asc" | "desc">([["asc", "Ascending"], ["desc", "Descending"]], item.view.sorts[0]?.dir ?? "asc");
      const apply = () => {
        item.view.sorts = prop.value ? [{ prop: prop.value, dir: dir.value === "desc" ? "desc" : "asc" }] : [];
        this.paint();
      };
      prop.addEventListener("change", apply);
      dir.addEventListener("change", apply);
      const done = button("Done", "primary");
      done.addEventListener("click", close);
      body.append(h("div", { class: "demo-rule" }, prop, dir), actions(done));
      return prop;
    });
  }

  private boardGroup(): Property | undefined {
    return this.props().find((p) => p.id === this.item?.view.boardBy);
  }

  private paintBoard(rows: RowData[]): void {
    const gp = this.boardGroup();
    this.boardEmpty.hidden = !!gp;
    this.lanes.hidden = !gp;
    if (!gp) {
      this.boardEmpty.replaceChildren(h("p", { text: "A board groups rows into columns. Choose a property to group by." }));
      const b = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("group"), h("span", { text: "Group by" }));
      b.addEventListener("click", () => this.groupMenu());
      this.boardEmpty.append(b);
      return;
    }
    const byId = new Map(rows.map((r) => [r.id, r]));
    const shownProps = this.props().filter((p) => p.type !== "title" && p.id !== gp.id).slice(0, 4);
    reconcile(
      this.cols,
      groupRows(gp, rows),
      (g) => g.key || "#none",
      () => {
        const head = h("div", { class: "board__colhead" }, h("span", { class: "board__label" }), h("span", { class: "board__count" }), h("button", { class: "icon-btn icon-btn--sm board__add", attrs: { type: "button", "aria-label": "New row in this group", "data-action": "add" } }, iconNode("plus")));
        return h("div", { class: "board__col" }, head, h("div", { class: "board__cards" }), h("button", { class: "board__new", attrs: { type: "button", "data-action": "add" } }, iconNode("plus"), h("span", { text: "New" })));
      },
      (col, g) => {
        col.dataset.group = g.key;
        setAttr(col, "data-color", g.color ?? null);
        const label = col.querySelector<HTMLElement>(".board__label") as HTMLElement;
        const sig = `${g.label}|${g.color ?? ""}|${gp.type}`;
        if (label.dataset.sig !== sig) {
          label.dataset.sig = sig;
          if ((gp.type === "select" || gp.type === "status") && g.key) label.replaceChildren(optionChip({ id: g.key, name: g.label, color: g.color ?? "default" }, gp.type === "status"));
          else label.replaceChildren(document.createTextNode(g.label || "Empty"));
        }
        setText(col.querySelector(".board__count") as HTMLElement, String(g.count));
        reconcile(
          col.querySelector(".board__cards") as HTMLElement,
          g.rows.map((id) => byId.get(id)).filter((r): r is RowData => !!r),
          (r) => r.id,
          (r) => {
            const card = h("div", { class: "card", attrs: { tabindex: "0", role: "article", draggable: "true", "data-row": r.id } }, h("div", { class: "card__title" }), h("div", { class: "card__props" }));
            return card;
          },
          (card, r) => {
            setText(card.querySelector(".card__title") as HTMLElement, r.title || "Untitled");
            setAttr(card.querySelector(".card__title") as HTMLElement, "data-untitled", r.title ? null : "true");
            const filled = shownProps.filter((p) => {
              const v = r.values[p.id];
              return !(v === undefined || v === null || v === "") || p.type === "checkbox";
            });
            reconcile(
              card.querySelector(".card__props") as HTMLElement,
              filled,
              (p) => p.id,
              (p) => h("div", { class: "card__prop", attrs: { "data-prop": p.id } }),
              (el, p) => paintCell(el, p, r, ctx, "card"),
            );
          },
        );
      },
    );
  }

  private onBoardClick(e: MouseEvent): void {
    const add = (e.target as HTMLElement).closest<HTMLElement>("[data-action=add]");
    const gp = this.boardGroup();
    if (!add || !gp) return;
    const key = add.closest<HTMLElement>(".board__col")?.dataset.group ?? "";
    const values: Record<string, unknown> = {};
    if (gp.type === "checkbox") values[gp.id] = key === "1";
    else if (key && (gp.type === "select" || gp.type === "status")) values[gp.id] = key;
    this.addRow(values);
  }

  private onBoardDouble(e: MouseEvent): void {
    const card = (e.target as HTMLElement).closest<HTMLElement>(".card");
    const title = (e.target as HTMLElement).closest<HTMLElement>(".card__title");
    const row = this.item?.rows.find((r) => r.id === card?.dataset.row);
    const tp = this.props().find((p) => p.type === "title");
    if (!title || !row || !tp) return;
    const field = h("input", { class: "cell-input", attrs: { type: "text", "aria-label": "Title" } });
    field.value = row.title;
    title.hidden = true;
    title.after(field);
    field.focus();
    field.select();
    let done = false;
    const finish = (commit: boolean) => {
      if (done) return;
      done = true;
      const text = field.value;
      field.remove();
      title.hidden = false;
      if (commit && text !== row.title) this.setValue(row, tp, text);
    };
    field.addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") finish(true);
      else if (ev.key === "Escape") finish(false);
    });
    field.addEventListener("blur", () => finish(true));
  }

  private bindDrag(): void {
    let dragging = "";
    this.cols.addEventListener("dragstart", (e) => {
      const card = (e.target as HTMLElement).closest<HTMLElement>(".card");
      if (!card) return;
      dragging = card.dataset.row ?? "";
      card.dataset.dragging = "true";
      e.dataTransfer?.setData("text/plain", dragging);
      if (e.dataTransfer) e.dataTransfer.effectAllowed = "move";
    });
    this.cols.addEventListener("dragend", () => {
      dragging = "";
      for (const c of Array.from(this.cols.querySelectorAll("[data-dragging],[data-drop]"))) {
        c.removeAttribute("data-dragging");
        c.removeAttribute("data-drop");
      }
    });
    this.cols.addEventListener("dragover", (e) => {
      const col = (e.target as HTMLElement).closest<HTMLElement>(".board__col");
      if (!col || !dragging) return;
      e.preventDefault();
      for (const c of Array.from(this.cols.querySelectorAll("[data-drop]"))) if (c !== col) c.removeAttribute("data-drop");
      col.dataset.drop = "true";
    });
    this.cols.addEventListener("drop", (e) => {
      const col = (e.target as HTMLElement).closest<HTMLElement>(".board__col");
      const gp = this.boardGroup();
      const row = this.item?.rows.find((r) => r.id === dragging);
      if (!col || !gp || !row) return;
      e.preventDefault();
      const key = col.dataset.group ?? "";
      if (gp.type === "checkbox") row.values[gp.id] = key === "1";
      else if (key) row.values[gp.id] = key;
      else delete row.values[gp.id];
      row.updated_at = Date.now();
      dragging = "";
      this.touched();
    });
  }
}
