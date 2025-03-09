package postgresresgistry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/notifique/internal/registry"
	"github.com/notifique/internal/server"
	c "github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/dto"
)

const InsertNotification = `
INSERT INTO notifications (
	id,
	title,
	contents,
	template_id,
	image_url,
	topic,
	priority,
	distribution_list,
	created_at,
	created_by,
	status
) VALUES (
	@id,
	@title,
	@contents,
	@templateId,
	@imageUrl,
	@topic,
	@priority,
	@distributionList,
	@createdAt,
	@createdBy,
	@status
);
`

const insertNotificationTemplateVariableContents = `
INSERT INTO notification_template_variable_contents(
	notification_id,
	name,
	value
) VALUES (
	@notificationId,
	@name,
	@value
);
`

const InsertNotificationRecipients = `
INSERT INTO notification_recipients (
	notification_id,
	recipient
) VALUES (
	@notificationId,
	@recipient
);
`

const InsertChannels = `
INSERT INTO notification_channels (
	notification_id,
	channel
) VALUES (
	@notificationId,
	@channel
);
`

const InsertNotificationStatusLog = `
INSERT INTO notification_status_log (
	notification_id,
    status_date,
    "status",
    error_message
) VALUES (
	@notificationId,
	@statusDate,
	@status,
	@errorMessage
);
`

const UpdateNotificationStatus = `
UPDATE
	notifications
SET
	status = @status
WHERE
	id = @notificationId;
`

const deleteTemplate = `
DELETE FROM
	notifications
WHERE
	id = $1;
`

const getNotificationStatusQry = `
SELECT
	status
FROM
	notifications
WHERE
	id = $1;
`

const getNotificationSummaries = `
SELECT
	id,
	topic,
	template_id,
	created_at,
	created_by,
	priority,
	status
FROM
	notifications
%s
ORDER BY
	id DESC
LIMIT
	@limit;
`

type notificationKey struct {
	Id string `db:"id"`
}

type notificationSummary struct {
	Id         string    `db:"id"`
	Topic      string    `db:"topic"`
	TemplateId *string   `db:"template_id"`
	CreatedAt  time.Time `db:"created_at"`
	CreatedBy  string    `db:"created_by"`
	Priority   string    `db:"priority"`
	Status     string    `db:"status"`
}

func (r *Registry) createStatusLog(ctx context.Context, tx pgx.Tx, statusLog c.NotificationStatusLog) error {

	args := pgx.NamedArgs{
		"notificationId": statusLog.NotificationId,
		"statusDate":     time.Now().Format(time.RFC3339Nano),
		"status":         statusLog.Status,
		"errorMessage":   statusLog.ErrorMsg,
	}

	_, err := tx.Exec(ctx, InsertNotificationStatusLog, args)

	return err
}

func (r *Registry) UpdateNotificationStatus(ctx context.Context, statusLog c.NotificationStatusLog) error {

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"notificationId": statusLog.NotificationId,
		"status":         statusLog.Status,
	}

	_, err = tx.Exec(ctx, UpdateNotificationStatus, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to update notification status logs - %w", err)
	}

	err = r.createStatusLog(ctx, tx, statusLog)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert notification status logs - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit notification status log - %w", err)
	}

	return nil
}

