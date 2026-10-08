import type { Engine, Page } from "../engine";
import type { Command } from "../engine/commands";
import { formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import type { Member, SearchHit, SearchQuery } from "../engine/types";
import { byId, delayedIndicator, h, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { openLayer } from "./layers";
import { reconcile } from "./list";
import { type MenuEntry, openMenu } from "./menu";
import { pageIcon } from "./pageicon";

type Item =
  | { kind: "heading"; key: string; label: string }
  | { kind: "page"; key: string; hit: SearchHit }
  | { kind: "action"; key: string; command: Command; chord: string }
  | { kind: "more"; key: string }
  | { kind: "empty"; key: string; label: string };

type DateRange = "any" | "today" | "week" | "month" | "year";

interface Filters {
  titleOnly: boolean;
  creators: string[];
  parent: { id: string; title: string } | null;
  date: DateRange;
  dateField: "edited" | "created";
  sort: "best" | "edited" | "created";
}

const DAY = 86400000;
const PAGE_SIZE = 20;

function fillMarked(el: HTMLElement, text: string): void {
  const parts: Node[] = [];
  let open = false;
  for (const piece of text.split(/([\u0002\u0003])/)) {
    if (piece === "\u0002") open = true;
    else if (piece === "\u0003") open = false;
    else if (piece) parts.push(open ? h("mark", { text: piece }) : document.createTextNode(piece));
  }
  const sig = text;
  if (el.dataset.sig === sig) return;
  el.dataset.sig = sig;
  el.replaceChildren(...parts);
}

function rangeStart(r: DateRange, now: number): number {
  const d = new Date(now);
  d.setHours(0, 0, 0, 0);
  switch (r) {
    case "today":
      return d.getTime();
    case "week":
      return d.getTime() - 6 * DAY;
    case "month":
      return d.getTime() - 29 * DAY;
    case "year":
      return d.getTime() - 364 * DAY;
    default:
      return 0;
  }
}

export function mountSearch(engine: Engine): void {
  const { state, api, commands, keymap } = engine;
  let parts: ReturnType<typeof build> | null = null;
  let closeLayer: (() => void) | null = null;
  let hideTimer: ReturnType<typeof setTimeout> | undefined;
  let seq = 0;
  let inflight: AbortController | null = null;
  let debounce: ReturnType<typeof setTimeout> | undefined;
  let items: Item[] = [];
  let active = 0;
  let hits: SearchHit[] = [];
  let more = false;
  let recents: SearchHit[] = [];
  let people: Member[] = [];
  const filters: Filters = { titleOnly: false, creators: [], parent: null, date: "any", dateField: "edited", sort: "best" };

  const workspace = () => state.get("workspaceId") ?? "";

  function build() {
    const input = h("input", {
      class: "srch__input",
      attrs: { type: "search", role: "combobox", "aria-expanded": "true", "aria-controls": "srch-list", "aria-autocomplete": "list", autocomplete: "off", spellcheck: "false", placeholder: t("Search pages, or type > for actions"), "aria-label": t("Search") },
    });
    const busy = h("span", { class: "srch__busy", attrs: { "aria-hidden": "true" } });
    const head = h("div", { class: "srch__head" }, h("span", { class: "srch__glass" }, iconNode("search")), input, busy);
    const chip = (label: string) => h("button", { class: "chip-btn", attrs: { type: "button", "aria-pressed": "false" } }, h("span", { text: label }), iconNode("chevron-down"));
    const titleChip = h("button", { class: "chip-btn", attrs: { type: "button", "aria-pressed": "false" } }, h("span", { text: t("Title only") }));
    const byChip = chip(t("Created by"));
    const inChip = chip(t("In"));
    const dateChip = chip(t("Date"));
    const sortChip = chip(t("Best match"));
    const filterRow = h("div", { class: "srch__filters", attrs: { role: "group", "aria-label": t("Filters") } }, titleChip, byChip, inChip, dateChip, h("span", { class: "srch__spacer" }), sortChip);
    const list = h("div", { class: "srch__list", attrs: { id: "srch-list", role: "listbox", "aria-label": t("Results") } });
    const foot = h(
      "div",
      { class: "srch__foot" },
      h("span", {}, h("kbd", { text: "↑" }), h("kbd", { text: "↓" }), " ", t("to move")),
      h("span", {}, h("kbd", { text: "Enter" }), " ", t("to open")),
      h("span", {}, h("kbd", { text: keymap.format("mod+enter").join("+") }), " ", t("in a new tab")),
      h("span", { class: "srch__took" }),
    );
    const dialog = h("div", { class: "dialog srch", attrs: { role: "dialog", "aria-modal": "true", "aria-label": t("Search"), tabindex: "-1" } }, head, filterRow, list, foot);
    const scrim = h("div", { class: "scrim", attrs: { "aria-hidden": "true" } });
    scrim.hidden = true;
    dialog.hidden = true;
    byId("overlay-root").append(scrim, dialog);

    input.addEventListener("input", () => schedule(110));
    input.addEventListener("keydown", (e) => {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        move(e.key === "ArrowDown" ? 1 : -1);
      } else if (e.key === "Enter") {
        e.preventDefault();
        choose(active, e.ctrlKey || e.metaKey);
      } else if (e.key === "Home" && e.ctrlKey) {
        e.preventDefault();
        setActive(firstSelectable(0, 1));
      }
    });
    list.addEventListener("pointermove", (e) => {
      const row = (e.target as Element).closest<HTMLElement>("[data-index]");
      if (row && Number(row.dataset.index) !== active) setActive(Number(row.dataset.index), false);
    });
    list.addEventListener("click", (e) => {
      const row = (e.target as Element).closest<HTMLElement>("[data-index]");
      if (row) choose(Number(row.dataset.index), e.ctrlKey || e.metaKey || e.button === 1);
    });
    titleChip.addEventListener("click", () => {
      filters.titleOnly = !filters.titleOnly;
      paintChips();
      schedule(0);
    });
    byChip.addEventListener("click", () => {
      const pick = (id: string) => {
        filters.creators = filters.creators.includes(id) ? filters.creators.filter((x) => x !== id) : [...filters.creators, id];
        paintChips();
        schedule(0);
      };
      const entries: MenuEntry[] = people.length
        ? people.filter((m) => !m.disabled).map((m) => ({ label: m.name || m.email, checked: filters.creators.includes(m.user_id), run: () => pick(m.user_id) }))
        : [{ label: t("Loading people"), run: () => undefined }];
      if (filters.creators.length) entries.push({ kind: "separator" }, { label: t("Anyone"), run: () => ((filters.creators = []), paintChips(), schedule(0)) });
      openMenu(byChip, entries, { side: "bottom", align: "start", label: t("Created by") });
    });
    inChip.addEventListener("click", () => {
      const r = state.get("route");
      const here = r.name === "page" ? engine.pages.get(r.id) : undefined;
      const set = (p: { id: string; title: string } | null) => {
        filters.parent = p;
        paintChips();
        schedule(0);
      };
      const entries: MenuEntry[] = [{ label: t("Anywhere in the workspace"), checked: !filters.parent, run: () => set(null) }];
      if (here) {
        entries.push({ kind: "heading", label: t("Inside") });
        const chain: Page[] = [];
        let cur: Page | undefined = here;
        for (let i = 0; cur && i < 32; i++) {
          chain.unshift(cur);
          cur = cur.parent_id ? engine.pages.get(cur.parent_id) : undefined;
        }
        for (const p of chain.reverse()) entries.push({ label: p.title || t("Untitled"), lead: pageIcon(p.icon, p.kind === "collection" ? "database" : "page"), checked: filters.parent?.id === p.id, run: () => set({ id: p.id, title: p.title || t("Untitled") }) });
      }
      openMenu(inChip, entries, { side: "bottom", align: "start", label: t("In") });
    });
    dateChip.addEventListener("click", () => {
      const set = (change: Partial<Filters>) => {
        Object.assign(filters, change);
        paintChips();
        schedule(0);
      };
      const ranges: [DateRange, string][] = [["any", t("Any time")], ["today", t("Today")], ["week", t("Last 7 days")], ["month", t("Last 30 days")], ["year", t("Last 12 months")]];
      openMenu(
        dateChip,
        [
          { kind: "heading", label: t("Date") },
          { label: t("Last edited"), checked: filters.dateField === "edited", run: () => set({ dateField: "edited" }) },
          { label: t("Created"), checked: filters.dateField === "created", run: () => set({ dateField: "created" }) },
          { kind: "separator" },
          ...ranges.map(([k, label]) => ({ label, checked: filters.date === k, run: () => set({ date: k }) })),
        ],
        { side: "bottom", align: "start", label: t("Date") },
      );
    });
    sortChip.addEventListener("click", () => {
      const set = (sort: Filters["sort"]) => {
        filters.sort = sort;
        paintChips();
        schedule(0);
      };
      openMenu(
        sortChip,
        [
          { label: t("Best match"), checked: filters.sort === "best", run: () => set("best") },
          { label: t("Last edited"), checked: filters.sort === "edited", run: () => set("edited") },
          { label: t("Created"), checked: filters.sort === "created", run: () => set("created") },
        ],
        { side: "bottom", align: "end", label: t("Sort") },
      );
    });
    return { scrim, dialog, input, busy, list, titleChip, byChip, inChip, dateChip, sortChip, took: foot.querySelector(".srch__took") as HTMLElement };
  }

  const paintChips = () => {
    if (!parts) return;
    const p = parts;
    setAttr(p.titleChip, "aria-pressed", String(filters.titleOnly));
    const names = filters.creators.map((id) => people.find((m) => m.user_id === id)?.name ?? "").filter(Boolean);
    setText(p.byChip.firstElementChild as HTMLElement, names.length ? t("By {names}", { names: names.join(", ") }) : t("Created by"));
    setAttr(p.byChip, "aria-pressed", String(filters.creators.length > 0));
    setText(p.inChip.firstElementChild as HTMLElement, filters.parent ? t("In {page}", { page: filters.parent.title }) : t("In"));
    setAttr(p.inChip, "aria-pressed", String(!!filters.parent));
    const rangeLabel: Record<DateRange, string> = { any: t("Date"), today: t("Today"), week: t("Last 7 days"), month: t("Last 30 days"), year: t("Last 12 months") };
    setText(p.dateChip.firstElementChild as HTMLElement, filters.date === "any" ? t("Date") : `${filters.dateField === "created" ? t("Created") : t("Edited")} ${rangeLabel[filters.date].toLowerCase()}`);
    setAttr(p.dateChip, "aria-pressed", String(filters.date !== "any"));
    const sortLabel = { best: t("Best match"), edited: t("Last edited"), created: t("Created") }[filters.sort];
    setText(p.sortChip.firstElementChild as HTMLElement, sortLabel);
  };

  const query = (): string => parts?.input.value.trim() ?? "";
  const actionMode = () => query().startsWith(">");

  const actions = (q: string): Item[] => {
    const words = q.toLowerCase().split(/\s+/).filter(Boolean);
    const chords = new Map<string, string>();
    for (const s of keymap.list(new Set(["global", "view", "editor"]))) if (s.chords[0]) chords.set(s.command, s.chords[0].join("+"));
    return commands
      .list()
      .filter((c) => c.enabled?.() !== false && !c.id.startsWith("layers.") && c.id !== "search.open")
      .filter((c) => {
        const hay = `${c.label()} ${c.group()}`.toLowerCase();
        return words.every((w) => hay.includes(w));
      })
      .slice(0, q ? 30 : 8)
      .map((command) => ({ kind: "action" as const, key: `a:${command.id}`, command, chord: chords.get(command.id) ?? "" }));
  };

  const compose = () => {
    const q = query();
    const next: Item[] = [];
    if (actionMode()) {
      const list = actions(q.slice(1).trim());
      next.push({ kind: "heading", key: "h:actions", label: t("Actions") }, ...(list.length ? list : [{ kind: "empty" as const, key: "e:actions", label: t("No action matches") }]));
    } else if (!q) {
      if (recents.length) next.push({ kind: "heading", key: "h:recent", label: t("Recent") }, ...recents.map((hit) => ({ kind: "page" as const, key: `r:${hit.id}`, hit })));
      const list = actions("");
      if (list.length) next.push({ kind: "heading", key: "h:actions", label: t("Actions") }, ...list);
    } else {
      next.push({ kind: "heading", key: "h:pages", label: t("Pages") });
      if (hits.length) next.push(...hits.map((hit) => ({ kind: "page" as const, key: `p:${hit.id}`, hit })));
      else next.push({ kind: "empty", key: "e:pages", label: t("No pages match") });
      if (more) next.push({ kind: "more", key: "more" });
      const list = actions(q).slice(0, 4);
      if (list.length) next.push({ kind: "heading", key: "h:actions", label: t("Actions") }, ...list);
    }
    items = next;
    if (!selectable(items[active])) active = firstSelectable(0, 1);
    paint();
  };

  const selectable = (it: Item | undefined) => !!it && (it.kind === "page" || it.kind === "action" || it.kind === "more");

  const firstSelectable = (from: number, dir: number): number => {
    for (let i = from; i >= 0 && i < items.length; i += dir) if (selectable(items[i])) return i;
    return -1;
  };

  const paint = () => {
    if (!parts) return;
    const p = parts;
    reconcile(
      p.list,
      items.map((it, index) => ({ it, index })),
      (x) => x.it.key,
      (x) => {
        const it = x.it;
        if (it.kind === "heading") return h("div", { class: "srch__section", attrs: { role: "presentation" } });
        if (it.kind === "empty") return h("div", { class: "srch__empty", attrs: { role: "presentation" } });
        if (it.kind === "more") return h("div", { class: "srch__row srch__row--more", attrs: { role: "option" } }, h("span", { class: "srch__icon" }, iconNode("chevron-down")), h("span", { class: "srch__title" }));
        if (it.kind === "action")
          return h("div", { class: "srch__row srch__row--action", attrs: { role: "option" } }, h("span", { class: "srch__icon" }, iconNode("keyboard")), h("span", { class: "srch__main" }, h("span", { class: "srch__title" }), h("span", { class: "srch__path" })), h("span", { class: "srch__meta" }));
        return h(
          "div",
          { class: "srch__row", attrs: { role: "option" } },
          h("span", { class: "srch__icon" }),
          h("span", { class: "srch__main" }, h("span", { class: "srch__title" }), h("span", { class: "srch__path" }), h("span", { class: "srch__snip" })),
          h("span", { class: "srch__meta" }),
        );
      },
      (el, x) => {
        const it = x.it;
        el.dataset.index = String(x.index);
        if (it.kind === "heading" || it.kind === "empty") {
          setText(el, it.label);
          return;
        }
        el.id = `srch-opt-${x.index}`;
        setAttr(el, "aria-selected", String(x.index === active));
        if (it.kind === "more") {
          setText(el.querySelector(".srch__title") as HTMLElement, t("Show more results"));
          return;
        }
        if (it.kind === "action") {
          setText(el.querySelector(".srch__title") as HTMLElement, it.command.label());
          setText(el.querySelector(".srch__path") as HTMLElement, it.command.group());
          setText(el.querySelector(".srch__meta") as HTMLElement, it.chord);
          return;
        }
        const hit = it.hit;
        const icon = el.querySelector(".srch__icon") as HTMLElement;
        const sig = `${hit.icon}|${hit.kind}|${hit.row}`;
        if (icon.dataset.sig !== sig) {
          icon.dataset.sig = sig;
          icon.replaceChildren(pageIcon(hit.icon, hit.kind === "collection" ? "database" : "page"));
        }
        fillMarked(el.querySelector(".srch__title") as HTMLElement, hit.title_hl || hit.title || t("Untitled"));
        setText(el.querySelector(".srch__path") as HTMLElement, hit.path.map((c) => c.title || t("Untitled")).join(" / "));
        const snip = el.querySelector(".srch__snip") as HTMLElement;
        const text = [hit.heading, hit.snippet].filter(Boolean).join("  ·  ");
        snip.hidden = !text;
        fillMarked(snip, text);
        setText(el.querySelector(".srch__meta") as HTMLElement, hit.updated_at ? formatRelative(hit.updated_at) : "");
      },
    );
    const cur = active >= 0 ? p.list.querySelector<HTMLElement>(`#srch-opt-${active}`) : null;
    setAttr(p.input, "aria-activedescendant", cur ? cur.id : null);
  };

  const setActive = (i: number, scroll = true) => {
    if (i < 0 || !parts) return;
    active = i;
    for (const el of Array.from(parts.list.querySelectorAll<HTMLElement>("[aria-selected='true']"))) el.setAttribute("aria-selected", "false");
    const cur = parts.list.querySelector<HTMLElement>(`#srch-opt-${i}`);
    if (cur) {
      cur.setAttribute("aria-selected", "true");
      parts.input.setAttribute("aria-activedescendant", cur.id);
      if (scroll) cur.scrollIntoView({ block: "nearest" });
    }
  };

  const move = (dir: number) => {
    const next = firstSelectable(active + dir, dir);
    if (next >= 0) setActive(next);
  };

  const choose = (i: number, newTab: boolean) => {
    const it = items[i];
    if (!it || !selectable(it)) return;
    if (it.kind === "more") {
      void fetchHits(true);
      return;
    }
    close();
    if (it.kind === "action") {
      setTimeout(() => commands.run(it.command.id), 0);
      return;
    }
    engine.openPage(it.hit.id, { newTab });
  };

  const request = (offset: number): SearchQuery => {
    const q: SearchQuery = { q: query(), workspace_id: workspace(), limit: PAGE_SIZE, offset };
    if (filters.titleOnly) q.in = "title";
    if (filters.creators.length) q.creator = filters.creators;
    if (filters.parent) q.parent = filters.parent.id;
    if (filters.date !== "any") {
      q.date = filters.dateField;
      q.from = rangeStart(filters.date, Date.now());
    }
    if (filters.sort !== "best") q.sort = filters.sort;
    return q;
  };

  const fetchHits = async (append: boolean) => {
    if (!parts) return;
    const p = parts;
    const mine = ++seq;
    inflight?.abort();
    const ctl = new AbortController();
    inflight = ctl;
    const done = delayedIndicator(() => p.busy.classList.add("is-on"), () => p.busy.classList.remove("is-on"), 220, 300);
    try {
      const res = await api.pages.search(request(append ? hits.length : 0), ctl.signal);
      if (mine !== seq) return;
      hits = append ? [...hits, ...res.hits.filter((x) => !hits.some((y) => y.id === x.id))] : res.hits;
      more = res.more;
      setText(p.took, t("{ms} ms", { ms: Math.round(res.took_ms) }));
      if (!append) active = -1;
      compose();
    } catch (e) {
      if (!ctl.signal.aborted) engine.reportError(e);
    } finally {
      done();
    }
  };

  const schedule = (delay: number) => {
    clearTimeout(debounce);
    const q = query();
    if (!q || actionMode()) {
      seq++;
      inflight?.abort();
      hits = [];
      more = false;
      compose();
      return;
    }
    debounce = setTimeout(() => void fetchHits(false), delay);
  };

  const loadRecents = async () => {
    const ws = workspace();
    try {
      const r = await api.pages.recent(12);
      recents = r.pages
        .filter((x) => x.workspace_id === ws && !x.trashed_at)
        .slice(0, 6)
        .map((x) => ({ id: x.id, title: x.title, title_hl: "", icon: x.icon, kind: x.kind, row: !!x.row, path: [], match: "title", heading: "", snippet: "", created_at: x.created_at, created_by: "", updated_at: x.updated_at }));
      if (!query()) compose();
    } catch {
      recents = [];
    }
  };

  const open = () => {
    parts ??= build();
    const p = parts;
    clearTimeout(hideTimer);
    if (p.dialog.hidden) {
      p.input.value = "";
      hits = [];
      more = false;
      active = 0;
      compose();
      void loadRecents();
      const ws = workspace();
      if (ws) void api.workspaces.members(ws).then((r) => ((people = r.members), paintChips()), () => undefined);
      paintChips();
    }
    p.scrim.hidden = false;
    p.dialog.hidden = false;
    requestAnimationFrame(() => {
      p.scrim.classList.add("is-visible");
      p.dialog.classList.add("is-visible");
    });
    if (!closeLayer) {
      closeLayer = openLayer({
        el: p.dialog,
        modal: true,
        dismissOnOutside: true,
        initialFocus: p.input,
        onClose: () => {
          closeLayer = null;
          hide();
          if (state.get("overlay") === "search") state.set("overlay", null);
        },
      });
    }
  };

  const close = () => {
    if (closeLayer) {
      const c = closeLayer;
      closeLayer = null;
      c();
    }
    if (state.get("overlay") === "search") state.set("overlay", null);
    hide();
  };

  const hide = () => {
    if (!parts) return;
    const p = parts;
    seq++;
    inflight?.abort();
    p.scrim.classList.remove("is-visible");
    p.dialog.classList.remove("is-visible");
    hideTimer = setTimeout(() => {
      p.scrim.hidden = true;
      p.dialog.hidden = true;
    }, 180);
  };

  state.watch("overlay", (v) => (v === "search" ? open() : closeLayer ? close() : undefined));
  if (state.get("overlay") === "search") open();
}
