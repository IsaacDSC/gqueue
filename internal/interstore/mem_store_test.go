package interstore

import (
	"context"
	"testing"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/google/uuid"
)

func TestMemStore_GetEvent(t *testing.T) {
	tests := []struct {
		name          string
		setupEvents   []domain.Event
		searchName    string
		expectedEvent domain.Event
		wantErr       bool
		wantErrType   error
	}{
		{
			name: "get existing event",
			setupEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "test-event",
					ServiceName: "test-service",
					State:       "active",
				},
			},
			searchName: "test-event",
			expectedEvent: domain.Event{
				ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				Name:        "test-event",
				ServiceName: "test-service",
				State:       "active",
			},
			wantErr: false,
		},
		{
			name: "get non-existing event",
			setupEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "test-event",
					ServiceName: "test-service",
					State:       "active",
				},
			},
			searchName:    "non-existing-event",
			expectedEvent: domain.Event{},
			wantErr:       true,
			wantErrType:   domain.EventNotFound,
		},
		{
			name:          "get event from empty store",
			setupEvents:   []domain.Event{},
			searchName:    "any-event",
			expectedEvent: domain.Event{},
			wantErr:       true,
			wantErrType:   domain.EventNotFound,
		},
		{
			name: "get event with multiple events in store",
			setupEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "event-one",
					ServiceName: "service-one",
					State:       "active",
				},
				{
					ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					Name:        "event-two",
					ServiceName: "service-two",
					State:       "inactive",
				},
				{
					ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
					Name:        "event-three",
					ServiceName: "service-three",
					State:       "active",
				},
			},
			searchName: "event-two",
			expectedEvent: domain.Event{
				ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
				Name:        "event-two",
				ServiceName: "service-two",
				State:       "inactive",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStore()
			ctx := context.Background()

			ms.Refresh(ctx, tt.setupEvents)

			event, err := ms.GetEvent(ctx, tt.searchName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.wantErrType != nil && err != tt.wantErrType {
					t.Errorf("expected error %v, got %v", tt.wantErrType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if event.ID != tt.expectedEvent.ID {
				t.Errorf("expected ID %v, got %v", tt.expectedEvent.ID, event.ID)
			}
			if event.Name != tt.expectedEvent.Name {
				t.Errorf("expected Name %v, got %v", tt.expectedEvent.Name, event.Name)
			}
			if event.ServiceName != tt.expectedEvent.ServiceName {
				t.Errorf("expected ServiceName %v, got %v", tt.expectedEvent.ServiceName, event.ServiceName)
			}
			if event.State != tt.expectedEvent.State {
				t.Errorf("expected State %v, got %v", tt.expectedEvent.State, event.State)
			}
		})
	}
}

func TestMemStore_Refresh(t *testing.T) {
	tests := []struct {
		name           string
		initialEvents  []domain.Event
		refreshEvents  []domain.Event
		expectedCount  int
		checkEventName string
		expectFound    bool
	}{
		{
			name:          "refresh empty store with events",
			initialEvents: []domain.Event{},
			refreshEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "new-event",
					ServiceName: "new-service",
					State:       "active",
				},
			},
			expectedCount:  1,
			checkEventName: "new-event",
			expectFound:    true,
		},
		{
			name: "refresh replaces all events",
			initialEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "old-event",
					ServiceName: "old-service",
					State:       "active",
				},
			},
			refreshEvents: []domain.Event{
				{
					ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					Name:        "new-event",
					ServiceName: "new-service",
					State:       "active",
				},
			},
			expectedCount:  1,
			checkEventName: "old-event",
			expectFound:    false,
		},
		{
			name: "refresh with empty list clears store",
			initialEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "existing-event",
					ServiceName: "existing-service",
					State:       "active",
				},
			},
			refreshEvents:  []domain.Event{},
			expectedCount:  0,
			checkEventName: "existing-event",
			expectFound:    false,
		},
		{
			name:          "refresh with multiple events",
			initialEvents: []domain.Event{},
			refreshEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "event-one",
					ServiceName: "service-one",
					State:       "active",
				},
				{
					ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					Name:        "event-two",
					ServiceName: "service-two",
					State:       "inactive",
				},
				{
					ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
					Name:        "event-three",
					ServiceName: "service-three",
					State:       "active",
				},
			},
			expectedCount:  3,
			checkEventName: "event-two",
			expectFound:    true,
		},
		{
			name:          "refresh with duplicate event names keeps last",
			initialEvents: []domain.Event{},
			refreshEvents: []domain.Event{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:        "duplicate-event",
					ServiceName: "first-service",
					State:       "inactive",
				},
				{
					ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					Name:        "duplicate-event",
					ServiceName: "second-service",
					State:       "active",
				},
			},
			expectedCount:  1,
			checkEventName: "duplicate-event",
			expectFound:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStore()
			ctx := context.Background()

			if len(tt.initialEvents) > 0 {
				ms.Refresh(ctx, tt.initialEvents)
			}

			ms.Refresh(ctx, tt.refreshEvents)

			eventsMap := ms.topicEvents.Load().(map[string]domain.Event)
			if len(eventsMap) != tt.expectedCount {
				t.Errorf("expected %d events, got %d", tt.expectedCount, len(eventsMap))
			}

			_, err := ms.GetEvent(ctx, tt.checkEventName)
			found := err == nil

			if found != tt.expectFound {
				t.Errorf("expected event '%s' found=%v, got found=%v", tt.checkEventName, tt.expectFound, found)
			}
		})
	}
}

func TestMemStore_DuplicateEventLastWins(t *testing.T) {
	ms := NewMemStore()
	ctx := context.Background()

	events := []domain.Event{
		{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:        "same-name",
			ServiceName: "first-service",
			State:       "inactive",
		},
		{
			ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Name:        "same-name",
			ServiceName: "second-service",
			State:       "active",
		},
	}

	ms.Refresh(ctx, events)

	event, err := ms.GetEvent(ctx, "same-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.ServiceName != "second-service" {
		t.Errorf("expected ServiceName 'second-service', got '%s'", event.ServiceName)
	}
	if event.State != "active" {
		t.Errorf("expected State 'active', got '%s'", event.State)
	}
}
