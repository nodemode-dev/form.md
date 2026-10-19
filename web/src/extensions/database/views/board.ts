import { t } from "../../../engine/i18n";
import type { CollectionView, GroupResult, LaneResult, Property, RowData } from "../../../engine/types";
import { byId, h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import { cardCover, optionChip, paintCell } from "../cells";
import { editCell } from "../editors";
import { openPanel } from "../panel";
import { asList, computed } from "../props";
import type { ViewHost, ViewRenderer } from "../surface";
import { openGroupPanel, orderedProps, saveConfig } from "../toolbar";
import { chartFormat } from "./chart";
import { configKey, prefillFromFilter } from "./shared";

export function groupValue(p: Property, key: string, row: RowData | undefined, fromKey: string): unknown {
  switch (p.type) {
    case "select":
    case "status":
      return key || null;
    case "checkbox":
      return key === "true";
    case "multi_select":
    case "person":
    case "relation": {
      const cur = row ? asList(row.values[p.id]).filter((x) => x !== fromKey) : [];
      return key ? [...cur.filter((x) => x !== key), key] : cur;
    }
    case "text":
    case "url":
    case "email":
    case "phone":
      return key;
  }
  return undefined;
}

export function cardProps(host: ViewHost, ctx: ReturnType<ViewHost["ctx"]>): Property[] {
  return orderedProps(ctx, host.view().config)
    .filter((x) => x.p.type !== "title" && x.vp.hidden === false)
    .map((x) => x.p);
}

export function paintCard(card: HTMLElement, row: RowData, ctx: ReturnType<ViewHost["ctx"]>, props: Property[], coverSource: string): void {
  const cover = card.querySelector<HTMLElement>(".card__cover");
  const src = cardCover(ctx, row, coverSource);
  if (cover) {
    cover.hidden = !src;
    const img = cover.querySelector("img");
    if (src && img && img.getAttribute("src") !== src) img.src = src;
  }
  const title = card.querySelector<HTMLElement>(".card__title");
  const titleProp = ctx.col().schema.find((p) => p.type === "title");
  if (title && titleProp) paintCell(title, titleProp, row, ctx, "card");
  const list = card.querySelector<HTMLElement>(".card__props");
  if (!list) return;
  const shown = props.filter((p) => {
    const v = row.values[p.id];
    return !(v === undefined || v === null || v === "" || (Array.isArray(v) && !v.length)) || p.type === "checkbox" || isMeter(p);
  });
  reconcile(
    list,
    shown,
    (p) => p.id,
    () => h("div", { class: "card__prop" }),
    (el, p) => {
      el.dataset.prop = p.id;
      setAttr(el, "data-meter", isMeter(p) ? "true" : null);
      paintCell(el, p, row, ctx, "card");
    },
  );
}

function isMeter(p: Property): boolean {
  return p.type === "number" && (p.display === "bar" || p.display === "ring");
}

export function makeCard(withCover: boolean): HTMLElement {
  const card = h("div", { class: "card", attrs: { tabindex: "0", role: "article" } });
  if (withCover) card.append(h("div", { class: "card__cover" }, h("img", { attrs: { alt: "", decoding: "async", loading: "lazy", draggable: "false" } })));
  card.append(h("div", { class: "card__title" }), h("div", { class: "card__props" }));
  return card;
}

export function boardView(host: ViewHost): ViewRenderer {
  const session = host.makeSession({ groupLimit: 50, search: host.search() });
  const ctx = host.ctx(session);
  const el = h("div", { class: "board" });
  const lanesEl = h("div", { class: "board__lanes" });
  const empty = h("div", { class: "board__empty" });
  el.append(empty, lanesEl);
  let destroyed = false;
  let pendingTitle = "";
  const collapsedLanes = new Set<string>();

  const view = (): CollectionView => host.view();
  const groupProp = () => ctx.col().schema.find((p) => p.id === view().config.group?.prop);
  const layout = () => view().config.layout ?? {};

  const paintEmpty = () => {
    const missing = !groupProp();
    empty.hidden = !missing;
    lanesEl.hidden = missing;
    if (!missing) return;
    empty.replaceChildren(h("p", { text: t("A board groups pages into columns. Choose a property to group by.") }));
    if (host.canWrite()) {
      const b = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("group"), h("span", { text: t("Group by") }));
      b.addEventListener("click", () => openGroupPanel(b, ctx, host.viewId));
      empty.append(b);
    }
  };

  const column = () => {
    const head = h("div", { class: "board__colhead" }, h("button", { class: "icon-btn icon-btn--sm board__fold", attrs: { type: "button", "aria-label": t("Collapse group"), "aria-expanded": "true" } }, iconNode("chevron-left")), h("span", { class: "board__label" }), h("span", { class: "board__count" }), h("button", { class: "icon-btn icon-btn--sm board__add", attrs: { type: "button", "aria-label": t("New page in this group") } }, iconNode("plus")), h("button", { class: "icon-btn icon-btn--sm board__menu", attrs: { type: "button", "aria-label": t("Group options") } }, iconNode("more")));
    const meter = h("div", { class: "board__meter", attrs: { "aria-hidden": "true" } }, h("i"));
    const cards = h("div", { class: "board__cards" });
    const more = h("button", { class: "board__more", attrs: { type: "button" } });
    const add = h("button", { class: "board__new", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("New") }));
    const total = h("div", { class: "board__total" });
    return h("div", { class: "board__col" }, head, meter, cards, more, add, total);
  };

  const limits = () => (layout().limits ?? {}) as Record<string, number>;
  const folded = () => new Set((layout().collapsed ?? []) as string[]);
  const summary = new Map<string, { total?: number; progress?: number }>();
  const summaryKey = (g: string, lane: string | undefined) => `${lane ?? ""}\u0000${g}`;
  const totalSpec = () => layout().total as { prop?: string; fn?: string } | undefined;
  const progressProp = () => ctx.col().schema.find((p) => p.id === (layout().progress as string | undefined) && (p.type === "checkbox" || p.type === "number"));
  let summarySeq = 0;
  let summaryTimer: ReturnType<typeof setTimeout> | undefined;
  const loadSummary = () => {
    clearTimeout(summaryTimer);
    summaryTimer = setTimeout(() => void fetchSummary(), 250);
  };
  const fetchSummary = async () => {
    const g = view().config.group;
    const sub = view().config.sub_group;
    const tot = totalSpec();
    const prog = progressProp();
    const mine = ++summarySeq;
    if (!g || (!tot?.fn && !prog)) {
      if (summary.size) {
        summary.clear();
        paint();
      }
      return;
    }
    const base = { kind: "bar" as const, x: { ...g, hidden: [] }, limit: 500, ...(sub ? { series: { ...sub, hidden: [] } } : {}) };
    const ask = (fn: string, prop?: string) => host.engine.api.collections.chart(host.colId, { view_id: host.viewId, chart: { ...base, fn, ...(prop ? { prop } : {}) }, search: host.search() || undefined });
    try {
      const [tr, pr] = await Promise.all([tot?.fn ? ask(tot.fn, tot.fn === "count" ? undefined : tot.prop) : Promise.resolve(null), prog ? ask(prog.type === "checkbox" ? "percent_checked" : "average", prog.id) : Promise.resolve(null)]);
      if (mine !== summarySeq || destroyed) return;
      summary.clear();
      const take = (r: typeof tr, field: "total" | "progress") => {
        if (!r) return;
        for (const s of r.series)
          r.x.forEach((x, i) => {
            const k = summaryKey(x.key, sub ? s.key : undefined);
            const cur = summary.get(k) ?? {};
            cur[field] = s.values[i] ?? 0;
            summary.set(k, cur);
          });
      };
      take(tr, "total");
      take(pr, "progress");
      paint();
    } catch (e) {
      if (mine === summarySeq) host.engine.reportError(e);
    }
  };

  const paintColumn = (col: HTMLElement, g: GroupResult, laneKey: string | undefined) => {
    const p = groupProp();
    col.dataset.group = g.key;
    if (laneKey !== undefined) col.dataset.lane = laneKey;
    else delete col.dataset.lane;
    const label = col.querySelector<HTMLElement>(".board__label");
    if (label) {
      const sig = `${g.label}|${g.color ?? ""}`;
      if (label.dataset.sig !== sig) {
        label.dataset.sig = sig;
        if (p && (p.type === "select" || p.type === "status" || p.type === "multi_select") && g.key) label.replaceChildren(optionChip({ id: g.key, name: g.label, color: g.color ?? "default" }, p.type === "status"));
        else label.replaceChildren(document.createTextNode(g.label || t("Empty")));
      }
    }
    const limit = limits()[g.key || "#none"] ?? 0;
    setText(col.querySelector(".board__count") as HTMLElement, limit ? `${g.count} / ${limit}` : String(g.count));
    setAttr(col, "data-over", limit && g.count > limit ? "true" : null);
    setAttr(col, "data-color", g.color ?? null);
    const isFolded = folded().has(g.key || "#none");
    setAttr(col, "data-folded", isFolded ? "true" : null);
    const fold = col.querySelector<HTMLElement>(".board__fold");
    if (fold) {
      setAttr(fold, "aria-expanded", isFolded ? "false" : "true");
      setAttr(fold, "aria-label", isFolded ? t("Expand group") : t("Collapse group"));
    }
    const sum = summary.get(summaryKey(g.key, laneKey));
    const prog = progressProp();
    const meter = col.querySelector<HTMLElement>(".board__meter");
    if (meter) {
      meter.hidden = !prog;
      const share = prog ? Math.max(0, Math.min(1, prog.type === "checkbox" ? (sum?.progress ?? 0) : (sum?.progress ?? 0) / (prog.max || 100))) : 0;
      const bar = meter.firstElementChild as HTMLElement;
      const w = `${Math.round(share * 1000) / 10}%`;
      if (bar.style.width !== w) bar.style.width = w;
      setAttr(meter, "title", prog ? `${prog.name}: ${Math.round(share * 100)}%` : null);
    }
    const totalEl = col.querySelector<HTMLElement>(".board__total");
    const tot = totalSpec();
    if (totalEl) {
      totalEl.hidden = !tot?.fn;
      if (tot?.fn) {
        const f = chartFormat(ctx.col(), { kind: "bar", fn: tot.fn, ...(tot.prop ? { prop: tot.prop } : {}) });
        const p2 = ctx.col().schema.find((x) => x.id === tot.prop);
        setText(totalEl, sum?.total !== undefined ? `${t(tot.fn === "count" ? "Count" : tot.fn === "sum" ? "Sum" : "Average")}${p2 && tot.fn !== "count" ? ` ${p2.name}` : ""} ${f.fmt(sum.total)}` : "");
      }
    }
    const editable = host.canWrite() && !!p && !computed(p.type);
    for (const b of Array.from(col.querySelectorAll<HTMLElement>(".board__add, .board__new"))) b.hidden = !editable;
    const props = cardProps(host, ctx);
    const coverSource = String(layout().cover ?? "content");
    reconcile(
      col.querySelector(".board__cards") as HTMLElement,
      g.rows.filter((id) => session.rows.has(id)),
      (id) => id,
      (id) => {
        const card = makeCard(true);
        card.dataset.row = id;
        if (host.canWrite()) card.append(h("button", { class: "icon-btn icon-btn--sm card__edit", attrs: { type: "button", "aria-label": t("Edit on the card") } }, iconNode("pencil")));
        return card;
      },
      (card, id) => {
        const row = session.rows.get(id);
        if (row) paintCard(card, row, ctx, props, coverSource);
        setAttr(card, "data-size", String(layout().size ?? "medium"));
      },
    );
    const more = col.querySelector<HTMLElement>(".board__more");
    if (more) {
      more.hidden = g.count <= g.rows.length;
      if (!more.hidden) setText(more, t("Load more ({n} more)", { n: g.count - g.rows.length }));
    }
  };

  const paintColumns = (wrap: HTMLElement, groups: GroupResult[], laneKey: string | undefined) => {
    const shown = groups.filter((g) => !g.hidden);
    const hidden = groups.filter((g) => g.hidden);
    reconcile(wrap, shown, (g) => g.key || "#none", column, (c, g) => paintColumn(c, g, laneKey));
    if (hidden.length && laneKey === undefined) {
      const box = h("div", { class: "board__hidden" }, h("div", { class: "menu__heading", text: t("Hidden groups") }));
      for (const g of hidden) {
        const b = h("button", { class: "board__hidden-item", attrs: { type: "button", "data-unhide": g.key } }, h("span", { text: g.label || t("Empty") }), h("span", { class: "board__count", text: String(g.count) }));
        box.append(b);
      }
      wrap.append(box);
    }
  };

  const paint = () => {
    if (destroyed) return;
    paintEmpty();
    if (!groupProp()) return;
    if (session.lanes) {
      reconcile(
        lanesEl,
        session.lanes.filter((l) => !l.hidden),
        (l) => l.key || "#none",
        () => h("section", { class: "board__lane" }, h("button", { class: "board__lanehead", attrs: { type: "button" } }, iconNode("chevron-down"), h("span", { class: "board__lanelabel" }), h("span", { class: "board__count" })), h("div", { class: "board__cols" })),
        (laneEl, l: LaneResult) => {
          laneEl.dataset.lane = l.key;
          setText(laneEl.querySelector(".board__lanelabel") as HTMLElement, l.label || t("Empty"));
          setText(laneEl.querySelector(".board__lanehead .board__count") as HTMLElement, String(l.count));
          const collapsed = collapsedLanes.has(l.key);
          setAttr(laneEl, "data-collapsed", collapsed ? "true" : null);
          const cols = laneEl.querySelector(".board__cols") as HTMLElement;
          cols.hidden = collapsed;
          if (!collapsed) paintColumns(cols, l.groups, l.key);
        },
      );
    } else {
      let cols = lanesEl.querySelector<HTMLElement>(":scope > .board__cols");
      if (!cols || lanesEl.childElementCount !== 1) {
        cols = h("div", { class: "board__cols" });
        lanesEl.replaceChildren(cols);
      }
      for (const old of Array.from(cols.querySelectorAll(".board__hidden"))) old.remove();
      paintColumns(cols, session.groups ?? [], undefined);
    }
    if (pendingTitle) {
      const card = el.querySelector<HTMLElement>(`.card[data-row="${pendingTitle}"] .card__title`);
      const row = session.rows.get(pendingTitle);
      const tp = ctx.col().schema.find((p) => p.type === "title");
      if (card && row && tp) {
        pendingTitle = "";
        editCell(card, tp, row, ctx);
      }
    }
  };

  session.on((c) => {
    if (c.kind === "rows" || c.kind === "reset") loadSummary();
    if (c.kind === "rows") {
      const props = cardProps(host, ctx);
      const coverSource = String(layout().cover ?? "content");
      for (const id of c.ids) {
        const row = session.rows.get(id);
        if (!row) continue;
        for (const card of Array.from(el.querySelectorAll<HTMLElement>(`.card[data-row="${id}"]`))) paintCard(card, row, ctx, props, coverSource);
      }
      return;
    }
    if (c.kind === "reset") paint();
  });

  const ready = groupProp() ? session.load() : Promise.resolve();
  paintEmpty();

  const addIn = (groupKey: string, laneKey: string | undefined, top: boolean) => {
    const p = groupProp();
    if (!p) return;
    const values = prefillFromFilter(ctx, view());
    const v = groupValue(p, groupKey, undefined, "");
    if (v !== undefined && v !== null && !(Array.isArray(v) && !v.length)) values[p.id] = v;
    const sub = ctx.col().schema.find((x) => x.id === view().config.sub_group?.prop);
    if (sub && laneKey !== undefined) {
      const sv = groupValue(sub, laneKey, undefined, "");
      if (sv !== undefined && sv !== null && !(Array.isArray(sv) && !sv.length)) values[sub.id] = sv;
    }
    const g = laneKey !== undefined ? session.findGroup(groupKey, laneKey) : session.findGroup(groupKey);
    const body: { values: Record<string, unknown>; before_id?: string; after_id?: string } = { values };
    const first = g?.rows[0];
    const last = g?.rows.at(-1);
    if (top && first) body.before_id = first;
    else if (!top && last) body.after_id = last;
    void host.engine.collections
      .createRow(ctx.colId, body)
      .then((row) => {
        pendingTitle = row.id;
      })
      .catch(host.engine.reportError);
  };

  el.addEventListener("click", (e) => {
    if (suppressClick) return;
    const target = e.target as HTMLElement;
    const unhide = target.closest<HTMLElement>("[data-unhide]");
    if (unhide) {
      const key = unhide.dataset.unhide ?? "";
      saveConfig(ctx, host.viewId, (cfg) => (cfg.group ? { ...cfg, group: { ...cfg.group, hidden: (cfg.group.hidden ?? []).filter((x) => x !== key) } } : cfg), 0);
      return;
    }
    const lanehead = target.closest<HTMLElement>(".board__lanehead");
    if (lanehead) {
      const k = lanehead.closest<HTMLElement>(".board__lane")?.dataset.lane ?? "";
      if (collapsedLanes.has(k)) collapsedLanes.delete(k);
      else collapsedLanes.add(k);
      paint();
      return;
    }
    const editBtn = target.closest<HTMLElement>(".card__edit");
    if (editBtn) {
      const card = editBtn.closest<HTMLElement>(".card");
      if (card) startEdit(card, null);
      return;
    }
    const editing = target.closest<HTMLElement>('.card[data-editing="true"]');
    if (editing) {
      const propEl = target.closest<HTMLElement>(".card__prop[data-prop]");
      if (propEl || target.closest(".card__title")) {
        startEdit(editing, propEl);
        return;
      }
    }
    const col = target.closest<HTMLElement>(".board__col");
    const key = col?.dataset.group ?? "";
    const lane = col?.dataset.lane;
    const foldKey = key || "#none";
    if (target.closest(".board__fold") || (col?.dataset.folded === "true" && target.closest(".board__colhead"))) {
      saveConfig(ctx, host.viewId, (cfg) => {
        const cur = new Set(((cfg.layout ?? {}).collapsed ?? []) as string[]);
        if (cur.has(foldKey)) cur.delete(foldKey);
        else cur.add(foldKey);
        return { ...cfg, layout: { ...(cfg.layout ?? {}), collapsed: [...cur] } };
      }, 0);
      return;
    }
    if (target.closest(".board__add")) return addIn(key, lane, true);
    if (target.closest(".board__new")) return addIn(key, lane, false);
    if (target.closest(".board__more")) {
      void session.moreInGroup(key, lane).catch(host.engine.reportError);
      return;
    }
    const menuBtn = target.closest<HTMLElement>(".board__menu");
    if (menuBtn) {
      const setLimit = (n: number) =>
        saveConfig(ctx, host.viewId, (cfg) => {
          const cur = { ...(((cfg.layout ?? {}).limits ?? {}) as Record<string, number>) };
          if (n > 0) cur[foldKey] = n;
          else delete cur[foldKey];
          return { ...cfg, layout: { ...(cfg.layout ?? {}), limits: cur } };
        }, 0);
      const entries: MenuEntry[] = [
        { label: t("Collapse group"), icon: "chevron-left", run: () => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), collapsed: [...new Set([...(((cfg.layout ?? {}).collapsed ?? []) as string[]), foldKey])] } }), 0) },
        { label: t("Hide group"), icon: "eye-off", run: () => saveConfig(ctx, host.viewId, (cfg) => (cfg.group ? { ...cfg, group: { ...cfg.group, hidden: [...(cfg.group.hidden ?? []), key] } } : cfg), 0) },
        {
          label: limits()[foldKey] ? t("Change the limit") : t("Set a limit"),
          icon: "hash",
          run: () =>
            openPanel(
              menuBtn,
              (panel, close) => {
                const i = h("input", { class: "input input--sm", attrs: { type: "number", min: "0", step: "1", "aria-label": t("Most cards in this group"), placeholder: t("No limit") } });
                i.value = limits()[foldKey] ? String(limits()[foldKey]) : "";
                i.addEventListener("keydown", (ev) => {
                  if (ev.key !== "Enter") return;
                  setLimit(Math.max(0, Math.floor(Number(i.value) || 0)));
                  close();
                });
                panel.append(h("div", { class: "dbpanel__pad dbpanel__stack" }, h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Most cards in this group") }), i)));
              },
              { label: t("Group limit"), align: "end" },
            ),
        },
        ...(limits()[foldKey] ? [{ label: t("Remove the limit"), icon: "close" as const, run: () => setLimit(0) }] : []),
      ];
      openMenu(menuBtn, entries, { side: "bottom", align: "end", label: t("Group") });
      return;
    }
    const ref = target.closest<HTMLElement>("a.ref");
    if (ref?.dataset.id) {
      e.preventDefault();
      host.openRow(ref.dataset.id, e);
      return;
    }
    if (target.closest("a")) return;
    const card = target.closest<HTMLElement>(".card");
    if (card?.dataset.row) host.openRow(card.dataset.row, e);
  });
  el.addEventListener("keydown", (e) => {
    const card = (e.target as Element).closest<HTMLElement>(".card");
    if (card?.dataset.row && e.key === "Enter" && card.dataset.editing !== "true") host.openRow(card.dataset.row, e);
    if (card && e.key === "Escape") setAttr(card, "data-editing", null);
  });

  const startEdit = (card: HTMLElement, propEl: HTMLElement | null) => {
    const row = session.rows.get(card.dataset.row ?? "");
    if (!row || !host.canWrite()) return;
    for (const other of Array.from(el.querySelectorAll<HTMLElement>('.card[data-editing="true"]'))) if (other !== card) setAttr(other, "data-editing", null);
    setAttr(card, "data-editing", "true");
    const p = propEl ? ctx.col().schema.find((x) => x.id === propEl.dataset.prop) : ctx.col().schema.find((x) => x.type === "title");
    const anchorEl = propEl ?? card.querySelector<HTMLElement>(".card__title");
    if (p && anchorEl) editCell(anchorEl, p, row, ctx);
  };
  const leaveEdit = (e: PointerEvent) => {
    const target = e.target as Element;
    if (target.closest('.card[data-editing="true"]') || target.closest("#overlay-root")) return;
    for (const c of Array.from(el.querySelectorAll<HTMLElement>('.card[data-editing="true"]'))) setAttr(c, "data-editing", null);
  };
  document.addEventListener("pointerdown", leaveEdit, true);

  let suppressClick = false;
  el.addEventListener("pointerdown", (e) => {
    const card = (e.target as Element).closest<HTMLElement>(".card");
    if (!card?.dataset.row || e.button !== 0 || !host.canWrite() || (e.target as Element).closest("a")) return;
    const p = groupProp();
    if (!p) return;
    const id = card.dataset.row;
    const fromCol = card.closest<HTMLElement>(".board__col");
    const fromKey = fromCol?.dataset.group ?? "";
    const fromLane = fromCol?.dataset.lane;
    const startX = e.clientX;
    const startY = e.clientY;
    let ghost: HTMLElement | null = null;
    let line: HTMLElement | null = null;
    let drop: { col: HTMLElement; before?: string; after?: string } | null = null;
    const rect = card.getBoundingClientRect();
    const move = (ev: PointerEvent) => {
      if (!ghost && Math.hypot(ev.clientX - startX, ev.clientY - startY) < 6) return;
      if (!ghost) {
        ghost = card.cloneNode(true) as HTMLElement;
        ghost.classList.add("card--ghost");
        ghost.style.width = `${rect.width}px`;
        byId("overlay-root").append(ghost);
        card.classList.add("is-dragging");
        line = h("div", { class: "drop-line" });
      }
      ghost.style.transform = `translate(${ev.clientX - (startX - rect.left)}px, ${ev.clientY - (startY - rect.top)}px) rotate(1.5deg)`;
      const under = document.elementsFromPoint(ev.clientX, ev.clientY).find((x) => x.classList.contains("board__col")) as HTMLElement | undefined;
      if (!under) return;
      const cards = Array.from(under.querySelectorAll<HTMLElement>(".card")).filter((c) => c !== card);
      let before: string | undefined;
      let after: string | undefined;
      for (const c of cards) {
        const r = c.getBoundingClientRect();
        if (ev.clientY < r.top + r.height / 2) {
          before = c.dataset.row;
          break;
        }
        after = c.dataset.row;
      }
      drop = { col: under, ...(before ? { before } : after ? { after } : {}) };
      const list = under.querySelector(".board__cards") as HTMLElement;
      const ref = before ? list.querySelector(`.card[data-row="${before}"]`) : null;
      if (line) list.insertBefore(line, ref);
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      if (!ghost) return;
      suppressClick = true;
      setTimeout(() => (suppressClick = false), 0);
      ghost.remove();
      line?.remove();
      card.classList.remove("is-dragging");
      if (!drop) return;
      const toKey = drop.col.dataset.group ?? "";
      const toLane = drop.col.dataset.lane;
      const row = session.rows.get(id);
      const values: Record<string, unknown> = {};
      if (toKey !== fromKey) {
        if (computed(p.type)) return;
        values[p.id] = groupValue(p, toKey, row, fromKey);
      }
      const sub = ctx.col().schema.find((x) => x.id === view().config.sub_group?.prop);
      if (sub && toLane !== undefined && toLane !== fromLane && !computed(sub.type)) values[sub.id] = groupValue(sub, toLane, row, fromLane ?? "");
      const target: { before_id?: string; after_id?: string; values?: Record<string, unknown> } = {};
      if (!view().config.sorts?.length) {
        if (drop.before) target.before_id = drop.before;
        else if (drop.after) target.after_id = drop.after;
      }
      if (Object.keys(values).length) target.values = values;
      if (!target.before_id && !target.after_id && !target.values) return;
      const fromG = session.findGroup(fromKey, fromLane);
      const toG = session.findGroup(toKey, toLane);
      if (fromG && toG) {
        fromG.rows = fromG.rows.filter((x) => x !== id);
        fromG.count--;
        const at = drop.before ? toG.rows.indexOf(drop.before) : drop.after ? toG.rows.indexOf(drop.after) + 1 : toG.rows.length;
        toG.rows.splice(Math.max(0, at), 0, id);
        toG.count++;
        paint();
      }
      void host.engine.collections.moveRow(ctx.colId, id, target).then(() => session.scheduleRefresh([id]), (err: unknown) => {
        host.engine.reportError(err);
        session.scheduleRefresh([]);
      });
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  });

  return {
    el,
    ready,
    ctx,
    configChanged(prev, next) {
      loadSummary();
      if (configKey(prev.config) !== configKey(next.config)) {
        if (groupProp()) void session.load().catch(host.engine.reportError);
        else paint();
      } else paint();
    },
    schemaChanged() {
      paint();
    },
    setSearch(q) {
      session.options.search = q;
      void session.load().catch(host.engine.reportError);
    },
    addNew() {
      const first = session.lanes?.[0]?.groups.find((g) => !g.hidden) ?? session.groups?.find((g) => !g.hidden);
      if (first) addIn(first.key, session.lanes?.[0]?.key, true);
    },
    settings(anchor) {
      const lay = layout();
      const set = (change: Record<string, unknown>) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), ...change } }), 0);
      const fileProps = ctx.col().schema.filter((p) => p.type === "files");
      const entries: MenuEntry[] = [
        { label: t("Group by"), icon: "group", run: () => openGroupPanel(anchor, ctx, host.viewId) },
        { label: t("Sub-group by"), icon: "rows", run: () => openGroupPanel(anchor, ctx, host.viewId, "sub_group") },
        { kind: "heading", label: t("Card preview") },
        { label: t("None"), checked: lay.cover === "none", run: () => set({ cover: "none" }) },
        { label: t("Page content"), checked: (lay.cover ?? "content") === "content", run: () => set({ cover: "content" }) },
        { label: t("Page cover"), checked: lay.cover === "cover", run: () => set({ cover: "cover" }) },
        ...fileProps.map((p) => ({ label: p.name, checked: lay.cover === p.id, run: () => set({ cover: p.id }) })),
        { kind: "heading", label: t("Card size") },
        ...(["small", "medium", "large"] as const).map((s) => ({ label: t(s.charAt(0).toUpperCase() + s.slice(1)), checked: (lay.size ?? "medium") === s, run: () => set({ size: s }) })),
        { kind: "heading", label: t("Column total") },
        { label: t("None"), checked: !totalSpec()?.fn, run: () => set({ total: null }) },
        { label: t("Count cards"), checked: totalSpec()?.fn === "count", run: () => set({ total: { fn: "count" } }) },
        ...ctx
          .col()
          .schema.filter((p) => p.type === "number")
          .flatMap((p) => [
            { label: t("Sum of {prop}", { prop: p.name }), checked: totalSpec()?.fn === "sum" && totalSpec()?.prop === p.id, run: () => set({ total: { fn: "sum", prop: p.id } }) },
            { label: t("Average of {prop}", { prop: p.name }), checked: totalSpec()?.fn === "average" && totalSpec()?.prop === p.id, run: () => set({ total: { fn: "average", prop: p.id } }) },
          ]),
        { kind: "heading", label: t("Progress bar") },
        { label: t("None"), checked: !progressProp(), run: () => set({ progress: null }) },
        ...ctx
          .col()
          .schema.filter((p) => p.type === "checkbox" || p.type === "number")
          .map((p) => ({ label: p.type === "checkbox" ? t("Share of {prop} checked", { prop: p.name }) : t("Average of {prop}", { prop: p.name }), checked: progressProp()?.id === p.id, run: () => set({ progress: p.id }) })),
      ];
      openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Layout") });
    },
    destroy() {
      destroyed = true;
      clearTimeout(summaryTimer);
      document.removeEventListener("pointerdown", leaveEdit, true);
      session.close();
      el.remove();
    },
  };
}
