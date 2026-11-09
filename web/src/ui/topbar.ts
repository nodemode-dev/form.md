import type { Engine, Page } from "../engine";
import { ApiError } from "../engine/api";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { tile, updateTile } from "./avatar";
import { h, ref, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { menuButton } from "./menu";
import { openShareDialog, pageMenu, setVisibility } from "./page-actions";
import { fallbackGlyph, pageIcon, paintIcon } from "./pageicon";
import { focusTitleSoon } from "./tree";

interface Crumb {
  key: string;
  label: string;
  icon: string;
  kind: string;
  target: string;
  current: boolean;
}

export function mountTopbar(engine: Engine): void {
  const { state, pages } = engine;
  const crumbs = ref("crumbs");
  const actions = ref("page-actions");
  const shareBtn = ref<HTMLButtonElement>("act-share");
  const visBtn = ref<HTMLButtonElement>("act-visibility");
  const favBtn = ref<HTMLButtonElement>("act-favorite");
  const moreBtn = ref<HTMLButtonElement>("act-more");
  const tabbar = ref("tabbar");
  const banner = ref("banner");

  const currentPage = (): Page | null => {
    const r = state.get("route");
    if (r.name !== "page") return null;
    const p = pages.get(r.id);
    if (p) return p;
    const d = state.get("detail");
    return d && d.id === r.id ? d : null;
  };

  const crumbList = (): Crumb[] => {
    const r = state.get("route");
    const w = engine.workspace();
    const wsCrumb: Crumb = { key: "ws", label: w?.name ?? "", icon: "", kind: "ws", target: "", current: r.name === "home" };
    if (r.name === "library") return [{ key: "library", label: t("Library"), icon: "", kind: "library", target: "", current: true }];
    if (r.name === "trash") return [wsCrumb, { key: "trash", label: t("Trash"), icon: "", kind: "trash", target: "", current: true }];
    if (r.name === "inbox") return [{ key: "inbox", label: t("Inbox"), icon: "", kind: "inbox", target: "", current: true }];
    if (r.name !== "page") return [wsCrumb];
    const p = currentPage();
    if (!p) return [];
    const d = state.get("detail");
    const chain = pages.isLoaded(p.workspace_id) ? pages.ancestors(p.id).map((a) => ({ id: a.id, title: a.title, icon: a.icon, kind: a.kind })) : d && d.id === p.id ? d.path.map((c) => ({ ...c, kind: "doc" })) : [];
    const out: Crumb[] = [];
    const shown = chain.length > 3 ? [chain[0], null, ...chain.slice(-2)] : chain;
    for (const c of shown) {
      if (!c) out.push({ key: "gap", label: "…", icon: "", kind: "gap", target: "", current: false });
      else out.push({ key: c.id, label: c.title || t("Untitled"), icon: c.icon, kind: c.kind, target: c.id, current: false });
    }
    out.push({ key: p.id, label: p.title || t("Untitled"), icon: p.icon, kind: p.kind, target: p.id, current: true });
    return out;
  };

  const paintCrumbs = () => {
    const list = crumbList();
    const existing = new Map<string, HTMLElement>();
    for (const child of Array.from(crumbs.children) as HTMLElement[]) if (child.dataset.key) existing.set(child.dataset.key, child);
    let cursor: Element | null = crumbs.firstElementChild;
    list.forEach((c, i) => {
      let el = existing.get(c.key);
      if (el) existing.delete(c.key);
      else {
        const lead = c.kind === "ws" ? tile("") : c.kind === "library" ? h("span", { class: "crumbs__glyph" }, iconNode("library")) : c.kind === "trash" ? h("span", { class: "crumbs__glyph" }, iconNode("trash")) : c.kind === "inbox" ? h("span", { class: "crumbs__glyph" }, iconNode("inbox")) : c.kind === "gap" ? h("span") : pageIcon(c.icon, fallbackGlyph(c.kind));
        el = h("button", { class: "crumbs__item", attrs: { type: "button" } }, h("span", { class: "crumbs__sep", attrs: { "aria-hidden": "true" } }, iconNode("chevron-right")), lead, h("span", { class: "crumbs__label" }));
        el.dataset.key = c.key;
      }
      el.dataset.target = c.target;
      el.dataset.kind = c.kind;
      setAttr(el, "data-first", i === 0 ? "true" : null);
      setAttr(el, "aria-current", c.current ? "page" : null);
      setText(el.querySelector(".crumbs__label") as HTMLElement, c.label);
      const lead = el.children[1] as HTMLElement;
      if (c.kind === "ws") {
        const w = engine.workspace();
        updateTile(lead, w?.name ?? "", w?.icon ?? "", w?.image ?? "");
      } else if (lead.classList.contains("page-icon")) paintIcon(lead, c.icon, fallbackGlyph(c.kind));
      if (el !== cursor) crumbs.insertBefore(el, cursor);
      else cursor = cursor.nextElementSibling;
    });
    for (const el of existing.values()) el.remove();
  };

  crumbs.addEventListener("click", (e) => {
    const el = (e.target as HTMLElement).closest<HTMLElement>(".crumbs__item");
    if (!el || el.getAttribute("aria-current") === "page") return;
    if (el.dataset.kind === "ws") engine.navigate({ name: "home" });
    else if (el.dataset.target) engine.openPage(el.dataset.target);
  });

  const paintActions = () => {
    const p = currentPage();
    setAttr(actions, "data-hidden", p ? null : "true");
    if (!p) return;
    const writable = engine.canWrite(p.workspace_id) && engine.can(p.id, "edit");
    const personal = engine.workspaceById(p.workspace_id)?.personal ?? true;
    visBtn.hidden = personal || !writable;
    shareBtn.hidden = personal;
    setText(shareBtn, t("Share"));
    const priv = p.visibility === "private";
    const visLabel = priv ? t("Private, only you can see it. Click to share with the workspace.") : t("Visible to the workspace. Click to make it private.");
    if (visBtn.dataset.state !== (priv ? "private" : "shared")) {
      visBtn.dataset.state = priv ? "private" : "shared";
      visBtn.replaceChildren(iconNode(priv ? "lock" : "users"));
    }
    setAttr(visBtn, "aria-label", visLabel);
    setAttr(visBtn, "data-tip", visLabel);
    const fav = pages.isFavorite(p.id);
    if (favBtn.getAttribute("aria-pressed") !== (fav ? "true" : "false")) {
      setAttr(favBtn, "aria-pressed", fav ? "true" : "false");
      favBtn.replaceChildren(iconNode(fav ? "star-fill" : "star"));
    }
    const favLabel = fav ? t("Remove from favorites") : t("Add to favorites");
    setAttr(favBtn, "aria-label", favLabel);
    setAttr(favBtn, "data-tip", favLabel);
    favBtn.dataset.command = "page.favorite";
  };

  visBtn.addEventListener("click", () => {
    const p = currentPage();
    if (p) void setVisibility(engine, p.id, p.visibility !== "private");
  });
  shareBtn.addEventListener("click", () => {
    const p = currentPage();
    if (p) openShareDialog(engine, p.id);
  });
  favBtn.addEventListener("click", (e) => {
    e.stopPropagation();
    engine.commands.run("page.favorite");
  });
  menuButton(
    moreBtn,
    () => {
      const p = currentPage();
      if (!p) return [];
      return pageMenu(engine, p.id, {
        addInside: () =>
          void pages
            .create(p.workspace_id, p.id)
            .then((c) => {
              engine.openPage(c.id);
              focusTitleSoon();
            })
            .catch(engine.reportError),
        trash: () => engine.commands.run("page.trash"),
      });
    },
    { side: "bottom", align: "end", label: t("Page actions") },
  );

  const fetching = new Set<string>();
  const tabPage = (id: string): Page | undefined => {
    const p = pages.get(id);
    if (p) return p;
    const d = state.get("detail");
    if (d && d.id === id) return d;
    if (!fetching.has(id)) {
      fetching.add(id);
      void engine.api.pages
        .get(id)
        .then((r) => pages.upsert(r.page))
        .catch((e: unknown) => {
          if (e instanceof ApiError && (e.status === 404 || e.status === 403)) engine.closeTab(id);
        })
        .finally(() => fetching.delete(id));
    }
    return undefined;
  };

  const paintTabs = () => {
    const tabs = state.get("tabs");
    const r = state.get("route");
    const show = tabs.length > 1;
    setAttr(tabbar, "data-open", show ? "true" : null);
    setAttr(document.documentElement, "data-tabbar", null);
    if (!show) {
      if (tabbar.firstElementChild) tabbar.replaceChildren();
      return;
    }
    const existing = new Map<string, HTMLElement>();
    for (const child of Array.from(tabbar.children) as HTMLElement[]) if (child.dataset.key) existing.set(child.dataset.key, child);
    let cursor: Element | null = tabbar.firstElementChild;
    for (const id of tabs) {
      let el = existing.get(id);
      if (el) existing.delete(id);
      else {
        el = h(
          "div",
          { class: "tab", attrs: { role: "tab", tabindex: "-1", "data-id": id } },
          pageIcon("", "page"),
          h("span", { class: "tab__title" }),
          h("button", { class: "tab__close", attrs: { type: "button", tabindex: "-1", "data-action": "close" } }, iconNode("close")),
        );
        el.dataset.key = id;
      }
      const p = tabPage(id);
      const title = p ? p.title || t("Untitled") : "";
      setText(el.querySelector(".tab__title") as HTMLElement, title);
      setAttr(el, "title", title);
      setAttr(el.querySelector(".tab__close") as HTMLElement, "aria-label", t("Close tab {title}", { title }));
      paintIcon(el.firstElementChild as HTMLElement, p?.icon ?? "", fallbackGlyph(p?.kind ?? "doc"));
      const active = r.name === "page" && r.id === id;
      setAttr(el, "aria-selected", active ? "true" : "false");
      el.tabIndex = active ? 0 : -1;
      if (el !== cursor) tabbar.insertBefore(el, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
  };

  tabbar.addEventListener("click", (e) => {
    const tab = (e.target as HTMLElement).closest<HTMLElement>(".tab");
    const id = tab?.dataset.id;
    if (!id) return;
    if ((e.target as HTMLElement).closest('[data-action="close"]')) engine.closeTab(id);
    else engine.openPage(id);
  });
  tabbar.addEventListener("auxclick", (e) => {
    if (e.button !== 1) return;
    const id = (e.target as HTMLElement).closest<HTMLElement>(".tab")?.dataset.id;
    if (!id) return;
    e.preventDefault();
    engine.closeTab(id);
  });
  tabbar.addEventListener("mousedown", (e) => {
    if (e.button === 1) e.preventDefault();
  });
  tabbar.addEventListener("keydown", (e) => {
    const tab = (e.target as HTMLElement).closest<HTMLElement>(".tab");
    if (!tab) return;
    const all = [...tabbar.querySelectorAll<HTMLElement>(".tab")];
    const i = all.indexOf(tab);
    let next: HTMLElement | undefined;
    if (e.key === "ArrowRight") next = all[i + 1];
    else if (e.key === "ArrowLeft") next = all[i - 1];
    else if (e.key === "Delete" || e.key === "Backspace") {
      engine.closeTab(tab.dataset.id ?? "");
      e.preventDefault();
      return;
    } else return;
    e.preventDefault();
    if (next) {
      engine.openPage(next.dataset.id ?? "");
      next.focus();
    }
  });

  let shownVersion = "";
  const paintBanner = () => {
    const v = state.get("newVersion");
    setAttr(banner, "data-open", v && v !== shownVersion ? "true" : null);
    if (!v || banner.dataset.version === v) return;
    banner.dataset.version = v;
    const reload = h("button", { class: "btn btn--primary", attrs: { type: "button" }, text: t("Reload") });
    const dismiss = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Dismiss"), "data-tip": t("Dismiss") } }, iconNode("close"));
    reload.addEventListener("click", () => {
      pages.flushAll();
      setTimeout(() => location.reload(), 150);
    });
    dismiss.addEventListener("click", () => {
      shownVersion = v;
      paintBanner();
    });
    banner.replaceChildren(h("span", { class: "banner__icon" }, iconNode("info")), h("span", { class: "banner__text", text: t("form.md {version} is ready. Reload to use it.", { version: v }) }), reload, dismiss);
  };

  const paint = () => {
    paintCrumbs();
    paintActions();
    paintTabs();
  };
  state.watch("route", paint);
  state.watch("tabs", paintTabs);
  state.watch("detail", () => {
    paintCrumbs();
    paintActions();
    paintTabs();
  });
  state.watch("workspaceId", paintCrumbs);
  state.watch("me", () => {
    paintCrumbs();
    paintActions();
  });
  state.watch("newVersion", paintBanner);
  pages.subscribe((c) => {
    if (c.kind === "favorites") return paintActions();
    if (c.kind !== "upsert" && c.kind !== "remove" && c.kind !== "loaded") return;
    const r = state.get("route");
    const touched = c.kind === "upsert" ? [c.page.id] : c.kind === "remove" ? c.pages.map((p) => p.id) : [];
    const tabs = state.get("tabs");
    const relevant = c.kind === "loaded" || touched.some((id) => tabs.includes(id) || (r.name === "page" && (id === r.id || pages.ancestors(r.id).some((a) => a.id === id))));
    if (!relevant) return;
    paintCrumbs();
    paintActions();
    paintTabs();
  });
  paint();
  paintBanner();

  const edited = ref("act-edited");
  const paintEdited = () => {
    const at = currentPage()?.updated_at ?? 0;
    setText(edited, at ? t("Edited {time}", { time: formatRelative(at) }) : "");
    setAttr(edited, "title", at ? formatDateTime(at) : null);
  };
  state.watch("route", paintEdited);
  state.watch("detail", paintEdited);
  pages.subscribe((c) => {
    const r = state.get("route");
    if (c.kind === "loaded" || (c.kind === "upsert" && r.name === "page" && c.page.id === r.id)) paintEdited();
  });
  setInterval(() => {
    if (state.get("route").name === "page" && document.visibilityState === "visible") paintEdited();
  }, 60_000);
  paintEdited();
}
