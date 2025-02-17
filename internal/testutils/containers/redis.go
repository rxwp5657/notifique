package containers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const TestBrokerChannelSize = 10

type RedisContainer struct {
	testcontainers.Container
	URI string
}

func (rc *RedisContainer) GetRedisUrl() (string, error) {

	if rc == nil {
		return "", fmt.Errorf("redis container is null")
	}

	return rc.URI, nil
}

func (rc *RedisContainer) GetBrokerChannelSize() (int, error) {

	if rc == nil {
		return 0, fmt.Errorf("redis container is null")
	}

	return TestBrokerChannelSize, nil
}

func NewRedisContainer(ctx context.Context) (*RedisContainer, func(), error) {

	port := "6379"

	req := testcontainers.ContainerRequest{
		Image:      "redis:7.4-rc1-alpine",
		WaitingFor: wait.ForExposedPort(),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create redis container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the redis host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return nil, nil, err
	}

	uri := fmt.Sprintf("redis://%s:%s", ip, mappedPort.Port())

	close := func() {
		err := container.Terminate(ctx)

		if err != nil {
			slog.Error("failed to terminate redis container", "reason", err)
		}
	}

	redisContainer := RedisContainer{
		Container: container,
		URI:       uri,
	}

	return &redisContainer, close, nil
}
