package publisher

import (
	"context"

	"github.com/hibiken/asynq"
)

type Publisher interface {
	Publish(ctx context.Context, topicName string, payload any, opts ...asynq.Option) error
}
