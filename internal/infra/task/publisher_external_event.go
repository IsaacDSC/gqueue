package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IsaacDSC/webhook/internal/structs"
	"github.com/hibiken/asynq"
)

func (t Tasks) publisherExternalEvent(ctx context.Context, task *asynq.Task) error {
	var payload structs.PublisherExternalEventDto
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	internalEvent, err := t.repo.GetInternalEvent(ctx, payload.EventName)
	if err != nil {
		return fmt.Errorf("get internal event: %w", err)
	}

	if len(internalEvent.Triggers) == 0 {
		return fmt.Errorf("no triggers: %w", asynq.SkipRetry)
	}

	externalEvent := payload.ToExternalEvent(internalEvent)
	if err := t.repo.CreateExternalEvent(ctx, externalEvent); err != nil {
		return fmt.Errorf("save external event: %w", err)
	}

	externalEvent.Delivered, err = t.gate.Publisher(ctx, externalEvent)
	if err != nil {
		return fmt.Errorf("gateway publish event: %w", err)
	}

	if len(externalEvent.Delivered) > 0 {
		if err := t.repo.SaveExternalEvent(ctx, externalEvent); err != nil {
			return fmt.Errorf("save updated external event: %w", err)
		}
	}

	if len(externalEvent.Delivered) != len(externalEvent.Triggers) {
		return errors.New("published triggers don't match")
	}

	return nil
}
