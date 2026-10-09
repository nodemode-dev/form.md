package ops

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/crypt"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Check struct {
	Level string
	Name  string
	Note  string
}

func Doctor(ctx context.Context, cfg config.Config) []Check {
	var out []Check
	add := func(level, name, note string, args ...any) {
		out = append(out, Check{level, name, fmt.Sprintf(note, args...)})
	}
	st, err := os.Stat(cfg.DataDir)
	switch {
	case err != nil:
		add("fail", "data directory", "%s: %v", cfg.DataDir, err)
		return out
	case st.Mode().Perm()&0o077 != 0:
		add("warn", "data directory", "%s is readable by other users (%v); chmod 700 it", cfg.DataDir, st.Mode().Perm())
	default:
		add("ok", "data directory", "%s", cfg.DataDir)
	}

	if ln, err := net.Listen("tcp", cfg.Addr); err == nil {
		ln.Close()
		add("ok", "server", "not running (%s is free)", cfg.Addr)
	} else {
		add("ok", "server", "something is listening on %s, likely the server", cfg.Addr)
	}

	keyPath := crypt.KeyPath(cfg.KeyFile)
	key, err := crypt.Load(cfg.DataDir, keyPath, store.HasData(cfg.DataDir))
	switch {
	case err != nil:
		add("fail", "encryption", "%v", err)
		return out
	case key == nil:
		add("warn", "encryption", "off: the database and files are stored as plain files (set FORMMD_KEY_FILE and run form-md encrypt)")
	default:
		add("ok", "encryption", "on, key %s matches", keyPath)
		if strings.HasPrefix(filepath.Clean(keyPath), filepath.Clean(cfg.DataDir)+string(filepath.Separator)) {
			add("warn", "encryption key", "the key file is inside the data directory; a copy of the directory then carries its own key")
		}
	}

	dbPath := filepath.Join(cfg.DataDir, store.FileName)
	if !store.HasData(cfg.DataDir) {
		add("warn", "database", "none yet at %s (created on first start)", dbPath)
		return out
	}
	db, err := openDB(dbPath, key, nil)
	if err != nil {
		add("fail", "database", "%v", err)
		return out
	}
	defer matchOwner(cfg.DataDir)
	defer db.Close()
	var check string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&check); err != nil || check != "ok" {
		add("fail", "database", "quick check: %v %s", err, check)
		return out
	}
	var version int
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	switch latest := store.LatestMigration(); {
	case version > latest:
		add("fail", "schema", "the database is at migration %d, newer than this binary knows (%d): run the newer release", version, latest)
	case version < latest:
		add("ok", "schema", "migration %d; %d more will run on the next start", version, latest-version)
	default:
		add("ok", "schema", "migration %d (current)", version)
	}
	if info, err := os.Stat(dbPath + "-wal"); err == nil && info.Size() > 256<<20 {
		add("warn", "database", "the write-ahead log is %d MB; a long-running reader may be blocking checkpoints", info.Size()>>20)
	} else {
		add("ok", "database", "integrity ok")
	}

	out = append(out, checkFiles(ctx, db, cfg.DataDir, key)...)

	if _, err := os.Stat(filepath.Join(cfg.DataDir, "secret.key")); err != nil {
		add("warn", "secret key", "secret.key is missing; saved provider keys and SSO secrets cannot be read")
	} else if raw, err := key.ReadFile(filepath.Join(cfg.DataDir, "secret.key")); err != nil || len(strings.TrimSpace(string(raw))) != 64 {
		add("fail", "secret key", "secret.key could not be read: %v", err)
	} else {
		add("ok", "secret key", "readable")
	}

	var mode, host string
	_ = db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = 'mail.mode'").Scan(&mode)
	_ = db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = 'mail.smtp.host'").Scan(&host)
	switch {
	case mode == "" || mode == "off":
		add("warn", "mail", "mail is off: invitations and inbox notices are not sent by mail")
	case mode == "smtp" && host == "":
		add("warn", "mail", "mail mode is SMTP but no SMTP server is set")
	default:
		add("ok", "mail", "mail mode %s", mode)
	}

	var auto, domain string
	_ = db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = 'https.auto'").Scan(&auto)
	_ = db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = 'https.domain'").Scan(&domain)
	switch {
	case !cfg.TLS() && auto == "true" && domain != "":
		add("ok", "tls", "automatic HTTPS for %s from Instance settings (turn off with: form-md admin set https.auto false)", domain)
	case cfg.ACME():
		add("ok", "tls", "ACME for %s via %s", strings.Join(cfg.ACMEDomains, ", "), cfg.ACMEDirectory)
	case cfg.TLSCert != "":
		out = append(out, checkCert(cfg.TLSCert, cfg.TLSKey))
	default:
		add("warn", "tls", "off: serve behind a TLS proxy or set FORMMD_ACME_DOMAINS")
	}
	return out
}

func checkFiles(ctx context.Context, db *sql.DB, dataDir string, key *crypt.Key) []Check {
	rows, err := db.QueryContext(ctx, "SELECT id || ext FROM files")
	if err != nil {
		return []Check{{"fail", "files", err.Error()}}
	}
	known := map[string]bool{}
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			known[n] = true
		}
	}
	rows.Close()
	dir := filepath.Join(dataDir, "files")
	entries, _ := os.ReadDir(dir)
	onDisk := map[string]bool{}
	plain := 0
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		onDisk[e.Name()] = true
		if key != nil {
			if f, err := os.Open(filepath.Join(dir, e.Name())); err == nil {
				if !crypt.IsSealed(f) {
					plain++
				}
				f.Close()
			}
		}
	}
	missing, orphan := 0, 0
	for n := range known {
		if !onDisk[n] {
			missing++
		}
	}
	for n := range onDisk {
		if !known[n] {
			orphan++
		}
	}
	var out []Check
	if missing > 0 {
		out = append(out, Check{"fail", "files", fmt.Sprintf("%d of %d stored files are missing from %s", missing, len(known), dir)})
	} else {
		out = append(out, Check{"ok", "files", fmt.Sprintf("%d stored files present", len(known))})
	}
	if orphan > 0 {
		out = append(out, Check{"warn", "files", fmt.Sprintf("%d files on disk have no record (removed by the hourly sweep after a day)", orphan)})
	}
	if plain > 0 {
		out = append(out, Check{"fail", "files", fmt.Sprintf("%d files are not encrypted; stop the server and run form-md encrypt", plain)})
	}
	return out
}

func checkCert(certFile, keyFile string) Check {
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return Check{"fail", "tls", err.Error()}
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return Check{"fail", "tls", err.Error()}
	}
	left := time.Until(leaf.NotAfter)
	if left < 14*24*time.Hour {
		return Check{"warn", "tls", fmt.Sprintf("certificate for %v expires %s", leaf.DNSNames, leaf.NotAfter.Format(time.DateOnly))}
	}
	return Check{"ok", "tls", fmt.Sprintf("certificate for %v valid until %s", leaf.DNSNames, leaf.NotAfter.Format(time.DateOnly))}
}

func Print(w io.Writer, checks []Check) bool {
	failed := false
	for _, c := range checks {
		fmt.Fprintf(w, "%-5s %-15s %s\n", c.Level, c.Name, c.Note)
		if c.Level == "fail" {
			failed = true
		}
	}
	return !failed
}
