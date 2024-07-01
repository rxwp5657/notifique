package containers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/go-connections/nat"
	"github.com/notifique/internal/deployments"
	"github.com/notifique/internal/publisher"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type RabbitMQContainer struct {
	testcontainers.Container
	URI string
}

type RabbitMQPriorityContainer struct {
	Container RabbitMQContainer
	Client    publisher.RabbitMQClient
	Queues    publisher.PriorityQueues
}

func (rc *RabbitMQPriorityContainer) GetRabbitMQUrl() (string, error) {
	return rc.Container.URI, nil
}

func (rc *RabbitMQPriorityContainer) GetPriorityQueues() publisher.PriorityQueues {
	return rc.Queues
}

func NewRabbitMQContainer(ctx context.Context) (RabbitMQContainer, func(), error) {

	port := "5672"
	userName := "admin"
	password := "password"

	container, err := rabbitmq.RunContainer(ctx,
		testcontainers.WithImage("rabbitmq:3.13.3"),
		rabbitmq.WithAdminUsername(userName),
		rabbitmq.WithAdminPassword(password),
	)

	if err != nil {
		return RabbitMQContainer{}, nil, fmt.Errorf("failed to create rabbitmq container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return RabbitMQContainer{}, nil, fmt.Errorf("failed to get the rabbitmq host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return RabbitMQContainer{}, nil, err
	}

	uri := fmt.Sprintf("amqp://%s:%s/", ip, mappedPort.Port())

	close := func() {
		err := container.Terminate(ctx)

		if err != nil {
			slog.Error("failed to terminate rabbitmq container", "reason", err)
		}
	}

	rabbitmqContainer := RabbitMQContainer{
		Container: container,
		URI:       uri,
	}

	return rabbitmqContainer, close, nil
}

func NewRabbitMQPriorityContainer(ctx context.Context) (*RabbitMQPriorityContainer, func(), error) {

	queues := NewPriorityQueueConfig()
	container, close, err := NewRabbitMQContainer(ctx)

	if err != nil {
		return nil, nil, nil
	}

	pc := RabbitMQPriorityContainer{
		Container: container,
		Queues:    queues,
	}

	deployer, closeDeployer, err := deployments.NewRabbitMQPriorityDeployer(&pc)

	if err != nil {
		return nil, close, fmt.Errorf("failed to make rabbitmq deployer - %w", err)
	}

	defer closeDeployer()

	deployer.Deploy()

	return &pc, close, nil
}
