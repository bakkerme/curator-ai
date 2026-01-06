package core

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

// WithLogger attaches a slog logger to the context.
// Callers should prefer passing a logger with useful correlation fields (e.g. flow_id, run_id).
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil || logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFromContext returns a slog logger attached to the context, or slog.Default() if absent.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}
