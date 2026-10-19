UPDATE pages SET icon = 'i:bolt' || substr(icon, 10) WHERE icon = 'i:sparkle' OR icon LIKE 'i:sparkle:%';
UPDATE workspaces SET icon = 'i:bolt' || substr(icon, 10) WHERE icon = 'i:sparkle' OR icon LIKE 'i:sparkle:%';
UPDATE teamspaces SET icon = 'i:bolt' || substr(icon, 10) WHERE icon = 'i:sparkle' OR icon LIKE 'i:sparkle:%';
UPDATE agents SET icon = 'i:bolt' || substr(icon, 10) WHERE icon = 'i:sparkle' OR icon LIKE 'i:sparkle:%';

DELETE FROM comment_reactions WHERE icon = 'sparkle' AND EXISTS (
	SELECT 1 FROM comment_reactions b WHERE b.comment_id = comment_reactions.comment_id AND b.user_id = comment_reactions.user_id AND b.icon = 'bolt'
);
UPDATE comment_reactions SET icon = 'bolt' WHERE icon = 'sparkle';

DELETE FROM page_reactions WHERE icon = 'sparkle' AND EXISTS (
	SELECT 1 FROM page_reactions b WHERE b.page_id = page_reactions.page_id AND b.user_id = page_reactions.user_id AND b.icon = 'bolt'
);
UPDATE page_reactions SET icon = 'bolt' WHERE icon = 'sparkle';
