package ops

import (
	"context"

	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

func SetSetting(ctx context.Context, dataDir, keyFile, key, value string) (string, error) {
	k, err := OpenKey(dataDir, keyFile)
	if err != nil {
		return "", err
	}
	st, err := store.Open(ctx, dataDir, k)
	if err != nil {
		return "", err
	}
	defer matchOwner(dataDir)
	defer st.Close()
	box, err := secrets.Open(dataDir, k)
	if err != nil {
		return "", err
	}
	set, err := settings.New(ctx, st, box)
	if err != nil {
		return "", err
	}
	if err := set.Set(ctx, map[string]string{key: value}); err != nil {
		return "", err
	}
	return set.Get(key), nil
}
