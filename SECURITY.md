# Security policy

## Supported versions

Security fixes go into the newest release. Please update before reporting, and
check whether the problem is still there.

| Version | Supported |
| --- | --- |
| 0.47.x (latest release) | Yes |
| Anything older | No |

## Reporting a vulnerability

**Do not open a public issue for a security problem.** Use GitHub's private
reporting instead: on the repository's **Security** tab, choose **Report a
vulnerability**. Include what you found, the steps to reproduce it, the version and
platform, and what an attacker could do with it.

You can expect an acknowledgement within a week. We will keep you informed while it
is fixed and credit you in the release notes, unless you would rather not be named.

## What counts

Form.md is a server that people sign in to, and it can be put on the internet, so the
things that matter most are the ways someone could reach what they should not:

- **Accounts and sign-in**: getting past passwords, TOTP, OpenID Connect or SAML,
  or taking over a session, an invitation, an API token or an OAuth token.
- **Access levels**: reading or changing a page, database, comment or file beyond
  what sharing grants, on any route, including the sync socket, search and exports.
- **Agents**: an MCP or API credential doing more than its scope and workspace allow,
  or reaching the terminal and Claude Code routes, which are for the signed-in owner
  only.
- **Public pages and forms**: a published site, share link or public form exposing
  anything beyond what was published, or getting past its password.
- **Data at rest**: reading the encrypted database or uploads without the key, or
  anything that leaks sealed secrets such as AI provider keys or mail credentials.
- **Imports, uploads and link previews**: writing outside the data directory, or making
  the server fetch an address it should not.
- **Webhooks**: forging or replaying a signed webhook.

Bugs that need someone to already control the server's machine, or to run a modified
build of Form.md, are ordinary bugs: please open an issue for those.
