CREATE TABLE scim_tokens (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL UNIQUE,
	label TEXT NOT NULL DEFAULT '',
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	last_used_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX scim_tokens_workspace ON scim_tokens (workspace_id);

CREATE TABLE scim_users (
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	external_id TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (workspace_id, user_id)
) STRICT;

CREATE INDEX scim_users_external ON scim_users (workspace_id, external_id);
