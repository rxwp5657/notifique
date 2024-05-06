package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	NOTIFICATION_TABLE           = "notifications"
	USER_CONFIG_TABLE            = "userConfig"
	USER_NOTIFICATIONS_TABLE     = "userNotifications"
	USER_NOTIFICATIONS_TOPIC_IDX = "topicIndex"
	DISTRIBUTION_LISTS_TABLE     = "distributionLists"
)

type DynamoDBStorage struct {
	client *dynamodb.Client
}

type dynamodbPrimaryKey interface {
	GetKey() (DynamoDBKey, error)
}

type DynamoDBKey map[string]types.AttributeValue

func marshallNextToken[T any](key *T, response *dynamodb.QueryOutput) (string, error) {
	err := attributevalue.UnmarshalMap(response.LastEvaluatedKey, &key)

	if err != nil {
		return "", fmt.Errorf("failed to unmarshall last evaluated key - %w", err)
	}

	jsonMarshalled, err := json.Marshal(key)

	if err != nil {
		return "", fmt.Errorf("failed to json marshall last evaluated key - %w", err)
	}

	return base64.StdEncoding.EncodeToString([]byte(jsonMarshalled)), nil
}

func unmarshallNextToken[T any](nextToken string, key *T) error {
	decoded, err := base64.StdEncoding.DecodeString(nextToken)

	if err != nil {
		return fmt.Errorf("failed to decode base64 nextToken - %w", err)
	}

	err = json.Unmarshal(decoded, &key)

	if err != nil {
		return fmt.Errorf("failed to json unmarshall nextToken - %w", err)
	}

	return nil
}

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

func makeTopicsFilter(topics []string) *expression.ConditionBuilder {

	if len(topics) == 0 {
		return nil
	}

	topicFilters := make([]expression.OperandBuilder, 0, len(topics))

	for _, topic := range topics {
		topicFilters = append(topicFilters, expression.Value(topic))
	}

	first := topicFilters[0]
	rest := make([]expression.OperandBuilder, 0)

	if len(topicFilters) > 1 {
		rest = topicFilters[1:]
	}

	cond := expression.In(expression.Name("topic"), first, rest...)
	return &cond
}

func addPageFilters[T dynamodbPrimaryKey](key T, qi *dynamodb.QueryInput, filters dto.PageFilter) error {

	if filters.MaxResults != nil {
		limit := int32(*filters.MaxResults)
		qi.Limit = &limit
	}

	if filters.NextToken != nil {
		err := unmarshallNextToken(*filters.NextToken, &key)

		if err != nil {
			return fmt.Errorf("failed to unmarshall token - %w", err)
		}

		dynamoDBKey, err := key.GetKey()

		if err != nil {
			return fmt.Errorf("failed to get model key - %w", err)
		}

		qi.ExclusiveStartKey = dynamoDBKey
	}

	return nil
}

func (s *DynamoDBStorage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

	page := dto.Page[dto.UserNotification]{}

	keyExp := expression.Key("userId").Equal(expression.Value(filters.UserId))
	builder := expression.NewBuilder().WithKeyCondition(keyExp)

	topicsFilter := makeTopicsFilter(filters.Topics)

	if topicsFilter != nil {
		builder.WithFilter(*topicsFilter)
	}

	expr, err := builder.Build()

	if err != nil {
		return page, fmt.Errorf("failed to build query - %w", err)
	}

	queryInput := dynamodb.QueryInput{
		TableName:                 aws.String(USER_NOTIFICATIONS_TABLE),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ScanIndexForward:          aws.Bool(false),
	}

	err = addPageFilters(&userNotificationKey{}, &queryInput, filters.PageFilter)

	if err != nil {
		return page, err
	}

	response, err := s.client.Query(ctx, &queryInput)

	if err != nil {
		return page, fmt.Errorf("failed to get notifications - %w", err)
	}

	var notifications []userNotification
	err = attributevalue.UnmarshalListOfMaps(response.Items, &notifications)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall user notifications %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := userNotificationKey{}
		encoded, err := marshallNextToken(&key, response)

		if err != nil {
			return page, err
		}

		nextToken = &encoded
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

	page.NextToken = nextToken
	page.PrevToken = filters.NextToken
	page.ResultCount = len(result)

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

func (s *DynamoDBStorage) addRecipients(ctx context.Context, recipients []distributionList) (int, error) {

	writeReq := make([]types.WriteRequest, 0, len(recipients))

	for _, recipient := range recipients {
		item, err := attributevalue.MarshalMap(recipient)

		if err != nil {
			return 0, fmt.Errorf("failed to marshal dl recipient - %w", err)
		}

		writeReq = append(writeReq, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		})
	}

	requestItems := map[string][]types.WriteRequest{
		DISTRIBUTION_LISTS_TABLE: writeReq,
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to add recipients to dl - %w", err)
	}

	return len(recipients), nil
}

func (s *DynamoDBStorage) CreateDistributionList(ctx context.Context, dlReq dto.DistributionList) error {

	recipients := make([]distributionList, 0, len(dlReq.Recipients))

	for _, recipient := range dlReq.Recipients {
		recipients = append(recipients, distributionList{
			Name:   dlReq.Name,
			UserId: recipient,
		})
	}

	_, err := s.addRecipients(ctx, recipients)

	return err
}

func (s *DynamoDBStorage) getDLSummary(ctx context.Context, listName string) (*dto.DistributionListSummary, error) {
	keyEx := expression.Key("name").Equal(expression.Value(listName))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()

	if err != nil {
		return nil, fmt.Errorf("failed to create expression - %w", err)
	}

	queryPaginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(DISTRIBUTION_LISTS_TABLE),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})

	count := 0
	for queryPaginator.HasMorePages() {
		resp, err := queryPaginator.NextPage(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to retrieve user page - %w", err)
		}

		count += int(resp.Count)
	}

	summary := dto.DistributionListSummary{
		Name:               listName,
		NumberOfRecipients: count,
	}

	return &summary, nil
}

