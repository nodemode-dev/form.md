import type { Engine, Me, NotifyPrefs, SessionInfo, Token } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { agentsTab } from "./account-agents";
import { avatar, COLORS, updateAvatar } from "./avatar";
import { confirmDialog, overlayDialog, type Tab } from "./dialog";
import { disclosure } from "./disclosure";
import { h, setAttr, setText } from "./dom";
import { actions, busy, button, choices, copyText, field, iconButton, input, section, status, toggle } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { uploadPicture } from "./pickers";
import { preferencesTab } from "./preferences";
import { currentSubscription, disablePush, enablePush, pushSupported } from "./push";
import { toast } from "./toast";

const COLOR_LABEL: Record<string, () => string> = {
  violet: () => t("Violet"),
  blue: () => t("Blue"),
  cyan: () => t("Cyan"),
  teal: () => t("Teal"),
  green: () => t("Green"),
  lime: () => t("Lime"),
  amber: () => t("Amber"),
  orange: () => t("Orange"),
  rose: () => t("Rose"),
  pink: () => t("Pink"),
};

function me(engine: Engine): Me {
  const m = engine.state.get("me");
  if (!m) throw new Error("not signed in");
  return m;
}

export function colorPicker(name: string, value: string, onPick: (color: string) => void): { el: HTMLElement; value(): string } {
  const el = h("div", { class: "swatches", attrs: { role: "radiogroup" } });
  for (const c of COLORS) {
    const r = h("input", { attrs: { type: "radio", name, value: c, "aria-label": COLOR_LABEL[c]?.() ?? c } });
    r.checked = c === value;
    r.addEventListener("change", () => onPick(c));
    el.append(h("label", { class: "swatch", attrs: { "data-color": c, "data-tip": COLOR_LABEL[c]?.() ?? c } }, r));
  }
  return { el, value: () => el.querySelector<HTMLInputElement>("input:checked")?.value ?? value };
}

export function describeAgent(ua: string): string {
  const browser = /Edg\//.test(ua)
    ? "Edge"
    : /Firefox\//.test(ua)
      ? "Firefox"
      : /Chrome\//.test(ua)
        ? "Chrome"
        : /Safari\//.test(ua)
          ? "Safari"
          : /curl|Go-http|python|node/i.test(ua)
            ? t("Script")
            : t("Browser");
  const os = /iPhone|iPad/.test(ua)
    ? "iOS"
    : /Android/.test(ua)
      ? "Android"
      : /Mac OS X|Macintosh/.test(ua)
        ? "macOS"
        : /Windows/.test(ua)
          ? "Windows"
          : /Linux/.test(ua)
            ? "Linux"
            : "";
  return os ? t("{browser} on {os}", { browser, os }) : browser;
}

