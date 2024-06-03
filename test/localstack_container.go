package test

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	deployments "github.com/notifique/deployments/sqs"
	"github.com/notifique/internal/publisher"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

type localStackContainer struct {
	testcontainers.Container
	URI     string
	SQSUrls publisher.SQSUrls
}

func setupLocalStack(ctx context.Context) (*localStackContainer, func() error, error) {

	port := "4566"

	container, err := localstack.RunContainer(
		ctx,
		testcontainers.WithImage("localstack/localstack:3.4"),
		testcontainers.WithEnv(map[string]string{
			"SERVICES": "sqs",
		}),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create sqs container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the sqs host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return nil, nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	client, err := publisher.MakeSQSClient(&uri)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create sqs client")
	}

	urls, err := deployments.MakeQueues(client)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create queues - %w", err)
	}

	closer := func() error { return container.Terminate(ctx) }
	return &localStackContainer{Container: container, URI: uri, SQSUrls: urls}, closer, nil
}
