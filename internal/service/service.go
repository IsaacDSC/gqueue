package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/IsaacDSC/webhook/internal/infra/cache"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/infra/task"
	"github.com/IsaacDSC/webhook/internal/structs"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/google/uuid"
)

type Webhook struct {
	repo      repository.Repository
	publisher publisher.Publisher
	cache     cache.Cache
}

func NewService(repo repository.Repository, publisher publisher.Publisher, cache cache.Cache) *Webhook {
	return &Webhook{repo: repo, publisher: publisher, cache: cache}
}

func (s Webhook) getKey(ctx context.Context, eventName string) cache.Key {
	return s.cache.Key("webhook", "internal_events", eventName)
}

func (s Webhook) CreateInternalEvent(ctx context.Context, input structs.CreateInternalEventDto) (output structs.InternalEvent, err error) {
	key := s.getKey(ctx, input.EventName)
	defaultTTL := s.cache.GetDefaultTTL()

	internalEvent, err := s.repo.GetInternalEvent(ctx, input.EventName)
	if err != nil {
		return structs.InternalEvent{}, fmt.Errorf("unable to get internal event: %w", err)
	}

	if internalEvent.ID != uuid.Nil {
		return structs.InternalEvent{}, errors.New("internal event already exists")
	}

	if err := s.cache.Hydrate(ctx, key, &output, defaultTTL, func(ctx context.Context) (any, error) {
		model := input.ToInternalEvent()
		if err := s.repo.CreateInternalEvent(ctx, model); err != nil {
			return structs.InternalEvent{}, err
		}
		return model, nil
	}); err != nil {
		return structs.InternalEvent{}, fmt.Errorf("failed to create internal event: %w", err)
	}

	return output, nil
}

func (s Webhook) RegisterTrigger(ctx context.Context, input structs.RegisterTriggersDto) (structs.InternalEvent, error) {
	if input.EventName == "" {
		return structs.InternalEvent{}, errors.New("event name and triggers are required")
	}

	internalEvent, err := s.repo.GetInternalEvent(ctx, input.EventName)
	if err != nil {
		return structs.InternalEvent{}, fmt.Errorf("unable to get internal event: %w", err)
	}

	if internalEvent.ID == uuid.Nil {
		return structs.InternalEvent{}, errors.New("not found internal event")
	}

	if len(internalEvent.Triggers) > 0 {
		if internalEvent.Triggers.AlreadyExist(input.ToTrigger()) {
			return structs.InternalEvent{}, errors.New("trigger already exists")
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
			return structs.InternalEvent{}, errors.New("failed to create internal event")
		}
		return internalEvent, nil
	}); err != nil {
		return structs.InternalEvent{}, fmt.Errorf("failed to register trigger: %w", err)
	}

	return internalEvent, nil
}

func (s Webhook) PublisherExternalEvent(ctx context.Context, input structs.PublisherExternalEventDto) error {
	if err := s.publisher.Publish(ctx, task.PublisherExternalEvent.String(), input); err != nil {
		return err
	}

	return nil
}
