package containers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/go-connections/nat"
	"github.com/notifique/shared/clients"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

type SQS struct {
	testcontainers.Container
	URI string
}

func (sc *SQS) GetSQSClientConfig() clients.SQSClientConfig {
	return clients.SQSClientConfig{
		BaseEndpoint: &sc.URI,
	}
}

func NewSQSContainer(ctx context.Context) (SQS, func(), error) {

	port := "4566"

	container, err := localstack.RunContainer(
		ctx,
		testcontainers.WithImage("localstack/localstack:3.4"),
		testcontainers.WithEnv(map[string]string{
			"SERVICES": "sqs",
		}),
	)

	if err != nil {
		return SQS{}, nil, fmt.Errorf("failed to create sqs container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return SQS{}, nil, fmt.Errorf("failed to get the sqs host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return SQS{}, nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	close := func() {
		err := container.Terminate(ctx)

		if err != nil {
			slog.Error("failed to terminate sqs container", "reason", err)
		}
	}

	sqsContainer := SQS{
		Container: container,
		URI:       uri,
	}

	return sqsContainer, close, nil
}
