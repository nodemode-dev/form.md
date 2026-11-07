import "./main.css";
import { createEngine, type Boot, type Engine } from "../../../web/src/engine";
import { ApiError } from "../../../web/src/engine/api";
import { plural, resolveLang, setLang } from "../../../web/src/engine/i18n";
import account from "../../../web/src/extensions/account";
import blocks from "../../../web/src/extensions/blocks";
import pagesExtension from "../../../web/src/extensions/pages";
import { activateAll, register } from "../../../web/src/extensions/registry";
import shell from "../../../web/src/extensions/shell";
import writing from "../../../web/src/extensions/writing";
import { mountUI, storageAdapter } from "../../../web/src/ui/app";
import { mountCommentsPane } from "../../../web/src/ui/comments-pane";
import { revealApp } from "../../../web/src/ui/dom";
import { errorMessage } from "../../../web/src/ui/errors";
import { mountInboxView } from "../../../web/src/ui/inbox-view";
import { prefetchPageDialogs } from "../../../web/src/ui/page-actions";
import { mountPageView } from "../../../web/src/ui/page-view";
import { mountHome, mountRoutes } from "../../../web/src/ui/routes";
import { mountSession } from "../../../web/src/ui/session";
import { toast } from "../../../web/src/ui/toast";
import { mountTopbar } from "../../../web/src/ui/topbar";
import { mountTrashView } from "../../../web/src/ui/trash-view";
import { mountTree } from "../../../web/src/ui/tree";
import { demoPlatform } from "./platform";
import { demoFetch } from "./server/api";
import { pages, favorites, state } from "./server/db";
import "./server/files";
import { seed, WS_ID } from "./server/seed";
import { installGuards } from "./guards";

declare global {
  interface Window {
    __demoOrigin?: string;
    __demoHost?: string;
    __engine?: Engine;
  }
}

window.__demoOrigin = "https://fielddesk.example";
window.__demoHost = "fielddesk.example";

async function start(): Promise<void> {
  seed();
  await setLang(resolveLang("en", "en"));
  const ws = state.workspace!;
  const me = {
    user: state.me!,
    credential: { kind: "session" as const, scope: "write" as const },
    instance: { name: "form.md" },
    workspaces: [ws],
    withheld_workspaces: 0,
    can_create_workspaces: false,
  };
  const boot: Boot = {
    me,
    instance_name: "form.md",
    workspace_id: WS_ID,
    seq: 0,
    pages: pages.filter((p) => !p.trashed_at),
    favorites: favorites.map((id) => ({ ...pages.find((p) => p.id === id)!, favorite_key: `fav-${id}` })),
    tags: [],
    teamspaces: [],
    inbox_unread: 0,
  };

  const store = storageAdapter();
  const engine = createEngine({
    storage: { get: (k) => store.get(k) ?? (k === "formmd.theme" ? "dark" : null), set: (k, v) => store.set(k, v) },
    fetch: demoFetch as typeof fetch,
    platform: demoPlatform(),
    now: () => performance.now(),
    boot,
    auth: "app",
  });
  window.__engine = engine;

  register(shell);
  register(account);
  register(pagesExtension);
  register(blocks);
  register(writing);
  activateAll(engine);
  mountUI(engine);
  engine.onError((e) => {
    if (e instanceof ApiError && e.code === "network") return;
    toast({ id: "error", tone: "danger", message: errorMessage(e) });
  });
  mountRoutes(engine);
  mountSession(engine);
  mountTree(engine);
  mountTopbar(engine);
  void import("../../../web/src/ui/shell-extras").then((m) => m.mountShellExtras(engine), engine.reportError);
  mountPageView(engine);
  mountCommentsPane(engine);
  mountInboxView(engine);
  mountTrashView(engine);
  mountHome(engine);
  installGuards(engine);
  engine.collections.onQueued = (n) => toast({ id: "row-outbox", message: plural(n, "{count} change is saved on this device", "{count} changes are saved on this device") });
  setTimeout(() => ("requestIdleCallback" in window ? requestIdleCallback(prefetchPageDialogs) : prefetchPageDialogs()), 1500);
  engine.start();
  revealApp();
}

void start();
