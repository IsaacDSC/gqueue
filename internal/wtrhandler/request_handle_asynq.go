package wtrhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
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

type ConsumerInsights interface {
	Consumed(ctx context.Context, input domain.ConsumerMetric) error
}

func GetRequestHandle(fetch Fetcher, insights ConsumerInsights) asyncadapter.Handle[RequestPayload] {

	insertInsights := func(ctx context.Context, payload RequestPayload, started time.Time, isSuccess bool) {
		l := ctxlogger.GetLogger(ctx)
		finished := time.Now()
		if err := insights.Consumed(ctx, domain.ConsumerMetric{
			TopicName:    payload.EventName,
			ConsumerName: payload.Trigger.ServiceName,
			TimeStarted:  started,
			TimeEnded:    finished,
			TimeDuration: time.Duration(finished.Sub(started).Milliseconds()),
			ACK:          isSuccess,
		}); err != nil {
			l.Warn("not save metric", "type", "consumer", "error", err.Error())
		}

	}

	return asyncadapter.Handle[RequestPayload]{
		Event: domain.EventQueueRequestToExternal,
		Handler: func(c asyncadapter.AsyncCtx[RequestPayload]) error {
			started := time.Now()
			ctx := c.Context()

			payload, err := c.Payload()
			if err != nil {
				return fmt.Errorf("get payload: %w", err)
			}

			headers := payload.mergeHeaders(payload.Trigger.Headers)
			if err := fetch.NotifyTrigger(ctx, payload.Data, headers, payload.Trigger); err != nil {
				insertInsights(ctx, payload, started, false)
				return fmt.Errorf("fetch trigger: %w", err)
			}

			insertInsights(ctx, payload, started, true)

			return nil
		},
	}
}
