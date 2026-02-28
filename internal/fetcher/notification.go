package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IsaacDSC/clienthttp"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/notifyopt"
	"github.com/IsaacDSC/gqueue/pkg/httpclient"
)

type Notification struct{}

func NewNotification() *Notification {
	return &Notification{}
}

func (n Notification) Notify(ctx context.Context, data map[string]any, headers map[string]string, consumer domain.Consumer, opt notifyopt.Kind) error {
	url := consumer.GetUrl()

	settings := make([]clienthttp.Option, 0)
	switch opt {
	case notifyopt.HighThroughput:
		settings = append(settings, httpclient.HighThroughputSettings()...)
	case notifyopt.LongRunning:
		settings = append(settings, httpclient.LongRunningSettings()...)
	default:
		settings = append(settings, httpclient.HighThroughputSettings()...)
	}

	return fetch(ctx, url, data, headers, settings...)
}

func (n Notification) NotifyConsumer(ctx context.Context, url string, data map[string]any, headers map[string]string) error {
	return fetch(ctx, url, data, headers)
}

func (n Notification) NotifyScheduler(ctx context.Context, url string, data any, headers map[string]string) error {
	return fetch(ctx, url, data, headers)
}

func fetch(ctx context.Context, url string, data any, headers map[string]string, settings ...clienthttp.Option) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	bodyReader := bytes.NewReader(payload)

	client := httpclient.NewHTTPClientWithLogging(ctx, settings...) //TODO: melhorar para ser instaciado uma vez só

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// #nosec G704 -- SSRF is intentional: this function sends webhooks to user-configured consumers endpoints
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
