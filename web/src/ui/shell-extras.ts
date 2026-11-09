import type { Engine, Page } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { navSpan } from "./app";
import { avatar } from "./avatar";
import { DEMOS, mountDemos } from "./demos";
import { byId, h, ref, setAttr, setText } from "./dom";
import { anchorTo } from "./floating";
import type { IconName } from "./icons";
import { type MenuEntry, menuButton } from "./menu";
import { fallbackGlyph, pageIcon } from "./pageicon";
import { focusTitleSoon } from "./tree";

const WIDTH_KEY = "formmd.sidebarWidth";
const MIN_WIDTH = 220;
const MAX_WIDTH = 480;

interface Person {
  id: string;
  name: string;
  tint: string;
  avatar: string;
}

function chordText(engine: Engine, id: string): string {
  return engine.keymap.chordFor(id)?.join(" ") ?? "";
}

function currentPage(engine: Engine): Page | null {
  const r = engine.state.get("route");
  if (r.name !== "page") return null;
  const d = engine.state.get("detail");
  return engine.pages.get(r.id) ?? (d && d.id === r.id ? d : null);
}

function mountHistory(engine: Engine): void {
  const { commands, keymap, state } = engine;
  const back = ref<HTMLButtonElement>("nav-back");
  const forward = ref<HTMLButtonElement>("nav-forward");
  const paint = () => {
    const s = navSpan();
    if (back.disabled === s.back) back.disabled = !s.back;
    if (forward.disabled === s.forward) forward.disabled = !s.forward;
  };
  const general = () => t("General");
  commands.register({
    id: "nav.back",
    label: () => t("Go back"),
    group: general,
    description: "Goes back to the previous view in this window's history. Does not change any data.",
    mutates: false,
    enabled: () => navSpan().back,
    run: () => history.back(),
  });
  commands.register({
    id: "nav.forward",
    label: () => t("Go forward"),
    group: general,
    description: "Goes forward again in this window's history after going back. Does not change any data.",
    mutates: false,
    enabled: () => navSpan().forward,
    run: () => history.forward(),
  });
  keymap.register({ id: "nav.back", command: "nav.back", chords: ["mod+["], scope: "global", whileTyping: true });
  keymap.register({ id: "nav.forward", command: "nav.forward", chords: ["mod+]"], scope: "global", whileTyping: true });
  state.watch("route", paint);
  window.addEventListener("popstate", paint);
  paint();
}

function mountResize(): void {
  const root = document.documentElement;
  const sidebar = byId("sidebar");
  const handle = ref("sb-resize");
  const clamp = (w: number) => Math.round(Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, w)));
  const apply = (w: number | null) => {
    if (w === null) root.style.removeProperty("--sidebar-w");
    else root.style.setProperty("--sidebar-w", `${w}px`);
    setAttr(handle, "aria-valuenow", String(Math.round(w ?? sidebar.getBoundingClientRect().width)));
  };
  const save = (w: number | null) => {
    try {
      if (w === null) localStorage.removeItem(WIDTH_KEY);
      else localStorage.setItem(WIDTH_KEY, String(w));
    } catch {
      return;
    }
  };
  handle.addEventListener("pointerdown", (e) => {
    if (e.button !== 0) return;
    e.preventDefault();
    const left = sidebar.getBoundingClientRect().left;
    let width = clamp(sidebar.getBoundingClientRect().width);
    let frame = 0;
    handle.setPointerCapture(e.pointerId);
    setAttr(root, "data-resizing", "");
    const move = (ev: PointerEvent) => {
      width = clamp(ev.clientX - left);
      if (!frame) frame = requestAnimationFrame(() => ((frame = 0), apply(width)));
    };
    const end = () => {
      cancelAnimationFrame(frame);
      handle.removeEventListener("pointermove", move);
      handle.removeEventListener("pointerup", end);
      handle.removeEventListener("pointercancel", end);
      setAttr(root, "data-resizing", null);
      apply(width);
      save(width);
    };
    handle.addEventListener("pointermove", move);
    handle.addEventListener("pointerup", end);
    handle.addEventListener("pointercancel", end);
  });
  handle.addEventListener("dblclick", () => {
    apply(null);
    save(null);
  });
  handle.addEventListener("keydown", (e) => {
    const step = e.key === "ArrowLeft" ? -16 : e.key === "ArrowRight" ? 16 : 0;
    if (!step) return;
    e.preventDefault();
    const set = Number.parseFloat(root.style.getPropertyValue("--sidebar-w"));
    const w = clamp((Number.isFinite(set) ? set : sidebar.getBoundingClientRect().width) + step);
    apply(w);
    save(w);
  });
  setAttr(handle, "aria-valuenow", String(Math.round(sidebar.getBoundingClientRect().width)));
}

