CREATE TABLE webhooks (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	kind TEXT NOT NULL DEFAULT 'webhook' CHECK (kind IN ('webhook', 'siem')),
	url TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	events TEXT NOT NULL DEFAULT '["*"]',
	secret TEXT NOT NULL,
	format TEXT NOT NULL DEFAULT 'json' CHECK (format IN ('json', 'splunk')),
	auth_header TEXT NOT NULL DEFAULT '',
	active INTEGER NOT NULL DEFAULT 1,
	failures INTEGER NOT NULL DEFAULT 0,
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX webhooks_workspace ON webhooks (workspace_id);

CREATE TABLE webhook_deliveries (
	id TEXT PRIMARY KEY,
	webhook_id TEXT NOT NULL REFERENCES webhooks (id) ON DELETE CASCADE,
	event TEXT NOT NULL,
	event_id TEXT NOT NULL,
	payload TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'delivered', 'failed')),
	attempts INTEGER NOT NULL DEFAULT 0,
	next_at INTEGER NOT NULL,
	last_status INTEGER NOT NULL DEFAULT 0,
	last_error TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	delivered_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX webhook_deliveries_due ON webhook_deliveries (status, next_at);
CREATE INDEX webhook_deliveries_hook ON webhook_deliveries (webhook_id, created_at);

CREATE TABLE event_cursors (
	name TEXT PRIMARY KEY,
	last_id INTEGER NOT NULL
) STRICT;
