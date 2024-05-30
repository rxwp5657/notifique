package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/notifique/dto"
)

type PostgresStorage struct {
	conn *pgxpool.Pool
}

type namedArgsBuilder[T any] func(val T) pgx.NamedArgs

func (ps *PostgresStorage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

	page := dto.Page[dto.UserNotification]{}

	args := pgx.NamedArgs{}
	whereFilters := make([]string, 0)

	limit := 25

	if filters.MaxResults != nil {
		limit = *filters.MaxResults
	}

	limit += 1
	args["limit"] = limit

	if filters.NextToken != nil {
		nextTokenFilter := "(id, user_id, created_at) <= (@id, @user_id, @created_at)"
		whereFilters = append(whereFilters, nextTokenFilter)

		unmarsalledKey, err := unmarshallNotificationKey(*filters.NextToken)

		if err != nil {
			return page, err
		}

		if unmarsalledKey.UserId != filters.UserId {
			return page, fmt.Errorf("invalid next token %s", *filters.NextToken)
		}

		args["id"] = unmarsalledKey.Id
		args["user_id"] = unmarsalledKey.UserId
		args["created_at"] = unmarsalledKey.CreatedAt
	}

	if len(filters.Topics) != 0 {
		whereFilters = append(whereFilters, "topics ANY (@topics)")
		args["topics"] = filters.Topics
	}

	whereStmt := strings.Join(whereFilters, "AND")

	if len(whereStmt) != 0 {
		whereStmt = fmt.Sprintf("WHERE %s", whereStmt)
	}

	query := fmt.Sprintf(GET_USER_NOTIFICATIONS, whereStmt)

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

	if numUserNotifications == limit {
		lastNotification := userNotifications[numUserNotifications-1]

		lastNotificationKey := userNotificationKey{
			Id:        lastNotification.Id,
			UserId:    filters.UserId,
			CreatedAt: lastNotification.CreatedAt,
		}

		key, err := lastNotificationKey.marshal()

		if err != nil {
			return page, err
		}

		page.NextToken = &key
		userNotifications = userNotifications[:numUserNotifications-1]
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(userNotifications)
	page.Data = userNotifications

	return page, nil
}

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

func (ps *PostgresStorage) SaveNotification(ctx context.Context, createdBy string, notification dto.NotificationReq) (string, error) {

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
	}

	var notificationId uuid.UUID

	err = tx.QueryRow(ctx, INSERT_NOTIFICATION, notificationArgs).Scan(&notificationId)

	if err != nil {
		tx.Rollback(ctx)
		return "", fmt.Errorf("failed to insert notification - %w", err)
	}

	recipientsBuilder := func(recipient string) pgx.NamedArgs {
		return pgx.NamedArgs{
			"notificationId": notificationId.String(),
			"recipient":      recipient,
		}
	}

	err = batchInsert(ctx, INSERT_RECIPIENTS, notification.Recipients, recipientsBuilder, tx)

	if err != nil {
		return "", err
	}

	channelsBuilder := func(channel string) pgx.NamedArgs {
		return pgx.NamedArgs{
			"notificationId": notificationId.String(),
			"channel":        channel,
		}
	}

	err = batchInsert(ctx, INSERT_CHANNELS, notification.Channels, channelsBuilder, tx)

	if err != nil {
		return "", err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to commit notification insert - %w", err)
	}

	return notificationId.String(), nil
}

func (ps *PostgresStorage) makeUserConfig(ctx context.Context, userId string) (*userConfig, error) {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start transaction - %w", err)
	}

	cfg := userConfig{
		EmailOptIn: true,
		SMSOptIn:   true,
		PushOptIn:  true,
		InAppOptIn: true,
	}

	args := pgx.NamedArgs{
		"userId":           userId,
		"emailOptIn":       cfg.EmailOptIn,
		"emailSnoozeUntil": cfg.EmailSnoozeUntil,
		"smsOptIn":         cfg.SMSOptIn,
		"smsSoozeUntil":    cfg.smsSoozeUntil,
		"inAppOptIn":       cfg.InAppOptIn,
		"inAppSnoozeUntil": cfg.InAppSnoozeUntil,
		"pushOptIn":        cfg.PushOptIn,
		"pushSnoozeUntil":  cfg.PushSnoozeUntil,
	}

	_, err = tx.Exec(ctx, INSERT_USER_CONFIG, args)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to inser user config - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to commit user config - %w", err)
	}

	return &cfg, nil
}

