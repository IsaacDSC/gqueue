package wtrhandler

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
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetExternalHandle(t *testing.T) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := pubadapter.NewMockPublisher(ctrl)

	tests := []struct {
		name           string
		payload        InternalPayload
		setupMock      func(*pubadapter.MockPublisher)
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
					WqType:     pubadapter.LowLatency,
				},
			},
			setupMock: func(m *pubadapter.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						gomock.Any(),
						"your-project-id-event-queue-internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, wqtype pubadapter.WQType, eventName string, payload InternalPayload, opts pubadapter.Opts) {
						// Validate payload content using assert
						assert.Equal(t, "user.created", payload.EventName)
						assert.Equal(t, "123", payload.Data["user_id"])
						assert.Equal(t, "test@example.com", payload.Data["email"])
						assert.Equal(t, "api", payload.Metadata.Source)
						assert.Equal(t, 3, payload.Opts.MaxRetries)
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
					WqType:     pubadapter.LowLatency,
				},
			},
			setupMock: func(m *pubadapter.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						gomock.Any(),
						"your-project-id-event-queue-internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, wqtype pubadapter.WQType, eventName string, payload InternalPayload, opts pubadapter.Opts) {
						// Validate payload content using assert
						assert.Equal(t, "order.completed", payload.EventName)
						assert.Equal(t, "ord_123456", payload.Data["order_id"])
						assert.Equal(t, 99.99, payload.Data["amount"])
						assert.Equal(t, "checkout-service", payload.Metadata.Source)
						assert.Equal(t, "production", payload.Metadata.Environment)
						assert.Equal(t, 5, payload.Opts.MaxRetries)
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
					WqType:     pubadapter.LowLatency,
				},
			},
			setupMock: func(m *pubadapter.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						gomock.Any(),
						"your-project-id-event-queue-internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, wqtype pubadapter.WQType, eventName string, payload InternalPayload, opts pubadapter.Opts) {
						// Validate payload content using assert
						assert.Equal(t, "notification.send", payload.EventName)
						assert.Equal(t, "user_456", payload.Data["user_id"])
						assert.Equal(t, "Welcome to our platform!", payload.Data["message"])
						assert.Equal(t, "email", payload.Data["channel"])
						assert.Equal(t, "notification-service", payload.Metadata.Source)
						assert.Equal(t, "1.5", payload.Metadata.Version)
						assert.Equal(t, "staging", payload.Metadata.Environment)
						assert.Equal(t, 2, payload.Opts.MaxRetries)
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
					MaxRetries: 3,
					WqType:     pubadapter.LowLatency,
				},
			},
			setupMock: func(m *pubadapter.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						gomock.Any(),
						"your-project-id-event-queue-internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, wqtype pubadapter.WQType, eventName string, payload InternalPayload, opts pubadapter.Opts) {
						// Validate payload even on error scenario using assert
						assert.Equal(t, "payment.failed", payload.EventName)
						assert.Equal(t, "pay_error123", payload.Data["payment_id"])
						assert.Equal(t, "insufficient_funds", payload.Data["reason"])
						assert.Equal(t, "payment-service", payload.Metadata.Source)
						assert.Equal(t, 3, payload.Opts.MaxRetries)
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
				Opts: domain.Opt{
					WqType: pubadapter.LowLatency,
				},
			},
			setupMock: func(m *pubadapter.MockPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						gomock.Any(),
						"your-project-id-event-queue-internal",
						gomock.AssignableToTypeOf(InternalPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, wqtype pubadapter.WQType, eventName string, payload InternalPayload, opts pubadapter.Opts) {
						// Validate empty/default payload using assert
						assert.Equal(t, "system.ping", payload.EventName)
						assert.Empty(t, payload.Data)
						assert.Equal(t, "health-check", payload.Metadata.Source)
						assert.Equal(t, "1.0", payload.Metadata.Version)
						assert.Equal(t, "test", payload.Metadata.Environment)
						// Validate default config options (should be zero values)
						assert.Equal(t, 0, payload.Opts.MaxRetries)
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
