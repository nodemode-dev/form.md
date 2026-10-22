# Operations

Day-to-day running of an instance: checking its health with `form-md doctor`, getting the owner back in with `form-md admin reset-owner`, knowing what is in the data directory, reading the logs, and upgrading. Every command reads the same `FORMMD_` variables as the server, so run it with the server's `FORMMD_DATA` (and key, when [[Encryption]] is on).

## Layout

| Path | What lives there |
|---|---|
| `cmd/form-md/tool.go` | Dispatches `keygen`, `encrypt`, `backup`, `restore`, `doctor`, `admin reset-owner` |
| `internal/ops/doctor.go` | Every `doctor` check |
| `internal/ops/encrypt.go` | `encrypt` and `ResetOwner` |
| `internal/ops/lock_unix.go`, `lock_other.go` | The data directory lock |
| `internal/ops/owner_unix.go` | `matchOwner`: after a command run as root, gives every file in the data directory to the directory's owner |
| `internal/ops/flush_unix.go` | SIGUSR1: save open documents |
| `internal/app/health.go` | `GET /healthz` |
| `internal/app/update.go` | The daily update check |
| `internal/ops/memory.go` | Memory limit from the container |
| `internal/app/notices.go` | `GET /notices` and `GET /licenses` |
| `deploy/fail2ban/` | `form-md.conf` (filter) and `form-md.local` (jail) |
| `internal/store/migrate.go`, `internal/store/migrations/` | Numbered schema migrations, run on start |

## Running commands on a systemd install

The service reads `/etc/form-md/env` (readable by root) and sets `FORMMD_DATA=/var/lib/form-md`. A small shell function gives a command the same environment:

```sh
fm() { sudo sh -c 'set -a; . /etc/form-md/env; FORMMD_DATA=/var/lib/form-md; exec /usr/local/bin/form-md "$@"' fm "$@"; }
fm doctor
```

If the key comes from a systemd credential rather than `FORMMD_KEY_FILE` in the env file, add `FORMMD_KEY_FILE=/etc/form-md/form-md.key;` before `exec`. Commands run this way run as root. On Unix, `backup`, `restore`, `encrypt`, `admin reset-owner`, `admin set` and `doctor` then hand every file in the data directory back to the directory's owner when they finish, so the service user can still write everything.

## form-md doctor

Checks the data directory, database, files and settings, prints one line per check (`ok`, `warn` or `fail`), and exits with status 1 if any check failed. It is safe to run while the server is up.

| Check | Reports |
|---|---|
| `data directory` | `fail` if it does not exist (stops here); `warn` if group or others have any permission on it (`chmod 700` it) |
| `server` | Whether something listens on `FORMMD_ADDR` (`likely the server`) or the address is free (`not running`) |
| `encryption` | `fail` with the key error (stops here, see [[Encryption]]); `warn` when encryption is off; `ok on, key PATH matches` |
| `encryption key` | `warn` when the key file is inside the data directory |
| `database` | `warn` when there is no database yet (stops here); `fail` when it cannot be opened or `PRAGMA quick_check` is not `ok` (stops here) |
| `schema` | `ok migration N (current)`; `ok migration N; M more will run on the next start`; `fail` when the database is newer than this binary knows |
| `database` | `warn` when `form-md.db-wal` is over 256 MB (a long-running reader may be blocking checkpoints), otherwise `ok integrity ok` |
| `files` | `fail` when files recorded in the database are missing from `files/`; `ok N stored files present`; `warn` for files on disk with no record (the hourly sweep removes them after a day); `fail` for unencrypted files in an encrypted instance |
| `secret key` | `warn` when `secret.key` is missing (saved provider keys and SSO secrets cannot be read); `fail` when it cannot be read; `ok readable` |
| `mail` | Reads the mail mode: `warn` when mail is off (invitations and inbox notices are not sent by mail); `warn` when the mode is SMTP but no SMTP server is set; otherwise `ok mail mode <mode>` |
| `tls` | `ok automatic HTTPS for <domain> from Instance settings` when `https.auto` is on with a domain and no TLS is set in the environment; `ok ACME for ...`; for `FORMMD_TLS_CERT`, `fail` if the pair does not load, `warn` within 14 days of expiry; `warn off` when none of these is set |

## form-md admin set

Writes one instance setting straight into the database, without signing in:

