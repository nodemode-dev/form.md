import type { AnalyticsPerson, Engine, PageAnalytics } from "../engine";
import { formatDate, formatDateTime, formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { avatar, updateAvatar } from "./avatar";
import { barChart } from "./bar-chart";
import { simpleDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { errorMessage } from "./errors";
import { choices, section } from "./form";
import { reconcile } from "./list";
import { toast } from "./toast";

export function kpi(label: string): { el: HTMLElement; set(v: number): void } {
  const value = h("span", { class: "kpi__value", text: "0" });
  return { el: h("div", { class: "kpi" }, value, h("span", { class: "kpi__label", text: label })), set: (v) => setText(value, v.toLocaleString()) };
}

function personRow(): HTMLElement {
  return h("div", { class: "row" }, avatar("", ""), h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })));
}

function paintPerson(el: HTMLElement, p: AnalyticsPerson, meta: string): void {
  updateAvatar(el.firstElementChild as HTMLElement, p.name, "");
  const [title, sub] = el.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
  setText(title, p.name || p.email || t("Someone"));
  setText(sub, meta);
  setAttr(sub, "title", formatDateTime(p.at));
}

export async function openAnalytics(engine: Engine, pageId: string): Promise<void> {
  let rep: PageAnalytics;
  try {
    rep = await engine.api.analytics.page(pageId, 28);
  } catch (e) {
    toast({ tone: "danger", message: errorMessage(e) });
    return;
  }
  simpleDialog(
    t("Page analytics"),
    (body) => {
      const period = choices<"7" | "28" | "90">(`an-period-${pageId}`, [
        ["7", t("7 days")],
        ["28", t("28 days")],
        ["90", t("90 days")],
      ], "28");
      const views = kpi(t("Views"));
      const viewers = kpi(t("Unique viewers"));
      const anon = kpi(t("Views from the web"));
      const chart = barChart(t("Views per day"));
      const origin = h("p", { class: "an-origin muted" });
      const edited = h("p", { class: "an-origin muted" });
      const editors = h("div", { class: "rows" });
      const people = h("div", { class: "rows" });
      const noPeople = h("p", { class: "muted" });
      const top = h("div", { class: "an-head" }, period.el);
      const kpis = h("div", { class: "kpis" }, views.el, viewers.el, anon.el);
      const editSec = section(t("Edited by"));
      editSec.append(editors);
      const viewSec = section(t("Viewed by"));
      viewSec.append(people, noPeople);
      body.append(top, kpis, h("div", { class: "chart-box" }, chart.el), origin, edited, editSec, viewSec);

      const paint = (r: PageAnalytics) => {
        views.set(r.views);
        viewers.set(r.viewers);
        anon.set(r.anonymous);
        anon.el.hidden = r.anonymous === 0;
        chart.update(r.series);
        setText(origin, r.created_by ? t("Created by {name} on {date}", { name: r.created_by.name, date: formatDate(r.created_at) }) : t("Created on {date}", { date: formatDate(r.created_at) }));
        setText(edited, r.updated_by ? t("Last edited by {name} {when}", { name: r.updated_by.name, when: formatRelative(r.updated_at) }) : t("Last edited {when}", { when: formatRelative(r.updated_at) }));
        reconcile(editors, r.editors, (p) => p.id, personRow, (el, p) => paintPerson(el, p, formatRelative(p.at)));
        editSec.hidden = r.editors.length === 0;
        reconcile(people, r.people, (p) => p.id, personRow, (el, p) => paintPerson(el, p, [formatRelative(p.at), plural(p.views ?? 0, "{count} view", "{count} views")].join(" · ")));
        noPeople.hidden = r.people.length > 0;
        setText(noPeople, r.history ? t("Nobody has opened it in this period.") : t("View history is turned off for this workspace."));
      };
      paint(rep);
      let seq = 0;
      period.el.addEventListener("change", async () => {
        const mine = ++seq;
        try {
          const r = await engine.api.analytics.page(pageId, Number(period.value()));
          if (mine === seq) paint(r);
        } catch (e) {
          toast({ tone: "danger", message: errorMessage(e) });
        }
      });
      return period.el.querySelector("input") ?? undefined;
    },
    undefined,
    "md",
  );
}
