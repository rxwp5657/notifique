package cache

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	c "github.com/notifique/internal/controllers"
	dto "github.com/notifique/internal/dto"
)

type RedisConfigurator interface {
	GetRedisUrl() (string, error)
}

type CacheRedisApi interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type RedisCache struct {
	client CacheRedisApi
}

func getNotificationStatusKey(notificationId string) string {
	return fmt.Sprintf("notifications:%s:status", notificationId)
}

func getHashKey(hash string) string {
	return fmt.Sprintf("notifications:hash:%s", hash)
}

func (rc *RedisCache) GetNotificationStatus(ctx context.Context, notificationId string) (*dto.NotificationStatus, error) {
	status, err := rc.client.Get(ctx, getNotificationStatusKey(notificationId)).Result()

	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve notification status - %w", err)
	}

	return (*dto.NotificationStatus)(&status), nil
}

func (rc *RedisCache) UpdateNotificationStatus(ctx context.Context, statusLog c.NotificationStatusLog) error {

	key := getNotificationStatusKey(statusLog.NotificationId)

	_, err := rc.client.
		Set(ctx, key, string(statusLog.Status), time.Duration(1*time.Hour)).
		Result()

	if err != nil {
		return fmt.Errorf("failed to set notification status - %w", err)
	}

	return nil
}

func (rc *RedisCache) NotificationExists(ctx context.Context, hash string) (bool, error) {

	_, err := rc.client.Get(ctx, getHashKey(hash)).Result()

	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to retrieve notification hash - %w", err)
	}

	return true, nil
}

func (rc *RedisCache) SetNotificationHash(ctx context.Context, hash string) error {

	key := getHashKey(hash)

	_, err := rc.client.
		Set(ctx, key, "1", time.Duration(5*time.Minute)).
		Result()

	if err != nil {
		return fmt.Errorf("failed to set notification hash - %w", err)
	}

	return nil
}

func (rc *RedisCache) DeleteNotificationHash(ctx context.Context, hash string) error {

	key := getHashKey(hash)

	_, err := rc.client.Del(ctx, key).Result()

	if err == redis.Nil {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete the notification hash - %w", err)
	}

	return nil
}

func NewRedisClient(c RedisConfigurator) (*redis.Client, error) {
	url, err := c.GetRedisUrl()

	if err != nil {
		return nil, err
	}

	opt, err := redis.ParseURL(url)

	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url - %w", err)
	}

	return redis.NewClient(opt), nil
}

func NewRedisCache(client CacheRedisApi) (*RedisCache, error) {
	cache := RedisCache{client: client}
	return &cache, nil
}
