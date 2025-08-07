package eventqueue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestPayload_mergeHeaders(t *testing.T) {
	tests := []struct {
		name           string
		payload        RequestPayload
		inputHeaders   map[string]string
		expectedResult map[string]string
	}{
		{
			name: "merge_with_empty_payload_headers",
			payload: RequestPayload{
				Headers: nil,
			},
			inputHeaders: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
			expectedResult: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
		},
		{
			name: "merge_with_existing_payload_headers",
			payload: RequestPayload{
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
					"User-Agent":      "webhook-client",
				},
			},
			inputHeaders: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
			expectedResult: map[string]string{
				"X-Custom-Header": "custom-value",
				"User-Agent":      "webhook-client",
				"Authorization":   "Bearer token",
				"Content-Type":    "application/json",
			},
		},
		{
			name: "override_existing_headers",
			payload: RequestPayload{
				Headers: map[string]string{
					"Content-Type": "text/plain",
					"User-Agent":   "webhook-client",
				},
			},
			inputHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
			expectedResult: map[string]string{
				"User-Agent":    "webhook-client",
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
		},
		{
			name: "empty_input_headers",
			payload: RequestPayload{
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
			inputHeaders: map[string]string{},
			expectedResult: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payload.mergeHeaders(tt.inputHeaders)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetRequestHandle(t *testing.T) {
	t.Run("returns_correct_queue_name_and_handler", func(t *testing.T) {
		queueName, handler := GetRequestHandle()

		assert.Equal(t, "event-queue.request-to-external", queueName)
		assert.NotNil(t, handler)
	})
}

func TestGetRequestHandle_Handler(t *testing.T) {
	// Create a test server to mock external webhook endpoint
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, http.MethodPost, r.Method)

		// Verify content type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify custom headers
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		assert.Equal(t, "webhook-client", r.Header.Get("User-Agent"))

		// Read and verify body
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "123", body["user_id"])
		assert.Equal(t, "test@example.com", body["email"])

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	tests := []struct {
		name           string
		payload        RequestPayload
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name: "successful_request",
			payload: RequestPayload{
				EventName: "user.created",
				Trigger: Trigger{
					ServiceName: "user-service",
					Type:        TriggerTypePersistent,
					BaseUrl:     testServer.URL,
					Path:        "/webhook",
					Headers: map[string]string{
						"User-Agent": "webhook-client",
					},
				},
				Data: map[string]any{
					"user_id": "123",
					"email":   "test@example.com",
				},
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the handler
			_, handler := GetRequestHandle()

			// Create task payload
			taskPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create asynq task
			task := asynq.NewTask("test-queue", taskPayload)

			// Execute handler
			ctx := context.Background()
			err = handler(ctx, task)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetRequestHandle_Handler_InvalidPayload(t *testing.T) {
	_, handler := GetRequestHandle()

	// Create task with invalid JSON payload
	task := asynq.NewTask("test-queue", []byte("invalid json"))

	ctx := context.Background()
	err := handler(ctx, task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payload:")
}

func TestFetch(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]any
		headers        map[string]string
		trigger        Trigger
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name: "successful_fetch",
			data: map[string]any{
				"user_id": "123",
				"email":   "test@example.com",
			},
			headers: map[string]string{
				"Authorization": "Bearer token",
				"User-Agent":    "webhook-client",
			},
			trigger: Trigger{
				ServiceName: "user-service",
				Type:        TriggerTypePersistent,
				BaseUrl:     "", // Will be set to test server URL
				Path:        "/webhook",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedError: false,
		},
		{
			name: "server_returns_error_status",
			data: map[string]any{
				"user_id": "123",
			},
			headers: map[string]string{},
			trigger: Trigger{
				ServiceName: "user-service",
				Type:        TriggerTypePersistent,
				BaseUrl:     "", // Will be set to test server URL
				Path:        "/webhook",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError:  true,
			expectedErrMsg: "unexpected status code: 500",
		},
		{
			name: "invalid_data_for_json_marshal",
			data: map[string]any{
				"invalid_channel": make(chan int), // This will cause JSON marshal to fail
			},
			headers: map[string]string{},
			trigger: Trigger{
				ServiceName: "user-service",
				Type:        TriggerTypePersistent,
				BaseUrl:     "http://example.com",
				Path:        "/webhook",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedError:  true,
			expectedErrMsg: "marshal data:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testServer *httptest.Server

			// Only create server if we don't expect a marshal error
			if !contains(tt.expectedErrMsg, "marshal data:") {
				testServer = httptest.NewServer(http.HandlerFunc(tt.serverResponse))
				defer testServer.Close()
				tt.trigger.BaseUrl = testServer.URL
			}

			ctx := context.Background()
			err := fetch(ctx, tt.data, tt.headers, tt.trigger)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFetch_InvalidURL(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{"test": "value"}
	headers := map[string]string{}

	// Create trigger with invalid URL scheme
	trigger := Trigger{
		ServiceName: "test-service",
		Type:        TriggerTypePersistent,
		BaseUrl:     "://invalid-url",
		Path:        "/webhook",
	}

	err := fetch(ctx, data, headers, trigger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create request:")
}

func TestFetch_NetworkError(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{"test": "value"}
	headers := map[string]string{}

	// Create trigger with unreachable URL
	trigger := Trigger{
		ServiceName: "test-service",
		Type:        TriggerTypePersistent,
		BaseUrl:     "http://localhost:99999", // Unreachable port
		Path:        "/webhook",
	}

	err := fetch(ctx, data, headers, trigger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "post request:")
}

func TestFetch_VerifyRequestHeaders(t *testing.T) {
	// Track the request received by the server
	var receivedRequest *http.Request

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	ctx := context.Background()
	data := map[string]any{
		"user_id": "123",
		"email":   "test@example.com",
	}
	headers := map[string]string{
		"Authorization": "Bearer token",
		"User-Agent":    "webhook-client",
		"X-Custom":      "custom-value",
	}

	trigger := Trigger{
		ServiceName: "user-service",
		Type:        TriggerTypePersistent,
		BaseUrl:     testServer.URL,
		Path:        "/webhook",
	}

	err := fetch(ctx, data, headers, trigger)
	require.NoError(t, err)

	// Verify headers were set correctly
	assert.Equal(t, "application/json", receivedRequest.Header.Get("Content-Type"))
	assert.Equal(t, "Bearer token", receivedRequest.Header.Get("Authorization"))
	assert.Equal(t, "webhook-client", receivedRequest.Header.Get("User-Agent"))
	assert.Equal(t, "custom-value", receivedRequest.Header.Get("X-Custom"))
}

func TestFetch_VerifyRequestBody(t *testing.T) {
	var receivedBody map[string]interface{}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedBody)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	ctx := context.Background()
	data := map[string]any{
		"user_id":   "123",
		"email":     "test@example.com",
		"metadata":  map[string]interface{}{"source": "api"},
		"count":     42,
		"is_active": true,
	}
	headers := map[string]string{}

	trigger := Trigger{
		ServiceName: "user-service",
		Type:        TriggerTypePersistent,
		BaseUrl:     testServer.URL,
		Path:        "/webhook",
	}

	err := fetch(ctx, data, headers, trigger)
	require.NoError(t, err)

	// Verify body was sent correctly
	assert.Equal(t, "123", receivedBody["user_id"])
	assert.Equal(t, "test@example.com", receivedBody["email"])
	assert.Equal(t, map[string]interface{}{"source": "api"}, receivedBody["metadata"])
	assert.Equal(t, float64(42), receivedBody["count"]) // JSON numbers are float64
	assert.Equal(t, true, receivedBody["is_active"])
}

// Helper function to check if a string contains a substring
func contains(str, substr string) bool {
	return len(substr) > 0 && len(str) >= len(substr) && str[:len(substr)] == substr
}
