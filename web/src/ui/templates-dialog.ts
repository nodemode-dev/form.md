import type { Engine } from "../engine";
import { formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import type { Page } from "../engine/types";
import { TYPE_INFO, VIEW_INFO } from "../extensions/database/props";
import { simpleDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { button, field, input, status } from "./form";
import { type IconName, iconNode } from "./icons";
import { pageIcon, paintIcon } from "./pageicon";

export interface StarterSummary {
  id: string;
  name: string;
  description: string;
  icon: string;
  kind: "doc" | "database";
  category: string;
  tags: string[];
}

interface PreviewItem {
  name: string;
  type: string;
}

export interface TemplatePreview {
  id: string;
  kind: "doc" | "database";
  title: string;
  icon: string;
  description: string;
  category: string;
  html: string;
  sections?: string[];
  views?: PreviewItem[];
  properties?: PreviewItem[];
  rows?: number;
}

export interface TemplateCategory {
  id: string;
  name: string;
  count: number;
}

export interface TemplateCatalog {
  categories: TemplateCategory[];
  starters: StarterSummary[];
  workspace: WorkspaceTemplate[];
}

export interface TemplateOpenOptions {
  category?: string;
  query?: string;
  preview?: string;
}

export interface WorkspaceTemplate {
  id: string;
  title: string;
  icon: string;
  kind: "doc" | "collection";
  updated_at: number;
  can_edit: boolean;
}

export interface TemplateTarget {
  parentId?: string;
  teamspaceId?: string;
}

export function useTemplate(engine: Engine, body: { starter?: string; page?: string; title?: string }, target: TemplateTarget = {}): Promise<Page> {
  const ws = engine.state.get("workspaceId");
  return engine.api.http
    .post<{ page: Page }>("/api/templates/use", { workspace_id: ws, parent_id: target.parentId ?? "", teamspace_id: target.teamspaceId ?? "", ...body })
    .then((r) => {
      engine.pages.upsert(r.page);
      return r.page;
    });
}

const catalogs = new Map<string, Promise<TemplateCatalog>>();
const previews = new Map<string, Promise<TemplatePreview>>();
const ready = new Map<string, TemplatePreview>();
const PAGE_W = 720;
const WS_CAT = "@workspace";

function fetchCatalog(engine: Engine, ws: string): Promise<TemplateCatalog> {
  return engine.api.http.get<TemplateCatalog>(`/api/templates?workspace=${encodeURIComponent(ws)}`).then((g) => ({
    categories: g.categories ?? [],
    starters: (g.starters ?? []).map((s) => ({ ...s, tags: s.tags ?? [] })),
    workspace: g.workspace ?? [],
  }));
}

export function loadTemplateCatalog(engine: Engine): Promise<TemplateCatalog> {
  const ws = engine.state.get("workspaceId");
  let p = catalogs.get(ws);
  if (!p) {
    p = fetchCatalog(engine, ws);
    p.catch(() => catalogs.delete(ws));
    catalogs.set(ws, p);
  }
  return p;
}

function loadPreview(engine: Engine, id: string): Promise<TemplatePreview> {
  let p = previews.get(id);
  if (!p) {
    p = engine.api.http.get<TemplatePreview>(`/api/templates/${encodeURIComponent(id)}/preview`).then((r) => {
      ready.set(id, r);
      return r;
    });
    p.catch(() => previews.delete(id));
    previews.set(id, p);
  }
  return p;
}

interface Fit {
  page: HTMLElement;
  sizer: HTMLElement;
  w: number;
}

const fits = new WeakMap<Element, Fit>();
let resizer: ResizeObserver | null = null;

function rescale(f: Fit, w: number): void {
  if (w <= 0) return;
  f.w = w;
  const s = w / PAGE_W;
  f.page.style.transform = `scale(${s})`;
  f.sizer.style.blockSize = `${Math.ceil(f.page.offsetHeight * s)}px`;
}

function watchSize(host: HTMLElement): void {
  resizer ??= new ResizeObserver((entries) => {
    for (const en of entries) {
      const f = fits.get(en.target);
      if (!en.target.isConnected) resizer?.unobserve(en.target);
      else if (f && Math.abs(en.contentRect.width - f.w) > 0.5) rescale(f, en.contentRect.width);
    }
  });
  resizer.observe(host);
}

function paintPreview(host: HTMLElement, p: TemplatePreview): void {
  if (host.dataset.preview !== p.id) return;
  const head = h("div", { class: "tplprev__head" }, pageIcon(p.icon, p.kind === "database" ? "table" : "page", "tplprev__icon"), h("div", { class: "tplprev__title", text: p.title }));
  const body = h("div", { class: `tplp tplp--${p.kind}` });
  body.innerHTML = p.html;
  const page = h("div", { class: "tplprev__page" }, head, body);
  const sizer = h("div", { class: "tplprev__sizer" }, page);
  host.replaceChildren(sizer);
  const f: Fit = { page, sizer, w: 0 };
  fits.set(host, f);
  rescale(f, host.clientWidth);
  watchSize(host);
}

export function mountTemplatePreview(host: HTMLElement, engine: Engine, id: string): void {
  if (!host.classList.contains("tplprev")) host.classList.add("tplprev");
  setAttr(host, "aria-hidden", "true");
  if (host.dataset.preview === id && host.firstElementChild) return;
  host.dataset.preview = id;
  const done = ready.get(id);
  if (done) {
    paintPreview(host, done);
    return;
  }
  if (host.firstElementChild) host.replaceChildren();
  setAttr(host, "aria-busy", "true");
  loadPreview(engine, id)
    .then((p) => {
      setAttr(host, "aria-busy", null);
      paintPreview(host, p);
    })
    .catch(() => {
      setAttr(host, "aria-busy", null);
      host.classList.add("is-failed");
    });
}

function kindLabel(kind: string): string {
  return kind === "database" || kind === "collection" ? t("Database") : t("Page");
}

function viewIcon(type: string): IconName {
  return (VIEW_INFO as Record<string, { icon: IconName } | undefined>)[type]?.icon ?? "table";
}

function propIcon(type: string): IconName {
  return (TYPE_INFO as Record<string, { icon: IconName } | undefined>)[type]?.icon ?? "text";
}

function insideGroup(label: string, entries: [IconName, string][]): HTMLElement {
  return h(
    "div",
    { class: "tplgal__igroup" },
    h("div", { class: "tplgal__ilabel", text: label }),
    h("ul", { class: "tplgal__ilist" }, ...entries.map(([icon, name]) => h("li", {}, iconNode(icon), h("span", { text: name })))),
  );
}

interface Item {
  starter?: StarterSummary;
  ws?: WorkspaceTemplate;
  text: string;
  card: HTMLButtonElement;
  update?: (w: WorkspaceTemplate) => void;
}

export function openTemplates(engine: Engine, target: TemplateTarget = {}, opts: TemplateOpenOptions = {}): void {
  const ws = engine.state.get("workspaceId");
  let lazy: IntersectionObserver | null = null;
  simpleDialog(
    t("Templates"),
    (body, close) => {
      const dlg = body.parentElement ?? body;
      dlg.classList.add("dialog--tplgal");
      const loadSt = status();
      const st = status();
      const searchInput = h("input", { class: "tplgal__input", attrs: { type: "search", placeholder: t("Search templates"), "aria-label": t("Search templates"), spellcheck: "false", autocomplete: "off" } });
      const cats = h("div", { class: "tplgal__cats", attrs: { role: "group", "aria-label": t("Template categories") } });
      const rail = h("aside", { class: "tplgal__rail" }, h("label", { class: "tplgal__search" }, iconNode("search"), searchInput), cats);
      const heading = h("h3", { class: "tplgal__heading", text: t("All templates") });
      const shown = h("span", { class: "tplgal__shown" });
      const grid = h("div", { class: "tplgal__grid", attrs: { "aria-label": t("Templates") } });
      const empty = h("p", { class: "tplgal__empty", text: t("No templates match that search.") });
      const wsEmpty = h("p", { class: "tplgal__empty", text: t("Pages saved as templates show up here. Use Save as template in a page's menu.") });
      empty.hidden = true;
      wsEmpty.hidden = true;
      const browse = h("div", { class: "tplgal__browse" }, h("div", { class: "tplgal__bar" }, heading, shown), grid, empty, wsEmpty, loadSt.el);
      const big = h("div", { class: "tplgal__big tplprev--scroll" });
      const bigWs = h("div", { class: "tplgal__big tplgal__big--ws" });
      bigWs.hidden = true;
      const dIcon = h("span", { class: "page-icon tplgal__dicon", attrs: { "aria-hidden": "true" } });
      const dName = h("h3", { class: "tplgal__dname" });
      const dMeta = h("p", { class: "tplgal__dmeta" });
      const dDesc = h("p", { class: "tplgal__ddesc" });
      const inside = h("div", { class: "tplgal__inside" });
      const titleInput = input({ maxlength: 200 });
      const useBtn = button(t("Use this template"), "primary");
      const backBtn = button(t("Back to templates"), "ghost");
      const info = h(
        "div",
        { class: "tplgal__info" },
        h("div", { class: "tplgal__about" }, h("div", { class: "tplgal__dhead" }, dIcon, dName), dMeta, dDesc, inside),
        h("div", { class: "tplgal__act" }, field(t("Title"), titleInput), h("div", { class: "tplgal__btns" }, useBtn, backBtn), st.el),
      );
      const detail = h("div", { class: "tplgal__detail" }, h("div", { class: "tplgal__stage" }, big, bigWs), info);
      detail.hidden = true;
      body.append(h("div", { class: "tplgal" }, rail, h("section", { class: "tplgal__main" }, browse, detail)));

      let cat = "";
      let words: string[] = [];
      let items: Item[] = [];
      let current: Item | null = null;
      let lastCard: HTMLElement | null = null;
      const catButtons = new Map<string, { btn: HTMLButtonElement; count: HTMLElement }>();
      const byCard = new WeakMap<Element, Item>();

      lazy = new IntersectionObserver(
        (entries) => {
          for (const en of entries) {
            if (!en.isIntersecting) continue;
            const box = en.target as HTMLElement;
            lazy?.unobserve(box);
            if (box.dataset.tpl) mountTemplatePreview(box, engine, box.dataset.tpl);
          }
        },
        { root: browse, rootMargin: "240px 0px" },
      );

      const cardShell = (box: HTMLElement, icon: HTMLElement, label: HTMLElement, meta: HTMLElement) =>
        h("button", { class: "tplcard", attrs: { type: "button" } }, box, h("span", { class: "tplcard__body" }, h("span", { class: "tplcard__name" }, icon, label), meta));

      const starterCard = (s: StarterSummary): Item => {
        const box = h("span", { class: "tplcard__prev tplprev" });
        box.dataset.tpl = s.id;
        const card = cardShell(box, pageIcon(s.icon, s.kind === "database" ? "table" : "page", "tplcard__icon"), h("span", { class: "tplcard__label", text: s.name }), h("span", { class: "tplcard__meta", text: `${t(s.category)} · ${kindLabel(s.kind)}` }));
        lazy?.observe(box);
        const it: Item = { starter: s, card, text: [s.name, s.description, s.category, t(s.category), ...s.tags].join(" ").toLowerCase() };
        byCard.set(card, it);
        return it;
      };

      const wsCard = (w: WorkspaceTemplate): Item => {
        const fallback = w.kind === "collection" ? "table" : "page";
        const glyph = pageIcon(w.icon, fallback, "tplcard__wsicon");
        const small = pageIcon(w.icon, fallback, "tplcard__icon");
        const label = h("span", { class: "tplcard__label" });
        const meta = h("span", { class: "tplcard__meta" });
        const card = cardShell(h("span", { class: "tplcard__prev tplcard__prev--ws" }, glyph), small, label, meta);
        const it: Item = { ws: w, card, text: "" };
        it.update = (n) => {
          it.ws = n;
          const fb = n.kind === "collection" ? "table" : "page";
          paintIcon(glyph, n.icon, fb);
          paintIcon(small, n.icon, fb);
          setText(label, n.title || t("Untitled"));
          setText(meta, `${t("From this workspace")} · ${kindLabel(n.kind)}`);
          it.text = `${n.title} ${t("From this workspace")}`.toLowerCase();
        };
        it.update(w);
        byCard.set(card, it);
        return it;
      };

      const inCat = (it: Item, c: string) => (c === "" ? true : c === WS_CAT ? !!it.ws : it.starter?.category === c);
      const visibleCards = () => items.map((it) => it.card).filter((c) => !c.hidden);

      const filter = () => {
        let n = 0;
        const counts = new Map<string, number>();
        for (const it of items) {
          const hit = words.every((w) => it.text.includes(w));
          if (hit) {
            const k = it.ws ? WS_CAT : (it.starter?.category ?? "");
            counts.set(k, (counts.get(k) ?? 0) + 1);
            counts.set("", (counts.get("") ?? 0) + 1);
          }
          const ok = hit && inCat(it, cat);
          if (it.card.hidden !== !ok) it.card.hidden = !ok;
          if (ok) n++;
        }
        for (const [k, b] of catButtons) {
          setText(b.count, String(counts.get(k) ?? 0));
          setAttr(b.btn, "aria-pressed", k === cat ? "true" : "false");
        }
        setText(heading, cat === "" ? t("All templates") : cat === WS_CAT ? t("From this workspace") : t(cat));
        setText(shown, String(n));
        const noWs = cat === WS_CAT && !items.some((it) => it.ws);
        if (wsEmpty.hidden !== !noWs) wsEmpty.hidden = !noWs;
        const none = n > 0 || noWs || items.length === 0;
        if (empty.hidden !== none) empty.hidden = none;
      };

      const fillInside = (it: Item, p: TemplatePreview) => {
        if (current !== it) return;
        const groups: HTMLElement[] = [];
        if (p.kind === "database") {
          groups.push(insideGroup(t("Views"), (p.views ?? []).map((v): [IconName, string] => [viewIcon(v.type), v.name])));
          groups.push(insideGroup(t("Properties"), (p.properties ?? []).map((x): [IconName, string] => [propIcon(x.type), x.name])));
          if (p.rows) groups.push(h("p", { class: "tplgal__note", text: plural(p.rows, "{count} sample row", "{count} sample rows") }));
        } else if (p.sections?.length) {
          groups.push(insideGroup(t("Sections"), p.sections.map((x): [IconName, string] => ["h2", x])));
        }
        inside.replaceChildren(...(groups.length ? [h("h4", { class: "tplgal__ihead", text: t("What's inside") }), ...groups] : []));
      };

      const showGrid = (focus: boolean) => {
        if (detail.hidden) return;
        detail.hidden = true;
        browse.hidden = false;
        setAttr(dlg, "data-own-keys", null);
        current = null;
        st.clear();
        if (focus && lastCard && !lastCard.hidden) lastCard.focus();
      };

      const openDetail = (it: Item) => {
        current = it;
        lastCard = it.card;
        st.clear();
        setAttr(useBtn, "aria-busy", null);
        const s = it.starter;
        const w = it.ws;
        const name = s ? s.name : w?.title || t("Untitled");
        paintIcon(dIcon, s ? s.icon : (w?.icon ?? ""), (s ? s.kind === "database" : w?.kind === "collection") ? "table" : "page");
        setText(dName, name);
        setText(dMeta, s ? `${t(s.category)} · ${kindLabel(s.kind)}` : w ? `${t("From this workspace")} · ${kindLabel(w.kind)} · ${t("Updated {when}", { when: formatRelative(w.updated_at) })}` : "");
        setText(dDesc, s?.description ?? "");
        dDesc.hidden = !s;
        titleInput.value = name;
        if (big.hidden !== !s) big.hidden = !s;
        if (bigWs.hidden !== !!s) bigWs.hidden = !!s;
        browse.hidden = true;
        detail.hidden = false;
        if (s) {
          mountTemplatePreview(big, engine, s.id);
          big.scrollTop = 0;
          const done = ready.get(s.id);
          if (done) fillInside(it, done);
          else {
            inside.replaceChildren();
            loadPreview(engine, s.id)
              .then((p) => fillInside(it, p))
              .catch(() => undefined);
          }
        } else if (w) {
          bigWs.replaceChildren(pageIcon(w.icon, w.kind === "collection" ? "table" : "page", "tplgal__wsicon"));
          inside.replaceChildren();
        }
        setAttr(dlg, "data-own-keys", "templates.preview");
        useBtn.focus();
      };

      const run = () => {
        const it = current;
        if (!it || useBtn.getAttribute("aria-busy")) return;
        setAttr(useBtn, "aria-busy", "true");
        const title = titleInput.value.trim();
        useTemplate(engine, it.starter ? { starter: it.starter.id, title } : { page: it.ws?.id ?? "", title }, target)
          .then((p) => {
            close();
            engine.openPage(p.id);
          })
          .catch((e: unknown) => {
            setAttr(useBtn, "aria-busy", null);
            st.error(e);
          });
      };

      const catBtn = (key: string, label: string) => {
        const count = h("span", { class: "tplgal__count" });
        const btn = h("button", { class: "tplgal__cat", attrs: { type: "button", "aria-pressed": "false" } }, h("span", { class: "tplgal__catname", text: label }), count);
        btn.addEventListener("click", () => {
          cat = key;
          showGrid(false);
          filter();
          browse.scrollTop = 0;
        });
        catButtons.set(key, { btn, count });
        return btn;
      };

      const patchWs = (list: WorkspaceTemplate[]) => {
        const old = new Map(items.filter((it) => it.ws).map((it) => [it.ws?.id ?? "", it]));
        const next = list.map((w) => {
          const ex = old.get(w.id);
          old.delete(w.id);
          if (ex) ex.update?.(w);
          return ex ?? wsCard(w);
        });
        for (const gone of old.values()) gone.card.remove();
        const anchor = items.find((it) => it.starter)?.card ?? null;
        const inPlace = next.every((it, i) => grid.children[i] === it.card);
        if (!inPlace) for (const it of next) grid.insertBefore(it.card, anchor);
        items = [...next, ...items.filter((it) => it.starter)];
        filter();
      };

      const build = (g: TemplateCatalog) => {
        cats.replaceChildren(catBtn("", t("All")), ...g.categories.map((c) => catBtn(c.name, t(c.name))), h("div", { class: "tplgal__sep", attrs: { role: "separator" } }), catBtn(WS_CAT, t("From this workspace")));
        items = [...g.workspace.map(wsCard), ...g.starters.map(starterCard)];
        grid.append(...items.map((it) => it.card));
        const want = (opts.category ?? "").toLowerCase();
        if (want === "workspace" || want === WS_CAT) cat = WS_CAT;
        else if (want) cat = g.categories.find((c) => c.id === want || c.name.toLowerCase() === want)?.name ?? "";
        if (opts.query) {
          searchInput.value = opts.query;
          words = opts.query.toLowerCase().split(/\s+/).filter(Boolean);
        }
        filter();
        const pick = opts.preview ? items.find((it) => it.starter?.id === opts.preview || it.ws?.id === opts.preview) : undefined;
        if (pick) openDetail(pick);
      };

      grid.addEventListener("click", (e) => {
        const card = (e.target as Element).closest(".tplcard");
        const it = card ? byCard.get(card) : undefined;
        if (it) openDetail(it);
      });
      grid.addEventListener("keydown", (e) => {
        const card = (e.target as Element).closest<HTMLElement>(".tplcard");
        if (!card) return;
        const vis = visibleCards();
        const i = vis.indexOf(card as HTMLButtonElement);
        let cols = 1;
        while (cols < vis.length && vis[cols]?.offsetTop === vis[0]?.offsetTop) cols++;
        const step: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: cols, ArrowUp: -cols };
        let j: number;
        if (e.key in step) j = i + (step[e.key] ?? 0);
        else if (e.key === "Home") j = 0;
        else if (e.key === "End") j = vis.length - 1;
        else return;
        e.preventDefault();
        if (j < 0 && e.key === "ArrowUp") {
          searchInput.focus();
          return;
        }
        const next = vis[Math.max(0, Math.min(vis.length - 1, j))];
        next?.focus();
        next?.scrollIntoView({ block: "nearest" });
      });
      searchInput.addEventListener("input", () => {
        words = searchInput.value.toLowerCase().split(/\s+/).filter(Boolean);
        showGrid(false);
        filter();
        browse.scrollTop = 0;
      });
      searchInput.addEventListener("keydown", (e) => {
        const first = visibleCards()[0];
        if (e.key === "ArrowDown" && first) {
          e.preventDefault();
          first.focus();
        } else if (e.key === "Enter" && first) {
          e.preventDefault();
          const it = byCard.get(first);
          if (it) openDetail(it);
        }
      });
      useBtn.addEventListener("click", run);
      backBtn.addEventListener("click", () => showGrid(true));
      titleInput.addEventListener("keydown", (e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          run();
        }
      });
      dlg.addEventListener("keydown", (e) => {
        if (e.key === "Escape" && !detail.hidden) {
          e.preventDefault();
          e.stopPropagation();
          showGrid(true);
        }
      });

      const cached = catalogs.has(ws);
      loadTemplateCatalog(engine)
        .then((g) => {
          build(g);
          if (cached)
            fetchCatalog(engine, ws)
              .then((fresh) => {
                catalogs.set(ws, Promise.resolve(fresh));
                patchWs(fresh.workspace);
              })
              .catch(() => undefined);
        })
        .catch((e: unknown) => loadSt.error(e));
      return searchInput;
    },
    () => lazy?.disconnect(),
    "md",
  );
}
