CREATE TABLE search_parts (
  id INTEGER PRIMARY KEY,
  page_id TEXT NOT NULL,
  part INTEGER NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('title', 'body', 'props')),
  heading TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX search_parts_page ON search_parts (page_id);

CREATE INDEX pages_hidden ON pages (workspace_id) WHERE visibility = 'private' OR is_template = 1;

CREATE VIRTUAL TABLE search_fts USING fts5(
  title,
  heading,
  body,
  ws,
  tokenize = 'porter unicode61 remove_diacritics 2',
  prefix = '2 3 4'
);

INSERT INTO search_fts (search_fts, rank) VALUES ('rank', 'bm25(10.0, 4.0, 1.0, 0.0)');

CREATE TABLE search_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
) STRICT;
