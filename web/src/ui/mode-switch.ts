import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import type { EditMode, EditorHost } from "../extensions/blocks/editor";
import { h, ref, setAttr, setText } from "./dom";
import { type IconName, iconNode } from "./icons";
import { menuButton } from "./menu";

const MODES: readonly (readonly [EditMode, IconName, () => string, () => string])[] = [
  ["edit", "pencil", () => t("Editing"), () => t("Change the page directly")],
  ["suggest", "comment", () => t("Suggesting"), () => t("Typing becomes suggestions an editor accepts")],
  ["view", "eye", () => t("Viewing"), () => t("Read without changing anything")],
];

export function mountModeSwitch(engine: Engine, host: EditorHost): void {
  const btn = ref<HTMLButtonElement>("act-mode");
  const paint = () => {
    const r = engine.state.get("route");
    const id = r.name === "page" ? r.id : "";
    const p = id ? (engine.pages.get(id) ?? engine.state.get("detail")) : null;
    const show = !!p && p.id === id && p.kind !== "collection" && host.current() === id && (host.canEdit() || host.canSuggest());
    btn.hidden = !show;
    if (!show) return;
    const mode = host.editMode();
    const def = MODES.find(([m]) => m === mode) ?? MODES[2];
    if (!def) return;
    if (btn.dataset.mode !== mode) {
      btn.dataset.mode = mode;
      btn.replaceChildren(iconNode(def[1]), h("span"));
    }
    setText(btn.lastElementChild as HTMLElement, def[2]());
    setAttr(btn, "aria-label", t("Mode: {mode}", { mode: def[2]() }));
  };
  menuButton(
    btn,
    () =>
      MODES.filter(([m]) => m === "view" || (m === "edit" ? host.canEdit() : host.canSuggest())).map(([m, icon, label, hint]) => ({
        label: label(),
        icon,
        hint: hint(),
        checked: host.editMode() === m,
        run: () => host.setMode(m),
      })),
    { side: "bottom", align: "end", label: t("Mode") },
  );
  host.onMode(paint);
  host.onShown(paint);
  engine.comments.subscribe(paint);
  engine.state.watch("route", paint);
  engine.state.watch("detail", paint);
  paint();
}
