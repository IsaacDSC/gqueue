package pubadapter

import (
	"context"
	"fmt"
)

type PublisherStrategy struct {
	InternalPublisher GenericPublisher
	ExternalPublisher GenericPublisher
}

var _ GenericPublisher = (*PublisherStrategy)(nil)

func NewStrategy(classificationResult *ClassificationResult) *PublisherStrategy {
	return &PublisherStrategy{
		InternalPublisher: classificationResult.InternalPublisher,
		ExternalPublisher: classificationResult.ExternalPublisher,
	}
}

func (s *PublisherStrategy) Publish(ctx context.Context, eventName string, payload any, opts Opts) error {
	switch opts.Type {
	case "internal":
		return s.InternalPublisher.Publish(ctx, eventName, payload, opts)
	case "external":
		return s.ExternalPublisher.Publish(ctx, eventName, payload, opts)
	default:
		return fmt.Errorf("invalid publish type: %s", opts.Type)
	}
}
