# HASHOUT_LOG · desktop shell · form.md

Living log of explainers, rationale and removed code for the desktop app in `src-tauri/` and its build scripts (`build_app`, `build_app.bat`, `tools/build_desktop`, `.github/workflows/desktop.yml`). Source files stay comment-free per dev-rules `hashout-log`; what they would have said lives here. Paths are relative to the repo root.

Current build version: v0.44.0
Last updated: 2026-10-20 · v0.44.0: version only (the menubar's route.home, templates.open and transfer.import commands now exist in the client); Phase 18: the desktop app

---

## Org-navi-map

```
src-tauri/
├── Cargo.toml / Cargo.lock      the window program (crate form-md-desktop, Tauri 2.12)
├── tauri.conf.json              product form.md, id ai.nodemode.formmd, splash window, bundle targets, the server as externalBin
├── build.rs                     tauri_build::build()
├── capabilities/default.json    core:default for the splash and main windows; no remote IPC
├── Info.plist                   merged into form.md.app: microphone reason, local networking allowed
├── entitlements.plist           hardened runtime: microphone input only
├── splash/index.html            the 360x180 window shown while the server starts
├── icons/                       32, 128, 256, 1024 PNG, icns (inset macOS tile), ico
├── binaries/                    form-md-server-<target triple>, copied in by tools/build_desktop (ignored by git)
├── NOTICES.md                   licences of the Rust crates the app links (go run ./tools/notices desktop)
└── src/
    ├── main.rs                  starts the server, waits for its ready line, opens the window, stops the server on quit
    └── menu.rs                  the macOS menubar and its command, zoom, reload and notices items
```

---

## src-tauri/src/main.rs {#main-rs}

### src-tauri/src/main.rs · the server as a sidecar
- refers-to: src-tauri/src/main.rs :: `start`, `desktop_key`, `data_dir`, `READY`, `PREFERRED_ADDR`
- kind: explainer
- logged: v0.43.0 · 2026-10-20

The app starts the same form-md binary the server builds use, renamed `form-md-server-<triple>` because Tauri's `externalBin` needs the triple suffix and the deb installs it beside the window program as `form-md-server`. It passes `serve`, a fresh 32-byte random `FORMMD_DESKTOP_KEY` per launch, `FORMMD_DATA` (the user's own value if set, else `~/form.md`) and `FORMMD_ADDR=127.0.0.1:8750`. It waits for the exact line `form-md desktop ready http://127.0.0.1:PORT` on standard output and opens the main window at `/desktop/enter?key=...`, which signs the owner in and redirects to `/`. Server log lines at ERROR level before the ready line are kept, so a failure dialog can say why (for example the data folder is locked by another form.md).

### src-tauri/src/main.rs · stopping the server
- refers-to: src-tauri/src/main.rs :: `stop_server`, `Server`, `STOP_WAIT`
- kind: explainer
- kind: warning-context
- logged: v0.43.0 · 2026-10-20

On quit the shell drops its handle on the server, which closes the server's standard input; the server treats that as the signal to stop cleanly (flush documents, release the data lock). The shell waits up to 8 seconds for the Terminated event and only then kills the process by pid. If the shell itself crashes the operating system closes the pipe and the server stops the same way. tauri-plugin-shell's own kill-on-exit list holds only children spawned from JavaScript, so it never kills this one early. A server that stops while the window is open shows a dialog and quits the app, since the window would otherwise show a dead page.

### src-tauri/src/main.rs · the main window
- refers-to: src-tauri/src/main.rs :: `open_main`, `is_local`, `open_outside`, `free_name`
- kind: explainer
- kind: decision
- logged: v0.43.0 · 2026-10-20

The window is 90% of the primary monitor in logical pixels, never below 980x640. `disable_drag_drop_handler` hands file drops to the page, so dropping files on the import area and dragging blocks keep working (Tauri would otherwise swallow them on Windows). Navigation stays on the app's own loopback port; any other http, https or mailto link, and every `target=_blank` or `window.open`, opens in the system browser. Downloads (exports) go to the Downloads folder with a numbered name when one exists, and the finished file is revealed in Finder or Explorer, because a webview has no download bar of its own. The capability gives the page no Tauri commands: the page talks only to its server.

### src-tauri/src/menu.rs · the macOS menubar
- refers-to: src-tauri/src/menu.rs :: `build`, `handle`, `Zoom`; src-tauri/src/main.rs :: `main` (set_menu)
- kind: decision
- logged: v0.43.0 · 2026-10-20

Owner's decision of 2026-10-20 (between a ribbon and none): the window itself stays free of a ribbon, and the full menus live where a Mac app keeps them, in the system menubar. Items with an id `cmd:<command>` call `window.__formmdRun(<command>)`, the client's door into its command registry, so the menubar and the page's own shortcuts run the same code. Web content sees key equivalents first, so when the page handles a shortcut the menu item does not fire twice. Windows and Linux get no in-window menubar, matching the page-first look; every item there is reachable from the page and its shortcuts. Zoom steps through 67% to 175% with `set_zoom`.

---

## Build scripts {#build}

### build_app, tools/build_desktop, build_app.bat · one command to the app
- refers-to: build_app (option loop, desktop hand-off); tools/build_desktop; build_app.bat
- kind: explainer
- logged: v0.43.0 · 2026-10-20

`./build_app` keeps its five server steps and then hands the fresh `bin/form-md` to `tools/build_desktop`, which checks the toolchain (Xcode tools on a Mac, WebKitGTK on Linux with an offer to apt-get it, rustup's Rust with an offer to install it, rustc and cargo from the same install, an Apple silicon Mac building for aarch64), checks that the Tauri command line in `web/node_modules` and the tauri crate in `Cargo.lock` share major and minor, checks that `tauri.conf.json` and `Cargo.toml` carry the release version, copies the server in under its triple name and runs `tauri build`. Ported from formdeck's build_app with two of its bugs avoided: `--config` is a known option here (formdeck's parser rejected it, so its signed builds never made updater artifacts) and Windows builds go through `build_app.bat` instead of bash. An E0463 failure clears the Rust cache and builds once more; on a Mac a failed .dmg step still leaves a usable form.md.app. `--server-only` keeps the old binary-only build.

### .github/workflows/desktop.yml · every platform on a tag
- refers-to: .github/workflows/desktop.yml
- kind: explainer
- logged: v0.43.0 · 2026-10-20

Five runners (macOS arm64 and Intel, Linux x64 and arm64, Windows x64) build on each `v*.*.*` tag and attach the .dmg, .deb, .rpm, .AppImage, .msi and setup .exe to the release that release.yml creates, waiting up to ten minutes for it to exist. Apple signing and notarising switch on when the APPLE_* secrets are set; without them a downloaded macOS build is unsigned and macOS blocks its first launch until it is allowed under System Settings, Privacy and Security, Open Anyway. An app built on the Mac itself with `./build_app` carries no download quarantine and opens directly.
