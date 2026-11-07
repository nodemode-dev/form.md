# HASHOUT_LOG · demos/views · form.md

Rationale and explainers for the database views demo. Source files stay comment-free; what they would have said lives here. Paths are relative to `demos/views/`.

Current build version: v0.49.3
Last updated: 2026-11-07

---

## Org-navi-map

```
demos/views/
├── index.html           build output, classic deferred script, opens from file://
├── assets/              build output: views.js (one IIFE), style.css (fonts inlined), media/ (NASA photos, WebP)
├── README.md            how to open, what it shows, how to rebuild
├── CREDITS.md           every photograph and data source
├── HASHOUT_LOG.md       this file
├── vite.config.ts       root and outDir inside this folder, emptyOutDir false
└── src/
    ├── index.html       shell markup in the app's class names, theme boot before first paint
    ├── main.ts, main.css  entry: mounts the real mountDatabase, sidebar of three databases, CSV open, reset, theme
    ├── engine.ts        the stub Engine the real client code needs, wired to the in-browser backend
    ├── server.ts        the in-browser backend behind a fake fetch
    ├── formula.ts       the formula language
    ├── data.ts, seed.ts the three collections, their properties, views and rows
    └── pageicon-shim.ts file URL shim for row photos
```

## Why this shape

- Demo 1 paints the app's table and board from a small re-implementation. This demo mounts the app's own `mountDatabase` and every real view renderer from `web/src/extensions/database`, so the toolbar, panels, cells, board, calendar, timeline, gallery, list, chart and heatmap are the shipped code, not a lookalike.
- The real client talks to the Go server over `fetch`. `server.ts` answers the same routes in the browser: collection load, query with filter, sort, group, search, range and calculations, row create, update and delete, schema operations, views, CSV export, relation search. It ports the server's query, group, filter, sort, calculation, rollup and chart semantics, and `formula.ts` ports the formula language, so formulas and rollups recompute after every edit. Pairs of relations stay in step (editing Program on a mission edits Missions on the program).
- Photos are displayed only, never read back, so they are plain files. Nothing the demo decodes lives outside the JS bundle.
- Three linked collections tell one story (US crewed spaceflight from Freedom 7 to Apollo 17) because relations, rollups and formulas only show their worth across tables: Astronauts roll up their missions, Programs roll up theirs.
- `main.ts` fixes the page clock to 20 July 1969 so the calendar and timeline open on the Apollo 11 month and "today" markers sit on real data. `Date` is subclassed once; only no-argument construction is shifted.

## Persisting a view change before the query

The client's `updateView` applies a change to its local copy at once and sends the PATCH after a debounce. The table then re-queries immediately by `view_id`, and a backend that only reads its stored view answers with the old filter, sort or group. `engine.ts` wraps `updateView` so the backend's stored view takes the new config synchronously before the client sends anything. The real server avoids the race by reading the view as saved; the demo has no such round trip.

## Operations that work

View switching across eight views of Crewed missions; filter rules; sort; group by any property; search; add and delete a row; edit a cell in place (title, number, select, date, relation); board drag between lanes; calendar paging by month; timeline; formulas (Days, Length, Experience, Summary) and rollups (Missions, Hours flown, First launch) recomputing; add a view from the plus tab; export the current view as CSV; open your own `.csv` or `.tsv` as a new database (typed by inference, table, board and chart views); reset data.

## Not in the demo

Forms, dashboards, the Map and Graph view types need a map tile source or a larger engine; they are listed in the add-view menu because the real menu lists them, and are not seeded. Automations, buttons and the AI autofill property need the server.

## Zero flash

The shell and the three tree rows are in the HTML. The theme is set before first paint from localStorage. The cover has fixed width, height and aspect. The database host has a fixed minimum height so the page does not jump when the first query answers. Views swap their stage in place; the toolbar and tabs are built once per database.

## Rebuild

From `demos/views`: `node ../../web/node_modules/vite/bin/vite.js build --config ./vite.config.ts`. The config imports `web/` by relative path and never writes outside `demos/views`.
