package task

import (
	"context"
	"github.com/IsaacDSC/webhook/internal/infra/gateway"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/hibiken/asynq"
)

type TaskName string

func (tn TaskName) String() string {
	return string(tn)
}

const (
	PublisherExternalEvent TaskName = "publisher_external_event"
)

type Tasks struct {
	repo *repository.MongoRepo
	gate *gateway.Hook
}

func NewTasks(repo *repository.MongoRepo, gate *gateway.Hook) Tasks {
	return Tasks{repo: repo, gate: gate}
}

func (t Tasks) GetTasks() map[TaskName]func(context.Context, *asynq.Task) error {
	return map[TaskName]func(context.Context, *asynq.Task) error{
		PublisherExternalEvent: t.publisherExternalEvent,
	}
}
