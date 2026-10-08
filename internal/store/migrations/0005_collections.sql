ALTER TABLE pages ADD COLUMN is_row INTEGER NOT NULL DEFAULT 0;

CREATE INDEX pages_rows ON pages (parent_id, is_row, sort_key);

CREATE TABLE collections (
	page_id TEXT PRIMARY KEY REFERENCES pages (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	schema TEXT NOT NULL DEFAULT '[]',
	inline INTEGER NOT NULL DEFAULT 0,
	next_number INTEGER NOT NULL DEFAULT 1,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX collections_workspace ON collections (workspace_id);

CREATE TABLE collection_views (
	id TEXT PRIMARY KEY,
	collection_id TEXT NOT NULL REFERENCES collections (page_id) ON DELETE CASCADE,
	sort_key TEXT NOT NULL,
	name TEXT NOT NULL,
	type TEXT NOT NULL CHECK (type IN ('table', 'board', 'list', 'gallery', 'calendar', 'timeline', 'form', 'chart', 'map', 'feed', 'dashboard')),
	config TEXT NOT NULL DEFAULT '{}',
	created_at INTEGER NOT NULL,
	created_by TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX collection_views_order ON collection_views (collection_id, sort_key);

CREATE TABLE collection_rows (
	page_id TEXT PRIMARY KEY REFERENCES pages (id) ON DELETE CASCADE,
	collection_id TEXT NOT NULL REFERENCES collections (page_id) ON DELETE CASCADE,
	vals TEXT NOT NULL DEFAULT '{}',
	number INTEGER NOT NULL DEFAULT 0,
	version INTEGER NOT NULL DEFAULT 1
) STRICT;

CREATE INDEX collection_rows_collection ON collection_rows (collection_id);

CREATE TABLE form_links (
	id TEXT PRIMARY KEY,
	view_id TEXT NOT NULL UNIQUE REFERENCES collection_views (id) ON DELETE CASCADE,
	token TEXT NOT NULL UNIQUE,
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	submissions INTEGER NOT NULL DEFAULT 0
) STRICT;
