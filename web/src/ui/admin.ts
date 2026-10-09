import { ApiError } from "../engine/api";
import type { AccessWorkspace, AdminUser, BreakGlassGrant, DeletionPlan, Engine, SettingsMap, StrandedWorkspace, TunnelStatus, WorkspaceRole } from "../engine";
import { formatDate, formatDateTime, formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { aiTab } from "./admin-ai";
import { avatar, updateAvatar } from "./avatar";
import { confirmDialog, confirmTyped, overlayDialog, simpleDialog, type Tab } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { actions, busy, button, choices, copyText, field, iconButton, input, section, select, status, textarea, toggle } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { toast } from "./toast";
import { roleOptions } from "./workspace-settings";

class SettingsStore {
  values: SettingsMap = {};
  secretSet = "";
  mailEnabled = false;
  private loading: Promise<void> | null = null;
  private listeners = new Set<() => void>();

  constructor(private readonly engine: Engine) {}

  load(force = false): Promise<void> {
    if (this.loading && !force) return this.loading;
    this.loading = this.engine.api.admin.settings().then((r) => this.apply(r));
    return this.loading;
  }

  async save(changes: SettingsMap): Promise<void> {
    this.apply(await this.engine.api.admin.saveSettings(changes));
  }

  get(key: string): string {
    return this.values[key] ?? "";
  }

  onChange(fn: () => void): void {
    this.listeners.add(fn);
  }

  private apply(r: { settings: SettingsMap; secret_set: string; mail_enabled: boolean }): void {
    this.values = r.settings;
    this.secretSet = r.secret_set;
    this.mailEnabled = r.mail_enabled;
    for (const fn of this.listeners) fn();
  }
}

interface Bound {
  key: string;
  label: string;
  control: HTMLElement;
  read(): string;
  write(): void;
}

class SettingsForm {
  readonly el = h("form", { class: "form-grid", attrs: { novalidate: "" } });
  private bound: Bound[] = [];
  private st = status();
  private saveBtn = button(t("Save"), "primary", undefined, "submit");

  constructor(private readonly store: SettingsStore) {
    this.el.addEventListener("submit", (e) => {
      e.preventDefault();
      void this.submit();
    });
    store.onChange(() => this.paint());
  }

  private bind(b: Bound): void {
    this.bound.push(b);
    b.control.addEventListener("input", () => setAttr(b.control, "aria-invalid", null));
  }

  text(key: string, label: string, hint?: string, type = "text", placeholder = ""): HTMLLabelElement {
    const el = input({ type, name: key, autocomplete: "off", placeholder });
    this.bind({ key, label, control: el, read: () => el.value.trim(), write: () => (el.value = this.store.get(key)) });
    return field(label, el, hint);
  }

  area(key: string, label: string, hint?: string, placeholder = ""): HTMLLabelElement {
    const el = textarea("", 6);
    el.name = key;
    el.placeholder = placeholder;
    el.spellcheck = false;
    this.bind({ key, label, control: el, read: () => el.value.trim(), write: () => (el.value = this.store.get(key)) });
    return field(label, el, hint);
  }

  number(key: string, label: string, hint?: string): HTMLLabelElement {
    const el = input({ type: "number", name: key, inputmode: "numeric" });
    this.bind({ key, label, control: el, read: () => el.value.trim(), write: () => (el.value = this.store.get(key)) });
    return field(label, el, hint);
  }

  secret(key: string, label: string, hint?: string): HTMLElement {
    const el = input({ type: "password", name: key, autocomplete: "new-password" });
    const remove = h("button", { class: "link-btn", attrs: { type: "button" }, text: t("Remove") });
    let cleared = false;
    remove.addEventListener("click", () => {
      cleared = true;
      el.value = "";
      el.placeholder = t("Removed when you save");
      remove.hidden = true;
    });
    this.bind({
      key,
      label,
      control: el,
      read: () => (el.value ? el.value : cleared ? "" : this.store.get(key)),
      write: () => {
        const set = this.store.get(key) === this.store.secretSet && this.store.secretSet !== "";
        cleared = false;
        el.value = "";
        el.placeholder = set ? t("Stored. Leave empty to keep it.") : "";
        remove.hidden = !set;
      },
    });
    const f = field(label, el, hint);
    f.append(remove);
    return f;
  }

  bool(key: string, label: string, hint?: string): HTMLElement {
    const sw = toggle(label, false, hint);
    this.bind({ key, label, control: sw.input, read: () => (sw.input.checked ? "true" : "false"), write: () => (sw.input.checked = this.store.get(key) === "true") });
    return sw.el;
  }

  choice<V extends string>(key: string, label: string, options: readonly (readonly [V, string])[], onPick?: (v: V) => void): HTMLLabelElement {
    const c = choices(key, options, options[0]![0]);
    this.bind({
      key,
      label,
      control: c.el,
      read: () => c.value(),
      write: () => {
        c.set(this.store.get(key) as V);
        onPick?.(c.value());
      },
    });
    if (onPick) c.el.addEventListener("change", () => onPick(c.value()));
    return field(label, c.el);
  }

  pick<V extends string>(key: string, label: string, options: readonly (readonly [V, string])[], hint?: string): HTMLLabelElement {
    const el = select(options);
    this.bind({ key, label, control: el, read: () => el.value, write: () => (el.value = this.store.get(key)) });
    return field(label, el, hint);
  }

  finish(extra?: HTMLElement): void {
    const row = actions(this.saveBtn);
    if (extra) row.prepend(extra);
    this.el.append(this.st.el, row);
  }

  paint(): void {
    for (const b of this.bound) b.write();
  }

  private async submit(): Promise<void> {
    this.st.clear();
    const changes: SettingsMap = {};
    for (const b of this.bound) {
      const v = b.read();
      if (v !== this.store.get(b.key)) changes[b.key] = v;
    }
    if (Object.keys(changes).length === 0) {
      this.st.say(t("Nothing changed."), "info");
      return;
    }
    try {
      await busy(this.saveBtn, () => this.store.save(changes));
      this.st.say(t("Settings saved."), "success");
    } catch (err) {
      const key = err instanceof ApiError ? String(err.data.key ?? "") : "";
      const b = this.bound.find((x) => x.key === key);
      if (b && err instanceof ApiError) {
        setAttr(b.control, "aria-invalid", "true");
        this.st.say(`${b.label}: ${err.message}`, "error");
        b.control.focus();
      } else {
        this.st.error(err);
      }
    }
  }
}

function settingsTab(store: SettingsStore, id: string, label: string, icon: Tab["icon"], build: (f: SettingsForm, panel: HTMLElement) => void, visible?: () => boolean): Tab {
  const tab: Tab = {
    id,
    label,
    icon,
    render(panel) {
      const f = new SettingsForm(store);
      build(f, panel);
      store
        .load()
        .then(() => f.paint())
        .catch((err) => toast({ message: String(err instanceof ApiError ? err.message : err), tone: "danger" }));
      return () => void store.load(true);
    },
  };
  if (visible) tab.visible = visible;
  return tab;
}

function generalTab(engine: Engine, store: SettingsStore): Tab {
  return settingsTab(store, "general", t("General"), "settings", (f, panel) => {
    const s1 = section(t("Instance"));
    s1.append(
      f.text("instance.name", t("Instance name"), t("Shown on the sign-in page and in the browser title.")),
      f.text("instance.public_url", t("Public address"), t("The address people use, for links in mail, invitations and sign-in callbacks."), "url", "https://notes.example.com"),
      f.bool("net.trust_proxy", t("Behind a reverse proxy"), t("Trust the X-Forwarded-Proto, X-Forwarded-Host and X-Forwarded-For headers. Turn on only when a proxy you run sets them.")),
      f.text("https.domain", t("Domain for automatic HTTPS"), t("The host name people reach this server at, for example notes.example.com."), "text", "notes.example.com"),
      f.bool("https.auto", t("Automatic HTTPS"), t("Takes effect at the next start: the server gets a certificate from Let's Encrypt and listens on ports 443 and 80 itself. Ignored when TLS is set in the environment.")),
      f.bool("workspaces.member_create", t("Members may create workspaces"), t("When off, only admins create shared workspaces. Everyone keeps a personal space.")),
    );
    const s2 = section(t("Housekeeping"));
    s2.append(
      f.number("trash.days", t("Empty the trash after (days)"), t("0 keeps trashed pages until someone empties the trash by hand.")),
      f.number("files.max_upload_mb", t("Largest upload (MB)"), t("Applies to attachments, covers, icons and pictures.")),
      f.pick(
        "audit.retention_days",
        t("Keep the activity log for"),
        [
          ["0", t("Forever")],
          ["30", t("30 days")],
          ["90", t("90 days")],
          ["180", t("180 days")],
          ["365", t("1 year")],
          ["730", t("2 years")],
        ] as const,
        t("Older entries are removed once a day. A change can only be taken back while its entry exists."),
      ),
      f.bool("update.check", t("Check for new versions"), t("Once a day, and only the version number is fetched.")),
    );
    const tokenRow = f.secret("tunnel.token", t("Tunnel token"));
    const s3 = section(t("Public tunnel"), t("Reach this server from the internet through Cloudflare without opening a port. Needs cloudflared on the server."));
    s3.append(
      f.choice(
        "tunnel.mode",
        t("Tunnel"),
        [
          ["off", t("Off")],
          ["quick", t("Quick tunnel (a random trycloudflare.com address)")],
          ["token", t("Named tunnel (a token from the Cloudflare dashboard)")],
        ] as const,
        (v) => {
          tokenRow.hidden = v !== "token";
        },
      ),
      tokenRow,
      f.bool("tunnel.autostart", t("Start with the server"), t("Otherwise start it here when you need it.")),
    );
    f.el.append(s1, s2, s3);
    f.finish();
    const state = h("p", { class: "section__lead" });
    const start = button(t("Start the tunnel"), "default", "globe");
    const stop = button(t("Stop the tunnel"), "default", "close");
    const paintTunnel = (st: TunnelStatus | null) => {
      if (!st) return;
      start.hidden = st.running;
      stop.hidden = !st.running;
      setText(state, !st.binary ? t("cloudflared is not installed on this server.") : st.running ? (st.url ? t("Running at {url}", { url: st.url }) : st.error ? t("Starting; last error: {error}", { error: st.error }) : t("Running.")) : st.error ? t("Stopped: {error}", { error: st.error }) : t("Stopped."));
    };
    start.addEventListener("click", () => void busy(start, () => engine.api.admin.tunnelStart().then(paintTunnel)).catch(engine.reportError));
    stop.addEventListener("click", () => void busy(stop, () => engine.api.admin.tunnelStop().then(paintTunnel)).catch(engine.reportError));
    void engine.api.admin.tunnel().then(paintTunnel, () => undefined);
    const s4 = section(t("Tunnel status"), t("Save the settings above first."));
    s4.append(state, h("div", { class: "form-actions form-actions--start" }, start, stop));
    panel.append(f.el, s4);
  });
}

function signInTab(store: SettingsStore): Tab {
  return settingsTab(store, "sign-in", t("Sign-in"), "lock", (f, panel) => {
    const domains = f.text("auth.signup_domains", t("Allowed email domains"), t("Comma separated, for example example.com, example.org."));
    const s1 = section(t("Who may create an account?"));
    s1.append(
      f.choice(
        "auth.signup",
        t("Registration"),
        [
          ["invite", t("By invitation only")],
          ["domain", t("Email domain allowed")],
          ["open", t("Open to anyone")],
        ] as const,
        (v) => {
          domains.hidden = v !== "domain";
        },
      ),
      domains,
      f.number("auth.session_days", t("Stay signed in for (days)"), t("Between 1 and 365. Applies to new sign-ins.")),
    );

    const callback = (id: string) => {
      const base = (store.get("instance.public_url") || location.origin).replace(/\/+$/, "");
      const url = `${base}/api/auth/sso/${id}/callback`;
      const copy = iconButton("copy", t("Copy address"));
      copy.addEventListener("click", async () => toast({ message: (await copyText(url)) ? t("Address copied.") : t("Copying failed."), id: "copy" }));
      const box = h("div", { class: "secret-box" }, h("span", { text: url }), copy);
      store.onChange(() => {
        const next = `${(store.get("instance.public_url") || location.origin).replace(/\/+$/, "")}/api/auth/sso/${id}/callback`;
        setText(box.firstElementChild!, next);
      });
      return field(t("Redirect address to register with the provider"), box);
    };

    const google = section(t("Google"), t("Google Cloud console, APIs and services, Credentials: an OAuth client of type web application."));
    google.append(h("div", { class: "form-row" }, f.text("sso.google.client_id", t("Client ID")), f.secret("sso.google.client_secret", t("Client secret"))), callback("google"));
    const ms = section(t("Microsoft"), t("Microsoft Entra admin center, App registrations: a web platform redirect and a client secret."));
    ms.append(
      h("div", { class: "form-row" }, f.text("sso.microsoft.client_id", t("Application (client) ID")), f.secret("sso.microsoft.client_secret", t("Client secret"))),
      f.text("sso.microsoft.tenant", t("Tenant"), t("common for any account, organizations for work accounts, or your tenant ID to allow only your organisation.")),
      callback("microsoft"),
    );
    const oidc = section(t("Other single sign-on (OpenID Connect)"), t("Any provider that publishes OpenID Connect discovery, such as Okta, Keycloak, Authentik or Zitadel."));
    oidc.append(
      h("div", { class: "form-row" }, f.text("sso.oidc.name", t("Button label"), "", "text", t("Single sign-on")), f.text("sso.oidc.issuer", t("Issuer address"), "", "url", "https://id.example.com")),
      h("div", { class: "form-row" }, f.text("sso.oidc.client_id", t("Client ID")), f.secret("sso.oidc.client_secret", t("Client secret"))),
      callback("oidc"),
    );
    const spField = (label: string, path: string) => {
      const box = h("div", { class: "secret-box" }, h("span"), iconButton("copy", t("Copy address")));
      const paint = () => setText(box.firstElementChild!, `${(store.get("instance.public_url") || location.origin).replace(/\/+$/, "")}${path}`);
      paint();
      store.onChange(paint);
      box.lastElementChild?.addEventListener("click", async () => toast({ message: (await copyText(box.firstElementChild?.textContent ?? "")) ? t("Address copied.") : t("Copying failed."), id: "copy" }));
      return field(label, box);
    };
    const saml = section(t("SAML 2.0"), t("Okta, Microsoft Entra ID, Google Workspace, OneLogin, JumpCloud and any SAML 2.0 identity provider. Register the addresses below, then paste the provider's metadata XML."));
    saml.append(
      h("div", { class: "form-row" }, spField(t("Entity ID and metadata"), "/api/auth/saml/metadata"), spField(t("Assertion consumer service (ACS)"), "/api/auth/saml/acs")),
      f.text("sso.saml.name", t("Button label"), "", "text", t("SAML single sign-on")),
      f.area("sso.saml.metadata", t("Identity provider metadata XML"), t("Includes the sign-on address and the signing certificate."), "<md:EntityDescriptor ...>"),
      h("div", { class: "form-row" }, f.text("sso.saml.email_attr", t("Email attribute"), t("Leave empty to use email, mail or the NameID.")), f.text("sso.saml.name_attr", t("Name attribute"), t("Leave empty to use displayName or the given and family names."))),
      f.bool("auth.saml_only", t("Require SAML sign-in"), t("Turns off passwords, other providers and sign-up for everyone but the instance owner, who keeps a password for emergencies.")),
    );
    f.el.append(s1, saml, google, ms, oidc);
    f.finish();
    panel.append(f.el);
  });
}

function emailTab(engine: Engine, store: SettingsStore): Tab {
  return settingsTab(store, "email", t("Email"), "mail", (f, panel) => {
    const smtp = h("div", { class: "form-grid" });
    const oauth = h("div", { class: "form-grid" });
    const callback = `${(store.get("instance.public_url") || location.origin).replace(/\/+$/, "")}/api/admin/mail/callback`;
    const account = h("p", { class: "section__lead" });
    const connect = h("a", { class: "btn", attrs: { target: "_blank", rel: "noopener", href: "/api/admin/mail/connect?mode=gmail" } }, iconNode("external"), h("span", { text: t("Connect the account") }));
    const paintAccount = () => {
      const who = store.get("mail.oauth.account");
      setText(account, who ? t("Sending as {account}.", { account: who }) : t("No account connected yet. Save the client ID and secret, then connect."));
    };
    const s1 = section(t("Sending mail"), t("Used for invitations and emergency access notices. Without it, invitation links are copied for you to pass on."));
    s1.append(
      f.choice(
        "mail.mode",
        t("Mail"),
        [
          ["off", t("Off")],
          ["smtp", t("SMTP server")],
          ["gmail", t("Gmail (Google Workspace)")],
          ["graph", t("Microsoft 365 (Graph)")],
        ] as const,
        (v) => {
          smtp.hidden = v !== "smtp";
          oauth.hidden = v !== "gmail" && v !== "graph";
          setAttr(connect, "href", `/api/admin/mail/connect?mode=${v}`);
          paintAccount();
        },
      ),
      f.text("mail.from", t("Sender address"), t("For example form.md <notes@example.com>."), "text"),
      smtp,
      oauth,
    );
    oauth.append(
      h("p", { class: "section__lead", text: t("Register an OAuth app with the provider, allow sending mail, and add this redirect address: {url}", { url: callback }) }),
      h("div", { class: "form-row" }, f.text("mail.oauth.client_id", t("Client ID")), f.secret("mail.oauth.client_secret", t("Client secret"))),
      account,
      h("div", { class: "form-actions form-actions--start" }, connect),
    );
    smtp.append(
      h("div", { class: "form-row" }, f.text("mail.smtp.host", t("Host")), f.number("mail.smtp.port", t("Port"))),
      h("div", { class: "form-row" }, f.text("mail.smtp.user", t("User")), f.secret("mail.smtp.password", t("Password"))),
      f.pick(
        "mail.smtp.security",
        t("Encryption"),
        [
          ["starttls", t("STARTTLS (usually port 587)")],
          ["tls", t("TLS from the start (usually port 465)")],
          ["none", t("None (local relays only)")],
        ] as const,
      ),
    );
    f.el.append(s1);
    f.finish();

    const to = input({ type: "email", name: "test-to", placeholder: engine.state.get("me")?.user.email ?? "" });
    const testStatus = status();
    const send = button(t("Send test mail"), "default", "mail");
    send.addEventListener("click", async () => {
      testStatus.clear();
      try {
        await busy(send, () => engine.api.admin.testMail(to.value.trim()));
        testStatus.say(t("Test mail sent to {address}.", { address: to.value.trim() || engine.state.get("me")?.user.email || "" }), "success");
      } catch (err) {
        testStatus.say(err instanceof ApiError && err.code === "mail_failed" ? t("Sending failed: {reason}", { reason: err.message }) : String(err), "error");
      }
    });
    const s2 = section(t("Test"), t("Save first, then send a test message."));
    s2.append(h("div", { class: "form-row form-row--end" }, field(t("Send to"), to), actions(send)), testStatus.el);
    panel.append(f.el, s2);
  });
}

function badgeList(u: AdminUser): HTMLElement[] {
  const out: HTMLElement[] = [];
  if (u.role === "owner") out.push(h("span", { class: "badge badge--accent", text: t("Owner") }));
  else if (u.role === "admin") out.push(h("span", { class: "badge badge--accent", text: t("Admin") }));
  if (u.totp_enabled) out.push(h("span", { class: "badge", text: t("2FA") }));
  if (u.disabled) out.push(h("span", { class: "badge badge--danger", text: t("Deactivated") }));
  return out;
}

function paintBadges(el: HTMLElement, u: AdminUser): void {
  const sig = [u.role, u.totp_enabled, u.disabled].join("|");
  if (el.dataset.sig === sig) return;
  el.dataset.sig = sig;
  el.replaceChildren(...badgeList(u));
}

function usersTab(engine: Engine): Tab {
  return {
    id: "users",
    label: t("Users"),
    icon: "users",
    render(panel) {
      const me = () => engine.state.get("me")!;
      const isOwner = () => me().user.role === "owner";
      let users: AdminUser[] = [];
      let access: AccessWorkspace[] = [];
      let selected = "";

      const search = input({ type: "search", name: "user-search", placeholder: t("Search people"), autocomplete: "off" });
      const create = button(t("New user"), "primary", "plus");
      const list = h("div", { class: "pick-list", attrs: { role: "listbox", "aria-label": t("People") } });
      const none = h("p", { class: "muted", text: t("Nobody found.") });
      none.hidden = true;
      const detail = h("div", { class: "users-detail" });
      const listStatus = status();

      const rowFor = () => {
        const face = avatar("", "violet");
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const badges = h("span", { class: "pick-row__badges" });
        const b = h("button", { class: "pick-row", attrs: { type: "button", role: "option", "aria-selected": "false" } }, face, h("span", { class: "row__main" }, title, meta), badges);
        b.addEventListener("click", () => choose(b.dataset.key ?? ""));
        return b;
      };
      const updateRow = (b: HTMLButtonElement, u: AdminUser) => {
        updateAvatar(b.querySelector<HTMLElement>(".avatar")!, u.name, u.color);
        const [title, meta] = b.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, u.id === me().user.id ? t("{name} (you)", { name: u.name }) : u.name);
        setText(meta, u.email);
        paintBadges(b.querySelector<HTMLElement>(".pick-row__badges")!, u);
        setAttr(b, "aria-selected", u.id === selected ? "true" : "false");
      };
      const paintList = () => {
        const q = search.value.trim().toLowerCase();
        const shown = q ? users.filter((u) => u.name.toLowerCase().includes(q) || u.email.includes(q)) : users;
        reconcile(list, shown, (u) => u.id, rowFor, updateRow);
        none.hidden = shown.length > 0;
      };
      search.addEventListener("input", paintList);

      const load = async () => {
        try {
          const [u, a] = await Promise.all([engine.api.admin.users(), engine.api.admin.access()]);
          users = u.users;
          access = a.workspaces;
          if (!users.some((x) => x.id === selected)) selected = "";
          paintList();
          paintDetail();
        } catch (err) {
          listStatus.error(err);
        }
      };

      const choose = (id: string) => {
        selected = id;
        paintList();
        paintDetail();
        detail.querySelector<HTMLElement>("h3, button")?.scrollIntoView({ block: "nearest" });
      };

      let detailKey = "-";
      let patchDetail: ((u: AdminUser) => void) | null = null;

      const paintDetail = () => {
        const u = users.find((x) => x.id === selected);
        const key = u ? [u.id, u.id === me().user.id, u.role === "owner", isOwner(), isOwner() || u.role === "member"].join("|") : "";
        if (key !== detailKey) {
          detailKey = key;
          if (u) {
            const built = buildDetail(u);
            patchDetail = built.patch;
            detail.replaceChildren(...built.nodes);
          } else {
            patchDetail = null;
            detail.replaceChildren(h("div", { class: "empty-state" }, iconNode("user"), h("p", { text: t("Pick a person to manage their account and workspace access.") })));
          }
        }
        if (u) patchDetail?.(u);
      };

      const buildDetail = (first: AdminUser): { nodes: HTMLElement[]; patch: (u: AdminUser) => void } => {
        let cur = first;
        const st = status();
        const face = avatar(first.name, first.color, true);
        const name = h("h3", { class: "section__title" });
        const email = h("span", { class: "row__meta" });
        const seen = h("span", { class: "row__meta" });
        const badges = h("span", { class: "pick-row__badges" });
        const nodes: HTMLElement[] = [h("div", { class: "detail-head" }, face, h("div", { class: "detail-head__text" }, name, email, seen, badges))];
        const patches: ((u: AdminUser) => void)[] = [
          (u) => {
            updateAvatar(face, u.name, u.color);
            setText(name, u.name);
            setText(email, u.email);
            setText(seen, [t("joined {date}", { date: formatDate(u.created_at) }), u.last_seen_at ? t("active {when}", { when: formatRelative(u.last_seen_at) }) : t("never signed in")].join(" · "));
            paintBadges(badges, u);
          },
        ];
        const patch = (u: AdminUser) => {
          cur = u;
          for (const p of patches) p(u);
        };

        if (first.id === me().user.id) {
          const open = button(t("Open account settings"), "default", "user");
          open.addEventListener("click", () => engine.openOverlay("account"));
          nodes.push(h("div", { class: "callout" }, iconNode("info"), h("div", { class: "form-grid" }, h("p", { text: t("This is you. Change your own account in your account settings.") }), actions(open))));
          return { nodes, patch };
        }
        if (first.role === "owner") {
          const acc = accessSection(() => cur, st);
          patches.push(acc.patch);
          nodes.push(h("div", { class: "callout" }, iconNode("info"), h("p", { text: t("The owner runs this instance. Their account is changed only by themselves.") })), acc.el, st.el);
          return { nodes, patch };
        }

        if (isOwner()) {
          const role = choices(
            `role-${first.id}`,
            [
              ["member", t("Member")],
              ["admin", t("Instance admin (manages everything)")],
            ] as const,
            first.role === "admin" ? "admin" : "member",
          );
          role.el.addEventListener("change", async () => {
            st.clear();
            try {
              await engine.api.admin.updateUser(cur.id, { role: role.value() });
              await load();
            } catch (err) {
              role.set(cur.role === "admin" ? "admin" : "member");
              st.error(err);
            }
          });
          patches.push((u) => role.set(u.role === "admin" ? "admin" : "member"));
          const s = section(t("Instance role"));
          s.append(role.el);
          nodes.push(s);
        }

        if (isOwner() || first.role === "member") {
          const s = section(t("Account"));
          const toggleActive = button(t("Deactivate"), "danger");
          const toggleLabel = toggleActive.querySelector("span")!;
          toggleActive.addEventListener("click", async () => {
            const was = cur;
            if (!was.disabled) {
              const ok = await confirmDialog({
                title: t("Deactivate {name}?", { name: was.name }),
                body: t("They are signed out everywhere and their tokens stop working. Nothing they made is lost, and you can reactivate them later."),
                confirm: t("Deactivate"),
                variant: "danger",
              });
              if (!ok) return;
            }
            try {
              await busy(toggleActive, () => engine.api.admin.updateUser(was.id, { disabled: !was.disabled }));
              toast({ message: was.disabled ? t("{name} is active again.", { name: was.name }) : t("{name} has been deactivated.", { name: was.name }), tone: "success" });
              await load();
            } catch (err) {
              st.error(err);
            }
          });
          const row = h("div", { class: "form-actions form-actions--start" }, toggleActive);
          patches.push((u) => {
            setText(toggleLabel, u.disabled ? t("Reactivate") : t("Deactivate"));
            setAttr(toggleActive, "class", u.disabled ? "btn" : "btn btn--danger");
          });
          if (isOwner()) {
            const reset = button(t("Reset two-factor"), "default", "shield");
            reset.addEventListener("click", async () => {
              const ok = await confirmDialog({
                title: t("Reset two-factor for {name}?", { name: cur.name }),
                body: t("Their authenticator and recovery codes stop working. They can sign in with the password alone and set it up again."),
                confirm: t("Reset"),
                variant: "danger",
              });
              if (!ok) return;
              try {
                await engine.api.admin.resetTOTP(cur.id);
                await load();
              } catch (err) {
                st.error(err);
              }
            });
            const pw = button(t("Set a new password"), "default", "key");
            pw.addEventListener("click", () => setPasswordDialog(cur));
            row.append(reset, pw);
            patches.push((u) => {
              reset.hidden = !u.totp_enabled;
            });
          }
          s.append(row);
          nodes.push(s);
        }

        const acc = accessSection(() => cur, st);
        patches.push(acc.patch);
        nodes.push(acc.el);

        if (isOwner()) {
          const s = section(t("Hand over or delete"));
          const hand = button(t("Hand over the instance"), "default", "building");
          hand.addEventListener("click", () => transferDialog(cur));
          const del = button(t("Delete user"), "danger", "trash");
          del.addEventListener("click", () => void deleteFlow(cur));
          s.append(h("div", { class: "form-actions form-actions--start" }, hand, del));
          patches.push((u) => {
            hand.disabled = u.disabled;
          });
          nodes.push(s);
        }
        nodes.push(st.el);
        return { nodes, patch };
      };

      const accessSection = (who: () => AdminUser, st: ReturnType<typeof status>): { el: HTMLElement; patch: () => void } => {
        const el = section(t("Workspace access"), t("Shared workspaces only. Personal spaces stay private to their owner."));
        const rows = h("div", { class: "rows" });
        const empty = h("p", { class: "muted", text: t("No shared workspaces yet.") });
        el.append(rows, empty);
        const rowFor = () => {
          const title = h("span", { class: "row__title" });
          const meta = h("span", { class: "row__meta" });
          const pick = select<WorkspaceRole | "">([["", t("No access")], ...roleOptions()]);
          const r = h("div", { class: "row" }, iconNode("building"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, pick));
          pick.addEventListener("change", async () => {
            st.clear();
            const before = r.dataset.role ?? "";
            try {
              await engine.api.admin.setAccess(who().id, r.dataset.key ?? "", pick.value as WorkspaceRole | "");
              access = (await engine.api.admin.access()).workspaces;
              patch();
            } catch (err) {
              pick.value = before;
              st.error(err);
            }
          });
          return r;
        };
        const updateRow = (r: HTMLElement, w: AccessWorkspace) => {
          const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
          setText(title, w.name);
          setText(meta, plural(w.members.length, "{count} member", "{count} members"));
          const role = w.members.find((m) => m.user_id === who().id)?.role ?? "";
          r.dataset.role = role;
          const pick = r.querySelector("select")!;
          if (pick.value !== role) pick.value = role;
          setAttr(pick, "aria-label", t("Access to {name}", { name: w.name }));
        };
        const patch = () => {
          reconcile(rows, access, (w) => w.id, rowFor, updateRow);
          empty.hidden = access.length > 0;
        };
        return { el, patch };
      };

      const setPasswordDialog = (u: AdminUser) => {
        simpleDialog(t("New password for {name}", { name: u.name }), (body, close) => {
          const pw = input({ type: "password", name: "new-password", autocomplete: "new-password" });
          const st = status();
          const save = button(t("Set password"), "primary", undefined, "submit");
          const cancel = button(t("Cancel"), "ghost");
          cancel.addEventListener("click", close);
          const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, h("p", { class: "section__lead", text: t("They are signed out everywhere and their tokens stop working. Pass the password on safely.") }), field(t("New password"), pw, t("At least 8 characters.")), st.el, actions(cancel, save));
          form.addEventListener("submit", async (e) => {
            e.preventDefault();
            st.clear();
            try {
              await busy(save, () => engine.api.admin.setPassword(u.id, pw.value));
              close();
              toast({ message: t("Password set for {name}.", { name: u.name }), tone: "success" });
            } catch (err) {
              st.error(err);
            }
          });
          body.append(form);
          return pw;
        });
      };

      const transferDialog = (u: AdminUser) => {
        simpleDialog(t("Hand the instance to {name}?", { name: u.name }), (body, close) => {
          const pw = input({ type: "password", name: "current-password", autocomplete: "current-password" });
          const st = status();
          const go = button(t("Hand over"), "danger", undefined, "submit");
          const cancel = button(t("Cancel"), "ghost");
          cancel.addEventListener("click", close);
          const form = h(
            "form",
            { class: "form-grid", attrs: { novalidate: "" } },
            h("p", { class: "section__lead", text: t("{name} becomes the owner: emergency access, deleting accounts and handing over again. You stay an admin and cannot undo this yourself.", { name: u.name }) }),
            field(t("Your password, to confirm"), pw),
            st.el,
            actions(cancel, go),
          );
          form.addEventListener("submit", async (e) => {
            e.preventDefault();
            st.clear();
            try {
              await busy(go, () => engine.api.admin.transfer(u.id, pw.value));
              close();
              await engine.refreshMe();
              engine.state.set("overlay", null);
              toast({ message: t("{name} is now the owner of this instance.", { name: u.name }), tone: "success" });
            } catch (err) {
              st.error(err);
            }
          });
          body.append(form);
          return pw;
        });
      };

      const deleteFlow = async (u: AdminUser) => {
        let plan: DeletionPlan;
        try {
          plan = await engine.api.admin.deletionPlan(u.id);
        } catch (err) {
          listStatus.error(err);
          return;
        }
        const items: HTMLElement[] = [];
        for (const p of plan.purge) items.push(h("li", { text: t("The personal space {name} is deleted with everything in it.", { name: p.name }) }));
        for (const p of plan.promote) items.push(h("li", { text: t("{user} becomes an admin of {name}.", { user: p.user_name, name: p.name }) }));
        for (const p of plan.hand_to_owner) items.push(h("li", { text: t("{name} is left with nobody in charge, so you take it on.", { name: p.name }) }));
        items.push(h("li", { text: t("Pages they made in shared workspaces stay where they are.") }));
        const body = h("div", { class: "form-grid" }, h("p", { class: "section__lead", text: t("Deactivating is usually enough. Nothing is lost that way. Deleting does this:") }), h("ul", { class: "plan" }, ...items));
        const ok = await confirmDialog({ title: t("Permanently delete {name}?", { name: u.name }), body, confirm: t("Continue"), variant: "danger" });
        if (!ok) return;
        const sure = await confirmTyped({ title: t("Delete {name}", { name: u.name }), body: t("This cannot be undone."), expect: u.email, confirm: t("Delete user") });
        if (!sure) return;
        try {
          await engine.api.admin.deleteUser(u.id);
          toast({ message: t("{name} was deleted.", { name: u.name }), tone: "success" });
          selected = "";
          await load();
          await engine.refreshMe();
        } catch (err) {
          listStatus.error(err);
        }
      };

      create.addEventListener("click", () => {
        simpleDialog(t("New user"), (body, close) => {
          const name = input({ name: "name", autocomplete: "off", maxlength: 80 });
          const email = input({ type: "email", name: "email", autocomplete: "off" });
          const pw = input({ type: "password", name: "new-password", autocomplete: "new-password" });
          const admin = toggle(t("Instance admin"), false, t("May manage everything except the owner."));
          const st = status();
          const go = button(t("Create user"), "primary", undefined, "submit");
          const cancel = button(t("Cancel"), "ghost");
          cancel.addEventListener("click", close);
          const form = h(
            "form",
            { class: "form-grid", attrs: { novalidate: "" } },
            h("p", { class: "section__lead", text: t("Creates the account straight away. No email is sent, so pass the password on yourself.") }),
            field(t("Name"), name),
            field(t("Email"), email),
            field(t("Initial password"), pw, t("At least 8 characters.")),
          );
          if (isOwner()) form.append(admin.el);
          form.append(st.el, actions(cancel, go));
          form.addEventListener("submit", async (e) => {
            e.preventDefault();
            st.clear();
            try {
              const res = await busy(go, () => engine.api.admin.createUser({ name: name.value, email: email.value, password: pw.value, admin: admin.input.checked }));
              close();
              selected = res.id;
              await load();
              toast({ message: t("Account created for {name}.", { name: name.value.trim() }), tone: "success" });
            } catch (err) {
              st.error(err);
            }
          });
          body.append(form);
          return name;
        });
      });

      panel.append(
        h("div", { class: "users-bar" }, h("label", { class: "search-field" }, iconNode("search"), search), create),
        h("div", { class: "users-pane" }, h("div", { class: "users-list" }, list, none), detail),
        listStatus.el,
      );
      void load();
      return () => void load();
    },
  };
}

