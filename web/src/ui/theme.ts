import type { Engine, Me, Prefs, ThemePref } from "../engine";
import { accentTone, type KeyValueStore, type Look, normalizeServerLook, PREF_KEYS, readLook, writeLook } from "../engine/prefs";
import { State } from "../engine/state";
import { setAttr } from "./dom";

const store: KeyValueStore = {
  get(key) {
    try {
      return localStorage.getItem(key);
    } catch {
      return null;
    }
  },
  set(key, value) {
    try {
      localStorage.setItem(key, value);
    } catch {
      return;
    }
  },
};

const looks = new State<{ look: Look }>({ look: readLook(store) });

export function currentLook(): Look {
  return looks.get("look");
}

export function watchLook(fn: (look: Look) => void): () => void {
  return looks.watch("look", fn);
}

export function setLook(patch: Partial<Omit<Look, "at">>): void {
  looks.set("look", { ...looks.get("look"), ...patch, at: Date.now() });
}

export function mountTheme(engine: Engine): void {
  const root = document.documentElement;
  const buttons = [...document.querySelectorAll<HTMLElement>("[data-theme-choice]")];
  const more = matchMedia("(prefers-contrast: more)");
  const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]');
  let adopting = false;

  const paintMeta = () => {
    const ground = getComputedStyle(root).getPropertyValue("--ground").trim();
    if (meta && ground && meta.content !== ground) meta.content = ground;
  };

  let stillFrame = 0;
  const still = () => {
    setAttr(root, "data-switching", "");
    cancelAnimationFrame(stillFrame);
    stillFrame = requestAnimationFrame(() => (stillFrame = requestAnimationFrame(() => setAttr(root, "data-switching", null))));
  };

  const paintLook = () => {
    const l = looks.get("look");
    still();
    setAttr(root, "data-dark-theme", l.dark);
    setAttr(root, "data-light-theme", l.light);
    setAttr(root, "data-density", l.density);
    setAttr(root, "data-contrast-pref", l.contrast);
    setAttr(root, "data-contrast", l.contrast === "high" || (l.contrast === "auto" && more.matches) ? "high" : null);
    if (l.accent) {
      if (root.style.getPropertyValue("--accent-pick") !== l.accent) root.style.setProperty("--accent-pick", l.accent);
      setAttr(root, "data-accent", accentTone(l.accent));
    } else {
      root.style.removeProperty("--accent-pick");
      setAttr(root, "data-accent", null);
    }
    paintMeta();
  };

  let timer: ReturnType<typeof setTimeout> | undefined;
  let sent = 0;
  const push = () => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      const me = engine.state.get("me");
      if (!me || me.credential.scope === "read") return;
      const appearance = { ...looks.get("look"), mode: engine.state.get("themePref") };
      sent = appearance.at;
      void engine.api.me.update({ prefs: { ...me.user.prefs, appearance } as Prefs }).then(
        (next) => engine.setMe(next),
        () => undefined,
      );
    }, 800);
  };

  const adopt = (me: Me | null) => {
    if (!me) return;
    const server = normalizeServerLook((me.user.prefs as Prefs & { appearance?: unknown }).appearance);
    const local = looks.get("look");
    if (!server || server.at < local.at) {
      if (local.at > 0 && local.at !== sent) push();
      return;
    }
    if (server.at === local.at) return;
    const { mode, ...look } = server;
    adopting = true;
    looks.set("look", look);
    engine.state.set("themePref", mode);
    adopting = false;
  };

  paintLook();
  looks.watch("look", (l) => {
    writeLook(store, l);
    paintLook();
    if (!adopting) push();
  });
  more.addEventListener("change", paintLook);
  window.addEventListener("storage", (e) => {
    if (e.key !== PREF_KEYS.look) return;
    const next = readLook(store);
    if (next.at <= looks.get("look").at) return;
    adopting = true;
    looks.set("look", next);
    adopting = false;
  });

  const fresh = (): boolean => {
    try {
      return localStorage.getItem("formmd.theme") === null;
    } catch {
      return false;
    }
  };
  if (fresh()) engine.state.set("themePref", "auto");

  const apply = () => {
    const pref = engine.state.get("themePref");
    if (root.getAttribute("data-theme") !== engine.resolvedTheme()) still();
    setAttr(root, "data-theme", engine.resolvedTheme());
    setAttr(root, "data-theme-pref", pref);
    for (const b of buttons) setAttr(b, "aria-pressed", b.dataset.themeChoice === pref ? "true" : "false");
    paintMeta();
  };

  engine.state.watch("themePref", apply, { immediate: true });
  engine.state.watch("themePref", () => {
    if (!adopting) looks.set("look", { ...looks.get("look"), at: Date.now() });
  });
  engine.state.watch("me", adopt, { immediate: true });
  matchMedia("(prefers-color-scheme: light)").addEventListener("change", () => {
    if (engine.state.get("themePref") === "auto") apply();
  });

  for (const b of buttons) {
    b.addEventListener("click", () => {
      const choice = b.dataset.themeChoice as ThemePref | undefined;
      if (choice) engine.commands.run(`shell.theme.${choice}`);
    });
  }
}
