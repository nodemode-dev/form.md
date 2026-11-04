<p align="center">
  <img src="assets/app-icon/form.md-256.png" width="128" height="128" alt="form.md icon">
</p>

<h1 align="center">form.md</h1>

<p align="center">
  <b>Workspace; documents, databases and realtime editing for people and agents.</b><br>
  Pages, seven database views, comments, sharing and publishing,<br>
  in one Go binary with one SQLite file. Self-hosted, no account elsewhere.
</p>

<p align="center">
  <img alt="version v0.47.9" src="https://img.shields.io/badge/version-v0.47.9-eef0f4?style=flat-square">
  <img alt="Go 1.26+" src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="TypeScript and Vite" src="https://img.shields.io/badge/TypeScript-Vite-3178c6?style=flat-square">
  <img alt="macOS, Windows, Linux" src="https://img.shields.io/badge/macOS%20%C2%B7%20Windows%20%C2%B7%20Linux-native-2f7bf5?style=flat-square">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/license-Apache--2.0-3a3a3a?style=flat-square">
  <img alt="Status: active" src="https://img.shields.io/badge/status-active-d69e2e?style=flat-square">
</p>

<p align="center">
  <a href="https://formmd.formai.studio/"><b>Download form.md on formai.studio</b></a> ·
  <a href="https://github.com/nodemode-dev/formcompiler/releases/tag/form.md-v0.47.7">Latest release</a> ·
  <a href="#run-it-from-source">Run it from source</a> ·
  <a href="#the-form-family">All Form apps</a>
</p>

<br>

<p align="center">
  <img src="docs/images/form.md-hero.png" alt="A trail report page in form.md with a cropped sunrise photo as the cover, a green callout and the Trail survey database below it" width="100%"><br>
  <sub>An original report page: a cropped photo cover, a callout and a live database view, in the editor.</sub>
</p>

<p align="center">
  <a href="#features">Features</a> · <a href="#everything-in-the-box">In the box</a> · <a href="#built-for-agents">Agents</a> · <a href="#under-the-hood">Under the hood</a> ·
  <a href="#download">Download</a> · <a href="#run-it-from-source">Run from source</a> · <a href="#the-form-family">Form family</a>
</p>

<br>

<table>
  <tr>
    <td width="25%" valign="top"><h3>One binary</h3>A static Go binary with the web client embedded and one SQLite file beside it. No database server.</td>
    <td width="25%" valign="top"><h3>Server-held documents</h3>Every page is a CRDT the server applies itself, so an API edit merges into what a person is typing.</td>
    <td width="25%" valign="top"><h3>Rows are pages</h3>Seven database views on one query engine, with relations, rollups and formulas.</td>
    <td width="25%" valign="top"><h3>Agent-ready</h3>An MCP server with OAuth, a REST API and a command line drive the same workspace.</td>
  </tr>
</table>

<br>

## Features

<table>
  <tr>
    <td width="50%" valign="top">
      <img src="docs/images/form.md-database-table.png" alt="The Trail survey database as a table with zone, status, length, condition, surveyed and volunteers columns" width="100%">
      <br><sub>The same rows, as a table with typed columns.</sub>
      <h3>Seven database views</h3>Every row is a page. Switch one query between table, board, list, gallery, calendar, timeline and form.
    </td>
    <td width="50%" valign="top">
      <img src="docs/images/form.md-database-board.png" alt="The Trail survey database as a board grouped by status: Open, Needs repair and Closed" width="100%">
      <br><sub>A board grouped by the Status property.</sub>
      <h3>Boards that follow the data</h3>Group by any select, drag rows between columns and add views without copying anything.
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <img src="docs/images/form.md-templates.png" alt="The template gallery with 90 templates in categories and live previews of dashboards" width="100%">
      <br><sub>The template gallery.</sub>
      <h3>Starts with 90 templates</h3>Documents, databases, dashboards and example projects, each previewed before you add it.
    </td>
    <td width="50%" valign="top">
      <h3>Edit together, share exactly</h3>Comments, suggested edits, history and six access levels, enforced by the server on every route. Publish a page as a site when you want it public.
    </td>
  </tr>
