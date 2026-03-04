package pubadapter

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type PubSubGoogle struct {
	client *pubsub.Client
}

var _ GenericPublisher = (*PubSubGoogle)(nil)

func NewPubSubGoogle(client *pubsub.Client) *PubSubGoogle {
	return &PubSubGoogle{client: client}
}

func (p *PubSubGoogle) Publish(ctx context.Context, topicName string, payload any, opts Opts) error {
	l := ctxlogger.GetLogger(ctx)

	bytesPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %v", err)
	}

	attributes := opts.Attributes
	if len(opts.Attributes) == 0 {
		attributes = map[string]string{
			"max_retries": "1",
			"topic":       topicName,
		}
	}

	topic := p.client.Topic(topicName)
	result := topic.Publish(ctx, &pubsub.Message{
		Data:       bytesPayload,
		Attributes: attributes,
	})

	id, err := result.Get(ctx)
	if err != nil {
		telemetry.PubSubPublisherRequests.Increment(
			ctx,
			attribute.String("topic", topicName),
			attribute.String("error", err.Error()),
		)

		return fmt.Errorf("could not publish message: %v", err)
	}

	l.Debug("Published message", "msg_id", id, "topic", topicName)

	return nil
}
