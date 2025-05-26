package gateway

import (
	"context"
	"github.com/IsaacDSC/webhook/internal/structs"
)

type Gateway interface {
	Publisher(ctx context.Context, event structs.ExternalEvent) (sent []structs.Trigger, err error)
}
