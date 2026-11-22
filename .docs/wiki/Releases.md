# Releases

What each version of form.md brought, newest first. One row per version; the wording is the project's own record of that release.

- **Current version:** v0.61.1
- **Where releases come from:** Releases built by formcompiler and the update check and installer reading its form.md-updates release.
- **Downloads:** the README's Download section, and [[Install]] for servers.
- **How a release is cut:** `BRANCHING.md` at the repo root.

## v0.60 and later

| Version | What it brought |
|---|---|
| v0.61.1 | Lets an account that has no password, such as the desktop app's owner, set its first one without being asked for a current one, and adds [[Self-hosting]], [[Auth0]] and the `deploy/gcp` scripts |
| v0.61.0 | A desktop app whose window signs in with a key the server makes at each start and accepts only once, so a sign-in from outside the app's window is refused, and third-party notices that a build writes exactly as they are kept, the demo media and the extension kit included |
| v0.60.0 | A desktop app on Windows in which nothing the server starts (Claude Code and the other agents, the tunnel, the server's own clean-up) opens a console window, with `FORMMD_ENGINE_WINDOWED=0` to open one with the server's output for a single start, and the views and publish demos rebuilt with the app's current styles |

## v0.50 to v0.59

| Version | What it brought |
|---|---|
| v0.59.0 | The offline demo rebuilt with the app's current styles, AMOLED Black and the shared terminal panel's included, and the Form family's token names available in form.md's stylesheets so shared pieces can use them, with every colour, size and theme of form.md as it was |
| v0.58.0 | A desktop app on Windows whose server always ends with it, however the app ends, a start that fails shown in a dialog with what the server last printed, its exit code and where the log is, and a wait of at most 90 seconds for the server to start; on macOS a server that has stopped but was not yet cleaned up no longer counts as still running; `./form-md claude` and `./form-md agents install` finding Claude Code the way the app does, `agents install` naming Antigravity, Antigravity in Connected apps, and the terminal panel's guide saying that Codex, Cursor and Antigravity each get a button when they are installed |
| v0.57.0 | A cleaner start and stop for the desktop app, on Windows most of all: a server left behind by an app that is gone is replaced at the next start where it used to block it, a second start on the same data folder is refused with a sentence that says form.md is already running there, the server stops within a few seconds once the app that started it is gone, and a start that fails shows its message before the loading window closes; and a folder of its own for every Antigravity tab, so two tabs open at once no longer take each other's access |
| v0.56.0 | Antigravity (Google's `agy` program) as a fourth agent beside Claude Code, Codex and Cursor's agent, with a tab of its own in the terminal panel, and one way of finding all four programs however form.md was started, which says where it looked when a program is missing and what the program printed when it does not start |
| v0.55.0 | The terminal panel as the one every Form app shares: it has a row or a column of its own in the window, so it no longer lies over the page, and a button in its head moves it between the bottom and the right; Codex and Cursor's agent opening in tabs of their own beside Claude Code's, with a More agents menu and a short guide the first time the panel opens; the panel's side, size and front tab remembered, a height set in an earlier version kept; the same keys as before (`Mod+J`, `Mod+Alt+J`, `Mod+Alt+K`), and with the cursor anywhere in the panel no other key reaching the page behind it; a Restart button in the panel's head where a double-click on a tab restarted it; From Claude Code as a command in Find a command where it was a button in Chat; and a page that asks the server nothing about the panel while it is closed |
| v0.54.0 | A token of its own for every Claude Code tab in the terminal panel, so Account settings, API tokens lists one "Claude Code tab in form.md" row per open tab where it listed one shared row, and closing a tab ends its token, while Chat keeps its single "Claude Code in form.md" token; a terminal that attaches only from the page the server itself served; AMOLED Black, a true-black dark theme for OLED screens, in Preferences; `./form-md agents install` registering Claude Code with a header that names `FORMMD_MCP_TOKEN` and no longer contains it; `window_command_run` and `extension_command` announced to agents as tools that can change the workspace, with `extension_command` no longer listed to a read-only credential; `agents/manifest.json`, the list of what form.md offers agents, written by `./form-md agents manifest` and checked by `--check`; server routes that open Codex and Cursor's agent in a terminal tab with a token each and report which of the three programs are installed, ahead of the window's own buttons for them; and the editor's live sync connecting again through the Vite dev server |
| v0.53.0 | A date range on database views (All dates, Today, Last 7 days, Last 30 days or a custom start and end date), kept per viewer and view in the browser and sent with every rows and chart request as `date_range`, which the server ANDs with the view's saved filter, plus Today, Last 7 days and Last 30 days as Is within filter options, and a view tab strip that stays inside the database with a count button listing the views scrolled out of sight |
| v0.52.2 | Builds the container image again, its interface build carrying the demo builds it packs |
| v0.52.1 | Opens a page the browser has cached as editable from the first frame, so the mode button no longer shows Viewing before Editing |
| v0.52.0 | Header alignment as a page style, the page header on the left by default, centred or on the right per page or for the workspace, and fixed the page top bar's Edited label arriving after the mode button |
| v0.51.0 | The three demos inside the app: a Demos section on Home, a Demos group in the Help menu, the native Help menu and Find a command, and demo.editor, demo.views and demo.publish for agents, each opening the demo's own build in a sandboxed frame over the window that never reads or writes your pages, with the demos' credited media shipped in the binary |
| v0.50.1 | The extension refusal for a manifest's API in the same words every Form app uses |
| v0.50.0 | Form extensions: form.md loads the extensions FormPortal installs and form.ext runs, each in a sandboxed frame with its own consent, from an Extensions menu, an Extensions panel and the command palette, with document.apply as one undo step and extensions_list and extension_command for agents |

## v0.40 to v0.49

| Version | What it brought |
|---|---|
| v0.49.3 | Two more offline demos, demos/views for the database views on a real dataset of US crewed spaceflight and demos/publish for writing, comments, page history, publishing and export on the whole window, each credited in its own CREDITS.md, and a link row between the three demos |
| v0.49.2 | The desktop app's splash as the Form family's rounded card with a progress bar, square corners gone on macOS, and a main window that appears whole once its first content is drawn |
| v0.49.1 | Real public-domain and freely licensed media in the offline demo, a notebook of Galileo, NASA and Pexels material with a real exoplanet database, all credited in demo/CREDITS.md |
| v0.49.0 | An offline browser demo in demo/ that opens from demo/index.html with no server |
| v0.48.0 | The README in the Form family template (#5), merged with main, and the unified app icon |
| v0.47.8 | The About tab and the installers naming NodeAgentc, Inc. as the publisher with the Apache-2.0 licence |
| v0.47.7 | Linux packages that recommend GStreamer's plugins and an AppImage that carries them, so the meeting recorder works on a fresh install |
| v0.47.6 | A container image built with Go 1.26.9, as go.mod states, and line endings fixed for Windows checkouts, CRLF for the .bat scripts and LF for everything else |
| v0.47.5 | A first paint that waits for the whole page shell, so the sidebar, the top bar and Home no longer jump while they load, and a CI check that loads the window under throttling |
| v0.47.4 | A Go test suite that checks the desktop key, the origin guard, the sync socket, live editing by two clients and an agent, migrations, Markdown round trips, viewer and share-link limits, and the version sites |
| v0.47.3 | A desktop release build that keeps its symbols, since macOS builds fail on a stripped binary |
| v0.47.2 | A web client with no known vulnerable dependency: DOMPurify 3.4.17 and markdown-it 15.0.3 |
| v0.47.1 | Names the copyright holder, NodeAgentc, Inc., and shows by nodemode.ai in Admin settings, About |
| v0.47.0 | Stores each form submission as one response record with computed columns and adds the response-received automation trigger |
| v0.46.3 | Third-party notices that cover every platform's Go modules, KaTeX's fonts under the OFL and the Linux AppImage's LGPL libraries |
| v0.46.2 | The Mirrored block under its own name in every language |
| v0.46.0 | Home leads with your own material, recent pages and templates, with Data & charts in a Templates side panel you can close, Example projects below the templates, the page aligned to the left, dropdowns that show every item and every scrollbar hidden |
| v0.45.3 | The app icon as the owner chose it, the white .MD in a white outline, with its clean edge |
| v0.45.2 | The app icon's light outer rim removed at every size |
| v0.45.1 | The new app icon: the owner's glass # with the F in front, a pointer and a .MD tag, in the app, the favicon and the desktop splash |
| v0.45.0 | Data & charts first on Home: 15 featured templates covering every chart kind and database view, and 14 example projects already filled in, each a home page with dashboards, sub-pages and related databases, for 90 templates in eleven categories |
| v0.44.0 | A page-first layout with Home at /home, 61 templates in nine categories with a previewing gallery, search filters and sorts, Find a command, find and replace, rebindable shortcuts, and Preferences with dark and light theme profiles, accent, density and high contrast |
| v0.43.0 | Form.md as a desktop app for macOS, Linux and Windows, built by ./build_app or build_app.bat, with the server inside it and the owner signed in |
| v0.42.1 | The build no longer stops when Go keeps its licence outside GOROOT, as Homebrew Go 1.27 does |
| v0.42.0 | Made for everyday people: a plain home with recent pages, simpler sharing, publishing, sign-in and settings, tables you can find and fill, phone-sized controls; formlit's icons throughout and the three-star sparkle retired; a Claude Code panel and chat that work like formlit's, with an ask bar and Ctrl/Cmd+J |
| v0.41.0 | Answers that can draw on the web, uploaded PDFs searchable by their text, boards and galleries kept on published pages, Mermaid drawn in its dark theme, the slash menu, format bar and cursors inside synced blocks, long-press drag in the page tree, a Graph tab in the library, webhook page events carrying the page, import sources named by file format |
| v0.40.0 | Page-tree import checked against twelve real Markdown and CSV page exports from public open-source test corpora and six defects fixed: links under folders starting with #, database links, ids left in titles, attached zips imported twice, videos reported as skipped, image alt text |

## v0.30 to v0.39

| Version | What it brought |
|---|---|
| v0.39.4 | Terminal styles kept inside the terminal pane, closing the v0.39.0 review list |
| v0.39.3 | A sign-out that lands while a connection is opening now closes it too |
| v0.39.2 | One sibling lookup at a time, an in-app agent token that recovers from a revoke without a restart, a bounded formula pattern cache |
| v0.39.1 | An OAuth grant whose access token runs out without a refresh closes its live connections within a minute |
| v0.39.0 | A release review: Go 1.26.9 and x/net v0.61.0 with no reachable vulnerabilities, offline database changes kept per account, expired sessions closing their live connections, sibling answers checked before agents see them |
| v0.38.1 | Starter hints as callouts and offline deletes checked across a server restart |
| v0.38.0 | Database changes made while the server was down are sent as soon as it is back, without a reload |
| v0.37.0 | Database views that really open offline, checked with the server stopped, which the v0.32.0 and v0.33.0 versions did not |
| v0.36.0 | Dashboard widgets moved with the arrow keys and a graph row list for screen readers |
| v0.35.0 | Pickers loaded on first use and a 260 KB start budget the build enforces |
| v0.34.0 | A lighter start: one translation chunk per language and the library loaded on demand |
| v0.33.0 | Board, gallery, calendar, timeline, map and chart views of kept and recent databases open offline without having been opened first, maps included |
| v0.32.0 | Those views offline as last seen, and terminal colours following a system theme switch |
| v0.31.0 | Terminals drawn correctly under the security policy, a panel that opens with its terminal, restart in place, a chat tab without the empty-state error |
| v0.30.0 | Revoking one token, session or OAuth grant closing its live streams, editing sockets and terminal attachments |

## v0.20 to v0.29

| Version | What it brought |
|---|---|
| v0.29.0 | Context packs that name formlit or formvis when they fit the task, and the family-routing sub-skill |
| v0.28.0 | The form family bridge: a public descriptor and tools that tell agents when formlit or formvis fits a task |
| v0.27.0 | Faster bulk row tools for agents |
| v0.26.0 | Rows added offline |
| v0.25.0 | Saved styles, offline row deletes, bulk row updates nearly three times faster |
| v0.24.0 | Annotation marks in Markdown and PDF, uploaded fonts in HTML exports, a list view that loads as you scroll |
| v0.23.0 | Editable single annotation marks, uploaded fonts on published pages, a windowed timeline |
| v0.22.0 | Offline database edits kept and sent later, page styles on public pages and exports, the slash setting in account settings |
| v0.21.0 | A phone block toolbar, renaming and removing fonts, agents kept out of the block a person is typing in |
| v0.20.0 | Phase 16 audit fixes: pictures from agents imported on the server, block alignment on public pages, a server-side default for command safety |

## v0.11 to v0.19

| Version | What it brought |
|---|---|
| v0.19.0 | Offline by default with larger databases, phone-width fixes, described window commands |
| v0.18.0 | The board block and creative embeds |
| v0.17.0 | Custom fonts and typewriter scrolling |
| v0.16.0 | Image annotations and block alignment |
| v0.15.0 | Page styles, focus mode and word counts |
| v0.14.0 | The terminal, Claude Code and its chat inside the app, window commands for agents, 26 languages |
| v0.13.0 | Context packs and agent connections |
| v0.12.0 | The visual data views |
| v0.11.0 | Encryption, ACME, operations commands, offline start and the installer |

## Related

[[Home]] · [[Install]] · [[Development]]