function workspacesTab(engine: Engine): Tab {
  return {
    id: "workspaces",
    label: t("Workspaces"),
    icon: "building",
    render(panel) {
      const me = () => engine.state.get("me")!;
      const isOwner = () => me().user.role === "owner";
      const st = status();

      const overview = h("div", { class: "rows" });
      const latest = new Map<string, AccessWorkspace>();
      const overviewRow = () => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const open = h("span", { class: "badge", text: t("Open to new accounts") });
        const reading = h("span", { class: "badge badge--danger", text: t("Reading") });
        const look = button(t("Emergency access"), "ghost", "alert");
        const r = h("div", { class: "row" }, iconNode("building"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, open, reading, look));
        look.addEventListener("click", () => {
          const w = latest.get(r.dataset.key ?? "");
          if (w) breakGlassDialog(w);
        });
        return r;
      };
      const updateOverview = (r: HTMLElement, w: AccessWorkspace) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, w.name);
        const admins = w.members.filter((m) => m.role === "admin").length;
        const agent = w.agent_access === "closed" ? t("no agents") : w.agent_access === "strict" ? t("signed-in agents only") : t("agents allowed");
        setText(meta, [plural(w.members.length, "{count} member", "{count} members"), plural(admins, "{count} admin", "{count} admins"), agent].join(" · "));
        latest.set(w.id, w);
        const [open, reading, look] = r.querySelector(".row__actions")!.children as unknown as [HTMLElement, HTMLElement, HTMLElement];
        const member = me().workspaces.some((x) => x.id === w.id && !x.break_glass);
        const isReading = me().workspaces.some((x) => x.id === w.id && x.break_glass);
        open.hidden = !w.auto_join;
        reading.hidden = !(isOwner() && !member && isReading);
        look.hidden = !(isOwner() && !member && !isReading);
      };

      const grants = h("div", { class: "rows" });
      const grantsSection = section(t("Your emergency access"), t("Read only, two hours each, recorded in the activity log and announced to the workspace admins."));
      const grantRow = () => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const end = button(t("End now"), "default");
        const r = h("div", { class: "row" }, iconNode("alert"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, end));
        end.addEventListener("click", async () => {
          try {
            await busy(end, () => engine.api.admin.revokeGrant(r.dataset.key ?? ""));
            await engine.refreshMe();
            void load();
          } catch (err) {
            st.error(err);
          }
        });
        return r;
      };
      const updateGrant = (r: HTMLElement, g: BreakGlassGrant) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, g.workspace_name);
        setText(meta, [g.reason, t("runs until {time}", { time: formatDateTime(g.expires_at) })].join(" · "));
      };
      grantsSection.append(grants);

      const stranded = h("div", { class: "rows" });
      const strandedEmpty = h("p", { class: "muted", text: t("All clear. Every workspace has someone in charge.") });
      const strandedRow = () => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const adopt = button(t("Adopt"), "default");
        const del = iconButton("trash", t("Delete"));
        const r = h("div", { class: "row" }, iconNode("warning"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, adopt, del));
        adopt.addEventListener("click", async () => {
          try {
            await busy(adopt, () => engine.api.admin.adopt(r.dataset.key ?? ""));
            await engine.refreshMe();
            toast({ message: t("Adopted {name}. It is in your list now.", { name: title.textContent ?? "" }), tone: "success" });
            void load();
          } catch (err) {
            st.error(err);
          }
        });
        del.addEventListener("click", async () => {
          const name = title.textContent ?? "";
          const ok = await confirmTyped({ title: t("Delete {name}?", { name }), body: t("Everything in it is deleted permanently."), expect: name, confirm: t("Delete workspace") });
          if (!ok) return;
          try {
            await engine.api.admin.deleteStranded(r.dataset.key ?? "");
            toast({ message: t("{name} was deleted.", { name }), tone: "success" });
            void load();
          } catch (err) {
            st.error(err);
          }
        });
        return r;
      };
      const updateStranded = (r: HTMLElement, w: StrandedWorkspace) => {
        const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, w.name);
        setText(meta, w.members > 0 ? t("Still has members: make one of them an admin, or adopt it.") : t("Nobody is left in it."));
      };
      const strandedSection = section(t("Workspaces with nobody in charge"), t("Left without an active admin, usually after an account was deactivated or deleted."));
      strandedSection.append(stranded, strandedEmpty);

      const breakGlassDialog = (w: AccessWorkspace) => {
        simpleDialog(t("Emergency access to {name}", { name: w.name }), (body, close) => {
          const reason = textarea("", 3);
          reason.placeholder = t("For example: legal hold request 2026-114, approved by the managing director.");
          reason.maxLength = 500;
          const fst = status();
          const go = button(t("Start emergency access"), "danger", undefined, "submit");
          const cancel = button(t("Cancel"), "ghost");
          cancel.addEventListener("click", close);
          const form = h(
            "form",
            { class: "form-grid", attrs: { novalidate: "" } },
            h("p", { class: "section__lead", text: t("You can read this workspace for two hours. The reason is recorded and sent to its admins.") }),
            field(t("Why?"), reason, t("At least 10 characters.")),
            fst.el,
            actions(cancel, go),
          );
          form.addEventListener("submit", async (e) => {
            e.preventDefault();
            fst.clear();
            try {
              const res = await busy(go, () => engine.api.admin.breakGlass(w.id, reason.value));
              close();
              await engine.refreshMe();
              void load();
              toast({
                message: t("Read access to {name} until {time}. Its admins have been told.", { name: w.name, time: formatDateTime(res.expires_at) }),
                tone: "warning",
                action: { label: t("Open"), run: () => engine.state.set("workspaceId", w.id) },
              });
            } catch (err) {
              fst.error(err);
            }
          });
          body.append(form);
          return reason;
        });
      };

      const load = async () => {
        try {
          const a = await engine.api.admin.access();
          reconcile(overview, a.workspaces, (w) => w.id, overviewRow, updateOverview);
          if (isOwner()) {
            const [g, s] = await Promise.all([engine.api.admin.grants(), engine.api.admin.stranded()]);
            reconcile(grants, g.grants, (x) => x.id, grantRow, updateGrant);
            grantsSection.hidden = g.grants.length === 0;
            reconcile(stranded, s.workspaces, (x) => x.id, strandedRow, updateStranded);
            strandedEmpty.hidden = s.workspaces.length > 0;
          }
        } catch (err) {
          st.error(err);
        }
      };

      const all = section(t("Shared workspaces"), isOwner() ? t("Every shared workspace on this instance. Personal spaces are never listed.") : t("Every shared workspace on this instance and who is in charge."));
      all.append(overview);
      panel.append(all, st.el);
      if (isOwner()) panel.append(grantsSection, strandedSection);
      grantsSection.hidden = true;
      void load();
      return () => void load();
    },
  };
}

