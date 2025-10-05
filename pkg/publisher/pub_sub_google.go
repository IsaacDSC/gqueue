package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

type PubSubGoogle struct {
	client *pubsub.Client
}

var _ Publisher = (*PubSubGoogle)(nil)

func NewPubSubGoogle(client *pubsub.Client) *PubSubGoogle {
	return &PubSubGoogle{client: client}
}

func (p *PubSubGoogle) Publish(ctx context.Context, topicName string, payload any, opts Opts) error {
	l := ctxlogger.GetLogger(ctx)
	l.Info("[*] Publisher msg to topic", "topic", topicName)

	bytesPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %v", err)
	}

	attributes := make(map[string]string)
	if len(opts.Attributes) == 0 {
		attributes = map[string]string{
			"max_attempts": "3",
			"topic":        topicName,
		}
	} else {
		attributes = opts.Attributes
	}

	topic := p.client.Topic(topicName)
	result := topic.Publish(ctx, &pubsub.Message{
		Data:       bytesPayload,
		Attributes: attributes,
	})

	id, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("could not publish message: %v", err)
	}

	l.Debug("Published message", "msg_id", id, "topic", topicName)

	return nil
}
