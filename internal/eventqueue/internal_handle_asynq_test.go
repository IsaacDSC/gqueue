package eventqueue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetInternalConsumerHandle(t *testing.T) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{
			Queues: cfg.AsynqQueues{
				"internal.default":       10,
				"internal.high-priority": 5,
				"internal.low":           1,
				"external.default":       10,
			},
		},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		payload        InternalPayload
		setupMocks     func(*publisher.MockPublisher, *MockRepository, *cachemanager.MockCache)
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
				Opts: domain.Opt{
					MaxRetries: 3,
					QueueType:  "internal.default",
				},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue.user.created")
				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "user.created").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(5 * time.Minute)
				mockCache.EXPECT().Once(gomock.Any(), key, gomock.Any(), 5*time.Minute, gomock.Any()).
					DoAndReturn(func(ctx context.Context, key cachemanager.Key, dest *[]domain.Event, ttl time.Duration, fetchFunc func(context.Context) (any, error)) error {
						events := []domain.Event{
							{
								Name:        "user.created",
								ServiceName: "user-service",
								Triggers: []domain.Trigger{
									{
										ServiceName: "notification-service",
										Type:        "persistent",
										Host:        "https://api.notification.com",
										Path:        "/webhook/user-created",
										Headers: map[string]string{
											"X-API-Key": "secret",
										},
									},
								},
							},
						}
						*dest = events
						return nil
					})

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
				Opts: domain.Opt{
					MaxRetries: 5,
					QueueType:  "internal.high-priority",
				},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue.order.completed")
				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "order.completed").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(5 * time.Minute)
				mockCache.EXPECT().Once(gomock.Any(), key, gomock.Any(), 5*time.Minute, gomock.Any()).
					DoAndReturn(func(ctx context.Context, key cachemanager.Key, dest *[]domain.Event, ttl time.Duration, fetchFunc func(context.Context) (any, error)) error {
						events := []domain.Event{
							{
								Name:        "order.completed",
								ServiceName: "order-service",
								Triggers: []domain.Trigger{
									{
										ServiceName: "billing-service",
										Type:        "persistent",
										Host:        "https://api.billing.com",
										Path:        "/webhook/order-completed",
										Headers: map[string]string{
											"X-API-Key": "billing-key",
										},
									},
									{
										ServiceName: "analytics-service",
										Type:        "notPersistent",
										Host:        "https://api.analytics.com",
										Path:        "/webhook/order-completed",
										Headers: map[string]string{
											"X-API-Key": "analytics-key",
										},
									},
								},
							},
						}
						*dest = events
						return nil
					})

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
			name: "error_event_not_found",
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
				Opts: domain.Opt{},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue.nonexistent.event")
				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "nonexistent.event").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(5 * time.Minute)
				mockCache.EXPECT().Once(gomock.Any(), key, gomock.Any(), 5*time.Minute, gomock.Any()).
					Return(domain.EventNotFound)
				// No publisher expectations since event is not found
			},
			expectedError: false, // EventNotFound should be handled gracefully
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
				Opts: domain.Opt{},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue.user.updated")
				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "user.updated").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(5 * time.Minute)
				mockCache.EXPECT().Once(gomock.Any(), key, gomock.Any(), 5*time.Minute, gomock.Any()).
					DoAndReturn(func(ctx context.Context, key cachemanager.Key, dest *[]domain.Event, ttl time.Duration, fetchFunc func(context.Context) (any, error)) error {
						events := []domain.Event{
							{
								Name:        "user.updated",
								ServiceName: "user-service",
								Triggers: []domain.Trigger{
									{
										ServiceName: "notification-service",
										Type:        "persistent",
										Host:        "https://api.notification.com",
										Path:        "/webhook/user-updated",
										Headers:     map[string]string{},
									},
								},
							},
						}
						*dest = events
						return nil
					})

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
				Opts: domain.Opt{},
			},
			setupMocks: func(mockPub *publisher.MockPublisher, mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue.archived.event")
				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "archived.event").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(5 * time.Minute)
				mockCache.EXPECT().Once(gomock.Any(), key, gomock.Any(), 5*time.Minute, gomock.Any()).
					DoAndReturn(func(ctx context.Context, key cachemanager.Key, dest *[]domain.Event, ttl time.Duration, fetchFunc func(context.Context) (any, error)) error {
						events := []domain.Event{
							{
								Name:        "archived.event",
								ServiceName: "archive-service",
								Triggers:    []domain.Trigger{}, // Empty triggers
							},
						}
						*dest = events
						return nil
					})
				// No publisher expectations since there are no triggers to process
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh mocks for each test
			mockPublisher := publisher.NewMockPublisher(ctrl)
			mockRepository := NewMockRepository(ctrl)
			mockCache := cachemanager.NewMockCache(ctrl)

			// Setup mocks
			tt.setupMocks(mockPublisher, mockRepository, mockCache)

			// Create the handler
			handle := GetInternalConsumerHandle(mockRepository, mockCache, mockPublisher)

			// Verify event name
			assert.Equal(t, "event-queue.internal", handle.Event)

			// Marshal payload to simulate asynq task
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create asynq task
			task := asynq.NewTask("event-queue.internal", payloadBytes)

			// Execute the handler
			err = handle.Handler(context.Background(), task)

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
	mockRepository := NewMockRepository(ctrl)
	mockCache := cachemanager.NewMockCache(ctrl)

	// Create the handler
	handle := GetInternalConsumerHandle(mockRepository, mockCache, mockPublisher)

	// Create task with invalid JSON payload
	invalidPayload := []byte(`{"invalid": "json"`)
	task := asynq.NewTask("event-queue.internal", invalidPayload)

	// Execute the handler
	err := handle.Handler(context.Background(), task)

	// Should return unmarshal error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payload")
}

func TestGetInternalConsumerHandle_CacheError(t *testing.T) {
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
	mockRepository := NewMockRepository(ctrl)
	mockCache := cachemanager.NewMockCache(ctrl)

	payload := InternalPayload{
		EventName: "test.event",
		Data:      Data{"test": "data"},
		Metadata:  Metadata{Headers: map[string]string{}},
		Opts:      domain.Opt{QueueType: "internal.default"},
	}

	key := cachemanager.Key("event-queue.test.event")
	mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "test.event").Return(key)
	mockCache.EXPECT().GetDefaultTTL().Return(5 * time.Minute)
	mockCache.EXPECT().Once(gomock.Any(), key, gomock.Any(), 5*time.Minute, gomock.Any()).
		Return(errors.New("cache error"))

	// Create the handler
	handle := GetInternalConsumerHandle(mockRepository, mockCache, mockPublisher)

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask("event-queue.internal", payloadBytes)

	err = handle.Handler(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get internal event: cache error")
}
