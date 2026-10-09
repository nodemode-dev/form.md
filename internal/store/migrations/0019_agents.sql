CREATE TABLE agents (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	created_by TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	icon TEXT NOT NULL DEFAULT '',
	instructions TEXT NOT NULL DEFAULT '',
	model TEXT NOT NULL DEFAULT '',
	trigger_kind TEXT NOT NULL DEFAULT 'manual' CHECK (trigger_kind IN ('manual', 'schedule', 'row_added', 'row_changed', 'page_changed')),
	trigger_config TEXT NOT NULL DEFAULT '{}',
	enabled INTEGER NOT NULL DEFAULT 1,
	next_run_at INTEGER NOT NULL DEFAULT 0,
	last_run_at INTEGER NOT NULL DEFAULT 0,
	last_status TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX agents_workspace ON agents (workspace_id, name);
CREATE INDEX agents_due ON agents (enabled, trigger_kind, next_run_at);

CREATE TABLE agent_chats (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	agent_id TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX agent_chats_user ON agent_chats (user_id, workspace_id, updated_at);
CREATE INDEX agent_chats_agent ON agent_chats (agent_id, updated_at);

CREATE TABLE agent_messages (
	id TEXT PRIMARY KEY,
	chat_id TEXT NOT NULL REFERENCES agent_chats (id) ON DELETE CASCADE,
	seq INTEGER NOT NULL,
	role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
	text TEXT NOT NULL DEFAULT '',
	parts TEXT NOT NULL DEFAULT '[]',
	steps TEXT NOT NULL DEFAULT '[]',
	run_id TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL
) STRICT;

CREATE UNIQUE INDEX agent_messages_seq ON agent_messages (chat_id, seq);

CREATE TABLE agent_memory (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	text TEXT NOT NULL,
	created_at INTEGER NOT NULL
) STRICT;

CREATE INDEX agent_memory_owner ON agent_memory (user_id, workspace_id, created_at);

CREATE TABLE agent_prefs (
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	instructions_page TEXT NOT NULL DEFAULT '',
	memory INTEGER NOT NULL DEFAULT 1,
	PRIMARY KEY (workspace_id, user_id)
) STRICT;
