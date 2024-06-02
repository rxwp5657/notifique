package test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	ddb "github.com/notifique/deployments/dynamodb"
)

type dynamodbContainer struct {
	testcontainers.Container
	URI string
}

func (ddbc *dynamodbContainer) GetURI() string {
	return ddbc.URI
}

func setupDynamoDB(ctx context.Context) (*dynamodbContainer, error) {

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
		return nil, err
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

	return &dynamodbContainer{Container: container, URI: uri}, nil
}
