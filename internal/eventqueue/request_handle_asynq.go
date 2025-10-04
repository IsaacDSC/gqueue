package eventqueue

import (
	"context"
	"fmt"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
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

func GetRequestHandle(fetch Fetcher) asyncadapter.Handle[RequestPayload] {
	return asyncadapter.Handle[RequestPayload]{
		Event: domain.EventQueueRequestToExternal,
		Handler: func(c asyncadapter.AsyncCtx[RequestPayload]) error {
			ctx := c.Context()
			payload, err := c.Payload()
			if err != nil {
				return fmt.Errorf("get payload: %w", err)
			}

			headers := payload.mergeHeaders(payload.Trigger.Headers)
			if err := fetch.NotifyTrigger(ctx, payload.Data, headers, payload.Trigger); err != nil {
				return fmt.Errorf("fetch trigger: %w", err)
			}

			return nil
		},
	}
}
