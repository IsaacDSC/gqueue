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

	mockPublisher := pubadapter.NewMockGenericPublisher(ctrl)

	tests := []struct {
		name           string
		payload        InternalPayload
		setupMock      func(*pubadapter.MockGenericPublisher)
		setupStoreMock func(*MockStore)
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
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "user.created").Return(domain.Event{
					Name: "user.created",
					Type: domain.TriggerTypeInternal,
					Triggers: []domain.Trigger{{
						ServiceName: "test-service",
						Host:        "http://localhost:8080",
						Path:        "/webhook",
					}},
				}, nil).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"your-project-id-event-queue-request-to-external",
						gomock.AssignableToTypeOf(RequestPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload RequestPayload, opts pubadapter.Opts) {
						// Validate payload content using assert
						assert.Equal(t, "user.created", payload.EventName)
						assert.Equal(t, "123", payload.Data["user_id"])
						assert.Equal(t, "test@example.com", payload.Data["email"])
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
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "order.completed").Return(domain.Event{
					Name: "order.completed",
					Type: domain.TriggerTypeInternal,
					Triggers: []domain.Trigger{{
						ServiceName: "test-service",
						Host:        "http://localhost:8080",
						Path:        "/webhook",
					}},
				}, nil).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"your-project-id-event-queue-request-to-external",
						gomock.AssignableToTypeOf(RequestPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload RequestPayload, opts pubadapter.Opts) {
						// Validate payload content using assert
						assert.Equal(t, "order.completed", payload.EventName)
						assert.Equal(t, "ord_123456", payload.Data["order_id"])
						assert.Equal(t, 99.99, payload.Data["amount"])
						// Validate headers
						assert.Equal(t, "Bearer token123", payload.Headers["Authorization"])
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
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "notification.send").Return(domain.Event{
					Name: "notification.send",
					Type: domain.TriggerTypeInternal,
					Triggers: []domain.Trigger{{
						ServiceName: "test-service",
						Host:        "http://localhost:8080",
						Path:        "/webhook",
					}},
				}, nil).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"your-project-id-event-queue-request-to-external",
						gomock.AssignableToTypeOf(RequestPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload RequestPayload, opts pubadapter.Opts) {
						// Validate payload content using assert
						assert.Equal(t, "notification.send", payload.EventName)
						assert.Equal(t, "user_456", payload.Data["user_id"])
						assert.Equal(t, "Welcome to our platform!", payload.Data["message"])
						assert.Equal(t, "email", payload.Data["channel"])
						// Validate headers
						assert.Equal(t, "notification-worker", payload.Headers["X-Service"])
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
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "payment.failed").Return(domain.Event{
					Name: "payment.failed",
					Type: domain.TriggerTypeInternal,
					Triggers: []domain.Trigger{{
						ServiceName: "test-service",
						Host:        "http://localhost:8080",
						Path:        "/webhook",
					}},
				}, nil).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"your-project-id-event-queue-request-to-external",
						gomock.AssignableToTypeOf(RequestPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload RequestPayload, opts pubadapter.Opts) {
						// Validate payload even on error scenario using assert
						assert.Equal(t, "payment.failed", payload.EventName)
						assert.Equal(t, "pay_error123", payload.Data["payment_id"])
						assert.Equal(t, "insufficient_funds", payload.Data["reason"])
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
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "system.ping").Return(domain.Event{
					Name: "system.ping",
					Type: domain.TriggerTypeInternal,
					Triggers: []domain.Trigger{{
						ServiceName: "test-service",
						Host:        "http://localhost:8080",
						Path:        "/webhook",
					}},
				}, nil).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"your-project-id-event-queue-request-to-external",
						gomock.AssignableToTypeOf(RequestPayload{}),
						gomock.Any(),
					).
					Do(func(ctx context.Context, eventName string, payload RequestPayload, opts pubadapter.Opts) {
						// Validate empty/default payload using assert
						assert.Equal(t, "system.ping", payload.EventName)
						assert.Empty(t, payload.Data)
					}).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name: "get_event_returns_event_not_found",
			payload: InternalPayload{
				EventName: "nonexistent.event",
				Data: Data{
					"key": "value",
				},
				Metadata: Metadata{
					Source:      "test-service",
					Version:     "1.0",
					Environment: "test",
				},
				Opts: domain.Opt{
					WqType: pubadapter.LowLatency,
				},
			},
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "nonexistent.event").Return(domain.Event{}, domain.EventNotFound).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				// No publish should be called when event is not found
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name: "get_event_returns_other_error",
			payload: InternalPayload{
				EventName: "error.event",
				Data: Data{
					"key": "value",
				},
				Metadata: Metadata{
					Source:      "test-service",
					Version:     "1.0",
					Environment: "test",
				},
				Opts: domain.Opt{
					WqType: pubadapter.LowLatency,
				},
			},
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "error.event").Return(domain.Event{}, errors.New("database connection error")).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				// No publish should be called when store returns error
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "event_with_zero_triggers",
			payload: InternalPayload{
				EventName: "no.triggers.event",
				Data: Data{
					"key": "value",
				},
				Metadata: Metadata{
					Source:      "test-service",
					Version:     "1.0",
					Environment: "test",
				},
				Opts: domain.Opt{
					WqType: pubadapter.LowLatency,
				},
			},
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "no.triggers.event").Return(domain.Event{
					Name:     "no.triggers.event",
					Type:     domain.TriggerTypeInternal,
					Triggers: []domain.Trigger{}, // Empty triggers
				}, nil).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				// No publish should be called when there are no triggers
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name: "event_with_multiple_triggers",
			payload: InternalPayload{
				EventName: "multi.trigger.event",
				Data: Data{
					"user_id": "456",
				},
				Metadata: Metadata{
					Source:      "multi-service",
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
			setupStoreMock: func(s *MockStore) {
				s.EXPECT().GetEvent(gomock.Any(), "multi.trigger.event").Return(domain.Event{
					Name: "multi.trigger.event",
					Type: domain.TriggerTypeInternal,
					Triggers: []domain.Trigger{
						{
							ServiceName: "service-one",
							Host:        "http://localhost:8081",
							Path:        "/webhook1",
						},
						{
							ServiceName: "service-two",
							Host:        "http://localhost:8082",
							Path:        "/webhook2",
						},
						{
							ServiceName: "service-three",
							Host:        "http://localhost:8083",
							Path:        "/webhook3",
						},
					},
				}, nil).Times(1)
			},
			setupMock: func(m *pubadapter.MockGenericPublisher) {
				m.EXPECT().
					Publish(
						gomock.Any(),
						"your-project-id-event-queue-request-to-external",
						gomock.AssignableToTypeOf(RequestPayload{}),
						gomock.Any(),
					).
					Return(nil).
					Times(3) // Should be called once for each trigger
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.setupMock(mockPublisher)

			// Create mocks for Store and PublisherInsights using generated mocks
			store := NewMockStore(ctrl)
			tt.setupStoreMock(store)

			insights := NewMockPublisherInsights(ctrl)
			insights.EXPECT().Published(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			// Create handler
			httpHandle := PublisherEvent(store, mockPublisher, insights)

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
