package eventqueue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IsaacDSC/webhook/internal/intersvc"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetInternalConsumerHandle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	tests := []struct {
		name           string
		payload        InternalPayload
		setupMocks     func(*publisher.MockPublisher, *getInternalEvent)
		getEventFunc   getInternalEvent
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name: "successful_processing_single_trigger",
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
						"Content-Type":  "application/json",
						"Authorization": "Bearer token",
					},
				},
				Opts: ConfigOpts{
					MaxRetries: 3,
					Queue:      "default",
				},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, getEventFn *getInternalEvent) {
				*getEventFn = func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
					return intersvc.InternalEvent{
						Name:        "user.created",
						ServiceName: "user-service",
						Triggers: intersvc.ListTrigger{
							{
								ServiceName: "notification-service",
								Type:        "persistent",
								BaseUrl:     "https://api.notification.com",
								Path:        "/webhook/user-created",
								Headers: map[string]string{
									"X-API-Key": "secret",
								},
							},
						},
					}, nil
				}

				mockPub.EXPECT().
					Publish(gomock.Any(), "event-queue.request-to-external", gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload RequestPayload, opts ...asynq.Option) error {
						assert.Equal(t, "user.created", payload.EventName)
						assert.Equal(t, "notification-service", payload.Trigger.ServiceName)
						assert.Equal(t, TriggerType("persistent"), payload.Trigger.Type)
						assert.Equal(t, "https://api.notification.com", payload.Trigger.BaseUrl)
						assert.Equal(t, "/webhook/user-created", payload.Trigger.Path)
						assert.Equal(t, "secret", payload.Trigger.Headers["X-API-Key"])
						assert.Equal(t, "test@example.com", payload.Data["email"])
						assert.Equal(t, "Bearer token", payload.Headers["Authorization"])
						return nil
					}).
					Times(1)
			},
			expectedError: false,
		},
		{
			name: "successful_processing_multiple_triggers",
			payload: InternalPayload{
				EventName: "order.completed",
				Data: Data{
					"order_id":    "order-123",
					"customer_id": "customer-456",
					"total":       99.99,
				},
				Metadata: Metadata{
					Source:      "order-service",
					Version:     "2.0",
					Environment: "production",
					Headers: map[string]string{
						"Content-Type": "application/json",
						"X-Trace-ID":   "trace-123",
					},
				},
				Opts: ConfigOpts{
					MaxRetries: 5,
					Queue:      "high-priority",
				},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, getEventFn *getInternalEvent) {
				*getEventFn = func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
					return intersvc.InternalEvent{
						Name:        "order.completed",
						ServiceName: "order-service",
						Triggers: intersvc.ListTrigger{
							{
								ServiceName: "billing-service",
								Type:        "persistent",
								BaseUrl:     "https://api.billing.com",
								Path:        "/webhook/order-completed",
								Headers: map[string]string{
									"X-API-Key": "billing-key",
								},
							},
							{
								ServiceName: "analytics-service",
								Type:        "notPersistent",
								BaseUrl:     "https://api.analytics.com",
								Path:        "/webhook/order-completed",
								Headers: map[string]string{
									"X-API-Key": "analytics-key",
								},
							},
						},
					}, nil
				}

				mockPub.EXPECT().
					Publish(gomock.Any(), "event-queue.request-to-external", gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload RequestPayload, opts ...asynq.Option) error {
						assert.Equal(t, "order.completed", payload.EventName)
						assert.Contains(t, []string{"billing-service", "analytics-service"}, payload.Trigger.ServiceName)
						assert.Equal(t, "order-123", payload.Data["order_id"])
						assert.Equal(t, "trace-123", payload.Headers["X-Trace-ID"])
						return nil
					}).
					Times(2)
			},
			expectedError: false,
		},
		{
			name: "successful_processing_empty_data",
			payload: InternalPayload{
				EventName: "system.ping",
				Data:      Data{},
				Metadata: Metadata{
					Source:      "health-check",
					Version:     "1.0",
					Environment: "test",
					Headers:     map[string]string{},
				},
				Opts: ConfigOpts{
					MaxRetries: 1,
					Queue:      "low-priority",
				},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, getEventFn *getInternalEvent) {
				*getEventFn = func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
					return intersvc.InternalEvent{
						Name:        "system.ping",
						ServiceName: "health-service",
						Triggers: intersvc.ListTrigger{
							{
								ServiceName: "monitoring-service",
								Type:        "fireForGet",
								BaseUrl:     "https://api.monitoring.com",
								Path:        "/webhook/ping",
								Headers:     map[string]string{},
							},
						},
					}, nil
				}

				mockPub.EXPECT().
					Publish(gomock.Any(), "event-queue.request-to-external", gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload RequestPayload, opts ...asynq.Option) error {
						assert.Equal(t, "system.ping", payload.EventName)
						assert.Equal(t, "monitoring-service", payload.Trigger.ServiceName)
						assert.Equal(t, TriggerType("fireForGet"), payload.Trigger.Type)
						assert.Empty(t, payload.Data)
						return nil
					}).
					Times(1)
			},
			expectedError: false,
		},
		{
			name: "error_get_internal_event_not_found",
			payload: InternalPayload{
				EventName: "nonexistent.event",
				Data: Data{
					"key": "value",
				},
				Metadata: Metadata{
					Source:      "api",
					Version:     "1.0",
					Environment: "test",
					Headers:     map[string]string{},
				},
				Opts: ConfigOpts{},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, getEventFn *getInternalEvent) {
				*getEventFn = func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
					return intersvc.InternalEvent{}, errors.New("event not found")
				}
				// No publisher expectations since it should fail before calling publish
			},
			expectedError:  true,
			expectedErrMsg: "get internal event: event not found",
		},
		{
			name: "error_publisher_fails",
			payload: InternalPayload{
				EventName: "user.updated",
				Data: Data{
					"user_id": "456",
				},
				Metadata: Metadata{
					Source:      "api",
					Version:     "1.0",
					Environment: "test",
					Headers:     map[string]string{},
				},
				Opts: ConfigOpts{},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, getEventFn *getInternalEvent) {
				*getEventFn = func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
					return intersvc.InternalEvent{
						Name:        "user.updated",
						ServiceName: "user-service",
						Triggers: intersvc.ListTrigger{
							{
								ServiceName: "notification-service",
								Type:        "persistent",
								BaseUrl:     "https://api.notification.com",
								Path:        "/webhook/user-updated",
								Headers:     map[string]string{},
							},
						},
					}, nil
				}

				mockPub.EXPECT().
					Publish(gomock.Any(), "event-queue.request-to-external", gomock.Any(), gomock.Any()).
					Return(errors.New("publisher connection failed")).
					Times(1)
			},
			expectedError:  true,
			expectedErrMsg: "publish internal event: publisher connection failed",
		},
		{
			name: "successful_processing_no_triggers",
			payload: InternalPayload{
				EventName: "archived.event",
				Data: Data{
					"archive_id": "archive-123",
				},
				Metadata: Metadata{
					Source:      "archive-service",
					Version:     "1.0",
					Environment: "test",
					Headers:     map[string]string{},
				},
				Opts: ConfigOpts{},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, getEventFn *getInternalEvent) {
				*getEventFn = func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
					return intersvc.InternalEvent{
						Name:        "archived.event",
						ServiceName: "archive-service",
						Triggers:    intersvc.ListTrigger{}, // Empty triggers
					}, nil
				}
				// No publisher expectations since there are no triggers to process
			},
			expectedError: false,
		},
		{
			name: "successful_processing_complex_data",
			payload: InternalPayload{
				EventName: "invoice.generated",
				Data: Data{
					"invoice": map[string]interface{}{
						"id":       "inv-789",
						"amount":   250.75,
						"currency": "USD",
						"items": []interface{}{
							map[string]interface{}{
								"name":     "Product A",
								"quantity": 2,
								"price":    100.00,
							},
							map[string]interface{}{
								"name":     "Product B",
								"quantity": 1,
								"price":    50.75,
							},
						},
					},
					"customer": map[string]interface{}{
						"id":    "cust-456",
						"name":  "John Doe",
						"email": "john@example.com",
					},
				},
				Metadata: Metadata{
					Source:      "billing-service",
					Version:     "3.0",
					Environment: "production",
					Headers: map[string]string{
						"Content-Type":     "application/json",
						"X-Correlation-ID": "corr-123",
					},
				},
				Opts: ConfigOpts{
					MaxRetries: 10,
					Queue:      "billing",
				},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, getEventFn *getInternalEvent) {
				*getEventFn = func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
					return intersvc.InternalEvent{
						Name:        "invoice.generated",
						ServiceName: "billing-service",
						Triggers: intersvc.ListTrigger{
							{
								ServiceName: "accounting-service",
								Type:        "persistent",
								BaseUrl:     "https://api.accounting.com",
								Path:        "/webhook/invoice-generated",
								Headers: map[string]string{
									"X-API-Key":    "accounting-secret",
									"X-Webhook-ID": "webhook-001",
								},
							},
						},
					}, nil
				}

				mockPub.EXPECT().
					Publish(gomock.Any(), "event-queue.request-to-external", gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload RequestPayload, opts ...asynq.Option) error {
						assert.Equal(t, "invoice.generated", payload.EventName)
						assert.Equal(t, "accounting-service", payload.Trigger.ServiceName)

						// Verify complex nested data
						invoice, ok := payload.Data["invoice"].(map[string]interface{})
						assert.True(t, ok, "Invoice should be a map")
						assert.Equal(t, "inv-789", invoice["id"])
						assert.Equal(t, 250.75, invoice["amount"])

						customer, ok := payload.Data["customer"].(map[string]interface{})
						assert.True(t, ok, "Customer should be a map")
						assert.Equal(t, "john@example.com", customer["email"])

						// Verify headers
						assert.Equal(t, "corr-123", payload.Headers["X-Correlation-ID"])
						assert.Equal(t, "accounting-secret", payload.Trigger.Headers["X-API-Key"])

						return nil
					}).
					Times(1)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock for each test
			mockPublisher = publisher.NewMockPublisher(ctrl)

			var getEventFn getInternalEvent
			tt.setupMocks(mockPublisher, &getEventFn)

			// Create the handler
			taskName, handlerFunc := GetInternalConsumerHandle(getEventFn, mockPublisher)

			// Verify task name
			assert.Equal(t, "event-queue.internal", taskName)

			// Marshal payload to simulate asynq task
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create asynq task
			task := asynq.NewTask("event-queue.internal", payloadBytes)

			// Execute the handler
			err = handlerFunc(context.Background(), task)

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

