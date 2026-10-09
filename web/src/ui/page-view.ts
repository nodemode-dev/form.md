import type { Engine, Page } from "../engine";
import { t } from "../engine/i18n";
import { byId, h, ref, setAttr, setText } from "./dom";
import { anchorTo } from "./floating";
import { parseIcon, TINTS } from "./glyphs";
import { iconNode } from "./icons";
import { openLayer } from "./layers";
import { type MenuEntry, openMenu } from "./menu";
import { fallbackGlyph, fileUrl, pageIcon, paintIcon } from "./pageicon";
import { openCoverPicker, openIconPicker } from "./pickers";
import { tagColorFor, focusTitleSoon } from "./tree";
import { mountEditor } from "../extensions/blocks";
import { mountDiscussion } from "./discussion";
import { type DatabaseViews, databaseViews, loadDatabase } from "../extensions/database/load";
import type { RowPropsHandle } from "../extensions/database/rowprops";
import type { DatabaseHandle } from "../extensions/database/surface";
import type { RowDetail } from "../engine/types";

const HOLD_MS = 450;
const HOLD_DB_MS = 1600;

const autosize = typeof CSS !== "undefined" && CSS.supports("field-sizing", "content") ? null : (el: HTMLTextAreaElement) => {
  el.style.height = "auto";
  el.style.height = `${el.scrollHeight}px`;
};

