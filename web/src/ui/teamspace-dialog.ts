import type { Engine, Member, TeamAccess, TeamMember, Teamspace } from "../engine";
import { t } from "../engine/i18n";
import { avatar } from "./avatar";
import { confirmDialog, simpleDialog } from "./dialog";
import { h, setText } from "./dom";
import { actions, busy, button, field, iconButton, input, optionCards, section, select, status, textarea } from "./form";
import { reconcile } from "./list";
import { toast } from "./toast";

const accessOptions = (): readonly (readonly [TeamAccess, string, string])[] => [
  ["open", t("Open"), t("Every workspace member can see and join it.")],
  ["closed", t("Closed"), t("Listed for everyone; only members see its pages.")],
  ["private", t("Private"), t("Only its members know it exists.")],
];

export function openTeamspaceDialog(engine: Engine, workspaceId: string, team: Teamspace | null): void {
  const manage = !team || team.can_manage;
  simpleDialog(
    team ? team.name : t("New teamspace"),
    (body, close) => {
      const name = input({ value: team?.name ?? "", placeholder: t("Design, Support, Q3 launch"), maxlength: 100 });
      const about = textarea(team?.description ?? "", 2);
      const access = optionCards("team-access", accessOptions(), team?.access ?? "open");
      const st = status();
      const save = button(team ? t("Save") : t("Create teamspace"), "primary", undefined, "submit");
      const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, field(t("Name"), name), field(t("Description"), about), field(t("Access"), access.el), st.el, actions(save));
      if (!manage) {
        name.disabled = true;
        about.disabled = true;
        access.disable(true);
        save.hidden = true;
      }
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st.clear();
        const changes = { name: name.value.trim(), description: about.value.trim(), access: access.value() };
        try {
          if (team) {
            await busy(save, () => engine.api.teamspaces.update(team.id, changes));
            st.say(t("Saved."), "success");
          } else {
            const r = await busy(save, () => engine.api.workspaces.createTeamspace(workspaceId, changes));
            await engine.pages.loadTeams(workspaceId);
            close();
            toast({ message: t("{name} is ready. Add pages with the plus beside it.", { name: r.teamspace.name }), tone: "success" });
          }
        } catch (err) {
          st.error(err);
        }
      });
      body.append(form);
      if (!team) return name;

      const people = section(t("Members"), t("Owners manage the teamspace and its members."));
      const list = h("div", { class: "rows" });
      const listStatus = status();
      const myId = engine.state.get("me")?.user.id ?? "";
      const memberRow = (m: TeamMember) => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const side = h("div", { class: "row__actions" });
        const r = h("div", { class: "row" }, avatar(m.name, ""), h("div", { class: "row__main" }, title, meta), side);
        if (manage) {
          const role = select(
            [
              ["owner", t("Owner")],
              ["member", t("Member")],
            ] as const,
            m.role === "owner" ? "owner" : "member",
          );
          role.setAttribute("aria-label", t("Role of {name}", { name: m.name }));
          role.addEventListener("change", async () => {
            listStatus.clear();
            try {
              await engine.api.teamspaces.setMember(team.id, m.user_id, role.value as "owner" | "member");
            } catch (err) {
              role.value = r.dataset.role ?? "member";
              listStatus.error(err);
            }
          });
          const remove = iconButton("trash", t("Remove"));
          remove.addEventListener("click", async () => {
            listStatus.clear();
            try {
              await engine.api.teamspaces.removeMember(team.id, m.user_id);
              refresh();
            } catch (err) {
              listStatus.error(err);
            }
          });
          side.append(role, remove);
        } else {
          side.append(h("span", { class: "badge" }));
        }
        return r;
      };
      const updateRow = (r: HTMLElement, m: TeamMember) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, m.user_id === myId ? t("{name} (you)", { name: m.name }) : m.name);
        setText(meta, m.email);
        r.dataset.role = m.role;
        const sel = r.querySelector<HTMLSelectElement>("select");
        if (sel && sel.value !== m.role) sel.value = m.role;
        const badge = r.querySelector<HTMLElement>(".badge");
        if (badge) setText(badge, m.role === "owner" ? t("Owner") : t("Member"));
      };
      const pick = select<string>([["", t("Choose a workspace member")]], "");
      pick.setAttribute("aria-label", t("Add a member"));
      const add = button(t("Add"), "default", "plus");
      const adder = h("div", { class: "form-row form-row--inline" }, pick, add);
      adder.hidden = !manage;
      let everyone: Member[] = [];
      let current: TeamMember[] = [];
      const fillPick = () => {
        const inTeam = new Set(current.map((m) => m.user_id));
        const options = everyone.filter((m) => !inTeam.has(m.user_id) && !m.disabled);
        const keep = new Set(options.map((m) => m.user_id));
        for (const o of Array.from(pick.options)) if (o.value && !keep.has(o.value)) o.remove();
        const have = new Set(Array.from(pick.options).map((o) => o.value));
        for (const m of options) if (!have.has(m.user_id)) pick.append(h("option", { attrs: { value: m.user_id }, text: `${m.name} (${m.email})` }));
        adder.hidden = !manage || options.length === 0;
      };
      const refresh = () => {
        engine.api.teamspaces
          .members(team.id)
          .then(({ members }) => {
            current = members;
            reconcile(list, members, (m) => m.user_id, memberRow, updateRow);
            fillPick();
          })
          .catch((err) => listStatus.error(err));
      };
      add.addEventListener("click", async () => {
        if (!pick.value) return;
        listStatus.clear();
        try {
          await busy(add, () => engine.api.teamspaces.setMember(team.id, pick.value, "member"));
          pick.value = "";
          refresh();
        } catch (err) {
          listStatus.error(err);
        }
      });
      if (manage) {
        engine.api.workspaces
          .members(workspaceId)
          .then(({ members }) => {
            everyone = members;
            fillPick();
          })
          .catch((err) => listStatus.error(err));
      }
      people.append(list, adder, listStatus.el);
      body.append(people);
      refresh();

      if (manage) {
        const zone = section(t("Delete teamspace"), t("Move or delete its pages first; an empty teamspace can be deleted."));
        const del = button(t("Delete teamspace"), "danger", "trash");
        const zoneStatus = status();
        del.addEventListener("click", async () => {
          const ok = await confirmDialog({ title: t("Delete {name}?", { name: team.name }), body: t("Its members keep their workspace access."), confirm: t("Delete"), variant: "danger" });
          if (!ok) return;
          try {
            await engine.api.teamspaces.remove(team.id);
            await engine.pages.loadTeams(workspaceId);
            close();
          } catch (err) {
            zoneStatus.error(err);
          }
        });
        zone.append(actions(del), zoneStatus.el);
        body.append(zone);
      }
      return manage ? name : undefined;
    },
    undefined,
    "md",
  );
}
