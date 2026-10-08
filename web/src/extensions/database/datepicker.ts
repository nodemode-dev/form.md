import { locale, weekStart } from "../../engine/format";
import { t } from "../../engine/i18n";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { type DateValue, isoDay, parseISO } from "./props";

export interface DatePickOptions {
  value: DateValue | null;
  allowEnd?: boolean;
  allowTime?: boolean;
  allowRemind?: boolean;
  onChange: (v: DateValue | null) => void;
}

function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

function timeOf(iso: string | undefined): string {
  const m = iso ? /T(\d{2}:\d{2})/.exec(iso) : null;
  return m?.[1] ?? "09:00";
}

function parseTyped(s: string): string | null {
  const v = s.trim();
  if (!v) return null;
  if (/^\d{4}-\d{2}-\d{2}$/.test(v)) return v;
  const d = new Date(v);
  if (!Number.isNaN(d.getTime()) && d.getFullYear() > 1000) return isoDay(d);
  return null;
}

export function datePicker(o: DatePickOptions): HTMLElement {
  let value: DateValue | null = o.value?.start ? { ...o.value } : null;
  const startParsed = value ? parseISO(value.start) : null;
  let month = startParsed ? new Date(startParsed.date.getFullYear(), startParsed.date.getMonth(), 1) : new Date(new Date().getFullYear(), new Date().getMonth(), 1);
  let hasEnd = !!value?.end;
  let hasTime = !!startParsed?.time;
  let picking: "start" | "end" = "start";

  const root = h("div", { class: "dpick" });
  const startIn = h("input", { class: "input input--sm dpick__in", attrs: { type: "text", "aria-label": t("Start date"), placeholder: t("Date") } });
  const endIn = h("input", { class: "input input--sm dpick__in", attrs: { type: "text", "aria-label": t("End date"), placeholder: t("End date") } });
  const startTime = h("input", { class: "input input--sm dpick__time", attrs: { type: "time", "aria-label": t("Start time") } });
  const endTime = h("input", { class: "input input--sm dpick__time", attrs: { type: "time", "aria-label": t("End time") } });
  const inputs = h("div", { class: "dpick__inputs" }, h("div", { class: "dpick__pair" }, startIn, startTime), h("div", { class: "dpick__pair", attrs: { "data-ref": "end" } }, endIn, endTime));
  const title = h("span", { class: "dpick__month" });
  const prev = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Previous month") } }, iconNode("chevron-left"));
  const next = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Next month") } }, iconNode("chevron-right"));
  const today = h("button", { class: "link-btn dpick__today", attrs: { type: "button" }, text: t("Today") });
  const head = h("div", { class: "dpick__head" }, title, today, prev, next);
  const grid = h("div", { class: "dpick__grid", attrs: { role: "grid" } });
  const endToggle = h("label", { class: "dpick__toggle" }, h("span", { text: t("End date") }), h("input", { attrs: { type: "checkbox", role: "switch" } }));
  const timeToggle = h("label", { class: "dpick__toggle" }, h("span", { text: t("Include time") }), h("input", { attrs: { type: "checkbox", role: "switch" } }));
  const remindSel = h("select", { class: "input input--sm", attrs: { "aria-label": t("Remind") } });
  for (const [v, label] of [["", t("None")], ["0", t("On the day")], ["1", t("1 day before")], ["2", t("2 days before")], ["7", t("1 week before")]] as const) {
    remindSel.append(h("option", { attrs: { value: v }, text: label }));
  }
  const remindRow = h("label", { class: "dpick__toggle" }, h("span", { text: t("Remind") }), remindSel);
  const clear = h("button", { class: "menu__item dpick__clear", attrs: { type: "button" } }, iconNode("close"), h("span", { class: "menu__label", text: t("Clear") }));
  const foot = h("div", { class: "dpick__foot" });
  if (o.allowEnd !== false) foot.append(endToggle);
  if (o.allowTime !== false) foot.append(timeToggle);
  if (o.allowRemind) foot.append(remindRow);
  foot.append(clear);
  root.append(inputs, head, grid, foot);

  const endCheck = endToggle.querySelector("input") as HTMLInputElement;
  const timeCheck = timeToggle.querySelector("input") as HTMLInputElement;

  const commit = () => o.onChange(value ? { ...value } : null);
  const compose = (day: string, time: string) => (hasTime ? `${day}T${time}` : day);
  const dayOf = (iso: string | undefined) => (iso ? iso.slice(0, 10) : "");

  const remindDays = (): string => {
    if (!value?.remind) return "";
    const s = parseISO(value.start);
    const r = parseISO(value.remind);
    if (!s || !r) return "";
    return String(Math.round((new Date(s.date.getFullYear(), s.date.getMonth(), s.date.getDate()).getTime() - new Date(r.date.getFullYear(), r.date.getMonth(), r.date.getDate()).getTime()) / 86400000));
  };

  const paint = () => {
    const fmt = new Intl.DateTimeFormat(locale(), { month: "long", year: "numeric" });
    setText(title, fmt.format(month));
    if (document.activeElement !== startIn) startIn.value = value ? dayOf(value.start) : "";
    if (document.activeElement !== endIn) endIn.value = value?.end ? dayOf(value.end) : "";
    startTime.value = timeOf(value?.start);
    endTime.value = timeOf(value?.end);
    startTime.hidden = !hasTime;
    endTime.hidden = !hasTime;
    (inputs.lastElementChild as HTMLElement).hidden = !hasEnd;
    endCheck.checked = hasEnd;
    timeCheck.checked = hasTime;
    remindSel.value = remindDays();
    setAttr(startIn, "data-active", hasEnd && picking === "start" ? "true" : null);
    setAttr(endIn, "data-active", hasEnd && picking === "end" ? "true" : null);
    const ws = weekStart();
    const first = new Date(month);
    const offset = (first.getDay() - ws + 7) % 7;
    first.setDate(1 - offset);
    const s = value ? parseISO(value.start)?.date : undefined;
    const e = value?.end ? parseISO(value.end)?.date : undefined;
    const now = new Date();
    if (grid.childElementCount === 0) {
      const days = new Intl.DateTimeFormat(locale(), { weekday: "narrow" });
      for (let i = 0; i < 7; i++) grid.append(h("span", { class: "dpick__dow", text: days.format(new Date(2024, 0, 7 + ((ws + i) % 7))) }));
      for (let i = 0; i < 42; i++) grid.append(h("button", { class: "dpick__day", attrs: { type: "button", tabindex: "-1" } }));
    }
    const cells = Array.from(grid.querySelectorAll<HTMLButtonElement>(".dpick__day"));
    cells.forEach((b, i) => {
      const d = new Date(first);
      d.setDate(first.getDate() + i);
      b.dataset.day = isoDay(d);
      setText(b, String(d.getDate()));
      setAttr(b, "data-out", d.getMonth() !== month.getMonth() ? "true" : null);
      setAttr(b, "data-today", sameDay(d, now) ? "true" : null);
      const isStart = !!s && sameDay(d, s);
      const isEnd = !!e && sameDay(d, e);
      setAttr(b, "aria-selected", isStart || isEnd ? "true" : "false");
      setAttr(b, "data-between", s && e && d > s && d < e ? "true" : null);
    });
  };

  grid.addEventListener("click", (ev) => {
    const b = (ev.target as Element).closest<HTMLElement>(".dpick__day");
    if (!b?.dataset.day) return;
    const day = b.dataset.day;
    if (!value || !hasEnd || picking === "start") {
      const time = timeOf(value?.start);
      value = { ...(value ?? {}), start: compose(day, time) };
      if (hasEnd) {
        if (!value.end || dayOf(value.end) < day) value.end = compose(day, timeOf(value.end));
        picking = "end";
      }
    } else {
      value.end = compose(day, timeOf(value.end));
      if (dayOf(value.end) < dayOf(value.start)) [value.start, value.end] = [value.end, value.start];
      picking = "start";
    }
    paint();
    commit();
  });
  prev.addEventListener("click", () => {
    month = new Date(month.getFullYear(), month.getMonth() - 1, 1);
    paint();
  });
  next.addEventListener("click", () => {
    month = new Date(month.getFullYear(), month.getMonth() + 1, 1);
    paint();
  });
  today.addEventListener("click", () => {
    const n = new Date();
    month = new Date(n.getFullYear(), n.getMonth(), 1);
    value = { ...(value ?? {}), start: compose(isoDay(n), timeOf(value?.start)) };
    paint();
    commit();
  });
  const typed = (which: "start" | "end") => {
    const el = which === "start" ? startIn : endIn;
    const day = parseTyped(el.value);
    if (!day) {
      if (!el.value.trim() && which === "start") {
        value = null;
        paint();
        commit();
      }
      return;
    }
    if (which === "start") value = { ...(value ?? {}), start: compose(day, timeOf(value?.start)) };
    else if (value) value.end = compose(day, timeOf(value.end));
    const p = parseISO(day);
    if (p) month = new Date(p.date.getFullYear(), p.date.getMonth(), 1);
    paint();
    commit();
  };
  startIn.addEventListener("change", () => typed("start"));
  endIn.addEventListener("change", () => typed("end"));
  startIn.addEventListener("focus", () => {
    picking = "start";
    paint();
  });
  endIn.addEventListener("focus", () => {
    picking = "end";
    paint();
  });
  for (const [el, which] of [[startTime, "start"], [endTime, "end"]] as const) {
    el.addEventListener("change", () => {
      if (!value) return;
      if (which === "start") value.start = `${dayOf(value.start)}T${el.value || "09:00"}`;
      else if (value.end) value.end = `${dayOf(value.end)}T${el.value || "09:00"}`;
      commit();
    });
  }
  endCheck.addEventListener("change", () => {
    hasEnd = endCheck.checked;
    if (value) {
      if (hasEnd) value.end = value.end ?? value.start;
      else delete value.end;
      commit();
    }
    picking = hasEnd ? "end" : "start";
    paint();
  });
  timeCheck.addEventListener("change", () => {
    hasTime = timeCheck.checked;
    if (value) {
      value.start = hasTime ? `${dayOf(value.start)}T${timeOf(value.start)}` : dayOf(value.start);
      if (value.end) value.end = hasTime ? `${dayOf(value.end)}T${timeOf(value.end)}` : dayOf(value.end);
      commit();
    }
    paint();
  });
  remindSel.addEventListener("change", () => {
    if (!value) return;
    const n = remindSel.value;
    if (n === "") delete value.remind;
    else {
      const s = parseISO(value.start);
      if (s) {
        const r = new Date(s.date);
        r.setDate(r.getDate() - Number(n));
        value.remind = `${isoDay(r)}T${s.time ? timeOf(value.start) : "09:00"}`;
      }
    }
    commit();
  });
  clear.addEventListener("click", () => {
    value = null;
    hasEnd = false;
    paint();
    commit();
  });
  paint();
  return root;
}
