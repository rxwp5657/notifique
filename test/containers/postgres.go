package containers

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	p "github.com/notifique/internal/deployments"
)

const (
	POSTGRES_DB       = "notifique"
	POSTGRES_USER     = "postgres"
	POSTGRES_PASSWORD = "postgres"
)

type PostgresContainer struct {
	testcontainers.Container
	URI     string
	Cleanup func() error
}

func (pc *PostgresContainer) GetPostgresUrl() (string, error) {
	return pc.URI, nil
}

func MakePostgresContainer(ctx context.Context) (*PostgresContainer, error) {

	port := "5432"

	req := testcontainers.ContainerRequest{
		Image:      "postgres:16.3",
		WaitingFor: wait.ForExposedPort(),
		Env: map[string]string{
			"POSTGRES_DB":       POSTGRES_DB,
			"POSTGRES_PASSWORD": POSTGRES_PASSWORD,
			"POSTGRES_USER":     POSTGRES_USER,
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to init the postgres container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get the dynamodb's host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return nil, fmt.Errorf("failed to acquire mapped port - %w", err)
	}

	uriTemplate := "postgres://%s:%s@%s:%s/%s?sslmode=disable"

	uri := fmt.Sprintf(
		uriTemplate,
		POSTGRES_USER,
		POSTGRES_PASSWORD,
		ip,
		mappedPort.Port(),
		POSTGRES_DB,
	)

	err = p.RunMigrations(uri)

	if err != nil {
		return nil, fmt.Errorf("failed to run migrations - %w", err)
	}

	cleanup := func() error { return container.Terminate(ctx) }
	return &PostgresContainer{URI: uri, Cleanup: cleanup}, nil
}
