package asyncadapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/hibiken/asynq"
)

type AdapterType string

type AsyncCtx[T any] struct {
	ctx         context.Context
	payload     T
	bytePayload []byte
}

func (c AsyncCtx[T]) Payload() (T, error) {
	var empty T

	if err := json.Unmarshal(c.bytePayload, &c.payload); err != nil {
		return empty, fmt.Errorf("unmarshal payload: %w", err)
	}

	return c.payload, nil
}

func (c AsyncCtx[T]) Context() context.Context {
	return c.ctx
}

type Handle[T any] struct {
	Event   string
	Handler func(c AsyncCtx[T]) error
}

func (h Handle[T]) ToAsynqHandler() asynqsvc.AsynqHandle {
	return asynqsvc.AsynqHandle{
		Event: h.Event,
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
