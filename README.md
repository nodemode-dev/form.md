<p align="center">
  <img src="assets/form-md_logo.svg" alt="form.md logo" width="160" />
</p>

<h1 align="center">form.md</h1>

<p align="center"><b>One workspace for people & agents: documents, databases & realtime editing in a single binary.</b></p>

<p align="center">
  <img src="https://img.shields.io/badge/version-v0.1.0-9a8cff" alt="version v0.1.0" />
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26+" />
  <img src="https://img.shields.io/badge/Node.js-22%2B-339933?logo=nodedotjs&logoColor=white" alt="Node.js 22+" />
  <img src="https://img.shields.io/badge/license-Apache--2.0-555555" alt="license Apache-2.0" />
</p>

form.md is a self-hosted workspace where people & AI agents write documents, run databases & edit together in realtime. It ships as one static Go binary w/ its web client embedded & one SQLite file beside it: no database server, no cache, no object store. The server holds the real collaborative document (a Yjs CRDT in pure Go), so an agent's edit merges into what a person is typing instead of resetting it. This repo is at **Phase 1 of 10**: the binary, storage & client shell exist; pages, the editor, databases & the agent surface follow per the roadmap.

**Stack:** Go 1.26 · pure-Go SQLite (modernc) · vanilla TypeScript & Vite · no UI framework · **Scale:** 11 MB static binary, 8 KB JS & 4 KB CSS gzipped for the shell

## Highlights

- **One static binary** built w/ `CGO_ENABLED=0`: client, migrations & license notices embedded; five release targets (linux & darwin on amd64/arm64, windows amd64)
- **Reads never queue behind writes** · one SQLite writer connection plus a query-only reader pool, numbered migrations in one transaction each, WAL checkpointed on clean shutdown
- **Zero first-paint flash** · the theme & sidebar state are applied by an inline boot script before CSS paints; verified frame by frame under 4x CPU & 500 kbps throttling, dark & light
- **Zero layout shift** · every async value has reserved space; measured CLS 0 across load & every shell interaction
- **One stacking scale, one popover primitive, one layer stack** · no raw `z-index` outside the tokens file; Escape, outside-press, inert focus trapping & focus return handled in one place
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

Then open http://127.0.0.1:8750.

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

form-md reads environment variables only; there is no config file & nothing secret is needed yet. `form-md help` prints the same table.

| Variable | Purpose |
|---|---|
| `FORMMD_ADDR` | Listen address, default `127.0.0.1:8750`; set `0.0.0.0:8750` to accept other machines |
| `FORMMD_DATA` | Data directory for the database & uploads, default `./data` |
| `FORMMD_TLS_CERT` | TLS certificate path; set together w/ `FORMMD_TLS_KEY` |
| `FORMMD_TLS_KEY` | TLS private key path; set together w/ `FORMMD_TLS_CERT` |

## Architecture

```
form.md/
├── cmd/form-md/        entry point & subcommands (serve, version, help)
├── embed.go            embeds web/dist & NOTICES.md into the binary
├── internal/
│   ├── app/            module registry, server lifecycle, /healthz, /notices
│   ├── config/         FORMMD_* environment variables
│   ├── httpx/          router, middleware chain, JSON errors
│   ├── static/         embedded client: caching, gzip, ETag, CSP, SPA fallback
│   ├── store/          SQLite writer & reader pool, migrations
│   └── ids/            128-bit random ids
├── web/                vanilla TypeScript client
│   ├── index.html      static shell skeleton & pre-paint theme boot
│   └── src/
│       ├── engine/     headless state, commands, keymap, health
│       ├── extensions/ feature modules registered w/ the engine
│       ├── ui/         DOM layer: layers, floating, tooltip, toasts, theme tokens
│       └── styles/     component styles on the token system
├── tools/              Go side install & boot scripts, NOTICES generator
├── assets/             repo-level art for this README, not part of the build
├── Makefile            build, release, check
└── form-md             dev boot for both sides
```

`internal/HASHOUT_LOG.md` & `web/HASHOUT_LOG.md` map every file & record the reasoning that would otherwise be code comments. Read them before changing code.

## Status of subsystems

- **Accounts & sign-in:** not built yet (Phase 2). Do not expose an instance to a network until then.
- **Pages, editor, realtime, databases, search, agents (MCP):** not built yet (Phases 3 to 8).
- **Windows scripts:** written but not yet run on a Windows host.

## Development

- One `main` branch for now; a branch model arrives w/ the wiki in Phase 10
- Conventions: surgical edits only, zero emojis anywhere incl. markdown, every colour & layer from the token files, no icon libraries, no UI framework, semver tags from `main`
- `make check` runs gofmt, `go vet` & the client type check

## Status

Phase 1 (skeleton binary & client shell) is complete at this version. Phase 2, accounts, roles & the single policy gate, is next. The full roadmap is kept outside the tree in the planning sidecar that ships w/ each release zip.

## Contributors

- `Owner` · nodemode-ai

### Licensing

Apache-2.0, see [LICENSE](LICENSE). Third-party components & their licenses are listed in [NOTICES.md](NOTICES.md), which the binary also serves at `/notices`.
