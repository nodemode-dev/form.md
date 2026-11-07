# form.md offline demo

A browser demo of form.md that runs from a double click. Open `demo/index.html` (file://) or serve the folder with any static server. There is no backend, no network request and no CDN: scripts, styles, fonts and samples are all in this folder.

## What it shows

The demo runs the real form.md client pieces against an in-memory store instead of the Go server.

- Markdown pages in the real editor core: the ProseMirror schema from `web/src/engine/doc/schema.ts`, the Markdown reader and writer from `web/src/extensions/blocks/markdown.ts`, the input rules and structure plugins, and a local Yjs document per page for sync and undo.
- A database in table and board views, painted with the app's own cell renderer (`web/src/extensions/database/cells.ts`), option chips, number and date formats, and the app's CSS.
- The app's design tokens, dark and light themes, inline icon set, menus, dialogs and toasts.

Operations that run on your own input:

1. Open or drop a `.md`, `.markdown` or `.txt` file: it becomes an editable page. A leading `# Heading` becomes the page title.
2. Edit with Markdown shortcuts (`#`, `-`, `1.`, `[]`, `>`, `**bold**`), the format strip, Ctrl or Cmd with B, I, U, E and Z.
3. Outline from headings, word count and reading time, find and replace on the page, search across all pages in the sidebar.
4. Export a page as Markdown (download, copy or view) through the real Markdown writer.
5. Open or drop a `.csv` or `.tsv` file: columns are typed by inference (number, date, checkbox, select, status, email, URL, text) and the file becomes a database.
6. Table view: sort by clicking a header, filter rules, group by a property, per-column calculations (sum, average, median, min, max, counts), inline cell editing, add and delete rows.
7. Board view grouped by any select or status property, with drag between lanes.
8. Table to database turns the first Markdown table on a page into a database. A database exports as CSV or a Markdown table, or inserts into a page.

The demo opens on a small workspace of real material, each piece credited in [CREDITS.md](CREDITS.md):

- A field notebook welcome page over a Pexels observatory photograph.
- Galileo on the Moon, 1610: two passages from Project Gutenberg's Sidereal Messenger with Galileo's own engravings, and a table of the Galilean moons from NASA.
- Moon missions: NASA photographs of Apollo 8, Apollo 11 and Artemis II with NASA's own captions.
- Dark sky weekend: a field plan with a gear list, a targets table and Pexels photographs.
- A database of 66 real planets from the NASA Exoplanet Archive, in table and board views.

Click a photo on a page to enlarge it. Pages carry a cover image like the real app. Photographs are plain files in `assets/media/`; the exoplanet rows are a script pack (`assets/media/exoplanets.js`) loaded on demand, because Chrome will not fetch local files from file://. If that pack cannot load, the demo falls back to a generated Trail survey database. Reset restores the workspace. Nothing is stored beyond the open tab except the theme and sidebar choice in localStorage.

## Not in the demo

Accounts, sharing, realtime collaboration, comments, publishing, the other five database views, formulas, relations, the MCP server and the Claude Code dock need the Go server and are not part of this demo. Slash menu and block handles are not wired.

## Rebuild

The committed files (`index.html` and `assets/`) are the build output. To rebuild, from the repository root:

```
cd web
npm ci
rm -f ../demo/assets/demo.js ../demo/assets/style.css
npx vite build --config vite.demo.config.ts
```

Source lives in `demo/src/`. The Vite config is `web/vite.demo.config.ts` so it resolves the web workspace's dependencies; it reads `demo/src/index.html`, bundles one classic script (no ES modules, so file:// works) and writes into `demo/`. To develop with reload, run `npx vite --config vite.demo.config.ts` in `web/` and open http://localhost:5760.

Type check the demo sources with `web/node_modules/.bin/tsc --noEmit -p demo/tsconfig.json`.

## Dependencies

No new dependencies. Everything comes from `web/package-lock.json`. Media and data are licensed as listed in [CREDITS.md](CREDITS.md), not under the repository licence.
