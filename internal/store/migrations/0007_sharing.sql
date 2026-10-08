CREATE TABLE teamspaces (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	icon TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	access TEXT NOT NULL DEFAULT 'open' CHECK (access IN ('open', 'closed', 'private')),
	sort_key TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	created_by TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX teamspaces_workspace ON teamspaces (workspace_id, sort_key);

CREATE TABLE teamspace_members (
	teamspace_id TEXT NOT NULL REFERENCES teamspaces (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
	joined_at INTEGER NOT NULL,
	PRIMARY KEY (teamspace_id, user_id)
) STRICT;

CREATE INDEX teamspace_members_user ON teamspace_members (user_id);

ALTER TABLE pages ADD COLUMN teamspace_id TEXT REFERENCES teamspaces (id) ON DELETE SET NULL;

CREATE TABLE groups (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	name TEXT NOT NULL COLLATE NOCASE,
	external_id TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT;

CREATE UNIQUE INDEX groups_name ON groups (workspace_id, name);

CREATE TABLE group_members (
	group_id TEXT NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	PRIMARY KEY (group_id, user_id)
) STRICT;

CREATE INDEX group_members_user ON group_members (user_id);

CREATE TABLE page_shares (
	id TEXT PRIMARY KEY,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	subject_kind TEXT NOT NULL CHECK (subject_kind IN ('user', 'group')),
	subject_id TEXT NOT NULL,
	level TEXT NOT NULL CHECK (level IN ('full', 'edit', 'content', 'create', 'comment', 'view')),
	expires_at INTEGER NOT NULL DEFAULT 0,
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	UNIQUE (page_id, subject_kind, subject_id)
) STRICT;

CREATE INDEX page_shares_subject ON page_shares (subject_kind, subject_id);
CREATE INDEX page_shares_workspace ON page_shares (workspace_id);

CREATE TABLE share_links (
	id TEXT PRIMARY KEY,
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL UNIQUE,
	level TEXT NOT NULL DEFAULT 'view' CHECK (level IN ('view', 'comment')),
	password_hash TEXT NOT NULL DEFAULT '',
	expires_at INTEGER NOT NULL DEFAULT 0,
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	revoked_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX share_links_page ON share_links (page_id);

ALTER TABLE invites ADD COLUMN page_id TEXT REFERENCES pages (id) ON DELETE CASCADE;
ALTER TABLE invites ADD COLUMN share_level TEXT NOT NULL DEFAULT '';
