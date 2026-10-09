import type { Engine } from "../../engine";
import { t } from "../../engine/i18n";
import type { ActionKind, AutomationAction, Member, Property } from "../../engine/types";
import { h, setText } from "../../ui/dom";
import { button, field, iconButton, input, select, textarea } from "../../ui/form";

export interface ActionContext {
  engine: Engine;
  workspaceId: string;
  collectionId?: string;
  automation: boolean;
}

export const KIND_LABELS: Record<ActionKind, string> = {
  edit: "Edit pages",
  add: "Add a page",
  notify: "Send a notification",
  mail: "Send mail",
  webhook: "Send a webhook",
  insert: "Insert blocks",
  open: "Open a page or link",
  confirm: "Ask to confirm first",
};

const SETTABLE = ["title", "text", "number", "select", "multi_select", "status", "date", "person", "checkbox", "url", "email", "phone", "relation"];

export interface ActionEditor {
  el: HTMLElement;
  value(): AutomationAction[];
}

function databases(c: ActionContext): [string, string][] {
  const pages = c.engine.pages.isLoaded(c.workspaceId) ? c.engine.pages.all(c.workspaceId) : [];
  return pages.filter((p) => p.kind === "collection").map((p) => [p.id, p.title || t("Untitled")]);
}

async function schemaOf(c: ActionContext, colId: string): Promise<Property[]> {
  if (!colId) return [];
  const col = await c.engine.collections.load(colId).catch(() => null);
  return col?.schema ?? [];
}

function valueControl(p: Property, current: unknown, members: Member[], automation: boolean): { el: HTMLElement; read(): unknown } {
  const pick = (opts: [string, string][], v: string) => {
    const s = select(opts, v);
    s.classList.add("select--sm");
    return s;
  };
  switch (p.type) {
    case "select":
    case "status":
    case "multi_select": {
      const cur = Array.isArray(current) ? String(current[0] ?? "") : typeof current === "string" ? current : "";
      const s = pick([["@clear", t("Clear")], ...(p.options ?? []).map((o) => [o.id, o.name] as [string, string])], cur || (p.options?.[0]?.id ?? "@clear"));
      return { el: s, read: () => (s.value === "@clear" ? "@clear" : p.type === "multi_select" ? [s.value] : s.value) };
    }
    case "checkbox": {
      const s = pick([["true", t("Checked")], ["false", t("Unchecked")], ["@toggle", t("Flip it")]], current === false ? "false" : current === "@toggle" ? "@toggle" : "true");
      return { el: s, read: () => (s.value === "@toggle" ? "@toggle" : s.value === "true") };
    }
    case "date": {
      const s = pick([["@today", t("Today")], ["@now", t("Now")], ["@clear", t("Clear")]], typeof current === "string" ? current : "@today");
      return { el: s, read: () => s.value };
    }
    case "person": {
      const cur = current === "@actor" ? "@actor" : Array.isArray(current) ? String(current[0] ?? "") : "@actor";
      const s = pick([["@actor", automation ? t("Person who triggered it") : t("Person who clicked")], ["@clear", t("Clear")], ...members.map((m) => [m.user_id, m.name || m.email] as [string, string])], cur);
      return { el: s, read: () => (s.value.startsWith("@") ? s.value : [s.value]) };
    }
    case "relation": {
      const s = pick([["@row", automation ? t("The page that triggered it") : t("This page")], ["@clear", t("Clear")]], current === "@clear" ? "@clear" : "@row");
      return { el: s, read: () => s.value };
    }
    case "number": {
      const inc = typeof current === "object" && current !== null && "inc" in current;
      const mode = pick([["set", t("Set to")], ["inc", t("Increase by")]], inc ? "inc" : "set");
      const n = input({ type: "number", value: String(inc ? (current as { inc: number }).inc : typeof current === "number" ? current : 1) });
      n.classList.add("input--sm");
      return { el: h("span", { class: "act__pair" }, mode, n), read: () => (mode.value === "inc" ? { inc: Number(n.value) || 0 } : Number(n.value) || 0) };
    }
  }
  const i = input({ value: typeof current === "string" ? current : "", placeholder: t("Text, or {{title}}, {{user}}, {{date}}") });
  i.classList.add("input--sm");
  return { el: i, read: () => i.value };
}

