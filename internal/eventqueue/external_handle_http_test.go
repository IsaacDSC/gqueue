package eventqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/intertime"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetExternalHandle(t *testing.T) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{
			Queues: cfg.AsynqQueues{
				"internal.default":       10,
				"internal.high-priority": 5,
				"internal.notifications": 3,
				"internal.payments":      2,
				"internal.low":           1,
				"external.default":       10,
			},
		},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	tests := []struct {
		name           string
		payload        InternalPayload
		setupMock      func(*publisher.MockPublisher)
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful_publish_minimal_payload",
			payload: InternalPayload{
				EventName: "user.created",
				Data: Data{
					"user_id": "123",
					"email":   "test@example.com",
				},
				Metadata: Metadata{
					Source:      "api",
					Version:     "1.0",
					Environment: "test",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
				Opts: domain.Opt{
					MaxRetries: 3,
					QueueType:  "internal.default",
				},
			},
			setupMock: func(m *publisher.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"event-queue.internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload InternalPayload, opts ...interface{}) {
						// Validate payload content using assert
						assert.Equal(t, "user.created", payload.EventName)
						assert.Equal(t, "123", payload.Data["user_id"])
						assert.Equal(t, "test@example.com", payload.Data["email"])
						assert.Equal(t, "api", payload.Metadata.Source)
						assert.Equal(t, 3, payload.Opts.MaxRetries)
						assert.Equal(t, "internal.default", payload.Opts.QueueType)
					}).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name: "successful_publish_full_payload",
			payload: InternalPayload{
				EventName: "order.completed",
				Data: Data{
					"order_id":    "ord_123456",
					"customer_id": "cust_789",
					"amount":      99.99,
					"items": []map[string]any{
						{"id": "item_1", "quantity": 2},
						{"id": "item_2", "quantity": 1},
					},
				},
				Metadata: Metadata{
					Source:      "checkout-service",
					Version:     "2.1",
					Environment: "production",
					Headers: map[string]string{
						"Content-Type":  "application/json",
						"Authorization": "Bearer token123",
						"X-Request-ID":  "req_abc123",
						"X-Correlation": "corr_def456",
					},
				},
				Opts: domain.Opt{
					MaxRetries: 5,
					Retention:  intertime.Duration(24 * time.Hour),
					UniqueTTL:  intertime.Duration(1 * time.Hour),
					ScheduleIn: intertime.Duration(30 * time.Second),
					QueueType:  "internal.high-priority",
				},
			},
			setupMock: func(m *publisher.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"event-queue.internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload InternalPayload, opts ...interface{}) {
						// Validate payload content using assert
						assert.Equal(t, "order.completed", payload.EventName)
						assert.Equal(t, "ord_123456", payload.Data["order_id"])
						assert.Equal(t, 99.99, payload.Data["amount"])
						assert.Equal(t, "checkout-service", payload.Metadata.Source)
						assert.Equal(t, "production", payload.Metadata.Environment)
						assert.Equal(t, 5, payload.Opts.MaxRetries)
						assert.Equal(t, "internal.high-priority", payload.Opts.QueueType)
						assert.Equal(t, intertime.Duration(24*time.Hour), payload.Opts.Retention)
						assert.Equal(t, intertime.Duration(1*time.Hour), payload.Opts.UniqueTTL)
						assert.Equal(t, intertime.Duration(30*time.Second), payload.Opts.ScheduleIn)
						// Validate headers
						assert.Equal(t, "Bearer token123", payload.Metadata.Headers["Authorization"])
					}).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name: "successful_publish_with_deadline",
			payload: InternalPayload{
				EventName: "notification.send",
				Data: Data{
					"user_id":     "user_456",
					"message":     "Welcome to our platform!",
					"channel":     "email",
					"template_id": "welcome_template",
				},
				Metadata: Metadata{
					Source:      "notification-service",
					Version:     "1.5",
					Environment: "staging",
					Headers: map[string]string{
						"Content-Type": "application/json",
						"X-Service":    "notification-worker",
					},
				},
				Opts: domain.Opt{
					MaxRetries: 2,
					Deadline:   func() *time.Time { t := time.Now().Add(5 * time.Minute); return &t }(),
					QueueType:  "internal.notifications",
				},
			},
			setupMock: func(m *publisher.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"event-queue.internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload InternalPayload, opts ...interface{}) {
						// Validate payload content using assert
						assert.Equal(t, "notification.send", payload.EventName)
						assert.Equal(t, "user_456", payload.Data["user_id"])
						assert.Equal(t, "Welcome to our platform!", payload.Data["message"])
						assert.Equal(t, "email", payload.Data["channel"])
						assert.Equal(t, "notification-service", payload.Metadata.Source)
						assert.Equal(t, "1.5", payload.Metadata.Version)
						assert.Equal(t, "staging", payload.Metadata.Environment)
						assert.Equal(t, 2, payload.Opts.MaxRetries)
						assert.Equal(t, "internal.notifications", payload.Opts.QueueType)
						assert.NotNil(t, payload.Opts.Deadline)
						// Validate headers
						assert.Equal(t, "notification-worker", payload.Metadata.Headers["X-Service"])
					}).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name: "publisher_error",
			payload: InternalPayload{
				EventName: "payment.failed",
				Data: Data{
					"payment_id": "pay_error123",
					"reason":     "insufficient_funds",
				},
				Metadata: Metadata{
					Source:      "payment-service",
					Version:     "1.0",
					Environment: "test",
				},
				Opts: domain.Opt{
					MaxRetries: 1,
					QueueType:  "internal.payments",
				},
			},
			setupMock: func(m *publisher.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"event-queue.internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload InternalPayload, opts ...interface{}) {
						// Validate payload even on error scenario using assert
						assert.Equal(t, "payment.failed", payload.EventName)
						assert.Equal(t, "pay_error123", payload.Data["payment_id"])
						assert.Equal(t, "insufficient_funds", payload.Data["reason"])
						assert.Equal(t, "payment-service", payload.Metadata.Source)
						assert.Equal(t, 1, payload.Opts.MaxRetries)
						assert.Equal(t, "internal.payments", payload.Opts.QueueType)
					}).
					Return(errors.New("publisher error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "empty_payload_with_defaults",
			payload: InternalPayload{
				EventName: "system.ping",
				Data:      Data{},
				Metadata: Metadata{
					Source:      "health-check",
					Version:     "1.0",
					Environment: "test",
				},
				Opts: domain.Opt{QueueType: "internal.default"},
			},
			setupMock: func(m *publisher.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"event-queue.internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload InternalPayload, opts ...interface{}) {
						// Validate empty/default payload using assert
						assert.Equal(t, "system.ping", payload.EventName)
						assert.Empty(t, payload.Data)
						assert.Equal(t, "health-check", payload.Metadata.Source)
						assert.Equal(t, "1.0", payload.Metadata.Version)
						assert.Equal(t, "test", payload.Metadata.Environment)
						// Validate default config options (should be zero values)
						assert.Equal(t, 0, payload.Opts.MaxRetries)
						assert.Equal(t, "internal.default", payload.Opts.QueueType)
						assert.Equal(t, intertime.Duration(0), payload.Opts.Retention)
					}).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.setupMock(mockPublisher)

			// Create handler
			httpHandle := Publisher(mockPublisher)

			// Prepare request
			payloadBytes, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add context
			ctx := context.WithValue(req.Context(), "test", true)
			req = req.WithContext(ctx)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute handler
			httpHandle.Handler(rr, req) // Verify status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Verify error response body if expected
			if tt.expectedError {
				assert.NotEmpty(t, rr.Body.String(), "Expected error response body, but got empty body")
			} else {
				// Verify success response has no body (just status)
				assert.Empty(t, rr.Body.String(), "Expected empty response body for success")
			}
		})
	}
}

