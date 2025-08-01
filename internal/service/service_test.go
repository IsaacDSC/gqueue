package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IsaacDSC/webhook/internal/infra/cache"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/structs"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCreateInternalEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepository(ctrl)
	mockPublisher := publisher.NewMockPublisher(ctrl)
	mockCache := cache.NewMockCache(ctrl)

	// Set up mock keys for all test cases
	testKey := cache.Key("webhook:internal_events:test-event")
	mockCache.EXPECT().Key("webhook", "internal_events", "test-event").Return(testKey).AnyTimes()

	existingEventKey := cache.Key("webhook:internal_events:existing-event")
	mockCache.EXPECT().Key("webhook", "internal_events", "existing-event").Return(existingEventKey).AnyTimes()

	errorEventKey := cache.Key("webhook:internal_events:error-event")
	mockCache.EXPECT().Key("webhook", "internal_events", "error-event").Return(errorEventKey).AnyTimes()

	cacheErrorEventKey := cache.Key("webhook:internal_events:cache-error-event")
	mockCache.EXPECT().Key("webhook", "internal_events", "cache-error-event").Return(cacheErrorEventKey).AnyTimes()

	mockCache.EXPECT().GetDefaultTTL().Return(time.Minute * 5).AnyTimes()

	testService := NewService(mockRepo, mockPublisher, mockCache)

	t.Run("Success - Create new internal event", func(t *testing.T) {
		ctx := context.Background()
		input := structs.CreateInternalEventDto{
			EventName:   "test-event",
			ServiceName: "test-service",
			RepoUrl:     "https://github.com/test/repo",
			TeamOwner:   "test-team",
		}

		expectedEvent := input.ToInternalEvent()

		// Mock repository call to check if event exists
		mockRepo.EXPECT().GetInternalEvent(gomock.Any(), input.EventName).Return(structs.InternalEvent{}, nil)

		// Mock cache hydrate
		mockCache.EXPECT().Hydrate(gomock.Any(), testKey, gomock.Any(), time.Minute*5, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ cache.Key, value any, _ time.Duration, fn cache.Fn) error {
				// Mock repository.CreateInternalEvent
				mockRepo.EXPECT().CreateInternalEvent(gomock.Any(), gomock.Any()).Return(nil)

				result, err := fn(ctx)
				if err != nil {
					return err
				}

				// Copy result to value
				*(value.(*structs.InternalEvent)) = result.(structs.InternalEvent)
				return nil
			})

		// Call service
		result, err := testService.CreateInternalEvent(ctx, input)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent.Name, result.Name)
		assert.Equal(t, expectedEvent.ServiceName, result.ServiceName)
		assert.Equal(t, expectedEvent.RepoUrl, result.RepoUrl)
		assert.Equal(t, expectedEvent.TeamOwner, result.TeamOwner)
		assert.NotEqual(t, uuid.Nil, result.ID)
	})

	t.Run("Error - Event already exists", func(t *testing.T) {
		ctx := context.Background()
		input := structs.CreateInternalEventDto{
			EventName:   "existing-event",
			ServiceName: "test-service",
		}

		existingEvent := structs.InternalEvent{
			ID:   uuid.New(),
			Name: "existing-event",
		}

		// Mock repository call returning existing event
		mockRepo.EXPECT().GetInternalEvent(gomock.Any(), input.EventName).Return(existingEvent, nil)

		// Call service
		_, err := testService.CreateInternalEvent(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "internal event already exists")
	})

	t.Run("Error - Repository error", func(t *testing.T) {
		ctx := context.Background()
		input := structs.CreateInternalEventDto{
			EventName: "error-event",
		}

		// Mock repository call with error
		mockRepo.EXPECT().GetInternalEvent(gomock.Any(), input.EventName).
			Return(structs.InternalEvent{}, errors.New("database error"))

		// Call service
		_, err := testService.CreateInternalEvent(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to get internal event")
	})

	t.Run("Error - Cache error", func(t *testing.T) {
		ctx := context.Background()
		input := structs.CreateInternalEventDto{
			EventName: "cache-error-event",
		}

		// Mock repository call
		mockRepo.EXPECT().GetInternalEvent(gomock.Any(), input.EventName).Return(structs.InternalEvent{}, nil)

		// Mock cache error
		mockCache.EXPECT().Hydrate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("cache error"))

		// Call service
		_, err := testService.CreateInternalEvent(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create internal event")
	})
}