func (s *DynamoDBStorage) GetDistributionLists(ctx context.Context, filter dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

	page := dto.Page[dto.DistributionListSummary]{}
	dlMap := make(map[string]int)

	scanPaginator := dynamodb.NewScanPaginator(s.client, &dynamodb.ScanInput{
		TableName: aws.String(DISTRIBUTION_LISTS_TABLE),
	})

	for scanPaginator.HasMorePages() {
		resp, err := scanPaginator.NextPage(ctx)

		if err != nil {
			return page, fmt.Errorf("failed to fetch distribution list page - %w", err)
		}

		var recipients []distributionList
		err = attributevalue.UnmarshalListOfMaps(resp.Items, &recipients)

		if err != nil {
			return page, fmt.Errorf("failed to unmarshall list of recipients - %w", err)
		}

		for _, r := range recipients {
			dlMap[r.Name] = dlMap[r.Name] + 1
		}
	}

	summaries := make([]dto.DistributionListSummary, 0, len(dlMap))

	for k, v := range dlMap {
		summary := dto.DistributionListSummary{
			Name:               k,
			NumberOfRecipients: v,
		}
		summaries = append(summaries, summary)
	}

	page.Data = summaries
	page.ResultCount = len(summaries)

	return page, nil
}

func (s *DynamoDBStorage) deleteRecipients(ctx context.Context, recipients []distributionList) (int, error) {

	deleteReq := make([]types.WriteRequest, 0, len(recipients))

	for _, recipient := range recipients {
		key, _ := recipient.GetKey()
		deleteReq = append(deleteReq, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: key,
			},
		})
	}

	requestItems := map[string][]types.WriteRequest{
		DISTRIBUTION_LISTS_TABLE: deleteReq,
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to delete recipients of dl - %w", err)
	}

	return len(recipients), nil
}

func (s *DynamoDBStorage) DeleteDistributionList(ctx context.Context, listName string) error {

	keyEx := expression.Key("name").Equal(expression.Value(listName))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()

	if err != nil {
		return fmt.Errorf("failed to create expression - %w", err)
	}

	queryPaginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(DISTRIBUTION_LISTS_TABLE),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})

	for queryPaginator.HasMorePages() {
		resp, err := queryPaginator.NextPage(ctx)

		if err != nil {
			return fmt.Errorf("failed to retrieve user page - %w", err)
		}

		var recipients []distributionList
		err = attributevalue.UnmarshalListOfMaps(resp.Items, &recipients)

		if err != nil {
			return fmt.Errorf("failed to unmarshal distribution list - %w", err)
		}

		_, err = s.deleteRecipients(ctx, recipients)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DynamoDBStorage) GetRecipients(ctx context.Context, distlistName string, filters dto.PageFilter) (dto.Page[string], error) {

	page := dto.Page[string]{}

	keyExp := expression.Key("name").Equal(expression.Value(distlistName))
	projExp := expression.NamesList(expression.Name("userId"))

	builder := expression.NewBuilder()
	expr, err := builder.WithKeyCondition(keyExp).WithProjection(projExp).Build()

	if err != nil {
		return page, fmt.Errorf("failed to build query - %w", err)
	}

	queryInput := dynamodb.QueryInput{
		TableName:                 aws.String(DISTRIBUTION_LISTS_TABLE),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		ProjectionExpression:      expr.Projection(),
	}

	err = addPageFilters(&userNotificationKey{}, &queryInput, filters)

	if err != nil {
		return page, err
	}

	response, err := s.client.Query(ctx, &queryInput)

	if err != nil {
		return page, fmt.Errorf("failed to get recipients - %w", err)
	}

	var recipients []distributionList
	err = attributevalue.UnmarshalListOfMaps(response.Items, &recipients)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall recipients - %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := distributionList{}
		encoded, err := marshallNextToken(&key, response)

		if err != nil {
			return page, err
		}

		nextToken = &encoded
	}

	result := make([]string, 0, len(recipients))

	for _, r := range recipients {
		result = append(result, r.UserId)
	}

	page.NextToken = nextToken
	page.PrevToken = filters.NextToken
	page.ResultCount = len(recipients)

	page.Data = result

	return page, nil
}

func (s *DynamoDBStorage) AddRecipients(ctx context.Context, listName string, recipients []string) (dto.DistributionListSummary, error) {

	dlRecipients := make([]distributionList, 0, len(recipients))

	for _, recipient := range recipients {
		dlRecipients = append(dlRecipients, distributionList{
			Name:   listName,
			UserId: recipient,
		})
	}

	_, err := s.addRecipients(ctx, dlRecipients)

	if err != nil {
		return dto.DistributionListSummary{}, err
	}

	summary, err := s.getDLSummary(ctx, listName)

	if err != nil {
		return dto.DistributionListSummary{}, err
	}

	return *summary, nil
}

func (s *DynamoDBStorage) DeleteRecipients(ctx context.Context, listName string, recipients []string) (dto.DistributionListSummary, error) {

	dlRecipients := make([]distributionList, 0, len(recipients))

	for _, r := range recipients {
		dlRecipients = append(dlRecipients, distributionList{
			Name:   listName,
			UserId: r,
		})
	}

	_, err := s.deleteRecipients(ctx, dlRecipients)

	if err != nil {
		return dto.DistributionListSummary{}, err
	}

	summary, err := s.getDLSummary(ctx, listName)

	if err != nil {
		return dto.DistributionListSummary{}, err
	}

	return *summary, nil
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