function mountSidebarExtras(engine: Engine): void {
  const { state, commands } = engine;
  const root = document.documentElement;
  const newBtn = ref<HTMLButtonElement>("sb-new");
  setText(ref("sb-search-keys"), chordText(engine, "search.open"));

  const paintAccess = () => {
    const me = state.get("me");
    if (me) setAttr(root, "data-desktop", (me.instance as { desktop?: boolean }).desktop ? "" : null);
    const w = engine.canWrite();
    if (newBtn.disabled === w) newBtn.disabled = !w;
  };
  state.watch("me", paintAccess);
  state.watch("workspaceId", paintAccess);
  paintAccess();

  const newDatabase = async () => {
    const ws = state.get("workspaceId");
    if (!ws) return;
    try {
      const r = await engine.api.pages.createCollection({ workspace_id: ws });
      engine.pages.upsert(r.page);
      engine.openPage(r.page.id);
      focusTitleSoon();
    } catch (e) {
      engine.reportError(e);
    }
  };
  menuButton(
    newBtn,
    () => [
      { label: t("New page"), icon: "page", hint: chordText(engine, "page.new"), run: () => commands.run("page.new") },
      { label: t("New database"), icon: "database", run: () => void newDatabase() },
      { kind: "separator" },
      { label: t("From a template"), icon: "gallery", run: () => void import("./templates-dialog").then((m) => m.openTemplates(engine), engine.reportError) },
      { label: t("Import"), icon: "upload", run: () => void import("./transfer-dialog").then((m) => m.openImport(engine, { workspaceId: state.get("workspaceId") }), engine.reportError) },
    ],
    { side: "bottom", align: "end", label: t("New") },
  );

  const link = (name: string, go: () => void) =>
    ref(name).addEventListener("click", (e) => {
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
      e.preventDefault();
      go();
    });
  link("sb-home", () => engine.navigate({ name: "home" }));
  link("sb-library", () => engine.navigate({ name: "library", view: "recent" }));
  ref("sb-settings").addEventListener("click", () => commands.run(commands.get("prefs.open") ? "prefs.open" : "account.open"));
  mountResize();
}

function mountCrumbSiblings(engine: Engine): void {
  const { pages, state } = engine;
  const crumbs = ref("crumbs");
  const menu = h("div", { class: "menu crumbmenu", attrs: { role: "menu", "aria-label": t("Pages at this level") } });
  let anchor: HTMLElement | null = null;
  let release: (() => void) | null = null;
  let showTimer: ReturnType<typeof setTimeout> | undefined;
  let hideTimer: ReturnType<typeof setTimeout> | undefined;

  const hide = () => {
    clearTimeout(showTimer);
    clearTimeout(hideTimer);
    if (!anchor) return;
    anchor = null;
    release?.();
    release = null;
    menu.classList.remove("is-visible");
    menu.remove();
  };
  const siblings = (id: string): Page[] => {
    const p = pages.get(id);
    if (!p || !pages.isLoaded(p.workspace_id)) return [];
    if (p.detached) return pages.shared(p.workspace_id);
    return p.parent_id ? pages.children(p.workspace_id, p.parent_id) : pages.roots(p.workspace_id, p.teamspace_id ?? "");
  };
  const show = (el: HTMLElement) => {
    const id = el.dataset.target ?? "";
    const list = id ? siblings(id) : [];
    if (list.length < 2 || !el.isConnected) return;
    hide();
    anchor = el;
    menu.replaceChildren(
      ...list.slice(0, 40).map((p) => {
        const b = h("button", { class: "menu__item", attrs: { type: "button", role: "menuitem", tabindex: "-1", "data-id": p.id } }, pageIcon(p.icon, fallbackGlyph(p.kind)), h("span", { class: "menu__label", text: p.title || t("Untitled") }), h("span"));
        if (p.id === id) setAttr(b, "aria-current", "true");
        return b;
      }),
    );
    byId("overlay-root").append(menu);
    release = anchorTo(menu, el, { side: "bottom", align: "start", offset: 4 });
    requestAnimationFrame(() => {
      if (anchor === el) menu.classList.add("is-visible");
    });
  };
  const leave = () => {
    clearTimeout(showTimer);
    clearTimeout(hideTimer);
    hideTimer = setTimeout(hide, 220);
  };

  crumbs.addEventListener("pointerover", (e) => {
    if (e.pointerType === "touch") return;
    clearTimeout(hideTimer);
    const el = (e.target as Element).closest<HTMLElement>(".crumbs__item");
    if (!el || el === anchor) return;
    clearTimeout(showTimer);
    showTimer = setTimeout(() => show(el), anchor ? 120 : 400);
  });
  crumbs.addEventListener("pointerleave", leave);
  crumbs.addEventListener("pointerdown", hide);
  menu.addEventListener("pointerenter", () => clearTimeout(hideTimer));
  menu.addEventListener("pointerleave", leave);
  menu.addEventListener("click", (e) => {
    const id = (e.target as Element).closest<HTMLElement>("[data-id]")?.dataset.id;
    hide();
    if (id) engine.openPage(id, { newTab: engine.platform.mac ? e.metaKey : e.ctrlKey });
  });
  document.addEventListener(
    "keydown",
    (e) => {
      if (e.key === "Escape" && anchor) hide();
    },
    true,
  );
  state.watch("route", hide);
}

