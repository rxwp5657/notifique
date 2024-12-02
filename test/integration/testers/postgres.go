package testers

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/notifique/dto"
	storage "github.com/notifique/internal/storage/postgres"
	"github.com/notifique/test/containers"
)

const getDLRecipients = `
SELECT
	"name",
	recipient
FROM
	distribution_list_recipients
WHERE
	"name" = $1;
`

const getNumDLsWithName = `
SELECT
	COUNT(*)
FROM
	distribution_lists
WHERE
	"name" = $1;
`

type PostgresIntegrationTester struct {
	*storage.PostgresStorage
	conn *pgxpool.Pool
}

type closer func()

func (t *PostgresIntegrationTester) ClearDB(ctx context.Context) error {
	_, err := t.conn.Exec(ctx, `
		TRUNCATE distribution_lists CASCADE;
		TRUNCATE distribution_list_recipients CASCADE;
	`)

	return err
}

func (t *PostgresIntegrationTester) GetDistributionList(ctx context.Context, dlName string) (dto.DistributionList, error) {
	rows, err := t.conn.Query(ctx, getDLRecipients, dlName)

	if err != nil {
		return dto.DistributionList{}, err
	}

	dl := dto.DistributionList{
		Name:       dlName,
		Recipients: []string{},
	}

	n, recipient := "", ""

	pgx.ForEachRow(rows, []any{&n, &recipient}, func() error {
		dl.Recipients = append(dl.Recipients, recipient)
		return nil
	})

	return dl, nil
}

func (t *PostgresIntegrationTester) DistributionListExists(ctx context.Context, dlName string) (bool, error) {

	var numDLsWithName int

	err := t.conn.QueryRow(ctx, getNumDLsWithName, dlName).Scan(&numDLsWithName)

	if err != nil {
		return false, fmt.Errorf("failed to retrieve count - %w", err)
	}

	return numDLsWithName != 0, nil
}

func NewPostgresIntegrationTester(ctx context.Context) (*PostgresIntegrationTester, closer, error) {
	container, containerCloser, err := containers.NewPostgresContainer(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create container - %w", err)
	}

	url, err := container.GetPostgresUrl()

	if err != nil {
		return nil, nil, err
	}

	conn, err := pgxpool.New(context.TODO(), url)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create pool - %w", err)
	}

	storage, err := storage.NewPostgresStorageFromPool(conn)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create storage - %w", err)
	}

	tester := PostgresIntegrationTester{
		PostgresStorage: storage,
		conn:            conn,
	}

	closer := func() {
		containerCloser()
	}

	return &tester, closer, nil
}
