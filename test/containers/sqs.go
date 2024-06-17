package containers

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/notifique/internal/deployments"
	"github.com/notifique/internal/publisher"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

type SQSContainer struct {
	testcontainers.Container
	URI     string
	Cleanup func() error
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

func NewSQSContainer(ctx context.Context) (SQSContainer, error) {

	port := "4566"

	container, err := localstack.RunContainer(
		ctx,
		testcontainers.WithImage("localstack/localstack:3.4"),
		testcontainers.WithEnv(map[string]string{
			"SERVICES": "sqs",
		}),
	)

	if err != nil {
		return SQSContainer{}, fmt.Errorf("failed to create sqs container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return SQSContainer{}, fmt.Errorf("failed to get the sqs host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return SQSContainer{}, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	cleanup := func() error { return container.Terminate(ctx) }

	sqsContainer := SQSContainer{
		Container: container,
		URI:       uri,
		Cleanup:   cleanup,
	}

	return sqsContainer, nil
}

func NewSQSPriorityContainer(ctx context.Context) (*SQSPriorityContainer, error) {
	queues := NewPriorityQueueConfig()
	container, err := NewSQSContainer(ctx)

	if err != nil {
		return nil, nil
	}

	pc := SQSPriorityContainer{
		Container: container,
		Queues:    queues,
	}

	deployer, cleanup, err := deployments.NewSQSPriorityDeployer(&pc)

	if err != nil {
		return nil, err
	}

	defer cleanup()

	deployedQueues, err := deployer.Deploy()

	if err != nil {
		return nil, err
	}

	pc.Queues = deployedQueues

	return &pc, nil
}
