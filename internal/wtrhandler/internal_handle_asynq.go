package wtrhandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/IsaacDSC/gqueue/pkg/logs"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
)

type Repository interface {
	GetInternalEvent(ctx context.Context, eventName, serviceName string, eventType string, state string) (domain.Event, error)
}

func GetInternalConsumerHandle(repo Repository, cc cachemanager.Cache, pub publisher.Publisher) asyncadapter.Handle[InternalPayload] {
	return asyncadapter.Handle[InternalPayload]{
		Event: domain.EventQueueInternal,
		Handler: func(c asyncadapter.AsyncCtx[InternalPayload]) error {
			ctx := c.Context()

			payload, err := c.Payload()
			if err != nil {
				return fmt.Errorf("get payload: %w", err)
			}

			var event domain.Event
			key := cc.Key(domain.CacheKeyEventPrefix, payload.EventName)

			err = cc.Once(ctx, key, &event, cc.GetDefaultTTL(), func(ctx context.Context) (any, error) {
				return repo.GetInternalEvent(ctx, payload.EventName, payload.ServiceName, "trigger", "active")
			})

			if errors.Is(err, domain.EventNotFound) {
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

				topic := topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueRequestToExternal)
				opts := publisher.Opts{Attributes: make(map[string]string), AsynqOpts: config}
				if err := pub.Publish(ctx, topic, input, opts); err != nil {
					return fmt.Errorf("publish internal event: %w", err)
				}
			}

			return nil
		},
	}
}
