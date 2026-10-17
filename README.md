<p align="center">
  <img src="assets/form-md_logo.svg" alt="form.md logo" width="160" />
</p>

<h1 align="center">form.md</h1>

<p align="center"><b>One workspace for people & agents: documents, databases & realtime editing in a single binary.</b></p>

<p align="center">
  <img src="https://img.shields.io/badge/version-v0.29.0-9a8cff" alt="version v0.29.0" />
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26+" />
  <img src="https://img.shields.io/badge/Node.js-22%2B-339933?logo=nodedotjs&logoColor=white" alt="Node.js 22+" />
  <img src="https://img.shields.io/badge/license-Apache--2.0-555555" alt="license Apache-2.0" />
</p>

form.md is a self-hosted workspace where people & AI agents write documents, run databases & edit together in realtime. It ships as one static Go binary w/ its web client embedded & one SQLite file beside it: no database server, no cache, no object store. The server holds the real collaborative document (a Yjs CRDT in pure Go), so an agent's edit merges into what a person is typing instead of resetting it. This repo is at **Phase 13 of 14**: the binary, storage, accounts, workspaces & the policy gate, the workspace shell (tree, tabs, favorites, tags, trash, library, covers & icons, uploads, live updates), the document engine w/ a block editor & realtime co-editing, databases w/ seven views, formulas, relations & rollups, public forms & CSV, full-text search w/ a command palette & the rest of the editor, collaboration & identity (sharing w/ six access levels, teamspaces, comments, suggested edits, an inbox, history, publishing, analytics, webhooks, an audit log, SCIM & SAML), & now agents & AI: an MCP server w/ OAuth, AI providers w/ an allow-list, writing help, AI blocks, autofill, a workspace agent & custom agents, meeting notes & answers from the workspace, automations, charts & dashboards, buttons, templates, map & feed views, importers from a dozen tools, Markdown, HTML & PDF export, a documented REST API, visual data views, & now context packs & a knowledge bank for agents w/ ready connections for Claude Code, Codex & Cursor.

**Stack:** Go 1.26 · pure-Go SQLite (modernc) · ygo CRDT · vanilla TypeScript & Vite · ProseMirror & Yjs · no UI framework · **Scale:** 28 MB static binary, 246 KB JS & 29 KB CSS gzipped up front; database views & charts, automations, button editing, import & export, the sharing, publishing & history dialogs, search, settings, the agent pane, KaTeX, the code highlighter, the Markdown parser & the Mermaid renderer load on demand

## Highlights

