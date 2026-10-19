import type { Engine, Page } from "../engine";
import { t } from "../engine/i18n";
import { h, ref, setAttr } from "./dom";
import { fallbackGlyph, pageIcon } from "./pageicon";

const RECENT_SHOWN = 6;

export function mountHome(engine: Engine): void {
  const root = document.documentElement;
  const list = ref("home-recent");
  const offline = ref("home-offline");
  const templates = ref<HTMLButtonElement>("home-templates");
  const importBtn = ref<HTMLButtonElement>("home-import");
  let shown = "";
  let token = 0;

  const workspace = () => engine.state.get("workspaceId");
  const newest = (ws: string) =>
    engine.pages
      .all(ws)
      .sort((a, b) => b.updated_at - a.updated_at)
      .slice(0, RECENT_SHOWN);
  const paintMode = () => {
    const ws = workspace();
    if (!ws || !engine.pages.isLoaded(ws)) return;
    setAttr(root, "data-home", engine.pages.all(ws).length ? "back" : "new");
  };
  const paintWrite = () => {
    templates.hidden = !engine.canWrite();
    importBtn.hidden = !engine.canWrite();
  };
  const paintList = (items: Page[]) => {
    const key = items.map((p) => `${p.id}:${p.title}:${p.icon}`).join("|");
    if (key === shown) return;
    shown = key;
    list.replaceChildren(
      ...items.map((p) =>
        h("li", {}, h("button", { class: "home-recent__item", attrs: { type: "button", "data-id": p.id } }, pageIcon(p.icon, fallbackGlyph(p.kind)), h("span", { class: "home-recent__label", text: p.title || t("Untitled") }))),
      ),
    );
  };
  const loadRecent = () => {
    const ws = workspace();
    if (!ws) return;
    const mine = ++token;
    engine.api.pages.recent(RECENT_SHOWN * 2).then(
      (r) => {
        if (mine !== token) return;
        const live = r.pages.filter((p) => p.workspace_id === ws && !p.trashed_at).slice(0, RECENT_SHOWN);
        paintList(live.length ? live : newest(ws));
      },
      () => {
        if (mine === token) paintList(newest(ws));
      },
    );
  };

  list.addEventListener("click", (e) => {
    const id = (e.target as HTMLElement).closest<HTMLElement>(".home-recent__item")?.dataset.id;
    if (id) engine.openPage(id);
  });
  templates.addEventListener("click", () => void import("./templates-dialog").then((m) => m.openTemplates(engine), engine.reportError));
  importBtn.addEventListener("click", () => void import("./transfer-dialog").then((m) => m.openImport(engine, { workspaceId: workspace() }), engine.reportError));

  engine.pages.subscribe((c) => {
    if (c.kind === "loaded" || c.kind === "upsert" || c.kind === "remove") paintMode();
  });
  engine.state.watch("workspaceId", () => {
    paintMode();
    paintWrite();
    loadRecent();
  });
  engine.state.watch("me", paintWrite);
  engine.state.watch("route", loadRecent);
  engine.state.watch("health", (health) => {
    offline.hidden = health.phase !== "down";
  });
  paintMode();
  paintWrite();
  loadRecent();
}