func TestRegisterTrigger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepository(ctrl)
	mockPublisher := publisher.NewMockPublisher(ctrl)
	mockCache := cache.NewMockCache(ctrl)

	// Set up mock key
	testKey := cache.Key("webhook:internal_events:test-event")
	mockCache.EXPECT().Key("webhook", "internal_events", "test-event").Return(testKey).AnyTimes()
	mockCache.EXPECT().GetDefaultTTL().Return(time.Minute * 5).AnyTimes()

	testService := NewService(mockRepo, mockPublisher, mockCache)

	t.Run("Success - Register new trigger", func(t *testing.T) {
		ctx := context.Background()
		input := structs.RegisterTriggersDto{
			EventName: "test-event",
			Trigger: structs.TriggerDto{
				Type:    string(structs.TriggerTypePersistent),
				BaseUrl: "https://example.com",
				Path:    "/webhook",
			},
		}

		existingEvent := structs.InternalEvent{
			ID:        uuid.New(),
			Name:      "test-event",
			Triggers:  []structs.Trigger{},
			CreatedAt: time.Now(),
		}

		// Mock repository call
		mockRepo.EXPECT().GetInternalEvent(gomock.Any(), input.EventName).Return(existingEvent, nil)

		// Mock cache hydrate
		mockCache.EXPECT().Hydrate(gomock.Any(), testKey, gomock.Any(), time.Minute*5, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ cache.Key, value any, _ time.Duration, fn cache.Fn) error {
				// Mock repository.SaveInternalEvent
				mockRepo.EXPECT().SaveInternalEvent(gomock.Any(), gomock.Any()).Return(nil)

				result, err := fn(ctx)
				if err != nil {
					return err
				}

				// Copy result to value
				*(value.(*structs.InternalEvent)) = result.(structs.InternalEvent)
				return nil
			})

		// Call service
		result, err := testService.RegisterTrigger(ctx, input)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, existingEvent.ID, result.ID)
		assert.Equal(t, existingEvent.Name, result.Name)
		assert.Len(t, result.Triggers, 1)
		assert.Equal(t, input.Trigger.Path, result.Triggers[0].Path)
	})

	t.Run("Error - Empty event name", func(t *testing.T) {
		ctx := context.Background()
		input := structs.RegisterTriggersDto{
			EventName: "",
		}

		// Call service
		_, err := testService.RegisterTrigger(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event name and triggers are required")
	})

	t.Run("Error - Event not found", func(t *testing.T) {
		ctx := context.Background()
		input := structs.RegisterTriggersDto{
			EventName: "non-existent-event",
		}

		// Mock repository call returning nil UUID (not found)
		mockRepo.EXPECT().GetInternalEvent(gomock.Any(), input.EventName).
			Return(structs.InternalEvent{}, nil)

		// Call service
		_, err := testService.RegisterTrigger(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found internal event")
	})

	t.Run("Error - Trigger already exists", func(t *testing.T) {
		ctx := context.Background()
		path := "/webhook"
		input := structs.RegisterTriggersDto{
			EventName: "test-event",
			Trigger: structs.TriggerDto{
				Type:    string(structs.TriggerTypePersistent),
				BaseUrl: "https://example.com",
				Path:    path,
			},
		}

		existingTrigger := structs.Trigger{
			ID:          uuid.New(),
			ServiceName: "test-service",
			Type:        structs.TriggerTypePersistent,
			BaseUrl:     "https://example.com",
			Path:        path,
			CreatedAt:   time.Now(),
		}

		existingEvent := structs.InternalEvent{
			ID:        uuid.New(),
			Name:      "test-event",
			Triggers:  []structs.Trigger{existingTrigger},
			CreatedAt: time.Now(),
		}

		// Mock repository call returning event with existing trigger
		mockRepo.EXPECT().GetInternalEvent(gomock.Any(), input.EventName).Return(existingEvent, nil)

		// Call service
		_, err := testService.RegisterTrigger(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trigger already exists")
	})
}

func TestPublisherExternalEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepository(ctrl)
	mockPublisher := publisher.NewMockPublisher(ctrl)
	mockCache := cache.NewMockCache(ctrl)

	testService := NewService(mockRepo, mockPublisher, mockCache)

	t.Run("Success - Publish external event", func(t *testing.T) {
		ctx := context.Background()
		input := structs.PublisherExternalEventDto{
			EventName: "test-event",
			Data:      map[string]interface{}{"key": "value"},
		}

		// Mock publisher - use the correct task name "publisher_external_event"
		mockPublisher.EXPECT().Publish(gomock.Any(), "publisher_external_event", input, gomock.Any()).Return(nil)

		// Call service
		err := testService.PublisherExternalEvent(ctx, input)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error - Publisher error", func(t *testing.T) {
		ctx := context.Background()
		input := structs.PublisherExternalEventDto{
			EventName: "error-event",
		}

		// Mock publisher with error - use the correct task name "publisher_external_event"
		mockPublisher.EXPECT().Publish(gomock.Any(), "publisher_external_event", input, gomock.Any()).
			Return(errors.New("publisher error"))

		// Call service
		err := testService.PublisherExternalEvent(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publisher error")
	})
}
