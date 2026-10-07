import type { Engine, ThemePref } from "../engine";
import { setAttr } from "./dom";

export function mountTheme(engine: Engine): void {
  const root = document.documentElement;
  const buttons = [...document.querySelectorAll<HTMLElement>("[data-theme-choice]")];

  const apply = () => {
    const pref = engine.state.get("themePref");
    setAttr(root, "data-theme", engine.resolvedTheme());
    setAttr(root, "data-theme-pref", pref);
    for (const b of buttons) setAttr(b, "aria-pressed", b.dataset.themeChoice === pref ? "true" : "false");
  };

  engine.state.watch("themePref", apply, { immediate: true });
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