- **Claude Code inside the workspace** · a bottom panel with real terminals, Claude Code tabs already connected to this form.md's MCP tools, and a Claude Code chat that shows each tool call as it runs; hand a chat to a terminal tab or bring one back (Mod+J, Mod+Alt+J, Mod+Alt+K)
- **Agents drive the window** · every command carries a description, typed parameters & whether it changes anything; agents list & run them through MCP (`window_commands`, `window_command_run`), REST or `form-md cli`
- **Page styles** · named saved styles for the workspace; six fonts (sans, serif, mono, rounded, slab, humanist) plus fonts you upload to the workspace (WOFF2, WOFF, TTF, OTF), three text sizes & widths, line spacing & justified or centred text per page, w/ a workspace default; styled pages paint styled from the first frame
- **Image annotations** · arrows, boxes, ellipses, freehand lines & text drawn over any picture in a page; pick any mark to move, recolour, retype or delete it; kept in Markdown & drawn in PDF exports; synced live & present in the first frame, public pages & HTML export
- **Block alignment** · left, centred, right or justified per paragraph & heading from the format bar
- **Boards** · a moodboard block where pictures, notes & link cards sit anywhere on a dotted surface; drop pictures straight onto it, drag & resize cards, & the board scales w/ the column on public pages too
- **Phone editing** · a block toolbar above the on-screen keyboard to add, move, turn into, duplicate & delete blocks
- **Agents never overwrite your typing** · an agent's replace or delete of the block you are typing in is refused until you move on, & inserts merge live
- **Offline by default** · the 20 pages & databases you opened most recently are kept on the device w/o marking them, a database kept offline keeps up to 1000 rows & their row pages, & rows added, edited or deleted offline are kept & sent when you are back
- **Creative embeds** · Canva, Behance, Sketchfab, Pinterest & Apple Music join Figma, Miro, YouTube, Vimeo, Spotify, SoundCloud, Google Docs & Maps & the rest; pictures on freshly published pages load at once
- **Writing tools** · focus mode that hides everything but the text & keeps the line you type mid-screen, live word, character & reading-time counts for the page, the selection or the paragraph, & an option that keeps the slash menu out of sentences
- **26 languages** · the whole window & its mail in English (default), Spanish (Spain & Mexico), Brazilian Portuguese, French (France & Canada), German, Italian, Dutch, Danish, Finnish, Norwegian Bokmål, Swedish, Russian, Polish, Czech, Hungarian, Ukrainian, Turkish, Simplified & Traditional Chinese, Japanese, Korean, Indonesian, Thai & Vietnamese
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
- **Agents edit beside people** · an MCP server (2026-07-28, older clients too) w/ OAuth 2.1 sign-in, consent per workspace & read or write scope; agents address blocks by id, so their edits merge w/ live typing, show as a named cursor & group into runs that undo in one click; a downloadable skill teaches agents the conventions
- **The right context at the right time** · `context_pack` hands an agent the workspace rules, recorded knowledge, the pages it named & the best passages, ranked & cut to a token budget per task type, or only what changed since its last pack; a knowledge bank of sourced facts & decisions that confirm, decay & retire; all numbers in one `policy.json` shared by the `contextual-compression` skill & the server
- **Form family aware** · `/.well-known/form-family.json` describes the instance, & `family_list` / `family_route` tell an agent whether formlit (photos) or formvis (image editing) runs on the same computer, which app fits a task & which of its tools match, loopback only
- **Claude Code, Codex & Cursor in one step** · copy-ready snippets in Connected apps, `form-md mcp` as a stdio bridge for command-based clients, the skill as MCP prompts & resources, & `CLAUDE.md`, `AGENTS.md`, `.mcp.json` & a Cursor rule in the repo
- **Bring your own models** · Anthropic, OpenAI, Gemini or any OpenAI-compatible server (Ollama, vLLM); keys sealed in the database, an owner allow-list, a default & a transcription model, per-person hourly limits, usage per model, a switch per workspace
- **AI in the editor** · Ctrl+J or the selection toolbar: rewrite, fix, shorten, change tone, explain, summarize, find action items, translate or any instruction, streamed into a preview & placed as real blocks; AI blocks that regenerate from their prompt; AI autofill properties that refill rows when they change
- **A workspace agent & custom agents** · chat w/ an agent that searches, writes pages, builds databases & fills rows through the same tools, reads your instructions page & remembers what you tell it; custom agents run on a schedule, when a database gets or changes a row, or when a page changes, each run undoable
- **Meeting notes & answers** · record the microphone & a tab or the system audio, get a transcript, summary & action items in the page; ask a question in the palette & get an answer cited from your pages
- **Charts & dashboards** · bar, line, area, donut, radial, gauge, histogram, dot plot (diverging beeswarm), period comparison, matrix & calendar heatmaps, computed by the server from the same query engine, so filters & access rules hold; click any mark to list & open its rows; dashboards of small, medium & wide cards rearranged by drag, KPI cards w/ a sparkline & the change; PNG & SVG export
- **A heatmap view** · a year, months, weeks by hour or days by hour of any date property, counted or summed, in your colour & scale; click a day to see its rows or add one on that date
- **A graph view** · rows as nodes & relations as links, laid out by a force simulation in a worker, coloured & clustered by a select, sized by links or a number; drag from one row to another to relate them, click a link to remove it, double-click to add a row, edit in a side panel
- **Kanban that keeps score** · column totals, a progress bar from a checkbox or number, card limits w/ an over-limit mark, collapsed columns & a pencil to edit a card's title & properties in place
- **Automations** · when a row is added, a property changes or on a schedule: edit rows, add rows, notify people, send mail & post signed webhooks; a burst of edits fires once, an automation never triggers itself, & each run says what happened
- **Buttons** · button blocks in any page & a button property in databases: insert blocks, open a page, confirm, & every automation action, w/ variables for the row, the person & the date
- **Templates** · a starter gallery of our own documents & databases, any page saved as a template, row templates inside a database w/ a default
- **Map & feed views** · rows w/ a place drawn on a bundled public-domain world map (no tile server); rows as posts w/ comments & reactions
- **More from the database** · a private calendar subscription address per calendar view, sub-items that fold under their parent, double-press a column edge to fit it, a link graph of the workspace & a Databases section in the sidebar
- **Bring everything in** · Notion & Confluence exports, Evernote, Markdown & HTML files or zips, Word, Excel, CSV & Asana, Trello boards, OPML & WorkFlowy, plain text & PDF text; page trees, databases w/ typed columns & row pages, pictures & links between pages all arrive intact, with progress & a report
- **Take everything out** · a page, a tree or the whole workspace as Markdown w/ databases as CSV (it imports back), as HTML styled like a published site, or as one PDF w/ bookmarks & pictures
- **A documented API** · an OpenAPI 3.1 document generated from the live router & a reference page at `/api/docs`; personal tokens or OAuth 2.1 tokens for the REST API; opaque cursors on long lists; per-token rate limits w/ standard headers
- **Enterprise identity** · SAML 2.0 w/ a SAML-only switch beside OIDC, SCIM 2.0 users & groups, signed webhooks w/ retries & auto-pause, an audit log w/ filters, CSV export, retention & a SIEM stream (Splunk HEC or JSON)
- **Opens without the server** · a service worker caches the app & a signed-in shell, visited pages live in the browser, & "Available offline" keeps a page & its subpages ready before you need them; a reload with the server down shows the tree & the text
- **Encrypted at rest w/ a key you hold** · every database page & WAL frame through an XTS-AES-256 SQLite VFS, uploads as chunked AES-256-GCM that still serve ranges, sealed secrets & certificates; the key lives outside the data directory, a wrong or missing key refuses to start, & `form-md encrypt` converts an existing instance
- **Run it yourself w/o a manual** · a checksum-verifying one-line installer w/ a hardened systemd unit, a distroless multi-arch image, built-in Let's Encrypt or a Cloudflare tunnel, `form-md backup` while people work, an atomic `restore`, `doctor`, an owner reset, a browser backup download, a daily update check & a fail2ban filter
- **Deutsch** · every screen, the sign-in page rendered by the server & the mail in German, per person
- **Mail through Gmail or Microsoft 365** · connect the sending account w/ OAuth instead of an SMTP password
- **Installable** · a web app manifest, a Sync button in the installed window & pull to refresh on phones
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

