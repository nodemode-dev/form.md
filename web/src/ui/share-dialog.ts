import type { AccessLevel, Engine, Group, GuestInvite, Member, Share, ShareLink, Sharing } from "../engine";
import { formatDate } from "../engine/format";
import { t } from "../engine/i18n";
import { avatar } from "./avatar";
import { simpleDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { actions, busy, button, copyText, iconButton, input, section, select, status } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { toast } from "./toast";

const DAY = 86_400_000;

export const levelOptions = (): readonly (readonly [AccessLevel, string])[] => [
  ["full", t("Full access")],
  ["edit", t("Can edit")],
  ["content", t("Can edit content")],
  ["comment", t("Can comment")],
  ["view", t("Can view")],
];

const levelName = (l: string) => levelOptions().find(([v]) => v === l)?.[1] ?? l;

const expiryOptions = (): readonly (readonly [string, string])[] => [
  ["0", t("No end date")],
  ["1", t("For 1 day")],
  ["7", t("For 7 days")],
  ["30", t("For 30 days")],
];

const expiresAt = (days: string) => (days === "0" ? 0 : Date.now() + Number(days) * DAY);

export function shareUrl(token: string): string {
  return `${location.origin}/l/${token}`;
}

export function openShareDialog(engine: Engine, pageId: string): void {
  const page = engine.pages.get(pageId) ?? engine.state.get("detail");
  const title = page?.title || t("Untitled");
  const ws = page?.workspace_id ?? "";
  const member = engine.canWrite(ws) || engine.workspaceById(ws)?.role === "viewer";
  const myId = engine.state.get("me")?.user.id ?? "";

  simpleDialog(
    t("Share {title}", { title }),
    (body, close) => {
      const st = status();
      const who = input({ placeholder: t("Email, person or group"), autocomplete: "off" });
      const options = h("datalist", { attrs: { id: `share-who-${pageId}` } });
      who.setAttribute("list", options.id);
      who.setAttribute("aria-label", t("Who to share with"));
      const level = select(levelOptions(), "edit");
      level.setAttribute("aria-label", t("Access level"));
      const until = select(expiryOptions(), "0");
      until.setAttribute("aria-label", t("Access ends"));
      const invite = button(t("Invite"), "primary", undefined, "submit");
      const reveal = h("div", { class: "secret-box" });
      reveal.hidden = true;
      const form = h("form", { class: "form-grid share-invite", attrs: { novalidate: "" } }, h("div", { class: "share-invite__row" }, who, level, until, invite), options, reveal, st.el);

      const general = h("div", { class: "row share-general" }, iconNode("globe"), h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })));
      const people = h("div", { class: "rows" });
      const peopleStatus = status();
      const access = section(t("People with access"));
      access.append(general, people, peopleStatus.el);

      const linkLevel = select<"view" | "comment">(
        [
          ["view", t("Can view")],
          ["comment", t("Can comment")],
        ],
        "view",
      );
      linkLevel.setAttribute("aria-label", t("Link access"));
      const linkUntil = select(expiryOptions(), "0");
      linkUntil.setAttribute("aria-label", t("Link ends"));
      const linkPassword = input({ type: "password", placeholder: t("Password (optional)"), autocomplete: "new-password" });
      linkPassword.setAttribute("aria-label", t("Link password"));
      const makeLink = button(t("Create link"), "default", "link");
      const linkReveal = h("div", { class: "secret-box" });
      linkReveal.hidden = true;
      const links = h("div", { class: "rows" });
      const linkStatus = status();
      const linkSection = section(t("Share links"), t("Anyone with the link can open this page without an account, with view or comment access."));
      linkSection.append(h("div", { class: "share-invite__row" }, linkLevel, linkUntil, linkPassword, makeLink), linkReveal, links, linkStatus.el);

      let state: Sharing | null = null;
      let everyone: Member[] = [];
      let groups: Group[] = [];

      const fillOptions = () => {
        const want = [...everyone.filter((m) => !m.disabled).map((m) => [m.email, m.name] as const), ...groups.map((g) => [t("Group: {name}", { name: g.name }), t("{count} members", { count: g.members })] as const)];
        const have = new Map(Array.from(options.children).map((o) => [(o as HTMLOptionElement).value, o as HTMLOptionElement]));
        for (const [value, label] of want) {
          if (have.has(value)) {
            have.delete(value);
            continue;
          }
          options.append(h("option", { attrs: { value, label } }));
        }
        for (const o of have.values()) o.remove();
      };

      const shareRow = (s: Share) => {
        const name = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const side = h("div", { class: "row__actions" });
        const face = s.kind === "group" ? h("span", { class: "row__icon" }, iconNode("users")) : avatar(s.name, "");
        return h("div", { class: "row" }, face, h("div", { class: "row__main" }, name, meta), side);
      };
      const updateShare = (r: HTMLElement, s: Share) => {
        const [name, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(name, s.subject_id === myId && s.kind === "user" ? t("{name} (you)", { name: s.name }) : s.name);
        const bits = [s.kind === "group" ? t("Group") : (s.email ?? ""), s.guest ? t("Guest") : "", s.inherited ? t("From {title}", { title: s.from_title || t("Untitled") }) : "", s.expires_at ? t("until {date}", { date: formatDate(s.expires_at) }) : ""];
        setText(meta, bits.filter(Boolean).join(" · "));
        const side = r.querySelector(".row__actions") as HTMLElement;
        const editable = !!state?.can_share && !s.inherited;
        if (side.dataset.mode !== (editable ? "edit" : s.subject_id === myId && s.kind === "user" && !s.inherited ? "self" : "read")) {
          side.replaceChildren();
          if (editable) {
            const sel = select(levelOptions(), s.level);
            sel.setAttribute("aria-label", t("Access of {name}", { name: s.name }));
            sel.addEventListener("change", async () => {
              peopleStatus.clear();
              try {
                await engine.api.sharing.update(r.dataset.key ?? "", sel.value, Number(r.dataset.expires ?? "0"));
                await refresh();
              } catch (err) {
                peopleStatus.error(err);
                await refresh();
              }
            });
            const remove = iconButton("trash", t("Remove access"));
            remove.addEventListener("click", () => void removeShare(r.dataset.key ?? ""));
            side.append(sel, remove);
            side.dataset.mode = "edit";
          } else if (s.subject_id === myId && s.kind === "user" && !s.inherited) {
            const leave = button(t("Leave"), "ghost");
            leave.addEventListener("click", () => void removeShare(r.dataset.key ?? "", true));
            side.append(h("span", { class: "badge" }), leave);
            side.dataset.mode = "self";
          } else {
            side.append(h("span", { class: "badge" }));
            side.dataset.mode = "read";
          }
        }
        r.dataset.expires = String(s.expires_at);
        const sel = side.querySelector<HTMLSelectElement>("select");
        if (sel && sel.value !== s.level) sel.value = s.level;
        const badge = side.querySelector<HTMLElement>(".badge");
        if (badge) setText(badge, levelName(s.level));
      };

      const inviteRow = () => {
        const name = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const cancel = iconButton("trash", t("Cancel invitation"));
        const r = h("div", { class: "row" }, iconNode("mail"), h("div", { class: "row__main" }, name, meta), h("div", { class: "row__actions" }, cancel));
        cancel.addEventListener("click", async () => {
          peopleStatus.clear();
          try {
            await engine.api.sharing.cancelInvite((r.dataset.key ?? "").slice(4));
            await refresh();
          } catch (err) {
            peopleStatus.error(err);
          }
        });
        return r;
      };
      const updateInvite = (r: HTMLElement, g: GuestInvite) => {
        const [name, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(name, g.email);
        setText(meta, [t("Invited as a guest"), levelName(g.level), t("expires {date}", { date: formatDate(g.expires_at) })].join(" · "));
      };

      const linkRow = () => {
        const name = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const revoke = iconButton("trash", t("Turn off link"));
        const r = h("div", { class: "row" }, iconNode("link"), h("div", { class: "row__main" }, name, meta), h("div", { class: "row__actions" }, revoke));
        revoke.addEventListener("click", async () => {
          linkStatus.clear();
          try {
            await engine.api.sharing.revokeLink(r.dataset.key ?? "");
            await refresh();
          } catch (err) {
            linkStatus.error(err);
          }
        });
        return r;
      };
      const updateLink = (r: HTMLElement, l: ShareLink) => {
        const [name, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(name, levelName(l.level));
        setText(meta, [t("made {date}", { date: formatDate(l.created_at) }), l.password ? t("password") : "", l.expires_at ? t("until {date}", { date: formatDate(l.expires_at) }) : ""].filter(Boolean).join(" · "));
      };

      const paint = (s: Sharing) => {
        state = s;
        const [gTitle, gMeta] = general.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        if (s.general === "private") {
          setText(gTitle, t("Private"));
          setText(gMeta, t("Only its owner and the people below can open it."));
        } else if (s.general === "teamspace" && s.teamspace) {
          setText(gTitle, s.teamspace.name);
          setText(gMeta, s.teamspace.access === "open" ? t("Everyone in the workspace can open it.") : t("Members of this teamspace can open it."));
        } else {
          setText(gTitle, engine.workspaceById(ws)?.name ?? t("Workspace"));
          setText(gMeta, t("Everyone in the workspace can open it."));
        }
        setAttr(general, "data-general", s.general);
        type Item = { key: string; share?: Share; invite?: GuestInvite };
        const items: Item[] = [...s.shares.map((x) => ({ key: x.id, share: x })), ...s.invites.map((x) => ({ key: `inv:${x.id}`, invite: x }))];
        reconcile(
          people,
          items,
          (i) => i.key,
          (i) => (i.share ? shareRow(i.share) : inviteRow()),
          (el, i) => (i.share ? updateShare(el, i.share) : i.invite ? updateInvite(el, i.invite) : undefined),
        );
        form.hidden = !s.can_share;
        linkSection.hidden = !s.can_share;
        reconcile(links, s.links, (l) => l.id, linkRow, updateLink);
      };

      const refresh = async () => {
        try {
          paint(await engine.api.sharing.get(pageId));
        } catch (err) {
          peopleStatus.error(err);
        }
      };

      const removeShare = async (id: string, self = false) => {
        peopleStatus.clear();
        try {
          await engine.api.sharing.remove(id);
          if (self) {
            close();
            toast({ message: t("You left {title}.", { title }) });
            return;
          }
          await refresh();
        } catch (err) {
          peopleStatus.error(err);
        }
      };

      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st.clear();
        reveal.hidden = true;
        const raw = who.value.trim();
        if (!raw) return;
        const group = groups.find((g) => t("Group: {name}", { name: g.name }) === raw || g.name === raw);
        const person = everyone.find((m) => m.email.toLowerCase() === raw.toLowerCase() || m.name === raw);
        const req = group
          ? { kind: "group" as const, subject_id: group.id, level: level.value, expires_at: expiresAt(until.value) }
          : person
            ? { kind: "user" as const, subject_id: person.user_id, level: level.value, expires_at: expiresAt(until.value) }
            : { kind: "user" as const, email: raw, level: level.value, expires_at: expiresAt(until.value) };
        try {
          const res = await busy(invite, () => engine.api.sharing.share(pageId, req));
          who.value = "";
          if (res.invite_url) {
            const url = `${location.origin}${res.invite_url}`;
            const copy = iconButton("copy", t("Copy link"));
            copy.addEventListener("click", async () => toast({ message: (await copyText(url)) ? t("Invitation link copied.") : t("Copying failed."), id: "copy" }));
            reveal.replaceChildren(h("span", { text: url }), copy);
            reveal.hidden = false;
            st.say(res.emailed ? t("Invitation sent. They join as a guest with access to this page only.") : t("Send them this link. They join as a guest with access to this page only."), "success");
          } else {
            st.say(t("Shared."), "success");
          }
          await refresh();
        } catch (err) {
          st.error(err);
        }
      });

      makeLink.addEventListener("click", async () => {
        linkStatus.clear();
        linkReveal.hidden = true;
        try {
          const body: { level: "view" | "comment"; expires_at?: number; password?: string } = { level: linkLevel.value as "view" | "comment" };
          const ends = expiresAt(linkUntil.value);
          if (ends) body.expires_at = ends;
          if (linkPassword.value) body.password = linkPassword.value;
          const r = await busy(makeLink, () => engine.api.sharing.createLink(pageId, body));
          const url = shareUrl(r.link.token ?? "");
          const copied = await copyText(url);
          const copy = iconButton("copy", t("Copy link"));
          copy.addEventListener("click", async () => toast({ message: (await copyText(url)) ? t("Link copied") : t("Copying failed."), id: "copy" }));
          linkReveal.replaceChildren(h("span", { text: url }), copy);
          linkReveal.hidden = false;
          linkPassword.value = "";
          linkStatus.say(copied ? t("Link copied. It is shown only once.") : t("Copy this link now. It is shown only once."), "success");
          await refresh();
        } catch (err) {
          linkStatus.error(err);
        }
      });

      body.append(form, access, linkSection);
      form.hidden = true;
      linkSection.hidden = true;
      void refresh();
      if (member && engine.canWrite(ws)) {
        void Promise.all([engine.api.workspaces.members(ws), engine.api.workspaces.groups(ws)])
          .then(([m, g]) => {
            everyone = m.members;
            groups = g.groups;
            fillOptions();
          })
          .catch(() => undefined);
      }
      body.append(actions(button(t("Done"), "ghost")));
      body.lastElementChild?.firstElementChild?.addEventListener("click", close);
      return who;
    },
    undefined,
    "md",
  );
}
