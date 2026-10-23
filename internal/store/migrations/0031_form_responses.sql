ALTER TABLE collection_rows ADD COLUMN response TEXT NOT NULL DEFAULT '';

CREATE TABLE automations_next (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	collection_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	created_by TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	trigger_kind TEXT NOT NULL CHECK (trigger_kind IN ('row_added', 'response_received', 'prop_edited', 'schedule')),
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

INSERT INTO automations_next (id, workspace_id, collection_id, created_by, name, enabled, trigger_kind, trigger_config, actions, base_url, next_run_at, last_run_at, last_status, runs, created_at, updated_at)
	SELECT a.id, a.workspace_id, a.collection_id, a.created_by, a.name, a.enabled,
		CASE WHEN a.trigger_kind = 'row_added' AND EXISTS (SELECT 1 FROM collection_views v WHERE v.collection_id = a.collection_id AND v.type = 'form') THEN 'response_received' ELSE a.trigger_kind END,
		CASE WHEN a.trigger_kind = 'row_added' AND EXISTS (SELECT 1 FROM collection_views v WHERE v.collection_id = a.collection_id AND v.type = 'form') THEN json_set(CASE WHEN json_valid(a.trigger_config) THEN a.trigger_config ELSE '{}' END, '$.manual', json('true')) ELSE a.trigger_config END,
		a.actions, a.base_url, a.next_run_at, a.last_run_at, a.last_status, a.runs, a.created_at, a.updated_at
	FROM automations a;

CREATE TEMP TABLE keep_automation_runs AS SELECT * FROM automation_runs;

DROP TABLE automations;
ALTER TABLE automations_next RENAME TO automations;
CREATE INDEX automations_collection ON automations (collection_id, created_at);
CREATE INDEX automations_due ON automations (enabled, trigger_kind, next_run_at);

INSERT INTO automation_runs SELECT * FROM temp.keep_automation_runs;
DROP TABLE temp.keep_automation_runs;
