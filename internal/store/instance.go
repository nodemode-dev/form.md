package store

import (
	"context"

	"github.com/nodemode-ai/form.md/internal/ids"
)

func (s *Store) Bootstrap(ctx context.Context, version string) (instanceID string, err error) {
	now := NowMS()
	if _, err = s.W.ExecContext(ctx,
		`INSERT INTO instance (key, value, updated_at) VALUES ('instance_id', ?, ?)
		 ON CONFLICT(key) DO NOTHING`, ids.New(), now); err != nil {
		return "", err
	}
	if _, err = s.W.ExecContext(ctx,
		`INSERT INTO instance (key, value, updated_at) VALUES ('app_version', ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		version, now); err != nil {
		return "", err
	}
	err = s.R.QueryRowContext(ctx, "SELECT value FROM instance WHERE key = 'instance_id'").Scan(&instanceID)
	return instanceID, err
}
