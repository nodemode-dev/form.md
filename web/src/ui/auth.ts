import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import type { InviteInfo } from "../engine/types";
import { h, setAttr, setText } from "./dom";
import { errorCode } from "./errors";
import { busy, status } from "./form";
import { iconNode } from "./icons";

const SSO_ERRORS: Record<string, () => string> = {
  denied: () => t("Sign-in was cancelled."),
  expired: () => t("The sign-in took too long. Try again."),
  unavailable: () => t("The sign-in provider could not be reached."),
  exchange: () => t("The sign-in provider refused the request."),
  invalid: () => t("The sign-in response could not be verified."),
  unverified: () => t("An account with this email exists but the email is not verified. Sign in with your password."),
  signup_closed: () => t("There is no account for this email and sign-up is by invitation."),
  disabled: () => t("This account is deactivated."),
  no_email: () => t("The provider did not share an email address."),
  saml_only: () => t("This instance signs people in through SAML single sign-on only."),
  internal: () => t("Something went wrong. Try again."),
};

export const WORKSPACE_COOKIE = "formmd_ws";

export function rememberWorkspace(id: string): void {
  document.cookie = `${WORKSPACE_COOKIE}=${encodeURIComponent(id)}; path=/; max-age=31536000; samesite=lax`;
}

function panel(name: string): HTMLFormElement {
  const el = document.querySelector<HTMLFormElement>(`.auth__panel[data-panel="${name}"]`);
  if (!el) throw new Error(`missing auth panel ${name}`);
  return el;
}

function value(form: HTMLFormElement, name: string): string {
  return (form.elements.namedItem(name) as HTMLInputElement | null)?.value ?? "";
}

function submitButton(form: HTMLFormElement): HTMLButtonElement {
  return form.querySelector<HTMLButtonElement>('button[type="submit"]')!;
}

function nextPath(): string {
  const p = location.pathname;
  const asked = new URLSearchParams(location.search).get("next") ?? "";
  if (asked.startsWith("/") && !asked.startsWith("//") && !asked.startsWith("/\\")) return asked;
  return p === "/login" || p === "/signup" || p.startsWith("/invite/") ? "/" : p + location.search;
}

function go(path: string): void {
  location.assign(path);
}

function bindPasswordToggles(): void {
  for (const pw of document.querySelectorAll<HTMLInputElement>('.auth__panel input[type="password"]')) {
    const shown = h("button", { class: "auth__pw-toggle", attrs: { type: "button", "aria-pressed": "false", "aria-label": t("Show password") } }, iconNode("eye"));
    const wrap = h("span", { class: "auth__pw" });
    pw.before(wrap);
    wrap.append(pw, shown);
    shown.addEventListener("click", () => {
      const on = pw.type === "password";
      pw.type = on ? "text" : "password";
      setAttr(shown, "aria-pressed", on ? "true" : "false");
      shown.replaceChildren(iconNode(on ? "eye-off" : "eye"));
    });
  }
}

function bindSetup(engine: Engine): void {
  const form = panel("setup");
  const st = status(form.querySelector<HTMLElement>(".form-status")!);
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    st.clear();
    try {
      await busy(submitButton(form), () => engine.api.auth.setup(value(form, "name"), value(form, "email"), value(form, "password")));
      go("/");
    } catch (err) {
      st.error(err);
    }
  });
}

function bindLogin(engine: Engine): void {
  const form = panel("login");
  const st = status(form.querySelector<HTMLElement>(".form-status")!);
  const totpField = form.querySelector<HTMLElement>('[data-ref="totp-field"]')!;
  const recoveryField = form.querySelector<HTMLElement>('[data-ref="recovery-field"]')!;
  const toggle = form.querySelector<HTMLButtonElement>('[data-ref="toggle-recovery"]')!;
  const params = new URLSearchParams(location.search);
  const ssoError = params.get("sso_error");
  if (ssoError) st.say((SSO_ERRORS[ssoError] ?? SSO_ERRORS.internal!)(), "error");
  const next = nextPath();
  for (const a of form.querySelectorAll<HTMLAnchorElement>(".auth__sso-btn")) a.href = `${a.getAttribute("href")}?next=${encodeURIComponent(next)}`;
  const oidc = engine.boot.sso?.find((p) => p.id === "oidc");
  if (oidc) setText(form.querySelector('[data-ref="oidc-btn"]')!, t("Continue with {name}", { name: oidc.name }));
  form.querySelector('[data-ref="owner-password"]')?.addEventListener("click", () => {
    document.documentElement.dataset.samlOnly = "false";
    (form.elements.namedItem("email") as HTMLInputElement).focus();
  });

  let useRecovery = false;
  toggle.addEventListener("click", () => {
    useRecovery = !useRecovery;
    totpField.hidden = useRecovery;
    recoveryField.hidden = !useRecovery;
    setText(toggle, useRecovery ? t("Use the authenticator code instead") : t("Use a recovery code instead"));
    (form.elements.namedItem(useRecovery ? "recovery" : "code") as HTMLInputElement).focus();
  });

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    st.clear();
    const totpStep = form.dataset.step === "totp";
    const code = totpStep && !useRecovery ? value(form, "code") : "";
    const recovery = totpStep && useRecovery ? value(form, "recovery") : "";
    try {
      await busy(submitButton(form), () => engine.api.auth.login(value(form, "email"), value(form, "password"), code, recovery));
      go(next);
    } catch (err) {
      if (errorCode(err) === "totp_required") {
        form.dataset.step = "totp";
        (form.elements.namedItem("code") as HTMLInputElement).focus();
        return;
      }
      if (errorCode(err) === "bad_credentials") {
        st.say(t("That email or password isn't right. Try again, or ask your workspace admin to reset it."), "error");
        return;
      }
      st.error(err);
    }
  });
}

