CREATE TABLE page_views (
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	day INTEGER NOT NULL,
	viewer TEXT NOT NULL,
	views INTEGER NOT NULL DEFAULT 1,
	first_at INTEGER NOT NULL,
	last_at INTEGER NOT NULL,
	PRIMARY KEY (page_id, day, viewer)
) STRICT;

CREATE INDEX page_views_workspace ON page_views (workspace_id, day);
CREATE INDEX page_views_day ON page_views (day);

CREATE TABLE analytics_prefs (
	user_id TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
	record INTEGER NOT NULL DEFAULT 1,
	updated_at INTEGER NOT NULL
) STRICT;

ALTER TABLE workspaces ADD COLUMN view_history INTEGER NOT NULL DEFAULT 1;
