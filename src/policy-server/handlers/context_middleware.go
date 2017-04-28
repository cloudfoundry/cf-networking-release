package handlers

import (
	"context"
	"net/http"
	"time"
)

//go:generate counterfeiter -o fakes/contextAdapter.go --fake-name ContextAdapter . contextAdapter
type contextAdapter interface {
	WithTimeout(context.Context, time.Duration) (context.Context, context.CancelFunc)
}

type ContextAdapter struct{}

func (*ContextAdapter) WithTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, duration)
}

type ContextWrapper struct {
	Duration       time.Duration
	ContextAdapter contextAdapter
}

func (a *ContextWrapper) Wrap(handle http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := a.ContextAdapter.WithTimeout(req.Context(), a.Duration)
		defer cancel()
		handle.ServeHTTP(w, req.WithContext(ctx))
	})
}
