# Host your own form.md

A step-by-step guide to running form.md on your own domain, as a private workspace for yourself, a small team or an organisation: pages, databases, boards and realtime editing, on a server you control, with the data in one encrypted folder.

This is how the form.md maintainers run their own instance. Greg and Mackin took a copy of form.md, called it formpunchlist, and put it on formai.monster as the punch list for the FormAI launch. This guide is that deployment with the names taken out, so it works for any domain.

Reference versions of the same material live in the wiki: `.docs/wiki/Self-hosting.md` and `.docs/wiki/Auth0.md`.

## Is this the right route for you

| You want | Use |
|---|---|
| form.md for one person on one computer | The desktop app. Nothing to host. Download it from the README |
| form.md for a few people who are never on the same network | This guide |
| To try it on a laptop and reach it from outside for an afternoon | The built-in tunnel, wiki page Tunnel |
| A server you already run behind your own reverse proxy | Wiki pages Install and TLS-and-ACME |

## What you end up with

- `https://notes.your-domain.com` (any name you choose), open to the people you invite and nobody else.
- One small virtual machine in Google Cloud running one program, `form-md`, under systemd. No database server, no cache, no object store, no reverse proxy.
- A certificate from Let's Encrypt that the server fetches and renews by itself.
- The database and every upload encrypted on disk with a key that you hold a copy of.
- A disk snapshot every day, kept for 14 days.

## How it fits together

form.md is one binary. It serves the web app, the API and the live editing socket from one port, and keeps everything in one data folder: a SQLite database, the uploaded files and a few keys.

