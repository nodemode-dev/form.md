# Tunnel

form.md can make itself reachable from the internet through a Cloudflare tunnel without opening a port or running a reverse proxy. The server runs `cloudflared` as a child process, watches it and restarts it when it exits. `cloudflared` reaches the server through a private unix socket in the data directory, and requests on that socket count as HTTPS from the visitor's address. You choose the tunnel in Instance settings and start or stop it there, or let it start with the server.

## Layout

| Path | What lives there |
|---|---|
| `internal/tunnel/tunnel.go` | The tunnel socket (`Listen`, `Socket`), finding `cloudflared`, starting and supervising it, the status, the request middleware, the admin routes |
| `internal/app/app.go` | `ExtraListener`: serves the same handler on the tunnel socket |
| `internal/settings/settings.go` | `tunnel.mode`, `tunnel.token`, `tunnel.autostart` |
| `internal/config/config.go` | `FORMMD_CLOUDFLARED` |
| `web/src/ui/admin.ts` | Instance settings, General: the **Public tunnel** and **Tunnel status** sections |

## Requirements

`cloudflared` must be installed on the server. The server uses `FORMMD_CLOUDFLARED` when it is set, otherwise it looks for `cloudflared` on `PATH`. When neither finds it, the status reads "cloudflared is not installed on this server." and starting answers `cloudflared is not installed on this server (or set FORMMD_CLOUDFLARED to its path)`.

## Settings

In **Instance settings, General, Public tunnel**:

| Field | Setting | Values |
|---|---|---|
| Tunnel | `tunnel.mode` | `off` (default); `quick`: a random `trycloudflare.com` address; `token`: a named tunnel from the Cloudflare dashboard |
| Tunnel token | `tunnel.token` | Only for `token`. A secret setting, sealed with `secret.key` |
| Start with the server | `tunnel.autostart` | Off by default. When on and the mode is not `off`, the tunnel starts when the server starts |

Save the settings, then use **Start the tunnel** or **Stop the tunnel** under **Tunnel status**. The status line shows `Running at https://....trycloudflare.com` for a quick tunnel once `cloudflared` prints the address, `Running.` for a named tunnel, or the last error. For a named tunnel it also shows the service to point the tunnel at: `unix:<socket path>`.

The same actions are API routes for signed-in admins (session only): `GET /api/admin/tunnel` (status), `POST /api/admin/tunnel/start` and `POST /api/admin/tunnel/stop`. Starting fails with `choose a tunnel mode first` when the mode is `off`, and with `save the tunnel token first` for a named tunnel without a token.

## The tunnel socket

At every start the server listens on a unix socket, `DATADIR/tunnel.sock`, with mode 0600 so only the server's own user can connect. When that path would be longer than 100 characters, the socket goes in the system temporary directory as `form-md-<hash>.sock` instead. If the socket cannot be created, the server logs `extra listener module=tunnel` and runs without it. The socket speaks plain HTTP even when the main listener serves TLS. Its path is `socket` in `GET /api/admin/tunnel`.

## How it runs

- **Quick tunnel:** `cloudflared tunnel --no-autoupdate --url unix:<socket>`. The public `trycloudflare.com` address is read from `cloudflared`'s output.
- **Named tunnel:** `cloudflared tunnel --no-autoupdate run`, with the token passed in the `TUNNEL_TOKEN` environment variable rather than on the command line. In the Cloudflare dashboard, point the tunnel's public hostname at the service `unix:<socket>`, using the path shown under Tunnel status. `cloudflared` runs as the server's user, so it can open the socket.
- If `cloudflared` exits, the server logs `tunnel exited; restarting`, waits, and starts it again. The wait starts at 1 second and doubles up to 1 minute; it goes back to 1 second after a run that lasted more than a minute. The status counts restarts.
- The tunnel stops when an admin stops it and when the server shuts down.

Log lines: `tunnel ready url=...` (quick tunnel address found), `tunnel started by=<user id>`, `tunnel stopped by=<user id>`, `tunnel exited; restarting`, `tunnel autostart err=...`.

## Requests through the tunnel

Every request that arrives on the tunnel socket is treated as an HTTPS request from the address in its `Cf-Connecting-Ip` header (`0.0.0.0` when the header is missing). So sign-in cookies are marked `Secure`, origin checks see `https://`, and rate limits, the audit log and fail2ban see the visitor's address. For a quick tunnel the request's host is also set to the `trycloudflare.com` address, so origin checks match it. Requests on the normal listener, including loopback TCP, are never treated this way, whatever headers they carry. None of this needs `net.trust_proxy`.

Set **Public address** (Instance settings, General) to the tunnel's public `https://` address, so links in mail, invitations and sign-in callbacks use it.

## The contract

1. Only requests on the tunnel socket are treated as tunneled. The socket stays mode 0600, and no TCP listener ever reads `Cf-Connecting-Ip`; anything wider lets a client claim any address.
2. The named tunnel token is a `Secret` setting and is passed to `cloudflared` through its environment, never as an argument (arguments are visible to every user on the machine).
3. Starting and stopping are admin-only, session-only routes.

## Gotchas

- The socket is open whenever the server runs, not only while the tunnel does; anything running as the server's user can connect to it and choose its `Cf-Connecting-Ip`.
- `tunnel.sock` lives in the data directory but is not part of backups; the server creates it again at start.
- A quick tunnel's address is whatever `cloudflared` prints; check the status line after every start and update **Public address** if it changed.
- The status and the restart counter live in memory and start over when the server restarts.

## Related

[[TLS-and-ACME]] · [[Configuration]] · [[Operations]]
