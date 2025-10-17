package cachemanager

import (
	"context"
	"time"
)

type Cache interface {
	Key(params ...string) Key
	Hydrate(ctx context.Context, key Key, value any, ttl time.Duration, fn Fn) error
	Once(ctx context.Context, key Key, value any, ttl time.Duration, fn Fn) error
	GetDefaultTTL() time.Duration
	IncrementValue(ctx context.Context, key Key, value any) error
	RemoveValue(ctx context.Context, key Key, fn func(ctx context.Context) error) error
}
