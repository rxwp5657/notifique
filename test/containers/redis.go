package containers

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

const TestBrokerChannelSize = 10

type RedisContainer struct {
	testcontainers.Container
	URI     string
	Cleanup func() error
}

func (rc *RedisContainer) GetRedisUrl() (string, error) {
	return rc.URI, nil
}

func (rc *RedisContainer) GetBrokerChannelSize() (int, error) {
	return TestBrokerChannelSize, nil
}

func NewRedisContainer(ctx context.Context) (RedisContainer, error) {

	port := "4566"

	container, err := redis.RunContainer(
		ctx,
		testcontainers.WithImage("redis:7.4-rc1-alpine"),
	)

	if err != nil {
		return RedisContainer{}, fmt.Errorf("failed to create redis container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return RedisContainer{}, fmt.Errorf("failed to get the redis host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return RedisContainer{}, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	cleanup := func() error { return container.Terminate(ctx) }

	redisContainer := RedisContainer{
		Container: container,
		URI:       uri,
		Cleanup:   cleanup,
	}

	return redisContainer, nil
}
