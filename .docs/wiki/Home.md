# form.md Wiki

form.md is a self-hosted workspace where people and AI agents write documents, run databases and edit together in realtime. It ships as one static Go binary (`form-md`) with the TypeScript web client embedded, and it keeps everything in one data directory: a SQLite database (`form-md.db`), the uploaded files and a few small key files. There is no separate database server, cache or object store. The server holds the collaborative document itself, so an agent writing through MCP or the REST API merges with what a person is typing. This wiki is for the people who install and run an instance and for the people who change its code.

**Current version:** v0.20.0 (Phase 16 audit fixes: pictures from agents imported on the server, block alignment on public pages, a server-side default for command safety; v0.19.0 brought offline by default with larger databases, phone-width fixes, described window commands; v0.18.0 brought the board block and creative embeds; v0.17.0 brought custom fonts and typewriter scrolling; v0.16.0 brought image annotations and block alignment; v0.15.0 brought page styles, focus mode and word counts; v0.14.0 brought the terminal, Claude Code and its chat inside the app, window commands for agents, 26 languages; v0.13.0 brought context packs and agent connections, v0.12.0 the visual data views, v0.11.0 encryption, ACME, operations commands, offline start and the installer) · **Stack:** Go 1.26, pure-Go SQLite, vanilla TypeScript and Vite, ProseMirror and Yjs · **Team:** one owner (nodemode-ai)

## Status at a glance

| Area | State |
|---|---|
| Server, storage, accounts, workspaces, editor, databases, search, sharing | Stable, built through Phases 1 to 10 |
| REST API with `/api/docs` and OAuth, MCP endpoint | Stable · see [[API-and-agents]] |
| Charts, heatmaps, the graph view, KPI dashboards and kanban upgrades | Working since v0.12.0 · see [[Data-views]] |
| Context packs, the knowledge bank, `form-md mcp`, Claude Code, Codex and Cursor connections | Working since v0.13.0; the client configs are checked by parsing, not yet inside each app · see [[Context-and-knowledge]] |
| Terminal, Claude Code and its chat in the app, window commands for agents, `form-md cli` | Working since v0.14.0; checked with a real Claude Code turn, not yet under browser throttling · see [[Terminal-and-Claude-Code]] |
| Page styles with a workspace default, focus mode, word counts | Working since v0.15.0; checked in Chrome under CPU and network throttling |
| Image annotations, per-block alignment | Working since v0.16.0; checked in Chrome under CPU and network throttling |
| Custom workspace fonts, typewriter scrolling | Working since v0.17.0; checked in Chrome under CPU and network throttling |
| Board block, five creative embed providers, pictures on fresh public pages | Working since v0.18.0; checked in Chrome under CPU and network throttling |
| Recent pages kept offline, 1000-row offline databases, phone-width top bar and code blocks, described window commands | Working since v0.19.0; checked in Chrome · see [[Offline]] and [[Data-views]] |
| Interface and mail in 26 languages | Since v0.14.0; es-MX and fr-CA want a native read |
| At-rest encryption, `form-md encrypt` | Working since v0.11.0, independently audited · see [[Encryption]] |
| Built-in ACME certificates | Working, **in testing**: checked against a test CA (Pebble), not yet against Let's Encrypt production · see [[TLS-and-ACME]] |
| `backup`, `restore`, `doctor`, `admin reset-owner` | Working since v0.11.0 · see [[Backup-and-restore]] and [[Operations]] |
| Offline start from the service worker, installable app | Working, **in testing**; the installed app not yet tried on a phone · see [[Offline]] |
| Cloudflare tunnel | Working, **in testing** against a stand-in, not yet against real Cloudflare · see [[Tunnel]] |
| Gmail and Microsoft 365 sending | Working, **in testing** against a stub provider, not yet against Google or Microsoft · see [[Mail]] |
| Automatic HTTPS from Instance settings, browser backup download, update check | Working since v0.11.0 · see [[TLS-and-ACME]], [[Backup-and-restore]], [[Operations]] |
| `install.sh`, systemd unit, container image, release workflow | Written and tried locally (the image built and answered `/healthz`), **not yet exercised by a pushed tag** · see [[Install]] |
| SAML and SCIM | Tested against a local identity provider only, not yet against Okta, Entra or Google Workspace |
| AI providers | Verified against a local stand-in, not yet against live provider keys |
| Windows scripts (`*.bat`) | Written, **not yet run** on a Windows host |
| Automated tests | None in the tree by design; CI runs format, vet, type check, a build and a smoke run · see [[Development]] |

## Where to go

- Put an instance on a server: [[Install]]
- Every environment variable and the settings they seed: [[Configuration]]
- Encrypt the data directory at rest: [[Encryption]]
- HTTPS with your own certificate, built-in ACME, or a reverse proxy: [[TLS-and-ACME]]
- Reach the server through Cloudflare without opening a port: [[Tunnel]]
- Sending mail through SMTP, Gmail or Microsoft 365: [[Mail]]
- Back up and restore: [[Backup-and-restore]]
- Health checks, the data directory, logs, updates, memory, fail2ban and upgrades: [[Operations]]
- Charts, heatmaps, the graph view, dashboards and boards: [[Data-views]]
- What works without a connection, and the installable app: [[Offline]]
- REST API, tokens, OAuth, rate limits and connecting agents: [[API-and-agents]]
- Repo layout, dev loop and conventions: [[Development]]
- Branches, commits and releases: `BRANCHING.md` at the repo root

## Editing this wiki

The source of truth for this wiki is the `.docs/wiki/` folder in the main repository. Edit pages there on a `docs/*` branch, merge the pull request into `main`, then publish:

```sh
git clone https://github.com/nodemode-ai/form.md.wiki.git
cp form.md/.docs/wiki/*.md form.md.wiki/
cd form.md.wiki
git add -A
git commit -m "docs: sync wiki from .docs/wiki"
git push
```

Direct edits in the GitHub wiki interface are overwritten by the next sync, so keep every change in the repository.

Changing a subsystem's behaviour (a command, an environment variable, a route, a file in the data directory) means updating its page in the same pull request. A page that lags the code is worse than no page, because it is trusted.

Formatting rules for all pages: no emojis anywhere, no em dashes (use an interpunct `·` or restructure the sentence), plain markdown, real paths and real commands rather than abstract descriptions.
