package domain

import (
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/pkg/intertime"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type Event struct {
	ID          uuid.UUID `json:"id" bson:"id"`
	Name        string    `json:"name" bson:"name"`
	ServiceName string    `json:"service_name" bson:"service_name"`
	State       string    `json:"state" bson:"state"`
	Type        Type      `json:"type" bson:"type"`
	Triggers    []Trigger `json:"triggers" bson:"triggers"`
}

func (e *Event) Validate() error {
	if e.Type != TriggerTypeInternal && e.Type != TriggerTypeExternal {
		return fmt.Errorf("invalid event type: %s", e.Type)
	}

	for _, trigger := range e.Triggers {
		if err := trigger.Option.Validate(); err != nil {
			return fmt.Errorf("invalid trigger option: %w", err)
		}
	}

	return nil
}

type Trigger struct {
	ServiceName string            `json:"service_name" bson:"service_name"`
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

type Type string

func (vt Type) String() string {
	return string(vt)
}

const (
	TriggerTypeInternal Type = "internal"
	TriggerTypeExternal Type = "external"
)

func (o Opt) Validate() error {
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
