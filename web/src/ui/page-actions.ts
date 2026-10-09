import type { Engine, Page } from "../engine";
import { t } from "../engine/i18n";
import { errorMessage } from "./errors";
import { copyText } from "./form";
import { openAnalytics } from "./analytics-dialog";
import type { MenuEntry } from "./menu";
import { tile } from "./avatar";
import { offlineIds } from "./offline-list";
import { toast } from "./toast";

export function openShareDialog(engine: Engine, id: string): void {
  void import("./share-dialog").then((m) => m.openShareDialog(engine, id), engine.reportError);
}

function openPublishDialog(engine: Engine, id: string): Promise<void> {
  return import("./publish-dialog").then((m) => m.openPublishDialog(engine, id), engine.reportError);
}

function openHistory(engine: Engine, id: string): void {
  void import("./history-dialog").then((m) => m.openHistory(engine, id), engine.reportError);
}

export async function setTemplate(engine: Engine, id: string, on: boolean): Promise<void> {
  try {
    const r = await engine.api.http.patch<{ page: Page }>(`/api/pages/${encodeURIComponent(id)}`, { is_template: on });
    if (r?.page) engine.pages.upsert(r.page);
    toast({ id: "template", tone: "success", message: on ? t("Saved as a template. Find it under Templates in the sidebar.") : t("This page is a normal page again.") });
  } catch (e) {
    engine.reportError(e);
  }
}

export function prefetchPageDialogs(): void {
  void import("./share-dialog").catch(() => undefined);
  void import("./publish-dialog").catch(() => undefined);
  void import("./history-dialog").catch(() => undefined);
}

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
  if (member && engine.can(id, "edit") && !p.row) entries.push({ label: t("Save as template"), icon: "gallery", run: () => void setTemplate(engine, id, true) });
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
  if (engine.can(id, "edit")) entries.push({ label: t("Analytics"), icon: "chart", run: () => void openAnalytics(engine, id) });
  if (engine.can(id, "view")) entries.push({ label: t("Export"), icon: "download", run: () => void import("./transfer-dialog").then((m) => m.openExport(engine, { pageId: id }), engine.reportError) });
  if (engine.can(id, "view") && p.kind === "doc" && "serviceWorker" in navigator) {
    entries.push({ label: t("Available offline"), icon: "offline", checked: offlineIds(engine).includes(id), run: () => void import("./offline").then((m) => m.toggleOffline(engine, id), engine.reportError) });
  }
  if (engine.can(id, "content") && p.kind === "doc" && !p.row)
    entries.push({ label: t("Import into this page"), icon: "upload", run: () => void import("./transfer-dialog").then((m) => m.openImport(engine, { workspaceId: p.workspace_id, parentId: id }), engine.reportError) });
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
