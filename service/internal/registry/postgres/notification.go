package postgresresgistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	sdto "github.com/notifique/shared/dto"
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

const getNotification = `
SELECT
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
	status,
	ARRAY_AGG(distinct channel) AS channels,
	ARRAY_AGG(distinct recipient) AS recipients,
	ARRAY_AGG(
		distinct nvc.name || '%s' || nvc.value
	) AS variable_names
FROM
	notifications AS n
JOIN
	notification_recipients AS nr ON
		nr.notification_id = n.id
JOIN
	notification_channels AS nc ON
		nc.notification_id = n.id
LEFT JOIN
	notification_template_variable_contents AS nvc ON
		nvc.notification_id = n.id
WHERE
	id = $1
GROUP BY
	n.id;
`

const insertRecipientNotificationStatusLog = `
INSERT INTO recipient_notification_status_log (
	notification_id,
	user_id,
	channel,
	status,
	error_message
) VALUES (
	@notificationId,
	@userId,
	@channel,
	@status,
	@errorMessage
);
`

const geRecipientNotificationStatuses = `
WITH lastest_status AS (
	SELECT
		user_id,
		status,
		channel,
		error_message,
		ROW_NUMBER() OVER (
			PARTITION BY notification_id, user_id, channel
			ORDER BY status_date DESC
		) AS rn
	FROM
		recipient_notification_status_log
	WHERE
		notification_id = @notificationId
)
SELECT
	user_id,
	status,
	channel,
	error_message
FROM
	lastest_status
WHERE
	%s
ORDER BY
	user_id
LIMIT
	@limit;
`

type notificationKey struct {
	Id string `db:"id"`
}

