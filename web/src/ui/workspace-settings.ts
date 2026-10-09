import type { AgentAccess, EmergencyGrant, Engine, Group, Invite, Member, TeamMember, Workspace, WorkspaceRole } from "../engine";
import { formatDate, formatDateTime, formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { avatar, tile, updateAvatar, updateTile } from "./avatar";
import { openIconPicker, uploadPicture } from "./pickers";
import { confirmDialog, confirmTyped, overlayDialog, simpleDialog, type Tab } from "./dialog";
import { h, setText } from "./dom";
import { actions, busy, button, copyText, field, iconButton, input, optionCards, section, select, status, textarea, toggle } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { toast } from "./toast";
import { sitesTab } from "./workspace-sites";

const MAX_RULES = 16000;

export const roleOptions = (): readonly (readonly [WorkspaceRole, string])[] => [
  ["admin", t("Admin")],
  ["member", t("Member (can edit)")],
  ["viewer", t("Viewer (read only)")],
];

const roleName = (r: string): string => (r === "admin" ? t("Admin") : r === "viewer" ? t("Viewer") : t("Member"));

export function isAdmin(w: Workspace): boolean {
  return w.role === "admin" && !w.break_glass;
}

export function boundTab(engine: Engine, id: string, label: string, icon: Tab["icon"], build: (panel: HTMLElement, w: Workspace) => (() => void) | void, visible?: () => boolean): Tab {
  const tab: Tab = {
    id,
    label,
    icon,
    render(panel) {
      let builtFor = "";
      let refresh: (() => void) | void;
      const ensure = () => {
        const w = engine.workspace();
        if (!w) return;
        if (w.id !== builtFor) {
          builtFor = w.id;
          panel.replaceChildren();
          refresh = build(panel, w);
        } else {
          refresh?.();
        }
      };
      ensure();
      return ensure;
    },
  };
  if (visible) tab.visible = visible;
  return tab;
}

async function afterMembershipLoss(engine: Engine, message: string): Promise<void> {
  await engine.refreshMe();
  engine.state.set("overlay", null);
  toast({ message, tone: "success" });
}

function generalTab(engine: Engine): Tab {
  return boundTab(engine, "general", t("General"), "settings", (panel, w) => {
    const admin = isAdmin(w);
    const name = input({ name: "workspace-name", value: w.name, maxlength: 80, autocomplete: "off" });
    name.disabled = !admin;
    const st = status();
    const save = button(t("Rename"), "primary", undefined, "submit");
    const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, field(t("Name"), name, t("Shown in the switcher and on cards.")), st.el);
    if (admin) form.append(actions(save));

    const current = () => engine.workspaceById(w.id) ?? w;
    const preview = tile(w.name, w.icon, w.image);
    const pickIcon = button(t("Choose icon"), "ghost", "shape");
    const pickImage = button(t("Upload image"), "ghost", "upload");
    const clearPic = button(t("Remove"), "ghost", "trash");
    const picStatus = status();
    const paintPic = () => {
      const cur = current();
      updateTile(preview, cur.name, cur.icon, cur.image);
      clearPic.hidden = !admin || (!cur.icon && !cur.image);
    };
    const savePic = async (body: { icon?: string; image?: string }) => {
      picStatus.clear();
      try {
        await engine.api.workspaces.update(w.id, body);
        await engine.refreshMe();
        paintPic();
      } catch (err) {
        picStatus.error(err);
      }
    };
    pickIcon.addEventListener("click", () =>
      openIconPicker(engine, pickIcon, {
        current: current().icon,
        upload: { purpose: "workspace", workspace_id: w.id },
        allowRemove: !!current().icon,
        onPick: (ref) => void savePic(ref.startsWith("f:") ? { image: ref, icon: "" } : { icon: ref, image: "" }),
        onRemove: () => void savePic({ icon: "" }),
      }),
    );
    pickImage.addEventListener("click", () =>
      void uploadPicture(engine, { purpose: "workspace", workspace_id: w.id }, "image/png,image/jpeg,image/gif,image/webp,image/svg+xml").then((ref) => {
        if (ref) void savePic({ image: ref, icon: "" });
      }),
    );
    clearPic.addEventListener("click", () => void savePic({ icon: "", image: "" }));
    pickIcon.hidden = !admin;
    pickImage.hidden = !admin;
    paintPic();
    const pic = h("div", { class: "ws-pic" }, preview, h("div", { class: "form-grid" }, h("div", { class: "ws-pic__actions" }, pickIcon, pickImage, clearPic), picStatus.el));
    form.prepend(pic);
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      st.clear();
      try {
        await busy(save, () => engine.api.workspaces.update(w.id, { name: name.value }));
        await engine.refreshMe();
        st.say(t("Workspace renamed."), "success");
      } catch (err) {
        st.error(err);
      }
    });

    const facts = h("div", { class: "rows" });
    const fact = (icon: Parameters<typeof iconNode>[0], title: string, meta: string) =>
      facts.append(h("div", { class: "row" }, iconNode(icon), h("div", { class: "row__main" }, h("span", { class: "row__title", text: title }), h("span", { class: "row__meta", text: meta }))));
    if (w.personal) fact("lock", t("Personal space"), t("Private to you. Nobody else can be added, not even an admin, and emergency access never reaches it."));
    else fact("users", plural(w.member_count, "{count} member", "{count} members"), t("Your role: {role}", { role: roleName(w.role) }));
    if (w.break_glass) fact("alert", t("Emergency access"), t("You are reading this workspace as the instance owner. Its admins have been told."));
    if (!admin && !w.personal) fact("info", t("Read only here"), t("Only an admin of this workspace can change its settings."));

    const top = section(t("Workspace"));
    top.append(form, facts);
    panel.append(top);

    if (admin) {
      const mode = optionCards<"split" | "tree">(
        "sidebar-mode",
        [
          ["split", t("Two sections"), t("Documents and collections are listed apart, each in its own section.")],
          ["tree", t("One tree"), t("Everything sits in one tree, collections among documents.")],
        ],
        w.sidebar_mode,
      );
      const modeStatus = status();
      mode.el.addEventListener("change", async () => {
        modeStatus.clear();
        const before = current().sidebar_mode;
        try {
          await engine.api.workspaces.update(w.id, { sidebar_mode: mode.value() });
          await engine.refreshMe();
        } catch (err) {
          mode.set(before);
          modeStatus.error(err);
        }
      });
      const s = section(t("Sidebar"), t("How this workspace's pages appear in the sidebar, for everyone in it."));
      s.append(mode.el, modeStatus.el);
      panel.append(s);
    }

    if (engine.state.get("me")?.user.role === "owner" && !w.personal) {
      const auto = toggle(t("Open to every new account"), w.auto_join, t("Every account created from now on joins this workspace as a member."));
      const autoStatus = status();
      auto.input.addEventListener("change", async () => {
        autoStatus.clear();
        try {
          await engine.api.workspaces.update(w.id, { auto_join: auto.input.checked });
          await engine.refreshMe();
        } catch (err) {
          auto.input.checked = !auto.input.checked;
          autoStatus.error(err);
        }
      });
      const s = section(t("New accounts"));
      s.append(auto.el, autoStatus.el);
      panel.append(s);
    }

    if (w.personal || w.break_glass) return;
    const zone = section(t("Leave or delete"));
    const zoneStatus = status();
    const leave = button(t("Leave workspace"), "default", "logout");
    leave.addEventListener("click", async () => {
      const ok = await confirmDialog({
        title: t("Leave {name}?", { name: w.name }),
        body: t("You lose access to everything in it until an admin adds you again."),
        confirm: t("Leave"),
        variant: "danger",
      });
      if (!ok) return;
      try {
        await engine.api.workspaces.removeMember(w.id, engine.state.get("me")?.user.id ?? "");
        await afterMembershipLoss(engine, t("You left {name}.", { name: w.name }));
      } catch (err) {
        zoneStatus.error(err);
      }
    });
    const row = h("div", { class: "form-actions form-actions--start" }, leave);
    if (admin) {
      const del = button(t("Delete workspace"), "danger", "trash");
      del.addEventListener("click", async () => {
        const ok = await confirmTyped({
          title: t("Delete {name}?", { name: w.name }),
          body: t("Every page, collection and file in it is deleted for every member. This cannot be undone."),
          expect: w.name,
          confirm: t("Delete workspace"),
        });
        if (!ok) return;
        try {
          await engine.api.workspaces.remove(w.id);
          await afterMembershipLoss(engine, t("{name} was deleted.", { name: w.name }));
        } catch (err) {
          zoneStatus.error(err);
        }
      });
      row.append(del);
    }
    zone.append(row, zoneStatus.el);
    panel.append(zone);
  });
}

