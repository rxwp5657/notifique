package postgresresgistry

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Registry struct {
	conn *pgxpool.Pool
}

type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresConfigurator interface {
	GetPostgresUrl() (string, error)
}

type namedArgsBuilder[T any] func(val T) pgx.NamedArgs

func batchInsert[T any](ctx context.Context, query string, data []T, builder namedArgsBuilder[T], tx pgx.Tx) error {

	batch := &pgx.Batch{}

	for _, e := range data {
		args := builder(e)
		batch.Queue(query, args)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	for _, e := range data {
		_, err := results.Exec()
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to insert entry %v - %w", e, err)
		}
	}

	return nil
}

func NewPostgresRegistry(configurator PostgresConfigurator) (*Registry, error) {

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