type recipientNotificationStatusKey struct {
	UserId         string `json:"userId"`
	NotificationId string `json:"notificationId"`
	Channel        string `json:"channel"`
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

type recipientNotificationStatus struct {
	UserId       string  `db:"user_id"`
	Status       string  `db:"status"`
	Channel      string  `db:"channel"`
	ErrorMessage *string `db:"error_message"`
}

func (r *Registry) createStatusLog(ctx context.Context, tx pgx.Tx, statusLog sdto.NotificationStatusLog) error {

	args := pgx.NamedArgs{
		"notificationId": statusLog.NotificationId,
		"statusDate":     time.Now().Format(time.RFC3339Nano),
		"status":         statusLog.Status,
		"errorMessage":   statusLog.ErrorMsg,
	}

	_, err := tx.Exec(ctx, InsertNotificationStatusLog, args)

	return err
}

func (r *Registry) UpdateNotificationStatus(ctx context.Context, statusLog sdto.NotificationStatusLog) error {

	exists, err := r.notificationExists(ctx, statusLog.NotificationId)

	if err != nil {
		return fmt.Errorf("failed to check if notification exists - %w", err)
	}

	if !exists {
		return internal.EntityNotFound{
			Id:   statusLog.NotificationId,
			Type: registry.NotificationType,
		}
	}

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

func (r *Registry) SaveNotification(ctx context.Context, createdBy string, notificationReq sdto.NotificationReq) (string, error) {

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
		"status":           sdto.Created,
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

	statusLog := sdto.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         sdto.Created,
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

func (r *Registry) GetNotificationStatus(ctx context.Context, id string) (sdto.NotificationStatus, error) {
	var status sdto.NotificationStatus

	err := r.conn.QueryRow(ctx, getNotificationStatusQry, id).Scan(&status)

	if err == pgx.ErrNoRows {
		return status, internal.EntityNotFound{Id: id, Type: registry.NotificationType}
	} else if err != nil {
		return status, fmt.Errorf("failed to query the notification status - %w", err)
	}

	return status, err
}

func (r *Registry) DeleteNotification(ctx context.Context, id string) error {

	status, err := r.GetNotificationStatus(ctx, id)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		return nil
	} else if err != nil {
		return err
	}

	canDelete := registry.IsDeletableStatus(status)

	if !canDelete {
		return internal.InvalidNotificationStatus{
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

func (r *Registry) GetNotifications(ctx context.Context, filters sdto.PageFilter) (sdto.Page[dto.NotificationSummary], error) {

	var page sdto.Page[dto.NotificationSummary]

	args := pgx.NamedArgs{"limit": internal.PageSize}
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
			Priority:     sdto.NotificationPriority(s.Priority),
			Status:       sdto.NotificationStatus(s.Status),
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

func (r *Registry) GetNotification(ctx context.Context, notificationId string) (dto.NotificationResp, error) {

	notification := dto.NotificationResp{}

	rawContents := struct {
		Title    *string
		Contents *string
	}{}

	var templateId *string = nil

	createdAt := time.Time{}
	channelsAgg := []string{}
	recipientsAgg := []string{}
	variablesAgg := []*string{}

	query := fmt.Sprintf(getNotification, internal.TemplateVariableNameSeparator)

	err := r.conn.QueryRow(ctx, query, notificationId).Scan(
		&notification.Id,
		&rawContents.Title,
		&rawContents.Contents,
		&templateId,
		&notification.Image,
		&notification.Topic,
		&notification.Priority,
		&notification.DistributionList,
		&createdAt,
		&notification.CreatedBy,
		&notification.Status,
		&channelsAgg,
		&recipientsAgg,
		&variablesAgg,
	)

	if err == pgx.ErrNoRows {
		return notification, internal.EntityNotFound{Id: notificationId, Type: registry.NotificationType}
	} else if err != nil {
		return notification, fmt.Errorf("failed to query the notification - %w", err)
	}

	notification.CreatedAt = createdAt.Format(time.RFC3339Nano)
	notification.Recipients = recipientsAgg

	channels := make([]sdto.NotificationChannel, 0, len(channelsAgg))

	for _, c := range channelsAgg {
		channels = append(channels, sdto.NotificationChannel(c))
	}

	notification.Channels = channels

	if templateId != nil {

		variables := make([]sdto.TemplateVariableContents, 0, len(variablesAgg))

		for _, v := range variablesAgg {

			if v == nil {
				continue
			}

			name, value, ok := strings.Cut(*v, "~")

			if !ok {
				return notification, fmt.Errorf("failed to parse template variable - %w", err)
			}

			variables = append(variables, sdto.TemplateVariableContents{
				Name:  name,
				Value: value,
			})
		}

		notification.TemplateContents = &sdto.TemplateContents{
			Id:        *templateId,
			Variables: variables,
		}

	} else {
		notification.RawContents = &sdto.RawContents{
			Title:    *rawContents.Title,
			Contents: *rawContents.Contents,
		}
	}

	return notification, nil
}

func (r *Registry) notificationExists(ctx context.Context, notificationId string) (bool, error) {

	var exists string

	err := r.conn.QueryRow(ctx, getNotificationStatusQry, notificationId).Scan(&exists)

	if err == pgx.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to query the notification - %w", err)
	}

	return true, nil
}

func (r *Registry) UpsertRecipientNotificationStatuses(ctx context.Context, notificationId string, statuses []sdto.RecipientNotificationStatus) error {

	exists, err := r.notificationExists(ctx, notificationId)

	if err != nil {
		return fmt.Errorf("failed to check if notification exists - %w", err)
	}

	if !exists {
		return internal.EntityNotFound{
			Id:   notificationId,
			Type: registry.NotificationType,
		}
	}

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	statusesArgs := make([]pgx.NamedArgs, 0, len(statuses))

	for _, status := range statuses {
		statusesArgs = append(statusesArgs, pgx.NamedArgs{
			"notificationId": notificationId,
			"userId":         status.UserId,
			"status":         status.Status,
			"channel":        status.Channel,
			"errorMessage":   status.ErrMsg,
		})
	}

	err = batchInsert(
		ctx,
		insertRecipientNotificationStatusLog,
		statusesArgs,
		tx,
	)

	if err != nil {
		rallbackErr := tx.Rollback(ctx)
		err = errors.Join(err, rallbackErr)
		return fmt.Errorf("failed to insert recipient notification statuses - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit recipient notification statuses - %w", err)
	}

	return nil
}

func (r *Registry) GetRecipientNotificationStatuses(ctx context.Context, notificationId string, filters sdto.NotificationRecipientStatusFilters) (sdto.Page[sdto.RecipientNotificationStatus], error) {
	page := sdto.Page[sdto.RecipientNotificationStatus]{}

	args := pgx.NamedArgs{
		"limit":          internal.PageSize,
		"notificationId": notificationId,
	}

	whereFilters := []string{"rn = 1"}

	if filters.MaxResults != nil {
		args["limit"] = *filters.MaxResults
	}

	if filters.NextToken != nil {
		whereFilters = append(whereFilters, "(user_id, channel) > (@userId, @channel)")

		var unmarsalledKey recipientNotificationStatusKey

		err := registry.UnmarshalKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		if unmarsalledKey.NotificationId != notificationId {
			return page, fmt.Errorf("notification id in the token does not match the notification id in the request")
		}

		args["userId"] = unmarsalledKey.UserId
		args["channel"] = unmarsalledKey.Channel
	}

	if len(filters.Channels) > 0 {
		whereFilters = append(whereFilters, "channel = ANY(@channels)")
		args["channels"] = filters.Channels
	}

	if len(filters.Statuses) > 0 {
		whereFilters = append(whereFilters, "status = ANY(@statuses)")
		args["statuses"] = filters.Statuses
	}

	whereStmt := strings.Join(whereFilters, " AND ")

	query := fmt.Sprintf(geRecipientNotificationStatuses, whereStmt)

	rows, err := r.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	collectedStatuses, err := pgx.CollectRows(rows, pgx.RowToStructByName[recipientNotificationStatus])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	statuses := make([]sdto.RecipientNotificationStatus, 0, len(collectedStatuses))

	for _, s := range collectedStatuses {
		statuses = append(statuses, sdto.RecipientNotificationStatus{
			UserId:  s.UserId,
			Channel: s.Channel,
			Status:  s.Status,
			ErrMsg:  s.ErrorMessage,
		})
	}

	numStatuses := len(statuses)

	if numStatuses == args["limit"] {
		lastStatus := statuses[numStatuses-1]
		lastKey := recipientNotificationStatusKey{
			UserId:         lastStatus.UserId,
			NotificationId: notificationId,
			Channel:        lastStatus.Channel,
		}

		key, err := registry.MarshalKey(lastKey)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(statuses)
	page.Data = statuses

	return page, nil
}
