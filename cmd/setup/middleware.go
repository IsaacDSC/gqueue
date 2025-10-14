package setup

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/logs"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// CORSConfig define as configurações de CORS
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig retorna uma configuração padrão de CORS
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 horas
	}
}

// CORSMiddleware adiciona headers de CORS para permitir requisições cross-origin
func CORSMiddleware(next http.Handler) http.Handler {
	return CORSMiddlewareWithConfig(DefaultCORSConfig())(next)
}

// CORSMiddlewareWithConfig cria um middleware de CORS com configuração personalizada
func CORSMiddlewareWithConfig(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Verificar se a origem é permitida
			if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				for _, allowedOrigin := range config.AllowedOrigins {
					if allowedOrigin == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			// Definir métodos permitidos
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

			// Definir headers permitidos
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

			// Definir headers expostos
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

			// Definir credenciais
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Definir Max-Age para cache de preflight
			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}

			// Para requisições OPTIONS (preflight), retorna apenas os headers
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Continua para o próximo handler
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
