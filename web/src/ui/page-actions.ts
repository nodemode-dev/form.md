import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import { errorMessage } from "./errors";
import { copyText } from "./form";
import { openHistory } from "./history-dialog";
import { openPublishDialog } from "./publish-dialog";
import type { MenuEntry } from "./menu";
import { tile } from "./avatar";
import { openShareDialog } from "./share-dialog";
import { toast } from "./toast";

export interface PageMenuHandlers {
  addInside: () => void;
  trash: () => void;
}

export function pageLink(id: string): string {
  return `${location.origin}/p/${id}`;
}

export async function duplicatePage(engine: Engine, id: string): Promise<void> {
  const p = engine.pages.get(id);
  if (!p) return;
  try {
    const copy = await engine.pages.duplicate(id, t("Copy of {title}", { title: p.title || t("Untitled") }));
    engine.openPage(copy.id);
  } catch (e) {
    toast({ tone: "danger", message: errorMessage(e) });
  }
}

export async function setVisibility(engine: Engine, id: string, priv: boolean): Promise<void> {
  engine.pages.edit(id, { visibility: priv ? "private" : "workspace" }, false);
  toast({ id: `vis-${id}`, message: priv ? t("Only you can see this page now.") : t("Everyone in the workspace can see this page now.") });
}

export async function moveToWorkspace(engine: Engine, id: string, workspaceId: string): Promise<void> {
  const target = engine.workspaceById(workspaceId);
  try {
    await engine.pages.move(id, { workspace_id: workspaceId, parent_id: "" });
    toast({ message: t("Moved to {name}", { name: target?.name ?? "" }), action: { label: t("Open"), run: () => engine.openPage(id) } });
  } catch (e) {
    toast({ tone: "danger", message: errorMessage(e) });
  }
}

export function pageMenu(engine: Engine, id: string, handlers: PageMenuHandlers): MenuEntry[] {
  const p = engine.pages.get(id) ?? engine.state.get("detail");
  if (!p || p.id !== id) return [];
  const member = engine.canWrite(p.workspace_id);
  const writable = engine.can(id, "edit");
  const fav = engine.pages.isFavorite(id);
  const personal = engine.workspaceById(p.workspace_id)?.personal ?? false;
  const entries: MenuEntry[] = [
    { label: t("Open in new tab"), icon: "new-tab", run: () => engine.openPage(id, { newTab: true }) },
    { label: fav ? t("Remove from favorites") : t("Add to favorites"), icon: fav ? "star-fill" : "star", hint: engine.keymap.chordFor("page.favorite")?.join(" ") ?? "", run: () => void engine.pages.toggleFavorite(id).catch(engine.reportError) },
  ];
  if (!personal) entries.push({ label: t("Share"), icon: "users", run: () => openShareDialog(engine, id) });
  if (engine.can(id, "full")) entries.push({ label: t("Publish to the web"), icon: "globe", run: () => void openPublishDialog(engine, id) });
  if (engine.can(id, "content")) entries.push({ label: t("Add a page inside"), icon: "plus", run: handlers.addInside });
  const parent = p.parent_id && !p.detached ? p.parent_id : "";
  if (parent ? engine.can(parent, "content") : member) entries.push({ label: t("Duplicate"), icon: "copy", run: () => void duplicatePage(engine, id) });
  if (writable && member) {
    if (!personal) {
      const priv = p.visibility === "private";
      entries.push({ label: priv ? t("Share with the workspace") : t("Make private"), icon: priv ? "users" : "lock", run: () => void setVisibility(engine, id, !priv) });
    }
    if (p.parent_id) {
      entries.push({ label: t("Move to top level"), icon: "move-up", run: () => void engine.pages.move(id, { parent_id: "", teamspace_id: engine.pages.teamOf(id) }).catch(engine.reportError) });
    }
    const others = (engine.state.get("me")?.workspaces ?? []).filter((w) => w.id !== p.workspace_id && engine.canWrite(w.id));
    if (others.length) {
      entries.push({ kind: "heading", label: t("Move to workspace") });
      for (const w of others) entries.push({ label: w.name, lead: tile(w.name, w.icon, w.image), run: () => void moveToWorkspace(engine, id, w.id) });
    }
  }
  entries.push({ kind: "separator" });
  if (engine.can(id, "view") && p.kind === "doc") entries.push({ label: t("Page history"), icon: "clock", run: () => openHistory(engine, id) });
  entries.push({
    label: t("Copy link"),
    icon: "link",
    run: () => void copyText(pageLink(id)).then((ok) => toast({ tone: ok ? "success" : "danger", message: ok ? t("Link copied") : t("Could not copy") })),
  });
  entries.push({
    label: t("Copy Markdown link"),
    icon: "copy",
    run: () => void copyText(`[${p.title || t("Untitled")}](${pageLink(id)})`).then((ok) => toast({ tone: ok ? "success" : "danger", message: ok ? t("Markdown link copied") : t("Could not copy") })),
  });
  if (writable) {
    entries.push({ kind: "separator" });
    entries.push({ label: t("Move to trash"), icon: "trash", danger: true, hint: engine.keymap.chordFor("page.trash")?.join(" ") ?? "", run: handlers.trash });
  }
  return entries;
}
