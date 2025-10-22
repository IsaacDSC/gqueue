package domain

import (
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/pkg/intertime"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type TypeEvent int

const (
	TypeEventTrigger TypeEvent = iota + 1
	TypeEventSchedule
)

var typeEventList = [3]string{"trigger", "trigger", "schedule"}

func (te TypeEvent) NewTypeEvent(input string) (TypeEvent, error) {
	for i := range typeEventList {
		if typeEventList[i] == input {
			return TypeEvent(i), nil
		}
	}

	return -1, fmt.Errorf("invalid type event: %s", input)
}

func (te TypeEvent) String() string {
	return typeEventList[te]
}

type Event struct {
	ID          uuid.UUID `json:"id" bson:"id"`
	Name        string    `json:"name" bson:"name"`
	ServiceName string    `json:"service_name" bson:"service_name"`
	RepoURL     string    `json:"repo_url" bson:"repo_url"`
	TeamOwner   string    `json:"team_owner" bson:"team_owner"`
	TypeEvent   TypeEvent `json:"type_event" bson:"type_event"`
	State       string    `json:"state" bson:"state"`
	Triggers    []Trigger `json:"triggers" bson:"triggers"`
}

func (e Event) Validate(validateType ValidateType) error {
	for _, trigger := range e.Triggers {
		if err := trigger.Option.Validate(validateType); err != nil {
			return fmt.Errorf("invalid trigger option: %w", err)
		}
	}

	return nil
}

type Trigger struct {
	ServiceName string            `json:"service_name" bson:"service_name"`
	Type        string            `json:"type" bson:"type"`
	Host        string            `json:"host" bson:"host"`
	Path        string            `json:"path" bson:"path"`
	Headers     map[string]string `json:"headers" bson:"headers"`
	Option      Opt               `json:"option" bson:"option"`
}

type Opt struct {
	// TODO: mover para uma conflig global ao invéz de usar por produtor
	Deadline  *time.Time         `json:"deadline" bson:"deadline"`
	Retention intertime.Duration `json:"retention" bson:"retention"`
	// REDIS
	UniqueTTL  intertime.Duration `json:"unique_ttl" bson:"unique_ttl"`
	ScheduleIn intertime.Duration `json:"schedule_in" bson:"schedule_in"`
	// ALL
	MaxRetries int               `json:"max_retries" bson:"max_retries"`
	WqType     pubadapter.WQType `json:"wq_type" bson:"wq_type"`
}

type ValidateType string

func (vt ValidateType) String() string {
	return string(vt)
}

const (
	ValidateTypeInternal ValidateType = "internal"
	ValidateTypeExternal ValidateType = "external"
)

func (o Opt) Validate(validateType ValidateType) error {
	if o.WqType == "" {
		return fmt.Errorf("wq type is required")
	}

	if err := o.WqType.Validate(); err != nil {
		return fmt.Errorf("invalid worker_type %w", err)
	}

	if o.MaxRetries < 0 || o.MaxRetries > 5 {
		return fmt.Errorf("max retries must be between 0 and 5")
	}

	return nil
}

func (o Opt) Attributes() map[string]string {
	return map[string]string{
		"max_retries": fmt.Sprintf("%d", o.MaxRetries),
		"wq_type":     fmt.Sprintf("%s", o.WqType),
	}
}

func (o Opt) ToAsynqOptions() []asynq.Option {
	opts := []asynq.Option{}

	if o.MaxRetries > 0 {
		opts = append(opts, asynq.MaxRetry(int(o.MaxRetries)))
	}
	if o.Retention > 0 {
		opts = append(opts, asynq.Retention(time.Duration(o.Retention)))
	}
	if o.Deadline != nil {
		opts = append(opts, asynq.Deadline(*o.Deadline))
	}
	if o.UniqueTTL > 0 {
		opts = append(opts, asynq.Unique(time.Duration(o.UniqueTTL)))
	}
	if o.ScheduleIn > 0 {
		opts = append(opts, asynq.ProcessIn(time.Duration(o.ScheduleIn)))
	}

	return opts
}
