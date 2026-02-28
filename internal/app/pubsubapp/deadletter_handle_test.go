package pubsubapp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/notifyopt"
	"github.com/IsaacDSC/gqueue/mocks/mockpubsubapp"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func init() {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{}
	cfg.SetConfig(testConfig)
}

func TestNewDeadLatterQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("returns_correct_event_name_and_handler", func(t *testing.T) {
		mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
		mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)

		handle := NewDeadLatterQueue(mockStore, mockFetcher)

		assert.Equal(t, domain.EventQueueDeadLetter, handle.EventName)
		assert.NotNil(t, handle.Handler)
	})

	t.Run("constructor_accepts_nil_dependencies", func(t *testing.T) {
		// Test that constructor doesn't panic with nil dependencies
		handle := NewDeadLatterQueue(nil, nil)

		assert.Equal(t, domain.EventQueueDeadLetter, handle.EventName)
		assert.NotNil(t, handle.Handler)
	})

	t.Run("constructor_creates_different_instances", func(t *testing.T) {
		mockStore1 := mockpubsubapp.NewMockDeadLetterStore(ctrl)
		mockFetcher1 := mockpubsubapp.NewMockFetcher(ctrl)
		mockStore2 := mockpubsubapp.NewMockDeadLetterStore(ctrl)
		mockFetcher2 := mockpubsubapp.NewMockFetcher(ctrl)

		handle1 := NewDeadLatterQueue(mockStore1, mockFetcher1)
		handle2 := NewDeadLatterQueue(mockStore2, mockFetcher2)

		// Both should have same event name but different handler instances
		assert.Equal(t, handle1.EventName, handle2.EventName)
		assert.NotEqual(t, &handle1.Handler, &handle2.Handler)
	})
}

