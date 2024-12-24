package storage_test

import (
	"context"
	"fmt"
	"time"

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

const getNotification = `
SELECT
	title,
	contents,
	image_url,
	topic,
	"priority",
	distribution_list
FROM
	notifications
WHERE
	id = $1;
`

const getNotificationStatus = `
SELECT
	"status"
FROM
	notifications
WHERE
	id = $1;
`

const getNotificationRecipients = `
SELECT
	recipient
FROM
	notification_recipients
WHERE
	notification_id = $1;
`

const getNotificationChannels = `
SELECT
	channel
FROM
	notification_channels
WHERE
	notification_id = $1;
`

const deleteUserNotification = `
DELETE FROM 
	ser_notifications
WHERE
	id = ANY($1);
`

type PostgresStorageTester struct {
	*storage.PostgresStorage
	conn *pgxpool.Pool
}

type closer func()

func (t *PostgresStorageTester) ClearDB(ctx context.Context) error {
	_, err := t.conn.Exec(ctx, `
		TRUNCATE distribution_lists CASCADE;
		TRUNCATE distribution_list_recipients CASCADE;
		TRUNCATE notifications CASCADE;
		TRUNCATE notification_status_log CASCADE;
		TRUNCATE notification_recipients CASCADE;
		TRUNCATE notification_channels CASCADE;
		TRUNCATE user_notifications;
		TRUNCATE user_config;
	`)

	return err
}

func (t *PostgresStorageTester) GetDistributionList(ctx context.Context, dlName string) (dto.DistributionList, error) {
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

func (t *PostgresStorageTester) DistributionListExists(ctx context.Context, dlName string) (bool, error) {

	var numDLsWithName int

	err := t.conn.QueryRow(ctx, getNumDLsWithName, dlName).Scan(&numDLsWithName)

	if err != nil {
		return false, fmt.Errorf("failed to retrieve count - %w", err)
	}

	return numDLsWithName != 0, nil
}

func (t *PostgresStorageTester) GetNotification(ctx context.Context, notificationId string) (dto.NotificationReq, error) {

	notification := dto.NotificationReq{}

	err := t.conn.QueryRow(ctx, getNotification, notificationId).
		Scan(
			&notification.Title,
			&notification.Contents,
			&notification.Image,
			&notification.Topic,
			&notification.Priority,
			&notification.DistributionList,
		)

	if err != nil {
		return notification, fmt.Errorf("failed to retrieve notification info - %w", err)
	}

	queryData := func(query string) ([]string, error) {

		data := []string{}

		rows, err := t.conn.Query(ctx, query, notificationId)

		if err != nil {
			return data, err
		}

		defer rows.Close()

		for rows.Next() {
			row := ""

			err = rows.Scan(&row)

			if err != nil {
				return data, err
			}

			data = append(data, row)
		}

		return data, nil
	}

	channels, err := queryData(getNotificationChannels)

	if err != nil {
		return notification, fmt.Errorf("failed to retrieve notification channels - %w", err)
	}

	notification.Channels = channels

	recipients, err := queryData(getNotificationRecipients)

	if err != nil {
		return notification, fmt.Errorf("failed to retrieve notification recipients - %w", err)
	}

	notification.Recipients = recipients

	return notification, nil
}

func (t *PostgresStorageTester) InsertUserNotifications(
	ctx context.Context,
	userId string,
	notifications []dto.UserNotification) error {

	tx, err := t.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	table := "user_notifications"
	columns := []string{
		"id",
		"user_id",
		"title",
		"contents",
		"created_at",
		"read_at",
		"image_url",
		"topic",
	}

	parseTime := func(timestamp *string) (*time.Time, error) {
		if timestamp == nil {
			return nil, nil
		}

		t, err := time.Parse(time.RFC3339Nano, *timestamp)

		return &t, err
	}

	_, err = t.conn.CopyFrom(ctx, pgx.Identifier{table}, columns, pgx.CopyFromSlice(
		len(notifications), func(i int) ([]any, error) {
			createdAt, err := parseTime(&notifications[i].CreatedAt)

			if err != nil {
				return nil, err
			}

			readAt, err := parseTime(notifications[i].ReadAt)

			if err != nil {
				return nil, err
			}

			return []any{
				notifications[i].Id,
				userId,
				notifications[i].Title,
				notifications[i].Contents,
				createdAt,
				readAt,
				notifications[i].Image,
				notifications[i].Topic,
			}, nil
		},
	))

	if err != nil {
		return fmt.Errorf("failed to copy notifications - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit user notification insert - %w", err)
	}

	return nil
}

func (t *PostgresStorageTester) DeleteUserNotifications(ctx context.Context, userId string, ids []string) error {
	tx, err := t.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	_, err = tx.Exec(ctx, deleteUserNotification, ids)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to delete user notification - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit user notification delete - %w", err)
	}

	return nil
}

func (t *PostgresStorageTester) GetNotificationStatus(ctx context.Context, notificationId string) (string, error) {

	status := ""

	err := t.conn.QueryRow(ctx, getNotificationStatus, notificationId).
		Scan(&status)

	if err != nil {
		return "", fmt.Errorf("failed to retrieve notification status - %w", err)
	}

	return status, nil
}

func NewPostgresIntegrationTester(ctx context.Context) (*PostgresStorageTester, closer, error) {
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

	tester := PostgresStorageTester{
		PostgresStorage: storage,
		conn:            conn,
	}

	closer := func() {
		containerCloser()
	}

	return &tester, closer, nil
}
