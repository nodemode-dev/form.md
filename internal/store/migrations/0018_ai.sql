CREATE TABLE ai_providers (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL CHECK (kind IN ('anthropic', 'openai', 'gemini', 'compatible')),
	name TEXT NOT NULL,
	base_url TEXT NOT NULL DEFAULT '',
	api_key TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE ai_models (
	provider_id TEXT NOT NULL REFERENCES ai_providers (id) ON DELETE CASCADE,
	model TEXT NOT NULL,
	label TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 0,
	audio INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (provider_id, model)
) STRICT;

CREATE TABLE ai_usage (
	id INTEGER PRIMARY KEY,
	user_id TEXT NOT NULL DEFAULT '',
	workspace_id TEXT NOT NULL DEFAULT '',
	provider_id TEXT NOT NULL,
	model TEXT NOT NULL,
	feature TEXT NOT NULL,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	at INTEGER NOT NULL
) STRICT;

CREATE INDEX ai_usage_at ON ai_usage (at);
CREATE INDEX ai_usage_workspace ON ai_usage (workspace_id, at);

ALTER TABLE workspaces ADD COLUMN ai_enabled INTEGER NOT NULL DEFAULT 1;
