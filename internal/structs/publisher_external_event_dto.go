package structs

import "github.com/google/uuid"

type PublisherExternalEventDto struct {
	EventName string         `json:"event_name"`
	Data      map[string]any `json:"data"`
}

func (p PublisherExternalEventDto) ToExternalEvent(internalEvent InternalEvent) ExternalEvent {
	return ExternalEvent{
		ID:       uuid.New(),
		Name:     p.EventName,
		Data:     p.Data,
		Triggers: internalEvent.Triggers,
	}
}
