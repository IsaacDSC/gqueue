package wtrhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetInternalConsumerHandle(t *testing.T) {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{},
	}
	cfg.SetConfig(testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		payload        InternalPayload
		setupMocks     func(*pubadapter.MockGenericPublisher, *MockRepository, *MockPublisherInsights)
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
				},
			},
			setupMocks: func(mockPub *pubadapter.MockGenericPublisher, mockRepo *MockRepository, mockInsights *MockPublisherInsights) {
				mockRepo.EXPECT().GetEvent(gomock.Any(), "user.created").Return(domain.Event{
					Name:        "user.created",
					ServiceName: "user-service",
					Type:        "internal",
					Triggers: []domain.Trigger{
						{
							ServiceName: "notification-service",
							Host:        "https://api.notification.com",
							Path:        "/webhook/user-created",
							Headers: map[string]string{
								"X-API-Key": "secret",
							},
						},
					},
				}, nil)

				mockPub.EXPECT().
					Publish(gomock.Any(), "your-project-id-event-queue-request-to-external", gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload RequestPayload, opts pubadapter.Opts) error {
						assert.Equal(t, "user.created", payload.EventName)
						assert.Equal(t, "notification-service", payload.Trigger.ServiceName)
						assert.Equal(t, "https://api.notification.com", payload.Trigger.BaseUrl)
						assert.Equal(t, "/webhook/user-created", payload.Trigger.Path)
						assert.Equal(t, "secret", payload.Trigger.Headers["X-API-Key"])
						assert.Equal(t, "test@example.com", payload.Data["email"])
						assert.Equal(t, "Bearer token", payload.Headers["Authorization"])
						return nil
					}).
					Times(1)

				mockInsights.EXPECT().
					Published(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.PublisherMetric) error {
						assert.Equal(t, "user.created", input.TopicName)
						assert.True(t, input.ACK)
						assert.NotZero(t, input.TimeStarted)
						assert.NotZero(t, input.TimeEnded)
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
					WqType:     pubadapter.LowLatency,
				},
			},
			setupMocks: func(mockPub *pubadapter.MockGenericPublisher, mockRepo *MockRepository, mockInsights *MockPublisherInsights) {
				mockRepo.EXPECT().GetEvent(gomock.Any(), "order.completed").Return(domain.Event{
					Name:        "order.completed",
					ServiceName: "order-service",
					Triggers: []domain.Trigger{
						{
							ServiceName: "billing-service",
							Host:        "https://api.billing.com",
							Path:        "/webhook/order-completed",
							Headers: map[string]string{
								"X-API-Key": "billing-key",
							},
						},
						{
							ServiceName: "analytics-service",
							Host:        "https://api.analytics.com",
							Path:        "/webhook/order-completed",
							Headers: map[string]string{
								"X-API-Key": "analytics-key",
							},
						},
					},
				}, nil)

				mockPub.EXPECT().
					Publish(gomock.Any(), "your-project-id-event-queue-request-to-external", gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload RequestPayload, opts pubadapter.Opts) error {
						assert.Equal(t, "order.completed", payload.EventName)
						assert.Contains(t, []string{"billing-service", "analytics-service"}, payload.Trigger.ServiceName)
						assert.Equal(t, "order-123", payload.Data["order_id"])
						assert.Equal(t, "trace-123", payload.Headers["X-Trace-ID"])
						return nil
					}).
					Times(2)

				mockInsights.EXPECT().
					Published(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.PublisherMetric) error {
						assert.Equal(t, "order.completed", input.TopicName)
						assert.True(t, input.ACK)
						assert.NotZero(t, input.TimeStarted)
						assert.NotZero(t, input.TimeEnded)
						return nil
					}).
					Times(1)
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
			setupMocks: func(mockPub *pubadapter.MockGenericPublisher, mockRepo *MockRepository, mockInsights *MockPublisherInsights) {
				mockRepo.EXPECT().GetEvent(gomock.Any(), "nonexistent.event").Return(domain.Event{}, domain.EventNotFound)

				// Insights is called even when event is not found
				mockInsights.EXPECT().
					Published(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.PublisherMetric) error {
						assert.Equal(t, "nonexistent.event", input.TopicName)
						assert.True(t, input.ACK)
						assert.NotZero(t, input.TimeStarted)
						assert.NotZero(t, input.TimeEnded)
						return nil
					}).
					Times(1)
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
			setupMocks: func(mockPub *pubadapter.MockGenericPublisher, mockRepo *MockRepository, mockInsights *MockPublisherInsights) {
				mockRepo.EXPECT().GetEvent(gomock.Any(), "user.updated").Return(domain.Event{
					Name:        "user.updated",
					ServiceName: "user-service",
					Triggers: []domain.Trigger{
						{
							ServiceName: "notification-service",
							Host:        "https://api.notification.com",
							Path:        "/webhook/user-updated",
							Headers:     map[string]string{},
						},
					},
				}, nil)

				mockPub.EXPECT().
					Publish(gomock.Any(), "your-project-id-event-queue-request-to-external", gomock.Any(), gomock.Any()).
					Return(errors.New("publisher connection failed")).
					Times(1)

				mockInsights.EXPECT().
					Published(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.PublisherMetric) error {
						assert.Equal(t, "user.updated", input.TopicName)
						assert.True(t, input.ACK)
						assert.NotZero(t, input.TimeStarted)
						assert.NotZero(t, input.TimeEnded)
						return nil
					}).
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
			setupMocks: func(mockPub *pubadapter.MockGenericPublisher, mockRepo *MockRepository, mockInsights *MockPublisherInsights) {
				mockRepo.EXPECT().GetEvent(gomock.Any(), "archived.event").Return(domain.Event{
					Name:        "archived.event",
					ServiceName: "archive-service",
					Triggers:    []domain.Trigger{}, // Empty triggers
				}, nil)

				// Insights is called even when there are no triggers to process
				mockInsights.EXPECT().
					Published(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, input domain.PublisherMetric) error {
						assert.Equal(t, "archived.event", input.TopicName)
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
			// Create fresh mocks for each test
			mockPublisher := pubadapter.NewMockGenericPublisher(ctrl)
			mockRepository := NewMockRepository(ctrl)
			mockInsights := NewMockPublisherInsights(ctrl)

			// Setup mocks
			tt.setupMocks(mockPublisher, mockRepository, mockInsights)

			// Create the handler
			handle := GetInternalConsumerHandle(mockRepository, mockPublisher, mockInsights)

			// Verify event name
			assert.Equal(t, "event-queue.internal", handle.EventName)

			// Marshal payload to simulate asynq task
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create AsyncCtx wrapper
			asyncCtx := asyncadapter.NewAsyncCtx[InternalPayload](context.Background(), payloadBytes)

			// Execute the handler
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
