import { t } from "../../engine/i18n";
import type { Collection, DateRangeInput, Property } from "../../engine/types";
import { h } from "../../ui/dom";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { openPanel } from "./panel";

const KEY = "formmd.daterange.";
const DAY = /^\d{4}-\d{2}-\d{2}$/;

export const RANGE_PRESETS: [string, string][] = [
  ["today", "Today"],
  ["past_7_days", "Last 7 days"],
  ["past_30_days", "Last 30 days"],
];

export function dateProps(col: Collection | undefined): Property[] {
  return (col?.schema ?? []).filter((p) => p.type === "date" || p.type === "created_time" || p.type === "last_edited_time");
}

function defaultProp(col: Collection | undefined): string {
  const list = dateProps(col);
  return (list.find((p) => p.type === "date") ?? list[0])?.id ?? "";
}

function valid(r: DateRangeInput, col: Collection | undefined): boolean {
  if (!dateProps(col).some((p) => p.id === r.prop)) return false;
  if (r.within) return RANGE_PRESETS.some(([k]) => k === r.within) && !r.from && !r.to;
  return !!r.from && !!r.to && DAY.test(r.from) && DAY.test(r.to) && r.from <= r.to;
}

export function readRange(viewId: string, col: Collection | undefined): DateRangeInput | undefined {
  try {
    const raw = localStorage.getItem(KEY + viewId);
    if (!raw) return undefined;
    const r = JSON.parse(raw) as DateRangeInput;
    return r && typeof r === "object" && valid(r, col) ? r : undefined;
  } catch {
    return undefined;
  }
}

export function writeRange(viewId: string, r: DateRangeInput | undefined): void {
  try {
    if (r) localStorage.setItem(KEY + viewId, JSON.stringify(r));
    else localStorage.removeItem(KEY + viewId);
  } catch {
    return;
  }
}

export function rangeLabel(r: DateRangeInput | undefined): string {
  if (!r) return t("Date range");
  const preset = RANGE_PRESETS.find(([k]) => k === r.within);
  if (preset) return t(preset[1]);
  return r.from === r.to ? (r.from ?? "") : `${r.from} – ${r.to}`;
}

function openCustom(anchor: HTMLElement, r: DateRangeInput | undefined, prop: string, apply: (next: DateRangeInput) => void): void {
  const iso = (back: number) => {
    const d = new Date();
    d.setDate(d.getDate() - back);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  };
  const back = r?.within === "past_30_days" ? 29 : r?.within === "past_7_days" ? 6 : 0;
  openPanel(
    anchor,
    (panel, close) => {
      const from = h("input", { class: "input input--sm", attrs: { type: "date", "aria-label": t("Start date"), required: "" } });
      const to = h("input", { class: "input input--sm", attrs: { type: "date", "aria-label": t("End date"), required: "" } });
      from.value = r?.from ?? iso(back);
      to.value = r?.to ?? iso(0);
      const ok = h("button", { class: "btn btn--primary btn--sm", attrs: { type: "submit" }, text: t("Apply") });
      const check = () => {
        const good = DAY.test(from.value) && DAY.test(to.value) && from.value <= to.value;
        ok.disabled = !good;
        to.setAttribute("aria-invalid", good || !to.value ? "false" : "true");
        return good;
      };
      from.addEventListener("input", check);
      to.addEventListener("input", check);
      check();
      const form = h(
        "form",
        { class: "dbpanel__pad dbpanel__stack daterange" },
        h("label", { class: "daterange__field" }, h("span", { text: t("Start date") }), from),
        h("label", { class: "daterange__field" }, h("span", { text: t("End date") }), to),
        h("div", { class: "dbpanel__row dbpanel__row--end" }, ok),
      );
      form.addEventListener("submit", (e) => {
        e.preventDefault();
        if (!check()) return;
        apply({ prop, from: from.value, to: to.value });
        close();
      });
      panel.append(form);
    },
    { label: t("Custom range"), side: "bottom", align: "end" },
  );
}

export function openRangeMenu(anchor: HTMLElement, col: Collection | undefined, r: DateRangeInput | undefined, apply: (next: DateRangeInput | undefined) => void): void {
  const prop = r?.prop ?? defaultProp(col);
  const entries: MenuEntry[] = [
    { label: t("All dates"), checked: !r, run: () => apply(undefined) },
    ...RANGE_PRESETS.map(([k, l]): MenuEntry => ({ label: t(l), checked: r?.within === k, run: () => apply({ prop, within: k }) })),
    { label: t("Custom range"), icon: "calendar", checked: !!r?.from, run: () => openCustom(anchor, r, prop, apply) },
  ];
  const props = dateProps(col);
  if (r && props.length > 1) {
    entries.push({ kind: "heading", label: t("Date property") });
    for (const p of props) entries.push({ label: p.name, checked: p.id === prop, run: () => apply({ ...r, prop: p.id }) });
  }
  openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Date range") });
}
