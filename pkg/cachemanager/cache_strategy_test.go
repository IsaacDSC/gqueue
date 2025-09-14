package cachemanager

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testPerson struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func setupRedisContainer(t *testing.T) (*redis.Client, func()) {
	ctx := context.Background()

	// Define the Redis container request
	redisPort := "6379/tcp"
	req := testcontainers.ContainerRequest{
		Image:        "redis:alpine",
		ExposedPorts: []string{redisPort},
		WaitingFor:   wait.ForListeningPort(nat.Port(redisPort)),
	}

	// Start the container
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start Redis container")

	// Get the mapped port and host
	mappedPort, err := redisContainer.MappedPort(ctx, nat.Port(redisPort))
	require.NoError(t, err, "Failed to get mapped port")
	host, err := redisContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	// Create the Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: host + ":" + mappedPort.Port(),
	})

	// Test the connection
	_, err = redisClient.Ping(ctx).Result()
	require.NoError(t, err, "Failed to ping Redis")

	// Return the client and cleanup function
	return redisClient, func() {
		if err := redisClient.Close(); err != nil {
			t.Logf("Error closing Redis client: %v", err)
		}
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("Error terminating Redis container: %v", err)
		}
	}
}

func TestKey(t *testing.T) {
	// Initialize a strategy without a real client since we're just testing key generation
	strategy := Strategy{}

	testCases := []struct {
		name     string
		params   []string
		expected Key
	}{
		{
			name:     "single parameter",
			params:   []string{"test"},
			expected: Key("test"),
		},
		{
			name:     "multiple parameters",
			params:   []string{"user", "123", "profile"},
			expected: Key("user-123-profile"),
		},
		{
			name:     "empty parameters",
			params:   []string{},
			expected: Key(""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := strategy.Key(tc.params...)
			assert.Equal(t, tc.expected, key)
		})
	}
}

func TestGetDefaultTTL(t *testing.T) {
	strategy := Strategy{}
	assert.Equal(t, 24*time.Hour, strategy.GetDefaultTTL())
}

func TestHydrate(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	strategy := NewStrategy(client)

	t.Run("successful hydration", func(t *testing.T) {
		key := Key("test-hydrate")
		expectedPerson := testPerson{Name: "John", Age: 30}

		var resultPerson testPerson

		err := strategy.Hydrate(ctx, key, &resultPerson, time.Minute, func(ctx context.Context) (any, error) {
			return expectedPerson, nil
		})

		require.NoError(t, err)
		assert.Equal(t, expectedPerson, resultPerson)

		// Verify the value was cached in Redis
		val, err := client.Get(ctx, key.String()).Bytes()
		require.NoError(t, err)

		var cachedPerson testPerson
		err = json.Unmarshal(val, &cachedPerson)
		require.NoError(t, err)
		assert.Equal(t, expectedPerson, cachedPerson)
	})

	t.Run("function error", func(t *testing.T) {
		key := Key("test-hydrate-error")
		var result testPerson

		err := strategy.Hydrate(ctx, key, &result, time.Minute, func(ctx context.Context) (any, error) {
			return nil, assert.AnError
		})

		assert.Error(t, err)

		// Verify the value was not cached in Redis
		exists, err := client.Exists(ctx, key.String()).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), exists)
	})
}

func TestOnce(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	strategy := NewStrategy(client)

	t.Run("cache miss", func(t *testing.T) {
		key := Key("test-once-miss")
		expectedPerson := testPerson{Name: "Alice", Age: 25}

		var resultPerson testPerson

		// Function should be called on cache miss
		fnCalled := false
		err := strategy.Once(ctx, key, &resultPerson, time.Minute, func(ctx context.Context) (any, error) {
			fnCalled = true
			return expectedPerson, nil
		})

		require.NoError(t, err)
		assert.True(t, fnCalled)
		assert.Equal(t, expectedPerson, resultPerson)
	})

	t.Run("cache hit", func(t *testing.T) {
		key := Key("test-once-hit")
		cachedPerson := testPerson{Name: "Bob", Age: 35}

		// Pre-cache the value
		cachedData, err := json.Marshal(cachedPerson)
		require.NoError(t, err)
		err = client.Set(ctx, key.String(), cachedData, time.Minute).Err()
		require.NoError(t, err)

		var resultPerson testPerson

		// Function should not be called on cache hit
		fnCalled := false
		err = strategy.Once(ctx, key, &resultPerson, time.Minute, func(ctx context.Context) (any, error) {
			fnCalled = true
			return testPerson{Name: "Wrong", Age: 0}, nil
		})

		require.NoError(t, err)
		assert.False(t, fnCalled)
		assert.Equal(t, cachedPerson, resultPerson)
	})

	t.Run("redis error", func(t *testing.T) {
		// Use a new client with incorrect connection to trigger Redis errors
		badClient := redis.NewClient(&redis.Options{
			Addr: "nonexistent.host:6379",
		})
		badStrategy := NewStrategy(badClient)

		key := Key("test-once-error")
		var result testPerson

		err := badStrategy.Once(ctx, key, &result, time.Minute, func(ctx context.Context) (any, error) {
			return testPerson{}, nil
		})

		assert.Error(t, err)
	})
}
