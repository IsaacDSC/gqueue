package eventqueue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/hibiken/asynq"
)

type RequestPayload struct {
	EventName string            `json:"event_name"`
	Trigger   Trigger           `json:"trigger"`
	Data      map[string]any    `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func (p RequestPayload) mergeHeaders(headers map[string]string) map[string]string {
	if p.Headers == nil {
		p.Headers = make(map[string]string)
	}

	for key, value := range headers {
		p.Headers[key] = value
	}

	return p.Headers
}

type Fetcher interface {
	NotifyTrigger(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error
}

func GetRequestHandle(fetch Fetcher) asynqsvc.AsynqHandle {
	return asynqsvc.AsynqHandle{
		Event: "event-queue.request-to-external",
		Handler: func(ctx context.Context, task *asynq.Task) error {
			var payload RequestPayload
			if err := json.Unmarshal(task.Payload(), &payload); err != nil {
				return fmt.Errorf("unmarshal payload: %w", err)
			}

			headers := payload.mergeHeaders(payload.Trigger.Headers)
			if err := fetch.NotifyTrigger(ctx, payload.Data, headers, payload.Trigger); err != nil {
				return fmt.Errorf("fetch trigger: %w", err)
			}

			return nil
		},
	}
}
