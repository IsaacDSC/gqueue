package interstore

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

type PersistentStore interface {
	GetAllEvents(ctx context.Context) ([]domain.Event, error)
}

type MemStore struct {
	topicEvents atomic.Value
	retryTopics atomic.Value
	tag         string

	persitentStore PersistentStore
}

func NewMemStore(persistentStore PersistentStore) *MemStore {
	ms := &MemStore{}
	ms.topicEvents.Store(make(map[string]domain.Event))
	ms.retryTopics.Store(make(map[string]domain.Event))
	ms.tag = "mem_store"
	ms.persitentStore = persistentStore

	return ms
}

func (ms *MemStore) LoadInMemStore(ctx context.Context) error {
	l := ctxlogger.GetLogger(ctx)
	events, err := ms.persitentStore.GetAllEvents(ctx)
	if err != nil && !errors.Is(err, domain.EventNotFound) {
		l.Error("Error loading events into in-memory store", "error", err)
		return fmt.Errorf("error loading events into in-memory store: %w", err)
	}

	ms.Refresh(ctx, events)

	return nil
}

func (ms *MemStore) GetEvent(ctx context.Context, eventName string) (domain.Event, error) {
	l := ctxlogger.GetLogger(ctx)

	eventsMap := ms.topicEvents.Load().(map[string]domain.Event)
	event, exists := eventsMap[eventName]
	if !exists {
		l.Warn("Event not found", "event_name", eventName, "tag", ms.tag)
		return domain.Event{}, domain.EventNotFound
	}

	return event, nil
}

// DEPRECATED GetAllSchedulers is deprecated and should not be used. It will be removed in future versions. Please use GetRetryEvent by eventName with state parameter instead.
func (ms *MemStore) GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error) {
	return nil, domain.EventNotFound
}

func (ms *MemStore) Refresh(ctx context.Context, events []domain.Event) {
	l := ctxlogger.GetLogger(ctx)

	// Convert slice of events to a map for efficient lookups
	eventsMap := make(map[string]domain.Event)
	for _, event := range events {
		eventsMap[event.Name] = event
	}

	// Update the in-memory map with the latest events
	ms.topicEvents.Store(eventsMap)

	l.Info("Refreshed in-memory store", "num_events", len(events), "tag", ms.tag)
}

func (ms *MemStore) RefreshRetryTopics(ctx context.Context, events []domain.Event) {
	l := ctxlogger.GetLogger(ctx)

	// Convert slice of events to a map for efficient lookups
	retryTopics := make(map[string]domain.Event)
	for _, event := range events {
		retryTopics[event.Name] = event
	}

	// Update the in-memory map with the latest events
	ms.retryTopics.Store(retryTopics)

	l.Info("Refreshed retry topics", "num_events", len(events), "tag", ms.tag)
}
