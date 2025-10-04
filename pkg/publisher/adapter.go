package publisher

import (
	"context"
)

type Publisher interface {
	Publish(ctx context.Context, topicName string, payload any, opts Opts) error
}
