package interstore

import (
	"context"

	"github.com/IsaacDSC/gqueue/internal/domain"
)

type Repository interface {
	GetInternalEvent(ctx context.Context, eventName string) (domain.Event, error)
	Save(ctx context.Context, event domain.Event) error
}
