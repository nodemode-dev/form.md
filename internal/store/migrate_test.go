package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func appliedVersions(t *testing.T, db *sql.DB) (versions []int, at map[int]int64) {
	t.Helper()
	rows, err := db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	at = map[int]int64{}
	for rows.Next() {
		var v int
		var a int64
		if err := rows.Scan(&v, &a); err != nil {
			t.Fatal(err)
		}
		versions = append(versions, v)
		at[v] = a
	}
	return versions, at
}

func TestMigrationsBuildAnEmptyDatabase(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	st, err := Open(ctx, dir, nil)
	if err != nil {
		t.Fatalf("open on an empty folder: %v", err)
	}
	versions, at := appliedVersions(t, st.R)
	if len(versions) == 0 || len(versions) != LatestMigration() {
		t.Fatalf("applied %d migrations, latest is %d", len(versions), LatestMigration())
	}
	for i, v := range versions {
		if v != i+1 {
			t.Fatalf("migration numbers are not contiguous: %v", versions)
		}
	}
	for _, table := range []string{"users", "workspaces", "pages", "doc_state", "doc_updates", "share_links", "api_tokens", "collection_rows", "automations"} {
		var n int
		if err := st.R.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&n); err != nil || n != 1 {
			t.Errorf("table %s missing after migrating from empty", table)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := Open(ctx, dir, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer again.Close()
	_, at2 := appliedVersions(t, again.R)
	if len(at2) != len(at) {
		t.Fatalf("a reopen changed the migration count: %d vs %d", len(at2), len(at))
	}
	for v, a := range at {
		if at2[v] != a {
			t.Fatalf("migration %d was applied again on reopen", v)
		}
	}
}

func seedBefore(t *testing.T, path string, upto int, statements []string) {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(0)&_pragma=synchronous(OFF)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL) STRICT`); err != nil {
		t.Fatal(err)
	}
	all, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range all {
		if m.version > upto {
			break
		}
		if _, err := db.Exec(m.sql); err != nil {
			t.Fatalf("seed migration %d: %v", m.version, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, 1)", m.version, m.name); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range statements {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
}

func TestMigrationsUpgradeThePreviousSchemaWithoutLosingData(t *testing.T) {
	if LatestMigration() < 31 {
		t.Skip("the upgrade under test is migration 31")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	seedBefore(t, path, 30, []string{
		`INSERT INTO users (id, email, name, password_hash, role, created_at, updated_at) VALUES ('u1', 'o@example.test', 'Owner', 'x', 'owner', 1, 1)`,
		`INSERT INTO workspaces (id, name, owner_id, created_at, updated_at) VALUES ('w1', 'Team', 'u1', 1, 1)`,
		`INSERT INTO pages (id, workspace_id, kind, sort_key, title, created_at, updated_at) VALUES ('form', 'w1', 'collection', 'a', 'Survey', 1, 1)`,
		`INSERT INTO pages (id, workspace_id, kind, sort_key, title, created_at, updated_at) VALUES ('plain', 'w1', 'collection', 'b', 'Tasks', 1, 1)`,
		`INSERT INTO collections (page_id, workspace_id, created_at, updated_at) VALUES ('form', 'w1', 1, 1)`,
		`INSERT INTO collections (page_id, workspace_id, created_at, updated_at) VALUES ('plain', 'w1', 1, 1)`,
		`INSERT INTO collection_views (id, collection_id, sort_key, name, type, created_at, updated_at) VALUES ('v1', 'form', 'a', 'Form', 'form', 1, 1)`,
		`INSERT INTO automations (id, workspace_id, collection_id, created_by, name, trigger_kind, actions, created_at, updated_at) VALUES ('a-form', 'w1', 'form', 'u1', 'on submit', 'row_added', '[{"kind":"notify"}]', 1, 1)`,
		`INSERT INTO automations (id, workspace_id, collection_id, created_by, name, trigger_kind, created_at, updated_at) VALUES ('a-plain', 'w1', 'plain', 'u1', 'on row', 'row_added', 1, 1)`,
		`INSERT INTO automations (id, workspace_id, collection_id, created_by, name, trigger_kind, created_at, updated_at) VALUES ('a-cron', 'w1', 'plain', 'u1', 'nightly', 'schedule', 1, 1)`,
		`INSERT INTO automation_runs (id, automation_id, at, status) VALUES ('r1', 'a-form', 5, 'ok')`,
		`INSERT INTO automation_runs (id, automation_id, at, status) VALUES ('r2', 'a-plain', 6, 'failed')`,
		`INSERT INTO collection_rows (page_id, collection_id, vals, number) VALUES ('plain', 'plain', '{"k":"kept"}', 1)`,
	})
	st, err := Open(context.Background(), dir, nil)
	if err != nil {
		t.Fatalf("upgrade from the previous schema: %v", err)
	}
	defer st.Close()
	versions, _ := appliedVersions(t, st.R)
	if len(versions) != LatestMigration() {
		t.Fatalf("applied %v, latest %d", versions, LatestMigration())
	}
	kinds := map[string]string{}
	rows, err := st.R.Query("SELECT id, trigger_kind FROM automations")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, kind string
		if err := rows.Scan(&id, &kind); err != nil {
			t.Fatal(err)
		}
		kinds[id] = kind
	}
	rows.Close()
	want := map[string]string{"a-form": "response_received", "a-plain": "row_added", "a-cron": "schedule"}
	for id, kind := range want {
		if kinds[id] != kind {
			t.Errorf("automation %s: trigger %q, want %q (all: %v)", id, kinds[id], kind, kinds)
		}
	}
	var actions, config string
	if err := st.R.QueryRow("SELECT actions, trigger_config FROM automations WHERE id = 'a-form'").Scan(&actions, &config); err != nil || actions != `[{"kind":"notify"}]` || config == "" {
		t.Errorf("the form automation lost its actions: %q %q %v", actions, config, err)
	}
	var runs int
	if err := st.R.QueryRow("SELECT COUNT(*) FROM automation_runs").Scan(&runs); err != nil || runs != 2 {
		t.Errorf("automation run history: %d rows after the upgrade, want 2 (%v)", runs, err)
	}
	var response, vals string
	if err := st.R.QueryRow("SELECT response, vals FROM collection_rows WHERE page_id = 'plain'").Scan(&response, &vals); err != nil || response != "" || vals != `{"k":"kept"}` {
		t.Errorf("existing rows changed in the upgrade: response %q vals %q %v", response, vals, err)
	}
	var title string
	if err := st.R.QueryRow("SELECT title FROM pages WHERE id = 'form'").Scan(&title); err != nil || title != "Survey" {
		t.Errorf("pages changed during the upgrade: %q %v", title, err)
	}
}

func TestAFailedMigrationLeavesNothingHalfApplied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	seedBefore(t, path, LatestMigration()-1, nil)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Store{W: db}
	err = s.apply(context.Background(), migration{version: 9999, name: "broken", sql: "CREATE TABLE half_done (id INTEGER); INSERT INTO no_such_table VALUES (1);"})
	if err == nil {
		t.Fatal("a broken migration reported success")
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name = 'half_done'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("the failed migration left a table behind (%d, %v)", n, err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 9999").Scan(&n); err != nil || n != 0 {
		t.Fatalf("the failed migration was recorded as applied (%d, %v)", n, err)
	}
}
