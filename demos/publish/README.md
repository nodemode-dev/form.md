# form.md writing and publishing demo

Demo 3 of 3. A browser demo of writing a long page in form.md and publishing it, run from a double click. Open `demos/publish/index.html` (file://) or serve the folder with any static server. There is no backend, no network request and no CDN: scripts, styles, fonts and pictures are all in this folder.

The other demos: [demo 1, editor and databases](../../demo/index.html) and [demo 2, database views](../views/index.html). A quiet link row at the foot of the sidebar moves between them.

## What it shows

The whole real form.md window runs against an in-browser stand-in for the Go server. The workspace is Field Desk. It opens on "The Moon, in the order we learned it", a long page with a cover, engravings, NASA photographs, a two-column row, a quote, a callout, a facts table, a toggle FAQ, user and page mentions and four open comment threads, with a child page, Sources and credits.

Things to try:

1. Edit the page: type, use the slash menu and the format bar, undo.
2. Select text and comment on it; reply, resolve, react, mention a teammate from the comments pane.
3. More menu, Page history: five saved versions from a rough first draft to the current text, a block diff against now, restore, or save a version by hand.
4. Share, Publish to the web: choose the address, theme, font, sub-pages, search indexing and a password, publish, then open the Published view in desktop or phone width. With a password the reader sees the gate first. Unpublish takes it down.
5. More menu, Export: Markdown (a zip with the page and its pictures) or HTML (one file with pictures and styles inlined, or a zip with sub-pages).
6. Sidebar, Import: choose `.md` files and each becomes a page you can edit, publish and export.

Pictures are Pexels photographs, NASA images and Galileo's engravings from Project Gutenberg, credited in [CREDITS.md](CREDITS.md). The essay, comments and teammates are written for the demo. Nothing is stored beyond the open tab except the theme and sidebar choice in localStorage; reload resets the workspace.

## Rebuild

The committed files (`index.html` and `assets/`) are the build output. From the repository root:

```
cd demos/publish
node ../../web/node_modules/vite/bin/vite.js build --config ./vite.config.mjs
```

Source is in `src/`. The config reads `web/` by relative path and writes only into this folder. No dependencies are added; everything comes from `web/package-lock.json`.

## Not in the demo

PDF export, sign-in, sharing by email, collaboration between real people and the Claude dock need the Go server. See [HASHOUT_LOG.md](HASHOUT_LOG.md).
