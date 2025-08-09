package domain

import (
	"fmt"
	"github.com/IsaacDSC/webhook/internal/cfg"
	"strings"
)

type Event struct {
	Name        string    `json:"name" bson:"name"`
	ServiceName string    `json:"service_name" bson:"service_name"`
	RepoURL     string    `json:"repo_url" bson:"repo_url"`
	TeamOwner   string    `json:"team_owner" bson:"team_owner"`
	Triggers    []Trigger `json:"triggers" bson:"triggers"`
}

func (e Event) Validate() error {
	for _, trigger := range e.Triggers {
		if err := trigger.Option.IsValid(); err != nil {
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
	MaxRetries int    `json:"max_retries" bson:"max_retries"`
	Timeout    int    `json:"timeout" bson:"timeout"`
	Retention  int    `json:"retention" bson:"retention"`
	UniqueTTL  int    `json:"unique_ttl" bson:"unique_ttl"`
	QueueType  string `json:"queue_type" bson:"queue_type"`
}

func (o Opt) IsValid() error {
	c := cfg.Get()

	if o.QueueType == "" {
		return fmt.Errorf("queue type is required")
	}

	if !strings.Contains(o.QueueType, "external") {
		return fmt.Errorf("invalid queue type: %s, you have use prefix external.<queue-name>", o.QueueType)
	}

	if !c.AsynqConfig.Queues.Contains(o.QueueType) {
		return fmt.Errorf("not found queue type: %s", o.QueueType)
	}

	return nil
}
