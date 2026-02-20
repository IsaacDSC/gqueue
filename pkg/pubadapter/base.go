package pubadapter

import (
	"context"
	"fmt"
)

type PublisherStrategy struct {
	TasksPublisher  GenericPublisher
	PubsubPublisher GenericPublisher
}

var _ GenericPublisher = (*PublisherStrategy)(nil)

func NewStrategy(classificationResult *ClassificationResult) *PublisherStrategy {
	return &PublisherStrategy{
		TasksPublisher:  classificationResult.InternalPublisher,
		PubsubPublisher: classificationResult.ExternalPublisher,
	}
}

func (s *PublisherStrategy) Publish(ctx context.Context, eventName string, payload any, opts Opts) error {
	switch opts.WQType {
	case LowThroughput, HighThroughput:
		return s.TasksPublisher.Publish(ctx, eventName, payload, opts)
	case LowLatency:
		return s.PubsubPublisher.Publish(ctx, eventName, payload, opts)
	default:
		return fmt.Errorf("invalid publish type: %s", opts.WQType)
	}
}
