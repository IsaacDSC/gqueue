package eventqueue

import (
	"fmt"
	"strings"
)

type TriggerType string

const (
	TriggerTypeFireForGet    TriggerType = "fireForGet"
	TriggerTypePersistent    TriggerType = "persistent"
	TriggerTypeNotPersistent TriggerType = "notPersistent"
)

type Trigger struct {
	ServiceName string            `json:"service_name"`
	Type        TriggerType       `json:"type"`
	BaseUrl     string            `json:"base_url"`
	Path        string            `json:"path"`
	Headers     map[string]string `bson:"headers"`
}

func (t *Trigger) GetUrl() string {
	baseURL := strings.TrimSuffix(t.BaseUrl, "/")
	path := strings.TrimPrefix(t.Path, "/")
	return fmt.Sprintf("%s/%s", baseURL, path)
}
