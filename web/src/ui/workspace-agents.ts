import type { AgentTrigger, AgentTriggerKind, CustomAgent, Engine, Workspace } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { confirmDialog, simpleDialog } from "./dialog";
import { h, setText } from "./dom";
import { actions, busy, button, field, iconButton, input, section, select, status, textarea, toggle } from "./form";
import { reconcile } from "./list";
import { toast } from "./toast";

const KINDS: [AgentTriggerKind, string][] = [
  ["manual", "Only when someone runs it"],
  ["schedule", "On a schedule"],
  ["row_added", "When a database gets a new row"],
  ["row_changed", "When a row in a database changes"],
  ["page_changed", "When a page changes"],
];
const EVERY: [NonNullable<AgentTrigger["every"]>, string][] = [
  ["once", "Once"],
  ["minutes", "Every few minutes"],
  ["hours", "Every few hours"],
  ["day", "Every day"],
  ["week", "Every week"],
];
const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

function describe(engine: Engine, a: CustomAgent): string {
  const tr = a.trigger;
  const title = (id?: string) => (id && engine.pages.get(id)?.title) || t("Untitled");
  switch (a.trigger_kind) {
    case "schedule":
      if (tr.every === "once") return t("Once, {when}", { when: tr.once_at ? formatDateTime(tr.once_at) : "" });
      if (tr.every === "minutes") return t("Every {n} minutes", { n: tr.n ?? 15 });
      if (tr.every === "hours") return t("Every {n} hours", { n: tr.n ?? 1 });
      if (tr.every === "week") return t("Every {day} at {at}", { day: t(DAYS[tr.weekday ?? 1] ?? "Monday"), at: tr.at ?? "09:00" });
      return t("Every day at {at}", { at: tr.at ?? "09:00" });
    case "row_added":
      return t("When a row is added to {db}", { db: title(tr.collection) });
    case "row_changed":
      return t("When a row of {db} changes", { db: title(tr.collection) });
    case "page_changed":
      return t("When {page} changes", { page: title(tr.page) });
  }
  return t("Runs when someone starts it");
}

function lastRun(a: CustomAgent): string {
  if (!a.last_run_at) return t("Has not run yet");
  const word = { running: t("running"), done: t("done"), failed: t("failed"), skipped: t("skipped") }[a.last_status] ?? a.last_status;
  return t("Last run {when}, {status}", { when: formatRelative(a.last_run_at), status: word });
}

