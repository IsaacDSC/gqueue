package wtrhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func init() {
	// Setup test configuration with valid queues
	testConfig := cfg.Config{
		AsynqConfig: cfg.AsynqConfig{
			Queues: cfg.AsynqQueues{
				"internal.default": 1,
				"external.default": 1,
			},
		},
	}
	cfg.SetConfig(testConfig)
}

// notifyCall struct to track calls made to the mock fetcher
type notifyCall struct {
	data    map[string]any
	headers map[string]string
	trigger Trigger
}

// mockFetcherWithCalls wraps MockFetcher to track calls
type mockFetcherWithCalls struct {
	*MockFetcher
	notifyCalls []notifyCall
}

func (m *mockFetcherWithCalls) NotifyTrigger(ctx context.Context, data map[string]any, headers map[string]string, trigger Trigger) error {
	m.notifyCalls = append(m.notifyCalls, notifyCall{
		data:    data,
		headers: headers,
		trigger: trigger,
	})
	return m.MockFetcher.NotifyTrigger(ctx, data, headers, trigger)
}

func TestNewDeadLatterQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("returns_correct_event_name_and_handler", func(t *testing.T) {
		mockStore := NewMockDeadLetterStore(ctrl)
		mockFetcher := NewMockFetcher(ctrl)

		handle := NewDeadLatterQueue(mockStore, mockFetcher)

		assert.Equal(t, domain.EventQueueDeadLetter, handle.Event)
		assert.NotNil(t, handle.Handler)
	})

	t.Run("constructor_accepts_nil_dependencies", func(t *testing.T) {
		// Test that constructor doesn't panic with nil dependencies
		handle := NewDeadLatterQueue(nil, nil)

		assert.Equal(t, domain.EventQueueDeadLetter, handle.Event)
		assert.NotNil(t, handle.Handler)
	})

	t.Run("constructor_creates_different_instances", func(t *testing.T) {
		mockStore1 := NewMockDeadLetterStore(ctrl)
		mockFetcher1 := NewMockFetcher(ctrl)
		mockStore2 := NewMockDeadLetterStore(ctrl)
		mockFetcher2 := NewMockFetcher(ctrl)

		handle1 := NewDeadLatterQueue(mockStore1, mockFetcher1)
		handle2 := NewDeadLatterQueue(mockStore2, mockFetcher2)

		// Both should have same event name but different handler instances
		assert.Equal(t, handle1.Event, handle2.Event)
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
			Triggers: []domain.Trigger{
				{
					ServiceName: "notification-service",
					Type:        "webhook",
					Host:        "http://localhost:8080",
					Path:        "/webhook/user-created",
					Headers: map[string]string{
						"Content-Type":  "application/json",
						"Authorization": "Bearer token",
					},
				},
				{
					ServiceName: "analytics-service",
					Type:        "webhook",
					Host:        "http://localhost:8081",
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
			Triggers: []domain.Trigger{
				{
					ServiceName: "email-service",
					Type:        "webhook",
					Host:        "http://localhost:8082",
					Path:        "/send-confirmation",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
		},
	}

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	mockFetcher := &mockFetcherWithCalls{
		MockFetcher: NewMockFetcher(ctrl),
		notifyCalls: []notifyCall{},
	}

	// Expect 3 calls to NotifyTrigger (2 triggers from first event + 1 trigger from second event)
	mockFetcher.MockFetcher.EXPECT().
		NotifyTrigger(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		Times(3)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// Verify that NotifyTrigger was called for each trigger in each event
	expectedCalls := 3 // 2 triggers from first event + 1 trigger from second event
	assert.Len(t, mockFetcher.notifyCalls, expectedCalls)

	// Verify first call (first trigger of first event)
	firstCall := mockFetcher.notifyCalls[0]
	assert.Equal(t, "user.created", firstCall.data["event"])
	assert.Equal(t, "test-message-id", firstCall.data["id"])
	assert.Equal(t, []byte(`{"user_id": "123", "action": "create"}`), firstCall.data["data"])
	assert.Equal(t, map[string]string{"source": "api", "version": "1.0"}, firstCall.data["metadata"])
	// Don't compare exact time due to marshaling precision loss
	assert.NotNil(t, firstCall.data["event_at"])
	assert.Equal(t, map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"}, firstCall.headers)
	assert.Equal(t, Trigger{
		ServiceName: "notification-service",
		Type:        TriggerType("webhook"),
		BaseUrl:     "http://localhost:8080",
		Path:        "/webhook/user-created",
		Headers:     map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"},
	}, firstCall.trigger)

	// Verify second call (second trigger of first event)
	secondCall := mockFetcher.notifyCalls[1]
	assert.Equal(t, "user.created", secondCall.data["event"])
	assert.Equal(t, "test-message-id", secondCall.data["id"])
	assert.Equal(t, []byte(`{"user_id": "123", "action": "create"}`), secondCall.data["data"])
	assert.Equal(t, map[string]string{"source": "api", "version": "1.0"}, secondCall.data["metadata"])
	assert.NotNil(t, secondCall.data["event_at"])
	assert.Equal(t, map[string]string{"X-API-Key": "analytics-key"}, secondCall.headers)
	assert.Equal(t, Trigger{
		ServiceName: "analytics-service",
		Type:        TriggerType("webhook"),
		BaseUrl:     "http://localhost:8081",
		Path:        "/analytics/event",
		Headers:     map[string]string{"X-API-Key": "analytics-key"},
	}, secondCall.trigger)

	// Verify third call (first trigger of second event)
	thirdCall := mockFetcher.notifyCalls[2]
	assert.Equal(t, "order.completed", thirdCall.data["event"])
	assert.Equal(t, "test-message-id", thirdCall.data["id"])
	assert.Equal(t, []byte(`{"user_id": "123", "action": "create"}`), thirdCall.data["data"])
	assert.Equal(t, map[string]string{"source": "api", "version": "1.0"}, thirdCall.data["metadata"])
	assert.NotNil(t, thirdCall.data["event_at"])
	assert.Equal(t, map[string]string{"Content-Type": "application/json"}, thirdCall.headers)
	assert.Equal(t, Trigger{
		ServiceName: "email-service",
		Type:        TriggerType("webhook"),
		BaseUrl:     "http://localhost:8082",
		Path:        "/send-confirmation",
		Headers:     map[string]string{"Content-Type": "application/json"},
	}, thirdCall.trigger)
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

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(nil, domain.EventNotFound).
		Times(1)

	mockFetcher := &mockFetcherWithCalls{
		MockFetcher: NewMockFetcher(ctrl),
		notifyCalls: []notifyCall{},
	}

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	// Should return nil when EventNotFound
	require.NoError(t, err)

	// Verify that NotifyTrigger was not called
	assert.Empty(t, mockFetcher.notifyCalls)
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

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(nil, expectedError).
		Times(1)

	mockFetcher := &mockFetcherWithCalls{
		MockFetcher: NewMockFetcher(ctrl),
		notifyCalls: []notifyCall{},
	}

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get all schedulers:")
	assert.Contains(t, err.Error(), "database connection failed")

	// Verify that NotifyTrigger was not called
	assert.Empty(t, mockFetcher.notifyCalls)
}

func TestDeadLetterQueue_Handler_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockDeadLetterStore(ctrl)
	mockFetcher := NewMockFetcher(ctrl)

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

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return([]domain.Event{}, nil).
		Times(1)

	mockFetcher := &mockFetcherWithCalls{
		MockFetcher: NewMockFetcher(ctrl),
		notifyCalls: []notifyCall{},
	}

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// Verify that NotifyTrigger was not called since there are no events
	assert.Empty(t, mockFetcher.notifyCalls)
}

func TestDeadLetterQueue_Handler_EventsWithNoTriggers(t *testing.T) {
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
			Name:        "event.without.triggers",
			ServiceName: "test-service",
			State:       "archived",
			Triggers:    []domain.Trigger{}, // No triggers
		},
	}

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	mockFetcher := &mockFetcherWithCalls{
		MockFetcher: NewMockFetcher(ctrl),
		notifyCalls: []notifyCall{},
	}

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// Verify that NotifyTrigger was not called since there are no triggers
	assert.Empty(t, mockFetcher.notifyCalls)
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
			Triggers: []domain.Trigger{
				{
					ServiceName: "webhook-service",
					Type:        "webhook",
					Host:        "http://localhost:8080",
					Path:        "/webhook",
					Headers:     map[string]string{"Content-Type": "application/json"},
				},
			},
		},
	}

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	fetcherError := errors.New("webhook delivery failed")
	mockFetcher := &mockFetcherWithCalls{
		MockFetcher: NewMockFetcher(ctrl),
		notifyCalls: []notifyCall{},
	}

	mockFetcher.MockFetcher.EXPECT().
		NotifyTrigger(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(fetcherError).
		Times(1)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	// The handler should continue even if NotifyTrigger fails (no error propagation in current implementation)
	require.NoError(t, err)

	// Verify that NotifyTrigger was called despite the error
	assert.Len(t, mockFetcher.notifyCalls, 1)
}

func TestDeadLetterQueue_Handler_MultipleEventsWithMixedTriggers(t *testing.T) {
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
			Triggers: []domain.Trigger{
				{
					ServiceName: "inventory-service",
					Type:        "webhook",
					Host:        "http://inventory.local",
					Path:        "/reserve",
					Headers:     map[string]string{"Authorization": "Bearer inventory-token"},
				},
			},
		},
		{
			Name:        "notification.send",
			ServiceName: "notification-service",
			State:       "archived",
			Triggers:    []domain.Trigger{}, // Event with no triggers
		},
		{
			Name:        "audit.log",
			ServiceName: "audit-service",
			State:       "archived",
			Triggers: []domain.Trigger{
				{
					ServiceName: "logging-service",
					Type:        "webhook",
					Host:        "http://logs.local",
					Path:        "/audit",
					Headers:     map[string]string{"X-Service": "audit"},
				},
				{
					ServiceName: "backup-service",
					Type:        "webhook",
					Host:        "http://backup.local",
					Path:        "/store",
					Headers:     map[string]string{"X-Backup": "true"},
				},
			},
		},
	}

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return(mockEvents, nil).
		Times(1)

	mockFetcher := &mockFetcherWithCalls{
		MockFetcher: NewMockFetcher(ctrl),
		notifyCalls: []notifyCall{},
	}

	// Expect 3 calls to NotifyTrigger: 1 from first event + 0 from second event + 2 from third event
	mockFetcher.MockFetcher.EXPECT().
		NotifyTrigger(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		Times(3)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// Should have 3 calls: 1 from first event + 0 from second event + 2 from third event
	assert.Len(t, mockFetcher.notifyCalls, 3)

	// Verify calls contain expected event names
	eventNames := make([]string, len(mockFetcher.notifyCalls))
	for i, call := range mockFetcher.notifyCalls {
		eventNames[i] = call.data["event"].(string)
	}

	assert.Contains(t, eventNames, "order.created")
	assert.Contains(t, eventNames, "audit.log")
	assert.NotContains(t, eventNames, "notification.send") // This event has no triggers

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
	assert.Equal(t, 2, auditCount, "audit.log should appear twice (2 triggers)")
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

	mockStore := NewMockDeadLetterStore(ctrl)
	mockStore.EXPECT().
		GetAllSchedulers(gomock.Any(), "archived").
		Return([]domain.Event{}, nil).
		Times(1)

	mockFetcher := NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(mockStore, mockFetcher)

	// Marshal pubsub message for AsyncCtx
	messageBytes, err := json.Marshal(pubsubMessage)
	require.NoError(t, err)

	asyncCtx := asyncadapter.NewAsyncCtx[pubsub.Message](context.Background(), messageBytes)
	err = handle.Handler(asyncCtx)

	require.NoError(t, err)

	// The expectation on the mock already verifies that "archived" state was passed
}
