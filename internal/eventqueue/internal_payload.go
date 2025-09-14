package eventqueue

import (
	"encoding/json"
	"fmt"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/hibiken/asynq"
)

type InternalPayload struct {
	ServiceName string     `json:"service_name"`
	EventName   string     `json:"event_name"`
	Data        Data       `json:"data"`
	Metadata    Metadata   `json:"metadata"`
	Opts        domain.Opt `json:"opts"`
}

func (p InternalPayload) Validate() error {
	if p.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if p.EventName == "" {
		return fmt.Errorf("event name is required")
	}

	if p.Data == nil {
		return fmt.Errorf("data is required")
	}

	if err := p.Opts.Validate(domain.ValidateTypeInternal); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	return nil
}

func (p InternalPayload) GetOpts() []asynq.Option {
	return p.Opts.ToAsynqOptions()
}

type Metadata struct {
	Source      string            `json:"source"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Headers     map[string]string `json:"headers"`
}

type Data map[string]any

func (d Data) ToBytes() []byte {
	data, _ := json.Marshal(d)
	return data
}