function profileTab(engine: Engine): Tab {
  return {
    id: "profile",
    label: t("Profile"),
    icon: "user",
    render(panel) {
      const u = me(engine).user;

      const preview = avatar(u.name, u.color, true, u.avatar);
      const name = input({ name: "name", value: u.name, autocomplete: "name", maxlength: 80 });
      const colors = colorPicker("profile-color", u.color, () => updateAvatar(preview, name.value || u.name, colors.value(), me(engine).user.avatar));
      name.addEventListener("input", () => updateAvatar(preview, name.value || me(engine).user.name, colors.value(), me(engine).user.avatar));
      const picUpload = button(t("Upload picture"), "ghost", "upload");
      const picRemove = button(t("Remove picture"), "ghost", "trash");
      const paintPicture = () => {
        const cur = me(engine).user;
        picRemove.hidden = !cur.avatar;
        updateAvatar(preview, name.value || cur.name, colors.value(), cur.avatar);
      };
      const savePicture = async (ref: string) => {
        try {
          engine.setMe(await engine.api.me.update({ avatar: ref }));
          paintPicture();
        } catch (err) {
          idStatus.error(err);
        }
      };
      picUpload.addEventListener("click", () =>
        void uploadPicture(engine, { purpose: "avatar" }).then((ref) => {
          if (ref) void savePicture(ref);
        }),
      );
      picRemove.addEventListener("click", () => void savePicture(""));
      picRemove.hidden = !u.avatar;
      const idStatus = status();
      const idSave = button(t("Save"), "primary", undefined, "submit");
      const idForm = h(
        "form",
        { class: "form-grid", attrs: { novalidate: "" } },
        h("div", { class: "profile-head" }, h("div", { class: "profile-pic" }, preview, picUpload, picRemove), h("div", { class: "form-grid" }, field(t("Name"), name), field(t("Colour"), colors.el))),
        idStatus.el,
        actions(idSave),
      );
      idForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        idStatus.clear();
        try {
          engine.setMe(await busy(idSave, () => engine.api.me.update({ name: name.value, color: colors.value() })));
          idStatus.say(t("Saved."), "success");
        } catch (err) {
          idStatus.error(err);
        }
      });

      const emailNow = h("span", { class: "row__title" });
      const emailBadge = h("span", { class: "badge" });
      const paintEmail = () => {
        const cur = me(engine).user;
        setText(emailNow, cur.email);
        setText(emailBadge, cur.email_verified ? t("Verified") : t("Not verified"));
        setAttr(emailBadge, "class", cur.email_verified ? "badge badge--accent" : "badge");
      };
      paintEmail();
      const newEmail = input({ type: "email", name: "email", autocomplete: "email" });
      const emailPassword = input({ type: "password", name: "current-password", autocomplete: "current-password" });
      const emailStatus = status();
      const emailSave = button(t("Change email"), "default", undefined, "submit");
      const emailForm = h(
        "form",
        { class: "form-grid", attrs: { novalidate: "" } },
        h("div", { class: "rows" }, h("div", { class: "row" }, iconNode("mail"), h("div", { class: "row__main" }, emailNow), emailBadge)),
        h("div", { class: "form-row" }, field(t("New email"), newEmail), field(t("Current password"), emailPassword)),
        emailStatus.el,
        actions(emailSave),
      );
      emailForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        emailStatus.clear();
        try {
          engine.setMe(await busy(emailSave, () => engine.api.me.email(emailPassword.value, newEmail.value)));
          newEmail.value = "";
          emailPassword.value = "";
          paintEmail();
          emailStatus.say(t("Email changed."), "success");
        } catch (err) {
          emailStatus.error(err);
        }
      });

      const current = input({ type: "password", name: "current-password", autocomplete: "current-password" });
      const next = input({ type: "password", name: "new-password", autocomplete: "new-password" });
      const again = input({ type: "password", name: "confirm-password", autocomplete: "new-password" });
      const pwStatus = status();
      const pwSave = button(t("Change password"), "default", undefined, "submit");
      const pwForm = h(
        "form",
        { class: "form-grid", attrs: { novalidate: "" } },
        field(t("Current password"), current),
        h("div", { class: "form-row" }, field(t("New password"), next, t("At least 8 characters.")), field(t("Repeat the new password"), again)),
        pwStatus.el,
        actions(pwSave),
      );
      pwForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        pwStatus.clear();
        if (next.value !== again.value) {
          pwStatus.say(t("The two new passwords do not match."), "error");
          return;
        }
        try {
          await busy(pwSave, () => engine.api.me.password(current.value, next.value));
          for (const el of [current, next, again]) el.value = "";
          toast({ message: t("Password changed. Every other session was signed out."), tone: "success" });
        } catch (err) {
          pwStatus.error(err);
        }
      });

      const identity = section(t("Identity"));
      identity.append(idForm);
      const email = section(t("Email"), t("Changing the address needs your current password."));
      email.append(emailForm);
      const password = section(t("Password"), t("A new password signs out every other session and revokes your API tokens."));
      password.append(pwForm);

      panel.append(identity, email, password);
      return () => {
        paintEmail();
        updateAvatar(preview, me(engine).user.name, me(engine).user.color);
      };
    },
  };
}

