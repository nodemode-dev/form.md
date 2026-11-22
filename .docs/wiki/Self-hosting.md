# Self-hosting

How to put your own form.md on your own domain for a small team: one virtual machine, the server's own HTTPS certificate, an encrypted data directory, accounts by invitation. The scripts are in `deploy/gcp`; the same steps work on any Linux host with systemd.

The long, step-by-step version of this page, written for people who have never run a server, is `.docs/SELF_HOSTING.md`.

## Layout

| Path | What lives there |
|---|---|
| `deploy/gcp/provision` | Creates the static address, the firewall rule, the snapshot schedule and the VM in Google Cloud, and prints the DNS records |
| `deploy/gcp/ship` | Builds the client and a linux binary from your checkout, copies them to the VM and runs `install` there |
| `deploy/gcp/install` | Runs on the VM as root: service user, directories, encryption key, env file, systemd unit |
| `deploy/gcp/instance.env` | The `/etc/form-md/env` a new VM starts with. Put your host name in it before the first `ship` |
| `deploy/form-md.service` | The hardened systemd unit, shared with the installer script |
| `install.sh` | The other route: installs a released binary instead of one built from your checkout ([[Install]]) |

## What you get

- One Compute Engine VM (e2-small, Debian 12, 20 GB balanced disk, Shielded VM, OS Login) with a static address and ports 80 and 443 open.
- The server terminating TLS itself with a Let's Encrypt certificate it fetches and renews ([[TLS-and-ACME]]).
- The SQLite database and uploads on the VM's disk, encrypted at rest with a key held as a systemd credential, so the service user never reads the key file ([[Encryption]]).
- 14 daily disk snapshots.
- Sign-up by invitation, which is the default.

A VM rather than a container platform that scales to zero or to many: the store is one SQLite file with a write-ahead log, which needs a local disk and exactly one server process.

## Before you start

- A domain or subdomain whose DNS records you can edit.
- A Google Cloud project with billing on, and `gcloud` signed in to it on your machine.
- Go 1.26 or newer and Node.js 22 or newer on your machine (`deploy/gcp/ship` builds there).
- Your own copy of the repository: a fork, or a clone you keep. You will commit one edited file to it.

## Steps

1. Put your host name in `deploy/gcp/instance.env`, in place of `notes.example.com`. For a bare domain that should also answer on www, list both, comma-separated: `FORMMD_ACME_DOMAINS=example.org,www.example.org`. Add `FORMMD_ACME_EMAIL=you@example.org` if you want the certificate authority to have a contact address. Commit the file to your copy.

2. Provision:

   ```sh
   GCP_PROJECT=your-project DOMAIN=notes.example.com deploy/gcp/provision
   ```

   It is safe to run twice: each resource is looked up before it is created. It prints the static address and the DNS records.

3. Create the DNS records it printed, at the registrar or DNS host of the domain:

   | Type | Name | Value | TTL |
   |---|---|---|---|
   | A | your host name | the static address | 300 |
   | A | `www` (only if listed in the env file) | the static address | 300 |
   | CAA (optional) | the domain | `0 issue "letsencrypt.org"` | 3600 |

   No AAAA record: the VM has no IPv6 address. If the zone is on Cloudflare, the A records must be DNS only, so the certificate authority reaches the VM itself on ports 80 and 443. Wait until `dig +short notes.example.com` prints the address.

4. Ship and install:

   ```sh
   GCP_PROJECT=your-project deploy/gcp/ship
   ```

   It builds the client and a static linux binary stamped with the `Makefile` version, copies them to the VM with the env file and the unit, and runs `deploy/gcp/install` there. On the first run it prints where the new encryption key is. Copy the key somewhere safe: a backup or a snapshot is unreadable without it.

5. Open `https://` and your host name at once and create the owner account. Until an owner exists, the set-up screen answers whoever opens the address first. The first request takes a few seconds while the certificate is fetched.

6. In Instance settings, General, set **Public address** to the `https://` address and give the instance a name. Links in mail, invitations and sign-in callbacks use the public address.

7. Turn on two-factor sign-in for your account (Account settings), then invite the team from Workspace settings. Registration stays **By invitation only** unless you change it in Instance settings, Sign-in; **Email domain allowed** lets anyone with an address at your company's domain create an account.

