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
      description: "Sets the color theme for this browser to light, dark or matching the system, depending on the command.",
      mutates: false,
      run: () => state.set("themePref", pref),
    });

  teardown = [
    commands.register({
      id: "shell.toggleSidebar",
      label: () => t("Show or hide the sidebar"),
      group: GENERAL,
      description: "Shows or hides the sidebar in this window, or the sidebar drawer on a narrow screen.",
      mutates: false,
      run: () => {
        if (platform.isNarrow()) state.set("drawer", !state.get("drawer"));
        else state.set("sidebar", state.get("sidebar") === "collapsed" ? "expanded" : "collapsed");
      },
    }),
    commands.register({
      id: "shell.showShortcuts",
      label: () => t("Keyboard shortcuts"),
      group: GENERAL,
      description: "Opens the dialog that lists the keyboard shortcuts.",
      mutates: false,
      run: () => state.set("overlay", "shortcuts"),
    }),
    commands.register({
      id: "shell.sync",
      label: () => t("Sync now"),
      group: GENERAL,
      description: "Reconnects to the server, checks that it is reachable and reloads the page list for the current workspace.",
      mutates: false,
      run: () => void syncNow(engine),
    }),
    commands.register({
      id: "shell.checkConnection",
      label: () => t("Check the server connection"),
      group: GENERAL,
      description: "Checks whether the server can be reached and updates the connection status shown in the window.",
      mutates: false,
      run: () => engine.health.check(),
    }),
    commands.register({
      id: "shell.theme.toggle",
      label: () => t("Switch between light and dark"),
      group: APPEARANCE,
      description: "Switches between the light and dark theme.",
      mutates: false,
      run: () => state.set("themePref", engine.resolvedTheme() === "dark" ? "light" : "dark"),
    }),
    commands.register({
      id: "prefs.open",
      label: () => t("Preferences"),
      group: APPEARANCE,
      description: "Opens Preferences in the account settings: appearance, themes, accent colour, density, contrast, language and time.",
      mutates: false,
      enabled: () => state.get("auth") === "app" && state.get("me") !== null,
      run: () => engine.openOverlay("account", "preferences"),
    }),
    commands.register({
      id: "route.home",
      label: () => t("Home"),
      group: GENERAL,
      description: "Opens the workspace's Home page: recent pages, material to import and templates.",
      mutates: false,
      enabled: () => state.get("auth") === "app",
      run: () => engine.navigate({ name: "home" }),
    }),
    commands.register({
      id: "templates.open",
      label: () => t("New from a template"),
      group: GENERAL,
      description: "Opens the template gallery to start a page or database from a template.",
      mutates: false,
      enabled: () => state.get("auth") === "app" && engine.canWrite(),
      run: () => void import("../../ui/templates-dialog").then((m) => m.openTemplates(engine), engine.reportError),
    }),
    commands.register({
      id: "transfer.import",
      label: () => t("Import"),
      group: GENERAL,
      description: "Opens the import dialog for Markdown, CSV, HTML, PDF, Word and the other supported files.",
      mutates: false,
      enabled: () => state.get("auth") === "app" && engine.canWrite(),
      run: () => void import("../../ui/transfer-dialog").then((m) => m.openImport(engine, { workspaceId: state.get("workspaceId") }), engine.reportError),
    }),
    themeCommand("light", "Light theme"),
    themeCommand("dark", "Dark theme"),
    themeCommand("auto", "Match the system theme"),

    keymap.register({ id: "shell.toggleSidebar.mod", command: "shell.toggleSidebar", chords: ["mod+\\"], scope: "global", whileTyping: true }),
    keymap.register({ id: "shell.toggleSidebar.key", command: "shell.toggleSidebar", chords: ["["], scope: "global" }),
    keymap.register({ id: "shell.showShortcuts", command: "shell.showShortcuts", chords: ["?", "mod+/"], scope: "global" }),
    keymap.register({ id: "shell.theme.toggle", command: "shell.theme.toggle", chords: ["mod+shift+l"], scope: "global", whileTyping: true }),
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
