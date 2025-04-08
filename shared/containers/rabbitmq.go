package containers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/go-connections/nat"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type RabbitMQ struct {
	testcontainers.Container
	URI string
}

func (rc *RabbitMQ) GetRabbitMQUrl() (string, error) {
	return rc.URI, nil
}

func NewRabbitMQContainer(ctx context.Context) (RabbitMQ, func(), error) {

	port := "5672"
	userName := "admin"
	password := "password"

	container, err := rabbitmq.RunContainer(ctx,
		testcontainers.WithImage("rabbitmq:3.13.3"),
		rabbitmq.WithAdminUsername(userName),
		rabbitmq.WithAdminPassword(password),
	)

	if err != nil {
		return RabbitMQ{}, nil, fmt.Errorf("failed to create rabbitmq container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return RabbitMQ{}, nil, fmt.Errorf("failed to get the rabbitmq host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return RabbitMQ{}, nil, err
	}

	uri := fmt.Sprintf("amqp://%s:%s/", ip, mappedPort.Port())

	close := func() {
		err := container.Terminate(ctx)

		if err != nil {
			slog.Error("failed to terminate rabbitmq container", "reason", err)
		}
	}

	rabbitmqContainer := RabbitMQ{
		Container: container,
		URI:       uri,
	}

	return rabbitmqContainer, close, nil
}
