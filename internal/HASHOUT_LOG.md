# HASHOUT_LOG · backend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the Go side of the repo (`cmd/`, `embed.go`, `internal/`, `tools/`). Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to the repo root.

Current build version: v0.42.1
Last updated: 2026-10-20 · v0.42.1: the notices tool finds the Go licence outside GOROOT; v0.42.0: Claude Code launched like formlit's, chat events for every window, form-md mcp autostart, the sparkle retired; v0.41.0: import sources by file format, PDF text in search, webhook page records, the web as an answer source, published boards and galleries, the grouped-offset fix; Phase 14: terminals, Claude Code and its chat, window commands, form-md cli, 26 languages

---

## Org-navi-map

```
form.md/
├── embed.go                         package formmd: embeds web/dist and NOTICES.md
├── go.mod, go.sum                   module github.com/nodemode-ai/form.md, Go 1.26+
├── NOTICES.md                       generated third-party notices, embedded, served at /notices
├── Makefile                         build, web, notices, server, release, check, clean
├── form-md                          root command set: boot (default), install, build, release, serve, claude, mcp, agents, cli, check, counts, clean
├── form-md.bat                      Windows dev boot: Go server on :8750 plus Vite on :5750
├── build_app                        one script from a clean clone to bin/form-md (or every target with --all)
├── form-md_oc_install (+ .bat)      installs both sides
├── cmd/
│   └── form-md/
│       ├── main.go                  subcommands (serve, version, help), wiring, exit codes
│       ├── cli.go                   form-md cli list|describe|run: the open window's command catalogue over REST
│       └── bridge.go                form-md mcp: stdio to Streamable HTTP bridge for command-based MCP clients
├── internal/
│   ├── HASHOUT_LOG.md               this file
│   ├── admin/
│   │   └── admin.go                 /api/admin/*: users, roles, access, ownership, emergency, stranded, settings, mail test
│   ├── app/
│   │   ├── app.go                   Deps (incl. Middleware), Module interface, Run (listen, serve, graceful shutdown)
│   │   ├── version.go               Version (set by -ldflags), Product name
│   │   ├── health.go                GET /healthz
│   │   └── notices.go               GET /notices
│   ├── agent/
│   │   ├── agent.go                 chats, messages, memory, prefs (instructions page)
│   │   ├── loop.go                  the agent turn: system prompt, tools, step loop, run finish
│   │   ├── custom.go                custom agents: CRUD, triggers, NextRun, runs list
│   │   ├── runner.go                schedule loop, event triggers, debounce, loop guards, launch
│   │   └── module.go                /api/agent*, /api/workspaces/{id}/agents, /api/agents/*
│   ├── agentrun/
│   │   └── agentrun.go              runs and their undo steps, undo in reverse, run listing
│   ├── ai/
│   │   ├── provider.go              Provider and Transcriber interfaces, messages, SSE reader
│   │   ├── anthropic.go             Messages API stream with tools
│   │   ├── openai.go                chat/completions stream with tools (OpenAI, Gemini, compatible), transcription
│   │   ├── service.go               providers, allow-list, resolve, Allowed, Exec, Run, Transcribe, usage
│   │   ├── writing.go               writing actions and their prompts
│   │   └── module.go                admin AI routes, models, workspace switch, complete, transcribe
│   ├── aifill/
│   │   └── aifill.go                AI autofill: per-type answers, fill all, refill on change
│   ├── analytics/
│   │   └── analytics.go             page views per viewer per day, page and workspace reports, view privacy
│   ├── answer/
│   │   └── answer.go                POST /api/ai/answer: sources from search, cited stream
│   ├── audit/
│   │   ├── audit.go                 audit_log writer: actor kind, action, workspace, page, field changes
│   │   └── query.go                 filtered, cursor-paged reads, facets, retention purge
│   ├── auditapi/
│   │   └── auditapi.go              GET /api/workspaces/{id}/audit and audit.csv for admins
│   ├── auth/
│   │   ├── apptoken.go              per-account write token for Claude Code inside form.md, rotated each server start
│   │   ├── password.go              argon2id hash and verify, concurrency cap, dummy hash
│   │   ├── totp.go                  TOTP check with replay step, otpauth URL, QR as SVG, recovery codes
│   │   ├── service.go               Service, session cookies, credential middleware, handler wrappers
│   │   ├── users.go                 setup, login, logout, signup, policy, /api/me, password, email, CreateAccount
│   │   ├── account.go               TOTP enrolment, API tokens, signed-in sessions
│   │   ├── oidc.go                  Google, Microsoft and generic OIDC sign-in, SSOProviders
│   │   ├── saml.go                  SAML 2.0 service provider: metadata, start, ACS, account linking, SAML-only
│   │   ├── invites.go               invitations: create, list, cancel, look up, accept
│   │   └── module.go                route table for /api/auth, /api/setup, /api/me, invites
│   ├── claudecode/                  terminals (PTY over WebSocket), Claude Code launch and chat (claude -p stream-json), window command bridge
│   ├── collection/                  databases: schema, rows, query stages, formula language, views, forms, CSV, public tables
│   ├── comment/                     threads, replies, reactions, resolve, mentions, suggested edits
│   ├── contextpack/
│   │   ├── policy.go                policy.json from the contextual-compression skill: types, validation, budgets, token estimate
│   │   ├── text.go                  words, stems, similarity, digest, outline
│   │   ├── bank.go                  the knowledge bank: record with dedupe and conflicts, verdicts, decay, search, Brief for agents
│   │   ├── pack.go                  context packs: sections, ranking, focus compression, delta, links, rendering
│   │   ├── review.go                hygiene report: stale, weak, duplicate and conflicting items
│   │   └── undo.go                  agent-run undo steps for knowledge writes
│   ├── config/
│   │   └── config.go                FORMMD_* env vars (incl. FETCH_PRIVATE, KEY_FILE, ACME_*, UPDATE_URL), defaults, validation
│   ├── crypt/
│   │   ├── key.go                   operator key: parse, KeyPath (env or systemd credential), Load with key check, HKDF subkeys
│   │   └── stream.go                sealed files: chunked AES-256-GCM writer, seekable reader, Seal/Unseal, ReadFile/WriteFile
│   ├── cryptvfs/
│   │   └── vfs.go                   SQLite shim VFS: XTS-AES-256 pages for the database, WAL and journals; temp files with a random key
│   ├── doc/
│   │   ├── schema.go                root fragment name, mark order, block ids, attribute readers
│   │   ├── room.go                  a page's CRDT: load, apply, persist, compact, awareness relay
│   │   ├── conn.go                  the /sync socket: frames, open and leave, write check, queues, identity
│   │   ├── service.go               rooms and sockets, materialiser, copier, Markdown read and write, backlinks, routes
│   │   ├── markdown.go              CRDT to Markdown, plain text, snippet, thumbnail and references
│   │   ├── parse.go                 Markdown to blocks (goldmark) and blocks to CRDT operations
│   │   ├── html.go                  CRDT to the editor's HTML for the first frame
│   │   ├── math.go                  $ and $$ in Markdown, MathML allow-list
│   │   ├── embed.go                 embed providers and the generic https embed
│   │   ├── synced.go                synced roots, cross-page first-frame rendering
│   │   ├── versions.go              page history: deflated Yjs states, list, read, save, restore record, purge
│   │   ├── blocks.go                block-addressed edits with inverses, outline, agent presence
│   │   ├── meeting.go               the meeting block's head text, shared by server and editor
│   │   └── public.go                the HTML writer's public mode for published pages
│   ├── events/
│   │   └── events.go                Hub: sequenced events, replay ring, audience filter, GET /api/events (SSE)
│   ├── family/
│   │   └── family.go                form family: descriptor, loopback lookups of formlit and formvis, task routing
│   ├── files/
│   │   ├── files.go                 uploads, sniffing, access-checked serving at /files/{id}, sweep
│   │   └── fetch.go                 POST /api/files/import: a linked file brought in with its size
│   ├── icon/
│   │   └── icon.go                  the one rule for icon values: a name from the shared set or an uploaded file
│   ├── i18n/
│   │   ├── i18n.go                  T(lang, msg, args), UserLang, LoadFS (merges the client catalogs embedded by embed.go)
│   │   └── de.go                    German for server-built mail and push text
│   ├── inbox/                       notifications, web push (VAPID), email fallback, reminders
│   ├── httpx/
│   │   ├── router.go                ServeMux wrapper, middleware chain, /api/ 404 catch-all
│   │   ├── middleware.go            request id, panic recovery, access log, security headers
│   │   ├── respond.go               JSON and error envelopes (code, message, data)
│   │   ├── request.go               APIError, Problem, Decode, client address, scheme and origin
│   │   └── ratelimit.go             per-key token buckets with a non-consuming check
│   ├── ids/
│   │   └── ids.go                   128-bit random ids, lowercase base32
│   ├── mail/
│   │   └── mail.go                  Mailer: SMTP (TLS, STARTTLS, none), OAuth sender registry
│   ├── mcp/
│   │   ├── tools_window.go          window_commands, window_command_describe, window_command_run
│   │   ├── server.go                /mcp transport: versions, headers, dispatch, errors
│   │   ├── tools.go                 tool registry, run grouping, idempotency, result framing
│   │   ├── tools_read.go            read tools
│   │   ├── tools_write.go           write tools with undo steps
│   │   ├── internal.go              Specs and Invoke for in-process agents
│   │   ├── tools_context.go         context_pack and knowledge_* tools, skill prompts and resources
│   │   └── skill.go                 GET /api/mcp/skill.zip (form-md plus contextual-compression)
│   ├── oauth/
│   │   └── oauth.go                 OAuth 2.1 for agents: metadata, DCR, CIMD, consent, codes, tokens, grants
│   ├── netguard/
│   │   └── netguard.go              the outbound HTTP client that only reaches public addresses
│   ├── order/
│   │   └── order.go                 fractional base62 sort keys: Between, After, Before, Valid
│   ├── pagestyle/                   page style validation shared by pages and the workspace default
│   ├── page/
│   │   ├── access.go                levels, the chain walk, viewer, Reach
│   │   ├── shares.go                shares, share links (resolve, grant, revoke), guest invitations, shared with me
│   │   ├── teams.go                 teamspaces and groups
│   │   ├── share_routes.go          routes for teamspaces, groups, shares and links
│   │   ├── page.go                  Meta, Service, access check, private owner, title and tag cleaning, emit
│   │   ├── tree.go                  Tree, Trash listing, Get with path, tags and tag colours
│   │   ├── write.go                 create, update, move (in and across workspaces), duplicate, trash, restore, purge
│   │   ├── extras.go                favorites with order, visits, recent pages
│   │   ├── content.go               SetContent from the materialiser, Readable, Titles
│   │   └── module.go                route table for pages, trash, tags, favorites, recent
│   ├── policy/
│   │   ├── principal.go             Principal: one shape for every credential, context helpers
│   │   └── gate.go                  Gate.Workspace access check, agent policy, emergency grants, memberships
│   ├── search/
│   │   ├── search.go                FTS5 index worker, tiered ranking, filters, GET /api/search
│   │   ├── passages.go              Markdown split into passages with heading paths
│   │   ├── sources.go               keyword OR search for AI answers, visibility-filtered
│   │   ├── context.go               Passages: any-word passages with rank, heading and page age for context packs
│   │   └── highlight.go             folding, stemming and snippet marks done in Go
│   ├── scim/                        SCIM 2.0 Users and Groups for a workspace, token management
│   ├── secrets/
│   │   └── secrets.go               AES-GCM sealing with data/secret.key, token minting and hashing
│   ├── server/
│   │   └── server.go                Build (wiring), Run, housekeeping, renderDocument, tile colours
│   ├── settings/
│   │   └── settings.go              typed instance settings table, validation, env fallback, snapshots
│   ├── site/                        published sites, custom domains, share-link pages, public files
│   ├── tunnel/
│   │   └── tunnel.go                supervised cloudflared (quick or token), status routes, tunnel-aware middleware
│   ├── ops/
│   │   ├── backup.go                Backup (VACUUM INTO plus files, tar.gz) and Restore (staged, checked, swapped beside or in place)
│   │   ├── encrypt.go               Encrypt (plain to encrypted, or seal leftovers), ResetOwner
│   │   ├── doctor.go                Doctor checks and Print
│   │   ├── lock_unix.go, lock_other.go     the data-directory lock (flock with pid; a plain file elsewhere)
│   │   ├── flush_unix.go, flush_other.go   SIGUSR1 flush request from backup to a running server
│   │   └── owner_unix.go, owner_other.go   give files back to the data directory's owner; mount-point test
│   ├── static/
│   │   ├── static.go                embedded client: rendered document, cache headers, gzip, ETag, CSP
│   │   ├── offline.go               /offline.json: core and all built assets for the service worker
│   │   └── localize.go              index.html text and labels translated once per language
│   ├── store/
│   │   ├── store.go                 SQLite writer connection plus reader pool, close with checkpoint
│   │   ├── tx.go                    Tx, Querier, ErrNotFound, NoRows, Bool
│   │   ├── migrate.go               numbered embedded migrations, one transaction each
│   │   ├── instance.go              instance id and last app version
│   │   └── migrations/
│   │       ├── 0001_instance.sql    instance key/value table
│   │       ├── 0002_accounts.sql    accounts, sessions, workspaces, tokens, invites, emergency, audit
│   │       ├── 0003_pages.sql       pages, favorites, page visits, tag colours, files
│   │       ├── 0004_docs.sql        document state, update log, references
│   │       ├── 0005_collections.sql databases, rows, views, form links
│   │       ├── 0006_search.sql      passages, the FTS5 table and its rank, index metadata
│   │       ├── 0007_sharing.sql     teamspaces, groups, page shares, share links, page invitations
│   │       ├── 0008_comments.sql    threads, comments, reactions
│   │       ├── 0009_suggestions.sql suggested edits
│   │       ├── 0010_inbox.sql       notifications, push subscriptions, notification preferences, reminders
│   │       ├── 0011_versions.sql    page history
│   │       ├── 0012_sites.sql       published sites, workspace publishing switch
│   │       ├── 0013_analytics.sql   page views, view privacy, workspace view history switch
│   │       ├── 0014_webhooks.sql    webhooks, deliveries, event cursors
│   │       └── 0015_scim.sql        SCIM tokens and provisioned users
│   ├── unfurl/
│   │   └── unfurl.go                GET /api/unfurl: page title, description, site, image and icon
│   ├── webhook/                     signed webhooks and SIEM streams: audit cursor, deliveries, retries, management
│   └── workspace/
│       ├── workspace.go             workspaces, members, access, rules, proposals, emergency log
│       └── module.go                route table for /api/workspaces
└── tools/
    ├── oc_install (+ .bat)          Go side install: go mod download
    ├── boot_backend (+ .bat)        Go side dev boot: go run on 127.0.0.1:8750
    ├── notices/
    │   ├── main.go                  generates NOTICES.md from go list and web/node_modules
    │   └── go-LICENSE               the Go licence, used when the toolchain keeps none in GOROOT
    ├── wikicheck/
    │   └── main.go                  fails when the wiki names a variable, command, route or page that does not exist
    └── skillcheck/
        └── main.go                  fails when policy.json breaks its rules or a skill file's frontmatter or routing is wrong
```

---

## Table of contents

