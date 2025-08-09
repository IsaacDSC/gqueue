package domain

import (
	"fmt"
	"github.com/IsaacDSC/webhook/internal/cfg"
	"github.com/IsaacDSC/webhook/pkg/intertime"
	"github.com/hibiken/asynq"
	"strings"
	"time"
)

type Event struct {
	Name        string    `json:"name" bson:"name"`
	ServiceName string    `json:"service_name" bson:"service_name"`
	RepoURL     string    `json:"repo_url" bson:"repo_url"`
	TeamOwner   string    `json:"team_owner" bson:"team_owner"`
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
	MaxRetries int                `json:"max_retries" bson:"max_retries"`
	Retention  intertime.Duration `json:"retention" bson:"retention"`
	ScheduleIn intertime.Duration `json:"schedule_in" bson:"schedule_in"`
	UniqueTTL  intertime.Duration `json:"unique_ttl" bson:"unique_ttl"`
	Deadline   *time.Time         `json:"deadline" bson:"deadline"`
	QueueType  string             `json:"queue_type" bson:"queue_type"`
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
	c := cfg.Get()

	if o.QueueType == "" {
		return fmt.Errorf("queue type is required")
	}

	if !strings.Contains(o.QueueType, validateType.String()) {
		return fmt.Errorf("invalid queue type: %s, you have use prefix %s.<queue-name>", o.QueueType, validateType.String())
	}

	if !c.AsynqConfig.Queues.Contains(o.QueueType) {
		return fmt.Errorf("not found queue type: %s", o.QueueType)
	}

	return nil
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

	if o.QueueType != "" {
		opts = append(opts, asynq.Queue(o.QueueType))
	}

	if o.QueueType == "" {
		opts = append(opts, asynq.Queue("internal.low"))
	}

	return opts
}
