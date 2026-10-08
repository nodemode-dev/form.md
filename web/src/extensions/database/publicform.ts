import type { Engine } from "../../engine";
import { t } from "../../engine/i18n";
import { h } from "../../ui/dom";
import { renderForm } from "./formrender";

export function mountPublicForm(engine: Engine): void {
  const card = document.querySelector<HTMLElement>(".auth__card");
  if (!card) return;
  const panel = h("div", { class: "auth__panel auth__panel--form", attrs: { "data-panel": "form" } });
  card.append(panel);
  const token = location.pathname.replace(/^\/form\//, "").replace(/\/$/, "");
  const show = (def: NonNullable<Engine["boot"]["form"]>) => {
    panel.replaceChildren(
      renderForm(def, {
        submit: (answers) => engine.api.collections.publicSubmit(token, answers),
      }),
    );
    document.title = def.title || t("Form");
  };
  if (engine.boot.form) {
    show(engine.boot.form);
    return;
  }
  void engine.api.collections
    .publicForm(token)
    .then((r) => show(r.form))
    .catch(() => panel.replaceChildren(h("h1", { class: "auth__title", text: t("This form is not available") }), h("p", { class: "auth__lead", text: t("The link may have been turned off by the people who shared it.") })));
}
