# Contributing to Form.md

Thank you for helping. Form.md aims to be one self-hosted workspace where people and
agents write documents, run databases and edit together, in a single binary, and it
gets there through real use and real contributions.

## Where to start

- **Use it and report what goes wrong.** A clear bug report, with steps, the version
  and your platform, is one of the most useful things you can give.
- **Look at the open issues.** Comment on one before starting larger work, so nobody
  builds the same thing twice.
- **Translations** are welcome: Form.md ships in 26 languages, with one folder per
  language in `web/src/i18n` and the server-rendered German in `internal/i18n`.

## Setting up

Clone the repository and run `./form-md_oc_install`, then `./form-md` (or
`form-md_oc_install.bat` and `form-md.bat` on Windows). The first checks Go and Node
and installs what the client needs; the second starts the server and the client. The
README's Quick Start has the details and the requirements, and the Development page of
the wiki maps every package.

The code lives in:

| Folder | What is there |
| --- | --- |
| `cmd/form-md` | the binary: `serve`, the operator commands, the `mcp` stdio bridge and `cli` |
| `internal` | the server (Go): storage, documents, databases, accounts, sharing, the MCP server and the rest |
| `web` | the client (TypeScript, Vite, ProseMirror and Yjs) |
| `src-tauri` | the desktop app (Rust, Tauri) |
| `tools` | build, notice, wiki and skill checks |
| `deploy` | the installer's service unit and operations files |
| `.docs/wiki` | the wiki's source |

## Before you open a pull request

Run the same checks CI runs:

```sh
(cd web && npm ci --no-audit --no-fund && npm audit --audit-level=low)
make check
make build VERSION=v0.0.0-ci
go test -count=1 ./...
```

`make check` runs gofmt, go vet, the wiki check, the skill check and the TypeScript
check. CI also cross-compiles the binary for Linux, macOS and Windows, starts it for a
smoke test, and runs the window smoke tests in `web/tests/window` with pytest and
Playwright.

- Keep each pull request to one change, with tests for what it adds or fixes.
- Write commit subjects the way the history does: `feat(web): ...`,
  `fix(collection): ...`, `docs(...)`, with the scopes listed on the Development page of
  the wiki and in [BRANCHING.md](BRANCHING.md). Leave the version alone; the maintainer
  bumps it when releasing.
- `.claude/skills/contextual-compression/policy.json` is read by the server as well as
  the skill: changing a number there changes the server.
- Some features were declined for patent risk and will not be added back: form answers
  written into each mapped property of the new row, form automations that analyse the
  stored properties of the new row, and synced blocks held as child blocks under the
  synced block. `CLAUDE.md` says how each works instead.

## How changes land

Only the maintainer pushes to `main`. Everyone else forks the repository and opens a pull request into `main`. A pull request merges when all three checks pass on an up-to-date branch and the maintainer has approved it after its last push.

## Releases

A release is a version, and only the maintainer makes one: after the version is bumped on
`main`, the maintainer cuts a `compile_vX.Y.Z` branch, and formcompiler
(`nodemode-dev/formcompiler`) builds every platform and publishes them together. No
push from a contributor starts a release. See [BRANCHING.md](BRANCHING.md).

## Licensing

Form.md is licensed under the Apache License 2.0 ([LICENSE](LICENSE), SPDX
`Apache-2.0`). By contributing, you agree that your contribution is licensed under the
same terms. Only add code you have the right to contribute, and name any third-party
code and its licence in the pull request.

## Patents and other people's IP

A feature may be held back or removed if it would infringe someone else's patent or
other intellectual property. The maintainers decide.

## Conduct

Everyone taking part follows the [code of conduct](CODE_OF_CONDUCT.md). Security
problems go through [SECURITY.md](SECURITY.md), never a public issue.
