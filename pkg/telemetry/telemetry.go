package telemetry

import (
	"context"
	"net/http"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	otelprom "go.opentelemetry.io/otel/exporters/prometheus"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config encapsulates metric provider initialization options.
type Config struct {
	Enabled bool
}

var (
	mu             sync.RWMutex
	meterProvider  *sdkmetric.MeterProvider
	metricsHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# metrics not initialized\n"))
	})
)

// New initializes the global MeterProvider and the HTTP metrics handler.
// It should be called once at application startup.
func New(cfg Config) (http.Handler, error) {
	mu.Lock()
	defer mu.Unlock()

	// If already initialized, just return the current handler.
	if meterProvider != nil {
		return metricsHandler, nil
	}

	var mp *sdkmetric.MeterProvider

	if !cfg.Enabled {
		// No-op provider: no metrics will be exported.
		mp = sdkmetric.NewMeterProvider()
		metricsHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# metrics disabled\n"))
		})
	} else {
		// Prometheus exporter: explicitly use the client_golang default registry
		// so promhttp.Handler() serves the same metrics.
		exp, err := otelprom.New(otelprom.WithRegisterer(promclient.DefaultRegisterer))
		if err != nil {
			return nil, err
		}

		// Use the exporter as a reader for the MeterProvider.
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(exp),
		)

		// Prometheus default handler to expose registered metrics.
		metricsHandler = promhttp.Handler()
	}

	otel.SetMeterProvider(mp)
	meterProvider = mp

	return metricsHandler, nil
}

// Shutdown stops the global MeterProvider and releases resources.
func Shutdown(ctx context.Context) error {
	mu.RLock()
	mp := meterProvider
	mu.RUnlock()

	if mp == nil {
		return nil
	}

	return mp.Shutdown(ctx)
}

// Meter returns a Meter from the global provider.
func Meter(name string) metric.Meter {
	mu.RLock()
	defer mu.RUnlock()

	if meterProvider == nil {
		return otel.Meter(name)
	}

	return meterProvider.Meter(name)
}

// Handler returns the current HTTP handler that exposes metrics.
func Handler() http.Handler {
	mu.RLock()
	defer mu.RUnlock()

	return metricsHandler
}
