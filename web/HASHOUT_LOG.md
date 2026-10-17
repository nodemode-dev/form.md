# HASHOUT_LOG · frontend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the web client in `web/`. Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to `web/`.

Current build version: v0.39.3
Last updated: 2026-10-16 · v0.26.0: offline row creates; v0.25.0: offline row deletes, saved styles; v0.24.0: list loading on scroll, marks in copied Markdown; v0.23.0: picking single marks, timeline windowing; v0.22.0: offline row outbox, Writing settings; v0.21.0: phone block toolbar, block presence, font rename and remove; v0.20.0: alignment on public pages, align.css; v0.19.0: offline by default, phone width, command metadata; v0.18.0: the board block; v0.17.0: custom fonts, typewriter scrolling; v0.16.0: image annotations, block alignment; v0.15.0: page styles, focus mode, word counts; v0.14.1: the vector canvas engine removed; Phase 14: the terminal and Claude Code dock, the command catalogue, 26 languages

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
