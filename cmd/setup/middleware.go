package setup

import (
	"context"
	"net/http"
	"time"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/logs"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

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
