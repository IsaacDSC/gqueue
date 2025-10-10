package wtrhandler

import (
	"context"

	"github.com/IsaacDSC/gqueue/internal/domain"
)

type PublisherInsights interface {
	Published(ctx context.Context, input domain.PublisherInsights) error
}
