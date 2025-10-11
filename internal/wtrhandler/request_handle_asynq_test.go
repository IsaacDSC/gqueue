package wtrhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

// mockFetcher implements the Fetcher interface for testing
type mockFetcher struct {
	notifyTriggerFunc func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error
}

func (m *mockFetcher) NotifyTrigger(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
	if m.notifyTriggerFunc != nil {
		return m.notifyTriggerFunc(ctx, data, headers, trigger)
	}
	return nil
}

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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFetch := &mockFetcher{}
		mockInsights := NewMockConsumerInsights(ctrl)
		handle := GetRequestHandle(mockFetch, mockInsights)

		assert.Equal(t, "event-queue.request-to-external", handle.Event)
		assert.NotNil(t, handle.Handler)
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
		mockFetcher    *mockFetcher
		setupMocks     func(*MockConsumerInsights)
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
			mockFetcher: &mockFetcher{
				notifyTriggerFunc: func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
					return nil
				},
			},
			setupMocks: func(mockInsights *MockConsumerInsights) {
				mockInsights.EXPECT().
					Consumed(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.ConsumerMetric) error {
						assert.Equal(t, "user.created", input.TopicName)
						assert.Equal(t, "user-service", input.ConsumerName)
						assert.True(t, input.ACK)
						assert.NotZero(t, input.TimeStarted)
						assert.NotZero(t, input.TimeEnded)
						return nil
					}).
					Times(1)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInsights := NewMockConsumerInsights(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(mockInsights)
			}

			// Get the handler
			handle := GetRequestHandle(tt.mockFetcher, mockInsights)

			// Create task payload
			taskPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create AsyncCtx wrapper
			asyncCtx := asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), taskPayload)

			// Execute handler
			err = handle.Handler(asyncCtx)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFetch := &mockFetcher{}
	mockInsights := NewMockConsumerInsights(ctrl)
	handle := GetRequestHandle(mockFetch, mockInsights)

	// Create AsyncCtx wrapper with invalid payload
	asyncCtx := asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), []byte("invalid json"))

	err := handle.Handler(asyncCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payload:")
}

func TestRequestPayload_mergeHeaders_Integration(t *testing.T) {
	tests := []struct {
		name                string
		data                map[string]any
		headers             map[string]string
		trigger             Trigger
		mockNotifyTriggerFn func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error
		setupMocks          func(*MockConsumerInsights)
		expectedError       bool
		expectedErrMsg      string
	}{
		{
			name: "successful_integration_test",
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
				BaseUrl:     "http://example.com",
				Path:        "/webhook",
			},
			mockNotifyTriggerFn: func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
				// Verify the merged headers are passed correctly
				assert.Equal(t, "Bearer token", headers["Authorization"])
				assert.Equal(t, "webhook-client", headers["User-Agent"])
				return nil
			},
			setupMocks: func(mockInsights *MockConsumerInsights) {
				mockInsights.EXPECT().
					Consumed(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.ConsumerMetric) error {
						assert.Equal(t, "user.created", input.TopicName)
						assert.Equal(t, "user-service", input.ConsumerName)
						assert.True(t, input.ACK)
						return nil
					}).
					Times(1)
			},
			expectedError: false,
		},
		{
			name: "fetcher_returns_error",
			data: map[string]any{
				"user_id": "123",
			},
			headers: map[string]string{},
			trigger: Trigger{
				ServiceName: "user-service",
				Type:        TriggerTypePersistent,
				BaseUrl:     "http://example.com",
				Path:        "/webhook",
			},
			mockNotifyTriggerFn: func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
				return assert.AnError
			},
			setupMocks: func(mockInsights *MockConsumerInsights) {
				mockInsights.EXPECT().
					Consumed(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.ConsumerMetric) error {
						assert.Equal(t, "user.created", input.TopicName)
						assert.Equal(t, "user-service", input.ConsumerName)
						assert.False(t, input.ACK)
						return nil
					}).
					Times(1)
			},
			expectedError:  true,
			expectedErrMsg: "fetch trigger:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInsights := NewMockConsumerInsights(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(mockInsights)
			}

			mockFetch := &mockFetcher{
				notifyTriggerFunc: tt.mockNotifyTriggerFn,
			}

			handle := GetRequestHandle(mockFetch, mockInsights)

			payload := RequestPayload{
				EventName: "user.created",
				Trigger:   tt.trigger,
				Data:      tt.data,
				Headers:   tt.headers,
			}

			// Create task payload
			taskPayload, err := json.Marshal(payload)
			require.NoError(t, err)

			// Create AsyncCtx wrapper
			asyncCtx := asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), taskPayload)

			// Execute handler
			err = handle.Handler(asyncCtx)

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

