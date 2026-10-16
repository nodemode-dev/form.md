import type { Engine, Page } from "../engine";
import { t } from "../engine/i18n";
import type { Unsubscribe } from "../engine/state";
import type { PageStyle } from "../engine/types";
import { setAttr } from "./dom";
import { openPrompt } from "../extensions/blocks/popover";
import { type MenuEntry, openMenu } from "./menu";

type Key = keyof PageStyle;

const OPTIONS: { key: Key; title: string; values: [string, string][] }[] = [
  { key: "font", title: "Font", values: [["sans", "Sans"], ["serif", "Serif"], ["mono", "Mono"], ["rounded", "Rounded"], ["slab", "Slab"], ["humanist", "Humanist"]] },
  { key: "size", title: "Text size", values: [["small", "Small"], ["default", "Default"], ["large", "Large"]] },
  { key: "width", title: "Page width", values: [["default", "Default"], ["wide", "Wide"], ["full", "Full width"]] },
  { key: "spacing", title: "Line spacing", values: [["compact", "Compact"], ["default", "Default"], ["relaxed", "Relaxed"]] },
  { key: "align", title: "Text alignment", values: [["start", "Left"], ["justify", "Justified"], ["center", "Centred"]] },
];

export function effectiveStyle(engine: Engine, p: Page): PageStyle {
  return { ...(engine.workspaceById(p.workspace_id)?.page_style ?? {}), ...(p.style ?? {}) };
}

export function applyStyle(el: HTMLElement, s: PageStyle): void {
  for (const { key } of OPTIONS) {
    const v = s[key];
    setAttr(el, `data-style-${key}`, v && v !== "default" && !(key === "font" && v === "sans") && !(key === "align" && v === "start") ? v : null);
  }
}

function currentPage(engine: Engine): Page | null {
  const r = engine.state.get("route");
  if (r.name !== "page") return null;
  return engine.pages.get(r.id) ?? null;
}

function setStyle(engine: Engine, p: Page, patch: PageStyle): PageStyle {
  const next: PageStyle = { ...(p.style ?? {}), ...patch };
  for (const k of Object.keys(next) as Key[]) if (!next[k]) delete next[k];
  engine.pages.edit(p.id, { style: next }, false);
  return next;
}

async function makeDefault(engine: Engine, p: Page): Promise<void> {
  const style = effectiveStyle(engine, p);
  await engine.api.workspaces.update(p.workspace_id, { page_style: style });
  await engine.refreshMe();
}

export function syncFonts(engine: Engine, workspaceId: string): void {
  const fonts = engine.workspaceById(workspaceId)?.fonts ?? [];
  const href = fonts.length ? `/api/workspaces/${encodeURIComponent(workspaceId)}/fonts.css?v=${fonts.map((f) => f.file.slice(2, 8)).join("")}` : "";
  const links = [...document.querySelectorAll<HTMLLinkElement>("link[data-fonts]")];
  if (href && links.some((l) => l.getAttribute("href") === href || l.getAttribute("href") === href.split("?")[0])) return;
  if (!href) return;
  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = href;
  link.dataset.fonts = "";
  link.addEventListener("load", () => {
    for (const old of links) old.remove();
  }, { once: true });
  document.head.appendChild(link);
}

async function uploadFont(engine: Engine, p: Page): Promise<void> {
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".woff2,.woff,.ttf,.otf,font/woff2,font/woff,font/ttf,font/otf";
  const file = await new Promise<File | null>((resolve) => {
    input.addEventListener("change", () => resolve(input.files?.[0] ?? null), { once: true });
    input.click();
  });
  if (!file) return;
  const up = await engine.api.files.upload(file, "font", { workspace_id: p.workspace_id, name: file.name });
  const family = file.name.replace(/\.(woff2?|ttf|otf)$/i, "").replace(/[^\p{L}\p{N} ._-]+/gu, " ").trim().slice(0, 60) || "Font";
  const fonts = [...(engine.workspaceById(p.workspace_id)?.fonts ?? []), { file: `f:${up.file.id}`, family }];
  await engine.api.workspaces.update(p.workspace_id, { fonts });
  await engine.refreshMe();
  syncFonts(engine, p.workspace_id);
  setStyle(engine, engine.pages.get(p.id) ?? p, { font: `f:${up.file.id}` });
}

async function saveFonts(engine: Engine, workspaceId: string, fonts: { file: string; family: string }[]): Promise<void> {
  await engine.api.workspaces.update(workspaceId, { fonts });
  await engine.refreshMe();
  syncFonts(engine, workspaceId);
}

