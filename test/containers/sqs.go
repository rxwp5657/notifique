package containers

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	deployments "github.com/notifique/deployments/sqs"
	"github.com/notifique/internal/publisher"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

type SQSContainerCleanupFn func() error

type SQSContainer struct {
	testcontainers.Container
	URI       string
	SQSQueues publisher.PriorityQueues
	CleanupFn SQSContainerCleanupFn
}

func MakeSQSContainer(ctx context.Context) (*SQSContainer, error) {

	port := "4566"

	container, err := localstack.RunContainer(
		ctx,
		testcontainers.WithImage("localstack/localstack:3.4"),
		testcontainers.WithEnv(map[string]string{
			"SERVICES": "sqs",
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create sqs container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get the sqs host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	cfg := publisher.SQSClientConfig{BaseEndpoint: &uri}
	client, err := publisher.MakeSQSClient(cfg)

	if err != nil {
		return nil, fmt.Errorf("failed to create sqs client")
	}

	low := PRIORITY_QUEUE_LOW_NAME
	medium := PRIORITY_QUEUE_MEDIUM_NAME
	high := PRIORITY_QUEUE_HIGH_NAME

	queues := publisher.PriorityQueues{
		Low:    &low,
		Medium: &medium,
		High:   &high,
	}

	urls, err := deployments.MakePriorityQueues(client, queues)

	if err != nil {
		return nil, fmt.Errorf("failed to create queues - %w", err)
	}

	cleanup := func() error { return container.Terminate(ctx) }

	sqsContainer := SQSContainer{
		Container: container,
		URI:       uri,
		SQSQueues: urls,
		CleanupFn: cleanup,
	}

	return &sqsContainer, nil
}
