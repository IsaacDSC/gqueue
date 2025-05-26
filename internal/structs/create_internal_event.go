package structs

import (
	"github.com/google/uuid"
	"time"
)

type CreateInternalEventDto struct {
	EventName   string `json:"event_name"`
	ServiceName string `json:"service_name"`
	RepoUrl     string `json:"repo_url"`
	TeamOwner   string `json:"team_owner"`
}

func (ie CreateInternalEventDto) ToInternalEvent() InternalEvent {
	return InternalEvent{
		ID:          uuid.New(),
		Name:        ie.EventName,
		ServiceName: ie.ServiceName,
		RepoUrl:     ie.RepoUrl,
		TeamOwner:   ie.TeamOwner,
		Triggers:    []Trigger{},
		CreatedAt:   time.Now(),
	}
}
