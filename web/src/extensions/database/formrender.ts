import { t } from "../../engine/i18n";
import type { Filter, FormDef, FormQuestionDef, Member } from "../../engine/types";
import { h, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";

export interface FormRenderOptions {
  submit: (answers: Record<string, unknown>) => Promise<void>;
  people?: () => Promise<Member[]>;
  search?: (collection: string, q: string) => Promise<{ id: string; title: string }[]>;
  preview?: boolean;
}

function answerEmpty(v: unknown): boolean {
  return v === undefined || v === null || v === "" || v === false || (Array.isArray(v) && v.length === 0);
}

export function whenMatches(when: Filter | undefined, answers: Record<string, unknown>, questions: FormQuestionDef[]): boolean {
  if (!when?.prop) return true;
  const q = questions.find((x) => x.prop === when.prop);
  const v = answers[when.prop];
  const want = when.value;
  const asText = (x: unknown) => {
    if (typeof x === "string" && q?.options) return q.options.find((o) => o.id === x)?.name ?? x;
    return String(x ?? "");
  };
  switch (when.cond) {
    case "empty":
      return answerEmpty(v);
    case "not_empty":
      return !answerEmpty(v);
    case "is":
    case "eq":
      if (typeof want === "boolean") return !!v === want;
      if (Array.isArray(want)) return want.includes(v as string);
      return asText(v).toLowerCase() === asText(want).toLowerCase();
    case "is_not":
    case "ne":
      if (Array.isArray(want)) return !want.includes(v as string);
      return asText(v).toLowerCase() !== asText(want).toLowerCase();
    case "contains":
      if (Array.isArray(v)) return (Array.isArray(want) ? want : [want]).some((w) => (v as unknown[]).includes(w));
      return asText(v).toLowerCase().includes(asText(want).toLowerCase());
    case "not_contains":
      if (Array.isArray(v)) return !(Array.isArray(want) ? want : [want]).some((w) => (v as unknown[]).includes(w));
      return !asText(v).toLowerCase().includes(asText(want).toLowerCase());
    case "gt":
      return Number(v) > Number(want);
    case "lt":
      return Number(v) < Number(want);
  }
  return true;
}

export function renderForm(def: FormDef, o: FormRenderOptions): HTMLElement {
  const answers: Record<string, unknown> = {};
  const root = h("form", { class: "fform", attrs: { novalidate: "" } });
  const title = h("h1", { class: "fform__title", text: def.title || t("Untitled form") });
  root.append(title);
  if (def.description) root.append(h("p", { class: "fform__desc", text: def.description }));
  const fields = h("div", { class: "fform__fields" });
  const status = h("p", { class: "form-status", attrs: { role: "alert" } });
  const submit = h("button", { class: "btn btn--primary", attrs: { type: "submit" }, text: def.submit || t("Submit") });
  root.append(fields, status, submit);
  if (def.closed) {
    submit.disabled = true;
    setText(status, t("This form is not taking responses."));
  }
  const blocks = new Map<string, HTMLElement>();

  const control = (q: FormQuestionDef): HTMLElement => {
    const set = (v: unknown) => {
      answers[q.prop] = v;
      refresh();
    };
    switch (q.type) {
      case "title":
      case "text": {
        const a = h(q.type === "text" ? "textarea" : "input", { class: "input", attrs: { name: q.prop, "aria-label": q.label ?? q.name } }) as HTMLInputElement | HTMLTextAreaElement;
        if (a instanceof HTMLTextAreaElement) a.rows = 3;
        a.addEventListener("input", () => set(a.value));
        return a;
      }
      case "number": {
        const a = h("input", { class: "input", attrs: { type: "number", step: "any", name: q.prop, "aria-label": q.label ?? q.name } });
        a.addEventListener("input", () => set(a.value === "" ? null : Number(a.value)));
        return a;
      }
      case "url":
      case "email":
      case "phone": {
        const a = h("input", { class: "input", attrs: { type: q.type === "phone" ? "tel" : q.type, name: q.prop, "aria-label": q.label ?? q.name } });
        a.addEventListener("input", () => set(a.value));
        return a;
      }
      case "checkbox": {
        const a = h("input", { attrs: { type: "checkbox", name: q.prop } });
        a.addEventListener("change", () => set(a.checked));
        return h("label", { class: "fform__check" }, a, h("span", { text: t("Yes") }));
      }
      case "date": {
        const a = h("input", { class: "input", attrs: { type: "date", name: q.prop, "aria-label": q.label ?? q.name } });
        a.addEventListener("change", () => set(a.value ? { start: a.value } : null));
        return a;
      }
      case "select":
      case "status":
      case "multi_select": {
        const multi = q.type === "multi_select";
        const box = h("div", { class: "fform__choices", attrs: { role: multi ? "group" : "radiogroup" } });
        for (const opt of q.options ?? []) {
          const inp = h("input", { attrs: { type: multi ? "checkbox" : "radio", name: q.prop, value: opt.id } });
          inp.addEventListener("change", () => {
            if (multi) set(Array.from(box.querySelectorAll<HTMLInputElement>("input:checked")).map((x) => x.value));
            else set(inp.value);
          });
          box.append(h("label", { class: "fform__choice", attrs: { "data-color": opt.color } }, inp, h("span", { text: opt.name })));
        }
        return box;
      }
      case "person": {
        const sel = h("select", { class: "input", attrs: { name: q.prop, "aria-label": q.label ?? q.name } });
        sel.append(h("option", { attrs: { value: "" }, text: t("Choose a person") }));
        void o.people?.().then((ms) => {
          for (const m of ms) if (!m.disabled) sel.append(h("option", { attrs: { value: m.user_id }, text: m.name }));
        });
        sel.addEventListener("change", () => set(sel.value ? [sel.value] : []));
        return sel;
      }
      case "relation": {
        const sel = h("select", { class: "input", attrs: { name: q.prop, "aria-label": q.label ?? q.name } });
        sel.append(h("option", { attrs: { value: "" }, text: t("Choose a page") }));
        if (q.collection) void o.search?.(q.collection, "").then((rows) => {
          for (const r of rows) sel.append(h("option", { attrs: { value: r.id }, text: r.title || t("Untitled") }));
        });
        sel.addEventListener("change", () => set(sel.value ? [sel.value] : []));
        return sel;
      }
      case "place": {
        const lat = h("input", { class: "input", attrs: { type: "number", step: "any", placeholder: t("Latitude"), "aria-label": t("Latitude") } });
        const lng = h("input", { class: "input", attrs: { type: "number", step: "any", placeholder: t("Longitude"), "aria-label": t("Longitude") } });
        const upd = () => set(lat.value && lng.value ? { lat: Number(lat.value), lng: Number(lng.value) } : null);
        lat.addEventListener("input", upd);
        lng.addEventListener("input", upd);
        return h("div", { class: "dbpanel__row" }, lat, lng);
      }
    }
    return h("span", { class: "faint", text: t("This question cannot be answered here.") });
  };

  for (const q of def.questions) {
    const label = h("span", { class: "fform__label", text: q.label || q.name });
    if (q.required) label.append(h("span", { class: "fform__req", attrs: { "aria-hidden": "true" }, text: " *" }));
    const block = h("div", { class: "fform__q", attrs: { "data-prop": q.prop } }, label);
    if (q.help) block.append(h("span", { class: "field__hint", text: q.help }));
    block.append(control(q));
    blocks.set(q.prop, block);
    fields.append(block);
  }

  const visible = (q: FormQuestionDef) => whenMatches(q.when, answers, def.questions);
  function refresh() {
    for (const q of def.questions) {
      const b = blocks.get(q.prop);
      if (b) b.hidden = !visible(q);
    }
  }
  refresh();

  root.addEventListener("submit", (e) => {
    e.preventDefault();
    if (o.preview) {
      setText(status, t("This is a preview. Share the form to collect responses."));
      return;
    }
    for (const q of def.questions) {
      if (!visible(q)) continue;
      if (q.required && answerEmpty(answers[q.prop])) {
        setText(status, t("{name} is required", { name: q.label || q.name }));
        blocks.get(q.prop)?.querySelector<HTMLElement>("input, textarea, select")?.focus();
        return;
      }
    }
    const out: Record<string, unknown> = {};
    for (const q of def.questions) if (visible(q) && !answerEmpty(answers[q.prop])) out[q.prop] = answers[q.prop];
    submit.disabled = true;
    setText(status, "");
    o.submit(out)
      .then(() => {
        root.replaceChildren(h("div", { class: "fform__done" }, iconNode("success"), h("h2", { text: def.done || t("Thank you. Your response was recorded.") })));
      })
      .catch((err: unknown) => {
        submit.disabled = false;
        setText(status, (err as Error).message);
      });
  });
  return root;
}
