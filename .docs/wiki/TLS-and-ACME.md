# TLS and ACME

form.md serves plain HTTP by default, on loopback only. For anything reachable from other machines, put HTTPS in front in one of three ways: give the server a certificate you manage, let it fetch and renew certificates itself over ACME (Let's Encrypt by default), or run it behind a reverse proxy that terminates TLS. Sign-in cookies are marked `Secure` only when the server sees the request as HTTPS, so pick one of the three before inviting people.

## Layout

| Path | What lives there |
|---|---|
| `internal/config/config.go` | `FORMMD_TLS_CERT`, `FORMMD_TLS_KEY`, `FORMMD_ACME_DOMAINS`, `FORMMD_ACME_EMAIL`, `FORMMD_ACME_DIRECTORY`, `FORMMD_HTTP_ADDR` and their checks |
| `internal/app/app.go` | `Run`: serves TLS when a certificate or ACME is configured, starts the challenge listener, shuts both down |
| `internal/app/acme.go` | The ACME manager (`golang.org/x/crypto/acme/autocert`), its cache, and the HTTP challenge listener |
| `internal/httpx/request.go` | How the server works out the client address, scheme and origin, with or without a trusted proxy |
| `DATADIR/acme/` | The certificate cache (account key and certificates) |

## Your own certificate

```sh
FORMMD_ADDR=0.0.0.0:443
FORMMD_TLS_CERT=/etc/form-md/tls/fullchain.pem
FORMMD_TLS_KEY=/etc/form-md/tls/privkey.pem
```

Both must be set together. The files are read at start, so restart the server after replacing them. `form-md doctor` loads the pair and warns when the certificate expires within 14 days.

## Built-in ACME

```sh
FORMMD_ADDR=0.0.0.0:443
FORMMD_ACME_DOMAINS=notes.example.com
FORMMD_ACME_EMAIL=ops@example.com
```

Put these in `/etc/form-md/env` and restart. The systemd unit grants `CAP_NET_BIND_SERVICE`, so the `form-md` user can bind ports 443 and 80.

What happens:

1. The server listens for HTTPS on `FORMMD_ADDR` and answers the CA's TLS-ALPN challenge there.
2. It opens a second, plain HTTP listener on `FORMMD_HTTP_ADDR` (default `:80`) that answers HTTP-01 challenges and redirects every other request to HTTPS. The log shows `acme http listener addr=... domains=...`.
3. On the first HTTPS request for a listed name it gets a certificate, then renews it on its own. Only the names in `FORMMD_ACME_DOMAINS` (comma-separated) are served; requests for other names get no certificate.
4. Certificates and the ACME account key are cached in `DATADIR/acme/`, so a restart does not ask the CA again. With [[Encryption]] on, every cache entry is sealed with the data key; an entry that cannot be unsealed counts as missing and is fetched again.

| Variable | Default | Purpose |
|---|---|---|
| `FORMMD_ACME_DOMAINS` | unset | Names to get certificates for; turns ACME on |
| `FORMMD_ACME_EMAIL` | unset | Contact address given to the CA |
| `FORMMD_ACME_DIRECTORY` | `https://acme-v02.api.letsencrypt.org/directory` | The CA's directory URL; set a staging or private CA's URL to test |
| `FORMMD_HTTP_ADDR` | `:80` | The challenge and redirect listener |

Check it:

```sh
journalctl -u form-md -n 20
curl -fsS https://notes.example.com/healthz
form-md doctor     # reports: ok tls ACME for notes.example.com via <directory>
```

## Behind a reverse proxy

Leave the server on loopback (`FORMMD_ADDR=127.0.0.1:8750`, the default) and let the proxy terminate TLS. The proxy must:

- forward the WebSocket at `/sync` (the realtime document socket) with its upgrade headers,
- forward the event stream at `/api/events` without buffering,
- send `X-Forwarded-Proto: https`,
- send `X-Forwarded-Host` with the public host name (without it the server uses the `Host` header the proxy passes on, which must then be the public name),
- send `X-Forwarded-For` (or `X-Real-Ip`) so rate limits and the audit log see real client addresses.

The server only reads those headers when the instance setting `net.trust_proxy` is on. Turn it on in one of two ways:

- **Before the first set-up:** add `FORMMD_TRUST_PROXY=true` to `/etc/form-md/env` and restart. The variable is the setting's default, so it applies from the first request, including the set-up screen.

  ```sh
  echo 'FORMMD_TRUST_PROXY=true' | sudo tee -a /etc/form-md/env
  sudo systemctl restart form-md
  ```

- **Later:** Instance settings, General, **Behind a reverse proxy**. A value saved there wins over the variable.

Also set **Public address** (Instance settings, General) to the `https://` address people use; it is used for links in mail, invitations and sign-in callbacks.

Why it matters: browser requests that change something (including sign-in and the first set-up) are refused as `cross-site request refused` unless their `Origin` matches the request origin as the server sees it, or the Public address. The `/sync` socket only accepts an `Origin` equal to the request origin as the server sees it. With `net.trust_proxy` off, the server sees `http://` behind a TLS proxy, refuses those requests and the socket, and does not mark session cookies `Secure`.

`form-md doctor` reports `warn tls off: serve behind a TLS proxy or set FORMMD_ACME_DOMAINS` in this setup; that is expected.

## The contract

1. `FORMMD_ACME_DOMAINS` and `FORMMD_TLS_CERT` are mutually exclusive, and the certificate and key are set together; `config.Load` enforces both. Keep new TLS options inside the same checks.
2. Anything written into `DATADIR/acme/` goes through the cache in `acme.go`, which seals it when encryption is on. `acme/` is in the backup and in `form-md encrypt`.
3. Code that needs the client address, scheme or origin calls `httpx.ClientIP`, `httpx.RequestScheme` or `httpx.RequestOrigin` with the `net.trust_proxy` setting, never the raw headers.
4. Only enable `net.trust_proxy` when the server can be reached through the proxy alone; otherwise anyone can forge the forwarded headers.

## Gotchas

- With ACME, `FORMMD_ADDR` must be on port 443: TLS-ALPN challenges and the redirect from port 80 both assume it.
- Port 80 must be free for the challenge listener; if it cannot bind, the server does not start.

## Related

[[Configuration]] · [[Encryption]] · [[Operations]]