function valuesEditor(c: ActionContext, props: () => Promise<Property[]>, initial: Record<string, unknown>, members: Member[]): { el: HTMLElement; read(): Record<string, unknown>; reload(): void } {
  const list = h("div", { class: "act__vals" });
  const add = button(t("Set a property"), "ghost", "plus");
  add.classList.add("btn--sm");
  const rows: { el: HTMLElement; read(): [string, unknown] | null }[] = [];
  let schema: Property[] = [];
  const addRow = (propId: string, current: unknown) => {
    const usable = schema.filter((p) => SETTABLE.includes(p.type));
    const prop = h("select", { class: "select select--sm", attrs: { "aria-label": t("Property") } });
    for (const p of usable) prop.append(h("option", { attrs: { value: p.id }, text: p.name }));
    prop.value = propId || usable[0]?.id || "";
    const slot = h("span", { class: "act__slot" });
    let ctl = valueControl(usable.find((p) => p.id === prop.value) ?? { id: "", name: "", type: "text" }, current, members, c.automation);
    slot.append(ctl.el);
    prop.addEventListener("change", () => {
      const p = usable.find((x) => x.id === prop.value);
      if (!p) return;
      ctl = valueControl(p, undefined, members, c.automation);
      slot.replaceChildren(ctl.el);
    });
    const rm = iconButton("close", t("Remove"));
    const el = h("div", { class: "act__val" }, prop, slot, rm);
    const row = { el, read: (): [string, unknown] | null => (prop.value ? [prop.value, ctl.read()] : null) };
    rm.addEventListener("click", () => {
      rows.splice(rows.indexOf(row), 1);
      el.remove();
    });
    rows.push(row);
    list.append(el);
  };
  const reload = () => {
    void props().then((s) => {
      schema = s;
      for (const r of rows) r.el.remove();
      rows.length = 0;
      for (const [k, v] of Object.entries(initial)) if (schema.some((p) => p.id === k)) addRow(k, v);
    });
  };
  add.addEventListener("click", () => addRow("", undefined));
  reload();
  return {
    el: h("div", { class: "act__group" }, list, add),
    read: () => Object.fromEntries(rows.map((r) => r.read()).filter((x): x is [string, unknown] => !!x)),
    reload,
  };
}

function peopleEditor(c: ActionContext, initial: string[], members: Member[], personProps: Property[]): { el: HTMLElement; read(): string[] } {
  const chosen = new Set(initial);
  const chips = h("div", { class: "act__chips" });
  const options: [string, string][] = [
    ["@actor", c.automation ? t("Person who triggered it") : t("Person who clicked")],
    ...(c.automation ? [["@creator", t("Automation owner")] as [string, string]] : []),
    ...personProps.map((p) => [`prop:${p.id}`, t("People in {prop}", { prop: p.name })] as [string, string]),
    ...members.map((m) => [m.user_id, m.name || m.email] as [string, string]),
  ];
  const label = (v: string) => options.find((o) => o[0] === v)?.[1] ?? v;
  const adder = h("select", { class: "select select--sm", attrs: { "aria-label": t("Add a person") } });
  const paint = () => {
    chips.replaceChildren(
      ...[...chosen].map((v) => {
        const x = h("button", { class: "act__chip", attrs: { type: "button", "aria-label": t("Remove {name}", { name: label(v) }) } }, h("span", { text: label(v) }));
        x.addEventListener("click", () => {
          chosen.delete(v);
          paint();
        });
        return x;
      }),
    );
    adder.replaceChildren(h("option", { attrs: { value: "" }, text: t("Add a person") }), ...options.filter((o) => !chosen.has(o[0])).map((o) => h("option", { attrs: { value: o[0] }, text: o[1] })));
    adder.value = "";
  };
  adder.addEventListener("change", () => {
    if (adder.value) chosen.add(adder.value);
    paint();
  });
  paint();
  return { el: h("div", { class: "act__group" }, chips, adder), read: () => [...chosen] };
}

