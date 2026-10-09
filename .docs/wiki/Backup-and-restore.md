# Backup and restore

`form-md backup FILE` writes one gzip-compressed tar archive with a consistent snapshot of the database and every file the instance needs, while the server keeps running; the instance owner can download the same archive from the browser. `form-md restore FILE` replaces the data directory with a backup, with the server stopped, and keeps the old directory aside until you delete it. Both read `FORMMD_DATA` (and the key, when encryption is on) exactly as the server does.

## Layout

| Path | What lives there |
|---|---|
| `internal/ops/backup.go` | `Backup`, `Restore`, the archive manifest, what is carried |
| `internal/ops/flush_unix.go` | `RequestFlush` (asks a running server to save open documents) and the SIGUSR1 handler |
| `internal/ops/lock_unix.go`, `lock_other.go` | The data directory lock that keeps `restore` away from a running server |
| `internal/ops/module.go` | `GET /api/admin/backup`: the browser download |

## Making a backup

```sh
FORMMD_DATA=/var/lib/form-md form-md backup /var/backups/form-md/form-md-2026-10-09.tgz
```

What happens:

1. It opens the data directory with the same key check as the server (see [[Encryption]]).
2. On Unix, if a server holds the data directory, it sends that server SIGUSR1 and waits up to 20 seconds for it to save every open document, so text typed a moment ago is in the snapshot.
3. It snapshots the database with `VACUUM INTO` into a temporary file next to the output. The snapshot is consistent even while people are writing. With encryption on, the snapshot is written through the encrypted VFS, so it is encrypted with the same key.
4. It writes the archive to a temporary file in the same directory, syncs it, sets mode 0600, and renames it to `FILE`. A failed backup never leaves a half-written `FILE`.
5. Run as root, it hands every file in the data directory back to the directory's owner, so files SQLite created while it read the database stay writable by the service user.

It prints, for example:

```
backup written to /var/backups/form-md/form-md-2026-10-09.tgz: database and 214 files, 52428800 bytes before compression (plain; the running server saved open documents first)
```

`encrypted, restore needs the same key` replaces `plain` for an encrypted instance.

### What the archive holds

| Entry | Source |
|---|---|
| `form-md.db` | The `VACUUM INTO` snapshot (no `-wal` or `-shm` needed) |
| `secret.key` | Seals the secrets stored in instance settings; without it, saved SSO secrets and provider keys cannot be read |
| `encryption.json` | Only on an encrypted instance |
| `files/...` | Every upload, as stored (sealed when encryption is on); names starting with `.` are skipped |
| `acme/...` | The certificate cache, when ACME is used |
| `form-md-backup.json` | Manifest: format, version of the binary that wrote it, time, whether it is encrypted, whether the server flushed first, file count and byte count |

Not included: `form-md.db-wal`, `form-md.db-shm`, `form-md.lock`, `form-md.flushed` and `tmp/` (unfinished import uploads).

**The key is never in the archive.** Keep it somewhere else; see [[Encryption]].

## Downloading a backup from the browser

The instance owner sees a **Backup** section with **Download a backup** in Instance settings, About. It calls `GET /api/admin/backup`, which only answers the owner in a signed-in browser session; admins, API tokens and OAuth tokens are refused.

The server saves every open document, builds the same archive as `form-md backup` in a temporary folder inside the data directory (`.download-*`, removed afterwards), and sends it as `form-md-backup-YYYY-MM-DD-HHMMSS.tgz`. On an encrypted instance the archive is encrypted and only the same key restores it. The log records `backup downloaded user=... bytes=...`.

The archive is built on the server's disk before it is sent, so the data directory needs free space for one more copy while it downloads. Restore it with `form-md restore` as below.

## Restoring

```sh
sudo systemctl stop form-md
sudo env FORMMD_DATA=/var/lib/form-md /usr/local/bin/form-md restore /var/backups/form-md/form-md-2026-10-09.tgz
sudo systemctl start form-md
```

For an encrypted backup, add `FORMMD_KEY_FILE=/etc/form-md/form-md.key` to the `env` line.

What happens:

