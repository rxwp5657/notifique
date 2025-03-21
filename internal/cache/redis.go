package cache

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/notifique/internal"
	redis "github.com/redis/go-redis/v9"
)

const endpointKey string = "notifications:endpoint"

type Key string

type RedisConfigurator interface {
	GetRedisUrl() (string, error)
}

type CacheRedisApi interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
}

type Cache interface {
	Set(ctx context.Context, k Key, value string, ttl time.Duration) error
	Get(ctx context.Context, k Key) (string, error, bool)
	Del(ctx context.Context, k Key) error
	DelWithPrefix(ctx context.Context, prefix Key) error
}

type RedisCache struct {
	client CacheRedisApi
}

func GetNotificationStatusKey(notificationId string) Key {
	return Key(fmt.Sprintf("notifications:%s:status", notificationId))
}

func GetHashKey(hash string) Key {
	return Key(fmt.Sprintf("notifications:hash:%s", hash))
}

func prepareEndpointPath(path string, userId *string) string {

	if userId == nil {
		empty := ""
		userId = &empty
	}

	return strings.Replace(path, "me", *userId, 1)
}

func GetEntpointKey(url *url.URL, userId *string) Key {

	path := prepareEndpointPath(url.Path, userId)
	key := fmt.Sprintf("%s:%s", internal.GetMd5Hash(url.Path), path)

	if url.RawQuery != "" {
		key = fmt.Sprintf("%s?%s", key, url.RawQuery)
	}

	return Key(fmt.Sprintf("%s:%s", endpointKey, key))
}

func GetEndpointKeyWithPrefix(path string, userId *string) Key {

	path = prepareEndpointPath(path, userId)
	path = fmt.Sprintf("%s*", path)

	key := fmt.Sprintf("%s:%s", internal.GetMd5Hash(path), path)

	return Key(fmt.Sprintf("%s:%s", endpointKey, key))
}

func (rc *RedisCache) Set(ctx context.Context, k Key, value string, ttl time.Duration) error {

	_, err := rc.client.
		Set(context.Background(), string(k), value, ttl).
		Result()

	if err != nil {
		return fmt.Errorf("failed to set key - %w", err)
	}

	return nil
}

func (rc *RedisCache) Get(ctx context.Context, k Key) (string, error, bool) {

	data, err := rc.client.Get(ctx, string(k)).Result()

	if err == redis.Nil {
		return "", nil, false
	} else if err != nil {
		return "", fmt.Errorf("failed to get key - %w", err), false
	}

	return data, nil, true
}

func (rc *RedisCache) Del(ctx context.Context, k Key) error {

	_, err := rc.client.Del(ctx, string(k)).Result()

	if err == redis.Nil {
		return nil
	}

	return err
}

func (rc *RedisCache) DelWithPrefix(ctx context.Context, prefix Key) error {
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = rc.client.Scan(ctx, cursor, string(prefix)+"*", 100).Result()

		if err != nil {
			return fmt.Errorf("failed to scan keys - %w", err)
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}

	_, err := rc.client.Del(ctx, keys...).Result()

	if err != nil {
		return fmt.Errorf("failed to delete keys - %w", err)
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
