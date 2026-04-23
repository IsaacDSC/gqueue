package pubsubapp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/notifyopt"
	"github.com/IsaacDSC/gqueue/mocks/mockpubsubapp"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDeadLatterQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	fetcher := mockpubsubapp.NewMockFetcher(ctrl)

	handle := NewDeadLatterQueue(store, fetcher)
	assert.Equal(t, domain.EventQueueDeadLetter, handle.EventName)
	assert.NotNil(t, handle.Handler)
}

func TestDeadLetterQueue_Handler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	payload := RequestPayload{EventName: "user.created", Data: map[string]any{"user_id": "123"}, Headers: map[string]string{"source": "api"}, PublishedAt: time.Now().UnixMilli()}
	event := domain.Event{Name: "user.created.dlq", Consumers: []domain.Consumer{{ServiceName: "notification-service", BaseUrl: "http://localhost:8080", Path: "/webhook/user-created", Headers: map[string]string{"Authorization": "Bearer token"}}, {ServiceName: "analytics-service", BaseUrl: "http://localhost:8081", Path: "/analytics/event", Headers: map[string]string{"X-API-Key": "analytics-key"}}}}

	store := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	store.EXPECT().GetAllSchedulers(gomock.Any(), "user.created.dlq").Return(event, nil).Times(1)

	fetcher := mockpubsubapp.NewMockFetcher(ctrl)
	fetcher.EXPECT().Notify(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), notifyopt.HighThroughput).DoAndReturn(
		func(_ context.Context, data map[string]any, headers map[string]string, consumer domain.Consumer, _ notifyopt.Kind) error {
			assert.Equal(t, "user.created.dlq", data["event"])
			assert.Equal(t, payload.Data, data["data"])
			assert.Equal(t, payload.Headers, data["metadata"])
			assert.Equal(t, payload.PublishedAt, data["event_at"])
			assert.Equal(t, headers, consumer.Headers)
			return nil
		}).Times(2)

	handle := NewDeadLatterQueue(store, fetcher)
	err := handle.Handler(asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), mustMarshal(t, payload)))
	require.NoError(t, err)
}

func TestDeadLetterQueue_Handler_EventNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	store.EXPECT().GetAllSchedulers(gomock.Any(), "unknown.dlq").Return(domain.Event{}, domain.EventNotFound).Times(1)

	handle := NewDeadLatterQueue(store, mockpubsubapp.NewMockFetcher(ctrl))
	err := handle.Handler(asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), mustMarshal(t, RequestPayload{EventName: "unknown", Data: map[string]any{"x": 1}})))
	require.NoError(t, err)
}

func TestDeadLetterQueue_Handler_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storeErr := errors.New("database connection failed")
	store := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	store.EXPECT().GetAllSchedulers(gomock.Any(), "user.created.dlq").Return(domain.Event{}, storeErr).Times(1)

	handle := NewDeadLatterQueue(store, mockpubsubapp.NewMockFetcher(ctrl))
	err := handle.Handler(asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), mustMarshal(t, RequestPayload{EventName: "user.created", Data: map[string]any{"x": 1}})))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get all schedulers")
	assert.Contains(t, err.Error(), storeErr.Error())
}

func TestDeadLetterQueue_Handler_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handle := NewDeadLatterQueue(mockpubsubapp.NewMockDeadLetterStore(ctrl), mockpubsubapp.NewMockFetcher(ctrl))
	err := handle.Handler(asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), []byte("invalid json")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get payload")
}

func TestDeadLetterQueue_Handler_FetcherError_DoesNotFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockpubsubapp.NewMockDeadLetterStore(ctrl)
	store.EXPECT().GetAllSchedulers(gomock.Any(), "order.created.dlq").Return(domain.Event{Name: "order.created.dlq", Consumers: []domain.Consumer{{ServiceName: "svc", BaseUrl: "http://localhost", Path: "/webhook"}}}, nil).Times(1)

	fetcher := mockpubsubapp.NewMockFetcher(ctrl)
	fetcher.EXPECT().Notify(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), notifyopt.HighThroughput).Return(errors.New("webhook failed")).Times(1)

	handle := NewDeadLatterQueue(store, fetcher)
	err := handle.Handler(asyncadapter.NewAsyncCtx[RequestPayload](context.Background(), mustMarshal(t, RequestPayload{EventName: "order.created", Data: map[string]any{"order_id": "o-1"}})))
	require.NoError(t, err)
}

func mustMarshal(t *testing.T, payload RequestPayload) []byte {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return b
}
