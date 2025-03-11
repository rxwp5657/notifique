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
}
