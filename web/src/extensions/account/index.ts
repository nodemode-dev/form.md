import type { Engine } from "../../engine";
import { t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import type { Extension } from "../registry";

const ACCOUNT = () => t("Account");
const WORKSPACE = () => t("Workspace");

let teardown: Unsubscribe[] = [];

function setup(engine: Engine): void {
  const { commands, keymap, state } = engine;
  const signedIn = () => state.get("auth") === "app" && state.get("me") !== null;
  const isAdmin = () => {
    const role = state.get("me")?.user.role;
    return role === "owner" || role === "admin";
  };

  teardown = [
    commands.register({
      id: "account.open",
      label: () => t("Account settings"),
      group: ACCOUNT,
      enabled: signedIn,
      run: () => engine.openOverlay("account"),
    }),
    commands.register({
      id: "account.languageTime",
      label: () => t("Language and time"),
      group: ACCOUNT,
      enabled: signedIn,
      run: () => engine.openOverlay("account", "language"),
    }),
    commands.register({
      id: "account.signOut",
      label: () => t("Sign out"),
      group: ACCOUNT,
      enabled: signedIn,
      run: async () => {
        await engine.api.auth.logout().catch(() => undefined);
        await engine.docs.clearLocal().catch(() => undefined);
        engine.platform.navigate("/login");
      },
    }),
    commands.register({
      id: "admin.open",
      label: () => t("Instance settings"),
      group: ACCOUNT,
      enabled: () => signedIn() && isAdmin(),
      run: () => engine.openOverlay("admin"),
    }),
    commands.register({
      id: "workspace.settings",
      label: () => t("Workspace settings"),
      group: WORKSPACE,
      enabled: () => signedIn() && engine.workspace() !== null,
      run: () => engine.openOverlay("workspace"),
    }),
    commands.register({
      id: "workspace.members",
      label: () => t("Members and invitations"),
      group: WORKSPACE,
      enabled: () => signedIn() && engine.workspace()?.personal === false,
      run: () => engine.openOverlay("workspace", "members"),
    }),
    commands.register({
      id: "workspace.new",
      label: () => t("New workspace"),
      group: WORKSPACE,
      enabled: () => signedIn() && state.get("me")?.can_create_workspaces === true,
      run: () => engine.openOverlay("new-workspace"),
    }),

    keymap.register({ id: "account.open", command: "account.open", chords: ["mod+,"], scope: "global", whileTyping: true }),
  ];
}

const account: Extension = {
  id: "account",
  setup,
  dispose() {
    for (const off of teardown) off();
    teardown = [];
  },
};

export default account;
