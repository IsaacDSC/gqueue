package repository

import (
	"context"
	"github.com/IsaacDSC/webhook/internal/structs"
)

type Repository interface {
	CreateExternalEvent(ctx context.Context, event structs.ExternalEvent) error
	SaveExternalEvent(ctx context.Context, event structs.ExternalEvent) error
	CreateInternalEvent(ctx context.Context, event structs.InternalEvent) error
	GetInternalEvent(ctx context.Context, eventName string) (structs.InternalEvent, error)
}
