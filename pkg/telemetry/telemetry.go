package telemetry

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

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

// state holds the current meter provider and metrics handler.
// It is published via atomic.Value after initialization so all reads are lock-free.
// Alternative design: dependency injection — New could return a *Telemetry struct
// (with Handler, Meter, Shutdown methods) and the application would pass it explicitly
// instead of using package-level globals; that would avoid any sync primitive.
type state struct {
	provider *sdkmetric.MeterProvider
	handler  http.Handler
	initErr  error // non-nil if New failed during init
}

var (
	once     sync.Once
	stateVal atomic.Value
)

func init() {
	stateVal.Store(&state{
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# metrics not initialized\n"))
		}),
	})
}

// New initializes the global MeterProvider and the HTTP metrics handler.
// It should be called once at application startup. Initialization runs at most once;
// subsequent calls return the same handler and any initial error.
func New(cfg Config) (http.Handler, error) {
	once.Do(func() {
		var s state
		if !cfg.Enabled {
			s.provider = sdkmetric.NewMeterProvider()
			s.handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("# metrics disabled\n"))
			})
		} else {
			exp, err := otelprom.New(otelprom.WithRegisterer(promclient.DefaultRegisterer))
			if err != nil {
				s.initErr = err
				s.handler = stateVal.Load().(*state).handler // keep placeholder
				stateVal.Store(&s)
				return
			}
			s.provider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(exp))
			s.handler = promhttp.Handler()
		}
		otel.SetMeterProvider(s.provider)
		stateVal.Store(&s)
	})

	cur := stateVal.Load().(*state)
	if cur.initErr != nil {
		return nil, cur.initErr
	}
	return cur.handler, nil
}

// Shutdown stops the global MeterProvider and releases resources.
func Shutdown(ctx context.Context) error {
	cur := stateVal.Load().(*state)
	if cur.provider == nil {
		return nil
	}
	return cur.provider.Shutdown(ctx)
}

// Meter returns a Meter from the global provider.
func Meter(name string) metric.Meter {
	cur := stateVal.Load().(*state)
	if cur.provider == nil {
		return otel.Meter(name)
	}
	return cur.provider.Meter(name)
}

// Handler returns the current HTTP handler that exposes metrics.
func Handler() http.Handler {
	return stateVal.Load().(*state).handler
}
