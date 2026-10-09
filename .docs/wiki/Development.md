# Development

How the repository is laid out, how to run both halves while changing them, and the conventions every change follows. The server is Go with no cgo (pure-Go SQLite); the client is vanilla TypeScript built with Vite, with no UI framework. A release binary embeds the built client from `web/dist` and `NOTICES.md` through `embed.go`.

## Layout

| Path | What lives there |
|---|---|
| `cmd/form-md/` | `main.go` (`serve`, `version`, `help`, server wiring) and `tool.go` (operator commands) |
| `embed.go` | Embeds `web/dist` and `NOTICES.md` into the binary |
| `internal/app/` | Module interface, server lifecycle (listen, TLS, ACME, shutdown), `/healthz`, `/notices`, `Version` |
| `internal/server/` | Wires every module together; renders the app document the client boots from |
| `internal/config/`, `internal/settings/` | `FORMMD_` variables; typed instance settings with sealed secrets |
| `internal/store/` | SQLite writer connection and reader pool, numbered migrations in `migrations/` |
| `internal/crypt/`, `internal/cryptvfs/`, `internal/secrets/` | At-rest encryption, the encrypting SQLite VFS, `secret.key` sealing |
| `internal/ops/` | `backup`, `restore`, `doctor`, `encrypt`, `admin reset-owner`, the data directory lock |
| `internal/auth/`, `internal/policy/`, `internal/oauth/` | Sign-in, sessions, tokens, SSO; the principal and the workspace gate; the OAuth 2.1 server |
| `internal/workspace/`, `internal/page/`, `internal/order/` | Workspaces and members; the page tree, trash, favorites; fractional sort keys |
| `internal/doc/` | Page bodies as CRDT rooms, the `/sync` socket, Markdown in and out, first-paint HTML |
| `internal/collection/` | Databases: schema, values, the query engine, views, forms, CSV; `formula/` is the formula language |
| `internal/files/`, `internal/transfer/` | Uploads; importers and exporters |
| `internal/mcp/`, `internal/agent/`, `internal/agentrun/`, `internal/ai/`, `internal/aifill/`, `internal/answer/` | MCP server, workspace and custom agents, undoable runs, model providers, AI autofill, cited answers |
| `internal/httpx/`, `internal/static/`, `internal/apidoc/` | Router and middleware; serving the embedded client and `/offline.json`; the OpenAPI document |
| `internal/` (the rest) | Search, comments, inbox, sites, analytics, webhooks, audit, SCIM, automations, templates and the smaller services, one package each |
| `web/src/engine/` | Headless core: state, page and database stores, document sync (`doc/`), live events, router, commands, keymap, API client, i18n |
| `web/src/extensions/` | Feature modules registered with the engine: `account`, `automation`, `blocks` (the block editor), `database` (views), `pages`, `shell`; `registry.ts` |
| `web/src/ui/` | The DOM layer: sidebar, top bar, page view, dialogs, settings, offline, tokens |
| `web/src/styles/`, `web/src/site/`, `web/src/sandbox/` | Component styles; published-site script; the sandboxed Mermaid renderer |
| `web/public/` | Served as is: `sw.js` (service worker), `maps/` |
| `tools/` | `boot_backend`, `oc_install` (Go side), `notices/` (generates `NOTICES.md`) |
| `deploy/`, `install.sh`, `Dockerfile` | Systemd unit, installer, container image |
| `.github/workflows/` | `ci.yml` (every push to `main` and every pull request), `release.yml` (tags) |

## Dev loop

Requirements: Go 1.26 or newer, Node.js 22 or newer.

```sh
./form-md_oc_install     # go mod download, then npm ci in web/
./form-md                # backend on :8750 and Vite on :5750
```

`./form-md` starts `tools/boot_backend` (`go run ./cmd/form-md serve` with `FORMMD_ADDR=127.0.0.1:8750` and `FORMMD_DATA=./data` unless you set them) and `web/boot_frontend` (`npm run dev`), and stops both on exit. Open `http://127.0.0.1:5750`: Vite serves the client with hot reload and proxies `/api`, `/healthz`, `/notices`, `/events`, `/mcp`, `/files`, `/oauth`, `/.well-known` and the `/sync` WebSocket to the backend, rewriting the `Origin` header so sign-in and the sync socket accept it. Port 8750 alone answers pages with "the web client is not built" until you run `make build`.

Windows has `form-md_oc_install.bat` and `form-md.bat`; they have not been run on a Windows host yet.

Before pushing:

```sh
make check               # gofmt, go vet, wiki check, skill check (policy.json), client type check
make build               # the same build CI runs
```

CI (`.github/workflows/ci.yml`) runs `make check`, `make build`, cross-compiles for linux/arm64, darwin/arm64 and windows/amd64, then a smoke run: starts the binary, waits for `/healthz`, runs `form-md backup` against it and runs `form-md doctor`.

## Conventions

### Commits

Every commit subject reads:

```
type(scope): summary (x.y.z)
```

- `type` is one of `feat`, `fix`, `perf`, `refactor`, `docs`, `build`, `chore`.
- `scope` is the area the change mostly lives in, usually a package or client area: `web`, `server`, `doc`, `editor`, `database`, `collection`, `auth`, `api`, `mcp`, `ops`, `offline`, `hashout`, `readme`, `release`, and so on.
- `summary` starts lower case (names keep their capitals), says what changed in plain words, and has no trailing period.
- `(x.y.z)` is the version the commit is building toward, without the `v`: every commit after `v0.10.1` that leads to the next release ends in `(0.11.0)`.
- The body explains what and why in short bullets, then what was verified and how.
- No emojis.

The release commit is `chore(release): vX.Y.Z (x.y.z)`; see `BRANCHING.md` at the repo root.

### No code comments: HASHOUT_LOG.md

Source files carry no comments. Everything a comment would have said (rationale, explainers, TODOs, removed code) is written in a HASHOUT log instead, indexed by file:

- `internal/HASHOUT_LOG.md` for the Go side (`cmd/`, `embed.go`, `internal/`, `tools/`),
- `web/HASHOUT_LOG.md` for the client.

Each starts with an org-navi-map of every file. Read the relevant log before changing code, and add your reasoning to it in the same change (commits of that kind use the `docs(hashout)` scope).

### No self-written tests: TEST_PLAN.md

The repository holds no unit or integration tests, on purpose. A test written in the same sitting as the code it checks encodes the author's own assumptions. When a change needs testing, the deliverable is a `TEST_PLAN.md`: what should be tested (behaviours, inputs, edge cases, failure modes, acceptance criteria), written for someone with no context of the change, with no test code in it. Test plans are kept with the planning documents outside the tree. Tests are written only when the owner asks for them. Verification that was done by hand or by probes is recorded in the commit body.

### Other rules

- Surgical edits: change only the lines a change needs.
- No emojis anywhere, including Markdown; no em dashes.
- Every colour and layer comes from the token files; no raw `z-index` outside the tokens file.
- No icon libraries and no UI framework; icons are inline SVG.
- Changing behaviour an operator sees (a command, a variable, a route, a file in the data directory) means updating the wiki page in `.docs/wiki/` in the same pull request.

## Related

`BRANCHING.md` · [[Install]] · [[Configuration]] · [[Home]]
