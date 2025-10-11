package storests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IsaacDSC/gqueue/internal/domain"
)

func (s *Store) GetAll(ctx context.Context) (output domain.Metrics, err error) {
	k := s.key("group-insights")
	insightsKeys, err := s.cache.SMembers(ctx, k).Result()
	if err != nil {
		err = fmt.Errorf("failed to get insights keys: %w", err)
		return
	}

	for _, key := range insightsKeys {
		values, er := s.cache.ZRange(ctx, key, 0, -1).Result()
		if er != nil {
			err = fmt.Errorf("failed to get insights values for key %s: %w", key, err)
			return
		}

		for _, v := range values {
			var insight domain.Metric
			if err = json.Unmarshal([]byte(v), &insight); err != nil {
				err = fmt.Errorf("failed to unmarshal insights for key %s: %w", key, err)
				return
			}

			output = append(output, insight)

		}

	}

	return
}
