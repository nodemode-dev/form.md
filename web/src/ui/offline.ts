import * as Y from "yjs";
import type { Engine } from "../engine";
import type { Collection, CollectionView } from "../engine/types";
import { plural, t } from "../engine/i18n";
import { offlineIds, setOfflineIds } from "./offline-list";
import { toast } from "./toast";

let started: Promise<ServiceWorkerRegistration | null> | null = null;

const ROWS_KEPT = 1000;
const RECENT_KEPT = 20;
const RECENT_OFF_KEY = "formmd.offlineRecentOff";

const idle = (fn: () => void, delay: number) => setTimeout(() => (window.requestIdleCallback ?? ((f: () => void) => f()))(fn), delay);

function post(reg: ServiceWorkerRegistration, message: { type: string }): void {
  (reg.active ?? reg.waiting ?? reg.installing)?.postMessage(message);
}

async function register(): Promise<ServiceWorkerRegistration | null> {
  if (!("serviceWorker" in navigator)) return null;
  try {
    const reg = (await navigator.serviceWorker.getRegistration("/")) ?? (await navigator.serviceWorker.register("/sw.js", { scope: "/" }));
    await navigator.serviceWorker.ready;
    return reg;
  } catch {
    return null;
  }
}

export function startOffline(engine: Engine): Promise<ServiceWorkerRegistration | null> {
  return (started ??= register().then((reg) => {
    if (!reg) return null;
    registerOfflineCommands(engine);
    const shell = () => {
      if (navigator.onLine && engine.state.get("me")) post(reg, { type: "offline-shell" });
    };
    shell();
    const bootRow = engine.boot.row?.row?.id;
    if (bootRow && navigator.onLine) post(reg, { type: "offline-api", urls: [`/api/rows/${encodeURIComponent(bootRow)}`] } as { type: string });
    let timer: ReturnType<typeof setTimeout> | undefined;
    engine.pages.subscribe(() => {
      clearTimeout(timer);
      timer = setTimeout(shell, 4000);
    });
    const saveData = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection?.saveData;
    if (!saveData) idle(() => post(reg, { type: "offline-precache" }), 20_000);
    idle(() => void refreshKept(engine), 30_000);
    return reg;
  }));
}

function fileRefs(el: Y.XmlFragment | Y.XmlElement, out: Set<string>): Set<string> {
  for (const c of el.toArray()) {
    if (!(c instanceof Y.XmlElement)) continue;
    for (const v of Object.values(c.getAttributes())) if (typeof v === "string" && v.startsWith("f:")) out.add(`/files/${v.slice(2)}`);
    fileRefs(c, out);
  }
  return out;
}

function postFiles(urls: Iterable<string>): void {
  const list = [...urls];
  if (list.length) void started?.then((reg) => reg && post(reg, { type: "offline-files", urls: list } as { type: string }));
}

function postApi(urls: string[]): void {
  if (urls.length) void started?.then((reg) => reg && post(reg, { type: "offline-api", urls } as { type: string }));
}

function pageFiles(engine: Engine, id: string, out: Set<string>): void {
  const p = engine.pages.get(id);
  for (const ref of [p?.cover ?? "", p?.icon ?? ""]) if (ref.startsWith("f:")) out.add(`/files/${ref.slice(2)}`);
}

function syncDoc(engine: Engine, id: string, ms = 20_000): Promise<boolean> {
  const s = engine.docs.open(id);
  return new Promise((resolve) => {
    let done = false;
    let off = () => {};
    const timer = setTimeout(() => finish(false), ms);
    function finish(ok: boolean): void {
      if (done) return;
      done = true;
      off();
      clearTimeout(timer);
      if (ok) {
        const files = fileRefs(s.fragment, new Set());
        pageFiles(engine, id, files);
        postFiles(files);
      }
      engine.docs.release(s);
      resolve(ok);
    }
    off = s.on((what) => {
      if (what === "synced") finish(true);
      else if (what === "denied") finish(false);
    });
    if (s.synced) finish(true);
  });
}

function keptPages(engine: Engine, roots: string[]): string[] {
  const seen = new Set<string>();
  for (const id of roots) for (const p of engine.pages.subtree(id)) if (p.kind === "doc" || p.kind === "collection") seen.add(p.id);
  return [...seen];
}

async function keepRows(engine: Engine, rows: string[]): Promise<void> {
  postApi(rows.flatMap((r) => [`/api/pages/${encodeURIComponent(r)}`, `/api/rows/${encodeURIComponent(r)}`]));
  let next = 0;
  const worker = async () => {
    while (next < rows.length && navigator.onLine) await syncDoc(engine, rows[next++] as string);
  };
  await Promise.all([worker(), worker(), worker(), worker()]);
}

