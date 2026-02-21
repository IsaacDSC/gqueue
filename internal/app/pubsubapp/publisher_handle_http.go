package pubsubapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"
)

type PublisherInsights interface {
	Published(ctx context.Context, input domain.PublisherMetric) error
}

type Store interface {
	GetEvent(ctx context.Context, eventName string) (domain.Event, error)
}

type RequestPayload struct {
	EventName string            `json:"event_name"`
	Consumer  domain.Consumer   `json:"consumer"`
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

func PublisherEvent(
	store Store,
	adaptpub pubadapter.GenericPublisher,
	insights PublisherInsights,
) httpadapter.HttpHandle {

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

	return httpadapter.HttpHandle{
		Path: "POST /api/v1/pubsub",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			ctx := r.Context()
			l := ctxlogger.GetLogger(ctx)

			var payload InternalPayload

			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := payload.Validate(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var err error
			defer insertInsights(ctx, payload, started, err == nil)

			event, err := store.GetEvent(ctx, payload.EventName)
			if errors.Is(err, domain.EventNotFound) {
				err = fmt.Errorf("event not found: %w", err)
				http.Error(w, "event not found", http.StatusNotFound)
				return
			}
			if err != nil {
				err = fmt.Errorf("get event: %w", err)
				l.Error("failed to get event", "error", err.Error())
				http.Error(w, "failed to get event", http.StatusInternalServerError)
				return
			}

			eventType := event.Type.String()
			if eventType == "" {
				l.Warn("event type is empty, defaulting to internal", "event_name", event.Name)
				eventType = domain.EventTypeInternal.String()
			}

			config := event.Option.ToAsynqOptions()
			for _, consumer := range event.Consumers {

				input := RequestPayload{
					EventName: event.Name,
					Data:      payload.Data,
					Headers:   payload.Metadata.Headers,
					Consumer: domain.Consumer{
						ServiceName: consumer.ServiceName,
						BaseUrl:     consumer.BaseUrl,
						Path:        consumer.Path,
						Headers:     consumer.Headers,
					},
				}

				topic := topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueRequestToExternal)
				opts := pubadapter.Opts{Attributes: make(map[string]string), AsynqOpts: config}
				if err = adaptpub.Publish(ctx, topic, input, opts); err != nil {
					err = fmt.Errorf("publish event: %w", err)
					l.Error("failed to publish event", "error", err.Error())
					http.Error(w, "failed to publish event", http.StatusInternalServerError)
					return
				}
			}

			w.WriteHeader(http.StatusCreated)
		},
	}
}
