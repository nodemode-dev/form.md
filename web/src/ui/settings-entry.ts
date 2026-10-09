import type { Engine } from "../engine";
import { mountAccount } from "./account";
import { mountAdmin } from "./admin";
import { mountWorkspaceSettings } from "./workspace-settings";

export function mountSettings(engine: Engine): void {
  mountAccount(engine);
  mountWorkspaceSettings(engine);
  mountAdmin(engine);
}
