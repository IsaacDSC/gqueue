package storests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/redis/go-redis/v9"
)

func (s *Store) Consumed(ctx context.Context, input domain.ConsumerMetric) error {
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
