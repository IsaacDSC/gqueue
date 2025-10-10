package wtrhandler

import (
	"context"

	"github.com/IsaacDSC/gqueue/internal/domain"
)

type ConsumerInsights interface {
	Consumed(ctx context.Context, input domain.ConsumerInsights) error
}
