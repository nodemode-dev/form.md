# HASHOUT_LOG · demo · form.md

Rationale and explainers for the offline demo. Source files stay comment-free; what they would have said lives here. Paths are relative to `demo/`.

Current build version: v0.49.1
Last updated: 2026-11-07 · v0.49.1 the media pass

---

## Org-navi-map

```
demo/
├── index.html           build output, classic deferred script, opens from file://
├── assets/              build output: demo.js (one IIFE), style.css (fonts inlined)
├── README.md            how to open, what it shows, how to rebuild
├── HASHOUT_LOG.md       this file
├── tsconfig.json        type check for demo/src (paths resolve to web/node_modules)
└── src/
    ├── index.html       shell markup copied in class names from web/index.html
    ├── main.ts, main.css  entry; main.css imports web/src/styles/main.css then demo.css
    ├── demo.css         the few classes the real CSS does not have
    ├── app.ts           shell: sidebar tree, routing by hash, page tools, import, drop
    ├── editor.ts        ProseMirror view over a per-page Y.Doc, format commands, find
    ├── dbview.ts        table and board views over an in-memory collection
    ├── store.ts         pages (Y.Doc) and collections, Markdown in and out, seeding
    ├── csv.ts           delimited reader, type inference, writers
    ├── query.ts         filter, sort, group and calculation functions
    ├── samples.ts       original sample Markdown and the procedural trail CSV
    ├── files.ts         file kind, text reading, download
    └── i18n-empty.ts    empty catalogue for the virtual:i18n import
web/vite.demo.config.ts  build config for the demo (port 5760, host false)
```

## Decisions

- The Go server holds pages, rows, CSV import, queries and export, so the demo replaces those with `store.ts`, `csv.ts` and `query.ts`. Everything the user sees on screen is the real client code or its CSS: schema, Markdown writer and reader, input rules, plugins, cell painter, menus, dialogs, toasts, tokens, icons.
- The whole `web/src/engine` and `ui/app.ts` stack assumes the server, so the demo mounts the editor and the cell painter directly rather than booting the engine. A stub `DbCtx` supplies the three fields the cell painter reads.
- Vite emits ES modules by default and Chrome blocks module scripts on file://. The demo build is one IIFE with dynamic imports inlined, and a small plugin turns the emitted module tag into `<script defer>`. Fonts are inlined into the CSS (limit 120000 bytes) so no font request crosses file:// origins.
- The Vite config lives in `web/` because Node resolves `vite` from there; its `root` points at `demo/src` and its `outDir` at `demo/` with `emptyOutDir: false`, so stale files are removed by hand (`rm -rf demo/assets`) before a rebuild. Bare imports from `demo/src` are resolved through `web/node_modules` by a resolver plugin.
- `data-auth="app"` on the html element is required: `auth.css` hides `.app` otherwise.
- Escape and the dialog close button are wired in `app.ts` with the layer stack's `closeTop`, because the real app does it through its command router.
- Views update in place: rows, cards, lanes, crumbs and tree rows go through `ui/list.ts` `reconcile`; cells are repainted by `paintCell` signature; the editor mount is created once and receives a new state per page. Inline editors overlay a cell and hide its value rather than rebuilding it.
- The Table view renders in flow mode and pages by 250 rows with a Load more button; the real virtual scroller needs the server query session.
- Page inserts from a database rebuild the target fragment with `prosemirrorToYXmlFragment` over the merged document, so an open editor receives them through its sync plugin.
- Format strip replaces the floating format bar, which needs the full editor context (comments, suggestions, AI).

## Media pass (v0.49.1)

- The demo now opens on real content: four pages and the exoplanet database in `src/content.ts`, with images in `assets/media/`. Licences and sources are in `CREDITS.md`.
- Images are plain relative files, not data packs: they are only displayed, never read for pixels, so file:// allows them. The editor's own image node renders them; each figure carries width and height so the space is reserved before the file decodes and nothing moves. Page covers use one persistent img whose src is swapped, and the Home cards do the same.
- The exoplanet rows are the one thing read as data, so they ship as `assets/media/exoplanets.js`, a script that sets `window.__formmdPacks["exoplanets.csv"]`. `src/packs.ts` injects it as a script tag and resolves null on error. `seed` awaits it before the first page opens; on null it builds the generated Trail survey from `samples.ts` instead. The Welcome, trail and meeting sample pages were removed because nothing used them any more.
- The pack and the page text are generated from NASA and Gutenberg sources by scratch scripts that are not in the repo; `CREDITS.md` says what was changed. Class is derived from radius by the generator, and the Class column comes before Method so the board groups by size class.
- Pages and databases carry a `cover` string. The cover markup mirrors the real `.cover` element so `pages.css` styles it as is.
- Clicking an image opens the app's own dialog with the larger image. The wrapper spans both columns of `.dialog__layout`, which has a fixed 196 px first column.
- The build instruction now removes only `demo.js` and `style.css`, because `assets/media/` lives inside `assets/` and `rm -rf` would delete it.
- `NOTICES.md` is generated by `go run ./tools/notices`; the demo media row added by hand will be dropped on regeneration unless the tool is taught to emit it.
- Not done: Webb and other ESA co-credited NASA images were left out because their reuse terms are not NASA-only. The Exoplanet Archive page states an acknowledgement, not a licence.

## Removed or not ported

- Floating format bar, slash menu, block handle, node views (callout icons, media, math, mermaid, toggles chrome), comments, suggestions, AI blocks, presence cursors.
- Database views other than table and board; formulas, relations, rollups; forms.
- Code highlighting (`hl.ts` pulls highlight.js).

## TODO

- Slash menu and block handle from `extensions/blocks` once they stop requiring `EditorCtx`.
- Chart and calendar views over the same in-memory collection.
