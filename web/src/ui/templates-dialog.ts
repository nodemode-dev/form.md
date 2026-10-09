import type { Engine } from "../engine";
import { formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import type { Page } from "../engine/types";
import { simpleDialog } from "./dialog";
import { h, setAttr } from "./dom";
import { status } from "./form";
import { pageIcon } from "./pageicon";

interface Starter {
  id: string;
  name: string;
  description: string;
  icon: string;
  kind: "doc" | "database";
  category: string;
}

interface WorkspaceTemplate {
  id: string;
  title: string;
  icon: string;
  kind: "doc" | "collection";
  updated_at: number;
  can_edit: boolean;
}

export interface TemplateTarget {
  parentId?: string;
  teamspaceId?: string;
}

export function useTemplate(engine: Engine, body: { starter?: string; page?: string }, target: TemplateTarget = {}): Promise<Page> {
  const ws = engine.state.get("workspaceId");
  return engine.api.http
    .post<{ page: Page }>("/api/templates/use", { workspace_id: ws, parent_id: target.parentId ?? "", teamspace_id: target.teamspaceId ?? "", ...body })
    .then((r) => {
      engine.pages.upsert(r.page);
      return r.page;
    });
}

export function openTemplates(engine: Engine, target: TemplateTarget = {}): void {
  const ws = engine.state.get("workspaceId");
  simpleDialog(
    t("Templates"),
    (body, close) => {
      const st = status();
      const cats = h("div", { class: "tplgal__cats", attrs: { role: "tablist", "aria-label": t("Categories") } });
      const grid = h("div", { class: "tplgal__grid" });
      const ownHead = h("div", { class: "tplgal__head", text: t("From this workspace") });
      const own = h("div", { class: "tplgal__grid" });
      const ownEmpty = h("p", { class: "muted", text: t("Pages saved as templates show up here. Use Save as template in a page's menu.") });
      body.append(h("div", { class: "tplgal" }, cats, grid, ownHead, own, ownEmpty, st.el));
      let cat = "";
      let starters: Starter[] = [];
      const run = (card: HTMLElement, req: { starter?: string; page?: string }) => {
        if (card.getAttribute("aria-busy")) return;
        setAttr(card, "aria-busy", "true");
        useTemplate(engine, req, target)
          .then((p) => {
            close();
            engine.openPage(p.id);
          })
          .catch((e: unknown) => {
            setAttr(card, "aria-busy", null);
            st.error(e);
          });
      };
      const card = (icon: string, fallback: string, name: string, desc: string) =>
        h("button", { class: "tplcard", attrs: { type: "button" } }, h("span", { class: "tplcard__icon" }, pageIcon(icon, fallback)), h("span", { class: "tplcard__name", text: name }), h("span", { class: "tplcard__desc", text: desc }));
      const paintStarters = () => {
        for (const b of Array.from(cats.children) as HTMLElement[]) setAttr(b, "aria-pressed", (b.dataset.cat ?? "") === cat ? "true" : "false");
        if (!grid.firstElementChild) {
          for (const s of starters) {
            const c = card(s.icon, s.kind === "database" ? "database" : "page", s.name, s.description);
            c.dataset.starter = s.id;
            c.dataset.cat = s.category;
            c.addEventListener("click", () => run(c, { starter: s.id }));
            grid.append(c);
          }
        }
        for (const c of Array.from(grid.children) as HTMLElement[]) c.hidden = !!cat && c.dataset.cat !== cat;
      };
      engine.api.http
        .get<{ starters: Starter[]; workspace: WorkspaceTemplate[] }>(`/api/templates?workspace=${encodeURIComponent(ws)}`)
        .then((g) => {
          starters = g.starters;
          const names = [...new Set(g.starters.map((s) => s.category))];
          cats.replaceChildren(
            ...["", ...names].map((c) => {
              const b = h("button", { class: "chip chip--btn", attrs: { type: "button", "data-cat": c, "aria-pressed": "false" }, text: c ? t(c) : t("All") });
              b.addEventListener("click", () => {
                cat = c;
                paintStarters();
              });
              return b;
            }),
          );
          paintStarters();
          own.replaceChildren(
            ...g.workspace.map((w) => {
              const c = card(w.icon, w.kind === "collection" ? "database" : "page", w.title || t("Untitled"), t("Updated {when}", { when: formatRelative(w.updated_at) }));
              c.dataset.page = w.id;
              c.addEventListener("click", () => run(c, { page: w.id }));
              return c;
            }),
          );
          ownEmpty.hidden = g.workspace.length > 0;
        })
        .catch((e: unknown) => st.error(e));
    },
    undefined,
    "md",
  );
}
