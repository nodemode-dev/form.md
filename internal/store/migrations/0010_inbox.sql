CREATE TABLE notifications (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	workspace_id TEXT NOT NULL DEFAULT '',
	kind TEXT NOT NULL,
	page_id TEXT NOT NULL DEFAULT '',
	thread_id TEXT NOT NULL DEFAULT '',
	actor_id TEXT NOT NULL DEFAULT '',
	actor_name TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	seen_at INTEGER NOT NULL DEFAULT 0,
	read_at INTEGER NOT NULL DEFAULT 0,
	archived_at INTEGER NOT NULL DEFAULT 0,
	pushed_at INTEGER NOT NULL DEFAULT 0,
	emailed_at INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX notifications_user ON notifications (user_id, archived_at, created_at);
CREATE INDEX notifications_pending ON notifications (read_at, seen_at, created_at);

CREATE TABLE push_subscriptions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	endpoint TEXT NOT NULL UNIQUE,
	p256dh TEXT NOT NULL,
	auth TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	failures INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX push_subscriptions_user ON push_subscriptions (user_id);

CREATE TABLE notify_prefs (
	user_id TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
	email TEXT NOT NULL DEFAULT 'all' CHECK (email IN ('all', 'mentions', 'off')),
	updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE reminders (
	page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
	user_id TEXT NOT NULL,
	ref TEXT NOT NULL,
	due_at INTEGER NOT NULL,
	fired_at INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (page_id, user_id, ref)
) STRICT;

CREATE INDEX reminders_due ON reminders (fired_at, due_at);
