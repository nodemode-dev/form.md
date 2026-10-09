# Encryption

form.md can encrypt its data directory at rest with a 256-bit key that the operator holds outside the data directory. With a key, the database, its write-ahead log, the uploaded files, import uploads, `secret.key` and the ACME certificate cache are stored encrypted, so a copy of the disk or of the data directory alone reveals no page text and no file contents. Encryption is off until you give the server a key. There is no recovery without the key.

## Layout

| Path | What lives there |
|---|---|
| `internal/crypt/key.go` | Key parsing, `KeyPath` (where the key is looked for), `Load` (checks the key against `encryption.json`), subkey derivation |
| `internal/crypt/stream.go` | Chunked AES-256-GCM for files and small blobs (`Writer`, `Open`, `Seal`, `Unseal`, `WriteFile`, `ReadFile`) |
| `internal/cryptvfs/vfs.go` | The SQLite shim VFS that encrypts database, WAL and journal pages with XTS-AES-256 |
| `internal/store/store.go` | Registers the VFS as `formmd-crypt` and opens the database through it when a key is set |
| `internal/ops/encrypt.go` | `form-md encrypt` |
| `DATADIR/encryption.json` | Written when a directory becomes encrypted: format, algorithms and a key check value (never the key) |

## Where the key comes from

`KeyPath` looks in this order:

1. `FORMMD_KEY_FILE`, when set.
2. Otherwise a systemd credential named `form-md.key`, that is `$CREDENTIALS_DIRECTORY/form-md.key`, when that file exists.

The key file holds 32 bytes written as 64 hex characters (what `form-md keygen` prints) or as base64.

```sh
form-md keygen
```

prints a new key and touches nothing else. Store a copy of the key somewhere other than your backups: a backup of an encrypted instance is useless without it.

## Turning it on with systemd

Create the key, readable by root only:

```sh
sudo sh -c 'umask 077; /usr/local/bin/form-md keygen > /etc/form-md/form-md.key'
```

Give it to the service as a credential with a drop-in, so the service user never needs read access to the file:

```sh
sudo mkdir -p /etc/systemd/system/form-md.service.d
sudo tee /etc/systemd/system/form-md.service.d/key.conf >/dev/null <<'EOF'
[Service]
LoadCredential=form-md.key:/etc/form-md/form-md.key
EOF
sudo systemctl daemon-reload
```

The installer already started the service once, so the data directory holds a database. Encrypt it with the server stopped. Run as root, the command hands every file in the data directory back to the directory's owner (`form-md`), so nothing else is needed before starting the service:

```sh
sudo systemctl stop form-md
sudo env FORMMD_DATA=/var/lib/form-md FORMMD_KEY_FILE=/etc/form-md/form-md.key /usr/local/bin/form-md encrypt
sudo systemctl start form-md
journalctl -u form-md -n 20
```

Look for the line `at-rest encryption on database=xts-aes-256 files=aes-256-gcm`. Then check with `form-md doctor` (see [[Operations]]), which should report `ok encryption on, key ... matches`.

Instead of the credential you can put `FORMMD_KEY_FILE=/etc/form-md/form-md.key` in `/etc/form-md/env`; the file must then be readable by the `form-md` user.

## A new instance

A server started with a key on an empty data directory writes `encryption.json` and encrypts from the first write; no `encrypt` step is needed.

```sh
form-md keygen > ~/form-md.key && chmod 600 ~/form-md.key
FORMMD_KEY_FILE=~/form-md.key FORMMD_DATA=./data form-md
```

## What `form-md encrypt` does

It needs the key (`FORMMD_KEY_FILE` or the credential) and refuses when there is no database yet or when a running server holds the data directory lock. On a plain directory it:

1. checkpoints the WAL and writes an encrypted copy of the database with `VACUUM INTO` to `form-md.db.encrypting`,
2. runs `PRAGMA integrity_check` on the copy and stops if it fails,
3. seals every file under `files/`, `tmp/` and `acme/`, and `secret.key` (files that are already sealed are skipped), and deletes the unfinished temporary files an interrupted upload or import leaves behind (names starting with a dot, such as `.upload-*` and `.import-*`), since nothing reads them again,
4. makes the copy readable by its owner only (mode 0600), renames it over `form-md.db`, removes the old `-wal` and `-shm`, and writes `encryption.json`,
5. when run as root, hands every file in the data directory back to the directory's owner.

It prints `encrypted the database and N files (M already encrypted, K unfinished temporary files removed)`. If it stops partway, run it again.

On a directory that is already encrypted it checks the key against `encryption.json` (a wrong key stops it), takes the lock, seals any file under those directories, and `secret.key`, that is still plain, skipping the sealed ones, and removes leftover temporary files the same way. This is what to run when `form-md doctor` reports `N files are not encrypted`:

```sh
sudo systemctl stop form-md
sudo env FORMMD_DATA=/var/lib/form-md FORMMD_KEY_FILE=/etc/form-md/form-md.key /usr/local/bin/form-md encrypt
sudo systemctl start form-md
```

