import { ApiError } from "../../engine/api";
import { t } from "../../engine/i18n";
import type { Property } from "../../engine/types";
import { h } from "../../ui/dom";
import { toast } from "../../ui/toast";
import type { DbCtx } from "./cells";

export const AUTOFILL_TYPES = new Set(["text", "number", "select", "multi_select", "checkbox", "url", "date"]);

const running = new Map<string, AbortController>();

export function autofillEditor(p: Property): HTMLElement {
  const on = h("input", { attrs: { type: "checkbox", role: "switch" } });
  on.checked = !!p.autofill;
  const prompt = h("textarea", { class: "input input--sm", attrs: { rows: "3", "aria-label": t("Autofill instruction"), placeholder: t("What should AI fill in? For example, a one-sentence summary of the page") } });
  prompt.value = p.autofill?.prompt ?? "";
  const auto = h("input", { attrs: { type: "checkbox", role: "switch" } });
  auto.checked = p.autofill?.auto ?? true;
  const body = h(
    "div",
    { class: "dbpanel__stack" },
    prompt,
    h("label", { class: "dpick__toggle" }, h("span", { text: t("Update when a row changes") }), auto),
    h("p", { class: "field__hint", text: t("Reads the row's properties and page, writes as you and counts toward your AI limit.") }),
  );
  body.hidden = !on.checked;
  const sync = () => {
    p.autofill = on.checked ? { prompt: prompt.value.trim(), auto: auto.checked } : null;
  };
  on.addEventListener("change", () => {
    body.hidden = !on.checked;
    sync();
    if (on.checked) prompt.focus();
  });
  prompt.addEventListener("input", sync);
  auto.addEventListener("change", sync);
  return h("div", { class: "dbpanel__stack prop-ai" }, h("label", { class: "dpick__toggle" }, h("span", { text: t("AI autofill") }), on), body);
}

export function autofillChanged(before: Property, after: Property): boolean {
  if (!after.autofill) return false;
  return !before.autofill || before.autofill.prompt !== after.autofill.prompt;
}

export function runAutofill(ctx: DbCtx, prop: Property, rows: string[] = []): void {
  const key = `${ctx.colId}:${prop.id}`;
  running.get(key)?.abort();
  const abort = new AbortController();
  running.set(key, abort);
  const id = `autofill-${key}`;
  let total = rows.length;
  let done = 0;
  let filled = 0;
  const say = (message: string, tone: "info" | "success" | "warning" = "info", durationMs = 60000) => toast({ id, message, tone, durationMs });
  say(t("Filling {name} with AI", { name: prop.name }));
  void ctx.engine.api.collections
    .autofill(
      ctx.colId,
      prop.id,
      rows,
      (event, data) => {
        if (event === "start") total = data.total ?? total;
        if (event === "row") {
          done++;
          if (data.ok) filled++;
          say(t("Filling {name}: {done} of {total}", { name: prop.name, done, total }));
        }
      },
      abort.signal,
    )
    .then(
      () => say(t("Filled {n} of {total} rows", { n: filled, total }), "success", 5000),
      (e: unknown) => {
        if (e instanceof ApiError && e.code === "cancelled") return;
        say(e instanceof ApiError ? e.message : t("Autofill stopped."), "warning", 8000);
      },
    )
    .finally(() => {
      if (running.get(key) === abort) running.delete(key);
    });
}