8. Set up mail so invitations and unread notifications are delivered ([[Mail]]).

To upgrade, pull the new version into your copy and run `deploy/gcp/ship` again: it replaces the binary and restarts the service, and leaves the env file and the key alone.

## Variables the scripts read

| Variable | Default | Used by | Purpose |
|---|---|---|---|
| `GCP_PROJECT` | none, required | provision, ship | The Google Cloud project id |
| `DOMAIN` | none, required | provision | The host name, for the DNS records it prints |
| `GCP_REGION` | `us-central1` | provision, ship | Region of the address and the snapshot schedule |
| `GCP_ZONE` | the region's `-a` zone | provision, ship | Zone of the VM |
| `VM_NAME` | `form-md` | provision, ship | Name of the VM; the address, firewall rule and schedule are named after it |
| `VM_MACHINE` | `e2-small` | provision | Machine type |
| `VM_DISK_GB` | `20` | provision | Boot disk size |
| `VM_TAG` | `form-md-web` | provision | Network tag the firewall rule applies to |
| `VM_ARCH` | `amd64` | ship | Processor the binary is built for |

These are read by the scripts only; the server never reads them.

## Without Google Cloud

Any Linux host with systemd, a public address and ports 80 and 443 reachable works the same way. Two routes:

- A release, with the installer script: download `install.sh` for the version you want and run it as [[Install]] describes, then put the lines of `deploy/gcp/instance.env` (with your host name) in `/etc/form-md/env`, add the encryption key as in [[Encryption]], and `sudo systemctl restart form-md`.
- Your own build: copy a Linux build of the binary (named `form-md`; the first half of `deploy/gcp/ship` shows the commands), `deploy/gcp/install`, `deploy/gcp/instance.env` and `deploy/form-md.service` into one folder on the host and run `sudo bash install ./form-md` there. It is the same script `ship` runs.

If you cannot open ports 80 and 443, use a tunnel instead ([[Tunnel]]).

## Stronger sign-in

Out of the box: passwords hashed with argon2id, two-factor sign-in per account, invitation-only registration and a sign-in rate limit, with a fail2ban filter in `deploy/fail2ban` ([[Operations]]). To put an identity provider in front, set up single sign-on in Instance settings, Sign-in: Google, Microsoft, any OpenID Connect provider, or SAML 2.0. [[Auth0]] walks through Auth0.

## Operations

- Logs: `sudo journalctl -u form-md -n 50` on the VM.
- Health: `curl -fsS https://notes.example.com/healthz`.
- Backups: the snapshot schedule keeps 14 daily disk snapshots; `form-md backup` makes a consistent copy while people work ([[Backup-and-restore]]). Keep the encryption key with the backups, never inside them.
- Checks: `form-md doctor` ([[Operations]]).
- Shell on the VM: `gcloud compute ssh form-md --zone=us-central1-a`.

## Gotchas

- Changing `/etc/form-md/env` needs `sudo systemctl restart form-md`. `install` writes that file only when it is absent, so editing `deploy/gcp/instance.env` later changes nothing on a VM that already has one: edit the file on the VM.
- The certificate is fetched on the first HTTPS request, and only once DNS points at the VM. If the address was opened before DNS was right, wait a minute and reload; `journalctl -u form-md` shows the certificate authority's answer.
- A proxied (orange cloud) Cloudflare record breaks certificate issuance, because the certificate authority then reaches Cloudflare and not the VM.
- The terminal and Claude Code panel are off on a hosted instance (`FORMMD_TERMINAL=off` in the env file): it would be a shell on the VM for the owner. Leave it off unless you want exactly that ([[Terminal-and-Claude-Code]]).
- `ship` warns, and carries on, when the start-up bundle measures over the development budget on your machine; `make build` and CI keep the hard limit.
- On another machine type or zone, pass the same `VM_NAME`, `GCP_REGION` and `GCP_ZONE` to `ship` that you passed to `provision`.
- The scripts in `deploy/gcp` have been syntax-checked only in this form; run `provision` on a scratch project first.

## Related

[[Install]] · [[Configuration]] · [[TLS-and-ACME]] · [[Encryption]] · [[Auth0]] · [[Backup-and-restore]] · [[Operations]]
