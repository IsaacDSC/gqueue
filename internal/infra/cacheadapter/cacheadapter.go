package cacheadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/structs"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type InternalEvent struct {
	client     *redis.Client
	repository repository.Repository
}

func NewInternalEventAdapter(client *redis.Client, repository repository.Repository) *InternalEvent {
	return &InternalEvent{client: client, repository: repository}
}

func (ca InternalEvent) Set(ctx context.Context, key string, internalEvent structs.InternalEvent) error {
	b, err := json.Marshal(internalEvent)
	if err != nil {
		return fmt.Errorf("set error on marshal: %w", err)
	}

	if err := ca.client.Set(ctx, key, b, ca.GetDefaultTTL()).Err(); err != nil {
		return fmt.Errorf("set error on set cache: %w", err)
	}

	if err := ca.repository.SaveInternalEvent(ctx, internalEvent); err != nil {
		return fmt.Errorf("set error on save to repository: %w", err)
	}

	return nil
}

func (ca InternalEvent) Get(ctx context.Context, key string, eventName string) (output structs.InternalEvent, err error) {
	b, err := ca.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			output, err = ca.repository.GetInternalEvent(ctx, eventName)
			if err != nil {
				return output, fmt.Errorf("get error on repository: %w", err)
			}
			if err := ca.client.Set(ctx, key, b, ca.GetDefaultTTL()).Err(); err != nil {
				return output, fmt.Errorf("get error on set cache: %w", err)
			}
		}
		return output, fmt.Errorf("get error on cache: %w", err)
	}

	if err := json.Unmarshal(b, &output); err != nil {
		return output, fmt.Errorf("get error on unmarshal: %w", err)
	}

	return
}

func (ca InternalEvent) GetDefaultTTL() time.Duration {
	return time.Hour * 24
}

func (ca InternalEvent) Key(params ...string) string {
	return strings.Join(params, ":")
}