export async function actionEditor(c: ActionContext, initial: AutomationAction[], kinds: ActionKind[]): Promise<ActionEditor> {
  const members = await c.engine.collections.people(c.workspaceId).catch(() => [] as Member[]);
  const ownSchema = await schemaOf(c, c.collectionId ?? "");
  const personProps = ownSchema.filter((p) => p.type === "person");
  const list = h("div", { class: "acts" });
  const cards: { el: HTMLElement; read(): AutomationAction }[] = [];
  const addMenu = h("select", { class: "select select--sm acts__add", attrs: { "aria-label": t("Add an action") } });
  addMenu.append(h("option", { attrs: { value: "" }, text: t("Add an action") }), ...kinds.map((k) => h("option", { attrs: { value: k }, text: t(KIND_LABELS[k]) })));

  const card = (a: AutomationAction) => {
    const head = h("div", { class: "act__head" }, h("span", { class: "act__kind", text: t(KIND_LABELS[a.kind]) }));
    const rm = iconButton("trash", t("Remove action"));
    head.append(rm);
    const body = h("div", { class: "act__body" });
    let read: () => AutomationAction = () => ({ kind: a.kind });
    switch (a.kind) {
      case "edit": {
        const canTrigger = !!c.collectionId;
        const target = select<"trigger" | "all">([...(canTrigger ? [["trigger", c.automation ? t("The page that triggered it") : t("This page")] as const] : []), ["all", t("Every page in a database that matches")] as const], a.target ?? (canTrigger ? "trigger" : "all"));
        const db = select(databases(c), a.collection ?? c.collectionId ?? "");
        const dbField = field(t("Database"), db);
        const vals = valuesEditor(c, () => schemaOf(c, target.value === "all" ? db.value : (c.collectionId ?? "")), a.values ?? {}, members);
        const sync = () => {
          dbField.hidden = target.value !== "all";
        };
        target.addEventListener("change", () => {
          sync();
          vals.reload();
        });
        db.addEventListener("change", () => vals.reload());
        sync();
        body.append(field(t("Which pages"), target), dbField, field(t("Set"), vals.el));
        read = () => {
          const out: AutomationAction = { kind: "edit", target: target.value as "trigger" | "all", values: vals.read() };
          if (target.value === "all") out.collection = db.value;
          if (a.filter) out.filter = a.filter;
          return out;
        };
        break;
      }
      case "add": {
        const db = select(databases(c), a.collection ?? c.collectionId ?? "");
        const title = input({ value: a.title ?? "", placeholder: t("Title, or {{title}}, {{date}}") });
        const vals = valuesEditor(c, () => schemaOf(c, db.value), a.values ?? {}, members);
        db.addEventListener("change", () => vals.reload());
        body.append(field(t("Database"), db), field(t("Title"), title), field(t("Also set"), vals.el));
        read = () => ({ kind: "add", collection: db.value, title: title.value, values: vals.read() });
        break;
      }
      case "notify":
      case "mail": {
        const who = peopleEditor(c, a.people ?? [], members, personProps);
        const subject = input({ value: a.subject ?? "", placeholder: t("Subject, or {{title}}") });
        const message = textarea(a.message ?? "", 3);
        message.placeholder = t("Message. {{title}}, {{user}}, {{url}} and property names in braces are filled in.");
        body.append(field(a.kind === "mail" ? t("To") : t("Who"), who.el));
        if (a.kind === "mail") body.append(field(t("Subject"), subject));
        body.append(field(t("Message"), message));
        read = () => ({ kind: a.kind, people: who.read(), message: message.value, ...(a.kind === "mail" ? { subject: subject.value } : {}) });
        break;
      }
      case "webhook": {
        const url = input({ type: "url", value: a.url ?? "", placeholder: "https://example.com/hook" });
        body.append(field(t("Address"), url, t("Posts JSON about the page, signed with the workspace automation key in X-FormMD-Signature.")));
        read = () => ({ kind: "webhook", url: url.value.trim() });
        break;
      }
      case "insert": {
        const md = textarea(a.markdown ?? "", 4);
        md.placeholder = t("Markdown to insert, for example - [ ] Follow up with {{user}} on {{date}}");
        const where = select<"below" | "above">([["below", t("Below the button")], ["above", t("Above the button")]], a.where ?? "below");
        body.append(field(t("Blocks"), md), field(t("Where"), where));
        read = () => ({ kind: "insert", markdown: md.value, where: where.value as "below" | "above" });
        break;
      }
      case "open": {
        const pages = c.engine.pages.isLoaded(c.workspaceId) ? c.engine.pages.all(c.workspaceId) : [];
        const page = select<string>([["", t("A link instead")], ...pages.map((p) => [p.id, p.title || t("Untitled")] as [string, string])], a.page ?? "");
        const url = input({ type: "url", value: a.url ?? "", placeholder: "https://" });
        const urlField = field(t("Link"), url);
        const sync = () => {
          urlField.hidden = !!page.value;
        };
        page.addEventListener("change", sync);
        sync();
        body.append(field(t("Page"), page), urlField);
        read = () => (page.value ? { kind: "open", page: page.value } : { kind: "open", url: url.value.trim() });
        break;
      }
      case "confirm": {
        const message = input({ value: a.message ?? "", placeholder: t("Are you sure?") });
        body.append(field(t("Question"), message));
        read = () => ({ kind: "confirm", message: message.value });
        break;
      }
    }
    const el = h("section", { class: "act", attrs: { "data-kind": a.kind } }, head, body);
    const entry = { el, read: () => read() };
    rm.addEventListener("click", () => {
      cards.splice(cards.indexOf(entry), 1);
      el.remove();
      setText(empty, cards.length ? "" : t("No actions yet."));
    });
    cards.push(entry);
    list.append(el);
    setText(empty, "");
  };
  const empty = h("p", { class: "muted acts__empty" });
  addMenu.addEventListener("change", () => {
    const k = addMenu.value as ActionKind;
    addMenu.value = "";
    if (!k) return;
    const fresh: AutomationAction = { kind: k };
    if (k === "edit" && c.collectionId) fresh.target = "trigger";
    if (k === "add") fresh.collection = c.collectionId ?? databases(c)[0]?.[0] ?? "";
    card(fresh);
  });
  for (const a of initial) card(a);
  setText(empty, cards.length ? "" : t("No actions yet."));
  return { el: h("div", { class: "acts__wrap" }, list, empty, addMenu), value: () => cards.map((x) => x.read()) };
}
