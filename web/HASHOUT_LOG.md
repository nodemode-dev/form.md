# HASHOUT_LOG · frontend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the web client in `web/`. Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to `web/`.

Current build version: v0.1.0
Last updated: 2026-10-07 · Phase 1 client shell

---

## Org-navi-map

```
web/
├── HASHOUT_LOG.md                   this file
├── index.html                       shell skeleton, pre-paint theme boot, icon placeholders
├── package.json                     scripts: dev, build, preview, typecheck
├── tsconfig.json                    browser code (src/)
├── tsconfig.node.json               vite.config.ts only
├── vite.config.ts                   port 5750, host false, proxy to :8750, icon inlining, dist placeholder
├── oc_install, oc_install.bat       npm ci (or npm install without a lockfile)
├── boot_frontend, boot_frontend.bat npm run dev
├── public/
│   └── favicon.svg                  brand mark on dark tile
├── dist/                            build output, embedded into the Go binary (only .gitkeep is tracked)
└── src/
    ├── main.ts                      create engine, register extensions, mount UI, start
    ├── engine/                      headless: no DOM access
    │   ├── index.ts                 Engine, EngineState, createEngine, Platform adapter contract
    │   ├── state.ts                 per-key observable state
    │   ├── commands.ts              command registry
    │   ├── keymap.ts                scoped shortcut registry, chord parse, match, format
    │   ├── api.ts                   fetch wrapper with typed ApiError
    │   ├── health.ts                server health polling
    │   └── prefs.ts                 persisted preference keys and readers
    ├── extensions/
    │   ├── registry.ts              register, activateAll, disposeAll
    │   └── shell/
    │       └── index.ts             shell commands and their shortcuts
    ├── ui/                          DOM layer: renders engine state, dispatches commands
    │   ├── app.ts                   mountUI, platform and storage adapters, command click delegation
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
    │       ├── tokens.css           every colour, size, space, radius and motion token
    │       └── z-layers.css         the one stacking scale
    └── styles/
        ├── main.css                 import order
        ├── base.css                 reset, body ground, kbd, focus ring, reduced motion
        ├── controls.css             icon button, text button, segmented control
        ├── shell.css                app grid, sidebar, sheet, topbar, drawer
        ├── doc.css                  document column and status card
        └── overlays.css             tooltip, scrim, dialog, shortcut rows, toasts
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