That shape decides the hosting. SQLite in write-ahead-log mode needs every process that touches the database to be on the same machine, and it does not work over a network filesystem ([SQLite, Write-Ahead Logging](https://www.sqlite.org/wal.html)). So form.md wants one machine with a real local disk and exactly one server process: a virtual machine, not a platform that starts and stops copies of your container. That is why formai.monster is a Compute Engine VM and not Cloud Run.

Everything a reverse proxy would normally do, form.md does itself: it terminates TLS, gets its own certificate, redirects port 80 to HTTPS and sets the security headers.

## What you need

- **A domain or subdomain** whose DNS records you can edit. A subdomain such as `notes.your-domain.com` is easiest.
- **A Google Cloud project** with billing turned on, and the `gcloud` command installed and signed in on your computer (`gcloud auth login`). The VM, its disk, its snapshots and its static address are billed by Google; check current prices for an e2-small before you start.
- **Go 1.26 or newer and Node.js 22 or newer** on your computer. The deploy script builds form.md there and copies the result to the VM.
- **git**, and a GitHub account if you want your copy to live there.
- About half an hour, most of it waiting for DNS.

Not on Google Cloud? Skip to "On any other Linux host" near the end; the steps on the server are the same.

## Step 1 · Get your own copy

Make the repository yours, so you have a place to keep your settings and any changes.

On GitHub, fork `nodemode-dev/form.md`, then:

```sh
git clone git@github.com:YOUR-ACCOUNT/form.md.git
cd form.md
git remote add upstream https://github.com/nodemode-dev/form.md.git
```

A plain clone that you never publish works just as well. A private copy is fine too: form.md is licensed Apache-2.0; keep `LICENSE` and `NOTICES.md` with any copy you pass on.

Check that it builds on your machine before involving a server:

```sh
./form-md_oc_install      # installs Go modules and the web client's packages
./build_app --server-only # builds bin/form-md
./bin/form-md version
```

## Step 2 · Tell it your address

Open `deploy/gcp/instance.env`. It is the settings file a new server starts with:

```sh
FORMMD_ADDR=0.0.0.0:443
FORMMD_HTTP_ADDR=:80
FORMMD_ACME_DOMAINS=notes.example.com
FORMMD_TERMINAL=off
FORMMD_FAMILY=off
```

Replace `notes.example.com` with your host name. That one line turns on automatic HTTPS for that name.

- For a bare domain that should also answer on www, list both: `FORMMD_ACME_DOMAINS=your-domain.com,www.your-domain.com`.
- Optionally add `FORMMD_ACME_EMAIL=you@your-domain.com`, the contact address given to the certificate authority.
- Leave `FORMMD_TERMINAL=off`. The in-app terminal is a shell on the machine the server runs on. On your laptop that is a feature; on a shared server it is not.
- `FORMMD_FAMILY=off` stops the server looking for the other Form apps, which only ever run beside the desktop app.

Commit the change to your copy:

```sh
git commit -am "chore(ops): our host name"
```

The installer refuses to set up a new server while the file still says `example.com`, so this step cannot be forgotten.

## Step 3 · Create the server

```sh
GCP_PROJECT=your-project-id DOMAIN=notes.your-domain.com deploy/gcp/provision
```

This creates four things in your project and prints what it did:

| Resource | Why |
|---|---|
| A static external address | So the DNS record never has to change ([Google Cloud, Reserve a static external IP address](https://docs.cloud.google.com/vpc/docs/reserve-static-external-ip-address)) |
| A firewall rule for ports 80 and 443 | The only two ports the world can reach |
| A daily snapshot schedule, 14 days kept | A floor under your backups |
| A VM: e2-small, Debian 12, 20 GB disk, Shielded VM, OS Login | The machine itself |

It is safe to run again: each resource is looked up first and created only if missing. To use another region, size or name, set `GCP_REGION`, `GCP_ZONE`, `VM_MACHINE`, `VM_DISK_GB` or `VM_NAME` (and pass the same `VM_NAME`, region and zone to the ship step later).

The last thing it prints is the static address and the DNS records to create.

## Step 4 · Point your domain at it

At your registrar or DNS host, create the records the script printed:

| Type | Name | Value | TTL |
|---|---|---|---|
| A | `notes` (your host name) | the static address | 300 |
| A | `www` (only if you listed it in step 2) | the static address | 300 |
| CAA (optional) | your domain | `0 issue "letsencrypt.org"` | 3600 |

Three things trip people up here:

- **No AAAA record.** The VM has no IPv6 address. A leftover AAAA record sends some visitors, and the certificate authority, somewhere else.
- **Cloudflare: DNS only.** If your zone is on Cloudflare, switch the record's proxy off (the grey cloud). A proxied record answers with Cloudflare's addresses and passes web traffic through Cloudflare; a DNS-only record answers with your server's own address ([Cloudflare, Proxy status](https://developers.cloudflare.com/dns/proxy-status/)). form.md gets its own certificate, and for that the certificate authority has to reach your VM directly.
- **The CAA record is optional.** With no CAA record any public certificate authority may issue for your domain; adding one that names `letsencrypt.org` restricts issuing to Let's Encrypt ([Let's Encrypt, CAA](https://letsencrypt.org/docs/caa/)). If your domain already has CAA records for another authority, you must add this one or issuing will fail.

Wait until the name resolves, then go on:

```sh
dig +short notes.your-domain.com     # must print the static address
```

## Step 5 · Build and install

```sh
GCP_PROJECT=your-project-id deploy/gcp/ship
```

On your computer this builds the web client and a single static Linux binary stamped with the version in the `Makefile`. It then copies the binary, the settings file and the systemd unit to the VM and runs the installer there, which:

1. creates a system user `form-md` that cannot log in, and the data folder `/var/lib/form-md`, readable by that user only;
2. writes a new encryption key to `/etc/form-md/form-md.key`, readable by root only, and hands it to the service as a systemd credential, so the data is encrypted from the first write and the service user never reads the key file;
3. copies your settings to `/etc/form-md/env`, but only if there is none yet;
4. installs the hardened systemd unit and starts the service.

When it finishes it tells you where the key is. **Copy the key somewhere safe now**, for example your password manager:

```sh
gcloud compute ssh form-md --zone=us-central1-a --command='sudo cat /etc/form-md/form-md.key'
```

A snapshot or a backup cannot be read without that key. Lose both the VM and the key and the data is gone.

## Step 6 · Create the owner, straight away

Open `https://notes.your-domain.com`. The first request takes a few seconds while the certificate is fetched. You see the set-up screen: name, email, password. That account is the owner of the instance.

Do this as soon as the install finishes. Until an owner exists, the set-up screen answers whoever opens the address first.

Then, signed in:

1. **Instance settings, General**: set **Public address** to `https://notes.your-domain.com` and give the instance a name. Links in mail and invitations, and sign-in callbacks, use this address.
2. **Account settings**: turn on two-factor sign-in and store the recovery codes.

Let's Encrypt validates over port 80 for the HTTP challenge, and the server answers the TLS challenge on port 443, which is why both ports are open and why nothing else may be listening on them ([Let's Encrypt, Challenge Types](https://letsencrypt.org/docs/challenge-types/)).

## Step 7 · Bring in your team

Nobody can create an account on your instance unless you allow it. **Instance settings, Sign-in, Registration** has three positions:

| Setting | Who can create an account | Good for |
|---|---|---|
| By invitation only (the default) | Only people you invite | A handful of named people |
| Email domain allowed | Anyone with an address at the domains you list | A company |
| Open to anyone | Anyone who finds the address | Almost never |

Invite people from **Workspace settings**. Each person gets a single-use link. Ask everyone to turn on two-factor sign-in.

Inside the workspace, sharing has six access levels, and teamspaces can be open, closed or private, so "the whole team sees this, only two of us see that" is a matter of sharing, not of running two servers.

## Step 8 · Turn on mail

Without mail, invitations are links you copy and send yourself, and nobody is emailed about unread mentions. With it, both happen by themselves. Instance settings, Email takes an SMTP account, or connects a Gmail or Microsoft 365 account with OAuth. Wiki page: Mail.

## Step 9 · Know how you would get it back

You now have three layers:

- **The daily disk snapshots** made by Google Cloud, 14 kept. They restore the whole VM.
- **`form-md backup`**, which writes one consistent archive of the data folder while people keep working. Run it on a schedule and copy the archive off the VM. Wiki page: Backup-and-restore.
- **The encryption key**, which you copied in step 5. Keep it with your backups, never inside them.

Try a restore once, on a scratch machine, before you need one.

## Making it yours

This is the part where your copy starts to differ from everyone else's. What Greg and Mackin changed for formai.monster is a fair picture of the range:

| Change | Where it lives | Code |
|---|---|---|
| The name on the sign-in page and in mail | Instance settings, General | None |
| Themes, accent colour, fonts, page styles | Preferences and Workspace settings | None |
| Their own starting content: a launch board with a punch list, a board by owner and a dashboard | A template in `internal/templates/`, built once into the team workspace by a small package of their own | Go |
| A table kept current from the GitHub API every few minutes | A new package beside the others in `internal/`, with its settings in the admin screen | Go and TypeScript |
| An email allowlist: only two addresses may ever hold an account, be invited or be shared with | One 40-line file in `internal/auth/` and a check at each place an address enters the system | Go |
| The deploy scripts in this guide | `deploy/gcp/` | Shell, now part of form.md |

Most teams never need the code-level rows. Templates, sharing and the registration setting cover "our workspace, our people". When you do change code:

- Keep your changes in your copy and take new releases from the original as they come:

  ```sh
  git fetch upstream
  git merge upstream/main
  deploy/gcp/ship
  ```

- Keep them small and beside the existing code, not inside it. A new package and one line that registers it merges cleanly for years; edits scattered through the core conflict on every release.
- Run `make check` before shipping. It is what the project's CI runs.
- Agents can do a lot of this work with you: the repository carries `CLAUDE.md` and `AGENTS.md` that tell a coding agent how the code is laid out and what the rules are.

## Stronger sign-in with Auth0

Out of the box, sign-in is email and password (hashed with argon2id), optional two-factor per account, a rate limit on attempts and invitation-only registration. That is sound for a small team. Putting an identity provider in front adds central control: one place to add and remove people, your company's login rules and multi-factor policy.

**Where this stands.** formai.monster does not use Auth0 yet; it is planned. form.md already signs people in through any OpenID Connect provider, and Auth0 is one, so no code is needed. But this exact pairing has not been tried against a real Auth0 tenant, so treat your first sign-in as the test, and read "Open points" in the wiki page Auth0 first.

In Auth0:

1. Create an application of type **Regular Web Application**.
2. Add `https://notes.your-domain.com/api/auth/sso/oidc/callback` to **Allowed Callback URLs**, the list of addresses Auth0 may send people back to after they sign in ([Auth0, Application Settings](https://auth0.com/docs/get-started/applications/application-settings)). form.md shows the exact address to copy in Instance settings, Sign-in.
3. Make sure tokens are signed with **RS256**, the algorithm Auth0 recommends ([Auth0, Signing Algorithms](https://auth0.com/docs/get-started/applications/signing-algorithms)). form.md accepts RS256 and ES256 and refuses HS256.
4. Note the application's **Domain**, **Client ID** and **Client Secret**. The Client ID is public; the secret is not.
5. Close the front door: on the database connection, turn on **Disable Sign Ups** and add your people yourself ([Auth0 Support, Disable Signups at Connection Level](https://support.auth0.com/center/s/article/Disable-Signups-at-Connection-Level)), or allow only your company's login.

In form.md, Instance settings, Sign-in, **Other single sign-on (OpenID Connect)**:

| Field | Value |
|---|---|
| Button label | `Auth0`, or your company's name |
| Issuer address | `https://` and the Auth0 Domain, for example `https://your-tenant.us.auth0.com` |
| Client ID | From the Auth0 application |
| Client secret | From the Auth0 application. It is stored sealed |

form.md finds everything else by itself from the provider's discovery document at `/.well-known/openid-configuration` under that address ([Auth0, Configure Applications with OIDC Discovery](https://auth0.com/docs/get-started/applications/configure-applications-with-oidc-discovery)).

Three things to know before you rely on it:

- **form.md still decides who gets an account.** Auth0 proves who someone is. With registration by invitation, a stranger who signs in at Auth0 is turned away by form.md. With "Email domain allowed", colleagues get an account on their first sign-in.
- **Passwords are not switched off.** Auth0 adds a button to the sign-in page; the email and password form stays. Accounts created through Auth0 have no password, so for them only the button works.
- **Two-factor moves to Auth0.** form.md asks for its own two-factor code at password sign-in only. For people who come in through Auth0, turn on multi-factor authentication in Auth0.

## Day to day

| Task | How |
|---|---|
| Upgrade | Merge the new version into your copy, run `deploy/gcp/ship`. It replaces the binary and restarts; settings and key are untouched |
| Read the log | `sudo journalctl -u form-md -n 50` on the VM |
| Is it up | `curl -fsS https://notes.your-domain.com/healthz` |
| Health check | `form-md doctor` on the VM, wiki page Operations |
| Change a server setting | Edit `/etc/form-md/env` on the VM, then `sudo systemctl restart form-md` |
| Locked out as owner | `form-md admin reset-owner` on the VM prints a new password |
| Shell on the VM | `gcloud compute ssh form-md --zone=us-central1-a` |

## When something goes wrong

| What you see | Usual cause |
|---|---|
| The browser warns about the certificate, or the page never loads | DNS does not point at the VM yet, a Cloudflare record is proxied, or an AAAA record points elsewhere. Fix DNS, wait a minute, reload |
| `install: deploy/gcp/instance.env still names example.com` | Step 2 was skipped. Edit the file and ship again |
| You edited `deploy/gcp/instance.env` and nothing changed | The installer writes `/etc/form-md/env` only once. Edit the file on the VM and restart |
| `ship` says the start-up bundle is over budget | A size estimate that varies with the local toolchain. It is a warning; the deploy continues |
| The service did not start | The installer prints the last 40 log lines, which name the reason. A missing or wrong encryption key and a port that is already taken both show up there |
| Invitations never arrive | Mail is not set up (step 8). Copy the invitation link by hand meanwhile |

## On any other Linux host

Nothing above depends on Google Cloud except step 3. On any Linux machine with systemd, a public address and ports 80 and 443 reachable:

1. Build a Linux binary the way the first half of `deploy/gcp/ship` does (the client build, then `go build` with `GOOS=linux`), or take a released binary.
2. Copy the binary (named `form-md`), `deploy/gcp/install`, `deploy/gcp/instance.env` (edited) and `deploy/form-md.service` into one folder on the host.
3. Run `sudo bash install ./form-md` in that folder.
4. Continue from step 4 (DNS) and step 6.

If you cannot open those ports at all, the wiki page Tunnel covers a Cloudflare tunnel instead.

## What has and has not been run

- The deployment itself (VM, DNS, certificate, encrypted data, snapshots) is what serves formai.monster today, built from the formpunchlist copy.
- The scripts in `deploy/gcp/` here are those scripts with the names turned into variables. That edit has been syntax-checked, not run against a new project.
- The Auth0 section is worked out from form.md's sign-in code and Auth0's documentation, and has not been run against an Auth0 tenant.

## Sources

Everything about form.md itself comes from this repository: the scripts in `deploy/gcp/`, `internal/auth/oidc.go`, `internal/config/config.go` and the wiki in `.docs/wiki/`. Outside facts come from the vendors' own documentation:

- SQLite, [Write-Ahead Logging](https://www.sqlite.org/wal.html): all processes on one host, no network filesystems.
- Let's Encrypt, [Challenge Types](https://letsencrypt.org/docs/challenge-types/): the HTTP challenge uses port 80 only.
- Let's Encrypt, [Certificate Authority Authorization (CAA)](https://letsencrypt.org/docs/caa/): what a CAA record does and the name to put in it.
- Cloudflare, [Proxy status](https://developers.cloudflare.com/dns/proxy-status/): proxied and DNS-only records.
- Google Cloud, [Reserve a static external IP address](https://docs.cloud.google.com/vpc/docs/reserve-static-external-ip-address).
- Auth0, [Configure Applications with OIDC Discovery](https://auth0.com/docs/get-started/applications/configure-applications-with-oidc-discovery), [Application Settings](https://auth0.com/docs/get-started/applications/application-settings), [Signing Algorithms](https://auth0.com/docs/get-started/applications/signing-algorithms).
- Auth0 Support, [Disable Signups at Connection Level](https://support.auth0.com/center/s/article/Disable-Signups-at-Connection-Level).

Each outside fact above rests on the one vendor page linked beside it; none was cross-checked against a second independent source.
