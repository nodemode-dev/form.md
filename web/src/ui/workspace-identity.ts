import type { Engine, ScimToken } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import type { Tab } from "./dialog";
import { confirmDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { busy, button, copyText, field, iconButton, input, section, status } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { toast } from "./toast";
import { boundTab, isAdmin } from "./workspace-settings";

export function identityTab(engine: Engine): Tab {
  return boundTab(
    engine,
    "identity",
    t("Provisioning"),
    "key",
    (panel, w) => {
      const st = status();
      const base = h("div", { class: "secret-box" }, h("span"), iconButton("copy", t("Copy address")));
      base.lastElementChild?.addEventListener("click", async () => toast({ id: "copy", message: (await copyText(base.firstElementChild?.textContent ?? "")) ? t("Address copied.") : t("Could not copy") }));
      const list = h("div", { class: "rows" });
      const empty = h("p", { class: "muted", text: t("No tokens yet.") });
      empty.hidden = true;
      const reveal = h("div", { class: "secret-box" });
      reveal.hidden = true;
      const label = input({ maxlength: 80, placeholder: t("Okta, Entra ID, Google Workspace") });
      label.setAttribute("aria-label", t("Token name"));
      const create = button(t("Create token"), "primary", "plus", "submit");
      const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, h("div", { class: "form-row form-row--inline" }, label, create));
      const sec = section(t("SCIM provisioning"), t("Let your identity provider create, update and deactivate members and groups of this workspace over SCIM 2.0. Deactivated people leave the workspace; their account stays."));
      sec.append(field(t("SCIM base address"), base), list, empty, form, reveal, st.el);
      panel.append(sec);

      const row = () => h("div", { class: "row" }, h("span", { class: "row__icon" }, iconNode("key")), h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })), h("div", { class: "row__actions" }));
      const update = (el: HTMLElement, x: ScimToken) => {
        const [title, meta] = el.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, x.label);
        setText(meta, x.last_used_at ? t("last used {when}", { when: formatRelative(x.last_used_at) }) : t("never used"));
        setAttr(meta, "title", formatDateTime(x.created_at));
        const side = el.querySelector(".row__actions") as HTMLElement;
        if (!side.firstElementChild) {
          const revoke = button(t("Revoke"), "ghost");
          revoke.addEventListener("click", async () => {
            const ok = await confirmDialog({ title: t("Revoke {name}?", { name: x.label }), body: t("The identity provider can no longer change members or groups with this token."), confirm: t("Revoke"), variant: "danger" });
            if (!ok) return;
            try {
              await busy(revoke, () => engine.api.scim.revoke(x.id));
              load();
            } catch (e) {
              st.error(e);
            }
          });
          side.append(revoke);
        }
      };
      const load = () => {
        engine.api.scim
          .tokens(w.id)
          .then((r) => {
            setText(base.firstElementChild!, r.base_url);
            reconcile(list, r.tokens, (x) => x.id, row, update);
            empty.hidden = r.tokens.length > 0;
          })
          .catch((e) => st.error(e));
      };
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st.clear();
        try {
          const r = await busy(create, () => engine.api.scim.create(w.id, label.value));
          label.value = "";
          const copy = iconButton("copy", t("Copy token"));
          copy.addEventListener("click", async () => toast({ id: "copy", message: (await copyText(r.token)) ? t("Token copied") : t("Could not copy") }));
          reveal.replaceChildren(h("span", { text: r.token }), copy);
          reveal.hidden = false;
          st.say(t("Paste this token into your identity provider now. It is shown only once."), "success");
          load();
        } catch (err) {
          st.error(err);
        }
      });
      load();
      return load;
    },
    () => {
      const w = engine.workspace();
      return !!w && isAdmin(w) && !w.personal;
    },
  );
}
