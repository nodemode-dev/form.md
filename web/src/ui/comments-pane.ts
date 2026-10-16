import type { CommentThread, Engine, Suggestion } from "../engine";
import { t } from "../engine/i18n";
import { editorHost } from "../extensions/blocks";
import { revealThread } from "../extensions/blocks/comments";
import { suggestionRange, suggestUI } from "../extensions/blocks/suggest";
import { reconcileThreads, type SuggestionView, suggestionView, type ThreadView } from "./comment-view";
import { byId, h, ref, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { reconcile } from "./list";

type Filter = "open" | "resolved";

export function mountCommentsPane(engine: Engine): void {
  const { state, commands, keymap } = engine;
  const btn = ref<HTMLButtonElement>("act-comments");
  const count = ref("act-comments-count");
  const close = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Close comments"), "data-tip": t("Close comments") } }, iconNode("close"));
  const tabs = h("div", { class: "segmented cpane__filter", attrs: { role: "group", "aria-label": t("Show") } });
  const tabBtn = (f: Filter, label: string) => {
    const b = h("button", { class: "segmented__btn segmented__btn--text", attrs: { type: "button", "data-filter": f, "aria-pressed": f === "open" ? "true" : "false" }, text: label });
    tabs.append(b);
    return b;
  };
  tabBtn("open", t("Open"));
  tabBtn("resolved", t("Resolved"));
  const people = h("select", { class: "select cpane__people", attrs: { "aria-label": t("Filter by person") } });
  const slist = h("div", { class: "cpane__suggestions" });
  const list = h("div", { class: "cpane__threads" });
  const empty = h("p", { class: "cpane__empty" });
  const pane = h(
    "aside",
    { class: "cpane", attrs: { "aria-label": t("Comments"), tabindex: "-1" } },
    h("header", { class: "cpane__head" }, h("h2", { class: "cpane__title", text: t("Comments") }), close),
    h("div", { class: "cpane__tools" }, tabs, people),
    h("div", { class: "cpane__list" }, slist, list),
    empty,
  );
  const suggestionViews = new Map<string, SuggestionView>();
  const decide = (s: Suggestion, accept: boolean) => {
    const v = editorHost()?.view();
    const ui = v ? suggestUI(v) : undefined;
    if (ui) void ui.decide(s, accept);
    else if (!accept) void engine.comments.run(s.page_id, () => engine.api.comments.decide(s.id, false)).catch(engine.reportError);
  };
  const jumpTo = (s: Suggestion) => {
    const v = editorHost()?.view();
    const r = v ? suggestionRange(v.state, s) : null;
    if (!v || !r) return;
    const dom = v.domAtPos(r.from).node;
    (dom instanceof Element ? dom : dom.parentElement)?.scrollIntoView({ block: "center", behavior: "smooth" });
  };
  pane.hidden = true;
  byId("overlay-root").append(pane);
  const views = new Map<string, ThreadView>();
  let filter: Filter = "open";
  let open = false;
  let pageId = "";
  let unwatch: (() => void) | null = null;

  const currentPage = () => {
    const r = state.get("route");
    return r.name === "page" ? r.id : "";
  };
  const workspaceId = () => engine.pages.get(pageId)?.workspace_id ?? state.get("workspaceId");

  const jump = (th: CommentThread) => {
    const v = editorHost()?.view();
    if (v) revealThread(v, th.id);
  };

  const paintCount = () => {
    const id = currentPage();
    const l = id ? engine.comments.get(id) : null;
    const n = l ? l.threads.filter((x) => !x.resolved_at).length + l.suggestions.filter((x) => x.status === "open").length : 0;
    setText(count, n ? String(n) : "");
    setAttr(btn, "aria-pressed", open ? "true" : "false");
    const label = n ? t("Comments, {count} open", { count: n }) : t("Comments");
    setAttr(btn, "aria-label", label);
    setAttr(btn, "data-tip", label);
  };

  const render = () => {
    paintCount();
    if (!open || !pageId) return;
    const listing = engine.comments.get(pageId);
    const authors = new Map<string, string>();
    for (const th of listing.threads) for (const c of th.comments) if (c.author_id) authors.set(c.author_id, c.author_name);
    for (const s of listing.suggestions) if (s.author_id) authors.set(s.author_id, s.author_name);
    const want = [["", t("Everyone")], ...[...authors].sort((a, b) => a[1].localeCompare(b[1]))];
    const keep = people.value;
    if (people.options.length !== want.length || want.some(([v], i) => people.options[i]?.value !== v)) {
      people.replaceChildren(...want.map(([v, label]) => h("option", { attrs: { value: v ?? "" }, text: label ?? "" })));
      people.value = authors.has(keep) ? keep : "";
    }
    const who = people.value;
    const shown = listing.threads
      .filter((x) => (filter === "open" ? !x.resolved_at : !!x.resolved_at))
      .filter((x) => !who || x.comments.some((c) => c.author_id === who))
      .sort((a, b) => (a.anchor === "page" ? 0 : 1) - (b.anchor === "page" ? 0 : 1) || a.created_at - b.created_at);
    reconcileThreads(engine, pageId, list, shown, listing, { workspaceId, onJump: jump }, views);
    const sugg = listing.suggestions.filter((x) => (filter === "open" ? x.status === "open" : x.status !== "open")).filter((x) => !who || x.author_id === who);
    reconcile(
      slist,
      sugg,
      (x) => x.id,
      (x) => {
        const v = suggestionView(engine, pageId, x, { onDecide: decide, onJump: jumpTo });
        suggestionViews.set(x.id, v);
        return v.el;
      },
      (_el, x) => suggestionViews.get(x.id)?.update(x, listing),
    );
    for (const id of [...suggestionViews.keys()]) if (!sugg.some((x) => x.id === id)) suggestionViews.delete(id);
    empty.hidden = shown.length > 0 || sugg.length > 0;
    setText(empty, filter === "open" ? (listing.can_comment ? t("No open comments. Select text and choose Comment to start one.") : t("No open comments.")) : t("Nothing resolved yet."));
  };

  const follow = () => {
    const id = open ? currentPage() : "";
    if (id === pageId) return render();
    unwatch?.();
    unwatch = null;
    pageId = id;
    list.replaceChildren();
    slist.replaceChildren();
    views.clear();
    suggestionViews.clear();
    if (id) {
      unwatch = engine.comments.watch(id);
      if (!engine.comments.has(id)) void engine.comments.load(id).catch(() => undefined);
    }
    render();
  };

  const setOpen = (next: boolean) => {
    if (open === next) return;
    open = next;
    pane.hidden = !open;
    setAttr(document.documentElement, "data-comments", open ? "open" : null);
    follow();
    if (open) requestAnimationFrame(() => pane.classList.add("is-visible"));
    else pane.classList.remove("is-visible");
  };

  btn.addEventListener("click", () => setOpen(!open));
  close.addEventListener("click", () => setOpen(false));
  tabs.addEventListener("click", (e) => {
    const f = (e.target as Element).closest<HTMLElement>("[data-filter]")?.dataset.filter as Filter | undefined;
    if (!f || f === filter) return;
    filter = f;
    for (const b of Array.from(tabs.children) as HTMLElement[]) setAttr(b, "aria-pressed", b.dataset.filter === f ? "true" : "false");
    render();
  });
  people.addEventListener("change", render);
  pane.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !(e.target as Element).closest(".composer")) {
      setOpen(false);
      btn.focus();
    }
  });

  let target: { page: string; thread: string } | null = null;
  const fromUrl = new URLSearchParams(location.search).get("thread");
  if (fromUrl && currentPage()) target = { page: currentPage(), thread: fromUrl };
  const aim = () => {
    if (!target || target.page !== currentPage()) return;
    if (!open) setOpen(true);
    const th = engine.comments.get(target.page).threads.find((x) => x.id === target?.thread);
    if (!th) return;
    if (th.resolved_at && filter === "open") {
      filter = "resolved";
      for (const b of Array.from(tabs.children) as HTMLElement[]) setAttr(b, "aria-pressed", b.dataset.filter === filter ? "true" : "false");
      render();
    }
    const el = list.querySelector<HTMLElement>(`.cthread[data-thread="${th.id}"]`);
    if (!el) return;
    target = null;
    el.scrollIntoView({ block: "nearest" });
    el.classList.add("is-target");
    setTimeout(() => el.classList.remove("is-target"), 2400);
    if (th.anchor !== "page") jump(th);
  };
  window.addEventListener("formmd:thread", (e) => {
    const d = (e as CustomEvent<{ page: string; thread: string }>).detail;
    target = d;
    requestAnimationFrame(aim);
  });
  engine.comments.subscribe((id) => {
    if (id === pageId || id === currentPage()) render();
    if (target && id === target.page) aim();
  });
  editorHost()?.onShown(() => requestAnimationFrame(aim));
  state.watch("route", () => {
    btn.hidden = !currentPage();
    if (open) follow();
    else paintCount();
    aim();
  });
  btn.hidden = !currentPage();
  paintCount();

  commands.register({
    id: "page.comments",
    label: () => t("Show or hide comments"),
    group: () => t("Pages"),
    description: "Shows or hides the comments pane for the open page. Does not change any comments. Needs a page to be open.",
    mutates: false,
    enabled: () => !!currentPage(),
    run: () => setOpen(!open),
  });
  keymap.register({ id: "page.comments", command: "page.comments", chords: ["mod+alt+m"], scope: "global", whileTyping: true });
}
