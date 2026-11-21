# HASHOUT_LOG · desktop shell · form.md

Living log of explainers, rationale and removed code for the desktop app in `src-tauri/` and its build scripts (`build_app`, `build_app.bat`, `tools/build_desktop`; `.github/workflows/desktop.yml` until v0.46.0). Source files stay comment-free per dev-rules `hashout-log`; what they would have said lives here. Paths are relative to the repo root.

Current build version: v0.61.0
Last updated: 2026-11-20 · v0.61.0 the window's key comes from the server's own output (#window-key); 2026-11-20 · v0.60.0 no shell change (the version only; the server is still started through tauri-plugin-shell, whose CREATE_NO_WINDOW keeps its console window away, see internal/HASHOUT_LOG.md #p31-no-console); 2026-11-20 · v0.59.0 no shell change (the version only); 2026-11-20 · v0.58.0 the server held in a job object on Windows, a failed start shown in a dialog with the server's last lines, its exit code and the log's path, the wait for the address limited to 90 seconds (#engine-start); 2026-11-19 · v0.57.0 a failed start shows its dialog before the loading window closes, and the server is given the app's process id (#windows-start); 2026-11-19 · v0.56.0 no shell change (the version only); 2026-11-19 · v0.55.0 no shell change (the version only); 2026-11-19 · v0.54.0 no shell change (the version only); 2026-11-13 · v0.53.0 no shell change (the version only); 2026-11-09 · v0.52.2 no shell change (the version only); 2026-11-09 · v0.52.1 no shell change (the version only); 2026-11-09 · v0.52.0 no shell change (the version only); 2026-11-09 · v0.51.0 the native Help menu lists the three demos (cmd:demo.editor, cmd:demo.views, cmd:demo.publish through __formmdRun; see web/HASHOUT_LOG.md #src-ui-demos-ts); 2026-11-03 · v0.47.8 the bundle names the publisher and the licence; 2026-10-22 · v0.46.1: desktop.yml and release.yml removed, formcompiler builds the app; v0.46.0: version only (Home's Templates side panel); v0.45.3: icons/* and the splash icon from the approved icon with the clean edge; v0.45.2: icons/* and the splash icon rebuilt without the light rim; v0.45.1: the owner's app icon in icons/* and on the splash; v0.45.0: version only; v0.44.0: version only (the menubar's route.home, templates.open and transfer.import commands now exist in the client); Phase 18: the desktop app

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

### .github/workflows/desktop.yml and release.yml removed · formcompiler builds and releases
- refers-to: .github/workflows/desktop.yml (removed), .github/workflows/release.yml (removed), .github/workflows/ci.yml (push branches)
- kind: removed-code
- logged: v0.46.1 · 2026-10-22

Every form* app is now built in one place, formcompiler (`nodemode-ai/formcompiler`), from a `compile_vX.Y.Z` branch cut from main's head. It drafts the release as `form.md-vX.Y.Z` on its own release page and holds the signing identities, so neither tag workflow stays: release.yml published the server release directly and desktop.yml attached the desktop bundles to it. ci.yml keeps its one job and also runs on `compile_v*` pushes. Both workflows already ran this repository's own commands, which formcompiler repeats with Go from go.mod, Node 22.22.0, `npm ci --no-audit --no-fund` in `web/` and, for the desktop app, Rust stable and on Linux `libwebkit2gtk-4.1-dev libayatana-appindicator3-dev librsvg2-dev libxdo-dev libssl-dev patchelf rpm`:

```text
release.yml, job binaries (ubuntu-24.04), on tags v[0-9]+.[0-9]+.[0-9]+:
  test "$(sed -n 's/^VERSION := //p' Makefile)" = "$GITHUB_REF_NAME"
  make check
  make release
  cp install.sh deploy/form-md.service release/
  gh release create "$GITHUB_REF_NAME" release/* --title "form.md $GITHUB_REF_NAME" --generate-notes
release.yml, job image: docker buildx for linux/amd64,linux/arm64 with build-arg VERSION=<tag>, pushed as
  ghcr.io/nodemode-ai/form.md:<tag> and ghcr.io/nodemode-ai/form.md:latest
desktop.yml, matrix macos-latest, macos-15-intel, ubuntu-22.04, ubuntu-22.04-arm, windows-latest:
  [ -n "$APPLE_CERTIFICATE" ] || unset APPLE_CERTIFICATE APPLE_CERTIFICATE_PASSWORD APPLE_SIGNING_IDENTITY
  [ -n "$APPLE_ID" ] || unset APPLE_ID APPLE_PASSWORD APPLE_TEAM_ID
  ./build_app --yes            (build_app.bat on Windows)
  artifacts: src-tauri/target/release/bundle/{dmg/*.dmg,appimage/*.AppImage,deb/*.deb,rpm/*.rpm,msi/*.msi,nsis/*.exe}
desktop.yml, job publish: waits up to ten minutes for the release, then gh release upload --clobber of every bundle
```

The update check and the installer now read formcompiler's rolling `form.md-updates` release (see internal/HASHOUT_LOG.md, update check).

### .github/workflows/desktop.yml · every platform on a tag (until v0.46.0)
- refers-to: .github/workflows/desktop.yml
- kind: explainer
- logged: v0.43.0 · 2026-10-20

Five runners (macOS arm64 and Intel, Linux x64 and arm64, Windows x64) build on each `v*.*.*` tag and attach the .dmg, .deb, .rpm, .AppImage, .msi and setup .exe to the release that release.yml creates, waiting up to ten minutes for it to exist. Apple signing and notarising switch on when the APPLE_* secrets are set; without them a downloaded macOS build is unsigned and macOS blocks its first launch until it is allowed under System Settings, Privacy and Security, Open Anyway. An app built on the Mac itself with `./build_app` carries no download quarantine and opens directly.

## The app icon on the splash {#splash-icon}

### src-tauri/splash/index.html and icons/* · the owner's # icon
- refers-to: src-tauri/splash/index.html, src-tauri/splash/icon.png; src-tauri/icons/*
- kind: fix
- logged: v0.45.1 · 2026-10-20

The icon files are the owner's redesign (see web/HASHOUT_LOG.md, app-icon-family). The splash drew the old F as inline SVG; it now shows `icon.png` (the 256 px icon, a copy of `icons/128x128@2x.png`) at 64 px, since `splash` is the frontendDist and is bundled whole. The window stays 360 by 180; the icon, label and bar fit with room.

### src-tauri/src/reveal.rs, splash/index.html, tauri.conf.json, build.rs, capabilities/main.json · formvis's splash and reveal
- refers-to: src-tauri/src/reveal.rs; src-tauri/splash/index.html; src-tauri/tauri.conf.json :: splash window; src-tauri/build.rs; src-tauri/capabilities/main.json; src-tauri/src/main.rs :: `main`, `open_main`
- kind: explainer
- logged: v0.49.2 · 2026-11-07

The splash is the family card from formvis: transparent html, the body is a 16 px rounded card (#0a0a0a, 1 px rgba(255,255,255,0.09) border), the app icon at 72 px, "Starting form.md" and a 180 x 3 bar filled with the family gradient on a 2.6 s curve. `window.formmdSplashDone` finishes the bar. The splash window is built hidden and `reveal::round_splash` shows it: on macOS it first makes the window non-opaque with a clear background and masks the content layer to a 16 px corner radius (so no square shows behind the card), elsewhere it only shows it. `macOSPrivateApi` was already on.

The main window is built with the page's dark background (10,10,10), visible only off macOS (on macOS `conceal` puts it on screen at alpha 0 with mouse events ignored, because a hidden window's web view stops its animation frames). The splash is no longer closed when the window is built; `reveal::show_main` runs when the page calls `app_ready` (or after a 20 s fallback), waits until 2.6 s after launch, finishes the bar for 220 ms, shows the main window whole and closes the splash. Failure paths (`fail`) still close the splash.

The main window shows the server's page from 127.0.0.1, a remote origin to Tauri, so `capabilities/main.json` lists that address for the main window only and allows one permission, `allow-app-ready`, generated by `build.rs` (the app manifest). The splash and main windows keep `core:default` from `default.json` only for the splash; the page gets no other command. Unlike formvis there is no traffic-light placement here, and the objc2 crates sit under the macOS target in Cargo.toml.

## Media playback on Linux {#linux-media}

### src-tauri/tauri.conf.json · GStreamer for the meeting recorder
- refers-to: src-tauri/tauri.conf.json :: bundle.linux (deb.recommends, rpm.recommends, appimage.bundleMediaFramework); formcompiler apps/form.md.json :: toolchain.linux_packages
- kind: rationale
- logged: v0.47.7 · 2026-10-31

The meeting recorder (`web/src/extensions/blocks/meeting.ts`, MediaRecorder) runs in WebKitGTK on Linux, which records and plays through GStreamer. The config now follows formdeck: the deb recommends `gstreamer1.0-plugins-base`, `gstreamer1.0-plugins-good` and `gstreamer1.0-libav`, the rpm recommends `gstreamer1-plugins-base` and `gstreamer1-plugins-good`, and the AppImage bundles the media framework. formcompiler installs the base and good sets on the Linux runner so the AppImage step finds them. apt and dnf install recommends by default; `dpkg -i` and `--no-install-recommends` skip them. On Ubuntu 24.04 `libwebkit2gtk-4.1-0` already depends on base and good but only suggests `gstreamer1.0-libav` (the H.264 and AAC decoders), so the deb recommends it. The AppImage carries base and good only, since libav would copy the runner's own FFmpeg into the app. Checked: the config parses and formcompiler's `registry.py check` passes; no Linux build was run here.

## The publisher and the licence in the bundle {#publisher}

### tauri.conf.json, Cargo.toml · publisher, copyright, license, licenseFile, authors
- refers-to: src-tauri/tauri.conf.json :: bundle.publisher, bundle.copyright, bundle.license, bundle.licenseFile; src-tauri/Cargo.toml :: authors
- kind: decision
- logged: v0.47.8 · 2026-11-03

The launch punch list (2026-11-03): the About of a downloaded app says NodeAgentc, Inc., not nodemode-ai, and lists the licence. The window's About is the admin dialog's About tab (`web/src/ui/admin.ts`); the macOS application menu's About panel reads the bundle's copyright, and Tauri's default About metadata takes its authors from `bundle.publisher`. The bundle now names the publisher, carries the SPDX id in its copyright line as formMCP and formskills already did, and sets `license` and `licenseFile` (`../LICENSE` from `src-tauri`); `authors` is the company, which is the .deb's Maintainer, and tauri-build writes `CompanyName` from `bundle.publisher` into the Windows version resource. Checked here: the JSON parses; no desktop build was run.

## src/menu.rs · Extensions menu {#menu-extensions}

- refers-to: src/menu.rs :: `build` (`extensions` submenu)
- kind: feature
- logged: v0.50.0 · 2026-11-07

A top-level Extensions menu between View and Window, per the Form Extension Contract (section 8): Extension Commands... (`extensions.menu`, which opens the window's own Extensions menu listing every contributed command), Extensions Panel, Reload Extensions and Manage in FormPortal. The native bar is built once at start, so the per-extension command items live in the window's menu and the command palette instead of the native menu; rebuilding the native submenu from the window would need a Tauri command and a remote IPC permission, left for later. Not built here (no WebKitGTK in this environment); the items use the same `item` helper and `cmd:` ids as the rest of the bar.

## src/main.rs · a failed start is said before the last window goes, and the server is told who started it {#windows-start}

- refers-to: src/main.rs :: `fail`, `start`
- kind: fix
- logged: v0.57.0 · 2026-11-19

`fail` closed the loading window and then asked for the dialog. When the start fails there is no main window yet, so the loading window is the last one, and closing it first can end the app before the dialog is seen: a silent exit. The dialog is now asked for first, and the loading window is closed and the app exits only in the dialog's own callback, after the person has answered it.

`start` gives the server `FORMMD_SHELL_PID`, this process's id. The server (internal/startup, see internal/HASHOUT_LOG.md #p29-windows-start) writes it into its lock record and stops when that process is gone. This is beside what was already there and is kept: the server stops when its standard input closes, and the app closes that pipe when it quits (`stop_server` drops the child).

Not done, and why. No job object: the family's `win_job.rs` takes a `std::process::Child`, and this app starts its server through tauri-plugin-shell, which gives none; holding the server in a job needs either the start moved to `std::process::Command` or a version of that file that opens the process by its id, and neither was written or type-checked. `engine_start.rs` (the last lines of the server's output in the dialog, a limit on the wait for the address) was not brought in: the dialog still shows the last line the server logged at level ERROR, which for a second start is the sentence "form.md is already running on [folder] (engine process N)." inside the server's log line. The wait for the address has no limit of its own; it ends when the server exits.

Not built and not run: this crate cannot be compiled here (no WebKitGTK), and nothing ran on Windows or macOS. The two edits were read against the types in use in the same file: `Command::env` already takes a `&String` and an `&OsStr` three lines above, and `close_splash` takes `&AppHandle`, which the callback owns as `quit`.

## src/win_job.rs, src/engine_start.rs, src/main.rs · the server ends with the app on Windows, a failed start says what the server said, and the wait for the address has a limit {#engine-start}

- refers-to: src/win_job.rs :: `hold_pid`, `end`; src/engine_start.rs :: `Tail`, `Failure`, `message`, `keep`, `ADDRESS_WAIT`; src/main.rs :: `start`, `fail_start`, `start_limit`, `stop_server`
- kind: fix
- logged: v0.58.0 · 2026-11-20

This closes what the entry above left out. The server is still started through tauri-plugin-shell, the way FormVis does it, so nothing about how it starts or how its output is read changed: `win_job.rs` is FormVis's file without its `hold(&Child)`, since this app has no `std::process::Child`, and `hold_pid` opens the server by the process id the plugin gives and puts it in a job object made with "kill on job close". When the app ends in any way on Windows, including from Task Manager, the system closes the job and the server goes with it. `stop_server` calls `end` before its own `taskkill`, which stays for the case where the job could not be made or the server could not be put in it.

`start` now reads the plugin's events through a second channel that also carries three timers (`Late`): the wait for the address (`ADDRESS_WAIT`, 90 seconds, the owner's figure for every app), one second after the server's exit for its last lines to arrive (`EXIT_WAIT`), and the end of the server's output. Every line the server writes on either stream before the window opens goes into a `Tail` (the newest 200, control characters removed, 300 characters at most each). A failed start shows one dialog with a sentence for the cause (could not start, stopped with its exit code, another copy holds the data folder for exit code 2, waited 90 seconds, the window could not be built), the last twelve lines, and the path of `form-md-server.log` in the app's log folder, which `keep` writes with the kept lines because the server itself logs only to its standard error. The old dialog showed only the last line logged at level ERROR. A stop after the window opened keeps its own sentence. The `stopped` flag is now set by the task that forwards the plugin's events, so `stop_server` still sees the exit after the reading loop has ended.

`SHELL_ENV` names `FORMMD_SHELL_PID`, the variable internal/startup watches. `windows-sys` 0.61.2 (Windows only) is the version 22 packages in the lock already use; the one line in the lock was added by hand.

Proved: the lock hand edit by `cargo update --workspace --offline` on a copy of `Cargo.toml` and the old lock with stub sources (its lock and ours do not differ); `win_job.rs` and `engine_start.rs` in the scratch crate with `cargo check --target x86_64-pc-windows-msvc`, and the five tests of `engine_start.rs` with `cargo test`. Not proved: `main.rs` itself, which cannot be compiled here (no WebKitGTK); it was read against FormVis's `main.rs` for every name and type it uses. Nothing ran on Windows.

## src/main.rs · the window's key comes from the server, once, over the server's own output {#window-key}

- refers-to: src/main.rs :: `KEY_LINE`, `window_key`, `start`, `open_main`
- kind: fix
- logged: v0.61.0 · 2026-11-20

The app no longer makes the key its window signs in with and no longer hands it to the server in `FORMMD_DESKTOP_KEY`. It starts the server with `FORMMD_DESKTOP=1` (no secret), and the server makes the key itself and prints one line, `form-md-window-key: <key>`, on its standard output before the ready line. That output is the pipe tauri-plugin-shell gives this app only. `window_key` takes the line when the key is at least 32 characters of `A-Z a-z 0-9 - _`; the line is not put in the `Tail`, so a failure dialog and `form-md-server.log` never carry it. The window opens at `/desktop/enter?key=...` as before, and the server accepts that key once per start. A ready line with no key before it fails the start with "the server gave no key for its window". The entry at the top of this file (the start of the server) still describes the old key; this entry replaces that part. `getrandom` is now unused here and stays in `Cargo.toml` until a build can prove the lock without it.

Not proved: the shell cannot be compiled here (no WebKitGTK, and this session has no cargo lane); the orchestrator type checks it for Windows and macOS. The server's side ran: see internal/HASHOUT_LOG.md, Phase 32.
