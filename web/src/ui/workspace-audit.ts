import type { AuditEntry, AuditQuery, Engine } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import type { Tab } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { button, field, input, section, select, status } from "./form";
import { reconcile } from "./list";
import { boundTab, isAdmin } from "./workspace-settings";
import { hookList } from "./workspace-webhooks";

const GROUPS: readonly (readonly [string, () => string])[] = [
  ["page.", () => t("Pages")],
  ["row.", () => t("Database rows")],
  ["collection.", () => t("Databases")],
  ["view.", () => t("Views")],
  ["comment.", () => t("Comments")],
  ["suggestion.", () => t("Suggestions")],
  ["member.", () => t("Members")],
  ["group.", () => t("Groups")],
  ["teamspace.", () => t("Teamspaces")],
  ["site.", () => t("Published sites")],
  ["webhook.", () => t("Webhooks")],
  ["workspace.", () => t("Workspace")],
  ["file.", () => t("Files")],
  ["scim.", () => t("Provisioning")],
];

const LABELS: Record<string, () => string> = {
  "page.create": () => t("Created a page"),
  "page.update": () => t("Changed a page"),
  "page.move": () => t("Moved a page"),
  "page.trash": () => t("Moved a page to the trash"),
  "page.restore": () => t("Restored a page"),
  "page.delete": () => t("Deleted a page"),
  "page.purge": () => t("Deleted a page for good"),
  "page.duplicate": () => t("Duplicated a page"),
  "page.share": () => t("Shared a page"),
  "page.unshare": () => t("Stopped sharing a page"),
  "page.link": () => t("Created a share link"),
  "page.link_open": () => t("Joined through a share link"),
  "page.invite_guest": () => t("Invited a guest"),
  "page.save_version": () => t("Saved a version"),
  "page.restore_version": () => t("Restored a version"),
  "row.create": () => t("Added a row"),
  "row.update": () => t("Edited a row"),
  "collection.schema": () => t("Changed database properties"),
  "collection.import_csv": () => t("Imported a CSV"),
  "view.create": () => t("Created a view"),
  "comment.create": () => t("Commented"),
  "comment.reply": () => t("Replied"),
  "comment.delete": () => t("Deleted a comment"),
  "suggestion.create": () => t("Suggested an edit"),
  "member.add": () => t("Added a member"),
  "member.remove": () => t("Removed a member"),
  "member.role": () => t("Changed a member's role"),
  "site.publish": () => t("Published a page"),
  "site.update": () => t("Changed a published site"),
  "site.unpublish": () => t("Unpublished a page"),
  "webhook.create": () => t("Added a webhook"),
  "webhook.delete": () => t("Deleted a webhook"),
  "file.upload": () => t("Uploaded a file"),
};

const actionLabel = (a: string) => LABELS[a]?.() ?? a;

function dayStart(v: string): number {
  return v ? new Date(`${v}T00:00:00`).getTime() : 0;
}

