# form.md Wiki

form.md is a self-hosted workspace where people and AI agents write documents, run databases and edit together in realtime. It ships as one static Go binary (`form-md`) with the TypeScript web client embedded, and it keeps everything in one data directory: a SQLite database (`form-md.db`), the uploaded files and a few small key files. There is no separate database server, cache or object store. The server holds the collaborative document itself, so an agent writing through MCP or the REST API merges with what a person is typing. This wiki is for the people who install and run an instance and for the people who change its code.

**Current version:** v0.37.0 (database views that really open offline, checked with the server stopped, which the v0.32.0 and v0.33.0 versions did not; v0.36.0 brought dashboard widgets moved with the arrow keys and a graph row list for screen readers; v0.35.0 brought pickers loaded on first use and a 260 KB start budget the build enforces; v0.34.0 brought a lighter start: one translation chunk per language and the library loaded on demand; v0.33.0 brought board, gallery, calendar, timeline, map and chart views of kept and recent databases open offline without having been opened first, maps included; v0.32.0 brought those views offline as last seen, and terminal colours following a system theme switch; v0.31.0 brought terminals drawn correctly under the security policy, a panel that opens with its terminal, restart in place, a chat tab without the empty-state error; v0.30.0 brought revoking one token, session or OAuth grant closing its live streams, editing sockets and terminal attachments; v0.29.0 brought context packs that name formlit or formvis when they fit the task, and the family-routing sub-skill; v0.28.0 brought the form family bridge: a public descriptor and tools that tell agents when formlit or formvis fits a task; v0.27.0 brought faster bulk row tools for agents; v0.26.0 brought rows added offline; v0.25.0 brought saved styles, offline row deletes, bulk row updates nearly three times faster; v0.24.0 brought annotation marks in Markdown and PDF, uploaded fonts in HTML exports, a list view that loads as you scroll; v0.23.0 brought editable single annotation marks, uploaded fonts on published pages, a windowed timeline; v0.22.0 brought offline database edits kept and sent later, page styles on public pages and exports, the slash setting in account settings; v0.21.0 brought a phone block toolbar, renaming and removing fonts, agents kept out of the block a person is typing in; v0.20.0 brought Phase 16 audit fixes: pictures from agents imported on the server, block alignment on public pages, a server-side default for command safety; v0.19.0 brought offline by default with larger databases, phone-width fixes, described window commands; v0.18.0 brought the board block and creative embeds; v0.17.0 brought custom fonts and typewriter scrolling; v0.16.0 brought image annotations and block alignment; v0.15.0 brought page styles, focus mode and word counts; v0.14.0 brought the terminal, Claude Code and its chat inside the app, window commands for agents, 26 languages; v0.13.0 brought context packs and agent connections, v0.12.0 the visual data views, v0.11.0 encryption, ACME, operations commands, offline start and the installer) · **Stack:** Go 1.26, pure-Go SQLite, vanilla TypeScript and Vite, ProseMirror and Yjs · **Team:** one owner (nodemode-ai)

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
| Board, gallery, calendar, timeline, map, graph, chart, dashboard and feed views of kept, recent and opened databases offline; heatmaps as last opened | Working since v0.37.0, checked with the server process stopped; the v0.32.0 and v0.33.0 versions did not work with the server really down · see [[Offline]] |
| Terminal and Claude Code panel checked under throttling: correct terminal cells, no empty first open, restart in place | Since v0.31.0; checked in Chrome at CPU 4x and 500 kbps · see [[Terminal-and-Claude-Code]] |
| Per-credential revoke: a signed-out session, deleted token or revoked OAuth grant loses its streams, sockets and terminals at once | Working since v0.30.0; checked in Chrome and over OAuth, and against v0.29.0 · see [[API-and-agents]] |
| Form family note in context packs, `family-routing` sub-skill, family numbers in `policy.json` | Since v0.29.0; checked against stubs of the two apps' endpoints · see [[Form-family]] |
| Form family descriptor, `family_list` and `family_route` for formlit and formvis | Since v0.28.0; checked against stubs of the two apps' endpoints, not the apps themselves · see [[Form-family]] |
| Bulk row tools for agents about twice as fast again (six times faster than v0.24.0 for updates) | Since v0.27.0; timed against v0.26.0 · see [[Data-views]] |
| Rows added offline, with client-made ids the server accepts | Working since v0.26.0; checked in Chrome and over the API · see [[Offline]] |
| Saved styles, offline row deletes, faster bulk row updates (template lookup) | Working since v0.25.0; checked in Chrome and timed against v0.24.0 |
| Annotation marks in Markdown and PDF exports, uploaded fonts in HTML exports, the list view loading as you scroll | Working since v0.24.0; checked in Chrome and by rasterizing the PDF |
| Picking, moving, recolouring and deleting single annotation marks, uploaded fonts on published pages, timeline windowing | Working since v0.23.0; checked in Chrome under CPU throttling |
| Offline cell edits kept and sent later, page styles on public pages and HTML exports, the slash setting in account settings | Working since v0.22.0; checked in Chrome · see [[Offline]] |
| Phone block toolbar, renaming and removing uploaded fonts, the person_editing guard for agent replaces | Working since v0.21.0; checked in Chrome under CPU throttling · see [[API-and-agents]] |
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