function renameFont(engine: Engine, anchor: HTMLElement, workspaceId: string, file: string, family: string): void {
  openPrompt({
    label: t("Rename this font..."),
    placeholder: t("Font name"),
    value: family,
    action: t("Rename"),
    rect: () => anchor.getBoundingClientRect(),
    onSubmit: (raw) => {
      const name = raw.replace(/[^\p{L}\p{N} ._-]+/gu, " ").trim().slice(0, 60);
      if (!name || name === family) return;
      const fonts = (engine.workspaceById(workspaceId)?.fonts ?? []).map((f) => (f.file === file ? { file: f.file, family: name } : f));
      void saveFonts(engine, workspaceId, fonts).catch(engine.reportError);
    },
  });
}

async function removeFont(engine: Engine, p: Page, file: string): Promise<void> {
  const fonts = (engine.workspaceById(p.workspace_id)?.fonts ?? []).filter((f) => f.file !== file);
  if (p.style?.font === file) setStyle(engine, p, { font: "sans" });
  const def = engine.workspaceById(p.workspace_id)?.page_style;
  if (def?.font === file) {
    const { font: _gone, ...rest } = def;
    await engine.api.workspaces.update(p.workspace_id, { page_style: rest });
  }
  await saveFonts(engine, p.workspace_id, fonts);
}

export function openStyleMenu(engine: Engine, anchor: HTMLElement, p: Page): void {
  const cur = effectiveStyle(engine, p);
  const entries: MenuEntry[] = [];
  const ws = engine.workspaceById(p.workspace_id);
  for (const o of OPTIONS) {
    entries.push({ kind: "heading", label: t(o.title) });
    for (const [v, label] of o.values) {
      const on = (cur[o.key] ?? (o.key === "font" ? "sans" : o.key === "align" ? "start" : "default")) === v;
      entries.push({ label: t(label), checked: on, run: () => setStyle(engine, p, { [o.key]: v } as PageStyle) });
    }
    if (o.key === "font") {
      for (const f of ws?.fonts ?? []) entries.push({ label: f.family, checked: cur.font === f.file, run: () => setStyle(engine, p, { font: f.file as `f:${string}` }) });
      if (ws?.role === "admin") entries.push({ label: t("Upload a font..."), icon: "upload", run: () => void uploadFont(engine, p).catch(engine.reportError) });
      const mine = ws?.role === "admin" ? ws.fonts?.find((f) => f.file === cur.font) : undefined;
      if (mine) {
        entries.push({ label: t("Rename this font..."), icon: "pencil", run: () => renameFont(engine, anchor, p.workspace_id, mine.file, mine.family) });
        entries.push({ label: t("Remove this font from the workspace"), icon: "trash", danger: true, run: () => void removeFont(engine, p, mine.file).catch(engine.reportError) });
      }
    }
  }
  entries.push({ kind: "separator" });
  entries.push({ label: t("Reset this page to the workspace style"), run: () => engine.pages.edit(p.id, { style: {} }, false) });
  if (ws && ws.role === "admin") entries.push({ label: t("Use this style for every page in the workspace"), run: () => void makeDefault(engine, p).catch(engine.reportError) });
  openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Page style") });
}

export function registerStyleCommands(engine: Engine): Unsubscribe[] {
  const group = () => t("Page style");
  const params = Object.fromEntries(OPTIONS.map((o) => [o.key, { type: "string" as const, enum: o.values.map(([v]) => v) }]));
  return [
    engine.commands.register({
      id: "page.style",
      label: () => t("Set the page style"),
      group,
      description: "Sets the open page's font, text size, page width, line spacing or text alignment; leave a field out to keep it.",
      when: "Use when the person wants a page to look different: a serif or mono font, wider page, larger text, justified text.",
      params,
      mutates: true,
      enabled: () => {
        const p = currentPage(engine);
        return !!p && engine.can(p.id, "content");
      },
      run: (args) => {
        const p = currentPage(engine);
        if (!p) throw new Error("no page is open");
        const patch: PageStyle = {};
        for (const o of OPTIONS) if (typeof args?.[o.key] === "string") (patch as Record<string, string>)[o.key] = args[o.key] as string;
        return setStyle(engine, p, patch);
      },
    }),
    engine.commands.register({
      id: "page.styleReset",
      label: () => t("Reset this page to the workspace style"),
      group,
      description: "Removes the open page's own style so it follows the workspace default again.",
      mutates: true,
      enabled: () => {
        const p = currentPage(engine);
        return !!p && engine.can(p.id, "content");
      },
      run: () => {
        const p = currentPage(engine);
        if (p) engine.pages.edit(p.id, { style: {} }, false);
      },
    }),
    engine.commands.register({
      id: "page.styleDefault",
      label: () => t("Use this style for every page in the workspace"),
      group,
      description: "Makes the open page's style the workspace default; pages with their own style keep it. Workspace admins only.",
      mutates: true,
      run: async () => {
        const p = currentPage(engine);
        if (!p) throw new Error("no page is open");
        await makeDefault(engine, p);
      },
    }),
  ];
}