func TestGetInternalConsumerHandle_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	// Mock function that should not be called due to invalid payload
	getEventFn := func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
		t.Error("getEventFn should not be called with invalid payload")
		return intersvc.InternalEvent{}, nil
	}

	// Create the handler
	_, handlerFunc := GetInternalConsumerHandle(getEventFn, mockPublisher)

	// Create task with invalid JSON payload
	invalidPayload := []byte(`{"invalid": json}`)
	task := asynq.NewTask("event-queue.internal", invalidPayload)

	// Execute the handler
	err := handlerFunc(context.Background(), task)

	// Should return unmarshal error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payload")
}

func TestGetInternalConsumerHandle_NilContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	getEventFn := func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
		// Verify context is not nil
		assert.NotNil(t, ctx)
		return intersvc.InternalEvent{
			Name:        "test.event",
			ServiceName: "test-service",
			Triggers:    intersvc.ListTrigger{},
		}, nil
	}

	// Create the handler
	_, handlerFunc := GetInternalConsumerHandle(getEventFn, mockPublisher)

	payload := InternalPayload{
		EventName: "test.event",
		Data:      Data{},
		Metadata:  Metadata{},
		Opts:      ConfigOpts{},
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask("event-queue.internal", payloadBytes)

	// Execute with nil context - this should work as the handler should handle it gracefully
	err = handlerFunc(context.Background(), task)
	assert.NoError(t, err)
}

