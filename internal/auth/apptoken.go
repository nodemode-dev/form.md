package auth

import (
	"context"
	"database/sql"
	"sync"

	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

const AppTokenName = "Claude Code in form.md"

var appTokens sync.Map

func (s *Service) AppToken(ctx context.Context, userID string) (string, error) {
	if v, ok := appTokens.Load(userID); ok {
		return v.(string), nil
	}
	plain := secrets.Token(TokenPrefix, 32)
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM api_tokens WHERE user_id = ? AND name = ?", userID, AppTokenName); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO api_tokens (id, user_id, name, token_hash, prefix, scope, all_workspaces, created_at)
			VALUES (?, ?, ?, ?, ?, 'write', ?, ?)`,
			ids.New(), userID, AppTokenName, secrets.Hash(plain), plain[:len(TokenPrefix)+6], store.Bool(true), store.NowMS())
		return err
	})
	if err != nil {
		return "", err
	}
	appTokens.Store(userID, plain)
	return plain, nil
}
