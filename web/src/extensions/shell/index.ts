import type { Engine, ThemePref } from "../../engine";
import type { Unsubscribe } from "../../engine/state";
import type { Extension } from "../registry";

const GENERAL = () => "General";
const APPEARANCE = () => "Appearance";

let teardown: Unsubscribe[] = [];

function setup(engine: Engine): void {
  const { state, commands, keymap, platform } = engine;

  const themeCommand = (pref: ThemePref, label: string) =>
    commands.register({
      id: `shell.theme.${pref}`,
      label: () => label,
      group: APPEARANCE,
      run: () => state.set("themePref", pref),
    });

  teardown = [
    commands.register({
      id: "shell.toggleSidebar",
      label: () => "Show or hide the sidebar",
      group: GENERAL,
      run: () => {
        if (platform.isNarrow()) state.set("drawer", !state.get("drawer"));
        else state.set("sidebar", state.get("sidebar") === "collapsed" ? "expanded" : "collapsed");
      },
    }),
    commands.register({
      id: "shell.showShortcuts",
      label: () => "Keyboard shortcuts",
      group: GENERAL,
      run: () => state.set("overlay", "shortcuts"),
    }),
    commands.register({
      id: "shell.checkConnection",
      label: () => "Check the server connection",
      group: GENERAL,
      run: () => engine.health.check(),
    }),
    commands.register({
      id: "shell.toggleTheme",
      label: () => "Switch between light and dark",
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
