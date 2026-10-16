import type { Engine, Page } from "../engine";
import { t } from "../engine/i18n";
import type { Unsubscribe } from "../engine/state";
import type { PageStyle } from "../engine/types";
import { setAttr } from "./dom";
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

export function openStyleMenu(engine: Engine, anchor: HTMLElement, p: Page): void {
  const cur = effectiveStyle(engine, p);
  const entries: MenuEntry[] = [];
  for (const o of OPTIONS) {
    entries.push({ kind: "heading", label: t(o.title) });
    for (const [v, label] of o.values) {
      const on = (cur[o.key] ?? (o.key === "font" ? "sans" : o.key === "align" ? "start" : "default")) === v;
      entries.push({ label: t(label), checked: on, run: () => setStyle(engine, p, { [o.key]: v } as PageStyle) });
    }
  }
  entries.push({ kind: "separator" });
  entries.push({ label: t("Reset this page to the workspace style"), run: () => engine.pages.edit(p.id, { style: {} }, false) });
  const ws = engine.workspaceById(p.workspace_id);
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
