CREATE TABLE suggestions (
	id TEXT PRIMARY KEY,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	anchor_start TEXT NOT NULL,
	anchor_end TEXT NOT NULL,
	original TEXT NOT NULL DEFAULT '',
	replacement TEXT NOT NULL DEFAULT '',
	author_id TEXT NOT NULL DEFAULT '',
	author_name TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'accepted', 'rejected')),
	decided_by TEXT NOT NULL DEFAULT '',
	decided_at INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX suggestions_page ON suggestions (page_id, status, created_at);