function twoFactorSection(engine: Engine): HTMLElement {
  const el = section(t("Two-factor sign-in"));
  const body = h("div", { class: "form-grid" });
  el.append(body);

  const showOff = () => {
    const start = button(t("Set up two-factor sign-in"), "primary", "shield");
    const st = status();
    body.replaceChildren(
      h("p", { class: "section__lead", text: t("Protect your account with an authenticator app such as 1Password, Google Authenticator or Aegis.") }),
      st.el,
      actions(start),
    );
    start.addEventListener("click", async () => {
      st.clear();
      try {
        const setup = await busy(start, () => engine.api.me.totpSetup());
        showEnrol(setup.secret, setup.qr_svg);
      } catch (err) {
        st.error(err);
      }
    });
  };

  const showEnrol = (secret: string, svg: string) => {
    const qr = h("img", {
      class: "qr",
      attrs: { src: `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`, alt: t("QR code for the authenticator app"), width: "168", height: "168" },
    });
    const copy = iconButton("copy", t("Copy key"));
    copy.addEventListener("click", async () => toast({ message: (await copyText(secret)) ? t("Key copied.") : t("Copying failed."), id: "copy" }));
    const code = input({ name: "code", inputmode: "numeric", autocomplete: "one-time-code", maxlength: 6 });
    code.classList.add("input--code");
    const st = status();
    const confirm = button(t("Turn on"), "primary", undefined, "submit");
    const cancel = button(t("Cancel"), "ghost");
    cancel.addEventListener("click", showOff);
    const form = h(
      "form",
      { class: "enrol", attrs: { novalidate: "" } },
      qr,
      h(
        "div",
        { class: "form-grid" },
        h("p", { class: "section__lead", text: t("Scan the code with your app. No camera? Type the key in by hand:") }),
        h("div", { class: "secret-box" }, h("span", { text: secret.replace(/(.{4})/g, "$1 ").trim() }), copy),
        field(t("Six-digit code from the app"), code),
        st.el,
        actions(cancel, confirm),
      ),
    );
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      st.clear();
      try {
        const res = await busy(confirm, () => engine.api.me.totpConfirm(code.value.replace(/\s/g, "")));
        await engine.refreshMe();
        showCodes(res.recovery_codes);
      } catch (err) {
        st.error(err);
      }
    });
    body.replaceChildren(form);
    code.focus();
  };

  const showCodes = (codes: string[]) => {
    const text = codes.join("\n") + "\n";
    const copy = button(t("Copy"), "default", "copy");
    copy.addEventListener("click", async () => toast({ message: (await copyText(text)) ? t("Recovery codes copied.") : t("Copying failed."), id: "copy" }));
    const download = button(t("Download"), "default", "download");
    download.addEventListener("click", () => {
      const a = h("a", { attrs: { href: URL.createObjectURL(new Blob([text], { type: "text/plain" })), download: "form-md-recovery-codes.txt" } });
      a.click();
      setTimeout(() => URL.revokeObjectURL(a.href), 1000);
    });
    const done = button(t("I have saved them"), "primary");
    done.addEventListener("click", showOn);
    body.replaceChildren(
      h("div", { class: "callout callout--warning" }, iconNode("warning"), h("p", { text: t("Two-factor sign-in is on. Save these recovery codes now. Each works once if you lose your phone, and they are not shown again.") })),
      h("div", { class: "codes" }, ...codes.map((c) => h("span", { text: c }))),
      actions(copy, download, done),
    );
    done.focus();
  };

  const showOn = () => {
    const code = input({ name: "code", inputmode: "numeric", autocomplete: "one-time-code", maxlength: 11 });
    code.classList.add("input--code");
    const st = status();
    const off = button(t("Turn off"), "danger", undefined, "submit");
    const form = h(
      "form",
      { class: "form-grid", attrs: { novalidate: "" } },
      h("div", { class: "rows" }, h("div", { class: "row" }, iconNode("shield"), h("div", { class: "row__main" }, h("span", { class: "row__title", text: t("Authenticator app") })), h("span", { class: "badge badge--accent", text: t("On") }))),
      field(t("Code or recovery code"), code, t("Enter a current code to turn two-factor sign-in off.")),
      st.el,
      actions(off),
    );
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      st.clear();
      const v = code.value.trim();
      const recovery = v.includes("-") || v.length > 6;
      try {
        await busy(off, () => engine.api.me.totpDisable(recovery ? "" : v, recovery ? v : ""));
        await engine.refreshMe();
        toast({ message: t("Two-factor sign-in is off."), tone: "warning" });
        showOff();
      } catch (err) {
        st.error(err);
      }
    });
    body.replaceChildren(form);
  };

  if (me(engine).user.totp_enabled) showOn();
  else showOff();
  return el;
}

