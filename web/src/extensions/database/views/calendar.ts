import { locale, weekStart } from "../../../engine/format";
import { t } from "../../../engine/i18n";
import type { CollectionView, Property, RowData } from "../../../engine/types";
import { byId, h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import { pageIcon } from "../../../ui/pageicon";
import { editCell } from "../editors";
import { computed, type DateValue, isoDay, parseISO } from "../props";
import type { ViewHost, ViewRenderer } from "../surface";
import { saveConfig } from "../toolbar";
import { prefillFromFilter } from "./shared";

const DAY = 86400000;

function addDays(d: Date, n: number): Date {
  const x = new Date(d);
  x.setDate(x.getDate() + n);
  return x;
}

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function diffDays(a: Date, b: Date): number {
  return Math.round((startOfDay(a).getTime() - startOfDay(b).getTime()) / DAY);
}

export function dateProps(host: ViewHost): Property[] {
  return host.col().schema.filter((p) => p.type === "date" || p.type === "created_time" || p.type === "last_edited_time" || (p.type === "formula" && /date|now|today/i.test(p.expr ?? "")));
}

export interface Span {
  row: RowData;
  start: Date;
  end: Date;
}

export function spanOf(row: RowData, p: Property): Span | null {
  const v = row.values[p.id];
  if (p.type === "created_time" || p.type === "last_edited_time") {
    if (typeof v !== "number") return null;
    const d = startOfDay(new Date(v));
    return { row, start: d, end: d };
  }
  let dv: DateValue | null = null;
  if (p.type === "formula" || p.type === "rollup") {
    const cv = v as { type?: string; value?: { start: string; end?: string } } | null;
    if (cv?.type === "date" && cv.value) dv = { start: cv.value.start, ...(cv.value.end ? { end: cv.value.end } : {}) };
  } else dv = (v as DateValue | undefined) ?? null;
  if (!dv?.start) return null;
  const s = parseISO(dv.start);
  if (!s) return null;
  const e = dv.end ? parseISO(dv.end) : null;
  const start = startOfDay(s.date);
  const end = e ? startOfDay(e.date) : start;
  return { row, start, end: end < start ? start : end };
}

export function calendarView(host: ViewHost): ViewRenderer {
  const view = (): CollectionView => host.view();
  const layout = () => view().config.layout ?? {};
  const dateProp = (): Property | undefined => {
    const want = layout().date as string | undefined;
    const list = dateProps(host);
    return list.find((p) => p.id === want) ?? list.find((p) => p.type === "date") ?? list[0];
  };
  let cursor = startOfDay(new Date());
  const mode = () => (layout().mode === "week" ? "week" : "month");
  const weekends = () => layout().weekends !== false;

  const range = (): { from: Date; to: Date; weeks: Date[] } => {
    const ws = weekStart();
    if (mode() === "week") {
      const s = addDays(cursor, -((cursor.getDay() - ws + 7) % 7));
      return { from: s, to: addDays(s, 6), weeks: [s] };
    }
    const first = new Date(cursor.getFullYear(), cursor.getMonth(), 1);
    const s = addDays(first, -((first.getDay() - ws + 7) % 7));
    const last = new Date(cursor.getFullYear(), cursor.getMonth() + 1, 0);
    const weeks: Date[] = [];
    for (let w = s; w <= last; w = addDays(w, 7)) weeks.push(w);
    return { from: s, to: addDays(weeks.at(-1) ?? s, 6), weeks };
  };

  const rangeInput = () => {
    const p = dateProp();
    const r = range();
    return p ? { prop: p.id, from: isoDay(r.from), to: isoDay(r.to) } : undefined;
  };
  const r0 = rangeInput();
  const session = host.makeSession({ limit: 1000, noGroups: true, search: host.search(), ...(r0 ? { range: r0 } : {}) });
  const ctx = host.ctx(session);
  const el = h("div", { class: "cal" });
  const title = h("span", { class: "cal__title" });
  const prev = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Previous") } }, iconNode("chevron-left"));
  const next = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Next") } }, iconNode("chevron-right"));
  const today = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Today") });
  const modeBtn = h("button", { class: "btn btn--sm", attrs: { type: "button" } });
  const head = h("div", { class: "cal__head" }, title, h("span", { class: "cal__spacer" }), modeBtn, today, prev, next);
  const dows = h("div", { class: "cal__dows" });
  const grid = h("div", { class: "cal__grid" });
  const empty = h("div", { class: "board__empty" });
  el.append(empty, head, dows, grid);
  let destroyed = false;
  let pendingTitle = "";

  const dayCols = (): number[] => {
    const ws = weekStart();
    const cols: number[] = [];
    for (let i = 0; i < 7; i++) {
      const dow = (ws + i) % 7;
      if (weekends() || (dow !== 0 && dow !== 6)) cols.push(i);
    }
    return cols;
  };

  const paint = () => {
    if (destroyed) return;
    const p = dateProp();
    empty.hidden = !!p;
    head.hidden = !p;
    dows.hidden = !p;
    grid.hidden = !p;
    if (!p) {
      empty.replaceChildren(h("p", { text: t("A calendar places pages by a date. Add a date property to use it.") }));
      return;
    }
    const r = range();
    const fmt = new Intl.DateTimeFormat(locale(), mode() === "week" ? { month: "long", day: "numeric", year: "numeric" } : { month: "long", year: "numeric" });
    setText(title, mode() === "week" ? `${fmt.format(r.from)} – ${fmt.format(r.to)}`.replace("–", "-") : fmt.format(cursor));
    setText(modeBtn, mode() === "week" ? t("Week") : t("Month"));
    const cols = dayCols();
    const dowFmt = new Intl.DateTimeFormat(locale(), { weekday: "short" });
    el.style.setProperty("--cal-cols", String(cols.length));
    if (dows.dataset.sig !== cols.join(",")) {
      dows.dataset.sig = cols.join(",");
      dows.replaceChildren(...cols.map((i) => h("span", { class: "cal__dow", text: dowFmt.format(addDays(r.from, i)) })));
    }
    const spans: Span[] = [];
    for (const id of session.ordered()) {
      const row = session.rows.get(id);
      const s = row ? spanOf(row, p) : null;
      if (s) spans.push(s);
    }
    spans.sort((a, b) => a.start.getTime() - b.start.getTime() || b.end.getTime() - a.end.getTime());
    const now = startOfDay(new Date());
    const maxLanes = mode() === "week" ? 50 : 4;
    type Bar = { key: string; row?: RowData; col: string; lane: number; contStart: boolean; contEnd: boolean; day?: string; n?: number };
    const editable = host.canWrite() && !computed(p.type);
    reconcile(
      grid,
      r.weeks,
      (ws) => isoDay(ws),
      () => h("div", { class: "cal__week" }, h("div", { class: "cal__days" }), h("div", { class: "cal__bars" })),
      (week, ws) => {
        const days = week.firstElementChild as HTMLElement;
        const barsEl = week.lastElementChild as HTMLElement;
        reconcile(
          days,
          cols.map((i) => addDays(ws, i)),
          (d) => isoDay(d),
          (d) => {
            const cell = h("div", { class: "cal__day", attrs: { "data-day": isoDay(d) } }, h("span", { class: "cal__num", text: String(d.getDate()) }));
            cell.append(h("button", { class: "icon-btn icon-btn--sm cal__add", attrs: { type: "button", "aria-label": t("New page on this day") } }, iconNode("plus")));
            return cell;
          },
          (cell, d) => {
            (cell.lastElementChild as HTMLElement).hidden = !host.canWrite();
            setAttr(cell, "data-out", mode() === "month" && d.getMonth() !== cursor.getMonth() ? "true" : null);
            setAttr(cell, "data-today", d.getTime() === now.getTime() ? "true" : null);
          },
        );
        const lanes: Date[] = [];
        const overflow = new Map<number, number>();
        const weekEnd = addDays(ws, 6);
        const bars: Bar[] = [];
        for (const s of spans) {
          if (s.end < ws || s.start > weekEnd) continue;
          const a = s.start < ws ? ws : s.start;
          const b = s.end > weekEnd ? weekEnd : s.end;
          const ai = cols.findIndex((c) => c >= diffDays(a, ws));
          let bi = -1;
          for (let k = cols.length - 1; k >= 0; k--)
            if ((cols[k] ?? 0) <= diffDays(b, ws)) {
              bi = k;
              break;
            }
          if (ai < 0 || bi < ai) continue;
          let lane = lanes.findIndex((endAt) => endAt < a);
          if (lane < 0) {
            lane = lanes.length;
            lanes.push(b);
          } else lanes[lane] = b;
          if (lane >= maxLanes) {
            for (let k = ai; k <= bi; k++) overflow.set(k, (overflow.get(k) ?? 0) + 1);
            continue;
          }
          bars.push({ key: s.row.id, row: s.row, col: `${ai + 1} / ${bi + 2}`, lane, contStart: s.start < ws, contEnd: s.end > weekEnd });
        }
        for (const [k, n] of overflow) bars.push({ key: `more:${k}`, col: `${k + 1} / ${k + 2}`, lane: maxLanes, contStart: false, contEnd: false, day: isoDay(addDays(ws, cols[k] ?? 0)), n });
        reconcile(
          barsEl,
          bars,
          (b) => b.key,
          (b) => {
            if (!b.row) return h("button", { class: "cal__more", attrs: { type: "button" } });
            const bar = h("div", { class: "cal__bar", attrs: { "data-row": b.row.id, tabindex: "0" } }, h("span", { class: "cal__icon" }), h("span", { class: "cal__bartitle" }), h("span", { class: "cal__resize", attrs: { "aria-hidden": "true" } }));
            return bar;
          },
          (node, b) => {
            node.style.gridColumn = b.col;
            node.style.gridRow = String(b.lane + 1);
            if (!b.row) {
              node.dataset.day = b.day ?? "";
              setText(node, t("{n} more", { n: b.n ?? 0 }));
              return;
            }
            const icon = node.firstElementChild as HTMLElement;
            if (icon.dataset.icon !== b.row.icon) {
              icon.dataset.icon = b.row.icon;
              icon.replaceChildren(pageIcon(b.row.icon, "page"));
            }
            setText(node.children[1] as HTMLElement, b.row.title || t("Untitled"));
            (node.lastElementChild as HTMLElement).hidden = !editable;
            setAttr(node, "data-cont-start", b.contStart ? "true" : null);
            setAttr(node, "data-cont-end", b.contEnd ? "true" : null);
          },
        );
      },
    );
    setAttr(el, "data-mode", mode());
    if (pendingTitle) {
      const bar = grid.querySelector<HTMLElement>(`.cal__bar[data-row="${pendingTitle}"]`);
      const row = session.rows.get(pendingTitle);
      const tp = ctx.col().schema.find((x) => x.type === "title");
      if (bar && row && tp) {
        pendingTitle = "";
        editCell(bar, tp, row, ctx);
      }
    }
  };

  session.on((c) => {
    if (c.kind !== "loading") paint();
  });
  const ready = dateProp() ? session.load() : Promise.resolve();
  paint();

  const reload = () => {
    const ri = rangeInput();
    if (ri) session.options.range = ri;
    else delete session.options.range;
    paint();
    if (ri) void session.load().catch(host.engine.reportError);
  };
  const shift = (n: number) => {
    cursor = mode() === "week" ? addDays(cursor, 7 * n) : new Date(cursor.getFullYear(), cursor.getMonth() + n, 1);
    reload();
  };
  prev.addEventListener("click", () => shift(-1));
  next.addEventListener("click", () => shift(1));
  today.addEventListener("click", () => {
    cursor = startOfDay(new Date());
    reload();
  });
  modeBtn.addEventListener("click", () => {
    const set = (m: string) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), mode: m } }), 0);
    openMenu(modeBtn, [
      { label: t("Month"), checked: mode() === "month", run: () => set("month") },
      { label: t("Week"), checked: mode() === "week", run: () => set("week") },
    ], { side: "bottom", align: "end", label: t("Calendar") });
  });

  const createOn = (day: string) => {
    const p = dateProp();
    if (!p || computed(p.type)) return;
    const values = prefillFromFilter(ctx, view(), { [p.id]: { start: day } });
    void host.engine.collections
      .createRow(ctx.colId, { values })
      .then((row) => {
        pendingTitle = row.id;
      })
      .catch(host.engine.reportError);
  };

  let suppress = false;
  grid.addEventListener("click", (e) => {
    if (suppress) return;
    const target = e.target as HTMLElement;
    const add = target.closest<HTMLElement>(".cal__add");
    if (add) {
      const day = add.closest<HTMLElement>(".cal__day")?.dataset.day;
      if (day) createOn(day);
      return;
    }
    const more = target.closest<HTMLElement>(".cal__more");
    if (more?.dataset.day) {
      cursor = parseISO(more.dataset.day)?.date ?? cursor;
      saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), mode: "week" } }), 0);
      return;
    }
    const bar = target.closest<HTMLElement>(".cal__bar");
    if (bar?.dataset.row) host.openRow(bar.dataset.row, e);
  });
  grid.addEventListener("dblclick", (e) => {
    const day = (e.target as Element).closest<HTMLElement>(".cal__day")?.dataset.day;
    if (day && host.canWrite()) createOn(day);
  });

  grid.addEventListener("pointerdown", (e) => {
    const bar = (e.target as Element).closest<HTMLElement>(".cal__bar");
    const p = dateProp();
    if (!bar?.dataset.row || !p || computed(p.type) || !host.canWrite() || e.button !== 0) return;
    const resizing = !!(e.target as Element).closest(".cal__resize");
    const row = session.rows.get(bar.dataset.row);
    const span = row ? spanOf(row, p) : null;
    if (!row || !span) return;
    const dayAt = (x: number, y: number) => (document.elementsFromPoint(x, y).find((n) => n.classList.contains("cal__day")) as HTMLElement | undefined)?.dataset.day;
    const fromDay = dayAt(e.clientX, e.clientY);
    if (!fromDay) return;
    let ghost: HTMLElement | null = null;
    let target = fromDay;
    const sx = e.clientX;
    const sy = e.clientY;
    const move = (ev: PointerEvent) => {
      if (!ghost && Math.hypot(ev.clientX - sx, ev.clientY - sy) < 5) return;
      if (!ghost) {
        ghost = bar.cloneNode(true) as HTMLElement;
        ghost.classList.add("cal__bar--ghost");
        ghost.style.width = `${bar.getBoundingClientRect().width}px`;
        byId("overlay-root").append(ghost);
        bar.classList.add("is-dragging");
      }
      ghost.style.transform = `translate(${ev.clientX + 6}px, ${ev.clientY + 6}px)`;
      const d = dayAt(ev.clientX, ev.clientY);
      if (d) {
        target = d;
        for (const c of Array.from(grid.querySelectorAll("[data-drop]"))) c.removeAttribute("data-drop");
        grid.querySelector(`.cal__day[data-day="${d}"]`)?.setAttribute("data-drop", "true");
      }
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      for (const c of Array.from(grid.querySelectorAll("[data-drop]"))) c.removeAttribute("data-drop");
      if (!ghost) return;
      ghost.remove();
      bar.classList.remove("is-dragging");
      suppress = true;
      setTimeout(() => (suppress = false), 0);
      const to = parseISO(target)?.date;
      const from = parseISO(fromDay)?.date;
      if (!to || !from) return;
      const delta = diffDays(to, from);
      if (!delta) return;
      const cur = (row.values[p.id] as DateValue | undefined) ?? { start: isoDay(span.start) };
      const shiftIso = (iso: string, n: number) => {
        const v = parseISO(iso);
        if (!v) return iso;
        const d = addDays(v.date, n);
        return v.time ? `${isoDay(d)}T${iso.slice(11, 16)}` : isoDay(d);
      };
      let next: DateValue;
      if (resizing) {
        const end = shiftIso(cur.end ?? cur.start, delta);
        next = { ...cur, end: end < cur.start ? cur.start : end };
      } else next = { ...cur, start: shiftIso(cur.start, delta), ...(cur.end ? { end: shiftIso(cur.end, delta) } : {}) };
      host.engine.collections.setValue(ctx.colId, row.id, p.id, next);
      session.scheduleRefresh([row.id]);
      paint();
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  });

  return {
    el,
    ready,
    ctx,
    configChanged(prev, nextView) {
      const pl = prev.config.layout ?? {};
      const nl = nextView.config.layout ?? {};
      if (JSON.stringify(prev.config.filter) !== JSON.stringify(nextView.config.filter) || pl.date !== nl.date || pl.mode !== nl.mode) reload();
      else paint();
    },
    schemaChanged: reload,
    setSearch(q) {
      session.options.search = q;
      void session.load().catch(host.engine.reportError);
    },
    addNew() {
      createOn(isoDay(new Date()));
    },
    settings(anchor) {
      const set = (change: Record<string, unknown>) => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), ...change } }), 0);
      const entries: MenuEntry[] = [{ kind: "heading", label: t("Show calendar by") }, ...dateProps(host).map((p) => ({ label: p.name, checked: dateProp()?.id === p.id, run: () => set({ date: p.id }) }))];
      entries.push({ kind: "separator" }, { label: t("Show weekends"), checked: weekends(), run: () => set({ weekends: !weekends() }) });
      openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Layout") });
    },
    destroy() {
      destroyed = true;
      session.close();
      el.remove();
    },
  };
}
