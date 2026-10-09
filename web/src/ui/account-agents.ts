import type { AgentGrant, AgentRun, Engine } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { confirmDialog, type Tab } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { copyText, iconButton, section, status } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { toast } from "./toast";

function copyBox(text: () => string, label: string): HTMLElement {
  const box = h("div", { class: "secret-box" }, h("span"), iconButton("copy", label));
  box.lastElementChild?.addEventListener("click", async () => toast({ id: "copy", message: (await copyText(text())) ? t("Copied.") : t("Could not copy") }));
  return box;
}

function snippets(url: string): [string, string][] {
  return [
    ["Claude Code", `claude mcp add --transport http form-md ${url}`],
    ["Cursor, VS Code and others (mcp.json)", JSON.stringify({ mcpServers: { "form-md": { url } } }, null, 2)],
    [t("Any MCP client"), url],
  ];
}

export function agentsTab(engine: Engine): Tab {
  return {
    id: "agents",
    label: t("Connected apps"),
    icon: "bot",
    render(panel) {
      const url = `${location.origin}/mcp`;
      const connect = section(t("Connect an agent"), t("Agents reach form.md through MCP. Add this address to the agent; it opens a page here to ask which workspaces it may use. A personal API token works too."));
      for (const [name, code] of snippets(url)) {
        const box = copyBox(() => code, t("Copy"));
        setText(box.firstElementChild as HTMLElement, code);
        box.classList.add("snippet");
        connect.append(h("p", { class: "field__label", text: name }), box);
      }
      const skill = h("a", { class: "btn btn--ghost", attrs: { href: "/api/mcp/skill.zip", download: "form-md-skill.zip" } }, iconNode("download"), h("span", { text: t("Download the form.md skill") }));
      connect.append(h("p", { class: "section__lead", text: t("A skill teaches Claude and other agents how to work here: the address, the tools and the conventions, such as reading workspace rules first and grouping writes into undoable runs.") }), h("div", { class: "form-actions" }, skill));

      const grantList = h("div", { class: "rows" });
      const noGrants = h("p", { class: "muted", text: t("No agent is connected.") });
      const grantStatus = status();
      const grantRow = () => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const off = iconButton("trash", t("Disconnect"));
        const r = h("div", { class: "row" }, iconNode("bot"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, h("span", { class: "badge" }), off));
        off.addEventListener("click", async () => {
          const ok = await confirmDialog({ title: t("Disconnect {name}?", { name: title.textContent ?? "" }), body: t("Its tokens stop working at once; it has to ask again to reconnect."), confirm: t("Disconnect"), variant: "danger" });
          if (!ok) return;
          try {
            await engine.api.me.revokeGrant(r.dataset.key ?? "");
            refresh();
          } catch (err) {
            grantStatus.error(err);
          }
        });
        return r;
      };
      const grantUpdate = (r: HTMLElement, g: AgentGrant) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, g.client);
        const used = g.last_used_at ? t("used {when}", { when: formatRelative(g.last_used_at) }) : t("never used");
        setText(meta, [g.workspaces.map((w) => w.name).join(", "), t("connected {date}", { date: formatDateTime(g.created_at) }), used].join(" · "));
        setText(r.querySelector<HTMLElement>(".badge")!, g.scope === "read" ? t("Read only") : t("Read and edit"));
      };

      const runList = h("div", { class: "rows" });
      const noRuns = h("p", { class: "muted", text: t("No agent has changed anything yet.") });
      const runStatus = status();
      const runRow = () => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const undo = iconButton("restore", t("Undo this run"));
        const r = h("div", { class: "row" }, iconNode("sparkle"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, h("span", { class: "badge" }), undo));
        undo.addEventListener("click", async () => {
          const ok = await confirmDialog({ title: t("Undo this run?"), body: t("Every change it made is reversed, newest first. Edits people made since stay where they do not overlap."), confirm: t("Undo"), variant: "danger" });
          if (!ok) return;
          try {
            const res = await engine.api.me.undoRun(r.dataset.key ?? "");
            toast({ id: "undo-run", tone: "success", message: t("Undid {n} changes.", { n: String(res.undone) }) });
            refresh();
          } catch (err) {
            runStatus.error(err);
            refresh();
          }
        });
        return r;
      };
      const runUpdate = (r: HTMLElement, run: AgentRun) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, run.client || t("Agent"));
        setText(meta, [run.title, t("{n} changes", { n: String(run.ops) }), formatRelative(run.updated_at)].join(" · "));
        const badge = r.querySelector<HTMLElement>(".badge")!;
        setText(badge, run.status === "undone" ? t("Undone") : "");
        badge.hidden = run.status !== "undone";
        const undo = r.querySelector<HTMLButtonElement>(".row__actions button")!;
        undo.hidden = run.status === "undone" || run.ops === 0;
        setAttr(r, "data-status", run.status);
      };

      const refresh = () => {
        engine.api.me
          .grants()
          .then(({ grants }) => {
            reconcile(grantList, grants, (g) => g.id, grantRow, grantUpdate);
            noGrants.hidden = grants.length > 0;
          })
          .catch((err) => grantStatus.error(err));
        engine.api.me
          .runs()
          .then(({ runs }) => {
            const shown = runs.filter((x) => x.ops > 0).slice(0, 30);
            reconcile(runList, shown, (x) => x.id, runRow, runUpdate);
            noRuns.hidden = shown.length > 0;
          })
          .catch((err) => runStatus.error(err));
      };

      const apps = section(t("Connected apps"), t("Agents you allowed through MCP sign-in. Each can do what you can do in the workspaces you chose."));
      apps.append(grantList, noGrants, grantStatus.el);
      const runs = section(t("Recent agent work"), t("Each run groups the changes an agent made together. Undo reverses a whole run."));
      runs.append(runList, noRuns, runStatus.el);
      panel.append(connect, apps, runs);
      refresh();
      return () => refresh();
    },
  };
}
