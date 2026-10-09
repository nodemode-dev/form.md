<p align="center">
  <img src="assets/form-md_logo.svg" alt="form.md logo" width="160" />
</p>

<h1 align="center">form.md</h1>

<p align="center"><b>One workspace for people & agents: documents, databases & realtime editing in a single binary.</b></p>

<p align="center">
  <img src="https://img.shields.io/badge/version-v0.6.0-9a8cff" alt="version v0.6.0" />
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26+" />
  <img src="https://img.shields.io/badge/Node.js-22%2B-339933?logo=nodedotjs&logoColor=white" alt="Node.js 22+" />
  <img src="https://img.shields.io/badge/license-Apache--2.0-555555" alt="license Apache-2.0" />
</p>

form.md is a self-hosted workspace where people & AI agents write documents, run databases & edit together in realtime. It ships as one static Go binary w/ its web client embedded & one SQLite file beside it: no database server, no cache, no object store. The server holds the real collaborative document (a Yjs CRDT in pure Go), so an agent's edit merges into what a person is typing instead of resetting it. This repo is at **Phase 7 of 11**: the binary, storage, accounts, workspaces & the policy gate, the workspace shell (tree, tabs, favorites, tags, trash, library, covers & icons, uploads, live updates), the document engine w/ a block editor & realtime co-editing, databases w/ seven views, formulas, relations & rollups, public forms & CSV, full-text search w/ a command palette & the rest of the editor, & now collaboration & identity: page sharing w/ six access levels, teamspaces, groups, guests & links, comments, suggested edits, an inbox w/ push & mail, page history w/ a merge-safe restore, publishing to the web, analytics, signed webhooks, an audit log, SCIM & SAML. The agent surface & AI follow per the roadmap.

**Stack:** Go 1.26 · pure-Go SQLite (modernc) · ygo CRDT · vanilla TypeScript & Vite · ProseMirror & Yjs · no UI framework · **Scale:** 25 MB static binary, 229 KB JS & 23 KB CSS gzipped up front; database views, search, settings, KaTeX, the code highlighter, the Markdown parser & the Mermaid renderer load on demand

## Highlights

