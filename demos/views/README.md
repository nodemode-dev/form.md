# form.md database views demo

Demo 2 of 3. A browser demo of form.md's database views that runs from a double click. Open `demos/views/index.html` (file://) or serve the folder with any static server. There is no backend, no network request and no CDN: scripts, styles, fonts, photos and data are all in this folder.

The other demos: [demo 1, editor and databases](../../demo/index.html) and [demo 3, writing and publishing](../publish/index.html). A quiet link row at the foot of the sidebar moves between them.

## What it shows

The real form.md database client (`web/src/extensions/database`) runs against an in-browser stand-in for the Go server. Three linked databases hold one real dataset, US crewed spaceflight from Freedom 7 to Apollo 17: Crewed missions (27 rows), Astronauts and Programs.

- Eight views of the same missions: All flights (table), By trajectory (board), Gallery, Crewed by program (grouped list), Calendar, Timeline, Hours by program (stacked chart), Launch heatmap. The plus tab adds any other view type.
- Relations: a mission links to its program and crew, and the pair updates the other side.
- Rollups: each astronaut rolls up missions, hours, first launch and last splashdown; each program rolls up flights, flight hours, longest flight and crew seats.
- Formulas: Days, Length, Crew-hours, Career span, Experience and a text Summary, recomputed after every edit.
- Filter rules, sorts, group and sub-group, search, per-column calculations, board drag between lanes, calendar paging, in-place cell editing, add and delete rows, row pages with photos.

Operations on your own input: open or drop a `.csv` or `.tsv` file and it becomes a database with typed columns and table, board and chart views. Export as CSV from the More menu saves the current view. Reset data restores the dataset. Nothing is stored beyond the open tab except the theme and sidebar choice in localStorage.

Photographs are NASA public-domain images, credited in [CREDITS.md](CREDITS.md).

## Rebuild

The committed files (`index.html` and `assets/`) are the build output. From the repository root:

```
cd demos/views
node ../../web/node_modules/vite/bin/vite.js build --config ./vite.config.ts
```

Source is in `src/`. The config reads `web/` by relative path and writes only into this folder. No dependencies are added; everything comes from `web/package-lock.json`.

## Not in the demo

Forms, dashboards, map and graph views need more than the browser provides; automations, buttons and AI autofill need the server. See [HASHOUT_LOG.md](HASHOUT_LOG.md).