export function cleanTag(raw: string): string {
  return raw.trim().replace(/^#+/, "").replace(/\s+/g, "-").replace(/,/g, "").slice(0, 40);
}

function editDistance(a: string, b: string, max: number): number {
  if (Math.abs(a.length - b.length) > max) return max + 1;
  let prev = Array.from({ length: b.length + 1 }, (_, i) => i);
  for (let i = 1; i <= a.length; i++) {
    const cur = [i];
    let rowMin = i;
    for (let j = 1; j <= b.length; j++) {
      const v = Math.min((cur[j - 1] ?? 0) + 1, (prev[j] ?? 0) + 1, (prev[j - 1] ?? 0) + (a[i - 1] === b[j - 1] ? 0 : 1));
      cur[j] = v;
      if (v < rowMin) rowMin = v;
    }
    if (rowMin > max) return max + 1;
    prev = cur;
  }
  return prev[b.length] ?? max + 1;
}

export function suggestTags(all: { tag: string; count: number }[], draft: string, already: string[], limit = 8): { tag: string; count: number; similar: boolean }[] {
  const used = new Set(already.map((x) => x.toLowerCase()));
  const pool = all.filter((x) => !used.has(x.tag.toLowerCase()));
  const q = cleanTag(draft).toLowerCase();
  if (!q) return pool.slice(0, limit).map((x) => ({ ...x, similar: false }));
  const prefix: { tag: string; count: number; similar: boolean }[] = [];
  const infix: typeof prefix = [];
  const near: typeof prefix = [];
  const max = q.length >= 5 ? 2 : q.length >= 3 ? 1 : 0;
  for (const x of pool) {
    const k = x.tag.toLowerCase();
    if (k === q) continue;
    if (k.startsWith(q)) prefix.push({ ...x, similar: false });
    else if (k.includes(q)) infix.push({ ...x, similar: false });
    else if (max > 0 && editDistance(q, k, max) <= max) near.push({ ...x, similar: true });
  }
  return [...prefix, ...infix, ...near].slice(0, limit);
}

export function mountPageView(engine: Engine): void {
  const { state, pages } = engine;
  const view = ref("view-page");

  const coverActions = h(
    "div",
    { class: "cover__actions" },
    h("button", { class: "cover__btn", attrs: { type: "button", "data-action": "cover" } }, iconNode("image"), h("span", { text: t("Change cover") })),
    h("button", { class: "cover__btn", attrs: { type: "button", "data-action": "uncover" } }, iconNode("trash"), h("span", { text: t("Remove") })),
  );
  const cover = h("div", { class: "cover", attrs: { "aria-hidden": "true" } }, coverActions);
  const bigIcon = pageIcon("", "page", "page-icon--lg");
  const iconBtn = h("button", { class: "page-head__icon", attrs: { type: "button", "aria-label": t("Change icon"), "data-action": "icon" } }, bigIcon);
  const addIcon = h("button", { class: "adder", attrs: { type: "button", "data-action": "icon" } }, iconNode("shape"), h("span", { text: t("Add icon") }));
  const addCover = h("button", { class: "adder", attrs: { type: "button", "data-action": "cover" } }, iconNode("image"), h("span", { text: t("Add cover") }));
  const addDesc = h("button", { class: "adder", attrs: { type: "button", "data-action": "describe" } }, iconNode("align-left"), h("span", { text: t("Add description") }));
  const addComment = h("button", { class: "adder", attrs: { type: "button", "data-action": "comment" } }, iconNode("comment"), h("span", { text: t("Add comment") }));
  const adders = h("div", { class: "page-head__adders" }, addIcon, addCover, addDesc, addComment);
  const title = h("textarea", { class: "page-title", attrs: { rows: "1", placeholder: t("Untitled"), "aria-label": t("Title"), spellcheck: "true", maxlength: "500" } });
  const desc = h("textarea", { class: "page-desc", attrs: { rows: "1", placeholder: t("Say what this page is for"), "aria-label": t("Description"), maxlength: "2000" } });
  const chips = h("span", { class: "page-tags__chips" });
  const tagInput = h("input", { class: "page-tags__input", attrs: { type: "text", "aria-label": t("Add a tag"), autocomplete: "off", spellcheck: "false", maxlength: "40" } });
  const tagsRow = h("div", { class: "page-tags" }, iconNode("tag"), chips, tagInput);
  const head = h("header", { class: "page-head" }, iconBtn, adders, title, desc, tagsRow);
  const subTitle = h("h2", { class: "page-body__title", text: t("Pages inside") });
  const subList = h("ul", { class: "subpages", attrs: { role: "list" } });
  const addSub = h("button", { class: "adder adder--row", attrs: { type: "button", "data-action": "subpage" } }, iconNode("plus"), h("span", { text: t("Add a page inside") }));
  const docBody = ref("doc-body");
  const editorWrap = h("div", { class: "editor" }, docBody);
  const backTitle = h("h2", { class: "page-body__title", text: t("Linked from") });
  const backList = h("ul", { class: "subpages backlinks", attrs: { role: "list" } });
  const dbHost = h("div", { class: "page-db" });
  const propsHost = h("div", { class: "page-props" });
  const body = h("div", { class: "page-body" }, dbHost, editorWrap, subTitle, subList, addSub, backTitle, backList);
  const discussion = mountDiscussion(engine, () => paint());
  const inner = h("div", { class: "page" }, head, propsHost, discussion.el, body);
  view.append(cover, inner);
  const editor = mountEditor(engine, docBody, editorWrap);
  let rowProps: RowPropsHandle | null = null;
  const rowPanel = (): RowPropsHandle | null => {
    const m = rowProps ? null : databaseViews();
    if (m) rowProps = m.mountRowProps(engine, propsHost);
    return rowProps;
  };
  backTitle.hidden = true;
  dbHost.hidden = true;

  const rowCache = new Map<string, Promise<RowDetail | null>>();
  if (engine.boot.row?.row) rowCache.set(engine.boot.row.row.id, Promise.resolve(engine.boot.row));
  const rowDetail = (id: string) => {
    let job = rowCache.get(id);
    if (!job) {
      job = engine.api.collections.row(id).catch(() => null);
      rowCache.set(id, job);
      setTimeout(() => rowCache.delete(id), 30000);
    }
    return job;
  };
  const rowResolved = new Map<string, RowDetail | null>();

  type DbSlot = { id: string; box: HTMLElement; handle: DatabaseHandle | null; ready: Promise<void>; gone: boolean };
  let db: DbSlot | null = null;
  let dbNext: DbSlot | null = null;
  const dropSlot = (s: DbSlot | null) => {
    if (!s) return;
    s.gone = true;
    s.handle?.destroy();
    s.handle = null;
  };
  const prepareDb = (id: string): Promise<void> => {
    if (db?.id === id) return Promise.resolve();
    if (dbNext?.id === id) return dbNext.ready;
    dropSlot(dbNext);
    const slot: DbSlot = { id, box: h("div", { class: "page-db__box" }), handle: null, ready: Promise.resolve(), gone: false };
    const attach = (m: DatabaseViews): Promise<void> => {
      if (slot.gone) return Promise.resolve();
      slot.handle = m.mountDatabase(engine, slot.box, id, { inline: false });
      return slot.handle.ready;
    };
    const loaded = databaseViews();
    slot.ready = loaded ? attach(loaded) : loadDatabase().then(attach);
    dbNext = slot;
    return slot.ready;
  };
  const showDb = (id: string | null) => {
    if (!id) {
      dropSlot(db);
      db = null;
      dbHost.hidden = true;
      return;
    }
    if (db?.id === id) return;
    if (dbNext?.id !== id) void prepareDb(id);
    const next = dbNext;
    dbNext = null;
    if (!next) return;
    dropSlot(db);
    dbHost.replaceChildren(next.box);
    db = next;
    dbHost.hidden = false;
  };
  const preparePage = (p: Page): Promise<unknown> => {
    const notes = engine.comments.load(p.id).catch(() => undefined);
    if (p.kind === "collection") return Promise.all([prepareDb(p.id), notes]);
    if (p.row) return Promise.all([editor.prepare(p.id), rowDetail(p.id).then((d) => rowResolved.set(p.id, d)), loadDatabase(), notes]);
    return Promise.all([editor.prepare(p.id), notes]);
  };
  const pageReady = (p: Page): boolean => {
    if (!engine.comments.has(p.id)) return false;
    if (p.kind === "collection") return db?.id === p.id;
    if (p.row && (!rowResolved.has(p.id) || !databaseViews())) return false;
    return editor.isReady(p.id);
  };

  let shownId = "";
  let descOpen = false;
  let pendingId = "";
  let releasedId = "";
  let onPage = false;

  const current = (): Page | null => {
    const r = state.get("route");
    if (r.name !== "page") return null;
    const p = pages.get(r.id);
    if (p) return p;
    const d = state.get("detail");
    return d && d.id === r.id ? d : null;
  };
  const writable = (p: Page) => {
    const d = state.get("detail");
    if (d && d.id === p.id) return d.can_write;
    return engine.can(p.id, "content");
  };

  const setValue = (el: HTMLTextAreaElement, value: string, force: boolean) => {
    if (el.value === value) return;
    if (!force && document.activeElement === el) return;
    el.value = value;
    autosize?.(el);
  };

  const paintCover = (p: Page) => {
    const ref = p.cover;
    view.classList.toggle("has-cover", !!ref);
    if (!ref) {
      setAttr(cover, "data-cover", null);
      cover.querySelector("img")?.remove();
      return;
    }
    if (ref.startsWith("g:")) {
      setAttr(cover, "data-cover", ref.slice(2));
      cover.querySelector("img")?.remove();
      return;
    }
    const icon = parseIcon(ref);
    if (icon.kind !== "file") return;
    setAttr(cover, "data-cover", "picture");
    const src = fileUrl(icon.id);
    let img = cover.querySelector("img");
    if (!img) {
      img = h("img", { attrs: { alt: "", decoding: "async", draggable: "false" } });
      cover.prepend(img);
    }
    if (img.getAttribute("src") !== src) img.src = src;
  };

  const paintTags = (p: Page, w: boolean) => {
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(chips.children) as HTMLElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = chips.firstElementChild;
    for (const tag of p.tags) {
      const k = tag.toLowerCase();
      let chip = existing.get(k);
      if (chip) existing.delete(k);
      else {
        chip = h("button", { class: "tag-chip", attrs: { type: "button" } });
        chip.dataset.key = k;
      }
      chip.dataset.tag = tag;
      setText(chip, `#${tag}`);
      setAttr(chip, "data-color", tagColorFor(engine, p.workspace_id, tag));
      if (chip !== cursor) chips.insertBefore(chip, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
    tagInput.hidden = !w;
    setAttr(tagInput, "placeholder", p.tags.length ? t("Add tag") : t("Add a tag"));
    tagsRow.hidden = !w && p.tags.length === 0;
  };

  let backSeq = 0;
  const loadBacklinks = async (id: string) => {
    const mine = ++backSeq;
    try {
      const r = await engine.api.pages.backlinks(id);
      if (mine !== backSeq || shownId !== id) return;
      paintLinks(backList, r.pages);
      backTitle.hidden = r.pages.length === 0;
    } catch {
      if (mine === backSeq) backTitle.hidden = true;
    }
  };

  const paintLinks = (list: HTMLUListElement, items: Page[]) => {
    const existing = new Map<string, HTMLLIElement>();
    for (const c of Array.from(list.children) as HTMLLIElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = list.firstElementChild;
    for (const k of items) {
      let li = existing.get(k.id);
      if (li) existing.delete(k.id);
      else {
        li = h("li", {}, h("a", { class: "subpage", attrs: { href: `/p/${k.id}`, "data-id": k.id } }, pageIcon(k.icon, fallbackGlyph(k.kind)), h("span", { class: "subpage__title" })));
        li.dataset.key = k.id;
      }
      const a = li.firstElementChild as HTMLElement;
      paintIcon(a.firstElementChild as HTMLElement, k.icon, fallbackGlyph(k.kind));
      setText(a.lastElementChild as HTMLElement, k.title || t("Untitled"));
      if (li !== cursor) list.insertBefore(li, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
  };

  const paintSubpages = (p: Page, w: boolean) => {
    const kids = pages.isLoaded(p.workspace_id) ? pages.children(p.workspace_id, p.id) : [];
    const existing = new Map<string, HTMLLIElement>();
    for (const c of Array.from(subList.children) as HTMLLIElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = subList.firstElementChild;
    for (const k of kids) {
      let li = existing.get(k.id);
      if (li) existing.delete(k.id);
      else {
        li = h("li", {}, h("a", { class: "subpage", attrs: { href: `/p/${k.id}`, "data-id": k.id } }, pageIcon(k.icon, fallbackGlyph(k.kind)), h("span", { class: "subpage__title" })));
        li.dataset.key = k.id;
      }
      const a = li.firstElementChild as HTMLElement;
      paintIcon(a.firstElementChild as HTMLElement, k.icon, fallbackGlyph(k.kind));
      setText(a.lastElementChild as HTMLElement, k.title || t("Untitled"));
      setAttr(a.lastElementChild as HTMLElement, "data-untitled", k.title ? null : "true");
      if (li !== cursor) subList.insertBefore(li, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
    subTitle.hidden = kids.length === 0;
    addSub.hidden = !w || p.kind === "collection";
  };

  const paint = () => {
    const p = current();
    if (!p) return;
    if (p.id !== shownId && releasedId !== p.id && onPage && !pageReady(p)) {
      if (pendingId === p.id) return;
      pendingId = p.id;
      document.documentElement.dataset.navigating = "";
      const release = () => {
        if (pendingId !== p.id) return;
        pendingId = "";
        releasedId = p.id;
        paint();
      };
      void preparePage(p).then(release, release);
      setTimeout(release, p.kind === "collection" ? HOLD_DB_MS : HOLD_MS);
      return;
    }
    pendingId = "";
    delete document.documentElement.dataset.navigating;
    onPage = true;
    const fresh = p.id !== shownId;
    if (fresh) {
      shownId = p.id;
      releasedId = "";
      descOpen = false;
      closeSuggest();
      tagInput.value = "";
      const isDb = p.kind === "collection";
      setAttr(view, "data-kind", isDb ? "collection" : p.row ? "row" : null);
      editorWrap.hidden = isDb;
      if (isDb) {
        if (!dbNext || dbNext.id !== p.id) void prepareDb(p.id);
        showDb(p.id);
      } else {
        showDb(null);
        editor.show(p.id);
      }
      if (p.row) {
        const d = rowResolved.get(p.id);
        const panel = rowPanel();
        if (d !== undefined && panel) void panel.show(p.id, d);
        else void Promise.all([rowDetail(p.id), loadDatabase()]).then(([x]) => {
          rowResolved.set(p.id, x);
          if (shownId === p.id) void rowPanel()?.show(p.id, x);
        }, engine.reportError);
      } else rowProps?.hide();
      void loadBacklinks(p.id);
      discussion.show(p.id);
    }
    const w = writable(p);
    setAttr(view, "data-readonly", w ? null : "true");
    title.readOnly = !w;
    desc.readOnly = !w;
    setValue(title, p.title, fresh);
    setValue(desc, p.description, fresh);
    paintCover(p);
    iconBtn.hidden = !p.icon;
    iconBtn.disabled = !w;
    if (p.icon) paintIcon(bigIcon, p.icon, fallbackGlyph(p.kind));
    addIcon.hidden = !w || !!p.icon;
    addCover.hidden = !w || !!p.cover;
    const showDesc = !!p.description || descOpen;
    addDesc.hidden = !w || showDesc;
    addComment.hidden = !engine.comments.get(p.id).can_comment || !discussion.el.hidden;
    adders.hidden = addIcon.hidden && addCover.hidden && addDesc.hidden && addComment.hidden;
    desc.hidden = !showDesc;
    coverActions.hidden = !w;
    paintTags(p, w);
    paintSubpages(p, w);
  };

  title.addEventListener("input", () => {
    const p = current();
    if (!p) return;
    const v = title.value.replace(/\r?\n/g, " ");
    if (v !== title.value) title.value = v;
    autosize?.(title);
    pages.edit(p.id, { title: v });
  });
  title.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (!desc.hidden) desc.focus();
      else title.blur();
    } else if (e.key === "Escape") title.blur();
  });
  title.addEventListener("blur", () => {
    const p = current();
    if (p) void pages.flush(p.id);
  });
  desc.addEventListener("input", () => {
    const p = current();
    if (!p) return;
    autosize?.(desc);
    pages.edit(p.id, { description: desc.value });
  });
  desc.addEventListener("blur", () => {
    const p = current();
    if (!p) return;
    void pages.flush(p.id);
    if (!desc.value.trim() && descOpen) {
      descOpen = false;
      paint();
    }
  });
  desc.addEventListener("keydown", (e) => {
    if (e.key === "Escape") desc.blur();
  });

  const chooseIcon = (anchor: HTMLElement) => {
    const p = current();
    if (!p || !writable(p)) return;
    openIconPicker(engine, anchor, {
      current: p.icon,
      upload: { purpose: "icon", page_id: p.id },
      allowRemove: !!p.icon,
      onPick: (ref) => pages.edit(p.id, { icon: ref }, false),
      onRemove: () => pages.edit(p.id, { icon: "" }, false),
    });
  };
  const chooseCover = (anchor: HTMLElement) => {
    const p = current();
    if (!p || !writable(p)) return;
    openCoverPicker(engine, anchor, {
      current: p.cover,
      pageId: p.id,
      onPick: (ref) => pages.edit(p.id, { cover: ref }, false),
      onRemove: () => pages.edit(p.id, { cover: "" }, false),
    });
  };

  view.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;
    const sub = target.closest<HTMLElement>(".subpage");
    if (sub?.dataset.id) {
      e.preventDefault();
      engine.openPage(sub.dataset.id, { newTab: engine.platform.mac ? e.metaKey : e.ctrlKey });
      return;
    }
    const chip = target.closest<HTMLElement>(".page-tags .tag-chip");
    if (chip) return tagMenu(chip);
    const action = target.closest<HTMLElement>("[data-action]")?.dataset.action;
    const p = current();
    if (!action || !p) return;
    const anchor = target.closest<HTMLElement>("[data-action]") as HTMLElement;
    if (action === "icon") chooseIcon(anchor);
    else if (action === "cover") chooseCover(anchor);
    else if (action === "uncover") pages.edit(p.id, { cover: "" }, false);
    else if (action === "describe") {
      descOpen = true;
      paint();
      desc.focus();
    } else if (action === "comment") {
      discussion.openComposer();
    } else if (action === "subpage") {
      void pages
        .create(p.workspace_id, p.id)
        .then((c) => {
          engine.openPage(c.id);
          focusTitleSoon();
        })
        .catch(engine.reportError);
    }
  });

  const setTags = (next: string[]) => {
    const p = current();
    if (p) pages.edit(p.id, { tags: next }, false);
  };

  const tagMenu = (chip: HTMLElement) => {
    const p = current();
    const tag = chip.dataset.tag ?? "";
    if (!p || !tag) return;
    const w = writable(p);
    const now = engine.pages.tagColor(p.workspace_id, tag);
    const entries: MenuEntry[] = [{ label: t("Show pages tagged #{tag}", { tag }), icon: "filter", run: () => state.set("tagFilter", tag) }];
    if (w) {
      entries.push({ kind: "heading", label: t("Colour") });
      entries.push({ label: t("Automatic"), checked: !now, run: () => void pages.setTagColor(p.workspace_id, tag, "").catch(engine.reportError) });
      for (const c of TINTS) {
        const dot = h("span", { class: "tag-dot", attrs: { "data-color": c } });
        entries.push({ label: c.charAt(0).toUpperCase() + c.slice(1), lead: dot, checked: now === c, run: () => void pages.setTagColor(p.workspace_id, tag, c).catch(engine.reportError) });
      }
      entries.push({ kind: "separator" }, { label: t("Remove tag"), icon: "close", danger: true, run: () => setTags(p.tags.filter((x) => x !== tag)) });
    }
    openMenu(chip, entries, { side: "bottom", align: "start", label: `#${tag}` });
  };

  let suggest: { panel: HTMLElement; close: () => void; items: { tag: string }[]; sel: number } | null = null;
  function closeSuggest() {
    suggest?.close();
    suggest = null;
  }
  const addTag = (raw: string) => {
    const p = current();
    const tag = cleanTag(raw);
    tagInput.value = "";
    closeSuggest();
    if (!p || !tag || p.tags.some((x) => x.toLowerCase() === tag.toLowerCase()) || p.tags.length >= 20) return;
    setTags([...p.tags, tag]);
  };
  const paintSuggest = () => {
    const p = current();
    if (!p) return;
    const hits = suggestTags(pages.tagCounts(p.workspace_id), tagInput.value, p.tags);
    if (!hits.length) return closeSuggest();
    if (!suggest) {
      const panel = h("div", { class: "popover suggest", attrs: { role: "listbox", "aria-label": t("Tag suggestions") } });
      byId("overlay-root").append(panel);
      const release = anchorTo(panel, tagInput, { side: "bottom", align: "start", offset: 4 });
      requestAnimationFrame(() => panel.classList.add("is-visible"));
      let closed = false;
      const closeLayer = openLayer({
        el: panel,
        modal: false,
        dismissOnOutside: true,
        onClose: () => {
          if (closed) return;
          closed = true;
          release();
          panel.remove();
          if (suggest?.panel === panel) suggest = null;
        },
      });
      panel.addEventListener("pointerdown", (e) => e.preventDefault());
      panel.addEventListener("click", (e) => {
        const b = (e.target as HTMLElement).closest<HTMLElement>("[data-tag]");
        if (b) addTag(b.dataset.tag ?? "");
      });
      suggest = { panel, close: closeLayer, items: [], sel: 0 };
    }
    suggest.items = hits;
    suggest.sel = Math.min(suggest.sel, hits.length - 1);
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(suggest.panel.children) as HTMLElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = suggest.panel.firstElementChild;
    hits.forEach((x, i) => {
      if (!suggest) return;
      const k = x.tag.toLowerCase();
      let b = existing.get(k);
      if (b) existing.delete(k);
      else {
        b = h("button", { class: "suggest__item", attrs: { type: "button", role: "option", tabindex: "-1" } }, h("span", { class: "tag-chip" }), h("span", { class: "suggest__count" }), h("span", { class: "suggest__hint" }));
        b.dataset.key = k;
      }
      b.dataset.tag = x.tag;
      const chip = b.firstElementChild as HTMLElement;
      setText(chip, `#${x.tag}`);
      setAttr(chip, "data-color", tagColorFor(engine, p.workspace_id, x.tag));
      setText(b.children[1] as HTMLElement, String(x.count));
      setText(b.children[2] as HTMLElement, x.similar ? t("similar") : "");
      setAttr(b, "aria-selected", i === suggest.sel ? "true" : "false");
      if (b !== cursor) suggest.panel.insertBefore(b, cursor);
      else cursor = cursor.nextElementSibling;
    });
    for (const el of existing.values()) el.remove();
  };
  let navigated = false;
  tagInput.addEventListener("focus", () => {
    navigated = false;
    paintSuggest();
  });
  tagInput.addEventListener("input", () => {
    navigated = false;
    if (suggest) suggest.sel = 0;
    paintSuggest();
  });
  tagInput.addEventListener("blur", () => {
    const v = tagInput.value;
    setTimeout(() => {
      if (document.activeElement === tagInput) return;
      closeSuggest();
      if (v.trim()) addTag(v);
    }, 0);
  });
  tagInput.addEventListener("keydown", (e) => {
    const p = current();
    if (!p) return;
    if ((e.key === "ArrowDown" || e.key === "ArrowUp") && suggest?.items.length) {
      e.preventDefault();
      const n = suggest.items.length;
      suggest.sel = navigated ? (suggest.sel + (e.key === "ArrowDown" ? 1 : -1) + n) % n : e.key === "ArrowDown" ? 0 : n - 1;
      navigated = true;
      paintSuggest();
    } else if (e.key === "Enter" || e.key === "," || e.key === "Tab") {
      const typed = tagInput.value.trim();
      if (e.key === "Tab" && !typed && !navigated) return;
      const pick = suggest && navigated ? suggest.items[suggest.sel]?.tag : undefined;
      if (!pick && !typed) return;
      e.preventDefault();
      addTag(pick ?? typed);
    } else if (e.key === "Escape") {
      if (suggest) {
        e.preventDefault();
        e.stopPropagation();
        closeSuggest();
      } else tagInput.blur();
    } else if (e.key === "Backspace" && !tagInput.value && p.tags.length) {
      setTags(p.tags.slice(0, -1));
    }
  });

  state.watch("route", (r) => {
    if (r.name !== "page") {
      onPage = false;
      pendingId = "";
      delete document.documentElement.dataset.navigating;
    }
    paint();
  });
  state.watch("detail", paint);
  state.watch("me", paint);
  pages.subscribe((c) => {
    const p = current();
    if (!p) return;
    if (c.kind === "upsert" && (c.page.id === p.id || c.page.parent_id === p.id || c.prev?.parent_id === p.id)) paint();
    else if (c.kind === "remove" && c.pages.some((x) => x.parent_id === p.id)) paint();
    else if (c.kind === "tags" || c.kind === "loaded") paint();
  });
  paint();
}
