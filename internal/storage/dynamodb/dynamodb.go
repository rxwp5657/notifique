package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/notifique/dto"
)

const (
	NOTIFICATION_TABLE               = "notifications"
	USER_CONFIG_TABLE                = "userConfig"
	USER_NOTIFICATIONS_TABLE         = "userNotifications"
	USER_NOTIFICATIONS_SECONDARY_IDX = "createdAtIndex"
	DISTRIBUTION_LISTS_TABLE         = "distributionLists"
)

type DynamoDBStorage struct {
	client *dynamodb.Client
}

type DynamoDBKey map[string]types.AttributeValue

func (s *DynamoDBStorage) getUserConfig(ctx context.Context, userId string) (*userConfig, error) {

	tmpConfig := userConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return nil, err
	}

	resp, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key: key, TableName: aws.String(USER_CONFIG_TABLE),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get the user config - %w", err)
	}

	if resp == nil {
		return nil, nil
	}

	config := userConfig{}

	err = attributevalue.UnmarshalMap(resp.Item, &config)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall the user config - %w", err)
	}

	return &config, nil
}

func (s *DynamoDBStorage) createUserConfig(ctx context.Context, userId string) (*userConfig, error) {
	config := userConfig{
		UserId:      userId,
		EmailConfig: channelConfig{OptIn: true, SnoozeUntil: nil},
		SMSConfig:   channelConfig{OptIn: true, SnoozeUntil: nil},
	}

	item, err := attributevalue.MarshalMap(config)

	if err != nil {
		return nil, fmt.Errorf("failed to marshall the user config - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(USER_CONFIG_TABLE),
		Item:      item,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to store notification - %w", err)
	}

	return &config, nil
}

func (s *DynamoDBStorage) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	id := uuid.NewString()

	notification := notification{
		Id:               id,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().Format(time.RFC3339),
		Title:            notificationReq.Title,
		Contents:         notificationReq.Contents,
		Image:            notificationReq.Image,
		Topic:            notificationReq.Topic,
		Priority:         notificationReq.Priority,
		DistributionList: notificationReq.DistributionList,
		Recipients:       notificationReq.Recipients,
		Channels:         notificationReq.Channels,
		Logs:             []notificationLog{},
	}

	item, err := attributevalue.MarshalMap(notification)

	if err != nil {
		return "", fmt.Errorf("failed to marshall notification - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(NOTIFICATION_TABLE),
		Item:      item,
	})

	if err != nil {
		return "", fmt.Errorf("failed to store notification - %w", err)
	}

	return id, nil
}

func (s *DynamoDBStorage) CreateUserNotification(ctx context.Context, userId string, notificationReq dto.UserNotificationReq) (string, error) {

	id := uuid.NewString()

	notification := userNotification{
		Id:        id,
		UserId:    userId,
		Title:     notificationReq.Title,
		Contents:  notificationReq.Contents,
		CreatedAt: time.Now().Format(time.RFC3339),
		Image:     notificationReq.Image,
		ReadAt:    nil,
		Topic:     notificationReq.Topic,
	}

	item, err := attributevalue.MarshalMap(notification)

	if err != nil {
		return "", fmt.Errorf("failed to marshall user notification - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(USER_NOTIFICATIONS_TABLE),
		Item:      item,
	})

	if err != nil {
		return "", fmt.Errorf("failed to store user notification - %w", err)
	}

	return id, nil
}

func (s *DynamoDBStorage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

	userId := &types.AttributeValueMemberS{Value: filters.UserId}
	keyExp := expression.Key("userId").Equal(expression.Value(userId))
	builder := expression.NewBuilder().WithKeyCondition(keyExp)

	topicFilters := make([]expression.OperandBuilder, 0)

	for _, topic := range filters.Topics {
		topicFilters = append(topicFilters, expression.Value(topic))
	}

	if len(topicFilters) > 0 {
		first := topicFilters[0]
		rest := make([]expression.OperandBuilder, 0)

		if len(topicFilters) > 1 {
			rest = topicFilters[1:]
		}

		in := expression.In(expression.Name("topic"), first, rest...)
		builder.WithCondition(in)
	}

	expr, err := builder.Build()

	page := dto.Page[dto.UserNotification]{}

	if err != nil {
		return page, fmt.Errorf("failed to build query - %w", err)
	}

	queryPaginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(USER_NOTIFICATIONS_TABLE),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		IndexName:                 aws.String(USER_NOTIFICATIONS_SECONDARY_IDX),
	})

	notifications := make([]userNotification, 0)

	for queryPaginator.HasMorePages() {
		response, err := queryPaginator.NextPage(ctx)

		if err != nil {
			return page, fmt.Errorf("failed to get page - %w", err)
		}

		var notificationsPage []userNotification
		err = attributevalue.UnmarshalListOfMaps(response.Items, &notificationsPage)

		if err != nil {
			return page, fmt.Errorf("failed to unmarshall user notifications %w", err)
		}

		notifications = append(notifications, notificationsPage...)
	}

	result := make([]dto.UserNotification, 0, len(notifications))

	for _, notification := range notifications {
		un := dto.UserNotification{
			Id:        notification.Id,
			Title:     notification.Title,
			Contents:  notification.Contents,
			CreatedAt: notification.CreatedAt,
			Image:     notification.Image,
			ReadAt:    notification.ReadAt,
			Topic:     notification.Topic,
		}

		result = append(result, un)
	}

	page.Data = result

	return page, nil
}

