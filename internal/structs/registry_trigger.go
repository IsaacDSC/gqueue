package structs

import (
	"github.com/google/uuid"
	"time"
)

type RegisterTriggersDto struct {
	EventName string     `json:"eventName"`
	Trigger   TriggerDto `json:"trigger"`
}

type TriggerDto struct {
	ServiceName string `bson:"serviceName"`
	Type        string `bson:"type"`
	BaseUrl     string `bson:"host"`
	Path        string `bson:"path"`
}

func (rt RegisterTriggersDto) ToTrigger() Trigger {
	return Trigger{
		ID:          uuid.New(),
		ServiceName: rt.Trigger.ServiceName,
		Type:        TriggerType(rt.Trigger.Type),
		BaseUrl:     rt.Trigger.BaseUrl,
		CreatedAt:   time.Now(),
		Path:        rt.Trigger.Path,
	}
}