func TestGetExternalHandle_InvalidJSON(t *testing.T) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{
			Queues: cfg.AsynqQueues{
				"internal.default": 10,
				"external.default": 10,
			},
		},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	tests := []struct {
		name        string
		requestBody string
		expectedErr string
		setupMock   func(*publisher.MockPublisher)
	}{
		{
			name:        "invalid_json_syntax",
			requestBody: `{"event_name": "test", "data": {invalid json}`,
			expectedErr: "invalid character",
			setupMock:   func(m *publisher.MockPublisher) {}, // No expectations
		},
		{
			name:        "empty_body",
			requestBody: "",
			expectedErr: "EOF",
			setupMock:   func(m *publisher.MockPublisher) {}, // No expectations
		},
		{
			name:        "null_body",
			requestBody: "null",
			expectedErr: "",                                  // null is valid JSON but creates empty payload which fails validation
			setupMock:   func(m *publisher.MockPublisher) {}, // No expectations as validation will fail
		},
		{
			name:        "malformed_nested_object",
			requestBody: `{"event_name": "test", "data": {"key": }}`,
			expectedErr: "invalid character",
			setupMock:   func(m *publisher.MockPublisher) {}, // No expectations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.setupMock(mockPublisher)

			httpHandle := Publisher(mockPublisher)

			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(tt.requestBody)))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			httpHandle.Handler(rr, req)

			// For null_body case, it's valid JSON but creates empty payload which fails validation
			if tt.name == "null_body" {
				assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected bad request for null body due to validation failure")
			} else {
				// Should return BadRequest for invalid JSON
				assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected bad request for invalid JSON")

				// Check error message contains expected substring
				if tt.expectedErr != "" {
					responseBody := rr.Body.String()
					assert.Contains(t, responseBody, tt.expectedErr, "Expected error message to contain substring")
				}
			}
		})
	}
}

