CREATE TABLE page_versions (
	id TEXT PRIMARY KEY,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL,
	kind TEXT NOT NULL DEFAULT 'auto' CHECK (kind IN ('auto', 'manual', 'restore')),
	title TEXT NOT NULL DEFAULT '',
	editors TEXT NOT NULL DEFAULT '[]',
	words INTEGER NOT NULL DEFAULT 0,
	size INTEGER NOT NULL DEFAULT 0,
	state BLOB NOT NULL
) STRICT;

CREATE INDEX page_versions_page ON page_versions (page_id, created_at);
