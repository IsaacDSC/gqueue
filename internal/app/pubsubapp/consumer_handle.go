package pubsubapp

import (
	"context"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/notifyopt"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/telemetry"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"
	"go.opentelemetry.io/otel/attribute"
)

type Fetcher interface {
	Notify(ctx context.Context, data map[string]any, headers map[string]string, consumer domain.Consumer, opt notifyopt.Kind) error
}

type ConsumerInsights interface {
	Consumed(ctx context.Context, input domain.ConsumerMetric) error
}

func GetRequestHandle(fetch Fetcher, insights ConsumerInsights) asyncadapter.Handle[RequestPayload] {

	insertInsights := func(ctx context.Context, payload RequestPayload, started time.Time, isSuccess bool) {
		l := ctxlogger.GetLogger(ctx)
		finished := time.Now()
		if err := insights.Consumed(ctx, domain.ConsumerMetric{
			TopicName:      payload.EventName,
			ConsumerName:   payload.Consumer.ServiceName,
			TimeStarted:    started,
			TimeEnded:      finished,
			TimeDurationMs: finished.Sub(started).Milliseconds(),
			ACK:            isSuccess,
		}); err != nil {
			l.Warn("not save metric", "type", "consumer", "error", err.Error())
		}

	}

	return asyncadapter.Handle[RequestPayload]{
		EventName: domain.EventQueueRequestToExternal,
		Handler: func(c asyncadapter.AsyncCtx[RequestPayload]) error {
			started := time.Now()
			ctx := c.Context()

			payload, err := c.Payload()
			if err != nil {
				return fmt.Errorf("get payload: %w", err)
			}

			headers := payload.mergeHeaders(payload.Consumer.Headers)
			if err := fetch.Notify(ctx, payload.Data, headers, payload.Consumer, notifyopt.HighThroughput); err != nil {
				insertInsights(ctx, payload, started, false)
				recordDuration(ctx, started, payload, err)
				return fmt.Errorf("fetch consumer: %w", err)
			}

			publishedTime := time.UnixMilli(payload.PublishedAt)
			lag := started.Sub(publishedTime).Seconds()
			topic := topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueRequestToExternal)
			telemetry.PubSubConsumerLagSeconds.Record(ctx, lag,
				attribute.String("topic", topic),
				attribute.String("consumer.service_name", payload.Consumer.ServiceName))

			insertInsights(ctx, payload, started, true)
			recordDuration(ctx, started, payload, nil)

			return nil
		},
	}
}

func recordDuration(ctx context.Context, started time.Time, payload RequestPayload, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("consumer.app_name", cfg.PUBSUB_APP_NAME),
		attribute.String("consumer.base_url", payload.Consumer.BaseUrl),
		attribute.String("consumer.path", payload.Consumer.Path),
		attribute.String("consumer.service_name", payload.Consumer.ServiceName),
	}

	if err != nil {
		attrs = append(attrs, attribute.Bool("success", false))
		attrs = append(attrs, attribute.String("error", err.Error()))
	} else {
		attrs = append(attrs, attribute.Bool("success", true))
	}

	duration := time.Since(started).Seconds()
	telemetry.PubSubConsumerDuration.Record(
		ctx, duration,
		attrs...,
	)
}
