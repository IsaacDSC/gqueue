package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/infra/task"
	"github.com/IsaacDSC/webhook/internal/structs"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/google/uuid"
)

type Webhook struct {
	repo      repository.Repository
	publisher publisher.Publisher
}

func NewService(repo repository.Repository, publisher publisher.Publisher) *Webhook {
	return &Webhook{repo: repo, publisher: publisher}
}

func (s Webhook) CreateInternalEvent(ctx context.Context, input structs.CreateInternalEventDto) (structs.InternalEvent, error) {
	model := input.ToInternalEvent()
	if err := s.repo.CreateInternalEvent(ctx, model); err != nil {
		return structs.InternalEvent{}, err
	}

	return model, nil
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
		alreadyExists := true
		for _, trigger := range internalEvent.Triggers {
			if trigger.ServiceName != input.Trigger.ServiceName {
				alreadyExists = false
				internalEvent.Triggers = append(internalEvent.Triggers, trigger)
			}
		}

		if alreadyExists {
			return structs.InternalEvent{}, errors.New("trigger already exists")
		}
	} else {
		internalEvent.Triggers = append(internalEvent.Triggers, input.ToTrigger())
	}

	if err := s.repo.SaveInternalEvent(ctx, internalEvent); err != nil {
		return structs.InternalEvent{}, errors.New("failed to create internal event")
	}

	return internalEvent, nil
}

func (s Webhook) PublisherExternalEvent(ctx context.Context, input structs.PublisherExternalEventDto) error {
	if err := s.publisher.Publish(ctx, task.PublisherExternalEvent.String(), input); err != nil {
		return err
	}

	return nil
}
