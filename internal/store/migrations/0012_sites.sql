CREATE TABLE sites (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	page_id TEXT NOT NULL UNIQUE REFERENCES pages (id) ON DELETE CASCADE,
	slug TEXT NOT NULL UNIQUE,
	hostname TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	image TEXT NOT NULL DEFAULT '',
	theme TEXT NOT NULL DEFAULT 'auto' CHECK (theme IN ('auto', 'light', 'dark')),
	font TEXT NOT NULL DEFAULT 'sans' CHECK (font IN ('sans', 'serif', 'mono')),
	password_hash TEXT NOT NULL DEFAULT '',
	password_rev INTEGER NOT NULL DEFAULT 0,
	indexing INTEGER NOT NULL DEFAULT 0,
	subpages INTEGER NOT NULL DEFAULT 1,
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE UNIQUE INDEX sites_hostname ON sites (hostname) WHERE hostname != '';
CREATE INDEX sites_workspace ON sites (workspace_id);

ALTER TABLE workspaces ADD COLUMN publishing INTEGER NOT NULL DEFAULT 1;
