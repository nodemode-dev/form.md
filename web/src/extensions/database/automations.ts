import type { Engine } from "../../engine";
import { formatDateTime, formatRelative } from "../../engine/format";
import { t } from "../../engine/i18n";
import type { Automation, AutomationTrigger, AutomationTriggerKind, Collection, Filter, Property } from "../../engine/types";
import { confirmDialog, simpleDialog } from "../../ui/dialog";
import { h, setText } from "../../ui/dom";
import { actions, busy, button, field, iconButton, input, select, status, toggle } from "../../ui/form";
import { reconcile } from "../../ui/list";
import { toast } from "../../ui/toast";
import { actionEditor } from "../automation/actions";
import { automationApi } from "../automation/api";

const KINDS: [AutomationTriggerKind, string][] = [
  ["row_added", "When a page is added"],
  ["prop_edited", "When a property is edited"],
  ["schedule", "On a schedule"],
];
const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

function propName(col: Collection, id: string): string {
  return col.schema.find((p) => p.id === id)?.name ?? t("a property");
}

function describe(col: Collection, a: Automation): string {
  const tr = a.trigger;
  let when: string;
  switch (a.trigger_kind) {
    case "row_added":
      when = t("When a page is added");
      break;
    case "prop_edited":
      when = tr.props?.length ? t("When {props} changes", { props: tr.props.map((id) => propName(col, id)).join(", ") }) : t("When any property changes");
      break;
    default:
      when = tr.every === "hours" ? t("Every {n} hours", { n: tr.n ?? 1 }) : tr.every === "week" ? t("Every {day} at {at}", { day: t(DAYS[tr.weekday ?? 1] ?? "Monday"), at: tr.at ?? "09:00" }) : t("Every day at {at}", { at: tr.at ?? "09:00" });
  }
  const n = a.actions.length;
  return `${when} · ${n === 1 ? t("1 action") : t("{n} actions", { n })}`;
}

function lastRun(a: Automation): string {
  if (!a.last_run_at) return a.next_run_at ? t("Next run {when}", { when: formatDateTime(a.next_run_at) }) : t("Has not run yet");
  const word = { ok: t("ok"), partial: t("partly failed"), failed: t("failed"), skipped: t("skipped"), "": "" }[a.last_status] ?? a.last_status;
  return t("Last run {when}, {status}", { when: formatRelative(a.last_run_at), status: word });
}

function conditionEditor(col: Collection, f: Filter | undefined): { el: HTMLElement; read(): Filter | undefined } {
  const usable = col.schema.filter((p) => ["select", "status", "checkbox"].includes(p.type));
  const prop = select<string>([["", t("Always")], ...usable.map((p) => [p.id, t("Only when {prop} is", { prop: p.name })] as [string, string])], f?.prop ?? "");
  const val = h("select", { class: "select" });
  const paintVals = () => {
    const p = usable.find((x) => x.id === prop.value);
    val.hidden = !p;
    if (!p) return;
    const opts: [string, string][] = p.type === "checkbox" ? [["true", t("Checked")], ["false", t("Unchecked")]] : (p.options ?? []).map((o) => [o.id, o.name]);
    val.replaceChildren(...opts.map(([v, l]) => h("option", { attrs: { value: v }, text: l })));
    const cur = f?.prop === p.id ? (Array.isArray(f.value) ? String(f.value[0] ?? "") : String(f.value ?? "")) : "";
    if (cur) val.value = cur;
  };
  prop.addEventListener("change", paintVals);
  paintVals();
  return {
    el: h("div", { class: "act__pair" }, prop, val),
    read: () => {
      const p = usable.find((x) => x.id === prop.value);
      if (!p) return undefined;
      return p.type === "checkbox" ? { prop: p.id, cond: "is", value: val.value === "true" } : { prop: p.id, cond: "is", value: [val.value] };
    },
  };
}

function propsPicker(col: Collection, chosen: string[]): { el: HTMLElement; read(): string[] } {
  const set = new Set(chosen);
  const wrap = h("div", { class: "act__checks" });
  for (const p of col.schema.filter((x: Property) => !["formula", "rollup", "created_time", "created_by", "last_edited_time", "last_edited_by", "unique_id", "button", "title"].includes(x.type))) {
    const box = h("input", { attrs: { type: "checkbox" } });
    box.checked = set.has(p.id);
    box.addEventListener("change", () => (box.checked ? set.add(p.id) : set.delete(p.id)));
    wrap.append(h("label", { class: "act__check" }, box, h("span", { text: p.name })));
  }
  return { el: wrap, read: () => [...set] };
}

