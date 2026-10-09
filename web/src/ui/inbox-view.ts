import type { Engine, InboxItem } from "../engine";
import type { InboxFilter } from "../engine/inbox";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { avatar, updateAvatar } from "./avatar";
import { h, ref, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { fallbackGlyph, pageIcon, paintIcon } from "./pageicon";
import { toast } from "./toast";

export function inboxSentence(n: InboxItem): [string, string, string] {
  const who = n.actor_name || t("Someone");
  const title = n.title || t("Untitled");
  switch (n.kind) {
    case "mention":
      return [who, t("mentioned you in"), title];
    case "comment":
      return [who, t("commented on"), title];
    case "reply":
      return [who, t("replied in"), title];
    case "suggestion":
      return [who, t("suggested an edit to"), title];
    case "decision":
      return [who, n.body.startsWith("accepted") ? t("accepted your suggestion in") : t("rejected your suggestion in"), title];
    case "share":
      return [who, t("shared with you"), title];
    case "assign":
      return [who, t("assigned you in"), title];
    case "reminder":
      return [t("Reminder"), t("in"), title];
  }
  return [who, "", title];
}

export function openNotification(engine: Engine, n: InboxItem): void {
  if (!n.read_at) void engine.inbox.mark([n.id], { read: true }).catch(engine.reportError);
  if (!n.page_id) return;
  engine.openPage(n.page_id);
  if (n.thread_id) window.dispatchEvent(new CustomEvent("formmd:thread", { detail: { page: n.page_id, thread: n.thread_id } }));
}

export function mountInboxView(engine: Engine): void {
  const { state, inbox } = engine;
  const view = ref("view-inbox");
  const tabs = h("div", { class: "segmented inbox__filter", attrs: { role: "group", "aria-label": t("Show") } });
  for (const [f, label] of [
    ["all", t("All")],
    ["unread", t("Unread")],
    ["archived", t("Archived")],
  ] as const) {
    tabs.append(h("button", { class: "segmented__btn segmented__btn--text", attrs: { type: "button", "data-filter": f, "aria-pressed": f === "all" ? "true" : "false" }, text: label }));
  }
  const readAll = h("button", { class: "btn btn--ghost", attrs: { type: "button" } }, iconNode("check"), h("span", { text: t("Mark all as read") }));
  const list = h("div", { class: "inbox__list", attrs: { role: "list" } });
  const empty = h("p", { class: "lib-empty" });
  const more = h("button", { class: "btn btn--ghost inbox__more", attrs: { type: "button" }, text: t("Show older") });
  view.append(h("div", { class: "lib" }, h("h1", { class: "lib-title", text: t("Inbox") }), h("div", { class: "lib-toolbar" }, tabs, readAll), list, empty, more));

  const row = (n: InboxItem) => {
    const lead = n.kind === "reminder" ? h("span", { class: "inbox__lead" }, iconNode("bell")) : avatar(n.actor_name, n.actor_color, false, n.actor_avatar);
    const who = h("strong");
    const verb = h("span");
    const title = h("strong", { class: "inbox__title" }, pageIcon(n.page_icon, fallbackGlyph("doc")), h("span"));
    const text = h("span", { class: "inbox__text" }, who, document.createTextNode(" "), verb, document.createTextNode(" "), title);
    const body = h("span", { class: "inbox__body" });
    const when = h("time", { class: "inbox__time" });
    const dot = h("span", { class: "inbox__dot", attrs: { "aria-hidden": "true" } });
    const read = h("button", { class: "icon-btn", attrs: { type: "button", "data-act": "read" } }, iconNode("check"));
    const archive = h("button", { class: "icon-btn", attrs: { type: "button", "data-act": "archive" } }, iconNode("archive"));
    return h(
      "div",
      { class: "inbox__item", attrs: { role: "listitem", tabindex: "0" } },
      lead,
      h("span", { class: "inbox__main" }, text, body),
      h("span", { class: "inbox__side" }, when, h("span", { class: "inbox__acts" }, read, archive), dot),
    );
  };
  const update = (el: HTMLElement, n: InboxItem) => {
    const lead = el.firstElementChild as HTMLElement;
    if (lead.classList.contains("avatar")) updateAvatar(lead, n.actor_name, n.actor_color, n.actor_avatar);
    const [who, verb, title] = inboxSentence(n);
    const text = el.querySelector(".inbox__text") as HTMLElement;
    setText(text.children[0] as HTMLElement, who);
    setText(text.children[1] as HTMLElement, verb);
    const tl = text.children[2] as HTMLElement;
    paintIcon(tl.firstElementChild as HTMLElement, n.page_icon, fallbackGlyph("doc"));
    setText(tl.lastElementChild as HTMLElement, title);
    const body = el.querySelector(".inbox__body") as HTMLElement;
    setText(body, n.kind === "decision" ? n.body.replace(/^(accepted|rejected): /, "") : n.body);
    body.hidden = !n.body;
    const when = el.querySelector(".inbox__time") as HTMLElement;
    setText(when, formatRelative(n.created_at));
    setAttr(when, "title", formatDateTime(n.created_at));
    setAttr(el, "data-unread", n.read_at ? null : "true");
    const read = el.querySelector('[data-act="read"]') as HTMLElement;
    const rl = n.read_at ? t("Mark as unread") : t("Mark as read");
    setAttr(read, "aria-label", rl);
    setAttr(read, "data-tip", rl);
    const archive = el.querySelector('[data-act="archive"]') as HTMLElement;
    const al = n.archived_at ? t("Move back to inbox") : t("Archive");
    setAttr(archive, "aria-label", al);
    setAttr(archive, "data-tip", al);
  };

  const paint = () => {
    for (const b of Array.from(tabs.children) as HTMLElement[]) setAttr(b, "aria-pressed", b.dataset.filter === inbox.filter ? "true" : "false");
    reconcile(list, inbox.items, (n) => n.id, row, update);
    empty.hidden = !inbox.loaded || inbox.items.length > 0;
    setText(empty, inbox.filter === "archived" ? t("Nothing archived.") : inbox.filter === "unread" ? t("You are all caught up.") : t("Mentions, replies, shares and reminders arrive here."));
    readAll.hidden = inbox.unread === 0 || inbox.filter === "archived";
    more.hidden = !inbox.more;
  };

  list.addEventListener("click", (e) => {
    const item = (e.target as Element).closest<HTMLElement>(".inbox__item");
    const n = inbox.items.find((x) => x.id === item?.dataset.key);
    if (!item || !n) return;
    const act = (e.target as Element).closest<HTMLElement>("[data-act]")?.dataset.act;
    if (act === "read") void inbox.mark([n.id], { read: !n.read_at }).catch(engine.reportError);
    else if (act === "archive") void inbox.mark([n.id], { archived: !n.archived_at }).catch(engine.reportError);
    else openNotification(engine, n);
  });
  list.addEventListener("keydown", (e) => {
    const item = (e.target as Element).closest<HTMLElement>(".inbox__item");
    if (!item || e.target !== item) return;
    const n = inbox.items.find((x) => x.id === item.dataset.key);
    if (e.key === "Enter" && n) openNotification(engine, n);
    else if (e.key === "ArrowDown") (item.nextElementSibling as HTMLElement | null)?.focus();
    else if (e.key === "ArrowUp") (item.previousElementSibling as HTMLElement | null)?.focus();
    else return;
    e.preventDefault();
  });
  tabs.addEventListener("click", (e) => {
    const f = (e.target as Element).closest<HTMLElement>("[data-filter]")?.dataset.filter as InboxFilter | undefined;
    if (f) inbox.setFilter(f);
  });
  readAll.addEventListener("click", () => void inbox.markAllRead().catch(engine.reportError));
  more.addEventListener("click", () => void inbox.loadMore().catch(engine.reportError));

  let unwatch: (() => void) | null = null;
  state.watch(
    "route",
    (r) => {
      if (r.name === "inbox" && !unwatch) {
        unwatch = inbox.watch();
        if (inbox.loaded) void inbox.load().catch(engine.reportError);
      } else if (r.name !== "inbox" && unwatch) {
        unwatch();
        unwatch = null;
      }
    },
    { immediate: true },
  );
  inbox.subscribe((c) => {
    if (c.kind === "arrived") {
      const n = c.items[0];
      if (!n || state.get("route").name === "inbox") return;
      const [who, verb, title] = inboxSentence(n);
      toast({ id: "inbox", message: `${who} ${verb} ${title}`, action: { label: t("Open"), run: () => openNotification(engine, n) } });
      return;
    }
    paint();
  });
  paint();
}
