package structs

import (
	"github.com/google/uuid"
	"time"
)

type InternalEvent struct {
	ID          uuid.UUID   `bson:"id"`
	Name        string      `bson:"name"`
	ServiceName string      `bson:"serviceName"`
	RepoUrl     string      `bson:"repoUrl"`
	TeamOwner   string      `bson:"teamOwner"`
	Triggers    ListTrigger `bson:"triggers"`
	CreatedAt   time.Time   `bson:"createdAt"`
	UpdatedAt   time.Time   `bson:"updatedAt"`
	DeletedAt   time.Time   `bson:"deletedAt"`
}

type ListTrigger []Trigger

func (lt ListTrigger) Add(input Trigger) ListTrigger {
	return append(lt, input)
}

func (lt ListTrigger) AlreadyExist(input Trigger) bool {
	for _, trigger := range lt {
		if trigger.Path == input.Path {
			return true
		}
	}

	return false
}

type TriggerType string

const (
	TriggerTypeFireForGet    TriggerType = "fireForGet"
	TriggerTypePersistent    TriggerType = "persistent"
	TriggerTypeNotPersistent TriggerType = "notPersistent"
)

type Trigger struct {
	ID          uuid.UUID   `bson:"id"`
	ServiceName string      `bson:"serviceName"`
	Type        TriggerType `bson:"type"`
	BaseUrl     string      `bson:"host"`
	Path        string      `bson:"path"`
	CreatedAt   time.Time   `bson:"createdAt"`
	UpdatedAt   time.Time   `bson:"updatedAt"`
	DeletedAt   time.Time   `bson:"deletedAt"`
}
