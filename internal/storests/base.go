package storests

import (
	"context"
	"strconv"
	"strings"
	"time"

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

func (s *Store) key(typeEvent string, values ...string) string {
	v := []string{insightsPrefix, typeEvent}
	v = append(v, values...)
	v = append(v, strconv.Itoa(time.Now().UTC().Day()))
	return strings.Join(v, separator)
}

func (s *Store) groupInsights(ctx context.Context, key string) error {
	k := s.key("group-insights")
	return s.cache.SAdd(ctx, k, key).Err()
}