Open http://127.0.0.1:5750 while developing. To build the single binary from a clean clone & run it:

```shell
./build_app          # checks Go & Node, installs, builds the client, compiles bin/form-md
./form-md serve      # or ./bin/form-md
```

`./form-md help` lists the rest: `release` (every platform into `release/`), `claude` (Claude Code in this terminal w/ form.md's tools), `mcp`, `agents install`, `cli list|describe|run`, `check`, `counts`, `clean`.

Then open http://127.0.0.1:8750. The first visit shows the set-up screen that creates the owner account; everyone else joins by invitation unless sign-up is opened in Instance settings.

### Windows

```shell
form-md_oc_install.bat
form-md.bat
```

### Servers

```shell
curl -fsSL https://github.com/nodemode-ai/form.md/releases/latest/download/install.sh | sh
```

The installer verifies the download against `SHA256SUMS`, installs `/usr/local/bin/form-md` & on systemd hosts a hardened `form-md` service w/ its data in `/var/lib/form-md`. With Docker: `docker run -p 8750:8750 -v form-md:/data ghcr.io/nodemode-ai/form.md`. Both are published by the release workflow from v0.11.0 on; the wiki in `.docs/wiki` covers install, encryption, TLS, backups & operations.

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
| `FORMMD_KEY_FILE` | At-rest encryption key (`form-md keygen` makes one), kept outside the data directory; a systemd credential named `form-md.key` works too |
| `FORMMD_ACME_DOMAINS` | Host names to get Let's Encrypt certificates for; w/ `FORMMD_ACME_EMAIL`, `FORMMD_ACME_DIRECTORY` & `FORMMD_HTTP_ADDR` |
| `FORMMD_CLOUDFLARED` | Path to `cloudflared` for the public tunnel when it is not on `PATH` |
| `FORMMD_UPDATE_URL` | Where the update check reads the latest release (a mirror on closed networks) |
| `FORMMD_FETCH_PRIVATE` | Set to `1` to let link previews & media imports reach private network addresses (intranet sites); off by default |
| `FORMMD_TRUST_PROXY` | Default for trusting `X-Forwarded-*` headers behind a reverse proxy; a value saved in Instance settings wins |
| `FORMMD_TRASH_DAYS` | Default for emptying the trash automatically, in days (0 disables); a value saved in Instance settings wins |
| `FORMMD_UPDATE_CHECK` | Default for the daily version check; a value saved in Instance settings wins |

Commands: `form-md serve` (default), `backup FILE`, `restore FILE`, `doctor`, `encrypt`, `keygen`, `admin reset-owner`, `admin set KEY VALUE`, `version`.

## Architecture

```
form.md/
├── cmd/form-md/        entry point & subcommands (serve, backup, restore, doctor, encrypt, keygen, admin, version)
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
│   ├── mcp/            the MCP server & its tools; oauth/ signs agents in
│   ├── contextpack/    context packs & the knowledge bank, driven by the skill's policy.json
│   ├── agentrun/       agent runs, their inverses & undo
│   ├── ai/             model providers, the allow-list, limits, writing help; aifill/ fills properties
│   ├── agent/          the workspace agent, memory & custom agents; answer/ writes cited answers
│   ├── automation/     automations & button runs: triggers, schedules, actions, signed webhooks
│   ├── templates/      the starter gallery & page templates
│   ├── feed/           the feed view's posts, reactions & comment counts
│   ├── graph/          the workspace link graph
│   ├── transfer/       import jobs & readers for each format; Markdown, HTML & PDF exports
│   ├── apidoc/         the OpenAPI document & the API reference page
│   ├── netguard/       the outbound HTTP client that only reaches public addresses
│   ├── unfurl/         link previews for link cards
│   ├── events/         the live event stream w/ replay & audience filtering
│   ├── icon/           the icon rule shared w/ the client's icon set
│   ├── admin/          users, ownership, emergency access, instance settings
│   ├── settings/       typed instance settings w/ sealed secrets
│   ├── secrets/        AES-GCM sealing & token minting
│   ├── crypt/          the operator key, key check & sealed files
│   ├── cryptvfs/       the encrypting SQLite VFS (XTS-AES-256 pages)
│   ├── ops/            backup, restore, doctor, encrypt, owner reset, the data-directory lock, memory sizing
│   ├── tunnel/         the supervised Cloudflare tunnel on a private socket
│   ├── i18n/           server-side catalogs for mail, push & the rendered page
│   ├── audit/          the activity record, its queries & retention
│   ├── mail/           SMTP, Gmail & Microsoft Graph sending
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
├── tools/              Go side install & boot scripts, NOTICES generator, the wiki check
├── deploy/             systemd unit, fail2ban filter & jail
├── install.sh          the one-line installer
├── Dockerfile          multi-arch distroless image
├── .docs/wiki/         operator & developer wiki, checked against the code by make check
├── assets/             repo-level art for this README, not part of the build
├── Makefile            build, release, check
└── form-md             dev boot for both sides
```

`internal/HASHOUT_LOG.md` & `web/HASHOUT_LOG.md` map every file & record the reasoning that would otherwise be code comments. Read them before changing code.

## Status of subsystems

- **Accounts & sign-in:** built. HTTPS from built-in ACME (checked against a local Pebble), operator certificates, a reverse proxy or the Cloudflare tunnel.
- **Mail:** SMTP, Gmail & Microsoft Graph. Gmail & Graph were checked against a stand-in provider, not yet against Google or Microsoft.
- **German:** built: every client string, the server-rendered page & mail.
- **Offline start:** built. Kept & visited databases open read-only w/ their first rows, & their row pages, pictures & files on kept pages open too; editing them, search & other server data need the server; text edits made offline sync when it is back.
- **Encryption at rest:** built. File sizes & WAL frame headers (w/ their checksums) stay visible; there is no key rotation yet.
- **Operations & packaging:** built. The systemd branch of the installer, the arm64 image & the workflows were not run in this sandbox; the tunnel was checked w/ a stand-in for cloudflared.
- **Pages & the shell:** built: tree, tabs, favorites, tags, notes mode, library, trash, covers, icons, uploads, live updates.
- **Editor & realtime co-editing:** built, incl. equations, a breadcrumb block, columns made by dragging, link cards filled by the server, media from links & pasted pages, embeds, Mermaid diagrams drawn in a sandbox & synced blocks across pages. Table cells cannot be merged, & a synced block has no slash menu inside it yet.
- **Search:** built: titles, bodies w/ heading paths & database properties, filters & sorts, from a palette (Ctrl or Cmd+K). The text of uploaded PDFs is not indexed yet; imported PDFs become pages whose text is.
- **Databases:** built: 21 property types, seven views, formulas, relations & rollups, groups & calculations, inline databases, row pages, public forms, CSV import & export, chart, dashboard, map & feed views, the button property, sub-items & calendar subscriptions.
- **Automations, buttons & templates:** built. Mail from automations goes only to workspace members; webhooks to private addresses need `FORMMD_FETCH_PRIVATE`.
- **Import & export:** built. Readers were checked against sample files made for this project, not yet against large real exports; scanned PDFs have no text to import. Imports run one or two at a time & a restart cancels a running one.
- **REST API:** built & documented at `/api/docs`. OAuth clients ask for the resource `<origin>/api`; MCP keeps its own tokens.
- **Sharing, comments, suggestions, inbox & history:** built. A restore keeps blocks someone is typing in as they are & says how many.
- **Publishing:** built. Custom domains need TLS from a reverse proxy (built-in ACME covers the instance's own names); databases publish as a table of their first view.
- **Analytics, webhooks, audit log, SCIM & SAML:** built. SAML & SCIM are tested against a local identity provider & provider-shaped requests, not yet against Okta, Entra or Google Workspace.
- **Agents (MCP) & AI:** built. Verified against a local provider stand-in, not yet against live Anthropic, OpenAI or Gemini keys. AI answers draw on the workspace only; web & connector sources are not built.
- **Context packs, knowledge & agent connections:** built. The stdio bridge & the configs were checked by parsing them & driving the bridge from a script, not yet inside the Codex, Cursor or Claude Code apps themselves. Ranking is lexical (full-text rank, word overlap, links & recency); there are no embeddings.
- **Windows scripts:** written but not yet run on a Windows host.

## Shortcuts

| Keys | Action |
|---|---|
| `Alt+N` | New page |
| `Ctrl+Shift+F` | Focus mode on or off (`Esc` leaves) |
| `Ctrl+J` | Terminal & Claude Code panel (owner, on this computer) |
| `Ctrl+Alt+J` · `Ctrl+Alt+K` | Claude Code in a terminal tab · Claude Code chat |
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
| `Ctrl+J` | Ask AI about the selection or at the cursor |
| `Ctrl+Shift+J` | Show or hide the agent |
| `?` | Every shortcut that is live right now |

On macOS, `Ctrl` reads as `Cmd`.

## Development

- Branch model in `BRANCHING.md`: `main` only, releases are annotated `vX.Y.Z` tags that the release workflow builds
- Conventions: surgical edits only, zero emojis anywhere incl. markdown, every colour & layer from the token files, no icon libraries, no UI framework, semver tags from `main`
- `make check` runs gofmt, `go vet`, the wiki-versus-code check & the client type check

## Status

Phases 1 to 10 (skeleton binary, accounts, workspaces, the policy gate, the workspace shell w/ pages & live updates, the document engine w/ the block editor & realtime co-editing, databases w/ their views, forms & CSV, search w/ the rest of the editor, collaboration, sharing & identity, agents & AI, automations, charts, buttons, templates, map & feed, & import, export & the documented API) , Phase 11 (offline start, encryption at rest, built-in HTTPS & the tunnel, backup & restore, packaging, German & the wiki) , Phase 12 (the graph & heatmap views, eight more chart kinds w/ drill-down, KPI dashboards & kanban upgrades) & Phase 13 (context packs, the knowledge bank, the contextual-compression skill, the stdio bridge & connections for Claude Code, Codex & Cursor) are complete at this version. Phase 14, a bridge to the other form* platforms, waits for those builds. The full roadmap is kept outside the tree in the planning sidecar that ships w/ each release zip.

## Contributors

- `Owner` · nodemode-ai

### Licensing

Apache-2.0, see [LICENSE](LICENSE). Third-party components & their licenses are listed in [NOTICES.md](NOTICES.md), which the binary also serves at `/notices`. The Inter & JetBrains Mono font files in `web/src/fonts/` are under the SIL Open Font License 1.1, w/ the license texts beside them. The map view's world outline in `web/public/maps/` is derived from Natural Earth (public domain) via world-atlas (ISC), w/ the notice beside it.
