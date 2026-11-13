import { t } from "../../../engine/i18n";
import type { CollectionView, GroupResult, RowData } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import { optionChip, paintCell } from "../cells";
import { editCell } from "../editors";
import type { ViewHost, ViewRenderer } from "../surface";
import { saveConfig } from "../toolbar";
import { cardProps, makeCard, paintCard } from "./board";
import { configKey, prefillFromFilter } from "./shared";

type Kind = "list" | "gallery";

function itemsView(host: ViewHost, kind: Kind): ViewRenderer {
  const session = host.makeSession({ limit: host.inline ? 25 : 60, groupLimit: host.inline ? 12 : 30, search: host.search() });
  const ctx = host.ctx(session);
  const el = h("div", { class: kind === "list" ? "lview" : "gallery" });
  const groupsEl = h("div", { class: "items__groups" });
  const more = h("button", { class: "tbl__more", attrs: { type: "button" } });
  const add = h("button", { class: "tbl__new", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("New") }));
  el.append(groupsEl, more, add);
  let destroyed = false;
  let pendingTitle = "";
  let shown = host.inline ? 25 : 60;
  const collapsed = new Set<string>();

  const view = (): CollectionView => host.view();
  const layout = () => view().config.layout ?? {};

  const item = () => {
    if (kind === "gallery") return makeCard(true);
    const li = h("div", { class: "lrow", attrs: { tabindex: "0", role: "listitem" } }, h("div", { class: "lrow__title" }), h("div", { class: "lrow__props" }));
    return li;
  };

  const paintItem = (node: HTMLElement, row: RowData) => {
    const props = cardProps(host, ctx);
    if (kind === "gallery") {
      paintCard(node, row, ctx, props, String(layout().cover ?? "content"));
      setAttr(node, "data-size", String(layout().size ?? "medium"));
      setAttr(node, "data-fit", layout().fit === "contain" ? "contain" : null);
      return;
    }
    const tp = ctx.col().schema.find((p) => p.type === "title");
    if (tp) paintCell(node.firstElementChild as HTMLElement, tp, row, ctx, "list");
    reconcile(
      node.lastElementChild as HTMLElement,
      props.filter((p) => {
        const v = row.values[p.id];
        return !(v === undefined || v === null || v === "" || (Array.isArray(v) && !v.length));
      }),
      (p) => p.id,
      () => h("span", { class: "lrow__prop" }),
      (c, p) => paintCell(c, p, row, ctx, "list"),
    );
  };

  const paintItems = (wrap: HTMLElement, ids: string[]) => {
    reconcile(
      wrap,
      ids.filter((id) => session.rows.has(id)),
      (id) => id,
      (id) => {
        const n = item();
        n.dataset.row = id;
        return n;
      },
      (n, id) => {
        const row = session.rows.get(id);
        if (row) paintItem(n, row);
      },
    );
  };

  const paint = () => {
    if (destroyed) return;
    const groups: GroupResult[] | null = session.groups ?? null;
    if (!groups) {
      let wrap = groupsEl.querySelector<HTMLElement>(":scope > .items__list");
      if (!wrap || groupsEl.childElementCount !== 1) {
        wrap = h("div", { class: "items__list" });
        groupsEl.replaceChildren(wrap);
      }
      const ids = session.ordered().slice(0, shown);
      paintItems(wrap, ids);
      more.hidden = session.total <= ids.length;
      if (!more.hidden) setText(more, t("Load more ({n} more)", { n: session.total - ids.length }));
    } else {
      more.hidden = true;
      reconcile(
        groupsEl,
        groups.filter((g) => !g.hidden),
        (g) => g.key || "#none",
        () => h("section", { class: "items__group" }, h("button", { class: "items__ghead", attrs: { type: "button" } }, iconNode("chevron-down"), h("span", { class: "items__glabel" }), h("span", { class: "board__count" })), h("div", { class: "items__list" }), h("button", { class: "tbl__more", attrs: { type: "button" } })),
        (sec, g) => {
          sec.dataset.group = g.key;
          const label = sec.querySelector(".items__glabel") as HTMLElement;
          const p = ctx.col().schema.find((x) => x.id === view().config.group?.prop);
          const sig = `${g.label}|${g.color ?? ""}`;
          if (label.dataset.sig !== sig) {
            label.dataset.sig = sig;
            if (p && (p.type === "select" || p.type === "status" || p.type === "multi_select") && g.key) label.replaceChildren(optionChip({ id: g.key, name: g.label, color: g.color ?? "default" }, p.type === "status"));
            else label.replaceChildren(document.createTextNode(g.label || t("Empty")));
          }
          setText(sec.querySelector(".items__ghead .board__count") as HTMLElement, String(g.count));
          const isCollapsed = collapsed.has(g.key);
          setAttr(sec, "data-collapsed", isCollapsed ? "true" : null);
          const list = sec.querySelector(".items__list") as HTMLElement;
          list.hidden = isCollapsed;
          if (!isCollapsed) paintItems(list, g.rows);
          const mb = sec.querySelector(":scope > .tbl__more") as HTMLElement;
          mb.hidden = isCollapsed || g.count <= g.rows.length;
          if (!mb.hidden) setText(mb, t("Load more ({n} more)", { n: g.count - g.rows.length }));
        },
      );
    }
    add.hidden = !host.canWrite();
    if (pendingTitle) {
      const target = el.querySelector<HTMLElement>(`[data-row="${pendingTitle}"] ${kind === "gallery" ? ".card__title" : ".lrow__title"}`);
      const row = session.rows.get(pendingTitle);
      const tp = ctx.col().schema.find((p) => p.type === "title");
      if (target && row && tp) {
        pendingTitle = "";
        editCell(target, tp, row, ctx);
      }
    }
  };

  session.on((c) => {
    if (c.kind === "rows") {
      for (const id of c.ids) {
        const row = session.rows.get(id);
        if (row) for (const n of Array.from(el.querySelectorAll<HTMLElement>(`[data-row="${id}"]`))) paintItem(n, row);
      }
      return;
    }
    if (c.kind === "reset") paint();
  });

  const boot = host.engine.collections.takeBootQuery(host.viewId);
  let ready: Promise<void>;
  if (boot && !host.search() && !session.options.dateRange) {
    session.seed(boot);
    paint();
    ready = Promise.resolve();
  } else ready = session.load();

  const addNew = () => {
    const values = prefillFromFilter(ctx, view());
    void host.engine.collections
      .createRow(ctx.colId, { values, top: true })
      .then((row) => {
        pendingTitle = row.id;
      })
      .catch(host.engine.reportError);
  };

  el.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;
    const head = target.closest<HTMLElement>(".items__ghead");
    if (head) {
      const k = head.closest<HTMLElement>(".items__group")?.dataset.group ?? "";
      if (collapsed.has(k)) collapsed.delete(k);
      else collapsed.add(k);
      paint();
      return;
    }
    const mb = target.closest<HTMLElement>(".tbl__more");
    if (mb) {
      const sec = mb.closest<HTMLElement>(".items__group");
      if (sec) void session.moreInGroup(sec.dataset.group ?? "").catch(host.engine.reportError);
      else {
        shown += 60;
        const need = shown;
        session.setVisible(0, need);
        paint();
      }
      return;
    }
    if (target.closest(".tbl__new")) return addNew();
    const ref = target.closest<HTMLElement>("a.ref");
    if (ref?.dataset.id) {
      e.preventDefault();
      host.openRow(ref.dataset.id, e);
      return;
    }
    if (target.closest("a")) return;
    const n = target.closest<HTMLElement>("[data-row]");
    if (n?.dataset.row) host.openRow(n.dataset.row, e);
  });
  el.addEventListener("keydown", (e) => {
    const n = (e.target as Element).closest<HTMLElement>("[data-row]");
    if (n?.dataset.row && e.key === "Enter") host.openRow(n.dataset.row, e);
  });

  const autoMore = host.inline
    ? null
    : new IntersectionObserver(
        (entries) => {
          if (!destroyed && !more.hidden && entries.some((x) => x.isIntersecting)) more.click();
        },
        { rootMargin: "600px 0px" },
      );
  autoMore?.observe(more);

  return {
    el,
    ready,
    ctx,
    configChanged(prev, next) {
      if (configKey(prev.config) !== configKey(next.config)) void session.load().catch(host.engine.reportError);
      else paint();
    },
    schemaChanged: paint,
    setSearch(q) {
      session.options.search = q;
      void session.load().catch(host.engine.reportError);
    },
    addNew,
    ...(kind === "gallery"
      ? {
          settings(anchor: HTMLElement) {
            const lay = layout();
            const set = (change: Record<string, unknown>) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), ...change } }), 0);
            const entries: MenuEntry[] = [
              { kind: "heading", label: t("Card preview") },
              { label: t("None"), checked: lay.cover === "none", run: () => set({ cover: "none" }) },
              { label: t("Page content"), checked: (lay.cover ?? "content") === "content", run: () => set({ cover: "content" }) },
              { label: t("Page cover"), checked: lay.cover === "cover", run: () => set({ cover: "cover" }) },
              ...ctx.col().schema.filter((p) => p.type === "files").map((p) => ({ label: p.name, checked: lay.cover === p.id, run: () => set({ cover: p.id }) })),
              { kind: "heading", label: t("Card size") },
              ...(["small", "medium", "large"] as const).map((s) => ({ label: t(s.charAt(0).toUpperCase() + s.slice(1)), checked: (lay.size ?? "medium") === s, run: () => set({ size: s }) })),
              { kind: "heading", label: t("Image") },
              { label: t("Fill the card"), checked: lay.fit !== "contain", run: () => set({ fit: "cover" }) },
              { label: t("Fit the whole image"), checked: lay.fit === "contain", run: () => set({ fit: "contain" }) },
            ];
            openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Layout") });
          },
        }
      : {}),
    destroy() {
      destroyed = true;
      autoMore?.disconnect();
      session.close();
      el.remove();
    },
  };
}

export function listView(host: ViewHost): ViewRenderer {
  return itemsView(host, "list");
}

export function galleryView(host: ViewHost): ViewRenderer {
  return itemsView(host, "gallery");
}
