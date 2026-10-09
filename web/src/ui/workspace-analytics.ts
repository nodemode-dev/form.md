import type { Engine, TopPage, WorkspaceAnalytics } from "../engine";
import { plural, t } from "../engine/i18n";
import { kpi } from "./analytics-dialog";
import { barChart } from "./bar-chart";
import type { Tab } from "./dialog";
import { h, setText } from "./dom";
import { choices, section, status, toggle } from "./form";
import { reconcile } from "./list";
import { fallbackGlyph, pageIcon, paintIcon } from "./pageicon";
import { boundTab, isAdmin } from "./workspace-settings";

export function analyticsTab(engine: Engine): Tab {
  return boundTab(
    engine,
    "analytics",
    t("Analytics"),
    "chart",
    (panel, w) => {
      const st = status();
      const history = toggle(t("Record who viewed each page"), true, t("Page analytics list the people who opened a page. Members can also opt out for themselves in their account settings."));
      const period = choices<"7" | "28" | "90">(`ws-an-${w.id}`, [
        ["7", t("7 days")],
        ["28", t("28 days")],
        ["90", t("90 days")],
      ], "28");
      const views = kpi(t("Views"));
      const viewers = kpi(t("Unique viewers"));
      const pages = kpi(t("Pages viewed"));
      const chart = barChart(t("Views per day"));
      const list = h("div", { class: "rows" });
      const empty = h("p", { class: "muted", text: t("No page views in this period.") });
      empty.hidden = true;
      const set = section(t("View history"));
      set.append(history.el);
      const activity = section(t("Activity"));
      activity.append(period.el, h("div", { class: "kpis" }, views.el, viewers.el, pages.el), h("div", { class: "chart-box" }, chart.el), st.el);
      const topSec = section(t("Most viewed pages"));
      topSec.append(list, empty);
      panel.append(set, activity, topSec);

      const row = (p: TopPage) => {
        const r = h("button", { class: "row row--button", attrs: { type: "button" } }, pageIcon(p.icon, fallbackGlyph(p.kind), "row__icon"), h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })));
        r.addEventListener("click", () => {
          engine.state.set("overlay", null);
          engine.openPage(p.id);
        });
        return r;
      };
      const update = (r: HTMLElement, p: TopPage) => {
        paintIcon(r.firstElementChild as HTMLElement, p.icon, fallbackGlyph(p.kind));
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, p.title || t("Untitled"));
        setText(meta, [plural(p.views, "{count} view", "{count} views"), plural(p.viewers, "{count} person", "{count} people")].join(" · "));
      };
      history.el.dataset.loading = "";
      activity.dataset.loading = "";
      const paint = (r: WorkspaceAnalytics) => {
        history.input.checked = r.history;
        delete history.el.dataset.loading;
        delete activity.dataset.loading;
        views.set(r.views);
        viewers.set(r.viewers);
        pages.set(r.pages);
        chart.update(r.series);
        reconcile(list, r.top, (p) => p.id, row, update);
        empty.hidden = r.top.length > 0;
      };
      let seq = 0;
      const load = () => {
        const mine = ++seq;
        engine.api.analytics
          .workspace(w.id, Number(period.value()))
          .then((r) => {
            if (mine === seq) paint(r);
          })
          .catch((e) => st.error(e));
      };
      period.el.addEventListener("change", load);
      history.input.addEventListener("change", async () => {
        const on = history.input.checked;
        try {
          await engine.api.analytics.setHistory(w.id, on);
          st.say(on ? t("View history is on.") : t("View history is off. Views are still counted, without names."), "success");
        } catch (e) {
          history.input.checked = !on;
          st.error(e);
        }
      });
      load();
      return load;
    },
    () => {
      const w = engine.workspace();
      return !!w && isAdmin(w);
    },
  );
}