- [embed.go](#embed-go)
- [Makefile](#makefile)
- [form-md and tools scripts](#scripts)
- [cmd/form-md/main.go](#cmd-form-md-main-go)
- [internal/app/app.go](#internal-app-app-go)
- [internal/app/version.go](#internal-app-version-go)
- [internal/app/health.go](#internal-app-health-go)
- [internal/config/config.go](#internal-config-config-go)
- [internal/httpx/router.go](#internal-httpx-router-go)
- [internal/httpx/middleware.go](#internal-httpx-middleware-go)
- [internal/ids/ids.go](#internal-ids-ids-go)
- [internal/static/static.go](#internal-static-static-go)
- [internal/store/store.go](#internal-store-store-go)
- [internal/store/migrate.go](#internal-store-migrate-go)
- [internal/store/instance.go](#internal-store-instance-go)
- [tools/notices/main.go](#tools-notices-main-go)
- [internal/store/tx.go and 0002_accounts.sql](#internal-store-accounts)
- [internal/secrets/secrets.go](#internal-secrets-secrets-go)
- [internal/settings/settings.go](#internal-settings-settings-go)
- [internal/httpx/request.go and ratelimit.go](#internal-httpx-request-go)
- [internal/policy](#internal-policy)
- [internal/audit/audit.go](#internal-audit-audit-go)
- [internal/auth](#internal-auth)
- [internal/workspace](#internal-workspace)
- [internal/mail/mail.go](#internal-mail-mail-go)
- [internal/admin/admin.go](#internal-admin-admin-go)
- [internal/server/server.go](#internal-server-server-go)
- [Phase 2 formatting and comment moves](#phase-2-housekeeping)
- [v0.2.1: icons, fonts and the system font](#v021)
- [Phase 3: pages, files and live events](#p3)
- [Phase 4: documents](#p4)
- [Phase 6: search and editor completion](#p6)
- [Phase 7: collaboration, sharing and identity](#p7)
- [Phase 8: agents and AI](#p8)
- [Phase 9: automations, charts, buttons, templates, views](#p9)
- [Phase 10: import, export and API](#p10)

---

## embed.go {#embed-go}

### embed.go · dist embed
- refers-to: embed.go :: `dist`, `Dist`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

`go:embed` cannot reach a parent directory, so the embed sits at the module root and `cmd/form-md` imports it. `all:web/dist` needs at least one file to exist, so `web/dist/.gitkeep` is committed and the Vite build re-creates it after emptying the folder (see the frontend log). Without a client build the binary still compiles and serves the API with a 503 explainer on page routes. `Dist` panics only if the embed path is wrong, which is a build-time mistake, not a runtime condition.

---

## Makefile {#makefile}

### Makefile · targets
- refers-to: Makefile :: `build`, `release`, `check`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

`make build` installs client dependencies if missing, builds the client, regenerates `NOTICES.md` and links `bin/form-md` with `CGO_ENABLED=0 -trimpath` and the version stamped through `-ldflags`. `make release` cross-compiles linux and darwin on amd64 and arm64 plus windows amd64 into `release/` as `form-md_vX.Y.Z_<os>_<arch>` with a `SHA256SUMS` file. `make check` fails on unformatted Go, vet findings or client type errors. `VERSION` is one of the version sites the bump commit edits. The binary goes to `bin/` so it can never overwrite the root `form-md` boot script.

---

## form-md and tools scripts {#scripts}

### form-md · dev boot
- refers-to: form-md :: whole script; tools/boot_backend; web/boot_frontend
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Starts the Go server with `go run` on 127.0.0.1:8750 (data in `./data` unless `FORMMD_DATA` is set) and Vite on 127.0.0.1:5750, and stops both on exit. Open the Vite port while developing; it proxies the API to Go. The dev-rules script suite is adapted for a Go root module: the Go side's `oc_install` and `boot_backend` live in `tools/` because the repo root is the Go module, and the root scripts take the command name rather than the repo name because a script literally named `form.md` reads as Markdown.

### form-md · process cleanup
- refers-to: form-md :: `trap`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

When stopping a dev session from another shell, signal the exact PIDs. A pattern match on "form-md" also matches any shell whose command line mentions the repo, including the one doing the killing.

---

## cmd/form-md/main.go {#cmd-form-md-main-go}

### cmd/form-md/main.go · run and serve
- refers-to: cmd/form-md/main.go :: `run`, `serve`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Exit codes: 0 clean, 1 runtime failure (storage, bind, serve), 2 usage or configuration error. `main` calls `os.Exit` only after `run` returns, so deferred `Store.Close` (WAL checkpoint) always runs. The module list is the backend extension seam: a new domain package exposes `Module()` and gets one line here. `static.Module()` mounts `/` and so must stay the catch-all; ServeMux precedence makes registration order irrelevant.

---

## internal/app/app.go {#internal-app-app-go}

### internal/app/app.go · Run
- refers-to: internal/app/app.go :: `Run`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Listens before serving so a busy port fails immediately and the logged URL is the real bound address. No `WriteTimeout` on purpose: the sync socket and the SSE stream (Phases 3 and 4) are long-lived; per-handler deadlines go through `http.ResponseController` instead. `ReadHeaderTimeout` guards against slow-header clients. The 15 second shutdown grace fits inside a 20 second systemd stop window with time left for the WAL checkpoint in `Store.Close`.

---

## internal/app/version.go {#internal-app-version-go}

### internal/app/version.go · Version
- refers-to: internal/app/version.go :: `Version`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Release builds set it with `-ldflags "-X github.com/nodemode-ai/form.md/internal/app.Version=vX.Y.Z"` (see `Makefile`). The default in source is bumped with each release so `go run` and `go install` report a sensible version. It is one of the version sites the bump commit touches.

---

## internal/app/health.go {#internal-app-health-go}

### internal/app/health.go · GET /healthz
- refers-to: internal/app/health.go :: `health.Mount`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Answers 200 with `status`, `product`, `version`, `storage`. A storage ping failure answers 503 with `status: degraded` so load balancers and the client's status card see it. The ping goes through the reader pool, so it never waits behind a long write.

---

## internal/config/config.go {#internal-config-config-go}

### internal/config/config.go · Load
- refers-to: internal/config/config.go :: `Load`, `Vars`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

`Vars` is the single table of env variables; `form-md help` prints it, so docs and behaviour cannot drift. Setting only one of `FORMMD_TLS_CERT` and `FORMMD_TLS_KEY` is an error rather than a silent fallback to plain HTTP. The default address is loopback; installers and the container image set `0.0.0.0:8750` explicitly.

---

## internal/httpx/router.go {#internal-httpx-router-go}

### internal/httpx/router.go · /api/ catch-all
- refers-to: internal/httpx/router.go :: `NewRouter`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Unknown `/api/...` paths answer a JSON 404 instead of falling through to the SPA's `index.html`, so a typo in a client call fails loudly. Middleware order, outermost first: request id, panic recovery, access log, security headers.

---

## internal/httpx/middleware.go {#internal-httpx-middleware-go}

### internal/httpx/middleware.go · SecurityHeaders
- refers-to: internal/httpx/middleware.go :: `SecurityHeaders`, `apiCSP`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Every response starts with a deny-all CSP; the static handler replaces it with the document CSP on HTML responses only. HSTS is sent only when the request itself arrived over TLS, so a plain-HTTP dev server never pins a browser to HTTPS.

### internal/httpx/middleware.go · recorder
- refers-to: internal/httpx/middleware.go :: `recorder.Unwrap`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

`Unwrap` lets `http.ResponseController` reach the real writer for `Flush`, hijack and deadlines. The SSE stream and the sync socket depend on it; do not remove it.

### internal/httpx/middleware.go · Recover
- refers-to: internal/httpx/middleware.go :: `Recover`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

`http.ErrAbortHandler` is re-panicked because it is the standard library's own signal to abort a response silently.

---

## internal/ids/ids.go {#internal-ids-ids-go}

### internal/ids/ids.go · New
- refers-to: internal/ids/ids.go :: `New`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

16 random bytes as unpadded lowercase base32: 26 characters, URL-safe, case-insensitive. Used for every primary key the server generates.

---

## internal/static/static.go {#internal-static-static-go}

### internal/static/static.go · buildCSP
- refers-to: internal/static/static.go :: `buildCSP`, `inlineScript`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

The document CSP allows inline scripts only by SHA-256 hash, computed at startup from the built `index.html`. The pattern matches `<script>` tags with no attributes, which is how the pre-paint theme boot is written. Adding attributes to that tag, or a second inline script, needs this pattern checked or the script is blocked.

### internal/static/static.go · New
- refers-to: internal/static/static.go :: `New`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Text assets over 512 bytes are gzip-compressed once at startup and kept in memory, which replaced a build-time compression plugin (plan revision v0.1.0). Files under `assets/` carry content hashes from Vite and are cached for a year as immutable; everything else, including `index.html`, is `no-cache` with an ETag, so a new release is picked up on the next load. Dotfiles (the `.gitkeep`) are skipped.

### internal/static/static.go · ServeHTTP
- refers-to: internal/static/static.go :: `Handler.ServeHTTP`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Order: method check, exact file match, then any path with an extension that did not match answers 404 (a missing asset must never receive `index.html`), then the SPA document for everything else. Without a client build, page routes answer 503 with a hint naming `make build` and the Vite dev port.

### internal/static/static.go · removed itoa helper
- refers-to: internal/static/static.go :: `serve` (Content-Length)
- kind: removed-code
- logged: v0.1.0 · 2026-10-07

```go
func itoa(n int) string {
	return strings.TrimSpace(strings.Replace(string(rune('0'+0)), "0", "", 1)) + formatInt(n)
}
```

A hand-rolled integer formatter written and removed in the same turn before the first commit; replaced by `strconv.Itoa`.

---

## internal/store/store.go {#internal-store-store-go}

### internal/store/store.go · Open
- refers-to: internal/store/store.go :: `Open`, `dsn`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

One writer connection (`MaxOpenConns 1`, `_txlock=immediate`) and a separate `query_only` reader pool sized to the CPU count (2 to 8). Writes queue on the single writer; reads never wait behind them. This removes the single-shared-connection deadlock class found in the reference study. The DSN deliberately has no `file:` prefix: modernc then strips its own parameters and passes the bare path to SQLite, so paths with spaces need no URI escaping. WAL is set on the writer only; readers inherit it from the file.

### internal/store/store.go · transaction reads
- refers-to: internal/store/store.go :: `Store.R`, `Store.W`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

Inside a write transaction, read through the transaction, never through `R`: the reader pool cannot see uncommitted writes and a read-modify-write split across the two would race.

### internal/store/store.go · Close
- refers-to: internal/store/store.go :: `Close`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Closes readers first, then runs `wal_checkpoint(TRUNCATE)` on the writer so the data directory holds a single self-contained `.db` file after a clean stop (verified: no `-wal` or `-shm` left behind).

### internal/store/store.go · file permissions
- refers-to: internal/store/store.go :: `Open`
- kind: todo
- logged: v0.1.0 · 2026-10-07

The data directory is created 0700, but SQLite creates the database file with the process umask (0644 observed). The directory already blocks other users; tighten the file itself to 0600 during Phase 10 hardening.

---

## internal/store/migrate.go {#internal-store-migrate-go}

### internal/store/migrate.go · migrate
- refers-to: internal/store/migrate.go :: `loadMigrations`, `migrate`, `apply`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Migrations are `NNNN_name.sql` files embedded in the binary, applied in version order, each in its own transaction together with its `schema_migrations` row, so a failed migration leaves nothing half-applied. Duplicate version numbers fail startup. Tables are `STRICT`, timestamps are integer Unix milliseconds (`NowMS`). Never edit an applied migration; add the next number.

---

## internal/store/instance.go {#internal-store-instance-go}

### internal/store/instance.go · Bootstrap
- refers-to: internal/store/instance.go :: `Bootstrap`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Creates the instance id once and records the version that last started against this database, for future upgrade checks. Verified stable across restarts.

---

## tools/notices/main.go {#tools-notices-main-go}

### tools/notices/main.go · main
- refers-to: tools/notices/main.go :: `main`, `goModules`, `webPackages`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Generates `NOTICES.md` from the modules actually linked into `./cmd/form-md` (`go list -deps`), the Go standard library licence, and the runtime `dependencies` of `web/package.json` plus Vite (its runtime helpers ship in the bundle). Development-only packages are excluded because they never ship. The file is generated output, rewritten whole by design, like a lockfile; regenerate it whenever dependencies change (`make notices`). It exits non-zero when a licence file is missing so a release cannot ship an incomplete notice.

### tools/notices/main.go · the Go licence without a GOROOT copy
- refers-to: tools/notices/main.go :: `stdlib`, `goLicense`; tools/notices/go-LICENSE
- kind: explainer
- kind: warning-context
- logged: v0.42.1 · 2026-10-20

`./build_app` failed on macOS with Homebrew Go 1.27.2: `open /opt/homebrew/Cellar/go/1.27.2/libexec/LICENSE: no such file or directory`. The Go licence is not guaranteed to sit in GOROOT (a Homebrew keg can keep it at the keg root, one level above `libexec`), and `runtime.GOROOT()` is deprecated. `stdlib` now asks `go env GOROOT`, tries `GOROOT/LICENSE` then `GOROOT/../LICENSE`, accepts a file only when it names The Go Authors (so an unrelated licence one level up is never picked), and otherwise uses `go-LICENSE`, the go1.26.9 licence embedded beside the tool. That text is byte-identical to the one `NOTICES.md` already carried, so the generated file does not change. The Go licence has been the same BSD-3-Clause text for years; if Go ever changes it, refresh `go-LICENSE` from a release's `LICENSE`.

---

## internal/store/tx.go and 0002_accounts.sql {#internal-store-accounts}

### internal/store/tx.go · Tx and Querier
- refers-to: internal/store/tx.go :: `Store.Tx`, `Querier`, `NoRows`, `Bool`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

`Tx` opens one write transaction on the writer connection (immediate lock, see `store.go`) and commits only when the function returns nil. `Querier` is the shared surface of `*sql.DB` and `*sql.Tx`, so helpers such as `audit.Log.Record` and `Gate` lookups run either standalone or inside a caller's transaction. `NoRows` turns `sql.ErrNoRows` into `store.ErrNotFound`, which handlers answer as 404. `Bool` converts to the 0/1 integers STRICT tables store.

### internal/store/migrations/0002_accounts.sql · schema decisions
- refers-to: internal/store/migrations/0002_accounts.sql :: `users`, `sessions`, `api_tokens`, `break_glass`, `audit_log`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

Exactly one owner is enforced by the partial unique index `users_single_owner`, not by application code. `password_hash` starting with `!` marks an SSO-only account: there is no password, and `VerifyLogin` sends it down the dummy-hash path so timing does not reveal it. `totp_last_step` is the replay guard: a code whose time step is not newer than the last accepted one is refused. Sessions, API tokens and invitations store only SHA-256 hashes of their secrets; `api_tokens.prefix` keeps the first characters for display. A token with `all_workspaces = 0` reaches only the rows in `api_token_workspaces`; an empty list means nothing. `break_glass.revoked_at` is 0 while a grant may run; expiry is `expires_at`. Workspace rules proposals live on the workspace row (`rules_proposal`, `_by`, `_at`), one at a time.

---

## internal/secrets/secrets.go {#internal-secrets-secrets-go}

### internal/secrets/secrets.go · Box
- refers-to: internal/secrets/secrets.go :: `Open`, `Box.Seal`, `Box.Unseal`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

The key is 32 random bytes in `data/secret.key`, created 0600 on first start. Sealed values are `v1:` plus base64 of nonce and ciphertext; the version prefix leaves room for key rotation. Losing or replacing the key file makes every sealed setting unreadable (they read back as empty and must be entered again), so backups must include it alongside the database. Phase 10 backup and restore carry both.

### internal/secrets/secrets.go · Token, Hash, Equal
- refers-to: internal/secrets/secrets.go :: `Token`, `Hash`, `Equal`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

`Token(prefix, n)` returns the prefix plus n random bytes in URL-safe base64 (API tokens use `fmd_`). `Hash` is hex SHA-256, used for every stored secret lookup; plain SHA-256 is right here because the inputs are high-entropy random tokens, not passwords. `Equal` compares in constant time.

---

## internal/settings/settings.go {#internal-settings-settings-go}

### internal/settings/settings.go · Defs and precedence
- refers-to: internal/settings/settings.go :: `Defs`, `Service.Get`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

`Defs` is the one table of instance settings, so the admin API, validation and defaults cannot drift. A value saved in the database wins, then the listed environment variable (only where the reference had one: `FORMMD_TRASH_DAYS`, `FORMMD_UPDATE_CHECK`), then the default. Secrets are sealed with `secrets.Box`. `workspaces.member_create` (default true) was added in v0.2.0 for parity with the reference's "users may create their own workspaces".

### internal/settings/settings.go · secrets in snapshots and writes
- refers-to: internal/settings/settings.go :: `SecretSet`, `Snapshot`, `Set`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

`Snapshot` never returns a secret: a stored one reads as the sentinel `SecretSet` (`"\x00set"`), an unset one as empty. `Set` skips a secret whose incoming value is the sentinel (unchanged) and stores an empty string to clear it. The admin client relies on exactly this: it sends the sentinel back for untouched secret fields and an empty value only after "Remove".

---

## internal/httpx/request.go and ratelimit.go {#internal-httpx-request-go}

### internal/httpx/request.go · APIError and Problem
- refers-to: internal/httpx/request.go :: `APIError`, `Fail`, `APIError.With`, `Problem`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Handlers return errors; `Problem` writes an `APIError` as `{"error":{"code","message","data"}}` with its status, maps `store.ErrNotFound` to 404, and logs anything else as a 500 without leaking it. Codes are stable identifiers the client translates (`web/src/ui/errors.ts`); messages are English fallbacks. `With` attaches data, for example the offending settings key.

### internal/httpx/request.go · Decode
- refers-to: internal/httpx/request.go :: `Decode`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

Bodies are capped at 8 MiB and unknown fields are refused, so a misspelt field fails loudly instead of being ignored. An empty body decodes as the zero value.

### internal/httpx/ratelimit.go · Limiter
- refers-to: internal/httpx/ratelimit.go :: `Limiter.Allow`, `Exhausted`, `Spend`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

Token buckets per key, in memory. `Exhausted` checks without consuming and `Spend` consumes after the fact, so the bad-token limiter is fed only by failures and a valid agent is never slowed by its own traffic. Full buckets are swept at most once a minute, which bounds memory to the keys active in the last minute or so. Sign-in uses 30 per minute with a burst of 10 per address; bad tokens 60 per minute with a burst of 20.

---

## internal/policy {#internal-policy}

### internal/policy/principal.go · Principal
- refers-to: internal/policy/principal.go :: `Principal`, `Reaches`, `CanWrite`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

The reference's best idea in this area, kept: every credential resolves to one shape carrying the person's instance role, the credential kind (session, token, oauth), scope and an optional workspace list. A credential can only narrow the person behind it: `Reaches` checks the list, `CanWrite` the scope. `AgentName` and `AgentRun` stay empty until the agent surfaces fill them in Phase 8 (from the MCP client's own description of itself); they are labels for the activity log and presence, never permissions.

### internal/policy/gate.go · Gate.Workspace
- refers-to: internal/policy/gate.go :: `Gate.Workspace`, `AgentAllowed`, `ErrHidden`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Order: the credential must reach the workspace, then a membership gives the role; without one, the owner's running emergency grant gives a read-only viewer access marked `BreakGlass`; then the workspace agent policy applies to non-session credentials (open: any; strict: OAuth only; closed: none). Every refusal is `ErrHidden`, answered as 404, so a workspace's existence is not revealed.

---

## internal/audit/audit.go {#internal-audit-audit-go}

### internal/audit/audit.go · Record
- refers-to: internal/audit/audit.go :: `Log.Record`, `Entry`, `Change`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Pass the open transaction as the querier so the entry commits or rolls back with the change it describes; nil writes through the writer connection. The actor kind defaults from the principal (human for sessions, agent for tokens and OAuth). Retention (`audit.retention_days`) is enforced by the Phase 7 activity log sweep.

---

## internal/auth {#internal-auth}

### internal/auth/password.go · argon2id
- refers-to: internal/auth/password.go :: `HashPassword`, `VerifyPassword`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

argon2id with m=64 MiB, t=1, p=4 and a semaphore of 4 concurrent hashes, so a burst of sign-ins cannot exhaust memory. An empty stored hash verifies against a fixed dummy hash, giving unknown accounts the same timing as real ones. Parameters are stored in each hash, so they can be raised later without breaking old hashes.

### internal/auth/totp.go · CheckTOTP
- refers-to: internal/auth/totp.go :: `CheckTOTP`, `totpSkew`, `NewRecoveryCodes`, `QRSVG`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Accepts the current 30-second step and one on either side, and only a step newer than the last accepted one, so a code cannot be replayed. Recovery codes are ten `xxxxx-xxxxx` codes stored hashed and deleted on use; input is normalised (case, spaces, missing dash). The QR code is rendered on the server as SVG with rsc.io/qr, so the secret never goes to a third-party service.

### internal/auth/service.go · Middleware
- refers-to: internal/auth/service.go :: `Middleware`, `originOK`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

Resolves a bearer token first, then the session cookie, into a `Principal` on the request context. Unsafe methods without a bearer token need an `Origin` matching the request origin or the configured public address; a missing `Origin` is allowed because browsers always send it on cross-site writes, and the cookie is SameSite=Lax as a second line. Read-only tokens are refused on unsafe methods everywhere except `/mcp`, whose JSON-RPC calls are checked per tool in Phase 8.

### internal/auth/service.go · RevokeCredentials
- refers-to: internal/auth/service.go :: `RevokeCredentials`, `OnRevoke`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Deletes sessions (optionally keeping one) and API tokens, then calls the `OnRevoke` hooks so later phases can drop OAuth grants, calendar feeds and open sockets for the account. Used by password changes, deactivation, ownership hand-over and admin password resets.

### internal/auth/users.go · VerifyLogin
- refers-to: internal/auth/users.go :: `VerifyLogin`, `checkSecondFactor`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Wrong email and wrong password give the same `bad_credentials` error. A correct password on an account with two-factor and no code answers `totp_required` (401), which the sign-in screen turns into the code step. Deactivated accounts answer `account_disabled` only after a correct password.

### internal/auth/users.go · CreateAccount
- refers-to: internal/auth/users.go :: `CreateAccount`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

Validates and creates an account inside the caller's transaction on behalf of an admin. The admin vouches for the address, so it counts as verified; that matters later for SSO email matching. (Moved here from a source comment.)

### internal/auth/oidc.go · sign-in flow
- refers-to: internal/auth/oidc.go :: `handleSSOStart`, `handleSSOCallback`, `verifyIDToken`, `ssoAccount`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Start hops to the public address first when one is set, so the state cookie lands on the host the provider will return to. State, nonce and the PKCE verifier travel in one sealed cookie scoped to `/api/auth/sso`. The ID token is verified against the issuer's published keys (RS256 or ES256), issuer (Microsoft's multi-tenant issuer is matched with the token's `tid`), audience, expiry and nonce. An existing identity link signs in; otherwise an account with the same email is linked only if both sides are verified (Microsoft uses `xms_edov`), else sign-up policy decides. Failures return to `/login?sso_error=<code>`.

### internal/auth/oidc.go · SSOProviders
- refers-to: internal/auth/oidc.go :: `SSOProviders`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Lists configured providers in display order (Google, Microsoft, other OIDC). It is the single source for the policy endpoint, the document's `data-sso` attribute and the boot block; a provider counts as configured only when it has everything sign-in needs (the generic one needs an issuer). Replaces `server.ssoNames` (see removed code below). (Moved here from a source comment.)

### internal/auth/invites.go · invitations
- refers-to: internal/auth/invites.go :: `handleCreateInvite`, `handleAcceptInvite`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Only admins of a shared workspace invite. The link is always returned (and copied by the client) whether or not mail is set up; when mail is on and an email was given, it is also sent. Acceptance paths: signed in (joins as that account, unless the invite is bound to another email), existing account (password plus second factor), or a new account. An invite bound to an email marks that account's email verified, because only its recipient could have used it. Each invite works once and expires after 14 days.

---

## internal/workspace {#internal-workspace}

### internal/workspace/workspace.go · Provision and personal spaces
- refers-to: internal/workspace/workspace.go :: `Provision`, `JoinInvite`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Every account gets a workspace named "Personal" with `personal = 1`; the owner also gets "Team". Workspaces with `auto_join` are joined as member at account creation. Personal spaces never take other members, cannot be deleted on their own (they go with the account) and are never reachable by emergency access.

### internal/workspace/workspace.go · SetCreatePolicy and MayCreate
- refers-to: internal/workspace/workspace.go :: `SetCreatePolicy`, `MayCreate`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Decides who may create shared workspaces; without a policy, everyone may. The server sets it to "admins, or anyone while `workspaces.member_create` is on", and `MayCreate` feeds `can_create_workspaces` in `/api/me` so the client hides the action. (Moved here from source comments.)

### internal/workspace/workspace.go · SetAccess
- refers-to: internal/workspace/workspace.go :: `SetAccess`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Grants, changes or, with an empty role, removes a membership on behalf of an instance admin, inside the caller's transaction. The last-admin guard still holds and personal spaces are never touched. Returns the role held before, empty when there was none, so the caller audits only real changes. (Moved here from a source comment.)

### internal/workspace/workspace.go · Emergency, EmergencyLog, EndEmergency
- refers-to: internal/workspace/workspace.go :: `Emergency`, `EmergencyLog`, `EndEmergency`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

`Emergency` is one emergency read grant the instance owner took on a workspace. `EmergencyLog` lists every grant on a workspace, newest first, capped at 200, for its admins. `EndEmergency` lets a workspace admin end a running grant early and audits it. Parity with the reference's per-workspace emergency access log. (Moved here from source comments.)

---

## internal/mail/mail.go {#internal-mail-mail-go}

### internal/mail/mail.go · Mailer
- refers-to: internal/mail/mail.go :: `Mailer.Send`, `Register`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Reads settings on every send, so changes apply without a restart. `mail.mode` picks SMTP or a registered OAuth sender; `Register` is the seam for the Gmail and Microsoft Graph senders in Phase 10. Messages are plain text, quoted-printable encoded, with a generated Message-ID and Date.

---

## internal/admin/admin.go {#internal-admin-admin-go}

### internal/admin/admin.go · who may do what
- refers-to: internal/admin/admin.go :: `target`, `updateUser`, `createUser`, `setAccess`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Admins list users, create members, deactivate and reactivate members, and set anyone's access to shared workspaces except their own (an admin cannot grant themselves a way in; the owner has emergency access for that). Only the owner changes instance roles, creates admins, changes another admin, resets second factors, sets passwords, deletes accounts, hands over ownership, takes emergency access and handles stranded workspaces. `target` refuses acting on yourself and on the owner.

### internal/admin/admin.go · deletion plan
- refers-to: internal/admin/admin.go :: `deletionPlan`, `deleteUser`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

The plan is computed before anything is deleted and shown to the owner: the personal space is purged, each shared workspace where the person was the last admin gets its longest-standing member promoted (members before viewers), and a workspace left with nobody goes to the owner. Deletion re-computes the plan inside the transaction so it cannot act on a stale one. `OnUserDelete` hooks let later phases clean up their own rows.

---

## internal/server/server.go {#internal-server-server-go}

### internal/server/server.go · renderDocument contract
- refers-to: internal/server/server.go :: `renderDocument`, `authAttrs`, `currentWorkspace`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

The rendered document is a contract with the client. Attributes: `data-auth` (setup, signed-out, signup, invite, app), `data-sso` (space-separated provider ids) and `data-signup` (open or closed). Slots: instance-name, user-name, user-initials, user-color, ws-name, ws-initial, ws-color. Boot block: `setup`, `sso`, `signup`, `instance_name`, `me`, `workspace_id`. The current workspace is the `formmd_ws` cookie when it names a workspace in the list, else the first shared one, else the first; `pickWorkspace` in `web/src/engine/index.ts` must choose the same, or the first frame shows one workspace and the next another. Signed-in visits to `/login` or `/signup` redirect to `/`; any path redirects to `/` while setup is pending.

### internal/server/server.go · TileColor
- refers-to: internal/server/server.go :: `TileColor`, `Initials`, `firstInitial`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

Hash `h = h*31 + rune` over the name, modulo the ten palette colours in order. `tileColor` in `web/src/ui/avatar.ts` is the same function with unsigned 32-bit wrap; change both together or tiles change colour after the first paint. Initials and the first initial are rune-safe.

### internal/server/server.go · removed ssoNames
- refers-to: internal/server/server.go :: `authAttrs`, `renderDocument`
- kind: removed-code
- logged: v0.2.0 · 2026-10-07

```go
func (s *Server) ssoNames() []map[string]string {
	out := []map[string]string{}
	names := map[string]string{"google": "Google", "microsoft": "Microsoft", "oidc": s.Settings.Get("sso.oidc.name")}
	for _, id := range []string{"google", "microsoft", "oidc"} {
		if s.Settings.Has("sso." + id + ".client_id") {
			n := names[id]
			if n == "" {
				n = "Single sign-on"
			}
			out = append(out, map[string]string{"id": id, "name": n})
		}
	}
	return out
}
```

Removed before the first commit. It duplicated `auth.SSOProviders` with a weaker test (client id only), so a provider missing its secret or issuer showed a button that could not work. `authAttrs` built `data-sso` the same way and now uses `SSOProviders` too.

### internal/server/server.go · housekeeping
- refers-to: internal/server/server.go :: `housekeeping`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Every 30 minutes: expired sessions are swept and invitations unused 30 days past expiry are deleted. Later phases add the trash purge and the activity log retention here.

### cmd/form-md/main.go · embedded time zones
- refers-to: cmd/form-md/main.go :: import `time/tzdata`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

Time zone preferences are validated with `time.LoadLocation`, which needs zoneinfo. Container images and minimal hosts often have none, so the binary embeds the database (about 450 KB) to stay self-contained.

### internal/static/static.go · Document and RenderIndex
- refers-to: internal/static/static.go :: `Document`, `RenderIndex`, `serveDocument`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

`index.html` is a template with three hooks: `data-auth="pending"` on the html tag becomes the sorted `data-*` attributes, `<!--slot:key-->` comments become escaped text, and the `<script type="application/json" id="boot">null</script>` placeholder becomes the boot JSON. `json.Marshal` escapes `<`, `>` and `&` as `<`, `>` and `&` by default, so no value can close the script tag; keep `encoding/json` with HTML escaping on for this. The renderer may also return a redirect. Rendered documents are `Cache-Control: no-store` because they carry the account. The boot block has a `type` attribute, so the inline-script CSP hash pattern (attribute-less scripts only) does not see it, and it is never executed.

---

## Phase 2 formatting and comment moves {#phase-2-housekeeping}

### gofmt alignment fixes
- refers-to: internal/auth/oidc.go :: `b64`; internal/mail/mail.go :: `Mailer`; internal/workspace/workspace.go :: `Rules`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Three whitespace-only corrections found by `gofmt -l` before the first Phase 2 commit: a one-line function body split onto its own lines, and struct field alignment after a field was removed from `Rules` and from `Mailer`. No behaviour change.

### doc comments moved out of source
- refers-to: internal/auth/users.go :: `CreateAccount`; internal/auth/oidc.go :: `SSOProviders`; internal/workspace/workspace.go :: `SetCreatePolicy`, `MayCreate`, `SetAccess`, `Emergency`, `EmergencyLog`, `EndEmergency`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

These functions were briefly written with Go doc comments; the text now lives in the entries above and the source is comment-free again.

### internal/static/static.go · removed no-op boot escape
- refers-to: internal/static/static.go :: `RenderIndex`
- kind: removed-code
- logged: v0.2.0 · 2026-10-07

```go
boot = bytes.ReplaceAll(boot, []byte("<"), []byte(`<`))
```

Replaced `<` with itself, so it did nothing; the escaping it meant to do is already done by `json.Marshal` (see the Document and RenderIndex entry). Removed in the Phase 2 review, after the wiring commit.

### cmd/form-md/main.go · usage lists setting defaults
- refers-to: cmd/form-md/main.go :: `usage`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

After the start-up variables, `help` lists every instance setting that has an environment default (from `settings.Defs`), so the README table and the binary agree. These variables only supply defaults; a value saved in Instance settings wins.

---

## v0.2.1: icons, fonts and the system font {#v021}

### internal/icon/icon.go · Valid
- refers-to: internal/icon/icon.go :: `Valid`, `pattern`
- kind: rationale
- logged: v0.2.1 · 2026-10-07

Your rule from open question 7: zero emojis. An icon value is empty (the default glyph), `i:<name>` or `i:<name>:<colour>` from the form.md icon set (lowercase words joined by hyphens, colour from the ten palette names), or `f:<id>` for an uploaded image (a 26-character id from `internal/ids`). Nothing else is stored, so no emoji or other free text can become an icon through the API, an agent or an import. Workspaces use it now (`cleanIcon`); pages use it in Phase 3. The Phase 9 importers map an emoji icon from another tool to the default icon rather than storing it.

### internal/workspace/workspace.go · removed icon length check
- refers-to: internal/workspace/workspace.go :: `cleanIcon`
- kind: removed-code
- logged: v0.2.1 · 2026-10-07

```go
if utf8.RuneCountInString(raw) > 200 {
	return "", httpx.Fail(http.StatusBadRequest, "bad_icon", "icon value too long")
}
return strings.TrimSpace(raw), nil
```

Accepted any text up to 200 characters, emoji included. Replaced by `icon.Valid`.

### internal/server/server.go · data-font
- refers-to: internal/server/server.go :: `renderDocument` (signed-in branch)
- kind: explainer
- logged: v0.2.1 · 2026-10-07

When the signed-in person chose the system font, the document carries `data-font="system"`, which the tokens file turns into the system font stacks before first paint. Absent attribute means Inter and JetBrains Mono.

### internal/static/static.go · fontPreload
- refers-to: internal/static/static.go :: `fontPreload`, `RenderIndex`
- kind: warning-context
- logged: v0.2.1 · 2026-10-07

For a document rendered with `data-font="system"`, every `<link rel="preload" ... as="font" ...>` is removed, so people on the system font never download Inter. The pattern needs `rel="preload"` first and `as="font"` later in the same tag, which is how `web/index.html` writes it and how Vite keeps it after hashing the file name. Reorder those attributes and the preload stays for everyone (wasted bytes, not a visible fault).

### tools/notices/main.go · fonts and OFL
- refers-to: tools/notices/main.go :: `fontFiles`, `fonts`, `classify`
- kind: explainer
- logged: v0.2.1 · 2026-10-07

The font files are not a Go module or an npm dependency (they were copied from `@fontsource-variable/inter` and `@fontsource-variable/jetbrains-mono` 5.3.0 into `web/src/fonts/` with their licence texts), so `fontFiles` lists them by hand: update the version there when the files are refreshed. The OFL check comes first in `classify` because the OFL's permission paragraph also starts with "Permission is hereby granted, free of charge", which the MIT check matched.

---

## Phase 3: pages, files and live events {#p3}

### internal/order/order.go · key shape
- refers-to: internal/order/order.go :: `Between`, `midpoint`, `After`, `Before`, `Valid`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Sibling order is a base62 string compared bytewise. `Between(a, b)` returns a key strictly between, with `""` meaning open on that side. A key never ends in `0`, so there is always room below it, and nothing is ever renumbered: a reorder writes one row. When two neighbours are adjacent digits the result grows by one character; repeated inserts at the same spot lengthen keys slowly (about one character per 30 inserts), which is fine for sidebars. The client has a port in `web/src/engine/order.ts` for optimistic placement only; the server always computes the stored key from `before_id`/`after_id`.

### internal/store/migrations/0003_pages.sql · schema notes
- refers-to: internal/store/migrations/0003_pages.sql
- kind: rationale
- logged: v0.3.0 · 2026-10-07

`trashed_at` is 0 for live pages (not NULL) so live lookups use plain equality on the index. `trash_batch` groups a subtree trashed together; restore brings back exactly that batch, never descendants trashed earlier on their own. `parent_id` cascades, so deleting a batch root deletes its whole subtree. `files.page_id` is set null when a page goes, so an attachment survives as an orphan until the sweep. `tag_colors.tag` is NOCASE so `Plan` and `plan` share a colour. `page_visits` keeps one row per user and page, updated in place.

### internal/page/page.go · private pages
- refers-to: internal/page/page.go :: `privateOwner`, `check`; internal/page/tree.go :: `visibleIn`
- kind: rationale
- logged: v0.3.0 · 2026-10-07

A private page and everything under it is readable by its owner only. `privateOwner` walks the ancestor chain: no private ancestor gives `""`, one owner gives that owner, two different owners give `"-"`, which matches nobody. Private pages stay hidden from workspace admins and from emergency access too; that is stricter than the reference on purpose. Personal spaces are private as a whole, so setting a page there to private is refused as pointless. Every unreadable page answers 404 like a missing one.

### internal/page/write.go · Move and moveAcross
- refers-to: internal/page/write.go :: `Move`, `moveAcross`, `republish`
- kind: warning-context
- logged: v0.3.0 · 2026-10-07

A same-workspace move checks for cycles with a recursive query and republishes the subtree when its privacy owner changed: everyone but the new owner gets `page_removed`, then each page is emitted with its own owner (never the root's, which would leak a deeper private page). A move to another workspace needs write access on both sides, takes the whole subtree and its files along, and is refused with `hidden_subpages` when the subtree holds someone else's private page: moving it would hand that page to a workspace where its owner may not be. Audit records `page.move_out` in the source and `page.move_in` in the target.

### internal/page/write.go · Duplicate
- refers-to: internal/page/write.go :: `Duplicate`, `DuplicateInput`; internal/page/page.go :: `OnDuplicate`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Copies only what the caller can see, beside the original. The copy's root is titled from the request (the client sends a translated "Copy of …") or "Copy of " plus the title, cut to 500 characters. Copies are never templates; the copier owns them. Phase 4 registers a copier through `OnDuplicate` to copy document content inside the same transaction.

### internal/page/write.go · TrashPage, Restore, Purge
- refers-to: internal/page/write.go :: `TrashPage`, `Restore`, `DeleteForever`, `EmptyTrash`, `Purge`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Trashing stamps the live subtree (including pages the caller cannot see, as the reference does) with one batch id. Restore clears that batch only and moves the page to the top level when its parent is no longer live. Restoring a page from the middle of a batch is allowed. Delete forever and empty trash delete batch roots and let the cascade take the rest. `Purge` runs from housekeeping every 30 minutes, deletes batch roots older than `trash.days` (0 disables) and audits each as `system`.

### internal/page/extras.go · favorites
- refers-to: internal/page/extras.go :: `SetFavorite`, `FavoriteInput`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

`PUT /api/favorites/{id}` with no body adds at the end and is a no-op for an existing favorite; `after_id` places it after another favorite, `top` places it first. Favorites span workspaces and list only pages the caller can still read. Changes publish a `user` event so the person's other tabs follow.

### internal/files/files.go · uploads
- refers-to: internal/files/files.go :: `Upload`, `store`, `authorize`
- kind: warning-context
- logged: v0.3.0 · 2026-10-07

The multipart form must send its text fields before the file part, since the body is streamed and never buffered. The limit comes from `files.max_upload_mb` on every request. The type is sniffed from content, not trusted from the client. Covers, icons and pictures must be PNG, JPEG, GIF or WebP; SVG is accepted for icons and workspace pictures only, and every file is served with a sandboxing CSP and `nosniff`, so a script inside an SVG never runs. Avatars belong to their uploader and to no workspace.

### internal/files/files.go · serving and sweep
- refers-to: internal/files/files.go :: `Serve`, `Sweep`, `Owned`, `InWorkspace`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

`/files/{id}` checks read access to the page, else the workspace, else (avatars) any signed-in account. The id is the ETag and the content never changes, so the cache is private and immutable. The sweep deletes rows nothing points at after a day (page gone, avatar or workspace picture replaced) and then files on disk with no row after an hour; it is the only place a deleted workspace's files leave the disk.

### internal/events/events.go · delivery and replay
- refers-to: internal/events/events.go :: `Hub`, `publish`, `audience`, `subscribe`, `Handler`
- kind: rationale
- logged: v0.3.0 · 2026-10-07

Every event gets a sequence number and goes into a ring of 2048. A client resumes with `Last-Event-ID` (or `?after=`); a cursor still inside the ring is replayed, one older or newer than the server's counter (after a restart) gets `reset: true` in the hello so the client refetches. Workspace events reach members and active emergency holders whose credential reaches the workspace and passes its agent policy; private page events reach their owner only; `Except` skips one user (the new owner during a republish). A subscriber whose queue of 128 fills up is dropped and reconnects with its cursor, rather than slowing everyone. A ping every 20 seconds keeps proxies and the client watchdog satisfied. Revoking a user's credentials closes their streams; shutdown closes all of them so a graceful stop does not wait out its grace period.

### internal/server/server.go · boot document
- refers-to: internal/server/server.go :: `renderDocument`, `landing`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

The signed-in document carries the event cursor (taken before any data is read, so replay may repeat an event but never miss one), the current workspace's tree, tags and the person's favorites, and for `/p/{id}` the page with its path. A page link selects the page's own workspace. `/` redirects to the last page visited in the workspace, else its first top-level page, so the first paint is never an empty home that the client would then replace. `data-route` and the tile and avatar kinds are attributes so CSS shows the right view and hides placeholder letters before the client runs.

### internal/app/app.go · Stopper
- refers-to: internal/app/app.go :: `Stopper`, `Run`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

A module that implements `Stop()` is registered with `RegisterOnShutdown`. Only the events module does today.

### internal/workspace · notices
- refers-to: internal/workspace/workspace.go :: `Notifier`, `Changed`, `MemberIDs`; internal/workspace/module.go; internal/admin/admin.go
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Handlers call `Changed` after a successful write: settings, rules and members notify the workspace; added, removed and regraded people, members of a deleted workspace, and the owner after an emergency grant or its end get a `user` notice so their workspace list refreshes. Member ids of a workspace about to be deleted are read before the delete.

---

## Phase 4: documents {#p4}

### internal/doc · why the server owns the CRDT
- refers-to: internal/doc/room.go; internal/doc/service.go :: `Write`
- kind: rationale
- logged: v0.4.0 · 2026-10-08

Every update a browser sends is applied to the room's own ygo document before it is stored or relayed, so the server can read, render, export and edit any page without a browser open. Server-side writes (`Write`, later agents and imports) are ordinary CRDT transactions with an origin, broadcast like a person's typing: nothing is reset, nobody reloads, and concurrent typing survives. This is the decisive difference from the reference, whose server stores bytes it cannot read.

### internal/doc/room.go · persistence and compaction
- refers-to: internal/doc/room.go :: `onUpdate`, `flush`, `loadRoom`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

`onUpdate` runs inside the room lock (ygo calls it at the end of each transaction), relays the update to every other subscriber, queues it, and arms a 300 ms flush timer. `flush` takes the queue under the lock, numbers it, and when 400 updates or 1 MiB have piled up since the last snapshot it encodes the full state in the same critical section, so the snapshot covers exactly the rows it replaces; the database writes happen outside the room lock under `persistMu`. A failed flush puts its batch back in front of the queue. A foreign-key failure means the page was deleted and the batch is dropped. A room whose fragment is empty gets one paragraph from the server at load, so two editors opening a new page never each create a first paragraph. Empty V1 updates are two bytes and are ignored.

### internal/doc/conn.go · the socket protocol
- refers-to: internal/doc/conn.go :: `frame`, `handle`, `open`, `syncMessage`; web/src/engine/doc/sync.ts
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

Every frame is a varuint kind, then for kinds 0 (y-protocols sync), 1 (awareness) and 3 (query awareness) a varstring page id and the raw payload; kind 2 is a JSON control message (`open`, `close`, `ping` from the client; `opened` with generation and write flag, `denied`, `refused`, `closed`, `error`, `ping` from the server). On `open` the server sends `opened`, its step 1 and the awareness snapshot under the room lock so later broadcasts cannot overtake them. The client side in `sync.ts` and `docs.ts` must agree byte for byte. Close codes: 4001 signed out (the client does not reconnect), 4002 fell behind, 4003 server stopping.

### internal/doc/conn.go · permission checks on a long-lived socket
- refers-to: internal/doc/conn.go :: `mayWrite`, `recheck`, `sameOrigin`
- kind: rationale
- logged: v0.4.0 · 2026-10-08

Write permission is decided per update frame through `pages.CanWrite`, cached for three seconds so typing does not query the database on every keystroke; a refusal is reported once per page as `refused`. Read access is rechecked every 20 seconds and a page that became hidden is closed with `closed`. Sign-out and deactivation close every socket of the account through the revoke hook. The Origin header must equal the request's own origin (proxy headers honoured when trusted), so `InsecureSkipVerify` on the library only means the check is ours. Ending one session from the devices list still leaves its socket open until the next read recheck, the same known gap as the event stream.

### internal/doc/room.go · awareness
- refers-to: internal/doc/room.go :: `applyAwareness`, `removeSub`; internal/doc/conn.go :: `identity`, `tintHex`
- kind: rationale
- logged: v0.4.0 · 2026-10-08

Awareness is relayed by our own code rather than a library object: a client id belongs to the first socket that sent it, entries from another socket for that id are ignored, the clock must not go backwards, states are capped at 8 KiB and 64 clients per page, and the `user` field is replaced with the account's id, name, palette name (`tint`) and its hex colour, so nobody can appear as someone else. When a socket leaves, its ids are broadcast as removed with the clock bumped, so a closed tab's cursor vanishes at once.

### internal/doc/service.go · materialiser
- refers-to: internal/doc/service.go :: `materialise`, `writeRefs`; internal/page/content.go :: `SetContent`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

1.5 s after the last update (at most 10 s while someone keeps typing) the room flushes, then summarises its fragment to Markdown, plain text, a 240-character snippet, the first picture as thumbnail, and page, file and person references. `doc_state.markdown` and `doc_refs` are rewritten; `pages.snippet`, `thumb`, `updated_at` and `updated_by` change and a page event goes out, so trees and notes cards update. Page links in `doc_refs` are what backlinks read; file references will keep attachments from being swept once they are referenced from another page (not wired into the sweep yet). Rooms unload 60 s after their last subscriber, after a final materialise.

### internal/doc/markdown.go · Markdown dialect
- refers-to: internal/doc/markdown.go; internal/doc/parse.go; web/src/extensions/blocks/markdown.ts
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

Both directions use the same conventions, and the client's writer mirrors the server's: callouts are GitHub alerts (`> [!NOTE]`, tone mapped to NOTE, TIP, IMPORTANT, WARNING, CAUTION), toggles are `<details>` with an optional `<h1>` to `<h3>` inside the summary, empty paragraphs are `<br>`, the table of contents is `<!-- toc -->`, underline is `<u>`, page mentions are links to `/p/<id>` (parsed back into mention chips), files are links or images pointing at `/files/<id>` (the parser looks the id up to choose image, video, audio or file), and a paragraph holding only an autolink becomes a link card or, for YouTube and Vimeo, an embed. Colours, highlights, icons and column layout do not survive a round trip. Marks open and close by common prefix in the fixed order link, strong, em, strike, underline, code, so runs that share marks do not produce `****`.

### internal/doc/parse.go · writing text into the CRDT
- refers-to: internal/doc/parse.go :: `writeNode`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

ygo's `YText.Insert` inherits the formatting at the cursor when `attrs` is nil or empty (unlike Yjs, where `{}` clears it), so each run of text is inserted unformatted in one piece and marks are applied afterwards with `Format` over UTF-16 ranges. Elements are attached to their parent before their attributes and children are written. Integer attributes are stored as float64 so they read back as numbers in the browser.

### internal/doc/html.go · first-frame HTML must match the editor
- refers-to: internal/doc/html.go; web/src/engine/doc/schema.ts :: `toDOM`; web/src/extensions/blocks/nodeviews.ts
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

The server writes the same tags, classes and attributes as the schema's `toDOM` and the node views, including the empty icon boxes that the client fills, `ProseMirror-trailingBreak` in empty text blocks and `width` and `height` on pictures for reserved space. Style attributes are never written because the document CSP forbids them. Change a block's DOM in all three places together, or the editor's first render will shift the page.

### internal/doc/service.go · Copy inside the duplicate transaction
- refers-to: internal/doc/service.go :: `Copy`; internal/page/page.go :: `OnDuplicate`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

The copier runs inside the page duplicate transaction on the single writer connection, so it must not call `flush` (which opens its own transaction and would deadlock). A loaded room is encoded from memory, which includes unflushed updates; otherwise snapshot and log are read through the same transaction.

### internal/app/app.go · Drainer
- refers-to: internal/app/app.go :: `Drainer`, `drainGrace`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

`Stop` hooks run when shutdown starts and close sockets; `Drain` runs after the HTTP server has returned, with its own four-second budget, and the documents module uses it to flush and materialise every open room before `main` closes the store.

### internal/static/static.go · html slots
- refers-to: internal/static/static.go :: `htmlSlotPattern`, `Document.HTML`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

`<!--html:name-->` is replaced with trusted server-made markup without escaping, after the escaped text slots, and only the documents service fills one (`doc-body`). Never put user text into `Document.HTML` without escaping it first.

### tools/notices/main.go · every bundled package
- refers-to: tools/notices/main.go :: `webPackages`, `licenseFiles`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

Web packages come from `package-lock.json`: every entry under `node_modules/` that is not marked dev, so transitive packages that end up in the bundle (entities, linkify-it, mdurl, orderedmap, w3c-keyname and others) are listed with their licence text. Without a lock file it falls back to the direct dependencies. argparse is listed because markdown-it depends on it, though only markdown-it's command line uses it.

### internal/page/page.go · databases and rows are pages
- refers-to: internal/page/page.go :: `Hooks`, `OnCreate`, `notify`; internal/page/write.go :: `Create`, `Duplicate`, `moveAcross`; internal/page/tree.go :: `workspacePages`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

A database is a page with `kind = 'collection'`; each row is a page with `is_row = 1` whose parent is the database. Rows never appear in the tree, recents or the sidebar count (`workspacePages(..., withRows=false)`). The collection service registers four hooks: creators (insert the `collections` row and the first view inside the page create transaction), remaps (rewrite relation ids and copy views and row values inside the duplicate transaction), movers (record the new workspace on the `collections` row when a database crosses workspaces; its rows move with it as children) and watchers (trash, restore and delete drop the snapshot and emit row events). Hook notifications are deferred until the transaction commits so an event never describes a write that rolled back.

### internal/collection/cache.go · one snapshot per database
- refers-to: internal/collection/cache.go :: `snap`, `loadSnap`, `maxCached`, `refresh`, `ordered`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

Queries read an in-memory snapshot of every row of a database (values decoded once), kept for the 64 most recently used databases. Writes update the snapshot in place after commit (`put`, `drop`); `refresh` re-sorts under the write lock, never under the read lock, because `ordered()` is read concurrently. A snapshot that cannot be trusted is dropped and rebuilt on the next read, never patched.

### internal/collection/eval.go · lazy values
- refers-to: internal/collection/eval.go :: `evaluator`, `value`, `visiting`, `memo`; internal/collection/formula/eval.go :: `maxSteps`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

Formula and rollup cells are computed on demand per query, memoised per row and property, so a filter on one formula does not compute every other one. A cell that is reached again while it is being computed is a cycle: it yields empty and records "this refers back to itself" as the cell's error. One formula run is capped at 200,000 steps.

### internal/collection/results.go · result cache
- refers-to: internal/collection/results.go :: `maxResults`; internal/collection/query.go :: `Query`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

The ordered, filtered id list for a view is cached by epoch, user, database and a hash of the effective config (view config plus search, range and extra filter), 24 entries. Paging through a view therefore sorts once; any write bumps the epoch and every entry goes stale. Rows are cut from the cached list per window.

### internal/collection/service.go · two-way relations
- refers-to: internal/collection/service.go :: `syncPairs`, `applyValues`, `finishWrite`; internal/collection/schemaops.go :: `addPair`, `dropPair`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

A two-way relation is two properties that name each other in `Pair`. A write to one side updates the other side of every added or removed target inside the same transaction, and every touched database gets its own event after commit. Deleting one side, switching it to one-way (`two_way` false) or pointing it at another database removes the paired property from the other database.

### internal/collection/schemaops.go · type changes keep what they can
- refers-to: internal/collection/schemaops.go :: `convert`, `displayText`, `pruneOptions`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

Changing a property's type rewrites each row through the value's display text, so text to number, select to text and the reverse keep what makes sense and drop the rest. Formulas are compiled against the schema before they are saved; a failure returns `bad_formula` with the message and position, and nothing is written.

### internal/collection/forms.go · public forms
- refers-to: internal/collection/forms.go :: `SetFormLink`, `PublicForm`, `PublicSubmit`, `submit`; internal/collection/service.go :: `formLimit`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

A public form is reachable only through its random token in `form_links`, which can be turned off (the token is deleted) and on again (a new token). Public submissions are limited to 20 a minute per address with a burst of 10. Conditional questions are evaluated on the server against the submitted answers; a hidden question's answer is discarded even when the client sent one. A public row has no author and no owner (`created_by` empty); a signed-in submission is attributed to the person unless the form is anonymous.

### internal/collection/csv.go · two-step import
- refers-to: internal/collection/csv.go :: `PreviewCSV`, `ImportCSV`, `stashTTL`, `guessType`; internal/order/order.go :: `Sequence`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

Preview parses the upload (delimiter sniffed, BOM removed), guesses a type per column and keeps the parsed rows for 20 minutes under a token tied to the user. Import takes the token once and inserts every row in one transaction with prepared statements. Sort keys come from `order.Sequence`, which gives fixed-width keys after the last row; repeated `After` calls would grow a key by one character every five rows and made a 20,000-row import take 38 seconds and send 300 KB windows.

### internal/order/order.go · Sequence
- refers-to: internal/order/order.go :: `Sequence`, `After`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

`Sequence(after, n)` returns n increasing keys, all greater than `after`: the key `After(after)` followed by a fixed-width base-62 counter and a closing `V`, so no key ends in `0`. Use it for any bulk append. Single appends with `After` still lengthen the last key by one character every five appends, which is fine at human pace and is noted in BUILD_NOTES.

### internal/httpx/respond.go · compressed JSON
- refers-to: internal/httpx/respond.go :: `JSONZ`, `gzPool`; internal/collection/module.go :: `okz`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

`JSONZ` encodes first, then gzips (best speed, pooled writers) when the body is at least 1,400 bytes and the request accepts gzip, and sets `Vary: Accept-Encoding`. It is used where bodies are large: database reads, queries and row reads. Plain `JSON` stays for everything else, and streaming routes must never use either.

### internal/collection/module.go · first page in the same trip
- refers-to: internal/collection/module.go :: `GET /api/collections/{id}`; internal/server/server.go :: `bootCollection`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

`GET /api/collections/{id}?view=<id>` also returns `first: {view_id, result}`, the first 100 rows of that view (or of the first view when the id is empty or unknown), for table and list views only. Opening a database therefore costs one round trip instead of two. The boot document does the same for the page being loaded.

### internal/collection/formula · the language
- refers-to: internal/collection/formula/parse.go; internal/collection/formula/eval.go :: `Compile`, `Program.Props`; internal/collection/formula/funcs.go :: `Names`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

A Pratt parser with infix operators, the ternary, the words `and`, `or` and `not`, method calls (`prop("Due").dateAdd(1, "days")`), lists and lambdas over `current` and `index`. Smart quotes and the symbols for not equal, at least and at most are accepted so pasted formulas work. `if`, `ifs`, `and`, `or`, `let`, `lets` and the list functions that take a lambda are special forms evaluated lazily. `Props()` lists the properties a formula reads; the formula check endpoint returns it with the result.

## Phase 6: search and editor completion {#p6}

### internal/search · passages and the worker
- refers-to: internal/search/passages.go :: `split`; internal/search/search.go :: `Touch`, `RowsChanged`, `run`, `prepare`, `drain`, `write`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

A body is split into passages of at most 1500 characters, each carrying the heading path above it, plus one title passage and one passage of database property text per row. Pages are queued by `Touch` (the materialiser and page writes) and `RowsChanged` (database edits) and indexed by one worker in batches of 100 after a 400 ms debounce. `prepare` drops passages of deleted pages and rebuilds everything when `indexVersion` changes, so a changed splitter only needs the constant bumped.

### internal/search/search.go · tiers and the rowid floor
- refers-to: internal/search/search.go :: `Search`, `ranked`, `tier`, `floor`, `candidates`, `rankCap`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

bm25 over every match of a common word is what made broad queries slow. Search therefore asks the title tier first and the body tier second, and the body tier ranks only the newest 2000 matching passages (a rowid floor found with one count query), keeping 200 candidates. Workspace and visibility filtering happens in Go on the candidates, with a second pass scoped to the workspace when too few survive. Keep the ws column out of MATCH: filtering on it in FTS cost more than it saved.

### internal/search/highlight.go · highlights from Go
- refers-to: internal/search/highlight.go :: `fold`, `stem`, `queryTerms`, `mark`, `snippet`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

FTS5's `highlight()` over `rowid IN (...)` scanned every match. The passage text is read by rowid and marked in Go with the same folding (accents, sharp s) and a light stemmer, using `\x02` and `\x03` as markers that the client turns into `mark` elements. Never send them as HTML.

### internal/doc/math.go · single-line $$ blocks and the MathML allow-list
- refers-to: internal/doc/math.go :: `mathBlockParser.Continue`, `mathBlock.done`, `sanitizeMathML`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

goldmark ignores a `Close` returned from `Open`, so a block that opens and closes on one line sets `done` and closes on the next `Continue`; without that it swallowed the following paragraphs. A line starting with `$$` that has text after the closing `$$` is a paragraph with inline math, not a block. Rendered MathML from the client is stored only after `sanitizeMathML` keeps the allowed elements and attributes.

### internal/netguard · one guard for every outbound fetch
- refers-to: internal/netguard/netguard.go :: `Blocked`, `CheckURL`, `NewClient`, `Reason`; internal/config/config.go :: `FETCH_PRIVATE`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

Every server-side fetch (media imports, link previews; webhooks and URL ingest later) must use this client. Names are resolved first and only public addresses are dialled, by address, so DNS rebinding cannot swap in a private one; the dialer's Control re-checks the connected address. Loopback, private, link-local, carrier-grade NAT, documentation, benchmarking, multicast and reserved ranges are blocked, and NAT64, 6to4 and Teredo forms with them. Redirects are re-checked and capped at five. The transport ignores HTTP_PROXY on purpose: a proxy would make the address check meaningless. `FORMMD_FETCH_PRIVATE=1` turns the guard off for intranets.

### internal/files/fetch.go · imports
- refers-to: internal/files/fetch.go :: `Import`, `remoteName`, `dims`, `webpDims`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

An import is an upload whose body is the response: the same `store` path sniffs the type, enforces the size limit and writes the audit record. Web pages are refused with `not_media`. Image size comes from the standard decoders, and WebP from its VP8, VP8L or VP8X header. `store` now sets the private `ext` on the returned file so `dims` can open it.

### internal/unfurl · previews
- refers-to: internal/unfurl/unfurl.go :: `Preview`, `head`, `frameable`, `handle`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

At most 1 MB of the head is tokenised; Open Graph, Twitter and plain meta tags are read up to `</head>` or `<body>`. The preview image and the best icon (apple-touch-icon over sized icons over /favicon.ico) are imported as page attachments so cards never point at another site. Framing is judged from X-Frame-Options and a frame-ancestors directive. Previews are cached for an hour, 256 entries, and at most eight fetches run at once. The endpoint needs write access to the page because it creates files.

### internal/doc/embed.go · the provider table is mirrored in the client
- refers-to: internal/doc/embed.go :: `providers`, `EmbedProvider`, `EmbedSrc`; web/src/engine/doc/schema.ts :: `PROVIDERS`, `embedSrc`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

Eighteen providers map a link to a player address and a default height; the generic `web` kind frames any https address. The server paints the iframe in the first frame and the editor paints it again, so the two tables must match entry for entry, regular expressions included (Go's RE2 and JavaScript agree on the subset used).

### internal/static/static.go · the sandbox page
- refers-to: internal/static/static.go :: `sandboxCSP`, `ServeHTTP`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

`/sandbox.html` (the Mermaid renderer) is served with a CSP whose first directive is `sandbox allow-scripts`, so even opened directly it runs in an opaque origin with no cookies or storage, and with X-Frame-Options SAMEORIGIN instead of the global DENY. Its module scripts are then cross-origin requests, which is why everything under `/assets/` carries `Access-Control-Allow-Origin: *`. The document CSP allows frames from any https origin because every frame is sandboxed.

### internal/doc/synced.go · synced roots and rendering without nested locks
- refers-to: internal/doc/synced.go :: `SyncRoot`, `syncedRefs`, `renderSynced`, `htmlWriter.synced`; internal/doc/service.go :: `Render`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

A synced block's content is the root `s:<key>` of the original page's document; pages hold only an atom with `page` and `key`. `Render` collects the references under the page's room lock, releases it, renders every other page's root under that page's lock one at a time, and only then renders the page. Rendering another room while holding the first lock would deadlock two pages that show each other's blocks. A reader who cannot open the original gets a fixed notice that the client prints word for word. Content inside a synced root is never expanded again (no nesting).

### internal/doc/markdown.go and parse.go · synced markers
- refers-to: internal/doc/markdown.go :: `room.summary`, case `synced`; internal/doc/parse.go :: `htmlBlock`, `skipping`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

The original writes `<!-- synced:<page>/<key> begin -->`, the content, then `<!-- synced end -->`, so search and exports see the text; a copy writes the marker alone. Reading Markdown back makes the atom and skips everything up to the end marker, because that content lives in the synced root and is not part of the page body. Materialising uses `room.summary`, which can read the room's other roots.

### tools/notices · license files and stable order
- refers-to: tools/notices/main.go :: `licenseFiles`, `webLicense`, the component sort
- kind: explainer
- logged: v0.6.0 · 2026-10-08

Mermaid brought packages that ship `license.txt` or no license file at all. A package that declares MIT in package.json without a file gets the standard MIT text with the author as the holder, marked as such; any other license without a file still stops the build. Packages present in two versions are ordered by version so the file does not change between builds.

---

## Phase 7: collaboration, sharing and identity {#p7}

### internal/page/access.go · the chain walk
- refers-to: internal/page/access.go :: `Level`, `viewer`, `step`, `chainLevel`, `levels`, `Reach`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

A page's level is computed from the root down: the root takes the workspace role (or the teamspace rule for its teamspace), a private page resets to none unless the reader is its owner, and a share on any page in the chain raises the level from there down. `levels` memoises the same walk over a whole tree. Guests have no role, so only shares reach them; `viewer` returns hidden when a non-member has no share in the workspace at all.

### internal/page/tree.go · Filter and the NOT IN trap
- refers-to: internal/page/tree.go :: `Filter`, `marks`
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

Listing filters (search, library) exclude hidden subtrees for members and keep only shared subtrees for guests with recursive CTEs. `marks(0)` must return `SELECT NULL WHERE 0`, a list with no rows; `NOT IN (NULL)` matches nothing and once hid every page from a member's search.

### internal/page/shares.go · links grant real shares
- refers-to: internal/page/shares.go :: `ResolveLink`, `GrantLink`, `RevokeLink`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

A signed-in person who opens `/l/<token>` gets a page share at the link's level with `created_by = link:<link id>` and the link's expiry, unless they already reach that level. Revoking the link deletes those rows in the same transaction. Anonymous readers get the public renderer instead and never a share.

### internal/comment · anchors and suggestions
- refers-to: internal/comment/comment.go :: `Create`, `Event`; internal/comment/suggest.go :: `Suggest`, `Revise`, `Decide`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

Text anchors and suggestions store the editor's base64 Yjs relative positions, which the server treats as opaque strings. Accepting a suggestion needs edit access; rejecting needs edit access or authorship. Reactions, revisions and decisions are audited, so webhooks see them.

### internal/inbox · push and mail timing
- refers-to: internal/inbox/inbox.go :: `Run`, `FromComment`, `FromRefs`, `FromShare`, `FromAssign`; internal/inbox/webpush.go
- kind: explainer
- logged: v0.7.0 · 2026-10-09

A notification is pushed if still unseen after 10 seconds and mailed if still unread after `notify.email_delay_minutes` while the person has no live stream (`Hub.Online`). Web Push is RFC 8291 aes128gcm with a VAPID ES256 token built from the standard library; the VAPID key is a sealed setting created on first boot. Reminders come from date mentions with `remind` and `at`, collected by the materialiser.

### internal/doc/versions.go · when a version is kept
- refers-to: internal/doc/versions.go :: `saveVersion`; internal/doc/service.go :: `materialise`, unload; internal/doc/room.go :: `changedSinceVersion`, `editors`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

A version is the whole Yjs state, deflated. Materialising keeps one at most every ten minutes, unloading keeps one when anything changed, a hand save always keeps one (or returns the last if nothing changed), and `?kind=restore` keeps one before a restore. `editors` lists who changed the page since the previous version. The empty paragraph a new room creates does not count as a change. Restoring happens in the client; the server only records it.

### internal/doc/room.go and conn.go · no-op syncs
- refers-to: internal/doc/room.go :: `deleteSetBytes`, `expectDS`, `onUpdate`; internal/doc/conn.go :: `syncMessage`
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

ygo's update event carries the new structs and the document's whole delete set, so a client that reconnects with its cached state produces a non-empty update even when nothing changed. Before applying a step 2, or an update with no new structs, the room records its delete set; `onUpdate` drops an update that has no structs and the same delete set. Without this, opening a page recorded the reader as its last editor, broadcast a frame and counted toward history.

### internal/doc/public.go and html.go · public mode
- refers-to: internal/doc/public.go :: `Public`, `RenderPublic`; internal/doc/html.go :: every `w.pub` branch
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The first-frame writer gains a public mode: page links and mentions go through `Public.Page` (an unpublished target becomes a "Private page" span with no title), files through `Public.File`, callout, mention and crumb icons and the toggle chevron are inlined, task boxes are disabled, user ids and reminder attributes are left out, remote media becomes a link, database blocks become tables and code bars show the language. Synced content from another page is shown only when that page is published too.

### internal/site · routes and scope
- refers-to: internal/site/site.go; internal/site/render.go :: `load`, `renderPage`, `siteCSP`; internal/site/module.go :: `Host`, `serve`, `unlockPost`, `fileAllowed`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

`/s/<slug>[/<page>]` for sites, `/l/<token>[/<page>]` for share links, and any path on a site's custom domain. The published set is the root and its descendants that are not trashed or private, limited to rows and inline databases when sub-pages are off. Files are served only when a published page references them (`doc_refs`), owns them, uses them as cover or icon, or the site uses one as its preview picture. Passwords set an HMAC cookie keyed by `sites.key` and the password revision, so changing the password signs everyone out. The host middleware runs before sessions and passes `/assets/` through so the site stylesheet loads on custom domains. A hostname equal to the instance's own host, an IP or localhost is refused.

### internal/site/render.go · the asset table
- refers-to: internal/site/render.go :: `loadAssets`; web/vite.config.ts :: `iconTable`, input `site`
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

The renderer reads the site entry's script, stylesheet and Inter preload from `.vite/manifest.json` and the icon and glyph SVGs from `.vite/icons.json`, both emitted by the Vite build. Without a built client the pages still render, unstyled and without icons.

### internal/analytics · counting views
- refers-to: internal/analytics/analytics.go :: `Record`, `Page`, `Workspace`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

One row per page, day and viewer; a repeat within ten minutes only moves `last_at`. Viewers are user ids, `h:<hash>` for someone who opted out or whose workspace turned view history off, and `v:<hash>` for web visitors (a daily hash of the site key, address and user agent, no cookie). Page reports need edit access; workspace reports are for admins and list only pages the admin can open.

### internal/webhook · the audit cursor
- refers-to: internal/webhook/webhook.go :: `pumpAudit`, `enqueue`, `deliverDue`, `sign`; internal/webhook/manage.go
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The dispatcher reads `audit_log` past the `audit` cursor every two seconds or when poked, turns each entry into an event named after its action, and queues a delivery for each active hook whose patterns match (a `siem` hook takes every audit event). Deliveries are signed `X-FormMD-Signature: t=<unix>,v1=<hex HMAC-SHA256 of "t.body">`, never follow redirects, retry at 30 s, 2 min, 10 min, 30 min, 2 h, 6 h and 12 h, and a hook pauses itself after 100 consecutive failed deliveries. Secrets and authorization headers are sealed.

### internal/audit/query.go and internal/auditapi · reading the log
- refers-to: internal/audit/query.go :: `Filter`, `Query`, `Facets`, `Purge`; internal/auditapi/auditapi.go
- kind: explainer
- logged: v0.7.0 · 2026-10-09

An action ending in a dot filters by prefix (`page.`); otherwise it is exact. Paging is by id descending with `before`. The CSV prefixes cells that start with `=`, `+`, `-`, `@`, tab or carriage return with an apostrophe. `audit.retention_days` is enforced by housekeeping.

### internal/scim · mapping
- refers-to: internal/scim/scim.go :: `Middleware`, `serve`; internal/scim/users.go; internal/scim/groups.go; internal/scim/manage.go
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The endpoint runs as middleware ahead of the session middleware, which would otherwise reject a `scim_` bearer token. A token belongs to one shared workspace. Users in scope are the workspace's members plus anyone in `scim_users` for it; `active` is membership. Creating links an existing account by email or creates one with an unusable random password (people then sign in through SSO or reset it); deactivating and deleting remove the membership through `workspace.SetAccess`, which keeps the last-admin rule. Groups are workspace groups with `external_id`; members must already be workspace members. Every change is audited as `system` with the label `SCIM <token name>`.

### internal/auth/saml.go · what the library leaves to us
- refers-to: internal/auth/saml.go :: `samlSP`, `handleSAMLStart`, `handleSAMLACS`, `samlAccount`; internal/auth/service.go :: the ACS origin exemption
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

gosaml2 verifies the signature and reports audience and time problems as warnings, so the ACS refuses `InvalidTime` and `NotInAudience` itself. It also refuses a reused assertion id (kept in memory until the assertion's end), a response whose `InResponseTo` differs from the request id sealed in the RelayState, and, when the state cookie is present, one that belongs to another sign-in. The ACS is exempt from the cross-site origin check because the identity provider posts it. Accounts link by NameID first, then by email; new accounts follow the sign-up policy. SAML-only refuses password sign-in for everyone but the owner, sign-up, and the other providers.

### internal/server/server.go · version slot and the inbox route
- refers-to: internal/server/server.go :: `renderDocument` (`slots["version"]`, `attrs["route"] = "inbox"`)
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The sidebar's version text is filled from `app.Version` on the server, so the footer line is its final height in the first frame. `/inbox` sets `data-route="inbox"` so a reload there paints the inbox view, not the home view the client would then swap out.

## Phase 8: agents and AI {#p8}

### internal/doc/blocks.go · edits by block id
- refers-to: internal/doc/blocks.go :: `Apply`, `Revert`, `Outline`, `editor`, `locate`, `room.agentHere`
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

Blocks are found by their id attribute, never by index, so an edit lands on the right block while people type elsewhere. All reads happen before a transaction and each edit runs in its own `doc.Transact`: `ToDelta` takes a read lock that deadlocks inside a transaction. Every edit returns an inverse that holds a JSON snapshot of the nodes it removed or replaced, which is what `agentrun` replays to undo. Schema rules the CRDT does not enforce are kept by hand: list items only inside lists (`asItems`, `wrapItems`), table cells hold paragraphs, a container emptied by an edit gets a filler paragraph, and an edit aimed inside a list item escalates to the item. The agent's presence is an awareness entry whose cursor is a relative position, cleared 20 seconds after the last edit.

### internal/agentrun · runs and undo
- refers-to: internal/agentrun/agentrun.go :: `Open`, `Record`, `Finish`, `Undo`, `List`, `Purge`; migration 0016
- kind: explainer
- logged: v0.8.0 · 2026-10-09

A run groups the writes of one task (`kind` mcp, chat, custom or ai) and belongs to the person behind the credential. Each write records a step with a registered undoer name and its inverse; `Undo` replays the steps newest first and marks the run undone. Undoers exist for blocks, page create, trash, update and move, rows add and update, schema and comments. Runs older than 90 days are purged by housekeeping.

### internal/mcp · transport
- refers-to: internal/mcp/server.go :: `handle`, `dispatch`; internal/mcp/tools.go :: `callTool`, `frame`
- kind: explainer
- logged: v0.8.0 · 2026-10-09

Built to the stateless 2026-07-28 revision: no initialize or session, the version, client info and capabilities ride in `_meta`, and the `MCP-Protocol-Version`, `Mcp-Method` and `Mcp-Name` headers must agree with the body (HeaderMismatch -32020; an unsupported version is -32022 with the supported list). Clients on 2025-11-25, 2025-06-18 and 2025-03-26 may still initialize. GET and DELETE answer 405. Read credentials never see write tools. Every result is framed `[untrusted result begin nonce=X] ... end nonce=X]` with a per-request nonce, writes return a `run_id`, and an `idempotency_key` replays the stored first result for 24 hours.

### internal/mcp/internal.go · in-process calls
- refers-to: internal/mcp/internal.go :: `Specs`, `Invoke`, `Session`
- kind: explainer
- logged: v0.8.0 · 2026-10-09

The workspace agent and custom agents call the same tool handlers as remote clients, so permissions, run grouping, undo, presence and framing are identical. `Invoke` builds a synthetic call with the session's display name, run kind and owning agent, forces the session's `run_id` onto writes and remembers the run id the first write opens. `run_id` and `idempotency_key` are left out of the specs the model sees.

### internal/oauth · OAuth 2.1 for agents
- refers-to: internal/oauth/oauth.go; internal/server/consent.go; internal/auth/oidc.go :: `BaseURL`, `MCPChallenge`; migration 0017
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

Protected resource metadata at `/.well-known/oauth-protected-resource[/mcp]`, authorization server metadata with client ID metadata documents and the `iss` response parameter. Public clients register dynamically or by a metadata document URL (https, or http only with FETCH_PRIVATE). PKCE S256 only, the `resource` must be the MCP address, codes live a minute, access tokens (`fmo_`) an hour and refresh tokens (`fmr_`) 30 days with rotation; reusing a code or a refresh token revokes the whole grant. OAuth tokens work only at `/mcp`. A password change revokes every grant.

### internal/ai · providers and the allow-list
- refers-to: internal/ai/service.go :: `resolve`, `allowed`, `Allowed`, `Exec`, `Run`, `Transcribe`; internal/ai/anthropic.go; internal/ai/openai.go; migration 0018
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

A model reference is `providerID:model`. Only models an instance admin enabled are offered or accepted: an explicit reference off the list is `model_unavailable`, and no reference means the default model, then the first enabled one. Keys are sealed with the instance key and never returned. `Run` checks the instance and workspace switches, the per-person hourly limit and a 30-per-minute burst before calling; `Exec` skips those checks so a loop (an agent's steps, autofill over many rows) is checked once by its caller with `Allowed`. Every call is logged in `ai_usage`. The HTTP client is ordinary on purpose: admins point it at local servers such as Ollama. Gemini goes through its OpenAI-compatible endpoint.

### internal/ai/module.go · checks before the stream
- refers-to: internal/ai/module.go :: `POST /api/ai/complete`, `POST /api/ai/transcribe`, `OpenStream`
- kind: explainer
- logged: v0.8.0 · 2026-10-09

`complete` resolves the model and runs every switch before it opens the event stream, so a refusal is a plain HTTP error rather than an error event after a 200. Events are `delta`, `done` and `error`. `transcribe` takes one audio segment up to 25 MB, keeps nothing, and is answered by the model an admin marked as audio.

### internal/ai/writing.go · prompts
- refers-to: internal/ai/writing.go :: `WritingSystem`, `actions`, `WritingPrompt`
- kind: explainer
- logged: v0.8.0 · 2026-10-09

Page text, selections, the text before the cursor and meeting notes reach the model inside `<page>`, `<selection>`, `<before>` and `<notes>` tags that the system prompt declares to be material, never instructions. A page is included only when there is no selection or the action needs context. A meeting transcript may run to 120,000 characters; other selections to 24,000.

### internal/aifill · autofill
- refers-to: internal/aifill/aifill.go :: `Fill`, `fillRow`, `ToValue`, `RowsChanged`, `PageChanged`, `schedule`; internal/collection/schema.go :: `Autofill`; internal/collection/schemaops.go :: `UpdateSchema`
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

A property's `autofill` holds the instruction, whether to refill on change, and `by`, which `UpdateSchema` sets to whoever saved it; background refills run as that person through a session-shaped principal. Answers are parsed per type and select answers only ever match existing options, so the model cannot invent options. The autofill write marks the row quiet for eight seconds so its own change event does not refill it again; refills wait four seconds after the last change and run two at a time.

### internal/agent · the workspace agent
- refers-to: internal/agent/loop.go :: `Run`, `system`, `tools`, `callTool`, `history`; internal/agent/agent.go
- kind: explainer
- logged: v0.8.0 · 2026-10-09

A turn saves the person's message, builds the system prompt (identity, workspace, today's date, the MCP rules, workspace rules, the instructions page or the custom agent's instructions, memory with ids, the page being looked at, the trigger), offers every MCP tool but `propose_rules` plus `remember` and `forget`, and loops up to 16 model steps. Tool results go back as `tool_result` parts; only the text of earlier turns is replayed as history. All writes of a turn share one run, finished as done or failed, and the reply stores its text and tool steps in order.

### internal/agent/runner.go · triggers and loops
- refers-to: internal/agent/runner.go :: `principal`, `Loop`, `due`, `OnEvent`, `debounce`, `launch`; internal/agent/custom.go :: `NextRun`; internal/webhook/webhook.go :: `OnEvent`
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

Custom agents run as their creator through an OAuth-kind principal limited to their workspace, so audit entries and webhooks record them as agents and the workspace agent policy applies. The schedule loop ticks every 15 seconds and claims a due agent by moving `next_run_at` with a compare-and-set, so a run is never started twice. Row and page triggers come from the webhook event stream: events whose actor is an agent are ignored, and because a page-content event carries no agent marker, an agent's page triggers are also ignored while it runs and for a minute on the pages its run touched. Row triggers wait three seconds, page triggers fifteen. Two runs at a time, ten minutes each at most.

### internal/search/sources.go and internal/answer · AI answers
- refers-to: internal/search/sources.go :: `Keywords`, `Sources`; internal/answer/answer.go
- kind: explainer
- logged: v0.8.0 · 2026-10-09

The page search needs every word, which fails for questions, so answers use an OR of the question's keywords (stop words dropped, prefix match) over titles, headings and bodies in the workspace, keep only pages the asker can read, two passages per page and eight in all. The stream sends the sources first, then the cited answer.

### internal/doc/meeting.go and html.go · the meeting block
- refers-to: internal/doc/meeting.go :: `MeetingMeta`; internal/doc/html.go :: the `meeting` and `ai_block` cases
- kind: explainer
- logged: v0.8.0 · 2026-10-09

The head text is the UTC date and rounded minutes, built the same way in the editor, so the first frame and the live editor agree. Icons are drawn on public pages only, as for toggles; in the app the icon slot has a fixed size.

### internal/httpx/middleware.go · Permissions-Policy
- refers-to: internal/httpx/middleware.go :: security headers
- kind: explainer
- logged: v0.8.0 · 2026-10-09

`microphone=(self)` and `display-capture=(self)` let meeting notes record; camera, geolocation and payment stay off.

## Phase 9: automations, charts, buttons, templates, views {#p9}

### internal/collection/chart.go · charts through the query path
- refers-to: internal/collection/chart.go :: `Chart`, `ChartSpec`, `measure`; internal/collection/module.go :: `POST /api/collections/{id}/chart`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

A chart is a query: the view's filters run through `results.ordered`, rows are bucketed with the same `bucketise` the grouping code uses (so date buckets and option order match the board), and each bucket is measured by the rollup aggregator. That keeps access rules, template-row hiding and formula values identical to the table. Presentation fields (colours, smoothing, labels) ride in the spec so the client draws from one object, and the result carries the snapshot epoch so a live event can skip a refetch that would change nothing.

### internal/automation/runner.go · events, debounce and the loop guard
- refers-to: internal/automation/runner.go :: `OnEvent`, `fire`, `Loop`, `RunButton`, `RunProperty`
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

Triggers read committed audit entries through the hooks pump, never service calls, so a rolled-back write never fires anything. Entries whose agent name starts with `Automation: ` (the name every automation run carries) are dropped, which is the whole loop guard: an automation's own edits cannot fire automations. Edits through personal or OAuth tokens and by agents do fire them (since v0.10.1); custom agents ignore agent-made events on their side, so an agent and an automation cannot feed each other either. Property edits wait 1.5 s per row and automation so a burst of typing fires once; new rows wait 300 ms so the initial values set right after creation are in the row when the filter is checked. The schedule loop ticks every 15 s and claims a due automation with a compare-and-set on `next_run_at`, so two ticks never run it twice. Buttons run as the presser without an agent name, so their edits are the person's edits and may fire automations.

### internal/automation/exec.go · actions and failures
- refers-to: internal/automation/exec.go :: `execute`, `reason`, `failure`; internal/automation/automation.go :: `signingKey`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

Actions run in order and the first failure stops the rest, recorded with a sentence a person can act on (`reason` maps API errors, the fetch guard and network errors). Mail goes only to workspace members, never to typed addresses. Webhooks go through `webhook.Send` and the fetch guard, signed with a per-workspace key in `automation_keys` sealed by the instance secret box, so a stolen database alone cannot forge signatures. Runs keep the last 50 outcomes per automation.

### internal/doc/button.go · the button node in Markdown and HTML
- refers-to: internal/doc/button.go :: `buttonEncode`, `buttonNode`; internal/doc/html.go and markdown.go :: the `button` cases
- kind: explainer
- logged: v0.9.0 · 2026-10-09

Markdown has no button, so the node is written as `<!-- button:<base64url JSON> -->`, which renders as nothing elsewhere and round-trips exactly. Published HTML shows only the label: actions can name webhooks and people, which a public reader must not see.

### internal/collection/templates.go · template rows
- refers-to: internal/collection/templates.go :: `templates`, `CreateTemplate`, `SetDefaultTemplate`, `fromTemplate`; internal/collection/cache.go :: `Row.Template`; internal/collection/results.go
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

Row templates are rows with `pages.is_template = 1`. They stay in the snapshot (so the template list is free) and every consumer skips them: `ordered`, the row count, `SearchRows`, the relation check and automation filters. Any new consumer of the snapshot must skip them too. A row from a template is made by `pages.Duplicate` (body and children), then a `row.create` audit entry so automations and webhooks see a new row, then `UpdateRow` for the values.

### internal/templates · starters
- refers-to: internal/templates/starters.go; internal/templates/templates.go :: `GET /api/templates`, `POST /api/templates/use`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

Starters are Go values written for this project, not data from any other product. Row dates are offsets (`+3d`) and `@me` is the person using the starter, resolved at use time so samples always look current. Since v0.38.1 the meeting notes and team wiki starters use `> [!TIP]` and `> [!NOTE]` alerts for their hints, which the Markdown parser turns into callouts (green and default); they had been plain quotes while alerts did not import as callouts.

### internal/doc/service.go · copying a page nobody has edited
- refers-to: internal/doc/service.go :: `Copy`
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

A page created from Markdown may have its body only in an open room, with no stored `doc_state` row yet. `Copy` now encodes the live state in that case; before, duplicating such a page (every template use does) gave an empty body.

### internal/comment/pagereact.go and internal/feed · page reactions
- refers-to: internal/comment/pagereact.go :: `ReactPage`, `PageReactions`, `CommentCounts`; internal/feed/feed.go :: `Feed`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

Feed posts react with the comment reaction icons, stored in their own table keyed by page, and the feed reads reactions and comment counts for a whole window in two queries. The feed is a view query with grouping off, so its order and filters are the view's.

### internal/collection/ics.go · calendar feeds
- refers-to: internal/collection/ics.go :: `CalendarFeed`, `FeedOwner`, `ICS`; internal/collection/module.go :: `GET /api/public/calendar/{file}`
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

The token is 160 random bits, stored as given (like form links) so the owner can see the address again. The public route rebuilds the owner's principal with `PrincipalFor` and forces a session-kind, read-scope principal: the view and page checks refuse anything else as "not visible". A deactivated owner's feeds stop because `PrincipalFor` refuses them. All-day events use `VALUE=DATE` with an exclusive end, as RFC 5545 requires.

### internal/collection/tree.go · sub-items order
- refers-to: internal/collection/tree.go :: `treeOrder`, `reachable`; internal/collection/service.go :: `Query` (`in.Tree`)
- kind: explainer
- logged: v0.9.0 · 2026-10-09

With a self relation named in `tree`, rows come back parent first, children only under expanded parents, with depth and child counts. A row whose parent is filtered out becomes a root, depth stops at 32, and rows caught in a parent cycle are shown as roots once. Grouping turns the tree off.

### internal/graph · the link graph
- refers-to: internal/graph/graph.go :: `GET /api/workspaces/{id}/graph`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

Nodes come from the page tree the caller can read, so the graph inherits every access rule; edges are parent links plus `doc_refs` between readable pages. Large workspaces are cut at 3,000 nodes with `truncated` set.

## Phase 10: import, export and API {#p10}

### internal/transfer/tree.go · one import tree, one writer
- refers-to: internal/transfer/tree.go :: `Node`, `Table`, `Asset`, `Bundle`, `writer.create`, `writer.body`, `materialise`
- kind: explainer
- logged: v0.10.0 · 2026-10-09

Every reader builds the same tree: pages with Markdown bodies, databases as headers and rows with optional row pages, and attachments as bytes. Bodies refer to other imported pages as `page:N` and to attachments as `asset:N` in link destinations. The writer creates every page first (so ids exist), then writes bodies, replacing `page:N` with `/p/<id>` and storing each asset once, on the first page that uses it, as `/files/<id>`. A failure on one page becomes a warning on the job; the rest continues.

### internal/transfer/source.go · zip limits
- refers-to: internal/transfer/source.go :: `vfs`, `addZip`, `maxEntries`, `maxDeclared`, `maxEntryBytes`, `maxRatio`
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

Uploads go to temp files under the data directory and zips are read in place. Limits: 50,000 entries, 4 GB declared, 256 MB per entry (also enforced while reading, since headers can lie), and entries over 50 MB that compress more than 200 to 1 are skipped. Zips at the root of the top zip, or named like the page-tree export's `-Part-N.zip` part files, are opened one level deep (since v0.40.0; zips inside page folders stay attachments); paths are cleaned so `..` cannot escape. `__MACOSX` and dot-underscore files are ignored.

### internal/transfer/markdown.go · Markdown, page-tree and HTML trees
- refers-to: internal/transfer/markdown.go :: `importTree`, `treeImporter.place`, `fillTable`, `convert`, `resolve`, `dropPageProps`, `pageAsides`
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

A file `A.md` owns the folder `A/`; a folder with no owner becomes a page of its own, and `index.md` or `README.md` inside a folder becomes that folder's body. Relative links are resolved against the file's folder, URL-decoded and matched case-insensitively to documents, CSVs or attachments. The page-zip mode strips the 32-hex ids from names, prefers the `_all.csv` twin, matches row pages to CSV rows by title in order (unmatched pages become extra rows), drops the `Key: value` lines such exports write under a row's title, turns `<aside>` into a callout without its leading emoji, and keeps relation cells as plain titles.

### internal/transfer · checked against real page exports
- refers-to: internal/transfer/markdown.go :: `titleOf`, `rewriteLinks`, `resolve`, `resolvePath`, `deadPageLink`; internal/transfer/source.go :: `addZip`, `pageZipPart`; internal/transfer/tree.go :: `Bundle.used`; internal/transfer/transfer.go (the leftover-file warning)
- kind: rationale
- logged: v0.40.0 · 2026-10-17

Until v0.40.0 the page-tree importer was only exercised against hand-made samples. It was then run against twelve real Markdown and CSV page exports from public open-source test corpora (used as local test input only, never committed here). They exposed six defects. (1) The exports leave `#` unescaped in link paths (`#letstalkaboutstress%20…/page.md`), and `resolve` treated anything starting with `#` as an anchor and cut every link at the first `#`; it now tries the whole destination as a path first and falls back to anchor or fragment handling only when nothing matches. (2) A database is linked as `Name id.md` although only the CSVs exist; a `.md` link that is not a document now also looks up the database that owns that path. (3) A page whose title ends in a space exports as a folder `name id /`, and the id stayed in its title because the pattern is anchored at the end; names are trimmed first. (4) Every zip one level inside the upload was opened, so zips attached to pages were imported as extra copies (21 databases instead of 6 in one export); only zips at the upload's root, or named like the export's `-Part-N.zip` split files, are opened now, and the rest stay attachments. (5) Files a page had already taken in as attachments, such as videos, were still reported as "Skipped … not imported"; `Bundle.used` records what links consumed and the warning skips those. (6) The export writes the image path as the alt text; when the alt equals the link it becomes the file's name. In page-zip mode a link to a `.md` or `.csv` page that is not in the export becomes its plain label rather than a dead relative link. Image references to files missing from an export are left as they are, since the export itself lacks them.

### internal/transfer/htmlmd.go · HTML to Markdown
- refers-to: internal/transfer/htmlmd.go :: `htmlToMarkdown`, `container`, `paragraph`, `block`, `items`, `inline`, `joinBlocks`
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

One converter serves HTML files, ENEX notes and HTML space-export pages. Containers gather inline runs into paragraphs and flush them at each block child, which is how line-per-div ENEX notes come out right. `en-todo` and checkbox inputs make task items; consecutive list blocks are joined tightly, and a plain list that follows a task list switches to `*` markers so the two stay separate lists. Bold, italic, strike and underline also come from inline styles. The HTML parser treats `<en-todo/>` as an open element, so its text is read from inside it.

### internal/transfer/enex.go, office.go, formats.go, pdftext.go · the other readers
- refers-to: internal/transfer/enex.go :: `importENEX`; internal/transfer/office.go :: `importDOCX`, `importXLSX`; internal/transfer/formats.go :: `importOPML`, `importBoardJSON`, `importProjectCSV`, `importSpaceExport`; internal/transfer/pdftext.go :: `pdfText`, `glyphsToText`
- kind: explainer
- logged: v0.10.0 · 2026-10-09

ENEX media are matched to resources by MD5; resources no note body uses are appended as files, and tags, created time and source URL go in a footer. Word: paragraphs, heading styles (also through `styles.xml` outline levels), numbering (bullet or ordered from `numbering.xml`), runs with bold, italic, strike and underline, hyperlinks from relationships, tables and embedded pictures. Excel: shared and inline strings, booleans, and dates from date number formats; one database per sheet. A board-export JSON file becomes a database with a board view by list, the row page holding the description, checklists and comments. Project-export CSVs are recognised by their headers. PDF text uses the glyph positions from `Page.Content()` (the row grouping in the library ignores `Td`), joins glyphs into lines by height, inserts spaces from gaps and breaks paragraphs at line gaps over 1.6 times the usual one.

### internal/transfer/export.go · exports
- refers-to: internal/transfer/export.go :: `collect`, `addRows`, `assignPaths`, `exportMarkdown`, `exportHTML`, `mountExports`
- kind: explainer
- logged: v0.10.0 · 2026-10-09

The set is the readable page tree (`pages.Tree`) under the root plus each database's rows from a query, capped at 5,000 pages. Paths follow the tree with names cleaned of path characters and pictographs and numbered when they collide, so the zip mirrors what our importer reads back. Markdown export rewrites `/p/<id>` to relative paths (database links to the CSV) or to absolute URLs for pages outside the export, and copies files into `files/`. HTML export goes through the published-site renderer with an export scope; a single page inlines its pictures as data URIs up to 24 MB.

### internal/transfer/pdf.go · PDF
- refers-to: internal/transfer/pdf.go :: `exportPDF`, `pdfR.block`, `pdfR.list`, `pdfR.table`, `pdfR.image`
- kind: explainer
- logged: v0.10.0 · 2026-10-09

The page's Markdown is parsed with goldmark (tables, strikethrough, task lists) and drawn with fpdf using the Go fonts embedded from golang.org/x/image, so no font files ship. Each page starts a new sheet with a bookmark at its depth; lists indent by 6 mm with drawn check boxes, quotes and callouts get a bar, code a shaded mono block, tables equal columns with wrapped cells, and PNG, JPEG and GIF pictures are embedded at up to the text width. Databases print their first view, six columns and 300 rows at most, with a note pointing to the Markdown export.

### internal/site/export.go · the export scope
- refers-to: internal/site/export.go :: `NewExport`, `Export.Page`; internal/site/render.go :: `scope.hrefFn`, `scope.fileFn`, `scope.export`, `assets.cssText`
- kind: explainer
- logged: v0.10.0 · 2026-10-09

An export scope carries its own link and file functions and an export flag, so the same renderer that serves published sites writes relative links and copied files, inlines the site stylesheet read from the embedded dist, and adds a small script for toggles. Pages come from the caller's readable tree, not from publishing settings.

### internal/collection/importtable.go and csv.go · tables from importers
- refers-to: internal/collection/importtable.go :: `ImportTable`, `AddBoardView`; internal/collection/csv.go :: `parseDateCell`, `guessType`, `stashed.ids`
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

`ImportTable` goes through the CSV import path with a stash entry that also collects the new row ids in order, so the importer can attach row pages. Dates may be ranges with an arrow, a spaced dash or an en dash. Type guesses accept small tables: a select when values repeat and stay short, a multi-select when comma-separated short tokens repeat. A multi-select now needs short tokens, which keeps free text with commas as text.

### internal/apidoc · the OpenAPI document
- refers-to: internal/apidoc/apidoc.go :: `routes`, `summarize`, `overrides`, `tagOf`, `public`, `build`, `page`; internal/httpx/router.go :: `Router.Patterns`
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

The document is built from the router's recorded patterns at request time, so a new route is documented the moment it is registered; summaries are generated from the path unless `overrides` has one, and `public` is a prefix list that must be kept in step with routes that skip authentication. The result is cached until the route count changes. `/api/docs` renders the same data with inline styles only, under its own CSP.

### internal/auth/service.go and internal/oauth · token limits and REST audience
- refers-to: internal/auth/service.go :: the middleware (`apiLimit`, `wrong_audience`); internal/oauth/oauth.go :: `apiResource`, `canonResource`, `protectedResource`; internal/policy/principal.go :: `Audience`
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

Tokens (personal or OAuth) are limited per credential in the auth middleware; sessions are not, and `/mcp` keeps its own limiter. An OAuth authorization may ask for `<origin>/api`; the grant stores the canonical resource and the principal's `Audience` is `api` for such tokens. The middleware allows OAuth tokens only where their audience matches: REST tokens off `/mcp`, MCP tokens only at `/mcp`.

### internal/httpx/cursor.go · cursors
- refers-to: internal/httpx/cursor.go :: `Cursor`, `EncodeCursor`, `DecodeCursor`; internal/page/window.go :: `pageWindow`
- kind: explainer
- logged: v0.10.0 · 2026-10-09

Cursors are base64url JSON with an offset or a time and id. Page lists use keyset order (created time, then id), so new pages do not shift a running walk; rows and search wrap offsets, the audit log wraps its `before` time. Clients must treat cursors as opaque.

## Phase 11: offline, encryption and ship {#p11}

### internal/crypt/key.go · the operator key
- refers-to: internal/crypt/key.go :: `Parse`, `KeyPath`, `Load`, `WriteMeta`, `check`, `Sub`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

The key is 32 bytes as hex or base64 in a file outside the data directory (`FORMMD_KEY_FILE`, or a systemd credential named `form-md.key`). `encryption.json` in the data directory holds an HMAC key-check value, never the key: a missing key with the file present refuses to start, a wrong key fails the check, and a key with a plain database refuses so nobody encrypts half a directory by accident (use `form-md encrypt`). Every cipher key is an HKDF-SHA256 subkey with the label `form.md/<purpose>` (`db`, `wal`, `journal`, `files`, `check`); changing a label makes existing data unreadable.

### internal/crypt/stream.go · sealed files
- refers-to: internal/crypt/stream.go :: `Writer`, `Reader`, `file.chunk`, `Open`, `IsSealed`, `Seal`, `Unseal`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

Format: `FMDENC1\n`, a 16-byte salt, then 64 KiB plaintext chunks each sealed with AES-256-GCM under a per-file key (HMAC of the files subkey and the salt), nonce = chunk index, additional data = index plus a last-chunk flag so truncation and reordering fail. The plaintext size is derived from the file size, so `http.ServeContent` can seek and serve ranges. Every method is nil-safe: a nil key writes and reads plain bytes, and readers accept unsealed files so a directory being converted keeps working.

### internal/cryptvfs/vfs.go · the encrypting SQLite VFS
- refers-to: internal/cryptvfs/vfs.go :: `Register`, `xOpen`, `xRead`, `xWrite`, `encrypted`, `ctr`, `ptr`, `fp`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

A shim over SQLite's default VFS, registered once per process through modernc's libc bindings: our file struct is the method pointer, a handle into a Go map, then the underlying file. Only reads and writes are intercepted; locking, sync and shared memory go to the default VFS untouched. Database pages (4096 bytes, the default page size, assumed throughout) use XTS-AES-256 with the byte offset as tweak; partial reads and writes of the database are done a page at a time. WAL and journal page writes are recognised by size (exactly one page; journals only at offsets 4 mod 8, where record bodies sit), WAL full-frame reads during recovery decrypt their body. Frame headers stay plain, so WAL checksums of plaintext are visible. Temporary files use AES-CTR with a random per-open key; the store also sets temp_store to memory. The io methods report version 2 so SQLite never memory-maps, and SQLITE_FCNTL_MMAP_SIZE is answered with 0. `ptr` converts through `unsafe.Add` because `go vet` flags direct uintptr conversions; a 24-byte frame-header `copy` in `xWrite` is a loop because Go 1.26.0's compiler crashed lowering that memmove (`disjointTypes`).

### internal/store/store.go · opening through the VFS
- refers-to: internal/store/store.go :: `Register`, `Open`, `dsn`, `VFSName`, `HasData`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

`Open` takes the key; with one, both pools open with `vfs=formmd-crypt`. `VACUUM INTO` a plain path inherits the source's VFS; ops still names the VFS in a URI target to be explicit.

### internal/app/acme.go · built-in ACME
- refers-to: internal/app/acme.go :: `acmeManager`, `sealedCache`, `serveChallenges`; internal/app/app.go :: `Run`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

autocert answers TLS-ALPN on the main listener and HTTP-01 plus an HTTPS redirect on `FORMMD_HTTP_ADDR`. Host names must have a dot (autocert refuses `localhost`). The cache lives in `acme/` and is sealed when encryption is on. Verified with Pebble 2.8.0 using `SSL_CERT_FILE` for its root; there is deliberately no insecure switch.

### internal/ops · backup, restore, encrypt, doctor, reset-owner
- refers-to: internal/ops/backup.go :: `Backup`, `Restore`, `swapInPlace`, `unpack`; internal/ops/encrypt.go :: `Encrypt`, `sealAll`, `sealFile`, `ResetOwner`; internal/ops/doctor.go :: `Doctor`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

Backup is online: it asks a running server to save open documents (SIGUSR1, see flush_unix.go), then `VACUUM INTO` a snapshot next to the output and tars it with `files/`, `acme/`, `secret.key` and `encryption.json`, never the key. Restore refuses while the data-directory lock is held, unpacks into a stage, checks integrity with the archive's key check, then swaps: beside the directory normally, inside it on a mount point or when the parent is not writable. Every command that opens the database as root ends with `matchOwner`, because SQLite creates the WAL and shm files as the caller and the service user could then not open them. `Encrypt` on an encrypted directory only seals leftovers.

### internal/ops/lock_unix.go and flush_unix.go · lock and flush request
- refers-to: internal/ops/lock_unix.go :: `Lock`; internal/ops/flush_unix.go :: `OnFlushRequest`, `RequestFlush`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

The server holds an exclusive flock on `form-md.lock` and writes its pid there. `RequestFlush` reads the pid, sends SIGUSR1 and waits for `form-md.flushed` to carry a newer timestamp (20 s at most). Outside Unix the lock is an exclusive-create file that a crash leaves behind and there is no flush request.

### internal/doc/recover.go · stale stored text after an unclean stop
- refers-to: internal/doc/recover.go :: `Recover`, `FlushAll`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

Updates are flushed 300 ms after an edit but Markdown, refs and search text are materialised after 1.5 s; a stop in between left `doc_state.markdown` stale until the next edit. At start the server re-materialises every page whose `updated_at` is newer than `materialised_at`.

### internal/static/offline.go and the offline shell
- refers-to: internal/static/offline.go :: `offline`, `serveOffline`; internal/server/server.go :: `renderDocument` (`/offline-shell`); internal/static/static.go :: the `/sw.js` CSP
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

`/offline.json` lists the built assets (core = what index.html references plus the preload lists). `/offline-shell` renders the signed-in document with `data-shell="offline"` and no route or page. `/sw.js` needs its own CSP with `connect-src 'self'`; under the default `default-src 'none'` every fetch from the worker failed.

### internal/static/localize.go and internal/i18n · German on the server
- refers-to: internal/static/localize.go :: `localize`, `translate`; internal/i18n/i18n.go :: `T`, `Has`, `LoadFS`, `UserLang`; embed.go :: `Catalogs`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

The client catalogs (web/src/i18n/<lang>/*.json) are embedded and merged after the Go-only mail strings. index.html text nodes and aria-label, placeholder and title values are translated once per language when their trimmed text is a catalog key; script and style blocks are skipped so the CSP hashes stay valid. Mail and push use the recipient's language, invitations the inviter's.

### internal/app/update.go · update check
- refers-to: internal/app/update.go :: `CheckUpdates`, `LatestRelease`, `Newer`; internal/app/health.go
- kind: explainer
- logged: v0.11.0 · 2026-10-09

Five seconds after start and then daily, while `update.check` is on, the server reads the latest release tag (or `FORMMD_UPDATE_URL`); `/healthz` reports `latest` only when it is newer than the running version.

### internal/tunnel · the Cloudflare tunnel
- refers-to: internal/tunnel/tunnel.go :: `Service.Start`, `Stop`, `supervise`, `watch`, `Middleware`, `binary`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

A supervised `cloudflared` child (PATH or `FORMMD_CLOUDFLARED`): quick mode points it at the local listener and reads the trycloudflare.com address from its log; token mode passes the token as `TUNNEL_TOKEN` so it never shows in the process list. Exits restart with backoff up to a minute; shutdown stops it (the module is a Stopper). While it runs, a request from loopback that carries `Cf-Connecting-Ip` is treated as arriving over TLS from that address (`r.TLS` set, `RemoteAddr` replaced), so the origin check, secure cookies and rate limits see the public side. Once stopped the header is ignored.

### internal/mail/oauth.go · Gmail and Microsoft Graph sending
- refers-to: internal/mail/oauth.go :: `Providers`, `OAuth.Start`, `Finish`, `accessToken`, `send`; internal/admin/admin.go :: `mailConnect`, `mailCallback`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

`mail.mode` gmail and graph had settings but no sender. An admin saves a client ID and secret, then opens `/api/admin/mail/connect` (session only), which redirects to the provider with PKCE and a 15-minute single-use state; the callback stores the refresh token sealed and the account from the ID token. Access tokens are cached until a minute before expiry; a rotated refresh token is saved. Gmail takes the MIME message base64url in JSON, Graph takes it base64 as text/plain. Verified against a stub provider, not against Google or Microsoft.

### internal/ops/module.go, memory.go · browser backup and memory sizing
- refers-to: internal/ops/module.go :: `Module.Mount` (`GET /api/admin/backup`); internal/ops/memory.go :: `SizeMemory`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

The owner downloads the same archive `form-md backup` writes, from a browser session only (tokens are refused); open documents are flushed in-process first and the temporary folder in the data directory is removed after streaming. Without `GOMEMLIMIT`, a cgroup memory limit sets Go's soft limit to 90 percent of it.

### internal/server/server.go · automatic HTTPS from Instance settings
- refers-to: internal/server/server.go :: `Build` (`https.auto`, `https.domain`)
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

With no TLS in the environment, `https.auto` plus `https.domain` turn on ACME at the next start and move the default loopback listener to `:443`. A non-default `FORMMD_ADDR` is kept as given.

### deploy/fail2ban · the shipped filter
- refers-to: deploy/fail2ban/form-md.conf, form-md.local; internal/auth/service.go :: `authFailure`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

The filter matches the `form-md auth failure` log line (password, second factor and bad token) from the systemd journal; the address is the client address as the server sees it, so behind a proxy turn on `net.trust_proxy`.

### internal/crypt/stream.go, internal/store/store.go · strict sealed reads
- refers-to: internal/crypt/stream.go :: `ErrNotSealed`, `Key.Reader`; internal/store/store.go :: `Open` (mode 0600, `PRAGMA page_size`)
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

With a key, a reader refuses a plain file instead of passing it through, so a file that skipped encryption (copied in by hand, or left from before `encrypt`) is never served as if it were fine; `doctor` counts such files and `encrypt` seals them. `sealFile` reads with `os.Open`, not through the key, so it is unaffected. The VFS encrypts in fixed 4096-byte units, so the store checks `PRAGMA page_size` on open and refuses any other size rather than corrupt it. The database file is created 0600 and chmodded on every start, and `encrypt` chmods its `VACUUM INTO` copy before the rename, because SQLite creates files with the process umask.

### internal/ops/encrypt.go · temporary leftovers
- refers-to: internal/ops/encrypt.go :: `sealAll`, `EncryptReport.Removed`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

`sealAll` walks `tmp/` as well as `carriedDirs`. Dot-prefixed files there (`.upload-*`, `.import-*`, `.seal-*`) are temporaries of a write that never finished; with the lock held nothing will rename them into place, so they are deleted rather than left as plain copies.

### internal/collection/cache.go · stale snapshot after a schema change
- refers-to: internal/collection/cache.go :: `cache.gen`, `cache.get`, `bump`
- kind: bug-context
- logged: v0.11.0 · 2026-10-09

`get` loads a snapshot outside the lock. A schema change or row refresh landing while that load ran used to find nothing to update (`setSchema` and `refresh` only touch a cached snapshot), and the load then cached what it had read before the change, so `GET /api/collections/{id}` kept serving the old schema until the next eviction. Every invalidation now bumps `gen`; a load that sees `gen` move retries (up to three times, then returns its result uncached).

## Phase 12 · visual data views

### internal/collection/chart.go · new kinds, dots and picks
- refers-to: internal/collection/chart.go :: `chartKinds`, `singleSeries`, `ChartInput.Pick`, `chartDots`, the limit block
- kind: explainer
- logged: v0.12.0 · 2026-10-09

Area, radial, histogram, compare, heatmap and calendar reuse the bucket and lane aggregation; only the drawing differs, and `singleSeries` drops the series grouping for the kinds that have one. Gauge is a number. A dot plot skips aggregation and returns each row's value with the first x bucket it falls in (rows in several buckets, as with multi-selects, appear once). A pick runs the same bucketing and returns the ids in the bucket (and lane) named by the keys the client drew, capped at 500, so the drill-down list always matches the mark. Calendar charts lift the bucket ceiling to 1,100 so three years of days fit.

### internal/collection/query.go · weekday, hour and explicit hidden
- refers-to: internal/collection/query.go :: `groupKeys` (`weekday`, `hour`), `ViewProp.Hidden`, `IsHidden`
- kind: bug-context
- logged: v0.12.0 · 2026-10-09

`weekday` keys are `d0` to `d6` (Sunday is 0) ranked from the person's first day of the week; `hour` keys are `h00` to `h23`. `ViewProp.Hidden` was a plain bool with `omitempty`, so the client's explicit `false` (shown on a card view, where the default is hidden) never reached storage and shown properties vanished after a reload; it is a pointer now and `IsHidden` reads it.

### internal/store/migrations/0024_view_types.sql · widening a CHECK
- refers-to: internal/store/migrations/0024_view_types.sql
- kind: warning-context
- logged: v0.12.0 · 2026-10-09

SQLite cannot alter a CHECK, so the table is rebuilt inside the migration's transaction. Foreign keys are on and cannot be turned off inside a transaction, so dropping the old table cascades into `form_links` and `calendar_feeds`; both are copied to temp tables first and put back after the rename. Any later table that references `collection_views` must be added to this dance if the table is ever rebuilt again.

## Phase 13 · context for agents

### internal/contextpack/policy.go · one policy for the skill and the server
- refers-to: embed.go :: `ContextSkill`; internal/contextpack/policy.go :: `LoadPolicy`, `ParsePolicy`, `Validate`; internal/server/server.go :: `Build`
- kind: explainer
- logged: v0.13.0 · 2026-10-09

`.claude/skills/contextual-compression/policy.json` is embedded into the binary and parsed at start; a file that breaks its rules stops `Build` with the list of problems, and `tools/skillcheck` runs the same validation in `make check`. The skill's prose quotes key names, never numbers, so the agents reading the skill and the server packing context cannot drift apart. `knowledge.kinds` must equal the list in the `knowledge` CHECK of migration 0025; widening it needs a table rebuild like 0024.

### internal/contextpack/pack.go · how a pack is filled
- refers-to: internal/contextpack/pack.go :: `Service.Pack`, `packing.passages`, `Policy.compress`, `packing.since`
- kind: explainer
- logged: v0.13.0 · 2026-10-09

Sections run in policy order. Each gets its share of the character budget (tokens times `chars_per_token`) plus whatever earlier sections left unused; a required section may take everything still free, and its overspend comes off the next. Rules are never cut. Passages come from `search.Passages` (any word, not all words) and are scored by the six weighted signals normalized by the sum of the weights, so `min_score` is a fraction; full-text rank is relative to the best candidate. A focus page goes in whole when it fits, else as heading-by-heading digests, else as an outline; a page included whole is not repeated as passages. Delta mode keeps only passages and related pages changed after the cutoff, which is an earlier pack of the same person, a time, or the agent's own last pack, clamped to `delta.max_hours`. Every pack is logged in `context_packs` with its sources so a delta can start from it.

### internal/contextpack/bank.go · knowledge, access and decay
- refers-to: internal/contextpack/bank.go :: `active`, `visible`, `Record`, `verdict`, `effective`, `rank`
- kind: warning-context
- logged: v0.13.0 · 2026-10-09

An item whose source page the reader cannot open is dropped before ranking, so knowledge never leaks a private page. Recording text at or above `duplicate_similarity` to an item of the same kind confirms that item instead; between the conflict and duplicate lines with the same subject it is recorded and returned as a conflict. Effective confidence halves every `half_life_days[kind]` since the last confirmation; items under `confidence.floor` are hidden, a contradiction that reaches the floor retires the item. Similarity is Jaccard over stemmed words without stopwords; single digits count as words because "3 attempts" and "5 attempts" are different facts.

### internal/mcp/tools_context.go · brief results and undo
- refers-to: internal/mcp/tools.go :: `Tool.Brief`; internal/mcp/tools_context.go :: `knowledgeRecord`, `knowledgeConfirm`; internal/contextpack/undo.go
- kind: explainer
- logged: v0.13.0 · 2026-10-09

Brief tools return their structured result as `structuredContent` but leave it out of the text, because the text already carries the rendered pack and doubling it would spend the budget the pack exists to save. Knowledge writes are recorded as `knowledge` steps of the agent run: undo deletes a recorded item or restores the previous confidence, counts and retirement of a confirmed, contradicted, superseded or retired one.

### cmd/form-md/bridge.go · the stdio bridge
- refers-to: cmd/form-md/bridge.go :: `bridge`, `forward`
- kind: explainer
- logged: v0.13.0 · 2026-10-09

Codex and other clients that start a command speak MCP over standard input and output, one JSON message per line. The bridge forwards each line to `/mcp` with the token, up to four at a time, and writes each answer back as one line; notifications get no answer. When a message carries `_meta` with a protocol version, the bridge sets `MCP-Protocol-Version`, `Mcp-Method` and `Mcp-Name` (base64-wrapped when not plain ASCII) because the 2026-07-28 transport refuses requests without them. Transport failures come back as JSON-RPC errors for the request's id, never as broken lines.

## Phase 14 · agent harness

### internal/claudecode/terminal.go · terminals
- refers-to: internal/claudecode/terminal.go :: `Terminals`, `Session.pump`, `Session.Attach`
- kind: explainer
- logged: v0.14.0 · 2026-10-16

Each terminal is a PTY (creack/pty, MIT) running the login shell, or Claude Code, in a folder; the reader goroutine keeps the last 256 KiB as scrollback so a reattached tab (reload, second window) replays its screen. Subscribers get chunks on buffered channels and a slow one drops chunks rather than stalling the shell. `gen` invalidates the pump of a killed process so a restart never mixes output. Output goes to the browser as binary WebSocket frames because a PTY chunk can end inside a UTF-8 sequence and a text frame must be valid UTF-8; the one control message, exit, is a text frame. Windows returns an error: creack/pty has no ConPTY path at this version.

### internal/claudecode/module.go · who may open a terminal
- refers-to: internal/claudecode/module.go :: `Service.enabled`, `Service.gate`
- kind: rationale
- logged: v0.14.0 · 2026-10-16

A terminal is a shell as the server's own user, so only a signed-in session of the instance owner may open one, and only when the server listens on loopback (the desktop and local-dev case) unless `FORMMD_TERMINAL=on` says otherwise; `FORMMD_TERMINAL=off` turns it off even locally. Agent tokens cannot reach the terminal routes. The WebSocket accepts loopback and Tauri origins only; the Vite proxy rewrites the origin to the backend's.

### internal/claudecode/chat.go · the chat
- refers-to: internal/claudecode/chat.go :: `chat.run`, `chatArgv`, `transcript`, `conversations`
- kind: explainer
- logged: v0.14.0 · 2026-10-16

Ported in behaviour from formlit's `cmd_chat.py`: `claude -p` with `--output-format stream-json --verbose --include-partial-messages`, form.md's MCP config and `--allowedTools mcp__formmd`, resuming the session id from the previous turn. Sub-agent events (with `parent_tool_use_id`) are skipped; text deltas, tool uses and tool results become `delta`, `tool` and `toolDone` SSE events; usage comes from `message_start` and `message_delta`, the context window from the result's `modelUsage`. The answer streams straight back on the POST rather than through the event hub, whose 128-event queue would close a subscriber on a long reply. Conversations started in Claude Code tabs live under `~/.claude/projects/<folder with non-alphanumerics as dashes>`; `transcript` reads one back for the Chat tab, skipping side chains, meta lines and tool-result echoes.

### internal/auth/apptoken.go · the in-app token
- refers-to: internal/auth/apptoken.go :: `AppToken`
- kind: rationale
- logged: v0.14.0 · 2026-10-16

Only the hash of an API token is stored, so the plain token for Claude Code inside form.md lives in memory for the process's life. On the first use after a start the old row with the same name is deleted and a new write token for all workspaces is made, so the account's token list shows exactly one "Claude Code in form.md" and revoking it there cuts the in-app agent off until the next start.

### internal/claudecode/bridge.go · window commands
- refers-to: internal/claudecode/bridge.go :: `Bridge.Run`, `Bridge.Answer`; internal/events/events.go :: `Hub.Send`
- kind: explainer
- logged: v0.14.0 · 2026-10-16

The window owns its commands, so the server only keeps the last catalogue each account's window sent and relays calls: `Run` publishes a `command.run` event with a call id to that account's sessions, waits up to a minute, and returns what the window posts to `/api/commands/result`. A credential that cannot write may still run commands whose spec says they change nothing. With no window online the call fails at once with a message telling the agent to ask the person to open form.md.

## Phase 15 · creative wishlist

### internal/pagestyle · page styles
- refers-to: internal/pagestyle/pagestyle.go :: `Clean`; internal/page/write.go :: `Update` (style); internal/workspace/workspace.go :: `Update` (page_style); internal/server/server.go :: `renderDocument` (style attrs); migrations/0026_page_style.sql
- kind: explainer
- logged: v0.15.0 · 2026-10-16

A style is five closed choices (font, size, width, spacing, align); `Clean` refuses anything else and stores an empty string for an all-default style, so the column never holds noise. A page's own style overlays the workspace default field by field. The first response for `/p/{id}` merges both and writes `data-style-*` attributes on the html element, skipping defaults, and the client writes the same attributes when it paints the page, so the first frame already has the final font and width. Fonts are system stacks in CSS, never fetched.

### workspace fonts
- refers-to: internal/pagestyle/pagestyle.go :: `CleanFonts`, `FontsCSS`; internal/workspace/module.go (GET fonts.css); internal/files/files.go :: `fontTypes`, `FontInWorkspace`; internal/static/static.go (Styles, Fonts); migrations/0027_fonts.sql
- kind: rationale
- logged: v0.17.0 · 2026-10-16

The content security policy allows styles only from this origin, so custom faces cannot be injected as an inline style; each workspace gets a small generated stylesheet instead, with one @font-face and one `data-style-font` rule per font, and the first response links it (and preloads the font file) only when the page's style uses a custom font. Font uploads are stored under the existing `workspace` purpose because `files.purpose` carries a CHECK constraint that a migration cannot widen without rebuilding the table; the upload's bytes must sniff as font/woff2, font/woff, font/ttf or font/otf, and `FontInWorkspace` checks that MIME type when a font list is saved. Families are limited to letters, digits, spaces, dots, dashes and underscores, which keeps them safe inside the generated CSS. `font-display: block` hides text for up to three seconds instead of swapping fonts mid-read. The system-font account setting strips every font preload, this one included.

### internal/site/module.go · files on freshly published pages
- refers-to: internal/site/module.go :: `fileAllowed`; internal/doc/service.go :: `LiveFile`
- kind: rationale
- logged: v0.18.0 · 2026-10-16

A public page renders from the live document, but `doc_refs` is written only when the document is materialised after its debounce, so a picture added a moment before publishing answered 404 to visitors. `fileAllowed` checks the stored refs first and then asks `LiveFile` whether any in-scope page that is loaded in memory references the file. `LiveFile` never loads a room (a page that is not in memory has already been materialised, so its refs are current) and holds the service lock while reading, in the same lock order as `unload`, so a room cannot be destroyed under it.

### internal/doc/board.go · board block
- refers-to: internal/doc/board.go :: `parseBoard`, `boardSVG`, `boardEncode`, `boardNode`; html.go, markdown.go, parse.go (board cases)
- kind: explainer
- logged: v0.18.0 · 2026-10-16

A board keeps its cards in one JSON attribute (`items`) plus a `height`, in board units 1000 wide. `parseBoard` is the single gate: at most 200 cards, positions inside the board, image cards only with an uploaded file reference, links only http or https without credentials, notes cut at 4000 bytes; anything else is dropped. The HTML is an SVG with a viewBox and one foreignObject per card, so the board scales with the column and positions need no inline styles under the public CSP. Markdown carries the board as `<!-- board:base64url(json) -->`, like buttons, and the writer notes every picture as a page file, which is what lets public pages serve them.

### internal/doc/embed.go · creative providers
- refers-to: internal/doc/embed.go :: `providers`
- kind: explainer
- logged: v0.18.0 · 2026-10-16

Canva uses the design's `/view?embed` address, Behance `/embed/project/{id}`, Sketchfab `/models/{uid}/embed`, Pinterest the `assets.pinterest.com/ext/embed.html?id=` widget page and Apple Music the `embed.music.apple.com` host with the same path. The editor's table in `web/src/engine/doc/schema.ts` must list the same patterns.

### internal/doc/remote.go · remote pictures imported after writes
- refers-to: internal/doc/remote.go :: `importRemote`, `SetRemoteImporter`; internal/doc/service.go :: `Write`; internal/doc/blocks.go :: `Apply`; internal/server/server.go (wiring to files.Import)
- kind: rationale
- logged: v0.20.0 · 2026-10-16

The client imports remote media only while a writable editor has the page open, so pages written by agents or the API kept remote pictures, which public pages and exports render as links. After a successful write the server imports up to 20 remote image srcs in a goroutine, as the writing principal (so the usual file authorization applies), and swaps them in one transaction. The fragment must be taken before `Transact`: calling `r.fragment()` inside a transaction locks the ygo document again and deadlocks the room. The importer is a function set from server.go so `doc` does not import `files`.

### internal/doc/blocks.go · person_editing guard
- refers-to: internal/doc/blocks.go :: `room.personIn`, `personTyping`, `editor.guard`, `replace`, `delete`; internal/doc/service.go :: `Write` (replace mode)
- kind: rationale
- logged: v0.21.0 · 2026-10-16

Inserts merge with typing, but a replace deletes and recreates the block, so it can lose keystrokes. The guard applies only to non-session principals (tokens and OAuth clients, which is how agents write) so a person's own browser and the undo of an agent run are never refused. It reads the `blocks` field of every awareness entry in the room; entries leave the room when their connection closes, so a stale cursor cannot block an agent for long. An empty block id means "anyone in this page", used for a whole-body replace.

### internal/site/render.go and internal/transfer/export.go · page style on public pages and exports
- refers-to: internal/pagestyle/pagestyle.go :: `Merge`, `Style.Attrs`; internal/site/render.go :: `load` (p.style, wsStyle), page main element; internal/transfer/export.go :: `exportHTML`
- kind: explainer
- logged: v0.22.0 · 2026-10-16

The page's main element carries `data-style-*` for every non-default field of the merged style, written by the server so the first paint is styled. `Attrs` re-checks each value against the allowed sets before it reaches HTML. Since v0.23.0 uploaded fonts are served too: `scope.fontFor` resolves a page's merged font to an `f:` ref only when the workspace font list has it, `fontsCSS` writes one @font-face and one `.site-main[data-style-font]` rule per font a page in scope uses, served at `<base>/fonts.css` (the public CSP forbids inline styles), and `fileAllowed` lets exactly those font files through. The HTML export still skips uploaded fonts because its scope has no workspace font list. The export merges the workspace default itself because the export scope has no workspace id.

### internal/doc/marks.go and internal/transfer/pdf.go · marks in Markdown and PDF
- refers-to: internal/doc/marks.go :: `marksEncode`, `marksDecode`, `MarksComment`; internal/doc/markdown.go (image); internal/doc/parse.go (marks comment); internal/transfer/pdf.go :: `pdfR.marks`, `lastImg`
- kind: explainer
- logged: v0.24.0 · 2026-10-16

An annotated picture is written as its image line, a blank line and `<!-- marks:base64url(json) -->`. The parser attaches that comment to the node emitted just before it only if that node is an image, and only after `parseMarks` accepts it, so a stray comment does nothing. The PDF export renders from Markdown: each picture records where it was placed, and the marks comment that follows it is drawn in that box with fpdf lines, rectangles, ellipses, a filled arrow head and bold text, scaled from the marks' own width and height.

### internal/collection/cache.go · row updates measured
- refers-to: internal/collection/cache.go :: `snap.put`, `snap.drop`
- kind: rationale
- logged: v0.24.0 · 2026-10-16

The audit flagged the linear scan in `put` and `drop`. A binary search was tried and timed against the previous build with 5000 rows: 1000 row updates through MCP took 19.0 s against 19.2 s, so the scan is not where the time goes (about 19 ms per row update is spent elsewhere, likely in the per-row write path). The code was left as it was; the per-row write cost is the thing to profile next.

### internal/collection/templates.go · template lookup
- refers-to: internal/collection/templates.go :: `templates`; migrations pages_templates index
- kind: rationale
- logged: v0.25.0 · 2026-10-16

`Get` lists a database's row templates and runs several times per row update, so the lookup must not depend on the row count. The query starts from the workspace's template pages through the partial index `pages_templates (workspace_id, is_template) WHERE is_template = 1` and joins the few hits to `collection_rows`; `CROSS JOIN` keeps SQLite from starting at `collection_rows_collection`, which visited every row. Found with a CPU profile of a throwaway build; 3000 updates of a 5000-row database went from 53.8 s to 19.1 s.

### internal/collection/service.go · client-made row ids
- refers-to: internal/collection/service.go :: `RowInput.ID`, `clientRowID`, `CreateRow`, `createRow`
- kind: rationale
- logged: v0.26.0 · 2026-10-16

Offline clients make the row id so later edits and deletes can refer to it before the server has seen the row. The id must have the form of server ids; a repeat in the same database returns the existing row (the replay may follow a create whose answer was lost), and an id held by any other page answers 409, which also stops one database's id being reused in another. Permission checks are unchanged: the caller still needs write access to the database.

The next cost in bulk updates is permission checking (about 37 percent of a profile): every row update walks the page's ancestor chain several times. A memo of chain results within one request would cut it without caching permissions across requests.

### internal/mcp/tools_write.go · bulk row tools
- refers-to: internal/mcp/tools_write.go :: `updateRows`, `addRows`, `addsOptions`
- kind: rationale
- logged: v0.27.0 · 2026-10-17

`update_rows` reads every row it was given with one `Query` by ids before the loop, both to check that each row belongs to the database and to capture the old values for the run's undo; write access is still checked per row by `UpdateRow`. Both bulk tools re-read the database description only when a row touched an option-bearing property, because the next row's option names must resolve to options the previous row may have just added. Together with the template lookup fix, 3000 updates of a 5000-row database went from 53.8 s (v0.24.0) to 8.6 s.

### internal/pagestyle · saved styles
- refers-to: internal/pagestyle/pagestyle.go :: `Preset`, `CleanPresets`, `ParsePresets`; internal/workspace/workspace.go (StylePresets); migrations/0028_style_presets.sql
- kind: explainer
- logged: v0.25.0 · 2026-10-16

A saved style is a name and a `Style`; each one goes through `Clean`, names are unique ignoring case (the later duplicate is dropped), and a workspace keeps up to 24. Choosing one replaces the page's own style with it rather than merging, so a preset always looks the same.

### internal/claudecode/bridge.go · CommandSpec decode
- refers-to: internal/claudecode/bridge.go :: `CommandSpec.UnmarshalJSON`
- kind: rationale
- logged: v0.20.0 · 2026-10-16

A catalogue entry without `mutates` decodes as true, matching the client default, so a missing flag can never let a read-only credential run a command.

### internal/collection/module.go · rows on the collection read
- refers-to: internal/collection/module.go (GET /api/collections/{id}, `rows`)
- kind: rationale
- logged: v0.19.0 · 2026-10-16

The first page of a table or list view stays at 100 rows for the window, which loads the rest as it scrolls. `rows` (clamped to 1000, the query ceiling) exists for the offline keeper, which needs one cacheable GET per view because the service worker cannot cache the POST queries the table uses for later windows.

### internal/family/family.go · form family bridge
- refers-to: internal/family/family.go (New, probe, Route, Module), internal/mcp/tools_family.go
- kind: rationale
- logged: v0.28.0 · 2026-10-17

formlit and formvis publish no descriptor and know nothing of form.md, so form.md reads what each already serves: formlit's `/api/health` and formvis's `/version` for the version, both apps' `/api/mcp/status` for the MCP endpoint, then `tools/list` on it. The `MCP-Protocol-Version: 2025-11-25` header is required by formlit for anything but initialize; the two identity headers are the ones each sibling logs callers by. Lookups are loopback-only (configured bases and reported endpoints alike), carry no credentials, wait 2 s, follow no redirects and are cached 30 s, so an agent asking repeatedly does not hammer a sibling and the route can never become an SSRF hop. form.md does not relay sibling tool calls: each sibling asks for consent in its own window, and a relay would hide which agent is calling. Routing is a plain keyword score (product vocabulary x3, plus one per word matched in each live tool's name and description) with a stopword list so words like "the" or "add" do not match every tool description. A configured entry that is not loopback falls back to the default base rather than failing, so a typo never points lookups off the machine.

### tools/bundlecheck · the up-front budget
- refers-to: tools/bundlecheck/main.go; Makefile (`web`)
- kind: rationale
- logged: v0.35.0 · 2026-10-17

The budget was a number in the build notes (250 KB at v0.9.0) that nobody measured, and the up-front script grew to 261.7 KB by v0.33.0 unnoticed. The check sums every `/assets/*.js` that `dist/index.html` loads or modulepreloads, gzipped with Go's default level, and fails `make build` (and so CI) above 260 KB. Go's gzip comes out about 1 KB larger than the `gzip` command on the same files, so this tool's number is the reference. 260 KB was chosen at v0.35.0 with the build at 255.1 KB, after the catalogs, the library view and the picker dialogs left the entry; what remains is the editor (ProseMirror, Yjs, the block node views) that the first page needs. `build_app` does not run it: a size check is a development gate, not an install step.

### internal/contextpack/pack.go · the Form family note
- refers-to: internal/contextpack/pack.go (FamilyHint, Pack, Render), internal/family/family.go (Suggest, Tuning), policy.json `family`
- kind: rationale
- logged: v0.29.0 · 2026-10-17

The note sits outside `sections` on purpose: giving it a share would take budget from every pack, while it appears only for picture work and is a few lines long; its characters are still added to `used_tokens`. `Suggest` checks the task against the siblings' vocabularies before any network call, so ordinary document tasks never wait on a sibling, and it drops a sibling that does not beat form.md's own score, so mixed tasks ("shot list for the album") stay in form.md unless the picture side dominates. The note goes through the caller's untrusted frame because the tool names in it come from the sibling. The weights, timeout and cache moved from constants into `policy.json` so the skill and the server agree, as the rest of the pack does.

### internal/auth/service.go · per-credential revoke
- refers-to: internal/auth/service.go (OnCredentialRevoke, CredentialRevoked, EndSession), internal/auth/account.go (handleDeleteToken, handleDeleteSession), internal/oauth/oauth.go (revokeGrant, revoke), internal/events/events.go, internal/doc/service.go and internal/claudecode/module.go (CloseCredential), internal/server/server.go
- kind: rationale
- logged: v0.30.0 · 2026-10-17

Before this, only a whole-user revoke closed live connections, and terminal attachments were not closed even then: a session signed out from another device kept its event stream, its sync socket and a live shell until the browser closed them. Every principal already carries one `CredentialID` (the session id, the API token id, or the OAuth grant id), so each revoke path names the id it removed and the three connection owners close what was opened with it. Session deletes use `DELETE ... RETURNING id` so the ids closed are exactly the rows removed, including "sign out other sessions". Revoking one OAuth access token closes its grant's streams too: they are keyed by grant, and the client reconnects with a fresh token if the grant still stands. Terminals are only detached, never killed, because the shell belongs to the owner, not to the session. Cancelling the attach context closes the socket at once (coder/websocket closes on a cancelled read), so no close frame is sent; the client sees the drop and its reconnect is refused.


### internal/auth/service.go · expiry closes live connections
- refers-to: internal/auth/service.go (SweepSessions, session resolution expiry branch), internal/server/server.go (sessionSweep), internal/auth/account.go (handleDeleteSession), internal/ops/encrypt.go (ResetOwner)
- kind: rationale
- logged: v0.39.0 · 2026-10-17

The v0.30.0 revoke covered sign-out and deletes but not expiry: a session that ran past `auth.session_days` kept its event stream, sync socket and terminal attachment, because the sweep deleted rows without naming them and ran only every 30 minutes. The sweep now uses `DELETE ... RETURNING id`, runs on its own one-minute ticker, and passes the ids to `CredentialRevoked`; the lazy expiry delete on a request does the same. `handleDeleteSession` runs the hooks before returning a row error, so ids already removed are still closed. `admin reset-owner` runs in another process and cannot reach the server's hooks, so it takes the data directory lock and refuses while a server runs.

### internal/family/family.go · sibling answers are cut down
- refers-to: internal/family/family.go (sameHost, toolName, clip, probe, Route, Mount)
- kind: rationale
- logged: v0.39.0 · 2026-10-17

A sibling's answers reach agents inside `context_pack` and `family_route`, so they are treated as input from another program, not as trusted text. The reported MCP endpoint must sit on the host and port form.md looked the sibling up at, so a process on another loopback port cannot be named as the sibling's server. Tool names outside `[A-Za-z0-9_.-]{1,64}` are dropped because they are printed as identifiers an agent may call; versions are cut to 32 runes and descriptions to 200 runes so a byte cut never leaves broken UTF-8. The descriptor builds its URLs with `BaseURL`, which honours `instance.public_url`, and is sent `no-store` because it changes with the proxy and settings.

### internal/oauth/oauth.go · access-token expiry closes the grant's streams
- refers-to: internal/oauth/oauth.go (ExpireAccess, resolve, Purge), internal/server/server.go (sessionSweep)
- kind: rationale
- logged: v0.39.1 · 2026-10-17

Streams opened with an OAuth access token are keyed by grant, not by token, so closing on each token's expiry would cut a client that already refreshed and opened its streams with the new token. `ExpireAccess` runs on the one-minute sweep, deletes expired access tokens with `RETURNING grant_id`, and revokes a grant's connections only when that grant has no unexpired access token left. A stream opened with an old token therefore lives on while the client holds a fresh one for the same grant; that is the same principal with the same scope and workspaces, so nothing is reachable that the live token could not reach. Refresh tokens are left to `Purge`, since deleting them does not end a connection.

### internal/family/family.go · one lookup per sibling at a time
- refers-to: internal/family/family.go (Service.flight, probe)
- kind: rationale
- logged: v0.39.2 · 2026-10-17

Every `context_pack` with picture words and every `GET /api/family` probes both siblings when the 30-second cache is cold, and an agent fanning out calls could start one lookup per call against a sibling that answers slowly. `flight` holds a channel per sibling while a lookup runs; later callers wait on it (or on their own context) and then read the cache. golang.org/x/sync's singleflight would do the same but is not a dependency, and the loop is short. A lookup whose caller's context ended is not cached, since its "not running" would be wrong for the next 30 seconds.

### internal/auth/apptoken.go · the in-app agent token
- refers-to: internal/auth/apptoken.go (AppToken), internal/claudecode/module.go (claudeArgv, the chat route)
- kind: rationale
- logged: v0.39.2 · 2026-10-17

The token was cached in a `sync.Map` for the life of the process with no check, so revoking it in Account settings cut the in-app agent off until a restart, and two Claude tabs opened at once could each mint a token, leaving the cache holding the one the database had already deleted. A mutex now serialises minting, and a cache hit is checked against `api_tokens` by hash; a missing row mints a new token under the same name. The map holds one string per owner who used the agent, so it needs no pruning.

### internal/collection/formula/funcs.go · bounded pattern cache
- refers-to: internal/collection/formula/funcs.go (reCache, reCount, compileRe)
- kind: rationale
- logged: v0.39.2 · 2026-10-17

`test`, `match`, `replace` and `replaceall` patterns can be built from row values, so the compiled-pattern cache had no bound. It is cleared once it passes 512 entries; a clear costs at most a recompile of the patterns in use, which is cheap next to an unbounded map in a long-running server.

### internal/auth/service.go · the credential is checked again once a connection is registered
- refers-to: internal/auth/service.go (CredentialLive), internal/events/events.go (Hub.live, Handler, Mount), internal/doc/conn.go (after register), internal/claudecode/module.go (attach)
- kind: rationale
- logged: v0.39.3 · 2026-10-17

The middleware checks the credential when the request arrives, but a long-lived connection is only registered for revoke hooks a moment later, after the WebSocket handshake or the stream header. A sign-out, token delete, grant revoke or account disable landing in that gap fired its hooks before the connection was in any list, so the connection stayed open. Each of the three owners now asks `CredentialLive` after registering: the session row (unexpired), the API token row, or the grant (unrevoked, with an unexpired access token), plus the user not disabled. Any revoke after registration reaches the hooks; any before it fails this check. It costs two indexed reads per new connection. The event hub gets the check from its module's `Mount` because the hub itself has no auth service.

### internal/search/pdf.go · text of referenced PDF files
- refers-to: internal/search/pdf.go (SetPDF, pdfParts), internal/search/search.go (load, indexVersion), internal/files/files.go (PDFBytes), internal/transfer/pdftext.go (PDFText), internal/server/server.go (finder.SetPDF)
- kind: rationale
- logged: v0.41.0 · 2026-10-19

The indexer adds passages from the PDF files a page body references, found through `doc_refs` rather than `files.page_id`, so removing a PDF from the body drops its text at the next re-index and replacing it changes the body, which already re-indexes the page. The extractor is the importer's, exported as `transfer.PDFText`, so imported and uploaded PDFs read the same. Limits: 4 PDFs per page, 16 MiB read per file through a LimitReader, 200 KiB of text per page, passages split on words at `maxPassage`; the indexer is single-threaded, so one file is in memory at a time, and the extractor's own 2000-page cap still holds. Scanned PDFs have no text layer and give nothing. `indexVersion` moved to 2 so existing pages rebuild once. `PDFBytes` takes no principal because only the indexer calls it, after the page itself passed the gate.

### internal/files/files.go · the sweep reads doc_refs
- refers-to: internal/files/files.go (Sweep)
- kind: rationale
- logged: v0.41.0 · 2026-10-19

The sweep removed unattached attachment, cover and icon files once no page used them as cover or icon, so a file linked only from another page's body could go. The delete now also requires that no `doc_refs` row of kind `file` targets it. Files attached to a page are unchanged.

### internal/webhook/webhook.go · the page record in payloads
- refers-to: internal/webhook/webhook.go (Event.DataOmitted, UsePages, attach, maxRecordBytes), internal/server/server.go (hooks.UsePages)
- kind: rationale
- logged: v0.41.0 · 2026-10-19

`page.create` and `page.update` carry `page.Meta` under `data`, the object REST returns under `page`, attached before the body is marshalled so the signature covers it. Over 64 KiB the record is dropped with `data_omitted: "size"`. Private and teamspace pages get `data_omitted: "restricted"`: a hook belongs to the workspace and its admin may not be able to read those pages. Row and comment events still carry ids only, because row values live in the collection service, not in `Meta`.

### internal/store/store.go · stale WAL files
- refers-to: internal/store/store.go (Open)
- kind: rationale
- logged: v0.41.0 · 2026-10-19

The database file was already created 0600; SQLite creates `-wal` and `-shm` with the main file's mode. A `-wal` or `-shm` left 0644 by an older version or a crash kept that mode, so `Open` sets any existing one to 0600 before opening.

### internal/order/order.go · After increments instead of halving
- refers-to: internal/order/order.go (After, bump)
- kind: rationale
- logged: v0.41.0 · 2026-10-19

`After` took the midpoint between the key and the top of the space, which added a character every five appends. `bump` increments the last character below the top digit and truncates after it, appending a digit only when the whole key is the top digit, so growth falls to about one character per 61 appends. Keys are compared as strings, so every existing key keeps its place and no migration is needed; a key that never grows would need a fixed-width integer part and a migration. The client's optimistic `keyBetween(a, "")` still halves until the server's key arrives.

### internal/answer/web.go · the web as an answer source
- refers-to: internal/answer/web.go (webOn, search, gather, readable, scrub, systemWeb), internal/answer/answer.go (New, the answer handler, options), internal/settings/settings.go (ai.web.mode, ai.web.url, ai.web.key)
- kind: rationale
- logged: v0.41.0 · 2026-10-19

The web is opt-in twice: an admin sets a search service (off by default) and the asker ticks it per question, so no workspace sends traffic out unless both happened. Only the question is sent to the search service, squashed and capped at 300 characters, never workspace text. Result pages go through the guarded fetcher with per-page and overall caps and are reduced to readable text (script, style, nav, header, footer, aside and forms dropped); a page that fails keeps its search snippet. Web text is framed like workspace text, numbered inside `<sources>` after the workspace sources so citations share one number space, with `<sources` neutralised so a page cannot close the block, under a system prompt that says to ignore instructions in sources. The search call uses its own netguard client that allows private addresses, because a LAN SearXNG is the expected setup; it only reaches the admin's configured address and only parses JSON, while result pages keep the full guard. The key is a sealed Secret behind the existing admin settings endpoint. Connectors are not built: `internal/family` discovers sibling tools but the siblings expose no search contract (id, title, snippet, URL) and family has no `tools/call` client.

### internal/site/render.go · boards and galleries on published pages
- refers-to: internal/collection/public.go (PublicTable, PublicRow, PublicGroup, publicCover), internal/site/render.go (table, cardsHTML, cardHTML, cardBody, boardHTML, galleryHTML)
- kind: rationale
- logged: v0.41.0 · 2026-10-19

`PublicTable` carries the view type, groups and covers, and the groups come from the same `bucketise` the app uses, so a published board has the app's column order, hidden groups, hide-empty and option colours. The renderer stays server HTML with no script, which the public CSP requires. Covers follow the view's cover setting and reserve space with `aspect-ratio`. A cover taken only from a files property may be refused by the public file allow-list and show as a broken picture; not yet exercised. Calendar, timeline, list and feed views still publish as a table.

### internal/collection/service.go · grouped query offset
- refers-to: internal/collection/service.go (Query, the group branch of makeGroups)
- kind: fix
- logged: v0.41.0 · 2026-10-19

The ungrouped branch clamped a negative offset; the group branch used it as given and indexed `rows[-1]`, a panic per request behind the recover middleware. Both branches now clamp at zero.

### internal/oauth/oauth.go · a blank scope means write
- refers-to: internal/oauth/oauth.go (normScope)
- kind: decision
- logged: v0.41.0 · 2026-10-19

The v0.41.0 defect audit asked whether a client that sends no scope should get read rather than write. It is left as write: MCP clients connect without a scope and need to write, the consent screen names the scope the person grants, and the person can still choose a read-only grant. Changing it would make every scope-less agent read-only after its next sign-in; that is a product decision for the owner, not a defect fix.

### internal/claudecode · run Claude Code the way formlit does
- refers-to: internal/claudecode/claude.go (ServerName, ToolPrefix, Note, Launch, childEnv, MCPConfig), internal/claudecode/terminal.go (agentDir, halt, Restart, Scrollback, RunningClaude), internal/claudecode/chat.go (Turn, Live, subscribe, fan), internal/claudecode/module.go (openClaude, /api/claude/open, /api/claude/chat/events, terminal input, read and resize), cmd/form-md/bridge.go (autostart, launchServer), cmd/form-md/detach_unix.go, detach_other.go
- kind: rationale
- logged: v0.42.0 · 2026-10-19

`--allowedTools mcp__form-md` and the login-shell `Launch` are ported from formlit's Claude Code arguments: without the first a terminal tab asked for every tool, and the login shell lets nvm or volta installs of claude find node. The server is `form-md` everywhere because the docs, the skill and .mcp.json already used that name; a second name registered one server twice and gave the note the wrong tool names. Chat events fan out from the run goroutine, not the request, so a reload cannot orphan an answer; `Live` is replaced as a whole copy, never mutated, so snapshots stay race-free; the POST stream stays because the hub's 128-event queue drops long replies. Restart halts without closing subscribers so a second window keeps its tab, and the old pump waits on a stale generation so the killed child is reaped. A prompt typed into a running tab is `prompt + "\r"` in one write, as formlit does, behind its own endpoint so `POST /api/terminal` always creates a tab. `form-md mcp` starts a server only for loopback http with a known data directory, so a bridge launched from a random folder never creates a stray ./data; setsid lives in build-tagged files because release targets include Windows.

### internal/icon/icon.go and migration 0029 · the sparkle retired
- refers-to: internal/icon/icon.go (Normalize), internal/comment/comment.go (Reactions, canonReaction), internal/comment/pagereact.go, internal/page/write.go, internal/page/teams.go, internal/workspace/workspace.go (cleanIcon), internal/site/render.go (iconSpan), internal/doc/html.go, internal/store/migrations/0029_retire_sparkle.sql
- kind: rationale
- logged: v0.42.0 · 2026-10-19

The owner banned the three-star sparkle app-wide. Bolt replaces it for page icons and reactions because it exists in both the glyph and icon sets and is not a star; the reaction keeps its "Brilliant" label so no catalog changes. `Normalize` runs before `Valid` on every write path, so old clients and agents that send sparkle still work while `Valid` stays strict and sparkle is not an accepted name. Documents are not rewritten: the public renderer normalizes callout icons when it reads them, which leaves CRDT history alone. The migration deletes a sparkle reaction where the same person already has bolt before renaming the rest, because the primary key (target, user, icon) would otherwise refuse the rename.