function sessionsSection(engine: Engine): { el: HTMLElement; refresh(): void } {
  const el = section(t("Signed-in devices"), t("Each browser you signed in with. Ending a session signs that device out."));
  const list = h("div", { class: "rows" });
  const st = status();
  const others = button(t("Sign out every other device"), "default", "logout");
  el.append(list, st.el, actions(others));

  const row = () => {
    const title = h("span", { class: "row__title" });
    const meta = h("span", { class: "row__meta" });
    const end = button(t("Sign out"), "ghost");
    const here = h("span", { class: "badge badge--accent", text: t("This device") });
    const r = h("div", { class: "row" }, iconNode("monitor"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, here, end));
    end.addEventListener("click", async () => {
      try {
        await busy(end, () => engine.api.me.endSession(r.dataset.key ?? ""));
        refresh();
      } catch (err) {
        st.error(err);
      }
    });
    return r;
  };
  const update = (r: HTMLElement, s: SessionInfo) => {
    const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
    setText(title, describeAgent(s.user_agent));
    setText(meta, [s.ip, t("active {when}", { when: formatRelative(s.last_seen_at) }), t("signed in {date}", { date: formatDateTime(s.created_at) })].filter(Boolean).join(" · "));
    r.querySelector<HTMLElement>(".row__actions .btn")!.hidden = s.current;
    r.querySelector<HTMLElement>(".row__actions .badge")!.hidden = !s.current;
  };
  const refresh = () => {
    engine.api.me
      .sessions()
      .then(({ sessions }) => {
        reconcile(list, sessions, (s) => s.id, row, update);
        others.disabled = sessions.length < 2;
      })
      .catch((err) => st.error(err));
  };
  others.addEventListener("click", async () => {
    try {
      await busy(others, () => engine.api.me.endSession("others"));
      toast({ message: t("Every other device was signed out."), tone: "success" });
      refresh();
    } catch (err) {
      st.error(err);
    }
  });
  refresh();
  return { el, refresh };
}

function notificationsTab(engine: Engine): Tab {
  return {
    id: "notifications",
    label: t("Notifications"),
    icon: "bell",
    render(panel) {
      const emailSt = status();
      const email = choices<NotifyPrefs["email"]>(
        "notify-email",
        [
          ["all", t("Everything in my inbox")],
          ["mentions", t("Only mentions, assignments and reminders")],
          ["off", t("Never")],
        ],
        "all",
      );
      const mailNote = h("p", { class: "muted" });
      const mailSec = section(t("Email"), t("When you have not opened form.md for a while, unread notifications arrive by email."));
      mailSec.append(email.el, mailNote, emailSt.el);
      email.el.addEventListener("change", async () => {
        emailSt.clear();
        try {
          await engine.api.inbox.setPrefs(email.value());
          emailSt.say(t("Saved."), "success");
        } catch (e) {
          emailSt.error(e);
        }
      });

      const pushSt = status();
      const pushState = h("p", { class: "muted" });
      const on = button(t("Turn on for this browser"), "primary", "bell");
      const off = button(t("Turn off for this browser"), "ghost");
      const test = button(t("Send a test"), "ghost");
      const pushSec = section(t("Push notifications"), t("If you do not see a notification within ten seconds, this browser shows it even when form.md is in the background."));
      pushSec.append(pushState, actions(off, test, on), pushSt.el);
      panel.append(mailSec, pushSec);

      let key = "";
      const refresh = async () => {
        try {
          const prefs = await engine.api.inbox.prefs();
          key = prefs.push_key;
          email.set(prefs.email);
          setText(mailNote, prefs.mail_enabled ? "" : t("Email is not set up on this instance yet, so nothing is sent for now."));
          mailNote.hidden = prefs.mail_enabled;
          const sub = await currentSubscription();
          const supported = pushSupported();
          on.hidden = !supported || !!sub;
          off.hidden = !sub;
          test.hidden = !sub;
          setText(pushState, !supported ? t("This browser cannot receive push notifications.") : sub ? t("On for this browser.") : Notification.permission === "denied" ? t("Blocked in this browser's site settings.") : t("Off for this browser."));
        } catch (e) {
          pushSt.error(e);
        }
      };
      on.addEventListener("click", async () => {
        pushSt.clear();
        try {
          const r = await busy(on, () => enablePush(engine, key));
          if (r === "denied") pushSt.say(t("The browser did not allow notifications."), "error");
          await refresh();
        } catch (e) {
          pushSt.error(e);
        }
      });
      off.addEventListener("click", async () => {
        pushSt.clear();
        try {
          await busy(off, () => disablePush(engine));
          await refresh();
        } catch (e) {
          pushSt.error(e);
        }
      });
      test.addEventListener("click", async () => {
        pushSt.clear();
        try {
          await busy(test, () => engine.api.inbox.testPush());
          pushSt.say(t("Sent. It should appear in a moment."), "success");
        } catch (e) {
          pushSt.error(e);
        }
      });
      void refresh();
      return () => void refresh();
    },
  };
}

