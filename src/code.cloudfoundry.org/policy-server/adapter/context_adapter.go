package adapter

import (
	"context"
	"time"
)

type ContextAdapter struct{}

func (*ContextAdapter) WithTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, duration)
}