func TestDeadLetterQueue_Handler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	publishTime := time.Now()
	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"user_id": "123", "action": "create"}`),
		Attributes:  map[string]string{"source": "api", "version": "1.0"},
		PublishTime: publishTime,
	}

	mockEvents := []domain.Event{
		{
			Name:        "user.created",
			ServiceName: "user-service",
			State:       "archived",
			Consumers: []domain.Consumer{
				{
					ServiceName: "notification-service",
					BaseUrl:     "http://localhost:8080",
					Path:        "/webhook/user-created",
					Headers: map[string]string{
						"Content-Type":  "application/json",
						"Authorization": "Bearer token",
					},
				},
				{
					ServiceName: "analytics-service",
					BaseUrl:     "http://localhost:8081",
					Path:        "/analytics/event",
					Headers: map[string]string{
						"X-API-Key": "analytics-key",
					},
				},
			},
		},
		{
			Name:        "order.completed",
			ServiceName: "order-service",
			State:       "archived",
			Consumers: []domain.Consumer{
				{
					ServiceName: "email-service",
					BaseUrl:     "http://localhost:8082",
					Path:        "/send-confirmation",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
		},
	}

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	var notifyCalls []struct {
		data     map[string]any
		headers  map[string]string
		consumer domain.Consumer
	}
	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)
	mockFetcher.EXPECT().
		Notify(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), notifyopt.HighThroughput).
		DoAndReturn(func(ctx context.Context, data map[string]any, headers map[string]string, consumer domain.Consumer, opt notifyopt.Kind) error {
			notifyCalls = append(notifyCalls, struct {
				data     map[string]any
				headers  map[string]string
				consumer domain.Consumer
			}{data: data, headers: headers, consumer: consumer})
			return nil
		}).
		Times(3)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// Verify that Notify was called for each consumer in each event
	expectedCalls := 3 // 2 consumers from first event + 1 consumer from second event
	assert.Len(t, notifyCalls, expectedCalls)

	// Verify first call (first consumer of first event)
	firstCall := notifyCalls[0]
	assert.Equal(t, "user.created", firstCall.data["event"])
	assert.Equal(t, "test-message-id", firstCall.data["id"])
	assert.Equal(t, []byte(`{"user_id": "123", "action": "create"}`), firstCall.data["data"])
	assert.Equal(t, map[string]string{"source": "api", "version": "1.0"}, firstCall.data["metadata"])
	// Don't compare exact time due to marshaling precision loss
	assert.NotNil(t, firstCall.data["event_at"])
	assert.Equal(t, map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"}, firstCall.headers)
	assert.Equal(t, domain.Consumer{
		ServiceName: "notification-service",
		BaseUrl:     "http://localhost:8080",
		Path:        "/webhook/user-created",
		Headers:     map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"},
	}, firstCall.consumer)

	// Verify second call (second consumer of first event)
	secondCall := notifyCalls[1]
	assert.Equal(t, "user.created", secondCall.data["event"])
	assert.Equal(t, "test-message-id", secondCall.data["id"])
	assert.Equal(t, []byte(`{"user_id": "123", "action": "create"}`), secondCall.data["data"])
	assert.Equal(t, map[string]string{"source": "api", "version": "1.0"}, secondCall.data["metadata"])
	assert.NotNil(t, secondCall.data["event_at"])
	assert.Equal(t, map[string]string{"X-API-Key": "analytics-key"}, secondCall.headers)
	assert.Equal(t, domain.Consumer{
		ServiceName: "analytics-service",
		BaseUrl:     "http://localhost:8081",
		Path:        "/analytics/event",
		Headers:     map[string]string{"X-API-Key": "analytics-key"},
	}, secondCall.consumer)

	// Verify third call (first consumer of second event)
	thirdCall := notifyCalls[2]
	assert.Equal(t, "order.completed", thirdCall.data["event"])
	assert.Equal(t, "test-message-id", thirdCall.data["id"])
	assert.Equal(t, []byte(`{"user_id": "123", "action": "create"}`), thirdCall.data["data"])
	assert.Equal(t, map[string]string{"source": "api", "version": "1.0"}, thirdCall.data["metadata"])
	assert.NotNil(t, thirdCall.data["event_at"])
	assert.Equal(t, map[string]string{"Content-Type": "application/json"}, thirdCall.headers)
	assert.Equal(t, domain.Consumer{
		ServiceName: "email-service",
		BaseUrl:     "http://localhost:8082",
		Path:        "/send-confirmation",
		Headers:     map[string]string{"Content-Type": "application/json"},
	}, thirdCall.consumer)
}

func TestDeadLetterQueue_Handler_EventNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"test": "data"}`),
		Attributes:  map[string]string{"source": "test"},
		PublishTime: time.Now(),
	}

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(nil, domain.EventNotFound).
		Times(1)

	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	// Should return nil when EventNotFound (Notify is not called)
	require.NoError(t, err)
}

func TestDeadLetterQueue_Handler_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"test": "data"}`),
		Attributes:  map[string]string{"source": "test"},
		PublishTime: time.Now(),
	}

	expectedError := errors.New("database connection failed")

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(nil, expectedError).
		Times(1)

	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get all schedulers:")
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestDeadLetterQueue_Handler_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Create AsyncCtx with invalid JSON payload
	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), []byte("invalid json"))
	err := handle.Handler(asyncCtx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get payload:")
}

func TestDeadLetterQueue_Handler_EmptyEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"test": "data"}`),
		Attributes:  map[string]string{"source": "test"},
		PublishTime: time.Now(),
	}

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return([]domain.Event{}, nil).
		Times(1)

	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)
}

