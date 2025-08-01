package intersvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/IsaacDSC/webhook/pkg/cache"
	"github.com/google/uuid"
)

type Repository interface {
	GetInternalEvent(ctx context.Context, eventName string) (InternalEvent, error)
	CreateInternalEvent(ctx context.Context, event InternalEvent) error
	SaveInternalEvent(ctx context.Context, event InternalEvent) error
}

type Web struct {
	repo  Repository
	cache cache.Cache
}

func NewWeb(repo Repository, cache cache.Cache) *Web {
	return &Web{repo: repo, cache: cache}
}

func (wb Web) CreateInternalEvent(ctx context.Context, input CreateInternalEventDto) (output InternalEvent, err error) {
	key := wb.getKey(ctx, input.EventName)
	defaultTTL := wb.cache.GetDefaultTTL()

	internalEvent, err := wb.repo.GetInternalEvent(ctx, input.EventName)
	if err != nil {
		return InternalEvent{}, fmt.Errorf("unable to get internal event: %w", err)
	}

	if internalEvent.ID != uuid.Nil {
		return InternalEvent{}, errors.New("internal event already exists")
	}

	if err := wb.cache.Hydrate(ctx, key, &output, defaultTTL, func(ctx context.Context) (any, error) {
		model := input.ToInternalEvent()
		if err := wb.repo.CreateInternalEvent(ctx, model); err != nil {
			return InternalEvent{}, err
		}
		return model, nil
	}); err != nil {
		return InternalEvent{}, fmt.Errorf("failed to create internal event: %w", err)
	}

	return output, nil
}

func (s Web) RegisterTrigger(ctx context.Context, input RegisterTriggersDto) (InternalEvent, error) {
	if input.EventName == "" {
		return InternalEvent{}, errors.New("event name and triggers are required")
	}

	internalEvent, err := s.repo.GetInternalEvent(ctx, input.EventName)
	if err != nil {
		return InternalEvent{}, fmt.Errorf("unable to get internal event: %w", err)
	}

	if internalEvent.ID == uuid.Nil {
		return InternalEvent{}, errors.New("not found internal event")
	}

	if len(internalEvent.Triggers) > 0 {
		if internalEvent.Triggers.AlreadyExist(input.ToTrigger()) {
			return InternalEvent{}, errors.New("trigger already exists")
		} else {
			internalEvent.Triggers = internalEvent.Triggers.Add(input.ToTrigger())
		}
	} else {
		internalEvent.Triggers = internalEvent.Triggers.Add(input.ToTrigger())
	}

	key := s.getKey(ctx, input.EventName)
	defaultTTL := s.cache.GetDefaultTTL()
	if err := s.cache.Hydrate(ctx, key, &internalEvent, defaultTTL, func(ctx context.Context) (any, error) {
		if err := s.repo.SaveInternalEvent(ctx, internalEvent); err != nil {
			return InternalEvent{}, errors.New("failed to create internal event")
		}
		return internalEvent, nil
	}); err != nil {
		return InternalEvent{}, fmt.Errorf("failed to register trigger: %w", err)
	}

	return internalEvent, nil
}

func (s Web) CreateConsumer(ctx context.Context, event InternalEvent) error {
	key := s.getKey(ctx, event.Name)
	defaultTTL := s.cache.GetDefaultTTL()

	if err := s.cache.Hydrate(ctx, key, &event, defaultTTL, func(ctx context.Context) (any, error) {
		if err := s.repo.CreateInternalEvent(ctx, event); err != nil {
			return InternalEvent{}, fmt.Errorf("failed to create internal event: %w", err)
		}
		return event, nil
	}); err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	return nil
}

func (wb Web) getKey(ctx context.Context, eventName string) cache.Key {
	return wb.cache.Key("webhook", "internal_events", eventName)
}
