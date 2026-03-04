package telemetry

import (
	"context"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"go.opentelemetry.io/otel/metric"
)

type meterKeyType struct{}

var meterKey = meterKeyType{}

// WithMeter injects the Meter into the context.
func WithMeter(ctx context.Context, m metric.Meter) context.Context {
	return context.WithValue(ctx, meterKey, m)
}

// MeterFromContext retrieves the Meter from the context, or a default Meter if absent.
func MeterFromContext(ctx context.Context) metric.Meter {
	l := ctxlogger.GetLogger(ctx)

	if ctx == nil {
		l.Warn("context is nil when getting meter from context")
		return Meter("default")
	}

	if m, ok := ctx.Value(meterKey).(metric.Meter); ok && m != nil {
		return m
	}

	l.Warn("meter not found in context")
	return Meter("default")
}
