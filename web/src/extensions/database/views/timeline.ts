import { locale } from "../../../engine/format";
import { t } from "../../../engine/i18n";
import type { CollectionView, Property, RowData } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { reconcile } from "../../../ui/list";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import { pageIcon } from "../../../ui/pageicon";
import { paintCell } from "../cells";
import { editCell } from "../editors";
import { asList, computed, type DateValue, isoDay, parseISO } from "../props";
import type { ViewHost, ViewRenderer } from "../surface";
import { saveConfig } from "../toolbar";
import { dateProps, spanOf } from "./calendar";
import { prefillFromFilter } from "./shared";

const DAY = 86400000;
const ROW_H = 36;
const ZOOM: Record<string, number> = { hours: 960, day: 48, week: 22, biweek: 12, month: 6, quarter: 2.4, year: 0.8 };
const ZOOM_LABEL: [string, string][] = [["hours", "Hours"], ["day", "Day"], ["week", "Week"], ["biweek", "Bi-week"], ["month", "Month"], ["quarter", "Quarter"], ["year", "Year"]];

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function addDays(d: Date, n: number): Date {
  const x = new Date(d);
  x.setDate(x.getDate() + n);
  return x;
}

export function timelineView(host: ViewHost): ViewRenderer {
  const session = host.makeSession({ limit: 1000, noGroups: true, search: host.search() });
  const ctx = host.ctx(session);
  const view = (): CollectionView => host.view();
  const layout = () => view().config.layout ?? {};
  const dateProp = (): Property | undefined => {
    const want = layout().date as string | undefined;
    const list = dateProps(host);
    return list.find((p) => p.id === want) ?? list.find((p) => p.type === "date") ?? list[0];
  };
  const endProp = (): Property | undefined => ctx.col().schema.find((p) => p.id === layout().end);
  const depsProp = (): Property | undefined => ctx.col().schema.find((p) => p.id === layout().deps && p.type === "relation");
  const zoom = () => (ZOOM[String(layout().zoom)] ? String(layout().zoom) : "week");
  const px = () => ZOOM[zoom()] ?? 22;
  const showTable = () => layout().table !== false;

  const el = h("div", { class: "tl" });
  const zoomBtn = h("button", { class: "btn btn--sm", attrs: { type: "button" } });
  const todayBtn = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Today") });
  const tableBtn = h("button", { class: "btn btn--sm", attrs: { type: "button" } });
  const bar = h("div", { class: "cal__head" }, h("span", { class: "cal__spacer" }), tableBtn, zoomBtn, todayBtn);
  const side = h("div", { class: "tl__side" });
  const sideHead = h("div", { class: "tl__sidehead", text: t("Name") });
  const sideRows = h("div", { class: "tl__siderows" });
  side.append(sideHead, sideRows);
  const scroll = h("div", { class: "tl__scroll" });
  const scale = h("div", { class: "tl__scale" });
  const canvas = h("div", { class: "tl__canvas" });
  const todayLine = h("div", { class: "tl__today" });
  const svgNS = "http://www.w3.org/2000/svg";
  const deps = document.createElementNS(svgNS, "svg");
  deps.setAttribute("class", "tl__deps");
  canvas.append(todayLine, deps);
  scroll.append(scale, canvas);
  const main = h("div", { class: "tl__main" }, side, scroll);
  const empty = h("div", { class: "board__empty" });
  el.append(empty, bar, main);
  let origin = startOfDay(addDays(new Date(), -30));
  let days = 120;
  let destroyed = false;
  let scrolledToday = false;
  let pendingTitle = "";

  const xOf = (d: Date) => ((d.getTime() - origin.getTime()) / DAY) * px();
  const dayAtX = (x: number) => startOfDay(new Date(origin.getTime() + (x / px()) * DAY));

  const spans = () => {
    const p = dateProp();
    const out: { id: string; row: RowData; start: Date | null; end: Date | null }[] = [];
    for (const id of session.ordered()) {
      const row = session.rows.get(id);
      if (!row) continue;
      const s = p ? spanOf(row, p) : null;
      let end = s?.end ?? null;
      const ep = endProp();
      if (ep && s) {
        const e = spanOf(row, ep);
        if (e && e.start >= s.start) end = e.start;
      }
      out.push({ id, row, start: s?.start ?? null, end });
    }
    return out;
  };

  const paintScale = () => {
    const z = zoom();
    const ticks: { key: string; x: number; label: string; major: boolean }[] = [];
    const fmtMonth = new Intl.DateTimeFormat(locale(), { month: "short", year: "numeric" });
    const fmtDay = new Intl.DateTimeFormat(locale(), { day: "numeric" });
    const fmtHour = new Intl.DateTimeFormat(locale(), { hour: "numeric" });
    for (let i = 0; i <= days; i++) {
      const d = addDays(origin, i);
      const first = d.getDate() === 1;
      if (z === "hours") {
        for (let hr = 0; hr < 24; hr += 2) {
          const dt = new Date(d);
          dt.setHours(hr);
          ticks.push({ key: `${isoDay(d)}:${hr}`, x: xOf(dt), label: hr === 0 ? `${fmtDay.format(d)} ${fmtHour.format(dt)}` : fmtHour.format(dt), major: hr === 0 });
        }
      } else if (z === "day" || z === "week") {
        if (z === "day" || d.getDay() === 1 || first) ticks.push({ key: isoDay(d), x: xOf(d), label: first ? fmtMonth.format(d) : fmtDay.format(d), major: first });
      } else if (first && (z !== "year" || d.getMonth() % 3 === 0)) ticks.push({ key: isoDay(d), x: xOf(d), label: fmtMonth.format(d), major: d.getMonth() === 0 });
    }
    reconcile(
      scale,
      ticks,
      (k) => k.key,
      () => h("span", { class: "tl__tick" }),
      (n, k) => {
        n.style.transform = `translateX(${k.x}px)`;
        setText(n, k.label);
        setAttr(n, "data-major", k.major ? "true" : null);
      },
    );
  };

  const paint = () => {
    if (destroyed) return;
    const p = dateProp();
    empty.hidden = !!p;
    bar.hidden = main.hidden = !p;
    if (!p) {
      empty.replaceChildren(h("p", { text: t("A timeline plots pages by a date range. Add a date property to use it.") }));
      return;
    }
    setText(zoomBtn, t(ZOOM_LABEL.find((z) => z[0] === zoom())?.[1] ?? "Week"));
    setText(tableBtn, showTable() ? t("Hide table") : t("Show table"));
    side.hidden = !showTable();
    const list = spans();
    let lo = startOfDay(addDays(new Date(), -30));
    let hi = startOfDay(addDays(new Date(), 60));
    for (const s of list) {
      if (s.start && s.start < lo) lo = s.start;
      if (s.end && s.end > hi) hi = s.end;
    }
    const pad = Math.ceil(240 / px()) + 7;
    const newOrigin = addDays(lo, -pad);
    const shift = newOrigin.getTime() !== origin.getTime() ? xOf(newOrigin) : 0;
    origin = newOrigin;
    days = Math.ceil((hi.getTime() - origin.getTime()) / DAY) + pad;
    const width = days * px();
    canvas.style.width = `${width}px`;
    scale.style.width = `${width}px`;
    canvas.style.height = `${Math.max(1, list.length + 1) * ROW_H}px`;
    if (shift) scroll.scrollLeft -= shift;
    paintScale();
    todayLine.style.transform = `translateX(${xOf(startOfDay(new Date())) + px() / 2}px)`;
    const editable = host.canWrite() && !computed(p.type);
    const tp = ctx.col().schema.find((x) => x.type === "title");
    reconcile(
      sideRows,
      list,
      (s) => s.id,
      (s) => h("div", { class: "tl__siderow", attrs: { "data-row": s.id, tabindex: "0" } }, h("span", { class: "tl__name" })),
      (n, s) => {
        if (tp) paintCell(n.firstElementChild as HTMLElement, tp, s.row, ctx, "list");
      },
    );
    sideRows.style.height = `${(list.length + 1) * ROW_H}px`;
    const bars = list.map((s, i) => ({ ...s, i }));
    reconcile(
      canvas,
      [{ id: "#today", i: -1 }, { id: "#deps", i: -1 }, ...bars],
      (s) => s.id,
      (s) => {
        if (s.id === "#today") return todayLine;
        if (s.id === "#deps") return deps as unknown as HTMLElement;
        return h("div", { class: "tl__bar", attrs: { "data-row": s.id, tabindex: "0" } }, h("span", { class: "tl__h tl__h--start", attrs: { "aria-hidden": "true" } }), h("span", { class: "tl__icon" }), h("span", { class: "tl__title" }), h("span", { class: "tl__h tl__h--end", attrs: { "aria-hidden": "true" } }));
      },
      (n, s) => {
        if (s.i < 0) return;
        const sp = bars[s.i];
        if (!sp) return;
        const top = s.i * ROW_H + 6;
        if (!sp.start) {
          n.classList.add("tl__bar--nodate");
          n.style.transform = `translate(${scroll.scrollLeft + 12}px, ${top}px)`;
          n.style.width = "";
          setText(n.children[2] as HTMLElement, `${sp.row.title || t("Untitled")}  ·  ${t("No date")}`);
          return;
        }
        n.classList.remove("tl__bar--nodate");
        const end = sp.end ?? sp.start;
        const x = xOf(sp.start);
        const w = Math.max(px(), xOf(addDays(end, 1)) - x);
        n.style.transform = `translate(${x}px, ${top}px)`;
        n.style.width = `${w}px`;
        const icon = n.children[1] as HTMLElement;
        if (icon.dataset.icon !== sp.row.icon) {
          icon.dataset.icon = sp.row.icon;
          icon.replaceChildren(pageIcon(sp.row.icon, "page"));
        }
        setText(n.children[2] as HTMLElement, sp.row.title || t("Untitled"));
        for (const hnd of Array.from(n.querySelectorAll<HTMLElement>(".tl__h"))) hnd.hidden = !editable;
      },
    );
    paintDeps(bars);
    if (!scrolledToday && scroll.clientWidth) {
      scrolledToday = true;
      scroll.scrollLeft = Math.max(0, xOf(startOfDay(new Date())) - scroll.clientWidth / 3);
    }
    if (pendingTitle) {
      const n = canvas.querySelector<HTMLElement>(`.tl__bar[data-row="${pendingTitle}"]`);
      const row = session.rows.get(pendingTitle);
      if (n && row && tp) {
        pendingTitle = "";
        editCell(n, tp, row, ctx);
      }
    }
  };

  const paintDeps = (bars: { id: string; row: RowData; start: Date | null; end: Date | null; i: number }[]) => {
    const dp = depsProp();
    deps.setAttribute("width", canvas.style.width.replace("px", "") || "0");
    deps.setAttribute("height", canvas.style.height.replace("px", "") || "0");
    const paths: { key: string; d: string }[] = [];
    if (dp) {
      const byId = new Map(bars.map((b) => [b.id, b]));
      for (const b of bars) {
        if (!b.start) continue;
        for (const dep of asList(b.row.values[dp.id])) {
          const from = byId.get(dep);
          if (!from?.start) continue;
          const x1 = xOf(addDays(from.end ?? from.start, 1));
          const y1 = from.i * ROW_H + ROW_H / 2;
          const x2 = xOf(b.start);
          const y2 = b.i * ROW_H + ROW_H / 2;
          const mid = Math.max(x1 + 10, Math.min(x2 - 10, (x1 + x2) / 2));
          paths.push({ key: `${dep}>${b.id}`, d: `M${x1} ${y1} H${mid} V${y2} H${x2 - 4} m-5 -4 l5 4 l-5 4` });
        }
      }
    }
    const existing = new Map<string, SVGPathElement>();
    for (const n of Array.from(deps.children) as SVGPathElement[]) existing.set(n.dataset.key ?? "", n);
    for (const pth of paths) {
      let n = existing.get(pth.key);
      if (n) existing.delete(pth.key);
      else {
        n = document.createElementNS(svgNS, "path");
        n.dataset.key = pth.key;
        deps.append(n);
      }
      if (n.getAttribute("d") !== pth.d) n.setAttribute("d", pth.d);
    }
    for (const n of existing.values()) n.remove();
  };

  session.on((c) => {
    if (c.kind !== "loading") paint();
  });
  const ready = dateProp() ? session.load() : Promise.resolve();
  paint();
  const sized = new ResizeObserver(() => {
    if (!scrolledToday && scroll.clientWidth) paint();
  });
  sized.observe(scroll);

  scroll.addEventListener("scroll", () => {
    sideRows.style.transform = `translateY(${-scroll.scrollTop}px)`;
  }, { passive: true });

  const create = (day: Date) => {
    const p = dateProp();
    if (!p || computed(p.type)) return;
    const values = prefillFromFilter(ctx, view(), { [p.id]: { start: isoDay(day), end: isoDay(addDays(day, 2)) } });
    void host.engine.collections
      .createRow(ctx.colId, { values })
      .then((row) => {
        pendingTitle = row.id;
      })
      .catch(host.engine.reportError);
  };

  let suppress = false;
  el.addEventListener("click", (e) => {
    if (suppress) return;
    const n = (e.target as Element).closest<HTMLElement>("[data-row]");
    if (n?.dataset.row) host.openRow(n.dataset.row, e);
  });
  canvas.addEventListener("dblclick", (e) => {
    if ((e.target as Element).closest(".tl__bar") || !host.canWrite()) return;
    const r = canvas.getBoundingClientRect();
    create(dayAtX(e.clientX - r.left));
  });

  canvas.addEventListener("pointerdown", (e) => {
    const n = (e.target as Element).closest<HTMLElement>(".tl__bar");
    const p = dateProp();
    if (!n?.dataset.row || !p || computed(p.type) || !host.canWrite() || e.button !== 0) return;
    const row = session.rows.get(n.dataset.row);
    if (!row) return;
    const cur = row.values[p.id] as DateValue | undefined;
    if (!cur?.start) return;
    const handle = (e.target as Element).closest(".tl__h");
    const which = handle?.classList.contains("tl__h--start") ? "start" : handle ? "end" : "move";
    const sx = e.clientX;
    const baseX = n.getBoundingClientRect().left;
    const baseW = n.getBoundingClientRect().width;
    let moved = false;
    let dd = 0;
    const move = (ev: PointerEvent) => {
      const dx = ev.clientX - sx;
      if (!moved && Math.abs(dx) < 4) return;
      moved = true;
      dd = Math.round(dx / px());
      const r = canvas.getBoundingClientRect();
      const x0 = baseX - r.left;
      if (which === "move") n.style.transform = n.style.transform.replace(/translate\([^,]+,/, `translate(${x0 + dd * px()}px,`);
      else if (which === "end") n.style.width = `${Math.max(px(), baseW + dd * px())}px`;
      else {
        n.style.transform = n.style.transform.replace(/translate\([^,]+,/, `translate(${x0 + dd * px()}px,`);
        n.style.width = `${Math.max(px(), baseW - dd * px())}px`;
      }
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      if (!moved) return;
      suppress = true;
      setTimeout(() => (suppress = false), 0);
      if (!dd) return paint();
      const shiftIso = (iso: string, k: number) => {
        const v = parseISO(iso);
        if (!v) return iso;
        const d = addDays(v.date, k);
        return v.time ? `${isoDay(d)}T${iso.slice(11, 16)}` : isoDay(d);
      };
      const end = cur.end ?? cur.start;
      let next: DateValue;
      if (which === "move") next = { ...cur, start: shiftIso(cur.start, dd), end: shiftIso(end, dd) };
      else if (which === "end") {
        const ne = shiftIso(end, dd);
        next = { ...cur, end: ne < cur.start ? cur.start : ne };
      } else {
        const ns = shiftIso(cur.start, dd);
        next = { ...cur, start: ns > end ? end : ns, end };
      }
      host.engine.collections.setValue(ctx.colId, row.id, p.id, next);
      paint();
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  });

  const set = (change: Record<string, unknown>) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), ...change } }), 0);
  zoomBtn.addEventListener("click", () =>
    openMenu(zoomBtn, ZOOM_LABEL.map(([k, l]) => ({ label: t(l), checked: zoom() === k, run: () => {
      scrolledToday = false;
      set({ zoom: k });
    } })), { side: "bottom", align: "end", label: t("Zoom") }),
  );
  tableBtn.addEventListener("click", () => set({ table: !showTable() }));
  todayBtn.addEventListener("click", () => {
    scroll.scrollLeft = Math.max(0, xOf(startOfDay(new Date())) - scroll.clientWidth / 3);
  });

  return {
    el,
    ready,
    ctx,
    configChanged(prev, nextView) {
      if (JSON.stringify(prev.config.filter) !== JSON.stringify(nextView.config.filter) || JSON.stringify(prev.config.sorts) !== JSON.stringify(nextView.config.sorts)) void session.load().catch(host.engine.reportError);
      else paint();
    },
    schemaChanged: paint,
    setSearch(q) {
      session.options.search = q;
      void session.load().catch(host.engine.reportError);
    },
    addNew() {
      create(startOfDay(new Date()));
    },
    settings(anchor) {
      const lay = layout();
      const entries: MenuEntry[] = [{ kind: "heading", label: t("Show timeline by") }, ...dateProps(host).map((p) => ({ label: p.name, checked: dateProp()?.id === p.id, run: () => set({ date: p.id }) }))];
      entries.push({ kind: "heading", label: t("End date") }, { label: t("Use the same property"), checked: !lay.end, run: () => set({ end: "" }) });
      for (const p of dateProps(host)) if (p.id !== dateProp()?.id) entries.push({ label: p.name, checked: lay.end === p.id, run: () => set({ end: p.id }) });
      const rels = ctx.col().schema.filter((p) => p.type === "relation" && (p.collection === ctx.colId || !p.collection));
      entries.push({ kind: "heading", label: t("Dependencies") }, { label: t("None"), checked: !lay.deps, run: () => set({ deps: "" }) });
      for (const p of rels) entries.push({ label: t("Blocked by {name}", { name: p.name }), checked: lay.deps === p.id, run: () => set({ deps: p.id }) });
      openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Layout") });
    },
    destroy() {
      destroyed = true;
      sized.disconnect();
      session.close();
      el.remove();
    },
  };
}
