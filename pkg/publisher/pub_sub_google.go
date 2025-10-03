package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/hibiken/asynq"
)

type PubSubGoogle struct {
	pub *pubsub.Publisher
}

var _ Publisher = (*PubSubGoogle)(nil)

func NewPubSubGoogle(pub *pubsub.Publisher) *PubSubGoogle {
	return &PubSubGoogle{pub: pub}
}

func (p *PubSubGoogle) Publish(ctx context.Context, topicName string, payload any, opts ...asynq.Option) error {
	l := ctxlogger.GetLogger(ctx)

	bytesPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %v", err)
	}

	result := p.pub.Publish(ctx, &pubsub.Message{Data: bytesPayload})
	id, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("could not publish message: %v", err)
	}

	l.Debug("Published message", "msg_id", id, "topic", topicName)

	return nil
}