</table>

## Everything in the box

<table>
  <tr>
    <td width="33%" valign="top"><h4>Documents</h4>Block editor · slash menu · callouts · toggles · tables · columns · code · mentions · backlinks</td>
    <td width="33%" valign="top"><h4>Databases</h4>Table · board · list · gallery · calendar · timeline · form · heatmap · graph · charts · automations</td>
    <td width="33%" valign="top"><h4>Formats</h4>Opens: Markdown · HTML · CSV · ENEX · OPML · Word · Excel · Saves: Markdown · HTML · PDF · CSV</td>
  </tr>
</table>

## Built for agents

The window's commands, the REST API and the MCP server all act on the same pages and databases.

```sh
./form-md cli list              # every command in the open window
./form-md cli describe ID       # one command with its typed parameters
./form-md mcp                   # MCP server on stdio for Claude Code, Codex and Cursor
```

`.mcp.json` and `.cursor/mcp.json` are already wired in this repo. The server also answers MCP over HTTP at `/mcp` and publishes its REST reference at `/api/docs`. See [`AGENTS.md`](AGENTS.md).

## Under the hood

- **Go server:** pure-Go SQLite, a Yjs-compatible CRDT in Go, one writer connection and a query-only reader pool.
- **Web client:** vanilla TypeScript and Vite with ProseMirror and Yjs, no UI framework, embedded into the binary.
- **Encrypted at rest:** an XTS-AES-256 SQLite VFS and AES-256-GCM uploads, with a key kept outside the data directory.
- **Tested:** 38 Go tests across the server, plus a TypeScript check (`./form-md check`).

## Download

