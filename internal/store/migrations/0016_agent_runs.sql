CREATE TABLE agent_runs (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	credential TEXT NOT NULL DEFAULT '',
	kind TEXT NOT NULL CHECK (kind IN ('mcp', 'chat', 'custom', 'ai')),
	client TEXT NOT NULL DEFAULT '',
	agent_id TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'done', 'undone', 'failed')),
	ops INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	undone_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX agent_runs_user ON agent_runs (user_id, updated_at);
CREATE INDEX agent_runs_agent ON agent_runs (agent_id, updated_at);

CREATE TABLE agent_ops (
	run_id TEXT NOT NULL REFERENCES agent_runs (id) ON DELETE CASCADE,
	seq INTEGER NOT NULL,
	kind TEXT NOT NULL,
	page_id TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT '',
	undo TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	PRIMARY KEY (run_id, seq)
) STRICT;

CREATE TABLE mcp_idempotency (
	credential TEXT NOT NULL,
	key TEXT NOT NULL,
	tool TEXT NOT NULL,
	args_hash TEXT NOT NULL,
	result TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (credential, key)
) STRICT;

CREATE INDEX mcp_idempotency_age ON mcp_idempotency (created_at);
