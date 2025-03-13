package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/notifique/internal/cache"
	"github.com/notifique/internal/controllers"
	"github.com/notifique/internal/dto"
	"github.com/notifique/internal/testutils"
	"github.com/notifique/internal/testutils/containers"
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

	notificationId := uuid.NewString()

	t.Run("Can set/retrieve the notification status", func(t *testing.T) {

		status := testutils.StatusPtr(dto.Created)

		err := redisCache.UpdateNotificationStatus(ctx, controllers.NotificationStatusLog{
			NotificationId: notificationId,
			Status:         *status,
		})

		assert.Nil(t, err)

		status, err = redisCache.GetNotificationStatus(ctx, notificationId)

		assert.Nil(t, err)
		assert.Equal(t, dto.Created, *status)
	})

	t.Run("Returns false when notification hash does not exist", func(t *testing.T) {
		hash := uuid.NewString()
		exists, err := redisCache.NotificationExists(ctx, hash)

		assert.Nil(t, err)
		assert.False(t, exists)
	})

	t.Run("Can set and retrieve notification hash", func(t *testing.T) {
		hash := uuid.NewString()

		err := redisCache.SetNotificationHash(ctx, hash)
		assert.Nil(t, err)

		exists, err := redisCache.NotificationExists(ctx, hash)
		assert.Nil(t, err)
		assert.True(t, exists)
	})

	t.Run("Can delete notification hash", func(t *testing.T) {
		hash := uuid.NewString()

		err := redisCache.SetNotificationHash(ctx, hash)
		assert.Nil(t, err)

		err = redisCache.DeleteNotificationHash(ctx, hash)
		assert.Nil(t, err)

		exists, err := redisCache.NotificationExists(ctx, hash)
		assert.Nil(t, err)
		assert.False(t, exists)
	})

	t.Run("No error when deleting non-existent notification hash", func(t *testing.T) {
		hash := uuid.NewString()
		err := redisCache.DeleteNotificationHash(ctx, hash)
		assert.Nil(t, err)
	})
}
