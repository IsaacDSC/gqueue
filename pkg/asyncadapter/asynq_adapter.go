package asyncadapter

import (
	"context"
	"fmt"

	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/hibiken/asynq"
)

func (h Handle[T]) ToAsynqHandler() asynqsvc.AsynqHandle {
	return asynqsvc.AsynqHandle{
		TopicName: h.EventName,
		Handler: func(ctx context.Context, task *asynq.Task) error {
			if err := h.Handler(AsyncCtx[T]{
				ctx:         ctx,
				bytePayload: task.Payload(),
			}); err != nil {
				return fmt.Errorf("handle task: %w", err)
			}

			return nil
		},
	}
}
