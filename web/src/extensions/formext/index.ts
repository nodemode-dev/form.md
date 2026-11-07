import type { Engine } from "../../engine";
import { request } from "../../engine/api";
import { t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import { simpleDialog } from "../../ui/dialog";
import { byId, h, ref, setAttr, setText } from "../../ui/dom";
import { actions, button } from "../../ui/form";
import { iconNode } from "../../ui/icons";
import { type MenuEntry, menuButton, openMenu } from "../../ui/menu";
import { type Tone, toast } from "../../ui/toast";
import { editorHost } from "../blocks";
import type { Extension } from "../registry";
import { applyOps, readDocument, registerDocCommands, selectionOf } from "./docops";
import { ERR, type ExtensionRecord, ExtensionHost, type Frame, HostError, type HostHandlers, type PanelDef, type Permission, PERMISSION_WORDS, type Theme } from "./host-bridge";

const PRODUCT = "formmd";
const MANAGE_URL = `formai://extensions?product=${PRODUCT}`;
const POLL_MS = 2000;
const OPEN_CAP = 200 * 1024 * 1024;
const READ_PARTS = ["all", "blocks", "markdown", "selection", "outline", "page"];

interface Listing {
  appVersion: string;
  root: string;
  stamp: [number, number];
  extensions: (Omit<ExtensionRecord, "path" | "reason"> & { path: string | null; reason: string })[];
}

type RowState = "ready" | "needs-permission" | "refused";

interface Tab {
  key: string;
  ext: ExtensionRecord;
  panel: PanelDef;
  tab: HTMLElement;
  label: HTMLElement;
  view: HTMLElement;
  frame: Frame;
}

let teardown: Unsubscribe[] = [];

function sourceLabel(source: string): string {
  return source === "development" ? t("Development") : source === "sideload" ? t("File") : t("Catalogue");
}

function stateLabel(s: RowState): string {
  return s === "ready" ? t("Ready") : s === "needs-permission" ? t("Needs Permission") : t("Refused");
}

function b64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let out = "";
  for (let i = 0; i < bytes.length; i += 0x8000) out += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(out);
}

function theme(): Theme {
  const root = document.documentElement;
  const s = getComputedStyle(root);
  const v = (n: string) => s.getPropertyValue(n).trim();
  return {
    mode: root.dataset.theme === "light" ? "light" : "dark",
    tokens: {
      "--bg": v("--ground"),
      "--panel": v("--sheet"),
      "--text": v("--text"),
      "--muted": v("--text-muted"),
      "--accent": v("--accent"),
      "--border": v("--border"),
      "--radius": v("--radius"),
      "--surface": v("--surface"),
      "--on-accent": v("--on-accent"),
      "--danger": v("--danger"),
      "--font": v("--font-sans"),
    },
  };
}

function consent(ext: ExtensionRecord): Promise<boolean> {
  return new Promise((resolve) => {
    let ok = false;
    simpleDialog(
      t("Allow {name}?", { name: ext.name }),
      (body, close) => {
        const list = h("ul", { class: "xconsent__list" });
        for (const p of ext.permissions) list.append(h("li", {}, iconNode("check"), h("span", { text: t(PERMISSION_WORDS[p as Permission] ?? p) })));
        const later = button(t("Not Now"), "ghost");
        const allow = button(t("Allow"), "primary");
        later.addEventListener("click", close);
        allow.addEventListener("click", () => {
          ok = true;
          close();
        });
        body.append(
          h("p", { class: "section__lead", text: t("{name} {version} asks to:", { name: ext.name, version: ext.version }) }),
          list,
          h("p", { class: "xconsent__note", text: t("It runs in a sandbox and can do only this. You are asked again when a new version asks for more.") }),
          actions(later, allow),
        );
        return allow;
      },
      () => resolve(ok),
    );
  });
}

