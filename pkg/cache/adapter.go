package cache

import (
	"context"
	"time"
)

type Cache interface {
	Key(params ...string) Key
	Hydrate(ctx context.Context, key Key, value any, ttl time.Duration, fn Fn) error
	Once(ctx context.Context, key Key, value any, ttl time.Duration, fn Fn) error
	GetDefaultTTL() time.Duration
}