**Get the app:** download form.md from [formai.studio](https://formmd.formai.studio/). That is the easiest way to install it.

**Latest release:** [form.md v0.47.7 on FormCompiler](https://github.com/nodemode-dev/formcompiler/releases/tag/form.md-v0.47.7). **Keep it updated:** [FormPortal](https://github.com/nodemode-dev/formportal) installs, updates and rolls back every Form app from the same signed catalogue.

| Platform | Builds |
|---|---|
| macOS Apple silicon | arm64 |
| macOS Intel | x64 |
| Linux | x64, arm64 |
| Windows | x64 |

All form.md builds are compiled and released from [`nodemode-dev/formcompiler`](https://github.com/nodemode-dev/formcompiler/releases), not from this repo. This repo holds the source.

## Run it from source

Prefer it in your browser, or want to contribute? Run the web UI straight from this repo.

**Requires:** Go 1.26 or newer and Node.js 22 or newer.

```sh
git clone git@github.com:nodemode-dev/form.md.git
cd form.md
./form-md_oc_install   # installs Go modules and the web client's packages
./form-md              # server at http://127.0.0.1:8750, web client at http://127.0.0.1:5750
```

`./form-md desktop dev` opens the desktop window instead, and `./build_app --server-only` compiles `bin/form-md`. On Windows use `form-md_oc_install.bat` and `form-md.bat`. The first visit shows a set-up screen that creates the owner account.

| Variable | Purpose |
|---|---|
| `FORMMD_ADDR` | Listen address, default `127.0.0.1:8750` |
| `FORMMD_DATA` | Data directory for the database and uploads, default `./data` |

> [!IMPORTANT]
> **Status:** active development. Instance-wide settings such as mail and single sign-on live in the app's Instance settings, and the wiki in `.docs/wiki` covers install, TLS, backups and operations.

**Contributing:** branch from `main`, keep changes to source here, and leave packaging and releases to FormCompiler. See [`BRANCHING.md`](BRANCHING.md) and [`AGENTS.md`](AGENTS.md).

## Repository

| Path | What it is |
|---|---|
| `cmd/form-md/` | Entry point and subcommands (serve, backup, restore, doctor, encrypt, keygen, admin, mcp, cli) |
| `internal/` | The Go server: pages, documents, databases, search, auth, sharing, MCP, transfer |
| `web/` | The TypeScript and Vite client |
| `src-tauri/` | The desktop shell around the server binary |

## The Form family

form.md is one of the **Form apps**: native creative tools that install, update and roll back through FormPortal and can all be driven by agents.

| | App | What it's for | Download | Code |
|:-:|---|---|---|---|
| <img src=".github/assets/family/form.md.png" alt="" width="32" height="32"> | **form.md** | Documents, databases and realtime editing together | [formai.studio](https://formmd.formai.studio/) | [GitHub](https://github.com/nodemode-dev/form.md) |
| <img src=".github/assets/family/format.png" alt="" width="32" height="32"> | **FormAT** | Chat and agent workspace on your key | [formai.studio](https://format.formai.studio/) | [GitHub](https://github.com/nodemode-dev/format) |
| <img src=".github/assets/family/formCAD.png" alt="" width="32" height="32"> | **FormCAD** | Drafting, parametric parts and NURBS surfaces | | [GitHub](https://github.com/nodemode-dev/formCAD) |
| <img src=".github/assets/family/formcel.png" alt="" width="32" height="32"> | **FormCel** | Spreadsheets agents can read and edit | [formai.studio](https://formcel.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formcel) |
| <img src=".github/assets/family/formcraft.png" alt="" width="32" height="32"> | **FormCraft** | Generate images, video, audio, 3D and worlds | | [GitHub](https://github.com/nodemode-dev/formcraft) |
| <img src=".github/assets/family/formcut.png" alt="" width="32" height="32"> | **FormCut** | Multi-track video editing with no upload | [formai.studio](https://formcut.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formcut) |
| <img src=".github/assets/family/formDAM.png" alt="" width="32" height="32"> | **FormDAM** | Photo library and asset manager | [formai.studio](https://formdam.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formDAM) |
| <img src=".github/assets/family/formdeck.png" alt="" width="32" height="32"> | **FormDeck** | Presentations and pitch decks | [formai.studio](https://formdeck.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formdeck) |
| <img src=".github/assets/family/formdesign.png" alt="" width="32" height="32"> | **FormDesign** | Page layout for print and the web | [formai.studio](https://formdesign.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formdesign) |
| <img src=".github/assets/family/formencoder.png" alt="" width="32" height="32"> | **FormEncoder** | Media encoding queue with presets | [formai.studio](https://formencoder.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formencoder) |
| <img src=".github/assets/family/formfabric.png" alt="" width="32" height="32"> | **FormFabric** | Sewing patterns draped with cloth simulation | | [GitHub](https://github.com/nodemode-dev/formfabric) |
| <img src=".github/assets/family/formlit.png" alt="" width="32" height="32"> | **FormLit** | Photo library and raw development | [formai.studio](https://formlit.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formlit) |
| <img src=".github/assets/family/formmorpheus.png" alt="" width="32" height="32"> | **FormMorpheus** | Frame-by-frame and performed animation | | [GitHub](https://github.com/nodemode-dev/formmorpheus) |
| <img src=".github/assets/family/formNET.png" alt="" width="32" height="32"> | **FormNET** | See what your computer sends online | | [GitHub](https://github.com/nodemode-dev/formNET) |
| <img src=".github/assets/family/formpath.png" alt="" width="32" height="32"> | **FormPath** | Vector illustration with artboards and type | [formai.studio](https://formpath.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formpath) |
| <img src=".github/assets/family/formpdf.png" alt="" width="32" height="32"> | **FormPDF** | Edit, redact, sign and fill PDFs | [formai.studio](https://formpdf.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formpdf) |
| <img src=".github/assets/family/formpromo.png" alt="" width="32" height="32"> | **FormPromo** | Promo films from your own media | [formai.studio](https://formpromo.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formpromo) |
| <img src=".github/assets/family/formref.png" alt="" width="32" height="32"> | **FormRef** | Reference boards for creative work | [formai.studio](https://formref.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formref) |
| <img src=".github/assets/family/formrig.png" alt="" width="32" height="32"> | **FormRig** | Rig, skin and retarget any mesh | | [GitHub](https://github.com/nodemode-dev/formrig) |
| <img src=".github/assets/family/formsculpt.png" alt="" width="32" height="32"> | **FormSculpt** | 3D sculpting, remeshing and painting | [formai.studio](https://formsculpt.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formsculpt) |
| <img src=".github/assets/family/formsurface.png" alt="" width="32" height="32"> | **FormSurface** | Texture painting with PBR export | [formai.studio](https://formsurface.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formsurface) |
| <img src=".github/assets/family/formtools.png" alt="" width="32" height="32"> | **FormTools** | Viewers, batch tools and file utilities | | [GitHub](https://github.com/nodemode-dev/formtools) |
| <img src=".github/assets/family/formtxt.png" alt="" width="32" height="32"> | **FormTxt** | Word processing with .docx round trips | [formai.studio](https://formtxt.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formtxt) |
| <img src=".github/assets/family/formvfx.png" alt="" width="32" height="32"> | **FormVFX** | Motion graphics and VFX compositing | [formai.studio](https://formvfx.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formvfx) |
| <img src=".github/assets/family/formvis.png" alt="" width="32" height="32"> | **FormVis** | Layered image editing with local AI | [formai.studio](https://formvis.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formvis) |
| <img src=".github/assets/family/formwav.png" alt="" width="32" height="32"> | **FormWav** | Multitrack audio editing and mixing | [formai.studio](https://formwav.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formwav) |
| <img src=".github/assets/family/formweb.png" alt="" width="32" height="32"> | **FormWeb** | Visual website building with a CMS | [formai.studio](https://formweb.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formweb) |
| <img src=".github/assets/family/formportal.png" alt="" width="32" height="32"> | **FormPortal** | Install, update and roll back Form apps | [formai.studio](https://formportal.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formportal) |
| <img src=".github/assets/family/formtray.png" alt="" width="32" height="32"> | **FormTray** | One dock icon for every app | [formai.studio](https://formtray.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formtray) |
| <img src=".github/assets/family/formMCP.png" alt="" width="32" height="32"> | **FormMCP** | One MCP connection to the family | [formai.studio](https://formmcp.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formMCP) |
| <img src=".github/assets/family/formskills.png" alt="" width="32" height="32"> | **FormSkills** | Every app's agent skills in one place | [formai.studio](https://formskill.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formskills) |
| <img src=".github/assets/family/formbench.png" alt="" width="32" height="32"> | **FormBench** | Parity, testing and rankings for the family | [formai.studio](https://formranker.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formbench) |
| <img src=".github/assets/family/formslot.png" alt="" width="32" height="32"> | **FormSlot** | Codebase memory for coding agents | | [GitHub](https://github.com/nodemode-dev/formslot) |
| <img src=".github/assets/family/subform.png" alt="" width="32" height="32"> | **SubForm** | Workbench for parallel Claude Code agents | | [GitHub](https://github.com/nodemode-dev/subform) |
| <img src=".github/assets/family/formcompiler.png" alt="" width="32" height="32"> | **FormCompiler** | Builds, signs and releases every app | [formai.studio](https://formcompiler.formai.studio/) | [GitHub](https://github.com/nodemode-dev/formcompiler) |
| <img src=".github/assets/family/formpipe.png" alt="" width="32" height="32"> | **FormPipe** | Release pipeline and update feeds | | [GitHub](https://github.com/nodemode-dev/formpipe) |

<br>

---

## Maintainers

| Maintainer | Role |
|---|---|
| [@nodemode-ai](https://github.com/nodemode-ai) | Owner |
| [@gbarnai](https://github.com/gbarnai) | Co-owner |

Code owners are set in [`.github/CODEOWNERS`](.github/CODEOWNERS).

## License

form.md is licensed under the Apache License 2.0 ([`LICENSE`](LICENSE), SPDX `Apache-2.0`). Copyright 2026 NodeAgentc, Inc. Third-party notices: [`NOTICES.md`](NOTICES.md), which the binary also serves at `/notices`.

<sub>Screenshot content: a cropped CC0 photo from Poly Haven; the page and data are original.</sub>
