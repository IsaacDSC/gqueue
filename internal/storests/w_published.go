package storests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/redis/go-redis/v9"
)

func (s *Store) Published(ctx context.Context, input domain.PublisherMetric) error {
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

	conf := cfg.Get()
	if err := s.cache.Expire(ctx, key, conf.Cache.DefaultTTL).Err(); err != nil {
		return fmt.Errorf("failed to set TTL for publisher event: %w", err)
	}

	return nil
}
