import type { Engine, ThemePref } from "../../engine";
import { t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import type { Extension } from "../registry";

const GENERAL = () => t("General");
const APPEARANCE = () => t("Appearance");

let teardown: Unsubscribe[] = [];

export async function syncNow(engine: Engine): Promise<void> {
  engine.reconnect();
  await engine.health.check();
  const ws = engine.state.get("workspaceId");
  if (ws && engine.state.get("health").phase === "ok") await engine.pages.load(ws, true).catch(() => undefined);
}

function setup(engine: Engine): void {
  const { state, commands, keymap, platform } = engine;

  const themeCommand = (pref: ThemePref, label: string) =>
    commands.register({
      id: `shell.theme.${pref}`,
      label: () => t(label),
      group: APPEARANCE,
      run: () => state.set("themePref", pref),
    });

  teardown = [
    commands.register({
      id: "shell.toggleSidebar",
      label: () => t("Show or hide the sidebar"),
      group: GENERAL,
      run: () => {
        if (platform.isNarrow()) state.set("drawer", !state.get("drawer"));
        else state.set("sidebar", state.get("sidebar") === "collapsed" ? "expanded" : "collapsed");
      },
    }),
    commands.register({
      id: "shell.showShortcuts",
      label: () => t("Keyboard shortcuts"),
      group: GENERAL,
      run: () => state.set("overlay", "shortcuts"),
    }),
    commands.register({
      id: "shell.sync",
      label: () => t("Sync now"),
      group: GENERAL,
      run: () => void syncNow(engine),
    }),
    commands.register({
      id: "shell.checkConnection",
      label: () => t("Check the server connection"),
      group: GENERAL,
      run: () => engine.health.check(),
    }),
    commands.register({
      id: "shell.toggleTheme",
      label: () => t("Switch between light and dark"),
      group: APPEARANCE,
      run: () => state.set("themePref", engine.resolvedTheme() === "dark" ? "light" : "dark"),
    }),
    themeCommand("light", "Light theme"),
    themeCommand("dark", "Dark theme"),
    themeCommand("auto", "Match the system theme"),

    keymap.register({ id: "shell.toggleSidebar.mod", command: "shell.toggleSidebar", chords: ["mod+\\"], scope: "global", whileTyping: true }),
    keymap.register({ id: "shell.toggleSidebar.key", command: "shell.toggleSidebar", chords: ["["], scope: "global" }),
    keymap.register({ id: "shell.showShortcuts", command: "shell.showShortcuts", chords: ["?", "mod+/"], scope: "global" }),
    keymap.register({ id: "shell.toggleTheme", command: "shell.toggleTheme", chords: ["mod+shift+l"], scope: "global", whileTyping: true }),
  ];
}

const shell: Extension = {
  id: "shell",
  setup,
  dispose() {
    for (const off of teardown) off();
    teardown = [];
  },
};

export default shell;
