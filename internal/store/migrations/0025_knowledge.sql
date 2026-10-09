CREATE TABLE knowledge (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	kind TEXT NOT NULL CHECK (kind IN ('fact', 'decision', 'glossary', 'constraint', 'preference')),
	subject TEXT NOT NULL DEFAULT '',
	text TEXT NOT NULL,
	source_page TEXT NOT NULL DEFAULT '',
	source_block TEXT NOT NULL DEFAULT '',
	confidence REAL NOT NULL,
	confirmations INTEGER NOT NULL DEFAULT 0,
	contradictions INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	created_by TEXT NOT NULL DEFAULT '',
	agent TEXT NOT NULL DEFAULT '',
	confirmed_at INTEGER NOT NULL,
	superseded_by TEXT NOT NULL DEFAULT '',
	retired_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX knowledge_workspace ON knowledge (workspace_id, retired_at);

CREATE TABLE context_packs (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL,
	agent_key TEXT NOT NULL DEFAULT '',
	task TEXT NOT NULL DEFAULT '',
	task_type TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	sources TEXT NOT NULL DEFAULT '[]'
) STRICT;

CREATE INDEX context_packs_user ON context_packs (user_id, agent_key, created_at);
