package middleware

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/logs"
	"github.com/IsaacDSC/gqueue/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/attribute"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "PATCH", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

func CORSMiddleware(next http.Handler) http.Handler {
	return CORSMiddlewareWithConfig(DefaultCORSConfig())(next)
}

func CORSMiddlewareWithConfig(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				if slices.Contains(config.AllowedOrigins, origin) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}

			if len(config.AllowedMethods) > 0 {
				methods := ""
				for i, method := range config.AllowedMethods {
					if i > 0 {
						methods += ", "
					}
					methods += method
				}
				w.Header().Set("Access-Control-Allow-Methods", methods)
			}

			if len(config.AllowedHeaders) > 0 {
				headers := ""
				for i, header := range config.AllowedHeaders {
					if i > 0 {
						headers += ", "
					}
					headers += header
				}
				w.Header().Set("Access-Control-Allow-Headers", headers)
			}

			if len(config.ExposedHeaders) > 0 {
				exposedHeaders := ""
				for i, header := range config.ExposedHeaders {
					if i > 0 {
						exposedHeaders += ", "
					}
					exposedHeaders += header
				}
				w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			w.Header().Set("Content-Type", "application/json")

			next.ServeHTTP(w, r)
		})
	}
}

func AsynqLogger(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		start := time.Now()
		logger := logs.With(
			"task_type", t.Type(),
			"task_payload", string(t.Payload()),
			"request_id", uuid.New().String(),
		)

		logger.Info("Start processing")

		ctx = ctxlogger.WithLogger(ctx, logger)

		err := h.ProcessTask(ctx, t)
		if err != nil {
			logger.Error("Error processing task", "error", err)
			return err
		}

		logger.Info("Finished processing", "elapsed_time", time.Since(start))

		return nil
	})
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		logger := logs.With(
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"request_id", r.Header.Get("X-Request-ID"),
		)

		ctx := ctxlogger.WithLogger(r.Context(), logger)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware records HTTP metrics per service using OpenTelemetry.
func MetricsMiddleware(serviceName string, next http.Handler) http.Handler {
	// Create instruments once per middleware chain.
	meter := telemetry.Meter(serviceName)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/metrics" || path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		rec := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Ensure the Meter is available in the request context.
		ctxWithMeter := telemetry.WithMeter(r.Context(), meter)
		r = r.WithContext(ctxWithMeter)

		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()

		attrs := []attribute.KeyValue{
			attribute.String("http.method", r.Method),
			attribute.String("http.route", path),
			attribute.Int("http.status_code", rec.statusCode),
			attribute.String("service.name", serviceName),
		}

		ctx := r.Context()
		telemetry.HTTPServerRequests.Increment(ctx, attrs...)
		telemetry.HTTPServerRequestDuration.Record(ctx, duration, attrs...)
	})
}
