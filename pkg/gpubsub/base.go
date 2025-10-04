package gpubsub

import (
	"context"

	"cloud.google.com/go/pubsub"
)

type Handle struct {
	Event   string
	Handler func(ctx context.Context, msg *pubsub.Message)
}
