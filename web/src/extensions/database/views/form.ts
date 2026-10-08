import { t } from "../../../engine/i18n";
import type { CollectionView, FormConfig, FormDef, FormQuestion, Property } from "../../../engine/types";
import { simpleDialog } from "../../../ui/dialog";
import { h, setText } from "../../../ui/dom";
import { copyText } from "../../../ui/form";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { openMenu } from "../../../ui/menu";
import { toast } from "../../../ui/toast";
import { renderForm } from "../formrender";
import { openPanel } from "../panel";
import { openAddProperty } from "../propedit";
import { computed, CONDS, condKind, TYPE_INFO } from "../props";
import type { ViewHost, ViewRenderer } from "../surface";
import { saveConfig } from "../toolbar";

const PUBLIC_TYPES = new Set(["title", "text", "number", "select", "multi_select", "status", "date", "checkbox", "url", "email", "phone", "place"]);

export function formView(host: ViewHost): ViewRenderer {
  const session = host.makeSession({ limit: 1 });
  const ctx = host.ctx(session);
  const view = (): CollectionView => host.view();
  const cfg = (): FormConfig => view().config.form ?? {};
  const el = h("div", { class: "formb" });
  const left = h("div", { class: "formb__main" });
  const right = h("aside", { class: "formb__side" });
  el.append(left, right);
  const titleIn = h("input", { class: "formb__title", attrs: { type: "text", placeholder: t("Form title"), "aria-label": t("Form title") } });
  const descIn = h("textarea", { class: "formb__desc", attrs: { rows: "2", placeholder: t("Say what this form is for"), "aria-label": t("Form description") } });
  const qs = h("div", { class: "formb__questions" });
  const addQ = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add a question") }));
  left.append(titleIn, descIn, qs, addQ);

  const submitIn = h("input", { class: "input input--sm", attrs: { type: "text", "aria-label": t("Submit button label") } });
  const doneIn = h("textarea", { class: "input input--sm", attrs: { rows: "2", "aria-label": t("Message after submitting") } });
  const anon = h("input", { attrs: { type: "checkbox", role: "switch" } });
  const open = h("input", { attrs: { type: "checkbox", role: "switch" } });
  const pub = h("input", { attrs: { type: "checkbox", role: "switch" } });
  const link = h("input", { class: "input input--sm", attrs: { type: "text", readonly: "", "aria-label": t("Public link") } });
  const copy = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("copy"), h("span", { text: t("Copy") }));
  const fill = h("button", { class: "btn btn--primary btn--sm", attrs: { type: "button" } }, iconNode("form"), h("span", { text: t("Fill in the form") }));
  const linkRow = h("div", { class: "dbpanel__row" }, link, copy);
  const pubNote = h("p", { class: "field__hint", text: t("Anyone with the link can respond without signing in. People and relation questions are left out of the public form.") });
  right.append(
    h("h3", { class: "formb__heading", text: t("Responses") }),
    h("label", { class: "dpick__toggle" }, h("span", { text: t("Accept responses") }), open),
    h("label", { class: "dpick__toggle" }, h("span", { text: t("Anonymous responses") }), anon),
    h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Submit button") }), submitIn),
    h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("After submitting") }), doneIn),
    h("h3", { class: "formb__heading", text: t("Share") }),
    h("label", { class: "dpick__toggle" }, h("span", { text: t("Share on the web") }), pub),
    linkRow,
    pubNote,
    fill,
  );

  const writable = () => host.canWrite();
  const save = (change: (f: FormConfig) => FormConfig, debounce = 300) => saveConfig(ctx, host.viewId, (c) => ({ ...c, form: change(structuredClone(c.form ?? {})) }), debounce);

  const questions = (): FormQuestion[] => {
    const c = cfg();
    if (c.questions?.length) return c.questions;
    return ctx
      .col()
      .schema.filter((p) => !computed(p.type))
      .map((p) => ({ prop: p.id, required: p.type === "title" }));
  };

  const setQuestions = (list: FormQuestion[]) => save((f) => ({ ...f, questions: list }), 0);

  const paintQuestions = () => {
    const list = questions();
    reconcile(
      qs,
      list,
      (q) => q.prop,
      () => {
        const card = h("div", { class: "formb__q" });
        card.append(
          h("div", { class: "formb__qhead" }, h("span", { class: "formb__qtype" }), h("span", { class: "spacer" }), h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "data-act": "up", "aria-label": t("Move up") } }, iconNode("arrow-up")), h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "data-act": "down", "aria-label": t("Move down") } }, iconNode("arrow-down")), h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "data-act": "remove", "aria-label": t("Remove question") } }, iconNode("close"))),
          h("input", { class: "formb__qlabel", attrs: { type: "text", "data-field": "label", "aria-label": t("Question") } }),
          h("input", { class: "formb__qhelp", attrs: { type: "text", "data-field": "help", placeholder: t("Help text"), "aria-label": t("Help text") } }),
          h("div", { class: "formb__qfoot" }, h("label", { class: "fform__check" }, h("input", { attrs: { type: "checkbox", "data-field": "required" } }), h("span", { text: t("Required") })), h("button", { class: "link-btn", attrs: { type: "button", "data-act": "when" } })),
        );
        return card;
      },
      (card, q) => {
        card.dataset.prop = q.prop;
        const p = ctx.col().schema.find((x) => x.id === q.prop);
        const type = card.querySelector(".formb__qtype") as HTMLElement;
        if (type.dataset.sig !== `${p?.type}|${p?.name}`) {
          type.dataset.sig = `${p?.type}|${p?.name}`;
          type.replaceChildren(iconNode(p ? TYPE_INFO[p.type].icon : "text"), h("span", { text: p?.name ?? "" }));
        }
        const label = card.querySelector<HTMLInputElement>("[data-field=label]") as HTMLInputElement;
        if (document.activeElement !== label) label.value = q.label ?? "";
        label.placeholder = p?.name ?? "";
        const help = card.querySelector<HTMLInputElement>("[data-field=help]") as HTMLInputElement;
        if (document.activeElement !== help) help.value = q.help ?? "";
        (card.querySelector("[data-field=required]") as HTMLInputElement).checked = !!q.required;
        const whenBtn = card.querySelector("[data-act=when]") as HTMLElement;
        const wp = q.when?.prop ? ctx.col().schema.find((x) => x.id === q.when?.prop) : undefined;
        setText(whenBtn, wp ? t("Shown when {name} matches", { name: wp.name }) : t("Always shown"));
        for (const x of Array.from(card.querySelectorAll<HTMLInputElement | HTMLButtonElement>("input, button"))) x.disabled = !writable();
        card.dataset.public = p && !PUBLIC_TYPES.has(p.type) ? "internal" : "";
      },
    );
  };

  const paint = () => {
    const c = cfg();
    if (document.activeElement !== titleIn) titleIn.value = c.title ?? "";
    titleIn.placeholder = ctx.col().title || t("Form title");
    if (document.activeElement !== descIn) descIn.value = c.description ?? "";
    if (document.activeElement !== submitIn) submitIn.value = c.submit ?? "";
    submitIn.placeholder = t("Submit");
    if (document.activeElement !== doneIn) doneIn.value = c.done ?? "";
    doneIn.placeholder = t("Thank you. Your response was recorded.");
    anon.checked = !!c.anonymous;
    open.checked = !c.closed;
    const token = view().form_token ?? "";
    pub.checked = !!token;
    link.value = token ? `${location.origin}/form/${token}` : "";
    linkRow.hidden = !token;
    for (const x of [titleIn, descIn, submitIn, doneIn, anon, open, pub]) x.disabled = !writable();
    addQ.hidden = !writable();
    paintQuestions();
  };

  titleIn.addEventListener("input", () => save((f) => ({ ...f, title: titleIn.value })));
  descIn.addEventListener("input", () => save((f) => ({ ...f, description: descIn.value })));
  submitIn.addEventListener("input", () => save((f) => ({ ...f, submit: submitIn.value })));
  doneIn.addEventListener("input", () => save((f) => ({ ...f, done: doneIn.value })));
  anon.addEventListener("change", () => save((f) => ({ ...f, anonymous: anon.checked }), 0));
  open.addEventListener("change", () => save((f) => ({ ...f, closed: !open.checked }), 0));
  pub.addEventListener("change", () => {
    void host.engine.api.collections
      .formLink(host.viewId, pub.checked)
      .then((r) => {
        host.engine.collections.localView(ctx.colId, host.viewId, { form_token: r.token });
        paint();
      })
      .catch(host.engine.reportError);
  });
  copy.addEventListener("click", () => void copyText(link.value).then((ok) => ok && toast({ id: "copied", tone: "success", message: t("Link copied") })));

  qs.addEventListener("input", (e) => {
    const target = e.target as HTMLInputElement;
    const card = target.closest<HTMLElement>(".formb__q");
    const field = target.dataset.field;
    if (!card?.dataset.prop || !field) return;
    const list = questions().map((q) => ({ ...q }));
    const q = list.find((x) => x.prop === card.dataset.prop);
    if (!q) return;
    if (field === "required") q.required = target.checked;
    else if (field === "label") q.label = target.value;
    else if (field === "help") q.help = target.value;
    save((f) => ({ ...f, questions: list }), field === "required" ? 0 : 300);
  });
  qs.addEventListener("change", (e) => {
    const target = e.target as HTMLInputElement;
    if (target.dataset.field !== "required") return;
    const card = target.closest<HTMLElement>(".formb__q");
    const list = questions().map((q) => ({ ...q }));
    const q = list.find((x) => x.prop === card?.dataset.prop);
    if (q) {
      q.required = target.checked;
      setQuestions(list);
    }
  });
  qs.addEventListener("click", (e) => {
    const b = (e.target as Element).closest<HTMLElement>("[data-act]");
    const card = b?.closest<HTMLElement>(".formb__q");
    if (!b || !card?.dataset.prop) return;
    const list = questions().map((q) => ({ ...q }));
    const i = list.findIndex((x) => x.prop === card.dataset.prop);
    if (i < 0) return;
    if (b.dataset.act === "remove") {
      list.splice(i, 1);
      setQuestions(list);
    } else if (b.dataset.act === "up" && i > 0) {
      [list[i - 1], list[i]] = [list[i] as FormQuestion, list[i - 1] as FormQuestion];
      setQuestions(list);
    } else if (b.dataset.act === "down" && i < list.length - 1) {
      [list[i + 1], list[i]] = [list[i] as FormQuestion, list[i + 1] as FormQuestion];
      setQuestions(list);
    } else if (b.dataset.act === "when") {
      const earlier = list.slice(0, i).map((q) => ctx.col().schema.find((p) => p.id === q.prop)).filter((p): p is Property => !!p);
      const q = list[i] as FormQuestion;
      openMenu(
        b,
        [
          { label: t("Always shown"), checked: !q.when, run: () => setQuestions(list.map((x, k) => (k === i ? { ...x, when: undefined } : x)) as FormQuestion[]) },
          ...earlier.flatMap((p) =>
            CONDS[condKind(p)].map(([cond, label, needs]) => ({
              label: `${p.name}: ${t(label)}`,
              checked: q.when?.prop === p.id && q.when.cond === cond,
              run: () => {
                const apply = (value?: unknown) => setQuestions(list.map((x, k) => (k === i ? { ...x, when: { prop: p.id, cond, ...(value !== undefined ? { value } : {}) } } : x)) as FormQuestion[]);
                if (!needs) return apply();
                if (p.options?.length) {
                  openMenu(b, p.options.map((o) => ({ label: o.name, run: () => apply([o.id]) })), { side: "bottom", align: "start", label: p.name });
                  return;
                }
                if (p.type === "checkbox") {
                  openMenu(b, [{ label: t("Checked"), run: () => apply(true) }, { label: t("Unchecked"), run: () => apply(false) }], { side: "bottom", align: "start", label: p.name });
                  return;
                }
                openPanel(
                  b,
                  (panel, close) => {
                    const input = h("input", { class: "input input--sm", attrs: { type: "text", "aria-label": t("Value for {name}", { name: p.name }) } });
                    input.addEventListener("keydown", (ev) => {
                      if (ev.key !== "Enter") return;
                      apply(p.type === "number" ? Number(input.value) : input.value);
                      close();
                    });
                    panel.append(h("div", { class: "dbpanel__pad" }, input));
                  },
                  { label: p.name },
                );
              },
            })),
          ),
        ],
        { side: "bottom", align: "start", label: t("Show this question") },
      );
    }
  });

  addQ.addEventListener("click", () => {
    const used = new Set(questions().map((q) => q.prop));
    const free = ctx.col().schema.filter((p) => !computed(p.type) && !used.has(p.id));
    openMenu(
      addQ,
      [
        ...free.map((p) => ({ label: p.name, icon: TYPE_INFO[p.type].icon, run: () => setQuestions([...questions(), { prop: p.id }]) })),
        { kind: "separator" as const },
        { label: t("New property"), icon: "plus" as const, run: () => openAddProperty(addQ, ctx, "", (id) => setQuestions([...questions(), { prop: id }])) },
      ],
      { side: "bottom", align: "start", label: t("Add a question") },
    );
  });

  fill.addEventListener("click", () => {
    void host.engine.api.collections
      .form(host.viewId)
      .then((r) => {
        simpleDialog(r.form.title || t("Form"), (body, close) => {
          body.append(
            renderForm(r.form as FormDef, {
              submit: async (answers) => {
                await host.engine.api.collections.submit(host.viewId, answers);
                setTimeout(close, 1200);
              },
              people: () => host.engine.collections.people(host.workspaceId()),
              search: (col, q) => host.engine.api.collections.search(col, q, 100).then((x) => x.rows),
            }),
          );
        });
      })
      .catch(host.engine.reportError);
  });

  paint();
  const off = host.engine.collections.subscribe(ctx.colId, () => paint());

  return {
    el,
    ready: Promise.resolve(),
    ctx,
    configChanged: paint,
    schemaChanged: paint,
    setSearch() {
      return;
    },
    destroy() {
      off();
      session.close();
      el.remove();
    },
  };
}