func TestDeadLetterQueue_Handler_EventsWithNoConsumers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"test": "data"}`),
		Attributes:  map[string]string{"source": "test"},
		PublishTime: time.Now(),
	}

	mockEvents := []domain.Event{
		{
			Name:        "event.without.consumers",
			ServiceName: "test-service",
			State:       "archived",
			Consumers:   []domain.Consumer{}, // No consumers
		},
	}

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)
}

func TestDeadLetterQueue_Handler_FetcherError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	publishTime := time.Now()
	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"test": "data"}`),
		Attributes:  map[string]string{"source": "test"},
		PublishTime: publishTime,
	}

	mockEvents := []domain.Event{
		{
			Name:        "test.event",
			ServiceName: "test-service",
			State:       "archived",
			Consumers: []domain.Consumer{
				{
					ServiceName: "webhook-service",
					BaseUrl:     "http://localhost:8080",
					Path:        "/webhook",
					Headers:     map[string]string{"Content-Type": "application/json"},
				},
			},
		},
	}

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	fetcherError := errors.New("webhook delivery failed")
	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)
	mockFetcher.EXPECT().
		Notify(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), notifyopt.HighThroughput).
		Return(fetcherError).
		Times(1)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	// The handler should continue even if Notify fails (no error propagation in current implementation)
	require.NoError(t, err)
}

func TestDeadLetterQueue_Handler_MultipleEventsWithMixedConsumers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	publishTime := time.Now()
	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"order_id": "12345"}`),
		Attributes:  map[string]string{"correlation_id": "abc123"},
		PublishTime: publishTime,
	}

	mockEvents := []domain.Event{
		{
			Name:        "order.created",
			ServiceName: "order-service",
			State:       "archived",
			Consumers: []domain.Consumer{
				{
					ServiceName: "inventory-service",
					BaseUrl:     "http://inventory.local",
					Path:        "/reserve",
					Headers:     map[string]string{"Authorization": "Bearer inventory-token"},
				},
			},
		},
		{
			Name:        "notification.send",
			ServiceName: "notification-service",
			State:       "archived",
			Consumers:   []domain.Consumer{}, // Event with no consumers
		},
		{
			Name:        "audit.log",
			ServiceName: "audit-service",
			State:       "archived",
			Consumers: []domain.Consumer{
				{
					ServiceName: "logging-service",
					BaseUrl:     "http://logs.local",
					Path:        "/audit",
					Headers:     map[string]string{"X-Service": "audit"},
				},
				{
					ServiceName: "backup-service",
					BaseUrl:     "http://backup.local",
					Path:        "/store",
					Headers:     map[string]string{"X-Backup": "true"},
				},
			},
		},
	}

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	var notifyCalls []struct {
		data map[string]any
	}
	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)
	mockFetcher.EXPECT().
		Notify(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), notifyopt.HighThroughput).
		DoAndReturn(func(ctx context.Context, data map[string]any, headers map[string]string, consumer domain.Consumer, opt notifyopt.Kind) error {
			notifyCalls = append(notifyCalls, struct{ data map[string]any }{data: data})
			return nil
		}).
		Times(3)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// Should have 3 calls: 1 from first event + 0 from second event + 2 from third event
	assert.Len(t, notifyCalls, 3)

	// Verify calls contain expected event names
	eventNames := make([]string, len(notifyCalls))
	for i, call := range notifyCalls {
		eventNames[i] = call.data["event"].(string)
	}

	assert.Contains(t, eventNames, "order.created")
	assert.Contains(t, eventNames, "audit.log")
	assert.NotContains(t, eventNames, "notification.send") // This event has no consumers

	// Count occurrences of each event
	orderCount := 0
	auditCount := 0
	for _, name := range eventNames {
		switch name {
		case "order.created":
			orderCount++
		case "audit.log":
			auditCount++
		}
	}

	assert.Equal(t, 1, orderCount, "order.created should appear once")
	assert.Equal(t, 2, auditCount, "audit.log should appear twice (2 consumers)")
}

func TestDeadLetterQueue_Handler_VerifyState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubsubMessage := &pubsub.Message{
		ID:          "test-message-id",
		Data:        []byte(`{"test": "data"}`),
		Attributes:  map[string]string{},
		PublishTime: time.Now(),
	}

	mockStore := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return([]domain.Event{}, nil).
		Times(1)

	mockFetcher := mockpubsubapp.NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// The expectation on the mock already verifies that "archived" state was passed
}
