import type { Engine, TrashItem } from "../engine";
import { formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { confirmDialog } from "./dialog";
import { h, ref, setAttr, setText } from "./dom";
import { errorMessage } from "./errors";
import { button } from "./form";
import { iconNode } from "./icons";
import { fallbackGlyph, pageIcon, paintIcon } from "./pageicon";
import { toast } from "./toast";

export function mountTrashView(engine: Engine): void {
  const { state, pages } = engine;
  const view = ref("view-trash");

  const search = h("input", { class: "input", attrs: { type: "search", placeholder: t("Find in trash"), "aria-label": t("Find in trash"), autocomplete: "off", spellcheck: "false" } });
  const emptyBtn = button(t("Empty trash"), "danger", "trash");
  const lead = h("p", { class: "section__lead", text: t("Pages you trash wait here. Restore brings back everything that was trashed together.") });
  const list = h("div", { class: "trash-list", attrs: { role: "list" } });
  const empty = h("p", { class: "lib-empty" });
  view.append(
    h(
      "div",
      { class: "lib" },
      h("h1", { class: "lib-title", text: t("Trash") }),
      lead,
      h("div", { class: "lib-toolbar" }, h("span", { class: "search-field lib-search-field" }, iconNode("search"), search), emptyBtn),
      list,
      empty,
    ),
  );

  let items: TrashItem[] = [];
  let seq = 0;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let loadedFor = "";

  const load = async () => {
    const ws = state.get("workspaceId");
    if (!ws) return;
    const mine = ++seq;
    try {
      const r = await engine.api.workspaces.trash(ws, search.value.trim());
      if (mine !== seq) return;
      items = r.items;
      loadedFor = ws;
      paint();
    } catch (e) {
      engine.reportError(e);
    }
  };

  const paint = () => {
    const w = engine.workspace();
    const writable = engine.canWrite();
    emptyBtn.hidden = !writable || !(w?.role === "admin" || w?.personal) || items.length === 0 || !!search.value.trim();
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(list.children) as HTMLElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = list.firstElementChild;
    for (const it of items) {
      let el = existing.get(it.id);
      if (el) existing.delete(it.id);
      else {
        el = h(
          "div",
          { class: "trash-row", attrs: { role: "listitem", "data-id": it.id } },
          pageIcon(it.icon, fallbackGlyph(it.kind)),
          h("span", { class: "trash-row__text" }, h("span", { class: "trash-row__title" }), h("span", { class: "trash-row__meta" })),
          h("span", { class: "trash-row__actions" }, button(t("Restore"), "ghost", "restore"), button(t("Delete forever"), "ghost", "trash")),
        );
        el.dataset.key = it.id;
        const [restore, del] = el.querySelectorAll<HTMLButtonElement>(".trash-row__actions .btn");
        restore?.setAttribute("data-action", "restore");
        del?.setAttribute("data-action", "delete");
      }
      paintIcon(el.querySelector(".page-icon") as HTMLElement, it.icon, fallbackGlyph(it.kind));
      const title = el.querySelector(".trash-row__title") as HTMLElement;
      setText(title, it.title || t("Untitled"));
      setAttr(title, "data-untitled", it.title ? null : "true");
      const who = it.trashed_by_name ? t("Deleted by {name}, {when}", { name: it.trashed_by_name, when: formatRelative(it.trashed_at ?? 0) }) : t("Deleted {when}", { when: formatRelative(it.trashed_at ?? 0) });
      const inside = it.pages > 1 ? ` · ${plural(it.pages - 1, "{count} page inside", "{count} pages inside")}` : "";
      setText(el.querySelector(".trash-row__meta") as HTMLElement, who + inside);
      for (const b of el.querySelectorAll<HTMLButtonElement>(".trash-row__actions .btn")) b.hidden = !writable;
      if (el !== cursor) list.insertBefore(el, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
    setText(empty, search.value.trim() ? t("No trashed page matches that.") : t("The trash is empty."));
    empty.hidden = items.length > 0 || loadedFor !== state.get("workspaceId");
  };

  list.addEventListener("click", (e) => {
    const action = (e.target as HTMLElement).closest<HTMLElement>("[data-action]")?.dataset.action;
    const id = (e.target as HTMLElement).closest<HTMLElement>(".trash-row")?.dataset.id;
    const item = items.find((x) => x.id === id);
    if (!action || !item) return;
    if (action === "restore") {
      items = items.filter((x) => x.id !== item.id);
      paint();
      void pages
        .restore(item.id)
        .then(() => toast({ tone: "success", message: t("Restored “{title}”", { title: item.title || t("Untitled") }), action: { label: t("Open"), run: () => engine.openPage(item.id) } }))
        .catch((err: unknown) => {
          toast({ tone: "danger", message: errorMessage(err) });
          void load();
        });
      return;
    }
    void confirmDialog({
      title: t("Delete forever?"),
      body: item.pages > 1 ? t("“{title}” and {count} pages inside will be deleted for good. This cannot be undone.", { title: item.title || t("Untitled"), count: item.pages - 1 }) : t("“{title}” will be deleted for good. This cannot be undone.", { title: item.title || t("Untitled") }),
      confirm: t("Delete forever"),
      variant: "danger",
    }).then((ok) => {
      if (!ok) return;
      items = items.filter((x) => x.id !== item.id);
      paint();
      engine.api.pages.deleteForever(item.id).catch((err: unknown) => {
        toast({ tone: "danger", message: errorMessage(err) });
        void load();
      });
    });
  });

  emptyBtn.addEventListener("click", () => {
    const ws = state.get("workspaceId");
    void confirmDialog({
      title: t("Empty the trash?"),
      body: t("Everything in this workspace's trash will be deleted for good. This cannot be undone."),
      confirm: t("Empty trash"),
      variant: "danger",
    }).then((ok) => {
      if (!ok) return;
      items = [];
      paint();
      engine.api.workspaces
        .emptyTrash(ws)
        .then((r) => toast({ tone: "success", message: plural(r.deleted, "Deleted {count} page for good", "Deleted {count} pages for good") }))
        .catch((err: unknown) => {
          toast({ tone: "danger", message: errorMessage(err) });
          void load();
        });
    });
  });

  search.addEventListener("input", () => {
    clearTimeout(timer);
    timer = setTimeout(() => void load(), 200);
  });

  state.watch("route", (r) => {
    if (r.name === "trash") void load();
  });
  state.watch("workspaceId", () => {
    if (state.get("route").name === "trash") void load();
  });
  state.watch("me", () => {
    if (state.get("route").name === "trash") paint();
  });
  pages.subscribe((c) => {
    if (state.get("route").name !== "trash") return;
    if ((c.kind === "trash" && c.workspaceId === state.get("workspaceId")) || c.kind === "remove") {
      clearTimeout(timer);
      timer = setTimeout(() => void load(), 120);
    }
  });
  if (state.get("route").name === "trash") void load();
}
