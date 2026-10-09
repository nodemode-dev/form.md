ALTER TABLE collections ADD COLUMN default_template TEXT NOT NULL DEFAULT '';

CREATE INDEX pages_templates ON pages (workspace_id, is_template) WHERE is_template = 1;