func TestMockFetcher_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name           string
		mockError      error
		expectedErrMsg string
	}{
		{
			name:           "network_error_simulation",
			mockError:      assert.AnError,
			expectedErrMsg: "fetch trigger:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockInsights := NewMockConsumerInsights(ctrl)
			mockInsights.EXPECT().
				Consumed(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, input domain.ConsumerMetric) error {
					assert.Equal(t, "user.created", input.TopicName)
					assert.Equal(t, "user-service", input.ConsumerName)
					assert.False(t, input.ACK)
					return nil
				}).
				Times(1)

			mockFetch := &mockFetcher{
				notifyTriggerFunc: func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
					return tt.mockError
				},
			}

			handle := GetRequestHandle(mockFetch, mockInsights)

			payload := RequestPayload{
				EventName: "user.created",
				Trigger: Trigger{
					ServiceName: "user-service",
					Type:        TriggerTypePersistent,
					BaseUrl:     "http://localhost:99999", // Unreachable port
					Path:        "/webhook",
				},
				Data:    map[string]any{"test": "value"},
				Headers: map[string]string{},
			}

			// Create task payload
			taskPayload, err := json.Marshal(payload)
			require.NoError(t, err)

			// Create AsyncCtx wrapper
			asyncCtx := asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), taskPayload)
			err = handle.Handler(asyncCtx)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

func TestGetRequestHandle_HeaderMerging(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Track the headers received by the mock fetcher
	var receivedHeaders map[string]string

	mockInsights := NewMockConsumerInsights(ctrl)
	mockInsights.EXPECT().
		Consumed(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, input domain.ConsumerMetric) error {
			assert.Equal(t, "user.created", input.TopicName)
			assert.Equal(t, "user-service", input.ConsumerName)
			assert.True(t, input.ACK)
			return nil
		}).
		Times(1)

	mockFetch := &mockFetcher{
		notifyTriggerFunc: func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
			receivedHeaders = headers
			return nil
		},
	}

	handle := GetRequestHandle(mockFetch, mockInsights)

	payload := RequestPayload{
		EventName: "user.created",
		Trigger: Trigger{
			ServiceName: "user-service",
			Type:        TriggerTypePersistent,
			BaseUrl:     "http://example.com",
			Path:        "/webhook",
			Headers: map[string]string{
				"X-Service": "webhook-service",
			},
		},
		Data: map[string]any{
			"user_id": "123",
			"email":   "test@example.com",
		},
		Headers: map[string]string{
			"Authorization": "Bearer token",
			"User-Agent":    "webhook-client",
			"X-Custom":      "custom-value",
		},
	}

	// Create task payload
	taskPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	// Create AsyncCtx wrapper
	asyncCtx := asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), taskPayload)
	err = handle.Handler(asyncCtx)
	require.NoError(t, err)

	// Verify headers were merged correctly (Headers should override Trigger.Headers)
	assert.Equal(t, "Bearer token", receivedHeaders["Authorization"])
	assert.Equal(t, "webhook-client", receivedHeaders["User-Agent"])
	assert.Equal(t, "custom-value", receivedHeaders["X-Custom"])
	assert.Equal(t, "webhook-service", receivedHeaders["X-Service"])
}

func TestGetRequestHandle_DataPassing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Track the data received by the mock fetcher
	var receivedData map[string]any

	mockInsights := NewMockConsumerInsights(ctrl)
	mockInsights.EXPECT().
		Consumed(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, input domain.ConsumerMetric) error {
			assert.Equal(t, "user.created", input.TopicName)
			assert.Equal(t, "user-service", input.ConsumerName)
			assert.True(t, input.ACK)
			return nil
		}).
		Times(1)

	mockFetch := &mockFetcher{
		notifyTriggerFunc: func(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
			receivedData = data
			return nil
		},
	}

	handle := GetRequestHandle(mockFetch, mockInsights)

	expectedData := map[string]any{
		"user_id":   "123",
		"email":     "test@example.com",
		"metadata":  map[string]interface{}{"source": "api"},
		"count":     42,
		"is_active": true,
	}

	payload := RequestPayload{
		EventName: "user.created",
		Trigger: Trigger{
			ServiceName: "user-service",
			Type:        TriggerTypePersistent,
			BaseUrl:     "http://example.com",
			Path:        "/webhook",
		},
		Data:    expectedData,
		Headers: map[string]string{},
	}

	// Create task payload
	taskPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	// Create AsyncCtx wrapper
	asyncCtx := asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), taskPayload)
	err = handle.Handler(asyncCtx)
	require.NoError(t, err)

	// Verify data was passed correctly
	assert.Equal(t, "123", receivedData["user_id"])
	assert.Equal(t, "test@example.com", receivedData["email"])
	assert.Equal(t, map[string]interface{}{"source": "api"}, receivedData["metadata"])
	assert.Equal(t, float64(42), receivedData["count"]) // JSON numbers are float64
	assert.Equal(t, true, receivedData["is_active"])
}