function openFiles(accept: string[], multiple: boolean): Promise<{ name: string; type: string; size: number; data: string }[]> {
  return new Promise((resolve, reject) => {
    const input = h("input", { attrs: { type: "file" } });
    if (accept.length) input.accept = accept.join(",");
    input.multiple = multiple;
    input.addEventListener("cancel", () => reject(new HostError(ERR.CANCELLED, t("No file was chosen"))));
    input.addEventListener("change", () => {
      const files = Array.from(input.files ?? []);
      if (!files.length) return reject(new HostError(ERR.CANCELLED, t("No file was chosen")));
      if (files.reduce((n, f) => n + f.size, 0) > OPEN_CAP) return reject(new HostError(ERR.REFUSED, t("The files are larger than 200 MB together")));
      void Promise.all(files.map(async (f) => ({ name: f.name, type: f.type, size: f.size, data: b64(await f.arrayBuffer()) }))).then(resolve, reject);
    });
    input.click();
  });
}

function confirmSave(ext: ExtensionRecord, name: string): Promise<boolean> {
  return new Promise((resolve) => {
    let ok = false;
    simpleDialog(
      t("Save {name}?", { name }),
      (body, close) => {
        const later = button(t("Cancel"), "ghost");
        const save = button(t("Save"), "primary");
        later.addEventListener("click", close);
        save.addEventListener("click", () => {
          ok = true;
          close();
        });
        body.append(h("p", { class: "section__lead", text: t("{ext} saves {name} in your downloads folder.", { ext: ext.name, name }) }), actions(later, save));
        return save;
      },
      () => resolve(ok),
    );
  });
}

