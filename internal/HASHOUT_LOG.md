# HASHOUT_LOG · backend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the Go side of the repo (`cmd/`, `embed.go`, `internal/`, `tools/`). Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to the repo root.

Current build version: v0.2.0
Last updated: 2026-10-07 · Phase 2 accounts, workspaces, policy and admin

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
│   │       └── 0002_accounts.sql    accounts, sessions, workspaces, tokens, invites, emergency, audit
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
