# Mail

form.md sends mail for invitations, password resets, emergency access notices, inbox notices that stay unread and automation actions. It can send through an SMTP server, through Gmail (Google Workspace) or through Microsoft 365 (Microsoft Graph). Mail is off until an admin sets it up in **Instance settings, Email**; without it, invitation links are copied for you to pass on.

## Layout

| Path | What lives there |
|---|---|
| `internal/mail/mail.go` | The mailer: builds messages and hands them to the sender for the chosen mode |
| `internal/mail/oauth.go` | Gmail and Graph: the OAuth connect flow, token refresh and sending |
| `internal/admin/admin.go` | `POST /api/admin/mail/test`, `GET /api/admin/mail/connect`, `GET /api/admin/mail/callback` |
| `internal/settings/settings.go` | The `mail.*` settings |
| `web/src/ui/admin.ts` | Instance settings, Email |

## Settings

| Field | Setting | Notes |
|---|---|---|
| Mode | `mail.mode` | `off` (default), `smtp`, `gmail`, `graph` |
| Sender address | `mail.from` | For example `form.md <notes@example.com>`. Nothing is sent without it |
| Host, Port, User, Password, Security | `mail.smtp.host`, `mail.smtp.port` (default 587), `mail.smtp.user`, `mail.smtp.password` (secret), `mail.smtp.security` (`starttls` default, `tls`, `none`) | SMTP only |
| Client ID, Client secret | `mail.oauth.client_id`, `mail.oauth.client_secret` (secret) | Gmail and Graph |
| (set by connecting) | `mail.oauth.refresh_token` (secret), `mail.oauth.account` | Gmail and Graph |

Secret settings are sealed with `secret.key` in the data directory. Mail counts as set up when `mail.from` is set and, for SMTP, a host is set, or for Gmail and Graph, an account has been connected.

## SMTP

Choose **SMTP server**, fill in the sender address and the server, save, and send a test mail from the same tab.

## Gmail and Microsoft 365

1. Register an OAuth app with the provider and allow it to send mail. The scopes form.md asks for are:
   - Gmail: `openid email https://www.googleapis.com/auth/gmail.send`
   - Graph: `openid email offline_access https://graph.microsoft.com/Mail.Send`
2. Add the redirect address shown on the Email tab. It is the **Public address** (or the address you opened the settings at, when none is set) followed by `/api/admin/mail/callback`, for example `https://notes.example.com/api/admin/mail/callback`. Set the Public address first if people reach the instance by another name.
3. In Instance settings, Email, choose **Gmail (Google Workspace)** or **Microsoft 365 (Graph)**, enter the sender address, client ID and client secret, and save.
4. Click **Connect the account**. A new tab goes to `GET /api/admin/mail/connect?mode=gmail` (or `graph`), which sends you to the provider's sign-in. Sign in as the mailbox that should send.
5. The provider sends you back to `/api/admin/mail/callback`, which answers "Mail is now sent as <account>." Close the tab and send a test mail.

Connecting needs a signed-in admin in a browser; tokens cannot start it. It uses PKCE, and a started connection expires after 15 minutes ("this sign-in expired; start again from Instance settings"). Gmail is asked for offline access with a consent prompt; Graph signs in through Microsoft's `common` endpoint.

On success the server saves the mode, the refresh token (sealed) and the account address taken from the sign-in. Access tokens are kept in memory and refreshed with the refresh token when they are within a minute of expiring; when the provider hands out a new refresh token, the new one is saved. If the provider gives no refresh token, the connection fails with `the provider gave no refresh token; remove the app's access in your account and connect again`.

## Checking it

- **Send a test mail** on the Email tab (`POST /api/admin/mail/test`), to yourself by default.
- `form-md doctor` warns when mail is off and when the mode is SMTP without a host; see [[Operations]].
- A provider that refuses a message shows up as `gmail answered <status>: ...` or `graph answered <status>: ...`.

## The contract

1. A new sending mode registers a sender with the mailer (`Mailer.Register`), is added to `mail.mode`'s options, and says in `Mailer.Enabled` what makes it usable.
2. Every credential a mode needs is a `Secret` setting; refresh tokens are saved again whenever the provider rotates them.
3. The connect and callback routes stay admin-only and session-only.

## Related

[[Configuration]] · [[Operations]] · [[TLS-and-ACME]]
