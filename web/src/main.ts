import { type AuthState, type Boot, createEngine, type Engine } from "./engine";
import { createClient } from "./engine/client";
import { resolveLang, setLang } from "./engine/i18n";
import account from "./extensions/account";
import blocks from "./extensions/blocks";
import database from "./extensions/database";
import { mountPublicForm } from "./extensions/database/publicform";
import pagesExtension from "./extensions/pages";
import { activateAll, register } from "./extensions/registry";
import shell from "./extensions/shell";
import { mountAccount } from "./ui/account";
import { mountAdmin } from "./ui/admin";
import { mountUI, platformAdapter, storageAdapter } from "./ui/app";
import { mountAuth } from "./ui/auth";
import { errorMessage } from "./ui/errors";
import { mountLibraryView } from "./ui/library-view";
import { mountPageView } from "./ui/page-view";
import { mountHome, mountRoutes } from "./ui/routes";
import { mountSession } from "./ui/session";
import { toast } from "./ui/toast";
import { mountTopbar } from "./ui/topbar";
import { mountTrashView } from "./ui/trash-view";
import { mountTree } from "./ui/tree";
import { mountWorkspaceSettings } from "./ui/workspace-settings";

const AUTH_STATES: readonly AuthState[] = ["pending", "setup", "signed-out", "signup", "invite", "app", "form"];

function readBoot(): Boot {
  try {
    return (JSON.parse(document.getElementById("boot")?.textContent ?? "null") as Boot | null) ?? {};
  } catch {
    return {};
  }
}

function readAuth(): AuthState {
  const v = document.documentElement.dataset.auth as AuthState | undefined;
  return v && AUTH_STATES.includes(v) ? v : "pending";
}

async function devBoot(fetchFn: typeof fetch): Promise<{ auth: AuthState; boot: Boot }> {
  const api = createClient(fetchFn);
  const root = document.documentElement;
  const path = location.pathname;
  if (path.startsWith("/form/")) return { auth: "form", boot: {} };
  const policy = await api.auth.policy();
  if (policy.setup_required) return { auth: "setup", boot: { setup: true } };
  const signup = policy.signup === "invite" ? "closed" : "open";
  root.dataset.signup = signup;
  root.dataset.sso = policy.sso.map((p) => p.id).join(" ");
  const boot: Boot = { sso: policy.sso, signup, instance_name: policy.instance_name };
  try {
    boot.me = await api.me.get();
    return { auth: path.startsWith("/invite/") ? "invite" : "app", boot };
  } catch {
    if (path.startsWith("/invite/")) return { auth: "invite", boot };
    return { auth: path === "/signup" && signup === "open" ? "signup" : "signed-out", boot };
  }
}

function guardSession(base: typeof fetch, engine: () => Engine | null): typeof fetch {
  let leaving = false;
  return async (input, init) => {
    const res = await base(input, init);
    if (res.status === 401 && !leaving && engine()?.state.get("auth") === "app") {
      const body = (await res
        .clone()
        .json()
        .catch(() => null)) as { error?: { code?: string } } | null;
      if (body?.error?.code === "unauthorized") {
        leaving = true;
        location.assign(`/login?next=${encodeURIComponent(location.pathname + location.search)}`);
      }
    }
    return res;
  };
}

async function start(): Promise<void> {
  let engine: Engine | null = null;
  const fetchFn = guardSession(window.fetch.bind(window), () => engine);
  let auth = readAuth();
  let boot = readBoot();
  if (auth === "pending") ({ auth, boot } = await devBoot(fetchFn));

  await setLang(resolveLang(boot.me?.user.prefs.lang ?? "", document.documentElement.lang || navigator.language));

  engine = createEngine({
    storage: storageAdapter(),
    fetch: fetchFn,
    platform: platformAdapter(),
    now: () => performance.now(),
    boot,
    auth,
  });

  register(shell);
  register(account);
  register(pagesExtension);
  register(blocks);
  register(database);
  activateAll(engine);
  mountUI(engine);
  if (auth === "form") {
    document.documentElement.dataset.auth = "form";
    mountPublicForm(engine);
    return;
  }
  mountAuth(engine);
  if (auth === "app") {
    engine.onError((e) => toast({ id: "error", tone: "danger", message: errorMessage(e) }));
    mountRoutes(engine);
    mountSession(engine);
    mountTree(engine);
    mountTopbar(engine);
    mountPageView(engine);
    mountLibraryView(engine);
    mountTrashView(engine);
    mountHome(engine);
    mountAccount(engine);
    mountWorkspaceSettings(engine);
    mountAdmin(engine);
    engine.start();
  }
}

void start();
