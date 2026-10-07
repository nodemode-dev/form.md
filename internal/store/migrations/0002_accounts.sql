CREATE TABLE settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL COLLATE NOCASE UNIQUE,
	name TEXT NOT NULL,
	color TEXT NOT NULL DEFAULT '',
	avatar TEXT NOT NULL DEFAULT '',
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
	email_verified INTEGER NOT NULL DEFAULT 0,
	disabled INTEGER NOT NULL DEFAULT 0,
	totp_secret TEXT NOT NULL DEFAULT '',
	totp_pending TEXT NOT NULL DEFAULT '',
	totp_enabled INTEGER NOT NULL DEFAULT 0,
	totp_last_step INTEGER NOT NULL DEFAULT 0,
	lang TEXT NOT NULL DEFAULT '',
	region TEXT NOT NULL DEFAULT '',
	time_zone TEXT NOT NULL DEFAULT '',
	hour_cycle TEXT NOT NULL DEFAULT '',
	week_start TEXT NOT NULL DEFAULT '',
	notes_mode INTEGER NOT NULL DEFAULT 0,
	system_font INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE UNIQUE INDEX users_single_owner ON users (role) WHERE role = 'owner';

CREATE TABLE sso_identities (
	provider TEXT NOT NULL,
	subject TEXT NOT NULL,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	email TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	PRIMARY KEY (provider, subject)
) STRICT;

CREATE TABLE recovery_codes (
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	code_hash TEXT NOT NULL,
	used_at INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (user_id, code_hash)
) STRICT;

CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	token_hash TEXT NOT NULL UNIQUE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	last_seen_at INTEGER NOT NULL,
	user_agent TEXT NOT NULL DEFAULT '',
	ip TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX sessions_user ON sessions (user_id);
CREATE INDEX sessions_expiry ON sessions (expires_at);

CREATE TABLE workspaces (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	icon TEXT NOT NULL DEFAULT '',
	image TEXT NOT NULL DEFAULT '',
	owner_id TEXT REFERENCES users (id) ON DELETE SET NULL,
	personal INTEGER NOT NULL DEFAULT 0,
	auto_join INTEGER NOT NULL DEFAULT 0,
	sidebar_mode TEXT NOT NULL DEFAULT 'split' CHECK (sidebar_mode IN ('split', 'tree')),
	agent_access TEXT NOT NULL DEFAULT 'open',
	rules TEXT NOT NULL DEFAULT '',
	rules_updated_at INTEGER NOT NULL DEFAULT 0,
	rules_proposal TEXT NOT NULL DEFAULT '',
	rules_proposal_by TEXT REFERENCES users (id) ON DELETE SET NULL,
	rules_proposal_at INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE workspace_members (
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	role TEXT NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
	joined_at INTEGER NOT NULL,
	PRIMARY KEY (workspace_id, user_id)
) STRICT;

CREATE INDEX workspace_members_user ON workspace_members (user_id);

CREATE TABLE api_tokens (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	token_hash TEXT NOT NULL UNIQUE,
	prefix TEXT NOT NULL,
	scope TEXT NOT NULL CHECK (scope IN ('read', 'write')),
	all_workspaces INTEGER NOT NULL DEFAULT 1,
	created_at INTEGER NOT NULL,
	last_used_at INTEGER NOT NULL DEFAULT 0,
	last_used_ip TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX api_tokens_user ON api_tokens (user_id);

CREATE TABLE api_token_workspaces (
	token_id TEXT NOT NULL REFERENCES api_tokens (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	PRIMARY KEY (token_id, workspace_id)
) STRICT;

CREATE TABLE invites (
	id TEXT PRIMARY KEY,
	token_hash TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL DEFAULT '' COLLATE NOCASE,
	role TEXT NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	created_by TEXT REFERENCES users (id) ON DELETE SET NULL,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	accepted_at INTEGER NOT NULL DEFAULT 0,
	accepted_by TEXT REFERENCES users (id) ON DELETE SET NULL
) STRICT;

CREATE TABLE break_glass (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	reason TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	revoked_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX break_glass_active ON break_glass (user_id, workspace_id, expires_at);

CREATE TABLE audit_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	at INTEGER NOT NULL,
	actor_kind TEXT NOT NULL CHECK (actor_kind IN ('human', 'agent', 'form', 'system')),
	actor_id TEXT NOT NULL DEFAULT '',
	actor_label TEXT NOT NULL DEFAULT '',
	agent_name TEXT NOT NULL DEFAULT '',
	agent_run TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL,
	workspace_id TEXT NOT NULL DEFAULT '',
	page_id TEXT NOT NULL DEFAULT '',
	detail TEXT NOT NULL DEFAULT '',
	changes TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX audit_log_workspace ON audit_log (workspace_id, id);
CREATE INDEX audit_log_page ON audit_log (page_id, id);
CREATE INDEX audit_log_run ON audit_log (agent_run) WHERE agent_run != '';
