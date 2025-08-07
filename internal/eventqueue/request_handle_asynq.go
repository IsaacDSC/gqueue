package eventqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/IsaacDSC/webhook/pkg/asynqsvc"
	"github.com/IsaacDSC/webhook/pkg/httpclient"
	"github.com/hibiken/asynq"
	"net/http"
)

type RequestPayload struct {
	EventName string            `json:"event_name"`
	Trigger   Trigger           `json:"trigger"`
	Data      map[string]any    `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func (p RequestPayload) mergeHeaders(headers map[string]string) map[string]string {
	if p.Headers == nil {
		p.Headers = make(map[string]string)
	}

	for key, value := range headers {
		p.Headers[key] = value
	}

	return p.Headers
}

func GetRequestHandle() asynqsvc.AsynqHandle {
	return asynqsvc.AsynqHandle{
		Event: "event-queue.request-to-external",
		Handler: func(ctx context.Context, task *asynq.Task) error {
			var payload RequestPayload
			if err := json.Unmarshal(task.Payload(), &payload); err != nil {
				return fmt.Errorf("unmarshal payload: %w", err)
			}

			headers := payload.mergeHeaders(payload.Trigger.Headers)
			if err := fetch(ctx, payload.Data, headers, payload.Trigger); err != nil {
				return fmt.Errorf("fetch trigger: %w", err)
			}

			return nil
		},
	}
}

func fetch(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	url := trigger.GetUrl()

	bodyReader := bytes.NewReader(payload)

	client := httpclient.NewHTTPClientWithLogging(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