async function warmView(engine: Engine, col: Collection, v: CollectionView): Promise<void> {
  if (!navigator.onLine) return;
  const api = engine.api.collections;
  if (v.type === "chart") {
    const { chartOf } = await import("../extensions/database/views/chart");
    await api.chart(col.id, { view_id: v.id, chart: chartOf(col, v.config.layout) });
  } else if (v.type === "board" || v.type === "gallery") await api.query(col.id, { view_id: v.id, offset: 0, limit: 100, group_limit: 50 });
  else if (v.type === "calendar" || v.type === "timeline" || v.type === "map") await api.query(col.id, { view_id: v.id, offset: 0, limit: ROWS_KEPT, no_groups: true });
}

function keep(engine: Engine, id: string): Promise<boolean> {
  postApi([`/api/pages/${encodeURIComponent(id)}`]);
  if (engine.pages.get(id)?.kind === "collection") {
    const base = `/api/collections/${encodeURIComponent(id)}`;
    return engine.api.collections.get(id).then(
      async (r) => {
        const views = r.collection.views.filter((v) => v.type === "table" || v.type === "list");
        postApi([`${base}?view=&rows=${ROWS_KEPT}`, ...views.map((v) => `${base}?view=${encodeURIComponent(v.id)}&rows=${ROWS_KEPT}`)]);
        for (const v of r.collection.views) await warmView(engine, r.collection, v).catch(() => undefined);
        const first = r.collection.views[0]?.id;
        const q = await engine.api.collections.query(id, { ...(first ? { view_id: first } : {}), limit: ROWS_KEPT, no_groups: true });
        await keepRows(engine, q.order);
        return true;
      },
      () => false,
    );
  }
  return syncDoc(engine, id);
}

function recentOff(): boolean {
  try {
    return localStorage.getItem(RECENT_OFF_KEY) === "1";
  } catch {
    return false;
  }
}

async function recentPages(engine: Engine): Promise<string[]> {
  if (recentOff()) return [];
  try {
    const r = await engine.api.pages.recent(RECENT_KEPT);
    return r.pages.filter((p) => p.kind === "doc" || p.kind === "collection").map((p) => p.id);
  } catch {
    return [];
  }
}

async function refreshKept(engine: Engine): Promise<void> {
  if (!navigator.onLine) return;
  const ids = new Set([...keptPages(engine, offlineIds(engine)), ...(await recentPages(engine))]);
  for (const id of ids) {
    if (!navigator.onLine) return;
    await keep(engine, id);
  }
}

function registerOfflineCommands(engine: Engine): void {
  engine.commands.register({
    id: "offline.recent",
    label: () => t("Keep recent pages offline"),
    group: () => t("Offline"),
    mutates: false,
    description: "Turns on or off keeping the 20 most recently opened pages and databases for offline use in this browser, in addition to the pages marked for offline use. On by default.",
    params: { on: { type: "boolean" } },
    run: (p) => {
      const on = typeof p?.on === "boolean" ? p.on : recentOff();
      try {
        if (on) localStorage.removeItem(RECENT_OFF_KEY);
        else localStorage.setItem(RECENT_OFF_KEY, "1");
      } catch {
        return;
      }
      toast({ id: "offline", message: on ? t("Recent pages will be kept for offline use") : t("Recent pages are no longer kept for offline use") });
      if (on) void refreshKept(engine);
    },
  });
}

export async function toggleOffline(engine: Engine, id: string): Promise<void> {
  const ids = offlineIds(engine);
  if (ids.includes(id)) {
    setOfflineIds(
      engine,
      ids.filter((x) => x !== id),
    );
    toast({ id: "offline", message: t("This page is no longer kept for offline use") });
    return;
  }
  setOfflineIds(engine, [...ids, id]);
  void startOffline(engine);
  const pages = keptPages(engine, [id]);
  toast({ id: "offline", message: plural(pages.length, "Saving {count} page for offline use", "Saving {count} pages for offline use"), durationMs: 60_000 });
  let ok = 0;
  for (const p of pages) if (await keep(engine, p)) ok++;
  toast({
    id: "offline",
    tone: ok === pages.length ? "success" : "warning",
    message: ok === pages.length ? plural(ok, "{count} page is available offline", "{count} pages are available offline") : t("{ok} of {all} pages were saved; the rest will be tried again later", { ok, all: pages.length }),
  });
}
