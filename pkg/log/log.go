package log

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

var (
	defaultLogger *zap.Logger
)

func FromCtx(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
			return logger
		}
	}
	return defaultLogger
}

func ToCtx(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func init() {
	defaultLogger, _ = zap.NewProduction()
}
