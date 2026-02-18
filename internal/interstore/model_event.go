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
	State       string
	Consumers   []byte
	Option      []byte
}

func (m ModelEvent) ToDomain() domain.Event {
	var consumers []domain.Consumer
	json.Unmarshal(m.Consumers, &consumers)

	var option domain.Opt
	json.Unmarshal(m.Option, &option)

	return domain.Event{
		ID:          m.ID,
		Name:        m.Name,
		ServiceName: m.ServiceName,
		State:       m.State,
		Consumers:   consumers,
		Option:      option,
	}
}