function mountPageMeta(engine: Engine): void {
  const { state, pages } = engine;
  const edited = ref("act-edited");
  const people = ref("act-presence");

  const paintEdited = () => {
    const at = currentPage(engine)?.updated_at ?? 0;
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

  const paintPeople = (list: Person[]) => {
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(people.children) as HTMLElement[]) if (c.dataset.key) existing.set(c.dataset.key, c);
    let cursor: Element | null = people.firstElementChild;
    for (const p of list) {
      let el = existing.get(p.id);
      if (el) existing.delete(p.id);
      else {
        el = avatar(p.name, p.tint, false, p.avatar);
        el.dataset.key = p.id;
      }
      setAttr(el, "title", p.name);
      if (el !== cursor) people.insertBefore(el, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const el of existing.values()) el.remove();
    setAttr(people, "aria-label", list.length ? t("Also here: {names}", { names: list.map((p) => p.name).join(", ") }) : null);
    setAttr(people, "role", list.length ? "group" : null);
  };
  let off: (() => void) | null = null;
  let retry: ReturnType<typeof setTimeout> | undefined;
  const attach = (tries = 0) => {
    clearTimeout(retry);
    off?.();
    off = null;
    const r = state.get("route");
    if (r.name !== "page") return paintPeople([]);
    const s = engine.docs.peek(r.id);
    if (!s?.awareness) {
      paintPeople([]);
      if (tries < 20) retry = setTimeout(() => attach(tries + 1), 500);
      return;
    }
    const me = state.get("me")?.user.id ?? "";
    const read = () => {
      const seen = new Map<string, Person>();
      s.awareness.getStates().forEach((st, client) => {
        const u = (st as { user?: Partial<Person> }).user;
        if (client === s.awareness.clientID || !u?.id || u.id === me || seen.has(u.id)) return;
        seen.set(u.id, { id: u.id, name: u.name ?? "", tint: u.tint ?? "violet", avatar: u.avatar ?? "" });
      });
      paintPeople([...seen.values()].slice(0, 4));
    };
    s.awareness.on("change", read);
    off = () => s.awareness.off("change", read);
    read();
  };
  state.watch("route", () => attach());
  attach();
}

function mountHelp(engine: Engine): void {
  const { commands } = engine;
  const item = (id: string, label: string, icon: IconName): MenuEntry[] => (commands.get(id) ? [{ label, icon, hint: chordText(engine, id), command: id, run: () => commands.run(id) }] : []);
  menuButton(
    ref("help-btn"),
    () => {
      const entries: MenuEntry[] = [...item("shell.showShortcuts", t("Keyboard shortcuts"), "keyboard"), ...item("palette.open", t("Find a command"), "search")];
      const demos = DEMOS.flatMap((d) => item(`demo.${d.id}`, d.title(), d.icon));
      if (demos.length) entries.push({ kind: "separator" }, { kind: "heading", label: t("Demos") }, ...demos);
      const version = engine.state.get("health").version;
      if (version) entries.push({ kind: "separator" }, { kind: "heading", label: t("form.md {version}", { version }) });
      return entries;
    },
    { side: "top", align: "end", label: t("Help") },
  );
}

export function mountShellExtras(engine: Engine): void {
  mountHistory(engine);
  mountSidebarExtras(engine);
  mountCrumbSiblings(engine);
  mountPageMeta(engine);
  mountDemos(engine);
  mountHelp(engine);
}
