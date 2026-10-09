# Offline

form.md opens without a connection to the server. A service worker keeps the built client and a copy of the signed-in app shell, and page bodies live in the browser's IndexedDB as the same CRDT documents the server holds, so pages saved on the device open, show their text and accept typing while the server is unreachable. Text edits merge into the server's copy when the connection comes back. Everything else (databases, search, comments, the inbox, uploads, AI and settings) needs the server.

## Layout

| Path | What lives there |
|---|---|
| `web/public/sw.js` | The service worker: asset cache, shell cache, offline navigation, push notifications |
| `web/src/ui/offline.ts` | Registers the worker, asks it to refresh the shell and cache assets, the "Available offline" action |
| `web/src/ui/offline-list.ts` | The per-account list of pages kept for offline use (in `localStorage`) |
| `web/src/engine/doc/docs.ts` | Document sessions persisted to IndexedDB (`y-indexeddb`); `clearLocal` on sign-out |
| `web/src/ui/status.ts` | The connection notice |
| `internal/static/offline.go` | `GET /offline.json`: the list of built assets |
| `internal/server/server.go` | `/offline-shell`: the signed-in app document without a page |

## How it starts offline

1. About 2.5 seconds after the app starts, it registers `/sw.js` with scope `/` (in browsers that support service workers).
2. On install, the worker fetches `/offline.json` and caches the `core` assets (those the app document references or preloads) in the cache `formmd-assets`.
3. About 20 seconds after start, when the browser is idle and not in data-saver mode, it caches every asset listed in `/offline.json` and drops cached assets the current build no longer lists.
4. While online and signed in, the app asks the worker to fetch `/offline-shell` and keep it in the cache `formmd-shell`. The shell is the app document the server renders for you, with your account, the current workspace's page tree, tags, teamspaces and favorites. The app refreshes it after start and again 4 seconds after the page tree changes.
5. Requests under `/assets/` are answered from the cache first. Navigations to `/`, `/p/<id>`, `/library`, `/trash` and `/inbox` go to the network first; when the network fails, the worker answers with the cached shell, marked with the right route so the layout does not move on first paint.

## Keeping pages on the device

Every page you open is kept in IndexedDB on that device. To keep pages you have not opened, use the page menu item **Available offline** (shown for document pages you can view, in browsers with service workers). It keeps that page and all of its subpages synced into IndexedDB now, shows how many were saved, and re-syncs them about 30 seconds after each start while online. Choosing it again stops keeping them.

## What works and what does not

| Works offline | Needs the server |
|---|---|
| Opening the app at `/`, a page, the library, the trash or the inbox route | Databases: rows, views, filters, charts and forms are answered by the server |
| The sidebar (page tree, favorites, tags) as of the last shell refresh | Search and the command palette's answers |
| Reading pages saved on this device | Comments, the inbox contents, sharing, history |
| Typing in those pages; edits are kept in IndexedDB and sync when the server is back | Uploads, files and pictures not already cached |
| | AI, agents, settings and anything else under `/api` |

A page that was neither opened nor kept on this device shows its title but no body. While the server is unreachable the app shows: "Cannot reach the server. Pages saved on this device stay open, and edits to their text sync when it is back." with a Retry button, and "Connection restored." when it is back.

## Sign-out

Signing out deletes the cached shell and every IndexedDB document database the app created (names starting `formmd-doc-`), so the next person on that browser starts clean. The asset cache holds only the public client build and is kept.

## For operators

- Browsers only run service workers in a secure context: a page served over HTTPS, or from `http://localhost` or `http://127.0.0.1`. An instance reached over plain HTTP by any other name gets no offline start; see [[TLS-and-ACME]]. (This is a browser rule, not a form.md setting.)
- Nothing to configure on the server: `/sw.js`, `/offline.json` and `/offline-shell` are served by the binary. `/sw.js` gets its own Content-Security-Policy (`default-src 'none'; connect-src 'self'`).
- Kept pages are plain copies in each browser's storage and are not covered by [[Encryption]]. Signing out removes them.
- After an upgrade, the new build's `/offline.json` lists new asset names; the worker caches them and prunes the old ones on the next idle pass.

## The contract

1. A new top-level app route that should open offline is added to `APP_ROUTES` in `web/public/sw.js` and handled for `/offline-shell` in `internal/server/server.go`.
2. The worker caches only same-origin `GET` requests under `/assets/` and the shell; API responses are never cached, so offline never shows stale data from `/api`.
3. Anything the app stores per account on the device is removed by `clearLocal` on sign-out.

## Related

[[API-and-agents]] · [[Operations]]