function editor(engine: Engine, col: Collection, a: Automation | null, done: () => void): void {
  simpleDialog(
    a ? t("Edit automation") : t("New automation"),
    (body, close) => {
      const name = input({ name: "automation-name", value: a?.name ?? "", placeholder: t("Name") });
      const kind = select<AutomationTriggerKind>(KINDS.map(([v, l]) => [v, t(l)] as const), a?.trigger_kind ?? "prop_edited");
      const props = propsPicker(col, a?.trigger.props ?? []);
      const cond = conditionEditor(col, a?.trigger.filter);
      const every = select<"hours" | "day" | "week">([["hours", t("Every few hours")], ["day", t("Every day")], ["week", t("Every week")]], a?.trigger.every ?? "day");
      const n = input({ type: "number", value: String(a?.trigger.n ?? 1) });
      n.min = "1";
      const at = input({ type: "time", value: a?.trigger.at ?? "09:00" });
      const day = select(DAYS.map((d, i) => [String(i), t(d)] as const), String(a?.trigger.weekday ?? 1));
      const fProps = field(t("Properties"), props.el, t("Leave all unticked to react to any property."));
      const fCond = field(t("Condition"), cond.el);
      const fEvery = field(t("Repeat"), every);
      const fN = field(t("Hours"), n);
      const fAt = field(t("Time"), at);
      const fDay = field(t("Day"), day);
      const paint = () => {
        const k = kind.value;
        fProps.hidden = k !== "prop_edited";
        fCond.hidden = k === "schedule";
        fEvery.hidden = k !== "schedule";
        fN.hidden = k !== "schedule" || every.value !== "hours";
        fAt.hidden = fDay.hidden = k !== "schedule" || every.value === "hours";
        fDay.hidden = k !== "schedule" || every.value !== "week";
      };
      kind.addEventListener("change", paint);
      every.addEventListener("change", paint);
      paint();
      const slot = h("div", { class: "acts__slot" });
      const st = status();
      const save = button(a ? t("Save") : t("Create automation"), "primary", undefined, "submit");
      const cancel = button(t("Cancel"), "ghost");
      cancel.addEventListener("click", close);
      let acts: Awaited<ReturnType<typeof actionEditor>> | null = null;
      void actionEditor({ engine, workspaceId: col.workspace_id, collectionId: col.id, automation: true }, a?.actions ?? [], ["edit", "add", "notify", "mail", "webhook"]).then((ed) => {
        acts = ed;
        slot.append(ed.el);
      });
      const form = h("form", { class: "form-grid automation-form", attrs: { novalidate: "" } }, field(t("Name"), name), field(t("Trigger"), kind), fProps, fEvery, fN, fAt, fDay, fCond, field(t("Do this"), slot), st.el, actions(cancel, save));
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st.clear();
        const k = kind.value as AutomationTriggerKind;
        const trigger: AutomationTrigger = {};
        if (k === "prop_edited") trigger.props = props.read();
        if (k !== "schedule") {
          const f = cond.read();
          if (f) trigger.filter = f;
        } else {
          trigger.every = every.value as "hours" | "day" | "week";
          trigger.tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
          if (trigger.every === "hours") trigger.n = Math.max(1, Number(n.value) || 1);
          else trigger.at = at.value || "09:00";
          if (trigger.every === "week") trigger.weekday = Number(day.value);
        }
        const payload: Partial<Automation> = { name: name.value.trim(), trigger_kind: k, trigger, actions: acts?.value() ?? [] };
        try {
          await busy(save, () => (a ? automationApi(engine).update(a.id, payload) : automationApi(engine).create(col.id, payload)));
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

export function openAutomations(engine: Engine, col: Collection): void {
  simpleDialog(
    t("Automations for {db}", { db: col.title || t("Untitled") }),
    (body) => {
      const lead = h("p", { class: "muted", text: t("Automations run on their own when pages are added, when properties change or on a schedule. They act as the person who last saved them, and they never set each other off.") });
      const list = h("div", { class: "rows automation-list" });
      const empty = h("p", { class: "muted", text: t("No automations yet.") });
      const st = status();
      const add = button(t("New automation"), "primary", "plus");
      let design = col.can_design;
      const api = automationApi(engine);
      const load = () => {
        api
          .list(col.id)
          .then((r) => {
            design = r.can_design;
            add.hidden = !design;
            reconcile(list, r.automations, (a) => a.id, row, update);
            empty.hidden = r.automations.length > 0;
          })
          .catch((err) => st.error(err));
      };
      add.addEventListener("click", () => editor(engine, col, null, load));
      const row = (a: Automation): HTMLElement => {
        const on = toggle(a.name, a.enabled);
        const meta = h("span", { class: "row__meta" });
        const run = iconButton("play", t("Run now"));
        const runs = iconButton("clock", t("Recent runs"));
        const edit = iconButton("pencil", t("Edit"));
        const del = iconButton("trash", t("Delete"));
        const history = h("div", { class: "agent-runs" });
        history.hidden = true;
        const el = h("div", { class: "row row--stack agent-row" }, h("div", { class: "agent-row__top" }, h("div", { class: "row__main" }, on.el, meta), h("div", { class: "row__actions" }, run, runs, edit, del)), history);
        const cur = () => (el as HTMLElement & { auto?: Automation }).auto ?? a;
        on.input.addEventListener("change", async () => {
          try {
            await api.update(a.id, { ...cur(), enabled: on.input.checked });
            load();
          } catch (err) {
            on.input.checked = !on.input.checked;
            st.error(err);
          }
        });
        run.addEventListener("click", async () => {
          try {
            const r = await api.run(a.id);
            const failed = r.outcome.failed ?? [];
            toast({ tone: failed.length ? "danger" : "success", message: failed.length ? failed.join("; ") : (r.outcome.done ?? []).join("; ") || t("Nothing to do") });
            load();
          } catch (err) {
            st.error(err);
          }
        });
        edit.addEventListener("click", () => editor(engine, col, cur(), load));
        del.addEventListener("click", async () => {
          if (!(await confirmDialog({ title: t("Delete {name}?", { name: cur().name }), body: t("The automation stops. Changes it already made stay."), confirm: t("Delete"), variant: "danger" }))) return;
          try {
            await api.remove(a.id);
            load();
          } catch (err) {
            st.error(err);
          }
        });
        runs.addEventListener("click", async () => {
          history.hidden = !history.hidden;
          if (history.hidden) return;
          try {
            const r = await api.runs(a.id);
            history.replaceChildren(...(r.runs.length ? r.runs.map((x) => h("div", { class: "agent-runs__row" }, h("span", { class: "row__meta", text: `${formatRelative(x.at)} · ${x.status}${x.detail ? ` · ${x.detail}` : ""}` }))) : [h("p", { class: "muted", text: t("No runs yet.") })]));
          } catch (err) {
            st.error(err);
          }
        });
        update(el, a);
        return el;
      };
      const update = (el: HTMLElement, a: Automation) => {
        (el as HTMLElement & { auto?: Automation }).auto = a;
        const lbl = el.querySelector(".switch__label");
        if (lbl) setText(lbl, a.name);
        const sw = el.querySelector<HTMLInputElement>("input[role=switch]");
        if (sw) {
          sw.checked = a.enabled;
          sw.disabled = !design;
        }
        setText(el.querySelector(".row__meta") as HTMLElement, `${describe(col, a)} · ${lastRun(a)}`);
        for (const b of Array.from(el.querySelectorAll<HTMLButtonElement>(".row__actions button"))) {
          const tip = b.getAttribute("aria-label");
          if (tip !== t("Recent runs")) b.hidden = !design;
        }
      };
      const foot = h("div", { class: "automation-key" });
      const me = engine.state.get("me");
      const role = me?.workspaces.find((w) => w.id === col.workspace_id)?.role;
      if (role === "admin") {
        const reveal = button(t("Show the signing key"), "ghost", "key");
        const keyOut = h("code", { class: "automation-key__value" });
        keyOut.hidden = true;
        reveal.addEventListener("click", async () => {
          try {
            const r = await api.key(col.workspace_id);
            setText(keyOut, r.key);
            keyOut.hidden = false;
            reveal.hidden = true;
          } catch (err) {
            st.error(err);
          }
        });
        foot.append(h("span", { class: "muted", text: t("Webhooks from automations and buttons are signed with the workspace key.") }), reveal, keyOut);
      }
      body.append(lead, list, empty, st.el, actions(add), foot);
      load();
    },
    undefined,
    "md",
  );
}
