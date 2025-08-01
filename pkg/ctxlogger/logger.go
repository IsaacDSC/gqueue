package ctxlogger

import (
	"context"

	"github.com/IsaacDSC/webhook/pkg/logs"
)

type LoggerKey struct{}

var loggerKey = LoggerKey{}

func WithLogger(ctx context.Context, logger *logs.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func GetLogger(ctx context.Context) *logs.Logger {
	if logger, ok := ctx.Value(loggerKey).(*logs.Logger); ok {
		return logger
	}

	return logs.Default()
}
