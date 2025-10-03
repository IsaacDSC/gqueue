package interstore

import (
	"context"

	"github.com/IsaacDSC/gqueue/internal/domain"
)

type Repository interface {
	GetInternalEvent(ctx context.Context, eventName, serviceName string, eventType string, state string) (domain.Event, error)
	Save(ctx context.Context, event domain.Event) error
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
	GetInternalEvents(ctx context.Context, filters domain.FilterEvents) ([]domain.Event, error)
}
