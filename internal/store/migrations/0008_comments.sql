CREATE TABLE comment_threads (
	id TEXT PRIMARY KEY,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	anchor TEXT NOT NULL DEFAULT 'page' CHECK (anchor IN ('page', 'text', 'block')),
	anchor_start TEXT NOT NULL DEFAULT '',
	anchor_end TEXT NOT NULL DEFAULT '',
	quote TEXT NOT NULL DEFAULT '',
	resolved_at INTEGER NOT NULL DEFAULT 0,
	resolved_by TEXT NOT NULL DEFAULT '',
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX comment_threads_page ON comment_threads (page_id, created_at);

CREATE TABLE comments (
	id TEXT PRIMARY KEY,
	thread_id TEXT NOT NULL REFERENCES comment_threads (id) ON DELETE CASCADE,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	body TEXT NOT NULL,
	author_id TEXT NOT NULL DEFAULT '',
	author_name TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	edited_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX comments_thread ON comments (thread_id, created_at);
CREATE INDEX comments_page ON comments (page_id);

CREATE TABLE comment_reactions (
	comment_id TEXT NOT NULL REFERENCES comments (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL,
	icon TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (comment_id, user_id, icon)
) STRICT;
