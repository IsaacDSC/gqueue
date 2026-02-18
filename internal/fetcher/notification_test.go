package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/IsaacDSC/gqueue/internal/wtrhandler"
)

func init() {
	// Set required environment variables for tests
	os.Setenv("GO_ENV", "test")
	os.Setenv("WQ_QUEUES", `{"internal.critical": 7, "internal.high": 5, "internal.medium": 3, "internal.low": 1, "external.critical": 7, "external.high": 5, "external.medium": 3, "external.low": 1}`)
	os.Setenv("CACHE_ADDR", "localhost:6379")
	os.Setenv("DB_DRIVER", "pg")
	os.Setenv("DB_CONNECTION_STRING", "postgresql://test:test@localhost:5432/test?sslmode=disable")
	os.Setenv("WQ_CONCURRENCY", "32")
}

func TestNotification_NotifyTrigger(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]any
		headers        map[string]string
		consumer       wtrhandler.Consumer
		serverResponse serverResponse
		wantErr        bool
		errContains    string
	}{
		{
			name: "successful notification with 200 status",
			data: map[string]any{
				"event_id": "12345",
				"message":  "test notification",
				"user_id":  67890,
			},
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "custom-value",
			},
			consumer: wtrhandler.Consumer{
				ServiceName: "test-service",
				BaseUrl:     "",
				Path:        "/webhook",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusOK,
				body:       `{"status": "success"}`,
			},
			wantErr: false,
		},
		{
			name: "successful notification with 201 status",
			data: map[string]any{
				"event_id": "67890",
				"type":     "user_created",
			},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			consumer: wtrhandler.Consumer{
				ServiceName: "user-service",
				BaseUrl:     "",
				Path:        "/users/webhook",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusCreated,
				body:       `{"id": "new-resource-id"}`,
			},
			wantErr: false,
		},
		{
			name: "successful notification with 204 status (no content)",
			data: map[string]any{
				"ping": "pong",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "ping-service",
				BaseUrl:     "",
				Path:        "/ping",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusNoContent,
				body:       "",
			},
			wantErr: false,
		},
		{
			name: "successful notification with 299 status (edge case)",
			data: map[string]any{
				"test": "boundary",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "boundary-service",
				BaseUrl:     "",
				Path:        "/boundary",
			},
			serverResponse: serverResponse{
				statusCode: 299,
				body:       `{"boundary": "test"}`,
			},
			wantErr: false,
		},
		{
			name: "failed notification with 400 bad request",
			data: map[string]any{
				"invalid": "data",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "validation-service",
				BaseUrl:     "",
				Path:        "/validate",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusBadRequest,
				body:       `{"error": "invalid data"}`,
			},
			wantErr:     true,
			errContains: "unexpected status code: 400",
		},
		{
			name: "failed notification with 401 unauthorized",
			data: map[string]any{
				"sensitive": "data",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "auth-service",
				BaseUrl:     "",
				Path:        "/secure",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusUnauthorized,
				body:       `{"error": "unauthorized"}`,
			},
			wantErr:     true,
			errContains: "unexpected status code: 401",
		},
		{
			name: "failed notification with 404 not found",
			data: map[string]any{
				"event": "not-found",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "missing-service",
				BaseUrl:     "",
				Path:        "/missing",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusNotFound,
				body:       `{"error": "not found"}`,
			},
			wantErr:     true,
			errContains: "unexpected status code: 404",
		},
		{
			name: "failed notification with 500 internal server error",
			data: map[string]any{
				"event": "server-error",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "error-service",
				BaseUrl:     "",
				Path:        "/error",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusInternalServerError,
				body:       `{"error": "internal server error"}`,
			},
			wantErr:     true,
			errContains: "unexpected status code: 500",
		},
		{
			name: "failed notification with 300 status (redirect - edge case)",
			data: map[string]any{
				"redirect": "test",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "redirect-service",
				BaseUrl:     "",
				Path:        "/redirect",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusMultipleChoices,
				body:       `{"location": "somewhere"}`,
			},
			wantErr:     true,
			errContains: "unexpected status code: 300",
		},
		{
			name: "trigger with complex URL construction",
			data: map[string]any{
				"complex": "url",
			},
			headers: map[string]string{},
			consumer: wtrhandler.Consumer{
				ServiceName: "complex-service",
				BaseUrl:     "",
				Path:        "/api/v1/webhooks",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusOK,
				body:       `{"success": true}`,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t, tt.serverResponse)
			defer server.Close()

			tt.consumer.BaseUrl = server.URL

			notification := NewNotification()

			ctx := context.Background()
			err := notification.Notify(ctx, tt.data, tt.headers, tt.consumer)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Notify() expected error but got none")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Notify() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Notify() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestNotification_NotifyTrigger_InvalidData(t *testing.T) {
	notification := NewNotification()
	ctx := context.Background()

	invalidData := map[string]any{
		"channel": make(chan int),
	}

	trigger := wtrhandler.Consumer{
		ServiceName: "test-service",
		BaseUrl:     "http://example.com",
		Path:        "/webhook",
	}

	err := notification.Notify(ctx, invalidData, nil, trigger)
	if err == nil {
		t.Error("Notify() expected error for invalid data but got none")
		return
	}

	if !containsString(err.Error(), "marshal data") {
		t.Errorf("Notify() error = %v, want error containing 'marshal data'", err)
	}
}

func TestNotification_NotifyConsumer(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		data           map[string]any
		headers        map[string]string
		serverResponse serverResponse
		wantErr        bool
		errContains    string
	}{
		{
			name: "successful consumer notification",
			url:  "",
			data: map[string]any{
				"consumer_id": "consumer123",
				"message":     "consumer notification",
			},
			headers: map[string]string{
				"X-Consumer": "test-consumer",
			},
			serverResponse: serverResponse{
				statusCode: http.StatusOK,
				body:       `{"received": true}`,
			},
			wantErr: false,
		},
		{
			name: "failed consumer notification",
			url:  "",
			data: map[string]any{
				"consumer_id": "consumer456",
			},
			headers: map[string]string{},
			serverResponse: serverResponse{
				statusCode: http.StatusBadRequest,
				body:       `{"error": "bad request"}`,
			},
			wantErr:     true,
			errContains: "unexpected status code: 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t, tt.serverResponse)
			defer server.Close()

			url := server.URL + "/consumer"

			notification := NewNotification()

			ctx := context.Background()
			err := notification.NotifyConsumer(ctx, url, tt.data, tt.headers)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NotifyConsumer() expected error but got none")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("NotifyConsumer() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NotifyConsumer() unexpected error = %v", err)
				}
			}
		})
	}
}

// Helper types and functions

type serverResponse struct {
	statusCode int
	body       string
}

func createTestServer(t *testing.T, response serverResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}

		var requestData map[string]any
		if err := json.Unmarshal(body, &requestData); err != nil {
			t.Errorf("Request body is not valid JSON: %v", err)
		}

		w.WriteHeader(response.statusCode)
		if response.body != "" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, response.body)
		}
	}))
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(substr) > 0 && len(s) > 0 &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}
