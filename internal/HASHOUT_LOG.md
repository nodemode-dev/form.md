# HASHOUT_LOG · backend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the Go side of the repo (`cmd/`, `embed.go`, `internal/`, `tools/`). Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to the repo root.

Current build version: v0.4.0
Last updated: 2026-10-08 · documents: CRDT rooms, sync socket, materialiser, Markdown, first-paint HTML

---

## Org-navi-map

```
form.md/
├── embed.go                         package formmd: embeds web/dist and NOTICES.md
├── go.mod, go.sum                   module github.com/nodemode-ai/form.md, Go 1.26+
├── NOTICES.md                       generated third-party notices, embedded, served at /notices
├── Makefile                         build, web, notices, server, release, check, clean
├── form-md, form-md.bat             dev boot: Go server on :8750 plus Vite on :5750
├── form-md_oc_install (+ .bat)      installs both sides
├── cmd/
│   └── form-md/
│       └── main.go                  subcommands (serve, version, help), wiring, exit codes
├── internal/
│   ├── HASHOUT_LOG.md               this file
│   ├── admin/
│   │   └── admin.go                 /api/admin/*: users, roles, access, ownership, emergency, stranded, settings, mail test
│   ├── app/
│   │   ├── app.go                   Deps (incl. Middleware), Module interface, Run (listen, serve, graceful shutdown)
│   │   ├── version.go               Version (set by -ldflags), Product name
│   │   ├── health.go                GET /healthz
│   │   └── notices.go               GET /notices
│   ├── audit/
│   │   └── audit.go                 audit_log writer: actor kind, action, workspace, page, field changes
│   ├── auth/
│   │   ├── password.go              argon2id hash and verify, concurrency cap, dummy hash
│   │   ├── totp.go                  TOTP check with replay step, otpauth URL, QR as SVG, recovery codes
│   │   ├── service.go               Service, session cookies, credential middleware, handler wrappers
│   │   ├── users.go                 setup, login, logout, signup, policy, /api/me, password, email, CreateAccount
│   │   ├── account.go               TOTP enrolment, API tokens, signed-in sessions
│   │   ├── oidc.go                  Google, Microsoft and generic OIDC sign-in, SSOProviders
│   │   ├── invites.go               invitations: create, list, cancel, look up, accept
│   │   └── module.go                route table for /api/auth, /api/setup, /api/me, invites
│   ├── config/
│   │   └── config.go                FORMMD_* env vars, defaults, validation
│   ├── doc/
│   │   ├── schema.go                root fragment name, mark order, block ids, attribute readers
│   │   ├── room.go                  a page's CRDT: load, apply, persist, compact, awareness relay
│   │   ├── conn.go                  the /sync socket: frames, open and leave, write check, queues, identity
│   │   ├── service.go               rooms and sockets, materialiser, copier, Markdown read and write, backlinks, routes
│   │   ├── markdown.go              CRDT to Markdown, plain text, snippet, thumbnail and references
│   │   ├── parse.go                 Markdown to blocks (goldmark) and blocks to CRDT operations
│   │   └── html.go                  CRDT to the editor's HTML for the first frame
│   ├── events/
│   │   └── events.go                Hub: sequenced events, replay ring, audience filter, GET /api/events (SSE)
│   ├── files/
│   │   └── files.go                 uploads, sniffing, access-checked serving at /files/{id}, sweep
│   ├── icon/
│   │   └── icon.go                  the one rule for icon values: a name from the shared set or an uploaded file
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
│   ├── order/
│   │   └── order.go                 fractional base62 sort keys: Between, After, Before, Valid
│   ├── page/
│   │   ├── page.go                  Meta, Service, access check, private owner, title and tag cleaning, emit
│   │   ├── tree.go                  Tree, Trash listing, Get with path, tags and tag colours
│   │   ├── write.go                 create, update, move (in and across workspaces), duplicate, trash, restore, purge
│   │   ├── extras.go                favorites with order, visits, recent pages
│   │   ├── content.go               SetContent from the materialiser, Readable, Titles
│   │   └── module.go                route table for pages, trash, tags, favorites, recent
│   ├── policy/
│   │   ├── principal.go             Principal: one shape for every credential, context helpers
│   │   └── gate.go                  Gate.Workspace access check, agent policy, emergency grants, memberships
│   ├── secrets/
│   │   └── secrets.go               AES-GCM sealing with data/secret.key, token minting and hashing
│   ├── server/
│   │   └── server.go                Build (wiring), Run, housekeeping, renderDocument, tile colours
│   ├── settings/
│   │   └── settings.go              typed instance settings table, validation, env fallback, snapshots
│   ├── static/
│   │   └── static.go                embedded client: rendered document, cache headers, gzip, ETag, CSP
│   ├── store/
│   │   ├── store.go                 SQLite writer connection plus reader pool, close with checkpoint
│   │   ├── tx.go                    Tx, Querier, ErrNotFound, NoRows, Bool
│   │   ├── migrate.go               numbered embedded migrations, one transaction each
│   │   ├── instance.go              instance id and last app version
│   │   └── migrations/
│   │       ├── 0001_instance.sql    instance key/value table
│   │       ├── 0002_accounts.sql    accounts, sessions, workspaces, tokens, invites, emergency, audit
│   │       ├── 0003_pages.sql       pages, favorites, page visits, tag colours, files
│   │       └── 0004_docs.sql        document state, update log, references
│   └── workspace/
│       ├── workspace.go             workspaces, members, access, rules, proposals, emergency log
│       └── module.go                route table for /api/workspaces
└── tools/
    ├── oc_install (+ .bat)          Go side install: go mod download
    ├── boot_backend (+ .bat)        Go side dev boot: go run on 127.0.0.1:8750
    └── notices/
        └── main.go                  generates NOTICES.md from go list and web/node_modules
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
