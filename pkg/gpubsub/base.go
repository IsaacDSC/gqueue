package gpubsub

import (
	"context"

	"cloud.google.com/go/pubsub"
)

type Handle struct {
	TopicName string
	Handler   func(ctx context.Context, msg *pubsub.Message)
}
