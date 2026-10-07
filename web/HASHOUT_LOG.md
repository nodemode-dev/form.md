# HASHOUT_LOG · frontend · form.md

Living log of all comment content, explainers, rationale, TODOs and removed code for the web client in `web/`. Source files stay comment-free per dev-rules `hashout-log`; everything they would have said lives here, indexed. Read this first, then edit code surgically. Paths are relative to `web/`.

Current build version: v0.2.1
Last updated: 2026-10-07 · fonts, appearance, panes patched in place

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
    ├── main.ts                      read boot and auth, set language, create engine, mount, start
    ├── fonts/                       Inter and JetBrains Mono variable woff2 (latin, latin-ext) plus OFL texts
    ├── engine/                      headless: no DOM access
    │   ├── index.ts                 Engine, EngineState (auth, me, workspace), createEngine, Platform contract
    │   ├── state.ts                 per-key observable state
    │   ├── commands.ts              command registry
    │   ├── keymap.ts                scoped shortcut registry, chord parse, match, format
    │   ├── api.ts                   request() and getJSON with typed ApiError (code, message, data)
    │   ├── client.ts                typed API client: auth, me, workspaces, admin
    │   ├── types.ts                 wire types mirroring the Go JSON
    │   ├── i18n.ts                  t(), plural(), lazy catalogs, language resolution
    │   ├── format.ts                Intl date, time and relative formatting from account preferences
    │   ├── health.ts                server health polling
    │   └── prefs.ts                 persisted preference keys and readers
    ├── extensions/
    │   ├── registry.ts              register, activateAll, disposeAll
    │   ├── shell/
    │   │   └── index.ts             shell commands and their shortcuts
    │   └── account/
    │       └── index.ts             account, workspace and admin commands, Ctrl or Cmd plus comma
    ├── ui/                          DOM layer: renders engine state, dispatches commands
    │   ├── app.ts                   mountUI, platform and storage adapters, command click delegation
    │   ├── auth.ts                  set-up, sign-in, sign-up and invitation screens, workspace cookie
    │   ├── session.ts               workspace switcher and account menu, live names and tiles
    │   ├── account.ts               account settings: profile, security, tokens, language and time
    │   ├── workspace-settings.ts    workspace settings and the new-workspace dialog
    │   ├── admin.ts                 instance settings: general, sign-in, email, users, workspaces, about
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
