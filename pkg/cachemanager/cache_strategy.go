package cachemanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	redis "github.com/redis/go-redis/v9"
)

// Fn defines a function type that takes a context and returns any value and an error.
type Fn func(ctx context.Context) (any, error)

// Key represents a cache key as a string.
type Key string

// String returns the string representation of the Key.
func (k Key) String() string {
	return string(k)
}

// Strategy provides caching methods using a Redis client.
type Strategy struct {
	appPrefix string
	client    *redis.Client
}

var _ Cache = (*Strategy)(nil)

// NewStrategy creates a new Strategy with the given Redis client.
func NewStrategy(appPrefix string, client *redis.Client) *Strategy {
	return &Strategy{appPrefix: appPrefix, client: client}
}

// Key constructs a cache key by joining the provided parameters with a separator.
func (s Strategy) Key(params ...string) Key {
	params = append([]string{s.appPrefix}, params...)
	return Key(strings.Join(params, ":"))
}

// GetDefaultTTL for cache entries, can be adjusted as needed.
func (s Strategy) GetDefaultTTL() time.Duration {
	conf := cfg.Get()
	return conf.Cache.DefaultTTL
}

// Hydrate executes the provided function, stores its result in the cache, and unmarshals it into value.
// It always refreshes the cache with the latest value.
func (s Strategy) Hydrate(ctx context.Context, key Key, value any, ttl time.Duration, fn Fn) error {
	v, err := fn(ctx)
	if err != nil {
		return fmt.Errorf("error executing function for key %s: %w", key.String(), err)
	}

	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("error marshalling value for key %s: %w", key.String(), err)
	}

	if err := s.client.Set(ctx, key.String(), b, ttl).Err(); err != nil {
		return fmt.Errorf("error setting value for key %s: %w", key.String(), err)
	}

	if err := json.Unmarshal(b, value); err != nil {
		return fmt.Errorf("error unmarshalling value for key %s: %w", key.String(), err)
	}

	return nil
}

// Once retrieves the value from the cache if present, otherwise executes the function, stores, and returns the result.
// It only executes the function if the cache is missing.
func (s Strategy) Once(ctx context.Context, key Key, value any, ttl time.Duration, fn Fn) error {
	exist, err := s.client.Exists(ctx, key.String()).Result()
	if err != nil {
		return fmt.Errorf("error checking existence of key %s: %w", key.String(), err)
	}

	if exist > 0 {
		v, err := s.client.Get(ctx, key.String()).Bytes()
		if err != nil {
			return fmt.Errorf("error getting value for key %s: %w", key.String(), err)
		}

		if err := json.Unmarshal(v, value); err != nil {
			return fmt.Errorf("error unmarshalling value for key %s: %w", key.String(), err)
		}

		return nil
	}

	res, err := fn(ctx)
	if err != nil {
		return fmt.Errorf("error executing function for key %s: %w", key.String(), err)
	}

	valueBytes, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("error marshalling value for key %s: %w", key.String(), err)
	}

	if err := s.client.Set(ctx, key.String(), valueBytes, ttl).Err(); err != nil {
		return fmt.Errorf("error setting value for key %s: %w", key.String(), err)
	}

	if err := json.Unmarshal(valueBytes, value); err != nil {
		return fmt.Errorf("error unmarshalling value after setting key %s: %w", key.String(), err)
	}

	return nil
}

func (s Strategy) IncrementValue(ctx context.Context, key Key, value any) error {
	val, err := s.client.Get(ctx, key.String()).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("error getting value for key %s: %w", key.String(), err)
	}

	var result []any
	if !errors.Is(err, redis.Nil) {
		json.Unmarshal([]byte(val), &result)
	}

	allreadyExists := false
	for i := range result {
		if result[i] == value {
			allreadyExists = true
			result[i] = value
			break
		}
	}

	if !allreadyExists {
		result = append(result, value)
	}

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error marshalling value for key %s: %w", key.String(), err)
	}

	conf := cfg.Get()
	if err := s.client.Set(ctx, key.String(), b, conf.Cache.DefaultTTL).Err(); err != nil {
		return fmt.Errorf("error setting value for key %s: %w", key.String(), err)
	}

	return nil
}