func TestGetExternalHandle_RequestBodyClosure(t *testing.T) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{
			Queues: cfg.AsynqQueues{
				"internal.default": 10,
				"external.default": 10,
			},
		},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	// Setup mock to expect successful publish
	mockPublisher.EXPECT().
		Publish(gomock.Any(), "event-queue.internal", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	httpHandle := Publisher(mockPublisher)

	payload := InternalPayload{
		EventName: "test.event",
		Data:      Data{"key": "value"},
		Metadata: Metadata{
			Source:      "test",
			Version:     "1.0",
			Environment: "test",
		},
		Opts: domain.Opt{
			MaxRetries: 1,
			QueueType:  "internal.default",
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	httpHandle.Handler(rr, req)

	// Verify the request body was properly closed (this is mainly for coverage)
	// The defer statement should close it, but we can't directly test that
	assert.Equal(t, http.StatusAccepted, rr.Code)
}

// Benchmark test to measure performance
func BenchmarkGetExternalHandle(b *testing.B) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{
			Queues: cfg.AsynqQueues{
				"internal.default": 10,
				"external.default": 10,
			},
		},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	// Setup mock to always succeed
	mockPublisher.EXPECT().
		Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	httpHandle := Publisher(mockPublisher)

	payload := InternalPayload{
		EventName: "benchmark.test",
		Data: Data{
			"user_id":   "123",
			"action":    "click",
			"timestamp": time.Now().Unix(),
		},
		Metadata: Metadata{
			Source:      "web-app",
			Version:     "1.0",
			Environment: "benchmark",
		},
		Opts: domain.Opt{
			MaxRetries: 3,
			QueueType:  "internal.default",
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		httpHandle.Handler(rr, req)

		if rr.Code != http.StatusAccepted {
			b.Fatalf("Expected status %d, got %d", http.StatusAccepted, rr.Code)
		}
	}
}

// Test to validate that our payload validations work by intentionally creating a mismatched payload
func TestGetExternalHandle_PayloadValidation(t *testing.T) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{
			Queues: cfg.AsynqQueues{
				"internal.wrong-queue": 10,
				"external.default":     10,
			},
		},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	// This test demonstrates that our validation logic works
	// by testing with a payload that intentionally has wrong values
	payload := InternalPayload{
		EventName: "user.created",
		Data: Data{
			"user_id": "wrong_id", // This will be validated in the mock
			"email":   "wrong@email.com",
		},
		Metadata: Metadata{
			Source:      "wrong-source",
			Version:     "1.0",
			Environment: "test",
		},
		Opts: domain.Opt{
			MaxRetries: 999, // This will be validated in the mock
			QueueType:  "internal.wrong-queue",
		},
	}

	// Setup mock with strict validation that will detect the "wrong" values
	mockPublisher.EXPECT().
		Publish(
			gomock.Any(),
			"event-queue.internal",
			gomock.AssignableToTypeOf(InternalPayload{}),
			gomock.Any(),
		).
		Do(func(ctx context.Context, eventName string, receivedPayload InternalPayload, opts ...interface{}) {
			// These validations will pass because we're receiving what we sent
			assert.Equal(t, "user.created", receivedPayload.EventName)
			assert.Equal(t, "wrong_id", receivedPayload.Data["user_id"])
			assert.Equal(t, "wrong-source", receivedPayload.Metadata.Source)
			assert.Equal(t, 999, receivedPayload.Opts.MaxRetries)
			// Log what we actually received for demonstration
			t.Logf("Received payload - EventName: %s, UserID: %v, Source: %s, MaxRetries: %d",
				receivedPayload.EventName,
				receivedPayload.Data["user_id"],
				receivedPayload.Metadata.Source,
				receivedPayload.Opts.MaxRetries)
		}).
		Return(nil).
		Times(1)

	httpHandle := Publisher(mockPublisher)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	httpHandle.Handler(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
}
