package containers

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	deployments "github.com/notifique/deployments/rabbitmq"
	"github.com/notifique/internal/publisher"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type RabbitMQContainer struct {
	testcontainers.Container
	URI       string
	Terminate func() error
}

type RabbitMQDeployer interface {
	Deploy(container RabbitMQContainer) error
}

type RabbitMQPriorityDeployer struct {
	Queues publisher.PriorityQueues
}

func (d RabbitMQPriorityDeployer) Deploy(container RabbitMQContainer) error {

	client, err := MakeRabbitMQClient(&container)

	if err != nil {
		return err
	}

	err = deployments.MakeRabbitMQPriorityQueues(*client, d.Queues)

	if err != nil {
		return fmt.Errorf("failed to deploy priority queues - %w", err)
	}

	return nil
}

func MakeRabbitMQContainer(ctx context.Context, deployer RabbitMQDeployer) (*RabbitMQContainer, error) {

	port := "5672"
	userName := "admin"
	password := "password"

	container, err := rabbitmq.RunContainer(ctx,
		testcontainers.WithImage("rabbitmq:3.13.3"),
		rabbitmq.WithAdminUsername(userName),
		rabbitmq.WithAdminPassword(password),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create rabbitmq container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get the rabbitmq host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("amqp://%s:%s/", ip, mappedPort.Port())

	terminate := func() error { return container.Terminate(ctx) }

	rabbitmqContainer := RabbitMQContainer{
		Container: container,
		URI:       uri,
		Terminate: terminate,
	}

	err = deployer.Deploy(rabbitmqContainer)

	if err != nil {
		return nil, fmt.Errorf("failed to deploy queues - %w", err)
	}

	return &rabbitmqContainer, nil
}

func MakeRabbitMQPriorityDeployer() *RabbitMQPriorityDeployer {
	deployer := RabbitMQPriorityDeployer{Queues: MakePriorityQueueConfig()}
	return &deployer
}

func MakeRabbitMQClient(container *RabbitMQContainer) (*publisher.RabbitMQClient, error) {

	url := publisher.RabbitMQURL(container.URI)
	client, err := publisher.MakeRabbitMQClient(url)

	if err != nil {
		return nil, fmt.Errorf("failed to create rabbitmq client - %w", err)
	}

	return &client, err
}

func MakeRabbitMQPriorityPub(client *publisher.RabbitMQClient, deployer *RabbitMQPriorityDeployer) (*publisher.RabbitMQPriorityPublisher, error) {

	cfg := publisher.RabbitMQPriorityPublisherConfg{
		Publisher: client,
		Queues:    deployer.Queues,
	}

	pub := publisher.MakeRabbitMQPriorityPub(cfg)

	return &pub, nil
}
