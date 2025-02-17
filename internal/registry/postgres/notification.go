package postgresresgistry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	c "github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/dto"
)

const InsertNotification = `
INSERT INTO notifications (
	title,
	contents,
	image_url,
	topic,
	priority,
	distribution_list,
	created_at,
	status
) VALUES (
	@title,
	@contents,
	@imageUrl,
	@topic,
	@priority,
	@distributionList,
	@createdAt,
	@status
) RETURNING
	id;
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

func (ps *Registry) createStatusLog(ctx context.Context, tx pgx.Tx, statusLog c.NotificationStatusLog) error {

	args := pgx.NamedArgs{
		"notificationId": statusLog.NotificationId,
		"statusDate":     time.Now().Format(time.RFC3339Nano),
		"status":         statusLog.Status,
		"errorMessage":   statusLog.ErrorMsg,
	}

	_, err := tx.Exec(ctx, InsertNotificationStatusLog, args)

	return err
}

func (ps *Registry) UpdateNotificationStatus(ctx context.Context, statusLog c.NotificationStatusLog) error {

	tx, err := ps.conn.Begin(ctx)

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

	err = ps.createStatusLog(ctx, tx, statusLog)

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

func (ps *Registry) SaveNotification(ctx context.Context, createdBy string, notification dto.NotificationReq) (string, error) {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to start transaction - %w", err)
	}

	notificationArgs := pgx.NamedArgs{
		"title":            notification.Title,
		"contents":         notification.Contents,
		"imageUrl":         notification.Image,
		"topic":            notification.Topic,
		"priority":         notification.Priority,
		"distributionList": notification.DistributionList,
		"createdAt":        time.Now().Format(time.RFC3339Nano),
		"status":           c.Created,
	}

	var notificationIdUUID uuid.UUID

	err = tx.QueryRow(ctx, InsertNotification, notificationArgs).Scan(&notificationIdUUID)

	if err != nil {
		tx.Rollback(ctx)
		return "", fmt.Errorf("failed to insert notification - %w", err)
	}

	notificationId := notificationIdUUID.String()

	recipientsBuilder := func(recipient string) pgx.NamedArgs {
		return pgx.NamedArgs{
			"notificationId": notificationId,
			"recipient":      recipient,
		}
	}

	err = batchInsert(
		ctx,
		InsertNotificationRecipients,
		notification.Recipients,
		recipientsBuilder,
		tx,
	)

	if err != nil {
		return "", err
	}

	channelsBuilder := func(channel string) pgx.NamedArgs {
		return pgx.NamedArgs{
			"notificationId": notificationId,
			"channel":        channel,
		}
	}

	err = batchInsert(
		ctx,
		InsertChannels,
		notification.Channels,
		channelsBuilder,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return "", err
	}

	statusLog := c.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         c.Created,
		ErrorMsg:       nil,
	}

	err = ps.createStatusLog(ctx, tx, statusLog)

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
