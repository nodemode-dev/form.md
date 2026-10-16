import type { Engine } from "../../engine";
import { t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import { registerStyleCommands } from "../../ui/page-style";
import { toast } from "../../ui/toast";
import type { Extension } from "../registry";

const PAGES = () => t("Pages");
const TABS = () => t("Tabs");

let teardown: Unsubscribe[] = [];

function currentId(engine: Engine): string {
  const r = engine.state.get("route");
  return r.name === "page" ? r.id : "";
}

function setup(engine: Engine): void {
  const { state, commands, keymap, pages } = engine;
  const focusTitle = () =>
    requestAnimationFrame(() =>
      requestAnimationFrame(() => {
        const el = document.querySelector<HTMLTextAreaElement>(".page-title");
        if (el && !el.readOnly) el.focus();
      }),
    );

  teardown = [
    commands.register({
      id: "page.new",
      label: () => t("New page"),
      group: PAGES,
      description: "Creates a new empty page at the top level of the current workspace and opens it with the title focused. Needs write access to the workspace.",
      mutates: true,
      enabled: () => engine.canWrite(),
      run: async () => {
        const ws = state.get("workspaceId");
        if (!ws) return;
        try {
          const p = await pages.create(ws);
          engine.openPage(p.id);
          focusTitle();
        } catch (e) {
          engine.reportError(e);
        }
      },
    }),
    commands.register({
      id: "page.newDatabase",
      label: () => t("New database"),
      group: PAGES,
      description: "Creates a new database (a page that holds rows) in the current workspace and opens it. Needs write access to the workspace.",
      mutates: true,
      enabled: () => engine.canWrite(),
      run: async () => {
        const ws = state.get("workspaceId");
        if (!ws) return;
        try {
          const r = await engine.api.pages.createCollection({ workspace_id: ws });
          pages.upsert(r.page);
          engine.openPage(r.page.id);
          focusTitle();
        } catch (e) {
          engine.reportError(e);
        }
      },
    }),
    commands.register({
      id: "page.newInside",
      label: () => t("New page inside this one"),
      group: PAGES,
      description: "Creates a new empty page as a child of the page that is currently open and opens it. Needs an open page that the user can add content to.",
      mutates: true,
      enabled: () => !!currentId(engine) && engine.can(currentId(engine), "content"),
      run: async () => {
        const parent = pages.get(currentId(engine));
        if (!parent) return;
        try {
          const p = await pages.create(parent.workspace_id, parent.id);
          engine.openPage(p.id);
          focusTitle();
        } catch (e) {
          engine.reportError(e);
        }
      },
    }),
    commands.register({
      id: "page.favorite",
      label: () => t("Star or unstar this page"),
      group: PAGES,
      description: "Adds the open page to the user's favorites, or removes it if it is already a favorite. Needs a page to be open.",
      mutates: true,
      enabled: () => !!currentId(engine),
      run: () => void pages.toggleFavorite(currentId(engine)).catch(engine.reportError),
    }),
    commands.register({
      id: "page.trash",
      label: () => t("Move to trash"),
      group: PAGES,
      description: "Moves the open page to the trash, from where it can be restored. Needs a page to be open that the user can edit.",
      mutates: true,
      enabled: () => {
        const p = pages.get(currentId(engine));
        return !!p && engine.can(p.id, "edit");
      },
      run: async () => {
        const id = currentId(engine);
        const p = pages.get(id);
        if (!p) return;
        try {
          await pages.trash(id);
          toast({ message: t("Moved “{title}” to the trash", { title: p.title || t("Untitled") }), action: { label: t("Undo"), run: () => void pages.restore(id).then(() => engine.openPage(id), engine.reportError) } });
        } catch (e) {
          engine.reportError(e);
        }
      },
    }),
    commands.register({
      id: "library.open",
      label: () => t("Library"),
      group: PAGES,
      description: "Switches the view to the library, showing recently opened pages. Does not change any data.",
      mutates: false,
      run: () => engine.navigate({ name: "library", view: "recent" }),
    }),
    commands.register({
      id: "search.open",
      label: () => t("Search"),
      group: PAGES,
      description: "Opens the search dialog so the user can look for pages. Needs a workspace to be selected.",
      mutates: false,
      enabled: () => !!state.get("workspaceId"),
      run: () => state.set("overlay", "search"),
    }),
    commands.register({
      id: "trash.open",
      label: () => t("Trash"),
      group: PAGES,
      description: "Switches the view to the trash, which lists pages that were moved to the trash. Does not change any data.",
      mutates: false,
      run: () => engine.navigate({ name: "trash" }),
    }),
    commands.register({
      id: "notes.toggle",
      label: () => t("Notes mode"),
      group: PAGES,
      description: "Turns the user's notes mode preference on or off and saves it to their account. Only available in a signed-in browser session.",
      mutates: true,
      enabled: () => state.get("me")?.credential.kind === "session",
      run: async () => {
        const me = state.get("me");
        if (!me) return;
        try {
          engine.setMe(await engine.api.me.update({ prefs: { ...me.user.prefs, notes_mode: !me.user.prefs.notes_mode } }));
        } catch (e) {
          engine.reportError(e);
        }
      },
    }),
    commands.register({
      id: "tabs.next",
      label: () => t("Next tab"),
      group: TABS,
      description: "Switches to the next open tab. Needs more than one tab to be open.",
      mutates: false,
      enabled: () => state.get("tabs").length > 1,
      run: () => void engine.cycleTab(1),
    }),
    commands.register({
      id: "tabs.prev",
      label: () => t("Previous tab"),
      group: TABS,
      description: "Switches to the previous open tab. Needs more than one tab to be open.",
      mutates: false,
      enabled: () => state.get("tabs").length > 1,
      run: () => void engine.cycleTab(-1),
    }),
    commands.register({
      id: "tabs.close",
      label: () => t("Close tab"),
      group: TABS,
      description: "Closes the tab of the open page in this window. The page itself is not changed. Needs an open page that has a tab.",
      mutates: false,
      enabled: () => !!currentId(engine) && state.get("tabs").includes(currentId(engine)),
      run: () => engine.closeTab(currentId(engine)),
    }),

    keymap.register({ id: "page.new", command: "page.new", chords: ["alt+n"], scope: "global", whileTyping: true }),
    ...registerStyleCommands(engine),
    keymap.register({ id: "page.favorite", command: "page.favorite", chords: ["f"], scope: "view" }),
    keymap.register({ id: "library.open", command: "library.open", chords: ["l"], scope: "view" }),
    keymap.register({ id: "search.open", command: "search.open", chords: ["mod+k", "mod+p"], scope: "global", whileTyping: true }),
    keymap.register({ id: "notes.toggle", command: "notes.toggle", chords: ["m"], scope: "view" }),
    keymap.register({ id: "tabs.next", command: "tabs.next", chords: ["mod+alt+arrowright", "alt+shift+arrowright"], scope: "global", whileTyping: true }),
    keymap.register({ id: "tabs.prev", command: "tabs.prev", chords: ["mod+alt+arrowleft", "alt+shift+arrowleft"], scope: "global", whileTyping: true }),
    keymap.register({ id: "tabs.close", command: "tabs.close", chords: ["alt+w"], scope: "global", whileTyping: true }),
  ];
}

const pagesExtension: Extension = {
  id: "pages",
  setup,
  dispose() {
    for (const off of teardown) off();
    teardown = [];
  },
};

export default pagesExtension;