```sh
fm admin set https.auto false
fm admin set net.trust_proxy true
```

It prints `KEY = VALUE; a running server picks this up at its next start`, so restart the server afterwards (it keeps settings in memory). The key must be one of the instance settings in `internal/settings/settings.go` and the value is checked the same way as in Instance settings (`expected true or false`, a number range, or one of the allowed values). Secret settings are sealed as usual. For every secret setting (those of the Secret kind in `internal/settings/settings.go`), the value is printed as `(saved)`. Use it as the way out when a setting keeps the server from starting or from being reached, for example automatic HTTPS ([[TLS-and-ACME]]) or `net.trust_proxy` before the first set-up.

## form-md admin reset-owner

For an owner locked out of their account. Stop the server first: the command takes the data directory lock and refuses with `the data directory is in use by a running server: stop it first` while one runs, so no editor, terminal or event stream stays attached to a session it signs out.

```sh
fm admin reset-owner
```

It sets a new random password for the instance owner, re-enables the account if it was disabled, turns two-step sign-in off, and signs out every session of that account. It prints:

```
owner owner@example.com: new password 3f9c...
two-step sign-in was turned off and every session signed out; set a new password after signing in
```

## The data directory

| Name | What it is |
|---|---|
| `form-md.db` | The SQLite database: pages, documents, databases, users, settings, the audit log |
| `form-md.db-wal` | SQLite write-ahead log; checkpointed and truncated on a clean shutdown |
| `form-md.db-shm` | SQLite's WAL index |
| `files/` | Uploads, one file per upload named by its id and extension; `.upload-*` are uploads in progress |
| `tmp/` | Import uploads waiting to be processed (`.import-*`) |
| `secret.key` | 64 hex characters that seal the secrets saved in Instance settings; created on first start |
| `encryption.json` | Present only when the directory is encrypted; holds a key check, never the key |
| `acme/` | Certificate cache, only with ACME |
| `form-md.lock` | The data directory lock; holds the server's process id |
| `form-md.flushed` | Time of the last save requested by SIGUSR1 (Unix) |
| `.download-*` | Temporary folder for a backup being downloaded from the browser; removed afterwards |
| `tunnel.sock` | The private socket `cloudflared` connects to (mode 0600), recreated at every start; see [[Tunnel]] |

The directory is created with mode 0700. Back up all of it with `form-md backup`, never by copying files while the server runs; see [[Backup-and-restore]].

### The lock

`serve`, `encrypt` and `restore` take an exclusive lock on `form-md.lock`. A second server on the same directory stops with `the data directory is in use by a running server: stop it first` (log line `data directory`, exit status 1). On Unix the lock is released when the process exits, even after a crash. Elsewhere the lock is the file itself: after a crash, make sure no server runs and delete `form-md.lock`.

## Health and logs

```sh
curl -fsS http://127.0.0.1:8750/healthz
```

answers `200` with `"status":"ok"` and the version, or `503` with `"status":"degraded","storage":"error"` when the database does not answer within 2 seconds. When the update check has seen a newer release, the answer also carries `"latest":"vX.Y.Z"`.

The server logs plain text lines (`time=... level=INFO msg=...`) to standard error at level info. Under systemd:

```sh
journalctl -u form-md -f
```

At start look for `form.md listening url=... version=... data=...`, and with a key `at-rest encryption on`. A start that fails logs one error line named for the stage that failed (`config`, `catalogs`, `encryption`, `data directory`, `storage`, `bootstrap`, `startup`, `server`) and exits: status 2 for `config` and `encryption`, status 1 for the others. On SIGTERM it logs `shutting down`, gives open requests 15 seconds, checkpoints the WAL and logs `stopped cleanly`.

`sudo systemctl reload form-md` sends SIGUSR1: the server saves every open document and keeps running.

`GET /notices` and `GET /licenses` both serve the third-party notices (`NOTICES.md`, embedded in the binary); Instance settings, About links to them as "Open-source licences".

## Update check

With `update.check` on (the default; Instance settings, General, Housekeeping, "Check for new versions", or `FORMMD_UPDATE_CHECK`), the server asks for the latest release 5 seconds after start and then once a day. It reads only the `tag_name` of a small JSON file: from formcompiler's rolling `form.md-updates` release (`https://github.com/nodemode-ai/formcompiler/releases/download/form.md-updates/latest.json`, where formcompiler keeps the newest version), or from `FORMMD_UPDATE_URL` when set (any address that answers the same JSON, for a mirror on a closed network). A failed check is logged as `update check failed` and tried again after 6 hours. Nothing is downloaded or installed.

