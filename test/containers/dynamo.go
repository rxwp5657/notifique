package containers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	ddb "github.com/notifique/internal/deployments"
	storage "github.com/notifique/internal/storage/dynamodb"
)

type DynamoContainer struct {
	testcontainers.Container
	URI     string
	Cleanup func() error
}

func (ddbc *DynamoContainer) GetURI() string {
	return ddbc.URI
}

func NewDynamoContainer(ctx context.Context) (*DynamoContainer, error) {

	port := "8000"

	req := testcontainers.ContainerRequest{
		Image:      "amazon/dynamodb-local:2.4.0",
		WaitingFor: wait.ForExposedPort(),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to init the dynamodb container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get the dynamodb's host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return nil, fmt.Errorf("failed to acquire mapped port")
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	cfg, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to load default config - %w", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(uri)
	})

	err = ddb.CreateTables(client)

	if err != nil {
		return nil, fmt.Errorf("failed to create tables - %v", err)
	}

	cleanup := func() error { return container.Terminate(ctx) }

	dc := DynamoContainer{
		Container: container,
		URI:       uri,
		Cleanup:   cleanup,
	}

	return &dc, nil
}

func (dc *DynamoContainer) GetDynamoClientConfig() storage.DynamoClientConfig {
	return storage.DynamoClientConfig{
		BaseEndpoint: &dc.URI,
	}
}
