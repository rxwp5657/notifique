package containers

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/notifique/internal/deployments"
	"github.com/notifique/internal/publisher"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type RabbitMQContainer struct {
	testcontainers.Container
	URI     string
	Cleanup func() error
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

func MakeRabbitMQContainer(ctx context.Context) (RabbitMQContainer, error) {

	port := "5672"
	userName := "admin"
	password := "password"

	container, err := rabbitmq.RunContainer(ctx,
		testcontainers.WithImage("rabbitmq:3.13.3"),
		rabbitmq.WithAdminUsername(userName),
		rabbitmq.WithAdminPassword(password),
	)

	if err != nil {
		return RabbitMQContainer{}, fmt.Errorf("failed to create rabbitmq container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return RabbitMQContainer{}, fmt.Errorf("failed to get the rabbitmq host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return RabbitMQContainer{}, err
	}

	uri := fmt.Sprintf("amqp://%s:%s/", ip, mappedPort.Port())

	cleanup := func() error { return container.Terminate(ctx) }

	rabbitmqContainer := RabbitMQContainer{
		Container: container,
		URI:       uri,
		Cleanup:   cleanup,
	}

	return rabbitmqContainer, nil
}

func MakeRabbitMQPriorityContainer(ctx context.Context) (*RabbitMQPriorityContainer, error) {

	queues := MakePriorityQueueConfig()
	container, err := MakeRabbitMQContainer(ctx)

	if err != nil {
		return nil, nil
	}

	pc := RabbitMQPriorityContainer{
		Container: container,
		Queues:    queues,
	}

	deployer, cleanup, err := deployments.MakeRabbitMQPriorityDeployer(&pc)

	if err != nil {
		return nil, fmt.Errorf("failed to make rabbitmq deployer - %w", err)
	}

	defer cleanup()

	deployer.Deploy()

	return &pc, nil
}
