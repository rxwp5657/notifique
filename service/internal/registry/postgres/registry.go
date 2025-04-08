package postgresresgistry

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/notifique/shared/clients"
)

type Registry struct {
	conn *pgxpool.Pool
}

type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func batchInsert(ctx context.Context, query string, args []pgx.NamedArgs, tx pgx.Tx) error {

	batch := &pgx.Batch{}

	for _, arg := range args {
		batch.Queue(query, arg)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	errArr := []error{}

	for _, e := range args {
		_, err := results.Exec()
		if err != nil {
			errArr = append(errArr, fmt.Errorf("failed to insert entry %v - %w", e, err))
		}
	}

	return errors.Join(errArr...)
}

func NewPostgresRegistry(configurator clients.PostgresConfigurator) (*Registry, error) {

	url, err := configurator.GetPostgresUrl()

	if err != nil {
		return nil, err
	}

	conn, err := pgxpool.New(context.TODO(), url)

	if err != nil {
		return nil, fmt.Errorf("failed to create pool - %w", err)
	}

	return &Registry{conn: conn}, nil
}

func NewPostgresRegistryFromPool(p *pgxpool.Pool) (*Registry, error) {

	if p == nil {
		return nil, fmt.Errorf("pool can't be nil")
	}

	return &Registry{conn: p}, nil
}
