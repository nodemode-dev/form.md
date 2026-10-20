import type { Engine, Page } from "../engine";
import { formatRelative, searchFold, searchHit } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { simpleDialog } from "./dialog";
import { h, ref, setAttr, setText } from "./dom";
import { fallbackGlyph, pageIcon, paintIcon } from "./pageicon";
import type { StarterSummary, TemplateCatalog } from "./templates-dialog";

const RECENT_SHOWN = 8;
const TEMPLATES_SHOWN = 10;
const FEATURE_SHOWN = 6;
const SHOWCASE = ["charts", "examples"];
const ACCEPT = ".zip,.md,.markdown,.txt,.html,.htm,.enex,.xml,.docx,.xlsx,.csv,.tsv,.json,.opml,.pdf";

function patchCards<T>(host: HTMLElement, items: readonly T[], key: (x: T) => string, make: (x: T) => HTMLElement, update: (el: HTMLElement, x: T) => void): void {
  const existing = new Map<string, HTMLElement>();
  for (const c of Array.from(host.children) as HTMLElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
  let cursor: Element | null = host.firstElementChild;
  for (const x of items) {
    const k = key(x);
    let el = existing.get(k);
    if (el) existing.delete(k);
    else {
      el = make(x);
      el.dataset.key = k;
    }
    update(el, x);
    if (el === cursor) {
      cursor = cursor.nextElementSibling;
    } else if (cursor?.classList.contains("is-skeleton")) {
      const next: Element | null = cursor.nextElementSibling;
      cursor.replaceWith(el);
      cursor = next;
    } else host.insertBefore(el, cursor);
  }
  for (const el of existing.values()) el.remove();
  for (const el of Array.from(host.children)) if (el.classList.contains("is-skeleton")) el.remove();
}

function cardShell(...preview: HTMLElement[]): HTMLButtonElement {
  return h("button", { class: "hcard", attrs: { type: "button" } }, h("span", { class: "hcard__preview" }, ...preview), h("span", { class: "hcard__body" }, h("span", { class: "hcard__name" }), h("span", { class: "hcard__meta" })));
}

export function mountHome(engine: Engine): void {
  const root = document.documentElement;
  const recentGrid = ref("home-recent");
  const offline = ref("home-offline");
  const material = ref("home-material");
  const tplSec = ref("home-tpl-sec");
  const chips = ref("home-chips");
  const tplGrid = ref("home-tpl-grid");
  const tplEmpty = ref("home-tpl-empty");
  const tplSearch = ref<HTMLInputElement>("home-tpl-search");
  const featSec = ref("home-feat-sec");
  const featGrid = ref("home-feat-grid");
  const exSec = ref("home-ex-sec");
  const exGrid = ref("home-ex-grid");
  let recent: Page[] = [];
  let token = 0;
  let catalog: TemplateCatalog | null = null;
  let catalogWs = "";
  let category = "";
  let query = "";
  let tplJob: Promise<typeof import("./templates-dialog")> | null = null;
  const templatesModule = () => {
    tplJob ??= import("./templates-dialog");
    tplJob.catch(() => (tplJob = null));
    return tplJob;
  };

  const workspace = () => engine.state.get("workspaceId");
  const onHome = () => engine.state.get("route").name === "home";
  const newest = (ws: string) =>
    engine.pages
      .all(ws)
      .sort((a, b) => b.updated_at - a.updated_at)
      .slice(0, RECENT_SHOWN);
  const paintMode = () => {
    const ws = workspace();
    if (!ws || !engine.pages.isLoaded(ws)) return;
    setAttr(root, "data-home", engine.pages.all(ws).length ? "back" : "new");
  };
  const paintDaypart = () => {
    const hour = new Date().getHours();
    setAttr(root, "data-daypart", hour < 12 ? "morning" : hour < 18 ? "afternoon" : "evening");
  };
  const paintWrite = () => {
    const w = engine.canWrite();
    if (material.hidden === w) material.hidden = !w;
    if (tplSec.hidden === w) tplSec.hidden = !w;
    if (featSec.hidden === w) featSec.hidden = !w;
    if (exSec.hidden === w) exSec.hidden = !w;
  };

  const makeRecent = () => cardShell(h("span", { class: "hcard__doc" }, pageIcon("", "page"), h("span", { class: "hcard__doc-title" }), h("span", { class: "hcard__doc-text" })));
  const paintRecentCard = (el: HTMLElement, p: Page) => {
    el.dataset.id = p.id;
    const doc = el.querySelector(".hcard__doc") as HTMLElement;
    const title = p.title || t("Untitled");
    paintIcon(doc.children[0] as HTMLElement, p.icon, fallbackGlyph(p.kind));
    setText(doc.children[1] as HTMLElement, title);
    setText(doc.children[2] as HTMLElement, p.snippet || "");
    setAttr(doc, "data-empty", p.snippet ? null : "true");
    setText(el.querySelector(".hcard__name") as HTMLElement, title);
    setText(el.querySelector(".hcard__meta") as HTMLElement, p.updated_at ? t("Edited {time}", { time: formatRelative(p.updated_at) }) : "");
  };
  const paintRecent = (items: Page[]) => {
    if (!items.length && !engine.pages.isLoaded(workspace())) return;
    recent = items;
    patchCards(recentGrid, items, (p) => p.id, makeRecent, paintRecentCard);
  };
  const loadRecent = () => {
    const ws = workspace();
    if (!ws) return;
    const mine = ++token;
    engine.api.pages.recent(RECENT_SHOWN * 2).then(
      (r) => {
        if (mine !== token) return;
        const live = r.pages.filter((p) => p.workspace_id === ws && !p.trashed_at).slice(0, RECENT_SHOWN);
        paintRecent(live.length ? live.map((p) => engine.pages.get(p.id) ?? p) : newest(ws));
      },
      () => {
        if (mine === token) paintRecent(newest(ws));
      },
    );
  };

  const chosenName = () => catalog?.categories.find((c) => c.id === category)?.name ?? "";
  const categoryName = (name: string) => (name ? t(name) : "");
  const previews = new IntersectionObserver(
    (entries) => {
      for (const en of entries) {
        if (!en.isIntersecting) continue;
        const host = en.target as HTMLElement;
        previews.unobserve(host);
        const shot = host.querySelector<HTMLElement>(".hcard__shot");
        const id = host.parentElement?.dataset.template;
        if (shot && id) void templatesModule().then((m) => m.mountTemplatePreview(shot, engine, id), () => undefined);
      }
    },
    { rootMargin: "200px" },
  );
  const makeTemplate = (s: StarterSummary) => {
    const el = cardShell(h("span", { class: "hcard__glyph" }, pageIcon(s.icon, s.kind === "database" ? "table" : "page")), h("span", { class: "hcard__shot" }));
    el.dataset.template = s.id;
    previews.observe(el.firstElementChild as HTMLElement);
    return el;
  };
  const paintTemplateCard = (el: HTMLElement, s: StarterSummary) => {
    setText(el.querySelector(".hcard__name") as HTMLElement, s.name);
    setText(el.querySelector(".hcard__meta") as HTMLElement, [categoryName(s.category), s.kind === "database" ? t("Database") : t("Page")].filter(Boolean).join(" · "));
    setAttr(el, "title", s.description || null);
  };
  const makeRich = (cls: string) => (s: StarterSummary) => {
    const el = h(
      "button",
      { class: `hcard ${cls}`, attrs: { type: "button" } },
      h("span", { class: "hcard__preview" }, h("span", { class: "hcard__glyph" }, pageIcon(s.icon, s.kind === "database" ? "table" : "page")), h("span", { class: "hcard__shot" })),
      h("span", { class: "hcard__body" }, h("span", { class: "hcard__name" }), h("span", { class: "hcard__desc" }), h("span", { class: "hcard__meta" })),
    );
    el.dataset.template = s.id;
    previews.observe(el.firstElementChild as HTMLElement);
    return el;
  };
  const paintRich = (el: HTMLElement, s: StarterSummary) => {
    setText(el.querySelector(".hcard__name") as HTMLElement, s.name);
    setText(el.querySelector(".hcard__desc") as HTMLElement, s.description);
    setText(
      el.querySelector(".hcard__meta") as HTMLElement,
      s.databases !== undefined || s.docs !== undefined
        ? `${plural(s.docs ?? 0, "{count} page", "{count} pages")} · ${plural(s.databases ?? 0, "{count} database", "{count} databases")}`
        : s.pages
          ? plural(s.pages, "{count} page", "{count} pages")
          : s.kind === "database"
            ? t("Database")
            : t("Page"),
    );
  };
  const catName = (id: string) => catalog?.categories.find((c) => c.id === id)?.name ?? "";
  const paintShowcase = () => {
    if (!catalog) return;
    const charts = catName("charts");
    const examples = catName("examples");
    const feat = catalog.starters
      .filter((s) => s.category === charts)
      .sort((a, b) => Number(!!b.featured) - Number(!!a.featured))
      .slice(0, FEATURE_SHOWN);
    const projects = catalog.starters.filter((s) => s.category === examples);
    patchCards(featGrid, feat, (s) => s.id, makeRich("hcard--big"), paintRich);
    patchCards(exGrid, projects, (s) => s.id, makeRich("hcard--ex"), paintRich);
    setAttr(featSec, "data-empty", feat.length ? null : "true");
    setAttr(exSec, "data-empty", projects.length ? null : "true");
  };
  const matches = (s: StarterSummary) =>
    (!category || s.category === chosenName()) && (!query || searchHit(searchFold(`${s.name} ${s.description} ${s.category} ${categoryName(s.category)} ${s.tags.join(" ")}`), query));
  const paintTemplates = () => {
    if (!catalog) return;
    paintShowcase();
    const rest = catalog.categories.filter((c) => c.count > 0 && !SHOWCASE.includes(c.id));
    const pool = catalog.starters.filter((s) => rest.some((c) => c.name === s.category));
    const all = [{ id: "", name: t("All"), count: pool.length }, ...rest];
    patchCards(
      chips,
      all,
      (c) => `c:${c.id}`,
      () => h("button", { class: "home-chip", attrs: { type: "button" } }, h("span", { class: "home-chip__name" }), h("span", { class: "home-chip__count" })),
      (el, c) => {
        el.dataset.cat = c.id;
        setText(el.firstElementChild as HTMLElement, c.id ? t(c.name) : c.name);
        setText(el.lastElementChild as HTMLElement, String(c.count));
        setAttr(el, "aria-pressed", c.id === category ? "true" : "false");
      },
    );
    const list = pool.filter(matches).slice(0, TEMPLATES_SHOWN);
    patchCards(tplGrid, list, (s) => s.id, makeTemplate, paintTemplateCard);
    if (tplEmpty.hidden !== list.length > 0) tplEmpty.hidden = list.length > 0;
  };
  const loadTemplates = () => {
    const ws = workspace();
    if (!ws || tplSec.hidden || !onHome() || catalogWs === ws) return;
    catalogWs = ws;
    void templatesModule()
      .then((m) => m.loadTemplateCatalog(engine))
      .then(
        (c) => {
          if (catalogWs !== ws) return;
          catalog = c;
          if (category && !c.categories.some((x) => x.id === category)) category = "";
          paintTemplates();
        },
        () => {
          if (catalogWs === ws) catalogWs = "";
        },
      );
  };

  const handFiles = (files: File[]) => {
    if (!files.length) return;
    void import("./transfer-dialog").then((m) => {
      m.openImport(engine, { workspaceId: workspace() });
      const pickers = document.querySelectorAll<HTMLInputElement>(".xfer__file");
      const pick = pickers[pickers.length - 1];
      if (!pick || typeof DataTransfer === "undefined") return;
      const dt = new DataTransfer();
      for (const f of files) dt.items.add(f);
      pick.files = dt.files;
      pick.dispatchEvent(new Event("change"));
    }, engine.reportError);
  };
  const chooseFiles = () => {
    const input = h("input", { attrs: { type: "file", multiple: "", accept: ACCEPT } });
    input.addEventListener("change", () => handFiles(Array.from(input.files ?? [])));
    input.click();
  };
  const pasteText = () =>
    simpleDialog(
      t("Paste text"),
      (body, close) => {
        const area = h("textarea", { class: "input", attrs: { rows: "12", "aria-label": t("Text to import"), placeholder: t("Paste notes, an outline or Markdown") } });
        const cancel = h("button", { class: "btn btn--ghost", attrs: { type: "button" }, text: t("Cancel") });
        const go = h("button", { class: "btn btn--primary", attrs: { type: "button" }, text: t("Continue") });
        cancel.addEventListener("click", close);
        go.addEventListener("click", () => {
          const text = area.value.trim();
          close();
          if (text) handFiles([new File([text], "pasted.md", { type: "text/markdown" })]);
        });
        body.append(area, h("div", { class: "form-actions" }, cancel, go));
        return area;
      },
      undefined,
      "md",
    );

  recentGrid.addEventListener("click", (e) => {
    const id = (e.target as HTMLElement).closest<HTMLElement>(".hcard[data-id]")?.dataset.id;
    if (id) engine.openPage(id, { newTab: engine.platform.mac ? e.metaKey : e.ctrlKey });
  });
  ref("home-see-all").addEventListener("click", () => engine.navigate({ name: "library", view: "recent" }));
  chips.addEventListener("click", (e) => {
    const chip = (e.target as HTMLElement).closest<HTMLElement>(".home-chip");
    if (!chip || chip.dataset.cat === undefined) return;
    category = chip.dataset.cat;
    paintTemplates();
  });
  tplSearch.addEventListener("input", () => {
    query = searchFold(tplSearch.value).trim();
    paintTemplates();
  });
  tplGrid.addEventListener("click", (e) => {
    const id = (e.target as HTMLElement).closest<HTMLElement>(".hcard[data-template]")?.dataset.template;
    if (id) void templatesModule().then((m) => m.openTemplates(engine, {}, { preview: id }), engine.reportError);
  });
  const openPreview = (e: Event) => {
    const id = (e.target as HTMLElement).closest<HTMLElement>(".hcard[data-template]")?.dataset.template;
    if (id) void templatesModule().then((m) => m.openTemplates(engine, {}, { preview: id }), engine.reportError);
  };
  featGrid.addEventListener("click", openPreview);
  exGrid.addEventListener("click", openPreview);
  ref("home-feat-all").addEventListener("click", () => void templatesModule().then((m) => m.openTemplates(engine, {}, { category: "charts" }), engine.reportError));
  ref("home-ex-all").addEventListener("click", () => void templatesModule().then((m) => m.openTemplates(engine, {}, { category: "examples" }), engine.reportError));
  ref("home-templates").addEventListener("click", () => void templatesModule().then((m) => m.openTemplates(engine, {}, { ...(category ? { category } : {}), ...(query ? { query: tplSearch.value.trim() } : {}) }), engine.reportError));
  ref("home-choose").addEventListener("click", chooseFiles);
  ref("home-paste").addEventListener("click", pasteText);
  material.addEventListener("dragover", (e) => {
    if (!e.dataTransfer?.types.includes("Files")) return;
    e.preventDefault();
    setAttr(material, "data-over", "true");
  });
  material.addEventListener("dragleave", (e) => {
    if (!material.contains(e.relatedTarget as Node | null)) setAttr(material, "data-over", null);
  });
  material.addEventListener("drop", (e) => {
    e.preventDefault();
    setAttr(material, "data-over", null);
    handFiles(Array.from(e.dataTransfer?.files ?? []));
  });

  engine.pages.subscribe((c) => {
    if (c.kind === "loaded" || c.kind === "upsert" || c.kind === "remove") paintMode();
    if (c.kind === "upsert" && recent.some((p) => p.id === c.page.id)) paintRecent(recent.map((p) => (p.id === c.page.id ? c.page : p)));
    else if (c.kind === "remove" && c.pages.some((x) => recent.some((p) => p.id === x.id))) paintRecent(recent.filter((p) => !c.pages.some((x) => x.id === p.id)));
  });
  engine.state.watch("workspaceId", () => {
    paintMode();
    paintWrite();
    loadRecent();
    loadTemplates();
  });
  engine.state.watch("me", () => {
    paintWrite();
    loadTemplates();
  });
  engine.state.watch("route", (r) => {
    if (r.name !== "home") return;
    paintDaypart();
    loadRecent();
    loadTemplates();
  });
  engine.state.watch("health", (health) => {
    offline.hidden = health.phase !== "down";
  });
  paintMode();
  paintWrite();
  loadRecent();
  loadTemplates();
}
