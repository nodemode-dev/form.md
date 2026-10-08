CREATE TABLE doc_state (
	page_id TEXT PRIMARY KEY REFERENCES pages (id) ON DELETE CASCADE,
	generation INTEGER NOT NULL DEFAULT 1,
	snapshot BLOB NOT NULL,
	snapshot_seq INTEGER NOT NULL DEFAULT 0,
	next_seq INTEGER NOT NULL DEFAULT 1,
	markdown TEXT NOT NULL DEFAULT '',
	text_size INTEGER NOT NULL DEFAULT 0,
	materialised_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE doc_updates (
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	seq INTEGER NOT NULL,
	data BLOB NOT NULL,
	user_id TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	PRIMARY KEY (page_id, seq)
) STRICT;

CREATE TABLE doc_refs (
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	kind TEXT NOT NULL CHECK (kind IN ('page', 'file', 'user')),
	target TEXT NOT NULL,
	PRIMARY KEY (page_id, kind, target)
) STRICT;

CREATE INDEX doc_refs_target ON doc_refs (kind, target);
