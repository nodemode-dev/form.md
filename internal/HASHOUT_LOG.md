# HASHOUT_LOG · backend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the Go side of the repo (`cmd/`, `embed.go`, `internal/`, `tools/`). Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to the repo root.

Current build version: v0.1.0
Last updated: 2026-10-07 · Phase 1 skeleton binary

---

## Org-navi-map

```
form.md/
├── embed.go                         package formmd: embeds web/dist and NOTICES.md
├── go.mod, go.sum                   module github.com/nodemode-ai/form.md, Go 1.26+
├── NOTICES.md                       generated third-party notices, embedded, served at /notices
├── cmd/
│   └── form-md/
│       └── main.go                  subcommands (serve, version, help), wiring, exit codes
├── internal/
│   ├── HASHOUT_LOG.md               this file
│   ├── app/
│   │   ├── app.go                   Deps, Module interface, Run (listen, serve, graceful shutdown)
│   │   ├── version.go               Version (set by -ldflags), Product name
│   │   ├── health.go                GET /healthz
│   │   └── notices.go               GET /notices
│   ├── config/
│   │   └── config.go                FORMMD_* env vars, defaults, validation
│   ├── httpx/
│   │   ├── router.go                ServeMux wrapper, middleware chain, /api/ 404 catch-all
│   │   ├── middleware.go            request id, panic recovery, access log, security headers
│   │   └── respond.go               JSON and error envelopes
│   ├── ids/
│   │   └── ids.go                   128-bit random ids, lowercase base32
│   ├── static/
│   │   └── static.go                embedded client: cache headers, gzip, ETag, CSP, SPA fallback
│   └── store/
│       ├── store.go                 SQLite writer connection plus reader pool, close with checkpoint
│       ├── migrate.go               numbered embedded migrations, one transaction each
│       ├── instance.go              instance id and last app version
│       └── migrations/
│           └── 0001_instance.sql    instance key/value table
└── tools/
    └── notices/
        └── main.go                  generates NOTICES.md from go list and web/node_modules
```

---

## Table of contents

- [embed.go](#embed-go)
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

---

## embed.go {#embed-go}

### embed.go · dist embed
- refers-to: embed.go :: `dist`, `Dist`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

`go:embed` cannot reach a parent directory, so the embed sits at the module root and `cmd/form-md` imports it. `all:web/dist` needs at least one file to exist, so `web/dist/.gitkeep` is committed and the Vite build re-creates it after emptying the folder (see the frontend log). Without a client build the binary still compiles and serves the API with a 503 explainer on page routes. `Dist` panics only if the embed path is wrong, which is a build-time mistake, not a runtime condition.

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
