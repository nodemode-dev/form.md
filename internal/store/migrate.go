package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	var out []migration
	seen := map[int]string{}
	for _, e := range entries {
		file := e.Name()
		num, name, ok := strings.Cut(strings.TrimSuffix(file, ".sql"), "_")
		if !ok {
			return nil, fmt.Errorf("migration %s: name must be NNNN_name.sql", file)
		}
		v, err := strconv.Atoi(num)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("migration %s: bad version", file)
		}
		if prev, dup := seen[v]; dup {
			return nil, fmt.Errorf("migrations %s and %s share version %d", prev, file, v)
		}
		seen[v] = file
		body, err := fs.ReadFile(migrationFS, "migrations/"+file)
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: v, name: name, sql: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.W.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at INTEGER NOT NULL
	) STRICT`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	applied := map[int]bool{}
	rows, err := s.W.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return err
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	if err := rows.Close(); err != nil {
		return err
	}

	all, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, m := range all {
		if applied[m.version] {
			continue
		}
		if err := s.apply(ctx, m); err != nil {
			return fmt.Errorf("migration %04d_%s: %w", m.version, m.name, err)
		}
	}
	return nil
}

func (s *Store) apply(ctx context.Context, m migration) error {
	tx, err := s.W.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
		m.version, m.name, NowMS()); err != nil {
		return err
	}
	return tx.Commit()
}
