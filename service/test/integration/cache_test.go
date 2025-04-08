package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/notifique/shared/cache"
	"github.com/notifique/shared/containers"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	ctx := context.Background()

	redis, closer, err := containers.NewRedisContainer(ctx)

	if err != nil {
		t.Fatal(err)
		return
	}

	defer closer()

	redisClient, err := cache.NewRedisClient(redis)

	if err != nil {
		t.Fatal(err)
		return
	}

	redisCache, err := cache.NewRedisCache(redisClient)

	if err != nil {
		t.Fatal(err)
		return
	}

	t.Run("Can set/get value", func(t *testing.T) {
		key := cache.Key("test-key")
		value := "test-value"

		err := redisCache.Set(ctx, key, value, time.Hour)
		assert.Nil(t, err)

		got, err, exists := redisCache.Get(ctx, key)
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, value, got)
	})

	t.Run("Get non-existent key returns no error", func(t *testing.T) {
		key := cache.Key("non-existent-key")

		got, err, exists := redisCache.Get(ctx, key)
		assert.Nil(t, err)
		assert.False(t, exists)
		assert.Equal(t, "", got)
	})

	t.Run("Can delete value", func(t *testing.T) {
		key := cache.Key("test-key-delete")
		value := "test-value"

		err := redisCache.Set(ctx, key, value, time.Hour)
		assert.Nil(t, err)

		err = redisCache.Del(ctx, key)
		assert.Nil(t, err)

		_, err, exists := redisCache.Get(ctx, key)
		assert.Nil(t, err)
		assert.False(t, exists)
	})

	t.Run("Can delete values with prefix", func(t *testing.T) {
		prefix := cache.Key("test-prefix")
		key1 := cache.Key("test-prefix:key1")
		key2 := cache.Key("test-prefix:key2")
		otherKey := cache.Key("other-key")

		// Set multiple keys
		err := redisCache.Set(ctx, key1, "value1", time.Hour)
		assert.Nil(t, err)
		err = redisCache.Set(ctx, key2, "value2", time.Hour)
		assert.Nil(t, err)
		err = redisCache.Set(ctx, otherKey, "other", time.Hour)
		assert.Nil(t, err)

		// Delete keys with prefix
		err = redisCache.DelWithPrefix(ctx, prefix)
		assert.Nil(t, err)

		// Verify prefixed keys are deleted
		_, err, exists := redisCache.Get(ctx, key1)
		assert.Nil(t, err)
		assert.False(t, exists)

		_, err, exists = redisCache.Get(ctx, key2)
		assert.Nil(t, err)
		assert.False(t, exists)

		// Verify other key still exists
		value, err, exists := redisCache.Get(ctx, otherKey)
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, "other", value)
	})
}
