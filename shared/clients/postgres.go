package clients

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfigurator interface {
	GetPostgresUrl() (string, error)
}

func NewPostgresPool(configurator PostgresConfigurator) (*pgxpool.Pool, error) {

	url, err := configurator.GetPostgresUrl()

	if err != nil {
		return nil, err
	}

	conn, err := pgxpool.New(context.TODO(), url)

	if err != nil {
		return nil, fmt.Errorf("failed to create pool - %w", err)
	}

	return conn, nil
}