- **The server owns the document** · every page body is a CRDT the server applies & reads itself, so a Markdown write over the API merges w/ what people are typing; one socket carries every open page w/ state-vector sync, a write check on every frame & cursors named by the server
- **Offline that survives closing the tab** · edits made without a connection are kept in the browser & merge when it comes back, in any browser
- **The body in the first frame** · the server renders the page body into the document & the editor adopts it, so a reload shows the text before any script runs; switching pages swaps header & body together once the next page is ready, prefetched on hover
- **A block editor** · slash menu, Markdown shortcuts, a drag handle w/ turn into, a selection toolbar w/ colours, @ & [[ mentions of pages, people & dates, toggles, callouts, code in 29 highlighted languages, tables, columns, media & PDF w/ upload progress, a live table of contents & backlinks under the body
- **Databases where every row is a page** · 21 property types incl. status, relations w/ two-way sync, rollups & formulas; table, board, list, gallery, calendar, timeline & form views on one query engine w/ nested AND/OR filters, multi-sort, groups & sub-groups & column calculations, all answered by the server, so a filter on a formula works on 20,000 rows; inline databases inside any document
- **Big tables stay smooth** · 20,000 rows scroll at a 54 ms p95 step under 4x CPU & 500 kbps throttling; rows load in compressed windows of 100, windows that leave the screen are cancelled, & opening a database costs one round trip
- **Forms anyone can fill** · build a form from a database's properties w/ required & conditional questions, share a public link that needs no account, & watch answers arrive as rows live
- **CSV in & out** · upload a file, check the guessed type of each column, import 20,000 rows in about a second; export any view w/ its filter & sort
- **Share exactly what you mean to** · six access levels from view to full access, walked down the tree; open, closed & private teamspaces; groups, guests by email & share links w/ expiry & password; private pages reset what they inherit; every level enforced by the server on every route incl. the sync socket
- **Comments & suggestions that follow the text** · threads on a selection, a block or the page w/ mentions, reactions, resolve & a filtered pane; suggest mode for people w/ comment access, accepted or rejected per change; anchors are CRDT positions, so they hold while others type
- **An inbox that reaches you** · mentions, replies, assignments, date reminders & shares arrive live, w/ browser push & an email fallback for what stays unread
- **History w/o losing anyone's typing** · versions while editing, by hand & before every restore; preview & a block diff; a restore merges into the live document, so text a collaborator is typing at that moment survives
- **Publish to the web** · a page & its subpages as a site at its own slug or a custom domain, server-rendered w/ the editor's own markup, SEO fields, themes, fonts, an optional password, indexing control & a sitemap; share links open without an account
- **Know what is read** · views & unique viewers per page & per workspace over time, a personal opt-out & an anonymous-history switch
- **Enterprise identity** · SAML 2.0 w/ a SAML-only switch beside OIDC, SCIM 2.0 users & groups, signed webhooks w/ retries & auto-pause, an audit log w/ filters, CSV export, retention & a SIEM stream (Splunk HEC or JSON)
- **One static binary** built w/ `CGO_ENABLED=0`: client, migrations & license notices embedded; five release targets (linux & darwin on amd64/arm64, windows amd64)
- **Reads never queue behind writes** · one SQLite writer connection plus a query-only reader pool, numbered migrations in one transaction each, WAL checkpointed on clean shutdown
- **Zero first-paint flash** · the server renders the sign-in state, the account & the current workspace into the page, & an inline boot script applies theme & sidebar before CSS paints; Inter & JetBrains Mono are preloaded w/ metric-matched fallbacks, so text never reflows when a font arrives; verified frame by frame under 4x CPU & 500 kbps throttling, dark & light, signed out & signed in
- **Zero layout shift** · every async value has reserved space; measured CLS 0 across load & every shell interaction
- **Live without refetching** · one event stream per tab w/ sequence ids & replay; another browser's rename, move or trash patches exactly the rows involved, never the whole list; private pages reach only their owner; a watchdog reconnects & a reset refetches without losing newer events
- **A tree that keeps up** · pointer drag w/ before, after & inside zones, auto-expand & auto-scroll; full keyboard walk; optimistic moves on fractional keys, so a reorder writes one row; trash in batches w/ undo
- **Tabs that behave like a browser's** · middle or modifier click opens a tab, back & forward restore the exact tab set, tabs persist across reloads
- **Icons without emoji** · 99 hand-drawn line icons in ten tints or an uploaded picture, enforced by the server; twelve gradient covers that collapse to a strip as you scroll
- **One stacking scale, one popover primitive, one layer stack** · no raw `z-index` outside the tokens file; Escape, outside-press, inert focus trapping & focus return handled in one place
- **Accounts done properly** · argon2id w/ equal-time unknown accounts, TOTP w/ replay protection & recovery codes, Google, Microsoft & any OpenID Connect provider w/ verified ID-token signatures, single-use invitations, API tokens narrowed by scope & workspace
- **One policy gate** · sessions & tokens resolve to one principal; a credential can only narrow its person; per-workspace agent policy (open, strict, closed); anything unreadable answers not found
- **Owner tools** · time-boxed, audited emergency read access, deletion w/ an impact plan, stranded-workspace recovery, ownership hand-over
- **Keyboard first** · one scoped shortcut registry (layout-independent letters), a live shortcut sheet on `?`, chord hints in every tooltip
- **Hardened by default** · hash-pinned CSP for the inline boot script, `nosniff`, frame denial, request ids, panic recovery, loopback-only default bind

## Quick Start

### Requirements

Go 1.26 or newer & Node.js 22 or newer. Everything else is installed by the scripts below.

### macOS & Linux

```shell
chmod +x form-md_oc_install form-md
./form-md_oc_install
./form-md
```

Open http://127.0.0.1:5750 while developing. To build the single binary & run it:

```shell
make build
./bin/form-md
```

Then open http://127.0.0.1:8750. The first visit shows the set-up screen that creates the owner account; everyone else joins by invitation unless sign-up is opened in Instance settings.

### Windows

```shell
form-md_oc_install.bat
form-md.bat
```

### Ports

| Service | URL | Notes |
|---|---|---|
| Client (dev) | http://127.0.0.1:5750 | Vite dev server, `server.host` false, `strictPort`, proxies API routes to the server |
| Server | http://127.0.0.1:8750 | `form-md serve`; also serves the embedded client in a release build |
| Client preview | http://127.0.0.1:5751 | `npm run preview` in `web/` |

## Configuration

Start-up settings come from environment variables; there is no config file. `form-md help` prints the same table. Everything else (instance name & public address, sign-up policy, single sign-on, mail, housekeeping) is set by an admin in **Instance settings** inside the app & stored in the database; secrets there are sealed w/ `secret.key` in the data directory, so back up the whole data directory, not just the database file.

| Variable | Purpose |
|---|---|
| `FORMMD_ADDR` | Listen address, default `127.0.0.1:8750`; set `0.0.0.0:8750` to accept other machines |
| `FORMMD_DATA` | Data directory for the database & uploads, default `./data` |
| `FORMMD_TLS_CERT` | TLS certificate path; set together w/ `FORMMD_TLS_KEY` |
| `FORMMD_TLS_KEY` | TLS private key path; set together w/ `FORMMD_TLS_CERT` |
| `FORMMD_FETCH_PRIVATE` | Set to `1` to let link previews & media imports reach private network addresses (intranet sites); off by default |
| `FORMMD_TRASH_DAYS` | Default for emptying the trash automatically, in days (0 disables); a value saved in Instance settings wins |
| `FORMMD_UPDATE_CHECK` | Default for the daily version check; a value saved in Instance settings wins |

## Architecture

```
form.md/
├── cmd/form-md/        entry point & subcommands (serve, version, help)
├── embed.go            embeds web/dist & NOTICES.md into the binary
├── internal/
│   ├── app/            module registry, server lifecycle, /healthz, /notices
│   ├── server/         wiring & the server-rendered page document
│   ├── auth/           passwords, sessions, TOTP, API tokens, OIDC & SAML sign-in, invitations
│   ├── policy/         the principal & the single workspace gate
│   ├── workspace/      workspaces, members, agent policy & rules
│   ├── page/           the page tree, privacy, moves, duplicates, trash, favorites, tags, recents
│   ├── order/          fractional sort keys
│   ├── files/          uploads & access-checked file serving
│   ├── doc/            page bodies: CRDT rooms, the sync socket, Markdown in & out, first-frame HTML
│   ├── collection/     databases: schema, values, the query engine, views, forms, CSV; formula/ is the formula language
│   ├── search/         full-text search: passages, the FTS5 index worker, ranking, highlights
│   ├── comment/        threads on text, blocks & pages, mentions, reactions, suggested edits
│   ├── inbox/          notifications, Web Push w/ VAPID, the mail fallback, date reminders
│   ├── site/           published sites, custom domains, passwords, share-link pages, sitemaps
│   ├── analytics/      page views & viewers, privacy preferences
│   ├── webhook/        signed webhooks & SIEM streams fed from the audit log, retries
│   ├── auditapi/       audit log queries, CSV export
│   ├── scim/           SCIM 2.0 users & groups per workspace
│   ├── netguard/       the outbound HTTP client that only reaches public addresses
│   ├── unfurl/         link previews for link cards
│   ├── events/         the live event stream w/ replay & audience filtering
│   ├── icon/           the icon rule shared w/ the client's icon set
│   ├── admin/          users, ownership, emergency access, instance settings
│   ├── settings/       typed instance settings w/ sealed secrets
│   ├── secrets/        AES-GCM sealing & token minting
│   ├── audit/          the activity record, its queries & retention
│   ├── mail/           SMTP sending
│   ├── config/         FORMMD_* environment variables
│   ├── httpx/          router, middleware chain, JSON errors, rate limits
│   ├── static/         embedded client: rendered document, caching, gzip, ETag, CSP
│   ├── store/          SQLite writer & reader pool, migrations
│   └── ids/            128-bit random ids
├── web/                vanilla TypeScript client
│   ├── index.html      page template the server fills: shell, sign-in screens, pre-paint theme boot
│   ├── sandbox.html    the sandboxed page that draws Mermaid diagrams
│   └── src/
│       ├── engine/     headless state, page & database stores, query sessions, live stream, document sync, router & tabs, commands, keymap, API client, i18n
│       ├── extensions/ feature modules registered w/ the engine, incl. the block editor & the database views
│       ├── ui/         DOM layer: sidebar tree, top bar, page header, library, trash, pickers, dialogs, settings, tokens
│       └── styles/     component styles on the token system
├── tools/              Go side install & boot scripts, NOTICES generator
├── assets/             repo-level art for this README, not part of the build
├── Makefile            build, release, check
└── form-md             dev boot for both sides
```

`internal/HASHOUT_LOG.md` & `web/HASHOUT_LOG.md` map every file & record the reasoning that would otherwise be code comments. Read them before changing code.

## Status of subsystems

- **Accounts & sign-in:** built. Put HTTPS in front before exposing an instance (built-in certificates arrive in Phase 11).
- **Mail:** SMTP only; Gmail & Microsoft Graph sending arrive in Phase 11.
- **German:** the language setting exists; the German translation arrives in Phase 11.
- **Pages & the shell:** built: tree, tabs, favorites, tags, notes mode, library, trash, covers, icons, uploads, live updates.
- **Editor & realtime co-editing:** built, incl. equations, a breadcrumb block, columns made by dragging, link cards filled by the server, media from links & pasted pages, embeds, Mermaid diagrams drawn in a sandbox & synced blocks across pages. Table cells cannot be merged, & a synced block has no slash menu inside it yet.
- **Search:** built: titles, bodies w/ heading paths & database properties, filters & sorts, from a palette (Ctrl or Cmd+K). PDF text arrives w/ the PDF importer (Phase 10).
- **Databases:** built: 21 property types, seven views, formulas, relations & rollups, groups & calculations, inline databases, row pages, public forms, CSV import & export. The button property is reserved for automations (Phase 9); chart, map & feed views arrive in Phase 9.
- **Sharing, comments, suggestions, inbox & history:** built. A restore keeps blocks someone is typing in as they are & says how many.
- **Publishing:** built. Custom domains need TLS from a reverse proxy until built-in certificates (Phase 11); databases publish as a table of their first view.
- **Analytics, webhooks, audit log, SCIM & SAML:** built. SAML & SCIM are tested against a local identity provider & provider-shaped requests, not yet against Okta, Entra or Google Workspace.
- **Agents (MCP) & AI:** not built yet (Phase 8).
- **Windows scripts:** written but not yet run on a Windows host.

## Shortcuts

| Keys | Action |
|---|---|
| `Alt+N` | New page |
| `F` | Add or remove the open page from favorites |
| `L` | Library |
| `M` | Notes mode on or off |
| `[` · `Ctrl+\` | Show or hide the sidebar |
| `Alt+Shift+→` · `Alt+Shift+←` | Next or previous tab (also `Ctrl+Alt+→/←` outside macOS) |
| `Alt+W` | Close the tab |
| `↑ ↓ ← →` · `Enter` · `F2` | Walk, fold & open the page tree; F2 renames |
| `Ctrl+Backspace` | Move the focused tree row to the trash, w/ undo |
| `/` · `@` · `[[` | In a page: block menu, mention a page, person or date, link or create a page |
| `Ctrl+B` · `Ctrl+I` · `Ctrl+U` · `Ctrl+E` · `Ctrl+Shift+X` | Bold, italic, underline, inline code, strikethrough |
| `Ctrl+K` | Link the selected text |
| `Ctrl+Alt+0` to `3` | Text, heading 1, 2 or 3 |
| `Ctrl+Shift+8` · `7` · `9` | Bulleted, numbered or to-do list |
| `Alt+↑` · `Alt+↓` · `Ctrl+D` | Move the block up or down, duplicate it |
| `Ctrl+Enter` | Tick or untick a to-do |
| `Ctrl+Shift+C` | Copy the page as Markdown |
| `?` | Every shortcut that is live right now |

On macOS, `Ctrl` reads as `Cmd`.

## Development

- One `main` branch for now; a branch model arrives w/ the wiki in Phase 11
- Conventions: surgical edits only, zero emojis anywhere incl. markdown, every colour & layer from the token files, no icon libraries, no UI framework, semver tags from `main`
- `make check` runs gofmt, `go vet` & the client type check

## Status

Phases 1 to 7 (skeleton binary, accounts, workspaces, the policy gate, the workspace shell w/ pages & live updates, the document engine w/ the block editor & realtime co-editing, databases w/ their views, forms & CSV, search w/ the rest of the editor, & collaboration, sharing & identity) are complete at this version. Phase 8, agents & AI, is next. The full roadmap is kept outside the tree in the planning sidecar that ships w/ each release zip.

## Contributors

- `Owner` · nodemode-ai

### Licensing

Apache-2.0, see [LICENSE](LICENSE). Third-party components & their licenses are listed in [NOTICES.md](NOTICES.md), which the binary also serves at `/notices`. The Inter & JetBrains Mono font files in `web/src/fonts/` are under the SIL Open Font License 1.1, w/ the license texts beside them.