function membersTab(engine: Engine): Tab {
  return boundTab(
    engine,
    "members",
    t("Members"),
    "users",
    (panel, w) => {
      const admin = isAdmin(w);
      const myId = engine.state.get("me")?.user.id ?? "";
      const listStatus = status();

      const list = h("div", { class: "rows" });
      const memberRow = (m: Member) => {
        const face = avatar(m.name, m.color);
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const side = h("div", { class: "row__actions" });
        const r = h("div", { class: "row" }, face, h("div", { class: "row__main" }, title, meta), side);
        if (admin) {
          const role = select(roleOptions(), m.role);
          role.setAttribute("aria-label", t("Role of {name}", { name: m.name }));
          role.addEventListener("change", async () => {
            listStatus.clear();
            const prev = r.dataset.role as WorkspaceRole;
            try {
              await engine.api.workspaces.setRole(w.id, m.user_id, role.value as WorkspaceRole);
              r.dataset.role = role.value;
              if (m.user_id === myId) await engine.refreshMe();
            } catch (err) {
              role.value = prev;
              listStatus.error(err);
            }
          });
          side.append(role);
          if (m.user_id !== myId) {
            const remove = iconButton("trash", t("Remove"));
            remove.addEventListener("click", async () => {
              const name = title.textContent ?? "";
              const ok = await confirmDialog({ title: t("Remove {name}?", { name }), body: t("They lose access to this workspace straight away."), confirm: t("Remove"), variant: "danger" });
              if (!ok) return;
              try {
                await engine.api.workspaces.removeMember(w.id, m.user_id);
                refreshMembers();
              } catch (err) {
                listStatus.error(err);
              }
            });
            side.append(remove);
          }
        } else {
          side.append(h("span", { class: "badge" }));
        }
        return r;
      };
      const updateMember = (r: HTMLElement, m: Member) => {
        updateAvatar(r.querySelector<HTMLElement>(".avatar")!, m.name, m.color);
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, m.user_id === myId ? t("{name} (you)", { name: m.name }) : m.name);
        setText(meta, [m.email, m.disabled ? t("deactivated") : t("joined {date}", { date: formatDate(m.joined_at) })].filter(Boolean).join(" · "));
        r.dataset.role = m.role;
        const sel = r.querySelector<HTMLSelectElement>("select");
        if (sel && sel.value !== m.role) sel.value = m.role;
        const badge = r.querySelector<HTMLElement>(".row__actions .badge");
        if (badge) setText(badge, roleName(m.role));
      };
      const refreshMembers = () => {
        engine.api.workspaces
          .members(w.id)
          .then(({ members }) => reconcile(list, members, (m) => m.user_id, memberRow, updateMember))
          .catch((err) => listStatus.error(err));
      };

      const people = section(t("Members"), t("Who is in, and in which role. Viewers read; members edit; admins also manage members, rules and agents."));
      people.append(list, listStatus.el);
      panel.append(people);
      refreshMembers();
      if (!admin) return refreshMembers;

      const email = input({ type: "email", name: "invite-email", placeholder: t("name@example.com"), autocomplete: "off" });
      const role = select(roleOptions(), "member");
      const inviteStatus = status();
      const send = button(t("Invite"), "primary", "mail", "submit");
      const reveal = h("div", { class: "secret-box" });
      reveal.hidden = true;
      const form = h(
        "form",
        { class: "form-grid", attrs: { novalidate: "" } },
        h("div", { class: "form-row" }, field(t("Email"), email, t("Leave empty for a link only.")), field(t("Role"), role)),
        reveal,
        inviteStatus.el,
        actions(send),
      );
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        inviteStatus.clear();
        try {
          const res = await busy(send, () => engine.api.workspaces.invite(w.id, email.value.trim(), role.value as WorkspaceRole));
          const copied = await copyText(res.url);
          const copy = iconButton("copy", t("Copy link"));
          copy.addEventListener("click", async () => toast({ message: (await copyText(res.url)) ? t("Invitation link copied.") : t("Copying failed."), id: "copy" }));
          reveal.replaceChildren(h("span", { text: res.url }), copy);
          reveal.hidden = false;
          inviteStatus.say(
            res.emailed ? t("Invitation sent by email. The link is valid for 14 days.") : copied ? t("Invitation link copied. It is valid for 14 days and works once.") : t("Share this link. It is valid for 14 days and works once."),
            "success",
          );
          email.value = "";
          refreshInvites();
        } catch (err) {
          inviteStatus.error(err);
        }
      });

      const pending = h("div", { class: "rows" });
      const pendingStatus = status();
      const inviteRow = () => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const cancel = iconButton("trash", t("Cancel invitation"));
        const r = h("div", { class: "row" }, iconNode("mail"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, cancel));
        cancel.addEventListener("click", async () => {
          try {
            await engine.api.workspaces.cancelInvite(w.id, r.dataset.key ?? "");
            refreshInvites();
          } catch (err) {
            pendingStatus.error(err);
          }
        });
        return r;
      };
      const updateInvite = (r: HTMLElement, inv: Invite) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, inv.email || t("Link only"));
        setText(meta, [roleName(inv.role), t("expires {date}", { date: formatDate(inv.expires_at) })].join(" · "));
      };
      const pendingSection = section(t("Open invitations"));
      const refreshInvites = () => {
        engine.api.workspaces
          .invites(w.id)
          .then(({ invites }) => {
            reconcile(pending, invites, (i) => i.id, inviteRow, updateInvite);
            pendingSection.hidden = invites.length === 0;
          })
          .catch((err) => pendingStatus.error(err));
      };
      pendingSection.append(pending, pendingStatus.el);

      const invite = section(t("Invite people"), t("New people get a link to create their account; people with an account join with it after confirming who they are."));
      invite.append(form);
      panel.append(invite, pendingSection, emergencySection(engine, w));
      refreshInvites();
      return () => {
        refreshMembers();
        refreshInvites();
      };
    },
    () => engine.workspace()?.personal === false,
  );
}

