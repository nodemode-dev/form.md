# form.md Wiki

form.md is a self-hosted workspace where people and AI agents write documents, run databases and edit together in realtime.

This wiki is for the people who install and run an instance and for the people who change its code.

## At a glance

| | |
|---|---|
| **Current version** | v0.61.1 · every version is on [[Releases]] |
| **What it is** | One static Go binary (`form-md`) with the TypeScript web client embedded |
| **Where your data lives** | One data directory: a SQLite database (`form-md.db`), the uploaded files and a few small key files. There is no separate database server, cache or object store |
| **People and agents together** | The server holds the collaborative document itself, so an agent writing through MCP or the REST API merges with what a person is typing |
| **Stack** | Go 1.26, pure-Go SQLite, vanilla TypeScript and Vite, ProseMirror and Yjs |
| **Team** | one owner (nodemode-ai) |
| **Releases** | Built by formcompiler; the update check and the installer read its form.md-updates release |

## Where to go

### Get it running

- Put an instance on a server: [[Install]]
- Your own copy on your own domain for a small team, start to finish: [[Self-hosting]]
- Every environment variable and the settings they seed: [[Configuration]]

### Keep it safe and reachable

- Encrypt the data directory at rest: [[Encryption]]
- HTTPS with your own certificate, built-in ACME, or a reverse proxy: [[TLS-and-ACME]]
- Reach the server through Cloudflare without opening a port: [[Tunnel]]
- Sign people in through Auth0: [[Auth0]]
- Sending mail through SMTP, Gmail or Microsoft 365: [[Mail]]
- Back up and restore: [[Backup-and-restore]]
- Health checks, the data directory, logs, updates, memory, fail2ban and upgrades: [[Operations]]

### Use it

- Charts, heatmaps, the graph view, dashboards and boards: [[Data-views]]
- What works without a connection, and the installable app: [[Offline]]
- REST API, tokens, OAuth, rate limits and connecting agents: [[API-and-agents]]
- Context packs and the knowledge bank for agents: [[Context-and-knowledge]]
- The terminal and Claude Code inside the app: [[Terminal-and-Claude-Code]]
- The other Form apps on the same computer: [[Form-family]]
- Extensions: [[Extensions]]

### Change it

- Repo layout, dev loop and conventions: [[Development]]
- Branches, commits and releases: `BRANCHING.md` at the repo root
- What each version brought: [[Releases]]

## What's new

- **v0.61.1** lets an account that has no password, such as the desktop app's owner, set its first one without being asked for a current one, and adds [[Self-hosting]], [[Auth0]] and the `deploy/gcp` scripts.
- **v0.61.0** brought a desktop app whose window signs in with a key the server makes at each start and accepts only once, so a sign-in from outside the app's window is refused, and third-party notices that a build writes exactly as they are kept, the demo media and the extension kit included.

Everything before that, version by version: [[Releases]].

## Status at a glance

How to read the State column:

| Word | Meaning |
|---|---|
| Stable | Built, in use, and not expected to change shape |
| Working | Built and checked the way the row says |
| **in testing** | Works against a stand-in or a test service, not yet against the real one |
| **not yet run**, **not yet tried** | Written, and not exercised where the row says |

### The core

| Area | State |
|---|---|
| Server, storage, accounts, workspaces, editor, databases, search, sharing | Stable, built through Phases 1 to 10 |
| REST API with `/api/docs` and OAuth, MCP endpoint | Stable · see [[API-and-agents]] |
| Interface and mail in 26 languages | Since v0.14.0; es-MX and fr-CA want a native read |
| Automated tests | None in the tree by design; CI runs format, vet, type check, a build and a smoke run · see [[Development]] |

### Writing and pages

| Area | State |
|---|---|
| Page styles with a workspace default, focus mode, word counts | Working since v0.15.0; checked in Chrome under CPU and network throttling |
| Image annotations, per-block alignment | Working since v0.16.0; checked in Chrome under CPU and network throttling |
| Custom workspace fonts, typewriter scrolling | Working since v0.17.0; checked in Chrome under CPU and network throttling |
| Board block, five creative embed providers, pictures on fresh public pages | Working since v0.18.0; checked in Chrome under CPU and network throttling |
| Phone block toolbar, renaming and removing uploaded fonts, the person_editing guard for agent replaces | Working since v0.21.0; checked in Chrome under CPU throttling · see [[API-and-agents]] |
| Picking, moving, recolouring and deleting single annotation marks, uploaded fonts on published pages, timeline windowing | Working since v0.23.0; checked in Chrome under CPU throttling |
| Annotation marks in Markdown and PDF exports, uploaded fonts in HTML exports, the list view loading as you scroll | Working since v0.24.0; checked in Chrome and by rasterizing the PDF |
| Saved styles, offline row deletes, faster bulk row updates (template lookup) | Working since v0.25.0; checked in Chrome and timed against v0.24.0 |

### Databases and views

| Area | State |
|---|---|
| Charts, heatmaps, the graph view, KPI dashboards and kanban upgrades | Working since v0.12.0 · see [[Data-views]] |
| Bulk row tools for agents about twice as fast again (six times faster than v0.24.0 for updates) | Since v0.27.0; timed against v0.26.0 · see [[Data-views]] |

