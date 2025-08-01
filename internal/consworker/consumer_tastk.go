package consworker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IsaacDSC/webhook/internal/intersvc"
	"github.com/hibiken/asynq"
)

type Payload struct {
	EventName string            `json:"event_name"`
	Data      map[string]any    `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
	Memento   TriggersOutput    `json:"memento"`
}

func (p Payload) toFetcherInput() ExternalPayload {
	return ExternalPayload{
		EventName:    p.EventName,
		Data:         p.Data,
		ExtraHeaders: getDefaultHeaders(),
		Triggers:     p.Memento.ToTrigger(),
	}
}

type getInternalEvent func(ctx context.Context, eventName string) (output intersvc.InternalEvent, err error)

func GetInternalConsumerHandle(getEventFn getInternalEvent) func(ctx context.Context, task *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		var payload Payload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}

		var (
			triggers TriggersOutput
			err      error
		)

		if !payload.Memento.Exist() {
			event, err := getEventFn(ctx, payload.EventName)
			if err != nil {
				return fmt.Errorf("get internal event: %w", err)
			}
			triggers, err = FetchAll(ctx, eventToExternEvent(payload, event))
		} else {
			triggers, err = FetchAll(ctx, payload.toFetcherInput())
		}

		if err != nil {
			hiddenPayload, err := hydratePayload(payload, triggers)
			if err != nil {
				return fmt.Errorf("hydrate payload: %w", err)
			}

			task.ResultWriter().Write(hiddenPayload)

			return fmt.Errorf("fetch triggers: %w", err)
		}

		return nil
	}
}

func hydratePayload(event Payload, memento TriggersOutput) ([]byte, error) {
	hydrated := Payload{
		EventName: event.EventName,
		Data:      event.Data,
		Memento:   memento,
	}

	data, err := json.Marshal(hydrated)
	if err != nil {
		return nil, fmt.Errorf("marshal hydrated payload: %w", err)
	}

	return data, nil
}