function groupsTab(engine: Engine): Tab {
  return boundTab(
    engine,
    "groups",
    t("Groups"),
    "users",
    (panel, w) => {
      const admin = isAdmin(w);
      const st = status();
      const list = h("div", { class: "rows" });
      let members: Member[] = [];
      const groupRow = (g: Group) => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const side = h("div", { class: "row__actions" });
        const r = h("div", { class: "row" }, iconNode("users"), h("div", { class: "row__main" }, title, meta), side);
        const open = button(admin ? t("Manage") : t("Members"), "ghost");
        open.addEventListener("click", () => manageGroup(g));
        side.append(open);
        if (admin) {
          const del = iconButton("trash", t("Delete group"));
          del.addEventListener("click", async () => {
            const ok = await confirmDialog({ title: t("Delete {name}?", { name: g.name }), body: t("Pages shared with this group stop being shared with its members."), confirm: t("Delete"), variant: "danger" });
            if (!ok) return;
            try {
              await engine.api.groups.remove(g.id);
              refresh();
            } catch (err) {
              st.error(err);
            }
          });
          side.append(del);
        }
        return r;
      };
      const updateGroup = (r: HTMLElement, g: Group) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, g.name);
        setText(meta, [plural(g.members, "{count} member", "{count} members"), g.external_id ? t("from your directory") : ""].filter(Boolean).join(" · "));
      };
      const refresh = () => {
        engine.api.workspaces
          .groups(w.id)
          .then(({ groups }) => reconcile(list, groups, (g) => g.id, groupRow, updateGroup))
          .catch((err) => st.error(err));
      };
      const manageGroup = (g: Group) => {
        simpleDialog(
          g.name,
          (body) => {
            const rows = h("div", { class: "rows" });
            const gst = status();
            const memberRow = (m: TeamMember) => {
              const r = h("div", { class: "row" }, avatar(m.name, ""), h("div", { class: "row__main" }, h("span", { class: "row__title", text: m.name }), h("span", { class: "row__meta", text: m.email })), h("div", { class: "row__actions" }));
              if (admin) {
                const remove = iconButton("trash", t("Remove"));
                remove.addEventListener("click", async () => {
                  try {
                    await engine.api.groups.removeMember(g.id, m.user_id);
                    load();
                  } catch (err) {
                    gst.error(err);
                  }
                });
                r.lastElementChild?.append(remove);
              }
              return r;
            };
            const pick = select<string>([["", t("Choose a workspace member")]], "");
            pick.setAttribute("aria-label", t("Add a member"));
            const add = button(t("Add"), "default", "plus");
            const adder = h("div", { class: "form-row form-row--inline" }, pick, add);
            adder.hidden = !admin;
            const load = () => {
              engine.api.groups
                .members(g.id)
                .then(({ members: inGroup }) => {
                  reconcile(rows, inGroup, (m) => m.user_id, memberRow, () => undefined);
                  const have = new Set(inGroup.map((m) => m.user_id));
                  pick.replaceChildren(h("option", { attrs: { value: "" }, text: t("Choose a workspace member") }));
                  for (const m of members) if (!have.has(m.user_id) && !m.disabled) pick.append(h("option", { attrs: { value: m.user_id }, text: `${m.name} (${m.email})` }));
                  refresh();
                })
                .catch((err) => gst.error(err));
            };
            add.addEventListener("click", async () => {
              if (!pick.value) return;
              try {
                await busy(add, () => engine.api.groups.addMember(g.id, pick.value));
                load();
              } catch (err) {
                gst.error(err);
              }
            });
            body.append(rows, adder, gst.el);
            load();
            return admin ? pick : undefined;
          },
          undefined,
          "md",
        );
      };
      const groups = section(t("Groups"), t("Share pages with a group at once. Changing a group changes who can open those pages."));
      groups.append(list, st.el);
      panel.append(groups);
      if (admin) {
        const name = input({ placeholder: t("Design team"), maxlength: 100 });
        name.setAttribute("aria-label", t("Group name"));
        const create = button(t("Create group"), "primary", "plus", "submit");
        const cst = status();
        const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, h("div", { class: "form-row form-row--inline" }, name, create), cst.el);
        form.addEventListener("submit", async (e) => {
          e.preventDefault();
          cst.clear();
          try {
            await busy(create, () => engine.api.workspaces.createGroup(w.id, name.value.trim()));
            name.value = "";
            refresh();
          } catch (err) {
            cst.error(err);
          }
        });
        const fresh = section(t("New group"));
        fresh.append(form);
        panel.append(fresh);
        engine.api.workspaces
          .members(w.id)
          .then((r) => {
            members = r.members;
          })
          .catch((err) => st.error(err));
      }
      refresh();
      return refresh;
    },
    () => engine.workspace()?.personal === false && engine.workspace()?.role !== "guest",
  );
}

