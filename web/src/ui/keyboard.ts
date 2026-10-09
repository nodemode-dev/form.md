import type { Engine, Scope } from "../engine";
import { t } from "../engine/i18n";
import { isTypingTarget } from "./dom";
import { closeTop, hasLayers, hasModal } from "./layers";

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

  window.addEventListener(
    "keydown",
    (e) => {
      const handled = keymap.handle(e, { typing: isTypingTarget(e.target), scopes: activeScopes() });
      if (handled) {
        e.preventDefault();
        e.stopPropagation();
      }
    },
    { capture: true },
  );
}
