import { formatDateTime, formatRelative } from "../../../engine/format";
import { t } from "../../../engine/i18n";
import type { CommentReaction, FeedItem, ReactionIcon } from "../../../engine/types";
import { avatar, tileColor } from "../../../ui/avatar";
import { composer, REACTIONS, reconcileThreads, type ThreadView } from "../../../ui/comment-view";
import { h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { reconcile } from "../../../ui/list";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import type { ViewHost, ViewRenderer } from "../surface";
import { configKey } from "./shared";

const PAGE = 10;
const enc = encodeURIComponent;

interface Card {
  el: HTMLElement;
  open: boolean;
  views: Map<string, ThreadView>;
  unwatch?: () => void;
  sig: string;
}

export function feedView(host: ViewHost): ViewRenderer {
  const engine = host.engine;
  const session = host.makeSession({ watch: () => void refresh() });
  const ctx = host.ctx(session);
  const list = h("div", { class: "feed__list", attrs: { role: "feed", "aria-busy": "true" } });
  const more = h("button", { class: "tbl__more", attrs: { type: "button" } });
  const empty = h("p", { class: "feed__empty muted", text: t("Nothing here yet. Pages you add show up as posts.") });
  more.hidden = true;
  empty.hidden = true;
  const el = h("div", { class: "feed", attrs: { "data-inline": host.inline ? "true" : "false" } }, list, empty, more);
  const items = new Map<string, FeedItem>();
  const cards = new Map<string, Card>();
  let order: string[] = [];
  let total = 0;
  let seq = 0;
  let destroyed = false;
  const sizes = new ResizeObserver((entries) => {
    for (const e of entries) measure((e.target as HTMLElement).dataset.row ?? "");
  });

  const fetchPage = (offset: number, limit: number) =>
    engine.api.http.post<{ items: FeedItem[]; total: number }>(`/api/collections/${enc(host.colId)}/feed`, { view_id: host.viewId, offset, limit, ...(host.search() ? { search: host.search() } : {}) });

  const reactions = (box: HTMLElement, id: string, list: CommentReaction[]) => {
    reconcile(
      box,
      list,
      (r) => r.icon,
      (r) => {
        const b = h("button", { class: "feed__react", attrs: { type: "button" } }, iconNode(r.icon), h("span", { class: "feed__count" }));
        b.addEventListener("click", () => void toggle(id, b.dataset.key as ReactionIcon));
        return b;
      },
      (b, r) => {
        setAttr(b, "aria-pressed", r.mine ? "true" : "false");
        setText(b.lastElementChild as HTMLElement, String(r.count));
        const label = REACTIONS.find(([i]) => i === r.icon)?.[1]() ?? r.icon;
        setAttr(b, "aria-label", `${label}: ${r.names.join(", ")}`);
        setAttr(b, "data-tip", `${label}: ${r.names.join(", ")}`);
      },
    );
  };

  const toggle = async (id: string, icon: ReactionIcon) => {
    const it = items.get(id);
    if (!it) return;
    const on = !it.reactions.find((r) => r.icon === icon)?.mine;
    try {
      const path = `/api/pages/${enc(id)}/reactions/${enc(icon)}`;
      const r = (on ? await engine.api.http.put(path, {}) : await engine.api.http.del(path)) as unknown as { reactions?: CommentReaction[] } | undefined;
      it.reactions = r?.reactions ?? [];
      paintCard(id);
    } catch (e) {
      engine.reportError(e);
    }
  };

  const makeCard = (id: string): HTMLElement => {
    const face = h("span", { class: "feed__face" });
    const who = h("span", { class: "feed__who" });
    const when = h("time", { class: "feed__when" });
    const open = h("button", { class: "icon-btn icon-btn--sm feed__open", attrs: { type: "button", "aria-label": t("Open page"), "data-tip": t("Open page") } }, iconNode("expand"));
    const title = h("button", { class: "feed__title", attrs: { type: "button" } });
    const doc = h("div", { class: "doc-body feed__doc" });
    const expand = h("button", { class: "link-btn feed__expand", attrs: { type: "button" } });
    expand.hidden = true;
    const reacts = h("div", { class: "feed__reacts" });
    const addReact = h("button", { class: "icon-btn icon-btn--sm feed__addreact", attrs: { type: "button", "aria-label": t("Add a reaction"), "data-tip": t("Add a reaction") } }, iconNode("smile"));
    const talk = h("button", { class: "feed__talk", attrs: { type: "button", "aria-expanded": "false" } }, iconNode("comment"), h("span", { class: "feed__talkn" }));
    const threads = h("div", { class: "feed__threads" });
    const start = composer(engine, {
      workspaceId: () => host.workspaceId(),
      placeholder: t("Add a comment"),
      submitLabel: t("Send comment"),
      onSubmit: async (body) => {
        await engine.comments.run(id, () => engine.api.comments.create(id, { anchor: "page", body }));
      },
    });
    const comments = h("section", { class: "feed__comments", attrs: { "aria-label": t("Comments") } }, threads, start.el);
    comments.hidden = true;
    const el = h(
      "article",
      { class: "feed__item" },
      h("header", { class: "feed__head" }, face, h("span", { class: "feed__meta" }, who, when), open),
      title,
      h("div", { class: "feed__docwrap" }, doc, expand),
      h("footer", { class: "feed__foot" }, reacts, addReact, h("span", { class: "spacer" }), talk),
      comments,
    );
    const c: Card = { el, open: false, views: new Map(), sig: "" };
    cards.set(id, c);
    doc.dataset.row = id;
    sizes.observe(doc);
    const go = (e: MouseEvent) => host.openRow(id, e);
    open.addEventListener("click", go);
    title.addEventListener("click", go);
    expand.addEventListener("click", (e) => {
      const it = items.get(id);
      if (it?.clipped && el.dataset.expanded === "true") return go(e);
      const on = el.dataset.expanded !== "true";
      setAttr(el, "data-expanded", on ? "true" : null);
      setText(expand, on ? (it?.clipped ? t("Open the page to read the rest") : t("Show less")) : t("Show more"));
    });
    addReact.addEventListener("click", () => {
      const it = items.get(id);
      const entries: MenuEntry[] = REACTIONS.map(([icon, label]) => ({ label: label(), icon, checked: !!it?.reactions.find((r) => r.icon === icon)?.mine, run: () => void toggle(id, icon) }));
      openMenu(addReact, entries, { side: "bottom", align: "start", label: t("Reactions") });
    });
    talk.addEventListener("click", () => {
      c.open = !c.open;
      comments.hidden = !c.open;
      setAttr(talk, "aria-expanded", c.open ? "true" : "false");
      if (c.open) {
        c.unwatch ??= engine.comments.watch(id);
        void engine.comments.load(id, true).then(() => paintComments(id), engine.reportError);
        start.focus();
      }
    });
    return el;
  };

  const paintComments = (id: string) => {
    const c = cards.get(id);
    const it = items.get(id);
    if (!c || !it) return;
    const listing = engine.comments.get(id);
    if (engine.comments.has(id)) {
      const n = listing.threads.reduce((a, th) => a + th.comments.length, 0);
      it.comments = n;
      if (listing.reactions) it.reactions = listing.reactions;
    }
    if (c.open) {
      const pageThreads = listing.threads.filter((th) => th.anchor === "page" && !th.resolved_at);
      reconcileThreads(engine, id, c.el.querySelector(".feed__threads") as HTMLElement, pageThreads, listing, { workspaceId: () => host.workspaceId() }, c.views);
    }
    paintCard(id);
  };

  const paintCard = (id: string) => {
    const c = cards.get(id);
    const it = items.get(id);
    if (!c || !it) return;
    const q = <T extends HTMLElement>(s: string) => c.el.querySelector(s) as T;
    const face = q<HTMLElement>(".feed__face");
    const author = it.author || t("Someone");
    if (face.dataset.name !== author) {
      face.dataset.name = author;
      face.replaceChildren(avatar(author, tileColor(author)));
    }
    setText(q(".feed__who"), author);
    const when = q<HTMLTimeElement>(".feed__when");
    setText(when, formatRelative(it.created_at));
    setAttr(when, "title", formatDateTime(it.created_at));
    setText(q(".feed__title"), it.title || t("Untitled"));
    const sig = `${it.updated_at}|${it.html.length}`;
    if (c.sig !== sig) {
      c.sig = sig;
      const doc = q<HTMLElement>(".feed__doc");
      doc.innerHTML = it.html;
      for (const a of Array.from(doc.querySelectorAll<HTMLElement>("[contenteditable]"))) a.removeAttribute("contenteditable");
      if (c.el.isConnected) measure(id);
    }
    reactions(q(".feed__reacts"), id, it.reactions);
    setText(q(".feed__talkn"), it.comments === 1 ? t("1 comment") : it.comments ? t("{n} comments", { n: it.comments }) : t("Comment"));
  };

  const measure = (id: string) => {
    const c = cards.get(id);
    const it = items.get(id);
    if (!c || !it) return;
    const doc = c.el.querySelector(".feed__doc") as HTMLElement;
    const expand = c.el.querySelector(".feed__expand") as HTMLButtonElement;
    const long = doc.scrollHeight > doc.clientHeight + 4 || !!it.clipped;
    expand.hidden = !long;
    if (long && c.el.dataset.expanded !== "true") setText(expand, t("Show more"));
  };

  const paint = () => {
    if (destroyed) return;
    reconcile(list, order, (id) => id, makeCard, (_el, id) => paintCard(id));
    for (const [id, c] of cards) {
      if (!order.includes(id)) {
        c.unwatch?.();
        cards.delete(id);
      }
    }
    empty.hidden = order.length > 0;
    more.hidden = order.length >= total;
    if (!more.hidden) setText(more, t("Load more ({n} more)", { n: total - order.length }));
    setAttr(list, "aria-busy", "false");
  };

  const load = async () => {
    const mine = ++seq;
    const r = await fetchPage(0, PAGE);
    if (mine !== seq || destroyed) return;
    items.clear();
    for (const it of r.items) items.set(it.id, it);
    order = r.items.map((x) => x.id);
    total = r.total;
    paint();
  };
  const refresh = async () => {
    const mine = ++seq;
    const r = await fetchPage(0, Math.min(25, Math.max(PAGE, order.length)));
    if (mine !== seq || destroyed) return;
    const keep = order.slice(r.items.length);
    for (const it of r.items) items.set(it.id, it);
    order = [...r.items.map((x) => x.id), ...keep.filter((id) => !r.items.some((x) => x.id === id))];
    total = r.total;
    paint();
  };
  more.addEventListener("click", async () => {
    const mine = ++seq;
    const r = await fetchPage(order.length, PAGE).catch((e: unknown) => (engine.reportError(e), null));
    if (!r || mine !== seq || destroyed) return;
    for (const it of r.items) {
      items.set(it.id, it);
      if (!order.includes(it.id)) order.push(it.id);
    }
    total = r.total;
    paint();
  });
  const offComments = engine.comments.subscribe((pageId) => {
    if (cards.has(pageId)) paintComments(pageId);
  });

  const ready = load().catch((e: unknown) => {
    engine.reportError(e);
    setAttr(list, "aria-busy", "false");
  });

  return {
    el,
    ready,
    ctx,
    configChanged(prev, next) {
      if (configKey(prev.config) !== configKey(next.config)) void load().catch(engine.reportError);
    },
    schemaChanged: () => void refresh().catch(engine.reportError),
    setSearch: () => void load().catch(engine.reportError),
    destroy() {
      destroyed = true;
      sizes.disconnect();
      offComments();
      for (const c of cards.values()) c.unwatch?.();
      session.close();
      el.remove();
    },
  };
}
