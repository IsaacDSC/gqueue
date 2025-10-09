package interstore

import (
	"encoding/json"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/google/uuid"
)

type ModelEvent struct {
	ID          uuid.UUID
	Name        string
	ServiceName string
	RepoURL     string
	TeamOwner   string
	TypeEvent   string
	State       string
	Triggers    []byte
}

func (m ModelEvent) ToDomain() domain.Event {
	typeEvent, _ := domain.TypeEventTrigger.NewTypeEvent(m.TypeEvent)

	var triggers []domain.Trigger
	json.Unmarshal(m.Triggers, &triggers)

	return domain.Event{
		ID:          m.ID,
		Name:        m.Name,
		ServiceName: m.ServiceName,
		RepoURL:     m.RepoURL,
		TeamOwner:   m.TeamOwner,
		TypeEvent:   typeEvent,
		State:       m.State,
		Triggers:    triggers,
	}
}