1. If the data directory exists, it takes the data directory lock. A running server holds it, so restore stops with `the data directory is in use by a running server: stop it first`.
2. It unpacks the archive into a staging directory `.form-md-restore-*` next to the data directory, in its parent. When the data directory is a mount point (such as a container volume) or the parent is not writable, it stages inside the data directory instead. Entries with unsafe paths stop it, and so does an archive without a manifest or without `form-md.db`.
3. If the manifest says the backup is encrypted, it needs the key (`this backup is encrypted: set FORMMD_KEY_FILE to its key`) and checks it against the backup's `encryption.json`.
4. It runs `PRAGMA integrity_check` on the staged database and stops if the result is not `ok`.
5. It swaps the restored data in and keeps the old data:
   - normally, it renames the current data directory to `DATADIR.before-restore-YYYYMMDD-HHMMSS` and renames the staging directory into place; if that second rename fails, the old directory is put back;
   - when it staged inside the data directory, it moves the current contents into `DATADIR/.before-restore-YYYYMMDD-HHMMSS-<random>` (leaving `form-md.lock` and earlier `.before-restore-` folders where they are), then moves the staged files up into the data directory.
6. Run as root, it gives every file in the restored data directory to the owner of the data directory as it was before (or of its parent, when there was none), so the service can start straight away.

It prints the version that made the backup, the number of files, and where the old directory went:

```
restored a backup from version vX.Y.Z (214 files) into /var/lib/form-md
the previous data directory was kept at /var/lib/form-md.before-restore-20261009-031502; delete it once the restore is checked
```

After starting the server, run `form-md doctor` (see [[Operations]]), sign in, open a few pages and files, then delete the `.before-restore-` directory. A backup from an older version is fine: the server runs any missing migrations on start.

On a systemd install run it with `sudo`: the staging directory goes in `/var/lib`, which the `form-md` user cannot write. An old copy kept inside the data directory (the second case in step 5) uses the same disk and is not carried by later backups.

## A nightly backup with cron

In root's crontab (`sudo crontab -e`), back up at 03:15 and keep 14 days. `%` must be escaped in crontab lines:

```cron
15 3 * * * env FORMMD_DATA=/var/lib/form-md /usr/local/bin/form-md backup /var/backups/form-md/form-md-$(date +\%Y\%m\%d).tgz && find /var/backups/form-md -name 'form-md-*.tgz' -mtime +14 -delete
```

Create the directory once with `sudo install -d -m 0700 /var/backups/form-md`. Add `FORMMD_KEY_FILE=/etc/form-md/form-md.key` after `env` on an encrypted instance. Copy the archives off the machine; a backup on the same disk does not survive the disk.

## In a container

Mount a directory for backups when you start the container, for example `-v /srv/form-md-backups:/backups` (the directory must be writable by user 65532). Then run the backup inside the running container, so it can signal the server:

```sh
docker exec form-md /usr/local/bin/form-md backup /backups/form-md-$(date +%Y%m%d).tgz
```

To restore, stop the container and run `restore` in a one-off container on the same volume. `/data` is a mount point, so restore works in place and keeps the old contents in `/data/.before-restore-...`:

```sh
docker stop form-md
docker run --rm -v form-md-data:/data -v /srv/form-md-backups:/backups ghcr.io/nodemode-ai/form.md:vX.Y.Z restore /backups/FILE.tgz
docker start form-md
```

## The contract

1. A file or directory in the data directory that the instance cannot work without is added to `carried` or `carriedDirs` in `internal/ops/backup.go`. Nothing else is backed up.
2. The database is copied only with `VACUUM INTO`, never by copying `form-md.db` while a server runs.
3. Restore unpacks and checks everything in a staging directory before it touches the live data, swaps by rename, and always keeps the previous data.
4. A server must keep answering SIGUSR1 by saving open documents and writing `form-md.flushed`; `backup` and `systemctl reload form-md` rely on it.

## Gotchas

- Outside Unix (Windows), `backup` cannot ask the server to save open documents first; the manifest then records `"live": false`. Text typed in the last moments may be missing from the snapshot.
- A backup of an encrypted instance restored without the key fails at step 3; a plain backup restored where a key is configured gives a plain directory that the server then refuses until you run `form-md encrypt`.
- `.before-restore-` copies, beside or inside the data directory, are full copies and are never deleted for you.

## Related

[[Operations]] · [[Encryption]] · [[Install]]