function emergencySection(engine: Engine, w: Workspace): HTMLElement {
  const el = section(t("Emergency access log"), t("Who looked in as the instance owner, and why. Every grant is read only and ends after two hours."));
  const list = h("div", { class: "rows" });
  const empty = h("p", { class: "muted", text: t("There has been no emergency access to this workspace.") });
  const st = status();
  el.append(list, empty, st.el);
  const row = () => {
    const title = h("span", { class: "row__title" });
    const meta = h("span", { class: "row__meta" });
    const end = button(t("End it now"), "default");
    const r = h("div", { class: "row" }, iconNode("alert"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, end));
    end.addEventListener("click", async () => {
      try {
        await busy(end, () => engine.api.workspaces.endEmergency(w.id, r.dataset.key ?? ""));
        refresh();
      } catch (err) {
        st.error(err);
      }
    });
    return r;
  };
  const update = (r: HTMLElement, g: EmergencyGrant) => {
    const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
    const now = Date.now();
    const running = g.revoked_at === 0 && g.expires_at > now;
    setText(title, g.reason);
    const state = running ? t("runs until {time}", { time: formatDateTime(g.expires_at) }) : g.revoked_at ? t("ended early") : t("expired");
    setText(meta, [g.user_name, formatDateTime(g.created_at), state].join(" · "));
    r.querySelector<HTMLElement>(".row__actions .btn")!.hidden = !running;
  };
  const refresh = () => {
    engine.api.workspaces
      .emergency(w.id)
      .then(({ grants }) => {
        reconcile(list, grants, (g) => g.id, row, update);
        empty.hidden = grants.length > 0;
      })
      .catch((err) => st.error(err));
  };
  refresh();
  return el;
}