function securityTab(engine: Engine): Tab {
  return {
    id: "security",
    label: t("Security"),
    icon: "shield",
    render(panel) {
      const sessions = sessionsSection(engine);
      panel.append(twoFactorSection(engine), sessions.el);
      return () => sessions.refresh();
    },
  };
}

function tokensTab(engine: Engine): Tab {
  return {
    id: "tokens",
    label: t("API tokens"),
    icon: "key",
    render(panel) {
      const wsName = (id: string) => me(engine).workspaces.find((w) => w.id === id)?.name ?? t("a workspace you left");

      const reveal = h("div", { class: "form-grid" });
      reveal.hidden = true;

      const name = input({ name: "token-name", placeholder: t("for example claude-code"), maxlength: 80, autocomplete: "off" });
      const scope = choices("token-scope", [["write", t("Read and write")], ["read", t("Read only")]] as const, "write");
      const limit = toggle(t("Only chosen workspaces"), false, t("The token reaches nothing else, even if you join more workspaces later."));
      const picks = h("div", { class: "check-list" });
      picks.hidden = true;
      const fillPicks = () => {
        reconcile(
          picks,
          me(engine).workspaces.filter((w) => !w.break_glass),
          (w) => w.id,
          (w) => h("label", { class: "check" }, h("input", { attrs: { type: "checkbox", value: w.id } }), h("span")),
          (el, w) => setText(el.querySelector("span")!, w.name),
        );
      };
      limit.input.addEventListener("change", () => {
        if (limit.input.checked) fillPicks();
        picks.hidden = !limit.input.checked;
      });
      const st = status();
      const create = button(t("Create token"), "primary", "plus", "submit");
      const form = h(
        "form",
        { class: "form-grid", attrs: { novalidate: "" } },
        field(t("Name"), name, t("Shown in the activity log next to what the token did.")),
        field(t("Access"), scope.el, t("Read-only tokens cannot create, edit, delete or upload.")),
        limit.el,
        picks,
        st.el,
        actions(create),
      );

      const list = h("div", { class: "rows" });
      const empty = h("p", { class: "muted", text: t("No tokens yet.") });
      const listStatus = status();

      const row = () => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const revoke = iconButton("trash", t("Revoke"));
        const r = h("div", { class: "row" }, iconNode("key"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, h("span", { class: "badge" }), revoke));
        revoke.addEventListener("click", async () => {
          const ok = await confirmDialog({
            title: t("Revoke {name}?", { name: title.textContent ?? "" }),
            body: t("Anything using this token stops working immediately."),
            confirm: t("Revoke"),
            variant: "danger",
          });
          if (!ok) return;
          try {
            await engine.api.me.deleteToken(r.dataset.key ?? "");
            refresh();
          } catch (err) {
            listStatus.error(err);
          }
        });
        return r;
      };
      const update = (r: HTMLElement, tk: Token) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, tk.name);
        const reach = tk.all_workspaces ? t("all workspaces") : tk.workspaces.map(wsName).join(", ");
        const used = tk.last_used_at ? t("used {when} from {ip}", { when: formatRelative(tk.last_used_at), ip: tk.last_used_ip || "?" }) : t("never used");
        setText(meta, [`${tk.prefix}...`, reach, t("created {date}", { date: formatDateTime(tk.created_at) }), used].join(" · "));
        const badge = r.querySelector<HTMLElement>(".badge")!;
        setText(badge, tk.scope === "read" ? t("Read only") : t("Read and write"));
      };
      const refresh = () => {
        engine.api.me
          .tokens()
          .then(({ tokens }) => {
            reconcile(list, tokens, (tk) => tk.id, row, update);
            empty.hidden = tokens.length > 0;
          })
          .catch((err) => listStatus.error(err));
      };

      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st.clear();
        const chosen = limit.input.checked ? [...picks.querySelectorAll<HTMLInputElement>("input:checked")].map((i) => i.value) : null;
        try {
          const res = await busy(create, () => engine.api.me.createToken(name.value, scope.value(), chosen));
          const copy = async () => toast({ message: (await copyText(res.token)) ? t("Token copied.") : t("Copying failed."), id: "copy" });
          const copyBtn = button(t("Copy token"), "primary", "copy");
          const copyIcon = iconButton("copy", t("Copy token"));
          copyBtn.addEventListener("click", copy);
          copyIcon.addEventListener("click", copy);
          const hide = button(t("Done"), "ghost");
          hide.addEventListener("click", () => {
            reveal.hidden = true;
            reveal.replaceChildren();
          });
          reveal.replaceChildren(
            h("div", { class: "callout callout--warning" }, iconNode("warning"), h("p", { text: t("Copy this token now. It is not shown again.") })),
            h("div", { class: "secret-box" }, h("span", { text: res.token }), copyIcon),
            actions(hide, copyBtn),
          );
          reveal.hidden = false;
          name.value = "";
          refresh();
        } catch (err) {
          st.error(err);
        }
      });

      const intro = section(t("API tokens"), t("Tokens let agents and scripts use the REST API and the MCP endpoint as you. A token can only narrow what you can do, never widen it."));
      intro.append(
        h(
          "p",
          { class: "muted" },
          h("a", { attrs: { href: "/api/docs", target: "_blank", rel: "noopener" }, text: t("API reference") }),
          document.createTextNode(" · "),
          h("a", { attrs: { href: "/api/openapi.json", target: "_blank", rel: "noopener" }, text: t("OpenAPI document") }),
        ),
      );
      const make = section(t("New token"));
      make.append(form);
      const existing = section(t("Your tokens"));
      existing.append(list, empty, listStatus.el);
      panel.append(intro, reveal, make, existing);
      refresh();
      return () => {
        if (limit.input.checked) fillPicks();
        refresh();
      };
    },
  };
}

