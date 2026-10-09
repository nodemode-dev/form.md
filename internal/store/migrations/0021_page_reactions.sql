CREATE TABLE page_reactions (
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	icon TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (page_id, user_id, icon)
) STRICT;
