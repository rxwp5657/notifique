package cache

import (
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

type RedisConfigurator interface {
	GetRedisUrl() (string, error)
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
