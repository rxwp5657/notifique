package containers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/go-connections/nat"
	"github.com/notifique/internal/deployments"
	"github.com/notifique/internal/publisher"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

type SQSContainer struct {
	testcontainers.Container
	URI string
}

type SQSPriorityContainer struct {
	Container SQSContainer
	Queues    publisher.PriorityQueues
}

func (sc *SQSPriorityContainer) GetSQSClientConfig() publisher.SQSClientConfig {
	return publisher.SQSClientConfig{
		BaseEndpoint: &sc.Container.URI,
	}
}

func (sc *SQSPriorityContainer) GetPriorityQueues() publisher.PriorityQueues {
	return sc.Queues
}

func NewSQSContainer(ctx context.Context) (SQSContainer, func(), error) {

	port := "4566"

	container, err := localstack.RunContainer(
		ctx,
		testcontainers.WithImage("localstack/localstack:3.4"),
		testcontainers.WithEnv(map[string]string{
			"SERVICES": "sqs",
		}),
	)

	if err != nil {
		return SQSContainer{}, nil, fmt.Errorf("failed to create sqs container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return SQSContainer{}, nil, fmt.Errorf("failed to get the sqs host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return SQSContainer{}, nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	close := func() {
		err := container.Terminate(ctx)

		if err != nil {
			slog.Error("failed to terminate sqs container", "reason", err)
		}
	}

	sqsContainer := SQSContainer{
		Container: container,
		URI:       uri,
	}

	return sqsContainer, close, nil
}

func NewSQSPriorityContainer(ctx context.Context) (*SQSPriorityContainer, func(), error) {
	queues := NewPriorityQueueConfig()
	container, close, err := NewSQSContainer(ctx)

	if err != nil {
		return nil, nil, err
	}

	pc := SQSPriorityContainer{
		Container: container,
		Queues:    queues,
	}

	deployer, err := deployments.NewSQSPriorityDeployer(&pc)

	if err != nil {
		return nil, close, err
	}

	deployedQueues, err := deployer.Deploy()

	if err != nil {
		return nil, close, err
	}

	pc.Queues = deployedQueues

	return &pc, close, nil
}
