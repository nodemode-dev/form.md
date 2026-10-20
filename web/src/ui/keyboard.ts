import type { Engine, Scope } from "../engine";
import { t } from "../engine/i18n";
import { editorHost } from "../extensions/blocks";
import { isTypingTarget } from "./dom";
import { closeTop, hasLayers, hasModal } from "./layers";
import { toast } from "./toast";

export const KEYS_STORAGE = "formmd.keys";

function openPageId(engine: Engine): string {
  const r = engine.state.get("route");
  return r.name === "page" ? r.id : "";
}

function findReady(engine: Engine): boolean {
  const id = openPageId(engine);
  return !!id && editorHost()?.current() === id;
}

function findWanted(): boolean {
  const el = document.activeElement;
  if (!(el instanceof HTMLElement) || !isTypingTarget(el)) return true;
  return !!el.closest("[data-view='page'], .findbar");
}

function registerFinder(engine: Engine): void {
  const { commands, keymap, state } = engine;
  const general = () => t("General");
  const pages = () => t("Pages");
  const desktop = () => (state.get("me")?.instance as { desktop?: boolean } | undefined)?.desktop === true;
  try {
    keymap.loadCustoms(JSON.parse(localStorage.getItem(KEYS_STORAGE) ?? "null"));
  } catch {
    keymap.loadCustoms(null);
  }
  setTimeout(() => (window.requestIdleCallback ?? ((f: () => void) => f()))(() => {
    void import("./command-palette").catch(() => undefined);
    void import("./find-in-page").catch(() => undefined);
  }), 1500);
  commands.register({
    id: "palette.open",
    label: () => t("Find a command"),
    group: general,
    description: "Opens the command finder, a list of every action in the app that can be searched and run from the keyboard. Does not change any data.",
    mutates: false,
    run: () => {
      let typed = "";
      const hold = (e: KeyboardEvent) => {
        if (e.key.length !== 1 || e.ctrlKey || e.metaKey || e.altKey) return;
        typed += e.key;
        e.preventDefault();
        e.stopPropagation();
      };
      window.addEventListener("keydown", hold, true);
      void import("./command-palette")
        .then((m) => {
          window.removeEventListener("keydown", hold, true);
          m.openPalette(engine, typed);
        })
        .catch((e: unknown) => {
          window.removeEventListener("keydown", hold, true);
          engine.reportError(e);
        });
    },
  });
  commands.register({
    id: "page.find",
    label: () => t("Find in page"),
    group: pages,
    description: "Opens the find and replace bar for the open page, which highlights matches in the text. Needs a page with text to be open.",
    mutates: false,
    enabled: () => findReady(engine),
    run: () => void import("./find-in-page").then((m) => m.openFind(engine), engine.reportError),
  });
  commands.register({
    id: "page.copyLink",
    label: () => t("Copy link to page"),
    group: pages,
    description: "Copies the address of the open page to the clipboard. Needs a page to be open.",
    mutates: false,
    enabled: () => !!openPageId(engine),
    run: () => {
      const id = openPageId(engine);
      if (!id) return;
      void navigator.clipboard.writeText(`${location.origin}/p/${id}`).then(
        () => toast({ tone: "success", message: t("Link copied") }),
        () => toast({ tone: "danger", message: t("Could not copy") }),
      );
    },
  });
  keymap.register({ id: "palette.open", command: "palette.open", chords: ["mod+shift+p"], scope: "global", whileTyping: true });
  keymap.register({ id: "page.find", command: "page.find", chords: ["mod+f"], scope: "global", whileTyping: true, when: findWanted });
  keymap.register({ id: "page.copyLink", command: "page.copyLink", chords: ["mod+l"], scope: "view" });
  keymap.register({ id: "page.new.desktop", command: "page.new", chords: ["mod+n"], scope: "global", whileTyping: true, when: desktop });
}

const MODAL_SCOPES: ReadonlySet<Scope> = new Set(["modal"]);
const BASE_SCOPES: ReadonlySet<Scope> = new Set(["global", "view"]);
const EDITOR_SCOPES: ReadonlySet<Scope> = new Set(["global", "view", "editor"]);

export function activeScopes(): ReadonlySet<Scope> {
  if (hasModal()) return MODAL_SCOPES;
  const el = document.activeElement;
  return el instanceof HTMLElement && el.closest(".ProseMirror") ? EDITOR_SCOPES : BASE_SCOPES;
}

export function mountKeyboard(engine: Engine): void {
  const { commands, keymap } = engine;

  commands.register({
    id: "layers.closeTop",
    label: () => t("Close the open panel or dialog"),
    group: () => "General",
    description: "Closes the topmost open panel, menu or dialog in this window.",
    mutates: false,
    enabled: () => hasLayers(),
    run: () => {
      closeTop();
    },
  });
  keymap.register({ id: "layers.closeTop.modal", command: "layers.closeTop", chords: ["escape"], scope: "modal", whileTyping: true });
  keymap.register({
    id: "layers.closeTop.global",
    command: "layers.closeTop",
    chords: ["escape"],
    scope: "global",
    whileTyping: true,
    when: () => hasLayers(),
  });
  registerFinder(engine);

  window.addEventListener(
    "keydown",
    (e) => {
      const own = e.target instanceof Element ? e.target.closest<HTMLElement>("[data-own-keys]") : null;
      const handled = keymap.handle(e, { typing: isTypingTarget(e.target), scopes: activeScopes(), only: own ? new Set((own.dataset.ownKeys ?? "").split(" ")) : undefined });
      if (handled) {
        e.preventDefault();
        e.stopPropagation();
      }
    },
    { capture: true },
  );
}
