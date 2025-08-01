package consworker

import (
	"fmt"
	"strings"
)

type ExternalPayload struct {
	EventName    string
	Data         map[string]any
	ExtraHeaders map[string]string
	Triggers     []Trigger
}

type TriggerType string

const (
	TriggerTypeFireForGet    TriggerType = "fireForGet"
	TriggerTypePersistent    TriggerType = "persistent"
	TriggerTypeNotPersistent TriggerType = "notPersistent"
)

type Trigger struct {
	ServiceName string
	Type        TriggerType
	BaseUrl     string
	Path        string
}

func (t *Trigger) GetUrl() string {
	baseURL := strings.TrimSuffix(t.BaseUrl, "/")
	path := strings.TrimPrefix(t.Path, "/")
	return fmt.Sprintf("%s/%s", baseURL, path)
}

type TriggerError struct {
	Trigger Trigger `json:"trigger"`
	Error   string  `json:"error,omitempty"`
}

type TriggersOutput []TriggerError

func (t TriggersOutput) Exist() bool {
	return len(t) > 0
}

func (t TriggersOutput) ToTrigger() []Trigger {
	triggers := make([]Trigger, len(t))
	for i, triggerError := range t {
		triggers[i] = triggerError.Trigger
	}
	return triggers
}
