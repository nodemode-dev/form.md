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
  the wiki and in [BRANCHING.md](BRANCHING.md). Leave the version alone; a maintainer
  bumps it when releasing.
- `.claude/skills/contextual-compression/policy.json` is read by the server as well as
  the skill: changing a number there changes the server.
- Some features were declined for patent risk and will not be added back: form answers
  written into each mapped property of the new row, form automations that analyse the
  stored properties of the new row, and synced blocks held as child blocks under the
  synced block. Instead, a submission is stored as one response record on the row
  and the columns are read from it, form automations run on the response-received
  event, and synced content stays in its own fragment addressed by key.

## Maintainers

Form.md is maintained by:

- the [nodemode-dev](https://github.com/nodemode-dev) organisation on GitHub
- [@nodemode-ai](https://github.com/nodemode-ai)
- [@gbarnai](https://github.com/gbarnai)

Reach them through issues and pull requests.

## How changes land

Only a maintainer pushes to `main`. Everyone else forks the repository and opens a pull request into `main`. A pull request merges when all three checks pass on an up-to-date branch and a maintainer has approved it after its last push.

## Releases

A release is a version, and only a maintainer makes one: after the version is bumped on
`main`, a maintainer cuts a `compile_vX.Y.Z` branch, and formcompiler
(`nodemode-dev/formcompiler`) builds every platform and publishes them together. No
push from a contributor starts a release. See [BRANCHING.md](BRANCHING.md).

## Licensing

Form.md is licensed under the Apache License 2.0 ([LICENSE](LICENSE), SPDX
`Apache-2.0`). By contributing, you agree that your contribution is licensed under the
same terms. Only add code you have the right to contribute, and name any third-party
code and its licence in the pull request.

## Patents and other people's IP

Some features are deliberately not built, or were taken out, because they come too close to someone else's patent, trademark, copyright or design. The maintainers hold that line with counsel; contributors do not decide it, and nobody here redesigns a feature on their own reading of a patent. This is engineering diligence, not legal advice.

### Features this project will not implement

- Form answers written into each mapped property of the new row. A submission is stored as one response record on the row and the columns are computed from it when read; a person's cell edit is an override that wins.
- Form automations that analyse the stored properties of the new row. They run on the response-received event and their condition reads the answers as submitted; a form row does not raise the plain row-added event.
- Synced blocks held as child blocks with parent pointers under the synced block. Synced content stays in a separate shared fragment addressed by key; the synced node holds only the page and the key.

A pull request that adds one of these, under any name, through any dependency, plug-in, model, template, agent tool or plain code, is closed with a link to this section. Lifting a decline needs the maintainers' written decision after a review, recorded here first.

### Rules for every contribution

- Patents: do not add a feature, or a dependency that provides one, that reproduces a method you know to be patented. If you are unsure, open an issue that describes what the feature does, not which patent it resembles. Patent numbers stay out of code, commits, comments, logs and changelogs.
- Trademarks: other vendors' product and feature names appear only to name a file format or compatibility ("opens <format> files", "for use with"). No command, menu, panel, tool, preset, template, theme or agent tool takes another product's feature name, and no text says "like X" or "the way X does it". Describe what the feature does.
- Copyright: never copy another product's help, tooltip, dialog or documentation text (link to it instead); never trace or import vendor icons; never bundle vendor templates, themes, presets, brushes, swatches, LUTs, materials, sample files, sound libraries, stock media or fonts; colour-book names only when reading a person's file. Compatibility layers carry names and signatures only. Test fixtures come from this repository's own scripts.
- Licences: every new dependency, binary, model, font, profile or data file gets a row in `NOTICES.md` before it is used. GPL stays outside the distributed bundle, AGPL nowhere, LGPL dynamically linked and replaceable, MPL files unedited. CC BY credited, CC BY-SA only unmodified, CC NC and ND never. Models with gated or non-commercial terms are optional downloads, never bundled.

## Conduct

Everyone taking part follows the [code of conduct](CODE_OF_CONDUCT.md). Security
problems go through [SECURITY.md](SECURITY.md), never a public issue.