func (ps *PostgresStorage) GetUserConfig(ctx context.Context, userId string) (dto.UserConfig, error) {

	args := pgx.NamedArgs{"userId": userId}

	var cfg userConfig

	err := ps.conn.QueryRow(ctx, GET_USER_CONFIG, args).Scan(
		&cfg.EmailOptIn,
		&cfg.EmailSnoozeUntil,
		&cfg.SMSOptIn,
		&cfg.smsSoozeUntil,
		&cfg.InAppOptIn,
		&cfg.InAppSnoozeUntil,
		&cfg.PushOptIn,
		&cfg.PushSnoozeUntil,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			newCfg, err := ps.makeUserConfig(ctx, userId)

			if err != nil {
				return dto.UserConfig{}, err
			}

			cfg = *newCfg
		}
	}

	return cfg.toDTO(), nil
}

func (ps *PostgresStorage) UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"userId":           userId,
		"emailOptIn":       config.EmailConfig.OptIn,
		"emailSnoozeUntil": config.EmailConfig.SnoozeUntil,
		"smsOptIn":         config.SMSConfig.OptIn,
		"smsSoozeUntil":    config.SMSConfig.SnoozeUntil,
		"inAppOptIn":       config.InAppConfig.OptIn,
		"inAppSnoozeUntil": config.InAppConfig.SnoozeUntil,
		"pushOptIn":        true,
		"pushSnoozeUntil":  nil,
	}

	_, err = tx.Exec(ctx, UPSERT_USER_CONFIG, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to upsert user config - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit user config update - %w", err)
	}

	return nil
}

func (ps *PostgresStorage) SetReadStatus(ctx context.Context, userId, notificationId string) error {
	return nil
}

func (ps *PostgresStorage) CreateDistributionList(ctx context.Context, distributionList dto.DistributionList) error {
	return nil
}

func (ps *PostgresStorage) GetDistributionLists(ctx context.Context, filter dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {
	return dto.Page[dto.DistributionListSummary]{}, nil
}

func (ps *PostgresStorage) DeleteDistributionList(ctx context.Context, distlistName string) error {
	return nil
}

func (ps *PostgresStorage) GetRecipients(ctx context.Context, distlistName string, filter dto.PageFilter) (dto.Page[string], error) {
	return dto.Page[string]{}, nil
}

func (ps *PostgresStorage) AddRecipients(ctx context.Context, distlistName string, recipients []string) (*dto.DistributionListSummary, error) {
	return nil, nil
}

func (ps *PostgresStorage) DeleteRecipients(ctx context.Context, distlistName string, recipients []string) (*dto.DistributionListSummary, error) {
	return nil, nil
}

func (ps *PostgresStorage) CreateUserNotification(ctx context.Context, userId string, un dto.UserNotification) error {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"id":        un.Id,
		"userId":    userId,
		"title":     un.Title,
		"contents":  un.Contents,
		"createdAt": un.CreatedAt,
		"imageUrl":  un.Image,
		"topic":     un.Topic,
		"readAt":    un.ReadAt,
	}

	_, err = tx.Exec(ctx, INSERT_USER_NOTIFICATION, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert user notifications - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit user notification insert - %w", err)
	}

	return nil
}

func (ps *PostgresStorage) DeleteUserNotification(ctx context.Context, userId string, un dto.UserNotification) error {
	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"id": un.Id,
	}

	_, err = tx.Exec(ctx, DELETE_USER_NOTIFICATION, args)

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

func MakePostgresStorage(url string) (*PostgresStorage, error) {
	conn, err := pgxpool.New(context.TODO(), url)

	if err != nil {
		return nil, fmt.Errorf("failed to create pool - %w", err)
	}

	return &PostgresStorage{conn: conn}, nil
}
