# HASHOUT_LOG · frontend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the web client in `web/`. Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to `web/`.

Current build version: v0.7.0
Last updated: 2026-10-09 · Phase 7: sharing, comments, suggest mode, inbox, history, publishing, analytics, webhooks, audit, provisioning, SAML

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
│   └── favicon.svg                  brand mark on dark tile
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
    │   ├── commands.ts              command registry
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
    │   ├── share-dialog.ts          people, groups, guests, links and the publish entry
    │   ├── teamspace-dialog.ts      teamspace create and settings
    │   ├── publish-dialog.ts        publish to the web: address, SEO, picture, theme, font, password, domain
    │   ├── history-dialog.ts        page history: versions, preview, block diff, save, restore
    │   ├── analytics-dialog.ts      page analytics: KPIs, chart, creator, editors, viewers
    │   ├── bar-chart.ts             inline SVG bar chart updated in place
    │   ├── comment-view.ts          thread, composer and suggestion views
    │   ├── comments-pane.ts         the comments pane with filters
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

Source strings are English and double as keys; catalogs in `src/i18n/<lang>.json` load lazily through `import.meta.glob`, so English costs nothing. `plural` keys are `"one|other"` with the translation in the same form. Supported languages are English and German; the German catalog arrives in Phase 10, until then German falls back to English.

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
