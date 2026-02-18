package wtrhandler

import (
	"fmt"
	"strings"
)

type Consumer struct {
	ServiceName string            `json:"service_name"`
	BaseUrl     string            `json:"base_url"`
	Path        string            `json:"path"`
	Headers     map[string]string `bson:"headers"`
}

func (t *Consumer) GetUrl() string {
	baseURL := strings.TrimSuffix(t.BaseUrl, "/")
	path := strings.TrimPrefix(t.Path, "/")
	return fmt.Sprintf("%s/%s", baseURL, path)
}
