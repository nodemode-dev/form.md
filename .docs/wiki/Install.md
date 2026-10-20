# Install

form.md is one binary, `form-md`, that serves the web client, the API and the realtime sync socket from one port. It needs a writable data directory and nothing else. There are five ways to get it running: the desktop app, the installer script (Linux with systemd, or macOS without a service), the release binary by hand, the container image, or a build from source.

## Desktop app

Since v0.43.0 each release also carries the desktop app: `form.md_X.Y.Z_aarch64.dmg` and `_x64.dmg` for macOS 12 or newer, a `.deb`, `.rpm` and `.AppImage` for Linux on x64 and arm64, and an `.msi` and setup `.exe` for Windows 10 or newer. Build it yourself with `./build_app` (macOS and Linux) or `build_app.bat` (Windows); the packages land in `src-tauri/target/release/bundle`.

The app is a window around the same `form-md` binary. It starts the server on this computer when it opens and stops it when you quit, signs you in as the owner without a password screen, and keeps your data in `~/form.md` (start the app with `FORMMD_DATA` set to use another folder). The server listens on `127.0.0.1:8750` so agents and `form-md mcp` find it at the usual address, or on another free port when 8750 is taken. It never listens beyond this computer; to share a workspace with other people, run the server on its own as below.

A macOS build downloaded from a release is unsigned unless the release workflow had Apple signing secrets: allow it once under System Settings, Privacy and Security, Open Anyway.

## Layout

| Path | What lives there |
|---|---|
| `install.sh` | The installer: download, checksum, install, and on Linux a systemd service |
| `deploy/form-md.service` | The systemd unit the installer puts in place (published with each release) |
| `Dockerfile` | Three-stage image build: client, server, distroless runtime |
| `Makefile` | `build`, `release`, `check` and their parts |
| `cmd/form-md/main.go` | `serve` (the default command), `version`, `help` |
| `cmd/form-md/tool.go` | `keygen`, `encrypt`, `backup`, `restore`, `doctor`, `admin reset-owner` |

## Ports

| Port | Service | Notes |
|---|---|---|
| 8750 | `form-md serve` | The server. Default listen address `127.0.0.1:8750` (loopback only); set `FORMMD_ADDR=0.0.0.0:8750` to accept other machines |
| 5750 | Vite dev server (development only) | Proxies `/api`, `/healthz`, `/notices`, `/events`, `/mcp`, `/files`, `/oauth`, `/.well-known` and the `/sync` WebSocket to `127.0.0.1:8750` |
| 5751 | `npm run preview` in `web/` (development only) | Serves a built client |

## The installer

The installer is `install.sh` at the repo root, and the release workflow attaches it to every release. Download it, read it, run it:

```sh
curl -fsSL -o install.sh https://github.com/nodemode-ai/form.md/releases/download/vX.Y.Z/install.sh
sh install.sh --version=vX.Y.Z
```

Without `--version` it asks the GitHub API for the latest release.

What it checks, in order, and stops on:

1. The system is `linux` or `darwin`, and the processor is `x86_64`/`amd64` or `aarch64`/`arm64`. Anything else stops with a pointer to the releases page.
2. `curl` or `wget` is present.
3. The version looks like `v1.2.3`.
4. `sha256sum` or `shasum` is present. It downloads `form-md_vX.Y.Z_<os>_<arch>` and `SHA256SUMS`, and stops with "checksum mismatch ... nothing was installed" if they disagree.
5. The downloaded binary runs (`form-md version`).
6. `sudo` is present when you are not root (not needed with `--no-service` and a writable bin directory).

Then it installs the binary as `$BIN_DIR/form-md` with mode 0755.

| Flag | Effect |
|---|---|
| `--version=vX.Y.Z` | Install that release instead of the latest |
| `--no-service` | Install the binary only; prints `run: FORMMD_DATA=./data <bin dir>/form-md` |
| `-h`, `--help` | Print usage |

| Installer variable | Default | Purpose |
|---|---|---|
| `FORMMD_VERSION` | `latest` | Same as `--version` |
| `FORMMD_BIN_DIR` | `/usr/local/bin` | Where the binary goes |
| `FORMMD_DATA_DIR` | `/var/lib/form-md` | Data directory for the service |
| `FORMMD_CONF_DIR` | `/etc/form-md` | Where the `env` file goes |
| `FORMMD_DOWNLOAD_URL` | the GitHub release for the version | Base URL to download the binary, `SHA256SUMS` and `form-md.service` from (a mirror). Set `FORMMD_VERSION` with it: "latest" is not looked up when this is set |

These five are read by the installer only; the server never reads them. For example, a binary-only install into your own bin directory, without sudo:

```sh
FORMMD_BIN_DIR=$HOME/.local/bin sh install.sh --version=vX.Y.Z --no-service
```

### The service it creates

A service is set up only on Linux with `systemctl` and a running systemd (`/run/systemd/system` exists), and not with `--no-service`. Otherwise it stops after installing the binary.

