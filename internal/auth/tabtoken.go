package auth

import (
	"context"

	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

const TabTokenPrefix = TokenPrefix + "tab_"

func (s *Service) TabToken(ctx context.Context, userID, name string) (string, string, error) {
	plain, id := secrets.Token(TabTokenPrefix, 32), ids.New()
	_, err := s.st.W.ExecContext(ctx, `
		INSERT INTO api_tokens (id, user_id, name, token_hash, prefix, scope, all_workspaces, created_at)
		VALUES (?, ?, ?, ?, ?, 'write', ?, ?)`,
		id, userID, name, secrets.Hash(plain), TabTokenPrefix, store.Bool(true), store.NowMS())
	if err != nil {
		return "", "", err
	}
	return plain, id, nil
}

func (s *Service) RevokeTabToken(ctx context.Context, id string) error {
	_, err := s.st.W.ExecContext(ctx, "DELETE FROM api_tokens WHERE id = ? AND prefix = ?", id, TabTokenPrefix)
	if err != nil {
		return err
	}
	s.CredentialRevoked(id)
	return nil
}

func (s *Service) SweepTabTokens(ctx context.Context) (int64, error) {
	res, err := s.st.W.ExecContext(ctx, "DELETE FROM api_tokens WHERE prefix = ?", TabTokenPrefix)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