When the latest release is newer than the running version:

- the log says `a newer version is available latest=... running=...`,
- `/healthz` carries `"latest"`,
- owners and admins see the version shown in the sidebar in the accent colour, with "vX.Y.Z is available" as its tooltip,
- Instance settings, About shows "<version>, and <latest> is available".

To upgrade, see Upgrading below.

## Memory

When `GOMEMLIMIT` is not set and the server runs under a memory limit (a container limit, or `MemoryMax=` on the systemd service), it sets Go's soft memory limit to 90 percent of that limit and logs `memory limit set from the container soft_limit_mb=...`. It looks first at its own cgroup (found through `/proc/self/cgroup`, cgroup v2), then at `/sys/fs/cgroup/memory.max` and `/sys/fs/cgroup/memory/memory.limit_in_bytes` (cgroup v1), and uses the first one that holds a number; with no limit anywhere (`max`) it does nothing. For example:

```sh
sudo systemctl edit form-md     # add: [Service] MemoryMax=1G
sudo systemctl restart form-md
``` Set `GOMEMLIMIT` yourself (for example `GOMEMLIMIT=900MiB` in `/etc/form-md/env`) to choose the limit, which turns this off.

## fail2ban

Every failed password, second factor or API token is logged at warning level as:

```
level=WARN msg="form-md auth failure" kind=password ip=203.0.113.7
```

`kind` is `password`, `totp` or `token`. `deploy/fail2ban/` ships a filter that matches this line in the systemd journal (`_SYSTEMD_UNIT=form-md.service`) and a jail that bans an address after 8 failures within 10 minutes for 1 hour, on ports `http`, `https` and `8750`:

```sh
sudo cp deploy/fail2ban/form-md.conf /etc/fail2ban/filter.d/form-md.conf
sudo cp deploy/fail2ban/form-md.local /etc/fail2ban/jail.d/form-md.local
sudo systemctl reload fail2ban
sudo fail2ban-client status form-md
```

The `ip` is the client address as the server sees it. Behind a reverse proxy turn on `net.trust_proxy` (see [[TLS-and-ACME]]), or every failure carries the proxy's address and fail2ban bans the proxy. Through a running [[Tunnel]] it is already the visitor's address.

## Upgrading

Migrations are embedded in the binary and run on start, each in its own transaction; there is no separate migrate step.

1. Back up: `fm backup /var/backups/form-md/before-upgrade.tgz`.
2. Install the new binary and restart:
   - with the installer, `sh install.sh --version=vX.Y.Z` installs the binary and restarts the service, which runs the migrations;
   - by hand, `sudo systemctl stop form-md`, replace `/usr/local/bin/form-md`, optionally run `fm doctor` to see `migration N; M more will run on the next start`, then `sudo systemctl start form-md`.
3. `form-md version`, `curl -fsS http://127.0.0.1:8750/healthz`, and `fm doctor` should report `schema ... (current)`.

To go back, stop the server, install the old binary and restore the backup from step 1. Do not run an old binary against a database a newer one has migrated; `doctor` reports that state as `fail schema ... newer than this binary knows (...): run the newer release`.

## The contract

1. A new operator command goes in `cmd/form-md/tool.go`, loads `config.Load()`, opens the key with `ops.OpenKey`, and takes `ops.Lock` when it must not run beside a server, as `encrypt` and `restore` do. If it can create files in the data directory, it calls `matchOwner` (`internal/ops/owner_unix.go`) when done, so running it as root never leaves files the service cannot write. List it in `usage()` in `cmd/form-md/main.go`.
2. A new `doctor` check returns `ok`, `warn` or `fail` with a note that says what to do. `fail` is for states where the instance is broken or data is at risk.
3. A new schema change is a new numbered file in `internal/store/migrations/`; never edit one that has shipped.

## Gotchas

- Without `FORMMD_DATA`, every command uses `./data` relative to where you run it, which is not the server's directory on a systemd install. Use the `fm` function above.

## Related

[[Backup-and-restore]] · [[Encryption]] · [[Configuration]] · [[Install]]
