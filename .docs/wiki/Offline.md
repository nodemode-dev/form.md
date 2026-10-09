# Offline

form.md opens without a connection to the server. A service worker keeps the built client and a copy of the signed-in app shell, and page bodies live in the browser's IndexedDB as the same CRDT documents the server holds, so pages saved on the device open, show their text and accept typing while the server is unreachable, including after the browser was closed and reopened offline (the device remembers whether you may edit each page). Text edits merge into the server's copy when the connection comes back. Databases opened or kept on the device open read-only with their first window of rows when their first view is a table or a list. Everything else (database edits, filters and further rows, search, comments, the inbox, uploads, AI and settings) needs the server.

## Layout

| Path | What lives there |
|---|---|
| `web/public/sw.js` | The service worker: asset cache, shell cache, offline navigation, push notifications |
| `web/public/manifest.webmanifest` | The web app manifest that makes form.md installable |
| `web/src/ui/pull.ts` | Pull to refresh in the installed app on touch devices |
| `web/src/ui/offline.ts` | Registers the worker, asks it to refresh the shell and cache assets, the "Available offline" action |
| `web/src/ui/offline-list.ts` | The per-account list of pages kept for offline use (in `localStorage`) |
| `web/src/engine/doc/docs.ts` | Document sessions persisted to IndexedDB (`y-indexeddb`); `clearLocal` on sign-out |
| `web/src/ui/status.ts` | The connection notice |
| `internal/static/offline.go` | `GET /offline.json`: the list of built assets |
| `internal/server/server.go` | `/offline-shell`: the signed-in app document without a page |

## How it starts offline

1. About 2.5 seconds after the app starts, it registers `/sw.js` with scope `/` (in browsers that support service workers). The worker does not take over the page that registered it; it controls pages from their next load.
2. On install, the worker fetches `/offline.json` and caches the `core` assets (those the app document references or preloads) in the cache `formmd-assets`.
3. About 20 seconds after start, when the browser is idle and not in data-saver mode, it caches every asset listed in `/offline.json` and drops cached assets the current build no longer lists.
4. While online and signed in, the app asks the worker to fetch `/offline-shell` and keep it in the cache `formmd-shell`. The shell is the app document the server renders for you, with your account, the current workspace's page tree, tags, teamspaces and favorites. The app refreshes it after start and again 4 seconds after the page tree changes.
5. Files and pictures under `/files/<id>` that this browser has loaded whole are kept in the cache `formmd-files` and answered from it when the network fails.
6. Requests under `/assets/` are answered from the cache first. Navigations to `/`, `/p/<id>`, `/library`, `/trash` and `/inbox` go to the network first; when the network fails, the worker answers with the cached shell, marked with the right route so the layout does not move on first paint.

## Keeping pages on the device

Every page you open is kept in IndexedDB on that device, and every database you open while the service worker controls the tab is kept in the worker's API cache (`formmd-api`: the reads `GET /api/collections/<id>` and `GET /api/rows/<id>`). To keep pages you have not opened, use the page menu item **Available offline** (shown for document and database pages you can view, not for database rows, in browsers with service workers). It keeps that page and all of its subpages synced into IndexedDB, and the first view of each database among them in the API cache, now, shows how many were saved, and re-syncs them about 30 seconds after each start while online. Choosing it again stops keeping them.

## What works and what does not

| Works offline | Needs the server |
|---|---|
| Opening the app at `/`, a page, the library, the trash or the inbox route | Database edits, other views, filters, more rows, charts and forms |
| Kept or opened databases, read-only, with the first window of rows of the view they opened with, when that view is a table or a list | Databases whose first view is a board, calendar, gallery, timeline, chart or form show their title and no rows |
| The sidebar (page tree, favorites, tags) of the workspace that was current at the last shell refresh | Search and the command palette's answers; the trees of other workspaces |
| Reading pages saved on this device | Comments, the inbox contents, sharing, history |
| Typing in those pages; edits are kept in IndexedDB and sync when the server is back | Uploads, and files or pictures this browser has not loaded before |
| Files and pictures this browser has loaded before | |
| | AI, agents, settings and anything else under `/api` |

A page that was neither opened nor kept on this device shows its title but no body. While the server is unreachable the app shows: "Cannot reach the server. Pages saved on this device stay open, and edits to their text sync when it is back." with a Retry button, and "Connection restored." when it is back.

## The installed app

`/manifest.webmanifest` (linked from the app document) names the app `form.md`, starts it at `/` and opens it in its own window (`"display": "standalone"`) with the form.md icon. Browsers that support installing web apps offer to install it; the installed app uses the same service worker and the same offline behaviour as a tab.

In the installed window, a **Sync now** button appears in the top bar. It reconnects the live connections, checks the server, and reloads the page tree of the current workspace when the server answers. On touch devices, pulling down at the top of the page in the installed window does the same. In a browser tab the button is hidden; the same action is registered as the command "Sync now".

## Sign-out

Signing out deletes the cached shell, the API cache, the file cache and every IndexedDB document database the app created (names starting `formmd-doc-`), so the next person on that browser starts clean. When the server answers that the session has ended (expired or revoked), the app deletes the shell, API and file caches before it goes to the sign-in page. The asset cache holds only the public client build and is kept.

## For operators

- Browsers only run service workers in a secure context: a page served over HTTPS, or from `http://localhost` or `http://127.0.0.1`. An instance reached over plain HTTP by any other name gets no offline start; see [[TLS-and-ACME]]. (This is a browser rule, not a form.md setting.)
- Nothing to configure on the server: `/sw.js`, `/manifest.webmanifest`, `/offline.json` and `/offline-shell` are served by the binary. `/sw.js` gets its own Content-Security-Policy (`default-src 'none'; connect-src 'self'`).
- Kept pages are plain copies in each browser's storage and are not covered by [[Encryption]]. Signing out removes them.
- After an upgrade, the new build's `/offline.json` lists new asset names; the worker caches them and prunes the old ones on the next idle pass.

## The contract

1. A new top-level app route that should open offline is added to `APP_ROUTES` in `web/public/sw.js` and handled for `/offline-shell` in `internal/server/server.go`.
2. The worker caches only same-origin `GET` requests: assets, the shell, the two database reads matched by `API_READS` and whole (non-range) `200` file responses. API and file reads go to the network first, so a cached copy is only ever shown when the server cannot be reached. A new cached read is added to the matching pattern in `web/public/sw.js` and to the cache names `clearLocal` and the session guard in `web/src/main.ts` delete.
3. Anything the app stores per account on the device is removed by `clearLocal` on sign-out.

## Related

[[API-and-agents]] · [[Operations]]
