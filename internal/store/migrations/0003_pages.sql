CREATE TABLE pages (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	parent_id TEXT REFERENCES pages (id) ON DELETE CASCADE,
	kind TEXT NOT NULL DEFAULT 'doc' CHECK (kind IN ('doc', 'collection')),
	sort_key TEXT NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	icon TEXT NOT NULL DEFAULT '',
	cover TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	tags TEXT NOT NULL DEFAULT '[]',
	visibility TEXT NOT NULL DEFAULT 'workspace' CHECK (visibility IN ('workspace', 'private')),
	owner_id TEXT REFERENCES users (id) ON DELETE SET NULL,
	is_template INTEGER NOT NULL DEFAULT 0,
	snippet TEXT NOT NULL DEFAULT '',
	thumb TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	created_by TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL,
	updated_by TEXT NOT NULL DEFAULT '',
	trashed_at INTEGER NOT NULL DEFAULT 0,
	trash_batch TEXT NOT NULL DEFAULT '',
	trashed_by TEXT NOT NULL DEFAULT '',
	trashed_by_kind TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX pages_children ON pages (parent_id, sort_key);
CREATE INDEX pages_workspace ON pages (workspace_id, trashed_at);
CREATE INDEX pages_trashed ON pages (trashed_at) WHERE trashed_at > 0;

CREATE TABLE favorites (
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	sort_key TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (user_id, page_id)
) STRICT;

CREATE TABLE page_visits (
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	visited_at INTEGER NOT NULL,
	PRIMARY KEY (user_id, page_id)
) STRICT;

CREATE INDEX page_visits_recent ON page_visits (user_id, visited_at);

CREATE TABLE tag_colors (
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	tag TEXT NOT NULL COLLATE NOCASE,
	color TEXT NOT NULL,
	PRIMARY KEY (workspace_id, tag)
) STRICT;

CREATE TABLE files (
	id TEXT PRIMARY KEY,
	workspace_id TEXT REFERENCES workspaces (id) ON DELETE CASCADE,
	page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
	owner_id TEXT REFERENCES users (id) ON DELETE SET NULL,
	purpose TEXT NOT NULL CHECK (purpose IN ('attachment', 'cover', 'icon', 'avatar', 'workspace')),
	name TEXT NOT NULL,
	ext TEXT NOT NULL DEFAULT '',
	mime TEXT NOT NULL,
	size INTEGER NOT NULL,
	sha256 TEXT NOT NULL,
	created_at INTEGER NOT NULL
) STRICT;

CREATE INDEX files_workspace ON files (workspace_id, created_at);
CREATE INDEX files_page ON files (page_id);
