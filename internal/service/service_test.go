package service

import (
	"context"
	"errors"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/infra/task"
	"github.com/IsaacDSC/webhook/internal/structs"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestWebhook_CreateInternalEvent(t *testing.T) {
	// Test table structure
	tests := []struct {
		name          string
		input         structs.CreateInternalEventDto
		setupMocks    func(mockRepo *repository.MockRepository)
		expectedEvent structs.InternalEvent
		expectedError error
	}{
		{
			name: "Success",
			input: structs.CreateInternalEventDto{
				EventName:   "test-event",
				ServiceName: "test-service",
				RepoUrl:     "https://github.com/test/repo",
				TeamOwner:   "test-team",
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				mockRepo.EXPECT().CreateInternalEvent(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedEvent: structs.InternalEvent{
				Name:        "test-event",
				ServiceName: "test-service",
				RepoUrl:     "https://github.com/test/repo",
				TeamOwner:   "test-team",
				Triggers:    []structs.Trigger{},
			},
			expectedError: nil,
		},
		{
			name: "Repository error",
			input: structs.CreateInternalEventDto{
				EventName:   "test-event",
				ServiceName: "test-service",
				RepoUrl:     "https://github.com/test/repo",
				TeamOwner:   "test-team",
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				mockRepo.EXPECT().CreateInternalEvent(gomock.Any(), gomock.Any()).Return(errors.New("repository error"))
			},
			expectedEvent: structs.InternalEvent{},
			expectedError: errors.New("repository error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepository(ctrl)
			mockPublisher := publisher.NewMockPublisher(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo)
			}

			service := NewService(mockRepo, mockPublisher)

			// Execute
			result, err := service.CreateInternalEvent(context.Background(), tt.input)

			// Assert
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				// Since we can't predict UUID and timestamps, we verify non-random fields
				assert.Equal(t, tt.expectedEvent.Name, result.Name)
				assert.Equal(t, tt.expectedEvent.ServiceName, result.ServiceName)
				assert.Equal(t, tt.expectedEvent.RepoUrl, result.RepoUrl)
				assert.Equal(t, tt.expectedEvent.TeamOwner, result.TeamOwner)
				assert.Equal(t, len(tt.expectedEvent.Triggers), len(result.Triggers))
				assert.NotEqual(t, uuid.Nil, result.ID)
				assert.False(t, result.CreatedAt.IsZero())
			}
		})
	}
}

func TestWebhook_PublisherExternalEvent(t *testing.T) {
	tests := []struct {
		name          string
		input         structs.PublisherExternalEventDto
		setupMocks    func(mockPublisher *publisher.MockPublisher)
		expectedError error
	}{
		{
			name: "Success",
			input: structs.PublisherExternalEventDto{
				EventName: "test-event",
				Data: map[string]any{
					"key": "value",
				},
			},
			setupMocks: func(mockPublisher *publisher.MockPublisher) {
				mockPublisher.EXPECT().Publish(
					gomock.Any(),
					task.PublisherExternalEvent.String(),
					gomock.Any(),
				).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "Publisher error",
			input: structs.PublisherExternalEventDto{
				EventName: "test-event",
				Data: map[string]any{
					"key": "value",
				},
			},
			setupMocks: func(mockPublisher *publisher.MockPublisher) {
				mockPublisher.EXPECT().Publish(
					gomock.Any(),
					task.PublisherExternalEvent.String(),
					gomock.Any(),
				).Return(errors.New("publisher error"))
			},
			expectedError: errors.New("publisher error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepository(ctrl)
			mockPublisher := publisher.NewMockPublisher(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockPublisher)
			}

			service := NewService(mockRepo, mockPublisher)

			// Execute
			err := service.PublisherExternalEvent(context.Background(), tt.input)

			// Assert
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhook_RegisterTrigger(t *testing.T) {
	// Create a fixed time for testing
	fixedTime := time.Now()

	// Create a fixed UUID for testing
	fixedID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name          string
		input         structs.RegisterTriggersDto
		setupMocks    func(mockRepo *repository.MockRepository)
		expectedEvent structs.InternalEvent
		expectedError error
	}{
		{
			name: "Success - First trigger",
			input: structs.RegisterTriggersDto{
				EventName: "test-event",
				Trigger: structs.TriggerDto{
					ServiceName: "trigger-service",
					Type:        "persistent",
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
				},
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				// Return existing event with no triggers
				mockRepo.EXPECT().GetInternalEvent(gomock.Any(), "test-event").Return(
					structs.InternalEvent{
						ID:          eventID,
						Name:        "test-event",
						ServiceName: "test-service",
						RepoUrl:     "https://github.com/test/repo",
						TeamOwner:   "test-team",
						Triggers:    []structs.Trigger{},
						CreatedAt:   fixedTime,
					}, nil)

				// Expect CreateInternalEvent to be called with an event containing the new trigger
				mockRepo.EXPECT().CreateInternalEvent(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, ie structs.InternalEvent) error {
						assert.Equal(t, eventID, ie.ID)
						assert.Equal(t, "test-event", ie.Name)
						assert.Equal(t, 1, len(ie.Triggers))
						assert.Equal(t, "trigger-service", ie.Triggers[0].ServiceName)
						assert.Equal(t, structs.TriggerType("persistent"), ie.Triggers[0].Type)
						assert.Equal(t, "https://example.com", ie.Triggers[0].BaseUrl)
						assert.Equal(t, "/webhook", ie.Triggers[0].Path)
						return nil
					})
			},
			expectedEvent: structs.InternalEvent{
				ID:          eventID,
				Name:        "test-event",
				ServiceName: "test-service",
				RepoUrl:     "https://github.com/test/repo",
				TeamOwner:   "test-team",
				Triggers:    []structs.Trigger{},
				CreatedAt:   fixedTime,
			},
			expectedError: nil,
		},
		{
			name: "Success - Add trigger to existing triggers",
			input: structs.RegisterTriggersDto{
				EventName: "test-event",
				Trigger: structs.TriggerDto{
					ServiceName: "trigger-service",
					Type:        "persistent",
					BaseUrl:     "https://example2.com",
					Path:        "/webhook2",
				},
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				// Return existing event with one trigger
				existingTrigger := structs.Trigger{
					ID:          fixedID,
					ServiceName: "trigger-service",
					Type:        structs.TriggerType("persistent"),
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
					CreatedAt:   fixedTime,
				}

				mockRepo.EXPECT().GetInternalEvent(gomock.Any(), "test-event").Return(
					structs.InternalEvent{
						ID:          eventID,
						Name:        "test-event",
						ServiceName: "test-service",
						RepoUrl:     "https://github.com/test/repo",
						TeamOwner:   "test-team",
						Triggers:    []structs.Trigger{existingTrigger},
						CreatedAt:   fixedTime,
					}, nil)

			},
			expectedEvent: structs.InternalEvent{
				ID:          eventID,
				Name:        "test-event",
				ServiceName: "test-service",
				RepoUrl:     "https://github.com/test/repo",
				TeamOwner:   "test-team",
				CreatedAt:   fixedTime,
				// We'll check the triggers in the test case
			},
			expectedError: errors.New("trigger already exists"),
		},
		{
			name: "Error - Empty EventName",
			input: structs.RegisterTriggersDto{
				EventName: "",
				Trigger: structs.TriggerDto{
					ServiceName: "trigger-service",
					Type:        "persistent",
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
				},
			},
			setupMocks:    func(mockRepo *repository.MockRepository) {},
			expectedEvent: structs.InternalEvent{},
			expectedError: errors.New("event name and triggers are required"),
		},
		{
			name: "Error - Event not found",
			input: structs.RegisterTriggersDto{
				EventName: "test-event",
				Trigger: structs.TriggerDto{
					ServiceName: "trigger-service",
					Type:        "persistent",
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
				},
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				mockRepo.EXPECT().GetInternalEvent(gomock.Any(), "test-event").Return(
					structs.InternalEvent{}, nil)
			},
			expectedEvent: structs.InternalEvent{},
			expectedError: errors.New("not found internal event"),
		},
		{
			name: "Error - Trigger already exists",
			input: structs.RegisterTriggersDto{
				EventName: "test-event",
				Trigger: structs.TriggerDto{
					ServiceName: "existing-service",
					Type:        "persistent",
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
				},
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				// Return existing event with a trigger with same service name
				existingTrigger := structs.Trigger{
					ID:          fixedID,
					ServiceName: "existing-service", // Same service name as input
					Type:        structs.TriggerType("persistent"),
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
					CreatedAt:   fixedTime,
				}

				mockRepo.EXPECT().GetInternalEvent(gomock.Any(), "test-event").Return(
					structs.InternalEvent{
						ID:          eventID,
						Name:        "test-event",
						ServiceName: "test-service",
						RepoUrl:     "https://github.com/test/repo",
						TeamOwner:   "test-team",
						Triggers:    []structs.Trigger{existingTrigger},
						CreatedAt:   fixedTime,
					}, nil)
			},
			expectedEvent: structs.InternalEvent{},
			expectedError: errors.New("trigger already exists"),
		},
		{
			name: "Error - GetInternalEvent error",
			input: structs.RegisterTriggersDto{
				EventName: "test-event",
				Trigger: structs.TriggerDto{
					ServiceName: "trigger-service",
					Type:        "persistent",
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
				},
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				mockRepo.EXPECT().GetInternalEvent(gomock.Any(), "test-event").Return(
					structs.InternalEvent{}, errors.New("database error"))
			},
			expectedEvent: structs.InternalEvent{},
			expectedError: errors.New("unable to get internal event: database error"),
		},
		{
			name: "Error - CreateInternalEvent error",
			input: structs.RegisterTriggersDto{
				EventName: "test-event",
				Trigger: structs.TriggerDto{
					ServiceName: "trigger-service",
					Type:        "persistent",
					BaseUrl:     "https://example.com",
					Path:        "/webhook",
				},
			},
			setupMocks: func(mockRepo *repository.MockRepository) {
				// Return existing event with no triggers
				mockRepo.EXPECT().GetInternalEvent(gomock.Any(), "test-event").Return(
					structs.InternalEvent{
						ID:          eventID,
						Name:        "test-event",
						ServiceName: "test-service",
						RepoUrl:     "https://github.com/test/repo",
						TeamOwner:   "test-team",
						Triggers:    []structs.Trigger{},
						CreatedAt:   fixedTime,
					}, nil)

				// CreateInternalEvent fails
				mockRepo.EXPECT().CreateInternalEvent(gomock.Any(), gomock.Any()).Return(
					errors.New("db error"))
			},
			expectedEvent: structs.InternalEvent{},
			expectedError: errors.New("failed to create internal event"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepository(ctrl)
			mockPublisher := publisher.NewMockPublisher(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo)
			}

			service := NewService(mockRepo, mockPublisher)

			// Execute
			result, err := service.RegisterTrigger(context.Background(), tt.input)

			// Assert
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)

				// For specific test cases, add additional assertions
				if tt.name == "Success - Add trigger to existing triggers" {
					assert.Equal(t, 2, len(result.Triggers))

					// First trigger should be the existing one
					assert.Equal(t, "trigger-service", result.Triggers[0].ServiceName)

					// Second trigger should be the new one
					assert.Equal(t, "another-service", result.Triggers[1].ServiceName)
					assert.Equal(t, structs.TriggerType("persistent"), result.Triggers[1].Type)
					assert.Equal(t, "https://example2.com", result.Triggers[1].BaseUrl)
					assert.Equal(t, "/webhook2", result.Triggers[1].Path)
				}

				assert.Equal(t, tt.expectedEvent.ID, result.ID)
				assert.Equal(t, tt.expectedEvent.Name, result.Name)
				assert.Equal(t, tt.expectedEvent.ServiceName, result.ServiceName)
				assert.Equal(t, tt.expectedEvent.RepoUrl, result.RepoUrl)
				assert.Equal(t, tt.expectedEvent.TeamOwner, result.TeamOwner)
			}
		})
	}
}