function editor(engine: Engine, w: Workspace, a: CustomAgent | null, done: () => void): void {
  simpleDialog(
    a ? t("Edit agent") : t("New agent"),
    (body, close) => {
      const name = input({ name: "agent-name", value: a?.name ?? "" });
      const instructions = textarea(a?.instructions ?? "", 8);
      instructions.placeholder = t("What should it do? For example: every morning, list overdue tasks from Tasks in a new page under Standups and mention their owners.");
      const kind = select<AgentTriggerKind>(KINDS.map(([v, l]) => [v, t(l)] as const), a?.trigger_kind ?? "manual");
      const every = select(EVERY.map(([v, l]) => [v, t(l)] as const), a?.trigger.every ?? "day");
      const n = input({ type: "number", name: "agent-n", value: String(a?.trigger.n ?? 15) });
      n.min = "1";
      const at = input({ type: "time", name: "agent-at", value: a?.trigger.at ?? "09:00" });
      const day = select(DAYS.map((d, i) => [String(i), t(d)] as const), String(a?.trigger.weekday ?? 1));
      const once = input({ type: "datetime-local", name: "agent-once" });
      if (a?.trigger.once_at) {
        const d = new Date(a.trigger.once_at);
        once.value = new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
      }
      const pages = engine.pages.isLoaded(w.id) ? engine.pages.all(w.id) : [];
      const db = select<string>([["", t("Choose a database")], ...pages.filter((p) => p.kind === "collection").map((p) => [p.id, p.title || t("Untitled")] as [string, string])], a?.trigger.collection ?? "");
      const page = select<string>([["", t("Choose a page")], ...pages.filter((p) => p.kind === "doc").map((p) => [p.id, p.title || t("Untitled")] as [string, string])], a?.trigger.page ?? "");
      const model = select<string>([["", t("Instance default")]], a?.model ?? "");
      void engine.api.ai.models(w.id).then((c) => {
        for (const m of c.models.filter((x) => !x.audio)) model.append(h("option", { attrs: { value: m.ref }, text: m.label }));
        model.value = a?.model ?? "";
      }, () => undefined);
      const fEvery = field(t("Repeat"), every);
      const fN = field(t("Interval"), n);
      const fAt = field(t("Time"), at);
      const fDay = field(t("Day"), day);
      const fOnce = field(t("When"), once);
      const fDb = field(t("Database"), db);
      const fPage = field(t("Page"), page);
      const paint = () => {
        const k = kind.value;
        const e = every.value;
        fEvery.hidden = k !== "schedule";
        fN.hidden = k !== "schedule" || (e !== "minutes" && e !== "hours");
        fAt.hidden = k !== "schedule" || (e !== "day" && e !== "week");
        fDay.hidden = k !== "schedule" || e !== "week";
        fOnce.hidden = k !== "schedule" || e !== "once";
        fDb.hidden = k !== "row_added" && k !== "row_changed";
        fPage.hidden = k !== "page_changed";
      };
      kind.addEventListener("change", paint);
      every.addEventListener("change", paint);
      paint();
      const st = status();
      const save = button(a ? t("Save") : t("Create agent"), "primary", undefined, "submit");
      const cancel = button(t("Cancel"), "ghost");
      cancel.addEventListener("click", close);
      const form = h(
        "form",
        { class: "form-grid", attrs: { novalidate: "" } },
        field(t("Name"), name),
        field(t("Instructions"), instructions, t("The agent acts as you, with your access, and every change it makes can be undone.")),
        field(t("Runs"), kind),
        fEvery,
        fN,
        fAt,
        fDay,
        fOnce,
        fDb,
        fPage,
        field(t("Model"), model),
        st.el,
        actions(cancel, save),
      );
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st.clear();
        const k = kind.value as AgentTriggerKind;
        const trigger: AgentTrigger = {};
        if (k === "schedule") {
          trigger.every = every.value as NonNullable<AgentTrigger["every"]>;
          trigger.tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
          if (trigger.every === "minutes" || trigger.every === "hours") trigger.n = Math.max(1, Number(n.value) || 1);
          if (trigger.every === "day" || trigger.every === "week") trigger.at = at.value || "09:00";
          if (trigger.every === "week") trigger.weekday = Number(day.value);
          if (trigger.every === "once") trigger.once_at = once.value ? new Date(once.value).getTime() : Date.now() + 60000;
        }
        if (k === "row_added" || k === "row_changed") trigger.collection = db.value;
        if (k === "page_changed") trigger.page = page.value;
        const payload: Partial<CustomAgent> = { name: name.value.trim(), instructions: instructions.value, trigger_kind: k, trigger, model: model.value };
        try {
          await busy(save, () => (a ? engine.api.agent.update(a.id, payload) : engine.api.agent.create(w.id, payload)));
          close();
          done();
        } catch (err) {
          st.error(err);
        }
      });
      body.append(form);
      return name;
    },
    undefined,
    "md",
  );
}