function agentsTab(engine: Engine): Tab {
  return boundTab(engine, "agents", t("Agents and rules"), "bot", (panel, w) => {
    const admin = isAdmin(w);
    const access = optionCards<AgentAccess>(
      "agent-access",
      [
        ["open", t("Anything they were granted"), t("Any token or connected app that was given this workspace.")],
        ["strict", t("Only signed-in connections"), t("Apps connected through sign-in only. Permanent tokens are refused, even ones naming this workspace. For confidential material.")],
        ["closed", t("No agents at all"), t("Browser sessions only.")],
      ],
      w.agent_access,
    );
    access.disable(!admin);
    const accessStatus = status();
    access.el.addEventListener("change", async () => {
      accessStatus.clear();
      const next = access.value();
      try {
        await engine.api.workspaces.update(w.id, { agent_access: next });
        await engine.refreshMe();
        accessStatus.say(t("Saved."), "success");
      } catch (err) {
        access.set(engine.workspace()?.agent_access ?? w.agent_access);
        accessStatus.error(err);
      }
    });
    const s1 = section(t("What agents may do here"), admin ? "" : t("Only an admin of this workspace can change this."));
    s1.append(access.el, accessStatus.el);

    const s2 = section(t("Workspace rules"), t("Conventions everyone, agents above all, follows in this workspace. Agents read them before they write and may only propose changes."));
    const proposal = h("div", { class: "proposal" });
    proposal.hidden = true;
    const box = textarea("", 10);
    box.placeholder = t("For example: Invoices go into Finance/Inbox. Titles start with the date. Never edit the Handbook section.");
    box.maxLength = MAX_RULES;
    box.readOnly = !admin;
    const count = h("span", { class: "field__hint" });
    const paintCount = () => setText(count, t("{n} of {max} characters", { n: box.value.length.toLocaleString(), max: MAX_RULES.toLocaleString() }));
    box.addEventListener("input", paintCount);
    const updated = h("p", { class: "muted" });
    const rulesStatus = status();
    const save = button(t("Save rules"), "primary", undefined, "submit");
    const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, h("label", { class: "field" }, box, count), updated, rulesStatus.el);
    if (admin) form.append(actions(save));
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      rulesStatus.clear();
      try {
        await busy(save, () => engine.api.workspaces.setRules(w.id, box.value));
        rulesStatus.say(t("Workspace rules saved."), "success");
        load();
      } catch (err) {
        rulesStatus.error(err);
      }
    });
    s2.append(proposal, form);
    panel.append(s1, s2);

    let dirty = false;
    box.addEventListener("input", () => {
      dirty = true;
    });

    let proposed = "";
    const proposalBy = h("strong");
    const proposalText = h("pre", { class: "proposal__text" });
    const useProposal = button(t("Load into editor"), "default");
    const dismissProposal = button(t("Dismiss proposal"), "ghost");
    useProposal.addEventListener("click", () => {
      box.value = proposed;
      dirty = true;
      paintCount();
      box.focus();
    });
    dismissProposal.addEventListener("click", async () => {
      try {
        await engine.api.workspaces.dismissProposal(w.id);
        toast({ message: t("Proposal dismissed.") });
        load();
        void engine.refreshMe();
      } catch (err) {
        rulesStatus.error(err);
      }
    });
    proposal.append(h("div", { class: "callout" }, iconNode("bot"), h("div", { class: "form-grid" }, proposalBy, proposalText, actions(dismissProposal, useProposal))));

    const load = () => {
      engine.api.workspaces
        .rules(w.id)
        .then((r) => {
          if (!dirty || box.value === r.rules) {
            box.value = r.rules;
            dirty = false;
          }
          paintCount();
          setText(updated, r.updated_at ? t("Last changed {when}.", { when: formatRelative(r.updated_at) }) : admin ? "" : t("No rules yet. A workspace admin can add them."));
          proposed = r.proposal && admin ? r.proposal : "";
          setText(proposalBy, t("Proposed by {name}, {when}", { name: r.proposal_by_name || t("an agent"), when: formatRelative(r.proposal_at ?? 0) }));
          setText(proposalText, proposed);
          proposal.hidden = proposed === "";
        })
        .catch((err) => rulesStatus.error(err));
    };
    load();
    return () => {
      access.set(engine.workspace()?.agent_access ?? w.agent_access);
      load();
    };
  });
}

