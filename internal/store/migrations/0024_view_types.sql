CREATE TABLE collection_views_next (
	id TEXT PRIMARY KEY,
	collection_id TEXT NOT NULL REFERENCES collections (page_id) ON DELETE CASCADE,
	sort_key TEXT NOT NULL,
	name TEXT NOT NULL,
	type TEXT NOT NULL CHECK (type IN ('table', 'board', 'list', 'gallery', 'calendar', 'timeline', 'form', 'chart', 'map', 'feed', 'dashboard', 'heatmap', 'graph')),
	config TEXT NOT NULL DEFAULT '{}',
	created_at INTEGER NOT NULL,
	created_by TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL
) STRICT;

INSERT INTO collection_views_next (id, collection_id, sort_key, name, type, config, created_at, created_by, updated_at)
	SELECT id, collection_id, sort_key, name, type, config, created_at, created_by, updated_at FROM collection_views;

CREATE TEMP TABLE keep_form_links AS SELECT * FROM form_links;
CREATE TEMP TABLE keep_calendar_feeds AS SELECT * FROM calendar_feeds;

DROP TABLE collection_views;
ALTER TABLE collection_views_next RENAME TO collection_views;
CREATE INDEX collection_views_order ON collection_views (collection_id, sort_key);

INSERT INTO form_links SELECT * FROM temp.keep_form_links;
INSERT INTO calendar_feeds SELECT * FROM temp.keep_calendar_feeds;
DROP TABLE temp.keep_form_links;
DROP TABLE temp.keep_calendar_feeds;
