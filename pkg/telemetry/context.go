package telemetry

import (
	"context"

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
	if ctx == nil {
		return Meter("default")
	}

	if m, ok := ctx.Value(meterKey).(metric.Meter); ok && m != nil {
		return m
	}

	return Meter("default")
}
