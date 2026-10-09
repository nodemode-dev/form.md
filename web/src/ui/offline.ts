import type { Engine } from "../engine";
import { plural, t } from "../engine/i18n";
import { offlineIds, setOfflineIds } from "./offline-list";
import { toast } from "./toast";

let started: Promise<ServiceWorkerRegistration | null> | null = null;

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
    const shell = () => {
      if (navigator.onLine && engine.state.get("me")) post(reg, { type: "offline-shell" });
    };
    shell();
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
  for (const id of roots) for (const p of engine.pages.subtree(id)) if (p.kind === "doc") seen.add(p.id);
  return [...seen];
}

async function refreshKept(engine: Engine): Promise<void> {
  if (!navigator.onLine) return;
  for (const id of keptPages(engine, offlineIds(engine))) {
    if (!navigator.onLine) return;
    await syncDoc(engine, id);
  }
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
  for (const p of pages) if (await syncDoc(engine, p)) ok++;
  toast({
    id: "offline",
    tone: ok === pages.length ? "success" : "warning",
    message: ok === pages.length ? plural(ok, "{count} page is available offline", "{count} pages are available offline") : t("{ok} of {all} pages were saved; the rest will be tried again later", { ok, all: pages.length }),
  });
}
