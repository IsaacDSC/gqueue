package consworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/IsaacDSC/webhook/internal/intersvc"
	"github.com/IsaacDSC/webhook/pkg/httpclient"
	"github.com/google/uuid"
)

func FetchAll(ctx context.Context, event ExternalPayload) (output TriggersOutput, err error) {
	if len(event.Triggers) == 0 {
		err = fmt.Errorf("no triggers to send, required at least one trigger")
		return
	}

	var wait sync.WaitGroup
	wait.Add(len(event.Triggers))

	triggersErrors := sync.Map{}
	for _, trigger := range event.Triggers {
		go func(trigger Trigger) {
			defer wait.Done()
			if err := fetch(ctx, event.Data, event.ExtraHeaders, trigger); err != nil {
				triggersErrors.Store(createKey(trigger), TriggerError{
					Error:   err.Error(),
					Trigger: trigger,
				})
			}
		}(trigger)
	}

	wait.Wait()

	triggersErrors.Range(func(key, value any) bool {
		if trigger, ok := value.(TriggerError); ok {
			output = append(output, trigger)
		}
		return true // continue iteration
	})

	if len(output) != 0 {
		err = fmt.Errorf("some triggers failed: %v", output)
	}

	return
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

func createKey(trigger Trigger) string {
	return fmt.Sprintf("%s/%s", trigger.BaseUrl, trigger.Path)
}

func eventToExternEvent(payload Payload, interEvent intersvc.InternalEvent) ExternalPayload {
	triggers := make([]Trigger, len(interEvent.Triggers))
	for i, trigger := range interEvent.Triggers {
		triggers[i] = Trigger{
			ServiceName: trigger.ServiceName,
			Type:        TriggerType(trigger.Type),
			BaseUrl:     trigger.BaseUrl,
			Path:        trigger.Path,
		}
	}

	return ExternalPayload{
		EventName:    payload.EventName,
		Data:         payload.Data,
		ExtraHeaders: getDefaultHeaders(),
		Triggers:     triggers,
	}
}

func getDefaultHeaders() map[string]string {
	return map[string]string{
		"Authorization":    "Bearer token",
		"Content-Type":     "application/json",
		"Accept":           "application/json",
		"User-Agent":       "webhook-client/1.0",
		"X-Correlation-ID": uuid.New().String(),
		"X-Request-ID":     uuid.New().String(),
	}
}
