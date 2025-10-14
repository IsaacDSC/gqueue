package asynqstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/IsaacDSC/gqueue/internal/asynqtask"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/redis/go-redis/v9"
)

type Cacher interface {
	FindAllTriggers(ctx context.Context) ([]domain.Event, error)
	FindAllQueues(ctx context.Context) ([]asynqtask.Queue, error)
	FindArchivedTasks(ctx context.Context, queue string) ([]string, error)
	GetMsgArchivedTask(ctx context.Context, queue, task string) (asynqtask.RawMsg, error)
	RemoveMsgArchivedTask(ctx context.Context, queue, task string) error
	RemoveItemsArchivedTasks(ctx context.Context, queue string, tasks ...string) error
	SetArchivedTasks(ctx context.Context, events []domain.Event) error
}

type Cache struct {
	cache *redis.Client
}

func NewCache(cache *redis.Client) *Cache {
	return &Cache{cache: cache}
}

var _ Cacher = (*Cache)(nil)

const archivedKey = "gqueue:consumers:schedule:archived"

func (c Cache) FindAllTriggers(ctx context.Context) ([]domain.Event, error) {
	cResult, err := c.cache.Get(ctx, archivedKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, asynqtask.ErrorNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get consumers: %w", err)
	}

	var results []domain.Event
	if err := json.Unmarshal([]byte(cResult), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal consumers: %w", err)
	}

	return results, nil
}

func (c Cache) FindAllQueues(ctx context.Context) ([]asynqtask.Queue, error) {
	const key = "asynq:queues"
	qResult, err := c.cache.SMembers(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return nil, asynqtask.ErrorNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get queues: %w", err)
	}

	queues := make([]asynqtask.Queue, len(qResult))
	for i, q := range qResult {
		queues[i] = asynqtask.Queue(q)
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

func (c Cache) GetMsgArchivedTask(ctx context.Context, queue, taskName string) (asynqtask.RawMsg, error) {
	const archivedTaskKey = "asynq:{%s}:t:%s"
	key := fmt.Sprintf(archivedTaskKey, queue, taskName)
	result, err := c.cache.HGet(ctx, key, "msg").Result()
	if err != nil {
		return "", fmt.Errorf("failed to fetch DLQ task: %w", err)
	}

	return asynqtask.RawMsg(result), nil
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

func (c Cache) SetArchivedTasks(ctx context.Context, events []domain.Event) error {
	b, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	conf := cfg.Get()
	if err := c.cache.Set(ctx, archivedKey, b, conf.Cache.DefaultTTL).Err(); err != nil {
		return fmt.Errorf("failed to set archived tasks: %w", err)
	}

	return nil
}
