package containers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	p "github.com/notifique/internal/deployments"
)

const (
	PostgresDb       = "notifique"
	PostgresUser     = "postgres"
	PostgresPassword = "postgres"
)

type PostgresContainer struct {
	testcontainers.Container
	URI string
}

func (pc *PostgresContainer) GetPostgresUrl() (string, error) {
	return pc.URI, nil
}

func NewPostgresContainer(ctx context.Context) (*PostgresContainer, func(), error) {

	port := "5432"

	req := testcontainers.ContainerRequest{
		Image:      "postgres:16.3",
		WaitingFor: wait.ForExposedPort(),
		Env: map[string]string{
			"POSTGRES_DB":       PostgresDb,
			"POSTGRES_PASSWORD": PostgresPassword,
			"POSTGRES_USER":     PostgresUser,
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to init the postgres container - %w", err)
	}

	ip, err := container.Host(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the dynamodb's host - %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))

	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire mapped port - %w", err)
	}

	uriTemplate := "postgres://%s:%s@%s:%s/%s?sslmode=disable"

	uri := fmt.Sprintf(
		uriTemplate,
		PostgresUser,
		PostgresPassword,
		ip,
		mappedPort.Port(),
		PostgresDb,
	)

	err = p.RunMigrations(uri)

	if err != nil {
		return nil, nil, err
	}

	close := func() {
		err := container.Terminate(ctx)

		if err != nil {
			slog.Error("failed to terminate postgres container", "reason", err)
		}
	}

	return &PostgresContainer{URI: uri}, close, nil
}
