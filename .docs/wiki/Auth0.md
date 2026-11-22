# Auth0

How to let people sign in to a hosted form.md through Auth0. form.md needs no Auth0-specific code for this: Auth0 is an OpenID Connect provider, and form.md already signs people in through any provider that publishes OpenID Connect discovery (Instance settings, Sign-in, **Other single sign-on (OpenID Connect)**).

**Status: not yet tried against a real Auth0 tenant.** Everything below was worked out from form.md's sign-in code and Auth0's documentation, and the one point that could not be settled from either is called out under Open points. Treat the first sign-in as the test.

## Layout

| Path | What lives there |
|---|---|
| `internal/auth/oidc.go` | Discovery, the authorization code flow with PKCE, ID token checks, matching or creating the account |
| `internal/settings/settings.go` | `sso.oidc.name`, `sso.oidc.issuer`, `sso.oidc.client_id`, `sso.oidc.client_secret` |
| `web/src/ui/admin.ts` | Instance settings, Sign-in tab (`signInTab`) |

## What form.md does at sign-in

1. Reads `<issuer>/.well-known/openid-configuration` and the signing keys it points to, and keeps them for an hour.
2. Sends the browser to the provider's authorization endpoint with `response_type=code`, the scopes `openid email profile`, a `state`, a `nonce`, a PKCE challenge (`S256`) and `prompt=select_account`.
3. Exchanges the code at the token endpoint, sending the client secret in the form body.
4. Checks the ID token: signature (RS256 or ES256 only), issuer equal to the one in the discovery document, audience equal to the client ID, expiry, nonce.
5. Finds the account: first by the provider's subject, if this person has signed in this way before; otherwise by email. An existing account is linked only when both sides say the email is verified. A new account is created only when registration allows that address.

## In Auth0

1. Create an application of type **Regular Web Application**.
2. In its settings, add the redirect address to **Allowed Callback URLs**. form.md shows the exact address in Instance settings, Sign-in, under the OpenID Connect fields: the public address followed by the callback path, for example `https://notes.example.com/api/auth/sso/oidc/callback`.
3. Under Advanced Settings, OAuth, make sure the JSON Web Token signature algorithm is **RS256**. form.md refuses tokens signed with HS256.
4. Keep the application's authentication method as a client secret sent by POST: form.md sends the secret in the body of the token request, not in a header.
5. Note the **Domain**, the **Client ID** and the **Client Secret**.
6. Decide who can get an Auth0 account. For a closed team, turn on **Disable Sign Ups** on the database connection and create the users yourself, or allow only the connections you want (for example your company's Google Workspace).

## In form.md

Set **Public address** first (Instance settings, General), because the redirect address is built from it. Then in Instance settings, Sign-in, **Other single sign-on (OpenID Connect)**:

| Field | Setting | Value |
|---|---|---|
| Button label | `sso.oidc.name` | `Auth0`, or your company's name. The sign-in page then reads "Continue with Auth0" |
| Issuer address | `sso.oidc.issuer` | `https://` followed by the Auth0 **Domain**, for example `https://your-tenant.us.auth0.com` |
| Client ID | `sso.oidc.client_id` | The application's Client ID |
| Client secret | `sso.oidc.client_secret` | The application's Client Secret. It is sealed in the database |

A trailing slash on the issuer address does no harm: form.md removes it to find the discovery document, and then checks tokens against the issuer that document states, which for Auth0 ends in a slash.

## Who gets in

Auth0 proves who someone is; form.md still decides whether they get an account.

- **By invitation only** (the default): a person with no form.md account is turned away with "There is no account for this email and sign-up is by invitation." Invite them first; once they have accepted, they can use the Auth0 button from then on, as long as Auth0 reports their email as verified.
- **Email domain allowed**: anyone Auth0 signs in with an address at a listed domain gets an account on first sign-in. This is the usual choice for a company.
- **Open to anyone**: everyone Auth0 lets through gets an account. Only sensible when Auth0 itself is closed (step 6 above).

The owner made at the set-up screen keeps their password. Signing in through Auth0 with the same, verified email links the two.

## What it does not do yet

- **Passwords stay on.** Adding Auth0 adds a button; it does not remove the email and password form. Only SAML has an "only this way" switch (`auth.saml_only`). Accounts created through Auth0 have no password, so for them the form is a dead end, but an invited account that set a password can still use it. If you need the identity provider to be the only door, the SAML 2.0 settings on the same tab are the route that has that switch today; otherwise keep password accounts to the owner. A password-off switch for OpenID Connect is the missing piece on the form.md side.
- **No sign-out at Auth0.** Signing out of form.md ends the form.md session only.
- **Roles and groups are not read from Auth0.** Workspace roles are set in form.md. SCIM is the route for provisioning.
- **Two-factor is separate.** form.md's own two-factor sign-in is asked for at password sign-in only; a sign-in through a provider is not asked for the code. For people who come through Auth0, turn on multi-factor authentication in Auth0.

## Open points

- `prompt=select_account` is sent to every OpenID Connect provider. Auth0's documentation describes `none` and `login` for this parameter and does not say what it does with `select_account`. If the first sign-in ends in an error page at Auth0 that names the prompt, that is the cause, and the fix is on the form.md side.
- The email claim: form.md needs `email` in the ID token and reads `email_verified` beside it. Both come with the `email` scope from a standard Auth0 connection; a custom connection that leaves them out ends in "The provider did not share an email address."

## If it fails

The sign-in page shows a short reason, and the server log has the detail (`sudo journalctl -u form-md -n 50`, lines with `sso`):

| Message on the sign-in page | Usual cause |
|---|---|
| The sign-in provider could not be reached | Wrong issuer address, or the server cannot reach Auth0 |
| The sign-in provider refused the request | Wrong client secret, or the redirect address is not in Allowed Callback URLs |
| The sign-in response could not be verified | Signature algorithm is not RS256, or the Client ID does not match |
| An account with this email exists but the email is not verified | Verify the email in Auth0, or in form.md, then try again |
| There is no account for this email and sign-up is by invitation | Invite the person, or allow their email domain |

## Related

[[Self-hosting]] · [[Configuration]] · [[TLS-and-ACME]] · [[Operations]]