async function saveFile(ext: ExtensionRecord, name: string, type: string, data: string): Promise<{ saved: boolean; name: string }> {
  const safe = name.replace(/[\\/:*?"<>|]/g, "_").slice(0, 200) || "download";
  if (!(await confirmSave(ext, safe))) throw new HostError(ERR.CANCELLED, t("The file was not saved"));
  const raw = atob(data);
  const bytes = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
  const url = URL.createObjectURL(new Blob([bytes], { type }));
  const a = h("a", { attrs: { href: url, download: safe } });
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 30_000);
  return { saved: true, name: safe };
}

function manage(): void {
  const a = h("a", { attrs: { href: MANAGE_URL } });
  a.click();
}

function setup(engine: Engine): void {
  const { commands, state } = engine;
  let list: ExtensionRecord[] = [];
  let root = "";
  let stamp = "";
  let loaded = false;
  let loading: Promise<void> | null = null;
  const extCommands = new Map<string, { sig: string; offs: Unsubscribe[] }>();
  const tabs = new Map<string, Tab>();
  let activeTab = "list";

  const handlers: HostHandlers = {
    product: PRODUCT,
    app: "form.md",
    appVersion: "",
    theme,
    notify: (ext, message, level) => {
      const tone: Tone = level === "error" ? "danger" : level === "warning" ? "warning" : level === "success" ? "success" : "info";
      toast({ tone, message: `${ext.name}: ${message}` });
    },
    setTitle: (ext, frame, title) => {
      for (const tab of tabs.values()) if (tab.frame.frame === frame) setText(tab.label, title ? `${ext.name}: ${title}` : `${ext.name}: ${tab.panel.title}`);
    },
    readDocument: (_ext, params) => {
      const what = params.what === undefined ? "all" : params.what;
      if (what === "file") throw new HostError(ERR.REFUSED, t("form.md pages are not files; read the page as Markdown with what: markdown"));
      if (typeof what !== "string" || !READ_PARTS.includes(what)) throw new HostError(ERR.INVALID, t("what must be one of {parts}", { parts: READ_PARTS.join(", ") }));
      return readDocument(engine, what);
    },
    applyDocument: (_ext, label, ops) => applyOps(label, ops),
    commands: () =>
      commands
        .catalogue()
        .filter((c) => !c.id.startsWith("extensions.") && !c.id.startsWith("ext."))
        .map((c) => ({ id: c.id, title: c.label, group: c.group, mutates: c.mutates, ...(c.params ? { args: c.params } : {}), ...(c.description ? { description: c.description } : {}) })),
    run: (_ext, command, args) => {
      const c = commands.catalogue().find((x) => x.id === command);
      if (!c) throw new HostError(ERR.REFUSED, t("form.md has no command {id}", { id: command }));
      if (c.risky || command.startsWith("extensions.") || command.startsWith("ext.")) throw new HostError(ERR.REFUSED, t("{id} cannot be run by an extension", { id: command }));
      return commands.invoke(command, args && typeof args === "object" ? (args as Record<string, unknown>) : {}).catch((e: unknown) => {
        throw new HostError(ERR.REFUSED, e instanceof Error ? e.message : String(e));
      });
    },
    openFiles: (_ext, accept, multiple) => openFiles(accept, multiple),
    saveFile: (ext, name, type, data) => saveFile(ext, name, type, data),
    consent,
  };
  const host = new ExtensionHost(handlers);

  const hidden = h("div", { class: "xhost", attrs: { "aria-hidden": "true" } });
  const listView = h("div", { class: "xpane__list", attrs: { role: "list" } });
  const empty = h(
    "div",
    { class: "xpane__empty" },
    h("span", { class: "xpane__empty-icon" }, iconNode("puzzle")),
    h("p", { class: "xpane__lead", text: t("No extensions yet. Find them in FormPortal or make one with form.ext.") }),
  );
  const manageBtn = button(t("Manage in FormPortal"), "default", "external");
  manageBtn.addEventListener("click", manage);
  empty.append(manageBtn);
  const where = h("p", { class: "xpane__where" });
  const listSection = h("section", { class: "xpane__view", attrs: { role: "tabpanel", "data-tab": "list" } }, empty, listView, where);
  const tabBar = h("div", { class: "xpane__tabs", attrs: { role: "tablist", "aria-label": t("Extensions") } });
  const listTab = h("button", { class: "xpane__tab", attrs: { type: "button", role: "tab", "aria-selected": "true", "data-tab": "list" } }, h("span", { class: "xpane__tab-label", text: t("Extensions") }));
  tabBar.append(listTab);
  const reloadBtn = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Reload Extensions"), "data-tip": t("Reload Extensions") } }, iconNode("refresh"));
  const closeBtn = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Close"), "data-tip": t("Close") } }, iconNode("close"));
  const body = h("div", { class: "xpane__body" }, listSection);
  const pane = h(
    "aside",
    { class: "apane xpane", attrs: { "aria-label": t("Extensions"), tabindex: "-1" } },
    h("header", { class: "apane__head xpane__head" }, tabBar, h("span", { class: "apane__actions" }, reloadBtn, closeBtn)),
    body,
  );
  pane.hidden = true;
  empty.hidden = true;
  byId("overlay-root").append(pane, hidden);

  let open = false;
  const setOpen = (next: boolean) => {
    if (open === next) return;
    open = next;
    pane.hidden = !open;
    if (open) requestAnimationFrame(() => pane.classList.add("is-visible"));
    else pane.classList.remove("is-visible");
  };
  closeBtn.addEventListener("click", () => setOpen(false));
  reloadBtn.addEventListener("click", () => void reload(true));

  const select = (key: string) => {
    activeTab = tabs.has(key) ? key : "list";
    for (const el of tabBar.children) setAttr(el, "aria-selected", (el as HTMLElement).dataset.tab === activeTab ? "true" : "false");
    for (const el of body.children) (el as HTMLElement).hidden = (el as HTMLElement).dataset.tab !== activeTab;
  };
  listTab.addEventListener("click", () => select("list"));

  const rowState = (ext: ExtensionRecord): RowState => (ext.state === "refused" ? "refused" : host.isGranted(ext) ? "ready" : "needs-permission");

  const closeTab = (key: string) => {
    const tab = tabs.get(key);
    if (!tab) return;
    tab.frame.dispose();
    tab.tab.remove();
    tab.view.remove();
    tabs.delete(key);
    if (activeTab === key) select("list");
  };

  const openDialog = (ext: ExtensionRecord, panel: PanelDef, activation: Parameters<Frame["activate"]>[0]) => {
    let frame: Frame | null = null;
    simpleDialog(
      `${ext.name}: ${panel.title}`,
      (holder) => {
        const box = h("div", { class: "xdialog__frame" });
        holder.append(box);
        void host
          .open(ext, box, { onResize: (px) => (box.style.blockSize = `${Math.max(120, px)}px`) })
          .then((f) => {
            frame = f;
            return f?.activate(activation);
          })
          .catch(engine.reportError);
      },
      () => frame?.dispose(),
      "md",
    );
  };

  const openPanel = async (ext: ExtensionRecord, panelId: string, activation?: Parameters<Frame["activate"]>[0]) => {
    const panel = ext.panels.find((p) => p.id === panelId);
    if (!panel || ext.state !== "ready") return;
    if (!(await host.ensureConsent(ext))) {
      paintRows();
      toast({ tone: "warning", message: t("{name} was not allowed to run", { name: ext.name }) });
      return;
    }
    paintRows();
    const act = activation ?? { reason: "panel" as const, panel: panel.id };
    if (panel.location === "dialog") return openDialog(ext, panel, act);
    const key = `${ext.id}/${panel.id}`;
    const existing = tabs.get(key);
    setOpen(true);
    if (existing) {
      select(key);
      if (activation) await existing.frame.activate(act);
      return;
    }
    const view = h("section", { class: "xpane__view xpane__view--frame", attrs: { role: "tabpanel", "data-tab": key } });
    const frame = await host.open(ext, view);
    if (!frame) return;
    if (tabs.has(key)) {
      frame.dispose();
      return select(key);
    }
    const label = h("span", { class: "xpane__tab-label", text: `${ext.name}: ${panel.title}` });
    const x = h("span", { class: "xpane__tab-close", attrs: { role: "button", "aria-label": t("Close {name}", { name: panel.title }) } }, iconNode("close"));
    const tab = h("button", { class: "xpane__tab", attrs: { type: "button", role: "tab", "aria-selected": "false", "data-tab": key } }, label, x);
    view.hidden = true;
    tabBar.append(tab);
    body.append(view);
    tabs.set(key, { key, ext, panel, tab, label, view, frame });
    tab.addEventListener("click", (e) => {
      if ((e.target as Element).closest(".xpane__tab-close")) closeTab(key);
      else select(key);
    });
    select(key);
    await frame.activate(act);
  };

  const runCommand = async (id: string, commandId: string, args: unknown = {}): Promise<unknown> => {
    const ext = list.find((e) => e.id === id);
    if (!ext) throw new Error(t("no extension {id} is installed", { id }));
    if (ext.state === "refused") throw new Error(t("{name} is refused: {reason}", { name: ext.name, reason: ext.reason ?? "" }));
    const command = ext.commands.find((c) => c.id === commandId);
    if (!command) throw new Error(t("{name} has no command {id}", { name: ext.name, id: commandId }));
    if (command.panel) {
      await openPanel(ext, command.panel, { reason: "command", command: command.id, panel: command.panel, args });
      if (!host.isGranted(ext)) throw new Error(t("{name} was not allowed to run", { name: ext.name }));
      return { opened: command.panel };
    }
    try {
      const result = await host.runCommand(ext, command.id, args, hidden);
      return { done: true, result: result ?? null };
    } finally {
      paintRows();
    }
  };

  const runFromUI = (id: string, cmd: string) => void runCommand(id, cmd).catch((e: unknown) => toast({ tone: "danger", message: e instanceof Error ? e.message : String(e) }));

  const rowSig = (ext: ExtensionRecord) => JSON.stringify([ext, rowState(ext)]);
  const rows = new Map<string, { el: HTMLElement; sig: string }>();
  const buildRow = (ext: ExtensionRecord): HTMLElement => {
    const s = rowState(ext);
    const head = h(
      "div",
      { class: "xrow__head" },
      h("span", { class: "xrow__name", text: ext.name }),
      h("span", { class: "xrow__meta", text: `${ext.version} · ${sourceLabel(ext.source)}` }),
      h("span", { class: "xrow__state", attrs: { "data-state": s }, text: stateLabel(s) }),
    );
    const el = h("div", { class: "xrow", attrs: { role: "listitem", "data-id": ext.id } }, head);
    if (ext.description) el.append(h("p", { class: "xrow__desc", text: ext.description }));
    if (ext.permissions.length) {
      const chips = h("div", { class: "xrow__chips" });
      for (const p of ext.permissions) chips.append(h("span", { class: "xrow__chip", attrs: { title: t(PERMISSION_WORDS[p] ?? p) }, text: p }));
      el.append(chips);
    }
    if (ext.state === "refused") el.append(h("p", { class: "xrow__reason", text: t("Refused: {reason}", { reason: ext.reason ?? "" }) }));
    if (ext.source === "development" && ext.path) el.append(h("p", { class: "xrow__path", attrs: { title: ext.path }, text: ext.path }));
    if (ext.state === "ready" && (ext.panels.length || s === "needs-permission")) {
      const row = h("div", { class: "xrow__actions" });
      for (const p of ext.panels) {
        const btn = button(ext.panels.length > 1 ? t("Open {name}", { name: p.title }) : t("Open"), "default");
        btn.addEventListener("click", () => void openPanel(ext, p.id).catch(engine.reportError));
        row.append(btn);
      }
      if (s === "needs-permission") {
        const allow = button(t("Allow"), "ghost");
        allow.addEventListener("click", () => void host.ensureConsent(ext).then(paintRows));
        row.append(allow);
      }
      el.append(row);
    }
    return el;
  };

  function paintRows(): void {
    let cursor: Element | null = listView.firstElementChild;
    const keep = new Set<string>();
    for (const ext of list) {
      keep.add(ext.id);
      const sig = rowSig(ext);
      let r = rows.get(ext.id);
      if (!r || r.sig !== sig) {
        const el = buildRow(ext);
        if (r) {
          if (cursor === r.el) cursor = el;
          r.el.replaceWith(el);
        }
        r = { el, sig };
        rows.set(ext.id, r);
      }
      if (r.el !== cursor) listView.insertBefore(r.el, cursor);
      else cursor = cursor.nextElementSibling;
    }
    for (const [id, r] of rows) {
      if (!keep.has(id)) {
        r.el.remove();
        rows.delete(id);
      }
    }
    empty.hidden = !loaded || list.length > 0;
    listView.hidden = list.length === 0;
    setText(where, root ? t("Read from {path}", { path: `${root}/extensions/${PRODUCT}` }) : "");
  }

  const syncCommands = () => {
    const want = new Map<string, ExtensionRecord>();
    for (const ext of list) if (ext.state === "ready") want.set(ext.id, ext);
    for (const [id, entry] of extCommands) {
      const ext = want.get(id);
      if (!ext || JSON.stringify([ext.name, ext.version, ext.commands]) !== entry.sig) {
        for (const off of entry.offs) off();
        extCommands.delete(id);
      }
    }
    for (const [id, ext] of want) {
      if (extCommands.has(id)) continue;
      const offs = ext.commands.map((c) =>
        commands.register({
          id: `ext.${ext.id}.${c.id}`,
          label: () => `${ext.name}: ${c.title}`,
          group: () => t("Extensions"),
          description: `Runs the "${c.title}" command of the ${ext.name} extension (${ext.id} ${ext.version}).`,
          params: { args: { type: "object", description: "Arguments passed to the extension" } },
          mutates: true,
          run: (params) => runCommand(ext.id, c.id, params?.args ?? {}),
        }),
      );
      extCommands.set(id, { sig: JSON.stringify([ext.name, ext.version, ext.commands]), offs });
    }
  };

  const apply = (data: Listing) => {
    const next: ExtensionRecord[] = data.extensions.map((e) => {
      const { path, reason, ...rest } = e;
      return { ...rest, ...(path ? { path } : {}), ...(reason ? { reason } : {}) };
    });
    const before = new Map(list.map((e) => [e.id, e]));
    for (const ext of next) {
      const old = before.get(ext.id);
      if (old && (old.version !== ext.version || old.state !== ext.state || old.entryUi !== ext.entryUi || JSON.stringify(old.panels) !== JSON.stringify(ext.panels))) {
        for (const key of [...tabs.keys()]) if (key.startsWith(`${ext.id}/`)) closeTab(key);
        host.disposeExtension(ext.id);
      }
      before.delete(ext.id);
    }
    for (const gone of before.values()) {
      for (const key of [...tabs.keys()]) if (key.startsWith(`${gone.id}/`)) closeTab(key);
      host.disposeExtension(gone.id);
    }
    list = next;
    root = data.root;
    handlers.appVersion = data.appVersion;
    stamp = JSON.stringify(data.stamp);
    loaded = true;
    paintRows();
    syncCommands();
  };

  const reload = (manual = false): Promise<void> => {
    loading ??= request<Listing>(fetch, "GET", "/api/extensions")
      .then(apply)
      .then(() => {
        if (manual) toast({ id: "ext-reload", tone: "success", message: t("Extensions reloaded") });
      })
      .catch((e: unknown) => {
        if (manual) engine.reportError(e);
      })
      .finally(() => (loading = null));
    return loading;
  };

  const poll = () => {
    if (!state.get("me") || !document.hasFocus() || loading) return;
    void request<{ stamp: [number, number] }>(fetch, "GET", "/api/extensions/stamp")
      .then((r) => {
        if (JSON.stringify(r.stamp) !== stamp) void reload();
      })
      .catch(() => undefined);
  };
  const timer = setInterval(poll, POLL_MS);
  const onFocus = () => poll();
  window.addEventListener("focus", onFocus);

  const menuEntries = (): MenuEntry[] => {
    const out: MenuEntry[] = [];
    for (const ext of list) {
      if (ext.state !== "ready") continue;
      for (const c of ext.commands) out.push({ label: `${ext.name}: ${c.title}`, icon: "puzzle", run: () => runFromUI(ext.id, c.id) });
    }
    if (out.length) out.push({ kind: "separator" });
    out.push(
      { label: t("Extensions Panel"), icon: "puzzle", run: () => openList() },
      { label: t("Reload Extensions"), icon: "refresh", run: () => void reload(true) },
      { label: t("Manage in FormPortal"), icon: "external", run: manage },
    );
    return out;
  };
  const sbBtn = ref<HTMLButtonElement>("sb-extensions");
  menuButton(sbBtn, menuEntries, { side: "top", align: "start", label: t("Extensions") });

  const openList = () => {
    setOpen(true);
    select("list");
    void reload();
  };

  let themeTimer: ReturnType<typeof setTimeout> | undefined;
  const themeWatch = new MutationObserver(() => {
    clearTimeout(themeTimer);
    themeTimer = setTimeout(() => host.setTheme(theme()), 60);
  });
  themeWatch.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme", "data-dark-theme", "data-light-theme", "data-contrast", "data-accent", "style"] });

  let hooked: unknown = null;
  let docTimer: ReturnType<typeof setTimeout> | undefined;
  let selTimer: ReturnType<typeof setTimeout> | undefined;
  const hook = () => {
    const ed = editorHost();
    if (!ed || hooked === ed) return;
    hooked = ed;
    const blocks = () => ed.view()?.state.doc.childCount ?? 0;
    teardown.push(
      ed.onUpdate((view, prev) => {
        const page = ed.current();
        if (!page) return;
        if (!view.state.doc.eq(prev.doc)) {
          clearTimeout(docTimer);
          docTimer = setTimeout(() => host.broadcast("document.changed", { page, reason: "edit", blocks: blocks() }), 250);
        }
        if (!view.state.selection.eq(prev.selection)) {
          clearTimeout(selTimer);
          selTimer = setTimeout(() => {
            const v = ed.view();
            if (v && ed.current() === page) host.broadcast("selection.changed", { page, ...selectionOf(v) });
          }, 150);
        }
      }),
      ed.onShown((page) => host.broadcast("document.changed", { page, reason: "opened", blocks: blocks() })),
    );
  };
  hook();

  teardown.push(
    ...registerDocCommands(engine),
    commands.register({ id: "extensions.panel", label: () => t("Extensions Panel"), group: () => t("Extensions"), description: "Opens the Extensions panel: every installed Form extension with its version, source, permissions and state.", mutates: false, run: openList }),
    commands.register({ id: "extensions.reload", label: () => t("Reload Extensions"), group: () => t("Extensions"), description: "Reads the extensions FormPortal and form.ext have set up again, and updates only what changed.", mutates: false, run: () => reload(true) }),
    commands.register({ id: "extensions.manage", label: () => t("Manage in FormPortal"), group: () => t("Extensions"), description: "Opens FormPortal at form.md's extensions.", mutates: false, run: manage }),
    commands.register({
      id: "extensions.menu",
      label: () => t("Extension Commands"),
      group: () => t("Extensions"),
      description: "Opens the Extensions menu in the sidebar, which lists every command extensions contribute.",
      mutates: false,
      run: () => {
        if (sbBtn.offsetParent) openMenu(sbBtn, menuEntries(), { side: "top", align: "start", label: t("Extensions") });
        else openList();
      },
    }),
    commands.register({
      id: "extensions.list",
      label: () => t("List extensions"),
      group: () => t("Extensions"),
      description: "Every installed Form extension with id, name, version, source, state (ready, needs-permission or refused) and its commands.",
      mutates: false,
      hidden: () => true,
      run: async () => {
        await reload();
        return {
          extensions: list.map((e) => ({ id: e.id, name: e.name, version: e.version, source: e.source, state: rowState(e), reason: e.reason ?? "", permissions: e.permissions, commands: e.commands, panels: e.panels })),
        };
      },
    }),
    commands.register({
      id: "extensions.run",
      label: () => t("Run an extension command"),
      group: () => t("Extensions"),
      description: "Runs a command an extension contributes, asking the person to allow the extension first if this version is not allowed yet.",
      params: { id: { type: "string", required: true, description: "Extension id" }, command: { type: "string", required: true, description: "Command id" }, args: { type: "object", description: "Arguments for the command" } },
      mutates: true,
      hidden: () => true,
      run: (p) => runCommand(String(p?.id), String(p?.command), p?.args ?? {}),
    }),
    state.watch("me", (me) => {
      if (me) void reload();
    }),
    state.watch("route", hook),
    () => {
      clearInterval(timer);
      window.removeEventListener("focus", onFocus);
      themeWatch.disconnect();
      for (const entry of extCommands.values()) for (const off of entry.offs) off();
      extCommands.clear();
      for (const key of [...tabs.keys()]) closeTab(key);
      pane.remove();
      hidden.remove();
    },
  );
  if (state.get("me")) void reload();
}

const formext: Extension = {
  id: "formext",
  setup,
  dispose() {
    for (const fn of teardown) fn();
    teardown = [];
  },
};

export default formext;
