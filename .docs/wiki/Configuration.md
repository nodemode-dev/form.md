# Configuration

form.md is configured in two places. Start-up settings (where to listen, where the data lives, TLS, the encryption key) come from environment variables prefixed `FORMMD_`; there is no config file. Everything else (instance name and public address, sign-up policy, single sign-on, mail, AI, housekeeping) is set by an admin in **Instance settings** inside the app and stored in the database. Three of those instance settings can take their default from an environment variable.

`form-md help` prints the same table as below, generated from the code, so it is always right for the binary you have.

## Layout

| Path | What lives there |
|---|---|
| `internal/config/config.go` | `Vars`: every `FORMMD_` start-up variable, its default and purpose; `Load()` validates them |
| `internal/settings/settings.go` | `Defs`: every instance setting, its type, default, range and optional `Env` variable |
| `/etc/form-md/env` | On a systemd install, the file the service reads its variables from (`EnvironmentFile=`) |

## Start-up variables

An empty value counts as unset and gives the default.

| Variable | Default | Purpose |
|---|---|---|
| `FORMMD_ADDR` | `127.0.0.1:8750` | Listen address, `host:port`. Set `0.0.0.0:8750` to accept connections from other machines. With ACME, set it to port 443 |
| `FORMMD_DATA` | `./data` | Data directory holding the database and uploads. Relative paths are made absolute from the working directory |
| `FORMMD_TLS_CERT` | unset | Path to a TLS certificate; set together with `FORMMD_TLS_KEY` |
| `FORMMD_TLS_KEY` | unset | Path to the TLS private key; set together with `FORMMD_TLS_CERT` |
| `FORMMD_KEY_FILE` | unset | Path to the at-rest encryption key (`form-md keygen` makes one). Keep it outside the data directory. When unset, a systemd credential named `form-md.key` is used if present. See [[Encryption]] |
| `FORMMD_ACME_DOMAINS` | unset | Comma-separated host names to get certificates for automatically. Needs ports 443 and 80 reachable. See [[TLS-and-ACME]] |
| `FORMMD_ACME_EMAIL` | unset | Contact address given to the certificate authority |
| `FORMMD_ACME_DIRECTORY` | `https://acme-v02.api.letsencrypt.org/directory` | ACME directory URL; point it at a staging or private CA to test |
| `FORMMD_HTTP_ADDR` | `:80` | With ACME on, the plain HTTP listener that answers challenges and redirects to HTTPS |
| `FORMMD_CLOUDFLARED` | unset | Path to `cloudflared` for the public tunnel; looked up on `PATH` when unset. See [[Tunnel]] |
| `FORMMD_UPDATE_URL` | unset | Where the daily update check reads the latest release; GitHub's releases API when unset. Point it at a mirror on a closed network. See [[Operations]] |
| `FORMMD_FETCH_PRIVATE` | unset | Set to `1` (or `true`) to let link previews, media imports, webhooks and other outbound requests reach private network addresses such as intranet sites. Off by default |

The server refuses to start (exit status 2, log line `config`) when:

- `FORMMD_ADDR` is not a valid `host:port`,
- only one of `FORMMD_TLS_CERT` and `FORMMD_TLS_KEY` is set,
- both `FORMMD_ACME_DOMAINS` and `FORMMD_TLS_CERT` are set.

## Instance settings with an environment default

| Variable | Setting | Default | Where it shows in the app |
|---|---|---|---|
| `FORMMD_TRUST_PROXY` | `net.trust_proxy` | `false` | Instance settings, General: "Behind a reverse proxy". Trust the `X-Forwarded-Proto`, `X-Forwarded-Host` and `X-Forwarded-For` headers; turn on only when a proxy you run sets them. See [[TLS-and-ACME]] |
| `FORMMD_TRASH_DAYS` | `trash.days` | `30` | Instance settings, General, Housekeeping: "Empty the trash after (days)". `0` keeps trashed pages until someone empties the trash by hand |
| `FORMMD_UPDATE_CHECK` | `update.check` | `true` | Instance settings, General, Housekeeping: "Check for new versions" (once a day, only the version number is fetched; see [[Operations]]) |

The order is: a value saved in Instance settings wins; otherwise the environment variable; otherwise the built-in default. So the variable seeds a fresh instance, and once an admin saves the field, the variable no longer has any effect. `form-md admin set KEY VALUE` saves any instance setting from the command line, the same as saving it in the app; it takes effect at the next start (see [[Operations]]).

## Instance settings without a variable

Everything else is set in the app by an admin and stored in the `settings` table. Secrets saved in the app (SSO client secrets, the SMTP password, saved AI provider keys and others) are sealed with `secret.key` in the data directory, which is why a backup must carry the whole data directory and not only `form-md.db`. Those that matter to operators:

- `instance.public_url` · Instance settings, General: "Public address". The address people use, for links in mail, invitations and sign-in callbacks.
- `https.auto`, `https.domain` · Instance settings, General: automatic HTTPS at the next start; see [[TLS-and-ACME]].
- `tunnel.mode`, `tunnel.token`, `tunnel.autostart` · Instance settings, General, Public tunnel; see [[Tunnel]].
- `mail.*` · Instance settings, Email; see [[Mail]].

## Setting variables

On a systemd install, edit `/etc/form-md/env` (one `NAME=value` per line) and restart:

```sh
sudoedit /etc/form-md/env
sudo systemctl restart form-md
```

The unit itself sets `FORMMD_DATA=/var/lib/form-md` and only allows writes under that path (`ReadWritePaths=`). To move the data directory, change both in a drop-in with `sudo systemctl edit form-md`.

By hand or in a container:

```sh
FORMMD_ADDR=0.0.0.0:8750 FORMMD_DATA=/srv/form-md ./form-md
docker run -e FORMMD_TRASH_DAYS=60 ...
```

The installer has five variables of its own (`FORMMD_VERSION`, `FORMMD_BIN_DIR`, `FORMMD_DATA_DIR`, `FORMMD_CONF_DIR`, `FORMMD_DOWNLOAD_URL`); the server does not read them. See [[Install]].

## The contract

1. A new start-up variable is added to `config.Vars` with a default and a purpose, read through `get()`, and validated in `Load()`. `form-md help` prints straight from that list; update this page in the same pull request.
2. A new instance setting is added to `settings.Defs` with a kind, a default and, for numbers, a range. Give it an `Env` only when operators need to seed it before anyone can sign in.
3. Anything secret is a `Secret` setting, sealed with `secret.key`, never a plain `Text` one.
4. Commands in `cmd/form-md/tool.go` load the same `config.Load()` as the server, so they see the same data directory and key; keep it that way.

## Gotchas

- `FORMMD_DATA` defaults to `./data` relative to the working directory. Starting the binary from another directory silently starts a fresh, empty instance there.
- Under systemd, do not set `FORMMD_DATA` in `/etc/form-md/env`: the unit sets it, and the service can only write under `/var/lib/form-md`.
- Changing `FORMMD_TRASH_DAYS` after an admin saved the trash setting does nothing.

## Related

[[Install]] · [[Encryption]] · [[TLS-and-ACME]] · [[Operations]]
