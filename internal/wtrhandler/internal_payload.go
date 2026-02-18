package wtrhandler

import (
	"encoding/json"
	"fmt"
)

type InternalPayload struct {
	EventName string   `json:"event_name"`
	Data      Data     `json:"data"`
	Metadata  Metadata `json:"metadata"`
}

func (p InternalPayload) Validate() error {
	if p.EventName == "" {
		return fmt.Errorf("event name is required")
	}

	if p.Data == nil {
		return fmt.Errorf("data is required")
	}

	return nil
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
