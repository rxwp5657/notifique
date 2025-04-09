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

type userNotification struct {
	Id        string     `db:"id"`
	Title     string     `db:"title"`
	Contents  string     `db:"contents"`
	CreatedAt time.Time  `db:"created_at"`
	ImageUrl  *string    `db:"image_url"`
	ReadAt    *time.Time `db:"read_at"`
	Topic     string     `db:"topic"`
}

type userNotificationKey struct {
	Id     string `json:"id"`
	UserId string `json:"userId"`
}

func (n *userNotification) toDTO() dto.UserNotification {
	var readAt *string = nil

	if n.ReadAt != nil {
		parsed := n.ReadAt.Format(time.RFC3339Nano)
		readAt = &parsed
	}

	notification := dto.UserNotification{
		Id:        n.Id,
		Title:     n.Title,
		Contents:  n.Contents,
		CreatedAt: n.CreatedAt.Format(time.RFC3339Nano),
		Image:     n.ImageUrl,
		ReadAt:    readAt,
		Topic:     n.Topic,
	}

	return notification
}

const GetUserNotifications = `
SELECT
	id,
	title,
	contents,
	created_at,
	image_url,
	read_at,
	topic
FROM
	user_notifications
%s
ORDER BY
	id DESC
LIMIT
	@limit;
`

const UpdateReadAt = `
UPDATE
	user_notifications
SET
	read_at = NOW()
WHERE
	id = @id AND
	user_id = @userId
RETURNING
	id;
`

const insertUserNotification = `
INSERT INTO
	user_notifications(
	user_id,
	id,
	title,
	contents,
	created_at,
	image_url,
	read_at,
	topic
) VALUES (
 	@userId,
	@id,
	@title,
	@contents,
	NOW(),
	@imageUrl,
	@readAt,
	@topic
);
`

func (ps *Registry) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (sdto.Page[dto.UserNotification], error) {

	page := sdto.Page[dto.UserNotification]{}

	args := pgx.NamedArgs{"limit": internal.PageSize}
	whereFilters := []string{}

	if filters.MaxResults != nil {
		args["limit"] = *filters.MaxResults
	}

	if filters.NextToken != nil {
		nextTokenFilter := "(user_id, id) < (@user_id, @id)"
		whereFilters = append(whereFilters, nextTokenFilter)

		var unmarsalledKey userNotificationKey
		err := registry.UnmarshalKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		if unmarsalledKey.UserId != filters.UserId {
			return page, fmt.Errorf("invalid next token %s", *filters.NextToken)
		}

		args["id"] = unmarsalledKey.Id
		args["user_id"] = unmarsalledKey.UserId
	} else {
		whereFilters = append(whereFilters, "user_id = @user_id")
		args["user_id"] = filters.UserId
	}

	if len(filters.Topics) != 0 {
		whereFilters = append(whereFilters, "topic = ANY(@topics)")
		args["topics"] = filters.Topics
	}

	whereStmt := strings.Join(whereFilters, " AND ")

	if len(whereStmt) != 0 {
		whereStmt = fmt.Sprintf("WHERE %s", whereStmt)
	}

	query := fmt.Sprintf(GetUserNotifications, whereStmt)

	rows, err := ps.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	notifications, err := pgx.CollectRows(rows, pgx.RowToStructByName[userNotification])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	userNotifications := make([]dto.UserNotification, 0, len(notifications))

	for _, n := range notifications {
		userNotifications = append(userNotifications, n.toDTO())
	}

	numUserNotifications := len(userNotifications)

	if numUserNotifications == args["limit"] {
		lastNotification := userNotifications[numUserNotifications-1]

		lastNotificationKey := userNotificationKey{
			Id:     lastNotification.Id,
			UserId: filters.UserId,
		}

		key, err := registry.MarshalKey(lastNotificationKey)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(userNotifications)
	page.Data = userNotifications

	return page, nil
}

func (ps *Registry) SetReadStatus(ctx context.Context, userId, notificationId string) error {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"id":     notificationId,
		"userId": userId,
	}

	var nId uuid.UUID

	err = tx.QueryRow(ctx, UpdateReadAt, args).Scan(&nId)

	if err != nil {
		tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return internal.EntityNotFound{
				Id:   notificationId,
				Type: registry.NotificationType,
			}
		}

		return fmt.Errorf("failed to update notification - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("commit failed - %w", err)
	}

	return nil
}

func (r *Registry) CreateNotifications(ctx context.Context, notifications []sdto.UserNotificationReq) ([]dto.UserNotification, error) {

	userNotifications := make([]dto.UserNotification, 0, len(notifications))

	if len(notifications) == 0 {
		return userNotifications, nil
	}

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return userNotifications, fmt.Errorf("failed to start transaction - %w", err)
	}

	args := make([]pgx.NamedArgs, 0, len(notifications))

	for _, n := range notifications {

		id, err := uuid.NewV7()

		if err != nil {
			return userNotifications, fmt.Errorf("failed to generate uuid - %w", err)
		}

		args = append(args, pgx.NamedArgs{
			"id":       id.String(),
			"userId":   n.UserId,
			"title":    n.Title,
			"contents": n.Contents,
			"imageUrl": n.Image,
			"readAt":   nil,
			"topic":    n.Topic,
		})

		userNotification := dto.UserNotification{
			Id:        id.String(),
			Title:     n.Title,
			Contents:  n.Contents,
			CreatedAt: time.Now().Format(time.RFC3339Nano),
			Image:     n.Image,
			ReadAt:    nil,
			Topic:     n.Topic,
		}

		userNotifications = append(userNotifications, userNotification)
	}

	err = batchInsert(ctx, insertUserNotification, args, tx)

	if err != nil {
		rallbackErr := tx.Rollback(ctx)
		err = errors.Join(err, rallbackErr)
		return userNotifications, fmt.Errorf("failed to batch insert user notifications - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return userNotifications, fmt.Errorf("commit failed - %w", err)
	}

	return userNotifications, nil
}
