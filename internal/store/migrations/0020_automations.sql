CREATE TABLE automations (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	collection_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	created_by TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	trigger_kind TEXT NOT NULL CHECK (trigger_kind IN ('row_added', 'prop_edited', 'schedule')),
	trigger_config TEXT NOT NULL DEFAULT '{}',
	actions TEXT NOT NULL DEFAULT '[]',
	base_url TEXT NOT NULL DEFAULT '',
	next_run_at INTEGER NOT NULL DEFAULT 0,
	last_run_at INTEGER NOT NULL DEFAULT 0,
	last_status TEXT NOT NULL DEFAULT '',
	runs INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX automations_collection ON automations (collection_id, created_at);
CREATE INDEX automations_due ON automations (enabled, trigger_kind, next_run_at);

CREATE TABLE automation_runs (
	id TEXT PRIMARY KEY,
	automation_id TEXT NOT NULL REFERENCES automations (id) ON DELETE CASCADE,
	at INTEGER NOT NULL,
	row_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL CHECK (status IN ('ok', 'partial', 'failed', 'skipped')),
	detail TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX automation_runs_automation ON automation_runs (automation_id, at);

CREATE TABLE automation_keys (
	workspace_id TEXT PRIMARY KEY REFERENCES workspaces (id) ON DELETE CASCADE,
	secret TEXT NOT NULL,
	created_at INTEGER NOT NULL
) STRICT;