export function auditTab(engine: Engine): Tab {
  return boundTab(
    engine,
    "audit",
    t("Audit log"),
    "shield",
    (panel, w) => {
      const st = status();
      const action = select<string>([["", t("All actions")]], "");
      action.setAttribute("aria-label", t("Action"));
      const actor = select<string>([["", t("Everyone")]], "");
      actor.setAttribute("aria-label", t("Person"));
      const from = input({ type: "date" });
      const to = input({ type: "date" });
      const exportLink = h("a", { class: "btn", attrs: { download: "" } }, h("span", { text: t("Export CSV") }));
      const filters = h("div", { class: "audit-filters" }, field(t("Action"), action), field(t("Person"), actor), field(t("From"), from), field(t("To"), to));
      const retention = h("p", { class: "muted audit-retention" });
      const list = h("div", { class: "rows audit-rows" });
      const empty = h("p", { class: "muted", text: t("Nothing matches these filters.") });
      empty.hidden = true;
      const more = button(t("Load more"), "ghost");
      more.hidden = true;
      const log = section(t("Audit log"), t("Every change to pages, databases, sharing, members and settings in this workspace, with who made it."));
      log.append(filters, h("div", { class: "audit-bar" }, retention, exportLink), list, empty, h("div", {}, more), st.el);
      const siem = section(t("Stream to a SIEM"), t("Send each audit event as it happens to Splunk, Datadog, Elastic or any HTTPS collector."));
      const streams = hookList(engine, w, "siem");
      siem.append(streams.el);
      panel.append(log, siem);

      let entries: AuditEntry[] = [];
      let next = 0;
      let seq = 0;
      const query = (): AuditQuery => {
        const q: AuditQuery = { action: action.value, actor: actor.value, from: dayStart(from.value), to: to.value ? dayStart(to.value) + 86_400_000 : 0 };
        return q;
      };
      const row = () => h("div", { class: "row audit-row" }, h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })), h("time", { class: "audit-row__when" }));
      const update = (el: HTMLElement, e: AuditEntry) => {
        const [title, meta] = el.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        const who = e.actor_label || (e.actor_kind === "system" ? t("form.md") : e.actor_kind === "form" ? t("A form") : t("Someone"));
        setText(title, `${who}${e.agent ? ` (${e.agent})` : ""} · ${actionLabel(e.action)}`);
        const bits = [e.page_id ? e.page_title || t("Untitled") : "", e.detail, e.action];
        setText(meta, bits.filter(Boolean).join(" · "));
        const when = el.querySelector("time") as HTMLElement;
        setText(when, formatRelative(e.at));
        setAttr(when, "title", formatDateTime(e.at));
        setAttr(when, "datetime", new Date(e.at).toISOString());
      };
      const fillFacets = (actions: string[], actors: { id: string; label: string }[]) => {
        const keepA = action.value;
        const keepP = actor.value;
        const groups = GROUPS.filter(([prefix]) => actions.some((a) => a.startsWith(prefix)));
        action.replaceChildren(h("option", { attrs: { value: "" }, text: t("All actions") }));
        for (const [prefix, label] of groups) {
          const og = h("optgroup", { attrs: { label: label() } });
          og.append(h("option", { attrs: { value: prefix }, text: t("All {group}", { group: label().toLowerCase() }) }));
          for (const a of actions.filter((x) => x.startsWith(prefix))) og.append(h("option", { attrs: { value: a }, text: actionLabel(a) }));
          action.append(og);
        }
        for (const a of actions.filter((x) => !GROUPS.some(([p]) => x.startsWith(p)))) action.append(h("option", { attrs: { value: a }, text: a }));
        action.value = keepA;
        actor.replaceChildren(h("option", { attrs: { value: "" }, text: t("Everyone") }), ...actors.map((p) => h("option", { attrs: { value: p.id }, text: p.label })));
        actor.value = keepP;
      };
      const paint = () => {
        reconcile(list, entries, (e) => String(e.id), row, update);
        empty.hidden = entries.length > 0;
        more.hidden = !next;
        setAttr(exportLink, "href", engine.api.audit.csvUrl(w.id, query()));
      };
      const load = (append = false) => {
        const mine = ++seq;
        const q = query();
        if (append) q.before = next;
        engine.api.audit
          .list(w.id, q)
          .then((r) => {
            if (mine !== seq) return;
            entries = append ? [...entries, ...r.entries] : r.entries;
            next = r.next;
            if (r.actions && r.actors) fillFacets(r.actions, r.actors);
            setText(retention, r.retention_days > 0 ? plural(r.retention_days, "Entries are kept for {count} day.", "Entries are kept for {count} days.") : t("Entries are kept until an instance owner sets a retention period."));
            paint();
          })
          .catch((e) => st.error(e));
      };
      for (const c of [action, actor, from, to]) c.addEventListener("change", () => load());
      more.addEventListener("click", () => load(true));
      load();
      streams.load();
      return () => {
        load();
        streams.load();
      };
    },
    () => {
      const w = engine.workspace();
      return !!w && isAdmin(w);
    },
  );
}
