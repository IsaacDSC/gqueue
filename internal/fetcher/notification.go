package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/wtrhandler"
	"github.com/IsaacDSC/gqueue/pkg/httpclient"
)

type Notification struct{}

func NewNotification() *Notification {
	return &Notification{}
}

func (n Notification) NotifyTrigger(ctx context.Context, data map[string]any, headers map[string]string, trigger wtrhandler.Trigger) error {
	url := trigger.GetUrl()
	return fetch(ctx, url, data, headers)
}

func (n Notification) NotifyConsumer(ctx context.Context, url string, data map[string]any, headers map[string]string) error {
	return fetch(ctx, url, data, headers)
}

func (n Notification) NotifyScheduler(ctx context.Context, url string, data any, headers map[string]string) error {
	return fetch(ctx, url, data, headers)
}

func fetch(ctx context.Context, url string, data any, headers map[string]string) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	bodyReader := bytes.NewReader(payload)

	client := httpclient.NewHTTPClientWithLogging(ctx) //TODO: melhorar para ser instaciado uma vez só

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// #nosec G704 -- SSRF is intentional: this function sends webhooks to user-configured trigger endpoints
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
