package interstore

import (
	"context"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, event domain.Event) error
	GetInternalEvent(ctx context.Context, eventName, serviceName string, state string) (domain.Event, error)
	GetInternalEvents(ctx context.Context, filters domain.FilterEvents) ([]domain.Event, error)
	DisabledEvent(ctx context.Context, eventID uuid.UUID) error
	UpdateEvent(ctx context.Context, event domain.Event) error
	GetEventByID(ctx context.Context, eventID uuid.UUID) (domain.Event, error)
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
	GetAllEvents(ctx context.Context) ([]domain.Event, error)
}
