package pubsubapp

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/notifyopt"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

type DeadLetterStore interface {
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
}

func NewDeadLatterQueue(store DeadLetterStore, fetcher Fetcher) asyncadapter.Handle[pubsub.Message] {
	return asyncadapter.Handle[pubsub.Message]{
		EventName: domain.EventQueueDeadLetter,
		Handler: func(c asyncadapter.AsyncCtx[pubsub.Message]) error {
			ctx := c.Context()
			l := ctxlogger.GetLogger(ctx)
			p, err := c.Payload()
			if err != nil {
				return fmt.Errorf("failed to get payload: %w", err)
			}

			// TODO: realizar um filtro por eventName para evitar
			events, err := store.GetAllSchedulers(ctx, "archived")
			if errors.Is(err, domain.EventNotFound) {
				return nil
			}

			if err != nil {
				l.Warn("Not found listers events when archived", "event_id", p.ID)
				return fmt.Errorf("failed to get all schedulers: %w", err)
			}

			for _, event := range events {
				for _, consumer := range event.Consumers {
					fetcher.Notify(ctx, map[string]any{
						"event":    event.Name,
						"id":       p.ID,
						"data":     p.Data,
						"metadata": p.Attributes,
						"event_at": p.PublishTime,
					}, consumer.Headers, domain.Consumer{
						ServiceName: consumer.ServiceName,
						BaseUrl:     consumer.BaseUrl,
						Path:        consumer.Path,
						Headers:     consumer.Headers,
					}, notifyopt.HighThroughput)
				}
			}

			return nil
		},
	}
}
