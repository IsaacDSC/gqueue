package storests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	insightsPrefix = "gqueue:insights"
	separator      = ":"
)

type Store struct {
	cache *redis.Client
}

func NewStore(cache *redis.Client) *Store {
	return &Store{cache: cache}
}

func (s *Store) Published(ctx context.Context, input domain.PublisherInsights) error {
	var isSuccess string
	if input.ACK {
		isSuccess = "success"
	} else {
		isSuccess = "failure"
	}

	key := s.key("event-published", isSuccess, input.TopicName)
	defer s.groupInsights(ctx, key)

	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	now := time.Now().UTC().UnixMilli()
	if err := s.cache.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: payload}).Err(); err != nil {
		return fmt.Errorf("failed to save publisher event: %w", err)
	}

	return nil
}

func (s *Store) Consumed(ctx context.Context, input domain.ConsumerInsights) error {
	var isSuccessPrefix string
	if input.ACK {
		isSuccessPrefix = "success"
	} else {
		isSuccessPrefix = "failure"
	}

	key := s.key("event-consumed", isSuccessPrefix, input.TopicName, input.ConsumerName)
	defer s.groupInsights(ctx, key)

	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	now := time.Now().UTC().UnixMilli()
	if err := s.cache.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: payload}).Err(); err != nil {
		return fmt.Errorf("failed to save consumed event: %w", err)
	}

	return nil
}

func (s *Store) key(typeEvent string, values ...string) string {
	v := []string{insightsPrefix, typeEvent}
	v = append(v, values...)
	return strings.Join(v, separator)
}

func (s *Store) groupInsights(ctx context.Context, key string) error {
	return s.cache.LPush(ctx, key, time.Now().UnixMilli()).Err()
}
