package asynqsvc

import (
	"context"
	"github.com/hibiken/asynq"
)

type AsynqHandle struct {
	Event   string
	Handler func(ctx context.Context, task *asynq.Task) error
}
