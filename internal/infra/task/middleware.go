package task

import (
	"context"
	"github.com/hibiken/asynq"
	"log"
	"time"
)

func LogMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		start := time.Now()
		log.Printf("Start processing %q", t.Type())

		log.Printf("Task ID: %s, Payload: %s", t.Type(), string(t.Payload()))
		err := h.ProcessTask(ctx, t)
		if err != nil {
			return err
		}
		log.Printf("Finished processing %q: Elapsed Time = %v", t.Type(), time.Since(start))
		return nil
	})
}