export function customAgentsSection(engine: Engine, w: Workspace): { el: HTMLElement; refresh: () => void } {
  const s = section(t("Custom agents"), t("Agents that work on their own: on a schedule, when a database gets a new or changed row, or when a page changes. Each acts as the person who created it."));
  const list = h("div", { class: "rows" });
  const empty = h("p", { class: "muted", text: t("No custom agents yet.") });
  const st = status();
  const add = button(t("New agent"), "default", "plus");
  add.hidden = w.role === "viewer" || w.role === "guest";
  const load = () => {
    engine.api.agent
      .list(w.id)
      .then(({ agents }) => {
        reconcile(list, agents, (a) => a.id, row, update);
        empty.hidden = agents.length > 0;
      })
      .catch((err) => st.error(err));
  };
  add.addEventListener("click", () => editor(engine, w, null, load));
  const row = (a: CustomAgent): HTMLElement => {
    const on = toggle(a.name, a.enabled);
    on.el.classList.add("agent-row__switch");
    const meta = h("span", { class: "row__meta" });
    const run = iconButton("play", t("Run now"));
    const edit = iconButton("pencil", t("Edit"));
    const runs = iconButton("clock", t("Recent runs"));
    const del = iconButton("trash", t("Delete"));
    const history = h("div", { class: "agent-runs" });
    history.hidden = true;
    const el = h("div", { class: "row row--stack agent-row" }, h("div", { class: "agent-row__top" }, h("div", { class: "row__main" }, on.el, meta), h("div", { class: "row__actions" }, run, runs, edit, del)), history);
    el.dataset.id = a.id;
    on.input.addEventListener("change", async () => {
      try {
        await engine.api.agent.update(a.id, { enabled: on.input.checked });
        load();
      } catch (err) {
        on.input.checked = !on.input.checked;
        st.error(err);
      }
    });
    run.addEventListener("click", async () => {
      try {
        await engine.api.agent.run(a.id);
        toast({ message: t("{name} is running.", { name: a.name }), tone: "info" });
        setTimeout(load, 1500);
      } catch (err) {
        st.error(err);
      }
    });
    edit.addEventListener("click", () => {
      const cur = (el as HTMLElement & { agent?: CustomAgent }).agent ?? a;
      editor(engine, w, cur, load);
    });
    del.addEventListener("click", async () => {
      if (!(await confirmDialog({ title: t("Delete {name}?", { name: a.name }), body: t("The agent stops running. Changes it already made stay, and each can still be undone from its run."), confirm: t("Delete"), variant: "danger" }))) return;
      try {
        await engine.api.agent.remove(a.id);
        load();
      } catch (err) {
        st.error(err);
      }
    });
    runs.addEventListener("click", async () => {
      history.hidden = !history.hidden;
      if (history.hidden) return;
      history.replaceChildren(h("p", { class: "muted", text: t("Loading") }));
      try {
        const r = await engine.api.agent.runs(a.id);
        history.replaceChildren();
        if (!r.runs.length) history.append(h("p", { class: "muted", text: t("No runs yet.") }));
        for (const x of r.runs) {
          const label = h("span", { class: "row__meta", text: `${formatRelative(x.created_at)} · ${x.status}${x.ops ? ` · ${x.ops === 1 ? t("1 change") : t("{n} changes", { n: x.ops })}` : ""}` });
          const undo = button(x.status === "undone" ? t("Undone") : t("Undo"), "ghost");
          undo.disabled = !x.run_id || !x.ops || x.status === "undone";
          undo.addEventListener("click", async () => {
            try {
              await engine.api.me.undoRun(x.run_id as string);
              setText(undo, t("Undone"));
              undo.disabled = true;
            } catch (err) {
              st.error(err);
            }
          });
          history.append(h("div", { class: "agent-runs__row" }, label, undo));
        }
      } catch (err) {
        st.error(err);
      }
    });
    update(el, a);
    return el;
  };
  const update = (el: HTMLElement, a: CustomAgent) => {
    (el as HTMLElement & { agent?: CustomAgent }).agent = a;
    const lbl = el.querySelector(".switch__label");
    if (lbl) setText(lbl, a.name);
    const sw = el.querySelector<HTMLInputElement>("input[role=switch]");
    if (sw) {
      sw.checked = a.enabled;
      sw.disabled = !a.can_edit;
    }
    setText(el.querySelector(".row__meta") as HTMLElement, `${describe(engine, a)} · ${lastRun(a)}`);
    for (const b of Array.from(el.querySelectorAll<HTMLButtonElement>(".row__actions button"))) {
      const tip = b.getAttribute("aria-label");
      if (tip === t("Edit") || tip === t("Delete")) b.hidden = !a.can_edit;
    }
  };
  s.append(list, empty, st.el, actions(add));
  load();
  return { el: s, refresh: load };
}
