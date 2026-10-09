CREATE TABLE oauth_clients (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL CHECK (kind IN ('registered', 'metadata')),
	name TEXT NOT NULL DEFAULT '',
	uri TEXT NOT NULL DEFAULT '',
	redirect_uris TEXT NOT NULL DEFAULT '[]',
	application_type TEXT NOT NULL DEFAULT 'native',
	fetched_at INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE oauth_requests (
	id TEXT PRIMARY KEY,
	client_id TEXT NOT NULL,
	redirect_uri TEXT NOT NULL,
	state TEXT NOT NULL DEFAULT '',
	challenge TEXT NOT NULL,
	scope TEXT NOT NULL,
	resource TEXT NOT NULL,
	expires_at INTEGER NOT NULL
) STRICT;

CREATE TABLE oauth_grants (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	client_id TEXT NOT NULL,
	client_name TEXT NOT NULL,
	scope TEXT NOT NULL CHECK (scope IN ('read', 'write')),
	all_workspaces INTEGER NOT NULL DEFAULT 0,
	resource TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	last_used_at INTEGER NOT NULL DEFAULT 0,
	revoked_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX oauth_grants_user ON oauth_grants (user_id);

CREATE TABLE oauth_grant_workspaces (
	grant_id TEXT NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	PRIMARY KEY (grant_id, workspace_id)
) STRICT;

CREATE TABLE oauth_codes (
	code_hash TEXT PRIMARY KEY,
	grant_id TEXT NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
	client_id TEXT NOT NULL,
	redirect_uri TEXT NOT NULL,
	challenge TEXT NOT NULL,
	resource TEXT NOT NULL,
	expires_at INTEGER NOT NULL,
	used_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE TABLE oauth_tokens (
	token_hash TEXT PRIMARY KEY,
	grant_id TEXT NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
	kind TEXT NOT NULL CHECK (kind IN ('access', 'refresh')),
	expires_at INTEGER NOT NULL,
	used_at INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL
) STRICT;

CREATE INDEX oauth_tokens_grant ON oauth_tokens (grant_id);
