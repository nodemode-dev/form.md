CREATE TABLE calendar_feeds (
	token TEXT PRIMARY KEY,
	view_id TEXT NOT NULL REFERENCES collection_views (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL,
	UNIQUE (view_id, user_id)
) STRICT;
