package registry_test

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/notifique/service/internal/dto"
	ps "github.com/notifique/service/internal/registry/postgres"
	"github.com/notifique/shared/containers"
	sdto "github.com/notifique/shared/dto"
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

const deleteUserNotification = `
DELETE FROM 
	ser_notifications
WHERE
	id = ANY($1);
`

const getNotificationTemplate = `
SELECT
	"name",
	"is_html",
	"description",
	title_template,
	contents_template
FROM
	notification_templates
WHERE
	id = $1;
`

const getNotificationTemplateVariables = `
SELECT
    "name",
    "type",
    "required",
    "validation"
FROM
	notification_template_variables
WHERE
	template_id = $1;
`

const countTemplateWithId = `
SELECT
	COUNT(*)
FROM
	notification_templates
WHERE
	id = $1;
`

type postgresresgistryTester struct {
	*ps.Registry
	conn *pgxpool.Pool
}

type closer func()

func (t *postgresresgistryTester) ClearDB(ctx context.Context) error {
	_, err := t.conn.Exec(ctx, `
		TRUNCATE distribution_lists CASCADE;
		TRUNCATE distribution_list_recipients CASCADE;
		TRUNCATE notifications CASCADE;
		TRUNCATE notification_status_log CASCADE;
		TRUNCATE notification_recipients CASCADE;
		TRUNCATE notification_channels CASCADE;
		TRUNCATE user_notifications;
		TRUNCATE user_config;
		TRUNCATE notification_templates CASCADE;
		TRUNCATE notification_template_variables CASCADE;
	`)

	return err
}

func (t *postgresresgistryTester) GetDistributionList(ctx context.Context, dlName string) (dto.DistributionList, error) {
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

func (t *postgresresgistryTester) DistributionListExists(ctx context.Context, dlName string) (bool, error) {

	var numDLsWithName int

	err := t.conn.QueryRow(ctx, getNumDLsWithName, dlName).Scan(&numDLsWithName)

	if err != nil {
		return false, fmt.Errorf("failed to retrieve count - %w", err)
	}

	return numDLsWithName != 0, nil
}

func (t *postgresresgistryTester) InsertUserNotifications(
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

func (t *postgresresgistryTester) DeleteUserNotifications(ctx context.Context, userId string, ids []string) error {
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

func (t *postgresresgistryTester) GetNotificationTemplate(ctx context.Context, templateId string) (dto.NotificationTemplateReq, error) {
	templateReq := dto.NotificationTemplateReq{}

	err := t.conn.QueryRow(ctx, getNotificationTemplate, templateId).
		Scan(
			&templateReq.Name,
			&templateReq.IsHtml,
			&templateReq.Description,
			&templateReq.TitleTemplate,
			&templateReq.ContentsTemplate,
		)

	if err != nil {
		return templateReq, fmt.Errorf("failed to retrieve notification template info - %w", err)
	}

	rows, err := t.conn.Query(ctx, getNotificationTemplateVariables, templateId)

	if err != nil {
		return templateReq, fmt.Errorf("failed to query template variables - %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		row := sdto.TemplateVariable{}

		err = rows.Scan(
			&row.Name,
			&row.Type,
			&row.Required,
			&row.Validation,
		)

		if err != nil {
			return templateReq, fmt.Errorf("failed to scan template variable - %w", err)
		}

		templateReq.Variables = append(templateReq.Variables, row)
	}

	return templateReq, nil
}

func (t *postgresresgistryTester) TemplateExists(ctx context.Context, id string) (bool, error) {

	count := 0

	err := t.conn.QueryRow(ctx, countTemplateWithId, id).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to query template count - %w", err)
	}

	return count > 0, nil
}

func NewPostgresIntegrationTester(ctx context.Context) (*postgresresgistryTester, closer, error) {
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

	registry, err := ps.NewPostgresRegistryFromPool(conn)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create registry - %w", err)
	}

	tester := postgresresgistryTester{
		Registry: registry,
		conn:     conn,
	}

	closer := func() {
		containerCloser()
	}

	return &tester, closer, nil
}
