#!/bin/sh
set -eu

REPO="nodemode-ai/form.md"
VERSION="${FORMMD_VERSION:-latest}"
BIN_DIR="${FORMMD_BIN_DIR:-/usr/local/bin}"
DATA_DIR="${FORMMD_DATA_DIR:-/var/lib/form-md}"
CONF_DIR="${FORMMD_CONF_DIR:-/etc/form-md}"
SERVICE=1

for arg in "$@"; do
  case "$arg" in
    --no-service) SERVICE=0 ;;
    --version=*) VERSION="${arg#--version=}" ;;
    -h|--help)
      echo "usage: install.sh [--version=vX.Y.Z] [--no-service]"
      echo "env: FORMMD_VERSION FORMMD_BIN_DIR FORMMD_DATA_DIR FORMMD_CONF_DIR FORMMD_DOWNLOAD_URL"
      exit 0 ;;
    *) echo "install.sh: unknown option $arg" >&2; exit 2 ;;
  esac
done

say() { printf 'form-md: %s\n' "$*"; }
die() { printf 'form-md: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "this installer needs $1"; }

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *) die "unsupported system $os; download a release by hand from https://github.com/$REPO/releases" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "unsupported processor $arch" ;;
esac

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL --retry 3 -o "$2" "$1"; }
  fetch_text() { curl -fsSL --retry 3 "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -q -O "$2" "$1"; }
  fetch_text() { wget -q -O - "$1"; }
else
  die "this installer needs curl or wget"
fi

if [ "$VERSION" = "latest" ] && [ -z "${FORMMD_DOWNLOAD_URL:-}" ]; then
  VERSION=$(fetch_text "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\(v[0-9][0-9.]*\)".*/\1/p' | head -n 1)
  [ -n "$VERSION" ] || die "could not find the latest release; set FORMMD_VERSION=vX.Y.Z"
fi
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) die "version must look like v1.2.3, got $VERSION" ;;
esac
BASE="${FORMMD_DOWNLOAD_URL:-https://github.com/$REPO/releases/download/$VERSION}"
asset="form-md_${VERSION}_${os}_${arch}"

if command -v sha256sum >/dev/null 2>&1; then
  sum() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  sum() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  die "this installer needs sha256sum or shasum to verify the download"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM
say "downloading $asset"
fetch "$BASE/$asset" "$tmp/form-md"
fetch "$BASE/SHA256SUMS" "$tmp/SHA256SUMS"
want=$(awk -v f="$asset" '$2 == f || $2 == "*" f { print $1 }' "$tmp/SHA256SUMS")
[ -n "$want" ] || die "SHA256SUMS has no entry for $asset"
got=$(sum "$tmp/form-md")
[ "$want" = "$got" ] || die "checksum mismatch for $asset (expected $want, got $got); nothing was installed"
say "checksum verified"
chmod 0755 "$tmp/form-md"
"$tmp/form-md" version >/dev/null || die "the downloaded binary does not run on this machine"

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  if [ -w "$BIN_DIR" ] && [ "$SERVICE" -eq 0 ]; then SUDO=""; else need sudo; SUDO="sudo"; fi
fi
$SUDO mkdir -p "$BIN_DIR"
$SUDO install -m 0755 "$tmp/form-md" "$BIN_DIR/form-md"
say "installed $BIN_DIR/form-md $("$BIN_DIR/form-md" version)"

if [ "$SERVICE" -eq 0 ] || [ "$os" != "linux" ] || ! command -v systemctl >/dev/null 2>&1 || [ ! -d /run/systemd/system ]; then
  say "no service set up; run: FORMMD_DATA=./data $BIN_DIR/form-md"
  exit 0
fi

if ! id form-md >/dev/null 2>&1; then
  $SUDO useradd --system --home-dir "$DATA_DIR" --shell /usr/sbin/nologin form-md 2>/dev/null \
    || $SUDO adduser --system --home "$DATA_DIR" --no-create-home --group form-md
fi
$SUDO mkdir -p "$DATA_DIR" "$CONF_DIR"
$SUDO chown form-md:form-md "$DATA_DIR"
$SUDO chmod 0700 "$DATA_DIR"
if [ ! -f "$CONF_DIR/env" ]; then
  printf 'FORMMD_ADDR=127.0.0.1:8750\n' | $SUDO tee "$CONF_DIR/env" >/dev/null
  $SUDO chmod 0640 "$CONF_DIR/env"
fi
fetch "$BASE/form-md.service" "$tmp/form-md.service" 2>/dev/null || cat > "$tmp/form-md.service" <<UNIT
[Unit]
Description=form.md
After=network-online.target
Wants=network-online.target

[Service]
User=form-md
Group=form-md
EnvironmentFile=-$CONF_DIR/env
Environment=FORMMD_DATA=$DATA_DIR
WorkingDirectory=$DATA_DIR
ExecStart=$BIN_DIR/form-md serve
Restart=on-failure
RestartSec=2

[Install]
WantedBy=multi-user.target
UNIT
$SUDO install -m 0644 "$tmp/form-md.service" /etc/systemd/system/form-md.service
$SUDO systemctl daemon-reload
$SUDO systemctl enable --now form-md.service
sleep 1
if $SUDO systemctl is-active --quiet form-md.service; then
  say "form-md is running; it starts again after a reboot"
  say "open http://127.0.0.1:8750 to create the owner account, then see $CONF_DIR/env for the address, TLS and encryption"
else
  die "the service did not start; see: journalctl -u form-md -n 50"
fi
