package taskstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/task"
	"github.com/redis/go-redis/v9"
)

type Cacher interface {
	FindAllConsumers(ctx context.Context) (task.QueueConsumers, error)
	FindAllQueues(ctx context.Context) (task.Queues, error)
	FindArchivedTasks(ctx context.Context, queue string) ([]string, error)
	GetMsgArchivedTask(ctx context.Context, queue, task string) (task.RawMsg, error)
	RemoveMsgArchivedTask(ctx context.Context, queue, task string) error
	RemoveItemsArchivedTasks(ctx context.Context, queue string, tasks ...string) error
}

type Cache struct {
	cache *redis.Client
}

func NewCache(cache *redis.Client) *Cache {
	return &Cache{cache: cache}
}

func (c Cache) FindAllConsumers(ctx context.Context) (task.QueueConsumers, error) {
	cResult, err := c.cache.Get(ctx, "gqueue:consumers:schedule:archived").Result()
	if errors.Is(err, redis.Nil) {
		return nil, task.ErrorNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get consumers: %w", err)
	}

	var results []domain.Event
	if err := json.Unmarshal([]byte(cResult), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal consumers: %w", err)
	}

	var queueOnConsumers task.QueueConsumers
	// TODO: implementation

	return queueOnConsumers, nil
}

func (c Cache) FindAllQueues(ctx context.Context) (task.Queues, error) {
	const key = "asynq:queues"
	qResult, err := c.cache.SMembers(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return nil, task.ErrorNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get queues: %w", err)
	}

	queues := make(task.Queues, len(qResult))
	for i, q := range qResult {
		queues[i] = task.Queue(q)
	}

	return queues, nil

}

func (c Cache) FindArchivedTasks(ctx context.Context, queue string) ([]string, error) {
	const zrangeKey = "asynq:{%s}:archived"
	result, err := c.cache.ZRange(ctx, fmt.Sprintf(zrangeKey, queue), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch archived tasks: %w", err)
	}

	return result, nil
}

func (c Cache) RemoveItemsArchivedTasks(ctx context.Context, queue string, tasks ...string) error {
	const zrangeKey = "asynq:{%s}:archived"
	if err := c.cache.ZRem(ctx, fmt.Sprintf(zrangeKey, queue), tasks).Err(); err != nil {
		return fmt.Errorf("failed to remove archived tasks from queue %s: %w", queue, err)
	}

	return nil
}

func (c Cache) GetMsgArchivedTask(ctx context.Context, queue, taskName string) (task.RawMsg, error) {
	const archivedTaskKey = "asynq:{%s}:t:%s"
	key := fmt.Sprintf(archivedTaskKey, queue, taskName)
	result, err := c.cache.HGet(ctx, key, "msg").Result()
	if err != nil {
		return "", fmt.Errorf("failed to fetch DLQ task: %w", err)
	}

	return task.RawMsg(result), nil
}

func (c Cache) RemoveMsgArchivedTask(ctx context.Context, queue, task string) error {
	const archivedTaskKey = "asynq:{%s}:t:%s"
	key := fmt.Sprintf(archivedTaskKey, queue, task)
	if err := c.Remove(ctx, key); err != nil {
		return fmt.Errorf("failed to remove archived task %s from queue %s: %w", task, queue, err)
	}

	return nil
}

func (c Cache) Remove(ctx context.Context, key string) error {
	if err := c.cache.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove key %s: %w", key, err)
	}

	return nil
}
