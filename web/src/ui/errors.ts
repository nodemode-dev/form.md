import { ApiError } from "../engine/api";
import { t } from "../engine/i18n";

const MESSAGES: Record<string, () => string> = {
  bad_credentials: () => t("Email or password is wrong."),
  totp_required: () => t("Enter the code from your authenticator app."),
  bad_totp: () => t("That code did not work."),
  account_disabled: () => t("This account is deactivated."),
  rate_limited: () => t("Too many attempts. Wait a minute and try again."),
  weak_password: () => t("Use at least 8 characters."),
  bad_email: () => t("Enter a valid email address."),
  bad_name: () => t("Enter a name of up to 80 characters."),
  email_taken: () => t("An account with this email already exists."),
  signup_closed: () => t("Sign-up is by invitation on this instance."),
  already_set_up: () => t("This instance already has an owner."),
  invite_invalid: () => t("This invitation does not exist or was already used."),
  invite_used: () => t("This invitation was already used."),
  invite_expired: () => t("This invitation has expired."),
  invite_other_email: () => t("This invitation is for a different email address."),
  bad_password: () => t("The password is wrong."),
  last_admin: () => t("A workspace needs at least one admin."),
  personal_space: () => t("Personal spaces cannot be shared."),
  no_such_user: () => t("No active account uses that email. Send an invitation instead."),
  already_member: () => t("That person is already a member."),
  no_workspaces: () => t("Choose at least one workspace you belong to, or all workspaces."),
  session_only: () => t("This needs a signed-in browser session."),
  forbidden: () => t("You are not allowed to do that."),
  owner_only: () => t("Only the owner can do that."),
  not_found: () => t("Not found."),
  network: () => t("The server cannot be reached."),
  timeout: () => t("The server took too long to answer."),
  rules_too_long: () => t("Rules can be up to 16,000 characters."),
  reason_required: () => t("Give a reason of at least 10 characters."),
  totp_active: () => t("Two-factor sign-in is already on."),
  mail_failed: () => t("The test message could not be sent."),
  create_closed: () => t("Only admins create workspaces on this instance."),
  self: () => t("Use your own account settings for this."),
  owner_protected: () => t("The owner account can only be changed by its owner."),
  not_stranded: () => t("This workspace has an admin again."),
  credential_narrowed: () => t("This credential is limited to chosen workspaces."),
  bad_setting: () => t("One of the values is not valid."),
  bad_prefs: () => t("One of the values is not supported."),
  bad_time_zone: () => t("Unknown time zone."),
  bad_color: () => t("Unknown colour."),
  bad_role: () => t("Unknown role."),
  bad_scope: () => t("Choose read only or read and write."),
  admins_only: () => t("Only an admin of this workspace can do that."),
  bad_origin: () => t("The request came from another site and was refused."),
  read_only: () => t("This credential can only read."),
  unauthorized: () => t("Your session has ended. Sign in again."),
};

export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    const known = MESSAGES[e.code];
    if (known) return known();
    return e.message ? e.message.charAt(0).toUpperCase() + e.message.slice(1) + "." : t("Something went wrong.");
  }
  return t("Something went wrong.");
}

export function errorCode(e: unknown): string {
  return e instanceof ApiError ? e.code : "";
}
