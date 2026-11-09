# HASHOUT_LOG · demos/publish · form.md

Rationale and explainers for the writing and publishing demo. Source files stay comment-free; what they would have said lives here. Paths are relative to `demos/publish/`.

Current build version: v0.49.3
Last updated: 2026-11-07

---

## Org-navi-map

```
demos/publish/
├── index.html           build output: the app's own shell markup inlined, classic deferred script
├── assets/              build output: publish.js (one IIFE), style.css (fonts inlined), media/ (photos, WebP) and media/pictures.js (the same pictures as base64)
├── README.md            how to open, what it shows, how to rebuild
├── CREDITS.md           every picture and its licence
├── HASHOUT_LOG.md       this file
├── vite.config.mjs      plugins and build settings, root and outDir inside this folder
└── src/
    ├── index.html       page skeleton with a shell marker the config fills from web/index.html
    ├── main.ts          mirrors web/src/main.ts start(): createEngine with injected fetch, platform and storage, the same extensions and mount calls
    ├── platform.ts      hash routing adapter, fake WebSocket speaking the y-sync framing, fake EventSource
    ├── guards.ts        intercepts export, import and the published view button
    ├── importer.ts      turns dropped or chosen .md files into pages
    ├── preview.ts       the in-page published view
    ├── main.css, publish.css  the app's stylesheet plus the few demo rules
    └── server/          in-memory stand-in for the Go server: db, ids, api, seed, site, files, packs, export, zip
```

## Why this shape

- Demo 1 mounts pieces of the client. Writing and publishing is a flow across the whole window (editor, comments pane, history, share and publish dialogs, topbar), so this demo runs the real window as formvis's demo does. The shell markup is taken from `web/index.html` at build time, so the sidebar, topbar, page view and comments pane are the app's own DOM.
- `main.ts` follows the real `start()` with injected `fetch`, platform and storage. The only seams are the ones the engine already offers.
- `server/` answers the routes the window calls: pages, page detail, comments, versions, sites, sharing, files, export, search, favorites, inbox and health. Anything else returns quiet empty JSON, so no request ever leaves the page.
- A fake `WebSocket` speaks the y-sync framing against one server-side `Y.Doc` per page, so the real ySync editor, undo, comment anchors (Yjs relative positions) and history snapshots work unchanged. A fake `EventSource` sends the hello event and raises `comments` events.
- The published view is `renderPage` re-implemented in TypeScript (`server/site.ts`) with the real `web/src/site/site.css`, so the in-page preview, the HTML export and what the Go server would serve share one layout. The password gate, theme, font and sub-page choices of the real dialog all take effect.
- Exports are built in the browser: Markdown through the real `MarkdownWriter`, with pictures copied to `files/`; HTML as a single file with pictures inlined and the site CSS inlined, or a zip with sub-pages. `zip.ts` writes stored zips.
- Pictures are read for export and for the published view, so they ship twice: as plain files for display and as `pictures.js` base64, loaded on demand by a script tag (Chrome refuses to fetch local files from file://).
- `/files/<id>` URLs in the client are rewritten at build time to relative paths (`offlineUrls` in the config), and `location.origin` and `location.host` read a fixed demo address, so share links read `fielddesk.example`.
- The embed block needs a remote iframe, so the page uses bookmark cards and local media instead.

## Zero flash

The shell is in the HTML and the theme is set before first paint. The config adds `has-cover` to the page view in the shell so the cover's reserved band exists before the page data arrives, and `publish.css` gives the page description a two-line minimum when a cover is present, so the page head does not move when the title and description fill in. Under a 4x CPU throttle the layout-shift total of the first load is about 0.03, from the tag chip arriving beside the tag field.

## Operations that work

Typing and the format bar in the real editor; comment on a selection, reply, resolve, react, mention; page history with the block diff, restore and save a version by hand; the Publish dialog (address, title, description, picture, theme, font, sub-pages, search indexing, password) and the Published view in desktop and phone widths, with the password gate; unpublish with its confirmation; Export as Markdown (zip with pictures) and HTML (single file or zip with sub-pages); import `.md` files from the sidebar; reset by reload.

## Not in the demo

PDF export, real sign-in and sharing by email, live collaboration between people, and the Claude dock need the Go server. Exporting to PDF shows a short notice.

## Inside the app (v0.51.0)

The app embeds this build and opens it in a sandboxed frame from Home, Help and Find a command (see `web/HASHOUT_LOG.md` #src-ui-demos-ts). The shell taken from `web/index.html` now carries Home's Demos section, so `publish.css` hides `[data-ref="home-demos-sec"]` here, and the window code registers no demo commands when it runs from file:// or under `/demo/` or `/demos/`, so a demo never opens a demo. In the sandbox the workspace cookie cannot be written; `rememberWorkspace` now ignores that.

## Rebuild

From `demos/publish`: `node ../../web/node_modules/vite/bin/vite.js build --config ./vite.config.mjs`. The config imports `web/` by relative path and never writes outside `demos/publish`.
