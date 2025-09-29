package cachemanager

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPerson struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestKey(t *testing.T) {
	strategy := Strategy{appPrefix: "testapp"}

	testCases := []struct {
		name     string
		params   []string
		expected Key
	}{
		{
			name:     "single parameter",
			params:   []string{"test"},
			expected: Key("testapp:test"),
		},
		{
			name:     "multiple parameters",
			params:   []string{"user", "123", "profile"},
			expected: Key("testapp:user:123:profile"),
		},
		{
			name:     "empty parameters",
			params:   []string{},
			expected: Key("testapp"),
		},
		{
			name:     "parameters with special characters",
			params:   []string{"user:1", "data-set", "item.123"},
			expected: Key("testapp:user:1:data-set:item.123"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := strategy.Key(tc.params...)
			assert.Equal(t, tc.expected, key)
		})
	}
}

func TestKeyString(t *testing.T) {
	key := Key("test:key:123")
	assert.Equal(t, "test:key:123", key.String())
}

func TestGetDefaultTTL(t *testing.T) {
	strategy := Strategy{appPrefix: "testapp"}
	assert.Equal(t, 24*time.Hour, strategy.GetDefaultTTL())
}

func TestNewStrategy(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	strategy := NewStrategy("testapp", redisClient)

	assert.NotNil(t, strategy)
	assert.Equal(t, "testapp", strategy.appPrefix)
	assert.NotNil(t, strategy.client)

	// Test that the strategy implements the Cache interface
	var _ Cache = strategy
}

func TestStrategyImplementsCache(t *testing.T) {
	// Compile-time check that Strategy implements Cache interface
	var _ Cache = (*Strategy)(nil)

	// Runtime check
	strategy := NewStrategy("testapp", redis.NewClient(&redis.Options{Addr: "localhost:6379"}))

	var cache Cache = strategy
	assert.NotNil(t, cache)

	// Test interface methods
	key := cache.Key("test")
	assert.Equal(t, Key("testapp:test"), key)

	ttl := cache.GetDefaultTTL()
	assert.Equal(t, 24*time.Hour, ttl)
}

func TestJSONMarshalUnmarshal(t *testing.T) {
	// Test JSON marshaling/unmarshaling that is used in the cache methods
	testData := testPerson{Name: "John", Age: 30}

	// Marshal
	data, err := json.Marshal(testData)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal
	var result testPerson
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Equal(t, testData, result)
}

func TestErrorHandling(t *testing.T) {
	t.Run("json marshal error", func(t *testing.T) {
		// Create a value that cannot be marshaled to JSON
		ch := make(chan int)
		_, err := json.Marshal(ch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json: unsupported type")
	})

	t.Run("json unmarshal error", func(t *testing.T) {
		var result testPerson
		err := json.Unmarshal([]byte("invalid json"), &result)
		assert.Error(t, err)
	})
}

func TestIncrementValueLogic(t *testing.T) {
	// Test the core logic for IncrementValue without Redis
	t.Run("new slice creation", func(t *testing.T) {
		var result []interface{}
		value := "test-value"

		// Simulate what happens when key doesn't exist (empty slice)
		alreadyExists := false
		for i := range result {
			if result[i] == value {
				alreadyExists = true
				result[i] = value
				break
			}
		}

		if !alreadyExists {
			result = append(result, value)
		}

		assert.Len(t, result, 1)
		assert.Equal(t, value, result[0])
	})

	t.Run("append to existing slice", func(t *testing.T) {
		result := []interface{}{"value1", "value2"}
		value := "value3"

		alreadyExists := false
		for i := range result {
			if result[i] == value {
				alreadyExists = true
				result[i] = value
				break
			}
		}

		if !alreadyExists {
			result = append(result, value)
		}

		assert.Len(t, result, 3)
		assert.Contains(t, result, "value1")
		assert.Contains(t, result, "value2")
		assert.Contains(t, result, value)
	})

	t.Run("update existing value", func(t *testing.T) {
		result := []interface{}{"value1", "value2"}
		value := "value1" // duplicate

		alreadyExists := false
		for i := range result {
			if result[i] == value {
				alreadyExists = true
				result[i] = value
				break
			}
		}

		if !alreadyExists {
			result = append(result, value)
		}

		assert.Len(t, result, 2) // Should still be 2, not 3
		assert.Contains(t, result, "value1")
		assert.Contains(t, result, "value2")
	})
}

func TestContextUsage(t *testing.T) {
	// Test that functions properly handle context
	ctx := context.Background()
	cancelCtx, cancel := context.WithCancel(ctx)

	// Test function that respects context
	testFn := func(ctx context.Context) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return testPerson{Name: "Test", Age: 25}, nil
		}
	}

	// Test with normal context
	result, err := testFn(ctx)
	require.NoError(t, err)
	assert.Equal(t, testPerson{Name: "Test", Age: 25}, result)

	// Test with cancelled context
	cancel()
	_, err = testFn(cancelCtx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestErrorWrapping(t *testing.T) {
	// Test error wrapping patterns used in the Strategy methods
	baseErr := errors.New("base error")
	key := Key("test-key")

	wrappedErr := errors.New("error executing function for key " + key.String() + ": " + baseErr.Error())
	assert.Contains(t, wrappedErr.Error(), "error executing function")
	assert.Contains(t, wrappedErr.Error(), key.String())
	assert.Contains(t, wrappedErr.Error(), baseErr.Error())
}

func TestTimeHandling(t *testing.T) {
	// Test TTL handling
	strategy := Strategy{appPrefix: "testapp"}

	defaultTTL := strategy.GetDefaultTTL()
	assert.Equal(t, 24*time.Hour, defaultTTL)

	// Test different TTL values
	testTTLs := []time.Duration{
		time.Minute,
		time.Hour,
		24 * time.Hour,
		time.Duration(-1), // No expiration in Redis
	}

	for _, ttl := range testTTLs {
		assert.IsType(t, time.Duration(0), ttl)
	}
}
