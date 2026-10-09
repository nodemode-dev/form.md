//go:build !unix

package ops

import "context"

func OnFlushRequest(ctx context.Context, dataDir string, flush func(context.Context)) {}

func RequestFlush(dataDir string) bool { return false }
