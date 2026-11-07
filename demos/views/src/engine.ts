import type { Engine, EngineState } from "../../../web/src/engine";
import { createClient } from "../../../web/src/engine/client";
import { CollectionStore } from "../../../web/src/engine/collections";
import { State } from "../../../web/src/engine/state";
import { toast } from "../../../web/src/ui/toast";
import { createBackend, type Backend, USER, WORKSPACE } from "./server";
import { seedCollections } from "./seed";

export interface Demo {
  engine: Engine;
  backend: Backend;
  onOpen(fn: (id: string) => void): void;
}

export function createDemo(): Demo {
  const backend = createBackend(seedCollections);
  const api = createClient(backend.fetch);
  const reportError = (e: unknown) => {
    const message = e instanceof Error ? e.message : "Something went wrong";
    toast({ id: "demo-error", tone: "warning", message });
  };
  const collections = new CollectionStore(api, reportError);
  backend.attach(collections);
  const persist = collections.updateView.bind(collections);
  collections.updateView = (colId, viewId, change, debounce) => {
    const stored = backend.collection(colId)?.views.find((v) => v.id === viewId);
    if (stored && change.config) stored.config = structuredClone(change.config);
    return persist(colId, viewId, change, debounce);
  };
  const state = new State<Pick<EngineState, "me" | "workspaceId" | "route" | "tabs">>({
    me: { user: { id: USER, name: "Demo", email: "", prefs: {} }, workspaces: [], instance: {} } as unknown as EngineState["me"],
    workspaceId: WORKSPACE,
    route: { name: "home" } as EngineState["route"],
    tabs: [],
  });
  let open: (id: string) => void = () => undefined;
  const pages = {
    get: () => undefined,
    all: () => [],
    isLoaded: () => false,
    subscribe: () => () => undefined,
    edit: () => undefined,
    upsert: () => undefined,
    children: () => [],
    ancestors: () => [],
  };
  const engine = {
    state,
    api,
    collections,
    pages,
    platform: { mac: /Mac|iPhone|iPad/.test(navigator.platform), isNarrow: () => window.innerWidth < 760, prefersLight: () => false },
    boot: {},
    reportError,
    openPage: (id: string) => open(id),
    navigate: () => undefined,
    workspace: () => null,
    workspaceById: () => null,
    can: () => true,
    canWrite: () => true,
    refreshMe: async () => undefined,
    resolvedTheme: () => (document.documentElement.getAttribute("data-theme") === "light" ? "light" : "dark"),
  } as unknown as Engine;
  return { engine, backend, onOpen: (fn) => (open = fn) };
}
