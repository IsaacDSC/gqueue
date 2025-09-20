package eventqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/IsaacDSC/gqueue/pkg/logs"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/hibiken/asynq"
)

type Repository interface {
	GetInternalEvent(ctx context.Context, eventName, serviceName string, eventType string, state string) (domain.Event, error)
}

func GetInternalConsumerHandle(repo Repository, cc cachemanager.Cache, publisher publisher.Publisher) asynqsvc.AsynqHandle {
	return asynqsvc.AsynqHandle{
		Event: "event-queue.internal",
		Handler: func(ctx context.Context, task *asynq.Task) error {
			var payload InternalPayload
			if err := json.Unmarshal(task.Payload(), &payload); err != nil {
				return fmt.Errorf("unmarshal payload: %w", err)
			}

			var event domain.Event
			key := cc.Key(domain.CacheKeyEventPrefix, payload.EventName)

			err := cc.Once(ctx, key, &event, cc.GetDefaultTTL(), func(ctx context.Context) (any, error) {
				return repo.GetInternalEvent(ctx, payload.EventName, payload.ServiceName, "trigger", "active")
			})

			if errors.Is(err, domain.EventNotFound) {
				// TODO: DEVE SER DISCARTADO O EVENTO >> SALVAR EVENTOS DISCARTADOS >> USAR O PADRÃO DO ASYNQ DE DISCARD
				logs.Warn("Event not found", "eventName", payload.EventName)
				return nil
			}

			if err != nil {
				logs.Error("error on consuming internal event", "eventName", payload.EventName, "error", err)
				return fmt.Errorf("get internal event: %w", err)
			}

			for _, tt := range event.Triggers {
				config := tt.Option.ToAsynqOptions()

				input := RequestPayload{
					EventName: event.Name,
					Data:      payload.Data,
					Headers:   payload.Metadata.Headers,
					Trigger: Trigger{
						ServiceName: tt.ServiceName,
						Type:        TriggerType(tt.Type),
						BaseUrl:     tt.Host,
						Path:        tt.Path,
						Headers:     tt.Headers,
					},
				}

				if err := publisher.Publish(ctx, "event-queue.request-to-external", input, config...); err != nil {
					return fmt.Errorf("publish internal event: %w", err)
				}
			}

			return nil
		},
	}
}
