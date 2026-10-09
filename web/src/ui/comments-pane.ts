import type { CommentThread, Engine } from "../engine";
import { t } from "../engine/i18n";
import { editorHost } from "../extensions/blocks";
import { revealThread } from "../extensions/blocks/comments";
import { reconcileThreads, type ThreadView } from "./comment-view";
import { byId, h, ref, setAttr, setText } from "./dom";
import { iconNode } from "./icons";

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
  const list = h("div", { class: "cpane__list" });
  const empty = h("p", { class: "cpane__empty" });
  const pane = h(
    "aside",
    { class: "cpane", attrs: { "aria-label": t("Comments"), tabindex: "-1" } },
    h("header", { class: "cpane__head" }, h("h2", { class: "cpane__title", text: t("Comments") }), close),
    h("div", { class: "cpane__tools" }, tabs, people),
    list,
    empty,
  );
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
    const n = id ? engine.comments.get(id).threads.filter((x) => !x.resolved_at).length : 0;
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
    empty.hidden = shown.length > 0;
    setText(empty, filter === "open" ? (listing.can_comment ? t("No open comments. Select text and choose Comment to start one.") : t("No open comments.")) : t("Nothing resolved yet."));
  };

  const follow = () => {
    const id = open ? currentPage() : "";
    if (id === pageId) return render();
    unwatch?.();
    unwatch = null;
    pageId = id;
    list.replaceChildren();
    views.clear();
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

  engine.comments.subscribe((id) => {
    if (id === pageId || id === currentPage()) render();
  });
  state.watch("route", () => {
    btn.hidden = !currentPage();
    if (open) follow();
    else paintCount();
  });
  btn.hidden = !currentPage();
  paintCount();

  commands.register({
    id: "page.comments",
    label: () => t("Show or hide comments"),
    group: () => t("Pages"),
    enabled: () => !!currentPage(),
    run: () => setOpen(!open),
  });
  keymap.register({ id: "page.comments", command: "page.comments", chords: ["mod+alt+m"], scope: "global", whileTyping: true });
}
