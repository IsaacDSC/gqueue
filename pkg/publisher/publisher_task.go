package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"log"
	"time"
)

type Task struct {
	client *asynq.Client
}

func NewPublisher(client *asynq.Client) *Task {
	return &Task{client: client}
}

func (t *Task) Publish(ctx context.Context, eventName string, payload any, opts ...asynq.Option) error {
	defaultOpts := NewDefaultOpt()
	definedOpts := make([]asynq.Option, 0, len(defaultOpts)+len(opts))
	definedOpts = append(definedOpts, defaultOpts...)
	definedOpts = append(definedOpts, opts...)

	p, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %v", err)
	}

	task := asynq.NewTask(eventName, p)
	info, err := t.client.Enqueue(task, opts...)
	if err != nil {
		return fmt.Errorf("could not schedule task: %v", err)
	}

	log.Printf("enqueued task: id=%s queue=%s", info.ID, info.Queue)

	return nil
}

func WithQueue(queue string) asynq.Option {
	return asynq.Queue(queue)
}

func WithMaxRetry(maxRetry int) asynq.Option {
	return asynq.MaxRetry(maxRetry)
}

func WithRetention(retention time.Duration) asynq.Option {
	return asynq.Retention(retention)
}

func WithProcessIn(processIn time.Duration) asynq.Option {
	return asynq.ProcessIn(processIn)
}

func NewDefaultOpt() []asynq.Option {
	return []asynq.Option{
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Retention(168 * time.Hour), // 7 days
	}
}
