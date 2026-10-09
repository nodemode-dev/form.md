import type { Engine, Workspace } from "../engine";
import { plural, t } from "../engine/i18n";
import { rememberWorkspace } from "./auth";
import { tile, updateAvatar, updateTile } from "./avatar";
import { ref, setAttr, setText } from "./dom";
import type { IconName } from "./icons";
import { type MenuEntry, menuButton } from "./menu";

function roleHint(w: Workspace): string {
  if (w.break_glass) return t("Emergency");
  if (w.personal) return t("Private");
  if (w.role === "guest") return t("Guest");
  return w.role === "viewer" ? t("Read-only") : "";
}

export function mountSession(engine: Engine): void {
  const { state, commands, keymap } = engine;
  const wsSwitch = ref<HTMLButtonElement>("ws-switch");
  const wsTile = ref("ws-tile");
  const wsName = ref("ws-name");
  const chip = ref<HTMLButtonElement>("user-chip");
  const avatarEl = ref("user-avatar");
  const userName = ref("user-name");

  const paintWorkspace = () => {
    const w = engine.workspace();
    const name = w?.name ?? "";
    updateTile(wsTile, name, w?.icon ?? "", w?.image ?? "");
    setText(wsName, name);
    setAttr(wsSwitch, "aria-label", t("Workspace: {name}", { name }));
  };
  const paintUser = () => {
    const u = state.get("me")?.user;
    if (!u) return;
    updateAvatar(avatarEl, u.name, u.color, u.avatar);
    setText(userName, u.name);
    setAttr(document.documentElement, "data-font", u.prefs.system_font ? "system" : null);
    setAttr(chip, "aria-label", t("Account menu for {name}", { name: u.name }));
  };

  state.watch("me", () => {
    paintUser();
    paintWorkspace();
  });
  state.watch("workspaceId", (id) => {
    if (id) rememberWorkspace(id);
    paintWorkspace();
  });
  paintUser();
  paintWorkspace();
  const startWs = state.get("workspaceId");
  if (startWs) rememberWorkspace(startWs);

  const commandItem = (id: string, icon: IconName, danger = false): MenuEntry[] => {
    const c = commands.get(id);
    if (!c || c.enabled?.() === false) return [];
    const chord = keymap.chordFor(id);
    return [{ label: c.label(), icon, hint: chord ? chord.join(" ") : "", danger, run: () => commands.run(id) }];
  };

  menuButton(
    wsSwitch,
    () => {
      const me = state.get("me");
      const current = state.get("workspaceId");
      const entries: MenuEntry[] = [{ kind: "heading", label: t("Workspaces") }];
      for (const w of me?.workspaces ?? []) {
        entries.push({ label: w.name, lead: tile(w.name, w.icon, w.image), hint: roleHint(w), checked: w.id === current, run: () => state.set("workspaceId", w.id) });
      }
      const withheld = me?.withheld_workspaces ?? 0;
      if (withheld > 0 && me?.user.role === "owner") {
        entries.push({
          label: plural(withheld, "{count} more workspace", "{count} more workspaces"),
          icon: "lock",
          hint: t("Emergency access"),
          run: () => engine.openOverlay("admin", "workspaces"),
        });
      }
      const tail = [...commandItem("workspace.settings", "settings"), ...commandItem("workspace.members", "users"), ...commandItem("workspace.new", "plus")];
      if (tail.length) entries.push({ kind: "separator" }, ...tail);
      return entries;
    },
    { side: "bottom", align: "start", label: t("Workspaces") },
  );

  menuButton(
    chip,
    () => {
      const me = state.get("me");
      const entries: MenuEntry[] = [];
      if (me) entries.push({ kind: "heading", label: me.user.email });
      entries.push(...commandItem("account.open", "user"), ...commandItem("account.languageTime", "globe"), ...commandItem("admin.open", "shield"));
      entries.push({ kind: "separator" }, ...commandItem("shell.showShortcuts", "keyboard"));
      entries.push({ kind: "separator" }, ...commandItem("account.signOut", "logout", true));
      return entries;
    },
    { side: "top", align: "start", label: t("Account") },
  );
}
