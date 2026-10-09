import { t } from "../../engine/i18n";
import type { AutomationAction, Property } from "../../engine/types";
import { simpleDialog } from "../../ui/dialog";
import { h } from "../../ui/dom";
import { actions, busy, button, field, status } from "../../ui/form";
import { actionEditor } from "../automation/actions";
import type { DbCtx } from "./cells";

function fromProp(p: Property): AutomationAction[] {
  return (p.actions ?? []).map((a) => ({ ...(a.config ?? {}), kind: a.kind }) as AutomationAction);
}

export function editButtonProperty(ctx: DbCtx, p: Property): void {
  simpleDialog(
    t("What {name} does", { name: p.name }),
    (body, close) => {
      const slot = h("div", { class: "acts__slot" });
      const st = status();
      const save = button(t("Save"), "primary", undefined, "submit");
      const cancel = button(t("Cancel"), "ghost");
      cancel.addEventListener("click", close);
      let read: (() => AutomationAction[]) | null = null;
      void actionEditor({ engine: ctx.engine, workspaceId: ctx.workspaceId() || ctx.col().workspace_id, collectionId: ctx.colId, automation: false }, fromProp(p), ["edit", "add", "notify", "mail", "webhook"]).then((ed) => {
        read = ed.value;
        slot.append(ed.el);
      });
      const form = h("form", { class: "form-grid automation-form", attrs: { novalidate: "" } }, field(t("When clicked"), slot, t("Runs on the row whose button is clicked, as the person who clicks.")), st.el, actions(cancel, save));
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st.clear();
        const list = (read ? read() : fromProp(p)).map(({ kind, ...config }) => ({ kind, config: config as Record<string, unknown> }));
        try {
          await busy(save, () => ctx.engine.collections.schema(ctx.colId, [{ op: "update", id: p.id, prop: { ...p, actions: list } }]));
          close();
        } catch (err) {
          st.error(err);
        }
      });
      body.append(form);
    },
    undefined,
    "md",
  );
}
