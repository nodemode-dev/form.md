import type { Engine, Site } from "../engine";
import { formatDate } from "../engine/format";
import { t } from "../engine/i18n";
import type { Tab } from "./dialog";
import { confirmDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { busy, button, iconButton, section, status, toggle } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { siteUrl } from "./publish-dialog";
import { boundTab, isAdmin } from "./workspace-settings";

export function sitesTab(engine: Engine): Tab {
  return boundTab(engine, "sites", t("Sites"), "globe", (panel, w) => {
    const admin = isAdmin(w);
    const st = status();
    const allow = toggle(t("Allow publishing to the web"), true, t("When off, every published page in this workspace stops being served until it is turned back on."));
    allow.input.disabled = !admin;
    const list = h("div", { class: "rows" });
    const empty = h("p", { class: "muted", text: t("Nothing is published yet. Publish a page from its menu or its Share dialog.") });
    const top = section(t("Publishing"));
    top.append(allow.el);
    const sites = section(t("Published pages"));
    sites.append(list, empty, st.el);
    panel.append(top, sites);

    const row = (x: Site) => {
      const title = h("a", { class: "row__title", attrs: { target: "_blank", rel: "noopener" } });
      const meta = h("span", { class: "row__meta" });
      const side = h("div", { class: "row__actions" });
      const open = iconButton("page", t("Open the page"));
      open.addEventListener("click", () => {
        engine.state.set("overlay", null);
        engine.openPage(x.page_id);
      });
      side.append(open);
      if (admin) {
        const stop = button(t("Unpublish"), "ghost");
        stop.addEventListener("click", async () => {
          const ok = await confirmDialog({ title: t("Unpublish {title}?", { title: x.page_title || t("Untitled") }), body: t("The address stops working right away."), confirm: t("Unpublish"), variant: "danger" });
          if (!ok) return;
          try {
            await busy(stop, () => engine.api.sites.unpublish(x.page_id));
            load();
          } catch (e) {
            st.error(e);
          }
        });
        side.append(stop);
      }
      return h("div", { class: "row" }, h("span", { class: "row__icon" }, iconNode("globe")), h("div", { class: "row__main" }, title, meta), side);
    };
    const update = (r: HTMLElement, x: Site) => {
      const title = r.querySelector(".row__title") as HTMLAnchorElement;
      setText(title, x.page_title || t("Untitled"));
      setAttr(title, "href", siteUrl(x));
      const bits = [siteUrl(x).replace(/^https?:\/\//, ""), x.password ? t("Password") : "", x.indexing ? t("Listed in search") : "", t("since {date}", { date: formatDate(x.created_at) })];
      setText(r.querySelector(".row__meta") as HTMLElement, bits.filter(Boolean).join(" · "));
    };
    const load = () => {
      engine.api.sites
        .list(w.id)
        .then((r) => {
          allow.input.checked = r.publishing;
          reconcile(list, r.sites, (x) => x.id, row, update);
          empty.hidden = r.sites.length > 0;
        })
        .catch((e) => st.error(e));
    };
    allow.input.addEventListener("change", async () => {
      const on = allow.input.checked;
      try {
        await engine.api.sites.setPublishing(w.id, on);
        toast(on);
      } catch (e) {
        allow.input.checked = !on;
        st.error(e);
      }
    });
    const toast = (on: boolean) => st.say(on ? t("Publishing is on.") : t("Publishing is off. Published pages are not served."), "success");
    load();
    return load;
  });
}
