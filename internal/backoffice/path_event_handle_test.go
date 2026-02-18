package backoffice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetPathEventHandle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	validUUID := uuid.New()

	tests := []struct {
		name           string
		eventID        string
		payload        domain.Event
		setupMocks     func(*MockRepository, *cachemanager.MockCache)
		expectedStatus int
	}{
		{
			name:    "success_update",
			eventID: validUUID.String(),
			payload: domain.Event{
				Name:        "user.updated",
				ServiceName: "user-service",
				State:       "active",
				Type:        "external",
				Option: domain.Opt{
					MaxRetries: 3,
				},
				Consumers: []domain.Consumer{
					{
						ServiceName: "notification-service",
						Host:        "https://api.example.com",
						Path:        "/webhook",
						Headers:     map[string]string{"Content-Type": "application/json"},
					},
				},
			},
			setupMocks: func(mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue:user-service:user.updated")
				ttl := 5 * time.Minute

				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "user-service", "user.updated").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(ttl)
				mockCache.EXPECT().Hydrate(
					gomock.Any(),
					key,
					gomock.Any(),
					ttl,
					gomock.Any(),
				).DoAndReturn(func(ctx context.Context, key cachemanager.Key, value any, ttl time.Duration, fn cachemanager.Fn) error {
					_, err := fn(ctx)
					return err
				})

				mockRepo.EXPECT().UpdateEvent(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "redis_error",
			eventID: validUUID.String(),
			payload: domain.Event{
				Name:        "user.created",
				ServiceName: "user-service",
				State:       "active",
			},
			setupMocks: func(mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue:user-service:user.created")
				ttl := 5 * time.Minute

				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "user-service", "user.created").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(ttl)
				mockCache.EXPECT().Hydrate(
					gomock.Any(),
					key,
					gomock.Any(),
					ttl,
					gomock.Any(),
				).Return(errors.New("redis connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:    "database_error",
			eventID: validUUID.String(),
			payload: domain.Event{
				Name:        "order.processed",
				ServiceName: "order-service",
				State:       "active",
			},
			setupMocks: func(mockRepo *MockRepository, mockCache *cachemanager.MockCache) {
				key := cachemanager.Key("event-queue:order-service:order.processed")
				ttl := 5 * time.Minute

				mockCache.EXPECT().Key(domain.CacheKeyEventPrefix, "order-service", "order.processed").Return(key)
				mockCache.EXPECT().GetDefaultTTL().Return(ttl)
				mockCache.EXPECT().Hydrate(
					gomock.Any(),
					key,
					gomock.Any(),
					ttl,
					gomock.Any(),
				).DoAndReturn(func(ctx context.Context, key cachemanager.Key, value any, ttl time.Duration, fn cachemanager.Fn) error {
					_, err := fn(ctx)
					return err
				})

				mockRepo.EXPECT().UpdateEvent(gomock.Any(), gomock.Any()).Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid_uuid",
			eventID:        "invalid-uuid",
			payload:        domain.Event{},
			setupMocks:     func(mockRepo *MockRepository, mockCache *cachemanager.MockCache) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockRepository(ctrl)
			mockCache := cachemanager.NewMockCache(ctrl)

			tt.setupMocks(mockRepo, mockCache)

			handle := GetPathEventHandle(mockCache, mockRepo)

			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/event/"+tt.eventID, bytes.NewReader(payloadBytes))
			req.SetPathValue("id", tt.eventID)

			rr := httptest.NewRecorder()
			handle.Handler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
