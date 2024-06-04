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
	"github.com/notifique/internal"

	c "github.com/notifique/controllers"
)

type PostgresStorage struct {
	conn *pgxpool.Pool
}

type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type namedArgsBuilder[T any] func(val T) pgx.NamedArgs

func (ps *PostgresStorage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

	page := dto.Page[dto.UserNotification]{}

	args := pgx.NamedArgs{"limit": 25}
	whereFilters := make([]string, 0)

	if filters.MaxResults != nil {
		limit := *filters.MaxResults
		args["limit"] = limit
	}

	if filters.NextToken != nil {
		nextTokenFilter := "(id, user_id, created_at) <= (@id, @user_id, @created_at)"
		whereFilters = append(whereFilters, nextTokenFilter)

		var unmarsalledKey userNotificationKey
		err := unmarsallKey(*filters.NextToken, &unmarsalledKey)

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

	if numUserNotifications == args["limit"] {
		lastNotification := userNotifications[numUserNotifications-1]

		lastNotificationKey := userNotificationKey{
			Id:        lastNotification.Id,
			UserId:    filters.UserId,
			CreatedAt: lastNotification.CreatedAt,
		}

		key, err := marshallKey(lastNotificationKey)

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

func addNotificationStatusLog(ctx context.Context, log notificationStatusLog, tx pgx.Tx) error {

	args := pgx.NamedArgs{
		"notificationId": log.NotificationId,
		"statusDate":     log.StatusDate,
		"status":         log.Status,
		"errorMessage":   log.Error,
	}

	_, err := tx.Exec(ctx, INSERT_NOTIFICATION_STATUS_LOG, args)

	if err != nil {
		return fmt.Errorf("failed to insert notification status log - %w", err)
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
		"status":           string(c.CREATED),
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

	err = batchInsert(
		ctx,
		INSERT_NOTIFICATION_RECIPIENTS,
		notification.Recipients,
		recipientsBuilder,
		tx,
	)

	if err != nil {
		return "", err
	}

	channelsBuilder := func(channel string) pgx.NamedArgs {
		return pgx.NamedArgs{
			"notificationId": notificationId.String(),
			"channel":        channel,
		}
	}

	err = batchInsert(
		ctx,
		INSERT_CHANNELS,
		notification.Channels,
		channelsBuilder,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return "", err
	}

	statusLog := notificationStatusLog{
		NotificationId: notificationId.String(),
		Status:         string(c.CREATED),
		StatusDate:     time.Now(),
		Error:          nil,
	}

	err = addNotificationStatusLog(ctx, statusLog, tx)

	if err != nil {
		tx.Rollback(ctx)
		return "", fmt.Errorf("failed to create notification status log - %w", err)
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

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"id":     notificationId,
		"userId": userId,
	}

	var nId uuid.UUID

	err = tx.QueryRow(ctx, UPDATE_READ_AT, args).Scan(&nId)

	if err != nil {
		tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return internal.NotificationNotFound{
				NotificationId: notificationId,
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

func getDistributionListSummary(ctx context.Context, listName string, rQuerier RowQuerier) (*dto.DistributionListSummary, error) {

	args := pgx.NamedArgs{"name": listName}

	var summary dto.DistributionListSummary

	err := rQuerier.QueryRow(ctx, GET_DISTRIBUTION_LIST, args).Scan(
		&summary.Name,
		&summary.NumberOfRecipients,
	)

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get distribution list - %w", err)
	}

	return &summary, nil
}

func (ps *PostgresStorage) CreateDistributionList(ctx context.Context, distributionList dto.DistributionList) error {

	list, err := getDistributionListSummary(ctx, distributionList.Name, ps.conn)

	if err != nil {
		return err
	}

	if list != nil {
		return internal.DistributionListAlreadyExists{
			Name: list.Name,
		}
	}

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"name":          distributionList.Name,
		"numRecipients": len(distributionList.Recipients),
	}

	_, err = tx.Exec(ctx, INSERT_DISTRIBUTION_LIST, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to create distribution list - %w", err)
	}

	recipientBuilder := func(recipient string) pgx.NamedArgs {
		return pgx.NamedArgs{
			"name":      distributionList.Name,
			"recipient": recipient,
		}
	}

	err = batchInsert(
		ctx,
		INSERT_DISTRIBUTION_LIST_RECIPIENT,
		distributionList.Recipients,
		recipientBuilder,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert distribution list recipients - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("commit failed - %w", err)
	}

	return nil
}

func (ps *PostgresStorage) GetDistributionLists(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

	page := dto.Page[dto.DistributionListSummary]{}

	args := pgx.NamedArgs{"limit": 25}

	nextTokenFilter := ""

	if filters.MaxResults != nil {
		limit := *filters.MaxResults
		args["limit"] = limit
	}

	if filters.NextToken != nil {
		nextTokenFilter = `WHERE ("name") > (@name)`

		var unmarsalledKey distributionListKey
		err := unmarsallKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		args["name"] = unmarsalledKey.Name
	}

	query := fmt.Sprintf(GET_DISTRIBUTION_LISTS, nextTokenFilter)
	rows, err := ps.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	summaries, err := pgx.CollectRows(rows, pgx.RowToStructByName[distributionListSummary])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	numSummaries := len(summaries)

	if numSummaries == args["limit"] {
		lastSummary := summaries[numSummaries-1]

		lastSummaryKey := distributionListKey{
			Name: lastSummary.Name,
		}

		key, err := marshallKey(lastSummaryKey)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	dtoSummaries := make([]dto.DistributionListSummary, 0, len(summaries))

	for _, summary := range summaries {
		s := dto.DistributionListSummary{
			Name:               summary.Name,
			NumberOfRecipients: summary.NumberOfRecipients,
		}

		dtoSummaries = append(dtoSummaries, s)
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(dtoSummaries)
	page.Data = dtoSummaries

	return page, nil
}

func (ps *PostgresStorage) DeleteDistributionList(ctx context.Context, distlistName string) error {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{"name": distlistName}

	// Should also delete its recipients
	_, err = tx.Exec(ctx, DELETE_DISTRIBUTION_LIST, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to delete distribution list - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("commit failed - %w", err)
	}

	return nil
}

func (ps *PostgresStorage) GetRecipients(ctx context.Context, distlistName string, filters dto.PageFilter) (dto.Page[string], error) {

	page := dto.Page[string]{}

	summary, err := getDistributionListSummary(ctx, distlistName, ps.conn)

	if err != nil {
		return page, fmt.Errorf("failed to get summary - %w", err)
	}

	if summary == nil {
		return page, internal.DistributionListNotFound{
			Name: distlistName,
		}
	}

	args := pgx.NamedArgs{"limit": 25}
	whereFilters := make([]string, 0)

	if filters.MaxResults != nil {
		limit := *filters.MaxResults
		args["limit"] = limit
	}

	if filters.NextToken != nil {
		filter := `("name", recipient) > (@name, @recipient)`
		whereFilters = append(whereFilters, filter)

		var unmarsalledKey distributionList
		err := unmarsallKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		if unmarsalledKey.Name != distlistName {
			return page, fmt.Errorf("invalid key %s", *filters.NextToken)
		}

		args["name"] = unmarsalledKey.Name
		args["recipient"] = unmarsalledKey.Recipient
	} else {
		filter := "name = @name"
		whereFilters = append(whereFilters, filter)
		args["name"] = distlistName
	}

	whereStmt := strings.Join(whereFilters, "AND")
	query := fmt.Sprintf(GET_DISTRIBUTION_LIST_RECIPIENTS, whereStmt)

	rows, err := ps.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	recipients, err := pgx.CollectRows(rows, pgx.RowToStructByName[recipient])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	numRecipients := len(recipients)

	if numRecipients == args["limit"] {
		lastRecipient := recipients[numRecipients-1]

		dl := distributionList{
			Name:      distlistName,
			Recipient: lastRecipient.Recipient,
		}

		key, err := marshallKey(dl)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	recipientsNames := make([]string, 0, len(recipients))

	for _, r := range recipients {
		recipientsNames = append(recipientsNames, r.Recipient)
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(recipients)
	page.Data = recipientsNames

	return page, nil
}

func (ps *PostgresStorage) AddRecipients(ctx context.Context, distlistName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := getDistributionListSummary(ctx, distlistName, ps.conn)

	if err != nil {
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	if exists == nil {
		return nil, internal.DistributionListNotFound{
			Name: distlistName,
		}
	}

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start transaction - %w", err)
	}

	recipientBuilder := func(recipient string) pgx.NamedArgs {
		return pgx.NamedArgs{
			"name":      distlistName,
			"recipient": recipient,
		}
	}

	err = batchInsert(
		ctx,
		INSERT_DISTRIBUTION_LIST_RECIPIENT,
		recipients,
		recipientBuilder,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed add recipients - %w", err)
	}

	summary, err := getDistributionListSummary(ctx, distlistName, tx)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	countArgs := pgx.NamedArgs{
		"numRecipients": summary.NumberOfRecipients,
		"name":          distlistName,
	}

	_, err = tx.Exec(ctx, UPDATE_RECIPIENTS_COUNT, countArgs)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to update recipients count - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("commit failed - %w", err)
	}

	return summary, nil
}

func (ps *PostgresStorage) DeleteRecipients(ctx context.Context, distlistName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := getDistributionListSummary(ctx, distlistName, ps.conn)

	if err != nil {
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	if exists == nil {
		return nil, internal.DistributionListNotFound{
			Name: distlistName,
		}
	}

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"name":       distlistName,
		"recipients": recipients,
	}

	_, err = tx.Exec(ctx, DELETE_DISTRIBUTION_LIST_RECIPIENTS, args)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed delete recipients - %w", err)
	}

	summary, err := getDistributionListSummary(ctx, distlistName, tx)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	countArgs := pgx.NamedArgs{
		"numRecipients": summary.NumberOfRecipients,
		"name":          distlistName,
	}

	_, err = tx.Exec(ctx, UPDATE_RECIPIENTS_COUNT, countArgs)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to update recipients count - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to commit delete recipients - %w", err)
	}

	return summary, nil
}

func (ps *PostgresStorage) CreateNotificationStatusLog(ctx context.Context, notificationId string, status c.NotificationStatus, errMsg *string) error {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"notificationId": notificationId,
		"statusDate":     time.Now().Format(time.RFC3339Nano),
		"status":         status,
		"errorMessage":   errMsg,
	}

	_, err = tx.Exec(ctx, INSERT_NOTIFICATION_STATUS_LOG, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert notification status log - %w", err)
	}

	updateArgs := pgx.NamedArgs{
		"status": string(status),
		"id":     notificationId,
	}

	_, err = tx.Exec(ctx, UPDATE_NOTIFICATION_STATUS, updateArgs)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to update notification status - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit notification status log - %w", err)
	}

	return nil
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