export function mountWorkspaceSettings(engine: Engine): void {
  const dlg = overlayDialog(engine, "workspace", t("Workspace settings"), [generalTab(engine), membersTab(engine), groupsTab(engine), sitesTab(engine), agentsTab(engine)]);
  engine.state.watch("overlay", (v) => {
    if (v === "workspace") dlg.title(t("{name} settings", { name: engine.workspace()?.name ?? "" }));
  });
  engine.state.watch("me", () => {
    if (dlg.isOpen()) dlg.title(t("{name} settings", { name: engine.workspace()?.name ?? "" }));
  });

  engine.state.watch("overlay", (v) => {
    if (v !== "new-workspace") return;
    simpleDialog(
      t("New workspace"),
      (body, close) => {
        const name = input({ name: "new-workspace", maxlength: 80, autocomplete: "off", placeholder: t("for example Marketing") });
        const st = status();
        const create = button(t("Create workspace"), "primary", "plus", "submit");
        const cancel = button(t("Cancel"), "ghost");
        cancel.addEventListener("click", close);
        const form = h(
          "form",
          { class: "form-grid", attrs: { novalidate: "" } },
          h("p", { class: "section__lead", text: t("You become its admin and can invite people next.") }),
          field(t("Name"), name),
          st.el,
          actions(cancel, create),
        );
        form.addEventListener("submit", async (e) => {
          e.preventDefault();
          st.clear();
          try {
            const res = await busy(create, () => engine.api.workspaces.create(name.value));
            await engine.refreshMe();
            engine.state.set("workspaceId", res.workspace.id);
            close();
            toast({ message: t("{name} is ready.", { name: res.workspace.name }), tone: "success", action: { label: t("Invite people"), run: () => engine.openOverlay("workspace", "members") } });
          } catch (err) {
            st.error(err);
          }
        });
        body.append(form);
        return name;
      },
      () => {
        if (engine.state.get("overlay") === "new-workspace") engine.state.set("overlay", null);
      },
    );
  });
}
