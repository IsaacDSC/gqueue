package service

import (
	"context"
	"github.com/IsaacDSC/webhook/internal/structs"
)

// Service defines the interface for the service layer operations
type Service interface {
	CreateInternalEvent(ctx context.Context, input structs.CreateInternalEventDto) (structs.InternalEvent, error)
	PublisherExternalEvent(ctx context.Context, input structs.PublisherExternalEventDto) error
	RegisterTrigger(ctx context.Context, input structs.RegisterTriggersDto) (structs.InternalEvent, error)
}
