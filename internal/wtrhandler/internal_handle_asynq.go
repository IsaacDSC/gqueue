package wtrhandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/logs"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
)

type Repository interface {
	GetInternalEvent(ctx context.Context, eventName, serviceName string, eventType string, state string) (domain.Event, error)
}

type PublisherInsights interface {
	Published(ctx context.Context, input domain.PublisherMetric) error
}

func GetInternalConsumerHandle(repo Repository, cc cachemanager.Cache, pub pubadapter.Publisher, insights PublisherInsights) asyncadapter.Handle[InternalPayload] {

	insertInsights := func(ctx context.Context, payload InternalPayload, started time.Time, isSuccess bool) {
		l := ctxlogger.GetLogger(ctx)
		finished := time.Now()

		if err := insights.Published(ctx, domain.PublisherMetric{
			TopicName:      payload.EventName,
			TimeStarted:    started,
			TimeEnded:      finished,
			TimeDurationMs: finished.Sub(started).Milliseconds(),
			ACK:            true,
		}); err != nil {
			l.Warn("not save metric", "type", "publisher", "error", err.Error())
		}

	}

	return asyncadapter.Handle[InternalPayload]{
		EventName: domain.EventQueueInternal,
		Handler: func(c asyncadapter.AsyncCtx[InternalPayload]) (err error) {
			started := time.Now()
			ctx := c.Context()

			payload, err := c.Payload()
			if err != nil {
				err = fmt.Errorf("get payload: %w", err)
				return
			}

			defer insertInsights(ctx, payload, started, err == nil)

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
				err = fmt.Errorf("get internal event: %w", err)
				return
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
				opts := pubadapter.Opts{Attributes: make(map[string]string), AsynqOpts: config}
				if err = pub.Publish(ctx, payload.Opts.WqType, topic, input, opts); err != nil {
					err = fmt.Errorf("publish internal event: %w", err)
					return
				}
			}

			return
		},
	}
}