## What is encrypted

| Data | How |
|---|---|
| `form-md.db` | Every page, in 4096-byte units, with XTS-AES-256 tweaked by its file offset, through the `formmd-crypt` VFS |
| `form-md.db-wal` | The page inside every WAL frame, with its own XTS key |
| Rollback journal | Page records, with their own XTS key |
| SQLite temporary files | AES-CTR with a random key per open file; temporary storage is kept in memory |
| `files/` (uploads) | Chunked AES-256-GCM: a `FMDENC1` header and 16-byte salt, then 64 KiB chunks, each with its own nonce and tag; the last chunk is marked, so a cut-off file does not read as complete. The file key is derived from the salt, so every file has its own, and ranged reads decrypt only the chunks they touch |
| `tmp/` (import uploads) | The same chunked format |
| `secret.key` | The same chunked format |
| `acme/` (certificate cache) | The same chunked format |
| Backup archives | The database inside is the encrypted snapshot and the files are copied as stored, so a backup needs the same key |

All subkeys (database, WAL, journal, files, key check) are derived from the one master key with HKDF-SHA256.

## What is not encrypted

- File sizes, the number of files, timestamps, and the names of files in `files/` (an id plus the original extension).
- The WAL file header and each WAL frame header (page number, database size, salts and the frame checksums).
- The `-shm` file (SQLite's WAL index).
- `encryption.json`, `form-md.lock` and the backup manifest `form-md-backup.json` (version, time, counts, size and whether the backup is encrypted).
- Tampering with database pages is not detected as such. XTS keeps page contents secret but carries no authentication tag, so a page changed on disk decrypts to noise, which SQLite then reports as a damaged database (`PRAGMA integrity_check`, `form-md doctor`). Files, uploads and `secret.key` use AES-GCM and refuse to open when changed.
- Copies made before encryption was turned on: earlier backup archives, a directory kept aside by `restore`, or snapshots of the disk. `form-md encrypt` converts only the data directory; delete or re-create older plain backups yourself.
- Anything outside the data directory: data in memory, traffic on the network (use [[TLS-and-ACME]]), and pages kept in browsers for offline use (see [[Offline]]).

## Missing or wrong key

`serve`, `backup`, `doctor` and `admin reset-owner` check the key against the data directory first; `serve` exits with status 2 and logs `encryption`, the other commands print the message and exit with status 1 (`doctor` reports it as a failed check). `restore` checks the key only when the backup is encrypted, against the backup's own `encryption.json`, and says `this backup is encrypted: set FORMMD_KEY_FILE to its key` when none is given.

| Situation | Message |
|---|---|
| Directory is encrypted, no key given | `this data directory is encrypted: set FORMMD_KEY_FILE to the key file` |
| Key does not match `encryption.json` | `the key does not match this data directory` |
| Key file is not 32 bytes of hex or base64 | `the key file must hold 32 bytes as 64 hex characters or base64 (form-md keygen makes one)` |
| Key given, directory holds a plain database | `a key is set but this data directory is not encrypted: stop the server and run form-md encrypt first` |
| A sealed file is damaged or was sealed with another key | `crypt: the encrypted data is damaged or the key is wrong` |
| A plain file sits where an encrypted one is expected (copied in by hand, or left from before `encrypt`) | `crypt: this file is not encrypted although the instance is; run form-md doctor` (the file is not served; `form-md encrypt` seals it) |
| The encrypted database does not use 4096-byte pages | `an encrypted database must use 4096-byte pages (found N)` |

**Losing the key loses the data.** There is no recovery path, no command to change the key, and no command to turn encryption off.

## The contract

1. Anything written to the data directory that holds user content goes through the store's key: `Key.Writer`, `Key.WriteFile` or `Key.Seal` to write, `Key.Open`, `Key.ReadFile` or `Key.Unseal` to read. These pass data through unchanged when encryption is off, so the same code serves both modes. With a key set, readers refuse a plain file (`crypt.ErrNotSealed`) rather than serve something that bypassed encryption; the only place that reads plain files under a key is `form-md encrypt`, which seals them.
2. The database file is created with mode 0600 and the store chmods it to 0600 on every start.
3. The database is opened only through `store.Open` or `ops.openDB`, which select the `formmd-crypt` VFS when a key is set. A second connection opened some other way reads ciphertext.
4. A new directory of user content in the data directory is added to `carriedDirs` in `internal/ops/backup.go`; `form-md backup` copies those directories and `form-md encrypt` seals them.
5. The key never goes into the data directory, the database or a log line. `form-md doctor` warns when the key file sits inside the data directory.

## Gotchas

- Restoring a plain backup into an instance that has a key set gives a plain directory, and the server then refuses to start with the "not encrypted" message. Run `form-md encrypt` after such a restore.

## Related

[[Backup-and-restore]] · [[Operations]] · [[Configuration]]
