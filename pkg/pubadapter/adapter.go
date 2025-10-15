package pubadapter

import (
	"context"
)

type GenericPublisher interface {
	Publish(ctx context.Context, topicName string, payload any, opts Opts) error
}

type Publisher interface {
	Publish(ctx context.Context, wqtype WQType, topicName string, payload any, opts Opts) error
}
