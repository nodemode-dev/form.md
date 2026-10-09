import type { Engine } from "../../../engine";
import { t } from "../../../engine/i18n";
import { h, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { openPanel, rectOf } from "../panel";

export interface PickList {
  engine: Engine;
  colId: string;
  anchor: Element;
  title: string;
  ids: string[];
  truncated?: boolean | undefined;
  openRow: (id: string, e?: MouseEvent | KeyboardEvent) => void;
  add?: { label: string; run: () => Promise<string | null> } | undefined;
}

export function openPickList(o: PickList): void {
  openPanel(
    rectOf(o.anchor),
    (panel, close) => {
      const head = h("div", { class: "picklist__head" }, h("span", { class: "picklist__title", text: o.title }), h("span", { class: "picklist__count", text: o.ids.length ? t("{n} rows", { n: o.ids.length }) + (o.truncated ? "+" : "") : t("No rows") }));
      const list = h("ul", { class: "picklist__list", attrs: { role: "list", "aria-busy": o.ids.length ? "true" : "false" } });
      for (let i = 0; i < Math.min(o.ids.length, 10); i++) list.append(h("li", { class: "picklist__row picklist__row--ph" }));
      panel.append(h("div", { class: "picklist" }, head, list));
      if (o.add) {
        const add = h("button", { class: "picklist__add", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: o.add.label }));
        add.addEventListener("click", () => {
          add.disabled = true;
          void o
            .add?.run()
            .then((id) => {
              close();
              if (id) o.openRow(id);
            })
            .catch((e: unknown) => {
              add.disabled = false;
              o.engine.reportError(e);
            });
        });
        panel.querySelector(".picklist")?.append(add);
      }
      if (!o.ids.length) return;
      void o.engine.api.collections
        .query(o.colId, { view_id: "", ids: o.ids.slice(0, 200), limit: 200 })
        .then((r) => {
          const byId = new Map(r.rows.map((row) => [row.id, row]));
          const items = o.ids
            .slice(0, 200)
            .map((id) => byId.get(id))
            .filter((x) => !!x)
            .map((row) => {
              const b = h("button", { class: "picklist__row", attrs: { type: "button" } }, iconNode("page"), h("span", { class: "picklist__name" }));
              setText(b.lastElementChild as HTMLElement, row.title || t("Untitled"));
              b.addEventListener("click", (e) => {
                close();
                o.openRow(row.id, e);
              });
              return h("li", {}, b);
            });
          list.replaceChildren(...items);
          list.setAttribute("aria-busy", "false");
        })
        .catch((e: unknown) => {
          list.replaceChildren();
          list.setAttribute("aria-busy", "false");
          o.engine.reportError(e);
        });
    },
    { label: o.title, side: "bottom", align: "start", className: "picklist-pop", focus: false },
  );
}