1. Creates a system user `form-md` with its home at the data directory.
2. Creates the data directory (owned by `form-md`, mode 0700) and the config directory.
3. Writes `/etc/form-md/env` with one line, `FORMMD_ADDR=127.0.0.1:8750`, mode 0640. An existing file is left alone.
4. Downloads `form-md.service` from the release (this is `deploy/form-md.service`) and rewrites its paths to your bin, data and config directories; if the download fails it writes a minimal unit with the same directories.
5. Installs it as `/etc/systemd/system/form-md.service`, runs `systemctl daemon-reload`, `systemctl enable form-md.service` and `systemctl restart form-md.service`. Because it restarts, running the installer again with a new `--version` upgrades a running install.
6. Checks that the service is active, then prints `open http://127.0.0.1:8750 to create the owner account`. If it is not active it stops with `see: journalctl -u form-md -n 50`.

With the default directories, the unit from `deploy/form-md.service` runs `/usr/local/bin/form-md serve` as `form-md`, reads `/etc/form-md/env`, sets `FORMMD_DATA=/var/lib/form-md`, restarts on failure, and hardens the process: `ProtectSystem=strict` with `ReadWritePaths=/var/lib/form-md`, `ProtectHome=true`, `PrivateTmp=true`, `UMask=0077`, `NoNewPrivileges=true`. It grants `CAP_NET_BIND_SERVICE` so the server can bind ports 443 and 80 for [[TLS-and-ACME]]. `systemctl reload form-md` sends SIGUSR1, which makes the server save its open documents.

### After installing

```sh
systemctl status form-md
curl -fsS http://127.0.0.1:8750/healthz
```

`/healthz` answers `{"status":"ok","product":"form.md","version":"...","storage":"ok"}`. Open `http://127.0.0.1:8750` (through an SSH tunnel if the server is remote) and the set-up screen creates the owner account. Everyone else joins by invitation unless sign-up is opened in Instance settings. If people will reach the instance through an HTTPS reverse proxy, add `FORMMD_TRUST_PROXY=true` to `/etc/form-md/env` and restart before the first set-up; see [[TLS-and-ACME]].

To change the address, TLS or encryption, edit `/etc/form-md/env` (see [[Configuration]]) and restart:

```sh
sudoedit /etc/form-md/env
sudo systemctl restart form-md
```

## Running the binary by hand

Each release carries `form-md_vX.Y.Z_<os>_<arch>` for `linux` and `darwin` on `amd64` and `arm64`, `form-md_vX.Y.Z_windows_amd64.exe`, and `SHA256SUMS`.

```sh
sha256sum -c --ignore-missing SHA256SUMS
chmod +x form-md_vX.Y.Z_linux_amd64
FORMMD_DATA=./data ./form-md_vX.Y.Z_linux_amd64
```

With no command the binary runs `serve`. `form-md help` prints every command and every environment variable with its default. The data directory is created (mode 0700) on first start.

## Docker

The `Dockerfile` builds the client with Node 22, the server with Go (`CGO_ENABLED=0`), and copies the binary into `gcr.io/distroless/static-debian12:nonroot`. The image sets `FORMMD_DATA=/data` and `FORMMD_ADDR=0.0.0.0:8750`, declares `/data` as a volume, exposes 8750, runs as user 65532, and its command is `serve`. The release workflow pushes `ghcr.io/nodemode-ai/form.md:vX.Y.Z` and `:latest` for `linux/amd64` and `linux/arm64`.

```sh
docker run -d --name form-md --restart unless-stopped \
  -p 127.0.0.1:8750:8750 \
  -v form-md-data:/data \
  ghcr.io/nodemode-ai/form.md:vX.Y.Z
```

Build the image yourself from a checkout:

```sh
docker build --build-arg VERSION=vX.Y.Z -t form-md .
```

Other commands run against the same volume, for example `docker exec form-md /usr/local/bin/form-md doctor`. Backups from a container are covered in [[Backup-and-restore]].

## Building from source

Requirements: Go 1.26 or newer, Node.js 22 or newer.

```sh
make build        # web client, NOTICES.md, then bin/form-md
./bin/form-md
```

| Target | What it does |
|---|---|
| `make build` | `web` (runs `web/oc_install` if `web/node_modules` is missing, then `npm run build`, which type checks and builds into `web/dist`), `notices` (regenerates `NOTICES.md`), `server` (builds `bin/form-md` with the version from the Makefile) |
| `make release` | Builds the client and notices, then cross-compiles `release/form-md_<VERSION>_<os>_<arch>` for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and windows/amd64, and writes `release/SHA256SUMS` |
| `make check` | `gofmt -l` (fails on any unformatted file), `go vet ./...`, `npm run typecheck` in `web/` |
| `make clean` | Removes `bin/`, `release/` and the built client |

`VERSION` comes from the top of the `Makefile`; override it with `make build VERSION=v0.0.0-dev`. The client is embedded from `web/dist` at build time, so a binary built without `make web` answers every page with "the web client is not built". For the two-process development loop on ports 5750 and 8750, see [[Development]].

## Gotchas

- Running the installer again replaces `/etc/systemd/system/form-md.service` and restarts the service. Put your own changes in a drop-in (`sudo systemctl edit form-md`), which lives beside it and survives.
- The default listen address is loopback only. A server that "does not answer" from another machine usually needs `FORMMD_ADDR` or a reverse proxy.

## Related

[[Configuration]] · [[TLS-and-ACME]] · [[Operations]] · [[Development]]