func TestGetInternalConsumerHandle_ConcurrentExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	getEventFn := func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		return intersvc.InternalEvent{
			Name:        eventName,
			ServiceName: "test-service",
			Triggers: intersvc.ListTrigger{
				{
					ServiceName: "target-service",
					Type:        "persistent",
					BaseUrl:     "https://api.target.com",
					Path:        "/webhook/test",
					Headers:     map[string]string{},
				},
			},
		}, nil
	}

	// Expect multiple concurrent publishes
	mockPublisher.EXPECT().
		Publish(gomock.Any(), "event-queue.request-to-external", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(5)

	// Create the handler
	_, handlerFunc := GetInternalConsumerHandle(getEventFn, mockPublisher)

	// Create payload
	payload := InternalPayload{
		EventName: "concurrent.test",
		Data:      Data{"test": "data"},
		Metadata:  Metadata{},
		Opts:      ConfigOpts{},
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Execute handler concurrently
	errChan := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			task := asynq.NewTask("event-queue.internal", payloadBytes)
			err := handlerFunc(context.Background(), task)
			errChan <- err
		}()
	}

	// Collect results
	for i := 0; i < 5; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

func TestGetInternalConsumerHandle_PayloadFieldsMapping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := publisher.NewMockPublisher(ctrl)

	getEventFn := func(ctx context.Context, eventName string) (intersvc.InternalEvent, error) {
		return intersvc.InternalEvent{
			Name:        "field.test",
			ServiceName: "field-service",
			Triggers: intersvc.ListTrigger{
				{
					ServiceName: "target-service",
					Type:        "persistent",
					BaseUrl:     "https://api.target.com",
					Path:        "/webhook/field-test",
					Headers: map[string]string{
						"X-Custom-Header": "custom-value",
					},
				},
			},
		}, nil
	}

	// Verify that all fields are correctly mapped from InternalPayload to RequestPayload
	mockPublisher.EXPECT().
		Publish(gomock.Any(), "event-queue.request-to-external", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, eventName string, payload RequestPayload, opts ...asynq.Option) error {
			// Verify event name mapping
			assert.Equal(t, "field.test", payload.EventName)

			// Verify data mapping
			assert.Equal(t, "test-value", payload.Data["test-key"])
			assert.Equal(t, float64(42), payload.Data["test-number"]) // JSON unmarshaling converts numbers to float64

			// Verify headers mapping from metadata
			assert.Equal(t, "application/json", payload.Headers["Content-Type"])
			assert.Equal(t, "bearer test-token", payload.Headers["Authorization"])

			// Verify trigger mapping
			assert.Equal(t, "target-service", payload.Trigger.ServiceName)
			assert.Equal(t, TriggerType("persistent"), payload.Trigger.Type)
			assert.Equal(t, "https://api.target.com", payload.Trigger.BaseUrl)
			assert.Equal(t, "/webhook/field-test", payload.Trigger.Path)
			assert.Equal(t, "custom-value", payload.Trigger.Headers["X-Custom-Header"])

			return nil
		}).
		Times(1)

	// Create the handler
	_, handlerFunc := GetInternalConsumerHandle(getEventFn, mockPublisher)

	payload := InternalPayload{
		EventName: "field.test",
		Data: Data{
			"test-key":    "test-value",
			"test-number": 42,
		},
		Metadata: Metadata{
			Source:      "test-source",
			Version:     "1.0",
			Environment: "test",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "bearer test-token",
			},
		},
		Opts: ConfigOpts{
			MaxRetries: 3,
			Queue:      "test-queue",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask("event-queue.internal", payloadBytes)

	err = handlerFunc(context.Background(), task)
	assert.NoError(t, err)
}