func (r *Registry) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to start transaction - %w", err)
	}

	id, err := uuid.NewV7()

	if err != nil {
		tx.Rollback(ctx)
		return "", fmt.Errorf("failed to generate notification id - %w", err)
	}

	notificationId := id.String()

	notificationArgs := pgx.NamedArgs{
		"id":               notificationId,
		"title":            nil,
		"contents":         nil,
		"templateId":       nil,
		"imageUrl":         notificationReq.Image,
		"topic":            notificationReq.Topic,
		"priority":         notificationReq.Priority,
		"distributionList": notificationReq.DistributionList,
		"createdAt":        time.Now().Format(time.RFC3339Nano),
		"createdBy":        createdBy,
		"status":           dto.Created,
	}

	if notificationReq.RawContents != nil {
		notificationArgs["title"] = notificationReq.RawContents.Title
		notificationArgs["contents"] = notificationReq.RawContents.Contents
	} else {
		notificationArgs["templateId"] = notificationReq.TemplateContents.Id
	}

	_, err = tx.Exec(ctx, InsertNotification, notificationArgs)

	if err != nil {
		tx.Rollback(ctx)
		return "", fmt.Errorf("failed to insert notification - %w", err)
	}

	if notificationReq.TemplateContents != nil {
		numVariables := len(notificationReq.TemplateContents.Variables)
		variables := make([]pgx.NamedArgs, 0, numVariables)

		for _, v := range notificationReq.TemplateContents.Variables {
			variables = append(variables, pgx.NamedArgs{
				"notificationId": notificationId,
				"name":           v.Name,
				"value":          v.Value,
			})
		}

		err := batchInsert(
			ctx,
			insertNotificationTemplateVariableContents,
			variables,
			tx,
		)

		if err != nil {
			tx.Rollback(ctx)
			return "", fmt.Errorf("failed to insert notification template variables - %w", err)
		}
	}

	recipientsArgs := make([]pgx.NamedArgs, 0, len(notificationReq.Recipients))

	for _, recipient := range notificationReq.Recipients {
		recipientsArgs = append(recipientsArgs, pgx.NamedArgs{
			"notificationId": notificationId,
			"recipient":      recipient,
		})
	}

	err = batchInsert(
		ctx,
		InsertNotificationRecipients,
		recipientsArgs,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return "", err
	}

	channelsArgs := make([]pgx.NamedArgs, 0, len(notificationReq.Channels))

	for _, channel := range notificationReq.Channels {
		channelsArgs = append(channelsArgs, pgx.NamedArgs{
			"notificationId": notificationId,
			"channel":        channel,
		})
	}

	err = batchInsert(
		ctx,
		InsertChannels,
		channelsArgs,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return "", err
	}

	statusLog := c.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         dto.Created,
		ErrorMsg:       nil,
	}

	err = r.createStatusLog(ctx, tx, statusLog)

	if err != nil {
		tx.Rollback(ctx)
		return "", fmt.Errorf("failed to create notification status logs - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to commit notification insert - %w", err)
	}

	return notificationId, nil
}

func (r *Registry) GetNotificationStatus(ctx context.Context, id string) (dto.NotificationStatus, error) {
	var status dto.NotificationStatus

	err := r.conn.QueryRow(ctx, getNotificationStatusQry, id).Scan(&status)

	if err == pgx.ErrNoRows {
		return status, server.EntityNotFound{Id: id, Type: registry.NotificationType}
	} else if err != nil {
		return status, fmt.Errorf("failed to query the notification status - %w", err)
	}

	return status, err
}

func (r *Registry) DeleteNotification(ctx context.Context, id string) error {

	status, err := r.GetNotificationStatus(ctx, id)

	if err != nil && errors.As(err, &server.EntityNotFound{}) {
		return nil
	} else if err != nil {
		return err
	}

	canDelete := registry.IsDeletableStatus(status)

	if !canDelete {
		return server.InvalidNotificationStatus{
			Id:     id,
			Status: string(status),
		}
	}

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	// Relies on ON DELETE CASCADE constraint to delete the notification
	// channels, recipients, and logs
	_, err = tx.Exec(ctx, deleteTemplate, id)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to delete notification - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit changes - %w", err)
	}

	return nil
}

func (r *Registry) GetNotifications(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.NotificationSummary], error) {

	var page dto.Page[dto.NotificationSummary]

	args := pgx.NamedArgs{"limit": server.PageSize}
	whereFilter := ""

	if filters.MaxResults != nil {
		args["limit"] = *filters.MaxResults
	}

	if filters.NextToken != nil {
		whereFilter = "id < @id"

		var unmarsalledKey notificationKey
		err := registry.UnmarshalKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		args["id"] = unmarsalledKey.Id
	}

	whereStmt := ""

	if whereFilter != "" {
		whereStmt = fmt.Sprintf("WHERE %s", whereFilter)
	}

	query := fmt.Sprintf(getNotificationSummaries, whereStmt)

	rows, err := r.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	collectedSummaries, err := pgx.CollectRows(rows, pgx.RowToStructByName[notificationSummary])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	summaries := make([]dto.NotificationSummary, 0, len(collectedSummaries))

	for _, s := range collectedSummaries {

		contentsType := dto.Raw

		if s.TemplateId != nil {
			contentsType = dto.Template
		}

		summaries = append(summaries, dto.NotificationSummary{
			Id:           s.Id,
			Topic:        s.Topic,
			ContentsType: contentsType,
			CreatedAt:    s.CreatedAt.Format(time.RFC3339Nano),
			CreatedBy:    s.CreatedBy,
			Priority:     dto.NotificationPriority(s.Priority),
			Status:       dto.NotificationStatus(s.Status),
		})
	}

	numSummaries := len(summaries)

	if numSummaries == args["limit"] {
		lastSummarie := summaries[numSummaries-1]
		lastKey := notificationKey{Id: lastSummarie.Id}

		key, err := registry.MarshalKey(lastKey)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(summaries)
	page.Data = summaries

	return page, nil
}
