# HASHOUT_LOG · desktop shell · form.md

Living log of explainers, rationale and removed code for the desktop app in `src-tauri/` and its build scripts (`build_app`, `build_app.bat`, `tools/build_desktop`; `.github/workflows/desktop.yml` until v0.46.0). Source files stay comment-free per dev-rules `hashout-log`; what they would have said lives here. Paths are relative to the repo root.

Current build version: v0.46.1
Last updated: 2026-10-22 · v0.46.1: desktop.yml and release.yml removed, formcompiler builds the app; v0.46.0: version only (Home's Templates side panel); v0.45.3: icons/* and the splash icon from the approved icon with the clean edge; v0.45.2: icons/* and the splash icon rebuilt without the light rim; v0.45.1: the owner's app icon in icons/* and on the splash; v0.45.0: version only; v0.44.0: version only (the menubar's route.home, templates.open and transfer.import commands now exist in the client); Phase 18: the desktop app

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

## Media playback on Linux {#linux-media}

### src-tauri/tauri.conf.json · GStreamer for the meeting recorder
- refers-to: src-tauri/tauri.conf.json :: bundle.linux (deb.recommends, rpm.recommends, appimage.bundleMediaFramework); formcompiler apps/form.md.json :: toolchain.linux_packages
- kind: rationale
- logged: v0.47.7 · 2026-10-31

The meeting recorder (`web/src/extensions/blocks/meeting.ts`, MediaRecorder) runs in WebKitGTK on Linux, which records and plays through GStreamer. The config now follows formdeck: the deb recommends `gstreamer1.0-plugins-base`, `gstreamer1.0-plugins-good` and `gstreamer1.0-libav`, the rpm recommends `gstreamer1-plugins-base` and `gstreamer1-plugins-good`, and the AppImage bundles the media framework. formcompiler installs the base and good sets on the Linux runner so the AppImage step finds them. apt and dnf install recommends by default; `dpkg -i` and `--no-install-recommends` skip them. On Ubuntu 24.04 `libwebkit2gtk-4.1-0` already depends on base and good but only suggests `gstreamer1.0-libav` (the H.264 and AAC decoders), so the deb recommends it. The AppImage carries base and good only, since libav would copy the runner's own FFmpeg into the app. Checked: the config parses and formcompiler's `registry.py check` passes; no Linux build was run here.