function bindSignup(engine: Engine): void {
  const form = panel("signup");
  const st = status(form.querySelector<HTMLElement>(".form-status")!);
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    st.clear();
    try {
      await busy(submitButton(form), () => engine.api.auth.signup(value(form, "name"), value(form, "email"), value(form, "password")));
      go("/");
    } catch (err) {
      st.error(err);
    }
  });
}

const ROLE_LABEL: Record<string, () => string> = {
  admin: () => t("admin"),
  member: () => t("member"),
  viewer: () => t("viewer"),
};

function bindInvite(engine: Engine): void {
  const form = panel("invite");
  const st = status(form.querySelector<HTMLElement>(".form-status")!);
  const token = location.pathname.split("/")[2] ?? "";
  const title = form.querySelector<HTMLElement>('[data-ref="invite-title"]')!;
  const lead = form.querySelector<HTMLElement>('[data-ref="invite-lead"]')!;
  const emailField = form.querySelector<HTMLElement>('[data-ref="invite-email-field"]')!;
  const codeField = form.querySelector<HTMLElement>('[data-ref="invite-code-field"]')!;
  let info: InviteInfo | null = null;

  engine.api.auth
    .invite(token)
    .then((i) => {
      info = i;
      if (i.page) {
        setText(title, t("Open {title}", { title: i.page.title || t("Untitled") }));
        setText(lead, t("{inviter} shared a page in {workspace} with you. You join as a guest who sees only what is shared.", { inviter: i.inviter || t("Someone"), workspace: i.workspace.name }));
      } else {
        setText(title, t("Join {workspace}", { workspace: i.workspace.name }));
        setText(lead, t("{inviter} invited you as {role}.", { inviter: i.inviter || t("Someone"), role: ROLE_LABEL[i.role]?.() ?? i.role }));
      }
      emailField.hidden = i.email !== "";
      if (i.signed_in_as) {
        setText(form.querySelector('[data-ref="invite-signed-in"]')!, t("You are signed in as {email}.", { email: i.signed_in_as }));
        form.dataset.mode = "signed-in";
      } else {
        form.dataset.mode = i.account_exists ? "existing" : "new";
      }
    })
    .catch((err) => {
      form.dataset.mode = "invalid";
      setText(title, t("Invitation unavailable"));
      setText(lead, "");
      st.error(err);
    });

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    if (!info) return;
    st.clear();
    const mode = form.dataset.mode;
    const body =
      mode === "existing"
        ? { email: info.email, password: value(form, "existing-password"), code: value(form, "existing-code") }
        : mode === "new"
          ? { name: value(form, "name"), email: info.email || value(form, "email"), password: value(form, "password") }
          : {};
    try {
      const res = await busy(submitButton(form), () => engine.api.auth.acceptInvite(token, body));
      rememberWorkspace(res.workspace_id);
      go(res.page_id ? `/p/${res.page_id}` : "/");
    } catch (err) {
      if (errorCode(err) === "totp_required") {
        codeField.hidden = false;
        (form.elements.namedItem("existing-code") as HTMLInputElement).focus();
        return;
      }
      st.error(err);
    }
  });
}

function bindConsent(engine: Engine, request: string): void {
  const form = panel("consent");
  const st = status(form.querySelector<HTMLElement>(".form-status")!);
  const deny = form.querySelector<HTMLButtonElement>('[data-ref="consent-deny"]')!;
  const send = async (approve: boolean, button: HTMLButtonElement) => {
    st.clear();
    const scope = (form.querySelector<HTMLInputElement>('input[name="scope"]:checked')?.value ?? "read") as "read" | "write";
    const workspaces = [...form.querySelectorAll<HTMLInputElement>('input[name="workspace"]:checked')].map((i) => i.value);
    if (approve && workspaces.length === 0) {
      st.error(new Error(t("Choose at least one workspace.")));
      return;
    }
    try {
      const res = await busy(button, () => engine.api.me.consent(request, approve, scope, workspaces));
      setAttr(form, "data-mode", "done");
      location.assign(res.redirect);
    } catch (err) {
      st.error(err);
    }
  };
  form.addEventListener("submit", (e) => {
    e.preventDefault();
    void send(true, submitButton(form));
  });
  deny.addEventListener("click", () => void send(false, deny));
}

export function mountAuth(engine: Engine): void {
  bindPasswordToggles();
  bindSetup(engine);
  bindLogin(engine);
  bindSignup(engine);
  if (engine.state.get("auth") === "invite") bindInvite(engine);
  const oauth = engine.boot.oauth;
  if (engine.state.get("auth") === "consent" && oauth) bindConsent(engine, oauth.request);
  engine.state.watch("auth", (v) => setAttr(document.documentElement, "data-auth", v), { immediate: true });
  const first = document.querySelector<HTMLInputElement>(`:root[data-auth="${engine.state.get("auth")}"] .auth__panel input:not([hidden] *)`);
  if (first && engine.state.get("auth") !== "app") {
    const visible = [...document.querySelectorAll<HTMLInputElement>(".auth__panel input")].find((i) => i.offsetParent !== null);
    (visible ?? first).focus({ preventScroll: true });
  }
}
