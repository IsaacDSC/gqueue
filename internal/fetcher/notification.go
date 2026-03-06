package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/IsaacDSC/clienthttp"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/notifyopt"
	"github.com/IsaacDSC/gqueue/pkg/httpclient"
	"github.com/IsaacDSC/gqueue/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
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

	return fetch(ctx, url, data, headers, opt, settings...)
}

func (n Notification) NotifyConsumer(ctx context.Context, url string, data map[string]any, headers map[string]string) error {
	return fetch(ctx, url, data, headers, notifyopt.LongRunning)
}

func (n Notification) NotifyScheduler(ctx context.Context, url string, data any, headers map[string]string) error {
	return fetch(ctx, url, data, headers, notifyopt.LongRunning)
}

func fetch(ctx context.Context, url string, data any, headers map[string]string, opt notifyopt.Kind, settings ...clienthttp.Option) error {
	start := time.Now()
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

	attrs := []attribute.KeyValue{
		attribute.String("http.service_name", opt.String()),
		attribute.String("http.method", req.Method),
		attribute.String("http.url", req.URL.String()),
		attribute.Int("http.status_code", resp.StatusCode),
	}

	duration := time.Since(start).Seconds()
	telemetry.HTTPClientRequests.Increment(ctx, attrs...)
	telemetry.HTTPClientRequestDuration.Record(ctx, duration, attrs...)

	if resp.StatusCode > 299 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
