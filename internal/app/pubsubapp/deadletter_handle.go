package pubsubapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/notifyopt"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

type DeadLetterStore interface {
	// SaveMsgNotDelivered(ctx context.Context, eventName string, msg any, ttl time.Duration) error

	GetAllSchedulers(ctx context.Context, eventName string) (domain.Event, error)
}

func NewDeadLatterQueue(store DeadLetterStore, fetcher Fetcher) asyncadapter.Handle[RequestPayload] {
	return asyncadapter.Handle[RequestPayload]{
		EventName: domain.EventQueueDeadLetter,
		Handler: func(c asyncadapter.AsyncCtx[RequestPayload]) error {
			ctx := c.Context()
			l := ctxlogger.GetLogger(ctx)

			p, err := c.Payload()
			if err != nil {
				return fmt.Errorf("failed to get payload: %w", err)
			}

			eventName := fmt.Sprintf("%s.dlq", p.EventName)

			event, err := store.GetAllSchedulers(ctx, eventName)
			if errors.Is(err, domain.EventNotFound) {
				// TODO: save in msg not delivered with TTL, and add endpoint get to msg not delivered
				l.Warn("Not found listers events when archived", "event_name", eventName)
				return nil
			}

			if err != nil {
				return fmt.Errorf("failed to get all schedulers: %w", err)
			}

			for _, consumer := range event.Consumers {
				err := fetcher.Notify(ctx, map[string]any{
					"event":    eventName,
					"data":     p.Data,
					"metadata": p.Headers,
					"event_at": p.PublishedAt,
				}, consumer.Headers, domain.Consumer{
					ServiceName: consumer.ServiceName,
					BaseUrl:     consumer.BaseUrl,
					Path:        consumer.Path,
					Headers:     consumer.Headers,
				}, notifyopt.HighThroughput)

				if err != nil {
					// TODO: save in msg not delivered with TTL, and add endpoint get to msg not delivered
					l.Warn("Failed to notify consumer", "error", err, "consumer", consumer)
				}
			}

			return nil
		},
	}
}