### Offline

| Area | State |
|---|---|
| Recent pages kept offline, 1000-row offline databases, phone-width top bar and code blocks, described window commands | Working since v0.19.0; checked in Chrome · see [[Offline]] and [[Data-views]] |
| Offline cell edits kept and sent later, page styles on public pages and HTML exports, the slash setting in account settings | Working since v0.22.0; checked in Chrome · see [[Offline]] |
| Rows added offline, with client-made ids the server accepts | Working since v0.26.0; sent after a server restart without a reload since v0.38.0, checked with the server process stopped and restarted · see [[Offline]] |
| Board, gallery, calendar, timeline, map, graph, chart, dashboard and feed views of kept, recent and opened databases offline; heatmaps as last opened | Working since v0.37.0, checked with the server process stopped; the v0.32.0 and v0.33.0 versions did not work with the server really down · see [[Offline]] |
| Offline start from the service worker, installable app | Working, **in testing**; the installed app not yet tried on a phone · see [[Offline]] |

### Agents and the terminal

| Area | State |
|---|---|
| Context packs, the knowledge bank, `form-md mcp`, Claude Code, Codex and Cursor connections | Working since v0.13.0; the client configs are checked by parsing, not yet inside each app · see [[Context-and-knowledge]] |
| Terminal, Claude Code and its chat in the app, window commands for agents, `form-md cli` | Working since v0.14.0; checked with a real Claude Code turn, not yet under browser throttling · see [[Terminal-and-Claude-Code]] |
| Terminal and Claude Code panel checked under throttling: correct terminal cells, no empty first open, restart in place | Since v0.31.0; checked in Chrome at CPU 4x and 500 kbps · see [[Terminal-and-Claude-Code]] |
| Per-credential revoke: a signed-out session, deleted token or revoked OAuth grant loses its streams, sockets and terminals at once | Working since v0.30.0; checked in Chrome and over OAuth, and against v0.29.0 · see [[API-and-agents]] |
| Form family descriptor, `family_list` and `family_route` for formlit and formvis | Since v0.28.0; checked against stubs of the two apps' endpoints, not the apps themselves · see [[Form-family]] |
| Form family note in context packs, `family-routing` sub-skill, family numbers in `policy.json` | Since v0.29.0; checked against stubs of the two apps' endpoints · see [[Form-family]] |
| AI providers | Verified against a local stand-in, not yet against live provider keys |

### Running an instance

| Area | State |
|---|---|
| At-rest encryption, `form-md encrypt` | Working since v0.11.0, independently audited · see [[Encryption]] |
| `backup`, `restore`, `doctor`, `admin reset-owner` | Working since v0.11.0 · see [[Backup-and-restore]] and [[Operations]] |
| Automatic HTTPS from Instance settings, browser backup download, update check | Working since v0.11.0 · see [[TLS-and-ACME]], [[Backup-and-restore]], [[Operations]] |
| The first password of an account that has none (the desktop owner, accounts made by single sign-on) | Working since v0.61.1; checked against the built server, the Profile tab not yet looked at in a browser · see [[Install]] |
| Built-in ACME certificates | Working, **in testing**: checked against a test CA (Pebble), not yet against Let's Encrypt production · see [[TLS-and-ACME]] |
| Cloudflare tunnel | Working, **in testing** against a stand-in, not yet against real Cloudflare · see [[Tunnel]] |
| Gmail and Microsoft 365 sending | Working, **in testing** against a stub provider, not yet against Google or Microsoft · see [[Mail]] |
| `install.sh`, systemd unit, container image, releases through formcompiler | Written and tried locally (the image built and answered `/healthz`), **not yet exercised by a formcompiler release** · see [[Install]] |
| SAML and SCIM | Tested against a local identity provider only, not yet against Okta, Entra or Google Workspace |
| Hosting on Google Cloud with `deploy/gcp`, Auth0 through OpenID Connect | The scripts are syntax-checked, **not yet run** on a new project; Auth0 **not yet tried** against a tenant · see [[Self-hosting]] and [[Auth0]] |
| Windows scripts (`*.bat`) | Written, **not yet run** on a Windows host |

## Editing this wiki

The source of truth for this wiki is the `.docs/wiki/` folder in the main repository. Edit pages there on a `docs/*` branch, merge the pull request into `main`, then publish:

```sh
git clone https://github.com/nodemode-dev/form.md.wiki.git
cp form.md/.docs/wiki/*.md form.md.wiki/
cd form.md.wiki
git add -A
git commit -m "docs: sync wiki from .docs/wiki"
git push
```

Direct edits in the GitHub wiki interface are overwritten by the next sync, so keep every change in the repository.

Changing a subsystem's behaviour (a command, an environment variable, a route, a file in the data directory) means updating its page in the same pull request. A page that lags the code is worse than no page, because it is trusted.

Formatting rules for all pages: no emojis anywhere, no em dashes (use an interpunct `·` or restructure the sentence), plain markdown, real paths and real commands rather than abstract descriptions.