const DEV_KEY = "fm.dev";

function devOn(): boolean {
  try {
    return localStorage.getItem(DEV_KEY) === "1";
  } catch {
    return false;
  }
}

function advancedTab(engine: Engine): Tab {
  return {
    id: "advanced",
    label: t("Advanced"),
    icon: "key",
    render(panel) {
      const on = devOn();
      const dev = toggle(t("Show developer tools"), on, t("Adds the terminal and other tools for people who build with form.md. Remembered on this device."));
      const intro = section(t("Advanced"), t("For people who connect scripts and agents. Most people never need these."));
      intro.append(dev.el);

      const parts = [tokensTab(engine), agentsTab(engine)].map((tab) => {
        const holder = h("div", { class: "form-grid" });
        const part = disclosure(tab.label, holder);
        let refresh: (() => void) | undefined;
        let drawn = false;
        const draw = () => {
          if (drawn) return;
          drawn = true;
          const r = tab.render(holder);
          if (r) refresh = r;
        };
        part.el.open = on;
        part.el.addEventListener("toggle", () => {
          if (part.el.open) draw();
        });
        if (on) draw();
        return { part, refresh: () => refresh?.() };
      });

      dev.input.addEventListener("change", () => {
        document.documentElement.toggleAttribute("data-dev", dev.input.checked);
        try {
          if (dev.input.checked) localStorage.setItem(DEV_KEY, "1");
          else localStorage.removeItem(DEV_KEY);
        } catch {
          dev.input.checked = !dev.input.checked;
          return;
        }
        for (const p of parts) p.part.el.open = dev.input.checked;
      });

      panel.append(intro, ...parts.map((p) => p.part.el));
      return () => {
        for (const p of parts) p.refresh();
      };
    },
  };
}

export function mountAccount(engine: Engine): void {
  overlayDialog(engine, "account", t("Account settings"), [profileTab(engine), preferencesTab(engine), notificationsTab(engine), securityTab(engine), advancedTab(engine)]);
}
