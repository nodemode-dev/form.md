import type { Engine, Page, StoreChange, Teamspace } from "../engine";
import { formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { byId, h, ref, setAttr, setText } from "./dom";
import { errorMessage } from "./errors";
import { type IconName, iconNode } from "./icons";
import { type MenuEntry, openMenu } from "./menu";
import { fallbackGlyph, pageIcon, paintIcon } from "./pageicon";
import { pageMenu } from "./page-actions";
import { openTeamspaceDialog } from "./teamspace-dialog";
import { toast } from "./toast";

type Zone = "before" | "after" | "inside";

interface Section {
  el: HTMLElement;
  body: HTMLElement;
  count: HTMLElement;
  add: HTMLButtonElement | null;
}

const SECTION_KEY = (id: string) => `formmd.section.${id}`;
const EXPANDED_KEY = (ws: string) => `formmd.expanded.${ws}`;

function readSet(key: string): Set<string> {
  try {
    const v = JSON.parse(localStorage.getItem(key) ?? "[]") as unknown;
    return new Set(Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : []);
  } catch {
    return new Set();
  }
}

function writeSet(key: string, set: Set<string>): void {
  try {
    localStorage.setItem(key, JSON.stringify([...set]));
  } catch {
    return;
  }
}

function sectionOpen(id: string): boolean {
  try {
    return localStorage.getItem(SECTION_KEY(id)) !== "0";
  } catch {
    return true;
  }
}

function makeSection(id: string, label: string, icon: IconName, addLabel = ""): Section {
  const count = h("span", { class: "sb-count" });
  const toggle = h(
    "button",
    { class: "sb-toggle", attrs: { type: "button", "aria-expanded": sectionOpen(id) ? "true" : "false" } },
    h("span", { class: "sb-icon" }, iconNode(icon)),
    h("span", { class: "sb-label", text: label }),
    count,
    h("span", { class: "sb-chev" }, iconNode("chevron-right")),
  );
  const head = h("div", { class: "sb-head" }, toggle);
  let add: HTMLButtonElement | null = null;
  if (addLabel) {
    add = h("button", { class: "icon-btn sb-add", attrs: { type: "button", "aria-label": addLabel, "data-tip": addLabel } }, iconNode("plus"));
    head.append(add);
  }
  const body = h("div", { class: "sb-body" });
  const el = h("section", { class: "sb-section", attrs: { "data-section": id, "data-open": sectionOpen(id) ? "true" : "false" } }, head, body);
  toggle.addEventListener("click", () => {
    const open = el.dataset.open !== "true";
    setAttr(el, "data-open", open ? "true" : "false");
    setAttr(toggle, "aria-expanded", open ? "true" : "false");
    try {
      localStorage.setItem(SECTION_KEY(id), open ? "1" : "0");
    } catch {
      return;
    }
  });
  return { el, body, count, add };
}

export function tagColorFor(engine: Engine, workspaceId: string, tag: string): string {
  const set = engine.pages.tagColor(workspaceId, tag);
  if (set) return set;
  let hash = 0;
  for (const ch of tag.toLowerCase()) hash = (Math.imul(hash, 31) + (ch.codePointAt(0) ?? 0)) >>> 0;
  const palette = ["violet", "blue", "cyan", "teal", "green", "lime", "amber", "orange", "rose", "pink"];
  return palette[hash % palette.length] ?? "violet";
}

export function mountTree(engine: Engine): void {
  const { state, pages } = engine;
  const nav = ref("nav");

  const favs = makeSection("favorites", t("Favorites"), "star");
  const tags = makeSection("tags", t("Tags"), "tag");
  const tree = makeSection("pages", t("Pages"), "page", t("New page"));
  const notes = makeSection("notes", t("Notes"), "cards", t("New page"));
  const teams = makeSection("teams", t("Teamspaces"), "users", t("New teamspace"));
  const shared = makeSection("shared", t("Shared with me"), "users");
  const favList = h("ul", { class: "sb-list", attrs: { role: "list" } });
  const cloud = h("div", { class: "tag-cloud" });
  const root = h("ul", { class: "tree", attrs: { role: "tree", "aria-label": t("Pages") } });
  const teamRoot = h("ul", { class: "tree", attrs: { role: "tree", "aria-label": t("Teamspaces") } });
  const sharedRoot = h("ul", { class: "tree", attrs: { role: "tree", "aria-label": t("Shared with me") } });
  const empty = h("p", { class: "sb-empty", text: t("Empty for now") });
  const filterLabel = h("span", { class: "sb-filter__tag" });
  const filterClear = h("button", { class: "link-btn", attrs: { type: "button" }, text: t("Clear filter") });
  const filterList = h("ul", { class: "sb-list", attrs: { role: "list" } });
  const filterEmpty = h("p", { class: "sb-empty", text: t("Nothing carries this tag.") });
  const filter = h("div", { class: "sb-filter" }, h("div", { class: "sb-filter__bar" }, iconNode("filter"), filterLabel, filterClear), filterList, filterEmpty);
  const noteChips = h("div", { class: "notes-filters" });
  const noteList = h("ul", { class: "notes-list", attrs: { role: "list" } });
  const noteEmpty = h("p", { class: "sb-empty" });
  const inboxCount = h("span", { class: "sb-count sb-count--badge" });
  const inboxRow = h(
    "a",
    { class: "sb-trash sb-inbox", attrs: { href: "/inbox" } },
    h("span", { class: "sb-icon" }, iconNode("inbox")),
    h("span", { class: "sb-label", text: t("Inbox") }),
    inboxCount,
  );
  const trashCount = h("span", { class: "sb-count" });
  const trashRow = h(
    "a",
    { class: "sb-trash", attrs: { href: "/trash" } },
    h("span", { class: "sb-icon" }, iconNode("trash")),
    h("span", { class: "sb-label", text: t("Trash") }),
    trashCount,
  );
  favs.body.append(favList);
  tags.body.append(cloud);
  tree.body.append(root, empty);
  teams.body.append(teamRoot);
  shared.body.append(sharedRoot);
  notes.body.append(noteChips, noteList, noteEmpty);
  const indicator = h("div", { class: "drop-indicator", attrs: { "aria-hidden": "true" } });
  indicator.hidden = true;
  nav.append(inboxRow, favs.el, tags.el, filter, notes.el, teams.el, tree.el, shared.el, trashRow, indicator);

  const nodes = new Map<string, HTMLLIElement>();
  const teamNodes = new Map<string, HTMLLIElement>();
  const teamKey = (id: string) => `team:${id}`;
  let ws = state.get("workspaceId");
  let expanded = readSet(EXPANDED_KEY(ws));
  let activeId = "";
  let notesUntagged = false;

  const writable = () => engine.canWrite(ws);
  const activePage = () => {
    const r = state.get("route");
    return r.name === "page" ? r.id : "";
  };

  const row = (li: Element) => li.firstElementChild as HTMLElement;

  const patchRow = (li: HTMLLIElement, p: Page, depth: number) => {
    const r = row(li);
    r.style.setProperty("--depth", String(depth));
    setAttr(r, "aria-level", String(depth + 1));
    const has = pages.hasChildren(p);
    const open = has && expanded.has(p.id);
    setAttr(r, "data-has-children", has ? "true" : null);
    setAttr(r, "aria-expanded", has ? (open ? "true" : "false") : null);
    setAttr(r, "aria-current", p.id === activeId ? "page" : null);
    setAttr(r, "data-private", p.visibility === "private" ? "true" : null);
    paintIcon(r.querySelector<HTMLElement>(".page-icon") as HTMLElement, p.icon, fallbackGlyph(p.kind));
    const title = r.querySelector(".tree-title") as HTMLElement;
    setText(title, p.title || t("Untitled"));
    setAttr(title, "data-untitled", p.title ? null : "true");
  };

  const createNode = (p: Page): HTMLLIElement => {
    const chev = h("span", { class: "tree-chev", attrs: { "data-action": "toggle", "aria-hidden": "true" } }, iconNode("chevron-right"));
    const actions = h(
      "span",
      { class: "tree-actions" },
      h("button", { class: "tree-act", attrs: { type: "button", tabindex: "-1", "data-action": "add", "aria-label": t("Add a page inside"), "data-tip": t("Add a page inside") } }, iconNode("plus")),
      h("button", { class: "tree-act", attrs: { type: "button", tabindex: "-1", "data-action": "more", "aria-label": t("More"), "data-tip": t("More") } }, iconNode("more")),
    );
    const r = h(
      "div",
      { class: "tree-row", attrs: { role: "treeitem", tabindex: "-1", "data-id": p.id } },
      chev,
      pageIcon(p.icon, fallbackGlyph(p.kind)),
      h("span", { class: "tree-title" }),
      actions,
    );
    const li = h("li", { class: "tree-node", attrs: { role: "none" } }, r);
    nodes.set(p.id, li);
    return li;
  };

  const renderList = (container: HTMLElement, parentId: string, depth: number, list = pages.children(ws, parentId)) => {
    const existing = new Map<string, HTMLLIElement>();
    for (const child of Array.from(container.children) as HTMLLIElement[]) {
      const k = child.dataset.key;
      if (k !== undefined) existing.set(k, child);
    }
    let prev: Element | null = null;
    for (const p of list) {
      let li = existing.get(p.id);
      if (li) existing.delete(p.id);
      else {
        li = nodes.get(p.id);
        if (!li) li = createNode(p);
        li.dataset.key = p.id;
      }
      patchRow(li, p, depth);
      const cursor: Element | null = prev ? prev.nextElementSibling : container.firstElementChild;
      if (li !== cursor) container.insertBefore(li, cursor);
      prev = li;
      syncChildren(li, p, depth);
    }
    for (const [id, el] of existing) {
      if (el.parentNode === container) el.remove();
      if (!pages.get(id)) nodes.delete(id);
    }
  };

  const syncChildren = (li: HTMLLIElement, p: Page, depth: number) => {
    const open = expanded.has(p.id) && pages.hasChildren(p);
    let group = li.children[1] as HTMLElement | undefined;
    if (!open) {
      group?.remove();
      return;
    }
    if (!group) {
      group = h("ul", { class: "tree-group", attrs: { role: "group" } });
      li.append(group);
    }
    renderList(group, p.id, depth + 1);
  };

  const depthOf = (li: HTMLElement) => Number(row(li).getAttribute("aria-level") ?? "1") - 1;

  const filtering = () => !!state.get("me")?.user.prefs.notes_mode || !!state.get("tagFilter");
  const teamWritable = (team: Teamspace) => writable() && (team.access === "open" || team.role !== "");

  const createTeamNode = (team: Teamspace): HTMLLIElement => {
    const r = h(
      "div",
      { class: "tree-row team-row", attrs: { role: "treeitem", tabindex: "-1", "data-team": team.id } },
      h("span", { class: "tree-chev", attrs: { "data-action": "toggle", "aria-hidden": "true" } }, iconNode("chevron-right")),
      pageIcon(team.icon, "users"),
      h("span", { class: "tree-title" }),
      h(
        "span",
        { class: "tree-actions" },
        h("button", { class: "tree-act", attrs: { type: "button", tabindex: "-1", "data-action": "team-add", "aria-label": t("Add a page"), "data-tip": t("Add a page") } }, iconNode("plus")),
        h("button", { class: "tree-act", attrs: { type: "button", tabindex: "-1", "data-action": "team-more", "aria-label": t("Teamspace options"), "data-tip": t("Teamspace options") } }, iconNode("more")),
      ),
    );
    const li = h("li", { class: "tree-node", attrs: { role: "none" } }, r);
    li.dataset.key = teamKey(team.id);
    teamNodes.set(team.id, li);
    return li;
  };

  const syncTeam = (li: HTMLLIElement, team: Teamspace) => {
    const r = row(li);
    const open = expanded.has(teamKey(team.id));
    r.style.setProperty("--depth", "0");
    setAttr(r, "aria-level", "1");
    setAttr(r, "data-has-children", "true");
    setAttr(r, "aria-expanded", open ? "true" : "false");
    setAttr(r, "data-access", team.access);
    setAttr(r, "data-joined", team.role ? "true" : "false");
    paintIcon(r.querySelector<HTMLElement>(".page-icon") as HTMLElement, team.icon, "users");
    setText(r.querySelector(".tree-title") as HTMLElement, team.name);
    const add = r.querySelector<HTMLElement>('[data-action="team-add"]');
    if (add) add.hidden = !teamWritable(team);
    let group = li.children[1] as HTMLElement | undefined;
    if (!open) {
      group?.remove();
      return;
    }
    if (!group) {
      group = h("ul", { class: "tree-group", attrs: { role: "group" } });
      li.append(group);
    }
    renderList(group, "", 1, pages.roots(ws, team.id));
  };

  const renderTeams = () => {
    const list = pages.teamspaces(ws);
    teams.el.hidden = filtering() || (list.length === 0 && !writable());
    setText(teams.count, list.length ? String(list.length) : "");
    if (teams.add) teams.add.hidden = !writable();
    const existing = new Map<string, HTMLLIElement>();
    for (const child of Array.from(teamRoot.children) as HTMLLIElement[]) if (child.dataset.key) existing.set(child.dataset.key, child);
    let cursor: Element | null = teamRoot.firstElementChild;
    for (const team of list) {
      let li = existing.get(teamKey(team.id));
      if (li) existing.delete(teamKey(team.id));
      else li = teamNodes.get(team.id) ?? createTeamNode(team);
      syncTeam(li, team);
      if (li !== cursor) teamRoot.insertBefore(li, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const [k, el] of existing) {
      el.remove();
      teamNodes.delete(k.slice(5));
    }
  };

  const renderShared = () => {
    const list = pages.shared(ws);
    shared.el.hidden = filtering() || list.length === 0;
    setText(shared.count, list.length ? String(list.length) : "");
    renderList(sharedRoot, "", 0, list);
  };

  const toggleTeam = (id: string, open?: boolean) => {
    const k = teamKey(id);
    const want = open ?? !expanded.has(k);
    if (want === expanded.has(k)) return false;
    if (want) expanded.add(k);
    else expanded.delete(k);
    writeSet(EXPANDED_KEY(ws), expanded);
    renderTeams();
    return true;
  };

  const renderRoot = () => {
    renderList(root, "", 0, pages.roots(ws, ""));
    renderTeams();
    renderShared();
    const n = pages.all(ws).length;
    setText(tree.count, n ? String(n) : "");
    empty.hidden = !pages.isLoaded(ws) || root.childElementCount > 0;
    ensureRoving();
  };

  const renderParent = (parentId: string) => {
    if (!parentId) return renderRoot();
    const li = nodes.get(parentId);
    const p = pages.get(parentId);
    if (!li?.isConnected || !p) return;
    const depth = depthOf(li);
    patchRow(li, p, depth);
    syncChildren(li, p, depth);
  };

  const flatRow = (p: Page, kind: "fav" | "flat"): HTMLLIElement => {
    const r = h(
      "div",
      { class: "flat-row", attrs: { tabindex: "-1", "data-id": p.id, "data-kind": kind } },
      pageIcon(p.icon, fallbackGlyph(p.kind)),
      h("span", { class: "flat-row__text" }, h("span", { class: "tree-title" }), h("span", { class: "flat-row__hint" })),
    );
    if (kind === "fav") r.append(h("button", { class: "tree-act", attrs: { type: "button", tabindex: "-1", "data-action": "unfav", "aria-label": t("Remove from favorites"), "data-tip": t("Remove from favorites") } }, iconNode("star-fill")));
    return h("li", { attrs: { role: "listitem" } }, r);
  };

  const patchFlat = (li: HTMLLIElement, p: Page, hint: string) => {
    const r = li.firstElementChild as HTMLElement;
    paintIcon(r.querySelector(".page-icon") as HTMLElement, p.icon, fallbackGlyph(p.kind));
    const title = r.querySelector(".tree-title") as HTMLElement;
    setText(title, p.title || t("Untitled"));
    setAttr(title, "data-untitled", p.title ? null : "true");
    setText(r.querySelector(".flat-row__hint") as HTMLElement, hint);
    setAttr(r, "aria-current", p.id === activeId ? "page" : null);
  };

  const reconcileFlat = (container: HTMLElement, list: Page[], kind: "fav" | "flat", hint: (p: Page) => string) => {
    const existing = new Map<string, HTMLLIElement>();
    for (const child of Array.from(container.children) as HTMLLIElement[]) if (child.dataset.key) existing.set(child.dataset.key, child);
    let cursor: Element | null = container.firstElementChild;
    for (const p of list) {
      let li = existing.get(p.id);
      if (li) existing.delete(p.id);
      else {
        li = flatRow(p, kind);
        li.dataset.key = p.id;
      }
      patchFlat(li, p, hint(p));
      if (li !== cursor) container.insertBefore(li, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
  };

  const parentHint = (p: Page) => {
    const parts: string[] = [];
    if (p.workspace_id !== ws) parts.push(engine.workspaceById(p.workspace_id)?.name ?? "");
    const parent = p.parent_id ? pages.get(p.parent_id) : undefined;
    if (parent) parts.push(parent.title || t("Untitled"));
    return parts.filter(Boolean).join(" / ");
  };

  const renderFavorites = () => {
    const list = pages.favorites();
    favs.el.hidden = list.length === 0;
    setText(favs.count, list.length ? String(list.length) : "");
    reconcileFlat(favList, list, "fav", parentHint);
  };

  const renderTags = () => {
    const counts = pages.tagCounts(ws);
    tags.el.hidden = counts.length === 0;
    setText(tags.count, counts.length ? String(counts.length) : "");
    const active = state.get("tagFilter").toLowerCase();
    const existing = new Map<string, HTMLButtonElement>();
    for (const child of Array.from(cloud.children) as HTMLButtonElement[]) if (child.dataset.key) existing.set(child.dataset.key, child);
    let cursor: Element | null = cloud.firstElementChild;
    for (const c of counts) {
      const k = c.tag.toLowerCase();
      let chip = existing.get(k);
      if (chip) existing.delete(k);
      else {
        chip = h("button", { class: "tag-chip", attrs: { type: "button", "data-tag": c.tag } }, h("span", { class: "tag-chip__name" }), h("span", { class: "tag-chip__count" }));
        chip.dataset.key = k;
      }
      chip.dataset.tag = c.tag;
      setText(chip.firstElementChild as HTMLElement, `#${c.tag}`);
      setText(chip.lastElementChild as HTMLElement, String(c.count));
      setAttr(chip, "data-color", tagColorFor(engine, ws, c.tag));
      setAttr(chip, "aria-pressed", k === active ? "true" : "false");
      setAttr(chip, "title", plural(c.count, "{count} page", "{count} pages"));
      if (chip !== cursor) cloud.insertBefore(chip, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
  };

  const tagged = (tag: string) =>
    pages
      .all(ws)
      .filter((p) => p.tags.some((x) => x.toLowerCase() === tag.toLowerCase()))
      .sort((a, b) => (a.title || "").localeCompare(b.title || ""));

  const renderFilter = () => {
    const tag = state.get("tagFilter");
    const notesMode = !!state.get("me")?.user.prefs.notes_mode;
    const on = !!tag && !notesMode;
    filter.hidden = !on;
    tree.el.hidden = on || notesMode || engine.workspaceById(ws)?.role === "guest";
    teams.el.hidden = on || notesMode || (pages.teamspaces(ws).length === 0 && !writable());
    shared.el.hidden = on || notesMode || pages.shared(ws).length === 0;
    notes.el.hidden = !notesMode;
    if (!on) return;
    setText(filterLabel, `#${tag}`);
    const list = tagged(tag);
    reconcileFlat(filterList, list, "flat", parentHint);
    filterEmpty.hidden = list.length > 0;
  };

  const renderNotes = () => {
    if (!state.get("me")?.user.prefs.notes_mode) return;
    const tag = state.get("tagFilter");
    let list = pages.all(ws).filter((p) => p.kind === "doc");
    if (tag) list = list.filter((p) => p.tags.some((x) => x.toLowerCase() === tag.toLowerCase()));
    else if (notesUntagged) list = list.filter((p) => p.tags.length === 0);
    list.sort((a, b) => b.updated_at - a.updated_at);
    setText(notes.count, String(list.length));
    const chips: [string, string, boolean][] = tag ? [["clear", `#${tag}`, true]] : [["all", t("All"), !notesUntagged], ["untagged", t("Untagged"), notesUntagged]];
    const existing = new Map<string, HTMLButtonElement>();
    for (const c of Array.from(noteChips.children) as HTMLButtonElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = noteChips.firstElementChild;
    for (const [key, label, on] of chips) {
      let b = existing.get(key);
      if (b) existing.delete(key);
      else {
        b = h("button", { class: "chip", attrs: { type: "button", "data-note-filter": key } });
        b.dataset.key = key;
      }
      setText(b, key === "clear" ? `${label} ×` : label);
      setAttr(b, "aria-pressed", on ? "true" : "false");
      if (b !== cursor) noteChips.insertBefore(b, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
    const cards = new Map<string, HTMLLIElement>();
    for (const c of Array.from(noteList.children) as HTMLLIElement[]) if (c.dataset.key) cards.set(c.dataset.key, c);
    cursor = noteList.firstElementChild;
    for (const p of list) {
      let li = cards.get(p.id);
      if (li) cards.delete(p.id);
      else {
        li = h(
          "li",
          { attrs: { role: "listitem" } },
          h(
            "div",
            { class: "note-card", attrs: { tabindex: "-1", "data-id": p.id } },
            h("span", { class: "note-card__main" }, h("span", { class: "note-card__title" }, pageIcon(p.icon, "page"), h("span", { class: "note-card__name" })), h("span", { class: "note-card__snippet" }), h("span", { class: "note-card__meta" })),
            h("img", { class: "note-card__thumb", attrs: { alt: "", loading: "lazy", decoding: "async" } }),
          ),
        );
        li.dataset.key = p.id;
      }
      const card = li.firstElementChild as HTMLElement;
      setAttr(card, "aria-current", p.id === activeId ? "page" : null);
      paintIcon(card.querySelector(".page-icon") as HTMLElement, p.icon, "page");
      setText(card.querySelector(".note-card__name") as HTMLElement, p.title || t("Untitled"));
      const snippet = card.querySelector(".note-card__snippet") as HTMLElement;
      setText(snippet, p.snippet);
      snippet.hidden = !p.snippet;
      const meta = card.querySelector(".note-card__meta") as HTMLElement;
      const metaText = [formatRelative(p.updated_at), ...p.tags.slice(0, 2).map((x) => `#${x}`)].join("  ");
      setText(meta, metaText);
      const thumb = card.querySelector(".note-card__thumb") as HTMLImageElement;
      thumb.hidden = !p.thumb;
      if (p.thumb && thumb.getAttribute("src") !== p.thumb) thumb.src = p.thumb;
      if (li !== cursor) noteList.insertBefore(li, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of cards.values()) el.remove();
    setText(noteEmpty, tag ? t("Nothing carries #{tag} yet.", { tag }) : notesUntagged ? t("Every note has a tag.") : t("Start with Alt+N."));
    noteEmpty.hidden = list.length > 0;
  };

  const paintWritable = () => {
    const w = writable();
    const split = engine.workspaceById(ws)?.sidebar_mode !== "tree";
    setText(tree.el.querySelector(".sb-label") as HTMLElement, split ? t("Documents") : t("Pages"));
    setAttr(nav, "data-readonly", w ? null : "true");
    if (tree.add) tree.add.hidden = !w;
    if (notes.add) notes.add.hidden = !w;
  };

  const paintTrash = () => {
    setAttr(trashRow, "aria-current", state.get("route").name === "trash" ? "page" : null);
    setAttr(inboxRow, "aria-current", state.get("route").name === "inbox" ? "page" : null);
    const n = engine.inbox.unread;
    setText(inboxCount, n ? (n > 99 ? "99+" : String(n)) : "");
    setAttr(inboxRow, "aria-label", n ? t("Inbox, {count} unread", { count: n }) : t("Inbox"));
  };
  engine.inbox.subscribe((c) => {
    if (c.kind !== "arrived") paintTrash();
  });

  const renderAll = () => {
    paintWritable();
    renderRoot();
    renderFavorites();
    renderTags();
    renderFilter();
    renderNotes();
    paintTrash();
  };

  const setActive = (id: string) => {
    if (activeId === id) return;
    const prev = activeId;
    activeId = id;
    for (const key of [prev, id]) {
      if (!key) continue;
      for (const el of nav.querySelectorAll<HTMLElement>(`[data-id="${key}"]`)) setAttr(el, "aria-current", key === activeId ? "page" : null);
    }
    revealActive();
  };

  const revealActive = () => {
    const id = activeId;
    const p = id ? pages.get(id) : undefined;
    if (p && p.workspace_id === ws) {
      let changed = false;
      for (const a of pages.ancestors(id)) {
        if (!expanded.has(a.id)) {
          expanded.add(a.id);
          changed = true;
        }
      }
      if (changed) {
        writeSet(EXPANDED_KEY(ws), expanded);
        renderRoot();
      }
      const r = nodes.get(id);
      if (r?.isConnected) {
        setRoving(row(r));
        row(r).scrollIntoView({ block: "nearest" });
      }
    }
  };

  let roving: HTMLElement | null = null;
  const setRoving = (el: HTMLElement | null) => {
    if (roving === el) return;
    if (roving) roving.tabIndex = -1;
    roving = el;
    if (el) el.tabIndex = 0;
  };
  const ensureRoving = () => {
    if (roving?.isConnected) return;
    roving = null;
    const first = nav.querySelector<HTMLElement>('.tree-row[aria-current="page"]') ?? nav.querySelector<HTMLElement>(".tree-row, .flat-row");
    setRoving(first);
  };

  const visibleRows = () =>
    [...nav.querySelectorAll<HTMLElement>(".tree-row, .flat-row, .note-card")].filter((el) => el.offsetParent !== null);

  const toggle = (id: string, open?: boolean) => {
    const p = pages.get(id);
    if (!p || !pages.hasChildren(p)) return false;
    const want = open ?? !expanded.has(id);
    if (want === expanded.has(id)) return false;
    if (want) expanded.add(id);
    else expanded.delete(id);
    writeSet(EXPANDED_KEY(ws), expanded);
    renderParent(id);
    return true;
  };

  const openFrom = (id: string, e: MouseEvent | KeyboardEvent) => {
    const newTab = e instanceof MouseEvent ? e.button === 1 || (engine.platform.mac ? e.metaKey : e.ctrlKey) : engine.platform.mac ? e.metaKey : e.ctrlKey;
    engine.openPage(id, { newTab });
  };

  const trashWithUndo = async (id: string) => {
    const p = pages.get(id);
    if (!p) return;
    const list = visibleRows();
    const at = list.findIndex((el) => el.dataset.id === id);
    try {
      const gone = await pages.trash(id);
      const heir = list[at + 1] ?? list[at - 1];
      if (heir?.isConnected) {
        setRoving(heir);
        heir.focus({ preventScroll: true });
      }
      toast({
        message: gone.length > 1 ? t("Moved “{title}” and {count} pages inside to the trash", { title: p.title || t("Untitled"), count: gone.length - 1 }) : t("Moved “{title}” to the trash", { title: p.title || t("Untitled") }),
        action: { label: t("Undo"), run: () => void pages.restore(id).catch(engine.reportError) },
      });
    } catch (e) {
      engine.reportError(e);
    }
  };

  const addInside = async (parentId: string, teamspaceId = "") => {
    try {
      const p = await pages.create(ws, parentId, "", teamspaceId);
      if (parentId && !expanded.has(parentId)) toggle(parentId, true);
      if (teamspaceId) toggleTeam(teamspaceId, true);
      engine.openPage(p.id);
      focusTitleSoon();
    } catch (e) {
      engine.reportError(e);
    }
  };

  const addDatabase = async (parentId: string) => {
    try {
      const r = await engine.api.pages.createCollection({ workspace_id: ws, ...(parentId ? { parent_id: parentId } : {}) });
      pages.upsert(r.page);
      if (parentId && !expanded.has(parentId)) toggle(parentId, true);
      engine.openPage(r.page.id);
      focusTitleSoon();
    } catch (e) {
      engine.reportError(e);
    }
  };

  const menuFor = (anchor: HTMLElement, id: string) => {
    const entries: MenuEntry[] = pageMenu(engine, id, {
      addInside: () => void addInside(id),
      trash: () => void trashWithUndo(id),
    });
    const at = entries.findIndex((e) => e.kind !== "separator" && e.kind !== "heading" && e.label === t("Add a page inside"));
    if (at >= 0) entries.splice(at + 1, 0, { label: t("Add a database inside"), icon: "database", run: () => void addDatabase(id) });
    openMenu(anchor, entries, { side: "bottom", align: "start", label: t("Page actions") });
  };

  const teamMenu = (anchor: HTMLElement, teamId: string) => {
    const team = pages.teamspace(ws, teamId);
    if (!team) return;
    const me = state.get("me")?.user.id ?? "";
    const entries: MenuEntry[] = [];
    if (teamWritable(team)) entries.push({ label: t("Add a page"), icon: "plus", run: () => void addInside("", teamId) });
    entries.push({ label: team.can_manage ? t("Teamspace settings") : t("Members"), icon: team.can_manage ? "settings" : "users", run: () => openTeamspaceDialog(engine, ws, team) });
    if (!team.role && team.access === "open" && writable()) {
      entries.push({ label: t("Join teamspace"), icon: "plus", run: () => void engine.api.teamspaces.setMember(teamId, me, "member").catch(engine.reportError) });
    }
    if (team.role) {
      entries.push({ kind: "separator" }, { label: t("Leave teamspace"), icon: "logout", danger: true, run: () => void engine.api.teamspaces.removeMember(teamId, me).catch(engine.reportError) });
    }
    openMenu(anchor, entries, { side: "bottom", align: "start", label: t("Teamspace options") });
  };

  nav.addEventListener("click", (e) => {
    if (suppressClick) {
      suppressClick = false;
      e.preventDefault();
      return;
    }
    const target = e.target as HTMLElement;
    const chip = target.closest<HTMLElement>(".tag-chip");
    if (chip) {
      const tag = chip.dataset.tag ?? "";
      state.set("tagFilter", state.get("tagFilter").toLowerCase() === tag.toLowerCase() ? "" : tag);
      return;
    }
    const noteFilter = target.closest<HTMLElement>("[data-note-filter]")?.dataset.noteFilter;
    if (noteFilter) {
      if (noteFilter === "clear") state.set("tagFilter", "");
      else notesUntagged = noteFilter === "untagged";
      renderNotes();
      return;
    }
    const action = target.closest<HTMLElement>("[data-action]");
    const teamRow = target.closest<HTMLElement>("[data-team]");
    if (teamRow) {
      const teamId = teamRow.dataset.team ?? "";
      const a = action?.dataset.action;
      setRoving(teamRow);
      if (a === "team-add") void addInside("", teamId);
      else if (a === "team-more" && action) teamMenu(action, teamId);
      else toggleTeam(teamId);
      return;
    }
    const host = target.closest<HTMLElement>("[data-id]");
    const id = host?.dataset.id ?? "";
    if (action && id) {
      const a = action.dataset.action;
      if (a === "toggle") toggle(id);
      else if (a === "add") void addInside(id);
      else if (a === "more") menuFor(action, id);
      else if (a === "unfav") void pages.toggleFavorite(id).catch(engine.reportError);
      return;
    }
    if (host && id) {
      setRoving(host);
      openFrom(id, e);
    }
  });

  nav.addEventListener("auxclick", (e) => {
    if (e.button !== 1) return;
    const host = (e.target as HTMLElement).closest<HTMLElement>("[data-id]");
    if (!host?.dataset.id) return;
    e.preventDefault();
    engine.openPage(host.dataset.id, { newTab: true });
  });

  nav.addEventListener("contextmenu", (e) => {
    const host = (e.target as HTMLElement).closest<HTMLElement>(".tree-row, .flat-row, .note-card");
    if (host?.dataset.team) {
      e.preventDefault();
      teamMenu(host, host.dataset.team);
      return;
    }
    if (!host?.dataset.id) return;
    e.preventDefault();
    menuFor(host, host.dataset.id);
  });

  nav.addEventListener("focusin", (e) => {
    const host = (e.target as HTMLElement).closest<HTMLElement>(".tree-row, .flat-row, .note-card");
    if (host) setRoving(host);
  });

  trashRow.addEventListener("click", (e) => {
    e.preventDefault();
    engine.navigate({ name: "trash" });
  });
  inboxRow.addEventListener("click", (e) => {
    e.preventDefault();
    engine.navigate({ name: "inbox" });
  });
  tree.add?.addEventListener("click", () => void addInside(""));
  notes.add?.addEventListener("click", () => void addInside(""));
  teams.add?.addEventListener("click", () => openTeamspaceDialog(engine, ws, null));
  filterClear.addEventListener("click", () => state.set("tagFilter", ""));

  nav.addEventListener("keydown", (e) => {
    const host = (e.target as HTMLElement).closest<HTMLElement>(".tree-row, .flat-row, .note-card");
    if (!host || e.target !== host) return;
    const id = host.dataset.id ?? "";
    const list = visibleRows();
    const i = list.indexOf(host);
    const mod = engine.platform.mac ? e.metaKey : e.ctrlKey;
    const go = (el: HTMLElement | undefined) => {
      if (!el) return;
      setRoving(el);
      el.focus();
    };
    const inTree = host.classList.contains("tree-row");
    const teamId = host.dataset.team;
    if (teamId && !["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)) {
      if (e.key === "Enter") toggleTeam(teamId);
      else if (e.key === "ArrowRight") {
        if (!toggleTeam(teamId, true)) go(teamNodes.get(teamId)?.querySelector<HTMLElement>(".tree-group .tree-row") ?? undefined);
      } else if (e.key === "ArrowLeft") toggleTeam(teamId, false);
      else if (e.key === "ContextMenu" || (e.key === "F10" && e.shiftKey)) teamMenu(host, teamId);
      else return;
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    switch (e.key) {
      case "ArrowDown":
        go(list[i + 1]);
        break;
      case "ArrowUp":
        go(list[i - 1]);
        break;
      case "Home":
        go(list[0]);
        break;
      case "End":
        go(list.at(-1));
        break;
      case "ArrowRight":
        if (!inTree) return;
        if (!toggle(id, true)) {
          const child = nodes.get(id)?.querySelector<HTMLElement>(".tree-group .tree-row");
          if (child) go(child);
          else return;
        }
        break;
      case "ArrowLeft": {
        if (!inTree) return;
        if (!toggle(id, false)) {
          const self = pages.get(id);
          const parent = self?.detached ? "" : self?.parent_id;
          const li = parent ? nodes.get(parent) : self?.teamspace_id ? teamNodes.get(self.teamspace_id) : undefined;
          if (li) go(row(li));
          else return;
        }
        break;
      }
      case "Enter":
        openFrom(id, e);
        break;
      case "F2":
        engine.openPage(id);
        focusTitleSoon();
        break;
      case "Backspace":
      case "Delete":
        if (!(mod || e.key === "Delete") || !writable() || !inTree) return;
        void trashWithUndo(id);
        break;
      case "ContextMenu":
        menuFor(host, id);
        break;
      default:
        if (e.key === "F10" && e.shiftKey) {
          menuFor(host, id);
          break;
        }
        return;
    }
    e.preventDefault();
    e.stopPropagation();
  });

  let suppressClick = false;
  let drag: {
    id: string;
    from: "tree" | "fav";
    ghost: HTMLElement;
    target: { el: HTMLElement; zone: Zone; kind: "tree" | "fav" } | null;
    hover: ReturnType<typeof setTimeout> | undefined;
    hoverId: string;
    scroll: number;
    x: number;
    y: number;
  } | null = null;

  const clearIndicator = () => {
    indicator.hidden = true;
    for (const el of nav.querySelectorAll("[data-drop]")) el.removeAttribute("data-drop");
  };

  const showIndicator = (el: HTMLElement, zone: Zone) => {
    clearIndicator();
    if (zone === "inside") {
      el.setAttribute("data-drop", "inside");
      return;
    }
    const navBox = nav.getBoundingClientRect();
    const box = el.getBoundingClientRect();
    const title = el.querySelector(".tree-title, .flat-row__text") ?? el;
    const left = title.getBoundingClientRect().left - navBox.left;
    const top = (zone === "before" ? box.top : box.bottom) - navBox.top + nav.scrollTop;
    indicator.style.transform = `translate(${Math.round(left)}px, ${Math.round(top - 1)}px)`;
    indicator.style.width = `${Math.max(24, Math.round(box.right - navBox.left - left - 8))}px`;
    indicator.hidden = false;
  };

  const resolveTarget = (x: number, y: number) => {
    if (!drag) return null;
    const under = document.elementFromPoint(x, y) as HTMLElement | null;
    if (!under || !nav.contains(under)) return null;
    const treeRow = under.closest<HTMLElement>(".tree-row");
    const favRow = under.closest<HTMLElement>('.flat-row[data-kind="fav"]');
    if (treeRow && drag.from === "tree") {
      if (treeRow.dataset.team) {
        const team = pages.teamspace(ws, treeRow.dataset.team);
        return team && teamWritable(team) ? { el: treeRow, zone: "inside" as Zone, kind: "tree" as const } : null;
      }
      const id = treeRow.dataset.id ?? "";
      if (id === drag.id || pages.wouldCycle(drag.id, id)) return null;
      const box = treeRow.getBoundingClientRect();
      const f = (y - box.top) / box.height;
      const zone: Zone = f < 0.3 ? "before" : f > 0.7 ? "after" : "inside";
      return { el: treeRow, zone, kind: "tree" as const };
    }
    if (favRow) {
      const box = favRow.getBoundingClientRect();
      return { el: favRow, zone: (y - box.top < box.height / 2 ? "before" : "after") as Zone, kind: "fav" as const };
    }
    if (under.closest('[data-section="favorites"]') && !favs.el.hidden) {
      const last = favList.lastElementChild?.firstElementChild as HTMLElement | null;
      if (last) return { el: last, zone: "after" as Zone, kind: "fav" as const };
    }
    return null;
  };

  const autoscroll = () => {
    if (!drag) return;
    const box = nav.getBoundingClientRect();
    const edge = 36;
    let v = 0;
    if (drag.y < box.top + edge) v = -Math.ceil((box.top + edge - drag.y) / 4);
    else if (drag.y > box.bottom - edge) v = Math.ceil((drag.y - (box.bottom - edge)) / 4);
    if (v) {
      nav.scrollTop += v;
      updateDrag(drag.x, drag.y);
    }
    drag.scroll = requestAnimationFrame(autoscroll);
  };

  const updateDrag = (x: number, y: number) => {
    if (!drag) return;
    drag.x = x;
    drag.y = y;
    drag.ghost.style.transform = `translate(${Math.round(x + 12)}px, ${Math.round(y + 8)}px)`;
    const target = resolveTarget(x, y);
    drag.target = target;
    if (!target) {
      clearIndicator();
      clearTimeout(drag.hover);
      drag.hoverId = "";
      return;
    }
    showIndicator(target.el, target.zone);
    const hoverId = target.zone === "inside" ? (target.el.dataset.id ?? "") : "";
    if (hoverId !== drag.hoverId) {
      clearTimeout(drag.hover);
      drag.hoverId = hoverId;
      if (hoverId) drag.hover = setTimeout(() => toggle(hoverId, true), 650);
    }
  };

  const endDrag = (commit: boolean) => {
    if (!drag) return;
    const d = drag;
    drag = null;
    cancelAnimationFrame(d.scroll);
    clearTimeout(d.hover);
    d.ghost.remove();
    clearIndicator();
    setAttr(nav, "data-dragging", null);
    nav.querySelector(".is-dragged")?.classList.remove("is-dragged");
    if (!commit || !d.target) return;
    const { el, zone, kind } = d.target;
    const targetId = el.dataset.id ?? "";
    if (kind === "fav") {
      const favIds = pages.favorites().map((p) => p.id).filter((x) => x !== d.id);
      const at = favIds.indexOf(targetId);
      const after = zone === "after" ? targetId : (favIds[at - 1] ?? "");
      void pages.moveFavorite(d.id, after).catch(engine.reportError);
      return;
    }
    const teamTarget = el.dataset.team;
    if (teamTarget) {
      if (!expanded.has(teamKey(teamTarget))) {
        expanded.add(teamKey(teamTarget));
        writeSet(EXPANDED_KEY(ws), expanded);
      }
      void pages.move(d.id, { parent_id: "", teamspace_id: teamTarget }).catch((e: unknown) => toast({ tone: "danger", message: errorMessage(e) }));
      return;
    }
    const target = pages.get(targetId);
    if (!target) return;
    let move: { parent_id: string; teamspace_id?: string; before_id?: string; after_id?: string };
    if (zone === "inside") move = { parent_id: target.id };
    else if (zone === "after" && expanded.has(target.id) && pages.hasChildren(target)) {
      const first = pages.children(ws, target.id)[0];
      move = first ? { parent_id: target.id, before_id: first.id } : { parent_id: target.id };
    } else if (zone === "before") move = { parent_id: target.parent_id, before_id: target.id };
    else move = { parent_id: target.parent_id, after_id: target.id };
    if (!move.parent_id) move.teamspace_id = target.teamspace_id ?? "";
    if (zone === "inside" && !expanded.has(target.id)) {
      expanded.add(target.id);
      writeSet(EXPANDED_KEY(ws), expanded);
    }
    void pages.move(d.id, move).catch((e: unknown) => toast({ tone: "danger", message: errorMessage(e) }));
  };

  nav.addEventListener("pointerdown", (e) => {
    if (e.button !== 0 || e.pointerType === "touch") return;
    const host = (e.target as HTMLElement).closest<HTMLElement>(".tree-row, .flat-row");
    if (!host || (e.target as HTMLElement).closest("button")) return;
    const id = host.dataset.id ?? "";
    const from = host.classList.contains("tree-row") ? "tree" : host.dataset.kind === "fav" ? "fav" : null;
    if (!id || !from || (from === "tree" && !writable())) return;
    const sx = e.clientX;
    const sy = e.clientY;
    const onMove = (ev: PointerEvent) => {
      if (!drag) {
        if (Math.hypot(ev.clientX - sx, ev.clientY - sy) < 5) return;
        const p = pages.get(id);
        const ghost = h("div", { class: "drag-ghost" }, pageIcon(p?.icon ?? "", fallbackGlyph(p?.kind ?? "doc")), h("span", { text: p?.title || t("Untitled") }));
        byId("overlay-root").append(ghost);
        host.classList.add("is-dragged");
        setAttr(nav, "data-dragging", "true");
        drag = { id, from, ghost, target: null, hover: undefined, hoverId: "", scroll: 0, x: ev.clientX, y: ev.clientY };
        drag.scroll = requestAnimationFrame(autoscroll);
      }
      updateDrag(ev.clientX, ev.clientY);
    };
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key === "Escape" && drag) {
        ev.preventDefault();
        ev.stopPropagation();
        finish(false);
      }
    };
    const finish = (commit: boolean) => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.removeEventListener("pointercancel", onCancel);
      window.removeEventListener("keydown", onKey, true);
      if (drag) {
        suppressClick = true;
        setTimeout(() => (suppressClick = false), 0);
        endDrag(commit);
      }
    };
    const onUp = () => finish(true);
    const onCancel = () => finish(false);
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onCancel);
    window.addEventListener("keydown", onKey, true);
  });

  const onChange = (c: StoreChange) => {
    switch (c.kind) {
      case "loaded":
        if (c.workspaceId === ws) {
          renderAll();
          revealActive();
        }
        return;
      case "favorites":
        renderFavorites();
        return;
      case "tags":
        if (c.workspaceId === ws) renderTags();
        return;
      case "teams":
        if (c.workspaceId === ws && pages.isLoaded(ws)) renderTeams();
        return;
      case "trash":
        return;
      case "upsert": {
        const p = c.page;
        const prev = c.prev;
        const homeOf = (x: Page) => (x.detached ? "" : x.parent_id);
        const regrouped = !!prev && ((prev.teamspace_id ?? "") !== (p.teamspace_id ?? "") || !!prev.detached !== !!p.detached);
        if (prev?.workspace_id === ws && (p.workspace_id !== ws || homeOf(prev) !== homeOf(p) || regrouped)) renderParent(homeOf(prev));
        if (p.workspace_id === ws) {
          if (!prev || homeOf(prev) !== homeOf(p) || regrouped || prev.sort_key !== p.sort_key || prev.workspace_id !== p.workspace_id) renderParent(homeOf(p));
          else {
            const li = nodes.get(p.id);
            if (li?.isConnected) patchRow(li, p, depthOf(li));
          }
          setText(tree.count, String(pages.all(ws).length));
          empty.hidden = root.childElementCount > 0;
          if (!prev || prev.tags.join("\n") !== p.tags.join("\n")) {
            renderTags();
            renderFilter();
          }
          renderNotes();
        }
        if (pages.isFavorite(p.id) || prev?.title !== p.title) renderFavorites();
        if (state.get("tagFilter")) renderFilter();
        return;
      }
      case "remove": {
        const parents = new Set<string>();
        for (const p of c.pages) if (p.workspace_id === ws) parents.add(p.detached ? "" : p.parent_id);
        for (const pid of parents) if (!c.pages.some((x) => x.id === pid)) renderParent(pid);
        for (const p of c.pages) nodes.delete(p.id);
        if (parents.size) {
          const n = pages.all(ws).length;
          setText(tree.count, n ? String(n) : "");
          empty.hidden = root.childElementCount > 0;
          renderTags();
          renderFilter();
          renderNotes();
        }
        renderFavorites();
        ensureRoving();
        return;
      }
    }
  };
  pages.subscribe(onChange);

  state.watch("workspaceId", (next) => {
    if (next === ws) return;
    ws = next;
    expanded = readSet(EXPANDED_KEY(ws));
    if (pages.isLoaded(ws)) renderAll();
  });
  state.watch("route", (r) => {
    setActive(r.name === "page" ? r.id : "");
    paintTrash();
  });
  state.watch("tagFilter", () => {
    renderTags();
    renderFilter();
    renderNotes();
  });
  state.watch("me", () => {
    paintWritable();
    renderFilter();
    renderNotes();
    renderFavorites();
  });

  renderAll();
  setActive(activePage());
}

export function focusTitleSoon(): void {
  requestAnimationFrame(() =>
    requestAnimationFrame(() => {
      const el = document.querySelector<HTMLTextAreaElement>(".page-title");
      if (el && !el.readOnly) {
        el.focus();
        el.setSelectionRange(el.value.length, el.value.length);
      }
    }),
  );
}
