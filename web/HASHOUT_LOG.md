# HASHOUT_LOG · frontend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the web client in `web/`. Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to `web/`.

Current build version: v0.52.1
Last updated: 2026-11-09 · v0.52.1 a cached page is ready only once its write flag is known, so the mode button and the editor open as Editing (#doc-ready-write); 2026-11-09 · v0.52.0 header alignment in the page style menu and the workspace default, the header spanning a database page's views; the Edited label painted by the top bar in the start bundle, so the mode button no longer moves when the lazy shell-extras chunk arrives late; 2026-11-09 · v0.51.0 the three demos open inside the window from Home, the Help menu and Find a command, in a sandboxed frame (#src-ui-demos-ts), 29.json in 25 languages, the workspace cookie write tolerated where cookies are blocked; 2026-11-03 · v0.47.8 the About tab names NodeAgentc, Inc. and links the licence; 2026-10-22 · v0.46.1: version only (releases through formcompiler); v0.46.0: Home's Templates side panel, dropdowns without scrolling, hidden scrollbars; v0.45.3: the approved icon's white .MD tag kept, approved1024.png; v0.45.2: the icon's edge without the light rim, make_icons.py; v0.45.1: the owner's app icon in favicon.svg; v0.45.0: Phase 19: Home's Data & charts and Example projects, the building state, project counts, chart number formats, KPI and gauge fitting, radial centre values, graph colours, palette order, Mermaid colours, 28.json in 25 languages; v0.44.0: Phase 18 interface: page-first shell and Home at /home, template gallery, search filters, Find a command, find and replace, rebindable keys, Preferences with theme profiles, 27.json in 25 languages; v0.41.0: per-card meters, the library Graph tab, Include the web, dark Mermaid, synced-block chrome, long-press tree drag; v0.26.0: offline row creates; v0.25.0: offline row deletes, saved styles; v0.24.0: list loading on scroll, marks in copied Markdown; v0.23.0: picking single marks, timeline windowing; v0.22.0: offline row outbox, Writing settings; v0.21.0: phone block toolbar, block presence, font rename and remove; v0.20.0: alignment on public pages, align.css; v0.19.0: offline by default, phone width, command metadata; v0.18.0: the board block; v0.17.0: custom fonts, typewriter scrolling; v0.16.0: image annotations, block alignment; v0.15.0: page styles, focus mode, word counts; v0.14.1: the vector canvas engine removed; Phase 14: the terminal and Claude Code dock, the command catalogue, 26 languages

---

## Org-navi-map

```
web/
├── HASHOUT_LOG.md                   this file
├── index.html                       shell skeleton, pre-paint theme boot, icon placeholders, preload slot
├── sandbox.html                     second entry: the Mermaid renderer, served sandboxed
├── package.json                     scripts: dev, build, preview, typecheck; mermaid's katex overridden
├── tsconfig.json                    browser code (src/)
├── tsconfig.node.json               vite.config.ts only
├── vite.config.ts                   port 5750, host false, proxy to :8750, icon inlining, dist placeholder
├── oc_install, oc_install.bat       npm ci (or npm install without a lockfile)
├── boot_frontend, boot_frontend.bat npm run dev
├── public/
│   ├── favicon.svg                  brand mark on dark tile
│   └── sw.js                        service worker: push, asset cache, offline shell for app navigations
├── tests/window/test_window.py      F7 window smoke (pytest and Python Playwright), run in CI, never installed in the app
├── dist/                            build output, embedded into the Go binary (only .gitkeep is tracked)
└── src/
    ├── main.ts                      read boot and auth, set language, create engine, mount, start
    ├── fonts/                       Inter and JetBrains Mono variable woff2 (latin, latin-ext) plus OFL texts
    ├── engine/                      headless: no DOM access
    │   ├── index.ts                 Engine, EngineState (auth, me, workspace, route, tabs, detail, live), createEngine, Platform contract
    │   ├── pages.ts                 PageStore: pages by parent, edits, moves, favorites, tags, store changes
    │   ├── live.ts                  Live: event stream client with watchdog, backoff and cursor
    │   ├── router.ts                routes, paths and the pure tab operations
    │   ├── order.ts                 fractional sort keys for optimistic placement
    │   ├── state.ts                 per-key observable state
    │   ├── commands.ts              command registry: metadata, typed params, invoke, catalogue for agents
    │   ├── keymap.ts                scoped shortcut registry, chord parse, match, format
    │   ├── api.ts                   request() and getJSON with typed ApiError (code, message, data)
    │   ├── client.ts                typed API client: auth, me, workspaces, admin
    │   ├── types.ts                 wire types mirroring the Go JSON
    │   ├── i18n.ts                  t(), plural(), lazy catalogs, language resolution
    │   ├── format.ts                Intl date, time and relative formatting from account preferences
    │   ├── health.ts                server health polling
    │   ├── prefs.ts                 persisted preference keys and readers
    │   └── doc/
    │       ├── schema.ts            ProseMirror schema, block ids, embed and href rules (DOM matches the server)
    │       ├── sync.ts              DocSync: one socket for every open page, frames, watchdog, backoff
    │       └── docs.ts              DocSession (Y.Doc, IndexedDB, awareness, generation) and DocStore cache
    ├── extensions/
    │   ├── registry.ts              register, activateAll, disposeAll
    │   ├── harness/                 dock and Claude Code commands and keys, catalogue mirroring to the server
    │   ├── writing/                 focus mode, word and reading-time counts, the slash-at-start option
    │   ├── shell/
    │   │   └── index.ts             shell commands and their shortcuts
    │   ├── account/
    │   │   └── index.ts             account, workspace and admin commands, Ctrl or Cmd plus comma
    │   ├── pages/
    │   │   └── index.ts             page, library, notes and tab commands and their shortcuts, search.open
    │   ├── database/                databases: views, row pages, forms, CSV (loaded with the first database)
    │   └── blocks/
    │       ├── index.ts             editor commands and shortcuts (editor scope), hover prefetch, mountEditor
    │       ├── editor.ts            EditorHost: binds sessions to one view, loading, paste, drop, clicks
    │       ├── plugins.ts           block ids, placeholders, code highlight, upload widgets, keys, structure
    │       ├── rules.ts             Markdown input rules
    │       ├── triggers.ts          slash, @ and [[ menus
    │       ├── catalog.ts           the block catalogue for the slash menu and turn into
    │       ├── commands.ts          turn into, insert, move, duplicate, delete, marks, checks
    │       ├── nodeviews.ts         code, task, toggle, callout, mention, toc, file, media, link card, embed, table
    │       ├── handle.ts            block handle: add, drag, block menu, caption
    │       ├── formatbar.ts         selection toolbar
    │       ├── media.ts             uploads with progress and image sizes
    │       ├── markdown.ts          Markdown writer and paste parser (markdown-it loaded on demand)
    │       ├── popover.ts           caret-anchored list and prompt popovers
    │       ├── context.ts           EditorCtx, caret and block rectangles
    │       ├── languages.ts         code languages and aliases
    │       ├── hl.ts                highlight.js core with 29 languages (its own chunk)
    │       ├── lightbox.ts          full-size image viewer
    │       ├── math.ts, mathml.ts   equation node view and panel, MathML sanitiser
    │       ├── math-render.ts       KaTeX to MathML with mhchem (its own chunk)
    │       ├── columns.ts           column view with resize, side drop, column commands, normalising
    │       ├── links.ts             link previews, media imports, paste choice, embed choice
    │       ├── diagram.ts           bridge to the sandboxed Mermaid renderer
    │       ├── ai.ts                Ask AI panel, target decoration, Ctrl+J, AI block view, safe Markdown preview
    │       ├── meeting.ts           meeting notes block: recorder, segments, transcription, summary
    │       └── synced.ts            synced block view with a nested editor, unsync
    ├── sandbox/
    │   └── main.ts                  renderer page: mermaid in strict mode, answers over postMessage
    ├── ui/                          DOM layer: renders engine state, dispatches commands
    │   ├── app.ts                   mountUI, platform and storage adapters, command click delegation
    │   ├── auth.ts                  set-up, sign-in, sign-up and invitation screens, workspace cookie
    │   ├── session.ts               workspace switcher and account menu, live names and tiles
    │   ├── routes.ts                data-route, document title, scroll memory, home view
    │   ├── tree.ts                  sidebar: favorites, tags, page tree, notes list, filter, trash link, drag
    │   ├── topbar.ts                breadcrumb, page actions, tab strip, update notice
    │   ├── page-view.ts             page header: cover, icon, title, description, tags, pages inside
    │   ├── page-actions.ts          the shared page menu and its actions
    │   ├── page-style.ts            page style menu, commands and the html attributes they set
    │   ├── share-dialog.ts          people, groups, guests, links and the publish entry
    │   ├── teamspace-dialog.ts      teamspace create and settings
    │   ├── publish-dialog.ts        publish to the web: address, SEO, picture, theme, font, password, domain
    │   ├── history-dialog.ts        page history: versions, preview, block diff, save, restore
    │   ├── analytics-dialog.ts      page analytics: KPIs, chart, creator, editors, viewers
    │   ├── bar-chart.ts             inline SVG bar chart updated in place
    │   ├── comment-view.ts          thread, composer and suggestion views
    │   ├── comments-pane.ts         the comments pane with filters
    │   ├── agent-pane.ts            the agent pane: chat, tool steps, undo, history, settings (its own chunk)
    │   ├── agent-dock.ts            bottom panel: xterm terminals, Claude Code tabs, Claude Code chat (its own chunk)
    │   ├── workspace-agents.ts      workspace settings: custom agents, editor, runs
    │   ├── account-agents.ts        account settings: connect snippets, skill download, connected apps, runs
    │   ├── admin-ai.ts              instance settings: AI providers, allow-list, defaults, limit, usage
    │   ├── discussion.ts            page-level discussion under the title
    │   ├── mode-switch.ts           Editing, Suggesting and Viewing switch
    │   ├── inbox-view.ts            inbox route
    │   ├── push.ts                  web push subscription
    │   ├── workspace-sites.ts       workspace settings: published pages and the publishing switch
    │   ├── workspace-analytics.ts   workspace settings: activity, most viewed pages, view history switch
    │   ├── workspace-webhooks.ts    workspace settings: webhooks and the shared destination list
    │   ├── workspace-audit.ts       workspace settings: audit log and SIEM streams
    │   ├── workspace-identity.ts    workspace settings: SCIM tokens
    │   ├── pickers.ts               icon picker, cover picker, picture upload
    │   ├── library-view.ts          library route
    │   ├── trash-view.ts            trash route
    │   ├── search.ts                search palette: recents, actions, filters, results (its own chunk)
    │   ├── glyphs.ts                the hand-drawn page icon set, tints, icon references
    │   ├── pageicon.ts              paint a glyph, a picture or the default in place
    │   ├── account.ts               account settings: profile, security, tokens, language and time
    │   ├── workspace-settings.ts    workspace settings and the new-workspace dialog
    │   ├── admin.ts                 instance settings: general, sign-in, email, users, workspaces, about
    │   ├── settings-entry.ts        the settings chunk: mounts the account, workspace and instance dialogs
    │   ├── dialog.ts                tabbed, simple, confirm and typed-confirm dialogs, overlay binding
    │   ├── menu.ts                  anchored menus and menu buttons
    │   ├── form.ts                  form controls, status line, busy state
    │   ├── list.ts                  keyed list reconcile
    │   ├── errors.ts                API error codes to sentences
    │   ├── avatar.ts                avatars, workspace tiles, palette, tile colour hash
    │   ├── dom.ts                   ref, setText, setAttr, h, delayedIndicator
    │   ├── icons.ts                 inline currentColor SVG icons (shared with vite.config.ts)
    │   ├── layers.ts                dismissable layer stack, inert, focus return
    │   ├── floating.ts              anchored placement with flip and clamp
    │   ├── tooltip.ts               single reusable tooltip with live chord hints
    │   ├── toast.ts                 toasts with tones, actions, dedupe by id
    │   ├── keyboard.ts              window key listener, active scopes, Escape binding
    │   ├── shortcuts-sheet.ts       modal listing the live shortcut registry
    │   ├── sidebar.ts               collapse, mobile drawer, topbar scrolled state
    │   ├── status.ts                server status card and connection toasts
    │   ├── theme.ts                 theme attributes and the segmented control
    │   └── theme/
    │       ├── tokens.css           every colour, size, space, radius and motion token, tint palette
    │       └── z-layers.css         the one stacking scale
    └── styles/
        ├── main.css                 import order
        ├── fonts.css                @font-face for the shipped fonts and their metric-matched fallbacks
        ├── base.css                 reset, body ground, kbd, focus ring, hidden, reduced motion
        ├── controls.css             icon button, text button, segmented control
        ├── forms.css                tints, avatars, tiles, buttons, fields, switches, rows, badges, callouts
        ├── shell.css                app grid, sidebar, workspace switch, user chip, sheet, topbar, drawer
        ├── doc.css                  document column and status card
        ├── overlays.css             tooltip, scrim, dialog, shortcut rows, toasts
        ├── panels.css               menus, dialog sizes, tabbed dialog layout
        ├── settings.css             settings dialog parts: swatches, option cards, users pane
        ├── covers.css               cover gradients, shared with the site stylesheet
        ├── comments.css             comments, threads, suggestions, inbox
        ├── pages.css                sidebar sections and rows, tabs, views, page header, pickers, library, trash
        ├── editor.css               document typography, blocks, handle, format bar, menus, cursors, uploads, lightbox
        └── auth.css                 sign-in card and the data-auth visibility rules
```

---

## Table of contents

- [index.html](#index-html)
- [vite.config.ts](#vite-config-ts)
- [src/engine/state.ts](#src-engine-state-ts)
- [src/engine/keymap.ts](#src-engine-keymap-ts)
- [src/engine/health.ts](#src-engine-health-ts)
- [src/engine/prefs.ts](#src-engine-prefs-ts)
- [src/extensions/shell/index.ts](#src-extensions-shell-index-ts)
- [src/ui/app.ts](#src-ui-app-ts)
- [src/ui/dom.ts](#src-ui-dom-ts)
- [src/ui/icons.ts](#src-ui-icons-ts)
- [src/ui/layers.ts](#src-ui-layers-ts)
- [src/ui/floating.ts](#src-ui-floating-ts)
- [src/ui/tooltip.ts](#src-ui-tooltip-ts)
- [src/ui/toast.ts](#src-ui-toast-ts)
- [src/ui/shortcuts-sheet.ts](#src-ui-shortcuts-sheet-ts)
- [src/ui/sidebar.ts](#src-ui-sidebar-ts)
- [src/ui/status.ts](#src-ui-status-ts)
- [src/ui/theme/z-layers.css](#src-ui-theme-z-layers-css)
- [src/styles/shell.css](#src-styles-shell-css)
- [src/styles/overlays.css](#src-styles-overlays-css)
- [Phase 2: document contract and start-up](#p2-start)
- [Phase 2: engine](#p2-engine)
- [Phase 2: sign-in and session](#p2-auth)
- [Phase 2: dialogs, menus and forms](#p2-kit)
- [Phase 2: settings dialogs](#p2-settings)
- [Phase 2: styles](#p2-styles)
- [v0.2.1: fonts, appearance and in-place panes](#v021)
- [Phase 3: engine](#p3-engine)
- [Phase 3: shell](#p3-shell)
- [Phase 4: documents and editor](#p4)
- [Phase 6: search and editor completion](#p6)
- [Phase 7: collaboration, sharing and identity](#p7)
- [Phase 8: agents and AI](#p8)
- [Phase 9: automations, charts, buttons, templates, views](#p9)
- [Phase 10: import, export and API](#p10)

---

## index.html {#index-html}

### index.html · pre-paint theme boot
- refers-to: index.html :: inline `<script>` in head
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

Runs before the stylesheet link so the first painted frame already has `data-theme`, `data-theme-pref` and `data-sidebar`. Dark is the default when nothing is stored. It must stay a bare `<script>` tag with no attributes and must stay the only inline script: the Go server pins it in the CSP by SHA-256 (backend log, `internal/static/static.go · buildCSP`). The storage keys `formmd.theme` and `formmd.sidebar` must match `src/engine/prefs.ts · PREF_KEYS`. Verified under 4x CPU and 500 kbps: every recorded frame from the first one carries the final theme, in dark and light.

### index.html · static shell skeleton
- refers-to: index.html :: `#app`, `#overlay-root`, `#toast-root`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

The whole shell is in the HTML so it paints before any JavaScript runs; `src/ui/*` binds to these nodes by `data-ref` and id and only patches text and attributes afterwards. Dynamic values (version, status fields) have reserved space in CSS. Overlay and toast roots are direct children of `body` per the stacking rules.

### index.html · first paint waits for the whole shell (FB-150, FB-151, FB-155, FB-156, FB-157)
- refers-to: index.html :: `<link rel="expect" href="#overlay-root" blocking="render">`
- kind: warning-context
- logged: v0.47.5 · 2026-10-26

Wrong: the shell is server-rendered HTML that the browser parses in pieces, and under CPU and network load a paint can land between two pieces. Anything laid out from its later siblings then moves when the rest arrives: the sidebar head actions and version (58 to 88 px wide, FB-150), the sidebar foot (101 to 163 px tall, FB-151), the owner-only Claude Code button and the topbar icons beside it (1404 to 1372 px, FB-155), the page actions (125 or 93 to 189 px wide, FB-156) and Home's drop icon (37 px down, FB-157). Each showed in about 1 load in 5 under 4x CPU and a slow network, found by the window smoke (F7). Changed: the `expect` link blocks rendering until `#overlay-root`, the last element of the shell, is parsed, so the first paint always has the whole shell. Browsers that do not know the link ignore it and behave as before. Reproduce each by sending the page cut before `sb-new`, `sb-settings`, `topbar__claude`, `act-favorite` or `home-drop__text`, waiting 1.2 s, then sending the rest: each moved (4.3e-5, 1.3e-3, 2.7e-5, 4.9e-4, 9.2e-5) before and none after. The marker must stay after everything the shell paints; an element added after `#overlay-root` is not covered.

### tests/window/test_window.py · the window smoke (F7)
- refers-to: tests/window/test_window.py :: `test_window_loads_throttled_without_shift_errors_or_refusals`
- kind: rationale
- logged: v0.47.5 · 2026-10-26

Starts the Go server built from this repo (`FORMMD_TEST_BIN`, default `bin/form-md`; `go build ./cmd/form-md` after `npm run build`, because `embed.go` embeds `web/dist`) on a free loopback port with a temporary data folder and HOME, `FORMMD_FAMILY=off` and a dead update URL. The server runs in desktop mode, which stops it when stdin closes, so the script keeps stdin open as a pipe like the desktop shell. The window is signed in the way the shell does it: `/desktop/enter?key=...`, then a doc is created over the API. Three documents are loaded in Chromium at 4x CPU and 150 ms, 1.6 Mbps: `/home` (waits for data-auth app and data-route home), `/p/<id>` (waits for an editable ProseMirror, so it cannot pass on a sign-in page) and `/sandbox.html` (the Mermaid renderer, a separate document). Each must have a title, no 4xx or 5xx, no failed request, no console or page error, and layout shift 0 after load, a bounded network idle wait (20 s, timeout ignored, the window keeps an events poll open) and a 3 s settle. A shift message names each node with its rect before and after.

One allowance: Chromium reports a fetch that completed with a 204 (the page-visit POST) as `requestfailed` with `net::ERR_ABORTED` after the response arrived; the page's own fetch resolves with 204 (checked in the page) and the server sent a plain 204. A failed request is therefore ignored only when the same request already answered 204. IGNORED_REQUESTS stays empty. Run locally: `FORMMD_TEST_BIN=... python -m pytest web/tests/window -q`.

### index.html · icon placeholders
- refers-to: index.html :: `<i data-icon="...">`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Replaced at dev-serve and build time by full inline SVG from `src/ui/icons.ts` (see `vite.config.ts · inlineIcons`), so icons are in the first paint and there is one icon source. An unknown name fails the build. The optional `class` attribute must come after `data-icon`.

---

## vite.config.ts {#vite-config-ts}

### vite.config.ts · server and preview
- refers-to: vite.config.ts :: `server`, `preview`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

`host: false` and `strictPort: true` are mandatory (dev-rules vite-server-guard): the dev server listens on 127.0.0.1:5750 only and fails loudly if the port is taken. Preview is 5751. The proxy forwards API, health, notices and future `/events`, `/mcp`, `/files`, `/sync` (WebSocket) to the Go server on 127.0.0.1:8750.

### vite.config.ts · keepDistPlaceholder
- refers-to: vite.config.ts :: `keepDistPlaceholder`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

`emptyOutDir` deletes `dist/.gitkeep`, which is tracked because the Go embed needs `web/dist` to contain a file before the client has ever been built. The plugin writes it back after every build so `git status` stays clean.

---

## src/engine/state.ts {#src-engine-state-ts}

### src/engine/state.ts · State
- refers-to: src/engine/state.ts :: `State.set`, `State.watch`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Per-key watchers instead of one change event: a subscriber hears only the key it renders, which is what lets the UI patch one node per change (DOM law). `set` ignores identical values (`Object.is`), so re-entrant writes from watchers settle immediately. `State` is invariant in its type parameter, so code that needs a slice takes callbacks (see `HealthSink`) rather than a narrower `State`.

---

## src/engine/keymap.ts {#src-engine-keymap-ts}

### src/engine/keymap.ts · matching rules
- refers-to: src/engine/keymap.ts :: `parseChord`, `Keymap.matches`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Chords are written like `mod+shift+l`. `mod` is Cmd on Apple platforms and Ctrl elsewhere; the other of the two is never bound and rejects a match. Letters and digits match the physical key (`KeyL`, `Digit1`) so keyboard layouts do not move shortcuts; named keys match `event.key`; symbols such as `?` and `[` match `event.key` with Shift ignored, because the symbol already implies it. IME composition is ignored.

### src/engine/keymap.ts · resolution order
- refers-to: src/engine/keymap.ts :: `Keymap.handle`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Candidates are filtered by active scope, by typing state (`whileTyping`) and by `when`, then ordered by scope rank (modal over editor over view over global) and most recent registration. The first command that runs consumes the key; a disabled command lets the next candidate try. `list` feeds the shortcut sheet and `chordFor` feeds tooltips, so both always show what is bound right now.

---

## src/engine/health.ts {#src-engine-health-ts}

### src/engine/health.ts · HealthMonitor
- refers-to: src/engine/health.ts :: `HealthMonitor.check`, `probe`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Polls `/healthz` every 30 s while healthy and every 5 s while down; concurrent `check` calls share one request. A 503 with a body is `degraded` (server up, storage failing); no body or a network error is `down`. The previous version is kept while down so the UI does not blank it. `checking` is published separately so the UI can show a delayed indicator.

---

## src/engine/prefs.ts {#src-engine-prefs-ts}

### src/engine/prefs.ts · PREF_KEYS
- refers-to: src/engine/prefs.ts :: `PREF_KEYS`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

These keys are also read by the inline boot script in `index.html`. Renaming one here without the other brings back the first-paint flash.

---

## src/extensions/shell/index.ts {#src-extensions-shell-index-ts}

### src/extensions/shell/index.ts · commands and bindings
- refers-to: src/extensions/shell/index.ts :: `setup`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Shell commands: toggle sidebar (`mod+\`, and `[` when not typing), shortcuts sheet (`?`, `mod+/`), switch light and dark (`mod+shift+l`), the three explicit theme choices, and a connection check. Toggle sidebar collapses the column on wide screens and opens the drawer on narrow ones; the engine asks the injected `Platform.isNarrow` rather than reading the viewport itself.

---

## src/ui/app.ts {#src-ui-app-ts}

### src/ui/app.ts · command click delegation
- refers-to: src/ui/app.ts :: `mountUI`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Any element with `data-command` runs that command on click, unless it sits inside an inert region. Buttons therefore need no individual listeners and their tooltip chord comes from the same id.

### src/ui/app.ts · storageAdapter
- refers-to: src/ui/app.ts :: `storageAdapter`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Every `localStorage` access is wrapped because it throws in some private modes and sandboxed frames; preferences then fall back to defaults instead of breaking startup.

---

## src/ui/dom.ts {#src-ui-dom-ts}

### src/ui/dom.ts · setText and setAttr
- refers-to: src/ui/dom.ts :: `setText`, `setAttr`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

Write only when the value differs, so repeated renders cause no DOM mutation, no style recalculation and no mutation records.

### src/ui/dom.ts · delayedIndicator
- refers-to: src/ui/dom.ts :: `delayedIndicator`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Shows a busy state only if work takes longer than 300 ms and then keeps it at least 250 ms, so fast operations never blink (dev-rules frontend reference).

---

### src/ui/dom.ts · revealApp
- refers-to: src/ui/dom.ts :: `revealApp`; src/main.ts :: `start`
- kind: explainer
- logged: v0.49.2 · 2026-11-07

In the desktop app the main window stays unseen behind the splash until the page says its first content is in place. `start` calls `revealApp` after `engine.start()` (and in the public-form branch); it waits three animation frames, so fonts and the shell settle, then invokes the Tauri command `app_ready` through `__TAURI_INTERNALS__`, once. Outside the desktop app the global does not exist and nothing happens, and no @tauri-apps/api dependency is added. The shell side is in src-tauri/HASHOUT_LOG.md, splash-icon.

## src/ui/icons.ts {#src-ui-icons-ts}

### src/ui/icons.ts · provenance and grid
- refers-to: src/ui/icons.ts :: `PATHS`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Every glyph is hand-drawn for form.md on a 24 grid with a 1.75 stroke, round caps and joins, `currentColor` only; no third-party icon set is used, so there is no attribution obligation. The `logo` mark is three filled blocks forming an F, the lower arm at reduced opacity. The module is imported by `vite.config.ts`, so it must not touch `document` at module load; `iconNode` does so lazily.

---

## src/ui/layers.ts {#src-ui-layers-ts}

### src/ui/layers.ts · stack behaviour
- refers-to: src/ui/layers.ts :: `openLayer`, `close`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

One stack for every dismissable surface. Escape closes the top layer (bindings in `ui/keyboard.ts`); a pointerdown outside the top layer closes it when `dismissOnOutside` is set. Modal layers make `#app` inert, which both traps focus and blocks pointer input behind them. On close, focus returns to `returnFocus` or to whatever was focused at open time. After an outside tap the browser moves focus to `body` on its own; that is expected and is not fought.

### src/ui/layers.ts · drawer is non-modal
- refers-to: src/ui/sidebar.ts :: drawer `openLayer` call
- kind: rationale
- logged: v0.1.0 · 2026-10-07

The mobile drawer lives inside `#app`, so making it modal would make the drawer itself inert. It is a non-modal layer instead, and `sidebar.ts` sets `#sheet` inert while it is open.

---

## src/ui/floating.ts {#src-ui-floating-ts}

### src/ui/floating.ts · computePlacement
- refers-to: src/ui/floating.ts :: `computePlacement`, `anchorTo`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Pure function: place on the requested side, flip to the opposite side only if that overflows less, then clamp along the cross axis inside an 8 px viewport padding. `anchorTo` re-places on scroll (capture, so nested scrollers count) and resize, batched to one animation frame. This is the single positioning primitive; menus, popovers and pickers in later phases use it rather than their own maths.

---

## src/ui/tooltip.ts {#src-ui-tooltip-ts}

### src/ui/tooltip.ts · one node, measured before visible
- refers-to: src/ui/tooltip.ts :: `mountTooltip`, `show`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

A single tooltip element is created once and reused; showing one patches its text and key caps in place. It is unhidden at opacity 0, measured and positioned, and only then faded in on the next frame, so it never flashes at a stale position. Hover shows after 450 ms, or immediately within 300 ms of another tooltip closing; keyboard focus shows it only for `:focus-visible`. Touch pointers never trigger it.

---

## src/ui/toast.ts {#src-ui-toast-ts}

### src/ui/toast.ts · lifecycle
- refers-to: src/ui/toast.ts :: `toast`, `leave`, `arm`, `pause`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Tones: info, success, warning, danger (danger uses `role="alert"`). A toast with the same `id` as a live one updates it in place and resets its timer instead of stacking a duplicate. `durationMs: 0` keeps it until dismissed. Hover and focus pause the timer. Leaving collapses the row height (`grid-template-rows` 1fr to 0fr) so the stack closes smoothly; a 400 ms fallback removes the node if `transitionend` never fires.

---

## src/ui/shortcuts-sheet.ts {#src-ui-shortcuts-sheet-ts}

### src/ui/shortcuts-sheet.ts · fill while hidden
- refers-to: src/ui/shortcuts-sheet.ts :: `fill`, `open`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

The dialog is built once. Its rows are regenerated only while the dialog is hidden, just before it opens, because bindings can change between openings; content is never rebuilt while visible. It lists global and view scopes, i.e. what works on the screen underneath.

---

## src/ui/sidebar.ts {#src-ui-sidebar-ts}

### src/ui/sidebar.ts · NARROW_QUERY
- refers-to: src/ui/sidebar.ts :: `NARROW_QUERY`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

Must match the `max-width: 767px` media queries in `src/styles/shell.css`, `doc.css` and `overlays.css`; widening past the breakpoint closes an open drawer.

---

## src/ui/status.ts {#src-ui-status-ts}

### src/ui/status.ts · connection toasts
- refers-to: src/ui/status.ts :: `mountStatus`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

Only phase transitions raise a toast, all under one id so they replace each other: down shows a persistent danger toast with Retry, degraded a persistent warning, and recovery a short success toast. Visibility and the browser `online` event trigger an immediate check.

---

## src/ui/theme/z-layers.css {#src-ui-theme-z-layers-css}

### src/ui/theme/z-layers.css · the layer table
- refers-to: src/ui/theme/z-layers.css :: `:root`
- kind: explainer
- logged: v0.1.0 · 2026-10-07

| Token | Value | Used by |
| --- | --- | --- |
| `--z-base` | 0 | normal flow |
| `--z-raised` | 100 | reserved |
| `--z-sticky` | 200 | reserved |
| `--z-chrome` | 300 | reserved |
| `--z-dropdown` | 400 | reserved for menus |
| `--z-scrim` | 500 | dialog scrim, drawer scrim |
| `--z-overlay` | 600 | dialog, mobile drawer |
| `--z-popover` | 700 | reserved for popovers inside dialogs |
| `--z-toast` | 800 | `#toast-root` |
| `--z-tooltip` | 900 | tooltip |
| `--z-debug` | 1000 | never shipped enabled |

No component file writes a numeric `z-index`.

---

## src/styles/shell.css {#src-styles-shell-css}

### src/styles/shell.css · topbar columns
- refers-to: src/styles/shell.css :: `.topbar__menu`, `.crumbs`, `.topbar__actions`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

Each topbar child has an explicit `grid-column`. Without it, hiding the menu button on desktop let auto-placement pull the action cluster into the flexible middle column, next to the breadcrumb, instead of the trailing edge. Found by the screenshot review during throttled verification.

### src/styles/shell.css · topbar actions anchored at the trailing edge (FB-149)
- refers-to: src/styles/shell.css :: `.topbar__actions`, `.crumbs`; src/styles/pages.css :: `.topbar__page-actions`; index.html :: `act-presence`, `act-mode`, `act-edited`
- kind: warning-context
- logged: v0.47.5 · 2026-10-26

Wrong: on a page the action cluster was an auto-width grid column, so when "Edited ..." text, the mode button and the visibility and share buttons arrived after first paint (and the skeleton's first state differed from the route's), the cluster grew leftwards and its box moved: three to five layout shifts (about 0.0004 each, 0.0018 in total) at 4x CPU and a slow network, found by the new window smoke (F7). Why it matters: the zero-shift law counts any move of a painted box. Changed: `.topbar__actions` and `.topbar__page-actions` are zero-width flex boxes with `justify-content: flex-end`, so their items overflow towards the leading edge and the right-hand items never move; `.crumbs` takes the room back with `padding-inline-end` (25rem, 15rem under 768px); in `index.html` the late arrivals come first (presence, mode, then the edited text), so one arriving never pushes an item that is already painted. Still moving by design: the edited text changes width when its relative time changes (once a minute), which moves the items left of it; not reserved here. Verified: window smoke 3 of 3 pass, shift 0, from 0.0018 before.

### src/styles/shell.css · collapse animation
- refers-to: src/styles/shell.css :: `.app`, `.sidebar`, `.sidebar__inner`
- kind: rationale
- logged: v0.1.0 · 2026-10-07

The grid column animates while `.sidebar__inner` keeps a fixed width, so text is clipped rather than re-wrapped during the motion. Visibility switches off only after the transition, which also removes the hidden sidebar from the tab order.

---

## src/styles/overlays.css {#src-styles-overlays-css}

### src/styles/overlays.css · #overlay-root is not positioned
- refers-to: src/styles/overlays.css :: `#overlay-root`
- kind: warning-context
- logged: v0.1.0 · 2026-10-07

A fixed or z-indexed overlay root would be a stacking context of its own and trap every overlay at one level, which put tooltips under toasts in the first draft. It stays `position: static`; each child is `position: fixed` and takes its own layer from the scale. `#toast-root` is positioned because it lays out its own stack.

### src/styles/overlays.css · removed draft rule
- refers-to: src/styles/overlays.css :: top of file
- kind: removed-code
- logged: v0.1.0 · 2026-10-07

```css
#overlay-root,
#toast-root {
  position: fixed;
  inset: 0;
  pointer-events: none;
}

#overlay-root {
  z-index: var(--z-overlay);
}
```

Replaced before the first commit for the stacking reason above.

---

## Phase 2: document contract and start-up {#p2-start}

### index.html · server-rendered hooks
- refers-to: index.html :: `data-auth="pending"`, `<!--slot:...-->`, `#boot`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

`index.html` is a template the Go server fills per request (see `internal/server/server.go · renderDocument contract` in the backend log). `data-auth="pending"` must stay exactly that string on the html tag, slot comments must sit directly inside the element whose text they become, and the boot placeholder must stay `<script type="application/json" id="boot">null</script>`. The auth screens live in `#auth` next to `#app`; `auth.css` shows one or the other from `data-auth`, so the right screen is the first frame. Static English text in this file is translated server-side in Phase 10 (German), not by the client, to keep that guarantee.

### src/main.ts · start order
- refers-to: src/main.ts :: `start`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Read `data-auth` and the boot block, load the language catalog (a no-op for English), create the engine, register extensions, mount the always-present UI and the auth screens, and mount the session chrome and settings dialogs only when signed in. Health polling starts only in the app; the sign-in page makes no background requests.

### src/main.ts · devBoot
- refers-to: src/main.ts :: `devBoot`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Stands in for the server-rendered document when Vite serves the raw page on :5750, by asking `/api/auth/policy` and `/api/me` for the same state and setting `data-sso` and `data-signup` itself. Production documents always arrive with `data-auth` and the boot block filled in, so this path never runs there. In development the screen can therefore change after first paint; that is expected and is the only place it happens. (Moved here from a source comment.)

### src/main.ts · guardSession
- refers-to: src/main.ts :: `guardSession`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

Sends the person back to `/login?next=...` when their session ends while the app is open (expired, signed out elsewhere, revoked by a password change), instead of leaving every later request failing in place. It reacts only to the `unauthorized` code while signed in, never to sign-in errors such as `bad_credentials` or `totp_required`. (Moved here from a source comment.)

---

## Phase 2: engine {#p2-engine}

### src/engine/index.ts · pickWorkspace
- refers-to: src/engine/index.ts :: `pickWorkspace`, `createEngine`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

The initial workspace prefers the server's `boot.workspace_id` (which came from the `formmd_ws` cookie), then the stored id, then the first shared workspace, then the first. The fallback order must match `currentWorkspace` on the server so the client never repaints the header with a different workspace.

### src/engine/index.ts · openOverlay
- refers-to: src/engine/index.ts :: `openOverlay`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Sets `overlayTab` to empty and then to the requested tab before setting `overlay`. The empty step makes asking for the same tab twice still fire, so a command such as "Language and time" switches tabs even when the account dialog is already open on another one.

### src/engine/client.ts · typed client
- refers-to: src/engine/client.ts :: `createClient`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

One namespace per server area (auth, me, workspaces, admin), every path in one place, every response typed from `types.ts`. 204 responses resolve to undefined. Errors are `ApiError` with the server's code, message and data.

### src/engine/i18n.ts · catalogs
- refers-to: src/engine/i18n.ts :: `t`, `plural`, `setLang`, `resolveLang`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Source strings are English and double as keys; catalogs live in `src/i18n/<lang>/*.json` (several parts per language) and load lazily, so English costs nothing. Since v0.34.0 the `form-md-i18n-catalogs` plugin in `vite.config.ts` serves `virtual:i18n` (a map of language to one dynamic import) and `virtual:i18n/<lang>` (that language's files merged in name order, the same order the old glob merged them); before, `import.meta.glob` put a loader per file (about 550) into a preloaded chunk, 38 KB raw, and a German start fetched 21 chunks. Types for the virtual module are in `src/i18n.d.ts`. `plural` keys are `"one|other"` with the translation in the same form. Supported languages are English and German (since v0.11.0). The server embeds the same files to translate index.html, so a string shown before the client starts must be a catalog key too.

### src/main.ts · the library view loads on demand
- refers-to: src/main.ts (`loadLibrary`); src/ui/library-view.ts :: `mountLibraryView`
- kind: rationale
- logged: v0.34.0 · 2026-10-17

The library view is mounted from its own chunk: at once when the app starts on `/library`, on the first navigation there, or in idle time three seconds after start. `mountLibraryView` already draws the current route when mounted late. Throttled probe p26 found the library pane never shown empty on a cold `/library` load, a click right after start, or a click after idle; the pane only becomes visible with its content.

### src/ui/page-view.ts, src/extensions/blocks/nodeviews.ts, src/extensions/blocks/board.ts · pickers on demand
- refers-to: src/ui/pickers.ts :: `openIconPicker`, `openCoverPicker`, `pickFile`; src/main.ts (idle prefetch)
- kind: rationale
- logged: v0.35.0 · 2026-10-17

The page icon and cover pickers, the callout icon picker and the board's picture button import `pickers.ts` when clicked, so it left the up-front script (about 1.6 KB gzip). `main.ts` prefetches it in idle time three seconds after start, so the first click rarely waits; the board's file chooser opens after that import, which matters because a file input needs a recent user activation. Other importers (account, workspace settings, CSV import, the file property editor) were already in on-demand chunks and keep their static imports.

### src/engine/format.ts · canonical
- refers-to: src/engine/format.ts :: `canonical`, `locale`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

Turns a language tag into one `Intl` accepts. Browsers can report forms such as `en-US@posix` (the headless test browser does), and every `Intl` constructor throws on them, which broke the sessions, tokens and language tabs in the first browser run. The `@...` suffix is dropped and the tag canonicalised, falling back to the plain language. (Moved here from a source comment.)

---

## Phase 2: sign-in and session {#p2-auth}

### src/ui/auth.ts · next and open redirects
- refers-to: src/ui/auth.ts :: `nextPath`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

`?next=` is honoured only for same-origin paths: it must start with `/` and not with `//` or `/\`, which browsers treat as another host. Anything else goes to `/`.

### src/ui/auth.ts · rememberWorkspace
- refers-to: src/ui/auth.ts :: `rememberWorkspace`, `WORKSPACE_COOKIE`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

The current workspace is a cookie (`formmd_ws`, one year, SameSite=Lax), not only local storage, because the server reads it to render the right workspace name into the first frame. Accepting an invitation sets it to the invited workspace.

### src/ui/session.ts · painting
- refers-to: src/ui/session.ts :: `mountSession`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

The header tile and name and the footer avatar and name are server-rendered; this module only updates them through `setText`/`setAttr`, which skip unchanged values, so the first paint is never touched. Menus are built from the command registry at open time, so their entries follow permissions (`enabled`) and show live shortcut hints. The owner sees how many workspaces they are not in, linking to emergency access.

### src/extensions/account/index.ts · commands
- refers-to: src/extensions/account/index.ts :: `setup`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Account, workspace and admin actions are commands, so menus, shortcuts and the Phase 6 command palette share them. Ctrl or Cmd plus comma opens account settings and works while typing. Sign-out navigates through `Platform.navigate` to keep the engine free of DOM access.

---

## Phase 2: dialogs, menus and forms {#p2-kit}

### src/ui/dialog.ts · tabbedDialog and overlayDialog
- refers-to: src/ui/dialog.ts :: `tabbedDialog`, `overlayDialog`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

The dialog DOM is built on first open, and each tab renders on its first visit; later visits call the refresh function the tab returned. `overlayDialog` binds a dialog to `state.overlay` and `overlayTab`: the state opens and closes it, and closing it by Escape, the close button or an outside click resets the state. Tabs can be hidden per open through `visible`.

### src/ui/dialog.ts · confirmTyped
- refers-to: src/ui/dialog.ts :: `confirmTyped`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Irreversible deletions (a workspace, a stranded workspace, an account) ask for the exact name or email to be typed; the button stays disabled until it matches.

### src/ui/menu.ts · reopen guard
- refers-to: src/ui/menu.ts :: `menuButton`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Clicking the anchor of an open menu closes it on pointerdown (outside-dismiss) and would reopen it on click. The anchor records when its menu closed and ignores a click within 350 ms of that. The guard applies only when a close was recorded.

### src/ui/menu.ts · removed reopen guard
- refers-to: src/ui/menu.ts :: `menuButton`
- kind: removed-code
- logged: v0.2.0 · 2026-10-07

```ts
const closedAt = Number(anchor.dataset.menuClosedAt ?? 0);
if (anchor.getAttribute("aria-expanded") === "true" || performance.now() - closedAt < 350) return;
```

With no close recorded, `closedAt` was 0, and `performance.now()` counts from page load, so every click in the first 350 ms after a load was ignored. Found by the browser verification clicking the account menu right after a reload.

### src/ui/form.ts · status and busy
- refers-to: src/ui/form.ts :: `status`, `busy`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

The status line reserves one line of height (`.form-status`), so showing an error never shifts the layout. `busy` disables the button at once but shows the busy look only after 300 ms and keeps it at least 250 ms, so fast requests do not flicker. Errors that are not `ApiError` are written to the console before the generic sentence is shown, so a client bug cannot hide behind "Something went wrong".

### src/ui/avatar.ts · tileColor
- refers-to: src/ui/avatar.ts :: `tileColor`, `COLORS`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

Must stay identical to `TileColor` in `internal/server/server.go` (hash `h*31 + code point`, unsigned 32-bit, modulo the palette in this order), or a workspace tile changes colour after the first paint.

---

## Phase 2: settings dialogs {#p2-settings}

### src/ui/account.ts · QR code
- refers-to: src/ui/account.ts :: `twoFactorSection`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

The server's SVG is shown through an `<img>` with a data URL rather than inserted as markup, so nothing in it can run; the document CSP already allows `data:` images. The key is also shown in groups of four for typing by hand.

### src/ui/account.ts · language change
- refers-to: src/ui/account.ts :: `languageTab`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Every control saves on change. A language change shows a "Reload now" note instead of re-rendering the open app, because the server renders the page in the chosen language and a reload is the one clean way to switch every string.

### src/ui/workspace-settings.ts · boundTab
- refers-to: src/ui/workspace-settings.ts :: `boundTab`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

Rebuilds a tab whenever the dialog opens on a different workspace than the one it was built for, and otherwise runs the tab's own refresh. (Moved here from a source comment.)

### src/ui/admin.ts · SettingsStore and SettingsForm
- refers-to: src/ui/admin.ts :: `SettingsStore`, `SettingsForm`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

`SettingsStore` holds one snapshot of the instance settings, shared by every tab of the dialog so a save on one tab is seen by the others (the sign-in tab's redirect addresses follow the public address saved on the general tab). `SettingsForm` binds controls to setting keys and sends only the keys whose value changed. Secret fields send the stored sentinel back unless typed into or removed (see `settings.go` in the backend log). A refused value marks its field invalid and names it. (Moved here from source comments.)

---

## Phase 2: styles {#p2-styles}

### src/styles/base.css · hidden attribute
- refers-to: src/styles/base.css :: `[hidden]`
- kind: rationale
- logged: v0.2.0 · 2026-10-07

The hidden attribute always wins over a component's own display value. Without `!important`, any class that sets `display` (`.btn`, `.section`, `.callout`) beat the attribute at equal specificity, which left a "Sign out" button on the current device, an empty emergency section and a reload note visible. This is the one deliberate `!important` outside the reduced-motion block. (Moved here from a source comment.)

### src/styles/base.css, forms.css, panels.css, settings.css · removed hidden overrides
- refers-to: src/styles/forms.css :: `.field`; src/styles/panels.css :: `.dialog__panel`; src/styles/settings.css :: `.check-list`
- kind: removed-code
- logged: v0.2.0 · 2026-10-07

```css
.field[hidden] { display: none; }
.dialog__panel[hidden] { display: none; }
.check-list[hidden] { display: none; }
```

Per-component patches for the problem above, made redundant by the global rule.

### src/styles/forms.css · tint fallback
- refers-to: src/styles/forms.css :: `.avatar`, `.tile`, `[data-color]`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

The default tint is a `var()` fallback (`var(--tint, var(--text-muted))`), not a declaration on `.avatar, .tile`. A declaration there has the same specificity as `[data-color="..."]` and comes later, so it overrode every chosen colour and all avatars rendered grey.

### src/styles/forms.css · removed rules
- refers-to: src/styles/forms.css :: `.avatar, .tile`, `.qr`
- kind: removed-code
- logged: v0.2.0 · 2026-10-07

```css
  --tint: var(--text-muted);
```

```css
.qr svg {
  display: block;
  inline-size: 100%;
  block-size: 100%;
}
```

The first was the grey override described above. The second styled an inline SVG QR code; the code is now an image (see `account.ts · QR code`).

### src/styles/forms.css · field alignment
- refers-to: src/styles/forms.css :: `.field`
- kind: explainer
- logged: v0.2.0 · 2026-10-07

`align-content: start` keeps a field's control at its natural height when a neighbour in the same `.form-row` is taller because of a hint line; without it a select next to a hinted input stretched to a taller box.

### src/styles/overlays.css · clicks while closing
- refers-to: src/styles/overlays.css :: `.scrim`, `.dialog`
- kind: warning-context
- logged: v0.2.0 · 2026-10-07

Scrims and dialogs take pointer events only while `.is-visible`. During the 220 ms fade-out they used to swallow clicks, so clicking the account menu right after closing a dialog did nothing. Found by the browser verification.

---

## v0.2.1: fonts, appearance and in-place panes {#v021}

### src/styles/fonts.css · no flash, no reflow
- refers-to: src/styles/fonts.css :: every `@font-face`; index.html :: font preload; src/ui/theme/tokens.css :: `--font-sans`, `--font-mono`
- kind: rationale
- logged: v0.2.1 · 2026-10-07

Three parts work together. The Inter latin text face is preloaded from the document head, so on a normal connection it is ready before first paint. Every face uses `font-display: optional`: if a face is not ready within the browser's short block period it is not used for that page at all, so text is never re-laid out when it arrives later (it is cached for the next load). The fallbacks "Inter Fallback" and "JetBrains Mono Fallback" are local Arial or Liberation Sans and Courier New or Liberation Mono with size, ascent, descent and line-gap overrides computed from the fonts' metrics (capsize's method, run once in scratch), so even a fallback page has the same line boxes. Verified: title and user-name widths constant across every frame of a throttled cold load and a warm load, CLS 0.

### src/styles/fonts.css · subsets and weights
- refers-to: src/styles/fonts.css :: `unicode-range`, `font-weight`
- kind: explainer
- logged: v0.2.1 · 2026-10-07

Latin covers English and German (including umlauts and sharp s); latin-ext is fetched only when a character in its range is on the page. Inter ships upright and italic (italic for the editor in Phase 4), JetBrains Mono upright only; a mono italic is synthesised. Variable weights: Inter 100 to 900, JetBrains Mono 100 to 800. Glyphs outside the subsets (for example the arrows and Mac modifier symbols in shortcut hints) come from the system stack in the token.

### src/ui/theme/tokens.css · data-font
- refers-to: src/ui/theme/tokens.css :: `--font-system-sans`, `--font-system-mono`, `:root[data-font="system"]`
- kind: explainer
- logged: v0.2.1 · 2026-10-07

The system stacks are tokens of their own; the shipped fonts sit in front of them in `--font-sans` and `--font-mono`. `data-font="system"` on the html tag points both at the system stacks. The server writes the attribute for people who chose it; `session.ts` keeps it in step when the choice changes.

### src/ui/account.ts · Appearance
- refers-to: src/ui/account.ts :: `profileTab` (Appearance section)
- kind: explainer
- logged: v0.2.1 · 2026-10-07

The theme choice mirrors `state.themePref` both ways, so it and the sidebar's segmented control always agree; it stays per device. The system font switch is an account preference saved at once and undone if the save fails.

### src/ui/account.ts · token workspace picker
- refers-to: src/ui/account.ts :: `tokensTab` (`fillPicks`)
- kind: explainer
- logged: v0.2.1 · 2026-10-07

The checkboxes are a keyed list now: a refresh adds or removes only the workspaces that changed and keeps the ticks and focus of the rest.

### src/ui/account.ts · removed picker rebuild
- refers-to: src/ui/account.ts :: `fillPicks`
- kind: removed-code
- logged: v0.2.1 · 2026-10-07

```ts
const chosen = new Set([...picks.querySelectorAll<HTMLInputElement>("input:checked")].map((i) => i.value));
picks.replaceChildren(
  ...me(engine)
    .workspaces.filter((w) => !w.break_glass)
    .map((w) => {
      const box = h("input", { attrs: { type: "checkbox", value: w.id } });
      box.checked = chosen.has(w.id);
      return h("label", { class: "check" }, box, h("span", { text: w.name }));
    }),
);
```

Rebuilt the visible list on every refresh of the tab.

### src/ui/admin.ts · users detail
- refers-to: src/ui/admin.ts :: `usersTab` (`paintDetail`, `buildDetail`, `accessSection`)
- kind: rationale
- logged: v0.2.1 · 2026-10-07

The detail pane is built once per selected person (keyed by id, whether it is you, whether it is the owner, whether you are the owner, and whether you may manage them) and patched on every later refresh through the collected patch functions. Handlers read the latest copy of the person, not the one the pane was built with. Selecting someone else swaps the pane in one operation, built off the page first. The previous version rebuilt the whole pane after every role change, deactivation or two-factor reset, which lost focus and redrew everything; it was replaced before v0.2.1 and is in git history at v0.2.0 (`paintDetail` in `web/src/ui/admin.ts`).

### src/ui/admin.ts · paintBadges and the overview row
- refers-to: src/ui/admin.ts :: `paintBadges`, `workspacesTab` (`overviewRow`, `updateOverview`)
- kind: explainer
- logged: v0.2.1 · 2026-10-07

Badge rows are replaced only when their signature (role, two-factor, deactivated) changes. Overview rows create their two badges and the emergency button once and only toggle `hidden`; the button looks up the latest copy of its workspace in `latest`.

### src/ui/workspace-settings.ts · proposal callout
- refers-to: src/ui/workspace-settings.ts :: `agentsTab` (`proposed`, `proposalBy`, `proposalText`)
- kind: explainer
- logged: v0.2.1 · 2026-10-07

Built once; a reload of the rules only sets its texts and visibility. "Load into editor" uses the latest proposal text.

### deliberate content swaps
- refers-to: src/ui/account.ts :: `twoFactorSection`; src/ui/account.ts :: token reveal; src/ui/workspace-settings.ts :: invitation link; src/ui/workspace-settings.ts :: `boundTab`
- kind: warning-context
- logged: v0.2.1 · 2026-10-07

These are the remaining places that replace a container's children, all on purpose: the two-factor steps (off, enrol, recovery codes, on) after a click, the one-time token and invitation link after a click, and a settings tab built for a different workspace while its dialog is opening (before it paints). Each builds the new content off the page and swaps it in one operation. Do not add a refresh path that calls them for unchanged content.

---

## Phase 3: engine {#p3-engine}

### src/engine/pages.ts · PageStore
- refers-to: src/engine/pages.ts :: `PageStore`, `StoreChange`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

One map of pages by id plus a set of child ids per `workspace:parent` slot; sorted child lists are cached per slot and dropped when a page enters or leaves it. Every change is one `StoreChange` (`upsert` with the previous version, `remove`, `loaded`, `favorites`, `tags`, `trash`), so views patch exactly the rows involved. A listener that throws is logged and does not stop the others or the write in progress. Pages from workspaces that are not loaded (favorites, tabs) live in the same map; `isLoaded` says whether a workspace's whole tree is there.

### src/engine/pages.ts · edit and flush
- refers-to: src/engine/pages.ts :: `edit`, `flush`, `withEdits`
- kind: warning-context
- logged: v0.3.0 · 2026-10-07

Field edits apply locally at once and go to the server 350 ms after the last keystroke (immediately for icon, cover, tags and visibility). While a request is in flight, its values and any newer local values overlay whatever the server or the event stream sends, so an echo of an older title can never jump the caret back. On failure the edit is dropped, an error is reported and the page is fetched fresh. `flushAll` runs on page hide.

### src/engine/pages.ts · optimistic moves
- refers-to: src/engine/pages.ts :: `move`, `keyFor`, `wouldCycle`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Within a workspace the page is placed at once with a client-side key between its new neighbours; the server's answer carries the real key, which sorts the same way, so nothing moves twice. A failed move puts the page back. Moves to another workspace are not optimistic: the page leaves when the server says so.

### src/engine/live.ts · stream client
- refers-to: src/engine/live.ts :: `Live`
- kind: rationale
- logged: v0.3.0 · 2026-10-07

Uses `EventSource` but does its own reconnecting, because the built-in retry has no backoff and no watchdog. Any message (including the 20 second ping) re-arms a 45 second watchdog; silence reconnects. Errors back off 1, 2, 5, 10, 20 seconds. The cursor is the last event id seen; it is replaced by the hello's sequence only on reset or first contact, so replayed events still advance it.

### src/engine/index.ts · start, resync and queueing
- refers-to: src/engine/index.ts :: `start`, `resync`, `dispatch`, `onEvent`
- kind: warning-context
- logged: v0.3.0 · 2026-10-07

With boot data the store is seeded synchronously and the stream starts at the boot cursor. Without it (the Vite dev server) the stream starts first and the first hello triggers a load. A hello with `reset` refetches every loaded workspace, favorites, the account and the open page; events that arrive meanwhile are queued and applied in order afterwards, so the snapshot never overwrites something newer. A hello with a different server version than the first one sets `newVersion`. Workspace `members`, `settings`, `rules` and `deleted` notices and user `workspaces` notices refresh the account.

### src/engine/router.ts and index.ts · tabs and history
- refers-to: src/engine/router.ts :: `openTab`, `openNewTab`, `closeTab`, `cycleTab`; src/engine/index.ts :: `commit`, `restoreNav`, `landing`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Opening a page reuses its tab, else replaces the active tab, else appends; opening in a new tab inserts after the active one. Every history entry stores the tab list and route, so back and forward restore the exact tab set. Tabs persist in `formmd.tabs`. Closing the active tab activates the neighbour that slides in, else the previous one, else home. Home is a waypoint: once the workspace's tree is loaded it opens the last page seen in that workspace (`formmd.last.<workspace>`), a tab from it, or its first page. The current workspace follows the open page.

---

## Phase 3: shell {#p3-shell}

### src/ui/tree.ts · keyed rendering
- refers-to: src/ui/tree.ts :: `renderList`, `syncChildren`, `patchRow`, `renderParent`, `onChange`
- kind: warning-context
- logged: v0.3.0 · 2026-10-07

Each page has one `li` for its lifetime (in `nodes`), with the row first and the child group second. A store change re-renders only the parents involved; a content change patches one row. The insertion cursor is recomputed from the previous sibling on every step, and leftovers are removed only if they are still in the list being rendered, because rendering a child group can pull a node out of the list the outer loop is walking (a page dropped inside its own former neighbour). Getting either wrong throws `NotFoundError` or deletes a row that just moved.

### src/ui/tree.ts · reveal and expansion
- refers-to: src/ui/tree.ts :: `setActive`, `revealActive`, `toggle`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Expanded rows persist per workspace in `formmd.expanded.<workspace>`. Opening a page expands its ancestors and scrolls its row into view; this runs again when the tree loads, since the shell mounts before the store is seeded.

### src/ui/tree.ts · pointer drag
- refers-to: src/ui/tree.ts :: `pointerdown` handler, `resolveTarget`, `updateDrag`, `endDrag`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Pointer events, not HTML drag and drop, so the ghost, zones and scrolling are ours. A drag starts after 5 px. The top 30 percent of a row means before, the bottom 30 percent after, the middle inside; dropping into the page's own subtree is not a target. After an expanded row with children means first child. Hovering inside for 650 ms expands the row; the list scrolls near its edges. Favorites accept before and after, and a tree row dropped there becomes a favorite at that spot. Escape cancels; the click that ends a drag is swallowed. Touch keeps native scrolling and does not drag.

### src/ui/tree.ts · keyboard
- refers-to: src/ui/tree.ts :: `keydown` handler, `setRoving`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

One row at a time has `tabindex="0"`. Up and down walk visible rows, right expands or enters, left collapses or climbs, Home and End jump, Enter opens (with Ctrl or Cmd in a new tab), F2 opens and focuses the title, Ctrl or Cmd plus Backspace (or Delete) trashes with an undo toast and hands focus to the next row, Shift+F10 or the context-menu key opens the menu.

### src/ui/page-view.ts · in-place header
- refers-to: src/ui/page-view.ts :: `paint`, `setValue`
- kind: warning-context
- logged: v0.3.0 · 2026-10-07

The header is built once. Switching pages rewrites its fields in one pass; until the next page's data exists (a link to a page outside the loaded tree) the previous page stays rather than blanking. A focused title or description is never overwritten except when the page changes. `field-sizing: content` sizes the textareas where supported; elsewhere `autosize` sets the height.

### src/ui/page-view.ts · tags
- refers-to: src/ui/page-view.ts :: `suggestTags`, `cleanTag`, tag input handlers
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Suggestions come from the workspace's tags by frequency: prefix matches, then substring matches, then close spellings (edit distance 1 from three letters, 2 from five). Enter, comma or Tab add the typed text unless a suggestion was chosen with the arrows; Backspace in an empty field removes the last tag. Tags are trimmed, lose a leading `#`, and spaces become hyphens, matching the server. A chip opens a menu to filter by the tag, set its workspace colour or remove it.

### src/ui/pickers.ts · popovers
- refers-to: src/ui/pickers.ts :: `popover`, `openIconPicker`, `openCoverPicker`, `uploadPicture`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

Pickers are anchored, non-modal layers on the popover layer, so they open above dialogs (workspace settings) and close on an outside press or Escape. The icon grid filters by hiding buttons, never rebuilding; a tint chosen while the page already has an icon recolours it at once. Uploads go through a temporary file input and report progress, success or the server's refusal in one toast.

### src/ui/routes.ts · views
- refers-to: src/ui/routes.ts :: `mountRoutes`; src/styles/pages.css :: `.view` rules
- kind: explainer
- logged: v0.3.0 · 2026-10-07

All four views exist in the document; `data-route` on the root shows one (the server sets it too, so the first paint shows the right one). Scroll positions are remembered per route and page.

### src/ui/topbar.ts · update notice
- refers-to: src/ui/topbar.ts :: `paintBanner`; src/styles/pages.css :: `.banner`
- kind: rationale
- logged: v0.3.0 · 2026-10-07

The notice floats over the sheet instead of taking a grid row, so its arrival never pushes the page down. Reload flushes pending edits first.

### src/styles/pages.css · picture placeholders
- refers-to: src/styles/pages.css :: `.tile[data-kind]`, `.avatar[data-kind]`
- kind: explainer
- logged: v0.3.0 · 2026-10-07

The server renders the initial letter and the kind of picture; until the client paints the icon or image, the letter is transparent, so a workspace with an icon never flashes its letter first.

### z-index: menus and popovers
- refers-to: src/styles/panels.css :: `.menu`; src/styles/pages.css :: `.popover`, `.drag-ghost`
- kind: warning-context
- logged: v0.3.0 · 2026-10-07

Menus, popovers and the drag ghost moved from `--z-dropdown` to `--z-popover`: anything anchored must be able to open from inside a modal dialog, which sits on `--z-overlay`. `--z-dropdown` is now unused and kept in the scale.

---

## Phase 4: documents and editor {#p4}

### src/engine/doc/docs.ts · sessions, readiness and generation
- refers-to: src/engine/doc/docs.ts :: `DocSession.opened`, `message`, `markReady`, `reset`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

A session is ready when IndexedDB has loaded a non-empty fragment or the server's step 2 has arrived, whichever is first, or when access is denied (so a waiting view can show the notice). On `opened` the session reads the generation stored in IndexedDB before it answers anything: incoming sync frames are queued in `gate` until then, and a mismatch clears the local copy and starts a fresh Y.Doc (`reset`), which the editor rebinds to. Local updates are sent only while the socket is live and the person may write; updates applied from IndexedDB or from the server (origin is the persistence or the session) are never echoed, and offline edits reach the server through the step 2 that answers the server's step 1 on the next `opened`. A read-only session does not answer the server's step 1.

### src/engine/doc/docs.ts · DocStore
- refers-to: src/engine/doc/docs.ts :: `DocStore.open`, `release`, `prefetch`, `trim`, `clearLocal`, `dbName`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

Sessions are kept in insertion order as an LRU of eight; a session with references is never evicted. `prefetch` opens and releases, so hovering a row loads the page into memory and IndexedDB without holding it. Database names are `formmd-doc-<user>-<page>`, so two accounts in one browser never share a copy, and sign-out deletes every `formmd-doc-` database. Hidden tabs set every local awareness state to null and restore an empty state when visible again; the cursor plugin refills it on the next selection.

### src/engine/doc/sync.ts · DocSync
- refers-to: src/engine/doc/sync.ts; backend log `internal/doc/conn.go · the socket protocol`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

One socket serves every page. Each channel is re-opened after a reconnect. The watchdog closes a socket that has heard nothing for 45 s (the server pings every 20 s). Backoff is jittered from 0.5 s to 15 s and reset by `reconnectNow` on online and visibility. Close code 4001 (signed out) stops reconnecting.

### src/engine/doc/schema.ts · names and DOM are a contract
- refers-to: src/engine/doc/schema.ts; backend log `internal/doc/html.go · first-frame HTML must match the editor`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

Node and mark names are the CRDT's element and format names, read by the Go side; renaming one orphans stored documents. Mark order (link, strong, em, strike, underline, highlight, color, code) decides nesting in the DOM and matches the server's. Block ids come from `newBlockId` (ten base32 characters) and are not read from pasted HTML, so a pasted block always gets a fresh id. The callout's class is `doc-callout` because `.callout` is the settings dialogs' notice box.

### src/extensions/blocks/editor.ts · zero-flash binding
- refers-to: src/extensions/blocks/editor.ts :: `bind`, `show`, `loading`
- kind: rationale
- logged: v0.4.0 · 2026-10-08

The view is created once with the server-rendered body element as its root (`mount`), and every later page is an `updateState` on the same view. The state's document comes from `initProseMirrorDoc` and its mapping is passed to `ySyncPlugin`, so the first render already shows the content and the plugin does not re-render on a timer. A page that is not ready yet shows nothing new: the page view holds (see below), and only past the hold does the editor switch to an empty, read-only state with a reserved height (`is-loading`). External pictures in pasted HTML are removed, because the document CSP allows only this site's images.

### src/ui/page-view.ts · the page switch hold
- refers-to: src/ui/page-view.ts :: `paint`, `HOLD_MS`, `onPage`
- kind: rationale
- logged: v0.4.0 · 2026-10-08

When the route moves from one page to another and the next document is not ready, the whole page view (header and body) keeps showing the old page until the document is ready or 450 ms pass, so a title is never painted over another page's body. Coming from another route, or the first paint, never holds. Hovering a page row for 90 ms prefetches its document, which makes most switches instant.

### src/extensions/blocks/plugins.ts · block ids
- refers-to: src/extensions/blocks/plugins.ts :: `idsPlugin`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

After any local change, blocks without an id or with an id already seen earlier in the document get a fresh one, so the first copy keeps its id after a split or a paste. Transactions that come from the sync plugin are skipped, so two people never fight over one block's id, and the fix is kept out of undo history.

### src/extensions/blocks/triggers.ts · slash, @ and [[
- refers-to: src/extensions/blocks/triggers.ts :: `triggerPlugin`, `TriggerView`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

A trigger opens only at the start of a text block or after whitespace, never in code. The typed character is inserted by the plugin in the same transaction that opens the menu, so the menu's start position is exact. The menu closes when the cursor leaves the run, the trigger character is deleted, the query has two spaces in a row, or (for the slash menu) nothing matches and a space was typed. Members for @ are fetched once a minute per workspace.

### src/extensions/blocks/index.ts · editor shortcuts in the shared keymap
- refers-to: src/extensions/blocks/index.ts :: `setup`; src/ui/keyboard.ts :: `activeScopes`; src/ui/shortcuts-sheet.ts :: `LISTED_SCOPES`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

Marks, turn into, undo and redo, moving and duplicating blocks and ticking to-dos are commands bound in the `editor` scope, which is active while focus is inside a `.ProseMirror` element. The window capture listener runs them before ProseMirror sees the key, and a command that is not enabled lets the key through (Ctrl+Enter falls back to leaving a code block). Enter, Backspace, Tab and the arrows stay in ProseMirror's own keymap.

### src/extensions/blocks/markdown.ts · on-demand parser
- refers-to: src/extensions/blocks/markdown.ts :: `preloadMarkdown`, `looksLikeMarkdown`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

markdown-it is loaded the first time the editor gets focus. Until it has loaded (or if loading failed offline, which resets the loader so the next focus retries) pasted text is handled as plain text. The highlighter chunk follows the same retry rule.

### src/extensions/blocks/nodeviews.ts · CSP and node views
- refers-to: src/extensions/blocks/nodeviews.ts; src/extensions/blocks/editor.ts :: `cursorBuilder`, `selectionBuilder`
- kind: warning-context
- logged: v0.4.0 · 2026-10-08

Nothing in the editor may set a `style` attribute: the document CSP has no `unsafe-inline` for styles. Colours travel as `data-color` or `data-tone` and resolve through the `--tint` tokens; positions use the CSSOM (`el.style.transform`), which the CSP allows. That is why the collaborator cursor and selection builders replace y-prosemirror's defaults.

### src/extensions/blocks/media.ts · uploads
- refers-to: src/extensions/blocks/media.ts :: `uploadFiles`, `boundary`
- kind: explainer
- logged: v0.4.0 · 2026-10-08

Uploads use XMLHttpRequest for progress. A local-only widget marks the insertion point at a block boundary while the file uploads; the finished block is inserted where the widget has been mapped to, so other people's edits in between do not misplace it. Images are measured first so the block reserves its height. The form fields keep the order the server reads: purpose, page_id, name, then the file.

### src/engine/collections.ts · query sessions and windows
- refers-to: src/engine/collections.ts :: `QuerySession`, `setVisible`, `pump`, `window`, `dropWindows`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

A session holds one view's result as `slots` (row ids by position, `undefined` until loaded) plus the rows and refs it has seen. Rows load in windows of 100. `setVisible` cancels any window that has left the visible range by more than one window, waits until scrolling pauses for 70 ms (or 220 ms at most while it keeps moving) and then fetches at most three missing windows, nearest first; each finished window pulls the next. A cancelled request rejects with `cancelled` and is never reported. Grouped views keep `groups` and `lanes` instead of slots and page within a group with `moreInGroup`.

### src/engine/collections.ts · edits are optimistic
- refers-to: src/engine/collections.ts :: `setValue`, `flushRow`, `pending`, `apply`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

`setValue` patches every open session at once and sends the change after the debounce; changes made while a write is in flight are folded into the next write. While a row has pending changes, live events for only that row are ignored, so a slow echo cannot undo typing. A failed write reports the error and refreshes the row from the server.

### src/engine/collections.ts · first pages
- refers-to: src/engine/collections.ts :: `seedQuery`, `takeBootQuery`, `load`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

The boot document and `GET /api/collections/{id}?view=` both carry the first 100 rows of a table or list view. They are kept per view id and taken once, within 15 seconds; an older one is ignored and the view queries instead. `load(id, false, view)` asks for the first page only when the collection is not cached yet.

### src/engine/api.ts · cancellable requests
- refers-to: src/engine/api.ts :: `request` (`cancel`); src/engine/client.ts :: `collections.query`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

`request` takes an optional `AbortSignal`. Aborting it throws `ApiError` with code `cancelled`, which callers drop silently; the timeout still throws `timeout`.

### src/ui/page-view.ts · switching to a database
- refers-to: src/ui/page-view.ts :: `HOLD_MS`, `HOLD_DB_MS`, `prepareDb`, `showDb`, `paint`; src/styles/shell.css :: `.topbar::after`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

The current page stays on screen until the next one is ready: 450 ms at most for a document, 1,600 ms for a database, whose readiness means its first rows are painted in a detached box. While a switch is held the root carries `data-navigating`, and after 180 ms a thin line moves along the bottom of the top bar; a switch that is ready sooner never shows it. Do not shorten the database hold below one slow round trip, or a database title appears over an empty table.

### src/styles/database.css · full-width databases without a width change
- refers-to: src/styles/database.css :: `.view--page`, `.page-db`, `--db-w`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

The page column keeps the same width for documents and databases; a database breaks out of it with container query units (`--db-w: max(100%, calc(100cqi - 2 * var(--space-10)))` and a negative start margin). Changing `.page` width by page kind made every switch between a document and a database shift the layout.

### src/extensions/database/views/table.ts · virtual rows
- refers-to: src/extensions/database/views/table.ts :: `ROW_H`, `OVERSCAN`, `paintFlat`, `visibleRange`
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

Ungrouped tables give the body the full height (`total * 34px`) and position only the visible rows plus 10 above and below with `transform`. Rows not yet loaded are drawn as placeholders of the same height, so nothing below moves when they fill. Grouped tables flow normally and page within each group. Keep `ROW_H` equal to `--row-h` in the stylesheet.

### src/extensions/database/surface.ts · switching views
- refers-to: src/extensions/database/surface.ts :: `mountView`, `ready`
- kind: explainer
- logged: v0.5.0 · 2026-10-08

A new view renders off screen and replaces the old one when its first result is painted, or after 450 ms. The surface's `ready` resolves after the first view has painted, which is what the page switch waits for.

### src/extensions/database/views/timeline.ts and calendar.ts · keyed painting
- refers-to: src/extensions/database/views/timeline.ts :: `ResizeObserver`; src/extensions/database/views/calendar.ts :: reconcile of cells and items
- kind: warning-context
- logged: v0.5.0 · 2026-10-08

The calendar reconciles days and items by key; it never rebuilds the month grid. The timeline measures its width before placing bars and scrolling to today, and repaints once through a ResizeObserver when it was first painted detached with zero width.

## Phase 6: search and editor completion {#p6}

### src/extensions/database/load.ts · the database chunk and module preload
- refers-to: src/extensions/database/load.ts :: `loadDatabase`, `databaseViews`; src/extensions/database/views-entry.ts; ../internal/static/static.go :: `readPreloads`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

The views, row pages, forms and CSV import load as one chunk the first time a database is shown, which brought the up-front script back under its budget. When the server renders a page that needs the chunk, it reads the Vite manifest and adds `modulepreload` links for it in the head, so a database page loaded directly does not wait for a second round trip.

### src/ui/search.ts · the palette
- refers-to: src/ui/search.ts :: `mountSearch`, `fillMarked`; src/main.ts :: `ensureSearch`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

Ctrl or Cmd plus K (or P) opens it; the chunk is fetched on first use and preloaded when the browser is idle. Typing is debounced and every request carries an abort signal, results are reconciled by key, and `\x02`/`\x03` markers from the server become `mark` elements through DOM nodes, never through HTML. A leading `>` switches to actions. Escape goes through the layer stack, whose close handler hides the palette.

### src/extensions/blocks/math.ts · equations
- refers-to: src/extensions/blocks/math.ts :: `MathView`, `mathRules`, `mathKeys`; src/extensions/blocks/math-render.ts :: `renderTex`; src/extensions/blocks/mathml.ts :: `mathElement`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

Block and inline equations store TeX and the rendered MathML; KaTeX (with mhchem) loads on first edit and produces MathML only, which every browser draws natively, so readers and the first frame need no script. MathML is rebuilt through an element and attribute allow-list on the client as on the server. `$$x$$` typed in text becomes inline math and `$$` on an empty line a block; a single `$` is left alone so prices stay prices.

### src/extensions/blocks/columns.ts · never touch the row's attributes
- refers-to: src/extensions/blocks/columns.ts :: `ColumnView.drag`, `is-resizing`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

The `columns` node has no node view, so ProseMirror re-reads any attribute change on its element: adding a class to the row while dragging a resize grip made the editor re-parse the row, rebuild every column and drop the pointer capture. State during a drag lives on the column's own element (its node view ignores its own attribute mutations); widths are painted as `data-w` on each column and written to the document once, on pointer up.

### src/extensions/blocks/columns.ts · side drops and empty columns
- refers-to: src/extensions/blocks/columns.ts :: `sideTarget`, `sideDrop`, `removeDragged`, `columnsPlugins`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

A drag over the outer fifth of a block (at most 64 px) shows a vertical marker and drops beside it; a drag anywhere else is ProseMirror's own drop. When that ordinary drop lifts the only block out of a column, ProseMirror leaves an empty paragraph to satisfy the schema; the drop handler remembers the column and the next drop transaction removes it (or unwraps a two-column row). Any row left with one column is unwrapped by the normalising plugin.

### src/extensions/blocks/handle.ts · the handle inside columns
- refers-to: src/extensions/blocks/handle.ts :: `track`, `lastX`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

The handle probes the column under the pointer rather than the editor's left edge, so blocks in later columns get their own handle. Pointer moves over the handle itself are ignored (the target stays put) and moves over a column resize grip hide it.

### src/extensions/blocks/links.ts · previews and imports
- refers-to: src/extensions/blocks/links.ts :: `linksPlugin`, `importMedia`, `unfurlBookmark`, `offerLinkChoice`, `embedFor`
- kind: explainer
- logged: v0.6.0 · 2026-10-08

Local transactions that add media or link cards mark the plugin dirty; the scan runs after the dispatch. Media whose source is another site or a data URL are imported once per block and replaced by the file block with its size, and an import that fails offers a retry in the block. Link cards are marked `fetched` before the request so collaborators do not fetch the same card. The paste menu anchors to a 1 px element in the editor wrapper, outside ProseMirror's DOM, because the menu sets attributes on its anchor.

### src/extensions/blocks/diagram.ts · the renderer bridge
- refers-to: src/extensions/blocks/diagram.ts :: `drawDiagram`, `sourceSum`; src/sandbox/main.ts; src/extensions/blocks/nodeviews.ts :: `CodeView.paintDiagram`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

One hidden iframe (`sandbox="allow-scripts"`, opaque origin) renders every diagram; replies are accepted only from its window. The SVG comes back sized from its viewBox and is shown only as an `img` from a data URL, so nothing it contains can run. The drawing, its size and a checksum of the source are stored on the code block by a writer, so readers and the first frame need no renderer; a stale checksum triggers a redraw. Dark themes invert the neutral drawing with a CSS filter. Mermaid splits labels into word spans, so tests must not look for a phrase inside the SVG.

### src/extensions/blocks/synced.ts · nested editors
- refers-to: src/extensions/blocks/synced.ts :: `SyncedView`, `createSynced`, `contentOf`; src/extensions/blocks/editor.ts :: `prepare`, `syncedSnapshot`, `doc`
- kind: warning-context
- logged: v0.6.0 · 2026-10-08

A synced block is an atom whose view opens the original page's session and mounts a nested editor bound with `ySyncPlugin` to that document's `s:<key>` root, with its own undo manager. The nested editor's context is the host's with `pageId`, `view` and `writable` overridden, so uploads land on the original page. The global keymap only handles editor shortcuts when the main view has focus, so the nested editor binds undo, redo and the marks itself. The view rebinds after a session reset (a new Y.Doc). The first frame's synced HTML is captured before ProseMirror replaces it and shown until the source session is ready; page switches inside the app prefetch the source documents (up to six, 1.5 s at most) so they are usually ready. Unsync all sets `released` in the original document's `synced` map; copies convert themselves when a writer has them open.

---

## Phase 7: collaboration, sharing and identity {#p7}

### src/engine/index.ts · can() and guests
- refers-to: src/engine/index.ts :: `can`, `canWrite`; src/engine/pages.ts :: `teams`, `roots`, `shared`, `teamOf`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

Every page row carries its level from the server; `can(id, level)` compares against it, so menus, the editor and the share dialog never compute access themselves. Guests have no workspace role: their tree is the shared roots only, and pages detached from a hidden parent arrive with `detached` set.

### src/extensions/blocks/comments.ts · anchors follow the text
- refers-to: src/extensions/blocks/comments.ts :: `encodeRange`, `resolveAnchor`, `commentsPlugin`
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

Anchors are Yjs relative positions encoded as base64. Remote transactions (ySync origin) rebuild the highlight decorations from the anchors; local ones map them. Read-only bubbles set the range before placing the floating element, or they place off-screen.

### src/extensions/blocks/suggest.ts · drafts as decorations
- refers-to: src/extensions/blocks/suggest.ts :: `suggestPlugin`, `suggestKey`, `suggestUI`; src/extensions/blocks/editor.ts :: modes
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

In Suggesting, `filterTransaction` blocks document changes and typing, keys and paste build drafts drawn as decorations (deletions inline, insertions as widgets with `side: -1`). The draft takes the server id as soon as the POST returns and keeps the widget key `${id}:${text}`, so the stored suggestion does not render twice. Accepting dispatches with meta `{apply: true}`, the only way past the filter; `restoreDoc` uses the same meta.

### src/extensions/blocks/editor.ts · restoreDoc keeps blocks being typed in
- refers-to: src/extensions/blocks/editor.ts :: `restoreDoc`, `remoteBlocks`; src/ui/history-dialog.ts :: restore
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

A restore replaces the whole document in one transaction; y-prosemirror turns it into a minimal Yjs diff, so typing that has not reached this browser yet merges. Top-level blocks holding another collaborator's cursor (from awareness, mapped through the binding) are taken from the live document instead of the version; such a block absent from the version is inserted after its nearest kept neighbour. The history dialog saves a `restore` version first.

### src/ui/history-dialog.ts · preview from the stored state
- refers-to: src/ui/history-dialog.ts :: `versionDoc`, `compare`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

A version's base64 Yjs state is applied to a scratch Y.Doc and turned into a ProseMirror node with `yXmlFragmentToProseMirrorRootNode`, then serialised with the schema's DOM serializer into the preview. Blocks are compared by block id (by position when a block has none): `changed` when the content differs, `gone` when the live page no longer has it.

### src/site · the published page script and stylesheet
- refers-to: src/site/main.ts; src/site/site.css; vite.config.ts :: input `site`, `iconTable`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The server renders published pages; this entry only opens toggles and reveals anchors inside closed toggles. The stylesheet imports the tokens, base, covers and editor styles so published blocks look like the editor, and overrides `body` to scroll. The build also emits `.vite/icons.json` (every UI icon and page glyph as SVG) for the server to inline.

### src/ui/publish-dialog.ts · partial updates
- refers-to: src/ui/publish-dialog.ts :: submit; internal/site/site.go :: `Input`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

Every site field is optional on the server and omitted fields keep their value; `password: ""` removes the password and an omitted password keeps it. The status is fetched before the dialog opens so it never opens empty and then fills.

### src/ui/form.ts and forms.css · switches that load
- refers-to: src/styles/forms.css :: `.switch[data-loading]`; src/ui/workspace-sites.ts, workspace-analytics.ts, account.ts
- kind: warning-context
- logged: v0.7.0 · 2026-10-09

A switch whose value comes from the server keeps its control invisible (`data-loading`) until the value arrives, so it never flips after paint. Lists that load start with their empty message hidden for the same reason.

### src/ui/bar-chart.ts · in-place bars
- refers-to: src/ui/bar-chart.ts :: `barChart`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The SVG and its bars are created once and reused; a new series only changes attributes, adding or removing bars when the number of days changes. Each bar has a `title` with the date, views and people.

### src/ui/auth.ts and index.html · the SAML button and SAML-only
- refers-to: index.html :: `data-sso-id="saml"`, `<!--slot:saml-name-->`, `.auth__alt--owner`; src/styles/auth.css :: `data-saml-only`; src/ui/auth.ts :: `owner-password`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The button's label is a server slot so it is right in the first frame. `data-saml-only="true"` hides the password step, the submit button, the divider and the sign-up line; the owner link sets it to `false` on the client to reveal the password form.

### src/main.ts and src/ui/settings-entry.ts · the settings chunk
- refers-to: src/main.ts :: `ensureSettings`; src/ui/settings-entry.ts :: `mountSettings`; src/ui/dialog.ts :: `overlayDialog`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The account, workspace and instance settings dialogs with all their tabs load as one chunk (23 KB gzipped), fetched when one of their overlays is asked for or at idle 2.5 s after start, like the search palette; this took the up-front script from 251 KB back to 229 KB. Because the chunk can arrive after the overlay was set, `overlayDialog` and the workspace dialog's overlay watchers run immediately on mount, so a dialog asked for early opens as soon as its code is there. A failed load clears the overlay so asking again retries.

### src/engine/index.ts · a role change refreshes page levels
- refers-to: src/engine/index.ts :: `setMe`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

Page levels come with the tree and the page detail, so a changed workspace role (for example member to viewer) would leave the old levels in place and the editor writable. `setMe` compares each workspace's role with the previous `me`; for each workspace whose role changed it reloads that tree if loaded and the open page's detail, and the editor's `detail` watcher then rechecks `editable`.

### src/styles/pages.css and index.html · first-frame chrome
- refers-to: src/styles/pages.css :: `:root:not([data-route="page"]) .topbar__page-actions`; index.html :: `<!--slot:version-->`
- kind: explainer
- logged: v0.7.0 · 2026-10-09

The page actions in the top bar are hidden by CSS on every route but a page, so a home or library load does not paint them and then remove them. The sidebar version text is a server slot, so the footer line has its height in the first frame instead of growing when health arrives.

## Phase 8: agents and AI {#p8}

### src/engine/api.ts · stream
- refers-to: src/engine/api.ts :: `stream`
- kind: explainer
- logged: v0.8.0 · 2026-10-09

POSTs JSON and reads server-sent events from the response body with a `TextDecoderStream`, splitting on blank lines. An `error` event becomes an `ApiError` with the server's code; an aborted signal becomes code `cancelled`, which callers treat as a quiet stop.

### src/extensions/blocks/ai.ts · the Ask AI panel
- refers-to: src/extensions/blocks/ai.ts :: `openAI`, `aiPlugins`, `AIPanel`, `domSelection`, `render`, `reconcile`, `clean`
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

The target range lives in plugin state and is mapped through every transaction, so remote edits during a stream move it correctly; a non-empty target is drawn with an inline decoration because focus moves to the panel. `openAI` first reads the DOM selection with `posAtDOM`: a key pressed right after a click or End can arrive before ProseMirror has read the new selection. The preview renders Markdown by parsing it into the schema and serializing that, never by inserting markdown-it's HTML, and drops media nodes first so a prompt injection cannot make the preview or the page load remote URLs. `reconcile` updates the preview in place (text data, matching elements recursively), so streaming replaces nothing.

### src/extensions/blocks/ai.ts · the AI block
- refers-to: src/extensions/blocks/ai.ts :: `AIBlockView`, `newAIBlock`
- kind: explainer
- logged: v0.8.0 · 2026-10-09

The stream shows in a live layer while the body is hidden by a class on the block, then one transaction replaces the body and the layer goes in the same frame. The Regenerate button is hidden for readers by CSS on a read-only editor, not by the view, because a node view is not rebuilt when writability changes.

### src/extensions/database/autofill.ts · autofill
- refers-to: src/extensions/database/autofill.ts :: `autofillEditor`, `runAutofill`; src/extensions/database/propedit.ts
- kind: explainer
- logged: v0.8.0 · 2026-10-09

Saving a property whose autofill was just turned on or whose instruction changed fills every row; progress goes to one toast that updates in place. A second run on the same property aborts the first.

### src/engine/collections.ts · coalesced refreshes
- refers-to: src/engine/collections.ts :: `scheduleRefresh`
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

Changed row ids accumulate in `dirty` until a refresh lands that was not superseded (its `seq` is still the latest). Before this, a burst of row events kept only the last event's ids, so rows updated in quick succession had fresh data but were never repainted.

### src/ui/agent-pane.ts · the agent pane
- refers-to: src/ui/agent-pane.ts :: `mountAgentPane`, `send`, `showMessage`; src/main.ts :: `ensureAgent`
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

Loaded on first open as its own chunk; `main.ts` owns the top-bar click and the Ctrl+Shift+J command so a click before the chunk loads still opens it. The pane is a flex column, not a grid: hidden children leave a grid's row assignment and pushed the composer into the stretching row. The person's message and a thinking row are added on send, before the network, so the empty state leaves inside the input window and nothing shifts when the stream starts. Escape closes the pane only when no menu or other layer is open.

### src/ui/workspace-agents.ts · custom agents
- refers-to: src/ui/workspace-agents.ts :: `customAgentsSection`, `editor`, `describe`
- kind: explainer
- logged: v0.8.0 · 2026-10-09

Schedules are sent with the browser's IANA time zone. The trigger fields shown follow the trigger and repeat choices. Rows reconcile by id, and Edit reads the latest agent kept on the row element.

### src/extensions/blocks/meeting.ts · recording
- refers-to: src/extensions/blocks/meeting.ts :: `Recorder`, `MeetingView`, `process`, `place`
- kind: warning-context
- logged: v0.8.0 · 2026-10-09

The microphone and, when chosen, the shared tab or system audio are mixed in an `AudioContext` into one stream recorded as 32 kbps Opus; the recorder restarts every ten minutes so each segment is a file the transcription model can read alone (a timeslice chunk would lack headers), and stops itself after four hours. Recorders live in a map by block id so a rebuilt node view picks up a recording in progress. A block left in a working state by a closed tab offers Reset.

### src/ui/search.ts · AI answers
- refers-to: src/ui/search.ts :: `runAnswer`, `stopAnswer`, `QUESTION`; src/styles/search.css :: `.srch__*` grid rows
- kind: explainer
- logged: v0.8.0 · 2026-10-09

"Ask AI" is listed first when the query reads as a question and after the pages otherwise, so Enter keeps opening the best page for ordinary searches. Each palette part has an explicit grid row because the answer section is hidden most of the time.

## Phase 9: automations, charts, buttons, templates, views {#p9}

### src/extensions/database/views/chartdraw.ts · keyed SVG and export
- refers-to: src/extensions/database/views/chartdraw.ts :: `drawChart`, `keyed`, `layer`, `standalone`, `exportChart`
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

Marks are reconciled by key inside fixed layers (grid, marks, labels), so a live update moves a bar's attributes instead of replacing the chart, and the frame keeps its size. The export copies computed fill, stroke and font values into presentation attributes, never a `style` attribute: the CSP forbids inline styles and the file must look right outside the app. PNG export draws that SVG onto a canvas at twice the size.

### src/extensions/database/views/chart.ts and dashboard.ts · settings and widgets
- refers-to: src/extensions/database/views/chart.ts :: `chartView`, `chartSettings`, `defaultSpec`; src/extensions/database/views/dashboard.ts :: `dashboardView`, `defaultWidgets`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

The chart spec lives in `layout.chart` of the view config and a dashboard's cards in `layout.widgets`; both save through `saveConfig` with a short debounce. Each card asks the server for its own figures and redraws on the session's `watch` callback, which fires on live events without running the table query.

### src/engine/collections.ts · pending view edits
- refers-to: src/engine/collections.ts :: `viewEdits`, `withEdits`, `SessionOptions.watch`, `SessionOptions.tree`
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

A view change is overlaid on every collection reload until the server confirms it (by sequence number), because on a slow link a reload triggered by an unrelated event could otherwise land first and put the old config back for a moment. `tree` and `expanded` pass the sub-items state to the query.

### src/extensions/database/automations.ts and src/extensions/automation · the dialog
- refers-to: src/extensions/database/automations.ts :: `openAutomations`; src/extensions/automation/actions.ts :: `actionEditor`; src/extensions/automation/api.ts :: `automationApi`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

The dialog and the action editor load on first use; the action editor is shared with buttons so both offer the same actions. The API client is a thin wrapper over `engine.api.http` in the lazy chunk rather than methods on the engine client, which keeps the up-front script under budget.

### src/extensions/blocks/button.ts and buttonrun.ts · buttons
- refers-to: src/extensions/blocks/button.ts :: `ButtonView`, `buttonRunner`, `newButton`; src/extensions/blocks/buttonrun.ts :: `runButton`, `insert`, `editButton`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

The node view is tiny and always loaded; running and editing load `buttonrun.ts` on first press. Confirm runs first, insert and open happen in the editor, and the remaining actions go to `POST /api/buttons/run` in one request. Inserted Markdown goes through the editor's Markdown parser with variables filled, in one transaction.

### src/extensions/database/views/map.ts · fitting
- refers-to: src/extensions/database/views/map.ts :: `mapView`, `fit`, `project`
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

The basemap is Web Mercator paths precomputed into `public/maps/world-110m.json` and fetched once. `fit` needs the frame's size; when the view is built before layout it sets `needFit` and the ResizeObserver fits on the first real size. Until then the SVG has no `data-ready` and its layers are hidden, so pins never show at the wrong scale.

### src/extensions/database/views/feed.ts · posts
- refers-to: src/extensions/database/views/feed.ts :: `feedView`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

The body excerpt has a fixed collapsed height and the Show more button is absolutely placed inside its wrapper, so measuring whether it is needed (with a ResizeObserver) never moves the post.

### src/ui/page-actions.ts · lazy page dialogs
- refers-to: src/ui/page-actions.ts :: `openShareDialog`, `prefetchPageDialogs`, `pageMenu` ("Save as template")
- kind: explainer
- logged: v0.9.0 · 2026-10-09

The share, publish and history dialogs load on first use and are prefetched at idle from the top bar, so the first click is not slower in practice and the up-front script lost about 5 KB.

### src/ui/templates-dialog.ts · the gallery
- refers-to: src/ui/templates-dialog.ts :: `openTemplates`, `useTemplate`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

One dialog serves the sidebar (new page) and the page view (into a parent). Starters and the workspace's templates come from one request; using one opens the new page.

### src/extensions/database/views/table.ts · sub-items and fitting columns
- refers-to: src/extensions/database/views/table.ts :: `paintRow`, `applyTree`, `autoFit`, `lastGrip`
- kind: warning-context
- logged: v0.9.0 · 2026-10-09

The chevron is prepended to the title cell, so the value span is found by class, not as the first child (painting the first child overwrote the chevron). Open branches live in localStorage per view and are sent with each query; the first-paint seed is skipped in tree mode because the seeded window is in flat order. The resize handle calls `preventDefault` on pointerdown, which suppresses `dblclick`, so a double press is detected by hand: two presses on the same handle within 400 ms.

### src/ui/graph-view.ts · the link graph
- refers-to: src/ui/graph-view.ts :: `openGraph`, `layout`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

The force layout runs a fixed number of ticks before the first draw (fewer above 1,500 nodes) with repulsion limited to neighbouring grid cells, so it settles in one pass instead of animating. Drawing is on a canvas at device pixel ratio; every interaction schedules one frame.

### src/ui/tree.ts · Databases section
- refers-to: src/ui/tree.ts :: `renderDatabases`
- kind: explainer
- logged: v0.9.0 · 2026-10-09

Recomputed only when a page event touches a database page (created, renamed, moved, trashed) and reconciled with `reconcileFlat`, so existing rows are kept and only changed ones are patched. Templates and databases that are themselves rows of another database are left out.

## Phase 10: import, export and API {#p10}

### src/ui/transfer-dialog.ts · import and export
- refers-to: src/ui/transfer-dialog.ts :: `openImport`, `send`, `openExport`, `download`
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

The upload uses XMLHttpRequest for upload progress (fetch has none); the job is then polled every 600 ms. The status area has a fixed three-row grid (bar, line, result row) present from the start, so starting, progress, finishing and the Open buttons never resize or move the dialog; the file list stays as it was until new files are picked. Exports are plain downloads through a temporary anchor, so the browser handles large zips and the session cookie. Both dialogs load on first use from the sidebar, the page menu and workspace settings.

### src/ui/menu.ts · reopening after an item
- refers-to: src/ui/menu.ts :: `menuButton`, the item click handler
- kind: warning-context
- logged: v0.10.0 · 2026-10-09

`menuClosedAt` stops the press that closed a menu from reopening it; choosing an item now clears it, since that close did not come from the anchor.

## Phase 11: offline and German {#p11}

### public/sw.js · offline start
- refers-to: public/sw.js :: `precache`, `refreshShell`, `fromCache`, `navigate`, the message handler
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

Install caches the core assets from `/offline.json` (they come from the HTTP cache, already loaded); `offline-precache` caches the rest and drops assets no longer built. `/assets/*` is cache-first with `ignoreVary` (assets vary on Accept-Encoding). App navigations (`/`, `/p/<id>`, `/library`, `/trash`, `/inbox`) are network-first; on failure the cached `/offline-shell` is returned with `data-route` injected for the requested path, which keeps the first paint still. The shell is cached only when it is a signed-in document; sign-out deletes it and the API cache (`DocStore.clearLocal`). `GET /api/collections/<id>` and `GET /api/rows/<id>` are network-first into `formmd-api`, so a database opened or kept online opens offline with its first window of rows (read-only); a miss on the exact URL falls back to the same path with another query. The worker never claims an open page: claiming mid-load made Chrome discard the modulepreloads as a cross-world mismatch and fetch those chunks twice. "Available offline" also asks the worker to cache a kept database's `?view=` read, because the page that asks may not be controlled yet.

### src/ui/offline.ts and offline-list.ts · Available offline
- refers-to: src/ui/offline.ts :: `startOffline`, `toggleOffline`, `syncDoc`, `refreshKept`; src/ui/offline-list.ts; src/ui/page-actions.ts :: the Available offline entry
- kind: explainer
- logged: v0.11.0 · 2026-10-09

Loaded 2.5 s after start. It registers the worker, refreshes the shell after start and 4 s after tree changes, asks for the full precache after 20 s idle (not with Save-Data) and re-syncs kept pages after 30 s. A kept page and its subpages are opened one by one until the server's step 2 arrives, which writes them to the per-account IndexedDB store. The list of kept roots is per account in localStorage; the menu reads it synchronously, so that part stays in the main chunk.

### src/ui/page-view.ts · first paint without a server body
- refers-to: src/ui/page-view.ts :: `paint` (the pending branch), `data-ready`; src/styles/pages.css :: `:root[data-shell="offline"] .view--page:not([data-ready]) > *`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

The first paint used to be immediate because the server always rendered the body. When it did not (the offline shell), the page painted empty and jumped when the document loaded. Now a first paint whose page is not the server-rendered one waits for the document like a navigation, up to 1.6 s, behind a hidden view.

### src/ui/topbar.ts · tabs offline, and the reserved tab bar
- refers-to: src/ui/topbar.ts :: `tabPage`, `paintTabs`; index.html :: the pre-paint script (`data-tabbar`); src/styles/pages.css :: `:root[data-tabbar="open"] .tabbar`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

`tabPage` fetches a page missing from the tree; it closed the tab on any failure, so offline every tab not yet seeded was closed and the route fell back to home. It now closes only on 404 or 403. The pre-paint script reserves the tab bar from history state or the saved tabs plus the current page, removing a 34 px push on every reload with more than one tab; `paintTabs` clears the attribute.

### src/main.ts and src/ui/status.ts · network errors
- refers-to: src/main.ts :: the `onError` handler; src/ui/status.ts
- kind: explainer
- logged: v0.11.0 · 2026-10-09

An `ApiError` with code `network` no longer raises its own toast; it asks the health check, whose single connection notice (a warning) says saved pages stay open and text edits sync later. The sidebar version keeps its last value while the server is unreachable.

### vite.config.ts · dev proxy Origin
- refers-to: vite.config.ts :: `toBackend`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

The string shorthand proxies with changeOrigin, so the backend saw Host 127.0.0.1:8750 and Origin localhost:5750 and refused every POST as cross-site (setup and sign-in included). Every HTTP proxy entry now rewrites Origin to the backend when one is present, as the sync socket already did.

### public/manifest.webmanifest, src/ui/pull.ts, shell.sync · installed app
- refers-to: public/manifest.webmanifest; index.html :: `.topbar__sync`; src/extensions/shell/index.ts :: `syncNow`, `shell.sync`; src/ui/pull.ts :: `mountPullToRefresh`; src/styles/shell.css :: `.topbar__sync`, `.pull`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

The manifest makes the app installable (standalone, dark background, the SVG mark). An installed window has no reload button, so a Sync button appears through `(display-mode: standalone)` in CSS, decided at first paint so nothing moves; it reconnects the live stream and the sync socket, checks health and reloads the tree. On touch devices in standalone mode, pulling the top of the page down past 72 px runs the same sync; the marker lives in the overlay root and moves by transform only.

### src/ui/admin.ts · the About tab names NodeAgentc, Inc. and links the licence
- refers-to: src/ui/admin.ts :: `aboutTab` (the "This instance" row's meta spans)
- kind: decision
- logged: v0.47.8 · 2026-11-03

The launch punch list (2026-11-03): the About of a downloaded app says NodeAgentc, Inc., not nodemode-ai, and lists the licence. The row under the form.md name now reads "Copyright 2026 NodeAgentc, Inc." (still `translate="no"`) with a link after it, "Apache License 2.0", to the LICENSE file on GitHub (`nodemode-dev/form.md/blob/main/LICENSE`), opened in a new tab like the Open-source licences button beside it. The desktop shell's bundle names the publisher and the licence too (see `src-tauri/HASHOUT_LOG.md`).

### src/ui/admin.ts · tunnel, mail accounts, HTTPS, backup and updates
- refers-to: src/ui/admin.ts :: `generalTab` (tunnel status and buttons, `https.domain`, `https.auto`, `net.trust_proxy`), `emailTab` (Gmail and Graph connect), `aboutTab` (backup download, latest version); src/ui/status.ts :: `data-update`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

Tunnel start and stop call `/api/admin/tunnel/*` and paint the returned status in place. Connecting a mail account opens the provider in a new tab after the settings are saved; the redirect address shown is built from the public address. The backup button is a plain download link, owner only. When `/healthz` reports a newer release, About says so and, for owners and admins, the sidebar version turns accent-coloured with the version in its title.

### src/engine/doc/docs.ts · editing kept pages after a cold offline start
- refers-to: src/engine/doc/docs.ts :: `DocSession.attach` (idb `gen`, `write`), `DocSession.opened`, `loaded`; src/extensions/blocks/editor.ts :: `prepare(id, local)`; src/ui/page-view.ts :: `HOLD_LOCAL_MS`
- kind: warning-context
- logged: v0.11.0 · 2026-10-09

Edit rights come from the server's `opened` frame, which never arrives offline, so each session stores `write` (1 or 0; y-indexeddb takes no booleans) beside `gen` and restores it from IndexedDB while not live. A page with stored `gen` but an empty fragment is still ready. The session marks ready as soon as IndexedDB has content, before reading the two keys, so the first paint is not delayed. In the offline shell, the page view waits on `loaded` (IndexedDB finished, content or not) with a 6 s ceiling instead of the 450 ms hold, because the hold used to release a kept page before IndexedDB answered under a slow CPU and showed the loading box, which then collapsed.

### public/sw.js, src/main.ts · file cache and session end
- refers-to: public/sw.js :: `FILES`, `fileRead`; src/main.ts :: `guardSession`; src/engine/doc/docs.ts :: `clearLocal`
- kind: explainer
- logged: v0.11.0 · 2026-10-09

Whole `200` responses for `/files/<id>` are kept network-first, so pictures on kept pages show offline; range responses are not stored. A `401 unauthorized` while in the app deletes the shell, API and file caches before the redirect to sign-in, so an expired session leaves no account data in the worker's caches.

### public/sw.js · view queries kept for offline
- refers-to: public/sw.js :: `QUERY_READS`, `queryKey`, `queryRead`, the fetch handler's POST branch
- kind: rationale
- logged: v0.32.0 · 2026-10-17

Board, gallery, calendar, timeline, map and chart views read through `POST .../query` and `.../chart`, each with its own body (group limits, a month range, aggregates), so a server-side first window like the table's would have had to copy every view's query and drift from it. The worker instead keeps the answer to the view's own request: Cache API entries must be GET requests, so the key is a synthetic GET of the same path with `?q=` and the hex SHA-256 of the body. Searches, later windows (`offset` above 0) and `ids` lookups are not kept, which bounds the entries to roughly one per view and calendar month visited. Answers are network-first and served only when `fetch` throws, and they live in the same `formmd-api` cache that a `401` on session end deletes. The keeper does not open these views itself, so only views opened online in this browser work offline.

### src/ui/offline.ts, public/sw.js, internal/static/offline.go · the keeper warms views
- refers-to: src/ui/offline.ts :: `warmView`, `keep`; public/sw.js :: `queryKey` (the `view` key), `queryRead`, the `/maps/` branch; internal/static/offline.go :: `offline`
- kind: rationale
- logged: v0.33.0 · 2026-10-17

The keeper cannot send the exact body a view would (the calendar's month range, the board's search state), so the worker also stores each answer under `?view=<view_id>` and falls back to it after the exact key. The keeper then asks once per board and gallery view (100 rows, 50 per group, as a grouped answer) and per calendar, timeline and map view (1000 rows, no groups), and builds a chart's request with the chart view's own `chartOf`, loaded by dynamic import so the up-front bundle does not carry it. Feed, dashboard, heatmap and graph views make several requests each and are not warmed. The map draws nothing without `/maps/world-110m.json`, a public file outside `/assets/`, so the manifest lists `.json` files under `/maps/` and the worker serves that path cache-first like the built assets.

### public/sw.js, src/ui/offline.ts · offline views, corrected
- refers-to: public/sw.js :: `SHAPED`, `queryKey`, the `offline-query` message, the `activate` handler; src/ui/offline.ts :: `warmView`, the route watcher in `startOffline`; src/extensions/database/views/dashboard.ts :: `widgetsOf`, `sparkSpecOf`
- kind: rationale
- logged: v0.37.0 · 2026-10-17

v0.32.0 and v0.33.0 were checked with Playwright's offline switch, which does not reach the service worker; the worker kept fetching from the server, so those checks passed whether or not the cache worked. With the server process stopped instead, none of the views opened offline. Two causes: the keeper's warm requests went from the page, which is not controlled by the worker on a first visit (registration, then a page load before activation), so they bypassed the worker and were never stored; and a database opened after the keeper's pass was never kept at all, so its own GET, which online arrives embedded in the page read, was missing offline. The keeper now hands each request to the worker (`offline-query`, the same route as `offline-api`), the worker claims pages on activation, and opening a database online keeps it two seconds later, once per session. The per-view fallback key is limited to plain `/query` bodies: a dashboard's widgets share a `view_id`, and a chart mark's `pick` request shares the `/chart` path, so a view-level fallback there served one widget's numbers to another and a chart's totals as a pick (an empty row list). Dashboard warming uses `widgetsOf` and `sparkSpecOf` from the dashboard module so the keeper sends byte-identical bodies. Offline checks for anything the worker serves must stop the server, not use the browser's offline switch.

### src/main.ts · the row outbox replays on reconnect
- refers-to: src/main.ts (the `live` watcher next to the `online` listener); src/engine/collections.ts :: `replay`, `queued`
- kind: rationale
- logged: v0.38.0 · 2026-10-17

The outbox replayed on the window `online` event and at start, which covers a device losing its network but not the commoner self-hosted case of the server restarting: the browser stays online, no event fires, and queued creates and edits waited for a reload. Probe p31, with the server process stopped and restarted, showed a row added during the outage still missing on the server 25 s after it was back; replaying when the `live` state returns to `live` fixed it. The same probe rechecked, with the server really stopped, what v0.11.0 to v0.26.0 had verified with a browser offline switch: the app starts, a kept page shows its text and picture and accepts typing, the typed text reaches the server once it is back (the sync socket reconnects on its own), and a kept database shows its rows.

### src/engine/collections.ts · the row outbox belongs to one account
- refers-to: src/engine/collections.ts :: `setOwner`, `key`; src/engine/index.ts (the `me` watcher); src/main.ts (`LAST_USER_KEY`); src/engine/doc/docs.ts :: `wipeLocalData`
- kind: rationale
- logged: v0.39.0 · 2026-10-17

The outbox, create and delete queues sat under fixed `localStorage` keys, so changes queued by one account on a shared browser were replayed under the next account's session after a sign-in. Each key now carries the user id; a legacy unsuffixed queue is moved to the first owner set, the one whose session the app starts with. The app also remembers the last user id and, when a different one starts it, deletes the caches and document databases first, because the session guard only clears them on a 401 and a fresh sign-in from the login page never passes through it. The queues are kept, not wiped, so an account that signs back in still sends what it queued. The legacy unsuffixed queues are deleted instead of adopted when the remembered last user differs from the one starting, since they were written by that last user; with no remembered user (the first start after the upgrade) the starting account takes them, the best guess available. `LAST_USER_KEY` is written only after the wipe finishes, so a tab closed mid-wipe wipes again at the next start.

### public/sw.js · view fallback and message origins
- refers-to: public/sw.js :: `queryKey` (`canonical`), `queryRead`, the `offline-api`, `offline-query` and `offline-files` message filters
- kind: rationale
- logged: v0.39.0 · 2026-10-17

The per-view fallback key was overwritten by every plain `/query` answer, so the last calendar month or a sort override seen online became what every other request for that view got offline. Only a canonical body (no range and no sorts, the keeper's own warm request) now writes it; any plain body may still read it. The message handlers resolved page-supplied URLs against the worker's origin but only tested the path, so an absolute URL to another origin with a matching path would be fetched; each filter now requires the worker's own origin.

### src/engine/i18n.ts · own-property lookups
- refers-to: src/engine/i18n.ts :: `resolveLang`, `setLang`
- kind: rationale
- logged: v0.39.0 · 2026-10-17

The language tag comes from a saved preference or the browser; `ALIASES[tag]` and `loaders[next]` on plain objects would return `Object.prototype` members for tags such as `constructor`. `Object.hasOwn` limits both to the listed languages.

### src/ui/session.ts · the offline shell's workspace
- refers-to: src/ui/session.ts :: `mountSession` (`rememberWorkspace(startWs)`)
- kind: bug-context
- logged: v0.11.0 · 2026-10-09

The server renders `/offline-shell` for the workspace in the remembered-workspace cookie. That cookie was only written on a switch, so after a deep link into another workspace the cached shell showed the wrong tree. The workspace the app boots into is now remembered at start.

### src/ui/offline.ts, public/sw.js · row pages and pictures offline
- refers-to: src/ui/offline.ts :: `keep`, `keepRows`, `fileRefs`, `postFiles`, `startOffline` (boot row); public/sw.js :: `API_READS` (`pages/<id>`), `offline-files`
- kind: explainer
- logged: v0.11.1 · 2026-10-09

A row page is not in the tree, so offline it can only be drawn from the cached page detail (`GET /api/pages/<id>`, which the engine's `loadDetail` reads on every page route) and the cached row read (`GET /api/rows/<id>`). Keeping a database caches both for every row in its first window and syncs each row's body into IndexedDB, four at a time. A row page opened from a server-rendered boot never fetches `/api/rows/<id>` (the boot carries it), so the worker is asked to fetch it once at start. After a kept page syncs, every `f:` attribute in its document, its cover and its icon are sent to the worker, which fetches the files it does not hold yet into `formmd-files`.

## Phase 12 · visual data views

### src/extensions/database/views/chartkinds.ts · the eight new kinds
- refers-to: src/extensions/database/views/chartkinds.ts :: `drawExtra`, `layoutCalendar`, `paintCells`, `heatLevel`; chartdraw.ts :: the helper exports
- kind: explainer
- logged: v0.12.0 · 2026-10-09

`drawChart` hands the new kinds to `drawExtra` before anything else, so the original bar, line and donut code is untouched. Everything is keyed SVG on the same six layers, so a live update moves marks. Heat levels run 0 to 5: linear divides by the maximum, quantile cuts the non-zero values at the 20th to 80th percentiles (with few distinct values fewer levels appear). The dot plot places dots by value along a band and steps them up and down until they do not overlap; overflow is placed by a hash of the row id rather than at random, so a redraw does not shuffle them. `layoutCalendar` is shared by the calendar chart kind and the heatmap view.

### src/extensions/database/views/heatmap.ts · fixed heights
- refers-to: src/extensions/database/views/heatmap.ts :: `heatmapView`, `paintBar`; src/styles/viz.css :: `.heatv[data-domain]`
- kind: warning-context
- logged: v0.12.0 · 2026-10-09

The frame height comes from CSS by layout (`data-domain`, set synchronously at mount) and the grid is fitted into it, so the view never grows when data arrives; before the first answer the cells paint at level 0. Ranges move by whole domains and are fetched with a `between` filter on the date property, so a year's request carries only that year.

### src/extensions/database/views/graph.ts and graphsim.ts · the graph
- refers-to: graph.ts :: `build`, `startSim`, `link`, `unlink`, `paintSide`; graphsim.ts :: `charge`, `linkForce`, `clusterForce`
- kind: explainer
- logged: v0.12.0 · 2026-10-09

Rows are fetched in pages of 1,000 up to 2,000 with `no_groups`; relation values become edges deduplicated per unordered pair (two-way relations would otherwise draw twice). The worker runs Barnes-Hut (theta 0.9) with d3-style link strength (one over the smaller degree) and a weak pull to each colour group's centroid, ticking in 14 ms slices and posting positions as transferable arrays; positions survive a rebuild by id. Edits go through `collections.setValue` and are mirrored in `rowVals` and the edge list at once, then the session's live update reloads the graph with a small reheat. Node colours are read from a probe element with `data-color`, converting `color(srgb …)` results for the canvas.

### src/extensions/database/views/dashboard.ts and board.ts · upgrades
- refers-to: dashboard.ts :: `Widget.size`, `Widget.kpi`, `startMove`; kpi.ts :: `drawKpi`; board.ts :: `fetchSummary`, `startEdit`, `layout().limits/collapsed/total/progress`
- kind: explainer
- logged: v0.12.0 · 2026-10-09

Old widgets with `wide` read as size `l`. A KPI asks the chart endpoint twice (the number and a line by the trend period) and shows the last 24 buckets. Board totals and progress come from one chart request each over the board's own grouping (and sub-grouping as series), so they count every row, not just the loaded window. Card editing is a mode: the pencil starts it, a click on a property inside an editing card opens that property's editor instead of the row page, and a press outside the card (but not inside an editor in the overlay root) ends it.

### src/extensions/database/views/dashboard.ts, graph.ts · keyboard and screen readers
- refers-to: dashboard.ts :: `keyMove`, `refocus`, `announce`; graph.ts :: `paintList`, `showNode`, `a11y`; styles/viz.css :: `.graphv__a11y`
- kind: rationale
- logged: v0.36.0 · 2026-10-17

Arrow keys on a widget's grip swap it with its neighbour through the same `save` the menu uses. `reconcile` moves the card with `insertBefore`, and moving a focused node drops focus, so `paint` puts focus back on the moved card's grip; a polite live region says the new position. The graph's canvas cannot be read, so a list of its rows (built with `reconcile`, keyed by row id, after every `build`) sits after it. It is clipped while focus is outside it rather than `hidden`, so screen readers still read it, and appears as a corner panel on `:focus-within`, so a sighted keyboard user never has focus on something invisible. A row's button calls `showNode`, the code the Find box's Enter used, now shared. Five neighbour names are spelled out, then "and N more", to keep each entry short.

## Phase 13 · context for agents

### src/ui/account-agents.ts · client snippets
- refers-to: account-agents.ts :: `snippets`
- kind: explainer
- logged: v0.13.0 · 2026-10-09

The labels are product and file names, so they are not run through `t()`. Codex starts MCP servers as commands, so its snippet runs the binary's `form-md mcp` bridge with `--url` set to this origin (the bridge appends `/mcp`) and a placeholder token, written as TOML that `~/.codex/config.toml` accepts as is. The snippets are built once when the tab renders and never change, so the tab cannot shift.

## Phase 14 · agent harness, commands, languages

### src/ui/agent-dock.ts · the bottom panel
- refers-to: src/ui/agent-dock.ts :: `mountAgentDock`, `TermView`, `buildChat`; src/extensions/harness/index.ts
- kind: explainer
- logged: v0.14.0 · 2026-10-16

Loaded on first use as its own chunk with xterm.js 6 and its fit addon (both MIT). The panel is a fixed sheet at `--z-chrome` with a top grip whose height is kept in localStorage (wrapped in try). Each terminal tab owns one xterm and one WebSocket; panes are created once and toggled with `hidden`, never rebuilt, and fit only while visible. Theme colours are read from the CSS tokens and reapplied when the theme preference changes. The chat renders Markdown through the same `render` and `reconcile` pair the agent pane uses, so a growing reply patches its existing nodes instead of replacing them. Keys: Mod+J panel, Mod+Alt+J Claude Code, Mod+Alt+K chat (Mod+Shift+J stays the built-in agent).

### src/ui/agent-dock.ts · xterm under the CSP, font, first open, restart
- refers-to: src/ui/agent-dock.ts :: `TermView.mount`, `adoptStyle`, `syncSheet`, `monoFamily`, `ready`, `show`, `restart`
- kind: rationale
- logged: v0.31.0 · 2026-10-17

xterm 6's DOM renderer and viewport write cell sizes, theme colours and scrollbar colours into `<style>` elements, which the app's `style-src 'self'` refuses, so terminals drew with the wrong cells since v0.14.0. Rather than loosening the policy or adding the WebGL renderer (which still injects the scrollbar style), each terminal adopts one constructable stylesheet: style elements xterm adds under its pane are detached, watched, and their text copied into the sheet, a CSSOM path the policy does not cover. Their first contents are set synchronously inside `term.open()`, before any observer runs, so each opening still logs about ten refused-style console messages; every later update is silent and nothing is lost. The font list is resolved from `--font-mono` because xterm measures cells on a canvas, where `var()` is not a font, and it measured 10px sans (11px rows, 1.66px letter spacing) instead; the font load runs in parallel with the first `/api/claude` request so the first measure sees the real font. `show()` opens the panel only once a terminal exists, so the first Mod+J no longer shows an empty panel for a round trip. Restart swaps the pane in place and keeps the tab element, where it used to remove the tab and append a new one at the end.

Since v0.39.4 the copied rules are scoped. The sheet is adopted by the whole document, so any `<style>` that appeared under a pane applied page-wide; xterm's own rules happen to name its owner class, but nothing enforced that. `syncSheet` parses the collected text into a scratch sheet and nests each style rule under `.dock__term[data-view="N"]`, a number unique to the view (a restart in place makes a new view for the same terminal id, so the id would not be unique). At-rules such as the cursor-blink `@keyframes` stay top level, where nesting would drop them. Checked in Chrome: a `body` rule injected into a pane no longer reached the body, a `.xterm-rows` rule still applied, all 791 adopted style rules carried the prefix, and the throttled dock pass (CPU 4x, 500 kbps, 400 ms) matched the unchanged build in shifts, removals, timing and screenshots.

### src/ui/agent-dock.ts · terminal colours follow `data-theme`
- refers-to: src/ui/agent-dock.ts :: `mountAgentDock` (the theme observer), `TermView.restyle`, `themeOf`; src/ui/theme.ts
- kind: rationale
- logged: v0.31.1 · 2026-10-17

Terminals re-read their colours when the root's `data-theme` attribute changes, not when `themePref` changes. With "Match the system theme" a system switch changes the resolved theme without touching the preference, so watching the preference left a dark terminal on a light page. The observer callback runs as a microtask before the next paint, so the terminal changes in the same frame as the page; `src/extensions/database/views/graph.ts` watches the same attribute for the same reason.

### src/engine/commands.ts · the catalogue
- refers-to: src/engine/commands.ts :: `Command`, `invoke`, `catalogue`; src/engine/index.ts :: `dispatch` (command.run)
- kind: explainer
- logged: v0.14.0 · 2026-10-16

The command shape follows formvis's: description, a "use when" sentence, typed parameters with enum and range, mutates and risky. `invoke` checks parameters before running, returns `{ ok: true }` when a command returns nothing, and is what agents reach: the harness extension mirrors `catalogue()` to the server 400 ms after any registration change, and a `command.run` event from the server runs `invoke` and posts the result or error back with its call id. Existing commands carry only id, label and group until Phase 15 fills in their metadata.

### src/engine/i18n.ts · 26 languages
- refers-to: src/engine/i18n.ts :: `LANGUAGES`, `resolveLang`; internal/i18n/i18n.go :: `Resolve`
- kind: explainer
- logged: v0.14.0 · 2026-10-16

Catalogue folders are named by BCP 47 tag (`es-MX`, `pt-BR`, `zh-Hant`), and each holds `01.json` to `08.json` (the interface strings in the canonical order) plus `server.json` (mail and notification strings the Go side reads from the same embedded tree). Resolution is exact tag first, then aliases (zh-TW and zh-HK to zh-Hant, zh-CN to zh-Hans, pt to pt-BR, es-419 to es-MX, no and nn to nb), then the base language; Go and TypeScript use the same table. German keeps its older 01 to 10 files and gains 11.json with this release's strings.

### src/canvas/ · removed
- refers-to: src/canvas/engine/geometry.ts, src/canvas/engine/ops.ts (deleted in v0.14.1)
- kind: removed-code
- logged: v0.14.1 · 2026-10-16

A DOM-free vector engine (anchors with handles, per-point corner rounding, shapes and path operations) landed in v0.14.0 as the start of an Illustrator-parity canvas. Your instruction of 2026-10-16: Illustrator parity does not belong in form.md, so the engine is deleted and no canvas page, block or route exists. The code stays reachable in history at commit `2680de4` if a vector surface is ever wanted for another form* product.

## Phase 15 · creative wishlist

### src/ui/page-style.ts · the Style menu
- refers-to: src/ui/page-style.ts :: `effectiveStyle`, `applyStyle`, `openStyleMenu`, `registerStyleCommands`; src/styles/page-style.css
- kind: explainer
- logged: v0.15.0 · 2026-10-16

Attributes go on the html element, not the page view, because the server writes them there for the first frame and the client must agree with it; a default value removes the attribute. Width works through `--doc-w` on `.view--page`, size through `--doc-text`, spacing through line height and `--block-gap`, alignment on paragraphs, list items and quotes only, so tables, code and databases keep their own alignment. The Style entry opens its own menu on the next frame from the page menu's anchor, after the page menu has closed.

### src/extensions/writing · focus mode and counts
- refers-to: src/extensions/writing/index.ts :: `countWords`, `countsFor`, `setup`; src/extensions/blocks/triggers.ts :: `slashAtStartOnly`; index.html (boot script)
- kind: explainer
- logged: v0.15.0 · 2026-10-16

Focus mode is `data-focus` on the html element, restored by the boot script before first paint, and hides chrome with `visibility` so nothing reflows; the app grid closes the sidebar column the same way the collapsed sidebar does. Counts run on the next animation frame after input in the document body or any selection change, never on a timer, and read the ProseMirror document rather than the DOM; `Intl.Segmenter` gives word counts for languages without spaces. The counter is hidden while the terminal panel is open so the two never overlap. The slash rule reads localStorage each time, so the setting applies without reloading the editor.

### src/extensions/blocks/annotate.ts · image annotations
- refers-to: src/extensions/blocks/annotate.ts :: `parseMarks`, `markNodes`, `paintMarks`, `startAnnotating`; nodeviews.ts :: `MediaView.annotate`; engine/doc/schema.ts (image `marks`, data-marks); internal/doc/marks.go :: `marksSVG`
- kind: explainer
- logged: v0.16.0 · 2026-10-16

Marks are stored in the picture's own pixel space (`w`, `h` at the time of the first mark) and drawn in an SVG with that viewBox and `xMidYMid meet`, laid over the image inside the frame, so they scale with any display width and stay put after resizing. Arrow heads are polygons computed from the line, not SVG markers, so there are no ids to collide between images. Each finished shape is one `setNodeMarkup` transaction; the overlay is repainted only when the JSON of the items changes, and the drawing in progress lives in its own nodes until it is committed. Colours are a fixed set of six-digit hex values, checked again by the Go renderer before they reach HTML, and text is escaped on both sides. Freehand strokes over 200 points are thinned by half before saving.

### src/extensions/blocks/align.ts · block alignment
- refers-to: src/extensions/blocks/align.ts :: `setAlign`; engine/doc/schema.ts :: `ALIGNS`, `alignOf`, `alignAttr`; internal/doc/marks.go :: `align`; styles/page-style.css
- kind: explainer
- logged: v0.16.0 · 2026-10-16

Alignment is an attribute on paragraphs and headings only, empty for left so existing documents are unchanged. Its CSS rule sits after the page-style rules with the same specificity, so a block's own alignment wins over the page's. `right` maps to `text-align: end` so right-to-left text behaves.

### src/ui/page-style.ts and src/extensions/writing · custom fonts and typewriter scrolling
- refers-to: page-style.ts :: `syncFonts`, `uploadFont`; writing/index.ts :: `centerCaret`; styles/writing.css; index.html (boot script)
- kind: explainer
- logged: v0.17.0 · 2026-10-16

`syncFonts` adds the workspace's fonts stylesheet as a link only when a page uses a custom font, with a version made from the font ids so a new upload is fetched, and removes the older link once the new one has loaded so nothing unstyles in between. Typewriter scrolling acts only after keystrokes in the document (a pointer press clears the flag), finds the nearest scrolling ancestor and moves it so the caret sits at 45 percent; focus mode adds 55vh of space under the page so the last line can reach that point. `data-typewriter-off` is restored by the boot script like `data-focus`.

### src/extensions/blocks/board.ts · board block
- refers-to: src/extensions/blocks/board.ts :: `BoardView`, `newBoard`, `readBoard`; styles/board.css; site/site.css
- kind: explainer
- logged: v0.18.0 · 2026-10-16

The board is an SVG with a fixed viewBox, so its height is known from the first frame and pictures arriving later never move the page. `sync` keeps one foreignObject per card id and only rebuilds a card when its kind, picture, link or editability changes; a note's text is set in place unless the note has focus, and cards are reordered only when their order differs, so a focused note is never moved. A drag moves the card's attributes directly and commits once on release (the moved card goes last, which brings it to the front); `update` skips syncing while a drag is running. `pointerdown` calls preventDefault, because the block is draggable and the browser's native drag would otherwise cancel the pointer stream after a few pixels; a press that does not move focuses a note's text, and a link card still opens on click. New cards go to the first free spot on a 40-unit grid, or under everything with the board grown to fit. Pictures dropped on the board upload as page attachments and keep their aspect ratio. The public site stylesheet imports `board.css` and `annotate.css`.

### src/ui/offline.ts and public/sw.js · recent pages and kept rows
- refers-to: src/ui/offline.ts :: `keep`, `recentPages`, `refreshKept`, `registerOfflineCommands`; public/sw.js :: `apiRead`
- kind: explainer
- logged: v0.19.0 · 2026-10-16

Keeping a database caches `?view=<id>&rows=1000` for every table and list view (and `?view=&rows=1000` for the default) and syncs the row pages of the first 1000 rows of the first view. Offline, `apiRead` answers a collection read that has no `rows` parameter from the `rows=1000` copy first, then the exact URL, then any variant. `refreshKept` adds the 20 most recent pages from `/api/recent` unless `formmd.offlineRecentOff` is set; the recent list comes from the server, so pages visited on another device are kept too. The `offline.recent` command registers when the service worker starts, so it is absent in browsers without one.

### src/extensions/blocks/handle.ts · phone block toolbar
- refers-to: src/extensions/blocks/handle.ts :: `BlockHandle` (phone bar, `caretBlock`, `moveCaretBlock`, `menu(anchor)`); styles/editor.css (.block-phonebar)
- kind: explainer
- logged: v0.21.0 · 2026-10-16

The hover handle is useless on touch, so at 720px and below a toolbar shows while the editor has focus, chosen purely in CSS (`:has(.ProseMirror-focused)`), so no script toggles it. It sits at `--keyboard-h` from the bottom, the height the visualViewport says the on-screen keyboard covers. Buttons prevent default on pointerdown so the editor keeps focus and the keyboard stays up; each click first points the handle's target at the block holding the cursor. The bar stays while its own menu is open (`:has([aria-expanded="true"])`), because the menu stays anchored to the button. The focused editor gets bottom padding the height of the bar so the last line is never under it.

### src/extensions/blocks/editor.ts · block presence for the agent guard
- refers-to: src/extensions/blocks/editor.ts :: `plugins` (formmd-presence); internal/doc/blocks.go :: `personIn`
- kind: explainer
- logged: v0.21.0 · 2026-10-16

The editor writes the ids of every block around the cursor (outermost first) into its awareness state as `blocks`, only when they change, and clears them on blur; a focus event dispatches an empty transaction so the ids come back. The server reads them to refuse agent replaces and deletes of a block a person is in.

### src/ui/page-style.ts · renaming and removing fonts
- refers-to: src/ui/page-style.ts :: `renameFont`, `removeFont`, `saveFonts`
- kind: explainer
- logged: v0.21.0 · 2026-10-16

The entries show only for admins and only when the page uses an uploaded font, so they always act on a font the person can see. Removing switches this page to Sans and drops the font from the workspace default style before saving the shorter list; other pages that used it fall back to the default face because the generated stylesheet no longer has a rule for it.

### src/engine/collections.ts · offline row outbox
- refers-to: src/engine/collections.ts :: `flushRow`, `queue`, `overlay`, `replay`, `resync`; src/main.ts (onQueued notice, online listener)
- kind: explainer
- logged: v0.22.0 · 2026-10-16

A row update that fails with status 0 (no network) is merged into `formmd.rowOutbox` in localStorage per row id, values last-wins, keeping the oldest time for ordering. `QuerySession.absorb` passes every row through `overlay`, so cached views and offline reloads show the queued values; the outbox is cached in memory after the first read. `replay` runs on the online event, on resync (before views refresh, so a refresh cannot paint old values) and at start when online; it stops at the first network failure and drops an entry the server refuses, reporting it. Only cell and title edits are queued: creating, deleting and moving rows still need the server because they change row ids and order.

### src/extensions/blocks/annotate.ts · picking single marks
- refers-to: src/extensions/blocks/annotate.ts :: `startAnnotating` (bbox, hit, showSel, remove, editText, keys)
- kind: explainer
- logged: v0.23.0 · 2026-10-16

Hit testing uses each mark's bounding box widened by three stroke widths, newest first, which is generous for thin lines and good enough for annotations; a text mark's box is estimated from its length and font size. A drag rewrites the picked mark's points from a copy taken at press time and repaints without writing; the write happens once on release. The Delete and Backspace listener is on the document in the capture phase and stops the event, because the picture sits inside the editor and the editor would otherwise delete the image block. The dashed outline is a separate rect with class `annot__selected`, appended after each repaint and removed when annotating stops.

### src/extensions/database/views/timeline.ts · bar windowing
- refers-to: src/extensions/database/views/timeline.ts :: `rowWindow`, `paint` (shown), scroll listener
- kind: explainer
- logged: v0.23.0 · 2026-10-16

Above 200 rows only bars within 30 rows of the visible area are reconciled into the canvas; `reconcile` keys them by row id, so bars that stay in the window are not touched when it moves. The scroll handler compares the window on an animation frame and repaints only when it changes. The canvas height and dependency paths still use every row.

### src/extensions/database/views/list.ts · loading on scroll
- refers-to: src/extensions/database/views/list.ts :: `autoMore`
- kind: explainer
- logged: v0.24.0 · 2026-10-16

The observer only presses the existing Load more button, so loading has one code path; it watches with a 600 px margin so the next rows usually arrive before the reader reaches the end, and it is disconnected in `destroy`. Inline lists inside pages are left alone so a page with an inline database does not load its rows just by scrolling past it.

### src/engine/collections.ts · offline row deletes
- refers-to: src/engine/collections.ts :: `deleteRows`, `deletes`, `deletedLocally`, `QuerySession.hide`, `apply`, `replay`
- kind: explainer
- logged: v0.25.0 · 2026-10-16

A delete that fails for lack of a network records each row id with its database in `formmd.rowDeletes`, drops any queued edits for those rows, and hides them from every open session (`hide` removes them from slots, groups and lanes and lowers the total). `apply` filters them out of every later result, which covers cached reads after an offline reload. `replay` sends deletes after edits, grouped per database, and refreshes those views.

### src/engine/collections.ts · offline row creates
- refers-to: src/engine/collections.ts :: `createRow`, `creates`, `localRows`, `newRowId`, `localRow`, `QuerySession.insertLocal`, `apply`, `replay`
- kind: explainer
- logged: v0.26.0 · 2026-10-16

`newRowId` encodes 16 random bytes as lowercase base32 without padding, 26 characters, which is what the server's `ids.New` makes. The new row is inserted into open sessions on a zero-delay timer, after the promise has resolved, because views record which row to focus in the promise's `then` and focus it on the next paint. `apply` appends this database's local rows when the result reaches the end of the list, so they appear after an offline reload; offline edits to them go through the normal edit outbox and the overlay. A delete of a row that exists only in `formmd.rowCreates` removes it there instead of queuing a delete.

### src/ui/account.ts · Writing section
- refers-to: src/ui/account.ts :: profileTab (Writing); src/extensions/blocks/triggers.ts :: `SLASH_KEY`
- kind: explainer
- logged: v0.22.0 · 2026-10-16

The toggle reads and writes the same localStorage key the editor reads at each slash, so it applies at once without a reload and matches the editor.slashAtStart command.

### src/engine/commands.ts · mutates defaults to true
- refers-to: src/engine/commands.ts :: `catalogue`; internal/claudecode/module.go (read-only check)
- kind: rationale
- logged: v0.19.0 · 2026-10-16

The catalogue tells the server whether a command changes data, and a read-only MCP credential may run only commands marked as not changing anything. A command that forgets the flag is therefore listed as changing data, so the mistake fails closed. Every command declares the flag explicitly anyway; a command that only changes the view or local preferences of this window says `mutates: false`.

### src/styles/shell.css and editor.css · phone width
- refers-to: src/styles/shell.css (max-width 767px block); src/styles/editor.css :: `.code`
- kind: rationale
- logged: v0.19.0 · 2026-10-16

At phone width the terminal and shortcut buttons are hidden (neither is usable on a touch screen) and the mode button keeps only its icon, its aria-label still naming the mode, so the page title in the top bar keeps room. `.code` is a grid whose implicit column sized to the longest line; `minmax(0, 1fr)` bounds it so `.code__pre` scrolls instead of widening the block.

### src/extensions/database/views/board.ts · per-card bars and rings
- refers-to: src/extensions/database/views/board.ts :: `paintCard`, `isMeter`; src/styles/database.css :: `.card__prop[data-meter]`
- kind: rationale
- logged: v0.41.0 · 2026-10-19

Cards already drew a number property shown as a bar or ring, but only while it had a value, so typing the first value made the card taller. A bar or ring row now stays on the card when empty, marked `data-meter`, with 16px (the ring height) reserved. Existing card nodes are reconciled and the attribute is set in place; nothing is rebuilt.

### src/ui/library-view.ts · the Graph tab
- refers-to: src/ui/library-view.ts :: `VIEWS`, `graphBox`, `syncGraph`; src/ui/graph-view.ts :: `mountGraph`, `openGraph`; src/engine/router.ts :: `LibraryView`, `LIBRARY_VIEWS`; src/styles/pages.css :: `.lib-graph`
- kind: rationale
- logged: v0.41.0 · 2026-10-19

`mountGraph` holds what the dialog did and returns a dispose that disconnects the ResizeObserver and cancels the frame; the dialog and the tab both call it. The tab imports graph-view only when entered, so the start bundle does not grow, and disposes on leaving the tab, the library route or the workspace. `.lib-graph` reserves the canvas height plus its bar before data arrives, so loading does not shift the page.

### src/ui/search.ts and src/ui/admin-ai.ts · Include the web
- refers-to: src/ui/search.ts (answer filter row, sources list); src/ui/admin-ai.ts (Web search for answers); src/engine/client.ts :: `ai.answer`, `ai.answerOptions`
- kind: rationale
- logged: v0.41.0 · 2026-10-19

The "Include the web" chip sits in the existing filter row and stays hidden unless `GET /api/ai/answer/options` says the web is on, so instances without a service see nothing new. It is off by default and resets after each question, so the web is chosen per question, never sticky. Web sources render as external links with their URL beside the number. The admin section keeps all fields visible whatever the mode, so changing the mode never moves the form.

### src/extensions/blocks/diagram.ts · Mermaid in dark appearance
- refers-to: src/extensions/blocks/diagram.ts :: `drawDiagram`, `darkAppearance`, `watchAppearance`; src/extensions/blocks/nodeviews.ts :: `CodeView.setLight`, `ensureDark`, `apply`; src/sandbox/main.ts (theme per render); src/styles/editor.css :: `.diagram__img:not([data-tone="dark"])`
- kind: rationale
- logged: v0.41.0 · 2026-10-19

Inverting a neutral drawing turned a diagram's own colours into their opposites. The sandbox renders Mermaid's dark theme on request, re-initialised with the full config each render, and the dark drawing is cached apart from the light one and never stored, so the document and public pages keep the light svg. The new image is decoded off the page and then src, data-tone, width and height are written in one task, so there is no blank or resize; the box keeps a min-block-size while the first dark drawing is pending. The filter still applies to any image not marked dark, which is what published pages show.

### src/extensions/blocks/synced.ts · the nested editor's chrome
- refers-to: src/extensions/blocks/synced.ts :: `SyncedView.bind`; src/extensions/blocks/editor.ts :: `EditorHost.nested`; src/extensions/blocks/context.ts :: `NestedKit`; src/extensions/blocks/cursors.ts; src/extensions/blocks/formatbar.ts :: `detach`
- kind: rationale
- logged: v0.41.0 · 2026-10-19

The nested editor reuses the host's trigger plugin, AI plugins, format bar and upload plugin rather than forks of them. The host builds them through `EditorCtx.nested()` because importing the format bar or triggers into synced.ts closes a cycle (nodeviews, synced, format bar, catalog, board) whose evaluation order left a module undefined at boot. The nested context's `doc()` is the source page's session document, so a block created inside the synced block lands in the document that owns it. `FormatBar.attach` registers its window mouseup listener with an AbortController and `detach` aborts it, so destroyed synced blocks leave no listener behind. There is no block handle inside: the handle is bound to the main editor's wrapper and hover.

### src/ui/tree.ts · long-press drag and the Databases section
- refers-to: src/ui/tree.ts :: `mountTree` pointerdown handler, `splitMode`, `renderRoot`, `renderDatabases`
- kind: rationale
- logged: v0.41.0 · 2026-10-19

Touch keeps native scrolling: a press arms a 450 ms timer that 8 px of movement cancels, and only when it fires does the mouse drag path begin. The non-passive touchmove and the capturing contextmenu listener exist only while a drag is live, so taps and scrolls are unaffected; on Android a long hold on a row now drags instead of opening the browser menu, and the row's own menu button still opens the row menu. In two-sections mode root databases leave Documents for the Databases section, and in one-tree mode the section hides, so no database is listed twice; only a mode change re-renders the two lists.

### src/ui/icons.ts · formlit's icon art
- refers-to: src/ui/icons.ts :: `PATHS`
- kind: rationale
- logged: v0.42.0 · 2026-10-19

The owner wants one icon language across form.md and formlit, so every entry with a formlit counterpart is redrawn from formlit's art under form.md's name and no caller changes. Shapes formlit draws filled carry fill="currentColor" with stroke none; other strokes use the wrapper's 1.75 except the small plus in search, whose look depends on its thinner width. agent, agent-chat, send and ai-chip come from formlit; sparkle is removed (banned), and formlit's wand and burst icons were not copied, so the only star left is the single rating star. Filled icons are heavier than stroked ones; that matches formlit.

### src/ui/agent-dock.ts · the dock as formlit has it
- refers-to: src/ui/agent-dock.ts (header, ask bar, select, TermView.reattach, the chat); src/styles/dock.css; src/engine/keymap.ts :: `KeyContext.only`; src/ui/keyboard.ts (data-own-keys)
- kind: rationale
- logged: v0.42.0 · 2026-10-19

The dock follows formlit's three-row grid (header, stage, ask bar) and its tab and chat proportions in form.md tokens. One ask bar serves the Claude Code tab and the chat, so the chat has no composer of its own. Restart reuses the same xterm and pane: the server's restart keeps the subscription, the client resets the terminal and re-attaches. App shortcuts run in a window capture listener, so a stage handler cannot stop them; a `data-own-keys` allow-list and `KeyContext.only` let the stage pass only the dock toggle. The Chat tab is appended and moved to the end when shown so opening it never shifts tabs on screen; the status text has a minimum width so "Claude is working..." never pushes the meter. Not yet like formlit: the dock overlays the page instead of taking a row of the shell, the client does not yet listen to `/api/claude/chat/events` after a reload, and there is no Claude section in settings.

### src/ui/home-view.ts and index.html · home, topbar and first paint
- refers-to: web/index.html (boot script order, data-home, data-owner, data-dev, home markup); src/ui/home-view.ts; src/main.ts (syncDev, the Claude probe); src/ui/status.ts; src/styles/shell.css, pages.css, doc.css
- kind: rationale
- logged: v0.42.0 · 2026-10-19

The boot JSON and a small inline script now sit before `#app`, so `data-home`, `data-owner` and `data-dev` exist before the first frame; the home's two states are CSS toggled by `data-home`, and the only runtime writes are the recents list children inside reserved rows and the offline notice's hidden flag. Recents load in a lazy chunk at boot and on workspace change, so they can be stale until the next of those. The Claude Code button shows from the owner role at first paint and hides only if the harness probe says it is off. The account switch sets `data-dev` directly as well as storing it, because the storage event reaches other tabs only. The theme boot defaults to auto, so a new browser follows the system from the first frame.

### src/ui/disclosure.ts · dialogs for everyday people
- refers-to: src/ui/disclosure.ts; src/ui/publish-dialog.ts; src/ui/share-dialog.ts; src/ui/account.ts (Advanced); src/ui/admin.ts (Advanced: network); src/ui/auth.ts (Show password)
- kind: rationale
- logged: v0.42.0 · 2026-10-19

Every disclosure is a native details element and only the summary chevron moves, by transform, so opening one never moves what sits above it. The publish dialog stacks its intro and the live address in one grid cell toggled by `data-off`, so publishing changes no height, and its sticky Publish bar covers the body padding so content never shows beneath it. The password toggle wraps the input in a same-size span in the same task as the form mounts, with end padding in CSS, so nothing shifts. The Advanced panels render on first open, which also defers their token and agent fetches.

### src/extensions/database · tables everyday people can find
- refers-to: src/extensions/database/index.ts (createCollection title, focusNew), surface.ts (empty hint, More menu, Rules), props.ts :: `typeEntries`, propedit.ts; src/extensions/blocks/catalog.ts; src/styles/editor.css, database.css
- kind: rationale
- logged: v0.42.0 · 2026-10-19

The empty-table hint is a sibling after the stage, so it appears below content and nothing above moves, and its hidden flag is written only when the state changes. Group, Properties, Layout and Rules share the existing More menu instead of a new button, and panels anchor to it. Toolbar buttons keep a fixed 28px height so labels widen the bar without changing its height. The common nine property types come first and the rest follow "More types" under plain names; type ids are untouched so stored schemas and the API are unchanged.

## Phase 18 · shell {#p18-shell}

### index.html · flat shell and the sidebar in the first frame
- refers-to: web/index.html (body boot script, `#sidebar`, `.topbar__nav`, `act-edited`, `act-presence`, `help-btn`, `view-home`); src/styles/shell.css :: `.sidebar`, `.sheet`, `.sb-item`, `:root[data-route] .sb-item[data-route-link]`
- kind: rationale
- logged: v0.44.0 · 2026-10-20

The main area is full-bleed beside the sidebar: no margin, border, radius or shadow on `.sheet`, and the sidebar carries `--surface` with a 1px end border, so the dotted ground in base.css only shows behind the sign-in card. Every sidebar row (Search, Home, Library, Templates, Import, Trash, Settings, Help) and the header buttons (collapse, Inbox, New) are static markup, so they paint in the first frame and the server localizes their text; scripts only bind them. Home, Library and Trash highlight from the root `data-route` in CSS, so the active row is right before any script runs. The body boot script, which already ran before `#app`, now also sets `data-daypart` (the greeting is three spans toggled by CSS, never a text swap), `data-desktop` from `me.instance.desktop`, and `--sidebar-w` from `formmd.sidebarWidth` when it is within 220 to 480. `status.ts` still binds `live-dot` and `version-text`: the dot sits next to the workspace switcher and the version text is kept for assistive tech and the Help menu. The theme control left the sidebar; theme choice belongs to Settings. The user chip stays on the web for the account menu and Sign out and is hidden by CSS under `data-desktop`.

### src/ui/app.ts · back and forward know where they are
- refers-to: src/ui/app.ts :: `navSpan`, `platformAdapter` pushHistory and replaceHistory, popstate listener
- kind: warning-context
- logged: v0.44.0 · 2026-10-20

The browser does not say whether a back or forward entry exists, so every entry the app writes carries `nav`, its position, and the furthest position is kept in sessionStorage under `formmd.navTop` so forward survives a reload. A push truncates forward history, as the browser does. Position 0 is the first entry this tab wrote, so Back never leaves the app. The popstate listener must update the position before `restoreNav`, because route watchers repaint the buttons. The extra `nav` field is ignored by `isNavState`.

### src/ui/shell-extras.ts · the shell's on-demand chunk
- refers-to: src/ui/shell-extras.ts :: `mountHistory`, `mountResize`, `mountSidebarExtras`, `mountCrumbSiblings`, `mountPageMeta`, `mountHelp`; src/main.ts (dynamic import after `mountTopbar`)
- kind: rationale
- logged: v0.44.0 · 2026-10-20

The up-front budget had about 1 KB left, so everything new in the shell that is not needed for the first frame loads as one chunk right after the top bar mounts: `nav.back` and `nav.forward` (Ctrl or Cmd with [ and ], disabled through `enabled` when `navSpan` says there is nowhere to go), the New menu, the Search shortcut hint, Home and Library row clicks, Settings (`prefs.open` when registered, else `account.open`), the resize handle, the breadcrumb sibling menu, the "Edited" time, presence avatars and the Help button. Until the chunk arrives the buttons are inert and back and forward stay disabled, which changes colour only. Resizing writes `--sidebar-w` on the root once per frame, turns the grid transition off through `data-resizing`, and saves on release; a double click resets, arrows step 16px. The sibling menu opens after 400 ms of hover, never takes focus (the editor keeps the caret) and is built while detached, then anchored. Presence reads the awareness states the server stamps with `user` (id, name, tint, avatar) and skips this window and this user; it waits for the page view to open the doc session, retrying for up to ten seconds.

### src/ui/tree.ts · static rows and dropping on Trash
- refers-to: src/ui/tree.ts :: `inboxRow`, `templatesRow`, `importRow`, `trashRow`, `resolveTarget`, `endDrag`, `autoscroll`, `paintWritable`
- kind: explainer
- logged: v0.44.0 · 2026-10-20

The Inbox, Templates, Import and Trash rows are bound by `data-ref` instead of created, and the sections now run Favorites, tag filter, notes, Shared spaces, Shared with me, the root tree, Databases, Tags. The root tree is labelled Private in a personal workspace and Pages otherwise, because in a shared workspace those pages are visible to members and the label must not claim otherwise. Trash lives in the pinned footer, outside the scrolling nav, so `resolveTarget` checks it before the nav bounds; dropping a tree row there trashes it with the usual undo toast. Autoscroll now runs only while the pointer is inside the nav's own band, so hovering Trash does not scroll the tree.

### src/ui/home-view.ts and src/styles/home.css · Home as a start page
- refers-to: src/ui/home-view.ts :: `mountHome`, `patchCards`, `handFiles`; web/index.html (`view-home` markup); src/styles/home.css; src/styles/doc.css (`data-when`, `.home-offline`)
- kind: warning-context
- logged: v0.44.0 · 2026-10-20

Home is static markup with skeleton cards: six in the one-row Recently visited grid (later rows have zero height and are clipped) and ten in the Templates grid. `patchCards` puts each real card in a skeleton's place and drops the leftovers, so the grids keep their height while data arrives; the category chips row has a reserved height and scrolls sideways instead of wrapping. The template catalogue and previews come from templates-dialog.ts (`loadTemplateCatalog`, `mountTemplatePreview`, `openTemplates`), imported on first need; previews mount when a card nears the viewport. Dropped, chosen or pasted material goes to the existing import dialog: transfer-dialog.ts has no files parameter, so the files are put on its file input through a DataTransfer and a change event, the same path a person picking files takes. The drop area and Templates section are hidden for read-only access, which can move content for viewers once at load.

## Phase 18 · preferences {#p18-prefs}

### index.html · pre-paint appearance
- refers-to: index.html :: inline `<script>` in head; src/engine/prefs.ts :: `PREF_KEYS.look`, `accentTone`; src/ui/theme.ts :: `paintLook`
- kind: warning-context
- logged: v0.44.0 · 2026-10-20

The head script now also reads `formmd.look` (JSON: dark, light, accent, density, contrast, at) and writes `data-dark-theme`, `data-light-theme`, `data-density`, `data-contrast-pref`, `data-contrast="high"` (resolved against `prefers-contrast: more`), `data-accent` (the accent's tone) and the `--accent-pick` custom property through the CSSOM, which the `style-src 'self'` policy allows where a style attribute would be refused. Each block has its own try, so a corrupt look value never skips the sidebar and tab attributes. If the html tag carries `data-look` (the account's appearance as JSON, written by the server) and its `at` is newer than the local one, it wins, including its light, dark or system mode, so another device paints the account's choice from the first frame. The tone formula and the 132 threshold must stay identical to `accentTone` in prefs.ts, and the attribute values must match what `paintLook` writes (always the normalised id, never absent), or the first module run changes an attribute. The meta theme colour is set for light here and refined from `--ground` by theme.ts. The CSP hash is computed from the built file at server start, so editing the script needs no server change.

### src/ui/theme/tokens.css · theme profiles, contrast, accent and density
- refers-to: src/ui/theme/tokens.css :: `[data-dark-theme]`, `[data-light-theme]`, `[data-contrast="high"]`, `[data-accent]`, `[data-density]` rules, `.theme-swatch`
- kind: explainer
- logged: v0.44.0 · 2026-10-20

Profiles keep form.md's token names and only set the colour tokens (ground, sheet, surface, surface-2, border, border-strong, text, text-muted, text-faint; the high contrast ones also hover and active); everything else falls through from the base dark or light block. A dark profile applies only while `data-theme="dark"` and a light one only while light, so following the system keeps working with two separate choices. The palettes are formdeck's theme profiles mapped onto these tokens (sheet halfway between its primary and secondary background, surface its secondary, surface-2 its tertiary); Night and Graphite are formdeck's two brand-named dark profiles renamed, and Sage and Mist are new light ones so the light grid is not sparse. Each profile selector also matches `.theme-swatch[data-dark=id]` or `[data-light=id]`, and the base blocks match every swatch of their scheme, so a preferences swatch paints the real palette from the same values instead of a copy. The contrast and accent rules have three-part selectors and come after the profiles, so they win over a profile's (0,3,0) rules by order. A custom accent derives `--accent`, `--accent-hover`, `--accent-wash`, `--on-accent` and `--focus` with color-mix: lifted toward white in dark, darkened toward black in light, and darkened further when `data-accent="light"` (yellow, orange, green, teal) so white text on it stays readable. Density changes only `--control`, `--row` (which is also the sidebar row height), `--text-nav` and `--topbar-h`.

### src/engine/prefs.ts · the look and the startup choice
- refers-to: src/engine/prefs.ts :: `Look`, `ServerLook`, `DARK_THEMES`, `LIGHT_THEMES`, `ACCENTS`, `normalizeLook`, `normalizeServerLook`, `readLook`, `writeLook`, `accentTone`, `readStartup`, `writeStartup`, `startOnHome`
- kind: explainer
- logged: v0.44.0 · 2026-10-20

`normalizeLook` is the one validator for anything read from storage, the server or another tab: unknown ids fall back to the default dark or light, a bad accent to none, a missing `at` to 0. The theme id lists must match the selectors in tokens.css and the server's pattern (lowercase letters, at most 12). `at` is the time of the last change in milliseconds and decides which copy wins between this browser and the account. `startOnHome(store)` is the hook for the desktop app's "On startup" choice: it is true when the person chose Home, and the shell is expected to skip restoring the last page at boot in that case; "Continue where you left off" is the existing behaviour.

### src/ui/theme.ts · the look store and account sync
- refers-to: src/ui/theme.ts :: `currentLook`, `watchLook`, `setLook`, `mountTheme` (`paintLook`, `paintMeta`, `push`, `adopt`, storage listener)
- kind: rationale
- logged: v0.44.0 · 2026-10-20

The look lives in its own small `State` beside the engine's, because it is a view concern and the preferences chunk reads and writes it without widening `EngineState`. Every change writes `formmd.look`, repaints in place through `setAttr` (an unchanged value writes nothing, so the first run after the head script changes no attribute) and, 800 ms after the last change, saves `appearance` with the mode into the account's prefs, which also covers the sidebar's light, system and dark buttons. A `me` update adopts the account's look when its `at` is newer, sets the mode too, and does not echo it back; when the local copy is newer it is pushed once (`sent` stops a server that does not store the field from causing a loop). A newer look written by another tab is adopted from the storage event. The meta theme colour follows the computed `--ground`, so it matches the profile in use.

### src/ui/preferences.ts · the Preferences pane
- refers-to: src/ui/preferences.ts :: `preferencesTab`, `appearanceGroup`, `languageGroup`, `writingGroup`, `privacyGroup`, `desktopGroup`, `radios`, `seg`, `themeGrid`; src/styles/prefs.css; src/ui/account.ts :: `mountAccount`
- kind: rationale
- logged: v0.44.0 · 2026-10-20

The pane is a tab of the account settings dialog, so it loads in the settings chunk and adds nothing to the start bundle. The layout follows formdeck's Preferences dialog: small uppercase group titles, a fixed label column and the control beside it, a segmented control for each small choice, theme swatches 28px with the name underneath, round accent swatches with a custom colour well. Every control applies at once; there is no Save. Segmented controls, theme grids and accents are radio groups of buttons: one tab stop, arrow keys move and choose, and only `aria-checked` and `tabindex` change, so nothing re-renders. Preset accent swatches get their colour through `style.setProperty`, which the style policy allows. The pane listens to `themePref` and the look, so a command, the sidebar buttons or another device update it while open. "Start week on Monday" writes `monday` when on; turning it off returns to automatic, and saving other fields keeps an existing Sunday or Saturday choice. The Desktop app group shows only when `/api/me` says `instance.desktop`, and its choice is stored on this computer under `formmd.startup`.

### src/ui/account.ts · moved into Preferences
- refers-to: src/ui/account.ts :: `profileTab`, `notificationsTab`, `mountAccount` (former `languageTab`, `REGIONS`)
- kind: removed-code
- logged: v0.44.0 · 2026-10-20

Profile's Appearance section (theme choices and the system font switch) and Writing section (the block menu switch), Notifications' Page views section (show my name in page analytics, now under Privacy) and the whole Language and time tab (language, region, time zone, clock, week start and the reload callout) moved into `src/ui/preferences.ts` with the same API calls and storage keys. The theme radio group became the Appearance segmented control; the week-start select became the Monday switch. The tab order is Profile, Preferences, Notifications, Security, Advanced.

### src/extensions/shell/index.ts and src/extensions/account/index.ts · theme toggle and Preferences commands
- refers-to: src/extensions/shell/index.ts :: `shell.theme.toggle`, `prefs.open`; src/extensions/account/index.ts :: `account.languageTime`; src-tauri/src/menu.rs :: `cmd:prefs.open`, `cmd:shell.theme.toggle`
- kind: explainer
- logged: v0.44.0 · 2026-10-20

`shell.toggleTheme` is renamed `shell.theme.toggle` (still Ctrl/Cmd+Shift+L), the id the desktop menubar already calls, next to `shell.theme.light`, `dark` and `auto`. `prefs.open` opens the account dialog on the Preferences tab and is what the menubar's Settings item runs. `account.languageTime` now opens Preferences because the Language and time tab is gone.

## Phase 18 · finder {#p18-finder}

### src/ui/search.ts · filters, sorts and fixed rows
- refers-to: src/ui/search.ts :: `Filters`, `FinderQuery`, `searchURL`, `runSearch`, `destinations`, `compose`, the In picker; src/styles/search.css (.srch__row, .srch__main, .is-stale, .srch-pick)
- kind: rationale
- kind: warning-context
- logged: v0.44.0 · 2026-10-20

The filter row reads Title only, Created by, Shared space, In, Date, then Sort on the right; the Shared space chip is hidden when the workspace has none, decided before the dialog shows. In opens a small picker with its own field: empty it offers the open page and its parents, then recently visited pages; typed, it filters every page of the workspace by title, and the server keeps sub-pages of the pick. Sorts are Best match and edited or created, newest or oldest first; the oldest-first variants send `dir=asc`. The engine client (`engine/client.ts`, not owned here) does not forward `teamspace` or `dir`, so `runSearch` builds the URL itself only when one of them is set and otherwise keeps using `api.pages.search`; once the client forwards both, `runSearch` can call it unconditionally. Every option row is 48px with title on the first line and path plus snippet on the second, headings are 28px, so results swap in place by key without the list changing height. A request slower than 220 ms dims the rows it is about to replace rather than clearing them. Typing three or more letters of home or settings adds a Go to group above the pages; Settings runs `prefs.open` when that command exists and opens account settings otherwise. Up to three commands follow the pages under Commands. With an empty query only recently visited pages show, timed by when they were visited, not edited.

### src/ui/command-palette.ts · Find a command
- refers-to: src/ui/command-palette.ts :: `Palette`, `openPalette`, `readRecent`, `remember`; src/ui/keyboard.ts (palette.open); src/styles/finder.css (.cpal)
- kind: explainer
- logged: v0.44.0 · 2026-10-20

Ported from formdeck's palette: twelve rows are built once and rewritten in place, every query word must appear in the id, label or group, and ranking gives 4 when the label starts with the first word, 2 when the label holds every word and 1 when the command is enabled, then sorts by label. Disabled commands stay in the list, muted, and Enter ignores them. Commands with required parameters are left out because the palette cannot supply them. With an empty query the last five commands run from the palette (localStorage `formmd.palette.recent`) come first under Recently used; group labels use two of the twelve rows. Shortcuts come from `keymap.chordsFor` so a rebinding shows at once, and `keymap.text` joins Mac glyphs without separators and other platforms with plus signs. Ctrl or Cmd plus Shift plus P opens it from anywhere, the editor included; the module loads on first use.

### src/ui/find-in-page.ts · find and replace in the page
- refers-to: src/ui/find-in-page.ts :: `FindBar`, `findHits`, `openFind`; src/ui/keyboard.ts :: `findReady`, `findWanted`; src/styles/finder.css (.findbar, .find-hit)
- kind: rationale
- kind: warning-context
- logged: v0.44.0 · 2026-10-20

Highlights are a `decorations` prop set once on the page's EditorView with `setProps`, not a plugin added by `state.reconfigure`: reconfiguring rebuilds every plugin view, and the y-prosemirror sync and undo views would be torn down and recreated, which can lose undo history. `setProps` merges with the host's props, so the host's own `setProps` calls and its `updateState` on page switch keep it; the function returns nothing while the bar is closed. Matches are computed per textblock with inline nodes as one placeholder character, so a match never spans a mention or equation, and recomputed whenever the document object changes, which covers collaborators' edits. Replace and Replace all are ordinary transactions (all matches in one, applied from the end), so undo and collaborators see them. The bar is a non-modal layer: Escape closes it through the existing layer binding and selects the current match in the editor. Ctrl or Cmd plus F is taken only when an editor is bound to the open page and focus is not in a text field outside the page, so the browser's find still works everywhere else. Read-only and suggesting pages get Find only.

### src/ui/shortcuts-editor.ts and src/engine/keymap.ts · rebinding
- refers-to: src/ui/shortcuts-sheet.ts :: `mountShortcutsSheet`; src/ui/shortcuts-editor.ts :: `mountShortcutsEditor`; src/engine/keymap.ts :: `setCustom`, `loadCustoms`, `customs`, `chordsFor`, `defaultChords`, `conflicts`, `fromEvent`, `text`; src/ui/keyboard.ts :: `KEYS_STORAGE`
- kind: explainer
- kind: decision
- logged: v0.44.0 · 2026-10-20

The sheet module is now a hook that loads the editor chunk the first time the overlay opens, so the up-front bundle shrank. A custom binding is stored per command and replaces the chords of every binding of that command while keeping each binding's scope, typing rule and condition; a command with no binding gets a global one, active while typing only when the chord has a modifier. Customs live in localStorage `formmd.keys` and are loaded by `mountKeyboard` before any key is handled; they are read lazily, so commands registered later are covered. Recording sets `data-own-keys` on the dialog so no shortcut fires, Backspace clears (an empty list, which hides the defaults), Escape cancels, and a chord equal to the only default removes the custom entry. Conflicts are any two commands sharing a chord in a common scope and are shown under each row; the editor does not refuse them, because the scope order in `handle` already decides which runs. Default chords whose binding condition is false right now (the desktop-only new page shortcut in a browser) are not shown.

### src/ui/keyboard.ts · new shortcuts
- refers-to: src/ui/keyboard.ts :: `registerFinder`
- kind: decision
- logged: v0.44.0 · 2026-10-20

Added: palette.open (Ctrl or Cmd plus Shift plus P, also while typing), page.find (Ctrl or Cmd plus F), page.copyLink (Ctrl or Cmd plus L, view scope, so never in a text field, where the browser's own use is left alone) and Ctrl or Cmd plus N for page.new only when `/api/me` says `instance.desktop`, since browsers keep that key. Ctrl or Cmd plus Shift plus L stays with the shell extension's theme toggle; if another module binds the same chord to a second theme command, the most recent registration runs alone and the shortcuts editor shows the overlap.

## Phase 18 · templates {#p18-templates}

### src/ui/templates-dialog.ts · the gallery
- refers-to: src/ui/templates-dialog.ts :: `openTemplates`, `loadTemplateCatalog`, `useTemplate`; src/styles/templates.css (.tplgal, .tplcard); src/styles/automations.css (old .tplgal and .tplcard rules removed)
- kind: explainer
- kind: decision
- logged: v0.44.0 · 2026-10-20

One 1040 by 80vh dialog with two modes in the same element: browse (left rail with search and categories with live counts, right a card grid) and detail (large preview left, name, description, what is inside, title field and buttons right). The grid is built once when the catalogue arrives; search and category only toggle `hidden` on cards and rewrite count text, and switching modes toggles two containers, so nothing is rebuilt. Search needs every typed word somewhere in the name, description, category or tags, inside the selected category. Workspace templates sit first in All and alone under From this workspace; they have no server preview, so their box shows the page icon. A cached catalogue paints at once and is then refetched, and only workspace cards are patched by id. While detail is open the dialog carries `data-own-keys`, which stops the global Escape binding so the dialog's own handler goes back to the grid; a second Escape closes. Arrow keys move by column count measured from the first row's offsetTop, Up from the first row returns to search, Down or Enter in search reaches the first visible card. `opts.category` accepts a category id or name or `workspace`; `opts.preview` takes a starter id or a workspace template page id.

### src/ui/templates-dialog.ts · mountTemplatePreview
- refers-to: src/ui/templates-dialog.ts :: `mountTemplatePreview`, `loadPreview`, `paintPreview`, `rescale`, `watchSize`; src/styles/templates.css (.tplprev, .tplp)
- kind: explainer
- kind: warning-context
- logged: v0.44.0 · 2026-10-20

The preview is a 720 px page scaled with a CSS transform to the host's width; one shared ResizeObserver rescales and drops hosts that left the document. A sizer element carries the scaled height so `.tplprev--scroll` hosts (the detail pane) scroll the whole page, while cards clip it. The host gets `.tplprev` (sheet background, 16:9 by default) synchronously, before any fetch, so the box never blinks; callers that size the host themselves win over the default aspect ratio. Requests are memoised per id and the resolved previews kept, so a preview already seen paints in the same task with no empty frame. The HTML is the server's escaped preview markup inserted with innerHTML; it never contains scripts, handlers or raw HTML from users because starters are compiled into the server. Gallery cards mount through an IntersectionObserver rooted at the scrolling grid with a 240 px margin.

## Phase 18 · integration {#p18-integration}

### src/engine/index.ts · Home is a place you can stay
- refers-to: src/engine/index.ts :: `landing`, `autoLand`, `bootDesktop`, `openPage`, `navigate`, `restoreNav`, the workspaceId watcher
- kind: explainer
- kind: decision
- logged: v0.44.0 · 2026-10-20

Before v0.44.0 `landing()` sent every arrival on the home route to the last page you had open, so Home was only ever seen in an empty workspace. Now it does that once: at boot when the start route is home, and after a workspace switch (`autoLand`). Any navigation by the person (`openPage`, `navigate`, `restoreNav`) clears it, so clicking Home, the Home menubar item or closing the last tab shows Home and stays there. In the desktop app (`boot.me.instance.desktop`) with Preferences, Desktop app, On startup set to Home (`startOnHome`), boot does not jump at all.

### src/extensions/shell/index.ts · commands the menubar needed
- refers-to: src/extensions/shell/index.ts :: `route.home`, `templates.open`, `transfer.import`; src/extensions/account/index.ts :: `account.signOut` (enabled)
- kind: explainer
- logged: v0.44.0 · 2026-10-20

The desktop menubar (src-tauri/src/menu.rs) calls commands by id; these three had no command before, so they are registered here and lazy-load their dialogs. Sign out is disabled inside the desktop app, where there is no password to sign back in with.

### src/ui/keyboard.ts · the palette and find load early
- refers-to: src/ui/keyboard.ts :: `registerFinder` (idle prefetch)
- kind: explainer
- logged: v0.44.0 · 2026-10-20

The command palette and find-in-page chunks are fetched 1.5 s after start in an idle callback. On first use they otherwise loaded after the shortcut, and the first letters typed went into the page. They stay out of the start budget because they are dynamic imports.

### src/ui/page-actions.ts and tools/bundlecheck · the start budget at 268 KB
- refers-to: src/ui/page-actions.ts (Analytics item); tools/bundlecheck/main.go (budget default)
- kind: decision
- logged: v0.44.0 · 2026-10-20

The interface rebuild (resizable sidebar, back and forward, preferences sync, theme profiles, custom shortcuts) put 266.0 KB gzip up front against a 262 KB budget. The page analytics dialog moved behind a dynamic import (1.8 KB back) and the budget rose to 268 KB, the same way it rose to 262 KB for formlit's icon art in v0.42.0. The next feature that needs start-up code should look at moving `page-actions` (4.6 KB) out of the first load before raising it again.

### src/engine/client.ts and types.ts · search sends the shared space and direction
- refers-to: src/engine/client.ts :: `pages.search`; src/engine/types.ts :: `SearchQuery` (teamspace, dir); src/ui/search.ts :: `runSearch`
- kind: fix
- logged: v0.44.0 · 2026-10-20

The search window first sent requests with a shared-space filter or an oldest-first sort through its own fetch, which skipped the session guard that sends a signed-out window to the sign-in page. The client now carries both parameters and every search goes through it.

### src/styles/home.css · Home fits its column
- refers-to: src/styles/home.css :: `.home`, `.home-sec`, `.home-sec__head`
- kind: fix
- logged: v0.44.0 · 2026-10-20

Grid tracks sized to content let the template chip strip (a flex row that scrolls sideways) widen the whole Home column past the window. `minmax(0, 1fr)` columns on `.home` and `.home-sec` and `min-inline-size: 0` on the section head keep it inside; measured in Chromium at 1440 px, nothing outside the strip's own scroller passes the column edge.

### Independent verification fixes (v0.44.0)
- refers-to: src/engine/pages.ts :: `put`; src/engine/router.ts :: `routePath`; src/engine/index.ts :: `autoLand`; public/sw.js :: `APP_ROUTES`; index.html (Home link); src/ui/home-view.ts :: `chosenName`, `categoryName`, `matches`; src/styles/home.css :: `.home-chips`, `.home-grid--row`; src/ui/keyboard.ts :: `palette.open` (key hold); src/ui/command-palette.ts :: `show(typed)`; src/engine/commands.ts :: `Command.hidden`; src/extensions/account/index.ts :: `account.signOut`; src/ui/shortcuts-editor.ts; src/ui/shell-extras.ts :: resize keys; src/ui/theme.ts :: `still`; src/styles/base.css :: `[data-switching]`; src/styles/search.css :: `.srch__filters > *`; src/styles/shell.css (phone collapse button, touch sizes); src/ui/tree.ts :: `pageCount`; src/ui/page-actions.ts (Rename)
- kind: fix
- logged: v0.44.0 · 2026-10-20

An independent check under 4x CPU and slow network found these; each was re-run after the fix. The page store cached each parent's sorted child list as page objects and only cleared it on a move, so a title typed into a new sub-page reverted to Untitled in the tree on the next re-render of its parent (it predates v0.44.0; `put` now clears the cache on every change). Home has its own address, `/home`: `/` still opens where you left off (the server redirects it to the last page), `/home` stays on Home through a reload and keeps the back history, and the desktop window starts at `/home?start=1` so the client can honour On startup either way. Home's chips compared a category id with the starter's category name and showed nothing; they compare names now, search matches category names, and the chips wrap instead of hiding three of them past an invisible scrollbar (they still scroll sideways on phones). Extra Recently visited cards no longer show as thin bordered strips under the first row. Keys typed in the moment the palette's code loads are held and handed to its input. Commands can be `hidden`; Sign out is hidden in the desktop app from the palette and the shortcuts sheet. The sidebar's keyboard resize steps from the set width, not the animating one. Colour transitions are switched off for two frames when the theme or look changes, so no button lags in the old theme. On phones the search filters keep their full labels, the drawer's collapse button gives way to the workspace name, and tab close, breadcrumbs, the page adders and the tag field reach 44 px. The Pages count leaves out databases when they have their own section. The page menu gains Rename (F2).

## Phase 19 · Home showcase and gallery {#p19-home}

### src/ui/home-view.ts, index.html, src/styles/home.css · Data & charts first
- refers-to: src/ui/home-view.ts :: `paintShowcase`, `makeRich`, `paintRich`, `SHOWCASE`, `FEATURE_SHOWN`, `paintTemplates` (strip without the two showcase categories); index.html :: `home-feat-sec`, `home-ex-sec`; src/styles/home.css :: `.home-feature`, `.home-grid--big`, `.home-grid--ex`, `.hcard__desc`, `.hcard--big .tplprev__head`
- kind: explainer
- kind: decision
- logged: v0.45.0 · 2026-10-20

Owner's request of 2026-10-20: the data visualisation templates are the main thing on Home, followed by at least 13 pre-filled example projects. Home now reads greeting, "Data & charts" (a tinted panel with a heading, one line of explanation and up to six large cards, featured first, about twice the area of a template card), "Example projects" (every project, with preview, description and "{n} pages"), the drop area, Recently visited and the Templates strip, which now lists only the other categories. Large cards hide the preview's title and view tabs and shrink its padding, so the chart itself fills the 16:9 frame; the name sits under it. Both grids ship skeleton cards in index.html (six and fourteen, the planned counts) that `patchCards` replaces in place, so nothing moves when the catalogue arrives; previews still mount lazily through the same IntersectionObserver and the templates-dialog chunk, so Home stays a lazy chunk and the up-front budget is unchanged. Cards open the gallery on that template's preview; See all opens it on the category. Both sections hide for read-only members, like the strip, and hide entirely if their category is empty.

### src/ui/templates-dialog.ts, src/styles/templates.css · gallery for projects
- refers-to: src/ui/templates-dialog.ts :: `StarterSummary.featured`, `StarterSummary.pages`, `TemplatePreview.tree`, `starterMeta`, `fillInside`, `run`, `.tplgal__use`; src/styles/templates.css :: `.tplcard__flag`, `.tplgal__tree`, `.tplgal__counts`, `.tplgal__use[aria-busy]`, `.pv*`, `.tplp-dash`, `.tplp-cal*`, `.tplp-heat*`, `.tplp-map`, `.tplp-diagram`
- kind: explainer
- logged: v0.45.0 · 2026-10-20

Featured starters carry a small "Featured" pill in the card's top corner (positioned over the preview, so it never shifts the grid). A project's meta reads "{n} pages", and its preview pane lists What's inside as an indented tree of every page and database with its icon, first view and row count, under a line of page, database and sample-row counts. Use this template keeps its width while the server builds: the label changes to "Building {n} pages…" with an indeterminate bar along the button's bottom edge (a full bar under reduced motion), the status line says it takes a few seconds, errors go to the status line and the label comes back. The `.pv-*` rules style the server-drawn previews: palette indexes and option colours resolve to `--accent` and `--c-*`, heat levels mix `--heat` into `--heat-0`, so every preview follows light, dark and the theme profiles.

## Phase 19 · chart fixes from the showcase {#p19-charts-client}

### src/extensions/database · rough edges the showcase content exposed
- refers-to: props.ts :: `formatNumber`; propedit.ts (Decimal places, Whole number); views/kpi.ts :: `drawKpi`; views/chartkinds.ts :: `radial`, `gauge`, `dots`; views/chart.ts :: `chartFormat`; views/graph.ts :: `readTheme`, the ResizeObserver and MutationObserver; src/styles/charts.css (data-pal 1 and 9); src/styles/templates.css (data-p 1 and 9); src/sandbox/main.ts :: `PALETTE`, `settings`
- kind: fix
- logged: v0.45.0 · 2026-10-20

Precision -1 (which the server already accepted) means whole numbers; the editor offers Whole number. With the default precision, percents show one decimal and money its currency's standard decimals instead of up to eight. A KPI value and a gauge's centre value shrink to fit beside the change badge and inside the arc. The radial centre shows the measure's own aggregate (average, median, min, max or sum) and rings scale to the chart's target when one is set. A scatter (dots) whose values all sit well above zero starts its axis just below the smallest value. Chart labels use the number format of formula and rollup properties when one is set. The graph view read its colours while still detached from the page (as happens when it opens from a tab), so every option colour resolved to an empty string and nodes drew black; it now reads them once it is attached, drops empty colours, re-reads them when the theme profile, accent or contrast changes, and takes the label halo colour from what is actually behind the canvas. Series palette slot 1 moved from violet to blue (violet to slot 9) because the default accent is violet and the first two series looked alike; the server previews use the same order. The diagram sandbox passes the app's palette to pie slices and XY charts, which were pale grey in the neutral theme.

## Phase 19 · independent check fixes {#p19-verify-client}

### src/extensions/database and src/ui · what the throttled check found
- refers-to: props.ts :: `formatCalc`, `formatNumber`; views/table.ts (footer label and value parts); views/board.ts (column total); views/dashboard.ts (number widgets); views/chartdraw.ts (donut and gauge centre fitting, label collisions, legend wrapping, `data-fs`); views/chartkinds.ts and views/chart.ts (`chartFormat` through `formatCalc`, one tick format per axis, short date ticks); views/graph.ts (source colours, grey tones, label placement); views/timeline.ts (first scroll, `tl__bar--out`); ui/tree.ts (`formmd:expand`); ui/templates-dialog.ts and ui/home-view.ts with engine/format.ts :: `searchFold`, `searchHit`; styles/charts.css, viz.css (`.graphv`), database.css
- kind: fix
- logged: v0.45.0 · 2026-10-20

An independent run under 4x CPU and slow network over five example projects and six featured templates (196 views) found these; each was re-checked after the fix. Calcs have their own formatter, `formatCalc(n, fn, p)`: percent calcs come from the server as fractions and show as 64%, counts as whole numbers, averages and medians to at most two decimals, anything else in the property's format and precision; footers split label and value so only the label is ever shortened. Plain number cells stay ungrouped so years read 2024. Chart centre values (donut, radial, gauge) are fitted to the hole and turn compact before they would be cut; every tick on an axis shares one compact format worked out per property; bar labels that collide go compact, then the smaller of a pair is hidden; legends wrap and step down through `data-fs` (the security policy refuses inline styles); week and day ticks drop the year when it is the current one. Grey options draw as a mid slate in charts and a mid grey in graphs. Graph rows from other databases are filled with a tone per source database, chosen away from the hues in use and named in the legend; labels are placed in priority order around their node and skipped when they would collide (they show on hover); `automations.css` rules for the link graph dialog no longer squash the canvas. A timeline opens on all its dated items when they fit, otherwise on today; a bar too short for its title carries it just after the bar. A ring or bar meter with no colour of its own uses the accent. After Use this template on a project, `formmd:expand` marks the new page expanded before it opens, so its sub-pages show in the tree. Template search folds accents and matches words from their start ("map" no longer finds heatmap, "cafe" finds café). On phones the template detail keeps its preview in view when focus moves to Use this template.

## App icon in the family construction {#app-icon-family}

### public/favicon.svg and the app icon · the owner's #F edit, cleaned up
- refers-to: web/public/favicon.svg; assets/form-md_icon.png, form-md_icon_macos.png, form-md_logo.svg; src-tauri/icons/*; src-tauri/splash/icon.png; assets/icon_src/
- kind: rationale
- logged: v0.45.1 · 2026-10-20

The icon is the owner's own edited mockup (assets/icon_src/form-md_edit_source.png), not a redraw: two vector redraws of it changed the artwork and were rejected. The tile was upscaled 4x with EDSR, lightly denoised and sharpened, the soft raster pointer and tag were inpainted out and replaced with crisp vector ones at the same spots (the send-shaped pointer, and the dark ".MD" tag with a white outline the owner preferred), and the tile edge was rebuilt as a clean antialiased mask. Every size is a resize of that 1024 image; the macOS .icns uses it as an 824 px tile inside a 1024 canvas with a 100 px margin and the family drop shadow. favicon.svg and the README logo wrap a PNG of it, so they match the app icon exactly (the favicon is 48 KB, cached after the first load; at 16 px the pointer and tag fall below a pixel and the # carries it). assets/icon_src/README.txt has the steps. The owner's branch `feat/app-icon-redesign` (six commits on v0.45.0) arrived as a bundle; its files were copied in by hand rather than merged, with the same bytes.

### The icon's edge · no light rim
- refers-to: assets/icon_src/make_icons.py; every icon file above
- kind: fix
- logged: v0.45.2 · 2026-10-20

The owner saw a white outer edge once the icon sat on a dark background: the v0.45.1 cut-out mask followed the mockup's light page, so the antialiased edge carried its grey all round and the lower right kept a crescent of its drop shadow (about 10,000 light pixels in the 1024 source ring, up to 236 grey). `make_icons.py` rebuilds every file from art1024.png with a superellipse cut fitted 2 px inside the dark tile, supersampled 4x, transparent pixels filled with the tile's edge colour, and premultiplied downscaling; a ring check over every output (PNGs, each .ico and .icns entry, the favicon and README logo) finds no edge pixel lighter than 60 when shown over black. The artwork inside the tile is unchanged (wrong for the tag: see the next entry).

### The icon's tag · the approved white .MD back
- refers-to: assets/icon_src/make_icons.py :: `cut_tile`, `APPROVED`; assets/icon_src/approved1024.png
- kind: fix
- logged: v0.45.3 · 2026-10-20

v0.45.2 rebuilt the icons from art1024.png alone, which is the stage before the final touch-up and still carries the mockup's first tag (a white pill with dark ".MD"), so the owner's chosen tag (white ".MD" in a white outline on the tile) was lost. The owner sent back the approved full, macOS and logo files (byte-identical to v0.45.1); the full icon is kept as `approved1024.png` and `cut_tile` takes every pixel it holds opaque, eroded 4 px from its old edge, exactly, with art1024.png only for the outer pixels the new superellipse cut reaches. Checked: the 953,276 inside pixels of the new 1024 icon equal the approved file with no difference, the tag and pointer included, and the edge check over every file still finds no light pixel.

### Home · the Templates side panel and a left-aligned page
- refers-to: index.html :: `.home__main`, `#home-side`, `home-side-toggle`, `home-side-close`; src/ui/home-view.ts :: `setSide`, `SIDE_KEY`; src/styles/home.css :: `.home`, `.home__main`, `.home-side*`, `.home-chips`
- kind: rationale
- logged: v0.46.0 · 2026-10-21

The owner asked for Home itself (Start from your material, Recently visited, Templates) to be the first thing seen, with Data & charts moved into a templates side panel and Example projects shown below Home's templates as well as in the gallery. Home is now a two-column grid: `.home__main` on the left and the `aside.home-side` (360 px, sticky, its own scroll) on the right; under 1080 px of view width (a container query on `.view--home`) the panel drops below the main column. The page sits at the left edge of the view (`margin-inline: 0`, up to 1600 px) instead of a centred 1120 px column, so the title and lead line up with the page and not with a centred box. The panel opens by default; the head's Templates button and the panel's close button toggle `data-home-side="closed"` on the root and keep the choice in `localStorage` `formmd.homeSide`; the boot script in index.html sets the attribute before first paint so a closed panel never flashes open. The panel and its button hide with the other template sections for read-only members. All labels reuse strings already translated (Templates, Close, See all), so no locale file changed.

The template chips sat between the title and the search and wrapped to as many rows as the width allowed once the catalogue arrived, which moved the grid below (CLS 0.013 under 4x CPU and a slow network in the narrower column). They now take their own full row under the title and search, never wrap (`flex-wrap: nowrap`, horizontal scroll with the scrollbar hidden), so the 28 px row is reserved from the first paint and the shift is gone (CLS 0).

### Dropdowns · items shown without scrolling
- refers-to: src/ui/floating.ts :: `computePlacement`; src/styles/panels.css :: `.menu`; src/styles/pages.css :: `.popover`; src/styles/editor.css :: `.listbox`; src/styles/database.css :: `.opt-editor__list`; src/styles/viz.css :: `.picklist__list`; src/styles/search.css :: `.srch-pick__list`
- kind: fix
- logged: v0.46.0 · 2026-10-21

Menus, popovers and pickers had fixed caps (300 to 520 px) and scrolled long lists. On the owner's request they now grow to the window: menus, popovers and listboxes up to the viewport less 16 px, lists inside a popover up to the viewport less 56 px. `computePlacement` clamped only the cross axis, so a tall menu that fitted neither below nor above ran off the screen; it now clamps both axes, letting a very tall menu cover its anchor rather than leave the window. A list longer than the whole window still scrolls, with the scrollbar hidden.

### Scrollbars · hidden everywhere, scrolling kept
- refers-to: src/styles/base.css :: `*`, `*::-webkit-scrollbar`
- kind: rationale
- logged: v0.46.0 · 2026-10-21

The owner wants every scrollbar in the app hidden with no exceptions while wheel, trackpad, keyboard and touch scrolling keep working. The reset sets `scrollbar-width: none` on every element and hides `::-webkit-scrollbar` (WebKit in the macOS app, Chromium), both `!important` so the older per-panel `scrollbar-width: thin` and `scrollbar-color` rules no longer show a bar. Those rules are left in place; they are inert now and can be removed when their files are next touched.

### Blocks · the mirrored block's own name
- refers-to: src/extensions/blocks/catalog.ts :: the block entry; src/extensions/blocks/handle.ts :: Turn into a mirrored block, its error toast; src/extensions/blocks/synced.ts :: label, menu, notice; internal/doc/synced.go :: the published-page notice; src/i18n/*/01.json
- kind: rationale
- logged: v0.46.2 · 2026-10-23

The family IP plan approved by the owner on 2026-10-23 renamed the block from another product's feature name to Mirrored block. Labels only: the node type, the `synced` files and classes, the `s:<key>` roots and every stored document are unchanged, and the generic verbs (Copy and sync, Unsync, Unsync all) stay. The seven English keys were renamed in place in all 25 tables with new translations built on each language's word for mirrored, because the old ones followed the other product's localized name.


## v0.47.0 · response-received trigger

- logged: v0.47.0 · 2026-10-23

The automation editor (`extensions/database/automations.ts`) lists **When a response is received** as its own trigger beside **When a page is added**. It shows the condition field (the condition is judged on the submitted answers) and a checkbox, **Also when a page is added by hand**, which sets `trigger.manual`. Both labels are in all 25 locale files (`i18n/*/01.json`). Types: `AutomationTriggerKind` gains `response_received`, `AutomationTrigger` gains `manual`.

## v0.47.1 · by nodemode.ai in About

- logged: v0.47.1 · 2026-10-24

Owner decision of 2026-10-23 for the form family: an app with an About box carries "by nodemode.ai" there. form.md's is the About tab of Admin settings (`ui/admin.ts`, `aboutTab`), so the credit is a third `row__meta` line under the form.md title and its version, static and marked `translate="no"`, with no catalog key since it is the same in every language. Checked in Chromium at 4x CPU and a slow network against a fresh instance: the tab opens with the line in place, CLS 0, no page error.

## v0.47.2 · zero npm audit findings

- refers-to: package.json :: `dependencies.markdown-it`; package-lock.json :: `node_modules/dompurify` (both written by npm, not by hand)
- logged: v0.47.2 · 2026-10-25

Family rule from `FAMILY_DEV_PLAN.md` v0.4.x Phase 2b: every frontend installs with zero `npm audit` findings. This tree had two, and both reach the shipped bundle. DOMPurify 3.4.16 (critical: a SAFE_FOR_TEMPLATES bypass in shadow-root output, and IN_PLACE hooks leaving handlers armed) comes in through Mermaid 11.17.2, whose range `^3.4.12` already allows 3.4.17, so `npm update dompurify` moved the lock with no change to `package.json` and no new override. markdown-it 15.0.2 (high: quadratic-time inline HTML and multi-line link reference definitions) is pinned directly and moved to 15.0.3. Family ledger FB-001 and FB-002. After: `npm audit` 0, `npm run typecheck` clean, `vite build` clean, and a Node render of headings, emphasis, inline HTML, a reference link and a fenced block gave the same HTML as before, with the two pathological inputs rendering in 30 ms and 4 ms. Mermaid diagrams were not drawn in a browser for this change.

## src/extensions/formext · Form extensions host (API 1) {#src-extensions-formext}

- refers-to: src/extensions/formext/index.ts :: `setup`, `consent`, `openFiles`, `saveFile`, `theme`; src/extensions/formext/docops.ts :: `DOC_OPS`, `applyOps`, `readDocument`, `selectionOf`, `registerDocCommands`; src/extensions/formext/host-bridge.ts (vendored kit file, unchanged); src/extensions/blocks/editor.ts :: `onUpdate`, the `formmd-updates` plugin; src/main.ts :: `register(formext)`; index.html :: `sb-extensions`; src/styles/extensions.css; src/ui/icons.ts :: `puzzle`
- kind: feature
- logged: v0.50.0 · 2026-11-07

The window side of the Form Extension Contract. `host-bridge.ts` is the kit's `host/host-bridge.ts` copied unchanged (Apache-2.0, nodemode; NOTICES row), re-copied after the 07:47 kit update (headless idle clock, `open`, `onRequest`); everything form.md-specific wraps it. Consent is asked before any frame exists: `openPanel` awaits `ensureConsent` before it builds the tab or the dialog (the same order as the kit's `open`), and `runCommand` asks before its hidden frame. Not Now is not stored, so the row stays Needs Permission and the next Open asks again. The folder name keeps `extensions/` meaning the client's own modules: `formext` is one more built-in module registered in `main.ts`.

Document API (the contract's `document.*` and `app.*` for form.md, written up in `docs/EXTENSIONS.md`): `readDocument` snapshots the page in the editor (page, writable, mode, top-level blocks with id, type, text, heading level, code language, to-do ticks, Markdown through `MarkdownWriter`, the selection as block id plus offsets in the block's text, and the outline). Edits go through new `doc.*` commands in the catalogue (insertMarkdown, insertText, replaceBlock, setBlockText, deleteBlock, moveBlock, replaceText, markText, select, plus undo and redo), because the existing `editor.*` commands need the editor focused and an extension frame has the focus. `applyOps` checks every op first, builds all of them into one ProseMirror transaction, and dispatches once between two `undoManager.stopCapturing()` calls: the sync plugin turns one dispatch into one Yjs transaction, so the whole call is one undo step that never merges with the person's typing. Any failing op throws before dispatch, so nothing is half applied. Inserted blocks get ids up front (`withIds`) so the result can name them. A blank page (one empty paragraph) is replaced rather than appended to. Writes are refused unless the editor is in edit mode with edit rights (not suggesting, not viewing). The `doc.*` commands are hidden from Find a Command, since they need arguments, but they are in the catalogue mirrored to the server, so MCP `window_command_run` and `form-md cli run` can use them too.

`app.run` runs any catalogue command except risky ones and the `extensions.*` / `ext.*` ones (no recursion through an extension). `net.fetch` is not wired (the window has no outbound network of its own), so the bridge answers -32003. `files.open` is a file input with the `cancel` event for -32002; `files.save` is a download.

UI: the Extensions item sits in the sidebar foot in `index.html`, so it is in final layout before first paint; its menu (`menuButton`, opening upward) is built when opened from the current list. The Extensions panel is a right-side pane with the agent pane's classes (`apane xpane`) and a tab strip: the list first, then one tab per open dock panel (`Name: Panel`, retitled by `ui.setTitle`). Rows are keyed by id with a signature of the record and its consent state; a reload replaces only rows whose signature changed and reorders by moving nodes, so unchanged rows and open panels stay. An extension whose version, state, entry or panels changed has its frames disposed and tabs closed; its contributed commands (`ext.<id>.<command>`, group Extensions, label `Name: Title`) are re-registered only when its name, version or commands changed. Dialog panels use `simpleDialog`; consent uses `simpleDialog` with the kit's permission words and Not Now / Allow. Toasts carry `Name: message`.

Polling: `GET /api/extensions/stamp` every 2 s while `document.hasFocus()` (true while an extension frame has focus) and on window focus; a changed stamp fetches the list. Reload Extensions fetches it always. Theme: a `MutationObserver` on `<html>` theme attributes and style sends `ext.theme` with `--bg` from `--ground`, `--panel` from `--sheet`, `--muted` from `--text-muted` and the rest by name. Events: `editor.ts` gained `onUpdate` (a plugin view that calls listeners on every view update); `document.changed` is sent 250 ms after the last content change and on `onShown` (`reason: "opened"`), `selection.changed` 150 ms after the selection settles.

Not done: translations of the new strings (they fall back to English); per-command items in the desktop's native menu (see src-tauri/HASHOUT_LOG.md).

Integration pass: panels and dialog panels open with the kit's `open()`, which awaits consent before the frame exists (a second check after the await stops a double click making two tabs). `doc.insertText` with `at: cursor` and text without a blank line now goes inline at the selection in its text block, unmarked, replacing selected text; before, it always became a new paragraph after the cursor's block, which put an extension's inline citation on a line of its own. Text with a blank line, or any other `at`, still becomes paragraphs. `files.save` has no save dialog in a browser, so, as the contract's fallback says, it confirms the name in form.md's own dialog first (Cancel is -32002) and then downloads into the downloads folder.

## src/ui/demos.ts · the three demos inside the window {#src-ui-demos-ts}

- refers-to: src/ui/demos.ts :: `DEMOS`, `openDemo`, `mountDemos`, `insideDemo`; src/ui/shell-extras.ts :: `mountHelp`, `mountShellExtras`; src/ui/menu.ts :: `MenuEntry.command` (`data-menu-command`); index.html :: `home-demos-sec`, `home-demos-grid`; src/styles/demos.css; vite.config.ts :: `demoBundles`; ../internal/static/static.go :: `demoCSP`; ../src-tauri/src/menu.rs :: Help; src/i18n/*/29.json; tests/window/test_window.py :: `test_every_demo_opens_in_the_window_from_home_and_help`
- kind: feature
- logged: v0.51.0 · 2026-11-09

Owner rule for the Form family: every app opens all of its demos from inside the app. form.md has three: `demo/` (editor and databases), `demos/views` (database views) and `demos/publish` (writing and publishing). Each one already is the real client (`demos/publish` boots the whole window through `createEngine`; `demos/views` mounts the real `mountDatabase`; `demo/` mounts the real editor and cell painter) run against an in-browser stand-in for the Go server, seeded by the demo's own `seed`/`content` code. So the in-app demo reuses that code as it is, by opening the committed demo build itself inside the window, rather than re-creating its pages in the person's workspace: the documents of form.md live on the server and are saved as they are typed, so a demo copied into the workspace would add pages the person never asked for, and the views demo's relations, rollups, formulas and eight views and the publish demo's comment threads, page history, teammates and publish settings cannot come through the importer. Opening the build gives exactly the browser demo, and it never reads or writes the person's pages.

Build: `demoBundles` emits `demo/index.html`, `demo/assets/**`, `demos/views/...` and `demos/publish/...` from the repository into `dist/` at the same relative paths, so the links between the three demos keep working and the Go binary (and the desktop app, which runs it) embeds them; in dev the same files are served by a middleware. The committed demo builds stay the single source; nothing is copied into the tree, and only the media those builds already load ship (credited in each demo's `CREDITS.md`; the NOTICES rows say they are now in the binary). About 12.7 MB of the 23 MB `dist/`.

Isolation: the frame is `sandbox="allow-scripts allow-downloads"` (no `allow-same-origin`), and the server sends `/demo/*` and `/demos/*` with `demoCSP`, which carries the same `sandbox` directive, `connect-src` unset under `default-src 'none'`, `frame-ancestors 'self'` and `X-Frame-Options: SAMEORIGIN` instead of the global `DENY`. The demo document has an opaque origin: it cannot read the session cookie, call `/api`, or touch the window's localStorage. Its own theme and sidebar writes are already wrapped in try/catch, so they fail quietly and it opens dark. `'unsafe-inline'` for scripts is allowed only for these sandboxed documents, for the demos' theme boot script.

UI: Home's Demos section is static markup in `index.html` before Example projects, three `hcard hcard--ex` buttons with a cover `img` inside the 16:9 `.hcard__preview` (width and height set, `loading="lazy"`), so nothing moves when the pictures arrive. The server's `localize` translates its text from the catalogs like the rest of the shell. `openDemo` builds a full-window stage (`.demo-stage`, `--z-overlay`) with a bar (a Demo badge, the title, a line saying nothing reaches the person's pages, and Close demo) over the frame, registered with `openLayer` as modal, so the app behind is inert, Escape closes it while focus is outside the frame, and focus returns. One stage at a time. `mountDemos` registers `demo.editor`, `demo.views` and `demo.publish` (group Demos, not mutating) so Find a Command, the window command catalogue (MCP `window_command_run`, `form-md cli run`) and the native Help menu (`cmd:demo.*` through `__formmdRun`) all open them; the in-window Help menu lists them under a Demos heading. Menu items now carry `data-menu-command` for tests (not `data-command`, which the shell routes globally).

Inside a demo: `demos/publish` builds from `index.html` and runs `mountShellExtras`, so its rebuilt bundle carries the Demos section and Help items too. `insideDemo()` (file:// or a path under `/demo/` or `/demos/`) hides the section and registers no demo commands there, so a demo never opens a demo.

Checked: the type check, the build, the Go build, the three demo builds, and in Chromium against a fresh server every demo opened from Home and from the Help menu (screenshots compared with each demo page opened on its own) and closed again; the window smoke test gained one test that does the same.

## v0.48.0 · header alignment

- refers-to: src/ui/page-style.ts :: `OPTIONS` (header), `DEFAULTS`, `applyStyle`, `openStyleMenu`, `page.style` command; src/engine/types.ts :: `PageStyle.header`; src/styles/page-style.css (header rules)
- logged: v0.48.0 · 2026-11-03

On a database page the header used to sit in the centred document column (720 px) while the views below run the full width less `--space-10` each side, so the title looked centred over a left-anchored board. The header now breaks out exactly the way `.page-db` does (`--head-w: max(100%, calc(100cqi - 2 * var(--space-10)))` with the same start margin), so on the default, left, the icon and title line up with the first view tab. On document pages the header stays in the document column, so it keeps lining up with the text below it. Header alignment, Centred or Right, in the page style menu aligns the header's contents inside that band: grid items centred or at the end, `text-align` inherited by the title and description textareas (Chrome's form-control styles otherwise keep them at the start), the adders and tags row justified, the icon's and adders' negative optical margins moved to the matching side, and the tag input sized to its placeholder (`field-sizing: content`) so "Add a tag" is not pushed off centre by a 120 px box. `DEFAULTS` replaces the inline `sans`/`start` checks in `applyStyle` and the menu. `data-kind` and `data-style-header` are both set in the same synchronous paint in page-view.ts, before the header is visible.

## v0.52.1 · a cached page is ready only once its write flag is known {#doc-ready-write}

- refers-to: src/engine/doc/docs.ts :: `DocSession.attach` (the IndexedDB `whenSynced` branch), `markReady`; src/ui/mode-switch.ts :: `mountModeSwitch`; src/extensions/blocks/editor.ts :: `canEdit`
- kind: fix
- logged: v0.52.1 · 2026-11-09

On a reload of a page this browser had cached, `attach` called `markReady` as soon as the IndexedDB copy had content and only then read the cached `write` flag in a second lookup. In between, the session was ready but not writable, so `canEdit` was false while `canSuggest` was true: the mode button showed Viewing (92 px) and the editor was read-only, then both turned to Editing (85 px) a few milliseconds later, a visible relabel and a layout shift. Chromium caught it in 6 of 8 reloads in one browser session. `markReady` now waits for the `gen` and `write` reads, which are the same two IndexedDB gets, so the first ready already carries the cached flag; 0 of 8 reloads after. A first load with no cache is unchanged (the server's `opened` sets `write` before sync step 2 marks ready).