function aboutTab(engine: Engine): Tab {
  return {
    id: "about",
    label: t("About"),
    icon: "info",
    render(panel) {
      const version = h("span", { class: "row__meta" });
      const storage = h("span", { class: "row__meta" });
      const paint = () => {
        const hl = engine.state.get("health");
        setText(version, hl.latest ? t("{version}, and {latest} is available", { version: hl.version ?? "", latest: hl.latest }) : (hl.version ?? t("unknown")));
        setText(storage, hl.storage ?? t("unknown"));
      };
      paint();
      engine.state.watch("health", paint);
      const licences = h("a", { class: "btn", attrs: { href: "/notices", target: "_blank", rel: "noopener" } }, iconNode("external"), h("span", { text: t("Open-source licences") }));
      const s = section(t("This instance"));
      s.append(
        h(
          "div",
          { class: "rows" },
          h("div", { class: "row" }, iconNode("logo"), h("div", { class: "row__main" }, h("span", { class: "row__title", text: "form.md" }), version), h("span")),
          h("div", { class: "row" }, iconNode("page"), h("div", { class: "row__main" }, h("span", { class: "row__title", text: t("Storage") }), storage), h("span")),
        ),
        h("p", { class: "section__lead", text: t("Everything form.md is built on, with each licence in full.") }),
        h("div", { class: "form-actions form-actions--start" }, licences),
      );
      panel.append(s);
      if (engine.state.get("me")?.user.role === "owner") {
        const download = h("a", { class: "btn", attrs: { href: "/api/admin/backup", download: "" } }, iconNode("download"), h("span", { text: t("Download a backup") }));
        const b = section(t("Backup"), t("The database and every uploaded file in one archive, taken while people keep working. An encrypted instance gives an encrypted archive that only its key restores. Restore it with form-md restore."));
        b.append(h("div", { class: "form-actions form-actions--start" }, download));
        panel.append(b);
      }
      return paint;
    },
  };
}

export function mountAdmin(engine: Engine): void {
  const store = new SettingsStore(engine);
  overlayDialog(engine, "admin", t("Instance settings"), [
    generalTab(engine, store),
    signInTab(store),
    emailTab(engine, store),
    usersTab(engine),
    workspacesTab(engine),
    aiTab(engine),
    aboutTab(engine),
  ]);
}