func (s *DynamoDBStorage) GetUserConfig(ctx context.Context, userId string) ([]dto.ChannelConfig, error) {

	config, err := s.getUserConfig(ctx, userId)

	if err != nil {
		return []dto.ChannelConfig{}, err
	}

	if config == nil {
		config, err = s.createUserConfig(ctx, userId)

		if err != nil {
			return []dto.ChannelConfig{}, err
		}
	}

	chCfg := []dto.ChannelConfig{
		{
			Channel:     "e-mail",
			OptIn:       config.EmailConfig.OptIn,
			SnoozeUntil: config.EmailConfig.SnoozeUntil,
		},
		{
			Channel:     "sms",
			OptIn:       config.EmailConfig.OptIn,
			SnoozeUntil: config.EmailConfig.SnoozeUntil,
		},
	}

	return chCfg, nil
}

func (s *DynamoDBStorage) SetReadStatus(ctx context.Context, userId, notificationId string) error {

	reatAt := time.Now().Format(time.RFC3339)
	update := expression.Set(expression.Name("readAt"), expression.Value(reatAt))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	tmpNotification := userNotification{Id: notificationId}
	key, err := tmpNotification.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(USER_NOTIFICATIONS_TABLE),
		Key:                       key,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return fmt.Errorf("failed to set the read status - %w", err)
	}

	return nil
}

func (s *DynamoDBStorage) UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error {

	usrCfg, err := s.getUserConfig(ctx, userId)

	if err != nil {
		return err
	}

	if usrCfg == nil {
		_, err := s.createUserConfig(ctx, userId)

		if err != nil {
			return err
		}
	}

	builder := expression.NewBuilder()

	makeUpdateExpr := func(key string, cfg dto.ChannelConfig) expression.UpdateBuilder {
		update := expression.Set(
			expression.Name(fmt.Sprintf("%v.optIn", key)),
			expression.Value(cfg.OptIn),
		)
		update.Set(
			expression.Name(fmt.Sprintf("%v.optIn", key)),
			expression.Value(cfg.SnoozeUntil),
		)

		return update
	}

	for _, config := range config.Config {
		switch config.Channel {
		case "e-mail":
			update := makeUpdateExpr("emailConfig", config)
			builder.WithUpdate(update)
		case "sms":
			update := makeUpdateExpr("smsConfig", config)
			builder.WithUpdate(update)
		}
	}

	expr, err := builder.Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	tmpConfig := userConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(USER_CONFIG_TABLE),
		Key:                       key,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return fmt.Errorf("failed to set the read status - %w", err)
	}

	return nil
}

func (s *DynamoDBStorage) CreateDistributionList(ctx context.Context, distributionList dto.DistributionList) error {
	return nil
}

func (s *DynamoDBStorage) GetDistributionLists(ctx context.Context, filter dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {
	page := dto.Page[dto.DistributionListSummary]{}

	return page, nil
}

func (s *DynamoDBStorage) DeleteDistributionList(ctx context.Context, distlistName string) error {
	return nil
}

func (s *DynamoDBStorage) GetRecipients(ctx context.Context, distlistName string, filter dto.PageFilter) (dto.Page[string], error) {
	return dto.Page[string]{}, nil
}

func (s *DynamoDBStorage) AddRecipients(ctx context.Context, distlistName string, recipients []string) (dto.DistributionListSummary, error) {
	return dto.DistributionListSummary{}, nil
}

func (s *DynamoDBStorage) DeleteRecipients(ctx context.Context, distlistName string, recipients []string) (dto.DistributionListSummary, error) {
	return dto.DistributionListSummary{}, nil
}

func MakeDynamoDBStorage() DynamoDBStorage {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Fatal(err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:8000")
	})

	return DynamoDBStorage{client: client}
}
